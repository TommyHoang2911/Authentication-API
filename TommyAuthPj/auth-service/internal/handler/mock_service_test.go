package handler

import (
	"auth-service/internal/model"

	"github.com/stretchr/testify/mock"
)

// MockAuthService is a mock implementation of the AuthService for testing
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(email, password string) (*model.User, error) {
	args := m.Called(email, password)
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockAuthService) Login(email, password string) (*model.User, string, string, error) {
	args := m.Called(email, password)
	return args.Get(0).(*model.User), args.String(1), args.String(2), args.Error(3)
}

func (m *MockAuthService) OAuthLoginURL(provider string, state string) (string, error) {
	args := m.Called(provider, state)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) OAuthCallback(provider, code string) (*model.User, string, string, error) {
	args := m.Called(provider, code)
	return args.Get(0).(*model.User), args.String(1), args.String(2), args.Error(3)
}

func (m *MockAuthService) GetUserByID(id int64) (*model.User, error) {
	args := m.Called(id)
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockAuthService) RefreshToken(refreshToken string) (string, error) {
	args := m.Called(refreshToken)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) SignOut(refreshToken string) error {
	args := m.Called(refreshToken)
	return args.Error(0)
}

func (m *MockAuthService) GenerateQRCode(deviceID string) (string, error) {
	args := m.Called(deviceID)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) VerifyQRCode(code string, userID int64) error {
	args := m.Called(code, userID)
	return args.Error(0)
}

func (m *MockAuthService) ExchangeCode(tempCode string) (*model.User, string, string, error) {
	args := m.Called(tempCode)
	return args.Get(0).(*model.User), args.String(1), args.String(2), args.Error(3)
}

func (m *MockAuthService) ConfirmEmail(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockAuthService) ResendConfirmationEmail(email string) error {
	args := m.Called(email)
	return args.Error(0)
}
