# A21 Doctor

## Current Command

```bash
go run ./cmd/a21 doctor
go run ./cmd/a21 doctor --output-dir reports
go run ./cmd/a21 serial-list
go run ./cmd/a21 firmware-device-check --artifact firmware/artifacts/<a21-stackchan...bin> --device-report reports/a21-devices.json --device-id stackchan-001 --commit <git-sha>
make doctor
```

`doctor` emits the preflight report, A21 firmware/tooling status, and also writes a timestamped JSON file:

```text
reports/a21-doctor-YYYYMMDD-HHMMSS.json
```

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
- repository-local PlatformIO core path
- validated firmware artifact count
- current git commit firmware artifact match
- local `/dev/cu.*` serial device inventory
- serial device USB-modem classification
- serial device process ownership via `lsof`
- voice provider health and provider network mode
- voice provider registry readiness for mock, Doubao realtime, OpenAI realtime, Bailian/DashScope, and DeepSeek
- voice provider smoke readiness, without executing paid or external provider calls
- voice realtime WebSocket readiness plan, without dialing providers
- optional V21 adapter health when `A21_V21_ADAPTER_URL` is configured

The firmware section intentionally checks repository-local paths under `.a21-tools/`. This keeps A21 firmware tooling isolated from X21/V21 and from global PlatformIO state.

The proxy section intentionally records only env variable names and direct-connect coverage labels. It blocks `HTTP_PROXY`, `HTTPS_PROXY`, or `ALL_PROXY` configurations that do not prove direct routing for localhost, loopback, `.local`, `10.0.0.0/8`, `10.21.0.0/16`, `172.16.0.0/12`, and `192.168.0.0/16`. `A21_PROVIDER_PROXY_URL` is reported separately as explicit provider egress configuration and is not treated as LAN bypass coverage.

The voice section includes provider network mode. `direct` means future provider HTTP clients will not inherit environment proxies. `explicit_proxy` means `A21_PROVIDER_PROXY_URL` is configured; doctor reports only the variable name and never prints the proxy URL, host, port, username, or password.

The voice provider registry is a readiness audit, not a real provider smoke test. It reports selected provider, required env names, present env names, and missing env names. It never prints API keys or model values. `A21_PROVIDER_PRIMARY` controls the selected provider in the report; unknown names are shown as `unknown_provider`, and legacy-looking provider names are shown as `invalid_legacy_provider` with blocking findings so raw X21/V21-looking values are never echoed back.

The voice smoke section is a dry-run plan inside `doctor`. It reports whether the selected provider has enough env to run a smoke test, which protocol would be used, which env variable names are involved, and which endpoint host would be contacted. It does not execute network calls and never prints key, model, proxy, or full URL values.

The `voice.realtime_plan` section is also dry-run. It currently supports OpenAI Realtime planning and Doubao realtime TTS planning. It reports provider, protocol, readiness status, required env names, network mode, and endpoint host. It never prints API keys, model values, voice IDs, auth headers, or full provider URLs, and it does not dial the provider.

Explicit provider smoke execution is a separate command:

```bash
go run ./cmd/a21 provider-smoke --provider deepseek --execute
go run ./cmd/a21 provider-smoke --provider bailian_dashscope --execute
```

Only OpenAI-compatible Chat Completions smoke is executable in this phase. Realtime WebSocket providers such as OpenAI Realtime, Doubao realtime TTS, and Doubao end-to-end realtime voice are reported as non-executable smoke targets; their realtime plans and adapters stay dry-run or fake-connection only until a dedicated explicit smoke command exists.

The V21 section is skipped when `A21_V21_ADAPTER_URL` is unset. When set, doctor probes `/healthz` on the adapter boundary and reports `healthy` or `unhealthy`. It does not print adapter credentials or raw secret-bearing URLs in findings.

`serial-list` emits just the serial inventory portion for physical-device prep. It does not flash, provision, reset, or open a serial monitor.

`firmware-device-check` validates a captured Gateway `/v1/devices` report against the packaged firmware artifact, expected device ID, and expected git commit. It also does not flash, provision, reset, or open a serial monitor.

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
