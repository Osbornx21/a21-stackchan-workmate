package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/gateway"
)

func TestProductVoiceReadinessAcceptsVoiceCloneCLIAlias(t *testing.T) {
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
		"A21_VOICE_CLONE_CLI=" + commandPath,
		"A21_VOICE_CLONE_REF_AUDIO=" + refAudio,
	}, productProviderReadiness{}, productStackChanReadiness{})

	if !voice.LocalTTSReady || !voice.VoicePipeline.HostLocalTTSReady {
		t.Fatalf("voice readiness = %+v, want A21_VOICE_CLONE_CLI alias to enable clone TTS", voice)
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
	if !strings.Contains(stdout.String(), `"status": "blocked"`) || !strings.Contains(stdout.String(), `"demo_ready": false`) {
		t.Fatalf("stdout missing blocked product readiness: %s", stdout.String())
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

func TestProductReadinessAcceptsCloudEdgeXiaozhiReportAsCandidate(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	data := productReadinessXiaozhiHostReportFixtureJSON()
	replacements := map[string]string{
		`"provider_executed": false`:                    `"provider_executed": true`,
		`"voice_pipeline_execution_mode": "host_local"`: `"voice_pipeline_execution_mode": "cloud_edge"`,
		`"asr_profile": "sherpa_onnx"`:                  `"asr_profile": "dashscope_qwen_asr_realtime"`,
		`"asr_profile_env": "A21_ASR_LOCAL_PROFILE"`:    `"asr_profile_env": "A21_ASR_CLOUD_PROFILE"`,
		`"llm_profile": "ollama_local"`:                 `"llm_profile": "stepfun"`,
		`"tts_profile": "sherpa_onnx_tts"`:              `"tts_profile": "dashscope_qwen_tts_realtime"`,
		`"host_local_asr_executed": true`:               `"host_local_asr_executed": false`,
		`"host_local_text_executed": true`:              `"host_local_text_executed": false`,
		`"host_local_tts_executed": true`:               `"host_local_tts_executed": false,` + "\n" + `    "host_product_chain_ready": true`,
	}
	for old, replacement := range replacements {
		data = strings.Replace(data, old, replacement, 1)
	}
	fixture := writeProductReadinessXiaozhiHostReportFixtureFromData(t, data)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:    server.URL,
		DeviceID:      "stackchan-001",
		XiaozhiReport: fixture,
	}, []string{
		"A21_PROVIDER_PRIMARY=stepfun",
		"A21_ASR_CLOUD_PROFILE=dashscope_qwen_asr_realtime",
		"A21_TEXT_STREAM_PROFILE=stepfun",
		"A21_TTS_FAST_PROFILE=dashscope_qwen_tts_realtime",
	})

	pipeline := report.Voice.VoicePipeline
	if !pipeline.HostLoopbackCandidateReady || !pipeline.HostProductChainReady || !report.Voice.ContinuousVoiceReady {
		t.Fatalf("voice pipeline = %+v, voice = %+v, findings = %#v, want cloud-edge candidate product-chain evidence absorbed", pipeline, report.Voice, report.Findings)
	}
	if pipeline.ExecutionMode != "cloud_edge" ||
		pipeline.ASRProfile != "dashscope_qwen_asr_realtime" ||
		pipeline.TextStreamProfile != "stepfun" ||
		pipeline.TTSProfile != "dashscope_qwen_tts_realtime" {
		t.Fatalf("voice pipeline = %+v, want cloud-edge provider profiles preserved", pipeline)
	}
	if pipeline.HostLocalASRReady || pipeline.HostLocalTextReady || pipeline.HostLocalTTSReady {
		t.Fatalf("voice pipeline = %+v, cloud-edge evidence must not masquerade as host-local execution", pipeline)
	}
	if report.LaunchReady || report.CanonicalDecision.LaunchReady || report.CanonicalDecision.PRDAccepted {
		t.Fatalf("canonical decision = %+v, launch = %v; cloud-edge host bench must stay below PRD acceptance", report.CanonicalDecision, report.LaunchReady)
	}
	if containsProductFinding(report.Findings, "xiaozhi_report_invalid", "") {
		t.Fatalf("findings = %#v, want cloud-edge xiaozhi report accepted", report.Findings)
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
	return newProductReadinessTestServerWithVoiceChain(t, devicesJSON, productReadinessVoiceChainProfilesJSON("stepfun", nil))
}

func newProductReadinessTestServerWithVoiceChain(t *testing.T, devicesJSON string, voiceChainJSON string) *httptest.Server {
	t.Helper()
	return newProductReadinessTestServerWithVoiceChainAndRoleplay(t, devicesJSON, voiceChainJSON, "")
}

func newProductReadinessTestServerWithRoleplay(t *testing.T, devicesJSON string, roleplayJSON string) *httptest.Server {
	t.Helper()
	return newProductReadinessTestServerWithVoiceChainAndRoleplay(t, devicesJSON, productReadinessVoiceChainProfilesJSON("stepfun", nil), roleplayJSON)
}

func newProductReadinessTestServerWithWakeWord(t *testing.T, devicesJSON string, wakeWordJSON string) *httptest.Server {
	t.Helper()
	return newProductReadinessTestServerWithVoiceChainRoleplayWake(t, devicesJSON, productReadinessVoiceChainProfilesJSON("stepfun", nil), "", wakeWordJSON)
}

func newProductReadinessTestServerWithVoiceChainAndRoleplay(t *testing.T, devicesJSON string, voiceChainJSON string, roleplayJSON string) *httptest.Server {
	t.Helper()
	return newProductReadinessTestServerWithVoiceChainRoleplayWake(t, devicesJSON, voiceChainJSON, roleplayJSON, `{"schema_version":"a21.gateway.wake_word.v1","mode":"builtin_xiaozhi","active_phrase":"你好小智","active_pinyin":"ni hao xiao zhi","threshold":30,"runtime_status":"active_builtin_model","runtime_configurable":false,"firmware_build_required":false}`)
}

func newProductReadinessTestServerWithVoiceChainRoleplayWake(t *testing.T, devicesJSON string, voiceChainJSON string, roleplayJSON string, wakeWordJSON string) *httptest.Server {
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
			_, _ = w.Write([]byte(wakeWordJSON))
		case "/v1/voice-chain-profiles":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(voiceChainJSON))
		case "/v1/roleplay-profile":
			if strings.TrimSpace(roleplayJSON) == "" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(roleplayJSON))
		default:
			http.NotFound(w, r)
		}
	}))
}

