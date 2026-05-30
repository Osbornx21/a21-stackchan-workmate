# A21 X21 One-Way Reference Gate

Status: active guardrail.

## Purpose

X21 contains hard-won real-device evidence around StackChan voice latency, ASR, TTS, VAD, streaming orchestration, wake word behavior, and physical-device debugging. A21 may read X21 as a frozen oracle for lessons and parameters.

This is a one-way gate. X21 is read-only. A21 must not modify X21, import X21 runtime identity, or recreate X21 architecture.

X21 is also not a mature-wheel source by default. Much of X21's successful behavior came from pragmatic hand-written fixes under delivery pressure. A21 may preserve the lesson, measurement, ordering rule, or failure taxonomy, but the replacement should first look for mature protocols, SDKs, libraries, or framework patterns according to `docs/engineering/A21_MATURE_VOICE_REUSE.md`.

## Allowed

A21 work may borrow:

- measured latency budgets, chunk sizes, VAD thresholds, wake-word thresholds, timeout values, backoff values, and buffering parameters;
- state-machine insights for listening, thinking, speaking, interrupted, wake-word, playback, and reconnect behavior;
- failure taxonomies, acceptance checklists, and hardware-observation lessons;
- provider behavior observations, as long as A21 routes every provider through Provider Spine.

Every borrowed item must be re-expressed through A21 packages, A21 protocol fields, A21 reports, A21 env names, and A21 trace/metric contracts.

## Forbidden

A21 must not recreate or copy:

- `server.py` monolith structure;
- audio-buffer flag soup;
- dual-spine plus feature-flag architecture drift;
- direct vendor hard-coding outside Provider Spine;
- Xiaozhi protocol or identity coupling;
- whole Python modules moved into Go;
- X21 process names, ports, env vars, logs, report names, firmware targets, or artifact paths.

Python from X21 may be read to understand algorithms or parameters, but implementation in A21 must be provider-neutral Go-first code or an explicitly isolated local tool with an ADR.

## Commit Message Rule

If a commit borrows any X21 parameter, algorithm, state-machine rule, or acceptance heuristic, the commit body must include a line in this form:

```text
借鉴 X21 <source path or module> 的 <parameter/algorithm/rule> -> A21 <package/command/doc> provider-neutral rewrite.
```

Examples:

```text
借鉴 X21 backend/app/domain/x21_protocol.py 的 barge-in cancel ordering -> A21 internal/gateway provider-neutral rewrite.
借鉴 X21 firmware wake_word threshold notes 的 wake threshold -> A21 firmware/stackchan guarded A21 config rewrite.
```

If no X21 material was used, do not add this line.

## Review Checklist

Before committing X21-informed work:

- the X21 source was read only;
- no X21 file was edited;
- no X21 runtime identity appears in A21 runtime code;
- no X21 provider, protocol, or firmware upload path bypasses A21 Provider Spine or firmware release discipline;
- the borrowed item is named in the commit body;
- tests or reports prove the A21 rewrite behavior without depending on X21 runtime.
