package app

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestXiaozhiStreamingProviderReadinessBlocksDefaultMock(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"xiaozhi-streaming-provider-readiness", "--output-dir", ""}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code=%d, want blocked stdout=%s", code, stdout.String())
	}
	for _, want := range []string{
		`"schema_version":"a21.xiaozhi_streaming_provider_readiness.v1"`,
		`"gate_status":"blocked"`,
		`"asr_mock_fixture_not_realtime"`,
		`"llm_mock_text_stream_not_product"`,
		`"tts_mock_fixture_not_realtime"`,
		`"prd_accepted":false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestXiaozhiStreamingProviderReadinessBlocksSherpaAndIflytekWAVBoundaries(t *testing.T) {
	t.Setenv("A21_ASR_LOCAL_PROFILE", "sherpa_onnx")
	t.Setenv("A21_TEXT_STREAM_PROFILE", "stepfun")
	t.Setenv("A21_TTS_FAST_PROFILE", "iflytek_tts")
	var stdout, stderr bytes.Buffer
	dir := t.TempDir()
	code := Run([]string{"xiaozhi-streaming-provider-readiness", "--output-dir", dir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code=%d, want blocked stdout=%s", code, stdout.String())
	}
	for _, want := range []string{
		`"gate_status":"blocked"`,
		`"profile":"sherpa_onnx"`,
		`"adapter":"local_sherpa_onnx_asr"`,
		`"asr_batch_wav_boundary_not_xiaozhi_streaming"`,
		`"adapter":"iflytek_tts_via_local_wav_adapter"`,
		`"tts_wav_file_boundary_not_xiaozhi_streaming"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"/Users/", "sk-", "Authorization", "Bearer", "raw_audio", "transcript"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked %q: %s", forbidden, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-xiaozhi-streaming-provider-readiness-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("matches=%v err=%v", matches, err)
	}
}

func TestXiaozhiStreamingProviderReadinessAcceptsOnlyAllStreamingFixture(t *testing.T) {
	t.Setenv("A21_ASR_LOCAL_PROFILE", "a21_fixture_streaming_asr")
	t.Setenv("A21_TEXT_STREAM_PROFILE", "a21_fixture_text_stream")
	t.Setenv("A21_TTS_FAST_PROFILE", "a21_fixture_streaming_tts")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"xiaozhi-streaming-provider-readiness", "--output-dir", ""}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	for _, want := range []string{
		`"gate_status":"passed"`,
		`"ready":true`,
		`"streaming":true`,
		`"next_required":[]`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}
