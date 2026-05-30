package firmwarecheck

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type DeviceIdentityOptions struct {
	ManifestPath      string
	ArtifactPath      string
	ReportPath        string
	ExpectedDeviceID  string
	ExpectedGitCommit string
	MaxDeviceAgeMS    int64
	NowMS             int64
}

type DeviceIdentityFirmware struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
	Board   string `json:"board,omitempty"`
	Commit  string `json:"commit,omitempty"`
}

type DeviceIdentityRecord struct {
	DeviceID          string                 `json:"device_id"`
	Firmware          DeviceIdentityFirmware `json:"firmware"`
	Capabilities      map[string]string      `json:"capabilities,omitempty"`
	IdentityStatus    string                 `json:"identity_status"`
	IdentityError     string                 `json:"identity_error,omitempty"`
	ConnectionStatus  string                 `json:"connection_status,omitempty"`
	DeviceAgeMS       int64                  `json:"device_age_ms,omitempty"`
	CurrentMode       string                 `json:"current_mode,omitempty"`
	CurrentExpression string                 `json:"current_expression,omitempty"`
	PlaybackStreamID  string                 `json:"playback_stream_id,omitempty"`
	LastSeenMS        int64                  `json:"last_seen_ms,omitempty"`
}

type DeviceIdentityResult struct {
	GuardID                  string               `json:"guard_id"`
	DeviceIdentityConfirmed  bool                 `json:"device_identity_confirmed"`
	FlashAllowed             bool                 `json:"flash_allowed"`
	NextRequiredConfirmation string               `json:"next_required_confirmation"`
	ExpectedDeviceID         string               `json:"expected_device_id"`
	DeviceAgeMS              int64                `json:"device_age_ms,omitempty"`
	MaxDeviceAgeMS           int64                `json:"max_device_age_ms,omitempty"`
	Device                   DeviceIdentityRecord `json:"device"`
	Artifact                 ArtifactResult       `json:"artifact"`
	OK                       bool                 `json:"ok"`
}

const (
	GatewayDeviceReportSchemaVersion = "a21.gateway.devices.v1"
	GatewayDeviceReportServiceName   = "a21-gateway"
)

type gatewayDeviceReport struct {
	SchemaVersion        string                 `json:"schema_version"`
	Service              string                 `json:"service"`
	GatewaySchemaVersion string                 `json:"gateway_schema_version"`
	GatewayService       string                 `json:"gateway_service"`
	Devices              []DeviceIdentityRecord `json:"devices"`
}

func ValidateDeviceIdentity(options DeviceIdentityOptions) (DeviceIdentityResult, error) {
	if err := validateExpectedDeviceID(options.ExpectedDeviceID); err != nil {
		return DeviceIdentityResult{}, err
	}
	if err := validateExpectedCommit(options.ExpectedGitCommit); err != nil {
		return DeviceIdentityResult{}, err
	}
	if options.MaxDeviceAgeMS <= 0 {
		return DeviceIdentityResult{}, fmt.Errorf("max device age guard is required for firmware device identity check")
	}
	if options.ReportPath == "" {
		return DeviceIdentityResult{}, fmt.Errorf("device report path is required")
	}

	artifact, err := ValidateArtifact(ArtifactOptions{
		ManifestPath:           options.ManifestPath,
		ArtifactPath:           options.ArtifactPath,
		RequireReleaseIndex:    true,
		RequireReleaseManifest: true,
	})
	if err != nil {
		return DeviceIdentityResult{}, err
	}
	if !sameGitCommit(options.ExpectedGitCommit, artifact.Commit) {
		return DeviceIdentityResult{}, fmt.Errorf("artifact commit %q does not match expected commit %q", artifact.Commit, options.ExpectedGitCommit)
	}

	report, err := readGatewayDeviceReport(options.ReportPath)
	if err != nil {
		return DeviceIdentityResult{}, err
	}
	device, ok := findDeviceIdentity(report, options.ExpectedDeviceID)
	if !ok {
		return DeviceIdentityResult{}, fmt.Errorf("device %q not found in Gateway report", options.ExpectedDeviceID)
	}
	if strings.Contains(strings.ToLower(device.DeviceID), "x21") || strings.Contains(strings.ToLower(device.DeviceID), "v21") {
		return DeviceIdentityResult{}, fmt.Errorf("device id contains forbidden legacy identity")
	}
	if device.IdentityStatus != "ok" {
		if device.IdentityError != "" {
			return DeviceIdentityResult{}, fmt.Errorf("device identity status %q: %s", device.IdentityStatus, device.IdentityError)
		}
		return DeviceIdentityResult{}, fmt.Errorf("device identity status %q", device.IdentityStatus)
	}
	if err := validateDeviceConnectionStatus(device); err != nil {
		return DeviceIdentityResult{}, err
	}
	deviceAgeMS, err := validateDeviceReportAge(device, options)
	if err != nil {
		return DeviceIdentityResult{}, err
	}
	if device.Firmware.ID != artifact.Manifest.FirmwareID {
		return DeviceIdentityResult{}, fmt.Errorf("device firmware_id %q does not match artifact %q", device.Firmware.ID, artifact.Manifest.FirmwareID)
	}
	if device.Firmware.Version != artifact.Manifest.Version {
		return DeviceIdentityResult{}, fmt.Errorf("device firmware version %q does not match artifact %q", device.Firmware.Version, artifact.Manifest.Version)
	}
	if device.Firmware.Board != artifact.Manifest.Board {
		return DeviceIdentityResult{}, fmt.Errorf("device firmware board %q does not match artifact %q", device.Firmware.Board, artifact.Manifest.Board)
	}
	if !sameGitCommit(device.Firmware.Commit, artifact.Commit) {
		return DeviceIdentityResult{}, fmt.Errorf("device firmware commit %q does not match artifact commit %q", device.Firmware.Commit, artifact.Commit)
	}

	return DeviceIdentityResult{
		GuardID:                  "a21.firmware.device_identity_guard.v1",
		DeviceIdentityConfirmed:  true,
		FlashAllowed:             false,
		NextRequiredConfirmation: "explicit_guarded_flash_command",
		ExpectedDeviceID:         options.ExpectedDeviceID,
		DeviceAgeMS:              deviceAgeMS,
		MaxDeviceAgeMS:           options.MaxDeviceAgeMS,
		Device:                   device,
		Artifact:                 artifact,
		OK:                       true,
	}, nil
}

