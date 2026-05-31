package audio

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMockFrontEndEvalReportsDeterministicRMSQuality(t *testing.T) {
	report := RunMockFrontEndEval()

	if report.Status != "mock_only" {
		t.Fatalf("Status = %q, want mock_only", report.Status)
	}
	if report.Dataset != "a21_mock_vad_fixture_v1" {
		t.Fatalf("Dataset = %q, want a21_mock_vad_fixture_v1", report.Dataset)
	}
	if report.Detector != "a21-rms-vad" {
		t.Fatalf("Detector = %q, want a21-rms-vad", report.Detector)
	}
	if report.FramesTotal != 5 || report.ExpectedSpeechFrames != 2 || report.DetectedSpeechFrames != 2 {
		t.Fatalf("frame counts = %+v", report)
	}
	if report.TruePositive != 2 || report.TrueNegative != 3 || report.FalsePositive != 0 || report.FalseNegative != 0 {
		t.Fatalf("confusion matrix = %+v", report)
	}
	if report.SpeechStartEvents != 1 || report.SpeechEndEvents != 1 {
		t.Fatalf("speech events = start %d end %d, want 1/1", report.SpeechStartEvents, report.SpeechEndEvents)
	}
	if report.SpeechStartLagMS == nil || *report.SpeechStartLagMS != 0 {
		t.Fatalf("SpeechStartLagMS = %v, want 0", report.SpeechStartLagMS)
	}
	if report.SpeechEndLagMS == nil || *report.SpeechEndLagMS != 20 {
		t.Fatalf("SpeechEndLagMS = %v, want 20", report.SpeechEndLagMS)
	}
	if report.PromotionGate != "not_production" {
		t.Fatalf("PromotionGate = %q, want not_production", report.PromotionGate)
	}
}

func TestBaselineFrontEndPlanExposesCandidateEvidenceShape(t *testing.T) {
	data, err := json.Marshal(BaselineFrontEndPlan())
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	rawCandidates, ok := report["candidates"].([]any)
	if !ok || len(rawCandidates) == 0 {
		t.Fatalf("candidates missing: %#v", report["candidates"])
	}
	candidates := map[string]map[string]any{}
	for _, raw := range rawCandidates {
		candidate, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("candidate shape = %#v", raw)
		}
		id, _ := candidate["id"].(string)
		candidates[id] = candidate
		for _, field := range []string{"status", "deployment_target", "available", "placeholder", "placeholder_reason", "required_evidence"} {
			if _, ok := candidate[field]; !ok {
				t.Fatalf("candidate %q missing %q: %#v", id, field, candidate)
			}
		}
	}
	for _, want := range []string{"webrtc_apm", "provider_side_vad", "silero_vad", "a21_rms_vad"} {
		if _, ok := candidates[want]; !ok {
			t.Fatalf("candidate list missing %q: %#v", want, candidates)
		}
	}
	if available, _ := candidates["webrtc_apm"]["available"].(bool); available {
		t.Fatalf("webrtc_apm must be unavailable until native adapter evidence exists: %#v", candidates["webrtc_apm"])
	}
	if placeholder, _ := candidates["webrtc_apm"]["placeholder"].(bool); !placeholder {
		t.Fatalf("webrtc_apm must be a placeholder: %#v", candidates["webrtc_apm"])
	}
	if available, _ := candidates["a21_rms_vad"]["available"].(bool); !available {
		t.Fatalf("a21_rms_vad baseline should be available as host-only development evidence: %#v", candidates["a21_rms_vad"])
	}
}

func TestRunMockFrontEndEvalExposesFastCompanionEvidenceContract(t *testing.T) {
	data, err := json.Marshal(RunMockFrontEndEval())
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"trace_id",
		"session_id",
		"device_id",
		"baseline_scope",
		"candidate_evidence",
		"fast_companion_evidence_contract",
		"redaction",
	} {
		if _, ok := report[want]; !ok {
			t.Fatalf("mock eval report missing %q: %#v", want, report)
		}
	}
	if report["baseline_scope"] != "host_only" || report["device_id"] != "none_host_fixture" {
		t.Fatalf("mock eval must stay host-only with no hardware identity: %#v", report)
	}
	candidates := collectFrontEndEvidenceByID(t, report["candidate_evidence"])
	for _, want := range []string{"webrtc_apm", "provider_side_vad", "silero_vad", "a21_rms_vad"} {
		if _, ok := candidates[want]; !ok {
			t.Fatalf("candidate_evidence missing %q: %#v", want, candidates)
		}
	}
	for _, id := range []string{"webrtc_apm", "provider_side_vad", "silero_vad"} {
		candidate := candidates[id]
		if available, _ := candidate["available"].(bool); available {
			t.Fatalf("%s must be unavailable in mock eval: %#v", id, candidate)
		}
		if placeholder, _ := candidate["placeholder"].(bool); !placeholder {
			t.Fatalf("%s must be placeholder in mock eval: %#v", id, candidate)
		}
	}
	contract := collectFrontEndEvidenceByID(t, report["fast_companion_evidence_contract"])
	for _, want := range []string{
		"mock_benchmark_preservation",
		"labelled_fixture_preservation",
		"office_noise_benchmark",
		"speaker_to_mic_echo_report",
		"speech_start_end_lag",
		"barge_in_stop_timing",
		"first_audio_waterfall_impact",
		"cpu_memory_profile",
		"metrics_continuity",
	} {
		if _, ok := contract[want]; !ok {
			t.Fatalf("fast companion evidence contract missing %q: %#v", want, contract)
		}
	}
}

