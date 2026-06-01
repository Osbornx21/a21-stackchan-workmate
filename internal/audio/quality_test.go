package audio

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnalyzePCM16LEQualityDetectsClippingAndLowHeadroom(t *testing.T) {
	pcm := pcm16Bytes(0, 1200, -1200, 32767, -32768, 29491)

	report, err := AnalyzePCM16LEQuality("pcm_s16le", 48000, 1, 60, pcm)
	if err != nil {
		t.Fatal(err)
	}

	if report.Status != "warning" {
		t.Fatalf("status = %q, want warning", report.Status)
	}
	if report.PeakAbs != 32768 || report.ClippedSamples != 2 {
		t.Fatalf("peak/clipped = %d/%d, want 32768/2", report.PeakAbs, report.ClippedSamples)
	}
	if report.ClipRatio <= 0 || report.RMSDBFS >= 0 {
		t.Fatalf("clip ratio/rms dbfs = %.6f/%.3f, want clipping and negative dbfs", report.ClipRatio, report.RMSDBFS)
	}
	for _, want := range []string{"audio_quality_clipping_detected", "audio_quality_low_headroom"} {
		if !stringSliceContainsAudioQuality(report.Findings, want) {
			t.Fatalf("findings missing %q: %+v", want, report.Findings)
		}
	}
	rendered := mustJSONAudioQuality(t, report)
	for _, forbidden := range []string{"data_base64", "pcm_s16le_base64", "raw_audio", "http://", "/Users/"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("quality report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestAnalyzePCM16LEQualityPassesQuietSpeechCandidate(t *testing.T) {
	report, err := AnalyzePCM16LEQuality("pcm_s16le", 24000, 1, 60, pcm16Bytes(0, 1600, -1600, 800, -800))
	if err != nil {
		t.Fatal(err)
	}

	if report.Status != "passed" {
		t.Fatalf("status = %q, want passed: %+v", report.Status, report.Findings)
	}
	if report.SampleRateHz != 24000 || report.Channels != 1 || report.DurationMS <= 0 {
		t.Fatalf("format/duration = %+v", report)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings = %+v, want none", report.Findings)
	}
}

func TestAnalyzePCM16MonoWAVQualityUsesAggregateOnly(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/a21-quality.wav"
	if err := WritePCM16MonoWAV(path, 16000, pcm16Bytes(0, 1000, -1000, 2000, -2000)); err != nil {
		t.Fatal(err)
	}

	report, err := AnalyzePCM16MonoWAVQuality(path, 16000)
	if err != nil {
		t.Fatal(err)
	}

	if report.Status != "passed" || report.Codec != "pcm_s16le" || report.SampleRateHz != 16000 {
		t.Fatalf("report = %+v", report)
	}
	rendered := mustJSONAudioQuality(t, report)
	if strings.Contains(rendered, path) || strings.Contains(rendered, dir) {
		t.Fatalf("quality report leaked local path: %s", rendered)
	}
}

func pcm16Bytes(samples ...int16) []byte {
	data := make([]byte, len(samples)*2)
	for i, sample := range samples {
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return data
}

func stringSliceContainsAudioQuality(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func mustJSONAudioQuality(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
