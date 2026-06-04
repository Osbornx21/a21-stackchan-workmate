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

The server hello includes stock downlink `audio_params` (`opus`, `24000 Hz`,
mono, `60 ms`) and an `audio` alias for current local tests. The client
`hello.features` object is parsed for `mcp`, `aec`,
`device_events`, and `debug_metrics`; `mcp` and `aec` remain stock xiaozhi
capability hints, while `device_events` and `debug_metrics` mark an isolated
debug profile in the device registry and are never echoed into the stock server
hello. It also accepts xiaozhi binary protocol versions 1, 2, and 3 after a
valid `hello` and active `listen/start`. Version 1 is a raw Opus payload.
Version 2 unwraps the 16-byte metadata header and preserves the timestamp.
Version 3 unwraps the compact 4-byte header. The current server seam records
Opus frame count and byte count, propagates or derives `device_id`, `trace_id`,
and `session_id`, and rejects legacy-looking X21/V21 identities.

By default the WS-1 seam remains a deterministic fixture path so stock protocol
tests stay cheap and offline. With
`A21_XIAOZHI_PRODUCT_CHAIN=host_local` (or `A21_PRODUCT_CHAIN=host_local`) the
Gateway fills missing xiaozhi pipeline defaults with
`A21_ASR_LOCAL_PROFILE=sherpa_onnx`, `A21_TTS_FAST_PROFILE=sherpa_onnx_tts`,
and, when local Ollama URL/model env is present and no text profile is selected,
`A21_TEXT_STREAM_PROFILE=local_ollama`. Explicit ASR/text/TTS env values always
win over these defaults.

Runtime hot-plug selection is separate from product readiness promotion. When
the operator or frontend explicitly sets `A21_TEXT_STREAM_PROFILE`,
`A21_PROVIDER_PRIMARY`, `A21_TTS_FAST_PROFILE`, or `A21_LOCAL_TTS_ENGINE`, the
host voice path may execute that configured candidate if it matches the A21
text-stream/TTS adapter contract. The built-in StepFun text-stream profile is
now an explicit route-eligible launch-policy LLM; other
`route_eligible=false` candidates can still be used for explicit
contest/listening runs, but those runs do not make them product-ready. Product
readiness still requires the separate route/evidence gates. Current host-side
TTS selectors include
`iflytek_tts` for the immediate real-time cloud TTS candidate and
`voice_clone_cli` for the retained voice-clone seam. `sherpa_onnx` remains
available as an emergency/diagnostic local path but is not the desired contest
voice.

Iflytek/Xfyun host TTS uses these secret env names only:
`A21_IFLYTEK_TTS_APP_ID`, `A21_IFLYTEK_TTS_API_KEY`, and
`A21_IFLYTEK_TTS_API_SECRET`. Reports may store `provider=iflytek_tts`,
`endpoint_host=tts-api.xfyun.cn`, `network_mode=direct`, aggregate timing, and
PCM quality, but must not store auth query values, credentials, input text,
provider output, raw/base64 audio, full URLs, proxy values, or local paths. The
WebSocket client intentionally ignores ambient `HTTP_PROXY` / `HTTPS_PROXY` in
the current direct mode; explicit WebSocket provider-proxy support needs its
own adapter transition.

The seam decodes valid uplink Opus frames to PCM16 to produce aggregate telemetry
(`decoded_frame_count`, `decoded_sample_count`, `decoded_duration_ms`) and an
honest `decode_status` such as `opus_decoded_pcm16` or `opus_decode_error`.
Decoded PCM also enters the existing `audio.Ingress` buffer and VAD markers so
ASR receives the same observable ingress surface as `/ws/audio`. When decoded
frames include VAD speech, `/v1/xiaozhi` runs the configured
`internal/providers` ASR -> text stream -> TTS pipeline and sends resulting PCM
through the paced Opus downlink. When no speech or no usable decoded frame is
available, it still emits the honest xiaozhi TTS lifecycle placeholder.

For stock product acceptance, `/v1/xiaozhi` is gated by the physical device
`listen.stop`: Gateway must not send speech downlink before that stop, and a
streaming ASR final should surface as stock `stt` before `tts.start`. Immediate
post-answer stock physical `listen.start` plus trailing Opus frames are treated
as playback-tail drain when input suppression is armed; the matching
`listen.stop` ends the suppressed session without starting another answer. The
older `/ws/audio` envelope socket remains a legacy/dev diagnostic and simulator
path for A21 control events and PCM fixtures. Batch/non-streaming ASR fallback
on `/v1/xiaozhi` is below product acceptance until it carries equivalent
`stt -> tts.start` evidence on the stock product socket.

Xiaozhi TTS binary downlink uses the Go `AudioRateController` primitive before
writing frames: default 60 ms frame slots, one-frame prebuffer, per-frame
abort checks, and reset on turn cancellation. Raw unpaced binary writes are
not accepted as an A21 product path.
The current downlink primitive accepts validated 16 kHz, 24 kHz, or 48 kHz mono
60 ms `pcm_s16le` provider audio, applies bounded PCM leveling before Opus
encode, and writes one xiaozhi binary frame through the current-turn pacer. The
leveling leaves tiny noise below the gate unchanged, lifts quiet non-silent TTS
frames up to a target peak with a maximum gain cap, and still keeps hot frames
below the headroom limit. This is a downlink building block, not ASR/LLM/TTS
product acceptance.

