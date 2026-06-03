# A21 Voice Mode Selection

`voice_mode` is the explicit operator/frontend product-mode selection for the
converged launch surface.

Current catalog:

- `roleplay`: available default. Uses A21 local audio front end, role/persona
  prompts, memory hints, voice-clone selection where configured, streaming ASR
  where available, streaming text, streaming TTS, and stock Xiaozhi playback.
- `professional`: available V21 evidence path. It is explicit, evidence-first,
  and rejected by roleplay/dialogue-only endpoints.

Gateway exposes the catalog at `GET /v1/voice-modes` and accepts selection by
`POST /v1/voice-modes` with `{"voice_mode":"roleplay"}` or
`{"voice_mode":"professional"}`. For internal test 3 compatibility,
`{"voice_mode":"dialogue"}` is accepted and normalized to selected
`roleplay`. The device registry includes
`current_voice_mode` so operators can see the active product mode next to the
legacy transport/runtime `current_mode`.

Launch rule: `roleplay` and `professional` are the only user-facing product
modes. Legacy labels such as `dialogue`, `workmate`, `companion`, and
`co_creation` normalize to `roleplay`; privacy, focus, local fallback, and error
remain state/policy fields rather than additional product modes. Professional
work must stay on the V21 adapter path and must not be routed through the
roleplay realtime chain.

## Gateway Profile Is Separate

`gateway_profile` is a separate frontend/operator selector for where the
StackChan product connects:

- `public_wss`: default when `A21_PUBLIC_GATEWAY_URL` is configured. Uses the
  main product public Gateway. Product deployment targets trusted `443` /
  `wss`; IP-only bring-up may temporarily use `http/ws`.
- `mac_local`: selectable local path. Keeps the Mac Gateway path for local
  models and local processing speed. `A21_MAC_LOCAL_GATEWAY_URL` can publish an
  explicit credential-free Mac/local `/v1/xiaozhi` WebSocket URL for frontend
  switching; unset deployments keep the request-host fallback.

Gateway exposes this catalog at `GET /v1/gateway-profiles` and accepts
selection by `POST /v1/gateway-profiles` with
`{"gateway_profile":"mac_local"}` or `{"gateway_profile":"public_wss"}`.
`public_wss` is rejected until the public URL is configured. This selector
must not add a third voice mode and must not change the professional/V21
boundary.
