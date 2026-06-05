# A21 Atomic Feature Review

Status: current product ledger.
Date: 2026-06-06.
Branch: `codex/a21-mainline-recovery-internal-test4-20260606`.
HEAD: `47129dc`.

This document converts the A21 PRD, handoff log, state machine, current code,
and internal test 4 recovery baseline into an atomic feature ledger. It is a
review surface, not a new product authorization. Internal test 4 remains the
protected floor until the P0 stabilization transitions close.

## Legend

| Status | Meaning |
| --- | --- |
| `published_floor` | Included in the internal test 4 release baseline or the recovered mainline floor. |
| `implemented` | Code, endpoint, CLI, or firmware overlay exists in the current mainline. |
| `host_verified` | Local tests, dry-run, fixture, or host-only reports exist. |
| `deployed_or_flashed` | Evidence records a deployment, flash, or NVS write in the internal test 4 line. |
| `physical_pending` | Code or contract exists, but product physical acceptance is incomplete. |
| `p0_regression` | User-reported product regression blocks launch confidence. |
| `mock_or_planned` | Contract, UI, or diagnostic exists, but real execution/product acceptance is not present. |
| `stale_doc_risk` | Older control text conflicts with the recovered mainline or latest state machine. |

## Current Mainline Fact

| Item | Current fact |
| --- | --- |
| Protected release | `a21-internal-test4`, commit `1387d58f364f6ae1c7258487fb5a9863567adc74`. |
| Recovered mainline | `origin/main` and local `main` are at `47129dc`, which is internal test 4 plus stabilization/recovery docs and one provider test-fake concurrency fix. |
| Verification | `go test ./...` passed on 2026-06-06 after this review started. Earlier recovery verification also recorded `make verify` passed. |
| Mainline delta from internal test 4 | 8 files, 739 insertions: recovery docs, P0 RCA docs, stabilization plan/state updates, and `internal/providers/realtime_test.go`. |
| Active stabilization override | P0 stabilization is active. No broad feature merge, provider switch, product flash, or side-branch expansion should occur unless it closes recovery or P0 acceptance. |
| Known stale control text | `docs/engineering/A21_CURRENT_CONTROL.md` still describes an older hardware-window checkout and should be refreshed after this ledger is accepted. |

## Product And Governance

| ID | Atomic feature | Surface | Status | Evidence | Gap / next action |
| --- | --- | --- | --- | --- | --- |
| CTRL-001 | A21 identity and namespace guard | `AGENTS.md`, `runtimeguard`, `make preflight` | `implemented` / `host_verified` | Runtime guard and namespace tests exist; `make verify` is the default gate. | Keep enforcing `a21` / `A21_`; reject X21 naming outside explicit docs/tests/adapter contexts. |
| CTRL-002 | Internal test 4 recovery floor | Git + release docs | `published_floor` / `host_verified` | `docs/engineering/A21_INTERNAL_TEST4_RECOVERY_BASELINE.md`; main recovered to `47129dc`. | Treat `1387d58` as fallback and `47129dc` as current safe mainline. |
| CTRL-003 | Mainline recovery after lean regression | Git + handoff | `implemented` / `host_verified` | `docs/engineering/A21_MAIN_RECOVERY_20260606.md`; `origin/main` fast-forwarded, no force push. | Add a future CI/main guard that proves `origin/main` contains the protected release and key product files. |
| CTRL-004 | Control-tower handoff discipline | `docs/agent_handoff_log.md`, `docs/project_state_machine.md` | `implemented` / `stale_doc_risk` | Handoff log and state machine exist again on main. | Shorten future entries and refresh stale first-read docs so one file does not point to an old branch. |
| CTRL-005 | Branch and worktree sprawl control | Git process | `p0_regression` | 91-branch concern and lean-main incident show governance failure. | Read-only branch unique-commit inventory before deleting or cherry-picking; no branch-name authority over release commit authority. |
| SEC-001 | Provider/Wi-Fi/secret redaction | App reports, docs, runtime guard | `implemented` / `host_verified` | Report schemas forbid key, prompt, transcript, full URL, proxy, and path leaks. | Continue secret scanning for changed docs and release assets. |
| OBS-001 | Trace, metrics, doctor, readiness reports | `/v1/traces`, `/metrics`, `doctor`, reports | `implemented` / `host_verified` | Gateway route and CLI report families exist. | Physical session traces are still needed for voice-loop RCA and final PRD acceptance. |

