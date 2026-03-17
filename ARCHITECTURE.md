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
│   ├── health/
│   ├── provider/
│   │   ├── github/
│   │   └── synthetic/
│   ├── telemetry/
│   └── config/
├── deployments/
│   ├── prometheus/
│   │   ├── prometheus.yml          # host-agent scrape target
│   │   └── prometheus.full.yml     # container scrape target (pulse-agent:9464)
│   └── grafana/
│       └── provisioning/
├── .github/
│   └── workflows/
│       └── ci.yml
├── Dockerfile
├── docker-compose.yml              # Prometheus + Grafana only (agent on host)
├── docker-compose.full.yml         # agent + Prometheus + Grafana (fully containerised)
├── .golangci.yml
├── Makefile
├── go.mod
├── AGENTS.md
└── ARCHITECTURE.md
```

## Data Sources

| Source | Type | Purpose | Instrument |
| :--- | :--- | :--- | :--- |
| GitHub | External API | Demonstrate I/O, error handling, and rate limiting | Gauge for stars and open issues |
| Synthetic | Mathematical signal | Provide deterministic load for testing and demo scenarios | Gauge for sine-wave output |

## Self-Observability Metrics

The agent instruments itself in addition to external providers:

| Metric | Type | Description |
| :--- | :--- | :--- |
| `gopher_pulse_agent_uptime_seconds` | Gauge | Seconds since the agent started; attribute `provider=agent` |
| `gopher_pulse_provider_errors_total` | Counter | Total provider collection errors; attribute `provider=<name>` |

Note: the OTel Prometheus exporter appends `_ratio` to gauge metric names, so `gopher_pulse_agent_uptime_seconds` appears as `gopher_pulse_agent_uptime_seconds_ratio` in Prometheus scrape output.

## HTTP Endpoints

| Path | Description |
| :--- | :--- |
| `/metrics` | Prometheus scrape endpoint (OpenTelemetry exporter) |
| `/health` | Liveness endpoint — always 200, returns `{"status":"ok","uptime":"<duration>"}` |

## Core Technical Decisions

- Providers should remain isolated behind a shared contract so new inputs can be added without collector refactors.
- Structured logs should use `log/slog` in JSON format.
- Concurrency should be managed centrally to support clean startup and shutdown.
- Telemetry setup should be explicit and owned by dedicated wiring code.
- The health handler lives in `internal/health` and has no dependency on telemetry.

## Roadmap

Completed:

- OpenTelemetry Prometheus exporter.
- GitHub provider for stars and issue counts.
- Synthetic provider for sine-wave generation.
- Local observability stack: Prometheus + Grafana via Docker Compose with a pre-provisioned dashboard.
- Self-observability metrics (uptime gauge, provider error counter).
- `/health` liveness endpoint.
- Full containerised stack via `docker-compose.full.yml` and multi-stage `Dockerfile`.
- GitHub Actions CI with test, lint, and 75% coverage gate.
- `.golangci.yml` with pinned linter set.

Phase 2:

- Add distributed tracing around outbound HTTP calls.
- Support dynamic configuration reload.
- Introduce histograms for latency distributions.

## Development Flow

### Agent (host)

```bash
make run          # build and run the agent (foreground)
make stop         # kill the agent (frees :9464)
```

### Local observability stack (agent on host)

```bash
make stack-up     # start Prometheus + Grafana in Docker
make stack-down   # stop and remove containers
make stack-logs   # tail container logs
```

| Service    | URL                           | Credentials   |
| :--------- | :---------------------------- | :------------ |
| Grafana    | http://localhost:3000         | admin / admin |
| Prometheus | http://localhost:9090         | —             |
| Agent      | http://localhost:9464/metrics | —             |
| Agent      | http://localhost:9464/health  | —             |

### Full containerised stack

```bash
make stack-full-up    # docker build + all three services
make stack-full-down  # stop and remove containers
```

Grafana is provisioned automatically with the Prometheus datasource and a **Gopher-Pulse** dashboard. Start the agent (or full stack) first — Prometheus begins scraping on the first 15-second interval.

## Notes

- Keep this file focused on architecture and intent.
- Keep implementation rules, defaults, and coding constraints in [AGENTS.md](./AGENTS.md).
