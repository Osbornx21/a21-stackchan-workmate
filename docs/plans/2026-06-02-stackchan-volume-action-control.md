# A21 StackChan Volume And Action Control Plan

Date: 2026-06-02

## Background And Problem Definition

During `T-AUDIO-001` foreground recording, the operator reported that the
physical StackChan sound still seemed below maximum volume. A previous attempt
mistakenly set macOS output volume, but the active hardware path is a stock
Xiaozhi WebSocket device and StackChan speaker gain is not controlled by macOS.

The operator needs a local desktop helper that can verify computer-to-StackChan
control, and the project needs a real path to raise or set the physical
StackChan speaker output level.

## Current System State

- Physical device `44:1b:f6:e2:6a:60` connects to A21 Gateway `21081` through
  stock `/v1/xiaozhi`.
- The device reports `speaker=available_xiaozhi_opus_downlink` and Opus
  uplink/downlink capability.
- A21 Gateway currently has no runtime HTTP endpoint that sets stock Xiaozhi
  StackChan speaker volume.
- `/v1/xiaozhi/control` supports state/face/display/motion only when the live
  socket negotiated the A21 debug device-event profile; the current physical
  stock socket rejected face/motion with HTTP 409.
- The old `/v1/devices/control` diagnostic tone volume belongs to the legacy
  A21 audio WebSocket path. The current stock Xiaozhi device is not connected
  on that audio WebSocket.
- Existing firmware code/patches show code-level volume knobs:
  `M5.Speaker.setVolume(96)` in the old A21 firmware path, and official codec
  `SetOutputVolume(...)` in official StackChan overlay lanes.

## Target State

- A desktop helper on the operator machine can truthfully report:
  - Gateway/network health.
  - Online physical StackChan identity and active transport.
  - Whether StackChan runtime speaker-volume control is available.
  - Whether host-pushed action control is currently accepted or blocked.
- A future guarded implementation can set or raise physical StackChan speaker
  output through the active official Xiaozhi-compatible firmware path, with
  traceable build/flash/evidence.

## Not Doing

- Do not use macOS system volume as StackChan speaker-volume evidence.
- Do not claim `/v1/devices/control` diagnostic tone volume changes product
  TTS volume on the current stock Xiaozhi path.
- Do not flash firmware, write NVS, or add protocol extensions without an
  explicit foreground hardware transition.
- Do not weaken stock Xiaozhi compatibility by leaking debug-only fields into
  stock hello/runtime messages.
- Do not claim PRD audio acceptance from host-only or diagnostic-only evidence.

## Impact Scope

- Desktop helper script under `tools/desktop/` and copied to the operator
  Desktop.
- Future implementation may touch:
  - `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
  - Gateway xiaozhi device-control/MCP adapter surfaces, if runtime control is
    chosen instead of fixed firmware volume.
  - Firmware build/flash evidence commands and readiness reports.

## Phased Execution

### Phase 0: Truthful Desktop Helper

Create a local helper that checks Gateway/device state and reports that current
stock Xiaozhi runtime volume control is unavailable.

Acceptance:

- Running `status` shows Gateway health, device state, and the volume/action
  boundary.
- Running action probes reports delivered or the exact HTTP block reason.
- The helper does not set macOS volume or claim physical sound changes.

### Phase 1: Decide Volume Control Strategy

Choose exactly one path:

- Fixed firmware output level: patch official Xiaozhi-compatible firmware to
  call official codec `SetOutputVolume(...)` at a selected value before audio
  output starts.
- Runtime device tool: add a stock-safe, debug-gated control surface for
  speaker gain only if the current firmware/official stack exposes a mature
  supported setter.

Acceptance:

- Decision records why the selected strategy is safer than the alternative.
- The strategy preserves stock Xiaozhi audio/protocol behavior.
- The selected volume range and default are documented.

### Phase 2: Worker Implementation

Dispatch a worker with one scoped transition:

- If fixed firmware: modify only the official Xiaozhi-compatible overlay and
  related tests/docs.
- If runtime control: modify only Gateway/transport validation and firmware
  event handling needed for a debug-gated speaker-volume command.

Acceptance:

- Focused tests pass.
- No provider/V21 execution.
- No firmware flash in the worker unless separately approved.
- Stock profile remains clean.

### Phase 3: Foreground Hardware Validation

After review, perform guarded build/flash or runtime-control test in a
foreground hardware window.

Acceptance:

- The exact artifact, board, port, firmware identity, and command are recorded.
- Operator/instrument recording confirms audible loudness change.
- New phone recording is analyzed for LUFS, peak, active-frame ratio, and
  spectral balance.
- `docs/agent_handoff_log.md` and `docs/project_state_machine.md` are updated.

## Rollback Plan

- If fixed firmware volume causes clipping/distortion, revert the overlay value
  and rebuild/flash the last known candidate.
- If runtime volume control is rejected by stock behavior, remove the runtime
  control surface and keep only fixed firmware volume.
- If physical audio remains bad after volume increase, return to `T-AUDIO-001`
  RCA and classify TTS/model vs physical speaker/decode separately.

## Risks

- Higher speaker output can clip or excite enclosure resonance.
- Changing codec volume in the wrong firmware path may have no effect on the
  active stock Xiaozhi runtime.
- Debug device-event control may be visible in capability registry while the
  live socket still rejects host-pushed events.
- A diagnostic tone can sound louder than product TTS and mislead acceptance.
- Reflashing without the T7 foreground guard can damage project state.

## Manual Confirmations Needed

- Target volume value or trial range for official codec output.
- Whether to pursue fixed firmware volume first or runtime volume control.
- Approval for any firmware build/flash/NVS/hardware execution.
- Operator availability to record before/after samples from the same phone
  position and prompt.
