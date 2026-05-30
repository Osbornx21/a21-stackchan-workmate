package audio

import (
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

func containsAll(value string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
