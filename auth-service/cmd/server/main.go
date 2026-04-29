package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appconfig "auth-service/internal/config"

	"github.com/gin-gonic/gin"

	"auth-service/docs"
	"auth-service/internal/handler"
	"auth-service/internal/repository"
	"auth-service/internal/service"
	"auth-service/internal/service/queue"
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

	emailService := service.NewEmailServiceWithTimeout(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.User,
		cfg.SMTP.Pass,
		cfg.SMTP.From,
		cfg.BaseURL,
		time.Duration(cfg.SMTP.TimeoutSeconds)*time.Second,
	)
	emailSender := queue.NewAsyncEmailSender(
		emailService,
		queue.Config{
			Workers:    cfg.EmailQueue.Workers,
			BufferSize: cfg.EmailQueue.BufferSize,
			MaxRetries: cfg.EmailQueue.MaxRetries,
			RetryDelay: time.Duration(cfg.EmailQueue.RetryDelaySeconds) * time.Second,
		},
		log.Default(),
	)

	oauthService := service.NewOAuthService(
		cfg.OAuth.GoogleClientID,
		cfg.OAuth.GoogleClientSecret,
		cfg.OAuth.GoogleRedirectURL,
		cfg.OAuth.FacebookClientID,
		cfg.OAuth.FacebookClientSecret,
		cfg.OAuth.FacebookRedirectURL,
	)

	authService := service.NewAuthService(userRepo, authCodeRepo, emailSender, oauthService, hub, tokenManager)
	authHandler := handler.NewAuthHandler(authService)
	healthHandler := handler.NewHealthHandler(db)

	if cfg.EnableSwagger {
		docs.SwaggerInfo.Host = cfg.SwaggerHost
	}
	r := httptransport.SetupRouter(authHandler, healthHandler, hub, tokenManager, cfg.EnableSwagger, cfg.EnablePprof, cfg.PprofAuthToken)

	srv := &http.Server{
		Addr:    cfg.Address(),
		Handler: r,
	}

	// Start the HTTP server in a background goroutine so we can handle shutdown signals.
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for SIGINT or SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-serverErr:
		log.Fatalf("server error: %v", err)
	case <-quit:
		log.Println("shutdown signal received")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	if err := emailSender.Shutdown(shutdownCtx); err != nil {
		log.Printf("email queue shutdown warning: %v", err)
	}
}
