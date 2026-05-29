package firmwarecheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ArtifactOptions struct {
	ManifestPath string
	ArtifactPath string
}

type ArtifactResult struct {
	Manifest     Manifest `json:"manifest"`
	ArtifactPath string   `json:"artifact_path"`
	SHA256Path   string   `json:"sha256_path"`
	SHA256       string   `json:"sha256"`
	Commit       string   `json:"commit"`
	Timestamp    string   `json:"timestamp"`
	OK           bool     `json:"ok"`
}

type UploadCheckOptions struct {
	ManifestPath string
	ArtifactPath string
	Port         string
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

	return ArtifactResult{
		Manifest:     manifest,
		ArtifactPath: options.ArtifactPath,
		SHA256Path:   shaPath,
		SHA256:       actualChecksum,
		Commit:       parsed.commit,
		Timestamp:    parsed.timestamp,
		OK:           true,
	}, nil
}

func ValidateUploadCandidate(options UploadCheckOptions) (UploadCheckResult, error) {
	if err := validateUploadPort(options.Port); err != nil {
		return UploadCheckResult{}, err
	}
	if err := validateExpectedCommit(options.Commit); err != nil {
		return UploadCheckResult{}, err
	}
	artifact, err := ValidateArtifact(ArtifactOptions{
		ManifestPath: options.ManifestPath,
		ArtifactPath: options.ArtifactPath,
	})
	if err != nil {
		return UploadCheckResult{}, err
	}
	if !sameGitCommit(options.Commit, artifact.Commit) {
		return UploadCheckResult{}, fmt.Errorf("artifact commit %q does not match expected commit %q", artifact.Commit, options.Commit)
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
