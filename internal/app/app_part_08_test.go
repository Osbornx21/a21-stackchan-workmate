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

	"a21.local/a21/internal/firmwarecheck"
)

func TestRunStackChanProductRecoveryRequiresROMDownloadWhenOfflineWithSerial(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.String())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/devices":
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": []
}`))
		case "/v1/stackchan/official/status":
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.stackchan.official.status.v1",
  "device_id": "44:1b:f6:e2:6a:60",
  "connected": false,
  "fallback_available": true,
  "delivered_transport": "xiaozhi_mcp_fallback_available",
  "physical_accepted": false,
  "next_action": "connect_official_stackchan_ws"
}`))
		default:
			t.Fatalf("unexpected product recovery request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	reportsDir := filepath.Join(tempDir, "reports")
	outputDir := filepath.Join(tempDir, "out")
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	serialPort := filepath.Join(tempDir, "cu.usbmodem1101")
	if err := os.WriteFile(serialPort, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	flashLog := filepath.Join(tempDir, "a21-official-xiaozhi-compatible-flash-20260605-074141.log")
	if err := os.WriteFile(flashLog, []byte(`Waiting for ESP32-S3 ROM download mode on `+serialPort+`
Timed out waiting for ESP32-S3 ROM download mode; hold BOOT, press/release RESET, keep holding BOOT, then retry.
Last esptool probe output:
A fatal error occurred: Failed to connect to ESP32-S3: No serial data received.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	writeProductReadinessReportFixtureFile(t, reportsDir, "a21-stackchan-official-xiaozhi-compatible-flash-20260605-074215-1780616535711678000.json", `{
  "schema_version": "a21.stackchan.official_xiaozhi_compatible_flash_execution.v1",
  "status": "failed",
  "flash_allowed": true,
  "flash_executed": false,
  "port": "`+serialPort+`",
  "esptool_before": "no_reset",
  "wait_rom_download_mode": true,
  "wait_rom_timeout_seconds": 30,
  "flash_log_file": "`+flashLog+`",
  "findings": [{"code": "flash_execute_failed", "message": "exit status 1"}]
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-accept",
		"--check", "product-recovery",
		"--gateway-url", server.URL,
		"--device-id", "44:1b:f6:e2:6a:60",
		"--upload-port", serialPort,
		"--serial-glob", filepath.Join(tempDir, "cu.usbmodem*"),
		"--reports-dir", reportsDir,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report stackChanProductRecoveryReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if report.Status != "product_offline_rom_download_required" {
		t.Fatalf("status = %q, want product_offline_rom_download_required: %+v", report.Status, report)
	}
	if !report.ROMDownloadRequired {
		t.Fatalf("ROMDownloadRequired = false, want true")
	}
	if !report.Serial.UploadPortPresent || len(report.Serial.Candidates) != 1 {
		t.Fatalf("serial report = %+v, want present candidate", report.Serial)
	}
	if report.LatestProductFlash == nil || report.LatestProductFlash.FlashExecuted {
		t.Fatalf("latest flash = %+v, want failed no-write report", report.LatestProductFlash)
	}
	if !report.LatestProductFlash.FlashLogEvidence.Found || !report.LatestProductFlash.FlashLogEvidence.ROMNoSerialData {
		t.Fatalf("flash log evidence = %+v, want no-serial ROM evidence", report.LatestProductFlash.FlashLogEvidence)
	}
	if !containsString(reportFindingCodes(report.Findings), "rom_probe_no_serial_data") {
		t.Fatalf("findings missing rom_probe_no_serial_data: %+v", report.Findings)
	}
	if !containsString(report.NextActions, "enter_esp32s3_rom_download_mode") {
		t.Fatalf("next actions missing ROM action: %+v", report.NextActions)
	}
	if report.ReportPath == "" {
		t.Fatalf("report path missing")
	}
	if _, err := os.Stat(report.ReportPath); err != nil {
		t.Fatalf("written report missing: %v", err)
	}
	if got := strings.Join(requests, "\n"); strings.Contains(got, "/control") {
		t.Fatalf("product recovery made control request: %s", got)
	}
}

func TestRunStackChanProductRecoveryReadyWhenOnlineAndOfficialRelayConnected(t *testing.T) {
	deviceID := "44:1b:f6:e2:6a:60"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/devices":
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [{
    "device_id": "` + deviceID + `",
    "identity_status": "ok",
    "connection_status": "online",
    "firmware": {
      "id": "a21-stackchan",
      "version": "0.1.0",
      "board": "m5stack-cores3",
      "commit": "abcdef1"
    },
    "runtime_echo": {
      "official_stackchan_auto_state": "speaking",
      "official_stackchan_packet_count": "3",
      "official_stackchan_surface_avatar": "delivered"
    },
    "last_event": "xiaozhi.hello"
  }]
}`))
		case "/v1/stackchan/official/status":
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.stackchan.official.status.v1",
  "device_id": "` + deviceID + `",
  "official_device_id": "` + deviceID + `",
  "connected": true,
  "connected_ms": 1200,
  "connected_since_ms": 1780610000000,
  "fallback_available": true,
  "delivered_transport": "stackchan_official_ws",
  "last_packet_count": 3,
  "physical_accepted": true,
  "official_action_surfaces": {"avatar": "delivered"},
  "next_action": "official_relay_ready"
}`))
		default:
			t.Fatalf("unexpected product recovery request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	reportsDir := filepath.Join(tempDir, "reports")
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProductReadinessReportFixtureFile(t, reportsDir, "a21-stackchan-official-xiaozhi-compatible-flash-20260605-060619-1780610779050566000.json", `{
  "schema_version": "a21.stackchan.official_xiaozhi_compatible_flash_execution.v1",
  "status": "passed",
  "flash_allowed": true,
  "flash_executed": true,
  "port": "/dev/cu.usbmodem1101"
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-product-recovery",
		"--gateway-url", server.URL,
		"--device-id", deviceID,
		"--serial-glob", filepath.Join(tempDir, "cu.usbmodem*"),
		"--reports-dir", reportsDir,
		"--output-dir", filepath.Join(tempDir, "out"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report stackChanProductRecoveryReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if report.Status != "product_online_official_relay_ready" {
		t.Fatalf("status = %q, want product_online_official_relay_ready: %+v", report.Status, report)
	}
	if !report.DeviceOnline || report.Device == nil {
		t.Fatalf("device online missing: %+v", report)
	}
	if !report.OfficialRelay.Checked || !report.OfficialRelay.Connected {
		t.Fatalf("official relay not connected: %+v", report.OfficialRelay)
	}
	if report.ROMDownloadRequired {
		t.Fatalf("ROMDownloadRequired = true, want false")
	}
}

func TestRunStackChanProductRecoveryPrefersOnlineCaseFoldedDevice(t *testing.T) {
	deviceID := "44:1b:f6:e2:6a:60"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/devices":
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [{
    "device_id": "44:1B:F6:E2:6A:60",
    "identity_status": "unknown",
    "connection_status": "stale",
    "device_age_ms": 400000,
    "last_seen_ms": 1780624371663,
    "last_event": "stackchan.official_ws.connected"
  }, {
    "device_id": "` + deviceID + `",
    "identity_status": "unknown",
    "connection_status": "online",
    "device_age_ms": 900,
    "last_seen_ms": 1780624802828,
    "last_event": "device.heartbeat"
  }]
}`))
		case "/v1/stackchan/official/status":
			if r.URL.Query().Get("device_id") != deviceID {
				t.Fatalf("device_id query = %q, want %q", r.URL.Query().Get("device_id"), deviceID)
			}
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.stackchan.official.status.v1",
  "device_id": "` + deviceID + `",
  "official_device_id": "` + deviceID + `",
  "connected": true,
  "fallback_available": true,
  "delivered_transport": "stackchan_official_ws",
  "physical_accepted": false,
  "next_action": "send_official_control_and_collect_physical_acceptance"
}`))
		default:
			t.Fatalf("unexpected product recovery request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	reportsDir := filepath.Join(tempDir, "reports")
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProductReadinessReportFixtureFile(t, reportsDir, "a21-stackchan-official-xiaozhi-compatible-flash-20260605-095236-1780624356871525000.json", `{
  "schema_version": "a21.stackchan.official_xiaozhi_compatible_flash_execution.v1",
  "status": "passed",
  "flash_allowed": true,
  "flash_executed": true,
  "port": "/dev/cu.usbmodem1101"
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-product-recovery",
		"--gateway-url", server.URL,
		"--device-id", deviceID,
		"--serial-glob", filepath.Join(tempDir, "cu.usbmodem*"),
		"--reports-dir", reportsDir,
		"--output-dir", filepath.Join(tempDir, "out"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report stackChanProductRecoveryReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if report.Status != "product_online_official_relay_ready" {
		t.Fatalf("status = %q, want product_online_official_relay_ready: %+v", report.Status, report)
	}
	if !report.DeviceOnline || report.Device == nil || report.Device.DeviceID != deviceID {
		t.Fatalf("device = %+v online=%v, want lowercase online record", report.Device, report.DeviceOnline)
	}
	if !report.OfficialRelay.Checked || !report.OfficialRelay.Connected {
		t.Fatalf("official relay = %+v, want connected", report.OfficialRelay)
	}
	if report.ROMDownloadRequired {
		t.Fatalf("ROMDownloadRequired = true, want false")
	}
}

func TestRunStackChanProductRecoveryRejectsInvalidDirectSourceIP(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-product-recovery",
		"--direct-source-ip", "not-an-ip",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--direct-source-ip must be a valid IP address") {
		t.Fatalf("stderr missing direct-source validation: %s", stderr.String())
	}
}

func TestRunStackChanProductRecoveryExecuteFlashRequiresConfirmationToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-product-recovery",
		"--execute-flash",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP") {
		t.Fatalf("stderr missing product flash confirmation token: %s", stderr.String())
	}
}

