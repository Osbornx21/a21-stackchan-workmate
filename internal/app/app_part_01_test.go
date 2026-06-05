package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/v21adapter"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := stdout.String(); got != "A21 0.1.0-dev (a21-core)\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"nope"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestProductReadinessReportsMockInputsWithoutProductDemoReady(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	ttsModelDir := createProductReadinessTTSModelDir(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{Path: "/dev/cu.usbmodem1101", USBModem: true, Usage: firmwarecheck.PortUsage{Exists: true}}}, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL: server.URL,
		DeviceID:   "stackchan-001",
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
		"A21_LOCAL_TTS_ENGINE=sherpa_onnx",
		"A21_SHERPA_ONNX_MODEL_DIR=" + ttsModelDir,
	})

	if report.Status != "blocked" || report.DemoReady || report.LaunchReady {
		t.Fatalf("status/demo/launch = %q/%v/%v, want blocked/false/false", report.Status, report.DemoReady, report.LaunchReady)
	}
	if !report.StackChan.SimulatorDeviceOnline || report.StackChan.PhysicalDeviceOnline {
		t.Fatalf("stackchan readiness = %+v, want simulator only", report.StackChan)
	}
	if report.Provider.Selected != "mock" || report.Provider.RealProviderReady {
		t.Fatalf("provider readiness = %+v, want mock selected and not real-ready", report.Provider)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{server.URL, "http://", "https://", ttsModelDir} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessReportsPersonalityMemoryStateWithoutLeakingText(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL: server.URL,
		DeviceID:   "stackchan-001",
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
		"A21_MEMORY_USER_PREFERENCES=偏好短句\n不要鸡汤",
		"A21_MEMORY_SESSION_NOTES=当前任务是座舱 PRD 评审\nhttp://secret.example/leak\n/Users/me/a21-secret.txt",
	})

	if !report.Memory.ContractReady || !report.Memory.Configured || !report.Memory.PromptInputReady {
		t.Fatalf("memory readiness = %+v, want configured prompt-ready contract", report.Memory)
	}
	if report.Memory.UserPreferenceCount != 2 || report.Memory.SessionMemoryCount != 1 {
		t.Fatalf("memory counts = preferences:%d session:%d, want 2/1", report.Memory.UserPreferenceCount, report.Memory.SessionMemoryCount)
	}
	if report.Memory.Redaction.MemoryTextStored ||
		report.Memory.Redaction.PromptTextStored ||
		report.Memory.Redaction.TranscriptStored ||
		report.Memory.Redaction.ProviderOutputStored ||
		report.Memory.Redaction.LocalPathsStored {
		t.Fatalf("memory redaction = %+v, want no sensitive material stored", report.Memory.Redaction)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"memory": {`,
		`"schema_version": "a21.personality_memory_state.v1"`,
		`"policy": "bounded_prompt_hints"`,
		`"memory_text_stored": false`,
		`"source_env":`,
		`"A21_MEMORY_USER_PREFERENCES"`,
		`"A21_MEMORY_SESSION_NOTES"`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("product readiness missing %q: %s", want, encoded.String())
		}
	}
	for _, forbidden := range []string{
		"偏好短句",
		"不要鸡汤",
		"当前任务是座舱 PRD 评审",
		"secret.example",
		"/Users/me",
		"raw prompt text",
		"raw transcript text",
		"raw provider output",
	} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessCanReachRealLaunchReadyWhenInputsArePresent(t *testing.T) {
	server := newProductReadinessTestServerWithWakeWord(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","identity_status":"valid","connection_status":"online","capabilities":{"microphone":"available_core_s3_i2s_24k_to_a21_16k"},"first_seen_ms":1,"last_seen_ms":2}]}`,
		`{"schema_version":"a21.gateway.wake_word.v1","mode":"custom_multinet","active_phrase":"你好小智","active_pinyin":"ni hao xiao zhi","desired_phrase":"小阿二一","desired_pinyin":"xiao a er yi","threshold":35,"runtime_status":"pending_firmware_build","runtime_configurable":false,"firmware_build_required":true,"code":"a21_wake_word_firmware_build_required"}`,
	)
	wakeWordDir := t.TempDir()
	wakeWordPackage := writeProductReadinessReportFixtureFile(t, wakeWordDir, "a21-wake-word-firmware-package-20260602-030000.json", productReadinessWakeWordFirmwarePackageReportFixtureJSON())
	wakeWordAcceptance := writeProductReadinessReportFixtureFile(t, wakeWordDir, "a21-wake-word-physical-acceptance-20260602-040000.json", productReadinessWakeWordPhysicalAcceptanceReportFixtureJSON())
	ttsModelDir := createProductReadinessTTSModelDir(t)
	asrModelDir := createProductReadinessASRModelDir(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{Path: "/dev/cu.usbmodem1101", USBModem: true, Usage: firmwarecheck.PortUsage{Exists: true}}}, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:                 server.URL,
		DeviceID:                   "stackchan-001",
		ProviderSmokeReport:        writeProductReadinessProviderSmokeReportFixture(t),
		V21ProfessionalReport:      writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t),
		V21AdapterSmokeReport:      writeProductReadinessV21AdapterSmokeReportFixture(t),
		PhysicalStackChanReport:    writeProductReadinessPhysicalStackChanReportFixture(t, map[string]any{"promotion_gate": "accepted", "acceptance_status": "prd_accepted", "prd_accepted": true}),
		WakeWordFirmwarePackage:    wakeWordPackage,
		WakeWordPhysicalAcceptance: wakeWordAcceptance,
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
		"A21_V21_ADAPTER_URL=" + server.URL,
		"A21_LOCAL_TTS_ENGINE=sherpa_onnx",
		"A21_SHERPA_ONNX_MODEL_DIR=" + ttsModelDir,
		"A21_LOCAL_ASR_PROVIDER=sherpa_onnx",
		"A21_SHERPA_ONNX_ASR_MODEL_DIR=" + asrModelDir,
	})

	if report.Status != "real_launch_ready" || !report.LaunchReady || report.DemoReady {
		t.Fatalf("status/launch/demo = %q/%v/%v, want real_launch_ready/true/false", report.Status, report.LaunchReady, report.DemoReady)
	}
	if !report.Provider.RealProviderReady || report.Provider.Selected != "deepseek" {
		t.Fatalf("provider readiness = %+v, want deepseek real-ready", report.Provider)
	}
	if !report.V21.Healthy || !report.StackChan.PhysicalDeviceOnline || !report.Voice.ContinuousVoiceReady {
		t.Fatalf("readiness = v21:%+v stackchan:%+v voice:%+v", report.V21, report.StackChan, report.Voice)
	}
	if !report.StackChan.PhysicalMicrophoneReady || report.StackChan.MicrophoneStatus != "available_core_s3_i2s_24k_to_a21_16k" {
		t.Fatalf("stackchan microphone readiness = %+v, want official bridge product microphone", report.StackChan)
	}
	if len(report.NextActions) != 0 {
		t.Fatalf("next actions = %#v, want none", report.NextActions)
	}
}

