package main

import (
	"context"
	"time"
)

// SyncConfig holds timeout and retry settings
type SyncConfig struct {
	MaxTimeout      time.Duration
	RateLimitWait   time.Duration
	MaxRetries      int
	RetryBackoff    time.Duration
}

// DefaultSyncConfig returns safe defaults for sync operations
func DefaultSyncConfig() SyncConfig {
	return SyncConfig{
		MaxTimeout:    5 * time.Minute,
		RateLimitWait: 30 * time.Second,
		MaxRetries:    3,
		RetryBackoff:  1 * time.Second,
	}
}

// RetryWithBackoff executes fn with exponential backoff on error
func RetryWithBackoff(ctx context.Context, fn func() error, config SyncConfig) error {
	var lastErr error
	backoff := config.RetryBackoff

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(backoff):
				// Continue after backoff
			case <-ctx.Done():
				return ctx.Err()
			}
			backoff *= 2 // Exponential backoff
		}

		if err := fn(); err != nil {
			lastErr = err
			// Check for rate limit error
			if isRateLimitError(err) {
				// Wait longer for rate limit
				select {
				case <-time.After(config.RateLimitWait):
				case <-ctx.Done():
					return ctx.Err()
				}
				continue
			}
			continue
		}
		return nil
	}

	return lastErr
}

// isRateLimitError checks if error is WhatsApp rate limiting (429)
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "429") || contains(errStr, "rate") || contains(errStr, "throttle")
}

// contains checks if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