## Device, Firmware, And Product Lane

| ID | Atomic feature | Surface | Status | Evidence | Gap / next action |
| --- | --- | --- | --- | --- | --- |
| FW-001 | Official-compatible product artifact lane | `a21-stackchan-official-xiaozhi-compatible.bin` | `published_floor` / `deployed_or_flashed` | Internal test 4 release artifact and guarded flash reports. | Keep generic `xiaozhi.bin` as dev-only; never use it for product flash. |
| FW-002 | Official StackChan front-end before AI.AGENT | M5Stack StackChan launcher + overlay | `implemented` / `deployed_or_flashed` | P0 power RCA says current overlay preserves official Launcher/Home before AI.AGENT. | Refresh old docs that still imply direct Xiaozhi autostart. |
| FW-003 | AI.AGENT enters A21 Xiaozhi runtime | Official app entry + A21 Gateway `/v1/xiaozhi` | `implemented` / `physical_pending` | User reported official front-end recovered and AI.AGENT entry works in the hardware window. | Product acceptance still depends on stable voice/body behavior after entry. |
| FW-004 | Product NVS provisioning | Official-compatible NVS lane | `deployed_or_flashed` | NVS report recorded OTA/WS config, app configured flag, servo calibration preservation, and single-slot Wi-Fi behavior. | User self-service provisioning without preloaded Wi-Fi is not accepted. |
| FW-005 | OTA discovery route | `/xiaozhi/ota/` | `implemented` / `deployed_or_flashed` | Gateway exposes OTA route; NVS points to A21 Gateway. | Official mobile app device-data parsing remains unresolved. |
| FW-006 | Product recovery executor | `stackchan-product-recovery` CLI/Make target | `implemented` / `host_verified` | Recovery executor plans/executes only guarded official-compatible app flash. | Use only in explicit hardware recovery windows. |
| FW-007 | sys_evt boot-loop recovery | Firmware overlay + Gateway relay URL handling | `deployed_or_flashed` | Handoff records stack overflow fix and percent-encoded MAC relay URL. | Confirm these changes remain in the product release path before any new firmware build. |
| FW-008 | No-USB physical power-key boot | PMIC/battery/rail path | `p0_regression` / `physical_pending` | P0 power RCA bounds failure before Gateway/provider/AI.AGENT when no serial/app log appears. | Timed no-USB press matrix against stock official and A21 product is still required. |
| FW-009 | Official mobile app binding/device data | Official app ecosystem | `p0_regression` / `mock_or_planned` | User saw `Failed to process device data`; no accepted RCA in current mainline. | Isolate official app data shape vs A21 NVS/Gateway OTA response. |

## Gateway And Protocol

