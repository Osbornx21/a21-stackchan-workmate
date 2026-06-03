# 2026-06-04 - V21 Professional Execution Validation

Status: accepted for local adapter-boundary execution; public Gateway topology
and physical PRD gates remain open.
Owner: A21 control tower.
Transition: `T-V21-PROFESSIONAL-EXECUTION-001`.
Created: 2026-06-04 CST.

## Goal

Close the server-side V21 professional execution gate through the explicit
A21/V21 adapter contract, without importing V21 internals into A21, leaking
queries or evidence, or weakening the remaining physical StackChan PRD gates.

This transition is about professional evidence only. It is not a firmware,
provider-selector, Xiaozhi protocol, or physical-audio acceptance transition.

## Current State

- Branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Current source HEAD:
  `20d11a0 docs(control): record stepfun cloud-edge evidence`.
- StepFun route eligibility and cloud-edge host voice evidence are accepted for
  their scoped server-side use.
- Fresh current reports:
  - `reports/a21-provider-smoke-20260604-035200-977132343.json`
  - `reports/a21-xiaozhi-streaming-provider-readiness-20260604-035317-1780516397705439206.json`
  - `reports/a21-xiaozhi-voice-bench-20260604-035502.222374820.json`
  - `reports/a21-product-readiness-20260604-040153.json`
  - `reports/a21-server-side-readiness-bundle-20260604-040207.json`
- Product/server-side readiness remain blocked by:
  - `v21_professional_execution`
  - `physical_stackchan_prd_acceptance`
- Control snapshot at plan creation:
  - `A21_V21_ADAPTER_URL`: missing in the current shell.
  - `A21_V21_BACKEND_URL`: missing in the current shell.
  - `A21_V21_ADAPTER_TOKEN`: missing in the current shell.
  - No process is listening on local `127.0.0.1:21121`.
  - Existing V21 reports are from 2026-06-01/02 and are not current evidence
    for the 2026-06-04 launch sprint.

## Execution Update - 2026-06-04 04:36 CST

- User supplied the V21 control thread
  `codex://threads/019e68bc-4fb6-7ce0-ad67-5b1dd0de478f`.
- Thread evidence confirmed V21 is a local/LAN Docker Compose service, with
  the demo backend intended on `18081` and old loopback backend history on
  `18080`.
- Docker Desktop was not running; it was started and the V21 LAN demo stack was
  brought up from `/Users/jiyurun/Documents/v21-knowledge-platform`.
- V21 local health passed at `127.0.0.1:18081` and LAN health passed at
  `192.168.1.20:18081`.
- V21 runtime config reported retrieval and LLM configured.
- V21 active collection discovery returned one collection with an active
  release.
- A21 bridge was started temporarily at `127.0.0.1:21121` pointing to
  `http://127.0.0.1:18081`.
- Adapter health passed.
- Fresh executed adapter smoke:
  `reports/a21-v21-adapter-smoke-20260604-043456.json`, `passed`,
  `configured=true`, `executed=true`, `redaction_ok=true`, evidence count `5`,
  speech block count `1`, screen card count `1`, follow-up count `1`,
  duration `726.483 ms`.
- Fresh readiness collectors with the latest StepFun/cloud-edge evidence and
  V21 report:
  `reports/a21-product-readiness-20260604-043528.json` and
  `reports/a21-server-side-readiness-bundle-20260604-043528.json`.
- The server-side bundle now marks provider, V21, and host voice evidence ready;
  remaining local-shell missing evidence is `gateway`, `wake_word`, and
  `voice_chain_selector`.
- The temporary A21 bridge process was stopped after evidence collection.
- V21 Docker containers remain running; V21 repository dirty worktree was not
  modified or reverted.

## Target State

- A fresh redacted V21 adapter smoke report proves real professional execution
  through the A21 adapter boundary:
  - schema `a21.v21_adapter_smoke.v1`
  - `status=passed`
  - `executed=true`
  - professional contract matches A21's required mode, latency profile,
    answer style, privacy scope, and max first-response budget
  - positive confidence and non-zero evidence/speech/card/follow-up counts
  - no prompt, query text, response text, evidence body, credential, full URL,
    proxy value, local path, transcript, or provider output is stored
- Product readiness and server-side readiness absorb the fresh report and no
  longer list V21 professional execution as a server-side blocker.
- Canonical launch state remains honest:
  `launch_ready=false` and `prd_accepted=false` until physical StackChan PRD
  evidence is collected.

## Not In Scope

- No V21 internal source reuse or V21 runtime identity in A21.
- No direct reads from V21 internals as an implementation dependency.
- No provider key changes, firmware build, flash, NVS write, or Xiaozhi
  protocol edits.
- No use of old V21 reports to close the current gate unless they are
  explicitly revalidated by a fresh A21 readiness run and still meet the current
  contract.
