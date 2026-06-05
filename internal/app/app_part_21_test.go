package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunStackChanHalfDuplexAcceptanceConfirmsMicTriggeredPlayback(t *testing.T) {
	probeStarted := false
	idleControlSeen := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				DeviceID                     string `json:"device_id"`
				State                        string `json:"state"`
				TraceID                      string `json:"trace_id"`
				SessionID                    string `json:"session_id"`
				AudioProbeOnly               bool   `json:"audio_probe_only"`
				MockPlaybackOnNextAudioFrame bool   `json:"mock_playback_on_next_audio_frame"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.DeviceID != "stackchan-001" {
				t.Fatalf("device_id = %q", payload.DeviceID)
			}
			if payload.State == "listening" {
				probeStarted = true
				if payload.AudioProbeOnly {
					t.Fatalf("half-duplex listening must allow Gateway playback: %+v", payload)
				}
				if !payload.MockPlaybackOnNextAudioFrame {
					t.Fatalf("half-duplex listening must arm one next-frame mock playback: %+v", payload)
				}
			}
			if payload.State == "idle" {
				idleControlSeen = true
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":%q,"session_id":%q,"device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`, payload.TraceID, payload.SessionID)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if probeStarted {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"speaking","servo_y":"48deg","rgb":"#002430","mic_frames_captured":"13","audio_ws_sent_audio_frames":"13","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"800","mic_last_nonzero_samples":"318","playback_buffer_total_chunks":"11","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"6","speaker_driver_errors":"0","speaker_last_stream_id":"a21-audio-stream-000001"},"identity_status":"ok","connection_status":"online","current_expression":"speaking","playback_stream_id":"a21-audio-stream-000001","last_session_id":"a21-session-half-duplex","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","mic_frames_captured":"10","audio_ws_sent_audio_frames":"10","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"20","mic_last_nonzero_samples":"20","playback_buffer_total_chunks":"10","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"5","speaker_driver_errors":"0","speaker_last_stream_id":"a21-old-stream"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-before","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if probeStarted {
				fmt.Fprint(w, strings.Join([]string{
					"a21_audio_frame_total 31",
					"a21_audio_ingress_frames_total 31",
					"a21_audio_ingress_rms 0.0042",
					"a21_audio_playback_chunk_total 9",
					`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 1`,
				}, "\n"))
				return
			}
			fmt.Fprint(w, strings.Join([]string{
				"a21_audio_frame_total 20",
				"a21_audio_ingress_frames_total 20",
				"a21_audio_ingress_rms 0.0001",
				"a21_audio_playback_chunk_total 8",
			}, "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-half-duplex-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--min-mic-frames", "1",
		"--min-playback-chunks", "1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !idleControlSeen {
		t.Fatalf("idle control not seen")
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_half_duplex_acceptance.v1"`,
		`"half_duplex_acceptance_status": "confirmed"`,
		`"mic_frames_captured_delta": 3`,
		`"audio_ws_sent_audio_frames_delta": 3`,
		`"gateway_audio_ingress_frames_delta": 11`,
		`"gateway_audio_playback_chunk_delta": 1`,
		`"playback_buffer_total_chunks_delta": 1`,
		`"speaker_frames_played_delta": 1`,
		`"audio_ws_delivery_ratio": 1`,
		"stackchan half-duplex acceptance ok (instrumented only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanSpeakerAcceptanceConfirmsInstrumentedDownlink(t *testing.T) {
	probeStarted := false
	probeStopped := false
	var speakingControlSeen bool
	var idleControlSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				DeviceID        string `json:"device_id"`
				State           string `json:"state"`
				Mode            string `json:"mode"`
				Text            string `json:"text"`
				TraceID         string `json:"trace_id"`
				SessionID       string `json:"session_id"`
				StreamID        string `json:"stream_id"`
				MockAudioChunks int    `json:"mock_audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.DeviceID != "stackchan-001" {
				t.Fatalf("device_id = %q", payload.DeviceID)
			}
			if payload.State == "speaking" {
				speakingControlSeen = true
				probeStarted = true
				if payload.MockAudioChunks != 8 {
					t.Fatalf("mock_audio_chunks = %d, want 8 padded playback frames", payload.MockAudioChunks)
				}
				if payload.StreamID != "a21-speaker-acceptance-stream" {
					t.Fatalf("stream_id = %q", payload.StreamID)
				}
				if payload.TraceID == "" || payload.SessionID == "" {
					t.Fatalf("speaker control missing trace/session: %+v", payload)
				}
			}
			if payload.State == "idle" {
				idleControlSeen = true
				probeStopped = true
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":%q,"session_id":%q,"device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`, payload.TraceID, payload.SessionID)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if probeStarted {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"speaking","servo_y":"48deg","rgb":"#002430","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"14","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"24","speaker_busy_ticks":"4","speaker_driver_errors":"0","speaker_last_stream_id":"a21-speaker-acceptance-stream"},"identity_status":"ok","connection_status":"online","current_expression":"speaking","playback_stream_id":"a21-speaker-acceptance-stream","last_session_id":"a21-session-speaker-acceptance","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"10","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"20","speaker_busy_ticks":"3","speaker_driver_errors":"0","speaker_last_stream_id":"a21-old-stream"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-before","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if probeStarted {
				fmt.Fprint(w, "a21_audio_playback_chunk_total 13\n")
				return
			}
			fmt.Fprint(w, "a21_audio_playback_chunk_total 9\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-speaker-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--mock-audio-chunks", "4",
		"--min-played-frames", "4",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !speakingControlSeen || !idleControlSeen || !probeStopped {
		t.Fatalf("speakingControlSeen=%v idleControlSeen=%v probeStopped=%v", speakingControlSeen, idleControlSeen, probeStopped)
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_speaker_acceptance.v1"`,
		`"speaker_acceptance_status": "confirmed"`,
		`"hardware_acceptance_scope": "instrumented_speaker_downlink"`,
		`"physical_sound_observed": false`,
		`"mock_audio_chunks": 4`,
		`"stream_id": "a21-speaker-acceptance-stream"`,
		`"playback_buffer_total_chunks_delta": 4`,
		`"speaker_frames_played_delta": 4`,
		`"gateway_audio_playback_chunk_delta": 4`,
		"stackchan speaker acceptance ok (instrumented only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-speaker-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("speaker acceptance reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanSpeakerAcceptanceBatchesAudibleProbe(t *testing.T) {
	var totalChunks int
	var speakingCalls int
	var idleControlSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				DeviceID        string `json:"device_id"`
				State           string `json:"state"`
				TraceID         string `json:"trace_id"`
				SessionID       string `json:"session_id"`
				StreamID        string `json:"stream_id"`
				MockAudioChunks int    `json:"mock_audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.DeviceID != "stackchan-001" {
				t.Fatalf("device_id = %q", payload.DeviceID)
			}
			if payload.State == "speaking" {
				speakingCalls++
				totalChunks += payload.MockAudioChunks
				if payload.MockAudioChunks != 8 {
					t.Fatalf("speaker batch mock_audio_chunks = %d, want 8 padded playback frames", payload.MockAudioChunks)
				}
				if payload.StreamID != "a21-speaker-acceptance-stream" {
					t.Fatalf("stream_id = %q", payload.StreamID)
				}
			}
			if payload.State == "idle" {
				idleControlSeen = true
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":%q,"session_id":%q,"device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`, payload.TraceID, payload.SessionID)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if totalChunks >= 50 {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"speaking","servo_y":"48deg","rgb":"#002430","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"60","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"70","speaker_busy_ticks":"4","speaker_driver_errors":"0","speaker_last_stream_id":"a21-speaker-acceptance-stream"},"identity_status":"ok","connection_status":"online","current_expression":"speaking","playback_stream_id":"a21-speaker-acceptance-stream","last_session_id":"a21-session-speaker-acceptance","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"10","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"20","speaker_busy_ticks":"3","speaker_driver_errors":"0","speaker_last_stream_id":"a21-old-stream"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-before","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if totalChunks >= 50 {
				fmt.Fprint(w, "a21_audio_playback_chunk_total 59\n")
				return
			}
			fmt.Fprint(w, "a21_audio_playback_chunk_total 9\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-speaker-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--mock-audio-chunks", "50",
		"--min-played-frames", "50",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if speakingCalls < 2 || totalChunks != 56 || !idleControlSeen {
		t.Fatalf("speakingCalls=%d totalChunks=%d idleControlSeen=%v", speakingCalls, totalChunks, idleControlSeen)
	}
	for _, want := range []string{
		`"mock_audio_chunks": 50`,
		`"expected_audio_duration_ms": 1000`,
		`"playback_buffer_total_chunks_delta": 50`,
		`"speaker_frames_played_delta": 50`,
		`"gateway_audio_playback_chunk_delta": 50`,
		"stackchan speaker acceptance ok (instrumented only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestDefaultStackChanSpeakerAcceptanceWindowLeavesPlaybackMargin(t *testing.T) {
	options := defaultStackChanSpeakerAcceptanceOptions()
	expectedAudioDurationMS := options.MockAudioChunks * stackChanSpeakerProbeChunkDurationMS
	if options.WindowMS < expectedAudioDurationMS+500 {
		t.Fatalf("default speaker window = %dms, want at least %dms for playback and runtime echo margin", options.WindowMS, expectedAudioDurationMS+500)
	}
}

func TestRunStackChanSpeakerAcceptanceBlocksBeforePlaybackOnFirmwareMismatch(t *testing.T) {
	controlCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			controlCalled = true
			http.Error(w, "control should not be called", http.StatusInternalServerError)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"older123"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"10","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"20","speaker_busy_ticks":"0","speaker_driver_errors":"0","speaker_last_stream_id":"a21-old-stream"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "a21_audio_playback_chunk_total 9\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-speaker-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "5a4936f9a993",
		"--window-ms", "1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code = 0, want blocked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if controlCalled {
		t.Fatalf("speaker control was called despite firmware commit mismatch")
	}
	for _, want := range []string{
		`"speaker_acceptance_status": "blocked"`,
		`"code": "firmware_commit_mismatch"`,
		"stackchan speaker acceptance blocked (instrumented only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanTouchAcceptancePassesTopTapWithGatewayTrace(t *testing.T) {
	var armed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			if r.Method != http.MethodPost {
				t.Fatalf("control method = %s", r.Method)
			}
			armed = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"trace_id":"a21-trace-touch-acceptance-top_tap","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		case "/v1/devices":
			lastEvent := ""
			lastTouchEvent := ""
			lastSource := ""
			lastTrace := ""
			lastTouchTrace := ""
			lastSeen := int64(1780000001000)
			lastTouchSeen := int64(0)
			if armed {
				lastEvent = "device.heartbeat"
				lastTouchEvent = "touch.top.tap"
				lastSource = "top_sensor"
				lastTrace = "a21-trace-device-000123"
				lastTouchTrace = "a21-trace-device-000123"
				lastSeen = time.Now().UnixMilli()
				lastTouchSeen = lastSeen
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abc123"},"capabilities":{"screen":"available","screen_touch":"available","top_touch":"available","speaker":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_event":%q,"last_touch_event":%q,"last_touch_source":%q,"last_trace_id":%q,"last_touch_trace_id":%q,"last_session_id":"a21-session-touch-acceptance","last_touch_session_id":"a21-session-touch-acceptance","last_seen_ms":%d,"last_touch_seen_ms":%d,"first_seen_ms":1780000000000}]}`, lastEvent, lastTouchEvent, lastSource, lastTrace, lastTouchTrace, lastSeen, lastTouchSeen)
		case "/v1/traces":
			if r.URL.Query().Get("trace_id") != "a21-trace-device-000123" {
				t.Fatalf("trace id = %q", r.URL.Query().Get("trace_id"))
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":"a21-trace-device-000123","events":[{"name":"device.touch.top.tap.received","trace_id":"a21-trace-device-000123","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","at_ms":%d,"offset_ms":0}],"summary":{"event_count":1,"last_offset_ms":0}}`, time.Now().UnixMilli())
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-touch-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--case", "top_tap",
		"--window-ms", "1000",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_touch_acceptance.v1"`,
		`"touch_acceptance_status": "passed"`,
		`"expected_event": "touch.top.tap"`,
		"stackchan touch acceptance ok (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-touch-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("touch acceptance reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanTouchAcceptanceMarksDirectionalCaseAsAffordanceBoundWhenMissed(t *testing.T) {
	var armed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			armed = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"trace_id":"a21-trace-touch-acceptance-top_swipe_backward","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		case "/v1/devices":
			lastEvent := ""
			lastSource := ""
			lastTrace := ""
			lastSeen := int64(1780000001000)
			if armed {
				lastEvent = "touch.top.tap"
				lastSource = "top_sensor"
				lastTrace = "a21-trace-device-000124"
				lastSeen = time.Now().UnixMilli()
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abc123"},"capabilities":{"screen":"available","screen_touch":"available","top_touch":"available","speaker":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_event":%q,"last_touch_source":%q,"last_trace_id":%q,"last_session_id":"a21-session-touch-acceptance","last_seen_ms":%d,"first_seen_ms":1780000000000}]}`, lastEvent, lastSource, lastTrace, lastSeen)
		case "/v1/traces":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":"a21-trace-device-000124","events":[{"name":"device.touch.top.tap.received","trace_id":"a21-trace-device-000124","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","at_ms":%d,"offset_ms":0}],"summary":{"event_count":1,"last_offset_ms":0}}`, time.Now().UnixMilli())
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-touch-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--case", "top_swipe_backward",
		"--window-ms", "50",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"acceptance_scope": "guided_directional_touch"`,
		`"needs_affordance": true`,
		`"code": "physical_affordance_required"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanHardwareMainlineReportsOrderedPlannedTracks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef123456"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","servo_x":"planned_continuous_rotation_axis","rgb":"available","camera":"planned_core_s3_camera","imu":"planned_9_axis_imu","ambient_light":"planned_ambient_light_sensor","proximity":"planned_proximity_sensor","battery":"planned_550mah_battery","nfc":"planned_nfc","infrared":"planned_infrared_tx_rx"},"runtime_echo":{"screen":"idle","servo_y":"48deg","rgb":"#002430"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-hardware-mainline","last_seen_ms":%d}]}`, time.Now().UnixMilli())
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-hardware-mainline",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_hardware_mainline.v1"`,
		`"hardware_mainline_status": "ready_for_diagnostic_spikes"`,
		`"flash_allowed": false`,
		`"delete_allowed": false`,
		`"capability_invariants":`,
		`"capability": "microphone"`,
		`"accepted": true`,
		`"capability": "imu"`,
		`"declared_status": "planned_9_axis_imu"`,
		`"promotion_allowed": false`,
		"stackchan hardware mainline ready (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Index(stdout.String(), `"capability": "imu"`) > strings.Index(stdout.String(), `"capability": "camera"`) {
		t.Fatalf("hardware mainline order should probe IMU before camera: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-hardware-mainline-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("hardware mainline reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanHardwareMainlineBlocksFalseAvailablePromotion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef123456"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","servo_x":"planned_continuous_rotation_axis","rgb":"available","camera":"planned_core_s3_camera","imu":"available","ambient_light":"planned_ambient_light_sensor","proximity":"planned_proximity_sensor","battery":"planned_550mah_battery","nfc":"planned_nfc","infrared":"planned_infrared_tx_rx"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-hardware-mainline","last_seen_ms":%d}]}`, time.Now().UnixMilli())
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-hardware-mainline",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"hardware_mainline_status": "blocked"`,
		`"code": "capability_status_not_honest"`,
		`"capability": "imu"`,
		`"declared_status": "available"`,
		`"accepted": false`,
		"stackchan hardware mainline blocked (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanHardwareMainlineBlocksReleaseMicrophonePromotion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef123456"},"capabilities":{"microphone":"available","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","servo_x":"planned_continuous_rotation_axis","rgb":"available","camera":"planned_core_s3_camera","imu":"planned_9_axis_imu","ambient_light":"planned_ambient_light_sensor","proximity":"planned_proximity_sensor","battery":"planned_550mah_battery","nfc":"planned_nfc","infrared":"planned_infrared_tx_rx"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-hardware-mainline","last_seen_ms":%d}]}`, time.Now().UnixMilli())
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-hardware-mainline",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"hardware_mainline_status": "blocked"`,
		`"code": "capability_status_not_honest"`,
		`"capability": "microphone"`,
		`"declared_status": "available"`,
		`"accepted": false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanHardwareMainlineBlocksMissingPlannedCapability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef123456"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","servo_x":"planned_continuous_rotation_axis","rgb":"available","ambient_light":"planned_ambient_light_sensor","proximity":"planned_proximity_sensor","battery":"planned_550mah_battery","nfc":"planned_nfc","infrared":"planned_infrared_tx_rx"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-hardware-mainline","last_seen_ms":%d}]}`, time.Now().UnixMilli())
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-hardware-mainline",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"hardware_mainline_status": "blocked"`,
		`"code": "capability_missing"`,
		"stackchan hardware mainline blocked (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanHardwareMainlineRejectsLegacyDeviceIDWithoutEchoingIt(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-hardware-mainline",
		"--gateway-url", "http://127.0.0.1:21080",
		"--device-id", "x21-stackchan-001",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21") {
		t.Fatalf("legacy identity leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "device id contains forbidden legacy identity") {
		t.Fatalf("stderr missing legacy identity guard: %s", stderr.String())
	}
}

