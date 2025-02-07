package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dasvh/enchante/internal/config"
)

func GetAuthHeader(cfg *config.Config) (string, string, error) {
	if !cfg.Auth.Enabled {
		return "", "", nil
	}
	switch cfg.Auth.Type {
	case "api_key":
		return cfg.Auth.APIKey.Header, cfg.Auth.APIKey.Value, nil
	case "basic":
		encoded := base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Basic.Username + ":" + cfg.Auth.Basic.Password))
		fmt.Println(cfg.Auth.Basic.Username)
		return "Authorization", "Basic " + encoded, nil
	case "oauth2":
		token, err := getOAuthToken(cfg.Auth.OAuth2)
		if err != nil {
			return "", "", err
		}
		return "Authorization", "Bearer " + token, nil
	default:
		return "", "", fmt.Errorf("unsupported auth type: %s", cfg.Auth.Type)
	}
}

func getOAuthToken(auth config.OAuth2Auth) (string, error) {
	data := fmt.Sprintf("client_id=%s&client_secret=%s&username=%s&password=%s&grant_type=%s",
		auth.ClientID, auth.ClientSecret, auth.Username, auth.Password, auth.GrantType)

	req, err := http.NewRequest("POST", auth.TokenURL, bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("token not found in response")
	}
	return token, nil
}
