package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/service"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// StreamHandler handles the song streaming endpoint.
type StreamHandler struct {
	libSvc   *service.LibraryService
	driveSvc *service.DriveService
}

// NewStreamHandler creates a new StreamHandler.
func NewStreamHandler(libSvc *service.LibraryService, driveSvc *service.DriveService) *StreamHandler {
	return &StreamHandler{libSvc: libSvc, driveSvc: driveSvc}
}

// StreamSong handles GET /v1/songs/:id/stream.
// This is a byte-proxy endpoint that pipes Drive file bytes to the client.
func (h *StreamHandler) StreamSong(c *gin.Context) {
	userID := middleware.GetUserID(c)
	songID := c.Param("id")

	// Get the song to find its source file ID
	song, err := h.libSvc.GetSong(c.Request.Context(), userID, songID)
	if err != nil || song == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: model.APIError{Code: "not_found", Message: "song not found"},
		})
		return
	}

	if song.Source != "drive" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "not_streamable", Message: "only drive songs can be streamed"},
		})
		return
	}

	// Get an authenticated Drive client
	client, err := h.driveSvc.GetClient(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "drive_error", Message: "failed to connect to drive"},
		})
		return
	}

	// Create Drive service
	srv, err := drive.NewService(c.Request.Context(), option.WithHTTPClient(client))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "drive_error", Message: "failed to create drive service"},
		})
		return
	}

	// Get file metadata for content length
	file, err := srv.Files.Get(song.SourceFileID).Fields("size, mimeType").Do()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "drive_error", Message: "failed to get file info"},
		})
		return
	}

	fileSize := file.Size
	mimeType := file.MimeType
	if song.MimeType != "" {
		mimeType = song.MimeType
	}

	// Handle Range requests for seeking support
	rangeHeader := c.GetHeader("Range")

	if rangeHeader != "" {
		// Parse Range header (e.g., "bytes=0-1023")
		var start, end int64
		_, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
		if err != nil {
			// Try parsing "bytes=0-" format
			fmt.Sscanf(rangeHeader, "bytes=%d-", &start)
			end = fileSize - 1
		}
		if end == 0 || end >= fileSize {
			end = fileSize - 1
		}

		// Create a request with Range header
		req := srv.Files.Get(song.SourceFileID)
		req.Header().Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
		resp, err := req.Download()
		if err != nil {
			log.Error().Err(err).Str("songId", songID).Msg("failed to download range")
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{
				Error: model.APIError{Code: "stream_error", Message: "failed to stream song"},
			})
			return
		}
		defer resp.Body.Close()

		contentLength := end - start + 1
		c.Header("Content-Type", mimeType)
		c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		c.Header("Accept-Ranges", "bytes")
		c.Status(http.StatusPartialContent)
		io.Copy(c.Writer, resp.Body)
		return
	}

	// Full file download
	resp, err := srv.Files.Get(song.SourceFileID).Download()
	if err != nil {
		log.Error().Err(err).Str("songId", songID).Msg("failed to download file")
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "stream_error", Message: "failed to stream song"},
		})
		return
	}
	defer resp.Body.Close()

	c.Header("Content-Type", mimeType)
	c.Header("Content-Length", strconv.FormatInt(fileSize, 10))
	c.Header("Accept-Ranges", "bytes")
	c.Status(http.StatusOK)
	io.Copy(c.Writer, resp.Body)
}

// RegisterRoutes registers stream routes on the given router group.
func (h *StreamHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/songs/:id/stream", h.StreamSong)
}
