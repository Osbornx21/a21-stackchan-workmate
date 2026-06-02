package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/providers"
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

func TestProductReadinessReportsMockDemoWithoutFullURLLeak(t *testing.T) {
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

	if report.Status != "mock_demo_ready" || !report.DemoReady || report.LaunchReady {
		t.Fatalf("status/demo/launch = %q/%v/%v, want mock_demo_ready/true/false", report.Status, report.DemoReady, report.LaunchReady)
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
		ProviderSmokeReport:     writeProductReadinessProviderSmokeReportFixture(t),
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

	if report.Status != "real_launch_ready" || !report.LaunchReady || !report.DemoReady {
		t.Fatalf("status/launch/demo = %q/%v/%v, want real_launch_ready/true/true", report.Status, report.LaunchReady, report.DemoReady)
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
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`))
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

func TestProductReadinessReportsServerSideCandidateWhenEvidenceSlicesPass(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:            server.URL,
		DeviceID:              "stackchan-001",
		ProviderSmokeReport:   writeProductReadinessProviderSmokeReportFixture(t),
		V21AdapterSmokeReport: writeProductReadinessV21AdapterSmokeReportFixture(t),
		XiaozhiReport:         writeProductReadinessXiaozhiHostReportFixture(t),
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
		"A21_V21_ADAPTER_URL=" + server.URL,
		"A21_TEXT_STREAM_PROFILE=deepseek",
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_TTS_FAST_PROFILE=sherpa_onnx_tts",
	})

	if report.Status != "server_side_candidate_ready" || !report.ServerSide.CandidateReady {
		t.Fatalf("status/server-side = %q/%+v, want server_side_candidate_ready", report.Status, report.ServerSide)
	}
	if report.LaunchReady || report.ServerSide.PRDAccepted {
		t.Fatalf("launch/server-prd = %v/%v, want no-hardware candidate below PRD acceptance", report.LaunchReady, report.ServerSide.PRDAccepted)
	}
	if report.ServerSide.AcceptanceStatus != "server_side_candidate_ready" ||
		!report.ServerSide.GatewayReady ||
		!report.ServerSide.ProviderEvidenceReady ||
		!report.ServerSide.V21ProfessionalEvidenceReady ||
		!report.ServerSide.HostVoiceLoopbackReady ||
		!report.ServerSide.WakeWordReady ||
		!report.ServerSide.RequiresPhysicalAcceptance {
		t.Fatalf("server-side readiness = %+v, want full no-hardware candidate and physical gate preserved", report.ServerSide)
	}
	if report.ServerSide.ProviderSmokeSourceReport != "a21-provider-smoke-real.json" ||
		report.ServerSide.V21ProfessionalSourceReport != "a21-v21-adapter-smoke-real.json" ||
		report.ServerSide.HostVoiceSourceReport != "a21-xiaozhi-host-local-report.json" {
		t.Fatalf("server-side source reports = %+v, want basename-only sources", report.ServerSide)
	}
	if len(report.ServerSide.MissingEvidence) != 0 {
		t.Fatalf("missing server-side evidence = %#v, want none", report.ServerSide.MissingEvidence)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{server.URL, "http://", "https://", "/Users/", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("server-side candidate leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestRunProviderEvidenceImportMakes5080labProviderSmokeUsable(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	dir := t.TempDir()
	bundle := writeProviderEvidenceImportBundle(t, map[string]string{
		"a21-provider-smoke-20260602-120000.json": productReadinessProviderSmokeReportFixtureJSON(),
	})
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "secret-value")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-evidence-import", "--bundle", bundle, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"schema_version": "a21.provider_evidence_import.v1"`,
		`"status": "accepted"`,
		`"provider_smoke_ready": true`,
		`"source_report": "a21-provider-smoke-20260602-120000.json"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{bundle, dir, "http://", "https://", "/Users/", "secret-value"} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("provider evidence import leaked %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("product-readiness code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report productReadinessReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode product readiness report: %v\n%s", err, stdout.String())
	}
	if !report.Provider.RealProviderReady || !report.Provider.SmokeEvidenceValid || !report.Provider.SmokeExecuted {
		t.Fatalf("provider readiness = %+v, want imported 5080lab provider smoke evidence", report.Provider)
	}
}

func TestRunProviderEvidenceImportRejectsSelectedProviderMismatchWithoutLeak(t *testing.T) {
	dir := t.TempDir()
	bundle := writeProviderEvidenceImportBundle(t, map[string]string{
		"a21-provider-smoke-20260602-120000.json": productReadinessProviderSmokeReportFixtureForProvider("local_ollama"),
	})
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "secret-value")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-evidence-import", "--bundle", bundle, "--output-dir", dir}, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("code = 0, want selected-provider mismatch rejected: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	rendered := stdout.String() + stderr.String()
	for _, want := range []string{`"status": "rejected"`, `"provider_smoke_ready": false`, `"code": "provider_smoke_report_mismatch"`} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
	for _, forbidden := range []string{bundle, dir, "secret-value", "http://", "https://", "/Users/"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("mismatched provider import leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunProviderEvidencePackageCreatesImportable5080labBundle(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	inputDir := t.TempDir()
	outputDir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, inputDir, "a21-provider-smoke-20260602-120000.json", productReadinessProviderSmokeReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, inputDir, "a21-provider-audio-smoke-cloud-asr.json", `{
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
  "report_path": "D:\\a21-mainland-latency-lab\\outbox\\a21-provider-audio-smoke-cloud-asr.json"
}`)
	writeProductReadinessReportFixtureFile(t, inputDir, "a21-provider-audio-smoke-cloud-tts.json", `{
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
  "report_path": "D:\\a21-mainland-latency-lab\\outbox\\a21-provider-audio-smoke-cloud-tts.json"
}`)
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "secret-value")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-evidence-package", "--input-dir", inputDir, "--output-dir", outputDir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"schema_version": "a21.provider_evidence_package.v1"`,
		`"status": "accepted"`,
		`"provider_smoke_ready": true`,
		`"source_report": "a21-provider-smoke-20260602-120000.json"`,
		`"a21-provider-audio-smoke-cloud-asr.json"`,
		`"a21-provider-audio-smoke-cloud-tts.json"`,
		`"bundle_path": "a21-5080lab-provider-evidence-`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{inputDir, outputDir, "http://", "https://", "/Users/", "secret-value"} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("provider evidence package leaked %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
	bundles, err := filepath.Glob(filepath.Join(outputDir, "a21-5080lab-provider-evidence-*.tgz"))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundles) != 1 {
		t.Fatalf("bundles = %#v, want one 5080lab provider evidence bundle", bundles)
	}

	importDir := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"provider-evidence-import", "--bundle", bundles[0], "--output-dir", importDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("import code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"a21-provider-audio-smoke-cloud-asr.json"`,
		`"a21-provider-audio-smoke-cloud-tts.json"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("import stdout missing %q: %s", want, stdout.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"provider-compat-matrix", "--use-latest-reports", "--reports-dir", importDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("provider-compat-matrix code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var matrix providerCompatMatrixReport
	if err := json.Unmarshal(stdout.Bytes(), &matrix); err != nil {
		t.Fatalf("decode provider compat matrix: %v\n%s", err, stdout.String())
	}
	if !matrix.Coverage.CloudASR || !matrix.Coverage.CloudTTS {
		t.Fatalf("coverage = %+v, want imported cloud ASR/TTS audio smoke coverage", matrix.Coverage)
	}
	for _, forbidden := range []string{inputDir, outputDir, importDir, "D:\\", "D:/", "outbox", "secret-value"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("provider compat matrix leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", importDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("product-readiness code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report productReadinessReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode product readiness report: %v\n%s", err, stdout.String())
	}
	if !report.Provider.RealProviderReady || !report.Provider.SmokeEvidenceValid || !report.Provider.SmokeExecuted {
		t.Fatalf("provider readiness = %+v, want packaged imported provider smoke evidence", report.Provider)
	}
}

func TestRunProviderEvidencePackageRejectsSelectedProviderMismatchWithoutLeak(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, inputDir, "a21-provider-smoke-20260602-120000.json", productReadinessProviderSmokeReportFixtureForProvider("local_ollama"))
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "secret-value")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-evidence-package", "--input-dir", inputDir, "--output-dir", outputDir}, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("code = 0, want selected-provider mismatch rejected: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	rendered := stdout.String() + stderr.String()
	for _, want := range []string{`"status": "rejected"`, `"provider_smoke_ready": false`, `"code": "provider_smoke_report_mismatch"`} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
	for _, forbidden := range []string{inputDir, outputDir, "secret-value", "http://", "https://", "/Users/"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("mismatched provider package leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	bundles, err := filepath.Glob(filepath.Join(outputDir, "a21-5080lab-provider-evidence-*.tgz"))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundles) != 0 {
		t.Fatalf("bundles = %#v, want no bundle for selected-provider mismatch", bundles)
	}
}

func TestRunProviderEvidencePackageRejectsUnsafeReportsWithoutLeak(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()
	unsafeReport := strings.Replace(productReadinessProviderSmokeReportFixtureJSON(), `"provider": "deepseek"`, `"provider": "secret-value"`, 1)
	unsafeReport = strings.Replace(unsafeReport, `"report_path": "a21-provider-smoke-real.json"`, `"local_path": "/Users/private/a21/report.json", "report_path": "a21-provider-smoke-real.json"`, 1)
	writeProductReadinessReportFixtureFile(t, inputDir, "a21-provider-smoke-20260602-120000.json", unsafeReport)
	writeProductReadinessReportFixtureFile(t, inputDir, "provider.env", "A21_LAB_DEEPSEEK_API_KEY=secret-value")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-evidence-package", "--input-dir", inputDir, "--output-dir", outputDir}, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("code = 0, want unsafe package rejected: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	rendered := stdout.String() + stderr.String()
	for _, want := range []string{`"status": "rejected"`, `"provider_smoke_ready": false`} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
	for _, forbidden := range []string{inputDir, outputDir, "http://", "https://", "/Users/", "secret-value", "../"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("unsafe provider package leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	bundles, err := filepath.Glob(filepath.Join(outputDir, "a21-5080lab-provider-evidence-*.tgz"))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundles) != 0 {
		t.Fatalf("bundles = %#v, want no bundle from unsafe reports", bundles)
	}
}

func TestRunProviderEvidenceImportRejectsUnsafeBundleWithoutLeak(t *testing.T) {
	dir := t.TempDir()
	bundle := writeProviderEvidenceImportBundle(t, map[string]string{
		"../a21-provider-smoke-20260602-120000.json": productReadinessProviderSmokeReportFixtureJSON(),
		"a21-provider-smoke-20260602-120001.json":    strings.Replace(productReadinessProviderSmokeReportFixtureJSON(), `"provider": "deepseek"`, `"provider": "secret-value"`, 1),
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-evidence-import", "--bundle", bundle, "--output-dir", dir}, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("code = 0, want unsafe bundle rejected: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	rendered := stdout.String() + stderr.String()
	for _, want := range []string{`"status": "rejected"`, `"provider_smoke_ready": false`} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
	for _, forbidden := range []string{bundle, dir, "http://", "https://", "/Users/", "secret-value", "../"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("unsafe provider bundle leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-provider-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("provider smoke files = %#v, want no import from unsafe bundle", matches)
	}
}

func TestProductReadinessServerSideCandidateRequiresHostVoiceLoopback(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:            server.URL,
		DeviceID:              "stackchan-001",
		ProviderSmokeReport:   writeProductReadinessProviderSmokeReportFixture(t),
		V21AdapterSmokeReport: writeProductReadinessV21AdapterSmokeReportFixture(t),
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
		"A21_V21_ADAPTER_URL=" + server.URL,
	})

	if report.ServerSide.CandidateReady {
		t.Fatalf("server-side readiness = %+v, want blocked without host voice loopback", report.ServerSide)
	}
	if report.ServerSide.AcceptanceStatus != "server_side_blocked" ||
		!report.ServerSide.ProviderEvidenceReady ||
		!report.ServerSide.V21ProfessionalEvidenceReady ||
		report.ServerSide.HostVoiceLoopbackReady {
		t.Fatalf("server-side readiness = %+v, want provider/v21 ready but host voice missing", report.ServerSide)
	}
	if !containsExactProductString(report.ServerSide.MissingEvidence, "host_voice_loopback") {
		t.Fatalf("missing server-side evidence = %#v, want host_voice_loopback", report.ServerSide.MissingEvidence)
	}
}

func TestProductReadinessIngestsPhysicalStackChanCandidateEvidence(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","identity_status":"valid","connection_status":"online","capabilities":{"microphone":"available_core_s3_i2s_24k_to_a21_16k"},"first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessPhysicalStackChanReportFixture(t, nil)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:              server.URL,
		DeviceID:                "stackchan-001",
		PhysicalStackChanReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
		"A21_V21_ADAPTER_URL=" + server.URL,
	})

	physical := report.StackChan.PhysicalEvidence
	if report.LaunchReady || physical.PRDAccepted {
		t.Fatalf("launch/prd = %v/%v, want candidate evidence not accepted", report.LaunchReady, physical.PRDAccepted)
	}
	if !physical.Valid ||
		physical.SourceReport != "a21-physical-stackchan-evidence-report.json" ||
		physical.ExecutionMode != "physical_stackchan" ||
		physical.PromotionGate != "candidate" ||
		physical.AcceptanceStatus != "physical_review_required" ||
		!physical.RequiredPhysicalMetricsAvailable ||
		!physical.MicEvidenceAvailable ||
		!physical.OperatorInstrumentObservationAvailable ||
		physical.HostLoopbackOnly ||
		!physical.CandidatePhysicalEvidence {
		t.Fatalf("physical evidence = %+v, want candidate physical evidence needing review", physical)
	}
	for _, want := range []string{"device_downlink_first_frame_ms", "device_playback_start_ms", "speech_end_to_first_audible_response_ms", "barge_in_stop_ms"} {
		if !physical.CanonicalMetricAvailability[want] {
			t.Fatalf("metric availability[%s] = false in %+v", want, physical.CanonicalMetricAvailability)
		}
	}
	if !containsProductAction(report.NextActions, "human physical StackChan review") ||
		!containsProductFinding(report.Findings, "physical_stackchan_review_required", "") {
		t.Fatalf("next actions/findings = %#v / %#v, want review-required state", report.NextActions, report.Findings)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{fixture, filepath.Dir(fixture), "operator transcript", "raw_audio", "provider output", "secret-token", "http://", "https://", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessIngestsPhysicalStackChanHostLoopbackAsHostOnly(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessPhysicalStackChanReportFixture(t, map[string]any{
		"execution_mode":    "host_loopback",
		"promotion_gate":    "not_production",
		"acceptance_status": "candidate_host_only",
		"prd_accepted":      false,
		"execution": map[string]any{
			"provider_executed": false,
			"v21_executed":      false,
			"hardware_executed": false,
		},
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:              server.URL,
		DeviceID:                "stackchan-001",
		PhysicalStackChanReport: fixture,
	}, []string{"A21_PROVIDER_PRIMARY=mock"})

	physical := report.StackChan.PhysicalEvidence
	if report.LaunchReady || physical.PRDAccepted || !physical.HostLoopbackOnly || physical.CandidatePhysicalEvidence || physical.PRDPhysicalAccepted {
		t.Fatalf("launch/physical = %v/%+v, want host-only evidence blocked", report.LaunchReady, physical)
	}
	if !physical.Valid || physical.AcceptanceStatus != "candidate_host_only" || physical.PromotionGate != "not_production" {
		t.Fatalf("physical evidence = %+v, want candidate_host_only not production", physical)
	}
	if !containsProductFinding(report.Findings, "physical_stackchan_host_loopback_only", "") {
		t.Fatalf("findings = %#v, want host-loopback physical finding", report.Findings)
	}
}

func TestRunProductReadinessCommandRejectsUnsafePhysicalStackChanReportWithoutLeak(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessPhysicalStackChanReportFixture(t, map[string]any{
		"transcript":      "operator transcript should not leak",
		"raw_audio":       "raw_audio_bytes",
		"provider_output": "provider output should not leak",
		"url":             "http://example.com/unsafe/full/url",
		"proxy":           "http://user:secret-token@proxy.local:7890",
		"local_path":      filepath.Join(t.TempDir(), "secret.wav"),
	})
	dir := t.TempDir()
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

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--physical-stackchan-report", fixture, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0 with invalid finding: %s", code, stderr.String())
	}
	rendered := stdout.String()
	if !strings.Contains(rendered, `"code": "physical_stackchan_report_invalid"`) {
		t.Fatalf("stdout missing fixed invalid finding: %s", rendered)
	}
	for _, forbidden := range []string{server.URL, fixture, filepath.Dir(fixture), "operator transcript should not leak", "raw_audio_bytes", "provider output should not leak", "secret-token", "proxy.local:7890", "http://", "https://"} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("unsafe physical report leaked %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestProductReadinessFutureAcceptedPhysicalReportRequiresCompleteEvidence(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","identity_status":"valid","connection_status":"online","capabilities":{"microphone":"available_core_s3_i2s_24k_to_a21_16k"},"first_seen_ms":1,"last_seen_ms":2}]}`)
	accepted := writeProductReadinessPhysicalStackChanReportFixture(t, map[string]any{"promotion_gate": "accepted", "acceptance_status": "prd_accepted", "prd_accepted": true})
	incompleteAccepted := writeProductReadinessPhysicalStackChanReportFixture(t, map[string]any{
		"promotion_gate":    "accepted",
		"acceptance_status": "prd_accepted",
		"prd_accepted":      true,
		"canonical_metrics": map[string]any{
			"device_downlink_first_frame_ms": map[string]any{"available": true, "value_ms": 430, "source": "device_runtime_echo"},
			"device_playback_start_ms":       map[string]any{"available": false},
		},
	})
	for _, tc := range []struct {
		name string
		path string
		want bool
	}{
		{name: "complete accepted", path: accepted, want: true},
		{name: "incomplete accepted", path: incompleteAccepted, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := buildProductReadinessReport(context.Background(), productReadinessOptions{
				GatewayURL:              server.URL,
				DeviceID:                "stackchan-001",
				PhysicalStackChanReport: tc.path,
			}, []string{
				"A21_PROVIDER_PRIMARY=mock",
			})
			if report.StackChan.PhysicalEvidence.PRDPhysicalAccepted != tc.want {
				t.Fatalf("physical evidence = %+v, want prd accepted %v", report.StackChan.PhysicalEvidence, tc.want)
			}
		})
	}
}

func TestProductReadinessIngestsV21ProfessionalReadinessReport(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessV21ProfessionalReportFixture(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:            server.URL,
		DeviceID:              "stackchan-001",
		V21ProfessionalReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
	})

	if report.LaunchReady || report.V21.Professional.PRDAccepted {
		t.Fatalf("launch/prd = %v/%v, want host-only professional report not accepted", report.LaunchReady, report.V21.Professional.PRDAccepted)
	}
	professional := report.V21.Professional
	if !professional.Valid ||
		!professional.CheckingAckWithin1200 ||
		!professional.EvidenceAvailable ||
		!professional.CardsAvailable ||
		!professional.FollowUpsAvailable ||
		professional.EvidenceCount != 2 ||
		professional.CardCount != 1 ||
		professional.FollowUpCount != 2 ||
		professional.ProfessionalAcceptanceStatus != "host_mock_ready" ||
		professional.SourceReport != "a21-v21-professional-readiness-host.json" ||
		professional.AdapterExecuted {
		t.Fatalf("professional readiness = %+v, want valid host-only professional contract", professional)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"professional"`,
		`"checking_ack_within_1200": true`,
		`"evidence_available": true`,
		`"cards_available": true`,
		`"follow_ups_available": true`,
		`"evidence_count": 2`,
		`"card_count": 1`,
		`"follow_up_count": 2`,
		`"professional_acceptance_status": "host_mock_ready"`,
		`"source_report": "a21-v21-professional-readiness-host.json"`,
		`"adapter_executed": false`,
		`"prd_accepted": false`,
		`"launch_ready": false`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("product readiness missing %q: %s", want, encoded.String())
		}
	}
	for _, forbidden := range []string{
		fixture,
		filepath.Dir(fixture),
		"professional readiness fixture query",
		"raw evidence text",
		"provider output",
		"secret-token",
		"http://",
		"https://",
		`"launch_ready": true`,
		`"prd_accepted": true`,
	} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessIngestsXiaozhiProfessionalGatewayReport(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:            server.URL,
		DeviceID:              "stackchan-001",
		V21ProfessionalReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
		"A21_V21_ADAPTER_URL=" + server.URL,
	})

	if report.LaunchReady || report.V21.Professional.PRDAccepted {
		t.Fatalf("launch/prd = %v/%v, want professional gateway evidence below launch gates", report.LaunchReady, report.V21.Professional.PRDAccepted)
	}
	if !report.V21.QueryExecuted {
		t.Fatalf("v21 readiness = %+v, want query executed from xiaozhi professional report", report.V21)
	}
	professional := report.V21.Professional
	if !professional.Valid ||
		!professional.CheckingAckWithin1200 ||
		!professional.EvidenceAvailable ||
		!professional.CardsAvailable ||
		!professional.FollowUpsAvailable ||
		professional.EvidenceCount != 5 ||
		professional.CardCount != 1 ||
		professional.FollowUpCount != 1 ||
		professional.ProfessionalAcceptanceStatus != "external_gateway_ready" ||
		professional.SourceReport != "a21-xiaozhi-professional-gateway.json" ||
		!professional.AdapterExecuted {
		t.Fatalf("professional readiness = %+v, want valid external Gateway professional contract", professional)
	}
	if containsProductAction(report.NextActions, "v21-adapter-smoke") {
		t.Fatalf("next actions = %#v, should not ask for adapter smoke after gateway professional proof", report.NextActions)
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
		"/Users/",
		"raw transcript text",
		"provider output",
		`"launch_ready": true`,
		`"prd_accepted": true`,
	} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessKeepsProfessionalHostReportBelowLaunchGates(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessV21ProfessionalReportFixture(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:            server.URL,
		DeviceID:              "stackchan-001",
		V21ProfessionalReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
	})

	if !report.V21.Professional.Valid {
		t.Fatalf("professional readiness = %+v, want ingested host report", report.V21.Professional)
	}
	if report.V21.Configured || report.V21.Healthy || report.StackChan.PhysicalDeviceOnline || report.LaunchReady || report.V21.Professional.PRDAccepted {
		t.Fatalf("readiness overclaimed launch from host report: v21=%+v stackchan=%+v launch=%v", report.V21, report.StackChan, report.LaunchReady)
	}
	if !containsProductAction(report.NextActions, "A21 V21 adapter boundary") || !containsProductAction(report.NextActions, "physical StackChan") {
		t.Fatalf("next actions = %#v, want real adapter and physical StackChan gates", report.NextActions)
	}
}

func TestProductReadinessRejectsV21ProfessionalReportMissingRequiredFields(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	tests := []struct {
		name      string
		mutate    func(string) string
		wantField string
	}{
		{
			name: "missing checking ack within 1200",
			mutate: func(data string) string {
				return strings.Replace(data, `  "checking_ack_within_1200": true,`+"\n", "", 1)
			},
			wantField: "checking_ack_within_1200",
		},
		{
			name: "missing evidence available",
			mutate: func(data string) string {
				return strings.Replace(data, `  "evidence_available": true,`+"\n", "", 1)
			},
			wantField: "evidence_available",
		},
		{
			name: "missing cards available",
			mutate: func(data string) string {
				return strings.Replace(data, `  "cards_available": true,`+"\n", "", 1)
			},
			wantField: "cards_available",
		},
		{
			name: "missing follow ups available",
			mutate: func(data string) string {
				return strings.Replace(data, `  "follow_ups_available": true,`+"\n", "", 1)
			},
			wantField: "follow_ups_available",
		},
		{
			name: "missing status",
			mutate: func(data string) string {
				return strings.Replace(data, `  "professional_acceptance_status": "host_mock_ready",`+"\n", "", 1)
			},
			wantField: "professional_acceptance_status",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := writeProductReadinessV21ProfessionalReportFixtureFromData(t, tt.mutate(productReadinessV21ProfessionalReportFixtureJSON()))

			report := buildProductReadinessReport(context.Background(), productReadinessOptions{
				GatewayURL:            server.URL,
				DeviceID:              "stackchan-001",
				V21ProfessionalReport: fixture,
			}, []string{
				"A21_PROVIDER_PRIMARY=mock",
			})

			if report.V21.Professional.Valid {
				t.Fatalf("professional readiness = %+v, want missing field to block valid ingestion", report.V21.Professional)
			}
			if !containsProductFinding(report.Findings, "v21_professional_report_missing_field", tt.wantField) {
				t.Fatalf("findings = %#v, want missing-field finding for %s", report.Findings, tt.wantField)
			}
		})
	}
}

func TestRunProductReadinessCommandAcceptsV21ProfessionalReportAndRedactsOutput(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessV21ProfessionalReportFixture(t)
	dir, err := os.MkdirTemp("", "a21-product-readiness-prof-")
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
	for _, want := range []string{
		`"professional"`,
		`"checking_ack_within_1200": true`,
		`"evidence_count": 2`,
		`"source_report": "a21-v21-professional-readiness-host.json"`,
		`"adapter_executed": false`,
		`"prd_accepted": false`,
		`"launch_ready": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{server.URL, fixture, filepath.Dir(fixture), "raw evidence text", "professional readiness fixture query", "provider output", "secret-token", "http://", "https://", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout leaked or overclaimed %q: %s", forbidden, rendered)
		}
	}
}

