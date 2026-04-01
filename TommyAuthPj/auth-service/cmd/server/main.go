package main

import (
	"io"
	"log"
	"os"

	appconfig "auth-service/internal/config"

	"github.com/gin-gonic/gin"

	"auth-service/docs"
	"auth-service/internal/handler"
	"auth-service/internal/repository"
	"auth-service/internal/service"
	httptransport "auth-service/internal/transport/http"
	ws "auth-service/internal/transport/ws"
	appjwt "auth-service/pkg/jwt"
)

// @title Auth Service API
// @version 1.0
// @description Authentication, OAuth, and QR login API for the auth service.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatalf("configuration failed: %v", err)
	}

	if cfg.ShouldLogToFile() {
		if err := os.MkdirAll("log", 0o755); err != nil {
			log.Fatalf("failed to create log directory: %v", err)
		}

		f, err := os.OpenFile("log/logger.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatalf("failed to open log file: %v", err)
		}
		defer f.Close()

		log.SetOutput(f)
		log.SetFlags(log.LstdFlags | log.Lshortfile)

		// Also capture Gin access / error logs in the same log file when running locally.
		gin.DefaultWriter = io.MultiWriter(os.Stdout, f)
		gin.DefaultErrorWriter = io.MultiWriter(os.Stderr, f)
	}

	db, err := appconfig.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}
	defer db.Close()

	if err := appconfig.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	authCodeRepo := repository.NewAuthCodeRepository(db)
	hub := ws.NewHub()
	tokenManager := appjwt.NewHMACManager(cfg.JWTSecret)

	emailService := service.NewEmailService(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.User, cfg.SMTP.Pass, cfg.SMTP.From, cfg.BaseURL)
	oauthService := service.NewOAuthService(
		cfg.OAuth.GoogleClientID,
		cfg.OAuth.GoogleClientSecret,
		cfg.OAuth.GoogleRedirectURL,
		cfg.OAuth.FacebookClientID,
		cfg.OAuth.FacebookClientSecret,
		cfg.OAuth.FacebookRedirectURL,
	)

	authService := service.NewAuthService(userRepo, authCodeRepo, emailService, oauthService, hub, tokenManager)
	authHandler := handler.NewAuthHandler(authService)

	if cfg.EnableSwagger {
		docs.SwaggerInfo.Host = cfg.SwaggerHost
	}
	r := httptransport.SetupRouter(authHandler, hub, tokenManager, cfg.EnableSwagger)
	if err := r.Run(cfg.Address()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
