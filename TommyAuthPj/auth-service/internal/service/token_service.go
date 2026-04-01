package service

import (
	"auth-service/internal/domain"
	appjwt "auth-service/pkg/jwt"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

type tokenUserService interface {
	CreateRefreshToken(userID int64, refreshToken string, expiry time.Time) error
	FindRefreshToken(refreshToken string) (int64, error)
	DeleteRefreshToken(refreshToken string) error
	GetUserByID(id int64) (*domain.User, error)
}

// TokenService handles JWT token generation and validation.
type TokenService struct {
	userService  tokenUserService
	tokenManager appjwt.Manager
}

// NewTokenService creates a new TokenService instance.
func NewTokenService(userService tokenUserService, tokenManager appjwt.Manager) *TokenService {
	return &TokenService{
		userService:  userService,
		tokenManager: tokenManager,
	}
}

// GenerateTokens generates both JWT access token and refresh token for a user.
func (s *TokenService) GenerateTokens(user *domain.User) (string, string, error) {
	// Generate JWT token
	accessToken, err := s.tokenManager.GenerateToken(user, 24*time.Hour)
	if err != nil {
		return "", "", err
	}

	// Generate refresh token
	refreshToken := generateRefreshToken()
	refreshTokenExpiry := time.Now().Add(time.Hour * 24 * 7) // 7 days

	// Create refresh token in database
	if err := s.userService.CreateRefreshToken(user.ID, refreshToken, refreshTokenExpiry); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// RefreshToken validates a refresh token and returns a new JWT token.
func (s *TokenService) RefreshToken(refreshToken string) (string, error) {
	// Find user associated with the refresh token
	userID, err := s.userService.FindRefreshToken(refreshToken)
	if err != nil {
		return "", errors.New("invalid or expired refresh token")
	}

	// Get user details
	user, err := s.userService.GetUserByID(userID)
	if err != nil {
		return "", errors.New("user not found")
	}

	// Generate new JWT token
	return s.tokenManager.GenerateToken(user, 24*time.Hour)
}

// SignOut invalidates the user's refresh token.
func (s *TokenService) SignOut(refreshToken string) error {
	return s.userService.DeleteRefreshToken(refreshToken)
}

// generateRefreshToken creates a cryptographically secure random refresh token
func generateRefreshToken() string {
	bytes := make([]byte, 32) // 32 bytes = 64 hex characters
	if _, err := rand.Read(bytes); err != nil {
		panic(err) // This should never happen in practice
	}
	return hex.EncodeToString(bytes)
}
