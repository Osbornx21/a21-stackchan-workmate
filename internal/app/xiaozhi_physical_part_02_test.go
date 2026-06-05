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

func TestProductReadinessKeepsXiaozhiPhysicalEvidenceActionWhenDeviceOffline(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	evidence := writeProductReadinessDebugXiaozhiPhysicalEvidenceReportFixture(t)

	report := buildProductReadinessReport(t.Context(), productReadinessOptions{
		GatewayURL:              server.URL,
		DeviceID:                "44:1b:f6:e2:6a:60",
		PhysicalStackChanReport: evidence,
	}, []string{"A21_PROVIDER_PRIMARY=mock"})

	if report.StackChan.PhysicalDeviceOnline {
		t.Fatalf("physical device online = true, want offline fixture")
	}
	if !report.StackChan.PhysicalEvidence.CandidatePhysicalVoiceEvidence {
		t.Fatalf("physical evidence = %+v, want candidate Xiaozhi voice evidence", report.StackChan.PhysicalEvidence)
	}
	for _, want := range []string{
		"bring a physical StackChan online",
		"operator audible observation",
		"barge-in playback stop_done",
	} {
		if !containsProductAction(report.NextActions, want) {
			t.Fatalf("next actions = %#v, want %q", report.NextActions, want)
		}
	}
	if containsProductAction(report.NextActions, "device playback ack") {
		t.Fatalf("next actions = %#v, want no missing playback ack once device_playback_start_ms is available", report.NextActions)
	}
}