func TestRunStackChanPhysicalEvidenceWritesPendingTemplateFromIdentity(t *testing.T) {
	dir := t.TempDir()
	artifactSHA := strings.Repeat("c", 64)
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, artifactSHA)
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_physical_evidence.v1"`,
		`"identity_acceptance_report_path": "` + identityPath + `"`,
		`"device_id": "stackchan-001"`,
		`"commit": "abcdef1"`,
		`"artifact_sha256": "` + artifactSHA + `"`,
		`"capability": "microphone"`,
		`"capability": "rgb"`,
		`"status": "pending"`,
		`"evidence_type": "operator_observation_required"`,
		`"flash_allowed": false`,
		"stackchan physical evidence template written (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-physical-evidence-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("physical evidence reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanPhysicalEvidenceCanFeedCapabilityAcceptance(t *testing.T) {
	dir := t.TempDir()
	artifactSHA := strings.Repeat("d", 64)
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, artifactSHA)
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	args := []string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}
	for _, pass := range []string{
		"microphone=gateway_audio_frame",
		"speaker=audible_playback",
		"screen=operator_visible_state",
		"screen_touch=touch_event",
		"top_touch=touch_event",
		"servo_y=servo_clamped_motion",
		"rgb=operator_visible_state",
	} {
		args = append(args, "--pass", pass)
	}
	code := Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("physical evidence code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	evidencePath := newestGlob(t, filepath.Join(outputDir, "a21-stackchan-physical-evidence-*.json"))
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"stackchan-capability-acceptance",
		"--identity-acceptance", identityPath,
		"--evidence", evidencePath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("capability acceptance code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"capability_acceptance_status": "confirmed"`) {
		t.Fatalf("capability acceptance not confirmed: %s", stdout.String())
	}
}

