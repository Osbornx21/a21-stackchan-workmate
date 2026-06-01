package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPhysicalStackChanEvidenceCompleteFixtureEmitsRedactedContract(t *testing.T) {
	dir := t.TempDir()
	fixturePath := writeTestPhysicalStackChanEvidenceFixture(t, dir, map[string]any{})
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"physical-stackchan-evidence", "--fixture", fixturePath, "--output-dir", outputDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	reportJSON := newestPhysicalStackChanEvidenceReport(t, outputDir, stdout.String())
	for _, want := range []string{
		`"schema_version": "a21.physical_stackchan_evidence.v1"`,
		`"execution_mode": "physical_stackchan"`,
		`"trace_id": "a21-trace-physical-stackchan-001"`,
		`"session_id": "a21-session-physical-stackchan-001"`,
		`"device_id": "stackchan-a21-001"`,
		`"fixture_path": "physical-stackchan-fixture.json"`,
		`"device_downlink_first_frame_ms": {`,
		`"device_playback_start_ms": {`,
		`"speech_end_to_first_audible_response_ms": {`,
		`"barge_in_stop_ms": {`,
		`"available": true`,
		`"physical_sound_observed": true`,
		`"operator_confirmed": true`,
		`"promotion_gate": "candidate"`,
		`"acceptance_status": "physical_review_required"`,
		`"prd_accepted": false`,
		`"code": "physical_review_required"`,
	} {
		if !strings.Contains(stdout.String(), want) || !strings.Contains(reportJSON, want) {
			t.Fatalf("physical report missing %q: stdout=%s report=%s", want, stdout.String(), reportJSON)
		}
	}
	for _, forbidden := range []string{
		dir,
		"transcript",
		"prompt",
		"provider output",
		"raw_audio",
		"data_base64",
		"http://example.com/unsafe/full/url",
		"secret-token",
		"proxy.local:7890",
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("physical report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func TestRunPhysicalStackChanEvidenceMissingCoreCategoriesFindings(t *testing.T) {
	cases := []struct {
		name     string
		override map[string]any
		code     string
	}{
		{name: "downlink receipt", override: map[string]any{"device_downlink_first_frame_ms": nil}, code: "physical_downlink_receipt_missing"},
		{name: "playback start", override: map[string]any{"device_playback_start_ms": nil}, code: "physical_playback_start_missing"},
		{name: "audible response", override: map[string]any{"speech_end_to_first_audible_response_ms": nil}, code: "physical_audible_response_missing"},
		{name: "barge in stop", override: map[string]any{"barge_in_stop_ms": nil}, code: "physical_barge_in_stop_missing"},
		{name: "mic counters rms", override: map[string]any{"mic": map[string]any{"frames_captured": 0}}, code: "physical_mic_counters_rms_missing"},
		{name: "operator instrument observation", override: map[string]any{"observation": map[string]any{"operator_confirmed": false}}, code: "physical_operator_instrument_observation_missing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			fixturePath := writeTestPhysicalStackChanEvidenceFixture(t, dir, tc.override)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run([]string{"physical-stackchan-evidence", "--fixture", fixturePath, "--output-dir", filepath.Join(dir, "reports")}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), `"code": "`+tc.code+`"`) {
				t.Fatalf("stdout missing finding %q: %s", tc.code, stdout.String())
			}
			if !strings.Contains(stdout.String(), `"promotion_gate": "not_production"`) || !strings.Contains(stdout.String(), `"acceptance_status": "blocked"`) {
				t.Fatalf("missing fixture should stay blocked: %s", stdout.String())
			}
		})
	}
}

