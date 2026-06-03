package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/runtimeguard"
)

func TestBuildFirmwareDoctorReportUsesExecutedOfficialXiaozhiCompatibleEvidenceOverNewerDryRun(t *testing.T) {
	root := t.TempDir()
	writeTestDoctorFirmwareRoot(t, root)
	reportsDir := filepath.Join(root, "reports")
	executedPath := writeOfficialXiaozhiCompatibleFlashEvidenceReport(t, reportsDir, "20260603-170812", map[string]any{
		"schema_version":     stackChanOfficialXiaozhiCompatibleFlashExecutionSchema,
		"generated_at_ms":    int64(1000),
		"status":             "passed",
		"firmware_candidate": "a21-stackchan-official-xiaozhi-compatible",
		"build_lane_role":    "product_candidate",
		"dry_run":            false,
		"flash_allowed":      true,
		"flash_executed":     true,
		"parts": []map[string]any{{
			"name":       "app",
			"offset":     "0x20000",
			"file":       "a21-stackchan-official-xiaozhi-compatible.bin",
			"sha256":     strings.Repeat("a", 64),
			"size_bytes": int64(123),
		}},
	})
	writeOfficialXiaozhiCompatibleFlashEvidenceReport(t, reportsDir, "20260604-090000", map[string]any{
		"schema_version":     stackChanOfficialXiaozhiCompatibleFlashPlanSchema,
		"generated_at_ms":    int64(2000),
		"status":             "ready",
		"firmware_candidate": "a21-stackchan-official-xiaozhi-compatible",
		"build_lane_role":    "product_candidate",
		"dry_run":            true,
		"flash_allowed":      false,
		"flash_executed":     false,
		"parts": []map[string]any{{
			"name":       "app",
			"offset":     "0x20000",
			"file":       "a21-stackchan-official-xiaozhi-compatible.bin",
			"sha256":     strings.Repeat("b", 64),
			"size_bytes": int64(123),
		}},
	})

	report := buildFirmwareDoctorReport(root, "abcdef1")
	var encoded bytes.Buffer
	if err := json.NewEncoder(&encoded).Encode(report); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		`"product_lane_artifact_evidence"`,
		`"status":"satisfied"`,
		`"source":"official_xiaozhi_compatible_flash_execution"`,
		`"artifact_file":"a21-stackchan-official-xiaozhi-compatible.bin"`,
		`"artifact_sha256":"` + strings.Repeat("a", 64) + `"`,
		`"report_path":"` + filepath.ToSlash(executedPath) + `"`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("doctor firmware report missing %q: %s", want, encoded.String())
		}
	}
	if strings.Contains(encoded.String(), "firmware_current_artifact_missing") {
		t.Fatalf("doctor should not warn current artifact missing when executed product-lane evidence is present: %s", encoded.String())
	}
	if strings.Contains(encoded.String(), strings.Repeat("b", 64)) {
		t.Fatalf("newer dry-run plan overrode executed evidence: %s", encoded.String())
	}
}

func TestBuildFirmwareDoctorReportDoesNotAcceptGenericXiaozhiBinAsProductLaneEvidence(t *testing.T) {
	root := t.TempDir()
	writeTestDoctorFirmwareRoot(t, root)
	writeOfficialXiaozhiCompatibleFlashEvidenceReport(t, filepath.Join(root, "reports"), "20260603-170812", map[string]any{
		"schema_version":     stackChanOfficialXiaozhiCompatibleFlashExecutionSchema,
		"generated_at_ms":    int64(1000),
		"status":             "passed",
		"firmware_candidate": "a21-stackchan-official-xiaozhi-compatible",
		"build_lane_role":    "product_candidate",
		"dry_run":            false,
		"flash_allowed":      true,
		"flash_executed":     true,
		"parts": []map[string]any{{
			"name":       "app",
			"offset":     "0x20000",
			"file":       "xiaozhi.bin",
			"sha256":     strings.Repeat("c", 64),
			"size_bytes": int64(123),
		}},
	})

	report := buildFirmwareDoctorReport(root, "abcdef1")
	var encoded bytes.Buffer
	if err := json.NewEncoder(&encoded).Encode(report); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(encoded.String(), `"product_lane_artifact_evidence"`) {
		t.Fatalf("generic xiaozhi.bin must not satisfy product-lane evidence: %s", encoded.String())
	}
	if !strings.Contains(encoded.String(), "firmware_current_artifact_missing") {
		t.Fatalf("missing current artifact warning should remain for generic xiaozhi.bin evidence: %s", encoded.String())
	}
}

