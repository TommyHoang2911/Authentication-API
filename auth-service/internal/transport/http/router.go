package httptransport

import (
	"auth-service/internal/handler"
	"auth-service/internal/middleware"
	ws "auth-service/internal/transport/ws"
	appjwt "auth-service/pkg/jwt"
	"crypto/subtle"
	"net"
	"net/http"
	stdpprof "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter configures and returns the Gin router with all routes.
func SetupRouter(authHandler *handler.AuthHandler, healthHandler *handler.HealthHandler, hub *ws.Hub, tokenManager appjwt.Manager, enableSwagger bool, enablePprof bool, pprofAuthToken string, swaggerUsername string, swaggerPassword string) *gin.Engine {
	r := gin.Default()
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		panic("failed to configure trusted proxies: " + err.Error())
	}

	config := cors.Config{
		AllowOrigins:     []string{os.Getenv("BASE_URL")},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// Configure CORS middleware
	r.Use(cors.New(config))

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "static")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Welcome to Auth Service",
		})
	})
	r.GET("/device-A-scan-qr", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "device-a.html", gin.H{
			"title": "Device A - Scan QR Code",
		})
	})

	if enableSwagger {
		r.GET("/swagger/*any", gin.BasicAuth(gin.Accounts{swaggerUsername: swaggerPassword}), ginSwagger.WrapHandler(swaggerfiles.Handler))
	}

	if enablePprof {
		registerPprofRoutes(r, pprofAuthToken)
	}

	// Health check routes
	r.GET("/health/liveness", healthHandler.Liveness)
	r.GET("/health/readiness", healthHandler.Readiness)

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
		protected.POST("/cancel_qr", authHandler.CancelQR)
	}

	return r
}

func registerPprofRoutes(r *gin.Engine, pprofAuthToken string) {
	debug := r.Group("/debug/pprof")
	debug.Use(pprofAccessMiddleware(strings.TrimSpace(pprofAuthToken)))
	debug.GET("/", gin.WrapF(stdpprof.Index))
	debug.GET("/cmdline", gin.WrapF(stdpprof.Cmdline))
	debug.GET("/profile", gin.WrapF(stdpprof.Profile))
	debug.GET("/symbol", gin.WrapF(stdpprof.Symbol))
	debug.POST("/symbol", gin.WrapF(stdpprof.Symbol))
	debug.GET("/trace", gin.WrapF(stdpprof.Trace))

	for _, profile := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		debug.GET("/"+profile, gin.WrapH(stdpprof.Handler(profile)))
	}
}

func pprofAccessMiddleware(pprofAuthToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := net.ParseIP(c.ClientIP())
		if ip != nil && ip.IsLoopback() {
			c.Next()
			return
		}

		if pprofAuthToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "pprof access is restricted"})
			return
		}

		providedToken := strings.TrimSpace(c.GetHeader("X-Admin-Token"))
		if subtle.ConstantTimeCompare([]byte(providedToken), []byte(pprofAuthToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin token"})
			return
		}

		c.Next()
	}
}
