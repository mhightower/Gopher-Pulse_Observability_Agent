package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mhightower/gopher-pulse/internal/collector"
	"github.com/mhightower/gopher-pulse/internal/config"
	"github.com/mhightower/gopher-pulse/internal/health"
	"github.com/mhightower/gopher-pulse/internal/provider"
	githubprovider "github.com/mhightower/gopher-pulse/internal/provider/github"
	"github.com/mhightower/gopher-pulse/internal/provider/synthetic"
	"github.com/mhightower/gopher-pulse/internal/telemetry"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("fatal error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	startTime := time.Now()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	tel, err := telemetry.New()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := tel.Shutdown(shutdownCtx); err != nil {
			logger.Error("telemetry shutdown error", slog.String("error", err.Error()))
		}
	}()

	providers := []provider.Provider{
		githubprovider.New(cfg.Repo, logger),
		synthetic.New(10.0, time.Minute),
	}

	col := collector.New(providers, cfg.Interval, logger, tel.Meter("gopher-pulse"))

	mux := http.NewServeMux()
	mux.Handle("/metrics", tel.Handler)
	mux.Handle("/health", health.Handler(startTime))

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}

	// Start the listener before the collector so a port conflict fails fast.
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}

	go func() {
		logger.Info("metrics endpoint listening", slog.String("addr", cfg.Addr))
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server error", slog.String("error", err.Error()))
			cancel()
		}
	}()

	logger.Info("starting pulse-agent",
		slog.String("repo", cfg.Repo),
		slog.Duration("interval", cfg.Interval),
		slog.String("addr", cfg.Addr),
	)

	if err := col.Run(ctx); err != nil {
		logger.Error("collector error", slog.String("error", err.Error()))
	}

	logger.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("stopped")
	return nil
}
