# A21 StackChan Official Wi-Fi Provisioning Punch-Through

Date: 2026-06-06

## Transition

`T-A21-STACKCHAN-OFFICIAL-WIFI-PROVISIONING-PUNCHTHROUGH-001`

## Current State

- Internal test 4 can use preloaded or preserved Wi-Fi credentials as the
  product floor.
- The official StackChan frontend is preserved before `AI.AGENT`, but the
  no-preloaded-Wi-Fi path has not been made explicit in A21 tooling.
- Official setup uses BLE/app events and writes `app_config/is_configed=true`
  only after Wi-Fi connects. A21's existing NVS lane can mark that gate true
  when credentials are written or preserved, which is correct for preloaded
  internal-test usage but not for first-user provisioning.
- A21 Gateway has Xiaozhi OTA/WS and StackChan body WS surfaces, but not the
  official StackChan account/device-info HTTP compatibility endpoints used by
  the official frontend account path.

## Target State

- Keep the internal-test4 preloaded/preserved Wi-Fi path intact.
- Add an explicit first-boot provisioning NVS mode that clears stale Wi-Fi
  credentials and `app_config/is_configed`, while still pointing OTA/WS to A21.
- Route official StackChan account/device-info calls to A21 Gateway and provide
  compatible JSON responses.
- Verify with focused unit tests and `make verify`; do not flash or write NVS
  in this transition.

## Actions

1. Add Gateway handlers for `/stackChan/device/user`,
   `/stackChan/device/info`, and `/stackChan/device/unbind`.
2. Patch the official-compatible overlay so account URLs use the A21 Gateway
   base URL, with `ws/wss` converted to `http/https` for account HTTP calls.
3. Add `--first-boot-config` to
   `a21-stackchan-official-xiaozhi-compatible-nvs`, plus Makefile/env support.
4. Extend NVS tests to prove first-boot mode clears Wi-Fi credentials and keeps
   `app_config/is_configed=0` until the official BLE setup succeeds.
5. Update handoff/state with results and remaining physical acceptance steps.

## Acceptance

- Focused Gateway tests pass for official StackChan device-data endpoints.
- Focused App/NVS tests pass for first-boot provisioning mode and overlay
  routing.
- `GOMAXPROCS=2 make verify` passes.
- No flash, no NVS write, no provider key in firmware, no prune/gc, and no
  rollback of internal-test3/internal-test4 behavior.