func newRoleplayVoiceProbeTestServer(t *testing.T, ready bool) *httptest.Server {
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
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`))
		case "/v1/wake-word":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.wake_word.v1","mode":"builtin_xiaozhi","active_phrase":"你好小智","active_pinyin":"ni hao xiao zhi","threshold":30,"runtime_status":"active_builtin_model","runtime_configurable":false,"firmware_build_required":false}`))
		case "/v1/voice-chain-profiles":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(productReadinessVoiceChainProfilesJSON("stepfun", nil)))
		case "/v1/roleplay-profile":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(productReadinessRoleplayProfileFixtureJSON("ready")))
		case "/v1/fast-companion/turn":
			var request gateway.FastCompanionTurnRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode fast companion request: %v", err)
			}
			if strings.TrimSpace(request.DeviceID) == "" || request.Mode != "roleplay" {
				t.Fatalf("fast companion request device/mode = %q/%q", request.DeviceID, request.Mode)
			}
			if request.LocalAudio.ASRProvider == "" || len(request.LocalAudio.Frames) != 1 || request.LocalAudio.Frames[0].DataBase64 == "" {
				t.Fatalf("fast companion local audio request = %+v", request.LocalAudio)
			}
			status := "boundary_ready"
			textStreamExecuted := false
			events := []map[string]any{}
			if ready {
				status = "pipeline_completed"
				textStreamExecuted = true
				events = append(events, map[string]any{
					"protocol":   "a21.device.v1",
					"device_id":  request.DeviceID,
					"kind":       "audio.playback.chunk",
					"seq":        4,
					"trace_id":   request.TraceID,
					"session_id": request.SessionID,
				})
			}
			response := map[string]any{
				"trace_id":             request.TraceID,
				"session_id":           request.SessionID,
				"device_id":            request.DeviceID,
				"mode":                 "roleplay",
				"status":               status,
				"route":                "fast_companion_hybrid",
				"audio_frontend":       "local_audio",
				"text_stream_provider": "stepfun",
				"provider_family":      "text_stream",
				"text_stream_executed": textStreamExecuted,
				"roleplay":             roleplayVoiceProbeTestRuntime(),
				"events":               events,
			}
			w.Header().Set("content-type", "application/json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("encode fast companion response: %v", err)
			}
		case "/v1/traces":
			traceID := r.URL.Query().Get("trace_id")
			markers := []string{
				"roleplay.profile.ready",
				"roleplay.memory.ready",
				"fast_companion.local_audio.frontend.accepted",
			}
			if ready {
				markers = append(markers,
					"fast_companion.voice_pipeline.start",
					"roleplay.prompt_input.used",
					"roleplay.voice_clone_profile.used",
					"provider.first_content",
					"audio.downlink.first_frame",
					"device.playback.start",
					"audio.playback.chunk.sent",
				)
			}
			events := make([]map[string]any, 0, len(markers))
			for i, marker := range markers {
				events = append(events, map[string]any{
					"name":       marker,
					"trace_id":   traceID,
					"session_id": "a21-session-roleplay-voice-probe-test",
					"device_id":  "stackchan-sim-001",
					"at_ms":      int64(1780549800000 + i),
					"offset_ms":  int64(i),
				})
			}
			w.Header().Set("content-type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"trace_id": traceID, "events": events}); err != nil {
				t.Fatalf("encode trace response: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func roleplayVoiceProbeTestRuntime() map[string]any {
	return map[string]any{
		"schema_version":             "a21.roleplay_runtime.v1",
		"mode":                       "roleplay",
		"roleplay_profile":           "a21_roleplay_wry_peer",
		"scenario":                   "engineer_pushback",
		"voice_clone_profile":        "a21_voice_clone_default",
		"soul_prompt_input_ready":    true,
		"prompt_parts":               []string{"core_identity", "tone_rules", "role_soul:a21_roleplay_wry_peer", "scenario:engineer_pushback", "memory:ready"},
		"memory_policy":              "bounded_prompt_hints",
		"memory_configured":          true,
		"memory_prompt_input_ready":  true,
		"memory_hint_count":          1,
		"prompt_composed":            true,
		"prompt_stored":              false,
		"memory_text_stored":         false,
		"transcript_stored":          false,
		"provider_output_stored":     false,
		"voice_clone_sample_stored":  false,
		"professional_route_allowed": false,
		"v21_executed":               false,
	}
}

func productReadinessVoiceChainProfilesJSON(selectedLLM string, findings []string) string {
	selectedLLM = firstNonEmpty(strings.TrimSpace(selectedLLM), "stepfun")
	payload := map[string]any{
		"schema_version":               "a21.gateway.voice_chain_profiles.v1",
		"service":                      "a21-gateway",
		"selected_voice_chain_mode":    "cascade",
		"selected_asr_profile":         "dashscope_qwen_asr_realtime",
		"selected_llm_profile":         selectedLLM,
		"fixed_tts_profile":            "dashscope_qwen_tts_realtime",
		"selected_tts_profile":         "dashscope_qwen_tts_realtime",
		"selected_realtime_provider":   "openai_realtime",
		"selected_voice_clone_profile": "a21_voice_clone_default",
		"hot_switch":                   true,
		"findings":                     findings,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(data)
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

func writeProductReadinessVoiceChainReadinessReportFixture(t *testing.T, llmProfile string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-xiaozhi-streaming-provider-readiness-real.json")
	if err := os.WriteFile(path, []byte(productReadinessVoiceChainReadinessReportFixtureJSON(llmProfile)), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
