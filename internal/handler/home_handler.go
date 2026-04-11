package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/service"
)

// HomeHandler handles the home endpoint.
type HomeHandler struct {
	libSvc *service.LibraryService
}

// NewHomeHandler creates a new HomeHandler.
func NewHomeHandler(libSvc *service.LibraryService) *HomeHandler {
	return &HomeHandler{libSvc: libSvc}
}

// GetHome handles GET /v1/home.
func (h *HomeHandler) GetHome(c *gin.Context) {
	userID := middleware.GetUserID(c)

	home, err := h.libSvc.GetHomeSections(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "home_error", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, home)
}

// RegisterRoutes registers home routes on the given router group.
func (h *HomeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/home", h.GetHome)
}