Host-side TTS and voice-pipeline reports now include an aggregate PCM quality
guard for generated `pcm_s16le` audio. The guard records format, duration,
sample count, peak/RMS dBFS, clipped-sample ratio, silence ratio, DC offset, and
fixed finding codes such as `audio_quality_clipping_detected` and
`audio_quality_low_headroom`. It must not store raw PCM, base64 audio, prompt
text, transcripts, provider output, full URLs, proxy values, or local paths.

WS-2 adds a host-side product voice pipeline contract under
`internal/providers`. It models decoded PCM frame metadata flowing through ASR,
streaming text, and TTS adapters, then returns downlink-ready
`VoiceAudioChunk` values: `pcm_s16le`, mono, 60 ms, with 48 kHz preferred for
local TTS so Gateway can avoid A21-owned upsampling before Opus encode. The
implementation supports both fixture/mock adapters and host-local adapters
(Sherpa ASR, route-eligible OpenAI-compatible or Ollama text stream, and Sherpa
TTS). It records stage markers such as `asr_first_partial_ms`,
`llm_first_content_ms`, `tts_first_audio_ms`, and
`audio_downlink_first_frame_ms`, preserves provider selection by A21 env/profile
names, and emits a redacted report that stores counts, format metadata, timing,
aggregate audio-quality metrics, and policy fields only. Host-local loopback
evidence proves the server-side product chain, but it is still not physical
StackChan first-audio acceptance, transcript quality acceptance, or final PRD
launch acceptance.

The Gateway xiaozhi fixture path consumes those report fields without storing
transcripts, provider output, or audio payloads in JSON. Its TTS start message
may include `voice_pipeline.schema_version`, `execution_mode`, chunk counts,
and timing fields. The actual mock audio travels only as paced binary Opus
frames; `data_base64` is not emitted in xiaozhi JSON.

`a21 xiaozhi-voice-bench --require-product-chain` rejects fixture-only or
partial host-local runs. It exits non-zero unless the Gateway trace proves one
non-fixture ASR stage, one non-fixture text-stream stage, and one non-fixture
TTS stage under `voice_pipeline.execution_mode=host_local`. Reports keep only
safe profile/env identifiers and must not store prompt text, transcript text,
provider output, full URLs, proxy values, credentials, raw audio, or local
paths.
`product-readiness` treats xiaozhi host product-chain evidence as usable only
when the report includes `repeat >= 3`, at least three answer turns, at least
three barge-in turns, zero failures, answer first-audio p95 under 1500 ms, and
barge-in stop p95 under 300 ms.

Text-stream fallback is provider-neutral. `A21_TEXT_STREAM_FALLBACK_PROFILE`
selects a secondary configured route-eligible text provider for the host-local
voice pipeline, and `local-voice-loopback --fallback-text-provider` exposes the
same behavior to the host verifier. Gateway and reports may emit a coarse
`provider_fallback_used` finding plus fallback provider/reason metadata, but
must not emit primary/fallback prompt text, transcripts, provider output,
reasoning, API keys, model values, full URLs, or proxy values. The streaming
xiaozhi answer path carries the same redacted fallback metadata on answer
`voice_pipeline` summaries instead of waiting for a final non-streaming report.

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

### Wake Word Configuration

Gateway exposes `GET|PUT /v1/wake-word` for the local simulator and operator UI.
It stores an A21 wake-word intent profile, not live stock-firmware behavior.
The stock xiaozhi WakeNet model remains active as `你好小智` until a dedicated
xiaozhi/ESP-SR MultiNet firmware build is produced and flashed through the
guarded firmware lane. Custom requests therefore return
`runtime_status=pending_firmware_build`, `firmware_build_required=true`, and
`code=a21_wake_word_firmware_build_required` instead of pretending the Gateway
can dynamically replace the device-side wake model.
All responses also carry machine-readable activation truth:
`firmware_status=builtin_active|custom_pending_firmware`,
`active_runtime_profile=builtin_xiaozhi_wakenet`,
`desired_firmware_profile=custom_multinet` only for custom requests,
`runtime_hot_swap_supported=false`, and `custom_runtime_active=false`.

The request accepts `mode=custom_multinet`, `desired_phrase`, `desired_pinyin`,
and a numeric `threshold` in the safe range 1-100. Gateway rejects secret-like,
URL-like, legacy-looking, malformed, oversized, or trailing-payload values
before persistence. The PUT body must be one JSON object no larger than 4 KiB.
The config path is `A21_WAKE_WORD_CONFIG_PATH` when set; otherwise Gateway
writes the local runtime file `.a21-run/gateway/a21-wake-word.json`. Reports and traces
must not store Wi-Fi credentials, provider keys, raw audio, transcripts,
prompts, full URLs, proxy values, or absolute local paths for this feature.

`a21 wake-word-firmware-plan [--config ...] [--output-dir reports]` turns the
stored intent into a machine-readable firmware plan. Built-in Xiaozhi remains a
`builtin_noop`. Custom MultiNet requests produce
`schema_version=a21.wake_word_firmware_plan.v1`,
`status=pending_firmware_build`, `firmware_build_required=true`,
`build_allowed=false`, and `flash_allowed=false`; the report names only the A21
firmware identity, CoreS3 board, target MultiNet profile, confirmation value,
next actions, and basename report path. It does not build, flash, execute a
provider, contact hardware, or log local config/report paths.
`product-readiness` can ingest the report through
`--wake-word-firmware-plan` or `--use-latest-reports`, but ingestion is evidence
of a controlled plan only. It is not evidence that the custom wake model is
compiled into firmware or active on StackChan hardware.

