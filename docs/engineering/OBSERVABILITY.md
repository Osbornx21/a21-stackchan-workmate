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
- wake-word runtime/config status, including `pending_firmware_build` when a
  custom MultiNet profile is stored but stock Xiaozhi WakeNet remains active
- optional V21 adapter health when `A21_V21_ADAPTER_URL` is configured

The current gateway also exposes `GET /metrics` for Phase 3A mock runtime metrics.

Gateway now also exposes `GET /v1/devices` for the current in-memory device registry. It records the latest control WebSocket device event, firmware identity, StackChan capability map, firmware `runtime_echo` map, identity validation status, current mode, current expression, active playback stream, last trace/session IDs, first/last seen timestamps, and read-time freshness (`connection_status` plus `device_age_ms`). The registry is intentionally transcript-free: it must not persist utterance text, professional answer text, evidence body, or screen-card body.

`runtime_echo` is emitted by firmware after it applies screen, motion, RGB, audio, and diagnostic sensor runtime state. Current stable visible keys are `screen`, `servo_y`, and `rgb`. Diagnostic mic-probe firmware also emits string counters for capture/send debugging, including mic frame count, driver errors, skip reasons, queue depth, queue drops, audio-WebSocket sent frame count, last absolute peak, and last nonzero sample count. The speaker/downlink lane emits playback buffer and speaker pump counters: queued chunks, accepted chunks, dropped chunks, clear count, played frames, busy ticks, driver errors, and last stream ID. The sensor-probe lane emits availability, sample/error counters, raw LTR553 ambient/proximity values, and INA226 battery voltage/current values. This is useful for capability evidence derivation and driver-state debugging, but physical office acceptance still needs separate visible/audible/operator or instrument observations.

Gateway also exposes `GET /v1/traces?trace_id=<trace_id>` for an in-memory mock waterfall. It currently records HTTP mock turn/interrupt receipts, control WebSocket device events, audio frames, audio ingress buffering, VAD adapter start/end markers, microphone probe acceptance markers, mock playback chunk sends, audio-path barge-in markers, realtime provider commit/downlink markers, V21 adapter markers, and outgoing control events with millisecond offsets. The response includes a summary for audio-frame-to-playback, V21 first result, barge-in stop, provider-commit-to-first-audio, event count, and last offset. This is a development observability surface, not the final durable trace backend.

The Gateway VAD path exposes detector-labelled Prometheus counters for each frame decision. This keeps the current deterministic RMS detector visible while allowing future mature VAD/AEC adapters to be compared without changing the audio WebSocket or barge-in contracts.

`internal/audio` now has a Silero VAD adapter boundary with an injectable runner.
It is fixture-ready only: successful runner decisions use the stable
`a21-silero-vad` detector label, and unavailable or failing runners fall back to
RMS with `a21-silero-vad-fallback-rms`. The fallback path must not expose model
paths, local paths, or runner error text in detector labels or reports. This is
not PRD physical VAD acceptance, first-audio acceptance, or evidence that a
Silero model runtime is installed.

The Silero boundary is Go-first and provider-neutral. Gateway tests can select it
through `audio.IngressConfig` without changing the audio WebSocket contract; the
default path remains `a21-rms-vad`. A host-local command runner exists as an
optional executable adapter hook via `scripts/a21_silero_vad.py`: PCM is passed
to the local process on stdin, the process returns JSON fields for
`speech_detected` and `score`, and command/model paths or runner errors are
collapsed to stable A21 status/finding codes before Gateway metrics or
recent-audio metadata see them. If optional Python dependencies or a local model
are unavailable, the runner reports unavailable and RMS remains the baseline
fallback. This hook is not wired to provider, V21, network, firmware, or hardware
execution.

Gateway also exposes `GET /v1/audio/recent` as a loopback-only development capture surface. It is intentionally not a LAN or cloud API. Default responses redact raw PCM and expose only recent frame metadata; `include_audio=1` is for local CLI use when building a temporary ASR WAV for physical StackChan mic-driven evidence. Reports may cite frame count, byte count, RMS, and WAV basename, but must not persist `data_base64` or raw audio.

