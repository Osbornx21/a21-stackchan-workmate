# A21 Latency Budget

## Principle

A21's latency target is perceived conversation quality, not a single backend number. Every measurement must name the network mode, device path, provider path, and whether proxy/TUN was involved.

## Companion Fast Path Targets

| Segment | P50 Target | P95 Target | Notes |
| --- | ---: | ---: | --- |
| device capture to gateway first audio frame | 40 ms | 120 ms | LAN path only |
| VAD start detection | 120 ms | 250 ms | depends on chunk size and noise |
| provider first response/audio | 700 ms | 1300 ms | provider-dependent |
| gateway downlink first frame | 40 ms | 120 ms | LAN path only |
| device playback start after first downlink | 80 ms | 180 ms | buffer dependent |
| total first audible response | 900 ms | 1500 ms | aspirational real-provider goal |
| barge-in stop | 180 ms | 300 ms | user speech to playback stopped |

## Professional Path Targets

| Segment | Target |
| --- | ---: |
| acknowledge professional mode | < 1200 ms |
| STT first partial | < 700 ms |
| V21 first result | < 1500 ms for warm local adapter |
| first spoken evidence/summary block | < 2500 ms |

Professional answers may take longer than companion replies, but they must stream progress and show evidence state instead of going silent.

Current Gateway default: V21 adapter calls have a 3000 ms hard timeout. The adapter request still asks V21 for `max_first_response_ms=1200`, but A21 keeps the outer timeout slightly wider so the fallback can be honest instead of racing transient local adapter overhead. A timeout records `v21.query.timeout`, returns the professional fallback copy, and observes `a21_v21_query_ms`.

## Measurement Requirements

Every latency report should include:

- `trace_id`
- `session_id`
- `device_id`
- mode
- network profile
- proxy profile
- provider name
- capture start/end
- gateway first frame
- VAD start/end
- provider send time
- provider first token/audio
- V21 query latency histogram when mode is professional
- TTS first chunk when applicable
- downlink first frame
- playback start
- interrupt detection and stop time

## Current Mock Benchmark

A21 has a mock-only latency benchmark:

```bash
go run ./cmd/a21 latency-bench --mock --iterations 5
make latency-bench
```

The report currently measures in-process Gateway paths for:

- `mock_turn_ms`
- `professional_turn_ms`
- `barge_in_stop_ms`
- `audio_ws_downlink_ms`
- `audio_ws_barge_in_stop_ms`

`audio_ws_downlink_ms` opens a mock audio WebSocket, sends one `audio.frame`, and measures until the Gateway returns the mock speaking control state plus a full 20 ms / 16 kHz / mono / `pcm_s16le` `audio.playback.chunk`. Each mock chunk carries 640 raw PCM bytes encoded in base64, so the measurement now covers the Gateway downlink envelope path for a real-sized first PCM chunk instead of a tiny placeholder payload. Each series reports `samples`, `p50_ms`, and `p95_ms` using nearest-rank percentiles.

`audio_ws_barge_in_stop_ms` opens the same mock audio WebSocket, establishes an active playback stream with a silent frame, then sends a voiced PCM16 frame that triggers mock VAD `vad.speech.start`. The measurement starts when the voiced frame is written and stops when Gateway returns `interrupted`. This protects the audio-channel interruption contract without claiming hardware microphone, speaker, AEC, or provider-cancel latency.

The Gateway audio ingress path now has a small bounded frame buffer and RMS-based mock VAD transition detector. It records `audio.ingress.buffered`, `vad.speech.start`, and `vad.speech.end` trace markers and exposes ingress/VAD Prometheus metrics. This is a control-point and observability baseline for future tuning; it is not production VAD, echo cancellation, full-duplex validation, or LAN jitter characterization.

When the selected Gateway voice provider exposes an explicit realtime session interface, `/ws/audio` now starts or reuses that session on detected speech, forwards non-silent speech frames, and calls provider commit/create-response on `vad.speech.end`. It records `provider.realtime_session.start`, `provider.audio.append`, and `provider.audio.commit`, and exposes `a21_realtime_audio_uplink_frames_total` plus `a21_realtime_audio_commit_total`. This proves the A21-owned provider-neutral uplink control point; it still does not prove provider output reads, real TTS first audio, AEC, or StackChan hardware full-duplex.

The browser simulator decodes and schedules those mock PCM chunks with WebAudio and stops scheduled sources on interruption, but mock results still do not represent real provider, LAN, microphone, speaker, or StackChan hardware latency. They only protect report shape, Gateway baseline behavior, and the client-side playback/cancellation development surface.

Future real-provider/device benchmarks must add environment fingerprint and store a report artifact before their results count for release decisions.

## Current Firmware Playback Control

The StackChan firmware now has a hardware-free playback state machine. It records the local transition from `speaking` plus `stream_id` to playback started, and it stops plus clears pending playback when the state leaves `speaking`, especially `interrupted`.

The audio WebSocket also accepts mock Gateway `audio.playback.chunk` downlink envelopes and stores them in a bounded firmware playback buffer keyed by `stream_id`. The buffer has a small fixed chunk cap and clears when the render state leaves `speaking`, so future real speaker playback has an explicit cancellation boundary instead of an unbounded queue.

This does not prove speaker output latency yet. It establishes the device-side control points that future real audio playback, mouth sync, and barge-in measurements must instrument as `device_downlink_first_frame_ms`, `device_playback_start_ms`, and `playback_stop_ms`.
