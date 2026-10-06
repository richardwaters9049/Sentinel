package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("SENTINEL_ENV", "")
	t.Setenv("SENTINEL_HTTP_ADDR", "")
	t.Setenv("SENTINEL_LOG_LEVEL", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("NATS_URL", "")
	t.Setenv("SENTINEL_DEPENDENCY_TIMEOUT", "")
	t.Setenv("SENTINEL_SHUTDOWN_TIMEOUT", "")
	t.Setenv("SENTINEL_READ_HEADER_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected defaults to load, got error: %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
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
