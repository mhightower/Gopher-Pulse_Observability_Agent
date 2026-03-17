package collector

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric/noop"

	"github.com/mhightower/gopher-pulse/internal/provider"
)

// stubProvider is a controllable provider for testing.
type stubProvider struct {
	name         string
	measurements []provider.Measurement
	err          error
	callCount    atomic.Int64
}

func (s *stubProvider) Name() string { return s.name }

func (s *stubProvider) Collect(_ context.Context) ([]provider.Measurement, error) {
	s.callCount.Add(1)
	if s.err != nil {
		return nil, s.err
	}
	return s.measurements, nil
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestCollectorRun_StopsOnContextCancel(t *testing.T) {
	stub := &stubProvider{
		name: "stub",
		measurements: []provider.Measurement{
			{Name: "gopher_pulse_test_value", Value: 1.0, Unit: "1"},
		},
	}

	c := New([]provider.Provider{stub}, 50*time.Millisecond, newLogger(), noop.NewMeterProvider().Meter("test"))

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- c.Run(ctx)
	}()

	// Let it tick a few times then cancel.
	time.Sleep(160 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not stop after context cancellation")
	}

	// Should have collected at least twice (immediate + at least one tick).
	if stub.callCount.Load() < 2 {
		t.Errorf("expected at least 2 collections, got %d", stub.callCount.Load())
	}
}

func TestCollectorRun_ProviderErrorDoesNotStop(t *testing.T) {
	failing := &stubProvider{
		name: "failing",
		err:  errors.New("upstream unavailable"),
	}
	good := &stubProvider{
		name: "good",
		measurements: []provider.Measurement{
			{Name: "gopher_pulse_test_value", Value: 42.0, Unit: "1"},
		},
	}

	c := New([]provider.Provider{failing, good}, 50*time.Millisecond, newLogger(), noop.NewMeterProvider().Meter("test"))

	ctx, cancel := context.WithTimeout(context.Background(), 160*time.Millisecond)
	defer cancel()

	err := c.Run(ctx)
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}

	// Good provider should still have been called despite the failing one.
	if good.callCount.Load() < 2 {
		t.Errorf("good provider called %d times, want at least 2", good.callCount.Load())
	}
}

func TestCollectorRun_CollectsImmediately(t *testing.T) {
	stub := &stubProvider{
		name: "stub",
		measurements: []provider.Measurement{
			{Name: "gopher_pulse_test_value", Value: 1.0, Unit: "1"},
		},
	}

	// Very long interval — only the immediate collection should fire before cancel.
	c := New([]provider.Provider{stub}, 10*time.Minute, newLogger(), noop.NewMeterProvider().Meter("test"))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = c.Run(ctx)

	// registerGauges does one collection, then collect() does another immediately.
	if stub.callCount.Load() < 2 {
		t.Errorf("expected immediate collection on start, got %d calls", stub.callCount.Load())
	}
}
