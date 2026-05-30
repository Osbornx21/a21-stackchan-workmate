# A21 Codex Audit

Date: 2026-05-30
Workspace: `/Users/jiyurun/Documents/New project`
Branch: `codex/a21-phase1-clean-skeleton`

## Current Repo Structure

```text
AGENTS.md
Makefile
cmd/a21/main.go
docs/a21/
docs/engineering/
docs/superpowers/
firmware/stackchan/
firmware/artifacts/         # ignored packaged A21 binaries + sha256 receipts
go.mod
internal/app/
internal/audio/
internal/buildinfo/
internal/firmwarecheck/
internal/gateway/
internal/protocol/
internal/providers/
internal/runtimeguard/
internal/v21adapter/
```

## Language And Tooling

- Primary language: Go.
- Module path: `a21.local/a21`.
- Go version in module: `1.26`.
- Current verification entrypoint: `make verify`.
- Current runtime entrypoint: `go run ./cmd/a21`.
- Firmware build system: repository-local PlatformIO virtualenv under `.a21-tools/`, with `PLATFORMIO_CORE_DIR` pinned to `.a21-tools/platformio-core`.
- No Node/pnpm workspace exists yet.
- No Docker/Compose runtime exists yet.

The external TS/pnpm monorepo proposal remains a future option for simulator/dev-console work, not the current repository foundation.

## Current Commands

```bash
make verify
make firmware-test
make firmware-build
make firmware-package
make firmware-artifact-check
make firmware-upload-check
make firmware-device-check
make gateway
make release-check
go run ./cmd/a21 version
go run ./cmd/a21 preflight
go run ./cmd/a21 doctor
go run ./cmd/a21 serial-list
```

`doctor` now combines runtime preflight with firmware manifest/toolchain/artifact/serial inventory, voice provider health, provider network mode, provider registry readiness, provider smoke dry-run status, and optional V21 adapter health when `A21_V21_ADAPTER_URL` is configured. It writes JSON reports under `reports/` and verifies that a packaged A21 firmware artifact exists for the current git commit.

`make release-check` is the local high-confidence gate. It runs Go verification, firmware native tests, clean-worktree firmware packaging, and doctor. Because packaging embeds the current git commit, run it only from a clean tree after the intended commit exists.

## Current A21 Code

- `internal/buildinfo`: canonical A21 service identity.
- `internal/app`: CLI dispatch for version, preflight, doctor, gateway, serial inventory, and firmware release guards.
- `internal/audio`: Gateway audio ingress buffer plus RMS-based mock VAD transition detector.
- `internal/firmwarecheck`: A21 firmware manifest validation, artifact packaging, sha256 validation, serial inventory, upload dry-run checks, and Gateway device-identity dry-run checks.
- `internal/gateway`: mock Gateway HTTP/WebSocket server, validated audio-frame ingress, bounded audio ingress observability, active playback stream tracking for audio-path barge-in, real-sized mock PCM downlink chunks, metrics, device registry, trace waterfall, provider health endpoint, and built-in simulator HTML.
- `internal/runtimeguard`: env, endpoint, cwd, port, proxy/no-proxy, and fingerprint guardrails.
- `internal/protocol`: versioned A21 envelopes, audio chunks, control events, device events, modes, and expression states.
- `internal/providers`: provider-neutral voice contracts, deterministic mock/cascade behavior, explicit provider HTTP network policy, provider readiness registry, and redacted provider smoke boundary.
- `firmware/stackchan`: PlatformIO CoreS3 firmware lane with A21-only identity, Wi-Fi/Gateway state machines, control/audio WebSocket probes, mock playback downlink buffering, protocol parsing, and native Unity tests.
- `internal/v21adapter`: professional-mode V21 adapter contract, HTTP client, mock client, and legacy-port boundary checks.

## Namespace Findings

Intentional X21/V21 references exist in:

- runtime guard configuration
- runtime guard tests
- architecture and baseline docs