func TestProductReadinessDoesNotTreatProviderEnvOnlyAsRealReady(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL: server.URL,
		DeviceID:   "stackchan-001",
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
	})

	if !report.Provider.SelectedConfigured || report.Provider.Selected != "deepseek" {
		t.Fatalf("provider config = %+v, want selected deepseek configured", report.Provider)
	}
	if report.Provider.RealProviderReady || report.Provider.TextStreamReady || report.Provider.SmokeEvidenceValid {
		t.Fatalf("provider readiness = %+v, want env-only config not to pass real provider gate", report.Provider)
	}
	if !containsProductAction(report.NextActions, "provider-smoke --provider") {
		t.Fatalf("next actions = %#v, want provider-smoke evidence action", report.NextActions)
	}
}

func TestProductReadinessAcceptsExecutedProviderSmokeEvidence(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessProviderSmokeReportFixture(t)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:          server.URL,
		DeviceID:            "stackchan-001",
		ProviderSmokeReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
	})

	if !report.Provider.RealProviderReady || !report.Provider.TextStreamReady || !report.Provider.SmokeEvidenceValid {
		t.Fatalf("provider readiness = %+v, want executed provider smoke to satisfy real text provider gate", report.Provider)
	}
	if report.Provider.Selected != "deepseek" || report.Provider.SmokeProvider != "deepseek" || report.Provider.SmokeStatus != "passed" {
		t.Fatalf("provider selection/evidence = %+v, want deepseek passed smoke evidence", report.Provider)
	}
	if report.Provider.SmokeSourceReport != "a21-provider-smoke-real.json" || !report.Provider.SmokeExecuted {
		t.Fatalf("provider smoke source = %+v, want basename source and executed=true", report.Provider)
	}
	if containsProductAction(report.NextActions, "provider-smoke") || containsProductAction(report.NextActions, "configure a real A21 provider") {
		t.Fatalf("next actions = %#v, should not keep provider gap after valid smoke evidence", report.NextActions)
	}
	if report.LaunchReady {
		t.Fatalf("launch_ready = true with provider-only evidence; full PRD gates must still block")
	}
}

