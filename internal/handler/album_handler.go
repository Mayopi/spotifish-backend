package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/service"
)

// AlbumHandler handles album endpoints.
type AlbumHandler struct {
	libSvc *service.LibraryService
}

// NewAlbumHandler creates a new AlbumHandler.
func NewAlbumHandler(libSvc *service.LibraryService) *AlbumHandler {
	return &AlbumHandler{libSvc: libSvc}
}

// ListAlbums handles GET /v1/albums.
func (h *AlbumHandler) ListAlbums(c *gin.Context) {
	userID := middleware.GetUserID(c)

	albums, err := h.libSvc.GetAlbums(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "list_albums_error", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"albums": albums})
}

// GetAlbumSongs handles GET /v1/albums/:id/songs.
func (h *AlbumHandler) GetAlbumSongs(c *gin.Context) {
	userID := middleware.GetUserID(c)
	albumID := c.Param("id")

	songs, err := h.libSvc.GetAlbumSongs(c.Request.Context(), userID, albumID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_album_id", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"songs": songs})
}

// RegisterRoutes registers album routes on the given router group.
func (h *AlbumHandler) RegisterRoutes(rg *gin.RouterGroup) {
	albums := rg.Group("/albums")
	{
		albums.GET("", h.ListAlbums)
		albums.GET("/:id/songs", h.GetAlbumSongs)
	}
}