`a21 wake-word-firmware-build-receipt --plan <plan.json>
--build-dir <xiaozhi-build> --review-report <review.json>
[--output-dir <receipt-dir>]` is the bridge from a reviewed external
xiaozhi/ESP-SR build to A21 packaging. `--build-review` is accepted as an alias
for `--review-report`. The review file must be
`schema_version=a21.wake_word_firmware_build_review.v1`, `status=reviewed`,
match the plan's firmware identity, CoreS3 target, custom MultiNet mode,
desired phrase, desired pinyin, and threshold, and name only basename build
artifacts: `app_binary`, `sdkconfig`, and `flasher_args`. Optional
`reviewer`/`build_tool` fields are allowed only as redacted labels. The command
validates the custom plan, review, `sdkconfig.json`, `flasher_args.json`, and
`xiaozhi.bin`, prints a redacted `a21.wake_word_firmware_build.v1` receipt with
the review basename, and writes `a21-wake-word-build.json` only when
`--output-dir` is supplied.

`a21 wake-word-firmware-package --plan <plan.json> --build-dir <xiaozhi-build>
[--build-receipt <receipt.json>] --commit <sha>
[--output-dir firmware/artifacts/wake-word]` is the next no-hardware boundary.
It requires the matching custom MultiNet plan, an explicit or build-dir-local
`a21-wake-word-build.json` receipt that names a reviewed-build report by
basename, CoreS3 `sdkconfig.json`, `flash_args`, and the required flash parts.
It writes an A21-named app `.bin`, `.sha256`, `.manifest.json`, and package
report with basenames and hashes only, including `build_review`. It does
not flash, touch a serial port, start Gateway, execute providers, or make the
wake word product-ready. The `product-readiness` command accepts
`--wake-word-firmware-package-report <report.json>` as below-activation evidence
under `wake_word`, but only the existing basename package source, artifact, and
manifest names may be surfaced there; `build_review` remains package-report
provenance. The rollup must remain `launch_ready=false` until guarded flash and
physical custom wake proof are present.
If `--build-dir` is absent or the build receipt is missing, the command emits
and writes a structured diagnostic package report with
`status=missing_build_dir|missing_build_receipt`, finding code
`wake_word_firmware_build_dir_missing|wake_word_firmware_build_receipt_missing`,
`package_written=false`, `flash_allowed=false`, and `product_ready=false`.

`a21 wake-word-physical-proof --physical-device-online
--firmware-flash-executed --guarded-flash-report <guarded-flash.json>
--operator-observed --wake-phrase-matched --false-wake-rejected
--stock-wake-rejected [--output-dir reports]` records the operator/hardware
proof after a guarded flash has already happened. It writes
`a21.wake_word_physical_proof.v1` with basename-only
`guarded_flash_report_source`, observed booleans, explicit false/stock wake
rejections, `redaction_ok=true`, and basename-only `report_path`. The command
does not touch serial, flash firmware, start Gateway, run providers, or play
audio; missing affirmative flags or unsafe report names reject without writing
an observed proof report.

`a21 wake-word-physical-acceptance --package-report <package.json>
--proof-report <physical-proof.json> [--output-dir reports]` is the post-flash
custom wake-word acceptance boundary. It consumes a matching
`a21.wake_word_firmware_package.v1` package report plus a redacted
`a21.wake_word_physical_proof.v1` operator/hardware proof and writes
`a21.wake_word_physical_acceptance.v1`. The acceptance report stores only
basename package/proof pointers and booleans for physical device online,
guarded flash executed, operator custom-wake observation, phrase match,
false-wake rejection, stock-wake rejection, and redaction. It does not flash,
touch serial, start Gateway, run providers, store audio, or store paths/URLs.
`product-readiness --wake-word-physical-acceptance-report <report.json>` closes
only `wake_word.product_ready` and server-side `wake_word_ready` when the
acceptance report matches the current Gateway custom MultiNet intent and the
same firmware package evidence. Full launch still requires the separate
physical StackChan PRD acceptance report.

### Fast Companion Runtime

`POST /v1/fast-companion/turn` remains a Gateway-owned fast companion seam.
When `local_audio.frames` is omitted, it reports the existing text-stream
boundary placeholder and does not execute a provider. When callers include
bounded `pcm_s16le` mono local-audio frames, Gateway decodes them only in
memory and passes them to the same provider-neutral `VoicePipelineRunner` used
by the xiaozhi runtime. Responses may include control events and audio playback
chunks, but must not echo transcripts, provider text, raw audio, base64 input
audio, full URLs, credentials, proxy values, or local paths.

Fast companion accepts the `roleplay` product mode and the legacy
`workmate`/`companion` runtime labels used by internal test 3. If the selected
`voice_mode` is `professional`, the endpoint returns `409` before provider or
V21 execution and points the caller to the professional path.

For internal test 4, the fast-companion response includes a redacted
`roleplay` runtime summary. It carries only selected profile IDs, scenario,
voice-clone profile, memory policy/count readiness, and boolean storage/route
guards. It must not return prompt text, memory text, transcripts, provider
output, voice-clone samples, URLs, credentials, local paths, or V21 evidence.
Gateway records `roleplay.profile.ready` for accepted turns and
`roleplay.memory.ready` only when memory hints are configured as prompt input.

