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

// Config represents the configuration for the application
type Config struct {
	Auth          AuthConfig    `yaml:"auth"`
	ProbingConfig ProbingConfig `yaml:"probe"`
}

// AuthConfig represents the authentication configuration
type AuthConfig struct {
	Enabled bool   `yaml:"enabled"`
	Type    string `yaml:"type,omitempty"`

	APIKey APIKeyAuth `yaml:"api_key,omitempty"`
	Basic  BasicAuth  `yaml:"basic,omitempty"`
	OAuth2 OAuth2Auth `yaml:"oauth2,omitempty"`
}

// APIKeyAuth represents the configuration for API Key authentication
type APIKeyAuth struct {
	Header string `yaml:"header"`
	Value  string `yaml:"value"`
}

// BasicAuth represents the configuration for Basic authentication
type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// OAuth2Auth represents the configuration for OAuth2 authentication
type OAuth2Auth struct {
	TokenURL     string `yaml:"token_url"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	GrantType    string `yaml:"grant_type"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
}

// ProbingConfig represents the probing configuration
type ProbingConfig struct {
	ConcurrentRequests int        `yaml:"concurrent_requests"`
	TotalRequests      int        `yaml:"total_requests"`
	RequestTimeoutMS   int        `yaml:"request_timeout_ms,omitempty"`
	DelayBetween       Delay      `yaml:"delay_between"`
	Endpoints          []Endpoint `yaml:"endpoints"`
}

// Delay represents the configuration for delay between requests
type Delay struct {
	Enabled bool   `yaml:"enabled"`
	Type    string `yaml:"type"`
	Min     int    `yaml:"min,omitempty"`
	Max     int    `yaml:"max,omitempty"`
	Fixed   int    `yaml:"fixed,omitempty"`
}

// Endpoint represents the configuration for an endpoint to probe
type Endpoint struct {
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Body    string            `yaml:"body,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

// LoadConfig loads the config from YAML and environment variables
func LoadConfig(filename string, logger *slog.Logger) (*Config, error) {
	err := godotenv.Load()
	if err := godotenv.Load(); err != nil {
		logger.Debug("No .env file found, continuing with YAML config")
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

	replaceEnvVariables(&config, logger)

	if config.ProbingConfig.RequestTimeoutMS == 0 {
		config.ProbingConfig.RequestTimeoutMS = DefaultRequestTimeout
	}

	logger.Info("Config loaded successfully", "file", filename)
	return &config, nil
}

// replaceEnvVariables replaces environment variables for authentication configuration
func replaceEnvVariables(config *Config, logger *slog.Logger) {
	config.Auth.Basic.Username = replaceEnv(config.Auth.Basic.Username, logger)
	config.Auth.Basic.Password = replaceEnv(config.Auth.Basic.Password, logger)
	config.Auth.APIKey.Header = replaceEnv(config.Auth.APIKey.Header, logger)
	config.Auth.APIKey.Value = replaceEnv(config.Auth.APIKey.Value, logger)
	config.Auth.OAuth2.TokenURL = replaceEnv(config.Auth.OAuth2.TokenURL, logger)
	config.Auth.OAuth2.ClientID = replaceEnv(config.Auth.OAuth2.ClientID, logger)
	config.Auth.OAuth2.ClientSecret = replaceEnv(config.Auth.OAuth2.ClientSecret, logger)
	config.Auth.OAuth2.GrantType = replaceEnv(config.Auth.OAuth2.GrantType, logger)
	config.Auth.OAuth2.Username = replaceEnv(config.Auth.OAuth2.Username, logger)
	config.Auth.OAuth2.Password = replaceEnv(config.Auth.OAuth2.Password, logger)
}

var (
	regexCurlyBraces = regexp.MustCompile(`\$\{([^}]+)}`)
	regexParentheses = regexp.MustCompile(`\$\(([^)]+)\)`)
)

// replaceEnv replaces environment variables in a string
func replaceEnv(value string, logger *slog.Logger) string {
	if match := regexCurlyBraces.FindStringSubmatch(value); match != nil {
		logger.Debug("Replacing environment variable with value", "variable", match[1], "value", os.Getenv(match[1]))
		return os.Getenv(match[1])
	} else if match := regexParentheses.FindStringSubmatch(value); match != nil {
		logger.Debug("Replacing environment variable with value", "variable", match[1], "value", os.Getenv(match[1]))
		return os.Getenv(match[1])
	}
	return value
}
