package config

import (
	"os"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     Config{Repo: "golang/go", Interval: 15 * time.Second, Addr: ":9464"},
			wantErr: false,
		},
		{
			name:    "empty repo",
			cfg:     Config{Repo: "", Interval: 15 * time.Second, Addr: ":9464"},
			wantErr: true,
		},
		{
			name:    "zero interval",
			cfg:     Config{Repo: "golang/go", Interval: 0, Addr: ":9464"},
			wantErr: true,
		},
		{
			name:    "negative interval",
			cfg:     Config{Repo: "golang/go", Interval: -1 * time.Second, Addr: ":9464"},
			wantErr: true,
		},
		{
			name:    "empty addr",
			cfg:     Config{Repo: "golang/go", Interval: 15 * time.Second, Addr: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvFallback(t *testing.T) {
	t.Run("uses env var when set", func(t *testing.T) {
		os.Setenv("PULSE_REPO", "test/repo")
		defer os.Unsetenv("PULSE_REPO")

		got := env("PULSE_REPO", "default/repo")
		if got != "test/repo" {
			t.Errorf("env() = %q, want %q", got, "test/repo")
		}
	})

	t.Run("uses fallback when env not set", func(t *testing.T) {
		os.Unsetenv("PULSE_REPO")
		got := env("PULSE_REPO", "default/repo")
		if got != "default/repo" {
			t.Errorf("env() = %q, want %q", got, "default/repo")
		}
	})

	t.Run("uses duration env var when valid", func(t *testing.T) {
		os.Setenv("PULSE_INTERVAL", "30s")
		defer os.Unsetenv("PULSE_INTERVAL")

		got := envDuration("PULSE_INTERVAL", 15*time.Second)
		if got != 30*time.Second {
			t.Errorf("envDuration() = %v, want %v", got, 30*time.Second)
		}
	})

	t.Run("uses fallback when duration env is invalid", func(t *testing.T) {
		os.Setenv("PULSE_INTERVAL", "not-a-duration")
		defer os.Unsetenv("PULSE_INTERVAL")

		got := envDuration("PULSE_INTERVAL", 15*time.Second)
		if got != 15*time.Second {
			t.Errorf("envDuration() = %v, want %v", got, 15*time.Second)
		}
	})
}
