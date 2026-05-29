# ADR 0002: Prometheus Go Client For Gateway Metrics

## Status

Accepted for Phase 3A.

## Context

A21 requires runtime observability from the first gateway phases. The project should not invent its own metrics exposition format when Prometheus already defines the operational path used by Go services.

## Decision

Use `github.com/prometheus/client_golang` and expose gateway metrics through `promhttp.HandlerFor` with a per-server registry.

Primary-source checks on 2026-05-29:

- Prometheus Go instrumentation tutorial: https://prometheus.io/docs/tutorials/instrumenting_http_server_in_go/
- Go package docs for `promhttp`: https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/promhttp

Reasons:

- official Prometheus Go client
- standard `/metrics` exposition path
- `promhttp` integrates directly with `net/http`
- per-server registries avoid global metric registration collisions in tests

## Consequences

The gateway now owns a narrow observability dependency. Metrics must remain A21-namespaced and adapter-neutral.

Later phases should add latency histograms, provider labels, device labels, and trace correlation only after the mock gateway contracts are stable.
