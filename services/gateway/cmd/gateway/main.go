package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/config"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/httpserver"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	logger := log.New(os.Stdout, "sentinel-gateway ", log.LstdFlags|log.LUTC)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpserver.New().Handler(),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Printf("starting address=%s environment=%s log_level=%s", cfg.HTTPAddr, cfg.Environment, cfg.LogLevel)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Printf("server failed error=%q", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Print("shutdown requested")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown failed error=%q", err)
		os.Exit(1)
	}

	logger.Print("stopped")
}
