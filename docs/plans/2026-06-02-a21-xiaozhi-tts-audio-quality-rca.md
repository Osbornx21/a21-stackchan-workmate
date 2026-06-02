# A21 Xiaozhi TTS Audio Quality RCA Plan

Status: active; Phase 1 host downlink isolation integrated, Phase 2 physical
A/B pending.
Transition: `T-AUDIO-001`.
Date: 2026-06-02.

## Background And Problem Definition

The physical StackChan candidate now connects to A21 Gateway through the stock
Xiaozhi WebSocket route and has produced microphone uplink, Gateway downlink,
and barge-in candidate evidence. The user reports that the audible sound is
still wrong and likely a TTS problem.

This is a launch blocker because the current evidence proves Gateway downlink
existence, not acceptable audible playback quality. The same issue must not be
misclassified as full protocol failure, old PCM bridge leakage, or PRD physical
acceptance.

## Current System State

- Control branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Current control HEAD during RCA: `49b9e458d44b`.
- Physical evidence:
  `reports/a21-xiaozhi-physical-evidence-20260602-213147.097784000.json`.
- Readiness evidence:
  `reports/a21-product-readiness-20260602-213204.json`.
- Host voice bench evidence:
  `reports/a21-xiaozhi-voice-bench-20260602-151821.528911000.json`.
- Physical serial evidence:
  `reports/a21-stackchan-physical-wake-serial-20260602-2130.log`.

Observed facts:

- The device is using stock Xiaozhi profile and Opus uplink/downlink, not the
  old A21 PCM bridge product path.
- The backend is still A21 Gateway `/v1/xiaozhi`, not upstream Xiaozhi cloud.
- The current host-local voice pipeline selects `sherpa_onnx_tts` through
  `A21_TTS_FAST_PROFILE` in the readiness evidence.
- The local TTS adapter writes 48 kHz mono PCM chunks for product downlink and
  the Gateway encodes those 60 ms chunks to Opus.
- Existing audio-quality guards check PCM clipping, silence, DC offset,
  headroom, and format. They do not judge voice model naturalness, prosody, or
  user-perceived product fit.
- The stock firmware serial log repeatedly contains `Unknown message type:
  listen` while still entering `speaking`, so audio binary downlink works but
  control-frame cleanliness needs review before launch.

## Target State

`S-AUDIO-TTS-VS-DOWNLINK-ISOLATED`

The project can say, with evidence, whether the bad sound comes primarily from:

- TTS/model/voice generation before Opus encode;
- Gateway PCM chunking/headroom/Opus encode/downlink pacing;
- physical firmware speaker/decode/playback behavior; or
- unsupported stock control messages polluting the runtime.

The result must preserve the current stock Xiaozhi product route and keep
physical acceptance honest.

## Non-Goals

- Do not flash firmware or write NVS.
- Do not start provider or V21 execution.
- Do not play Mac audio.
- Do not claim launch readiness from host-only or operator-unobserved evidence.
- Do not rewrite the voice architecture or switch away from the Go-first
  Gateway foundation.
- Do not reintroduce the old PCM bridge as the product audio path.

## Impact Scope

Likely affected surfaces:

- `internal/app/xiaozhi_voice_bench.go` and related tests if host downlink
  decoded-audio quality evidence is missing.
- `internal/providers/voice_pipeline.go` and TTS adapter tests only if the
  report contract must expose existing PCM quality evidence more clearly.
- `docs/engineering/PROTOCOL.md` only if protocol cleanliness or evidence
  semantics need clarification.
- `docs/project_state_machine.md` and `docs/agent_handoff_log.md` for control
  state.

Forbidden by default:

- firmware source edits;
- flash/NVS/serial writes;
- hardware runtime manipulation from a worker;
- provider/V21 execution;
- Mac audio playback.

## Phased Execution

### Phase 0 - Control RCA And Worker Dispatch

Actions:

- Confirm branch, HEAD, and dirty state.
- Read physical evidence, readiness, serial evidence, and relevant source.
- Dispatch a read-only worker to audit whether audio optimization landed and
  whether the runtime path is stock Xiaozhi or still diagnostic/PCM bridge.

Acceptance:

- Worker returns no-file-change audit.
- Main thread records a ranked hypothesis and the next isolation transition.

Status:

- Completed in the control thread on 2026-06-02.

### Phase 1 - Host Downlink Objective Isolation

Actions:

- Verify or implement a host-only xiaozhi bench/report path that can compare
  generated TTS PCM quality with post-Opus-decoded downlink quality.
- Prefer extending existing `xiaozhi-voice-bench` evidence over adding a new
  command.
- Keep raw PCM, base64 audio, transcript, provider output, full URL, proxy
  value, and local paths out of reports.
- Run focused tests for audio quality, TTS adapter, xiaozhi Opus downlink, and
  physical-evidence gates.

Acceptance:

- A report can show selected TTS profile, PCM quality before downlink when
  available, and decoded Opus/downlink quality without physical acceptance
  overclaim.
- If the host post-Opus quality is objectively bad, the failure can be fixed
  before another hardware window.
