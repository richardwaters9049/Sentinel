package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultEnvironment = "development"
	defaultHTTPAddr    = ":8080"
	defaultLogLevel    = "info"
)

type Config struct {
	Environment string
	HTTPAddr    string
	LogLevel    string
}

func Load() (Config, error) {
	cfg := Config{
		Environment: getEnv("SENTINEL_ENV", defaultEnvironment),
		HTTPAddr:    getEnv("SENTINEL_HTTP_ADDR", defaultHTTPAddr),
		LogLevel:    strings.ToLower(getEnv("SENTINEL_LOG_LEVEL", defaultLogLevel)),
	}

	if !strings.HasPrefix(cfg.HTTPAddr, ":") && !strings.Contains(cfg.HTTPAddr, ":") {
		return Config{}, fmt.Errorf("SENTINEL_HTTP_ADDR must be a host:port or :port value")
	}

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("unsupported SENTINEL_LOG_LEVEL %q", cfg.LogLevel)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
