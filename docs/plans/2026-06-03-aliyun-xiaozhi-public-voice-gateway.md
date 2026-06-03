# 2026-06-03 - Aliyun Xiaozhi Public Voice Gateway

Status: active.
Owner: A21 control tower.
Transition: `T-ALIYUN-001-XIAOZHI-PUBLIC-VOICE-GATEWAY`.

## Decision

Use the Aliyun host as an A21 Xiaozhi-compatible public voice relay / cloud
edge Gateway, not as a local ASR/TTS model server.

The Aliyun public Gateway is now the main A21 voice-edge Gateway. It is a
product `gateway_profile`, not an experiment and not a replacement for the Mac
local Gateway's local-model speed lane. When `A21_PUBLIC_GATEWAY_URL` is
configured, `public_wss` is the default product profile. The production target
is public `443` / trusted `wss` StackChan access; IP-only ECS bring-up may
temporarily use `http/ws` until a domain and trusted certificate are attached.
`mac_local` remains a selectable frontend/operator profile for Mac-local
models and local processing.

Target path:

`StackChan -> wss://A21 Aliyun Gateway:443/v1/xiaozhi -> cloud ASR -> LLM stream -> cloud TTS -> Opus downlink -> StackChan`

The server owns long WebSocket sessions, stock Xiaozhi JSON/binary frame
timing, provider orchestration, barge-in, trace, health, and redacted evidence.
Provider keys stay on the server. Firmware receives only the Gateway URL and
never stores provider credentials.

## Current Evidence

- Current repo: `codex/network`, HEAD `2f8a63f`, dirty/staged.
- Provider compatibility matrix is `ready`:
  `reports/a21-provider-compat-matrix-20260603-120639-539522000.json`.
- Cloud ASR candidate: Iflytek p95 final `134.142 ms`.
- Cloud TTS candidate: Iflytek first-audio `71.688 ms`.
- Route-eligible LLM evidence exists through DeepSeek first-content p95
  `745.516 ms`.
- Product readiness remains blocked by physical StackChan online/acceptance and
  wake-word product readiness, not by provider API availability:
  `reports/a21-product-readiness-20260603-120713.json`.
- Mac direct Doubao realtime TTS execution reached provider configuration but
  failed before `tts_session.update`; direct and explicit-proxy TLS/WebSocket
  diagnostics to the Volcengine realtime gateway hit reset/EOF before
  application events.
- Local 21081 Gateway has real device `44:1b:f6:e2:6a:60`, but it is stale.
  Mac en0 currently has `10.98.141.239`, not the historical `192.168.1.20`
  path used by the device.
- The earlier Aliyun SWAS/lightweight server `101.132.117.182` is retained as
  an experimental/backup chain only.
- User supplied the new main Aliyun ECS configuration:
  instance `i-uf63f4ymqc2dxtljxz2n`, public IP `47.103.57.217`, region
  `cn-shanghai`, purpose A21 voice edge / Xiaozhi-compatible Gateway public
  entry.

## Deployment Evidence - 2026-06-03 13:31 CST, Experimental SWAS

- The earlier Aliyun target was a SWAS/lightweight application server in
  `cn-shanghai`, not an ECS instance. CLI access works through
  `aliyun swas-open run-command` with explicit
  `--region cn-shanghai --endpoint swas.cn-shanghai.aliyuncs.com`.
- The current tracked A21 workspace snapshot was transferred through SWAS
  command chunks to `/opt/a21`; remote base64 length and tar SHA-256 matched
  the local artifact before extraction.
- Go `1.26.3` and Linux build prerequisites were installed on the server.
  The default `proxy.golang.org` path timed out from Aliyun; rebuild passed
  with `GOPROXY=https://goproxy.cn,direct` and
  `GOSUMDB=sum.golang.google.cn`.
