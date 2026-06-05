# StackChan Official Power And UI Parity Recovery

Status: build-ready transition.
Created: 2026-06-05.

## Goal

Recover the product StackChan from the foreground report that pressing the
upper-left power button while unplugged only flashes the screen/red LED and then
stops, while also restoring the official launcher/setup/mobile association
front-end as the default product entry.

## Root-Cause Finding

The no-cable symptom occurs before Gateway, provider, or voice runtime can be
involved. The failing chain is:

1. Physical power key wakes the StackChan PMIC.
2. AXP2101 must hold board power long enough for ESP32 boot and HAL init.
3. Official launcher/setup/apps run and preserve Wi-Fi/mobile association.
4. User selects the AI agent/A21 mode, then Xiaozhi/Gateway voice starts.

Current A21 product overlay changed step 2 by writing AXP2101 registers that are
not present in the official StackChan package:

- `WriteReg(0x10, common_config | 0x04)`
- `WriteReg(0x22, 0b110)`

It also changed step 3 by defining `A21_OEM_AUTOSTART_XIAOZHI=1`, patching
`main.cpp`, calling `GetHAL().startXiaozhi()` directly after app install, and
parking `app_main`. That disabled the official launcher/setup/mobile association
path. The direct-start path was useful for the previous no-welcome fix, but it
is the wrong product default once the official front-end is accepted.

## Target State

- PMIC initialization remains byte-for-byte official for the power-key
  lifecycle: A21 product overlay no longer patches `firmware/main/hal/board/stackchan.cc`.
- Official launcher/setup/home/app selection lifecycle remains default:
  A21 product overlay no longer patches `firmware/main/main.cpp` and no longer
  defines `A21_OEM_AUTOSTART_XIAOZHI`.
- A21 remains the backend and mode service after the user enters the official
  AI agent / A21 front-end:
  `secret_logic::get_server_url()` returns
  `CONFIG_A21_STACKCHAN_OFFICIAL_GATEWAY_BASE_URL`.
- Official `AppAvatar` relay keeps the A21 `device_id` query parameter so the
  Gateway can bind body/action state.

## Acceptance

- Focused overlay tests pass.
- `make verify` passes.
- Product build lane
  `a21-stackchan-official-xiaozhi-compatible-build` passes with
  `official_avatar_action_preserved=true`,
  `official_xiaozhi_start_preserved=true`, and `minimal_bridge_screen=false`.
- Guarded product flash uses only
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- Foreground operator confirms no-USB power-key boot reaches the official
  StackChan front-end instead of only flashing and stopping.

## Forbidden Actions

- Do not use the generic `xiaozhi.bin` product flash lane.
- Do not write provider keys into firmware.
- Do not reset or write NVS for this transition.
- Do not run Git prune/gc cleanup.
- Do not roll back internal-test3 Gateway/voice protocol changes.
