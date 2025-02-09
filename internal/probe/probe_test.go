package probe

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dasvh/enchante/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMakeRequestHandlesErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expectErr  error
	}{
		{"Success Response", 200, nil},
		{"Server Error", 500, ErrStatusCode},
		{"Forbidden", 403, ErrStatusCode},
		{"Not Found", 404, ErrStatusCode},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer mockServer.Close()

			testEndpoint := config.Endpoint{
				URL:    mockServer.URL,
				Method: "GET",
			}

			results := make(chan time.Duration, 1)
			var wg sync.WaitGroup
			wg.Add(1)

			err := makeRequest(testEndpoint, "Authorization", "Bearer test-token", config.Delay{}, 0, &wg, results)
			wg.Wait()
			close(results)

			if tc.expectErr == nil {
				assert.NoError(t, err, "Unexpected error")
			} else {
				assert.Error(t, err, "Expected an error but got none")
				assert.True(t, errors.Is(err, tc.expectErr), "Expected wrapped error type: %v, got: %v", tc.expectErr, err)
				assert.Contains(t, err.Error(), fmt.Sprintf("status code %d", tc.statusCode), "Error should include the status code")
			}
		})
	}
}

func TestRequestTimeout(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	testEndpoint := config.Endpoint{
		URL:    mockServer.URL,
		Method: "GET",
	}

	results := make(chan time.Duration, 5)
	var wg sync.WaitGroup
	wg.Add(1)

	start := time.Now()
	err := makeRequest(testEndpoint, "Authorization", "Bearer test-token", config.Delay{}, 5*time.Millisecond, &wg, results) // Correct timeout
	wg.Wait()
	elapsed := time.Since(start).Milliseconds()

	close(results)

	assert.Error(t, err, "Expected a timeout error")
	assert.Contains(t, err.Error(), "Client.Timeout exceeded", "Expected timeout error message")
	assert.GreaterOrEqualf(t, elapsed, int64(5), "Expected elapsed time to be at least 5ms, got %d", elapsed)
}

func TestNetworkFailure(t *testing.T) {
	testEndpoint := config.Endpoint{
		URL:    "http://invalid-url.local",
		Method: "GET",
	}

	results := make(chan time.Duration, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	err := makeRequest(testEndpoint, "Authorization", "Bearer test-token", config.Delay{}, 100, &wg, results)
	wg.Wait()
	close(results)

	assert.Error(t, err, "Expected a network failure error")
	assert.True(t, errors.Is(err, ErrRequestFailed), "Expected wrapped network failure error")
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

	makeRequest(testEndpoint, "Authorization", "Bearer test-token", config.Delay{}, 0, &wg, results)

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
			makeRequest(testEndpoint, "Authorization", "Bearer test-token", tc.delayConfig, 1, &wg, results)
			wg.Wait()
			elapsed := time.Since(start).Milliseconds()

			close(results)

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