func TestProductReadinessAcceptsExecutedProviderSmokeEvidenceWithoutLocalKey(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessProviderSmokeReportFixture(t)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:          server.URL,
		DeviceID:            "stackchan-001",
		ProviderSmokeReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
	})

	if !report.Provider.RealProviderReady || !report.Provider.TextStreamReady || !report.Provider.SmokeEvidenceValid {
		t.Fatalf("provider readiness = %+v, want executed 5080lab provider smoke to satisfy real provider gate without local key", report.Provider)
	}
	if report.Provider.Selected != "deepseek" || !report.Provider.SelectedConfigured || len(report.Provider.MissingEnv) != 0 {
		t.Fatalf("provider selection = %+v, want imported smoke evidence to promote selected provider without local env gap", report.Provider)
	}
	if report.Provider.SmokeSourceReport != "a21-provider-smoke-real.json" || !report.Provider.SmokeExecuted {
		t.Fatalf("provider smoke source = %+v, want basename source and executed=true", report.Provider)
	}
	if containsProductAction(report.NextActions, "provider-smoke") || containsProductAction(report.NextActions, "configure a real A21 provider") {
		t.Fatalf("next actions = %#v, should not keep provider gap after valid external smoke evidence", report.NextActions)
	}
}

func TestProductReadinessUsesGatewayVoiceChainProviderWhenLocalProviderIsMock(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessProviderSmokeReportFixtureFromData(t, productReadinessProviderSmokeReportFixtureForProvider("stepfun"))

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:          server.URL,
		DeviceID:            "stackchan-001",
		ProviderSmokeReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
	})

	if !report.Provider.RealProviderReady || !report.Provider.TextStreamReady || !report.Provider.SmokeEvidenceValid {
		t.Fatalf("provider readiness = %+v, want Gateway-selected StepFun smoke to satisfy product provider gate", report.Provider)
	}
	if report.Provider.Primary != "stepfun" || report.Provider.Selected != "stepfun" || report.Provider.SmokeProvider != "stepfun" {
		t.Fatalf("provider readiness = %+v, want Gateway voice-chain StepFun selection", report.Provider)
	}
	if containsProductFinding(report.Findings, "provider_smoke_report_mismatch", "") ||
		containsProductAction(report.NextActions, "configure a real A21 provider") {
		t.Fatalf("readiness findings/actions = %#v/%#v, want no provider mismatch or local mock gap", report.Findings, report.NextActions)
	}
}

func TestProductReadinessKeepsExplicitLocalProviderOverGatewayVoiceChainProvider(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessProviderSmokeReportFixtureFromData(t, productReadinessProviderSmokeReportFixtureForProvider("stepfun"))

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:          server.URL,
		DeviceID:            "stackchan-001",
		ProviderSmokeReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
	})

	if report.Provider.RealProviderReady || report.Provider.SmokeEvidenceValid {
		t.Fatalf("provider readiness = %+v, want explicit local DeepSeek selection to reject StepFun smoke", report.Provider)
	}
	if report.Provider.Selected != "deepseek" {
		t.Fatalf("provider readiness = %+v, want explicit local provider preserved", report.Provider)
	}
	if !containsProductFinding(report.Findings, "provider_smoke_report_mismatch", "") {
		t.Fatalf("findings = %#v, want mismatch when explicit local provider conflicts with smoke", report.Findings)
	}
}

