package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spotifish/backend/internal/model"
)

// rateLimiterEntry tracks request counts per user.
type rateLimiterEntry struct {
	count    int
	windowStart time.Time
}

// RateLimiter returns a middleware that limits requests per user per minute.
func RateLimiter(maxPerMin int) gin.HandlerFunc {
	var mu sync.Mutex
	entries := make(map[string]*rateLimiterEntry)
	window := time.Minute

	// Cleanup goroutine to prevent memory leaks
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for key, entry := range entries {
				if now.Sub(entry.windowStart) > 2*window {
					delete(entries, key)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		// Use userId if authenticated, otherwise use IP
		key := c.ClientIP()
		if userID, exists := c.Get("userId"); exists {
			if uid, ok := userID.(string); ok {
				key = "user:" + uid
			}
		}

		mu.Lock()
		now := time.Now()
		entry, exists := entries[key]
		if !exists || now.Sub(entry.windowStart) >= window {
			entries[key] = &rateLimiterEntry{count: 1, windowStart: now}
			mu.Unlock()
			c.Next()
			return
		}

		entry.count++
		if entry.count > maxPerMin {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, model.ErrorResponse{
				Error: model.APIError{
					Code:    "rate_limited",
					Message: "too many requests, please try again later",
				},
			})
			return
		}
		mu.Unlock()
		c.Next()
	}
}
