package worker

import (
	"context"
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"github.com/spotifish/backend/internal/repository"
	"github.com/spotifish/backend/internal/service"
)

// SyncWorker manages scheduled and on-demand sync jobs.
type SyncWorker struct {
	syncSvc   *service.SyncService
	driveRepo *repository.DriveRepository
	cron      *cron.Cron
	mu        sync.Mutex
	running   map[string]bool // userID -> is sync running
}

// NewSyncWorker creates a new SyncWorker.
func NewSyncWorker(syncSvc *service.SyncService, driveRepo *repository.DriveRepository) *SyncWorker {
	return &SyncWorker{
		syncSvc:   syncSvc,
		driveRepo: driveRepo,
		cron:      cron.New(),
		running:   make(map[string]bool),
	}
}

// Start begins the cron scheduler for periodic syncs.
func (w *SyncWorker) Start(intervalHours int) error {
	spec := fmt.Sprintf("0 */%d * * *", intervalHours) // every N hours at minute 0
	_, err := w.cron.AddFunc(spec, func() {
		w.syncAllUsers()
	})
	if err != nil {
		return fmt.Errorf("add cron job: %w", err)
	}

	w.cron.Start()
	log.Info().Int("intervalHours", intervalHours).Msg("sync worker started")
	return nil
}

// Stop gracefully stops the cron scheduler.
func (w *SyncWorker) Stop() {
	ctx := w.cron.Stop()
	<-ctx.Done()
	log.Info().Msg("sync worker stopped")
}

// syncAllUsers triggers sync for all users with a connected Drive folder.
func (w *SyncWorker) syncAllUsers() {
	ctx := context.Background()
	userIDs, err := w.driveRepo.GetAllConnectedUsers(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get connected users for scheduled sync")
		return
	}

	log.Info().Int("userCount", len(userIDs)).Msg("starting scheduled sync for all connected users")

	for _, userID := range userIDs {
		w.mu.Lock()
		if w.running[userID] {
			w.mu.Unlock()
			continue // already running for this user
		}
		w.running[userID] = true
		w.mu.Unlock()

		go func(uid string) {
			defer func() {
				w.mu.Lock()
				delete(w.running, uid)
				w.mu.Unlock()
			}()

			if _, err := w.syncSvc.EnqueueSync(ctx, uid); err != nil {
				log.Error().Err(err).Str("userId", uid).Msg("scheduled sync failed")
			}
		}(userID)
	}
}
