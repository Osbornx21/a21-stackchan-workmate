package firmwarecheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageArtifactWritesReleaseIndexEntry(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	input := filepath.Join(dir, "firmware.bin")
	if err := os.WriteFile(input, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef123456"), 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(dir, "artifacts")

	result, err := PackageArtifact(PackageOptions{
		ManifestPath: manifest,
		InputPath:    input,
		OutputDir:    outputDir,
		Commit:       "abcdef123456",
		Timestamp:    "20260530-073000",
	})
	if err != nil {
		t.Fatalf("PackageArtifact returned error: %v", err)
	}

	wantIndex := filepath.Join(outputDir, ReleaseIndexFileName)
	if result.ReleaseIndexPath != wantIndex {
		t.Fatalf("ReleaseIndexPath = %q, want %q", result.ReleaseIndexPath, wantIndex)
	}
	data, err := os.ReadFile(result.ReleaseIndexPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("release index lines = %d, want 1: %s", len(lines), string(data))
	}
	var entry ReleaseIndexEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatal(err)
	}
	if entry.SchemaVersion != "a21.firmware.release.v1" {
		t.Fatalf("schema_version = %q", entry.SchemaVersion)
	}
	if entry.ArtifactPath != result.ArtifactPath || entry.SHA256Path != result.SHA256Path || entry.SHA256 != result.SHA256 {
		t.Fatalf("release index entry does not match package result: %#v vs %#v", entry, result)
	}
	if entry.FirmwareID != "a21-stackchan" || entry.Version != "0.1.0" || entry.Board != "m5stack-cores3" {
		t.Fatalf("release index firmware identity mismatch: %#v", entry)
	}
	if entry.Commit != "abcdef123456" || entry.Timestamp != "20260530-073000" {
		t.Fatalf("release index build identity mismatch: %#v", entry)
	}
}

func TestPackageArtifactWritesPerArtifactReleaseManifest(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	input := filepath.Join(dir, "firmware.bin")
	if err := os.WriteFile(input, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef123456"), 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(dir, "artifacts")

	result, err := PackageArtifact(PackageOptions{
		ManifestPath: manifest,
		InputPath:    input,
		OutputDir:    outputDir,
		Commit:       "abcdef123456",
		Timestamp:    "20260530-073000",
	})
	if err != nil {
		t.Fatalf("PackageArtifact returned error: %v", err)
	}

	wantManifest := result.ArtifactPath + ".manifest.json"
	if result.ReleaseManifestPath != wantManifest {
		t.Fatalf("ReleaseManifestPath = %q, want %q", result.ReleaseManifestPath, wantManifest)
	}
	data, err := os.ReadFile(result.ReleaseManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var releaseManifest FirmwareReleaseManifest
	if err := json.Unmarshal(data, &releaseManifest); err != nil {
		t.Fatal(err)
	}
	if releaseManifest.SchemaVersion != "a21.firmware.artifact_manifest.v1" {
		t.Fatalf("schema_version = %q", releaseManifest.SchemaVersion)
	}
	if releaseManifest.Project != "A21" || releaseManifest.FirmwareID != "a21-stackchan" || releaseManifest.Version != "0.1.0" || releaseManifest.Board != "m5stack-cores3" {
		t.Fatalf("release manifest identity mismatch: %#v", releaseManifest)
	}
	if releaseManifest.Commit != "abcdef123456" || releaseManifest.Timestamp != "20260530-073000" {
		t.Fatalf("release manifest build identity mismatch: %#v", releaseManifest)
	}
	if releaseManifest.ArtifactPath != result.ArtifactPath || releaseManifest.SHA256Path != result.SHA256Path || releaseManifest.SHA256 != result.SHA256 {
		t.Fatalf("release manifest does not match package result: %#v vs %#v", releaseManifest, result)
	}
	if releaseManifest.ArtifactName != filepath.Base(result.ArtifactPath) {
		t.Fatalf("artifact_name = %q, want %q", releaseManifest.ArtifactName, filepath.Base(result.ArtifactPath))
	}
}

func TestPackageArtifactRejectsLegacyOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	input := filepath.Join(dir, "firmware.bin")
	if err := os.WriteFile(input, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef123456"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := PackageArtifact(PackageOptions{
		ManifestPath: manifest,
		InputPath:    input,
		OutputDir:    filepath.Join(dir, "x21-artifacts"),
		Commit:       "abcdef123456",
		Timestamp:    "20260530-073000",
	})
	if err == nil {
		t.Fatal("expected legacy output directory to be rejected")
	}
	if !strings.Contains(err.Error(), "forbidden legacy identity") {
		t.Fatalf("error = %q, want forbidden legacy identity", err)
	}
}
