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

func TestProductReadinessDoesNotAskForLocalV21AdapterWhenProfessionalGatewayEvidenceIsReady(t *testing.T) {
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

	if report.V21.Healthy {
		t.Fatalf("v21 readiness = %+v, want local adapter runtime to remain unconfigured", report.V21)
	}
	if !productProfessionalRitualReady(report.V21) || !productProfessionalReadRecordReady(report.V21) {
		t.Fatalf("v21 professional execution = %+v, want external Gateway professional proof accepted", report.V21.ProfessionalExecution)
	}
	if containsProductAction(report.NextActions, "A21_V21_ADAPTER_URL") {
		t.Fatalf("next actions = %#v, should not ask for local adapter config after external Gateway professional proof", report.NextActions)
	}
	if report.LaunchReady {
		t.Fatalf("launch_ready = true with professional-only evidence; physical gates must still block")
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
		`"query_scope": "public_only"`,
		`"workspace_status": "searchable"`,
		`"source_scope_counts": {`,
		`"public": 5`,
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
