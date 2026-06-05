package app

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func buildStackChanOfficialBaselineReport(options stackChanOfficialBaselineOptions) (stackChanOfficialBaselineReport, error) {
	sourceRoot := filepath.Clean(options.SourceRoot)
	if sourceRoot == "." || sourceRoot == "" {
		return stackChanOfficialBaselineReport{}, fmt.Errorf("source root is required")
	}
	if err := validateA21OfficialScratchDir(options.WorkDir); err != nil {
		return stackChanOfficialBaselineReport{}, fmt.Errorf("work dir invalid: %w", err)
	}
	if err := validateA21OfficialScratchDir(options.BuildDir); err != nil {
		return stackChanOfficialBaselineReport{}, fmt.Errorf("build dir invalid: %w", err)
	}
	gitRoot, err := gitOutput(sourceRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return stackChanOfficialBaselineReport{}, fmt.Errorf("source is not a git checkout")
	}
	sourceRoot = strings.TrimSpace(gitRoot)

	commit, err := gitOutput(sourceRoot, "rev-parse", "HEAD")
	if err != nil {
		return stackChanOfficialBaselineReport{}, fmt.Errorf("read source commit: %w", err)
	}
	statusOutput, _ := gitOutput(sourceRoot, "status", "--porcelain", "--untracked-files=all")
	dirtyFiles := countNonEmptyLines(statusOutput)

	report := stackChanOfficialBaselineReport{
		SchemaVersion:    stackChanOfficialBaselineSchema,
		GeneratedAtMS:    time.Now().UnixMilli(),
		Status:           "ready",
		SourceExportMode: "git_head_archive_read_only",
		Execute:          options.Execute,
		SourceRoot:       sourceRoot,
		SourceCommit:     strings.TrimSpace(commit),
		SourceClean:      dirtyFiles == 0,
		DirtyFileCount:   dirtyFiles,
		WorkDir:          filepath.Clean(options.WorkDir),
		BuildDir:         filepath.Clean(options.BuildDir),
		IDFExport:        filepath.Clean(options.IDFExport),
	}
	for _, overlay := range options.Overlays {
		report.Overlays = append(report.Overlays, stackChanOfficialBaselineOverlay{
			Path: filepath.Clean(overlay),
		})
	}
	report.Evidence = inspectStackChanOfficialEvidence(sourceRoot)
	applyStackChanOfficialCandidateContract(&report, sourceRoot, options.Overlays)
	if !report.SourceClean {
		report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
			Code:    "source_worktree_dirty",
			Message: "source checkout has local changes; A21 will export from git HEAD only",
		})
	}
	if !report.Evidence.TrackedSDKConfigHasNoLegacyIdentity {
		report.addFinding("tracked_config_legacy_identity", "tracked official sdkconfig contains forbidden legacy identity")
	}
	if !report.Evidence.HalMicTestUsesOutputData ||
		!report.Evidence.CoreS3UsesESPCodecDev ||
		!report.Evidence.CoreS3CreatesDuplexChannels {
		report.addFinding("missing_official_audio_codec", "official StackChan audio codec evidence is incomplete")
	}
	if !report.Evidence.ReposDeclareXiaoZhiAudioService {
		report.addFinding("missing_xiaozhi_audio_dependency", "official repos manifest does not declare xiaozhi audio service dependency")
	}
	if len(report.Findings) > 0 {
		for _, finding := range report.Findings {
			if strings.HasPrefix(finding.Code, "missing_") || finding.Code == "tracked_config_legacy_identity" {
				report.Status = "failed"
				break
			}
		}
	}
	return report, nil
}

