package main

import (
	"io"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"auth-service/config"
	"auth-service/internal/handler"
	"auth-service/internal/repository"
	"auth-service/internal/service"
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

	db, err := config.InitDB()
	if err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}
	// defer db.Close()

	// Run migrations
	dbURL := "postgres://tommyhoang:Aa@123456@localhost:5432/authdb?sslmode=disable"
	if err := config.RunMigrations(dbURL); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo)
	authHandler := handler.NewAuthHandler(authService)

	r := router.SetupRouter(authHandler)
	r.Run(":8080")
}
