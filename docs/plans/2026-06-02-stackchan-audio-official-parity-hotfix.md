# 2026-06-02 StackChan Audio Official Parity Hotfix

## Background And Problem Definition

The physical StackChan can connect to A21 over the stock Xiaozhi WebSocket path,
but operator recordings still sound weak, blurred, intermittent, and hard to
understand. The failed volume-92 A/B recording
`/Users/jiyurun/Downloads/军民公路259号 7.m4a` measured `-33.4 LUFS` and
`-14.3 dBFS`, which means volume alone is not accepted.

Parallel read-only inspections of official `xiaozhi-esp32`, StackChan codec
control, and the A21 Gateway found the most urgent deltas:

- Official client uplink hello uses Opus `16000 Hz` mono `60 ms`.
- Official server/downlink defaults and CoreS3 output path are `24000 Hz`.
- Stock firmware does not accept server-to-device `type:"listen"` replies.
- A21 rebuilt the downlink Opus encoder for every `60 ms` TTS chunk.
- A21 had no runtime speaker-volume command on the current stock
  `/v1/xiaozhi` session, despite official firmware exposing
  `self.audio_speaker.set_volume` through MCP.
- A21 also had no host-controlled physical long-TTS trigger on the live stock
  socket, forcing the operator to use wake/touch flow even when the task was
  only speaker/TTS A/B recording.

## Current System State

- Physical device `44:1b:f6:e2:6a:60` has connected to Gateway `21081` through
  stock `/v1/xiaozhi`.
- Firmware volume-92 candidate has been flashed, but physical audio acceptance
  failed.
- Previous local TTS downlink was forced to `48000 Hz`; an emergency intermediate
  patch briefly tested `16000 Hz`, but official parity points to `24000 Hz`
  downlink.
- Gateway action control still requires debug device-event negotiation for
  face/motion/display, which is separate from stock audio playback.

## Target State

- Gateway stock server hello advertises `24000 Hz` downlink audio.
- Local/mock TTS downlink chunks are `pcm_s16le`, `24000 Hz`, mono, `60 ms`.
- Stock physical devices no longer receive unsupported `listen` replies.
- A single downlink Opus encoder is reused for contiguous frames in a turn.
- Pacing avoids an avoidable multi-frame burst at the start of TTS.
- Gateway exposes a stock-safe runtime speaker-volume endpoint that sends the
  official MCP tool call to the live Xiaozhi WebSocket.
- Gateway exposes a foreground host-say endpoint that sends stock TTS lifecycle
  and binary Opus downlink to the live Xiaozhi WebSocket for physical speaker
  tests.
- Desktop helper can set StackChan volume and play foreground speaker-test text
  through those endpoints.

## Non-Goals

- Do not rewrite A21 into another stack.
- Do not change provider keys, V21 access, or firmware secrets.
- Do not claim physical PRD acceptance from host-only tests.
- Do not use macOS volume, legacy `/ws/audio`, or diagnostic tone as product
  Xiaozhi TTS evidence.
- Do not treat the host-say endpoint as normal product dialogue, wake-word
  activation, or PRD acceptance.
- Do not write NVS or flash firmware in this hotfix transition.

## Impact Scope

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/providers/voice_pipeline_adapters.go`
- `internal/providers/voice_pipeline_real_adapters_test.go`
- `internal/app/xiaozhi_voice_bench.go`
- `internal/app/app_test.go`
- `internal/transport/xiaozhi/builders.go`
- `internal/transport/xiaozhi/frame_test.go`
- `tools/desktop/a21-stackchan-control.command`
- `/Users/jiyurun/Desktop/A21-StackChan-Control.command`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

## Phased Execution

### Phase 1: Official Audio Contract Alignment

- Set server hello/downlink contract to `24000 Hz` while keeping client/uplink
  validation at `16000 Hz`.
- Set local and mock TTS chunks to `24000 Hz` mono `60 ms`.

Acceptance:

- Server hello tests expect `24000 Hz` downlink.
- Voice pipeline and local TTS tests expect `24000 Hz` downlink chunks.

### Phase 2: Stock Protocol Cleanup

- Suppress unsupported `listen` replies for stock physical MAC-address devices.
- Keep debug/virtual test behavior available for internal harnesses.

Acceptance:

- Unit test proves stock physical MAC devices do not receive listen replies.
- Existing abort/barge-in tests continue to pass.

### Phase 3: Playback Smoothness Risk Reduction

- Reuse a turn-level downlink Opus encoder for same-format contiguous TTS frames.
- Reduce initial downlink prebuffer from five frames to one frame.

Acceptance:

- Unit test proves same-format turn frames reuse the encoder and format changes
  rebuild it.
- Focused Gateway playback/barge-in tests pass.

### Phase 4: Runtime Speaker Volume

- Add `POST /v1/xiaozhi/speaker-volume`.
- Require live Xiaozhi socket and `hello.features.mcp=true`.
- Send `{"type":"mcp","payload":{...tools/call self.audio_speaker.set_volume}}`
  to the device WebSocket.
- Update the desktop helper `volume` command to call this endpoint.

Acceptance:

- Unit test proves the endpoint emits the official MCP tool call with the
  requested `volume`.
- Desktop helper syntax checks pass.

### Phase 5: Physical Retest

- Add `POST /v1/xiaozhi/say`.
- Store the live session with registered stock sockets so the endpoint uses the
  same write lock, turn cancellation, pacing, traces, and Opus downlink path as
  normal Gateway turns.
- Restart or relaunch Gateway with the hotfix.
- Set StackChan volume to `100` through `/v1/xiaozhi/speaker-volume`.
- Trigger one long real stock Xiaozhi TTS turn through `/v1/xiaozhi/say`.
- Record with the same phone position and analyze LUFS/peak/RMS/active ratio.

Acceptance:

- Unit test proves `/v1/xiaozhi/say` writes `tts/start`,
  `tts/sentence_start`, binary Opus, and `tts/stop` to the same stock socket.
- Physical recording shows clearer, less broken speech, or the failure is
  explicitly recorded as a remaining audio/TTS model problem.

## Rollback Plan

- Revert the Gateway/provider/transport patches if focused tests fail.
- Keep the already flashed firmware unchanged.
- If runtime volume command causes unexpected device behavior, stop using the
  endpoint and continue with the firmware volume-100 candidate plan.
- If 24k downlink worsens host or physical quality, revert local TTS/downlink to
  the last working host contract and record the evidence.

## Risks

- `gopus` still internally uses a 48 kHz frame-size interface; this hotfix
  removes per-frame encoder reset and aligns TTS output, but does not replace
  the Opus library.
- Stock MCP delivery is asynchronous; the endpoint can prove delivery, not
  device-side applied volume, unless a device response or physical recording is
  captured.
- Physical sound may still be dominated by the TTS model, speaker enclosure, or
  codec gain behavior after protocol cleanup.

## Human Confirmation Points

- Operator must approve any Gateway restart that would interrupt the current
  physical session.
- Operator must provide the next physical recording for acceptance.
- No firmware flash or NVS write occurs in this transition without explicit
  command approval.
