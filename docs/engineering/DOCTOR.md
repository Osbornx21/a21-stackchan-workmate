# A21 Doctor

## Current Command

```bash
go run ./cmd/a21 doctor
go run ./cmd/a21 doctor --output-dir reports
go run ./cmd/a21 serial-list
go run ./cmd/a21 firmware-check --kind device --artifact firmware/artifacts/<a21-stackchan...bin> --device-report reports/a21-devices.json --device-id stackchan-001 --commit <git-sha>
go run ./cmd/a21 provider-realtime-plan --provider doubao_realtime
go run ./cmd/a21 provider-realtime-plan --provider doubao_tts_realtime
go run ./cmd/a21 v21-adapter-smoke --output-dir reports
go run ./cmd/a21 v21-adapter-smoke --execute --output-dir reports
go run ./cmd/a21 provider-latency-bench --provider mock --iterations 5 --output-dir reports
go run ./cmd/a21 provider-latency-bench --provider deepseek --fixture reports/a21-redacted-audio-fixture.json --iterations 30 --output-dir reports
go run ./cmd/a21 product-readiness --gateway-url http://127.0.0.1:21080 --output-dir reports
go run ./cmd/a21 lan-probe --target a21-gateway=127.0.0.1:21080 --output-dir reports
go run ./cmd/a21 stackchan-accept --check mic-probe --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> --window-ms 5000 --min-delivery-ratio 0.95 --output-dir reports
go run ./cmd/a21 stackchan-accept --check half-duplex --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> --window-ms 1500 --min-mic-frames 1 --min-playback-chunks 1 --min-delivery-ratio 0.95 --output-dir reports
go run ./cmd/a21 stackchan-accept --check speaker --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> --window-ms 1500 --mock-audio-chunks 50 --min-played-frames 50 --output-dir reports
go run ./cmd/a21 stackchan-accept --check sensor-probe --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> --window-ms 1500 --min-samples 10 --min-battery-mv 3000 --output-dir reports
go run ./cmd/a21 stackchan-fast-companion-turn --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --repeat 3 --output-dir reports
go run ./cmd/a21 stackchan-fast-companion-turn --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --listen-source stackchan_mic --mic-window-ms 1200 --min-mic-frames 1 --repeat 3 --output-dir reports
make doctor
make lan-probe
make stackchan-mic-probe-acceptance
make stackchan-half-duplex-acceptance
make stackchan-imu-probe-acceptance
make stackchan-sensor-probe-acceptance
make stackchan-speaker-acceptance
make stackchan-fast-companion-turn
make provider-realtime-plan
make provider-latency-bench
make v21-adapter-smoke
make v21-adapter-smoke-execute
```

`doctor` emits the preflight report, A21 firmware/tooling status, and also writes a timestamped JSON file:

```text
reports/a21-doctor-YYYYMMDD-HHMMSS.json
```

Audio front-end evaluation has a matching report artifact path:

```bash
make audio-front-end-eval
A21_AUDIO_FIXTURE=reports/a21-audio-fixture.json make audio-front-end-eval
```

It writes:

```text
reports/a21-audio-front-end-eval-YYYYMMDD-HHMMSS.json
```

These reports are evidence artifacts for VAD/AEC evaluation. They do not contain raw audio frames. The current report contract is host-only and no-execute: it records `device_id=none_host_fixture`, `baseline_scope=host_only`, `provider_executed=false`, `v21_executed=false`, `hardware_executed=false`, and redaction booleans showing that raw PCM, base64 audio, transcripts, prompts, provider output, reasoning, credentials, full URLs, proxy URLs, and full local paths were not stored. The saved `report_path` is a basename only.

It checks what the current foundation can truthfully check:

- legacy env variable prefixes
- A21 endpoint env vars pointing at known legacy X21/V21 ports
- global proxy env presence without printing values
- direct-connect `NO_PROXY` / `A21_NO_PROXY` coverage when a global proxy exists
- legacy working directories
- A21 reserved port conflicts, except for a verified running A21 Gateway on
  the active launch service port `21080`
- minimum network/DNS fingerprint
- StackChan firmware manifest identity
- repository-local PlatformIO venv path
- repository-local PlatformIO pinned version, currently `6.1.19`
- repository-local PlatformIO core path
- validated firmware artifact count
- current git commit firmware artifact match
- local `/dev/cu.*` serial device inventory
- serial device USB-modem classification
- serial device process ownership via `lsof`
- selected voice provider local health and provider network mode
- voice provider registry readiness for mock, Doubao realtime, OpenAI realtime, Bailian/DashScope, and DeepSeek
- voice provider smoke readiness, without executing paid or external provider calls
- voice realtime WebSocket readiness plan, without dialing providers
- optional V21 adapter health when `A21_V21_ADAPTER_URL` is configured

