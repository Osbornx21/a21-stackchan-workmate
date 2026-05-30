package firmwarecheck

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ArtifactOptions struct {
	ManifestPath           string
	ArtifactPath           string
	RequireReleaseIndex    bool
	RequireReleaseManifest bool
}

type ArtifactResult struct {
	Manifest            Manifest                 `json:"manifest"`
	ArtifactPath        string                   `json:"artifact_path"`
	SHA256Path          string                   `json:"sha256_path"`
	SHA256              string                   `json:"sha256"`
	Commit              string                   `json:"commit"`
	Timestamp           string                   `json:"timestamp"`
	Build               *FirmwareBuildProvenance `json:"build,omitempty"`
	ReleaseManifestPath string                   `json:"release_manifest_path,omitempty"`
	ReleaseIndexPath    string                   `json:"release_index_path,omitempty"`
	OK                  bool                     `json:"ok"`
}

type UploadCheckOptions struct {
	ManifestPath string
	ArtifactPath string
	Port         string
	Commit       string
}

type LatestArtifactOptions struct {
	ManifestPath string
	ArtifactDir  string
	Commit       string
}

type UploadCheckResult struct {
	GuardID                  string         `json:"guard_id"`
	DryRun                   bool           `json:"dry_run"`
	FlashAllowed             bool           `json:"flash_allowed"`
	NextRequiredConfirmation string         `json:"next_required_confirmation"`
	Artifact                 ArtifactResult `json:"artifact"`
	Port                     string         `json:"port"`
	PortUsage                PortUsage      `json:"port_usage,omitempty"`
	OK                       bool           `json:"ok"`
}

func ValidateArtifact(options ArtifactOptions) (ArtifactResult, error) {
	manifestResult, err := LoadAndValidate(options.ManifestPath)
	if err != nil {
		return ArtifactResult{}, err
	}
	if options.ArtifactPath == "" {
		return ArtifactResult{}, fmt.Errorf("artifact path is required")
	}
	info, err := os.Stat(options.ArtifactPath)
	if err != nil {
		return ArtifactResult{}, err
	}
	if info.IsDir() {
		return ArtifactResult{}, fmt.Errorf("artifact path must be a firmware binary")
	}

	manifest := manifestResult.Manifest
	name := filepath.Base(options.ArtifactPath)
	parsed, err := parseArtifactName(name, manifest)
	if err != nil {
		return ArtifactResult{}, err
	}

	shaPath := options.ArtifactPath + ".sha256"
	expectedChecksum, err := readArtifactChecksum(shaPath, name)
	if err != nil {
		return ArtifactResult{}, err
	}
	actualChecksum, err := fileSHA256(options.ArtifactPath)
	if err != nil {
		return ArtifactResult{}, err
	}
	if !strings.EqualFold(expectedChecksum, actualChecksum) {
		return ArtifactResult{}, fmt.Errorf("artifact checksum mismatch")
	}
	if err := validateEmbeddedArtifactIdentity(options.ArtifactPath, manifest, parsed.commit); err != nil {
		return ArtifactResult{}, err
	}

	result := ArtifactResult{
		Manifest:     manifest,
		ArtifactPath: options.ArtifactPath,
		SHA256Path:   shaPath,
		SHA256:       actualChecksum,
		Commit:       parsed.commit,
		Timestamp:    parsed.timestamp,
		OK:           true,
	}
	if options.RequireReleaseIndex {
		releaseIndexPath := filepath.Join(filepath.Dir(options.ArtifactPath), ReleaseIndexFileName)
		build, err := validateReleaseIndexEntry(releaseIndexPath, result)
		if err != nil {
			return ArtifactResult{}, err
		}
		result.Build = &build
		result.ReleaseIndexPath = releaseIndexPath
	}
	if options.RequireReleaseManifest {
		releaseManifestPath := options.ArtifactPath + ".manifest.json"
		build, err := validateArtifactReleaseManifest(releaseManifestPath, result)
		if err != nil {
			return ArtifactResult{}, err
		}
		if result.Build != nil && *result.Build != build {
			return ArtifactResult{}, fmt.Errorf("release index and artifact release manifest build provenance mismatch")
		}
		result.Build = &build
		result.ReleaseManifestPath = releaseManifestPath
	}
	return result, nil
}

func validateEmbeddedArtifactIdentity(path string, manifest Manifest, commit string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	required := []struct {
		label string
		value string
	}{
		{label: "firmware_id", value: manifest.FirmwareID},
		{label: "version", value: manifest.Version},
		{label: "board", value: manifest.Board},
		{label: "commit", value: commit},
	}
	for _, item := range required {
		if item.value == "" || !bytes.Contains(data, []byte(item.value)) {
			return fmt.Errorf("artifact missing embedded A21 %s identity", item.label)
		}
	}
	return nil
}

