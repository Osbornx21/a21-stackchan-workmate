package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/firmwarecheck"
)

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
