package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"a21.local/a21/internal/providers"
)

func TestRunLANProbeRejectsCredentialTargetWithoutEchoingSecret(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"lan-probe",
		"--target", "a21-gateway=http://user:secret-token@127.0.0.1:21080",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "must not include credentials") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	for _, forbidden := range []string{"secret-token", "user:", "127.0.0.1:21080"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("lan probe leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunLANProbeRejectsLegacyInternalPortWithoutEchoingTarget(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"lan-probe",
		"--target", "a21-gateway=127.0.0.1:18080",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "legacy internal port") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	for _, forbidden := range []string{"a21-gateway", "127.0.0.1", "18080"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("lan probe leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunProviderRealtimePlanDoubaoTTSDoesNotLeakSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "zh_female_kailangjiejie_moon_bigtts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-plan", "--provider", "doubao_tts_realtime"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "doubao_tts_realtime"`,
		`"protocol": "websocket_realtime"`,
		`"status": "ready"`,
		`"configured": true`,
		`"executed": false`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
		`"api_key_env": "A21_DOUBAO_API_KEY|A21_DOUBAO_ACCESS_TOKEN"`,
		`"model_env": "A21_DOUBAO_TTS_MODEL"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie_moon_bigtts", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("provider realtime plan leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunProviderRealtimePlanDoubaoS2SDoesNotLeakSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_APP_ID", "app-a21-secret")
	t.Setenv("A21_DOUBAO_RESOURCE_ID", "resource-a21-secret")
	t.Setenv("A21_DOUBAO_REALTIME_MODEL", "doubao-s2s")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-plan", "--provider", "doubao_realtime"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "doubao_realtime"`,
		`"protocol": "websocket_realtime"`,
		`"status": "ready"`,
		`"configured": true`,
		`"executed": false`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
		`"api_key_env": "A21_DOUBAO_API_KEY"`,
		`"model_env": "A21_DOUBAO_REALTIME_MODEL"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("provider realtime plan leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunProviderRealtimePlanRejectsLegacyProviderWithoutEchoingValue(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-plan", "--provider", "x21_voice"}, &stdout, &stderr)
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

func TestRunProviderRealtimePlanRejectsExecuteFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-plan", "--execute"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "does not support --execute") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunAudioFrontEndPlanListsMatureCandidates(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-plan"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"status": "plan_only"`,
		`"id": "webrtc_apm"`,
		`"id": "provider_side_vad"`,
		`"id": "silero_vad"`,
		`"id": "a21_rms_vad"`,
		`"status": "dev_only"`,
		`"a21_vad_detector_decisions_total{detector,result}"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"x21", "v21", "https://", "http://"} {
		if strings.Contains(strings.ToLower(stdout.String()), forbidden) {
			t.Fatalf("audio front-end plan leaked forbidden identity %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunAudioFrontEndEvalMockReportsQualityMetrics(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-eval", "--mock"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"status": "mock_only"`,
		`"dataset": "a21_mock_vad_fixture_v1"`,
		`"detector": "a21-rms-vad"`,
		`"frames_total": 5`,
		`"true_positive": 2`,
		`"true_negative": 3`,
		`"false_positive": 0`,
		`"false_negative": 0`,
		`"speech_start_lag_ms": 0`,
		`"speech_end_lag_ms": 20`,
		`"baseline_scope": "host_only"`,
		`"device_id": "none_host_fixture"`,
		`"candidate_evidence"`,
		`"fast_companion_evidence_contract"`,
		`"provider_executed": false`,
		`"v21_executed": false`,
		`"hardware_executed": false`,
		`"raw_audio_stored": false`,
		`"transcripts_stored": false`,
		`"promotion_gate": "not_production"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, want := range []string{
		`"id": "webrtc_apm"`,
		`"id": "provider_side_vad"`,
		`"id": "silero_vad"`,
		`"id": "a21_rms_vad"`,
		`"available": false`,
		`"placeholder": true`,
		`"id": "speaker_to_mic_echo_report"`,
		`"id": "barge_in_stop_timing"`,
		`"id": "first_audio_waterfall_impact"`,
		`"id": "metrics_continuity"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing evidence contract %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"https://", "http://", "pcm_s16le_base64"} {
		if strings.Contains(strings.ToLower(stdout.String()), forbidden) {
			t.Fatalf("stdout leaked forbidden fragment %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunAudioFrontEndEvalRequiresMockFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-eval"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "requires --mock") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunAudioFrontEndEvalFixtureReportsQualityMetrics(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-front-end-fixture.json")
	data := `{
  "schema_version": "a21.audio.frontend_fixture.v1",
  "dataset": "a21_cli_fixture_test",
  "detector": "a21-rms-vad",
  "sample_rate_hz": 16000,
  "channels": 1,
  "duration_ms": 20,
  "frames": [
    {"seq": 1, "expected_speech": false, "pcm_s16le_base64": "` + appTestPCM16Base64(0) + `"},
    {"seq": 2, "expected_speech": true, "pcm_s16le_base64": "` + appTestPCM16Base64(12000) + `"},
    {"seq": 3, "expected_speech": false, "pcm_s16le_base64": "` + appTestPCM16Base64(0) + `"},
    {"seq": 4, "expected_speech": false, "pcm_s16le_base64": "` + appTestPCM16Base64(0) + `"}
  ]
}`
	if err := os.WriteFile(fixture, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-eval", "--fixture", fixture}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"status": "fixture"`,
		`"dataset": "a21_cli_fixture_test"`,
		`"frames_total": 4`,
		`"true_positive": 1`,
		`"true_negative": 3`,
		`"speech_start_lag_ms": 0`,
		`"speech_end_lag_ms": 20`,
		`"promotion_gate": "requires_recorded_office_and_physical_acceptance"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunAudioFrontEndEvalFixtureReadErrorRedactsFullPath(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-secret-fixture-missing.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"audio-front-end-eval", "--fixture", fixture}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), filepath.Base(fixture)) {
		t.Fatalf("stderr missing fixture basename: %s", stderr.String())
	}
	for _, forbidden := range []string{fixture, dir} {
		if strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("stderr leaked full fixture path %q: %s", forbidden, stderr.String())
		}
	}
}

func TestRunAudioFrontEndEvalWritesReportArtifact(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-eval", "--mock", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-audio-front-end-eval-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("report files = %d, want 1", len(matches))
	}
	if !strings.Contains(stdout.String(), `"report_path":`) || !strings.Contains(stdout.String(), filepath.Base(matches[0])) {
		t.Fatalf("stdout missing report path: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), dir) {
		t.Fatalf("stdout leaked full local report path %q: %s", dir, stdout.String())
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"schema_version": "a21.audio.frontend_eval.v1"`, `"status": "mock_only"`, `"speech_start_lag_ms": 0`, `"speech_end_lag_ms": 20`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("report file missing %q: %s", want, data)
		}
	}
	for _, forbidden := range []string{"pcm_s16le_base64", "x21"} {
		if strings.Contains(strings.ToLower(string(data)), forbidden) {
			t.Fatalf("report file leaked %q: %s", forbidden, data)
		}
	}
	if strings.Contains(string(data), dir) {
		t.Fatalf("report file leaked full local report path %q: %s", dir, data)
	}
}

func TestRunAudioFrontEndEvalReportIncludesTraceableEnvironmentMetadataWithoutProxySecrets(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://user:secret@example.invalid:8080")
	t.Setenv("A21_PROVIDER_PROXY_URL", "http://provider-secret@example.invalid:9000")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"audio-front-end-eval", "--mock", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"metadata"`,
		`"generated_at"`,
		`"current_commit"`,
		`"fingerprint"`,
		`"proxy"`,
		`"global_proxy_env"`,
		`"HTTPS_PROXY"`,
		`"provider_proxy_env"`,
		`"A21_PROVIDER_PROXY_URL"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000", "pcm_s16le_base64"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked %q: %s", forbidden, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-audio-front-end-eval-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("report files = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, want := range []string{
		`"metadata"`,
		`"current_commit"`,
		`"fingerprint"`,
		`"proxy"`,
		`"global_proxy_env"`,
		`"provider_proxy_env"`,
	} {
		if !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: %s", want, reportJSON)
		}
	}
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000", "pcm_s16le_base64"} {
		if strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report leaked %q: %s", forbidden, reportJSON)
		}
	}
}

