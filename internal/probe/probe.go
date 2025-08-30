package probe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dasvh/enchante/internal/auth"

	"github.com/dasvh/enchante/internal/config"
)

var (
	ErrRequestFailed = errors.New("request error")
	ErrStatusCode    = errors.New("received non-200 status code")
)

// RunProbe runs the probe test with the given configuration
func RunProbe(ctx context.Context, cfg *config.Config, logger *slog.Logger) {
	var wg sync.WaitGroup
	results := make(chan time.Duration, cfg.ProbingConfig.TotalRequests)
	jobs := make(chan config.Endpoint, cfg.ProbingConfig.TotalRequests)

	startTest := time.Now()
	var successCount, failureCount int
	var countMutex sync.Mutex

	// start worker routines
	for worker := 0; worker < cfg.ProbingConfig.ConcurrentRequests; worker++ {
		wg.Go(func() {
			logger.Debug("Worker started", "worker_id", worker)

			for {
				select {
				case <-ctx.Done(): // check if the context has been cancelled
					logger.Warn("Worker stopped due to cancellation", "worker_id", worker)
					return
				case endpoint, ok := <-jobs:
					if !ok {
						logger.Debug("Worker finished", "worker_id", worker)
						return
					}

					logger.Debug("Worker processing request", "worker_id", worker, "url", endpoint.URL)
					headers, err := getHeadersForEndpoint(endpoint, &cfg.Auth, logger)
					if err != nil {
						logger.Error("Error getting headers for endpoint",
							"url", endpoint.URL,
							"auth_enabled", endpoint.AuthConfig != nil && endpoint.AuthConfig.Enabled,
							"error", err)
						countMutex.Lock()
						failureCount++
						countMutex.Unlock()
						continue
					}

					err = makeRequest(ctx, endpoint, headers, cfg.ProbingConfig.DelayBetween, time.Duration(cfg.ProbingConfig.RequestTimeoutMS)*time.Millisecond, results, logger)
					countMutex.Lock()
					if err != nil {
						failureCount++
					} else {
						successCount++
					}
					countMutex.Unlock()
				}
			}
		})
	}

	// add jobs to the queue
	go func() {
		for i := 0; i < cfg.ProbingConfig.TotalRequests; i++ {
			for _, endpoint := range cfg.ProbingConfig.Endpoints {
				select {
				case <-ctx.Done():
					logger.Warn("Job queue stopped due to cancellation")
					return
				case jobs <- endpoint:
					logger.Debug("Job added to queue", "method", endpoint.Method, "url", endpoint.URL)
				}
			}
		}
		close(jobs)
		logger.Debug("Job queue closed")
	}()

	// wait for all workers to finish before closing the results channel
	go func() {
		wg.Wait()
		close(results)
		logger.Debug("All workers finished, closing error and result channels")
	}()

	var totalDuration time.Duration
	count := 0
	for duration := range results {
		totalDuration += duration
		count++
	}

	if count > 0 {
		avgTime := totalDuration / time.Duration(count)
		logger.Info("Test completed",
			"total_requests", count, // TODO: this is misleading since it doesn't account for failed requests
			"successful_requests", successCount,
			"failed_requests", failureCount,
			"duration", time.Since(startTest),
			"avg_response_time", avgTime)
	} else {
		logger.Warn("No requests were successful", "failed_requests", failureCount)
	}
}

// makeRequest makes an HTTP request to the given endpoint and sends the response time to the results channel
func makeRequest(ctx context.Context, endpoint config.Endpoint, headers map[string]string, delay config.Delay, timeout time.Duration, results chan<- time.Duration, logger *slog.Logger) error {
	if delay.Enabled {
		if delay.Type == "random" {
			sleepTime := rand.Intn(delay.Max-delay.Min) + delay.Min
			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		} else {
			time.Sleep(time.Duration(delay.Fixed) * time.Millisecond)
		}
	}

	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	var reqBody io.Reader
	if endpoint.Body != "" {
		reqBody = bytes.NewReader([]byte(endpoint.Body))
	}

	req, err := http.NewRequestWithContext(ctx, endpoint.Method, endpoint.URL, reqBody)
	if err != nil {
		logger.Error("Failed to create request", "url", endpoint.URL, "error", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Request failed", "url", endpoint.URL, "error", err)
		return fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logger.Warn("Received non-200 response", "url", endpoint.URL, "status_code", resp.StatusCode)
		return fmt.Errorf("%w: status code %d", ErrStatusCode, resp.StatusCode)
	}

	elapsed := time.Since(start)
	results <- elapsed

	logger.Debug("Request successful", "url", endpoint.URL, "status_code", resp.StatusCode, "response_time", elapsed)
	return nil
}

// getHeadersForEndpoint returns the headers to be used for the given endpoint
func getHeadersForEndpoint(endpoint config.Endpoint, globalAuth *config.AuthConfig, logger *slog.Logger) (map[string]string, error) {
	headers := make(map[string]string)

	for key, value := range endpoint.Headers {
		headers[key] = value
	}

	authConfig := endpoint.AuthConfig
	if authConfig == nil {
		authConfig = globalAuth
	}

	if authConfig != nil && authConfig.Enabled {
		logger.Debug("Getting auth header", "auth_type", authConfig.Type)
		authHeader, authValue, err := auth.GetAuthHeader(authConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to get auth header: %w", err)
		}
		if authHeader != "" {
			headers[authHeader] = authValue
		}
	}

	headers["User-Agent"] = "Mozilla/5.0 (compatible; EnchanteBot/1.0)"

	return headers, nil
}
