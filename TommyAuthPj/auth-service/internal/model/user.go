package model

import "time"

// User represents a registered account. Password is omitted when
// serializing to JSON to avoid leaking sensitive data.
type User struct {
	ID                      int64      `json:"id"`
	Email                   string     `json:"email"`
	Password                string     `json:"-"`
	RefreshToken            string     `json:"-"`
	RefreshTokenExpiry      time.Time  `json:"-"`
	EmailConfirmed          bool       `json:"email_confirmed"`
	ConfirmationToken       *string    `json:"-"`
	ConfirmationTokenExpiry *time.Time `json:"-"`
	CreatedAt               time.Time  `json:"created_at"`
}

// AuthCode represents a temporary authentication code for QR sign-in.
type AuthCode struct {
	ID        int64     `json:"id"`
	UserID    *int64    `json:"user_id"` // nullable - user_id is set only after Device A verifies the code
	Code      string    `json:"code"`
	DeviceID  string    `json:"device_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}