func TestRunProductReadinessCommandAcceptsV21AdapterSmokeReportAndRedactsOutput(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessV21AdapterSmokeReportFixture(t)
	dir, err := os.MkdirTemp("", "a21-product-readiness-adapter-smoke-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	t.Setenv("A21_V21_ADAPTER_URL", server.URL)
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

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--v21-adapter-smoke-report", fixture, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"query_executed": true`,
		`"valid": true`,
		`"checking_ack_within_1200": true`,
		`"evidence_available": true`,
		`"cards_available": true`,
		`"follow_ups_available": true`,
		`"evidence_count": 5`,
		`"card_count": 1`,
		`"follow_up_count": 1`,
		`"professional_acceptance_status": "adapter_smoke_passed"`,
		`"source_report": "a21-v21-adapter-smoke-real.json"`,
		`"adapter_executed": true`,
		`"prd_accepted": false`,
		`"report_path"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{server.URL, fixture, filepath.Dir(fixture), "查一下语音唤醒误触发", "raw retrieved evidence", "provider output", "secret-token", "http://", "https://", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("product readiness leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestProductReadinessExposesV21ProfessionalExecutionForRealAdapterReport(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessV21AdapterSmokeReportFixture(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:              server.URL,
		DeviceID:                "stackchan-001",
		V21AdapterSmokeReport:   fixture,
		V21ProfessionalReport:   "",
		PhysicalStackChanReport: "",
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
		"A21_V21_ADAPTER_URL=" + server.URL,
	})

	execution := report.V21.ProfessionalExecution
	if !execution.Valid ||
		execution.SourceKind != "v21_adapter_smoke_report" ||
		execution.SourceReport != "a21-v21-adapter-smoke-real.json" ||
		!execution.QueryExecuted ||
		!execution.AdapterExecuted ||
		!execution.CheckingAckWithin1200 ||
		!execution.EvidenceAvailable ||
		!execution.CardsAvailable ||
		!execution.FollowUpsAvailable ||
		execution.EvidenceCount != 5 ||
		execution.CardCount != 1 ||
		execution.FollowUpCount != 1 ||
		execution.ProfessionalAcceptanceStatus != "adapter_smoke_passed" ||
		!execution.RedactionOK ||
		execution.PRDAccepted {
		t.Fatalf("v21 professional execution = %+v, want explicit redacted executed adapter evidence", execution)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	rendered := encoded.String()
	for _, want := range []string{
		`"v21_professional_execution"`,
		`"source_kind": "v21_adapter_smoke_report"`,
		`"source_report": "a21-v21-adapter-smoke-real.json"`,
		`"query_executed": true`,
		`"adapter_executed": true`,
		`"redaction_ok": true`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("product readiness missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{server.URL, fixture, filepath.Dir(fixture), "查一下语音唤醒误触发", "raw retrieved evidence", "provider output", "secret-token", "http://", "https://", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("product readiness leaked or overclaimed %q: %s", forbidden, rendered)
		}
	}
}

func TestProductReadinessCountsExecutedProfessionalReportWithoutLiveV21Health(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:            server.URL,
		DeviceID:              "stackchan-001",
		V21ProfessionalReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
	})

	if report.V21.Healthy || !report.V21.ProfessionalExecution.Valid {
		t.Fatalf("v21 health/execution = %v/%+v, want no live health but accepted external execution", report.V21.Healthy, report.V21.ProfessionalExecution)
	}
	if !report.ServerSide.V21ProfessionalEvidenceReady ||
		containsExactProductString(report.ServerSide.MissingEvidence, "v21_professional_smoke") ||
		containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "v21_professional_execution") {
		t.Fatalf("server/canonical = %+v/%+v, want executed professional report to close V21 evidence gap", report.ServerSide, report.CanonicalDecision)
	}
	if report.LaunchReady || report.CanonicalDecision.PRDAccepted {
		t.Fatalf("launch/canonical = %v/%v, want professional evidence to reduce gap without PRD overclaim", report.LaunchReady, report.CanonicalDecision.PRDAccepted)
	}
}

func TestProductReadinessRejectsWeakV21AdapterSmokeReport(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	tests := []struct {
		name      string
		mutate    func(string) string
		wantField string
	}{
		{
			name: "missing generated timestamp",
			mutate: func(data string) string {
				return strings.Replace(data, `  "generated_at_ms": 1780337400000,`+"\n", "", 1)
			},
			wantField: "generated_at_ms",
		},
		{
			name: "missing professional max first response",
			mutate: func(data string) string {
				return strings.Replace(data, `  "max_first_response_ms": 1200,`+"\n", "", 1)
			},
			wantField: "max_first_response_ms",
		},
		{
			name: "missing redaction flag",
			mutate: func(data string) string {
				return strings.Replace(data, `  "redaction_ok": true,`+"\n", "", 1)
			},
			wantField: "redaction_ok",
		},
		{
			name: "zero confidence",
			mutate: func(data string) string {
				return strings.Replace(data, `  "confidence": 0.77,`, `  "confidence": 0,`, 1)
			},
			wantField: "",
		},
		{
			name: "wrong privacy scope",
			mutate: func(data string) string {
				return strings.Replace(data, `  "privacy_scope": "professional_only",`, `  "privacy_scope": "private",`, 1)
			},
			wantField: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := writeProductReadinessV21AdapterSmokeReportFixtureFromData(t, tt.mutate(productReadinessV21AdapterSmokeReportFixtureJSON()))

			report := buildProductReadinessReport(context.Background(), productReadinessOptions{
				GatewayURL:            server.URL,
				DeviceID:              "stackchan-001",
				V21AdapterSmokeReport: fixture,
			}, []string{
				"A21_PROVIDER_PRIMARY=mock",
				"A21_V21_ADAPTER_URL=" + server.URL,
			})

			if report.V21.Professional.Valid || report.V21.QueryExecuted {
				t.Fatalf("v21 readiness = %+v, want weak adapter smoke rejected", report.V21)
			}
			code := "v21_adapter_smoke_report_invalid"
			if tt.wantField != "" {
				code = "v21_adapter_smoke_report_missing_field"
			}
			if !containsProductFinding(report.Findings, code, tt.wantField) {
				t.Fatalf("findings = %#v, want %s/%s", report.Findings, code, tt.wantField)
			}
			var encoded bytes.Buffer
			if err := writeJSONProductReadiness(&encoded, report); err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{fixture, filepath.Dir(fixture), server.URL, "查一下语音唤醒误触发", "http://", "https://"} {
				if strings.Contains(encoded.String(), forbidden) {
					t.Fatalf("product readiness leaked %q: %s", forbidden, encoded.String())
				}
			}
		})
	}
}

func TestRunProductReadinessCommandUsesLatestReportsWithoutPathLeak(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-provider-smoke-20260601-191000.json", productReadinessProviderSmokeReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-xiaozhi-voice-bench-20260601-191935.json", productReadinessXiaozhiHostReportFixtureJSON())
	newerFailedXiaozhi := strings.Replace(productReadinessXiaozhiHostReportFixtureJSON(), `"failure_count": 0`, `"failure_count": 3`, 1)
	writeProductReadinessReportFixtureFile(t, dir, "a21-xiaozhi-voice-bench-20260601-192500.json", newerFailedXiaozhi)
	professionalFixture := writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t)
	physicalFixture := writeProductReadinessPhysicalStackChanReportFixture(t, map[string]any{
		"promotion_gate":    "candidate",
		"acceptance_status": "physical_review_required",
		"prd_accepted":      false,
	})
	writeProductReadinessReportFixtureFile(t, dir, "a21-v21-adapter-smoke-20260601-161500.json", `{"schema_version":"a21.v21_adapter_smoke.v1","status":"passed"}`)
	copyProductReadinessReportFixture(t, professionalFixture, filepath.Join(dir, "a21-xiaozhi-professional-bench-20260601-160123.json"))
	copyProductReadinessReportFixture(t, physicalFixture, filepath.Join(dir, "a21-physical-stackchan-evidence-20260601-150001.json"))
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "secret-value")
	t.Setenv("A21_V21_ADAPTER_URL", server.URL)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"status": "server_side_candidate_ready"`,
		`"canonical_decision"`,
		`"authority": "a21.product_readiness.v1"`,
		`"full_prd_status": "server_side_candidate_only"`,
		`"host_only_evidence_use": "gap_reduction_only"`,
		`"server_side_candidate_ready": true`,
		`"missing_real_evidence"`,
		`"physical_stackchan_online"`,
		`"physical_stackchan_prd_acceptance"`,
		`"real_provider_ready": true`,
		`"smoke_evidence_valid": true`,
		`"smoke_source_report": "a21-provider-smoke-20260601-191000.json"`,
		`"source_report": "a21-xiaozhi-voice-bench-20260601-191935.json"`,
		`"latest_report_candidate_skipped"`,
		`"detail": "xiaozhi_voice:a21-xiaozhi-voice-bench-20260601-192500.json"`,
		`"continuous_voice_ready": true`,
		`"host_product_chain_ready": true`,
		`"professional_acceptance_status": "external_gateway_ready"`,
		`"source_report": "a21-xiaozhi-professional-bench-20260601-160123.json"`,
		`"source_report": "a21-physical-stackchan-evidence-20260601-150001.json"`,
		`"adapter_executed": true`,
		`"candidate_physical_evidence": true`,
		`"host_loopback_candidate_ready": true`,
		`"candidate_ready": true`,
		`"requires_physical_acceptance": true`,
		`"launch_ready": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{server.URL, dir, professionalFixture, physicalFixture, "http://", "https://", "/Users/", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("latest readiness leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
	if strings.Contains(rendered, "configure a real A21 provider") {
		t.Fatalf("latest readiness should not keep provider gap after provider smoke evidence: %s", rendered)
	}
	if strings.Contains(rendered, "continuous_voice_pipeline") {
		t.Fatalf("latest readiness should not keep continuous voice gap after host product-chain evidence: %s", rendered)
	}
	if strings.Contains(rendered, "v21_adapter_smoke_report_missing_field") {
		t.Fatalf("latest readiness should not ingest adapter-smoke noise when professional proof exists: %s", rendered)
	}
}

func TestRunProductReadinessLatestProviderSmokeSkipsMismatchedSelectedProvider(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	dir := t.TempDir()
	deepseekPath := writeProductReadinessReportFixtureFile(t, dir, "a21-provider-smoke-20260602-100000.json", productReadinessProviderSmokeReportFixtureJSON())
	localOllamaPath := writeProductReadinessReportFixtureFile(t, dir, "a21-provider-smoke-20260602-100100.json", productReadinessProviderSmokeReportFixtureForProvider("local_ollama"))
	older := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Minute)
	if err := os.Chtimes(deepseekPath, older, older); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(localOllamaPath, newer, newer); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "secret-value")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report productReadinessReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode product readiness report: %v\n%s", err, stdout.String())
	}
	if !report.Provider.RealProviderReady || !report.Provider.SmokeEvidenceValid || report.Provider.SmokeSourceReport != "a21-provider-smoke-20260602-100000.json" {
		t.Fatalf("provider readiness = %+v, want older selected deepseek evidence", report.Provider)
	}
	if containsProductFinding(report.Findings, "provider_smoke_report_mismatch", "") {
		t.Fatalf("findings = %#v, want mismatched latest skipped during selection, not attached", report.Findings)
	}
	if !containsProductFinding(report.Findings, "latest_report_candidate_skipped", "provider_smoke:a21-provider-smoke-20260602-100100.json") {
		t.Fatalf("findings = %#v, want skipped latest mismatched provider report", report.Findings)
	}
	for _, forbidden := range []string{server.URL, dir, deepseekPath, localOllamaPath, "secret-value", "http://", "https://", "/Users/"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("latest provider selection leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunProductReadinessUsesLatestRealtimeFixtureWithoutPathLeak(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-provider-realtime-fixture-20260602-101500.json", productReadinessRealtimeFixtureReportFixtureJSON())
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "secret-value")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "zh_female_kailangjiejie_moon_bigtts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"realtime_evidence_valid": true`,
		`"realtime_provider": "doubao_tts_realtime"`,
		`"realtime_family": "voice_hybrid"`,
		`"realtime_status": "passed"`,
		`"realtime_executed": true`,
		`"realtime_route_eligible": false`,
		`"realtime_source_report": "a21-provider-realtime-fixture-20260602-101500.json"`,
		`"voice_realtime_ready": false`,
		`"real_provider_ready": false`,
		`"launch_ready": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{server.URL, dir, "secret-value", "doubao-tts", "zh_female_kailangjiejie", "http://", "https://", "/Users/", `"launch_ready": true`, `"real_provider_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("latest realtime readiness leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunProductReadinessLatestV21SelectionPrefersNewestUsableAdapterSmoke(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-provider-smoke-20260602-100000.json", productReadinessProviderSmokeReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-xiaozhi-voice-bench-20260602-100100.json", productReadinessXiaozhiHostReportFixtureJSON())
	professionalFixture := writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t)
	professionalPath := copyProductReadinessReportFixture(t, professionalFixture, filepath.Join(dir, "a21-xiaozhi-professional-bench-20260602-100200.json"))
	adapterPath := writeProductReadinessReportFixtureFile(t, dir, "a21-v21-adapter-smoke-20260602-100300.json", productReadinessV21AdapterSmokeReportFixtureJSON())
	older := time.Date(2026, 6, 2, 10, 2, 0, 0, time.UTC)
	newer := older.Add(time.Minute)
	if err := os.Chtimes(professionalPath, older, older); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(adapterPath, newer, newer); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "secret-value")
	t.Setenv("A21_V21_ADAPTER_URL", server.URL)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report productReadinessReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode product readiness report: %v\n%s", err, stdout.String())
	}
	if report.V21.Professional.SourceReport != "a21-v21-adapter-smoke-20260602-100300.json" ||
		report.V21.Professional.ProfessionalAcceptanceStatus != "adapter_smoke_passed" ||
		report.V21.ProfessionalExecution.SourceKind != "v21_adapter_smoke_report" ||
		!report.V21.Professional.AdapterExecuted {
		t.Fatalf("v21 professional selection = %+v execution = %+v, want newest usable adapter smoke", report.V21.Professional, report.V21.ProfessionalExecution)
	}
	for _, forbidden := range []string{server.URL, dir, professionalFixture, professionalPath, adapterPath, "http://", "https://", "/Users/", "secret-value"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("latest v21 selection leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunServerSideReadinessBundleUsesLatestReportsWithoutPathLeak(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-provider-smoke-20260602-100000.json", productReadinessProviderSmokeReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-xiaozhi-voice-bench-20260602-100100.json", productReadinessXiaozhiHostReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-v21-adapter-smoke-20260602-100200.json", productReadinessV21AdapterSmokeReportFixtureJSON())
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "secret-value")
	t.Setenv("A21_V21_ADAPTER_URL", server.URL)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"server-side-readiness-bundle", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"schema_version": "a21.server_side_readiness_bundle.v1"`,
		`"status": "server_side_candidate_ready"`,
		`"canonical_decision"`,
		`"authority": "a21.product_readiness.v1"`,
		`"full_prd_status": "server_side_candidate_only"`,
		`"host_only_evidence_use": "gap_reduction_only"`,
		`"server_side_candidate_ready": true`,
		`"candidate_ready": true`,
		`"launch_ready": false`,
		`"prd_accepted": false`,
		`"requires_physical_acceptance": true`,
		`"product_readiness_status": "server_side_candidate_ready"`,
		`"provider_smoke_source_report": "a21-provider-smoke-20260602-100000.json"`,
		`"v21_professional_source_report": "a21-v21-adapter-smoke-20260602-100200.json"`,
		`"host_voice_source_report": "a21-xiaozhi-voice-bench-20260602-100100.json"`,
		`"payloads_stored": false`,
		`"report_path": "a21-server-side-readiness-bundle-`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-server-side-readiness-bundle-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("bundle reports = %d, want 1: %v", len(matches), matches)
	}
	bundleData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		server.URL,
		dir,
		"http://",
		"https://",
		"/Users/",
		"secret-value",
		`"launch_ready": true`,
		`"prd_accepted": true`,
		`"missing_evidence"`,
	} {
		if strings.Contains(rendered, forbidden) || strings.Contains(string(bundleData), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("server-side bundle leaked or overclaimed %q: stdout=%s report=%s stderr=%s", forbidden, rendered, bundleData, stderr.String())
		}
	}
}

func TestRunServerSideReadinessBundleRequireCandidateFailsWithMissingHostVoice(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-provider-smoke-20260602-100000.json", productReadinessProviderSmokeReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-v21-adapter-smoke-20260602-100200.json", productReadinessV21AdapterSmokeReportFixtureJSON())
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "secret-value")
	t.Setenv("A21_V21_ADAPTER_URL", server.URL)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"server-side-readiness-bundle", "--gateway-url", server.URL, "--use-latest-reports", "--require-candidate", "--output-dir", dir}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"status": "server_side_blocked"`,
		`"candidate_ready": false`,
		`"host_voice_loopback"`,
		`"go run ./cmd/a21 xiaozhi-voice-bench --repeat 3 --require-product-chain --output-dir reports"`,
		`"report_path": "a21-server-side-readiness-bundle-`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{server.URL, dir, "http://", "https://", "/Users/", "secret-value", `"candidate_ready": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("server-side bundle leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunServerSideReadinessBundleDoesNotCollectMockProviderAsServerEvidence(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-xiaozhi-voice-bench-20260602-100100.json", productReadinessXiaozhiHostReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-v21-adapter-smoke-20260602-100200.json", productReadinessV21AdapterSmokeReportFixtureJSON())
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"server-side-readiness-bundle", "--gateway-url", server.URL, "--use-latest-reports", "--collect-missing", "--execute-provider-smoke", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0 without --require-candidate: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"status": "server_side_blocked"`,
		`"candidate_ready": false`,
		`"name": "provider_smoke"`,
		`"status": "skipped"`,
		`"reason": "configure a real A21 provider before provider smoke"`,
		"configure a real A21 provider with A21_PROVIDER_PRIMARY plus its required env names",
		`"launch_ready": false`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	providerReports, err := filepath.Glob(filepath.Join(dir, "a21-provider-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(providerReports) != 0 {
		t.Fatalf("provider reports = %v, want no mock provider execution", providerReports)
	}
	for _, forbidden := range []string{
		"provider-smoke --provider mock",
		server.URL,
		dir,
		"http://",
		"https://",
		"/Users/",
		`"candidate_ready": true`,
		`"launch_ready": true`,
		`"prd_accepted": true`,
	} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("server-side mock provider evidence leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunServerSideReadinessBundleCollectsMissingHostVoiceRequiresProductChain(t *testing.T) {
	gatewayServer := newGatewayServerFromEnv(nil)
	httpServer := httptest.NewServer(gatewayServer.Handler())
	t.Cleanup(httpServer.Close)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-provider-smoke-20260602-100000.json", productReadinessProviderSmokeReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-v21-adapter-smoke-20260602-100200.json", productReadinessV21AdapterSmokeReportFixtureJSON())
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "secret-value")
	t.Setenv("A21_V21_ADAPTER_URL", httpServer.URL)
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
		"--gateway-url", httpServer.URL,
		"--use-latest-reports",
		"--collect-missing",
		"--collect-repeat", "1",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"collection"`,
		`"enabled": true`,
		`"name": "host_voice_loopback"`,
		`"status": "failed"`,
		`"reason": "exit_code_1"`,
		`"command": "go run ./cmd/a21 xiaozhi-voice-bench --repeat 1 --require-product-chain --output-dir reports"`,
		`"source_report": "a21-xiaozhi-voice-bench-`,
		`"absorbed_by_readiness": false`,
		`"host_voice_loopback_ready": false`,
		`"go run ./cmd/a21 xiaozhi-voice-bench --repeat 3 --require-product-chain --output-dir reports"`,
		`"launch_ready": false`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-xiaozhi-voice-bench-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("voice bench reports = %d, want 1: %v", len(matches), matches)
	}
	for _, forbidden := range []string{httpServer.URL, dir, "http://", "https://", "/Users/", "secret-value", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("server-side collect leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunServerSideReadinessBundleCollectMissingSkipsExternalWithoutAuthorization(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-xiaozhi-voice-bench-20260602-100100.json", productReadinessXiaozhiHostReportFixtureJSON())
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "secret-value")
	t.Setenv("A21_V21_ADAPTER_URL", server.URL)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"server-side-readiness-bundle", "--gateway-url", server.URL, "--use-latest-reports", "--collect-missing", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0 without --require-candidate: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"status": "server_side_blocked"`,
		`"candidate_ready": false`,
		`"name": "provider_smoke"`,
		`"reason": "requires --execute-provider-smoke"`,
		`"name": "v21_professional_smoke"`,
		`"reason": "requires --execute-v21-smoke"`,
		`"provider_smoke"`,
		`"v21_professional_smoke"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	providerReports, err := filepath.Glob(filepath.Join(dir, "a21-provider-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(providerReports) != 0 {
		t.Fatalf("provider reports = %v, want no implicit provider execution", providerReports)
	}
	v21Reports, err := filepath.Glob(filepath.Join(dir, "a21-v21-adapter-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(v21Reports) != 0 {
		t.Fatalf("v21 reports = %v, want no implicit v21 execution", v21Reports)
	}
	for _, forbidden := range []string{server.URL, dir, "http://", "https://", "/Users/", "secret-value", `"candidate_ready": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("server-side collect leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

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
		"--gateway-url", newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`).URL,
		"--use-latest-reports",
		"--collect-missing",
		"--execute-provider-smoke",
		"--execute-v21-smoke",
		"--require-candidate",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if providerCalls != 3 || v21Calls != 1 {
		t.Fatalf("provider/v21 calls = %d/%d, want 3/1", providerCalls, v21Calls)
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"status": "server_side_candidate_ready"`,
		`"candidate_ready": true`,
		`"name": "provider_smoke"`,
		`"name": "v21_professional_smoke"`,
		`"execution_authorized": true`,
		`"external_execution": true`,
		`"absorbed_by_readiness": true`,
		`"source_report": "a21-provider-smoke-`,
		`"source_report": "a21-v21-adapter-smoke-`,
		`"provider_smoke_source_report": "a21-provider-smoke-`,
		`"v21_professional_source_report": "a21-v21-adapter-smoke-`,
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
	for _, forbidden := range []string{
		provider.URL,
		v21.URL,
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

func TestProductVoiceReadinessRejectsLegacyVoiceClonePaths(t *testing.T) {
	dir := t.TempDir()
	legacyDir := filepath.Join(dir, "x21-voice")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	commandPath := filepath.Join(legacyDir, "a21-voice-clone-wrapper")
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

	if voice.LocalTTSReady || voice.VoicePipeline.HostLocalTTSReady {
		t.Fatalf("voice readiness = %+v, want legacy clone path blocked", voice)
	}
}

func TestRunProductReadinessCommandWritesReport(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	dir := t.TempDir()
	ttsModelDir := createProductReadinessTTSModelDir(t)
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
	t.Setenv("A21_V21_ADAPTER_URL", "")
	t.Setenv("A21_LOCAL_TTS_ENGINE", "sherpa_onnx")
	t.Setenv("A21_SHERPA_ONNX_MODEL_DIR", ttsModelDir)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": "mock_demo_ready"`) {
		t.Fatalf("stdout missing mock demo readiness: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), server.URL) || strings.Contains(stdout.String(), "http://") {
		t.Fatalf("stdout leaked full URL: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), ttsModelDir) {
		t.Fatalf("stdout leaked local model path: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-product-readiness-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("product readiness report matches = %v, %v", matches, err)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(reportData), ttsModelDir) {
		t.Fatalf("report leaked local model path: %s", reportData)
	}
}

func TestRunProductReadinessCommandAcceptsXiaozhiReportAndRedactsOutput(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessXiaozhiHostReportFixture(t)
	dir := t.TempDir()
	t.Setenv("A21_PROVIDER_PRIMARY", "ollama_local")
	t.Setenv("A21_TEXT_STREAM_PROFILE", "ollama_local")
	t.Setenv("A21_ASR_LOCAL_PROFILE", "sherpa_onnx")
	t.Setenv("A21_TTS_FAST_PROFILE", "sherpa_onnx_tts")
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--xiaozhi-report", fixture, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"host_loopback_candidate_ready": true`,
		`"answer_first_audio_p95_ms": 386`,
		`"barge_in_stop_p95_ms": 0`,
		`"acceptance_status": "candidate_host_only"`,
		`"execution_mode": "host_local"`,
		`"asr_profile_env": "A21_ASR_LOCAL_PROFILE"`,
		`"text_stream_profile_env": "A21_TEXT_STREAM_PROFILE"`,
		`"tts_profile_env": "A21_TTS_FAST_PROFILE"`,
		`"source_report": "a21-xiaozhi-host-local-report.json"`,
		`"launch_ready": false`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{server.URL, fixture, filepath.Dir(fixture), "http://", "https://", "data_base64", "transcript", "provider output", "secret", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout leaked or overclaimed %q: %s", forbidden, rendered)
		}
	}
}

func TestProductReadinessAcceptsProductChainXiaozhiReportWithoutPromotingProvider(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessXiaozhiHostReportFixtureFromData(t, strings.Replace(productReadinessXiaozhiHostReportFixtureJSON(), `"provider_executed": false`, `"provider_executed": true`, 1))

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:    server.URL,
		DeviceID:      "stackchan-001",
		XiaozhiReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
		"A21_LOCAL_TTS_ENGINE=sherpa_onnx",
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_TEXT_STREAM_PROFILE=local_ollama",
		"A21_TTS_FAST_PROFILE=sherpa_onnx_tts",
	})

	if !report.Voice.VoicePipeline.HostLoopbackCandidateReady {
		t.Fatalf("voice pipeline = %+v, want host-loopback candidate ready", report.Voice.VoicePipeline)
	}
	if report.Provider.RealProviderReady || report.Provider.SmokeEvidenceValid {
		t.Fatalf("provider readiness = %+v, xiaozhi host-loopback must not promote real provider smoke", report.Provider)
	}
	if containsProductFinding(report.Findings, "xiaozhi_report_invalid", "") {
		t.Fatalf("findings = %#v, want product-chain xiaozhi report accepted", report.Findings)
	}
	if report.LaunchReady || report.Voice.VoicePipeline.PRDAccepted {
		t.Fatalf("launch/voice PRD = %v/%v, host-loopback remains candidate only", report.LaunchReady, report.Voice.VoicePipeline.PRDAccepted)
	}
}

func TestProductReadinessRejectsFixtureXiaozhiReportClaimingProviderExecution(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	data := strings.Replace(productReadinessXiaozhiHostReportFixtureJSON(), `"provider_executed": false`, `"provider_executed": true`, 1)
	data = strings.Replace(data, `"voice_pipeline_execution_mode": "host_local"`, `"voice_pipeline_execution_mode": "fixture"`, 1)
	fixture := writeProductReadinessXiaozhiHostReportFixtureFromData(t, data)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:    server.URL,
		DeviceID:      "stackchan-001",
		XiaozhiReport: fixture,
	}, nil)

	if report.Voice.VoicePipeline.HostLoopbackCandidateReady {
		t.Fatalf("voice pipeline = %+v, want fixture provider-execution claim rejected", report.Voice.VoicePipeline)
	}
	if !containsProductFinding(report.Findings, "xiaozhi_report_invalid", "") {
		t.Fatalf("findings = %#v, want invalid xiaozhi report finding", report.Findings)
	}
}

func TestRunProductReadinessCommandAcceptsLocalVoiceLoopbackReport(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessLocalVoiceLoopbackReportFixture(t)
	dir := t.TempDir()
	t.Setenv("A21_PROVIDER_PRIMARY", "local_ollama")
	t.Setenv("A21_TEXT_STREAM_PROFILE", "local_ollama")
	t.Setenv("A21_LOCAL_OLLAMA_MODEL", "qwen2.5:0.5b")
	t.Setenv("A21_LOCAL_OLLAMA_BASE_URL", "http://127.0.0.1:11434")
	t.Setenv("A21_ASR_LOCAL_PROFILE", "sherpa_onnx")
	t.Setenv("A21_TTS_FAST_PROFILE", "sherpa_onnx_tts")
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--xiaozhi-report", fixture, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"host_loopback_candidate_ready": true`,
		`"host_local_asr_ready": true`,
		`"host_local_text_ready": true`,
		`"host_local_tts_ready": true`,
		`"answer_first_audio_p95_ms": 1408.818`,
		`"barge_in_stop_p95_ms": 0.111`,
		`"acceptance_status": "host_local_loopback_passed"`,
		`"execution_mode": "host_local"`,
		`"asr_profile": "sherpa_onnx"`,
		`"text_stream_profile": "local_ollama"`,
		`"tts_profile": "sherpa_onnx_tts"`,
		`"source_report": "a21-local-voice-loopback-report.json"`,
		`"launch_ready": false`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{server.URL, fixture, filepath.Dir(fixture), "http://", "https://", "data_base64", "用户原文", "provider output", "secret", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stdout leaked or overclaimed %q: %s", forbidden, rendered)
		}
	}
}

func TestProductReadinessRejectsLocalVoiceLoopbackMissingRepeat(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	data := strings.Replace(productReadinessLocalVoiceLoopbackReportFixtureJSON(), `  "repeat": 3,`+"\n", "", 1)
	fixture := writeProductReadinessLocalVoiceLoopbackReportFixtureFromData(t, data)

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

	if report.Voice.ContinuousVoiceReady || report.ServerSide.HostVoiceLoopbackReady {
		t.Fatalf("voice/server readiness = %+v/%+v, want missing-repeat local loopback blocked", report.Voice, report.ServerSide)
	}
	if !containsProductFinding(report.Findings, "xiaozhi_report_missing_field", "repeat") {
		t.Fatalf("findings = %#v, want missing repeat finding", report.Findings)
	}
	if !containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "continuous_voice_pipeline") {
		t.Fatalf("missing real evidence = %#v, want continuous voice gap preserved", report.CanonicalDecision.MissingRealEvidence)
	}
}

func TestProductVoiceReadinessDetectsDefaultLocalModelCache(t *testing.T) {
	t.Chdir(t.TempDir())
	createProductReadinessModelFiles(t, filepath.Join(".a21-tools", "sherpa-onnx-models", "vits-icefall-zh-aishell3"), []string{
		"model.onnx",
		"lexicon.txt",
		"tokens.txt",
		"phone.fst",
		"date.fst",
		"number.fst",
	})
	createProductReadinessModelFiles(t, filepath.Join(".a21-tools", "sherpa-onnx-asr-models", "sherpa-onnx-paraformer-zh-small-2024-03-09"), []string{
		"model.int8.onnx",
		"tokens.txt",
	})

	voice := buildProductVoiceReadiness([]string{
		"A21_LOCAL_TTS_ENGINE=sherpa_onnx",
		"A21_LOCAL_ASR_PROVIDER=sherpa_onnx",
	}, productProviderReadiness{RealProviderReady: true}, productStackChanReadiness{
		PhysicalDeviceOnline:    true,
		PhysicalMicrophoneReady: true,
	})

	if !voice.LocalTTSReady || !voice.RealASRReady || !voice.ContinuousVoiceReady {
		t.Fatalf("voice readiness = %+v, want default local cache ready", voice)
	}
}

func TestProductVoiceReadinessSurfacesDefaultASRCacheWithoutExplicitEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	createProductReadinessModelFiles(t, filepath.Join(".a21-tools", "sherpa-onnx-asr-models", "sherpa-onnx-paraformer-zh-small-2024-03-09"), []string{
		"model.int8.onnx",
		"tokens.txt",
	})

	voice := buildProductVoiceReadiness(nil, productProviderReadiness{}, productStackChanReadiness{})

	if !voice.RealASRReady || voice.ASRProvider != "sherpa_onnx" {
		t.Fatalf("ASR readiness = provider:%q ready:%v, want repo-local sherpa cache ready", voice.ASRProvider, voice.RealASRReady)
	}
	if !voice.VoicePipeline.HostLocalASRReady || voice.VoicePipeline.ASRProfile != "sherpa_onnx" {
		t.Fatalf("pipeline ASR = %+v, want sherpa_onnx host-local candidate", voice.VoicePipeline)
	}
}

