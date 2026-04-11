package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spotifish/backend/internal/model"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	pool    *pgxpool.Pool
	artPath string
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(pool *pgxpool.Pool, artPath string) *HealthHandler {
	return &HealthHandler{pool: pool, artPath: artPath}
}

// Healthz handles GET /healthz (liveness probe).
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz handles GET /readyz (readiness probe — verifies DB connectivity).
func (h *HealthHandler) Readyz(c *gin.Context) {
	if err := h.pool.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "database unreachable",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// ServeArt handles GET /v1/art/:key — serves album art files from disk.
func (h *HealthHandler) ServeArt(c *gin.Context) {
	key := c.Param("key")
	path := filepath.Join(h.artPath, key+".img")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: model.APIError{Code: "not_found", Message: "art not found"},
		})
		return
	}

	c.File(path)
}

// RegisterRoutes registers health and art routes.
func (h *HealthHandler) RegisterRoutes(engine *gin.Engine, rg *gin.RouterGroup) {
	engine.GET("/healthz", h.Healthz)
	engine.GET("/readyz", h.Readyz)
	rg.GET("/art/:key", h.ServeArt)
}
