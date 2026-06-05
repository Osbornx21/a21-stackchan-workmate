package app

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunXiaozhiPhysicalEvidenceAcceptsGatewayTraceMarkers(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false)
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
		`"value_ms": 110`,
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

func TestRunXiaozhiInstrumentObservationWritesRedactedReportForPhysicalEvidence(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false)
	dir := t.TempDir()
	var observationStdout bytes.Buffer
	var observationStderr bytes.Buffer

	code := RunLab([]string{
		"xiaozhi-instrument-observation",
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--gateway-answer-first-downlink-ms", "110",
		"--gateway-first-downlink-to-audible-ms", "650",
		"--audible-energy-rms", "0.09",
		"--noise-floor-rms", "0.01",
		"--device-playback-observed",
		"--gateway-first-downlink-to-device-playback-start-ms", "30",
		"--device-playback-observation-source", "device_runtime_echo",
		"--output-dir", dir,
	}, &observationStdout, &observationStderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, observationStdout.String(), observationStderr.String())
	}
	observationReport := newestXiaozhiInstrumentObservationReport(t, dir, observationStdout.String())
	for _, want := range []string{
		`"schema_version": "a21.xiaozhi_instrument_observation.v1"`,
		`"method": "instrument_nonzero_audible_energy"`,
		`"instrument": "calibrated_audio_recorder"`,
		`"gateway_first_downlink_to_audible_ms": 650`,
		`"speech_end_to_first_audible_response_ms": 760`,
		`"device_playback_observed": true`,
		`"gateway_first_downlink_to_device_playback_start_ms": 30`,
		`"audio_payload_stored": false`,
		`"encoded_audio_payload_stored": false`,
		`"report_path": "a21-xiaozhi-instrument-observation-`,
	} {
		if !strings.Contains(observationStdout.String(), want) || !strings.Contains(observationReport, want) {
			t.Fatalf("observation report missing %q: stdout=%s report=%s", want, observationStdout.String(), observationReport)
		}
	}
	for _, forbidden := range []string{dir, server.URL, "transcript", "prompt", "provider output", "raw_audio", "data_base64", "secret-token", "/Users/"} {
		if strings.Contains(observationStdout.String(), forbidden) || strings.Contains(observationReport, forbidden) {
			t.Fatalf("observation report leaked %q: stdout=%s report=%s", forbidden, observationStdout.String(), observationReport)
		}
	}

	var physicalStdout bytes.Buffer
	var physicalStderr bytes.Buffer
	code = RunLab([]string{
		"xiaozhi-physical-evidence",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--instrument-observation-report", filepath.Join(dir, newestXiaozhiInstrumentObservationBasename(t, observationStdout.String())),
		"--output-dir", dir,
	}, &physicalStdout, &physicalStderr)

	if code != 0 {
		t.Fatalf("physical evidence code = %d, want 0: stdout=%s stderr=%s", code, physicalStdout.String(), physicalStderr.String())
	}
	for _, want := range []string{
		`"operator.audible_observation": {`,
		`"available": true`,
		`"source": "instrument_observation"`,
		`"speech_end_to_first_audible_response_ms": {`,
		`"device_playback_start_ms": {`,
		`"source": "trusted_runtime_observation"`,
		`"acceptance_status": "physical_review_required"`,
		`"code": "xiaozhi_physical_instrument_observation_present"`,
	} {
		if !strings.Contains(physicalStdout.String(), want) {
			t.Fatalf("physical report missing %q: %s", want, physicalStdout.String())
		}
	}
	if strings.Contains(physicalStdout.String(), `"code": "xiaozhi_physical_operator_observation_missing"`) {
		t.Fatalf("physical report still missing operator observation: %s", physicalStdout.String())
	}
}

func TestRunXiaozhiPhysicalEvidenceRejectsUnsafeGatewayValuesWithoutLeak(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, true)
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

