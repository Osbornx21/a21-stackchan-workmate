package firmwarecheck

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const StackChanMicProbePlatformIOEnv = "a21_stackchan_cores3_mic_probe"
const MicProbeCapabilityStatus = "diagnostic_probe_m5unified_i2s_capture"

type MicProbeFlashPlanOptions struct {
	ManifestPath      string
	ArtifactPath      string
	Port              string
	ExpectedGitCommit string
	PortUsage         PortUsage
	BuildDir          string
	CoreDir           string
	NowMS             int64
}

type MicProbeFlashPlanResult struct {
	GuardID                  string               `json:"guard_id"`
	GeneratedAtMS            int64                `json:"generated_at_ms"`
	DryRun                   bool                 `json:"dry_run"`
	FlashAllowed             bool                 `json:"flash_allowed"`
	NextRequiredConfirmation string               `json:"next_required_confirmation"`
	Port                     string               `json:"port"`
	ArtifactPath             string               `json:"artifact_path"`
	ArtifactSHA256           string               `json:"artifact_sha256"`
	Commit                   string               `json:"commit"`
	PlatformIOEnv            string               `json:"platformio_env"`
	BuildDir                 string               `json:"build_dir"`
	Parts                    []BootstrapFlashPart `json:"parts"`
	OK                       bool                 `json:"ok"`
	ReportPath               string               `json:"report_path,omitempty"`
}

func BuildMicProbeFlashPlan(options MicProbeFlashPlanOptions) (MicProbeFlashPlanResult, error) {
	generatedAtMS := options.NowMS
	if generatedAtMS <= 0 {
		generatedAtMS = time.Now().UnixMilli()
	}
	if err := validateUploadPort(options.Port); err != nil {
		return MicProbeFlashPlanResult{}, err
	}
	if err := validateExpectedCommit(options.ExpectedGitCommit); err != nil {
		return MicProbeFlashPlanResult{}, err
	}
	if !options.PortUsage.Exists {
		return MicProbeFlashPlanResult{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if options.PortUsage.InUse {
		return MicProbeFlashPlanResult{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, options.PortUsage.Detail)
	}

	manifestResult, err := LoadAndValidate(options.ManifestPath)
	if err != nil {
		return MicProbeFlashPlanResult{}, err
	}
	buildDir := options.BuildDir
	if buildDir == "" {
		buildDir = filepath.Join("firmware", "stackchan", ".pio", "build", StackChanMicProbePlatformIOEnv)
	}
	coreDir := options.CoreDir
	if coreDir == "" {
		coreDir = filepath.Join(".a21-tools", "platformio-core")
	}
	artifactPath := options.ArtifactPath
	if artifactPath == "" {
		artifactPath = filepath.Join(buildDir, "firmware.bin")
	}
	if err := validateMicProbeBuildPath(buildDir, artifactPath); err != nil {
		return MicProbeFlashPlanResult{}, err
	}
	if containsForbiddenPackagePathIdentity(artifactPath) || containsForbiddenPackagePathIdentity(buildDir) || containsForbiddenPackagePathIdentity(coreDir) {
		return MicProbeFlashPlanResult{}, fmt.Errorf("mic probe flash path contains forbidden legacy identity")
	}
	if err := validateEmbeddedArtifactIdentity(artifactPath, manifestResult.Manifest, options.ExpectedGitCommit); err != nil {
		return MicProbeFlashPlanResult{}, fmt.Errorf("mic probe firmware identity invalid: %w", err)
	}
	if err := validateMicProbeDiagnosticMarker(artifactPath); err != nil {
		return MicProbeFlashPlanResult{}, err
	}
	checksum, err := fileSHA256(artifactPath)
	if err != nil {
		return MicProbeFlashPlanResult{}, err
	}
	parts, err := bootstrapFlashParts(buildDir, coreDir, artifactPath)
	if err != nil {
		return MicProbeFlashPlanResult{}, err
	}
	return MicProbeFlashPlanResult{
		GuardID:                  "a21.firmware.mic_probe_flash_plan.v1",
		GeneratedAtMS:            generatedAtMS,
		DryRun:                   true,
		FlashAllowed:             false,
		NextRequiredConfirmation: "firmware-mic-probe-flash-execute_with_confirmation_token",
		Port:                     options.Port,
		ArtifactPath:             artifactPath,
		ArtifactSHA256:           checksum,
		Commit:                   options.ExpectedGitCommit,
		PlatformIOEnv:            StackChanMicProbePlatformIOEnv,
		BuildDir:                 buildDir,
		Parts:                    parts,
		OK:                       true,
	}, nil
}

func validateMicProbeBuildPath(buildDir string, artifactPath string) error {
	if filepath.Base(filepath.Clean(buildDir)) != StackChanMicProbePlatformIOEnv {
		return fmt.Errorf("mic probe build dir must end with %s", StackChanMicProbePlatformIOEnv)
	}
	if filepath.Base(filepath.Clean(artifactPath)) != "firmware.bin" {
		return fmt.Errorf("mic probe artifact must be PlatformIO firmware.bin")
	}
	expectedParent, err := filepath.Abs(filepath.Clean(buildDir))
	if err != nil {
		expectedParent = filepath.Clean(buildDir)
	}
	artifactParent, err := filepath.Abs(filepath.Dir(filepath.Clean(artifactPath)))
	if err != nil {
		artifactParent = filepath.Dir(filepath.Clean(artifactPath))
	}
	if expectedParent != artifactParent {
		return fmt.Errorf("mic probe artifact must be inside the mic probe build dir")
	}
	return nil
}

func validateMicProbeDiagnosticMarker(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Contains(data, []byte(MicProbeCapabilityStatus)) {
		return fmt.Errorf("mic probe firmware missing diagnostic status marker")
	}
	return nil
}