func newProductReadinessTestServer(t *testing.T, devicesJSON string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"service":"a21-gateway","status":"ok"}`))
		case "/simulator":
			w.Header().Set("content-type", "text/html")
			_, _ = w.Write([]byte("<!doctype html><title>A21 Simulator</title>"))
		case "/v1/devices":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(devicesJSON))
		case "/v1/wake-word":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.wake_word.v1","mode":"builtin_xiaozhi","active_phrase":"你好小智","active_pinyin":"ni hao xiao zhi","threshold":30,"runtime_status":"active_builtin_model","runtime_configurable":false,"firmware_build_required":false}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func createProductReadinessTTSModelDir(t *testing.T) string {
	t.Helper()
	return createProductReadinessModelFiles(t, filepath.Join(t.TempDir(), "a21-tts-model"), []string{
		"model.onnx",
		"lexicon.txt",
		"tokens.txt",
		"phone.fst",
		"date.fst",
		"number.fst",
	})
}

func createProductReadinessASRModelDir(t *testing.T) string {
	t.Helper()
	return createProductReadinessModelFiles(t, filepath.Join(t.TempDir(), "a21-asr-model"), []string{
		"model.int8.onnx",
		"tokens.txt",
	})
}

func createProductReadinessModelFiles(t *testing.T, dir string, names []string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("a21"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func setA21DirectProxyBypassForTest(t *testing.T) {
	t.Helper()
	direct := "localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16"
	t.Setenv("NO_PROXY", direct)
	t.Setenv("A21_NO_PROXY", direct)
}

func containsProductAction(actions []string, want string) bool {
	for _, action := range actions {
		if strings.Contains(action, want) {
			return true
		}
	}
	return false
}

func containsExactProductString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsProductFinding(findings []productReadinessFinding, code string, detail string) bool {
	for _, finding := range findings {
		if finding.Code == code && strings.Contains(finding.Detail, detail) {
			return true
		}
	}
	return false
}

func writeProductReadinessXiaozhiHostReportFixture(t *testing.T) string {
	t.Helper()
	return writeProductReadinessXiaozhiHostReportFixtureFromData(t, productReadinessXiaozhiHostReportFixtureJSON())
}

func writeProductReadinessXiaozhiHostReportFixtureFromData(t *testing.T, data string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-xiaozhi-host-local-report.json")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeProductReadinessReportFixtureFile(t *testing.T, dir string, name string, data string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func copyProductReadinessReportFixture(t *testing.T, source string, target string) string {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return target
}

func writeProviderEvidenceImportBundle(t *testing.T, entries map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-5080lab-provider-evidence-20260602-120000.tgz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	for name, content := range entries {
		data := []byte(content)
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(data)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func writeProductReadinessLocalVoiceLoopbackReportFixture(t *testing.T) string {
	t.Helper()
	return writeProductReadinessLocalVoiceLoopbackReportFixtureFromData(t, productReadinessLocalVoiceLoopbackReportFixtureJSON())
}

func writeProductReadinessLocalVoiceLoopbackReportFixtureFromData(t *testing.T, data string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-local-voice-loopback-report.json")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func productReadinessLocalVoiceLoopbackReportFixtureJSON() string {
	return `{
  "schema_version": "a21.audio.local_voice_loopback.v1",
  "status": "passed",
  "repeat": 3,
  "asr_provider": "sherpa_onnx",
  "text_stream_provider": "local_ollama",
  "text_stream_executed": true,
  "tts_provider": "sherpa_onnx",
  "answer_first_audio_total_p95_ms": 1408.818,
  "barge_in_stop_p95_ms": 0.111,
  "local_ack_first_audio_total_ms": 961.904,
  "asr_transcript_policy": "transcript_not_recorded",
  "text_stream_endpoint_host": "127.0.0.1:11434",
  "report_path": "reports/a21-local-voice-loopback-report.json"
}`
}

func writeProductReadinessV21ProfessionalReportFixture(t *testing.T) string {
	t.Helper()
	return writeProductReadinessV21ProfessionalReportFixtureFromData(t, productReadinessV21ProfessionalReportFixtureJSON())
}

func writeProductReadinessV21ProfessionalReportFixtureFromData(t *testing.T, data string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-v21-professional-readiness-host.json")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeProductReadinessProviderSmokeReportFixture(t *testing.T) string {
	t.Helper()
	return writeProductReadinessProviderSmokeReportFixtureFromData(t, productReadinessProviderSmokeReportFixtureJSON())
}

func writeProductReadinessProviderSmokeReportFixtureFromData(t *testing.T, data string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-provider-smoke-real.json")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeProductReadinessRealtimeFixtureReportFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-provider-realtime-fixture-real.json")
	if err := os.WriteFile(path, []byte(productReadinessRealtimeFixtureReportFixtureJSON()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-xiaozhi-professional-gateway.json")
	data := `{
  "schema_version": "a21.xiaozhi_professional_bench.v1",
  "generated_at_unix_ms": 1780300883954,
  "source_profile": "external_gateway",
  "scenario": "success",
  "acceptance_status": "external_gateway_ready",
  "prd_accepted": false,
  "gateway": "loopback:21080",
  "input_audio": {
    "source": "wav",
    "file": "a21-sherpa-onnx-tts.wav",
    "opus_frame_count": 18
  },
  "trace_id": "a21-trace-xiaozhi-professional-bench-001",
  "session_id": "a21-session-xiaozhi-professional-bench-001",
  "device_id": "stackchan-virtual-a21-professional-bench-001",
  "checking_feedback_observed": true,
  "checking_feedback_ms": 1,
  "checking_feedback_within_1200": true,
  "professional_result_observed": true,
  "professional_result_after_checking": true,
  "abort_stop_observed": true,
  "stale_result_suppressed": true,
  "evidence_count": 5,
  "screen_card_count": 1,
  "follow_up_count": 1,
  "confidence_present": true,
  "no_placeholder_utterance": true,
  "no_asr_text_leak": true,
  "v21_query_first_result_ms": 178,
  "tts_stop_observed": true,
  "failure_count": 0,
  "execution": {
    "provider_executed": false,
    "v21_executed": true,
    "hardware_executed": false,
    "gateway_runtime": "external_gateway"
  },
  "redaction": {
    "payloads_stored": false,
    "asr_text_stored": false,
    "evidence_body_stored": false,
    "full_url_stored": false,
    "local_path_stored": false,
    "prompt_stored": false,
    "provider_output_stored": false
  },
  "report_path": "a21-xiaozhi-professional-gateway.json"
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeProductReadinessV21AdapterSmokeReportFixture(t *testing.T) string {
	t.Helper()
	return writeProductReadinessV21AdapterSmokeReportFixtureFromData(t, productReadinessV21AdapterSmokeReportFixtureJSON())
}

func writeProductReadinessV21AdapterSmokeReportFixtureFromData(t *testing.T, data string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-v21-adapter-smoke-real.json")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func productReadinessV21AdapterSmokeReportFixtureJSON() string {
	return `{
  "schema_version": "a21.v21_adapter_smoke.v1",
  "generated_at_ms": 1780337400000,
  "adapter": "a21-v21-adapter",
  "protocol": "a21_v21_query",
  "status": "passed",
  "configured": true,
  "executed": true,
  "endpoint_host": "127.0.0.1:21121",
  "mode": "professional",
  "latency_profile": "fast_first",
  "answer_style": "voice_first_with_citations",
  "privacy_scope": "professional_only",
  "max_first_response_ms": 1200,
  "query_path": "/a21/v21/query",
  "health_path": "/healthz",
  "duration_ms": 1330.653,
  "confidence": 0.77,
  "evidence_count": 5,
  "speech_block_count": 1,
  "screen_card_count": 1,
  "follow_up_count": 1,
  "redaction_ok": true,
  "report_path": "a21-v21-adapter-smoke-real.json",
  "detail": "v21 adapter query smoke succeeded"
}`
}

func writeProductReadinessPhysicalStackChanReportFixture(t *testing.T, overrides map[string]any) string {
	t.Helper()
	report := map[string]any{
		"schema_version":    "a21.physical_stackchan_evidence.v1",
		"execution_mode":    "physical_stackchan",
		"trace_id":          "a21-trace-physical-stackchan-001",
		"session_id":        "a21-session-physical-stackchan-001",
		"device_id":         "stackchan-001",
		"fixture_path":      "physical-stackchan-fixture.json",
		"promotion_gate":    "candidate",
		"acceptance_status": "physical_review_required",
		"prd_accepted":      false,
		"report_path":       "a21-physical-stackchan-evidence-report.json",
		"execution": map[string]any{
			"provider_executed": false,
			"v21_executed":      false,
			"hardware_executed": true,
		},
		"stage_availability": map[string]any{
			"device.downlink.first_frame":          map[string]any{"available": true, "value_ms": 430, "source": "device_runtime_echo"},
			"device.playback.start":                map[string]any{"available": true, "value_ms": 520, "source": "device_runtime_echo"},
			"speech_end_to_first_audible_response": map[string]any{"available": true, "value_ms": 760, "source": "operator_or_instrument"},
			"barge_in.detected":                    map[string]any{"available": true, "value_ms": 50, "source": "gateway_trace"},
			"barge_in.stop":                        map[string]any{"available": true, "value_ms": 130, "source": "operator_or_instrument"},
			"barge_in.playback_stop_requested":     map[string]any{"available": true, "value_ms": 90, "source": "gateway_trace"},
			"barge_in.playback_stop_done":          map[string]any{"available": true, "value_ms": 130, "source": "device_runtime_echo"},
		},
		"canonical_metrics": map[string]any{
			"device_downlink_first_frame_ms":          map[string]any{"available": true, "value_ms": 430, "source": "device_runtime_echo"},
			"device_playback_start_ms":                map[string]any{"available": true, "value_ms": 520, "source": "device_runtime_echo"},
			"speech_end_to_first_audible_response_ms": map[string]any{"available": true, "value_ms": 760, "source": "operator_or_instrument"},
			"barge_in_detected_ms":                    map[string]any{"available": true, "value_ms": 50, "source": "gateway_trace"},
			"barge_in_stop_ms":                        map[string]any{"available": true, "value_ms": 130, "source": "operator_or_instrument"},
			"barge_in_playback_stop_requested_ms":     map[string]any{"available": true, "value_ms": 90, "source": "gateway_trace"},
			"barge_in_playback_stop_done_ms":          map[string]any{"available": true, "value_ms": 130, "source": "device_runtime_echo"},
		},
		"mic": map[string]any{
			"available":            true,
			"frames_captured":      320,
			"frames_delivered":     318,
			"rms":                  0.13,
			"delivery_ratio":       0.99375,
			"driver_error_count":   0,
			"queue_drop_count":     0,
			"nonzero_sample_count": 4096,
		},
		"observation": map[string]any{
			"available":               true,
			"physical_sound_observed": true,
			"operator_confirmed":      true,
			"method":                  "operator_and_instrument",
			"instrument":              "calibrated_audio_recorder",
			"observed_audible_ms":     760,
			"observed_stop_ms":        130,
		},
		"findings": []map[string]any{{
			"code":     "physical_review_required",
			"severity": "info",
			"message":  "physical metrics are present but still require explicit review before PRD acceptance",
		}},
		"redaction": map[string]any{
			"user_text_stored":             false,
			"instruction_text_stored":      false,
			"model_text_stored":            false,
			"audio_payload_stored":         false,
			"encoded_audio_payload_stored": false,
			"network_locator_stored":       false,
			"network_route_stored":         false,
			"filesystem_locator_stored":    false,
			"secret_material_stored":       false,
			"internal_thought_stored":      false,
		},
	}
	for key, value := range overrides {
		if value == nil {
			delete(report, key)
			continue
		}
		report[key] = value
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-physical-stackchan-evidence-report.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func productReadinessV21ProfessionalReportFixtureJSON() string {
	return `{
  "schema_version": "a21.v21_professional_readiness.v1",
  "status": "passed",
  "checking_ack_available": true,
  "checking_ack_ms": 4,
  "checking_ack_within_1200": true,
  "evidence_completed_ms": 42,
  "evidence_completed_after_ack": true,
  "evidence_available": true,
  "cards_available": true,
  "follow_ups_available": true,
  "evidence_count": 2,
  "evidence_types": ["meeting", "doc"],
  "card_count": 1,
  "follow_up_count": 2,
  "adapter_configured": true,
  "adapter_executed": false,
  "redaction_ok": true,
  "professional_acceptance_status": "host_mock_ready",
  "report_path": "a21-v21-professional-readiness-host.json"
}`
}

func productReadinessProviderSmokeReportFixtureJSON() string {
	return `{
  "schema_version": "a21.provider_smoke.v1",
  "generated_at_ms": 1780335600000,
  "provider": "deepseek",
  "family": "text_stream",
  "protocol": "openai_chat_completions",
  "status": "passed",
  "configured": true,
  "executed": true,
  "route_eligible": true,
  "stream": true,
  "repeat": 3,
  "http_status": 200,
  "duration_ms": 488.25,
  "attempts": [
    {
      "index": 1,
      "http_status": 200,
      "first_byte_ms": 112.5,
      "first_content_ms": 188.75,
      "total_duration_ms": 488.25,
      "content_delta_count": 2,
      "done": true
    },
    {
      "index": 2,
      "http_status": 200,
      "first_byte_ms": 118.5,
      "first_content_ms": 198.75,
      "total_duration_ms": 492.25,
      "content_delta_count": 2,
      "done": true
    },
    {
      "index": 3,
      "http_status": 200,
      "first_byte_ms": 120.5,
      "first_content_ms": 208.75,
      "total_duration_ms": 500.25,
      "content_delta_count": 2,
      "done": true
    }
  ],
  "timing_summary": {
    "repeat": 3,
    "first_byte_p50_ms": 118.5,
    "first_byte_p95_ms": 140.1,
    "first_byte_p99_ms": 142.1,
    "first_content_p50_ms": 198.75,
    "first_content_p95_ms": 220.2,
    "first_content_p99_ms": 222.2,
    "total_duration_p50_ms": 492.25,
    "total_duration_p95_ms": 510.3,
    "total_duration_p99_ms": 512.3
  },
  "trace_id": "a21-trace-provider-smoke-001",
  "trace_markers": [
    {"name": "provider_first_byte", "value_ms": 112.5},
    {"name": "provider_first_content", "value_ms": 188.75}
  ],
  "metrics": [
    {"name": "a21_provider_first_byte_ms", "value": 112.5},
    {"name": "a21_provider_first_content_ms", "value": 188.75}
  ],
  "network_mode": "direct",
  "endpoint_host": "api.deepseek.com",
  "api_key_env": "A21_LAB_DEEPSEEK_API_KEY",
  "model_env": "A21_DEEPSEEK_MODEL",
  "base_url_env": "A21_DEEPSEEK_BASE_URL",
  "detail": "provider smoke request succeeded",
  "report_path": "a21-provider-smoke-real.json"
}`
}

func productReadinessProviderSmokeReportFixtureForProvider(provider string) string {
	data := productReadinessProviderSmokeReportFixtureJSON()
	switch provider {
	case "local_ollama":
		replacements := map[string]string{
			`"provider": "deepseek"`:                        `"provider": "local_ollama"`,
			`"protocol": "openai_chat_completions"`:         `"protocol": "ollama_chat"`,
			`"endpoint_host": "api.deepseek.com"`:           `"endpoint_host": "127.0.0.1:11434"`,
			`"api_key_env": "A21_LAB_DEEPSEEK_API_KEY"`:     `"api_key_env": ""`,
			`"model_env": "A21_DEEPSEEK_MODEL"`:             `"model_env": "A21_LOCAL_OLLAMA_MODEL"`,
			`"base_url_env": "A21_DEEPSEEK_BASE_URL"`:       `"base_url_env": "A21_LOCAL_OLLAMA_BASE_URL"`,
			`"report_path": "a21-provider-smoke-real.json"`: `"report_path": "a21-provider-smoke-local-ollama.json"`,
		}
		for old, newValue := range replacements {
			data = strings.ReplaceAll(data, old, newValue)
		}
	}
	return data
}

func productReadinessRealtimeFixtureReportFixtureJSON() string {
	return `{
  "schema_version": "a21.provider_smoke.v1",
  "generated_at_ms": 1780335605000,
  "provider": "doubao_tts_realtime",
  "family": "voice_hybrid",
  "protocol": "websocket_realtime_fixture",
  "status": "passed",
  "configured": true,
  "executed": true,
  "route_eligible": false,
  "duration_ms": 3.25,
  "network_mode": "direct",
  "endpoint_host": "ai-gateway.vei.volces.com",
  "api_key_env": "A21_DOUBAO_API_KEY",
  "model_env": "A21_DOUBAO_TTS_MODEL",
  "base_url_env": "A21_DOUBAO_TTS_REALTIME_URL",
  "detail": "offline realtime fixture passed; no provider network call performed",
  "report_path": "a21-provider-realtime-fixture-real.json"
}`
}

func productReadinessXiaozhiHostReportFixtureJSON() string {
	return `{
  "schema_version": "a21.xiaozhi_voice_bench.v1",
  "execution_mode": "host_loopback",
  "baseline_scope": "host_only",
  "device_id": "stackchan-virtual-a21-bench-001",
  "repeat": 3,
  "acceptance_status": "candidate_host_only",
  "prd_accepted": false,
  "summary": {
    "answer_first_audio_total_p50_ms": 360,
    "answer_first_audio_total_p95_ms": 386,
    "barge_in_stop_p50_ms": 0,
    "barge_in_stop_p95_ms": 0
  },
  "counts": {
    "answer_turn_count": 3,
    "barge_in_turn_count": 3,
    "failure_count": 0
  },
  "execution": {
    "provider_executed": false,
    "v21_executed": false,
    "hardware_executed": false,
    "voice_pipeline_observed": true,
    "voice_pipeline_execution_mode": "host_local",
    "asr_profile": "sherpa_onnx",
    "asr_profile_env": "A21_ASR_LOCAL_PROFILE",
    "llm_profile": "ollama_local",
    "llm_profile_env": "A21_TEXT_STREAM_PROFILE",
    "tts_profile": "sherpa_onnx_tts",
    "tts_profile_env": "A21_TTS_FAST_PROFILE",
    "host_local_asr_executed": true,
    "host_local_text_executed": true,
    "host_local_tts_executed": true
  },
  "redaction": {
    "payloads_stored": false,
    "credential_values_stored": false,
    "full_urls_stored": false,
    "local_paths_stored": false
  }
}`
}

func TestRunPromotionReadinessBlocksExternalPromotionWithoutTarget(t *testing.T) {
	dir := t.TempDir()
	writePromotionReadinessGitScript(t, dir, promotionGitScriptOptions{})
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"promotion-readiness"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.promotion_readiness.v1"`,
		`"branch": "codex/a21-integration-governance-slices"`,
		`"review_ready": true`,
		`"external_promotion_ready": false`,
		`"promotion_remote_missing"`,
		`"promotion_target_branch_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunPromotionReadinessBlocksMissingTopicAncestor(t *testing.T) {
	dir := t.TempDir()
	writePromotionReadinessGitScript(t, dir, promotionGitScriptOptions{
		remoteNames:            "origin\n",
		targetBranchConfigured: true,
		missingAncestor:        "6b7fdf0",
	})
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"promotion-readiness", "--target-remote", "origin", "--target-branch", "codex/a21-mainline-current"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"review_ready": false`,
		`"external_promotion_ready": false`,
		`"promotion_topic_not_ancestor"`,
		`"branch": "codex/a21-provider-spine-deepseek-textstream"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunGateIntegrationWrapsPromotionReadiness(t *testing.T) {
	dir := t.TempDir()
	writePromotionReadinessGitScript(t, dir, promotionGitScriptOptions{})
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"gate", "--scope", "integration"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.gate.v1"`,
		`"scope": "integration"`,
		`"promotion_readiness"`,
		`"promotion_remote_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunGateHardwareRequiresCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"gate", "--scope", "hardware"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--command requires a value") {
		t.Fatalf("stderr missing command error: %s", stderr.String())
	}
}

func TestRunGateHardwareWrapsControlGuard(t *testing.T) {
	allowA21ControlGuardForTest(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"gate", "--scope", "hardware", "--command", "provider-smoke --execute"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.gate.v1"`,
		`"scope": "hardware"`,
		`"control_guard"`,
		`"command": "provider-smoke --execute"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanAcceptRequiresCheck(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"stackchan-accept", "--device-id", "stackchan-001"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--check requires a value") {
		t.Fatalf("stderr missing check error: %s", stderr.String())
	}
}

func TestRunStackChanAcceptDispatchesCheckHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"stackchan-accept", "--check", "touch", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "a21 stackchan-accept --check touch") {
		t.Fatalf("stdout missing touch help: %s", stdout.String())
	}
}

func TestRunDeprecatedStackChanAcceptAliasStillDispatches(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"stackchan-touch-acceptance", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "a21 stackchan-accept --check touch") {
		t.Fatalf("stdout missing touch help: %s", stdout.String())
	}
}

func TestRunNamespaceAuditReadsTrackedFiles(t *testing.T) {
	dir := t.TempDir()
	writeNamespaceAuditGitScript(t, dir, "cmd/a21/main.go\ninternal/v21adapter/client.go\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"namespace-audit"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"ok": true`,
		`"files_scanned": 2`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunNamespaceAuditRejectsLegacyPathWithoutEchoingIt(t *testing.T) {
	dir := t.TempDir()
	writeNamespaceAuditGitScript(t, dir, "cmd/a21/main.go\napps/x21-gateway/main.go\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"namespace-audit"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"namespace_legacy_path"`) {
		t.Fatalf("stdout missing namespace finding: %s", stdout.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21-gateway") || strings.Contains(strings.ToLower(stderr.String()), "x21-gateway") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestGatewayServerFromEnvDefaultsToMockDespiteSelectedPrimary(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_PROVIDER_PRIMARY=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/providers/voice/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"provider":"a21-mock-voice"`) {
		t.Fatalf("health = %s, want mock provider", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "sk-a21-secret") || strings.Contains(rec.Body.String(), "doubao-tts") {
		t.Fatalf("health leaked provider config: %s", rec.Body.String())
	}
}

func TestGatewayServerOptionsFromEnvWiresXiaozhiVoicePipelineAdapters(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_TEXT_STREAM_PROFILE=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=sk-a21-secret",
		"A21_TTS_FAST_PROFILE=macos_say",
	})
	if options.XiaozhiVoicePipelineAdapters == nil {
		t.Fatal("xiaozhi voice pipeline adapters not configured")
	}
	adapters := *options.XiaozhiVoicePipelineAdapters
	if adapters.ExecutionMode != "host_local" {
		t.Fatalf("execution mode = %q, want host_local", adapters.ExecutionMode)
	}
	if adapters.ASR.Name() != "sherpa_onnx" || adapters.TextStream.Name() != "deepseek" || adapters.TTS.Name() != "macos_say" {
		t.Fatalf("adapters = %s/%s/%s, want sherpa_onnx/deepseek/macos_say", adapters.ASR.Name(), adapters.TextStream.Name(), adapters.TTS.Name())
	}
	selectionBytes, err := json.Marshal(adapters.Selection)
	if err != nil {
		t.Fatal(err)
	}
	selectionPayload := string(selectionBytes)
	if strings.Contains(selectionPayload, "sk-a21-secret") {
		t.Fatalf("selection leaked secret: %s", selectionPayload)
	}
}

func TestGatewayServerOptionsFromEnvProductChainModeDefaultsHostLocalAdapters(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_XIAOZHI_PRODUCT_CHAIN=host_local",
		"A21_LOCAL_OLLAMA_BASE_URL=http://127.0.0.1:11434",
		"A21_LOCAL_OLLAMA_MODEL=qwen2.5",
	})
	if options.XiaozhiVoicePipelineAdapters == nil {
		t.Fatal("xiaozhi voice pipeline adapters not configured")
	}
	adapters := *options.XiaozhiVoicePipelineAdapters
	if adapters.ExecutionMode != "host_local" {
		t.Fatalf("execution mode = %q, want host_local", adapters.ExecutionMode)
	}
	if adapters.ASR.Name() != "sherpa_onnx" || adapters.TextStream.Name() != "local_ollama" || adapters.TTS.Name() != "sherpa_onnx_tts" {
		t.Fatalf("adapters = %s/%s/%s, want sherpa_onnx/local_ollama/sherpa_onnx_tts", adapters.ASR.Name(), adapters.TextStream.Name(), adapters.TTS.Name())
	}
	selectionBytes, err := json.Marshal(adapters.Selection)
	if err != nil {
		t.Fatal(err)
	}
	selectionPayload := string(selectionBytes)
	for _, forbidden := range []string{"http://", "127.0.0.1", "qwen2.5"} {
		if strings.Contains(selectionPayload, forbidden) {
			t.Fatalf("selection leaked config detail %q: %s", forbidden, selectionPayload)
		}
	}
}

func TestGatewayCLIOptionsApplyProductChainEnvOverrides(t *testing.T) {
	env := gatewayEnvWithCLIOptions([]string{
		"A21_LOCAL_OLLAMA_MODEL=old-model",
	}, gatewayCLIOptions{
		ProductChain:       "host_local",
		LocalOllamaBaseURL: "http://127.0.0.1:11434",
		LocalOllamaModel:   "qwen2.5:0.5b",
		VoiceTextMaxTokens: "32",
	})
	if got := appEnvValue(env, "A21_XIAOZHI_PRODUCT_CHAIN"); got != "host_local" {
		t.Fatalf("A21_XIAOZHI_PRODUCT_CHAIN = %q, want host_local", got)
	}
	if got := appEnvValue(env, "A21_LOCAL_OLLAMA_BASE_URL"); got != "http://127.0.0.1:11434" {
		t.Fatalf("A21_LOCAL_OLLAMA_BASE_URL = %q, want loopback ollama", got)
	}
	if got := appEnvValue(env, "A21_LOCAL_OLLAMA_MODEL"); got != "qwen2.5:0.5b" {
		t.Fatalf("A21_LOCAL_OLLAMA_MODEL = %q, want override", got)
	}
	if got := appEnvValue(env, "A21_VOICE_TEXT_MAX_TOKENS"); got != "32" {
		t.Fatalf("A21_VOICE_TEXT_MAX_TOKENS = %q, want 32", got)
	}
}

func TestRunGatewayHelpIncludesProductChainWarmupFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"gateway", "--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "--warm-product-chain") {
		t.Fatalf("help missing --warm-product-chain: %s", stdout.String())
	}
}

func TestGatewayServerOptionsFromEnvWiresStockProfessionalRoute(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_XIAOZHI_STOCK_PROFESSIONAL_ROUTE=professional",
	})
	if !options.XiaozhiStockProfessional {
		t.Fatal("xiaozhi stock professional route not configured")
	}
}

func TestGatewayServerOptionsFromEnvWiresSileroVADConfig(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_VAD_PREFERENCE=silero",
		"A21_SILERO_VAD_COMMAND=/tmp/a21-silero-runner",
		"A21_SILERO_VAD_MODEL=/tmp/a21-silero.onnx",
		"A21_SILERO_VAD_TIMEOUT_MS=90",
	})
	if options.AudioIngressConfig.VADPreference != audio.VADDetectorPreferenceSilero {
		t.Fatalf("VAD preference = %q, want silero", options.AudioIngressConfig.VADPreference)
	}
	if options.AudioIngressConfig.SileroRunner == nil {
		t.Fatal("Silero runner was not configured from env")
	}
	if options.AudioIngressConfig.VADTimeout != 90*time.Millisecond {
		t.Fatalf("VAD timeout = %s, want 90ms", options.AudioIngressConfig.VADTimeout)
	}
}

func TestGatewayServerFromEnvUsesSelectedProviderOnlyWhenExplicit(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_GATEWAY_VOICE_PROVIDER=selected",
		"A21_PROVIDER_PRIMARY=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/providers/voice/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"provider":"a21-doubao-realtime-tts"`) {
		t.Fatalf("health = %s, want selected Doubao provider", rec.Body.String())
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie", "Authorization", "Bearer"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestGatewayServerFromEnvUsesSelectedDoubaoRealtimeAsDegradedBoundary(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_GATEWAY_VOICE_PROVIDER=selected",
		"A21_PROVIDER_PRIMARY=doubao_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_APP_ID=app-a21-secret",
		"A21_DOUBAO_RESOURCE_ID=resource-a21-secret",
		"A21_DOUBAO_REALTIME_MODEL=doubao-s2s",
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/providers/voice/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{
		`"provider":"a21-doubao-realtime-voice"`,
		`"status":"degraded"`,
		`"configured":true`,
		`"detail":"execution disabled pending verified Doubao realtime speech-to-speech adapter"`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("health missing %q: %s", want, rec.Body.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestGatewayServerSelectedRealtimeProviderKeepsTextTurnGuarded(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_GATEWAY_VOICE_PROVIDER=selected",
		"A21_PROVIDER_PRIMARY=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"普通文本轮不应该误拨 provider","mode":"workmate"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "StartRealtimeTTSSession") {
		t.Fatalf("response missing guarded realtime-session guidance: %s", rec.Body.String())
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie", "Authorization", "Bearer"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestGatewayServerSelectedDoubaoRealtimeRejectsRealtimeSessionWithoutSecrets(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_GATEWAY_VOICE_PROVIDER=selected",
		"A21_PROVIDER_PRIMARY=doubao_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_APP_ID=app-a21-secret",
		"A21_DOUBAO_RESOURCE_ID=resource-a21-secret",
		"A21_DOUBAO_REALTIME_MODEL=doubao-s2s",
	})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"先不要真的连接豆包","mode":"workmate","trace_id":"a21-trace-doubao-guard","session_id":"a21-session-doubao-guard"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502: %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{
		`"provider":"a21-doubao-realtime-voice"`,
		`"status":"error"`,
		`"trace_id":"a21-trace-doubao-guard"`,
		`"text":"Doubao realtime speech-to-speech execution is disabled until the A21 adapter has verified official API shape, credentialed smoke, cancellation, and latency behavior"`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("response missing %q: %s", want, rec.Body.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestGatewayServerFromEnvUsesConfiguredV21AdapterForProfessionalMode(t *testing.T) {
	var sawProfessionalRequest bool
	adapter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a21/v21/query" {
			t.Fatalf("path = %q, want /a21/v21/query", r.URL.Path)
		}
		var request struct {
			Mode         string `json:"mode"`
			Utterance    string `json:"utterance"`
			PrivacyScope string `json:"privacy_scope"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		sawProfessionalRequest = request.Mode == "professional" &&
			request.Utterance == "查一下真实 adapter" &&
			request.PrivacyScope == "professional_only"
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"trace_id":"a21-trace-env-v21",
			"fast_answer":"adapter answer marker",
			"confidence":0.91,
			"evidence":[{"title":"adapter source","type":"doc","source_id":"v21-adapter-source","summary":"adapter summary"}],
			"speech_blocks":["adapter speech"],
			"screen_cards":[{"label":"adapter","text":"adapter card"}],
			"follow_ups":["adapter followup"]
		}`))
	}))
	defer adapter.Close()
	server := newGatewayServerFromEnv([]string{"A21_V21_ADAPTER_URL=" + adapter.URL})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下真实 adapter","mode":"professional","trace_id":"a21-trace-env-v21","session_id":"a21-session-env-v21"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !sawProfessionalRequest {
		t.Fatal("gateway did not call configured A21 V21 adapter")
	}
	if !strings.Contains(rec.Body.String(), "adapter answer marker") {
		t.Fatalf("response did not use adapter answer: %s", rec.Body.String())
	}
}

