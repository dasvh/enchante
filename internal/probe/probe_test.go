package probe

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dasvh/enchante/internal/config"
	"github.com/dasvh/enchante/internal/testutil"
	"github.com/stretchr/testify/assert"
)

var defaultTimeout = time.Duration(config.DefaultRequestTimeout) * time.Millisecond

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

			err := makeRequest(context.Background(), testEndpoint, "Authorization", "Bearer test-token", config.Delay{}, defaultTimeout, results, testutil.Logger)
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
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	testEndpoint := config.Endpoint{
		URL:    mockServer.URL,
		Method: "GET",
	}

	results := make(chan time.Duration, 1)
	timeout := 10 * time.Millisecond

	start := time.Now()
	err := makeRequest(context.Background(), testEndpoint, "Authorization", "Bearer test-token", config.Delay{}, timeout, results, testutil.Logger)
	elapsed := time.Since(start).Milliseconds()

	close(results)

	assert.Error(t, err, "Expected a timeout error")
	assert.Contains(t, err.Error(), "context deadline exceeded", "Expected timeout error message")
	assert.GreaterOrEqualf(t, elapsed, int64(timeout.Milliseconds()), "Expected elapsed time to be at least %dms, got %dms", timeout.Milliseconds(), elapsed)
}

func TestNetworkFailure(t *testing.T) {
	testEndpoint := config.Endpoint{
		URL:    "http://invalid-url.local",
		Method: "GET",
	}

	results := make(chan time.Duration, 1)

	err := makeRequest(context.Background(), testEndpoint, "Authorization", "Bearer test-token", config.Delay{}, 100, results, testutil.Logger)
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

	makeRequest(context.Background(), testEndpoint, "Authorization", "Bearer test-token", config.Delay{}, defaultTimeout, results, testutil.Logger)
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

			start := time.Now()
			makeRequest(context.Background(), testEndpoint, "Authorization", "Bearer test-token", tc.delayConfig, defaultTimeout, results, testutil.Logger)
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
	RunProbe(context.Background(), cfg, testutil.Logger)
	elapsed := time.Since(start)

	assert.Greater(t, elapsed, time.Duration(0))
}

func TestTotalRequestsCount(t *testing.T) {
	var requestCount int32 // Use int32 instead of int

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			atomic.AddInt32(&requestCount, 1) // Safe atomic increment
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		ProbingConfig: config.ProbingConfig{
			ConcurrentRequests: 2,
			TotalRequests:      5,
			RequestTimeoutMS:   10,
			Endpoints: []config.Endpoint{
				{URL: mockServer.URL, Method: "GET"},
			},
		},
	}

	start := time.Now()
	RunProbe(context.Background(), cfg, testutil.Logger)
	elapsed := time.Since(start)

	assert.Greater(t, elapsed, time.Duration(0))
	assert.Equal(t, int32(5), requestCount)
}

func TestRunProbeHandlesCancellation(t *testing.T) {
	var requestCount int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method == "GET" {
			requestCount++
		}
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		ProbingConfig: config.ProbingConfig{
			ConcurrentRequests: 2,
			TotalRequests:      10000, // ✅ Reduce total requests
			DelayBetween: config.Delay{
				Enabled: true,
				Type:    "fixed",
				Fixed:   10,
			},
			Endpoints: []config.Endpoint{
				{URL: mockServer.URL, Method: "GET"},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	go RunProbe(ctx, cfg, testutil.Logger)
	// cancel after a short delay
	time.Sleep(250 * time.Nanosecond)
	cancel()

	// wait for logs to finish writing
	time.Sleep(100 * time.Millisecond)

	logs := testutil.GetLogs()
	fmt.Println(logs)
	assert.Contains(t, logs, "Job queue stopped due to cancellation", "Expected job queue cancellation log")
	assert.Contains(t, logs, "Worker stopped due to cancellation", "Expected worker cancellation log")
}