func TestRunAudioFrontEndEvalRejectsLegacyReportDirWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-eval", "--mock", "--output-dir", filepath.Join("reports", "x21-audio")}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21") {
		t.Fatalf("legacy report path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestValidateA21ReportDirAllowsOpaqueTempPathSubstrings(t *testing.T) {
	for _, path := range []string{
		filepath.Join("tmp", "a21randomx21suffix", "reports"),
		filepath.Join("tmp", "a21randomv21suffix", "reports"),
	} {
		if err := validateA21ReportDir(path); err != nil {
			t.Fatalf("validateA21ReportDir(%q) = %v, want nil for opaque temp substring", path, err)
		}
		inputPath := filepath.Join(path, "a21-input.wav")
		if err := validateA21InputPath(inputPath); err != nil {
			t.Fatalf("validateA21InputPath(%q) = %v, want nil for opaque temp substring", inputPath, err)
		}
	}
	for _, path := range []string{
		filepath.Join("reports", "x21-audio"),
		filepath.Join("reports", "v21-audio"),
		filepath.Join("reports", "a21-x21-audio"),
	} {
		if err := validateA21ReportDir(path); err == nil {
			t.Fatalf("validateA21ReportDir(%q) = nil, want legacy token rejection", path)
		}
		if err := validateA21InputPath(filepath.Join(path, "a21-input.wav")); err == nil {
			t.Fatalf("validateA21InputPath(%q) = nil, want legacy token rejection", path)
		}
	}
}

func TestRunFirmwareDeviceReportWritesA21Report(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			t.Fatalf("path = %q, want /v1/devices", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "capabilities": {
        "microphone": "available",
        "speaker": "available",
        "screen": "available",
        "screen_touch": "available",
        "top_touch": "available",
        "servo_y": "available",
        "rgb": "available"
      },
      "connection_status": "online",
      "device_age_ms": 42,
      "current_mode": "workmate",
      "current_expression": "speaking",
      "last_seen_ms": 1780000000000
    }
  ]
}`))
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", server.URL,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.firmware.device_report.v1"`,
		`"gateway_schema_version": "a21.gateway.devices.v1"`,
		`"gateway_service": "a21-gateway"`,
		`"device_report_path":`,
		`"gateway_url":`,
		`"device_id": "stackchan-001"`,
		`"capabilities":`,
		`"screen_touch": "available"`,
		`"servo_y": "available"`,
		`"connection_status": "online"`,
		`"device_age_ms": 42`,
		`"current_mode": "workmate"`,
		`"current_expression": "speaking"`,
		`"last_seen_ms": 1780000000000`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-devices-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("device report files = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"devices"`)) {
		t.Fatalf("report missing devices: %s", string(data))
	}
	for _, forbidden := range []string{"x21", "v21"} {
		if strings.Contains(strings.ToLower(string(data)), forbidden) {
			t.Fatalf("report contains forbidden identity %q: %s", forbidden, string(data))
		}
	}
}