### Xiaozhi MCP And Expression Contract

The xiaozhi transport package now carries a host-only WS-5 contract for future
device-control integration and the Gateway exposes stock-safe live control
surfaces:

- MCP JSON-RPC envelopes are limited to `initialize`, `tools/list`, and
  `tools/call` request shapes. The transport package builds and parses the
  envelopes; Gateway execution is explicitly limited to documented, whitelisted
  live tools.
- `POST /v1/xiaozhi/speaker-volume` sends the official stock firmware MCP
  `tools/call` for `self.audio_speaker.set_volume` to the already connected
  `/v1/xiaozhi` WebSocket. It requires a valid `device_id`, `volume` in
  `0..100`, an online socket, and `hello.features.mcp=true`. Delivery proves the
  MCP command was sent to the device socket; physical loudness acceptance still
  requires operator or instrument evidence.
- `POST /v1/xiaozhi/mcp-control` is the low-risk official MCP status/control
  parity surface. It sends only whitelisted stock `tools/call` messages for
  `self.get_device_status`, `self.screen.set_brightness`,
  `self.screen.set_theme`, and `self.screen.get_info` to an already connected
  `/v1/xiaozhi` WebSocket. It requires a valid `device_id`, an online socket,
  `hello.features.mcp=true`, and trace/session/device identity. Missing
  `trace_id` or `session_id` is filled with A21 IDs. Brightness is bounded to
  `0..100`; theme is a bounded token without URL/path characters; status and
  info requests accept no user arguments. The endpoint rejects non-whitelisted
  tools, including reboot, firmware upgrade, camera/photo, screen snapshot,
  camera stream/video, NFC, infrared, and app lifecycle controls, before any
  MCP websocket write. Delivery proves only that the MCP request was sent; it
  does not capture raw MCP responses and is not physical screen/status product
  acceptance.
- Official StackChan/Xiaozhi status-display parity is recorded as A21 device
  registry state, not as custom firmware drawing. Gateway normalizes official
  state words into the stable A21 `display_state` vocabulary: `starting`,
  `wifi_configuring`, `idle`, `connecting`, `listening`, `thinking`,
  `speaking`, `upgrading`, `audio_testing`, `error`, and `fatal_error`.
  Official aliases such as `activating` normalize to `connecting`; unknown,
  legacy-looking, X21-looking, or V21-looking state names normalize to `error`.
  Registry updates carry `display_state_source`, `display_state_trace_id`,
  `display_state_session_id`, `display_state_updated_at_ms`, and
  `display_state_physical_accepted=false` until a separate physical screen or
  runtime echo acceptance report proves the device actually rendered the state.
  The required trace markers are `stackchan.display_state.received`,
  `stackchan.display_state.normalized`, and
  `stackchan.display_state.registry_updated`.
- `POST /v1/xiaozhi/say` is an operator foreground test path for an already
  connected stock Xiaozhi device. It validates `device_id` and exactly one
  playable source: either `text` or `wav_path`. `text` uses the configured
  Gateway TTS path; `wav_path` must point to a local A21-compatible 16 kHz mono
  PCM WAV and is converted into the same downlink chunk shape used by TTS. The
  Gateway starts a turn on the live `/v1/xiaozhi` WebSocket, writes the stock
  TTS lifecycle (`start`, `sentence_start`, binary Opus downlink, `stop`), and
  returns only after the audio has been delivered. Responses for WAV playback
  may include `audio_source=wav_file` and the safe file basename only; they must
  not include the full path, raw/base64 audio, transcript, provider output,
  credentials, proxy values, or local path values. This is for physical
  speaker/TTS A/B tests and urgent operator playback; it does not replace
  wake-word activation, normal listen/ASR flow, or physical PRD acceptance. For
  stock physical devices it also arms a short post-`say` input suppression
  window so speaker playback is less likely to be captured as a fresh
  listen/voice-pipeline turn.
- `tools/call` builders validate tool names, reject legacy-looking X21/V21
  names, and redact sensitive argument fields such as keys, tokens, prompt or
  transcript text, raw/base64 audio, provider output, URLs, proxies, and local
  paths.
- The product audio contract is stock-compatible Xiaozhi voice firmware plus
  A21 Gateway `/v1/xiaozhi`: Opus uplink/downlink, listen/abort, turn cancel,
  pacing, and playback traces must stay compatible with Xiaozhi audio
  expectations.
- Product screen and body behavior are not implemented by custom drawing inside
  the Xiaozhi voice firmware. A21 product avatar/action must route through the
  official StackChan avatar/action implementation, with A21 only supplying
  bounded OEM semantic events. Later personality-specific expressions may be
  layered on that official adapter, not on a new hand-drawn face engine.
- Stock Xiaozhi `type=llm` messages may still carry a stock `emotion` field for
  compatibility, but that is not the A21 product avatar contract and cannot be
  used as physical screen/action acceptance.
- Optional motion parameters clamp `y_angle` to the stock-safe 5-85 degree
  range before any later adapter may send them.
