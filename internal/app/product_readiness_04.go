package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func productVoiceChainReadinessReportPathSafe(path string) bool {
	path = strings.TrimSpace(strings.ToLower(path))
	return path == "" ||
		(!strings.Contains(path, "http://") &&
			!strings.Contains(path, "https://") &&
			!strings.Contains(path, "/users/") &&
			!strings.Contains(path, "secret") &&
			!strings.Contains(path, "token") &&
			!containsLegacyIdentity(path))
}

func productVoiceChainReadinessReportContainsForbiddenValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			if productVoiceChainReadinessReportContainsForbiddenValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productVoiceChainReadinessReportContainsForbiddenValue(child) {
				return true
			}
		}
	case string:
		lower := strings.ToLower(typed)
		if strings.TrimSpace(lower) != "v21_adapter_only" && containsLegacyIdentity(typed) {
			return true
		}
		for _, forbidden := range []string{
			"http://",
			"https://",
			"/users/",
			"bearer ",
			"sk-",
			"raw prompt",
			"prompt text",
			"raw transcript",
			"transcript text",
			"raw provider output",
			"provider output",
			"raw reasoning",
			"reasoning text",
			"data_base64",
			"audio_base64",
			"secret-value",
		} {
			if strings.Contains(lower, forbidden) {
				return true
			}
		}
	}
	return false
}

func productStringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

type productWakeWordFirmwarePlanEvidence struct {
	Valid                 bool
	Status                string
	SourceReport          string
	DryRun                bool
	FirmwareBuildRequired bool
	BuildAllowed          bool
	FlashAllowed          bool
	Mode                  string
	DesiredPhrase         string
	DesiredPinyin         string
	Threshold             int
}

type productWakeWordFirmwarePlanFixture struct {
	SchemaVersion            string `json:"schema_version"`
	Status                   string `json:"status"`
	DryRun                   *bool  `json:"dry_run"`
	FirmwareBuildRequired    *bool  `json:"firmware_build_required"`
	BuildAllowed             *bool  `json:"build_allowed"`
	FlashAllowed             *bool  `json:"flash_allowed"`
	FirmwareID               string `json:"firmware_id"`
	TargetBoard              string `json:"target_board"`
	TargetProfile            string `json:"target_profile"`
	GuardTier                string `json:"guard_tier"`
	Mode                     string `json:"mode"`
	DesiredPhrase            string `json:"desired_phrase"`
	DesiredPinyin            string `json:"desired_pinyin"`
	Threshold                int    `json:"threshold"`
	RuntimeStatus            string `json:"runtime_status"`
	NextRequiredConfirmation string `json:"next_required_confirmation"`
	ReportPath               string `json:"report_path"`
}

