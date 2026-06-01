package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunXiaozhiPhysicalEvidenceAcceptsGatewayTraceMarkers(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false)
	dir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"xiaozhi-physical-evidence",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	reportJSON := newestXiaozhiPhysicalEvidenceReport(t, dir, stdout.String())
	for _, want := range []string{
		`"schema_version": "a21.xiaozhi_physical_evidence.v1"`,
		`"execution_mode": "physical_xiaozhi_gateway"`,
		`"device_id": "44:1b:f6:e2:6a:60"`,
		`"trace_id": "a21-trace-44-1b-f6-e2-6a-60"`,
		`"session_id": "a21-session-44-1b-f6-e2-6a-60"`,
		`"profile": "stock"`,
		`"physical_xiaozhi.online": {`,
		`"xiaozhi.profile.stock": {`,
		`"xiaozhi.opus.decode": {`,
		`"audio.ingress.pcm": {`,
		`"vad.speech.end": {`,
		`"xiaozhi.listen.auto_stop": {`,
		`"xiaozhi.tts.downlink": {`,
		`"answer.first_downlink": {`,
		`"gateway_answer_first_downlink_ms": {`,
		`"value_ms": 310`,
		`"audio_frame_count": 2`,
		`"audio_payload_stored": false`,
		`"encoded_audio_payload_stored": false`,
		`"promotion_gate": "not_production"`,
		`"acceptance_status": "candidate_gateway_downlink"`,
		`"prd_accepted": false`,
		`"code": "xiaozhi_physical_device_playback_ack_missing"`,
		`"code": "xiaozhi_physical_operator_observation_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
	for _, forbidden := range []string{
		server.URL,
		dir,
		"data_base64",
		"raw_audio",
		"provider output",
		"transcript",
		"secret-token",
		`"device_playback_start_ms": {` + "\n      " + `"available": true`,
		`"speech_end_to_first_audible_response_ms": {` + "\n      " + `"available": true`,
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report leaked or overclaimed %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func TestRunXiaozhiPhysicalEvidenceRejectsUnsafeGatewayValuesWithoutLeak(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, true)
	dir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"xiaozhi-physical-evidence",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("code = 0, want unsafe gateway values rejected: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	rendered := stdout.String() + stderr.String()
	if !strings.Contains(rendered, "xiaozhi physical evidence gateway data unsafe") {
		t.Fatalf("missing safe error: %s", rendered)
	}
	for _, forbidden := range []string{
		"operator transcript should not leak",
		"http://example.com/unsafe/full/url",
		"secret-token",
		"/Users/",
		server.URL,
		dir,
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("unsafe value leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestProductReadinessIngestsXiaozhiPhysicalEvidenceWithoutPRDAcceptance(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"44:1b:f6:e2:6a:60","identity_status":"unknown","connection_status":"online","capabilities":{"xiaozhi_profile":"stock","xiaozhi_transport":"websocket","xiaozhi_audio":"opus_16000hz_mono_60ms","microphone":"available_stock_xiaozhi"},"first_seen_ms":1,"last_seen_ms":2}]}`)
	evidence := writeProductReadinessXiaozhiPhysicalEvidenceReportFixture(t)

	report := buildProductReadinessReport(t.Context(), productReadinessOptions{
		GatewayURL:              server.URL,
		DeviceID:                "44:1b:f6:e2:6a:60",
		PhysicalStackChanReport: evidence,
	}, []string{"A21_PROVIDER_PRIMARY=mock"})

	physical := report.StackChan.PhysicalEvidence
	if report.LaunchReady || physical.PRDAccepted || physical.PRDPhysicalAccepted {
		t.Fatalf("launch/physical = %v/%+v, want gateway-only xiaozhi evidence not PRD accepted", report.LaunchReady, physical)
	}
	if !physical.Valid ||
		!physical.CandidatePhysicalVoiceEvidence ||
		!physical.GatewayDownlinkPhysicalDeviceEvidence ||
		physical.RequiredPhysicalMetricsAvailable ||
		physical.OperatorInstrumentObservationAvailable ||
		physical.AcceptanceStatus != "candidate_gateway_downlink" {
		t.Fatalf("physical evidence = %+v, want candidate gateway-downlink voice evidence only", physical)
	}
	for _, want := range []string{
		"device playback ack",
		"operator audible observation",
	} {
		if !containsProductAction(report.NextActions, want) {
			t.Fatalf("next actions = %#v, want %q", report.NextActions, want)
		}
	}
	for _, code := range []string{
		"xiaozhi_physical_gateway_downlink_candidate",
		"xiaozhi_physical_device_playback_ack_missing",
		"xiaozhi_physical_operator_observation_missing",
	} {
		if !containsProductFinding(report.Findings, code, "") && !containsXiaozhiPhysicalString(physical.FindingCodes, code) {
			t.Fatalf("missing finding code %q: findings=%#v physical=%+v", code, report.Findings, physical)
		}
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{evidence, filepath.Dir(evidence), server.URL, "http://", "https://", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func newXiaozhiPhysicalEvidenceTestServer(t *testing.T, unsafe bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/devices", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "a21.gateway.devices.v1",
			"service":        "a21-gateway",
			"devices": []map[string]any{{
				"device_id":         "44:1b:f6:e2:6a:60",
				"identity_status":   "unknown",
				"connection_status": "online",
				"capabilities": map[string]string{
					"xiaozhi_profile":   "stock",
					"xiaozhi_transport": "websocket",
					"xiaozhi_audio":     "opus_16000hz_mono_60ms",
				},
				"last_trace_id":   "a21-trace-44-1b-f6-e2-6a-60",
				"last_session_id": "a21-session-44-1b-f6-e2-6a-60",
				"first_seen_ms":   1,
				"last_seen_ms":    2,
			}},
		})
	})
	mux.HandleFunc("/v1/traces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		events := []map[string]any{
			{"name": "xiaozhi.hello.received", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1000, "offset_ms": 0},
			{"name": "xiaozhi.listen.start", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1010, "offset_ms": 10},
			{"name": "xiaozhi.opus_frame.decoded", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1040, "offset_ms": 40},
			{"name": "audio.ingress.buffered", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1060, "offset_ms": 60},
			{"name": "vad.speech.end", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1200, "offset_ms": 200},
			{"name": "xiaozhi.listen.auto_stop", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1210, "offset_ms": 210},
			{"name": "xiaozhi.voice_pipeline.start", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1220, "offset_ms": 220},
			{"name": "tts.first_audio", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1290, "offset_ms": 290},
			{"name": "xiaozhi.tts.opus_frame.downlink", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1310, "offset_ms": 310},
			{"name": "audio.downlink.first_frame", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1310, "offset_ms": 310},
			{"name": "xiaozhi.voice_pipeline.completed", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1600, "offset_ms": 600},
		}
		if unsafe {
			events = append(events, map[string]any{"name": "operator transcript should not leak", "trace_id": "http://example.com/unsafe/full/url", "session_id": "/Users/secret-token/session", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1601, "offset_ms": 601})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"trace_id": "a21-trace-44-1b-f6-e2-6a-60",
			"events":   events,
			"summary": map[string]any{
				"event_count":                        len(events),
				"last_offset_ms":                     600,
				"xiaozhi_listen_to_audio_ingress_ms": 60,
				"xiaozhi_opus_decode_ms":             40,
				"tts_first_audio_ms":                 290,
				"audio_downlink_first_frame_ms":      310,
				"answer_first_audio_total_ms":        310,
			},
		})
	})
	mux.HandleFunc("/v1/audio/recent", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "a21.gateway.audio_recent.v1",
			"device_id":      "44:1b:f6:e2:6a:60",
			"trace_id":       "a21-trace-44-1b-f6-e2-6a-60",
			"session_id":     "a21-session-44-1b-f6-e2-6a-60",
			"include_audio":  false,
			"frames": []map[string]any{
				{"device_id": "44:1b:f6:e2:6a:60", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "seq": 1, "received_at_ms": 1001, "sample_rate_hz": 16000, "channels": 1, "duration_ms": 60, "data_bytes": 1920, "rms": 0.12, "vad_detector": "a21-vad", "vad_status": "speech", "speech_detected": true, "speech_active": true, "ingress_buffer_frames": 1},
				{"device_id": "44:1b:f6:e2:6a:60", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "seq": 2, "received_at_ms": 1061, "sample_rate_hz": 16000, "channels": 1, "duration_ms": 60, "data_bytes": 1920, "rms": 0.09, "vad_detector": "a21-vad", "vad_status": "speech_end", "speech_detected": true, "speech_active": false, "ingress_buffer_frames": 2},
			},
		})
	})
	return httptest.NewServer(mux)
}

