package service

import (
	"auth-service/internal/model"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenService handles JWT token generation and validation.
type TokenService struct {
	userService *UserService
	jwtSecret   string
}

// NewTokenService creates a new TokenService instance.
func NewTokenService(userService *UserService, jwtSecret string) *TokenService {
	return &TokenService{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

// GenerateTokens generates both JWT access token and refresh token for a user.
func (s *TokenService) GenerateTokens(user *model.User) (string, string, error) {
	// Generate JWT token
	accessToken, err := s.generateJWTToken(user)
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
	return s.generateJWTToken(user)
}

// SignOut invalidates the user's refresh token.
func (s *TokenService) SignOut(refreshToken string) error {
	return s.userService.DeleteRefreshToken(refreshToken)
}

// generateJWTToken creates a JWT token for the user.
func (s *TokenService) generateJWTToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"email": user.Email,
		"id":    user.ID,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// generateRefreshToken creates a cryptographically secure random refresh token
func generateRefreshToken() string {
	bytes := make([]byte, 32) // 32 bytes = 64 hex characters
	if _, err := rand.Read(bytes); err != nil {
		panic(err) // This should never happen in practice
	}
	return hex.EncodeToString(bytes)
}
