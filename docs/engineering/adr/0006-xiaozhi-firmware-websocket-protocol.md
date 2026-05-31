# ADR 0006: Xiaozhi Firmware And WebSocket Protocol As A21 Hardware SDK

Status: accepted by user direction.
Date: 2026-06-01.
Scope: A21 P0 hardware/audio-front-end direction.

## Context

A21 repeatedly spent engineering effort on custom StackChan firmware, JSON/base64
PCM playback, local playback smoothing, firmware bridge planning, and physical
mic/speaker acceptance gates. The latest foreground validation showed that
machine-readable playback receipts could pass while the user still heard
obvious current cutoff and telegraph-like playback. That invalidates the custom
firmware/protocol path as the P0 product route.

xiaozhi firmware already covers the dirty and time-expensive embedded layer:
board support, codec drivers, I2S capture/playback, Opus framing, VAD, wake
word, AEC, provisioning, WebSocket client lifecycle, binary audio frames, and
device-control messages. A21 should consume that mature firmware/protocol layer
instead of rebuilding it.

## Decision

A21 will use xiaozhi firmware and the xiaozhi WebSocket protocol as the P0
hardware SDK and audio front end.

The A21 product core moves to a compatible server:

1. Receive xiaozhi WebSocket `hello`, `listen`, `abort`, and binary Opus media.
2. Decode Opus to PCM.
3. Run ASR through the selected A21 provider path, initially local Paraformer
   or iFlytek WebSocket by mode.
4. Run LLM through the selected A21 text provider, initially the lab-proven
   SiliconFlow/StepFun finalist set according to current provider evidence.
5. Run TTS through mode-specific providers: Matcha for fast local response,
   Volcengine for higher quality, or iFlytek for balanced pure-cloud fallback.
6. Encode TTS audio back to Opus.
7. Send xiaozhi-compatible binary audio and JSON control events back to the
   device.

StackChan remains valuable for avatar rendering, servo motion, and embodiment.
Those surfaces must integrate above or alongside the xiaozhi firmware/protocol
layer. They must not keep the custom A21 audio front end alive as the P0 path.

## Frozen Lanes

The following lanes are archived as historical evidence and must not continue
as active embedded implementation work:

- Custom A21 StackChan JSON/base64 PCM playback and host tail-drain smoothing.
- Official StackChan/CoreS3 PCM bridge root/rescue work.
- A21 custom ASR/TTS hardware-smoke branches that depend on the custom device
  audio protocol.
- P1 custom streaming/barge-in pipeline work built on the old A21 device-audio
  protocol.
- Any firmware build/flash/NVS/serial/upload flow attached to those lanes.

## Consequences

- No embedded build, flash, NVS, serial, PlatformIO, ESP-IDF, or esptool action
  is authorized for the frozen lanes.
- No future A21 P0 acceptance may treat JSON/base64 PCM playback receipts as
  product proof.
- Next implementation work must target a xiaozhi-protocol-compatible A21 server
  spike, not another firmware playback patch.
- Provider, ASR, LLM, TTS, personality, V21 integration, multimodal behavior,
  observability, and session policy remain A21-owned server concerns.
- xiaozhi protocol compatibility is not yet implemented or proven by this ADR.
  A later spike must verify the handshake, binary Opus receive path, abort
  behavior, TTS return audio, and barge-in timing against a real device.

## Non-Goals

- This ADR does not approve firmware flashing.
- This ADR does not bless a provider default change.
- This ADR does not merge xiaozhi source into A21.
- This ADR does not claim M3 physical proof.
- This ADR does not remove existing historical reports, branches, or evidence.
