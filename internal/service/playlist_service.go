package service

import (
	"context"
	"fmt"

	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/repository"
)

// PlaylistService handles playlist business logic.
type PlaylistService struct {
	playlistRepo *repository.PlaylistRepository
	songRepo     *repository.SongRepository
}

// NewPlaylistService creates a new PlaylistService.
func NewPlaylistService(playlistRepo *repository.PlaylistRepository, songRepo *repository.SongRepository) *PlaylistService {
	return &PlaylistService{playlistRepo: playlistRepo, songRepo: songRepo}
}

// List returns all playlists for a user.
func (s *PlaylistService) List(ctx context.Context, userID string) ([]*model.Playlist, error) {
	return s.playlistRepo.List(ctx, userID)
}

// Create creates a new playlist.
func (s *PlaylistService) Create(ctx context.Context, userID, name string) (*model.Playlist, error) {
	return s.playlistRepo.Create(ctx, &model.Playlist{UserID: userID, Name: name})
}

// Rename renames a playlist, verifying ownership.
func (s *PlaylistService) Rename(ctx context.Context, userID, playlistID, name string) error {
	if _, err := s.verifyOwnership(ctx, userID, playlistID); err != nil {
		return err
	}
	return s.playlistRepo.Rename(ctx, userID, playlistID, name)
}

// Delete deletes a playlist, verifying ownership.
func (s *PlaylistService) Delete(ctx context.Context, userID, playlistID string) error {
	if _, err := s.verifyOwnership(ctx, userID, playlistID); err != nil {
		return err
	}
	return s.playlistRepo.Delete(ctx, userID, playlistID)
}

// AddSong adds a song to a playlist.
func (s *PlaylistService) AddSong(ctx context.Context, userID, playlistID, songID string) error {
	if _, err := s.verifyOwnership(ctx, userID, playlistID); err != nil {
		return err
	}
	// Verify song exists
	song, err := s.songRepo.GetByID(ctx, userID, songID)
	if err != nil {
		return err
	}
	if song == nil {
		return fmt.Errorf("song not found")
	}
	return s.playlistRepo.AddSong(ctx, playlistID, songID)
}

// RemoveSong removes a song from a playlist.
func (s *PlaylistService) RemoveSong(ctx context.Context, userID, playlistID, songID string) error {
	if _, err := s.verifyOwnership(ctx, userID, playlistID); err != nil {
		return err
	}
	return s.playlistRepo.RemoveSong(ctx, playlistID, songID)
}

// ReplaceSongs replaces the full ordered song list.
func (s *PlaylistService) ReplaceSongs(ctx context.Context, userID, playlistID string, songIDs []string) error {
	if _, err := s.verifyOwnership(ctx, userID, playlistID); err != nil {
		return err
	}
	return s.playlistRepo.ReplaceSongs(ctx, playlistID, songIDs)
}

// GetSongs returns the songs in a playlist.
func (s *PlaylistService) GetSongs(ctx context.Context, userID, playlistID string) ([]*model.Song, error) {
	if _, err := s.verifyOwnership(ctx, userID, playlistID); err != nil {
		return nil, err
	}
	return s.playlistRepo.GetSongs(ctx, playlistID)
}

// verifyOwnership checks that the playlist belongs to the user.
func (s *PlaylistService) verifyOwnership(ctx context.Context, userID, playlistID string) (*model.Playlist, error) {
	playlist, err := s.playlistRepo.GetByID(ctx, userID, playlistID)
	if err != nil {
		return nil, err
	}
	if playlist == nil {
		return nil, fmt.Errorf("playlist not found")
	}
	return playlist, nil
}
