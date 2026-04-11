package model

import "time"

// Playlist represents a user's playlist.
type Playlist struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	SongCount int       `json:"songCount"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// PlaylistSong represents a song's membership in a playlist.
type PlaylistSong struct {
	PlaylistID string `json:"playlistId"`
	SongID     string `json:"songId"`
	Position   int    `json:"position"`
}

// PlaylistDetail is a playlist with its songs included.
type PlaylistDetail struct {
	Playlist
	Songs []*Song `json:"songs"`
}
