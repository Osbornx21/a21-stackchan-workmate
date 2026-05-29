package firmwarecheck

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateArtifactRequiresEmbeddedA21Identity(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeArtifactWithChecksum(t, artifact, []byte("a21-stackchan 0.1.0 m5stack-cores3"))

	_, err := ValidateArtifact(ArtifactOptions{ManifestPath: manifest, ArtifactPath: artifact})
	if err == nil {
		t.Fatal("expected artifact without embedded commit identity to be rejected")
	}
	if !strings.Contains(err.Error(), "embedded") || !strings.Contains(err.Error(), "commit") {
		t.Fatalf("error = %q, want embedded commit identity", err)
	}
}

func TestValidateArtifactAcceptsEmbeddedA21Identity(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeArtifactWithChecksum(t, artifact, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef123456"))

	result, err := ValidateArtifact(ArtifactOptions{ManifestPath: manifest, ArtifactPath: artifact})
	if err != nil {
		t.Fatalf("ValidateArtifact returned error: %v", err)
	}
	if !result.OK {
		t.Fatal("OK = false, want true")
	}
}

func writeArtifactManifest(t *testing.T, dir string) string {
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

func writeArtifactWithChecksum(t *testing.T, artifactPath string, content []byte) {
	t.Helper()
	if err := os.WriteFile(artifactPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	checksum := hex.EncodeToString(sum[:])
	if err := os.WriteFile(artifactPath+".sha256", []byte(checksum+"  "+filepath.Base(artifactPath)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
