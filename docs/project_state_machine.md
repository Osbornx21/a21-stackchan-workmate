# A21 Project State Machine

Status: active state document.
Last updated: 2026-06-02.

This document records A21 as a set of explicit transitions. A conversation is an
execution surface; the repository state, plans, handoff log, tests, and evidence
are the project memory.

## Project State

Current total state: `S-HW-PHYSICAL-XIAOZHI-GATEWAY-DOWNLINK-CANDIDATE`

Active child transition: `T-HW-VOLUME-001`.

A21 has a Go-first Gateway/Core foundation, stock-compatible Xiaozhi transport,
official StackChan avatar/action relay, provider/V21 boundaries, a repo-carried
control workflow, and a freshly rebuilt official StackChan Xiaozhi-compatible
firmware candidate. The official-compatible candidate has been flashed in a
foreground hardware window, the latest firmware enters the official Xiaozhi
runtime directly, and the device has now connected to an A21 Gateway over the
stock Xiaozhi WebSocket path. A physical wake/turn produced Gateway uplink,
downlink, and barge-in candidate evidence. The user reports the audible sound
is still wrong and likely TTS-related. `T-AUDIO-001` Phase 1 host downlink
isolation is integrated in `xiaozhi-voice-bench` via decoded Opus
`downlink_audio_quality`, and a fresh host/Gateway run shows post-Opus answer
quality passing on the active A21 route. The current control focus remains
`T-HW-VOLUME-001`: raise or expose real StackChan-side speaker output
honestly. The fixed official codec output-volume candidate is now prepared in
the repo-owned Xiaozhi-compatible overlay. The no-write official-compatible
build/report passed with app SHA-256
`2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`.
The matching candidate was flashed through an operator-approved foreground
hardware window on `/dev/cu.usbmodem1101`; flash report
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`
is `status=passed`, `dry_run=false`, `flash_allowed=true`, and
`flash_executed=true`. Gateway `127.0.0.1:21081` remained healthy after flash
and the physical device `44:1b:f6:e2:6a:60` remained registered online. Physical
audible A/B remains pending. Explicit real/local provider evidence exists, but
generic readiness refreshes can still fall back to `mock` if they omit the
selected provider report/env. It is not yet full PRD accepted because audible
playback observation or trusted device playback ack and custom wake proof remain
missing.

Current control branch:

- `codex/a21-hardware-window-20260602-stackchan-prd`

Current notable baseline:

- `18dc553 docs(control): prepare stackchan volume flash gate`
- `f0603f5 fix(firmware): set official xiaozhi codec volume`
- `4613946 fix(firmware): enter official xiaozhi runtime directly`
- `eeacbd3 docs(control): recover hardware network state`
- `e7e9b03 feat(firmware): autostart official xiaozhi candidate`
- `37ef8f3 feat(firmware): add official xiaozhi nvs connection config`
- `6f34091 feat(firmware): add official xiaozhi compatible flash plan`
- `d28560a docs(control): dispatch stackchan volume worker`
- `f49abde docs(audio): plan stackchan volume control`
- `751de08 docs(audio): record stock xiaozhi playback boundary`
- `59f30f4 docs(control): record official flash seam worker dispatch`
- `69c4bbe docs(control): add handoff and state machine workflow`
- `987bbb0 feat(firmware): add official xiaozhi compatible stackchan build`

## Module States

| Module | State | Evidence | Next state |
| --- | --- | --- | --- |
| Control workflow | `S1-REPO-CARRIED-CONTROL` | Commit `69c4bbe`; `docs/agent_handoff_log.md`, `docs/project_state_machine.md`, and `docs/plans/` exist | `S2-WORKER-TRANSITION-OPERATING` |
| Gateway `/v1/xiaozhi` | `S3-PHYSICAL-DEVICE-CONNECTED` | Gateway on `127.0.0.1:21081` / LAN port `21081` accepted the physical device via stock Xiaozhi WebSocket; trace `a21-trace-44-1b-f6-e2-6a-60` has Opus uplink, VAD, ASR final, provider first content, TTS first audio, and Opus downlink | `S4-AUDIBLE-PLAYBACK-ACCEPTED` |
| Official StackChan avatar/action relay | `S2-HOST-READY` | Gateway/transport mapping exists for official StackChan packets | `S3-FLASHED-OFFICIAL-CANDIDATE` |
| Firmware candidate | `S5B-VOLUME-CANDIDATE-FLASHED` | Commit `f0603f5`; foreground flash report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`; app SHA-256 `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`; rollback baseline remains report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`, app SHA-256 `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d` | `S6-PRD-PHYSICAL-ACCEPTED` |
| Device connection/NVS | `S3-LAN-GATEWAY-CONNECTED` | Latest guarded NVS execution `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-212542-1780406742553040000.json` pointed OTA/WS to the LAN-bound A21 Gateway; reset serial log shows OTA connection to `21081` and activation | `S4-STABLE-RECONNECT-EVIDENCE` |
| Provider hot-plug | `S3-REAL-PROVIDER-EVIDENCE-READY` | Read-only worker `019e88dd-3efa-78e2-a7d3-7063089cbf30` confirmed usable explicit provider evidence: DeepSeek smoke `reports/a21-provider-smoke-20260602-112710-368364000.json` passed with `executed=true`, `stream=true`, `repeat=5`, `route_eligible=true`, and `first_content_p95_ms=745.516`; local Ollama smoke `reports/a21-provider-smoke-20260602-075644-199710000.json` passed when selected explicitly; readiness `reports/a21-product-readiness-20260602-160841.json` has `provider.real_provider_ready=true` and `server_side.provider_evidence_ready=true` | `S4-SELECTED-PROVIDER-READINESS-REFRESHED` |
| V21 adapter | `S2-HOST-READY` | Adapter contract exists; no firmware key or V21 internals should leak into A21 | `S3-PROFESSIONAL-EVIDENCE-RUN` |
| Memory/personality | `S1-IMPLEMENTED-HOST` | Host-side memory/personality work exists but needs current PRD burn-down refresh | `S2-READINESS-REVIEWED` |
| Physical StackChan acceptance | `S2-CANDIDATE-GATEWAY-DOWNLINK` | `reports/a21-xiaozhi-physical-evidence-20260602-213147.097784000.json` reports physical device online, stock profile, mic delivery ratio 1, answer first downlink 555 ms, and barge-in metrics; PRD accepted remains false | `S3-AUDIBLE-PLAYBACK-AND-PRD-ACCEPTED` |
| Xiaozhi audio/protocol | `S2-STOCK-OPUS-A21-GATEWAY-COMPAT-WARNING` | Physical path uses stock Xiaozhi profile and Opus uplink/downlink, but serial log `reports/a21-stackchan-physical-wake-serial-20260602-2130.log` shows repeated stock-firmware `Unknown message type: listen` warnings | `S3-STOCK-CLEAN-AUDIO-ISOLATED` |
| TTS/audio quality | `S1C-STOCK-XIAOZHI-OPERATOR-RECORDING-PENDING` | `reports/a21-xiaozhi-voice-bench-20260602-220325.719331000.json` passed host product-chain bench on `21080` with `sherpa_onnx_tts`, answer p95 397 ms, and decoded Opus `downlink_audio_quality=passed`; `reports/a21-xiaozhi-voice-bench-20260602-220344.175240000.json` passed a one-round host check on the physical LAN Gateway `21081`; on 2026-06-02 22:16 CST the foreground attempt to push long TTS through `stackchan-local-tts-playback` failed with Gateway `409 device audio websocket is not connected`, proving the old PCM control surface is not connected to the current stock Xiaozhi physical session; current physical evidence still lacks operator/instrument audible observation | `S2-TTS-VS-DOWNLINK-ROOT-CAUSE-ISOLATED` |
| StackChan volume/action control | `S3-FOREGROUND-FLASHED-A-B-PENDING` | Desktop helper `tools/desktop/a21-stackchan-control.command` and `/Users/jiyurun/Desktop/A21-StackChan-Control.command` report Gateway/device health but no current stock `/v1/xiaozhi` runtime speaker-volume setter; action probes were rejected with HTTP 409 debug-profile negotiation; legacy diagnostic tone was rejected with HTTP 409 audio WebSocket block. Worker thread `019e88c3-8fa7-7e53-8e46-ab3ff6e637b9` prepared a fixed official codec path in `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`: `Board::GetInstance().GetAudioCodec()->SetOutputVolume(92)` before `GetHAL().startXiaozhi()`. No-write build report `reports/a21-stackchan-official-baseline-20260602-225207-1780411927040145000.json` passed with app SHA-256 `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`; no-write flash plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-225600-1780412160509265000.json` was followed by foreground execute report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json` with `status=passed`, `dry_run=false`, `flash_executed=true`. No NVS write, Gateway runtime change, audio A/B acceptance, or PRD acceptance has occurred after the flash. | `S4-PHYSICAL-LOUDNESS-A-B-RECORDED` |

## Active Transition

### T-HW-VOLUME-001: StackChan Physical Speaker Volume Control

Current state:

- `S3-FOREGROUND-FLASHED-A-B-PENDING`

Target state:

- `S4-PHYSICAL-LOUDNESS-A-B-RECORDED`

Trigger:

- The user clarified the target is StackChan device loudness, not macOS system
  volume.
- The latest phone recording is stronger than the previous one but still has
  low sustained loudness and narrow active speech spectrum.
- Current stock `/v1/xiaozhi` has no Gateway runtime speaker-volume setter, so
  the fastest honest path is a guarded firmware-side output-gain candidate.
- Worker thread `019e88c3-8fa7-7e53-8e46-ab3ff6e637b9` has prepared the fixed
  official codec output-volume patch for main-thread review.
- Main-thread no-write build/report has passed for the patched official
  Xiaozhi-compatible candidate.
- Main-thread no-write flash-plan report is ready, and read-only flash-gate
  worker `019e88d6-1734-75d2-897b-aa2da3069885` confirmed the artifact hash,
  USB port, dry-run safety fields, and rollback baseline.
- The operator explicitly opened the foreground hardware window and supplied the
  guarded execute command with confirmation token.
- Main-thread flash execute passed with report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`.

