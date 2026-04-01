package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"auth-service/internal/domain"
	appjwt "auth-service/pkg/jwt"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func testTokenManager() *appjwt.HMACManager {
	return appjwt.NewHMACManager("test-secret")
}

// Test that JWTAuthMiddleware returns a valid handler function
func TestJWTAuthMiddleware_ReturnsHandler(t *testing.T) {
	middleware := JWTAuthMiddleware(testTokenManager())
	assert.NotNil(t, middleware)
}

func TestJWTAuthMiddleware_MissingAuthHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuthMiddleware(testTokenManager()))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_InvalidTokenFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuthMiddleware(testTokenManager()))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat token")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuthMiddleware(testTokenManager()))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_EmptyBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuthMiddleware(testTokenManager()))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tm := testTokenManager()
	router := gin.New()
	router.Use(JWTAuthMiddleware(tm))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	user := &domain.User{ID: 42, Email: "test@example.com"}
	token, err := tm.GenerateToken(user, time.Hour)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTAuthMiddleware_MissingIDClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tm := testTokenManager()
	router := gin.New()
	router.Use(JWTAuthMiddleware(tm))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Build a valid JWT that deliberately omits the "id" claim.
	claims := gojwt.MapClaims{
		"email": "test@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	raw := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	token, err := raw.SignedString([]byte("test-secret"))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