`audio-front-end-eval --mock` emits a JSON report for the deterministic RMS baseline. `audio-front-end-eval --fixture <path>` emits the same report shape for labelled PCM frame fixtures, including measurable `speech_start_lag_ms` and `speech_end_lag_ms`. With `--output-dir reports`, it also writes `reports/a21-audio-front-end-eval-YYYYMMDD-HHMMSS.json`. The report includes generated timestamp, current commit, network/DNS fingerprint, and redacted proxy-policy metadata without raw PCM frames or proxy secrets. It also carries `candidate_evidence`, `fast_companion_evidence_contract`, host-only identity (`device_id=none_host_fixture`, `baseline_scope=host_only`), explicit no-execute flags for provider, V21, and hardware, and redaction booleans for raw audio, base64 audio, transcripts, prompts, provider output, reasoning, credentials, full URLs, proxy URLs, and full local paths. `report_path` is stored as a basename. These are not runtime Prometheus endpoints; they are offline report shapes future recorded-office and physical-device VAD/AEC evaluations must preserve.

`latency-bench --mock --output-dir reports` emits `reports/a21-latency-bench-YYYYMMDD-HHMMSS.json` with the current commit, generated timestamp, network/DNS fingerprint, and redacted proxy-policy metadata. This makes latency runs comparable across home, Shanghai office, LAN, and proxy configurations without logging proxy URLs or credentials.

`product-readiness` also reads Gateway `GET /v1/wake-word` and includes a
redacted `wake_word` block. Built-in Xiaozhi WakeNet status is product-ready
when Gateway reports it active. A stored custom MultiNet profile with
`firmware_build_required=true` is reported as a launch blocker with
`wake_word_firmware_build_required`; product readiness must not mark it as
launch-ready until the guarded firmware build/flash lane proves the custom wake
model is active.

`wake-word-firmware-plan --output-dir reports` writes
`reports/a21-wake-word-firmware-plan-YYYYMMDD-HHMMSS-*.json` with
`schema_version=a21.wake_word_firmware_plan.v1`. The report is deliberately
no-execute: `dry_run=true`, `build_allowed=false`, `flash_allowed=false`.
Built-in wake-word state is `builtin_noop`; custom MultiNet state is
`pending_firmware_build` with `wake_word_firmware_build_required`,
`a21-stackchan`, `m5stack-cores3`, and `xiaozhi_esp_sr_multinet`. It stores
only sanitized wake-word intent fields and a basename `report_path`, not config
paths, report directories, URLs, keys, proxy values, audio, or transcripts.
`product-readiness --wake-word-firmware-plan <report.json>` and
`product-readiness --use-latest-reports` ingest this report only when it
matches the current Gateway wake-word intent. The rollup then exposes
`firmware_plan_available`, `firmware_plan_status`, basename
`firmware_plan_source_report`, and the plan's no-execute flags under
`wake_word`; it still keeps launch blocked until a later guarded build/flash
and physical acceptance prove the custom model on the device.

`wake-word-firmware-build-receipt` requires an explicit `--review-report`
(`--build-review` alias) with `schema_version=a21.wake_word_firmware_build_review.v1`
and `status=reviewed`; it prints `a21.wake_word_firmware_build.v1` and writes
`a21-wake-word-build.json` only when `--output-dir` is supplied. The review and
receipt record sanitized intent fields plus build-dir basename, review basename,
sdkconfig basename, `flasher_args.json`, and `xiaozhi.bin`; they are build
evidence only and do not imply activation.

