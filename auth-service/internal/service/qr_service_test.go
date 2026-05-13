package service

import (
	"auth-service/internal/repository"
	"auth-service/internal/testutil"
	ws "auth-service/internal/transport/ws"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testQRDeviceID   = "device-1"
	testQRCode       = "qr-code"
	testQRTempCode   = "temp-code"
	testQRUserID     = int64(12)
	testQRAuthCodeID = int64(1)
	testQRNewCodeID  = int64(2)
)

// SQL query patterns
var (
	insertAuthCodeQuery = `
INSERT INTO auth_codes (user_id, code, device_id, expires_at, used, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id
`
	selectAuthCodeQuery = `
SELECT id, user_id, code, device_id, expires_at, used, created_at
FROM auth_codes
WHERE code = $1 AND used = FALSE AND expires_at > NOW()
`
	updateAuthCodeUsedQuery = `UPDATE auth_codes SET used = TRUE WHERE code = $1`
)

// setupQRServiceTest creates a QRService with mocked database for testing
func setupQRServiceTest(t *testing.T) (*QRService, sqlmock.Sqlmock) {
	db, sqlMock := testutil.NewSQLMock(t)
	repo := repository.NewAuthCodeRepository(db)
	hub := ws.NewHub()
	return NewQRService(repo, nil, nil, hub), sqlMock
}

func TestQRService_NewQRService(t *testing.T) {
	hub := ws.NewHub()
	service := NewQRService(nil, nil, nil, hub)

	assert.NotNil(t, service)
}

