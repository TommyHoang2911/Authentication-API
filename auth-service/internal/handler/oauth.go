package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"

	customErrors "auth-service/internal/errors"

	"github.com/gin-gonic/gin"
)

// OAuthLogin godoc
// @Summary Start OAuth login
// @Description Redirect the user to the selected OAuth provider login page.
// @Tags OAuth
// @Param provider path string true "OAuth provider" Enums(google,facebook)
// @Success 307 {string} string "Temporary Redirect"
// @Header 307 {string} Location "Redirect URL to the OAuth provider's login page"
// @Failure 400 {object} ErrorResponse
// @Router /auth/{provider}/login [get]
// OAuthLogin redirects user to provider consent page.
func (h *AuthHandler) OAuthLogin(c *gin.Context) {
	provider := c.Param("provider")
	state := generateOAuthState(c)

	url, err := h.authService.OAuthLoginURL(provider, state)
	if err != nil {
		log.Printf("OAuthLogin: failed to build auth url: %v", err)
		RespondWithError(c, http.StatusBadRequest, err)
		return
	}
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)
	secure := c.Request.TLS != nil
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/oauth",
		MaxAge:   600,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// OAuthCallback godoc
// @Summary Complete OAuth login
// @Description Exchange provider code for application tokens.
// @Tags OAuth
// @Produce json
// @Param provider path string true "OAuth provider" Enums(google,facebook)
// @Param code query string true "Authorization code"
// @Param state query string true "OAuth state"
// @Success 200 {object} OAuthLoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/{provider}/callback [get]
// OAuthCallback handles provider callback, exchanges code, and returns tokens.
func (h *AuthHandler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		RespondWithError(c, http.StatusBadRequest, customErrors.NewAPI("validation", "missing oauth code or state", nil))
		return
	}

	savedState, err := c.Cookie("oauth_state")
	if err != nil || savedState != state {
		// Expire oauth_state cookie to prevent state replay even on failure.
		c.SetCookie("oauth_state", "", -1, "/", "", false, true)
		RespondWithError(c, http.StatusBadRequest, customErrors.NewAPI("validation", "invalid oauth state", nil))
		return
	}
	// State has been successfully validated; expire oauth_state cookie to prevent reuse.
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	user, token, refreshToken, err := h.authService.OAuthCallback(provider, code)
	if err != nil {
		log.Printf("OAuthCallback: authentication failed: %v", err)
		RespondWithError(c, http.StatusUnauthorized, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "oauth login successful",
		"user":          user,
		"token":         token,
		"refresh_token": refreshToken,
	})
}

func generateOAuthState(c *gin.Context) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Printf("OAuthLogin: Generate state failed: %v", err)
		RespondWithError(c, http.StatusInternalServerError, customErrors.NewSystem("oauth_state", "failed to generate oauth state", err))
		return ""
	}
	return hex.EncodeToString(b)
}
