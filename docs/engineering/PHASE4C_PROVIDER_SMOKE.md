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
go run ./cmd/a21 provider-smoke --provider deepseek --stream --repeat 3
go run ./cmd/a21 provider-smoke --provider deepseek --execute --stream --repeat 3 --output-dir reports
go run ./cmd/a21 provider-smoke --provider deepseek --output-dir reports
```

`--execute` is required for a real network call. `--stream` switches OpenAI-compatible providers to SSE streaming mode, and `--repeat N` repeats the same redacted smoke up to 10 times so A21 can collect first-byte, first-content, and total-duration timing evidence without logging prompt, model value, API key, proxy URL, generated text, or reasoning text.

`--output-dir reports` writes a timestamped redacted evidence report:

```text
reports/a21-provider-smoke-YYYYMMDD-HHMMSS-nnnnnnnnn.json
```

The saved report is useful for paid OpenAI-compatible text-stream smoke runs because it preserves provider, provider family, protocol, status, execution flag, HTTP status, duration, streaming repeat count, first-byte/first-content summary timings, network mode, endpoint host, fallback markers, and trace/metric names without recording API key values, model values, proxy URLs, full request URLs, prompt text, output text, or reasoning text.

## Executable Providers

Current executable P0 text-stream smoke providers are derived from `ProviderProfile` records instead of a separate smoke-only table:

- `deepseek`

It uses OpenAI-compatible Chat Completions with a tiny request. DeepSeek is the current P0 route-eligible `text_stream` profile for A21's fast companion provider spine. Other cloud/local candidates stay out of the executable P0 registry until M3 evidence is clean and an ADR or follow-up slice promotes them. Streaming smoke uses OpenAI-compatible data-only SSE chunks and records redacted timing evidence.

`A21_PROVIDER_PROFILES_PATH` is intentionally not active in the P0 smoke path. Local profile override remains a later extension point after the single-provider route has real latency and failure evidence.

The provider network client defaults to `direct` mode and ignores ambient `HTTP_PROXY`, `HTTPS_PROXY`, and `ALL_PROXY`. If Shanghai network conditions require explicit provider egress, set `A21_PROVIDER_PROXY_URL`; reports show only the env variable name and network mode, never the proxy URL value.

## Non-Executable Providers

Realtime providers and agent bridges are not part of the P0 `provider-smoke` registry. Existing OpenAI/Doubao realtime code remains as a guarded plan/fixture boundary, but Phase 4C does not expose it as a selectable smoke target and does not pretend an HTTP chat request proves realtime audio or agent readiness.

Later realtime planning phases add lower-level realtime WebSocket readiness reports in `internal/providers/realtime.go`. That path can build redacted OpenAI realtime, Doubao realtime speech-to-speech, and Doubao realtime TTS connection plans and send generic test events through injected fake WebSocket connections, but it still does not execute realtime provider smoke from `provider-smoke`.

## Env

DeepSeek smoke:

- `A21_LAB_DEEPSEEK_API_KEY`
- optional `A21_DEEPSEEK_MODEL`, default `deepseek-v4-flash`
- optional `A21_DEEPSEEK_BASE_URL`, default `https://api.deepseek.com`

No other provider env is required for Phase 4C provider smoke.

## Source Baseline

The P0 OpenAI-compatible smoke path follows the public provider contract rather than a custom request shape:

- DeepSeek official API docs expose `/chat/completions` and current model IDs under the OpenAI-compatible API: https://api-docs.deepseek.com/api/create-chat-completion
- OpenAI official Realtime docs describe server-to-server WebSocket sessions for Realtime, so A21 does not treat OpenAI Realtime as an HTTP Chat Completions smoke target: https://developers.openai.com/api/docs/guides/realtime-websocket
- OpenAI official Realtime conversations docs describe `session.update`, streamed audio buffer events, and server audio delta events: https://developers.openai.com/api/docs/guides/realtime-conversations#handling-audio-with-websockets
- Volcengine official realtime AI voice docs show that Doubao/Volcengine realtime voice belongs to an RTC/OpenAPI-oriented stack with interruption, latency, embedded hardware, knowledge-base, and MCP concerns; A21 will keep it behind a provider-specific adapter: https://www.volcengine.com/docs/6348/1902994

## Safety Rules

- `doctor` never executes provider smoke.
- `provider-smoke` without `--execute` never performs network I/O.
- Reports expose env variable names, protocol, status, network mode, and endpoint host only.
- Reports never expose API key values, model values, proxy values, full URLs, prompt text, generated content, or reasoning content.
- Streaming parser support is limited to OpenAI-compatible `delta.content`, `delta.reasoning`, `delta.reasoning_content`, and `data: [DONE]` events. Provider-specific semantics stay inside `internal/providers`.
- Non-2xx response bodies are converted to status, body byte count, and SHA-256 hash only.
- When streaming primary smoke fails, the report records a deterministic mock fallback marker and metric so the failure is visible without pretending the paid provider succeeded.
- `--output-dir` reports must use the A21 report namespace and reject legacy X21/V21-looking paths through the shared report-dir guard.
- Unknown provider names are redacted to `unknown_provider`.
- Baidu/Huawei provider names are redacted to `blocked_provider` and fail.
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
