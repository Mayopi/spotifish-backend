package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spotifish/backend/internal/model"
)

// SettingsRepository handles database operations for user settings.
type SettingsRepository struct {
	pool *pgxpool.Pool
}

// NewSettingsRepository creates a new SettingsRepository.
func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

// GetByUserID retrieves settings for a user.
func (r *SettingsRepository) GetByUserID(ctx context.Context, userID string) (*model.UserSettings, error) {
	var s model.UserSettings
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, theme, default_sort_field, default_sort_dir, updated_at
		 FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&s.UserID, &s.Theme, &s.DefaultSortField, &s.DefaultSortDir, &s.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		// Return defaults
		return &model.UserSettings{
			UserID:           userID,
			Theme:            "system",
			DefaultSortField: "title",
			DefaultSortDir:   "asc",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return &s, nil
}

// Upsert creates or updates user settings.
func (r *SettingsRepository) Upsert(ctx context.Context, s *model.UserSettings) (*model.UserSettings, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO user_settings (user_id, theme, default_sort_field, default_sort_dir, updated_at)
		 VALUES ($1, $2, $3, $4, now())
		 ON CONFLICT (user_id) DO UPDATE SET
		     theme = EXCLUDED.theme,
		     default_sort_field = EXCLUDED.default_sort_field,
		     default_sort_dir = EXCLUDED.default_sort_dir,
		     updated_at = now()
		 RETURNING updated_at`,
		s.UserID, s.Theme, s.DefaultSortField, s.DefaultSortDir,
	).Scan(&s.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("upsert settings: %w", err)
	}
	return s, nil
}
