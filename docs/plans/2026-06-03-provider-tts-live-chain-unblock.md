# T-PROVIDER-002b Iflytek/Real-TTS Live Chain Unblock Plan

Status: active plan
Date: 2026-06-03
Owner: A21 control tower
Transition: `T-PROVIDER-002b-IFLYTEK-REAL-TTS-LIVE-CHAIN-UNBLOCK`

## Background And Problem Definition

`T-PROVIDER-002` added hot-pluggable StepFun text streaming and
`iflytek_tts`, while preserving `voice_clone_cli`. Focused tests and
`make verify` passed. Live Mac direct evidence is mixed: StepFun
`provider-smoke --execute --stream` passed, but Mac direct Iflytek WebSocket
TTS failed at dial and StepFun `local-voice-loopback` hit network timeouts.

The problem is now not "does A21 have a TTS seam"; it does. The problem is
getting a real, low-latency, non-secret-leaking TTS egress path that can produce
playable WAV/PCM for physical StackChan acceptance.

## Current System State

- Current code baseline: commit `b5a405e`.
- `iflytek_tts` is exposed through local TTS smoke, loopback, StackChan
  playback, fast companion turn, and Gateway `A21_TTS_FAST_PROFILE`.
- `reports/provider-tts-candidate/a21-provider-smoke-20260603-010652-957877000.json`
  passed for StepFun direct text stream.
- `reports/provider-tts-candidate/a21-local-tts-smoke-20260603-010638.json`
  failed for Iflytek Mac direct WebSocket with finding
  `iflytek_tts_websocket_dial_failed`.
- 5080 outbox report contains plaintext credentials. Values must not enter
  repo, reports, prompts, handoff logs, or shell snippets.

## Target State

`S-PROVIDER-TTS-REAL-DIALOGUE-CANDIDATE-RUNNING`

- A real TTS source produces a redacted passed report and a playable WAV/PCM
  asset.
- StepFun plus the real TTS source runs in `local-voice-loopback` or equivalent
  host-side chain without leaking text/provider output/secrets.
- Physical StackChan receives that audio through the stock Xiaozhi playback
  path and the operator can judge quality.

## Non-Goals

- Do not modify firmware, NVS, wake word, V21, or accepted 3x Gateway gain.
- Do not change global macOS/Codex proxy settings.
- Do not commit provider credentials or create tracked secret files.
- Do not promote StepFun to product route-eligible without a separate evidence
  promotion.
- Do not remove `voice_clone_cli`.

## Impact Scope

- 5080/Alibaba relay or explicit WebSocket provider-egress adapter.
- `reports/provider-tts-candidate/` redacted live evidence.
- Optional docs update in `docs/engineering/NETWORK.md` if an explicit
  WebSocket proxy/relay adapter is added.
- State and handoff docs.

## Phased Execution

### Phase 1: Confirm Fastest Egress

Actions:

- Check whether 5080 can synthesize Iflytek TTS from its existing report/script
  without returning secrets.
- If 5080 can run it, prefer 5080 relay over Mac direct.
- If 5080 cannot run it, design the smallest A21 WebSocket provider-egress
  adapter that accepts an explicit `A21_PROVIDER_PROXY_URL` without affecting
  LAN/Gateway/StackChan traffic.

Acceptance:

- Chosen egress path is documented as `5080_relay`, `explicit_ws_proxy`, or
  `blocked`.
- No secret values are printed or written.

### Phase 2: Produce Redacted TTS Evidence

Actions:

- Run TTS synthesis through the chosen egress path.
- Return or generate only basename reports and an audio file suitable for
  StackChan playback.
- Confirm report fields: provider, endpoint host, network/relay mode, first
  audio timing, PCM quality, report basename.

Acceptance:

- TTS report status is `passed`, or failure status has a concrete redacted
  finding.
- Report contains no auth query, API key, text, transcript, provider output,
  raw/base64 audio, full URL, proxy value, or local path.

### Phase 3: Text+TTS Chain

Actions:

- Run StepFun `step-1-8k` plus the accepted TTS path in a host-side loopback or
  equivalent redacted chain.
- If Mac direct StepFun remains unstable, run the chain from 5080/relay and
  import only redacted evidence.

Acceptance:

- Report shows real text stream and real TTS in one chain.
- StepFun remains explicit compatibility candidate unless later promoted.

### Phase 4: Physical Playback

Actions:

- Verify Gateway `21081` health and physical device freshness.
- Play the generated TTS through stock Xiaozhi `/v1/xiaozhi/say` or
  `stackchan-local-tts-playback`.
- Ask operator for listening acceptance.

Acceptance:

- Physical trace records stock TTS lifecycle and Opus chunks.
- Operator accepts or rejects the new voice with a specific reason.

## Rollback

- Keep accepted 3x Gateway gain.
- Use `sherpa_onnx` only as emergency/diagnostic fallback.
- Use `voice_clone_cli` if a configured cloned/persona voice is the selected
  fallback.
- Remove only temporary untracked relay artifacts if they were created.

## Risks

- 5080 report credentials may need rotation outside this repo.
- Direct Mac WebSocket egress is already observed unstable.
- Explicit WebSocket proxy support could accidentally route LAN traffic through
  provider egress if not scoped carefully.

## Human Confirmation Points

- Whether to rotate the exposed 5080 report credentials.
- Whether operator accepts the first successful real TTS playback.

## Worker Execution Task

Worker owns only `T-PROVIDER-002b`.

Allowed:

- Read repo docs/code and current reports.
- Use 5080 SSH/outbox/inbox workflow.
- Create redacted reports under `reports/provider-tts-candidate/`.
- Propose or implement a narrowly scoped WebSocket egress adapter only if 5080
  relay is blocked.

Forbidden:

- Firmware flash/NVS writes.
- Global proxy changes.
- Printing or committing secrets.
- Editing wake-word or half-duplex code.

Return summary format:

- What was attempted.
- Reports produced.
- Whether TTS smoke passed.
- Whether StepFun+TTS chain passed.
- Whether physical playback was attempted.
- Deviations from this plan.
- Remaining blocker and next action.
