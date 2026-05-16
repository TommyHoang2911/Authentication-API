package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"time"

	"auth-service/internal/config"
	"auth-service/internal/service"

	pb "github.com/TommyHoang2911/Authentication-API/pb/notification" // Local proto package
	"google.golang.org/grpc"
)

// NotificationServer implements the NotificationService gRPC server
type NotificationServer struct {
	pb.UnimplementedNotificationServiceServer
	emailService service.EmailSender
	cfg          *config.App
}

// NewNotificationServer creates a new NotificationServer instance
func NewNotificationServer(emailService service.EmailSender, cfg *config.App) *NotificationServer {
	return &NotificationServer{
		emailService: emailService,
		cfg:          cfg,
	}
}

// SendEmail implements the SendEmail RPC method
func (s *NotificationServer) SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error) {
	// Validate request
	if req.ToEmail == "" {
		return &pb.SendEmailResponse{
			Success: false,
			Message: "recipient email is required",
		}, nil
	}

	if req.Subject == "" {
		return &pb.SendEmailResponse{
			Success: false,
			Message: "subject is required",
		}, nil
	}

	// Log incoming request
	log.Printf("Received email request: to=%s, subject=%s", req.ToEmail, req.Subject)

	// Send email with integrated SMTP/SendGrid logic
	if err := s.sendEmail(ctx, req.ToEmail, req.Subject, req.Body); err != nil {
		log.Printf("Error sending email to %s: %v", req.ToEmail, err)
		return &pb.SendEmailResponse{
			Success: false,
			Message: fmt.Sprintf("failed to send email: %v", err),
		}, nil
	}

	log.Printf("Email successfully sent to %s", req.ToEmail)
	return &pb.SendEmailResponse{
		Success: true,
		Message: "email sent successfully",
	}, nil
}

// sendEmail sends email using SMTP with proper configuration
func (s *NotificationServer) sendEmail(ctx context.Context, to, subject, body string) error {
	// Format email message with proper headers
	message := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.cfg.SMTP.From,
		to,
		subject,
		body,
	)

	// Setup SMTP authentication if credentials are provided
	var auth smtp.Auth
	if s.cfg.SMTP.User != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTP.User, s.cfg.SMTP.Pass, s.cfg.SMTP.Host)
	}

	// Prepare SMTP server address
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTP.Host, s.cfg.SMTP.Port)

	// Send email via SMTP
	smtpTimeout := time.Duration(s.cfg.SMTP.TimeoutSeconds) * time.Second

	// Use context with timeout for the email sending operation
	ctx, cancel := context.WithTimeout(ctx, smtpTimeout)
	defer cancel()

	// Send email
	if err := smtp.SendMail(addr, auth, s.cfg.SMTP.From, []string{to}, []byte(message)); err != nil {
		return fmt.Errorf("send email via SMTP: %w", err)
	}

	return nil
}

// StartServer starts the gRPC notification server on specified port
func StartServer(port string, emailService service.EmailSender, cfg *config.App) (*grpc.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	grpcServer := grpc.NewServer()
	notificationServer := NewNotificationServer(emailService, cfg)
	pb.RegisterNotificationServiceServer(grpcServer, notificationServer)

	log.Printf("Starting gRPC notification server on port %s", port)

	// Serve in a goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	return grpcServer, nil
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate SMTP configuration
	if cfg.SMTP.Host == "" || cfg.SMTP.Port == "" || cfg.SMTP.From == "" {
		log.Fatalf("SMTP configuration incomplete: Host=%s, Port=%s, From=%s",
			cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.From)
	}

	// Initialize email service with SMTP settings
	smtpTimeout := time.Duration(cfg.SMTP.TimeoutSeconds) * time.Second
	emailService := service.NewEmailServiceWithTimeout(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.User,
		cfg.SMTP.Pass,
		cfg.SMTP.From,
		cfg.BaseURL,
		smtpTimeout,
	)

	// Start gRPC server on port 50051
	grpcPort := "50051"
	if _, err := StartServer(grpcPort, emailService, cfg); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Notification service running. Press Ctrl+C to stop.")

	// Keep the server running
	select {}
}
