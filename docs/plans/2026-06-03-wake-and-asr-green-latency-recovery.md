# 2026-06-03 - Wake Failure And ASR Green Latency Recovery

Status: active.
Owner: A21 control tower.
Transitions:

- `T-WAKE-003-ZI-YUE-PHRASE-TUNING`
- `T-ASR-GREEN-LATENCY-001-XIAOZHI-LISTEN-AUTO-STOP`

## Background And Problem

The operator physically rejected the flashed `紫悦` wake candidate: saying
`紫悦` produced no response. This must be recorded as a wake failure, not a
pending success.

The operator also reported that after ASR turns green, the device waits for a
long time. Live trace `a21-trace-44-1b-f6-e2-6a-60` shows repeated stock
Xiaozhi `listen.start` windows, including turns that reached roughly 25s, 38s,
70s, and 108s before auto-stop or before a new listen started. This is a
Gateway listen/VAD收口 problem and must not be mixed into wake acceptance.

## Current System State

- Product firmware lane remains
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- Current flashed wake config is custom MultiNet `zi yue` / display `紫悦`,
  threshold `20`, with AFE/HiStackChan WakeNet disabled.
- The physical device is online through Gateway `127.0.0.1:21081`.
- The latest trace contains valid `vad.speech.end` and
  `xiaozhi.listen.auto_stop` events, but some listen windows are far too long.
- Trace summary currently becomes misleading when one physical trace id is
  reused over many reconnects/turns.

## Target State

- Wake remains product-lane only and is marked failed until a new physical proof
  passes.
- The displayed wake identity remains `紫悦`, but the custom MultiNet command
  list includes longer phrase alternatives so the local recognizer has a more
  robust match surface.
- Gateway stock Xiaozhi listen has a bounded maximum duration after speech is
  detected, with an environment override.
- Trace summaries prefer the latest complete event pair instead of pairing the
  first event in a long reused trace with a later turn.

## Non-Goals

- Do not flash bare `xiaozhi.bin`.
- Do not change provider selection, TTS gain, V21, NVS, or Wi-Fi.
- Do not claim wake product readiness without operator physical proof.
- Do not rewrite the audio front-end or replace official Xiaozhi protocol.

## Impact Scope

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `internal/app/official_stackchan_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

## Execution Steps

1. Mark `T-WAKE-002` physical proof as rejected in the state machine.
2. Tune the custom wake command string while keeping display `紫悦`.
3. Add a Gateway max-listen safety stop for stock Xiaozhi turns once speech has
   been detected.
4. Add an env override `A21_XIAOZHI_LISTEN_MAX_MS`.
5. Fix trace summary pairing so reused hardware trace ids do not inflate
   latency numbers across unrelated turns.
6. Run focused tests.
7. If tests pass, commit.
8. Build and guarded-flash the updated official-compatible product app only
   after the worktree is clean and the official-compatible flash plan is clean.
9. Ask the operator to retry wake physically.

## Acceptance Criteria

- Focused Gateway tests prove max-listen auto-stop records
  `xiaozhi.listen.max_duration_auto_stop` and starts the voice pipeline.
- Focused app test proves `A21_XIAOZHI_LISTEN_MAX_MS` wiring.
- Focused overlay test proves wake display remains `紫悦` and command aliases
  remain in the official-compatible product lane.
- `git diff --check` passes.
- `make verify` passes before any flash execute.
- Physical wake remains not accepted until the operator confirms response.

## Rollback

- Revert only this transition's Gateway/app/overlay/doc changes.
- If firmware physical behavior regresses, reflash the last accepted
  official-compatible app artifact through the guarded product lane.
- If wake false-activates, keep the product lane and tune the command list or
  threshold in a new transition.

## Risks

- MultiNet may still reject or weakly recognize a short two-syllable command.
- Max listen duration can cut off unusually long utterances; keep it env
  configurable.
- Trace summary improvement is diagnostic only and must not be used as physical
  acceptance by itself.

## Manual Confirmation Points

- Operator confirms whether `紫悦`, `紫悦紫悦`, `你好紫悦`, or `小紫悦` wakes the
  device.
- Operator confirms whether the green ASR waiting time is materially shorter.

## 2026-06-03 Gateway No-Speech Cooldown Update

Live trace `a21-trace-44-1b-f6-e2-6a-60` later showed a second failure mode
after the firmware listen-bound flash: repeated
`listen.stop -> placeholder_no_asr_tts -> listen.start` loops. The trace had
`xiaozhi.listen.start=333`, `audio.ingress.buffered=25115`, and
`stackchan.official_auto.not_connected=1025`.

The bounded Gateway fix is:

- arm a short stock-physical input cooldown after `placeholder_no_asr_tts`;
- record `xiaozhi.no_speech.input_suppression_armed`;
- record reason-specific listen suppression such as
  `xiaozhi.listen.start.suppressed_after_no_speech`;
- keep `/v1/xiaozhi/say` and touch-abort suppression reason-specific as
  `suppressed_after_host_say` and `suppressed_after_barge`.

Acceptance evidence for this update:

- focused Gateway tests prove immediate restart after no-speech placeholder is
  ignored and does not open a second turn;
- `make verify` passes;
- Gateway `a21-gateway-21081` is restarted from the patched worktree;
- fresh device trace after restart contains only `xiaozhi.hello.received`
  before operator touch/wake.

This update does not claim wake acceptance. It only reduces the green-listening
loop risk so wake and touch can be physically validated from a cleaner idle
state.