`wake-word-firmware-package` writes `a21.wake_word_firmware_package.v1` after a
reviewed xiaozhi/ESP-SR build lane emits or returns a matching receipt that
names the reviewed-build report by basename. The package report records only
sanitized intent fields, basenames, SHA-256 hashes, CoreS3 target metadata, and
flash-part inventory. It must not store build paths, config paths, full URLs,
proxy values, credentials, audio, transcripts, prompts, or provider output.
Package evidence is below activation: `product_ready=false`,
`flash_allowed=false`, and `flash_executed=false`. The `product-readiness`
command accepts `--wake-word-firmware-package-report <report.json>` and exposes
only the existing basename package source, artifact, and manifest names under
`wake_word`; `build_review` stays package-report provenance. It keeps
`wake_word.product_ready=false`, `server_side.wake_word_ready=false`, and
`launch_ready=false` until guarded flash plus physical custom wake proof exists.
When the package command lacks `--build-dir` or `a21-wake-word-build.json`, it
still emits a redacted diagnostic report with
`status=missing_build_dir|missing_build_receipt`; these reports are actionable
operator evidence only and are not accepted as package availability.

`wake-word-physical-proof` writes `a21.wake_word_physical_proof.v1` from
operator-confirmed hardware-window observations. It requires explicit flags for
physical online, guarded flash executed, operator custom-wake observation,
phrase match, false-wake rejection, and stock-wake rejection. It records only a
basename guarded-flash report source and redacted booleans, and rejects missing
affirmations or unsafe names without writing an observed proof.
`wake-word-physical-acceptance` then writes
`a21.wake_word_physical_acceptance.v1` from that proof plus the matching package
report. It records basename-only package source, artifact, manifest, guarded
flash report, and report path fields, plus booleans for physical online, flash
executed, operator observation, phrase match, false wake rejection, stock wake
rejection, and redaction. `product-readiness --wake-word-physical-acceptance-report`
and `--use-latest-reports` accept it only when it matches the current Gateway
custom MultiNet intent and the matching package report. Accepted evidence sets
`wake_word.physical_acceptance_*`, `wake_word.product_ready=true`, and
`server_side.wake_word_ready=true`; it still does not imply `launch_ready=true`
without the separate physical StackChan PRD evidence.

`product-readiness` also emits a `server_side` observability block for the
no-hardware candidate chain. It summarizes the already-redacted Gateway,
provider-smoke, professional V21, host voice loopback, and wake-word readiness
signals into `candidate_ready`, fixed `missing_evidence` labels, and
basename-only source report fields. This block is intentionally below launch
acceptance: a server candidate can be green while `launch_ready=false` and
`requires_physical_acceptance=true`, so dashboards must not treat it as
physical StackChan PRD acceptance.

`product-readiness` is the canonical full-PRD decision report. Its
`canonical_decision` block sets `authority=a21.product_readiness.v1`,
`full_prd_status`, `launch_ready`, `prd_accepted`,
`server_side_candidate_ready`, `host_only_evidence_use=gap_reduction_only`,
`missing_real_evidence`, and `missing_report_fields`. Host-only reports can
remove server-side gaps but cannot make `prd_accepted=true`; missing required
fields from child reports remain explicit `missing_report_fields` entries
instead of being encoded as false readiness.
When any executed server-side evidence has closed a real provider, V21, or
host voice gap but another server-side gate is still missing, the top-level
status is `server_side_blocked` instead of `mock_demo_ready`.

`server-side-readiness-bundle` packages that same no-hardware chain into a
single redacted report artifact, `a21.server_side_readiness_bundle.v1`. It
keeps per-slice readiness, source report basenames, fixed missing-evidence
labels, and safe collection commands for Gateway, provider smoke, V21
professional smoke, host voice loopback, and wake-word readiness. It must not
store prompts, transcripts, provider output, evidence bodies, full URLs,
credential values, proxy values, or local paths, and its `candidate_ready` field
must never be interpreted as physical StackChan PRD acceptance.
The bundle includes the same `canonical_decision` block from product readiness
so report consumers have one launch/PRD authority even when collecting
server-side candidate evidence.
When `--collect-missing` is used, the bundle also includes a `collection` block
with fixed per-step status and reason labels. Host voice loopback can be
collected locally through the Gateway; provider and V21 smoke execution remain
separately authorized by `--execute-provider-smoke` and `--execute-v21-smoke`.
Collection reports must never embed child command stdout/stderr or raw network,
provider, prompt, transcript, evidence, key, proxy, or local path values.

