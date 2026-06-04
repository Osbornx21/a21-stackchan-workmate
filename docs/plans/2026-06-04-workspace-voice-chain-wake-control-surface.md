# A21 Workspace Voice Chain And Wake Control Surface

Date: 2026-06-04
Owner: Codex main control thread
Status: completed

## Transition

`T-WORKSPACE-VOICE-CHAIN-WAKE-CONTROL-SURFACE-001`

## Current State

Gateway already exposes safe frontend/operator contracts for voice chain and
wake-word configuration:

- `GET/POST/PUT /v1/voice-chain-profiles` selects `cascade` or `realtime`,
  cascade ASR/LLM profiles, realtime provider, and safe voice profile. This is
  a runtime selector and does not itself execute a provider.
- `GET/PUT /v1/wake-word` stores wake-word intent. Built-in Xiaozhi WakeNet
  remains active unless a separately guarded firmware build/flash/evidence
  path promotes a custom MultiNet profile.
- The simulator has controls for these contracts, and `/workspace` now has
  workspace and roleplay controls, but it still lacks a product-console
  provider/voice-chain/wake-word setup surface.

## Target State

`/workspace` becomes the product console for the full front-configurable voice
setup: provider lane, ASR/LLM/realtime provider, effective TTS/voice readout,
and wake-word intent. The page must be honest that saving these values does not
run providers, activate custom wake-word firmware, flash hardware, or prove
physical acceptance.

## Trigger

The active sprint objective explicitly asks for provider, voice clone, wake
word, role soul, and role memory frontend configuration that can be reflected
in the voice conversation while preserving internal test 3 endpoint voice
experience.

## Action

1. Add a voice-chain setup section to `/workspace`.
2. Populate voice-chain mode, ASR, LLM, and realtime provider options from
   `/v1/voice-chain-profiles`.
3. Save the selected voice-chain settings back through
   `/v1/voice-chain-profiles`.
4. Add a wake-word setup section to `/workspace`.
5. Load/save/reset wake-word intent through `/v1/wake-word`.
6. Show safe status fields only: selected profile IDs, effective TTS profile,
   hot-switch readiness, active built-in wake phrase, desired custom phrase,
   firmware build requirement, runtime hot-swap false, and custom runtime
   false.

## Acceptance

- `GET /workspace` includes voice-chain controls for chain mode, ASR, LLM,
  realtime provider, save, and safe selected/effective readouts.
- `GET /workspace` includes wake-word controls for mode, desired phrase,
  desired pinyin, threshold, save, reset, and safe activation truth readouts.
- Saving voice-chain settings updates `/v1/voice-chain-profiles` and does not
  call a provider.
- Saving custom wake-word intent updates `/v1/wake-word` and continues to show
  built-in Xiaozhi WakeNet as active with custom firmware pending.
- Page tests still reject forbidden terms and external URLs.
- Focused Gateway tests pass.
- `git diff --check` passes.
- `GOMAXPROCS=2 make verify` passes.
- Playwright verifies desktop and mobile render, voice-chain save, wake-word
  save/reset, and no horizontal overflow.

## Failure State

Stop if the slice needs a new backend route, provider execution, V21 execution,
raw audio/voice data, firmware build, flash, serial, NVS, or physical hardware
action.

## Rollback Path

Revert only this transition's `/workspace` page, tests, plan, and control docs.
Do not revert internal test 3 endpoint voice/protocol changes or previous
internal test 4 roleplay/professional/workspace contracts.

## Next State

`WORKSPACE-VOICE-CHAIN-WAKE-CONTROL-SURFACE-READY`

## Verification - 2026-06-04

- `go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestVoiceChainProfiles|TestWakeWord|TestRoleplayProfile|TestWorkspaceDocumentUpload|TestWorkspaceIndex|TestWorkspaceSource|TestProfessionalWorkspace' -count=1`:
  passed.
- `git diff --check`: passed.
- Playwright opened `http://127.0.0.1:21080/workspace` on a local Gateway,
  selected `cascade`, `doubao_asr_realtime`, `stepfun`, and
  `openai_realtime`, saved voice-chain settings, and confirmed
  `/v1/voice-chain-profiles` returned the selected profiles with
  `hot_switch=true`.
- Playwright saved custom wake-word intent `custom_multinet` for
  `小阿二一` / `xiao a er yi` with threshold `35`, confirmed
  `/v1/wake-word` returned `runtime_status=pending_firmware_build`,
  `firmware_build_required=true`, `firmware_status=custom_pending_firmware`,
  `active_runtime_profile=builtin_xiaozhi_wakenet`,
  `runtime_hot_swap_supported=false`, and `custom_runtime_active=false`, then
  reset back to `builtin_xiaozhi`.
- Desktop evidence: `.a21-run/evidence/workspace-voice-wake-control-desktop.png`.
- Mobile evidence: `.a21-run/evidence/workspace-voice-wake-control-mobile.png`.
- Export evidence: `.a21-run/evidence/a21-workspace-voice-wake-export.json`.

No provider, V21, ECS, firmware build, serial, NVS, prune/gc, flash, or
physical hardware action occurred.
