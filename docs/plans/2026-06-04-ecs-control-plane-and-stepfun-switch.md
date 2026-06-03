# 2026-06-04 - ECS Control Plane And StepFun Switch

Status: blocked on ECS StepFun env provisioning.
Owner: A21 control tower.
Transition: `T-ECS-STEPFUN-001-CONTROL-PLANE-AND-RUNTIME-SWITCH`.
Created: 2026-06-04 CST.

## Goal

Move the public Gateway from internal test 3 DeepSeek fallback state toward the
launch-policy StepFun state without breaking the working public `/v1/xiaozhi`
entrypoint, leaking secrets, or claiming PRD launch before physical evidence is
collected.

## Current State

- Public Gateway HTTP `80` is reachable again:
  `http://47.103.57.217/healthz` returns service `a21-gateway`, status `ok`.
- Product StackChan `44:1b:f6:e2:6a:60` is online on the public Gateway.
- Current voice-chain state is cascade:
  ASR `dashscope_qwen_asr_realtime`, LLM `deepseek`, TTS
  `dashscope_qwen_tts_realtime`.
- Current selector finding: `stepfun_not_selected`.
- SSH from this Mac succeeds only when an explicit existing local identity is
  used; the default SSH agent has no loaded identities.
- ECS `a21-gateway` and Caddy are active, and remote
  `127.0.0.1:21081/healthz` returns ok.
- The root-owned remote provider env file exists with mode `600`, but
  `A21_LAB_STEPFUN_API_KEY` and `A21_STEPFUN_MODEL` are missing.
- Public `:21081` direct HTTP currently returns `Empty reply from server`;
  product traffic should continue through public `80` unless an ECS diagnosis
  proves otherwise.

## Target State

- ECS control-plane access is recovered or an approved operator runs the
  commands from this plan on the host.
- Remote StepFun env names are verified as present or missing without printing
  key/model values.
- If StepFun is configured, `a21-gateway` is switched to StepFun and restarted
  through the guarded ECS service lane.
- Fresh read-only snapshots prove `/healthz`, `/v1/devices`,
  `/v1/voice-chain-profiles`, `/v1/gateway-profiles`, and `/xiaozhi/ota/`.
- Fresh evidence is collected in order: host bench, physical Xiaozhi evidence,
  product readiness, server-side readiness bundle.
- Launch remains blocked unless physical playback ack/stop_done/trusted
  audible observation is also present.

## Trigger

Use this plan when the control tower is ready to mutate the remote runtime or
when the public Gateway again shows `stepfun_not_selected` after local
protocol/readiness adaptation has passed.

## Execution Result - 2026-06-04 02:24 CST

Completed:

- ECS service health verified.
- Remote provider env file presence, owner, and mode verified.
- StepFun env-name presence checked without printing values.
- Fresh product readiness and server-side readiness reports were written:
  `reports/a21-product-readiness-20260604-022419.json` and
  `reports/a21-server-side-readiness-bundle-20260604-022420.json`.

Blocked:

- `A21_LAB_STEPFUN_API_KEY` is missing on ECS.
- `A21_STEPFUN_MODEL` is missing on ECS.

No runtime switch was attempted. Continue at Action Plan step 2 after an
approved operator provisions those missing env names in the remote root-only
provider env file and restarts/validates `a21-gateway`.

## Stop Rules

- Do not write provider keys, key values, model values, Wi-Fi credentials,
  transcripts, prompts, raw/base64 audio, proxy secrets, or local secret paths
  into repo, reports, logs, traces, shell output, or chat.
- Do not flash firmware or write NVS in this transition.
- Do not use the generic `xiaozhi.bin` product flash lane.
- Do not switch the public runtime to StepFun if the required env names are
  absent.
- Do not run `POST /v1/voice-chain-profiles` as a blind public hot switch if
  the remote provider configuration has not been verified.
- Do not claim `launch_ready=true` or `prd_accepted=true` from server-side or
  host-only evidence.

## Action Plan

1. Recover or confirm ECS control-plane access.

   - Preferred: restore the approved SSH key for
     `ssh root@47.103.57.217`.
   - Fallback: an approved operator runs the same commands on the ECS console.
   - Verify only:

     ```bash
     systemctl is-active a21-gateway
     systemctl is-active caddy
     ss -ltnp | grep -E ':(80|443|21081)\b'
     curl -fsS http://127.0.0.1:21081/healthz
     ```