- If host quality is clean, the next proof moves to physical A/B.

Status:

- Completed by worker `019e8895-43a6-7e23-a4f3-601f0451ab50` and integrated on
  the control branch.
- `xiaozhi-voice-bench` now decodes captured binary downlink Opus frames into
  aggregate `downlink_audio_quality` evidence.
- No raw PCM, base64 audio, transcript, provider output, full URL, proxy value,
  local path, firmware write, NVS write, serial write, provider execution, V21
  execution, or Mac audio playback was introduced.
- The result remains host-only candidate evidence and does not satisfy physical
  audible acceptance.

### Phase 2 - Foreground Physical A/B

Actions:

- Keep the same flashed firmware, NVS route, Gateway port, and stock Xiaozhi
  path.
- Change only the TTS candidate or fixture source in a foreground-controlled
  run.
- Compare current `sherpa_onnx_tts` against a known-good pre-rendered or
  voice-clone candidate when available.
- Collect operator or instrument audible observation tied to a fresh trace.
- Regenerate `xiaozhi-physical-evidence` and product readiness.

Acceptance:

- If known-good fixture or alternate TTS is clean and `sherpa_onnx_tts` is bad,
  classify the blocker as TTS/model/voice quality.
- If both are bad through the same route, classify the blocker as
  Opus/downlink/device playback.
- If both are clean after cleanup, proceed to playback ack and launch-readiness
  closure.

### Phase 3 - Narrow Fix Or Route Decision

Actions:

- If TTS is the root cause, switch the product TTS candidate through an
  explicit host-side profile and document why.
- If Opus/downlink is the root cause, fix the chunking, rate, pacing, headroom,
  or encode/decode seam with tests.
- If protocol cleanliness is the root cause, remove or gate stock-incompatible
  server control messages such as stock-client `listen` acks.

Acceptance:

- Focused tests pass.
- Host downlink objective evidence passes.
- A fresh physical trace has audible observation or trusted playback ack.
- Product readiness still keeps non-closed gates false.

## Rollback Plan

- Do not mutate firmware or NVS in this transition.
- If a TTS profile change hurts latency or quality, revert the profile/env
  selection and keep the current `sherpa_onnx_tts` path as the baseline.
- If a protocol cleanup breaks host tests, revert the cleanup and keep the
  existing compatibility shim until a stock-specific fix is ready.
- Preserve all before/after reports and keep host-only reports out of PRD
  acceptance.

## Risk List

- Host audio-quality metrics can pass while the voice still sounds unnatural.
- Physical speaker path can sound bad even when host post-Opus metrics pass.
- Stock firmware warnings can be mistaken for audio codec failure.
- Changing TTS may introduce latency regressions.
- Voice-clone or remote TTS dependencies may be unavailable in a hardware
  window.
- Reports can accidentally overclaim physical acceptance if candidate labels
  are loosened.

## Manual Confirmation Points

- Operator must confirm any foreground Gateway restart or TTS profile swap used
  for physical A/B.
- Operator or approved instrument must provide audible observation before PRD
  physical acceptance.
- Any firmware flash, NVS write, or serial write remains outside this plan and
  requires a separate guarded hardware transition.

## Worker Execution Task

Worker transition:

- `T-AUDIO-001 Phase 1: host downlink objective isolation`.

Worker branch/worktree:

- Create a scoped worker worktree from the current control state, suggested
  branch `codex/a21-xiaozhi-audio-quality-ab-20260602`.

Worker objective:

- Determine whether the existing host evidence can already isolate TTS PCM
  quality from Opus/downlink quality.
- If not, implement the smallest report/test addition needed, preferably in
  `xiaozhi-voice-bench`, to decode captured xiaozhi binary downlink frames and
  attach redacted aggregate `audio_quality` evidence.
- Do not touch the physical device.

Worker boundaries:

- Allowed writes, only if needed:
  - `internal/app/xiaozhi_voice_bench.go`
  - `internal/app/app_test.go` or a focused xiaozhi bench test file
  - `internal/audio` tests only if a bug is proven
  - `docs/engineering/PROTOCOL.md` only if report semantics change
- Forbidden:
  - firmware edits;
  - flash, NVS, serial, or hardware writes;
  - Gateway start/stop on the main hardware port;
  - provider/V21 execution;
  - Mac audio playback;
  - raw audio/transcript/provider output/full URL/proxy/local path leakage.

Required verification:

- `go test ./internal/audio ./internal/providers ./internal/gateway ./internal/app -run 'PCMQuality|LocalTTS|VoicePipeline|Xiaozhi.*Opus|Xiaozhi.*Stock|XiaozhiWebSocketListen|XiaozhiPhysicalEvidence|XiaozhiVoiceBench' -count=1`
- `git diff --check`
- If Go changes are broad, run `make verify`.

Worker return format:

- branch/HEAD/dirty;
- files changed;
- what evidence/report capability was found or added;
- whether audio-quality optimization was already landed or needed changes;
- exact tests and pass/fail;
- whether any plan boundary was violated;
- remaining TTS vs Opus/device blockers;
- exact next foreground physical A/B command or runbook step for the control
  thread.
