# ADR 0003: Firmware JSON Parser And Native Tests

## Status

Accepted for Phase 5C.

## Context

A21 StackChan firmware must parse Gateway control envelopes without leaking provider logic, X21/V21 identities, or ad hoc string parsing into the device lane.

The parser must be small enough for ESP32-S3 firmware but testable without physical hardware.

## Decision

Use `bblanchon/ArduinoJson @ 7.4.3` for firmware JSON parsing and a PlatformIO `native` environment with Unity for parser unit tests.

Primary-source checks on 2026-05-30:

- PlatformIO ArduinoJson registry: https://registry.platformio.org/libraries/bblanchon/ArduinoJson
- ArduinoJson 7 deserialization tutorial: https://arduinojson.org/v7/tutorial/deserialization/
- ArduinoJson `deserializeJson()` API: https://arduinojson.org/v7/api/json/deserializejson/

Reasons:

- mature embedded C++ JSON library
- header-only and portable
- directly supports `JsonDocument` and `deserializeJson()`
- PlatformIO registry support
- avoids fragile custom JSON parsing on firmware
- native unit tests can run without flashing StackChan

## Consequences

Firmware protocol parsing now has a host-testable layer in `firmware/stackchan/include/a21_firmware_protocol.h`.

CoreS3 builds and native tests both pin their dependencies. New firmware protocol behavior should add a native test before being wired into the device runtime.