// TestQRService_GenerateQRCode tests QR code generation
func TestQRService_GenerateQRCode(t *testing.T) {
	t.Run("successfully generates QR code", func(t *testing.T) {
		service, sqlMock := setupQRServiceTest(t)

		sqlMock.ExpectQuery(regexp.QuoteMeta(insertAuthCodeQuery)).
			WithArgs(nil, sqlmock.AnyArg(), testQRDeviceID, sqlmock.AnyArg(), false, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(testQRAuthCodeID))

		code, err := service.GenerateQRCode(testQRDeviceID)
		require.NoError(t, err)
		assert.NotEmpty(t, code)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error on code insertion", func(t *testing.T) {
		service, sqlMock := setupQRServiceTest(t)

		sqlMock.ExpectQuery(regexp.QuoteMeta(insertAuthCodeQuery)).
			WithArgs(nil, sqlmock.AnyArg(), testQRDeviceID, sqlmock.AnyArg(), false, sqlmock.AnyArg()).
			WillReturnError(sqlmock.ErrCancelled)

		code, err := service.GenerateQRCode(testQRDeviceID)
		require.Error(t, err)
		assert.Empty(t, code)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

// TestQRService_VerifyQRCode tests QR code verification and user linking
func TestQRService_VerifyQRCode(t *testing.T) {
	t.Run("successfully verifies QR code and links to user", func(t *testing.T) {
		service, sqlMock := setupQRServiceTest(t)
		now := time.Now().Add(time.Minute)

		sqlMock.ExpectQuery(regexp.QuoteMeta(selectAuthCodeQuery)).
			WithArgs(testQRCode).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "code", "device_id", "expires_at", "used", "created_at"}).
				AddRow(testQRAuthCodeID, nil, testQRCode, testQRDeviceID, now, false, now))

		sqlMock.ExpectExec(regexp.QuoteMeta(updateAuthCodeUsedQuery)).
			WithArgs(testQRCode).
			WillReturnResult(sqlmock.NewResult(0, 1))

		sqlMock.ExpectQuery(regexp.QuoteMeta(insertAuthCodeQuery)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "", sqlmock.AnyArg(), false, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(testQRNewCodeID))

		err := service.VerifyQRCode(testQRCode, testQRUserID)
		require.NoError(t, err)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("QR code not found or expired", func(t *testing.T) {
		service, sqlMock := setupQRServiceTest(t)

		sqlMock.ExpectQuery(regexp.QuoteMeta(selectAuthCodeQuery)).
			WithArgs(testQRCode).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "code", "device_id", "expires_at", "used", "created_at"}))

		err := service.VerifyQRCode(testQRCode, testQRUserID)
		require.EqualError(t, err, "invalid or expired code")
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

// TestQRService_ExchangeCode tests temporary code exchange for refresh token
func TestQRService_ExchangeCode(t *testing.T) {
	t.Run("temp code not linked to user", func(t *testing.T) {
		service, sqlMock := setupQRServiceTest(t)
		now := time.Now().Add(time.Minute)

		sqlMock.ExpectQuery(regexp.QuoteMeta(selectAuthCodeQuery)).
			WithArgs(testQRTempCode).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "code", "device_id", "expires_at", "used", "created_at"}).
				AddRow(int64(3), nil, testQRTempCode, "", now, false, now))

		sqlMock.ExpectExec(regexp.QuoteMeta(updateAuthCodeUsedQuery)).
			WithArgs(testQRTempCode).
			WillReturnResult(sqlmock.NewResult(0, 1))

		_, _, _, err := service.ExchangeCode(testQRTempCode)
		require.EqualError(t, err, "temp code not linked to a user")
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("code not found or expired", func(t *testing.T) {
		service, sqlMock := setupQRServiceTest(t)

		sqlMock.ExpectQuery(regexp.QuoteMeta(selectAuthCodeQuery)).
			WithArgs(testQRTempCode).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "code", "device_id", "expires_at", "used", "created_at"}))

		_, _, _, err := service.ExchangeCode(testQRTempCode)
		require.EqualError(t, err, "invalid or expired temp code")
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

// mockHub is a test mock for WebSocket Hub that captures sent messages
type mockHub struct {
	messages []map[string]interface{}
	codes    []string
}

func newMockHub() *mockHub {
	return &mockHub{
		messages: make([]map[string]interface{}, 0),
		codes:    make([]string, 0),
	}
}

func (m *mockHub) SendMessage(code string, message map[string]interface{}) {
	m.codes = append(m.codes, code)
	m.messages = append(m.messages, message)
}

func (m *mockHub) RegisterConnection(code string, conn *websocket.Conn) {}
func (m *mockHub) UnregisterConnection(code string)                 {}

// setupQRServiceTestWithMockHub creates a QRService with mocked database and mock hub for testing
func setupQRServiceTestWithMockHub(t *testing.T) (*QRService, sqlmock.Sqlmock, *mockHub) {
	db, sqlMock := testutil.NewSQLMock(t)
	repo := repository.NewAuthCodeRepository(db)
	mockHub := newMockHub()
	service := NewQRService(repo, nil, nil, mockHub)
	return service, sqlMock, mockHub
}

// TestQRService_CancelQRCode tests QR code cancellation
func TestQRService_CancelQRCode(t *testing.T) {
	t.Run("successfully cancels QR code", func(t *testing.T) {
		service, sqlMock, mockHub := setupQRServiceTestWithMockHub(t)

		sqlMock.ExpectExec(regexp.QuoteMeta(updateAuthCodeUsedQuery)).
			WithArgs(testQRCode).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.CancelQRCode(testQRCode)
		require.NoError(t, err)
		require.NoError(t, sqlMock.ExpectationsWereMet())

		// Verify WebSocket message was sent
		assert.Len(t, mockHub.messages, 1)
		assert.Len(t, mockHub.codes, 1)
		assert.Equal(t, testQRCode, mockHub.codes[0])
		assert.Equal(t, "cancelled", mockHub.messages[0]["status"])
		assert.Equal(t, "QR code has been cancelled", mockHub.messages[0]["message"])
	})

	t.Run("database error when marking code as used", func(t *testing.T) {
		service, sqlMock, mockHub := setupQRServiceTestWithMockHub(t)

		sqlMock.ExpectExec(regexp.QuoteMeta(updateAuthCodeUsedQuery)).
			WithArgs(testQRCode).
			WillReturnError(sqlmock.ErrCancelled)

		err := service.CancelQRCode(testQRCode)
		require.Error(t, err)
		require.NoError(t, sqlMock.ExpectationsWereMet())

		// Verify error WebSocket message was sent
		assert.Len(t, mockHub.messages, 1)
		assert.Len(t, mockHub.codes, 1)
		assert.Equal(t, testQRCode, mockHub.codes[0])
		assert.Equal(t, "error", mockHub.messages[0]["status"])
		assert.Equal(t, "Failed to cancel QR code", mockHub.messages[0]["message"])
	})
}