func newestXiaozhiPhysicalEvidenceReport(t *testing.T, outputDir string, stdout string) string {
	t.Helper()
	if !strings.Contains(stdout, `"report_path": "a21-xiaozhi-physical-evidence-`) {
		t.Fatalf("stdout missing basename report path: %s", stdout)
	}
	if strings.Contains(stdout, outputDir) {
		t.Fatalf("stdout leaked output dir: %s", stdout)
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-xiaozhi-physical-evidence-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func writeProductReadinessXiaozhiPhysicalEvidenceReportFixture(t *testing.T) string {
	t.Helper()
	report := map[string]any{
		"schema_version":         "a21.xiaozhi_physical_evidence.v1",
		"execution_mode":         "physical_xiaozhi_gateway",
		"device_id":              "44:1b:f6:e2:6a:60",
		"trace_id":               "a21-trace-44-1b-f6-e2-6a-60",
		"session_id":             "a21-session-44-1b-f6-e2-6a-60",
		"profile":                "stock",
		"physical_device_online": true,
		"audio_frame_count":      2,
		"promotion_gate":         "not_production",
		"acceptance_status":      "candidate_gateway_downlink",
		"prd_accepted":           false,
		"execution": map[string]any{
			"provider_executed": false,
			"v21_executed":      false,
			"hardware_executed": false,
		},
		"gateway_metrics": map[string]any{
			"gateway_answer_first_downlink_ms": map[string]any{"available": true, "value_ms": 310, "source": "gateway_trace"},
		},
		"stage_availability": map[string]any{
			"physical_xiaozhi.online":      map[string]any{"available": true, "source": "gateway_device_registry"},
			"xiaozhi.profile.stock":        map[string]any{"available": true, "source": "gateway_device_registry"},
			"xiaozhi.opus.decode":          map[string]any{"available": true, "source": "gateway_trace"},
			"audio.ingress.pcm":            map[string]any{"available": true, "source": "gateway_trace"},
			"vad.speech.end":               map[string]any{"available": true, "source": "gateway_trace"},
			"xiaozhi.listen.auto_stop":     map[string]any{"available": true, "source": "gateway_trace"},
			"xiaozhi.tts.downlink":         map[string]any{"available": true, "source": "gateway_trace"},
			"answer.first_downlink":        map[string]any{"available": true, "value_ms": 310, "source": "gateway_trace"},
			"device.playback.ack":          map[string]any{"available": false},
			"operator.audible_observation": map[string]any{"available": false},
		},
		"canonical_metrics": map[string]any{
			"device_downlink_first_frame_ms":          map[string]any{"available": false},
			"device_playback_start_ms":                map[string]any{"available": false},
			"speech_end_to_first_audible_response_ms": map[string]any{"available": false},
			"barge_in_detected_ms":                    map[string]any{"available": false},
			"barge_in_stop_ms":                        map[string]any{"available": false},
			"barge_in_playback_stop_requested_ms":     map[string]any{"available": false},
			"barge_in_playback_stop_done_ms":          map[string]any{"available": false},
		},
		"mic": map[string]any{"available": true, "frames_captured": 2, "frames_delivered": 2, "rms": 0.105, "delivery_ratio": 1.0, "nonzero_sample_count": 3840},
		"observation": map[string]any{
			"available":               false,
			"physical_sound_observed": false,
			"operator_confirmed":      false,
		},
		"findings": []map[string]any{
			{"code": "xiaozhi_physical_gateway_downlink_candidate", "severity": "info", "message": "stock Xiaozhi physical device reached Gateway downlink, but this is not audible playback acceptance"},
			{"code": "xiaozhi_physical_device_playback_ack_missing", "severity": "error", "message": "missing device playback ack such as device.playback.start or trusted runtime echo"},
			{"code": "xiaozhi_physical_operator_observation_missing", "severity": "error", "message": "missing operator audible observation or instrumented first audible playback evidence"},
		},
		"redaction": map[string]any{
			"user_text_stored":             false,
			"instruction_text_stored":      false,
			"model_text_stored":            false,
			"audio_payload_stored":         false,
			"encoded_audio_payload_stored": false,
			"network_locator_stored":       false,
			"network_route_stored":         false,
			"filesystem_locator_stored":    false,
			"secret_material_stored":       false,
			"internal_thought_stored":      false,
		},
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-xiaozhi-physical-evidence-report.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func containsXiaozhiPhysicalString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
