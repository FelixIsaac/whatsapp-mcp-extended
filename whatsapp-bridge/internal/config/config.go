package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	APIPort     int
	APIBindHost string // API_BIND_HOST: default 127.0.0.1 (safe for local dev); set 0.0.0.0 in Docker

	// History sync configuration (Phase 4)
	HistorySyncDaysLimit uint32 // HISTORY_SYNC_DAYS_LIMIT env var
	HistorySyncSizeMB    uint32 // HISTORY_SYNC_SIZE_MB env var
	StorageQuotaMB       uint32 // STORAGE_QUOTA_MB env var

	// Presence ping configuration
	// PRESENCE_PING_ENABLED=false disables presence broadcasts to contacts (default true)
	// PRESENCE_PING_INTERVAL sets how often to ping (default 20m; reduce below 25m risks bot fingerprinting)
	PresencePingEnabled  bool
	PresencePingInterval time.Duration

	// Human-like presence behaviour
	// WA_PRESENCE_MODE=human (default) keeps the account offline and only goes
	// online for a short window around outgoing activity. always_online restores the
	// legacy "online while connected" behaviour.
	PresenceMode      string
	PresenceLingerMin time.Duration // PRESENCE_LINGER_MIN
	PresenceLingerMax time.Duration // PRESENCE_LINGER_MAX
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	cfg := &Config{
		APIPort:     8080,
		APIBindHost: "127.0.0.1",
		// History sync defaults
		HistorySyncDaysLimit: 365,   // 1 year default
		HistorySyncSizeMB:    5000,  // 5GB default
		StorageQuotaMB:       10240, // 10GB default
		// Presence ping defaults
		PresencePingEnabled:  true,
		PresencePingInterval: 20 * time.Minute,
		// Human-like presence defaults
		PresenceMode:      "human",
		PresenceLingerMin: 8 * time.Second,
		PresenceLingerMax: 15 * time.Second,
	}

	// Override with environment variables if set
	if port := os.Getenv("API_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.APIPort = p
		}
	}

	if bindHost := os.Getenv("API_BIND_HOST"); bindHost != "" {
		cfg.APIBindHost = bindHost
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

	if enabled := os.Getenv("PRESENCE_PING_ENABLED"); enabled == "false" {
		cfg.PresencePingEnabled = false
	}

	if interval := os.Getenv("PRESENCE_PING_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil && d > 0 {
			cfg.PresencePingInterval = d
		}
	}

	// WA_PRESENCE_MODE is the documented name; the JUNO_ prefixed variant stays readable so an
	// existing deployment keeps working after the rename.
	mode := os.Getenv("WA_PRESENCE_MODE")
	if mode == "" {
		mode = os.Getenv("JUNO_WA_PRESENCE_MODE")
	}
	if mode == "always_online" {
		cfg.PresenceMode = "always_online"
	}

	if min := os.Getenv("PRESENCE_LINGER_MIN"); min != "" {
		if d, err := time.ParseDuration(min); err == nil && d > 0 {
			cfg.PresenceLingerMin = d
		}
	}

	if max := os.Getenv("PRESENCE_LINGER_MAX"); max != "" {
		if d, err := time.ParseDuration(max); err == nil && d > 0 {
			cfg.PresenceLingerMax = d
		}
	}

	if cfg.PresenceLingerMax < cfg.PresenceLingerMin {
		cfg.PresenceLingerMax = cfg.PresenceLingerMin
	}

	return cfg
}
