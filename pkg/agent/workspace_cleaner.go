package agent

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// DefaultIgnorePatterns are used when no .picoclawignore file exists
var DefaultIgnorePatterns = []string{
	"logs/",
	"health/",
	"sessions/",
	"state/",
	"projects/",
	"archive/",
}

// MaxFileSizeForContext is the maximum file size (in bytes) that will be included in context
const MaxFileSizeForContext = 50 * 1024 // 50KB

// WorkspaceCleaner handles workspace cleanup and optimization
type WorkspaceCleaner struct {
	workspace string
}

// NewWorkspaceCleaner creates a new workspace cleaner
func NewWorkspaceCleaner(workspace string) *WorkspaceCleaner {
	return &WorkspaceCleaner{workspace: workspace}
}

// CleanupLargeFiles truncates files that are too large to be useful in context
// This prevents token bloat from massive log files
func (wc *WorkspaceCleaner) CleanupLargeFiles() error {
	maxSize := 5 * 1024 * 1024 // 5MB max per file

	patterns := []string{
		"logs/",
		"health/",
	}

	for _, pattern := range patterns {
		if err := wc.truncateMatchingFiles(pattern, maxSize); err != nil {
			logger.WarnCF("workspace", "Error during cleanup", map[string]any{
				"pattern": pattern,
				"error":   err.Error(),
			})
		}
	}

	return nil
}

// truncateMatchingFiles truncates files matching the given pattern
func (wc *WorkspaceCleaner) truncateMatchingFiles(pattern string, maxSize int) error {
	// Handle directory patterns like "logs/"
	if strings.HasSuffix(pattern, "/") {
		dir := filepath.Join(wc.workspace, strings.TrimSuffix(pattern, "/"))
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if !info.Mode().IsRegular() {
				continue
			}
			wc.truncateFile(filepath.Join(dir, info.Name()), maxSize)
		}
		return nil
	}

	return nil
}

// truncateFile truncates a file to maxSize bytes (keeps the end)
func (wc *WorkspaceCleaner) truncateFile(path string, maxSize int) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	if int(info.Size()) <= maxSize {
		return
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// Keep only the last maxSize bytes
	truncated := data[len(data)-maxSize:]

	// Write back
	if err := os.WriteFile(path, truncated, 0644); err != nil {
		logger.WarnCF("workspace", "Failed to truncate file", map[string]any{
			"path":  path,
			"error": err.Error(),
		})
		return
	}

	logger.InfoCF("workspace", "Truncated large file", map[string]any{
		"path":     path,
		"original": info.Size(),
		"new_size": maxSize,
	})
}
