# T-HALF-DUPLEX-001 Normal Dialogue Half-Duplex Acceptance Plan

Status: active plan
Date: 2026-06-03
Owner: A21 control tower
Transition: `T-HALF-DUPLEX-001-NORMAL-DIALOGUE-ECHO-SUPPRESSION-ACCEPTANCE`

## Background And Problem Definition

Foreground host-say suppression has been physically observed after the accepted
3x audio hotfix. That is not the same as normal dialogue half-duplex: after a
real user turn and assistant TTS, StackChan must not hear itself and start a
new unwanted turn, while still allowing touch/barge-in and later listening.

## Current System State

- Accepted audio hotfix commit: `060d2bb`; current baseline: `b5a405e`.
- `/v1/xiaozhi/say` arms short input suppression and recorded markers
  `xiaozhi.say.input_suppression_armed=1` and
  `xiaozhi.listen.start.input_suppressed=1`.
- Existing `stackchan-half-duplex-acceptance` command measures diagnostic
  mic-probe counters, audio websocket delivery, Gateway ingress, playback
  chunks, and speaker runtime echo.
- Current physical official-compatible firmware may not be the diagnostic
  mic-probe build. If microphone capability is not
  `diagnostic_probe_m5unified_i2s_capture`, this transition blocks honestly.

## Target State

`S-HALF-DUPLEX-NORMAL-DIALOGUE-PHYSICAL-ACCEPTED`

- A physical normal dialogue or instrumented equivalent proves no self-trigger
  after assistant TTS.
- Report contains mic/playback/Gateway counters and no transcript or provider
  output.
- The system remains interruptible by touch/barge-in.

## Non-Goals

- Do not change TTS gain, provider selection, wake-word firmware, V21, or
  firmware without a separate guarded plan.
- Do not claim full PRD acceptance from host-only tests.
- Do not mask self-trigger by permanently disabling microphone/listen.

## Impact Scope

- Gateway/device reports.
- `stackchan-half-duplex-acceptance` report.
- Optional narrowly scoped Gateway suppression tuning only if evidence proves
  a specific self-trigger window.

## Phased Execution

### Phase 1: Live Readiness Check

Actions:

- Check Gateway health and `/v1/devices`.
- Confirm device id, firmware commit, connection freshness, microphone
  capability, speaker capability, playback runtime echo, and last session.

Acceptance:

- Device is online and fresh enough for an acceptance attempt, or blocker is
  recorded with exact report fields.

### Phase 2: Instrumented Half-Duplex Probe

Actions:

- Run:
  `A21_GATEWAY_URL=http://127.0.0.1:21081 A21_DEVICE_ID=<device> make stackchan-half-duplex-acceptance`
- Use conservative thresholds first: `min-mic-frames=1`,
  `min-playback-chunks=1`, delivery ratio `0.95`.

Acceptance:

- Report status is confirmed and all deltas pass thresholds.
- If blocked, findings identify mic capability, gateway metrics, delivery
  ratio, playback drops, or speaker driver errors.

### Phase 3: Normal Dialogue Observation

Actions:

- After provider/TTS path is playable, run a normal turn using
  `stackchan-fast-companion-turn` or the Gateway xiaozhi path.
- Observe whether immediate post-TTS microphone ingress starts a new turn.

Acceptance:

- Trace shows TTS completion, suppression/idle transition, and no unintended
  new ASR/provider turn from self-playback.

### Phase 4: Minimal Fix If Needed

Actions:

- If self-trigger happens, tune only the smallest Gateway/turn suppression
  window required.
- Add a focused test for the exact timing/marker failure.

Acceptance:

- Focused test passes.
- Physical observation is repeated and accepted.

## Rollback

- Revert only the half-duplex suppression change if it blocks user barge-in or
  touch.
- Keep accepted 3x gain and stock protocol.

## Risks

- Current firmware may not expose diagnostic mic counters.
- Phone-room audio can still be picked up; report counters must distinguish
  device self-trigger from room acoustics.
- Over-suppression can make the device feel deaf.

## Human Confirmation Points

- Operator must confirm whether the device self-triggers after TTS.
- Operator must confirm touch/barge-in still works.

## Worker Execution Task

Worker owns only readiness/probe/report work for `T-HALF-DUPLEX-001`.

Allowed:

- Read Gateway/device state.
- Run `stackchan-half-duplex-acceptance` if the live device is online.
- Add focused host tests/docs if a code issue is found.

Forbidden:

- Firmware flash/NVS writes.
- Provider/TTS changes.
- Wake-word changes.
- Claiming full PRD acceptance without physical proof.

Return summary format:

- Gateway/device state.
- Command run and report path.
- Pass/block status and findings.
- Whether any code changed.
- Next physical action.