func ValidateUploadCandidate(options UploadCheckOptions) (UploadCheckResult, error) {
	if err := validateUploadPort(options.Port); err != nil {
		return UploadCheckResult{}, err
	}
	if err := validateExpectedCommit(options.Commit); err != nil {
		return UploadCheckResult{}, err
	}
	artifact, err := ValidateArtifact(ArtifactOptions{
		ManifestPath:           options.ManifestPath,
		ArtifactPath:           options.ArtifactPath,
		RequireReleaseIndex:    true,
		RequireReleaseManifest: true,
	})
	if err != nil {
		return UploadCheckResult{}, err
	}
	if !sameGitCommit(options.Commit, artifact.Commit) {
		return UploadCheckResult{}, fmt.Errorf("artifact commit %q does not match expected commit %q", artifact.Commit, options.Commit)
	}
	if err := validateLatestReleaseIndexArtifact(artifact.ReleaseIndexPath, artifact); err != nil {
		return UploadCheckResult{}, err
	}
	return UploadCheckResult{
		GuardID:                  "a21.firmware.upload_guard.v1",
		DryRun:                   true,
		FlashAllowed:             false,
		NextRequiredConfirmation: "physical_device_identity",
		Artifact:                 artifact,
		Port:                     options.Port,
		OK:                       true,
	}, nil
}

func ValidateLatestArtifactForCommit(options LatestArtifactOptions) (ArtifactResult, error) {
	manifestResult, err := LoadAndValidate(options.ManifestPath)
	if err != nil {
		return ArtifactResult{}, err
	}
	if options.ArtifactDir == "" {
		return ArtifactResult{}, fmt.Errorf("artifact directory is required")
	}
	if err := validateExpectedCommit(options.Commit); err != nil {
		return ArtifactResult{}, err
	}
	releaseIndexPath := filepath.Join(options.ArtifactDir, ReleaseIndexFileName)
	data, err := os.ReadFile(releaseIndexPath)
	if err != nil {
		return ArtifactResult{}, fmt.Errorf("release index missing or unreadable: %w", err)
	}
	manifest := manifestResult.Manifest
	var latest ReleaseIndexEntry
	found := false
	for lineNumber, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		var entry ReleaseIndexEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return ArtifactResult{}, fmt.Errorf("release index line %d is invalid: %w", lineNumber+1, err)
		}
		if releaseRecordPathContainsForbiddenIdentity(entry.ArtifactPath, entry.SHA256Path) {
			return ArtifactResult{}, fmt.Errorf("release index contains forbidden legacy path identity")
		}
		if entry.SchemaVersion != "a21.firmware.release.v1" ||
			entry.FirmwareID != manifest.FirmwareID ||
			entry.Version != manifest.Version ||
			entry.Board != manifest.Board ||
			!sameGitCommit(entry.Commit, options.Commit) {
			continue
		}
		if found && entry.Timestamp == latest.Timestamp && filepath.Base(entry.ArtifactPath) != filepath.Base(latest.ArtifactPath) {
			return ArtifactResult{}, fmt.Errorf("release index has ambiguous latest artifact timestamp %q for commit %q", entry.Timestamp, options.Commit)
		}
		if !found || entry.Timestamp > latest.Timestamp {
			latest = entry
			found = true
		}
	}
	if !found {
		return ArtifactResult{}, fmt.Errorf("no release index entry for commit %q", options.Commit)
	}
	result, err := ValidateArtifact(ArtifactOptions{
		ManifestPath:           options.ManifestPath,
		ArtifactPath:           latest.ArtifactPath,
		RequireReleaseIndex:    true,
		RequireReleaseManifest: true,
	})
	if err != nil {
		return ArtifactResult{}, err
	}
	if !sameGitCommit(options.Commit, result.Commit) {
		return ArtifactResult{}, fmt.Errorf("artifact commit %q does not match expected commit %q", result.Commit, options.Commit)
	}
	if err := validateLatestReleaseIndexArtifact(result.ReleaseIndexPath, result); err != nil {
		return ArtifactResult{}, err
	}
	return result, nil
}

