# Voice Mode Hardware Ritual

Status: active local transition.
Date: 2026-06-05 CST.

## Goal

Make the PRD-required user-initiated A21 mode switch visible on the prototype:
roleplay and professional mode changes should have a small screen/RGB/servo
ritual over the existing safe Xiaozhi MCP product path.

## Transition

- Current state: `/v1/voice-modes` selects `roleplay` or `professional` and
  returns ritual metadata, but hardware does not visibly change unless a
  separate body command is run.
- Target state: `/v1/voice-mode-ritual` selects the mode and delivers a bounded
  screen/RGB/head MCP sequence; `/workspace` exposes roleplay/professional
  ritual buttons.
- Acceptance: focused tests prove the endpoint sends only approved MCP tools,
  records trace/registry evidence, does not execute provider/V21, and keeps
  `physical_accepted=false` until operator evidence.

## Boundaries

- No firmware build or flash.
- No NVS write.
- No provider or V21 execution.
- No camera/NFC/IR/reboot/OTA/app-lifecycle exposure.
- No internal-test3 voice/protocol rollback.