func TestProductReadinessIngestsRealtimeFixtureWithoutPromotingRealLaunch(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessRealtimeFixtureReportFixture(t)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:              server.URL,
		DeviceID:                "stackchan-001",
		ProviderRealtimeReport:  fixture,
		PhysicalStackChanReport: writeProductReadinessPhysicalStackChanReportFixture(t, map[string]any{"promotion_gate": "candidate", "acceptance_status": "physical_review_required", "prd_accepted": false}),
	}, []string{
		"A21_PROVIDER_PRIMARY=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=secret-value",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	})

	if !report.Provider.RealtimeEvidenceValid || !report.Provider.RealtimeExecuted {
		t.Fatalf("provider realtime evidence = %+v, want valid executed fixture evidence", report.Provider)
	}
	if report.Provider.RealtimeSourceReport != "a21-provider-realtime-fixture-real.json" ||
		report.Provider.RealtimeProvider != "doubao_tts_realtime" ||
		report.Provider.RealtimeFamily != "voice_hybrid" ||
		report.Provider.RealtimeStatus != "passed" {
		t.Fatalf("provider realtime source/evidence = %+v, want basename doubao_tts_realtime fixture", report.Provider)
	}
	if report.Provider.RealtimeRouteEligible || report.Provider.VoiceRealtimeReady {
		t.Fatalf("provider realtime readiness = %+v, want visible fixture evidence but no route-eligible realtime readiness", report.Provider)
	}
	if report.Provider.RealProviderReady || report.LaunchReady || report.CanonicalDecision.PRDAccepted {
		t.Fatalf("readiness overclaimed from fixture-only realtime evidence: provider=%+v launch=%v canonical=%+v", report.Provider, report.LaunchReady, report.CanonicalDecision)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{fixture, filepath.Dir(fixture), server.URL, "secret-value", "doubao-tts", "zh_female_kailangjiejie", "http://", "https://", `"launch_ready": true`, `"real_provider_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessRejectsUnsafeOrNonRealProviderSmokeEvidence(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	tests := []struct {
		name   string
		mutate func(string) string
	}{
		{
			name: "mock provider",
			mutate: func(data string) string {
				data = strings.ReplaceAll(data, `"provider": "deepseek"`, `"provider": "mock"`)
				data = strings.ReplaceAll(data, `"family": "text_stream"`, `"family": "mock"`)
				data = strings.ReplaceAll(data, `"protocol": "openai_chat_completions"`, `"protocol": "mock"`)
				return data
			},
		},
		{
			name: "not executed",
			mutate: func(data string) string {
				data = strings.ReplaceAll(data, `"status": "passed"`, `"status": "ready"`)
				data = strings.ReplaceAll(data, `"executed": true`, `"executed": false`)
				return data
			},
		},
		{
			name: "unsafe prompt leak",
			mutate: func(data string) string {
				return strings.Replace(data, `  "detail": "provider smoke request succeeded"`, `  "prompt": "raw prompt text",
  "detail": "provider smoke request succeeded"`, 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := writeProductReadinessProviderSmokeReportFixtureFromData(t, tt.mutate(productReadinessProviderSmokeReportFixtureJSON()))

			report := buildProductReadinessReport(context.Background(), productReadinessOptions{
				GatewayURL:          server.URL,
				DeviceID:            "stackchan-001",
				ProviderSmokeReport: fixture,
			}, []string{"A21_PROVIDER_PRIMARY=mock"})

			if report.Provider.RealProviderReady || report.Provider.SmokeEvidenceValid {
				t.Fatalf("provider readiness = %+v, want invalid provider smoke not accepted", report.Provider)
			}
			if !containsProductFinding(report.Findings, "provider_smoke_report_invalid", "") {
				t.Fatalf("findings = %#v, want provider_smoke_report_invalid", report.Findings)
			}
			var encoded bytes.Buffer
			if err := writeJSONProductReadiness(&encoded, report); err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{fixture, filepath.Dir(fixture), "raw prompt text", "http://", "https://", `"launch_ready": true`} {
				if strings.Contains(encoded.String(), forbidden) {
					t.Fatalf("product readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
				}
			}
		})
	}
}

func TestProductReadinessBlocksLaunchWhenCustomWakeWordNeedsFirmwareBuild(t *testing.T) {
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
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","identity_status":"valid","connection_status":"online","capabilities":{"microphone":"available_core_s3_i2s_24k_to_a21_16k"},"first_seen_ms":1,"last_seen_ms":2}]}`))
		case "/v1/wake-word":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.wake_word.v1","mode":"custom_multinet","active_phrase":"你好小智","active_pinyin":"ni hao xiao zhi","desired_phrase":"小阿二一","desired_pinyin":"xiao a er yi","threshold":35,"runtime_status":"pending_firmware_build","runtime_configurable":false,"firmware_build_required":true,"code":"a21_wake_word_firmware_build_required","message":"Custom wake words require a dedicated xiaozhi/ESP-SR MultiNet firmware build; Gateway only persists the requested profile."}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ttsModelDir := createProductReadinessTTSModelDir(t)
	asrModelDir := createProductReadinessASRModelDir(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{Path: "/dev/cu.usbmodem1101", USBModem: true, Usage: firmwarecheck.PortUsage{Exists: true}}}, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:              server.URL,
		DeviceID:                "stackchan-001",
		V21AdapterSmokeReport:   writeProductReadinessV21AdapterSmokeReportFixture(t),
		PhysicalStackChanReport: writeProductReadinessPhysicalStackChanReportFixture(t, map[string]any{"promotion_gate": "accepted", "acceptance_status": "prd_accepted", "prd_accepted": true}),
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
		"A21_V21_ADAPTER_URL=" + server.URL,
		"A21_LOCAL_TTS_ENGINE=sherpa_onnx",
		"A21_SHERPA_ONNX_MODEL_DIR=" + ttsModelDir,
		"A21_LOCAL_ASR_PROVIDER=sherpa_onnx",
		"A21_SHERPA_ONNX_ASR_MODEL_DIR=" + asrModelDir,
	})

	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"wake_word"`,
		`"mode": "custom_multinet"`,
		`"active_phrase": "你好小智"`,
		`"desired_phrase": "小阿二一"`,
		`"runtime_status": "pending_firmware_build"`,
		`"firmware_build_required": true`,
		`"code": "a21_wake_word_firmware_build_required"`,
		`"wake_word_firmware_build_required"`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("product readiness missing %q: %s", want, encoded.String())
		}
	}
	if report.LaunchReady || report.Status == "real_launch_ready" {
		t.Fatalf("status/launch = %q/%v, want custom wake word pending firmware to block launch", report.Status, report.LaunchReady)
	}
	if !containsProductAction(report.NextActions, "wake word firmware") {
		t.Fatalf("next actions = %#v, want wake word firmware action", report.NextActions)
	}
	for _, forbidden := range []string{server.URL, "http://", "https://", "secret-value", ttsModelDir, asrModelDir} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessTreatsStalePhysicalDeviceAsOffline(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","identity_status":"valid","connection_status":"online","device_age_ms":360001,"capabilities":{"microphone":"available_core_s3_i2s_24k_to_a21_16k"},"first_seen_ms":1,"last_seen_ms":2}]}`)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL: server.URL,
		DeviceID:   "stackchan-001",
	}, []string{})

	if report.StackChan.PhysicalDeviceOnline || !report.StackChan.PhysicalDeviceStale {
		t.Fatalf("stackchan readiness = %+v, want stale physical device not online", report.StackChan)
	}
	if report.StackChan.Status != "physical_stale" || report.StackChan.PhysicalDeviceAgeMS != 360001 {
		t.Fatalf("stackchan status/age = %q/%d, want physical_stale/360001", report.StackChan.Status, report.StackChan.PhysicalDeviceAgeMS)
	}
	if !containsProductAction(report.NextActions, "physical StackChan online") {
		t.Fatalf("next actions = %#v, want physical online action", report.NextActions)
	}
}

