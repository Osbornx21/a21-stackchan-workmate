# 2026-06-04 - Selected Voice Chain Readiness Ingress

Status: completed.
Owner: A21 control tower.
Transition: `T-VOICE-CHAIN-EVIDENCE-001-SELECTED-VOICE-CHAIN-READINESS-INGRESS`.
Created: 2026-06-04 CST.

## Goal

Let product readiness and server-side readiness ingest the existing
`a21.xiaozhi_streaming_provider_readiness.v1` static no-execute capability
report for the currently selected A21 voice chain.

This closes a bookkeeping gap: A21 already has a CLI that classifies selected
ASR, LLM, and TTS streaming/provider capability without executing providers,
but the product readiness ledger currently only sees the Gateway selector and
host-loopback/bench evidence.

## Current State

- Branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Source HEAD at plan start:
  `4989f9a feat(gateway): add workspace voice probe`.
- Working tree at plan start:
  clean and synced with origin.
- Internal test 3 endpoint voice/protocol changes are preserved.
- The active internal test 4 workspace console and voice probe are already
  implemented; this plan must not redo those controls.

## Target State

- `a21 product-readiness` accepts
  `--voice-chain-readiness-report <report.json>`.
- `a21 product-readiness --use-latest-reports` discovers the newest
  `a21-xiaozhi-streaming-provider-readiness-*.json` report.
- `a21 server-side-readiness-bundle` passes the same evidence through to the
  underlying product readiness report.
- The product report exposes whether static capability evidence is available,
  whether it matches the selected Gateway voice-chain selector, and whether
  ASR/LLM/TTS are static-ready.
- Report output stores only safe metadata: basename source report, profile IDs,
  booleans, execution mode, gate status, and finding codes.
- This evidence does not promote PRD launch readiness, physical acceptance,
  or provider execution by itself.

## Boundaries

- No provider execution.
- No V21 execution.
- No ECS runtime changes.
- No Gateway behavior/protocol changes.
- No firmware build, flash, serial, NVS, or physical hardware action.
- No prune/gc or destructive Git cleanup.
- No report deletion or evidence archive rewrite.
- Keep A21 namespace only.

## Implementation Steps

1. Add a `VoiceChainReadinessReport` option and CLI flag to product readiness
   and server-side readiness bundle.
2. Resolve latest static voice-chain readiness report in `--use-latest-reports`.
3. Add a defensive loader for `a21.xiaozhi_streaming_provider_readiness.v1`:
   schema/version validation, single JSON object, safe identifiers, redaction
   flags false, and no forbidden keys/values.
4. Attach valid evidence to `productVoiceChainReadiness` only when it matches
   the currently selected ASR/LLM/TTS profiles; mismatches become safe findings.
5. Add focused tests for matched evidence, mismatched evidence, bundle passthrough,
   and latest-report discovery without path/secret leakage.
6. Update protocol/current-control/state/handoff docs.
7. Run focused tests, `git diff --check`, and `GOMAXPROCS=2 make verify`.

## Acceptance

- Focused `internal/app` tests passed.
- `git diff --check` passed.
- `GOMAXPROCS=2 make verify` passed.
- Reports do not contain full URLs, local absolute paths, provider secrets,
  provider outputs, raw prompts, transcripts, document bodies, or audio payloads.
- `launch_ready` and `prd_accepted` remain false unless existing physical and
  launch gates independently pass.

## Rollback

Revert only the files changed by this transition:

- `internal/app/product_demo.go`
- `internal/app/server_side_readiness_bundle.go`
- `internal/app/app_test.go`
- docs updated by this plan

Do not revert internal test 3 protocol/audio commits or internal test 4
workspace console/voice probe commits.
