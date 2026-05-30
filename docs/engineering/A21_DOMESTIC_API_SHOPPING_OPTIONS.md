# A21 Domestic API Shopping Options

Date: 2026-05-30
Scope: domestic-only API and paid capability selection for A21.
Audience: downstream shopping/procurement/evaluation process.

## Hard Boundary

A21 is for domestic use only in this shopping pass. Do not buy foreign model APIs or foreign AI SaaS for A21.

Excluded from purchasing in this pass:

- OpenAI, Anthropic, Google/Gemini, Azure OpenAI, AWS Bedrock, Groq, Mistral-hosted, Cohere, Voyage, ElevenLabs, Deepgram, AssemblyAI.
- Foreign hosted vector/search SaaS such as Pinecone, Qdrant Cloud, MongoDB Atlas/Voyage, Weaviate Cloud.
- Foreign realtime/voice-agent SaaS such as LiveKit Cloud, Retell, Vapi, Daily, Twilio AI voice.

OpenAI-compatible means only protocol compatibility. It does not mean buying OpenAI.

## A21 Guardrails For Buyers

- Provider keys must live only in A21 Core/Gateway or operator secret storage.
- No provider API key may enter StackChan firmware, PlatformIO build flags, artifact manifests, logs, traces, screenshots, or reports.
- Prefer mainland China endpoints, mainland data residency, Chinese invoices, explicit quota controls, and API-key scoped subaccounts.
- Any provider that cannot clearly say where audio/text is processed should be marked `blocked_pending_data_region`.
- All traffic to StackChan, localhost, LAN, `.local`, and the A21 V21 adapter must stay direct and must not inherit global proxies.
- For paid smoke tests, buy enough quota for repeated short calls, not a production commitment.

## Current P0 A21 Provider Env Map

The current executable provider-smoke registry is deliberately narrow:

| A21 provider | Capability | Required env |
| --- | --- | --- |
| `deepseek` | text LLM / professional reasoning | `A21_LAB_DEEPSEEK_API_KEY`; optional `A21_DEEPSEEK_MODEL`, default `deepseek-v4-flash` |

Candidate providers below remain procurement/research inputs, not current executable P0 provider-smoke profiles:

| A21 provider | Capability | Required env when promoted |
| --- | --- | --- |
| `bailian_dashscope` | Alibaba Bailian/DashScope LLM, ASR/TTS family, OpenAI-compatible text smoke | `A21_DASHSCOPE_API_KEY`, `A21_DASHSCOPE_MODEL` |
| `doubao_realtime` | Doubao realtime speech-to-speech candidate | `A21_DOUBAO_API_KEY`, `A21_DOUBAO_APP_ID`, `A21_DOUBAO_RESOURCE_ID`, `A21_DOUBAO_REALTIME_MODEL` |
| `doubao_tts_realtime` | Doubao realtime TTS / voice output candidate | `A21_DOUBAO_API_KEY`, `A21_DOUBAO_TTS_MODEL`, `A21_DOUBAO_TTS_VOICE` |

Excluded current repo provider:

| A21 provider | Reason |
| --- | --- |
| `openai_realtime` | Foreign model provider; do not buy for this domestic pass. Keep only as code boundary/reference unless explicitly re-approved. |

Optional shared env:

| Env | Purpose |
| --- | --- |
| `A21_PROVIDER_PRIMARY` | selected provider for reports/plans |
| `A21_GATEWAY_VOICE_PROVIDER=selected` | explicit opt-in before Gateway uses selected provider |
| `A21_PROVIDER_PROXY_URL` | explicit cloud-provider egress proxy only; never LAN/StackChan/V21 |
| `A21_V21_ADAPTER_URL` | internal/professional V21 adapter boundary, not V21 internals |

## Purchase Priority Summary