Actions:

- Use `docs/plans/2026-06-02-stackchan-volume-action-control.md`.
- Keep the main thread as architecture/control only.
- Keep the integrated fixed official codec output-volume patch and focused
  guard test.
- Use no-write build report
  `reports/a21-stackchan-official-baseline-20260602-225207-1780411927040145000.json`
  as the candidate evidence for foreground hardware review.
- Use no-write flash-plan report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-225600-1780412160509265000.json`
  as the final pre-flash guard evidence.
- Use foreground flash execute report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`
  as the hardware-write evidence.
- Trigger a real stock Xiaozhi turn on StackChan, collect the operator's
  post-flash phone recording, and compare it against the previous phone sample.
- Convert the operator or instrument observation into a redacted sidecar, then
  regenerate `xiaozhi-physical-evidence` and product readiness.
- Do not add runtime volume protocol, do not use macOS volume as evidence, do
  not use legacy local audio playback, do not write NVS, do not execute
  provider/V21, and do
  not promote diagnostic tone or host-only evidence to PRD acceptance.

Acceptance conditions:

- The official Xiaozhi-compatible overlay explicitly sets official codec output
  volume before entering the Xiaozhi runtime.
- Existing `GetHAL().startXiaozhi()` behavior remains preserved.
- Focused guard/test or build-report evidence proves the volume setting exists
  in the candidate artifact path.
