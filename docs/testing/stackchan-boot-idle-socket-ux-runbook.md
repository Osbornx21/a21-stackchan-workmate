# StackChan Boot Idle Socket UX Runbook

Status: active testing note.
Transition: `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001`.
Sub-transition: `T-BOOT-IDLE-SOCKET-001`.

This runbook is independent from the implementation plan in
`docs/plans/2026-06-03-boot-idle-socket-and-not-connected-ux.md`. It is meant
to help integration review and operator validation without owning the firmware
overlay implementation.

## Scope

Validate that the A21 StackChan product lane handles boot, idle connection, and
touch/listen recovery honestly:

- before the stock Xiaozhi WebSocket is connected, the device must not look like
  it is already listening for a turn;
- not-connected UX must be calm and explicit enough for the operator to know
  the device is waiting for Gateway/socket readiness;
- screen touch may request or accelerate socket/listen readiness, but must not
  strand the device in infinite green listening;
- after a socket exists and speech is detected, the Gateway listen bound remains
  the acceptance backstop, currently `A21_XIAOZHI_LISTEN_MAX_MS`.

## Existing Focused Coverage

- `TestDeviceControlArmWithoutSocketDoesNotLeaveValidationState` proves the old
  A21 audio WebSocket control path returns `409` when no socket is present and
  does not leave validation/listening state armed.
- `TestXiaozhiWebSocketMaxListenDurationAutoStopsAfterSpeech` proves the stock
  Xiaozhi listen path records `xiaozhi.listen.max_duration_auto_stop`,
  `xiaozhi.listen.auto_stop`, and `xiaozhi.voice_pipeline.start` after speech
  is detected.
- `TestGatewayServerOptionsFromEnvWiresXiaozhiListenMaxDuration` proves
  `A21_XIAOZHI_LISTEN_MAX_MS` reaches Gateway options.

These are host-side tests. They do not prove physical boot-screen UX until the
product overlay is built, flashed through the official-compatible lane, and
observed by the operator.

## Operator Acceptance Checklist

Use the official-compatible product lane only:
`a21-stackchan-official-xiaozhi-compatible.bin`.

1. Boot without touching the screen.
   - Expected: StackChan does not present a misleading permanent green
     listening state before the stock Xiaozhi WebSocket is connected.
   - Expected: any disconnected/waiting display is honest and recoverable.

2. Observe pre-socket idle.
   - Expected: wake-word silence before socket readiness is not counted as a
     wake failure by itself.
   - Expected: the operator can distinguish "waiting for connection" from
     "listening to speech".

3. Touch the screen once while not connected.
   - Expected: touch may open or retry the stock socket path.
   - Expected: if listen starts but no valid speech/turn completes, the display
     returns to idle or not-connected feedback instead of staying green forever.

4. Speak a short utterance after socket readiness.
   - Expected: trace includes a bounded listen turn with
     `xiaozhi.listen.auto_stop` and `xiaozhi.voice_pipeline.start`.
   - Expected: if speech was detected, green listening should stop near the
     configured `A21_XIAOZHI_LISTEN_MAX_MS` bound unless the implementation
     records an explicit, reviewed exception.

5. Disconnect Gateway or block socket readiness in a controlled no-provider
   setup.
   - Expected: the device reports not-connected/waiting feedback, not product
     failure or silent green listening.

## Evidence To Collect

- branch and HEAD used for the product build;
- product build report and app SHA-256;
- no-write flash plan report, if a flash is planned;
- guarded product flash report, if a flash is executed;
- Gateway trace ID for the touch/listen attempt;
- trace event counts for `xiaozhi.listen.start`,
  `xiaozhi.listen.max_duration_auto_stop`, `xiaozhi.listen.auto_stop`, and
  `xiaozhi.voice_pipeline.start`;
- operator observation for not-connected screen feedback and whether green
  listening recovers.

## Integration Recommendation

Merge the control-tower plan and overlay tests first. Treat this runbook as a
review checklist and physical acceptance companion. Do not use it to justify a
flash, service restart, provider run, V21 run, or wake-word acceptance by
itself.
