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

## Xiaozhi WebSocket Compatibility

The Gateway exposes the WS-1 xiaozhi compatibility seam at
`/v1/xiaozhi` on the existing A21 Gateway port. It accepts stock-style xiaozhi
handshake headers `Device-Id` and `Protocol-Version`; `device_id` may also be
provided in the JSON body or query string for local harnesses. It accepts
stock-style xiaozhi JSON control messages:

- `hello`
- `listen` with `state=start|detect|stop`
- `abort`

The server hello includes stock `audio_params` and an `audio` alias for current
local tests. The client `hello.features` object is parsed for `mcp`, `aec`,
`device_events`, and `debug_metrics`; `mcp` and `aec` remain stock xiaozhi
capability hints, while `device_events` and `debug_metrics` mark an isolated
debug profile in the device registry and are never echoed into the stock server
hello. It also accepts xiaozhi binary protocol versions 1, 2, and 3 after a
valid `hello` and active `listen/start`. Version 1 is a raw Opus payload.
Version 2 unwraps the 16-byte metadata header and preserves the timestamp.
Version 3 unwraps the compact 4-byte header. The current server seam records
Opus frame count and byte count, propagates or derives `device_id`, `trace_id`,
and `session_id`, and rejects legacy-looking X21/V21 identities.

This is not yet the complete product voice chain. The current WS-1 seam decodes
valid uplink Opus frames to PCM16 to produce aggregate telemetry
(`decoded_frame_count`, `decoded_sample_count`, `decoded_duration_ms`) and an
honest `decode_status` such as `opus_decoded_pcm16` or `opus_decode_error`.
Decoded PCM also enters the existing `audio.Ingress` buffer and VAD markers so
the next ASR slice has the same observable ingress surface as `/ws/audio`.
When decoded frames include VAD speech, `/v1/xiaozhi` can run the
`internal/providers` fixture pipeline and send the resulting mock TTS PCM chunk
through the paced Opus downlink. When no speech or no usable decoded frame is
available, it still emits the honest xiaozhi TTS lifecycle placeholder. This
is fixture plumbing only: it does not execute real ASR/LLM/TTS providers and
must not be cited as audible product acceptance.

Future xiaozhi TTS binary downlink must use the Go `AudioRateController`
primitive before writing frames: default 60 ms frame slots, five-frame
prebuffer, per-frame abort checks, and reset on turn cancellation. Raw unpaced
binary writes are not accepted as an A21 product path.
The current downlink primitive accepts only validated 24 kHz mono 60 ms
`pcm_s16le` provider audio, encodes it to Opus, and writes one xiaozhi binary
frame through the current-turn pacer. This is a downlink building block, not
ASR/LLM/TTS product acceptance.

WS-2 adds a host-side product voice pipeline contract under
`internal/providers`. It models decoded PCM frame metadata flowing through ASR,
streaming text, and TTS adapters, then returns downlink-ready
`VoiceAudioChunk` values: `pcm_s16le`, 24 kHz, mono, 60 ms. The current
implementation is fixture/mock only. It records stage markers such as
`asr_first_partial_ms`, `llm_first_content_ms`, `tts_first_audio_ms`, and
`audio_downlink_first_frame_ms`, preserves provider selection by A21 env/profile
names, and emits a redacted report that stores counts, format metadata, timing,
and policy fields only. It must not be cited as real provider execution,
physical StackChan first-audio acceptance, transcript quality evidence, or PRD
latency acceptance.

The Gateway xiaozhi fixture path consumes those report fields without storing
transcripts, provider output, or audio payloads in JSON. Its TTS start message
may include `voice_pipeline.schema_version`, `execution_mode`, chunk counts,
and timing fields. The actual mock audio travels only as paced binary Opus
frames; `data_base64` is not emitted in xiaozhi JSON.

Each `listen/start` creates a Gateway-owned xiaozhi turn and returns a stable
A21 `turn_id` in the accepted reply. Each `abort` cancels the current turn
context, clears current-turn ownership, resets the downlink pacer, and returns
one `tts/stop` with the cancelled `turn_id`. Future provider and TTS frame code
must check current-turn ownership before every device-facing frame send; stale
turn downlink attempts are suppressed at the host seam and traced without
emitting JSON or binary device frames.

