package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spotifish/backend/internal/model"
)

// DriveRepository handles database operations for Drive credentials and folders.
type DriveRepository struct {
	pool *pgxpool.Pool
}

// NewDriveRepository creates a new DriveRepository.
func NewDriveRepository(pool *pgxpool.Pool) *DriveRepository {
	return &DriveRepository{pool: pool}
}

// SaveCredentials stores or updates encrypted Drive credentials.
func (r *DriveRepository) SaveCredentials(ctx context.Context, cred *model.DriveCredential) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO drive_credentials (user_id, encrypted_access, encrypted_refresh, expires_at, scope, updated_at)
		 VALUES ($1, $2, $3, $4, $5, now())
		 ON CONFLICT (user_id) DO UPDATE SET
		     encrypted_access = EXCLUDED.encrypted_access,
		     encrypted_refresh = EXCLUDED.encrypted_refresh,
		     expires_at = EXCLUDED.expires_at,
		     scope = EXCLUDED.scope,
		     updated_at = now()`,
		cred.UserID, cred.EncryptedAccess, cred.EncryptedRefresh, cred.ExpiresAt, cred.Scope,
	)
	if err != nil {
		return fmt.Errorf("save drive credentials: %w", err)
	}
	return nil
}

// GetCredentials retrieves encrypted Drive credentials for a user.
func (r *DriveRepository) GetCredentials(ctx context.Context, userID string) (*model.DriveCredential, error) {
	var cred model.DriveCredential
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, encrypted_access, encrypted_refresh, expires_at, scope, updated_at
		 FROM drive_credentials WHERE user_id = $1`, userID,
	).Scan(&cred.UserID, &cred.EncryptedAccess, &cred.EncryptedRefresh, &cred.ExpiresAt, &cred.Scope, &cred.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get drive credentials: %w", err)
	}
	return &cred, nil
}

// DeleteCredentials removes Drive credentials for a user.
func (r *DriveRepository) DeleteCredentials(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM drive_credentials WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete drive credentials: %w", err)
	}
	return nil
}

// SetFolder sets or updates the active Drive folder for a user.
func (r *DriveRepository) SetFolder(ctx context.Context, folder *model.DriveFolder) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO drive_folders (user_id, folder_id, folder_name)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE SET
		     folder_id = EXCLUDED.folder_id,
		     folder_name = EXCLUDED.folder_name`,
		folder.UserID, folder.FolderID, folder.FolderName,
	)
	if err != nil {
		return fmt.Errorf("set drive folder: %w", err)
	}
	return nil
}

// GetFolder retrieves the active Drive folder for a user.
func (r *DriveRepository) GetFolder(ctx context.Context, userID string) (*model.DriveFolder, error) {
	var folder model.DriveFolder
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, folder_id, folder_name, last_synced_at
		 FROM drive_folders WHERE user_id = $1`, userID,
	).Scan(&folder.UserID, &folder.FolderID, &folder.FolderName, &folder.LastSyncedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get drive folder: %w", err)
	}
	return &folder, nil
}

// DeleteFolder removes the Drive folder for a user.
func (r *DriveRepository) DeleteFolder(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM drive_folders WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete drive folder: %w", err)
	}
	return nil
}

// UpdateLastSyncedAt updates the last synced timestamp for a folder.
func (r *DriveRepository) UpdateLastSyncedAt(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE drive_folders SET last_synced_at = now() WHERE user_id = $1`, userID,
	)
	if err != nil {
		return fmt.Errorf("update last synced at: %w", err)
	}
	return nil
}

// GetAllConnectedUsers returns user IDs for all users with a connected Drive folder.
func (r *DriveRepository) GetAllConnectedUsers(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT df.user_id FROM drive_folders df
		 JOIN drive_credentials dc ON dc.user_id = df.user_id`)
	if err != nil {
		return nil, fmt.Errorf("get connected users: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan user id: %w", err)
		}
		userIDs = append(userIDs, id)
	}
	return userIDs, rows.Err()
}
