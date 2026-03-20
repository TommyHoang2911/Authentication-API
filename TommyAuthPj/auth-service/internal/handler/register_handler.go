package handler

import (
	"log"
	"net/http"

	"auth-service/internal/model"

	"github.com/gin-gonic/gin"
)

// AuthServiceInterface defines the methods needed by the handler
type AuthServiceInterface interface {
	Register(email, password string) (*model.User, error)
	Login(email, password string) (*model.User, string, string, error)
	GetUserByID(id int64) (*model.User, error)
	RefreshToken(refreshToken string) (string, error)
	SignOut(userID int64) error
}

type AuthHandler struct {
	authService AuthServiceInterface
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func NewAuthHandler(authService AuthServiceInterface) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {

	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Register: invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.Register(req.Email, req.Password)

	if err != nil {
		log.Printf("Register: service error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user registered successfully", "user": user})
}

func (h *AuthHandler) Login(c *gin.Context) {

	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Login: invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, refreshToken, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		log.Printf("Login: authentication failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "login successful", "user": user, "token": token, "refresh_token": refreshToken})
}

func (h *AuthHandler) GetUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		log.Printf("GetUser: user not authenticated (missing user_id in context)")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	user, err := h.authService.GetUserByID(userID.(int64))
	if err != nil {
		log.Printf("GetUser: failed to retrieve user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("RefreshToken: invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		log.Printf("RefreshToken: failed to refresh token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *AuthHandler) SignOut(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		log.Printf("SignOut: user not authenticated (missing user_id in context)")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	if err := h.authService.SignOut(userID.(int64)); err != nil {
		log.Printf("SignOut: failed to sign out: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign out"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "signed out successfully"})
}
