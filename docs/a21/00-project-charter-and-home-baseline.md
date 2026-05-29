# A21 Project Charter and Home Environment Baseline

Date: 2026-05-29
Location context: home environment now; Shanghai office environment will be audited separately.
Workspace: `/Users/jiyurun/Documents/New project`

## Mission

A21 is a new StackChan-centered voice and companion system. It must not inherit X21 or V21 process names, ports, runtime assumptions, data volumes, environment variables, or development habits.

The end state is a low-latency, full-duplex, emotionally expressive StackChan experience: natural listening, fast response, voice cloning, roleplay, deep knowledge access, rich screen expressions, motion/RGB/touch semantics, and a firmware/OEM layer that feels polished rather than patched together.

## Non-Negotiable Principles

- User experience is the final judge. Latency, interruption behavior, speech naturalness, screen emotion, touch, motion, and reliability must be evaluated together.
- Engineering discipline comes before speed theater. Every subsystem needs clear ownership, measurable behavior, logs/traces/metrics, and a reproducible repair path.
- Do not build from superstition. When latency, proxy, audio, network, or hardware behavior is surprising, investigate root cause before changing code.
- Do not repeat X21/V21 identity leakage. A21 uses `A21_*` env vars, `a21-*` process names, `a21` compose project names, `a21_*` volumes, and an A21-only port range.
- Do not reuse legacy defaults. X21 `8000/10095` and V21 `8080/18080/4173/42173/16686/16687` are treated as occupied legacy territory.
- Use good wheels. Prefer mature, observable, replaceable components for ASR, TTS, vector retrieval, reranking, audio transport, firmware tooling, and tracing.
- Keep boundaries small and explicit. Firmware, device bridge, realtime voice engine, knowledge service, role/persona service, console, evals, and ops must be separable and testable.
- Every important claim needs evidence. "Fast", "works", "low latency", "full duplex", and "device ready" require recorded measurements or reproducible tests.
- Paid APIs are resources, not architecture. DeepSeek API, Alibaba Cloud domain resources, and Bailian/DashScope APIs may be used when they are the right tool, but A21 must keep provider adapters replaceable and must never hard-code the business logic to one vendor.

## Initial A21 Namespace

Reserved A21 local port block, verified free on 2026-05-29:

- `21080`: A21 backend/API and device OTA entry
- `21081`: A21 realtime voice/WebSocket entry if split from API
- `21073`: A21 console/dev UI
- `21086`: A21 tracing/observability UI if exposed locally
- `21095`: A21 local ASR sidecar if needed
- `21114`: A21 local model/LLM adapter if needed
- `21434`: A21 local Ollama-compatible adapter if needed

These are provisional but must remain A21-owned. A startup preflight should fail if any A21 port is already occupied by non-A21 processes.

## Current Machine Baseline

- Mac: MacBook Pro, Apple M4 Pro, 14 cores, 48 GB memory.
- OS: macOS 26.5, Darwin 25.5.0, arm64.
- Disk at workspace: 151 GiB available.
- Workspace repo: empty Git repository on `main`, no remote, no commits.
- Xcode developer path: `/Applications/Xcode.app/Contents/Developer`.
- ESP-IDF: `/Users/jiyurun/esp/esp-idf-v5.5.2`, `idf.py --version` reports ESP-IDF v5.5.2.
- ESP serial device detected: `/dev/cu.usbmodem1101`, USB vendor Espressif, USB JTAG/serial debug unit.

## Runtime Baseline

- Default Node from PATH: Homebrew Node v25.8.0 at `/opt/homebrew/bin/node`.
- nvm Node present: v22.22.0 at `/Users/jiyurun/.nvm/versions/node/v22.22.0/bin/node`.
- Codex bundled Node present: v24.14.0.
- Default Python from PATH: Homebrew Python 3.14.3.
- Codex bundled Python: 3.12.13.
- ESP-IDF Python env: Python 3.14.3 under `/Users/jiyurun/.espressif/python_env/idf5.5_py3.14_env`.

A21 must not rely on ambient PATH. Project commands should pin exact tool paths or use a project-managed runtime entrypoint.

## Network Baseline

