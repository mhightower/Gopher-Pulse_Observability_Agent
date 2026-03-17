// Package collector orchestrates provider execution and metric emission on a
// configurable schedule.
package collector

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/mhighto/gopher-pulse/internal/provider"
)

// Collector runs each registered provider on a fixed interval and records
// their measurements as OpenTelemetry gauge observations.
type Collector struct {
	providers []provider.Provider
	interval  time.Duration
	logger    *slog.Logger
	meter     metric.Meter
}

// New constructs a Collector. All arguments are required.
func New(providers []provider.Provider, interval time.Duration, logger *slog.Logger, meter metric.Meter) *Collector {
	return &Collector{
		providers: providers,
		interval:  interval,
		logger:    logger,
		meter:     meter,
	}
}

// Run starts the collection loop. It blocks until ctx is cancelled, then
// returns nil. Any provider error is logged and counted but does not stop the loop.
func (c *Collector) Run(ctx context.Context) error {
	c.logger.Info("collector started", slog.Duration("interval", c.interval))

	gauges, err := c.registerGauges(ctx)
	if err != nil {
		return fmt.Errorf("registering gauges: %w", err)
	}

	tick := time.NewTicker(c.interval)
	defer tick.Stop()

	// Collect immediately on start, then on each tick.
	c.collect(ctx, gauges)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("collector stopped")
			return nil
		case <-tick.C:
			c.collect(ctx, gauges)
		}
	}
}

// collect runs all providers once and records their measurements.
func (c *Collector) collect(ctx context.Context, gauges map[string]metric.Float64Gauge) {
	for _, p := range c.providers {
		measurements, err := p.Collect(ctx)
		if err != nil {
			c.logger.Warn("provider collection failed",
				slog.String("provider", p.Name()),
				slog.String("error", err.Error()),
			)
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

// registerGauges does a single dry-run collection to discover all metric names
// and pre-registers a Float64Gauge for each one.
func (c *Collector) registerGauges(ctx context.Context) (map[string]metric.Float64Gauge, error) {
	gauges := make(map[string]metric.Float64Gauge)

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
				return nil, fmt.Errorf("register gauge %q: %w", m.Name, err)
			}
			gauges[m.Name] = g
		}
	}

	return gauges, nil
}
