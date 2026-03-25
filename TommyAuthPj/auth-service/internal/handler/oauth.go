package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// OAuthLogin redirects user to provider consent page.
func (h *AuthHandler) OAuthLogin(c *gin.Context) {
	provider := c.Param("provider")
	state := generateOAuthState(c)

	url, err := h.authService.OAuthLoginURL(provider, state)
	if err != nil {
		log.Printf("OAuthLogin: failed to build auth url: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// c.SetCookie("oauth_state", state, 600, "/", "", false, true)
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

// OAuthCallback handles provider callback, exchanges code, and returns tokens.
func (h *AuthHandler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing oauth code or state"})
		return
	}

	savedState, err := c.Cookie("oauth_state")
	if err != nil || savedState != state {
		// Expire oauth_state cookie to prevent state replay even on failure.
		c.SetCookie("oauth_state", "", -1, "/", "", false, true)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
		return
	}
	// State has been successfully validated; expire oauth_state cookie to prevent reuse.
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	user, token, refreshToken, err := h.authService.OAuthCallback(provider, code)
	if err != nil {
		log.Printf("OAuthCallback: authentication failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return ""
	}
	return hex.EncodeToString(b)
}
