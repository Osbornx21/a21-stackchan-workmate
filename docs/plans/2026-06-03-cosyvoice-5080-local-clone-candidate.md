# T-COSYVOICE-5080 Local Clone Candidate Recovery Plan

Status: active plan
Date: 2026-06-03
Owner: A21 control tower
Transition: `T-COSYVOICE-5080-LOCAL-CLONE-CANDIDATE-CHECK`

## Background And Problem Definition

A21 has retained `voice_clone_cli` as the product-facing clone/persona TTS seam.
The immediate contest voice path currently uses the 5080 relay
StepFun+Iflytek candidate because it produced positive physical StackChan
listening feedback. The user remembers CosyVoice can clone voices and asked
5080 to check whether a local CosyVoice path already exists and can be tested.

## Current System State

- 5080 SSH is online through the LAN Windows/PowerShell lane.
- `D:/a21-mainland-latency-lab`, `inbox`, and `outbox` exist.
- CosyVoice source and a `.venv` exist, but the CosyVoice venv currently lacks
  `torch` and `tqdm`.
- CosyVoice classes can be imported from the IndexTTS venv, and CUDA is
  available there.
- No usable `pretrained_models/CosyVoice-*` weights were found.
- IndexTTS2 source, venv, runner, and partial checkpoints exist with
  `torch 2.8.0+cu128`, but inference currently fails because
  `checkpoints/qwen0.6bemo4-merge/` is missing or not loadable.
- F5-TTS and GPT-SoVITS source traces exist, but no usable checkpoint/run path
  was confirmed.

## Target State

`S-COSYVOICE-OR-INDEXTTS2-VOICE-CLONE-CANDIDATE-SMOKE-PASSED`

- A 5080 local clone-capable TTS path generates a short redacted smoke WAV.
- The generated report is suitable for A21 `voice_clone_cli` candidate
  evaluation.
- The current contest default remains StepFun+Iflytek unless operator
  listening acceptance prefers the clone path.

## Non-Goals

- Do not change A21 Gateway defaults or provider route eligibility.
- Do not store raw voice samples, reference text, model prompts, or full remote
  paths in the repo.
- Do not copy 5080 credentials into A21 docs/reports.
- Do not download or install large models without a scoped worker task and a
  redacted source/size summary.
- Do not touch firmware, NVS, V21, or StackChan hardware.

## Impact Scope

- 5080 lab cache and outbox only.
- Optional ignored A21 reports under `reports/provider-tts-candidate/` if a
  redacted smoke artifact is imported later.
- `docs/project_state_machine.md` and `docs/agent_handoff_log.md`.

## Phased Execution

### Phase 1: Locate Existing Local Assets

Actions:

- Search only bounded 5080 lab/workspace/model-cache locations.
- Identify usable CosyVoice, IndexTTS2, F5-TTS, or GPT-SoVITS code, venv, and
  checkpoint paths.

Acceptance:

- Report lists model family, environment status, missing dependency or
  checkpoint, and whether CUDA is available.

### Phase 2: Restore The Fastest Existing Candidate

Actions:

- Prefer existing cached weights over fresh downloads.
- If IndexTTS2 only lacks `qwen0.6bemo4-merge`, restore that checkpoint first.
- If CosyVoice weights are locally cached, prefer ordinary SFT smoke before
  zero-shot clone smoke.

Acceptance:

- A local command can generate a short WAV into 5080 outbox.
- No raw reference audio or text leaves the remote lab.

### Phase 3: Redacted A21 Candidate Smoke

Actions:

- Run `voice_clone_cli` style smoke through the bridge or an equivalent wrapper.
- Return only safe basenames, timing, model family, and aggregate WAV quality.

Acceptance:

- A21 can compare the candidate against the accepted StepFun+Iflytek relay WAV.
- Report contains no credential values, raw/base64 audio, prompt text,
  reference text, full path, proxy value, or provider output.

## Rollback

- Keep StepFun+Iflytek as the current accepted contest voice-chain candidate.
- Leave `voice_clone_cli` configured only when a clone smoke has passed.
- Remove only temporary 5080 work directories created by this transition.

## Risks

- Model downloads may be large and slow.
- Some clone models may require license review before contest or product use.
- Reference voice handling has privacy implications and must stay out of repo
  logs and reports.

## Human Confirmation Points

- Whether to allow 5080 to download missing model weights if no local cache is
  found.
- Which reference voice sample is allowed for clone evaluation.
- Operator listening acceptance after the first generated clone WAV.

## Worker Execution Task

Worker owns only `T-COSYVOICE-5080-LOCAL-CLONE-CANDIDATE-CHECK`.

Allowed:

- Use 5080 LAN SSH and outbox/inbox workflow.
- Search bounded local model/cache locations.
- Run a smoke only if required weights already exist or are explicitly restored
  under this plan.
- Return safe basenames and redacted timing/quality summaries.

Forbidden:

- Firmware flash/NVS writes.
- Global proxy changes.
- Printing or committing secrets.
- Storing raw reference audio, reference text, prompts, or full paths in the
  repo.
- Changing A21 default provider/TTS selection.

Return summary format:

- 5080 connection status.
- Candidate environment and missing assets.
- Smoke command/result.
- Outbox basenames.
- Redaction check result.
- Whether the candidate can be wired to `voice_clone_cli`.
- Next fastest action.
