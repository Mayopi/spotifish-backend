package config

import (
	"os"
	"testing"
)

func TestLoad_RequiredFields(t *testing.T) {
	// Backup existing env vars
	origDB := os.Getenv("DATABASE_URL")
	origJWT := os.Getenv("JWT_SIGNING_KEY")
	origGC := os.Getenv("GOOGLE_CLIENT_ID")
	defer func() {
		os.Setenv("DATABASE_URL", origDB)
		os.Setenv("JWT_SIGNING_KEY", origJWT)
		os.Setenv("GOOGLE_CLIENT_ID", origGC)
	}()

	// Missing DATABASE_URL
	os.Setenv("DATABASE_URL", "")
	os.Setenv("JWT_SIGNING_KEY", "test")
	os.Setenv("GOOGLE_CLIENT_ID", "test")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}

	// Missing JWT_SIGNING_KEY
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("JWT_SIGNING_KEY", "")
	_, err = Load()
	if err == nil {
		t.Fatal("expected error for missing JWT_SIGNING_KEY")
	}

	// Missing GOOGLE_CLIENT_ID
	os.Setenv("JWT_SIGNING_KEY", "test")
	os.Setenv("GOOGLE_CLIENT_ID", "")
	_, err = Load()
	if err == nil {
		t.Fatal("expected error for missing GOOGLE_CLIENT_ID")
	}
}

func TestLoad_Defaults(t *testing.T) {
	origDB := os.Getenv("DATABASE_URL")
	origJWT := os.Getenv("JWT_SIGNING_KEY")
	origGC := os.Getenv("GOOGLE_CLIENT_ID")
	defer func() {
		os.Setenv("DATABASE_URL", origDB)
		os.Setenv("JWT_SIGNING_KEY", origJWT)
		os.Setenv("GOOGLE_CLIENT_ID", origGC)
	}()

	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("JWT_SIGNING_KEY", "test")
	os.Setenv("GOOGLE_CLIENT_ID", "test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("default Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("default LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.SyncIntervalHours != 6 {
		t.Errorf("default SyncIntervalHours = %d, want %d", cfg.SyncIntervalHours, 6)
	}
	if cfg.RateLimitPerMin != 200 {
		t.Errorf("default RateLimitPerMin = %d, want %d", cfg.RateLimitPerMin, 200)
	}
	if cfg.AlbumArtPath != "./art" {
		t.Errorf("default AlbumArtPath = %q, want %q", cfg.AlbumArtPath, "./art")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	origDB := os.Getenv("DATABASE_URL")
	origJWT := os.Getenv("JWT_SIGNING_KEY")
	origGC := os.Getenv("GOOGLE_CLIENT_ID")
	origPort := os.Getenv("PORT")
	origSync := os.Getenv("SYNC_INTERVAL_HOURS")
	origRate := os.Getenv("RATE_LIMIT_PER_MIN")
	defer func() {
		os.Setenv("DATABASE_URL", origDB)
		os.Setenv("JWT_SIGNING_KEY", origJWT)
		os.Setenv("GOOGLE_CLIENT_ID", origGC)
		os.Setenv("PORT", origPort)
		os.Setenv("SYNC_INTERVAL_HOURS", origSync)
		os.Setenv("RATE_LIMIT_PER_MIN", origRate)
	}()

	os.Setenv("DATABASE_URL", "postgres://prod/db")
	os.Setenv("JWT_SIGNING_KEY", "secret")
	os.Setenv("GOOGLE_CLIENT_ID", "client-id")
	os.Setenv("PORT", "9090")
	os.Setenv("SYNC_INTERVAL_HOURS", "12")
	os.Setenv("RATE_LIMIT_PER_MIN", "500")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.SyncIntervalHours != 12 {
		t.Errorf("SyncIntervalHours = %d, want %d", cfg.SyncIntervalHours, 12)
	}
	if cfg.RateLimitPerMin != 500 {
		t.Errorf("RateLimitPerMin = %d, want %d", cfg.RateLimitPerMin, 500)
	}
}
