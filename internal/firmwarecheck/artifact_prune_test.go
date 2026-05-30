package firmwarecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildArtifactPrunePlanKeepsCurrentAndRecentWithoutDeleting(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	oldArtifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-1111111-20260530-010000.bin")
	recentArtifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-2222222-20260530-020000.bin")
	currentArtifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-3333333-20260530-030000.bin")
	writeReleaseLedgerArtifact(t, oldArtifact, []byte("a21-stackchan 0.1.0 m5stack-cores3 1111111 old"))
	writeReleaseLedgerArtifact(t, recentArtifact, []byte("a21-stackchan 0.1.0 m5stack-cores3 2222222 recent"))
	writeReleaseLedgerArtifact(t, currentArtifact, []byte("a21-stackchan 0.1.0 m5stack-cores3 3333333 current"))

	plan, err := BuildArtifactPrunePlan(ArtifactPrunePlanOptions{
		ManifestPath: manifest,
		ArtifactDir:  dir,
		Commit:       "3333333",
		KeepRecent:   2,
	})
	if err != nil {
		t.Fatalf("BuildArtifactPrunePlan returned error: %v", err)
	}
	if !plan.DryRun || plan.DeleteAllowed {
		t.Fatalf("dry/delete flags = %v/%v, want true/false", plan.DryRun, plan.DeleteAllowed)
	}
	if plan.CurrentArtifactPath != currentArtifact {
		t.Fatalf("CurrentArtifactPath = %q, want %q", plan.CurrentArtifactPath, currentArtifact)
	}
	if !artifactRetentionContains(plan.Keep, currentArtifact, "current") {
		t.Fatalf("keep missing current artifact: %#v", plan.Keep)
	}
	if !artifactRetentionContains(plan.Keep, recentArtifact, "recent") {
		t.Fatalf("keep missing recent artifact: %#v", plan.Keep)
	}
	if !artifactRetentionContains(plan.PruneCandidates, oldArtifact, "older_than_keep_recent") {
		t.Fatalf("prune candidates missing old artifact: %#v", plan.PruneCandidates)
	}
	for _, path := range []string{oldArtifact, recentArtifact, currentArtifact} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("artifact %q was removed by dry-run plan: %v", path, err)
		}
	}
}

func TestBuildArtifactPrunePlanRequiresReleaseLedgerCurrentArtifact(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)
	looseArtifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-3333333-20260530-030000.bin")
	writeArtifactWithChecksum(t, looseArtifact, []byte("a21-stackchan 0.1.0 m5stack-cores3 3333333 loose"))

	_, err := BuildArtifactPrunePlan(ArtifactPrunePlanOptions{
		ManifestPath: manifest,
		ArtifactDir:  dir,
		Commit:       "3333333",
		KeepRecent:   2,
	})
	if err == nil {
		t.Fatal("expected prune plan without release-ledger current artifact to be rejected")
	}
	if !strings.Contains(err.Error(), "release") {
		t.Fatalf("error = %q, want release-ledger failure", err)
	}
}

func TestBuildArtifactPrunePlanRejectsLegacyArtifactDir(t *testing.T) {
	dir := t.TempDir()
	manifest := writeArtifactManifest(t, dir)

	_, err := BuildArtifactPrunePlan(ArtifactPrunePlanOptions{
		ManifestPath: manifest,
		ArtifactDir:  filepath.Join(dir, "x21-artifacts"),
		Commit:       "3333333",
		KeepRecent:   2,
	})
	if err == nil {
		t.Fatal("expected legacy artifact directory to be rejected")
	}
	if !strings.Contains(err.Error(), "forbidden legacy") {
		t.Fatalf("error = %q, want forbidden legacy", err)
	}
}

func artifactRetentionContains(records []ArtifactRetentionRecord, artifactPath string, reason string) bool {
	for _, record := range records {
		if record.ArtifactPath == artifactPath && record.Reason == reason {
			return true
		}
	}
	return false
}

func writeReleaseLedgerArtifact(t *testing.T, artifactPath string, content []byte) {
	t.Helper()
	writeArtifactWithChecksum(t, artifactPath, content)
	checksum := readTestChecksum(t, artifactPath+".sha256")
	commit := testArtifactCommitFromPath(t, artifactPath)
	timestamp := testArtifactTimestampFromPath(t, artifactPath)
	if err := appendReleaseIndexEntry(filepath.Join(filepath.Dir(artifactPath), ReleaseIndexFileName), ReleaseIndexEntry{
		SchemaVersion: "a21.firmware.release.v1",
		FirmwareID:    "a21-stackchan",
		Version:       "0.1.0",
		Board:         "m5stack-cores3",
		Commit:        commit,
		Timestamp:     timestamp,
		ArtifactPath:  artifactPath,
		SHA256Path:    artifactPath + ".sha256",
		SHA256:        checksum,
		Build:         testBuildProvenance(artifactPath),
	}); err != nil {
		t.Fatal(err)
	}
	writeArtifactReleaseManifestWithIdentity(t, artifactPath, checksum, commit, timestamp)
}

func testArtifactCommitFromPath(t *testing.T, artifactPath string) string {
	t.Helper()
	parts := strings.Split(strings.TrimSuffix(filepath.Base(artifactPath), ".bin"), "-")
	if len(parts) < 4 {
		t.Fatalf("artifact path %q lacks commit", artifactPath)
	}
	return parts[len(parts)-3]
}

func testArtifactTimestampFromPath(t *testing.T, artifactPath string) string {
	t.Helper()
	parts := strings.Split(strings.TrimSuffix(filepath.Base(artifactPath), ".bin"), "-")
	if len(parts) < 5 {
		t.Fatalf("artifact path %q lacks timestamp", artifactPath)
	}
	return parts[len(parts)-2] + "-" + parts[len(parts)-1]
}
