package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"auth-service/config"
	"auth-service/internal/handler"
	"auth-service/internal/repository"
	"auth-service/internal/service"
	"auth-service/internal/service/websocket"
	"auth-service/router"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "production"
	}

	if appEnv == "development" {
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

	dbURL := os.Getenv("DATABASE_URL")

	if dbURL == "" {
		db_user := os.Getenv("DB_USER")
		db_pass := os.Getenv("DB_PASSWORD")
		port := os.Getenv("PORT")
		dbURL = fmt.Sprintf("postgresql://%s:%s@localhost:%s/authdb?sslmode=disable", db_user, db_pass, port)
	}

	db, err := config.InitDB(dbURL)
	if err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}
	// defer db.Close()

	// Run migrations
	if err := config.RunMigrations(dbURL); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	authCodeRepo := repository.NewAuthCodeRepository(db)
	hub := websocket.NewHub()

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("SMTP_FROM")
	baseURL := os.Getenv("BASE_URL")

	emailService := service.NewEmailService(smtpHost, smtpPort, smtpUser, smtpPass, smtpFrom, baseURL)
	oauthService := service.NewOAuthService(
		os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
		os.Getenv("FACEBOOK_OAUTH_CLIENT_ID"),
		os.Getenv("FACEBOOK_OAUTH_CLIENT_SECRET"),
		os.Getenv("FACEBOOK_OAUTH_REDIRECT_URL"),
	)

	authService := service.NewAuthService(userRepo, authCodeRepo, emailService, oauthService, hub)
	authHandler := handler.NewAuthHandler(authService)

	r := router.SetupRouter(authHandler, hub)
	r.Run(":8080")
}
