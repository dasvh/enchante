package probe

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
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

func RunProbe(cfg *config.Config, logger *slog.Logger) {
	var wg sync.WaitGroup
	results := make(chan time.Duration, cfg.ProbingConfig.TotalRequests)
	errorsChan := make(chan error, cfg.ProbingConfig.TotalRequests)

	header, value, err := auth.GetAuthHeader(cfg, logger)
	if err != nil {
		logger.Error("Error getting authentication header", "error", err)
		return
	}

	startTest := time.Now()

	for _, endpoint := range cfg.ProbingConfig.Endpoints {
		logger.Info("Testing", "method", endpoint.Method, "url", endpoint.URL)
		for i := 0; i < cfg.ProbingConfig.ConcurrentRequests; i++ {
			wg.Add(1)
			go func(ep config.Endpoint) {
				defer wg.Done()
				for j := 0; j < cfg.ProbingConfig.TotalRequests/cfg.ProbingConfig.ConcurrentRequests; j++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						err := makeRequest(ep, header, value, cfg.ProbingConfig.DelayBetween, time.Duration(cfg.ProbingConfig.RequestTimeoutMS), results, logger)
						if err != nil {
							errorsChan <- err
						}
					}()
				}
			}(endpoint)
		}
	}

	go func() {
		wg.Wait()
		close(results)
		close(errorsChan)
	}()

	for err := range errorsChan {
		logger.Warn("Request error", "error", err)
	}

	var totalDuration time.Duration
	count := 0
	for duration := range results {
		totalDuration += duration
		count++
	}

	if count > 0 {
		avgTime := totalDuration / time.Duration(count)
		logger.Info("Test completed", "total_requests", count, "duration", time.Since(startTest), "avg_response_time", avgTime)
	} else {
		logger.Warn("No requests were successful")
	}
}

func makeRequest(endpoint config.Endpoint, authHeader, authValue string, delay config.Delay, timeout time.Duration, results chan<- time.Duration, logger *slog.Logger) error {
	if delay.Enabled {
		if delay.Type == "random" {
			sleepTime := rand.Intn(delay.Max-delay.Min) + delay.Min
			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		} else {
			time.Sleep(time.Duration(delay.Fixed) * time.Millisecond)
		}
	}

	start := time.Now()
	client := &http.Client{
		Timeout: timeout,
	}

	var reqBody io.Reader
	if endpoint.Body != "" {
		reqBody = bytes.NewReader([]byte(endpoint.Body))
	}

	req, err := http.NewRequest(endpoint.Method, endpoint.URL, reqBody)
	if err != nil {
		logger.Error("Failed to create request", "url", endpoint.URL, "error", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
	}
	if authHeader != "" {
		req.Header.Set(authHeader, authValue)
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

	logger.Info("Request successful", "url", endpoint.URL, "status_code", resp.StatusCode, "response_time", elapsed)
	return nil
}
