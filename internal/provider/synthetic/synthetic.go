// Package synthetic provides a deterministic sine-wave metric provider for
// testing and demo scenarios.
package synthetic

import (
	"context"
	"math"
	"time"

	"github.com/mhightower/gopher-pulse/internal/provider"
)

// Provider emits a sine-wave value derived from the current time.
type Provider struct {
	amplitude float64
	periodSec float64
}

// New returns a synthetic Provider. amplitude sets the peak value and period
// sets the length of one full wave cycle.
func New(amplitude float64, period time.Duration) *Provider {
	return &Provider{
		amplitude: amplitude,
		periodSec: period.Seconds(),
	}
}

// Name returns the stable provider identifier.
func (p *Provider) Name() string {
	return "synthetic"
}

// Collect returns a single sine-wave measurement based on the current time.
// It returns immediately and always succeeds unless the context is already done.
func (p *Provider) Collect(ctx context.Context) ([]provider.Measurement, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	t := float64(time.Now().UnixNano()) / 1e9
	value := p.amplitude * math.Sin(2*math.Pi*t/p.periodSec)

	return []provider.Measurement{
		{
			Name:  "gopher_pulse_synthetic_wave",
			Value: value,
			Unit:  "1",
			Attributes: map[string]string{
				"provider": p.Name(),
			},
		},
	}, nil
}