Provider comparison reports must use the shared benchmark contract in `docs/engineering/A21_PROVIDER_BENCHMARKS.md`. External benchmark names and leaderboards may appear in engineering notes, but runtime evidence must use A21 metric names, A21 trace IDs, and redacted A21 reports before it can influence promotion.

The WS-2 product voice pipeline contract emits schema
`a21.voice_pipeline.fixture.v1` from `internal/providers` tests and future
host-side callers. This report is deliberately redacted: it keeps
`trace_id`, `session_id`, `device_id`, execution mode, adapter profile/env
names, audio format counts, output chunk counts, and stage timings, while
recording only policy markers for transcript, provider output, audio payload,
full URL, proxy value, and local path handling. Current fixture timings include
`asr_first_partial_ms`, `asr_final_ms`, `llm_first_content_ms`,
`tts_first_audio_ms`, `audio_downlink_first_frame_ms`,
`speech_end_to_final_asr_ms`, `speech_end_to_first_llm_token_ms`, and, on
cancel, `provider_cancel_ms` plus `barge_in_stop_ms`. These are contract fields
for the mock pipeline and must not be promoted to physical first-audio or real
provider latency evidence without later measured runs.

`provider-latency-bench` is now hardened as a Fast Companion candidate-chain
report shape. Its `metric_terms` block maps TTFS, TTFT, FTTS, and TTFA onto A21
stages and canonical metrics. Its `canonical_metrics` block preserves
p50/p95/p99 series for A21 names such as `transport_ingress_ms`,
`codec_decode_ms`, `asr_first_partial_ms`, `asr_final_ms`,
`llm_first_content_ms`, `provider_first_byte_ms`,
`provider_first_content_ms`, `tts_first_audio_ms`,
`downlink_first_frame_ms`, `audio_downlink_first_frame_ms`,
`device_playback_start_ms`, `barge_in_detected_ms`, `barge_in_stop_ms`,
`provider_cancel_ms`, `provider_cancel_done_ms`, `playback_stop_ms`,
`playback_stop_done_ms`, and the shared provider-benchmark canonical fields.
The unqualified `downlink_first_frame_ms` key is retained only as a legacy
compatibility alias and is still covered by placeholder availability metadata.
For mock/fixture-only reports, its `stage_availability` block marks stages as
placeholders with a fixed reason, source trace marker, sample count, and
p50/p95/p99 redacted stats. Reports also carry `prd_accepted=false`; these
fields are durable JSON report fields, not Prometheus runtime metrics and not
production acceptance evidence.

`provider-latency-bench --mode host_loopback --fixture <report.json>` can also
ingest an already-redacted host-loopback report, such as the
`local-voice-loopback`, `xiaozhi-voice-bench`, or
`a21.virtual_xiaozhi_harness.v1` family. Ingested reports map available
host-only timing fields into the same stage and canonical-metric taxonomy,
including `answer_first_audio_p95_ms` and `barge_in_stop_p95_ms` when enough
samples exist. For virtual Xiaozhi harness aggregate reports,
`first_audio_samples_ms` feeds answer-first-audio metrics and
`abort_stop_samples_ms` feeds barge-in-stop metrics. Missing stages stay
explicit findings. Physical-only acceptance remains blocked unless the source
report contains physical StackChan evidence for device playback.

`physical-stackchan-evidence --fixture <report.json> --output-dir reports`
packages an already-collected physical StackChan fixture into schema
`a21.physical_stackchan_evidence.v1`. It is fixture-only: it does not start
Gateway, providers, V21, firmware tools, serial monitors, or hardware actions.
The report keeps only A21 trace/session/device IDs, execution booleans,
canonical physical metrics, microphone counters, operator/instrument
observation flags, structured findings, and a basename `report_path`. Complete
fixture evidence may reach `promotion_gate=candidate`, but still keeps
`prd_accepted=false` and `acceptance_status=physical_review_required` until an
explicit human review promotes it. Host-loopback fixtures remain
`candidate_host_only`; missing or unsafe physical evidence remains blocked.