No new runtime package, command, process, or service name uses X21/V21 identity.

`a21 namespace-audit` now scans tracked file paths and blocks X21/V21-looking runtime paths outside the explicit V21 adapter/docs boundary. `make release-check` runs it before latency and firmware gates.

Firmware-specific protections now include:

- `firmware_id` must be `a21-stackchan`.
- PlatformIO envs must use `a21_` naming.
- artifact names must start with `a21-stackchan-`.
- artifacts containing X21/V21 names are rejected.
- upload ports containing X21/V21 names are rejected.
- release-index and per-artifact manifest paths containing X21/V21 names are rejected before upload-path dry-runs can pass.
- `firmware-current-artifact-check` validates the newest current-commit package from the release ledger and runs inside `make release-check`.
- firmware package requires a clean git worktree.
- firmware upload remains dry-run only and returns `flash_allowed: false`.
- raw PlatformIO upload targets are blocked by `scripts/a21_block_raw_upload.py` before any flash action can run.
- firmware device identity guard validates a Gateway `/v1/devices` capture against the exact packaged artifact and still returns `flash_allowed: false`.
- non-serial `/dev/*` paths such as `/dev/null` are rejected.
- StackChan Y-axis servo clamp is fixed at 5 to 85 degrees and covered in native firmware tests.
- semantic render states now map to safe Y-axis motion targets through a driver interface; CoreS3 currently uses a no-op driver until calibrated hardware output is added.
- semantic render states now map to RGB state colors through a driver interface; CoreS3 currently uses a no-op driver until calibrated RGB hardware output is added.
- semantic touch intents now map to A21 `touch.wake_or_listen` and `touch.barge_in` device events while preserving `screen` versus `top_sensor` source metadata; CoreS3 button inputs currently feed this runtime as a hardware-free development path.
- playback now has a tested no-op driver boundary that starts on speaking `stream_id`, stops and clears on barge-in, and replaces streams with stop/clear before restart; audio downlink chunks are parsed into a bounded firmware buffer keyed by `stream_id`; real speaker sample output is still not implemented.

## Legacy Neighbor Risks

Known local legacy territory:

- X21 backend: `8000`
- X21 local FunASR: `10095`
- V21/VKP services: `8080`, `18080`, `4173`, `42173`, `16686`, `16687`

A21 blocks endpoint env vars that point to known legacy ports and blocks A21 reserved port conflicts.

## Proxy And Network Findings

The local environment uses Dragon Cat Lite and DNS mapping into `198.18.0.x`. Phase 1 preflight records:

- default interface
- external DNS probe result
- proxy env variable names, not values
- direct-connect proxy bypass coverage when a global proxy exists

It blocks startup reports when the minimum fingerprint is missing. It also blocks global `HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY` configurations unless `NO_PROXY` or `A21_NO_PROXY` covers localhost, loopback, `.local`, and private LAN CIDRs including the A21 lab range. Explicit `A21_PROVIDER_PROXY_URL` is reported as provider egress configuration and does not satisfy LAN bypass coverage.

Provider HTTP clients default to `direct` mode and do not inherit environment proxies. `A21_PROVIDER_PROXY_URL` switches future HTTP provider adapters to `explicit_proxy` mode, while doctor still reports only env variable names and never prints proxy values.

Provider registry readiness currently audits `mock`, `doubao_realtime`, `doubao_tts_realtime`, `openai_realtime`, `bailian_dashscope`, and `deepseek`. It reports selected provider, capability labels, required env names, present env names, and missing env names only. Unknown and legacy-looking primary provider values are redacted to safe sentinel strings before doctor output is serialized.

Doubao realtime speech-to-speech now has an A21 voice provider boundary for selected-provider health, missing-env diagnostics, cancellation acknowledgements, and runtime selection tests. When configured, it reports `degraded` with an explicit execution guard rather than pretending to be an executable realtime path. It does not dial Volcengine and ordinary `StartTurn` execution is deliberately blocked until the official API shape, credentialed smoke, cancellation behavior, and latency are verified.

