package service

import (
	"context"

	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/repository"
)

// SettingsService handles user settings business logic.
type SettingsService struct {
	repo *repository.SettingsRepository
}

// NewSettingsService creates a new SettingsService.
func NewSettingsService(repo *repository.SettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}

// GetSettings returns the settings for a user.
func (s *SettingsService) GetSettings(ctx context.Context, userID string) (*model.UserSettings, error) {
	return s.repo.GetByUserID(ctx, userID)
}

// UpdateSettings applies a partial update to user settings.
func (s *SettingsService) UpdateSettings(ctx context.Context, userID string, patch *model.UserSettingsPatch) (*model.UserSettings, error) {
	// Get current settings
	current, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply patch
	if patch.Theme != nil {
		current.Theme = *patch.Theme
	}
	if patch.DefaultSortField != nil {
		current.DefaultSortField = *patch.DefaultSortField
	}
	if patch.DefaultSortDir != nil {
		current.DefaultSortDir = *patch.DefaultSortDir
	}

	return s.repo.Upsert(ctx, current)
}