func loadProductWakeWordFirmwarePlanEvidence(path string) (productWakeWordFirmwarePlanEvidence, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productWakeWordFirmwarePlanEvidence{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productWakeWordFirmwarePlanContainsForbiddenValue(raw) {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	var fixture productWakeWordFirmwarePlanFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	if missingField := missingProductWakeWordFirmwarePlanField(fixture); missingField != "" {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{missingProductWakeWordFirmwarePlanFieldFinding(missingField)}
	}
	if !validProductWakeWordFirmwarePlanFixture(fixture) {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	return productWakeWordFirmwarePlanEvidence{
		Valid:                 true,
		Status:                strings.TrimSpace(fixture.Status),
		SourceReport:          filepath.Base(filepath.Clean(path)),
		DryRun:                *fixture.DryRun,
		FirmwareBuildRequired: *fixture.FirmwareBuildRequired,
		BuildAllowed:          *fixture.BuildAllowed,
		FlashAllowed:          *fixture.FlashAllowed,
		Mode:                  strings.TrimSpace(fixture.Mode),
		DesiredPhrase:         strings.TrimSpace(fixture.DesiredPhrase),
		DesiredPinyin:         strings.TrimSpace(fixture.DesiredPinyin),
		Threshold:             fixture.Threshold,
	}, nil
}

func attachProductWakeWordFirmwarePlan(readiness *productWakeWordReadiness, plan productWakeWordFirmwarePlanEvidence) []productReadinessFinding {
	if !matchingProductWakeWordFirmwarePlan(*readiness, plan) {
		return []productReadinessFinding{{
			Code:    "wake_word_firmware_plan_mismatch",
			Message: "Wake word firmware plan does not match the current Gateway wake-word intent",
			Detail:  plan.SourceReport,
		}}
	}
	readiness.FirmwarePlanAvailable = true
	readiness.FirmwarePlanStatus = plan.Status
	readiness.FirmwarePlanSource = plan.SourceReport
	readiness.FirmwarePlanDryRun = plan.DryRun
	readiness.FirmwarePlanBuild = plan.BuildAllowed
	readiness.FirmwarePlanFlash = plan.FlashAllowed
	return []productReadinessFinding{{
		Code:    "wake_word_firmware_plan_available",
		Message: "Wake word firmware plan evidence is available for the current stored intent",
		Detail:  plan.SourceReport,
	}}
}

func matchingProductWakeWordFirmwarePlan(readiness productWakeWordReadiness, plan productWakeWordFirmwarePlanEvidence) bool {
	if !readiness.Available || strings.TrimSpace(readiness.Mode) != plan.Mode {
		return false
	}
	if readiness.FirmwareBuildRequired != plan.FirmwareBuildRequired {
		return false
	}
	if readiness.Threshold != plan.Threshold {
		return false
	}
	if plan.Mode == wakeWordFirmwareCustomMode {
		return strings.TrimSpace(readiness.DesiredPhrase) == plan.DesiredPhrase &&
			strings.TrimSpace(readiness.DesiredPinyin) == plan.DesiredPinyin
	}
	return plan.Mode == wakeWordFirmwareBuiltinMode
}

func validProductWakeWordFirmwarePlanFixture(fixture productWakeWordFirmwarePlanFixture) bool {
	if fixture.SchemaVersion != wakeWordFirmwarePlanSchema ||
		!*fixture.DryRun ||
		*fixture.BuildAllowed ||
		*fixture.FlashAllowed ||
		strings.TrimSpace(fixture.FirmwareID) != wakeWordFirmwareID ||
		strings.TrimSpace(fixture.TargetBoard) != wakeWordFirmwareTargetBoard ||
		strings.TrimSpace(fixture.TargetProfile) != wakeWordFirmwareTargetProfile ||
		strings.TrimSpace(fixture.GuardTier) != wakeWordFirmwareGuardTier ||
		fixture.Threshold < 1 ||
		fixture.Threshold > 100 {
		return false
	}
	status := strings.TrimSpace(fixture.Status)
	mode := strings.TrimSpace(fixture.Mode)
	switch status {
	case "pending_firmware_build":
		return mode == wakeWordFirmwareCustomMode &&
			*fixture.FirmwareBuildRequired &&
			strings.TrimSpace(fixture.DesiredPhrase) != "" &&
			strings.TrimSpace(fixture.DesiredPinyin) != "" &&
			strings.TrimSpace(fixture.RuntimeStatus) == "pending_firmware_build" &&
			strings.TrimSpace(fixture.NextRequiredConfirmation) == wakeWordFirmwareBuildConfirm
	case "builtin_noop":
		return mode == wakeWordFirmwareBuiltinMode && !*fixture.FirmwareBuildRequired
	default:
		return false
	}
}

func missingProductWakeWordFirmwarePlanField(fixture productWakeWordFirmwarePlanFixture) string {
	switch {
	case fixture.DryRun == nil:
		return "dry_run"
	case fixture.FirmwareBuildRequired == nil:
		return "firmware_build_required"
	case fixture.BuildAllowed == nil:
		return "build_allowed"
	case fixture.FlashAllowed == nil:
		return "flash_allowed"
	default:
		return ""
	}
}

func missingProductWakeWordFirmwarePlanFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "wake_word_firmware_plan_missing_field",
		Message: "Wake word firmware plan report is missing a required field",
		Detail:  field,
	}
}

func invalidProductWakeWordFirmwarePlanFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "wake_word_firmware_plan_invalid",
		Message: "Wake word firmware plan report is invalid or unsafe",
	}
}

func productWakeWordFirmwarePlanContainsForbiddenValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			if productWakeWordFirmwarePlanContainsForbiddenValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productWakeWordFirmwarePlanContainsForbiddenValue(child) {
				return true
			}
		}
	case string:
		lower := strings.ToLower(typed)
		if containsLegacyIdentity(typed) {
			return true
		}
		for _, forbidden := range []string{"http://", "https://", "/users/", "secret", "token", "proxy", "transcript", "provider output", "raw audio"} {
			if strings.Contains(lower, forbidden) {
				return true
			}
		}
	}
	return false
}

