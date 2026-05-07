package config

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	apperrors "auth-service/internal/errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type SMTP struct {
	Host           string
	Port           string
	User           string
	Pass           string
	From           string
	TimeoutSeconds int
}

type OAuth struct {
	GoogleClientID       string
	GoogleClientSecret   string
	GoogleRedirectURL    string
	FacebookClientID     string
	FacebookClientSecret string
	FacebookRedirectURL  string
}

type EmailQueue struct {
	Workers           int
	BufferSize        int
	MaxRetries        int
	RetryDelaySeconds int
}

// App centralizes runtime configuration for the service.
type App struct {
	Env             string
	ServerPort      string
	DatabaseURL     string
	JWTSecret       string
	BaseURL         string
	EnableSwagger   bool
	EnablePprof     bool
	PprofAuthToken  string
	SwaggerHost     string
	SwaggerUsername string
	SwaggerPassword string
	SMTP            SMTP
	EmailQueue      EmailQueue
	OAuth           OAuth
}

func Load() (*App, error) {
	_ = godotenv.Load()

	env := getEnv("APP_ENV", "production")
	serverPort := getEnv("SERVER_PORT", getEnv("PORT", "8080"))
	cfg := &App{
		Env:            env,
		ServerPort:     serverPort,
		DatabaseURL:    databaseURL(),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		BaseURL:        getEnv("BASE_URL", fmt.Sprintf("http://localhost:%s", serverPort)),
		EnablePprof:    getEnvBool("ENABLE_PPROF", false),
		PprofAuthToken: strings.TrimSpace(os.Getenv("PPROF_AUTH_TOKEN")),
		SMTP: SMTP{
			Host:           os.Getenv("SMTP_HOST"),
			Port:           os.Getenv("SMTP_PORT"),
			User:           os.Getenv("SMTP_USER"),
			Pass:           os.Getenv("SMTP_PASS"),
			From:           os.Getenv("SMTP_FROM"),
			TimeoutSeconds: getEnvInt("SMTP_TIMEOUT_SECONDS", 10),
		},
		EmailQueue: EmailQueue{
			Workers:           getEnvInt("EMAIL_QUEUE_WORKERS", 2),
			BufferSize:        getEnvInt("EMAIL_QUEUE_BUFFER_SIZE", 100),
			MaxRetries:        getEnvInt("EMAIL_QUEUE_MAX_RETRIES", 3),
			RetryDelaySeconds: getEnvInt("EMAIL_QUEUE_RETRY_DELAY_SECONDS", 1),
		},
		OAuth: OAuth{
			GoogleClientID:       os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
			GoogleClientSecret:   os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
			GoogleRedirectURL:    os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
			FacebookClientID:     os.Getenv("FACEBOOK_OAUTH_CLIENT_ID"),
			FacebookClientSecret: os.Getenv("FACEBOOK_OAUTH_CLIENT_SECRET"),
			FacebookRedirectURL:  os.Getenv("FACEBOOK_OAUTH_REDIRECT_URL"),
		},
	}

	cfg.EnableSwagger = true
	if cfg.EnableSwagger {
		swaggerHostEnvKey := "SWAGGER_HOST_" + strings.ToUpper(cfg.Env)
		cfg.SwaggerHost = os.Getenv(swaggerHostEnvKey)
		if strings.TrimSpace(cfg.SwaggerHost) == "" {
			return nil, apperrors.NewValidation(swaggerHostEnvKey + " is required when swagger is enabled")
		}
		cfg.SwaggerUsername = os.Getenv("SWAGGER_USERNAME")
		cfg.SwaggerPassword = os.Getenv("SWAGGER_PASSWORD")
		if strings.TrimSpace(cfg.SwaggerUsername) == "" || strings.TrimSpace(cfg.SwaggerPassword) == "" {
			return nil, apperrors.NewValidation("SWAGGER_USERNAME and SWAGGER_PASSWORD are required when swagger is enabled")
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *App) Validate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return apperrors.NewValidation("database configuration is required")
	}
	if strings.TrimSpace(c.JWTSecret) == "" {
		return apperrors.NewValidation("JWT_SECRET is required")
	}
	if c.JWTSecret == "your-secret-key" {
		return apperrors.NewValidation("JWT_SECRET must be changed from the insecure default value")
	}
	if strings.TrimSpace(c.ServerPort) == "" {
		return apperrors.NewValidation("server port is required")
	}
	if c.EmailQueue.Workers <= 0 {
		return apperrors.NewValidation("EMAIL_QUEUE_WORKERS must be greater than 0")
	}
	if c.EmailQueue.BufferSize <= 0 {
		return apperrors.NewValidation("EMAIL_QUEUE_BUFFER_SIZE must be greater than 0")
	}
	if c.EmailQueue.MaxRetries < 0 {
		return apperrors.NewValidation("EMAIL_QUEUE_MAX_RETRIES must be greater than or equal to 0")
	}
	if c.EmailQueue.RetryDelaySeconds <= 0 {
		return apperrors.NewValidation("EMAIL_QUEUE_RETRY_DELAY_SECONDS must be greater than 0")
	}
	if c.SMTP.TimeoutSeconds <= 0 {
		return apperrors.NewValidation("SMTP_TIMEOUT_SECONDS must be greater than 0")
	}
	if c.EnablePprof && c.Env == "production" && c.PprofAuthToken == "" {
		return apperrors.NewValidation("PPROF_AUTH_TOKEN is required when ENABLE_PPROF=true in production")
	}
	return nil
}

func (c *App) ShouldLogToFile() bool {
	return c.Env == "development"
}

func (c *App) Address() string {
	return ":" + c.ServerPort
}

// InitDB opens and validates a PostgreSQL connection.
func InitDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}

// RunMigrations runs all pending migrations against the database.
func RunMigrations(dbURL string) error {
	m, err := migrate.New("file://db/migrations", dbURL)
	if err != nil {
		return fmt.Errorf("new migration instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func databaseURL() string {
	if dbURL := os.Getenv("DATABASE_URL"); strings.TrimSpace(dbURL) != "" {
		return dbURL
	}

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "authdb")
	sslMode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPass, dbHost, dbPort, dbName, sslMode)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: invalid integer value for %s=%q, using fallback %d: %v\n", key, value, fallback, err)
		return fallback
	}

	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}

	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		fmt.Fprintf(os.Stderr, "warning: invalid boolean value for %s=%q, using fallback %t\n", key, value, fallback)
		return fallback
	}
}
