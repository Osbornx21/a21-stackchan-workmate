# 2026-06-03 - Xiaozhi Full Realtime Voice Convergence

Status: active control plan.
Owner: A21 control tower plus scoped workers.
Transition: `T-XIAOZHI-FULL-REALTIME-VOICE-CONVERGENCE-001`.

## Background And Problem Definition

The user review is accepted as the target architecture: A21 must behave like a
Xiaozhi realtime voice device, not like a turn-buffered HTTP question/answer
bot hidden behind a stock-looking WebSocket.

The target chain is:

`mic/i2s -> local wake/VAD -> Opus frame queue -> long Xiaozhi transport ->
streaming ASR -> streaming LLM -> streaming TTS -> paced Opus playback`

A21 has already moved several pieces in the right direction: stock-shaped
`/v1/xiaozhi`, binary Opus ingress/downlink, a Gateway streaming ASR session
seam, a Sherpa JSONL helper candidate, a streaming TTS adapter seam, and a
realtime provider-shape gate. Those are not enough by themselves. The product
is still not accepted until a real stock `/v1/xiaozhi` turn proves the complete
runtime chain on the physical StackChan path, with wake/state/audio behavior
kept honest.

## Current System State

- Main branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Current observed mainline HEAD at plan creation: `61b27ac`.
- Product firmware lane is `a21-stackchan-official-xiaozhi-compatible`; bare
  `xiaozhi.bin` is a non-product comparison artifact and must not be treated as
  a candidate.
- Gateway `/v1/xiaozhi` uses a long WebSocket with JSON control and binary Opus
  media.
- A21 decodes incoming Opus frames and can append PCM frames into a
  `providers.StreamingASRSession` during `listen.start`.
- `VoicePipelineRunner` can reuse `VoicePipelineRequest.ASRTranscript` from a
  streaming ASR final and avoid re-running batch ASR for that turn.
- `VoicePipelineRunner.RunStream()` emits TTS chunks while LLM text deltas are
  segmented, but the behavior is only truly realtime when the selected TTS
  adapter yields audio before a full WAV/file/provider response boundary.
- `doubao_tts_realtime` is a selectable streaming TTS seam, but no real
  provider execution or physical trace has proven it yet.
- `sherpa_onnx_streaming` has a subprocess JSONL helper candidate, but no real
  model smoke or live Gateway trace has proven it yet.
- Local/Iflytek/voice-clone TTS profiles remain file/WAV boundary candidates
  unless a future adapter proves otherwise.
- Wake remains physically unaccepted. Screen tap can be used as an interim
  operator trigger, but it cannot close local wake acceptance.
- Static readiness reports and unit tests must not be promoted to physical PRD
  acceptance.

## Target State

The convergence target is `S-XIAOZHI-FULL-REALTIME-PHYSICAL-CANDIDATE`:

- The physical StackChan product firmware connects to A21 over the stock
  Xiaozhi-compatible socket.
- User/operator can trigger a turn through local wake or, until wake is fixed,
  an explicitly labeled screen-tap fallback.
- The trace contains real Opus ingress from device mic frames.
- Streaming ASR emits first partial before speech/listen finalization and final
  is reused by `VoicePipelineRunner` without batch WAV ASR.
- Streaming LLM emits first content before full answer completion.
- Streaming TTS emits first audio chunk before provider EOF or any WAV/file
  boundary.
- Gateway sends `tts start`, answer `sentence_start`, binary Opus downlink
  frames, and `tts stop` with stable Idle/Listening/Speaking recovery.
- Barge-in/cancel clears provider/downlink work and returns to an honest state.
- Evidence remains redacted and separates host/static/candidate evidence from
  physical product acceptance.

## Non-Goals

- Do not flash bare `xiaozhi.bin`.
- Do not turn `/v1/xiaozhi/say`, host loopback, mock ASR/TTS, or complete WAV
  playback into acceptance evidence.
