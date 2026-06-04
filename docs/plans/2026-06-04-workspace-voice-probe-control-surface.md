# A21 Workspace Voice Probe Control Surface

Date: 2026-06-04
Owner: Codex main control thread
Status: completed

## Transition

`T-WORKSPACE-VOICE-PROBE-CONTROL-SURFACE-001`

## Current State

`/workspace` now configures workspace scope, source/read metadata, roleplay
profile/memory/voice selection, provider voice-chain selection, and wake-word
intent through existing safe Gateway contracts. The remaining gap in this cut
is product-console evidence that the selected frontend configuration is visible
on a dialogue-path probe without running real providers, V21, firmware, serial,
or hardware.

## Target State

`/workspace` exposes a safe Voice Probe panel:

- Roleplay probe uses the existing fast companion boundary and displays only
  route/status, selected role soul, scenario, voice profile, memory count, and
  trace markers.
- Professional probe uses the existing mock professional route and displays
  only route/status, trace marker count, professional read-record metadata, and
  failure/completion status.
- The page never renders raw utterance text, prompt body, evidence body,
  provider result, private location, credential, document content, or audio
  bytes.

## Trigger

Internal test 4 must move beyond pure configuration: roleplay, voice clone,
memory, provider setup, wake intent, and professional mode must be observable
from the product console as runtime dialogue-path metadata without regressing
internal test 3 endpoint voice behavior.

## Action

1. Add a Voice Probe section to `/workspace`.
2. Add a roleplay probe action that posts to the existing
   `/v1/fast-companion/turn` no-provider local boundary and renders only safe
   roleplay runtime fields.
3. Add a professional probe action that posts to the existing `/v1/mock-turn`
   professional route and refreshes `/v1/professional-read-records` filtered by
   the generated trace id.
4. Add trace refresh through existing `/v1/traces`, rendering only marker names
   and summary counts.
5. Extend safe metadata export with the last probe summary.
6. Update page tests and control documents.

## Acceptance

- `GET /workspace` includes Voice Probe controls and uses only existing
  Gateway routes.
- Roleplay probe displays the selected role soul, scenario, voice profile,
  memory readiness/count, prompt readiness, and safe route/status.
- Professional probe displays trace/read-record status and does not display raw
  query text or evidence bodies.
- Page tests still reject forbidden terms and external URLs.
- Focused Gateway tests pass.
- Playwright verifies roleplay and professional probe behavior on desktop and
  mobile with no horizontal overflow.
- `git diff --check` passes.
- `GOMAXPROCS=2 make verify` passes.

## Failure State

Stop if this slice requires a new backend route, real provider execution, real
V21 execution, document indexing, audio capture, firmware build, flash, serial,
NVS, ECS secret edits, prune/gc, or physical hardware action.

## Rollback Path

Revert only this transition's workspace page, page test, plan, and control-doc
updates. Do not revert internal test 3 endpoint voice/protocol changes or the
prior internal test 4 workspace, roleplay, voice-chain, and wake-word console
cuts.

## Next State

`WORKSPACE-VOICE-PROBE-CONTROL-SURFACE-READY`

## Verification - 2026-06-04

- `go test ./internal/gateway -run 'TestWorkspaceConsolePageServed' -count=1`:
  passed.
- `go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestFastCompanionHybridRoutesLocalAudioFrontendToTextStreamBoundary|TestProfessionalReadRecordsCompleteWithRedactedScopeMetadata|TestProfessionalReadRecordsFailSafelyWhenV21Unavailable|TestProfessionalVoiceTrigger|TestRoleplayProfile|TestVoiceChainProfiles|TestWakeWord|TestProfessionalWorkspace|TestWorkspaceDocumentUpload|TestWorkspaceIndex|TestWorkspaceSource' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- Playwright opened `http://127.0.0.1:21080/workspace` on a local Gateway,
  selected `a21_roleplay_wry_peer`, `engineer_pushback`, and
  `a21_voice_clone_default`, saved one bounded memory hint, and ran the
  roleplay probe.
- Roleplay probe observed `route=fast_companion_hybrid`, trace event count
  `14`, roleplay summary
  `a21_roleplay_wry_peer / engineer_pushback / prompt=true`,
  `voice=a21_voice_clone_default`, and `memory=ready / 1`.
- Professional probe observed `route=professional_mock_turn`, trace event count
  `13`, and one `a21-professional-read-*` record with
  `status=completed`, `query_scope=public_only`, and
  `workspace_status=searchable` in the host-local default mock path.
- Desktop evidence:
  `.a21-run/evidence/workspace-voice-probe-desktop.png`.
- Mobile evidence:
  `.a21-run/evidence/workspace-voice-probe-mobile.png`.
- Playwright JSON evidence:
  `.a21-run/evidence/workspace-voice-probe-playwright.json`.

No provider, real V21, ECS, firmware build, serial, NVS, prune/gc, flash, or
physical hardware action occurred.
