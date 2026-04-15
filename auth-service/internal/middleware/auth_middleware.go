package middleware

import (
	"log"
	"net/http"
	"strings"

	appjwt "auth-service/pkg/jwt"

	"github.com/gin-gonic/gin"
)

func JWTAuthMiddleware(tokenManager appjwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Printf("JWTAuthMiddleware: missing Authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			log.Printf("JWTAuthMiddleware: bearer token missing in Authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header format"})
			return
		}

		claims, err := tokenManager.ParseToken(tokenString)
		if err != nil {
			log.Printf("JWTAuthMiddleware: token parse/validation error: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		id, exists := claims["id"]
		if !exists {
			log.Printf("JWTAuthMiddleware: missing id claim in token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}
		switch v := id.(type) {
		case float64:
			c.Set("user_id", int64(v))
		default:
			log.Printf("JWTAuthMiddleware: unexpected type for id claim: %T", id)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		c.Next()
	}
}
