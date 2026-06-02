# 2026-06-03 - Boot Idle Socket And Not Connected UX

Status: active.
Owner: A21 control tower.
Transition: `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001`.
Sub-transition: `T-BOOT-IDLE-SOCKET-001`.

## Background And Problem

The latest physical observation shows the wake failure is not only a phrase
recognition issue. The device cannot wake by voice before the Xiaozhi socket is
opened, and touching the screen opens the socket by entering listening, which
then presents as an endless green ASR/listening state.

This means the product currently binds three separate concerns too tightly:

- boot/network connection;
- idle socket/hello establishment;
- user intent to start listening.

The user also reported that when no connection exists, the screen gives no
graceful feedback. The device should make the not-yet-connected state visible
with a small, calm UI affordance instead of appearing inert.

## Current System State

- Firmware product lane is
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- A21 direct autostart bypasses the welcome/setup trap and starts Xiaozhi
  runtime directly; this plan changes that to preload official apps, request
  Xiaozhi immediately, and still bypass the visible welcome/setup flow.
- Gateway `21081` can accept the stock Xiaozhi WebSocket once the device opens
  the channel.
- The current touch path opens the audio channel by entering listening.
- A prior source tree contains a quiet idle channel pattern named for X21; A21
  may reuse the behavior but must not carry X21 product identity or user-facing
  copy.

## Target State

- After boot and activation, StackChan opens the Xiaozhi WebSocket/hello path
  while remaining in idle/standby.
- Official StackChan apps are installed before the immediate Xiaozhi request,
  preserving the hardware/front-end initialization surface without showing the
  setup trap.
- Voice wake can be detected from idle after the device is connected.
- The device does not send `listen.start` merely to establish the socket.
- Touch may still intentionally enter listening, but it must not be the only
  way to create the socket.
- When the socket is not ready, the screen shows a small graceful status such
  as "正在连接紫悦服务..." and then "紫悦已就绪，可以叫我".
- A21 overlay and tests use A21 names and copy, not X21 names/copy.

## Non-Goals

- Do not restore the welcome/setup first-run screen.
- Do not flash bare `xiaozhi.bin`.
- Do not change provider/TTS/V21/gain policy.
- Do not make wake product-ready claims until physical wake proof passes.
- Do not invent a debug protocol extension in stock Xiaozhi hello.

## Impact Scope

- Official-compatible overlay:
  `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- Focused overlay tests:
  `internal/app/official_stackchan_test.go`
- Governance docs:
  `docs/project_state_machine.md`
  `docs/agent_handoff_log.md`

## Execution Steps

1. Add an A21-named Kconfig option for quiet idle websocket preconnect.
2. Patch the official source so the idle control-channel logic uses A21 names
   and user-facing `紫悦` copy.
3. Ensure quiet preconnect calls `OpenAudioChannel()` while idle but does not
   call `SendStartListening()`.
4. Keep idle state wake detection enabled.
5. Add focused tests asserting A21 names/copy and absence of added X21 keep
   control symbols/copy.
6. Build the official-compatible product app and inspect `sdkconfig.json`.
7. Run no-write flash plan, then guarded flash execute only from a clean
   worktree.
8. Verify device appears connected/ready without touch, then physically test
   voice wake.

## Acceptance Criteria

- Focused app tests pass.
- `make verify` passes.
- Generated `sdkconfig.json` includes
  `A21_STACKCHAN_KEEP_CONTROL_CHANNEL=true`.
- Device reconnects to Gateway after boot without needing screen touch.
- Screen gives a graceful connecting/ready message when connection is not yet
  available.
- No `listen.start` is observed merely from boot preconnect.
- Physical wake remains pending until the operator confirms wake works.

## Rollback

- Revert this overlay/test/doc transition only.
- Reflash the previous official-compatible product app if boot UI, socket, or
  audio behavior regresses.

## Risks

- A quiet Xiaozhi WebSocket may time out if the server expects active traffic;
  monitor reconnect behavior.
- Touch still intentionally starts listening. If it remains too sticky, a
  separate transition should change click semantics or add device-side
  stop/timeout behavior.
- User-facing copy must remain small and non-invasive.

## Manual Confirmation Points

- Whether the device connects to Gateway after boot without touch.
- Whether the screen shows the connecting/ready hint before connection.
- Whether voice wake works from idle once connected.
- Whether touching screen still causes an unacceptable infinite green state.
