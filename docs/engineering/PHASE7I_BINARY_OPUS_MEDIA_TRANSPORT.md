# A21 Phase 7I Binary Opus Media Transport Planning Contract

## Purpose

Phase 7I defines the PRD-reviewable contract for moving A21 device media from
JSON/base64 `pcm_s16le` fixtures toward an A21-owned binary Opus media profile.
It is planning and contract work only.

This phase does not implement an Opus encoder, decoder, Gateway runtime path,
firmware parser, hardware effect, provider path, V21 path, native codec
dependency, or production transport. Current runtime behavior remains
JSON/base64 PCM for safe Gateway, simulator, fixture, and diagnostic evidence.

## Goals

- Reserve an A21 binary media profile that keeps StackChan provider-neutral.
- Preserve `trace_id`, `session_id`, and `device_id` across media frames.
- Define a frame/envelope draft before native codec or firmware work begins.
- Keep redaction, proxy, direct-connect, and report safety reviewable.
- Name the future gates needed before binary Opus can affect latency claims.
- Keep PCM fixtures available for deterministic protocol and diagnostic tests.

## Non-Goals

- No Gateway runtime or service startup.
- No provider execute, V21 execute, AgentTask runtime, or real device path.
- No firmware, NVS, flash, raw upload, serial write, or `/v1/devices/control`.
- No production dependency, native Opus dependency, or WebRTC dependency.
- No durable payload report containing raw audio, Opus bytes, PCM bytes,
  base64 audio, prompt text, transcript text, provider output, reasoning,
  credentials, full URLs, proxy URLs, or full local paths.
- No latency, quality, AEC, full-duplex, or StackChan hardware acceptance claim.

## Current Boundary

Current A21 audio evidence is still JSON/base64 `pcm_s16le`:

- `audio.frame` uplink and `audio.playback.chunk` downlink use the A21 JSON
  envelope and base64 PCM payload.
- `internal/protocol` reserves `opus` as an `AudioCodec`, but current validation
  accepts only `pcm_s16le`.
- `audio-front-end-plan` and `audio-front-end-eval` are host-only report and
  candidate evidence contracts. They do not approve binary Opus media.
- `provider-latency-bench` is a mock/fixture/host-loopback report shape. It
  does not prove real provider first-audio, downlink, playback, or Opus impact.
- StackChan microphone, speaker, half-duplex, and hardware acceptance gates
  remain separate physical windows. This contract does not replace them.

## Draft Media Profile

Profile identity:

- `schema_version`: `a21.media.opus_plan.v1`
- `media_profile`: `a21_binary_opus_v1`
- `codec`: `opus`
- `sample_rate_hz`: 16000 or 48000, selected by a future adapter spike
- `channels`: 1 for the first StackChan path
- `frame_duration_ms`: 20, 40, or 60; promotion requires measured evidence
- `direction`: `uplink`, `downlink`, or `loopback_fixture`
- `payload_format`: binary Opus payload bytes, never JSON/base64 in the final
  media lane

Control and identity remain A21-owned. A future connection must negotiate or
declare the media profile with ordinary A21 JSON control metadata before any
binary media frame is accepted.

## Frame And Envelope Draft

The draft separates control metadata from binary media bytes:

```text
a21 media connection
  -> JSON control envelope: profile, stream_id, trace_id, session_id, device_id
  -> repeated binary media frames
  -> JSON control events for state, interruption, fallback, and errors
```

Planned per-frame metadata:

| Field | Purpose |
| --- | --- |
| `magic` | A21 binary media frame marker, exact bytes TBD by fixture spike |
| `version` | binary media frame version |
| `header_len` | lets future fixtures add fields without re-parsing payload |
| `flags` | key frame, end of segment, discontinuity, fallback, reserved bits |
| `direction` | uplink or downlink |
| `seq` | monotonically increasing frame sequence within the stream |
| `stream_id` | playback or capture stream identity |
| `trace_id_ref` | compact reference to negotiated `trace_id` |
| `session_id_ref` | compact reference to negotiated `session_id` |
| `device_id_ref` | compact reference to negotiated `device_id` |
| `capture_started_at_ms` | sender capture timestamp when available |
| `duration_ms` | media duration represented by the Opus frame |
| `payload_len` | byte length of the Opus payload |
| `payload` | Opus bytes, never persisted in durable reports |

