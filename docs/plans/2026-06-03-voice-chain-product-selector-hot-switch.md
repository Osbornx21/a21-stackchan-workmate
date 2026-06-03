# 2026-06-03 - Voice Chain Product Selector And Hot Switch

Status: implemented locally; verification in progress.
Owner: A21 control tower.
Transition: `T-VOICE-CHAIN-SELECTOR-001-CASCADE-REALTIME-HOTSWITCH`.

## Trigger

The user wants the frontend to expose two dialogue-chain modes:

- cascade: ASR -> LLM -> fixed TTS, with selectable ASR and LLM providers and a fastest-LLM recommendation.
- realtime: end-to-end realtime provider, with visible provider selection.

Voice cloning must be separate from product mode and transport mode. When voice clone is selected, A21 should choose a default TTS profile and show voice/tonality names instead of raw provider IDs.

## Current State

- `voice_mode` already has `dialogue` / `professional`.
- `gateway_profile` already has `public_wss` / `mac_local`.
- `cloud_voice_profile` is currently a no-execute catalog/selector and does not mutate runtime ASR/LLM/TTS selection.
- `/v1/realtime/session` exists and can use `A21_PROVIDER_PRIMARY=openai_realtime|doubao_realtime|doubao_tts_realtime`, but it is not exposed as the unified frontend dialogue-chain selector.
- `/v1/xiaozhi` cascade runtime is built from env at Gateway startup; it is not yet operator-hot-switchable through an API.

## Target State

- Add `GET/POST /v1/voice-chain-profiles` as the product selector for dialogue-chain shape.
- Keep it independent from `voice_mode` and `gateway_profile`.
- `cascade` exposes ASR and LLM choices and keeps TTS fixed to the current default realtime TTS profile.
- `realtime` exposes provider choices and updates the provider used by `/v1/realtime/session`.
- Voice clone exposes safe voice names/profiles separately and maps to a default TTS profile without storing reference text/audio or secrets.
- Runtime hot switch updates Gateway in-memory state and device registry immediately; it does not write provider keys into repo, firmware, or public reports.

## Non-Goals

- Do not claim physical PRD green.
- Do not add new provider credentials.
- Do not execute providers in tests.
- Do not make professional mode use opaque realtime providers.
- Do not replace the existing public/mac Gateway selector.

## Acceptance

- API lists `cascade` and `realtime` modes with safe provider IDs and labels.
- POST cascade can change ASR and LLM and rebuild `/v1/xiaozhi` cascade runner in memory.
- POST realtime can change visible realtime provider and rebuild the realtime voice provider in memory.
- Selecting a voice-clone profile updates only the voice/tts selector, not `voice_mode` or `gateway_profile`.
- Simulator displays and saves the selector.
- Device registry includes current voice-chain mode and selected providers.
- Focused gateway tests and `git diff --check` pass.

## Implementation Notes

- Added `GET/POST/PUT /v1/voice-chain-profiles`.
- Added simulator controls for chain mode, cascade ASR, cascade LLM, realtime
  provider, and voice/clone selection.
- `cascade` defaults to DashScope realtime ASR, StepFun LLM, and fixed
  DashScope realtime TTS.
- `deepseek` remains visible as fallback and is not the recommended LLM.
- `realtime` hot-switches the selected realtime provider by updating the
  existing server-side runtime gate `A21_GATEWAY_VOICE_PROVIDER=selected` plus
  `A21_PROVIDER_PRIMARY`.
- Voice-clone selection maps the effective TTS profile to `voice_clone_cli`
  while keeping `voice_mode`, `gateway_profile`, and `cloud_voice_profile`
  separate.

## Rollback

Revert the endpoint, simulator controls, runtime setter, and docs. Existing `voice_mode`, `gateway_profile`, `cloud_voice_profile`, and env-startup runtime remain intact.
