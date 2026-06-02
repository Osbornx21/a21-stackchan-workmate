# Stock Xiaozhi Protocol Cleanup Plan

Status: ready for scoped implementation worker.
Transition: `T-PROTOCOL-001`.
Date: 2026-06-02.

## Problem

The physical stock Xiaozhi firmware serial log
`reports/a21-stackchan-physical-wake-serial-20260602-2130.log` repeatedly shows
`Unknown message type: listen` while the device still enters `speaking` and
receives binary audio downlink. The strongest hypothesis is that A21 Gateway is
sending server-to-device `{"type":"listen", ...}` acknowledgement JSON frames
for device-originated `listen/start|detect|stop`, and stock firmware accepts
`listen` only as a device-to-server control message.

This is a stock protocol cleanliness issue, not proof of TTS, Opus, or speaker
failure by itself.

## Evidence And Source Locations

- Serial evidence:
  `/Users/jiyurun/Documents/New project/reports/a21-stackchan-physical-wake-serial-20260602-2130.log`
  shows repeated `Unknown message type: listen` around listening/speaking
  transitions.
- Gateway source:
  `internal/gateway/server.go` `handleXiaozhiText` writes
  `xiaozhiBaseReply(session, "listen", ...)` for `listen/start`, `detect`,
  ignored/suppressed start, and ignored stop.
- Gateway reply builder:
  `internal/gateway/server.go` `xiaozhiBaseReply` emits `type`, `state`,
  `status`, trace/session/device IDs, and optional `turn_id`.
- Stock-compatible transport parser:
  `internal/transport/xiaozhi/frame.go` accepts `listen` as parsed control
  input, but the current package does not distinguish device-to-server from
  server-to-device JSON message allowlists.
- Existing tests:
  `internal/gateway/server_test.go` currently expects or consumes these listen
  acks in many host harness flows, including turn ID, suppress/restart, and
  voice-pipeline tests.
- Stock/debug isolation:
  `internal/transport/xiaozhi/device_extension.go`,
  `internal/transport/xiaozhi/device_extension_test.go`, and
  `docs/engineering/PROTOCOL.md` already require `device_events`/debug
  extensions to stay out of stock profile.

## Implementation Worker Task

Create a scoped worker branch from the current control state and implement the
smallest stock-safe cleanup for server-to-device listen acknowledgements.

Allowed changes:

- `internal/gateway/server.go`
- focused `internal/gateway/server_test.go` updates
- focused `internal/transport/xiaozhi` builder/parser tests only if a
  direction-specific allowlist is added
- `docs/engineering/PROTOCOL.md` only to clarify that stock server-to-device
  JSON should not include `listen` acknowledgements
- `docs/agent_handoff_log.md`

Forbidden changes:

- No firmware edits, flash, NVS writes, hardware access, Mac audio playback, or
  Gateway start/stop.
- No provider or V21 execution.
- Do not add `device_events`, A21 debug fields, or `type=device` to stock
  hello/runtime.
- Do not remove `turn_id` tracing or turn ownership internally; if stock JSON
  acks are removed, preserve the host-side trace/state contract.

Suggested implementation shape:

1. Add or update a failing test that exercises a stock Xiaozhi WebSocket flow
   and asserts no server-to-device JSON frame has `type=listen` after
   `listen/start|detect|stop`. Keep `tts/start`, `tts/sentence_start`,
   `tts/stop`, `hello`, and binary Opus behavior intact.
2. Change Gateway so stock profile does not send `xiaozhiBaseReply(...,
   "listen", ...)` frames to the device. If a local harness still needs these
   acknowledgements, gate them behind an explicit debug profile or helper path,
   never behind stock hello.
3. Preserve tracing for `xiaozhi.listen.start`, `xiaozhi.listen.detect`,
   `xiaozhi.listen.stop`, ignored/suppressed states, `xiaozhi.turn.start`, and
   `turn_id` ownership.
4. Update host tests that were reading listen acks to wait for the next valid
   stock downstream message or inspect traces instead.

## Acceptance

- Focused tests prove stock `/v1/xiaozhi` no longer sends server-to-device
  `type=listen` JSON.
- Existing `tts/start`, `tts/sentence_start`, paced binary Opus downlink,
  `tts/stop`, abort, barge-in, and host voice-pipeline tests still pass.
- Debug-only playback/device event behavior remains gated by
  `features.device_events=true`; stock profile still rejects `type=device`.
- No protocol implementation change introduces X21/V21 naming or A21 debug
  fields into stock hello/runtime.
- `git diff --check` passes.

## Suggested Verification

Run, at minimum:

```bash
go test ./internal/transport/xiaozhi -count=1
go test ./internal/gateway -run 'Xiaozhi|xiaozhi|Stock|Playback|Turn|Abort|Barge|VoicePipeline' -count=1
git diff --check
```

If Go code changes are broader than the ack cleanup, also run:

```bash
make verify
```

## Risks

- Many current host tests consume listen acks as a convenient synchronization
  point; updating tests incorrectly could mask a real turn lifecycle regression.
- Removing acks without preserving internal turn ownership could break barge-in
  cancellation or trace evidence.
- Gating acks through `device_events` or debug profile must not leak that debug
  negotiation into stock server hello or physical stock runtime.
