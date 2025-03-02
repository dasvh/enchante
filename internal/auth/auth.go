package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/dasvh/enchante/internal/config"
)

// GetAuthHeader returns the header and value for the authentication method specified in the authConfig
func GetAuthHeader(authConfig *config.AuthConfig, logger *slog.Logger) (string, string, error) {
	if !authConfig.Enabled {
		logger.Info("Authentication is disabled")
		return "", "", nil
	}
	switch authConfig.Type {
	case "api_key":
		logger.Info("Using API Key authentication")
		return authConfig.APIKey.Header, authConfig.APIKey.Value, nil
	case "basic":
		logger.Info("Using Basic authentication")
		encoded := base64.StdEncoding.EncodeToString([]byte(authConfig.Basic.Username + ":" + authConfig.Basic.Password))
		return "Authorization", "Basic " + encoded, nil
	case "oauth2":
		logger.Info("Using OAuth2 authentication")
		token, err := getOAuthToken(authConfig.OAuth2, logger)
		if err != nil {
			logger.Error("Failed to fetch OAuth token", "error", err)
			return "", "", err
		}
		return "Authorization", "Bearer " + token, nil
	default:
		logger.Error("Unsupported authentication type", "auth_type", authConfig.Type)
		return "", "", fmt.Errorf("unsupported auth type: %s", authConfig.Type)
	}
}

// getOAuthToken retrieves an OAuth2 token using the provided configuration
func getOAuthToken(auth config.OAuth2Auth, logger *slog.Logger) (string, error) {
	logger.Debug("Requesting OAuth2 token", "url", auth.TokenURL, "client_id", auth.ClientID)
	data := fmt.Sprintf("client_id=%s&client_secret=%s&username=%s&password=%s&grant_type=%s&scope=%s",
		auth.ClientID, auth.ClientSecret, auth.Username, auth.Password, auth.GrantType, auth.Scope)

	req, err := http.NewRequest("POST", auth.TokenURL, bytes.NewBufferString(data))
	if err != nil {
		logger.Error("Failed to create OAuth2 request", "error", err)
		return "", fmt.Errorf("failed to create OAuth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("OAuth2 request failed", "error", err)
		return "", fmt.Errorf("OAuth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Warn("OAuth2 server returned non-200 status", "status", resp.StatusCode)
		return "", fmt.Errorf("OAuth server returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to parse OAuth2 response", "error", err)
		return "", fmt.Errorf("failed to read OAuth response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Error("Failed to parse OAuth2 response", "error", err)
		return "", fmt.Errorf("failed to parse OAuth response: %w", err)
	}

	token, ok := result["access_token"].(string)
	if !ok {
		logger.Error("OAuth2 response did not contain an access_token")
		return "", fmt.Errorf("access_token not found in response")
	}

	logger.Debug("Successfully retrieved OAuth2 token")
	return token, nil
}