Stock xiaozhi listen modes such as `realtime`, `auto`, and `manual` are
transport hints, not A21 product modes. During bounded physical acceptance,
`A21_XIAOZHI_STOCK_PROFESSIONAL_ROUTE=professional` can explicitly map stock
listen turns to A21 `professional` inside Gateway without adding debug fields
to the stock protocol. Professional checking, fallback, and result speech use
the same Gateway-owned TTS adapter and paced OPUS binary downlink as normal
voice answers; the structured evidence/card JSON remains metadata and must not
replace audible device-facing frames.

### Xiaozhi MCP And Expression Contract

The xiaozhi transport package now carries a host-only WS-5 contract for future
device-control integration:

- MCP JSON-RPC envelopes are limited to `initialize`, `tools/list`, and
  `tools/call` request shapes. This package only builds and parses the
  envelopes; it does not execute `tools/call`, discover live device tools, or
  connect to Gateway.
- `tools/call` builders validate tool names, reject legacy-looking X21/V21
  names, and redact sensitive argument fields such as keys, tokens, prompt or
  transcript text, raw/base64 audio, provider output, URLs, proxies, and local
  paths.
- Server-to-device expression uses stock xiaozhi `type=llm` messages with an
  `emotion` field. A21 expression states currently map to
  `idle`, `listening`, `thinking`, `speaking`, `interrupted`, `professional`,
  and `error`.
- Optional motion parameters clamp `y_angle` to the stock-safe 5-85 degree
  range before any later adapter may send them.

### A21 StackChan Device Extension

WS-5 also defines an A21-only StackChan semantic device extension in
`internal/transport/xiaozhi`. It is host-only schema, builder, and parser
proof for future Gateway-to-device integration; it does not make `type=device`
part of the stock xiaozhi profile. Stock hello and server hello remain free of
debug or device-extension requirements. A debug client that explicitly
advertises `features.device_events=true` receives only an A21-namespaced server
allowance: `a21.profile=debug` and `a21.device_events=true`. Stock server
hellos remain free of `a21`, `device_events`, and `debug_metrics`. A host may
build `type=device` extension events only when the connected profile explicitly
advertises `features.device_events=true` or the host has selected an A21
debug/StackChan extension profile.

Current extension event kinds are `state`, `face`, `display`, `motion`,
`heartbeat`, and debug-profile-only `playback`. Values are provider-neutral A21
semantics:

- `state`: `idle`, `listening`, `thinking`, `speaking`, `error`
- `face`: `idle`, `attentive`, `thinking`, `speaking`, `happy`, `error`
- `display`: `status`, `asr`, `tts`
- `motion`: `look_up`, `nod`, `shake`, `stop`, `dance`
- `playback`: `start` or `stop_done`, with optional `stream_id`; accepted only
  after `features.device_events=true`. `start` is recorded as
  `device.playback.start`; `stop_done` is recorded as
  `device.playback.stop_done` and may prove barge-in stop completion when it
  follows `barge_in.detected`.

The repo-owned firmware overlay
`firmware/xiaozhi/overlays/a21-debug-playback-ack.patch` keeps this out of the
stock profile by default. It adds `CONFIG_A21_DEBUG_DEVICE_EVENTS=n`,
advertises client `features.device_events=true` only in that debug build,
requires the server `a21.profile=debug` / `a21.device_events=true` allowance,
and emits redacted playback runtime echoes after decoded PCM reaches the
firmware audio output task and after server TTS stop has moved the device out
of speaking state. For interrupt reasons such as `abort`, `barge`, or `wake`,
the overlay also clears the firmware decoder/playback queue before reporting
`stop_done`; normal completion stops still preserve the stock late-stop
playback-drain behavior. A local touch/wake abort also clears the device
decoder queue immediately before sending the abort upstream, so physical audio
does not wait for the server round trip before stopping. On CoreS3 debug
firmware, screen touch-down while already speaking triggers that abort path
immediately and consumes the later touch release, avoiding a fragile
release-only interruption window.

Motion `y_angle` is clamped to 5-85 when present. Inline assistant text marks
such as `[face:happy]` and `[motion:nod]` are parsed on the host by stripping
the marks from spoken text and returning semantic extension events. Unsupported
kinds or values return stable transport errors and must not leak legacy
identity strings into reports or stdout.

