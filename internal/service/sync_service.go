package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/repository"
)

// audioMimeTypes defines the audio MIME types we recognize.
var audioMimeTypes = map[string]bool{
	"audio/mpeg":  true,
	"audio/mp3":   true,
	"audio/flac":  true,
	"audio/x-flac": true,
	"audio/ogg":   true,
	"audio/wav":   true,
	"audio/x-wav": true,
	"audio/aac":   true,
	"audio/mp4":   true,
	"audio/x-m4a": true,
}

// SyncService handles library sync operations.
type SyncService struct {
	syncRepo     *repository.SyncRepository
	songRepo     *repository.SongRepository
	driveRepo    *repository.DriveRepository
	driveSvc     *DriveService
	metaSvc      *MetadataService
	artSvc       *ArtStorageService
}

// NewSyncService creates a new SyncService.
func NewSyncService(
	syncRepo *repository.SyncRepository,
	songRepo *repository.SongRepository,
	driveRepo *repository.DriveRepository,
	driveSvc *DriveService,
	metaSvc *MetadataService,
	artSvc *ArtStorageService,
) *SyncService {
	return &SyncService{
		syncRepo:  syncRepo,
		songRepo:  songRepo,
		driveRepo: driveRepo,
		driveSvc:  driveSvc,
		metaSvc:   metaSvc,
		artSvc:    artSvc,
	}
}

// EnqueueSync creates a sync job if none is running for the user.
func (s *SyncService) EnqueueSync(ctx context.Context, userID string) (*model.SyncJob, error) {
	// Check for existing running/queued job
	existing, err := s.syncRepo.GetRunning(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check running sync: %w", err)
	}
	if existing != nil {
		return existing, nil // idempotent
	}

	job := &model.SyncJob{
		UserID: userID,
		State:  model.SyncStateQueued,
	}
	job, err = s.syncRepo.Create(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("create sync job: %w", err)
	}

	// Run sync in background
	go func() {
		bgCtx := context.Background()
		if err := s.RunSync(bgCtx, userID, job.ID); err != nil {
			log.Error().Err(err).Str("jobId", job.ID).Str("userId", userID).Msg("sync failed")
		}
	}()

	return job, nil
}

// GetStatus returns the latest sync job status.
func (s *SyncService) GetStatus(ctx context.Context, userID string) (*model.SyncJob, error) {
	return s.syncRepo.GetLatest(ctx, userID)
}