- A no-write official-compatible build/report passes before any flash is
  requested.
- A no-write flash plan is ready and keeps `flash_allowed=false` until the
  operator explicitly runs the guarded execute command with the confirmation
  token.
- Foreground flash execute passed with `dry_run=false`, `flash_allowed=true`,
  and `flash_executed=true`.
- Gateway health and physical device registry are checked after flash.
- Before/after audible A/B is recorded before physical loudness is accepted.
- `git diff --check` passes, and any touched Go guard tests pass.
- State and handoff docs record that physical before/after audibility remains
  unaccepted until foreground flash plus phone/instrument A/B.

Failure state:

- `F-HW-VOLUME-001-NO-OFFICIAL-CODEC-SEAM` if the worker cannot locate a safe
  official codec setting point.
- `F-HW-VOLUME-001-SCOPE-DRIFT` if runtime protocol, Gateway behavior,
  provider/V21, NVS, flash, or macOS audio is touched outside the plan.
- `F-HW-VOLUME-001-PHYSICAL-OVERCLAIM` if the code candidate is treated as
  product voice acceptance without foreground physical evidence.

Rollback path:

- Revert the overlay/test/doc patch if the candidate build or guard fails.
- Keep the currently flashed firmware and NVS route unchanged until an explicit
  hardware window approves flash.
