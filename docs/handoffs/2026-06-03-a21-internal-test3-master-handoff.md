# A21 Internal Test 3 Master Handoff

Date: 2026-06-03 CST  
Thread scope: public-Gateway Xiaozhi voice main-chain closure, firmware lane
recovery, provider selector/runtime recovery, and internal test 3 release.  
Audience: next A21 control-tower thread, release/integration worker, physical
validation worker.

This document is the repo-carried recovery point for the current thread. It is
deliberately more detailed than the short release note. Use it before continuing
Gateway, provider, firmware, or physical StackChan work.

## Executive State

- Current branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Current repo HEAD:
  `221c153 docs(release): publish internal test 3`.
- Current remote:
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash`
  is at the same HEAD.
- Tracked worktree state at handoff creation: clean.
- Existing untracked noise only:
  `.DS_Store`, `internal/.DS_Store`,
  `docs/engineering/A21_GOVERNANCE_REMEDIATION_PLAN.md`.
- Internal test 3 package:
  `dist/a21-internal-test3-20260603-233245`.
- Internal test 3 tarball:
  `dist/a21-internal-test3-20260603-233245.tar.gz`.
- Tarball SHA-256:
  `62d2e62fcab0dee9cbf1e4ae5c49ec5941ce162ec65c0bf2c873b3e70ffda93f`.
- Package source commit:
  `074e3d877d33`.
- Important SHA distinction:
  the release package source archive is pinned to `074e3d877d33`; repo HEAD
  `221c153` adds the release documentation after packaging. Do not treat this
  as a wrong package.

Internal test 3 status:

- Voice main chain is accepted for internal testing by the control thread.
- Main public product Gateway is `47.103.57.217`.
- StackChan device WebSocket URL is `ws://47.103.57.217/v1/xiaozhi`.
- Mac-local Gateway remains a selectable local-fast path.
- This is not full PRD launch green. Machine-readable physical evidence is
  still `candidate_gateway_downlink`, and product readiness remains blocked.

## Hard Boundaries

- Do not put provider keys, Wi-Fi credentials, private keys, transcripts,
  prompt text, local secret paths, or audio payloads into repo, firmware,
  reports, or package docs.
- Do not flash product StackChan with bare `xiaozhi.bin`.
- Product firmware lane remains:
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- The public Gateway is now the main product Gateway, not a candidate label.
- The old SWAS host was experimental/backup only; do not promote it over
  `47.103.57.217`.
- X21 is reference-only. Do not introduce X21 runtime identity or package names
  into A21 deliverables.
- Host/bench evidence is useful but is not physical PRD acceptance.
- User audible acceptance is important control evidence, but full PRD evidence
  still needs machine-readable playback/stop acknowledgement or a trusted
  observation sidecar.

## Live Runtime Snapshot

Snapshot taken during this handoff refresh on 2026-06-03 CST:

```text
curl http://47.103.57.217/healthz
status: ok
service: a21-gateway
version: 0.1.0-dev
```

Product device:

```text
device_id: 44:1b:f6:e2:6a:60
connection_status: online
xiaozhi_transport: websocket
xiaozhi_audio: opus_16000hz_mono_60ms
current_voice_mode: dialogue
current_voice_chain_mode: cascade
current_asr_profile: dashscope_qwen_asr_realtime
current_llm_profile: deepseek
current_tts_profile: dashscope_qwen_tts_realtime
current_realtime_provider: doubao_realtime
current_voice_clone_profile: a21_voice_default_dashscope
current_cloud_voice_profile: a21_doubao_tts_realtime
last_trace_id: a21-trace-44-1b-f6-e2-6a-60
last_session_id: a21-session-44-1b-f6-e2-6a-60
last_event: xiaozhi.hello
```

Voice-chain selector surface:

```text
selected_voice_chain_mode: cascade
selected_asr_profile: dashscope_qwen_asr_realtime
selected_llm_profile: deepseek
fixed_tts_profile: dashscope_qwen_tts_realtime
hot_switch: true
findings: stepfun_not_selected
```

Selector intent:

- Cascade mode: user selects ASR and LLM; TTS remains fixed for quality and
  latency comparability.
- Realtime mode: user sees/selects realtime provider.
- Voice clone: separate voice selection maps to a default TTS/profile.
- StepFun 8k is still the recommended fast LLM when credentials are present,
  but it is not selected on the live public Gateway at this handoff point.

OTA snapshot:

```text
curl http://47.103.57.217/xiaozhi/ota/
websocket.url: ws://47.103.57.217/v1/xiaozhi
```

