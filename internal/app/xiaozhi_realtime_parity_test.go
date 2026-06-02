package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestXiaozhiRealtimeParityClassifiesTurnBufferedPhysicalTrace(t *testing.T) {
	server := newXiaozhiRealtimeParityTestServer(t, "turn_buffered")
	defer server.Close()
	dir := t.TempDir()

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"xiaozhi-realtime-parity",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--output-dir", dir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.xiaozhi_realtime_parity.v1"`,
		`"classification": "turn_buffered_xiaozhi_candidate"`,
		`"streaming_asr_before_speech_end": false`,
		`"xiaozhi_realtime_turn_buffered"`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"http://secret", "/Users/", "token", "raw_audio", "transcript"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked %q: %s", forbidden, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-xiaozhi-realtime-parity-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("matches=%v err=%v", matches, err)
	}
}

func TestXiaozhiRealtimeParityClassifiesRealtimeCandidateOrdering(t *testing.T) {
	server := newXiaozhiRealtimeParityTestServer(t, "realtime")
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"xiaozhi-realtime-parity",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--output-dir", "",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	for _, want := range []string{
		`"classification": "xiaozhi_realtime_candidate"`,
		`"streaming_asr_before_speech_end": true`,
		`"provider_before_asr_final": true`,
		`"downlink_before_pipeline_completed": true`,
		`"xiaozhi_realtime_candidate_not_product_accepted"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestXiaozhiRealtimeParityBlocksFakeSayPath(t *testing.T) {
	server := newXiaozhiRealtimeParityTestServer(t, "fake_say")
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"xiaozhi-realtime-parity",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--output-dir", "",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code=%d, want blocked non-zero stdout=%s", code, stdout.String())
	}
	for _, want := range []string{
		`"classification": "blocked"`,
		`"forbidden_fake_path": 1`,
		`"xiaozhi_realtime_fake_path_detected"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestXiaozhiRealtimeParityDoesNotAcceptTransportOnlyTrace(t *testing.T) {
	server := newXiaozhiRealtimeParityTestServer(t, "transport_only")
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"xiaozhi-realtime-parity",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--output-dir", "",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code=%d, want non-zero for transport-only trace stdout=%s", code, stdout.String())
	}
	for _, want := range []string{
		`"classification": "stock_opus_transport_only"`,
		`"opus_frames_decoded": 1`,
		`"xiaozhi_realtime_vad_speech_end_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func newXiaozhiRealtimeParityTestServer(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/devices", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "a21.gateway.devices.v1",
			"service":        "a21-gateway",
			"devices": []map[string]any{{
				"device_id":         "44:1b:f6:e2:6a:60",
				"connection_status": "online",
				"capabilities": map[string]string{
					"xiaozhi_profile":   "stock",
					"xiaozhi_transport": "websocket",
					"xiaozhi_audio":     "opus_16000hz_mono_60ms",
				},
				"last_trace_id":   "a21-trace-44-1b-f6:e2:6a:60",
				"last_session_id": "a21-session-44-1b-f6-e2-6a-60",
				"first_seen_ms":   1,
				"last_seen_ms":    2,
			}},
		})
	})
	mux.HandleFunc("/v1/traces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		events := xiaozhiRealtimeParityTraceEvents(mode)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"trace_id": "a21-trace-44-1b-f6-e2-6a-60",
			"events":   events,
			"summary": map[string]any{
				"event_count":                        len(events),
				"last_offset_ms":                     780,
				"xiaozhi_listen_to_audio_ingress_ms": 60,
				"xiaozhi_opus_decode_ms":             20,
				"asr_first_partial_ms":               map[bool]any{true: 70, false: nil}[mode == "realtime"],
				"llm_first_content_ms":               20,
				"tts_first_audio_ms":                 30,
				"audio_downlink_first_frame_ms":      20,
				"device_playback_start_ms":           30,
				"answer_first_audio_total_ms":        310,
			},
		})
	})
	return httptest.NewServer(mux)
}

func xiaozhiRealtimeParityTraceEvents(mode string) []map[string]any {
	events := []map[string]any{
		xiaozhiRealtimeParityTraceEvent("xiaozhi.hello.received", 1000, 0),
		xiaozhiRealtimeParityTraceEvent("xiaozhi.listen.start", 1010, 10),
		xiaozhiRealtimeParityTraceEvent("xiaozhi.opus_frame.received", 1040, 40),
		xiaozhiRealtimeParityTraceEvent("xiaozhi.opus_frame.decoded", 1060, 60),
		xiaozhiRealtimeParityTraceEvent("audio.ingress.buffered", 1070, 70),
		xiaozhiRealtimeParityTraceEvent("vad.speech.start", 1080, 80),
	}
	if mode == "transport_only" {
		return events
	}
	if mode == "realtime" {
		events = append(events,
			xiaozhiRealtimeParityTraceEvent("asr.stream.start", 1090, 90),
			xiaozhiRealtimeParityTraceEvent("asr.audio.append", 1100, 100),
			xiaozhiRealtimeParityTraceEvent("asr.first_partial", 1120, 120),
			xiaozhiRealtimeParityTraceEvent("provider.first_content", 1160, 160),
			xiaozhiRealtimeParityTraceEvent("tts.first_audio", 1190, 190),
			xiaozhiRealtimeParityTraceEvent("audio.downlink.first_frame", 1210, 210),
			xiaozhiRealtimeParityTraceEvent("xiaozhi.tts.opus_frame.downlink", 1210, 210),
			xiaozhiRealtimeParityTraceEvent("xiaozhi.voice_pipeline.answer.downlink", 1210, 210),
		)
	}
	events = append(events,
		xiaozhiRealtimeParityTraceEvent("vad.speech.end", 1300, 300),
		xiaozhiRealtimeParityTraceEvent("xiaozhi.listen.auto_stop", 1310, 310),
		xiaozhiRealtimeParityTraceEvent("asr.final", 1340, 340),
	)
	if mode != "realtime" {
		events = append(events,
			xiaozhiRealtimeParityTraceEvent("xiaozhi.voice_pipeline.start", 1320, 320),
			xiaozhiRealtimeParityTraceEvent("provider.first_content", 1400, 400),
			xiaozhiRealtimeParityTraceEvent("tts.first_audio", 1440, 440),
			xiaozhiRealtimeParityTraceEvent("audio.downlink.first_frame", 1460, 460),
			xiaozhiRealtimeParityTraceEvent("xiaozhi.tts.opus_frame.downlink", 1460, 460),
			xiaozhiRealtimeParityTraceEvent("xiaozhi.voice_pipeline.answer.downlink", 1460, 460),
		)
	} else {
		events = append(events, xiaozhiRealtimeParityTraceEvent("xiaozhi.voice_pipeline.start", 1090, 90))
	}
	if mode == "fake_say" {
		events = append(events, xiaozhiRealtimeParityTraceEvent("xiaozhi.say.start", 1470, 470))
	}
	events = append(events,
		xiaozhiRealtimeParityTraceEvent("device.playback.start", 1490, 490),
		xiaozhiRealtimeParityTraceEvent("xiaozhi.voice_pipeline.completed", 1780, 780),
	)
	return events
}

func xiaozhiRealtimeParityTraceEvent(name string, atMS int, offsetMS int) map[string]any {
	return map[string]any{
		"name":       name,
		"trace_id":   "a21-trace-44-1b-f6-e2-6a-60",
		"session_id": "a21-session-44-1b-f6-e2-6a-60",
		"device_id":  "44:1b:f6:e2:6a:60",
		"at_ms":      atMS,
		"offset_ms":  offsetMS,
	}
}
