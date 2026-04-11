package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spotifish/backend/internal/config"
	"github.com/spotifish/backend/internal/database"
	"github.com/spotifish/backend/internal/server"
)

func main() {
	// Load .env file if it exists (development)
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Configure logging
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

	log.Info().Str("port", cfg.Port).Msg("starting spotifish backend")

	// Connect to database
	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	// Run migrations
	if err := database.RunMigrations(cfg.DatabaseURL, "migrations"); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	// Initialize dependencies
	deps := server.InitDependencies(pool, cfg)

	// Ensure album art directory exists
	if err := deps.ArtSvc.EnsureDir(); err != nil {
		log.Fatal().Err(err).Msg("failed to create album art directory")
	}

	// Start sync worker
	if err := deps.SyncWorker.Start(cfg.SyncIntervalHours); err != nil {
		log.Fatal().Err(err).Msg("failed to start sync worker")
	}
	defer deps.SyncWorker.Stop()

	// Setup router
	router := server.SetupRouter(pool, deps, cfg)

	// Start HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second, // 5 min for streaming
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	log.Info().Str("addr", srv.Addr).Msg("server listening")

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server exited cleanly")
}