The firmware section intentionally checks repository-local paths under `.a21-tools/`. This keeps A21 firmware tooling isolated from X21/V21 and from global PlatformIO state. If the local `pio` version is missing or not `6.1.19`, doctor warns; run `make firmware-tools` to create or repair the pinned toolchain.

The proxy section intentionally records only env variable names and direct-connect coverage labels. It blocks `HTTP_PROXY`, `HTTPS_PROXY`, or `ALL_PROXY` configurations that do not prove direct routing for localhost, loopback, `.local`, `10.0.0.0/8`, `10.21.0.0/16`, `172.16.0.0/12`, and `192.168.0.0/16`. `A21_PROVIDER_PROXY_URL` is reported separately as explicit provider egress configuration and is not treated as LAN bypass coverage.

Explicit LAN reachability is a separate command:

```bash
go run ./cmd/a21 lan-probe --target a21-gateway=127.0.0.1:21080 --output-dir reports
A21_LAN_TARGET=a21-gateway=127.0.0.1:21080 A21_LAN_SAMPLES=5 make lan-probe
```

`lan-probe` uses direct TCP dials, not HTTP clients or proxy-aware provider clients. It writes `reports/a21-lan-probe-YYYYMMDD-HHMMSS.json` and includes target status, passed/failed sample counts, p50, p95, jitter, current commit, network fingerprint, and redacted proxy-policy metadata. It is not part of default `doctor` because home development may not have StackChan or office-only endpoints online. It never stores proxy URLs, proxy credentials, URL credentials, API keys, or raw X21 target values.

`doctor` and `gate --scope host` still block unknown occupants of A21 reserved
ports. The only launch-validation exception is `21080` when a direct
`/healthz` probe returns `service=a21-gateway` and `status=ok`; this lets
product-chain, provider, V21, and physical evidence commands run against an
already-running Gateway without treating the required Gateway as a port
conflict. Other reserved ports, non-A21 services, and failed identity probes
remain blocking findings.

The voice section includes selected provider local health, Gateway runtime provider, and provider network mode. `direct` means future provider HTTP clients will not inherit environment proxies. `explicit_proxy` means `A21_PROVIDER_PROXY_URL` is configured; doctor reports only the variable name and never prints the proxy URL, host, port, username, or password.

Selected provider health is local configuration health, not external connectivity proof. For example, `A21_PROVIDER_PRIMARY=doubao_realtime` reports the Doubao realtime speech-to-speech provider object and missing/present required env state without dialing Volcengine; when configured it remains `degraded` with an execution guard until real S2S smoke is verified. `A21_PROVIDER_PRIMARY=doubao_tts_realtime` reports the separate TTS-only provider object. Gateway runtime remains on its explicit provider configuration path; doctor health alone does not switch Gateway turn handling away from mock.

`voice.gateway_provider` reports what Gateway would use at startup. If `A21_GATEWAY_VOICE_PROVIDER` is unset or `mock`, this remains `a21-mock-voice` even when real provider credentials are configured. `A21_GATEWAY_VOICE_PROVIDER=selected` is required before Gateway injects the provider selected by `A21_PROVIDER_PRIMARY`.

The voice provider registry is a readiness audit, not a real provider smoke test. It reports selected provider, required env names, present env names, and missing env names. It never prints API keys or model values. `A21_PROVIDER_PRIMARY` controls the selected provider in the report; unknown names are shown as `unknown_provider`, and legacy-looking provider names are shown as `invalid_legacy_provider` with blocking findings so raw X21/V21-looking values are never echoed back.

The voice smoke section is a dry-run plan inside `doctor`. It reports whether the selected provider has enough env to run a smoke test, which protocol would be used, which env variable names are involved, and which endpoint host would be contacted. It does not execute network calls and never prints key, model, proxy, or full URL values.

The `voice.realtime_plan` section is also dry-run. It currently supports OpenAI Realtime planning, Doubao end-to-end realtime speech-to-speech planning, and Doubao realtime TTS planning. It reports provider, protocol, readiness status, required env names, network mode, and endpoint host. It never prints API keys, app IDs, resource IDs, model values, voice IDs, auth headers, or full provider URLs, and it does not dial the provider. The same realtime plan can be inspected directly with `provider-realtime-plan` when an operator needs a smaller provider-only report.

Explicit provider smoke execution is a separate command:

```bash
go run ./cmd/a21 provider-smoke --provider deepseek --execute
go run ./cmd/a21 provider-smoke --provider deepseek --stream --repeat 3
go run ./cmd/a21 provider-smoke --provider deepseek --execute --stream --repeat 3 --output-dir reports
go run ./cmd/a21 provider-smoke --provider deepseek --output-dir reports
```