- Do not remove the accepted 3x contest audio gain or existing volume hotfixes
  in this transition.
- Do not remove voice-clone capability. Voice-clone local TTS may remain a
  fallback while the realtime product path uses a better streaming provider.
- Do not store raw audio, transcripts, prompts, provider outputs, credentials,
  full URLs, or absolute local paths in evidence reports.
- Do not rewrite the entire Gateway in one uncontrolled change.

## Impact Scope

- `internal/gateway/server.go`
  - Turn lifecycle, state markers, ASR streaming session use, downlink pacing,
    barge-in/cancel, and trace markers.
- `internal/providers/voice_pipeline.go`
  - Streaming ASR reuse, LLM segmentation, TTS chunk emission, report semantics.
- `internal/providers/voice_pipeline_adapters.go`
  - Real Sherpa streaming helper and realtime TTS adapter selection.
- `internal/app/xiaozhi_realtime_parity.go`
  - Evidence classification for stock physical traces.
- `internal/app/xiaozhi_streaming_provider_readiness.go`
  - Static provider-shape readiness, kept below physical acceptance.
- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
  - Wake/state/official StackChan behavior only if worker evidence shows a
    device-side overlay gap.
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

## Worker Fan-Out

Three read-only workers have been dispatched from the control thread:

1. Protocol/state-machine parity worker:
   - Thread: `019e8a8a-7268-7210-bf9c-eca8ab1c5c6d`
   - Scope: official Xiaozhi protocol, transport, Application state machine,
     and A21 protocol/Gateway comparison.
2. ESP32/CoreS3 audio HAL worker:
   - Thread: `019e8a8a-7263-7b20-94b9-9b2847aa741d`
   - Scope: official CoreS3 codec/HAL, AudioService, Opus, AFE/wake/VAD/AEC,
     MCP volume, display/touch/RGB/servo behavior.
3. A21 runtime gap worker:
   - Thread: `019e8a8a-7265-71d3-ae47-0dd708965ec6`
   - Scope: actual `/v1/xiaozhi` path and remaining batch/file/blocking gaps.

Workers are read-only. They must return evidence and transition suggestions;
they must not edit, commit, build, flash, start/stop services, execute
provider/V21 calls, write NVS, or play audio.

Second read-only cross-check on current mainline `188b341`:

1. Protocol/state-machine audit:
   - Thread: `019e8ac3-c9f3-7cc3-b8a1-c27cc2748168`
   - Result: A21 matches stock-shaped WebSocket hello/listen/abort, binary
     Opus ingress, and paced downlink, but it remains narrower than full
     Xiaozhi transport because MQTT+UDP is not implemented. WebSocket-only is
     acceptable for the immediate product lane, with MQTT+UDP planned later.
2. Endpoint voice parity audit:
   - Thread: `019e8ac3-c9f7-7721-9f6c-1bce1e69af4c`
   - Result: highest endpoint parity risks are custom wake vs official
     AFE/WakeNet behavior and the parked direct-Xiaozhi path bypassing the
     normal Mooncake/AppLauncher lifecycle. These are physical/product-lane
     transitions, not host smoke acceptance.
3. Gateway/provider realtime gap audit:
   - Thread: `019e8ac3-c9f6-7350-a66e-e51dcdc8109e`
   - Result: the next host-side implementation transition should bridge ASR
     partials into LLM/TTS before ASR final, while keeping real ASR model and
     real TTS execution as separate truthful blockers.
4. Architecture reuse strategy audit:
   - Thread: `019e8ac3-c9fa-7ed0-8b61-625a418a84c2`
   - Result: choose incremental A21 convergence on Xiaozhi firmware/protocol
     and audio-service patterns. Do not embed the full Python/Java/Vue
     Xiaozhi server stack unless an ADR creates an isolated voice-engine
     adapter and preserves A21 Go/provider/V21 boundaries.

