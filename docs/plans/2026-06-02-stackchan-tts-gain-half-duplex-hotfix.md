# 2026-06-02 StackChan TTS Gain And Half-Duplex Hotfix

## Background And Problem Definition

The stock Xiaozhi audio parity hotfix improved physical sound quality and
reduced electrical noise, but the physical StackChan output is still not loud
enough at runtime volume `100`.

Operator recording evidence:

- `7.m4a` from 3 seconds: `-35.56 LUFS`, `-14.65 dBTP`.
- `8.m4a` from 3 seconds: `-31.78 LUFS`, `-10.56 dBTP`.
- `9.m4a` from 3 seconds: `-36.99 LUFS`, `-16.05 dBTP`.
- `9.m4a` best early 10-second window from 3 seconds:
  `-34.79 LUFS`, `-16.05 dBTP`.
- The first boosted physical retest
  `/Users/jiyurun/Downloads/纳仕张江国际社区云庐B区.m4a` from 3 seconds measured
  `-21.65 LUFS`, `-4.72 dBTP`, with no obvious clipping in FFmpeg `astats`.
- The follow-up operator recording
  `/Users/jiyurun/Downloads/浦东新区第二中心小学(申江校区) 3.m4a` from 3 seconds
  measured `-26.4 LUFS`, `-9.0 dBFS` true peak, and the operator reported
  subtle electrical interruption and reduced clarity on the 4x gain candidate.
  The current direction is to roll the Gateway max-gain cap back to 3x.

This points away from protocol mismatch as the primary remaining issue and
toward conservative TTS/downlink gain staging. The live long-TTS path also
showed that speaker playback can be captured again by the microphone, so
half-duplex input suppression must be handled as a separate follow-up.

## Current System State

- Gateway is running on `127.0.0.1:21081` in tmux session
  `a21-gateway-21081`.
- Physical device `44:1b:f6:e2:6a:60` reconnects to stock `/v1/xiaozhi` and
  advertises `hello.features.mcp=true`.
- `POST /v1/xiaozhi/speaker-volume` can deliver stock MCP
  `self.audio_speaker.set_volume` with `volume=100`.
- `POST /v1/xiaozhi/say` can deliver foreground TTS lifecycle and Opus binary
  downlink on the live stock socket.
- `xiaozhiDownlinkPCM16` now applies bounded downlink leveling; the max-gain
  cap is being reduced from 4x to 3x after operator listening feedback.
- The 3x Gateway candidate was relaunched in tmux session
  `a21-gateway-21081`; trace `a21-trace-stackchan-volume-1780417211` delivered
  runtime volume `100`, and trace `a21-trace-stackchan-say-1780417217`
  delivered `text_chars=176`, `audio_chunks=579`,
  `answer_first_audio_total_ms=1656`, and host-say input suppression markers.

## Target State

- Gateway applies conservative, bounded PCM level normalization before Opus
  encode on Xiaozhi downlink TTS.
- Quiet non-silent TTS frames become materially louder.
- Full-scale or already-hot frames remain protected from clipping.
- Silence and tiny background noise are not amplified.
- The behavior is covered by focused tests and documented as a Gateway hotfix,
  not as physical PRD acceptance.
- Host-say foreground tests arm a short input-suppression window on stock
  physical devices to reduce immediate self-trigger from speaker-to-microphone
  echo.

## Non-Goals

- Do not change stock Xiaozhi protocol messages or firmware in this transition.
- Do not change provider credentials, V21 access, or runtime product-chain
  selection.
- Do not treat phone recording evidence as exact device SPL.
- Do not solve custom wake word in this transition.
- Do not claim half-duplex acceptance until a separate physical retest proves
  speaker playback no longer retriggers the voice pipeline.

## Impact Scope

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

## Phased Execution

### Phase 1: Objective Recording Analysis

- Analyze `9.m4a` with the same FFmpeg loudness windows used for `7.m4a` and
  `8.m4a`.
- Record the result in the handoff log and state machine.

Acceptance:

- LUFS/peak values for `9.m4a` are captured with concrete start/window values.

### Phase 2: Bounded Downlink Gain

- Replace pure headroom limiting with bounded level normalization:
  - leave silence and tiny noise unchanged;
  - lift non-silent quiet TTS frames toward a target peak;
  - cap maximum lift per frame at the current 3x candidate;
  - keep absolute headroom below clipping.

Acceptance:

- Test proves quiet TTS frames are boosted.
- Test proves tiny noise is not boosted.
- Test proves full-scale frames are still capped below headroom.
- Focused Xiaozhi Gateway tests pass.

### Phase 3: Physical Retest

- Restart Gateway from the updated tree.
- Reconnect the physical StackChan if needed.
- Deliver `volume=100`.
- Play long TTS through `/v1/xiaozhi/say`.
- Analyze the next operator recording against `8.m4a` and `9.m4a`.

Acceptance:

- Physical recording improves materially without clipping or harshness, or the
  remaining failure is explicitly routed to firmware codec gain or TTS model.
- For the 3x candidate, operator listening acceptance remains pending after
  trace `a21-trace-stackchan-say-1780417217`.

### Phase 4: Half-Duplex Follow-Up

- Add a short host-say-only suppression window for stock physical devices.
- If later normal dialogue still captures playback through the microphone,
  start a separate transition for broader post-TTS suppression/AEC/ducking.

Acceptance:

- Focused test proves immediate listen restart and speech Opus after host-say
  are ignored for stock physical devices.
- A later physical trace shows no unwanted voice-pipeline retrigger after
  host-say. Normal TTS playback remains a separate acceptance item.

## Rollback Plan

- Revert the Gateway PCM normalization change and keep the stock protocol and
  firmware unchanged.
- If boosted output is harsh or clipped in physical recording, reduce target
  peak or maximum gain further, then retest through `/v1/xiaozhi/say`.
- If physical loudness is still low after safe host gain, move to guarded
  firmware codec-gain/volume persistence rather than stacking more Gateway
  amplification.

## Risks

- Per-frame gain can create some level pumping if applied too aggressively.
- Boosting very quiet noise would worsen hiss; the noise gate must prevent this.
- Phone recordings measure the room path, not exact device electrical output.

## Human Confirmation Points

- Operator should keep phone position consistent for the next recording.
- Operator should report whether boosted TTS sounds harsh, clipped, or merely
  louder.
