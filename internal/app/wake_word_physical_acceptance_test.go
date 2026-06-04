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
)

func TestRunWakeWordPhysicalProofCommandRecordsObservedProofNoHardware(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"wake-word-physical-proof",
		"--physical-device-online",
		"--firmware-flash-executed",
		"--guarded-flash-report", "a21-wake-word-guarded-flash-20260602-040000.json",
		"--operator-observed",
		"--wake-phrase-matched",
		"--false-wake-rejected",
		"--stock-wake-rejected",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"schema_version": "a21.wake_word_physical_proof.v1"`,
		`"status": "observed"`,
		`"physical_device_online": true`,
		`"firmware_flash_executed": true`,
		`"guarded_flash_report_source": "a21-wake-word-guarded-flash-20260602-040000.json"`,
		`"operator_custom_wake_observation_present": true`,
		`"wake_phrase_matched": true`,
		`"false_wake_accepted": false`,
		`"stock_wake_accepted": false`,
		`"redaction_ok": true`,
		`"report_path": "a21-wake-word-physical-proof-`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{dir, "http://", "https://", "/Users/", "secret", "token", "raw audio"} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("wake physical proof leaked %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-wake-word-physical-proof-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("proof reports = %#v, %v, want one report", matches, err)
	}
}

func TestRunWakeWordPhysicalProofCommandRejectsMissingAffirmativeProofNoReport(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"wake-word-physical-proof",
		"--physical-device-online",
		"--firmware-flash-executed",
		"--guarded-flash-report", "a21-wake-word-guarded-flash-20260602-040000.json",
		"--wake-phrase-matched",
		"--false-wake-rejected",
		"--stock-wake-rejected",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("code = 0, want rejection: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %s, want no report", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-wake-word-physical-proof-*.json"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("proof reports = %#v, %v, want no report", matches, err)
	}
	for _, forbidden := range []string{dir, "a21-wake-word-guarded-flash-20260602-040000.json", "/Users/", "secret", "token"} {
		if strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("rejection leaked %q: stderr=%s", forbidden, stderr.String())
		}
	}
}

func TestRunWakeWordPhysicalProofCommandRejectsUnsafeGuardedFlashReportNoLeak(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"wake-word-physical-proof",
		"--physical-device-online",
		"--firmware-flash-executed",
		"--guarded-flash-report", "/Users/private/a21/a21-wake-word-guarded-flash-20260602-040000.json",
		"--operator-observed",
		"--wake-phrase-matched",
		"--false-wake-rejected",
		"--stock-wake-rejected",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("code = 0, want unsafe guarded flash report rejection: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %s, want no report", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-wake-word-physical-proof-*.json"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("proof reports = %#v, %v, want no report", matches, err)
	}
	for _, forbidden := range []string{dir, "/Users/private", "a21-wake-word-guarded-flash-20260602-040000.json", "secret", "token"} {
		if strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("unsafe rejection leaked %q: stderr=%s", forbidden, stderr.String())
		}
	}
}

