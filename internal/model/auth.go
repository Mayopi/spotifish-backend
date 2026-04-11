package model

import "time"

// TokenPair holds the access and refresh tokens issued to a client.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// RefreshToken represents a stored refresh token in the database.
type RefreshToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expiresAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
}
