package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spotifish/backend/internal/model"
)

// PlaylistRepository handles database operations for playlists.
type PlaylistRepository struct {
	pool *pgxpool.Pool
}

// NewPlaylistRepository creates a new PlaylistRepository.
func NewPlaylistRepository(pool *pgxpool.Pool) *PlaylistRepository {
	return &PlaylistRepository{pool: pool}
}

// List returns all playlists for a user, including song counts.
func (r *PlaylistRepository) List(ctx context.Context, userID string) ([]*model.Playlist, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.user_id, p.name, p.created_at, p.updated_at,
		        (SELECT COUNT(*) FROM playlist_songs ps WHERE ps.playlist_id = p.id) as song_count
		 FROM playlists p WHERE p.user_id = $1
		 ORDER BY p.updated_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list playlists: %w", err)
	}
	defer rows.Close()

	var playlists []*model.Playlist
	for rows.Next() {
		var p model.Playlist
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.CreatedAt, &p.UpdatedAt, &p.SongCount); err != nil {
			return nil, fmt.Errorf("scan playlist: %w", err)
		}
		playlists = append(playlists, &p)
	}
	return playlists, rows.Err()
}

// Create inserts a new playlist.
func (r *PlaylistRepository) Create(ctx context.Context, playlist *model.Playlist) (*model.Playlist, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO playlists (user_id, name)
		 VALUES ($1, $2)
		 RETURNING id, created_at, updated_at`,
		playlist.UserID, playlist.Name,
	).Scan(&playlist.ID, &playlist.CreatedAt, &playlist.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("create playlist: %w", err)
	}
	return playlist, nil
}

// GetByID retrieves a playlist by ID and user.
func (r *PlaylistRepository) GetByID(ctx context.Context, userID, playlistID string) (*model.Playlist, error) {
	var p model.Playlist
	err := r.pool.QueryRow(ctx,
		`SELECT p.id, p.user_id, p.name, p.created_at, p.updated_at,
		        (SELECT COUNT(*) FROM playlist_songs ps WHERE ps.playlist_id = p.id) as song_count
		 FROM playlists p WHERE p.id = $1 AND p.user_id = $2`,
		playlistID, userID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.CreatedAt, &p.UpdatedAt, &p.SongCount)

	if err != nil {
		return nil, fmt.Errorf("get playlist: %w", err)
	}
	return &p, nil
}

// Rename updates a playlist's name.
func (r *PlaylistRepository) Rename(ctx context.Context, userID, playlistID, name string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE playlists SET name = $3, updated_at = now()
		 WHERE id = $1 AND user_id = $2`,
		playlistID, userID, name,
	)
	if err != nil {
		return fmt.Errorf("rename playlist: %w", err)
	}
	return nil
}

// Delete removes a playlist.
func (r *PlaylistRepository) Delete(ctx context.Context, userID, playlistID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM playlists WHERE id = $1 AND user_id = $2`,
		playlistID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete playlist: %w", err)
	}
	return nil
}

// AddSong adds a song to a playlist at the end.
func (r *PlaylistRepository) AddSong(ctx context.Context, playlistID, songID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO playlist_songs (playlist_id, song_id, position)
		 VALUES ($1, $2, COALESCE((SELECT MAX(position) + 1 FROM playlist_songs WHERE playlist_id = $1), 0))
		 ON CONFLICT (playlist_id, song_id) DO NOTHING`,
		playlistID, songID,
	)
	if err != nil {
		return fmt.Errorf("add song to playlist: %w", err)
	}
	return nil
}

// RemoveSong removes a song from a playlist.
func (r *PlaylistRepository) RemoveSong(ctx context.Context, playlistID, songID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM playlist_songs WHERE playlist_id = $1 AND song_id = $2`,
		playlistID, songID,
	)
	if err != nil {
		return fmt.Errorf("remove song from playlist: %w", err)
	}
	return nil
}

// ReplaceSongs replaces the entire ordered song list in a playlist.
func (r *PlaylistRepository) ReplaceSongs(ctx context.Context, playlistID string, songIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete all existing songs
	_, err = tx.Exec(ctx, `DELETE FROM playlist_songs WHERE playlist_id = $1`, playlistID)
	if err != nil {
		return fmt.Errorf("delete existing songs: %w", err)
	}

	// Insert new ordered list
	for i, songID := range songIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO playlist_songs (playlist_id, song_id, position) VALUES ($1, $2, $3)`,
			playlistID, songID, i,
		)
		if err != nil {
			return fmt.Errorf("insert song at position %d: %w", i, err)
		}
	}

	// Update playlist updated_at
	_, err = tx.Exec(ctx, `UPDATE playlists SET updated_at = now() WHERE id = $1`, playlistID)
	if err != nil {
		return fmt.Errorf("update playlist timestamp: %w", err)
	}

	return tx.Commit(ctx)
}

// GetSongs returns all songs in a playlist, ordered by position.
func (r *PlaylistRepository) GetSongs(ctx context.Context, playlistID string) ([]*model.Song, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.id, s.user_id, s.source, s.source_file_id, s.title, s.artist, s.album,
		        s.duration_ms, s.mime_type, s.album_art_object_key, s.drive_modified_at, s.added_at
		 FROM songs s
		 JOIN playlist_songs ps ON ps.song_id = s.id
		 WHERE ps.playlist_id = $1
		 ORDER BY ps.position`, playlistID,
	)
	if err != nil {
		return nil, fmt.Errorf("get playlist songs: %w", err)
	}
	defer rows.Close()
	return scanSongs(rows)
}

// CountByUserID returns the total playlist count for a user.
func (r *PlaylistRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM playlists WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count playlists: %w", err)
	}
	return count, nil
}
