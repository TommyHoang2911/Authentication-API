package repository

import (
	"database/sql"
	"time"

	"auth-service/internal/model"
)

// UserRepository performs CRUD operations on the users table.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository returns a new instance bound to the provided database connection.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user record and updates the user object with its
// auto-generated ID.
func (r *UserRepository) Create(user *model.User) error {
	query := `
INSERT INTO users (email, password, refresh_token, refresh_token_expiry, email_confirmed, confirmation_token, confirmation_token_expiry, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id
`
	user.CreatedAt = time.Now()
	return r.db.QueryRow(query, user.Email, user.Password, user.RefreshToken, user.RefreshTokenExpiry, user.EmailConfirmed, user.ConfirmationToken, user.ConfirmationTokenExpiry, user.CreatedAt).Scan(&user.ID)
}

// FindByEmail retrieves a user by their email address.
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	query := `SELECT id, email, password, refresh_token, refresh_token_expiry, email_confirmed, confirmation_token, confirmation_token_expiry, created_at FROM users WHERE email = $1`
	user := &model.User{}
	err := r.db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Password, &user.RefreshToken, &user.RefreshTokenExpiry, &user.EmailConfirmed, &user.ConfirmationToken, &user.ConfirmationTokenExpiry, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// EmailExists checks if a user with the given email already exists.
func (r *UserRepository) EmailExists(email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := r.db.QueryRow(query, email).Scan(&exists)
	return exists, err
}

// FindByID retrieves a user by their ID.
func (r *UserRepository) FindByID(id int64) (*model.User, error) {
	query := `SELECT id, email, password, refresh_token, refresh_token_expiry, email_confirmed, confirmation_token, confirmation_token_expiry, created_at FROM users WHERE id = $1`
	user := &model.User{}
	err := r.db.QueryRow(query, id).Scan(&user.ID, &user.Email, &user.Password, &user.RefreshToken, &user.RefreshTokenExpiry, &user.EmailConfirmed, &user.ConfirmationToken, &user.ConfirmationTokenExpiry, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateRefreshToken updates the refresh token and expiry for a user.
func (r *UserRepository) UpdateRefreshToken(userID int64, refreshToken string, expiry time.Time) error {
	query := `UPDATE users SET refresh_token = $1, refresh_token_expiry = $2 WHERE id = $3`
	_, err := r.db.Exec(query, refreshToken, expiry, userID)
	return err
}

// CreateRefreshToken inserts a new refresh token record.
func (r *UserRepository) CreateRefreshToken(userID int64, token string, expiresAt time.Time) error {
	query := `
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
`
	_, err := r.db.Exec(query, userID, token, expiresAt)
	return err
}

// FindRefreshToken retrieves a user ID by refresh token if it exists and is not expired.
func (r *UserRepository) FindRefreshToken(token string) (int64, error) {
	query := `
SELECT user_id FROM refresh_tokens 
WHERE token = $1 AND expires_at > NOW()
`
	var userID int64
	err := r.db.QueryRow(query, token).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

// DeleteRefreshToken removes a refresh token record.
func (r *UserRepository) DeleteRefreshToken(token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`
	_, err := r.db.Exec(query, token)
	return err
}

// DeleteRefreshTokensByUserID removes all refresh tokens for a user.
func (r *UserRepository) DeleteRefreshTokensByUserID(userID int64) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}

// FindRefreshTokenByUserID retrieves the refresh token for a user if it exists and is not expired.
func (r *UserRepository) FindRefreshTokenByUserID(userID int64) (string, error) {
	query := `
SELECT token FROM refresh_tokens 
WHERE user_id = $1 AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1
`
	var token string
	err := r.db.QueryRow(query, userID).Scan(&token)
	if err != nil {
		return "", err
	}
	return token, nil
}

// FindByConfirmationToken retrieves a user by their confirmation token if it exists and is not expired.
func (r *UserRepository) FindByConfirmationToken(token string) (*model.User, error) {
	query := `
SELECT id, email, password, refresh_token, refresh_token_expiry, email_confirmed, confirmation_token, confirmation_token_expiry, created_at 
FROM users 
WHERE confirmation_token = $1 AND confirmation_token_expiry > NOW()
`
	user := &model.User{}
	err := r.db.QueryRow(query, token).Scan(&user.ID, &user.Email, &user.Password, &user.RefreshToken, &user.RefreshTokenExpiry, &user.EmailConfirmed, &user.ConfirmationToken, &user.ConfirmationTokenExpiry, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// ConfirmEmail updates a user's email confirmation status and clears the confirmation token.
func (r *UserRepository) ConfirmEmail(userID int64) error {
	query := `UPDATE users SET email_confirmed = TRUE, confirmation_token = NULL, confirmation_token_expiry = NULL WHERE id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}

// UpdateConfirmationToken updates a user's confirmation token and expiry.
func (r *UserRepository) UpdateConfirmationToken(userID int64, token string, expiry time.Time) error {
	query := `UPDATE users SET confirmation_token = $1, confirmation_token_expiry = $2 WHERE id = $3`
	_, err := r.db.Exec(query, token, expiry, userID)
	return err
}
