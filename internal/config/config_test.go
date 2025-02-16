package config

import (
	"os"
	"testing"

	"github.com/dasvh/enchante/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestAuthConfig(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		expected AuthConfig
	}{
		{
			name: "OAuth2 Authentication",
			yamlData: `
auth:
  enabled: true
  type: "oauth2"
  oauth2:
    token_url: "http://keycloak/realms/realm/protocol/openid-connect/token"
    client_id: "client-id"
    client_secret: "client-secret"
    grant_type: "password"
    username: "user"
    password: "pass"
    scope: "openid profile email"
`,
			expected: AuthConfig{
				Enabled: true,
				Type:    "oauth2",
				OAuth2: OAuth2Auth{
					TokenURL:     "http://keycloak/realms/realm/protocol/openid-connect/token",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
					GrantType:    "password",
					Username:     "user",
					Password:     "pass",
					Scope:        "openid profile email",
				},
			},
		},
		{
			name: "Basic Authentication",
			yamlData: `
auth:
  enabled: true
  type: "basic"
  basic:
    username: "test-user"
    password: "test-pass"
`,
			expected: AuthConfig{
				Enabled: true,
				Type:    "basic",
				Basic: BasicAuth{
					Username: "test-user",
					Password: "test-pass",
				},
			},
		},
		{
			name: "API Key Authentication",
			yamlData: `
auth:
  enabled: true
  type: "api_key"
  api_key:
    header: "X-API-Key"
    value: "api-secret-key"
`,
			expected: AuthConfig{
				Enabled: true,
				Type:    "api_key",
				APIKey: APIKeyAuth{
					Header: "X-API-Key",
					Value:  "api-secret-key",
				},
			},
		},
		{
			name: "No Authentication",
			yamlData: `
auth:
  enabled: false
  type: ""
`,
			expected: AuthConfig{
				Enabled: false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
			assert.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tc.yamlData)
			assert.NoError(t, err)
			tmpFile.Close()

			cfg, err := LoadConfig(tmpFile.Name(), testutil.Logger)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, cfg.Auth)
		})
	}
}

func TestProbingConfig(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		expected ProbingConfig
	}{
		{
			name: "Basic ProbingConfig",
			yamlData: `
probe:
  concurrent_requests: 10
  total_requests: 100
  request_timeout_ms: 440
  delay_between:
    enabled: false
`,
			expected: ProbingConfig{
				ConcurrentRequests: 10,
				TotalRequests:      100,
				RequestTimeoutMS:   440,
				DelayBetween: Delay{
					Enabled: false,
				},
			},
		},
		{
			name: "Random Delay Enabled",
			yamlData: `
probe:
  concurrent_requests: 5
  total_requests: 50
  delay_between:
    enabled: true
    type: "random"
    min: 200
    max: 800
`,
			expected: ProbingConfig{
				ConcurrentRequests: 5,
				TotalRequests:      50,
				RequestTimeoutMS:   DefaultRequestTimeout,
				DelayBetween: Delay{
					Enabled: true,
					Type:    "random",
					Min:     200,
					Max:     800,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
			assert.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tc.yamlData)
			assert.NoError(t, err)
			tmpFile.Close()

			cfg, err := LoadConfig(tmpFile.Name(), testutil.Logger)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, cfg.ProbingConfig)
		})
	}
}

func TestEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		expected []Endpoint
	}{
		{
			name: "Single Endpoint",
			yamlData: `
probe:
  endpoints:
    - url: "https://api.example.com/resource"
      method: "GET"
      headers:
        Authorization: "Bearer token"
`,
			expected: []Endpoint{
				{
					URL:    "https://api.example.com/resource",
					Method: "GET",
					Headers: map[string]string{
						"Authorization": "Bearer token",
					},
				},
			},
		},
		{
			name: "Multiple Endpoints with Body",
			yamlData: `
probe:
  endpoints:
    - url: "https://api.example.com/get"
      method: "GET"
    - url: "https://api.example.com/post"
      method: "POST"
      body: '{"key": "value"}'
      headers:
        Content-Type: "application/json"
`,
			expected: []Endpoint{
				{
					URL:    "https://api.example.com/get",
					Method: "GET",
				},
				{
					URL:    "https://api.example.com/post",
					Method: "POST",
					Body:   `{"key": "value"}`,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
			assert.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tc.yamlData)
			assert.NoError(t, err)
			tmpFile.Close()

			cfg, err := LoadConfig(tmpFile.Name(), testutil.Logger)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, cfg.ProbingConfig.Endpoints)
		})
	}
}

func TestEnvVariableReplacement(t *testing.T) {
	os.Setenv("TEST_USERNAME", "env-user")
	os.Setenv("TEST_PASSWORD", "env-pass")
	defer os.Clearenv()

	yamlData := `
auth:
  enabled: true
  type: "basic"
  basic:
    username: "${TEST_USERNAME}"
    password: "$(TEST_PASSWORD)"
`

	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(yamlData)
	assert.NoError(t, err)
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name(), testutil.Logger)
	assert.NoError(t, err)
	assert.Equal(t, "env-user", cfg.Auth.Basic.Username)
	assert.Equal(t, "env-pass", cfg.Auth.Basic.Password)
}

func TestEndpointAuthOverride(t *testing.T) {
	os.Setenv("TEST_CLIENT_ID", "env-client-id")
	os.Setenv("TEST_CLIENT_SECRET", "env-client-secret")
	defer os.Clearenv()

	yamlData := `
auth:
  enabled: true
  type: "oauth2"
  oauth2:
    token_url: "http://global-auth/token"
    client_id: "global-client"
    client_secret: "global-secret"
    grant_type: "password"
    username: "global-user"
    password: "global-pass"
    scope: "openid profile email"
probe:
  endpoints:
    - url: "https://api.example.com/no-auth"
      method: "GET"
      auth:
        enabled: false
    - url: "https://api.example.com/override-auth"
      method: "POST"
      auth:
        enabled: true
        type: "oauth2"
        oauth2:
          token_url: "http://override-auth/token"
          client_id: "${TEST_CLIENT_ID}"
          client_secret: "$(TEST_CLIENT_SECRET)"
          grant_type: "password"
          username: "endpoint-user"
          password: "endpoint-pass"
          scope: "custom-scope"
`

	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(yamlData)
	assert.NoError(t, err)
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name(), testutil.Logger)
	assert.NoError(t, err)

	assert.True(t, cfg.Auth.Enabled)
	assert.Equal(t, "oauth2", cfg.Auth.Type)
	assert.Equal(t, "http://global-auth/token", cfg.Auth.OAuth2.TokenURL)

	assert.False(t, cfg.ProbingConfig.Endpoints[0].AuthConfig.Enabled)

	endpointAuth := cfg.ProbingConfig.Endpoints[1].AuthConfig
	assert.True(t, endpointAuth.Enabled)
	assert.Equal(t, "oauth2", endpointAuth.Type)
	assert.Equal(t, "http://override-auth/token", endpointAuth.OAuth2.TokenURL)
	assert.Equal(t, "env-client-id", endpointAuth.OAuth2.ClientID)
	assert.Equal(t, "env-client-secret", endpointAuth.OAuth2.ClientSecret)
	assert.Equal(t, "endpoint-user", endpointAuth.OAuth2.Username)
	assert.Equal(t, "endpoint-pass", endpointAuth.OAuth2.Password)
	assert.Equal(t, "custom-scope", endpointAuth.OAuth2.Scope)
}
