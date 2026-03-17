// Package provider defines the shared contract that all metric providers must satisfy.
package provider

import "context"

// Provider collects measurements from a single data source.
type Provider interface {
	// Name returns a stable lowercase identifier for this provider (e.g. "github", "synthetic").
	Name() string
	// Collect fetches the current measurements. It must respect context cancellation.
	Collect(ctx context.Context) ([]Measurement, error)
}

// Measurement is a single normalized data point returned by a provider.
type Measurement struct {
	Name       string
	Value      float64
	Unit       string
	Attributes map[string]string
}
