# A21 Internal Test 4 Environment Switch Handoff

Date: 2026-06-04 11:33 CST
Branch: `codex/a21-hardware-window-20260603-wifi-provisioning-flash`
Last pushed HEAD: `d4a974c docs(control): hand off roleplay expression plan`

This is the no-repeat recovery point for switching development environments.
Read this file first, then run `git status --short --branch`. Do not restart
from the internal test 3 handoff or rescan every historical branch unless this
file contradicts the working tree.

## Current Working Tree

Tracked modified files at this handoff:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`

No untracked files were present before this handoff document was created.

The modified files are a local, uncommitted implementation slice for
`T-INTERNAL-TEST4-ROLEPLAY-OFFICIAL-EXPRESSION-PLAN-001`. They are not pushed
and must not be treated as merged product progress.

Current uncommitted slice:

- Adds `expression_plan` to `/v1/roleplay-profile`.
- Builds a no-send official StackChan action plan from selected role soul,
  scenario, memory readiness, and voice-clone profile.
- Uses existing `stackchantransport.BuildOfficialActionPlan`.
- Keeps `delivery_policy=no_send_plan_only`.
- Keeps aggregate and per-action `physical_accepted=false`.
- Adds simulator readout for expression action/packet counts.
- Adds focused Gateway test coverage for deterministic content and leak
  prevention.

Recorded focused test before this handoff:

```bash
go test ./internal/gateway -run 'TestSimulatorPageServed|TestRoleplayProfileEndpointReturnsOfficialExpressionPlanWithoutSendingHardware|TestRoleplayProfileEndpointSelectsSoulProfile|TestRoleplayProfileEndpointPersistsScenarioVoiceClone|TestRoleplayProfileEndpointSetsRuntimeMemoryHints' -count=1
```

Recorded result: passed.

Not yet done for this slice:

- Protocol docs are not updated for `expression_plan`.
- Project state machine is not updated for expression-plan readiness.
- `docs/plans/2026-06-04-roleplay-official-expression-plan.md` is still
  `Status: active control plan`.
- `docs/engineering/A21_CURRENT_CONTROL.md` is not updated for this slice.
- `docs/agent_handoff_log.md` is not updated with a completion entry for this
  slice.
- `git diff --check` and `GOMAXPROCS=2 make verify` have not been freshly run
  after the expression-plan code change.
- The slice is not committed or pushed.

Recommended handling:

1. If continuing this slice, finish only the missing docs/state/handoff updates,
   then run focused Gateway tests, `git diff --check`, and
   `GOMAXPROCS=2 make verify`.
2. If prioritizing visible product progress, either leave this slice parked as
   uncommitted work or explicitly ask before discarding it. Do not silently
   revert it.

## Already Pushed On Current Mainline

The following current-mainline commits are ancestors of HEAD and do not need to
be repeated:

- `57cdfb7 feat(gateway): accept local workspace document uploads`
- `c2a1260 feat(gateway): add selectable roleplay soul profiles`
- `509d688 feat(gateway): fold speaker volume into mcp control`
- `6a63e71 feat(gateway): track workspace index requests`
- `6498ede feat(app): route v21 bridge through native voice query`
- `976a1d2 feat(gateway): reflect roleplay state in device registry`
- `d4a974c docs(control): hand off roleplay expression plan`

Product meaning:

- A21 now exposes the two launch modes `roleplay` and `professional`.
- Roleplay has safe profile/scenario/memory/voice-clone control surfaces.
- Professional mode has an explicit ritual/cue and safe read-record ledger.
- Workspace APIs can create upload jobs, accept local multipart document
  intake, expose source readiness, and record no-execute index requests.
- A21 `v21-adapter-bridge` now prefers V21 native
  `/internal/v1/knowledge/voice-query` and carries A21 v2 scope fields.
- Device registry reflects selected roleplay state safely.

Why the product still does not feel much more advanced:

- Uploads are stored local only; no parsing, OCR, chunking, embedding, durable
  cloud storage, account binding, or real searchable index exists yet.
- The index ledger is `indexing_requested_no_execute`; it is not searchable
  readiness.
- Professional V21 evidence is adapter-contract evidence, not a permanent cloud
  topology or full tenant ACL release.
- Roleplay soul/memory/voice-clone selection reaches runtime contracts and
  safe state surfaces, but physical expression/audio acceptance is not closed.
- The web/app product surface for normal users is still not built.
- StackChan hardware parity remains partly planned or no-send, not physical
  proof for all surfaces.

## Worker And Branch State

Closed subagent:

- `019e9011-80b2-73c3-a0a8-7390a3e1f153` / Poincare
- Transition: `T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001`
- Result: completed a lightweight already-merged audit without file changes.
- It reported `git diff --check` passed in the main worktree and worker
  worktree.
- It warned that continuing to edit the older worker branch would create
  duplicate/churn because the branch is behind current mainline work.

Branches not ancestors of current HEAD and must not be blindly merged:

- `codex/stackchan-official-hw-parity-gap-map-001`
  (`d02a592 docs(stackchan): map official hardware parity gaps`)
- `codex/stackchan-official-mcp-status-parity-001`
  (`3ec1540 feat(gateway): add xiaozhi mcp status parity`)

Important branch warning:

- A naive diff from those older worker branches to current HEAD shows thousands
  of deletions because the workers were cut before later internal test 4 work.
- Do not merge or reset to them wholesale.
- If a specific change is needed, cherry-pick or manually port only after
  reviewing the exact files and preserving current internal test 3/4 protocol
  changes.

## Still Not Done

Highest product gaps:

1. Real cloud/web/app workspace UX:
   account/device binding, upload list, scope selector, source status,
   citations, delete/export, and hardware access control.
2. Real indexing path:
   parse/chunk/embed/index stored documents, then expose true
   `searchable` readiness behind the A21/V21 adapter contract.
3. V21 merge/release:
   the V21 source-scope worker exists as worker evidence, but current A21
   mainline only proves adapter use of the native voice-query contract.
4. Roleplay expression:
   current local slice only exposes a no-send plan. It does not send avatar,
   motion, RGB, servo, screen, or audio effects to hardware.
5. Physical StackChan PRD acceptance:
   wake, touch/barge-in, playback start/stop, screen/avatar/motion/RGB/servo
   evidence remain separate foreground hardware windows.
6. Custom wake:
   internal test 3 accepted voice-main-chain testing, but full PRD still lacks
   accepted custom wake proof.
7. Durable runtime:
   many Gateway ledgers are memory-only; restart persistence, auth, tenant ACL,
   and cloud deployment topology remain open.

## Next Recommended Mainline Order

1. Stop repeating broad audits. Start from this handoff and current `git
   status`.
2. Decide whether to finish or park the local expression-plan slice.
3. If finishing it: complete docs/state/handoff, verify, commit, push.
4. Then move to visible product progress:
   - build the web/app workspace surface over existing safe Gateway APIs, or
   - implement the real index adapter path behind A21/V21 v2 scope, or
   - schedule a foreground hardware window for expression/touch/barge-in
     physical evidence.
5. Keep internal test 3 protocol/audio/firmware changes intact. Do not revert
   accepted endpoint voice changes.

## Explicit Do-Not-Repeat List

- Do not re-read every historical plan before acting.
- Do not re-audit internal test 3 from scratch unless a regression is observed.
- Do not treat the two older StackChan parity branches as ready merge sources.
- Do not prune or gc Git loose objects.
- Do not flash firmware, write serial/NVS, or touch hardware without an
  explicit foreground hardware window.
- Do not run provider, V21, ECS, or real indexing work just to make a doc claim.
- Do not claim full PRD readiness until physical StackChan evidence and real
  workspace/index/V21 gates are closed.