func buildStackChanOfficialSmokeFlashReport(options stackChanOfficialSmokeFlashOptions) (stackChanOfficialSmokeFlashReport, error) {
	buildDir := filepath.Clean(options.BuildDir)
	if err := validateA21OfficialScratchDir(buildDir); err != nil {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("build dir invalid: %w", err)
	}
	if containsLegacyIdentityPathToken(buildDir) {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("build dir contains forbidden legacy identity")
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return stackChanOfficialSmokeFlashReport{}, err
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}
	parts, err := collectOfficialSmokeFlashParts(buildDir)
	if err != nil {
		return stackChanOfficialSmokeFlashReport{}, err
	}
	report := stackChanOfficialSmokeFlashReport{
		SchemaVersion:            stackChanOfficialAudioSmokeFlashSchema,
		GeneratedAtMS:            time.Now().UnixMilli(),
		Status:                   "ready",
		DryRun:                   !options.Execute,
		FlashAllowed:             false,
		FlashExecuted:            false,
		Port:                     options.Port,
		BuildDir:                 buildDir,
		IDFExport:                filepath.Clean(options.IDFExport),
		NextRequiredConfirmation: "stackchan-official-audio-smoke-flash-execute_with_confirmation_token",
		Parts:                    parts,
	}
	return report, nil
}

func buildStackChanOfficialBaselineFlashReport(options stackChanOfficialSmokeFlashOptions) (stackChanOfficialSmokeFlashReport, error) {
	buildDir := filepath.Clean(options.BuildDir)
	if err := validateA21OfficialScratchDir(buildDir); err != nil {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("build dir invalid: %w", err)
	}
	if containsLegacyIdentityPathToken(buildDir) {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("build dir contains forbidden legacy identity")
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return stackChanOfficialSmokeFlashReport{}, err
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}
	parts, err := collectOfficialBaselineFlashParts(buildDir)
	if err != nil {
		return stackChanOfficialSmokeFlashReport{}, err
	}
	report := stackChanOfficialSmokeFlashReport{
		SchemaVersion:            stackChanOfficialBaselineFlashSchema,
		GeneratedAtMS:            time.Now().UnixMilli(),
		Status:                   "ready",
		DryRun:                   !options.Execute,
		FlashAllowed:             false,
		FlashExecuted:            false,
		Port:                     options.Port,
		BuildDir:                 buildDir,
		IDFExport:                filepath.Clean(options.IDFExport),
		NextRequiredConfirmation: "stackchan-official-baseline-flash-execute_with_confirmation_token",
		Parts:                    parts,
	}
	return report, nil
}

func buildStackChanOfficialPCMBridgeFlashPlanReport(options stackChanOfficialPCMBridgeFlashPlanOptions) (stackChanOfficialPCMBridgeFlashPlanReport, error) {
	buildDir := filepath.Clean(options.BuildDir)
	if err := validateA21OfficialScratchDir(buildDir); err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, fmt.Errorf("build dir invalid: %w", err)
	}
	if containsLegacyIdentityPathToken(buildDir) {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, fmt.Errorf("build dir contains forbidden legacy identity")
	}
	deviceID := strings.TrimSpace(options.DeviceID)
	if err := validateOfficialPCMBridgeDeviceID(deviceID); err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, err
	}
	audioWS, err := parseOfficialPCMBridgeAudioWSURL(options.AudioWSURL, deviceID)
	if err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, err
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, err
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}
	parts, err := collectOfficialPCMBridgeFlashParts(buildDir)
	if err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, err
	}
	schema := stackChanOfficialPCMBridgeFlashPlanSchema
	if options.Execute {
		schema = stackChanOfficialPCMBridgeFlashExecutionSchema
	}
	return stackChanOfficialPCMBridgeFlashPlanReport{
		SchemaVersion:            schema,
		GeneratedAtMS:            time.Now().UnixMilli(),
		Status:                   "ready",
		DryRun:                   !options.Execute,
		FlashAllowed:             false,
		FlashExecuted:            false,
		Port:                     options.Port,
		BuildDir:                 buildDir,
		IDFExport:                filepath.Clean(options.IDFExport),
		DeviceID:                 deviceID,
		AudioWS:                  audioWS,
		NextRequiredConfirmation: "bridge_nvs_provisioning_and_flash_execute_guard_not_implemented",
		Parts:                    parts,
	}, nil
}