- `/opt/a21/bin/a21` was built successfully and is running as systemd service
  `a21-gateway` on `127.0.0.1:21080` with
  `A21_PUBLIC_GATEWAY_URL=https://101.132.117.182`,
  `A21_XIAOZHI_PRODUCT_CHAIN=host_local`, and no provider credential values.
- Existing nginx on port `80` served the prior control console, so Caddy was
  not allowed to take over the host. The nginx site was backed up and updated
  so only A21 paths (`/healthz`, `/simulator`, `/v1/`, `/xiaozhi/ota`,
  `/ws/audio`) proxy to `127.0.0.1:21080`; the existing root console remains
  proxied to `127.0.0.1:8000`.
- nginx now listens on `80` and `443`. Current `443` uses a 30-day A21
  self-signed certificate for IP-only bring-up. This proves the network/TLS
  route with `curl -k`, but it is not product-trusted TLS for StackChan until a
  real domain/certificate or device trust decision is made.
- External Mac verification passed for:
  - `http://101.132.117.182/healthz`
  - `https://101.132.117.182/healthz` with `-k`
  - `https://101.132.117.182/v1/gateway-profiles` with `-k`
  - `https://101.132.117.182/xiaozhi/ota/` with `-k`, returning
    `wss://101.132.117.182/v1/xiaozhi`
- Strict HTTPS verification without `-k` currently fails with
  `SSL certificate problem: self signed certificate`, as expected.
- Host-only public WebSocket bench over `http://101.132.117.182` accepted
  hello/listen and produced trace metrics, but remains blocked:
  answer turn passed, barge-in turn failed with `turn_read_failed`, and
  `product_chain_not_executed` because no real cloud provider secrets were
  injected. Report:
  `reports/a21-xiaozhi-voice-bench-20260603-133114.031578000.json`.

This SWAS chain is no longer the main product Gateway after the user created
the higher-spec ECS. It remains useful only as fallback evidence and should not
be described as the current primary A21 public voice edge.

## Main ECS Target - 2026-06-03

- ECS instance: `i-uf63f4ymqc2dxtljxz2n`
- Public IP: `47.103.57.217`
- Region: `cn-shanghai`
- Role: main A21 voice edge / Xiaozhi-compatible Gateway public entry.
- Mac local Gateway remains selectable as `mac_local` for local model and
  local processing workflows.
- Provider credentials must be injected only through a server-side secret
  path; do not place provider keys in repo files, command history, systemd
  unit files, Caddy/Nginx config, firmware, reports, or logs.

## Deployment Evidence - 2026-06-03 14:22 CST, Main ECS

- Current tracked A21 snapshot was transferred to `47.103.57.217` as
  `/tmp/a21-tracked-current-main-ecs.tar.gz`; SHA-256 matched
  `1782eaec865fc771452956942f0563466c56b24c3a51bda1bb315028c34fb98a`
  before extraction.
- Remote `/opt/a21` was replaced through `/opt/a21.next` after successful
  build/verification; previous `/opt/a21` was preserved under a timestamped
  backup directory.
- Remote Go is `go1.26.3 linux/amd64`; remote `make verify` passed.
- `a21-gateway.service` is enabled and active, running
  `/opt/a21/bin/a21 gateway --addr 127.0.0.1:21081 --public-gateway-url
  http://47.103.57.217 --product-chain host_local --voice-text-max-tokens 32`.
- Caddy is enabled and active on public `80` and `443`, reverse proxying A21
  paths to `127.0.0.1:21081`. `443` uses a temporary 30-day IP SAN
  self-signed certificate for bring-up only; strict trusted TLS still requires
  a real domain/certificate.
- External Mac verification passed:
  - `http://47.103.57.217/healthz`
  - `http://47.103.57.217/v1/gateway-profiles`, with
    `selected_gateway_profile=public_wss`
  - `http://47.103.57.217/xiaozhi/ota/`, returning
    `ws://47.103.57.217/v1/xiaozhi`
  - `https://47.103.57.217/healthz` with `-k`
  - `https://47.103.57.217/xiaozhi/ota/` with `-k`, also returning
    `ws://47.103.57.217/v1/xiaozhi`
