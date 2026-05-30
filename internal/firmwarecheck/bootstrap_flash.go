package firmwarecheck

import (
	"fmt"
	"path/filepath"
	"time"
)

type BootstrapFlashPlanOptions struct {
	ManifestPath      string
	ArtifactPath      string
	Port              string
	ExpectedGitCommit string
	PortUsage         PortUsage
	BuildDir          string
	CoreDir           string
	NowMS             int64
}

type BootstrapFlashPart struct {
	Offset string `json:"offset"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type BootstrapFlashPlanResult struct {
	GuardID                  string               `json:"guard_id"`
	GeneratedAtMS            int64                `json:"generated_at_ms"`
	DryRun                   bool                 `json:"dry_run"`
	FlashAllowed             bool                 `json:"flash_allowed"`
	NextRequiredConfirmation string               `json:"next_required_confirmation"`
	Port                     string               `json:"port"`
	ArtifactPath             string               `json:"artifact_path"`
	ArtifactSHA256           string               `json:"artifact_sha256"`
	Commit                   string               `json:"commit"`
	Upload                   UploadCheckResult    `json:"upload"`
	Parts                    []BootstrapFlashPart `json:"parts"`
	OK                       bool                 `json:"ok"`
	ReportPath               string               `json:"report_path,omitempty"`
}

func BuildBootstrapFlashPlan(options BootstrapFlashPlanOptions) (BootstrapFlashPlanResult, error) {
	generatedAtMS := options.NowMS
	if generatedAtMS <= 0 {
		generatedAtMS = time.Now().UnixMilli()
	}
	upload, err := ValidateUploadCandidate(UploadCheckOptions{
		ManifestPath: options.ManifestPath,
		ArtifactPath: options.ArtifactPath,
		Port:         options.Port,
		Commit:       options.ExpectedGitCommit,
	})
	if err != nil {
		return BootstrapFlashPlanResult{}, err
	}
	if !options.PortUsage.Exists {
		return BootstrapFlashPlanResult{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if options.PortUsage.InUse {
		return BootstrapFlashPlanResult{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, options.PortUsage.Detail)
	}
	upload.PortUsage = options.PortUsage

	buildDir := options.BuildDir
	if buildDir == "" {
		buildDir = filepath.Join("firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3")
	}
	coreDir := options.CoreDir
	if coreDir == "" {
		coreDir = filepath.Join(".a21-tools", "platformio-core")
	}
	parts, err := bootstrapFlashParts(buildDir, coreDir, upload.Artifact.ArtifactPath)
	if err != nil {
		return BootstrapFlashPlanResult{}, err
	}
	return BootstrapFlashPlanResult{
		GuardID:                  "a21.firmware.bootstrap_flash_plan.v1",
		GeneratedAtMS:            generatedAtMS,
		DryRun:                   true,
		FlashAllowed:             false,
		NextRequiredConfirmation: "firmware-bootstrap-flash-execute_with_confirmation_token",
		Port:                     options.Port,
		ArtifactPath:             upload.Artifact.ArtifactPath,
		ArtifactSHA256:           upload.Artifact.SHA256,
		Commit:                   upload.Artifact.Commit,
		Upload:                   upload,
		Parts:                    parts,
		OK:                       true,
	}, nil
}

func bootstrapFlashParts(buildDir string, coreDir string, artifactPath string) ([]BootstrapFlashPart, error) {
	candidates := []struct {
		offset string
		path   string
	}{
		{offset: "0x0000", path: filepath.Join(buildDir, "bootloader.bin")},
		{offset: "0x8000", path: filepath.Join(buildDir, "partitions.bin")},
		{offset: "0xe000", path: filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin")},
		{offset: "0x10000", path: artifactPath},
	}
	parts := make([]BootstrapFlashPart, 0, len(candidates))
	for _, candidate := range candidates {
		sha, err := fileSHA256(candidate.path)
		if err != nil {
			return nil, fmt.Errorf("bootstrap flash image %s missing or unreadable: %w", candidate.offset, err)
		}
		parts = append(parts, BootstrapFlashPart{
			Offset: candidate.offset,
			Path:   candidate.path,
			SHA256: sha,
		})
	}
	return parts, nil
}
