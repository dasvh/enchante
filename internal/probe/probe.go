package probe

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/dasvh/enchante/internal/auth"
	"github.com/dasvh/enchante/internal/config"
)

func RunProbe(cfg *config.Config) {
	var wg sync.WaitGroup
	results := make(chan time.Duration, cfg.ProbingConfig.TotalRequests)

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
					go makeRequest(ep, header, value, cfg.ProbingConfig.DelayBetween, &wg, results)
				}
			}(endpoint)
		}
	}

	wg.Wait()
	close(results)

	var totalDuration time.Duration
	count := 0
	for duration := range results {
		totalDuration += duration
		count++
	}

	avgTime := totalDuration / time.Duration(count)
	fmt.Printf("\nCompleted %d requests in %s\n", count, time.Since(startTest))
	fmt.Printf("Average response time: %s\n", avgTime)
}
func makeRequest(endpoint config.Endpoint, authHeader, authValue string, delay config.Delay, wg *sync.WaitGroup, results chan<- time.Duration) {
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
	client := &http.Client{}

	var reqBody io.Reader
	if endpoint.Body != "" {
		reqBody = bytes.NewReader([]byte(endpoint.Body))
	} else {
		reqBody = nil
	}

	req, err := http.NewRequest(endpoint.Method, endpoint.URL, reqBody)
	if err != nil {
		fmt.Println("Request creation error:", err)
		return
	}

	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
	}
	if authHeader != "" {
		req.Header.Set(authHeader, authValue)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Request error:", err)
		return
	}
	defer resp.Body.Close()

	results <- time.Since(start)
}
