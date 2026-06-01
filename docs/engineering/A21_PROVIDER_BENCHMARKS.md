# A21 Provider Benchmark Contract

## Purpose

A21 should reuse mature public benchmark methods instead of inventing a new gate for every ASR, TTS, LLM, or speech-to-speech provider. This document is the mainline contract for importing those methods without letting an external project become A21's architecture.

External benchmark projects are reference material only. They may contribute datasets, measurement definitions, provider adapter ideas, and report-shape examples. A21 promotion decisions still require A21-owned traces, redacted reports, network/proxy metadata, and StackChan evidence when hardware behavior is claimed.

## Reference Inputs

Current reference classes:

| Reference class | Example projects | What A21 may reuse | What A21 must not assume |
| --- | --- | --- | --- |
| STT/ASR latency and accuracy | `pipecat-ai/stt-benchmark`, Coval STT benchmarks | TTFS-style stop-of-speech to final transcript timing, semantic WER ideas, provider comparison datasets | That clean benchmark audio predicts office noise, StackChan microphone quality, or A21 VAD behavior |
| TTS first-audio latency | Coval TTS benchmarks, Picovoice text-to-speech benchmark | TTFA/FTTS definitions, p50/p95/p99 reporting, streaming TTS measurement shape | That API first byte equals playable audio, or that TTS-only timing proves total turn latency |
| LLM streaming latency | Picovoice voice-assistant simulation, A21 `provider-smoke --stream` | TTFT, first content, total streaming duration, fallback markers | That text-only streaming proves voice turn quality or interruption behavior |
| Speech-to-speech providers | Speko-style S2S leaderboards, Pipecat S2S adapters, provider realtime docs | provider capability vocabulary, realtime session boundaries, first-audio and cancellation probes | That opaque S2S is acceptable for professional mode or replaces A21 provider-neutral events |
| Voice-assistant quality | VoiceBench-style speech assistant evaluation | spoken instruction coverage, quality/safety prompt sets, audio capability comparison | That quality leaderboards prove mainland latency, cost, or StackChan experience |

These names are examples, not dependencies. Adding one as a production dependency still needs the normal A21 dependency check and rationale.

## Canonical Metrics

A21 benchmark reports must preserve the following canonical names when the data is available:

| Metric | Meaning |
| --- | --- |
| `speech_end_to_final_asr_ms` | user speech end to final ASR transcript |
| `speech_end_to_first_llm_token_ms` | user speech end to first LLM token/content |
| `llm_request_to_first_token_ms` | text provider request to first token/content |
| `first_llm_token_to_first_tts_audio_ms` | first LLM token/content to first playable TTS audio |
| `tts_request_to_first_audio_ms` | TTS request to first playable audio |
| `provider_commit_to_first_audio_ms` | realtime provider commit/create-response to first audio-bearing downlink |
| `gateway_downlink_first_frame_ms` | Gateway first available playback chunk to device-facing downlink |
| `device_downlink_first_frame_ms` | physical StackChan receipt of first playback frame |
| `device_playback_start_ms` | physical StackChan speaker playback start |
| `speech_end_to_first_audible_response_ms` | user speech end to physical or instrumented first audible response |
| `barge_in_stop_ms` | interrupting user speech to playback stopped/cancelled |

Reports must use p50, p95, and p99 when sample counts are high enough. Averages may appear only as supporting data.

## Required Report Shape

Every A21 provider benchmark report must include:

- `trace_id`, `session_id`, and `device_id` or an explicit `device_id=none_host_fixture` style marker;
- provider family and provider profile;
- network profile, DNS/network fingerprint, and redacted proxy policy metadata;
- explicit execution mode: `mock`, `fixture`, `host_loopback`, `physical_stackchan`, or `external_lab`;
- sample count, fixture identity, and audio format, without storing raw PCM, base64 audio, transcript text, prompt text, model output, provider reasoning, proxy values, API keys, full URLs, or full local model paths;
- stage timings using the canonical metric names above;
- p50/p95/p99 series for repeated runs;
- failure/fallback counts and structured redacted findings;
- a `promotion_gate` field with one of `not_production`, `candidate`, or `accepted`.

Provider-specific event names must stay inside provider adapters. Device and Gateway reports must use A21 event names only.

## Promotion Rules

A provider or provider combination can move toward the fast companion lane only after the same report shape can compare at least:

- a local or mock baseline;
- the candidate ASR path;
- the candidate LLM/text-stream path;
- the candidate TTS or realtime S2S path;
- interruption/cancel behavior;
- failure behavior when the provider is unavailable or slow;
- mainland-network behavior and explicit proxy mode;
- cost and key-handling boundary;
- subjective voice quality notes kept outside secret-bearing logs.

