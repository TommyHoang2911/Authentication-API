package config

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq" // postgres driver
)

// InitDB reads configuration from the environment and opens a connection
// to the PostgreSQL database. It returns a *sql.DB that must be closed
// by the caller when the application shuts down.
func InitDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		host := os.Getenv("DB_HOST")
		db_user := os.Getenv("DB_USER")
		db_pass := os.Getenv("DB_PASSWORD")
		port := os.Getenv("PORT")
		// default placeholder pointing to a local database. Users should
		// override this with a real connection string in production.
		dsn = fmt.Sprintf("%s://%s:%s@localhost:%s/authdb?sslmode=disable", host, db_user, db_pass, port)
	}

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