- If a flashed volume trial sounds worse, flash back to the last accepted
  official-compatible app SHA recorded in the firmware flash reports.

Next state:

- `S4-PHYSICAL-LOUDNESS-A-B-RECORDED`

### T-AUDIO-001: Isolate Xiaozhi TTS Quality From Opus/Device Playback

Current state:

- `S1C-STOCK-XIAOZHI-OPERATOR-RECORDING-PENDING`

Target state:

- `S2-TTS-VS-DOWNLINK-ROOT-CAUSE-ISOLATED`

Trigger:

- Physical Gateway downlink is now proven as a candidate path, but the user
  reports that the audible sound is still wrong and likely TTS-related.
- Previous audio/TTS optimization commits are present, but they only prove
  host-side PCM/downlink guardrails, not physical audible quality.

Actions:

- Use `docs/plans/2026-06-02-a21-xiaozhi-tts-audio-quality-rca.md`.
- Keep the main thread as control tower.
- Dispatch a worker for host downlink objective isolation.
- Integrate the worker result so `xiaozhi-voice-bench` can report decoded
  Opus/downlink PCM quality without storing raw audio.
- Run host-only product-chain bench against current A21 Gateways to verify
  post-Opus quality before touching physical firmware or TTS profile routing.
- Avoid hardware writes, NVS writes, provider/V21 execution, unapproved Mac
  audio playback, and PRD overclaiming.
- After worker return, run a foreground physical A/B only if the operator
  approves any Gateway/TTS profile change.
- For the current stock Xiaozhi physical connection, do not use
  `stackchan-local-tts-playback` as evidence: it requires the legacy A21 audio
  WebSocket and returned `409 device audio websocket is not connected` during
  the 2026-06-02 foreground recording window.
- Collect the next physical audible sample by triggering a real stock Xiaozhi
  listen/audio turn on the device, or first create a separate planned
  transition for a stock-safe TTS injection seam.

Acceptance conditions:

- The project can classify the bad sound as TTS/model, Opus/downlink, firmware
  playback, or stock-control compatibility with concrete evidence.
- Host-only and physical evidence remain labeled separately.
- Any follow-up fix has focused tests and does not reintroduce the old PCM
  bridge as a product path.

Failure state:

- `F-AUDIO-001-UNISOLATED` if reports still cannot distinguish TTS generation
  from post-Opus/downlink quality.
- `F-AUDIO-001-PHYSICAL-OVERCLAIM` if host-only evidence is promoted to PRD
  acceptance.
- `F-AUDIO-001-SCOPE-DRIFT` if a worker touches firmware, NVS, provider/V21,
  or Mac audio outside the plan.

Rollback path:

- Keep the current `sherpa_onnx_tts` host-local route as baseline.
- Revert any host-only report additions if they destabilize tests.
- Do not mutate firmware/NVS in this transition, so hardware rollback should
  not be needed.

Next state:

- `S2-TTS-VS-DOWNLINK-ROOT-CAUSE-ISOLATED`

### T-HW-002: Recover Network/Relay And Collect Physical Evidence

Current state:

- `S-HW-PHYSICAL-XIAOZHI-GATEWAY-DOWNLINK-CANDIDATE`

Target state:

- `S3-PHYSICAL-VOICE-EVIDENCE`

Trigger:

- The official Xiaozhi-compatible candidate was flashed through the T7
  foreground guard.
- The latest firmware no longer enters the setup/QR or watchdog failure path.
- The old temporary relay returned `503`; a LAN-bound A21 Gateway on port
  `21081` was verified by `/healthz` and `/xiaozhi/ota/`.
- A foreground guarded NVS update pointed OTA/WS to the LAN Gateway and the
  device connected through stock Xiaozhi WebSocket after wake.

Actions:

- Complete audible playback or trusted device playback ack evidence for the
  physical Xiaozhi turn.
- Preserve the physical Gateway trace and serial evidence without storing raw
  audio or transcript bodies.
- Close real provider smoke on the host side, then rerun readiness.
- Keep custom wake as blocked until guarded wake firmware flash and physical
  custom wake proof are recorded.

Acceptance conditions:

- Device connects to A21 Gateway using stock-compatible Xiaozhi protocol.
- Real microphone input, Gateway downlink, barge-in stop, official
  avatar/action, wake behavior, and provider rotation evidence are recorded
  without leaking keys or debug-only protocol fields.