## What This Thread Completed

### 1. Product Gateway Direction

The thread moved the product path from local/Mac-only and experimental public
reachability into a main public Gateway on ECS:

- New main host: `47.103.57.217`.
- Purpose: A21 voice edge / Xiaozhi-compatible public Gateway.
- Device URL: `ws://47.103.57.217/v1/xiaozhi`.
- Gateway runs as a systemd service behind public HTTP/HTTPS reverse proxy.
- OTA returns the public WebSocket URL.
- Mac Gateway remains available as a frontend-selectable local-fast profile for
  local models and local processing.

Key result:

- Public Gateway is the primary product voice path for internal test 3.
- Mac Gateway is not removed; it is a selectable local profile.

### 2. Firmware Lane Recovery

The thread repeatedly corrected firmware identity and build-path mistakes and
locked the product lane back to the official-compatible artifact:

- Product artifact:
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- Product candidate:
  `a21-stackchan-official-xiaozhi-compatible`.
- Latest real flash evidence in the package:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-170812-1780477692092580000.json`.
- That flash report has `flash_executed=true`.
- No bare `xiaozhi.bin` product flash is used by the internal test 3 release
  evidence.

Firmware-side work preserved or restored:

- Official Xiaozhi-compatible startup path.
- Official avatar/action behavior.
- No welcome-screen interruption after StackChan app preload.
- Product flash-lane guardrails.
- Xiaozhi-style Wi-Fi provisioning contract: hotspot default, BluFi and
  acoustic named as alternatives.
- Custom wake work for `紫悦` and related wake asset fixes.
- Wake on idle socket behavior.

Current caveat:

- `make preflight` and `make doctor` still warn
  `firmware_current_artifact_missing` and `wake_word_firmware_build_required`
  in the local readiness surface. This does not erase the latest real flash
  evidence, but it means formal readiness still wants a refreshed artifact/build
  pointer before launch.

### 3. Xiaozhi Protocol And Audio State Machine

The thread re-centered A21 on the stock Xiaozhi-style WebSocket/Opus state
machine instead of inventing a parallel audio protocol.

Target shape:

```text
hello
listen.start
binary Opus uplink, 16 kHz mono, 60 ms frames
listen.stop or trusted turn end
stt partial/final
tts.start
binary Opus downlink
tts.stop
abort / wake-as-barge-in while speaking
```

Important fixes in this thread:

- Fast-ack is no longer a hard dependency for the answer path.
- DashScope realtime TTS adapter was rebuilt around the provider lifecycle
  instead of synchronous-RPC assumptions.
- Streaming ASR final arriving after the short commit wait can still claim the
  turn and start the answer.
- Non-empty cloud ASR final is treated as speech evidence even if local
  Gateway RMS/VAD missed speech.
- ASR partials no longer start audible speech before the user finishes.
- Stock physical MAC devices no longer let the two-frame Gateway VAD hangover
  auto-stop user speech too early.
- Speaking self-loop was fixed: if answer audio already went out and provider
  finalization errors later, A21 records degraded completion instead of
  falling into local fallback and starting a new prompt loop.
- Post-TTS drain suppresses tail `listen.start` sessions.
- Audio from suppressed listen sessions is ignored and cannot pollute wake
  pre-roll or the next ASR turn.
- Stop-tail frames after a suppressed session are drained.
- Barge-in/abort remains allowed while the device is truly speaking.

Practical outcome:

- The user accepted the voice main-chain breakthrough for internal test 3.
- Host bench shows answer and barge-in timing.
- Physical reports show device online, Opus ingress, and Gateway downlink.
- Device-level playback ack and stop_done remain missing from stock firmware
  telemetry.

### 4. Provider And Voice Selector Work

The thread moved provider work from hard-coded assumptions toward user-visible
selection surfaces:

- Added voice-chain hot switch selector.
- Added cascade versus realtime selector model.
- Cascade mode exposes ASR and LLM profiles while fixing TTS.
- Realtime mode exposes realtime providers.
- Voice clone is modeled separately from mode and Gateway profile.
- Added cloud voice profile catalog and no-execute selector surfaces.
- Added provider-neutral surfaces for Bailian/Qwen/CosyVoice, Doubao realtime
  and clone, and MiniMax TTS/clone families.
- Added simulator/device-registry readout for selected cloud voice profile.
- Added env-name-only readiness reporting.

Current live provider state:

- ASR: `dashscope_qwen_asr_realtime`.
- LLM: `deepseek`.
- TTS: `dashscope_qwen_tts_realtime`.
- Realtime provider shown: `doubao_realtime`.
- Voice clone profile shown: `a21_voice_default_dashscope`.
- Finding: `stepfun_not_selected`.

Important caveat:

- The user asked why DeepSeek is still selected. Current truth is that the
  live public Gateway still reports DeepSeek as the selected LLM, with StepFun
  only recommended. Switching to StepFun requires root-only secret/env update
  on the ECS, Gateway restart, and fresh bench/physical evidence. Do not fake
  this in docs or local config.

### 5. Validation And Release Packaging

Internal test 3 release package includes:

- Packaged binary: `bin/a21`.
- Source archive: `source/a21-source-074e3d877d33.tar.gz`.
- Public Gateway snapshots:
  health, devices, gateway profiles, voice-chain profiles, OTA.
- Voice bench report.
- Physical evidence report.
- Product readiness and server-side readiness bundle.
- Streaming provider readiness report.
- Latest real official-compatible flash evidence.
- Packaged binary host gate report.
- SHA256 manifest.

Fresh validation results from release closure:

```text
make verify: passed
make preflight: passed with firmware/wake warnings
make doctor: passed with firmware/wake warnings
packaged ./bin/a21 gate --scope host: passed with same warnings
public gateway health/profile/OTA/devices curls: passed
tarball SHA check: passed
manifest JSON validation: passed
git diff --check HEAD~1..HEAD: passed
```

Latest host bench included in package:

```text
report: reports/a21-xiaozhi-voice-bench-20260603-233014.342186000.json
status: passed
acceptance_status: candidate_host_only
prd_accepted: false
ASR: dashscope_qwen_asr_realtime
LLM: deepseek
TTS: dashscope_qwen_tts_realtime
answer_first_audio_p95_ms: 792
barge_in_stop_p95_ms: 20
failure_count: 0
```

Latest physical evidence included in package:

```text
report: reports/a21-xiaozhi-physical-evidence-20260603-232946.250456000.json
device: 44:1b:f6:e2:6a:60
status: candidate_gateway_downlink
observed: device online, Opus ingress, Gateway downlink
missing: device.playback.ack, device.playback.stop_done, trusted operator audible observation sidecar
```

Latest readiness included in package:

```text
product readiness:
  report: reports/a21-product-readiness-20260603-233027.json
  demo_ready: true
  launch_ready: false
  status: server_side_blocked