func TestProductReadinessBlocksLaunchWithoutExecutedV21AdapterSmoke(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","identity_status":"valid","connection_status":"online","capabilities":{"microphone":"available_core_s3_i2s_24k_to_a21_16k"},"first_seen_ms":1,"last_seen_ms":2}]}`)
	ttsModelDir := createProductReadinessTTSModelDir(t)
	asrModelDir := createProductReadinessASRModelDir(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{Path: "/dev/cu.usbmodem1101", USBModem: true, Usage: firmwarecheck.PortUsage{Exists: true}}}, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:              server.URL,
		DeviceID:                "stackchan-001",
		PhysicalStackChanReport: writeProductReadinessPhysicalStackChanReportFixture(t, map[string]any{"promotion_gate": "accepted", "acceptance_status": "prd_accepted", "prd_accepted": true}),
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
		"A21_V21_ADAPTER_URL=" + server.URL,
		"A21_LOCAL_TTS_ENGINE=sherpa_onnx",
		"A21_SHERPA_ONNX_MODEL_DIR=" + ttsModelDir,
		"A21_LOCAL_ASR_PROVIDER=sherpa_onnx",
		"A21_SHERPA_ONNX_ASR_MODEL_DIR=" + asrModelDir,
	})

	if report.LaunchReady || report.Status == "real_launch_ready" {
		t.Fatalf("status/launch = %q/%v, want launch blocked without executed V21 adapter smoke", report.Status, report.LaunchReady)
	}
	if report.V21.QueryExecuted || report.V21.Professional.Valid || report.V21.Professional.AdapterExecuted {
		t.Fatalf("v21 readiness = %+v, want no executed professional evidence", report.V21)
	}
	if !containsProductAction(report.NextActions, "v21-adapter-smoke") {
		t.Fatalf("next actions = %#v, want executed V21 adapter smoke action", report.NextActions)
	}
}

