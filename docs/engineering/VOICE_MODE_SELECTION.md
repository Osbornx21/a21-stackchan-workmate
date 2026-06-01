# A21 Voice Mode Selection

`voice_mode` is an explicit operator/frontend selection. It is separate from the
product `mode` values such as `workmate`, `private`, `focus`, and
`professional`.

Current catalog:

- `edge_cloud`: available default. Uses the existing A21 local audio front end
  and explicit provider seams.
- `pure_cloud`: planned spike-only. It can be selected and displayed, but it
  must not silently execute provider, V21, firmware, or hardware paths.

Gateway exposes the catalog at `GET /v1/voice-modes` and accepts selection by
`POST /v1/voice-modes` with `{"voice_mode":"edge_cloud"}` or
`{"voice_mode":"pure_cloud"}`. The device registry includes
`current_voice_mode` so operators can see the active selection next to
`current_mode`.

Launch rule: planned modes may reduce confusion by being visible, but they do
not count as PRD execution evidence. Relevant turn paths must reject planned
modes honestly instead of rerouting.
