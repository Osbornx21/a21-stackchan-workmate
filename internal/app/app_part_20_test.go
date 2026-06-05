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
)

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
