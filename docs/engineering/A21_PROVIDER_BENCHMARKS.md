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

Professional mode must not be promoted through an opaque realtime S2S benchmark alone. It needs the V21 adapter evidence path with visible query, evidence, confidence, and fallback state.

## Mainline CLI Direction

The current `latency-bench --mock`, `audio-front-end-eval`, `provider-smoke --stream`, `local-voice-loopback`, and `stackchan-fast-companion-turn` reports are partial pieces of this contract. Future real-provider work should extend this family instead of creating separate ad hoc gates.

`xiaozhi-voice-bench` is the Xiaozhi-protocol host-loopback member of the same
family. It is allowed to contact an already-running local Gateway and exercise
`/v1/xiaozhi` with either the default synthetic Opus uplink or `--input-wav`
speech fixtures encoded into 60 ms Opus frames, but it does not start Gateway,
providers, V21, firmware, or hardware. Reports store only the fixture basename
and Opus frame count, never local paths or audio payloads. Its successful state
is `candidate_host_only`, not `accepted`, and it exists to compare answer
first-audio and barge-in stop timings before physical StackChan promotion.

The first `provider-latency-bench` scaffold now exists as a mock/fixture-only
candidate-chain report shape:

```bash
go run ./cmd/a21 provider-latency-bench \
  --provider <profile> \
  --fixture <a21-redacted-audio-fixture> \
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
`gateway_downlink_first_frame_ms`, `device_downlink_first_frame_ms`, and
`speech_end_to_first_audible_response_ms`. Every current stage is marked
`available=false`, `placeholder=true`, and carries a fixed placeholder reason.
When `--fixture` points at a redacted JSON sidecar, the report may include
`schema_version=a21.provider_latency_fixture.v1`, fixture identity, audio
format, sample rate, channel count, duration, sample count, window length, and
window count. The report stores only the fixture basename. Invalid or unsafe
sidecars produce structured redacted findings instead of panics or raw errors.
Reports must not store prompt text, transcript text, provider output, provider
reasoning, raw PCM, base64 audio, key values, full provider URLs, proxy URLs, or
full local fixture paths.

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
