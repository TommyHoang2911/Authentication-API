package jwt

import (
	"errors"
	"fmt"
	"time"

	"auth-service/internal/domain"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// Manager encapsulates JWT generation and validation.
type Manager interface {
	GenerateToken(user *domain.User, ttl time.Duration) (string, error)
	ParseToken(tokenString string) (gojwt.MapClaims, error)
}

// HMACManager issues and validates HMAC-signed JWTs.
type HMACManager struct {
	secret []byte
}

func NewHMACManager(secret string) *HMACManager {
	return &HMACManager{secret: []byte(secret)}
}

func (m *HMACManager) GenerateToken(user *domain.User, ttl time.Duration) (string, error) {
	if len(m.secret) == 0 {
		return "", errors.New("jwt secret is not configured")
	}

	claims := gojwt.MapClaims{
		"email": user.Email,
		"id":    user.ID,
		"exp":   time.Now().Add(ttl).Unix(),
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *HMACManager) ParseToken(tokenString string) (gojwt.MapClaims, error) {
	if len(m.secret) == 0 {
		return nil, errors.New("jwt secret is not configured")
	}

	parser := gojwt.NewParser(gojwt.WithValidMethods([]string{gojwt.SigningMethodHS256.Alg()}))
	token, err := parser.Parse(tokenString, func(token *gojwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != gojwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(gojwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
