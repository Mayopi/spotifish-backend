package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Logger returns a middleware that logs requests using zerolog.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		event := log.Info()
		if status >= 400 {
			event = log.Warn()
		}
		if status >= 500 {
			event = log.Error()
		}

		event.
			Int("status", status).
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Int("bodySize", c.Writer.Size()).
			Msg("request")
	}
}