| ID | Atomic feature | Surface | Status | Evidence | Gap / next action |
| --- | --- | --- | --- | --- | --- |
| GW-001 | Xiaozhi stock WebSocket | `/v1/xiaozhi` | `implemented` / `host_verified` | Gateway route exists; host benches and tests cover Xiaozhi protocol surfaces. | Product voice trace needed for wake/loop latency RCA. |
| GW-002 | Official StackChan body relay | `/stackChan/ws` | `implemented` / `physical_pending` | Route and status/control surfaces exist; relay can disconnect inside AI.AGENT. | Prefer official relay when connected, fall back to Xiaozhi MCP robot tools when not. |
| GW-003 | Device registry | `/v1/devices` | `implemented` / `deployed_or_flashed` | Records MAC/device state and product profile fields. | Mainline should retain MAC casefold behavior and avoid duplicate stale records. |
| GW-004 | Trace lookup | `/v1/traces` | `implemented` / `host_verified` | Workspace console and RCA flow rely on trace lookup. | Need field session traces for voice loop and body reaction evidence. |
| GW-005 | Hardware acceptance board | `/v1/hardware-acceptance` | `implemented` / `host_verified` | Routes and workspace console controls exist. | Physical acceptance remains incomplete for power, voice, body amplitude, and provisioning. |
| GW-006 | Power lifecycle evidence gate | `/v1/power-lifecycle`, `/v1/power-lifecycle-acceptance` | `implemented` / `host_verified` | Gate requires explicit no-USB cold boot metadata. | Do not accept USB-online status as no-cable power proof. |
| GW-007 | Voice mode catalog | `/v1/voice-modes` | `implemented` / `host_verified` | `roleplay` and `professional` are the only user-facing modes; `dialogue` normalizes to `roleplay`. | Keep mode switching user-initiated. |
| GW-008 | Gateway profile catalog | `/v1/gateway-profiles` | `implemented` | `public_wss` and `mac_local` are separate from voice mode. | Public trusted `wss` product deployment remains operational work. |
| GW-009 | Cloud voice profile catalog | `/v1/cloud-voice-profiles` | `implemented` / `mock_or_planned` | No-execute selector exists for Bailian/Doubao/MiniMax style profiles. | Real adapter execution and physical promotion still require separate evidence. |
| GW-010 | Workspace console | `/workspace` | `implemented` | Console calls device, traces, body, screen, roleplay, professional, workspace, and wake-word APIs. | Needs UI truth cleanup so stale docs do not imply unaccepted capabilities. |

## Voice And Provider Spine

| ID | Atomic feature | Surface | Status | Evidence | Gap / next action |
| --- | --- | --- | --- | --- | --- |
| VOICE-001 | Provider profile registry | `internal/providers` | `implemented` / `host_verified` | Text stream, realtime, voice-hybrid, local audio contracts and tests exist. | Keep new providers profile-driven, not Gateway-specific branches. |
| VOICE-002 | OpenAI-compatible text streaming | `provider-smoke --stream` | `implemented` / `host_verified` | Streaming parser supports content/reasoning/done timing and redaction. | Fresh selected-provider executed smoke is separate from physical acceptance. |
| VOICE-003 | Selected voice chain | `/v1/voice-chain-profiles` | `implemented` | ASR/LLM/TTS/realtime selected profile fields exist in registry/console. | No provider chain changes while voice-loop RCA is open unless tied to P0 evidence. |
| VOICE-004 | DashScope realtime ASR/TTS | `internal/providers/dashscope_realtime.go` and runtime profiles | `implemented` | File and tests exist on recovered mainline. | Need current runtime evidence before claiming better latency or wake sensitivity. |
| VOICE-005 | Doubao/OpenAI realtime voice adapters | `internal/providers/*realtime*` | `implemented` / `host_verified` | Offline realtime fixture and provider tests exist; fake race fixed. | Realtime is opt-in; do not route professional mode through opaque realtime. |
| VOICE-006 | Xiaozhi voice pipeline | Gateway `/v1/xiaozhi` + adapters | `implemented` / `physical_pending` | Product path supports Opus ingress/downlink and selected chain readiness. | P0 voice-loop RCA needs one aligned physical trace. |
| VOICE-007 | Local ASR/TTS seams | `local-asr-smoke`, `local-tts-smoke`, `local-voice-loopback` | `implemented` / `host_verified` | Local and wrapper contracts exist with redacted reports. | Host-local evidence does not equal StackChan speaker acceptance. |
| VOICE-008 | Voice clone CLI seam | `voice_clone_cli`, IndexTTS2 bridge | `implemented` / `mock_or_planned` | Contract and bridge docs exist. | Later 5080 voice-clone optimizations are not on main and need cherry-pick review. |
| VOICE-009 | Wake-word config and firmware package gates | `/v1/wake-word`, wake-word CLI | `implemented` / `physical_pending` | Wake-word config surface and build/acceptance commands exist. | User reports lower wake sensitivity; no before/after physical proof yet. |
| VOICE-010 | Barge-in / half-duplex | Gateway tests and StackChan acceptance commands | `implemented` / `physical_pending` | Half-duplex and playback acceptance commands exist. | Product symptom includes self-answer loop; do not mark PRD barge-in accepted until trace proves stop/cancel timing. |
| VOICE-011 | Voice-loop P0 RCA | Trace-driven RCA | `p0_regression` | `docs/engineering/A21_P0_VOICE_LOOP_RCA_20260605.md`; fake provider race fixed. | Capture wake/listen/audio/ASR/TTS/playback/relisten trace and apply one minimal fix. |
| VOICE-012 | Latency benchmark contract | `latency-bench`, `provider-latency-bench`, `xiaozhi-voice-bench` | `implemented` / `host_verified` | Canonical metrics and report shapes exist. | Physical first-audible and barge-in stop evidence still missing. |