func TestBuildOfficePreflightReportAnnotatesOfficialXiaozhiCompatibleEvidenceWithoutFlashReadiness(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.usbmodemA21",
			USBModem: true,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	t.Cleanup(func() {
		listFirmwareSerialDevices = originalLister
	})

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	reportsDir := filepath.Join(dir, "reports")
	writeOfficialXiaozhiCompatibleFlashEvidenceReport(t, reportsDir, "20260603-170812", map[string]any{
		"schema_version":     stackChanOfficialXiaozhiCompatibleFlashExecutionSchema,
		"generated_at_ms":    int64(1000),
		"status":             "passed",
		"firmware_candidate": "a21-stackchan-official-xiaozhi-compatible",
		"build_lane_role":    "product_candidate",
		"dry_run":            false,
		"flash_allowed":      true,
		"flash_executed":     true,
		"parts": []map[string]any{{
			"name":       "app",
			"offset":     "0x20000",
			"file":       "a21-stackchan-official-xiaozhi-compatible.bin",
			"sha256":     strings.Repeat("d", 64),
			"size_bytes": int64(123),
		}},
	})
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

	report := buildOfficePreflightReport(officePreflightOptions{
		ManifestPath:   manifest,
		ArtifactDir:    filepath.Join(dir, "artifacts"),
		GatewayURL:     server.URL,
		DeviceID:       "stackchan-001",
		Commit:         "abcdef1",
		MaxDeviceAgeMS: 300000,
		OutputDir:      reportsDir,
	})
	var encoded bytes.Buffer
	if err := writeJSONOfficePreflight(&encoded, report); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		`"ready_for_flash_plan": false`,
		`"flash_allowed": false`,
		`"product_lane_artifact_evidence"`,
		`"status": "satisfied"`,
		`"artifact_file": "a21-stackchan-official-xiaozhi-compatible.bin"`,
		`"code": "official_product_lane_artifact_evidence_present"`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("office preflight missing %q: %s", want, encoded.String())
		}
	}
	if strings.Contains(encoded.String(), `"code": "current_artifact_invalid"`) {
		t.Fatalf("office preflight should annotate official evidence instead of reporting current_artifact_invalid: %s", encoded.String())
	}
}

func TestBuildDoctorReportKeepsWakeWordFirmwareBuildRequiredWithOfficialProductLaneEvidence(t *testing.T) {
	root := t.TempDir()
	writeTestDoctorFirmwareRoot(t, root)
	writeOfficialXiaozhiCompatibleFlashEvidenceReport(t, filepath.Join(root, "reports"), "20260603-170812", map[string]any{
		"schema_version":     stackChanOfficialXiaozhiCompatibleFlashExecutionSchema,
		"generated_at_ms":    int64(1000),
		"status":             "passed",
		"firmware_candidate": "a21-stackchan-official-xiaozhi-compatible",
		"build_lane_role":    "product_candidate",
		"dry_run":            false,
		"flash_allowed":      true,
		"flash_executed":     true,
		"parts": []map[string]any{{
			"name":       "app",
			"offset":     "0x20000",
			"file":       "a21-stackchan-official-xiaozhi-compatible.bin",
			"sha256":     strings.Repeat("e", 64),
			"size_bytes": int64(123),
		}},
	})
	configPath := filepath.Join(root, "a21-wake-word.json")
	if err := os.WriteFile(configPath, []byte(`{
		"mode":"custom_multinet",
		"desired_phrase":"小阿二一",
		"desired_pinyin":"xiao a er yi",
		"threshold":35
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_WAKE_WORD_CONFIG_PATH", configPath)

	report := buildDoctorReport(runtimeguard.PreflightReport{}, root, "abcdef1")
	var encoded bytes.Buffer
	if err := writeJSONDoctorReport(&encoded, report); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		`"product_lane_artifact_evidence"`,
		`"status": "satisfied"`,
		`"wake_word_firmware_build_required"`,
		`"firmware_build_required": true`,
	} {
		if !strings.Contains(encoded.String(), want) {
			t.Fatalf("doctor report missing %q: %s", want, encoded.String())
		}
	}
}

func writeTestDoctorFirmwareRoot(t *testing.T, root string) {
	t.Helper()
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
}

func writeOfficialXiaozhiCompatibleFlashEvidenceReport(t *testing.T, reportsDir string, stamp string, report map[string]any) string {
	t.Helper()
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(reportsDir, "a21-stackchan-official-xiaozhi-compatible-flash-"+stamp+".json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
