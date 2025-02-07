package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/dasvh/enchante/internal/auth"
	"github.com/dasvh/enchante/internal/config"
)

func main() {
	loadedConfig, err := config.LoadConfig("testdata/BasicAuthNoDelay.yaml")
	if err != nil {
		return
	}

	fmt.Println(loadedConfig.Auth.Enabled)
	fmt.Println(loadedConfig.ProbingConfig.ConcurrentRequests)

	header, value, err := auth.GetAuthHeader(loadedConfig)
	if err != nil {
		return
	}

	fmt.Println(header)
	fmt.Println(value)

	err = godotenv.Load()
	if err != nil {
		return
	}

	fmt.Println("BASIC_USERNAME: ", os.Getenv("BASIC_USERNAME"))
}