## Roleplay, Personality, And Product Modes

| ID | Atomic feature | Surface | Status | Evidence | Gap / next action |
| --- | --- | --- | --- | --- | --- |
| RP-001 | Roleplay as default product mode | PRD + `/v1/voice-modes` | `implemented` | PRD v0.5 defines `roleplay` as default and `professional` as explicit switch. | Keep legacy `dialogue/workmate/companion` as aliases only. |
| RP-002 | Soul/personality profiles | `docs/personality`, `internal/personality` | `implemented` / `host_verified` | Role souls, scenarios, tone rules, composer tests exist. | Need field quality review after voice loop is stable. |
| RP-003 | Roleplay profile selector | `/v1/roleplay-profile` | `implemented` | Endpoint returns selected role, scenario, voice clone profile, memory summary, expression plan. | Confirm console labels map cleanly to product UX. |
| RP-004 | Memory hints | Roleplay profile request/runtime summary | `implemented` / `host_verified` | Memory text is not stored in reports; prompt input readiness is exposed. | Long-term memory CRUD is still non-goal/limited until explicit confirmation flow lands. |
| RP-005 | Roleplay voice probe | `roleplay-voice-probe` CLI | `implemented` / `host_verified` | Probe/report generator exists. | Physical roleplay listening quality remains gated by P0 voice RCA. |
| RP-006 | Roleplay expression plan | Body/screen actions from role state | `implemented` / `physical_pending` | Expression plan has action/packet counts and redaction fields. | Do not claim expression parity until body reactions feel at least stock-level. |

## Professional Mode And Workspace

| ID | Atomic feature | Surface | Status | Evidence | Gap / next action |
| --- | --- | --- | --- | --- | --- |
| PRO-001 | V21 adapter boundary | `internal/v21adapter`, docs | `implemented` / `host_verified` | Professional contract validates mode, privacy scope, query scope, evidence/card/follow-up fields. | Real V21 quality/citation acceptance remains V21-side evidence. |
| PRO-002 | Professional workspace selector | `/v1/professional-workspace` | `implemented` | Tracks user/workspace/query scope and readiness. | Account/auth/multi-user cloud product remains planned. |
| PRO-003 | Workspace document upload intake | `/v1/workspace-documents` | `implemented` / `mock_or_planned` | Multipart intake, local store limits, redaction fields, document/source/job records exist. | Real cloud index execution is planned; uploaded content must stay private and scoped. |
| PRO-004 | Upload/index job ledger | `/v1/workspace-upload-jobs`, `/v1/workspace-index-jobs`, `/v1/workspace-sources` | `implemented` | Metadata, readiness, source scope counts, and delete/retry surfaces exist. | Need real indexing worker and source lifecycle acceptance. |
| PRO-005 | Device binding guard | `/v1/workspace-device-bindings` | `implemented` | Binding policy and active/revoked counts exist. | Product account/device binding UX remains incomplete. |
| PRO-006 | Professional query endpoint | `/v1/professional-query` | `implemented` / `host_verified` | Query returns evidence-shaped response and read record metadata. | Hardware-initiated professional consult needs physical/user-confirmed route proof. |
| PRO-007 | Professional read records | `/v1/professional-read-records` | `implemented` | Redacted records for workspace queries exist. | Needs real adapter execution and evidence review UX before full PRD acceptance. |
| PRO-008 | Professional voice trigger | Xiaozhi/professional bench and mode ritual | `implemented` / `physical_pending` | Plans and bench routes exist; screen/voice checking cue specified. | Must not auto-switch from roleplay; field acceptance pending. |
| PRO-009 | Public/personal query scopes | `public_only`, `personal_only`, `personal_plus_public` | `implemented` / `mock_or_planned` | PRD and workspace runtime carry scope counts. | Real public/personal corpus and cloud account isolation not complete. |

