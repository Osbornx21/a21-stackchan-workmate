# A21 Server Mainline

Status: active control source.
Date: 2026-06-01.
Branch: `codex/a21-server-mainline`.
Current control checkout observed 2026-06-01: branch
`codex/a21-hardware-window-20260601-xiaozhi-physical`, HEAD
`645d6989273f`, with an uncommitted verified host-control increment.

This file is the narrow control source for the xiaozhi protocol pivot. It is
not a ledger. Do not add commit-by-commit history here. Record only the current
truth needed to keep the mainline honest and shippable.

## Product Target

A21 is a StackChan/CoreS3 desktop workmate. The pivot keeps the PRD v0.4
product requirements and replaces A21-owned firmware/audio transport with a
xiaozhi-firmware-compatible server path:

```text
xiaozhi firmware
  -> xiaozhi WebSocket protocol
  -> A21 server
  -> ASR / LLM / TTS provider spine
  -> xiaozhi device control plus StackChan expression and servo integration
```

A21 still owns provider keys, proxy policy, V21 adapter access, observability,
mode routing, product personality, and acceptance evidence. StackChan remains a
thin sensing and expression device.

## Current Baseline

- Archived previous hardware line tag: `archive/hardware-line-20260601`.
- Server mainline base commit: `37f156701d99f66675efbad6d62bd6e60d4959d2`.
- Pivot ADR: `docs/engineering/adr/0006-xiaozhi-firmware-websocket-protocol.md`.
- Embedded freeze record: `docs/engineering/A21_EMBEDDED_FREEZE_ARCHIVE.md`.
- WS-1 protocol fixture package: `internal/transport/xiaozhi`.
- WS-1 Gateway seam: `/v1/xiaozhi` on the existing A21 Gateway port.
- WS-1 host Opus codec boundary: `internal/audio/opuscodec`, covering mono
  PCM16 60 ms encode/decode at the xiaozhi 16 kHz uplink and 24 kHz downlink
  rates through a pinned pure-Go Opus library.
- WS-1 Gateway Opus telemetry hook: `/v1/xiaozhi` decodes valid uplink Opus
  frames to PCM16 for aggregate frame/sample/duration telemetry only.
- WS-2 precursor: decoded xiaozhi PCM16 now feeds existing `audio.Ingress`,
  VAD markers, and loopback-only recent-audio redaction without provider
  execution.
- P0-1 feature/profile guard: `/v1/xiaozhi` parses `hello.features` and records
  stock versus debug profile state in the sanitized device registry without
  echoing debug extensions into the server hello.
- P0-2 pacing primitive: `internal/audio.AudioRateController` provides 60 ms
  frame pacing, five-frame prebuffer, abort checks, and reset semantics for the
  future xiaozhi TTS Opus downlink.
- P0-2 downlink primitive: Gateway can encode validated 24 kHz or 48 kHz mono
  60 ms PCM TTS chunks into xiaozhi Opus and write them only through the
  current-turn pacer. Local TTS now emits 48 kHz chunks to avoid A21-owned
  24 kHz to 48 kHz interpolation before Opus encode; the full ASR/LLM/TTS
  product route is still not accepted.
- P0-4 fixture pipeline wiring: on xiaozhi `listen/stop`, decoded speech frames
  can flow through the mock ASR/text/TTS pipeline contract and return one paced
  binary Opus downlink frame. This proves plumbing only; real providers,
  physical playback, and PRD first-audio acceptance are still not accepted.
- P0-3 turn foundation: `/v1/xiaozhi` now creates a current turn on
  `listen/start` and cancels it on `abort`, resetting the downlink pacer before
  future frame sends can observe stale turn state.
- WS-5 contract precursor: `internal/transport/xiaozhi` can build and parse
  minimal MCP JSON-RPC `initialize`, `tools/list`, and sanitized `tools/call`
  envelopes, build stock `type=llm` emotion messages from A21 expression
  states, and clamp planned motion `y_angle` to 5-85 degrees. This is not
  Gateway/device-control integration or hardware acceptance.
- WS-6 split trace summary: `/v1/traces` reports xiaozhi ingress/codec,
  ASR, LLM, TTS, first downlink, device playback, and total answer-first-audio
  deltas when the matching trace markers exist.
- WS-6 provider benchmark report contract: `provider-latency-bench` now splits
  host-only placeholder fields for transport ingress, codec decode, ASR
  partial/final, LLM first content, TTS first audio, audio downlink first
  frame, device playback start, and barge-in detected/provider cancel
  done/playback stop done. Each stage carries availability, placeholder state,
  source trace marker, and p50/p95/p99 stats while keeping
  `acceptance_status=not_accepted` and `prd_accepted=false`.
- External Gateway professional bench support: `xiaozhi-professional-bench`
  can drive an already-running `/v1/xiaozhi` Gateway and mark
  `source_profile=external_gateway` only when the same trace proves V21 query
  start and first-result markers. It stores only timing/status/count fields.
- Product readiness can ingest `a21.xiaozhi_professional_bench.v1` through
  `--v21-professional-report` as V21 professional evidence while keeping
  `prd_accepted=false` and physical/product launch gates closed.
- `product-readiness --use-latest-reports` can assemble the latest known
  redacted voice, V21 professional, adapter-smoke, and physical evidence
  reports from `reports/` without hand-copying report paths.
