package router

import (
	"auth-service/internal/handler"
	"auth-service/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(authHandler *handler.AuthHandler) *gin.Engine {
	r := gin.Default()

	// health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Public routes
	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)
	r.POST("/refresh_token", authHandler.RefreshToken)

	// Protected routes
	protected := r.Group("/")
	protected.Use(middleware.JWTAuthMiddleware())
	{
		// get user info
		protected.GET("/user", authHandler.GetUser)
		// sign out (invalidate refresh token)
		protected.POST("/sign_out", authHandler.SignOut)
	}

	return r
}
