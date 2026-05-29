# A21 Phase 5A Firmware Foundation Implementation Plan

**Goal:** Establish an isolated, version-controlled StackChan/CoreS3 firmware foundation with strict build and flashing discipline.

## Scope

Phase 5A implements:

- isolated local PlatformIO toolchain under `.a21-tools/`
- firmware release/build discipline documentation
- firmware manifest with A21 identity, board, version, and artifact rules
- validation script to prevent X21/V21 naming and wrong board IDs
- minimal PlatformIO firmware skeleton for M5Stack CoreS3 / StackChan
- build verification with `pio run` when dependencies resolve

Phase 5A does not implement:

- flashing/upload
- Wi-Fi credentials
- real WebSocket device client
- audio capture/playback
- servo/LED hardware control
- OTA

## Tasks

- [x] Install isolated PlatformIO.
- [x] Document firmware build and flashing discipline.
- [x] Add firmware manifest and validator.
- [x] Add minimal PlatformIO firmware skeleton.
- [x] Run manifest validator.
- [x] Run `pio run`.
- [x] Commit as `feat: add a21 firmware foundation`.
