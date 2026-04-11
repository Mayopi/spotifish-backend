package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/repository"
)

// PlaybackHandler handles playback event endpoints.
type PlaybackHandler struct {
	playbackRepo *repository.PlaybackEventRepository
}

// NewPlaybackHandler creates a new PlaybackHandler.
func NewPlaybackHandler(playbackRepo *repository.PlaybackEventRepository) *PlaybackHandler {
	return &PlaybackHandler{playbackRepo: playbackRepo}
}

type playbackEventRequest struct {
	SongID    string `json:"songId" binding:"required"`
	EventType string `json:"eventType" binding:"required"`
	Position  int64  `json:"positionMs"`
}

// RecordEvent handles POST /v1/playback/events.
func (h *PlaybackHandler) RecordEvent(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req playbackEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	// Validate event type
	switch req.EventType {
	case "started", "completed", "skipped":
		// valid
	default:
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_event_type", Message: "eventType must be 'started', 'completed', or 'skipped'"},
		})
		return
	}

	event := &model.PlaybackEvent{
		UserID:    userID,
		SongID:    req.SongID,
		EventType: req.EventType,
		Position:  req.Position,
	}

	if _, err := h.playbackRepo.Create(c.Request.Context(), event); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "record_event_error", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// RegisterRoutes registers playback routes on the given router group.
func (h *PlaybackHandler) RegisterRoutes(rg *gin.RouterGroup) {
	playback := rg.Group("/playback")
	{
		playback.POST("/events", h.RecordEvent)
	}
}
