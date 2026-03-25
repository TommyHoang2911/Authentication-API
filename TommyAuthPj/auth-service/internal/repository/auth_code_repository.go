package repository

import (
	"auth-service/internal/model"
	"database/sql"
	"time"
)

// AuthCodeRepository performs CRUD operations on the auth_codes table.
type AuthCodeRepository struct {
	db *sql.DB
}

// NewAuthCodeRepository returns a new instance bound to the provided database connection.
func NewAuthCodeRepository(db *sql.DB) *AuthCodeRepository {
	return &AuthCodeRepository{db: db}
}

// Create inserts a new auth code record.
func (r *AuthCodeRepository) Create(authCode *model.AuthCode) error {
	query := `
INSERT INTO auth_codes (user_id, code, device_id, expires_at, used, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id
`
	authCode.CreatedAt = time.Now()
	return r.db.QueryRow(query, authCode.UserID, authCode.Code, authCode.DeviceID, authCode.ExpiresAt, authCode.Used, authCode.CreatedAt).Scan(&authCode.ID)
}

// FindByCode retrieves an auth code by code if it exists, is not used, and not expired.
func (r *AuthCodeRepository) FindByCode(code string) (*model.AuthCode, error) {
	query := `
SELECT id, user_id, code, device_id, expires_at, used, created_at
FROM auth_codes
WHERE code = $1 AND used = FALSE AND expires_at > NOW()
`
	authCode := &model.AuthCode{}
	err := r.db.QueryRow(query, code).Scan(&authCode.ID, &authCode.UserID, &authCode.Code, &authCode.DeviceID, &authCode.ExpiresAt, &authCode.Used, &authCode.CreatedAt)
	if err != nil {
		return nil, err
	}
	return authCode, nil
}

// MarkAsUsed marks an auth code as used.
func (r *AuthCodeRepository) MarkAsUsed(code string) error {
	query := `UPDATE auth_codes SET used = TRUE WHERE code = $1`
	_, err := r.db.Exec(query, code)
	return err
}

// DeleteExpired deletes expired auth codes.
func (r *AuthCodeRepository) DeleteExpired() error {
	query := `DELETE FROM auth_codes WHERE expires_at <= NOW()`
	_, err := r.db.Exec(query)
	return err
}
