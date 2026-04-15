package handler

import "auth-service/internal/domain"

// AuthServiceInterface defines the methods needed by the handler
type AuthServiceInterface interface {
	Register(email, password string) (*domain.User, error)
	Login(email, password string) (*domain.User, string, string, error)
	OAuthLoginURL(provider string, state string) (string, error)
	OAuthCallback(provider, code string) (*domain.User, string, string, error)
	GetUserByID(id int64) (*domain.User, error)
	RefreshToken(refreshToken string) (string, error)
	SignOut(refreshToken string) error
	GenerateQRCode(deviceID string) (string, error)
	VerifyQRCode(code string, userID int64) error
	ExchangeCode(tempCode string) (*domain.User, string, string, error)
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

type ErrorResponse struct {
	Error string `json:"error" example:"invalid request"`
}

type MessageResponse struct {
	Message string `json:"message" example:"operation completed successfully"`
}

type TokenResponse struct {
	Token string `json:"token" example:"jwt-token"`
}

type QRCodeResponse struct {
	Code string `json:"code" example:"generated-qr-code"`
}

type UserEnvelope struct {
	User *domain.User `json:"user"`
}

type RegisterResponse struct {
	Message string       `json:"message" example:"user registered successfully"`
	User    *domain.User `json:"user"`
}

type LoginResponse struct {
	Message      string       `json:"message" example:"login successful"`
	User         *domain.User `json:"user"`
	Token        string       `json:"token" example:"jwt-token"`
	RefreshToken string       `json:"refresh_token" example:"refresh-token"`
}

type OAuthLoginResponse struct {
	Message      string       `json:"message" example:"oauth login successful"`
	User         *domain.User `json:"user"`
	Token        string       `json:"token" example:"jwt-token"`
	RefreshToken string       `json:"refresh_token" example:"refresh-token"`
}

type ExchangeCodeResponse struct {
	Message      string       `json:"message" example:"code exchanged successfully"`
	User         *domain.User `json:"user"`
	Token        string       `json:"token" example:"jwt-token"`
	SessionToken string       `json:"session_token" example:"refresh-token"`
}
