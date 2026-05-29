# A21 Phase 2B WebSocket Boundary

## Purpose

Phase 2B adds the first real-time transport boundary while keeping A21 on deterministic mock behavior. It is a transport milestone, not a real voice milestone.

## Dependency

Gateway WebSocket handling uses `github.com/coder/websocket` as accepted in `docs/engineering/adr/0001-websocket-library.md`.

## Endpoints

All Phase 2B endpoints run on the existing gateway address, default `127.0.0.1:21080`.

- `GET /healthz`
- `GET /v1/devices`
- `GET /v1/traces?trace_id=<trace_id>`
- `POST /v1/mock-turn`
- `POST /v1/mock-interrupt`
- `GET /ws/control`
- `GET /ws/audio`

## Control WebSocket

`/ws/control` accepts an A21 `device.event` envelope with `DeviceEventPayload`.

Supported mock events:

- `mock.turn`: emits `listening`, `thinking`, `speaking`
- `interrupt`: emits `interrupted`, `listening`
- `touch.wake_or_listen`: emits the same low-latency mock turn sequence while preserving semantic touch origin in payload metadata
- `touch.barge_in`: emits the same interruption sequence while preserving semantic touch origin in payload metadata

Trace and session IDs are propagated from the device event when supplied. If missing, the mock gateway assigns deterministic A21 IDs.

Gateway records a lightweight in-memory trace waterfall for mock transport events. This supports simulator and local debugging before OpenTelemetry or durable trace storage is introduced.

Firmware-originated device events can include firmware ID, version, board, and commit. Gateway stores the latest record in `GET /v1/devices`, marks identity as `ok`, `unknown`, or `invalid`, and returns an error control event when a provided firmware identity contains forbidden X21/V21 naming or mismatches the A21 StackChan target.

## Audio WebSocket

`/ws/audio` accepts an A21 `audio.frame` envelope with an `AudioChunk` payload and returns one mock `control.event` ack in `listening` state. This proves the audio transport boundary exists without claiming real PCM processing.

Later firmware work now consumes this boundary with a guarded mock `audio.frame` sender. That does not change the Phase 2B claim: real capture, playback, VAD, and full-duplex media behavior remain outside this boundary milestone.

## Boundaries

Phase 2B itself did not implement the browser simulator. Phase 2C now adds the first built-in simulator against these WebSocket endpoints.

Phase 2B still does not implement:

- real audio capture
- real audio playback
- VAD
- jitter buffer
- provider streaming
- production firmware media integration

Those belong to later phases after this transport contract is stable.
