package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/spotifish/backend/internal/model"
)

// Recovery returns a middleware that recovers from panics.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).
					Str("path", c.Request.URL.Path).
					Msg("recovered from panic")

				c.AbortWithStatusJSON(http.StatusInternalServerError, model.ErrorResponse{
					Error: model.APIError{Code: "internal_error", Message: "an internal error occurred"},
				})
			}
		}()
		c.Next()
	}
}
