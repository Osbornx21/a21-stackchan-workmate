package firmwarecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildBootstrapFlashPlanConfirmsImagesWithoutAllowingFlash(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))
	buildDir := filepath.Join(dir, "build")
	coreDir := filepath.Join(dir, "core")
	writeFile(t, filepath.Join(buildDir, "bootloader.bin"), []byte("bootloader"))
	writeFile(t, filepath.Join(buildDir, "partitions.bin"), []byte("partitions"))
	writeFile(t, filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"), []byte("boot-app"))

	result, err := BuildBootstrapFlashPlan(BootstrapFlashPlanOptions{
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
		t.Fatalf("BuildBootstrapFlashPlan returned error: %v", err)
	}
	if result.GuardID != "a21.firmware.bootstrap_flash_plan.v1" {
		t.Fatalf("GuardID = %q", result.GuardID)
	}
	if !result.DryRun || result.FlashAllowed {
		t.Fatalf("dry_run/flash_allowed = %v/%v, want true/false", result.DryRun, result.FlashAllowed)
	}
	if result.Port != "/dev/cu.usbmodemA21" || result.ArtifactSHA256 == "" || result.Commit != "abcdef123456" {
		t.Fatalf("unexpected identity fields: %#v", result)
	}
	if len(result.Parts) != 4 {
		t.Fatalf("parts = %d, want 4: %#v", len(result.Parts), result.Parts)
	}
	offsets := []string{"0x0000", "0x8000", "0xe000", "0x10000"}
	for i, want := range offsets {
		if result.Parts[i].Offset != want || result.Parts[i].SHA256 == "" {
			t.Fatalf("part %d = %#v, want offset %s and sha", i, result.Parts[i], want)
		}
	}
}

func TestBuildBootstrapFlashPlanRejectsMissingImagePart(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))

	_, err := BuildBootstrapFlashPlan(BootstrapFlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true},
		BuildDir:          filepath.Join(dir, "missing-build"),
		CoreDir:           filepath.Join(dir, "missing-core"),
	})
	if err == nil {
		t.Fatal("expected missing image part to be rejected")
	}
	if !strings.Contains(err.Error(), "bootstrap flash image") {
		t.Fatalf("error = %q, want bootstrap image error", err)
	}
}

func TestBuildBootstrapFlashPlanRejectsBusyPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))

	_, err := BuildBootstrapFlashPlan(BootstrapFlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true, InUse: true, Detail: "p1234 monitor"},
	})
	if err == nil {
		t.Fatal("expected busy port to be rejected")
	}
	if !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("error = %q, want already in use", err)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
