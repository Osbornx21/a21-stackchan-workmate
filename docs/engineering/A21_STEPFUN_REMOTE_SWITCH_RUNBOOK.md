# A21 StepFun Remote Switch Runbook

Status: operational runbook for the public Gateway StepFun switch.
Scope: ECS-side provider selection and redacted post-switch evidence only.

This runbook moves the public A21 Gateway cascade LLM from DeepSeek fallback to
the StepFun launch profile. It does not authorize provider execution from the
Mac, firmware changes, hardware flashes, remote secret writes by an agent, raw
transcript capture, prompt capture, audio payload capture, proxy-value capture,
or full URL capture.

## Preconditions

- Run these steps on the approved ECS host as an operator with root access.
- Keep all provider secret values out of shell history, stdout, reports, docs,
  chat, and screenshots.
- Keep the provider secret file root-owned and readable only by root:
  `/etc/a21/secrets/provider.env`.
- Use only A21 namespaced env vars and profile IDs.
- Preserve A21 direct-connect proxy policy for localhost, LAN, `.local`,
  StackChan, and V21 adapter traffic.

## Provider Env Contract

`/etc/a21/secrets/provider.env` must contain env names for the active cascade
chain. Record only env names and profile IDs in reports:

| Surface | Required profile ID or env name |
| --- | --- |
| Chain mode | `cascade` |
| ASR profile | `dashscope_qwen_asr_realtime` |
| LLM profile | `stepfun` |
| TTS profile | `dashscope_qwen_tts_realtime` |
| StepFun key env | `A21_LAB_STEPFUN_API_KEY` |
| StepFun model env | `A21_STEPFUN_MODEL` |
| DashScope key env | `A21_DASHSCOPE_API_KEY` |

The file may include additional A21 env names needed by the existing service
unit, but it must not include X21 names or non-A21 provider profile names.

## Switch

1. On ECS, inspect permissions without printing secret values:

```bash
sudo ls -l /etc/a21/secrets/provider.env
sudo stat -c '%U %G %a %n' /etc/a21/secrets/provider.env
```

Expected: owner `root`, group `root`, mode `600` or stricter.

2. Edit the root-only provider env file on ECS. Do not echo values:

```bash
sudoedit /etc/a21/secrets/provider.env
```

Set the cascade selector env names to the required profile IDs and ensure the
StepFun and DashScope credential env names are present with operator-provided
values. Do not record those values.

3. Restart the Gateway service:

```bash
sudo systemctl restart a21-gateway
sudo systemctl is-active a21-gateway
```

4. Query a fresh voice-chain profile snapshot without recording an origin or
full URL. `A21_GATEWAY_ORIGIN` must be set out of band for the shell only:

```bash
a21_get() { curl -fsS --noproxy '*' "${A21_GATEWAY_ORIGIN:?set A21_GATEWAY_ORIGIN}$1"; }
a21_post() { curl -fsS --noproxy '*' -H 'Content-Type: application/json' -d "$2" "${A21_GATEWAY_ORIGIN:?set A21_GATEWAY_ORIGIN}$1"; }

a21_get /v1/voice-chain-profiles |
  jq '{schema_version,selected_voice_chain_mode,selected_asr_profile,selected_llm_profile,selected_tts_profile,selected_voice_clone_profile,findings}'
```

Expected: `selected_llm_profile` is `stepfun`, `selected_voice_chain_mode` is
`cascade`, and `findings` does not include `stepfun_not_selected`.

5. If the restart came up on DeepSeek fallback, re-apply the in-memory selector
through the Gateway contract, then query it again:

```bash
a21_post /v1/voice-chain-profiles '{"voice_chain_mode":"cascade","asr_profile":"dashscope_qwen_asr_realtime","llm_profile":"stepfun","voice_clone_profile":"a21_voice_default_dashscope"}' |
  jq '{schema_version,selected_voice_chain_mode,selected_asr_profile,selected_llm_profile,selected_tts_profile,selected_voice_clone_profile,findings}'

a21_get /v1/voice-chain-profiles |
  jq '{schema_version,selected_voice_chain_mode,selected_asr_profile,selected_llm_profile,selected_tts_profile,selected_voice_clone_profile,findings}'
```

## Post-Switch Evidence

Run evidence only from approved ECS/operator contexts. Do not run provider
execute from the Mac.

1. Static provider readiness rerun:

```bash
go run ./cmd/a21 xiaozhi-streaming-provider-readiness --output-dir reports
```

Expected report shape:

- `selection.llm_profile=stepfun`
- `llm.selection_role=launch_selected`
- `llm.required_env` lists StepFun env names
- `llm.present_env` lists present env names
- no key values, model values, raw prompts, raw transcripts, full URLs, proxy
  values, local secret paths, or audio payloads

2. Host bench rerun after the fresh selector snapshot:

```bash
go run ./cmd/a21 xiaozhi-voice-bench --gateway-url "$A21_GATEWAY_ORIGIN" --repeat 3 --require-product-chain --output-dir reports
```

Expected: host bench remains host or candidate evidence only. It may show
`llm_profile=stepfun`, but it must not claim physical PRD acceptance.

3. Physical evidence rerun for the target StackChan turn:

```bash
go run ./cmd/a21 xiaozhi-physical-evidence --gateway-url "$A21_GATEWAY_ORIGIN" --device-id "$A21_DEVICE_ID" --trace-id "$A21_TRACE_ID" --session-id "$A21_SESSION_ID" --output-dir reports
go run ./cmd/a21 stackchan-accept --check xiaozhi-half-duplex --gateway-url "$A21_GATEWAY_ORIGIN" --device-id "$A21_DEVICE_ID" --trace-id "$A21_TRACE_ID" --session-id "$A21_SESSION_ID" --output-dir reports
```

Expected: physical evidence is matched to the requested `device_id`,
`trace_id`, and `session_id`. Gateway downlink alone remains candidate evidence
until device playback acknowledgement, playback stop completion, or trusted
operator/instrument audible observation is present.

4. Readiness reruns:

```bash
go run ./cmd/a21 product-readiness --gateway-url "$A21_GATEWAY_ORIGIN" --device-id "$A21_DEVICE_ID" --use-latest-reports --output-dir reports
go run ./cmd/a21 server-side-readiness-bundle --gateway-url "$A21_GATEWAY_ORIGIN" --device-id "$A21_DEVICE_ID" --use-latest-reports --output-dir reports
```

Expected: reports expose env names and profile IDs only. If StepFun is still
not selected, the blocker must remain explicit as `stepfun_not_selected` rather
than being collapsed into a generic provider fallback.

## Rollback

Rollback is also ECS-only:

1. Restore the previous root-only `/etc/a21/secrets/provider.env` profile IDs.
2. Restart `a21-gateway`.
3. Query `/v1/voice-chain-profiles`.
4. Record only profile IDs, env names, report basenames, and status/findings.
