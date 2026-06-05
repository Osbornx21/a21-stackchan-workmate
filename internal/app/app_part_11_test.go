package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunProviderLatencyBenchFixtureReportsInvalidSidecarWithoutPayloadLeak(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-invalid-audio-fixture.json")
	data := `{
  "schema_version": "a21.provider_latency_fixture.v1",
  "identity": "/tmp/private/leaky-fixture",
  "audio": {
    "format": "pcm_s16le",
    "sample_rate_hz": 0,
    "channels": 1
  },
  "sample": {},
  "window": {
    "window_ms": 20
  },
  "prompt": "secret prompt text",
  "transcript": "secret transcript text",
  "raw_pcm": "secret pcm bytes",
  "proxy_url": "http://user:pass@example.invalid:8080"
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
		`"fixture_id": "a21-invalid-audio-fixture.json"`,
		`"findings"`,
		`"code": "fixture_sidecar_invalid"`,
		`"message": "fixture metadata sidecar is invalid or unsafe"`,
		`"failure_count": 1`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{dir, "/tmp/private", "leaky-fixture", "secret", "example.invalid", "8080", "user:pass"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunProviderLatencyBenchFixtureRejectsOversizedSidecarWithoutPayloadLeak(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-oversized-audio-fixture.json")
	payload := strings.Repeat("secret prompt transcript raw_pcm data_base64 http://user:pass@example.invalid:8080 /tmp/a21/private sk-a21-secret\n", 700)
	if err := os.WriteFile(fixture, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--fixture", fixture, "--iterations", "1"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	assertProviderLatencyBenchInvalidSidecarFinding(t, stdout.Bytes(), "a21-oversized-audio-fixture.json")
	for _, forbidden := range []string{dir, payload, "secret", "prompt", "transcript", "raw_pcm", "data_base64", "example.invalid", "8080", "user:pass", "/tmp/a21/private", "sk-a21-secret"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunProviderLatencyBenchFixtureRejectsUnknownFieldSidecarWithoutValueLeak(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-unknown-field-audio-fixture.json")
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
  },
  "unexpected_metadata": "secret prompt transcript raw_pcm data_base64 http://user:pass@example.invalid:8080 /tmp/a21/private sk-a21-secret"
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
	assertProviderLatencyBenchInvalidSidecarFinding(t, stdout.Bytes(), "a21-unknown-field-audio-fixture.json")
	for _, forbidden := range []string{dir, "unexpected_metadata", "secret", "prompt", "transcript", "raw_pcm", "data_base64", "example.invalid", "8080", "user:pass", "/tmp/a21/private", "sk-a21-secret"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, stdout.String())
		}
	}
}

func assertProviderLatencyBenchInvalidSidecarFinding(t *testing.T, reportJSON []byte, fixtureID string) {
	t.Helper()
	var report providerLatencyBenchReport
	if err := json.Unmarshal(reportJSON, &report); err != nil {
		t.Fatalf("decode provider latency report: %v\n%s", err, string(reportJSON))
	}
	if report.ExecutionMode != "fixture" {
		t.Fatalf("execution mode = %q, want fixture: %s", report.ExecutionMode, string(reportJSON))
	}
	if report.Fixture == nil {
		t.Fatalf("fixture missing: %s", string(reportJSON))
	}
	if report.Fixture.FixtureID != fixtureID {
		t.Fatalf("fixture id = %q, want %q: %s", report.Fixture.FixtureID, fixtureID, string(reportJSON))
	}
	if report.Fixture.Metadata != nil {
		t.Fatalf("metadata should not be retained for invalid sidecar: %#v", report.Fixture.Metadata)
	}
	if report.Counts.FailureCount != 1 {
		t.Fatalf("failure count = %d, want 1: %s", report.Counts.FailureCount, string(reportJSON))
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings = %d, want 1: %s", len(report.Findings), string(reportJSON))
	}
	finding := report.Findings[0]
	if finding.Code != "fixture_sidecar_invalid" || finding.Message != "fixture metadata sidecar is invalid or unsafe" {
		t.Fatalf("finding = %#v, want fixed redacted invalid sidecar finding", finding)
	}
	if report.Redaction.PayloadsStored ||
		report.Redaction.CredentialValuesStored ||
		report.Redaction.FullURLsStored ||
		report.Redaction.LocalPathsStored {
		t.Fatalf("redaction flags indicate stored sensitive material: %#v", report.Redaction)
	}
	if report.Execution.ProviderExecuted || report.Execution.V21Executed || report.Execution.HardwareExecuted {
		t.Fatalf("execution flags indicate forbidden execution: %#v", report.Execution)
	}
}

func TestRunProviderLatencyBenchFixtureRejectsTrailingSidecarPayload(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-trailing-payload-fixture.json")
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
}
{"prompt": "secret trailing prompt"}`
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
		`"fixture_id": "a21-trailing-payload-fixture.json"`,
		`"code": "fixture_sidecar_invalid"`,
		`"failure_count": 1`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{dir, "secret trailing prompt", `"prompt"`} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunProviderLatencyBenchHostLoopbackUsesCanonicalMode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--iterations", "1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"execution_mode": "host_loopback"`) {
		t.Fatalf("stdout missing canonical host_loopback mode: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "host_baseline") {
		t.Fatalf("stdout contains deprecated host_baseline mode: %s", stdout.String())
	}
}

func TestRunProviderLatencyBenchIngestsHostLoopbackReport(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-host-loopback-report.json")
	writeProviderLatencyBenchHostLoopbackFixture(t, fixture)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report providerLatencyBenchReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider latency report: %v\n%s", err, stdout.String())
	}
	if report.ExecutionMode != "host_loopback" || report.BaselineScope != "host_only" {
		t.Fatalf("wrong host-loopback identity: mode=%q scope=%q", report.ExecutionMode, report.BaselineScope)
	}
	if report.Fixture == nil || report.Fixture.FixtureID != "a21-host-loopback-report.json" {
		t.Fatalf("fixture id not redacted to basename: %#v", report.Fixture)
	}
	for _, stage := range []string{"asr_first_partial_ms", "asr_final_ms", "llm_first_content_ms", "tts_first_audio_ms", "audio_downlink_first_frame_ms", "barge_in_stop_ms"} {
		availability := providerLatencyBenchStageByName(t, report, stage)
		if !availability.Available || availability.Placeholder {
			t.Fatalf("stage %s availability = available:%v placeholder:%v, want measured host evidence", stage, availability.Available, availability.Placeholder)
		}
	}
	playback := providerLatencyBenchStageByName(t, report, "device_playback_start_ms")
	if playback.Available || playback.Placeholder {
		t.Fatalf("physical playback should remain unavailable without physical evidence: %#v", playback)
	}
	answer := report.CanonicalMetrics["answer_first_audio_p95_ms"]
	if !answer.Available || answer.P95MS >= 1500 || answer.SourceStage != "answer_first_audio_ms" {
		t.Fatalf("answer_first_audio_p95_ms = %#v, want host-only p95 < 1500", answer)
	}
	barge := report.CanonicalMetrics["barge_in_stop_p95_ms"]
	if !barge.Available || barge.P95MS >= 300 || barge.SourceStage != "barge_in_stop_ms" {
		t.Fatalf("barge_in_stop_p95_ms = %#v, want host-only p95 < 300", barge)
	}
	physical := report.CanonicalMetrics["speech_end_to_first_audible_response_ms"]
	if physical.Available || physical.Placeholder {
		t.Fatalf("physical audible response should not be available from host-only fixture: %#v", physical)
	}
	if report.AcceptanceStatus != "candidate_host_only" || report.PRDAccepted {
		t.Fatalf("acceptance = %q prd=%v, want host-only candidate without PRD acceptance", report.AcceptanceStatus, report.PRDAccepted)
	}
	if report.Execution.ProviderExecuted || report.Execution.V21Executed || report.Execution.HardwareExecuted {
		t.Fatalf("host-loopback ingestion should not mark execution flags true: %#v", report.Execution)
	}
}

func TestRunProviderLatencyBenchBlocksHostLoopbackCandidateOnTTSAudioQualityWarning(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-host-loopback-audio-quality-warning.json")
	writeProviderLatencyBenchHostLoopbackFixture(t, fixture)
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	payload["tts_audio_quality"] = map[string]any{
		"status":         "warning",
		"codec":          "pcm_s16le",
		"sample_rate_hz": 16000,
		"channels":       1,
		"peak_abs":       32767,
		"findings": []string{
			"audio_quality_clipping_detected",
			"audio_quality_low_headroom",
		},
	}
	data, err = json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture, data, 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report providerLatencyBenchReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider latency report: %v\n%s", err, stdout.String())
	}
	answer := report.CanonicalMetrics["answer_first_audio_p95_ms"]
	if !answer.Available || answer.P95MS >= 1500 {
		t.Fatalf("answer_first_audio_p95_ms = %#v, want latency evidence preserved", answer)
	}
	barge := report.CanonicalMetrics["barge_in_stop_p95_ms"]
	if !barge.Available || barge.P95MS >= 300 {
		t.Fatalf("barge_in_stop_p95_ms = %#v, want barge-in evidence preserved", barge)
	}
	if report.AcceptanceStatus == "candidate_host_only" || report.PRDAccepted {
		t.Fatalf("acceptance = %q prd=%v, want audio quality warning to block host-only candidate", report.AcceptanceStatus, report.PRDAccepted)
	}
	if !providerLatencyBenchHasFinding(report, "host_loopback_tts_audio_quality_failed") {
		t.Fatalf("findings = %#v, want TTS audio quality gate finding", report.Findings)
	}
	if report.Counts.FailureCount != 2 {
		t.Fatalf("failure count = %d, want TTS quality plus physical playback finding: %#v", report.Counts.FailureCount, report.Findings)
	}
	rendered := stdout.String()
	for _, forbidden := range []string{dir, fixture, "32767", "pcm_s16le", "audio_quality_clipping_detected", "audio_quality_low_headroom"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout leaked audio-quality detail %q: %s", forbidden, rendered)
		}
	}
}

func TestRunProviderLatencyBenchIngestsHostLocalXiaozhiExecutionWithoutAcceptance(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-xiaozhi-host-local-slow-report.json")
	data := `{
  "schema_version": "a21.xiaozhi_voice_bench.v1",
  "execution_mode": "host_loopback",
  "baseline_scope": "host_only",
  "device_id": "stackchan-virtual-a21-bench-001",
  "acceptance_status": "blocked",
  "prd_accepted": false,
  "execution": {
    "provider_executed": false,
    "v21_executed": false,
    "hardware_executed": false,
    "voice_pipeline_observed": true,
    "voice_pipeline_execution_mode": "host_local",
    "asr_profile": "local_sherpa_onnx",
    "asr_profile_env": "A21_ASR_PROFILE",
    "llm_profile": "ollama_local",
    "llm_profile_env": "A21_LLM_PROFILE",
    "tts_profile": "sherpa_onnx_tts",
    "tts_profile_env": "A21_TTS_PROFILE",
    "host_local_asr_executed": true,
    "host_local_text_executed": true,
    "host_local_tts_executed": true
  },
  "answer_turns": [
    {"trace_summary": {"xiaozhi_listen_to_audio_ingress_ms": 30, "xiaozhi_opus_decode_ms": 44, "asr_first_partial_ms": 120, "asr_final_ms": 180, "llm_first_content_ms": 400, "tts_first_audio_ms": 900, "audio_downlink_first_frame_ms": 1200, "answer_first_audio_total_ms": 1600}},
    {"trace_summary": {"xiaozhi_listen_to_audio_ingress_ms": 31, "xiaozhi_opus_decode_ms": 45, "asr_first_partial_ms": 122, "asr_final_ms": 185, "llm_first_content_ms": 420, "tts_first_audio_ms": 920, "audio_downlink_first_frame_ms": 1250, "answer_first_audio_total_ms": 1700}},
    {"trace_summary": {"xiaozhi_listen_to_audio_ingress_ms": 32, "xiaozhi_opus_decode_ms": 46, "asr_first_partial_ms": 124, "asr_final_ms": 190, "llm_first_content_ms": 440, "tts_first_audio_ms": 940, "audio_downlink_first_frame_ms": 1300, "answer_first_audio_total_ms": 1800}}
  ],
  "barge_in_turns": [
    {"trace_summary": {"barge_in_detected_ms": 20, "provider_cancel_ms": 45, "provider_cancel_done_ms": 50, "playback_stop_ms": 170, "playback_stop_done_ms": 180, "barge_in_stop_ms": 180}},
    {"trace_summary": {"barge_in_detected_ms": 22, "provider_cancel_ms": 47, "provider_cancel_done_ms": 52, "playback_stop_ms": 185, "playback_stop_done_ms": 195, "barge_in_stop_ms": 195}},
    {"trace_summary": {"barge_in_detected_ms": 24, "provider_cancel_ms": 49, "provider_cancel_done_ms": 54, "playback_stop_ms": 190, "playback_stop_done_ms": 205, "barge_in_stop_ms": 205}}
  ]
}`
	if err := os.WriteFile(fixture, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report providerLatencyBenchReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider latency report: %v\n%s", err, stdout.String())
	}
	answer := report.CanonicalMetrics["answer_first_audio_p95_ms"]
	if !answer.Available || answer.P95MS != 1800 {
		t.Fatalf("answer_first_audio_p95_ms = %#v, want available slow host-local p95", answer)
	}
	asrFinal := providerLatencyBenchStageByName(t, report, "asr_final_ms")
	if !asrFinal.Available || asrFinal.Samples != 3 || asrFinal.P95MS != 190 {
		t.Fatalf("asr final stage = %#v, want available from xiaozhi trace summary", asrFinal)
	}
	playback := providerLatencyBenchStageByName(t, report, "device_playback_start_ms")
	if playback.Available || playback.Placeholder {
		t.Fatalf("physical playback should stay unavailable without physical evidence: %#v", playback)
	}
	if report.AcceptanceStatus != "not_accepted" || report.PromotionGate != "not_production" || report.PRDAccepted {
		t.Fatalf("acceptance=%q gate=%q prd=%v, want not accepted/not production", report.AcceptanceStatus, report.PromotionGate, report.PRDAccepted)
	}
	if report.Execution.ProviderExecuted || report.Execution.V21Executed || report.Execution.HardwareExecuted {
		t.Fatalf("host-local fixture must not claim provider/v21/hardware execution: %#v", report.Execution)
	}
	if report.Execution.VoicePipelineExecutionMode != "host_local" ||
		!report.Execution.HostLocalASRExecuted ||
		!report.Execution.HostLocalTextExecuted ||
		!report.Execution.HostLocalTTSExecuted ||
		report.Execution.ASRProfile != "local_sherpa_onnx" ||
		report.Execution.ASRProfileEnv != "A21_ASR_PROFILE" ||
		report.Execution.LLMProfile != "ollama_local" ||
		report.Execution.LLMProfileEnv != "A21_LLM_PROFILE" ||
		report.Execution.TTSProfile != "sherpa_onnx_tts" ||
		report.Execution.TTSProfileEnv != "A21_TTS_PROFILE" {
		t.Fatalf("execution semantics = %#v, want honest host-local adapter evidence", report.Execution)
	}
	rendered := stdout.String()
	for _, forbidden := range []string{dir, fixture, `"prd_accepted": true`, `"provider_executed": true`, `"hardware_executed": true`} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout leaked or overclaimed %q: %s", forbidden, rendered)
		}
	}
}

func TestRunProviderLatencyBenchHostLoopbackRedactsUnsafeReportPayload(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-host-loopback-leaky-report.json")
	data := `{
  "schema_version": "a21.xiaozhi_voice_bench.v1",
  "execution_mode": "host_loopback",
  "gateway": "http://user:pass@example.invalid:21080",
  "answer_turns": [
    {
      "trace_summary": {
        "asr_first_partial_ms": 100,
        "prompt": "secret prompt",
        "transcript": "secret transcript",
        "provider_output": "secret provider output",
        "data_base64": "c2VjcmV0"
      }
    }
  ]
}`
	if err := os.WriteFile(fixture, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"fixture_id": "a21-host-loopback-leaky-report.json"`,
		`"code": "host_loopback_report_invalid"`,
		`"failure_count": 1`,
		`"local_paths_stored": false`,
		`"full_urls_stored": false`,
		`"payloads_stored": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{dir, fixture, "http://", "example.invalid", "user:pass", "secret", "prompt", "transcript", "provider_output", "data_base64", "c2VjcmV0"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, rendered)
		}
	}
}

func TestRunProviderLatencyBenchHostLoopbackPartialReportFindsMissingStages(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-host-loopback-partial-report.json")
	data := `{
  "schema_version": "a21.xiaozhi_voice_bench.v1",
  "execution_mode": "host_loopback",
  "baseline_scope": "host_only",
  "answer_turns": [
    {
      "trace_summary": {
        "asr_first_partial_ms": 111
      }
    }
  ]
}`
	if err := os.WriteFile(fixture, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report providerLatencyBenchReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider latency report: %v\n%s", err, stdout.String())
	}
	asr := providerLatencyBenchStageByName(t, report, "asr_first_partial_ms")
	if !asr.Available {
		t.Fatalf("partial report should keep available ASR timing: %#v", asr)
	}
	tts := providerLatencyBenchStageByName(t, report, "tts_first_audio_ms")
	if tts.Available {
		t.Fatalf("missing TTS timing should be unavailable: %#v", tts)
	}
	if report.Counts.FailureCount == 0 || len(report.Findings) == 0 {
		t.Fatalf("partial report should include honest findings: %#v", report.Findings)
	}
	for _, finding := range report.Findings {
		if strings.Contains(finding.Message, dir) || strings.Contains(finding.Message, "trace_summary") {
			t.Fatalf("finding should stay redacted and low-information: %#v", finding)
		}
	}
}

func TestRunProviderLatencyBenchIngestsVirtualXiaozhiHarnessReport(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-virtual-xiaozhi-report.json")
	writeProviderLatencyBenchVirtualXiaozhiFixture(t, fixture)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report providerLatencyBenchReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider latency report: %v\n%s", err, stdout.String())
	}
	if report.Fixture == nil || report.Fixture.FixtureID != "a21-virtual-xiaozhi-report.json" {
		t.Fatalf("fixture id not redacted to basename: %#v", report.Fixture)
	}
	answer := report.CanonicalMetrics["answer_first_audio_p95_ms"]
	if !answer.Available || answer.P95MS != 1200 || answer.SourceStage != "answer_first_audio_ms" {
		t.Fatalf("answer_first_audio_p95_ms = %#v, want virtual harness p95", answer)
	}
	barge := report.CanonicalMetrics["barge_in_stop_p95_ms"]
	if !barge.Available || barge.P95MS != 260 || barge.SourceStage != "barge_in_stop_ms" {
		t.Fatalf("barge_in_stop_p95_ms = %#v, want virtual harness abort-stop p95", barge)
	}
	answerStage := providerLatencyBenchStageByName(t, report, "answer_first_audio_ms")
	if !answerStage.Available || answerStage.Samples != 3 || answerStage.P95MS != 1200 {
		t.Fatalf("answer stage = %#v, want three virtual first-audio samples", answerStage)
	}
	bargeStage := providerLatencyBenchStageByName(t, report, "barge_in_stop_ms")
	if !bargeStage.Available || bargeStage.Samples != 3 || bargeStage.P95MS != 260 {
		t.Fatalf("barge stage = %#v, want three virtual abort-stop samples", bargeStage)
	}
	physicalStage := providerLatencyBenchStageByName(t, report, "device_playback_start_ms")
	if physicalStage.Available || physicalStage.Placeholder {
		t.Fatalf("physical playback should remain unavailable without physical evidence: %#v", physicalStage)
	}
	physicalMetric := report.CanonicalMetrics["speech_end_to_first_audible_response_ms"]
	if physicalMetric.Available || physicalMetric.Placeholder {
		t.Fatalf("physical audible response should not be available from virtual fixture: %#v", physicalMetric)
	}
	if report.PRDAccepted {
		t.Fatalf("virtual host harness must not claim PRD acceptance")
	}
	if report.Execution.ProviderExecuted || report.Execution.V21Executed || report.Execution.HardwareExecuted {
		t.Fatalf("virtual harness ingestion should not mark execution flags true: %#v", report.Execution)
	}
}

func TestRunProviderLatencyBenchVirtualXiaozhiPartialReportFindsMissingAbortStop(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-virtual-xiaozhi-partial.json")
	data := `{
  "schema": "a21.virtual_xiaozhi_harness.v1",
  "target": "127.0.0.1:21080/v1/xiaozhi",
  "profile": "xiaozhi",
  "device_id": "stackchan-virtual-a21-bench-001",
  "trace_id": "a21-trace-virtual-001",
  "session_id": "a21-session-virtual-001",
  "runs": 3,
  "successful_runs": 3,
  "first_audio_samples_ms": [820, 930, 1200],
  "first_audio_p95_ms": 1200,
  "abort_stop_samples_ms": [],
  "abort_stop_p95_ms": null,
  "host_candidate": false,
  "prd_accepted": false
}`
	if err := os.WriteFile(fixture, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report providerLatencyBenchReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider latency report: %v\n%s", err, stdout.String())
	}
	answer := report.CanonicalMetrics["answer_first_audio_p95_ms"]
	if !answer.Available || answer.P95MS != 1200 {
		t.Fatalf("partial virtual report should keep first-audio timing: %#v", answer)
	}
	barge := report.CanonicalMetrics["barge_in_stop_p95_ms"]
	if barge.Available || barge.Samples != 0 {
		t.Fatalf("missing abort-stop should be unavailable: %#v", barge)
	}
	if report.Counts.FailureCount == 0 || len(report.Findings) == 0 {
		t.Fatalf("partial virtual report should include honest findings: %#v", report.Findings)
	}
}

func TestRunProviderLatencyBenchVirtualXiaozhiAcceptsZeroAbortStopSamples(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-virtual-xiaozhi-zero-abort.json")
	data := `{
  "schema": "a21.virtual_xiaozhi_harness.v1",
  "target": "127.0.0.1:21080/v1/xiaozhi",
  "profile": "xiaozhi",
  "device_id": "stackchan-virtual-a21-bench-001",
  "trace_id": "a21-trace-virtual-001",
  "session_id": "a21-session-virtual-001",
  "runs": 3,
  "successful_runs": 3,
  "first_audio_samples_ms": [1032, 1123, 1223],
  "first_audio_p95_ms": 1223,
  "abort_stop_samples_ms": [0, 0, 0],
  "abort_stop_p95_ms": 0,
  "host_candidate": true,
  "prd_accepted": false
}`
	if err := os.WriteFile(fixture, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	var report providerLatencyBenchReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode provider latency report: %v\n%s", err, stdout.String())
	}
	barge := report.CanonicalMetrics["barge_in_stop_p95_ms"]
	if !barge.Available || barge.Samples != 3 || barge.P95MS != 0 {
		t.Fatalf("zero abort-stop samples should be valid observed timings: %#v", barge)
	}
	bargeStage := providerLatencyBenchStageByName(t, report, "barge_in_stop_ms")
	if !bargeStage.Available || bargeStage.Samples != 3 || bargeStage.P95MS != 0 {
		t.Fatalf("zero abort-stop stage should be available: %#v", bargeStage)
	}
	if report.PRDAccepted {
		t.Fatalf("virtual host harness must not claim PRD acceptance")
	}
}

func TestRunProviderLatencyBenchVirtualXiaozhiRedactsUnsafeReportPayload(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-virtual-xiaozhi-leaky.json")
	data := `{
  "schema": "a21.virtual_xiaozhi_harness.v1",
  "target": "https://user:pass@example.invalid/v1/xiaozhi",
  "profile": "xiaozhi",
  "device_id": "stackchan-virtual-a21-bench-001",
  "trace_id": "a21-trace-virtual-001",
  "session_id": "a21-session-virtual-001",
  "first_audio_samples_ms": [820, 930, 1200],
  "first_audio_p95_ms": 1200,
  "abort_stop_samples_ms": [180, 220, 260],
  "abort_stop_p95_ms": 260,
  "run_reports": [
    {
      "prompt": "secret prompt",
      "transcript": "secret transcript",
      "provider_output": "secret provider output",
      "data_base64": "c2VjcmV0",
      "proxy_url": "http://user:pass@example.invalid:7890"
    }
  ],
  "prd_accepted": false
}`
	if err := os.WriteFile(fixture, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"fixture_id": "a21-virtual-xiaozhi-leaky.json"`,
		`"code": "host_loopback_report_invalid"`,
		`"failure_count": 1`,
		`"local_paths_stored": false`,
		`"full_urls_stored": false`,
		`"payloads_stored": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{dir, fixture, "https://", "http://", "example.invalid", "user:pass", "secret", "prompt", "transcript", "provider_output", "data_base64", "proxy_url", "c2VjcmV0"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, rendered)
		}
	}
}

func writeProviderLatencyBenchVirtualXiaozhiFixture(t *testing.T, path string) {
	t.Helper()
	data := `{
  "schema": "a21.virtual_xiaozhi_harness.v1",
  "target": "127.0.0.1:21080/v1/xiaozhi",
  "profile": "xiaozhi",
  "device_id": "stackchan-virtual-a21-bench-001",
  "trace_id": "a21-trace-virtual-001",
  "session_id": "a21-session-virtual-001",
  "protocol_version": 1,
  "runs": 3,
  "successful_runs": 3,
  "exit_codes": [0, 0, 0],
  "first_audio_samples_ms": [820, 930, 1200],
  "first_audio_p95_ms": 1200,
  "abort_stop_samples_ms": [180, 220, 260],
  "abort_stop_p95_ms": 260,
  "host_candidate": true,
  "prd_accepted": false
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeProviderLatencyBenchHostLoopbackFixture(t *testing.T, path string) {
	t.Helper()
	data := `{
  "schema_version": "a21.xiaozhi_voice_bench.v1",
  "execution_mode": "host_loopback",
  "baseline_scope": "host_only",
  "device_id": "stackchan-virtual-a21-bench-001",
  "answer_turns": [
    {
      "trace_id": "a21-trace-host-answer-01",
      "session_id": "a21-session-host-answer-01",
      "trace_summary": {
        "xiaozhi_listen_to_audio_ingress_ms": 30,
        "xiaozhi_opus_decode_ms": 44,
        "asr_first_partial_ms": 120,
        "asr_final_ms": 180,
        "provider_first_byte_ms": 220,
        "provider_first_content_ms": 260,
        "llm_first_content_ms": 260,
        "tts_first_audio_ms": 410,
        "audio_downlink_first_frame_ms": 460,
        "answer_first_audio_total_ms": 900
      }
    },
    {
      "trace_id": "a21-trace-host-answer-02",
      "session_id": "a21-session-host-answer-02",
      "trace_summary": {
        "xiaozhi_listen_to_audio_ingress_ms": 32,
        "xiaozhi_opus_decode_ms": 46,
        "asr_first_partial_ms": 125,
        "asr_final_ms": 185,
        "provider_first_byte_ms": 225,
        "provider_first_content_ms": 270,
        "llm_first_content_ms": 270,
        "tts_first_audio_ms": 420,
        "audio_downlink_first_frame_ms": 470,
        "answer_first_audio_total_ms": 940
      }
    },
    {
      "trace_id": "a21-trace-host-answer-03",
      "session_id": "a21-session-host-answer-03",
      "trace_summary": {
        "xiaozhi_listen_to_audio_ingress_ms": 34,
        "xiaozhi_opus_decode_ms": 48,
        "asr_first_partial_ms": 130,
        "asr_final_ms": 190,
        "provider_first_byte_ms": 230,
        "provider_first_content_ms": 280,
        "llm_first_content_ms": 280,
        "tts_first_audio_ms": 430,
        "audio_downlink_first_frame_ms": 480,
        "answer_first_audio_total_ms": 980
      }
    }
  ],
  "barge_in_turns": [
    {"trace_summary": {"barge_in_detected_ms": 20, "provider_cancel_ms": 45, "provider_cancel_done_ms": 50, "playback_stop_ms": 170, "playback_stop_done_ms": 180, "barge_in_stop_ms": 180}},
    {"trace_summary": {"barge_in_detected_ms": 22, "provider_cancel_ms": 47, "provider_cancel_done_ms": 52, "playback_stop_ms": 185, "playback_stop_done_ms": 195, "barge_in_stop_ms": 195}},
    {"trace_summary": {"barge_in_detected_ms": 24, "provider_cancel_ms": 49, "provider_cancel_done_ms": 54, "playback_stop_ms": 190, "playback_stop_done_ms": 205, "barge_in_stop_ms": 205}}
  ]
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func providerLatencyBenchStageByName(t *testing.T, report providerLatencyBenchReport, name string) providerLatencyBenchStageAvailability {
	t.Helper()
	for _, stage := range report.StageAvailability {
		if stage.Stage == name {
			return stage
		}
	}
	t.Fatalf("stage %q missing: %#v", name, report.StageAvailability)
	return providerLatencyBenchStageAvailability{}
}

func providerLatencyBenchHasFinding(report providerLatencyBenchReport, code string) bool {
	for _, finding := range report.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
