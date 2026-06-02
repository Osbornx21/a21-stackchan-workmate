# T-PROVIDER-PLAYBACK-001 Stock Xiaozhi Relay WAV Playback Plan

Status: active plan
Date: 2026-06-03
Owner: A21 control tower
Transition: `T-PROVIDER-PLAYBACK-001-STOCK-XIAOZHI-RELAY-WAV-PLAYBACK`

## Background And Problem Definition

`T-PROVIDER-002b` proved the fastest real voice path through 5080 relay:
Iflytek TTS passed and StepFun `step-1-8k` plus Iflytek TTS produced a
candidate WAV. The next physical test must play that candidate through the
current stock Xiaozhi WebSocket path.

The existing `stackchan-local-tts-playback --wav` command posts to
`/v1/devices/control`, which requires the legacy A21 device audio WebSocket and
has already returned `409 device audio websocket is not connected` for the
current stock Xiaozhi firmware. It must not be used as stock Xiaozhi physical
acceptance evidence.

## Current System State

- Current baseline: commit `6eb8062`.
- Gateway `127.0.0.1:21081` is healthy.
- Device `44:1b:f6:e2:6a:60` is registered but stale.
- Relay WAV exists:
  `reports/provider-tts-candidate/a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav`.
- `/v1/xiaozhi/say` already sends stock TTS lifecycle and Opus binary downlink,
  but currently synthesizes from text instead of accepting a pre-generated WAV.

## Target State

`S-PROVIDER-TTS-REAL-DIALOGUE-CANDIDATE-RUNNING`

- `/v1/xiaozhi/say` can accept either text or a pre-generated local A21 WAV.
- WAV playback uses the same stock `tts/start`, `sentence_start`, Opus binary
  downlink, `tts/stop`, pacing, trace markers, official StackChan speaking
  relay, and post-say input suppression as text playback.
- Response/report surfaces do not expose the full WAV path, raw/base64 audio,
  transcript text, provider output, credentials, proxy values, or local paths.

## Non-Goals

- Do not change firmware, NVS, provider selection, accepted 3x gain, wake word,
  or half-duplex logic.
- Do not revive legacy `/v1/devices/control` as stock Xiaozhi evidence.
- Do not auto-play if the physical device is stale.
- Do not claim physical acceptance from a unit test or host-only WAV parse.

## Impact Scope

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

## Phased Execution

### Phase 1: Failing Gateway Test

Actions:

- Add a focused test that connects a stock Xiaozhi WebSocket, writes a small
  A21-compatible 16 kHz mono PCM WAV, posts `/v1/xiaozhi/say` with `wav_path`,
  and asserts:
  - stock `tts/start`;
  - stock `tts/sentence_start`;
  - one or more binary Opus downlink frames;
  - stock `tts/stop`;
  - response `delivered_transport=xiaozhi_ws`;
  - response records `audio_source=wav_file` or equivalent without full path.

Acceptance:

- The test fails before production code because `wav_path` is unsupported or
  because `text is required`.

### Phase 2: Minimal Gateway Implementation

Actions:

- Add optional `wav_path` to `XiaozhiSayRequest`.
- Require exactly one playable source: non-empty `text` or non-empty
  `wav_path`.
- For `wav_path`, read A21-compatible 16 kHz mono PCM WAV chunks as 60 ms
  `providers.VoiceAudioChunk` values and feed them through
  `writeXiaozhiOpusDownlink`.
- Use the existing TTS lifecycle and post-say suppression path.
- Return only safe metadata: source kind, basename, chunk count, trace/session,
  and no raw audio/path/transcript/provider text.

Acceptance:

- The new focused test passes.
- Existing `/v1/xiaozhi/say` text tests continue passing.

### Phase 3: Physical Playback Gate

Actions:

- Check Gateway health and `/v1/devices`.
- If device `44:1b:f6:e2:6a:60` is fresh/online, post `/v1/xiaozhi/say` with
  the relay WAV path and collect operator listening feedback.
- If the device is stale, block without sending playback.

Acceptance:

- Physical playback report/trace records stock Xiaozhi downlink chunks, or the
  blocker states the device was stale.

## Rollback

- Revert the Gateway `wav_path` support and tests.
- Keep `b5a405e` provider code, accepted 3x gain, firmware, NVS, wake package,
  and half-duplex state unchanged.

## Risks

- Sending 16 kHz Opus downlink to a stock path may differ from the accepted
  24 kHz text-TTS path. The implementation should reuse existing accepted
  `writeXiaozhiOpusDownlink` validation and trace the source clearly.
- Local path values must not leak into response bodies, docs, or reports beyond
  safe basenames.
- Device stale state can make the code path pass while physical acceptance is
  still blocked.

## Human Confirmation Points

- Operator must wake/reconnect StackChan before physical playback can count.
- Operator must accept or reject the real relay voice after hearing it through
  the physical StackChan speaker.

## Worker Execution Task

Worker owns only `T-PROVIDER-PLAYBACK-001`.

Allowed:

- Add focused tests and minimal Gateway implementation for `/v1/xiaozhi/say`
  `wav_path` support.
- Update protocol docs and state/handoff docs for this transition.
- Run focused Gateway tests and `git diff --check`.

Forbidden:

- Firmware flash or NVS writes.
- Provider/TTS adapter changes.
- Global proxy changes.
- Half-duplex or wake-word implementation changes.
- Physical playback unless Gateway reports the device online/fresh.

Return summary format:

- Test red result.
- Implementation summary.
- Files changed.
- Tests run.
- Whether physical playback was attempted.
- Report/trace path if produced.
- Remaining blocker and next action.
