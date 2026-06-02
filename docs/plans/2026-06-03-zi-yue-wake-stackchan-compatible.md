# 2026-06-03 - Zi Yue Wake In StackChan-Compatible App

Status: active.
Owner: A21 control tower.
Transition: `T-WAKE-002-ZI-YUE-IN-STACKCHAN-COMPATIBLE-APP`.

## Background And Problem

The 2026-06-03 wake-word flash attempt proved that flashing a bare
`xiaozhi.bin` can make the device boot into the plain Xiaozhi UI and lose the
StackChan avatar/action surface. That path is now guarded and must stay
non-product/dev-only.

The next wake-word action must keep the product candidate in the
`a21-stackchan-official-xiaozhi-compatible` lane while adding the requested
Chinese wake word `紫悦`.

## Current System State

- Current firmware product candidate: `a21-stackchan-official-xiaozhi-compatible`.
- Current source lane preserves official StackChan `AppAvatar`, `AppAiAgent`,
  `GetHAL().startXiaozhi()`, and the official codec path.
- Current overlay sets codec output volume before entering the official
  Xiaozhi runtime.
- Current build config still uses the stock WakeNet/AFE wake setup; custom
  MultiNet wake is not yet integrated into this product candidate.
- The restored physical device is back on the StackChan-compatible UI after the
  bare `xiaozhi.bin` incident.

## Target State

- The product candidate remains
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- The firmware build uses official StackChan board identity:
  `CONFIG_BOARD_TYPE_M5STACK_STACK_CHAN=y`.
- The wake implementation uses official-source custom MultiNet config:
  `CONFIG_USE_CUSTOM_WAKE_WORD=y`.
- The wake phrase is `zi yue` and the display/greeting string is `紫悦`.
- Initial threshold is `20`, matching the official StackChan custom wake default
  and prioritizing wake responsiveness for contest acceptance.
- MultiNet7 Chinese quantized model is included for the custom wake path when
  the source Kconfig supports it.
- The stock HiStackChan WakeNet model is explicitly disabled so the product
  candidate has one active wake identity: `紫悦`.
- A21 autostart preserves the official StackChan main flow: official apps are
  installed first, then A21 sets codec volume and requests Xiaozhi start through
  `GetHAL().requestXiaozhiStart()`. The final official
  `GetHAL().startXiaozhi()` path remains preserved.
- The product candidate can be built, no-write planned, and flashed only through
  `a21-stackchan-official-xiaozhi-compatible-flash-*`.

## Non-Goals

- Do not flash or promote a bare `xiaozhi.bin` as an A21 product app.
- Do not replace the StackChan avatar, screen, touch, servo, RGB, or Xiaozhi
  runtime surface in this transition.
- Do not change provider selection, TTS routing, Gateway protocol, V21 adapter,
  Wi-Fi/NVS provisioning, or audio gain policy.
- Do not claim full wake acceptance until the physical device responds to
  `紫悦` after the product-candidate flash.
- Do not treat the separate xiaozhi/StackChan hardware-behavior comparison
  thread as a conclusion until it returns evidence.

## Impact Scope

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- Focused firmware overlay tests under `internal/app/`
- Governance docs:
  - `docs/project_state_machine.md`
  - `docs/agent_handoff_log.md`

## Execution Steps

1. Update the official-compatible overlay to switch wake config from AFE
   WakeNet to custom MultiNet with `zi yue` / `紫悦`, threshold `20`, and the
   Chinese MultiNet7 model symbol when supported by the source tree.
2. Add or extend focused tests to assert the product overlay still preserves
   official Xiaozhi runtime startup and now carries the `紫悦` custom wake
   contract.
3. Assert the overlay installs the official StackChan apps before A21 requests
   Xiaozhi autostart, so the compatible app does not bypass official app/action
   initialization.
4. Run focused Go tests for the official StackChan app lane.
5. Run `git diff --check`.
6. Build the product candidate:
   `make a21-stackchan-official-xiaozhi-compatible-build`.
7. Inspect the generated build config under
   `/tmp/a21-stackchan-official-build/config/sdkconfig.json` and confirm board,
   wake type, phrase, display string, threshold, and app artifact.
8. Inspect patched `firmware/main/main.cpp` in the official-clean workdir and
   confirm app installation happens before A21 `requestXiaozhiStart()`.
9. Run no-write flash plan:
   `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 make a21-stackchan-official-xiaozhi-compatible-flash-plan`.
10. Because the user explicitly allowed this round to burn/verify after merge,
   execute the guarded flash only if the no-write plan remains product-candidate
   clean:
   `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP_FLASH_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP make a21-stackchan-official-xiaozhi-compatible-flash-execute`.
11. After flash, run the lightest live verification available in the current
   Gateway state: volume command and/or a short stock `/v1/xiaozhi/say` trace.
12. Ask the operator to physically try `紫悦`; only mark wake accepted if the
    device responds on the StackChan-compatible UI.

## Acceptance Criteria

- Focused tests pass.
- `git diff --check` passes.
- Product build passes and emits
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`.
- Build config proves:
  - `BOARD_TYPE_M5STACK_STACK_CHAN=true`;
  - `USE_CUSTOM_WAKE_WORD=true`;
  - `USE_AFE_WAKE_WORD=false`;
  - `CUSTOM_WAKE_WORD=zi yue`;
  - `CUSTOM_WAKE_WORD_DISPLAY=紫悦`;
  - `CUSTOM_WAKE_WORD_THRESHOLD=20`;
  - `SR_MN_CN_MULTINET7_QUANT=true` when available.
  - `SR_WN_WN9_HISTACKCHAN_TTS3=false`.
- Flash plan and execute reports identify the app as
  `a21-stackchan-official-xiaozhi-compatible.bin` at app offset `0x20000`.
- Physical device still shows the StackChan-compatible app after reboot.
- Wake acceptance remains pending until the operator confirms `紫悦` works.

## Rollback

- If build fails, revert only the overlay/test changes from this transition.
- If the physical device UI regresses, immediately reflash the last accepted
  StackChan-compatible candidate via
  `a21-stackchan-official-xiaozhi-compatible-flash-execute`.
- If `紫悦` false-wakes, keep the product lane and tune only
  `CONFIG_CUSTOM_WAKE_WORD_THRESHOLD`, first from `20` to `35`.
- If `紫悦` fails to wake, keep the product lane and test phrase alternatives
  or a trained custom model in a separate planned transition.

## Risks

- `zi yue` may be too short for robust MultiNet wake detection and may need
  threshold or phrase tuning.
- A threshold of `20` improves sensitivity but can increase false wakes.
- Custom wake may affect first-turn audio timing depending on whether wake data
  is sent. This transition keeps `CONFIG_SEND_WAKE_WORD_DATA=n` to avoid
  introducing a protocol behavior change.
- The source tree may silently change model symbols; build config inspection is
  required before flash.

## Manual Confirmation Points

- The operator must confirm the physical device still shows the StackChan UI
  after flash.
- The operator must confirm whether saying `紫悦` wakes the device without
  touching the screen.
- The operator must report false wake or missed wake behavior before threshold
  tuning is promoted.

## Worker Task

Worker boundary for this transition:

- Own only the official-compatible wake overlay and focused tests.
- Do not touch Gateway/provider/TTS/network/V21 code.
- Do not flash, write NVS, start long-running services, or execute provider
  calls.
- Return:
  - files changed;
  - focused tests run;
  - whether the overlay still preserves official StackChan runtime;
  - whether implementation deviated from this plan;
  - residual risks.
