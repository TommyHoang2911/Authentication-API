package service

import (
	"auth-service/internal/model"
	"auth-service/internal/repository"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService coordinates business logic related to authentication and
// user management. It depends on a repository for persistence.
type AuthService struct {
	repo *repository.UserRepository
}

const jwtSecret = "your-secret-key" // TODO: move to environment variable

// NewAuthService constructs an AuthService with the provided repository.
func NewAuthService(repo *repository.UserRepository) *AuthService {
	return &AuthService{repo: repo}
}

// Register creates a new user record and hashes the password before storing.
func (s *AuthService) Register(email string, password string) (*model.User, error) {
	// hash the password using bcrypt with default cost
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:              email,
		Password:           string(hashed),
		RefreshToken:       "",
		RefreshTokenExpiry: time.Time{},
		CreatedAt:          time.Now(),
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(email string, password string) (*model.User, string, string, error) {
	// fetch the user by email
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, "", "", err
	}

	// verify password against stored hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}

	// generate JWT token
	claims := jwt.MapClaims{
		"email": user.Email,
		"id":    user.ID,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, "", "", err
	}

	// generate refresh token
	refreshToken := generateRefreshToken()
	refreshTokenExpiry := time.Now().Add(time.Hour * 24 * 7) // 7 days

	// check if refresh token already exists for this user
	existingToken, err := s.repo.FindRefreshTokenByUserID(user.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, "", "", err
	}

	// only create new refresh token if one doesn't exist
	if existingToken == "" {
		if err := s.repo.CreateRefreshToken(user.ID, refreshToken, refreshTokenExpiry); err != nil {
			return nil, "", "", err
		}
	} else {
		refreshToken = existingToken
	}

	return user, tokenString, refreshToken, nil
}

// GetUserByID retrieves a user by ID, omitting sensitive fields.
func (s *AuthService) GetUserByID(id int64) (*model.User, error) {
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

// RefreshToken validates a refresh token and returns a new JWT token.
func (s *AuthService) RefreshToken(refreshToken string) (string, error) {
	// Find user associated with the refresh token
	userID, err := s.repo.FindRefreshToken(refreshToken)
	if err != nil {
		return "", errors.New("invalid or expired refresh token")
	}

	// Get user details
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return "", errors.New("user not found")
	}

	// Generate new JWT token
	claims := jwt.MapClaims{
		"email": user.Email,
		"id":    user.ID,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// SignOut invalidates the user's refresh tokens.
func (s *AuthService) SignOut(userID int64) error {
	return s.repo.DeleteRefreshTokensByUserID(userID)
}

// generateRefreshToken creates a cryptographically secure random refresh token
func generateRefreshToken() string {
	bytes := make([]byte, 32) // 32 bytes = 64 hex characters
	if _, err := rand.Read(bytes); err != nil {
		panic(err) // This should never happen in practice
	}
	return hex.EncodeToString(bytes)
}
