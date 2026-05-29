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

## Required Diagnostics

Phase 1 already records:

- default route interface
- external DNS probe IP
- proxy env variable names without values

Future `a21 doctor` expansion must add:

- raw LAN ping and jitter
- StackChan reachability or mDNS
- V21 adapter health
- provider endpoint connectivity by adapter
- `NO_PROXY` coverage for direct-connect set
- detection of global `HTTP_PROXY`, `HTTPS_PROXY`, and `ALL_PROXY`
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
