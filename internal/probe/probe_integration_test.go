package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	RunProbe(context.Background(), cfg, testutil.Logger)
	elapsed := time.Since(start)

	assert.Greater(t, elapsed, time.Duration(0), "Probe should have taken > 0s")
	assert.Equal(t, 4, getCount, "Expected exactly 4 GET requests")
	assert.Equal(t, 4, postCount, "Expected exactly 4 POST requests")
}

func TestProbeWithOAuth2(t *testing.T) {
	const fakeToken = "mock-oauth-token"

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		if r.FormValue("client_id") == "test-client" && r.FormValue("client_secret") == "test-secret" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token": "` + fakeToken + `"}`))
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}))
	defer tokenServer.Close()

	var receivedToken string
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("Authorization")
		if receivedToken != "Bearer "+fakeToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Authenticated request success"}`))
	}))
	defer apiServer.Close()

	t.Logf("OAuth2 server running at: %s", tokenServer.URL)
	t.Logf("API server running at: %s", apiServer.URL)

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
			Type:    "oauth2",
			OAuth2: config.OAuth2Auth{
				TokenURL:     tokenServer.URL,
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				GrantType:    "client_credentials",
			},
		},
		ProbingConfig: config.ProbingConfig{
			ConcurrentRequests: 2,
			TotalRequests:      5,
			RequestTimeoutMS:   2000,
			DelayBetween: config.Delay{
				Enabled: true,
				Type:    "fixed",
				Fixed:   10,
			},
			Endpoints: []config.Endpoint{
				{URL: apiServer.URL, Method: "GET"},
			},
		},
	}

	start := time.Now()
	RunProbe(context.Background(), cfg, testutil.Logger)
	elapsed := time.Since(start)

	assert.Greater(t, elapsed, time.Duration(0), "Probe should have taken > 0s")
	assert.Equal(t, "Bearer "+fakeToken, receivedToken, "Expected request to include Bearer token")
}
