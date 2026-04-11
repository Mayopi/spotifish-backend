package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"github.com/spotifish/backend/internal/model"
	"github.com/spotifish/backend/internal/repository"
	"google.golang.org/api/idtoken"
)

const (
	accessTokenDuration  = 15 * time.Minute
	refreshTokenDuration = 30 * 24 * time.Hour // 30 days
	refreshTokenBytes    = 32
)

// AuthService handles authentication logic.
type AuthService struct {
	userRepo     *repository.UserRepository
	authRepo     *repository.AuthRepository
	jwtKey       []byte
	googleClient string
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo *repository.UserRepository,
	authRepo *repository.AuthRepository,
	jwtSigningKey string,
	googleClientID string,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		authRepo:     authRepo,
		jwtKey:       []byte(jwtSigningKey),
		googleClient: googleClientID,
	}
}

// SignInWithGoogle verifies a Google ID token and returns a JWT pair + user.
func (s *AuthService) SignInWithGoogle(ctx context.Context, googleIDToken string) (*model.TokenPair, *model.User, error) {
	// Verify the Google ID token
	payload, err := idtoken.Validate(ctx, googleIDToken, s.googleClient)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid google id token: %w", err)
	}

	googleSub := payload.Subject
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)

	// Find or create user
	user, err := s.userRepo.FindByGoogleSub(ctx, googleSub)
	if err != nil {
		return nil, nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		user = &model.User{
			GoogleSub:   googleSub,
			Email:       email,
			DisplayName: name,
		}
		user, err = s.userRepo.Create(ctx, user)
		if err != nil {
			return nil, nil, fmt.Errorf("create user: %w", err)
		}
		log.Info().Str("userId", user.ID).Str("email", email).Msg("new user created")
	}

	// Issue token pair
	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return pair, user, nil
}

// RefreshTokens rotates a refresh token and returns a new JWT pair.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	tokenHash := hashToken(refreshToken)

	// Find the stored token
	stored, err := s.authRepo.FindRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("find refresh token: %w", err)
	}
	if stored == nil {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	// Revoke the old token
	if err := s.authRepo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	// Get the user
	user, err := s.userRepo.GetByID(ctx, stored.UserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Issue a new pair
	return s.issueTokenPair(ctx, user)
}

// SignOut revokes a refresh token.
func (s *AuthService) SignOut(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	return s.authRepo.RevokeRefreshToken(ctx, tokenHash)
}

// ValidateJWT validates a JWT and returns its claims.
func (s *AuthService) ValidateJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// issueTokenPair creates a new access token (JWT) and refresh token.
func (s *AuthService) issueTokenPair(ctx context.Context, user *model.User) (*model.TokenPair, error) {
	// Create JWT
	now := time.Now()
	claims := jwt.MapClaims{
		"userId": user.ID,
		"email":  user.Email,
		"iat":    now.Unix(),
		"exp":    now.Add(accessTokenDuration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(s.jwtKey)
	if err != nil {
		return nil, fmt.Errorf("sign jwt: %w", err)
	}

	// Create refresh token
	refreshBytes := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshToken := hex.EncodeToString(refreshBytes)
	refreshHash := hashToken(refreshToken)

	expiresAt := now.Add(refreshTokenDuration)
	if err := s.authRepo.CreateRefreshToken(ctx, user.ID, refreshHash, expiresAt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// hashToken returns the SHA-256 hash of a token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