| Priority | Buy / enable | Why |
| --- | --- | --- |
| P0 | Volcengine/Doubao realtime speech-to-speech access | Main domestic companion fast-path candidate for low-latency voice, interruption, character, voice-clone style behavior. |
| P0 | Volcengine/Doubao realtime TTS or speech synthesis access | Best current A21-shaped lane for expressive Chinese voice output and voice-clone experiments. |
| P0 | Alibaba Bailian/DashScope API bundle | Broad domestic fallback for Qwen text, ASR, TTS, Qwen-Omni-Realtime, embeddings, rerank. |
| P0 | DeepSeek API | Cheap/strong domestic reasoning/text baseline and current executable provider smoke path. |
| P0 | V21 adapter endpoint permission | Needed for professional evidence mode; must be adapter boundary only. |
| P1 | StepFun realtime/audio/text API | Promising domestic voice interaction alternative; needs adapter work and data-region confirmation. |
| P1 | Tencent Cloud Hunyuan + ASR/TTS + VectorDB | Enterprise domestic fallback, especially if Tencent account/procurement path is easier. |
| P1 | Zhipu GLM API | Domestic text, embedding, rerank, STT/TTS, knowledge API alternative. |
| P2 | Baidu Qianfan + speech + VectorDB | Enterprise fallback; useful if Baidu cloud is already available. |
| P2 | iFLYTEK speech stack | Mature ASR/TTS/voice-clone vendor; good for speech-quality bakeoff. |
| P2 | MiniMax speech / voice clone | Strong TTS/voice-clone candidate; confirm mainland data handling before buying. |
| P2 | Huawei Cloud SIS / KooSearch / MaaS | Enterprise/government procurement fallback; useful if Huawei Cloud is mandated. |
| P2 | SiliconFlow | Domestic aggregation for text/embedding/rerank experiments; not preferred for core voice path. |

## 1. End-To-End Realtime Speech-To-Speech

This category is the closest to A21's companion fast path: user speech in, model understands/responds, audio out, with low latency and interruption behavior.