- No health-only V21 claim. Adapter `/healthz` is necessary but insufficient.

## Worker Execution Task

If this transition is delegated, the worker receives one task:

- Discover or run the A21/V21 adapter boundary, execute exactly one redacted
  `v21-adapter-smoke --execute` when the boundary is verified, and produce a
  short handoff with report basenames and readiness deltas only.

Worker boundary conditions:

- No code edits unless the adapter contract itself rejects valid current
  evidence; stop and report before changing code.
- No secret, query, evidence body, transcript, raw response, URL credential,
  proxy value, or local path in stdout, reports, docs, or chat.
- No provider execute beyond the V21 adapter smoke.
- No hardware, firmware, NVS, Gateway protocol, or ECS provider-secret mutation.
- Do not delete, move, prune, or rewrite historical reports or worktrees.

Required worker summary format:

```text
What changed:
Files changed:
Reports produced:
Tests/commands run:
Result:
Deviations from plan:
Remaining blockers:
Next suggested action:
```

## Execution Plan

1. Record this scoped transition in `docs/project_state_machine.md`,
   `docs/engineering/A21_CURRENT_CONTROL.md`,
   `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`, and
   `docs/agent_handoff_log.md`.
2. Run safe local discovery only:
   - env-name presence checks
   - local `21121` listener check
   - CLI help / contract checks
   - latest-report age check
3. If `A21_V21_ADAPTER_URL` is configured and health succeeds, run:

   ```bash
   go run ./cmd/a21 v21-adapter-smoke --execute --output-dir reports
   ```

4. If only a V21 backend boundary is available, start the A21 bridge with
   explicit env/flags, then run the same adapter smoke against the bridge:

   ```bash
   go run ./cmd/a21 v21-adapter-bridge --addr 127.0.0.1:21121 --v21-url <redacted-v21-backend>
   A21_V21_ADAPTER_URL=http://127.0.0.1:21121 go run ./cmd/a21 v21-adapter-smoke --execute --output-dir reports
   ```

5. Re-run server-side readiness with the fresh V21 report plus the latest
   StepFun/cloud-edge reports.
6. If adapter/backend is missing, produce a fresh dry-run report and a precise
   operator ask instead of blocking silently.
7. Commit only control-doc and redacted report-pointer updates after
   verification.

## Acceptance Conditions

- `git diff --check` passes.
- Focused readiness tests pass if code is touched; no code touch is expected.
- Fresh V21 adapter smoke report is passed and executed.
- Product/server-side readiness source fields point to the fresh V21 report.
- V21 is no longer a server-side blocker.
- Physical StackChan PRD evidence remains a separate blocker.

Acceptance status:

- Accepted for V21 local adapter-boundary execution evidence.
- Not accepted as permanent public-Gateway V21 topology.
- Not accepted for physical StackChan PRD launch.

## Failure States

- `adapter_boundary_missing`: no `A21_V21_ADAPTER_URL`, no bridge listener, and
  no approved backend URL.
- `adapter_health_failed`: adapter URL is configured but `/healthz` fails.
- `backend_unavailable`: bridge cannot start because the V21 backend health or
  active collection discovery fails.
- `adapter_smoke_failed`: the adapter query path runs but the contract,
  confidence, counts, latency, or redaction checks fail.
- `redaction_violation`: any report/stdout/doc/chat includes restricted query,
  response, evidence, credential, proxy, path, or transcript content.

## Rollback Path

This transition should not mutate runtime state except for starting a local
adapter bridge process when explicitly needed. Stop the bridge process if it is
started. Do not revert internal test 3 protocol or StepFun evidence commits.
Revert only this plan/control-doc update if it contains incorrect state.

## Operator Ask If Blocked

Provide one of the following, preferably through an ignored env file or a
working shell session rather than chat:

- A running A21 adapter boundary:

  ```bash
  export A21_V21_ADAPTER_URL=http://127.0.0.1:21121
  ```

- Or an approved V21 backend boundary that the A21 bridge may call:

  ```bash
  export A21_V21_BACKEND_URL=http://127.0.0.1:18080
  export A21_V21_COLLECTION_ID=<active-collection-id-if-auto-discovery-is-not-allowed>
  ```

Do not provide V21 document contents, query answers, raw evidence, or secrets in
chat. The A21 adapter smoke stores only redacted status/count/timing evidence.

## Next State

On success:

- `S-PUBLIC-GATEWAY-STEPFUN-CLOUD-EDGE-SERVER-SIDE-V21-READY-PHYSICAL-PENDING`

On missing boundary:

- `S-PUBLIC-GATEWAY-STEPFUN-CLOUD-EDGE-SERVER-SIDE-EVIDENCE-READY-V21-BOUNDARY-BLOCKED-PHYSICAL-PENDING`