The provider catalog includes the PRD reference profiles for mainland text-stream candidates, local text providers, existing realtime references, and future agent-task bridges. Catalog visibility is not execution authorization: only `mock`, `deepseek`, and `local_ollama` are route-eligible in the built-in P0 provider-smoke path. `A21_PROVIDER_PROFILES_PATH` may point at a local JSON file containing additional A21-namespaced OpenAI-compatible text-stream profiles. A loaded profile can become route-eligible only when the profile is valid and explicitly sets `route_eligible=true`; invalid files produce redacted findings keyed to the env var name, not the file path or raw provider values. `local_ollama` is a direct local-fallback text-stream lane and still requires explicit `A21_LOCAL_OLLAMA_BASE_URL` plus `A21_LOCAL_OLLAMA_MODEL`. Realtime WebSocket providers such as OpenAI Realtime, Doubao realtime TTS, and Doubao end-to-end realtime voice remain redacted plan or fake-connection boundaries only until a dedicated explicit smoke command exists.

OpenAI-compatible text-stream providers that are not route-eligible can still
run `provider-smoke --execute --stream` for compatibility testing. These
reports keep `route_eligible=false`, so product readiness cannot absorb them as
launch evidence.

`provider-compat-matrix` rolls ASR, LLM, and TTS evidence into one redacted
local/cloud matrix:

For the current pinned 5080lab provider closure, use
`docs/engineering/A21_PROVIDER_5080LAB_TEST_PLAN_20260602.md`.

```bash
go run ./cmd/a21 provider-compat-matrix --use-latest-reports --output-dir reports
go run ./cmd/a21 provider-compat-matrix --provider-full-summary reports/a21-provider-full-YYYYMMDD-HHMMSS/a21-provider-full-summary.json --output-dir reports
go run ./cmd/a21 provider-compat-matrix --provider-audio-smoke-report reports/a21-provider-audio-smoke-cloud-asr.json --provider-audio-smoke-report reports/a21-provider-audio-smoke-cloud-tts.json --output-dir reports
```

Cloud ASR/TTS lab results should use schema `a21.provider_audio_smoke.v1` with
only low-information timing fields such as `asr_final_p95_ms` or
`tts_first_audio_p95_ms`, provider name, endpoint host, booleans, and a
basename report path. Do not include transcript text, provider output, raw
audio, prompt text, full URLs, proxy values, model values, or secrets.

Agent-task profiles are reported as readiness visibility only. `hermes_agent`
and `mimo_agent` stay in the `agent_task` family, are not route-eligible, and
are not realtime-capable. The lane is disabled by default and becomes locally
configured only when `A21_AGENT_PROVIDER_PRIMARY` explicitly selects one of
those profile names. `A21_PROVIDER_PRIMARY` does not configure the agent-task
lane and reports an invalid agent-task primary if pointed at `hermes_agent` or
`mimo_agent`. The package-level scaffold has fake `sse`, `http`, and `stdio`
event streams plus A21 semantic redaction tests. Real Hermes/MiMo execution is
limited to the explicit host-only `agent-io-smoke --execute` command; there is
still no doctor, Gateway runtime, realtime, V21, or hardware execution path.

```bash
go run ./cmd/a21 agent-plan --mode co_creation --agent hermes_agent --endpoint-url http://127.0.0.1:21130/a21/agent-task --output-dir reports
go run ./cmd/a21 agent-io-smoke --endpoint-url http://127.0.0.1:21130/a21/agent-task --output-dir reports
A21_HERMES_AGENT_URL=http://127.0.0.1:21130/a21/agent-task make agent-io-smoke-execute
A21_HERMES_AGENT_URL=http://127.0.0.1:8642/v1 A21_HERMES_AGENT_KEY=<redacted> make agent-io-smoke-execute
```

`agent-io-smoke` reports endpoint readiness without network I/O unless
`--execute` is present. With `--execute`, it uses a direct no-ambient-proxy HTTP
client and writes only redacted status, duration, safe content type, text
length, event count, planner markers, and redaction booleans.

When `--output-dir reports` is supplied, `provider-smoke` writes `reports/a21-provider-smoke-YYYYMMDD-HHMMSS-nnnnnnnnn.json`. The nanosecond suffix prevents concurrent smoke runs from overwriting each other. This report is redacted evidence for provider readiness or explicit smoke execution and now carries `schema_version=a21.provider_smoke.v1` plus `generated_at_ms`; the stored `report_path` is a basename only. It never stores API keys, model values, proxy URLs, full provider URLs, prompt text, generated content, or reasoning content. Streaming smoke records repeat count, first-byte, first-content, total-duration, fallback marker, and trace/metric names only.

`provider-realtime-plan` intentionally rejects `--execute`. It is not a smoke test and not connectivity proof; it is a redacted readiness plan.

Offline realtime fixture smoke is a separate fake-connection command:

```bash
go run ./cmd/a21 provider-realtime-fixture --provider doubao_tts_realtime --execute
go run ./cmd/a21 provider-realtime-fixture --provider openai_realtime --execute
```