- `internal/transport/stackchan` is the host-side official StackChan adapter
  contract. It maps A21 semantic `state`/`face`/`motion` events into the
  official StackChan WebSocket binary frame shape: `ControlAvatar` (`0x03`),
  `ControlMotion` (`0x04`), or `DanceSequence` (`0x14`), with a 1-byte type,
  4-byte big-endian payload length, and official JSON payload consumed by
  `updateAvatarFromJson`, `updateMotionFromJson`, or `DanceModifier`.
  The adapter exposes an `OfficialActionPlan` with `packet_count`,
  `official_action_surfaces`, and
  `official_action_physical_accepted=false` metadata so operator tools can see
  which avatar, pitch, yaw, and RGB semantic surfaces were requested without
  treating delivery as physical acceptance. `servo_x` yaw sequences are marked
  as candidate-only and clamped before frame construction. RGB is currently a
  semantic `*_no_rgb_frame` marker because this adapter does not emit an
  official RGB packet. `display`, `heartbeat`, camera, video, call, and
  playback diagnostics remain outside this avatar/action adapter.
- Gateway exposes `/stackChan/ws` as the official StackChan avatar/action relay
  path for upstream-compatible clients. A21 validation may send semantic
  commands through `POST /v1/stackchan/official/control`; Gateway then writes
  official binary avatar/action packets to the connected `/stackChan/ws`
  socket and returns only redacted action metadata: packet count, semantic
  surface labels, and physical-accepted false. The same metadata is mirrored
  into `/v1/devices.runtime_echo` with `official_stackchan_` keys. This relay
  is separate from the stock Xiaozhi voice socket and must not add A21 visual
  semantics to Xiaozhi stock audio messages.

### A21 StackChan Device Extension

WS-5 also defines an A21-only StackChan semantic device extension in
`internal/transport/xiaozhi`. It is a host/Gateway OEM schema, builder, and
parser proof for a future official StackChan avatar/action adapter; it does not
make `type=device` part of the stock Xiaozhi audio profile, and it does not
authorize A21-specific display rendering inside the Xiaozhi voice firmware.
Stock hello and server hello remain free of debug or device-extension
requirements. A debug client that explicitly advertises
`features.device_events=true` receives only an A21-namespaced server allowance:
`a21.profile=debug` and `a21.device_events=true`. Stock server hellos remain
free of `a21`, `device_events`, and `debug_metrics`. A host may build
`type=device` extension events only when the connected profile explicitly
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

For launch readiness, the extension's visual/action events are candidate
transport evidence only until a flashed official StackChan avatar/action
adapter emits physical observation or accepted runtime echo. Gateway delivery
of `state`, `face`, `display`, or `motion` is not PRD screen/action acceptance.

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

A21 mode values are semantic product and office-state signals, not provider names.
The launch product modes are only:

- `roleplay`
- `professional`

Compatibility and state labels may still appear on lower-level runtime/control
surfaces:

- `workmate`
- `companion`
- `co_creation`
- `dialogue`
- `focus`
- `public`
- `private`
- `muted`
- `local_fallback`
- `error`

Only `professional` with `professional_only` privacy is allowed to trigger the V21 adapter path. `focus`, `public`, `private`, and `muted` are office visibility/privacy states and must remain visible to the user without silently becoming professional retrieval context. Explicit `private` privacy keeps Agent I/O and professional/V21 evidence routing blocked, even if a caller also asks for professional evidence.

`voice_mode` is the operator-visible product-mode selector for the converged
launch surface. `GET /v1/voice-modes` returns `a21.gateway.voice_modes.v1` with
only `roleplay` and `professional`. `roleplay` is the default low-latency
embodied role/personality chain with memory hints and voice-clone selection.
`professional` is the evidence-first V21 adapter path and must be rejected by
roleplay/dialogue-only endpoints instead of silently switching provider, V21,
or firmware behavior. Legacy labels such as `dialogue`, `workmate`,
`companion`, and `co_creation` normalize to `roleplay` at product-contract
surfaces; visibility/privacy states remain separate policy fields.
The same response carries `selected_ritual` and per-mode `ritual` metadata so
web/app/hardware surfaces can show one consistent mode switch ritual. The
professional ritual uses screen label `PRO`, expression `professional`, trace
marker `professional.checking_feedback.sent`, workspace policy
`professional_only`, `v21_allowed=true`, and `physical_accepted=false`. Its
cue text is the same checking acknowledgement used before V21 query execution:
"我在查，先把证据和置信度拉出来。" The ritual contract must not include user
utterances, workspace text, evidence bodies, provider output, URLs, paths,
credentials, or raw audio.

Explicit spoken professional triggers are allowed only as a user-initiated mode
switch from default/roleplay/workmate/companion contexts into the existing
professional path. Recognized phrases include "专业模式", "认真查一下",
"帮我查 V21", and "给我证据". Negated phrases such as "不要进专业检索",
"不用专业模式", and "别查 V21" must not route to V21, and `focus`,
`public`, `private`, and `muted` remain policy/state modes that are not
silently upgraded. Gateway traces only `professional.voice_trigger.detected`
and, on the stock Xiaozhi socket, `xiaozhi.professional_route.voice_trigger`;
it must not trace trigger utterance text, evidence bodies, provider output,
document text, URLs, paths, credentials, voice transcript, or audio. When the
Xiaozhi streaming ASR final is already available, the professional route reuses
that final text instead of running a second batch ASR pass.

