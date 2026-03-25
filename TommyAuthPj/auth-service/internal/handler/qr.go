package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GenerateQR generates a QR code for device-based authentication
func (h *AuthHandler) GenerateQR(c *gin.Context) {
	var req GenerateQRRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("GenerateQR: invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	code, err := h.authService.GenerateQRCode(req.DeviceID)
	if err != nil {
		log.Printf("GenerateQR: failed to generate code: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate QR code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": code})
}

// VerifyQR verifies a QR code scanned by the authenticated user
func (h *AuthHandler) VerifyQR(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		log.Printf("VerifyQR: user not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req VerifyQRRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("VerifyQR: invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.authService.VerifyQRCode(req.Code, userID.(int64))
	if err != nil {
		log.Printf("VerifyQR: failed to verify code: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "QR code verified successfully"})
}

// ExchangeCode exchanges a temporary auth code for JWT tokens
func (h *AuthHandler) ExchangeCode(c *gin.Context) {
	var req ExchangeCodeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ExchangeCode: invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, sessionToken, err := h.authService.ExchangeCode(req.TempCode)
	if err != nil {
		log.Printf("ExchangeCode: failed to exchange code: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "code exchanged successfully",
		"user":          user,
		"token":         token,
		"session_token": sessionToken,
	})
}