func TestGatewayServerFromEnvInvalidV21AdapterURLFailsProfessionalModeWithoutMockEvidence(t *testing.T) {
	server := newGatewayServerFromEnv([]string{"A21_V21_ADAPTER_URL=http://127.0.0.1:18080"})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下真实 adapter","mode":"professional","trace_id":"a21-trace-env-v21-invalid","session_id":"a21-session-env-v21-invalid"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "V21 现在没接上") {
		t.Fatalf("response should fail honestly instead of using mock evidence: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "历史讨论主要集中") {
		t.Fatalf("response used mock V21 evidence despite invalid configured adapter: %s", rec.Body.String())
	}
}

func TestRunSerialListEmitsSerialDevices(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{
			{Path: "/dev/cu.usbmodemA21", USBModem: true, Usage: firmwarecheck.PortUsage{Exists: true}},
		}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"serial-list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{`"serial_devices"`, "/dev/cu.usbmodemA21", `"usb_modem": true`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunDoctorEmitsReport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	if !strings.Contains(stdout.String(), `"result"`) {
		t.Fatalf("stdout missing result report: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"fingerprint"`) {
		t.Fatalf("stdout missing fingerprint report: %q", stdout.String())
	}
}

func TestRunDoctorWritesReportFile(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-doctor-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("report files = %d, want 1", len(matches))
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"fingerprint"`) {
		t.Fatalf("report file missing fingerprint: %s", data)
	}
}

func TestRunDoctorIncludesFirmwareSection(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	if !strings.Contains(stdout.String(), `"firmware"`) {
		t.Fatalf("stdout missing firmware section: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"serial_devices"`) {
		t.Fatalf("stdout missing serial devices: %q", stdout.String())
	}
}

