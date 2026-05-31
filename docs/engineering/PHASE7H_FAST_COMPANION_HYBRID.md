# A21 Phase 7H Fast Companion Hybrid Boundary

## Purpose

Phase 7H adds the first Gateway-level routing boundary for the Fast Companion Hybrid lane. It connects local audio front-end results to a provider-neutral `text_stream` placeholder waterfall without executing a real provider, V21 query, TTS engine, Gateway runtime service, or physical StackChan control path.

This slice is intentionally a boundary and trace-shape implementation. It does not claim final ASR quality, provider latency, local TTS quality, physical speaker playback, or M3 hardware acceptance.

## Gateway Route

Gateway exposes:

```text
POST /v1/fast-companion/turn
```

The request carries:

```json
{
  "device_id": "stackchan-sim-001",
  "mode": "companion",
  "trace_id": "a21-trace-example",
  "session_id": "a21-session-example",
  "local_audio": {
    "asr_provider": "mock_asr",
    "first_partial_ms": 42,
    "final_transcript_chars": 11
  }
}
```

`local_audio.asr_provider` is required so this boundary cannot be entered with an empty or ambiguous audio front-end result. Local audio counters must be non-negative.

The response is deliberately redacted and provider-neutral:

```json
{
  "status": "boundary_ready",
  "route": "fast_companion_hybrid",
  "audio_frontend": "local_audio",
  "provider_family": "text_stream",
  "text_stream_provider": "mock_text_stream",
  "text_stream_executed": false
}
```

The route emits control events only. It does not emit `audio.playback.chunk`, call `VoiceProvider.StartTurn`, call V21, open provider network connections, persist provider reports, or require `A21_GATEWAY_VOICE_PROVIDER=selected`.

## Trace Waterfall

The route records the unified waterfall markers required by the Fast Companion Hybrid lane:

```text
fast_companion.turn.received
fast_companion.local_audio.frontend.accepted
asr.first_partial
provider.text_stream.route.placeholder
provider.first_byte
provider.first_content
tts.first_audio
audio.downlink.first_frame
device.playback.start
```

`provider.first_byte`, `provider.first_content`, `tts.first_audio`, `audio.downlink.first_frame`, and `device.playback.start` are placeholders in this phase. They reserve the Gateway trace shape so later provider, TTS, downlink, and device work can fill real timings without changing the trace contract.

## Mode Boundary

`companion` and `workmate` can use this boundary. Other modes are rejected at this boundary until a later ADR defines their product behavior. `professional` must not enter the opaque realtime or Fast Companion placeholder route; professional mode continues through the V21 evidence path and records `v21.query.start` plus `v21.query.first_result` or an honest V21 failure marker.

The existing realtime route still rejects `professional` mode with guidance to use the professional V21 evidence path.

## Safety

This phase adds no ports, environment variables, containers, firmware commands, physical device writes, provider execution, V21 execution, proxy behavior, or durable reports.

Gateway default mock safety remains unchanged: selected provider runtime still requires explicit `A21_GATEWAY_VOICE_PROVIDER=selected`, and this Fast Companion boundary does not use that runtime even when a `VoiceProvider` is injected in tests.

## Acceptance

Current tests cover:

- local audio front-end result to `text_stream` placeholder routing
- rejection of unsupported modes, missing local audio provider identity, and negative local audio counters
- trace markers for ASR, provider, TTS, downlink, and device playback placeholders
- no selected `VoiceProvider` start from the Fast Companion boundary
- no V21 call from companion/workmate Fast Companion boundary
- professional realtime rejection remains intact
- professional V21 evidence path remains intact
