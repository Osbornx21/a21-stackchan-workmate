# Phase 5 AgentTask Bridge

Status: T1/T2 scaffold only.

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
`stdio` style streams. It does not execute Hermes, MiMo, OpenClaw, shell tools,
Gateway runtime, V21, provider network calls, or hardware commands.

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

This is intentionally stricter than a normal text provider request because the
agent lane may eventually broker long-running tools. Future runtime work must
keep this guard in front of any real adapter.

## Verification

Current T1/T2 verification:

```bash
go test ./internal/providers -run 'AgentTask|ProviderCatalog|ProviderProfile' -count=1
```

Future T4/T6 work is still required before any real external agent process,
provider execution, V21 execution, Gateway runtime startup, or physical
StackChan path can be used.
