package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/firmwarecheck"
)

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