func TestRunStackChanProductRecoveryExecuteFlashSkipsWhenProductAlreadyOnline(t *testing.T) {
	originalRunner := runStackChanOfficialXiaozhiCompatibleFlashCommand
	runStackChanOfficialXiaozhiCompatibleFlashCommand = func(ctx context.Context, logPath string, script string) error {
		t.Fatalf("flash runner should not be called when product is already online")
		return nil
	}
	defer func() {
		runStackChanOfficialXiaozhiCompatibleFlashCommand = originalRunner
	}()

	deviceID := "44:1b:f6:e2:6a:60"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/devices":
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [{
    "device_id": "` + deviceID + `",
    "identity_status": "ok",
    "connection_status": "online",
    "firmware": {"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"}
  }]
}`))
		case "/v1/stackchan/official/status":
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.stackchan.official.status.v1",
  "device_id": "` + deviceID + `",
  "connected": false,
  "fallback_available": true,
  "delivered_transport": "xiaozhi_mcp_fallback_available",
  "physical_accepted": false,
  "next_action": "connect_official_stackchan_ws"
}`))
		default:
			t.Fatalf("unexpected product recovery request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-product-recovery",
		"--gateway-url", server.URL,
		"--device-id", deviceID,
		"--execute-flash",
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP",
		"--output-dir", filepath.Join(tempDir, "out"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report stackChanProductRecoveryExecutionReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode execution report: %v\n%s", err, stdout.String())
	}
	if report.Status != "product_online_flash_skipped" {
		t.Fatalf("status = %q, want product_online_flash_skipped: %+v", report.Status, report)
	}
	if report.Flash != nil {
		t.Fatalf("flash report present despite online skip: %+v", report.Flash)
	}
	if !containsString(reportFindingCodes(report.Findings), "product_already_online") {
		t.Fatalf("findings missing product_already_online: %+v", report.Findings)
	}
}