func buildStackChanOfficialXiaozhiCompatibleFlashReport(options stackChanOfficialXiaozhiCompatibleFlashOptions) (stackChanOfficialXiaozhiCompatibleFlashReport, error) {
	buildDir := filepath.Clean(options.BuildDir)
	if err := validateA21OfficialScratchDir(buildDir); err != nil {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("build dir invalid: %w", err)
	}
	if containsLegacyIdentityPathToken(buildDir) {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("build dir contains forbidden legacy identity")
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, err
	}
	esptoolBefore, err := validateStackChanOfficialXiaozhiCompatibleEsptoolBefore(options.EsptoolBefore)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, err
	}
	waitROMTimeoutSeconds := stackChanOfficialXiaozhiCompatibleWaitROMTimeoutSeconds(options)
	if options.WaitROM {
		if esptoolBefore != "no_reset" {
			return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("wait-rom requires --esptool-before no_reset so the manual ROM download state is not reset before flashing")
		}
		if waitROMTimeoutSeconds <= 0 || waitROMTimeoutSeconds > 300 {
			return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("wait-rom timeout seconds must be between 1 and 300")
		}
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}
	parts, err := collectOfficialXiaozhiCompatibleFlashParts(buildDir)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, err
	}
	schema := stackChanOfficialXiaozhiCompatibleFlashPlanSchema
	if options.Execute {
		schema = stackChanOfficialXiaozhiCompatibleFlashExecutionSchema
	}
	return stackChanOfficialXiaozhiCompatibleFlashReport{
		SchemaVersion:            schema,
		GeneratedAtMS:            time.Now().UnixMilli(),
		Status:                   "ready",
		FirmwareCandidate:        stackChanOfficialXiaozhiCompatibleFirmwareCandidate,
		BuildLaneRole:            "product_candidate",
		DryRun:                   !options.Execute,
		FlashAllowed:             false,
		FlashExecuted:            false,
		Port:                     options.Port,
		EsptoolBefore:            esptoolBefore,
		WaitROM:                  options.WaitROM,
		WaitROMTimeoutSeconds:    waitROMTimeoutSeconds,
		BuildDirName:             filepath.Base(buildDir),
		IDFExportName:            filepath.Base(filepath.Clean(options.IDFExport)),
		NextRequiredConfirmation: "a21-stackchan-official-xiaozhi-compatible-flash-execute_with_confirmation_token",
		Parts:                    parts,
	}, nil
}

func buildStackChanOfficialXiaozhiCompatibleNVSReport(options stackChanOfficialXiaozhiCompatibleNVSOptions) (stackChanOfficialXiaozhiCompatibleNVSReport, error) {
	runDir := filepath.Clean(options.RunDir)
	if err := validateA21OfficialRunDir(runDir); err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, fmt.Errorf("run dir invalid: %w", err)
	}
	ota, err := parseOfficialXiaozhiNVSOTAURL(options.OTAURL)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, err
	}
	websocket, err := parseOfficialXiaozhiNVSWebSocketURL(options.WebSocketURL, options.WebSocketVersion)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, err
	}
	wifiCredentialsRequested, err := validateOfficialXiaozhiNVSWiFiCredentials(options.WiFiSSID, options.WiFiPassword)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, err
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, err
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}

	schema := stackChanOfficialXiaozhiCompatibleNVSPlanSchema
	if options.Execute {
		schema = stackChanOfficialXiaozhiCompatibleNVSExecutionSchema
	}
	return stackChanOfficialXiaozhiCompatibleNVSReport{
		SchemaVersion: schema,
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "ready",
		DryRun:        !options.Execute,
		WriteAllowed:  false,
		WriteExecuted: false,
		Port:          options.Port,
		IDFExport:     filepath.Clean(options.IDFExport),
		RunDir:        runDir,
		OTA:           ota,
		WebSocket:     websocket,
		Partition: stackChanOfficialPCMBridgeNVSPartition{
			Offset:    stackChanOfficialPCMBridgeNVSOffset,
			SizeHex:   stackChanOfficialPCMBridgeNVSSizeHex,
			SizeBytes: stackChanOfficialPCMBridgeNVSSizeBytes,
		},
		Safety: stackChanOfficialXiaozhiCompatibleNVSSafety{
			BackupBeforeWrite:                 true,
			PreserveExistingEntries:           true,
			OnlyMutatesXiaozhiConnectionKeys:  true,
			PreservesWiFiCredentials:          !wifiCredentialsRequested,
			AllowsExplicitWiFiCredentialWrite: wifiCredentialsRequested,
			ReportRedactsValues:               true,
		},
		Tools: stackChanOfficialPCMBridgeNVSTools{
			NVSToolPath:       officialIDFToolPath(options.IDFExport, "components/nvs_flash/nvs_partition_tool/nvs_tool.py"),
			NVSGeneratorPath:  officialIDFToolPath(options.IDFExport, "components/nvs_flash/nvs_partition_generator/nvs_partition_gen.py"),
			EsptoolModuleName: "esptool",
			IDFPythonPath:     options.IDFPython,
		},
		NextRequiredConfirmation: "a21-stackchan-official-xiaozhi-compatible-nvs-execute_with_confirmation_token",
	}, nil
}

