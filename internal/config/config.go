package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Auth          AuthConfig    `yaml:"auth"`
	ProbingConfig ProbingConfig `yaml:"probe"`
}

type AuthConfig struct {
	Enabled bool   `yaml:"enabled"`
	Type    string `yaml:"type,omitempty"`

	APIKey APIKeyAuth `yaml:"api_key,omitempty"`
	Basic  BasicAuth  `yaml:"basic,omitempty"`
	OAuth2 OAuth2Auth `yaml:"oauth2,omitempty"`
}

type APIKeyAuth struct {
	Header string `yaml:"header"`
	Value  string `yaml:"value"`
}

type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type OAuth2Auth struct {
	TokenURL     string `yaml:"token_url"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	GrantType    string `yaml:"grant_type"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
}

type ProbingConfig struct {
	ConcurrentRequests int        `yaml:"concurrent_requests"`
	TotalRequests      int        `yaml:"total_requests"`
	DelayBetween       Delay      `yaml:"delay_between"`
	Endpoints          []Endpoint `yaml:"endpoints"`
}

type Delay struct {
	Enabled bool   `yaml:"enabled"`
	Type    string `yaml:"type"`
	Min     int    `yaml:"min,omitempty"`
	Max     int    `yaml:"max,omitempty"`
	Fixed   int    `yaml:"fixed,omitempty"`
}

type Endpoint struct {
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Body    string            `yaml:"body,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	return &config, nil
}
