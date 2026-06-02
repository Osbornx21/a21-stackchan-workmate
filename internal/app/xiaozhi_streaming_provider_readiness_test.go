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

func TestXiaozhiStreamingProviderReadinessBlocksSherpaStreamingWhenHelperMissing(t *testing.T) {
	t.Setenv("A21_ASR_LOCAL_PROFILE", "sherpa_onnx_streaming")
	t.Setenv("A21_TEXT_STREAM_PROFILE", "stepfun")
	t.Setenv("A21_TTS_FAST_PROFILE", "iflytek_tts")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"xiaozhi-streaming-provider-readiness", "--output-dir", ""}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code=%d, want blocked stdout=%s", code, stdout.String())
	}
	for _, want := range []string{
		`"gate_status":"blocked"`,
		`"profile":"sherpa_onnx_streaming"`,
		`"adapter":"local_sherpa_onnx_streaming_asr"`,
		`"streaming":true`,
		`"uses_file_boundary":false`,
		`"uses_wav_boundary":false`,
		`"asr_sherpa_streaming_helper_or_model_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "asr_batch_wav_boundary_not_xiaozhi_streaming") {
		t.Fatalf("streaming sherpa profile was misclassified as batch WAV: %s", stdout.String())
	}
}

func TestXiaozhiStreamingProviderReadinessAcceptsConfiguredSherpaStreamingASRStageOnly(t *testing.T) {
	t.Setenv("A21_ASR_LOCAL_PROFILE", "sherpa_onnx_streaming")
	t.Setenv("A21_SHERPA_ONNX_STREAMING_HELPER", "/redacted/a21-sherpa-streaming-helper")
	t.Setenv("A21_SHERPA_ONNX_ASR_MODEL_DIR", "/redacted/a21-sherpa-model")
	t.Setenv("A21_TEXT_STREAM_PROFILE", "stepfun")
	t.Setenv("A21_TTS_FAST_PROFILE", "iflytek_tts")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"xiaozhi-streaming-provider-readiness", "--output-dir", ""}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code=%d, want blocked by TTS stdout=%s", code, stdout.String())
	}
	for _, want := range []string{
		`"adapter":"local_sherpa_onnx_streaming_asr"`,
		`"ready":true`,
		`"real_provider":true`,
		`"streaming":true`,
		`"tts_wav_file_boundary_not_xiaozhi_streaming"`,
		`"gate_status":"blocked"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"/redacted/", "/Users/", "a21-sherpa-streaming-helper", "a21-sherpa-model"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestXiaozhiStreamingProviderReadinessBlocksDoubaoRealtimeTTSWhenConfigMissing(t *testing.T) {
	t.Setenv("A21_ASR_LOCAL_PROFILE", "sherpa_onnx_streaming")
	t.Setenv("A21_TEXT_STREAM_PROFILE", "stepfun")
	t.Setenv("A21_TTS_FAST_PROFILE", "doubao_tts_realtime")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"xiaozhi-streaming-provider-readiness", "--output-dir", ""}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code=%d, want blocked stdout=%s", code, stdout.String())
	}
	for _, want := range []string{
		`"gate_status":"blocked"`,
		`"profile":"doubao_tts_realtime"`,
		`"adapter":"doubao_realtime_tts_adapter"`,
		`"streaming":true`,
		`"uses_file_boundary":false`,
		`"uses_wav_boundary":false`,
		`"tts_doubao_realtime_config_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "tts_wav_file_boundary_not_xiaozhi_streaming") {
		t.Fatalf("doubao realtime TTS was misclassified as WAV: %s", stdout.String())
	}
}

func TestXiaozhiStreamingProviderReadinessAcceptsConfiguredDoubaoRealtimeTTSStageOnly(t *testing.T) {
	t.Setenv("A21_ASR_LOCAL_PROFILE", "sherpa_onnx_streaming")
	t.Setenv("A21_TEXT_STREAM_PROFILE", "stepfun")
	t.Setenv("A21_TTS_FAST_PROFILE", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "zh_female_kailangjiejie_moon_bigtts")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"xiaozhi-streaming-provider-readiness", "--output-dir", ""}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code=%d, want blocked by ASR helper stdout=%s", code, stdout.String())
	}
	for _, want := range []string{
		`"adapter":"doubao_realtime_tts_adapter"`,
		`"ready":true`,
		`"real_provider":true`,
		`"streaming":true`,
		`"asr_sherpa_streaming_helper_or_model_missing"`,
		`"gate_status":"blocked"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie_moon_bigtts", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked %q: %s", forbidden, stdout.String())
		}
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
