package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spotifish/backend/internal/model"
)

// FavoriteRepository handles database operations for favorites.
type FavoriteRepository struct {
	pool *pgxpool.Pool
}

// NewFavoriteRepository creates a new FavoriteRepository.
func NewFavoriteRepository(pool *pgxpool.Pool) *FavoriteRepository {
	return &FavoriteRepository{pool: pool}
}

// Add adds a song to the user's favorites.
func (r *FavoriteRepository) Add(ctx context.Context, userID, songID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO favorites (user_id, song_id)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id, song_id) DO NOTHING`,
		userID, songID,
	)
	if err != nil {
		return fmt.Errorf("add favorite: %w", err)
	}
	return nil
}

// Remove removes a song from the user's favorites.
func (r *FavoriteRepository) Remove(ctx context.Context, userID, songID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM favorites WHERE user_id = $1 AND song_id = $2`,
		userID, songID,
	)
	if err != nil {
		return fmt.Errorf("remove favorite: %w", err)
	}
	return nil
}

// List returns all favorited songs for a user (full Song objects).
func (r *FavoriteRepository) List(ctx context.Context, userID string) ([]*model.Song, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.id, s.user_id, s.source, s.source_file_id, s.title, s.artist, s.album,
		        s.duration_ms, s.mime_type, s.album_art_object_key, s.drive_modified_at, s.added_at
		 FROM songs s
		 JOIN favorites f ON f.song_id = s.id
		 WHERE f.user_id = $1
		 ORDER BY f.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list favorites: %w", err)
	}
	defer rows.Close()
	return scanSongs(rows)
}

// ListRecent returns the most recently favorited songs for a user.
func (r *FavoriteRepository) ListRecent(ctx context.Context, userID string, limit int) ([]*model.Song, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.id, s.user_id, s.source, s.source_file_id, s.title, s.artist, s.album,
		        s.duration_ms, s.mime_type, s.album_art_object_key, s.drive_modified_at, s.added_at
		 FROM songs s
		 JOIN favorites f ON f.song_id = s.id
		 WHERE f.user_id = $1
		 ORDER BY f.created_at DESC
		 LIMIT $2`, userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list recent favorites: %w", err)
	}
	defer rows.Close()
	return scanSongs(rows)
}

// IsFavorite checks if a song is in the user's favorites.
func (r *FavoriteRepository) IsFavorite(ctx context.Context, userID, songID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND song_id = $2)`,
		userID, songID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check favorite: %w", err)
	}
	return exists, nil
}

// CountByUserID returns the total favorites count for a user.
func (r *FavoriteRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM favorites WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count favorites: %w", err)
	}
	return count, nil
}