func TestRunPhysicalStackChanEvidenceHostLoopbackStaysCandidateHostOnly(t *testing.T) {
	dir := t.TempDir()
	fixturePath := writeTestPhysicalStackChanEvidenceFixture(t, dir, map[string]any{
		"execution_mode": "host_loopback",
		"device_id":      "none_host_fixture",
		"execution": map[string]any{
			"provider_executed": false,
			"v21_executed":      false,
			"hardware_executed": false,
		},
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"physical-stackchan-evidence", "--fixture", fixturePath, "--output-dir", filepath.Join(dir, "reports")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"execution_mode": "host_loopback"`,
		`"device_id": "none_host_fixture"`,
		`"promotion_gate": "not_production"`,
		`"acceptance_status": "candidate_host_only"`,
		`"prd_accepted": false`,
		`"code": "host_loopback_only"`,
		`"available": false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunPhysicalStackChanEvidenceUnsafeFixtureIsRedacted(t *testing.T) {
	dir := t.TempDir()
	fixturePath := writeTestPhysicalStackChanEvidenceFixture(t, dir, map[string]any{
		"transcript":      "please leak this transcript",
		"prompt":          "please leak this prompt",
		"provider_output": "provider output should stay out",
		"raw_audio":       "raw_audio_bytes",
		"data_base64":     "U0VDUkVUX0FVRElP",
		"url":             "http://example.com/unsafe/full/url",
		"proxy":           "http://user:secret-token@proxy.local:7890",
		"local_path":      filepath.Join(dir, "secret.wav"),
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"physical-stackchan-evidence", "--fixture", fixturePath, "--output-dir", filepath.Join(dir, "reports")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0 redacted report: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	reportJSON := newestPhysicalStackChanEvidenceReport(t, filepath.Join(dir, "reports"), stdout.String())
	if !strings.Contains(stdout.String(), `"code": "unsafe_fixture_content_redacted"`) || !strings.Contains(reportJSON, `"code": "unsafe_fixture_content_redacted"`) {
		t.Fatalf("unsafe finding missing: stdout=%s report=%s", stdout.String(), reportJSON)
	}
	for _, forbidden := range []string{
		"please leak this transcript",
		"please leak this prompt",
		"provider output should stay out",
		"raw_audio_bytes",
		"U0VDUkVUX0FVRElP",
		"http://example.com/unsafe/full/url",
		"secret-token",
		"proxy.local:7890",
		dir,
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("unsafe fixture leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func writeTestPhysicalStackChanEvidenceFixture(t *testing.T, dir string, overrides map[string]any) string {
	t.Helper()
	fixture := map[string]any{
		"schema_version": "a21.physical_stackchan_fixture.v1",
		"execution_mode": "physical_stackchan",
		"trace_id":       "a21-trace-physical-stackchan-001",
		"session_id":     "a21-session-physical-stackchan-001",
		"device_id":      "stackchan-a21-001",
		"execution": map[string]any{
			"provider_executed": false,
			"v21_executed":      false,
			"hardware_executed": true,
		},
		"device_downlink_first_frame_ms":          430,
		"device_playback_start_ms":                520,
		"speech_end_to_first_audible_response_ms": 760,
		"barge_in_detected_ms":                    50,
		"barge_in_stop_ms":                        130,
		"barge_in_playback_stop_requested_ms":     90,
		"barge_in_playback_stop_done_ms":          130,
		"mic": map[string]any{
			"frames_captured":      320,
			"frames_delivered":     318,
			"rms":                  0.13,
			"delivery_ratio":       0.99375,
			"driver_error_count":   0,
			"queue_drop_count":     0,
			"nonzero_sample_count": 4096,
		},
		"observation": map[string]any{
			"physical_sound_observed": true,
			"operator_confirmed":      true,
			"method":                  "operator_and_instrument",
			"instrument":              "calibrated_audio_recorder",
			"observed_audible_ms":     760,
			"observed_stop_ms":        130,
		},
	}
	for key, value := range overrides {
		if value == nil {
			delete(fixture, key)
			continue
		}
		fixture[key] = value
	}
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "physical-stackchan-fixture.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func newestPhysicalStackChanEvidenceReport(t *testing.T, outputDir string, stdout string) string {
	t.Helper()
	if !strings.Contains(stdout, `"report_path": "a21-physical-stackchan-evidence-`) {
		t.Fatalf("stdout missing basename report path: %s", stdout)
	}
	if strings.Contains(stdout, outputDir) {
		t.Fatalf("stdout leaked output dir: %s", stdout)
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-physical-stackchan-evidence-*.json"))
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
