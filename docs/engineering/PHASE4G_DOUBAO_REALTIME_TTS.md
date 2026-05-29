# A21 Phase 4G Doubao Realtime TTS Boundary

## Purpose

Phase 4G adds a Doubao realtime TTS WebSocket boundary for A21's China-mainland provider lane.

This is not the end-to-end Doubao speech-to-speech companion path. It is a provider-specific TTS event mapper and dry-run connection plan for the edge realtime TTS API shape. A21 keeps this separate from `doubao_realtime` S2S so the project does not accidentally treat TTS text events as full-duplex audio conversation.

## Source Baseline

Volcengine's edge realtime TTS WebSocket documentation describes a WebSocket endpoint at `wss://ai-gateway.vei.volces.com/v1/realtime?model=...`, initialized with `tts_session.update`, streaming text through `input_text.append` and `input_text.done`, and receiving audio through events such as `response.audio.delta`.

Official docs used:

- https://www.volcengine.com/docs/6561/1786928

## Implemented Boundary

Code:

- `internal/providers/doubao_realtime_tts.go`
- `internal/providers/doubao_realtime_tts_test.go`
- `internal/providers/realtime.go`
- `internal/providers/catalog.go`

New provider catalog entry:

- `doubao_tts_realtime`

Required env names:

- `A21_DOUBAO_API_KEY`
- `A21_DOUBAO_TTS_MODEL`
- `A21_DOUBAO_TTS_VOICE`

Capabilities:

- Builds a redacted realtime WebSocket plan for Doubao realtime TTS.
- Reports only endpoint host and env names; never prints API key, model, voice, auth headers, or full URL.
- Maps A21-side text into Doubao TTS events:
  - `tts_session.update`
  - `input_text.append`
  - `input_text.done`
- Maps Doubao `response.audio.delta` into A21 `VoiceEvent` audio chunks.
- Reuses the existing `RealtimeWebSocketAdapter` test seam; tests use fake connections only.

## Safety Rules

- No real Doubao network call is made by tests or doctor.
- `provider-smoke --execute` remains unsupported for realtime WebSocket providers.
- `doubao_tts_realtime` is TTS-only and must not be used as the companion full-duplex S2S adapter.
- StackChan firmware never sees Doubao event names, API keys, model names, or voice IDs.
- Gateway-facing code should continue to consume A21 voice/audio events only.

## Non-Goals

Phase 4G does not:

- implement Doubao S2S
- implement RTC room/token handling
- switch Gateway traffic to Doubao
- stream real office microphone audio
- execute paid provider smoke
- implement retry/backoff/cost accounting

## Test Coverage

Covered by:

```bash
go test ./internal/providers -run 'TestRealtimeWebSocketPlanFromEnvBuildsDoubaoTTS|TestRealtimeWebSocketPlanFromEnvSkipsDoubaoTTS|TestDoubaoRealtimeTTS|TestProviderCatalogReportsDoubaoTTS' -count=1
go test ./internal/providers
```

The tests cover redacted readiness planning, required env handling, provider catalog readiness, documented event shapes, blank text rejection, server audio delta mapping, and fake-session event ordering.

## Next Step

The next provider slice should add a `DoubaoRealtimeTTSProvider` wrapper with `StartTurn` explicitly scoped to text-to-TTS professional/audio playback. The companion fast path still needs a separate Doubao S2S design based on the official realtime voice/RTC product contract.
