# A21 Phase 4E OpenAI Realtime Event Mapping

## Purpose

Phase 4E adds the first provider-specific realtime event mapper. It converts between A21 audio/provider-neutral events and OpenAI Realtime WebSocket JSON events without executing a provider network call.

This moves A21 closer to a real low-latency voice path while preserving the core boundary:

- StackChan speaks A21 protocol only.
- Gateway will call A21 provider interfaces only.
- OpenAI event names stay inside `internal/providers`.
- No provider key enters firmware, simulator, docs output, or device messages.

## Source Baseline

OpenAI's Realtime WebSocket guide says server-to-server integrations exchange JSON events over a WebSocket and initialize sessions with `session.update`.

OpenAI's Realtime conversation guide describes WebSocket audio input with `input_audio_buffer.append`, `input_audio_buffer.commit`, and `response.create`. It also describes interruption with `response.cancel`, and server audio output through `response.output_audio.delta` events containing Base64-encoded audio data.

## Implemented Boundary

Code:

- `internal/providers/openai_realtime.go`
- `internal/providers/openai_realtime_test.go`
- `internal/providers/voice.go`

Capabilities:

- Validates A21 PCM S16LE mono audio chunks before provider mapping.
- Maps valid A21 audio chunks to `input_audio_buffer.append`.
- Emits manual-turn events in order: `input_audio_buffer.commit`, then `response.create`.
- Emits OpenAI `response.cancel` without leaking A21-local cancel reasons into provider JSON.
- Maps OpenAI `response.output_audio.delta` events into A21 `VoiceEvent` values with `VoiceAudioChunk`.
- Ignores unrelated OpenAI server events instead of treating them as errors.
- Writes mapped events to the injected `RealtimeWebSocketSession` for local tests.

## Non-Goals

Phase 4E does not:

- dial OpenAI
- stream microphone audio from Gateway yet
- parse every OpenAI server event
- implement conversation truncation
- implement tool calls
- implement Doubao event mapping
- expose provider event names to firmware

## Test Coverage

Covered by:

```bash
go test ./internal/providers -run 'TestOpenAIRealtime' -count=1
go test ./internal/providers
```

The tests cover mapping, validation, cancel/turn sequencing, provider output audio deltas, ignored unrelated events, and injected-session writes.

## Next Step

Phase 4F wraps this mapper in an `OpenAIRealtimeVoiceProvider` session boundary. Remaining provider work should add dry-run CLI visibility first, then an explicit executable smoke using a tiny local test audio fixture, short timeout, and cost-aware reporting.