## Body, Screen, Touch, And Hardware Parity

| ID | Atomic feature | Surface | Status | Evidence | Gap / next action |
| --- | --- | --- | --- | --- | --- |
| BODY-001 | Official MCP status | `/v1/xiaozhi/device-status`, `/v1/xiaozhi/mcp-control` | `implemented` / `host_verified` | Whitelisted `self.get_device_status`. | Raw official response capture remains redacted/limited. |
| BODY-002 | Speaker volume MCP | `/v1/xiaozhi/speaker-volume` | `implemented` / `physical_pending` | Whitelisted official speaker volume tool. | Physical loudness/playback acceptance remains separate. |
| BODY-003 | Screen brightness/theme/info | `/v1/xiaozhi/screen-brightness`, `/v1/xiaozhi/screen-theme`, MCP controls | `implemented` / `physical_pending` | Serial evidence for brightness/theme was recorded in hardware parity docs. | Product UI parity with official display states remains incomplete. |
| BODY-004 | Screen and top touch events | Product event registry + acceptance CLI | `implemented` / `deployed_or_flashed` | Touch acceptance reports exist for screen/top tap and swipes. | User later reported haptic/RGB reaction regressions; retest required. |
| BODY-005 | Touch-triggered body reactions | `A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS`, MCP LED/head | `implemented` / `p0_regression` | Trace/report evidence existed for touch reaction sequence. | User reports reaction/RGB/vibration felt rolled back; compare stock vs A21 and restore stock-level amplitude. |
| BODY-006 | State-triggered body reactions | `A21_XIAOZHI_PRODUCT_STATE_REACTIONS` | `implemented` / `physical_pending` | Gateway gates and trace markers exist. | Must remain disabled or conservative until physical UX is not worse than stock. |
| BODY-007 | Body preset/motion/scene APIs | `/v1/xiaozhi/body-preset`, `/body-motion`, `/body-scene` | `implemented` / `physical_pending` | Console controls and traces exist. | Product-level physical acceptance and pacing still required. |
| BODY-008 | Official robot MCP LED/head controls | `self.robot.set_led_color`, `self.robot.set_head_angles` | `implemented` / `physical_pending` | Hardware charter records serial HAL evidence for LED/head angles. | Servo amplitude is user-reported too weak; use official StackChan semantics before custom tuning. |
| BODY-009 | Official avatar/body relay frames | `/stackChan/ws`, `/v1/stackchan/official/control` | `implemented` / `physical_pending` | Control/status endpoint exists; relay is first transport when connected. | Inside AI.AGENT, relay may be disconnected; keep MCP fallback as product runtime path. |
| BODY-010 | Mic/speaker capability | Xiaozhi Opus ingress/downlink | `implemented` / `physical_pending` | Stock transport evidence exists. | Microphone physical acceptance and wake sensitivity remain P0. |
| BODY-011 | IMU diagnostic | Diagnostic firmware/acceptance commands | `mock_or_planned` | IMU probe build/flash/acceptance lanes exist. | Not product-available until read-only diagnostic evidence passes. |
| BODY-012 | Ambient/proximity/battery diagnostic | Sensor probe build/flash/acceptance commands | `mock_or_planned` | Sensor diagnostic discipline exists. | Battery/PMIC P0 may need hardware-level evidence outside app code. |
| BODY-013 | Camera | Official hardware/MCP photo reference | `mock_or_planned` | Privacy spike planned. | Keep blocked until consent, local handling, and redacted evidence exist. |
| BODY-014 | NFC and infrared | Official hardware reference | `mock_or_planned` | High-risk spike planned. | No product semantics or acceptance yet. |