This is a protocol contract only. It is not actual servo, RGB, screen, MCP
tool, Gateway, firmware, or physical hardware acceptance. Future
Gateway/device-control integration must first consume tools discovered from the
connected device and then prove the resulting behavior with traceable device
evidence.

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

Gateway keeps a bounded in-memory recent-audio capture for development diagnostics and mic-driven fast-companion evidence. `GET /v1/audio/recent` is loopback-only and must reject LAN callers; the default response redacts `data_base64` and returns only metadata such as frame count, byte count, RMS, VAD decision, and trace/session IDs. A local CLI may request `include_audio=1` over `127.0.0.1` to build a temporary ASR WAV, but saved A21 reports must not include raw audio or `data_base64`.

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

## Planned Binary Opus Media Profile

The future A21 binary Opus media profile is a planning contract inside this
live protocol document. It is not a production protocol acceptance and does not
change the current JSON/base64 `pcm_s16le` runtime contract.

The planned profile keeps A21 control metadata separate from binary media
payloads: JSON control negotiation carries `trace_id`, `session_id`,
`device_id`, `stream_id`, media profile, codec, frame duration, and direction;
future binary frames carry compact references plus Opus payload bytes. PCM
fixtures remain the compatibility and fallback lane until an encoder/decoder
adapter spike, wire-format fixture, Gateway loopback fixture, provider
first-audio waterfall report, device playback receipt, CPU/memory profile, LAN
jitter/fallback report, and hardware window acceptance all pass under explicit
authorization.

Current `AudioCodecOpus` is reserved vocabulary for the older A21 envelope
path. The xiaozhi compatibility seam can receive raw Opus binary frames, and
Gateway can decode valid 60 ms mono uplink frames for telemetry and
loopback-only ingress inspection, but it must not treat those frames as
provider-ready audio until a later ASR/frontend slice adds the explicit handoff.
Current A21 envelope validation still accepts `pcm_s16le` frames, and no
Gateway, provider, firmware, or physical device path should treat Opus media as
product-complete from this document.

## Control And Device Events

Phase 2B supports:

- device event `mock.turn` -> control events `listening`, `thinking`, `speaking`
- device event `interrupt` -> control events `interrupted`, `listening`
- device event `touch.wake_or_listen` -> same turn path as `mock.turn`, with optional `touch_source`
- device event `touch.barge_in` -> same interruption path as `interrupt`, with optional `touch_source`
- device event `runtime.echo` -> registry-only acknowledgement that firmware applied current screen/motion/RGB state
- audio frame -> mock listening ack, mock speaking state with `stream_id`, then `audio.playback.chunk`
- simulator/bench audio frame with a realtime-capable provider -> Gateway starts or reuses a provider realtime session, forwards detected speech frames, and commits on `vad.speech.end`
- physical StackChan audio frame with a realtime-capable provider -> Gateway requires an explicit `realtime_on_next_speech` arm before it starts a provider realtime session; unarmed physical speech frames are measured and suppressed instead of creating paid/external provider traffic
- realtime provider output event -> Gateway emits A21 `control.event` and, when audio exists, A21 `audio.playback.chunk`

Firmware-originated `device.event` payloads may also carry build identity:

