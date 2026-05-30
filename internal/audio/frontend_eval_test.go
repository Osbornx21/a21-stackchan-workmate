package audio

import "testing"

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
	if report.PromotionGate != "not_production" {
		t.Fatalf("PromotionGate = %q, want not_production", report.PromotionGate)
	}
}
