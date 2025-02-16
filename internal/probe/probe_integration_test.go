package probe

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dasvh/enchante/internal/config"
	"github.com/dasvh/enchante/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestProbeAgainstLocalServer(t *testing.T) {
	var getCount, postCount int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getCount++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "GET success"}`))
		case "POST":
			postCount++
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"message": "POST success"}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer mockServer.Close()

	t.Logf("Test server running at: %s", mockServer.URL)

	cfg := &config.Config{
		ProbingConfig: config.ProbingConfig{
			ConcurrentRequests: 2,
			TotalRequests:      4,
			RequestTimeoutMS:   1000,
			DelayBetween:       config.Delay{Enabled: false},
			Endpoints: []config.Endpoint{
				{URL: mockServer.URL, Method: "GET"},
				{URL: mockServer.URL, Method: "POST"},
			},
		},
	}

	start := time.Now()
	RunProbe(t.Context(), cfg, testutil.Logger)
	elapsed := time.Since(start)

	assert.Greater(t, elapsed, time.Duration(0), "Probe should have taken > 0s")
	assert.Equal(t, 4, getCount, "Expected exactly 4 GET requests")
	assert.Equal(t, 4, postCount, "Expected exactly 4 POST requests")
}

func TestProbeWithAuthOverrides(t *testing.T) {
	const globalToken = "global-auth-token"
	const endpointToken = "endpoint-auth-token"

	globalAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token": "` + globalToken + `"}`))
	}))
	defer globalAuthServer.Close()

	endpointAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token": "` + endpointToken + `"}`))
	}))
	defer endpointAuthServer.Close()

	var receivedHeaders []string
	var mu sync.Mutex

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedHeaders = append(receivedHeaders, r.Header.Get("Authorization"))
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Authenticated request success"}`))
	}))
	defer apiServer.Close()

	t.Logf("Global OAuth2 server: %s", globalAuthServer.URL)
	t.Logf("Endpoint-specific OAuth2 server: %s", endpointAuthServer.URL)
	t.Logf("API server: %s", apiServer.URL)

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
			Type:    "oauth2",
			OAuth2: config.OAuth2Auth{
				TokenURL:     globalAuthServer.URL,
				ClientID:     "global-client",
				ClientSecret: "global-secret",
				GrantType:    "client_credentials",
			},
		},
		ProbingConfig: config.ProbingConfig{
			ConcurrentRequests: 1,
			TotalRequests:      1,
			RequestTimeoutMS:   50,
			Endpoints: []config.Endpoint{
				// global auth token should be used
				{URL: apiServer.URL, Method: "GET"},
				// no auth token should be used
				{
					URL:    apiServer.URL,
					Method: "GET",
					AuthConfig: &config.AuthConfig{
						Enabled: false,
					},
				},
				// endpoint-specific auth token should be used
				{
					URL:    apiServer.URL,
					Method: "GET",
					AuthConfig: &config.AuthConfig{
						Enabled: true,
						Type:    "oauth2",
						OAuth2: config.OAuth2Auth{
							TokenURL:     endpointAuthServer.URL,
							ClientID:     "endpoint-client",
							ClientSecret: "endpoint-secret",
							GrantType:    "client_credentials",
						},
					},
				},
			},
		},
	}

	RunProbe(t.Context(), cfg, testutil.Logger)

	assert.Equal(t, 3, len(receivedHeaders), "Expected 3 requests to be made")
	assert.Equal(t, "Bearer "+globalToken, receivedHeaders[0], "Expected first request to use global auth token")
	assert.Equal(t, "", receivedHeaders[1], "Expected second request to send no Authorization header")
	assert.Equal(t, "Bearer "+endpointToken, receivedHeaders[2], "Expected third request to use endpoint-specific auth token")
}
