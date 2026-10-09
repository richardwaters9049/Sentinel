package config

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/access"
)

const (
	defaultEnvironment       = "development"
	defaultHTTPAddr          = "127.0.0.1:8080"
	defaultLogLevel          = "info"
	defaultDatabaseURL       = "postgres://sentinel:sentinel_dev_only@127.0.0.1:55432/sentinel?sslmode=disable"
	defaultNATSURL           = "nats://127.0.0.1:4222"
	defaultMLURL             = "http://127.0.0.1:8090"
	defaultDependencyWait    = 10 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
)

type Config struct {
	OTLPEndpoint      string
	ConsoleOrigin     string
	Access            *access.Verifier
	AuthMode          string
	Environment       string
	HTTPAddr          string
	LogLevel          string
	DatabaseURL       string
	NATSURL           string
	MLURL             string
	BehaviourEnabled  bool
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
		MLURL:             getEnv("SENTINEL_ML_URL", defaultMLURL),
		BehaviourEnabled:  getBoolEnv("SENTINEL_BEHAVIOUR_ENABLED", false),
		DependencyTimeout: getDurationEnv("SENTINEL_DEPENDENCY_TIMEOUT", defaultDependencyWait),
		ShutdownTimeout:   getDurationEnv("SENTINEL_SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
		ReadHeaderTimeout: getDurationEnv("SENTINEL_READ_HEADER_TIMEOUT", defaultReadHeaderTimeout),
	}
	if path := os.Getenv("DATABASE_URL_FILE"); path != "" {
		if os.Getenv("DATABASE_URL") != "" {
			return Config{}, fmt.Errorf("configure only one database URL source")
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return Config{}, fmt.Errorf("DATABASE_URL_FILE must be a private regular file")
		}
		file, err := os.Open(path)
		if err != nil {
			return Config{}, fmt.Errorf("database secret is unavailable")
		}
		opened, err := file.Stat()
		if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() || opened.Mode().Perm()&0077 != 0 {
			file.Close()
			return Config{}, fmt.Errorf("database secret changed while opening")
		}
		raw, err := io.ReadAll(io.LimitReader(file, 4097))
		file.Close()
		if err != nil || len(raw) > 4096 || len(strings.TrimSpace(string(raw))) == 0 {
			return Config{}, fmt.Errorf("invalid database secret")
		}
		cfg.DatabaseURL = strings.TrimSpace(string(raw))
	}

	cfg.ConsoleOrigin = getEnv("SENTINEL_CONSOLE_ORIGIN", "http://127.0.0.1:3000")
	origin, err := url.Parse(cfg.ConsoleOrigin)
	if err != nil || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || origin.Path != "" || (origin.Scheme != "http" && origin.Scheme != "https") {
		return Config{}, fmt.Errorf("SENTINEL_CONSOLE_ORIGIN must be an exact HTTP(S) origin")
	}
	if origin.Scheme == "http" && origin.Hostname() != "localhost" && (net.ParseIP(origin.Hostname()) == nil || !net.ParseIP(origin.Hostname()).IsLoopback()) {
		return Config{}, fmt.Errorf("non-local console sessions require an HTTPS origin")
	}
	cfg.AuthMode = getEnv("SENTINEL_AUTH_MODE", "required")
	if os.Getenv("SENTINEL_AUTH_MODE") == "" && cfg.Environment == "development" {
		cfg.AuthMode = "development"
	}
	switch cfg.AuthMode {
	case "development":
		if cfg.Environment != "development" {
			return Config{}, fmt.Errorf("development authentication mode requires SENTINEL_ENV=development")
		}
		if os.Getenv("SENTINEL_AUTH_CREDENTIALS_FILE") != "" {
			return Config{}, fmt.Errorf("credential file requires authentication mode required")
		}
	case "required":
		verifier, err := access.LoadFile(os.Getenv("SENTINEL_AUTH_CREDENTIALS_FILE"), time.Now().UTC())
		if err != nil {
			return Config{}, err
		}
		cfg.Access = verifier
		if !verifier.CollectorsReady() {
			return Config{}, fmt.Errorf("collector credentials require unique subjects and Ed25519 public keys")
		}
	default:
		return Config{}, fmt.Errorf("unsupported SENTINEL_AUTH_MODE")
	}
	cfg.OTLPEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if cfg.OTLPEndpoint != "" {
		endpoint, err := url.Parse(cfg.OTLPEndpoint)
		if err != nil || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Path != "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
			return Config{}, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT must be an HTTP(S) origin")
		}
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

	if cfg.BehaviourEnabled && strings.TrimSpace(cfg.MLURL) == "" {
		return Config{}, fmt.Errorf("SENTINEL_ML_URL must not be empty when behavioural analytics are enabled")
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

func getBoolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
