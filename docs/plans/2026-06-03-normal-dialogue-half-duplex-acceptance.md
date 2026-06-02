# T-HALF-DUPLEX-001/002 Normal Dialogue Half-Duplex Acceptance Plan

Status: active plan, no-flash observation track open
Date: 2026-06-03
Owner: A21 control tower
Transition: `T-HALF-DUPLEX-001-NORMAL-DIALOGUE-ECHO-SUPPRESSION-ACCEPTANCE`
and `T-HALF-DUPLEX-002-NO-FLASH-NORMAL-DIALOGUE-OBSERVATION`

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
- Current physical official-compatible firmware is stock Xiaozhi style, not
  the diagnostic mic-probe build. Missing diagnostic counters block only the
  instrumented counter gate; they must not block no-flash normal-dialogue
  self-trigger observation.

## Target State

`S-HALF-DUPLEX-NORMAL-DIALOGUE-PHYSICAL-ACCEPTED`

- A physical normal dialogue or operator-free stock Xiaozhi observation proves
  no self-trigger after assistant TTS.
- No-flash observation reports contain trace event counts and self-trigger
  event names, without dialog text or provider output.
- Instrumented reports contain mic/playback/Gateway counters only when a
  diagnostic firmware exposes them.
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
- A blocked diagnostic-counter report does not by itself block the contest
  no-flash path if Phase 3 proves no self-trigger through stock traces.

### Phase 3: Normal Dialogue Observation

Actions:

- After provider/TTS path is playable, run a normal turn using
  `stackchan-fast-companion-turn` or the Gateway xiaozhi path.
- Observe whether immediate post-TTS microphone ingress starts a new turn.
- If operator touch/wake is unavailable, use the stock `/v1/xiaozhi/say`
  foreground path with a known accepted WAV, wait at least 20 seconds after
  delivery, and inspect the same trace for `xiaozhi.listen.start`,
  `provider.start_turn.start`, `provider.realtime_session.start`, or
  `xiaozhi.voice_pipeline.start`.

Acceptance:

- Trace shows TTS completion, suppression/idle transition, and no unintended
  new ASR/provider turn from self-playback.
- Report status may be `candidate_passed_no_self_trigger` when the no-flash
  stock trace has no self-trigger event names. This is enough to continue the
  contest path, but it remains below diagnostic counter acceptance and full PRD
  acceptance.

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

- Operator confirmation is preferred for final product acceptance, but it is
  not required to keep the no-flash engineering path moving when trace evidence
  already shows no self-trigger.
- Operator must eventually confirm touch/barge-in still works before full PRD
  acceptance.

## 2026-06-03 Execution Update

- Instrumented no-flash counter gate was rerun while device
  `44:1b:f6:e2:6a:60` was online. Report
  `reports/a21-stackchan-half-duplex-acceptance-20260603-015622.json` remained
  blocked because stock firmware exposes `available_xiaozhi_opus_ingress` and
  `available_xiaozhi_opus_downlink`, not A21 diagnostic mic/playback counters.
- Main thread then executed the no-flash normal-dialogue observation track
  without waiting for firmware flash or operator click:
  - trace `a21-trace-no-flash-dialogue-observe-1780423245`;
  - session `a21-session-no-flash-dialogue-observe-1780423245`;
  - input audio basename
    `a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav`;
  - stock `/v1/xiaozhi/say` delivered `40` audio chunks;
  - wait-after-say window was updated to `20000 ms`;
  - trace event count reached `869`;
  - self-trigger event names were empty;
  - `xiaozhi.listen.start.input_suppressed=1`.
- Generated report:
  `reports/a21-no-flash-normal-dialogue-observation-20260603-020055.json`
  has `status=candidate_passed_no_self_trigger`.
- This moves the contest path past the "not flashed / operator did not click"
  blocker. Diagnostic counter acceptance remains a separate optional firmware
  path, not a blocker for continuing voice-chain and wake work.

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
