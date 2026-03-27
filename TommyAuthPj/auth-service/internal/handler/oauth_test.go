package handler

import (
	"auth-service/internal/model"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthHandler_OAuthLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		provider       string
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectRedirect bool
	}{
		{
			name:     "successful oauth login redirect",
			provider: "google",
			setupMock: func(m *MockAuthService) {
				m.On("OAuthLoginURL", "google", mock.AnythingOfType("string")).Return("https://accounts.google.com/o/oauth2/auth", nil)
			},
			expectedStatus: http.StatusTemporaryRedirect,
			expectRedirect: true,
		},
		{
			name:     "unsupported provider",
			provider: "twitter",
			setupMock: func(m *MockAuthService) {
				m.On("OAuthLoginURL", "twitter", mock.AnythingOfType("string")).Return("", errors.New("unsupported oauth provider"))
			},
			expectedStatus: http.StatusBadRequest,
			expectRedirect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{}
			tt.setupMock(mockService)

			handler := NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/auth/"+tt.provider+"/login", nil)
			c.Request = req
			c.Params = gin.Params{{Key: "provider", Value: tt.provider}}

			handler.OAuthLogin(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectRedirect {
				assert.NotEmpty(t, w.Header().Get("Set-Cookie"))
				assert.Contains(t, w.Header().Get("Set-Cookie"), "oauth_state=")
				assert.Equal(t, "https://accounts.google.com/o/oauth2/auth", w.Header().Get("Location"))
			} else {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_OAuthCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		provider       string
		query          string
		cookie         *http.Cookie
		setupMock      func(*MockAuthService)
		expectedStatus int
	}{
		{
			name:           "missing code or state",
			provider:       "google",
			query:          "state=abc",
			cookie:         nil,
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid oauth state",
			provider: "google",
			query:    "code=test-code&state=abc",
			cookie: &http.Cookie{
				Name:  "oauth_state",
				Value: "different",
			},
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "oauth callback success",
			provider: "google",
			query:    "code=test-code&state=abc",
			cookie: &http.Cookie{
				Name:  "oauth_state",
				Value: "abc",
			},
			setupMock: func(m *MockAuthService) {
				user := &model.User{ID: 1, Email: "oauth@example.com"}
				m.On("OAuthCallback", "google", "test-code").Return(user, "jwt-token", "refresh-token", nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "oauth callback service error",
			provider: "facebook",
			query:    "code=test-code&state=state123",
			cookie: &http.Cookie{
				Name:  "oauth_state",
				Value: "state123",
			},
			setupMock: func(m *MockAuthService) {
				m.On("OAuthCallback", "facebook", "test-code").Return((*model.User)(nil), "", "", errors.New("oauth authentication failed"))
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{}
			tt.setupMock(mockService)

			handler := NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req := httptest.NewRequest(http.MethodGet, "/auth/"+tt.provider+"/callback?"+tt.query, nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			c.Request = req
			c.Params = gin.Params{{Key: "provider", Value: tt.provider}}

			handler.OAuthCallback(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "oauth login successful", response["message"])
				assert.Equal(t, "jwt-token", response["token"])
				assert.Equal(t, "refresh-token", response["refresh_token"])
				user, ok := response["user"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "oauth@example.com", user["email"])
			} else {
				errVal, exists := response["error"]
				assert.True(t, exists)
				assert.NotEmpty(t, strings.TrimSpace(errVal.(string)))
			}

			mockService.AssertExpectations(t)
		})
	}
}
