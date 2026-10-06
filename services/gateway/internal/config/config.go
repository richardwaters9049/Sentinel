package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEnvironment       = "development"
	defaultHTTPAddr          = ":8080"
	defaultLogLevel          = "info"
	defaultDatabaseURL       = "postgres://sentinel:sentinel_dev_only@127.0.0.1:55432/sentinel?sslmode=disable"
	defaultNATSURL           = "nats://127.0.0.1:4222"
	defaultDependencyWait    = 10 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
)

type Config struct {
	Environment       string
	HTTPAddr          string
	LogLevel          string
	DatabaseURL       string
	NATSURL           string
	DependencyTimeout time.Duration
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Environment:       getEnv("SENTINEL_ENV", defaultEnvironment),
		HTTPAddr:          getEnv("SENTINEL_HTTP_ADDR", defaultHTTPAddr),
		LogLevel:          strings.ToLower(getEnv("SENTINEL_LOG_LEVEL", defaultLogLevel)),
		DatabaseURL:       getEnv("DATABASE_URL", defaultDatabaseURL),
		NATSURL:           getEnv("NATS_URL", defaultNATSURL),
		DependencyTimeout: getDurationEnv("SENTINEL_DEPENDENCY_TIMEOUT", defaultDependencyWait),
		ShutdownTimeout:   getDurationEnv("SENTINEL_SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
		ReadHeaderTimeout: getDurationEnv("SENTINEL_READ_HEADER_TIMEOUT", defaultReadHeaderTimeout),
	}

	if !strings.HasPrefix(cfg.HTTPAddr, ":") && !strings.Contains(cfg.HTTPAddr, ":") {
		return Config{}, fmt.Errorf("SENTINEL_HTTP_ADDR must be a host:port or :port value")
	}

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("unsupported SENTINEL_LOG_LEVEL %q", cfg.LogLevel)
	}

	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return Config{}, fmt.Errorf("DATABASE_URL must not be empty")
	}

	if strings.TrimSpace(cfg.NATSURL) == "" {
		return Config{}, fmt.Errorf("NATS_URL must not be empty")
	}

	if cfg.DependencyTimeout <= 0 {
		return Config{}, fmt.Errorf("SENTINEL_DEPENDENCY_TIMEOUT must be greater than zero")
	}

	if cfg.ShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("SENTINEL_SHUTDOWN_TIMEOUT must be greater than zero")
	}

	if cfg.ReadHeaderTimeout <= 0 {
		return Config{}, fmt.Errorf("SENTINEL_READ_HEADER_TIMEOUT must be greater than zero")
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

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err == nil {
		return duration
	}

	if seconds, convErr := strconv.Atoi(value); convErr == nil {
		return time.Duration(seconds) * time.Second
	}

	return -1
}