func TestRunWakeWordPhysicalAcceptanceConsumesProducedPhysicalProof(t *testing.T) {
	dir := t.TempDir()
	packagePath := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-030000.json", productReadinessWakeWordFirmwarePackageReportFixtureJSON())
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"wake-word-physical-proof",
		"--physical-device-online",
		"--firmware-flash-executed",
		"--guarded-flash-report", "a21-wake-word-guarded-flash-20260602-040000.json",
		"--operator-observed",
		"--wake-phrase-matched",
		"--false-wake-rejected",
		"--stock-wake-rejected",
		"--output-dir", dir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("proof code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-wake-word-physical-proof-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("proof reports = %#v, %v, want one report", matches, err)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"wake-word-physical-acceptance",
		"--package-report", packagePath,
		"--proof-report", matches[0],
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("acceptance code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"schema_version": "a21.wake_word_physical_acceptance.v1"`,
		`"status": "accepted"`,
		`"product_ready": true`,
		`"guarded_flash_report_source": "a21-wake-word-guarded-flash-20260602-040000.json"`,
		`"false_wake_accepted": false`,
		`"stock_wake_accepted": false`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
}

func TestRunWakeWordPhysicalAcceptanceCommandRecordsPackagePhysicalProofNoHardware(t *testing.T) {
	dir := t.TempDir()
	packagePath := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-030000.json", productReadinessWakeWordFirmwarePackageReportFixtureJSON())
	proofPath := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-physical-proof-20260602-040000.json", productReadinessWakeWordPhysicalProofReportFixtureJSON())
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"wake-word-physical-acceptance",
		"--package-report", packagePath,
		"--proof-report", proofPath,
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"schema_version": "a21.wake_word_physical_acceptance.v1"`,
		`"status": "accepted"`,
		`"product_ready": true`,
		`"mode": "custom_multinet"`,
		`"desired_phrase": "小阿二一"`,
		`"desired_pinyin": "xiao a er yi"`,
		`"threshold": 35`,
		`"package_source_report": "a21-wake-word-firmware-package-20260602-030000.json"`,
		`"package_artifact_name": "a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-abcdef123456-20260602-030000.bin"`,
		`"package_manifest_name": "a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-abcdef123456-20260602-030000.bin.manifest.json"`,
		`"physical_device_online": true`,
		`"firmware_flash_executed": true`,
		`"guarded_flash_report_source": "a21-wake-word-guarded-flash-20260602-040000.json"`,
		`"operator_custom_wake_observation_present": true`,
		`"wake_phrase_matched": true`,
		`"false_wake_accepted": false`,
		`"stock_wake_accepted": false`,
		`"report_path": "a21-wake-word-physical-acceptance-`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{packagePath, proofPath, dir, "http://", "https://", "/Users/", "secret", "token", "raw audio"} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("wake physical acceptance leaked %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-wake-word-physical-acceptance-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("acceptance reports = %#v, %v, want one report", matches, err)
	}
}

func TestProductReadinessAcceptsWakeWordPhysicalAcceptanceWithMatchingPackage(t *testing.T) {
	server := newProductReadinessCustomWakeTestServer(t)
	stubWakeWordPhysicalAcceptanceSerialLister(t)
	dir := t.TempDir()
	packagePath := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-030000.json", productReadinessWakeWordFirmwarePackageReportFixtureJSON())
	acceptancePath := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-physical-acceptance-20260602-040000.json", productReadinessWakeWordPhysicalAcceptanceReportFixtureJSON())
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"product-readiness",
		"--gateway-url", server.URL,
		"--wake-word-firmware-package-report", packagePath,
		"--wake-word-physical-acceptance-report", acceptancePath,
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report productReadinessReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode product readiness report: %v\n%s", err, stdout.String())
	}
	if !report.WakeWord.ProductReady || !report.ServerSide.WakeWordReady {
		t.Fatalf("wake readiness = %+v server_side=%+v, want accepted custom wake proof", report.WakeWord, report.ServerSide)
	}
	if containsExactProductString(report.ServerSide.MissingEvidence, "wake_word") ||
		containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "wake_word_product_ready") {
		t.Fatalf("wake evidence still missing: server=%#v canonical=%#v", report.ServerSide.MissingEvidence, report.CanonicalDecision.MissingRealEvidence)
	}
	if report.LaunchReady || report.ServerSide.PRDAccepted {
		t.Fatalf("launch/server PRD = %v/%v, want wake proof to close only wake server-side gap", report.LaunchReady, report.ServerSide.PRDAccepted)
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"physical_acceptance_source_report": "a21-wake-word-physical-acceptance-20260602-040000.json"`,
		`"wake_word_physical_acceptance_available"`,
		`"physical_stackchan_online"`,
		`"physical_stackchan_prd_acceptance"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{packagePath, acceptancePath, dir, server.URL, "http://", "https://", "/Users/", `"launch_ready": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("wake physical readiness leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestProductReadinessRejectsWakeWordPhysicalAcceptanceWithoutPhysicalProof(t *testing.T) {
	server := newProductReadinessCustomWakeTestServer(t)
	stubWakeWordPhysicalAcceptanceSerialLister(t)
	dir := t.TempDir()
	packagePath := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-030000.json", productReadinessWakeWordFirmwarePackageReportFixtureJSON())
	data := strings.Replace(productReadinessWakeWordPhysicalAcceptanceReportFixtureJSON(), `"physical_device_online": true`, `"physical_device_online": false`, 1)
	acceptancePath := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-physical-acceptance-20260602-040000.json", data)
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")

	report := buildProductReadinessReport(context.Background(), productReadinessOptions{
		GatewayURL:                 server.URL,
		DeviceID:                   "stackchan-001",
		WakeWordFirmwarePackage:    packagePath,
		WakeWordPhysicalAcceptance: acceptancePath,
	}, []string{"A21_PROVIDER_PRIMARY=mock"})

	if report.WakeWord.ProductReady || report.ServerSide.WakeWordReady {
		t.Fatalf("wake readiness = %+v server_side=%+v, want physical proof required", report.WakeWord, report.ServerSide)
	}
	if !containsExactProductString(report.ServerSide.MissingEvidence, "wake_word") ||
		!containsExactProductString(report.CanonicalDecision.MissingRealEvidence, "wake_word_product_ready") {
		t.Fatalf("missing evidence = server %#v canonical %#v, want wake gap preserved", report.ServerSide.MissingEvidence, report.CanonicalDecision.MissingRealEvidence)
	}
	if !hasProductReadinessFinding(report.Findings, "wake_word_physical_acceptance_invalid") {
		t.Fatalf("findings = %#v, want wake_word_physical_acceptance_invalid", report.Findings)
	}
}

func TestProductReadinessRejectsWakeWordPhysicalAcceptanceMismatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string) string
	}{
		{
			name: "package artifact mismatch",
			mutate: func(data string) string {
				return strings.Replace(data, `a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-abcdef123456-20260602-030000.bin`, `a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-deadbee-20260602-030000.bin`, 1)
			},
		},
		{
			name: "current gateway config mismatch",
			mutate: func(data string) string {
				return strings.Replace(data, `"desired_phrase": "小阿二一"`, `"desired_phrase": "小阿二二"`, 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newProductReadinessCustomWakeTestServer(t)
			stubWakeWordPhysicalAcceptanceSerialLister(t)
			dir := t.TempDir()
			packagePath := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-030000.json", productReadinessWakeWordFirmwarePackageReportFixtureJSON())
			acceptancePath := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-physical-acceptance-20260602-040000.json", tt.mutate(productReadinessWakeWordPhysicalAcceptanceReportFixtureJSON()))

			report := buildProductReadinessReport(context.Background(), productReadinessOptions{
				GatewayURL:                 server.URL,
				DeviceID:                   "stackchan-001",
				WakeWordFirmwarePackage:    packagePath,
				WakeWordPhysicalAcceptance: acceptancePath,
			}, []string{"A21_PROVIDER_PRIMARY=mock"})

			if report.WakeWord.ProductReady || report.ServerSide.WakeWordReady {
				t.Fatalf("wake readiness = %+v server_side=%+v, want mismatch rejected", report.WakeWord, report.ServerSide)
			}
			if !hasProductReadinessFinding(report.Findings, "wake_word_physical_acceptance_mismatch") {
				t.Fatalf("findings = %#v, want wake_word_physical_acceptance_mismatch", report.Findings)
			}
		})
	}
}

func TestProductReadinessRejectsUnsafeWakeWordPhysicalAcceptanceWithoutLeak(t *testing.T) {
	server := newProductReadinessCustomWakeTestServer(t)
	stubWakeWordPhysicalAcceptanceSerialLister(t)
	dir := t.TempDir()
	packagePath := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-030000.json", productReadinessWakeWordFirmwarePackageReportFixtureJSON())
	data := strings.Replace(productReadinessWakeWordPhysicalAcceptanceReportFixtureJSON(), `"report_path": "a21-wake-word-physical-acceptance-20260602-040000.json"`, `"local_path": "/Users/private/a21/wake-proof.json",
  "api_key": "secret-token-value",
  "report_path": "a21-wake-word-physical-acceptance-20260602-040000.json"`, 1)
	acceptancePath := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-physical-acceptance-20260602-040000.json", data)
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"product-readiness",
		"--gateway-url", server.URL,
		"--wake-word-firmware-package-report", packagePath,
		"--wake-word-physical-acceptance-report", acceptancePath,
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"wake_word_physical_acceptance_invalid"`,
		`"wake_word_ready": false`,
		`"wake_word_product_ready"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{packagePath, acceptancePath, dir, server.URL, "/Users/private", "secret-token-value", "http://", "https://", `"wake_word_ready": true`, `"launch_ready": true`} {
		if strings.Contains(rendered, forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("unsafe wake physical acceptance leaked or overclaimed %q: stdout=%s stderr=%s", forbidden, rendered, stderr.String())
		}
	}
}

func TestProductReadinessUsesLatestMatchingWakeWordPhysicalAcceptance(t *testing.T) {
	server := newProductReadinessCustomWakeTestServer(t)
	stubWakeWordPhysicalAcceptanceSerialLister(t)
	dir := t.TempDir()
	writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-firmware-package-20260602-030000.json", productReadinessWakeWordFirmwarePackageReportFixtureJSON())
	matching := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-physical-acceptance-20260602-040000.json", productReadinessWakeWordPhysicalAcceptanceReportFixtureJSON())
	mismatched := writeProductReadinessReportFixtureFile(t, dir, "a21-wake-word-physical-acceptance-20260602-050000.json", strings.Replace(productReadinessWakeWordPhysicalAcceptanceReportFixtureJSON(), `"threshold": 35`, `"threshold": 45`, 1))
	oldTime := time.Unix(1780344000, 0)
	newTime := oldTime.Add(time.Hour)
	if err := os.Chtimes(matching, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(mismatched, newTime, newTime); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_PROVIDER_PRIMARY", "mock")
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
	if !report.WakeWord.ProductReady || !report.ServerSide.WakeWordReady {
		t.Fatalf("wake readiness = %+v server_side=%+v, want latest matching physical acceptance selected", report.WakeWord, report.ServerSide)
	}
	rendered := stdout.String()
	for _, want := range []string{
		`"physical_acceptance_source_report": "a21-wake-word-physical-acceptance-20260602-040000.json"`,
		`"detail": "wake_word_physical_acceptance:a21-wake-word-physical-acceptance-20260602-050000.json"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("stdout missing %q: %s", want, rendered)
		}
	}
	if strings.Contains(rendered, `"physical_acceptance_source_report": "a21-wake-word-physical-acceptance-20260602-050000.json"`) {
		t.Fatalf("selected stale/mismatched latest acceptance report: %s", rendered)
	}
}