Provider smoke currently supports dry-run reports for all registered providers and explicit `--execute` smoke for `deepseek` and `bailian_dashscope` via OpenAI-compatible Chat Completions. Realtime WebSocket providers are reported as `unsupported` for HTTP smoke until dedicated realtime adapters exist.

## Verification Evidence

Fresh verification after the current Gateway/Simulator/Firmware guard baseline:

```text
make verify                 PASS
make firmware-test          PASS, 49/49 native firmware tests
go run ./cmd/a21 doctor     PASS, current artifact detected
firmware-artifact-check     PASS for current commit artifact
firmware-upload-check       FAILS SAFE when serial port is busy
firmware-upload-check       FAILS SAFE for /dev/null non-serial path
pio run -e a21_stackchan_native -t upload
                           FAILS SAFE through A21 raw-upload blocker
```

Known defensive checks:

```text
A21_PROVIDER_ENDPOINT=http://127.0.0.1:8000 ... preflight  exits 1
HTTPS_PROXY=http://127.0.0.1:7890 NO_PROXY=localhost ... preflight exits 1
PATH without route/dig ... preflight                         exits 1
firmware-upload-check --port auto ...                         exits 1
firmware-upload-check --port /dev/null ...                    exits 1
```

## Gaps

- Gateway and simulator are mock-first and deterministic; Gateway now has a bounded ingress buffer and RMS mock VAD markers, but physical microphone acceptance, physical speaker acceptance, production VAD, production jitter tuning, AEC, and provider audio streaming remain future work.
- Firmware has disciplined Wi-Fi/Gateway/control/audio transport probes, full-size mock audio-frame uplink, guarded mic PCM frame queueing and audio-frame uplink, bounded mock downlink buffering, a CoreS3 speaker pump build path, and a microphone capture policy with a speaker-busy guard, but it still does not claim physical mic quality, physical speaker acceptance, VAD, full-duplex, OTA, or real-world office latency.
- Metrics, voice provider health, and in-memory trace waterfall exist, including audio ingress/VAD markers, audio-path barge-in cancel markers, and professional V21 query latency; OpenTelemetry export and durable trace storage remain future work.
- V21 adapter contract, mock Gateway professional path, timeout fallback, latency metric, optional doctor health, and simulator evidence-card rendering exist; real V21 endpoint smoke remains future work.
- Provider-neutral mock/cascade contracts include health status, provider HTTP network policy, provider readiness registry, redacted provider smoke checks, a Gateway provider health endpoint, and explicit cancel reason/stream acknowledgements; no realtime provider adapters exist yet.
- Mock `latency-bench` exists for Gateway mock/professional/barge-in/audio-WS-downlink/audio-WS-barge-in report shape. `audio_ws_downlink_ms` measures until a full 20 ms / 16 kHz / mono / `pcm_s16le` silence chunk is returned, and `audio_ws_barge_in_stop_ms` measures from an interrupting voiced audio frame to the `interrupted` control event; real provider, LAN, microphone, speaker, and hardware latency benches remain future work.
- Simulator now has browser microphone/mock-burst controls, local mock playback state, Gateway downlink chunk count, buffer depth, active stream display, real PCM decode/schedule via WebAudio, and scheduled-source stop on interruption; real provider TTS and StackChan speaker output remain future work.
- Protocol and simulator now expose office visibility modes for private/public/pro/muted/listening states; only `professional` calls the V21 adapter.
- No CI yet.

These gaps are phase boundaries, not Phase 1 regressions.

## Recommended Next Step

Proceed through the next phase without diluting the Go core:

1. Add real V21 adapter smoke only after the Shanghai/V21 runtime endpoint is explicitly identified.
2. Expand simulator microphone/playback only after latency and trace fields are stable.
3. Keep real firmware flashing disabled until a separate explicit guarded flash command requires both upload-check and device-identity receipts.
