package model

import "time"

// SyncJob represents a library sync job.
type SyncJob struct {
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	State          string     `json:"state"` // "queued", "running", "paused", "succeeded", "failed"
	ProcessedCount int        `json:"processedCount"`
	TotalCount     *int       `json:"totalCount,omitempty"`
	LastError      string     `json:"lastError,omitempty"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	PausedAt       *time.Time `json:"pausedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// SyncState constants.
const (
	SyncStateQueued    = "queued"
	SyncStateRunning   = "running"
	SyncStatePaused    = "paused"
	SyncStateSucceeded = "succeeded"
	SyncStateFailed    = "failed"
)
