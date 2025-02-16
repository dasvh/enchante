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
	Scope        string `yaml:"scope,omitempty"`
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
	URL        string            `yaml:"url"`
	Method     string            `yaml:"method"`
	Body       string            `yaml:"body,omitempty"`
	Headers    map[string]string `yaml:"headers,omitempty"`
	AuthConfig *AuthConfig       `yaml:"auth,omitempty"`
}

// LoadConfig loads the config from YAML and environment variables
func LoadConfig(filename string, logger *slog.Logger) (*Config, error) {
	if envErr := godotenv.Load(); envErr != nil {
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
	replaceAuthEnvVars(&config.Auth, logger)

	for i := range config.ProbingConfig.Endpoints {
		if config.ProbingConfig.Endpoints[i].AuthConfig != nil {
			replaceAuthEnvVars(config.ProbingConfig.Endpoints[i].AuthConfig, logger)
		}
	}
}

// replaceAuthEnvVars replaces environment variables in an AuthConfig
func replaceAuthEnvVars(auth *AuthConfig, logger *slog.Logger) {
	auth.Basic.Username = replaceEnv(auth.Basic.Username, logger)
	auth.Basic.Password = replaceEnv(auth.Basic.Password, logger)
	auth.APIKey.Header = replaceEnv(auth.APIKey.Header, logger)
	auth.APIKey.Value = replaceEnv(auth.APIKey.Value, logger)
	auth.OAuth2.TokenURL = replaceEnv(auth.OAuth2.TokenURL, logger)
	auth.OAuth2.ClientID = replaceEnv(auth.OAuth2.ClientID, logger)
	auth.OAuth2.ClientSecret = replaceEnv(auth.OAuth2.ClientSecret, logger)
	auth.OAuth2.GrantType = replaceEnv(auth.OAuth2.GrantType, logger)
	auth.OAuth2.Username = replaceEnv(auth.OAuth2.Username, logger)
	auth.OAuth2.Password = replaceEnv(auth.OAuth2.Password, logger)
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
