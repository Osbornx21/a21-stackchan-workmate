# A21 Phase 4C Provider Smoke

## Purpose

Phase 4C adds the first real-provider smoke boundary without making provider calls part of ordinary startup, doctor, firmware, simulator, or Gateway traffic.

The goal is disciplined readiness:

- know whether a provider is configured
- know which protocol the smoke path would use
- prove provider env values are not leaked
- keep all provider egress behind the A21 provider network policy
- require explicit operator intent before any paid or external provider request

## Current Scope

`internal/providers.ProviderSmokeFromEnv` builds a redacted smoke report for the selected provider. `doctor` includes this report as a dry-run plan only.

The CLI command is:

```bash
go run ./cmd/a21 provider-smoke --provider deepseek
go run ./cmd/a21 provider-smoke --provider deepseek --execute
```

`--execute` is required for a real network call.

## Executable Providers

Current executable smoke providers:

- `deepseek`
- `bailian_dashscope`

Both use OpenAI-compatible Chat Completions with a tiny non-streaming request.

The provider network client defaults to `direct` mode and ignores ambient `HTTP_PROXY`, `HTTPS_PROXY`, and `ALL_PROXY`. If Shanghai network conditions require explicit provider egress, set `A21_PROVIDER_PROXY_URL`; reports show only the env variable name and network mode, never the proxy URL value.

## Non-Executable Providers

Current non-executable smoke providers:

- `openai_realtime`
- `doubao_realtime`

They are realtime WebSocket providers, not Chat Completions providers. Phase 4C reports them as `unsupported` for smoke execution instead of pretending an HTTP chat request proves realtime audio readiness.

Phase 4D adds a lower-level realtime WebSocket adapter skeleton in `internal/providers/realtime.go`. That skeleton can build a redacted OpenAI realtime connection plan and send generic `session.update` / `response.cancel` events through an injected WebSocket connection, but it still does not execute realtime provider smoke from `provider-smoke`.

## Env

DeepSeek smoke:

- `A21_DEEPSEEK_API_KEY`
- `A21_DEEPSEEK_MODEL`
- optional `A21_DEEPSEEK_BASE_URL`, default `https://api.deepseek.com`

Bailian/DashScope smoke:

- `A21_DASHSCOPE_API_KEY`
- `A21_DASHSCOPE_MODEL`
- optional `A21_DASHSCOPE_BASE_URL`, default `https://dashscope.aliyuncs.com/compatible-mode/v1`

Realtime providers:

- `A21_OPENAI_API_KEY`
- `A21_OPENAI_REALTIME_MODEL`
- `A21_DOUBAO_API_KEY`
- `A21_DOUBAO_REALTIME_MODEL`

## Source Baseline

The OpenAI-compatible smoke path follows the public provider contracts rather than custom request shapes:

- DeepSeek official API docs expose `/chat/completions` and current model IDs under the OpenAI-compatible API: https://api-docs.deepseek.com/api/create-chat-completion
- Alibaba Cloud Model Studio/Bailian official docs list the OpenAI-compatible Beijing base URL `https://dashscope.aliyuncs.com/compatible-mode/v1` and HTTP endpoint `POST /chat/completions`: https://help.aliyun.com/zh/model-studio/compatibility-of-openai-with-dashscope
- OpenAI official Realtime docs describe server-to-server WebSocket sessions for Realtime, so A21 does not treat OpenAI Realtime as an HTTP Chat Completions smoke target: https://developers.openai.com/api/docs/guides/realtime-websocket
- OpenAI official Realtime conversations docs describe `session.update`, streamed audio buffer events, and server audio delta events: https://developers.openai.com/api/docs/guides/realtime-conversations#handling-audio-with-websockets
- Volcengine official realtime AI voice docs show that Doubao/Volcengine realtime voice belongs to an RTC/OpenAPI-oriented stack with interruption, latency, embedded hardware, knowledge-base, and MCP concerns; A21 will keep it behind a provider-specific adapter: https://www.volcengine.com/docs/6348/1902994

## Safety Rules

- `doctor` never executes provider smoke.
- `provider-smoke` without `--execute` never performs network I/O.
- Reports expose env variable names, protocol, status, network mode, and endpoint host only.
- Reports never expose API key values, model values, proxy values, or full URLs.
- Unknown provider names are redacted to `unknown_provider`.
- legacy-looking provider names are redacted to `invalid_legacy_provider` and fail.
- StackChan firmware never sees any provider key or provider smoke configuration.

## Current Boundaries

Phase 4C does not implement:

- realtime WebSocket smoke
- Doubao realtime adapter
- production OpenAI realtime adapter
- provider latency histograms
- provider cost accounting
- provider-specific retry or backoff

Those belong in later provider-adapter phases after this smoke boundary and the Phase 4D realtime transport skeleton are stable.