func buildStackChanOfficialPCMBridgeNVSReport(options stackChanOfficialPCMBridgeNVSOptions) (stackChanOfficialPCMBridgeNVSReport, error) {
	runDir := filepath.Clean(options.RunDir)
	if err := validateA21OfficialRunDir(runDir); err != nil {
		return stackChanOfficialPCMBridgeNVSReport{}, fmt.Errorf("run dir invalid: %w", err)
	}
	deviceID := strings.TrimSpace(options.DeviceID)
	if err := validateOfficialPCMBridgeDeviceID(deviceID); err != nil {
		return stackChanOfficialPCMBridgeNVSReport{}, err
	}
	audioWS, err := parseOfficialPCMBridgeAudioWSURL(options.AudioWSURL, deviceID)
	if err != nil {
		return stackChanOfficialPCMBridgeNVSReport{}, err
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return stackChanOfficialPCMBridgeNVSReport{}, err
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return stackChanOfficialPCMBridgeNVSReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return stackChanOfficialPCMBridgeNVSReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return stackChanOfficialPCMBridgeNVSReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}

	schema := stackChanOfficialPCMBridgeNVSPlanSchema
	if options.Execute {
		schema = stackChanOfficialPCMBridgeNVSExecutionSchema
	}
	return stackChanOfficialPCMBridgeNVSReport{
		SchemaVersion: schema,
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "ready",
		DryRun:        !options.Execute,
		WriteAllowed:  false,
		WriteExecuted: false,
		Port:          options.Port,
		IDFExport:     filepath.Clean(options.IDFExport),
		RunDir:        runDir,
		DeviceID:      deviceID,
		AudioWS:       audioWS,
		Partition: stackChanOfficialPCMBridgeNVSPartition{
			Offset:    stackChanOfficialPCMBridgeNVSOffset,
			SizeHex:   stackChanOfficialPCMBridgeNVSSizeHex,
			SizeBytes: stackChanOfficialPCMBridgeNVSSizeBytes,
		},
		Safety: stackChanOfficialPCMBridgeNVSSafety{
			BackupBeforeWrite:       true,
			PreserveExistingEntries: true,
			OnlyMutatesA21Namespace: true,
			ReportRedactsValues:     true,
		},
		Tools: stackChanOfficialPCMBridgeNVSTools{
			NVSToolPath:       officialIDFToolPath(options.IDFExport, "components/nvs_flash/nvs_partition_tool/nvs_tool.py"),
			NVSGeneratorPath:  officialIDFToolPath(options.IDFExport, "components/nvs_flash/nvs_partition_generator/nvs_partition_gen.py"),
			EsptoolModuleName: "esptool",
		},
		NextRequiredConfirmation: "stackchan-official-pcm-bridge-nvs-execute_with_confirmation_token",
	}, nil
}

func collectOfficialSmokeFlashParts(buildDir string) ([]stackChanOfficialSmokeFlashPart, error) {
	return collectOfficialFlashPartsForApp(buildDir, "a21-stackchan-official-audio-smoke.bin")
}

func collectOfficialBaselineFlashParts(buildDir string) ([]stackChanOfficialSmokeFlashPart, error) {
	return collectOfficialFlashPartsForApp(buildDir, "stack-chan.bin")
}

func collectOfficialPCMBridgeFlashParts(buildDir string) ([]stackChanOfficialSmokeFlashPart, error) {
	return collectOfficialFlashPartsForApp(buildDir, "a21-stackchan-official-pcm-bridge.bin")
}

