// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package auth

import (
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// TokenRefresher manages automatic OAuth token refresh.
//
// Design rationale:
// - OAuth tokens typically expire after 1 hour
// - Refresh tokens allow getting new access tokens without re-authentication
// - Background refresh prevents service interruption
// - Refresh is done 5 minutes before expiry to ensure valid tokens
type TokenRefresher struct {
	checkInterval time.Duration
	quitChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
}

// NewTokenRefresher creates a new token refresher.
// The check interval determines how often to check for tokens needing refresh.
// Default is 1 minute, which provides a good balance between responsiveness
// and resource usage.
func NewTokenRefresher() *TokenRefresher {
	return &TokenRefresher{
		checkInterval: time.Minute,
		quitChan:      make(chan struct{}),
	}
}

// Start begins the automatic token refresh loop.
// This runs in a background goroutine and checks for tokens that need refresh.
func (tr *TokenRefresher) Start() {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if tr.quitChan != nil {
		// Already running
		return
	}

	tr.quitChan = make(chan struct{})
	tr.wg.Add(1)

	go tr.runLoop()

	logger.InfoC("auth", "Token refresher started")
}

// Stop gracefully stops the token refresher.
func (tr *TokenRefresher) Stop() {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if tr.quitChan == nil {
		// Not running
		return
	}

	close(tr.quitChan)
	tr.quitChan = nil
	tr.wg.Wait()

	logger.InfoC("auth", "Token refresher stopped")
}

// runLoop runs the token refresh check loop.
func (tr *TokenRefresher) runLoop() {
	defer tr.wg.Done()

	ticker := time.NewTicker(tr.checkInterval)
	defer ticker.Stop()

	// Do initial check on startup
	tr.checkAndRefreshTokens()

	for {
		select {
		case <-tr.quitChan:
			return
		case <-ticker.C:
			tr.checkAndRefreshTokens()
		}
	}
}

// checkAndRefreshTokens checks all stored credentials and refreshes those that need it.
func (tr *TokenRefresher) checkAndRefreshTokens() {
	store, err := LoadStore()
	if err != nil {
		logger.ErrorCF("auth", "Failed to load auth store", map[string]any{"error": err.Error()})
		return
	}

	if len(store.Credentials) == 0 {
		return
	}

	refreshed := 0

	for provider, cred := range store.Credentials {
		if !cred.NeedsRefresh() {
			continue
		}

		// Check if provider uses OAuth
		if cred.AuthMethod != "oauth" {
			continue
		}

		// Check if refresh token is available
		if cred.RefreshToken == "" {
			logger.WarnCF("auth", "Token needs refresh but no refresh token available", map[string]any{
				"provider": provider,
			})
			continue
		}

		// Get OAuth config for this provider
		var cfg OAuthProviderConfig
		switch provider {
		case "openai":
			cfg = OpenAIOAuthConfig()
		case "google-antigravity", "antigravity":
			cfg = GoogleAntigravityOAuthConfig()
		default:
			logger.DebugCF("auth", "No OAuth config for provider", map[string]any{"provider": provider})
			continue
		}

		// Refresh the token
		refreshedCred, err := RefreshAccessToken(cred, cfg)
		if err != nil {
			logger.ErrorCF("auth", "Failed to refresh token", map[string]any{
				"provider": provider,
				"error":    err.Error(),
			})
			continue
		}

		// Update the stored credential
		store.Credentials[provider] = refreshedCred
		if err := SaveStore(store); err != nil {
			logger.ErrorCF("auth", "Failed to save refreshed token", map[string]any{
				"provider": provider,
				"error":    err.Error(),
			})
			continue
		}

		refreshed++
		logger.InfoCF("auth", "Token refreshed successfully", map[string]any{
			"provider":  provider,
			"expiresAt": refreshedCred.ExpiresAt.Format(time.RFC3339),
		})
	}

	if refreshed > 0 {
		logger.InfoCF("auth", "Token refresh cycle complete", map[string]any{
			"refreshed": refreshed,
		})
	}
}

// IsRunning returns true if the token refresher is currently running.
func (tr *TokenRefresher) IsRunning() bool {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return tr.quitChan != nil
}
