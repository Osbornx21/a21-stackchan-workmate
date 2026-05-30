# A21 StackChan Hardware Capability Charter

## Purpose

StackChan is not a low-end edge endpoint. A21 treats it as an embodied robot surface with a strong CoreS3 base and robot body. Firmware remains thin for model, proxy, and V21 responsibilities, but it must not be thin in product expression.

A21 must either use a hardware capability well, keep it explicitly planned, or leave it out of the user-facing product. It must not ship a custom effect that is worse than the original StackChan experience.

## Capability Contract

Every firmware-originated `device.event` and `runtime.echo` carries a semantic `capabilities` map. This map is not acceptance proof; it is the device's honest declaration of what A21 knows about the hardware surface.

Current stable capability keys:

- `microphone`
- `speaker`
- `screen`
- `screen_touch`
- `top_touch`
- `servo_y`
- `servo_x`
- `rgb`
- `camera`
- `imu`
- `ambient_light`
- `proximity`
- `battery`
- `nfc`
- `infrared`

Status values must be literal and honest:

- `available`: implemented in the current product firmware and covered by tests or acceptance commands.
- `planned_*`: real StackChan hardware or product surface, but not yet product-ready in A21.
- `disabled_*`: intentionally disabled because the current implementation would be unsafe, unstable, or misleading.
- `diagnostic_*`: available only in a special diagnostic build and not part of the product firmware.

## Current Release Declaration

The current product firmware declares:

- `speaker`, `screen`, `screen_touch`, `top_touch`, `servo_y`, and `rgb` as `available`.
- `microphone` as `disabled_m5unified_i2s_stop_crash_guard` in the release build, with a separate diagnostic probe path.
- `imu` as `planned_9_axis_imu` in the release build, with a separate read-only diagnostic probe path.
- `servo_x`, `camera`, `ambient_light`, `proximity`, `battery`, `nfc`, and `infrared` as planned capabilities.

This prevents A21 from silently shrinking StackChan into a screen-plus-LED device while still avoiding false claims about unimplemented hardware.

## Product Gate

Before a planned capability can become `available`, it needs:

- a firmware unit or native test for its parser, state machine, clamp, or driver boundary;
- a Gateway or CLI acceptance surface when the capability leaves the device;
- an operator or instrumented evidence report for physical behavior;
- no leakage of provider keys, V21 internals, X21 names, or global proxy assumptions;
- product behavior that is at least not worse than the original StackChan capability and ideally clearly better for A21's desk-workmate experience.

If a capability fails this gate, keep it as `planned_*`, `diagnostic_*`, or `disabled_*`.

## Near-Term Hardware Tracks

- IMU: posture, bump/pickup detection, attention cues, and safe expression transitions.
- Ambient light and proximity: adaptive brightness, wake/sleep, and interaction distance.
- Battery: honest power state, local fallback behavior, and low-power expression.
- Second servo axis: coordinated head pose once mechanical range and safety clamps are verified.
- Camera: local presence/gesture or privacy-safe visual context, never silent surveillance.
- NFC/infrared: explicit opt-in desk interactions, not hidden automation.

## Hardware Mainline Command

Use this command before starting any planned-hardware driver work:

```bash
go run ./cmd/a21 stackchan-hardware-mainline --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --output-dir reports
```

The report schema is `a21.stackchan_hardware_mainline.v1`.

The command:

- reads the current Gateway `/v1/devices` report with direct no-proxy LAN access;
- confirms the intended StackChan device is online and declares every planned hardware key;
- emits an ordered hardware track list;
- keeps `dry_run=true`, `flash_allowed=false`, and `delete_allowed=false`;
- writes `reports/a21-stackchan-hardware-mainline-*.json`.

The ordered track list is:

1. `imu`: read-only diagnostic probe.
2. `ambient_light`: read-only diagnostic probe.
3. `proximity`: read-only diagnostic probe.
4. `battery`: read-only diagnostic probe.
5. `servo_x`: motion safety spike.
6. `camera`: privacy-safe vision spike.
7. `nfc`: explicit opt-in interaction spike.
8. `infrared`: explicit opt-in interaction spike.

This report is not proof that a planned capability is product-ready. It is the gate that prevents A21 from forgetting the hardware surface while preserving firmware build and flash discipline.

IMU diagnostic discipline:

- default release firmware must keep `imu=planned_9_axis_imu`;
- `a21_stackchan_cores3_imu_probe` is the only current IMU diagnostic build;
- `make firmware-imu-probe-build` must pass before any physical IMU test window;
- `make firmware-imu-probe-upload-blocker-check` must continue proving raw PlatformIO uploads are blocked;
- IMU runtime echo is telemetry, not product promotion.
