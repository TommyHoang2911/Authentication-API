package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Register handles user registration
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

// Login handles user authentication and token generation
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

	c.JSON(http.StatusOK, gin.H{
		"message":       "login successful",
		"user":          user,
		"token":         token,
		"refresh_token": refreshToken,
	})
}

// GetUser retrieves the current authenticated user's information
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

// RefreshToken validates a refresh token and returns a new access token
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

// SignOut invalidates the user's refresh token
func (h *AuthHandler) SignOut(c *gin.Context) {
	var req SignOutRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("SignOut: invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.SignOut(req.RefreshToken); err != nil {
		log.Printf("SignOut: failed to sign out: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign out"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "signed out successfully"})
}

// ConfirmEmail confirms a user's email using a confirmation token
func (h *AuthHandler) ConfirmEmail(c *gin.Context) {
	var req ConfirmEmailRequest
	token := c.Query("token")

	if token == "" {
		log.Printf("ConfirmEmail: missing confirmation token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing confirmation token"})
		return
	}
	req.Token = token

	if err := h.authService.ConfirmEmail(req.Token); err != nil {
		log.Printf("ConfirmEmail: failed to confirm email: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "email confirmed successfully"})
}

// ResendConfirmationEmail resends the confirmation email to a user
func (h *AuthHandler) ResendConfirmationEmail(c *gin.Context) {
	var req ResendConfirmationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ResendConfirmationEmail: invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.ResendConfirmationEmail(req.Email); err != nil {
		log.Printf("ResendConfirmationEmail: failed to resend confirmation email: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "confirmation email sent successfully"})
}
