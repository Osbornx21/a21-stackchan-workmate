# 2026-06-04 - StepFun Route Eligibility Promotion

Status: accepted; V21 and physical launch gates remain open.
Owner: A21 control tower.
Transition: `T-STEPFUN-ROUTE-001-LAUNCH-POLICY-PROMOTION`.
Created: 2026-06-04 CST.

## Goal

Promote the built-in StepFun text-stream profile from compatibility-only
candidate to the launch-policy route-eligible LLM profile, so executed StepFun
`provider-smoke` evidence can be absorbed by product readiness after the public
Gateway selector is already switched to StepFun.

This transition does not change the internal test 3 `/v1/xiaozhi` protocol,
firmware lane, product flash state, physical acceptance rules, provider secret
values, or the selected public Gateway transport.

## Current State

- Current branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Internal test 3 protocol and endpoint-side voice acceptance are treated as
  the immutable baseline for this slice.
- Public Gateway `http://47.103.57.217` is healthy.
- Public `/v1/voice-chain-profiles` now reports selected LLM `stepfun`.
- Product device `44:1b:f6:e2:6a:60` is online on
  `ws://47.103.57.217/v1/xiaozhi`.
- Executed StepFun provider smoke exists locally:
  `reports/a21-provider-smoke-20260604-023711-678466985.json`.
- The report passed execution and streaming checks but has
  `route_eligible=false`, so readiness rejects it as `provider_smoke`.

## Execution Update - 2026-06-04 03:33 CST

- Source promotion is committed as
  `3741c4a feat(providers): promote stepfun route eligibility`.
- Control/audit commits through `5dba606` are pushed to
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- ECS has not been updated to `3741c4a` or newer from this thread because SSH
  control-plane access is unavailable.
- Public Gateway paths currently return empty HTTP replies even though TCP
  ports are reachable, so fresh runtime evidence cannot be collected until ECS
  access or operator intervention is restored.

## Execution Update - 2026-06-04 04:02 CST

- ECS access was recovered through the approved jump path.
- Commit `d9362a7 feat(readiness): accept cloud-edge xiaozhi evidence` was
  deployed to ECS via `/opt/a21.next` safe swap.
- ECS `provider.env` remains root-owned and mode `600`; the runtime selector
  now uses only A21 profile IDs for DashScope ASR, StepFun LLM, and DashScope
  TTS.
- Fresh StepFun provider smoke:
  `reports/a21-provider-smoke-20260604-035200-977132343.json`, `passed`,
  `executed=true`, `stream=true`, `route_eligible=true`.
- Fresh static provider readiness:
  `reports/a21-xiaozhi-streaming-provider-readiness-20260604-035317-1780516397705439206.json`,
  `gate_status=passed`, `stepfun_selected`.
- Fresh Xiaozhi host bench:
  `reports/a21-xiaozhi-voice-bench-20260604-035502.222374820.json`,
  `candidate_host_only`, `cloud_edge`, `failure_count=0`.
- Fresh product/server-side readiness reports absorb provider and host voice
  evidence:
  `reports/a21-product-readiness-20260604-040153.json` and
  `reports/a21-server-side-readiness-bundle-20260604-040207.json`.
- Remaining launch blockers are now `v21_professional_execution` and
  `physical_stackchan_prd_acceptance`.

## Target State

- Built-in `stepfun` is explicit `route_eligible=true`.
- Provider catalog tests name StepFun in the route-eligible set.
- Documentation no longer describes StepFun as compatibility-only for the
  current launch policy.
- Product readiness can absorb a fresh executed StepFun provider-smoke report
  as real provider evidence.
- Full PRD launch remains blocked until physical playback/stop/audible evidence
  is present.

## Stop Rules

- Do not revert any internal test 3 Gateway protocol, firmware, or endpoint
  acceptance changes.
- Do not modify `/v1/xiaozhi` message order, listen/speak boundaries, Opus
  queueing, suppression, or barge-in behavior in this transition.
- Do not write provider key values, model values, transcripts, prompts, raw or
  base64 audio, proxy secrets, or local secret paths into repo, reports, logs,
  stdout, or chat.
- Do not flash firmware or write NVS.
- Do not claim `launch_ready=true` or `prd_accepted=true` without physical
  evidence.

## Action Plan

1. Update the built-in provider catalog so StepFun is route-eligible.
2. Update catalog tests to keep the route-eligible set explicit.
3. Update stale provider/readiness docs and current control records.
4. Run focused provider/app tests and `GOMAXPROCS=2 make verify`.
5. Commit the small promotion patch.
6. Deploy the commit to ECS through the existing `/opt/a21.next` safe swap.
7. Run fresh StepFun provider smoke on ECS and copy only the redacted report
   into local `reports/`.
8. Re-run product readiness and server-side readiness with the new report.

## Acceptance Conditions

- No tracked protocol or firmware files are reverted.
- Focused provider and readiness tests pass.
- `make verify` passes.
- A fresh StepFun provider smoke report has `status=passed`,
  `executed=true`, `stream=true`, and `route_eligible=true`.
- Product/server-side readiness no longer lists `provider_smoke` as the active
  server-side blocker.
- Canonical launch decision remains honest about remaining physical evidence
  gaps.

Acceptance status:

- Accepted for StepFun route eligibility and cloud-edge host voice evidence.
- Not accepted for full PRD launch.
- Remaining gates: V21 professional execution and physical StackChan playback /
  stop / audible acceptance.

## Failure States

- `test_failed`: focused or full verification fails.
- `readiness_still_rejects_stepfun`: fresh route-eligible smoke is still not
  absorbed by readiness.
- `public_gateway_deploy_failed`: remote tests, build, restart, or public
  health checks fail.
- `physical_gap_misclaimed`: any report or doc claims physical PRD acceptance
  without playback/stop/audible evidence.

## Rollback

If deploy or public health fails, restore `/opt/a21.prev` on ECS using the
existing handoff rollback pattern. Do not alter firmware, provider secrets, or
product flash state during rollback.

## Next State

- `S-PUBLIC-GATEWAY-STEPFUN-CLOUD-EDGE-SERVER-SIDE-EVIDENCE-READY-V21-AND-PHYSICAL-PENDING`.
