package firmwarecheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type PackageOptions struct {
	ManifestPath string
	InputPath    string
	OutputDir    string
	Commit       string
	Timestamp    string
}

type PackageResult struct {
	ArtifactPath string `json:"artifact_path"`
	SHA256Path   string `json:"sha256_path"`
	SHA256       string `json:"sha256"`
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
	return PackageResult{ArtifactPath: artifactPath, SHA256Path: shaPath, SHA256: checksum}, nil
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