func TestRunFirmwareDeviceReportRejectsGatewayWithoutA21Identity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      }
    }
  ]
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", server.URL,
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "A21 Gateway identity") {
		t.Fatalf("stderr = %q, want gateway identity rejection", stderr.String())
	}
	if strings.Contains(stdout.String(), "stackchan-001") {
		t.Fatalf("stdout should not contain accepted device report: %s", stdout.String())
	}
}

func TestRunFirmwareDeviceReportRejectsLegacyGatewayServiceWithoutEchoingIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "x21-gateway",
  "devices": []
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", server.URL,
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy gateway identity rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-gateway") {
		t.Fatalf("legacy gateway identity leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareDeviceReportRejectsLegacyOutputDirWithoutEchoingPath(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", "http://127.0.0.1:21080",
		"--output-dir", filepath.Join(t.TempDir(), "x21-reports"),
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want forbidden legacy identity", stderr.String())
	}
	if strings.Contains(strings.ToLower(stderr.String()), "x21-reports") {
		t.Fatalf("stderr should not echo legacy path: %q", stderr.String())
	}
}

func TestRunFirmwareDeviceReportRejectsGatewayCredentialsWithoutEchoingSecret(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", "http://user:secret@127.0.0.1:21080",
		"--output-dir", t.TempDir(),
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "must not include credentials") {
		t.Fatalf("stderr = %q, want credential rejection", stderr.String())
	}
	if strings.Contains(stderr.String(), "secret") || strings.Contains(stderr.String(), "user:") {
		t.Fatalf("stderr leaked gateway credentials: %q", stderr.String())
	}
}

