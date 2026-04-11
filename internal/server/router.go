package server

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spotifish/backend/internal/config"
	"github.com/spotifish/backend/internal/handler"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/repository"
	"github.com/spotifish/backend/internal/service"
	"github.com/spotifish/backend/internal/worker"
)

// Dependencies holds all initialized service dependencies for the server.
type Dependencies struct {
	// Repositories
	UserRepo     *repository.UserRepository
	AuthRepo     *repository.AuthRepository
	SettingsRepo *repository.SettingsRepository
	DriveRepo    *repository.DriveRepository
	SongRepo     *repository.SongRepository
	SyncRepo     *repository.SyncRepository
	PlaylistRepo *repository.PlaylistRepository
	FavoriteRepo *repository.FavoriteRepository
	PlaybackRepo *repository.PlaybackEventRepository

	// Services
	AuthSvc     *service.AuthService
	SettingsSvc *service.SettingsService
	DriveSvc    *service.DriveService
	SyncSvc     *service.SyncService
	LibrarySvc  *service.LibraryService
	PlaylistSvc *service.PlaylistService
	FavoriteSvc *service.FavoriteService
	MetaSvc     *service.MetadataService
	ArtSvc      *service.ArtStorageService

	// Worker
	SyncWorker *worker.SyncWorker
}

// InitDependencies initializes all repositories, services, and workers.
func InitDependencies(pool *pgxpool.Pool, cfg *config.Config) *Dependencies {
	// Repositories
	userRepo := repository.NewUserRepository(pool)
	authRepo := repository.NewAuthRepository(pool)
	settingsRepo := repository.NewSettingsRepository(pool)
	driveRepo := repository.NewDriveRepository(pool)
	songRepo := repository.NewSongRepository(pool)
	syncRepo := repository.NewSyncRepository(pool)
	playlistRepo := repository.NewPlaylistRepository(pool)
	favoriteRepo := repository.NewFavoriteRepository(pool)
	playbackRepo := repository.NewPlaybackEventRepository(pool)

	// Services
	authSvc := service.NewAuthService(userRepo, authRepo, cfg.JWTSigningKey, cfg.GoogleClientID)
	settingsSvc := service.NewSettingsService(settingsRepo)
	driveSvc := service.NewDriveService(driveRepo, songRepo, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURI, cfg.DriveEncryptionKey)
	metaSvc := service.NewMetadataService()
	artSvc := service.NewArtStorageService(cfg.AlbumArtPath)
	syncSvc := service.NewSyncService(syncRepo, songRepo, driveRepo, driveSvc, metaSvc, artSvc)
	librarySvc := service.NewLibraryService(songRepo, playbackRepo)
	playlistSvc := service.NewPlaylistService(playlistRepo, songRepo)
	favoriteSvc := service.NewFavoriteService(favoriteRepo, songRepo)

	// Worker
	syncWorker := worker.NewSyncWorker(syncSvc, driveRepo)

	return &Dependencies{
		UserRepo: userRepo, AuthRepo: authRepo, SettingsRepo: settingsRepo,
		DriveRepo: driveRepo, SongRepo: songRepo, SyncRepo: syncRepo,
		PlaylistRepo: playlistRepo, FavoriteRepo: favoriteRepo, PlaybackRepo: playbackRepo,
		AuthSvc: authSvc, SettingsSvc: settingsSvc, DriveSvc: driveSvc,
		SyncSvc: syncSvc, LibrarySvc: librarySvc, PlaylistSvc: playlistSvc,
		FavoriteSvc: favoriteSvc, MetaSvc: metaSvc, ArtSvc: artSvc,
		SyncWorker: syncWorker,
	}
}

// SetupRouter creates and configures the Gin router with all routes.
func SetupRouter(pool *pgxpool.Pool, deps *Dependencies, cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// Global middleware
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimiter(cfg.RateLimitPerMin))

	// API version group
	v1 := r.Group("/v1")

	// Public routes (no auth required)
	authHandler := handler.NewAuthHandler(deps.AuthSvc)
	authHandler.RegisterRoutes(v1)

	// Health and art (no auth required)
	healthHandler := handler.NewHealthHandler(pool, cfg.AlbumArtPath)
	healthHandler.RegisterRoutes(r, v1)

	// Authenticated routes
	authed := v1.Group("")
	authed.Use(middleware.Auth(deps.AuthSvc))
	{
		// User profile & settings
		userHandler := handler.NewUserHandler(deps.UserRepo, deps.SettingsSvc)
		userHandler.RegisterRoutes(authed)

		// Drive
		driveHandler := handler.NewDriveHandler(deps.DriveSvc)
		driveHandler.RegisterRoutes(authed)

		// Sync
		syncHandler := handler.NewSyncHandler(deps.SyncSvc)
		syncHandler.RegisterRoutes(authed)

		// Songs & Library
		songHandler := handler.NewSongHandler(deps.LibrarySvc)
		songHandler.RegisterRoutes(authed)

		// Streaming
		streamHandler := handler.NewStreamHandler(deps.LibrarySvc, deps.DriveSvc)
		streamHandler.RegisterRoutes(authed)

		// Artists
		artistHandler := handler.NewArtistHandler(deps.LibrarySvc)
		artistHandler.RegisterRoutes(authed)

		// Albums
		albumHandler := handler.NewAlbumHandler(deps.LibrarySvc)
		albumHandler.RegisterRoutes(authed)

		// Home
		homeHandler := handler.NewHomeHandler(deps.LibrarySvc)
		homeHandler.RegisterRoutes(authed)

		// Playlists
		playlistHandler := handler.NewPlaylistHandler(deps.PlaylistSvc)
		playlistHandler.RegisterRoutes(authed)

		// Favorites
		favoriteHandler := handler.NewFavoriteHandler(deps.FavoriteSvc)
		favoriteHandler.RegisterRoutes(authed)

		// Playback events
		playbackHandler := handler.NewPlaybackHandler(deps.PlaybackRepo)
		playbackHandler.RegisterRoutes(authed)
	}

	return r
}
