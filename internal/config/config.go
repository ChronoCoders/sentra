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
	ControlURL  string
	AuthToken   string
	ServerID    string
	TLSCert     string
	TLSKey      string
	TLSAuto     bool
	Insecure    bool
}

// Load loads configuration from environment variables.
func Load() *Config {
	return &Config{
		DBPath:      getEnv("SENTRA_DB", "sentra.db"),
		JWTSecret:   getEnv("SENTRA_JWT_SECRET", "dev-secret"),
		WGInterface: getEnv("SENTRA_WG_INTERFACE", "wg0"),
		Port:        getEnv("PORT", "8080"),
		ControlURL:  getEnv("SENTRA_CONTROL_URL", "http://localhost:8080"),
		AuthToken:   getEnv("SENTRA_AUTH_TOKEN", ""),
		ServerID:    getEnv("SENTRA_SERVER_ID", "local"),
		TLSCert:     getEnv("SENTRA_TLS_CERT", ""),
		TLSKey:      getEnv("SENTRA_TLS_KEY", ""),
		TLSAuto:     getEnv("SENTRA_TLS_AUTO", "false") == "true",
		Insecure:    getEnv("SENTRA_INSECURE_SKIP_VERIFY", "false") == "true",
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return strings.TrimSpace(value)
	}
	return fallback
}
