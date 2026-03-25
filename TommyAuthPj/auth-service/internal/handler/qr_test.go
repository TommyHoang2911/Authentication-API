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

func TestAuthHandler_GenerateQR(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful generate QR",
			requestBody: GenerateQRRequest{
				DeviceID: "device-123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("GenerateQRCode", "device-123").Return("test-qr-code-123", nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"code": "test-qr-code-123",
			},
		},
		{
			name: "service error",
			requestBody: GenerateQRRequest{
				DeviceID: "device-123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("GenerateQRCode", "device-123").Return("", errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "invalid request - missing device_id",
			requestBody:    map[string]interface{}{},
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			var reqBody []byte
			if tt.requestBody != nil {
				reqBody, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/generate_qr", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			handler.GenerateQR(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedBody["code"], response["code"])
			} else if tt.expectedStatus == http.StatusInternalServerError {
				assert.NotEmpty(t, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_VerifyQR(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		userID         interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful verify QR",
			requestBody: VerifyQRRequest{
				Code: "valid-code",
			},
			userID: int64(1),
			mockSetup: func(m *MockAuthService) {
				m.On("VerifyQRCode", "valid-code", int64(1)).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "QR code verified successfully",
			},
		},
		{
			name:           "invalid JSON",
			requestBody:    map[string]interface{}{},
			userID:         int64(1),
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "unauthenticated",
			requestBody: VerifyQRRequest{
				Code: "valid-code",
			},
			userID:         nil,
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid code",
			requestBody: VerifyQRRequest{
				Code: "expired-code",
			},
			userID: int64(1),
			mockSetup: func(m *MockAuthService) {
				m.On("VerifyQRCode", "expired-code", int64(1)).Return(errors.New("invalid or expired code"))
			},
			expectedStatus: http.StatusUnauthorized,
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

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/verify_qr", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			handler.VerifyQR(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedBody["message"], response["message"])
			} else {
				assert.NotEmpty(t, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_ExchangeCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful exchange code",
			requestBody: ExchangeCodeRequest{
				TempCode: "valid-temp-code",
			},
			mockSetup: func(m *MockAuthService) {
				user := &model.User{ID: 1, Email: "test@example.com"}
				m.On("ExchangeCode", "valid-temp-code").Return(user, "new-jwt-token", "new-session-token", nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "code exchanged successfully",
				"user": map[string]interface{}{
					"id":    float64(1),
					"email": "test@example.com",
				},
				"token":         "new-jwt-token",
				"session_token": "new-session-token",
			},
		},
		{
			name:           "invalid JSON",
			requestBody:    map[string]interface{}{},
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid temp code",
			requestBody: ExchangeCodeRequest{
				TempCode: "invalid-temp-code",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("ExchangeCode", "invalid-temp-code").Return((*model.User)(nil), "", "", errors.New("invalid or expired temp code"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "invalid or expired temp code",
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
			req := httptest.NewRequest(http.MethodPost, "/exchange_code", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			handler.ExchangeCode(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedBody["message"], response["message"])
				assert.Equal(t, tt.expectedBody["token"], response["token"])
				assert.Equal(t, tt.expectedBody["session_token"], response["session_token"])
			} else {
				assert.NotEmpty(t, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}
