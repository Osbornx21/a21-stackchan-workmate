# 2026-06-04 - Public Gateway Code Sync Before StepFun

Status: active runtime safety plan.
Owner: A21 control tower.
Transition: `T-PUBLIC-GATEWAY-002-CODE-SYNC-BEFORE-STEPFUN`.
Created: 2026-06-04 CST.

## Goal

Deploy the locally verified protocol/readiness/provider-selector fixes to the
public Gateway before any StepFun selector switch. The immediate purpose is to
remove the stale remote binary behavior where ECS-side static
`xiaozhi-streaming-provider-readiness` can report StepFun as ready even when
required StepFun env names are missing.

This transition does not provision provider secrets, execute providers, switch
StepFun, flash firmware, write NVS, or claim launch readiness.

## Current State

- Local branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Local full verification:
  `GOMAXPROCS=2 make verify` passed after the protocol-adaptation round.
- Public Gateway:
  `http://47.103.57.217`.
- Public product entrypoint:
  `ws://47.103.57.217/v1/xiaozhi`.
- ECS services:
  `a21-gateway` active, Caddy active, remote
  `127.0.0.1:21081/healthz` ok.
- Remote provider env file:
  present, root-owned, mode `600`.
- Missing remote env names:
  `A21_LAB_STEPFUN_API_KEY`, `A21_STEPFUN_MODEL`.
- Stale remote behavior observed:
  remote `/opt/a21/bin/a21 xiaozhi-streaming-provider-readiness` reported
  StepFun ready with `gate_status=passed` even though the env-name check showed
  StepFun env names missing.

## Target State

- The current verified repository commit is deployed to `/opt/a21` on ECS.
- Previous remote `/opt/a21` is preserved under `/opt/a21.prev`.
- Remote `go test ./internal/gateway ./internal/app -run
  'VoiceChainProfiles|GatewayProfiles|XiaozhiStreamingProviderReadiness'`
  passes before service swap.
- Remote binary `./bin/a21 xiaozhi-streaming-provider-readiness` blocks
  StepFun when StepFun env names are absent and reports missing env names only.
- Public `80` health, `/v1/devices`, `/v1/voice-chain-profiles`,
  `/v1/gateway-profiles`, and `/xiaozhi/ota/` remain healthy after restart.
- Live selector remains DeepSeek fallback until StepFun env names are
  provisioned and the explicit switch plan resumes.

## Stop Rules

- Do not edit `/etc/a21/secrets/provider.env`.
- Do not print provider secret values, model values, URLs with credentials,
  proxy values, transcripts, prompt text, raw/base64 audio, or local secret
  paths.
- Do not run provider execute.
- Do not POST/PUT `/v1/voice-chain-profiles` to StepFun in this transition.
- Do not flash firmware or write NVS.
- Do not deploy an uncommitted worktree snapshot; deploy a commit that can be
  recovered from the repository.

## Action Plan

1. Commit the current local protocol-adaptation and control-state changes,
   excluding unrelated local noise such as `.DS_Store` and the separate
   governance remediation proposal.

2. Transfer the committed tree to ECS using `git archive HEAD`.

3. On ECS:

   ```bash
   set -e
   export PATH=/usr/local/go/bin:$PATH
   rm -rf /opt/a21.next
   mkdir -p /opt/a21.next
   tar -xf - -C /opt/a21.next
   cd /opt/a21.next
   go test ./internal/gateway ./internal/app -run 'VoiceChainProfiles|GatewayProfiles|XiaozhiStreamingProviderReadiness' -count=1
   go build -o /opt/a21.next/bin/a21 ./cmd/a21
   ```

4. Swap and restart:

   ```bash
   systemctl stop a21-gateway
   rm -rf /opt/a21.prev
   mv /opt/a21 /opt/a21.prev
   mv /opt/a21.next /opt/a21
   systemctl start a21-gateway
   systemctl is-active a21-gateway
   curl -fsS http://127.0.0.1:21081/healthz
   ```

5. Verify remote static StepFun env gate without values:

   ```bash
   cd /opt/a21
   set -a
   . /etc/a21/secrets/provider.env
   set +a
   A21_ASR_PROFILE=cloud \
   A21_ASR_CLOUD_PROFILE=dashscope_qwen_asr_realtime \
   A21_TEXT_STREAM_PROFILE=stepfun \
   A21_TTS_FAST_PROFILE=dashscope_qwen_tts_realtime \
   ./bin/a21 xiaozhi-streaming-provider-readiness --output-dir reports
   ```

   Expected until operator provisions StepFun:

   - command exits non-zero
   - `gate_status=blocked`
   - LLM profile `stepfun`
   - missing env includes `A21_LAB_STEPFUN_API_KEY` and `A21_STEPFUN_MODEL`
   - no credential or model values appear

6. Verify public product entrypoints from the control Mac:

   ```bash
   curl --noproxy '*' -fsS http://47.103.57.217/healthz
   curl --noproxy '*' -fsS http://47.103.57.217/v1/devices
   curl --noproxy '*' -fsS http://47.103.57.217/v1/voice-chain-profiles
   curl --noproxy '*' -fsS http://47.103.57.217/v1/gateway-profiles
   curl --noproxy '*' -fsS http://47.103.57.217/xiaozhi/ota/
   ```

## Acceptance Conditions

- Local commit exists and is the source of the remote deploy.
- Remote focused tests pass before swap.
- Remote binary reports StepFun missing env names honestly.
- Public Gateway health and OTA remain healthy.
- Product device remains registered or its absence is recorded honestly.
- `/v1/voice-chain-profiles` still reports DeepSeek fallback and
  `stepfun_not_selected` until the separate StepFun env provisioning and switch
  transition resumes.

## Failure States

- `remote_build_failed`: remote tests or build fail.
- `gateway_restart_failed`: service does not restart or local health fails.
- `public_gateway_regressed`: public `80` health or OTA fails after swap.
- `remote_readiness_false_positive_persists`: remote static readiness still
  reports StepFun ready when env names are missing.

## Rollback

If the restart or public verification fails:

```bash
systemctl stop a21-gateway
test -d /opt/a21.prev
rm -rf /opt/a21.failed
mv /opt/a21 /opt/a21.failed
mv /opt/a21.prev /opt/a21
systemctl start a21-gateway
systemctl is-active a21-gateway
curl -fsS http://127.0.0.1:21081/healthz
```

Do not alter firmware or provider secrets during rollback.

## Next State

- If accepted:
  `S-PUBLIC-GATEWAY-CODE-SYNCED-STEPFUN-ENV-BLOCKED`.
- If failed and rolled back:
  `S-PUBLIC-GATEWAY-ROLLBACK-REQUIRED-STEPFUN-ENV-BLOCKED`.