func TestProductReadinessExposesProfessionalBridgeStateWithoutQueryExecution(t *testing.T) {
	queryCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"service":"a21-v21-adapter","status":"ok"}`))
		case "/simulator":
			w.Header().Set("content-type", "text/html")
			_, _ = w.Write([]byte("<!doctype html><title>A21 Simulator</title>"))
		case "/v1/devices":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"44:1b:f6:e2:6a:60","identity_status":"valid","connection_status":"online","first_seen_ms":1,"last_seen_ms":2},{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`))
		case "/a21/v21/query":
			queryCalled = true
			http.Error(w, "query execution is out of scope for product readiness", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL: server.URL,
		DeviceID:   "stackchan-001",
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
		"A21_V21_ADAPTER_URL=" + server.URL,
	})

	if queryCalled {
		t.Fatal("product readiness must not execute the V21 professional query path")
	}
	if !report.V21.Configured || !report.V21.Healthy || !report.V21.ProfessionalBridgeReady {
		t.Fatalf("v21 readiness = %+v", report.V21)
	}
	if !report.V21.CheckingFeedbackSupported || report.V21.MaxFirstResponseMS != 1200 {
		t.Fatalf("v21 readiness missing 1200ms checking feedback support: %+v", report.V21)
	}
	if !report.V21.EvidenceContractReady || report.V21.QueryExecuted {
		t.Fatalf("v21 readiness contract/query execution = %+v", report.V21)
	}
	if report.V21.QueryPath != v21adapter.QueryPath || report.V21.HealthPath != v21adapter.HealthPath {
		t.Fatalf("v21 readiness paths = %+v", report.V21)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{server.URL, "http://", "https://"} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessBlocksLaunchForDiagnosticMicrophone(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","identity_status":"valid","connection_status":"online","capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture"},"first_seen_ms":1,"last_seen_ms":2}]}`)
	ttsModelDir := createProductReadinessTTSModelDir(t)
	asrModelDir := createProductReadinessASRModelDir(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{Path: "/dev/cu.usbmodem1101", USBModem: true, Usage: firmwarecheck.PortUsage{Exists: true}}}, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL: server.URL,
		DeviceID:   "stackchan-001",
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
		"A21_V21_ADAPTER_URL=" + server.URL,
		"A21_LOCAL_TTS_ENGINE=sherpa_onnx",
		"A21_SHERPA_ONNX_MODEL_DIR=" + ttsModelDir,
		"A21_LOCAL_ASR_PROVIDER=sherpa_onnx",
		"A21_SHERPA_ONNX_ASR_MODEL_DIR=" + asrModelDir,
	})

	if report.LaunchReady || report.Voice.ContinuousVoiceReady {
		t.Fatalf("launch/voice = %v/%v, want both blocked by diagnostic microphone", report.LaunchReady, report.Voice.ContinuousVoiceReady)
	}
	if !report.StackChan.PhysicalDeviceOnline || report.StackChan.PhysicalMicrophoneReady {
		t.Fatalf("stackchan readiness = %+v, want online physical device without product microphone", report.StackChan)
	}
	if report.StackChan.MicrophoneStatus != "diagnostic_probe_m5unified_i2s_capture" {
		t.Fatalf("microphone status = %q, want diagnostic status", report.StackChan.MicrophoneStatus)
	}
	if !containsProductAction(report.NextActions, "promote StackChan microphone") {
		t.Fatalf("next actions = %#v, want microphone promotion action", report.NextActions)
	}
}

