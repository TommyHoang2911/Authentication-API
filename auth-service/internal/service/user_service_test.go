package service

import (
	"auth-service/internal/repository"
	"auth-service/internal/testutil"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test constants for user service
const (
	testUserServiceEmail        = "test@example.com"
	testUserServicePassword     = "password123"
	testUserServiceID           = int64(1)
	testUserServiceUserID       = int64(2)
	testUserServiceToken        = "valid-token"
	testUserServiceBadToken     = "bad-token"
	testUserServiceConfirmToken = "confirm-token"
	testUserServiceMissingEmail = "missing@example.com"
)

// SQL query patterns for user service
var (
	insertUserQuery = `
INSERT INTO users (email, password, oauth_provider, oauth_provider_id, refresh_token, refresh_token_expiry, email_confirmed, confirmation_token, confirmation_token_expiry, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id
`
	selectUserByConfirmTokenQuery = `
SELECT id, email, password, oauth_provider, oauth_provider_id, refresh_token, refresh_token_expiry, email_confirmed, confirmation_token, confirmation_token_expiry, created_at
FROM users 
WHERE confirmation_token = $1 AND confirmation_token_expiry > NOW()
`
	selectUserByEmailQuery        = `SELECT id, email, password, oauth_provider, oauth_provider_id, refresh_token, refresh_token_expiry, email_confirmed, confirmation_token, confirmation_token_expiry, created_at FROM users WHERE email = $1`
	updateUserEmailConfirmedQuery = `UPDATE users SET email_confirmed = TRUE, confirmation_token = NULL, confirmation_token_expiry = NULL WHERE id = $1`
)

type mockEmailSender struct {
	mock.Mock
}

func (m *mockEmailSender) SendRegistrationConfirmation(to, confirmationToken string) error {
	args := m.Called(to, confirmationToken)
	return args.Error(0)
}

// setupUserServiceTest creates a UserService with mocked database for testing
func setupUserServiceTest(t *testing.T, emailSender EmailSender) (*UserService, sqlmock.Sqlmock) {
	db, sqlMock := testutil.NewSQLMock(t)
	repo := repository.NewUserRepository(db)
	return NewUserService(repo, emailSender), sqlMock
}

func TestUserService_NewUserService(t *testing.T) {
	repo := &repository.UserRepository{}
	emailSender := &mockEmailSender{}

	service := NewUserService(repo, emailSender)

	assert.NotNil(t, service)
	assert.Equal(t, repo, service.repo)
	assert.Equal(t, emailSender, service.emailService)
}

// TestUserService_Register tests user registration process
func TestUserService_Register(t *testing.T) {
	t.Run("successfully creates user and sends confirmation email", func(t *testing.T) {
		emailSender := &mockEmailSender{}
		service, sqlMock := setupUserServiceTest(t, emailSender)

		t.Setenv("APP_ENV", "")
		t.Setenv("SMTP_TO", "")

		sqlMock.ExpectQuery(regexp.QuoteMeta(insertUserQuery)).
			WithArgs(testUserServiceEmail, sqlmock.AnyArg(), nil, nil, nil, nil, false, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(testUserServiceID))

		emailSender.On("SendRegistrationConfirmation", testUserServiceEmail, mock.AnythingOfType("string")).Return(nil)

		user, err := service.Register(testUserServiceEmail, testUserServicePassword)
		require.NoError(t, err)
		assert.Equal(t, testUserServiceID, user.ID)
		assert.Equal(t, testUserServiceEmail, user.Email)
		assert.False(t, user.EmailConfirmed)
		emailSender.AssertExpectations(t)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error on user creation", func(t *testing.T) {
		emailSender := &mockEmailSender{}
		service, sqlMock := setupUserServiceTest(t, emailSender)

		t.Setenv("APP_ENV", "")
		t.Setenv("SMTP_TO", "")

		sqlMock.ExpectQuery(regexp.QuoteMeta(insertUserQuery)).
			WithArgs(testUserServiceEmail, sqlmock.AnyArg(), nil, nil, nil, nil, false, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(sql.ErrConnDone)

		_, err := service.Register(testUserServiceEmail, testUserServicePassword)
		require.Error(t, err)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

// TestUserService_ConfirmEmail tests email confirmation process
func TestUserService_ConfirmEmail(t *testing.T) {
	t.Run("successfully confirms valid token", func(t *testing.T) {
		service, sqlMock := setupUserServiceTest(t, nil)

		now := time.Now()
		sqlMock.ExpectQuery(regexp.QuoteMeta(selectUserByConfirmTokenQuery)).
			WithArgs(testUserServiceToken).
			WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "oauth_provider", "oauth_provider_id", "refresh_token", "refresh_token_expiry", "email_confirmed", "confirmation_token", "confirmation_token_expiry", "created_at"}).
				AddRow(testUserServiceUserID, testUserServiceEmail, "hashed", nil, nil, "", now, false, testUserServiceToken, now, now))

		sqlMock.ExpectExec(regexp.QuoteMeta(updateUserEmailConfirmedQuery)).
			WithArgs(testUserServiceUserID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.ConfirmEmail(testUserServiceToken)
		require.NoError(t, err)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("invalid or expired token", func(t *testing.T) {
		service, sqlMock := setupUserServiceTest(t, nil)

		sqlMock.ExpectQuery(regexp.QuoteMeta(selectUserByConfirmTokenQuery)).
			WithArgs(testUserServiceBadToken).
			WillReturnError(sql.ErrNoRows)

		err := service.ConfirmEmail(testUserServiceBadToken)
		require.EqualError(t, err, "[USC_CONFIRM_EMAIL_ERROR] invalid or expired confirmation token")
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

// TestUserService_ResendConfirmationEmail tests resending confirmation email
func TestUserService_ResendConfirmationEmail(t *testing.T) {
	t.Run("successfully sends confirmation email to user", func(t *testing.T) {
		emailSender := &mockEmailSender{}
		service, sqlMock := setupUserServiceTest(t, emailSender)

		t.Setenv("APP_ENV", "")
		t.Setenv("SMTP_TO", "")

		now := time.Now()
		nonExpired := now.Add(time.Hour)
		sqlMock.ExpectQuery(regexp.QuoteMeta(selectUserByEmailQuery)).
			WithArgs(testUserServiceEmail).
			WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "oauth_provider", "oauth_provider_id", "refresh_token", "refresh_token_expiry", "email_confirmed", "confirmation_token", "confirmation_token_expiry", "created_at"}).
				AddRow(testUserServiceID, testUserServiceEmail, "hashed", nil, nil, "", now, false, testUserServiceConfirmToken, nonExpired, now))

		emailSender.On("SendRegistrationConfirmation", testUserServiceEmail, testUserServiceConfirmToken).Return(nil)

		err := service.ResendConfirmationEmail(testUserServiceEmail)
		require.NoError(t, err)
		emailSender.AssertExpectations(t)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("expired confirmation token refreshes token and resends email", func(t *testing.T) {
		emailSender := &mockEmailSender{}
		service, sqlMock := setupUserServiceTest(t, emailSender)

		t.Setenv("APP_ENV", "")
		t.Setenv("SMTP_TO", "")

		expired := time.Now().Add(-time.Hour)
		sqlMock.ExpectQuery(regexp.QuoteMeta(selectUserByEmailQuery)).
			WithArgs(testUserServiceEmail).
			WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "oauth_provider", "oauth_provider_id", "refresh_token", "refresh_token_expiry", "email_confirmed", "confirmation_token", "confirmation_token_expiry", "created_at"}).
				AddRow(testUserServiceID, testUserServiceEmail, "hashed", nil, nil, "", expired, false, testUserServiceConfirmToken, expired, time.Now()))

		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET confirmation_token = $1, confirmation_token_expiry = $2 WHERE id = $3`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), testUserServiceID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		emailSender.On("SendRegistrationConfirmation", testUserServiceEmail, mock.AnythingOfType("string")).Return(nil)

		err := service.ResendConfirmationEmail(testUserServiceEmail)
		require.NoError(t, err)
		emailSender.AssertExpectations(t)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("already confirmed email returns an error", func(t *testing.T) {
		service, sqlMock := setupUserServiceTest(t, nil)

		now := time.Now()
		sqlMock.ExpectQuery(regexp.QuoteMeta(selectUserByEmailQuery)).
			WithArgs(testUserServiceEmail).
			WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "oauth_provider", "oauth_provider_id", "refresh_token", "refresh_token_expiry", "email_confirmed", "confirmation_token", "confirmation_token_expiry", "created_at"}).
				AddRow(testUserServiceID, testUserServiceEmail, "hashed", nil, nil, "", now, true, testUserServiceConfirmToken, now, now))

		err := service.ResendConfirmationEmail(testUserServiceEmail)
		require.EqualError(t, err, "email already confirmed")
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("missing confirmation token returns an error", func(t *testing.T) {
		service, sqlMock := setupUserServiceTest(t, nil)

		now := time.Now()
		sqlMock.ExpectQuery(regexp.QuoteMeta(selectUserByEmailQuery)).
			WithArgs(testUserServiceEmail).
			WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "oauth_provider", "oauth_provider_id", "refresh_token", "refresh_token_expiry", "email_confirmed", "confirmation_token", "confirmation_token_expiry", "created_at"}).
				AddRow(testUserServiceID, testUserServiceEmail, "hashed", nil, nil, "", now, false, nil, nil, now))

		err := service.ResendConfirmationEmail(testUserServiceEmail)
		require.EqualError(t, err, "no confirmation token found")
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		service, sqlMock := setupUserServiceTest(t, nil)

		sqlMock.ExpectQuery(regexp.QuoteMeta(selectUserByEmailQuery)).
			WithArgs(testUserServiceMissingEmail).
			WillReturnError(sql.ErrNoRows)

		err := service.ResendConfirmationEmail(testUserServiceMissingEmail)
		require.EqualError(t, err, "[USC_RESEND_CONFIRMATION_EMAIL_ERROR] user not found")
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})
}