Physical StackChan promotion additionally requires device-side evidence for downlink receipt, playback start, and barge-in stop. Host fixtures and browser simulators can be `candidate` evidence, but they must not be marked as physical first-audio acceptance.

`physical-stackchan-evidence` is the current fixture-only report contract for
that physical promotion layer. It can package already-collected device
downlink, playback-start, first-audible-response, barge-in-stop, microphone,
and operator/instrument observations into a redacted
`a21.physical_stackchan_evidence.v1` report, but it does not execute hardware
or accept the PRD by itself.

Professional mode must not be promoted through an opaque realtime S2S benchmark alone. It needs the V21 adapter evidence path with visible query, evidence, confidence, and fallback state.

## Mainline CLI Direction

The current `latency-bench --mock`, `audio-front-end-eval`, `provider-smoke --stream`, `local-voice-loopback`, and `stackchan-fast-companion-turn` reports are partial pieces of this contract. Future real-provider work should extend this family instead of creating separate ad hoc gates.

`provider-smoke --execute --stream --repeat 3 --output-dir reports` is the
current real text-provider evidence source for product readiness. Its
`a21.provider_smoke.v1` report can reduce only the provider gap, and only when
`product-readiness --provider-smoke-report <report.json>` sees that the report
matches the currently selected configured provider, is non-mock and
route-eligible, contains three or more successful streaming attempts with
first-byte, first-content, and total-duration p50/p95/p99 timings, has no
fallback marker, and passes the same
no-prompt/no-transcript/no-output/no-reasoning/no-secret/no-full-URL redaction
checks. It does not prove ASR, TTS, V21, physical playback, barge-in stop, or
PRD launch acceptance by itself.

## 5080lab Selected Provider Execution Package

Run selected-provider closure on `5080lab` or another approved clean mainland
lab host only. Do not run the executed provider smoke on the proxy-affected Mac.
The Mac may run dry-run readiness checks and may ingest returned redacted
reports with `product-readiness`.

Use this exact sequence on `5080lab`, replacing only the provider/env file with
the selected route-eligible profile:

```bash
cd "<A21 repo checkout>"
git status --short --branch
git rev-parse --short HEAD

mkdir -p .a21-run/5080lab reports/5080lab-provider
chmod 700 .a21-run/5080lab
set -a
. ./.a21-run/5080lab/provider.env
set +a

unset HTTP_PROXY HTTPS_PROXY ALL_PROXY
export NO_PROXY="localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16"

go run ./cmd/a21 doctor --output-dir reports/5080lab-provider
go run ./cmd/a21 provider-smoke --provider "$A21_PROVIDER_PRIMARY" --stream --repeat 3 --output-dir reports/5080lab-provider
go run ./cmd/a21 provider-smoke --provider "$A21_PROVIDER_PRIMARY" --execute --stream --repeat 10 --output-dir reports/5080lab-provider

LATEST_PROVIDER_REPORT="$(ls -t reports/5080lab-provider/a21-provider-smoke-*.json | head -n 1)"
go run ./cmd/a21 product-readiness --provider-smoke-report "$LATEST_PROVIDER_REPORT" --output-dir reports/5080lab-provider
go run ./cmd/a21 server-side-readiness-bundle --provider-smoke-report "$LATEST_PROVIDER_REPORT" --output-dir reports/5080lab-provider
tar -czf "reports/a21-5080lab-provider-evidence-$(date +%Y%m%d-%H%M%S).tgz" -C reports/5080lab-provider .
```

`.a21-run/5080lab/provider.env` must stay local to the lab host and should
contain only `A21_` variables such as `A21_PROVIDER_PRIMARY`, the selected
provider key env, model env when required, optional base-url env, and optional
`A21_PROVIDER_PROFILES_PATH` for valid `a21_`-namespaced loaded profiles. Built
in candidates that are not route-eligible must be loaded as explicit
route-eligible `a21_` profiles before they can close this package.

The dry-run `provider-smoke` must report `configured=true`,
`route_eligible=true`, `stream=true`, `executed=false`, and safe env names. The
executed report must report `status=passed`, `executed=true`, `stream=true`,
`repeat>=3`, successful attempts, no activated fallback, and these non-zero
summary fields: `first_byte_p50_ms`, `first_byte_p95_ms`,
`first_byte_p99_ms`, `first_content_p50_ms`, `first_content_p95_ms`,
`first_content_p99_ms`, `total_duration_p50_ms`, `total_duration_p95_ms`, and
`total_duration_p99_ms`.

Return these files to A21:

- `reports/5080lab-provider/a21-doctor-*.json`
- `reports/5080lab-provider/a21-provider-smoke-*.json` for both dry-run and executed runs
- `reports/5080lab-provider/a21-product-readiness-*.json`
- `reports/5080lab-provider/a21-server-side-readiness-bundle-*.json`
- `reports/a21-5080lab-provider-evidence-*.tgz`