func validateReleaseIndexEntry(path string, artifact ArtifactResult) (FirmwareBuildProvenance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FirmwareBuildProvenance{}, fmt.Errorf("release index missing or unreadable: %w", err)
	}
	expectedArtifactName := filepath.Base(artifact.ArtifactPath)
	expectedSHAName := filepath.Base(artifact.SHA256Path)
	for lineNumber, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		var entry ReleaseIndexEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return FirmwareBuildProvenance{}, fmt.Errorf("release index line %d is invalid: %w", lineNumber+1, err)
		}
		if releaseRecordPathContainsForbiddenIdentity(entry.ArtifactPath, entry.SHA256Path) {
			return FirmwareBuildProvenance{}, fmt.Errorf("release index contains forbidden legacy path identity")
		}
		if filepath.Base(entry.ArtifactPath) != expectedArtifactName {
			continue
		}
		if entry.SchemaVersion != "a21.firmware.release.v1" ||
			entry.FirmwareID != artifact.Manifest.FirmwareID ||
			entry.Version != artifact.Manifest.Version ||
			entry.Board != artifact.Manifest.Board ||
			!sameGitCommit(entry.Commit, artifact.Commit) ||
			entry.Timestamp != artifact.Timestamp ||
			filepath.Base(entry.SHA256Path) != expectedSHAName ||
			!strings.EqualFold(entry.SHA256, artifact.SHA256) {
			return FirmwareBuildProvenance{}, fmt.Errorf("release index entry for %q does not match artifact identity or checksum", expectedArtifactName)
		}
		if err := validateFirmwareBuildProvenance(entry.Build, artifact.Manifest); err != nil {
			return FirmwareBuildProvenance{}, fmt.Errorf("release index build provenance invalid: %w", err)
		}
		return entry.Build, nil
	}
	return FirmwareBuildProvenance{}, fmt.Errorf("release index has no entry for artifact %q", expectedArtifactName)
}

func validateLatestReleaseIndexArtifact(path string, artifact ArtifactResult) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("release index missing or unreadable: %w", err)
	}
	expectedName := filepath.Base(artifact.ArtifactPath)
	latestName := expectedName
	latestTimestamp := artifact.Timestamp
	for lineNumber, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		var entry ReleaseIndexEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return fmt.Errorf("release index line %d is invalid: %w", lineNumber+1, err)
		}
		if releaseRecordPathContainsForbiddenIdentity(entry.ArtifactPath, entry.SHA256Path) {
			return fmt.Errorf("release index contains forbidden legacy path identity")
		}
		if entry.SchemaVersion != "a21.firmware.release.v1" ||
			entry.FirmwareID != artifact.Manifest.FirmwareID ||
			entry.Version != artifact.Manifest.Version ||
			entry.Board != artifact.Manifest.Board ||
			!sameGitCommit(entry.Commit, artifact.Commit) {
			continue
		}
		entryName := filepath.Base(entry.ArtifactPath)
		if entry.Timestamp > latestTimestamp {
			latestTimestamp = entry.Timestamp
			latestName = entryName
		}
		if entry.Timestamp == artifact.Timestamp && entryName != expectedName {
			return fmt.Errorf("release index has ambiguous latest artifact timestamp %q for commit %q", entry.Timestamp, artifact.Commit)
		}
	}
	if latestName != expectedName || latestTimestamp != artifact.Timestamp {
		return fmt.Errorf("artifact %q is not the latest release index entry for commit %q; latest is %q at %s", expectedName, artifact.Commit, latestName, latestTimestamp)
	}
	return nil
}

func validateArtifactReleaseManifest(path string, artifact ArtifactResult) (FirmwareBuildProvenance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FirmwareBuildProvenance{}, fmt.Errorf("artifact release manifest missing or unreadable: %w", err)
	}
	var releaseManifest FirmwareReleaseManifest
	if err := json.Unmarshal(data, &releaseManifest); err != nil {
		return FirmwareBuildProvenance{}, fmt.Errorf("artifact release manifest is invalid: %w", err)
	}
	expectedArtifactName := filepath.Base(artifact.ArtifactPath)
	expectedSHAName := filepath.Base(artifact.SHA256Path)
	if releaseRecordPathContainsForbiddenIdentity(releaseManifest.ArtifactPath, releaseManifest.SHA256Path) {
		return FirmwareBuildProvenance{}, fmt.Errorf("artifact release manifest contains forbidden legacy path identity")
	}
	if releaseManifest.SchemaVersion != ReleaseManifestSchemaVersion ||
		releaseManifest.Project != artifact.Manifest.Project ||
		releaseManifest.FirmwareID != artifact.Manifest.FirmwareID ||
		releaseManifest.Version != artifact.Manifest.Version ||
		releaseManifest.Board != artifact.Manifest.Board ||
		!sameGitCommit(releaseManifest.Commit, artifact.Commit) ||
		releaseManifest.Timestamp != artifact.Timestamp ||
		releaseManifest.ArtifactName != expectedArtifactName ||
		filepath.Base(releaseManifest.ArtifactPath) != expectedArtifactName ||
		filepath.Base(releaseManifest.SHA256Path) != expectedSHAName ||
		!strings.EqualFold(releaseManifest.SHA256, artifact.SHA256) {
		return FirmwareBuildProvenance{}, fmt.Errorf("artifact release manifest for %q does not match artifact identity or checksum", expectedArtifactName)
	}
	joined := strings.ToLower(strings.Join([]string{
		releaseManifest.Project,
		releaseManifest.FirmwareID,
		releaseManifest.Version,
		releaseManifest.Board,
		releaseManifest.Commit,
		releaseManifest.ArtifactName,
	}, " "))
	if strings.Contains(joined, "x21") || strings.Contains(joined, "v21") {
		return FirmwareBuildProvenance{}, fmt.Errorf("artifact release manifest contains forbidden legacy identity")
	}
	if err := validateFirmwareBuildProvenance(releaseManifest.Build, artifact.Manifest); err != nil {
		return FirmwareBuildProvenance{}, fmt.Errorf("artifact release manifest build provenance invalid: %w", err)
	}
	return releaseManifest.Build, nil
}