It validates provider wrapper event flow without dialing a provider. It is still not connectivity, latency, audio-quality, or paid-provider proof.

`latency-bench --mock --output-dir reports` writes `reports/a21-latency-bench-YYYYMMDD-HHMMSS.json` and includes `report_path` in stdout. `make latency-bench` uses this mode so mock latency evidence is preserved for environment comparisons. The report also includes `generated_at`, `current_commit`, network/DNS fingerprint, and doctor-style redacted proxy-policy metadata. It reports env variable names such as `HTTPS_PROXY` or `A21_PROVIDER_PROXY_URL`, but never proxy values, hosts, ports, usernames, passwords, keys, or model IDs.

Real ASR/TTS/LLM/S2S provider latency comparison is governed by `docs/engineering/A21_PROVIDER_BENCHMARKS.md`. Until `provider-latency-bench` is promoted beyond mock/fixture scaffolding, provider comparisons must cite the existing A21 reports they used, such as `provider-smoke --stream`, `local-voice-loopback`, `stackchan-fast-companion-turn`, `audio-front-end-eval`, `latency-bench --mock`, or the scaffolded `provider-latency-bench` shape, and must list unmeasured stages explicitly.

`product-readiness` is the launch/demo status rollup. Provider env presence is
configuration only, not real-provider readiness. Launch readiness requires the
currently selected configured provider to be backed by an executed
`provider-smoke --execute --stream --repeat 3` report passed with
`--provider-smoke-report`, or found by `--use-latest-reports`. The rollup
accepts only non-mock route-eligible text-stream reports with passed status,
matching current provider selection, three or more successful streaming
attempts, first-byte/first-content timing, no fallback marker, and no forbidden
prompt/transcript/output/reasoning/full URL/proxy/key/local-path fields. For
`--use-latest-reports`, newer provider-smoke reports for a different selected
provider are skipped with basename-only findings so an older matching 5080lab
report can still close the provider gap. `provider-evidence-package` and
`provider-evidence-import` enforce the same selected-provider match when the
operator environment contains a configured selected route-eligible text provider.
For 5080lab handoff, generate the exact print-only operator bundle with
`make provider-5080lab-runbook A21_PROVIDER=<selected-provider>`; it refuses
blank or `mock` providers and prints the lab smoke, package, return, import,
and product-readiness commands without executing provider traffic on the control
machine.
For custom wake words, run `wake-word-firmware-build-receipt` after a reviewed
xiaozhi/ESP-SR build and pass `--review-report <review.json>` (or
`--build-review`) so the resulting `a21-wake-word-build.json` names the
reviewed-build report by basename. Then pass the receipt to
`wake-word-firmware-package --build-receipt`; both remain no-flash,
below-activation evidence until a guarded hardware-window flash and physical
wake proof pass.
A21 must not package or flash from the frozen external X21 `xiaozhi-esp32`
checkout; the firmware flash, wake-word receipt, and wake-word package commands
reject build directories from that source.
For V21, it treats
`A21_V21_ADAPTER_URL` health as adapter availability only; launch readiness also
requires an executed `v21-adapter-smoke --execute` report passed with
`--v21-adapter-smoke-report`, or an external-Gateway
`a21.xiaozhi_professional_bench.v1` report passed through
`--v21-professional-report` after Gateway traces prove the V21 query markers.
`--use-latest-reports` scans the selected output directory for the latest
known A21 provider-smoke, voice, professional, adapter-smoke, physical evidence,
and wake-word firmware-plan/package/physical-acceptance reports, then ingests
them through the same explicit report contracts.
`--wake-word-firmware-plan <report.json>` is the explicit equivalent for custom
wake-word firmware planning evidence. `wake-word-firmware-package` output is
lower than activation: it proves only that a reviewed xiaozhi/ESP-SR build was
wrapped into A21-named artifact files. After the guarded hardware-window flash
and operator custom-wake observation are complete,
`wake-word-physical-proof --physical-device-online --firmware-flash-executed
--guarded-flash-report <guarded-flash.json> --operator-observed
--wake-phrase-matched --false-wake-rejected --stock-wake-rejected --output-dir
reports` records the redacted operator proof. Then
`wake-word-physical-acceptance --package-report <package.json> --proof-report
<physical-proof.json> --output-dir reports` writes the accepted wake proof that
can close `wake_word.product_ready`. The rollup
ingests only fixed status/count/timing fields, keeps `prd_accepted=false`, and
never stores query text, answer text, evidence bodies, full URLs, credentials,
proxy values, or local paths. If physical StackChan is currently offline but a
valid candidate physical report exists, `next_actions` still carries the
missing audible/playback/barge-in evidence rather than collapsing the result
to only "bring the device online". For local speech, it now
recognizes either explicit `A21_SHERPA_ONNX_MODEL_DIR` /
`A21_SHERPA_ONNX_ASR_MODEL_DIR` values or the repository-local `.a21-tools`
sherpa-onnx model caches when their required model files are present. This is a
static readiness check only: it never stores full local paths and does not
execute ASR/TTS. Execution evidence still comes from `local-tts-smoke`,
`local-asr-smoke`, `local-voice-loopback`, and physical StackChan receipts.
When the default repo-local ASR cache is present and no explicit ASR provider
env is set, product-readiness reports `sherpa_onnx` as the host-local ASR
candidate so the remaining launch gaps focus on provider/V21/hardware evidence
instead of re-asking for an already installed local model.