Reject the package if any report stores API key values, model values, prompt
text, transcript text, provider output, provider reasoning, proxy values, full
provider URLs, URL credentials, local paths, or non-A21 provider profile names.
`endpoint_host`, `api_key_env`, `model_env`, and `base_url_env` are allowed
because they are redacted host/env labels. Provider closure still does not
prove ASR, TTS, V21, physical StackChan playback, or barge-in acceptance.

`local-tts-smoke`, `local-voice-loopback`, and Gateway voice-pipeline summaries
also carry an aggregate `audio_quality` block for generated PCM16 TTS audio.
This is a host-side guardrail for symptoms such as clipping, low headroom,
near-silence, DC offset, and suspicious output format, not physical speaker
quality acceptance. Reports store only aggregate metrics and fixed finding
codes; audio payloads, prompts, transcripts, provider output, full URLs, proxy
values, model secrets, and local paths stay out of the quality block.

`xiaozhi-voice-bench` is the Xiaozhi-protocol host-loopback member of the same
family. It is allowed to contact an already-running local Gateway and exercise
`/v1/xiaozhi` with either the default synthetic Opus uplink or `--input-wav`
speech fixtures encoded into 60 ms Opus frames, but it does not start Gateway,
providers, V21, firmware, or hardware. Reports store only the fixture basename
and Opus frame count, never local paths or audio payloads. Its successful state
is `candidate_host_only`, not `accepted`, and it exists to compare answer
first-audio and barge-in stop timings before physical StackChan promotion.

The first `provider-latency-bench` scaffold now exists as a mock/fixture plus
host-loopback candidate-chain report shape:

```bash
go run ./cmd/a21 provider-latency-bench \
  --provider <profile> \
  --mode host_loopback \
  --fixture <a21-redacted-host-loopback-report.json> \
  --iterations 30 \
  --output-dir reports
```

This scaffold does not execute real providers, V21, Gateway runtime services, or
physical StackChan paths. It emits A21-owned `trace_id`, `session_id`,
`device_id=none_host_fixture`, provider profile/family labels, redacted
network/proxy metadata, stage waterfall placeholders, p50/p95/p99 summaries,
fallback/failure counts, `promotion_gate=not_production`,
`acceptance_status=not_accepted`, and `prd_accepted=false`. The v2 report shape
also includes machine-readable `metric_terms`, `canonical_metrics`, and
`stage_availability` blocks so TTFS, TTFT, FTTS, and TTFA vocabulary can be
compared against A21 canonical fields without promoting the numbers as real
latency. `stage_availability` is the per-segment contract: each entry carries
`available`, `placeholder`, `source_trace_marker`, sample count, and p50/p95/p99
redacted stats. The canonical block covers `transport_ingress_ms`,
`codec_decode_ms`, `asr_first_partial_ms`, `asr_final_ms`,
`llm_first_content_ms`, `provider_first_byte_ms`,
`provider_first_content_ms`, `tts_first_audio_ms`,
`downlink_first_frame_ms` as the legacy compatibility alias,
`audio_downlink_first_frame_ms`, `device_playback_start_ms`,
`barge_in_detected_ms`, `barge_in_stop_ms`, `provider_cancel_ms`,
`provider_cancel_done_ms`, `playback_stop_ms`, `playback_stop_done_ms`,
`speech_end_to_final_asr_ms`, `speech_end_to_first_llm_token_ms`,
`llm_request_to_first_token_ms`, `first_llm_token_to_first_tts_audio_ms`,
`tts_request_to_first_audio_ms`, `provider_commit_to_first_audio_ms`,
`gateway_downlink_first_frame_ms`, `device_downlink_first_frame_ms`,
`speech_end_to_first_audible_response_ms`, `answer_first_audio_ms`,
`answer_first_audio_p95_ms`, and `barge_in_stop_p95_ms`. Mock and fixture
placeholder stages remain `available=false`, `placeholder=true`, and carry a
fixed placeholder reason. When `--mode host_loopback --fixture <report.json>`
ingests a redacted `local-voice-loopback`, `xiaozhi-voice-bench`, or
`a21.virtual_xiaozhi_harness.v1` aggregate report, available host-only
ASR/provider/TTS/downlink/barge-in fields may be marked `available=true`,
`placeholder=false`, and summarized as `acceptance_status=candidate_host_only`
if enough samples satisfy the PRD host-only latency thresholds. Virtual Xiaozhi
harness reports contribute `first_audio_samples_ms` / `first_audio_p95_ms` to
`answer_first_audio_*` metrics and `abort_stop_samples_ms` /
`abort_stop_p95_ms` to `barge_in_stop_*` metrics. Physical-only fields such as
`device_playback_start_ms` and `speech_end_to_first_audible_response_ms` remain
unavailable unless the report contains physical StackChan evidence. When an
ingested `xiaozhi-voice-bench` report contains a redacted `execution` summary
from Gateway `voice_pipeline` messages, `provider-latency-bench` preserves only
the safe execution semantics: `voice_pipeline_execution_mode`,
ASR/LLM/TTS profile names and `A21_` env names, plus host-local ASR/text/TTS
stage booleans. These fields are host-only evidence and do not flip
`provider_executed`, `v21_executed`, `hardware_executed`, `promotion_gate`, or
`prd_accepted`.

