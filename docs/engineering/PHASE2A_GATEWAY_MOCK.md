# A21 Phase 2A Gateway Mock

## Purpose

Phase 2A proves a no-hardware interaction loop without introducing WebSocket, browser UI, real audio, or real providers.

## Current Endpoints

- `GET /healthz`
- `POST /v1/mock-turn`
- `POST /v1/mock-interrupt`

## Run

```bash
make gateway
```

## Smoke

```bash
curl -s http://127.0.0.1:21080/healthz
curl -s -X POST http://127.0.0.1:21080/v1/mock-turn \
  -d '{"device_id":"stackchan-sim-001","text":"先说，我在","mode":"workmate"}'
```

## Boundaries

This phase does not claim real-time audio. It creates deterministic state/control events so Phase 2B can add WebSocket transport and a simulator against stable contracts.
