# StackChan Official Power And UI Parity Recovery

Status: flashed, official front-end restored, App BLE secret pending.
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
- Product state machine boundary is explicit:
  boot, PMIC, launcher, Home, Setup, Wi-Fi provisioning, Wi-Fi NVS persistence,
  mobile association surfaces, and up-swipe Home navigation stay official until
  the user opens the official `AI.AGENT` app. Opening `AI.AGENT` is the only
  product transition into the A21 runtime.
- A21 remains the backend and mode service after the user enters the official
  AI agent / A21 front-end:
  `secret_logic::get_server_url()` returns
  `CONFIG_A21_STACKCHAN_OFFICIAL_GATEWAY_BASE_URL`.
- Official `AppAvatar` relay keeps the A21 `device_id` query parameter so the
  Gateway can bind body/action state.
- The official Xiaozhi voice client receives its endpoint through A21 OTA:
  `CONFIG_OTA_URL` points to `http://47.103.57.217/xiaozhi/ota/`, and the
  Gateway returns `ws://47.103.57.217/v1/xiaozhi`.
- The official body/action relay uses the same A21 Gateway origin and connects
  to `/stackChan/ws?deviceType=StackChan&device_id=...`.
- Existing Wi-Fi credentials remain owned by the official firmware/NVS path;
  A21 must not reset, rewrite, or fork that path for this transition.

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
- Foreground operator opens `AI.AGENT` and confirms voice/body runtime connects
  to A21 Gateway while the official Home/Setup/Wi-Fi experience remains usable
  before entry and recoverable on reboot.

## App BLE Association Finding

The restored official mobile App currently reports `Failed to process device
data` during BLE association. This is not a Gateway, PMIC, or A21 voice runtime
failure. The App shows that toast while processing BLE `notifyState` type `4`;
it expects `data.state` to be RSA-OAEP(SHA-256) encrypted, decrypts it through
`RsaUtil.decryptStackChanBlue`, and then reads the first 12 plaintext
characters as the device MAC.

The public official firmware source keeps
`secret_logic::generate_handshake_token()` as the placeholder `hi-stack-chan`,
and the public App source keeps `ValueConstant.stackChanBluePrivateKey` empty.
The operator-provided device ID `441BF6E26A60` is the needed plaintext MAC, but
the stock official App still requires the matching BLE public key or the closed
official `secret_logic` implementation to produce the encrypted state it can
decrypt.

Until that secret material or an A21 companion-App/key-pair path is available,
the product-compatible route is: preserve official front-end and Wi-Fi/NVS,
enter A21 by opening `AI.AGENT`, and route voice/body links to A21 Gateway.

## Forbidden Actions

- Do not use the generic `xiaozhi.bin` product flash lane.
- Do not write provider keys into firmware.
- Do not reset or write NVS for this transition.
- Do not run Git prune/gc cleanup.
- Do not roll back internal-test3 Gateway/voice protocol changes.
