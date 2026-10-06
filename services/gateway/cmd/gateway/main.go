package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/config"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/database"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/httpserver"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/messaging"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/readiness"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg.DatabaseURL, cfg.DependencyTimeout)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	migrationCtx, cancelMigrations := context.WithTimeout(ctx, cfg.DependencyTimeout)
	if err := db.ApplyMigrations(migrationCtx); err != nil {
		cancelMigrations()
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}
	cancelMigrations()

	natsClient, err := messaging.Connect(ctx, cfg.NATSURL, cfg.DependencyTimeout)
	if err != nil {
		logger.Error("NATS connection failed", "error", err)
		os.Exit(1)
	}
	defer natsClient.Close()

	telemetryService := telemetry.NewService(natsClient)
	persistenceHandler := telemetry.PersistenceHandler(db)

	subscription, err := natsClient.StartTelemetryConsumer(
		ctx,
		persistenceHandler,
		func(err error) {
			logger.Warn("telemetry consumer issue", "error", err)
		},
	)
	if err != nil {
		logger.Error("telemetry persistence consumer failed to start", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := subscription.Unsubscribe(); err != nil {
			logger.Warn("telemetry subscription cleanup failed", "error", err)
		}
	}()

	readinessChecker := readiness.New(cfg.DependencyTimeout, map[string]readiness.CheckFunc{
		"postgres": db.Ping,
		"nats":     natsClient.Ping,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpserver.New(readinessChecker, telemetryService, db).Handler(),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("gateway starting",
			"address", cfg.HTTPAddr,
			"environment", cfg.Environment,
			"telemetry_subject", messaging.TelemetrySubject,
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
			return
		}

		serverErrors <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("gateway shutdown requested")
	case err := <-serverErrors:
		if err != nil {
			logger.Error("gateway server failed", "error", err)
			os.Exit(1)
		}
		return
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("gateway shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("gateway stopped")
}

func parseLogLevel(value string) slog.Level {
	switch value {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
