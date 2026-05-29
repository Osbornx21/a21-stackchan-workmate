# A21 Phase 4F OpenAI Realtime Provider Wrapper

## Purpose

Phase 4F wraps the Phase 4D WebSocket transport and Phase 4E OpenAI event mapper in a provider-owned session boundary.

This is still not a production OpenAI voice rollout. It gives A21 a configured provider object that can be health-checked, can open an explicit realtime audio session through an injected dialer, and can send audio/commit/cancel events in the provider lane without exposing provider JSON to Gateway or firmware.

## Implemented Boundary

Code:

- `internal/providers/openai_realtime_provider.go`
- `internal/providers/openai_realtime_provider_test.go`

Capabilities:

- Builds `OpenAIRealtimeVoiceProvider` from `A21_OPENAI_API_KEY`, `A21_OPENAI_REALTIME_MODEL`, optional `A21_OPENAI_SAFETY_IDENTIFIER`, and optional `A21_OPENAI_REALTIME_INSTRUCTIONS`.
- Reports health without printing API keys, model values, auth headers, or full provider URLs.
- Refuses to start when required OpenAI env is missing.
- Opens an explicit realtime session using the existing `RealtimeWebSocketAdapter`.
- Sends A21 audio chunks through the OpenAI realtime event mapper.
- Sends manual commit/create events in order.
- Sends provider cancel without leaking A21-local cancel reason fields.
- Keeps ordinary `StartTurn` disabled so a text-mode Gateway path cannot accidentally dial OpenAI.

## Safety Rules

- `StartRealtimeSession` is the only active session entrypoint.
- `StartTurn` returns an error and does not dial the provider.
- Tests use injected fake WebSocket connections only.
- `doctor` remains dry-run and does not execute provider calls.
- `provider-smoke --execute` still does not execute OpenAI Realtime.
- StackChan firmware never sees provider env, provider headers, or provider event names.

## Non-Goals

Phase 4F does not:

- switch Gateway traffic to OpenAI
- stream live office microphone audio
- implement real provider smoke
- parse every OpenAI server event
- add provider retry/backoff
- add billing/cost accounting
- implement Doubao realtime

## Test Coverage

Covered by:

```bash
go test ./internal/providers -run 'TestOpenAIRealtimeVoiceProvider' -count=1
go test ./internal/providers
```

The tests cover configured health redaction, missing env handling, explicit session lifecycle, event ordering, no provider-event secret leakage, and the guard that prevents `StartTurn` from dialing OpenAI.

## Next Step

Phase 4G adds a Doubao realtime TTS WebSocket boundary as a separate TTS-only provider lane. Remaining OpenAI provider work should add a redacted realtime plan CLI or extend `provider-smoke` with a dry-run realtime plan command. A real `--execute` smoke should be added only after it can use a tiny local test audio fixture, explicit operator intent, short timeout, and cost-aware reporting.