`product-readiness --physical-stackchan-report <report.json>` ingests this
redacted report into `stackchan.physical_evidence` using the input basename
only. Candidate physical evidence improves readiness visibility but keeps launch
blocked with `physical_stackchan_review_required`; host-loopback evidence stays
host-only; malformed or unsafe evidence emits the fixed
`physical_stackchan_report_invalid` finding without raw details. A future
accepted report is represented only when it explicitly carries PRD acceptance
and all required physical metrics, microphone counters, and operator/instrument
observations are present.

The live protocol contract reserves future observability fields for binary Opus
media. It is planning-only: reserved trace markers such as
`media.opus.profile.negotiated`, `media.opus.uplink.frame.received`,
`media.opus.downlink.frame.sent`, `media.opus.decode.error`, and
`media.opus.fallback_to_pcm` are not current runtime events. Reserved metrics
such as `a21_media_opus_frames_total`, `a21_media_opus_bytes_total`,
`a21_media_opus_decode_error_total`, and `a21_media_opus_jitter_ms_bucket` are
not current Prometheus metrics.

Future Opus reports may store schema/profile names, codec labels, aggregate
frame counts, aggregate byte counts, timing summaries, and redacted proxy mode
metadata. They must not store raw PCM, base64 audio, Opus payload bytes,
prompts, transcripts, provider output, reasoning, credentials, full URLs, proxy
URLs, model values, or full local paths.

Gateway WS-1 now records xiaozhi compatibility markers for the server seam:
`xiaozhi.hello.received`, `xiaozhi.listen.start`,
`xiaozhi.listen.detect`, `xiaozhi.listen.stop`,
`xiaozhi.listen.stop.ignored`, `xiaozhi.opus_frame.received`,
`xiaozhi.opus_frame.decoded`, `xiaozhi.opus_frame.decode_error`,
`xiaozhi.opus_frame.ignored_not_listening`, `xiaozhi.opus_no_frames`,
`xiaozhi.opus_decoded_pcm16`, `xiaozhi.opus_decode_error`,
`xiaozhi.opus_partial_decode_error`, `xiaozhi.abort.received`, and
`xiaozhi.tts.stop`. Valid decoded xiaozhi Opus frames also emit the ordinary
`audio.ingress.buffered` and VAD markers through the existing ingress path.
Client `hello.features` are represented only as sanitized `/v1/devices`
capabilities: stock `mcp`/`aec` hints stay in the stock profile, while
`device_events` and `debug_metrics` are marked as an isolated debug profile.
When a debug Xiaozhi client negotiates `features.device_events=true`, Gateway
returns the debug-only A21 server-hello allowance `a21.profile=debug` and
`a21.device_events=true`. A `type=device`, `kind=playback`, `playback=start`
extension is recorded as `device.playback.start`; `playback=stop_done` is
recorded as `device.playback.stop_done` and can close the physical barge-in
stop-done metric when it follows `barge_in.detected`. The same messages are
rejected for stock profiles. The repo-owned firmware overlay emits these acks
only after that A21 allowance is present; stock firmware builds keep the option
disabled by default.
The xiaozhi turn foundation adds `xiaozhi.turn.start` on `listen/start` and
`xiaozhi.turn.cancel`, `turn_cancelled`, and `downlink_queue_cleared` on
`abort`; barge-in-style abort reasons also add `barge_in_detected` alongside
the existing `barge_in.detected` compatibility marker. These are
turn-control markers only, not physical device playback stop proof yet.
When the explicit stock professional route override is enabled, Gateway records
`xiaozhi.professional_route.stock_override` without storing the utterance or
stock listen-mode value.
Professional checking/fallback/result speech records `tts.first_audio`,
`audio.downlink.first_frame`, and `xiaozhi.tts.opus_frame.downlink` when the
Gateway TTS adapter successfully produces and writes OPUS frames; structured
professional JSON alone is not physical audible proof.
Provider text fallback records `fallback.used` and `provider.failover` in the
same Gateway trace when the voice pipeline reports an activated fallback. These
markers are provider-neutral and do not carry prompt, transcript, provider
output, endpoint, model, key, or proxy values.
Local fallback records `fallback.used` and `local_fallback.entered` when the
voice/provider pipeline cannot produce a playable answer after local listening.
Xiaozhi fallback also records `xiaozhi.local_fallback.sent` after the fallback
sentence is emitted. Local fallback increments the generic fallback counter, but
does not increment provider failover because no alternate provider succeeded.
`/v1/traces` now summarizes split latency deltas for
`xiaozhi_listen_to_audio_ingress_ms`, `xiaozhi_opus_decode_ms`,
`asr_first_partial_ms`, `llm_first_content_ms`, `tts_first_audio_ms`,
`audio_downlink_first_frame_ms`, `device_playback_start_ms`, and
`answer_first_audio_total_ms` when the corresponding markers exist.
Paced xiaozhi TTS downlink frames record `xiaozhi.tts.opus_frame.downlink` only
after a binary frame write succeeds through the current turn guard.
Suppressed stale-turn downlink attempts record
`xiaozhi.tts.stale_frame_suppressed` without storing or emitting frame payloads.
These markers prove protocol/session/codec/ingress telemetry only; they are not
ASR, TTS, real-device playback, or PRD latency acceptance evidence.

