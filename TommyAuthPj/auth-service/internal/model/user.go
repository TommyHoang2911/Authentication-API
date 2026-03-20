package model

import "time"

// User represents a registered account. Password is omitted when
// serializing to JSON to avoid leaking sensitive data.
type User struct {
	ID                 int64     `json:"id"`
	Email              string    `json:"email"`
	Password           string    `json:"-"`
	RefreshToken       string    `json:"-"`
	RefreshTokenExpiry time.Time `json:"-"`
	CreatedAt          time.Time `json:"created_at"`
}
