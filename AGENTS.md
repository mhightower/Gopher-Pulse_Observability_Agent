# AGENTS.md: Gopher-Pulse Guidance for Agentic Code Generators

This file is the primary implementation guide for coding agents working in this repository. Use it to make generation decisions. Keep changes minimal, idiomatic Go, and aligned with the architecture summary in [ARCHITECTURE.md](./ARCHITECTURE.md).

## Project Context

Gopher-Pulse is a Go observability agent that collects metrics from multiple providers and exposes them through OpenTelemetry and a Prometheus scrape endpoint.

Primary stack and defaults:

- Go 1.25+
- OpenTelemetry Go SDK for metrics
- Prometheus exporter for scraping
- Podman and compose files for local runtime
- `log/slog` for structured logs

When the codebase is still being scaffolded, use the package layout and rules in this file instead of inventing a different structure.

## Priorities

Prioritize work in this order:

1. Correctness and clean shutdown behavior.
2. Observable behavior with consistent metric naming and structured logs.
3. Testability through small interfaces and injected dependencies.
4. Minimal public API surface.
5. Straightforward package boundaries under `internal/`.

## Repository Layout

Unless the user asks for a different shape, use this project structure when creating code:

```text
.
├── cmd/
│   └── pulse-agent/
│       └── main.go
├── internal/
│   ├── collector/
│   ├── provider/
│   │   ├── github/
│   │   └── synthetic/
│   ├── telemetry/
│   └── config/
├── Makefile
├── go.mod
├── AGENTS.md
└── ARCHITECTURE.md
```

Placement rules:

- Put the executable entrypoint in `cmd/pulse-agent`.
- Put application logic in `internal/`.
- Keep provider-specific code inside `internal/provider/<name>`.
- Keep OpenTelemetry setup, exporter wiring, and metric registration in `internal/telemetry`.
- Keep configuration structs and loading logic in `internal/config`.
- Do not create cross-package utility dumping grounds.

## Build and Test Commands

Once the project is scaffolded with `go.mod`, use these commands:

```bash
# Build the binary
make build
# or: go build -o pulse-agent ./cmd/pulse-agent

# Run all tests
make test
# or: go test ./...

# Run a single test by name
make test TEST=TestCollectorSchedule
# or: go test ./internal/collector -run TestCollectorSchedule -v

# Run tests with coverage
make coverage
# or: go test -cover ./...

# Lint the codebase
make lint
# or: golangci-lint run (requires golangci-lint installed)

# Format code
make fmt
# or: gofmt -w . && goimports -w .

# Run locally with defaults
./pulse-agent --repo="golang/go" --interval=15s

# Verify metrics are exposed
curl http://localhost:9464/metrics
```

## Go Conventions

Use standard Go patterns by default:

- Prefer small structs with explicit constructors such as `collector.New` and `provider.New`.
- Keep types and functions unexported unless another package must use them.
- Wrap errors with `fmt.Errorf("context: %w", err)`.
- Accept `context.Context` on operations that block, poll, fetch, or export.
- Do not use package-level mutable globals for configuration, logger, clients, or meters.
- Prefer composition over inheritance-style abstraction.
- Use table-driven tests for non-trivial branching.

### Imports

- Group imports in three blocks: stdlib, external, then internal. Blank line between groups.
- Use `gofmt` and `goimports` to keep imports organized automatically.

```go
import (
	"context"
	"fmt"
	
	"go.opentelemetry.io/otel"
	
	"github.com/mhighto/gopher-pulse/internal/provider"
)
```

### Formatting and Naming

- Follow Go naming conventions: `CamelCase` for exported, `camelCase` for unexported.
- Use short variable names for loop counters and temporary values.
- Metric names: use snake_case with `gopher_pulse_` prefix (e.g., `gopher_pulse_github_stars`).
- Provider names: use lowercase stable identifiers (e.g., `github`, `synthetic`).

### Types and Interfaces

- Keep interfaces small. Prefer specific interfaces close to usage.
- Use explicit constructors like `collector.New(...)` and `provider.New(...)`.
- Accept `context.Context` on all I/O operations: fetching, polling, exporting.
- Do not use package-level global variables for configuration, loggers, clients, or meters.
- Prefer composition over inheritance-style abstraction.

Example:

```go
type Collector struct {
	providers []provider.Provider
	interval  time.Duration
	logger    *slog.Logger
	meter     metric.Meter
}

func New(providers []provider.Provider, interval time.Duration, logger *slog.Logger, meter metric.Meter) *Collector {
	return &Collector{providers, interval, logger, meter}
}

func (c *Collector) Run(ctx context.Context) error {
	// Uses context for cancellation and timeouts
}
```

### Error Handling

- Wrap errors with context: `fmt.Errorf("loading config: %w", err)`.
- Return errors instead of hiding them. Do not silently log and continue without signaling failure.
- Use separate error metrics for transient vs. fatal failures.
- Validate configuration at startup. Invalid config is a fatal error.

```go
if err != nil {
	return fmt.Errorf("fetch github data: %w", err)
}
```

### Logging

- Use `log/slog` with structured fields in JSON format.
- Inject the logger into components that need it.
- Include `provider` field in all provider logs.
- Include collection interval for scheduler logs when relevant.
- Log external transient failures at `Warn` unless the process cannot continue.
- Log programmer errors and invalid configuration as `Error` and return them.
- Avoid noisy logs inside tight loops unless they are debug-only.

