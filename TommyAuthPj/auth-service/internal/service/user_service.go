package service

import (
	"auth-service/internal/model"
	"auth-service/internal/repository"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// UserService handles user-related operations like registration, login, and user retrieval.
type UserService struct {
	repo         *repository.UserRepository
	emailService EmailSender
}

// NewUserService creates a new UserService instance.
func NewUserService(repo *repository.UserRepository, emailService EmailSender) *UserService {
	return &UserService{repo: repo, emailService: emailService}
}

// Register creates a new user record and hashes the password before storing.
func (s *UserService) Register(email string, password string) (*model.User, error) {
	// hash the password using bcrypt with default cost
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Generate confirmation token
	confirmationToken := generateConfirmationToken()
	confirmationTokenExpiry := time.Now().Add(24 * time.Hour) // 24 hours

	user := &model.User{
		Email:                   email,
		Password:                string(hashed),
		RefreshToken:            "",
		RefreshTokenExpiry:      time.Time{},
		EmailConfirmed:          false,
		ConfirmationToken:       &confirmationToken,
		ConfirmationTokenExpiry: &confirmationTokenExpiry,
		CreatedAt:               time.Now(),
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	if s.emailService != nil {
		appEnv := os.Getenv("APP_ENV")
		send_to := os.Getenv("SMTP_TO")
		if appEnv == "" {
			send_to = user.Email
		}
		if err := s.emailService.SendRegistrationConfirmation(send_to, confirmationToken); err != nil {
			// registration valid but email failed; log and continue
			// ideally use structured logger in real app
			// no return here to avoid blocking signup for email issues
			// if you prefer hard failure, return err
			// return nil, err
			fmt.Printf("warning: failed to send registration email to %s: %v\n", user.Email, err)
		}
	}

	return user, nil
}

// Login authenticates a user and returns the user if credentials are valid.
func (s *UserService) Login(email string, password string) (*model.User, error) {
	// fetch the user by email
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, err
	}

	// Check if email is confirmed
	if !user.EmailConfirmed {
		return nil, errors.New("email not confirmed")
	}

	// verify password against stored hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

// GetOrCreateOAuthUser resolves an OAuth identity to a local user account.
func (s *UserService) GetOrCreateOAuthUser(email, provider, providerID string) (*model.User, error) {
	user, err := s.repo.FindByOAuth(provider, providerID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Link to existing account by email when possible.
	if email != "" {
		existing, emailErr := s.repo.FindByEmail(email)
		if emailErr == nil {
			if existing.OAuthProvider != nil && existing.OAuthProviderID != nil {
				if *existing.OAuthProvider != provider || *existing.OAuthProviderID != providerID {
					return nil, errors.New("email already linked to another OAuth account")
				}
				return existing, nil
			}

			if err := s.repo.LinkOAuthProvider(existing.ID, provider, providerID); err != nil {
				return nil, err
			}
			linked, err := s.repo.FindByID(existing.ID)
			if err != nil {
				return nil, err
			}
			return linked, nil
		}
		if !errors.Is(emailErr, sql.ErrNoRows) {
			return nil, emailErr
		}
	}

	placeholderPassword := generateConfirmationToken()
	hashed, err := bcrypt.GenerateFromPassword([]byte(placeholderPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	providerCopy := provider
	providerIDCopy := providerID

	newUser := &model.User{
		Email:                   email,
		Password:                string(hashed),
		OAuthProvider:           &providerCopy,
		OAuthProviderID:         &providerIDCopy,
		RefreshToken:            "",
		RefreshTokenExpiry:      time.Time{},
		EmailConfirmed:          true,
		ConfirmationToken:       nil,
		ConfirmationTokenExpiry: nil,
		CreatedAt:               time.Now(),
	}

	if err := s.repo.Create(newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

// GetUserByID retrieves a user by ID, omitting sensitive fields.
func (s *UserService) GetUserByID(id int64) (*model.User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	// Omit password and refresh token
	user.Password = ""
	user.RefreshToken = ""
	user.RefreshTokenExpiry = time.Time{}
	return user, nil
}

// CreateRefreshToken creates a new refresh token for the user.
func (s *UserService) CreateRefreshToken(userID int64, refreshToken string, expiry time.Time) error {
	return s.repo.CreateRefreshToken(userID, refreshToken, expiry)
}

// FindRefreshToken finds the user ID associated with a refresh token.
func (s *UserService) FindRefreshToken(refreshToken string) (int64, error) {
	return s.repo.FindRefreshToken(refreshToken)
}

// DeleteRefreshToken invalidates a refresh token.
func (s *UserService) DeleteRefreshToken(refreshToken string) error {
	return s.repo.DeleteRefreshToken(refreshToken)
}

// ConfirmEmail confirms a user's email using a confirmation token.
func (s *UserService) ConfirmEmail(token string) error {
	user, err := s.repo.FindByConfirmationToken(token)
	if err != nil {
		return errors.New("invalid or expired confirmation token")
	}

	if user.EmailConfirmed {
		return errors.New("email already confirmed")
	}

	return s.repo.ConfirmEmail(user.ID)
}

// ResendConfirmationEmail resends the confirmation email to a user.
func (s *UserService) ResendConfirmationEmail(email string) error {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return errors.New("user not found")
	}

	if user.EmailConfirmed {
		return errors.New("email already confirmed")
	}

	if user.ConfirmationToken == nil {
		return errors.New("no confirmation token found")
	}

	if s.emailService != nil {
		appEnv := os.Getenv("APP_ENV")
		send_to := os.Getenv("SMTP_TO")
		if appEnv == "" {
			send_to = user.Email
		}
		if err := s.emailService.SendRegistrationConfirmation(send_to, *user.ConfirmationToken); err != nil {
			return err
		}
	}

	return nil
}

// generateConfirmationToken creates a cryptographically secure confirmation token
func generateConfirmationToken() string {
	bytes := make([]byte, 32) // 32 bytes = 64 hex characters
	if _, err := rand.Read(bytes); err != nil {
		panic(err) // This should never happen in practice
	}
	return hex.EncodeToString(bytes)
}