`xiaozhi-voice-bench` reports now also expose
`execution.host_product_chain_ready`. The flag is host-only: it means one
redacted Gateway `voice_pipeline` chain observed host-local ASR, text stream,
and TTS stages with non-mock profiles while staying below physical PRD
acceptance. `product-readiness` derives the same value from older reports that
only contain the stage booleans, but honors the explicit flag when it is
present. Only this stricter host product-chain evidence may clear canonical
`missing_real_evidence=continuous_voice_pipeline`; fixture/demo loopbacks can
remain `candidate_host_only` while still leaving that canonical gap open.
When an ingested host-loopback report contains `audio_quality`,
`local_ack_audio_quality`, or `tts_audio_quality`, warning/failed quality states
and fixed PCM guard findings such as clipping, low headroom, near-silence, DC
offset, invalid payload/format, unsupported codec, unavailable quality, or
format mismatch produce structured redacted findings and block
`candidate_host_only`. The benchmark still preserves the latency waterfall so
the team can see that timing passed while audio quality failed, but it must not
store peak values, codec internals, raw finding lists, paths, prompts,
transcripts, provider output, or audio payloads in the promoted report.
When `--fixture` points at a redacted JSON sidecar, the report may include
`schema_version=a21.provider_latency_fixture.v1`, fixture identity, audio
format, sample rate, channel count, duration, sample count, window length, and
window count. The report stores only the fixture basename. Invalid or unsafe
sidecars produce structured redacted findings instead of panics or raw errors.
Reports must not store prompt text, transcript text, provider output, provider
reasoning, raw PCM, base64 audio, key values, full provider URLs, proxy URLs, or
full local fixture paths.

The host-local Xiaozhi voice pipeline text lane keeps spoken answers concise by
default. `A21_VOICE_TEXT_MAX_TOKENS` controls the provider-neutral text budget
used by voice pipeline OpenAI-compatible and Ollama text-stream adapters. The
default is 24 tokens; values are sanitized into the 8..96 range. This knob does
not change standalone provider smoke defaults, does not relax PRD latency gates,
and does not authorize transcript or provider-output capture in reports.

The host-local product path now uses the provider catalog as its text-stream
gate instead of hard-coding one cloud vendor. `A21_PROVIDER_PROFILES_PATH` can
add `a21_`-namespaced OpenAI-compatible profiles; only profiles that pass the
catalog validator and explicitly set `route_eligible=true` may be selected by
`local-voice-loopback`, `stackchan-fast-companion-turn`, or the host-local
voice pipeline. Running with `--execute-text-provider` is still explicit
execution authorization, and true mainland latency evidence must be gathered on
`5080lab` or another approved clean mainland lab host. Httptest-backed provider
tests on a proxy-affected Mac prove protocol/redaction behavior only.

The same host-local text path supports an explicit secondary provider through
`A21_TEXT_STREAM_FALLBACK_PROFILE` or the loopback-only
`--fallback-text-provider` / `A21_LOCAL_TEXT_FALLBACK_PROVIDER`. Fallback is
allowed only to another configured, route-eligible text-stream profile. Reports
record `provider_fallback_used`, the fallback provider name, and a coarse
reason such as `primary_failed`, but still never record prompt text, transcript
text, provider output, provider reasoning, API keys, model values, full URLs, or
proxy values. This closes the PRD server-side router contract for
`primary fail -> fallback success`; it does not by itself prove mainland p95/p99
or physical StackChan playback.

This v2 hardening is a report-contract and metric-shape change only. It does
not authorize provider execute, V21 execute, Gateway runtime startup, binary
Opus transport, AEC adapter implementation, WebRTC/ESP-SR native adapters, or
hardware acceptance. Until a later T4/T6/T8 window adds real measurements,
provider comparisons must cite this scaffold only as report-shape evidence and
must list unmeasured real ASR, provider, TTS, downlink, physical playback, and
barge-in stages explicitly.

## Non-Goals

- Do not vendor a benchmark framework just to make a leaderboard inside A21.
- Do not add public benchmark output to release gates unless the report was rerun through A21's own redaction, network, and trace contract.
- Do not treat provider marketing numbers as evidence.
- Do not collapse ASR, LLM, TTS, and device playback into one opaque number unless the per-stage waterfall is also preserved.
