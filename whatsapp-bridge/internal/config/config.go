package config

// Config holds application configuration
type Config struct {
	APIPort int
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		APIPort: 8080,
	}
}