2. Verify StepFun env-name presence without values.

   ```bash
   sudo sh -c '
   set -eu
   test -f /etc/a21/secrets/provider.env
   chmod 600 /etc/a21/secrets/provider.env
   . /etc/a21/secrets/provider.env
   for name in A21_LAB_STEPFUN_API_KEY A21_STEPFUN_MODEL; do
     if [ -n "${!name:-}" ]; then
       printf "%s=present\n" "$name"
     else
       printf "%s=missing\n" "$name"
     fi
   done
   '
   ```

3. Switch only if StepFun env names are present.

   - Follow `docs/engineering/A21_STEPFUN_REMOTE_SWITCH_RUNBOOK.md`.
   - Keep provider env values out of stdout and reports.
   - Restart `a21-gateway` only through systemd.

4. Collect fresh read-only snapshots from the control Mac using direct routing.

   ```bash
   curl --noproxy '*' -fsS http://47.103.57.217/healthz
   curl --noproxy '*' -fsS http://47.103.57.217/v1/devices
   curl --noproxy '*' -fsS http://47.103.57.217/v1/voice-chain-profiles
   curl --noproxy '*' -fsS http://47.103.57.217/v1/gateway-profiles
   curl --noproxy '*' -fsS http://47.103.57.217/xiaozhi/ota/
   ```

5. Collect post-switch host and physical evidence.

   ```bash
   NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
   go run ./cmd/a21 xiaozhi-voice-bench \
     --gateway-url http://47.103.57.217 \
     --device-id stackchan-virtual-a21-bench-001 \
     --output-dir reports

   NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
   go run ./cmd/a21 stackchan-accept \
     --check xiaozhi-physical-evidence \
     --gateway-url http://47.103.57.217 \
     --device-id 44:1b:f6:e2:6a:60 \
     --output-dir reports

   NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
   go run ./cmd/a21 product-readiness \
     --gateway-url http://47.103.57.217 \
     --device-id 44:1b:f6:e2:6a:60 \
     --use-latest-reports \
     --output-dir reports

   NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
   go run ./cmd/a21 server-side-readiness-bundle \
     --gateway-url http://47.103.57.217 \
     --device-id 44:1b:f6:e2:6a:60 \
     --use-latest-reports \
     --output-dir reports
   ```

## Acceptance Conditions

- `make verify` stays green before deploy/runtime mutation.
- Public `80` health remains `ok` after any restart.
- `/v1/voice-chain-profiles` reports selected LLM `stepfun` and no
  `stepfun_not_selected` finding.
- Product device `44:1b:f6:e2:6a:60` remains online.
- Post-switch host bench completes without provider/key leakage.
- Product readiness/server-side readiness names remaining blockers honestly.
- If playback ack/stop_done/trusted audible evidence is still absent,
  `launch_ready=false` and `prd_accepted=false`.

## Failure States

- `ssh_control_plane_unavailable`: SSH key rejected or ECS console unavailable.
- `stepfun_env_missing`: required StepFun env names are absent.
- `gateway_restart_failed`: systemd restart fails or health does not recover.
- `public_gateway_http_failed`: public `80` health/OTA stops responding.
- `physical_evidence_incomplete`: physical device is offline or evidence lacks
  playback ack/stop_done/trusted audible observation.

## Rollback

If the Gateway becomes unhealthy after a runtime switch or deploy, rollback to
the previous remote `/opt/a21` directory and DeepSeek fallback service config
using the rollback commands recorded in
`docs/handoffs/2026-06-03-a21-internal-test3-master-handoff.md`. Do not alter
firmware or product flash state during this rollback.

## Next State

- If accepted: `S-PUBLIC-GATEWAY-STEPFUN-SELECTED-PHYSICAL-EVIDENCE-PENDING`.
- If blocked by control-plane access:
  `S-PUBLIC-GATEWAY-HEALTHY-STEPFUN-SWITCH-BLOCKED-BY-SSH`.
- If blocked by missing StepFun env:
  `S-PUBLIC-GATEWAY-HEALTHY-STEPFUN-SWITCH-BLOCKED-BY-ENV`.