func TestRunXiaozhiPhysicalEvidenceAcceptsProductRuntimeSessionDrift(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true, false, false, false, false, false, true)
	dir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{
		"xiaozhi-physical-evidence",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-speaking-barge-test",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	reportJSON := newestXiaozhiPhysicalEvidenceReport(t, dir, stdout.String())
	for _, want := range []string{
		`"session_id": "a21-session-speaking-barge-test"`,
		`"device.playback.ack": {`,
		`"device_playback_start_ms": {`,
		`"source": "device_runtime_echo"`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
}

func TestRunXiaozhiPhysicalEvidenceRejectsTargetMismatches(t *testing.T) {
	for _, tt := range []struct {
		name      string
		deviceID  string
		traceID   string
		sessionID string
		playback  []bool
	}{
		{
			name:      "stale device trace",
			deviceID:  "44:1b:f6:e2:6a:60",
			traceID:   "a21-trace-stale-target",
			sessionID: "a21-session-44-1b-f6-e2-6a-60",
		},
		{
			name:      "stale session",
			deviceID:  "44:1b:f6:e2:6a:60",
			traceID:   "a21-trace-44-1b-f6-e2-6a-60",
			sessionID: "a21-session-stale-target",
		},
		{
			name:      "stale device id",
			deviceID:  "44:1b:f6:e2:6a:60",
			traceID:   "a21-trace-44-1b-f6-e2-6a-60",
			sessionID: "a21-session-44-1b-f6-e2-6a-60",
			playback:  []bool{false, false, false, false, false, true},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := newXiaozhiPhysicalEvidenceTestServer(t, false, tt.playback...)
			dir := t.TempDir()
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := RunLab([]string{
				"xiaozhi-physical-evidence",
				"--gateway-url", server.URL,
				"--device-id", tt.deviceID,
				"--trace-id", tt.traceID,
				"--session-id", tt.sessionID,
				"--output-dir", dir,
			}, &stdout, &stderr)

			if code != 1 {
				t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			for _, want := range []string{"target mismatch"} {
				if !strings.Contains(stderr.String(), want) {
					t.Fatalf("stderr missing %q: %s", want, stderr.String())
				}
			}
			for _, forbidden := range []string{
				server.URL,
				"http://",
				"https://",
				"data_base64",
				"raw_audio",
				"transcript",
				`"schema_version": "a21.xiaozhi_physical_evidence.v1"`,
			} {
				if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
					t.Fatalf("target mismatch leaked or wrote report %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
				}
			}
			matches, err := filepath.Glob(filepath.Join(dir, "a21-xiaozhi-physical-evidence-*.json"))
			if err != nil {
				t.Fatal(err)
			}
			if len(matches) != 0 {
				t.Fatalf("reports = %v, want none for target mismatch", matches)
			}
		})
	}
}

func TestRunXiaozhiPhysicalEvidenceConsumesValidInstrumentObservation(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true)
	dir := t.TempDir()
	observation := writeXiaozhiInstrumentObservationReport(t, map[string]any{})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{
		"xiaozhi-physical-evidence",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--instrument-observation-report", observation,
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	reportJSON := newestXiaozhiPhysicalEvidenceReport(t, dir, stdout.String())
	for _, want := range []string{
		`"device.playback.ack": {`,
		`"operator.audible_observation": {`,
		`"device_playback_start_ms": {`,
		`"value_ms": 30`,
		`"speech_end_to_first_audible_response_ms": {`,
		`"value_ms": 760`,
		`"source": "instrument_observation"`,
		`"physical_sound_observed": true`,
		`"method": "instrument_nonzero_audible_energy"`,
		`"instrument": "calibrated_audio_recorder"`,
		`"observed_audible_ms": 760`,
		`"promotion_gate": "candidate"`,
		`"acceptance_status": "physical_review_required"`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
	for _, forbidden := range []string{
		observation,
		filepath.Dir(observation),
		`"code": "xiaozhi_physical_device_playback_ack_missing"`,
		`"code": "xiaozhi_physical_operator_observation_missing"`,
		"data_base64",
		"raw_audio",
		"transcript",
		"provider output",
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report leaked or overclaimed %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func TestRunXiaozhiPhysicalEvidenceConsumesInstrumentPlaybackRuntimeEcho(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false)
	dir := t.TempDir()
	observation := writeXiaozhiInstrumentObservationReport(t, map[string]any{
		"device_playback_observed":                           true,
		"device_playback_observation_source":                 "serial_state_machine_echo",
		"gateway_first_downlink_to_device_playback_start_ms": 120,
		"gateway_first_downlink_to_audible_ms":               650,
		"speech_end_to_first_audible_response_ms":            760,
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{
		"xiaozhi-physical-evidence",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--instrument-observation-report", observation,
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	reportJSON := newestXiaozhiPhysicalEvidenceReport(t, dir, stdout.String())
	for _, want := range []string{
		`"device.playback.ack": {`,
		`"device_playback_start_ms": {`,
		`"value_ms": 120`,
		`"source": "trusted_runtime_observation"`,
		`"operator.audible_observation": {`,
		`"promotion_gate": "candidate"`,
		`"acceptance_status": "physical_review_required"`,
		`"prd_accepted": false`,
		`"code": "xiaozhi_physical_barge_in_stop_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
	if strings.Contains(reportJSON, `"device.playback.ack": {
      "available": true,
      "source": "device_runtime_echo"`) {
		t.Fatalf("instrument playback ack should not be labeled as stock debug runtime echo: stdout=%s report=%s", stdout.String(), reportJSON)
	}
	for _, forbidden := range []string{
		observation,
		filepath.Dir(observation),
		`"code": "xiaozhi_physical_device_playback_ack_missing"`,
		`"code": "xiaozhi_physical_operator_observation_missing"`,
		"transcript",
		"raw_audio",
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report leaked or overclaimed %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func TestRunXiaozhiPhysicalEvidenceMapsGatewayBargeInMarkers(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, false, true)
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
		`"barge_in_detected_ms": {`,
		`"barge_in_stop_ms": {`,
		`"value_ms": 65`,
		`"barge_in_playback_stop_requested_ms": {`,
		`"source": "gateway_trace"`,
		`"barge_in_playback_stop_done_ms": {`,
		`"available": false`,
		`"code": "xiaozhi_physical_device_playback_ack_missing"`,
		`"code": "xiaozhi_physical_operator_observation_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
}

func TestRunXiaozhiPhysicalEvidenceMapsDebugPlaybackStopDone(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true, true, true)
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
		`"barge_in_playback_stop_done_ms": {`,
		`"value_ms": 120`,
		`"source": "device_runtime_echo"`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
}

func TestRunXiaozhiHalfDuplexAcceptanceUsesStockTraceEvidence(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true, true, true)
	dir := t.TempDir()
	observation := writeXiaozhiInstrumentObservationReport(t, map[string]any{
		"device_playback_observed":                           true,
		"device_playback_observation_source":                 "device_runtime_echo",
		"gateway_first_downlink_to_device_playback_start_ms": 30,
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-accept",
		"--check", "xiaozhi-half-duplex",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--instrument-observation-report", observation,
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	reportJSON := newestXiaozhiHalfDuplexAcceptanceReport(t, dir, stdout.String())
	for _, want := range []string{
		`"schema_version": "a21.xiaozhi_half_duplex_acceptance.v1"`,
		`"hardware_acceptance_scope": "stock_xiaozhi_mic_to_tts_downlink"`,
		`"half_duplex_acceptance_status": "physical_review_required"`,
		`"diagnostic_mic_probe_required": false`,
		`"physical_device_online": true`,
		`"audio_frame_count": 2`,
		`"mic_available": true`,
		`"downlink_available": true`,
		`"playback_ack_available": true`,
		`"barge_in_stop_available": true`,
		`"prd_accepted": false`,
		`"source": "xiaozhi_physical_evidence"`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
	if !strings.Contains(stdout.String(), "stackchan stock Xiaozhi half-duplex acceptance needs physical review") {
		t.Fatalf("stdout missing physical review line: %s", stdout.String())
	}
	for _, forbidden := range []string{
		observation,
		filepath.Dir(observation),
		server.URL,
		"diagnostic_probe_m5unified_i2s_capture",
		"microphone_not_diagnostic_probe",
		"secret-token",
		"transcript",
		"data_base64",
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report leaked or used diagnostic-only requirement %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func TestRunXiaozhiHalfDuplexAcceptanceDerivesLatestDeviceTrace(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true, true, true)
	dir := t.TempDir()
	observation := writeXiaozhiInstrumentObservationReport(t, map[string]any{
		"device_playback_observed":                           true,
		"device_playback_observation_source":                 "device_runtime_echo",
		"gateway_first_downlink_to_device_playback_start_ms": 30,
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-accept",
		"--check", "xiaozhi-half-duplex",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--instrument-observation-report", observation,
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"trace_id": "a21-trace-44-1b-f6-e2-6a-60"`,
		`"session_id": "a21-session-44-1b-f6-e2-6a-60"`,
		`"half_duplex_acceptance_status": "physical_review_required"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunXiaozhiPhysicalEvidenceIgnoresInvalidInstrumentPlaybackRuntimeEcho(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false)
	dir := t.TempDir()
	observation := writeXiaozhiInstrumentObservationReport(t, map[string]any{
		"device_playback_observed":                           true,
		"device_playback_observation_source":                 "serial_state_machine_echo",
		"gateway_first_downlink_to_device_playback_start_ms": 900,
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{
		"xiaozhi-physical-evidence",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--instrument-observation-report", observation,
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want invalid playback echo ignored with blocked report: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"device.playback.ack": {`,
		`"available": false`,
		`"code": "xiaozhi_physical_instrument_playback_timing_invalid"`,
		`"code": "xiaozhi_physical_device_playback_ack_missing"`,
		`"promotion_gate": "not_production"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{
		observation,
		filepath.Dir(observation),
		`"source": "trusted_runtime_observation"`,
		`"promotion_gate": "candidate"`,
	} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked or overclaimed %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunXiaozhiPhysicalEvidenceIgnoresMismatchedInstrumentObservation(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true)
	dir := t.TempDir()
	observation := writeXiaozhiInstrumentObservationReport(t, map[string]any{
		"trace_id": "a21-trace-other",
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{
		"xiaozhi-physical-evidence",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--instrument-observation-report", observation,
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want mismatch ignored with blocked report: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"operator.audible_observation": {`,
		`"available": false`,
		`"code": "xiaozhi_physical_instrument_observation_mismatch"`,
		`"code": "xiaozhi_physical_operator_observation_missing"`,
		`"acceptance_status": "candidate_gateway_downlink"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), observation) || strings.Contains(stdout.String(), filepath.Dir(observation)) {
		t.Fatalf("stdout leaked observation path: %s", stdout.String())
	}
}

func TestRunXiaozhiPhysicalEvidenceIgnoresInconsistentInstrumentTiming(t *testing.T) {
	cases := []struct {
		name      string
		overrides map[string]any
	}{
		{name: "inconsistent delta", overrides: map[string]any{"gateway_first_downlink_to_audible_ms": 12}},
		{name: "negative timing", overrides: map[string]any{"gateway_first_downlink_to_audible_ms": -1}},
		{name: "implausible timing", overrides: map[string]any{"speech_end_to_first_audible_response_ms": 120000}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := newXiaozhiPhysicalEvidenceTestServer(t, false, true)
			dir := t.TempDir()
			observation := writeXiaozhiInstrumentObservationReport(t, tc.overrides)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := RunLab([]string{
				"xiaozhi-physical-evidence",
				"--gateway-url", server.URL,
				"--device-id", "44:1b:f6:e2:6a:60",
				"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
				"--session-id", "a21-session-44-1b-f6-e2-6a-60",
				"--instrument-observation-report", observation,
				"--output-dir", dir,
			}, &stdout, &stderr)

			if code != 0 {
				t.Fatalf("code = %d, want invalid timing ignored with blocked report: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			for _, want := range []string{
				`"operator.audible_observation": {`,
				`"available": false`,
				`"code": "xiaozhi_physical_instrument_observation_timing_invalid"`,
				`"code": "xiaozhi_physical_operator_observation_missing"`,
				`"acceptance_status": "candidate_gateway_downlink"`,
			} {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout missing %q: %s", want, stdout.String())
				}
			}
			for _, forbidden := range []string{
				observation,
				filepath.Dir(observation),
				`"source": "instrument_observation"`,
				`"promotion_gate": "candidate"`,
			} {
				if strings.Contains(stdout.String(), forbidden) {
					t.Fatalf("stdout leaked or overclaimed %q: %s", forbidden, stdout.String())
				}
			}
		})
	}
}

func TestRunXiaozhiPhysicalEvidenceRejectsUnsafeInstrumentObservationWithoutLeak(t *testing.T) {
	server := newXiaozhiPhysicalEvidenceTestServer(t, false, true)
	dir := t.TempDir()
	observation := writeXiaozhiInstrumentObservationReport(t, map[string]any{
		"local_path":  filepath.Join(dir, "unsafe.wav"),
		"data_base64": "U0VDUkVUX0FVRElP",
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{
		"xiaozhi-physical-evidence",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--trace-id", "a21-trace-44-1b-f6-e2-6a-60",
		"--session-id", "a21-session-44-1b-f6-e2-6a-60",
		"--instrument-observation-report", observation,
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("code = 0, want unsafe instrument observation rejected: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	rendered := stdout.String() + stderr.String()
	if !strings.Contains(rendered, "xiaozhi physical evidence instrument observation unsafe") {
		t.Fatalf("missing safe error: %s", rendered)
	}
	for _, forbidden := range []string{
		observation,
		filepath.Dir(observation),
		"unsafe.wav",
		"U0VDUkVUX0FVRElP",
		server.URL,
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("unsafe observation leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
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

func TestProductReadinessIngestsInstrumentedXiaozhiEvidenceAsBlockedCandidate(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"44:1b:f6:e2:6a:60","identity_status":"unknown","connection_status":"online","capabilities":{"xiaozhi_profile":"stock","xiaozhi_transport":"websocket","xiaozhi_audio":"opus_16000hz_mono_60ms","microphone":"available_stock_xiaozhi"},"first_seen_ms":1,"last_seen_ms":2}]}`)
	evidence := writeProductReadinessXiaozhiInstrumentedPhysicalEvidenceReportFixture(t)

	report := buildProductReadinessReport(t.Context(), productReadinessOptions{
		GatewayURL:              server.URL,
		DeviceID:                "44:1b:f6:e2:6a:60",
		PhysicalStackChanReport: evidence,
	}, []string{"A21_PROVIDER_PRIMARY=mock"})

	physical := report.StackChan.PhysicalEvidence
	if report.LaunchReady || physical.PRDAccepted || physical.PRDPhysicalAccepted {
		t.Fatalf("launch/physical = %v/%+v, want instrumented xiaozhi candidate not PRD accepted", report.LaunchReady, physical)
	}
	if !physical.Valid ||
		!physical.CandidatePhysicalVoiceEvidence ||
		!physical.GatewayDownlinkPhysicalDeviceEvidence ||
		physical.RequiredPhysicalMetricsAvailable ||
		!physical.OperatorInstrumentObservationAvailable ||
		physical.AcceptanceStatus != "physical_review_required" ||
		physical.PromotionGate != "candidate" {
		t.Fatalf("physical evidence = %+v, want blocked instrumented physical voice candidate", physical)
	}
	if !physical.CanonicalMetricAvailability["device_playback_start_ms"] ||
		!physical.CanonicalMetricAvailability["speech_end_to_first_audible_response_ms"] ||
		physical.CanonicalMetricAvailability["barge_in_stop_ms"] {
		t.Fatalf("canonical availability = %#v, want playback/audible present but barge-in missing", physical.CanonicalMetricAvailability)
	}
	for _, code := range []string{
		"xiaozhi_physical_device_playback_ack_missing",
		"xiaozhi_physical_operator_observation_missing",
	} {
		if containsProductFinding(report.Findings, code, "") || containsXiaozhiPhysicalString(physical.FindingCodes, code) {
			t.Fatalf("unexpected missing-evidence code %q: findings=%#v physical=%+v", code, report.Findings, physical)
		}
	}
}

func TestProductReadinessIngestsDebugXiaozhiPhysicalEvidenceAsCandidate(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"44:1b:f6:e2:6a:60","identity_status":"unknown","connection_status":"online","capabilities":{"xiaozhi_profile":"debug","xiaozhi_feature_device_events":"true","xiaozhi_debug_extension_isolated":"true","xiaozhi_transport":"websocket","xiaozhi_audio":"opus_16000hz_mono_60ms","microphone":"available_xiaozhi_opus_ingress"},"first_seen_ms":1,"last_seen_ms":2}]}`)
	evidence := writeProductReadinessDebugXiaozhiPhysicalEvidenceReportFixture(t)

	report := buildProductReadinessReport(t.Context(), productReadinessOptions{
		GatewayURL:              server.URL,
		DeviceID:                "44:1b:f6:e2:6a:60",
		PhysicalStackChanReport: evidence,
	}, []string{"A21_PROVIDER_PRIMARY=mock"})

	physical := report.StackChan.PhysicalEvidence
	if report.LaunchReady || physical.PRDAccepted || physical.PRDPhysicalAccepted {
		t.Fatalf("launch/physical = %v/%+v, want debug xiaozhi candidate not accepted", report.LaunchReady, physical)
	}
	if !physical.Valid ||
		!physical.CandidatePhysicalVoiceEvidence ||
		!physical.GatewayDownlinkPhysicalDeviceEvidence ||
		physical.CandidatePhysicalEvidence ||
		physical.HostLoopbackOnly ||
		physical.AcceptanceStatus != "candidate_gateway_downlink" ||
		physical.PromotionGate != "not_production" {
		t.Fatalf("physical evidence = %+v, want debug xiaozhi gateway-downlink candidate", physical)
	}
	if !physical.CanonicalMetricAvailability["device_playback_start_ms"] ||
		physical.CanonicalMetricAvailability["barge_in_playback_stop_done_ms"] {
		t.Fatalf("canonical availability = %#v, want debug playback start but no false stop_done", physical.CanonicalMetricAvailability)
	}
	if containsXiaozhiPhysicalString(physical.FindingCodes, "xiaozhi_physical_device_playback_ack_missing") {
		t.Fatalf("finding codes = %#v, want no playback ack missing when device_playback_start_ms is available", physical.FindingCodes)
	}
	if containsProductAction(report.NextActions, "stock Xiaozhi") {
		t.Fatalf("next actions = %#v, want profile-neutral Xiaozhi action", report.NextActions)
	}
	if containsProductAction(report.NextActions, "device playback ack") {
		t.Fatalf("next actions = %#v, want no missing playback ack once device_playback_start_ms is available", report.NextActions)
	}
	for _, want := range []string{
		"operator audible observation",
		"barge-in playback stop_done",
	} {
		if !containsProductAction(report.NextActions, want) {
			t.Fatalf("next actions = %#v, want %q", report.NextActions, want)
		}
	}
}
