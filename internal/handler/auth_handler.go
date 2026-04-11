package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spotifish/backend/internal/middleware"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authSvc *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

type signInRequest struct {
	IDToken string `json:"idToken" binding:"required"`
}

type signInResponse struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	User         *model.User `json:"user"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type signOutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// SignInWithGoogle handles POST /v1/auth/google.
func (h *AuthHandler) SignInWithGoogle(c *gin.Context) {
	var req signInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: "idToken is required"},
		})
		return
	}

	pair, user, err := h.authSvc.SignInWithGoogle(c.Request.Context(), req.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error: model.APIError{Code: "auth_failed", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, signInResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		User:         user,
	})
}

// RefreshTokens handles POST /v1/auth/refresh.
func (h *AuthHandler) RefreshTokens(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: "refreshToken is required"},
		})
		return
	}

	pair, err := h.authSvc.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_refresh_token", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, pair)
}

// SignOut handles POST /v1/auth/sign-out.
func (h *AuthHandler) SignOut(c *gin.Context) {
	var req signOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: "refreshToken is required"},
		})
		return
	}

	if err := h.authSvc.SignOut(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "sign_out_failed", Message: err.Error()},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// RegisterRoutes registers auth routes on the given router group.
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/google", h.SignInWithGoogle)
		auth.POST("/refresh", h.RefreshTokens)
		auth.POST("/sign-out", h.SignOut)
	}
}

// UserHandler handles user profile and settings endpoints.
type UserHandler struct {
	userRepo    userGetter
	settingsSvc *service.SettingsService
}

type userGetter interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userRepo userGetter, settingsSvc *service.SettingsService) *UserHandler {
	return &UserHandler{userRepo: userRepo, settingsSvc: settingsSvc}
}

// GetMe handles GET /v1/me.
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: model.APIError{Code: "user_not_found", Message: "user not found"},
		})
		return
	}
	c.JSON(http.StatusOK, user)
}

// GetSettings handles GET /v1/me/settings.
func (h *UserHandler) GetSettings(c *gin.Context) {
	userID := middleware.GetUserID(c)
	settings, err := h.settingsSvc.GetSettings(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "settings_error", Message: err.Error()},
		})
		return
	}
	c.JSON(http.StatusOK, settings)
}

// UpdateSettings handles PATCH /v1/me/settings.
func (h *UserHandler) UpdateSettings(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var patch model.UserSettingsPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.APIError{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	settings, err := h.settingsSvc.UpdateSettings(c.Request.Context(), userID, &patch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.APIError{Code: "settings_error", Message: err.Error()},
		})
		return
	}
	c.JSON(http.StatusOK, settings)
}

// RegisterRoutes registers user routes on the given router group.
func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/me", h.GetMe)
	me := rg.Group("/me")
	{
		me.GET("/settings", h.GetSettings)
		me.PATCH("/settings", h.UpdateSettings)
	}
}
