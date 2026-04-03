package httptransport

import (
	"auth-service/internal/handler"
	"auth-service/internal/middleware"
	ws "auth-service/internal/transport/ws"
	appjwt "auth-service/pkg/jwt"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter configures and returns the Gin router with all routes.
func SetupRouter(authHandler *handler.AuthHandler, hub *ws.Hub, tokenManager appjwt.Manager, enableSwagger bool) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	if enableSwagger {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	}

	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)
	r.GET("/auth/:provider/login", authHandler.OAuthLogin)
	r.GET("/auth/:provider/callback", authHandler.OAuthCallback)
	r.POST("/generate_qr", authHandler.GenerateQR)
	r.POST("/exchange_code", authHandler.ExchangeCode)
	r.GET("/confirm_email", authHandler.ConfirmEmail)
	r.POST("/resend_confirmation", authHandler.ResendConfirmationEmail)
	r.GET("/ws", hub.HandleWebSocket)

	protected := r.Group("/")
	protected.Use(middleware.JWTAuthMiddleware(tokenManager))
	{
		protected.GET("/user", authHandler.GetUser)
		protected.POST("/sign_out", authHandler.SignOut)
		protected.POST("/refresh_token", authHandler.RefreshToken)
		protected.POST("/verify_qr", authHandler.VerifyQR)
	}

	return r
}