| Vendor | Product / capability | API shape | Buy / ask for | A21 fit | Current repo state | Official source |
| --- | --- | --- | --- | --- | --- | --- |
| Volcengine / Doubao | End-to-end realtime voice model / realtime conversation | RTC/OpenAPI oriented `StartVoiceChat`; S2S app id/token; model version config | S2S service, APP ID, access token/API key, realtime quota, model versions O/SC, voice clone permission, concurrency | Best P0 candidate for domestic companion fast path | `doubao_realtime` boundary exists but execution blocked until official shape and smoke are verified | [Volcengine end-to-end realtime voice](https://www.volcengine.com/docs/6348/1902994) |
| Alibaba Bailian/DashScope | Qwen-Omni-Realtime | WebSocket and WebRTC; `DASHSCOPE_API_KEY`; mainland Beijing endpoint available | Qwen-Omni-Realtime model permission, realtime quota, WebSocket/WebRTC access, Beijing region key | Strong P0/P1 alternative; good for comparison with Doubao | Adapter not yet implemented under `bailian_dashscope` realtime | [Qwen-Omni-Realtime](https://help.aliyun.com/zh/model-studio/realtime) |
| StepFun | Realtime voice interaction / Step-Audio | WebSocket Realtime API; built-in ASR/VAD/context; API key | realtime model permission, StepAudio model access, quota, data-region confirmation | P1 challenger for expressive companion behavior | No adapter yet | [StepFun realtime voice](https://platform.stepfun.com/docs/llm/realtime) |
| Zhipu AI | WSS audio/video call / realtime API | WSS realtime entry in BigModel docs | realtime permission, model list, audio in/out formats, quota, mainland processing confirmation | P1/P2 candidate if GLM ecosystem is already used | No adapter yet | [Zhipu API overview](https://docs.bigmodel.cn/cn/api/introduction) |
| Volcengine Edge AI Gateway | Voice chat agent / realtime API | `wss://ai-gateway.vei.volces.com/v1/realtime` for some agent/model forms | model-bound gateway API key, agent model permission, quota | Useful fallback if pure S2S product contract is too heavy | Not directly mapped; adjacent to Doubao realtime/TTS boundary | [Volcengine voice chatbot realtime](https://www.volcengine.com/docs/6893/1389041) |

Selection questions:

- Does it support server-side WebSocket/RTC from A21 Gateway, not browser-only?
- Can A21 control VAD/turn taking, barge-in, cancel/truncate, and first-audio timing?
- Does it return transcript/subtitle events for professional-mode observability?
- What audio codecs/sample rates are supported: PCM16 16 kHz, Opus, Ogg Opus?
- Does voice clone require separate review, consent, or offline training?
- Are prompts, audio, and generated voice retained or used for training?

## 2. Realtime TTS, Expressive Voice, Voice Clone

This category supports A21 speaking, professional-mode readout, voice identity, and emotion. It does not replace full speech-to-speech.

| Vendor | Product / capability | API shape | Buy / ask for | A21 fit | Current repo state | Official source |
| --- | --- | --- | --- | --- | --- | --- |
| Volcengine / Doubao | Doubao realtime TTS | `wss://ai-gateway.vei.volces.com/v1/realtime?model=...`; bearer API key | TTS model, voice ids, voice clone permission, PCM/Opus output, concurrency | P0 voice-output lane | `doubao_tts_realtime` provider wrapper exists; real network smoke not enabled yet | [Volcengine realtime TTS](https://www.volcengine.com/docs/6893/1527770) |
| Alibaba Bailian/DashScope | CosyVoice WebSocket | `wss://dashscope.aliyuncs.com/api-ws/v1/inference`; bearer key | CosyVoice v2/v3 model access, voice clone/zero-shot features, Beijing region key | P0/P1 TTS bakeoff, especially if buying DashScope bundle anyway | No dedicated adapter yet | [CosyVoice WebSocket](https://help.aliyun.com/zh/model-studio/developer-reference/cosyvoice-websocket-api) |
| Alibaba Bailian/DashScope | Qwen-TTS realtime | WebSocket realtime TTS | Qwen-TTS model permission, voices, streaming output formats | P1 TTS alternative to CosyVoice | No adapter yet | [Qwen-TTS realtime](https://help.aliyun.com/zh/model-studio/interactive-process-of-qwen-tts-realtime-synthesis) |
| Tencent Cloud | Tencent TTS / streaming TTS / voice cloning | HTTP/SSE/API forms depending product | TTS package, streaming permission, premium voices, voice clone/data deletion terms | P1/P2 enterprise fallback | No adapter yet | [Tencent TTS product](https://cloud.tencent.cn/product/tts), [TRTC TTS](https://cloud.tencent.com/document/product/647/131300) |
| iFLYTEK | TTS, long text TTS, one-sentence voice clone | WebSocket/HTTP depending service | TTS app id/key/secret, RT/long text, voice clone permission, IP whitelist | P1/P2 mature speech vendor bakeoff | No adapter yet | [iFLYTEK realtime ASR](https://www.xfyun.cn/doc/asr/rtasr/API.html), [iFLYTEK voice clone](https://www.xfyun.cn/doc/spark/reproduction.html) |
| MiniMax | Speech TTS / Voice Clone | HTTP API; voice clone endpoint | TTS quota, voice clone quota, voice retention rules, mainland processing confirmation | P1/P2 high-quality expressive TTS candidate; confirm domestic data handling | No adapter yet | [MiniMax voice clone](https://platform.minimax.io/docs/api-reference/voice-cloning-clone) |
| StepFun | StepAudio 2.5 TTS | `https://api.stepfun.com/v1/audio/speech`; API key | TTS model access, zero-shot voice clone, quota, emotion/control features | P1 expressive voice candidate | No adapter yet | [StepAudio 2.5 TTS](https://platform.stepfun.com/docs/zh/guides/models/stepaudio-2.5-tts) |
| Baidu AI Cloud | Speech synthesis | HTTP/WebSocket options depending product | TTS quota, voice list, streaming support, mainland region | P2 fallback | No adapter yet | [Baidu speech docs entry](https://cloud.baidu.com/article/4021959) |
| Huawei Cloud | SIS realtime TTS | WebSocket `rtts` API | SIS TTS quota, project id, IAM/subaccount, region | P2 enterprise fallback | No adapter yet | [Huawei SIS API overview](https://support.huaweicloud.com/api-sis/sis_03_0005.html) |

Selection questions:

- First audio latency under Shanghai office network.
- Naturalness in short workplace replies, not only long narration.
- Barge-in/cancel behavior and whether unfinished audio can be truncated.
- Output format compatible with StackChan path: PCM16 16 kHz mono or convertible Opus/PCM.
- Voice clone consent, retention, deletion, audit trail, and commercial-use terms.

## 3. Realtime ASR / Streaming Speech Recognition

ASR is needed for the cascaded professional path, visible transcript, V21 query confirmation, and fallback when S2S is opaque.

| Vendor | Product / capability | API shape | Buy / ask for | A21 fit | Official source |
| --- | --- | --- | --- | --- | --- |
| Alibaba Bailian/DashScope | Qwen-ASR-Realtime | WebSocket realtime; VAD/manual modes | model access, Beijing key, hotwords, punctuation, timestamps | P0/P1 because it pairs with DashScope bundle | [Qwen-ASR realtime](https://help.aliyun.com/zh/model-studio/qwen-asr-realtime-interaction-process) |
| Alibaba Bailian/DashScope | Paraformer realtime | `wss://dashscope.aliyuncs.com/api-ws/v1/inference` | Paraformer model access, vocabulary/hotwords, 16 kHz PCM | P1 mature ASR baseline | [Paraformer realtime WebSocket](https://help.aliyun.com/zh/model-studio/websocket-for-paraformer-real-time-service) |
| Alibaba Cloud Intelligent Speech | Real-time speech transcription | WebSocket speech interaction product | project config, region, private network option if on Alibaba ECS | P1 if not using Bailian ASR | [Alibaba speech WebSocket](https://help.aliyun.com/zh/isi/developer-reference/websocket) |
| Tencent Cloud | Realtime ASR WebSocket | WebSocket | ASR package, SecretId/SecretKey, vocabulary/hotword, timestamps | P1/P2 enterprise fallback | [Tencent realtime ASR](https://cloud.tencent.com/document/product/586/48982) |
| Volcengine | Realtime ASR / self-deployed ASR through AI Gateway | WebSocket realtime | ASR model, gateway key, audio format, quota | P1 if staying with Volcengine voice stack | [Volcengine realtime ASR](https://www.volcengine.com/docs/6893/1827259) |
| iFLYTEK | Realtime ASR / IAT | WebSocket | app id, api key/secret, IP whitelist, dialects/hotwords | P1/P2 mature speech comparison | [iFLYTEK realtime ASR](https://www.xfyun.cn/doc/asr/rtasr/API.html) |
| Baidu AI Cloud | Realtime speech recognition | WebSocket/REST depending product | API key/secret, access token, hotwords, websocket mode | P2 fallback | [Baidu Qianfan/docs entry](https://cloud.baidu.com/doc/qianfan-docs/s/qm8qxemze) |
| Huawei Cloud | SIS realtime ASR | WebSocket | SIS package, project id, IAM, region | P2 enterprise fallback | [Huawei SIS realtime ASR](https://support.huaweicloud.com/intl/zh-cn/api-sis/sis_03_0030.html) |

Selection questions:

- Partial result latency and final result latency with 20 ms PCM frames.
- Hotword support for cockpit/product/team vocabulary.
- Timestamps, confidence, punctuation, and sentence-end behavior.
- Audio privacy and retention.
- How billing rounds audio duration.

## 4. Text LLM / Reasoning / Professional Summarization

This category supports A21's non-realtime brain, professional summarization, fallback dialogue, router decisions, and smoke tests.

| Vendor | Product / capability | API shape | Buy / ask for | A21 fit | Current repo state | Official source |
| --- | --- | --- | --- | --- | --- | --- |
| DeepSeek | DeepSeek API | OpenAI/Anthropic-compatible; `https://api.deepseek.com`; models `deepseek-v4-flash`, `deepseek-v4-pro` | API key, prepaid balance, model access, rate limit | P0 text reasoning and current executable smoke | `deepseek` smoke execution exists | [DeepSeek quick start](https://api-docs.deepseek.com/), [pricing/models](https://api-docs.deepseek.com/quick_start/pricing) |
| Alibaba Bailian/DashScope | Qwen models via Model Studio | OpenAI-compatible `https://dashscope.aliyuncs.com/compatible-mode/v1` | API key, Qwen model access, Beijing region, rate limits | P0 broad domestic text baseline | `bailian_dashscope` smoke execution exists | [DashScope OpenAI-compatible](https://help.aliyun.com/zh/model-studio/compatibility-of-openai-with-dashscope) |
| Volcengine Ark / Doubao | Doubao text models | Ark / OpenAI-compatible style in Ark ecosystem | Ark API key, model endpoints, quota, data retention terms | P1 if voice stack also uses Volcengine | No adapter yet | [Volcengine docs portal](https://www.volcengine.com/docs) |
| Tencent Cloud | Hunyuan / TokenHub | OpenAI-compatible and Tencent API forms | Hunyuan/TokenHub key, model access, rate limits | P1/P2 enterprise fallback | No adapter yet | [Tencent Hunyuan OpenAI-compatible](https://cloud.tencent.com/document/product/1729/111007) |
| Baidu AI Cloud | Qianfan / ERNIE | OpenAI-compatible SDK usage supported | API key, Qianfan models, quota, region | P2 enterprise fallback | No adapter yet | [Baidu Qianfan quick start](https://cloud.baidu.com/doc/qianfan-docs/s/qm8qxemze) |
| Zhipu AI | GLM models | `https://open.bigmodel.cn/api/paas/v4`; bearer key | API key, GLM model access, embedding/rerank if bundled | P1/P2 text plus retrieval alternative | No adapter yet | [Zhipu API overview](https://docs.bigmodel.cn/cn/api/introduction) |
| Moonshot / Kimi | Kimi API | OpenAI-compatible `https://api.moonshot.cn/v1` | API key, Kimi model access, long-context quota | P2 long-context text candidate | No adapter yet | [Kimi API quick start](https://platform.kimi.com/docs/guide/start-using-kimi-api) |
| StepFun | Step text models / Step Plan | OpenAI-compatible `https://api.stepfun.com/v1`; Step Plan path for subscribed plan | API key or Step Plan, text models, router, quota | P1 if buying Step voice too | No adapter yet | [StepFun OpenAI migration](https://platform.stepfun.com/docs/guide/openai), [Step Plan](https://platform.stepfun.com/docs/zh/stepplan/overview) |
| MiniMax | MiniMax text models | MiniMax platform API | API key, text + speech bundle if useful | P2 if voice clone performs well | No adapter yet | [MiniMax platform docs](https://platform.minimax.io/) |
| SiliconFlow | Aggregated domestic/open models | OpenAI-compatible style; also embedding/rerank | API key, model list, rate limits, data-region confirmation | P2 experiment aggregator, not core voice | No adapter yet | [SiliconFlow docs](https://docs.siliconflow.com/) |
| Huawei Cloud | ModelArts MaaS / Pangu | Huawei MaaS API | MaaS permission, models, enterprise SLA | P2 if Huawei procurement is required | No adapter yet | [Huawei MaaS docs](https://support.huaweicloud.com/usermanual-maas-modelarts/) |

Selection questions:

- Chinese workplace tone quality and professional evidence summarization.
- Streaming first token latency from Shanghai office.
- Function/tool calling behavior.
- JSON stability for router/provider decisions.
- Context window and price for long meeting/history inputs.
- Whether API responses include stable usage fields for cost accounting.

## 5. Embedding, Rerank, Vector DB, Knowledge Store

A21 should not silently become V21. Use this category only for future A21-owned memory/knowledge, or for side-by-side evaluation. Professional evidence remains through the V21 adapter unless an ADR changes it.

| Vendor | Product / capability | Buy / ask for | A21 fit | Official source |
| --- | --- | --- | --- | --- |
| Alibaba Bailian/DashScope | Text/image embedding and `qwen3-rerank` | embedding model, rerank model, quota, batch support | P0/P1 if building A21 knowledge service; pairs with DashScope bundle | [Alibaba embedding and rerank](https://help.aliyun.com/zh/model-studio/embedding-and-rerank/) |
| Volcengine | VikingDB + embedding | VikingDB instance, embedding model, dense/sparse support, RAG/knowledge/memory products | P1 if buying Volcengine voice stack | [VikingDB embedding](https://www.volcengine.com/docs/84313/2173286), [VikingDB docs](https://www.volcengine.com/docs/6459/1163945) |
| Tencent Cloud | VectorDB | VectorDB instance, AI suite, hybrid retrieval/rerank support, VPC/private access | P1/P2 enterprise fallback | [Tencent VectorDB](https://cloud.tencent.com.cn/product/vdb) |
| Baidu AI Cloud | VectorDB / ElasticsearchBES + Qianfan embeddings | VectorDB/BES instance, Qianfan embeddings, RAG example support | P2 enterprise fallback | [Baidu VectorDB docs](https://bce-cdn.bj.bcebos.com/p3m/pdf/bce-doc/online/VDB/VDB.pdf) |
| Zhipu AI | embedding/rerank/knowledge APIs | embedding, rerank, knowledge API, document parsing if needed | P1/P2 compact vendor option | [Zhipu API overview](https://docs.bigmodel.cn/cn/api/introduction) |
| Huawei Cloud | KooSearch / embedding/rerank services | KooSearch or AI search package, embedding/rerank model access | P2 if Huawei cloud mandated | [Huawei KooSearch product docs](https://support.huaweicloud.com/intl/zh-cn/productdesc-koosearch/) |
| SiliconFlow | Embedding and rerank APIs | BGE/Qwen embedding/rerank models, rate limits, data-region confirmation | P2 cheap experiment path | [SiliconFlow rerank](https://docs.siliconflow.com/en/api-reference/rerank/create-rerank) |
| Self-hosted open source | Milvus / pgvector / Elasticsearch / local BGE reranker | compute/storage only; no external model API if self-hosted | Good for privacy and repeatable eval; not a paid model API | Use only if ops capacity exists |

Selection questions:

- Hybrid retrieval support: dense + sparse + BM25 + metadata filters.
- Rerank max documents, max tokens, latency, and price.
- Document deletion and data retention.
- VPC/private access and audit logs.
- Whether the service can store cockpit/company docs under the required compliance boundary.

## 6. Network, Cloud, Domain, Observability

These are not model APIs, but A21 will need them for deployment and verification.

| Category | Domestic options | Buy / enable | A21 notes |
| --- | --- | --- | --- |
| Domain / DNS / SSL | Alibaba Cloud, Tencent Cloud, Volcengine, Huawei Cloud | domain, ICP path if needed, DNS, TLS cert | Existing A21 docs mention Alibaba Cloud domain resources; keep `a21` namespace. |
| Small cloud host / relay | Alibaba ECS, Tencent CVM, Volcengine ECS, Huawei ECS | 1-2 small mainland hosts for relay/remote smoke, security groups, logs | Do not expose StackChan or firmware endpoints casually. |
| Observability logs/traces | Alibaba SLS/ARMS/Prometheus, Tencent CLS/APM/Prometheus, Volcengine Observe, Huawei AOM/LTS | log storage, metrics, trace backend, alerting | Current repo has Prometheus metrics but no durable OTel backend yet. |
| Secret management | Alibaba KMS/Secrets Manager, Tencent KMS/SSM, Volcengine KMS, Huawei KMS | provider API key storage, rotation, scoped subaccounts | Required before shared office/remote operation. |
| Explicit provider egress | domestic proxy/VPN or cloud NAT controlled by user | stable egress, no secret leakage, direct LAN bypass | Only for cloud-provider calls; never for localhost/LAN/StackChan/V21-local. |

## 7. Validation And Test Budget To Buy

For each shortlisted provider, procurement should reserve a small explicit test budget before production purchase.

| Test | Needs paid capability? | Notes |
| --- | --- | --- |
| `provider-smoke --execute deepseek` | yes, tiny | Current repo supports real tiny Chat Completions smoke. |
| Bailian/DashScope text smoke | yes, future | Procurement candidate only; not in the current P0 executable smoke registry. |
| Doubao realtime S2S credentialed smoke | yes, future | Not implemented yet; buy only enough for short fixture and cancellation/latency tests. |
| Doubao realtime TTS credentialed smoke | yes, future | Needs fixture, timeout, output redaction, first-audio metrics. |
| DashScope Qwen-Omni realtime smoke | yes, future | Needs A21 adapter and fixture. |
| ASR bakeoff | yes | Use the same labelled Shanghai office audio fixtures across vendors. |
| TTS/voice clone bakeoff | yes | Use fixed A21 workmate lines; include consent and deletion proof for cloned voices. |
| V21 adapter smoke | internal | Needs adapter endpoint permission, not external model purchase. |
| LAN/StackChan physical acceptance | no model quota by default | Needs device, office network, serial port, audio/screen/touch/servo/RGB evidence. |

Minimum smoke bundle:

- 100-500 short text calls for each text provider under evaluation.
- 30-60 minutes of realtime ASR per ASR provider.
- 30-60 minutes of realtime TTS/S2S per voice provider.
- 3-5 voice clone attempts per TTS vendor if clone quality is in scope.
- Enough quota to repeat tests in home network and Shanghai office network.

## 8. Suggested Domestic Shortlist For The Next Shopping Process

Start with this smaller set unless procurement already has preferred vendor contracts:

1. Volcengine/Doubao: realtime S2S + realtime TTS + optional Ark text + optional VikingDB.
2. Alibaba Bailian/DashScope: Qwen text + Qwen-Omni-Realtime + Qwen/Paraformer ASR + CosyVoice/Qwen-TTS + embedding/rerank.
3. DeepSeek: text reasoning baseline and cheap professional-mode smoke.
4. StepFun: realtime voice + StepAudio TTS + text model if the demo latency/voice quality looks good.
5. Tencent Cloud: Hunyuan + realtime ASR + streaming TTS + VectorDB as enterprise fallback.
6. Zhipu AI: GLM text + embedding/rerank + optional realtime/audio APIs as compact fallback.
7. iFLYTEK: ASR/TTS/voice clone bakeoff if speech quality is not good enough from cloud-model vendors.

## 9. Procurement Fields To Fill

For every candidate, the shopping process should fill:

```yaml
vendor:
product:
capability_category:
mainland_endpoint:
data_processing_region:
data_retention_policy:
training_opt_out:
api_key_scope:
subaccount_supported:
invoice_supported:
trial_quota:
paid_quota:
rate_limit_rpm:
rate_limit_tpm_or_audio_minutes:
concurrency_limit:
audio_formats:
sample_rates:
streaming_protocols:
barge_in_or_cancel_support:
voice_clone_terms:
hotword_support:
timestamps_or_confidence:
usage_fields_for_cost_accounting:
sla_or_support_channel:
official_docs:
quoted_price:
shopping_status: pending
blockers:
```

## Source Index

- A21 repo provider catalog: `internal/providers/catalog.go`
- A21 provider smoke docs: `docs/engineering/PHASE4C_PROVIDER_SMOKE.md`
- A21 V21 adapter docs: `docs/engineering/V21_INTEGRATION.md`
- DeepSeek API: https://api-docs.deepseek.com/
- Alibaba Bailian/DashScope OpenAI-compatible: https://help.aliyun.com/zh/model-studio/compatibility-of-openai-with-dashscope
- Alibaba Qwen-Omni-Realtime: https://help.aliyun.com/zh/model-studio/realtime
- Alibaba Paraformer realtime ASR: https://help.aliyun.com/zh/model-studio/websocket-for-paraformer-real-time-service
- Alibaba Qwen-ASR realtime: https://help.aliyun.com/zh/model-studio/qwen-asr-realtime-interaction-process
- Alibaba CosyVoice WebSocket: https://help.aliyun.com/zh/model-studio/developer-reference/cosyvoice-websocket-api
- Alibaba embedding/rerank: https://help.aliyun.com/zh/model-studio/embedding-and-rerank/
- Volcengine realtime TTS: https://www.volcengine.com/docs/6893/1527770
- Volcengine end-to-end realtime voice: https://www.volcengine.com/docs/6348/1902994
- Volcengine realtime ASR gateway: https://www.volcengine.com/docs/6893/1827259
- Volcengine VikingDB embedding: https://www.volcengine.com/docs/84313/2173286
- Tencent Hunyuan OpenAI-compatible: https://cloud.tencent.com/document/product/1729/111007
- Tencent realtime ASR: https://cloud.tencent.com/document/product/586/48982
- Tencent TTS: https://cloud.tencent.cn/product/tts
- Tencent VectorDB: https://cloud.tencent.com.cn/product/vdb
- Baidu Qianfan quick start: https://cloud.baidu.com/doc/qianfan-docs/s/qm8qxemze
- Baidu VectorDB docs: https://bce-cdn.bj.bcebos.com/p3m/pdf/bce-doc/online/VDB/VDB.pdf
- iFLYTEK realtime ASR: https://www.xfyun.cn/doc/asr/rtasr/API.html
- iFLYTEK voice clone: https://www.xfyun.cn/doc/spark/reproduction.html
- Zhipu AI API overview: https://docs.bigmodel.cn/cn/api/introduction
- Kimi API quick start: https://platform.kimi.com/docs/guide/start-using-kimi-api
- MiniMax voice clone: https://platform.minimax.io/docs/api-reference/voice-cloning-clone
- StepFun OpenAI migration: https://platform.stepfun.com/docs/guide/openai
- StepFun realtime voice: https://platform.stepfun.com/docs/llm/realtime
- StepAudio 2.5 TTS: https://platform.stepfun.com/docs/zh/guides/models/stepaudio-2.5-tts
- SiliconFlow rerank: https://docs.siliconflow.com/en/api-reference/rerank/create-rerank
- Huawei SIS product: https://www.huaweicloud.com/product/sis.html
- Huawei SIS API overview: https://support.huaweicloud.com/api-sis/sis_03_0005.html
- Huawei KooSearch docs: https://support.huaweicloud.com/intl/zh-cn/productdesc-koosearch/
