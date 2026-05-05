package service

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

const defaultSMTPTimeout = 10 * time.Second

// EmailSender represents an object that can send registration emails.
type EmailSender interface {
	SendRegistrationConfirmation(to, confirmationToken string) error
}

// EmailService sends outgoing email messages using SMTP.
type EmailService struct {
	smtpHost string
	port     string
	username string
	password string
	from     string
	baseURL  string
	timeout  time.Duration
}

// NewEmailService creates a new EmailService instance.
func NewEmailService(host, port, username, password, from, baseURL string) *EmailService {
	return NewEmailServiceWithTimeout(host, port, username, password, from, baseURL, defaultSMTPTimeout)
}

// NewEmailServiceWithTimeout creates a new EmailService instance with an explicit SMTP timeout.
func NewEmailServiceWithTimeout(host, port, username, password, from, baseURL string, timeout time.Duration) *EmailService {
	if timeout <= 0 {
		timeout = defaultSMTPTimeout
	}

	return &EmailService{
		smtpHost: host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		baseURL:  baseURL,
		timeout:  timeout,
	}
}

// SendRegistrationConfirmation sends the registration confirmation email with a confirmation link.
func (s *EmailService) SendRegistrationConfirmation(to, confirmationToken string) error {
	confirmationURL := fmt.Sprintf("%s/confirm_email?token=%s", s.baseURL, confirmationToken)

	subject := "Please confirm your email address"
	body := fmt.Sprintf(`Welcome! Thanks for registering.

Please click the link below to confirm your email address:
%s

This link will expire in 24 hours.

If you didn't create an account, please ignore this email.`, confirmationURL)

	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", s.from, to, subject, body)

	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.port)
	var auth smtp.Auth
	if s.username != "" || s.password != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.smtpHost)
	}

	return s.sendMail(addr, auth, s.from, []string{to}, []byte(message))
}

func (s *EmailService) sendMail(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	dialer := net.Dialer{Timeout: s.timeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial smtp server: %w", err)
	}

	if err := conn.SetDeadline(time.Now().Add(s.timeout)); err != nil {
		conn.Close()
		return fmt.Errorf("set smtp deadline: %w", err)
	}

	client, err := smtp.NewClient(conn, s.smtpHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil && !isLoopbackSMTPHost(s.smtpHost) {
		ok, _ := client.Extension("STARTTLS")
		if !ok {
			return fmt.Errorf("smtp server %q does not advertise STARTTLS; refusing to authenticate over plaintext connection", s.smtpHost)
		}
		config := &tls.Config{ServerName: s.smtpHost}
		if err = client.StartTLS(config); err != nil {
			return fmt.Errorf("start TLS: %w", err)
		}

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate smtp client: %w", err)
		}
	}

	fromAddr, err := mail.ParseAddress(from)
	cleanFrom := from
	if err == nil {
		cleanFrom = fromAddr.Address
	} else {
		cleanFrom = strings.TrimSpace(from)
	}

	if err := client.Mail(cleanFrom); err != nil {
		return fmt.Errorf("set smtp sender: %w", err)
	}
	
	// In ra chuỗi hiển thị bình thường
	log.Printf("[DEBUG-EMAIL-RCPT] Original string: '%s'", to[0])

	// QUAN TRỌNG NHẤT: In ra mã Hex (hệ cơ số 16) của từng byte trong chuỗi
	log.Printf("[DEBUG-EMAIL-HEX] Hex format: %x", to[0])
	// 2. Xử lý Envelope Recipient: Làm sạch từng email nhận
	for _, recipient := range to {
		toAddr, err := mail.ParseAddress(recipient)
		cleanTo := recipient
		if err == nil {
			cleanTo = toAddr.Address
		} else {
			cleanTo = strings.TrimSpace(recipient)
		}

		if err := client.Rcpt(cleanTo); err != nil {
			return fmt.Errorf("set smtp recipient: %w", err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start smtp data: %w", err)
	}

	if _, err := writer.Write(msg); err != nil {
		writer.Close()
		return fmt.Errorf("write smtp message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close smtp message writer: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("quit smtp client: %w", err)
	}

	return nil
}

func isLoopbackSMTPHost(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
