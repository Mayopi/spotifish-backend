package model

import "time"

// User represents an authenticated user.
type User struct {
	ID          string    `json:"id"`
	GoogleSub   string    `json:"googleSub"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
}

// UserStats represents aggregated user library and playback statistics.
type UserStats struct {
	SongCount           int `json:"songCount"`
	FavoriteSongCount   int `json:"favoriteSongCount"`
	PlaylistCount       int `json:"playlistCount"`
	RecentlyPlayedCount int `json:"recentlyPlayedCount"`
	PlaybackEventCount  int `json:"playbackEventCount"`
}
