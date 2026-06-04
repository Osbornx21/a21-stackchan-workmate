# StackChan Official MCP Speaker Volume Freeze

Date: 2026-06-04

Status: implemented and verified.

Transition: `T-STACKCHAN-OFFICIAL-MCP-SPEAKER-VOLUME-FREEZE-001`

## Current State

- A21 already has a dedicated `POST /v1/xiaozhi/speaker-volume` endpoint that
  sends the stock Xiaozhi MCP `tools/call` for
  `self.audio_speaker.set_volume`.
- The low-risk official MCP status/control endpoint
  `POST /v1/xiaozhi/mcp-control` currently allows device status and screen
  controls, but not speaker volume.
- The hardware capability charter marks MCP speaker volume as
  live-whitelisted, physically used for runtime volume, and ready to freeze.

## Target State

- `POST /v1/xiaozhi/mcp-control` also allows
  `self.audio_speaker.set_volume` with bounded `volume=0..100`.
- The dedicated `/v1/xiaozhi/speaker-volume` endpoint remains as a
  compatibility/product shortcut.
- Both paths keep A21 identity, trace/session/device IDs, redacted send
  markers, and no raw MCP response capture.

## Boundary Conditions

- No Gateway service start, provider/V21 execution, firmware build, flash,
  serial, NVS, ECS change, or physical hardware action.
- No raw MCP response bodies, screenshots, audio, transcripts, provider output,
  URLs, local paths, or credentials in responses, traces, reports, or docs.
- High-risk tools remain blocked before websocket write: reboot, firmware
  upgrade, camera/photo, screen snapshot, stream/video, NFC, infrared, and app
  lifecycle.

## Implementation Steps

1. Add `volume` to `XiaozhiMCPControlRequest`.
2. Extend the MCP whitelist helper to allow `self.audio_speaker.set_volume`
   with `volume=0..100` and no screen arguments.
3. Record the existing safe trace marker
   `xiaozhi.mcp.speaker_volume.sent` and safe device activity metadata.
4. Extend Gateway MCP tests to prove unified delivery and unsafe argument
   rejection.
5. Update protocol, observability, control, state-machine, and handoff docs.

## Acceptance

- Unified `/v1/xiaozhi/mcp-control` sends the official
  `self.audio_speaker.set_volume` MCP tool call with bounded volume.
- Illegal volume values, missing volume, or mixed screen/volume arguments are
  rejected before websocket write.
- Existing dedicated `/v1/xiaozhi/speaker-volume` behavior remains covered.
- Focused Gateway MCP tests, `git diff --check`, and `make verify` pass.

## Failure State

- A high-risk MCP tool becomes writable through the unified endpoint.
- Raw MCP result or private data appears in response/trace/doc evidence.
- Speaker volume claims physical loudness acceptance without a fresh hardware
  report.

## Rollback

- Remove `self.audio_speaker.set_volume` from the unified MCP whitelist while
  keeping the existing dedicated speaker-volume endpoint.

## Next State

- Official low-risk MCP control has a single consistent Gateway surface for
  status, screen, and frozen speaker volume. Physical speaker/playback
  acceptance still requires device evidence.

## Implementation Result

- Added `volume` to `XiaozhiMCPControlRequest`.
- Unified `/v1/xiaozhi/mcp-control` now allows
  `self.audio_speaker.set_volume` with bounded `volume=0..100`.
- Existing `/v1/xiaozhi/speaker-volume` behavior remains unchanged.
- Unified MCP control records `xiaozhi.mcp.speaker_volume.sent` and bounded
  `speaker_volume` activity metadata.
- Missing volume, out-of-range volume, and mixed screen/volume arguments are
  rejected before websocket write.

## Verification

- `go test ./internal/gateway -run 'TestXiaozhiSpeakerVolumeUsesStockMCPToolCall|TestXiaozhiMCPStatusParityAllowsOnlyScopedTools|TestXiaozhiMCPStatusParityBlocksHighRiskTools|TestXiaozhiMCPStatusParityRequiresSafeArguments' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
