package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dasvh/enchante/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestGetAuthHeader(t *testing.T) {
	tests := []struct {
		name           string
		authCfg        config.AuthConfig
		expectedHeader string
		expectedValue  string
		expectErr      bool
	}{
		{
			name: "Basic Authentication",
			authCfg: config.AuthConfig{
				Enabled: true,
				Type:    "basic",
				Basic: config.BasicAuth{
					Username: "test-user",
					Password: "test-pass",
				},
			},
			expectedHeader: "Authorization",
			expectedValue:  "Basic dGVzdC11c2VyOnRlc3QtcGFzcw==", // Base64 of "test-user:test-pass"
		},
		{
			name: "API Key Authentication",
			authCfg: config.AuthConfig{
				Enabled: true,
				Type:    "api_key",
				APIKey: config.APIKeyAuth{
					Header: "X-API-Key",
					Value:  "api-secret",
				},
			},
			expectedHeader: "X-API-Key",
			expectedValue:  "api-secret",
		},
		{
			name: "No Authentication",
			authCfg: config.AuthConfig{
				Enabled: false,
			},
			expectedHeader: "",
			expectedValue:  "",
		},
		{
			name: "Unsupported Authentication",
			authCfg: config.AuthConfig{
				Enabled: true,
				Type:    "unsupported",
			},
			expectedHeader: "",
			expectedValue:  "",
			expectErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{Auth: tc.authCfg}
			header, value, err := GetAuthHeader(cfg)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedHeader, header)
				assert.Equal(t, tc.expectedValue, value)
			}
		})
	}
}

// Mock OAuth2 Token Server
func TestOAuth2Authentication(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token": "mocked-token"}`))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
			Type:    "oauth2",
			OAuth2: config.OAuth2Auth{
				TokenURL:     mockServer.URL,
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				Username:     "user",
				Password:     "pass",
				GrantType:    "password",
			},
		},
	}

	header, value, err := GetAuthHeader(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "Authorization", header)
	assert.Equal(t, "Bearer mocked-token", value)
}