When the Gateway/simulator, executed provider smoke, executed professional V21
evidence, host Xiaozhi/local voice loopback evidence, and wake-word runtime are
all ready, `product-readiness` also emits a `server_side` block with
`candidate_ready=true` and top-level `status=server_side_candidate_ready`.
This is the no-hardware server candidate gate: source report fields are
basenames only, `requires_physical_acceptance=true` preserves the hardware
gate, and `launch_ready` remains false until physical StackChan playback,
microphone, wake-word, and PRD acceptance evidence are present. Missing pieces
are reported as fixed `server_side.missing_evidence` labels such as
`provider_smoke`, `v21_professional_smoke`, `host_voice_loopback`, or
`wake_word`.

`server-side-readiness-bundle` is the control-tower wrapper for that no-hardware
gate:

```bash
go run ./cmd/a21 server-side-readiness-bundle --use-latest-reports --output-dir reports
make server-side-readiness-bundle
make server-side-readiness-collect
```

It writes `reports/a21-server-side-readiness-bundle-YYYYMMDD-HHMMSS.json` with
schema `a21.server_side_readiness_bundle.v1`. The bundle summarizes the
redacted provider-smoke, V21 professional, host voice loopback, Gateway, and
wake-word signals plus fixed missing-evidence labels and safe collection
commands. `--require-candidate` returns nonzero when the server-side candidate
chain is incomplete. It is still a no-hardware artifact: it stores only
basename source reports and redaction booleans, keeps `prd_accepted=false`, and
does not replace physical StackChan launch acceptance.
With `--collect-missing`, the command may collect missing host-side Xiaozhi
voice-loopback evidence through the Gateway and then rebuild the bundle from
the refreshed latest reports. It does not execute provider or V21 network
smokes unless the operator also passes `--execute-provider-smoke` and/or
`--execute-v21-smoke`; otherwise those collection steps are recorded as
`skipped` with fixed reasons. Collection output stores only step names, fixed
status/reason codes, safe command labels, and basename source reports; it never
stores subcommand stdout, stderr, full URLs, credentials, transcripts, prompts,
provider output, or evidence bodies.

`audio-front-end-plan` and `audio-front-end-eval` now expose a machine-readable Fast Companion VAD/AEC adapter evidence shape. WebRTC APM, ESP-SR, provider-side VAD, and Silero VAD runtime candidates are placeholders or unavailable until a later authorized adapter or hardware window supplies evidence. The A21 RMS detector remains an available host-only development baseline, not a production candidate.

`provider-latency-bench` now exists as a mock/fixture scaffold for the shared
candidate-chain report. It accepts `--provider`, `--fixture`, `--mode
mock|fixture|host_loopback`, `--iterations`, and `--output-dir`, but it
intentionally rejects `--execute`. The report includes A21 trace/session/device
IDs, provider profile/family/protocol labels, redacted network/proxy metadata,
ASR/provider/TTS/downlink/device/barge-in placeholder timings, p50/p95/p99
summaries, fallback/failure counts, execution flags showing no provider/V21/
hardware execution, and `promotion_gate=not_production`. The hardened report
also includes machine-readable `metric_terms`, `canonical_metrics`, and
`stage_availability` fields for TTFS/TTFT/FTTS/TTFA and A21 canonical metric
comparison. Current stages remain `available=false` placeholders, including
ASR first partial, provider first byte/content, TTS first audio, legacy
downlink first frame, audio downlink first frame, device playback start,
barge-in stop, provider cancel, and playback stop. Fixture reports store only
the fixture basename, not the full local path.
A JSON fixture sidecar can contribute `a21.provider_latency_fixture.v1` audio
metadata: fixture identity, format, sample rate, channel count, duration, sample
count, window length, and window count. Missing, invalid, oversized,
unknown-field, payload-bearing, or unsafe sidecars are reported as structured
redacted findings and increment the failure count without panicking or echoing
raw error text. The report never stores prompt, transcript, provider output,
reasoning, key values, full URLs, proxy URLs, full local paths, raw PCM, or
base64 audio payloads. This is report-contract/metric-shape hardening only; it
does not authorize provider execute, V21 execute, Gateway runtime startup,
binary Opus transport, AEC adapter work, WebRTC/ESP-SR native adapters, or
hardware acceptance.

