# A21 Phase 4M Gateway Realtime Session Boundary

Phase 4M adds the first provider-neutral realtime session entrypoints to Gateway:

```bash
POST /v1/realtime/session
POST /v1/realtime/session/cancel
```

The endpoints use the existing A21 `VoiceProvider` contract. They do not let StackChan talk to OpenAI, Doubao, Bailian, DeepSeek, or V21 directly.

## What It Proves

- Gateway can start a realtime voice turn through the selected runtime `VoiceProvider`.
- Gateway can cancel the active speaking stream by `trace_id`, `session_id`, and `device_id`.
- Cancel can infer `stream_id` from the active speaking provider event.
- Realtime start/cancel emit trace markers.
- Provider start/cancel latency is exposed through Prometheus histograms.
- `professional` mode is rejected by the realtime fast path so evidence work stays on the V21 professional path.

## New Metrics

- `a21_realtime_session_total`
- `a21_realtime_session_cancel_total`
- `a21_voice_provider_start_turn_ms_bucket`
- `a21_voice_provider_cancel_ms_bucket`

## Trace Markers

- `realtime.session.start.received`
- `provider.start_turn.start`
- `provider.start_turn.first_event`
- `provider.start_turn.end`
- `provider.start_turn.error`
- `realtime.session.cancel.received`
- `provider.cancel.start`
- `provider.cancel.end`
- `provider.cancel.error`

## Safety

Gateway remains mock by default. Real provider runtime still requires `A21_GATEWAY_VOICE_PROVIDER=selected`, and real provider credentials must stay in Gateway/provider config only.

This phase does not add real provider network smoke, real provider audio downlink, or firmware flashing.
