package firmwarecheck

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateDeviceIdentityConfirmsMatchingGatewayReport(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))
	report := writeDeviceIdentityReport(t, dir, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef123456"
      },
      "last_seen_ms": 1780000000000
    }
  ]
}`)

	result, err := ValidateDeviceIdentity(DeviceIdentityOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		ReportPath:        report,
		ExpectedDeviceID:  "stackchan-001",
		ExpectedGitCommit: "abcdef123456",
	})
	if err != nil {
		t.Fatalf("ValidateDeviceIdentity returned error: %v", err)
	}
	if !result.OK {
		t.Fatal("OK = false, want true")
	}
	if !result.DeviceIdentityConfirmed {
		t.Fatal("DeviceIdentityConfirmed = false, want true")
	}
	if result.FlashAllowed {
		t.Fatal("FlashAllowed = true, want false")
	}
	if result.NextRequiredConfirmation != "explicit_guarded_flash_command" {
		t.Fatalf("NextRequiredConfirmation = %q", result.NextRequiredConfirmation)
	}
	if result.Device.DeviceID != "stackchan-001" {
		t.Fatalf("DeviceID = %q", result.Device.DeviceID)
	}
}

func TestValidateDeviceIdentityRejectsMismatchedDeviceCommit(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))
	report := writeDeviceIdentityReport(t, dir, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "123456abcdef"
      }
    }
  ]
}`)

	_, err := ValidateDeviceIdentity(DeviceIdentityOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		ReportPath:        report,
		ExpectedDeviceID:  "stackchan-001",
		ExpectedGitCommit: "abcdef123456",
	})
	if err == nil {
		t.Fatal("expected mismatched device commit to be rejected")
	}
	if !strings.Contains(err.Error(), "commit") {
		t.Fatalf("error = %q, want commit", err)
	}
}

func TestValidateDeviceIdentityRejectsLegacyDeviceID(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))
	report := writeDeviceIdentityReport(t, dir, `{
  "devices": [
    {
      "device_id": "x21-stackchan",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef123456"
      }
    }
  ]
}`)

	_, err := ValidateDeviceIdentity(DeviceIdentityOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		ReportPath:        report,
		ExpectedDeviceID:  "x21-stackchan",
		ExpectedGitCommit: "abcdef123456",
	})
	if err == nil {
		t.Fatal("expected legacy device id to be rejected")
	}
	if !strings.Contains(err.Error(), "legacy") {
		t.Fatalf("error = %q, want legacy", err)
	}
}

func writeDeviceIdentityManifest(t *testing.T, dir string) string {
	t.Helper()
	manifest := filepath.Join(dir, "a21-firmware.json")
	if err := os.WriteFile(manifest, []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(`[platformio]
default_envs = a21_stackchan_cores3

[env:a21_stackchan_cores3]
platform = espressif32@7.0.1
board = m5stack-cores3
framework = arduino
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
extra_scripts =
  pre:scripts/a21_block_raw_upload.py
  pre:scripts/a21_build_identity.py
lib_deps =
  m5stack/M5Unified @ 0.2.16
  bblanchon/ArduinoJson @ 7.4.3
  links2004/WebSockets @ 2.7.3

[env:a21_stackchan_native]
platform = native
test_framework = unity
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
extra_scripts =
  pre:scripts/a21_block_raw_upload.py
  pre:scripts/a21_build_identity.py
lib_deps =
  bblanchon/ArduinoJson @ 7.4.3
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func writeDeviceIdentityReport(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "devices.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeDeviceIdentityArtifact(t *testing.T, artifactPath string, content []byte) {
	t.Helper()
	commit := deviceIdentityArtifactCommitFromName(t, artifactPath)
	content = append(content, []byte("\na21-stackchan\n0.1.0\nm5stack-cores3\n"+commit+"\n")...)
	if err := os.WriteFile(artifactPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	checksum := hex.EncodeToString(sum[:])
	if err := os.WriteFile(artifactPath+".sha256", []byte(checksum+"  "+filepath.Base(artifactPath)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func deviceIdentityArtifactCommitFromName(t *testing.T, artifactPath string) string {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(artifactPath), ".bin")
	parts := strings.Split(name, "-")
	if len(parts) < 4 {
		t.Fatalf("artifact name %q lacks commit field", name)
	}
	return parts[len(parts)-3]
}
