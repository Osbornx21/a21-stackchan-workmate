package firmwarecheck

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type DeviceIdentityOptions struct {
	ManifestPath      string
	ArtifactPath      string
	ReportPath        string
	ExpectedDeviceID  string
	ExpectedGitCommit string
}

type DeviceIdentityFirmware struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
	Board   string `json:"board,omitempty"`
	Commit  string `json:"commit,omitempty"`
}

type DeviceIdentityRecord struct {
	DeviceID       string                 `json:"device_id"`
	Firmware       DeviceIdentityFirmware `json:"firmware"`
	IdentityStatus string                 `json:"identity_status"`
	IdentityError  string                 `json:"identity_error,omitempty"`
	LastSeenMS     int64                  `json:"last_seen_ms,omitempty"`
}

type DeviceIdentityResult struct {
	GuardID                  string               `json:"guard_id"`
	DeviceIdentityConfirmed  bool                 `json:"device_identity_confirmed"`
	FlashAllowed             bool                 `json:"flash_allowed"`
	NextRequiredConfirmation string               `json:"next_required_confirmation"`
	ExpectedDeviceID         string               `json:"expected_device_id"`
	Device                   DeviceIdentityRecord `json:"device"`
	Artifact                 ArtifactResult       `json:"artifact"`
	OK                       bool                 `json:"ok"`
}

type gatewayDeviceReport struct {
	Devices []DeviceIdentityRecord `json:"devices"`
}

func ValidateDeviceIdentity(options DeviceIdentityOptions) (DeviceIdentityResult, error) {
	if err := validateExpectedDeviceID(options.ExpectedDeviceID); err != nil {
		return DeviceIdentityResult{}, err
	}
	if err := validateExpectedCommit(options.ExpectedGitCommit); err != nil {
		return DeviceIdentityResult{}, err
	}
	if options.ReportPath == "" {
		return DeviceIdentityResult{}, fmt.Errorf("device report path is required")
	}

	artifact, err := ValidateArtifact(ArtifactOptions{
		ManifestPath:        options.ManifestPath,
		ArtifactPath:        options.ArtifactPath,
		RequireReleaseIndex: true,
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
		Device:                   device,
		Artifact:                 artifact,
		OK:                       true,
	}, nil
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
	return report, nil
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
