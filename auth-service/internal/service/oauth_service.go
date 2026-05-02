package service

import (
	apperrors "auth-service/internal/errors"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
)

// OAuthUserInfo stores normalized identity information from OAuth providers.
type OAuthUserInfo struct {
	Provider   string
	ProviderID string
	Email      string
}

// OAuthService handles provider-specific OAuth workflows.
type OAuthService struct {
	googleConfig   *oauth2.Config
	facebookConfig *oauth2.Config
	httpClient     *http.Client
}

// NewOAuthService creates an OAuth service for Google and Facebook.
func NewOAuthService(googleClientID, googleClientSecret, googleRedirectURL, facebookClientID, facebookClientSecret, facebookRedirectURL string) *OAuthService {
	service := &OAuthService{httpClient: http.DefaultClient}

	if googleClientID != "" && googleClientSecret != "" && googleRedirectURL != "" {
		service.googleConfig = &oauth2.Config{
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			RedirectURL:  googleRedirectURL,
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint:     google.Endpoint,
		}
	}

	if facebookClientID != "" && facebookClientSecret != "" && facebookRedirectURL != "" {
		service.facebookConfig = &oauth2.Config{
			ClientID:     facebookClientID,
			ClientSecret: facebookClientSecret,
			RedirectURL:  facebookRedirectURL,
			Scopes:       []string{"email"},
			Endpoint:     facebook.Endpoint,
		}
	}

	return service
}

// IsProviderSupported returns whether OAuth is configured for the provider.
func (s *OAuthService) IsProviderSupported(provider string) bool {
	switch strings.ToLower(provider) {
	case "google":
		return s.googleConfig != nil
	case "facebook":
		return s.facebookConfig != nil
	default:
		return false
	}
}

// AuthCodeURL builds an OAuth authorization URL for the provider.
func (s *OAuthService) AuthCodeURL(provider, state string) (string, error) {
	cfg, err := s.providerConfig(provider)
	if err != nil {
		return "", err
	}
	return cfg.AuthCodeURL(state), nil
}

// Authenticate exchanges the code and fetches user info.
func (s *OAuthService) Authenticate(provider, code string) (*OAuthUserInfo, error) {
	cfg, err := s.providerConfig(provider)
	if err != nil {
		return nil, apperrors.NewOAuthService("authenticate", "unsupported oauth provider", err)
	}

	token, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		return nil, apperrors.NewOAuthService("authenticate", "failed to exchange code for token", err)
	}

	switch strings.ToLower(provider) {
	case "google":
		return s.fetchGoogleUser(token.AccessToken)
	case "facebook":
		return s.fetchFacebookUser(token.AccessToken)
	default:
		return nil, apperrors.NewOAuthService("authenticate", "unsupported oauth provider", nil)
	}
}

func (s *OAuthService) providerConfig(provider string) (*oauth2.Config, error) {
	switch strings.ToLower(provider) {
	case "google":
		if s.googleConfig == nil {
			return nil, apperrors.NewOAuthService("providerConfig", "google oauth is not configured", nil)
		}
		return s.googleConfig, nil
	case "facebook":
		if s.facebookConfig == nil {
			return nil, apperrors.NewOAuthService("providerConfig", "facebook oauth is not configured", nil)
		}
		return s.facebookConfig, nil
	default:
		return nil, apperrors.NewOAuthService("providerConfig", "unsupported oauth provider", nil)
	}
}

func (s *OAuthService) fetchGoogleUser(accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, apperrors.NewOAuthService("fetchGoogleUser", "failed to create request", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, apperrors.NewOAuthService("fetchGoogleUser", "failed to fetch user info", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, apperrors.NewOAuthService("fetchGoogleUser", "google userinfo request failed", fmt.Errorf("status %d", resp.StatusCode))
	}

	var payload struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, apperrors.NewOAuthService("fetchGoogleUser", "failed to decode user info", err)
	}
	if payload.ID == "" {
		return nil, apperrors.NewOAuthService("fetchGoogleUser", "google user id not found", nil)
	}

	return &OAuthUserInfo{
		Provider:   "google",
		ProviderID: payload.ID,
		Email:      payload.Email,
	}, nil
}

func (s *OAuthService) fetchFacebookUser(accessToken string) (*OAuthUserInfo, error) {
	url := fmt.Sprintf("https://graph.facebook.com/me?fields=id,email&access_token=%s", accessToken)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facebook userinfo request failed with status %d", resp.StatusCode)
	}

	var payload struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.ID == "" {
		return nil, errors.New("facebook user id not found")
	}

	return &OAuthUserInfo{
		Provider:   "facebook",
		ProviderID: payload.ID,
		Email:      payload.Email,
	}, nil
}
