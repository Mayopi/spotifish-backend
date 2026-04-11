package service

import (
	"context"
	"fmt"

	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/repository"
)

// FavoriteService handles favorites business logic.
type FavoriteService struct {
	favoriteRepo *repository.FavoriteRepository
	songRepo     *repository.SongRepository
}

// NewFavoriteService creates a new FavoriteService.
func NewFavoriteService(favoriteRepo *repository.FavoriteRepository, songRepo *repository.SongRepository) *FavoriteService {
	return &FavoriteService{favoriteRepo: favoriteRepo, songRepo: songRepo}
}

// List returns all favorite songs for a user.
func (s *FavoriteService) List(ctx context.Context, userID string) ([]*model.Song, error) {
	return s.favoriteRepo.List(ctx, userID)
}

// Add adds a song to favorites.
func (s *FavoriteService) Add(ctx context.Context, userID, songID string) error {
	// Verify song exists
	song, err := s.songRepo.GetByID(ctx, userID, songID)
	if err != nil {
		return err
	}
	if song == nil {
		return fmt.Errorf("song not found")
	}
	return s.favoriteRepo.Add(ctx, userID, songID)
}

// Remove removes a song from favorites.
func (s *FavoriteService) Remove(ctx context.Context, userID, songID string) error {
	return s.favoriteRepo.Remove(ctx, userID, songID)
}
