# A21 Server Mainline

Status: active control source.
Date: 2026-06-01.
Branch: `codex/a21-server-mainline`.

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
- Host-only `make verify`: passing after the WS-1 protocol fixture and Gateway
  seam.
- Host-only `gate --scope host`: passing with only
  `firmware_current_artifact_missing` warning.
- Current known warning: no release-ledger-validated A21 firmware artifact
  matches the current server-mainline commits; this is not a server-mainline
  blocker unless a future slice claims firmware release acceptance.
- Current xiaozhi seam limitation: Gateway can count raw Opus frames and decode
  valid uplink frames to PCM telemetry, but ASR, provider streaming, TTS encode,
  binary downlink, and real device proof remain not accepted.

## Acceptance Board

| PRD criterion | Required evidence | Current status |
| --- | --- | --- |
| Presence and workmate experience | user-visible StackChan expression, voice, and office-mode behavior | not accepted |
| First audible companion response | P50 < 900 ms, P95 < 1500 ms on accepted chain | not accepted |
| Barge-in | playback/speaking stop P95 < 300 ms plus provider cancel/playback stop trace | not accepted |
| Professional mode | within 1200 ms "checking" feedback, final answer has conclusion, evidence, confidence, follow-up | not accepted |
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

Continue WS-1 from the current fixture package and Gateway seam into Opus
decode/encode and real xiaozhi-device handshake proof only under an explicit
hardware/provider execution window. Until that window exists, keep work in
host-only tests and report-contract slices.
