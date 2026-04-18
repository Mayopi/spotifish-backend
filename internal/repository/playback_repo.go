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
		`SELECT s.id, s.user_id, s.source, s.source_file_id, s.title, s.artist, s.album,
		        s.duration_ms, s.mime_type, s.album_art_object_key, s.drive_modified_at, s.added_at
		 FROM songs s
		 JOIN (
		     SELECT pe.song_id, MAX(pe.created_at) AS last_played_at
		     FROM playback_events pe
		     WHERE pe.user_id = $1 AND pe.event_type IN ('started', 'completed')
		     GROUP BY pe.song_id
		     ORDER BY MAX(pe.created_at) DESC
		     LIMIT $2
		 ) recent ON recent.song_id = s.id
		 WHERE s.user_id = $1
		 ORDER BY recent.last_played_at DESC`, userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get recently played: %w", err)
	}
	defer rows.Close()
	return scanSongs(rows)
}

// CountDistinctSongsByUserID returns unique songs with playback history for a user.
func (r *PlaybackEventRepository) CountDistinctSongsByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT song_id)
		 FROM playback_events
		 WHERE user_id = $1 AND event_type IN ('started', 'completed')`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count distinct playback songs: %w", err)
	}
	return count, nil
}

// CountEventsByUserID returns total playback event count for a user.
func (r *PlaybackEventRepository) CountEventsByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM playback_events WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count playback events: %w", err)
	}
	return count, nil
}
