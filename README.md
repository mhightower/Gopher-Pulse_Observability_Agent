# Gopher-Pulse Observability Agent

Gopher-Pulse is a Go-based observability agent intended to collect metrics from multiple providers and expose them through OpenTelemetry for Prometheus scraping.

## Status

Phase 1 is complete and the agent is fully operational:

- OpenTelemetry Prometheus exporter setup
- GitHub provider for stars and open issue counts
- Synthetic provider for deterministic signal generation
- Structured JSON logging via `log/slog`
- Clean context-based shutdown

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

```bash
make run
curl http://localhost:9464/metrics
```
