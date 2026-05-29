# A21 Phase 3A Metrics Implementation Plan

**Goal:** Add the first Prometheus-compatible gateway metrics without expanding into full tracing or latency dashboards yet.

## Scope

Phase 3A implements:

- `GET /metrics`
- mock turn counter
- barge-in/interrupt counter
- mock audio frame counter
- active WebSocket connection gauge by channel
- docs for the current metrics boundary

Phase 3A does not implement:

- OpenTelemetry traces
- latency histograms
- Grafana dashboards
- provider/v21 metrics
- firmware/device health metrics

## Tasks

- [x] Add failing `/metrics` tests.
- [x] Add Prometheus Go client dependency.
- [x] Register A21 gateway counters and gauges in a per-server registry.
- [x] Record HTTP mock flow and audio WebSocket metrics.
- [x] Document Phase 3A.
- [x] Run full verification and commit as `feat: add a21 gateway metrics`.
