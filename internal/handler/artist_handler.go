package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/service"
)

// ArtistHandler handles artist endpoints.
type ArtistHandler struct {
	libSvc *service.LibraryService
}

// NewArtistHandler creates a new ArtistHandler.
func NewArtistHandler(libSvc *service.LibraryService) *ArtistHandler {
	return &ArtistHandler{libSvc: libSvc}
}

// ListArtists handles GET /v1/artists.
func (h *ArtistHandler) ListArtists(c *gin.Context) {
	userID := middleware.GetUserID(c)

	artists, err := h.libSvc.GetArtists(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "list_artists_error", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"artists": artists})
}

// GetArtistSongs handles GET /v1/artists/:id/songs.
func (h *ArtistHandler) GetArtistSongs(c *gin.Context) {
	userID := middleware.GetUserID(c)
	artistID := c.Param("id")

	songs, err := h.libSvc.GetArtistSongs(c.Request.Context(), userID, artistID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_artist_id", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"songs": songs})
}

// RegisterRoutes registers artist routes on the given router group.
func (h *ArtistHandler) RegisterRoutes(rg *gin.RouterGroup) {
	artists := rg.Group("/artists")
	{
		artists.GET("", h.ListArtists)
		artists.GET("/:id/songs", h.GetArtistSongs)
	}
}