type productWakeWordFirmwarePackageEvidence struct {
	Valid          bool
	Status         string
	SourceReport   string
	PackageWritten bool
	FlashAllowed   bool
	FlashExecuted  bool
	ProductReady   bool
	Mode           string
	DesiredPhrase  string
	DesiredPinyin  string
	Threshold      int
	ArtifactName   string
	ManifestName   string
}

type productWakeWordFirmwarePackageFixture struct {
	SchemaVersion    string                     `json:"schema_version"`
	GeneratedAtMS    *int64                     `json:"generated_at_ms"`
	Status           string                     `json:"status"`
	PackageWritten   *bool                      `json:"package_written"`
	FlashAllowed     *bool                      `json:"flash_allowed"`
	FlashExecuted    *bool                      `json:"flash_executed"`
	ProductReady     *bool                      `json:"product_ready"`
	FirmwareID       string                     `json:"firmware_id"`
	TargetBoard      string                     `json:"target_board"`
	TargetProfile    string                     `json:"target_profile"`
	Mode             string                     `json:"mode"`
	DesiredPhrase    string                     `json:"desired_phrase"`
	DesiredPinyin    string                     `json:"desired_pinyin"`
	Threshold        *int                       `json:"threshold"`
	Commit           string                     `json:"commit"`
	Timestamp        string                     `json:"timestamp"`
	SourcePlanReport string                     `json:"source_plan_report"`
	BuildReceipt     string                     `json:"build_receipt"`
	BuildDirName     string                     `json:"build_dir_name"`
	ArtifactName     string                     `json:"artifact_name"`
	SHA256Name       string                     `json:"sha256_name"`
	ManifestName     string                     `json:"manifest_name"`
	SHA256           string                     `json:"sha256"`
	Parts            []xiaozhiFirmwareFlashPart `json:"parts"`
	ReportPath       string                     `json:"report_path"`
}

