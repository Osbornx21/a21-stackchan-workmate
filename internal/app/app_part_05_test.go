package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/providers"
)

func TestRunServerSideReadinessBundleCollectsAuthorizedProviderAndV21Evidence(t *testing.T) {
	var providerCalls int
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("provider path = %q, want /chat/completions", r.URL.Path)
		}
		providerCalls++
		if r.Header.Get("Authorization") != "Bearer sk-a21-bundle-secret" {
			t.Fatalf("provider auth header missing")
		}
		var body struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "a21-bundle-hidden-model" || !body.Stream {
			t.Fatalf("provider body = %+v, want hidden model and stream=true", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"content":"bundle ok"}}]}`,
			`data: [DONE]`,
			``,
		}, "\n")))
	}))
	t.Cleanup(provider.Close)
	var v21Calls int
	v21 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","service":"a21-v21-adapter"}`))
			return
		}
		if r.URL.Path != "/a21/v21/query" {
			t.Fatalf("v21 path = %q, want /a21/v21/query", r.URL.Path)
		}
		v21Calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"trace_id":"a21-trace-bundle-v21",
			"fast_answer":"可执行证据正常。",
			"confidence":0.91,
			"evidence":[{"title":"Bundle Evidence","type":"doc","source_id":"a21-doc-001","summary":"已验证。"}],
			"speech_blocks":["我在查，先给你结论。"],
			"screen_cards":[{"label":"结论","text":"bundle evidence ok"}],
			"follow_ups":["继续看延迟吗？"]
		}`))
	}))
	t.Cleanup(v21.Close)
	dir := t.TempDir()
	profilePath := filepath.Join(t.TempDir(), "a21-provider-profiles.json")
	if err := os.WriteFile(profilePath, []byte(`{
		"name": "a21_bundle_vendor",
		"label": "A21 bundle vendor",
		"family": "text_stream",
		"protocol": "openai_chat_completions",
		"capabilities": ["llm", "streaming_text", "text_stream"],
		"api_key_env": "A21_BUNDLE_VENDOR_API_KEY",
		"model_env": "A21_BUNDLE_VENDOR_MODEL",
		"base_url_env": "A21_BUNDLE_VENDOR_BASE_URL",
		"endpoint_path": "/chat/completions",
		"route_eligible": true
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeProductReadinessReportFixtureFile(t, dir, "a21-xiaozhi-voice-bench-20260602-100100.json", productReadinessXiaozhiHostReportFixtureJSON())
	roleplayReadyPath := writeProductReadinessReportFixtureFile(t, dir, "a21-roleplay-voice-probe-20260602-100300.json", productReadinessRoleplayVoiceReportFixtureJSON("ready"))
	roleplayReadyTime := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(roleplayReadyPath, roleplayReadyTime, roleplayReadyTime); err != nil {
		t.Fatal(err)
	}
	professionalV21 := &xiaozhiProfessionalBenchV21Client{delay: 25 * time.Millisecond}
	professionalAdapters := providers.VoicePipelineAdapters{
		ASR:        xiaozhiProfessionalBenchASRAdapter{text: xiaozhiProfessionalBenchASRSentinel},
		TextStream: xiaozhiProfessionalBenchTextStreamAdapter{},
		TTS:        providers.NewMockTTSAdapter("a21-bundle-professional-tts"),
	}
	professionalGateway := gateway.NewServerWithOptions(gateway.ServerOptions{
		V21Client:                    professionalV21,
		XiaozhiVoicePipelineAdapters: &professionalAdapters,
	})
	professionalHTTP := httptest.NewServer(professionalGateway.Handler())
	t.Cleanup(professionalHTTP.Close)
	t.Setenv("A21_PROVIDER_PROFILES_PATH", profilePath)
	t.Setenv("A21_PROVIDER_PRIMARY", "a21_bundle_vendor")
	t.Setenv("A21_BUNDLE_VENDOR_API_KEY", "sk-a21-bundle-secret")
	t.Setenv("A21_BUNDLE_VENDOR_MODEL", "a21-bundle-hidden-model")
	t.Setenv("A21_BUNDLE_VENDOR_BASE_URL", provider.URL)
	t.Setenv("A21_V21_ADAPTER_URL", v21.URL)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"server-side-readiness-bundle",
		"--gateway-url", professionalHTTP.URL,
		"--use-latest-reports",
		"--collect-missing",
		"--execute-provider-smoke",
		"--execute-v21-smoke",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if providerCalls != 3 || v21Calls != 1 || professionalV21.lastUtterance() != "" {
		t.Fatalf("provider/v21/professional utterance = %d/%d/%q, want 3/1/no lab ritual", providerCalls, v21Calls, professionalV21.lastUtterance())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"status": "server_side_blocked"`,
		`"candidate_ready": false`,
		`"name": "provider_smoke"`,
		`"name": "v21_professional_smoke"`,
		`"name": "professional_ritual_execution"`,
		`"execution_authorized": true`,
		`"external_execution": true`,
		`"absorbed_by_readiness": true`,
		`"reason": "lab command moved to cmd/a21-lab"`,
		`"command": "go run ./cmd/a21-lab xiaozhi-professional-bench --gateway-url \u003cgateway\u003e --output-dir reports"`,
		`"source_report": "a21-provider-smoke-`,
		`"source_report": "a21-v21-adapter-smoke-`,
		`"provider_smoke_source_report": "a21-provider-smoke-`,
		`"professional_ritual_ready": false`,
		`"launch_ready": false`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	providerReports, err := filepath.Glob(filepath.Join(dir, "a21-provider-smoke-*.json"))
	if err != nil || len(providerReports) != 1 {
		t.Fatalf("provider reports = %d, %v: %v", len(providerReports), err, providerReports)
	}
	v21Reports, err := filepath.Glob(filepath.Join(dir, "a21-v21-adapter-smoke-*.json"))
	if err != nil || len(v21Reports) != 1 {
		t.Fatalf("v21 reports = %d, %v: %v", len(v21Reports), err, v21Reports)
	}
	professionalReports, err := filepath.Glob(filepath.Join(dir, "a21-xiaozhi-professional-bench-*.json"))
	if err != nil || len(professionalReports) != 0 {
		t.Fatalf("professional reports = %d, %v, want none after lab split: %v", len(professionalReports), err, professionalReports)
	}
	for _, forbidden := range []string{
		provider.URL,
		v21.URL,
		professionalHTTP.URL,
		dir,
		profilePath,
		"http://",
		"https://",
		"/Users/",
		"sk-a21-bundle-secret",
		"a21-bundle-hidden-model",
		"bundle ok",
		"可执行证据正常",
		`"launch_ready": true`,
		`"prd_accepted": true`,
	} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("server-side authorized collect leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunProductReadinessCommandUsesLatestWakeWordFirmwarePlanWithoutPathLeak(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"service":"a21-gateway","status":"ok"}`))
		case "/simulator":
			w.Header().Set("content-type", "text/html")
			_, _ = w.Write([]byte("<!doctype html><title>A21 Simulator</title>"))
		case "/v1/devices":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`))
		case "/v1/wake-word":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.wake_word.v1","mode":"custom_multinet","active_phrase":"你好小智","active_pinyin":"ni hao xiao zhi","desired_phrase":"小阿二一","desired_pinyin":"xiao a er yi","threshold":35,"runtime_status":"pending_firmware_build","runtime_configurable":false,"firmware_build_required":true,"code":"a21_wake_word_firmware_build_required","message":"Custom wake words require a dedicated xiaozhi/ESP-SR MultiNet firmware build; Gateway only persists the requested profile."}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-plan-20260602-010000.json", `{
  "schema_version": "a21.wake_word_firmware_plan.v1",
  "generated_at_ms": 1780333200000,
  "status": "pending_firmware_build",
  "dry_run": true,
  "firmware_build_required": true,
  "build_allowed": false,
  "flash_allowed": false,
  "firmware_id": "a21-stackchan",
  "target_board": "m5stack-cores3",
  "target_profile": "xiaozhi_esp_sr_multinet",
  "guard_tier": "T7",
  "mode": "custom_multinet",
  "active_phrase": "你好小智",
  "active_pinyin": "ni hao xiao zhi",
  "desired_phrase": "小阿二一",
  "desired_pinyin": "xiao a er yi",
  "threshold": 35,
  "runtime_status": "pending_firmware_build",
  "runtime_configurable": false,
  "next_required_confirmation": "BUILD_A21_WAKE_WORD_FIRMWARE",
  "report_path": "a21-wake-word-firmware-plan-20260602-010000.json"
}`)
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"mode": "custom_multinet"`,
		`"firmware_build_required": true`,
		`"firmware_plan_available": true`,
		`"firmware_plan_status": "pending_firmware_build"`,
		`"firmware_plan_source_report": "a21-wake-word-firmware-plan-20260602-010000.json"`,
		`"firmware_plan_dry_run": true`,
		`"firmware_plan_build_allowed": false`,
		`"firmware_plan_flash_allowed": false`,
		`"wake_word_firmware_plan_available"`,
		`"launch_ready": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{server.URL, dir, "http://", "https://", "/Users/", `"launch_ready": true`, "secret", "token"} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("wake word plan readiness leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunProductReadinessCommandUsesLatestMatchingWakeWordFirmwarePlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"service":"a21-gateway","status":"ok"}`))
		case "/simulator":
			w.Header().Set("content-type", "text/html")
			_, _ = w.Write([]byte("<!doctype html><title>A21 Simulator</title>"))
		case "/v1/devices":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`))
		case "/v1/wake-word":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.wake_word.v1","mode":"custom_multinet","active_phrase":"你好小智","active_pinyin":"ni hao xiao zhi","desired_phrase":"小阿二一","desired_pinyin":"xiao a er yi","threshold":35,"runtime_status":"pending_firmware_build","runtime_configurable":false,"firmware_build_required":true,"code":"a21_wake_word_firmware_build_required","message":"Custom wake words require a dedicated xiaozhi/ESP-SR MultiNet firmware build; Gateway only persists the requested profile."}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-plan-20260602-010000.json", `{
  "schema_version": "a21.wake_word_firmware_plan.v1",
  "generated_at_ms": 1780333200000,
  "status": "pending_firmware_build",
  "dry_run": true,
  "firmware_build_required": true,
  "build_allowed": false,
  "flash_allowed": false,
  "firmware_id": "a21-stackchan",
  "target_board": "m5stack-cores3",
  "target_profile": "xiaozhi_esp_sr_multinet",
  "guard_tier": "T7",
  "mode": "custom_multinet",
  "active_phrase": "你好小智",
  "active_pinyin": "ni hao xiao zhi",
  "desired_phrase": "小阿二一",
  "desired_pinyin": "xiao a er yi",
  "threshold": 35,
  "runtime_status": "pending_firmware_build",
  "runtime_configurable": false,
  "next_required_confirmation": "BUILD_A21_WAKE_WORD_FIRMWARE",
  "report_path": "a21-wake-word-firmware-plan-20260602-010000.json"
}`)
	writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-plan-20260602-020000.json", `{
  "schema_version": "a21.wake_word_firmware_plan.v1",
  "generated_at_ms": 1780336800000,
  "status": "pending_firmware_build",
  "dry_run": true,
  "firmware_build_required": true,
  "build_allowed": false,
  "flash_allowed": false,
  "firmware_id": "a21-stackchan",
  "target_board": "m5stack-cores3",
  "target_profile": "xiaozhi_esp_sr_multinet",
  "guard_tier": "T7",
  "mode": "custom_multinet",
  "active_phrase": "你好小智",
  "active_pinyin": "ni hao xiao zhi",
  "desired_phrase": "小阿二二",
  "desired_pinyin": "xiao a er er",
  "threshold": 35,
  "runtime_status": "pending_firmware_build",
  "runtime_configurable": false,
  "next_required_confirmation": "BUILD_A21_WAKE_WORD_FIRMWARE",
  "report_path": "a21-wake-word-firmware-plan-20260602-020000.json"
}`)
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"mode": "custom_multinet"`,
		`"firmware_build_required": true`,
		`"firmware_plan_available": true`,
		`"firmware_plan_source_report": "a21-wake-word-firmware-plan-20260602-010000.json"`,
		`"wake_word_firmware_plan_available"`,
		`"detail": "wake_word_firmware_plan:a21-wake-word-firmware-plan-20260602-020000.json"`,
		`"launch_ready": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{
		`"firmware_plan_source_report": "a21-wake-word-firmware-plan-20260602-020000.json"`,
		`"wake_word_firmware_plan_mismatch"`,
		server.URL,
		dir,
		"http://",
		"https://",
		"/Users/",
		`"launch_ready": true`,
	} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("latest matching wake word plan leaked or misselected %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunProductReadinessCommandSkipsCustomWakeWordPlanForBuiltinRuntime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"service":"a21-gateway","status":"ok"}`))
		case "/simulator":
			w.Header().Set("content-type", "text/html")
			_, _ = w.Write([]byte("<!doctype html><title>A21 Simulator</title>"))
		case "/v1/devices":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`))
		case "/v1/wake-word":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.wake_word.v1","mode":"builtin_xiaozhi_wakenet","active_phrase":"你好小智","active_pinyin":"ni hao xiao zhi","desired_phrase":"你好小智","desired_pinyin":"ni hao xiao zhi","threshold":30,"runtime_status":"active_builtin_model","runtime_configurable":false,"firmware_build_required":false,"code":"","message":""}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-plan-20260602-020000.json", `{
  "schema_version": "a21.wake_word_firmware_plan.v1",
  "generated_at_ms": 1780336800000,
  "status": "pending_firmware_build",
  "dry_run": true,
  "firmware_build_required": true,
  "build_allowed": false,
  "flash_allowed": false,
  "firmware_id": "a21-stackchan",
  "target_board": "m5stack-cores3",
  "target_profile": "xiaozhi_esp_sr_multinet",
  "guard_tier": "T7",
  "mode": "custom_multinet",
  "active_phrase": "你好小智",
  "active_pinyin": "ni hao xiao zhi",
  "desired_phrase": "小阿二一",
  "desired_pinyin": "xiao a er yi",
  "threshold": 35,
  "runtime_status": "pending_firmware_build",
  "runtime_configurable": false,
  "next_required_confirmation": "BUILD_A21_WAKE_WORD_FIRMWARE",
  "report_path": "a21-wake-word-firmware-plan-20260602-020000.json"
}`)
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"mode": "builtin_xiaozhi_wakenet"`,
		`"product_ready": true`,
		`"detail": "wake_word_firmware_plan:a21-wake-word-firmware-plan-20260602-020000.json"`,
		`"launch_ready": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{
		`"firmware_plan_available": true`,
		`"wake_word_firmware_plan_mismatch"`,
		server.URL,
		dir,
		"http://",
		"https://",
		"/Users/",
		`"launch_ready": true`,
	} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("builtin wake word latest plan leaked or misselected %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunProductReadinessCommandIngestsWakeWordFirmwarePackageWithoutGreenOrLeaks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"service":"a21-gateway","status":"ok"}`))
		case "/simulator":
			w.Header().Set("content-type", "text/html")
			_, _ = w.Write([]byte("<!doctype html><title>A21 Simulator</title>"))
		case "/v1/devices":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`))
		case "/v1/wake-word":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.wake_word.v1","mode":"custom_multinet","active_phrase":"你好小智","active_pinyin":"ni hao xiao zhi","desired_phrase":"小阿二一","desired_pinyin":"xiao a er yi","threshold":35,"runtime_status":"pending_firmware_build","runtime_configurable":false,"firmware_build_required":true,"code":"a21_wake_word_firmware_build_required","message":"Custom wake words require a dedicated xiaozhi/ESP-SR MultiNet firmware build; Gateway only persists the requested profile."}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	fixture := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-030000.json", productReadinessWakeWordFirmwarePackageReportFixtureJSON())
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--wake-word-firmware-package-report", fixture, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"mode": "custom_multinet"`,
		`"product_ready": false`,
		`"firmware_package_available": true`,
		`"firmware_package_status": "packaged"`,
		`"firmware_package_source_report": "a21-wake-word-firmware-package-20260602-030000.json"`,
		`"firmware_package_artifact_name": "a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-abcdef123456-20260602-030000.bin"`,
		`"firmware_package_manifest_name": "a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-abcdef123456-20260602-030000.bin.manifest.json"`,
		`"firmware_package_flash_allowed": false`,
		`"firmware_package_flash_executed": false`,
		`"wake_word_firmware_package_available"`,
		`"wake_word_ready": false`,
		`"launch_ready": false`,
		`use the wake word firmware package in a guarded hardware-window flash plan and collect physical custom wake proof`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbiddenAction := range []string{
		`use the wake word firmware plan to prepare the guarded build/package and hardware-window flash`,
		`"wake_word_ready": true`,
		`"launch_ready": true`,
	} {
		if strings.Contains(rendered, forbiddenAction) {
			t.Fatalf("wake word package readiness has wrong action/overclaim %q: %s", forbiddenAction, rendered)
		}
	}
	for _, forbidden := range []string{
		fixture,
		filepath.Dir(fixture),
		server.URL,
		"http://",
		"https://",
		"/Users/",
		"secret",
		"token",
	} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("wake word package readiness leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunProductReadinessCommandUsesLatestMatchingWakeWordFirmwarePackage(t *testing.T) {
	server := newProductReadinessCustomWakeTestServer(t)
	dir := t.TempDir()
	matching := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-030000.json", productReadinessWakeWordFirmwarePackageReportFixtureJSON())
	mismatched := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-050000.json", strings.Replace(productReadinessWakeWordFirmwarePackageReportFixtureJSON(), `"threshold": 35`, `"threshold": 45`, 1))
	older := time.Date(2026, 6, 2, 3, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 2, 5, 0, 0, 0, time.UTC)
	if err := os.Chtimes(matching, older, older); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(mismatched, newer, newer); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"firmware_package_available": true`,
		`"firmware_package_source_report": "a21-wake-word-firmware-package-20260602-030000.json"`,
		`"wake_word_firmware_package_available"`,
		`"detail": "wake_word_firmware_package:a21-wake-word-firmware-package-20260602-050000.json"`,
		`"product_ready": false`,
		`"wake_word_ready": false`,
		`"launch_ready": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{
		`"firmware_package_source_report": "a21-wake-word-firmware-package-20260602-050000.json"`,
		matching,
		mismatched,
		dir,
		server.URL,
		"http://",
		"https://",
		"/Users/",
		`"wake_word_ready": true`,
		`"launch_ready": true`,
	} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("latest matching wake word package leaked or misselected %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunProductReadinessCommandRejectsWakeWordFirmwarePackageMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"service":"a21-gateway","status":"ok"}`))
		case "/simulator":
			w.Header().Set("content-type", "text/html")
			_, _ = w.Write([]byte("<!doctype html><title>A21 Simulator</title>"))
		case "/v1/devices":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`))
		case "/v1/wake-word":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.wake_word.v1","mode":"custom_multinet","active_phrase":"你好小智","active_pinyin":"ni hao xiao zhi","desired_phrase":"小阿二一","desired_pinyin":"xiao a er yi","threshold":35,"runtime_status":"pending_firmware_build","runtime_configurable":false,"firmware_build_required":true,"code":"a21_wake_word_firmware_build_required","message":"Custom wake words require a dedicated xiaozhi/ESP-SR MultiNet firmware build; Gateway only persists the requested profile."}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	data := strings.Replace(productReadinessWakeWordFirmwarePackageReportFixtureJSON(), `"desired_phrase": "小阿二一"`, `"desired_phrase": "小阿二二"`, 1)
	fixture := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-030000.json", data)
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--wake-word-firmware-package-report", fixture, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"wake_word_firmware_package_mismatch"`,
		`"detail": "a21-wake-word-firmware-package-20260602-030000.json"`,
		`"firmware_package_available": false`,
		`"wake_word_ready": false`,
		`"launch_ready": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{fixture, filepath.Dir(fixture), server.URL, "http://", "https://", "/Users/", `"wake_word_ready": true`, `"launch_ready": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("wake word package mismatch leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunProductReadinessCommandRejectsUnsafeWakeWordFirmwarePackageWithoutLeak(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"service":"a21-gateway","status":"ok"}`))
		case "/simulator":
			w.Header().Set("content-type", "text/html")
			_, _ = w.Write([]byte("<!doctype html><title>A21 Simulator</title>"))
		case "/v1/devices":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`))
		case "/v1/wake-word":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.wake_word.v1","mode":"custom_multinet","active_phrase":"你好小智","active_pinyin":"ni hao xiao zhi","desired_phrase":"小阿二一","desired_pinyin":"xiao a er yi","threshold":35,"runtime_status":"pending_firmware_build","runtime_configurable":false,"firmware_build_required":true,"code":"a21_wake_word_firmware_build_required","message":"Custom wake words require a dedicated xiaozhi/ESP-SR MultiNet firmware build; Gateway only persists the requested profile."}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	data := strings.Replace(productReadinessWakeWordFirmwarePackageReportFixtureJSON(), `"report_path": "a21-wake-word-firmware-package-20260602-030000.json"`, `"local_path": "/Users/private/a21/package.json",
  "api_key": "secret-token-value",
  "report_path": "a21-wake-word-firmware-package-20260602-030000.json"`, 1)
	fixture := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-030000.json", data)
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--wake-word-firmware-package-report", fixture, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	if !strings.Contains(rendered, `"code": "wake_word_firmware_package_invalid"`) {
		t.Fatalf("stdout missing fixed invalid finding: %s", rendered)
	}
	for _, forbidden := range []string{fixture, filepath.Dir(fixture), server.URL, "/Users/private", "secret-token-value", "http://", "https://", `"wake_word_ready": true`, `"launch_ready": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("unsafe wake word package leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func productReadinessWakeWordFirmwarePackageReportFixtureJSON() string {
	return `{
  "schema_version": "a21.wake_word_firmware_package.v1",
  "generated_at_ms": 1780340400000,
  "status": "packaged",
  "package_written": true,
  "flash_allowed": false,
  "flash_executed": false,
  "product_ready": false,
  "firmware_id": "a21-stackchan",
  "target_board": "m5stack-cores3",
  "target_profile": "xiaozhi_esp_sr_multinet",
  "mode": "custom_multinet",
  "desired_phrase": "小阿二一",
  "desired_pinyin": "xiao a er yi",
  "threshold": 35,
  "commit": "abcdef123456",
  "timestamp": "20260602-030000",
  "source_plan_report": "a21-wake-word-firmware-plan-20260602-010000.json",
  "build_receipt": "a21-wake-word-build.json",
  "build_dir_name": "build-m5stack-core-s3",
  "artifact_name": "a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-abcdef123456-20260602-030000.bin",
  "sha256_name": "a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-abcdef123456-20260602-030000.bin.sha256",
  "manifest_name": "a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-abcdef123456-20260602-030000.bin.manifest.json",
  "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "ota": {
    "scheme": "https",
    "host": "ota.a21.local",
    "path": "/firmware/a21-wake-word.bin"
  },
  "parts": [
    {
      "name": "app",
      "offset": "0x10000",
      "file": "xiaozhi.bin",
      "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
      "size_bytes": 4096
    }
  ],
  "next_required_actions": [
    "Run a guarded wake-word firmware flash plan against this package in a foreground hardware window.",
    "Prove the custom wake model is active on the physical StackChan before product readiness can pass."
  ],
  "findings": [
    {
      "code": "wake_word_firmware_package_below_activation",
      "severity": "info",
      "message": "Wake word firmware package exists, but flash and physical custom wake evidence are still required."
    }
  ],
  "report_path": "a21-wake-word-firmware-package-20260602-030000.json"
}`
}

func TestRunProductReadinessCommandRejectsUnsafeV21ProfessionalReportWithoutLeak(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessV21ProfessionalReportFixtureFromData(t, strings.Replace(productReadinessV21ProfessionalReportFixtureJSON(), `  "report_path": "a21-v21-professional-readiness-host.json"`, `  "prompt": "professional readiness fixture query",
  "raw_evidence": "raw evidence text",
  "local_path": "/Users/private/a21/report.json",
  "report_path": "a21-v21-professional-readiness-host.json"`, 1))
	dir, err := os.MkdirTemp("", "a21-product-readiness-prof-unsafe-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--v21-professional-report", fixture, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	rendered := stdout.String()
	if !strings.Contains(rendered, `"code": "v21_professional_report_invalid"`) {
		t.Fatalf("stdout missing fixed invalid finding: %s", rendered)
	}
	for _, forbidden := range []string{server.URL, fixture, filepath.Dir(fixture), "professional readiness fixture query", "raw evidence text", "/Users/private", "http://", "https://"} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("unsafe professional report leaked %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestProductReadinessRejectsXiaozhiReportMissingCandidateFields(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	tests := []struct {
		name      string
		mutate    func(string) string
		wantField string
	}{
		{
			name: "missing barge p95",
			mutate: func(data string) string {
				return strings.Replace(data, `    "barge_in_stop_p50_ms": 0,
    "barge_in_stop_p95_ms": 0`, `    "barge_in_stop_p50_ms": 0`, 1)
			},
			wantField: "summary.barge_in_stop_p95_ms",
		},
		{
			name: "missing answer p95",
			mutate: func(data string) string {
				return strings.Replace(data, `    "answer_first_audio_total_p50_ms": 360,
    "answer_first_audio_total_p95_ms": 386,`, `    "answer_first_audio_total_p50_ms": 360,`, 1)
			},
			wantField: "summary.answer_first_audio_total_p95_ms",
		},
		{
			name: "missing failure count",
			mutate: func(data string) string {
				return strings.Replace(data, `    "barge_in_turn_count": 3,
    "failure_count": 0`, `    "barge_in_turn_count": 3`, 1)
			},
			wantField: "counts.failure_count",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := writeProductReadinessXiaozhiHostReportFixtureFromData(t, tt.mutate(productReadinessXiaozhiHostReportFixtureJSON()))

			report := buildProductReadinessReport(context.Background(), productReadinessOptions{
				GatewayURL:    server.URL,
				DeviceID:      "stackchan-001",
				XiaozhiReport: fixture,
			}, []string{
				"A21_LOCAL_TTS_ENGINE=sherpa_onnx",
				"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
				"A21_TEXT_STREAM_PROFILE=ollama_local",
				"A21_TTS_FAST_PROFILE=sherpa_onnx_tts",
			})

			if report.Voice.VoicePipeline.HostLoopbackCandidateReady {
				t.Fatalf("voice pipeline = %+v, want missing field to block candidate readiness", report.Voice.VoicePipeline)
			}
			if !containsProductFinding(report.Findings, "xiaozhi_report_missing_field", tt.wantField) {
				t.Fatalf("findings = %#v, want missing-field finding for %s", report.Findings, tt.wantField)
			}
			if !containsExactProductString(report.CanonicalDecision.MissingReportFields, "xiaozhi_report:"+tt.wantField) {
				t.Fatalf("canonical missing fields = %#v, want xiaozhi_report:%s", report.CanonicalDecision.MissingReportFields, tt.wantField)
			}
			var encoded bytes.Buffer
			if err := writeJSONProductReadiness(&encoded, report); err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{fixture, filepath.Dir(fixture), "http://", "https://"} {
				if strings.Contains(encoded.String(), forbidden) {
					t.Fatalf("product readiness leaked forbidden fragment %q: %s", forbidden, encoded.String())
				}
			}
		})
	}
}

func TestProductVoiceReadinessDoesNotOverclaimEnvOnlyPipeline(t *testing.T) {
	voice := buildProductVoiceReadiness([]string{
		"A21_LOCAL_TTS_ENGINE=sherpa_onnx",
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_TEXT_STREAM_PROFILE=ollama_local",
		"A21_TTS_FAST_PROFILE=sherpa_onnx_tts",
	}, productProviderReadiness{}, productStackChanReadiness{
		PhysicalDeviceOnline:    true,
		PhysicalMicrophoneReady: true,
	})

	if voice.VoicePipeline.HostLocalTextReady || voice.VoicePipeline.HostLocalTTSReady || voice.LocalTTSReady || voice.ContinuousVoiceReady {
		t.Fatalf("voice readiness = %+v, want env-only profile labels not to claim text/TTS/runtime readiness", voice)
	}
}

func TestProductVoiceReadinessAcceptsConfiguredVoiceCloneTTS(t *testing.T) {
	dir := t.TempDir()
	commandPath := filepath.Join(dir, "a21-voice-clone-wrapper")
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	refAudio := filepath.Join(dir, "a21-persona-reference.wav")
	if err := os.WriteFile(refAudio, []byte("RIFF-a21-reference"), 0o644); err != nil {
		t.Fatal(err)
	}

	voice := buildProductVoiceReadiness([]string{
		"A21_TTS_FAST_PROFILE=voice_clone_cli",
		"A21_VOICE_CLONE_COMMAND=" + commandPath,
		"A21_VOICE_CLONE_REF_AUDIO=" + refAudio,
	}, productProviderReadiness{}, productStackChanReadiness{})

	if !voice.LocalTTSReady || voice.LocalTTSEngine != "voice_clone_cli" || !voice.VoicePipeline.HostLocalTTSReady {
		t.Fatalf("voice readiness = %+v, want configured clone TTS host-local ready", voice)
	}
	if voice.ContinuousVoiceReady {
		t.Fatalf("continuous voice ready = true, want clone TTS readiness not to overclaim full pipeline")
	}
}