func collectOfficialXiaozhiCompatibleFlashParts(buildDir string) ([]stackChanOfficialXiaozhiFlashPart, error) {
	fullParts, err := collectOfficialFlashPartsForApp(buildDir, stackChanOfficialXiaozhiCompatibleAppBinary)
	if err != nil {
		return nil, err
	}
	parts := make([]stackChanOfficialXiaozhiFlashPart, 0, len(fullParts))
	for _, part := range fullParts {
		parts = append(parts, stackChanOfficialXiaozhiFlashPart{
			Name:      part.Name,
			Offset:    part.Offset,
			File:      filepath.Base(part.Path),
			SHA256:    part.SHA256,
			SizeBytes: part.SizeBytes,
		})
	}
	return parts, nil
}

type officialStackChanProductLaneEvidenceCandidate struct {
	evidence     officialStackChanProductLaneArtifactEvidence
	priority     int
	observedAtMS int64
}

type officialStackChanProductLaneRawPart struct {
	Name        string `json:"name"`
	Offset      string `json:"offset"`
	FlashOffset string `json:"flash_offset"`
	File        string `json:"file"`
	Path        string `json:"path"`
	SHA256      string `json:"sha256"`
}

type officialStackChanProductLaneRawReport struct {
	SchemaVersion     string                                `json:"schema_version"`
	GeneratedAtMS     int64                                 `json:"generated_at_ms"`
	Status            string                                `json:"status"`
	FirmwareCandidate string                                `json:"firmware_candidate"`
	BuildLaneRole     string                                `json:"build_lane_role"`
	DryRun            bool                                  `json:"dry_run"`
	FlashExecuted     bool                                  `json:"flash_executed"`
	Parts             []officialStackChanProductLaneRawPart `json:"parts"`
	Build             officialStackChanProductLaneRawBuild  `json:"build"`
}

type officialStackChanProductLaneRawBuild struct {
	BuildExecuted bool                                  `json:"build_executed"`
	Artifacts     []officialStackChanProductLaneRawPart `json:"artifacts"`
}

func discoverOfficialStackChanProductLaneArtifactEvidence(reportDir string) *officialStackChanProductLaneArtifactEvidence {
	reportDir = filepath.Clean(strings.TrimSpace(reportDir))
	if reportDir == "" || reportDir == "." || containsLegacyIdentityPathToken(reportDir) {
		return nil
	}
	info, err := os.Stat(reportDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	patterns := []string{
		"a21-stackchan-official-xiaozhi-compatible-flash-*.json",
		"a21-stackchan-official-baseline-*.json",
	}
	candidates := make([]officialStackChanProductLaneEvidenceCandidate, 0)
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(reportDir, pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			candidate, ok := readOfficialStackChanProductLaneArtifactEvidence(match)
			if ok {
				candidates = append(candidates, candidate)
			}
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].priority != candidates[j].priority {
			return candidates[i].priority > candidates[j].priority
		}
		if candidates[i].observedAtMS != candidates[j].observedAtMS {
			return candidates[i].observedAtMS > candidates[j].observedAtMS
		}
		return candidates[i].evidence.ReportPath > candidates[j].evidence.ReportPath
	})
	evidence := candidates[0].evidence
	return &evidence
}

