package handler

import (
	"net/http"
	"strconv"

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

// ListRecentlyPlayed handles GET /v1/playback/recent.
func (h *PlaybackHandler) ListRecentlyPlayed(c *gin.Context) {
	userID := middleware.GetUserID(c)
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: "limit must be a positive integer"},
		})
		return
	}
	if limit > 100 {
		limit = 100
	}

	songs, err := h.playbackRepo.GetRecentlyPlayed(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "list_recent_played_error", Message: err.Error()},
		})
		return
	}
	if songs == nil {
		songs = make([]*model.Song, 0)
	}

	c.JSON(http.StatusOK, gin.H{"songs": songs})
}

// RegisterRoutes registers playback routes on the given router group.
func (h *PlaybackHandler) RegisterRoutes(rg *gin.RouterGroup) {
	playback := rg.Group("/playback")
	{
		playback.POST("/events", h.RecordEvent)
		playback.GET("/recent", h.ListRecentlyPlayed)
	}
}
