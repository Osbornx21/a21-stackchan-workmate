package firmwarecheck

import (
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

const ReleaseIndexFileName = "a21-firmware-release-index.jsonl"
const ReleaseManifestSchemaVersion = "a21.firmware.artifact_manifest.v1"

type PackageOptions struct {
	ManifestPath string
	InputPath    string
	OutputDir    string
	Commit       string
	Timestamp    string
}

type PackageResult struct {
	ArtifactPath        string `json:"artifact_path"`
	SHA256Path          string `json:"sha256_path"`
	SHA256              string `json:"sha256"`
	ReleaseManifestPath string `json:"release_manifest_path"`
	ReleaseIndexPath    string `json:"release_index_path"`
}

type ReleaseIndexEntry struct {
	SchemaVersion string `json:"schema_version"`
	FirmwareID    string `json:"firmware_id"`
	Version       string `json:"version"`
	Board         string `json:"board"`
	Commit        string `json:"commit"`
	Timestamp     string `json:"timestamp"`
	ArtifactPath  string `json:"artifact_path"`
	SHA256Path    string `json:"sha256_path"`
	SHA256        string `json:"sha256"`
}

type FirmwareReleaseManifest struct {
	SchemaVersion string `json:"schema_version"`
	Project       string `json:"project"`
	FirmwareID    string `json:"firmware_id"`
	Version       string `json:"version"`
	Board         string `json:"board"`
	Commit        string `json:"commit"`
	Timestamp     string `json:"timestamp"`
	ArtifactPath  string `json:"artifact_path"`
	ArtifactName  string `json:"artifact_name"`
	SHA256Path    string `json:"sha256_path"`
	SHA256        string `json:"sha256"`
}

func PackageArtifact(options PackageOptions) (PackageResult, error) {
	manifestResult, err := LoadAndValidate(options.ManifestPath)
	if err != nil {
		return PackageResult{}, err
	}
	if options.InputPath == "" {
		return PackageResult{}, fmt.Errorf("input firmware binary is required")
	}
	if options.OutputDir == "" {
		return PackageResult{}, fmt.Errorf("output directory is required")
	}
	if containsForbiddenPackagePathIdentity(options.InputPath) || containsForbiddenPackagePathIdentity(options.OutputDir) {
		return PackageResult{}, fmt.Errorf("firmware package path contains forbidden legacy identity")
	}
	if ok, _ := regexp.MatchString(`^[0-9a-fA-F]{7,40}$`, options.Commit); !ok {
		return PackageResult{}, fmt.Errorf("commit must be a 7-40 character git sha")
	}
	if ok, _ := regexp.MatchString(`^\d{8}-\d{6}$`, options.Timestamp); !ok {
		return PackageResult{}, fmt.Errorf("timestamp must be YYYYMMDD-HHMMSS")
	}
	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return PackageResult{}, err
	}
	manifest := manifestResult.Manifest
	artifactName := fmt.Sprintf("%s-%s-%s-%s-%s.bin", manifest.ArtifactPrefix, manifest.Version, manifest.Board, options.Commit, options.Timestamp)
	artifactPath := filepath.Join(options.OutputDir, artifactName)
	checksum, err := copyWithSHA256(options.InputPath, artifactPath)
	if err != nil {
		return PackageResult{}, err
	}
	shaPath := artifactPath + ".sha256"
	if err := os.WriteFile(shaPath, []byte(checksum+"  "+artifactName+"\n"), 0o644); err != nil {
		return PackageResult{}, err
	}
	entry := ReleaseIndexEntry{
		SchemaVersion: "a21.firmware.release.v1",
		FirmwareID:    manifest.FirmwareID,
		Version:       manifest.Version,
		Board:         manifest.Board,
		Commit:        options.Commit,
		Timestamp:     options.Timestamp,
		ArtifactPath:  artifactPath,
		SHA256Path:    shaPath,
		SHA256:        checksum,
	}
	releaseManifestPath := artifactPath + ".manifest.json"
	if err := writeFirmwareReleaseManifest(releaseManifestPath, manifest, entry); err != nil {
		return PackageResult{}, err
	}
	releaseIndexPath := filepath.Join(options.OutputDir, ReleaseIndexFileName)
	if err := appendReleaseIndexEntry(releaseIndexPath, entry); err != nil {
		return PackageResult{}, err
	}
	return PackageResult{
		ArtifactPath:        artifactPath,
		SHA256Path:          shaPath,
		SHA256:              checksum,
		ReleaseManifestPath: releaseManifestPath,
		ReleaseIndexPath:    releaseIndexPath,
	}, nil
}

func writeFirmwareReleaseManifest(path string, manifest Manifest, entry ReleaseIndexEntry) error {
	releaseManifest := FirmwareReleaseManifest{
		SchemaVersion: ReleaseManifestSchemaVersion,
		Project:       manifest.Project,
		FirmwareID:    entry.FirmwareID,
		Version:       entry.Version,
		Board:         entry.Board,
		Commit:        entry.Commit,
		Timestamp:     entry.Timestamp,
		ArtifactPath:  entry.ArtifactPath,
		ArtifactName:  filepath.Base(entry.ArtifactPath),
		SHA256Path:    entry.SHA256Path,
		SHA256:        entry.SHA256,
	}
	data, err := json.MarshalIndent(releaseManifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func appendReleaseIndexEntry(path string, entry ReleaseIndexEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func containsForbiddenPackagePathIdentity(path string) bool {
	lowerPath := strings.ToLower(path)
	return strings.Contains(lowerPath, "x21") || strings.Contains(lowerPath, "v21")
}

func copyWithSHA256(inputPath string, outputPath string) (string, error) {
	input, err := os.Open(inputPath)
	if err != nil {
		return "", err
	}
	defer input.Close()

	output, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer output.Close()

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(output, hasher), input); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