func releaseRecordPathContainsForbiddenIdentity(paths ...string) bool {
	for _, path := range paths {
		if containsForbiddenPackagePathIdentity(path) {
			return true
		}
	}
	return false
}

type parsedArtifactName struct {
	commit    string
	timestamp string
}

func parseArtifactName(name string, manifest Manifest) (parsedArtifactName, error) {
	lowerName := strings.ToLower(name)
	if strings.Contains(lowerName, "x21") || strings.Contains(lowerName, "v21") {
		return parsedArtifactName{}, fmt.Errorf("artifact name contains forbidden legacy identity")
	}
	if !strings.HasSuffix(name, ".bin") {
		return parsedArtifactName{}, fmt.Errorf("artifact must be a .bin file")
	}
	expectedPrefix := manifest.ArtifactPrefix + "-" + manifest.Version + "-"
	if !strings.HasPrefix(name, expectedPrefix) {
		return parsedArtifactName{}, fmt.Errorf("artifact name must match manifest prefix and version")
	}
	tail := strings.TrimSuffix(strings.TrimPrefix(name, expectedPrefix), ".bin")
	match := regexp.MustCompile(`^(.+)-([0-9a-fA-F]{7,40})-(\d{8}-\d{6})$`).FindStringSubmatch(tail)
	if len(match) != 4 {
		return parsedArtifactName{}, fmt.Errorf("artifact name must include board, git commit, and timestamp")
	}
	board := match[1]
	if board != manifest.Board {
		return parsedArtifactName{}, fmt.Errorf("artifact board %q does not match manifest board %q", board, manifest.Board)
	}
	return parsedArtifactName{commit: match[2], timestamp: match[3]}, nil
}

func readArtifactChecksum(path string, artifactName string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return "", fmt.Errorf("artifact checksum file is empty")
	}
	checksum := fields[0]
	if ok, _ := regexp.MatchString(`^[0-9a-fA-F]{64}$`, checksum); !ok {
		return "", fmt.Errorf("artifact checksum must be sha256 hex")
	}
	if len(fields) >= 2 && filepath.Base(fields[1]) != artifactName {
		return "", fmt.Errorf("artifact checksum filename does not match artifact")
	}
	return checksum, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func validateUploadPort(port string) error {
	if port == "" {
		return fmt.Errorf("upload port is required")
	}
	normalized := strings.ToLower(strings.TrimSpace(port))
	switch normalized {
	case "auto", "default", "any", "latest":
		return fmt.Errorf("upload port must be an explicit serial device path")
	}
	if !filepath.IsAbs(port) || !strings.HasPrefix(port, "/dev/") {
		return fmt.Errorf("upload port must be an explicit serial device path")
	}
	base := strings.ToLower(filepath.Base(port))
	if !(strings.HasPrefix(base, "cu.") || strings.HasPrefix(base, "tty.") || strings.HasPrefix(base, "ttyusb") || strings.HasPrefix(base, "ttyacm")) {
		return fmt.Errorf("upload port must be an explicit serial device path")
	}
	if strings.Contains(normalized, "x21") || strings.Contains(normalized, "v21") {
		return fmt.Errorf("upload port contains forbidden legacy identity")
	}
	return nil
}

func validateExpectedCommit(commit string) error {
	if commit == "" {
		return fmt.Errorf("expected commit is required")
	}
	if ok, _ := regexp.MatchString(`^[0-9a-fA-F]{7,40}$`, commit); !ok {
		return fmt.Errorf("expected commit must be a 7-40 character git sha")
	}
	return nil
}

func sameGitCommit(expected string, actual string) bool {
	expected = strings.ToLower(expected)
	actual = strings.ToLower(actual)
	return expected == actual || strings.HasPrefix(expected, actual) || strings.HasPrefix(actual, expected)
}
