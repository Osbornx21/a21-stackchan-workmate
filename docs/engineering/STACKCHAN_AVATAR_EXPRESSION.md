# A21 StackChan Avatar Expression Baseline

## Purpose

A21 must not treat the StackChan screen as a small status terminal. The screen is the face. The firmware expression layer therefore uses an avatar-pose contract instead of hard-coding one-off drawing logic into `main.cpp`.

This document records the first implementation baseline and the external references that shape it.

## Reference Baseline

NIO describes NOMI as an emotional companion in its smart cockpit system. For A21, this means the face should communicate state and attention before it explains itself with text.

M5Stack's StackChan documentation already treats expression, head movement, RGB status, top touch gestures, sleep, and avatar mirroring as one coordinated product surface. The official StackChan docs also describe the mobile Avatar feature as controlling head movement plus eye and mouth position. A21 should follow that shape: state drives face, motion, light, and touch semantics together.

The open-source Stack-chan project lists face display, expressions, gaze, speech, add-ons, and servo control as core robot capabilities. The related `m5stack-avatar` library provides the mature firmware-level model A21 should align with: expression, eye-open ratio, gaze, breath, mouth-open ratio, lip sync, color palette, move, zoom, and rotation.

Sources:

- https://www.nio.io/et9
- https://www.nio.com/innovation
- https://docs.m5stack.com/en/StackChan
- https://github.com/stack-chan/stack-chan
- https://github.com/stack-chan/m5stack-avatar
- https://raw.githubusercontent.com/stack-chan/m5stack-avatar/master/src/Avatar.h
- https://raw.githubusercontent.com/stack-chan/m5stack-avatar/master/src/Expression.h

## A21 Contract

The firmware header `a21_firmware_display.h` exposes:

- `A21AvatarExpression`
- `A21FaceFrame`
- `a21FaceFrameForState(A21RenderState)`

`A21FaceFrame` intentionally mirrors the mature avatar-engine concepts rather than pixel geometry:

- `expression`
- `eye_open_ratio`
- `gaze_vertical`
- `gaze_horizontal`
- `mouth_open_ratio`
- `breath_ratio`
- `auto_blink`
- `show_status_label`

This keeps A21 compatible with a later `m5stack-avatar` adapter while allowing a simple fallback renderer during hardware bring-up.

## Current Firmware Rendering

The current CoreS3 firmware uses a minimal fallback renderer:

- draws a face first, not a text-only status page;
- keeps a visible A21 identity and firmware label;
- keeps state labels such as `PRO MODE` visible for acceptance;
- maps `SPEAKING` to an open mouth pose;
- maps `THINKING` to a doubt/up-gaze pose;
- maps `LISTENING` to a focused/closed-mouth pose;
- maps `PRO MODE` to a restrained, evidence-first neutral pose;
- leaves RGB and servo behavior in their existing dedicated modules.

This fallback must remain replaceable. It is not the long-term animation engine.

## Library Adoption Rule

Before replacing the fallback renderer with `m5stack-avatar`, A21 needs one isolated spike:

```bash
env PLATFORMIO_CORE_DIR="$PWD/.a21-tools/platformio-core" \
  .a21-tools/platformio-venv/bin/pio pkg search "M5Stack-Avatar"
```

The package registry currently exposes `meganetaaan/M5Stack-Avatar` version `0.10.0`. A production adoption must:

- pin the exact library version in `platformio.ini`;
- pass `make firmware-test`;
- pass `make firmware-build`;
- keep `make firmware-avatar-spike-build` passing;
- verify task stack and redraw behavior on CoreS3;
- preserve visible `PRO MODE`, `LOCAL`, and error labels;
- preserve A21 firmware build and flash discipline;
- be documented in this file or a short ADR.

Current spike target:

```bash
make firmware-avatar-spike-build
```

This uses the isolated `a21_stackchan_cores3_avatar_spike` PlatformIO environment, pins `meganetaaan/M5Stack-Avatar @ 0.10.0`, and compiles the A21 render-state-to-avatar-expression mapping against the library API. It is intentionally a build-only compatibility lane, not a release package and not a flash command.

## Product Guardrail

A21 expression work is complete only when the device feels present. Backend correctness is not enough:

- listening should look like attention;
- thinking should never look dead;
- speaking should be driven by audio energy or mouth-open ratio, not text length;
- professional mode should look calmer and more precise;
- failure should look recoverable, not alarming.