func TestProductReadinessIngestsXiaozhiHostLoopbackCandidateEvidence(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessXiaozhiHostReportFixture(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:    server.URL,
		DeviceID:      "stackchan-001",
		XiaozhiReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=ollama_local",
		"A21_TEXT_STREAM_PROFILE=ollama_local",
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_TTS_FAST_PROFILE=sherpa_onnx_tts",
	})

	if report.LaunchReady || report.Voice.VoicePipeline.PRDAccepted {
		t.Fatalf("launch/prd = %v/%v, want host-only evidence not accepted", report.LaunchReady, report.Voice.VoicePipeline.PRDAccepted)
	}
	if !report.Voice.RealASRReady || report.Voice.ASRProvider != "sherpa_onnx" {
		t.Fatalf("voice readiness = %+v, want current voice pipeline ASR labels ready", report.Voice)
	}
	pipeline := report.Voice.VoicePipeline
	if !pipeline.HostLoopbackCandidateReady || pipeline.AcceptanceStatus != "candidate_host_only" || pipeline.SourceReport != "a21-xiaozhi-host-local-report.json" {
		t.Fatalf("voice pipeline = %+v, want host-loopback candidate from basename source", pipeline)
	}
	if pipeline.AnswerFirstAudioP95MS != 386 || pipeline.BargeInStopP95MS != 0 || pipeline.FailureCount != 0 {
		t.Fatalf("voice pipeline timings = %+v, want p95 answer 386, barge 0, no failures", pipeline)
	}
	if pipeline.ExecutionMode != "host_local" ||
		pipeline.ASRProfile != "sherpa_onnx" ||
		pipeline.ASRProfileEnv != "A21_ASR_LOCAL_PROFILE" ||
		pipeline.TextStreamProfile != "ollama_local" ||
		pipeline.TextStreamProfileEnv != "A21_TEXT_STREAM_PROFILE" ||
		pipeline.TTSProfile != "sherpa_onnx_tts" ||
		pipeline.TTSProfileEnv != "A21_TTS_FAST_PROFILE" ||
		!pipeline.HostLocalASRReady ||
		!pipeline.HostLocalTextReady ||
		!pipeline.HostLocalTTSReady {
		t.Fatalf("voice pipeline selection = %+v, want host-local A21 env/profile evidence", pipeline)
	}
	if containsProductAction(report.NextActions, "A21_LOCAL_ASR_PROVIDER") {
		t.Fatalf("next actions = %#v, should not ask for stale ASR env when voice pipeline ASR is ready", report.NextActions)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		fixture,
		filepath.Dir(fixture),
		"http://",
		"https://",
		"data_base64",
		"transcript",
		"provider output",
		"secret",
		`"prd_accepted": true`,
		`"launch_ready": true`,
	} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessClosesContinuousVoiceGapForHostProductChain(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessXiaozhiHostReportFixture(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:    server.URL,
		DeviceID:      "stackchan-001",
		XiaozhiReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_TEXT_STREAM_PROFILE=local_ollama",
		"A21_TTS_FAST_PROFILE=sherpa_onnx_tts",
	})

	if !report.Voice.ContinuousVoiceReady || !report.Voice.VoicePipeline.HostProductChainReady {
		t.Fatalf("voice readiness = %+v, want host product-chain ready without physical promotion", report.Voice)
	}
	if report.LaunchReady || report.CanonicalDecision.LaunchReady || report.CanonicalDecision.PRDAccepted {
		t.Fatalf("canonical decision = %+v, launch = %v; host-only voice must stay below PRD acceptance", report.CanonicalDecision, report.LaunchReady)
	}
	if containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "continuous_voice_pipeline") {
		t.Fatalf("missing real evidence = %#v, want continuous voice gap closed by host product-chain evidence", report.CanonicalDecision.MissingRealEvidence)
	}
	for _, want := range []string{"real_provider_smoke", "v21_professional_execution", "physical_stackchan_online", "physical_stackchan_prd_acceptance"} {
		if !containsExactProductString(report.CanonicalDecision.MissingRealEvidence, want) {
			t.Fatalf("missing real evidence = %#v, want remaining launch gate %q preserved", report.CanonicalDecision.MissingRealEvidence, want)
		}
	}
}

