package service

import (
	"auth-service/internal/domain"
	"auth-service/internal/repository"
	"auth-service/internal/testutil"
	appjwt "auth-service/pkg/jwt"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testJWTSecret     = "test-secret-key-for-testing"
	testToken         = "good-token"
	testBadToken      = "bad-token"
	testTokenToDelete = "token-to-delete"
	testUserID        = int64(42)
	testUserEmail     = "test@example.com"
	testUserPassword  = "hashed"
	testUser99ID      = int64(99)
	testUser99Email   = "test99@example.com"
)

// SQL query patterns for mocking
var (
	selectRefreshTokenQuery = `
SELECT user_id FROM refresh_tokens 
WHERE token = $1 AND expires_at > NOW()
`
	selectUserQuery         = `SELECT id, email, password, oauth_provider, oauth_provider_id, refresh_token, refresh_token_expiry, email_confirmed, confirmation_token, confirmation_token_expiry, created_at FROM users WHERE id = $1`
	insertRefreshTokenQuery = `
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
`
	deleteRefreshTokenQuery = `DELETE FROM refresh_tokens WHERE token = $1`
)

// setupTokenServiceTest creates a TokenService with mocked database
func setupTokenServiceTest(t *testing.T) (*TokenService, sqlmock.Sqlmock) {
	db, sqlMock := testutil.NewSQLMock(t)
	repo := repository.NewUserRepository(db)
	userService := NewUserService(repo, nil)
	service := NewTokenService(userService, appjwt.NewHMACManager(testJWTSecret))
	return service, sqlMock
}

// setupTokenServiceWithMockUser creates a TokenService with a mock UserService
func setupTokenServiceWithMockUser(t *testing.T) *TokenService {
	mockUserService := &UserService{}
	return NewTokenService(mockUserService, appjwt.NewHMACManager(testJWTSecret))
}

func TestTokenService_NewTokenService(t *testing.T) {
	service := setupTokenServiceWithMockUser(t)
	assert.NotNil(t, service)
	assert.NotNil(t, service.userService)
}

// TestTokenService_GenerateTokens tests JWT and refresh token generation
func TestTokenService_GenerateTokens(t *testing.T) {
	t.Run("successful token generation", func(t *testing.T) {
		service, sqlMock := setupTokenServiceTest(t)
		user := &domain.User{ID: testUser99ID, Email: testUser99Email}

		sqlMock.ExpectExec(regexp.QuoteMeta(insertRefreshTokenQuery)).
			WithArgs(testUser99ID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		accessToken, refreshToken, err := service.GenerateTokens(user)
		require.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error on refresh token creation", func(t *testing.T) {
		service, sqlMock := setupTokenServiceTest(t)
		user := &domain.User{ID: testUser99ID, Email: testUser99Email}

		sqlMock.ExpectExec(regexp.QuoteMeta(insertRefreshTokenQuery)).
			WithArgs(testUser99ID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(sql.ErrConnDone)

		_, _, err := service.GenerateTokens(user)
		require.Error(t, err)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

// TestTokenService_RefreshToken tests refresh token validation and JWT generation
func TestTokenService_RefreshToken(t *testing.T) {
	t.Run("invalid or expired refresh token", func(t *testing.T) {
		service, sqlMock := setupTokenServiceTest(t)

		sqlMock.ExpectQuery(regexp.QuoteMeta(selectRefreshTokenQuery)).
			WithArgs(testBadToken).
			WillReturnError(sql.ErrNoRows)

		_, err := service.RefreshToken(testBadToken)
		require.EqualError(t, err, "invalid or expired refresh token")
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("valid refresh token generates new access token", func(t *testing.T) {
		service, sqlMock := setupTokenServiceTest(t)
		now := time.Now()

		// First query: find user ID from refresh token
		sqlMock.ExpectQuery(regexp.QuoteMeta(selectRefreshTokenQuery)).
			WithArgs(testToken).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(testUserID))

		// Second query: get user details
		sqlMock.ExpectQuery(regexp.QuoteMeta(selectUserQuery)).
			WithArgs(testUserID).
			WillReturnRows(sqlmock.NewRows(
				[]string{"id", "email", "password", "oauth_provider", "oauth_provider_id",
					"refresh_token", "refresh_token_expiry", "email_confirmed",
					"confirmation_token", "confirmation_token_expiry", "created_at"}).
				AddRow(testUserID, testUserEmail, testUserPassword, nil, nil, "", now, true, nil, nil, now))

		accessToken, err := service.RefreshToken(testToken)
		require.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("user not found error", func(t *testing.T) {
		service, sqlMock := setupTokenServiceTest(t)

		sqlMock.ExpectQuery(regexp.QuoteMeta(selectRefreshTokenQuery)).
			WithArgs(testToken).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(testUserID))

		sqlMock.ExpectQuery(regexp.QuoteMeta(selectUserQuery)).
			WithArgs(testUserID).
			WillReturnError(sql.ErrNoRows)

		_, err := service.RefreshToken(testToken)
		require.Error(t, err)
		require.Equal(t, "user not found", err.Error())
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

// TestTokenService_SignOut tests refresh token invalidation
func TestTokenService_SignOut(t *testing.T) {
	t.Run("successfully invalidates refresh token", func(t *testing.T) {
		service, sqlMock := setupTokenServiceTest(t)

		sqlMock.ExpectExec(regexp.QuoteMeta(deleteRefreshTokenQuery)).
			WithArgs(testTokenToDelete).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.SignOut(testTokenToDelete)
		require.NoError(t, err)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error on token deletion", func(t *testing.T) {
		service, sqlMock := setupTokenServiceTest(t)

		sqlMock.ExpectExec(regexp.QuoteMeta(deleteRefreshTokenQuery)).
			WithArgs(testTokenToDelete).
			WillReturnError(sql.ErrConnDone)

		err := service.SignOut(testTokenToDelete)
		require.Error(t, err)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})
}
