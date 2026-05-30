package firmwarecheck

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSensorProbeFlashPlanConfirmsDiagnosticProbeWithoutAllowingFlash(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_sensor_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	writeMicProbeFlashImages(t, buildDir, coreDir, artifact, []byte("diagnostic_probe_ltr553_ambient_light\ndiagnostic_probe_ltr553_proximity\ndiagnostic_probe_ina226_battery\n"))

	result, err := BuildSensorProbeFlashPlan(SensorProbeFlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true},
		BuildDir:          buildDir,
		CoreDir:           coreDir,
		NowMS:             123456789,
	})
	if err != nil {
		t.Fatalf("BuildSensorProbeFlashPlan returned error: %v", err)
	}
	if result.GuardID != "a21.firmware.sensor_probe_flash_plan.v1" {
		t.Fatalf("GuardID = %q", result.GuardID)
	}
	if !result.DryRun || result.FlashAllowed {
		t.Fatalf("dry_run/flash_allowed = %v/%v, want true/false", result.DryRun, result.FlashAllowed)
	}
	if result.PlatformIOEnv != "a21_stackchan_cores3_sensor_probe" || result.Commit != "abcdef123456" {
		t.Fatalf("unexpected probe identity: %#v", result)
	}
	if len(result.Parts) != 4 {
		t.Fatalf("parts = %d, want 4", len(result.Parts))
	}
}

func TestBuildSensorProbeFlashPlanRejectsIMUProbeBuildDirectory(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_imu_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	writeMicProbeFlashImages(t, buildDir, coreDir, artifact, []byte("diagnostic_probe_ltr553_ambient_light\ndiagnostic_probe_ltr553_proximity\ndiagnostic_probe_ina226_battery\n"))

	_, err := BuildSensorProbeFlashPlan(SensorProbeFlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true},
		BuildDir:          buildDir,
		CoreDir:           coreDir,
	})
	if err == nil {
		t.Fatal("expected IMU probe build dir to be rejected")
	}
	if !strings.Contains(err.Error(), "sensor probe") {
		t.Fatalf("error = %q, want sensor probe", err)
	}
}

func TestBuildSensorProbeFlashPlanRejectsArtifactWithoutEveryDiagnosticStatus(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_sensor_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	writeMicProbeFlashImages(t, buildDir, coreDir, artifact, []byte("diagnostic_probe_ltr553_ambient_light\ndiagnostic_probe_ltr553_proximity\n"))

	_, err := BuildSensorProbeFlashPlan(SensorProbeFlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true},
		BuildDir:          buildDir,
		CoreDir:           coreDir,
	})
	if err == nil {
		t.Fatal("expected missing diagnostic status to be rejected")
	}
	if !strings.Contains(err.Error(), "diagnostic") {
		t.Fatalf("error = %q, want diagnostic", err)
	}
}
