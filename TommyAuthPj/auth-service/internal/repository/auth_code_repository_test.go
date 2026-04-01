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

func TestAuthCodeRepository_Create(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := NewAuthCodeRepository(db)

	expiresAt := time.Now().Add(2 * time.Minute)
	authCode := &domain.AuthCode{
		Code:      "code-1",
		DeviceID:  "device-1",
		ExpiresAt: expiresAt,
		Used:      false,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
INSERT INTO auth_codes (user_id, code, device_id, expires_at, used, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id
`)).
		WithArgs(authCode.UserID, authCode.Code, authCode.DeviceID, authCode.ExpiresAt, authCode.Used, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(15)))

	err := repo.Create(authCode)
	require.NoError(t, err)
	assert.Equal(t, int64(15), authCode.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthCodeRepository_FindByCode(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := NewAuthCodeRepository(db)

	now := time.Now().Add(time.Minute)
	userID := int64(3)
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id, user_id, code, device_id, expires_at, used, created_at
FROM auth_codes
WHERE code = $1 AND used = FALSE AND expires_at > NOW()
`)).
		WithArgs("code-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "code", "device_id", "expires_at", "used", "created_at"}).
			AddRow(int64(11), userID, "code-1", "device-1", now, false, now))

	result, err := repo.FindByCode("code-1")
	require.NoError(t, err)
	assert.Equal(t, int64(11), result.ID)
	require.NotNil(t, result.UserID)
	assert.Equal(t, userID, *result.UserID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthCodeRepository_MarkAsUsed(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := NewAuthCodeRepository(db)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE auth_codes SET used = TRUE WHERE code = $1`)).
		WithArgs("code-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.MarkAsUsed("code-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthCodeRepository_DeleteExpired(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := NewAuthCodeRepository(db)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM auth_codes WHERE expires_at <= NOW()`)).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err := repo.DeleteExpired()
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
