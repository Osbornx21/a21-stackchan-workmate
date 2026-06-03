# 2026-06-03 - Dialogue-First Low-Latency PRD Convergence

Status: active.
Owner: A21 control tower.

## Transition

`T-DIALOGUE-001-LOW-LATENCY-CHAIN-CONVERGENCE`

## Current State

A21 has a stock-compatible `/v1/xiaozhi` path, streaming ASR seams, a
text-stream LLM path, a Doubao realtime TTS seam, and a professional/V21 path.
The PRD and protocol still expose too many product modes, and the current
readiness reports can prove components without naming the intended product
shape clearly.

## Target State

The product mode contract is narrowed to two user-facing modes:

- `dialogue`: the default low-latency spoken workmate/companion chain.
- `professional`: the explicit evidence-first V21 adapter chain.

Older labels such as `workmate`, `companion`, `co_creation`, and `roleplay`
remain compatibility aliases that normalize to `dialogue` at product-contract
surfaces. They must not become separate launch targets.

## Scope

- Add a product-mode normalization contract and tests.
- Update Gateway voice-mode catalog language to the dialogue/professional
  product split.
- Update Xiaozhi streaming provider readiness so the configured dialogue chain
  is reported as a dialogue-chain gate, not a generic provider-shape gate.
- Keep Doubao realtime credentials out of repo files, docs, reports, and
  command lines. Only env var names may appear.
- Keep `professional` on the V21/evidence path.

## Non-Scope

- No provider execution in this transition.
- No V21 execution.
- No Gateway start/stop.
- No `/v1/xiaozhi/say`.
- No firmware build, flash, NVS, serial, hardware action, or audio playback.
- No claim of PRD or physical acceptance.

## Acceptance

- Red tests first prove `workmate`/`companion` normalize to `dialogue` and
  product surfaces expose only `dialogue` plus `professional`.
- Readiness report for
  `sherpa_onnx_streaming + stepfun + doubao_tts_realtime` with env names
  present returns a passed dialogue-chain static gate while keeping
  `prd_accepted=false`.
- Reports do not store credential values, model values, voice values, URLs,
  transcripts, local paths, or audio payloads.
- `git diff --check`, focused tests, `make verify`, `make preflight`, and
  `make doctor` pass or any remaining blockers are recorded honestly.

## Rollback

Revert this plan, the product-mode normalization helper/tests, the Gateway
catalog wording, and readiness-report field additions. Existing Xiaozhi
transport, provider seams, V21 professional path, and hardware state remain
unchanged.
