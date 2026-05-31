# A21 Embedded Freeze Archive

Date: 2026-06-01.
Status: active freeze.
Owner: A21 control tower.

## Freeze Decision

All A21 embedded development flows are frozen. All embedded builds are paused.
The previous custom A21 StackChan firmware, protocol, and audio direction is no
longer the active P0 route.

A21 now treats xiaozhi firmware and the xiaozhi WebSocket protocol as the P0
hardware SDK and audio-front-end target. A21-owned work moves to a compatible
server that receives Opus from xiaozhi firmware, runs ASR/LLM/TTS, encodes
Opus, and sends xiaozhi-compatible media/control messages back.

## Frozen Workstreams

| Lane | Representative branch/worktree | Freeze status | Notes |
| --- | --- | --- | --- |
| Custom StackChan JSON/base64 PCM playback | `codex/a21-p0-voice-quality-integration-20260601`, `codex/a21-p0-tts-current-cutoff-mature-fix-20260601`, `codex/a21-p0-audio-current-cutoff-fix` | archived | Machine receipts passed while foreground hearing still failed with current cutoff and telegraph-like playback. |
| Official PCM bridge root/rescue | `codex/a21-p0-official-pcm-bridge-root-20260601`, `codex/a21-p0-official-pcm-bridge-5080-rescue-20260601` | archived | Retained as evidence only; this lane is no longer the P0 route. |
| ASR/TTS hardware smoke on custom protocol | `codex/a21-p0-asr-recognition-20260601`, voice-quality integration branch | archived | Useful for redaction and evidence-gate lessons, not active hardware architecture. |
| Custom streaming/barge-in pipeline | `codex/a21-p0-streaming-pipeline-bargein-20260601` | archived | Superseded by xiaozhi protocol compatibility and server-side pipeline design. |
| Historical hardware validation summaries | `codex/a21-hardware-full-validation-20260531`, `codex/a21-hardware-progress-summary` | evidence only | Retained for capability history and regression context. |

## Hard Stop Rules

- Do not run firmware builds for frozen lanes.
- Do not flash firmware, write NVS, open serial upload/monitor loops, or run raw
  upload tools for frozen lanes.
- Do not run PlatformIO, ESP-IDF, esptool, or device-control acceptance as a
  continuation of the frozen custom protocol.
- Do not promote old JSON/base64 PCM receipts, tail-drain fields, or custom
  device playback reports as M3 physical proof.
- Do not use X21 as the product architecture source. X21 remains historical
  reference only.

## Allowed Next Direction

The next active implementation lane is a xiaozhi-protocol-compatible A21 server
slice:

1. xiaozhi WebSocket handshake and session state.
2. `hello`, `listen`, `abort`, and binary Opus receive/send compatibility.
3. Opus decode to PCM and PCM encode to Opus.
4. ASR adapter boundary for local Paraformer or iFlytek WebSocket.
5. LLM provider boundary using the lab-proven low-latency provider set.
6. TTS adapter boundary for Matcha, Volcengine, and iFlytek by mode.
7. Barge-in and interrupt behavior measured through xiaozhi-compatible abort
   semantics.
8. Redacted observability that keeps transcript, raw audio, provider output,
   secrets, full URLs, and full local paths out of reports.

This archive does not implement that server. It freezes the wrong embedded
direction and records the new architectural boundary.
