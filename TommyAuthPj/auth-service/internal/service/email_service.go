package service

import (
	"fmt"
	"net/smtp"
)

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
}

// NewEmailService creates a new EmailService instance.
func NewEmailService(host, port, username, password, from, baseURL string) *EmailService {
	return &EmailService{
		smtpHost: host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		baseURL:  baseURL,
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
	auth := smtp.PlainAuth("", s.username, s.password, s.smtpHost)

	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(message))
}