func TestProductReadinessIngestsAcceptedXiaozhiPhysicalEvidence(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"44:1b:f6:e2:6a:60","identity_status":"unknown","connection_status":"online","capabilities":{"xiaozhi_profile":"debug","xiaozhi_feature_device_events":"true","xiaozhi_debug_extension_isolated":"true","xiaozhi_transport":"websocket","xiaozhi_audio":"opus_16000hz_mono_60ms","microphone":"available_xiaozhi_opus_ingress"},"first_seen_ms":1,"last_seen_ms":2}]}`)
	evidence := writeProductReadinessAcceptedXiaozhiPhysicalEvidenceReportFixture(t)

	report := buildProductReadinessReport(t.Context(), productReadinessOptions{
		GatewayURL:              server.URL,
		DeviceID:                "44:1b:f6:e2:6a:60",
		PhysicalStackChanReport: evidence,
	}, []string{"A21_PROVIDER_PRIMARY=mock"})

	physical := report.StackChan.PhysicalEvidence
	if !physical.Valid ||
		!physical.PRDAccepted ||
		!physical.PRDPhysicalAccepted ||
		!physical.RequiredPhysicalMetricsAvailable ||
		!physical.MicEvidenceAvailable ||
		!physical.OperatorInstrumentObservationAvailable ||
		!physical.GatewayDownlinkPhysicalDeviceEvidence ||
		physical.CandidatePhysicalVoiceEvidence ||
		physical.CandidatePhysicalEvidence ||
		physical.HostLoopbackOnly ||
		physical.AcceptanceStatus != "prd_accepted" ||
		physical.PromotionGate != "accepted" {
		t.Fatalf("physical evidence = %+v, want accepted xiaozhi physical PRD evidence", physical)
	}
	for _, want := range []string{
		"device_downlink_first_frame_ms",
		"device_playback_start_ms",
		"speech_end_to_first_audible_response_ms",
		"barge_in_detected_ms",
		"barge_in_stop_ms",
		"barge_in_playback_stop_requested_ms",
		"barge_in_playback_stop_done_ms",
	} {
		if !physical.CanonicalMetricAvailability[want] {
			t.Fatalf("canonical availability[%s] = false in %+v", want, physical.CanonicalMetricAvailability)
		}
	}
	if !containsXiaozhiPhysicalString(physical.FindingCodes, "xiaozhi_physical_prd_accepted") {
		t.Fatalf("finding codes = %#v, want xiaozhi_physical_prd_accepted", physical.FindingCodes)
	}
	if containsProductAction(report.NextActions, "physical Xiaozhi PRD acceptance") {
		t.Fatalf("next actions = %#v, want no xiaozhi physical acceptance action after accepted evidence", report.NextActions)
	}
}

func TestRunXiaozhiPhysicalEvidenceAcceptsDebugPlaybackRuntimeEcho(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true, true, true, true)
	dir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{
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
		`"profile": "debug"`,
		`"xiaozhi.profile.debug": {`,
		`"device.playback.ack": {`,
		`"device.playback.stop_done": {`,
		`"device_playback_start_ms": {`,
		`"value_ms": 30`,
		`"barge_in_playback_stop_done_ms": {`,
		`"value_ms": 120`,
		`"promotion_gate": "not_production"`,
		`"acceptance_status": "candidate_gateway_downlink"`,
		`"code": "xiaozhi_physical_operator_observation_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
	for _, forbidden := range []string{
		`"code": "xiaozhi_physical_stock_profile_missing"`,
		`"code": "xiaozhi_physical_device_playback_ack_missing"`,
		"stock Xiaozhi physical device reached Gateway downlink",
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report should not include %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func TestRunXiaozhiPhysicalEvidenceRejectsUnrelatedLateStopDone(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true, true, true, true, true)
	dir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{
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
		`"device.playback.stop_done": {` + "\n      " + `"available": false`,
		`"barge_in_playback_stop_done_ms": {` + "\n      " + `"available": false`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
	for _, forbidden := range []string{
		`"value_ms": 2000`,
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report should not pair late stop_done %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func TestRunXiaozhiHalfDuplexAcceptanceMarksMissingPlaybackStopDoneAndTrustedObservationBelowPRD(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true, true)
	dir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-accept",
		"--check", "xiaozhi-half-duplex",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0 candidate evidence below PRD: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	reportJSON := newestXiaozhiHalfDuplexAcceptanceReport(t, dir, stdout.String())
	for _, want := range []string{
		`"half_duplex_acceptance_status": "candidate_gateway_trace_not_prd"`,
		`"playback_ack_available": true`,
		`"barge_in_stop_done_available": false`,
		`"physical_sound_observed": false`,
		`"prd_accepted": false`,
		`"code": "xiaozhi_half_duplex_stop_done_missing"`,
		`"code": "xiaozhi_half_duplex_audible_observation_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
	for _, forbidden := range []string{
		server.URL,
		`"half_duplex_acceptance_status": "physical_review_required"`,
		`"prd_accepted": true`,
		"transcript",
		"data_base64",
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("half-duplex report leaked or overclaimed %q: stdout=%s report=%s stderr=%s", forbidden, stdout.String(), reportJSON, stderr.String())
		}
	}
}

func TestRunXiaozhiPhysicalPRDReviewPromotesCandidateEvidence(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true, true, true)
	sourceDir := t.TempDir()
	reviewDir := t.TempDir()
	observation := writeXiaozhiInstrumentObservationReport(t, map[string]any{
		"device_playback_observed":                           true,
		"device_playback_observation_source":                 "device_runtime_echo",
		"gateway_first_downlink_to_device_playback_start_ms": 30,
	})

	var physicalStdout bytes.Buffer
	var physicalStderr bytes.Buffer
	physicalCode := RunLab([]string{
		"xiaozhi-physical-evidence",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--instrument-observation-report", observation,
		"--output-dir", sourceDir,
	}, &physicalStdout, &physicalStderr)
	if physicalCode != 0 {
		t.Fatalf("physical evidence code = %d, want 0: stdout=%s stderr=%s", physicalCode, physicalStdout.String(), physicalStderr.String())
	}
	physicalPath := singleXiaozhiReportPath(t, sourceDir, "a21-xiaozhi-physical-evidence-*.json")

	var halfStdout bytes.Buffer
	var halfStderr bytes.Buffer
	halfCode := Run([]string{
		"stackchan-accept",
		"--check", "xiaozhi-half-duplex",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--instrument-observation-report", observation,
		"--output-dir", sourceDir,
	}, &halfStdout, &halfStderr)
	if halfCode != 0 {
		t.Fatalf("half-duplex code = %d, want 0: stdout=%s stderr=%s", halfCode, halfStdout.String(), halfStderr.String())
	}
	halfPath := singleXiaozhiReportPath(t, sourceDir, "a21-xiaozhi-half-duplex-acceptance-*.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{
		"xiaozhi-physical-prd-review",
		"--physical-evidence-report", physicalPath,
		"--half-duplex-report", halfPath,
		"--confirm", xiaozhiPhysicalPRDReviewConfirm,
		"--output-dir", reviewDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	acceptedPath := singleXiaozhiReportPath(t, reviewDir, "a21-xiaozhi-physical-evidence-*.json")
	reportJSON := readTextFile(t, acceptedPath)
	for _, want := range []string{
		`"promotion_gate": "accepted"`,
		`"acceptance_status": "prd_accepted"`,
		`"prd_accepted": true`,
		`"code": "xiaozhi_physical_prd_accepted"`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("accepted report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
	for _, forbidden := range []string{
		server.URL,
		sourceDir,
		reviewDir,
		"candidate_gateway_downlink",
		"transcript",
		"data_base64",
		"raw_audio",
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("accepted report leaked or retained forbidden marker %q: stdout=%s report=%s stderr=%s", forbidden, stdout.String(), reportJSON, stderr.String())
		}
	}

	var accepted xiaozhiPhysicalEvidenceReport
	if err := json.Unmarshal([]byte(reportJSON), &accepted); err != nil {
		t.Fatal(err)
	}
	readiness := buildProductXiaozhiPhysicalReadiness(filepath.Base(acceptedPath), accepted)
	if !readiness.PRDPhysicalAccepted ||
		!readiness.RequiredPhysicalMetricsAvailable ||
		!readiness.MicEvidenceAvailable ||
		!readiness.OperatorInstrumentObservationAvailable ||
		readiness.CandidatePhysicalVoiceEvidence ||
		readiness.PromotionGate != "accepted" ||
		readiness.AcceptanceStatus != "prd_accepted" {
		t.Fatalf("readiness = %+v, want PRD physical accepted", readiness)
	}
}

func TestRunXiaozhiPhysicalPRDReviewRejectsMissingConfirm(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{
		"xiaozhi-physical-prd-review",
		"--physical-evidence-report", "reports/a21-xiaozhi-physical-evidence.json",
		"--half-duplex-report", "reports/a21-xiaozhi-half-duplex-acceptance.json",
	}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), xiaozhiPhysicalPRDReviewConfirm) {
		t.Fatalf("stderr = %s, want explicit confirmation marker", stderr.String())
	}
}

func TestRunXiaozhiPhysicalPRDReviewRejectsMissingStopDone(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true, true)
	sourceDir := t.TempDir()
	reviewDir := t.TempDir()
	observation := writeXiaozhiInstrumentObservationReport(t, map[string]any{
		"device_playback_observed":                           true,
		"device_playback_observation_source":                 "device_runtime_echo",
		"gateway_first_downlink_to_device_playback_start_ms": 30,
	})

	var physicalStdout bytes.Buffer
	var physicalStderr bytes.Buffer
	physicalCode := RunLab([]string{
		"xiaozhi-physical-evidence",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--instrument-observation-report", observation,
		"--output-dir", sourceDir,
	}, &physicalStdout, &physicalStderr)
	if physicalCode != 0 {
		t.Fatalf("physical evidence code = %d, want 0: stdout=%s stderr=%s", physicalCode, physicalStdout.String(), physicalStderr.String())
	}
	physicalPath := singleXiaozhiReportPath(t, sourceDir, "a21-xiaozhi-physical-evidence-*.json")

	var halfStdout bytes.Buffer
	var halfStderr bytes.Buffer
	halfCode := Run([]string{
		"stackchan-accept",
		"--check", "xiaozhi-half-duplex",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--instrument-observation-report", observation,
		"--output-dir", sourceDir,
	}, &halfStdout, &halfStderr)
	if halfCode != 0 {
		t.Fatalf("half-duplex code = %d, want 0 candidate: stdout=%s stderr=%s", halfCode, halfStdout.String(), halfStderr.String())
	}
	halfPath := singleXiaozhiReportPath(t, sourceDir, "a21-xiaozhi-half-duplex-acceptance-*.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{
		"xiaozhi-physical-prd-review",
		"--physical-evidence-report", physicalPath,
		"--half-duplex-report", halfPath,
		"--confirm", xiaozhiPhysicalPRDReviewConfirm,
		"--output-dir", reviewDir,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "blocked") {
		t.Fatalf("stderr = %s, want blocked review", stderr.String())
	}
	if matches, err := filepath.Glob(filepath.Join(reviewDir, "a21-xiaozhi-physical-evidence-*.json")); err != nil || len(matches) != 0 {
		t.Fatalf("promoted reports = %v err=%v, want none", matches, err)
	}
}

func newXiaozhiPhysicalEvidenceTestServer(t *testing.T, unsafe bool, playback ...bool) *httptest.Server {
	t.Helper()
	includePlayback := len(playback) > 0 && playback[0]
	includeBargeIn := len(playback) > 1 && playback[1]
	debugProfile := len(playback) > 3 && playback[3]
	lateStopDone := len(playback) > 4 && playback[4]
	mismatchAudioDevice := len(playback) > 5 && playback[5]
	sessionDrift := len(playback) > 6 && playback[6]
	deviceRuntimeSessionID := "a21-session-44-1b-f6-e2-6a-60"
	requestSessionID := deviceRuntimeSessionID
	if sessionDrift {
		requestSessionID = "a21-session-speaking-barge-test"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/devices", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		profile := "stock"
		if debugProfile {
			profile = "debug"
		}
		capabilities := map[string]string{
			"xiaozhi_profile":   profile,
			"xiaozhi_transport": "websocket",
			"xiaozhi_audio":     "opus_16000hz_mono_60ms",
		}
		if debugProfile {
			capabilities["xiaozhi_feature_device_events"] = "true"
			capabilities["xiaozhi_debug_extension_isolated"] = "true"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "a21.gateway.devices.v1",
			"service":        "a21-gateway",
			"devices": []map[string]any{{
				"device_id":         "44:1b:f6:e2:6a:60",
				"identity_status":   "unknown",
				"connection_status": "online",
				"capabilities":      capabilities,
				"runtime_echo": map[string]string{
					"roleplay_prompt_text_stored": "false",
					"roleplay_memory_text_stored": "false",
					"last_state_reaction_state":   "listening",
					"last_state_reaction_status":  "delivered",
					"robot_led_red":               "0",
					"robot_led_green":             "72",
					"robot_led_blue":              "168",
					"robot_head_yaw":              "0",
					"robot_head_pitch":            "30",
					"robot_head_speed":            "180",
				},
				"last_trace_id":   "a21-trace-44-1b-f6-e2-6a-60",
				"last_session_id": deviceRuntimeSessionID,
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
			{"name": "roleplay.prompt_input.used", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1230, "offset_ms": 230},
			{"name": "tts.first_audio", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1290, "offset_ms": 290},
			{"name": "xiaozhi.tts.opus_frame.downlink", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1310, "offset_ms": 310},
			{"name": "audio.downlink.first_frame", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1310, "offset_ms": 310},
			{"name": "xiaozhi.voice_pipeline.completed", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1600, "offset_ms": 600},
		}
		if sessionDrift {
			events = append([]map[string]any{{
				"name":       "xiaozhi.say.start",
				"trace_id":   "a21-trace-44-1b-f6-e2-6a-60",
				"session_id": requestSessionID,
				"device_id":  "44:1b:f6:e2:6a:60",
				"at_ms":      990,
				"offset_ms":  0,
			}}, events...)
		}
		if includePlayback {
			events = append(events, map[string]any{"name": "device.playback.start", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1340, "offset_ms": 340})
		}
		if includeBargeIn {
			events = append(events,
				map[string]any{"name": "xiaozhi.listen.barge_in", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1700, "offset_ms": 700},
				map[string]any{"name": "barge_in.detected", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1700, "offset_ms": 700},
				map[string]any{"name": "playback.stop", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": 1765, "offset_ms": 765},
			)
		}
		if len(playback) > 2 && playback[2] {
			stopDoneAtMS := 1820
			stopDoneOffsetMS := 820
			if lateStopDone {
				stopDoneAtMS = 3700
				stopDoneOffsetMS = 2700
			}
			events = append(events, map[string]any{"name": "device.playback.stop_done", "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "device_id": "44:1b:f6:e2:6a:60", "at_ms": stopDoneAtMS, "offset_ms": stopDoneOffsetMS})
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
				"device_playback_start_ms":           map[bool]any{true: 30, false: nil}[includePlayback],
				"answer_first_audio_total_ms":        310,
				"barge_in_stop_ms":                   map[bool]any{true: 65, false: nil}[includeBargeIn],
			},
		})
	})
	mux.HandleFunc("/v1/audio/recent", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		audioDeviceID := "44:1b:f6:e2:6a:60"
		if mismatchAudioDevice {
			audioDeviceID = "44:1b:f6:e2:6a:61"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "a21.gateway.audio_recent.v1",
			"device_id":      audioDeviceID,
			"trace_id":       "a21-trace-44-1b-f6-e2-6a-60",
			"session_id":     "a21-session-44-1b-f6-e2-6a-60",
			"include_audio":  false,
			"frames": []map[string]any{
				{"device_id": audioDeviceID, "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "seq": 1, "received_at_ms": 1001, "sample_rate_hz": 16000, "channels": 1, "duration_ms": 60, "data_bytes": 1920, "rms": 0.12, "vad_detector": "a21-vad", "vad_status": "speech", "speech_detected": true, "speech_active": true, "ingress_buffer_frames": 1},
				{"device_id": audioDeviceID, "trace_id": "a21-trace-44-1b-f6-e2-6a-60", "session_id": "a21-session-44-1b-f6-e2-6a-60", "seq": 2, "received_at_ms": 1061, "sample_rate_hz": 16000, "channels": 1, "duration_ms": 60, "data_bytes": 1920, "rms": 0.09, "vad_detector": "a21-vad", "vad_status": "speech_end", "speech_detected": true, "speech_active": false, "ingress_buffer_frames": 2},
			},
		})
	})
	return httptest.NewServer(mux)
}

func writeXiaozhiInstrumentObservationReport(t *testing.T, overrides map[string]any) string {
	t.Helper()
	report := map[string]any{
		"schema_version":                       "a21.xiaozhi_instrument_observation.v1",
		"trace_id":                             "a21-trace-44-1b-f6-e2-6a-60",
		"session_id":                           "a21-session-44-1b-f6-e2-6a-60",
		"device_id":                            "44:1b:f6:e2:6a:60",
		"method":                               "instrument_nonzero_audible_energy",
		"instrument":                           "calibrated_audio_recorder",
		"physical_sound_observed":              true,
		"observed_nonzero_audible_energy":      true,
		"gateway_first_downlink_to_audible_ms": 650,
		"speech_end_to_first_audible_response_ms": 760,
		"audible_energy_rms":                      0.09,
		"noise_floor_rms":                         0.01,
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
	for key, value := range overrides {
		if value == nil {
			delete(report, key)
			continue
		}
		report[key] = value
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-xiaozhi-instrument-observation.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func newestXiaozhiHalfDuplexAcceptanceReport(t *testing.T, outputDir string, stdout string) string {
	t.Helper()
	if !strings.Contains(stdout, `"report_path": "a21-xiaozhi-half-duplex-acceptance-`) {
		t.Fatalf("stdout missing basename report path: %s", stdout)
	}
	if strings.Contains(stdout, outputDir) {
		t.Fatalf("stdout leaked output dir: %s", stdout)
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-xiaozhi-half-duplex-acceptance-*.json"))
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

func singleXiaozhiReportPath(t *testing.T, outputDir string, pattern string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(outputDir, pattern))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1 for %s: %v", len(matches), pattern, matches)
	}
	return matches[0]
}

func newestXiaozhiInstrumentObservationReport(t *testing.T, outputDir string, stdout string) string {
	t.Helper()
	basename := newestXiaozhiInstrumentObservationBasename(t, stdout)
	path := filepath.Join(outputDir, basename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), outputDir) {
		t.Fatalf("observation report leaked output dir: %s", string(data))
	}
	return string(data)
}

func newestXiaozhiInstrumentObservationBasename(t *testing.T, stdout string) string {
	t.Helper()
	const marker = `"report_path": "`
	idx := strings.Index(stdout, marker)
	if idx < 0 {
		t.Fatalf("stdout missing observation report_path: %s", stdout)
	}
	rest := stdout[idx+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatalf("stdout has invalid report_path: %s", stdout)
	}
	basename := rest[:end]
	if !strings.HasPrefix(basename, "a21-xiaozhi-instrument-observation-") || !strings.HasSuffix(basename, ".json") {
		t.Fatalf("observation report_path = %q, want basename", basename)
	}
	if strings.Contains(basename, "/") || strings.Contains(basename, "\\") {
		t.Fatalf("observation report_path leaked path: %q", basename)
	}
	return basename
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

func writeProductReadinessXiaozhiInstrumentedPhysicalEvidenceReportFixture(t *testing.T) string {
	t.Helper()
	reportPath := writeProductReadinessXiaozhiPhysicalEvidenceReportFixture(t)
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	report["promotion_gate"] = "candidate"
	report["acceptance_status"] = "physical_review_required"
	report["canonical_metrics"].(map[string]any)["device_downlink_first_frame_ms"] = map[string]any{"available": true, "value_ms": 310, "source": "gateway_trace"}
	report["canonical_metrics"].(map[string]any)["device_playback_start_ms"] = map[string]any{"available": true, "value_ms": 30, "source": "device_runtime_echo"}
	report["canonical_metrics"].(map[string]any)["speech_end_to_first_audible_response_ms"] = map[string]any{"available": true, "value_ms": 760, "source": "instrument_observation"}
	report["stage_availability"].(map[string]any)["device.playback.ack"] = map[string]any{"available": true, "source": "device_runtime_echo"}
	report["stage_availability"].(map[string]any)["operator.audible_observation"] = map[string]any{"available": true, "source": "instrument_observation"}
	report["observation"] = map[string]any{
		"available":               true,
		"physical_sound_observed": true,
		"operator_confirmed":      false,
		"method":                  "instrument_nonzero_audible_energy",
		"instrument":              "calibrated_audio_recorder",
		"observed_audible_ms":     760,
	}
	report["findings"] = []map[string]any{
		{"code": "xiaozhi_physical_instrument_observation_present", "severity": "info", "message": "instrument observed nonzero audible energy after Gateway first downlink"},
		{"code": "xiaozhi_physical_barge_in_stop_missing", "severity": "error", "message": "physical barge-in stop evidence is missing"},
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-xiaozhi-instrumented-physical-evidence-report.json")
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeProductReadinessDebugXiaozhiPhysicalEvidenceReportFixture(t *testing.T) string {
	t.Helper()
	reportPath := writeProductReadinessXiaozhiPhysicalEvidenceReportFixture(t)
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	report["profile"] = "debug"
	report["canonical_metrics"].(map[string]any)["device_playback_start_ms"] = map[string]any{"available": true, "value_ms": 31, "source": "device_runtime_echo"}
	stages := report["stage_availability"].(map[string]any)
	stages["xiaozhi.profile.stock"] = map[string]any{"available": false}
	stages["xiaozhi.profile.debug"] = map[string]any{"available": true, "source": "gateway_device_registry"}
	stages["device.playback.ack"] = map[string]any{"available": true, "source": "device_runtime_echo"}
	report["findings"] = []map[string]any{
		{"code": "xiaozhi_physical_gateway_downlink_candidate", "severity": "info", "message": "Xiaozhi physical device reached Gateway downlink, but this is not audible playback acceptance"},
		{"code": "xiaozhi_physical_operator_observation_missing", "severity": "error", "message": "missing operator audible observation or instrumented first audible playback evidence"},
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-xiaozhi-debug-physical-evidence-report.json")
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeProductReadinessAcceptedXiaozhiPhysicalEvidenceReportFixture(t *testing.T) string {
	t.Helper()
	reportPath := writeProductReadinessDebugXiaozhiPhysicalEvidenceReportFixture(t)
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	report["promotion_gate"] = "accepted"
	report["acceptance_status"] = "prd_accepted"
	report["prd_accepted"] = true

	metrics := report["canonical_metrics"].(map[string]any)
	metrics["device_downlink_first_frame_ms"] = map[string]any{"available": true, "value_ms": 392, "source": "gateway_trace"}
	metrics["device_playback_start_ms"] = map[string]any{"available": true, "value_ms": 30, "source": "device_runtime_echo"}
	metrics["speech_end_to_first_audible_response_ms"] = map[string]any{"available": true, "value_ms": 422, "source": "instrument_observation"}
	metrics["barge_in_detected_ms"] = map[string]any{"available": true, "value_ms": 0, "source": "gateway_trace"}
	metrics["barge_in_stop_ms"] = map[string]any{"available": true, "value_ms": 0, "source": "gateway_trace"}
	metrics["barge_in_playback_stop_requested_ms"] = map[string]any{"available": true, "value_ms": 0, "source": "gateway_trace"}
	metrics["barge_in_playback_stop_done_ms"] = map[string]any{"available": true, "value_ms": 108, "source": "device_runtime_echo"}

	stages := report["stage_availability"].(map[string]any)
	stages["device.playback.ack"] = map[string]any{"available": true, "source": "device_runtime_echo"}
	stages["device.playback.stop_done"] = map[string]any{"available": true, "source": "device_runtime_echo"}
	stages["operator.audible_observation"] = map[string]any{"available": true, "source": "instrument_observation"}
	stages["xiaozhi.profile.stock"] = map[string]any{"available": false}
	stages["xiaozhi.profile.debug"] = map[string]any{"available": true, "source": "gateway_device_registry"}

	report["observation"] = map[string]any{
		"available":               true,
		"physical_sound_observed": true,
		"operator_confirmed":      false,
		"method":                  "instrument_nonzero_audible_energy",
		"instrument":              "calibrated_audio_recorder",
		"observed_audible_ms":     422,
	}
	report["findings"] = []map[string]any{
		{"code": "xiaozhi_physical_prd_accepted", "severity": "info", "message": "Xiaozhi physical evidence satisfies PRD acceptance after consecutive physical rounds"},
	}

	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-xiaozhi-accepted-physical-evidence-report.json")
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
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
