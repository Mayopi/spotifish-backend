package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spotifish/backend/internal/model"
)

// AuthRepository handles database operations for refresh tokens.
type AuthRepository struct {
	pool *pgxpool.Pool
}

// NewAuthRepository creates a new AuthRepository.
func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

// CreateRefreshToken stores a new hashed refresh token.
func (r *AuthRepository) CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO auth_refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

// FindRefreshToken retrieves a refresh token by its hash (only non-revoked, non-expired).
func (r *AuthRepository) FindRefreshToken(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at
		 FROM auth_refresh_tokens
		 WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`,
		tokenHash,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find refresh token: %w", err)
	}
	return &token, nil
}

// RevokeRefreshToken marks a refresh token as revoked.
func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE auth_refresh_tokens SET revoked_at = now() WHERE token_hash = $1`,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user.
func (r *AuthRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE auth_refresh_tokens SET revoked_at = now()
		 WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}
	return nil
}