func TestRunDoctorIncludesWakeWordPendingFirmwareBuildWithoutPathLeaks(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "a21-wake-word.json")
	if err := os.WriteFile(configPath, []byte(`{
		"mode":"custom_multinet",
		"desired_phrase":"小阿二一",
		"desired_pinyin":"xiao a er yi",
		"threshold":35
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_WAKE_WORD_CONFIG_PATH", configPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"doctor", "--output-dir", t.TempDir()}, &stdout, &stderr)

	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"wake_word"`,
		`"schema_version": "a21.gateway.wake_word.v1"`,
		`"mode": "custom_multinet"`,
		`"active_phrase": "你好小智"`,
		`"desired_phrase": "小阿二一"`,
		`"desired_pinyin": "xiao a er yi"`,
		`"threshold": 35`,
		`"runtime_status": "pending_firmware_build"`,
		`"firmware_build_required": true`,
		`"code": "a21_wake_word_firmware_build_required"`,
		`"wake_word_firmware_build_required"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{configPath, dir, "A21_WAKE_WORD_CONFIG_PATH", "secret", "token", "http://", "https://"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor wake-word report leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorIncludesProxyPolicy(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	for _, want := range []string{`"proxy"`, `"global_proxy_configured"`, `"direct_connect_ok"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunDoctorIncludesVoiceProviderHealth(t *testing.T) {
	originalProbe := probeVoiceProviderHealth
	probeVoiceProviderHealth = func(ctx context.Context) (providers.VoiceProviderHealth, error) {
		return providers.VoiceProviderHealth{
			Provider:   "a21-test-voice",
			Status:     providers.VoiceProviderHealthy,
			Configured: true,
			Realtime:   true,
		}, nil
	}
	defer func() {
		probeVoiceProviderHealth = originalProbe
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	for _, want := range []string{`"voice"`, `"provider": "a21-test-voice"`, `"status": "healthy"`, `"healthy": true`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunDoctorIncludesProviderNetworkPolicy(t *testing.T) {
	t.Setenv("A21_PROVIDER_PROXY_URL", "http://provider-secret@127.0.0.1:7891")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	for _, want := range []string{`"network"`, `"mode": "explicit_proxy"`, `"proxy_configured": true`, `"provider_proxy_env"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"provider-secret", "7891"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked provider proxy value %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorIncludesProviderCatalogWithoutSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-secret")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	for _, want := range []string{`"providers"`, `"primary": "deepseek"`, `"name": "deepseek"`, `"family": "text_stream"`, `"configured": true`, `"A21_LAB_DEEPSEEK_API_KEY"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-chat"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked provider value %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorVoiceHealthFollowsSelectedProviderWithoutSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "zh_female_kailangjiejie_moon_bigtts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", t.TempDir()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "a21-doubao-realtime-tts"`,
		`"gateway_provider": "a21-mock-voice"`,
		`"status": "healthy"`,
		`"configured": true`,
		`"primary": "doubao_tts_realtime"`,
		`"name": "doubao_tts_realtime"`,
		`"family": "voice_hybrid"`,
		`"status": "unsupported"`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), `"provider_unknown"`) {
		t.Fatalf("doctor reported selected realtime TTS profile as unknown: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), `"provider": "a21-mock-voice"`) {
		t.Fatalf("doctor voice health still reports mock provider: %s", stdout.String())
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie_moon_bigtts", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorVoiceHealthReportsDoubaoRealtimeDegradedWithoutSecrets(t *testing.T) {
	setA21DirectProxyBypassForTest(t)
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_APP_ID", "app-a21-secret")
	t.Setenv("A21_DOUBAO_RESOURCE_ID", "resource-a21-secret")
	t.Setenv("A21_DOUBAO_REALTIME_MODEL", "doubao-s2s")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", t.TempDir()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "a21-doubao-realtime-voice"`,
		`"gateway_provider": "a21-mock-voice"`,
		`"status": "degraded"`,
		`"healthy": false`,
		`"configured": true`,
		`"detail": "execution disabled pending verified Doubao realtime speech-to-speech adapter"`,
		`"primary": "doubao_realtime"`,
		`"name": "doubao_realtime"`,
		`"family": "voice_realtime"`,
		`"status": "unsupported"`,
		`"status": "ready"`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), `"provider_unknown"`) {
		t.Fatalf("doctor reported selected realtime profile as unknown: %s", stdout.String())
	}
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorReportsExplicitGatewayVoiceProviderRuntime(t *testing.T) {
	t.Setenv("A21_GATEWAY_VOICE_PROVIDER", "selected")
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "zh_female_kailangjiejie_moon_bigtts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", t.TempDir()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"gateway_provider": "a21-doubao-realtime-tts"`) {
		t.Fatalf("stdout missing explicit gateway provider: %s", stdout.String())
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorReportsExplicitGatewayDoubaoRealtimeRuntimeAsDegraded(t *testing.T) {
	setA21DirectProxyBypassForTest(t)
	t.Setenv("A21_GATEWAY_VOICE_PROVIDER", "selected")
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_APP_ID", "app-a21-secret")
	t.Setenv("A21_DOUBAO_RESOURCE_ID", "resource-a21-secret")
	t.Setenv("A21_DOUBAO_REALTIME_MODEL", "doubao-s2s")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", t.TempDir()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "a21-doubao-realtime-voice"`,
		`"gateway_provider": "a21-doubao-realtime-voice"`,
		`"status": "degraded"`,
		`"detail": "execution disabled pending verified Doubao realtime speech-to-speech adapter"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorIncludesRealtimePlanWithoutSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "openai_realtime")
	t.Setenv("A21_OPENAI_API_KEY", "sk-a21-secret")
	t.Setenv("A21_OPENAI_REALTIME_MODEL", "gpt-realtime-2")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"realtime_plan"`,
		`"provider": "openai_realtime"`,
		`"protocol": "websocket_realtime"`,
		`"status": "ready"`,
		`"endpoint_host": "api.openai.com"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "gpt-realtime-2", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorBlocksLegacyProviderPrimaryWithoutEchoingValue(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "x21_voice")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"provider_legacy_identity"`) {
		t.Fatalf("stdout missing provider legacy finding: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "x21_voice") {
		t.Fatalf("stdout leaked legacy provider value: %s", stdout.String())
	}
}

func TestRunProviderSmokeDryRunDoesNotLeakSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-secret")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-smoke", "--provider", "deepseek"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{`"provider": "deepseek"`, `"status": "ready"`, `"executed": false`, `"api_key_env": "A21_LAB_DEEPSEEK_API_KEY"`} {
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

func TestRunProviderSmokeReportsHotPlugProfileRouteEligibility(t *testing.T) {
	profilePath := filepath.Join(t.TempDir(), "a21-provider-profiles.json")
	if err := os.WriteFile(profilePath, []byte(`{
		"name": "a21_ws7a_cli_vendor",
		"label": "A21 WS7A CLI Vendor",
		"family": "text_stream",
		"protocol": "openai_chat_completions",
		"capabilities": ["llm", "text_stream"],
		"api_key_env": "A21_WS7A_CLI_VENDOR_API_KEY",
		"model_env": "A21_WS7A_CLI_VENDOR_MODEL",
		"base_url_env": "A21_WS7A_CLI_VENDOR_BASE_URL",
		"endpoint_path": "/chat/completions",
		"route_eligible": true
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_PROVIDER_PROFILES_PATH", profilePath)
	t.Setenv("A21_PROVIDER_PRIMARY", "a21_ws7a_cli_vendor")
	t.Setenv("A21_WS7A_CLI_VENDOR_API_KEY", "sk-a21-cli-secret")
	t.Setenv("A21_WS7A_CLI_VENDOR_MODEL", "cli-hidden-model")
	t.Setenv("A21_WS7A_CLI_VENDOR_BASE_URL", "http://127.0.0.1:9/v1")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-smoke", "--provider", "a21_ws7a_cli_vendor", "--stream", "--repeat", "2"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "a21_ws7a_cli_vendor"`,
		`"status": "ready"`,
		`"configured": true`,
		`"executed": false`,
		`"stream": true`,
		`"repeat": 2`,
		`"route_eligible": true`,
		`"api_key_env": "A21_WS7A_CLI_VENDOR_API_KEY"`,
		`"model_env": "A21_WS7A_CLI_VENDOR_MODEL"`,
		`"base_url_env": "A21_WS7A_CLI_VENDOR_BASE_URL"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{
		profilePath,
		filepath.Dir(profilePath),
		"sk-a21-cli-secret",
		"cli-hidden-model",
		"http://127.0.0.1:9/v1",
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("provider smoke CLI leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

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

	code := Run([]string{"provider-latency-bench", "--provider", "deepseek", "--iterations", "3", "--output-dir", dir}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--iterations", "2"}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "fixture", "--iterations", "2"}, &stdout, &stderr)

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
	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--fixture", "/tmp/a21/private/audio-fixture.wav", "--iterations", "1"}, &stdout, &stderr)
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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--fixture", fixture, "--iterations", "1"}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--fixture", fixture, "--iterations", "1"}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--fixture", fixture, "--iterations", "1"}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--fixture", fixture, "--iterations", "1"}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--fixture", fixture, "--iterations", "1"}, &stdout, &stderr)

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
	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--iterations", "1"}, &stdout, &stderr)
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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

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

	code := Run([]string{"provider-latency-bench", "--provider", "mock", "--mode", "host_loopback", "--fixture", fixture}, &stdout, &stderr)

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

func TestRunXiaozhiVoiceBenchReportsHostOnlyCandidateEvidence(t *testing.T) {
	gatewayServer := newGatewayServerFromEnv(nil)
	httpServer := httptest.NewServer(gatewayServer.Handler())
	t.Cleanup(httpServer.Close)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
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

	code := Run([]string{
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

func TestRunXiaozhiVoiceBenchSupportsRedactedInputWAVFixture(t *testing.T) {
	gatewayServer := newGatewayServerFromEnv(nil)
	httpServer := httptest.NewServer(gatewayServer.Handler())
	t.Cleanup(httpServer.Close)
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "a21-input.wav")
	writeAppTestWAV(t, wavPath, 16000, bytes.Repeat([]byte{0x70, 0x17}, 960*2))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
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

	code := Run([]string{
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

	code := Run([]string{
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

	code := Run([]string{
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
	code := Run([]string{"provider-latency-bench", "--mode", "host_baseline"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "--mode must be mock, fixture, or host_loopback") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunProviderLatencyBenchRejectsExecute(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"provider-latency-bench", "--execute"}, &bytes.Buffer{}, &stderr)
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

func TestRunLocalTTSSmokeRejectsLegacyReportDir(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-tts-smoke", "--output-dir", filepath.Join(t.TempDir(), "v21-reports")}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
}

func TestRunLocalASRSmokeWritesRedactedReport(t *testing.T) {
	original := runSherpaONNXASRSmoke
	t.Cleanup(func() { runSherpaONNXASRSmoke = original })
	runSherpaONNXASRSmoke = func(ctx context.Context, options audio.LocalASROptions) (audio.LocalASRReport, error) {
		if options.ModelDir != "/tmp/a21-asr-model" || options.Family != "paraformer" || options.WAVPath != "/tmp/a21-asr-model/test_wavs/private.wav" {
			t.Fatalf("asr options = %+v", options)
		}
		return audio.LocalASRReport{
			SchemaVersion:    "a21.audio.local_asr.v1",
			GeneratedAtMS:    time.Now().UnixMilli(),
			Status:           "passed",
			Provider:         "sherpa_onnx",
			Engine:           "paraformer",
			ModelDir:         "a21-asr-model",
			WAVName:          "private.wav",
			InputDurationMS:  1200,
			DecodeDurationMS: 90,
			RealTimeFactor:   0.075,
			TextChars:        8,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-asr-smoke", "--engine", "sherpa_onnx", "--family", "paraformer", "--model-dir", "/tmp/a21-asr-model", "--wav", "/tmp/a21-asr-model/test_wavs/private.wav", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.audio.local_asr.v1"`,
		`"status": "passed"`,
		`"provider": "sherpa_onnx"`,
		`"engine": "paraformer"`,
		`"model_dir": "a21-asr-model"`,
		`"wav_name": "private.wav"`,
		`"text_chars": 8`,
		`"report_path"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-asr-smoke-*.json"))
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
	for _, forbidden := range []string{"private transcript", "/tmp/a21-asr-model", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("local ASR smoke leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalASRSmokeRejectsLegacyReportDir(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-asr-smoke", "--output-dir", filepath.Join(t.TempDir(), "v21-reports")}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
}

func TestWriteLocalASRSmokeReportUsesUniqueNamesForRapidWrites(t *testing.T) {
	dir := t.TempDir()
	report := audio.LocalASRReport{
		SchemaVersion:    "a21.audio.local_asr.v1",
		GeneratedAtMS:    time.Now().UnixMilli(),
		Status:           "passed",
		Provider:         "sherpa_onnx",
		Engine:           "paraformer",
		TranscriptPolicy: "transcript_not_recorded",
	}

	first, err := writeLocalASRSmokeReport(dir, report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := writeLocalASRSmokeReport(dir, report)
	if err != nil {
		t.Fatal(err)
	}

	if first == second {
		t.Fatalf("rapid report writes used the same path: %s", first)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-asr-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("reports = %d, want 2: %v", len(matches), matches)
	}
}

func TestRunLocalVoiceLoopbackHelpListsLocalOllama(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{"mock_text_stream", "deepseek", "local_ollama"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help missing %q: %s", want, stdout.String())
		}
	}
}

func TestFastCompanionVoicePreviewKeepsFirstSpeechShort(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "sentence boundary",
			in:   "先稳住。后面这句不要进首段语音。",
			want: "先稳住",
		},
		{
			name: "rune cap",
			in:   "这是一个很长很长的中文回复内容",
			want: "这是一个很长很长的中文回",
		},
		{
			name: "ascii tail",
			in:   "这是来自本地 Ollama 的回复",
			want: "这是来自本地",
		},
		{
			name: "ascii only",
			in:   "A21 loopback response",
			want: "A21 loopback response",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fastCompanionVoicePreview(tc.in); got != tc.want {
				t.Fatalf("preview = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFastCompanionTextStreamPromptUsesPersonalityRuntimeAssets(t *testing.T) {
	prompt := fastCompanionTextStreamPrompt("a21 mock transcript")

	for _, want := range []string{
		"Core Identity",
		"Tone Rules",
		"Workmate Mode",
		"不超过12个字",
		"a21 mock transcript",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
	for _, forbidden := range []string{
		"Professional Mode",
		"Companion Mode",
		"Post-Meeting Playbook",
		"Failure Overlay",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt included unselected asset %q:\n%s", forbidden, prompt)
		}
	}
}

func TestFastCompanionTextStreamPromptIncludesBoundedMemoryHints(t *testing.T) {
	t.Setenv("A21_MEMORY_USER_PREFERENCES", "偏好短句")
	t.Setenv("A21_MEMORY_SESSION_NOTES", "当前任务是座舱 PRD 评审")

	prompt := fastCompanionTextStreamPrompt("a21 mock transcript")

	for _, want := range []string{
		"Workmate Mode",
		"Memory Hints",
		"user_preference:user_preference_1",
		"偏好短句",
		"session_memory:session_memory_1",
		"当前任务是座舱 PRD 评审",
		"a21 mock transcript",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestRunLocalVoiceLoopbackWritesRedactedReport(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-test.wav")
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
			DurationMS:      10,
			TTSFirstAudioMS: 10,
			AudioQuality: &audio.PCMQualityReport{
				Status:       "passed",
				Codec:        "pcm_s16le",
				SampleRateHz: 16000,
				Channels:     1,
				PeakAbs:      1600,
				RMSDBFS:      -28.2,
			},
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text", "真实输入不要进报告", "--repeat", "2", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.audio.local_voice_loopback.v1"`,
		`"status": "passed"`,
		`"vad_status": "mock_only"`,
		`"asr_provider": "mock_asr"`,
		`"text_stream_provider": "mock_text_stream"`,
		`"tts_provider": "macos_say"`,
		`"local_ack_audio_quality"`,
		`"tts_audio_quality"`,
		`"repeat": 2`,
		`"tts_first_audio_p50_ms"`,
		`"first_audio_total_p95_ms"`,
		`"barge_in_status": "benchmarked"`,
		`"report_path"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
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
	for _, forbidden := range []string{"真实输入不要进报告", "A21 loopback response", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackSupportsVoiceClonePersonaReport(t *testing.T) {
	original := synthesizeVoiceCloneCLI
	t.Cleanup(func() { synthesizeVoiceCloneCLI = original })
	refAudio := filepath.Join(t.TempDir(), "a21-persona-reference.wav")
	if err := os.WriteFile(refAudio, []byte("RIFF-a21-reference"), 0o644); err != nil {
		t.Fatal(err)
	}
	var ttsInputs []string
	synthesizeVoiceCloneCLI = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInputs = append(ttsInputs, options.Text)
		if options.VoiceCloneCommand != "/a21/bin/a21-index-tts2-wrapper" ||
			options.VoiceCloneModel != "Index-TTS2" ||
			options.VoiceCloneReferenceAudioPath != refAudio ||
			options.VoiceCloneReferenceText != "参考文本不能进报告" ||
			options.VoiceClonePersona != "A21 Workmate" ||
			options.VoiceCloneStyle != "Warm-Pro" {
			t.Fatalf("clone options = %+v", options)
		}
		outputPath := filepath.Join(options.OutputDir, fmt.Sprintf("a21-voice-clone-loopback-%d.wav", len(ttsInputs)))
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
			DurationMS:      14,
			TTSFirstAudioMS: 14,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"local-voice-loopback",
		"--engine", "voice_clone_cli",
		"--text", "用户原文不要进报告",
		"--clone-command", "/a21/bin/a21-index-tts2-wrapper",
		"--clone-model", "Index-TTS2",
		"--clone-ref-audio", refAudio,
		"--clone-ref-text", "参考文本不能进报告",
		"--voice-persona", "A21 Workmate",
		"--voice-style", "Warm-Pro",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if len(ttsInputs) != 2 || ttsInputs[0] != "嗯，我在。" || ttsInputs[1] != "A21 loopback response" {
		t.Fatalf("tts inputs = %#v, want local ack and answer preview", ttsInputs)
	}
	for _, want := range []string{
		`"tts_provider": "voice_clone_cli"`,
		`"tts_model": "index_tts2"`,
		`"tts_voice": "a21_workmate"`,
		`"tts_voice_persona": "a21_workmate"`,
		`"tts_style_profile": "warm_pro"`,
		`"tts_reference_audio": "a21-persona-reference.wav"`,
		`"local_ack_tts_provider": "voice_clone_cli"`,
		`"local_ack_tts_model": "index_tts2"`,
		`"local_ack_tts_voice_persona": "a21_workmate"`,
		`"local_ack_tts_style_profile": "warm_pro"`,
		`"local_ack_tts_reference_audio": "a21-persona-reference.wav"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("reports = %v, %v", matches, err)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"用户原文不要进报告", "A21 loopback response", "嗯，我在", "参考文本不能进报告", refAudio, "Authorization", "Bearer", "raw_audio", "data_base64"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("voice clone loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackCanUseDeepSeekTextStreamWithoutLeakingContent(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	var ttsInput string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInput = options.Text
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-deepseek-test.wav")
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
			DurationMS:      12,
			TTSFirstAudioMS: 12,
		}, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			MaxTokens int `json:"max_tokens"`
			Messages  []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.MaxTokens != 12 {
			t.Fatalf("max_tokens = %d, want 12", body.MaxTokens)
		}
		if len(body.Messages) != 1 || !strings.Contains(body.Messages[0].Content, "12个字") || !strings.Contains(body.Messages[0].Content, "a21 mock transcript") {
			t.Fatalf("fast companion prompt not applied: %+v", body.Messages)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"reasoning":"先识别情绪"}}]}`,
			`data: {"choices":[{"delta":{"content":"收到我会帮你稳住"}}]}`,
			`data: [DONE]`,
			``,
		}, "\n")))
	}))
	t.Cleanup(server.Close)
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DEEPSEEK_BASE_URL", server.URL)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text-provider", "deepseek", "--execute-text-provider", "--text", "用户原文不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if ttsInput != "收到我会帮你稳住" {
		t.Fatalf("tts input = %q, want provider voice preview", ttsInput)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"text_stream_provider": "deepseek"`,
		`"text_stream_executed": true`,
		`"text_stream_content_delta_count": 1`,
		`"text_stream_reasoning_delta_count": 1`,
		`"tts_provider": "macos_say"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
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
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-chat", "用户原文不要进报告", "收到我会帮你稳住", "先识别情绪", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackCanUseLocalOllamaTextStreamWithoutLeakingContent(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	var ttsInput string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInput = options.Text
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-ollama-test.wav")
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
			DurationMS:      12,
			TTSFirstAudioMS: 12,
		}, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("path = %q, want /api/chat", r.URL.Path)
		}
		var body struct {
			Model    string `json:"model"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
			Options struct {
				NumPredict int `json:"num_predict"`
			} `json:"options"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "qwen2.5:0.5b" || body.Options.NumPredict != 12 {
			t.Fatalf("ollama body = %+v", body)
		}
		if len(body.Messages) != 1 || !strings.Contains(body.Messages[0].Content, "12个字") || !strings.Contains(body.Messages[0].Content, "a21 mock transcript") {
			t.Fatalf("fast companion prompt not applied: %+v", body.Messages)
		}
		_, _ = w.Write([]byte(strings.Join([]string{
			`{"message":{"content":"收到我会帮你稳住"}}`,
			`{"done":true}`,
			``,
		}, "\n")))
	}))
	t.Cleanup(server.Close)
	t.Setenv("A21_PROVIDER_PRIMARY", "local_ollama")
	t.Setenv("A21_LOCAL_OLLAMA_BASE_URL", server.URL)
	t.Setenv("A21_LOCAL_OLLAMA_MODEL", "qwen2.5:0.5b")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text-provider", "local_ollama", "--execute-text-provider", "--text", "用户原文不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if ttsInput != "收到我会帮你稳住" {
		t.Fatalf("tts input = %q, want local Ollama voice preview", ttsInput)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"text_stream_provider": "local_ollama"`,
		`"text_stream_executed": true`,
		`"text_stream_content_delta_count": 1`,
		`"text_stream_reasoning_delta_count": 0`,
		`"tts_provider": "macos_say"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("reports = %v, %v", matches, err)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"qwen2.5:0.5b", "用户原文不要进报告", "收到我会帮你稳住", server.URL, "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackCanUseHotPlugTextStreamProfileWithoutLeakingContent(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	var ttsInput string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInput = options.Text
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-hotplug-test.wav")
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
			DurationMS:      12,
			TTSFirstAudioMS: 12,
		}, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		var body struct {
			Model     string `json:"model"`
			MaxTokens int    `json:"max_tokens"`
			Messages  []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "hidden-hotplug-model" || body.MaxTokens != 12 {
			t.Fatalf("provider request body = %+v", body)
		}
		if len(body.Messages) != 1 || !strings.Contains(body.Messages[0].Content, "12个字") || !strings.Contains(body.Messages[0].Content, "a21 mock transcript") {
			t.Fatalf("fast companion prompt not applied: %+v", body.Messages)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"reasoning":"不要泄露推理"}}]}`,
			`data: {"choices":[{"delta":{"content":"热插拔已接入"}}]}`,
			`data: [DONE]`,
			``,
		}, "\n")))
	}))
	t.Cleanup(server.Close)
	profilePath := filepath.Join(t.TempDir(), "a21-provider-profiles.json")
	if err := os.WriteFile(profilePath, []byte(`{
		"name": "a21_loopback_vendor",
		"label": "A21 loopback vendor",
		"family": "text_stream",
		"protocol": "openai_chat_completions",
		"capabilities": ["llm", "streaming_text", "text_stream"],
		"api_key_env": "A21_LOOPBACK_VENDOR_API_KEY",
		"model_env": "A21_LOOPBACK_VENDOR_MODEL",
		"base_url_env": "A21_LOOPBACK_VENDOR_BASE_URL",
		"endpoint_path": "/chat/completions",
		"route_eligible": true
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_PROVIDER_PROFILES_PATH", profilePath)
	t.Setenv("A21_PROVIDER_PRIMARY", "a21_loopback_vendor")
	t.Setenv("A21_LOOPBACK_VENDOR_API_KEY", "sk-a21-hotplug-secret")
	t.Setenv("A21_LOOPBACK_VENDOR_MODEL", "hidden-hotplug-model")
	t.Setenv("A21_LOOPBACK_VENDOR_BASE_URL", server.URL+"/v1")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text-provider", "a21_loopback_vendor", "--execute-text-provider", "--text", "用户原文不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if ttsInput != "热插拔已接入" {
		t.Fatalf("tts input = %q, want hotplug provider voice preview", ttsInput)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"text_stream_provider": "a21_loopback_vendor"`,
		`"text_stream_executed": true`,
		`"text_stream_content_delta_count": 1`,
		`"text_stream_reasoning_delta_count": 1`,
		`"tts_provider": "macos_say"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("reports = %v, %v", matches, err)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{profilePath, filepath.Dir(profilePath), server.URL, "sk-a21-hotplug-secret", "hidden-hotplug-model", "用户原文不要进报告", "热插拔已接入", "不要泄露推理", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s stderr=%s report=%s", forbidden, stdout.String(), stderr.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackFallsBackToConfiguredTextProviderWithoutLeakingContent(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	var ttsInput string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInput = options.Text
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-fallback-test.wav")
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
			DurationMS:      12,
			TTSFirstAudioMS: 12,
		}, nil
	}
	var sawPrimary bool
	var sawFallback bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/primary/"):
			sawPrimary = true
			http.Error(w, `primary failed sk-a21-primary-secret`, http.StatusServiceUnavailable)
		case strings.HasPrefix(r.URL.Path, "/fallback/"):
			sawFallback = true
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte(strings.Join([]string{
				`data: {"choices":[{"delta":{"reasoning":"fallback reasoning must not leak"}}]}`,
				`data: {"choices":[{"delta":{"content":"兜底已接管"}}]}`,
				`data: [DONE]`,
				``,
			}, "\n")))
		default:
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	profilePath := filepath.Join(t.TempDir(), "a21-provider-profiles.json")
	if err := os.WriteFile(profilePath, []byte(`{
		"name": "a21_loopback_fallback",
		"label": "A21 loopback fallback",
		"family": "text_stream",
		"protocol": "openai_chat_completions",
		"capabilities": ["llm", "streaming_text", "text_stream"],
		"api_key_env": "A21_LOOPBACK_FALLBACK_API_KEY",
		"model_env": "A21_LOOPBACK_FALLBACK_MODEL",
		"base_url_env": "A21_LOOPBACK_FALLBACK_BASE_URL",
		"endpoint_path": "/chat/completions",
		"route_eligible": true
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_PROVIDER_PROFILES_PATH", profilePath)
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-primary-secret")
	t.Setenv("A21_DEEPSEEK_BASE_URL", server.URL+"/primary")
	t.Setenv("A21_LOOPBACK_FALLBACK_API_KEY", "sk-a21-fallback-secret")
	t.Setenv("A21_LOOPBACK_FALLBACK_MODEL", "hidden-fallback-model")
	t.Setenv("A21_LOOPBACK_FALLBACK_BASE_URL", server.URL+"/fallback")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text-provider", "deepseek", "--fallback-text-provider", "a21_loopback_fallback", "--execute-text-provider", "--text", "用户原文不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !sawPrimary || !sawFallback {
		t.Fatalf("saw primary/fallback = %v/%v, want both", sawPrimary, sawFallback)
	}
	if ttsInput != "兜底已接管" {
		t.Fatalf("tts input = %q, want fallback provider voice preview", ttsInput)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"text_stream_provider": "a21_loopback_fallback"`,
		`"text_stream_fallback_used": true`,
		`"text_stream_fallback_provider": "a21_loopback_fallback"`,
		`"text_stream_fallback_reason": "primary_failed"`,
		`"text_stream_executed": true`,
		`"tts_provider": "macos_say"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("reports = %v, %v", matches, err)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{profilePath, filepath.Dir(profilePath), server.URL, "sk-a21-primary-secret", "sk-a21-fallback-secret", "hidden-fallback-model", "用户原文不要进报告", "兜底已接管", "fallback reasoning", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("loopback fallback leaked %q: stdout=%s stderr=%s report=%s", forbidden, stdout.String(), stderr.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackRecordsLocalAckSeparatelyFromProviderAnswer(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	var ttsInputs []string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInputs = append(ttsInputs, options.Text)
		outputPath := filepath.Join(options.OutputDir, fmt.Sprintf("a21-local-voice-loopback-ack-test-%d.wav", len(ttsInputs)))
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		firstAudioMS := float64(13)
		if len(ttsInputs) == 1 {
			firstAudioMS = 7
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
			DurationMS:      firstAudioMS,
			TTSFirstAudioMS: firstAudioMS,
		}, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"content":"这是正式回答"}}]}`,
			`data: [DONE]`,
			``,
		}, "\n")))
	}))
	t.Cleanup(server.Close)
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DEEPSEEK_BASE_URL", server.URL)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text-provider", "deepseek", "--execute-text-provider", "--text", "用户原文不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if len(ttsInputs) != 2 {
		t.Fatalf("tts calls = %d, want local ack + provider answer: %#v", len(ttsInputs), ttsInputs)
	}
	if !strings.Contains(ttsInputs[0], "我在") || ttsInputs[0] == "这是正式回答" {
		t.Fatalf("first TTS input should be local ack, got %q", ttsInputs[0])
	}
	if ttsInputs[1] != "这是正式回答" {
		t.Fatalf("second TTS input = %q, want provider answer", ttsInputs[1])
	}
	for _, want := range []string{
		`"local_ack_enabled": true`,
		`"local_ack_tts_first_audio_ms": 7`,
		`"local_ack_first_audio_total_ms"`,
		`"answer_first_audio_total_p95_ms"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
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
	var report localVoiceLoopbackReport
	if err := json.Unmarshal(reportData, &report); err != nil {
		t.Fatal(err)
	}
	if report.AnswerFirstAudioP95MS < 20 {
		t.Fatalf("answer first audio total = %.3f, want local ack gate + answer TTS", report.AnswerFirstAudioP95MS)
	}
	for _, forbidden := range []string{"sk-a21-secret", "用户原文不要进报告", "这是正式回答", "我在", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackCanUseSherpaASRWithoutLeakingTranscript(t *testing.T) {
	originalTTS := synthesizeMacOSSay
	originalASR := runSherpaONNXASR
	t.Cleanup(func() {
		synthesizeMacOSSay = originalTTS
		runSherpaONNXASR = originalASR
	})
	runSherpaONNXASR = func(ctx context.Context, options audio.LocalASROptions) (audio.LocalASRResult, error) {
		if options.ModelDir != "/tmp/a21-asr-model" || options.Family != "paraformer" || options.WAVPath != "/tmp/a21-asr.wav" {
			t.Fatalf("asr options = %+v", options)
		}
		return audio.LocalASRResult{
			Transcript: "真实转写不要进报告",
			Report: audio.LocalASRReport{
				SchemaVersion:    "a21.audio.local_asr.v1",
				GeneratedAtMS:    time.Now().UnixMilli(),
				Status:           "passed",
				Provider:         "sherpa_onnx",
				Engine:           "paraformer",
				ModelDir:         "a21-asr-model",
				WAVName:          "a21-asr.wav",
				InputDurationMS:  1200,
				DecodeDurationMS: 88,
				RealTimeFactor:   0.073,
				TextChars:        9,
				TranscriptPolicy: "transcript_not_recorded",
			},
		}, nil
	}
	var ttsInput string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInput = options.Text
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-asr-test.wav")
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
			DurationMS:      11,
			TTSFirstAudioMS: 11,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--asr-provider", "sherpa_onnx", "--asr-family", "paraformer", "--asr-model-dir", "/tmp/a21-asr-model", "--asr-wav", "/tmp/a21-asr.wav", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if ttsInput != "A21 loopback response" {
		t.Fatalf("tts input = %q, want mock text stream response", ttsInput)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"asr_provider": "sherpa_onnx"`,
		`"asr_engine": "paraformer"`,
		`"asr_model_dir": "a21-asr-model"`,
		`"asr_wav_name": "a21-asr.wav"`,
		`"asr_text_chars": 9`,
		`"asr_transcript_policy": "transcript_not_recorded"`,
		`"tts_provider": "macos_say"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
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
	for _, forbidden := range []string{"真实转写不要进报告", "/tmp/a21-asr-model", "/tmp/a21-asr.wav", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunStackChanLocalTTSPlaybackSendsRedactedAudioChunks(t *testing.T) {
	original := synthesizeSherpaONNX
	t.Cleanup(func() { synthesizeSherpaONNX = original })
	var receivedChunks int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			DeviceID    string `json:"device_id"`
			Text        string `json:"text"`
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.DeviceID != "stackchan-001" {
			t.Fatalf("request = %+v", request)
		}
		if len(request.AudioChunks) > 0 && request.Text != "A21 LOCAL TTS" {
			t.Fatalf("audio request text = %q", request.Text)
		}
		receivedChunks += len(request.AudioChunks)
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	synthesizeSherpaONNX = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, "a21-sherpa-playback-test.wav")
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 640))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "sherpa_onnx",
			Engine:          "vits_icefall_zh_aishell3",
			Voice:           "sid_21",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     1280,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      33,
			TTSFirstAudioMS: 33,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-local-tts-playback", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--engine", "sherpa_onnx", "--text", "不要写进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if receivedChunks != 8 {
		t.Fatalf("received chunks = %d, want 8 padded speaker-batch chunks", receivedChunks)
	}
	for _, want := range []string{`"schema_version": "a21.stackchan_local_tts_playback.v1"`, `"status": "passed"`, `"tts_provider": "sherpa_onnx"`, `"playback_chunks": 2`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "不要写进报告") {
		t.Fatalf("playback report leaked text: %s", stdout.String())
	}
}

func TestRunStackChanLocalTTSPlaybackSupportsVoiceClonePersonaReport(t *testing.T) {
	original := synthesizeVoiceCloneCLI
	t.Cleanup(func() { synthesizeVoiceCloneCLI = original })
	refAudio := filepath.Join(t.TempDir(), "a21-persona-reference.wav")
	if err := os.WriteFile(refAudio, []byte("RIFF-a21-reference"), 0o644); err != nil {
		t.Fatal(err)
	}
	var receivedChunks int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			DeviceID    string `json:"device_id"`
			Text        string `json:"text"`
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.DeviceID != "stackchan-001" {
			t.Fatalf("request = %+v", request)
		}
		if len(request.AudioChunks) > 0 && request.Text != "A21 LOCAL TTS" {
			t.Fatalf("audio request text = %q", request.Text)
		}
		receivedChunks += len(request.AudioChunks)
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	synthesizeVoiceCloneCLI = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		if options.Text != "不要写进报告" ||
			options.VoiceCloneCommand != "/a21/bin/a21-index-tts2-wrapper" ||
			options.VoiceCloneModel != "Index-TTS2" ||
			options.VoiceCloneReferenceAudioPath != refAudio ||
			options.VoiceCloneReferenceText != "参考文本不能进报告" ||
			options.VoiceClonePersona != "A21 Workmate" ||
			options.VoiceCloneStyle != "Warm-Pro" {
			t.Fatalf("clone options = %+v", options)
		}
		outputPath := filepath.Join(options.OutputDir, "a21-voice-clone-playback-test.wav")
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 640))
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
			OutputBytes:     1280,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      29,
			TTSFirstAudioMS: 29,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"stackchan-local-tts-playback",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--engine", "voice_clone_cli",
		"--text", "不要写进报告",
		"--clone-command", "/a21/bin/a21-index-tts2-wrapper",
		"--clone-model", "Index-TTS2",
		"--clone-ref-audio", refAudio,
		"--clone-ref-text", "参考文本不能进报告",
		"--voice-persona", "A21 Workmate",
		"--voice-style", "Warm-Pro",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if receivedChunks != 8 {
		t.Fatalf("received chunks = %d, want 8 padded speaker-batch chunks", receivedChunks)
	}
	for _, want := range []string{
		`"tts_provider": "voice_clone_cli"`,
		`"tts_engine": "voice_clone_cli"`,
		`"tts_model": "index_tts2"`,
		`"tts_voice": "a21_workmate"`,
		`"tts_voice_persona": "a21_workmate"`,
		`"tts_style_profile": "warm_pro"`,
		`"tts_reference_audio": "a21-persona-reference.wav"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{server.URL, "不要写进报告", "参考文本不能进报告", refAudio, "Authorization", "Bearer", "raw_audio", "data_base64"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("voice clone playback report leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunStackChanLocalTTSPlaybackPrerollsInitialAudioBuffer(t *testing.T) {
	original := synthesizeSherpaONNX
	t.Cleanup(func() { synthesizeSherpaONNX = original })
	var batchSizes []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.AudioChunks) > 0 {
			batchSizes = append(batchSizes, len(request.AudioChunks))
		}
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	synthesizeSherpaONNX = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, "a21-sherpa-playback-preroll-test.wav")
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 320*10))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "sherpa_onnx",
			Engine:          "vits_icefall_zh_aishell3",
			Voice:           "sid_21",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     6400,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      33,
			TTSFirstAudioMS: 33,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-local-tts-playback", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--engine", "sherpa_onnx", "--text", "不要写进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if got, want := fmt.Sprint(batchSizes), "[8 8]"; got != want {
		t.Fatalf("audio batch sizes = %s, want %s", got, want)
	}
}

func TestRunStackChanLocalTTSPlaybackKeepsSteadyBatchesLargeEnoughForSpeakerPump(t *testing.T) {
	original := synthesizeSherpaONNX
	t.Cleanup(func() { synthesizeSherpaONNX = original })
	var batchSizes []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.AudioChunks) > 0 {
			batchSizes = append(batchSizes, len(request.AudioChunks))
		}
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	synthesizeSherpaONNX = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, "a21-sherpa-playback-steady-batch-test.wav")
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 320*20))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "sherpa_onnx",
			Engine:          "vits_icefall_zh_aishell3",
			Voice:           "sid_21",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     12800,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      400,
			TTSFirstAudioMS: 33,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-local-tts-playback", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--engine", "sherpa_onnx", "--text", "不要写进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if got, want := fmt.Sprint(batchSizes), "[8 8 8]"; got != want {
		t.Fatalf("audio batch sizes = %s, want %s", got, want)
	}
}

func TestRunStackChanLocalTTSPlaybackPacesOfficialCodecBridge(t *testing.T) {
	original := synthesizeSherpaONNX
	t.Cleanup(func() { synthesizeSherpaONNX = original })
	var requestTimes []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.AudioChunks) > 0 {
			requestTimes = append(requestTimes, time.Now())
		}
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	synthesizeSherpaONNX = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, "a21-sherpa-playback-prebuffer-test.wav")
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 320*40))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "sherpa_onnx",
			Engine:          "vits_icefall_zh_aishell3",
			Voice:           "sid_21",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     25600,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      800,
			TTSFirstAudioMS: 33,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-local-tts-playback", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--engine", "sherpa_onnx", "--text", "不要写进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if len(requestTimes) < 5 {
		t.Fatalf("audio request count = %d, want at least 5", len(requestTimes))
	}
	for i := 1; i <= 2; i++ {
		if gap := requestTimes[i].Sub(requestTimes[i-1]); gap < 120*time.Millisecond {
			t.Fatalf("official codec bridge pacing gap %d = %s, want >= 120ms", i, gap)
		}
	}
}

func TestRunStackChanLocalTTSPlaybackCanSendExistingA21WAV(t *testing.T) {
	original := synthesizeSherpaONNX
	t.Cleanup(func() { synthesizeSherpaONNX = original })
	synthesizeSherpaONNX = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		t.Fatal("existing WAV playback must not synthesize TTS")
		return audio.LocalTTSReport{}, nil
	}
	var receivedChunks int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		receivedChunks += len(request.AudioChunks)
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "a21-existing-playback.wav")
	writeAppTestWAV(t, wavPath, 16000, bytes.Repeat([]byte{1, 0}, 320*3))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-local-tts-playback", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--wav", wavPath, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if receivedChunks != 8 {
		t.Fatalf("received chunks = %d, want 8 padded speaker-batch chunks", receivedChunks)
	}
	for _, want := range []string{`"tts_provider": "wav_file"`, `"tts_audio_path": "a21-existing-playback.wav"`, `"playback_chunks": 3`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanFastCompanionTurnHelpListsLocalOllama(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-fast-companion-turn", "--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{"mock_text_stream", "deepseek", "local_ollama"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help output missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanFastCompanionTurnDeliversAckAndAnswerWithoutLeakingText(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, fmt.Sprintf("a21-fast-companion-%d.wav", time.Now().UnixNano()))
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 640))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "macos_say",
			Voice:           "Tingting",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     1280,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      11,
			TTSFirstAudioMS: 11,
		}, nil
	}
	var audioRequests int
	var idleRequests int
	var totalChunks int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices":
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"speaker":"available","screen":"available","rgb":"available","servo_y":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/v1/devices/control":
			var request struct {
				DeviceID    string `json:"device_id"`
				State       string `json:"state"`
				Text        string `json:"text"`
				AudioChunks []struct {
					DataBase64 string `json:"data_base64"`
				} `json:"audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.DeviceID != "stackchan-001" {
				t.Fatalf("device id = %q", request.DeviceID)
			}
			if len(request.AudioChunks) > 0 {
				audioRequests++
				totalChunks += len(request.AudioChunks)
				if request.Text != "A21 LOCAL TTS" {
					t.Fatalf("audio request text = %q", request.Text)
				}
			} else if request.State == "idle" {
				idleRequests++
			}
			fmt.Fprint(w, `{"trace_id":"a21-trace-fast","session_id":"a21-session-fast","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		default:
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-fast-companion-turn", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--engine", "macos_say", "--text", "用户输入不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if audioRequests < 2 || totalChunks < 4 {
		t.Fatalf("audio requests/chunks = %d/%d, want ack and answer playback", audioRequests, totalChunks)
	}
	if idleRequests != 1 {
		t.Fatalf("idle requests = %d, want final playback clear", idleRequests)
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_fast_companion_turn.v1"`,
		`"status": "passed"`,
		`"device_online": true`,
		`"local_ack_playback_chunks": 2`,
		`"answer_playback_chunks": 2`,
		`"playback_cleared": true`,
		`"m3_candidate": false`,
		`"listen_source": "host_fixture"`,
		`"report_path"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-stackchan-fast-companion-turn-*.json"))
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
	for _, forbidden := range []string{"用户输入不要进报告", "A21 loopback response", "嗯，我在", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("fast companion report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunStackChanFastCompanionTurnWritesFailedReportWhenGatewayUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
		http.Error(w, "gateway not ready", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-fast-companion-turn", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--text", "不要泄漏这句话", "--output-dir", dir}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_fast_companion_turn.v1"`,
		`"status": "failed"`,
		`"device_online": false`,
		`"m3_candidate": false`,
		`"gateway device report failed"`,
		`"report_path"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-stackchan-fast-companion-turn-*.json"))
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
	for _, forbidden := range []string{"不要泄漏这句话", "Authorization", "Bearer", "sk-"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("failed report leaked %q: stdout=%s stderr=%s report=%s", forbidden, stdout.String(), stderr.String(), reportData)
		}
	}
}

func TestRunStackChanFastCompanionTurnConsumesStackChanMicEvidenceWithoutLeakingAudio(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, fmt.Sprintf("a21-fast-companion-mic-%d.wav", time.Now().UnixNano()))
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 640))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "macos_say",
			Voice:           "Tingting",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     1280,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      11,
			TTSFirstAudioMS: 11,
		}, nil
	}
	payload := appTestPCM16Base64(12000)
	frames := make([]string, 0, 20)
	for seq := 1; seq <= 20; seq++ {
		rms := 0.21
		if seq == 1 {
			rms = 0.23
		}
		frames = append(frames, fmt.Sprintf(`{"device_id":"stackchan-001","trace_id":"a21-trace-audio-%06d","session_id":"%%s","seq":%d,"sample_rate_hz":16000,"channels":1,"duration_ms":20,"data_bytes":640,"data_base64":"%s","rms":%.2f,"vad_detector":"a21-rms-vad","speech_detected":true,"speech_active":true,"ingress_buffer_frames":%d}`, seq, seq, payload, rms, seq))
	}
	var sawListening bool
	var sawIdle bool
	var audioRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices":
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","rgb":"available","servo_y":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/v1/audio/recent":
			if r.URL.Query().Get("include_audio") != "1" {
				t.Fatalf("recent audio request missing include_audio=1: %s", r.URL.RawQuery)
			}
			if r.URL.Query().Get("device_id") != "stackchan-001" {
				t.Fatalf("recent audio device = %q", r.URL.Query().Get("device_id"))
			}
			limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
			if err != nil || limit < 64 {
				t.Fatalf("recent audio limit = %q, want enough frames for capture window", r.URL.Query().Get("limit"))
			}
			sessionID := r.URL.Query().Get("session_id")
			responseFrames := strings.ReplaceAll(strings.Join(frames, ","), "%s", sessionID)
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.audio_recent.v1","device_id":"stackchan-001","session_id":"%s","include_audio":true,"frames":[%s]}`, sessionID, responseFrames)
		case "/v1/devices/control":
			var request struct {
				DeviceID       string `json:"device_id"`
				State          string `json:"state"`
				Text           string `json:"text"`
				AudioProbeOnly bool   `json:"audio_probe_only"`
				AudioChunks    []struct {
					DataBase64 string `json:"data_base64"`
				} `json:"audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.DeviceID != "stackchan-001" {
				t.Fatalf("device id = %q", request.DeviceID)
			}
			if request.State == "listening" && request.AudioProbeOnly {
				sawListening = true
			}
			if request.State == "idle" {
				sawIdle = true
			}
			if len(request.AudioChunks) > 0 {
				audioRequests++
			}
			fmt.Fprint(w, `{"trace_id":"a21-trace-fast","session_id":"a21-session-fast","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		default:
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-fast-companion-turn", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--engine", "macos_say", "--listen-source", "stackchan_mic", "--repeat", "1", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s\n%s", code, stderr.String(), stdout.String())
	}
	if !sawListening || !sawIdle || audioRequests < 2 {
		t.Fatalf("listening/idle/audio = %t/%t/%d, want physical listen and ack/answer playback", sawListening, sawIdle, audioRequests)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"listen_source": "stackchan_mic"`,
		`"listen_detected": true`,
		`"physical_mic_frame_count": 20`,
		`"physical_mic_audio_bytes": 12800`,
		`"physical_mic_duration_ms": 400`,
		`"physical_mic_capture_window_ms": 1200`,
		`"physical_mic_rms_max": 0.23`,
		`"physical_mic_wav_name"`,
		`"m3_candidate": false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-stackchan-fast-companion-turn-*.json"))
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
	for _, forbidden := range []string{payload, "data_base64", "Authorization", "Bearer", "sk-"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("stackchan mic report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunStackChanFastCompanionTurnRejectsStackChanMicWithoutPhysicalEvidence(t *testing.T) {
	var controlRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices":
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","rgb":"available","servo_y":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/v1/audio/recent":
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.audio_recent.v1","device_id":"stackchan-001","session_id":"%s","include_audio":true,"frames":[]}`, r.URL.Query().Get("session_id"))
		case "/v1/devices/control":
			controlRequests++
			var request struct {
				State       string `json:"state"`
				AudioChunks []struct {
					DataBase64 string `json:"data_base64"`
				} `json:"audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if len(request.AudioChunks) > 0 {
				t.Fatalf("stackchan_mic without physical evidence must not deliver synthetic playback")
			}
		default:
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-fast-companion-turn", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--listen-source", "stackchan_mic", "--output-dir", dir}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if controlRequests == 0 {
		t.Fatalf("control requests = %d, want listening and cleanup controls", controlRequests)
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_fast_companion_turn.v1"`,
		`"status": "failed"`,
		`"listen_source": "stackchan_mic"`,
		`"m3_candidate": false`,
		`"physical mic evidence missing"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunV21AdapterSmokeExecutesQueryAndWritesRedactedReport(t *testing.T) {
	var sawProfessionalRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a21/v21/query" {
			t.Fatalf("path = %q, want /a21/v21/query", r.URL.Path)
		}
		var request struct {
			TraceID            string `json:"trace_id"`
			SessionID          string `json:"session_id"`
			Mode               string `json:"mode"`
			Utterance          string `json:"utterance"`
			LatencyProfile     string `json:"latency_profile"`
			AnswerStyle        string `json:"answer_style"`
			PrivacyScope       string `json:"privacy_scope"`
			MaxFirstResponseMS int    `json:"max_first_response_ms"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		sawProfessionalRequest = request.TraceID != "" &&
			request.SessionID != "" &&
			request.Mode == "professional" &&
			request.Utterance == "查一下语音唤醒误触发" &&
			request.LatencyProfile == "fast_first" &&
			request.AnswerStyle == "voice_first_with_citations" &&
			request.PrivacyScope == "professional_only" &&
			request.MaxFirstResponseMS == 1200
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"trace_id":"a21-trace-v21-smoke",
			"fast_answer":"历史讨论集中在多人说话和相似音节误唤醒。",
			"confidence":0.82,
			"evidence":[{"title":"语音唤醒体验复盘","type":"meeting","source_id":"v21-doc-001","summary":"提到多人说话导致误唤醒。"}],
			"speech_blocks":["我先说结论。"],
			"screen_cards":[{"label":"结论","text":"误唤醒集中在 2 类场景"}],
			"follow_ups":["要不要按车型展开？"]
		}`))
	}))
	defer server.Close()
	dir, err := os.MkdirTemp("", "a21-adapter-smoke-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"v21-adapter-smoke",
		"--adapter-url", server.URL,
		"--query", "查一下语音唤醒误触发",
		"--execute",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !sawProfessionalRequest {
		t.Fatal("server did not receive professional V21 adapter smoke request")
	}
	for _, want := range []string{
		`"schema_version": "a21.v21_adapter_smoke.v1"`,
		`"generated_at_ms"`,
		`"adapter": "a21-v21-adapter"`,
		`"status": "passed"`,
		`"executed": true`,
		`"mode": "professional"`,
		`"latency_profile": "fast_first"`,
		`"answer_style": "voice_first_with_citations"`,
		`"privacy_scope": "professional_only"`,
		`"max_first_response_ms": 1200`,
		`"query_path": "/a21/v21/query"`,
		`"evidence_count": 1`,
		`"speech_block_count": 1`,
		`"screen_card_count": 1`,
		`"follow_up_count": 1`,
		`"redaction_ok": true`,
		`"report_path": "a21-v21-adapter-smoke-`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-v21-adapter-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("v21 adapter smoke reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, forbidden := range []string{"语音唤醒", "历史讨论", server.URL, dir, filepath.ToSlash(dir)} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("v21 adapter smoke leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func TestRunV21ProfessionalReadinessWritesRedactedHostReport(t *testing.T) {
	dir, err := os.MkdirTemp("", "a21-prof-readiness-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"v21-professional-readiness",
		"--adapter-url", "http://127.0.0.1:21121",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.v21_professional_readiness.v1"`,
		`"checking_ack_available": true`,
		`"checking_ack_within_1200": true`,
		`"evidence_available": true`,
		`"cards_available": true`,
		`"follow_ups_available": true`,
		`"adapter_configured": true`,
		`"adapter_executed": false`,
		`"redaction_ok": true`,
		`"professional_acceptance_status": "host_mock_ready"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %s: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-v21-professional-readiness-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("professional readiness reports = %d, %v", len(matches), err)
	}
	reportJSON, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"professional readiness fixture query",
		"语音唤醒体验复盘",
		"提到多人说话",
		"要不要按车型",
		"http://",
		"https://",
		dir,
		"secret",
		"api_key",
		"provider output",
		"reasoning",
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportJSON), forbidden) {
			t.Fatalf("professional readiness leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
	if !strings.Contains(stdout.String(), `"report_path": "a21-v21-professional-readiness-`) {
		t.Fatalf("stdout report_path should be basename-only: %s", stdout.String())
	}
}

func TestRunV21ProfessionalReadinessReportsMisconfigurationWithoutLeak(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"v21-professional-readiness",
		"--adapter-url", "http://user:secret-token@127.0.0.1:21121/a21/v21/query",
	}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"status": "blocked"`,
		`"adapter_configured": true`,
		`"adapter_executed": false`,
		`"professional_acceptance_status": "adapter_misconfigured"`,
		`"code": "v21_adapter_misconfigured"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %s: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"secret-token", "user:", "http://", "127.0.0.1:21121", "/a21/v21/query"} {
		if strings.Contains(stdout.String()+stderr.String(), forbidden) {
			t.Fatalf("misconfigured readiness leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestV21AdapterBridgeExecutesRealBackendRetrievalContract(t *testing.T) {
	activeReleaseID := "rel_active"
	var sawRetrievalQuery bool
	v21Backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/healthz":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/collections":
			if r.Header.Get("X-Dev-Role") != "VIEWER" {
				t.Fatalf("missing viewer dev header")
			}
			writeV21BridgeJSON(w, http.StatusOK, []v21CollectionView{{
				ID:              "col_vehicle",
				Name:            "Vehicle Knowledge",
				ActiveReleaseID: &activeReleaseID,
			}})
		case "/api/v1/collections/col_vehicle/retrieval/query":
			var request struct {
				Query string `json:"query"`
				Limit int    `json:"limit"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			sawRetrievalQuery = request.Query == "哪些车有儿童锁" && request.Limit == 5
			writeV21BridgeJSON(w, http.StatusOK, v21RetrievalQueryResponse{
				CollectionID: "col_vehicle",
				Results: []v21RetrievalResult{{
					AnchorID:    "ca_child_lock",
					SourceLabel: "儿童锁证据",
					Excerpt:     "G02ES、G02ESVR 支持座椅儿童锁。",
					Score:       0.91,
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer v21Backend.Close()
	handler, err := newV21AdapterBridgeHandler(context.Background(), v21AdapterBridgeOptions{V21URL: v21Backend.URL})
	if err != nil {
		t.Fatal(err)
	}
	adapter := httptest.NewServer(handler)
	defer adapter.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"v21-adapter-smoke", "--adapter-url", adapter.URL, "--query", "哪些车有儿童锁", "--execute"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !sawRetrievalQuery {
		t.Fatal("bridge did not call V21 retrieval query with the adapter contract")
	}
	for _, want := range []string{`"status": "passed"`, `"evidence_count": 1`, `"follow_up_count": 1`, `"confidence": 0.91`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestV21AdapterBridgeRetriesChildLockASRFragmentWithoutLeakingQuery(t *testing.T) {
	activeReleaseID := "rel_active"
	var queries []string
	v21Backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/healthz":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/collections":
			writeV21BridgeJSON(w, http.StatusOK, []v21CollectionView{{
				ID:              "col_vehicle",
				Name:            "Vehicle Knowledge",
				ActiveReleaseID: &activeReleaseID,
			}})
		case "/api/v1/collections/col_vehicle/retrieval/query":
			var request struct {
				Query string `json:"query"`
				Limit int    `json:"limit"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			queries = append(queries, request.Query)
			if request.Query != "儿童锁 车型 车门" {
				writeV21BridgeJSON(w, http.StatusOK, v21RetrievalQueryResponse{
					CollectionID: "col_vehicle",
					Results:      []v21RetrievalResult{},
				})
				return
			}
			writeV21BridgeJSON(w, http.StatusOK, v21RetrievalQueryResponse{
				CollectionID: "col_vehicle",
				Results: []v21RetrievalResult{{
					AnchorID:    "ca_child_lock",
					SourceLabel: "儿童锁证据",
					Excerpt:     "G02ES、G02ESVR 支持座椅儿童锁。",
					Score:       0.91,
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer v21Backend.Close()
	handler, err := newV21AdapterBridgeHandler(context.Background(), v21AdapterBridgeOptions{V21URL: v21Backend.URL})
	if err != nil {
		t.Fatal(err)
	}
	adapter := httptest.NewServer(handler)
	defer adapter.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"v21-adapter-smoke", "--adapter-url", adapter.URL, "--query", "儿童是", "--execute"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !reflect.DeepEqual(queries, []string{"儿童是", "儿童锁 车型 车门"}) {
		t.Fatalf("queries = %#v, want original then child-lock expansion", queries)
	}
	for _, want := range []string{`"status": "passed"`, `"evidence_count": 1`, `"speech_block_count": 1`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"儿童是", "儿童锁 车型 车门", v21Backend.URL} {
		if strings.Contains(stdout.String()+stderr.String(), forbidden) {
			t.Fatalf("adapter smoke leaked forbidden query fragment %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestV21AdapterBridgeRejectsNonProfessionalModeBeforeRetrieval(t *testing.T) {
	activeReleaseID := "rel_active"
	var retrievalCalls int
	v21Backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/healthz":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/collections":
			writeV21BridgeJSON(w, http.StatusOK, []v21CollectionView{{
				ID:              "col_vehicle",
				Name:            "Vehicle Knowledge",
				ActiveReleaseID: &activeReleaseID,
			}})
		case "/api/v1/collections/col_vehicle/retrieval/query":
			retrievalCalls++
			http.Error(w, "should not be called", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer v21Backend.Close()
	handler, err := newV21AdapterBridgeHandler(context.Background(), v21AdapterBridgeOptions{V21URL: v21Backend.URL})
	if err != nil {
		t.Fatal(err)
	}
	adapter := httptest.NewServer(handler)
	defer adapter.Close()

	resp, err := http.Post(adapter.URL+v21adapter.QueryPath, "application/json", strings.NewReader(`{
		"trace_id":"a21-trace-v21-unsafe",
		"session_id":"a21-session-v21-unsafe",
		"mode":"workmate",
		"privacy_scope":"professional_only",
		"utterance":"查一下语音唤醒误触发"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if retrievalCalls != 0 {
		t.Fatalf("retrieval calls = %d, want 0", retrievalCalls)
	}
}

func TestV21AdapterBridgeNoResultsReturnsControlledRedactedFailure(t *testing.T) {
	activeReleaseID := "rel_active"
	v21Backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/healthz":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/collections":
			writeV21BridgeJSON(w, http.StatusOK, []v21CollectionView{{
				ID:              "col_vehicle",
				Name:            "Vehicle Knowledge",
				ActiveReleaseID: &activeReleaseID,
			}})
		case "/api/v1/collections/col_vehicle/retrieval/query":
			writeV21BridgeJSON(w, http.StatusOK, v21RetrievalQueryResponse{
				CollectionID: "col_vehicle",
				Results:      []v21RetrievalResult{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer v21Backend.Close()
	handler, err := newV21AdapterBridgeHandler(context.Background(), v21AdapterBridgeOptions{V21URL: v21Backend.URL})
	if err != nil {
		t.Fatal(err)
	}
	adapter := httptest.NewServer(handler)
	defer adapter.Close()

	resp, err := http.Post(adapter.URL+v21adapter.QueryPath, "application/json", strings.NewReader(`{
		"trace_id":"a21-trace-v21-no-results",
		"session_id":"a21-session-v21-no-results",
		"mode":"professional",
		"privacy_scope":"professional_only",
		"utterance":"查一下语音唤醒误触发"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusFailedDependency {
		t.Fatalf("status = %d, want 424: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), `"code":"no_evidence"`) {
		t.Fatalf("body missing redacted no_evidence code: %s", body)
	}
	for _, forbidden := range []string{"查一下语音唤醒误触发", v21Backend.URL, "col_vehicle"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("bridge failure leaked %q: %s", forbidden, body)
		}
	}
}

func TestRunLANProbeWritesDirectRedactedReport(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://user:secret@example.invalid:8080")
	t.Setenv("A21_PROVIDER_PROXY_URL", "http://provider-secret@example.invalid:9000")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"lan-probe",
		"--target", "a21-gateway=" + listener.Addr().String(),
		"--samples", "3",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.lan_probe.v1"`,
		`"ok": true`,
		`"name": "a21-gateway"`,
		`"status": "passed"`,
		`"direct": true`,
		`"samples": 3`,
		`"passed_samples": 3`,
		`"failed_samples": 0`,
		`"p50_ms"`,
		`"p95_ms"`,
		`"jitter_ms"`,
		`"metadata"`,
		`"proxy"`,
		`"report_path"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-lan-probe-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("lan probe reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000", "x21"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("lan probe leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

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
		`"api_key_env": "A21_DOUBAO_API_KEY"`,
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

func TestBuildV21DoctorReportRedactsHealthFailureSecrets(t *testing.T) {
	originalProbe := probeV21AdapterHealth
	probeV21AdapterHealth = func(ctx context.Context, baseURL string) error {
		return errors.New(`Get "http://user:secret-token@127.0.0.1:21121/healthz": connection refused`)
	}
	defer func() {
		probeV21AdapterHealth = originalProbe
	}()

	report := buildV21DoctorReport("http://user:secret-token@127.0.0.1:21121")

	if report.Status != "unhealthy" {
		t.Fatalf("status = %q, want unhealthy", report.Status)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(report.Findings))
	}
	if strings.Contains(report.Findings[0].Detail, "secret-token") || strings.Contains(report.Findings[0].Detail, "user:") {
		t.Fatalf("detail leaked credentials: %q", report.Findings[0].Detail)
	}
}

func TestBuildFirmwareDoctorReportFindsCurrentArtifact(t *testing.T) {
	root := t.TempDir()
	manifest := writeTestFirmwareManifest(t, filepath.Join(root, "firmware", "stackchan"))
	_ = manifest
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-venv", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".a21-tools", "platformio-venv", "bin", "pio"), []byte("#!/bin/sh\necho 'PlatformIO Core, version 6.1.19'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-core"), 0o755); err != nil {
		t.Fatal(err)
	}
	artifactDir := filepath.Join(root, "firmware", "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	report := buildFirmwareDoctorReport(root, "abcdef1")

	if !report.ManifestOK {
		t.Fatalf("ManifestOK = false, findings = %#v", report.Findings)
	}
	if !report.PlatformIOVenvOK {
		t.Fatal("expected local PlatformIO venv to be detected")
	}
	if !report.PlatformIOVersionOK {
		t.Fatalf("expected pinned PlatformIO version, got %#v", report)
	}
	if !report.PlatformIOCoreOK {
		t.Fatal("expected local PlatformIO core to be detected")
	}
	if report.CurrentArtifactPath != artifact {
		t.Fatalf("CurrentArtifactPath = %q, want %q", report.CurrentArtifactPath, artifact)
	}
}

func TestBuildFirmwareDoctorReportRejectsLooseCurrentArtifact(t *testing.T) {
	root := t.TempDir()
	writeTestFirmwareManifest(t, filepath.Join(root, "firmware", "stackchan"))
	pioPath := filepath.Join(root, ".a21-tools", "platformio-venv", "bin", "pio")
	if err := os.MkdirAll(filepath.Dir(pioPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pioPath, []byte("#!/bin/sh\necho 'PlatformIO Core, version 6.1.19'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-core"), 0o755); err != nil {
		t.Fatal(err)
	}
	artifactDir := filepath.Join(root, "firmware", "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	content := []byte("firmware\na21-stackchan\n0.1.0\nm5stack-cores3\nabcdef1\n")
	if err := os.WriteFile(artifact, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	if err := os.WriteFile(artifact+".sha256", []byte(hex.EncodeToString(sum[:])+"  "+filepath.Base(artifact)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := buildFirmwareDoctorReport(root, "abcdef1")

	if report.CurrentArtifactPath != "" {
		t.Fatalf("CurrentArtifactPath = %q, want empty for loose artifact", report.CurrentArtifactPath)
	}
	found := false
	for _, finding := range report.Findings {
		if finding.Code == "firmware_current_artifact_missing" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("findings missing firmware_current_artifact_missing: %#v", report.Findings)
	}
}

func TestBuildFirmwareDoctorReportWarnsOnPlatformIOVersionMismatch(t *testing.T) {
	root := t.TempDir()
	writeTestFirmwareManifest(t, filepath.Join(root, "firmware", "stackchan"))
	pioPath := filepath.Join(root, ".a21-tools", "platformio-venv", "bin", "pio")
	if err := os.MkdirAll(filepath.Dir(pioPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pioPath, []byte("#!/bin/sh\necho 'PlatformIO Core, version 6.2.0'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-core"), 0o755); err != nil {
		t.Fatal(err)
	}

	report := buildFirmwareDoctorReport(root, "")

	if report.PlatformIOVersionOK {
		t.Fatalf("PlatformIOVersionOK = true, want false: %#v", report)
	}
	if report.PlatformIOVersion != "6.2.0" || report.ExpectedPlatformIOVersion != "6.1.19" {
		t.Fatalf("version fields = got %q expected %q", report.PlatformIOVersion, report.ExpectedPlatformIOVersion)
	}
	var found bool
	for _, finding := range report.Findings {
		if finding.Code == "firmware_platformio_version_mismatch" {
			found = true
			if strings.Contains(strings.ToLower(finding.Detail), "x21") || strings.Contains(strings.ToLower(finding.Detail), "v21") {
				t.Fatalf("finding detail leaked legacy identity: %#v", finding)
			}
		}
	}
	if !found {
		t.Fatalf("findings missing firmware_platformio_version_mismatch: %#v", report.Findings)
	}
}

func TestRunDoctorRejectsUnknownFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--wat"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunGatewayHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"gateway", "--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "a21 gateway") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunGatewayRejectsUnknownFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"gateway", "--wat"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunLatencyBenchMockEmitsPercentileReport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"latency-bench", "--mock", "--iterations", "3"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"mode": "mock"`,
		`"iterations": 3`,
		`"mock_turn_ms"`,
		`"professional_turn_ms"`,
		`"barge_in_stop_ms"`,
		`"audio_ws_downlink_ms"`,
		`"audio_ws_barge_in_stop_ms"`,
		`"p50_ms"`,
		`"p95_ms"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunLatencyBenchWritesReportWhenOutputDirProvided(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"latency-bench", "--mock", "--iterations", "2", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"report_path"`) {
		t.Fatalf("stdout missing report_path: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-latency-bench-*.json"))
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
	for _, want := range []string{`"mode": "mock"`, `"iterations": 2`, `"audio_ws_downlink_ms"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("report missing %q: %s", want, string(data))
		}
	}
}

func TestRunLatencyBenchReportIncludesTraceableEnvironmentMetadataWithoutProxySecrets(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://user:secret@example.invalid:8080")
	t.Setenv("A21_PROVIDER_PROXY_URL", "http://provider-secret@example.invalid:9000")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"latency-bench", "--mock", "--iterations", "1", "--output-dir", dir}, &stdout, &stderr)

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
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked proxy value fragment %q: %s", forbidden, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-latency-bench-*.json"))
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
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000"} {
		if strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report leaked proxy value fragment %q: %s", forbidden, reportJSON)
		}
	}
}

func TestRunLatencyBenchRequiresMockMode(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"latency-bench"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--mock is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLatencyBenchPercentileUsesNearestRank(t *testing.T) {
	got := percentileMS([]time.Duration{time.Millisecond, 2 * time.Millisecond, 100 * time.Millisecond}, 0.95)
	if got != 100 {
		t.Fatalf("p95 = %v, want 100", got)
	}
}

func TestRunFirmwareCheckRejectsMissingManifest(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", filepath.Join(t.TempDir(), "missing.json")}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunFirmwareCheckAcceptsA21Manifest(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "a21-firmware.json")
	platformio := filepath.Join(dir, "platformio.ini")
	data := []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`)
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(platformio, []byte(testPlatformIOConfig("m5stack-cores3")), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware manifest ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareCheckDispatchesKindHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--kind", "upload", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"a21 firmware-check --kind upload", "--artifact", "--port", "--commit"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunDeprecatedFirmwareCheckAliasStillDispatches(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"firmware-upload-check", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "a21 firmware-check --kind upload") {
		t.Fatalf("stdout missing upload help: %s", stdout.String())
	}
}

func TestRunPlanExecuteCommandDispatchesHelp(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{args: []string{"firmware-bootstrap-flash", "--help"}, want: "a21 firmware-bootstrap-flash"},
		{args: []string{"firmware-bootstrap-flash-execute", "--help"}, want: "a21 firmware-bootstrap-flash --execute"},
		{args: []string{"stackchan-official-pcm-bridge-nvs", "--help"}, want: "a21 stackchan-official-pcm-bridge-nvs"},
		{args: []string{"stackchan-official-pcm-bridge-nvs-execute", "--help"}, want: "a21 stackchan-official-pcm-bridge-nvs"},
	} {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), tc.want) {
				t.Fatalf("stdout missing %q: %s", tc.want, stdout.String())
			}
		})
	}
}

func TestRunFirmwareCheckRejectsWrongPlatformIOBoard(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "a21-firmware.json")
	data := []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`)
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(testPlatformIOConfig("m5stack-core2")), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "platformio board") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwarePackageCreatesVersionedArtifact(t *testing.T) {
	originalDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalDetector
	}()

	dir := t.TempDir()
	manifest := filepath.Join(dir, "a21-firmware.json")
	if err := os.WriteFile(manifest, []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(testPlatformIOConfig("m5stack-cores3")), 0o644); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3", "firmware.bin")
	if err := os.MkdirAll(filepath.Dir(input), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("firmware a21-stackchan 0.1.0 m5stack-cores3 abcdef1"), 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(dir, "artifacts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-package",
		"--manifest", manifest,
		"--input", input,
		"--output-dir", outputDir,
		"--commit", "abcdef1",
		"--timestamp", "20260530-004500",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	artifact := filepath.Join(outputDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	if _, err := os.Stat(artifact); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(artifact + ".sha256"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(artifact + ".manifest.json"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), filepath.Base(artifact)) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "release_manifest_path") {
		t.Fatalf("stdout missing release_manifest_path: %s", stdout.String())
	}
}

func TestRunFirmwarePackageRejectsDirtySourceTree(t *testing.T) {
	originalDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{
			Root:   "/tmp/a21",
			Clean:  false,
			Detail: " M firmware/stackchan/src/main.cpp\n?? scratch.bin",
		}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	input := filepath.Join(dir, "firmware.bin")
	if err := os.WriteFile(input, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef1"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-package",
		"--manifest", manifest,
		"--input", input,
		"--output-dir", filepath.Join(dir, "artifacts"),
		"--commit", "abcdef1",
		"--timestamp", "20260530-004500",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "source tree is dirty") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "firmware/stackchan/src/main.cpp") {
		t.Fatalf("stderr missing dirty detail: %q", stderr.String())
	}
}

func TestRunFirmwareCheckRejectsUnpinnedPlatformIOPlatform(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	config := strings.ReplaceAll(testPlatformIOConfig("m5stack-cores3"), "platform = espressif32@7.0.1", "platform = espressif32")
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "platform") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareCheckRejectsLegacyGatewayPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	config := strings.ReplaceAll(testPlatformIOConfig("m5stack-cores3"), "-D A21_GATEWAY_PORT=21080", "-D A21_GATEWAY_PORT=8080")
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "gateway port") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareArtifactCheckAcceptsMatchingArtifactAndChecksum(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware artifact ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareArtifactCheckRejectsWrongBoardInArtifactName(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-core2-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "board") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareArtifactCheckRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	if err := os.WriteFile(artifact, []byte("firmware"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact+".sha256", []byte(strings.Repeat("0", 64)+"  "+filepath.Base(artifact)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "checksum") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareCurrentArtifactCheckAcceptsNewestCommitArtifact(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	oldArtifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	newArtifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-005500.bin")
	writeFirmwareArtifactWithChecksum(t, oldArtifact, []byte("old firmware"))
	writeFirmwareArtifactWithChecksum(t, newArtifact, []byte("new firmware"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-current-artifact-check",
		"--manifest", manifest,
		"--artifact-dir", dir,
		"--commit", "abcdef1",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), newArtifact) {
		t.Fatalf("stdout missing newest artifact %q: %s", newArtifact, stdout.String())
	}
	if !strings.Contains(stdout.String(), "firmware current artifact ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareCurrentArtifactCheckRequiresCommit(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"firmware-current-artifact-check"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--commit requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareArtifactPrunePlanWritesNoDeleteReport(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifactDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldArtifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-1111111-20260530-010000.bin")
	currentArtifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-2222222-20260530-020000.bin")
	writeFirmwareArtifactWithChecksum(t, oldArtifact, []byte("old firmware"))
	writeFirmwareArtifactWithChecksum(t, currentArtifact, []byte("current firmware"))
	reportDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-prune-plan",
		"--manifest", manifest,
		"--artifact-dir", artifactDir,
		"--commit", "2222222",
		"--keep-recent", "1",
		"--output-dir", reportDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.firmware.artifact_prune_plan.summary.v1"`,
		`"delete_allowed": false`,
		`"keep_count": 1`,
		`"prune_candidate_count": 1`,
		`"manual_review_count": 0`,
		`"report_path":`,
		"firmware artifact prune plan ok (no files deleted)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), oldArtifact) {
		t.Fatalf("stdout should be summary-only when --output-dir is set: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(reportDir, "a21-firmware-artifact-prune-plan-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %v, want one prune-plan report", matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"schema_version": "a21.firmware.artifact_prune_plan.v1"`,
		`"reason": "current"`,
		`"reason": "older_than_keep_recent"`,
		oldArtifact,
		currentArtifact,
	} {
		if !strings.Contains(string(reportData), want) {
			t.Fatalf("report missing %q: %s", want, string(reportData))
		}
	}
	for _, path := range []string{oldArtifact, currentArtifact} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("artifact %q was removed by prune plan: %v", path, err)
		}
	}
}

func TestRunFirmwareUploadCheckRequiresExplicitPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--port requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckHelpShowsRequiredCommit(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"firmware-upload-check", "--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	for _, want := range []string{"--artifact", "--port", "--commit"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareUploadCheckRequiresExpectedCommit(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--commit requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckAcceptsExplicitDevicePort(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware upload dry-run guard ok (no flash performed)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	for _, want := range []string{
		`"guard_id": "a21.firmware.upload_guard.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"port_usage"`,
		`"exists": true`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareUploadCheckRejectsUnexpectedCommit(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "1234567",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "commit") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsAmbiguousPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "auto",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "explicit serial device path") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsNonSerialDevPath(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/null",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "serial device path") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsNonUSBMacSerialPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.Bluetooth-Incoming-Port",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "USB serial device path") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsBusyPort(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: true, Detail: "p1234 cpio"}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "already in use") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsMissingPort(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "does not exist") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsLegacyArtifactPathWithoutEchoingPath(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	legacyArtifact := filepath.Join(dir, "x21-artifacts", "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", legacyArtifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-artifacts") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareDeviceCheckAcceptsMatchingGatewayReport(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": `+fmt.Sprint(time.Now().UnixMilli())+`
    }
  ]
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		"firmware device identity guard ok (no flash performed)",
		`"guard_id": "a21.firmware.device_identity_guard.v1"`,
		`"device_identity_confirmed": true`,
		`"flash_allowed": false`,
		`"expected_device_id": "stackchan-001"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareDeviceCheckRejectsLegacyReportPathWithoutEchoingPath(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	legacyReport := filepath.Join(dir, "x21-reports", "devices.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", legacyReport,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-reports") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareDeviceCheckRequiresFreshnessGuard(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": `+fmt.Sprint(time.Now().UnixMilli())+`
    }
  ]
}`)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--max-device-age-ms requires a value") {
		t.Fatalf("stderr = %q, want max age usage error", stderr.String())
	}
}

func TestRunFirmwareDeviceCheckRequiresDeviceID(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{"devices":[]}`)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", report,
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--device-id requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareDeviceCheckRejectsStaleGatewayReport(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": 1000
    }
  ]
}`)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "5000",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "stale") {
		t.Fatalf("stderr = %q, want stale", stderr.String())
	}
}

func TestRunFirmwareDeviceCheckRejectsStaleConnectionStatus(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "stale",
      "device_age_ms": 300001,
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": 1780000000000
    }
  ]
}`)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "connection_status") || !strings.Contains(stderr.String(), "stale") {
		t.Fatalf("stderr = %q, want connection_status stale", stderr.String())
	}
}

func TestRunFirmwareFlashPlanBuildsNoFlashReceipt(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": `+fmt.Sprint(time.Now().UnixMilli())+`
    }
  ]
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		"firmware flash plan guard ok (no flash performed)",
		`"guard_id": "a21.firmware.flash_plan_guard.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"port": "/dev/cu.usbmodemA21"`,
		`"device_id": "stackchan-001"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareFlashPlanRejectsLegacyInputPathWithoutEchoingPath(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	legacyReport := filepath.Join(dir, "v21-reports", "devices.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--device-report", legacyReport,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "v21") || strings.Contains(strings.ToLower(stderr.String()), "v21-reports") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareFlashPlanRequiresFreshnessGuard(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": `+fmt.Sprint(time.Now().UnixMilli())+`
    }
  ]
}`)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--max-device-age-ms requires a value") {
		t.Fatalf("stderr = %q, want max device age usage error", stderr.String())
	}
}