`xiaozhi-voice-bench` packages those host markers into schema
`a21.xiaozhi_voice_bench.v1`. The report includes a loopback/remote Gateway
label, stock/debug profile name, protocol version, redacted input source
metadata (`synthetic_sine` or `wav_fixture` basename plus Opus frame count),
answer turn receipts, barge-in turn receipts, first-audio p50/p95,
abort-stop p50/p95, execution booleans, redaction booleans, and per-turn
`/v1/traces` summaries for
`xiaozhi_opus_decode_ms`, `asr_first_partial_ms`, `asr_final_ms`,
`llm_first_content_ms`, `tts_first_audio_ms`, `audio_downlink_first_frame_ms`,
`answer_first_audio_total_ms`, and available barge-in/device markers. It
intentionally stores no raw Opus/PCM, base64 payload, transcript, prompt,
provider output, credential value, full URL, proxy value, or full local path.
Its top-level `execution` block may record the redacted `voice_pipeline`
execution mode, selected ASR/LLM/TTS profile names and `A21_` env names, and
host-local ASR/text/TTS stage booleans. `fixture` remains the default when no
host-local pipeline summary is observed. Even when the host loopback p95 values
satisfy the PRD numbers, the report remains candidate evidence and keeps
`provider_executed=false`, `v21_executed=false`, `hardware_executed=false`, and
`prd_accepted=false` until physical StackChan markers are present.

`xiaozhi-professional-bench` packages professional-mode Gateway evidence into
schema `a21.xiaozhi_professional_bench.v1`. In external Gateway mode the report
may carry `source_profile=external_gateway`,
`acceptance_status=external_gateway_ready`, `v21_executed=true`, and a redacted
`v21_query_first_result_ms` only after `/v1/traces` contains V21 query start and
first-result markers for the same trace. It still records no ASR text, prompt,
retrieved evidence body, provider output, full URL, or full local path, and it
still keeps `provider_executed=false`, `hardware_executed=false`, and
`prd_accepted=false`.

AgentTask bridge reports include package-level T1/T2 semantic reports in
`internal/providers` and the host-only `agent-plan` / `agent-io-smoke` CLI
reports. Provider semantic reports use schema
`a21.agent_task.semantic_report.v1`, preserve A21 `trace_id` and `session_id`,
and store event kind/final markers plus redacted text/tool markers. Planner
reports use `a21.agent_plan.v1`; smoke reports use `a21.agent_io_smoke.v1`.
They must not store external-agent text, tool payloads, credentials, full URLs,
local paths, provider env values, or raw agent control payloads. This scaffold
is not a Gateway runtime path and does not execute V21, providers, realtime
voice, firmware, or hardware.

Current Agent I/O planner markers are:

