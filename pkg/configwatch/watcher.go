// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package configwatch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// CallbackFunc is called when a config change is detected.
// The path parameter is the path to the changed config file.
// Return an error to indicate the reload failed, which will be logged.
type CallbackFunc func(ctx context.Context, path string) error

// Watcher monitors config files for changes and triggers reload callbacks.
//
// Design rationale:
// - File watching enables zero-downtime config updates
// - Debouncing prevents multiple rapid reloads (e.g., from editor saves)
// - Graceful shutdown ensures clean resource cleanup
// - Thread-safe operations allow concurrent access
type Watcher struct {
	configPath string
	callback    CallbackFunc
	debounce   time.Duration
	watcher    *fsnotify.Watcher
	quitChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
}

// NewWatcher creates a new config file watcher.
//
// Parameters:
//   - configPath: Path to the config file to watch
//   - callback: Function to call when config changes
//   - debounce: Minimum time between reloads (prevents rapid reloads)
//
// Recommended debounce is 100-500ms to handle editor save behavior.
func NewWatcher(configPath string, callback CallbackFunc, debounce time.Duration) (*Watcher, error) {
	if callback == nil {
		return nil, fmt.Errorf("callback cannot be nil")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &Watcher{
		configPath: configPath,
		callback:   callback,
		debounce:  debounce,
		watcher:   watcher,
		quitChan:  make(chan struct{}),
	}, nil
}

// Start begins watching the config file for changes.
func (w *Watcher) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.quitChan != nil {
		// Already running
		return fmt.Errorf("watcher already running")
	}

	// Watch the directory (not the file itself, for editor compatibility)
	configDir := filepath.Dir(w.configPath)
	if err := w.watcher.Add(configDir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	w.quitChan = make(chan struct{})
	w.wg.Add(1)

	go w.runLoop()

	logger.InfoCF("config", "Config watcher started", map[string]any{
		"path": w.configPath,
	})

	return nil
}

// Stop stops watching the config file.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.quitChan == nil {
		// Not running
		return
	}

	close(w.quitChan)
	w.wg.Wait()

	// Remove the watch
	w.watcher.Close()
	w.quitChan = nil

	logger.InfoC("config", "Config watcher stopped")
}

// runLoop runs the file watching loop.
func (w *Watcher) runLoop() {
	defer w.wg.Done()

	var timer *time.Timer
	var timerChan <-chan time.Time

	for {
		select {
		case <-w.quitChan:
			// Stop timer if running
			if timer != nil {
				timer.Stop()
			}
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only care about write and create events on our config file
			if event.Name != w.configPath {
				continue
			}

			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			logger.DebugCF("config", "Config file changed, debouncing...", map[string]any{
				"path": event.Name,
			})

			// Reset or create timer for debouncing
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(w.debounce)
			timerChan = timer.C

		case <-timerChan:
			// Debounce period elapsed, trigger reload
			logger.InfoCF("config", "Config file changed, reloading...", map[string]any{
				"path": w.configPath,
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := w.callback(ctx, w.configPath)
			cancel()

			if err != nil {
				logger.ErrorCF("config", "Failed to reload config", map[string]any{
					"error": err.Error(),
				})
			} else {
				logger.InfoC("config", "Config reloaded successfully")
			}

			// Nullify timer so we don't accidentally select on it again
			timer = nil
			timerChan = nil

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			logger.ErrorCF("config", "Watcher error", map[string]any{
				"error": err.Error(),
			})
		}
	}
}

// IsRunning returns true if the watcher is currently running.
func (w *Watcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.quitChan != nil
}
