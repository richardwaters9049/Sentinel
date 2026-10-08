package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("SENTINEL_ENV", "")
	t.Setenv("SENTINEL_AUTH_MODE", "")
	t.Setenv("SENTINEL_AUTH_CREDENTIALS_FILE", "")
	t.Setenv("SENTINEL_HTTP_ADDR", "")
	t.Setenv("SENTINEL_LOG_LEVEL", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("NATS_URL", "")
	t.Setenv("SENTINEL_ML_URL", "")
	t.Setenv("SENTINEL_BEHAVIOUR_ENABLED", "")
	t.Setenv("SENTINEL_DEPENDENCY_TIMEOUT", "")
	t.Setenv("SENTINEL_SHUTDOWN_TIMEOUT", "")
	t.Setenv("SENTINEL_READ_HEADER_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected defaults to load, got error: %v", err)
	}

	if cfg.HTTPAddr != "127.0.0.1:8080" {
		t.Fatalf("expected default HTTP address, got %q", cfg.HTTPAddr)
	}

	if cfg.DependencyTimeout != 10*time.Second {
		t.Fatalf("expected 10s dependency timeout, got %s", cfg.DependencyTimeout)
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("SENTINEL_LOG_LEVEL", "verbose")

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid log level to be rejected")
	}
}

func TestLoadParsesDurationSecondsFallback(t *testing.T) {
	t.Setenv("SENTINEL_DEPENDENCY_TIMEOUT", "7")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	if cfg.DependencyTimeout != 7*time.Second {
		t.Fatalf("expected 7s dependency timeout, got %s", cfg.DependencyTimeout)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv("SENTINEL_DEPENDENCY_TIMEOUT", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid duration to be rejected")
	}
}

func TestLoadBehaviourAnalyticsOptIn(t *testing.T) {
	t.Setenv("SENTINEL_BEHAVIOUR_ENABLED", "true")
	t.Setenv("SENTINEL_ML_URL", "http://127.0.0.1:8090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected behaviour config to load: %v", err)
	}
	if !cfg.BehaviourEnabled {
		t.Fatal("expected behavioural analytics to be enabled")
	}
	if cfg.MLURL != "http://127.0.0.1:8090" {
		t.Fatalf("unexpected ML URL %q", cfg.MLURL)
	}
}

func TestAuthenticationConfigFailsClosed(t *testing.T) {
	for _, tc := range []struct{ environment, mode, file string }{
		{"production", "", ""}, {"production", "development", ""}, {"development", "typo", ""},
		{"development", "required", "/missing"}, {"development", "development", "/unexpected"},
	} {
		t.Run(tc.environment+tc.mode+tc.file, func(t *testing.T) {
			t.Setenv("SENTINEL_ENV", tc.environment)
			t.Setenv("SENTINEL_AUTH_MODE", tc.mode)
			t.Setenv("SENTINEL_AUTH_CREDENTIALS_FILE", tc.file)
			if _, err := Load(); err == nil {
				t.Fatal("unsafe authentication config accepted")
			}
		})
	}
}
