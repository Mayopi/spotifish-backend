package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/rs/zerolog/log"
	"github.com/spotifish/backend/internal/crypto"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/repository"
)

// DriveService handles Google Drive integration.
type DriveService struct {
	driveRepo    *repository.DriveRepository
	songRepo     *repository.SongRepository
	oauthConfig  *oauth2.Config
	encryptionKey string
}

// NewDriveService creates a new DriveService.
func NewDriveService(
	driveRepo *repository.DriveRepository,
	songRepo *repository.SongRepository,
	clientID, clientSecret, redirectURI, encryptionKey string,
) *DriveService {
	return &DriveService{
		driveRepo: driveRepo,
		songRepo:  songRepo,
		oauthConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Scopes:       []string{drive.DriveReadonlyScope},
			Endpoint:     google.Endpoint,
		},
		encryptionKey: encryptionKey,
	}
}

// Connect exchanges an authorization code for Drive tokens and stores them encrypted.
func (s *DriveService) Connect(ctx context.Context, userID, authCode string) error {
	token, err := s.oauthConfig.Exchange(ctx, authCode)
	if err != nil {
		return fmt.Errorf("exchange auth code: %w", err)
	}

	encAccess, err := crypto.Encrypt([]byte(token.AccessToken), s.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}

	encRefresh, err := crypto.Encrypt([]byte(token.RefreshToken), s.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt refresh token: %w", err)
	}

	cred := &model.DriveCredential{
		UserID:           userID,
		EncryptedAccess:  encAccess,
		EncryptedRefresh: encRefresh,
		ExpiresAt:        token.Expiry,
		Scope:            drive.DriveReadonlyScope,
	}

	if err := s.driveRepo.SaveCredentials(ctx, cred); err != nil {
		return fmt.Errorf("save credentials: %w", err)
	}

	log.Info().Str("userId", userID).Msg("drive connected")
	return nil
}

// ListFolders lists subfolders of a given parent folder.
func (s *DriveService) ListFolders(ctx context.Context, userID, parentID string) ([]model.DriveFolderInfo, error) {
	client, err := s.getClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	if parentID == "" {
		parentID = "root"
	}

	query := fmt.Sprintf("'%s' in parents AND mimeType = 'application/vnd.google-apps.folder' AND trashed = false", parentID)
	fileList, err := srv.Files.List().Q(query).Fields("files(id, name)").OrderBy("name").Do()
	if err != nil {
		return nil, fmt.Errorf("list drive folders: %w", err)
	}

	folders := make([]model.DriveFolderInfo, len(fileList.Files))
	for i, f := range fileList.Files {
		folders[i] = model.DriveFolderInfo{ID: f.Id, Name: f.Name}
	}
	return folders, nil
}

// SetActiveFolder sets the user's active Drive folder.
func (s *DriveService) SetActiveFolder(ctx context.Context, userID, folderID, folderName string) error {
	return s.driveRepo.SetFolder(ctx, &model.DriveFolder{
		UserID:     userID,
		FolderID:   folderID,
		FolderName: folderName,
	})
}

// Disconnect revokes Drive access and optionally deletes the library.
func (s *DriveService) Disconnect(ctx context.Context, userID string, deleteLibrary bool) error {
	if deleteLibrary {
		if err := s.songRepo.DeleteAllDriveSongs(ctx, userID); err != nil {
			log.Warn().Err(err).Str("userId", userID).Msg("failed to delete drive songs during disconnect")
		}
	}

	if err := s.driveRepo.DeleteFolder(ctx, userID); err != nil {
		return fmt.Errorf("delete folder: %w", err)
	}
	if err := s.driveRepo.DeleteCredentials(ctx, userID); err != nil {
		return fmt.Errorf("delete credentials: %w", err)
	}

	log.Info().Str("userId", userID).Bool("deletedLibrary", deleteLibrary).Msg("drive disconnected")
	return nil
}

// GetDriveService returns a Google Drive API service for the user.
func (s *DriveService) GetDriveService(ctx context.Context, userID string) (*drive.Service, error) {
	client, err := s.getClient(ctx, userID)
	if err != nil {
		return nil, err
	}
	return drive.NewService(ctx, option.WithHTTPClient(client))
}

// GetClient returns an authenticated HTTP client for Drive API calls.
func (s *DriveService) GetClient(ctx context.Context, userID string) (*http.Client, error) {
	return s.getClient(ctx, userID)
}

// getClient creates an OAuth2-authenticated HTTP client with auto token refresh.
func (s *DriveService) getClient(ctx context.Context, userID string) (*http.Client, error) {
	cred, err := s.driveRepo.GetCredentials(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("drive not connected")
	}

	accessToken, err := crypto.Decrypt(cred.EncryptedAccess, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt access token: %w", err)
	}

	refreshToken, err := crypto.Decrypt(cred.EncryptedRefresh, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt refresh token: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  string(accessToken),
		RefreshToken: string(refreshToken),
		Expiry:       cred.ExpiresAt,
		TokenType:    "Bearer",
	}

	// Create a token source that auto-refreshes, then save new tokens when refreshed
	tokenSource := s.oauthConfig.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	// If the token was refreshed, persist the new tokens
	if newToken.AccessToken != string(accessToken) {
		encAccess, err := crypto.Encrypt([]byte(newToken.AccessToken), s.encryptionKey)
		if err == nil {
			encRefresh := cred.EncryptedRefresh
			if newToken.RefreshToken != "" && newToken.RefreshToken != string(refreshToken) {
				encRefresh, _ = crypto.Encrypt([]byte(newToken.RefreshToken), s.encryptionKey)
			}
			_ = s.driveRepo.SaveCredentials(ctx, &model.DriveCredential{
				UserID:           userID,
				EncryptedAccess:  encAccess,
				EncryptedRefresh: encRefresh,
				ExpiresAt:        newToken.Expiry,
				Scope:            cred.Scope,
				UpdatedAt:        time.Now(),
			})
		}
	}

	return oauth2.NewClient(ctx, oauth2.StaticTokenSource(newToken)), nil
}
