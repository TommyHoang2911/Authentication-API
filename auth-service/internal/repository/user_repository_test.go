package repository

import (
	"auth-service/internal/domain"
	"auth-service/internal/testutil"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := NewUserRepository(db)

	user := &domain.User{
		Email:          "test@example.com",
		Password:       "hashed",
		EmailConfirmed: false,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
INSERT INTO users (email, password, oauth_provider, oauth_provider_id, refresh_token, refresh_token_expiry, email_confirmed, confirmation_token, confirmation_token_expiry, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id
`)).
		WithArgs(user.Email, user.Password, user.OAuthProvider, user.OAuthProviderID, user.RefreshToken, user.RefreshTokenExpiry, user.EmailConfirmed, user.ConfirmationToken, user.ConfirmationTokenExpiry, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))

	err := repo.Create(user)
	require.NoError(t, err)
	assert.Equal(t, int64(10), user.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := NewUserRepository(db)

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password, oauth_provider, oauth_provider_id, refresh_token, refresh_token_expiry, email_confirmed, confirmation_token, confirmation_token_expiry, created_at FROM users WHERE email = $1`)).
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "oauth_provider", "oauth_provider_id", "refresh_token", "refresh_token_expiry", "email_confirmed", "confirmation_token", "confirmation_token_expiry", "created_at"}).
			AddRow(int64(1), "test@example.com", "hashed", nil, nil, nil, nil, true, nil, nil, now))

	user, err := repo.FindByEmail("test@example.com")
	require.NoError(t, err)
	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ConfirmEmail(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := NewUserRepository(db)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET email_confirmed = TRUE, confirmation_token = NULL, confirmation_token_expiry = NULL WHERE id = $1`)).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.ConfirmEmail(7)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByConfirmationToken_NotFound(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id, email, password, oauth_provider, oauth_provider_id, refresh_token, refresh_token_expiry, email_confirmed, confirmation_token, confirmation_token_expiry, created_at
FROM users 
WHERE confirmation_token = $1 AND confirmation_token_expiry > NOW()
`)).
		WithArgs("missing-token").
		WillReturnError(assert.AnError)

	_, err := repo.FindByConfirmationToken("missing-token")
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
