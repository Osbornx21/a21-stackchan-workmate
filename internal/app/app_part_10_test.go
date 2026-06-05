package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/providers"
)

func TestRunProviderSmokeWritesRedactedReportWhenOutputDirProvided(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-secret")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-smoke", "--provider", "deepseek", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"report_path"`) {
		t.Fatalf("stdout missing report_path: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-provider-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("provider smoke reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, want := range []string{
		`"schema_version": "a21.provider_smoke.v1"`,
		`"generated_at_ms"`,
		`"provider": "deepseek"`,
		`"status": "ready"`,
		`"executed": false`,
		`"report_path": "a21-provider-smoke-`,
	} {
		if !strings.Contains(reportJSON, want) {
			t.Fatalf("provider smoke report missing %q: %s", want, reportJSON)
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-chat", dir, filepath.ToSlash(dir)} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("provider smoke leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func TestWriteProviderSmokeReportDoesNotOverwriteSameSecondReports(t *testing.T) {
	dir := t.TempDir()
	first, err := writeProviderSmokeReport(dir, providers.ProviderSmokeReport{Provider: "deepseek", Status: providers.ProviderSmokeReady})
	if err != nil {
		t.Fatal(err)
	}
	second, err := writeProviderSmokeReport(dir, providers.ProviderSmokeReport{Provider: "deepseek-p0-collision-check", Status: providers.ProviderSmokeReady})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("report paths collided: %s", first)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-provider-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("provider smoke reports = %d, want 2: %v", len(matches), matches)
	}
}

func TestRunProviderSmokeAcceptsStreamRepeatFlags(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-secret")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-smoke", "--provider", "deepseek", "--stream", "--repeat", "3"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{`"provider": "deepseek"`, `"status": "ready"`, `"stream": true`, `"repeat": 3`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-chat"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunProviderCompatMatrixIngests5080FullSummary(t *testing.T) {
	dir := t.TempDir()
	summaryPath := filepath.Join(dir, "a21-provider-full-summary.json")
	summary := `{
  "schema_version": "a21.provider_full_validation.v1",
  "lab_llm_stream_summary": [
    {
      "provider": "siliconflow",
      "attempts": 5,
      "ok": 5,
      "failures": 0,
      "first_content_p50_ms": 320.5,
      "first_content_p95_ms": 360.5,
      "total_p95_ms": 900.25,
      "last_error": null
    }
  ],
  "a21_deepseek_execute": {
    "schema_version": "a21.provider_smoke.v1",
    "generated_at_ms": 1780226954489,
    "provider": "deepseek",
    "family": "text_stream",
    "protocol": "openai_chat_completions",
    "status": "passed",
    "configured": true,
    "executed": true,
    "route_eligible": true,
    "stream": true,
    "repeat": 5,
    "http_status": 200,
    "timing_summary": {
      "repeat": 5,
      "first_byte_p95_ms": 2542.379,
      "first_content_p95_ms": 3000.25,
      "total_duration_p95_ms": 4323.647
    },
    "network_mode": "direct",
    "endpoint_host": "api.deepseek.com",
    "base_url_env": "A21_DEEPSEEK_BASE_URL",
    "api_key_env": "A21_LAB_DEEPSEEK_API_KEY",
    "model_env": "A21_DEEPSEEK_MODEL",
    "report_path": "reports/a21-provider-full-20260531-192912/a21-provider-smoke-20260531-192923-493856000.json",
    "detail": "provider streaming smoke request succeeded"
  },
  "local_asr": [
    {
      "schema_version": "a21.audio.local_asr.v1",
      "status": "passed",
      "provider": "sherpa_onnx",
      "engine": "paraformer",
      "model_dir": "sherpa-onnx-paraformer-zh-small-2024-03-09",
      "decode_duration_ms": 492.076,
      "real_time_factor": 0.181369,
      "report_path": "reports/a21-provider-full-20260531-192912/a21-local-asr-smoke-20260531-193314.json",
      "matrix_model": "sherpa-onnx-paraformer-zh-small-2024-03-09",
      "matrix_stage": "local_asr",
      "exit_code": 0
    }
  ],
  "local_tts": [
    {
      "schema_version": "a21.audio.local_tts.v1",
      "status": "passed",
      "provider": "sherpa_onnx",
      "engine": "vits_icefall_zh_aishell3",
      "model_dir": "vits-icefall-zh-aishell3",
      "duration_ms": 439.041,
      "tts_first_audio_ms": 439.041,
      "report_path": "reports/a21-provider-full-20260531-192912/a21-local-tts-smoke-20260531-193309.json",
      "matrix_model": "vits-icefall-zh-aishell3",
      "matrix_stage": "local_tts",
      "exit_code": 0
    }
  ]
}`
	if err := os.WriteFile(summaryPath, []byte(summary), 0o600); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(dir, "reports")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-compat-matrix", "--provider-full-summary", summaryPath, "--output-dir", outputDir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report providerCompatMatrixReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider compat matrix: %v\n%s", err, stdout.String())
	}
	if report.SchemaVersion != providerCompatMatrixSchemaVersion || report.SourceSummaryReport != "a21-provider-full-summary.json" {
		t.Fatalf("schema/source = %q/%q", report.SchemaVersion, report.SourceSummaryReport)
	}
	if !report.Coverage.LocalASR || !report.Coverage.CloudLLM || !report.Coverage.LocalTTS {
		t.Fatalf("coverage = %+v, want local ASR/cloud LLM/local TTS covered", report.Coverage)
	}
	for _, want := range []string{"cloud_asr", "local_llm", "cloud_tts"} {
		if !containsExactProductString(report.MissingCapabilities, want) {
			t.Fatalf("missing capabilities lacks %q: %#v", want, report.MissingCapabilities)
		}
	}
	if len(report.Rows) < 4 || report.ReportPath == "" {
		t.Fatalf("rows/report_path = %d/%q, want rows and written report", len(report.Rows), report.ReportPath)
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-provider-compat-matrix-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("matrix report matches = %v, %v", matches, err)
	}
	for _, forbidden := range []string{dir, filepath.ToSlash(dir), summaryPath} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("provider compat matrix leaked path %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunProviderCompatMatrixUseLatestReportsIncludesLocalLLM(t *testing.T) {
	reportDir := t.TempDir()
	providerFullDir := filepath.Join(reportDir, "a21-provider-full-20260531-192912")
	if err := os.MkdirAll(providerFullDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providerFullDir, "a21-provider-full-summary.json"), []byte(`{
  "schema_version": "a21.provider_full_validation.v1",
  "lab_llm_stream_summary": [],
  "local_asr": [],
  "local_tts": []
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	localLLM := providers.ProviderSmokeReport{
		SchemaVersion: providers.ProviderSmokeSchemaVersion,
		GeneratedAtMS: time.Now().UnixMilli(),
		Provider:      "local_ollama",
		Family:        string(providers.ProviderFamilyTextStream),
		Protocol:      "ollama_chat",
		Status:        providers.ProviderSmokePassed,
		Configured:    true,
		Executed:      true,
		RouteEligible: true,
		Stream:        true,
		Repeat:        3,
		ReportPath:    "a21-provider-smoke-local-ollama.json",
		Detail:        "provider streaming smoke request succeeded",
	}
	data, err := json.Marshal(localLLM)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "a21-provider-smoke-20260602-120000.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-compat-matrix", "--use-latest-reports", "--reports-dir", reportDir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report providerCompatMatrixReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider compat matrix: %v\n%s", err, stdout.String())
	}
	if !report.Coverage.LocalLLM {
		t.Fatalf("coverage = %+v, want local LLM covered", report.Coverage)
	}
	if report.SourceSummaryReport != "a21-provider-full-summary.json" {
		t.Fatalf("source summary = %q", report.SourceSummaryReport)
	}
	for _, forbidden := range []string{reportDir, filepath.ToSlash(reportDir)} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("provider compat matrix leaked path %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunProviderCompatMatrixUseLatestReportsIncludesVoiceCloneLocalTTS(t *testing.T) {
	reportDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(reportDir, "a21-local-tts-smoke-20260602-120100.json"), []byte(`{
  "schema_version": "a21.audio.local_tts.v1",
  "generated_at_ms": 1780368260000,
  "status": "passed",
  "provider": "voice_clone_cli",
  "engine": "voice_clone_cli",
  "voice": "a21_workmate",
  "model": "index_tts2",
  "voice_persona": "a21_workmate",
  "style_profile": "workmate_warm",
  "reference_audio": "a21-persona-reference.wav",
  "output_format": "wav_pcm_s16le_16000_mono",
  "tts_first_audio_ms": 71.688,
  "report_path": "D:\\a21-mainland-latency-lab\\outbox\\a21-local-tts-smoke-20260602-120100.json"
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-compat-matrix", "--use-latest-reports", "--reports-dir", reportDir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report providerCompatMatrixReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider compat matrix: %v\n%s", err, stdout.String())
	}
	if !report.Coverage.LocalTTS {
		t.Fatalf("coverage = %+v, want voice_clone_cli local TTS covered", report.Coverage)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows = %#v, want one voice clone local TTS row", report.Rows)
	}
	row := report.Rows[0]
	if row.Provider != "voice_clone_cli" || row.Model != "index_tts2" || row.SourceReport != "a21-local-tts-smoke-20260602-120100.json" {
		t.Fatalf("voice clone row = %+v", row)
	}
	for _, forbidden := range []string{reportDir, filepath.ToSlash(reportDir), "D:\\", "D:/", "outbox"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("provider compat matrix leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunProviderCompatMatrixIngestsCloudAudioSmokeReports(t *testing.T) {
	dir := t.TempDir()
	asrPath := filepath.Join(dir, "a21-provider-audio-smoke-asr.json")
	ttsPath := filepath.Join(dir, "a21-provider-audio-smoke-tts.json")
	if err := os.WriteFile(asrPath, []byte(`{
  "schema_version": "a21.provider_audio_smoke.v1",
  "generated_at_ms": 1780368200000,
  "stage": "asr",
  "placement": "cloud",
  "provider": "iflytek",
  "status": "passed",
  "executed": true,
  "configured": true,
  "endpoint_host": "iat-api.xfyun.cn",
  "asr_final_p95_ms": 354,
  "report_path": "D:\\a21-mainland-latency-lab\\outbox\\a21-provider-audio-smoke-asr.json"
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ttsPath, []byte(`{
  "schema_version": "a21.provider_audio_smoke.v1",
  "generated_at_ms": 1780368200001,
  "stage": "tts",
  "placement": "cloud",
  "provider": "iflytek",
  "status": "passed",
  "executed": true,
  "configured": true,
  "endpoint_host": "tts-api.xfyun.cn",
  "tts_first_audio_p95_ms": 90,
  "report_path": "D:\\a21-mainland-latency-lab\\outbox\\a21-provider-audio-smoke-tts.json"
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"provider-compat-matrix",
		"--provider-audio-smoke-report", asrPath,
		"--provider-audio-smoke-report", ttsPath,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report providerCompatMatrixReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider compat matrix: %v\n%s", err, stdout.String())
	}
	if !report.Coverage.CloudASR || !report.Coverage.CloudTTS {
		t.Fatalf("coverage = %+v, want cloud ASR and cloud TTS covered", report.Coverage)
	}
	for _, want := range []string{`"asr_final_p95_ms": 354`, `"tts_first_audio_ms": 90`, `"source_report": "a21-provider-audio-smoke-asr.json"`, `"source_report": "a21-provider-audio-smoke-tts.json"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{dir, filepath.ToSlash(dir), "D:\\", "D:/", "outbox"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("provider compat matrix leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunProviderSmokeRejectsLegacyProviderWithoutEchoingValue(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-smoke", "--provider", "x21_voice"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"provider": "invalid_legacy_provider"`) {
		t.Fatalf("stdout missing redacted provider: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "x21_voice") || strings.Contains(stderr.String(), "x21_voice") {
		t.Fatalf("legacy provider leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunProviderLatencyBenchMockEmitsRedactedCandidateChainReport(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://user:secret@example.invalid:8080")
	t.Setenv("A21_PROVIDER_PROXY_URL", "http://provider-secret@example.invalid:9000")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-secret")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "deepseek", "--iterations", "3", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	reportJSON := stdout.String()
	for _, want := range []string{
		`"schema_version": "a21.provider_latency_bench.v1"`,
		`"execution_mode": "mock"`,
		`"baseline_scope": "host_only"`,
		`"trace_id": "a21-trace-provider-latency-bench-000001"`,
		`"session_id": "a21-session-provider-latency-bench-000001"`,
		`"device_id": "none_host_fixture"`,
		`"profile": "deepseek"`,
		`"label": "DeepSeek text stream"`,
		`"family": "text_stream"`,
		`"network"`,
		`"mode": "explicit_proxy"`,
		`"proxy"`,
		`"HTTPS_PROXY"`,
		`"A21_PROVIDER_PROXY_URL"`,
		`"asr_first_partial_ms"`,
		`"provider_first_byte_ms"`,
		`"provider_first_content_ms"`,
		`"tts_first_audio_ms"`,
		`"downlink_first_frame_ms"`,
		`"device_playback_start_ms"`,
		`"barge_in_stop_ms"`,
		`"provider_cancel_ms"`,
		`"p50_ms"`,
		`"p95_ms"`,
		`"p99_ms"`,
		`"fallback_count": 0`,
		`"failure_count": 0`,
		`"promotion_gate": "not_production"`,
		`"provider_executed": false`,
		`"v21_executed": false`,
		`"hardware_executed": false`,
		`"report_path"`,
	} {
		if !strings.Contains(reportJSON, want) {
			t.Fatalf("stdout missing %q: %s", want, reportJSON)
		}
	}
	for _, forbidden := range []string{
		"sk-a21-secret",
		"secret",
		"example.invalid",
		"8080",
		"9000",
		"prompt",
		"transcript",
		"reasoning",
		"provider output",
		"/tmp/",
		dir,
	} {
		if strings.Contains(reportJSON, forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, reportJSON)
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-provider-latency-bench-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("provider latency bench reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	fileJSON := string(data)
	for _, want := range []string{`"promotion_gate": "not_production"`, `"provider_first_content_ms"`, `"report_path"`} {
		if !strings.Contains(fileJSON, want) {
			t.Fatalf("report missing %q: %s", want, fileJSON)
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "secret", "example.invalid", "8080", "9000", "/tmp/", dir} {
		if strings.Contains(fileJSON, forbidden) {
			t.Fatalf("report leaked forbidden fragment %q: %s", forbidden, fileJSON)
		}
	}
}

func TestRunProviderLatencyBenchV2ReportsMetricShapeContract(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--iterations", "2"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider latency report: %v\n%s", err, stdout.String())
	}
	assertProviderLatencyBenchMetricTerms(t, report)
	assertProviderLatencyBenchCanonicalMetrics(t, report)
	assertProviderLatencyBenchStageAvailability(t, report)
	rendered := stdout.String()
	for _, want := range []string{
		`"promotion_gate": "not_production"`,
		`"execution_mode": "host_loopback"`,
		`"audio_downlink_first_frame_ms"`,
		`"playback_stop_ms"`,
		`"provider_cancel_ms"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{"host_baseline", `"provider_executed": true`, `"v21_executed": true`, `"hardware_executed": true`} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout contains forbidden fragment %q: %s", forbidden, rendered)
		}
	}
}

func TestRunProviderLatencyBenchReportsWS6SegmentContract(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "fixture", "--iterations", "2"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider latency report: %v\n%s", err, stdout.String())
	}
	assertProviderLatencyBenchWS6Segments(t, report)
	rendered := stdout.String()
	for _, forbidden := range []string{
		`"promotion_gate": "accepted"`,
		`"acceptance_status": "accepted"`,
		`"prd_accepted": true`,
		`"provider_executed": true`,
		`"v21_executed": true`,
		`"hardware_executed": true`,
		"secret prompt",
		"secret transcript",
		"provider output",
		"data_base64",
		"/tmp/",
		"http://user:pass@",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout contains forbidden fragment %q: %s", forbidden, rendered)
		}
	}
	for _, want := range []string{
		`"promotion_gate": "not_production"`,
		`"acceptance_status": "not_accepted"`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
}

func assertProviderLatencyBenchWS6Segments(t *testing.T, report map[string]any) {
	t.Helper()
	rawStages, ok := report["stage_availability"].([]any)
	if !ok || len(rawStages) == 0 {
		t.Fatalf("stage_availability missing or empty: %#v", report["stage_availability"])
	}
	required := map[string]string{
		"transport_ingress_ms":          "audio.frame.received",
		"codec_decode_ms":               "xiaozhi.opus_frame.decoded",
		"asr_first_partial_ms":          "asr.first_partial",
		"asr_final_ms":                  "asr.final",
		"llm_first_content_ms":          "provider.first_content",
		"tts_first_audio_ms":            "tts.first_audio",
		"audio_downlink_first_frame_ms": "audio.downlink.first_frame",
		"device_playback_start_ms":      "device.playback.start",
		"barge_in_detected_ms":          "barge_in.detected",
		"provider_cancel_done_ms":       "provider.cancel.end",
		"playback_stop_done_ms":         "playback.stop",
	}
	seen := map[string]map[string]any{}
	for _, rawStage := range rawStages {
		stage, ok := rawStage.(map[string]any)
		if !ok {
			t.Fatalf("stage availability has unexpected shape: %#v", rawStage)
		}
		name, _ := stage["stage"].(string)
		seen[name] = stage
	}
	for stageName, traceMarker := range required {
		stage, ok := seen[stageName]
		if !ok {
			t.Fatalf("stage_availability missing WS-6 stage %q: %#v", stageName, rawStages)
		}
		if available, _ := stage["available"].(bool); available {
			t.Fatalf("WS-6 stage %q unexpectedly available: %#v", stageName, stage)
		}
		if placeholder, _ := stage["placeholder"].(bool); !placeholder {
			t.Fatalf("WS-6 stage %q should remain placeholder: %#v", stageName, stage)
		}
		if got, _ := stage["source_trace_marker"].(string); got != traceMarker {
			t.Fatalf("WS-6 stage %q source marker = %q, want %q: %#v", stageName, got, traceMarker, stage)
		}
		for _, field := range []string{"samples", "p50_ms", "p95_ms", "p99_ms"} {
			if _, ok := stage[field]; !ok {
				t.Fatalf("WS-6 stage %q missing %q redacted stats: %#v", stageName, field, stage)
			}
		}
	}
}

func assertProviderLatencyBenchMetricTerms(t *testing.T, report map[string]any) {
	t.Helper()
	rawTerms, ok := report["metric_terms"].([]any)
	if !ok || len(rawTerms) == 0 {
		t.Fatalf("metric_terms missing or empty: %#v", report["metric_terms"])
	}
	seenTerms := map[string]bool{}
	for _, rawTerm := range rawTerms {
		term, ok := rawTerm.(map[string]any)
		if !ok {
			t.Fatalf("metric term has unexpected shape: %#v", rawTerm)
		}
		name, _ := term["term"].(string)
		stage, _ := term["a21_stage"].(string)
		canonical, _ := term["canonical_metric"].(string)
		if name == "" || stage == "" || canonical == "" {
			t.Fatalf("metric term missing required fields: %#v", term)
		}
		seenTerms[name] = true
	}
	for _, want := range []string{"TTFS", "TTFT", "FTTS", "TTFA"} {
		if !seenTerms[want] {
			t.Fatalf("metric_terms missing %s: %#v", want, rawTerms)
		}
	}
}

func assertProviderLatencyBenchCanonicalMetrics(t *testing.T, report map[string]any) {
	t.Helper()
	metrics, ok := report["canonical_metrics"].(map[string]any)
	if !ok || len(metrics) == 0 {
		t.Fatalf("canonical_metrics missing or empty: %#v", report["canonical_metrics"])
	}
	summary, ok := report["summary"].(map[string]any)
	if !ok || len(summary) == 0 {
		t.Fatalf("summary missing or empty: %#v", report["summary"])
	}
	for metric := range summary {
		if _, ok := metrics[metric]; !ok {
			t.Fatalf("canonical_metrics missing summary metric %q: %#v", metric, metrics)
		}
	}
	for _, want := range []string{
		"asr_first_partial_ms",
		"provider_first_byte_ms",
		"provider_first_content_ms",
		"tts_first_audio_ms",
		"downlink_first_frame_ms",
		"audio_downlink_first_frame_ms",
		"device_playback_start_ms",
		"barge_in_stop_ms",
		"provider_cancel_ms",
		"playback_stop_ms",
		"speech_end_to_final_asr_ms",
		"speech_end_to_first_llm_token_ms",
		"llm_request_to_first_token_ms",
		"first_llm_token_to_first_tts_audio_ms",
		"tts_request_to_first_audio_ms",
		"provider_commit_to_first_audio_ms",
		"gateway_downlink_first_frame_ms",
		"device_downlink_first_frame_ms",
		"speech_end_to_first_audible_response_ms",
	} {
		raw, ok := metrics[want]
		if !ok {
			t.Fatalf("canonical_metrics missing %q: %#v", want, metrics)
		}
		series, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("canonical metric %q has unexpected shape: %#v", want, raw)
		}
		for _, field := range []string{"samples", "p50_ms", "p95_ms", "p99_ms", "available", "placeholder_reason"} {
			if _, ok := series[field]; !ok {
				t.Fatalf("canonical metric %q missing %q: %#v", want, field, series)
			}
		}
	}
}

func assertProviderLatencyBenchStageAvailability(t *testing.T, report map[string]any) {
	t.Helper()
	rawStages, ok := report["stage_availability"].([]any)
	if !ok || len(rawStages) == 0 {
		t.Fatalf("stage_availability missing or empty: %#v", report["stage_availability"])
	}
	summary, ok := report["summary"].(map[string]any)
	if !ok || len(summary) == 0 {
		t.Fatalf("summary missing or empty: %#v", report["summary"])
	}
	seenStages := map[string]bool{}
	for _, rawStage := range rawStages {
		stage, ok := rawStage.(map[string]any)
		if !ok {
			t.Fatalf("stage availability has unexpected shape: %#v", rawStage)
		}
		name, _ := stage["stage"].(string)
		reason, _ := stage["placeholder_reason"].(string)
		if name == "" || reason == "" {
			t.Fatalf("stage availability missing stage/reason: %#v", stage)
		}
		if available, _ := stage["available"].(bool); available {
			t.Fatalf("stage %q unexpectedly available in report-shape scaffold: %#v", name, stage)
		}
		if placeholder, _ := stage["placeholder"].(bool); !placeholder {
			t.Fatalf("stage %q should be marked placeholder: %#v", name, stage)
		}
		seenStages[name] = true
	}
	for metric := range summary {
		if !seenStages[metric] {
			t.Fatalf("stage_availability missing summary metric %q: %#v", metric, rawStages)
		}
	}
	for _, want := range []string{
		"asr_first_partial_ms",
		"provider_first_byte_ms",
		"provider_first_content_ms",
		"tts_first_audio_ms",
		"downlink_first_frame_ms",
		"audio_downlink_first_frame_ms",
		"device_playback_start_ms",
		"barge_in_stop_ms",
		"provider_cancel_ms",
		"playback_stop_ms",
	} {
		if !seenStages[want] {
			t.Fatalf("stage_availability missing %q: %#v", want, rawStages)
		}
	}
}

func TestRunProviderLatencyBenchFixtureRedactsFixturePath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--fixture", "/tmp/a21/private/audio-fixture.wav", "--iterations", "1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"execution_mode": "fixture"`,
		`"fixture_id": "audio-fixture.wav"`,
		`"profile": "mock"`,
		`"family": "mock"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"/tmp/a21", "private", "audio-fixture.wav/"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked fixture path fragment %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunProviderLatencyBenchFixtureReadsRedactedAudioMetadataSidecar(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-redacted-audio-fixture.json")
	data := `{
  "schema_version": "a21.provider_latency_fixture.v1",
  "identity": "a21_fixture_alpha",
  "audio": {
    "format": "pcm_s16le",
    "sample_rate_hz": 16000,
    "channels": 1,
    "duration_ms": 1200
  },
  "sample": {
    "sample_count": 19200
  },
  "window": {
    "window_ms": 20,
    "window_count": 60
  }
}`
	if err := os.WriteFile(fixture, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--fixture", fixture, "--iterations", "1"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"execution_mode": "fixture"`,
		`"fixture_id": "a21-redacted-audio-fixture.json"`,
		`"schema_version": "a21.provider_latency_fixture.v1"`,
		`"identity": "a21_fixture_alpha"`,
		`"format": "pcm_s16le"`,
		`"sample_rate_hz": 16000`,
		`"channels": 1`,
		`"duration_ms": 1200`,
		`"sample_count": 19200`,
		`"window_ms": 20`,
		`"window_count": 60`,
		`"failure_count": 0`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{dir, "raw_pcm", "data_base64", "prompt", "transcript", "provider output", "reasoning", "http://"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, stdout.String())
		}
	}
}
