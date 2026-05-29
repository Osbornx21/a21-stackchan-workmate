# ADR 0004: Firmware WebSocket Client Library

## Status

Accepted.

## Context

A21 StackChan firmware needs a Gateway control WebSocket before audio transport. The project principle is to use mature infrastructure rather than hand-roll WebSocket framing on ESP32-S3.

PlatformIO registry inspection on 2026-05-30 showed `links2004/WebSockets` as a public Arduino-compatible WebSocket client/server library based on RFC6455, with version `2.7.3`, published 2026-01-07, repository `https://github.com/Links2004/arduinoWebSockets.git`, and LGPL-2.1 license.

## Decision

Use `links2004/WebSockets @ 2.7.3` for the StackChan/CoreS3 WebSocket client path.

The library is wrapped behind `A21GatewayWSDriver` in `a21_firmware_gateway_ws.h`. A21 runtime state and protocol parsing stay in A21 code; library types do not leak into tests, protocol headers, or Gateway semantics.

## Consequences

- Firmware builds remain pinned and reproducible.
- Native tests use a fake driver and do not depend on the Arduino WebSocket library.
- If the library is insufficient for future low-latency audio, A21 can replace the driver adapter without changing the state machine or protocol parser.
