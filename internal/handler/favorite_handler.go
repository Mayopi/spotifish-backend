package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/service"
)

// FavoriteHandler handles favorite endpoints.
type FavoriteHandler struct {
	favSvc *service.FavoriteService
}

// NewFavoriteHandler creates a new FavoriteHandler.
func NewFavoriteHandler(favSvc *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{favSvc: favSvc}
}

// ListFavorites handles GET /v1/favorites.
func (h *FavoriteHandler) ListFavorites(c *gin.Context) {
	userID := middleware.GetUserID(c)

	songs, err := h.favSvc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "list_favorites_error", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"favorites": songs})
}

// AddFavorite handles PUT /v1/favorites/:songId.
func (h *FavoriteHandler) AddFavorite(c *gin.Context) {
	userID := middleware.GetUserID(c)
	songID := c.Param("songId")

	if err := h.favSvc.Add(c.Request.Context(), userID, songID); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "add_favorite_error", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveFavorite handles DELETE /v1/favorites/:songId.
func (h *FavoriteHandler) RemoveFavorite(c *gin.Context) {
	userID := middleware.GetUserID(c)
	songID := c.Param("songId")

	if err := h.favSvc.Remove(c.Request.Context(), userID, songID); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "remove_favorite_error", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// RegisterRoutes registers favorite routes on the given router group.
func (h *FavoriteHandler) RegisterRoutes(rg *gin.RouterGroup) {
	favorites := rg.Group("/favorites")
	{
		favorites.GET("", h.ListFavorites)
		favorites.PUT("/:songId", h.AddFavorite)
		favorites.DELETE("/:songId", h.RemoveFavorite)
	}
}