// RunSync executes the full sync process for a user.
func (s *SyncService) RunSync(ctx context.Context, userID, jobID string) error {
	// Get folder
	folder, err := s.driveRepo.GetFolder(ctx, userID)
	if err != nil || folder == nil {
		return s.failJob(ctx, jobID, "no drive folder connected")
	}

	// Update job to running
	now := time.Now()
	job := &model.SyncJob{
		ID:        jobID,
		State:     model.SyncStateRunning,
		StartedAt: &now,
	}
	if err := s.syncRepo.Update(ctx, job); err != nil {
		return fmt.Errorf("update job to running: %w", err)
	}

	// Get Drive service
	client, err := s.driveSvc.GetClient(ctx, userID)
	if err != nil {
		return s.failJob(ctx, jobID, fmt.Sprintf("drive client error: %v", err))
	}
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return s.failJob(ctx, jobID, fmt.Sprintf("drive service error: %v", err))
	}

	// Walk the folder recursively to find all audio files
	log.Info().Str("userId", userID).Str("folder", folder.FolderName).Msg("starting sync")
	var driveFiles []*drive.File
	if err := s.walkFolder(srv, folder.FolderID, &driveFiles); err != nil {
		return s.failJob(ctx, jobID, fmt.Sprintf("walk folder error: %v", err))
	}

	totalCount := len(driveFiles)
	job.TotalCount = &totalCount
	_ = s.syncRepo.Update(ctx, job)

	// Get existing file IDs for change detection
	existingIDs, err := s.songRepo.GetDriveFileIDs(ctx, userID)
	if err != nil {
		return s.failJob(ctx, jobID, fmt.Sprintf("get existing ids error: %v", err))
	}

	// Track which file IDs we've seen to detect deletions
	seenFileIDs := make(map[string]bool)
	processed := 0

	for _, file := range driveFiles {
		seenFileIDs[file.Id] = true

		// Check if file needs update
		modifiedTime, _ := time.Parse(time.RFC3339, file.ModifiedTime)
		existing, _ := s.songRepo.GetBySourceFileID(ctx, userID, "drive", file.Id)

		if existing != nil && existing.DriveModifiedAt != nil &&
			!modifiedTime.After(*existing.DriveModifiedAt) {
			processed++
			continue // unchanged
		}

		// Download file for metadata extraction
		resp, err := srv.Files.Get(file.Id).Download()
		if err != nil {
			log.Warn().Err(err).Str("fileId", file.Id).Msg("failed to download file for metadata")
			processed++
			continue
		}

		// Read the file into memory for metadata extraction
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Warn().Err(err).Str("fileId", file.Id).Msg("failed to read file")
			processed++
			continue
		}

		// Extract metadata
		reader := bytes.NewReader(buf.Bytes())
		meta, pictureData, err := s.metaSvc.Extract(reader)
		if err != nil {
			log.Warn().Err(err).Str("fileId", file.Id).Msg("failed to extract metadata")
		}
		if meta == nil {
			meta = &SongMetadata{Title: file.Name, Artist: "Unknown", Album: "Unknown"}
		}

		// If title is still "Unknown", use filename without extension
		if meta.Title == "Unknown" || meta.Title == "" {
			name := file.Name
			if idx := strings.LastIndex(name, "."); idx != -1 {
				name = name[:idx]
			}
			meta.Title = name
		}

		// Save album art
		var artKey string
		if pictureData != nil && len(pictureData) > 0 {
			artKey = uuid.New().String()
			if err := s.artSvc.SaveArt(artKey, pictureData); err != nil {
				log.Warn().Err(err).Msg("failed to save album art")
				artKey = ""
			}
		}

		// Upsert song
		song := &model.Song{
			UserID:           userID,
			Source:           "drive",
			SourceFileID:     file.Id,
			Title:            meta.Title,
			Artist:           meta.Artist,
			Album:            meta.Album,
			DurationMs:       meta.Duration,
			MimeType:         file.MimeType,
			AlbumArtObjectKey: artKey,
			DriveModifiedAt:  &modifiedTime,
		}
		if _, err := s.songRepo.Upsert(ctx, song); err != nil {
			log.Warn().Err(err).Str("fileId", file.Id).Msg("failed to upsert song")
		}

		processed++
		// Update progress periodically
		if processed%10 == 0 {
			job.ProcessedCount = processed
			_ = s.syncRepo.Update(ctx, job)
		}
	}

	// Delete songs for files that no longer exist in Drive
	var deleteIDs []string
	for id := range existingIDs {
		if !seenFileIDs[id] {
			deleteIDs = append(deleteIDs, id)
		}
	}
	if len(deleteIDs) > 0 {
		deleted, _ := s.songRepo.DeleteBySourceFileIDs(ctx, userID, deleteIDs)
		log.Info().Int("deleted", deleted).Msg("removed songs for deleted drive files")
	}

	// Update folder last synced
	_ = s.driveRepo.UpdateLastSyncedAt(ctx, userID)

	// Mark job as succeeded
	finishedAt := time.Now()
	job.State = model.SyncStateSucceeded
	job.ProcessedCount = processed
	job.FinishedAt = &finishedAt
	if err := s.syncRepo.Update(ctx, job); err != nil {
		return fmt.Errorf("update job to succeeded: %w", err)
	}

	log.Info().Str("userId", userID).Int("processed", processed).Int("total", totalCount).Msg("sync completed")
	return nil
}

// walkFolder recursively lists all audio files in a Drive folder.
func (s *SyncService) walkFolder(srv *drive.Service, folderID string, files *[]*drive.File) error {
	pageToken := ""
	for {
		query := fmt.Sprintf("'%s' in parents AND trashed = false", folderID)
		call := srv.Files.List().Q(query).Fields("nextPageToken, files(id, name, mimeType, modifiedTime)").PageSize(1000)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		result, err := call.Do()
		if err != nil {
			return fmt.Errorf("list files: %w", err)
		}

		for _, f := range result.Files {
			if f.MimeType == "application/vnd.google-apps.folder" {
				// Recurse into subfolder
				if err := s.walkFolder(srv, f.Id, files); err != nil {
					return err
				}
			} else if audioMimeTypes[f.MimeType] {
				*files = append(*files, f)
			}
		}

		pageToken = result.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return nil
}

// failJob marks a sync job as failed.
func (s *SyncService) failJob(ctx context.Context, jobID, errMsg string) error {
	now := time.Now()
	job := &model.SyncJob{
		ID:         jobID,
		State:      model.SyncStateFailed,
		LastError:  errMsg,
		FinishedAt: &now,
	}
	return s.syncRepo.Update(ctx, job)
}
