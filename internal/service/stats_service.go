package service

import (
	"context"
	"fmt"

	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/repository"
)

// StatsService handles user-level aggregate statistics.
type StatsService struct {
	songRepo     *repository.SongRepository
	favoriteRepo *repository.FavoriteRepository
	playlistRepo *repository.PlaylistRepository
	playbackRepo *repository.PlaybackEventRepository
}

// NewStatsService creates a new StatsService.
func NewStatsService(
	songRepo *repository.SongRepository,
	favoriteRepo *repository.FavoriteRepository,
	playlistRepo *repository.PlaylistRepository,
	playbackRepo *repository.PlaybackEventRepository,
) *StatsService {
	return &StatsService{
		songRepo:     songRepo,
		favoriteRepo: favoriteRepo,
		playlistRepo: playlistRepo,
		playbackRepo: playbackRepo,
	}
}

// GetUserStats returns aggregate stats for a user.
func (s *StatsService) GetUserStats(ctx context.Context, userID string) (*model.UserStats, error) {
	songCount, err := s.songRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get song count: %w", err)
	}

	favoriteCount, err := s.favoriteRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get favorite count: %w", err)
	}

	playlistCount, err := s.playlistRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get playlist count: %w", err)
	}

	recentlyPlayedCount, err := s.playbackRepo.CountDistinctSongsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get recently played count: %w", err)
	}

	playbackEventCount, err := s.playbackRepo.CountEventsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get playback event count: %w", err)
	}

	return &model.UserStats{
		SongCount:           songCount,
		FavoriteSongCount:   favoriteCount,
		PlaylistCount:       playlistCount,
		RecentlyPlayedCount: recentlyPlayedCount,
		PlaybackEventCount:  playbackEventCount,
	}, nil
}
