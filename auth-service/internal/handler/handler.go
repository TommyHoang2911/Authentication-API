package handler

// AuthHandler coordinates HTTP request handling for authentication and QR code operations.
// It delegates business logic to the AuthService.
type AuthHandler struct {
	authService AuthServiceInterface
}

// NewAuthHandler constructs an AuthHandler with the provided service.
func NewAuthHandler(authService AuthServiceInterface) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}