func TestRunFirmwareFlashPlanRejectsActivePlaybackDevice(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	reportPayload := map[string]any{
		"schema_version": "a21.gateway.devices.v1",
		"service":        "a21-gateway",
		"devices": []map[string]any{{
			"device_id":          "stackchan-001",
			"identity_status":    "ok",
			"connection_status":  "online",
			"current_expression": "speaking",
			"playback_stream_id": "a21-stream-active",
			"firmware": map[string]string{
				"id":      "a21-stackchan",
				"version": "0.1.0",
				"board":   "m5stack-cores3",
				"commit":  "abcdef1",
			},
			"last_seen_ms": time.Now().UnixMilli(),
		}},
	}
	reportBytes, err := json.Marshal(reportPayload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(report, reportBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "active playback") {
		t.Fatalf("stderr = %q, want active playback", stderr.String())
	}
}

func TestRunFirmwareFlashPlanWritesReportWhenOutputDirProvided(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": `+fmt.Sprint(time.Now().UnixMilli())+`
    }
  ]
}`)
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"report_path"`) {
		t.Fatalf("stdout missing report_path: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-firmware-flash-plan-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("flash plan report files = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, want := range []string{
		`"guard_id": "a21.firmware.flash_plan_guard.v1"`,
		`"generated_at_ms"`,
		`"flash_allowed": false`,
		`"report_path"`,
		`"device_id": "stackchan-001"`,
		`"port": "/dev/cu.usbmodemA21"`,
	} {
		if !strings.Contains(reportJSON, want) {
			t.Fatalf("flash plan report missing %q: %s", want, reportJSON)
		}
	}
	if strings.Contains(reportJSON, `"flash_allowed": true`) {
		t.Fatalf("flash plan report unexpectedly allows flash: %s", reportJSON)
	}
}

func TestRunFirmwareBootstrapFlashPlanBuildsNoFlashReceipt(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	buildDir, coreDir := writeTestBootstrapFlashImages(t, dir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-bootstrap-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		"firmware bootstrap flash plan ok (no flash performed)",
		`"guard_id": "a21.firmware.bootstrap_flash_plan.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"next_required_confirmation": "firmware-bootstrap-flash-execute_with_confirmation_token"`,
		`"port": "/dev/cu.usbmodemA21"`,
		`"offset": "0x0000"`,
		`"offset": "0x10000"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareBootstrapFlashPlanRejectsLegacyInputPathWithoutEchoingPath(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "v21-artifacts", "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	buildDir, coreDir := writeTestBootstrapFlashImages(t, dir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-bootstrap-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "v21-artifacts") || strings.Contains(strings.ToLower(stderr.String()), "v21-artifacts") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareBootstrapFlashExecuteRequiresConfirmationToken(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	buildDir, coreDir := writeTestBootstrapFlashImages(t, dir)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-bootstrap-flash-execute",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--confirm WRITE_A21_STACKCHAN_FIRMWARE is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareBootstrapFlashExecuteRunsEsptoolCommandWithPlan(t *testing.T) {
	allowA21ControlGuardForTest(t)
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runFirmwareBootstrapFlashCommand
	var command []string
	runFirmwareBootstrapFlashCommand = func(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
		command = append([]string(nil), args...)
		_, _ = fmt.Fprintln(stdout, "stub esptool ok")
		return nil
	}
	defer func() {
		runFirmwareBootstrapFlashCommand = originalRunner
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	buildDir, coreDir := writeTestBootstrapFlashImages(t, dir)
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-bootstrap-flash-execute",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
		"--confirm", "WRITE_A21_STACKCHAN_FIRMWARE",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		"stub esptool ok",
		"firmware bootstrap flash executed",
		`"schema_version": "a21.firmware.bootstrap_flash_execution.v1"`,
		`"flash_executed": true`,
		`"control_guard"`,
		`"dry_run": false`,
		`"port": "/dev/cu.usbmodemA21"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	commandText := strings.Join(command, " ")
	for _, want := range []string{"esptool.py", "--chip esp32s3", "--port /dev/cu.usbmodemA21", "write_flash", "0x0000", "0x8000", "0xe000", "0x10000"} {
		if !strings.Contains(commandText, want) {
			t.Fatalf("command missing %q: %v", want, command)
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-firmware-bootstrap-flash-execution-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %v, want one bootstrap execution report", matches)
	}
}

func TestRunFirmwareMicProbeFlashPlanWritesNoFlashReport(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalSourceDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalSourceDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	buildDir, coreDir, artifact := writeTestMicProbeFlashImages(t, dir, "abcdef1")
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-mic-probe-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"guard_id": "a21.firmware.mic_probe_flash_plan.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"platformio_env": "a21_stackchan_cores3_mic_probe"`,
		"firmware mic probe flash plan ok",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-firmware-mic-probe-flash-plan-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("plan reports = %d, want 1", len(matches))
	}
}

func TestRunFirmwareMicProbeFlashExecuteRequiresConfirmationToken(t *testing.T) {
	originalSourceDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalSourceDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	buildDir, coreDir, artifact := writeTestMicProbeFlashImages(t, dir, "abcdef1")

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-mic-probe-flash-execute",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--confirm WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareMicProbeFlashExecuteRunsEsptoolCommandWithPlan(t *testing.T) {
	allowA21ControlGuardForTest(t)
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalSourceDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalSourceDetector
	}()
	originalRunner := runFirmwareBootstrapFlashCommand
	var command []string
	runFirmwareBootstrapFlashCommand = func(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
		command = append([]string(nil), args...)
		_, _ = fmt.Fprintln(stdout, "stub esptool ok")
		return nil
	}
	defer func() {
		runFirmwareBootstrapFlashCommand = originalRunner
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	buildDir, coreDir, artifact := writeTestMicProbeFlashImages(t, dir, "abcdef1")
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-mic-probe-flash-execute",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
		"--confirm", "WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		"stub esptool ok",
		"firmware mic probe flash executed",
		`"schema_version": "a21.firmware.mic_probe_flash_execution.v1"`,
		`"flash_executed": true`,
		`"control_guard"`,
		`"platformio_env": "a21_stackchan_cores3_mic_probe"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	commandText := strings.Join(command, " ")
	for _, want := range []string{"esptool.py", "--chip esp32s3", "--port /dev/cu.usbmodemA21", "write_flash", "0x0000", "0x8000", "0xe000", "0x10000"} {
		if !strings.Contains(commandText, want) {
			t.Fatalf("command missing %q: %v", want, command)
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-firmware-mic-probe-flash-execution-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("execution reports = %d, want 1", len(matches))
	}
}

func TestRunFirmwareIMUProbeFlashPlanWritesNoFlashReport(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalSourceDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalSourceDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	buildDir, coreDir, artifact := writeTestIMUProbeFlashImages(t, dir, "abcdef1")
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-imu-probe-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"guard_id": "a21.firmware.imu_probe_flash_plan.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"platformio_env": "a21_stackchan_cores3_imu_probe"`,
		"firmware IMU probe flash plan ok",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-firmware-imu-probe-flash-plan-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("plan reports = %d, want 1", len(matches))
	}
}

func TestRunFirmwareIMUProbeFlashExecuteRequiresConfirmationToken(t *testing.T) {
	originalSourceDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalSourceDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	buildDir, coreDir, artifact := writeTestIMUProbeFlashImages(t, dir, "abcdef1")

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-imu-probe-flash-execute",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--confirm WRITE_A21_STACKCHAN_IMU_PROBE_FIRMWARE is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareSensorProbeFlashPlanWritesNoFlashReport(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalSourceDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalSourceDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	buildDir, coreDir, artifact := writeTestSensorProbeFlashImages(t, dir, "abcdef1")
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-sensor-probe-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"guard_id": "a21.firmware.sensor_probe_flash_plan.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"platformio_env": "a21_stackchan_cores3_sensor_probe"`,
		"firmware sensor probe flash plan ok",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-firmware-sensor-probe-flash-plan-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("plan reports = %d, want 1", len(matches))
	}
}