func TestRunFrontEndEvalFromFixtureReportsFrameMetrics(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-front-end-fixture.json")
	data := `{
  "schema_version": "a21.audio.frontend_fixture.v1",
  "dataset": "a21_recorded_fixture_test",
  "detector": "a21-rms-vad",
  "sample_rate_hz": 16000,
  "channels": 1,
  "duration_ms": 20,
  "frames": [
    {"seq": 1, "expected_speech": false, "pcm_s16le_base64": "` + frontEndEvalPCM16Base64(0) + `"},
    {"seq": 2, "expected_speech": true, "pcm_s16le_base64": "` + frontEndEvalPCM16Base64(12000) + `"},
    {"seq": 3, "expected_speech": false, "pcm_s16le_base64": "` + frontEndEvalPCM16Base64(0) + `"},
    {"seq": 4, "expected_speech": false, "pcm_s16le_base64": "` + frontEndEvalPCM16Base64(0) + `"}
  ]
}`
	if err := os.WriteFile(fixture, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := RunFrontEndEvalFromFixture(fixture)
	if err != nil {
		t.Fatalf("RunFrontEndEvalFromFixture returned error: %v", err)
	}
	if report.Status != "fixture" || report.Dataset != "a21_recorded_fixture_test" || report.Detector != "a21-rms-vad" {
		t.Fatalf("report identity = %+v", report)
	}
	if report.FramesTotal != 4 || report.ExpectedSpeechFrames != 1 || report.DetectedSpeechFrames != 1 {
		t.Fatalf("frame counts = %+v", report)
	}
	if report.TruePositive != 1 || report.TrueNegative != 3 || report.FalsePositive != 0 || report.FalseNegative != 0 {
		t.Fatalf("confusion matrix = %+v", report)
	}
	if report.SpeechStartEvents != 1 || report.SpeechEndEvents != 1 {
		t.Fatalf("speech events = start %d end %d, want 1/1", report.SpeechStartEvents, report.SpeechEndEvents)
	}
	if report.SpeechStartLagMS == nil || *report.SpeechStartLagMS != 0 {
		t.Fatalf("SpeechStartLagMS = %v, want 0", report.SpeechStartLagMS)
	}
	if report.SpeechEndLagMS == nil || *report.SpeechEndLagMS != 20 {
		t.Fatalf("SpeechEndLagMS = %v, want 20", report.SpeechEndLagMS)
	}
	if report.PromotionGate != "requires_recorded_office_and_physical_acceptance" {
		t.Fatalf("PromotionGate = %q", report.PromotionGate)
	}
}

func TestRunFrontEndEvalFromFixtureRejectsWrongFrameSize(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-front-end-fixture.json")
	data := `{
  "schema_version": "a21.audio.frontend_fixture.v1",
  "dataset": "a21_bad_frame_fixture",
  "detector": "a21-rms-vad",
  "sample_rate_hz": 16000,
  "channels": 1,
  "duration_ms": 20,
  "frames": [
    {"seq": 1, "expected_speech": false, "pcm_s16le_base64": "AA=="}
  ]
}`
	if err := os.WriteFile(fixture, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := RunFrontEndEvalFromFixture(fixture)
	if err == nil {
		t.Fatal("expected wrong-sized PCM frame to be rejected")
	}
	if got := err.Error(); got == "" || !containsAll(got, []string{"frame", "bytes"}) {
		t.Fatalf("error = %q, want frame byte-size failure", got)
	}
}

func collectFrontEndEvidenceByID(t *testing.T, raw any) map[string]map[string]any {
	t.Helper()
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("evidence list missing: %#v", raw)
	}
	result := map[string]map[string]any{}
	for _, item := range items {
		fields, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("evidence item shape = %#v", item)
		}
		id, _ := fields["id"].(string)
		if id == "" {
			t.Fatalf("evidence item missing id: %#v", fields)
		}
		result[id] = fields
	}
	return result
}

func containsAll(value string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
