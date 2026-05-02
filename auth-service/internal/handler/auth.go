package handler

import (
	"log"
	"net/http"

	customErrors "auth-service/internal/errors"

	"github.com/gin-gonic/gin"
)

// Register godoc
// @Summary Register a new user
// @Description Create a new user account with email and password.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration payload"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} ErrorResponse
// @Router /register [post]
// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Register: invalid request: %v", err)
		RespondWithError(c, http.StatusBadRequest, customErrors.NewAPI("validation", "invalid request", err))
		return
	}

	user, err := h.authService.Register(req.Email, req.Password)
	if err != nil {
		log.Printf("Register: service error: %v", err)
		RespondWithError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user registered successfully", "user": user})
}

// Login godoc
// @Summary Login with email and password
// @Description Authenticate a confirmed user and return access and refresh tokens.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login payload"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /login [post]
// Login handles user authentication and token generation
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Login: invalid request: %v", err)
		RespondWithError(c, http.StatusBadRequest, customErrors.NewAPI("validation", "invalid request", err))
		return
	}

	user, token, refreshToken, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		log.Printf("Login error: %v", err)
		RespondWithError(c, http.StatusUnauthorized, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "login successful",
		"user":          user,
		"token":         token,
		"refresh_token": refreshToken,
	})
}

// GetUser godoc
// @Summary Get current user
// @Description Retrieve the authenticated user profile.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserEnvelope
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user [get]
// GetUser retrieves the current authenticated user's information
func (h *AuthHandler) GetUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		log.Printf("GetUser: user not authenticated (missing user_id in context)")
		RespondWithError(c, http.StatusUnauthorized, customErrors.NewAPI("auth", "user not authenticated", nil))
		return
	}

	user, err := h.authService.GetUserByID(userID.(int64))
	if err != nil {
		log.Printf("GetUser: failed to retrieve user: %v", err)
		RespondWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Validate a refresh token and return a new access token.
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RefreshTokenRequest true "Refresh token payload"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /refresh_token [post]
// RefreshToken validates a refresh token and returns a new access token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("RefreshToken: invalid request: %v", err)
		RespondWithError(c, http.StatusBadRequest, customErrors.NewAPI("validation", "invalid request", err))
		return
	}

	token, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		log.Printf("RefreshToken: failed to refresh token: %v", err)
		RespondWithError(c, http.StatusUnauthorized, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// SignOut godoc
// @Summary Sign out user
// @Description Invalidate a user's refresh token.
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SignOutRequest true "Sign out payload"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sign_out [post]
// SignOut invalidates the user's refresh token
func (h *AuthHandler) SignOut(c *gin.Context) {
	var req SignOutRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("SignOut: invalid request: %v", err)
		RespondWithError(c, http.StatusBadRequest, customErrors.NewAPI("validation", "invalid request", err))
		return
	}

	if err := h.authService.SignOut(req.RefreshToken); err != nil {
		log.Printf("SignOut: failed to sign out: %v", err)
		RespondWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "signed out successfully"})
}

// ConfirmEmail godoc
// @Summary Confirm user email
// @Description Confirm a user account using the email confirmation token.
// @Tags Auth
// @Produce json
// @Param token query string true "Confirmation token"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Router /confirm_email [get]
// ConfirmEmail confirms a user's email using a confirmation token
func (h *AuthHandler) ConfirmEmail(c *gin.Context) {
	var req ConfirmEmailRequest
	token := c.Query("token")

	if token == "" {
		log.Printf("ConfirmEmail: missing confirmation token")
		RespondWithError(c, http.StatusBadRequest, customErrors.NewAPI("validation", "missing confirmation token", nil))
		return
	}
	req.Token = token

	if err := h.authService.ConfirmEmail(req.Token); err != nil {
		log.Printf("ConfirmEmail: failed to confirm email: %v", err)
		RespondWithError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "email confirmed successfully"})
}

// ResendConfirmationEmail godoc
// @Summary Resend confirmation email
// @Description Resend the email confirmation message to a registered user.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body ResendConfirmationRequest true "Resend confirmation payload"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Router /resend_confirmation [post]
// ResendConfirmationEmail resends the confirmation email to a user
func (h *AuthHandler) ResendConfirmationEmail(c *gin.Context) {
	var req ResendConfirmationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ResendConfirmationEmail: invalid request: %v", err)
		RespondWithError(c, http.StatusBadRequest, customErrors.NewAPI("validation", "invalid request", err))
		return
	}

	if err := h.authService.ResendConfirmationEmail(req.Email); err != nil {
		log.Printf("ResendConfirmationEmail: failed to resend confirmation email: %v", err)
		RespondWithError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "confirmation email sent successfully"})
}
