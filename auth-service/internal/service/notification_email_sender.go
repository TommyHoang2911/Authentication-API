package service

import (
	"context"
	"fmt"
	"time"

	pb "github.com/TommyHoang2911/Authentication-API/pb/notification"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NotificationEmailSender sends registration confirmation emails through the Notification Service.
type NotificationEmailSender struct {
	client  pb.NotificationServiceClient
	conn    *grpc.ClientConn
	baseURL string
}

// NewNotificationEmailSender creates a single shared gRPC client connection to the Notification Service.
func NewNotificationEmailSender(address string, baseURL string) (*NotificationEmailSender, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial notification service at %s: %w", address, err)
	}

	client := pb.NewNotificationServiceClient(conn)
	return &NotificationEmailSender{client: client, conn: conn, baseURL: baseURL}, nil
}

// Close shuts down the gRPC connection.
func (s *NotificationEmailSender) Close() error {
	if s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

// SendRegistrationConfirmation sends a confirmation email by calling the Notification Service.
func (s *NotificationEmailSender) SendRegistrationConfirmation(to, confirmationToken string) error {
	confirmationURL := fmt.Sprintf("%s/confirm_email?token=%s", s.baseURL, confirmationToken)
	subject := "Please confirm your email address"
	body := fmt.Sprintf(`Welcome! Thanks for registering.

Please click the link below to confirm your email address:
%s

This link will expire in 24 hours.

If you didn't create an account, please ignore this email.`, confirmationURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.client.SendEmail(ctx, &pb.SendEmailRequest{
		ToEmail: to,
		Subject: subject,
		Body:    body,
	})
	if err != nil {
		return fmt.Errorf("notification service send email: %w", err)
	}

	return nil
}
