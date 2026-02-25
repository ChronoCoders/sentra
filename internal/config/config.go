package config

import (
	"os"
	"strings"
)

// Config holds application configuration.
type Config struct {
	DBPath      string
	JWTSecret   string
	WGInterface string
	Port        string
}

// Load loads configuration from environment variables.
func Load() *Config {
	return &Config{
		DBPath:      getEnv("SENTRA_DB", "sentra.db"),
		JWTSecret:   getEnv("SENTRA_JWT_SECRET", "dev-secret"),
		WGInterface: getEnv("SENTRA_WG_INTERFACE", "wg0"),
		Port:        getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return strings.TrimSpace(value)
	}
	return fallback
}
