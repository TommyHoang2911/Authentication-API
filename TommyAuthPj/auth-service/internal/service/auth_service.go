package service

import (
	"auth-service/internal/model"
	"auth-service/internal/repository"
	"auth-service/internal/service/websocket"
)

// AuthService coordinates business logic related to authentication and
// user management. It delegates to specialized services for different operations.
type AuthService struct {
	userService  *UserService
	tokenService *TokenService
	qrService    *QRService
}

const jwtSecret = "your-secret-key" // TODO: move to environment variable

// NewAuthService constructs an AuthService with the provided repositories.
func NewAuthService(repo *repository.UserRepository, authCodeRepo *repository.AuthCodeRepository, emailService *EmailService, hub *websocket.Hub) *AuthService {
	userService := NewUserService(repo, emailService)
	tokenService := NewTokenService(userService, jwtSecret)
	qrService := NewQRService(authCodeRepo, userService, tokenService, hub)

	return &AuthService{
		userService:  userService,
		tokenService: tokenService,
		qrService:    qrService,
	}
}

// Register creates a new user record and hashes the password before storing.
func (s *AuthService) Register(email string, password string) (*model.User, error) {
	return s.userService.Register(email, password)
}

// Login authenticates a user and returns user data with tokens.
func (s *AuthService) Login(email string, password string) (*model.User, string, string, error) {
	user, err := s.userService.Login(email, password)
	if err != nil {
		return nil, "", "", err
	}

	accessToken, refreshToken, err := s.tokenService.GenerateTokens(user)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, refreshToken, nil
}

// GetUserByID retrieves a user by ID, omitting sensitive fields.
func (s *AuthService) GetUserByID(id int64) (*model.User, error) {
	return s.userService.GetUserByID(id)
}

// RefreshToken validates a refresh token and returns a new JWT token.
func (s *AuthService) RefreshToken(refreshToken string) (string, error) {
	return s.tokenService.RefreshToken(refreshToken)
}

// SignOut invalidates the user's refresh token.
func (s *AuthService) SignOut(refreshToken string) error {
	return s.tokenService.SignOut(refreshToken)
}

// GenerateQRCode creates a temporary auth code for QR sign-in (Device B).
func (s *AuthService) GenerateQRCode(deviceID string) (string, error) {
	return s.qrService.GenerateQRCode(deviceID)
}

// VerifyQRCode verifies the auth code scanned by Device A and sends a temporary code to Device B via WebSocket.
func (s *AuthService) VerifyQRCode(code string, userID int64) error {
	return s.qrService.VerifyQRCode(code, userID)
}

// ExchangeCode exchanges the temporary code for JWT tokens.
func (s *AuthService) ExchangeCode(tempCode string) (*model.User, string, string, error) {
	return s.qrService.ExchangeCode(tempCode)
}

// ConfirmEmail confirms a user's email using a confirmation token.
func (s *AuthService) ConfirmEmail(token string) error {
	return s.userService.ConfirmEmail(token)
}

// ResendConfirmationEmail resends the confirmation email to a user.
func (s *AuthService) ResendConfirmationEmail(email string) error {
	return s.userService.ResendConfirmationEmail(email)
}