Binary Opus media transport remains a planned direction captured in the live
protocol and mature-voice contracts. There is no `doctor` Opus runtime check,
no Gateway startup, no native codec probe, and no production dependency.
A future doctor check may report binary media readiness only after the
wire-format compatibility fixture, Gateway loopback fixture, encoder/decoder
adapter spike, LAN jitter/fallback report, CPU/memory profile, device playback
receipt, and hardware-window acceptance each have redacted A21 evidence. Until
then, doctor output must not imply that Opus transport is available or accepted.

The V21 section is skipped when `A21_V21_ADAPTER_URL` is unset. When set, doctor probes `/healthz` on the adapter boundary through a direct no-ambient-proxy HTTP client and reports `healthy` or `unhealthy`. It does not print adapter credentials or raw secret-bearing URLs in findings.

V21 adapter query smoke is intentionally a separate command, not a doctor side effect:

```bash
go run ./cmd/a21 v21-adapter-smoke --output-dir reports
A21_V21_ADAPTER_URL=http://127.0.0.1:21121 make v21-adapter-smoke-execute
```

Without `--execute`, it only reports whether an adapter URL is configured and
writes `reports/a21-v21-adapter-smoke-YYYYMMDD-HHMMSS.json` when requested.
With `--execute`, it posts the professional query contract to `/a21/v21/query`
through a direct no-ambient-proxy HTTP client. The report records
`schema_version=a21.v21_adapter_smoke.v1`, `generated_at_ms`, endpoint host,
fixed health/query paths, `mode=professional`, `latency_profile=fast_first`,
`answer_style=voice_first_with_citations`,
`privacy_scope=professional_only`, `max_first_response_ms=1200`, duration,
confidence, response counts, and `redaction_ok`; the saved `report_path` is a
basename only. It never stores query text, answer text, evidence summaries,
document quotes, full adapter URLs, credentials, proxy URLs, or API keys.

`serial-list` emits just the serial inventory portion for physical-device prep. It does not flash, provision, reset, or open a serial monitor.

Upload and flash-plan guards accept only explicit USB serial-looking ports. On macOS that means `cu.*`/`tty.*` names containing `usbmodem` or `usbserial`; Bluetooth and debug-console paths are intentionally rejected even though they appear in the serial inventory.

Official StackChan speaker smoke uses a separate guarded flash family:

```bash
make stackchan-official-audio-smoke-build
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX make stackchan-official-audio-smoke-flash-plan
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE \
make stackchan-official-audio-smoke-flash-execute
```

This lane is allowed only for the official codec speaker baseline. It builds from official StackChan Git `HEAD`, records hashes from IDF `flash_args`, and requires the same explicit USB serial-port discipline. It is not production A21 firmware and cannot replace `firmware-package`, `firmware-check --kind upload`, or steady-state `firmware-flash-plan`.

`stackchan-official-pcm-bridge-build` is the matching build lane for M3 downlink preparation. It exports official StackChan Git `HEAD`, applies the A21 PCM bridge overlay, and must produce `a21-stackchan-official-pcm-bridge.bin` at app offset `0x20000`. `stackchan-official-pcm-bridge-nvs` is no-write by default and records the NVS partition offset `0x9000`, size `0x4000`, explicit USB serial readiness, `device_id`, and redacted audio-websocket endpoint fields. `stackchan-official-pcm-bridge-nvs --execute` requires `WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS`, backs up current NVS, preserves existing entries including servo calibration, mutates only `a21/device_id` plus `a21/audio_ws_url`, and writes only the NVS partition. `stackchan-official-pcm-bridge-flash` is no-write by default and records the bridge app hash plus required flash parts. `stackchan-official-pcm-bridge-flash --execute` is the guarded bridge app flashing lane and must consume the normal explicit USB serial, control gate, and confirmation-token guards.

`namespace-audit` scans tracked file paths and blocks X21/V21-looking runtime paths outside the explicit V21 adapter/docs boundary. It is part of `make release-check` so path-level project identity drift is caught before merge.

`firmware-check --kind current-artifact` validates the newest packaged artifact for the current git commit from `a21-firmware-release-index.jsonl`, then re-runs artifact, release-index, and per-artifact manifest checks. The release index and per-artifact manifest must point to the same checked artifact and checksum paths; a same-named binary outside the selected artifact directory is rejected. It is part of `make release-check` immediately after `firmware-package`.

The firmware section's `current_artifact_path` is populated only through the same release-ledger guard. Loose `.bin + .sha256` files may be counted and individually checksum-checked, but they are not current release candidates without `a21-firmware-release-index.jsonl` and the sibling artifact manifest.

