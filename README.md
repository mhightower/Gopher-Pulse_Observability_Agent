# Gopher-Pulse Observability Agent

Gopher-Pulse is a Go-based observability agent intended to collect metrics from multiple providers and expose them through OpenTelemetry for Prometheus scraping.

The repository is currently in a documentation-first stage. The implementation has not been scaffolded yet, but the project direction and coding rules are defined.

## Status

- Documentation and implementation guidance are in place
- Runtime code, tests, and deployment artifacts are still to be created

The first implementation phase is expected to cover:

- OpenTelemetry Prometheus exporter setup
- GitHub provider for stars and open issue counts
- Synthetic provider for deterministic signal generation
- Local runtime support for the agent and Prometheus

## Why This Exists

The project is meant to demonstrate a practical observability agent with an emphasis on:

- Observability-first design using the OpenTelemetry Go SDK
- Resilient collection from external systems
- Extensible provider-based design
- Clean shutdown and predictable runtime behavior
- Testable, maintainable Go code

## Documentation Map

- [ARCHITECTURE.md](./ARCHITECTURE.md): System design summary, planned structure, and roadmap
- [AGENT.md](./AGENT.md): Implementation rules and coding guidance for agent-driven development
- [CLAUDE.md](./CLAUDE.md): Redirects Claude-style tooling to the canonical implementation guide

## Next Steps

The next logical step is to scaffold the Go project and begin the first implementation phase described in [ARCHITECTURE.md](./ARCHITECTURE.md).
