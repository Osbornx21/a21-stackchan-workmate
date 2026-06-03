# T-CLOUD-VOICE-001 Pure Cloud Voice Provider Matrix Plan

Status: active branch plan; expanded to no-execute control surface.
Date: 2026-06-03.
Owner: A21 control tower.
Branch: `codex/a21-pure-cloud-voice-matrix`.

## Trigger

The public Gateway is now deployed, and A21 needs to try many pure-cloud voice
schemes quickly:

- Bailian Qwen-TTS realtime, including voice cloning / custom voice IDs.
- Bailian CosyVoice families.
- Doubao voice clone and realtime TTS families.
- MiniMax TTS and voice clone families.

The user wants all of them fully supported, frontend configurable, dispatchable,
and low-latency.

## Current State

- A21 already has explicit `voice_mode` (`dialogue` / `professional`) and
  `gateway_profile` (`public_wss` / `mac_local`) selectors.
- A21 provider spine already separates `text_stream`, `voice_realtime`,
  `voice_hybrid`, `agent_task`, and `local_audio`.
- `doubao_tts_realtime` has an adapter seam and fake runtime smoke, but real
  provider execution has not been run in this branch.
- `voice_clone_cli` is retained for clone/persona voices, including local or
  5080-hosted engines.
- `professional` mode remains V21-only and must not be routed through opaque
  realtime/S2S providers.

## Target State

`S-CLOUD-VOICE-MATRIX-CONTROL-READY`

- A21 has a provider-neutral cloud voice matrix covering Bailian Qwen-TTS,
  Bailian CosyVoice, Doubao, and MiniMax.
- The matrix defines frontend-visible profile IDs, safe config/env names,
  dispatch statuses, latency metrics, redaction rules, and promotion gates.
- Gateway exposes a no-execute `cloud_voice_profile` catalog and selector.
- The simulator can configure the selected cloud voice profile without touching
  `voice_mode` or `gateway_profile`.
- `a21 doctor` reports the cloud voice catalog with env names only.
- Future workers can implement adapters one provider at a time without
  re-arguing architecture or confusing catalog visibility with runtime
  readiness.

## Non-Goals

- Do not execute real providers in this branch.
- Do not start, stop, or reconfigure Gateway.
- Do not touch firmware, NVS, serial ports, or physical hardware.
- Do not change A21 default provider or default voice profile.
- Do not store clone reference audio, reference text, provider output, prompt
  text, transcripts, full URLs, local paths, model values, or secrets.

## Action

1. Recover A21 state and isolate this work in a side branch/worktree.
2. Search prior A21 memory and repo docs for provider-spine, voice-mode,
   Doubao realtime TTS, CosyVoice/clone, MiniMax, and 5080-lab decisions.
3. Check current official docs for Bailian/DashScope, Volcengine Doubao, and
   MiniMax voice APIs.
4. Write `docs/engineering/A21_CLOUD_VOICE_PROVIDER_MATRIX.md`.
5. Update `docs/project_state_machine.md` and `docs/agent_handoff_log.md`.
6. Validate the documentation diff and keep branch state recoverable.
7. Implement `GET/POST /v1/cloud-voice-profiles` as a no-execute safe catalog
   and selector.
8. Add simulator selector/readouts and `a21 doctor` report visibility.
9. Validate focused gateway/app/provider tests plus repository diff checks.

## Acceptance

- New matrix covers all requested provider families.
- Matrix preserves A21 boundaries:
  - `voice_mode` is not provider routing.
  - `gateway_profile` is not provider routing.
  - `professional` remains V21-only.
  - StackChan remains a thin client with no provider keys.
- Matrix defines a frontend/config/downlink path through safe profile IDs and
  server-side dispatch statuses.
- Matrix defines concrete follow-up slices for adapter, smoke, frontend, 5080,
  and physical acceptance work.
- `GET /v1/cloud-voice-profiles` lists Bailian Qwen/Qwen3/CosyVoice/Omni,
  Doubao, and MiniMax profiles with safe IDs, statuses, capabilities, and
  env names only.
- `POST /v1/cloud-voice-profiles` accepts only known safe IDs, rejects
  unknown/legacy values without echoing them, and does not alter `voice_mode`
  or `gateway_profile`.
- Device registry exposes `current_cloud_voice_profile` only as a safe ID.
- Simulator and `a21 doctor` consume the same no-secret catalog surface.
- `git diff --check` passes.

## Failure State

- The branch would fail this transition if it changes runtime defaults, claims
  unexecuted providers are ready, leaks secrets or payloads into docs/reports,
  creates hidden provider routing, or lets `cloud_voice_profile` mutate
  `voice_mode` / `gateway_profile`.

## Rollback

- Revert only this branch's documentation and no-execute control-plane
  additions/updates.
- Main hardware/public-Gateway branches and physical device state are untouched.

## Next State

`S-CLOUD-VOICE-MATRIX-CONTROL-READY` should dispatch workers in this order:

1. Reconcile Doubao realtime TTS against current Volcengine docs and run fake
   plus real lab smoke.
2. Implement Bailian Qwen-TTS realtime fixture adapter.
3. Implement Bailian CosyVoice and voice-clone report command.
4. Implement MiniMax TTS and voice-clone report command.
5. Add server-side safe dispatch and readiness plan endpoint.
6. Run 5080 redacted smoke matrix.
7. Run physical StackChan A/B and barge-in acceptance.

## Worker Boundary Conditions

Allowed:

- Read repo docs/code and memory.
- Search official provider docs.
- Edit docs/plan/handoff/state for this transition.
- Edit Gateway/app/provider control-plane code for no-execute catalog,
  selector, simulator, and doctor visibility.
- Run documentation validation.
- Run focused Go tests and `make verify`.

Forbidden:

- Real provider execution.
- Gateway start/stop/restart.
- V21 execution.
- Firmware build/flash/NVS/serial/hardware.
- Audio playback or raw audio capture.
- Default provider/runtime changes.

## Worker Completion Summary Format

- What changed.
- Files changed.
- Sources checked.
- Tests/validation run.
- Deviations from plan.
- Remaining issues.
- Next suggested worker slice.
