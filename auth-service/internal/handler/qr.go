package handler

import (
	"log"
	"net/http"

	customErrors "auth-service/internal/errors"

	"github.com/gin-gonic/gin"
)

// GenerateQR godoc
// @Summary Generate QR auth code
// @Description Generate a QR-based authentication code for a device.
// @Tags QR
// @Accept json
// @Produce json
// @Param request body GenerateQRRequest true "Device payload"
// @Success 200 {object} QRCodeResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /generate_qr [post]
// GenerateQR generates a QR code for device-based authentication
func (h *AuthHandler) GenerateQR(c *gin.Context) {
	var req GenerateQRRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("GenerateQR: invalid request: %v", err)
		RespondWithError(c, http.StatusBadRequest, customErrors.NewAPI("validation", "invalid request", err))
		return
	}

	code, err := h.authService.GenerateQRCode(req.DeviceID)
	if err != nil {
		log.Printf("GenerateQR: failed to generate code: %v", err)
		RespondWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": code})
}

// VerifyQR godoc
// @Summary Verify QR auth code
// @Description Verify a scanned QR code for the authenticated user.
// @Tags QR
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body VerifyQRRequest true "Verification payload"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /verify_qr [post]
// VerifyQR verifies a QR code scanned by the authenticated user
func (h *AuthHandler) VerifyQR(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		log.Printf("VerifyQR: user not authenticated")
		RespondWithError(c, http.StatusUnauthorized, customErrors.NewAPI("auth", "user not authenticated", nil))
		return
	}

	var req VerifyQRRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("VerifyQR: invalid request: %v", err)
		RespondWithError(c, http.StatusBadRequest, customErrors.NewAPI("validation", "invalid request", err))
		return
	}

	err := h.authService.VerifyQRCode(req.Code, userID.(int64))
	if err != nil {
		log.Printf("VerifyQR: failed to verify code: %v", err)
		RespondWithError(c, http.StatusUnauthorized, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "QR code verified successfully"})
}

// ExchangeCode godoc
// @Summary Exchange temporary code
// @Description Exchange a temporary QR authentication code for access tokens.
// @Tags QR
// @Accept json
// @Produce json
// @Param request body ExchangeCodeRequest true "Exchange payload"
// @Success 200 {object} ExchangeCodeResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /exchange_code [post]
// ExchangeCode exchanges a temporary auth code for JWT tokens
func (h *AuthHandler) ExchangeCode(c *gin.Context) {
	var req ExchangeCodeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ExchangeCode: invalid request: %v", err)
		RespondWithError(c, http.StatusBadRequest, customErrors.NewAPI("validation", "invalid request", err))
		return
	}

	user, token, sessionToken, err := h.authService.ExchangeCode(req.TempCode)
	if err != nil {
		log.Printf("ExchangeCode: failed to exchange code: %v", err)
		RespondWithError(c, http.StatusUnauthorized, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "code exchanged successfully",
		"user":          user,
		"token":         token,
		"session_token": sessionToken,
	})
}