func readOfficialStackChanProductLaneArtifactEvidence(reportPath string) (officialStackChanProductLaneEvidenceCandidate, bool) {
	cleanPath := filepath.Clean(reportPath)
	if containsLegacyIdentityPathToken(cleanPath) {
		return officialStackChanProductLaneEvidenceCandidate{}, false
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return officialStackChanProductLaneEvidenceCandidate{}, false
	}
	var raw officialStackChanProductLaneRawReport
	if err := json.Unmarshal(data, &raw); err != nil {
		return officialStackChanProductLaneEvidenceCandidate{}, false
	}
	generatedAtMS := raw.GeneratedAtMS
	if generatedAtMS == 0 {
		if info, err := os.Stat(cleanPath); err == nil {
			generatedAtMS = info.ModTime().UnixMilli()
		}
	}
	switch raw.SchemaVersion {
	case stackChanOfficialXiaozhiCompatibleFlashExecutionSchema, stackChanOfficialXiaozhiCompatibleFlashPlanSchema:
		file, sha, ok := officialXiaozhiCompatibleArtifactFromParts(raw.Parts)
		if !ok ||
			strings.TrimSpace(raw.FirmwareCandidate) != stackChanOfficialXiaozhiCompatibleFirmwareCandidate ||
			strings.TrimSpace(raw.BuildLaneRole) != "product_candidate" {
			return officialStackChanProductLaneEvidenceCandidate{}, false
		}
		status := "planned"
		source := "official_xiaozhi_compatible_flash_plan"
		priority := 1
		if raw.SchemaVersion == stackChanOfficialXiaozhiCompatibleFlashExecutionSchema &&
			raw.FlashExecuted &&
			!raw.DryRun &&
			strings.TrimSpace(raw.Status) == "passed" {
			status = "satisfied"
			source = "official_xiaozhi_compatible_flash_execution"
			priority = 3
		}
		return officialStackChanProductLaneEvidenceCandidate{
			evidence: officialStackChanProductLaneArtifactEvidence{
				SchemaVersion:       officialProductLaneArtifactEvidenceSchema,
				Status:              status,
				Source:              source,
				ReportPath:          cleanPath,
				SourceSchemaVersion: raw.SchemaVersion,
				FirmwareCandidate:   raw.FirmwareCandidate,
				BuildLaneRole:       raw.BuildLaneRole,
				ArtifactFile:        file,
				ArtifactSHA256:      sha,
				FlashExecuted:       raw.FlashExecuted,
				DryRun:              raw.DryRun,
				GeneratedAtMS:       generatedAtMS,
			},
			priority:     priority,
			observedAtMS: generatedAtMS,
		}, true
	case stackChanOfficialBaselineSchema:
		file, sha, ok := officialXiaozhiCompatibleArtifactFromParts(raw.Build.Artifacts)
		if !ok ||
			strings.TrimSpace(raw.FirmwareCandidate) != stackChanOfficialXiaozhiCompatibleFirmwareCandidate ||
			strings.TrimSpace(raw.BuildLaneRole) != "product_candidate" ||
			strings.TrimSpace(raw.Status) != "passed" ||
			!raw.Build.BuildExecuted {
			return officialStackChanProductLaneEvidenceCandidate{}, false
		}
		return officialStackChanProductLaneEvidenceCandidate{
			evidence: officialStackChanProductLaneArtifactEvidence{
				SchemaVersion:       officialProductLaneArtifactEvidenceSchema,
				Status:              "build_available",
				Source:              "official_xiaozhi_compatible_build",
				ReportPath:          cleanPath,
				SourceSchemaVersion: raw.SchemaVersion,
				FirmwareCandidate:   raw.FirmwareCandidate,
				BuildLaneRole:       raw.BuildLaneRole,
				ArtifactFile:        file,
				ArtifactSHA256:      sha,
				DryRun:              false,
				GeneratedAtMS:       generatedAtMS,
			},
			priority:     2,
			observedAtMS: generatedAtMS,
		}, true
	default:
		return officialStackChanProductLaneEvidenceCandidate{}, false
	}
}

func officialXiaozhiCompatibleArtifactFromParts(parts []officialStackChanProductLaneRawPart) (string, string, bool) {
	for _, part := range parts {
		if strings.TrimSpace(part.Name) != "app" {
			continue
		}
		name := filepath.Base(firstNonEmpty(strings.TrimSpace(part.File), strings.TrimSpace(part.Path)))
		if name != stackChanOfficialXiaozhiCompatibleAppBinary {
			return "", "", false
		}
		offset := firstNonEmpty(strings.TrimSpace(part.Offset), strings.TrimSpace(part.FlashOffset))
		if offset != "" && offset != "0x20000" {
			return "", "", false
		}
		sha := strings.ToLower(strings.TrimSpace(part.SHA256))
		if sha != "" && !isSHA256Hex(sha) {
			return "", "", false
		}
		return name, sha, true
	}
	return "", "", false
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func officialProductLaneEvidenceSatisfiesArtifact(evidence *officialStackChanProductLaneArtifactEvidence) bool {
	if evidence == nil {
		return false
	}
	return evidence.Status == "satisfied" || evidence.Status == "build_available"
}
