# A21 Workspace Roleplay Control Surface

Date: 2026-06-04
Owner: Codex main control thread
Status: completed

## Transition

`T-WORKSPACE-ROLEPLAY-CONTROL-SURFACE-001`

## Current State

Gateway already exposes safe runtime contracts for roleplay:

- `GET/POST/PUT /v1/roleplay-profile` selects role soul, scenario, voice
  profile, and bounded memory hints.
- Roleplay prompt composition already reaches fast-companion and stock
  Xiaozhi voice turns through runtime-only prompt input.
- The selected safe voice profile already reaches the provider-neutral voice
  pipeline and TTS adapter request.
- The simulator has controls for these values, but the host-local
  product-oriented `/workspace` console only shows a roleplay expression
  readout.

## Target State

`/workspace` becomes the first product console where a user can configure the
default embodied roleplay experience: role soul, scenario, voice profile, and
bounded memory hint. Saving the controls must reuse the existing roleplay
contract and keep API/page responses redacted.

## Trigger

The active sprint objective asks for provider/voice clone/wake word/role soul/
role memory frontend configuration that is reflected in voice conversation,
with stronger role immersion and no regression of internal test 3 endpoint
voice behavior.

## Action

1. Add a roleplay setup section to `/workspace`.
2. Populate role soul and scenario options from `/v1/roleplay-profile`.
3. Populate voice options from `/v1/voice-chain-profiles`.
4. Save roleplay profile/scenario/voice/memory hint through
   `/v1/roleplay-profile`.
5. Clear roleplay memory through `clear_memory=true`.
6. Show only safe readiness/status fields: selected IDs, prompt-input
   readiness, memory status/count, expression plan packet/action counts, and
   physical acceptance false.

## Acceptance

- `GET /workspace` includes roleplay controls for role soul, scenario, voice
  profile, memory hint, save, and clear.
- Saving a roleplay setup through the page updates
  `/v1/roleplay-profile` and reflects selected role soul/scenario/voice plus
  memory readiness/count in the page.
- Voice profile selection remains mediated by the existing Gateway roleplay/
  voice-chain contracts; no provider key or voice data is exposed.
- Page tests still reject forbidden terms and external URLs.
- Focused Gateway tests pass.
- `git diff --check` passes.
- `GOMAXPROCS=2 make verify` passes.
- Playwright verifies desktop and mobile render, save, clear, and no
  horizontal overflow.

## Failure State

Stop if the slice requires a new backend route, provider execution, V21
execution, raw memory/prompt display, voice data display, firmware, serial,
NVS, or physical hardware action.

## Rollback Path

Revert only this transition's `/workspace` page, tests, plan, and control docs.
Do not revert internal test 3 endpoint voice/protocol changes or previous
internal test 4 roleplay/professional contracts.

## Next State

`WORKSPACE-ROLEPLAY-CONTROL-SURFACE-READY`

## Verification - 2026-06-04

- `go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestRoleplayProfile|TestVoiceChainProfiles|TestWorkspaceDocumentUpload|TestWorkspaceIndex|TestWorkspaceSource|TestProfessionalWorkspace' -count=1`:
  passed.
- `git diff --check`: passed.
- Playwright opened `http://127.0.0.1:21080/workspace` on a local Gateway,
  selected `a21_roleplay_wry_peer`, selected `engineer_pushback`, selected
  `a21_voice_clone_default`, saved one bounded roleplay memory hint, confirmed
  `/v1/roleplay-profile` returned `memory_count=1`,
  `prompt_composed=true`, `expression_actions=4`, and
  `physical_accepted=false`, then cleared memory and exported safe workspace
  metadata.
- Desktop evidence: `.a21-run/evidence/workspace-roleplay-control-desktop.png`.
- Mobile evidence: `.a21-run/evidence/workspace-roleplay-control-mobile.png`.
- Export evidence: `.a21-run/evidence/a21-workspace-roleplay-export.json`.

No provider, V21, ECS, firmware, serial, NVS, prune/gc, flash, or physical
hardware action occurred.
