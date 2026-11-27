package config

import "testing"

func TestLoadFromRequiresDatabaseAndRedis(t *testing.T) {
	_, err := LoadFrom(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected missing configuration to fail")
	}
}

func TestLoadFromDefaultsQueueNames(t *testing.T) {
	cfg, err := LoadFrom(func(key string) string {
		switch key {
		case "API_ENVIRONMENT":
			return "test"
		case "API_HTTP_ADDR":
			return ":8080"
		case "API_DATABASE_URL":
			return "postgres://sentinel:secret@postgres:5432/sentinel?sslmode=disable"
		case "API_REDIS_URL":
			return "redis://:secret@redis:6379/0"
		case "API_AUTH_SECRET":
			return "test-secret-that-is-at-least-32b"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.JobsQueue != "sentinel:scan-jobs" || cfg.ResultsQueue != "sentinel:scan-results" {
		t.Fatalf("queue names = %s %s", cfg.JobsQueue, cfg.ResultsQueue)
	}
	if cfg.TrustedProxies != nil {
		t.Fatalf("trusted proxies defaulted to %#v", cfg.TrustedProxies)
	}
}

func TestLoadFromRejectsInvalidProxy(t *testing.T) {
	_, err := LoadFrom(func(key string) string {
		switch key {
		case "API_ENVIRONMENT":
			return "test"
		case "API_HTTP_ADDR":
			return ":8080"
		case "API_DATABASE_URL":
			return "postgres://sentinel:secret@postgres:5432/sentinel?sslmode=disable"
		case "API_REDIS_URL":
			return "redis://:secret@redis:6379/0"
		case "API_TRUSTED_PROXIES":
			return "not-a-proxy"
		default:
			return ""
		}
	})
	if err == nil {
		t.Fatal("expected invalid proxy configuration to fail")
	}
}

func TestLoadDatabaseURL(t *testing.T) {
	url, err := databaseURLFrom(func(key string) string {
		if key == "API_DATABASE_URL" {
			return "postgres://sentinel:secret@postgres:5432/sentinel?sslmode=disable"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("load database url: %v", err)
	}
	if url == "" {
		t.Fatal("expected database url")
	}

	if _, err := databaseURLFrom(func(string) string { return "" }); err == nil {
		t.Fatal("expected missing database url to fail")
	}
}

func TestLoadFromRejectsUnsupportedSchemes(t *testing.T) {
	_, err := LoadFrom(func(key string) string {
		switch key {
		case "API_ENVIRONMENT":
			return "development"
		case "API_HTTP_ADDR":
			return ":8080"
		case "API_DATABASE_URL":
			return "mysql://localhost/sentinel"
		case "API_REDIS_URL":
			return "redis://localhost:6379/0"
		default:
			return ""
		}
	})
	if err == nil {
		t.Fatal("expected unsupported database scheme to fail")
	}
}