- Audible playback observation or trusted device playback ack is present.
- Host/mock/candidate evidence remains labeled separately from physical
  acceptance.

Failure state:

- `F-HW-002-STALE-RELAY` if the recorded temporary relay no longer resolves or
  forwards OTA/WS.
- `F-HW-002-WIFI-NOT-CONFIGURED` if the device remains in AP config mode.
- `F-HW-002-UNCONFIRMED-WRITE` if any worker attempts background NVS/flash
  writes.
- `F-HW-002-NO-PHYSICAL-EVIDENCE` if the device connects but evidence is not
  recorded.
- `F-HW-002-AUDIBLE-ACK-MISSING` if Gateway downlink exists but physical
  audible playback or trusted device playback ack remains unproven.

Rollback path:

- Preserve boot/NVS evidence before changing connection settings.
- Restore the previous known-good official StackChan package through a guarded
  foreground flash path if the A21 candidate must be reverted.

Next state:

- `S3-PHYSICAL-VOICE-EVIDENCE`

## Completed Transitions

| Transition | Result | Notes |
| --- | --- | --- |
| T-FW-001: Freeze external X21 Xiaozhi builds | Completed | Commit `f7c95f0`; protects A21 from consuming external X21 Xiaozhi build dirs. |
| T-GW-001: Sync Xiaozhi turns to official StackChan | Completed | Commit `a36206f`; supports official StackChan turn synchronization. |
| T-GW-002: Relay official StackChan avatar actions | Completed | Commit `ea51c67`; maps Gateway state/action to official StackChan relay. |
| T-TR-001: Map A21 events to official StackChan frames | Completed | Commit `cdabe89`; keeps screen/action relay on official StackChan packet shapes. |
| T-FW-002: Add official Xiaozhi-compatible StackChan build | Completed host/build candidate | Commit `987bbb0`; candidate build lane exists, physical flash still pending. |
| T-GOV-001: Establish repo-carried workflow state | Completed | Commit `69c4bbe`; adds handoff log, state machine, and plan discipline. |
| T-VERIFY-001: Integrated host verification after governance merge | Completed | `make verify` passed; mainline official candidate rebuild passed with app SHA-256 `053d3ba0d0c8690898967337a02bce8d3fd957899ebdabe4d9f4ae1e2b28c80d`. |
| T-FW-004: Add official compatible candidate flash seam | Completed | Commit `6f34091`; no-write plan passed for `/dev/cu.usbmodem1101` with `dry_run=true`, `flash_allowed=false`, and app offset `0x20000`. |
| T-FW-005: Add official compatible NVS connection config | Completed | Commit `37ef8f3`; T7 NVS write report `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-204515-1780404315388792000.json` preserved servo calibration and Wi-Fi credentials while updating OTA/WS route. |
| T-FW-006: Autostart official Xiaozhi candidate | Superseded | Commit `e7e9b03`; initial autostart removed setup gate but hit a setup-uninstall watchdog path during field testing. |
| T-FW-007: Enter official Xiaozhi runtime directly | Completed | Commit `4613946`; latest app SHA-256 `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d`; flashed through report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`. |
| T-GOV-002: Recover hardware network state docs | Completed | Commit `eeacbd3`; reconciled collapsed control thread, active plan, state machine, and handoff log before foreground NVS execution. |
| T-HW-002a: Refresh connection route and prove physical Gateway downlink | Completed candidate | Latest NVS execution `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-212542-1780406742553040000.json`; serial logs `reports/a21-stackchan-direct-xiaozhi-serial-reset-20260602-2128.log` and `reports/a21-stackchan-physical-wake-serial-20260602-2130.log`; physical evidence report `reports/a21-xiaozhi-physical-evidence-20260602-213147.097784000.json`; readiness report `reports/a21-product-readiness-20260602-213204.json`. |
| T-AUDIO-000: Read-only Xiaozhi audio/protocol audit | Completed | Worker thread `019e888d-f57d-7922-8e48-24b00255a122` found audio optimization landed, the physical audio path is stock-profile Xiaozhi Opus through A21 Gateway rather than old PCM bridge, and the most likely bad-sound boundary is TTS generation before Opus/downlink. |
| T-AUDIO-001a: Host downlink objective isolation | Completed | Worker thread `019e8895-43a6-7e23-a4f3-601f0451ab50`; `xiaozhi-voice-bench` now decodes captured binary downlink Opus frames and reports redacted aggregate `downlink_audio_quality` while preserving host-only candidate semantics. |
| T-AUDIO-001b: Host/Gateway post-Opus quality run | Completed host-only candidate | `reports/a21-xiaozhi-voice-bench-20260602-220325.719331000.json` passed 3-repeat host product-chain bench with `sherpa_onnx_tts`, answer p95 397 ms, and post-Opus quality passed; `reports/a21-xiaozhi-voice-bench-20260602-220344.175240000.json` passed a one-round check on the physical LAN Gateway; readiness `reports/a21-product-readiness-20260602-220447.json` and bundle `reports/a21-server-side-readiness-bundle-20260602-220501.json` still correctly block launch. |
| T-AUDIO-001c: Foreground long-TTS push attempt | Blocked, evidence-preserving | macOS output volume was set to 100 after explicit operator request; physical device `44:1b:f6:e2:6a:60` was online on stock Xiaozhi Opus via Gateway `21081`; `go run ./cmd/a21 stackchan-local-tts-playback --gateway-url http://127.0.0.1:21081 --device-id 44:1b:f6:e2:6a:60 --engine sherpa_onnx ...` returned `409 device audio websocket is not connected`, so no valid physical StackChan playback was claimed. |
| T-OPS-001: Desktop StackChan control boundary helper | Completed | Added `tools/desktop/a21-stackchan-control.command` and copied it to `/Users/jiyurun/Desktop/A21-StackChan-Control.command`; status check passed against Gateway `21081`; action probes and diagnostic tone correctly reported current stock-session 409 blockers instead of claiming control. |
| T-HW-VOLUME-001a: Fixed official codec volume candidate prep/build | Completed no-hardware candidate | Worker thread `019e88c3-8fa7-7e53-8e46-ab3ff6e637b9` prepared official codec `SetOutputVolume(92)` in the Xiaozhi-compatible overlay before `GetHAL().startXiaozhi()` and added a Go guard test; main branch integrated the patch, focused Go tests, official StackChan Go subset, clean official `HEAD` patch-apply check, `git diff --check`, and no-write build report `reports/a21-stackchan-official-baseline-20260602-225207-1780411927040145000.json` passed. App SHA-256: `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`. |
| T-HW-VOLUME-001b: Fixed official codec volume flash gate | Completed no-write gate | No-write flash plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-225600-1780412160509265000.json` is `status=ready`, `port=/dev/cu.usbmodem1101`, `dry_run=true`, `flash_allowed=false`, `flash_executed=false`, and app SHA-256 `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`; read-only worker `019e88d6-1734-75d2-897b-aa2da3069885` confirmed it is enough to request an operator-approved hardware window, not enough for physical acceptance. Read-only worker `019e88d6-ad61-7513-b379-aa21ae7db150` prepared the A/B evidence runbook and confirmed `stackchan-local-tts-playback`, Mac volume, diagnostic tone, host-only bench, and dry-run reports are not valid physical loudness acceptance. Rollback baseline: `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`, app SHA-256 `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d`. |
| T-HW-VOLUME-001c: Foreground fixed codec volume flash | Completed hardware write, A/B pending | Operator supplied guarded execute command; flash report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json` is `status=passed`, `port=/dev/cu.usbmodem1101`, `dry_run=false`, `flash_allowed=true`, `flash_executed=true`, app SHA-256 `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`. Post-flash Gateway `127.0.0.1:21081` health returned ok and device `44:1b:f6:e2:6a:60` remained registered online. This is not physical loudness acceptance. |
| T-PROTOCOL-001a: Stock Xiaozhi listen warning plan-first audit | Completed plan | Worker `019e88dd-1517-77c2-a002-7db771ec016a` created `docs/plans/2026-06-02-stock-xiaozhi-protocol-cleanup.md`; strongest hypothesis is Gateway sends server-to-device `type=listen` ack frames that stock firmware does not accept, while binary Opus/TTS downlink still reaches `speaking`. |
| T-PROVIDER-001a: Provider readiness truth audit | Completed read-only audit | Worker `019e88dd-3efa-78e2-a7d3-7063089cbf30` found `T-PROVIDER-001` stale as a missing-smoke blocker. Usable explicit provider evidence exists: DeepSeek smoke `reports/a21-provider-smoke-20260602-112710-368364000.json`, local Ollama smoke `reports/a21-provider-smoke-20260602-075644-199710000.json`, provider-ready readiness `reports/a21-product-readiness-20260602-160841.json`, and server-side bundle `reports/a21-server-side-readiness-bundle-20260602-160841.json`. Newer generic readiness reports that selected `mock` are invocation/context regressions, not absence of provider evidence. |

