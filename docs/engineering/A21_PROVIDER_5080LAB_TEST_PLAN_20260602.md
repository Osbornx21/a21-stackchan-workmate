# A21 Provider 5080lab Test Plan

Status: pinned execution plan  
Date: 2026-06-02  
Owner: A21 control tower  
Baseline branch: `codex/a21-integration-runtime-readiness-20260601`  
Minimum compatible baseline: `1a18efb8b346 feat(provider): add compatibility matrix`

## 0. Decision

This is the single pinned provider closure plan. Do not open another provider
benchmark lane unless this plan is either completed or explicitly superseded.

The current `provider-compat-matrix` is `partial` only because real cloud ASR
and cloud TTS evidence are missing:

```text
coverage.local_asr = true
coverage.cloud_asr = false
coverage.local_llm = true
coverage.cloud_llm = true
coverage.local_tts = true
coverage.cloud_tts = false
missing_capabilities = ["cloud_asr", "cloud_tts"]
```

Therefore the immediate 5080lab target is not more discussion. It is to return
redacted cloud ASR and cloud TTS evidence, plus a fresh selected-provider LLM
smoke bundle, so the matrix can reach `status=ready`.

## 1. Definition Of Done

5080lab provider work is complete only when all of these are true:

- A returned bundle is accepted by:

```bash
go run ./cmd/a21 provider-evidence-import \
  --bundle reports/a21-5080lab-provider-evidence-YYYYMMDD-HHMMSS.tgz \
  --output-dir reports
```

- The imported reports include:

```text
a21-provider-smoke-*.json
a21-provider-audio-smoke-cloud-asr.json
a21-provider-audio-smoke-cloud-tts.json
```

- The local control machine then reports no provider matrix gaps:

```bash
go run ./cmd/a21 provider-compat-matrix --use-latest-reports --output-dir reports
```

Required result:

```json
{
  "status": "ready",
  "coverage": {
    "local_asr": true,
    "cloud_asr": true,
    "local_llm": true,
    "cloud_llm": true,
    "local_tts": true,
    "cloud_tts": true
  },
  "missing_capabilities": []
}
```

- `product-readiness` with the returned selected provider smoke sees
`provider.real_provider_ready=true`. It must still keep physical StackChan and
custom wake-word gates separate.

## 2. 5080lab Run Order

Run on 5080lab or another approved clean mainland lab host. Do not run executed
cloud provider tests on the proxy-affected Mac.

```powershell
cd <A21 repo checkout>
git status --short --branch
git rev-parse --short=12 HEAD
git fetch --all --prune
git checkout codex/a21-integration-runtime-readiness-20260601
git pull --ff-only
git rev-parse --short=12 HEAD
```

Expected current HEAD is `1a18efb8b346` or newer on the same branch.

Disable ambient proxies for A21 direct/local traffic:

```powershell
Remove-Item Env:HTTP_PROXY -ErrorAction SilentlyContinue
Remove-Item Env:HTTPS_PROXY -ErrorAction SilentlyContinue
Remove-Item Env:ALL_PROXY -ErrorAction SilentlyContinue
$env:NO_PROXY="localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16"
$env:no_proxy=$env:NO_PROXY
```

Create an isolated env file on 5080lab. Do not commit or return this file.

```powershell
mkdir .a21-run\5080lab -Force
notepad .a21-run\5080lab\provider.env.ps1
```

Required contents for the selected route-eligible cloud LLM lane:

```powershell
$env:A21_PROVIDER_PRIMARY="deepseek"
$env:A21_LAB_DEEPSEEK_API_KEY="<secret>"
```

Optional compatibility env for non-route-eligible text-stream candidates:

```powershell
$env:A21_LAB_SILICONFLOW_API_KEY="<secret>"
$env:A21_SILICONFLOW_MODEL="<model>"
$env:A21_LAB_STEPFUN_API_KEY="<secret>"
$env:A21_STEPFUN_MODEL="<model>"
$env:A21_DASHSCOPE_API_KEY="<secret>"
$env:A21_DASHSCOPE_MODEL="<model>"
$env:A21_LAB_MOONSHOT_API_KEY="<secret>"
$env:A21_MOONSHOT_MODEL="<model>"
$env:A21_LAB_VOLCENGINE_ARK_API_KEY="<secret>"
$env:A21_VOLCENGINE_ARK_MODEL="<model>"
```

Load env and run the selected route provider smoke:

