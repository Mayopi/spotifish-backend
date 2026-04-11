package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/service"
)

// DriveHandler handles Drive connection endpoints.
type DriveHandler struct {
	driveSvc *service.DriveService
}

// NewDriveHandler creates a new DriveHandler.
func NewDriveHandler(driveSvc *service.DriveService) *DriveHandler {
	return &DriveHandler{driveSvc: driveSvc}
}

type connectRequest struct {
	AuthCode string `json:"authCode" binding:"required"`
}

type setFolderRequest struct {
	FolderID   string `json:"folderId" binding:"required"`
	FolderName string `json:"folderName" binding:"required"`
}

// Connect handles POST /v1/drive/connect.
func (h *DriveHandler) Connect(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req connectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: "authCode is required"},
		})
		return
	}

	if err := h.driveSvc.Connect(c.Request.Context(), userID, req.AuthCode); err != nil {
		log.Error().Err(err).Str("userId", userID).Msg("drive connect failed")
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "drive_connect_failed", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListFolders handles GET /v1/drive/folders.
func (h *DriveHandler) ListFolders(c *gin.Context) {
	userID := middleware.GetUserID(c)
	parentID := c.DefaultQuery("parentId", "root")

	folders, err := h.driveSvc.ListFolders(c.Request.Context(), userID, parentID)
	if err != nil {
		log.Error().Err(err).Str("userId", userID).Str("parentId", parentID).Msg("drive list folders failed")
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "drive_unauthorized", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"folders": folders})
}

// SetConnection handles POST /v1/drive/connection.
func (h *DriveHandler) SetConnection(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req setFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: "folderId and folderName are required"},
		})
		return
	}

	if err := h.driveSvc.SetActiveFolder(c.Request.Context(), userID, req.FolderID, req.FolderName); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "set_folder_failed", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteConnection handles DELETE /v1/drive/connection.
func (h *DriveHandler) DeleteConnection(c *gin.Context) {
	userID := middleware.GetUserID(c)
	deleteLibrary := c.DefaultQuery("deleteLibrary", "false") == "true"

	if err := h.driveSvc.Disconnect(c.Request.Context(), userID, deleteLibrary); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "disconnect_failed", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// RegisterRoutes registers drive routes on the given router group.
func (h *DriveHandler) RegisterRoutes(rg *gin.RouterGroup) {
	drive := rg.Group("/drive")
	{
		drive.POST("/connect", h.Connect)
		drive.GET("/folders", h.ListFolders)
		drive.POST("/connection", h.SetConnection)
		drive.DELETE("/connection", h.DeleteConnection)
	}
}
