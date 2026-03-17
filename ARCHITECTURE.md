# ARCHITECTURE.md: Gopher-Pulse Design Overview

This document captures the project narrative, architectural intent, and roadmap. For coding rules and generation defaults, use [AGENTS.md](./AGENTS.md).

## Executive Summary

Gopher-Pulse is a high-performance observability agent designed to demonstrate modern SRE principles. Instead of only checking service availability, it normalizes multiple data sources, including live GitHub API metrics and synthetic mathematical signals, into standardized OpenTelemetry metrics.

Project goals:

- Observability first through the OpenTelemetry Go SDK
- Resilient collection from external APIs with timeouts and bounded retry behavior
- Decoupled provider-based architecture that scales to additional data sources

## High-Level Architecture

The system follows a provider-collector pattern.

1. Providers fetch or generate raw data points.
2. The collector schedules providers and normalizes their output.
3. OpenTelemetry manages metric instruments and aggregation.
4. The Prometheus exporter exposes a scrape endpoint, typically on `:9464`.

This separation keeps data acquisition independent from metric export.

## Project Structure

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

## Phase 1 Data Sources

| Source | Type | Purpose | Instrument |
| :--- | :--- | :--- | :--- |
| GitHub | External API | Demonstrate I/O, error handling, and rate limiting | Gauge for stars and open issues |
| Synthetic | Mathematical signal | Provide deterministic load for testing and demo scenarios | Gauge for sine-wave output |

## Core Technical Decisions

- Providers should remain isolated behind a shared contract so new inputs can be added without collector refactors.
- Structured logs should use `log/slog` in JSON format.
- Concurrency should be managed centrally to support clean startup and shutdown.
- Telemetry setup should be explicit and owned by dedicated wiring code.

## Roadmap

Phase 1 (complete):

- OpenTelemetry Prometheus exporter.
- GitHub provider for stars and issue counts.
- Synthetic provider for sine-wave generation.

Phase 2:

- Add distributed tracing around outbound HTTP calls.
- Support dynamic configuration reload.
- Add health and SLI-oriented endpoints.
- Introduce histograms for latency distributions.
- Add local container orchestration for the agent and Prometheus.

## Development Flow

```bash
make build
./pulse-agent --repo="golang/go" --interval=15s
curl http://localhost:9464/metrics
```

## Notes

- Keep this file focused on architecture and intent.
- Keep implementation rules, defaults, and coding constraints in [AGENTS.md](./AGENTS.md).
