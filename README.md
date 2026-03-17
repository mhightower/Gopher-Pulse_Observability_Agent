# Gopher-Pulse Observability Agent

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

Gopher-Pulse is a Go-based observability agent intended to collect metrics from multiple providers and expose them through OpenTelemetry for Prometheus scraping.

## Status

The agent is fully operational with self-observability, a health endpoint, CI/CD, and a containerised full-stack deployment option:

- OpenTelemetry Prometheus exporter setup
- GitHub provider for stars and open issue counts
- Synthetic provider for deterministic signal generation
- Self-observability metrics: agent uptime gauge and per-provider error counter
- Health endpoint (`/health`) reporting uptime and liveness
- Structured JSON logging via `log/slog`
- Clean context-based shutdown
- Grafana + Prometheus local observability stack via Docker Compose
- Full containerised stack (agent + Prometheus + Grafana) via `docker-compose.full.yml`
- GitHub Actions CI: test, lint, and +80% coverage gate
- `.golangci.yml` with pinned linter set

## Why This Exists

The project is meant to demonstrate a practical observability agent with an emphasis on:

- Observability-first design using the OpenTelemetry Go SDK
- Resilient collection from external systems
- Extensible provider-based design
- Clean shutdown and predictable runtime behavior
- Testable, maintainable Go code

## Documentation Map

- [ARCHITECTURE.md](./ARCHITECTURE.md): System design summary, planned structure, and roadmap
- [AGENTS.md](./AGENTS.md): Implementation rules and coding guidance for agent-driven development
- [CLAUDE.md](./CLAUDE.md): Redirects Claude-style tooling to the canonical implementation guide

## Quick Start

### Run the agent locally

```bash
make run          # build and run (foreground)
make stop         # kill the agent by port
```

Endpoints while the agent is running:

- **Metrics:** `http://localhost:9464/metrics`
- **Health:** `http://localhost:9464/health`

### Start the observability stack (host agent + Grafana + Prometheus)

This starts Prometheus and Grafana in Docker and scrapes the agent running on the host.

```bash
make stack-up     # start containers in the background
make stack-down   # stop and remove containers
make stack-logs   # tail container logs
```

- **Grafana:** `http://localhost:3000` — login `admin` / `admin`
  - The **Gopher-Pulse** dashboard is pre-provisioned under Dashboards
- **Prometheus:** `http://localhost:9090`

### Start the full containerised stack (agent + Prometheus + Grafana)

Builds the agent image and starts all three services in Docker.

```bash
make stack-full-up    # docker build + compose up
make stack-full-down  # stop and remove containers
```

### Other commands

```bash
make build        # compile the binary
make test         # run all tests
make coverage     # run tests with coverage report
make lint         # run golangci-lint
make fmt          # gofmt + goimports
make clean        # remove binary and coverage artifacts
make docker-build # build the Docker image only
```

## License

This project is licensed under the [MIT License](./LICENSE).

Copyright (c) 2026 Marcus Hightower
