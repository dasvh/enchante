package probe

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dasvh/enchante/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMakeRequest(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	testEndpoint := config.Endpoint{
		URL:    mockServer.URL,
		Method: "GET",
	}

	results := make(chan time.Duration, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	makeRequest(testEndpoint, "Authorization", "Bearer test-token", config.Delay{}, &wg, results)

	wg.Wait()
	close(results)

	assert.Len(t, results, 1)
}

func TestAuthHeaderIsSet(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	testEndpoint := config.Endpoint{
		URL:    mockServer.URL,
		Method: "GET",
	}

	results := make(chan time.Duration, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	makeRequest(testEndpoint, "Authorization", "Bearer test-token", config.Delay{}, &wg, results)

	wg.Wait()
	close(results)
	assert.Len(t, results, 1)
}

func TestRequestDelays(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	tests := []struct {
		name          string
		delayConfig   config.Delay
		expectedMinMs int64
		expectedMaxMs int64
	}{
		{
			name: "Fixed",
			delayConfig: config.Delay{
				Enabled: true,
				Fixed:   500,
			},
			expectedMinMs: 500,
			expectedMaxMs: 510,
		},
		{
			name: "Random",
			delayConfig: config.Delay{
				Enabled: true,
				Type:    "random",
				Min:     200,
				Max:     600,
			},
			expectedMinMs: 200,
			expectedMaxMs: 610,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testEndpoint := config.Endpoint{
				URL:    mockServer.URL,
				Method: "GET",
			}

			results := make(chan time.Duration, 1)
			var wg sync.WaitGroup
			wg.Add(1)

			start := time.Now()
			makeRequest(testEndpoint, "Authorization", "Bearer test-token", tc.delayConfig, &wg, results)
			wg.Wait()
			elapsed := time.Since(start).Milliseconds()

			close(results)

			fmt.Println("Elapsed time: ", elapsed)
			assert.GreaterOrEqual(t, elapsed, tc.expectedMinMs)
			assert.LessOrEqual(t, elapsed, tc.expectedMaxMs)
		})
	}
}

func TestConcurrentRequests(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		ProbingConfig: config.ProbingConfig{
			ConcurrentRequests: 5,
			TotalRequests:      10,
			Endpoints: []config.Endpoint{
				{URL: mockServer.URL, Method: "GET"},
			},
		},
	}

	start := time.Now()
	RunProbe(cfg)
	elapsed := time.Since(start)

	assert.Greater(t, elapsed, time.Duration(0))
}
