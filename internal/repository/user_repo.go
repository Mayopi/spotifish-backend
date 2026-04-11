package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spotifish/backend/internal/model"
)

// UserRepository handles database operations for users.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// FindByGoogleSub finds a user by their Google subject ID.
func (r *UserRepository) FindByGoogleSub(ctx context.Context, googleSub string) (*model.User, error) {
	var user model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, google_sub, email, display_name, created_at
		 FROM users WHERE google_sub = $1`, googleSub,
	).Scan(&user.ID, &user.GoogleSub, &user.Email, &user.DisplayName, &user.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by google sub: %w", err)
	}
	return &user, nil
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (google_sub, email, display_name)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		user.GoogleSub, user.Email, user.DisplayName,
	).Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// GetByID retrieves a user by their ID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, google_sub, email, display_name, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.GoogleSub, &user.Email, &user.DisplayName, &user.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}
