package model

import "time"

// DriveCredential represents stored Drive OAuth credentials (encrypted).
type DriveCredential struct {
	UserID           string    `json:"userId"`
	EncryptedAccess  []byte    `json:"-"`
	EncryptedRefresh []byte    `json:"-"`
	ExpiresAt        time.Time `json:"expiresAt"`
	Scope            string    `json:"scope"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// DriveFolder represents the user's selected Drive folder.
type DriveFolder struct {
	UserID       string     `json:"userId"`
	FolderID     string     `json:"folderId"`
	FolderName   string     `json:"folderName"`
	LastSyncedAt *time.Time `json:"lastSyncedAt,omitempty"`
}

// DriveFolderInfo is a folder entry returned from the Drive API.
type DriveFolderInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PlaybackEvent represents a playback event sent by the client.
type PlaybackEvent struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	SongID    string    `json:"songId"`
	EventType string    `json:"eventType"` // "started", "completed", "skipped"
	Position  int64     `json:"positionMs"`
	CreatedAt time.Time `json:"createdAt"`
}
