package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api/model"
	"glt-calendar-service/utils"
	"io"
	"net/http"
	"time"
)

// TokenManager handles Google OAuth token operations
type TokenManager struct {
	client *http.Client
}

// NewTokenManager creates a new TokenManager instance
func NewTokenManager() *TokenManager {
	return &TokenManager{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetTokenResponse retrieves a token either from session or by exchanging auth code
// Returns token response and error if any
func (tm *TokenManager) GetTokenResponse(context *gin.Context) (*model.GoogleTokenResponse, error) {
	// First check if a token exists in session
	session, _ := sessionManager.GetContextOrSession(context)
	if session != nil && session.Data != nil && session.Data.TokenResponse != nil {
		if session.Data.TokenResponse.AccessToken != "" {
			// If a token exists and might be expired, refresh it first
			refreshedSession, err := tm.EnsureValidToken(context, session)
			if err != nil {
				return nil, fmt.Errorf("failed to refresh existing token: %w", err)
			}
			return refreshedSession.Data.TokenResponse, nil
		}
	}

	// No token in session, get from request
	var req model.GoogleTokenRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		return nil, fmt.Errorf("invalid request format: %w", err)
	}

	// Exchange authorization code for token
	tokenResp, err := tm.exchangeCodeForToken(req.Code, req.RedirectUri)
	if err != nil {
		return nil, err
	}

	return tokenResp, nil
}

// GetAccessToken retrieves a valid access token, refreshing if necessary
// Returns the access token string and error if any
func (tm *TokenManager) GetAccessToken(context *gin.Context) (string, error) {
	session, err := sessionManager.GetContextOrSession(context)
	if err != nil {
		return "", err
	}

	refreshedSession, err := tm.EnsureValidToken(context, session)
	if err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	accessToken := refreshedSession.Data.TokenResponse.AccessToken
	if accessToken == "" {
		return "", fmt.Errorf("access token not found in session")
	}
	return accessToken, nil
}

// EnsureValidToken checks if the token is valid and refreshes if needed
// Returns updated session and error if any
func (tm *TokenManager) EnsureValidToken(context *gin.Context, session *model.Session) (*model.Session, error) {
	if !session.IsTokenExpired() {
		return session, nil
	}

	logger.Info("Access token is expired or about to expire, refreshing...")

	refreshToken := session.Data.TokenResponse.RefreshToken
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token not found in session")
	}

	newToken, err := tm.refreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update the session with new token info
	session.Data.TokenResponse.AccessToken = newToken.AccessToken
	session.Data.TokenResponse.ExpiresIn = newToken.ExpiresIn
	session.Data.TokenResponse.TokenType = newToken.TokenType
	session.Data.TokenResponse.CreatedAt = utils.GetCurrentTime()
	session.UpdateDate = utils.GetCurrentTime()

	// Preserve refresh token if new response doesn't include one
	if newToken.RefreshToken != "" {
		session.Data.TokenResponse.RefreshToken = newToken.RefreshToken
	}

	// Save an updated session
	if err := sessionManager.UpdateSession(session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Update cookie
	sessionManager.SetCookie(context, &model.Cookie{
		Name:     "session_id",
		Value:    session.SessionID,
		MaxAge:   24 * 60 * 60,
		Path:     "/",
		Domain:   "",
		Secure:   false,
		HttpOnly: true,
	})

	// Get an updated session
	updatedSession, err := sessionManager.GetContextOrSession(context)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated session: %w", err)
	}

	return updatedSession, nil
}

// exchangeCodeForToken exchanges authorization code for token
// Returns token response and error if any
func (tm *TokenManager) exchangeCodeForToken(code, redirectUri string) (*model.GoogleTokenResponse, error) {
	request := map[string]string{
		"code":          code,
		"client_id":     cfg.GoogleOAuth2.ClientID,
		"client_secret": cfg.GoogleOAuth2.ClientSecret,
		"redirect_uri":  redirectUri,
		"grant_type":    "authorization_code",
	}

	return tm.sendTokenRequest(model.GoogleOAuth2TokenUrl, request)
}

// refreshToken refreshes an access token using refresh token
// Returns token response and error if any
func (tm *TokenManager) refreshToken(refreshToken string) (*model.GoogleTokenResponse, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	data := map[string]string{
		"client_id":     cfg.GoogleOAuth2.ClientID,
		"client_secret": cfg.GoogleOAuth2.ClientSecret,
		"refresh_token": refreshToken,
		"grant_type":    "refresh_token",
	}

	return tm.sendTokenRequest(model.GoogleOAuth2RefreshTokenUrl, data)
}

// sendTokenRequest sends a token request to specified URL
// Returns token response and error if any
func (tm *TokenManager) sendTokenRequest(url string, data map[string]string) (*model.GoogleTokenResponse, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request data: %w", err)
	}

	resp, err := tm.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send token request: %w", err)
	}

	defer utils.CloseResponseBody(resp, "TokenRequest")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}

		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("token request failed: %s - %s", errorResp.Error, errorResp.Description)
	}

	var tokenResponse model.GoogleTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResponse.AccessToken == "" {
		return nil, fmt.Errorf("received empty access token in response")
	}

	tokenResponse.CreatedAt = utils.GetCurrentTime()
	return &tokenResponse, nil
}