`firmware-artifact-prune-plan` is the no-delete artifact retention receipt. It uses the release-ledger current artifact guard, keeps the current package plus recent release-ledger-valid packages, lists older valid packages as prune candidates, and lists loose or incomplete packages for manual review. It writes `reports/a21-firmware-artifact-prune-plan-YYYYMMDD-HHMMSS.json`, prints only a summary when `--output-dir` is set, and always sets `delete_allowed=false`.

`office-handoff` writes `reports/a21-office-handoff-YYYYMMDD-HHMMSS.json`. It is the home-to-office transfer manifest: current release-ledger artifact, artifact-retention summary, serial inventory, and explicit next physical acceptance actions. It never contacts a model provider, never contacts V21, never contacts Gateway, and sets both `flash_allowed=false` and `delete_allowed=false`.

`firmware-device-report` fetches Gateway `/v1/devices` through an A21 direct HTTP client and writes `reports/a21-devices-YYYYMMDD-HHMMSS.json`. It rejects known X21/V21 legacy ports before dialing, then requires the response to declare `schema_version=a21.gateway.devices.v1` and `service=a21-gateway` before any device identity is accepted. Use this instead of hand-written curl captures before device identity or flash-plan checks.

`office-preflight` is the no-flash现场验收入口 for taking A21 into the office. It composes the newest current-commit firmware artifact, direct Gateway `/v1/devices` capture, Gateway identity check, device identity freshness check, flash-plan-equivalent device quiescence check, proxy/fingerprint metadata, and serial inventory into `reports/a21-office-preflight-YYYYMMDD-HHMMSS.json`. It also writes the paired `a21-devices-YYYYMMDD-HHMMSS.json` used by downstream guards. It still sets `flash_allowed=false`; success only means the next step may be `firmware-flash-plan` with an explicit USB serial port.

`office-acceptance` reads an `a21-office-handoff` report and an `a21-office-preflight` report, plus an optional `a21-firmware-flash-plan` report. It writes `reports/a21-office-acceptance-YYYYMMDD-HHMMSS.json` and cross-checks schema, no-flash/no-delete state, commit, artifact, SHA, and device consistency. A passing status is `ready_for_physical_acceptance`; it still does not flash or claim the physical device has been accepted.

`stackchan-accept --check identity` reads an `a21-office-acceptance` report, then takes a fresh direct Gateway `/v1/devices` capture and serial inventory. It writes `reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json` with `hardware_acceptance_scope=identity_only` and cross-checks the current release-ledger artifact, device ID, firmware identity, freshness, quiescent state, and available USB serial candidate. A passing status is `identity_confirmed`; it still does not prove microphone, speaker, screen, servo, RGB, OTA, latency, or real flashing acceptance.

`stackchan-accept --check mic-probe` is a diagnostic-only physical microphone gate for the isolated `a21_stackchan_cores3_mic_probe` firmware lane. It fetches Gateway `/v1/devices` and `/metrics` with direct no-proxy HTTP, requires the device to report `microphone=diagnostic_probe_m5unified_i2s_capture`, checks firmware identity and commit, validates runtime mic counters/sample evidence, verifies Gateway ingress/VAD metrics, and writes `reports/a21-stackchan-mic-probe-acceptance-YYYYMMDD-HHMMSS.json`. With `--window-ms`, it commands `LISTENING + audio_probe_only`, compares before/after deltas, calculates mic capture rate plus microphone-to-audio-WS and audio-WS-to-Gateway delivery ratios, and clears back to `IDLE`, so previous cumulative playback or audio counters do not contaminate this probe. The success status is `mic_probe_acceptance_status=confirmed` with `production_capability_promoted=false`; it does not claim production microphone availability, AEC, full-duplex, provider latency, or speaker acceptance.

`stackchan-accept --check half-duplex` is the first physical mic-to-speaker loop gate. It snapshots `/v1/devices` and `/metrics`, commands `LISTENING` without `audio_probe_only`, arms exactly one `mock_playback_on_next_audio_frame`, then waits for real StackChan microphone uplink frames to trigger the Gateway mock turn. It requires Gateway audio ingress, Gateway playback chunks, firmware playback-buffer deltas, and speaker-pump deltas to move in the same traced window before clearing back to `IDLE`. It writes `reports/a21-stackchan-half-duplex-acceptance-YYYYMMDD-HHMMSS.json` with `hardware_acceptance_scope=mic_to_mock_playback` and `physical_sound_observed=false`. Passing means the real device can drive a minimal half-duplex A21 loop through Gateway mock playback instrumentation; it still does not claim real ASR/LLM/TTS, AEC, full-duplex, or human-accepted product audio quality.