```go
logger.Info("collection started", slog.String("provider", "github"), slog.Duration("interval", 15*time.Second))
logger.Warn("github api rate limited", slog.Int("retry_after", 3600))
logger.Error("invalid config", slog.String("reason", "missing repo flag"))
```

## Testing Strategy

Default to TDD. Target 85%+ coverage for non-trivial code.

### Test Guidelines

- Unit-test providers with mocked transports. Do not make live API calls.
- Use table-driven tests for parsing, validation, and mapping logic.
- Test context cancellation and shutdown paths.
- Test error propagation and retry boundaries.
- Test metric registration and attribute cardinality.

Example:

```go
func TestGitHubProviderCollect(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockGitHubClient
		want    []Measurement
		wantErr bool
	}{
		{
			name:    "successful collection",
			client:  &mockGitHubClient{stars: 25000},
			want:    []Measurement{{Name: "stars", Value: 25000}},
			wantErr: false,
		},
		{
			name:    "network error",
			client:  &mockGitHubClient{err: io.EOF},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test implementation
		})
	}
}
```

### Run Single Test

```bash
go test ./internal/provider/github -run TestCollect -v
```

## Provider Contract

All providers must implement this shape:

```go
type Provider interface {
	Name() string
	Collect(ctx context.Context) ([]Measurement, error)
}

type Measurement struct {
	Name       string
	Value      float64
	Unit       string
	Attributes map[string]string
}
```

Provider rules:

- `Name()` returns a stable provider identifier such as `github` or `synthetic`.
- `Collect()` must respect context cancellation and return quickly when the context is done.
- Providers return normalized measurements, not OpenTelemetry instruments.
- Providers should be stateless where practical. If state is required, keep it private to the provider package.
- Providers must not start unmanaged background goroutines.
- Providers must not log and swallow errors silently. Return the error to the collector with wrapping context.

## Collector Rules

The collector orchestrates provider execution and metric emission.

- The collector owns the scrape loop and scheduling.
- Use a single parent context for coordinated shutdown.
- Stop loops with context cancellation, not ad hoc boolean flags.
- Keep provider polling intervals configurable.
- Validate provider results before emitting metrics.
- Separate provider collection from telemetry export logic.

Default lifecycle:

1. Load configuration.
2. Build logger, telemetry, and providers.
3. Start the collector with a parent context.
4. Expose the Prometheus endpoint.
5. On signal, cancel the context and wait for goroutines to exit.

## Telemetry Rules

Observability is the main product feature. Do not treat telemetry as an afterthought.

Metric naming rules:

- Use the `gopher_pulse_` prefix for repo-defined metrics.
- Use snake_case metric names.
- Prefer stable names over dynamically constructed names.
- Keep units explicit where applicable.

Metric examples:

- `gopher_pulse_github_stars`
- `gopher_pulse_github_open_issues`
- `gopher_pulse_synthetic_wave`

Attribute rules:

- Keep attributes low cardinality.
- Safe defaults are `provider`, `repo`, `status`, and `source_type`.
- Do not use request IDs, timestamps, URLs, or arbitrary user input as metric attributes.
- If an attribute can grow without bound, do not add it.

Instrumentation rules:

- Create meter providers and exporter wiring in `internal/telemetry`.
- Register instruments once during startup, not on every collection cycle.
- Record provider failures in both logs and error metrics.
- Use histograms for duration and counters for failures.
- Use gauges for current point-in-time values such as stars or synthetic wave values.

## Error Handling and Retries

Use a predictable error strategy.

- Wrap and return errors instead of hiding them.
- Treat invalid configuration as a startup error.
- Treat transient upstream failures as recoverable unless repeated failure should trip health status.
- For HTTP providers, use context timeouts and bounded retries with backoff.
- Do not retry forever.
- Distinguish transport errors, rate limiting, and response decoding failures in logs.

Default retry behavior for external APIs:

- Maximum 3 attempts per collection cycle.
- Exponential backoff with jitter.
- Respect rate-limit responses and surface them clearly.
- If retries fail, log the failure and continue the next scheduled cycle.

## Configuration Rules

Use explicit configuration flow.

- Keep config structs in `internal/config`.
- Parse config in one place, validate it, and pass it through constructors.
- Avoid reading environment variables deep inside provider or collector packages.
- Support sensible defaults for local development.
- Make interval, listen address, GitHub repository targets, and timeout values configurable.

If both CLI flags and environment variables exist, use this precedence unless the user says otherwise:

1. CLI flags
2. Environment variables
3. Hard-coded defaults

## Generation Defaults

When generating code in this repo, prefer these defaults unless the user asks otherwise:

- Inject dependencies through constructors.
- Keep interfaces close to where they are consumed.
- Add only the minimum comments needed to explain non-obvious behavior.
- Avoid speculative abstractions.
- Prefer one clear implementation over multiple extension points.
- Do not create packages until there is a real need.
- Keep the first version of a provider simple and make failure modes explicit.

## What To Avoid

Avoid these patterns unless the user explicitly requests them:

- Global singletons for config, logger, or telemetry state
- Reflection-heavy configuration systems
- Dynamic metric names derived from runtime data
- Background goroutines owned by providers without collector supervision
- Test suites that depend on network access
- Excessive package splitting for a small codebase

## Delivery Expectations

When implementing or modifying code:

1. Update tests with behavior changes.
2. Keep documentation aligned with the actual package structure.
3. Prefer minimal diffs over broad refactors.
4. If the requested change conflicts with these defaults, follow the user request and document the deviation in the response.
