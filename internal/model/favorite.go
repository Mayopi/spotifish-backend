package model

import "time"

// Favorite represents a user's liked song.
type Favorite struct {
	UserID    string    `json:"userId"`
	SongID    string    `json:"songId"`
	CreatedAt time.Time `json:"createdAt"`
}
