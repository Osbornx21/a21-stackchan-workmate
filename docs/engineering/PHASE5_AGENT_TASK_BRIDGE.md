# Phase 5 AgentTask Bridge

Status: T1/T2 scaffold plus host-only T4 Agent I/O smoke.

## Boundary

The AgentTask bridge is an Agent I/O Layer for long-running external-agent
tasks. It is not an A21 router, second brain, backend orchestrator, realtime
first-response path, StackChan control plane, V21 internals path, or Gateway
runtime writer.

The only configured agent-task profiles in the built-in provider catalog are:

- `hermes_agent`
- `mimo_agent`

Both remain `agent_task` family profiles. They are not route-eligible and are
not realtime-capable. The lane is disabled by default. Readiness requires an
explicit `A21_AGENT_PROVIDER_PRIMARY` value selecting exactly one supported
agent profile. `A21_PROVIDER_PRIMARY` does not configure this lane and cannot
select agent-task profiles.

## Contract

`internal/providers` defines the T1/T2 contract:

- `AgentTaskProvider`
- `AgentTaskRequest`
- `AgentTaskEvent`

`AgentTaskRequest` carries `trace_id`, `session_id`, `task`, and `context`.
`AgentTaskEvent` carries A21 trace/session identity and one of these event
kinds:

- `started`
- `progress`
- `text_delta`
- `tool_call_redacted`
- `result`
- `error`
- `final`

The scaffold includes a fake event stream provider for `sse`, `http`, and
`stdio` style streams. The host-only CLI adds an explicit Hermes/MiMo HTTP smoke
surface, but it remains outside Gateway runtime and realtime first response. It
does not execute shell tools, V21, provider-network calls, StackChan hardware
commands, firmware paths, or `/v1/devices/control`.

## Agent I/O Plan And Smoke

`internal/agentplan` selects the A21-owned execution plan before any external
agent is contacted:

- `native_core`: default workmate path.
- `agent_io`: allowed only in `co_creation` or `roleplay`, with explicit agent
  binding and endpoint configuration.
- `v21_professional`: professional mode only; Agent I/O is blocked and memory
  stays pointer-only.

The planner sends external agents only a task summary or current-turn fallback,
trace/session ids, output constraints, selected task-window memory, and
non-sensitive preferences. It filters private memory, companion history, V21
evidence bodies, raw external-agent output, and sensitive items.

Dry-run plan:

```bash
go run ./cmd/a21 agent-plan --mode co_creation --agent hermes_agent --agent-name Hermes --endpoint-url http://127.0.0.1:21130/a21/agent-task --output-dir reports
make agent-plan
```

Redacted smoke report without network I/O:

```bash
go run ./cmd/a21 agent-io-smoke --endpoint-url http://127.0.0.1:21130/a21/agent-task --output-dir reports
make agent-io-smoke
```

Explicit host-only Hermes acceptance:

```bash
A21_HERMES_AGENT_URL=http://127.0.0.1:21130/a21/agent-task make agent-io-smoke-execute
```

`A21_AGENT_PROVIDER_PRIMARY` selects the agent profile. `A21_HERMES_AGENT_URL`
is the Hermes-specific endpoint shortcut, and `A21_AGENT_IO_ENDPOINT_URL` is the
generic endpoint override. `A21_HERMES_AGENT_KEY` is the Hermes Bearer token,
and `A21_AGENT_IO_API_KEY` is the generic credential override. Endpoint and key
values are never written to stdout or reports; reports keep only a coarse label
such as `loopback:21130`.

## Semantic Mapping

External agent events are mapped into an A21-owned semantic report:

- schema: `a21.agent_task.semantic_report.v1`
- per-event schema: `a21.agent_task.semantic_event.v1`
- preserved fields: kind, final marker, `trace_id`, `session_id`
- redacted evidence: text length, `redacted_text`, `tool_call_redacted`
- optional error code after safe-code normalization

The semantic report must not store agent text, tool payloads, credentials, full
URLs, local paths, provider env values, or raw external-agent control payloads.

## Safety Guard

Agent task requests are rejected or reported with red findings when task or
context indicates:

- firmware command/write
- NVS, flash, raw upload, or write-flash work
- provider environment mutation or exposure
- V21 internals mutation
- Gateway runtime state writes
- `/v1/devices/control`
- physical device paths such as serial ports
- endpoint credentials

This is intentionally stricter than a normal text provider request because the
agent lane may eventually broker long-running tools. Future runtime work must
keep this guard in front of any real adapter.

`agent-io-smoke --execute` uses a direct HTTP client with no ambient proxy
inheritance. When the configured endpoint ends in `/v1`, it treats the endpoint
as an OpenAI-compatible Hermes base URL and posts to `/v1/chat/completions` with
`Authorization: Bearer <A21_HERMES_AGENT_KEY>`. It accepts JSON, SSE, or plain
text responses but stores only HTTP status, safe content type, duration, text
length, event count, planner markers, and redaction booleans. It never stores
task text, memory text, external-agent response text, full URLs, credentials,
provider env values, V21 evidence bodies, firmware commands, or device commands.

## Verification

Current T1/T2 verification:

```bash
go test ./internal/providers -run 'AgentTask|ProviderCatalog|ProviderProfile' -count=1
go test ./internal/agentplan ./internal/app -run 'Agent|AgentIO' -count=1
```

The host-only T4 smoke above is the only real external Agent I/O path in this
slice. Future approved work is still required before any Gateway runtime agent
execution, V21 execution, realtime first-response use, or physical StackChan
path can be used.