- `firmware_id`: must be `a21-stackchan` when present
- `firmware_version`: semver-like A21 firmware version
- `firmware_board`: must be `m5stack-cores3` when present
- `firmware_commit`: git SHA embedded by the A21 PlatformIO pre-build script
- `capabilities`: semantic StackChan hardware capability map. Current simulator and firmware preserve the full A21 StackChan capability surface: `microphone`, `speaker`, `screen`, `screen_touch`, `top_touch`, `servo_y`, `servo_x`, `rgb`, `camera`, `imu`, `ambient_light`, `proximity`, `battery`, `nfc`, and `infrared`. Current physical CoreS3 release firmware reports `microphone` as `disabled_m5unified_i2s_stop_crash_guard`; the isolated `a21_stackchan_cores3_mic_probe` build reports `diagnostic_probe_m5unified_i2s_capture` and must not be treated as production microphone availability. Stable visible/control surfaces remain reported as `available` when initialized. Unimplemented but real StackChan surfaces are reported as `planned_*` instead of being hidden or falsely marked available.
- `touch_source`: `screen` or `top_sensor` for semantic touch-origin events
- `runtime_echo`: device-applied runtime echo map. Current firmware reports `screen` as the applied render state, `servo_y` as the clamped applied Y-axis angle, and `rgb` as the applied RGB color. Diagnostic mic-probe firmware also reports string counters such as `mic_frames_captured`, `mic_driver_errors`, skip counters, mic queue depth, `audio_ws_sent_audio_frames`, `mic_last_abs_peak`, and `mic_last_nonzero_samples` so physical microphone bring-up can distinguish capture, queue, WebSocket send, and all-zero sample failures. The playback path reports `playback_buffer_queued_chunks`, `playback_buffer_total_chunks`, `playback_buffer_dropped_chunks`, `playback_buffer_clear_count`, `speaker_frames_played`, `speaker_busy_ticks`, `speaker_driver_errors`, and `speaker_last_stream_id` so speaker bring-up can separate downlink buffering, cancellation, M5 speaker queue backpressure, and driver failure. Diagnostic sensor-probe firmware reports `sensor_available`, `sensor_samples`, `sensor_read_errors`, `ambient_light_raw`, `proximity_raw`, `battery_mv`, and `battery_ma` so CoreS3 LTR553 and StackChan-BSP INA226 bring-up can separate bus/device availability from changing environment/power telemetry. Gateway records this for diagnostics and capability evidence derivation, but it is still not a substitute for physical operator observation.

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

`POST /v1/devices/control` is an A21-only physical validation surface for a connected StackChan. It writes semantic `control.event` envelopes to the registered device audio WebSocket and can optionally append up to eight non-silent 20 ms PCM chunks for speaker/playback smoke testing or up to eight caller-provided `audio.playback.chunk` payloads for local TTS playback batches.

Longer speaker acceptance probes must preserve that per-request cap. The `stackchan-speaker-acceptance` CLI may send about one second of requested probe audio, but it does so as multiple bounded control requests, each with `mock_audio_chunks <= 8`, sharing the same A21 trace/session/stream identifiers. The final physical playback request may be padded to a full eight-frame speaker batch so firmware does not need to start a stream with a fragmented `playRaw` call. Host-side local playback must prebuffer the first three eight-frame batches when enough audio exists; this gives the device-side jitter buffer room to absorb HTTP, WebSocket, JSON parsing, and M5 speaker scheduling overhead instead of exposing every 160 ms block boundary as an audible gap.

Request fields:

- `device_id`: required A21 device ID
- `state`: optional expression state, default `listening`
- `mode`: optional A21 mode, default `workmate`
- `text`: optional short screen/status text
- `trace_id` and `session_id`: optional explicit trace/session IDs
- `stream_id`: required when the caller wants a stable speaking stream; generated only for simple speaking validation
- `diagnostic_tone_hz`, `diagnostic_tone_duration_ms`, `diagnostic_tone_volume`: optional physical speaker diagnostic tone. This is a device-local M5Unified tone path for isolating speaker/I2S/amplifier behavior from Gateway WAV/base64/playback-buffer streaming. It must not be used as product TTS.
- `mock_audio_chunks`: 0-8 non-silent chunks for physical speaker validation
- `audio_chunks`: 0-8 caller-provided playback chunks. Each chunk must be `pcm_s16le`, mono, 16/24/48 kHz, 20 or 40 ms, and its `stream_id` must match the request `stream_id`.
- `audio_probe_only`: optional diagnostic flag for the requested trace/session. When true, Gateway keeps accepting and measuring matching `audio.frame` uplink frames but suppresses mock listening/speaking/playback responses. Use this for physical microphone probes so Gateway does not force StackChan into `speaking` while measuring capture.
- `mock_playback_on_next_audio_frame`: optional physical validation flag for the requested trace/session. When true, Gateway arms exactly one mock playback response for the next matching physical StackChan `audio.frame`, then immediately disarms it.
- `realtime_on_next_speech`: optional physical provider flag for the requested trace/session. When true, Gateway arms exactly one provider realtime-session start for the next matching physical StackChan speech frame. This is separate from mock playback validation and prevents ambient office audio from opening realtime provider sessions by accident.

The audio WebSocket mock path keeps simulator devices silent by default, but physical `stackchan-*` devices can receive a short non-silent PCM chunk after an explicit one-shot validation arm or after the VAD `speech_start` uplink frame. Later frames in the same speech segment are measured but do not produce more mock playback. That distinction lets the half-duplex acceptance prove a real microphone-to-Gateway-to-speaker loop without creating a mic/playback feedback loop, changing simulator expectations, or claiming a real provider response.

