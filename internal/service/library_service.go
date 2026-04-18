package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/repository"
)

// LibraryService handles library browsing and search logic.
type LibraryService struct {
	songRepo     *repository.SongRepository
	playbackRepo *repository.PlaybackEventRepository
	favoriteRepo *repository.FavoriteRepository
}

// NewLibraryService creates a new LibraryService.
func NewLibraryService(
	songRepo *repository.SongRepository,
	playbackRepo *repository.PlaybackEventRepository,
	favoriteRepo *repository.FavoriteRepository,
) *LibraryService {
	return &LibraryService{
		songRepo:     songRepo,
		playbackRepo: playbackRepo,
		favoriteRepo: favoriteRepo,
	}
}

// ListSongs returns a paginated list of songs.
func (s *LibraryService) ListSongs(ctx context.Context, userID, cursor string, limit int, sortBy, sortDir string) ([]*model.Song, string, error) {
	return s.songRepo.List(ctx, userID, cursor, limit, sortBy, sortDir)
}

// GetSong returns a single song by ID.
func (s *LibraryService) GetSong(ctx context.Context, userID, songID string) (*model.Song, error) {
	return s.songRepo.GetByID(ctx, userID, songID)
}

// SearchSongs performs full-text search.
func (s *LibraryService) SearchSongs(ctx context.Context, userID, query string) ([]*model.Song, error) {
	return s.songRepo.Search(ctx, userID, query, 100)
}

// GetArtists returns all artist groups.
func (s *LibraryService) GetArtists(ctx context.Context, userID string) ([]model.ArtistGroup, error) {
	return s.songRepo.GetArtists(ctx, userID)
}

// GetArtistSongs decodes the artist ID and returns songs by that artist.
func (s *LibraryService) GetArtistSongs(ctx context.Context, userID, artistID string) ([]*model.Song, error) {
	nameBytes, err := base64.URLEncoding.DecodeString(artistID)
	if err != nil {
		return nil, fmt.Errorf("invalid artist id: %w", err)
	}
	return s.songRepo.ListByArtist(ctx, userID, string(nameBytes))
}

// GetAlbums returns all album groups.
func (s *LibraryService) GetAlbums(ctx context.Context, userID string) ([]model.AlbumGroup, error) {
	return s.songRepo.GetAlbums(ctx, userID)
}

// GetAlbumSongs decodes the album ID and returns songs in that album.
func (s *LibraryService) GetAlbumSongs(ctx context.Context, userID, albumID string) ([]*model.Song, error) {
	decoded, err := base64.URLEncoding.DecodeString(albumID)
	if err != nil {
		return nil, fmt.Errorf("invalid album id: %w", err)
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid album id format")
	}
	return s.songRepo.ListByAlbum(ctx, userID, parts[0], parts[1])
}

// GetHomeSections returns curated home screen sections.
func (s *LibraryService) GetHomeSections(ctx context.Context, userID string) (*model.HomeResponse, error) {
	var sections []model.HomeSection

	// Recently Added
	recentSongs, err := s.songRepo.RecentlyAdded(ctx, userID, 20)
	if err == nil && len(recentSongs) > 0 {
		sections = append(sections, model.HomeSection{
			Title: "Recently Added",
			Type:  "songs",
			Items: recentSongs,
		})
	}

	// Recently Played
	recentlyPlayed, err := s.playbackRepo.GetRecentlyPlayed(ctx, userID, 20)
	if err == nil && len(recentlyPlayed) > 0 {
		sections = append(sections, model.HomeSection{
			Title: "Recently Played",
			Type:  "songs",
			Items: recentlyPlayed,
		})
	}

	// Favorites
	favoriteSongs, err := s.favoriteRepo.ListRecent(ctx, userID, 20)
	if err == nil && len(favoriteSongs) > 0 {
		sections = append(sections, model.HomeSection{
			Title: "Favorite Songs",
			Type:  "songs",
			Items: favoriteSongs,
		})
	}

	// Top Artists
	artists, err := s.songRepo.GetArtists(ctx, userID)
	if err == nil && len(artists) > 0 {
		limit := 10
		if len(artists) < limit {
			limit = len(artists)
		}
		sections = append(sections, model.HomeSection{
			Title: "Your Artists",
			Type:  "artists",
			Items: artists[:limit],
		})
	}

	// Library Stats
	songCount, err := s.songRepo.CountByUserID(ctx, userID)
	if err == nil {
		sections = append(sections, model.HomeSection{
			Title: "Drive Library",
			Type:  "stats",
			Items: map[string]int{"songCount": songCount},
		})
	}

	return &model.HomeResponse{Sections: sections}, nil
}
