package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"a21.local/a21/internal/firmwarecheck"
)

func TestProductReadinessReportsServerSideCandidateWhenEvidenceSlicesPass(t *testing.T) {
	server := newProductReadinessTestServerWithRoleplay(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessRoleplayProfileFixtureJSON("ready"),
	)
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
		V21ProfessionalReport: writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t),
		V21AdapterSmokeReport: writeProductReadinessV21AdapterSmokeReportFixture(t),
		XiaozhiReport:         writeProductReadinessXiaozhiHostReportFixture(t),
		RoleplayVoiceReport:   writeProductReadinessRoleplayVoiceReportFixture(t, "ready"),
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
		!report.ServerSide.ProfessionalRitualReady ||
		!report.ServerSide.HostVoiceLoopbackReady ||
		!report.ServerSide.RoleplayVoiceRuntimeReady ||
		!report.ServerSide.WakeWordReady ||
		!report.ServerSide.RequiresPhysicalAcceptance {
		t.Fatalf("server-side readiness = %+v, want full no-hardware candidate and physical gate preserved", report.ServerSide)
	}
	if report.ServerSide.ProviderSmokeSourceReport != "a21-provider-smoke-real.json" ||
		report.ServerSide.V21ProfessionalSourceReport != "a21-xiaozhi-professional-gateway.json" ||
		report.ServerSide.ProfessionalRitualSourceReport != "a21-xiaozhi-professional-gateway.json" ||
		report.ServerSide.HostVoiceSourceReport != "a21-xiaozhi-host-local-report.json" ||
		report.ServerSide.RoleplayVoiceSourceReport != "a21-roleplay-voice-probe-ready.json" {
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

func TestProductReadinessBlocksServerSideCandidateWhenStepFunNotSelected(t *testing.T) {
	server := newProductReadinessTestServerWithVoiceChain(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessVoiceChainProfilesJSON("deepseek", []string{"stepfun_not_selected"}),
	)
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
		V21ProfessionalReport: writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t),
		V21AdapterSmokeReport: writeProductReadinessV21AdapterSmokeReportFixture(t),
		XiaozhiReport:         writeProductReadinessXiaozhiHostReportFixture(t),
		RoleplayVoiceReport:   writeProductReadinessRoleplayVoiceReportFixture(t, "ready"),
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
		"A21_V21_ADAPTER_URL=" + server.URL,
		"A21_TEXT_STREAM_PROFILE=deepseek",
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_TTS_FAST_PROFILE=sherpa_onnx_tts",
	})

	if report.Status != "server_side_blocked" || report.ServerSide.CandidateReady {
		t.Fatalf("status/server-side = %q/%+v, want StepFun launch-policy blocker", report.Status, report.ServerSide)
	}
	chain := report.Voice.VoiceChain
	if !chain.Available ||
		chain.Status != "launch_policy_blocked" ||
		chain.SelectedVoiceChainMode != "cascade" ||
		chain.SelectedLLMProfile != "deepseek" ||
		chain.FixedTTSProfile != "dashscope_qwen_tts_realtime" ||
		chain.LaunchPolicyExpectedLLMProfile != "stepfun" ||
		chain.LaunchPolicySatisfied {
		t.Fatalf("voice-chain readiness = %+v, want selected DeepSeek with StepFun blocker", chain)
	}
	if report.ServerSide.VoiceChain.SelectedLLMProfile != "deepseek" ||
		report.ServerSide.VoiceChain.Status != "launch_policy_blocked" {
		t.Fatalf("server-side voice-chain = %+v, want same selected DeepSeek blocker", report.ServerSide.VoiceChain)
	}
	if !containsExactProductString(report.ServerSide.MissingEvidence, "stepfun_not_selected") ||
		!containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "stepfun_not_selected") ||
		!containsProductFinding(report.Findings, "stepfun_not_selected", "") {
		t.Fatalf("missing/finding = server:%#v canonical:%#v findings:%#v, want stepfun_not_selected blocker", report.ServerSide.MissingEvidence, report.CanonicalDecision.MissingRealEvidence, report.Findings)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"voice_chain"`,
		`"selected_voice_chain_mode": "cascade"`,
		`"selected_asr_profile": "dashscope_qwen_asr_realtime"`,
		`"selected_llm_profile": "deepseek"`,
		`"fixed_tts_profile": "dashscope_qwen_tts_realtime"`,
		`"selected_tts_profile": "dashscope_qwen_tts_realtime"`,
		`"launch_policy_expected_llm_profile": "stepfun"`,
		`"stepfun_not_selected"`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("product readiness JSON missing %q: %s", want, encoded.String())
		}
	}
	for _, forbidden := range []string{server.URL, "http://", "https://", "/Users/", "secret-value", `"candidate_ready": true`, `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessIngestsSelectedVoiceChainStaticReadiness(t *testing.T) {
	server := newProductReadinessTestServerWithVoiceChain(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessVoiceChainProfilesJSON("stepfun", nil),
	)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:                server.URL,
		DeviceID:                  "stackchan-001",
		VoiceChainReadinessReport: writeProductReadinessVoiceChainReadinessReportFixture(t, "stepfun"),
	}, nil)

	chain := report.Voice.VoiceChain
	if !chain.Available ||
		!chain.CapabilityEvidenceAvailable ||
		!chain.CapabilityEvidenceMatched ||
		chain.CapabilityEvidenceStatus != "passed" ||
		chain.CapabilityEvidenceSourceReport != "a21-xiaozhi-streaming-provider-readiness-real.json" ||
		chain.CapabilityExecutionMode != "static_no_execute_provider_capability_gate" ||
		!chain.StaticCapabilityReady ||
		!chain.ASRStreamingReady ||
		!chain.LLMStreamingReady ||
		!chain.TTSStreamingReady {
		t.Fatalf("voice-chain capability = %+v, want selected static provider-chain evidence absorbed", chain)
	}
	if !containsExactProductString(chain.CapabilityFindings, "stepfun_selected") {
		t.Fatalf("capability findings = %#v, want stepfun_selected", chain.CapabilityFindings)
	}
	if report.LaunchReady || report.CanonicalDecision.PRDAccepted {
		t.Fatalf("launch/canonical = %v/%+v, static evidence must not promote PRD", report.LaunchReady, report.CanonicalDecision)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"capability_evidence_available": true`,
		`"capability_evidence_matched": true`,
		`"capability_evidence_source_report": "a21-xiaozhi-streaming-provider-readiness-real.json"`,
		`"static_capability_ready": true`,
		`"asr_streaming_ready": true`,
		`"llm_streaming_ready": true`,
		`"tts_streaming_ready": true`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("product readiness JSON missing %q: %s", want, encoded.String())
		}
	}
	for _, forbidden := range []string{server.URL, "http://", "https://", "/Users/", "secret-value", "sk-a21", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessSurfacesRoleplayImmersionReadiness(t *testing.T) {
	server := newProductReadinessTestServerWithRoleplay(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessRoleplayProfileFixtureJSON("ready"),
	)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL: server.URL,
		DeviceID:   "stackchan-001",
	}, nil)

	roleplay := report.Roleplay
	if !roleplay.Available ||
		!roleplay.ImmersionReady ||
		roleplay.Status != "ready" ||
		roleplay.SelectedRoleplayProfile != "a21_roleplay_wry_peer" ||
		roleplay.SelectedScenario != "engineer_pushback" ||
		roleplay.SelectedVoiceCloneProfile != "a21_voice_clone_default" ||
		!roleplay.SoulPromptInputReady ||
		!roleplay.PromptComposed ||
		roleplay.PromptPartCount != 3 ||
		!roleplay.MemoryConfigured ||
		!roleplay.MemoryPromptInputReady ||
		!roleplay.MemoryReady ||
		roleplay.MemoryHintCount != 1 ||
		!roleplay.VoiceCloneProfileReady ||
		!roleplay.ExpressionPlanAvailable ||
		roleplay.ExpressionDeliveryPolicy != "no_send_plan_only" ||
		roleplay.ExpressionActionCount != 4 ||
		roleplay.ExpressionPacketCount != 4 ||
		roleplay.PhysicalAccepted ||
		roleplay.ExpressionPhysicalAccepted ||
		!roleplay.ExpressionRedactionOK ||
		!roleplay.RuntimeRedactionOK ||
		roleplay.PromptStored ||
		roleplay.MemoryTextStored ||
		roleplay.TranscriptStored ||
		roleplay.ProviderOutputStored ||
		roleplay.AudioStored ||
		roleplay.VoiceCloneSampleStored ||
		roleplay.ProfessionalRouteAllowed ||
		roleplay.V21Executed {
		t.Fatalf("roleplay readiness = %+v, want safe immersion-ready contract", roleplay)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"roleplay"`,
		`"immersion_ready": true`,
		`"selected_roleplay_profile": "a21_roleplay_wry_peer"`,
		`"selected_scenario": "engineer_pushback"`,
		`"selected_voice_clone_profile": "a21_voice_clone_default"`,
		`"memory_prompt_input_ready": true`,
		`"expression_delivery_policy": "no_send_plan_only"`,
		`"physical_accepted": false`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("product readiness JSON missing %q: %s", want, encoded.String())
		}
	}
	for _, forbidden := range []string{server.URL, "http://", "https://", "/Users/", "secret-value", "raw roleplay prompt", "memory text", "provider output", "voice sample", `"physical_accepted": true`, `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("roleplay readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessRejectsUnsafeRoleplayProfile(t *testing.T) {
	server := newProductReadinessTestServerWithRoleplay(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessRoleplayProfileFixtureJSON("unsafe"),
	)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL: server.URL,
		DeviceID:   "stackchan-001",
	}, nil)

	if report.Roleplay.Available || report.Roleplay.Status != "unavailable" || !containsProductFinding(report.Findings, "roleplay_profile_invalid", "") {
		t.Fatalf("roleplay/readiness/findings = %+v / %+v, want unsafe profile rejected", report.Roleplay, report.Findings)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{server.URL, "http://", "https://", "/Users/", "secret-value", "raw roleplay prompt", "memory text", "provider output", "voice sample", "secret_roleplay", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("unsafe roleplay readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessSurfacesRoleplayVoiceRuntimeEvidence(t *testing.T) {
	server := newProductReadinessTestServerWithRoleplay(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessRoleplayProfileFixtureJSON("ready"),
	)
	fixture := writeProductReadinessRoleplayVoiceReportFixture(t, "ready")

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:          server.URL,
		DeviceID:            "stackchan-001",
		RoleplayVoiceReport: fixture,
	}, nil)

	roleplay := report.Roleplay
	if !roleplay.VoiceRuntimeEvidence ||
		!roleplay.VoiceRuntimeMatched ||
		!roleplay.VoiceRuntimeReady ||
		roleplay.VoiceRuntimeStatus != "passed" ||
		roleplay.VoiceRuntimeSourceReport != "a21-roleplay-voice-probe-ready.json" ||
		roleplay.VoiceRuntimeRoute != "fast_companion_hybrid" ||
		roleplay.VoiceRuntimeExecutionMode != "cloud_edge" ||
		roleplay.VoiceRuntimeTraceMarkers != 8 ||
		!roleplay.VoiceRuntimeTextExecuted ||
		!roleplay.VoiceRuntimePromptUsed ||
		!roleplay.VoiceRuntimeVoiceCloneUsed ||
		!roleplay.VoiceRuntimeAudioDownlink ||
		!roleplay.VoiceRuntimePlaybackStart {
		t.Fatalf("roleplay voice runtime = %+v, want matched ready evidence", roleplay)
	}
	if containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "roleplay_voice_runtime") {
		t.Fatalf("canonical missing evidence = %#v, want roleplay voice runtime gap closed", report.CanonicalDecision.MissingRealEvidence)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"voice_runtime_evidence_available": true`,
		`"voice_runtime_evidence_matched": true`,
		`"voice_runtime_ready": true`,
		`"voice_runtime_source_report": "a21-roleplay-voice-probe-ready.json"`,
		`"voice_runtime_prompt_input_used": true`,
		`"voice_runtime_voice_clone_used": true`,
		`"voice_runtime_audio_downlink_observed": true`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("product readiness JSON missing %q: %s", want, encoded.String())
		}
	}
	for _, forbidden := range []string{server.URL, fixture, filepath.Dir(fixture), "http://", "https://", "/Users/", "secret-value", "raw roleplay prompt", "memory text", "provider output", "voice sample", "data_base64", "audio_base64", "transcript", `"physical_accepted": true`, `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("roleplay voice readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessRejectsUnsafeRoleplayVoiceRuntimeEvidence(t *testing.T) {
	server := newProductReadinessTestServerWithRoleplay(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessRoleplayProfileFixtureJSON("ready"),
	)
	fixture := writeProductReadinessRoleplayVoiceReportFixture(t, "unsafe")

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:          server.URL,
		DeviceID:            "stackchan-001",
		RoleplayVoiceReport: fixture,
	}, nil)

	if report.Roleplay.VoiceRuntimeEvidence || report.Roleplay.VoiceRuntimeReady || !containsProductFinding(report.Findings, "roleplay_voice_report_invalid", "") {
		t.Fatalf("roleplay/findings = %+v / %#v, want unsafe runtime report rejected", report.Roleplay, report.Findings)
	}
	if !containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "roleplay_voice_runtime") {
		t.Fatalf("canonical missing evidence = %#v, want roleplay runtime gap preserved", report.CanonicalDecision.MissingRealEvidence)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{server.URL, fixture, filepath.Dir(fixture), "http://", "https://", "/Users/", "secret-value", "raw roleplay prompt", "memory text", "provider output", "voice sample", "data_base64", "audio_base64", "transcript", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("unsafe roleplay voice readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestRunProductReadinessUsesLatestRoleplayVoiceRuntimeReport(t *testing.T) {
	server := newProductReadinessTestServerWithRoleplay(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessRoleplayProfileFixtureJSON("ready"),
	)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a21-roleplay-voice-probe-20260604-131000.json"), []byte(productReadinessRoleplayVoiceReportFixtureJSON("ready")), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"voice_runtime_ready": true`,
		`"voice_runtime_source_report": "a21-roleplay-voice-probe-20260604-131000.json"`,
		`"voice_runtime_prompt_input_used": true`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{server.URL, dir, "http://", "https://", "raw roleplay prompt", "memory text", "provider output", "voice sample", "data_base64", "audio_base64", "transcript", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked or overclaimed %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunRoleplayVoiceProbeWritesReadyReportAndProductReadinessCanIngest(t *testing.T) {
	server := newRoleplayVoiceProbeTestServer(t, true)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"roleplay-voice-probe", "--gateway-url", server.URL, "--device-id", "44:1b:f6:e2:6a:60", "--trace-id", "a21-trace-roleplay-voice-probe-test-ready", "--session-id", "a21-session-roleplay-voice-probe-test-ready", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s\n%s", code, stderr.String(), stdout.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.roleplay_voice_probe.v1"`,
		`"status": "passed"`,
		`"mode": "roleplay"`,
		`"execution_mode": "gateway_fast_companion"`,
		`"observed": true`,
		`"prompt_input_used": true`,
		`"voice_clone_profile_used": true`,
		`"audio_downlink_first_frame_observed": true`,
		`"device_playback_start_observed": true`,
		`"audio_chunk_count": 2`,
		`"physical_accepted": false`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-roleplay-voice-probe-*.json"))
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
	for _, forbidden := range []string{server.URL, dir, "http://", "https://", "data_base64", "audio_base64", "raw roleplay prompt", "memory text", "provider output", "voice sample", "secret-value", "Authorization", "Bearer", "sk-"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("roleplay voice probe leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}

	var readinessStdout bytes.Buffer
	var readinessStderr bytes.Buffer
	readinessCode := Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &readinessStdout, &readinessStderr)

	if readinessCode != 0 {
		t.Fatalf("product-readiness code = %d, want 0: %s\n%s", readinessCode, readinessStderr.String(), readinessStdout.String())
	}
	for _, want := range []string{
		`"voice_runtime_evidence_available": true`,
		`"voice_runtime_evidence_matched": true`,
		`"voice_runtime_ready": true`,
		`"voice_runtime_source_report": "` + filepath.Base(matches[0]) + `"`,
		`"voice_runtime_execution_mode": "gateway_fast_companion"`,
	} {
		if !strings.Contains(readinessStdout.String(), want) {
			t.Fatalf("product-readiness stdout missing %q: %s", want, readinessStdout.String())
		}
	}
	for _, forbidden := range []string{server.URL, dir, "http://", "https://", "data_base64", "audio_base64", "raw roleplay prompt", "memory text", "provider output", "voice sample", "secret-value", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(readinessStdout.String(), forbidden) {
			t.Fatalf("product-readiness leaked or overclaimed %q: %s", forbidden, readinessStdout.String())
		}
	}
}

func TestProductRoleplayVoiceSafeOptionalIDAcceptsDeviceMACOnly(t *testing.T) {
	for _, value := range []string{"44:1b:f6:e2:6a:60", "stackchan-sim-001", "a21-trace-roleplay-voice-probe-test"} {
		if !productRoleplayVoiceSafeOptionalID(value) {
			t.Fatalf("identity %q rejected, want accepted", value)
		}
	}
	for _, value := range []string{
		"http://47.103.57.217",
		"/Users/a21/report.json",
		"bearer secret-value",
		"44:1b:f6:e2:6a:60/../../secret",
		"trace id with spaces",
	} {
		if productRoleplayVoiceSafeOptionalID(value) {
			t.Fatalf("identity %q accepted, want rejected", value)
		}
	}
}

func TestRunRoleplayVoiceProbeWritesBlockedReportWhenPipelineIncomplete(t *testing.T) {
	server := newRoleplayVoiceProbeTestServer(t, false)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"roleplay-voice-probe", "--gateway-url", server.URL, "--device-id", "stackchan-sim-001", "--trace-id", "a21-trace-roleplay-voice-probe-test-blocked", "--session-id", "a21-session-roleplay-voice-probe-test-blocked", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0 for blocked evidence without --require-ready: %s\n%s", code, stderr.String(), stdout.String())
	}
	for _, want := range []string{
		`"status": "blocked"`,
		`"observed": false`,
		`"status": "boundary_ready"`,
		`"text_stream_executed": false`,
		`"audio_downlink_first_frame_observed": false`,
		`"device_playback_start_observed": false`,
		`"audio_chunk_count": 0`,
		`"voice_pipeline_not_completed"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-roleplay-voice-probe-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	for _, forbidden := range []string{server.URL, dir, "http://", "https://", "data_base64", "audio_base64", "raw roleplay prompt", "memory text", "provider output", "voice sample", "secret-value", "Authorization", "Bearer", "sk-", `"physical_accepted": true`, `"prd_accepted": true`} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("blocked roleplay voice probe leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}

	requireDir := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"roleplay-voice-probe", "--gateway-url", server.URL, "--device-id", "stackchan-sim-001", "--trace-id", "a21-trace-roleplay-voice-probe-test-require", "--session-id", "a21-session-roleplay-voice-probe-test-require", "--output-dir", requireDir, "--require-ready"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1 with --require-ready for blocked evidence", code)
	}
	if !strings.Contains(stdout.String(), `"status": "blocked"`) || !strings.Contains(stderr.String(), "roleplay voice probe not ready") {
		t.Fatalf("require-ready stdout/stderr = %s / %s, want blocked report and readiness failure", stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), server.URL) || strings.Contains(stderr.String(), requireDir) {
		t.Fatalf("require-ready stderr leaked URL/path: %s", stderr.String())
	}
}

func TestProductReadinessRejectsVoiceChainStaticReadinessMismatch(t *testing.T) {
	server := newProductReadinessTestServerWithVoiceChain(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessVoiceChainProfilesJSON("stepfun", nil),
	)

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:                server.URL,
		DeviceID:                  "stackchan-001",
		VoiceChainReadinessReport: writeProductReadinessVoiceChainReadinessReportFixture(t, "deepseek"),
	}, nil)

	chain := report.Voice.VoiceChain
	if !chain.CapabilityEvidenceAvailable ||
		chain.CapabilityEvidenceMatched ||
		chain.StaticCapabilityReady ||
		!containsExactProductString(chain.Findings, "voice_chain_capability_report_mismatch") ||
		!containsProductFinding(report.Findings, "voice_chain_capability_report_mismatch", "a21-xiaozhi-streaming-provider-readiness-real.json") {
		t.Fatalf("voice-chain mismatch = %+v findings=%+v, want mismatch visible and not absorbed", chain, report.Findings)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{server.URL, "http://", "https://", "/Users/", "secret-value", "sk-a21", `"static_capability_ready": true`, `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("mismatched product readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestServerSideReadinessBundleAcceptsVoiceChainStaticReadinessReport(t *testing.T) {
	server := newProductReadinessTestServerWithVoiceChain(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessVoiceChainProfilesJSON("stepfun", nil),
	)

	report := buildServerSideReadinessBundleReport(context.Background(), productReadinessOptions{
		GatewayURL:                server.URL,
		DeviceID:                  "stackchan-001",
		VoiceChainReadinessReport: writeProductReadinessVoiceChainReadinessReportFixture(t, "stepfun"),
	}, nil, serverSideReadinessCollection{})

	if !report.VoiceChain.CapabilityEvidenceAvailable ||
		!report.VoiceChain.CapabilityEvidenceMatched ||
		!report.VoiceChain.StaticCapabilityReady ||
		report.VoiceChain.CapabilityEvidenceSourceReport != "a21-xiaozhi-streaming-provider-readiness-real.json" ||
		!report.ServerSide.VoiceChain.StaticCapabilityReady {
		t.Fatalf("bundle voice-chain capability = %+v server=%+v, want static readiness passthrough", report.VoiceChain, report.ServerSide.VoiceChain)
	}
	var encoded bytes.Buffer
	if err := writeJSONServerSideReadinessBundle(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"voice_chain"`,
		`"capability_evidence_available": true`,
		`"capability_evidence_matched": true`,
		`"static_capability_ready": true`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("bundle JSON missing %q: %s", want, encoded.String())
		}
	}
	for _, forbidden := range []string{server.URL, "http://", "https://", "/Users/", "secret-value", "sk-a21", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("bundle leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestRunProductReadinessUsesLatestVoiceChainReadinessReport(t *testing.T) {
	server := newProductReadinessTestServerWithVoiceChain(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessVoiceChainProfilesJSON("stepfun", nil),
	)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-xiaozhi-streaming-provider-readiness-20260602-100000.json", productReadinessVoiceChainReadinessReportFixtureJSON("stepfun"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"product-readiness", "--gateway-url", server.URL, "--use-latest-reports", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"capability_evidence_available": true`,
		`"capability_evidence_matched": true`,
		`"capability_evidence_source_report": "a21-xiaozhi-streaming-provider-readiness-20260602-100000.json"`,
		`"static_capability_ready": true`,
		`"launch_ready": false`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{server.URL, "http://", "https://", "/Users/", "secret-value", "sk-a21", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("latest voice-chain readiness leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestServerSideReadinessBundleSurfacesStepFunNotSelected(t *testing.T) {
	server := newProductReadinessTestServerWithVoiceChain(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessVoiceChainProfilesJSON("deepseek", []string{"stepfun_not_selected"}),
	)
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	report := buildServerSideReadinessBundleReport(context.Background(), productReadinessOptions{
		GatewayURL:            server.URL,
		DeviceID:              "stackchan-001",
		ProviderSmokeReport:   writeProductReadinessProviderSmokeReportFixture(t),
		V21AdapterSmokeReport: writeProductReadinessV21AdapterSmokeReportFixture(t),
		XiaozhiReport:         writeProductReadinessXiaozhiHostReportFixture(t),
		RoleplayVoiceReport:   writeProductReadinessRoleplayVoiceReportFixture(t, "ready"),
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=secret-value",
		"A21_V21_ADAPTER_URL=" + server.URL,
		"A21_TEXT_STREAM_PROFILE=deepseek",
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_TTS_FAST_PROFILE=sherpa_onnx_tts",
	}, serverSideReadinessCollection{})

	if report.Status != "server_side_blocked" || report.CandidateReady {
		t.Fatalf("bundle status/candidate = %q/%v, want StepFun selector blocker", report.Status, report.CandidateReady)
	}
	if report.VoiceChain.SelectedLLMProfile != "deepseek" ||
		report.VoiceChain.Status != "launch_policy_blocked" ||
		report.VoiceChain.LaunchPolicySatisfied {
		t.Fatalf("bundle voice-chain = %+v, want selected DeepSeek blocker", report.VoiceChain)
	}
	if !containsExactProductString(report.MissingEvidence, "stepfun_not_selected") ||
		!containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "stepfun_not_selected") {
		t.Fatalf("bundle missing evidence = %#v canonical=%#v, want stepfun_not_selected", report.MissingEvidence, report.CanonicalDecision.MissingRealEvidence)
	}
	var encoded bytes.Buffer
	if err := writeJSONServerSideReadinessBundle(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"voice_chain"`,
		`"selected_llm_profile": "deepseek"`,
		`"launch_policy_expected_llm_profile": "stepfun"`,
		`"stepfun_not_selected"`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("bundle JSON missing %q: %s", want, encoded.String())
		}
	}
	for _, forbidden := range []string{server.URL, "http://", "https://", "/Users/", "secret-value", `"candidate_ready": true`, `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("bundle leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
	}
}

func TestProductReadinessReportsServerSideBlockedWhenWakeWordBlocksRealSlices(t *testing.T) {
	server := newProductReadinessCustomWakeTestServer(t)
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
		V21ProfessionalReport: writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t),
		V21AdapterSmokeReport: writeProductReadinessV21AdapterSmokeReportFixture(t),
		XiaozhiReport:         writeProductReadinessXiaozhiHostReportFixture(t),
		RoleplayVoiceReport:   writeProductReadinessRoleplayVoiceReportFixture(t, "ready"),
	}, []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_V21_ADAPTER_URL=" + server.URL,
		"A21_TEXT_STREAM_PROFILE=deepseek",
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_TTS_FAST_PROFILE=sherpa_onnx_tts",
	})

	if report.Status != "server_side_blocked" || report.LaunchReady || report.ServerSide.CandidateReady {
		t.Fatalf("status/launch/server-side = %q/%v/%+v, want server_side_blocked without launch/candidate", report.Status, report.LaunchReady, report.ServerSide)
	}
	if !report.Provider.RealProviderReady ||
		!report.ServerSide.ProviderEvidenceReady ||
		!report.ServerSide.V21ProfessionalEvidenceReady ||
		!report.ServerSide.ProfessionalRitualReady ||
		!report.ServerSide.HostVoiceLoopbackReady ||
		!report.ServerSide.RoleplayVoiceRuntimeReady ||
		report.ServerSide.WakeWordReady {
		t.Fatalf("server-side evidence = provider:%+v server:%+v, want real slices ready and wake blocked", report.Provider, report.ServerSide)
	}
	if len(report.ServerSide.MissingEvidence) != 1 || report.ServerSide.MissingEvidence[0] != "wake_word" {
		t.Fatalf("missing server-side evidence = %#v, want wake_word only", report.ServerSide.MissingEvidence)
	}
	for _, blocked := range []string{"real_provider_smoke", "v21_professional_execution", "continuous_voice_pipeline"} {
		if containsExactProductString(report.CanonicalDecision.MissingRealEvidence, blocked) {
			t.Fatalf("missing real evidence = %#v, should not keep closed server-side gap %q", report.CanonicalDecision.MissingRealEvidence, blocked)
		}
	}
	for _, want := range []string{"physical_stackchan_online", "physical_stackchan_prd_acceptance", "wake_word_product_ready"} {
		if !containsExactProductString(report.CanonicalDecision.MissingRealEvidence, want) {
			t.Fatalf("missing real evidence = %#v, want remaining launch gate %q", report.CanonicalDecision.MissingRealEvidence, want)
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
