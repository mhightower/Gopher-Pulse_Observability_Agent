package synthetic

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	p := New(1.0, time.Minute)
	if got := p.Name(); got != "synthetic" {
		t.Errorf("Name() = %q, want %q", got, "synthetic")
	}
}

func TestCollect(t *testing.T) {
	tests := []struct {
		name      string
		amplitude float64
		period    time.Duration
		ctx       func() context.Context
		wantErr   bool
		check     func(t *testing.T, value float64)
	}{
		{
			name:      "returns one measurement",
			amplitude: 10.0,
			period:    time.Minute,
			ctx:       context.Background,
			wantErr:   false,
			check: func(t *testing.T, value float64) {
				if math.Abs(value) > 10.0 {
					t.Errorf("value %f exceeds amplitude 10.0", value)
				}
			},
		},
		{
			name:      "value within amplitude bounds",
			amplitude: 50.0,
			period:    time.Minute,
			ctx:       context.Background,
			wantErr:   false,
			check: func(t *testing.T, value float64) {
				if value < -50.0 || value > 50.0 {
					t.Errorf("value %f outside [-50, 50]", value)
				}
			},
		},
		{
			name:      "cancelled context returns error",
			amplitude: 1.0,
			period:    time.Minute,
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.amplitude, tt.period)
			measurements, err := p.Collect(tt.ctx())

			if (err != nil) != tt.wantErr {
				t.Fatalf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if len(measurements) != 1 {
				t.Fatalf("Collect() returned %d measurements, want 1", len(measurements))
			}

			m := measurements[0]
			if m.Name != "gopher_pulse_synthetic_wave" {
				t.Errorf("Name = %q, want %q", m.Name, "gopher_pulse_synthetic_wave")
			}
			if m.Attributes["provider"] != "synthetic" {
				t.Errorf("provider attribute = %q, want %q", m.Attributes["provider"], "synthetic")
			}
			if tt.check != nil {
				tt.check(t, m.Value)
			}
		})
	}
}

func TestCollectDeterminism(t *testing.T) {
	// Two providers with the same amplitude and period should produce values
	// in the same bounded range, confirming the wave formula is consistent.
	p1 := New(1.0, 60*time.Second)
	p2 := New(1.0, 60*time.Second)

	m1, err := p1.Collect(context.Background())
	if err != nil {
		t.Fatalf("p1.Collect() error: %v", err)
	}
	m2, err := p2.Collect(context.Background())
	if err != nil {
		t.Fatalf("p2.Collect() error: %v", err)
	}

	// Both values must be within [-1, 1].
	for _, m := range []float64{m1[0].Value, m2[0].Value} {
		if m < -1.0 || m > 1.0 {
			t.Errorf("value %f outside amplitude bounds [-1, 1]", m)
		}
	}
}
