package model

import "time"

// UserSettings holds user-scoped configuration.
type UserSettings struct {
	UserID           string    `json:"userId"`
	Theme            string    `json:"theme"`
	DefaultSortField string    `json:"defaultSortField"`
	DefaultSortDir   string    `json:"defaultSortDirection"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// UserSettingsPatch represents a partial update to user settings.
type UserSettingsPatch struct {
	Theme            *string `json:"theme,omitempty"`
	DefaultSortField *string `json:"defaultSortField,omitempty"`
	DefaultSortDir   *string `json:"defaultSortDirection,omitempty"`
}
