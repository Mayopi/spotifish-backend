package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/service"
)

// PlaylistHandler handles playlist endpoints.
type PlaylistHandler struct {
	playlistSvc *service.PlaylistService
}

// NewPlaylistHandler creates a new PlaylistHandler.
func NewPlaylistHandler(playlistSvc *service.PlaylistService) *PlaylistHandler {
	return &PlaylistHandler{playlistSvc: playlistSvc}
}

type createPlaylistRequest struct {
	Name string `json:"name" binding:"required"`
}

type renamePlaylistRequest struct {
	Name string `json:"name" binding:"required"`
}

type addSongRequest struct {
	SongID string `json:"songId" binding:"required"`
}

type replaceSongsRequest struct {
	SongIDs []string `json:"songIds" binding:"required"`
}

// ListPlaylists handles GET /v1/playlists.
func (h *PlaylistHandler) ListPlaylists(c *gin.Context) {
	userID := middleware.GetUserID(c)

	playlists, err := h.playlistSvc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "list_playlists_error", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"playlists": playlists})
}

// CreatePlaylist handles POST /v1/playlists.
func (h *PlaylistHandler) CreatePlaylist(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req createPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: "name is required"},
		})
		return
	}

	playlist, err := h.playlistSvc.Create(c.Request.Context(), userID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "create_playlist_error", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusCreated, playlist)
}

// RenamePlaylist handles PATCH /v1/playlists/:id.
func (h *PlaylistHandler) RenamePlaylist(c *gin.Context) {
	userID := middleware.GetUserID(c)
	playlistID := c.Param("id")
	var req renamePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: "name is required"},
		})
		return
	}

	if err := h.playlistSvc.Rename(c.Request.Context(), userID, playlistID, req.Name); err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: model.APIError{Code: "playlist_not_found", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// DeletePlaylist handles DELETE /v1/playlists/:id.
func (h *PlaylistHandler) DeletePlaylist(c *gin.Context) {
	userID := middleware.GetUserID(c)
	playlistID := c.Param("id")

	if err := h.playlistSvc.Delete(c.Request.Context(), userID, playlistID); err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: model.APIError{Code: "playlist_not_found", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// AddSong handles POST /v1/playlists/:id/songs.
func (h *PlaylistHandler) AddSong(c *gin.Context) {
	userID := middleware.GetUserID(c)
	playlistID := c.Param("id")
	var req addSongRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: "songId is required"},
		})
		return
	}

	if err := h.playlistSvc.AddSong(c.Request.Context(), userID, playlistID, req.SongID); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "add_song_error", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveSong handles DELETE /v1/playlists/:id/songs/:songId.
func (h *PlaylistHandler) RemoveSong(c *gin.Context) {
	userID := middleware.GetUserID(c)
	playlistID := c.Param("id")
	songID := c.Param("songId")

	if err := h.playlistSvc.RemoveSong(c.Request.Context(), userID, playlistID, songID); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "remove_song_error", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ReplaceSongs handles PUT /v1/playlists/:id/songs.
func (h *PlaylistHandler) ReplaceSongs(c *gin.Context) {
	userID := middleware.GetUserID(c)
	playlistID := c.Param("id")
	var req replaceSongsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: "songIds array is required"},
		})
		return
	}

	if err := h.playlistSvc.ReplaceSongs(c.Request.Context(), userID, playlistID, req.SongIDs); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "replace_songs_error", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// RegisterRoutes registers playlist routes on the given router group.
func (h *PlaylistHandler) RegisterRoutes(rg *gin.RouterGroup) {
	playlists := rg.Group("/playlists")
	{
		playlists.GET("", h.ListPlaylists)
		playlists.POST("", h.CreatePlaylist)
		playlists.PATCH("/:id", h.RenamePlaylist)
		playlists.DELETE("/:id", h.DeletePlaylist)
		playlists.POST("/:id/songs", h.AddSong)
		playlists.DELETE("/:id/songs/:songId", h.RemoveSong)
		playlists.PUT("/:id/songs", h.ReplaceSongs)
	}
}
