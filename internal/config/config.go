package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// Config holds all runtime configuration for the agent.
type Config struct {
	Repo     string
	Interval time.Duration
	Addr     string
}

// Load parses CLI flags, falls back to environment variables, then hard-coded defaults.
// It returns an error if the resulting config fails validation.
func Load() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.Repo, "repo", env("PULSE_REPO", "golang/go"), "GitHub repository to monitor (owner/name)")
	flag.DurationVar(&cfg.Interval, "interval", envDuration("PULSE_INTERVAL", 15*time.Second), "Collection interval")
	flag.StringVar(&cfg.Addr, "addr", env("PULSE_ADDR", ":9464"), "Prometheus metrics listen address")
	flag.Parse()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Repo == "" {
		return fmt.Errorf("repo must not be empty")
	}
	if c.Interval <= 0 {
		return fmt.Errorf("interval must be positive, got %s", c.Interval)
	}
	if c.Addr == "" {
		return fmt.Errorf("addr must not be empty")
	}
	return nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}
