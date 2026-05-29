# A21 Phase 3A Metrics

## Purpose

Phase 3A adds the first Prometheus-compatible gateway metrics. This is the start of A21's "可观测可修复" runtime base, not the final latency dashboard.

## Endpoint

```text
GET http://127.0.0.1:21080/metrics
```

## Current Metrics

- `a21_mock_turn_total`
- `a21_barge_in_total`
- `a21_audio_frame_total`
- `a21_ws_connections_active{channel="control"}`
- `a21_ws_connections_active{channel="audio"}`

## Boundaries

Current metrics cover only mock gateway behavior. They do not yet cover:

- first-audio latency
- VAD timing
- provider first chunk
- v21 query timing
- device playback timing
- firmware reconnects
- proxy failures

Those should be added as the corresponding runtime path becomes real.