func TestRunFirmwareDeviceReportRejectsLegacyGatewayPortWithoutDialingOrEchoingIt(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", "http://127.0.0.1:8080",
		"--output-dir", t.TempDir(),
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "legacy internal port") {
		t.Fatalf("stderr = %q, want legacy internal port rejection", stderr.String())
	}
	for _, forbidden := range []string{"127.0.0.1", "8080"} {
		if strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("stderr leaked %q: %s", forbidden, stderr.String())
		}
	}
}

func TestRunFirmwareDeviceReportRejectsLegacyGatewayDeviceWithoutEchoingIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "x21-stackchan",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      }
    }
  ]
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", server.URL,
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy identity rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-stackchan") {
		t.Fatalf("legacy identity leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareDeviceReportRejectsLegacyCapabilityWithoutEchoingIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "capabilities": {
        "screen": "x21-compatible"
      }
    }
  ]
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", server.URL,
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy identity rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-compatible") {
		t.Fatalf("legacy capability leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunProviderRealtimeFixtureExecutesDoubaoTTSWithoutSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "zh_female_kailangjiejie_moon_bigtts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-fixture", "--provider", "doubao_tts_realtime", "--execute"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "doubao_tts_realtime"`,
		`"protocol": "websocket_realtime_fixture"`,
		`"status": "passed"`,
		`"executed": true`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("fixture output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunProviderRealtimeFixtureWritesRedactedReport(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "zh_female_kailangjiejie_moon_bigtts")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-realtime-fixture", "--provider", "doubao_tts_realtime", "--execute", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-provider-realtime-fixture-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("matches = %#v, want one realtime fixture report", matches)
	}
	var report providers.ProviderSmokeReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	wantBase := filepath.Base(matches[0])
	if report.ReportPath != wantBase {
		t.Fatalf("report_path = %q, want basename %q", report.ReportPath, wantBase)
	}
	fileData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fileData), `"report_path": "`+wantBase+`"`) {
		t.Fatalf("file report missing basename report_path: %s", string(fileData))
	}
	for _, forbidden := range []string{dir, matches[0], "sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie", "Authorization", "Bearer", "http://", "https://", "/Users/"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) || strings.Contains(string(fileData), forbidden) {
			t.Fatalf("fixture report leaked %q: stdout=%s stderr=%s file=%s", forbidden, stdout.String(), stderr.String(), string(fileData))
		}
	}
}

func TestRunProviderRealtimeFixtureRejectsLegacyProviderWithoutEchoingValue(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-fixture", "--provider", "x21_realtime", "--execute"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"provider": "invalid_legacy_provider"`) {
		t.Fatalf("stdout missing invalid legacy provider: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "x21_realtime") || strings.Contains(stderr.String(), "x21_realtime") {
		t.Fatalf("legacy provider leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunDoctorIncludesV21AdapterHealthWhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Fatalf("path = %q, want /healthz", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	t.Setenv("A21_V21_ADAPTER_URL", server.URL)

	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"v21"`,
		`"configured": true`,
		`"healthy": true`,
		`"health_path": "/healthz"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunDoctorReportsV21AdapterSkippedWhenUnconfigured(t *testing.T) {
	t.Setenv("A21_V21_ADAPTER_URL", "")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"v21"`,
		`"configured": false`,
		`"status": "skipped"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}
