# StackChan Official MCP Status Parity Plan

Transition id: `T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001`

Follow-up freeze:
`T-STACKCHAN-OFFICIAL-MCP-SPEAKER-VOLUME-FREEZE-001` in
`docs/plans/2026-06-04-stackchan-official-mcp-speaker-volume-freeze.md`
later folds the already-existing `self.audio_speaker.set_volume` delivery path
into the same `/v1/xiaozhi/mcp-control` surface while keeping this transition's
screen/status scope historically intact.

Parent plan:
`docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`.

Capability charter:
`docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`, especially the
Official StackChan Parity Gap Map frozen by
`T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001`.

## Current State

- The official parity gap map is frozen and identifies low-risk MCP/status
  parity as the next worker transition.
- Gateway currently exposes stock Xiaozhi MCP delivery for
  `self.audio_speaker.set_volume` only.
- `self.get_device_status`, `self.screen.set_brightness`,
  `self.screen.set_theme`, and `self.screen.get_info` are not Gateway
  whitelisted.
- `self.reboot`, `self.upgrade_firmware`, `self.camera.take_photo`,
  `self.screen.snapshot`, camera/video, NFC, infrared, and app lifecycle remain
  out of scope.
- The current worker branch starts from the pushed main-control dispatch commit
  `919e4c4 docs(control): prepare mcp status parity dispatch`.

## Target State

- Gateway has a low-risk, A21-namespaced Xiaozhi MCP control surface that can
  construct and deliver only these newly allowed tools:
  `self.get_device_status`, `self.screen.set_brightness`,
  `self.screen.set_theme`, and `self.screen.get_info`.
- Every request carries or receives generated `trace_id`, `session_id`, and
  `device_id`.
- Trace/device activity records only redacted delivery markers and safe control
  metadata, never raw MCP response bodies, screenshots, images, provider
  output, secrets, URLs, paths, transcripts, or audio payloads.
- Docs/state/handoff clearly say this is control-contract parity only. It is
  not firmware work, physical acceptance, or product acceptance.

## Do Not Do

- Do not build firmware, flash firmware, touch serial, write NVS, start Gateway,
  execute provider APIs, execute V21, or touch physical hardware.
- Do not expose or implement `self.reboot`, `self.upgrade_firmware`,
  `self.camera.take_photo`, `self.screen.snapshot`, camera stream/video, NFC,
  infrared, or app lifecycle controls.
- Do not store or log raw MCP response bodies, screenshots, images, provider
  output, secrets, full URLs, local paths, transcripts, or raw/base64 audio.
- Do not claim official support as A21 product acceptance.
- Do not revert internal test 3 Xiaozhi voice/protocol work or internal test 4
  roleplay/professional/workspace job work.

## Impact Scope

- Primary code: `internal/gateway/server.go`,
  `internal/gateway/server_test.go`, and, if useful for operator visibility,
  `internal/gateway/simulator.go`.
- Docs/state: `docs/engineering/PROTOCOL.md`,
  `docs/engineering/OBSERVABILITY.md`,
  `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`,
  `docs/project_state_machine.md`, and `docs/agent_handoff_log.md`.
- No new ports, environment variables, production dependencies, containers, or
  firmware artifacts.

## Execution Steps

1. Add a Gateway route for the scoped MCP status/control surface.
2. Define typed request/response structs with an explicit action/tool whitelist.
3. Build stock MCP `tools/call` envelopes through the existing Xiaozhi transport
   builder and wrap them in the existing `/v1/xiaozhi` text message format with
   `trace_id`, `session_id`, and `device_id`.
4. Validate per-tool arguments:
   - `self.get_device_status`: no user arguments.
   - `self.screen.set_brightness`: bounded integer brightness.
   - `self.screen.set_theme`: bounded named theme string, no URLs/paths.
   - `self.screen.get_info`: no user arguments.
5. Block all non-whitelisted MCP tools, especially reboot, upgrade, camera,
   snapshot, stream/video, NFC, infrared, and app lifecycle names.
6. Record redacted trace markers and redacted device activity only after
   delivery succeeds.
7. Add focused tests for allowed tools, high-risk blocked tools, no raw result
   storage, trace/session/device propagation, and simulator presence if the
   simulator is touched.
8. Update protocol, observability, capability charter, project state, and
   handoff log.
9. Run focused tests, `git diff --check`, and `GOMAXPROCS=2 make verify`.
10. Commit and push the scoped worker branch if verification passes.

## Acceptance

- Allowed requests construct and deliver the correct stock MCP `tools/call`
  payloads with no raw response capture.
- Blocked tool names return a client error and no MCP websocket message.
- Brightness/theme/info/status requests carry or are assigned `trace_id`,
  `session_id`, and `device_id`, and record only redacted trace markers.
- Docs/state/handoff agree that this transition is low-risk parity/control
  contract only and does not include firmware or physical product acceptance.
- Focused tests, `git diff --check`, and `GOMAXPROCS=2 make verify` pass.

## Failure State

- Gateway allows a high-risk official tool or any legacy identity-looking MCP
  tool.
- Gateway records raw MCP response bodies, screenshots, image data, provider
  output, secrets, full URLs, local paths, transcripts, or audio payloads.
- The docs imply product acceptance without physical evidence.
- Verification fails.

## Rollback

- Revert the single scoped worker commit for this transition.
- Keep existing `self.audio_speaker.set_volume` behavior and all internal test 3
  and internal test 4 work intact.
- Keep firmware, flash artifacts, NVS, serial state, Gateway runtime, providers,
  V21, and physical hardware untouched.

## Human Confirmation Points

- Main control must review before promoting this worker result from low-risk
  parity/control contract to any product or physical acceptance claim.
- Any future expansion to reboot, OTA, camera/photo, snapshot, stream/video,
  NFC, infrared, app lifecycle, or firmware changes requires a new plan or ADR
  and explicit operator approval.