This endpoint is not a provider path, not a conversation transcript API, and not a replacement for real VAD/STT/LLM/TTS. It exists so office acceptance can command a real device into `listening` or play a bounded validation beep while preserving trace/session evidence.

Professional `control.event` payloads can now include explicit evidence fields:

- `confidence`
- `evidence[]`
- `speech_blocks[]`
- `screen_cards[]`
- `follow_ups[]`

This keeps V21 professional evidence visible to the client without pretending it is ordinary chat text.

Current firmware derives playback start/stop from `control.event` state plus `stream_id`: `speaking` starts the stream, and non-speaking states stop and clear pending playback. The audio WebSocket can also parse `audio.playback.chunk` into a bounded 32-frame firmware jitter buffer keyed by `stream_id`; accepted `pcm_s16le`, 16 kHz, mono, 20 ms payloads are decoded into fixed 640-byte PCM frames before they enter the buffer. The CoreS3 speaker pump waits for a full initial eight-frame preroll, then coalesces up to eight contiguous 20 ms frames into one 160 ms playback block before handing it to M5Unified playback. Host-side local playback batches prebuffer the first three batches, pad the final short batch with silence, and wait for the full final playback window before sending idle, reducing audible boundary noise from isolated `playRaw` calls while preserving the low-latency protocol frame size. The firmware also raises the M5Unified speaker task priority for this streaming path. The buffer is cleared when the render state leaves `speaking`, especially on `interrupted`. For hardware diagnosis, a `control.event` may carry the `diagnostic_tone_*` fields above; firmware plays that tone locally through M5Unified and reports `speaker_tone_requests`, `speaker_tone_driver_errors`, and last tone parameters in `runtime_echo`.

Future control events should still cover explicit playback start/stop, subtitle deltas, mode update event kinds, device status, and trace markers when real audio chunks are present.

Provider audio deltas are translated back into A21 voice/audio events before any Gateway or device-facing code sees them. For example, OpenAI `response.output_audio.delta` is mapped inside `internal/providers` to an A21 `VoiceEvent` with `VoiceAudioChunk`; firmware still receives only A21 downlink playback chunks and semantic control events.

Provider audio uplink is also provider-neutral. Gateway `/ws/audio` may forward `audio.frame` payloads to a provider that implements the A21 realtime session interface, but firmware still sends only A21 `audio.frame` envelopes and never receives provider-specific session commands. For physical StackChan devices, provider realtime startup is opt-in per trace/session through `realtime_on_next_speech`; simulator and benchmark devices keep automatic behavior for deterministic development tests.

Provider audio downlink uses the same provider-neutral session interface. Gateway consumes `VoiceEvent` values from `Events()` and serializes them back to the audio WebSocket as A21 envelopes. OpenAI realtime `response.output_audio.delta` is currently read and mapped inside the provider adapter before Gateway sees it.

## AgentTask Bridge Events

The AgentTask bridge is currently a provider-package T1/T2 contract, not a
device or Gateway runtime protocol. `AgentTaskRequest` carries `trace_id`,
`session_id`, `task`, and `context`. External-agent stream events are consumed
as `AgentTaskEvent` values with kinds `started`, `progress`, `text_delta`,
`tool_call_redacted`, `result`, `error`, and `final`.

Before any A21 surface consumes them, these events are mapped into
`a21.agent_task.semantic_event.v1` report entries. The mapper preserves
trace/session identity and final/error markers, but it stores only text length
or redaction markers for external text and tools. It does not forward provider
payloads, tool payloads, credentials, full URLs, local paths, provider env
values, or direct control commands.

AgentTask events are not allowed to become StackChan control events, realtime
voice events, professional V21 evidence, or `/v1/devices/control` requests
without a future approved adapter and runtime gate.

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
- `xiaozhi_listen_to_audio_ingress_ms`
- `xiaozhi_opus_decode_ms`
- `asr_first_partial_ms`
- `llm_first_content_ms`
- `tts_first_audio_ms`
- `audio_downlink_first_frame_ms`
- `device_playback_start_ms`
- `answer_first_audio_total_ms`

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