func loadProductWakeWordFirmwarePackageEvidence(path string) (productWakeWordFirmwarePackageEvidence, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productWakeWordFirmwarePackageEvidence{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	if productWakeWordFirmwarePackageContainsForbiddenKey(raw) || productWakeWordFirmwarePackageContainsForbiddenValue(raw) {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	var fixture productWakeWordFirmwarePackageFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	if missingField := missingProductWakeWordFirmwarePackageField(fixture); missingField != "" {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{missingProductWakeWordFirmwarePackageFieldFinding(missingField)}
	}
	if !validProductWakeWordFirmwarePackageFixture(fixture) {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	return productWakeWordFirmwarePackageEvidence{
		Valid:          true,
		Status:         strings.TrimSpace(fixture.Status),
		SourceReport:   filepath.Base(filepath.Clean(path)),
		PackageWritten: *fixture.PackageWritten,
		FlashAllowed:   *fixture.FlashAllowed,
		FlashExecuted:  *fixture.FlashExecuted,
		ProductReady:   *fixture.ProductReady,
		Mode:           strings.TrimSpace(fixture.Mode),
		DesiredPhrase:  strings.TrimSpace(fixture.DesiredPhrase),
		DesiredPinyin:  strings.TrimSpace(fixture.DesiredPinyin),
		Threshold:      *fixture.Threshold,
		ArtifactName:   strings.TrimSpace(fixture.ArtifactName),
		ManifestName:   strings.TrimSpace(fixture.ManifestName),
	}, nil
}

func attachProductWakeWordFirmwarePackage(readiness *productWakeWordReadiness, evidence productWakeWordFirmwarePackageEvidence) []productReadinessFinding {
	if !matchingProductWakeWordFirmwarePackage(*readiness, evidence) {
		return []productReadinessFinding{{
			Code:    "wake_word_firmware_package_mismatch",
			Message: "Wake word firmware package does not match the current Gateway wake-word intent",
			Detail:  evidence.SourceReport,
		}}
	}
	readiness.FirmwarePackageAvailable = true
	readiness.FirmwarePackageStatus = evidence.Status
	readiness.FirmwarePackageSource = evidence.SourceReport
	readiness.FirmwarePackageArtifact = evidence.ArtifactName
	readiness.FirmwarePackageManifest = evidence.ManifestName
	readiness.FirmwarePackageWritten = evidence.PackageWritten
	readiness.FirmwarePackageFlash = evidence.FlashAllowed
	readiness.FirmwarePackageExecuted = evidence.FlashExecuted
	return []productReadinessFinding{{
		Code:    "wake_word_firmware_package_available",
		Message: "Wake word firmware package evidence is available, but guarded flash and physical custom wake proof are still required",
		Detail:  evidence.SourceReport,
	}}
}

func matchingProductWakeWordFirmwarePackage(readiness productWakeWordReadiness, evidence productWakeWordFirmwarePackageEvidence) bool {
	return readiness.Available &&
		readiness.FirmwareBuildRequired &&
		strings.TrimSpace(readiness.Mode) == evidence.Mode &&
		strings.TrimSpace(readiness.DesiredPhrase) == evidence.DesiredPhrase &&
		strings.TrimSpace(readiness.DesiredPinyin) == evidence.DesiredPinyin &&
		readiness.Threshold == evidence.Threshold
}

func validProductWakeWordFirmwarePackageFixture(fixture productWakeWordFirmwarePackageFixture) bool {
	commit := strings.ToLower(strings.TrimSpace(fixture.Commit))
	timestamp := strings.TrimSpace(fixture.Timestamp)
	artifactName := strings.TrimSpace(fixture.ArtifactName)
	expectedArtifactName := fmt.Sprintf("%s-%s-%s-%s.bin", wakeWordFirmwareArtifactPrefix, wakeWordFirmwareTargetBoard, commit, timestamp)
	if fixture.SchemaVersion != wakeWordFirmwarePackageSchema ||
		*fixture.GeneratedAtMS <= 0 ||
		strings.TrimSpace(fixture.Status) != "packaged" ||
		!*fixture.PackageWritten ||
		*fixture.FlashAllowed ||
		*fixture.FlashExecuted ||
		*fixture.ProductReady ||
		strings.TrimSpace(fixture.FirmwareID) != wakeWordFirmwareID ||
		strings.TrimSpace(fixture.TargetBoard) != wakeWordFirmwareTargetBoard ||
		strings.TrimSpace(fixture.TargetProfile) != wakeWordFirmwareTargetProfile ||
		strings.TrimSpace(fixture.Mode) != wakeWordFirmwareCustomMode ||
		strings.TrimSpace(fixture.DesiredPhrase) == "" ||
		strings.TrimSpace(fixture.DesiredPinyin) == "" ||
		*fixture.Threshold < 1 ||
		*fixture.Threshold > 100 ||
		!productWakeWordFirmwarePackageValidHex(commit, 7, 40) ||
		!productWakeWordFirmwarePackageValidTimestamp(timestamp) ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.SourcePlanReport, ".json") ||
		strings.TrimSpace(fixture.BuildReceipt) != "a21-wake-word-build.json" ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.BuildDirName, "") ||
		artifactName != expectedArtifactName ||
		!productWakeWordFirmwarePackageBasenameOK(artifactName, ".bin") ||
		strings.TrimSpace(fixture.SHA256Name) != artifactName+".sha256" ||
		strings.TrimSpace(fixture.ManifestName) != artifactName+".manifest.json" ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.ManifestName, ".json") ||
		!productWakeWordFirmwarePackageValidHex(strings.TrimSpace(fixture.SHA256), 64, 64) ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.ReportPath, ".json") ||
		!validProductWakeWordFirmwarePackageParts(fixture.Parts) {
		return false
	}
	return true
}

func validProductWakeWordFirmwarePackageParts(parts []xiaozhiFirmwareFlashPart) bool {
	hasApp := false
	for _, part := range parts {
		if strings.TrimSpace(part.Name) == "" ||
			strings.TrimSpace(part.Offset) == "" ||
			!strings.HasPrefix(strings.TrimSpace(part.Offset), "0x") ||
			!productWakeWordFirmwarePackageBasenameOK(part.File, "") ||
			!productWakeWordFirmwarePackageValidHex(strings.TrimSpace(part.SHA256), 64, 64) ||
			part.SizeBytes <= 0 {
			return false
		}
		if strings.TrimSpace(part.Name) == "app" && strings.TrimSpace(part.File) == "xiaozhi.bin" {
			hasApp = true
		}
	}
	return hasApp
}

func missingProductWakeWordFirmwarePackageField(fixture productWakeWordFirmwarePackageFixture) string {
	switch {
	case fixture.GeneratedAtMS == nil:
		return "generated_at_ms"
	case fixture.PackageWritten == nil:
		return "package_written"
	case fixture.FlashAllowed == nil:
		return "flash_allowed"
	case fixture.FlashExecuted == nil:
		return "flash_executed"
	case fixture.ProductReady == nil:
		return "product_ready"
	case fixture.Threshold == nil:
		return "threshold"
	default:
		return ""
	}
}

func missingProductWakeWordFirmwarePackageFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "wake_word_firmware_package_missing_field",
		Message: "Wake word firmware package report is missing a required field",
		Detail:  field,
	}
}

func invalidProductWakeWordFirmwarePackageFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "wake_word_firmware_package_invalid",
		Message: "Wake word firmware package report is invalid or unsafe",
	}
}

func productWakeWordFirmwarePackageBasenameOK(value string, suffix string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "/") || strings.Contains(value, "\\") || strings.Contains(value, "..") || containsLegacyIdentity(value) {
		return false
	}
	if suffix != "" && !strings.HasSuffix(value, suffix) {
		return false
	}
	return value == filepath.Base(filepath.Clean(value))
}

func productWakeWordFirmwarePackageValidHex(value string, minLen int, maxLen int) bool {
	if len(value) < minLen || len(value) > maxLen {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func productWakeWordFirmwarePackageValidTimestamp(value string) bool {
	if len(value) != len("20260602-030000") || value[8] != '-' {
		return false
	}
	for index, char := range value {
		if index == 8 {
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func productWakeWordFirmwarePackageContainsForbiddenKey(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if forbiddenProductWakeWordFirmwarePackageKey(key) || productWakeWordFirmwarePackageContainsForbiddenKey(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productWakeWordFirmwarePackageContainsForbiddenKey(child) {
				return true
			}
		}
	}
	return false
}

func forbiddenProductWakeWordFirmwarePackageKey(key string) bool {
	normalized := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(strings.TrimSpace(key)))
	switch normalized {
	case "raw_pcm", "raw_audio", "pcm_bytes", "data_base64", "audio_base64", "base64_audio",
		"prompt", "transcript", "provider_output", "reasoning", "credential_values",
		"api_key", "access_token", "token", "full_url", "url", "proxy_url", "local_path":
		return true
	default:
		return false
	}
}

func productWakeWordFirmwarePackageContainsForbiddenValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			if productWakeWordFirmwarePackageContainsForbiddenValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productWakeWordFirmwarePackageContainsForbiddenValue(child) {
				return true
			}
		}
	case string:
		lower := strings.ToLower(typed)
		if containsLegacyIdentity(typed) {
			return true
		}
		for _, forbidden := range []string{"http://", "https://", "/users/", "secret", "token", "proxy", "transcript", "provider output", "raw audio", "base64"} {
			if strings.Contains(lower, forbidden) {
				return true
			}
		}
	}
	return false
}

type productWakeWordPhysicalAcceptanceEvidence struct {
	Valid                                bool
	Status                               string
	ProductReady                         bool
	SourceReport                         string
	Mode                                 string
	DesiredPhrase                        string
	DesiredPinyin                        string
	Threshold                            int
	PackageSourceReport                  string
	PackageArtifactName                  string
	PackageManifestName                  string
	PhysicalDeviceOnline                 bool
	FirmwareFlashExecuted                bool
	GuardedFlashReportSource             string
	OperatorCustomWakeObservationPresent bool
	WakePhraseMatched                    bool
	FalseWakeAccepted                    bool
	StockWakeAccepted                    bool
	RedactionOK                          bool
}

type productWakeWordPhysicalAcceptanceFixture struct {
	SchemaVersion                        string `json:"schema_version"`
	GeneratedAtMS                        *int64 `json:"generated_at_ms"`
	Status                               string `json:"status"`
	ProductReady                         *bool  `json:"product_ready"`
	Mode                                 string `json:"mode"`
	DesiredPhrase                        string `json:"desired_phrase"`
	DesiredPinyin                        string `json:"desired_pinyin"`
	Threshold                            *int   `json:"threshold"`
	PackageSourceReport                  string `json:"package_source_report"`
	PackageArtifactName                  string `json:"package_artifact_name"`
	PackageManifestName                  string `json:"package_manifest_name"`
	PhysicalDeviceOnline                 *bool  `json:"physical_device_online"`
	FirmwareFlashExecuted                *bool  `json:"firmware_flash_executed"`
	GuardedFlashReportSource             string `json:"guarded_flash_report_source"`
	OperatorCustomWakeObservationPresent *bool  `json:"operator_custom_wake_observation_present"`
	WakePhraseMatched                    *bool  `json:"wake_phrase_matched"`
	FalseWakeAccepted                    *bool  `json:"false_wake_accepted"`
	StockWakeAccepted                    *bool  `json:"stock_wake_accepted"`
	RedactionOK                          *bool  `json:"redaction_ok"`
	ReportPath                           string `json:"report_path"`
}