func TestRunStackChanPhysicalEvidenceDerivesGatewayObservableSignals(t *testing.T) {
	dir := t.TempDir()
	artifactSHA := strings.Repeat("f", 64)
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, artifactSHA)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/devices":
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "speaking",
      "runtime_echo": {"screen": "speaking", "servo_y": "48deg", "rgb": "#002430"},
      "last_event": "touch.wake_or_listen",
      "last_touch_source": "screen",
      "last_trace_id": "a21-trace-physical-auto",
      "firmware": {"id": "a21-stackchan", "version": "0.1.0", "board": "m5stack-cores3", "commit": "abcdef1"},
      "capabilities": {
        "microphone": "available",
        "speaker": "available",
        "screen": "available",
        "screen_touch": "available",
        "top_touch": "available",
        "servo_y": "available",
        "rgb": "available"
      },
      "last_seen_ms": 1780000000000
    }
  ]
}`))
		case "/v1/traces":
			if r.URL.Query().Get("trace_id") != "a21-trace-physical-auto" {
				t.Fatalf("trace id = %q, want a21-trace-physical-auto", r.URL.Query().Get("trace_id"))
			}
			_, _ = w.Write([]byte(`{
  "trace_id": "a21-trace-physical-auto",
  "events": [
    {"name": "audio.frame.received", "trace_id": "a21-trace-physical-auto", "device_id": "stackchan-001", "at_ms": 1780000001000},
    {"name": "audio.playback.chunk.sent", "trace_id": "a21-trace-physical-auto", "device_id": "stackchan-001", "at_ms": 1780000002000}
  ],
  "summary": {"event_count": 2}
}`))
		default:
			t.Fatalf("path = %q", r.URL.Path)
		}
	}))
	defer server.Close()

	outputDir := filepath.Join(dir, "reports")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--derive-gateway",
		"--gateway-url", server.URL,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	evidencePath := newestGlob(t, filepath.Join(outputDir, "a21-stackchan-physical-evidence-*.json"))
	data, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`"gateway_url": "` + server.URL + `"`,
		`"gateway_trace_id": "a21-trace-physical-auto"`,
		`"capability": "microphone"`,
		`"evidence_type": "gateway_audio_frame"`,
		`"capability": "speaker"`,
		`"evidence_type": "gateway_audio_downlink"`,
		`"capability": "screen"`,
		`"evidence_type": "device_screen_echo"`,
		`"capability": "servo_y"`,
		`"evidence_type": "device_servo_echo"`,
		`"capability": "rgb"`,
		`"evidence_type": "device_rgb_echo"`,
		`"capability": "screen_touch"`,
		`"evidence_type": "gateway_touch_event"`,
		`"capability": "top_touch"`,
		`"status": "pending"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("evidence missing %q: %s", want, text)
		}
	}
}
