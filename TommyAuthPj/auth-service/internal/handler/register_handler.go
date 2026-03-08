package handler

import (
	"net/http"

	"auth-service/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: service.NewAuthService(),
	}
}

func (h *AuthHandler) Register(c *gin.Context) {

	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.Register(req.Email, req.Password)

	if err != nil {
		c.JSON(500, gin.H{"error": "register failed"})
		return
	}

	c.JSON(200, user)
}