server-side readiness bundle:
  report: reports/a21-server-side-readiness-bundle-20260603-233027.json
  status: server_side_blocked
```

## Commit Ledger

The full branch range can be reconstructed with:

```bash
git log --oneline --reverse 4fa66c85300c..221c153
```

High-signal integration sequence:

```text
5d3b548 chore(network): default make targets to A21 direct no-proxy
1e2b37b feat(gateway): add xiaozhi oem control bridge
9a4f557 fix(firmware): remove custom avatar fallback
cdabe89 feat(transport): map a21 events to official stackchan frames
ea51c67 feat(gateway): relay official stackchan avatar actions
a36206f feat(gateway): sync xiaozhi turns to official stackchan
f7c95f0 fix(firmware): freeze external x21 xiaozhi builds
987bbb0 feat(firmware): add official xiaozhi compatible stackchan build
69c4bbe docs(control): add handoff and state machine workflow
6f34091 feat(firmware): add official xiaozhi compatible flash plan
37ef8f3 feat(firmware): add official xiaozhi nvs connection config
e7e9b03 feat(firmware): autostart official xiaozhi candidate
4613946 fix(firmware): enter official xiaozhi runtime directly
2fe4947 feat(audio): report xiaozhi downlink quality
f0603f5 fix(firmware): set official xiaozhi codec volume
060d2bb fix(audio): accept stackchan xiaozhi playback hotfix
b5a405e feat(audio): add hot-pluggable iflytek tts candidate
7a8af2d feat(gateway): play relay wav through xiaozhi say
b912a82 fix(firmware): guard stackchan product flash lane
46586da fix(firmware): add zi yue wake to stackchan app
7f3225e fix(firmware): bypass stackchan welcome setup
5242349 fix(voice): bound xiaozhi listen and tune zi yue wake
a986d6b fix(firmware): preload stackchan apps before xiaozhi
9ba8bc1 fix(firmware): skip welcome after stackchan app preload
88cb8a9 fix(firmware): bound stackchan listening state
8e4df0b fix(firmware): split zi yue wake commands
4b2f6d5 fix(gateway): suppress no-speech listen loops
ae4f5da feat(app): add xiaozhi realtime parity gate
3585144 feat(gateway): add xiaozhi streaming asr seam
e694550 fix(firmware): park after stackchan xiaozhi autostart
b49a69b feat(app): gate xiaozhi streaming provider readiness
399a2a6 feat(providers): add sherpa streaming asr seam
7906975 feat(providers): add streaming tts seam
9c7cdbb feat(a21): add sherpa streaming asr smoke
0d2fc73 feat(a21): add streaming tts runtime smoke
a8d51d4 feat(a21): bridge xiaozhi asr partials to streaming answer
2270a8d feat(a21): discover canonical sherpa streaming asr cache
06d4cd1 feat(a21): require real profiles for xiaozhi realtime parity
4a0ee38 fix(a21): keep xiaozhi asr commit nonblocking
2debc74 feat(a21): preserve stock xiaozhi stt and wake preroll
fc79156 feat(a21): queue xiaozhi opus ingress
2f8a63f feat(xiaozhi): suppress stale opus ingress after abort
d15a7b4 feat: promote public voice gateway and wifi provisioning
face173 feat: enable public edge dashscope voice downlink
e8c9427 feat(xiaozhi): add stock half-duplex acceptance
4c4178a fix(stackchan): keep official dependency cache clean
49ea6fb feat(cloud-voice): add no-execute profile selector
ee55ccd feat(voice): add cloud-edge realtime provider adapters
c104b3c feat(gateway): add voice chain hot switch selector
8107270 fix(gateway): continue answer after fast ack miss
daf6c76 fix(gateway): trace voice pipeline failure stage
867d7ae fix(providers): add dashscope realtime event ids
368b802 fix(providers): surface realtime tts empty audio failures
5b5bd4d fix(providers): align dashscope realtime tts lifecycle
7dcac2e fix(providers): follow dashscope tts commit finish order
0a716fb chore(gateway): trace realtime tts failure stages
412b0a4 chore(voice): classify tts adapter startup failures
8752b8d fix(app): count cloud edge xiaozhi product chain
9d24331 docs(voice): record public cloud edge bench pass
760deb0 docs(voice): record physical public gateway trace
b01e867 fix(firmware): override asset wake commands for zi yue
920e2e2 fix(firmware): fit wake assets partition
aa80523 fix(firmware): allow wake on idle socket
27d8a34 fix(gateway): defer xiaozhi speech until listen stop
b487c39 docs(gateway): record xiaozhi listen boundary deploy
0161d84 fix(gateway): answer after late streaming asr final
272edea fix(gateway): treat streaming asr final as speech evidence
0aa1eda fix(gateway): prevent xiaozhi speaking self loop
862791e fix(gateway): drop suppressed xiaozhi listen audio
e17aa3d fix(gateway): drain suppressed xiaozhi listen tail
074e3d8 docs(gateway): record xiaozhi suppressed listen verification
221c153 docs(release): publish internal test 3
```

The many `docs(control)` commits between these implementation commits are not
noise. They are the control-tower audit trail and should be preserved when this
branch is integrated.

## Operator Runbook

From repo root:

```bash
git status --short --branch
cat .a21-run/latest-internal-test3-package.path
shasum -c dist/a21-internal-test3-20260603-233245.tar.gz.sha256
```

Public Gateway checks:

```bash
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
  curl -sS http://47.103.57.217/healthz | jq .

NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
  curl -sS http://47.103.57.217/v1/devices \
  | jq '.devices[] | select(.device_id=="44:1b:f6:e2:6a:60")'

NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
  curl -sS http://47.103.57.217/v1/voice-chain-profiles | jq .

NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
  curl -sS http://47.103.57.217/xiaozhi/ota/ | jq .
```

Host bench:

```bash
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
go run ./cmd/a21 xiaozhi-voice-bench \
  --gateway-url http://47.103.57.217 \
  --repeat 1 \
  --timeout-ms 16000 \
  --require-product-chain \
  --output-dir reports
```

Physical evidence:

```bash
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
go run ./cmd/a21 xiaozhi-physical-evidence \
  --gateway-url http://47.103.57.217 \
  --device-id 44:1b:f6:e2:6a:60 \
  --trace-id a21-trace-44-1b-f6-e2-6a-60 \
  --session-id a21-session-44-1b-f6-e2-6a-60 \
  --output-dir reports
```

Readiness:

```bash
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
go run ./cmd/a21 product-readiness \
  --gateway-url http://47.103.57.217 \
  --device-id 44:1b:f6:e2:6a:60 \
  --use-latest-reports \
  --output-dir reports

NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
go run ./cmd/a21 server-side-readiness-bundle \
  --gateway-url http://47.103.57.217 \
  --device-id 44:1b:f6:e2:6a:60 \
  --use-latest-reports \
  --output-dir reports
