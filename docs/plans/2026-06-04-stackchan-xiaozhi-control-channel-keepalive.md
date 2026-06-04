# StackChan Xiaozhi Control Channel Keepalive Plan

Transition id: `T-STACKCHAN-XIAOZHI-CONTROL-KEEPALIVE-001`

Goal: make the A21 official-compatible Xiaozhi product control channel recover
quickly after a Gateway safe-swap/restart, without regressing internal-test3
voice protocol, MCP body/status controls, or product flash guards.

Current state:

- Product firmware keeps a quiet `/v1/xiaozhi` websocket open while idle.
- `EnsureA21ControlChannel()` only reconnects when
  `protocol_->IsAudioChannelOpened()` is false.
- Official `WebsocketProtocol` has no idle heartbeat and uses a 120 second
  incoming timeout, so a dead idle socket can block control-channel recovery
  after Gateway restart.
- Gateway already accepts `type=device, kind=heartbeat`, but product firmware
  does not send a product-safe heartbeat.

Target state:

- Product firmware advertises `features.keepalive_events=true` together with
  the existing product playback allowance.
- Gateway returns `a21.profile=product` with `keepalive_events=true` only for a
  hardware-MAC Xiaozhi client when the existing product playback-events runtime
  gate is enabled.
- Product firmware sends a low-risk `type=device, kind=heartbeat` on the idle
  control channel; send failure closes the stale channel so the existing
  reconnect loop can open a fresh one.
- Product allowance still rejects debug-only state/face/motion/display events.

Acceptance:

- Gateway tests prove product heartbeat is accepted and traced, while stock
  clients and non-heartbeat debug events remain blocked.
- Overlay contract tests prove the official-compatible product firmware carries
  the keepalive feature flag, server allowance parser, heartbeat send method,
  and stale-channel close path.
- `git diff --check`, focused tests, and `GOMAXPROCS=2 make verify` pass.
- Physical acceptance remains a foreground hardware window: deploy Gateway,
  build/flash only through `a21-stackchan-official-xiaozhi-compatible.bin`, and
  verify device reconnect after Gateway restart.

Forbidden actions:

- Do not use generic `xiaozhi.bin` or any generic xiaozhi flash lane for the
  product StackChan device.
- Do not store provider keys in firmware.
- Do not weaken MCP whitelist, numeric JSON-RPC ids, or inbound MCP response
  redaction.
- Do not prune/gc Git loose objects.
