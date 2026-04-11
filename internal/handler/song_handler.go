package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/service"
)

// SongHandler handles song/library endpoints.
type SongHandler struct {
	libSvc *service.LibraryService
}

// NewSongHandler creates a new SongHandler.
func NewSongHandler(libSvc *service.LibraryService) *SongHandler {
	return &SongHandler{libSvc: libSvc}
}

// ListSongs handles GET /v1/songs.
func (h *SongHandler) ListSongs(c *gin.Context) {
	userID := middleware.GetUserID(c)
	cursor := c.Query("cursor")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	sortBy := c.DefaultQuery("sort", "title")
	sortDir := c.DefaultQuery("dir", "asc")

	songs, nextCursor, err := h.libSvc.ListSongs(c.Request.Context(), userID, cursor, limit, sortBy, sortDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "list_songs_error", Message: err.Error()},
		})
		return
	}

	resp := gin.H{"songs": songs}
	if nextCursor != "" {
		resp["nextCursor"] = nextCursor
	}
	c.JSON(http.StatusOK, resp)
}

// GetSong handles GET /v1/songs/:id.
func (h *SongHandler) GetSong(c *gin.Context) {
	userID := middleware.GetUserID(c)
	songID := c.Param("id")

	song, err := h.libSvc.GetSong(c.Request.Context(), userID, songID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "get_song_error", Message: err.Error()},
		})
		return
	}
	if song == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: model.APIError{Code: "not_found", Message: "song not found"},
		})
		return
	}

	c.JSON(http.StatusOK, song)
}

// SearchSongs handles GET /v1/songs/search.
func (h *SongHandler) SearchSongs(c *gin.Context) {
	userID := middleware.GetUserID(c)
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: "q parameter is required"},
		})
		return
	}

	songs, err := h.libSvc.SearchSongs(c.Request.Context(), userID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "search_error", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"songs": songs})
}

// RegisterRoutes registers song routes on the given router group.
func (h *SongHandler) RegisterRoutes(rg *gin.RouterGroup) {
	songs := rg.Group("/songs")
	{
		songs.GET("", h.ListSongs)
		songs.GET("/search", h.SearchSongs)
		songs.GET("/:id", h.GetSong)
	}
}