- Host-only public WebSocket bench over `http://47.103.57.217` accepted
  hello/listen and observed the host-local voice-pipeline shape, but remains
  blocked: provider/V21/hardware execution flags are false, LLM profile is
  `mock`, barge-in turn failed with `turn_read_failed`, and
  `prd_accepted=false`. Report:
  `reports/a21-xiaozhi-voice-bench-20260603-142101.627764000.json`.

## Target State

- A21 Gateway exposes `GET/POST /v1/gateway-profiles`.
- The frontend can select `mac_local` or configured `public_wss` independently
  from `dialogue` / `professional`.
- `public_wss` is the default profile when `A21_PUBLIC_GATEWAY_URL` is
  configured and valid.
- `mac_local` keeps Mac Gateway local model / local processing speed as a
  selectable local profile.
- Aliyun exposes `443` with TLS and routes `/v1/xiaozhi` to the A21 Go Gateway.
- `A21_PUBLIC_GATEWAY_URL` configures the public profile. It accepts
  `https://...` / `wss://.../v1/xiaozhi` for product TLS and `http://...` /
  `ws://.../v1/xiaozhi` for IP-only bring-up, always without credentials.
- Gateway runs in A21 namespace with provider env injected only on the server.
- StackChan connects over public `wss` and sends a fresh stock `hello`.
- One physical candidate trace records:
  - device online hello;
  - real mic/Opus ingress;
  - ASR partial/final;
  - LLM first token/content;
  - TTS first audio;
  - Opus downlink;
  - audible playback;
  - barge-in or abort;
  - idle recovery.
- Result remains physical candidate until wake-word product proof is complete.

## Immediate Run Order

1. Do not switch Mac networks during this run. Local Wi-Fi recovery is slow and
   out of scope for the current low-latency push.
2. Bring up the new ECS `47.103.57.217` as the main `public_wss` Gateway:
   open public ingress, proxy to A21 Gateway, and point StackChan at
   `ws://47.103.57.217/v1/xiaozhi` for immediate IP-only bring-up. Attach a
   domain/trusted certificate before promoting the address to final
   `wss://<A21 public host>/v1/xiaozhi`. Preserve Mac local Gateway as
   selectable `mac_local`.
3. Run provider chain through cloud ASR + LLM stream + cloud TTS. Do not run
   local ASR/TTS models on the 2C2G host.
4. Collect a redacted physical candidate report and rerun product readiness.

## Non-Goals

- Do not rewrite A21 away from Go-first Gateway.
- Do not copy xinnan server architecture wholesale.
- Do not put provider keys in firmware.
- Do not claim full PRD green while wake-word product proof is missing.
- Do not treat `/v1/xiaozhi/say` foreground WAV playback as full realtime
  dialogue acceptance.

## Acceptance

- `healthz` and `/v1/xiaozhi` are reachable through public `443`.
- `GET /v1/gateway-profiles` shows `public_wss` as the default when
  `A21_PUBLIC_GATEWAY_URL` is configured, and keeps `mac_local` available for
  frontend/operator switching.
- Selecting `public_wss` makes `/xiaozhi/ota/` return the configured public
  `ws`/`wss` `/v1/xiaozhi` URL; selecting `mac_local` keeps request-host local
  `ws`/`wss` generation.
- StackChan is online and fresh in Gateway registry.
- Redacted trace/report contains only basenames, timings, counts, trace/session
  identifiers, provider names, endpoint host labels, and redaction booleans.
- No provider key, Authorization header, full URL, prompt/transcript/provider
  output, raw/base64 audio, proxy value, or local absolute path is stored.

## Rollback

- Keep existing local Gateway and 5080 evidence intact.
- Revert only Aliyun deployment configuration or DNS/security-group changes.
- Device can be pointed back to local LAN Gateway after Wi-Fi/LAN stabilizes.
