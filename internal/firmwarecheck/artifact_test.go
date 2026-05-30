package firmwarecheck

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func TestValidateUploadCandidateRequiresReleaseIndexEntry(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeArtifactWithChecksum(t, artifact, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef123456"))

	_, err := ValidateUploadCandidate(UploadCheckOptions{
		ManifestPath: manifest,
		ArtifactPath: artifact,
		Port:         "/dev/cu.usbmodemA21",
		Commit:       "abcdef123456",
	})
	if err == nil {
		t.Fatal("expected upload candidate without release index to be rejected")
	}
	if !strings.Contains(err.Error(), "release index") {
		t.Fatalf("error = %q, want release index", err)
	}
}

func TestValidateUploadCandidateRequiresPerArtifactReleaseManifest(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeArtifactWithChecksum(t, artifact, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef123456"))
	checksum := readTestChecksum(t, artifact+".sha256")
	if err := appendReleaseIndexEntry(filepath.Join(dir, ReleaseIndexFileName), ReleaseIndexEntry{
		SchemaVersion: "a21.firmware.release.v1",
		FirmwareID:    "a21-stackchan",
		Version:       "0.1.0",
		Board:         "m5stack-cores3",
		Commit:        "abcdef123456",
		Timestamp:     "20260530-004500",
		ArtifactPath:  artifact,
		SHA256Path:    artifact + ".sha256",
		SHA256:        checksum,
		Build:         testBuildProvenance(artifact),
	}); err != nil {
		t.Fatal(err)
	}

	_, err := ValidateUploadCandidate(UploadCheckOptions{
		ManifestPath: manifest,
		ArtifactPath: artifact,
		Port:         "/dev/cu.usbmodemA21",
		Commit:       "abcdef123456",
	})
	if err == nil {
		t.Fatal("expected upload candidate without per-artifact release manifest to be rejected")
	}
	if !strings.Contains(err.Error(), "artifact release manifest") {
		t.Fatalf("error = %q, want artifact release manifest", err)
	}
}

func TestValidateUploadCandidateRejectsMissingBuildProvenance(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeArtifactWithChecksum(t, artifact, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef123456"))
	checksum := readTestChecksum(t, artifact+".sha256")
	if err := appendReleaseIndexEntry(filepath.Join(dir, ReleaseIndexFileName), ReleaseIndexEntry{
		SchemaVersion: "a21.firmware.release.v1",
		FirmwareID:    "a21-stackchan",
		Version:       "0.1.0",
		Board:         "m5stack-cores3",
		Commit:        "abcdef123456",
		Timestamp:     "20260530-004500",
		ArtifactPath:  artifact,
		SHA256Path:    artifact + ".sha256",
		SHA256:        checksum,
		Build:         testBuildProvenance(artifact),
	}); err != nil {
		t.Fatal(err)
	}
	writeArtifactReleaseManifestWithoutBuildProvenance(t, artifact, checksum)

	_, err := ValidateUploadCandidate(UploadCheckOptions{
		ManifestPath: manifest,
		ArtifactPath: artifact,
		Port:         "/dev/cu.usbmodemA21",
		Commit:       "abcdef123456",
	})
	if err == nil {
		t.Fatal("expected upload candidate without build provenance to be rejected")
	}
	if !strings.Contains(err.Error(), "build provenance") {
		t.Fatalf("error = %q, want build provenance", err)
	}
}

func TestValidateUploadCandidateAcceptsReleaseIndexEntry(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeArtifactWithChecksum(t, artifact, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef123456"))
	checksum := readTestChecksum(t, artifact+".sha256")
	indexPath := filepath.Join(dir, ReleaseIndexFileName)
	if err := appendReleaseIndexEntry(indexPath, ReleaseIndexEntry{
		SchemaVersion: "a21.firmware.release.v1",
		FirmwareID:    "a21-stackchan",
		Version:       "0.1.0",
		Board:         "m5stack-cores3",
		Commit:        "abcdef123456",
		Timestamp:     "20260530-004500",
		ArtifactPath:  artifact,
		SHA256Path:    artifact + ".sha256",
		SHA256:        checksum,
		Build:         testBuildProvenance(artifact),
	}); err != nil {
		t.Fatal(err)
	}
	writeArtifactReleaseManifest(t, artifact, checksum)

	result, err := ValidateUploadCandidate(UploadCheckOptions{
		ManifestPath: manifest,
		ArtifactPath: artifact,
		Port:         "/dev/cu.usbmodemA21",
		Commit:       "abcdef123456",
	})
	if err != nil {
		t.Fatalf("ValidateUploadCandidate returned error: %v", err)
	}
	if result.Artifact.ReleaseIndexPath != indexPath {
		t.Fatalf("ReleaseIndexPath = %q, want %q", result.Artifact.ReleaseIndexPath, indexPath)
	}
}

func TestValidateUploadCandidateRejectsReleaseIndexChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeArtifactWithChecksum(t, artifact, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef123456"))
	if err := appendReleaseIndexEntry(filepath.Join(dir, ReleaseIndexFileName), ReleaseIndexEntry{
		SchemaVersion: "a21.firmware.release.v1",
		FirmwareID:    "a21-stackchan",
		Version:       "0.1.0",
		Board:         "m5stack-cores3",
		Commit:        "abcdef123456",
		Timestamp:     "20260530-004500",
		ArtifactPath:  artifact,
		SHA256Path:    artifact + ".sha256",
		SHA256:        strings.Repeat("0", 64),
	}); err != nil {
		t.Fatal(err)
	}

	_, err := ValidateUploadCandidate(UploadCheckOptions{
		ManifestPath: manifest,
		ArtifactPath: artifact,
		Port:         "/dev/cu.usbmodemA21",
		Commit:       "abcdef123456",
	})
	if err == nil {
		t.Fatal("expected upload candidate with mismatched release index checksum to be rejected")
	}
	if !strings.Contains(err.Error(), "release index") {
		t.Fatalf("error = %q, want release index", err)
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

func readTestChecksum(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		t.Fatalf("checksum file %q is empty", path)
	}
	return fields[0]
}

func writeArtifactReleaseManifest(t *testing.T, artifactPath string, checksum string) {
	t.Helper()
	data, err := json.MarshalIndent(FirmwareReleaseManifest{
		SchemaVersion: ReleaseManifestSchemaVersion,
		Project:       "A21",
		FirmwareID:    "a21-stackchan",
		Version:       "0.1.0",
		Board:         "m5stack-cores3",
		Commit:        "abcdef123456",
		Timestamp:     "20260530-004500",
		ArtifactPath:  artifactPath,
		ArtifactName:  filepath.Base(artifactPath),
		SHA256Path:    artifactPath + ".sha256",
		SHA256:        checksum,
		Build:         testBuildProvenance(artifactPath),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath+".manifest.json", append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeArtifactReleaseManifestWithoutBuildProvenance(t *testing.T, artifactPath string, checksum string) {
	t.Helper()
	data, err := json.MarshalIndent(FirmwareReleaseManifest{
		SchemaVersion: ReleaseManifestSchemaVersion,
		Project:       "A21",
		FirmwareID:    "a21-stackchan",
		Version:       "0.1.0",
		Board:         "m5stack-cores3",
		Commit:        "abcdef123456",
		Timestamp:     "20260530-004500",
		ArtifactPath:  artifactPath,
		ArtifactName:  filepath.Base(artifactPath),
		SHA256Path:    artifactPath + ".sha256",
		SHA256:        checksum,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath+".manifest.json", append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testBuildProvenance(artifactPath string) FirmwareBuildProvenance {
	return FirmwareBuildProvenance{
		BuildSystem:     "platformio",
		PlatformIOEnv:   "a21_stackchan_cores3",
		PlatformIOBoard: "m5stack-cores3",
		SourcePath:      filepath.Join(filepath.Dir(filepath.Dir(artifactPath)), ".pio", "build", "a21_stackchan_cores3", "firmware.bin"),
		SourceName:      "firmware.bin",
	}
}