## Phased Execution Steps

### Phase 0: Cross-Check Current Truth

Actions:

- Read worker reports when available.
- Confirm current branch/HEAD/dirty state.
- Confirm `T-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-001` and
  `T-XIAOZHI-STREAMING-TTS-ADAPTER-001` are mainline candidates, not runtime
  acceptance.
- Confirm whether any current report already proves real provider execution or
  physical `/v1/xiaozhi` realtime parity.

Acceptance:

- `docs/project_state_machine.md` and `docs/agent_handoff_log.md` record this
  convergence plan and worker dispatch.
- No code change is made before the read-only cross-check is recorded.

### Phase 1: Real Sherpa Streaming ASR No-Audio Smoke

Transition:

- `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001`

Actions:

- Locate approved local Sherpa streaming model path and confirm expected
  `streaming_zipformer` files.
- Run the JSONL helper in a no-audio/no-hardware smoke with synthetic silence
  or a tiny generated PCM frame only if that does not play audio or call a
  provider.
- Add or update a redacted report if one does not exist.

Acceptance:

- Helper starts a real Sherpa streaming recognizer or returns a stable
  model-missing finding.
- No WAV file is written in the streaming path.
- No transcript/audio payload/path/secret is stored in reports.
- If model is missing, transition ends as an honest blocker, not a fake green.

Rollback:

- Revert report-only additions or helper-smoke CLI additions.

### Phase 2: Streaming TTS Runtime Proof Without Physical Hardware

Transition:

- `T-STREAMING-TTS-RUNTIME-PROOF-001`

Actions:

- Use the existing realtime TTS adapter seam with fake realtime connection
  tests and, only when explicitly authorized, a credentialed provider smoke.
- Prove first audio chunk is emitted after first provider audio delta and before
  provider EOF.
- Keep local/Iflytek/voice-clone WAV paths blocked in the strict realtime gate.

Acceptance:

- Provider tests prove no WAV/file boundary for the selected realtime TTS
  adapter.
- `xiaozhi-streaming-provider-readiness` can pass only with streaming ASR
  helper env, streaming LLM profile, and streaming TTS env configured.
- Runtime provider execution, if performed later, produces redacted evidence
  and does not alter Codex proxy settings.

Rollback:

- Revert additive adapter/profile/readiness changes; existing contest audio
  path remains available.

### Phase 3: Stock `/v1/xiaozhi` Realtime Parity Gate

Transition:

- `T-XIAOZHI-PHYSICAL-REALTIME-PARITY-001`

Actions:

- Use only product firmware and stock `/v1/xiaozhi` socket.
- Trigger one physical turn by wake if wake is accepted; otherwise use a
  clearly labeled screen-tap fallback.
- Run the realtime parity report against the trace.
- Verify event ordering:
  - `xiaozhi.listen.start`
  - real `xiaozhi.opus_frame.received/decoded`
  - `asr.stream.start`
  - `asr.audio.append`
  - `asr.first_partial`
  - `asr.final`
  - `llm.first_content` or equivalent text-stream first-content marker
  - `tts.first_audio`
  - `audio.downlink.first_frame`
  - `xiaozhi.voice_pipeline.completed`
  - `tts stop` / idle recovery

Acceptance:

- Classification reaches `xiaozhi_realtime_candidate`, not only
  `turn_buffered_xiaozhi_candidate` or `stock_opus_transport_only`.
- Report uses physical device id and rejects `/say`, host loopback,
  fast-companion, mock, and local fallback traces.
- Operator notes whether trigger was wake or screen tap.

Rollback:

- Evidence-only rollback is deleting the new report. Runtime changes must be
  reverted per their own transition if they caused regression.

### Phase 3a: ASR Partial To LLM Realtime Bridge

Transition:

- `T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001`

Actions:

- Add a narrow host-side bridge so ASR partial events from the stock
  `/v1/xiaozhi` media path can start LLM streaming before ASR final.
