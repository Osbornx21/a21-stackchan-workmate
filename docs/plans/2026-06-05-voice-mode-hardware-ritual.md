# Voice Mode Hardware Ritual

Status: deployed; physical acceptance pending.
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

## Deployment Evidence

- Commit `f19b7d9 feat(gateway): add voice mode hardware ritual` added
  `/v1/voice-mode-ritual` and `/workspace` roleplay/professional ritual
  controls.
- Commit `a5d9b9d fix(gateway): pace voice mode rituals` reused the existing
  body-scene pacing policy so the four-step screen/RGB/head ritual is visible
  instead of emitted as a near-instant burst.
- Local red/green evidence: before implementation,
  `TestVoiceModeRitualProfessionalSendsHardwareSequence` missed
  `step_delay_ms`; after implementation the focused Gateway tests and
  `GOMAXPROCS=2 make verify` passed.
- ECS `47.103.57.217` is deployed at `a5d9b9d`; remote focused Gateway tests,
  remote Go build, systemd restart, loopback `/healthz`, and public direct
  `/healthz` passed.
- Live professional trace
  `a21-trace-mode-ritual-professional-paced-a5d9b9d-202606050330` returned
  `step_delay_ms=180`, `total_planned_delay_ms=540`, and trace
  `summary.last_offset_ms=541`.
- Live roleplay trace
  `a21-trace-mode-ritual-roleplay-paced-a5d9b9d-202606050331` returned
  `step_delay_ms=180`, `total_planned_delay_ms=540`, and trace
  `summary.last_offset_ms=541`.
- Final public `/v1/devices` check showed product device
  `44:1b:f6:e2:6a:60` online, `current_voice_mode=roleplay`,
  `screen_theme=auto`, `screen_brightness=58`, RGB `120/48/96`, head
  `yaw=0,pitch=24,speed=180`, and
  `voice_mode_ritual_physical_accepted=false`.

This transition is machine-readable product-socket evidence for the PRD mode
switch body feedback. It is not physical acceptance until the operator or an
instrument confirms the visible screen/RGB/head change.
