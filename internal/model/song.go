package model

import "time"

// Song represents a song in the library.
type Song struct {
	ID               string     `json:"id"`
	UserID           string     `json:"userId"`
	Source           string     `json:"source"` // "drive" or "local"
	SourceFileID     string     `json:"sourceFileId"`
	Title            string     `json:"title"`
	Artist           string     `json:"artist"`
	Album            string     `json:"album"`
	DurationMs       int64      `json:"durationMs"`
	MimeType         string     `json:"mimeType,omitempty"`
	AlbumArtObjectKey string    `json:"albumArtObjectKey,omitempty"`
	DriveModifiedAt  *time.Time `json:"driveModifiedAt,omitempty"`
	AddedAt          time.Time  `json:"addedAt"`
}

// ArtistGroup represents a grouped artist entry for browsing.
type ArtistGroup struct {
	ID        string `json:"id"`   // base64url-encoded artist name
	Name      string `json:"name"`
	SongCount int    `json:"songCount"`
	ArtURL    string `json:"artUrl,omitempty"`
}

// AlbumGroup represents a grouped album entry for browsing.
type AlbumGroup struct {
	ID        string `json:"id"`   // base64url-encoded album+artist
	Name      string `json:"name"`
	Artist    string `json:"artist"`
	SongCount int    `json:"songCount"`
	ArtURL    string `json:"artUrl,omitempty"`
}

// SongPage represents a paginated result of songs.
type SongPage struct {
	Songs      []*Song `json:"songs"`
	NextCursor string  `json:"nextCursor,omitempty"`
}
