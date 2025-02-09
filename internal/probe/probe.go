package probe

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

func RunProbe(cfg *config.Config) {
	var wg sync.WaitGroup
	results := make(chan time.Duration, cfg.ProbingConfig.TotalRequests)
	errorsChan := make(chan error, cfg.ProbingConfig.TotalRequests)

	header, value, err := auth.GetAuthHeader(cfg)
	if err != nil {
		fmt.Println("Error getting authentication header:", err)
		return
	}

	startTest := time.Now()
	for _, endpoint := range cfg.ProbingConfig.Endpoints {
		fmt.Printf("Testing: %s %s\n", endpoint.Method, endpoint.URL)
		for i := 0; i < cfg.ProbingConfig.ConcurrentRequests; i++ {
			wg.Add(1)
			go func(ep config.Endpoint) {
				defer wg.Done()
				for j := 0; j < cfg.ProbingConfig.TotalRequests/cfg.ProbingConfig.ConcurrentRequests; j++ {
					wg.Add(1)
					go func() {
						err := makeRequest(ep, header, value, cfg.ProbingConfig.DelayBetween, time.Duration(cfg.ProbingConfig.RequestTimeoutMS), &wg, results)
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
		fmt.Println("Request error:", err)
	}

	var totalDuration time.Duration
	count := 0
	for duration := range results {
		totalDuration += duration
		count++
	}

	if count > 0 {
		avgTime := totalDuration / time.Duration(count)
		fmt.Printf("\nCompleted %d requests in %s\n", count, time.Since(startTest))
		fmt.Printf("Average response time: %s\n", avgTime)
	} else {
		fmt.Println("\nNo successful requests.")
	}
}

func makeRequest(endpoint config.Endpoint, authHeader, authValue string, delay config.Delay, timeout time.Duration, wg *sync.WaitGroup, results chan<- time.Duration) error {
	defer wg.Done()

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
		return fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: status code %d", ErrStatusCode, resp.StatusCode)
	}

	results <- time.Since(start)
	return nil
}