- Current active LAN IP: `192.168.1.20` on Wi-Fi.
- Wi-Fi radio: 802.11ax, 2.4 GHz channel 6, 20 MHz, strong signal.
- Gateway: `192.168.1.1`; gateway ping avg about 4.8 ms.
- LAN peers observed: `192.168.1.2` and `192.168.1.26`; peer ping showed visible jitter.
- Active proxy/VPN application: `/Applications/龙猫云_Lite.app`, with root `lmclientCore`.
- Legacy Clash Verge helper remains enabled at system level.
- DNS responses for external domains are mapped to `198.18.0.x`, and default external traffic is routed through `utun8`.
- `networkquality` ran through `utun8`, not raw Wi-Fi: downlink about 4.6 Mbps, uplink about 105 Mbps, idle latency about 848 ms, responsiveness low.

This network state is acceptable for general development, but it is not valid as a low-latency voice benchmark. A21 needs separate measurements for proxy mode, direct LAN mode, and Shanghai office mode.

## Legacy Process and Port Baseline

Legacy processes currently active:

- X21 backend: `0.0.0.0:8000`, cwd `/Users/jiyurun/Documents/小马暴力`.
- X21 cloudflared tunnel: exposes `127.0.0.1:8000`.
- X21 local FunASR: `127.0.0.1:10095`.
- X21 desktop device control script: `/Users/jiyurun/Desktop/x21-device-control.command`.
- V21AIR Docker Compose stack: active from `/Users/jiyurun/Documents/v21-knowledge-platform/deploy/compose`.
- Old voice knowledge platform Compose stack: active from `xiaozhi-voice-knowledge-platform`.
- Ollama server: `127.0.0.1:11434`.

A21 must treat all of these as legacy neighbors and never silently connect to them.

## Immediate Environment Fix Applied

The global Git config contained stale proxies:

- `http.proxy = http://127.0.0.1:7897`
- `https.proxy = http://127.0.0.1:7897`

Port `7897` was not listening, and `git ls-remote https://github.com/git/git HEAD` failed immediately through that proxy. The config was backed up to `~/.gitconfig.a21-env-audit-20260529-2212.bak`, then the stale global Git proxy entries were removed. GitHub access was verified afterward.

## A21 Preflight Gates

Phase 1 local startup preflight must:

- Reject startup if process cwd or env vars contain X21/V21 identifiers.
- Reject startup if A21 endpoint env vars point to known X21/V21 ports.
- Reject startup if any A21 reserved port is occupied by a non-A21 process.
- Reject startup if the runtime cannot record a minimum network/DNS fingerprint.
- Record the default network interface and the external DNS probe IP, including whether it resolves into the `198.18.0.x` mapped range.
- Reject `.env` files or shell environments that define legacy `X21_*`, `V21_*`, `ROLEPLAY_*`, or `VOICE_KNOWLEDGE_*` settings in an A21 runtime.

Container deployment preflight, before any A21 Docker/Compose runtime is introduced, must:

- Reject startup if compose project or volume names contain X21/V21 identifiers.
- Reject startup if the compose project is not named `a21` or volumes are not prefixed `a21_`.

Real-device acceptance preflight must:

- Record active network mode: raw LAN, proxy/TUN, office authenticated Wi-Fi, coworker Wi-Fi, wired LAN, or hotspot.
- Record audio input/output devices and sample rates before voice tests.
- Record serial devices and expected StackChan hardware identity before flashing or provisioning.
- Separate local development tests from real device acceptance.
- Reject provider calls that would silently inherit ambient HTTP/SOCKS proxy settings unless the provider adapter explicitly allows that network mode.

## Available Paid Provider Resources

The user can provide paid API access when needed:

- DeepSeek API.
- Alibaba Cloud domain resources.
- Alibaba Bailian/DashScope API.

These resources should enter A21 through provider adapters with explicit health checks, latency probes, fallback behavior, and cost/usage accounting. They must not bias the core architecture decision.

## Next Shanghai Office Audit Checklist

- Identify office Wi-Fi authentication path and whether StackChan can join it directly.
- Test `wang301` Wi-Fi latency, jitter, multicast/mDNS behavior, and client isolation.
- Test wired LAN path between high-performance PCs, MacBook Pros, and StackChan.
- Measure raw LAN ping/jitter between all development hosts and device.
- Decide whether A21 should run central services on a MacBook, high-performance PC, or dedicated mini host.
- Confirm whether Dragon Cat Lite/proxy must be disabled, bypassed, or split-tunneled for LAN and provider calls.
- Measure first-audio latency separately for ASR partial, LLM first token, TTS first audio, device playback, and full-duplex interruption.

## Phase 1 Implementation Pointer

Phase 1 starts the clean A21 skeleton from `docs/superpowers/plans/2026-05-29-a21-phase1-clean-skeleton.md`.

The first verification target is:

```bash
make verify
```