func TestProductReadinessKeepsContinuousVoiceGapForFixtureOnlyXiaozhi(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	data := strings.Replace(productReadinessXiaozhiHostReportFixtureJSON(), `"voice_pipeline_execution_mode": "host_local"`, `"voice_pipeline_execution_mode": "fixture"`, 1)
	data = strings.Replace(data, `"asr_profile": "sherpa_onnx"`, `"asr_profile": "mock-local-asr"`, 1)
	data = strings.Replace(data, `"llm_profile": "ollama_local"`, `"llm_profile": "mock"`, 1)
	data = strings.Replace(data, `"tts_profile": "sherpa_onnx_tts"`, `"tts_profile": "mock-fast-tts"`, 1)
	data = strings.Replace(data, `"host_local_asr_executed": true`, `"host_local_asr_executed": false`, 1)
	data = strings.Replace(data, `"host_local_text_executed": true`, `"host_local_text_executed": false`, 1)
	data = strings.Replace(data, `"host_local_tts_executed": true`, `"host_local_tts_executed": false`, 1)
	fixture := writeProductReadinessXiaozhiHostReportFixtureFromData(t, data)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:    server.URL,
		DeviceID:      "stackchan-001",
		XiaozhiReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
	})

	if report.Voice.ContinuousVoiceReady || report.Voice.VoicePipeline.HostProductChainReady {
		t.Fatalf("voice readiness = %+v, want fixture-only report below continuous voice readiness", report.Voice)
	}
	if report.ServerSide.HostVoiceLoopbackReady {
		t.Fatalf("server-side readiness = %+v, want fixture-only report below host voice product-chain readiness", report.ServerSide)
	}
	if !containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "continuous_voice_pipeline") {
		t.Fatalf("missing real evidence = %#v, want continuous voice gap preserved for fixture-only report", report.CanonicalDecision.MissingRealEvidence)
	}
}

func TestProductReadinessHonorsExplicitXiaozhiHostProductChainFalse(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	data := strings.Replace(productReadinessXiaozhiHostReportFixtureJSON(), `"host_local_tts_executed": true`, `"host_local_tts_executed": true,
    "host_product_chain_ready": false`, 1)
	fixture := writeProductReadinessXiaozhiHostReportFixtureFromData(t, data)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:    server.URL,
		DeviceID:      "stackchan-001",
		XiaozhiReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_TEXT_STREAM_PROFILE=local_ollama",
		"A21_TTS_FAST_PROFILE=sherpa_onnx_tts",
	})

	if report.Voice.ContinuousVoiceReady || report.Voice.VoicePipeline.HostProductChainReady {
		t.Fatalf("voice readiness = %+v, want explicit false host_product_chain_ready to block continuous voice readiness", report.Voice)
	}
	if !containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "continuous_voice_pipeline") {
		t.Fatalf("missing real evidence = %#v, want continuous voice gap preserved when explicit product-chain flag is false", report.CanonicalDecision.MissingRealEvidence)
	}
}

func TestProductReadinessKeepsContinuousVoiceGapForInsufficientXiaozhiRounds(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	data := strings.Replace(productReadinessXiaozhiHostReportFixtureJSON(), `"repeat": 3`, `"repeat": 1`, 1)
	data = strings.Replace(data, `"answer_turn_count": 3`, `"answer_turn_count": 1`, 1)
	data = strings.Replace(data, `"barge_in_turn_count": 3`, `"barge_in_turn_count": 1`, 1)
	fixture := writeProductReadinessXiaozhiHostReportFixtureFromData(t, data)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:    server.URL,
		DeviceID:      "stackchan-001",
		XiaozhiReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_TEXT_STREAM_PROFILE=local_ollama",
		"A21_TTS_FAST_PROFILE=sherpa_onnx_tts",
	})

	if report.Voice.ContinuousVoiceReady || report.Voice.VoicePipeline.HostProductChainReady {
		t.Fatalf("voice readiness = %+v, want one-round xiaozhi report below continuous voice readiness", report.Voice)
	}
	if report.ServerSide.HostVoiceLoopbackReady {
		t.Fatalf("server-side readiness = %+v, want one-round xiaozhi report below host voice readiness", report.ServerSide)
	}
	if !containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "continuous_voice_pipeline") {
		t.Fatalf("missing real evidence = %#v, want continuous voice gap preserved for insufficient host rounds", report.CanonicalDecision.MissingRealEvidence)
	}
}
