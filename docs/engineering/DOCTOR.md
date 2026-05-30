# A21 Doctor

## Current Command

```bash
go run ./cmd/a21 doctor
go run ./cmd/a21 doctor --output-dir reports
go run ./cmd/a21 serial-list
go run ./cmd/a21 firmware-device-check --artifact firmware/artifacts/<a21-stackchan...bin> --device-report reports/a21-devices.json --device-id stackchan-001 --commit <git-sha>
go run ./cmd/a21 provider-realtime-plan --provider doubao_realtime
go run ./cmd/a21 provider-realtime-plan --provider doubao_tts_realtime
go run ./cmd/a21 v21-adapter-smoke --output-dir reports
go run ./cmd/a21 v21-adapter-smoke --execute --output-dir reports
go run ./cmd/a21 lan-probe --target a21-gateway=127.0.0.1:21080 --output-dir reports
make doctor
make lan-probe
make provider-realtime-plan
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

These reports are evidence artifacts for VAD/AEC evaluation. They do not contain raw audio frames.

It checks what the current foundation can truthfully check:

- legacy env variable prefixes
- A21 endpoint env vars pointing at known legacy X21/V21 ports
- global proxy env presence without printing values
- direct-connect `NO_PROXY` / `A21_NO_PROXY` coverage when a global proxy exists
- legacy working directories
- A21 reserved port conflicts
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
A21_LAN_TARGET=a21-gateway=127.0.0.1:21080 make lan-probe
```

`lan-probe` uses direct TCP dials, not HTTP clients or proxy-aware provider clients. It writes `reports/a21-lan-probe-YYYYMMDD-HHMMSS.json` and includes target status, duration, current commit, network fingerprint, and redacted proxy-policy metadata. It is not part of default `doctor` because home development may not have StackChan or office-only endpoints online. It never stores proxy URLs, proxy credentials, URL credentials, API keys, or raw X21 target values.

The voice section includes selected provider local health, Gateway runtime provider, and provider network mode. `direct` means future provider HTTP clients will not inherit environment proxies. `explicit_proxy` means `A21_PROVIDER_PROXY_URL` is configured; doctor reports only the variable name and never prints the proxy URL, host, port, username, or password.

Selected provider health is local configuration health, not external connectivity proof. For example, `A21_PROVIDER_PRIMARY=doubao_tts_realtime` reports the Doubao realtime TTS provider object and missing/present required env state without dialing Volcengine. Gateway runtime remains on its explicit provider configuration path; doctor health alone does not switch Gateway turn handling away from mock.

`voice.gateway_provider` reports what Gateway would use at startup. If `A21_GATEWAY_VOICE_PROVIDER` is unset or `mock`, this remains `a21-mock-voice` even when real provider credentials are configured. `A21_GATEWAY_VOICE_PROVIDER=selected` is required before Gateway injects the provider selected by `A21_PROVIDER_PRIMARY`.

The voice provider registry is a readiness audit, not a real provider smoke test. It reports selected provider, required env names, present env names, and missing env names. It never prints API keys or model values. `A21_PROVIDER_PRIMARY` controls the selected provider in the report; unknown names are shown as `unknown_provider`, and legacy-looking provider names are shown as `invalid_legacy_provider` with blocking findings so raw X21/V21-looking values are never echoed back.

The voice smoke section is a dry-run plan inside `doctor`. It reports whether the selected provider has enough env to run a smoke test, which protocol would be used, which env variable names are involved, and which endpoint host would be contacted. It does not execute network calls and never prints key, model, proxy, or full URL values.

The `voice.realtime_plan` section is also dry-run. It currently supports OpenAI Realtime planning, Doubao end-to-end realtime speech-to-speech planning, and Doubao realtime TTS planning. It reports provider, protocol, readiness status, required env names, network mode, and endpoint host. It never prints API keys, app IDs, resource IDs, model values, voice IDs, auth headers, or full provider URLs, and it does not dial the provider. The same realtime plan can be inspected directly with `provider-realtime-plan` when an operator needs a smaller provider-only report.

Explicit provider smoke execution is a separate command:

```bash
go run ./cmd/a21 provider-smoke --provider deepseek --execute
go run ./cmd/a21 provider-smoke --provider bailian_dashscope --execute
go run ./cmd/a21 provider-smoke --provider deepseek --output-dir reports
```

