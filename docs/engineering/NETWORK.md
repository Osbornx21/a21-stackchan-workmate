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

Firmware Wi-Fi credentials are not provider credentials, but they are still local secrets. They must not be committed into `platformio.ini`, firmware docs, logs, doctor reports, or artifact names. Default firmware builds may contain no SSID and must enter local fallback. Local hardware bring-up may use ignored `firmware/stackchan/include/a21_firmware_secrets.local.h`; longer-term provisioning must preserve StackChan calibration/NVS keys and stay A21-namespaced.

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

## Current Proxy Guard

`preflight` and `doctor` now evaluate a proxy policy report. When any global proxy env var is configured (`HTTP_PROXY`, `HTTPS_PROXY`, or `ALL_PROXY`), A21 requires `NO_PROXY` or `A21_NO_PROXY` to cover the full direct-connect set above. Missing coverage blocks startup with `proxy_direct_bypass_missing`.

Accepted bypass forms include exact hosts, host:port values, bracketed IPv6 host:port values, `.local`/`*.local`, and CIDR ranges. A broader CIDR may cover a narrower required range, so `10.0.0.0/8` covers the StackChan lab range `10.21.0.0/16`.

The report records env variable names only. It does not print proxy URLs, credentials, or provider proxy values. `A21_PROVIDER_PROXY_URL` is treated as an explicit provider egress setting, not as permission for LAN, localhost, `.local`, StackChan, or local V21 adapter traffic to inherit global proxy behavior.

## Provider Egress Policy

Future HTTP-based provider adapters must build clients through `internal/providers.NewProviderHTTPClient`. The default mode is `direct`, which disables ambient environment proxy inheritance. If `A21_PROVIDER_PROXY_URL` is set, provider HTTP clients switch to `explicit_proxy` mode and use that URL for cloud provider egress only.

The current HTTP provider proxy support accepts `http` and `https` proxy URLs. SOCKS and provider-specific WebSocket dialers need a separate adapter implementation and tests before use. Doctor reports only the env variable name and network mode; it never prints provider proxy values.

## Required Diagnostics

Phase 1 already records:

- default route interface
- external DNS probe IP
- proxy env variable names without values
- proxy/no-proxy coverage for the direct-connect set

Future `a21 doctor` expansion must add:

- raw LAN ping and jitter
- StackChan reachability or mDNS
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

## Failure Copy

Network failures must be honest and calm:

- "I can hear you, but I cannot reach the outside model right now."
- "The office network looks like it is sending me around a proxy. I am checking the local path first."
- "V21 is not reachable right now. I can keep the question and retry."
