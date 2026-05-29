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

Each series reports `samples`, `p50_ms`, and `p95_ms` using nearest-rank percentiles. Mock results do not represent real provider, LAN, microphone, speaker, or StackChan hardware latency. They only protect report shape and Gateway baseline behavior.

Future real-provider/device benchmarks must add environment fingerprint and store a report artifact before their results count for release decisions.

## Current Firmware Playback Control

The StackChan firmware now has a hardware-free playback state machine. It records the local transition from `speaking` plus `stream_id` to playback started, and it stops plus clears pending playback when the state leaves `speaking`, especially `interrupted`.

This does not prove speaker output latency yet. It establishes the device-side control point that future real audio playback, mouth sync, and barge-in measurements must instrument as `device_playback_start_ms` and `playback_stop_ms`.