- Preserve the current final transcript and batch fallback path.
- Add ordering trace markers and parity-gate rejection for turn-buffered-only
  evidence.

Acceptance:

- Tests prove `asr.first_partial` precedes listen stop or VAD speech end in a
  stock `/v1/xiaozhi` turn.
- Tests prove `llm.first_content` can occur before `asr.final` for the
  partial-driven path.
- Tests prove first TTS audio chunk before pipeline completion, without WAV or
  file boundaries.
- Static, mock, `/say`, host-loopback, and fast-companion traces remain
  rejected as full realtime acceptance.

Rollback:

- Revert bridge changes and return to the final-transcript voice pipeline.
  Existing ASR/TTS seams and paced Opus downlink remain intact.

### Phase 4: Wake And State-Machine Closure

Transition:

- `T-WAKE-LOCAL-XIAOZHI-PARITY-001`

Actions:

- Based on worker evidence, choose either:
  - restore official Xiaozhi wake behavior exactly;
  - tune custom `紫悦` only if ESP-SR/MultiNet evidence supports the phrase;
  - use a longer local phrase as launch fallback while keeping UI copy honest.
- Ensure idle/connecting/listening/speaking states do not trap the device in
  infinite listening or setup/welcome UI.
- Add not-connected UI feedback if the socket is unavailable.

Acceptance:

- Physical proof shows wake from idle before cloud interaction.
- Wake does not require a pre-existing cloud conversation turn.
- Listening state auto-stops or recovers cleanly.
- If wake remains unavailable, product readiness stays red with a documented
  fallback trigger.

Rollback:

- Revert only wake overlay/config changes. Do not replace product firmware with
  bare `xiaozhi.bin`.

### Phase 5: Audio/HAL Parity And Official StackChan Behavior

Transition:

- `T-STACKCHAN-OFFICIAL-AUDIO-HAL-PARITY-001`

Actions:

- Apply only evidence-backed differences from the ESP32/CoreS3 audio worker:
  codec volume/NVS/MCP verification, official AudioService path, AFE/AEC/VAD,
  RGB/touch/servo speaking/listening behavior, and App preload ordering.
- Keep accepted 3x Gateway gain as contest path unless a strictly better
  official-parity path is proven.

Acceptance:

- Product firmware keeps StackChan avatar and official hardware behaviors.
- Audio path stays on official CoreS3/AW88298/ES7210 codec/HAL.
- Operator recording and trace evidence improve or preserve accepted clarity.

Rollback:

- Revert overlay/HAL changes and reflash the last accepted product app only
  under guarded flash rules.

## Risks

- Static green from `xiaozhi-streaming-provider-readiness` can be mistaken for
  runtime green. Every report must state whether provider/physical execution
  occurred.
- A streaming interface can still hide a file boundary internally. Tests must
  assert first audio/partial before EOF or complete response.
- Local wake can remain the slowest path if phrase/assets are wrong. Do not
  claim wake readiness until physical proof exists.
- Provider network behavior can differ between Codex proxy and real mainland
  device path. Codex proxy settings must not be changed casually; LAN/StackChan
  direct-connect policy remains explicit.
- Barge-in can regress if provider sessions and downlink pacer are not cancelled
  together.

## Human Confirmation Needed

- Physical trigger method for parity trace: wake or screen tap.
- Audible quality and interruption observation for the final physical pass.
- Permission before any firmware build/flash, NVS write, provider execution, or
  audio playback.

## Worker Execution Contract

Implementation workers must receive exactly one transition. They must return:

- Branch/HEAD/dirty.
- What this stage did.
- Files changed.
- Tests/builds/run commands and results.
- Whether it deviated from the plan.
- Remaining blockers.
- Recommended next transition.

Workers must not silently expand scope into firmware flashing, provider
execution, V21 execution, audio playback, NVS writes, or unrelated refactors.
