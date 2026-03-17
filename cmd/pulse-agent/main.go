package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

func main() {
	repo := flag.String("repo", "golang/go", "GitHub repository to monitor (owner/name)")
	interval := flag.Duration("interval", 15*time.Second, "Collection interval")
	addr := flag.String("addr", ":9464", "Prometheus metrics listen address")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	exporter, err := prometheus.New()
	if err != nil {
		logger.Error("failed to create prometheus exporter", slog.String("error", err.Error()))
		os.Exit(1)
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	defer func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			logger.Error("failed to shut down meter provider", slog.String("error", err.Error()))
		}
	}()

	logger.Info("starting pulse-agent",
		slog.String("repo", *repo),
		slog.Duration("interval", *interval),
		slog.String("addr", *addr),
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		logger.Info("metrics endpoint listening", slog.String("addr", *addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server error", slog.String("error", err.Error()))
			cancel()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("stopped")
}
