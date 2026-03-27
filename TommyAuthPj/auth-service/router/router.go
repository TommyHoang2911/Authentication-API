package router

import (
	"auth-service/internal/handler"
	"auth-service/internal/middleware"
	"auth-service/internal/service/websocket"

	"github.com/gin-gonic/gin"
)

func SetupRouter(authHandler *handler.AuthHandler, hub *websocket.Hub) *gin.Engine {
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
	r.GET("/auth/:provider/login", authHandler.OAuthLogin)
	r.GET("/auth/:provider/callback", authHandler.OAuthCallback)
	r.POST("/generate_qr", authHandler.GenerateQR)
	r.POST("/exchange_code", authHandler.ExchangeCode)
	r.GET("/confirm_email", authHandler.ConfirmEmail)
	r.POST("/resend_confirmation", authHandler.ResendConfirmationEmail)
	r.GET("/ws", hub.HandleWebSocket)

	// Protected routes
	protected := r.Group("/")
	protected.Use(middleware.JWTAuthMiddleware())
	{
		// get user info
		protected.GET("/user", authHandler.GetUser)
		// sign out (invalidate refresh token)
		protected.POST("/sign_out", authHandler.SignOut)
		// refresh token (must be authenticated)
		protected.POST("/refresh_token", authHandler.RefreshToken)
		// verify QR code (must be authenticated)
		protected.POST("/verify_qr", authHandler.VerifyQR)
	}

	return r
}
