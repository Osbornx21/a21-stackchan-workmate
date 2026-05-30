# A21 Phase 4I Provider Realtime Plan CLI

## Purpose

Phase 4I adds an explicit realtime provider planning command:

```bash
go run ./cmd/a21 provider-realtime-plan
go run ./cmd/a21 provider-realtime-plan --provider openai_realtime
go run ./cmd/a21 provider-realtime-plan --provider doubao_realtime
go run ./cmd/a21 provider-realtime-plan --provider doubao_tts_realtime
```

or:

```bash
make provider-realtime-plan
A21_PROVIDER=doubao_tts_realtime make provider-realtime-plan
```

This separates realtime WebSocket readiness inspection from the larger `doctor` report. It gives operators a small, redacted, no-network command for checking whether OpenAI Realtime, Doubao end-to-end realtime speech-to-speech, or Doubao realtime TTS is configured enough to be connected by an explicit adapter later.

## Implemented Boundary

Code:

- `internal/app/app.go`
- `internal/app/app_test.go`
- `Makefile`

Capabilities:

- Reads `A21_PROVIDER_PRIMARY` when `--provider` is omitted.
- Accepts explicit `--provider`.
- Emits the existing `ProviderSmokeReport` JSON shape from `RealtimeWebSocketPlanFromEnv`.
- Supports current realtime plan targets:
  - `openai_realtime`
  - `doubao_realtime`
  - `doubao_tts_realtime`
  - `mock` as skipped
- Redacts API keys, model values, voice IDs, auth headers, proxy URLs, and full provider URLs.
- Rejects legacy-looking provider names without echoing their raw value.
- Rejects `--execute`.

## Safety Rules

- The command never dials a provider.
- The command never opens microphone, speaker, serial, or firmware paths.
- The command must not be reinterpreted as proof of real provider connectivity.
- A future executable realtime smoke must be a separate explicit command with tiny fixture audio, short timeout, operator intent, and cost-aware output.

## Test Coverage

Covered by:

```bash
go test ./internal/app -run 'TestRunProviderRealtimePlan' -count=1
```

The tests cover Doubao S2S ready output, Doubao TTS ready output, secret/model/app/resource/voice/header redaction, legacy provider redaction, and `--execute` rejection.

## Next Step

The next provider slice should add a fixture-based realtime smoke design document and a disabled command skeleton, then later implement execution only after the fixture and timeout/cost controls are in place.
