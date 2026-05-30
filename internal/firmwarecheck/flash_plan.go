package firmwarecheck

import "fmt"

type FlashPlanOptions struct {
	ManifestPath      string
	ArtifactPath      string
	Port              string
	ReportPath        string
	ExpectedDeviceID  string
	ExpectedGitCommit string
	PortUsage         PortUsage
	MaxDeviceAgeMS    int64
	NowMS             int64
}

type FlashPlanResult struct {
	GuardID                  string               `json:"guard_id"`
	DryRun                   bool                 `json:"dry_run"`
	FlashAllowed             bool                 `json:"flash_allowed"`
	NextRequiredConfirmation string               `json:"next_required_confirmation"`
	Port                     string               `json:"port"`
	DeviceID                 string               `json:"device_id"`
	ArtifactPath             string               `json:"artifact_path"`
	ArtifactSHA256           string               `json:"artifact_sha256"`
	Commit                   string               `json:"commit"`
	Upload                   UploadCheckResult    `json:"upload"`
	DeviceIdentity           DeviceIdentityResult `json:"device_identity"`
	OK                       bool                 `json:"ok"`
}

func BuildFlashPlan(options FlashPlanOptions) (FlashPlanResult, error) {
	upload, err := ValidateUploadCandidate(UploadCheckOptions{
		ManifestPath: options.ManifestPath,
		ArtifactPath: options.ArtifactPath,
		Port:         options.Port,
		Commit:       options.ExpectedGitCommit,
	})
	if err != nil {
		return FlashPlanResult{}, err
	}
	if !options.PortUsage.Exists {
		return FlashPlanResult{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if options.PortUsage.InUse {
		return FlashPlanResult{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, options.PortUsage.Detail)
	}
	upload.PortUsage = options.PortUsage

	device, err := ValidateDeviceIdentity(DeviceIdentityOptions{
		ManifestPath:      options.ManifestPath,
		ArtifactPath:      options.ArtifactPath,
		ReportPath:        options.ReportPath,
		ExpectedDeviceID:  options.ExpectedDeviceID,
		ExpectedGitCommit: options.ExpectedGitCommit,
		MaxDeviceAgeMS:    options.MaxDeviceAgeMS,
		NowMS:             options.NowMS,
	})
	if err != nil {
		return FlashPlanResult{}, err
	}
	if upload.FlashAllowed || device.FlashAllowed {
		return FlashPlanResult{}, fmt.Errorf("upstream guard unexpectedly allowed flashing")
	}
	if upload.Artifact.SHA256 != device.Artifact.SHA256 {
		return FlashPlanResult{}, fmt.Errorf("upload and device identity guards reference different artifact checksums")
	}
	if !sameGitCommit(upload.Artifact.Commit, device.Artifact.Commit) {
		return FlashPlanResult{}, fmt.Errorf("upload and device identity guards reference different commits")
	}

	return FlashPlanResult{
		GuardID:                  "a21.firmware.flash_plan_guard.v1",
		DryRun:                   true,
		FlashAllowed:             false,
		NextRequiredConfirmation: "future_explicit_guarded_flash_command",
		Port:                     options.Port,
		DeviceID:                 options.ExpectedDeviceID,
		ArtifactPath:             upload.Artifact.ArtifactPath,
		ArtifactSHA256:           upload.Artifact.SHA256,
		Commit:                   upload.Artifact.Commit,
		Upload:                   upload,
		DeviceIdentity:           device,
		OK:                       true,
	}, nil
}