`roleplay_profile` is the frontend/Gateway selector for the default embodied
roleplay runtime under `voice_mode=roleplay`. `GET /v1/roleplay-profile`
returns `a21.gateway.roleplay_profile.v1`, the selected roleplay profile,
scenario, voice-clone profile, redacted memory readiness, a redacted runtime
summary, and safe option catalogs. `POST` or `PUT /v1/roleplay-profile` can
change the selected roleplay scenario, voice-clone profile, and bounded
session memory hints through `memory_hints`; `clear_memory=true` clears only
Gateway runtime roleplay hints, not environment-provided operator hints or any
professional/V21 workspace. Selecting a voice-clone profile through this
endpoint also updates the existing
`voice_chain_profile` selection so roleplay voice clone reaches the selected
TTS boundary without adding a second provider selector. The current profile
catalog intentionally exposes only `a21_roleplay_default`; scenario options are
playbook labels under the roleplay product mode, not new product modes.

The roleplay runtime composes personality prompt input only in memory and only
from safe memory hints exposed by the personality package. Runtime memory hints
are filtered by the same bounded policy as `A21_MEMORY_*`: unsafe URLs, local
paths, and credential-looking strings are rejected before storage, long hints
are truncated, and only sanitized prompt-input text is kept in Gateway memory.
For roleplay voice turns, Gateway passes the composed personality/scenario/
memory prompt to the provider-neutral voice pipeline text-stream request while
keeping the prompt out of `VoicePipelineReport`, traces, and API responses.
Reports may expose `prompt_input_ready` and redaction policy
`prompt_input_not_recorded`, but must not expose the prompt body. Responses,
simulator readouts, and traces must report readiness booleans, counts, and
finding codes rather than storing or echoing the composed prompt, raw memory
text, user transcript, provider text, voice sample, professional evidence, or
V21 query/result. `professional_route_allowed` and `v21_executed` must remain
false on this endpoint and on fast-companion roleplay turns.

`professional_workspace` is the Gateway-owned professional query-scope
contract for internal test 4. `GET /v1/professional-workspace` returns
`a21.gateway.professional_workspace.v1`, adapter contract
`a21.v21_adapter_query.v2`, selected redacted `user_id`, `workspace_id`,
`query_scope`, `professional_only` privacy, query-scope options, and storage
redaction flags. `POST` or `PUT /v1/professional-workspace` can update
redacted labels and one of three query scopes: `public_only`, `personal_only`,
or `personal_plus_public`. This endpoint does not upload files, index
documents, execute V21, store document text, store utterance text, store
retrieved text, or persist provider output.

When the professional path does execute V21, Gateway sends the same v2 scope
fields to the A21/V21 adapter: `device_id`, `user_id`, `workspace_id`,
`query_scope`, `privacy_scope=professional_only`, `latency_profile=fast_first`,
`answer_style=voice_first_with_citations`, and `max_first_response_ms=1200`.
Traces may record `professional.workspace.ready` and
`professional.query_scope.<scope>`, but must not record user labels, workspace
labels, utterance text, evidence bodies, full URLs, credentials, local paths, or
private document contents. V21 adapter responses may include redacted
`source_scope_counts` and `workspace_status`; these are counts/status only, not
retrieved document bodies.

`professional_read_records` is the Gateway-owned read ledger for professional
queries. `GET /v1/professional-read-records` returns
`a21.gateway.professional_read_records.v1` with memory-only records that can be
filtered by `record_id`, `trace_id`, or `session_id`. A record starts before a
professional V21 query and finishes as `completed` or `failed`. Stored fields
are limited to safe IDs, redacted `user_id`/`workspace_id`, `query_scope`,
`privacy_scope`, `latency_profile`, `answer_style`, `utterance_bucket`,
safe `source_scope_counts`, safe `workspace_status`, timestamps, and redaction
flags. It must not store utterance text, retrieved text, evidence bodies,
screen-card text, speech blocks, provider output, document text, full URLs,
local paths, credentials, API keys, voice transcripts, or audio.

`workspace_upload_jobs` is the no-execute upload/import/index job contract for
the A21 workspace surface. `GET /v1/workspace-upload-jobs` returns
`a21.gateway.workspace_upload_jobs.v1` and redacted job metadata. `POST
/v1/workspace-upload-jobs` creates a job from metadata only:
`user_id`, `workspace_id`, `source_scope`, `source_kind`, `document_label`,
`content_type`, `size_bytes`, and optional trace/session/device IDs. `PUT
/v1/workspace-upload-jobs` accepts low-risk control actions: `mark_failed`,
`retry`, `mark_searchable`/`mark_indexed_metadata_only`, and `delete`. The
endpoint must reject raw document text, file bytes, base64 payloads, import
URLs, local paths, credentials, API keys, and provider outputs. Current job
status values such as `accepted_no_execute`, `failed`, `deleted`, and index
statuses such as `not_started_no_execute` or `searchable_metadata_only` are
contract/readiness metadata only. Jobs created directly on this endpoint do not
mean upload bytes were stored or V21 indexing ran. Jobs linked from
`/v1/workspace-documents` may carry `status=stored_local_pending_index`,
`storage_status=stored_local`, and a safe `document_hash`; they still keep
`index_status=not_started_no_execute` and `v21_execution_allowed=false`.

