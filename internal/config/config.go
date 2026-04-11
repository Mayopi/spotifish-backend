package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the application.
type Config struct {
	Port     string
	LogLevel string

	// Database
	DatabaseURL string

	// Authentication
	JWTSigningKey    string
	GoogleClientID   string
	GoogleClientSecret string

	// Drive OAuth
	GoogleRedirectURI   string
	DriveEncryptionKey  string

	// Album Art Storage
	AlbumArtPath string

	// Sync
	SyncIntervalHours int

	// Rate Limiting
	RateLimitPerMin int
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		JWTSigningKey:      getEnv("JWT_SIGNING_KEY", ""),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURI:  getEnv("GOOGLE_REDIRECT_URI", ""),
		DriveEncryptionKey: getEnv("DRIVE_ENCRYPTION_KEY", ""),
		AlbumArtPath:       getEnv("ALBUM_ART_PATH", "./art"),
		SyncIntervalHours:  getEnvInt("SYNC_INTERVAL_HOURS", 6),
		RateLimitPerMin:    getEnvInt("RATE_LIMIT_PER_MIN", 200),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSigningKey == "" {
		return nil, fmt.Errorf("JWT_SIGNING_KEY is required")
	}
	if cfg.GoogleClientID == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}
