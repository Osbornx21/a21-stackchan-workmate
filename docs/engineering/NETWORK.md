# A21 Network And Proxy Rules

## Current Reality

A21 is being developed in China mainland conditions, with home development first and Shanghai office acceptance later. The current home network has active proxy/VPN behavior and external DNS mapping into `198.18.0.x`.

Phase 1 preflight records the default network interface and an external DNS probe IP. This is not yet a latency benchmark.

## Boundary

StackChan must not know provider network details. Firmware connects only to A21 Gateway/Core on LAN. Gateway/Core owns:

- provider access
- proxy policy
- V21 adapter access
- network diagnostics
- retry/fallback behavior

Firmware Wi-Fi credentials are not provider credentials, but they are still
local secrets. They must not be committed into `platformio.ini`, firmware docs,
logs, doctor reports, or artifact names. Default firmware builds may contain no
SSID and must enter device-side Wi-Fi provisioning, not a dead local fallback.
The product lane follows Xiaozhi's startup model: use stored NVS SSIDs first,
then enter a build-selected provisioning method. Hotspot/SoftAP captive portal
is the A21 product default; BluFi and acoustic provisioning remain named
build-time alternatives. Local hardware bring-up may still use ignored
`firmware/stackchan/include/a21_firmware_secrets.local.h`, but provisioning
must preserve StackChan calibration/NVS keys and stay A21-namespaced.

## Direct-Connect Set

These targets must not silently route through global proxies:

- `localhost`
- `127.0.0.1`
- `::1`
- `.local`
- `10.0.0.0/8`
- `10.21.0.0/16`
- `172.16.0.0/12`
- `192.168.0.0/16`

Future provider adapters may use explicit proxy settings, but LAN/device/V21-local traffic must stay direct unless a migration/audit command says otherwise.

Agent I/O endpoint smoke follows the same direct-connect rule. Local Hermes or
MiMo bridges configured by `A21_HERMES_AGENT_URL` or
`A21_AGENT_IO_ENDPOINT_URL` must not inherit ambient `HTTP_PROXY`,
`HTTPS_PROXY`, or `ALL_PROXY`.

## Current Proxy Guard

`preflight` and `doctor` now evaluate a proxy policy report. When any global proxy env var is configured (`HTTP_PROXY`, `HTTPS_PROXY`, or `ALL_PROXY`), A21 requires `NO_PROXY` or `A21_NO_PROXY` to cover the full direct-connect set above. Missing coverage blocks startup with `proxy_direct_bypass_missing`.

Accepted bypass forms include exact hosts, host:port values, bracketed IPv6 host:port values, `.local`/`*.local`, and CIDR ranges. A broader CIDR may cover a narrower required range, so `10.0.0.0/8` covers the StackChan lab range `10.21.0.0/16`.

The Makefile exports `A21_DIRECT_NO_PROXY` to `NO_PROXY` and `no_proxy` for
standard A21 make targets so `make verify`, `make preflight`, and `make doctor`
use the required direct-connect set even when the parent shell has an incomplete
proxy bypass list. Direct `go run ./cmd/a21 ...` invocations still intentionally
inherit the caller's environment and may block if the bypass coverage is
incomplete.

The report records env variable names only. It does not print proxy URLs, credentials, or provider proxy values. `A21_PROVIDER_PROXY_URL` is treated as an explicit provider egress setting, not as permission for LAN, localhost, `.local`, StackChan, or local V21 adapter traffic to inherit global proxy behavior.

## Provider Egress Policy

Future HTTP-based provider adapters must build clients through `internal/providers.NewProviderHTTPClient`. The default mode is `direct`, which disables ambient environment proxy inheritance. If `A21_PROVIDER_PROXY_URL` is set, provider HTTP clients switch to `explicit_proxy` mode and use that URL for cloud provider egress only.

The current HTTP provider proxy support accepts `http` and `https` proxy URLs. SOCKS and provider-specific WebSocket dialers need a separate adapter implementation and tests before use. Doctor reports only the env variable name and network mode; it never prints provider proxy values.

The current Iflytek/Xfyun host TTS WebSocket adapter is direct-only: it uses an
A21-owned HTTP client with ambient `HTTP_PROXY` / `HTTPS_PROXY` disabled and
reports only `network_mode=direct`. `A21_PROVIDER_PROXY_URL` does not yet apply
to WebSocket TTS. If explicit provider-proxy WebSocket support is needed, add a
separate adapter transition with tests and redacted reporting.

Mac-side A21 verification commands that fetch Gateway HTTP/WS evidence also use
direct clients with ambient proxy disabled. If a local TUN/VPN route still
captures public A21 or Aliyun traffic after proxy env has been bypassed, set
`A21_DIRECT_SOURCE_IP=<local-en0-ip>` for the verification command. This binds
the outbound TCP source address, equivalent to `curl --interface`, and is a
control-machine diagnostic escape hatch only. Do not set it in firmware, ECS
systemd units, provider env, or product runtime configuration.

`agent-io-smoke --execute` is intentionally not a cloud provider egress client.
It uses an HTTP transport with `Proxy: nil`, records only a coarse endpoint
label, and refuses URL credentials or device-control paths.

## Public Gateway Profile

A21 exposes a product public Gateway profile for StackChan as the main
voice-edge entry when a public URL is configured. This is a transport profile,
not a new product mode and not a local model host replacement.