func TestRunStackChanProductRecoveryExecuteFlashRunsOfficialWaitROMAndPostCheck(t *testing.T) {
	allowA21ControlGuardForTest(t)
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runStackChanOfficialXiaozhiCompatibleFlashCommand
	var ranScript string
	runStackChanOfficialXiaozhiCompatibleFlashCommand = func(ctx context.Context, logPath string, script string) error {
		ranScript = script
		return nil
	}
	defer func() {
		runStackChanOfficialXiaozhiCompatibleFlashCommand = originalRunner
	}()

	deviceID := "44:1b:f6:e2:6a:60"
	deviceRequests := 0
	officialRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/devices":
			deviceRequests++
			if deviceRequests == 1 {
				_, _ = w.Write([]byte(`{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [{
    "device_id": "` + deviceID + `",
    "identity_status": "ok",
    "connection_status": "online",
    "firmware": {"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},
    "last_event": "xiaozhi.hello"
  }]
}`))
		case "/v1/stackchan/official/status":
			officialRequests++
			if officialRequests == 1 {
				_, _ = w.Write([]byte(`{
  "schema_version": "a21.stackchan.official.status.v1",
  "device_id": "` + deviceID + `",
  "connected": false,
  "fallback_available": true,
  "delivered_transport": "xiaozhi_mcp_fallback_available",
  "physical_accepted": false,
  "next_action": "connect_official_stackchan_ws"
}`))
				return
			}
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.stackchan.official.status.v1",
  "device_id": "` + deviceID + `",
  "official_device_id": "` + deviceID + `",
  "connected": true,
  "fallback_available": true,
  "delivered_transport": "stackchan_official_ws",
  "last_packet_count": 2,
  "physical_accepted": false,
  "next_action": "send_official_control_and_collect_physical_acceptance"
}`))
		default:
			t.Fatalf("unexpected product recovery request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "out")
	serialPort := filepath.Join(tempDir, "cu.usbmodemA21")
	writeTestFile(t, serialPort, "")
	buildDir := writeTestOfficialXiaozhiCompatibleBuild(t)
	idfExport := filepath.Join(tempDir, "export.sh")
	writeTestFile(t, idfExport, "#!/bin/sh\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-product-recovery",
		"--gateway-url", server.URL,
		"--device-id", deviceID,
		"--upload-port", serialPort,
		"--serial-glob", filepath.Join(tempDir, "cu.usbmodem*"),
		"--reports-dir", outputDir,
		"--output-dir", outputDir,
		"--execute-flash",
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP",
		"--build-dir", buildDir,
		"--idf-export", idfExport,
		"--wait-rom-timeout-seconds", "75",
		"--post-check-delay-ms", "0",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"A21_FLASH_PORT='" + serialPort + "'",
		"A21_WAIT_ROM_DEADLINE=$((SECONDS + 75))",
		`python -m esptool --chip esp32s3 --port "$candidate" -b 115200 --before no_reset --after no_reset --no-stub chip_id`,
		`--port "${A21_FLASH_PORT}"`,
		"--before 'no_reset'",
		"write_flash @flash_args",
	} {
		if !strings.Contains(ranScript, want) {
			t.Fatalf("flash script missing %q: %s", want, ranScript)
		}
	}
	var report stackChanProductRecoveryExecutionReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode execution report: %v\n%s", err, stdout.String())
	}
	if report.Status != "flash_passed_product_online_official_relay_ready" {
		t.Fatalf("status = %q, want flash_passed_product_online_official_relay_ready: %+v", report.Status, report)
	}
	if report.Precheck.Status != "product_offline_recovery_required" && report.Precheck.Status != "product_offline_rom_download_required" {
		t.Fatalf("precheck status = %q, want offline recovery status", report.Precheck.Status)
	}
	if report.Flash == nil || !report.Flash.FlashExecuted || !report.Flash.WaitROM || report.Flash.EsptoolBefore != "no_reset" {
		t.Fatalf("flash report = %+v, want executed wait-ROM no_reset product flash", report.Flash)
	}
	if report.PostCheck == nil || !report.PostCheck.DeviceOnline || !report.PostCheck.OfficialRelay.Connected {
		t.Fatalf("postcheck = %+v, want online official relay", report.PostCheck)
	}
	for _, forbidden := range []string{"xiaozhi.bin", buildDir, idfExport} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("execution report leaked forbidden detail %q: %s", forbidden, stdout.String())
		}
	}
}

func reportFindingCodes(findings []stackChanProductRecoveryFinding) []string {
	codes := make([]string, 0, len(findings))
	for _, finding := range findings {
		codes = append(codes, finding.Code)
	}
	return codes
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

func TestGatewayServerOptionsFromEnvCloudEdgeProductChainDoesNotDefaultToLocalSherpa(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_XIAOZHI_PRODUCT_CHAIN=cloud_edge",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_ASR_MODEL=doubao-asr",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
		"A21_LAB_DEEPSEEK_API_KEY=sk-a21-deepseek-secret",
	})
	if options.XiaozhiVoicePipelineAdapters == nil {
		t.Fatal("xiaozhi voice pipeline adapters not configured")
	}
	if got := appEnvValue(options.CloudVoiceEnv, "A21_XIAOZHI_FAST_ACK_ENABLED"); got != "true" {
		t.Fatalf("A21_XIAOZHI_FAST_ACK_ENABLED = %q, want true for cloud_edge delayed fast ack", got)
	}
	if got := appEnvValue(options.CloudVoiceEnv, "A21_XIAOZHI_FAST_ACK_DELAY_MS"); got != "700" {
		t.Fatalf("A21_XIAOZHI_FAST_ACK_DELAY_MS = %q, want 700 for cloud_edge product chain", got)
	}
	if got := appEnvValue(options.CloudVoiceEnv, "A21_XIAOZHI_STT_SCREEN_POLICY"); got != "status_only" {
		t.Fatalf("A21_XIAOZHI_STT_SCREEN_POLICY = %q, want status_only for cloud_edge product chain", got)
	}
	adapters := *options.XiaozhiVoicePipelineAdapters
	if adapters.ExecutionMode != "cloud_edge" {
		t.Fatalf("execution mode = %q, want cloud_edge", adapters.ExecutionMode)
	}
	if adapters.ASR.Name() != "doubao_asr_realtime" || adapters.TextStream.Name() != "deepseek" || adapters.TTS.Name() != "doubao_tts_realtime" {
		t.Fatalf("adapters = %s/%s/%s, want doubao_asr_realtime/deepseek/doubao_tts_realtime", adapters.ASR.Name(), adapters.TextStream.Name(), adapters.TTS.Name())
	}
	selectionBytes, err := json.Marshal(adapters.Selection)
	if err != nil {
		t.Fatal(err)
	}
	selectionPayload := string(selectionBytes)
	for _, want := range []string{
		`"asr_mode":"cloud"`,
		`"asr_profile":"doubao_asr_realtime"`,
		`"asr_profile_env":"A21_ASR_CLOUD_PROFILE"`,
		`"llm_profile":"deepseek"`,
		`"tts_profile":"doubao_tts_realtime"`,
	} {
		if !strings.Contains(selectionPayload, want) {
			t.Fatalf("selection missing %q: %s", want, selectionPayload)
		}
	}
	for _, forbidden := range []string{"sherpa_onnx", "sk-a21-secret", "doubao-asr", "doubao-tts", "zh_female"} {
		if strings.Contains(selectionPayload, forbidden) {
			t.Fatalf("selection leaked or kept local default %q: %s", forbidden, selectionPayload)
		}
	}
}

func TestGatewayServerOptionsFromEnvCloudEdgePrefersDashScopeWhenConfigured(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_XIAOZHI_PRODUCT_CHAIN=cloud_edge",
		"A21_DASHSCOPE_API_KEY=sk-a21-dashscope-secret",
		"A21_DASHSCOPE_ASR_MODEL=qwen3-asr-flash-realtime-secret",
		"A21_DASHSCOPE_TTS_MODEL=qwen3-tts-flash-realtime-secret",
		"A21_DASHSCOPE_TTS_VOICE=CherrySecret",
		"A21_LAB_DEEPSEEK_API_KEY=sk-a21-deepseek-secret",
	})
	if options.XiaozhiVoicePipelineAdapters == nil {
		t.Fatal("xiaozhi voice pipeline adapters not configured")
	}
	adapters := *options.XiaozhiVoicePipelineAdapters
	if adapters.ExecutionMode != "cloud_edge" {
		t.Fatalf("execution mode = %q, want cloud_edge", adapters.ExecutionMode)
	}
	if adapters.ASR.Name() != "dashscope_qwen_asr_realtime" ||
		adapters.TextStream.Name() != "deepseek" ||
		adapters.TTS.Name() != "dashscope_qwen_tts_realtime" {
		t.Fatalf("adapters = %s/%s/%s, want dashscope_qwen_asr_realtime/deepseek/dashscope_qwen_tts_realtime", adapters.ASR.Name(), adapters.TextStream.Name(), adapters.TTS.Name())
	}
	selectionBytes, err := json.Marshal(adapters.Selection)
	if err != nil {
		t.Fatal(err)
	}
	selectionPayload := string(selectionBytes)
	for _, want := range []string{
		`"asr_mode":"cloud"`,
		`"asr_profile":"dashscope_qwen_asr_realtime"`,
		`"asr_profile_env":"A21_ASR_CLOUD_PROFILE"`,
		`"llm_profile":"deepseek"`,
		`"tts_profile":"dashscope_qwen_tts_realtime"`,
	} {
		if !strings.Contains(selectionPayload, want) {
			t.Fatalf("selection missing %q: %s", want, selectionPayload)
		}
	}
	for _, forbidden := range []string{"sherpa_onnx", "sk-a21", "qwen3-asr", "qwen3-tts", "CherrySecret"} {
		if strings.Contains(selectionPayload, forbidden) {
			t.Fatalf("selection leaked or kept local default %q: %s", forbidden, selectionPayload)
		}
	}
}

func TestGatewayServerOptionsFromEnvCloudEdgePrefersStepFunOverDeepSeekWhenConfigured(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_XIAOZHI_PRODUCT_CHAIN=cloud_edge",
		"A21_DASHSCOPE_API_KEY=sk-a21-dashscope-secret",
		"A21_LAB_STEPFUN_API_KEY=sk-a21-stepfun-secret",
		"A21_STEPFUN_MODEL=step-1-8k-secret",
		"A21_LAB_DEEPSEEK_API_KEY=sk-a21-deepseek-secret",
	})
	if options.XiaozhiVoicePipelineAdapters == nil {
		t.Fatal("xiaozhi voice pipeline adapters not configured")
	}
	adapters := *options.XiaozhiVoicePipelineAdapters
	if adapters.TextStream.Name() != "stepfun" {
		t.Fatalf("text stream = %q, want stepfun", adapters.TextStream.Name())
	}
	selectionBytes, err := json.Marshal(adapters.Selection)
	if err != nil {
		t.Fatal(err)
	}
	selectionPayload := string(selectionBytes)
	for _, want := range []string{
		`"llm_profile":"stepfun"`,
		`"llm_profile_env":"A21_TEXT_STREAM_PROFILE"`,
	} {
		if !strings.Contains(selectionPayload, want) {
			t.Fatalf("selection missing %q: %s", want, selectionPayload)
		}
	}
	for _, forbidden := range []string{"deepseek", "sk-a21", "step-1-8k-secret"} {
		if strings.Contains(selectionPayload, forbidden) {
			t.Fatalf("selection leaked or chose wrong provider %q: %s", forbidden, selectionPayload)
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

func TestGatewayServerOptionsFromEnvWiresProductPlaybackEvents(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS=true",
	})
	if !options.XiaozhiProductPlaybackEvents {
		t.Fatal("xiaozhi product playback events not configured")
	}
}

func TestGatewayServerOptionsFromEnvWiresProductTouchEvents(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_XIAOZHI_PRODUCT_TOUCH_EVENTS=true",
	})
	if !options.XiaozhiProductTouchEvents {
		t.Fatal("xiaozhi product touch events not configured")
	}
}

func TestGatewayServerOptionsFromEnvWiresProductTouchReactions(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS=true",
	})
	if !options.XiaozhiProductTouchReactions {
		t.Fatal("xiaozhi product touch reactions not configured")
	}
}

func TestGatewayServerOptionsFromEnvWiresProductStateReactions(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_XIAOZHI_PRODUCT_STATE_REACTIONS=true",
	})
	if !options.XiaozhiProductStateReactions {
		t.Fatal("xiaozhi product state reactions not configured")
	}
}
