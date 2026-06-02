# T-XIAOZHI-REALTIME-VOICE-PARITY-001 Plan

Date: 2026-06-03

Transition:
`T-XIAOZHI-REALTIME-VOICE-PARITY-001`

## Background and Problem Definition

The product target is not a normal request/response voice bot. The target is a
Xiaozhi-style realtime voice device:

`mic/i2s -> local wake/VAD -> Opus frame queue -> long realtime transport ->
streaming ASR -> streaming LLM -> streaming TTS -> Opus frame playback`

The user review correctly calls out the core risk: A21 can look
Xiaozhi-compatible at the WebSocket/Opus boundary while still behaving like a
turn-buffered system behind the Gateway. If A21 waits for full speech capture,
full ASR, full LLM, or full TTS before downlink playback, the perceived latency
and interaction feel will not match Xiaozhi even if the protocol envelope is
stock.

## Current System State

- Product device lane is the official StackChan Xiaozhi-compatible firmware,
  not the bare `xiaozhi.bin` incident lane.
- Device-side transport is stock-shaped `/v1/xiaozhi`: JSON control and binary
  Opus share one WebSocket connection.
- Device hello advertises Opus mono 16 kHz uplink with 60 ms frames; Gateway
  sends 24 kHz mono Opus downlink.
- Device-side audio currently relies on the official CoreS3/AudioCodec path in
  the official-compatible firmware.
- `xiaozhi-voice-bench` is host-loopback evidence. It opens its own virtual
  WebSocket client and may send synthetic/fixture Opus, so it must not be used
  as physical StackChan acceptance.
- `xiaozhi-physical-evidence` already records physical Gateway/downlink
  evidence, but it is not a strict realtime-parity red/green gate.
- Wake remains unaccepted physically. The system must not claim local wake is
  product-ready until operator proof exists.
- Gateway no-speech cooldown has suppressed the immediate
  `placeholder_no_asr_tts -> listen.start` loop once in live trace evidence.

## Target State

Create a repo-carried realtime parity gate that can tell a future model whether
the live path is:

- `stock_opus_transport_only`: stock-looking transport but not realtime enough;
- `turn_buffered_xiaozhi_candidate`: real physical Opus ingress/downlink with
  ASR/LLM/TTS after speech end;
- `xiaozhi_realtime_candidate`: real physical Opus ingress with streaming ASR,
  streaming LLM, streaming TTS, and first downlink before full answer
  completion;
- `physical_product_accepted`: only after operator proof covers local wake,
  audible playback, interruption, no setup trap, and stable idle recovery.

This transition should produce redacted evidence from a real `/v1/xiaozhi`
trace and explicitly distinguish physical transport success from full Xiaozhi
realtime parity.

## Non-Goals

- Do not use `/v1/xiaozhi/say`, `stackchan-fast-companion-turn`, old
  `/v1/devices/control`, or a synthetic host loopback as product acceptance.
- Do not flash bare `xiaozhi.bin`.
- Do not remove the current working audio gain/source hotfixes.
- Do not store raw transcripts, prompts, provider outputs, credentials, raw
  audio payloads, or full local filesystem paths in reports.
- Do not claim wake-word product readiness without physical proof.
- Do not rewrite the whole Gateway/provider stack in one step during the
  hardware window.

## Impact Range

- `internal/app`: add or refine CLI/reporting evidence for realtime Xiaozhi
  parity.
- `internal/gateway`: only if trace markers needed for parity are missing.
  Avoid behavioral changes in this transition unless tests show an evidence
  marker is impossible to observe.
- `docs/project_state_machine.md`: add the realtime-parity transition and
  status.
- `docs/agent_handoff_log.md`: record evidence, tests, and remaining risks.
- Future transitions may touch provider streaming adapters and firmware wake
  behavior, but this transition starts with evidence separation.

## Phased Execution

### Phase 1: Evidence Gate

Add a trace-only CLI/report that reads Gateway `/v1/devices`, `/v1/traces`, and
optionally `/v1/audio/recent`, then classifies a real StackChan trace without
driving a fake turn.

Required fields:

- schema/version and generated timestamp;
- gateway label, device id, trace id, session id;
- physical device online/profile/transport flags;
- counts for Opus received/decoded, PCM ingress, VAD speech events, listen
  starts/stops, ASR partial/final, provider first content, TTS first audio,
  downlink first frame, voice pipeline start/completed, device playback start;
- streaming classification and a reason;
- blocked findings for fake paths such as `/say`, host loopback, fast companion,
  missing Opus ingress, missing downlink, or missing pipeline completion;
- redaction flags proving no payloads/credentials/full URLs/local paths are
  stored.

Acceptance:

- Unit tests cover a passing physical trace, missing streaming markers, and
  forbidden fake path markers.
- The report never promotes physical product acceptance by itself.

### Phase 2: Live Physical Smoke

Use the evidence gate after an operator-triggered real `/v1/xiaozhi` turn. The
operator may trigger by screen tap until wake is accepted.

Acceptance:

- Report uses the live device id `44:1b:f6:e2:6a:60`.
- Evidence includes real Opus ingress and Opus downlink from the stock Xiaozhi
  socket.
- If ASR/LLM/TTS are still turn-buffered, the report must say so.

### Phase 3: Realtime Provider Migration Plan

If Phase 2 reports `turn_buffered_xiaozhi_candidate`, create a follow-up worker
transition for replacing the turn-buffered backend with a realtime streaming
provider path.

Acceptance:

- The follow-up transition identifies provider API candidates, cancellation,
  backpressure, jitter buffering, and trace markers before implementation.

## Rollback Plan

- The evidence gate is additive. Rollback is deleting the new CLI/report and
  removing state/log entries.
- No firmware, NVS, provider credential, or audio gain rollback should be needed
  for Phase 1.
- If a Gateway trace marker change becomes necessary and regresses tests, revert
  only that marker change and keep the plan/status docs.

## Risks

- Existing trace names may not distinguish true streaming ASR from final-only
  ASR unless the gate inspects event ordering.
- Host-loopback reports can look green unless the report explicitly rejects
  synthetic device ids and fake path markers.
- Local wake failure can hide realtime quality because the operator must tap to
  enter listening.
- A full realtime provider migration is larger than this evidence transition
  and should not be bundled with the current report gate.

## Human Confirmation Needed

- Trigger one real physical `/v1/xiaozhi` turn after the report gate is merged.
- Confirm whether wake was used or the screen was tapped.
- Confirm audible playback, interruption, and whether the device returned to
  idle without the setup/welcome screen.
