package middleware

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const jwtSecret = "your-secret-key" // TODO: move to environment variable

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Printf("JWTAuthMiddleware: missing Authorization header")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			log.Printf("JWTAuthMiddleware: bearer token missing in Authorization header")
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			log.Printf("JWTAuthMiddleware: token parse/validation error: %v", err)
			return
		}

		// Extract claims from token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Printf("JWTAuthMiddleware: invalid token claims")
			return
		}

		// Set user_id from claims in context
		if id, exists := claims["id"]; exists {
			c.Set("user_id", int64(id.(float64)))
		}

		c.Next()
	}
}
