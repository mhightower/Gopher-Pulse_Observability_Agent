// Package collector orchestrates provider execution and metric emission on a
// configurable schedule.
package collector

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/mhightower/gopher-pulse/internal/provider"
)

// Collector runs each registered provider on a fixed interval and records
// their measurements as OpenTelemetry gauge observations.
type Collector struct {
	providers []provider.Provider
	interval  time.Duration
	logger    *slog.Logger
	meter     metric.Meter
	startTime time.Time
}

// New constructs a Collector. All arguments are required.
func New(providers []provider.Provider, interval time.Duration, logger *slog.Logger, meter metric.Meter) *Collector {
	return &Collector{
		providers: providers,
		interval:  interval,
		logger:    logger,
		meter:     meter,
		startTime: time.Now(),
	}
}

// Run starts the collection loop. It blocks until ctx is cancelled, then
// returns nil. Any provider error is logged and counted but does not stop the loop.
func (c *Collector) Run(ctx context.Context) error {
	c.logger.Info("collector started", slog.Duration("interval", c.interval))

	gauges, errorCounter, err := c.registerInstruments(ctx)
	if err != nil {
		return fmt.Errorf("registering instruments: %w", err)
	}

	tick := time.NewTicker(c.interval)
	defer tick.Stop()

	// Collect immediately on start, then on each tick.
	c.collect(ctx, gauges, errorCounter)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("collector stopped")
			return nil
		case <-tick.C:
			c.collect(ctx, gauges, errorCounter)
		}
	}
}

// collect runs all providers once and records their measurements.
func (c *Collector) collect(ctx context.Context, gauges map[string]metric.Float64Gauge, errorCounter metric.Int64Counter) {
	// Record agent uptime.
	if g, ok := gauges["gopher_pulse_agent_uptime_seconds"]; ok {
		g.Record(ctx, time.Since(c.startTime).Seconds(),
			metric.WithAttributes(attribute.String("provider", "agent")),
		)
	}

	for _, p := range c.providers {
		measurements, err := p.Collect(ctx)
		if err != nil {
			c.logger.Warn("provider collection failed",
				slog.String("provider", p.Name()),
				slog.String("error", err.Error()),
			)
			errorCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("provider", p.Name())))
			continue
		}

		for _, m := range measurements {
			g, ok := gauges[m.Name]
			if !ok {
				c.logger.Warn("no gauge registered for measurement", slog.String("name", m.Name))
				continue
			}
			g.Record(ctx, m.Value)
			c.logger.Info("recorded measurement",
				slog.String("provider", p.Name()),
				slog.String("name", m.Name),
				slog.Float64("value", m.Value),
			)
		}
	}
}

// registerInstruments does a single dry-run collection to discover all metric
// names and pre-registers a Float64Gauge for each one, plus the agent-level
// uptime gauge and provider error counter.
func (c *Collector) registerInstruments(ctx context.Context) (map[string]metric.Float64Gauge, metric.Int64Counter, error) {
	gauges := make(map[string]metric.Float64Gauge)

	// Register the agent uptime gauge.
	uptimeGauge, err := c.meter.Float64Gauge(
		"gopher_pulse_agent_uptime_seconds",
		metric.WithDescription("Seconds since the agent started"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register gauge %q: %w", "gopher_pulse_agent_uptime_seconds", err)
	}
	gauges["gopher_pulse_agent_uptime_seconds"] = uptimeGauge

	// Register the provider error counter.
	errorCounter, err := c.meter.Int64Counter(
		"gopher_pulse_provider_errors_total",
		metric.WithDescription("Total number of provider collection errors"),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register counter %q: %w", "gopher_pulse_provider_errors_total", err)
	}

	// Dry-run each provider to discover dynamic gauge names.
	for _, p := range c.providers {
		measurements, err := p.Collect(ctx)
		if err != nil {
			c.logger.Warn("provider unavailable during gauge registration, skipping",
				slog.String("provider", p.Name()),
				slog.String("error", err.Error()),
			)
			continue
		}

		for _, m := range measurements {
			if _, exists := gauges[m.Name]; exists {
				continue
			}
			g, err := c.meter.Float64Gauge(m.Name, metric.WithUnit(m.Unit))
			if err != nil {
				return nil, nil, fmt.Errorf("register gauge %q: %w", m.Name, err)
			}
			gauges[m.Name] = g
		}
	}

	return gauges, errorCounter, nil
}