`workspace_documents` is the local document-upload intake contract for the A21
workspace surface. `POST /v1/workspace-documents` accepts only
`multipart/form-data` with one `file` part plus safe metadata fields:
`user_id`, `workspace_id`, `source_scope`, `document_label`, `content_type`,
`trace_id`, `session_id`, and `device_id`. Gateway stores bytes only in an
A21-owned runtime store, using `A21_WORKSPACE_DOCUMENT_STORE_DIR` when set or
`.a21-run/gateway/workspace-documents` by default. Single-upload size is
conservatively bounded by `A21_WORKSPACE_DOCUMENT_MAX_BYTES` when set or the
Gateway default. The response schema is
`a21.gateway.workspace_documents.v1` and returns only safe document/source/job
metadata: IDs, source scope, content type, size, `sha256:` document hash,
`storage_status=stored_local`, `readiness=stored_local_pending_index`, and
redaction flags. It must not return or trace document text, raw bytes, base64
payloads, original private filenames, local storage paths, import URLs,
credentials, provider output, V21 evidence, or document-derived text. This
endpoint does not parse, chunk, embed, OCR, index, upload to cloud storage, or
execute V21.

`workspace_sources` is the memory-only source/readiness registry derived from
workspace upload/import job metadata. `GET /v1/workspace-sources` returns
`a21.gateway.workspace_sources.v1` and can filter by `source_id`, `user_id`,
`workspace_id`, and `source_scope`. Source records are limited to redacted
source/job IDs, redacted user/workspace labels, source scope, source kind,
document ID/hash when present, document label, content type, size, readiness,
storage status, index status, safe timestamps, and redaction flags. Readiness
values such as `metadata_only`, `stored_local_pending_index`,
`searchable_metadata_only`, `failed_metadata_only`, and
`deleted_metadata_only` are explicitly scoped: `stored_local_pending_index`
means A21 accepted local bytes and has not indexed them; the other metadata
statuses are not proof of real document storage, retrieval, or V21 indexing.
`/v1/professional-workspace` may surface
`source_scope_counts`, `searchable_source_scope_counts`, and
`query_scope_readiness` from this registry while keeping
`v21_execution_allowed=false` until a separately verified adapter execution
path is used.

`gateway_profile` is the operator/frontend transport selector for where the
StackChan product connects. It is independent from `voice_mode`: selecting
`public_wss` must not turn roleplay into professional mode, and selecting
`professional` must not silently move the device to a public Gateway.
`GET /v1/gateway-profiles` returns `a21.gateway.profiles.v1` with `mac_local`
and `public_wss`. `public_wss` is the default profile when
`A21_PUBLIC_GATEWAY_URL` is configured, because the Aliyun edge is the main
product Gateway. The production endpoint should be `https` or `wss` without
credentials; IP-only ECS bring-up may temporarily use `http` or `ws`, which
normalizes to `ws://.../v1/xiaozhi`. `mac_local` remains selectable for
Mac-local models and local processing. Public control surfaces should configure
`A21_MAC_LOCAL_GATEWAY_URL` when they need to display or select the actual Mac
Gateway address; otherwise `mac_local` falls back to the local request-host
WebSocket URL. `POST /v1/gateway-profiles` changes the selected profile, and
`/xiaozhi/ota/` uses that profile to return either the configured Mac/local
WebSocket URL, the local request-host WebSocket URL, or the configured public
WebSocket endpoint.

`voice_chain_profile` is the operator/frontend selector for the dialogue voice
pipeline shape. It is independent from `voice_mode`, `gateway_profile`, and the
catalog-only `cloud_voice_profile` surface. `GET /v1/voice-chain-profiles`
returns `a21.gateway.voice_chain_profiles.v1` with two chain modes:
`cascade` and `realtime`. `cascade` is the default ASR -> LLM -> fixed TTS
path; it exposes selectable ASR and LLM profiles while keeping TTS fixed so
latency testing does not silently change voice quality. The recommended LLM is
`stepfun`; `deepseek` remains visible only as a fallback option. `realtime`
exposes the end-to-end realtime provider used by `/v1/realtime/session`.
`POST` or `PUT /v1/voice-chain-profiles` hot-switches the in-memory Gateway
runtime by updating the selected ASR, LLM, realtime provider, and voice clone
profile. The hot switch sets the existing `A21_GATEWAY_VOICE_PROVIDER=selected`
runtime gate before selecting `A21_PROVIDER_PRIMARY`, so the realtime boundary
uses the selected realtime provider instead of the mock voice provider.

Voice clone selection is a separate voice/tonality choice under the same
selector response. Public UI should show safe voice names such as `A21 natural
voice` or `A21 cloned voice`, not provider secrets, raw model IDs, reference
text, reference audio, full URLs, or local file paths. Selecting a clone maps
the effective TTS profile to `voice_clone_cli`; selecting the default natural
voice keeps the fixed cascade TTS at `dashscope_qwen_tts_realtime`. The device
registry exposes only selected profile IDs and safe names:
`current_voice_chain_mode`, `current_asr_profile`, `current_llm_profile`,
`current_tts_profile`, `current_realtime_provider`, and
`current_voice_clone_profile`.

For roleplay voice turns, the selected safe `voice_clone_profile` is carried as
runtime-only provider-neutral metadata on `VoicePipelineRequest` and
`TTSAdapterRequest`. `VoicePipelineReport.input.voice_clone_profile` may expose
only the safe A21 profile ID, and the report redaction policy must include
`voice_clone_sample_not_recorded`. Trace output may record
`roleplay.voice_clone_profile.used` when a clone profile is selected, but must
not record voice samples, reference audio/text, local paths, URLs, raw model
IDs, credentials, prompt text, transcripts, provider output, or raw/base64
audio.

