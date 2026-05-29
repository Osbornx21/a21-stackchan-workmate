# A21 Observability

## Principle

A21 must be observable before it is optimized. "Probably the network" is not an acceptable diagnosis.

## Current Signals

The current CLI preflight/doctor report emits:

- preflight result
- blocking findings
- default route interface
- external DNS probe IP
- proxy env variable names without values
- current voice provider health through the provider adapter contract
- optional V21 adapter health when `A21_V21_ADAPTER_URL` is configured

The current gateway also exposes `GET /metrics` for Phase 3A mock runtime metrics.

Gateway now also exposes `GET /v1/devices` for the current in-memory device registry. It records the latest control WebSocket device event, firmware identity, identity validation status, last trace/session IDs, and first/last seen timestamps.

Gateway also exposes `GET /v1/traces?trace_id=<trace_id>` for an in-memory mock waterfall. It currently records HTTP mock turn/interrupt receipts, control WebSocket device events, audio frames, audio ingress buffering, mock VAD start/end markers, mock playback chunk sends, and outgoing control events with millisecond offsets. This is a development observability surface, not the final durable trace backend.

Gateway also exposes `GET /v1/providers/voice/health` for the current voice provider adapter. It returns provider name, health status, configured state, realtime capability, optional active child provider, and detail text. Unavailable providers return HTTP 503 so future real-provider failures can be distinguished from device and firmware failures.

## Trace Fields

Future runtime spans should include:

- `trace_id`
- `session_id`
- `device_id`
- `turn_id`
- `mode`
- `network_interface`
- `dns_probe_ip`
- `proxy_mode`
- `provider`
- `adapter`
- `audio_chunk_id`
- `stream_id`
- `error_code`

Current mock trace events include:

- `http.mock_turn.received`
- `http.mock_interrupt.received`
- `device.mock.turn.received`
- `device.interrupt.received`
- `device.touch.wake_or_listen.received`
- `device.touch.barge_in.received`
- `audio.frame.received`
- `audio.ingress.buffered`
- `audio.ingress.invalid`
- `vad.speech.start`
- `vad.speech.end`
- `audio.playback.chunk.sent`
- `v21.query.start`
- `v21.query.first_result`
- `v21.query.error`
- `control.listening.sent`
- `control.professional.sent`
- `control.thinking.sent`
- `control.speaking.sent`
- `control.interrupted.sent`

## Voice Waterfall Events

Every voice turn should eventually expose:

- `audio.capture.start`
- `audio.capture.end`
- `gateway.audio.first_frame`
- `vad.speech.start`
- `vad.speech.end`
- `asr.first_partial`
- `llm.first_token`
- `v21.query.start`
- `v21.first_result`
- `tts.first_chunk`
- `device.downlink.first_frame`
- `device.playback.start`
- `barge_in.detected`
- `provider.cancel`
- `playback.stop`
- `fallback.used`

## Metrics

Current Prometheus metrics:

- `a21_mock_turn_total`
- `a21_barge_in_total`
- `a21_audio_frame_total`
- `a21_audio_playback_chunk_total`
- `a21_audio_ingress_frames_total`
- `a21_audio_ingress_dropped_frames_total`
- `a21_audio_ingress_buffer_depth`
- `a21_vad_speech_start_total`
- `a21_vad_speech_end_total`
- `a21_device_identity_invalid_total`
- `a21_v21_query_ms_bucket`
- `a21_ws_connections_active`

Future Prometheus metrics should include:

- `a21_session_total`
- `a21_session_active`
- `a21_first_audio_ms_bucket`
- `a21_audio_uplink_ms_bucket`
- `a21_audio_downlink_ms_bucket`
- `a21_vad_duration_ms_bucket`
- `a21_tts_first_chunk_ms_bucket`
- `a21_barge_in_total`
- `a21_barge_in_stop_ms_bucket`
- `a21_provider_error_total`
- `a21_proxy_misconfig_total`
- `a21_device_disconnect_total`
- `a21_fallback_total`

## Logs

Runtime logs must be structured JSON once long-running services exist. Minimum fields:

```json
{
  "level": "info",
  "service": "a21-core",
  "event": "preflight",
  "trace_id": "a21-trace-example",
  "session_id": "a21-session-example",
  "device_id": "stackchan-001"
}
```

Do not log provider keys, raw secrets, or proxy URLs with credentials.

## Dashboards

Future dashboards:

- A21 Latency Waterfall
- A21 Provider Health
- A21 Device Health
- A21 Proxy And Network
- A21 V21 Professional Mode
- A21 Barge-In
- A21 Errors And Fallback