- `public_wss` is enabled and selected by default when
  `A21_PUBLIC_GATEWAY_URL` is a valid public endpoint without URL credentials,
  token strings, or secret strings. Product deployment targets `https://...`
  or `wss://.../v1/xiaozhi`; IP-only bring-up may temporarily use
  `http://...` or `ws://.../v1/xiaozhi`.
- `mac_local` remains selectable for Mac-local models and local processing.
  When the frontend or public control surface must show the actual Mac path,
  configure `A21_MAC_LOCAL_GATEWAY_URL` with a credential-free
  `ws://.../v1/xiaozhi` or `wss://.../v1/xiaozhi` URL. If it is unset, Gateway
  falls back to the local request-host URL.
- Public ingress should terminate TLS on external `443` with Caddy/Nginx and
  reverse proxy to the A21 Gateway process on the existing internal port.
- No provider key may be stored in firmware or URL configuration.
- `/xiaozhi/ota/` returns the selected profile's WebSocket URL. Behind TLS
  reverse proxy, `X-Forwarded-Proto: https` maps local request-host OTA output
  to `wss://.../v1/xiaozhi`.

## Mainland Lab Execution Boundary

Real mainland provider latency, p95/p99, cold-start, and error-rate evidence
must be collected on `5080lab` or another explicitly approved clean mainland
lab host, not on a Mac whose traffic is affected by ambient proxy/VPN routing.
This development Mac may run contract tests, fixture ingestion, local Ollama,
mock provider paths, and httptest-backed provider simulations only. Reports
from a proxy-affected host can prove redaction and protocol shape, but they are
not PRD launch evidence for mainland cloud-provider performance.

When a provider profile is promoted for product-path experimentation, the code
path must still be A21-owned and provider-neutral: `A21_PROVIDER_PROFILES_PATH`
may add an `a21_`-namespaced OpenAI-compatible, `route_eligible=true` text
profile for loopback or Gateway voice-pipeline tests, but real paid execution
remains a T4/provider window and should be run from the clean lab host.
The canonical 5080lab command sequence and return-file checklist live in
`docs/engineering/A21_PROVIDER_BENCHMARKS.md`. The Mac-side control tower may
ingest the returned `a21-provider-smoke-*.json` with `product-readiness`, but it
must not substitute a Mac `provider-smoke --execute` run for mainland p50/p95/p99
evidence.

## Required Diagnostics

Phase 1 already records:

- default route interface
- external DNS probe IP
- proxy env variable names without values
- proxy/no-proxy coverage for the direct-connect set
- explicit LAN TCP probe receipts through `a21 lan-probe`

## Port Registry

Current A21-owned ports are `21080` for Gateway HTTP/WebSocket, reserved
`21081` realtime, `21073` console, `21086` observability, `21095` ASR sidecar,
`21114` LLM adapter, optional host-only `21130` Agent I/O bridge, `21121` V21
adapter boundary, and reserved `21434` local-model bridge. A21 must not adopt
new ports without updating this section and the runtime guards.

Known legacy ports are treated as contaminated for A21 endpoint env vars and
LAN probes: `8000`, `8080`, `10095`, `18080`, `4173`, `42173`, `16686`, and
`16687`. A21 may reach V21 only through the A21 adapter boundary, not V21
internals.

## LAN Probe

`lan-probe` is an explicit office/home reachability command. It does not run as a default doctor side effect because StackChan and office-only endpoints may legitimately be offline during home development.

```bash
go run ./cmd/a21 lan-probe --target a21-gateway=127.0.0.1:21080 --output-dir reports
A21_LAN_TARGET=a21-gateway=127.0.0.1:21080 A21_LAN_SAMPLES=5 make lan-probe
go run ./cmd/a21 lan-probe --target a21-v21-adapter=127.0.0.1:21121 --samples 20 --timeout-ms 1000 --output-dir reports
```

The command performs direct TCP dials and writes:

```text
reports/a21-lan-probe-YYYYMMDD-HHMMSS.json
```

The report includes generated timestamp, current commit, network/DNS fingerprint, redacted proxy-policy metadata, target name, normalized `host:port`, direct flag, status, sample count, passed/failed samples, p50, p95, jitter, and a coarse error code. It must not include proxy URLs, proxy hosts/ports, proxy credentials, API keys, URL credentials, or X21 identities.

`lan-probe` also refuses known legacy internal ports used by X21/V21/VKP services. Use the A21 V21 adapter boundary (`21121`) for professional-mode reachability evidence; do not probe V21 internals such as `8080` or `18080` from A21 tooling.

Use it to compare:

- home development network
- Shanghai office authenticated Wi-Fi
- `wang301` Wi-Fi
- wired LAN Gateway host
- A21 V21 adapter boundary

Future `a21 doctor` expansion must add:

- raw LAN ping and jitter
- StackChan reachability or mDNS beyond explicit TCP probes
- V21 adapter health
- provider endpoint connectivity by adapter
- warning when DNS maps external domains to `198.18.0.x`

## Shanghai Office Audit

Before real office acceptance, record:

- office Wi-Fi login flow
- whether StackChan can join authenticated Wi-Fi
- `wang301` Wi-Fi latency and client-isolation behavior
- wired LAN path between high-performance PCs, MacBook Pros, and StackChan
- proxy on/off and split-tunnel behavior
- provider latency from office network
- LAN jitter between gateway host and device
- `lan-probe` reports for Gateway, adapter, and any StackChan-adjacent listening endpoint

## Failure Copy

Network failures must be honest and calm:

- "I can hear you, but I cannot reach the outside model right now."
- "The office network looks like it is sending me around a proxy. I am checking the local path first."
- "V21 is not reachable right now. I can keep the question and retry."
