package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Liveness Check: Is the application running?
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"message": "Auth service is alive",
	})
}

// Readiness Check: Is the application ready to handle requests?
func (h *HealthHandler) Readiness(c *gin.Context) {
	if err := h.db.Ping(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "DOWN",
			"message": "Database ping error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"message": "Auth service is ready",
	})
}