- `agent.plan.start`
- `memory.policy.selected`
- `scenario.selected`
- `agent_io.message.sent`
- `agent.plan.selected`

Reserved AgentTask semantic marker names for future runtime work are:

- `agent_task.started`
- `agent_task.progress`
- `agent_task.text_delta.redacted`
- `agent_task.tool_call.redacted`
- `agent_task.result.redacted`
- `agent_task.error`
- `agent_task.final`

Gateway also exposes `GET /v1/providers/voice/health` for the current voice provider adapter. It returns provider name, health status, configured state, realtime capability, optional active child provider, and detail text. Unavailable providers return HTTP 503 so future real-provider failures can be distinguished from device and firmware failures.

Gateway now exposes the first provider-neutral realtime session boundary:

- `POST /v1/realtime/session`
- `POST /v1/realtime/session/cancel`

These endpoints still run through the A21 `VoiceProvider` interface and are safe with the default mock Gateway provider. They add session/cancel trace markers and provider latency histograms without allowing professional-mode evidence work to disappear into an opaque realtime provider.

The audio WebSocket realtime path now records provider audio uplink and downlink separately. Uplink covers VAD-driven provider session start, speech-frame append, and commit on speech end. Downlink covers provider `VoiceEvent` output streaming back to A21 `control.event` and `audio.playback.chunk` envelopes.

`provider-realtime-fixture --execute --output-dir reports` records the
no-network realtime adapter fixture as
`a21-provider-realtime-fixture-*.json`. `product-readiness` ingests this report
through `--provider-realtime-fixture-report` or `--use-latest-reports` and
emits `provider.realtime_evidence_valid`, `provider.realtime_provider`,
`provider.realtime_family`, `provider.realtime_protocol`,
`provider.realtime_status`, `provider.realtime_executed`,
`provider.realtime_route_eligible`, `provider.realtime_evidence_mode`, and
`provider.realtime_source_report`. These fields are observability evidence
only; offline fixture evidence must not set `real_provider_ready`,
`launch_ready`, or PRD acceptance.

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
- `device.runtime.echo.received`
- `xiaozhi.hello.received`
- `xiaozhi.listen.start`
- `xiaozhi.listen.detect`
- `xiaozhi.listen.stop`
- `xiaozhi.listen.stop.ignored`
- `xiaozhi.opus_frame.received`
- `xiaozhi.opus_frame.decoded`
- `xiaozhi.opus_frame.decode_error`
- `xiaozhi.opus_frame.ignored_not_listening`
- `xiaozhi.opus_no_frames`
- `xiaozhi.opus_decoded_pcm16`
- `xiaozhi.opus_decode_error`
- `xiaozhi.opus_partial_decode_error`
- `xiaozhi.abort.received`
- `xiaozhi.tts.stop`
- `audio.frame.received`
- `audio.ingress.buffered`
- `audio.ingress.invalid`
- `audio.probe.frame.accepted`
- `provider.realtime_audio.physical_armed`
- `provider.realtime_audio.physical_suppressed`
- `vad.speech.start`
- `vad.speech.end`
- `barge_in.detected`
- `playback.stop`
- `provider.cancel`
- `realtime.session.start.received`
- `provider.start_turn.start`
- `provider.start_turn.first_event`
- `provider.start_turn.end`
- `provider.start_turn.error`
- `realtime.session.cancel.received`
- `provider.cancel.start`
- `provider.cancel.end`
- `provider.cancel.error`
- `provider.realtime_session.start`
- `provider.realtime_session.error`
- `provider.audio.append`
- `provider.audio.append.error`
- `provider.audio.commit`
- `provider.audio.commit.error`
- `provider.audio.downlink`
- `provider.audio.first_downlink`
- `fast_companion.turn.received`
- `fast_companion.local_audio.frontend.accepted`
- `provider.text_stream.route.placeholder`
- `fast_companion.voice_pipeline.start`
- `fast_companion.voice_pipeline.completed`
- `fast_companion.voice_pipeline.unavailable`
- `xiaozhi.voice_pipeline.start`
- `xiaozhi.voice_pipeline.completed`
- `xiaozhi.voice_pipeline.unavailable`
- `local_fallback.entered`
- `xiaozhi.local_fallback.sent`
- `asr.first_partial`
- `asr.final`
- `provider.first_byte`
- `provider.first_content`
- `tts.first_audio`
- `audio.downlink.first_frame`
- `device.playback.start`
- `audio.playback.chunk.sent`
- `xiaozhi.tts.opus_frame.downlink`
- `professional.checking_feedback.sent`
- `xiaozhi.professional_asr_empty`
- `xiaozhi.professional_asr_unavailable`
- `xiaozhi.professional_result_suppressed`
- `v21.query.start`
- `v21.query.first_result`
- `v21.query.error`
- `control.listening.sent`
- `control.professional.sent`
- `control.thinking.sent`
- `control.speaking.sent`
- `control.interrupted.sent`
- `control.local_fallback.sent`

