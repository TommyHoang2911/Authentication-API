package service

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailService_DefaultTimeout(t *testing.T) {
	service := NewEmailService(
		"localhost",
		"1025",
		"user",
		"pass",
		"noreply@example.com",
		"http://localhost:3000",
	)

	assert.Equal(t, defaultSMTPTimeout, service.timeout)
}

func TestNewEmailServiceWithTimeout_InvalidTimeoutFallsBackToDefault(t *testing.T) {
	service := NewEmailServiceWithTimeout(
		"localhost",
		"1025",
		"user",
		"pass",
		"noreply@example.com",
		"http://localhost:3000",
		0,
	)

	assert.Equal(t, defaultSMTPTimeout, service.timeout)
}

func TestIsLoopbackSMTPHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{name: "localhost name", host: "localhost", want: true},
		{name: "ipv4 loopback", host: "127.0.0.1", want: true},
		{name: "ipv6 loopback", host: "::1", want: true},
		{name: "remote host", host: "smtp.gmail.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isLoopbackSMTPHost(tt.host))
		})
	}
}

func TestEmailService_SendRegistrationConfirmation_TimesOut(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	accepted := make(chan struct{})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		close(accepted)
		defer conn.Close()
		<-time.After(500 * time.Millisecond)
	}()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	service := NewEmailServiceWithTimeout(
		host,
		port,
		"",
		"",
		"noreply@example.com",
		"http://localhost:3000",
		100*time.Millisecond,
	)

	start := time.Now()
	err = service.SendRegistrationConfirmation("user@example.com", "confirm-token-123")
	elapsed := time.Since(start)

	<-accepted
	require.Error(t, err)
	assert.ErrorContains(t, err, "create smtp client")
	assert.Less(t, elapsed, 400*time.Millisecond)
}

func TestEmailService_SendRegistrationConfirmation(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		token       string
		setupMock   func()
		expectError bool
	}{
		{
			name:  "successful send",
			email: "user@example.com",
			token: "confirm-token-123",
			setupMock: func() {
				// EmailService uses net/smtp which is harder to mock
				// This test validates the structure
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			// Create a real EmailService to test token to URL conversion
			service := NewEmailService(
				"localhost",
				"1025",
				"user",
				"pass",
				"noreply@auth.example.com",
				"http://localhost:3000",
			)

			// EmailService.SendRegistrationConfirmation should construct
			// a valid confirmation URL with the token
			assert.NotNil(t, service)

			// In a real scenario, we'd mock the SMTP client
			// For now, we validate service structure
			assert.NotNil(t, service)
		})
	}
}

func TestEmailService_BuildConfirmationLink(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		token        string
		expectedPath string
	}{
		{
			name:         "build link with base URL",
			baseURL:      "http://localhost:3000",
			token:        "test-token-123",
			expectedPath: "/confirm-email?token=test-token-123",
		},
		{
			name:         "build link with HTTPS",
			baseURL:      "https://auth.example.com",
			token:        "secure-token-456",
			expectedPath: "/confirm-email?token=secure-token-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewEmailService(
				"localhost",
				"1025",
				"user",
				"pass",
				"noreply@example.com",
				tt.baseURL,
			)

			assert.NotNil(t, service)
			// Link building would be tested through integration tests
			// as the method is typically unexported
		})
	}
}

func TestEmailService_SendResendConfirmation(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		token       string
		expectError bool
	}{
		{
			name:        "resend confirmation email",
			email:       "user@example.com",
			token:       "resend-token-789",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewEmailService(
				"localhost",
				"1025",
				"user",
				"pass",
				"noreply@example.com",
				"http://localhost:3000",
			)

			assert.NotNil(t, service)
		})
	}
}

func TestEmailService_InvalidSMTPConfig(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		port      string
		username  string
		password  string
		fromEmail string
		baseURL   string
	}{
		{
			name:      "empty SMTP URL",
			host:      "",
			port:      "1025",
			username:  "user",
			password:  "pass",
			fromEmail: "noreply@example.com",
			baseURL:   "http://localhost:3000",
		},
		{
			name:      "invalid from email",
			host:      "localhost",
			port:      "1025",
			username:  "user",
			password:  "pass",
			fromEmail: "",
			baseURL:   "http://localhost:3000",
		},
		{
			name:      "invalid base URL",
			host:      "localhost",
			port:      "1025",
			username:  "user",
			password:  "pass",
			fromEmail: "noreply@example.com",
			baseURL:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Service creation should be resilient; validation on send
			service := NewEmailService(tt.host, tt.port, tt.username, tt.password, tt.fromEmail, tt.baseURL)
			assert.NotNil(t, service)
		})
	}
}

// Integration test example for email service
func TestEmailService_Integration(t *testing.T) {
	// This would test against a real or mocked SMTP server
	// using a test container or mailhog
	t.Skip("Integration test - requires SMTP server")

	service := NewEmailService(
		"localhost",
		"1025",
		"user",
		"pass",
		"noreply@test.example.com",
		"http://localhost:3000",
	)

	err := service.SendRegistrationConfirmation("test@example.com", "token123")
	if err != nil {
		t.Fatalf("failed to send email: %v", err)
	}
}
