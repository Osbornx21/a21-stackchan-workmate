# StackChan Official Hardware Parity Full Landing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the gap between A21's current StackChan product lane and the official Xiaozhi StackChan hardware/control/status surface without weakening A21 identity, firmware flash guards, provider-key isolation, or physical-evidence discipline.

**Architecture:** Treat official StackChan/Xiaozhi as a reference hardware and protocol surface, not as an authority over A21 product behavior. Land parity in staged transitions: first freeze the gap map, then expose low-risk MCP/status controls, then restore official-compatible display/action semantics, then graduate physical hardware surfaces only with diagnostics and acceptance evidence.

**Tech Stack:** Go Gateway/Core, A21 protocol/device registry, stock `/v1/xiaozhi` WebSocket, official StackChan binary avatar/action frames, guarded official-compatible firmware overlay, repo-carried docs/plans/handoff/state machine, `make verify`, `make preflight`, `make doctor`.

---

## Control-Tower Transition

Transition id: `T-STACKCHAN-OFFICIAL-HARDWARE-PARITY-001`

Current state:

- A21 product firmware uses the official-compatible app artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- The product device `44:1b:f6:e2:6a:60` has connected through the stock
  `/v1/xiaozhi` path.
- Runtime speaker volume and accepted relay audio have physical/operator
  evidence.
- Current A21 live hardware/control surface is narrower than the official
  StackChan/Xiaozhi package: A21 mainly exposes stock audio, speaker volume,
  limited official avatar/action relay, selected touch acceptance cases, and
  planned capability tracks.
- Full PRD physical acceptance remains open.

Target state:

- A21 has a documented official-hardware parity matrix covering every hardware
  control and status-display surface from the official package.
- Every gap has an owner transition, reason, difficulty, acceptance evidence,
  rollback path, and main-thread dispatch boundary.
- Low-risk Gateway/MCP/status controls are implemented and tested without
  firmware flash.
- Medium-risk display/avatar/action/touch/RGB/servo controls are implemented
  only behind A21 semantic contracts and physical acceptance gates.
- High-risk camera/video/NFC/infrared/app-lifecycle surfaces remain scoped
  spikes until privacy, safety, and physical evidence justify product use.

Trigger:

- Operator comparison found that A21's current hardware control and status
  display are still below the official Xiaozhi StackChan package.

Action:

- Execute the staged tasks below in main-control order. Do not dispatch a later
  physical/firmware stage until its prerequisite doc/test/runtime evidence is
  present.

Acceptance condition:

- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`,
  `docs/engineering/PROTOCOL.md`, `docs/engineering/A21_CURRENT_CONTROL.md`,
  `docs/project_state_machine.md`, and `docs/agent_handoff_log.md` agree on
  what is available, diagnostic, planned, blocked, or product-accepted.
- `GET /v1/devices` and related Gateway control surfaces expose no hidden or
  false hardware claims.
- Official-compatible product flash guard remains intact:
  `a21-stackchan-official-xiaozhi-compatible.bin` is the only product StackChan
  app artifact.
- Relevant Go tests pass, `git diff --check` passes, and `make verify` passes
  for code/doc changes checked by the repo.
- Physical promotion requires a fresh device-specific report with `trace_id`,
  `session_id`, `device_id`, and no raw secrets/transcripts/audio payloads.

Failure state:

- A worker exposes official features as product-available without physical or
  diagnostic evidence.
- A worker adds an unguarded flash/NVS/serial command.
- A worker stores provider credentials, raw transcripts, raw audio, or
  provider outputs in repo docs/logs/reports.
- A worker bypasses A21 Gateway/Core ownership by placing provider keys or
  network policy in firmware.
- A worker regresses the no-welcome official-compatible product boot path.

Rollback path:

- Revert only the scoped worker commit(s) for the failed transition.
- Keep the latest accepted product firmware artifact and NVS values unchanged
  unless the main control thread explicitly approves a guarded rollback flash.
- Keep product readiness blocked rather than downgrading acceptance checks.

Next state:

- `S-STACKCHAN-OFFICIAL-HARDWARE-PARITY-MAPPED`
- `S-STACKCHAN-OFFICIAL-HARDWARE-PARITY-LOW-RISK-CONTROLS-LANDED`
- `S-STACKCHAN-OFFICIAL-HARDWARE-PARITY-PHYSICAL-EVIDENCE-GATED`

## Reference Baseline

Use these local references for the first parity pass:

- Official StackChan source:
  `/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan`.
- Official StackChan root README hardware table:
  screen, capacitive touch, camera, proximity and ambient light sensor, IMU,
  microSD, speaker, dual microphones, battery, two servos, 12 RGB LEDs, IR,
  NFC, OTA, mobile/remote controls.
- Official StackChan HAL:
  `firmware/main/hal/hal.h`,
  `firmware/main/hal/hal.cpp`,
  `firmware/main/hal/hal_ws_avatar.cpp`,
  `firmware/main/hal/hal_servo.cpp`,
  `firmware/main/hal/hal_io_expander.cpp`,
  `firmware/main/hal/hal_imu.cpp`,
  `firmware/main/hal/hal_head_touch.cpp`.
- Official Xiaozhi source inside the StackChan tree:
  `firmware/xiaozhi-esp32/main/application.cc`,
  `firmware/xiaozhi-esp32/main/device_state_machine.*`,
  `firmware/xiaozhi-esp32/main/mcp_server.cc`,
  `firmware/xiaozhi-esp32/main/boards/common/board.*`,
  `firmware/xiaozhi-esp32/main/display/display.h`,
  `firmware/xiaozhi-esp32/main/audio/audio_service.*`,
  `firmware/xiaozhi-esp32/main/protocols/protocol.h`,
  `firmware/xiaozhi-esp32/main/protocols/websocket_protocol.cc`.

Reference caution:

- The local source trees may contain historical branch edits. The baseline
  worker must record the exact Git remotes, branches, HEAD commits, dirty
  state, and whether each quoted file is read from `HEAD:` or the working tree.
- Do not copy official code into A21 blindly. Map official behavior into A21's
  provider-neutral and device-neutral contracts.

## Parity Matrix To Freeze

The baseline worker must freeze this matrix into
`docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md` or a linked
subsection in `docs/engineering/A21_CURRENT_CONTROL.md`.

| Surface | Official package | A21 current | Landing class |
| --- | --- | --- | --- |
| Microphone | Dual mic, audio service, Opus uplink, VAD/wake/AEC paths | Stock `/v1/xiaozhi` uplink observed, PRD physical acceptance still open | Physical evidence gate |
| Speaker | Codec output, volume, playback queue, speaking UI | Runtime volume MCP and accepted 3x relay evidence | Freeze plus playback-start proof |
| Screen | Status, notification, emotion, chat, theme, power save, status bar | Limited status messages and A21 registry/expression contract | Medium-risk UI parity |
| Screen touch | Tap toggles listen/config paths | Acceptance CLI cases exist | Low-medium physical gate |
| Top touch | Press/release/swipe zones | Acceptance CLI cases exist | Low-medium physical gate |
| Servo Y/pitch | Official pitch servo angle/torque/motion | Stable A21 surface | Keep and verify |
| Servo X/yaw | Official yaw servo angle/PWM/motion | Planned or partial sequence use | Medium-high safety gate |
| RGB | 12 LED individual/all control | Available/planned state expression, no rich API | Medium Gateway/action mapping |
| Camera/photo | Camera init, MCP photo | Planned | High privacy spike |
| Camera stream/video | JPEG frame, start/stop stream, video mode | Not product surface | High privacy/bandwidth spike |
| IMU | BMI270 shake event | Planned diagnostic | Medium diagnostic-to-product |
| Ambient/proximity | Official hardware surface | Planned diagnostic | Medium diagnostic-to-product |
| Battery/charging | PMIC/battery status and UI status bar | Planned/status gap | Low read-only, medium product |
| NFC | Official hardware surface | Planned | High product-semantics spike |
| Infrared | Official hardware surface | Planned | High product-semantics spike |
| MCP device status | `self.get_device_status` | Not live-whitelisted | Low |
| MCP speaker volume | `self.audio_speaker.set_volume` | Live-whitelisted | Done/freeze |
| MCP brightness/theme/info | screen brightness, theme, info, snapshot | Not live-whitelisted | Low-medium |
| MCP reboot/OTA | system info, reboot, upgrade | Not product-exposed | High guardrail |
| Device state display | Xiaozhi starting/wifi/idle/connect/listen/speak/upgrade/fatal | Coarser A21 state/trace view | Medium-high |
| Official WS avatar | avatar, motion, dance, heartbeat, text, call, video, camera, audio stream | Only avatar/motion/dance/heartbeat subset | Staged medium/high |
| App lifecycle | Mooncake/AppLauncher/AppAiAgent/AppAvatar/AppDance/AppSetup | Product path parks after direct Xiaozhi start to avoid setup trap | High reconciliation |
| OTA/provisioning | Official OTA/provisioning/mobile app ecosystem | Product OTA route exists, app ecosystem not product surface | Medium/high by scope |

## File Map

Documentation files:

- Modify `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md` to carry
  the frozen official-vs-A21 capability table and promotion gates.
- Modify `docs/engineering/PROTOCOL.md` to document any new Gateway endpoint,
  MCP whitelist entry, official binary frame, event kind, trace marker, and
  physical acceptance limit.
- Modify `docs/engineering/A21_CURRENT_CONTROL.md` to summarize the current
  active hardware parity state and operator next actions.
- Modify `docs/engineering/OBSERVABILITY.md` when a new runtime marker, metric,
  or trace classification is introduced.
- Modify `docs/engineering/LATENCY_BUDGET.md` only if a control path becomes
  latency-sensitive.
- Modify `docs/project_state_machine.md` after each worker completion.
- Append `docs/agent_handoff_log.md` after each worker round.

Gateway/protocol files:

- Modify `internal/gateway/server.go` for new HTTP controls, MCP whitelist
  routing, device registry fields, trace markers, and official-control writes.
- Modify `internal/gateway/server_test.go` for endpoint, whitelist, redaction,
  trace, and stock-profile regression tests.
- Modify `internal/protocol/message.go` only if the device event/control
  schema needs new stable A21 fields.
- Modify `internal/protocol/message_test.go` for identity, namespace, and
  schema guard coverage.
- Modify `internal/transport/stackchan/official_avatar.go` for new official
  binary frame builders or stricter mapping.
- Modify `internal/transport/stackchan/official_avatar_test.go` for frame shape
  and unsupported-event behavior.

CLI/readiness files:

- Modify `internal/app/app_stackchan_mainline.go` for capability promotion
  checks and generated reports.
- Modify `internal/app/app_stackchan_touch.go` only for touch acceptance
  additions.
- Modify related `internal/app/*_test.go` files for no-flash/no-secret/no-raw
  evidence behavior.
- Modify readiness collectors only after a new evidence report is stable.

Firmware files:

- Modify `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
  only in transitions that explicitly allow firmware work.
- Do not add a product flash command outside the guarded official-compatible
  lane.
- Do not use generic `xiaozhi-firmware-flash-*` or `xiaozhi.bin` for the
  product StackChan device.

## Worker Summary Format

Every worker must return:

```markdown
Transition:
What changed:
Files changed:
Tests run and results:
Runtime or physical evidence:
Deviations from plan:
Remaining issues:
Next suggested action:
Forbidden actions avoided:
```

## Task 1: Freeze Official Hardware Parity Gap Map

Transition id: `T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001`

Boundary:

- Read-only against official source trees.
- Repo edits limited to docs and state/handoff files.
- No code implementation, no provider execution, no Gateway start, no firmware
  build, no flash, no serial, no NVS write.

Files:

- Modify `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- Modify `docs/engineering/A21_CURRENT_CONTROL.md`
- Modify `docs/project_state_machine.md`
- Append `docs/agent_handoff_log.md`

- [ ] Step 1: Record official source identity.

Run:

```bash
git -C "/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan" remote -v
git -C "/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan" status --short --branch
git -C "/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan" rev-parse HEAD
git -C "/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan/firmware/xiaozhi-esp32" remote -v
git -C "/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan/firmware/xiaozhi-esp32" status --short --branch
git -C "/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan/firmware/xiaozhi-esp32" rev-parse HEAD
```

Expected:

- Commands either pass or the worker records the exact available local source
  path used instead.
- Dirty official source trees are recorded as dirty and treated as reference
  material only.

- [ ] Step 2: Extract official control/status inventory.

Run:

```bash
rg -n "DataType|StartCamera|StopCamera|ControlAvatar|ControlMotion|Dance|SetBrightness|SetTheme|get_device_status|take_photo|DeviceState|SetStatus|SetEmotion|ShowNotification|Battery|Charging|Imu|Touch|Servo|RGB|NFC|Infrared" \
  "/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan/firmware" \
  "/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan/firmware/xiaozhi-esp32/main"
```

Expected:

- Output is summarized, not pasted wholesale.
- Summary covers every row in the parity matrix above.

- [ ] Step 3: Extract A21 current inventory.

Run:

```bash
rg -n "capabilities|runtime_echo|speaker-volume|official/control|ControlAvatar|ControlMotion|DanceSequence|screen_touch|top_barge_in|imu|ambient_light|proximity|battery|camera|nfc|infrared|servo_x|rgb" \
  docs/engineering internal firmware/stackchan-official
```

Expected:

- Output is summarized into current available/diagnostic/planned/blocked
  states.
- No A21 planned surface is promoted only because the official package has it.

- [ ] Step 4: Update the capability charter.

Add a section named:

```markdown
## Official StackChan Parity Gap Map
```

The section must include:

- official source identity,
- A21 source identity,
- the parity matrix,
- one landing class per surface,
- explicit statement that full official availability is not product acceptance.

- [ ] Step 5: Update control docs and state.

`docs/engineering/A21_CURRENT_CONTROL.md` must name
`T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001` and point to this plan.

`docs/project_state_machine.md` must add the transition as a next candidate or
active transition according to the main control thread's decision.

- [ ] Step 6: Verify docs.

Run:

```bash
git diff --check
make verify
```

Expected:

- Both pass.
- If `make verify` fails for unrelated dirty main-control work, record the
  exact failing package/file and do not mask it.

Commit message:

```bash
git add docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md docs/engineering/A21_CURRENT_CONTROL.md docs/project_state_machine.md docs/agent_handoff_log.md
git commit -m "docs(stackchan): map official hardware parity gaps"
```

## Task 2: Add Low-Risk Xiaozhi MCP And Device Status Parity

Transition id: `T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001`

Boundary:

- Gateway-only and tests.
- No firmware build, no flash, no serial, no NVS write.
- Use stock Xiaozhi MCP tools already advertised by the connected firmware.
- Do not expose reboot, upgrade, camera, snapshot upload, or privacy-sensitive
  tools in this task.

Files:

- Modify `internal/gateway/server.go`
- Modify `internal/gateway/server_test.go`
- Modify `docs/engineering/PROTOCOL.md`
- Modify `docs/engineering/A21_CURRENT_CONTROL.md`
- Append `docs/agent_handoff_log.md`

Control surface:

```http
POST /v1/xiaozhi/device-status
POST /v1/xiaozhi/screen-brightness
POST /v1/xiaozhi/screen-theme
GET  /v1/xiaozhi/mcp-capabilities?device_id=<device_id>
```

Allowed MCP tools:

```text
self.get_device_status
self.screen.set_brightness
self.screen.set_theme
self.screen.get_info
```

Request shape:

```json
{
  "device_id": "44:1b:f6:e2:6a:60",
  "trace_id": "a21-trace-operator-control",
  "session_id": "a21-session-operator-control"
}
```

Brightness request shape:

```json
{
  "device_id": "44:1b:f6:e2:6a:60",
  "brightness": 70,
  "trace_id": "a21-trace-screen-brightness",
  "session_id": "a21-session-screen-brightness"
}
```

Theme request shape:

```json
{
  "device_id": "44:1b:f6:e2:6a:60",
  "theme": "dark",
  "trace_id": "a21-trace-screen-theme",
  "session_id": "a21-session-screen-theme"
}
```

- [ ] Step 1: Write failing endpoint tests.

Test cases:

- rejects missing `device_id`;
- rejects brightness outside `0..100`;
- rejects theme not in `light|dark|auto`;
- rejects a stock device without MCP capability;
- sends the exact MCP tool names above;
- records trace markers without raw tool result bodies;
- never accepts `self.reboot`, `self.upgrade_firmware`, `self.camera.take_photo`,
  or `self.screen.snapshot` in this transition.

Run:

```bash
go test ./internal/gateway -run 'TestXiaozhi.*(DeviceStatus|ScreenBrightness|ScreenTheme|MCPCapabilities)' -count=1
```

Expected:

- Tests fail because endpoints or tool mapping do not exist yet.

- [ ] Step 2: Implement minimal Gateway handlers.

Implementation requirements:

- Reuse the existing Xiaozhi live socket registry and MCP request path used by
  `POST /v1/xiaozhi/speaker-volume`.
- Require the connected device to advertise `mcp`.
- Require or synthesize A21 trace/session IDs and keep them in traces.
- Return redacted metadata only:

```json
{
  "ok": true,
  "device_id": "44:1b:f6:e2:6a:60",
  "mcp_tool": "self.get_device_status",
  "trace_id": "a21-trace-operator-control",
  "session_id": "a21-session-operator-control",
  "result_redacted": true
}
```

- Do not persist raw MCP payloads in docs/logs/reports.

- [ ] Step 3: Update protocol docs.

Add a subsection under the Xiaozhi MCP/expression contract describing:

- endpoint names,
- accepted fields,
- rejected tools,
- redaction behavior,
- product acceptance limit.

- [ ] Step 4: Run focused and broad verification.

Run:

```bash
go test ./internal/gateway -count=1
git diff --check
make verify
```

Expected:

- All pass, or unrelated main-control dirty failures are recorded exactly.

Commit message:

```bash
git add internal/gateway/server.go internal/gateway/server_test.go docs/engineering/PROTOCOL.md docs/engineering/A21_CURRENT_CONTROL.md docs/agent_handoff_log.md
git commit -m "feat(gateway): add safe xiaozhi mcp status controls"
```

## Task 3: Align A21 Device Registry With Official Status Display

Transition id: `T-STACKCHAN-OFFICIAL-STATUS-DISPLAY-PARITY-001`

Boundary:

- Gateway/protocol/app registry only.
- No firmware build or flash.
- Do not claim physical screen rendering until operator evidence exists.

Files:

- Modify `internal/protocol/message.go`
- Modify `internal/protocol/message_test.go`
- Modify `internal/gateway/server.go`
- Modify `internal/gateway/server_test.go`
- Modify `docs/engineering/PROTOCOL.md`
- Modify `docs/engineering/OBSERVABILITY.md`
- Append `docs/agent_handoff_log.md`

Stable A21 display state map:

```text
starting
wifi_configuring
idle
connecting
listening
thinking
speaking
upgrading
audio_testing
error
fatal_error
```

Official-to-A21 mapping:

```text
unknown -> error
starting -> starting
wifi_configuring -> wifi_configuring
idle -> idle
connecting -> connecting
listening -> listening
speaking -> speaking
upgrading -> upgrading
activating -> connecting
audio_testing -> audio_testing
fatal_error -> fatal_error
```

- [x] Step 1: Write failing protocol tests.

Test cases:

- device events can carry `display_state`;
- unknown official states normalize to `error`;
- identity guard rejects forbidden namespace pollution in status fields;
- every display state is ready to carry `trace_id`, `session_id`, `device_id`.

Run:

```bash
go test ./internal/protocol -run 'Test.*DisplayState|Test.*Identity' -count=1
```

Expected:

- Tests fail before schema/mapping exists.

- [x] Step 2: Write Gateway registry tests.

Test cases:

- `/v1/devices` includes latest `display_state`;
- status updates do not erase capabilities/runtime echo;
- status updates are timestamped and trace-linked;
- status display evidence is not treated as physical acceptance.

Run:

```bash
go test ./internal/gateway -run 'Test.*Device.*DisplayState|Test.*StatusDisplay' -count=1
```

Expected:

- Tests fail before registry support exists.

- [x] Step 3: Implement registry fields and trace markers.

Required trace markers:

```text
stackchan.display_state.received
stackchan.display_state.normalized
stackchan.display_state.registry_updated
```

Registry response shape:

```json
{
  "device_id": "44:1b:f6:e2:6a:60",
  "display_state": "listening",
  "display_state_source": "xiaozhi",
  "display_state_trace_id": "a21-trace-display-state",
  "display_state_session_id": "a21-session-display-state",
  "display_state_physical_accepted": false
}
```

- [x] Step 4: Update docs.

Document that this is registry/status parity, not physical screen acceptance.

- [x] Step 5: Verify.

Run:

```bash
go test ./internal/protocol ./internal/gateway -count=1
git diff --check
make verify
```

Expected:

- All pass.

Commit message:

```bash
git add internal/protocol/message.go internal/protocol/message_test.go internal/gateway/server.go internal/gateway/server_test.go docs/engineering/PROTOCOL.md docs/engineering/OBSERVABILITY.md docs/agent_handoff_log.md
git commit -m "feat(stackchan): record official display state parity"
```

## Task 4: Expand Official Avatar, Motion, RGB, And Servo Semantic Mapping

Transition id: `T-STACKCHAN-OFFICIAL-ACTION-PARITY-001`

Boundary:

- Gateway/transport mapping and simulator tests first.
- Physical device control only after main thread opens a hardware window.
- No firmware flash in the first worker.
- Keep `display` unsupported inside `internal/transport/stackchan` unless a
  real official packet shape is implemented and tested.

Files:

- Modify `internal/transport/stackchan/official_avatar.go`
- Modify `internal/transport/stackchan/official_avatar_test.go`
- Modify `internal/gateway/server.go`
- Modify `internal/gateway/server_test.go`
- Modify `docs/engineering/PROTOCOL.md`
- Modify `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- Append `docs/agent_handoff_log.md`

Semantic action map:

```text
state.idle -> avatar neutral, pitch center, rgb soft_idle
state.listening -> avatar attentive, pitch slight_up, rgb listening
state.thinking -> avatar thinking, pitch small_nod, rgb thinking
state.speaking -> avatar speaking, pitch mouth_sync_candidate, rgb speaking
state.error -> avatar error, pitch center, rgb error
motion.nod -> official motion pitch sequence
motion.shake -> official motion yaw sequence
motion.dance -> official DanceSequence packet
motion.stop -> neutral motion stop
```

- [ ] Step 1: Write failing transport tests.

Test cases:

- every semantic state emits a deterministic official packet sequence;
- yaw movement remains clamped and documented as `servo_x_candidate`;
- dance uses official `DanceSequence` frame;
- heartbeat behavior is unchanged;
- unsupported camera/video/call frames are not emitted by this task.

Run:

```bash
go test ./internal/transport/stackchan -count=1
```

Expected:

- Tests fail for missing mapping or metadata.

- [ ] Step 2: Implement semantic mapping.

Implementation requirements:

- Reuse official frame types already present in the adapter.
- Keep packet builders small and testable.
- Clamp yaw/pitch values before frame construction.
- Return a structured unsupported result for hardware that is not available
  or not accepted.

- [ ] Step 3: Add Gateway control tests.

Test cases:

- `POST /v1/stackchan/official/control` emits expected packet count;
- action requests include `trace_id`, `session_id`, `device_id`;
- disconnected official avatar socket returns a truthful blocked response;
- physical acceptance stays false until a report says otherwise.

Run:

```bash
go test ./internal/gateway -run 'Test.*Official.*Control|Test.*StackChan.*Action' -count=1
```

Expected:

- Tests fail before Gateway mapping updates, then pass after implementation.

- [ ] Step 4: Verify.

Run:

```bash
go test ./internal/transport/stackchan ./internal/gateway -count=1
git diff --check
make verify
```

Expected:

- All pass.

Commit message:

```bash
git add internal/transport/stackchan/official_avatar.go internal/transport/stackchan/official_avatar_test.go internal/gateway/server.go internal/gateway/server_test.go docs/engineering/PROTOCOL.md docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md docs/agent_handoff_log.md
git commit -m "feat(stackchan): expand official action parity mapping"
```

## Task 5: Physical Touch, Barge-In, RGB, And Servo Evidence Window

Transition id: `T-STACKCHAN-OFFICIAL-ACTION-PHYSICAL-EVIDENCE-001`

Boundary:

- Requires main thread approval for a live hardware window.
- No flash unless main thread explicitly says the current product firmware is
  stale for this evidence.
- Use existing product device and stock `/v1/xiaozhi` route.
- Do not use generic firmware flash commands.

Files:

- Modify `internal/app/app_stackchan_touch.go` only if an acceptance case is
  missing.
- Modify `internal/app/*stackchan*_test.go` for report parsing if needed.
- Modify readiness collectors only after evidence schema is stable.
- Append `docs/agent_handoff_log.md`
- Modify `docs/project_state_machine.md`

Evidence commands:

```bash
go run ./cmd/a21 stackchan-accept --check touch --case screen_touch --gateway-url http://127.0.0.1:21080 --device-id 44:1b:f6:e2:6a:60 --output-dir reports
go run ./cmd/a21 stackchan-accept --check touch --case top_tap --gateway-url http://127.0.0.1:21080 --device-id 44:1b:f6:e2:6a:60 --output-dir reports
go run ./cmd/a21 stackchan-accept --check touch --case top_swipe_forward --gateway-url http://127.0.0.1:21080 --device-id 44:1b:f6:e2:6a:60 --output-dir reports
go run ./cmd/a21 stackchan-accept --check touch --case top_swipe_backward --gateway-url http://127.0.0.1:21080 --device-id 44:1b:f6:e2:6a:60 --output-dir reports
go run ./cmd/a21 stackchan-accept --check touch --case top_barge_in --gateway-url http://127.0.0.1:21080 --device-id 44:1b:f6:e2:6a:60 --output-dir reports
```

- [ ] Step 1: Confirm live device identity.

Run:

```bash
NO_PROXY=127.0.0.1,localhost curl -sS http://127.0.0.1:21080/v1/devices
```

Expected:

- Device `44:1b:f6:e2:6a:60` is online.
- Device route is stock `/v1/xiaozhi`.
- Device identity and capability map contain only A21/StackChan/Xiaozhi context
  strings permitted by AGENTS.

- [ ] Step 2: Run physical touch cases.

Expected:

- Each case creates a report under `reports/`.
- Each passed report includes `trace_id`, `session_id`, `device_id`.
- Failed reports remain useful and do not claim acceptance.

- [ ] Step 3: Run official action evidence.

Use the Gateway official control endpoint to send:

```json
{
  "device_id": "44:1b:f6:e2:6a:60",
  "events": [
    {"kind": "state", "name": "listening"},
    {"kind": "motion", "name": "nod"},
    {"kind": "motion", "name": "shake"},
    {"kind": "motion", "name": "dance"},
    {"kind": "state", "name": "idle"}
  ],
  "trace_id": "a21-trace-official-action-physical",
  "session_id": "a21-session-official-action-physical"
}
```

Expected:

- Gateway writes official frame packets to a connected official avatar socket,
  or returns a truthful blocked result if the socket is not connected.
- Operator or instrument evidence records screen/body/RGB response without raw
  audio/transcripts/secrets.

- [ ] Step 4: Update state and readiness only from passed evidence.

Rules:

- Touch cases can promote only their specific touch surface.
- Action packets can promote only the packet delivery/control surface unless
  the operator report confirms physical movement/display/RGB.
- Do not promote camera, NFC, infrared, battery, or IMU.

- [ ] Step 5: Verify.

Run:

```bash
git diff --check
make verify
```

Expected:

- All pass, or unrelated dirty failures are recorded.

Commit message:

```bash
git add docs/project_state_machine.md docs/agent_handoff_log.md reports
git commit -m "test(stackchan): record official action physical evidence"
```

## Task 6: Sensor And Battery Diagnostic Promotion Track

Transition id: `T-STACKCHAN-SENSOR-BATTERY-DIAGNOSTIC-001`

Boundary:

- Diagnostic first.
- No product behavior change until diagnostics pass.
- Firmware work requires a separate main-thread hardware window.
- No provider/V21 execution.

Files:

- Modify `internal/app/app_stackchan_mainline.go`
- Modify `internal/app/*stackchan*_test.go`
- Modify `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- Modify `docs/engineering/PROTOCOL.md`
- Modify firmware overlay only after main-thread approval.
- Append `docs/agent_handoff_log.md`

Diagnostic surfaces:

```text
imu
ambient_light
proximity
battery
```

- [ ] Step 1: Add report fields or confirm existing fields.

Expected report fields:

```json
{
  "sensor_available": true,
  "sensor_samples": 20,
  "sensor_read_errors": 0,
  "ambient_light_raw": 120,
  "proximity_raw": 7,
  "battery_mv": 4010,
  "battery_ma": -80,
  "trace_id": "a21-trace-sensor-diagnostic",
  "session_id": "a21-session-sensor-diagnostic",
  "device_id": "44:1b:f6:e2:6a:60"
}
```

- [ ] Step 2: Write readiness tests.

Test cases:

- `sensor_samples > 0` and `sensor_read_errors == 0` can mark diagnostic pass;
- a fixed, non-changing value is diagnostic only, not product accepted;
- battery `mv` without charging/current context is read-only status, not power
  management acceptance;
- missing fields leave the surface `planned_*`.

Run:

```bash
go test ./internal/app -run 'Test.*StackChan.*(Sensor|Battery|Mainline)' -count=1
```

Expected:

- Tests fail before report handling exists or pass if already implemented.

- [ ] Step 3: Implement report ingestion.

Implementation requirements:

- Preserve the existing ordered tracks.
- Promote from `planned_*` to `diagnostic_*` only when report evidence passes.
- Product `available` requires a later physical/product behavior transition.

- [ ] Step 4: Verify.

Run:

```bash
go test ./internal/app -count=1
git diff --check
make verify
```

Expected:

- All pass.

Commit message:

```bash
git add internal/app/app_stackchan_mainline.go internal/app/*stackchan*_test.go docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md docs/engineering/PROTOCOL.md docs/agent_handoff_log.md
git commit -m "feat(stackchan): add sensor battery diagnostic promotion"
```

## Task 7: Audio, Wake, AEC, And Playback-Start Acceptance Closure

Transition id: `T-STACKCHAN-AUDIO-WAKE-PLAYBACK-PARITY-001`

Boundary:

- This is the highest-priority physical PRD path.
- Do not mix it with camera/video/NFC/IR.
- Keep accepted 3x gain frozen unless the main thread explicitly opens an
  audio-quality experiment.
- Firmware changes require guarded product artifact build and foreground flash.

Files:

- Modify `internal/gateway/server.go`
- Modify `internal/gateway/server_test.go`
- Modify relevant `internal/app/*xiaozhi*` or `internal/app/*stackchan*` files.
- Modify `docs/engineering/PROTOCOL.md`
- Modify `docs/engineering/A21_CURRENT_CONTROL.md`
- Modify `docs/project_state_machine.md`
- Append `docs/agent_handoff_log.md`
- Modify firmware overlay only if playback-start/stop evidence cannot be
  collected through existing stock events.

Acceptance targets:

```text
wake_from_idle
listen_start
opus_ingress
asr_final_or_streaming_final
llm_first_content
tts_first_audio
opus_downlink
trusted_playback_start_or_operator_audible
barge_in_or_stop_done
idle_recovery
```

- [ ] Step 1: Freeze current audio evidence before changes.

Run:

```bash
go run ./cmd/a21 product-readiness --output-dir reports
go run ./cmd/a21 server-side-readiness --output-dir reports
```

Expected:

- Reports show provider, V21, and host voice state accurately.
- Physical PRD remains blocked until playback/audible/barge-in evidence is
  fresh and device-specific.

- [ ] Step 2: Collect wake and physical turn evidence.

Operator phrases:

```text
Zi Yue
Zi Yue Zi Yue
Ni Hao Zi Yue
Xiao Zi Yue
```

Expected:

- At least one wake phrase from idle produces a stock Xiaozhi trace with the
  acceptance targets listed above.
- Touch-initiated evidence is allowed as a fallback report but cannot close
  wake acceptance.

- [ ] Step 3: Add playback-start classification if needed.

If existing reports can prove trusted audible playback, update the readiness
classifier only.

If existing reports cannot prove playback start, plan a separate firmware
debug overlay for a stock-compatible playback-start marker. That overlay must:

- be documented as diagnostic or product-safe before build;
- keep provider keys out of firmware;
- use the official-compatible product artifact name;
- preserve no-welcome behavior;
- keep flash guarded by confirmation token and board/artifact checks.

- [ ] Step 4: Verify.

Run:

```bash
go test ./internal/gateway ./internal/app -count=1
git diff --check
make verify
make preflight
make doctor
```

Expected:

- All pass except known documented host warnings.
- Product readiness still blocks honestly if physical evidence is incomplete.

Commit message:

```bash
git add internal/gateway/server.go internal/gateway/server_test.go internal/app docs/engineering/PROTOCOL.md docs/engineering/A21_CURRENT_CONTROL.md docs/project_state_machine.md docs/agent_handoff_log.md reports
git commit -m "test(stackchan): close audio wake playback evidence gate"
```

## Task 8: Official App Lifecycle And No-Welcome Reconciliation

Transition id: `T-STACKCHAN-OFFICIAL-APP-LIFECYCLE-PARITY-001`

Boundary:

- Firmware/app lifecycle spike first.
- No product flash until source review, build report, and flash plan pass.
- Do not regress the current no-welcome direct Xiaozhi path.
- Do not make A21 a generic Mooncake app clone.

Files:

- Modify `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- Modify `internal/app/official_stackchan_test.go`
- Modify `docs/engineering/PROTOCOL.md`
- Modify `docs/engineering/A21_CURRENT_CONTROL.md`
- Modify `docs/project_state_machine.md`
- Append `docs/agent_handoff_log.md`

Question to answer:

```text
Can A21 preserve official StackChan AppLauncher/AppAiAgent/AppAvatar/AppDance/AppSetup status surfaces while still entering the stock Xiaozhi runtime directly and avoiding the welcome/setup trap?
```

- [ ] Step 1: Source-read official lifecycle.

Inspect:

```bash
rg -n "AppLauncher|AppAiAgent|AppAvatar|AppDance|AppSetup|startXiaozhi|home_indicator|status_bar|welcome|setup" \
  "/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan/firmware/main"
```

Expected:

- Worker records which official app lifecycle pieces are status/UI only and
  which mutate network/device setup.

- [ ] Step 2: Write failing build/guard tests.

Test cases:

- official-compatible product patch still calls `GetHAL().startXiaozhi()`;
- no code path falls into setup/uninstall welcome trap after Xiaozhi start;
- status bar/home indicator initialization remains possible when safe;
- artifact remains `a21-stackchan-official-xiaozhi-compatible.bin`;
- generic `xiaozhi.bin` product flash remains rejected.

Run:

```bash
go test ./internal/app -run 'Test.*Official.*Xiaozhi.*(Lifecycle|Welcome|Flash|Artifact)' -count=1
```

Expected:

- Tests fail before lifecycle guard exists.

- [ ] Step 3: Implement the smallest safe lifecycle restoration.

Allowed restoration:

- status bar/home indicator,
- avatar/action UI surfaces,
- dance/action handler if it does not require app center setup,
- read-only device status display.

Forbidden restoration in this transition:

- setup wizard,
- app uninstall/install flow,
- provider credentials in firmware,
- broad mobile app center behavior,
- unguarded OTA or flash behavior.

- [ ] Step 4: Build without flashing.

Run the existing guarded official-compatible build command used by the repo.
The worker must record the exact command from `go run ./cmd/a21 --help` or
existing official-stackchan app tests before executing it.

Expected:

- Build report passes.
- Artifact name is `a21-stackchan-official-xiaozhi-compatible.bin`.
- No flash execute is run in this worker unless the main thread explicitly
  authorizes it after reviewing the build report.

- [ ] Step 5: Verify.

Run:

```bash
go test ./internal/app -count=1
git diff --check
make verify
```

Expected:

- All pass.

Commit message:

```bash
git add firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch internal/app/official_stackchan_test.go docs/engineering/PROTOCOL.md docs/engineering/A21_CURRENT_CONTROL.md docs/project_state_machine.md docs/agent_handoff_log.md
git commit -m "feat(firmware): reconcile official stackchan lifecycle"
```

## Task 9: Camera, Video, NFC, And Infrared Spikes

Transition id: `T-STACKCHAN-HIGH-RISK-HARDWARE-SPIKES-001`

Boundary:

- Spikes only.
- No product availability promotion.
- No raw images, video frames, faces, NFC payloads, IR codes, credentials, or
  private environment details in repo reports.
- Camera/video require an explicit privacy posture before any runtime command.

Files:

- Create or modify a dedicated spike plan under `docs/plans/` for each surface
  that main control approves.
- Modify `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- Modify `docs/engineering/PROTOCOL.md`
- Append `docs/agent_handoff_log.md`

Spikes:

```text
camera_photo_status_only
camera_stream_privacy_guard
nfc_presence_without_payload
infrared_send_receive_guard
```

- [ ] Step 1: Create separate per-surface spike plan.

Each spike plan must include:

- product reason,
- privacy/safety policy,
- data retention policy,
- exact commands,
- acceptance evidence,
- rollback path.

- [ ] Step 2: Keep charter honest.

All four surfaces remain `planned_*` or `diagnostic_*` until physical and
privacy evidence passes.

- [ ] Step 3: Verify docs.

Run:

```bash
git diff --check
make verify
```

Expected:

- Both pass.

Commit message:

```bash
git add docs/plans docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md docs/engineering/PROTOCOL.md docs/agent_handoff_log.md
git commit -m "docs(stackchan): scope high risk hardware spikes"
```

## Dispatch Order

Recommended main-thread order:

1. Dispatch Task 1 immediately as a docs-only worker.
2. Dispatch Task 2 after Task 1 lands; it is the safest functional parity win.
3. Dispatch Task 3 after Task 2; status display needs the MCP/status vocabulary.
4. Dispatch Task 4 after Task 3; action mapping should consume stable display
   states.
5. Dispatch Task 5 only in an approved hardware window.
6. Dispatch Task 7 in parallel with Task 5 only if the operator is ready for
   physical wake/playback/barge-in testing; otherwise keep it next in line.
7. Dispatch Task 6 after low-risk status controls land; it is diagnostic and
   should not block the immediate PRD voice gate.
8. Dispatch Task 8 after wake/playback evidence is stable or after the main
   thread decides official UI parity is more urgent than PRD acceptance.
9. Dispatch Task 9 only after main control approves privacy/product scope for
   each high-risk surface.

## Global Verification Gate

Run before marking the whole parity program closed:

```bash
go test ./internal/transport/stackchan ./internal/protocol ./internal/gateway ./internal/app -count=1
git diff --check
make verify
make preflight
make doctor
```

Expected:

- Tests pass.
- Preflight/doctor warnings are documented and do not include product flash
  guard failures, namespace pollution, provider key leakage, or direct-connect
  proxy violations.
- Product readiness remains blocked until physical evidence is genuinely
  complete.

## Main Thread Prompt

Use this prompt when handing the plan back to the main control thread:

```text
Main control thread, please schedule T-STACKCHAN-OFFICIAL-HARDWARE-PARITY-001.

Detailed plan:
docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md

Recommended first worker:
T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001, docs-only/read-only against official source trees.

Worker boundary:
No code implementation, no provider execution, no Gateway start, no firmware build, no flash, no serial, no NVS write. Edits limited to docs/engineering capability/current-control docs, project_state_machine, and agent_handoff_log.

Required summary format:
Transition / What changed / Files changed / Tests run and results / Runtime or physical evidence / Deviations from plan / Remaining issues / Next suggested action / Forbidden actions avoided.

After Task 1 lands, schedule the low-risk Gateway MCP/status worker before any firmware or high-risk hardware work.
```