## Product Surfaces, Release, And Deployment

| ID | Atomic feature | Surface | Status | Evidence | Gap / next action |
| --- | --- | --- | --- | --- | --- |
| REL-001 | Internal test 3 | Release package | `published_floor` | Team voice testing accepted by user before protocol churn. | Do not roll back inner-test3 voice protocol improvements. |
| REL-002 | Internal test 4 | GitHub release + firmware bundle | `published_floor` | Release assets uploaded at `1387d58`. | It is usable floor, not full PRD acceptance. |
| REL-003 | Public Gateway | ECS `47.103.57.217` | `deployed_or_flashed` / `physical_pending` | Handoff records SSH safe-swap deploys and health checks during hardware windows. | Current public runtime health needs a fresh recovery pass after mainline reset. |
| REL-004 | Product readiness rollup | `product-readiness`, `server-side-readiness-bundle` | `implemented` / `host_verified` | CLI rollups ingest provider, V21, Xiaozhi, physical, and release evidence. | Rollup should stay false until P0 physical gaps close. |
| REL-005 | Office/device handoff reports | `office-handoff`, `office-preflight`, `office-acceptance` | `implemented` | CLI gates exist for artifact, device, and acceptance reports. | Use for next controlled internal trial. |
| REL-006 | Repository artifact hygiene | `firmware/artifacts`, reports, branches | `p0_regression` | Many historical firmware artifacts and branches make project state hard to reason about. | Classify before deleting; move bulky historical artifacts to releases/archive plan rather than pruning blindly. |

## Development Progress Review

### Phase 0 - A21 Foundation

The project established a Go-first A21 spine with `cmd/a21`, runtime guard,
namespace/proxy discipline, firmware artifact checks, `doctor`, `preflight`,
`make verify`, traces, metrics, and redacted reports. This phase is broadly
implemented and testable.

### Phase 1 - Provider And Voice Spine

A21 added provider-neutral text stream, realtime voice, cloud voice catalog,
local ASR/TTS, voice-clone CLI, provider smoke, 5080lab import/package,
latency benchmarks, and selected voice-chain surfaces. The architecture is
stronger than a single vendor path, but physical voice experience is still
blocked by the P0 wake/latency/self-loop RCA.

### Phase 2 - StackChan/Xiaozhi Product Path

The project moved from a generic device client toward the official-compatible
StackChan path: official OTA/NVS, `/v1/xiaozhi`, official Home/Launcher before
AI.AGENT, official body relay, Xiaozhi MCP controls, guarded product flash, and
product recovery executor. Internal test 4 is the product floor. The remaining
hard blockers are no-USB power, official app binding/provisioning, and body
parity acceptance.

### Phase 3 - Roleplay And Professional Product Shape

The PRD is now clear: A21 has two user-facing modes, `roleplay` and
`professional`. Roleplay has soul/personality/scenario/memory/voice-clone
surfaces. Professional mode has a V21 adapter boundary, workspace selectors,
document upload/index/source/device-binding ledgers, professional query, and
read records. These are implemented mostly as product/control contracts, not
yet as fully accepted cloud account + real index + hardware consult UX.

### Phase 4 - Internal Test 3 To Internal Test 4

