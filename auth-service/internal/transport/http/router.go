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
	"strings"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter configures and returns the Gin router with all routes.
func SetupRouter(authHandler *handler.AuthHandler, healthHandler *handler.HealthHandler, hub *ws.Hub, tokenManager appjwt.Manager, enableSwagger bool, enablePprof bool, pprofAuthToken string) *gin.Engine {
	r := gin.Default()
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		panic("failed to configure trusted proxies: " + err.Error())
	}

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "static")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Welcome to Auth Service",
		})
	})

	if enableSwagger {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
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