```powershell
. .\.a21-run\5080lab\provider.env.ps1
mkdir reports\5080lab-provider -Force

go run .\cmd\a21 doctor --output-dir reports\5080lab-provider
go run .\cmd\a21 provider-smoke --provider $env:A21_PROVIDER_PRIMARY --stream --repeat 3 --output-dir reports\5080lab-provider
go run .\cmd\a21 provider-smoke --provider $env:A21_PROVIDER_PRIMARY --execute --stream --repeat 5 --output-dir reports\5080lab-provider
```

Optional but requested for completeness: rerun configured cloud LLM candidates.
Failures are acceptable only when the report is redacted and the failure reason
is visible without leaking keys, provider output, prompts, full URLs, or proxy
values.

```powershell
go run .\cmd\a21 provider-smoke --provider siliconflow --execute --stream --repeat 5 --output-dir reports\5080lab-provider
go run .\cmd\a21 provider-smoke --provider stepfun --execute --stream --repeat 5 --output-dir reports\5080lab-provider
go run .\cmd\a21 provider-smoke --provider bailian_dashscope --execute --stream --repeat 5 --output-dir reports\5080lab-provider
go run .\cmd\a21 provider-smoke --provider moonshot --execute --stream --repeat 5 --output-dir reports\5080lab-provider
go run .\cmd\a21 provider-smoke --provider volcengine_ark --execute --stream --repeat 5 --output-dir reports\5080lab-provider
```

## 3. Cloud ASR/TTS Required Reports

A21 currently ingests cloud ASR/TTS through redacted lab receipts using schema
`a21.provider_audio_smoke.v1`. 5080lab must run the actual vendor/SDK/API tests
on its side, then write only low-information summary fields.

Cloud ASR report:

```json
{
  "schema_version": "a21.provider_audio_smoke.v1",
  "generated_at_ms": 1780368200000,
  "stage": "asr",
  "placement": "cloud",
  "provider": "iflytek",
  "status": "passed",
  "executed": true,
  "configured": true,
  "endpoint_host": "iat-api.xfyun.cn",
  "asr_final_p95_ms": 354,
  "report_path": "a21-provider-audio-smoke-cloud-asr.json"
}
```

Cloud TTS report:

```json
{
  "schema_version": "a21.provider_audio_smoke.v1",
  "generated_at_ms": 1780368200001,
  "stage": "tts",
  "placement": "cloud",
  "provider": "iflytek",
  "status": "passed",
  "executed": true,
  "configured": true,
  "endpoint_host": "tts-api.xfyun.cn",
  "tts_first_audio_p95_ms": 90,
  "report_path": "a21-provider-audio-smoke-cloud-tts.json"
}
```

Write them to:

```powershell
reports\5080lab-provider\a21-provider-audio-smoke-cloud-asr.json
reports\5080lab-provider\a21-provider-audio-smoke-cloud-tts.json
```

The report must not contain transcript text, provider output, raw audio,
base64 audio, prompt text, full URLs, proxy values, local paths, model values,
API keys, Authorization headers, or reasoning.

## 4. Package And Return

Before packaging, 5080lab must prove the matrix is ready inside the lab output
directory:

```powershell
go run .\cmd\a21 provider-compat-matrix --use-latest-reports --reports-dir reports\5080lab-provider --output-dir reports\5080lab-provider
```

Then package:

```powershell
go run .\cmd\a21 provider-evidence-package --input-dir reports\5080lab-provider --output-dir reports
```

Return only:

```text
reports\a21-5080lab-provider-evidence-YYYYMMDD-HHMMSS.tgz
```

Do not return `.env`, `provider.env.ps1`, secrets, screenshots with keys,
raw audio, transcripts, provider outputs, logs with Authorization headers, or
full local path dumps.

## 5. Control Machine Import

After receiving the bundle:

```bash
go run ./cmd/a21 provider-evidence-import \
  --bundle reports/a21-5080lab-provider-evidence-YYYYMMDD-HHMMSS.tgz \
  --output-dir reports

go run ./cmd/a21 provider-compat-matrix --use-latest-reports --output-dir reports

A21_PROVIDER_PRIMARY=deepseek \
A21_LAB_DEEPSEEK_API_KEY=<configured in local secret manager only> \
go run ./cmd/a21 product-readiness --use-latest-reports --output-dir reports
```

Acceptance:

- provider matrix `status=ready`;
- `missing_capabilities=[]`;
- `provider.real_provider_ready=true` for the selected route provider;
- launch readiness still honestly blocked only by physical StackChan and custom
  wake-word gates if those have not been completed.
