package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spotifish/backend/internal/model"
)

// SyncRepository handles database operations for sync jobs.
type SyncRepository struct {
	pool *pgxpool.Pool
}

// NewSyncRepository creates a new SyncRepository.
func NewSyncRepository(pool *pgxpool.Pool) *SyncRepository {
	return &SyncRepository{pool: pool}
}

// Create inserts a new sync job.
func (r *SyncRepository) Create(ctx context.Context, job *model.SyncJob) (*model.SyncJob, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO sync_jobs (user_id, state)
		 VALUES ($1, $2)
		 RETURNING id, created_at`,
		job.UserID, job.State,
	).Scan(&job.ID, &job.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("create sync job: %w", err)
	}
	return job, nil
}

// GetLatest returns the most recent sync job for a user.
func (r *SyncRepository) GetLatest(ctx context.Context, userID string) (*model.SyncJob, error) {
	var job model.SyncJob
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, state, processed_count, total_count, last_error, started_at, finished_at, created_at
		 FROM sync_jobs WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT 1`, userID,
	).Scan(&job.ID, &job.UserID, &job.State, &job.ProcessedCount, &job.TotalCount,
		&job.LastError, &job.StartedAt, &job.FinishedAt, &job.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest sync job: %w", err)
	}
	return &job, nil
}

// GetRunning returns a currently running/queued sync job for a user.
func (r *SyncRepository) GetRunning(ctx context.Context, userID string) (*model.SyncJob, error) {
	var job model.SyncJob
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, state, processed_count, total_count, last_error, started_at, finished_at, created_at
		 FROM sync_jobs
		 WHERE user_id = $1 AND state IN ('queued', 'running')
		 ORDER BY created_at DESC LIMIT 1`, userID,
	).Scan(&job.ID, &job.UserID, &job.State, &job.ProcessedCount, &job.TotalCount,
		&job.LastError, &job.StartedAt, &job.FinishedAt, &job.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get running sync job: %w", err)
	}
	return &job, nil
}

// Update updates a sync job's fields.
func (r *SyncRepository) Update(ctx context.Context, job *model.SyncJob) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sync_jobs SET
		     state = $2, processed_count = $3, total_count = $4,
		     last_error = $5, started_at = $6, finished_at = $7
		 WHERE id = $1`,
		job.ID, job.State, job.ProcessedCount, job.TotalCount,
		job.LastError, job.StartedAt, job.FinishedAt,
	)
	if err != nil {
		return fmt.Errorf("update sync job: %w", err)
	}
	return nil
}
