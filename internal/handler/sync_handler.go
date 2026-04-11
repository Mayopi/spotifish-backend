package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/service"
)

// SyncHandler handles sync endpoints.
type SyncHandler struct {
	syncSvc *service.SyncService
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(syncSvc *service.SyncService) *SyncHandler {
	return &SyncHandler{syncSvc: syncSvc}
}

// RunSync handles POST /v1/sync/run.
func (h *SyncHandler) RunSync(c *gin.Context) {
	userID := middleware.GetUserID(c)

	job, err := h.syncSvc.EnqueueSync(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "sync_failed", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"syncJobId": job.ID,
		"state":     job.State,
	})
}

// GetStatus handles GET /v1/sync/status.
func (h *SyncHandler) GetStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)

	job, err := h.syncSvc.GetStatus(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "sync_status_error", Message: err.Error()},
		})
		return
	}

	if job == nil {
		c.JSON(http.StatusOK, gin.H{
			"state":        "none",
			"lastSyncedAt": nil,
		})
		return
	}

	c.JSON(http.StatusOK, job)
}

// RegisterRoutes registers sync routes on the given router group.
func (h *SyncHandler) RegisterRoutes(rg *gin.RouterGroup) {
	sync := rg.Group("/sync")
	{
		sync.POST("/run", h.RunSync)
		sync.GET("/status", h.GetStatus)
	}
}
