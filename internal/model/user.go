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
