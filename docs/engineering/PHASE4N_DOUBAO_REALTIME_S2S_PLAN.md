# A21 Phase 4N Doubao Realtime Speech-To-Speech Plan

## Purpose

Phase 4N makes Doubao end-to-end realtime speech-to-speech a first-class readiness target for A21 without dialing Volcengine and without switching Gateway runtime away from mock by accident.

This matters because Doubao realtime speech-to-speech is A21's China-mainland companion fast-path candidate, while `doubao_tts_realtime` is only a TTS-specific professional/audio output candidate.

## Current Implementation

`provider-realtime-plan` now supports:

```bash
go run ./cmd/a21 provider-realtime-plan --provider doubao_realtime
```

Required env names:

- `A21_DOUBAO_API_KEY`
- `A21_DOUBAO_APP_ID`
- `A21_DOUBAO_RESOURCE_ID`
- `A21_DOUBAO_REALTIME_MODEL`

The report emits:

- provider: `doubao_realtime`
- protocol: `websocket_realtime`
- status: `ready` or `skipped`
- configured: boolean
- endpoint host: `ai-gateway.vei.volces.com` when configured
- required env names and missing env names

The report never prints API key, app id, resource id, model value, auth headers, proxy values, or full provider URLs.

## Boundaries

This is not provider connectivity proof. It does not open a WebSocket, stream microphone audio, call a paid endpoint, or validate Doubao audio semantics.

Gateway runtime still defaults to mock. Even with all Doubao env configured, Gateway does not use Doubao unless a later explicit runtime adapter is implemented and `A21_GATEWAY_VOICE_PROVIDER=selected` is intentionally set.

StackChan firmware remains provider-neutral and receives no Doubao credentials or Doubao-native events.