## Blocked Transitions

| Transition | Blocker | Required unblock |
| --- | --- | --- |
| T-HW-002b: Full StackChan physical acceptance after Gateway downlink | Gateway downlink exists, but device playback ack or operator/instrument audible observation is missing | Collect accepted audible playback evidence, device playback timing, and barge-in playback stop_done, then regenerate `xiaozhi-physical-evidence`. |
| T-AUDIO-002: Declare product voice quality acceptable | Physical sound is reported wrong; host/Gateway post-Opus answer metrics pass, but foreground physical A/B and audible observation are still missing; legacy PCM control playback is not connected to the current stock Xiaozhi physical session | Complete `T-AUDIO-001` Phase 2 through a real stock Xiaozhi operator-triggered turn, or approve a separate planned stock-safe TTS injection transition, then record operator/instrument observation. |
| T-HW-VOLUME-001d: Foreground StackChan volume A/B acceptance | Fixed official codec volume candidate is now flashed, but no post-flash phone/instrument recording or audible observation has been accepted; current stock Xiaozhi runtime still has no Gateway volume setter | Trigger a real stock Xiaozhi turn, collect the post-flash phone recording, compare loudness/clipping/intelligibility against the previous sample, then regenerate physical/readiness evidence. |
| T-PROVIDER-001b: Selected provider readiness refresh | Explicit real/local provider evidence exists, but newest generic readiness selected `mock` because it omitted explicit provider env/report | Rerun readiness and server-side bundle with the chosen provider report/env pinned, preserving redacted evidence and avoiding firmware key storage. |
| T-PRD-001: Declare full PRD physical acceptance | Candidate physical Gateway evidence exists but PRD accepted remains false | Close `T-HW-002b` and custom wake proof, refresh selected-provider readiness without falling back to `mock`, then rerun product readiness. |
| T-FW-003: Custom wake-word product acceptance | Needs guarded flash and physical proof | Wake package review, false-wake rejection, operator wake proof. |

