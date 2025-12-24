package config

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	APIPort int

	// History sync configuration (Phase 4)
	HistorySyncDaysLimit uint32 // HISTORY_SYNC_DAYS_LIMIT env var
	HistorySyncSizeMB    uint32 // HISTORY_SYNC_SIZE_MB env var
	StorageQuotaMB       uint32 // STORAGE_QUOTA_MB env var
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	cfg := &Config{
		APIPort: 8080,
		// History sync defaults
		HistorySyncDaysLimit: 365,   // 1 year default
		HistorySyncSizeMB:    5000,  // 5GB default
		StorageQuotaMB:       10240, // 10GB default
	}

	// Override with environment variables if set
	if port := os.Getenv("API_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.APIPort = p
		}
	}

	if days := os.Getenv("HISTORY_SYNC_DAYS_LIMIT"); days != "" {
		if d, err := strconv.ParseUint(days, 10, 32); err == nil {
			cfg.HistorySyncDaysLimit = uint32(d)
		}
	}

	if size := os.Getenv("HISTORY_SYNC_SIZE_MB"); size != "" {
		if s, err := strconv.ParseUint(size, 10, 32); err == nil {
			cfg.HistorySyncSizeMB = uint32(s)
		}
	}

	if quota := os.Getenv("STORAGE_QUOTA_MB"); quota != "" {
		if q, err := strconv.ParseUint(quota, 10, 32); err == nil {
			cfg.StorageQuotaMB = uint32(q)
		}
	}

	return cfg
}
