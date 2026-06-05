package app

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/providers"
)

func TestRunXiaozhiVoiceBenchReportsHostOnlyCandidateEvidence(t *testing.T) {
	gatewayServer := newGatewayServerFromEnv(nil)
	httpServer := httptest.NewServer(gatewayServer.Handler())
	t.Cleanup(httpServer.Close)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{
		"xiaozhi-voice-bench",
		"--gateway-url", httpServer.URL,
		"--repeat", "1",
		"--timeout-ms", "5000",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"schema_version": "a21.xiaozhi_voice_bench.v1"`,
		`"execution_mode": "host_loopback"`,
		`"baseline_scope": "host_only"`,
		`"gateway": "loopback:`,
		`"profile": "xiaozhi"`,
		`"protocol_version": 1`,
		`"hello_accepted": true`,
		`"listen_ack": true`,
		`"binary_downlink_frames"`,
		`"downlink_audio_quality"`,
		`"codec": "opus_decoded_pcm_s16le"`,
		`"sample_rate_hz": 24000`,
		`"status": "passed"`,
		`"trace_summary"`,
		`"xiaozhi_opus_decode_ms"`,
		`"asr_first_partial_ms"`,
		`"asr_final_ms"`,
		`"llm_first_content_ms"`,
		`"tts_first_audio_ms"`,
		`"audio_downlink_first_frame_ms"`,
		`"answer_first_audio_total_ms"`,
		`"answer_first_audio_total_p95_ms"`,
		`"barge_in_stop_p95_ms"`,
		`"acceptance_status": "candidate_host_only"`,
		`"prd_accepted": false`,
		`"provider_executed": false`,
		`"v21_executed": false`,
		`"hardware_executed": false`,
		`"voice_pipeline_observed": true`,
		`"voice_pipeline_execution_mode": "fixture"`,
		`"asr_profile": "mock-local-asr"`,
		`"asr_profile_env": "A21_ASR_LOCAL_PROFILE"`,
		`"llm_profile": "mock"`,
		`"llm_profile_env": "A21_PROVIDER_PRIMARY"`,
		`"tts_profile": "mock-fast-tts"`,
		`"tts_profile_env": "A21_TTS_FAST_PROFILE"`,
		`"host_local_asr_executed": false`,
		`"host_local_text_executed": false`,
		`"host_local_tts_executed": false`,
		`"host_product_chain_ready": false`,
		`"payloads_stored": false`,
		`"report_path"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{
		httpServer.URL,
		dir,
		"data_base64",
		"transcript",
		"provider output",
		"secret",
		`"prd_accepted": true`,
		`"acceptance_status": "accepted"`,
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, rendered)
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-xiaozhi-voice-bench-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("xiaozhi voice bench reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), httpServer.URL) || strings.Contains(string(data), dir) {
		t.Fatalf("report leaked gateway URL or output dir: %s", string(data))
	}
}

func TestRunXiaozhiVoiceBenchRequireProductChainRejectsFixture(t *testing.T) {
	gatewayServer := newGatewayServerFromEnv(nil)
	httpServer := httptest.NewServer(gatewayServer.Handler())
	t.Cleanup(httpServer.Close)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{
		"xiaozhi-voice-bench",
		"--gateway-url", httpServer.URL,
		"--repeat", "1",
		"--timeout-ms", "5000",
		"--require-product-chain",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("code = 0, want non-zero for fixture chain: stdout=%s", stdout.String())
	}
	rendered := stdout.String()
	if !strings.Contains(rendered, `"code": "product_chain_not_executed"`) {
		t.Fatalf("stdout missing product_chain_not_executed finding: %s", rendered)
	}
	if strings.Contains(rendered, `"host_local_text_executed": true`) {
		t.Fatalf("fixture report must not claim host-local text execution: %s", rendered)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %s, want empty", stderr.String())
	}
}

func TestXiaozhiVoiceBenchWebSocketFailureFindingIsRedacted(t *testing.T) {
	if got := xiaozhiVoiceBenchWebSocketFailureFinding(&http.Response{StatusCode: http.StatusForbidden}, fmt.Errorf("http://user:pass@example.invalid/secret")); got != "gateway_websocket_unavailable_http_403" {
		t.Fatalf("http finding = %q, want gateway_websocket_unavailable_http_403", got)
	}
	proxyErr := fmt.Errorf("proxy http://user:pass@127.0.0.1:7897 failed for /Users/me/a21-secret.txt with token")
	if got := xiaozhiVoiceBenchWebSocketFailureFinding(nil, proxyErr); got != "gateway_websocket_unavailable_proxy_or_tunnel" {
		t.Fatalf("proxy finding = %q, want gateway_websocket_unavailable_proxy_or_tunnel", got)
	} else {
		for _, forbidden := range []string{"http://", "user:pass", "/Users/", "token", "secret"} {
			if strings.Contains(got, forbidden) {
				t.Fatalf("finding leaked %q: %q", forbidden, got)
			}
		}
	}
	if got := xiaozhiVoiceBenchWebSocketFailureFinding(nil, fmt.Errorf("dial tcp 127.0.0.1:21080: connect: operation not permitted")); got != "gateway_websocket_unavailable_operation_not_permitted" {
		t.Fatalf("permission finding = %q, want gateway_websocket_unavailable_operation_not_permitted", got)
	}
}

func TestXiaozhiVoiceBenchExecutionFromPipelineRequiresNonMockProductStages(t *testing.T) {
	execution := xiaozhiVoiceBenchExecutionFromPipeline(map[string]any{
		"execution_mode": "host_local",
		"selection": map[string]any{
			"asr_profile":     "sherpa_onnx",
			"asr_profile_env": "A21_ASR_LOCAL_PROFILE",
			"llm_profile":     "mock",
			"llm_profile_env": "A21_PROVIDER_PRIMARY",
			"tts_profile":     "sherpa_onnx_tts",
			"tts_profile_env": "A21_TTS_FAST_PROFILE",
		},
	})

	if !execution.HostLocalASRExecuted || execution.HostLocalTextExecuted || !execution.HostLocalTTSExecuted {
		t.Fatalf("execution = %+v, want host-local ASR/TTS only and mock text rejected", execution)
	}
	if xiaozhiVoiceBenchProductChainReady(execution) {
		t.Fatalf("execution = %+v, want product chain not ready with mock text", execution)
	}
}

func TestXiaozhiVoiceBenchExecutionFromPipelineIgnoresFastAckConfiguredSummary(t *testing.T) {
	execution := xiaozhiVoiceBenchExecutionFromPipeline(map[string]any{
		"schema_version": "a21.voice_pipeline.fast_ack.v1",
		"status":         "running",
		"stage":          "fast_ack",
		"execution_mode": "host_local",
		"selection": map[string]any{
			"asr_profile":     "sherpa_onnx",
			"asr_profile_env": "A21_ASR_LOCAL_PROFILE",
			"llm_profile":     "local_ollama",
			"llm_profile_env": "A21_TEXT_STREAM_PROFILE",
			"tts_profile":     "sherpa_onnx_tts",
			"tts_profile_env": "A21_TTS_FAST_PROFILE",
		},
	})

	if execution.HostLocalASRExecuted || execution.HostLocalTextExecuted || execution.HostLocalTTSExecuted || execution.ProviderExecuted || execution.HostProductChainReady {
		t.Fatalf("execution = %+v, want fast-ack configuration not counted as executed product chain", execution)
	}
	if execution.VoicePipelineExecutionMode != "host_local" {
		t.Fatalf("execution mode = %q, want host_local preserved for diagnostics", execution.VoicePipelineExecutionMode)
	}
}

func TestXiaozhiVoiceBenchExecutionFromPipelineMarksNonMockTextProviderExecuted(t *testing.T) {
	execution := xiaozhiVoiceBenchExecutionFromPipeline(map[string]any{
		"execution_mode": "host_local",
		"selection": map[string]any{
			"asr_profile":     "sherpa_onnx",
			"asr_profile_env": "A21_ASR_LOCAL_PROFILE",
			"llm_profile":     "local_ollama",
			"llm_profile_env": "A21_TEXT_STREAM_PROFILE",
			"tts_profile":     "sherpa_onnx_tts",
			"tts_profile_env": "A21_TTS_FAST_PROFILE",
		},
	})

	if !execution.ProviderExecuted || !xiaozhiVoiceBenchProductChainReady(execution) {
		t.Fatalf("execution = %+v, want non-mock text provider counted as executed product chain", execution)
	}
	if !execution.HostProductChainReady {
		t.Fatalf("execution = %+v, want explicit host product-chain flag", execution)
	}
}

func TestXiaozhiVoiceBenchExecutionFromPipelineMarksCloudEdgeProductChain(t *testing.T) {
	execution := xiaozhiVoiceBenchExecutionFromPipeline(map[string]any{
		"execution_mode": "cloud_edge",
		"selection": map[string]any{
			"asr_profile":     "dashscope_qwen_asr_realtime",
			"asr_profile_env": "A21_ASR_CLOUD_PROFILE",
			"llm_profile":     "deepseek",
			"llm_profile_env": "A21_TEXT_STREAM_PROFILE",
			"tts_profile":     "dashscope_qwen_tts_realtime",
			"tts_profile_env": "A21_TTS_FAST_PROFILE",
		},
	})

	if execution.VoicePipelineExecutionMode != "cloud_edge" || !execution.ProviderExecuted || !execution.HostProductChainReady {
		t.Fatalf("execution = %+v, want cloud edge provider product chain", execution)
	}
	if execution.HostLocalASRExecuted || execution.HostLocalTextExecuted || execution.HostLocalTTSExecuted || execution.HardwareExecuted {
		t.Fatalf("execution = %+v, want cloud edge not host-local or hardware execution", execution)
	}
}

func TestRunXiaozhiVoiceBenchSupportsRedactedInputWAVFixture(t *testing.T) {
	gatewayServer := newGatewayServerFromEnv(nil)
	httpServer := httptest.NewServer(gatewayServer.Handler())
	t.Cleanup(httpServer.Close)
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "a21-input.wav")
	writeAppTestWAV(t, wavPath, 16000, bytes.Repeat([]byte{0x70, 0x17}, 960*2))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{
		"xiaozhi-voice-bench",
		"--gateway-url", httpServer.URL,
		"--input-wav", wavPath,
		"--repeat", "1",
		"--timeout-ms", "5000",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"source": "wav_fixture"`,
		`"wav_name": "a21-input.wav"`,
		`"opus_frame_count": 2`,
		`"acceptance_status": "candidate_host_only"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{wavPath, dir, "data_base64", "raw_audio"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, rendered)
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-xiaozhi-voice-bench-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("xiaozhi voice bench reports = %v, %v", matches, err)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), wavPath) || strings.Contains(string(data), dir) || strings.Contains(string(data), "data_base64") {
		t.Fatalf("report leaked local path or audio payload: %s", data)
	}
}

func TestRunXiaozhiProfessionalBenchReportsHostMockRuntimeContract(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{
		"xiaozhi-professional-bench",
		"--fake-v21-delay-ms", "150",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"schema_version": "a21.xiaozhi_professional_bench.v1"`,
		`"source_profile": "host_mock"`,
		`"acceptance_status": "host_mock_ready"`,
		`"prd_accepted": false`,
		`"checking_feedback_observed": true`,
		`"checking_feedback_within_1200": true`,
		`"professional_result_observed": true`,
		`"professional_result_after_checking": true`,
		`"abort_stop_observed": true`,
		`"stale_result_suppressed": true`,
		`"evidence_count": 1`,
		`"screen_card_count": 1`,
		`"follow_up_count": 1`,
		`"confidence_present": true`,
		`"no_placeholder_utterance": true`,
		`"no_asr_text_leak": true`,
		`"tts_stop_observed": true`,
		`"failure_count": 0`,
		`"payloads_stored": false`,
		`"provider_executed": false`,
		`"v21_executed": false`,
		`"hardware_executed": false`,
		`"report_path"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{
		dir,
		"a21 host mock professional ASR sentinel",
		"xiaozhi professional voice turn",
		"RAW_SECRET_EVIDENCE_BODY",
		"RAW_SECRET_CARD_TEXT",
		"RAW_SECRET_FOLLOW_UP",
		"data_base64",
		"transcript",
		"prompt text",
		"provider output",
		"http://",
		"https://",
		`"prd_accepted": true`,
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, rendered)
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-xiaozhi-professional-bench-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("xiaozhi professional bench reports = %v, %v", matches, err)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, forbidden := range []string{
		dir,
		"a21 host mock professional ASR sentinel",
		"RAW_SECRET_EVIDENCE_BODY",
		"RAW_SECRET_CARD_TEXT",
		"RAW_SECRET_FOLLOW_UP",
		"http://",
		"https://",
	} {
		if strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report leaked forbidden fragment %q: %s", forbidden, reportJSON)
		}
	}
}

func TestRunXiaozhiProfessionalBenchReportsExternalGatewayRuntimeContract(t *testing.T) {
	dir := t.TempDir()
	v21 := &xiaozhiProfessionalBenchV21Client{delay: 25 * time.Millisecond}
	adapters := providers.VoicePipelineAdapters{
		ASR:        xiaozhiProfessionalBenchASRAdapter{text: xiaozhiProfessionalBenchASRSentinel},
		TextStream: xiaozhiProfessionalBenchTextStreamAdapter{},
		TTS:        providers.NewMockTTSAdapter("a21-host-mock-tts"),
	}
	server := gateway.NewServerWithOptions(gateway.ServerOptions{
		V21Client:                    v21,
		XiaozhiVoicePipelineAdapters: &adapters,
	})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{
		"xiaozhi-professional-bench",
		"--gateway-url", httpServer.URL,
		"--device-id", "stackchan-virtual-a21-professional-external-001",
		"--protocol-version", "3",
		"--timeout-ms", "3000",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if got := v21.lastUtterance(); got != xiaozhiProfessionalBenchASRSentinel {
		t.Fatalf("v21 utterance = %q, want ASR-derived sentinel", got)
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"schema_version": "a21.xiaozhi_professional_bench.v1"`,
		`"source_profile": "external_gateway"`,
		`"gateway": "loopback:`,
		`"input_audio": {`,
		`"source": "synthetic_opus"`,
		`"acceptance_status": "external_gateway_ready"`,
		`"prd_accepted": false`,
		`"checking_feedback_observed": true`,
		`"checking_feedback_within_1200": true`,
		`"professional_result_observed": true`,
		`"professional_result_after_checking": true`,
		`"abort_stop_observed": true`,
		`"stale_result_suppressed": true`,
		`"evidence_count": 1`,
		`"screen_card_count": 1`,
		`"follow_up_count": 1`,
		`"confidence_present": true`,
		`"no_placeholder_utterance": true`,
		`"failure_count": 0`,
		`"provider_executed": false`,
		`"v21_executed": true`,
		`"hardware_executed": false`,
		`"gateway_runtime": "external_gateway"`,
		`"read_record": {`,
		`"completed": true`,
		`"query_scope": "public_only"`,
		`"privacy_scope": "professional_only"`,
		`"workspace_status": "searchable"`,
		`"voice_text_stored": false`,
		`"report_path"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{
		dir,
		httpServer.URL,
		xiaozhiProfessionalBenchASRSentinel,
		"RAW_SECRET_EVIDENCE_BODY",
		"RAW_SECRET_CARD_TEXT",
		"RAW_SECRET_FOLLOW_UP",
		"data_base64",
		"transcript",
		"prompt text",
		"provider output",
		"http://",
		"https://",
		`"prd_accepted": true`,
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, rendered)
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-xiaozhi-professional-bench-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("xiaozhi professional external reports = %v, %v", matches, err)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, forbidden := range []string{dir, httpServer.URL, xiaozhiProfessionalBenchASRSentinel, "RAW_SECRET", "http://", "https://"} {
		if strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report leaked forbidden fragment %q: %s", forbidden, reportJSON)
		}
	}
}

func TestRunXiaozhiProfessionalBenchReportsFailingFakePathWithoutLeak(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{
		"xiaozhi-professional-bench",
		"--scenario", "v21_failure",
	}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.xiaozhi_professional_bench.v1"`,
		`"source_profile": "host_mock"`,
		`"acceptance_status": "host_mock_blocked"`,
		`"prd_accepted": false`,
		`"checking_feedback_observed": true`,
		`"professional_result_observed": false`,
		`"code": "professional_result_missing"`,
		`"failure_count": 1`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{
		"a21 host mock professional ASR sentinel",
		"RAW_SECRET",
		"secret-token",
		"http://",
		"https://",
		"provider output",
		"reasoning",
	} {
		if strings.Contains(stdout.String()+stderr.String(), forbidden) {
			t.Fatalf("failing bench leaked forbidden fragment %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunProviderLatencyBenchRejectsDeprecatedHostBaselineMode(t *testing.T) {
	var stderr bytes.Buffer
	code := RunLab([]string{"provider-latency-bench", "--mode", "host_baseline"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "--mode must be mock, fixture, or host_loopback") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunProviderLatencyBenchRejectsExecute(t *testing.T) {
	var stderr bytes.Buffer
	code := RunLab([]string{"provider-latency-bench", "--execute"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "does not support --execute") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunLocalTTSSmokeWritesRedactedReport(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		if options.Text != "这句话不应该出现在报告里" {
			t.Fatalf("text = %q", options.Text)
		}
		outputPath := filepath.Join(options.OutputDir, "a21-local-tts-test.wav")
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "macos_say",
			Voice:           "Tingting",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     8,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      12.5,
			TTSFirstAudioMS: 12.5,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-tts-smoke", "--engine", "macos_say", "--text", "这句话不应该出现在报告里", "--voice", "Tingting", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{`"schema_version": "a21.audio.local_tts.v1"`, `"status": "passed"`, `"provider": "macos_say"`, `"report_path"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-tts-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"这句话不应该出现在报告里", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("local TTS smoke leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalTTSSmokeSupportsSherpaONNXEngine(t *testing.T) {
	original := synthesizeSherpaONNX
	t.Cleanup(func() { synthesizeSherpaONNX = original })
	synthesizeSherpaONNX = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		if options.ModelDir != "/tmp/a21-sherpa-model" || options.SpeakerID != 21 {
			t.Fatalf("sherpa options = %+v", options)
		}
		outputPath := filepath.Join(options.OutputDir, "a21-sherpa-test.wav")
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "sherpa_onnx",
			Engine:          "vits_icefall_zh_aishell3",
			Voice:           "sid_21",
			ModelDir:        "vits-icefall-zh-aishell3",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     8,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      22.5,
			TTSFirstAudioMS: 22.5,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-tts-smoke", "--engine", "sherpa_onnx", "--text", "不能进报告", "--model-dir", "/tmp/a21-sherpa-model", "--speaker-id", "21", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{`"provider": "sherpa_onnx"`, `"engine": "vits_icefall_zh_aishell3"`, `"model_dir": "vits-icefall-zh-aishell3"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "不能进报告") || strings.Contains(stdout.String(), "/tmp/a21-sherpa-model") {
		t.Fatalf("sherpa smoke leaked sensitive content: %s", stdout.String())
	}
}

func TestRunLocalTTSSmokeSupportsVoiceCloneCLIEngine(t *testing.T) {
	original := synthesizeVoiceCloneCLI
	t.Cleanup(func() { synthesizeVoiceCloneCLI = original })
	refAudio := filepath.Join(t.TempDir(), "a21-persona-reference.wav")
	if err := os.WriteFile(refAudio, []byte("RIFF-a21-reference"), 0o644); err != nil {
		t.Fatal(err)
	}
	refTextPath := filepath.Join(t.TempDir(), "a21-reference.txt")
	if err := os.WriteFile(refTextPath, []byte("参考文本不能进报告"), 0o600); err != nil {
		t.Fatal(err)
	}
	command := "/usr/bin/ssh -i /a21-lab/secrets/a21_5080_fixture_ed25519 21@192.168.1.6 powershell -NoProfile -File D:/a21-mainland-latency-lab/outbox/a21-index-tts2-wrapper.ps1"
	synthesizeVoiceCloneCLI = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		if options.Text != "不能进报告" ||
			options.VoiceCloneCommand != command ||
			options.VoiceCloneModel != "Index-TTS2" ||
			options.VoiceCloneReferenceAudioPath != refAudio ||
			options.VoiceCloneReferenceText != "" ||
			options.VoiceCloneReferenceTextPath != refTextPath ||
			options.VoiceClonePersona != "A21 Workmate" ||
			options.VoiceCloneStyle != "Warm-Pro" {
			t.Fatalf("clone options = %+v", options)
		}
		outputPath := filepath.Join(options.OutputDir, "a21-voice-clone-test.wav")
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "voice_clone_cli",
			Engine:          "voice_clone_cli",
			Voice:           "a21_workmate",
			Model:           "index_tts2",
			VoicePersona:    "a21_workmate",
			StyleProfile:    "warm_pro",
			ReferenceAudio:  "a21-persona-reference.wav",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     8,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      18.5,
			TTSFirstAudioMS: 18.5,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"local-tts-smoke",
		"--engine", "voice_clone_cli",
		"--text", "不能进报告",
		"--clone-command", command,
		"--clone-model", "Index-TTS2",
		"--clone-ref-audio", refAudio,
		"--clone-ref-text-file", refTextPath,
		"--voice-persona", "A21 Workmate",
		"--voice-style", "Warm-Pro",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "voice_clone_cli"`,
		`"engine": "voice_clone_cli"`,
		`"voice": "a21_workmate"`,
		`"model": "index_tts2"`,
		`"voice_persona": "a21_workmate"`,
		`"style_profile": "warm_pro"`,
		`"reference_audio": "a21-persona-reference.wav"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"不能进报告", "参考文本不能进报告", refTextPath, refAudio, dir, "a21_5080_fixture_ed25519", "192.168.1.6", "Authorization", "Bearer", "raw_audio", "data_base64"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("voice clone smoke leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunLocalTTSSmokeSupportsIflytekTTSEngineSelection(t *testing.T) {
	original := synthesizeIflytekTTS
	t.Cleanup(func() { synthesizeIflytekTTS = original })
	synthesizeIflytekTTS = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		if options.Text != "不能进报告" {
			t.Fatalf("iflytek options = %+v", options)
		}
		outputPath := filepath.Join(options.OutputDir, "a21-iflytek-test.wav")
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "iflytek_tts",
			Engine:          "iflytek_tts",
			EndpointHost:    "tts-api.xfyun.cn",
			Voice:           "xiaoyan",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     8,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      16.5,
			TTSFirstAudioMS: 16.5,
		}, nil
	}

	tests := []struct {
		name string
		args []string
		env  string
	}{
		{
			name: "flag",
			args: []string{"local-tts-smoke", "--engine", "iflytek_tts", "--text", "不能进报告"},
		},
		{
			name: "env",
			args: []string{"local-tts-smoke", "--text", "不能进报告"},
			env:  "iflytek_tts",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				t.Setenv("A21_LOCAL_TTS_ENGINE", tt.env)
			}
			dir := t.TempDir()
			args := append([]string(nil), tt.args...)
			args = append(args, "--output-dir", dir)
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(args, &stdout, &stderr)

			if code != 0 {
				t.Fatalf("code = %d, want 0: %s", code, stderr.String())
			}
			for _, want := range []string{
				`"provider": "iflytek_tts"`,
				`"engine": "iflytek_tts"`,
				`"endpoint_host": "tts-api.xfyun.cn"`,
				`"voice": "xiaoyan"`,
			} {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout missing %q: %s", want, stdout.String())
				}
			}
			for _, forbidden := range []string{"不能进报告", dir, "Authorization", "Bearer", "data_base64", "raw_audio", "http://", "wss://"} {
				if strings.Contains(stdout.String(), forbidden) {
					t.Fatalf("iflytek smoke leaked %q: %s", forbidden, stdout.String())
				}
			}
		})
	}
}

func TestRunLocalTTSSmokeWritesIflytekFailureReport(t *testing.T) {
	original := synthesizeIflytekTTS
	t.Cleanup(func() { synthesizeIflytekTTS = original })
	synthesizeIflytekTTS = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		return audio.LocalTTSReport{
			SchemaVersion: "a21.audio.local_tts.v1",
			GeneratedAtMS: time.Now().UnixMilli(),
			Status:        "failed",
			Provider:      "iflytek_tts",
			Engine:        "iflytek_tts",
			EndpointHost:  "tts-api.xfyun.cn",
			NetworkMode:   "direct",
			Voice:         "xiaoyan",
			OutputFormat:  "wav_pcm_s16le_16000_mono",
			TextBytes:     len([]byte(options.Text)),
			Findings:      []string{"iflytek_tts_websocket_dial_failed_http_403"},
		}, fmt.Errorf("iflytek TTS websocket dial failed")
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-tts-smoke", "--engine", "iflytek_tts", "--text", "不能进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	for _, want := range []string{
		`"status": "failed"`,
		`"provider": "iflytek_tts"`,
		`"endpoint_host": "tts-api.xfyun.cn"`,
		`"network_mode": "direct"`,
		`"iflytek_tts_websocket_dial_failed_http_403"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-tts-smoke-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("reports = %v, %v", matches, err)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"不能进报告", dir, "Authorization", "Bearer", "data_base64", "raw_audio", "http://", "wss://"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("iflytek failure report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}