- Current host-only voice evidence:
  `reports/a21-xiaozhi-voice-bench-20260601-191935.646589000.json` reports
  `acceptance_status=candidate_host_only`, repeat 3,
  answer-first-audio p50/p95 369/372 ms, barge-in stop p50/p95 0/0 ms, and
  `prd_accepted=false`.
- Current professional evidence:
  `reports/a21-xiaozhi-professional-bench-20260601-160123.954052000.json`
  reports `acceptance_status=external_gateway_ready`, V21 executed through an
  external Gateway trace, checking feedback within 1200 ms, V21 first result
  178 ms, evidence/card/follow-up counts 5/1/1, and `prd_accepted=false`.
- Current readiness evidence:
  `reports/a21-product-readiness-20260601-191947.json` reports Gateway,
  local Ollama, local sherpa ASR/TTS, and V21 adapter health as ready but
  `launch_ready=false`; the open launch gaps are executed V21 proof in the
  readiness rollup and physical StackChan evidence.
- Host-only `make verify`: passing on 2026-06-01 after the current
  professional/readiness/doctor/Silero increment.
- Host-only `gate --scope host` and `preflight`: passing after port recheck.
- `doctor`: passing with only `firmware_current_artifact_missing` warning.
- Current known warning: no release-ledger-validated A21 firmware artifact
  matches the current server-mainline commits; this is not a server-mainline
  blocker unless a future slice claims firmware release acceptance.
- Current xiaozhi seam limitation: the host path can prove fast local
  ASR/LLM/TTS and professional Gateway/V21 contracts, but the physical
  StackChan chain, audible quality, xiaozhi device-control/StackChan avatar MCP
  integration, and product launch acceptance remain not accepted.

## Acceptance Board

| PRD criterion | Required evidence | Current status |
| --- | --- | --- |
| Presence and workmate experience | user-visible StackChan expression, voice, and office-mode behavior | not accepted |
| First audible companion response | P50 < 900 ms, P95 < 1500 ms on accepted chain | host candidate p50/p95 369/372 ms; physical audible acceptance not accepted |
| Barge-in | playback/speaking stop P95 < 300 ms plus provider cancel/playback stop trace | host candidate p95 0 ms; physical playback stop not accepted |
| Professional mode | within 1200 ms "checking" feedback, final answer has conclusion, evidence, confidence, follow-up | external Gateway/V21 candidate ready; product acceptance still not accepted |
| Provider hot plug | new OpenAI-compatible text provider through profile/env/smoke without Gateway business edits | partially scaffolded, not accepted |
| Safety | provider keys only in env/secret manager; no secrets in Git, reports, logs, firmware, traces | host gate passing, ongoing |

All acceptance statuses stay `not accepted` until machine-readable reports prove
the PRD threshold. Candidate or host-only fixture evidence must not be promoted
as physical/product acceptance.

## Workstream Rules

- Single integration branch: `codex/a21-server-mainline`.
- Maximum concurrent write agents: 3.
- Each write agent gets one disjoint workstream and one declared write set.
- Red branches do not merge.
- Provider/V21 execution, Gateway runtime startup, and hardware writes require
  explicit control-window declarations before use.
- No raw firmware upload or copied flash command is allowed.
- No new ledger, phase-document swarm, or god-file rebuild is allowed.

## Workstreams

1. WS-1 xiaozhi protocol transport adapter.
   - Primary write set: `internal/transport` plus minimal Gateway seam.
   - Acceptance: hello/listen/abort JSON fixtures, binary OPUS frame shape, and
     trace/session/device propagation. No provider or hardware execution.

2. WS-2 inference streaming pipeline.
   - Primary write set: `internal/providers`, `internal/audio`.
   - Acceptance: ASR/LLM/TTS timing report shape with first-content and
     first-audio markers. Provider execution requires a separate T4 window.

3. WS-3 session, mode, personality, and barge-in.
   - Primary write set: Gateway session state and product copy.
   - Acceptance: professional is excluded from opaque realtime, xiaozhi abort
     maps to provider cancel/playback stop, and PRD mode transitions are tested.

4. WS-4 V21 professional adapter lane.
   - Primary write set: `internal/v21adapter` and app adapter boundary.
   - Acceptance: adapter contract, timeout fallback, redacted reports, and no
     V21 internals in A21.

5. WS-5 StackChan expression and xiaozhi device-control integration.
   - Primary write set: future avatar/MCP adapter surface.
   - Acceptance: expression/servo/device-control evidence without regressing
     the original StackChan experience.

6. WS-6 observability and honest gates.
   - Primary write set: metrics, report schemas, and gates.
   - Acceptance: trace IDs and PRD latency markers exist before optimization.

## Next Action

1. Keep the current verified increment intact; do not split it across another
   parallel write lane until it is intentionally checkpointed.
2. Run the next no-hardware control slice by feeding the external professional
   report and current xiaozhi voice bench report into `product-readiness`, with
   Gateway/V21 runtime startup only inside an explicit local execution window.
3. In parallel only as read-only or host-only work, run the audio-quality A/B
   plan from the xiaozhi comparison thread: clipping/level scan, sample-rate
   normalization, Opus frame/bitrate comparison, and pacer/prebuffer review.
4. Keep physical StackChan proof closed until the device is intentionally
   brought back from official xiaozhi firmware and the user opens a hardware
   window. Mac must not play trigger audio; the user speaks to StackChan.
