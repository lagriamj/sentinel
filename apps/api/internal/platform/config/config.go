package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultJobsQueue      = "sentinel:scan-jobs"
	defaultResultsQueue   = "sentinel:scan-results"
	defaultShutdownPeriod = 10 * time.Second
	defaultAccessTTL      = 15 * time.Minute
	defaultRefreshTTL     = 30 * 24 * time.Hour
	placeholderAuthSecret = "change-me-local-only-auth-secret-value"
)

// Config is the process configuration loaded from the environment.
// Connection URLs are never written to logs by this package.
type Config struct {
	Environment     string
	HTTPAddr        string
	DatabaseURL     string
	RedisURL        string
	JobsQueue       string
	ResultsQueue    string
	TrustedProxies  []string
	ShutdownTimeout time.Duration
	AuthSecret      string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	CookieSecure    bool
}

// Load reads API configuration from the process environment.
func Load() (Config, error) {
	return LoadFrom(os.Getenv)
}

// LoadDatabaseURL reads only the PostgreSQL URL. Migration commands use this
// so they do not require the HTTP server or Redis settings.
func LoadDatabaseURL() (string, error) {
	return databaseURLFrom(os.Getenv)
}

func databaseURLFrom(getenv func(string) string) (string, error) {
	value := getenv("API_DATABASE_URL")
	if err := requireScheme(value, "API_DATABASE_URL", "postgres", "postgresql"); err != nil {
		return "", err
	}
	return value, nil
}

// LoadFrom reads API configuration using the provided lookup.
func LoadFrom(getenv func(string) string) (Config, error) {
	cfg := Config{
		Environment:     getenv("API_ENVIRONMENT"),
		HTTPAddr:        getenv("API_HTTP_ADDR"),
		DatabaseURL:     getenv("API_DATABASE_URL"),
		RedisURL:        getenv("API_REDIS_URL"),
		JobsQueue:       valueOrDefault(getenv("API_JOBS_QUEUE"), defaultJobsQueue),
		ResultsQueue:    valueOrDefault(getenv("API_RESULTS_QUEUE"), defaultResultsQueue),
		ShutdownTimeout: defaultShutdownPeriod,
		AuthSecret:      getenv("API_AUTH_SECRET"),
		AccessTokenTTL:  defaultAccessTTL,
		RefreshTokenTTL: defaultRefreshTTL,
		CookieSecure:    getenv("API_ENVIRONMENT") == "production",
	}
	proxies, err := splitProxies(getenv("API_TRUSTED_PROXIES"))
	if err != nil {
		return Config{}, err
	}
	cfg.TrustedProxies = proxies

	if raw := getenv("API_ACCESS_TOKEN_TTL"); raw != "" {
		ttl, err := time.ParseDuration(raw)
		if err != nil || ttl < time.Minute || ttl > time.Hour {
			return Config{}, errors.New("API_ACCESS_TOKEN_TTL must be between 1m and 1h")
		}
		cfg.AccessTokenTTL = ttl
	}
	if raw := getenv("API_REFRESH_TOKEN_TTL"); raw != "" {
		ttl, err := time.ParseDuration(raw)
		if err != nil || ttl < time.Hour || ttl > 90*24*time.Hour {
			return Config{}, errors.New("API_REFRESH_TOKEN_TTL must be between 1h and 2160h")
		}
		cfg.RefreshTokenTTL = ttl
	}
	if raw := getenv("API_COOKIE_SECURE"); raw != "" {
		switch raw {
		case "true":
			cfg.CookieSecure = true
		case "false":
			cfg.CookieSecure = false
		default:
			return Config{}, errors.New("API_COOKIE_SECURE must be true or false")
		}
	}

	if raw := getenv("API_SHUTDOWN_TIMEOUT"); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil || timeout <= 0 {
			return Config{}, errors.New("API_SHUTDOWN_TIMEOUT must be a positive duration")
		}
		cfg.ShutdownTimeout = timeout
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) validate() error {
	switch cfg.Environment {
	case "development", "test", "production":
	default:
		return errors.New("API_ENVIRONMENT must be development, test, or production")
	}
	if cfg.HTTPAddr == "" {
		return errors.New("API_HTTP_ADDR is required")
	}
	if err := requireScheme(cfg.DatabaseURL, "API_DATABASE_URL", "postgres", "postgresql"); err != nil {
		return err
	}
	if err := requireScheme(cfg.RedisURL, "API_REDIS_URL", "redis", "rediss"); err != nil {
		return err
	}
	if err := requireQueueName("API_JOBS_QUEUE", cfg.JobsQueue); err != nil {
		return err
	}
	if err := requireQueueName("API_RESULTS_QUEUE", cfg.ResultsQueue); err != nil {
		return err
	}
	if len(cfg.AuthSecret) < 32 || cfg.AuthSecret == placeholderAuthSecret || strings.ContainsAny(cfg.AuthSecret, " \t\r\n") {
		return errors.New("API_AUTH_SECRET must be at least 32 characters and must not be the placeholder")
	}
	return nil
}

func requireScheme(raw, name string, schemes ...string) error {
	if raw == "" {
		return fmt.Errorf("%s is required", name)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("%s is not a valid URL", name)
	}
	for _, scheme := range schemes {
		if parsed.Scheme == scheme {
			return nil
		}
	}
	return fmt.Errorf("%s has an unsupported scheme", name)
}

func requireQueueName(name, value string) error {
	if value == "" || strings.ContainsAny(value, " \t\r\n") {
		return fmt.Errorf("%s must be a non-empty queue name", name)
	}
	return nil
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func splitProxies(value string) ([]string, error) {
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if net.ParseIP(part) == nil {
			if _, _, err := net.ParseCIDR(part); err != nil {
				return nil, errors.New("API_TRUSTED_PROXIES contains an invalid address")
			}
		}
		items = append(items, part)
	}
	return items, nil
}