Only OpenAI-compatible Chat Completions smoke is executable in this phase. Realtime WebSocket providers such as OpenAI Realtime, Doubao realtime TTS, and Doubao end-to-end realtime voice are reported as non-executable smoke targets; their realtime plans and adapters stay dry-run or fake-connection only until a dedicated explicit smoke command exists.

When `--output-dir reports` is supplied, `provider-smoke` writes `reports/a21-provider-smoke-YYYYMMDD-HHMMSS.json`. This report is redacted evidence for provider readiness or explicit smoke execution. It never stores API keys, model values, proxy URLs, or full provider URLs.

`provider-realtime-plan` intentionally rejects `--execute`. It is not a smoke test and not connectivity proof; it is a redacted readiness plan.

Offline realtime fixture smoke is a separate fake-connection command:

```bash
go run ./cmd/a21 provider-realtime-fixture --provider doubao_tts_realtime --execute
go run ./cmd/a21 provider-realtime-fixture --provider openai_realtime --execute
```

It validates provider wrapper event flow without dialing a provider. It is still not connectivity, latency, audio-quality, or paid-provider proof.

`latency-bench --mock --output-dir reports` writes `reports/a21-latency-bench-YYYYMMDD-HHMMSS.json` and includes `report_path` in stdout. `make latency-bench` uses this mode so mock latency evidence is preserved for environment comparisons. The report also includes `generated_at`, `current_commit`, network/DNS fingerprint, and doctor-style redacted proxy-policy metadata. It reports env variable names such as `HTTPS_PROXY` or `A21_PROVIDER_PROXY_URL`, but never proxy values, hosts, ports, usernames, passwords, keys, or model IDs.

The V21 section is skipped when `A21_V21_ADAPTER_URL` is unset. When set, doctor probes `/healthz` on the adapter boundary through a direct no-ambient-proxy HTTP client and reports `healthy` or `unhealthy`. It does not print adapter credentials or raw secret-bearing URLs in findings.

V21 adapter query smoke is intentionally a separate command, not a doctor side effect:

```bash
go run ./cmd/a21 v21-adapter-smoke --output-dir reports
A21_V21_ADAPTER_URL=http://127.0.0.1:21121 make v21-adapter-smoke-execute
```

Without `--execute`, it only reports whether an adapter URL is configured and writes `reports/a21-v21-adapter-smoke-YYYYMMDD-HHMMSS.json` when requested. With `--execute`, it posts the professional query contract to `/a21/v21/query` through a direct no-ambient-proxy HTTP client. The report records status, endpoint host, fixed paths, duration, confidence, and response counts. It never stores query text, answer text, evidence summaries, document quotes, full adapter URLs, credentials, proxy URLs, or API keys.

`serial-list` emits just the serial inventory portion for physical-device prep. It does not flash, provision, reset, or open a serial monitor.

Upload and flash-plan guards accept only explicit USB serial-looking ports. On macOS that means `cu.*`/`tty.*` names containing `usbmodem` or `usbserial`; Bluetooth and debug-console paths are intentionally rejected even though they appear in the serial inventory.

`namespace-audit` scans tracked file paths and blocks X21/V21-looking runtime paths outside the explicit V21 adapter/docs boundary. It is part of `make release-check` so path-level project identity drift is caught before merge.

`firmware-current-artifact-check` validates the newest packaged artifact for the current git commit from `a21-firmware-release-index.jsonl`, then re-runs artifact, release-index, and per-artifact manifest checks. It is part of `make release-check` immediately after `firmware-package`.

`firmware-device-report` fetches Gateway `/v1/devices` through an A21 direct HTTP client and writes `reports/a21-devices-YYYYMMDD-HHMMSS.json`. Use this instead of hand-written curl captures before device identity or flash-plan checks.

`firmware-device-check` validates a captured Gateway `/v1/devices` report against the packaged firmware artifact, expected device ID, expected git commit, and optional freshness window. The Makefile wrapper passes `--max-device-age-ms 300000` by default so a stale device report cannot become part of flash-plan evidence. It also does not flash, provision, reset, or open a serial monitor.

`firmware-flash-plan` composes the artifact, upload-port, and device-identity guards into a single no-flash receipt. The Makefile wrapper writes `reports/a21-firmware-flash-plan-YYYYMMDD-HHMMSS.json`, includes `generated_at_ms` and `report_path`, and still sets `flash_allowed: false`.

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
