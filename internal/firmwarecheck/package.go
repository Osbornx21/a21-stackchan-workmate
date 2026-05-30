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
const StackChanPlatformIOEnv = "a21_stackchan_cores3"

type PackageOptions struct {
	ManifestPath    string
	InputPath       string
	OutputDir       string
	Commit          string
	Timestamp       string
	PlatformIOEnv   string
	PlatformIOBoard string
}

type PackageResult struct {
	ArtifactPath        string `json:"artifact_path"`
	SHA256Path          string `json:"sha256_path"`
	SHA256              string `json:"sha256"`
	ReleaseManifestPath string `json:"release_manifest_path"`
	ReleaseIndexPath    string `json:"release_index_path"`
}

type ReleaseIndexEntry struct {
	SchemaVersion string                  `json:"schema_version"`
	FirmwareID    string                  `json:"firmware_id"`
	Version       string                  `json:"version"`
	Board         string                  `json:"board"`
	Commit        string                  `json:"commit"`
	Timestamp     string                  `json:"timestamp"`
	ArtifactPath  string                  `json:"artifact_path"`
	SHA256Path    string                  `json:"sha256_path"`
	SHA256        string                  `json:"sha256"`
	Build         FirmwareBuildProvenance `json:"build"`
}

type FirmwareReleaseManifest struct {
	SchemaVersion string                  `json:"schema_version"`
	Project       string                  `json:"project"`
	FirmwareID    string                  `json:"firmware_id"`
	Version       string                  `json:"version"`
	Board         string                  `json:"board"`
	Commit        string                  `json:"commit"`
	Timestamp     string                  `json:"timestamp"`
	ArtifactPath  string                  `json:"artifact_path"`
	ArtifactName  string                  `json:"artifact_name"`
	SHA256Path    string                  `json:"sha256_path"`
	SHA256        string                  `json:"sha256"`
	Build         FirmwareBuildProvenance `json:"build"`
}

type FirmwareBuildProvenance struct {
	BuildSystem     string `json:"build_system"`
	PlatformIOEnv   string `json:"platformio_env"`
	PlatformIOBoard string `json:"platformio_board"`
	SourcePath      string `json:"source_path"`
	SourceName      string `json:"source_name"`
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
	build, err := packageBuildProvenance(options, manifest)
	if err != nil {
		return PackageResult{}, err
	}
	if err := validateEmbeddedArtifactIdentity(options.InputPath, manifest, options.Commit); err != nil {
		return PackageResult{}, fmt.Errorf("source firmware identity invalid: %w", err)
	}
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
		Build:         build,
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
		Build:         entry.Build,
	}
	data, err := json.MarshalIndent(releaseManifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func packageBuildProvenance(options PackageOptions, manifest Manifest) (FirmwareBuildProvenance, error) {
	if options.PlatformIOEnv == "" {
		return FirmwareBuildProvenance{}, fmt.Errorf("platformio env is required for firmware package provenance")
	}
	if options.PlatformIOEnv != StackChanPlatformIOEnv {
		return FirmwareBuildProvenance{}, fmt.Errorf("platformio env %q does not match A21 StackChan env %q", options.PlatformIOEnv, StackChanPlatformIOEnv)
	}
	if options.PlatformIOBoard == "" {
		return FirmwareBuildProvenance{}, fmt.Errorf("platformio board is required for firmware package provenance")
	}
	if options.PlatformIOBoard != manifest.Board {
		return FirmwareBuildProvenance{}, fmt.Errorf("platformio board %q does not match manifest board %q", options.PlatformIOBoard, manifest.Board)
	}
	sourceName := filepath.Base(options.InputPath)
	build := FirmwareBuildProvenance{
		BuildSystem:     "platformio",
		PlatformIOEnv:   options.PlatformIOEnv,
		PlatformIOBoard: options.PlatformIOBoard,
		SourcePath:      options.InputPath,
		SourceName:      sourceName,
	}
	if err := validateFirmwareBuildProvenance(build, manifest); err != nil {
		return FirmwareBuildProvenance{}, err
	}
	return build, nil
}

func validateFirmwareBuildProvenance(build FirmwareBuildProvenance, manifest Manifest) error {
	joined := strings.ToLower(strings.Join([]string{
		build.BuildSystem,
		build.PlatformIOEnv,
		build.PlatformIOBoard,
		build.SourcePath,
		build.SourceName,
	}, " "))
	if strings.Contains(joined, "x21") || strings.Contains(joined, "v21") {
		return fmt.Errorf("firmware build provenance contains forbidden legacy identity")
	}
	if build.BuildSystem != "platformio" {
		return fmt.Errorf("firmware build system must be platformio")
	}
	if build.PlatformIOEnv != StackChanPlatformIOEnv {
		return fmt.Errorf("firmware platformio env %q does not match %q", build.PlatformIOEnv, StackChanPlatformIOEnv)
	}
	if build.PlatformIOBoard != manifest.Board {
		return fmt.Errorf("firmware platformio board %q does not match manifest board %q", build.PlatformIOBoard, manifest.Board)
	}
	if build.SourceName != "firmware.bin" || filepath.Base(build.SourcePath) != build.SourceName {
		return fmt.Errorf("firmware build source must be PlatformIO firmware.bin")
	}
	normalizedSource := filepath.ToSlash(filepath.Clean(build.SourcePath))
	requiredSegment := "/firmware/stackchan/.pio/build/" + build.PlatformIOEnv + "/" + build.SourceName
	requiredRelative := strings.TrimPrefix(requiredSegment, "/")
	if !strings.HasSuffix(normalizedSource, requiredSegment) && normalizedSource != requiredRelative {
		return fmt.Errorf("firmware build source must come from firmware/stackchan/.pio/build/%s/firmware.bin", build.PlatformIOEnv)
	}
	return nil
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