```

ECS service checks:

```bash
ssh root@47.103.57.217
systemctl status a21-gateway
journalctl -u a21-gateway -f
systemctl status caddy
curl -sS http://127.0.0.1:21081/healthz | jq .
```

Deploy pattern used in this thread:

```bash
git archive --format=tar HEAD | ssh root@47.103.57.217 '
set -e
export PATH=/usr/local/go/bin:$PATH
rm -rf /opt/a21.next
mkdir -p /opt/a21.next
tar -xf - -C /opt/a21.next
cd /opt/a21.next
go test ./internal/gateway -count=1
go build -o /opt/a21.next/bin/a21 ./cmd/a21
systemctl stop a21-gateway
rm -rf /opt/a21.prev
mv /opt/a21 /opt/a21.prev
mv /opt/a21.next /opt/a21
systemctl start a21-gateway
sleep 2
systemctl is-active a21-gateway
curl -fsS http://127.0.0.1:21081/healthz
'
```

Rollback pattern:

```bash
ssh root@47.103.57.217 '
set -e
systemctl stop a21-gateway
test -d /opt/a21.prev
rm -rf /opt/a21.failed
mv /opt/a21 /opt/a21.failed
mv /opt/a21.prev /opt/a21
systemctl start a21-gateway
systemctl is-active a21-gateway
curl -fsS http://127.0.0.1:21081/healthz
'
```

## Remaining Work

### P0 Next

1. Switch live public Gateway LLM from DeepSeek fallback to StepFun 8k when the
   StepFun credentials/model are present on the ECS.
2. Restart Gateway and rerun:
   `/v1/voice-chain-profiles`, host bench, physical evidence, readiness bundle.
3. Add or ingest trusted physical audible/playback evidence:
   `device.playback.ack`, `device.playback.stop_done`, or a trusted operator
   observation sidecar.

### P1 Next

1. Harden frontend/provider state machine around cascade versus realtime:
   cascade selects ASR/LLM, fixed TTS; realtime selects provider; clone selects
   voice/TTS mapping.
2. Promote the selector UI only after it is backed by runtime state, not just
   static docs.
3. Add formal WSS/domain/TLS path for product use. Internal test 3 currently
   uses `ws://47.103.57.217/v1/xiaozhi`.
4. Refresh firmware-current artifact pointer so preflight/doctor warnings do
   not obscure the already-flashed official-compatible product lane.

### Do Not Start By Doing

- Do not rebuild firmware from a bare Xiaozhi tree and flash `xiaozhi.bin`.
- Do not switch the Mac network.
- Do not store provider secrets in repo, docs, shell transcripts, or firmware.
- Do not claim `launch_ready=true` from host bench or candidate physical
  downlink evidence.
- Do not re-open old PCM/base64 hardware audio paths for product acceptance.

## Recovery Checklist For The Next Thread

1. Read this file.
2. Read `docs/agent_handoff_log.md` from the latest entries upward.
3. Read `docs/project_state_machine.md`.
4. Run `git status --short --branch`.
5. Check public Gateway health/profile/OTA/device state.
6. Confirm whether StepFun is still `stepfun_not_selected`.
7. If changing runtime, preserve root-only secret handling and rerun host plus
   physical evidence.
8. If changing firmware, stay on
   `a21-stackchan-official-xiaozhi-compatible.bin` and use the guarded flash
   lane only.

## Final Classification

Internal test 3 is a real voice-main-chain release for team testing:

- Public Gateway main path is live.
- StackChan connects over stock Xiaozhi WebSocket.
- ASR/LLM/TTS cascade path is wired.
- Host bench passes with low-latency answer and barge-in timing.
- Physical device evidence shows online Opus ingress and Gateway downlink.
- User accepted the audible voice main-chain breakthrough in the control
  thread.

It is not full PRD launch ready:

- StepFun is recommended but not selected.
- Runtime LLM is still DeepSeek.
- Machine-readable physical evidence lacks playback ack and stop_done.
- Product readiness remains `server_side_blocked`.
- Trusted operator audible acceptance has not yet been encoded into a report.