StackChan Wi-Fi provisioning is device-side and follows Xiaozhi's startup
model. Stored NVS credentials are tried first. If none are available, the
firmware enters Wi-Fi provisioning instead of requiring a hardcoded SSID or
falling into a silent local-only state. The A21 official-compatible product
overlay selects Hotspot/SoftAP captive portal by default and keeps BluFi and
acoustic provisioning as explicit build-time alternatives. Provider keys and
Gateway secrets stay server-side and are never stored in Wi-Fi provisioning
payloads or firmware.

`local_fallback` is both a mode and an expression state. Gateway enters it when
the local voice/provider pipeline cannot produce a playable answer after local
listening has started. The user-facing fallback sentence is fixed and local; the
response, trace, and device registry must not store or leak prompt text, raw
audio, provider output, provider keys, full URLs, proxy URLs, or local paths.

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
- `current_voice_mode`: current explicit operator voice-mode selection
- `current_expression`: latest expression/render state from A21 `control.event`
- `display_state`: latest stable A21 status-display state normalized from
  official Xiaozhi/StackChan state words or A21 device events
- `display_state_source`: `xiaozhi`, `device_event`, `official_stackchan`, or
  `unknown`
- `display_state_trace_id` and `display_state_session_id`: trace/session that
  caused the latest display-state update
- `display_state_updated_at_ms`: Gateway timestamp for the latest
  display-state update
- `display_state_physical_accepted`: always false until a separate physical
  screen/status acceptance report exists
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
- `mode`: optional legacy control label, default `workmate`; user-facing
  `voice_mode` defaults to `roleplay`
- `text`: optional short screen/status text
- `trace_id` and `session_id`: optional explicit trace/session IDs
- `stream_id`: required when the caller wants a stable speaking stream; generated only for simple speaking validation
- `diagnostic_tone_hz`, `diagnostic_tone_duration_ms`, `diagnostic_tone_volume`: optional physical speaker diagnostic tone. This is a device-local M5Unified tone path for isolating speaker/I2S/amplifier behavior from Gateway WAV/base64/playback-buffer streaming. It must not be used as product TTS.
- `mock_audio_chunks`: 0-8 non-silent chunks for physical speaker validation
- `audio_chunks`: 0-8 caller-provided playback chunks. Each chunk must be `pcm_s16le`, mono, 16/24/48 kHz, 20 or 40 ms, and its `stream_id` must match the request `stream_id`.
- `audio_probe_only`: optional diagnostic flag for the requested trace/session. When true, Gateway keeps accepting and measuring matching `audio.frame` uplink frames but suppresses mock listening/speaking/playback responses. Use this for physical microphone probes so Gateway does not force StackChan into `speaking` while measuring capture.
- `mock_playback_on_next_audio_frame`: optional physical validation flag for the requested trace/session. When true, Gateway arms the next matching physical StackChan `audio.frame`, emits `mock_audio_chunks` playback chunks when explicitly set from 0 to 8, defaults to one chunk when omitted, then immediately disarms.
- `realtime_on_next_speech`: optional physical provider flag for the requested trace/session. When true, Gateway arms exactly one provider realtime-session start for the next matching physical StackChan speech frame. This is separate from mock playback validation and prevents ambient office audio from opening realtime provider sessions by accident.

The audio WebSocket mock path keeps simulator devices silent by default, but physical `stackchan-*` devices can receive a short non-silent PCM chunk after an explicit one-shot validation arm or after the VAD `speech_start` uplink frame. Later frames in the same speech segment are measured but do not produce more mock playback. That distinction lets the half-duplex acceptance prove a real microphone-to-Gateway-to-speaker loop without creating a mic/playback feedback loop, changing simulator expectations, or claiming a real provider response.

Stock Xiaozhi product firmware uses a separate physical half-duplex gate:
`stackchan-accept --check xiaozhi-half-duplex`. This gate reads stock
`/v1/xiaozhi` traces and `/v1/audio/recent` instead of diagnostic device
runtime counters. It may derive `trace_id` and `session_id` from the device
registry's latest `last_trace_id` and `last_session_id`, then requires real
device online evidence, Opus decode, PCM ingress, listen/VAD lifecycle,
TTS Opus downlink, playback observation when available, and barge-in stop
markers. Its report schema is `a21.xiaozhi_half_duplex_acceptance.v1`, its
scope is `stock_xiaozhi_mic_to_tts_downlink`, and
`diagnostic_mic_probe_required=false`. It never promotes PRD readiness by
itself; a passing machine trace is still `physical_review_required`, while a
hello-only or downlink-only trace remains blocked/candidate evidence.

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

The offline realtime fixture report is a reporting surface, not a new device or
Gateway protocol. `provider-realtime-fixture --execute --output-dir reports`
uses a fake WebSocket connection to exercise the provider adapter and writes
`a21-provider-realtime-fixture-*.json` with redacted env-name labels,
endpoint-host-only network metadata, explicit route eligibility, and a
basename-only `report_path`. `product-readiness` may surface this under
`provider.realtime_*`; it must not promote offline fixture evidence to
`real_provider_ready`, `launch_ready`, or PRD acceptance.

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
