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
const StackChanIMUProbePlatformIOEnv = "a21_stackchan_cores3_imu_probe"
const IMUProbeCapabilityStatus = "diagnostic_probe_m5unified_imu"
const StackChanSensorProbePlatformIOEnv = "a21_stackchan_cores3_sensor_probe"
const SensorProbeAmbientLightCapabilityStatus = "diagnostic_probe_ltr553_ambient_light"
const SensorProbeProximityCapabilityStatus = "diagnostic_probe_ltr553_proximity"
const SensorProbeBatteryCapabilityStatus = "diagnostic_probe_ina226_battery"

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

type IMUProbeFlashPlanOptions = MicProbeFlashPlanOptions
type IMUProbeFlashPlanResult = MicProbeFlashPlanResult
type SensorProbeFlashPlanOptions = MicProbeFlashPlanOptions
type SensorProbeFlashPlanResult = MicProbeFlashPlanResult

type diagnosticProbeFlashConfig struct {
	Label                    string
	PlatformIOEnv            string
	CapabilityStatus         string
	CapabilityStatuses       []string
	GuardID                  string
	NextRequiredConfirmation string
}

func BuildMicProbeFlashPlan(options MicProbeFlashPlanOptions) (MicProbeFlashPlanResult, error) {
	return buildDiagnosticProbeFlashPlan(options, diagnosticProbeFlashConfig{
		Label:                    "mic probe",
		PlatformIOEnv:            StackChanMicProbePlatformIOEnv,
		CapabilityStatus:         MicProbeCapabilityStatus,
		GuardID:                  "a21.firmware.mic_probe_flash_plan.v1",
		NextRequiredConfirmation: "firmware-mic-probe-flash-execute_with_confirmation_token",
	})
}

func BuildIMUProbeFlashPlan(options IMUProbeFlashPlanOptions) (IMUProbeFlashPlanResult, error) {
	return buildDiagnosticProbeFlashPlan(options, diagnosticProbeFlashConfig{
		Label:                    "IMU probe",
		PlatformIOEnv:            StackChanIMUProbePlatformIOEnv,
		CapabilityStatus:         IMUProbeCapabilityStatus,
		GuardID:                  "a21.firmware.imu_probe_flash_plan.v1",
		NextRequiredConfirmation: "firmware-imu-probe-flash-execute_with_confirmation_token",
	})
}

func BuildSensorProbeFlashPlan(options SensorProbeFlashPlanOptions) (SensorProbeFlashPlanResult, error) {
	return buildDiagnosticProbeFlashPlan(options, diagnosticProbeFlashConfig{
		Label:         "sensor probe",
		PlatformIOEnv: StackChanSensorProbePlatformIOEnv,
		CapabilityStatuses: []string{
			SensorProbeAmbientLightCapabilityStatus,
			SensorProbeProximityCapabilityStatus,
			SensorProbeBatteryCapabilityStatus,
		},
		GuardID:                  "a21.firmware.sensor_probe_flash_plan.v1",
		NextRequiredConfirmation: "firmware-sensor-probe-flash-execute_with_confirmation_token",
	})
}

func buildDiagnosticProbeFlashPlan(options MicProbeFlashPlanOptions, config diagnosticProbeFlashConfig) (MicProbeFlashPlanResult, error) {
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
		buildDir = filepath.Join("firmware", "stackchan", ".pio", "build", config.PlatformIOEnv)
	}
	coreDir := options.CoreDir
	if coreDir == "" {
		coreDir = filepath.Join(".a21-tools", "platformio-core")
	}
	artifactPath := options.ArtifactPath
	if artifactPath == "" {
		artifactPath = filepath.Join(buildDir, "firmware.bin")
	}
	if err := validateDiagnosticProbeBuildPath(config.Label, config.PlatformIOEnv, buildDir, artifactPath); err != nil {
		return MicProbeFlashPlanResult{}, err
	}
	if containsForbiddenPackagePathIdentity(artifactPath) || containsForbiddenPackagePathIdentity(buildDir) || containsForbiddenPackagePathIdentity(coreDir) {
		return MicProbeFlashPlanResult{}, fmt.Errorf("%s flash path contains forbidden legacy identity", config.Label)
	}
	if err := validateEmbeddedArtifactIdentity(artifactPath, manifestResult.Manifest, options.ExpectedGitCommit); err != nil {
		return MicProbeFlashPlanResult{}, fmt.Errorf("%s firmware identity invalid: %w", config.Label, err)
	}
	if err := validateDiagnosticProbeMarkers(config.Label, config.capabilityStatuses(), artifactPath); err != nil {
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
		GuardID:                  config.GuardID,
		GeneratedAtMS:            generatedAtMS,
		DryRun:                   true,
		FlashAllowed:             false,
		NextRequiredConfirmation: config.NextRequiredConfirmation,
		Port:                     options.Port,
		ArtifactPath:             artifactPath,
		ArtifactSHA256:           checksum,
		Commit:                   options.ExpectedGitCommit,
		PlatformIOEnv:            config.PlatformIOEnv,
		BuildDir:                 buildDir,
		Parts:                    parts,
		OK:                       true,
	}, nil
}

func validateMicProbeBuildPath(buildDir string, artifactPath string) error {
	return validateDiagnosticProbeBuildPath("mic probe", StackChanMicProbePlatformIOEnv, buildDir, artifactPath)
}

func validateDiagnosticProbeBuildPath(label string, platformIOEnv string, buildDir string, artifactPath string) error {
	if filepath.Base(filepath.Clean(buildDir)) != platformIOEnv {
		return fmt.Errorf("%s build dir must end with %s", label, platformIOEnv)
	}
	if filepath.Base(filepath.Clean(artifactPath)) != "firmware.bin" {
		return fmt.Errorf("%s artifact must be PlatformIO firmware.bin", label)
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
		return fmt.Errorf("%s artifact must be inside the diagnostic probe build dir", label)
	}
	return nil
}

func validateMicProbeDiagnosticMarker(path string) error {
	return validateDiagnosticProbeMarker("mic probe", MicProbeCapabilityStatus, path)
}

func validateDiagnosticProbeMarker(label string, capabilityStatus string, path string) error {
	return validateDiagnosticProbeMarkers(label, []string{capabilityStatus}, path)
}

func (config diagnosticProbeFlashConfig) capabilityStatuses() []string {
	if len(config.CapabilityStatuses) > 0 {
		return config.CapabilityStatuses
	}
	if config.CapabilityStatus != "" {
		return []string{config.CapabilityStatus}
	}
	return nil
}

func validateDiagnosticProbeMarkers(label string, capabilityStatuses []string, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, capabilityStatus := range capabilityStatuses {
		if !bytes.Contains(data, []byte(capabilityStatus)) {
			return fmt.Errorf("%s firmware missing diagnostic status marker", label)
		}
	}
	return nil
}
