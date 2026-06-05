package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/firmwarecheck"
)

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
		execution.QueryScope != "public_only" ||
		execution.WorkspaceStatus != "searchable" ||
		execution.SourceScopeCounts["public"] != 5 ||
		execution.EvidenceCount != 5 ||
		execution.CardCount != 1 ||
		execution.FollowUpCount != 1 ||
		execution.ProfessionalAcceptanceStatus != "adapter_smoke_passed" ||
		!execution.RedactionOK ||
		execution.PRDAccepted {
		t.Fatalf("v21 professional execution = %+v, want explicit redacted executed adapter evidence", execution)
	}
	if !report.ServerSide.V21ProfessionalEvidenceReady ||
		report.ServerSide.ProfessionalRitualReady ||
		containsExactProductString(report.ServerSide.MissingEvidence, "v21_professional_smoke") ||
		!containsExactProductString(report.ServerSide.MissingEvidence, "professional_ritual_execution") {
		t.Fatalf("server-side readiness = %+v, want adapter smoke to close V21 evidence but not professional ritual", report.ServerSide)
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
		`"query_scope": "public_only"`,
		`"workspace_status": "searchable"`,
		`"source_scope_counts": {`,
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
		!report.ServerSide.ProfessionalRitualReady ||
		containsExactProductString(report.ServerSide.MissingEvidence, "v21_professional_smoke") ||
		containsExactProductString(report.ServerSide.MissingEvidence, "professional_ritual_execution") ||
		containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "v21_professional_execution") {
		t.Fatalf("server/canonical = %+v/%+v, want executed professional report to close V21 evidence gap", report.ServerSide, report.CanonicalDecision)
	}
	if report.LaunchReady || report.CanonicalDecision.PRDAccepted {
		t.Fatalf("launch/canonical = %v/%v, want professional evidence to reduce gap without PRD overclaim", report.LaunchReady, report.CanonicalDecision.PRDAccepted)
	}
}

