package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dasvh/enchante/internal/config"
	"github.com/dasvh/enchante/internal/testutil"
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
			expectedValue:  "Basic dGVzdC11c2VyOnRlc3QtcGFzcw==",
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
			header, value, err := GetAuthHeader(cfg, testutil.Logger)

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
				Scope:        "openid profile email",
			},
		},
	}

	header, value, err := GetAuthHeader(cfg, testutil.Logger)
	assert.NoError(t, err)
	assert.Equal(t, "Authorization", header)
	assert.Equal(t, "Bearer mocked-token", value)
}

func TestOAuth2Errors(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		expectErr      error
	}{
		{"Valid OAuth2 Response", `{"access_token": "mocked-token"}`, 200, nil},
		{"Invalid JSON Response", `invalid json`, 200, errors.New("failed to parse OAuth response")},
		{"Missing Token", `{}`, 200, errors.New("access_token not found in response")},
		{"OAuth2 Server Error", `{"error": "invalid_request"}`, 400, errors.New("OAuth server returned status")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.mockStatusCode)
				w.Write([]byte(tc.mockResponse))
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
						Scope:        "openid profile email",
					},
				},
			}

			_, _, err := GetAuthHeader(cfg, testutil.Logger)

			if tc.expectErr == nil {
				assert.NoError(t, err, "Unexpected error")
			} else {
				assert.Error(t, err, "Expected an error but got none")
				assert.Contains(t, err.Error(), tc.expectErr.Error(), "Expected %v, got %v", tc.expectErr, err)
			}
		})
	}
}
