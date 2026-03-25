package handler

import (
	"auth-service/internal/model"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthHandler_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful registration",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				user := &model.User{ID: 1, Email: "test@example.com"}
				m.On("Register", "test@example.com", "password123").Return(user, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"message": "user registered successfully",
				"user": map[string]interface{}{
					"id":    float64(1),
					"email": "test@example.com",
				},
			},
		},
		{
			name: "invalid JSON",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
			},
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "validation error",
			},
		},
		{
			name: "service error",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Register", "test@example.com", "password123").Return((*model.User)(nil), errors.New("user already exists"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "user already exists",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			handler.Register(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectedStatus == http.StatusCreated {
				assert.Equal(t, tt.expectedBody["message"], response["message"])
				user := response["user"].(map[string]interface{})
				assert.Equal(t, tt.expectedBody["user"].(map[string]interface{})["email"], user["email"])
			} else {
				assert.NotEmpty(t, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful login",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				user := &model.User{ID: 1, Email: "test@example.com"}
				m.On("Login", "test@example.com", "password123").Return(user, "jwt-token", "refresh-token", nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "login successful",
				"user": map[string]interface{}{
					"id":    float64(1),
					"email": "test@example.com",
				},
				"token":         "jwt-token",
				"refresh_token": "refresh-token",
			},
		},
		{
			name: "invalid JSON",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
			},
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "authentication failed",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Login", "test@example.com", "wrongpassword").Return((*model.User)(nil), "", "", errors.New("invalid credentials"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "invalid credentials",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			handler.Login(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Equal(t, tt.expectedBody["message"], response["message"])
				if tt.expectedStatus == http.StatusOK {
					assert.Equal(t, tt.expectedBody["token"], response["token"])
					assert.Equal(t, tt.expectedBody["refresh_token"], response["refresh_token"])
				} else {
					assert.Equal(t, tt.expectedBody["error"], response["error"])
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_GetUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userID         interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:   "successful get user",
			userID: int64(1),
			mockSetup: func(m *MockAuthService) {
				user := &model.User{ID: 1, Email: "test@example.com"}
				m.On("GetUserByID", int64(1)).Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"user": map[string]interface{}{
					"id":    float64(1),
					"email": "test@example.com",
				},
			},
		},
		{
			name:           "unauthenticated",
			userID:         nil,
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "user not authenticated",
			},
		},
		{
			name:   "service error",
			userID: int64(1),
			mockSetup: func(m *MockAuthService) {
				m.On("GetUserByID", int64(1)).Return((*model.User)(nil), errors.New("user not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "failed to get user",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.userID != nil {
				c.Set("user_id", tt.userID)
			}

			req := httptest.NewRequest(http.MethodGet, "/user", nil)
			c.Request = req

			handler.GetUser(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			assert.Equal(t, tt.expectedBody["error"], response["error"])

			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful refresh",
			requestBody: RefreshTokenRequest{
				RefreshToken: "valid-refresh-token",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("RefreshToken", "valid-refresh-token").Return("new-jwt-token", nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"token": "new-jwt-token",
			},
		},
		{
			name: "invalid JSON",
			requestBody: map[string]interface{}{
				"refresh_token": "",
			},
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "refresh failed",
			requestBody: RefreshTokenRequest{
				RefreshToken: "invalid-refresh-token",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("RefreshToken", "invalid-refresh-token").Return("", errors.New("invalid refresh token"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "invalid refresh token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			handler.RefreshToken(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Equal(t, tt.expectedBody["token"], response["token"])
				assert.Equal(t, tt.expectedBody["error"], response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_SignOut(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
	}{
		{
			name: "successful sign out",
			requestBody: SignOutRequest{
				RefreshToken: "valid-refresh-token",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("SignOut", "valid-refresh-token").Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid JSON",
			requestBody: map[string]interface{}{
				"refresh_token": "",
			},
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			requestBody: SignOutRequest{
				RefreshToken: "invalid-token",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("SignOut", "invalid-token").Return(errors.New("token not found"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/signout", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			handler.SignOut(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}
