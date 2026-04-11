package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spotifish/backend/internal/model"
)

// PlaybackEventRepository handles database operations for playback events.
type PlaybackEventRepository struct {
	pool *pgxpool.Pool
}

// NewPlaybackEventRepository creates a new PlaybackEventRepository.
func NewPlaybackEventRepository(pool *pgxpool.Pool) *PlaybackEventRepository {
	return &PlaybackEventRepository{pool: pool}
}

// Create records a playback event.
func (r *PlaybackEventRepository) Create(ctx context.Context, event *model.PlaybackEvent) (*model.PlaybackEvent, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO playback_events (user_id, song_id, event_type, position_ms)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		event.UserID, event.SongID, event.EventType, event.Position,
	).Scan(&event.ID, &event.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("create playback event: %w", err)
	}
	return event, nil
}

// GetRecentlyPlayed returns recently played song IDs (unique, most recent first).
func (r *PlaybackEventRepository) GetRecentlyPlayed(ctx context.Context, userID string, limit int) ([]*model.Song, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT ON (pe.song_id)
		        s.id, s.user_id, s.source, s.source_file_id, s.title, s.artist, s.album,
		        s.duration_ms, s.mime_type, s.album_art_object_key, s.drive_modified_at, s.added_at
		 FROM playback_events pe
		 JOIN songs s ON s.id = pe.song_id
		 WHERE pe.user_id = $1 AND pe.event_type IN ('started', 'completed')
		 ORDER BY pe.song_id, pe.created_at DESC
		 LIMIT $2`, userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get recently played: %w", err)
	}
	defer rows.Close()
	return scanSongs(rows)
}