`stackchan-accept --check imu-probe` is the read-only IMU telemetry gate. It snapshots `/v1/devices`, waits a bounded window, snapshots again, and requires the device to report `imu=diagnostic_probe_m5unified_imu`, increasing sample counters, zero new read errors by default, non-zero acceleration evidence, and a concrete posture string. It writes `reports/a21-stackchan-imu-probe-acceptance-YYYYMMDD-HHMMSS.json` with `hardware_acceptance_scope=diagnostic_imu_only` and `production_capability_promoted=false`. Passing means IMU telemetry is alive in the isolated diagnostic build; it does not approve product gestures or release-firmware IMU promotion.

`stackchan-accept --check sensor-probe` is the read-only ambient/proximity/battery telemetry gate. It snapshots `/v1/devices`, waits a bounded window, snapshots again, and requires the device to report `ambient_light=diagnostic_probe_ltr553_ambient_light`, `proximity=diagnostic_probe_ltr553_proximity`, and `battery=diagnostic_probe_ina226_battery`, increasing sensor sample counters, zero new read errors by default, and battery voltage above the configured threshold. It writes `reports/a21-stackchan-sensor-probe-acceptance-YYYYMMDD-HHMMSS.json` with `hardware_acceptance_scope=diagnostic_sensor_only` and `production_capability_promoted=false`. Passing means the isolated sensor diagnostic path is alive; it does not approve adaptive brightness, presence behavior, power-state UI, or release-firmware promotion.

`stackchan-accept --check speaker` is an instrumented speaker/downlink gate for a connected StackChan. It snapshots `/v1/devices` and `/metrics`, commands `SPEAKING` with bounded non-silent `audio.playback.chunk` frames through `/v1/devices/control`, waits the requested window, checks runtime echo deltas for playback buffer and M5 speaker pump counters, checks Gateway playback-chunk metrics, then sends `IDLE`. Its default `--mock-audio-chunks 50` sends about 1000 ms of 20 ms mock PCM chunks, while the default `--window-ms 1500` leaves margin for the final speaker-pump frame and runtime echo update. The CLI batches that long probe into bounded `/v1/devices/control` requests of at most four chunks each. This keeps the Gateway per-request safety contract intact while making the probe long enough for an operator to hear. It writes `reports/a21-stackchan-speaker-acceptance-YYYYMMDD-HHMMSS.json` with `expected_audio_duration_ms` and `physical_sound_observed=false`: passing means Gateway downlink, firmware buffering, and speaker pump instrumentation moved for the commanded stream, not that a human heard final product-quality audio.

`xiaozhi-instrument-observation` writes a redacted `reports/a21-xiaozhi-instrument-observation-YYYYMMDD-HHMMSS.NNNNNNNNN.json` sidecar for the physical xiaozhi evidence window. It takes the captured `trace_id`, `session_id`, `device_id`, Gateway answer-first-downlink timing, downlink-to-audible timing, RMS/noise-floor values, and optional trusted playback-start observation, then derives `speech_end_to_first_audible_response_ms`. It does not contact Gateway, does not store raw audio, transcripts, provider output, URLs, proxy values, credentials, or local paths. `xiaozhi-physical-evidence --instrument-observation-report <report>` remains the authority that cross-checks this sidecar against the live Gateway trace before any physical evidence can move beyond candidate status.

`firmware-check --kind device` validates a captured Gateway `/v1/devices` report against the packaged firmware artifact, expected device ID, expected git commit, required A21 Gateway identity, required online status, and required freshness window. It accepts the raw Gateway identity fields or the `gateway_schema_version` / `gateway_service` fields written by `firmware-device-report`; it rejects naked hand-written `devices` JSON. The Makefile wrapper passes `--max-device-age-ms 300000` by default so a stale or offline device report cannot become part of flash-plan evidence. It also does not flash, provision, reset, or open a serial monitor.

`firmware-flash-plan` composes the artifact, upload-port, and device-identity guards into a single no-flash receipt. The Makefile wrapper writes `reports/a21-firmware-flash-plan-YYYYMMDD-HHMMSS.json`, includes `generated_at_ms` and `report_path`, and still sets `flash_allowed: false`.

The flash-plan guard also refuses Gateway reports that show `current_expression=speaking`, a non-empty `playback_stream_id`, `current_expression=thinking|professional|error`, or `current_mode=professional|local_fallback|error`. Fresh identity is not enough; firmware operations must not be planned while the device is actively playing, speaking, thinking, in professional mode, or in a failure/fallback state.

## Exit Codes

Current behavior:

- `0`: required checks pass
- `1`: a blocking check fails
- `2`: CLI usage error, such as unknown command or invalid doctor flag

Warnings-only exit code is reserved for a later expanded doctor.

## Future Expansion

The full doctor should eventually add human-readable table output and cover:

- project identity and namespace
- Go/Node/Python tooling
- A21 ports
- StackChan LAN reachability
- provider connectivity and credential presence
- mock audio loop
- metrics/trace availability
- firmware compile status and guarded physical-upload readiness

Do not add fake checks. A doctor item should be implemented only when the repository has the subsystem it claims to check.
