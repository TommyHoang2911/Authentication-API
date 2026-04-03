package repository

import (
	"time"

	"auth-service/internal/domain"
)

// UserStore defines persistence operations required by user-facing services.
type UserStore interface {
	Create(user *domain.User) error
	FindByEmail(email string) (*domain.User, error)
	EmailExists(email string) (bool, error)
	FindByID(id int64) (*domain.User, error)
	FindByOAuth(provider, providerID string) (*domain.User, error)
	LinkOAuthProvider(userID int64, provider, providerID string) error
	UpdateRefreshToken(userID int64, refreshToken string, expiry time.Time) error
	CreateRefreshToken(userID int64, token string, expiresAt time.Time) error
	FindRefreshToken(token string) (int64, error)
	DeleteRefreshToken(token string) error
	DeleteRefreshTokensByUserID(userID int64) error
	FindRefreshTokenByUserID(userID int64) (string, error)
	FindByConfirmationToken(token string) (*domain.User, error)
	ConfirmEmail(userID int64) error
	UpdateConfirmationToken(userID int64, token string, expiry time.Time) error
}

// AuthCodeStore defines persistence operations for QR and temporary auth codes.
type AuthCodeStore interface {
	Create(authCode *domain.AuthCode) error
	FindByCode(code string) (*domain.AuthCode, error)
	MarkAsUsed(code string) error
	DeleteExpired() error
}
