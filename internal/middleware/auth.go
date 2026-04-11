package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/service"
)

// Auth returns a middleware that validates JWT tokens and sets userId in context.
func Auth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{
				Error: model.APIError{Code: "unauthorized", Message: "missing authorization header"},
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{
				Error: model.APIError{Code: "unauthorized", Message: "invalid authorization format"},
			})
			return
		}

		claims, err := authSvc.ValidateJWT(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{
				Error: model.APIError{Code: "token_expired", Message: "invalid or expired token"},
			})
			return
		}

		userID, ok := claims["userId"].(string)
		if !ok || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{
				Error: model.APIError{Code: "invalid_token", Message: "token missing userId claim"},
			})
			return
		}

		c.Set("userId", userID)
		if email, ok := claims["email"].(string); ok {
			c.Set("email", email)
		}

		c.Next()
	}
}

// GetUserID extracts the userId from the Gin context (set by Auth middleware).
func GetUserID(c *gin.Context) string {
	userID, _ := c.Get("userId")
	if id, ok := userID.(string); ok {
		return id
	}
	return ""
}