func newProductReadinessCustomWakeTestServer(t *testing.T) *httptest.Server {
	t.Helper()
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
		case "/v1/voice-chain-profiles":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(productReadinessVoiceChainProfilesJSON("stepfun", nil)))
		case "/v1/roleplay-profile":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(productReadinessRoleplayProfileFixtureJSON("ready")))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func stubWakeWordPhysicalAcceptanceSerialLister(t *testing.T) {
	t.Helper()
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})
}

func hasProductReadinessFinding(findings []productReadinessFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func productReadinessWakeWordPhysicalProofReportFixtureJSON() string {
	return `{
  "schema_version": "a21.wake_word_physical_proof.v1",
  "generated_at_ms": 1780344000000,
  "status": "observed",
  "physical_device_online": true,
  "firmware_flash_executed": true,
  "guarded_flash_report_source": "a21-wake-word-guarded-flash-20260602-040000.json",
  "operator_custom_wake_observation_present": true,
  "wake_phrase_matched": true,
  "false_wake_accepted": false,
  "stock_wake_accepted": false,
  "redaction_ok": true,
  "report_path": "a21-wake-word-physical-proof-20260602-040000.json"
}`
}

func productReadinessWakeWordPhysicalAcceptanceReportFixtureJSON() string {
	return `{
  "schema_version": "a21.wake_word_physical_acceptance.v1",
  "generated_at_ms": 1780344000000,
  "status": "accepted",
  "product_ready": true,
  "mode": "custom_multinet",
  "desired_phrase": "小阿二一",
  "desired_pinyin": "xiao a er yi",
  "threshold": 35,
  "package_source_report": "a21-wake-word-firmware-package-20260602-030000.json",
  "package_artifact_name": "a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-abcdef123456-20260602-030000.bin",
  "package_manifest_name": "a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-abcdef123456-20260602-030000.bin.manifest.json",
  "physical_device_online": true,
  "firmware_flash_executed": true,
  "guarded_flash_report_source": "a21-wake-word-guarded-flash-20260602-040000.json",
  "operator_custom_wake_observation_present": true,
  "wake_phrase_matched": true,
  "false_wake_accepted": false,
  "stock_wake_accepted": false,
  "redaction_ok": true,
  "report_path": "a21-wake-word-physical-acceptance-20260602-040000.json"
}`
}