func validateDeviceConnectionStatus(device DeviceIdentityRecord) error {
	if device.ConnectionStatus != "online" {
		return fmt.Errorf("device connection_status %q is not online", device.ConnectionStatus)
	}
	return nil
}

func validateDeviceReportAge(device DeviceIdentityRecord, options DeviceIdentityOptions) (int64, error) {
	if options.MaxDeviceAgeMS <= 0 {
		return 0, nil
	}
	if device.LastSeenMS <= 0 {
		return 0, fmt.Errorf("device report missing last_seen_ms for freshness guard")
	}
	nowMS := options.NowMS
	if nowMS <= 0 {
		nowMS = time.Now().UnixMilli()
	}
	ageMS := nowMS - device.LastSeenMS
	if ageMS < 0 {
		return 0, fmt.Errorf("device report last_seen_ms is in the future")
	}
	if ageMS > options.MaxDeviceAgeMS {
		return ageMS, fmt.Errorf("device report stale: last_seen_ms age %dms exceeds max %dms", ageMS, options.MaxDeviceAgeMS)
	}
	return ageMS, nil
}

func readGatewayDeviceReport(path string) (gatewayDeviceReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return gatewayDeviceReport{}, err
	}
	var report gatewayDeviceReport
	if err := json.Unmarshal(data, &report); err != nil {
		return gatewayDeviceReport{}, err
	}
	if err := validateGatewayDeviceReportIdentity(report); err != nil {
		return gatewayDeviceReport{}, err
	}
	return report, nil
}

func validateGatewayDeviceReportIdentity(report gatewayDeviceReport) error {
	schemaVersion := strings.TrimSpace(report.SchemaVersion)
	service := strings.TrimSpace(report.Service)
	if schemaVersion == "a21.firmware.device_report.v1" {
		schemaVersion = strings.TrimSpace(report.GatewaySchemaVersion)
		service = strings.TrimSpace(report.GatewayService)
	}
	if containsLegacyIdentity(schemaVersion) || containsLegacyIdentity(service) {
		return fmt.Errorf("gateway device report contains forbidden legacy identity")
	}
	if schemaVersion != GatewayDeviceReportSchemaVersion ||
		service != GatewayDeviceReportServiceName {
		return fmt.Errorf("gateway device report is missing required A21 Gateway identity")
	}
	return nil
}

func containsLegacyIdentity(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "x21") || strings.Contains(lower, "v21")
}

func findDeviceIdentity(report gatewayDeviceReport, deviceID string) (DeviceIdentityRecord, bool) {
	for _, device := range report.Devices {
		if device.DeviceID == deviceID {
			return device, true
		}
	}
	return DeviceIdentityRecord{}, false
}

func validateExpectedDeviceID(deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("expected device id is required")
	}
	if strings.Contains(strings.ToLower(deviceID), "x21") || strings.Contains(strings.ToLower(deviceID), "v21") {
		return fmt.Errorf("expected device id contains forbidden legacy identity")
	}
	return nil
}
