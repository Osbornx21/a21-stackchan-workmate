# A21 Phase 4H Doubao Realtime TTS Provider Wrapper

## Purpose

Phase 4H wraps the Doubao realtime TTS event mapper in a provider-owned session boundary.

This is still not the A21 companion fast path. It gives A21 a configured, health-checkable Doubao TTS provider object and an explicit realtime TTS session for text-to-audio work. The Gateway is not switched to this provider, and ordinary text turns cannot accidentally dial Doubao.

## Implemented Boundary

Code:

- `internal/providers/doubao_realtime_tts_provider.go`
- `internal/providers/doubao_realtime_tts_provider_test.go`

Capabilities:

- Builds `DoubaoRealtimeTTSProvider` from:
  - `A21_DOUBAO_API_KEY`
  - `A21_DOUBAO_TTS_MODEL`
  - `A21_DOUBAO_TTS_VOICE`
  - optional `A21_DOUBAO_TTS_OUTPUT_FORMAT`
  - optional `A21_DOUBAO_TTS_SAMPLE_RATE_HZ`
- Reports health without printing API keys, model values, voice IDs, auth headers, or full provider URLs.
- Refuses to start when required Doubao env is missing.
- Opens an explicit realtime TTS session through an injected `RealtimeDialer`.
- Sends the provider-correct first event, `tts_session.update`, not OpenAI-style `session.update`.
- Sends text with `input_text.append` and completes text with `input_text.done`.
- Maps server audio events back through the Phase 4G A21 voice-event mapper.
- Keeps ordinary `StartTurn` disabled so Gateway text paths cannot accidentally dial Doubao.

## Safety Rules

- `StartRealtimeTTSSession` is the only active session entrypoint.
- `StartTurn` returns an error and does not dial the provider.
- Tests use injected fake WebSocket connections only.
- `doctor` remains dry-run and does not execute provider calls.
- `provider-smoke --execute` still does not execute realtime WebSocket providers.
- StackChan firmware never sees provider env, provider headers, voice IDs, or provider event names.

## Non-Goals

Phase 4H does not:

- switch Gateway traffic to Doubao
- implement Doubao S2S
- execute paid provider smoke
- stream live office microphone audio
- add provider retry/backoff
- add billing/cost accounting

## Test Coverage

Covered by:

```bash
go test ./internal/providers -run 'TestDoubaoRealtimeTTSProvider' -count=1
go test ./internal/providers
```

The tests cover configured health redaction, missing env handling, explicit session lifecycle, provider-correct event ordering, no auth leakage in provider JSON, and the guard that prevents ordinary `StartTurn` from dialing Doubao.

## Next Step

Phase 4I adds an explicit `provider-realtime-plan` CLI that can render OpenAI and Doubao realtime plans without going through `doctor`, still with no network execution. After that, add a tiny fixture-based executable smoke only with explicit operator intent, short timeout, and cost-aware reporting.