The compatibility fixture gate must prove byte order, header length, sequence
rules, error handling, and trace/session/device references before Gateway or
firmware runtime work can parse this format.

## Trace And Metrics Contract

Future runtime spans must preserve:

- `trace_id`
- `session_id`
- `device_id`
- `stream_id`
- `media_profile`
- `codec`
- `frame_duration_ms`
- `direction`
- `network_profile`
- `proxy_profile`
- `fallback_used`

Planned trace markers:

- `media.opus.profile.negotiated`
- `media.opus.uplink.frame.received`
- `media.opus.downlink.frame.sent`
- `media.opus.decode.error`
- `media.opus.fallback_to_pcm`
- `media.opus.jitter_buffer.underflow`
- `media.opus.jitter_buffer.recovered`

Planned metrics:

- `a21_media_opus_frames_total{direction}`
- `a21_media_opus_bytes_total{direction}`
- `a21_media_opus_decode_error_total{direction}`
- `a21_media_opus_fallback_total{reason}`
- `a21_media_opus_frame_duration_ms_bucket{direction}`
- `a21_media_opus_jitter_ms_bucket{direction}`
- `a21_media_opus_jitter_buffer_depth{direction}`

These are reserved names. They are not current Prometheus metrics and must not
be documented as production signals until runtime implementation exists.

## Redaction Contract

Planning, fixture, and future runtime reports may store:

- schema/profile names
- codec label
- sample rate
- channel count
- frame duration
- frame counts
- aggregate byte counts
- sequence gap counts
- p50/p95/p99 timing summaries
- redacted network/proxy metadata
- basename-only fixture/report paths

Reports must not store raw PCM, base64 audio, Opus payload bytes, prompts,
transcripts, provider output, reasoning, credentials, full URLs, proxy URLs,
proxy host/port/user/password, full local paths, or provider model values.

## Direct-Connect And Proxy Contract

StackChan, localhost, LAN, `.local`, and V21 adapter traffic must remain direct
and must not silently inherit global proxies. Provider egress remains explicit
through A21 provider network policy.

A future binary media transport must therefore prove:

- local/LAN media sockets use direct dials or equivalent no-ambient-proxy paths;
- provider egress proxy settings are not reused for device media;
- redacted reports record only proxy mode labels and env variable names;
- no report prints a provider endpoint, proxy endpoint, LAN full URL, or local
  filesystem path.

## Fallback Contract

PCM remains the required fallback and fixture lane until binary Opus passes all
gates. A future implementation must fail closed:

- invalid profile negotiation rejects binary media and keeps current PCM path;
- unsupported frame duration falls back to PCM or returns a redacted error;
- sequence gaps and decode failures emit trace markers and do not claim audio
  success;
- fallback reports store reason codes and aggregate counts only;
- fallback must preserve A21 control events, interruption, and honest user
  failure behavior.

## Future Acceptance Gates

Binary Opus media cannot be promoted until a later authorized slice supplies:

- Opus encoder/decoder adapter spike with mature library review.
- Wire-format compatibility fixture.
- Gateway loopback fixture.
- Provider first-audio waterfall impact report.
- Device playback receipt with trace/session/device continuity.
- CPU and memory profile on target Gateway and StackChan paths.
- LAN jitter and fallback report.
- Hardware window acceptance for microphone, downlink, speaker playback,
  interruption, and audible product quality.

Each gate must state whether it is host-only, Gateway loopback, provider
execute, V21 execute, or physical hardware evidence. Provider/V21 execution and
hardware windows require separate authorization and must not be inferred from
this planning contract.

## Forbidden Actions For This Phase

- provider `--execute`
- real provider calls
- V21 execute
- Gateway runtime or service startup
- durable payload reports with raw audio or media bytes
- firmware, NVS, flash, raw upload, or serial writes
- real `/v1/devices/control`
- physical device path
- production dependency
- native Opus or WebRTC dependency
- secrets, prompts, transcripts, provider output, reasoning, raw audio, full
  URL, proxy URL, or full local path leakage

## Handoff Shape

This slice intentionally stays docs-only. A plan-only CLI/report shape would be
safe only after the frame fixture names and machine-readable report fields are
reviewed; adding one now would create extra test surface without improving the
PRD decision. The next implementation slice can add a plan CLI under TDD if
the control tower wants machine-readable evidence before native adapter spikes.
