# ADR 0005: Official StackChan PCM Bridge App Flash Gate

## Status

Draft. Not accepted.

`stackchan-official-pcm-bridge-flash-execute` remains T8 blocked. This ADR is
the precondition document for a future reviewed execute guard; it does not
authorize implementation, app flashing, NVS writes, provider execution, V21
execution, Gateway runtime, serial writes, raw uploads, or `/v1/devices/control`
traffic.

## Context

A21 is moving the M3 audio downlink path away from the rejected hand-written
M5Unified playback path and toward the official StackChan/CoreS3 codec/HAL
boundary. The current bridge app build lane can produce
`a21-stackchan-official-pcm-bridge.bin` from official StackChan Git `HEAD`,
apply the A21 PCM bridge overlay, and validate a no-write flash plan. The
current NVS lane can provision only `a21/device_id` and `a21/audio_ws_url` after
backing up and preserving the existing NVS partition.

The app flash is higher risk than the NVS-only lane because it writes the app
partition and changes the booted behavior of a physical StackChan. If this lane
is run from a stale branch, detached background worktree, wrong USB port, wrong
artifact, dirty tree, missing rollback package, or busy device state, A21 can
lose the known-good embodied baseline or create confusing evidence that mixes
official StackChan, A21 bridge, X21/V21 assumptions, and local hardware state.

## Decision

Keep `stackchan-official-pcm-bridge-flash-execute` classified as T8 until a
future accepted ADR and reviewed execute guard prove that it can be operated as
a foreground hardware-write lane.

No implementation should add the execute command until the guard consumes all
of the evidence listed below and writes a durable execution receipt. Raw
`pio upload`, raw `idf.py flash`, copied `esptool write_flash`, and generic app
flashing remain forbidden regardless of this ADR.

## Why The Lane Is Still T8 Blocked

- The current `stackchan-official-pcm-bridge-flash-plan` is deliberately
  no-write. It validates the app artifact and required flash parts but does not
  prove that a foreground hardware window is open.
- The accepted NVS lane only proves scoped NVS mutation. It does not prove that
  writing the app partition is safe.
- The bridge app depends on device-local NVS values. App flashing without a
  fresh NVS receipt can boot into a silent or misleading state.
- The bridge app changes the physical audio/runtime surface. A bad flash can
  regress speaker, screen, RGB, touch, servo, idle, or recovery behavior below
  the original StackChan baseline.
- Current control policy requires hardware writes to run only from a clean
  `codex/a21-hardware-window-*` branch in a foreground thread. Background
  `.codex/worktrees` and detached HEADs must be rejected.
- The existing control ledger records the app flash-execute lane as blocked
  until ADR, reviewed guard, and fresh verification are accepted.

## Evidence Required To Downgrade To T7

T7 means a real physical write is allowed only inside a foreground hardware
window. Downgrading this command from T8 to T7 requires all of the following:

- Accepted ADR replacing this draft, with explicit owner, scope, rollback
  policy, and stop rules.
- Reviewed execute guard implemented in A21 code, with tests proving it rejects
  detached HEAD, background `.codex/worktrees`, dirty worktrees, wrong branch
  patterns, missing confirmation token, missing port, missing artifact, missing
  NVS evidence, missing rollback package, and stale verification.
- Strict identity checks: A21 firmware identity, `device_id`, board
  `m5stack-cores3`, bridge app version, current Git commit, official StackChan
  source `HEAD`, overlay identity, app artifact name, app offset `0x20000`, and
  SHA-256 for every flash part.
- Strict upload-target checks: explicit USB serial-looking port, serial process
  ownership check, no auto-selected port, no Bluetooth/debug-console path, and
  operator-visible port in the execution receipt.
- Fresh NVS receipt proving partition `0x9000/0x4000` was backed up, parsed,
  regenerated, verified, and written only for `a21/device_id` plus
  `a21/audio_ws_url`, while preserving existing entries and reporting whether
  servo calibration was present.
- Rollback package containing the last known-good app and flash parts, hashes,
  flash offsets, recovery command plan, and the matching NVS backup receipt.
- Device readiness checks proving the device is idle, not speaking, not
  thinking, not in professional/local-fallback/error mode, has no active
  playback stream, is thermally safe by operator observation, has screen output
  visible, and has servo motion stopped or at a safe neutral position.
- Operator token distinct from the NVS token, for example a future
  `WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP`, and recorded exactly in the
  guard schema without accepting partial matches.
- Fresh control evidence from the same clean hardware-window branch:
  `make verify`, `go run ./cmd/a21 preflight`, `go run ./cmd/a21 doctor`, and
  `go run ./cmd/a21 control-guard --command 'stackchan-official-pcm-bridge-flash-execute'`.
- Execution receipt including `trace_id`, `session_id`, `device_id`, branch,
  commit, worktree path, dirty state, tier, command, artifact hashes, flash
  offsets, NVS receipt path, rollback package path, port, operator token status,
  start/end timestamps, and final device state.

## Evidence Required To Downgrade To T6

T6 is physical validation, not a write. Before any app flash execute exists,
A21 may introduce or run only no-write/read-only validation around this lane if
it proves:

- The command cannot write flash, NVS, serial, or Gateway device control.
- It reads existing receipts and device reports only, or prompts an operator to
  record visible state without sending control traffic.
- It validates that required T7 evidence would be present for a future hardware
  window, but still reports `flash_allowed=false`.
- It keeps provider/V21 execution out of scope and records only redacted
  websocket endpoint fields, never full URLs, credentials, proxy values,
  transcript text, provider output, or evidence bodies.
- It leaves the device in the same state it found it, or marks the validation
  failed if idle/safe state cannot be established without control traffic.

## Future Execute Guard Shape

A future execute guard should be a two-step gate:

1. Rebuild the no-write flash plan from the selected build directory and compare
   it with the NVS receipt, rollback package, current commit, and control-guard
   result.
2. Only after every guard passes, call repository-local esptool from the
   official StackChan build directory with the reviewed `flash_args` parts. The
   command must not shell out to raw copied snippets and must not rely on global
   PlatformIO or ambient upload settings.

The guard should fail closed. Missing evidence, mismatched hashes, stale
receipts, non-idle device state, unknown capability declarations, thermal or
servo uncertainty, or report redaction failure must keep the command blocked.

## Consequences

- A21 can keep the official codec/HAL bridge direction without normalizing a
  risky app flash path.
- The NVS-only lane remains useful evidence, but it is not enough to approve an
  app partition write.
- Future implementation work has a concrete checklist for moving the command to
  T7, while read-only physical validation can be designed separately as T6.
- The control tower can reject any proposed execute command that lacks ADR,
  guard, rollback, NVS, identity, upload-target, idle/safety, operator-token, and
  fresh verification evidence.