func TestRunFirmwareSensorProbeFlashExecuteRequiresConfirmationToken(t *testing.T) {
	originalSourceDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalSourceDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	buildDir, coreDir, artifact := writeTestSensorProbeFlashImages(t, dir, "abcdef1")

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-sensor-probe-flash-execute",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--confirm WRITE_A21_STACKCHAN_SENSOR_PROBE_FIRMWARE is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunStackChanIMUProbeAcceptanceConfirmsReadOnlySamples(t *testing.T) {
	after := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/devices":
			if after {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"imu":"diagnostic_probe_m5unified_imu","screen":"available","rgb":"available"},"runtime_echo":{"imu_available":"1","imu_samples":"27","imu_read_errors":"0","imu_accel_mg_x":"5","imu_accel_mg_y":"996","imu_accel_mg_z":"80","imu_gyro_mdps_x":"0","imu_gyro_mdps_y":"0","imu_gyro_mdps_z":"0","imu_posture":"upright"},"identity_status":"ok","connection_status":"online","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			after = true
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"imu":"diagnostic_probe_m5unified_imu","screen":"available","rgb":"available"},"runtime_echo":{"imu_available":"1","imu_samples":"10","imu_read_errors":"0","imu_accel_mg_x":"4","imu_accel_mg_y":"997","imu_accel_mg_z":"75","imu_gyro_mdps_x":"0","imu_gyro_mdps_y":"0","imu_gyro_mdps_z":"0","imu_posture":"upright"},"identity_status":"ok","connection_status":"online","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		default:
			t.Fatalf("path = %q, want /v1/devices", r.URL.Path)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-imu-probe-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--min-samples", "10",
		"--min-accel-total-mg", "500",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_imu_probe_acceptance.v1"`,
		`"imu_probe_acceptance_status": "confirmed"`,
		`"imu": "diagnostic_probe_m5unified_imu"`,
		`"imu_samples_delta": 17`,
		`"imu_posture": "upright"`,
		"stackchan IMU probe acceptance ok",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-imu-probe-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %v, want one IMU probe report", matches)
	}
}

func TestRunStackChanSensorProbeAcceptanceConfirmsReadOnlyTelemetry(t *testing.T) {
	after := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/devices":
			if after {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"ambient_light":"diagnostic_probe_ltr553_ambient_light","proximity":"diagnostic_probe_ltr553_proximity","battery":"diagnostic_probe_ina226_battery","screen":"available","rgb":"available"},"runtime_echo":{"sensor_available":"1","sensor_samples":"31","sensor_read_errors":"0","ambient_light_raw":"220","proximity_raw":"7","battery_mv":"4012","battery_ma":"-42"},"identity_status":"ok","connection_status":"online","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			after = true
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"ambient_light":"diagnostic_probe_ltr553_ambient_light","proximity":"diagnostic_probe_ltr553_proximity","battery":"diagnostic_probe_ina226_battery","screen":"available","rgb":"available"},"runtime_echo":{"sensor_available":"1","sensor_samples":"12","sensor_read_errors":"0","ambient_light_raw":"180","proximity_raw":"3","battery_mv":"4008","battery_ma":"-40"},"identity_status":"ok","connection_status":"online","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		default:
			t.Fatalf("path = %q, want /v1/devices", r.URL.Path)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-sensor-probe-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--min-samples", "10",
		"--min-battery-mv", "3000",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_sensor_probe_acceptance.v1"`,
		`"sensor_probe_acceptance_status": "confirmed"`,
		`"ambient_light": "diagnostic_probe_ltr553_ambient_light"`,
		`"proximity": "diagnostic_probe_ltr553_proximity"`,
		`"battery": "diagnostic_probe_ina226_battery"`,
		`"sensor_samples_delta": 19`,
		`"ambient_light_raw": 220`,
		`"proximity_raw": 7`,
		`"battery_mv": 4012`,
		"stackchan sensor probe acceptance ok",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-sensor-probe-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %v, want one sensor probe report", matches)
	}
}

func TestRunOfficePreflightBuildsNoFlashReport(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.usbmodemA21",
			USBModem: true,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
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
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": ` + fmt.Sprint(time.Now().UnixMilli()) + `
    }
  ]
}`))
	}))
	defer server.Close()

	outputDir := filepath.Join(dir, "reports")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-preflight",
		"--manifest", manifest,
		"--artifact-dir", dir,
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.office_preflight.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"ready_for_flash_plan": true`,
		`"next_required_confirmation": "firmware-flash-plan_with_explicit_usb_port"`,
		`"gateway_schema_version": "a21.gateway.devices.v1"`,
		`"gateway_service": "a21-gateway"`,
		`"device_report_path"`,
		`"report_path"`,
		`"/dev/cu.usbmodemA21"`,
		"office preflight ok (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	reportMatches, err := filepath.Glob(filepath.Join(outputDir, "a21-office-preflight-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(reportMatches) != 1 {
		t.Fatalf("office preflight reports = %d, want 1: %v", len(reportMatches), reportMatches)
	}
	deviceMatches, err := filepath.Glob(filepath.Join(outputDir, "a21-devices-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(deviceMatches) != 1 {
		t.Fatalf("device reports = %d, want 1: %v", len(deviceMatches), deviceMatches)
	}
}

func TestRunOfficePreflightFailsWithoutUSBSerialCandidate(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.Bluetooth-Incoming-Port",
			USBModem: false,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": ` + fmt.Sprint(time.Now().UnixMilli()) + `
    }
  ]
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-preflight",
		"--manifest", manifest,
		"--artifact-dir", dir,
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	for _, want := range []string{
		`"ready_for_flash_plan": false`,
		`"flash_allowed": false`,
		`"code": "usb_serial_candidate_missing"`,
		"office preflight failed",
	} {
		if !strings.Contains(stdout.String()+stderr.String(), want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
	if strings.Contains(stdout.String()+stderr.String(), "x21") || strings.Contains(stdout.String()+stderr.String(), "v21") {
		t.Fatalf("office preflight leaked legacy identity stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunOfficePreflightRejectsActivePlaybackDevice(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.usbmodemA21",
			USBModem: true,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_expression": "speaking",
      "playback_stream_id": "a21-stream-active",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": ` + fmt.Sprint(time.Now().UnixMilli()) + `
    }
  ]
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-preflight",
		"--manifest", manifest,
		"--artifact-dir", dir,
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	for _, want := range []string{
		`"ready_for_flash_plan": false`,
		`"flash_allowed": false`,
		`"code": "device_not_quiescent"`,
		"active playback",
		"office preflight failed",
	} {
		if !strings.Contains(stdout.String()+stderr.String(), want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
}

func TestRunOfficePreflightRejectsLegacyArtifactDirWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-preflight",
		"--artifact-dir", filepath.Join(t.TempDir(), "x21-artifacts"),
		"--gateway-url", "http://127.0.0.1:21080",
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy artifact dir rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-artifacts") {
		t.Fatalf("legacy artifact dir leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunOfficeHandoffWritesNoFlashManifest(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.usbmodemA21",
			USBModem: true,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifactDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	currentArtifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, currentArtifact, []byte("firmware"))
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-handoff",
		"--manifest", manifest,
		"--artifact-dir", artifactDir,
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.office_handoff.v1"`,
		`"flash_allowed": false`,
		`"delete_allowed": false`,
		`"physical_acceptance_required": true`,
		`"office_preflight_required": true`,
		`"current_artifact_path": "` + currentArtifact + `"`,
		`"usb_serial_candidate_count": 1`,
		"office handoff manifest ok (no flash, no delete)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-office-handoff-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("handoff reports = %d, want 1: %v", len(matches), matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(reportData), currentArtifact) {
		t.Fatalf("handoff report missing artifact path: %s", string(reportData))
	}
}

func TestRunOfficeHandoffRejectsLegacyArtifactDirWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-handoff",
		"--artifact-dir", filepath.Join(t.TempDir(), "x21-artifacts"),
		"--commit", "abcdef1",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy artifact dir rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-artifacts") {
		t.Fatalf("legacy artifact dir leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunOfficeAcceptanceAcceptsMatchingHandoffAndPreflight(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.usbmodemA21",
			USBModem: true,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifactDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "idle",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": ` + fmt.Sprint(time.Now().UnixMilli()) + `
    }
  ]
}`))
	}))
	defer server.Close()
	outputDir := filepath.Join(dir, "reports")

	var handoffOut bytes.Buffer
	if code := Run([]string{
		"office-handoff",
		"--manifest", manifest,
		"--artifact-dir", artifactDir,
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &handoffOut, &bytes.Buffer{}); code != 0 {
		t.Fatalf("office-handoff code = %d: %s", code, handoffOut.String())
	}
	var preflightOut bytes.Buffer
	var preflightErr bytes.Buffer
	if code := Run([]string{
		"office-preflight",
		"--manifest", manifest,
		"--artifact-dir", artifactDir,
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", outputDir,
	}, &preflightOut, &preflightErr); code != 0 {
		t.Fatalf("office-preflight code = %d: stdout=%s stderr=%s", code, preflightOut.String(), preflightErr.String())
	}
	handoffPath := newestGlob(t, filepath.Join(outputDir, "a21-office-handoff-*.json"))
	preflightPath := newestGlob(t, filepath.Join(outputDir, "a21-office-preflight-*.json"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-acceptance",
		"--handoff", handoffPath,
		"--office-preflight", preflightPath,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("office-acceptance code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.office_acceptance.v1"`,
		`"physical_acceptance_status": "ready_for_physical_acceptance"`,
		`"flash_allowed": false`,
		`"delete_allowed": false`,
		`"handoff_report_path": "` + handoffPath + `"`,
		`"office_preflight_report_path": "` + preflightPath + `"`,
		`"artifact_path": "` + artifact + `"`,
		`"device_id": "stackchan-001"`,
		"office acceptance gate ok (no flash, no delete)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	acceptancePath := newestGlob(t, filepath.Join(outputDir, "a21-office-acceptance-*.json"))
	data, err := os.ReadFile(acceptancePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"physical_acceptance_status": "ready_for_physical_acceptance"`) {
		t.Fatalf("acceptance report missing ready status: %s", string(data))
	}
}

func TestRunOfficeAcceptanceRejectsMismatchedReports(t *testing.T) {
	dir := t.TempDir()
	handoffPath := filepath.Join(dir, "a21-office-handoff.json")
	preflightPath := filepath.Join(dir, "a21-office-preflight.json")
	if err := os.WriteFile(handoffPath, []byte(`{
  "schema_version": "a21.office_handoff.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "commit": "abcdef1",
  "current_artifact_path": "firmware/artifacts/a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(preflightPath, []byte(`{
  "schema_version": "a21.office_preflight.v1",
  "dry_run": true,
  "flash_allowed": false,
  "ready_for_flash_plan": true,
  "device_id": "stackchan-001",
  "commit": "2222222",
  "artifact": {
    "artifact_path": "firmware/artifacts/a21-stackchan-0.1.0-m5stack-cores3-2222222-20260530-004500.bin",
    "sha256": "abc"
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-acceptance",
		"--handoff", handoffPath,
		"--office-preflight", preflightPath,
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"physical_acceptance_status": "blocked"`,
		`"code": "commit_mismatch"`,
		`"code": "artifact_mismatch"`,
		"office acceptance gate failed",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunOfficeAcceptanceRejectsLegacyReportPathWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-acceptance",
		"--handoff", filepath.Join(t.TempDir(), "x21-office-handoff.json"),
		"--office-preflight", filepath.Join(t.TempDir(), "a21-office-preflight.json"),
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-office") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunStackChanIdentityAcceptanceConfirmsFreshDevice(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.usbmodemA21",
			USBModem: true,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifactDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	artifactSHA := readTestSHA256(t, artifact+".sha256")
	officeAcceptancePath := filepath.Join(dir, "a21-office-acceptance.json")
	if err := os.WriteFile(officeAcceptancePath, []byte(`{
  "schema_version": "a21.office_acceptance.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "physical_acceptance_status": "ready_for_physical_acceptance",
  "commit": "abcdef1",
  "artifact_path": "`+artifact+`",
  "artifact_sha256": "`+artifactSHA+`",
  "device_id": "stackchan-001"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
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
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "idle",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": ` + fmt.Sprint(time.Now().UnixMilli()) + `
    }
  ]
}`))
	}))
	defer server.Close()

	outputDir := filepath.Join(dir, "reports")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-identity-acceptance",
		"--office-acceptance", officeAcceptancePath,
		"--manifest", manifest,
		"--artifact-dir", artifactDir,
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_identity_acceptance.v1"`,
		`"hardware_acceptance_scope": "identity_only"`,
		`"identity_acceptance_status": "identity_confirmed"`,
		`"flash_allowed": false`,
		`"office_acceptance_report_path": "` + officeAcceptancePath + `"`,
		`"artifact_path": "` + artifact + `"`,
		`"artifact_sha256": "` + artifactSHA + `"`,
		`"device_id": "stackchan-001"`,
		`"device_identity_confirmed": true`,
		`"/dev/cu.usbmodemA21"`,
		"stackchan identity acceptance ok (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-identity-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("identity acceptance reports = %d, want 1: %v", len(matches), matches)
	}
	deviceReports, err := filepath.Glob(filepath.Join(outputDir, "a21-devices-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(deviceReports) != 1 {
		t.Fatalf("device reports = %d, want 1: %v", len(deviceReports), deviceReports)
	}
}