Internal test 3 was accepted for team voice testing after major protocol work.
Internal test 4 packaged the cloud workspace/product lane and official-compatible
firmware floor. It was good enough to publish, but the user-reported P0 issues
mean it must be treated as a fallback floor rather than a completed launch.

### Phase 5 - Regression And Recovery

The mainline accident happened because `main` was an older lean ancestor while
internal test 4 lived on release/hardware branches. The recovered mainline is
now back on the internal test 4 floor. The immediate priority is no longer
feature expansion; it is controlled stabilization and selective cherry-pick
review of any side-branch work that might contain real improvements.

## Current P0 Gap List

| Priority | Gap | Why it matters | Required proof before closure |
| --- | --- | --- | --- |
| P0 | No-USB physical power-key boot | A physical product that cannot reliably power on is not acceptable. | Timed no-USB PWRKEY matrix, stock official vs A21 comparison, battery/PMIC evidence. |
| P0 | Voice wake sensitivity, delayed reply, self-answer loop | This destroys the core experience even if code tests pass. | One aligned product trace covering wake/listen/audio/ASR/TTS/playback/tts.stop/relisten. |
| P0 | Body parity regression | The product must feel like StackChan/Xiaozhi with a full body, not a weaker custom shell. | Stock-vs-A21 touch/RGB/servo/vibration matrix and amplitude/transport decision. |
| P0 | No-preloaded-Wi-Fi provisioning | Internal users must be able to recover/configure without a developer terminal. | Fresh/no-Wi-Fi path through official setup/mobile/app or a documented internal-test limitation plus guarded preload path. |
| P0 | Official app `Failed to process device data` | It blocks the official front-end/binding path the user is willing to accept. | RCA of official device-data shape, NVS/app_config, OTA response, and Gateway handoff. |
| P1 | ECS/public Gateway fresh health | Mainline was recovered; runtime needs reconfirmation. | Read-only public health/devices/official status checks, then deploy only if code differs. |
| P1 | Voice clone/5080 optimizations | Potential valuable work exists off main. | Selective cherry-pick review proving no deletion of internal test 4 capabilities. |
| P1 | Repo artifact/branch hygiene | Large files and many branches obscure truth. | Read-only unique-commit inventory and artifact archival plan before deletion. |

## Recommended Next Transitions

1. `T-A21-CONTROL-DOC-REFRESH-001`
   - Refresh stale first-read control docs so they point to `47129dc`, the
     internal test 4 floor, and this atomic ledger.

2. `T-A21-P0-POWER-BOOT-EVIDENCE-001`
   - Execute the timed no-USB PWRKEY matrix without adding PMIC patches.

3. `T-A21-P0-VOICE-TRACE-CAPTURE-001`
   - Capture one product voice trace and classify loop origin before changing
     providers, prompts, or firmware.

4. `T-A21-P0-BODY-PARITY-MATRIX-001`
   - Compare stock official vs A21 for touch feedback, RGB, vibration, yaw,
     pitch, and scene amplitude. Use official MCP/body semantics first.

5. `T-A21-P0-PROVISIONING-APP-RCA-001`
   - Isolate official app device-data failure and decide whether internal test
     uses official setup, guarded preload, or both.

6. `T-A21-SIDE-BRANCH-UNIQUE-COMMIT-INVENTORY-001`
   - Produce a read-only list of side-branch unique commits and candidate
     cherry-picks against the internal test 4 floor.

## Hard Rules From This Review

- Do not treat a branch name as authority over a release commit.
- Do not promote an endpoint/control contract to product acceptance without
  physical or runtime evidence matching the PRD claim.
- Do not fix power, voice, and body regressions in one mixed patch.
- Do not reintroduce speculative PMIC writes unless stock official passes and
  A21 fails under the same no-USB conditions.
- Do not switch providers or rewrite personality to hide a voice-loop symptom.
- Do not prune/delete branches or artifacts until unique commits and release
  assets are classified.
