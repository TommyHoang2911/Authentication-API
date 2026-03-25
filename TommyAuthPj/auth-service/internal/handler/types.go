package handler

import "auth-service/internal/model"

// AuthServiceInterface defines the methods needed by the handler
type AuthServiceInterface interface {
	Register(email, password string) (*model.User, error)
	Login(email, password string) (*model.User, string, string, error)
	OAuthLoginURL(provider string, state string) (string, error)
	OAuthCallback(provider, code string) (*model.User, string, string, error)
	GetUserByID(id int64) (*model.User, error)
	RefreshToken(refreshToken string) (string, error)
	SignOut(refreshToken string) error
	GenerateQRCode(deviceID string) (string, error)
	VerifyQRCode(code string, userID int64) error
	ExchangeCode(tempCode string) (*model.User, string, string, error)
	ConfirmEmail(token string) error
	ResendConfirmationEmail(email string) error
}

// User Request/Response Types

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type SignOutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// QR Code Request/Response Types

type GenerateQRRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

type VerifyQRRequest struct {
	Code string `json:"code" binding:"required"`
}

type ExchangeCodeRequest struct {
	TempCode string `json:"temp_code" binding:"required"`
}

// Email Confirmation Request Types

type ConfirmEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type ResendConfirmationRequest struct {
	Email string `json:"email" binding:"required,email"`
}
