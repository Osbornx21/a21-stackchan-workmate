# A21 Protocol

## Current Contract

The current Go protocol package defines a small device envelope with trace/session metadata and raw JSON payloads:

```go
const ProtocolVersion = "a21.device.v1"

type Kind string

const (
	KindAudioFrame         Kind = "audio.frame"
	KindAudioPlaybackChunk Kind = "audio.playback.chunk"
	KindDeviceEvent        Kind = "device.event"
	KindAssistantState     Kind = "assistant.state"
	KindScreenExpression   Kind = "screen.expression"
	KindMotionCommand      Kind = "motion.command"
	KindControlEvent       Kind = "control.event"
)

type Envelope struct {
	Protocol  string          `json:"protocol"`
	DeviceID  string          `json:"device_id"`
	Kind      Kind            `json:"kind"`
	Seq       uint64          `json:"seq"`
	TraceID   string          `json:"trace_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	SentAtMS  int64           `json:"sent_at_ms,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}
```

This remains intentionally small. It establishes the A21 namespace, control events, device events, and mock audio chunks without pretending to solve all device media concerns.

## Protocol Rules

- Every message family must be versioned.
- Device messages must include `device_id`.
- Runtime paths carry `trace_id` and `session_id`.
- Audio and control traffic may share an envelope but should have separate transport surfaces when real-time behavior requires it.
- StackChan firmware should receive semantic commands, not provider-specific events.
- V21 evidence must not be encoded as generic chat text when in professional mode; it needs explicit evidence/card fields in later contracts.

## Modes

A21 mode values are semantic product and office-state signals, not provider names:

- `workmate`
- `companion`
- `co_creation`
- `roleplay`
- `professional`
- `focus`
- `public`
- `private`
- `muted`
- `local_fallback`
- `error`

Only `professional` is allowed to trigger the V21 adapter path. `public`, `private`, and `muted` are office visibility/privacy states and must remain visible to the user without silently becoming professional retrieval context.

## Audio Chunks

Current mock audio payload:

- codec: `pcm_s16le` first, Opus later if needed
- sample rates: 16000, 24000, or 48000 Hz
- mono audio for first StackChan path
- device frame duration: 20 ms or 40 ms for LAN responsiveness
- provider aggregation: adapter-specific, often 100-200 ms

The firmware supports two uplink sources with the same A21 `audio.frame` shape. The debug/mock path emits a complete deterministic silence payload: `pcm_s16le`, 16 kHz, mono, 20 ms. The guarded mic path records one capture-safe PCM16 frame, puts it through a bounded firmware queue, base64-encodes the real samples, and sends the same envelope shape. Each frame contains 640 raw PCM bytes encoded as an 856-character base64 string, so Gateway ingress, VAD, and WebSocket sizing are exercised with real frame dimensions instead of a tiny placeholder payload.

Gateway validates `audio.frame` payloads before buffering, VAD, mock playback, barge-in, or provider forwarding. Current accepted PCM frames must decode to the exact byte count implied by `sample_rate_hz * duration_ms * channels * 2 / 1000`; for example, 16 kHz mono 20 ms must be exactly 640 bytes. Invalid frames record `audio.ingress.invalid` and return an `error` control event instead of pretending playback succeeded.

Device uplink uses `audio.frame`.

Gateway downlink uses `audio.playback.chunk` with the same A21 envelope and a playback payload:

- `stream_id`: stable playback stream identifier
- `codec`: currently `pcm_s16le`
- `sample_rate_hz`: 16000, 24000, or 48000
- `channels`: 1
- `duration_ms`: 20 or 40
- `data_base64`: encoded PCM payload

The current Gateway mock downlink emits a complete deterministic silence payload for the first safe subset: `pcm_s16le`, 16 kHz, mono, 20 ms chunks. Each chunk contains 640 raw PCM bytes encoded in `data_base64`, which is large enough to exercise the real envelope size, client decode path, and firmware buffering boundary instead of relying on a placeholder string.

This proves A21 protocol shape, trace/session propagation, device-to-Gateway and Gateway-to-device media envelope direction, simulator PCM scheduling, and firmware buffering/cancellation semantics. It does not prove real provider TTS, physical speaker output, physical microphone quality, mouth sync, or full-duplex capture.

For a single mock trace/session, Gateway keeps the same `stream_id` across consecutive playback chunks. New traces may allocate a new stream. This mirrors the future TTS stream contract and prevents device/simulator buffers from treating every chunk as a replacement stream.

## Control And Device Events

Phase 2B supports:

- device event `mock.turn` -> control events `listening`, `thinking`, `speaking`
- device event `interrupt` -> control events `interrupted`, `listening`
- device event `touch.wake_or_listen` -> same turn path as `mock.turn`, with optional `touch_source`
- device event `touch.barge_in` -> same interruption path as `interrupt`, with optional `touch_source`
- device event `runtime.echo` -> registry-only acknowledgement that firmware applied current screen/motion/RGB state
- audio frame -> mock listening ack, mock speaking state with `stream_id`, then `audio.playback.chunk`
- audio frame with a realtime-capable provider -> Gateway starts or reuses a provider realtime session, forwards detected speech frames, and commits on `vad.speech.end`
- realtime provider output event -> Gateway emits A21 `control.event` and, when audio exists, A21 `audio.playback.chunk`

Firmware-originated `device.event` payloads may also carry build identity:

- `firmware_id`: must be `a21-stackchan` when present
- `firmware_version`: semver-like A21 firmware version
- `firmware_board`: must be `m5stack-cores3` when present
- `firmware_commit`: git SHA embedded by the A21 PlatformIO pre-build script
- `capabilities`: semantic StackChan hardware capability map. Current simulator reports `microphone`, `speaker`, `screen`, `screen_touch`, `top_touch`, `servo_y`, and `rgb` as `available`. Current physical CoreS3 firmware reports `microphone` as `disabled_m5unified_i2s_stop_crash_guard` because the M5Unified mic task can crash in ESP-IDF `i2s_stop`; the other visible/control surfaces remain reported as `available` when initialized.
- `touch_source`: `screen` or `top_sensor` for semantic touch-origin events
- `runtime_echo`: device-applied runtime echo map. Current firmware reports `screen` as the applied render state, `servo_y` as the clamped applied Y-axis angle, and `rgb` as the applied RGB color. Gateway records this for diagnostics and capability evidence derivation, but it is still not a substitute for physical operator observation.

Gateway records this identity, capability map, and runtime echo in the device registry and rejects events whose firmware identity, capability values, or runtime echo values contain forbidden X21/V21 naming or mismatched A21 board/firmware fields. Capability values are evidence of the declared device surface; runtime echo values are evidence that firmware applied a state to its runtime drivers. Neither is proof that physical microphone, speaker, touch, servo, RGB, or screen acceptance has passed.

Firmware audio WebSocket connections include the A21 device identity as a query parameter:

- default path: `/ws/audio?device_id=stackchan-001`
- Gateway rejects legacy-looking device IDs in this registration path
- the connection can still be registered from the first valid `audio.frame` for simulator and older client compatibility

This lets Gateway deliver validation commands to a connected physical StackChan without guessing which anonymous audio socket belongs to which device.

The `/v1/devices` registry response must identify the serving process before any device list is trusted:

- `schema_version`: `a21.gateway.devices.v1`
- `service`: `a21-gateway`

Gateway also records the device's current control state for office acceptance:

- `connection_status`: computed at `/v1/devices` read time as `online`, `stale`, or `unknown`
- `device_age_ms`: age of the latest observed device/control event at read time
- `current_mode`: latest semantic mode from A21 `control.event`
- `current_expression`: latest expression/render state from A21 `control.event`
- `playback_stream_id`: active speaking stream when one is present
- `capabilities`: latest semantic device capability map reported by firmware or simulator
- `runtime_echo`: latest device-applied screen/servo/RGB echo reported by firmware

The current online window is 300000 ms. Anything older is `stale`; this is aligned with the default firmware device identity freshness guard. This field is an operator acceptance aid, not flash permission.

The registry intentionally does not persist utterance text, professional answer text, evidence summaries, or screen-card content. It is an operational state surface for "is this device listening, speaking, professional, private, muted, local, or in error", not a conversation transcript.

## Device Validation Control

`POST /v1/devices/control` is an A21-only physical validation surface for a connected StackChan. It writes semantic `control.event` envelopes to the registered device audio WebSocket and can optionally append up to eight non-silent 20 ms PCM chunks for speaker/playback smoke testing.

Request fields:

- `device_id`: required A21 device ID
- `state`: optional expression state, default `listening`
- `mode`: optional A21 mode, default `workmate`
- `text`: optional short screen/status text
- `trace_id` and `session_id`: optional explicit trace/session IDs
- `stream_id`: required when the caller wants a stable speaking stream; generated only for simple speaking validation
- `mock_audio_chunks`: 0-8 non-silent chunks for physical speaker validation

This endpoint is not a provider path, not a conversation transcript API, and not a replacement for real VAD/STT/LLM/TTS. It exists so office acceptance can command a real device into `listening` or play a bounded validation beep while preserving trace/session evidence.

Professional `control.event` payloads can now include explicit evidence fields:

- `confidence`
- `evidence[]`
- `speech_blocks[]`
- `screen_cards[]`
- `follow_ups[]`

This keeps V21 professional evidence visible to the client without pretending it is ordinary chat text.

Current firmware derives playback start/stop from `control.event` state plus `stream_id`: `speaking` starts the stream, and non-speaking states stop and clear pending playback. The audio WebSocket can also parse `audio.playback.chunk` into a bounded firmware buffer keyed by `stream_id`; accepted `pcm_s16le`, 16 kHz, mono, 20 ms payloads are decoded into fixed 640-byte PCM frames before they enter the buffer. The CoreS3 speaker pump copies decoded frames into stable slots before handing them to M5Unified playback. The buffer is cleared when the render state leaves `speaking`, especially on `interrupted`.

Future control events should still cover explicit playback start/stop, subtitle deltas, mode update event kinds, device status, and trace markers when real audio chunks are present.

Provider audio deltas are translated back into A21 voice/audio events before any Gateway or device-facing code sees them. For example, OpenAI `response.output_audio.delta` is mapped inside `internal/providers` to an A21 `VoiceEvent` with `VoiceAudioChunk`; firmware still receives only A21 downlink playback chunks and semantic control events.

Provider audio uplink is also provider-neutral. Gateway `/ws/audio` may forward `audio.frame` payloads to a provider that implements the A21 realtime session interface, but firmware still sends only A21 `audio.frame` envelopes and never receives provider-specific session commands.

Provider audio downlink uses the same provider-neutral session interface. Gateway consumes `VoiceEvent` values from `Events()` and serializes them back to the audio WebSocket as A21 envelopes. OpenAI realtime `response.output_audio.delta` is currently read and mapped inside the provider adapter before Gateway sees it.

The first realtime session HTTP boundary is:

- `POST /v1/realtime/session`
- `POST /v1/realtime/session/cancel`

It is provider-neutral and returns ordinary A21 `control.event` envelopes. When a provider event carries audio, Gateway also returns A21 `audio.playback.chunk` envelopes with the same trace/session identity. The start endpoint accepts `device_id`, optional `text`, `mode`, `trace_id`, and `session_id`; the cancel endpoint accepts `device_id`, optional `mode`, `trace_id`, `session_id`, `stream_id`, and `reason`. If `stream_id` is omitted during cancel, Gateway uses the active stream recorded from the prior speaking provider event for that trace/session/device.

`professional` mode is intentionally rejected by this realtime boundary. Professional work must use the auditable V21 path with explicit evidence, confidence, speech blocks, and screen cards.

## Trace Summary

`GET /v1/traces?trace_id=<trace_id>` returns the ordered trace event list and a small latency summary:

- `event_count`
- `last_offset_ms`
- `audio_frame_to_playback_ms`
- `v21_query_first_result_ms`
- `barge_in_stop_ms`
- `provider_commit_to_first_audio_ms`

These values are computed from A21 trace markers and are meant for development diagnosis and simulator visibility. They do not claim physical-device first-audio latency until StackChan capture, LAN jitter, speaker buffer, and playback-start markers are present.

## Barge-In Requirements

Interrupt is not just stop audio. It must coordinate:

- local playback stop
- mouth/speaking state stop
- provider cancel/truncate/reset
- conversation state update
- trace marker
- transition back to listening

## Compatibility Rule

No public A21 protocol field may contain X21 or V21 naming unless the message is explicitly part of a V21 adapter contract.