func TestRunStackChanIdentityAcceptanceRejectsBlockedOfficeAcceptance(t *testing.T) {
	dir := t.TempDir()
	officeAcceptancePath := filepath.Join(dir, "a21-office-acceptance.json")
	if err := os.WriteFile(officeAcceptancePath, []byte(`{
  "schema_version": "a21.office_acceptance.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "physical_acceptance_status": "blocked",
  "commit": "abcdef1",
  "device_id": "stackchan-001"
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-identity-acceptance",
		"--office-acceptance", officeAcceptancePath,
		"--gateway-url", "http://127.0.0.1:1",
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_identity_acceptance.v1"`,
		`"identity_acceptance_status": "blocked"`,
		`"flash_allowed": false`,
		`"code": "office_acceptance_not_ready"`,
		"stackchan identity acceptance failed",
	} {
		if !strings.Contains(stdout.String()+stderr.String(), want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
}

func TestRunStackChanIdentityAcceptanceRejectsLegacyReportPathWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-identity-acceptance",
		"--office-acceptance", filepath.Join(t.TempDir(), "x21-office-acceptance.json"),
		"--gateway-url", "http://127.0.0.1:21080",
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-office") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunStackChanCapabilityAcceptanceConfirmsPhysicalEvidence(t *testing.T) {
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "artifacts", "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	artifactSHA := strings.Repeat("a", 64)
	identityPath := filepath.Join(dir, "a21-stackchan-identity-acceptance.json")
	if err := os.WriteFile(identityPath, []byte(`{
  "schema_version": "a21.stackchan_identity_acceptance.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "hardware_acceptance_scope": "identity_only",
  "identity_acceptance_status": "identity_confirmed",
  "device_id": "stackchan-001",
  "commit": "abcdef1",
  "artifact_path": "`+artifactPath+`",
  "artifact_sha256": "`+artifactSHA+`",
  "device_identity": {
    "device_identity_confirmed": true,
    "device": {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "idle",
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
      "last_seen_ms": 1780000000000
    }
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(dir, "a21-stackchan-physical-evidence.json")
	if err := os.WriteFile(evidencePath, []byte(`{
  "schema_version": "a21.stackchan_physical_evidence.v1",
  "device_id": "stackchan-001",
  "commit": "abcdef1",
  "artifact_sha256": "`+artifactSHA+`",
  "observations": [
    {"capability": "microphone", "status": "passed", "evidence_type": "gateway_audio_frame", "observed_at_ms": 1780000001000},
    {"capability": "speaker", "status": "passed", "evidence_type": "audible_playback", "observed_at_ms": 1780000002000},
    {"capability": "screen", "status": "passed", "evidence_type": "operator_visible_state", "observed_at_ms": 1780000003000},
    {"capability": "screen_touch", "status": "passed", "evidence_type": "touch_event", "observed_at_ms": 1780000004000},
    {"capability": "top_touch", "status": "passed", "evidence_type": "touch_event", "observed_at_ms": 1780000005000},
    {"capability": "servo_y", "status": "passed", "evidence_type": "servo_clamped_motion", "observed_at_ms": 1780000006000},
    {"capability": "rgb", "status": "passed", "evidence_type": "operator_visible_state", "observed_at_ms": 1780000007000}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(dir, "reports")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-capability-acceptance",
		"--identity-acceptance", identityPath,
		"--evidence", evidencePath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_capability_acceptance.v1"`,
		`"hardware_acceptance_scope": "physical_capability_evidence"`,
		`"capability_acceptance_status": "confirmed"`,
		`"flash_allowed": false`,
		`"identity_acceptance_report_path": "` + identityPath + `"`,
		`"evidence_report_path": "` + evidencePath + `"`,
		`"device_id": "stackchan-001"`,
		`"artifact_sha256": "` + artifactSHA + `"`,
		`"capability": "microphone"`,
		`"capability": "servo_y"`,
		"stackchan capability acceptance ok (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-capability-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("capability acceptance reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanCapabilityAcceptanceBlocksMissingCapabilityEvidence(t *testing.T) {
	dir := t.TempDir()
	identityPath := filepath.Join(dir, "a21-stackchan-identity-acceptance.json")
	if err := os.WriteFile(identityPath, []byte(`{
  "schema_version": "a21.stackchan_identity_acceptance.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "hardware_acceptance_scope": "identity_only",
  "identity_acceptance_status": "identity_confirmed",
  "device_id": "stackchan-001",
  "commit": "abcdef1",
  "artifact_sha256": "`+strings.Repeat("b", 64)+`",
  "device_identity": {
    "device_identity_confirmed": true,
    "device": {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {"id": "a21-stackchan", "version": "0.1.0", "board": "m5stack-cores3", "commit": "abcdef1"},
      "capabilities": {
        "microphone": "available",
        "speaker": "available",
        "screen": "available",
        "screen_touch": "available",
        "top_touch": "available",
        "servo_y": "available",
        "rgb": "available"
      }
    }
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(dir, "a21-stackchan-physical-evidence.json")
	if err := os.WriteFile(evidencePath, []byte(`{
  "schema_version": "a21.stackchan_physical_evidence.v1",
  "device_id": "stackchan-001",
  "commit": "abcdef1",
  "artifact_sha256": "`+strings.Repeat("b", 64)+`",
  "observations": [
    {"capability": "microphone", "status": "passed", "evidence_type": "gateway_audio_frame", "observed_at_ms": 1780000001000}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-capability-acceptance",
		"--identity-acceptance", identityPath,
		"--evidence", evidencePath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_capability_acceptance.v1"`,
		`"capability_acceptance_status": "blocked"`,
		`"code": "capability_evidence_missing"`,
		`"message": "physical evidence missing for required StackChan capability \"speaker\""`,
		"stackchan capability acceptance failed",
	} {
		if !strings.Contains(stdout.String()+stderr.String(), want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
}

func TestRunStackChanCapabilityAcceptanceRejectsLegacyEvidencePathWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-capability-acceptance",
		"--identity-acceptance", filepath.Join(t.TempDir(), "a21-stackchan-identity-acceptance.json"),
		"--evidence", filepath.Join(t.TempDir(), "x21-physical-evidence.json"),
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-physical") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunStackChanMicProbeAcceptanceConfirmsDiagnosticMicPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"listening","servo_y":"38deg","rgb":"#003010","mic_frames_captured":"321","audio_ws_sent_audio_frames":"321","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"1052","mic_last_nonzero_samples":"320"},"identity_status":"ok","connection_status":"online","current_expression":"listening","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, strings.Join([]string{
				"a21_audio_frame_total 374",
				"a21_audio_ingress_frames_total 374",
				"a21_audio_ingress_rms 0.01296457627255574",
				"a21_audio_playback_chunk_total 0",
				`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 99`,
			}, "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-mic-probe-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--min-frames", "300",
		"--min-abs-peak", "100",
		"--min-nonzero-samples", "300",
		"--min-gateway-rms", "0.001",
		"--min-vad-speech", "1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_mic_probe_acceptance.v1"`,
		`"mic_probe_acceptance_status": "confirmed"`,
		`"hardware_acceptance_scope": "diagnostic_microphone_only"`,
		`"production_capability_promoted": false`,
		`"microphone": "diagnostic_probe_m5unified_i2s_capture"`,
		`"mic_frames_captured": 321`,
		`"mic_last_abs_peak": 1052`,
		`"mic_last_nonzero_samples": 320`,
		`"gateway_audio_frame_total": 374`,
		`"gateway_audio_ingress_rms": 0.01296457627255574`,
		`"gateway_vad_speech_total": 99`,
		"stackchan mic probe acceptance ok (diagnostic only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-mic-probe-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("mic probe acceptance reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanMicProbeAcceptanceRunsWindowedProbeWithDeltas(t *testing.T) {
	probeStarted := false
	probeStopped := false
	var probeControlSeen bool
	var idleControlSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				DeviceID       string `json:"device_id"`
				State          string `json:"state"`
				Mode           string `json:"mode"`
				Text           string `json:"text"`
				TraceID        string `json:"trace_id"`
				SessionID      string `json:"session_id"`
				AudioProbeOnly bool   `json:"audio_probe_only"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.DeviceID != "stackchan-001" {
				t.Fatalf("device_id = %q", payload.DeviceID)
			}
			if payload.State == "listening" {
				probeControlSeen = true
				probeStarted = true
				if !payload.AudioProbeOnly {
					t.Fatalf("probe control did not set audio_probe_only: %+v", payload)
				}
				if payload.TraceID == "" || payload.SessionID == "" {
					t.Fatalf("probe control missing trace/session: %+v", payload)
				}
			}
			if payload.State == "idle" {
				idleControlSeen = true
				probeStopped = true
				if payload.AudioProbeOnly {
					t.Fatalf("idle control should clear audio_probe_only: %+v", payload)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":%q,"session_id":%q,"device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`, payload.TraceID, payload.SessionID)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if probeStarted {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","mic_frames_captured":"350","audio_ws_sent_audio_frames":"350","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"900","mic_last_nonzero_samples":"319"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","mic_frames_captured":"40","audio_ws_sent_audio_frames":"40","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"20","mic_last_nonzero_samples":"20"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if probeStarted {
				fmt.Fprint(w, strings.Join([]string{
					"a21_audio_frame_total 430",
					"a21_audio_ingress_frames_total 430",
					"a21_audio_ingress_rms 0.0042",
					"a21_audio_playback_chunk_total 7",
					`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 11`,
				}, "\n"))
				return
			}
			fmt.Fprint(w, strings.Join([]string{
				"a21_audio_frame_total 100",
				"a21_audio_ingress_frames_total 100",
				"a21_audio_ingress_rms 0.0001",
				"a21_audio_playback_chunk_total 7",
				`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 1`,
			}, "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-mic-probe-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--min-frames", "300",
		"--min-abs-peak", "100",
		"--min-nonzero-samples", "300",
		"--min-gateway-rms", "0.001",
		"--min-vad-speech", "1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !probeControlSeen || !idleControlSeen || !probeStopped {
		t.Fatalf("probeControlSeen=%v idleControlSeen=%v probeStopped=%v", probeControlSeen, idleControlSeen, probeStopped)
	}
	for _, want := range []string{
		`"window_ms": 1`,
		`"mic_frames_captured_delta": 310`,
		`"audio_ws_sent_audio_frames_delta": 310`,
		`"audio_ws_delivery_ratio": 1`,
		`"gateway_audio_frame_delta": 330`,
		`"gateway_audio_ingress_frames_delta": 330`,
		`"gateway_ingress_delivery_ratio": 1`,
		`"gateway_audio_playback_chunk_total": 7`,
		`"gateway_audio_playback_chunk_delta": 0`,
		`"gateway_vad_speech_delta": 10`,
		`"mic_probe_acceptance_status": "confirmed"`,
		"stackchan mic probe acceptance ok (diagnostic only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanMicProbeAcceptanceBlocksLowDeliveryRatio(t *testing.T) {
	probeStarted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				State string `json:"state"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.State == "listening" {
				probeStarted = true
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"trace_id":"a21-trace-mic-probe","session_id":"a21-session-mic-probe","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if probeStarted {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"listening","servo_y":"38deg","rgb":"#003010","mic_frames_captured":"350","audio_ws_sent_audio_frames":"200","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"900","mic_last_nonzero_samples":"319"},"identity_status":"ok","connection_status":"online","current_expression":"listening","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","mic_frames_captured":"40","audio_ws_sent_audio_frames":"40","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"20","mic_last_nonzero_samples":"20"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if probeStarted {
				fmt.Fprint(w, strings.Join([]string{
					"a21_audio_frame_total 170",
					"a21_audio_ingress_frames_total 170",
					"a21_audio_ingress_rms 0.0042",
					"a21_audio_playback_chunk_total 7",
					`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 11`,
				}, "\n"))
				return
			}
			fmt.Fprint(w, strings.Join([]string{
				"a21_audio_frame_total 100",
				"a21_audio_ingress_frames_total 100",
				"a21_audio_ingress_rms 0.0001",
				"a21_audio_playback_chunk_total 7",
				`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 1`,
			}, "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-mic-probe-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--min-frames", "50",
		"--min-delivery-ratio", "0.95",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"audio_ws_delivery_ratio": 0.516`,
		`"gateway_ingress_delivery_ratio": 0.438`,
		`"code": "audio_ws_delivery_ratio_below_threshold"`,
		`"code": "gateway_ingress_delivery_ratio_below_threshold"`,
		"stackchan mic probe acceptance blocked (diagnostic only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanMicProbeAcceptanceTreatsMissingSpeechSeriesAsZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","mic_frames_captured":"90","audio_ws_sent_audio_frames":"90","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"120","mic_last_nonzero_samples":"300"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, strings.Join([]string{
				"a21_audio_frame_total 90",
				"a21_audio_ingress_frames_total 90",
				"a21_audio_ingress_rms 0.0042",
				"a21_audio_playback_chunk_total 0",
			}, "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-mic-probe-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--min-frames", "90",
		"--min-vad-speech", "0",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"gateway_vad_speech_total": 0`,
		`"mic_probe_acceptance_status": "confirmed"`,
		"stackchan mic probe acceptance ok (diagnostic only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanHalfDuplexAcceptanceConfirmsMicTriggeredPlayback(t *testing.T) {
	probeStarted := false
	idleControlSeen := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				DeviceID                     string `json:"device_id"`
				State                        string `json:"state"`
				TraceID                      string `json:"trace_id"`
				SessionID                    string `json:"session_id"`
				AudioProbeOnly               bool   `json:"audio_probe_only"`
				MockPlaybackOnNextAudioFrame bool   `json:"mock_playback_on_next_audio_frame"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.DeviceID != "stackchan-001" {
				t.Fatalf("device_id = %q", payload.DeviceID)
			}
			if payload.State == "listening" {
				probeStarted = true
				if payload.AudioProbeOnly {
					t.Fatalf("half-duplex listening must allow Gateway playback: %+v", payload)
				}
				if !payload.MockPlaybackOnNextAudioFrame {
					t.Fatalf("half-duplex listening must arm one next-frame mock playback: %+v", payload)
				}
			}
			if payload.State == "idle" {
				idleControlSeen = true
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":%q,"session_id":%q,"device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`, payload.TraceID, payload.SessionID)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if probeStarted {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"speaking","servo_y":"48deg","rgb":"#002430","mic_frames_captured":"13","audio_ws_sent_audio_frames":"13","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"800","mic_last_nonzero_samples":"318","playback_buffer_total_chunks":"11","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"6","speaker_driver_errors":"0","speaker_last_stream_id":"a21-audio-stream-000001"},"identity_status":"ok","connection_status":"online","current_expression":"speaking","playback_stream_id":"a21-audio-stream-000001","last_session_id":"a21-session-half-duplex","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","mic_frames_captured":"10","audio_ws_sent_audio_frames":"10","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"20","mic_last_nonzero_samples":"20","playback_buffer_total_chunks":"10","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"5","speaker_driver_errors":"0","speaker_last_stream_id":"a21-old-stream"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-before","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if probeStarted {
				fmt.Fprint(w, strings.Join([]string{
					"a21_audio_frame_total 31",
					"a21_audio_ingress_frames_total 31",
					"a21_audio_ingress_rms 0.0042",
					"a21_audio_playback_chunk_total 9",
					`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 1`,
				}, "\n"))
				return
			}
			fmt.Fprint(w, strings.Join([]string{
				"a21_audio_frame_total 20",
				"a21_audio_ingress_frames_total 20",
				"a21_audio_ingress_rms 0.0001",
				"a21_audio_playback_chunk_total 8",
			}, "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-half-duplex-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--min-mic-frames", "1",
		"--min-playback-chunks", "1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !idleControlSeen {
		t.Fatalf("idle control not seen")
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_half_duplex_acceptance.v1"`,
		`"half_duplex_acceptance_status": "confirmed"`,
		`"mic_frames_captured_delta": 3`,
		`"audio_ws_sent_audio_frames_delta": 3`,
		`"gateway_audio_ingress_frames_delta": 11`,
		`"gateway_audio_playback_chunk_delta": 1`,
		`"playback_buffer_total_chunks_delta": 1`,
		`"speaker_frames_played_delta": 1`,
		`"audio_ws_delivery_ratio": 1`,
		"stackchan half-duplex acceptance ok (instrumented only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanSpeakerAcceptanceConfirmsInstrumentedDownlink(t *testing.T) {
	probeStarted := false
	probeStopped := false
	var speakingControlSeen bool
	var idleControlSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				DeviceID        string `json:"device_id"`
				State           string `json:"state"`
				Mode            string `json:"mode"`
				Text            string `json:"text"`
				TraceID         string `json:"trace_id"`
				SessionID       string `json:"session_id"`
				StreamID        string `json:"stream_id"`
				MockAudioChunks int    `json:"mock_audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.DeviceID != "stackchan-001" {
				t.Fatalf("device_id = %q", payload.DeviceID)
			}
			if payload.State == "speaking" {
				speakingControlSeen = true
				probeStarted = true
				if payload.MockAudioChunks != 8 {
					t.Fatalf("mock_audio_chunks = %d, want 8 padded playback frames", payload.MockAudioChunks)
				}
				if payload.StreamID != "a21-speaker-acceptance-stream" {
					t.Fatalf("stream_id = %q", payload.StreamID)
				}
				if payload.TraceID == "" || payload.SessionID == "" {
					t.Fatalf("speaker control missing trace/session: %+v", payload)
				}
			}
			if payload.State == "idle" {
				idleControlSeen = true
				probeStopped = true
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":%q,"session_id":%q,"device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`, payload.TraceID, payload.SessionID)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if probeStarted {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"speaking","servo_y":"48deg","rgb":"#002430","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"14","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"24","speaker_busy_ticks":"4","speaker_driver_errors":"0","speaker_last_stream_id":"a21-speaker-acceptance-stream"},"identity_status":"ok","connection_status":"online","current_expression":"speaking","playback_stream_id":"a21-speaker-acceptance-stream","last_session_id":"a21-session-speaker-acceptance","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"10","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"20","speaker_busy_ticks":"3","speaker_driver_errors":"0","speaker_last_stream_id":"a21-old-stream"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-before","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if probeStarted {
				fmt.Fprint(w, "a21_audio_playback_chunk_total 13\n")
				return
			}
			fmt.Fprint(w, "a21_audio_playback_chunk_total 9\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-speaker-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--mock-audio-chunks", "4",
		"--min-played-frames", "4",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !speakingControlSeen || !idleControlSeen || !probeStopped {
		t.Fatalf("speakingControlSeen=%v idleControlSeen=%v probeStopped=%v", speakingControlSeen, idleControlSeen, probeStopped)
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_speaker_acceptance.v1"`,
		`"speaker_acceptance_status": "confirmed"`,
		`"hardware_acceptance_scope": "instrumented_speaker_downlink"`,
		`"physical_sound_observed": false`,
		`"mock_audio_chunks": 4`,
		`"stream_id": "a21-speaker-acceptance-stream"`,
		`"playback_buffer_total_chunks_delta": 4`,
		`"speaker_frames_played_delta": 4`,
		`"gateway_audio_playback_chunk_delta": 4`,
		"stackchan speaker acceptance ok (instrumented only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-speaker-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("speaker acceptance reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanSpeakerAcceptanceBatchesAudibleProbe(t *testing.T) {
	var totalChunks int
	var speakingCalls int
	var idleControlSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				DeviceID        string `json:"device_id"`
				State           string `json:"state"`
				TraceID         string `json:"trace_id"`
				SessionID       string `json:"session_id"`
				StreamID        string `json:"stream_id"`
				MockAudioChunks int    `json:"mock_audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.DeviceID != "stackchan-001" {
				t.Fatalf("device_id = %q", payload.DeviceID)
			}
			if payload.State == "speaking" {
				speakingCalls++
				totalChunks += payload.MockAudioChunks
				if payload.MockAudioChunks != 8 {
					t.Fatalf("speaker batch mock_audio_chunks = %d, want 8 padded playback frames", payload.MockAudioChunks)
				}
				if payload.StreamID != "a21-speaker-acceptance-stream" {
					t.Fatalf("stream_id = %q", payload.StreamID)
				}
			}
			if payload.State == "idle" {
				idleControlSeen = true
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":%q,"session_id":%q,"device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`, payload.TraceID, payload.SessionID)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if totalChunks >= 50 {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"speaking","servo_y":"48deg","rgb":"#002430","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"60","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"70","speaker_busy_ticks":"4","speaker_driver_errors":"0","speaker_last_stream_id":"a21-speaker-acceptance-stream"},"identity_status":"ok","connection_status":"online","current_expression":"speaking","playback_stream_id":"a21-speaker-acceptance-stream","last_session_id":"a21-session-speaker-acceptance","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"10","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"20","speaker_busy_ticks":"3","speaker_driver_errors":"0","speaker_last_stream_id":"a21-old-stream"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-before","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if totalChunks >= 50 {
				fmt.Fprint(w, "a21_audio_playback_chunk_total 59\n")
				return
			}
			fmt.Fprint(w, "a21_audio_playback_chunk_total 9\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-speaker-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--mock-audio-chunks", "50",
		"--min-played-frames", "50",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if speakingCalls < 2 || totalChunks != 56 || !idleControlSeen {
		t.Fatalf("speakingCalls=%d totalChunks=%d idleControlSeen=%v", speakingCalls, totalChunks, idleControlSeen)
	}
	for _, want := range []string{
		`"mock_audio_chunks": 50`,
		`"expected_audio_duration_ms": 1000`,
		`"playback_buffer_total_chunks_delta": 50`,
		`"speaker_frames_played_delta": 50`,
		`"gateway_audio_playback_chunk_delta": 50`,
		"stackchan speaker acceptance ok (instrumented only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestDefaultStackChanSpeakerAcceptanceWindowLeavesPlaybackMargin(t *testing.T) {
	options := defaultStackChanSpeakerAcceptanceOptions()
	expectedAudioDurationMS := options.MockAudioChunks * stackChanSpeakerProbeChunkDurationMS
	if options.WindowMS < expectedAudioDurationMS+500 {
		t.Fatalf("default speaker window = %dms, want at least %dms for playback and runtime echo margin", options.WindowMS, expectedAudioDurationMS+500)
	}
}

func TestRunStackChanSpeakerAcceptanceBlocksBeforePlaybackOnFirmwareMismatch(t *testing.T) {
	controlCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			controlCalled = true
			http.Error(w, "control should not be called", http.StatusInternalServerError)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"older123"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"10","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"20","speaker_busy_ticks":"0","speaker_driver_errors":"0","speaker_last_stream_id":"a21-old-stream"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "a21_audio_playback_chunk_total 9\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-speaker-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "5a4936f9a993",
		"--window-ms", "1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code = 0, want blocked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if controlCalled {
		t.Fatalf("speaker control was called despite firmware commit mismatch")
	}
	for _, want := range []string{
		`"speaker_acceptance_status": "blocked"`,
		`"code": "firmware_commit_mismatch"`,
		"stackchan speaker acceptance blocked (instrumented only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanTouchAcceptancePassesTopTapWithGatewayTrace(t *testing.T) {
	var armed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			if r.Method != http.MethodPost {
				t.Fatalf("control method = %s", r.Method)
			}
			armed = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"trace_id":"a21-trace-touch-acceptance-top_tap","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		case "/v1/devices":
			lastEvent := ""
			lastSource := ""
			lastTrace := ""
			lastSeen := int64(1780000001000)
			if armed {
				lastEvent = "touch.top.tap"
				lastSource = "top_sensor"
				lastTrace = "a21-trace-device-000123"
				lastSeen = time.Now().UnixMilli()
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abc123"},"capabilities":{"screen":"available","screen_touch":"available","top_touch":"available","speaker":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_event":%q,"last_touch_source":%q,"last_trace_id":%q,"last_session_id":"a21-session-touch-acceptance","last_seen_ms":%d,"first_seen_ms":1780000000000}]}`, lastEvent, lastSource, lastTrace, lastSeen)
		case "/v1/traces":
			if r.URL.Query().Get("trace_id") != "a21-trace-device-000123" {
				t.Fatalf("trace id = %q", r.URL.Query().Get("trace_id"))
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":"a21-trace-device-000123","events":[{"name":"device.touch.top.tap.received","trace_id":"a21-trace-device-000123","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","at_ms":%d,"offset_ms":0}],"summary":{"event_count":1,"last_offset_ms":0}}`, time.Now().UnixMilli())
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-touch-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--case", "top_tap",
		"--window-ms", "1000",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_touch_acceptance.v1"`,
		`"touch_acceptance_status": "passed"`,
		`"expected_event": "touch.top.tap"`,
		"stackchan touch acceptance ok (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-touch-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("touch acceptance reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanTouchAcceptanceMarksDirectionalCaseAsAffordanceBoundWhenMissed(t *testing.T) {
	var armed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			armed = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"trace_id":"a21-trace-touch-acceptance-top_swipe_backward","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		case "/v1/devices":
			lastEvent := ""
			lastSource := ""
			lastTrace := ""
			lastSeen := int64(1780000001000)
			if armed {
				lastEvent = "touch.top.tap"
				lastSource = "top_sensor"
				lastTrace = "a21-trace-device-000124"
				lastSeen = time.Now().UnixMilli()
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abc123"},"capabilities":{"screen":"available","screen_touch":"available","top_touch":"available","speaker":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_event":%q,"last_touch_source":%q,"last_trace_id":%q,"last_session_id":"a21-session-touch-acceptance","last_seen_ms":%d,"first_seen_ms":1780000000000}]}`, lastEvent, lastSource, lastTrace, lastSeen)
		case "/v1/traces":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":"a21-trace-device-000124","events":[{"name":"device.touch.top.tap.received","trace_id":"a21-trace-device-000124","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","at_ms":%d,"offset_ms":0}],"summary":{"event_count":1,"last_offset_ms":0}}`, time.Now().UnixMilli())
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-touch-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--case", "top_swipe_backward",
		"--window-ms", "50",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"acceptance_scope": "guided_directional_touch"`,
		`"needs_affordance": true`,
		`"code": "physical_affordance_required"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanHardwareMainlineReportsOrderedPlannedTracks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef123456"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","servo_x":"planned_continuous_rotation_axis","rgb":"available","camera":"planned_core_s3_camera","imu":"planned_9_axis_imu","ambient_light":"planned_ambient_light_sensor","proximity":"planned_proximity_sensor","battery":"planned_550mah_battery","nfc":"planned_nfc","infrared":"planned_infrared_tx_rx"},"runtime_echo":{"screen":"idle","servo_y":"48deg","rgb":"#002430"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-hardware-mainline","last_seen_ms":%d}]}`, time.Now().UnixMilli())
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-hardware-mainline",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_hardware_mainline.v1"`,
		`"hardware_mainline_status": "ready_for_diagnostic_spikes"`,
		`"flash_allowed": false`,
		`"delete_allowed": false`,
		`"capability_invariants":`,
		`"capability": "microphone"`,
		`"accepted": true`,
		`"capability": "imu"`,
		`"declared_status": "planned_9_axis_imu"`,
		`"promotion_allowed": false`,
		"stackchan hardware mainline ready (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Index(stdout.String(), `"capability": "imu"`) > strings.Index(stdout.String(), `"capability": "camera"`) {
		t.Fatalf("hardware mainline order should probe IMU before camera: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-hardware-mainline-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("hardware mainline reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanHardwareMainlineBlocksFalseAvailablePromotion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef123456"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","servo_x":"planned_continuous_rotation_axis","rgb":"available","camera":"planned_core_s3_camera","imu":"available","ambient_light":"planned_ambient_light_sensor","proximity":"planned_proximity_sensor","battery":"planned_550mah_battery","nfc":"planned_nfc","infrared":"planned_infrared_tx_rx"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-hardware-mainline","last_seen_ms":%d}]}`, time.Now().UnixMilli())
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-hardware-mainline",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"hardware_mainline_status": "blocked"`,
		`"code": "capability_status_not_honest"`,
		`"capability": "imu"`,
		`"declared_status": "available"`,
		`"accepted": false`,
		"stackchan hardware mainline blocked (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanHardwareMainlineBlocksReleaseMicrophonePromotion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef123456"},"capabilities":{"microphone":"available","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","servo_x":"planned_continuous_rotation_axis","rgb":"available","camera":"planned_core_s3_camera","imu":"planned_9_axis_imu","ambient_light":"planned_ambient_light_sensor","proximity":"planned_proximity_sensor","battery":"planned_550mah_battery","nfc":"planned_nfc","infrared":"planned_infrared_tx_rx"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-hardware-mainline","last_seen_ms":%d}]}`, time.Now().UnixMilli())
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-hardware-mainline",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"hardware_mainline_status": "blocked"`,
		`"code": "capability_status_not_honest"`,
		`"capability": "microphone"`,
		`"declared_status": "available"`,
		`"accepted": false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanHardwareMainlineBlocksMissingPlannedCapability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef123456"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","servo_x":"planned_continuous_rotation_axis","rgb":"available","ambient_light":"planned_ambient_light_sensor","proximity":"planned_proximity_sensor","battery":"planned_550mah_battery","nfc":"planned_nfc","infrared":"planned_infrared_tx_rx"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-hardware-mainline","last_seen_ms":%d}]}`, time.Now().UnixMilli())
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-hardware-mainline",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"hardware_mainline_status": "blocked"`,
		`"code": "capability_missing"`,
		"stackchan hardware mainline blocked (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanHardwareMainlineRejectsLegacyDeviceIDWithoutEchoingIt(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-hardware-mainline",
		"--gateway-url", "http://127.0.0.1:21080",
		"--device-id", "x21-stackchan-001",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21") {
		t.Fatalf("legacy identity leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "device id contains forbidden legacy identity") {
		t.Fatalf("stderr missing legacy identity guard: %s", stderr.String())
	}
}

func TestRunStackChanPhysicalEvidenceWritesPendingTemplateFromIdentity(t *testing.T) {
	dir := t.TempDir()
	artifactSHA := strings.Repeat("c", 64)
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, artifactSHA)
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_physical_evidence.v1"`,
		`"identity_acceptance_report_path": "` + identityPath + `"`,
		`"device_id": "stackchan-001"`,
		`"commit": "abcdef1"`,
		`"artifact_sha256": "` + artifactSHA + `"`,
		`"capability": "microphone"`,
		`"capability": "rgb"`,
		`"status": "pending"`,
		`"evidence_type": "operator_observation_required"`,
		`"flash_allowed": false`,
		"stackchan physical evidence template written (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-physical-evidence-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("physical evidence reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanPhysicalEvidenceCanFeedCapabilityAcceptance(t *testing.T) {
	dir := t.TempDir()
	artifactSHA := strings.Repeat("d", 64)
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, artifactSHA)
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	args := []string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}
	for _, pass := range []string{
		"microphone=gateway_audio_frame",
		"speaker=audible_playback",
		"screen=operator_visible_state",
		"screen_touch=touch_event",
		"top_touch=touch_event",
		"servo_y=servo_clamped_motion",
		"rgb=operator_visible_state",
	} {
		args = append(args, "--pass", pass)
	}
	code := Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("physical evidence code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	evidencePath := newestGlob(t, filepath.Join(outputDir, "a21-stackchan-physical-evidence-*.json"))
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"stackchan-capability-acceptance",
		"--identity-acceptance", identityPath,
		"--evidence", evidencePath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("capability acceptance code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"capability_acceptance_status": "confirmed"`) {
		t.Fatalf("capability acceptance not confirmed: %s", stdout.String())
	}
}

func TestRunStackChanPhysicalEvidenceDerivesGatewayObservableSignals(t *testing.T) {
	dir := t.TempDir()
	artifactSHA := strings.Repeat("f", 64)
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, artifactSHA)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/devices":
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "speaking",
      "runtime_echo": {"screen": "speaking", "servo_y": "48deg", "rgb": "#002430"},
      "last_event": "touch.wake_or_listen",
      "last_touch_source": "screen",
      "last_trace_id": "a21-trace-physical-auto",
      "firmware": {"id": "a21-stackchan", "version": "0.1.0", "board": "m5stack-cores3", "commit": "abcdef1"},
      "capabilities": {
        "microphone": "available",
        "speaker": "available",
        "screen": "available",
        "screen_touch": "available",
        "top_touch": "available",
        "servo_y": "available",
        "rgb": "available"
      },
      "last_seen_ms": 1780000000000
    }
  ]
}`))
		case "/v1/traces":
			if r.URL.Query().Get("trace_id") != "a21-trace-physical-auto" {
				t.Fatalf("trace id = %q, want a21-trace-physical-auto", r.URL.Query().Get("trace_id"))
			}
			_, _ = w.Write([]byte(`{
  "trace_id": "a21-trace-physical-auto",
  "events": [
    {"name": "audio.frame.received", "trace_id": "a21-trace-physical-auto", "device_id": "stackchan-001", "at_ms": 1780000001000},
    {"name": "audio.playback.chunk.sent", "trace_id": "a21-trace-physical-auto", "device_id": "stackchan-001", "at_ms": 1780000002000}
  ],
  "summary": {"event_count": 2}
}`))
		default:
			t.Fatalf("path = %q", r.URL.Path)
		}
	}))
	defer server.Close()

	outputDir := filepath.Join(dir, "reports")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--derive-gateway",
		"--gateway-url", server.URL,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	evidencePath := newestGlob(t, filepath.Join(outputDir, "a21-stackchan-physical-evidence-*.json"))
	data, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`"gateway_url": "` + server.URL + `"`,
		`"gateway_trace_id": "a21-trace-physical-auto"`,
		`"capability": "microphone"`,
		`"evidence_type": "gateway_audio_frame"`,
		`"capability": "speaker"`,
		`"evidence_type": "gateway_audio_downlink"`,
		`"capability": "screen"`,
		`"evidence_type": "device_screen_echo"`,
		`"capability": "servo_y"`,
		`"evidence_type": "device_servo_echo"`,
		`"capability": "rgb"`,
		`"evidence_type": "device_rgb_echo"`,
		`"capability": "screen_touch"`,
		`"evidence_type": "gateway_touch_event"`,
		`"capability": "top_touch"`,
		`"status": "pending"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("evidence missing %q: %s", want, text)
		}
	}
}

func TestRunStackChanPhysicalEvidenceRejectsLegacyPassWithoutEchoingIt(t *testing.T) {
	dir := t.TempDir()
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, strings.Repeat("e", 64))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--pass", "microphone=x21_probe",
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21_probe") {
		t.Fatalf("legacy pass leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func writeTestStackChanIdentityAcceptanceReport(t *testing.T, dir string, artifactSHA string) string {
	t.Helper()
	identityPath := filepath.Join(dir, "a21-stackchan-identity-acceptance.json")
	artifactPath := filepath.Join(dir, "artifacts", "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(identityPath, []byte(`{
  "schema_version": "a21.stackchan_identity_acceptance.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "hardware_acceptance_scope": "identity_only",
  "identity_acceptance_status": "identity_confirmed",
  "device_id": "stackchan-001",
  "commit": "abcdef1",
  "artifact_path": "`+artifactPath+`",
  "artifact_sha256": "`+artifactSHA+`",
  "device_identity": {
    "device_identity_confirmed": true,
    "device": {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "idle",
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
      "last_seen_ms": 1780000000000
    }
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return identityPath
}

func writeTestFirmwareManifest(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "a21-firmware.json")
	if err := os.WriteFile(manifest, []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(testPlatformIOConfig("m5stack-cores3")), 0o644); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func writeTestGatewayDeviceReport(t *testing.T, reportPath string, content string) {
	t.Helper()
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "{") && !strings.Contains(content, `"schema_version"`) {
		content = strings.TrimPrefix(content, "{")
		content = `{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",` + content
	}
	if err := os.WriteFile(reportPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newestGlob(t *testing.T, pattern string) string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatalf("no files match %q", pattern)
	}
	return matches[len(matches)-1]
}

func testPlatformIOConfig(board string) string {
	return `[platformio]
default_envs = a21_stackchan_cores3

[env:a21_stackchan_cores3]
platform = espressif32@7.0.1
board = ` + board + `
framework = arduino
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
extra_scripts =
  pre:scripts/a21_block_raw_upload.py
  pre:scripts/a21_build_identity.py
lib_deps =
  m5stack/M5Unified @ 0.2.16
  bblanchon/ArduinoJson @ 7.4.3
  links2004/WebSockets @ 2.7.3

[env:a21_stackchan_native]
platform = native
test_framework = unity
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
extra_scripts =
  pre:scripts/a21_block_raw_upload.py
  pre:scripts/a21_build_identity.py
lib_deps =
  bblanchon/ArduinoJson @ 7.4.3
`
}

func writeFirmwareArtifactWithChecksum(t *testing.T, artifactPath string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	commit := testArtifactCommitFromName(t, artifactPath)
	content = append(content, []byte("\na21-stackchan\n0.1.0\nm5stack-cores3\n"+commit+"\n")...)
	if err := os.WriteFile(artifactPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	checksum := hex.EncodeToString(sum[:])
	if err := os.WriteFile(artifactPath+".sha256", []byte(checksum+"  "+filepath.Base(artifactPath)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := firmwarecheck.ReleaseIndexEntry{
		SchemaVersion: "a21.firmware.release.v1",
		FirmwareID:    "a21-stackchan",
		Version:       "0.1.0",
		Board:         "m5stack-cores3",
		Commit:        commit,
		Timestamp:     testArtifactTimestampFromName(t, artifactPath),
		ArtifactPath:  artifactPath,
		SHA256Path:    artifactPath + ".sha256",
		SHA256:        checksum,
		Build:         testFirmwareBuildProvenance(artifactPath),
	}
	manifestData, err := json.MarshalIndent(firmwarecheck.FirmwareReleaseManifest{
		SchemaVersion: firmwarecheck.ReleaseManifestSchemaVersion,
		Project:       "A21",
		FirmwareID:    entry.FirmwareID,
		Version:       entry.Version,
		Board:         entry.Board,
		Commit:        entry.Commit,
		Timestamp:     entry.Timestamp,
		ArtifactPath:  artifactPath,
		ArtifactName:  filepath.Base(artifactPath),
		SHA256Path:    artifactPath + ".sha256",
		SHA256:        checksum,
		Build:         testFirmwareBuildProvenance(artifactPath),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath+".manifest.json", append(manifestData, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(filepath.Dir(artifactPath), firmwarecheck.ReleaseIndexFileName)
	file, err := os.OpenFile(indexPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		t.Fatal(err)
	}
}

func writeTestBootstrapFlashImages(t *testing.T, dir string) (string, string) {
	t.Helper()
	buildDir := filepath.Join(dir, "build")
	coreDir := filepath.Join(dir, "platformio-core")
	for path, content := range map[string][]byte{
		filepath.Join(buildDir, "bootloader.bin"): []byte("a21 bootloader"),
		filepath.Join(buildDir, "partitions.bin"): []byte("a21 partitions"),
		filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"): []byte("a21 boot app"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return buildDir, coreDir
}

func writeTestMicProbeFlashImages(t *testing.T, dir string, commit string) (string, string, string) {
	t.Helper()
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_mic_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	for path, content := range map[string][]byte{
		filepath.Join(buildDir, "bootloader.bin"): []byte("a21 mic probe bootloader"),
		filepath.Join(buildDir, "partitions.bin"): []byte("a21 mic probe partitions"),
		filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"): []byte("a21 mic probe boot app"),
		artifact: []byte("a21-stackchan\n0.1.0\nm5stack-cores3\n" + commit + "\ndiagnostic_probe_m5unified_i2s_capture\n"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return buildDir, coreDir, artifact
}

func writeTestIMUProbeFlashImages(t *testing.T, dir string, commit string) (string, string, string) {
	t.Helper()
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_imu_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	for path, content := range map[string][]byte{
		filepath.Join(buildDir, "bootloader.bin"): []byte("a21 IMU probe bootloader"),
		filepath.Join(buildDir, "partitions.bin"): []byte("a21 IMU probe partitions"),
		filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"): []byte("a21 IMU probe boot app"),
		artifact: []byte("a21-stackchan\n0.1.0\nm5stack-cores3\n" + commit + "\ndiagnostic_probe_m5unified_imu\n"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return buildDir, coreDir, artifact
}

func writeTestSensorProbeFlashImages(t *testing.T, dir string, commit string) (string, string, string) {
	t.Helper()
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_sensor_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	for path, content := range map[string][]byte{
		filepath.Join(buildDir, "bootloader.bin"): []byte("a21 sensor probe bootloader"),
		filepath.Join(buildDir, "partitions.bin"): []byte("a21 sensor probe partitions"),
		filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"): []byte("a21 sensor probe boot app"),
		artifact: []byte("a21-stackchan\n0.1.0\nm5stack-cores3\n" + commit + "\ndiagnostic_probe_ltr553_ambient_light\ndiagnostic_probe_ltr553_proximity\ndiagnostic_probe_ina226_battery\n"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return buildDir, coreDir, artifact
}

func readTestSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		t.Fatalf("sha256 file %q is empty", path)
	}
	return fields[0]
}

func testFirmwareBuildProvenance(artifactPath string) firmwarecheck.FirmwareBuildProvenance {
	return firmwarecheck.FirmwareBuildProvenance{
		BuildSystem:     "platformio",
		PlatformIOEnv:   "a21_stackchan_cores3",
		PlatformIOBoard: "m5stack-cores3",
		SourcePath:      filepath.Join(filepath.Dir(artifactPath), "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3", "firmware.bin"),
		SourceName:      "firmware.bin",
	}
}

func writeNamespaceAuditGitScript(t *testing.T, dir string, files string) {
	t.Helper()
	path := filepath.Join(dir, "git")
	content := "#!/bin/sh\nif [ \"$1\" = \"ls-files\" ]; then\ncat <<'EOF'\n" + files + "EOF\nelse\nexit 2\nfi\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

type promotionGitScriptOptions struct {
	remoteNames            string
	targetBranchConfigured bool
	missingAncestor        string
}

func writePromotionReadinessGitScript(t *testing.T, dir string, options promotionGitScriptOptions) {
	t.Helper()
	path := filepath.Join(dir, "git")
	mainExit := "1"
	masterExit := "1"
	remoteNames := options.remoteNames
	if options.targetBranchConfigured {
		mainExit = "0"
	}
	content := `#!/bin/sh
if [ "$1" = "-C" ]; then
  shift
  shift
fi
case "$1 $2 $3" in
  "rev-parse --abbrev-ref HEAD")
    echo "codex/a21-integration-governance-slices"
    exit 0
    ;;
  "rev-parse --short=12 HEAD")
    echo "abcdef123456"
    exit 0
    ;;
  "status --porcelain --untracked-files=all")
    exit 0
    ;;
  "remote  ")
    cat <<'EOF'
` + remoteNames + `EOF
    exit 0
    ;;
  "show-ref --verify --quiet")
    if [ "$4" = "refs/heads/main" ]; then
      exit ` + mainExit + `
    fi
    if [ "$4" = "refs/heads/master" ]; then
      exit ` + masterExit + `
    fi
    exit 1
    ;;
  "merge-base --is-ancestor "*)
    if [ "$3" = "` + options.missingAncestor + `" ]; then
      exit 1
    fi
    exit 0
    ;;
esac
exit 2
`
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func testArtifactCommitFromName(t *testing.T, artifactPath string) string {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(artifactPath), ".bin")
	parts := strings.Split(name, "-")
	if len(parts) < 4 {
		t.Fatalf("artifact name %q lacks commit field", name)
	}
	return parts[len(parts)-3]
}

func testArtifactTimestampFromName(t *testing.T, artifactPath string) string {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(artifactPath), ".bin")
	parts := strings.Split(name, "-")
	if len(parts) < 5 {
		t.Fatalf("artifact name %q lacks timestamp field", name)
	}
	return parts[len(parts)-2] + "-" + parts[len(parts)-1]
}

func writeAppTestWAV(t *testing.T, path string, sampleRate int, pcm []byte) {
	t.Helper()
	byteRate := sampleRate * 2
	blockAlign := 2
	header := []byte{
		'R', 'I', 'F', 'F',
		0, 0, 0, 0,
		'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ',
		16, 0, 0, 0,
		1, 0,
		1, 0,
		byte(sampleRate), byte(sampleRate >> 8), byte(sampleRate >> 16), byte(sampleRate >> 24),
		byte(byteRate), byte(byteRate >> 8), byte(byteRate >> 16), byte(byteRate >> 24),
		byte(blockAlign), byte(blockAlign >> 8),
		16, 0,
		'd', 'a', 't', 'a',
		byte(len(pcm)), byte(len(pcm) >> 8), byte(len(pcm) >> 16), byte(len(pcm) >> 24),
	}
	riffSize := uint32(len(header) - 8 + len(pcm))
	header[4] = byte(riffSize)
	header[5] = byte(riffSize >> 8)
	header[6] = byte(riffSize >> 16)
	header[7] = byte(riffSize >> 24)
	if err := os.WriteFile(path, append(header, pcm...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func appTestPCM16Base64(sample int16) string {
	const samplesPer20MS16K = 320
	data := make([]byte, samplesPer20MS16K*2)
	for i := 0; i < samplesPer20MS16K; i++ {
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}
