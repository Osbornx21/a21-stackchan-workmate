package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := stdout.String(); got != "A21 0.1.0-dev (a21-core)\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"nope"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunDoctorEmitsReport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	if !strings.Contains(stdout.String(), `"result"`) {
		t.Fatalf("stdout missing result report: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"fingerprint"`) {
		t.Fatalf("stdout missing fingerprint report: %q", stdout.String())
	}
}

func TestRunDoctorWritesReportFile(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-doctor-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("report files = %d, want 1", len(matches))
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"fingerprint"`) {
		t.Fatalf("report file missing fingerprint: %s", data)
	}
}

func TestRunDoctorRejectsUnknownFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--wat"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunGatewayHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"gateway", "--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "a21 gateway") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunGatewayRejectsUnknownFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"gateway", "--wat"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunFirmwareCheckRejectsMissingManifest(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", filepath.Join(t.TempDir(), "missing.json")}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunFirmwareCheckAcceptsA21Manifest(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "a21-firmware.json")
	platformio := filepath.Join(dir, "platformio.ini")
	data := []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`)
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(platformio, []byte("[env:a21_stackchan_cores3]\nboard = m5stack-cores3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware manifest ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareCheckRejectsWrongPlatformIOBoard(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "a21-firmware.json")
	data := []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`)
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte("[env:a21_stackchan_cores3]\nboard = m5stack-core2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "platformio board") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwarePackageCreatesVersionedArtifact(t *testing.T) {
	dir := t.TempDir()
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
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte("[env:a21_stackchan_cores3]\nboard = m5stack-cores3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "firmware.bin")
	if err := os.WriteFile(input, []byte("firmware"), 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(dir, "artifacts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-package",
		"--manifest", manifest,
		"--input", input,
		"--output-dir", outputDir,
		"--commit", "abcdef1",
		"--timestamp", "20260530-004500",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	artifact := filepath.Join(outputDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	if _, err := os.Stat(artifact); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(artifact + ".sha256"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), filepath.Base(artifact)) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareArtifactCheckAcceptsMatchingArtifactAndChecksum(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware artifact ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareArtifactCheckRejectsWrongBoardInArtifactName(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-core2-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "board") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareArtifactCheckRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	if err := os.WriteFile(artifact, []byte("firmware"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact+".sha256", []byte(strings.Repeat("0", 64)+"  "+filepath.Base(artifact)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "checksum") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRequiresExplicitPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--port requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRequiresExpectedCommit(t *testing.T) {
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
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--commit requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckAcceptsExplicitDevicePort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware upload guard ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareUploadCheckRejectsUnexpectedCommit(t *testing.T) {
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
		"--commit", "1234567",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "commit") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

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

func writeTestFirmwareManifest(t *testing.T, dir string) string {
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
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte("[env:a21_stackchan_cores3]\nboard = m5stack-cores3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func writeFirmwareArtifactWithChecksum(t *testing.T, artifactPath string, content []byte) {
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
