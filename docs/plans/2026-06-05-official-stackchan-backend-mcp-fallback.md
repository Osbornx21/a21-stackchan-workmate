# Official StackChan Backend MCP Fallback

Date: 2026-06-05 CST
Owner: A21 control tower
Transition: `T-OFFICIAL-STACKCHAN-BACKEND-MCP-FALLBACK-001`

## Current State

- `/v1/stackchan/official/control` is truthful and strict: without a connected
  official `/stackChan/ws` relay it returns HTTP 409.
- `/workspace` has a client-side fallback that catches that 409 and then calls
  the separate Xiaozhi MCP body-preset/body-motion endpoints.
- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` flagged official
  StackChan body parity as incomplete when relay delivery and MCP fallback are
  split outside the Gateway state machine.

## Target State

- Keep strict official delivery as the default.
- Add an explicit `allow_mcp_fallback=true` request flag for
  `/v1/stackchan/official/control`.
- When the official relay is disconnected and fallback is explicitly allowed,
  map safe official state/face/motion semantics to existing bounded Xiaozhi MCP
  body presets or motions.
- Record fallback traces and device registry metadata without claiming official
  frame delivery or physical acceptance.

## Boundaries

- Do not flash firmware, write NVS, reboot the device, expose power/shutdown,
  OTA, camera, NFC, infrared, snapshot/video, or app-lifecycle controls.
- Do not modify provider or V21 execution paths.
- Do not mark physical acceptance true.
- Preserve internal-test3 voice/protocol behavior.

## Acceptance

- Default official control without `/stackChan/ws` still returns HTTP 409.
- Explicit fallback delivers a Xiaozhi MCP sequence and returns
  `status=fallback_delivered`,
  `delivered_transport=xiaozhi_mcp_sequence`,
  `official_action_physical_accepted=false`, and
  `official_action_fallback_reason=official_stackchan_ws_disconnected`.
- Traces include `stackchan.official_mcp_fallback.*` markers.
- `/v1/devices.runtime_echo` records `official_stackchan_fallback_status` and
  keeps `official_stackchan_packets=0`.
- Focused Gateway tests and full project gates pass.

## Rollback

- Remove `allow_mcp_fallback` handling and keep the existing strict official
  409 behavior plus the older workspace client-side fallback.