## Next Candidate Transitions

1. `T-HW-VOLUME-001: StackChan Physical Speaker Volume Control`
   - Current phase: fixed official codec output-volume candidate has been
     flashed through the guarded foreground command.
   - Next action: trigger a real stock Xiaozhi turn and collect the post-flash
     phone recording with the same phone position and prompt class.
   - Acceptance for the current phase is flash execution only; physical loudness
     still needs A/B evidence.
   - Xiaozhi has mature `self.audio_speaker.set_volume` MCP-style evidence in
     docs/logs, but runtime protocol support must be a separate planned
     stock-safe transition rather than a silent Gateway extension.

2. `T-AUDIO-001: Isolate Xiaozhi TTS Quality From Opus/Device Playback`
   - Phase 1 and host/Gateway post-Opus checks are integrated and passing.
   - Control thread next runs foreground physical audible A/B only after
     operator approval for any Gateway/TTS profile swap.
   - Output decides whether the next fix is TTS voice/model quality, physical
     firmware/speaker playback, or protocol cleanup.

3. `T-PROTOCOL-001: Clean Stock Xiaozhi Control Compatibility`
   - Review whether stock-device `listen` ack replies should be suppressed,
     gated, or changed while preserving host bench semantics.
   - Plan ready in `docs/plans/2026-06-02-stock-xiaozhi-protocol-cleanup.md`;
     worker `019e88dd-1517-77c2-a002-7db771ec016a` identified the strongest
     hypothesis as server-to-device `type=listen` ack frames from Gateway.
   - Acceptance requires no more stock-firmware `Unknown message type: listen`
     warning in a fresh physical turn, without breaking listen/abort tests.

4. `T-PROVIDER-001b: Selected Provider Readiness Refresh`
   - Explicit DeepSeek and local Ollama provider evidence exists; do not rerun
     generic readiness in a way that silently selects `mock`.
   - Next action is a host-side readiness/bundle refresh pinned to the selected
     provider smoke report and provider env.
   - Keep provider keys host-side only and reports redacted; do not change
     firmware provider storage.
