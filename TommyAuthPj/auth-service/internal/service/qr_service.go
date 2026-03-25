package service

import (
	"auth-service/internal/model"
	"auth-service/internal/repository"
	"auth-service/internal/service/websocket"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// QRService handles QR code generation, verification, and exchange operations.
type QRService struct {
	authCodeRepo *repository.AuthCodeRepository
	userService  *UserService
	tokenService *TokenService
	hub          *websocket.Hub
}

// NewQRService creates a new QRService instance.
func NewQRService(authCodeRepo *repository.AuthCodeRepository, userService *UserService, tokenService *TokenService, hub *websocket.Hub) *QRService {
	return &QRService{
		authCodeRepo: authCodeRepo,
		userService:  userService,
		tokenService: tokenService,
		hub:          hub,
	}
}

// GenerateQRCode creates a temporary auth code for QR sign-in (Device B).
func (s *QRService) GenerateQRCode(deviceID string) (string, error) {
	code := generateHashedQRCode(deviceID)
	expiresAt := time.Now().Add(2 * time.Minute)

	authCode := &model.AuthCode{
		UserID:    nil, // No user yet; will be linked when Device A verifies
		Code:      code,
		DeviceID:  deviceID,
		ExpiresAt: expiresAt,
		Used:      false,
	}

	if err := s.authCodeRepo.Create(authCode); err != nil {
		return "", err
	}

	return code, nil
}

// VerifyQRCode verifies the auth code scanned by Device A and sends a temporary code to Device B via WebSocket.
func (s *QRService) VerifyQRCode(code string, userID int64) error {
	authCode, err := s.authCodeRepo.FindByCode(code)
	if err != nil {
		return errors.New("invalid or expired code")
	}

	// Mark the original code as used
	if err := s.authCodeRepo.MarkAsUsed(code); err != nil {
		return err
	}

	// Generate a temporary code linked to the user
	tempCode := generateRefreshToken()
	expiresAt := time.Now().Add(5 * time.Minute) // Short-lived temp code

	tempAuthCode := &model.AuthCode{
		UserID:    &userID,
		Code:      tempCode,
		ExpiresAt: expiresAt,
		Used:      false,
	}

	if err := s.authCodeRepo.Create(tempAuthCode); err != nil {
		return err
	}

	// Send success and temp code to Device B via WebSocket
	message := map[string]interface{}{
		"status":    "success",
		"temp_code": tempCode,
		"device_id": authCode.DeviceID,
		"message":   "QR code verified successfully",
	}
	s.hub.SendMessage(code, message)

	return nil
}

// ExchangeCode exchanges the temporary code for JWT tokens.
func (s *QRService) ExchangeCode(tempCode string) (*model.User, string, string, error) {
	authCode, err := s.authCodeRepo.FindByCode(tempCode)
	if err != nil {
		return nil, "", "", errors.New("invalid or expired temp code")
	}

	if authCode.Used {
		return nil, "", "", errors.New("temp code already used")
	}

	// Mark as used
	if err := s.authCodeRepo.MarkAsUsed(tempCode); err != nil {
		return nil, "", "", err
	}

	// Get user
	if authCode.UserID == nil {
		return nil, "", "", errors.New("temp code not linked to a user")
	}

	user, err := s.userService.GetUserByID(*authCode.UserID)
	if err != nil {
		return nil, "", "", err
	}

	// Generate tokens
	accessToken, refreshToken, err := s.tokenService.GenerateTokens(user)
	if err != nil {
		return nil, "", "", err
	}

	s.hub.UnregisterConnection(authCode.Code)

	return user, accessToken, refreshToken, nil
}

// generateHashedQRCode creates a SHA256 hash of device_id mixed with random salt
func generateHashedQRCode(deviceID string) string {
	// Generate random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		panic(err) // This should never happen in practice
	}

	// Combine device_id and salt
	data := append([]byte(deviceID), salt...)

	// Hash the combination
	hash := sha256.Sum256(data)

	// Return hex-encoded hash
	return hex.EncodeToString(hash[:])
}
