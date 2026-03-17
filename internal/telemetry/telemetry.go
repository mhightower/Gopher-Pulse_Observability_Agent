// Package telemetry sets up the OpenTelemetry meter provider and Prometheus exporter.
package telemetry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Provider holds the OTel MeterProvider and the HTTP handler for Prometheus scraping.
type Provider struct {
	MeterProvider *sdkmetric.MeterProvider
	Handler       http.Handler
}

// New creates a Prometheus-backed OTel MeterProvider and returns the scrape handler.
func New() (*Provider, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))

	return &Provider{
		MeterProvider: mp,
		Handler:       promhttp.Handler(),
	}, nil
}

// Meter returns a named Meter from the underlying provider.
func (p *Provider) Meter(name string) metric.Meter {
	return p.MeterProvider.Meter(name)
}

// Shutdown flushes and stops the meter provider. Call this on process exit.
func (p *Provider) Shutdown(ctx context.Context) error {
	if err := p.MeterProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown meter provider: %w", err)
	}
	return nil
}