func TestProductReadinessRejectsXiaozhiProfessionalReportWithoutReadRecord(t *testing.T) {
	server := newProductReadinessTestServer(t, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`)
	fixture := writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t)
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw, "read_record")
	mutated, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	missingReadRecord := filepath.Join(dir, "a21-xiaozhi-professional-missing-read-record.json")
	if err := os.WriteFile(missingReadRecord, mutated, 0o644); err != nil {
		t.Fatal(err)
	}

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:            server.URL,
		DeviceID:              "stackchan-001",
		V21ProfessionalReport: missingReadRecord,
	}, []string{
		"A21_PROVIDER_PRIMARY=mock",
	})

	if report.V21.Professional.Valid ||
		report.ServerSide.V21ProfessionalEvidenceReady ||
		report.ServerSide.ProfessionalRitualReady ||
		report.ServerSide.ProfessionalReadRecordReady {
		t.Fatalf("v21/server readiness = %+v/%+v, want missing read-record report rejected", report.V21, report.ServerSide)
	}
	if !containsProductFinding(report.Findings, "v21_professional_report_missing_field", "read_record.observed") {
		t.Fatalf("findings = %#v, want missing read_record.observed", report.Findings)
	}
	var encoded bytes.Buffer
	if err := writeJSONProductReadiness(&encoded, report); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{fixture, filepath.Dir(fixture), missingReadRecord, dir, server.URL, "http://", "https://", "/Users/", `"candidate_ready": true`, `"launch_ready": true`} {
		if strings.Contains(encoded.String(), forbidden) {
			t.Fatalf("product readiness leaked or overclaimed %q: %s", forbidden, encoded.String())
		}
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
	server := newProductReadinessTestServerWithRoleplay(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessRoleplayProfileFixtureJSON("ready"),
	)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-provider-smoke-20260601-191000.json", productReadinessProviderSmokeReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-xiaozhi-voice-bench-20260601-191935.json", productReadinessXiaozhiHostReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-roleplay-voice-probe-20260601-191940.json", productReadinessRoleplayVoiceReportFixtureJSON("ready"))
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
		`"voice_runtime_source_report": "a21-roleplay-voice-probe-20260601-191940.json"`,
		`"roleplay_voice_runtime_ready": true`,
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

func TestRunProductReadinessLatestV21SelectionPrefersProfessionalBenchOverNewerAdapterSmoke(t *testing.T) {
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
	if report.V21.Professional.SourceReport != "a21-xiaozhi-professional-bench-20260602-100200.json" ||
		report.V21.Professional.ProfessionalAcceptanceStatus != "external_gateway_ready" ||
		report.V21.ProfessionalExecution.SourceKind != "xiaozhi_professional_bench_report" ||
		report.ServerSide.ProfessionalRitualSourceReport != "a21-xiaozhi-professional-bench-20260602-100200.json" ||
		!report.ServerSide.ProfessionalRitualReady ||
		!report.V21.Professional.AdapterExecuted {
		t.Fatalf("v21 professional selection = %+v execution = %+v server=%+v, want professional bench to preserve ritual evidence", report.V21.Professional, report.V21.ProfessionalExecution, report.ServerSide)
	}
	for _, forbidden := range []string{server.URL, dir, professionalFixture, professionalPath, adapterPath, "http://", "https://", "/Users/", "secret-value"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("latest v21 selection leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunServerSideReadinessBundleUsesLatestReportsWithoutPathLeak(t *testing.T) {
	server := newProductReadinessTestServerWithRoleplay(t,
		`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-sim-001","identity_status":"unknown","connection_status":"online","first_seen_ms":1,"last_seen_ms":2}]}`,
		productReadinessRoleplayProfileFixtureJSON("ready"),
	)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-provider-smoke-20260602-100000.json", productReadinessProviderSmokeReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-xiaozhi-voice-bench-20260602-100100.json", productReadinessXiaozhiHostReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-v21-adapter-smoke-20260602-100200.json", productReadinessV21AdapterSmokeReportFixtureJSON())
	professionalFixture := writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t)
	copyProductReadinessReportFixture(t, professionalFixture, filepath.Join(dir, "a21-xiaozhi-professional-bench-20260602-100250.json"))
	writeProductReadinessReportFixtureFile(t, dir, "a21-roleplay-voice-probe-20260602-100300.json", productReadinessRoleplayVoiceReportFixtureJSON("ready"))
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
		`"v21_professional_source_report": "a21-xiaozhi-professional-bench-20260602-100250.json"`,
		`"professional_ritual_ready": true`,
		`"professional_ritual_source_report": "a21-xiaozhi-professional-bench-20260602-100250.json"`,
		`"professional_ritual"`,
		`"host_voice_source_report": "a21-xiaozhi-voice-bench-20260602-100100.json"`,
		`"roleplay_voice_source_report": "a21-roleplay-voice-probe-20260602-100300.json"`,
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
		`"go run ./cmd/a21-lab xiaozhi-voice-bench --repeat 3 --require-product-chain --output-dir reports"`,
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
		`"status": "skipped"`,
		`"reason": "lab command moved to cmd/a21-lab"`,
		`"command": "go run ./cmd/a21-lab xiaozhi-voice-bench --repeat 1 --require-product-chain --output-dir reports"`,
		`"absorbed_by_readiness": false`,
		`"host_voice_loopback_ready": false`,
		`"go run ./cmd/a21-lab xiaozhi-voice-bench --repeat 3 --require-product-chain --output-dir reports"`,
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
	if len(matches) != 0 {
		t.Fatalf("voice bench reports = %d, want 0 after lab split: %v", len(matches), matches)
	}
	for _, forbidden := range []string{httpServer.URL, dir, "http://", "https://", "/Users/", "secret-value", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("server-side collect leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestRunServerSideReadinessBundleCollectsMissingRoleplayVoiceRuntime(t *testing.T) {
	server := newRoleplayVoiceProbeTestServer(t, true)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-provider-smoke-20260602-100000.json", productReadinessProviderSmokeReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-xiaozhi-voice-bench-20260602-100100.json", productReadinessXiaozhiHostReportFixtureJSON())
	writeProductReadinessReportFixtureFile(t, dir, "a21-v21-adapter-smoke-20260602-100200.json", productReadinessV21AdapterSmokeReportFixtureJSON())
	professionalRitualFixture := writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t)
	copyProductReadinessReportFixture(t, professionalRitualFixture, filepath.Join(dir, "a21-xiaozhi-professional-bench-20260602-100250.json"))
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

	code := Run([]string{
		"server-side-readiness-bundle",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-sim-001",
		"--use-latest-reports",
		"--collect-missing",
		"--require-candidate",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"status": "server_side_candidate_ready"`,
		`"candidate_ready": true`,
		`"name": "roleplay_voice_runtime"`,
		`"status": "passed"`,
		`"command": "go run ./cmd/a21 roleplay-voice-probe --require-ready --output-dir reports"`,
		`"source_report": "a21-roleplay-voice-probe-`,
		`"absorbed_by_readiness": true`,
		`"professional_ritual_ready": true`,
		`"professional_ritual_source_report": "a21-xiaozhi-professional-bench-20260602-100250.json"`,
		`"roleplay_voice_runtime_ready": true`,
		`"roleplay_voice_source_report": "a21-roleplay-voice-probe-`,
		`"launch_ready": false`,
		`"prd_accepted": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-roleplay-voice-probe-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("roleplay voice reports = %d, want 1: %v", len(matches), matches)
	}
	for _, forbidden := range []string{server.URL, dir, "http://", "https://", "/Users/", "secret-value", "data_base64", "audio_base64", "raw roleplay prompt", "memory text", "provider output", "voice sample", `"launch_ready": true`, `"prd_accepted": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("server-side roleplay collection leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
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
		`"name": "professional_ritual_execution"`,
		`"reason": "lab command moved to cmd/a21-lab"`,
		`"command": "go run ./cmd/a21-lab xiaozhi-professional-bench --gateway-url \u003cgateway\u003e --output-dir reports"`,
		`"provider_smoke"`,
		`"v21_professional_smoke"`,
		`"professional_ritual_execution"`,
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
	professionalReports, err := filepath.Glob(filepath.Join(dir, "a21-xiaozhi-professional-bench-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(professionalReports) != 0 {
		t.Fatalf("professional reports = %v, want no implicit professional execution", professionalReports)
	}
	for _, forbidden := range []string{server.URL, dir, "http://", "https://", "/Users/", "secret-value", `"candidate_ready": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("server-side collect leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}
