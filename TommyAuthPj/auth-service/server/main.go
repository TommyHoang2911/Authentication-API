package main

import (
	"auth-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func main() {

	r := gin.Default()

	authHandler := handler.NewAuthHandler()

	r.POST("/register", authHandler.Register)

	r.Run(":8080")
}
