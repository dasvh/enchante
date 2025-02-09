package config

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"github.com/goccy/go-yaml"
	"github.com/joho/godotenv"
)

const DefaultRequestTimeout = 2000

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
	RequestTimeoutMS   int        `yaml:"request_timeout_ms,omitempty"`
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

func LoadConfig(filename string, logger *slog.Logger) (*Config, error) {
	err := godotenv.Load()
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, continuing with YAML config")
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		logger.Error("Failed to read config file", "file", filename, "error", err)
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		logger.Error("Failed to parse YAML", "file", filename, "error", err)
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	replaceEnvVariables(&config)

	if config.ProbingConfig.RequestTimeoutMS == 0 {
		config.ProbingConfig.RequestTimeoutMS = DefaultRequestTimeout
	}

	logger.Info("Config loaded successfully", "file", filename)
	return &config, nil
}

func replaceEnvVariables(config *Config) {
	config.Auth.Basic.Username = replaceEnv(config.Auth.Basic.Username)
	config.Auth.Basic.Password = replaceEnv(config.Auth.Basic.Password)
	config.Auth.APIKey.Header = replaceEnv(config.Auth.APIKey.Header)
	config.Auth.APIKey.Value = replaceEnv(config.Auth.APIKey.Value)
	config.Auth.OAuth2.TokenURL = replaceEnv(config.Auth.OAuth2.TokenURL)
	config.Auth.OAuth2.ClientID = replaceEnv(config.Auth.OAuth2.ClientID)
	config.Auth.OAuth2.ClientSecret = replaceEnv(config.Auth.OAuth2.ClientSecret)
	config.Auth.OAuth2.GrantType = replaceEnv(config.Auth.OAuth2.GrantType)
	config.Auth.OAuth2.Username = replaceEnv(config.Auth.OAuth2.Username)
	config.Auth.OAuth2.Password = replaceEnv(config.Auth.OAuth2.Password)
}

var (
	regexCurlyBraces = regexp.MustCompile(`\$\{([^}]+)}`)
	regexParentheses = regexp.MustCompile(`\$\(([^)]+)\)`)
)

func replaceEnv(value string) string {
	if match := regexCurlyBraces.FindStringSubmatch(value); match != nil {
		return os.Getenv(match[1])
	} else if match := regexParentheses.FindStringSubmatch(value); match != nil {
		return os.Getenv(match[1])
	}
	return value
}
