package firmwarecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMicProbeFlashPlanConfirmsDiagnosticProbeWithoutAllowingFlash(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_mic_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	writeMicProbeFlashImages(t, buildDir, coreDir, artifact, []byte("diagnostic_probe_m5unified_i2s_capture"))

	result, err := BuildMicProbeFlashPlan(MicProbeFlashPlanOptions{
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
		t.Fatalf("BuildMicProbeFlashPlan returned error: %v", err)
	}
	if result.GuardID != "a21.firmware.mic_probe_flash_plan.v1" {
		t.Fatalf("GuardID = %q", result.GuardID)
	}
	if !result.DryRun || result.FlashAllowed {
		t.Fatalf("dry_run/flash_allowed = %v/%v, want true/false", result.DryRun, result.FlashAllowed)
	}
	if result.PlatformIOEnv != "a21_stackchan_cores3_mic_probe" || result.Commit != "abcdef123456" {
		t.Fatalf("unexpected probe identity: %#v", result)
	}
	if len(result.Parts) != 4 {
		t.Fatalf("parts = %d, want 4", len(result.Parts))
	}
}

func TestBuildMicProbeFlashPlanRejectsProductionBuildDirectory(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	writeMicProbeFlashImages(t, buildDir, coreDir, artifact, []byte("diagnostic_probe_m5unified_i2s_capture"))

	_, err := BuildMicProbeFlashPlan(MicProbeFlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true},
		BuildDir:          buildDir,
		CoreDir:           coreDir,
	})
	if err == nil {
		t.Fatal("expected production build dir to be rejected")
	}
	if !strings.Contains(err.Error(), "mic probe") {
		t.Fatalf("error = %q, want mic probe", err)
	}
}

func TestBuildMicProbeFlashPlanRejectsArtifactWithoutDiagnosticStatus(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_mic_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	writeMicProbeFlashImages(t, buildDir, coreDir, artifact, []byte("production build"))

	_, err := BuildMicProbeFlashPlan(MicProbeFlashPlanOptions{
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

func writeMicProbeFlashImages(t *testing.T, buildDir string, coreDir string, artifact string, marker []byte) {
	t.Helper()
	for path, content := range map[string][]byte{
		filepath.Join(buildDir, "bootloader.bin"): []byte("a21 bootloader"),
		filepath.Join(buildDir, "partitions.bin"): []byte("a21 partitions"),
		filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"): []byte("a21 boot app"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
		t.Fatal(err)
	}
	body := append([]byte("a21-stackchan\n0.1.0\nm5stack-cores3\nabcdef123456\n"), marker...)
	if err := os.WriteFile(artifact, body, 0o644); err != nil {
		t.Fatal(err)
	}
}