Current trace summary fields:

- `event_count`
- `last_offset_ms`
- `audio_frame_to_playback_ms`
- `v21_query_first_result_ms`
- `barge_in_stop_ms`
- `provider_commit_to_first_audio_ms`

`v21-professional-readiness` emits schema `a21.v21_professional_readiness.v1`
as host-only report-contract evidence. It records `checking_ack_ms`,
`checking_ack_within_1200`, `evidence_completed_ms`, and
`evidence_completed_after_ack` to prove the local “checking” acknowledgement is
available before mock evidence completion. The report also carries
`evidence_available`, `cards_available`, `follow_ups_available`,
`adapter_configured`, `adapter_executed=false`, `redaction_ok`, fixed findings,
and `professional_acceptance_status`. It is not real V21 execution, physical
device acceptance, retrieval-quality proof, or PRD completion evidence.

`v21-adapter-smoke --execute` emits schema `a21.v21_adapter_smoke.v1` as real
adapter-boundary execution evidence. Product readiness accepts it only when the
report keeps the professional query contract visible through fixed labels
(`mode=professional`, `latency_profile=fast_first`,
`answer_style=voice_first_with_citations`, `privacy_scope=professional_only`,
`max_first_response_ms=1200`), has positive confidence plus evidence, speech,
card, and follow-up counts, records `redaction_ok=true`, and stores only a
basename `report_path`. It still does not prove physical StackChan audio,
provider TTS, or PRD launch acceptance by itself.

## Voice Waterfall Events

Every voice turn should eventually expose:

- `audio.capture.start`
- `audio.capture.end`
- `audio.frame.received`
- `vad.speech.start`
- `vad.speech.end`
- `asr.first_partial`
- `provider.first_byte`
- `provider.first_content`
- `v21.query.start`
- `v21.query.first_result`
- `tts.first_audio`
- `audio.downlink.first_frame`
- `device.playback.start`
- `barge_in.detected`
- `provider.cancel`
- `playback.stop`
- `fallback.used`
- `local_fallback.entered`

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
- `a21_vad_detector_decisions_total{detector,result}`
- `a21_device_identity_invalid_total`
- `a21_realtime_session_total`
- `a21_realtime_session_cancel_total`
- `a21_realtime_audio_uplink_frames_total`
- `a21_realtime_audio_commit_total`
- `a21_realtime_audio_downlink_events_total`
- `a21_realtime_first_audio_ms_bucket`
- `a21_voice_provider_start_turn_ms_bucket`
- `a21_voice_provider_cancel_ms_bucket`
- `a21_provider_failover_total`
- `a21_fallback_total`
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
- `a21_asr_final_ms_bucket`
- `a21_llm_first_token_ms_bucket`
- `a21_barge_in_total`
- `a21_barge_in_stop_ms_bucket`
- `a21_provider_error_total`
- `a21_proxy_misconfig_total`
- `a21_device_disconnect_total`
- `a21_fallback_total`
- `a21_media_opus_frames_total`
- `a21_media_opus_decode_error_total`
- `a21_media_opus_fallback_total`

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
