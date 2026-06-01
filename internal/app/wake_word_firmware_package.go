package app

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
	"time"
)

const (
	wakeWordFirmwarePackageSchema      = "a21.wake_word_firmware_package.v1"
	wakeWordFirmwareBuildReceiptSchema = "a21.wake_word_firmware_build.v1"
	wakeWordFirmwareArtifactPrefix     = "a21-wake-word-xiaozhi-esp-sr-multinet"
)

type wakeWordFirmwarePackageOptions struct {
	PlanPath  string
	BuildDir  string
	OutputDir string
	Commit    string
	Timestamp string
}

type wakeWordFirmwareBuildReceipt struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	FirmwareID    string `json:"firmware_id"`
	TargetBoard   string `json:"target_board"`
	TargetProfile string `json:"target_profile"`
	Mode          string `json:"mode"`
	DesiredPhrase string `json:"desired_phrase"`
	DesiredPinyin string `json:"desired_pinyin"`
	Threshold     int    `json:"threshold"`
	AppBinary     string `json:"app_binary"`
}

type wakeWordFirmwarePackageReport struct {
	SchemaVersion       string                           `json:"schema_version"`
	GeneratedAtMS       int64                            `json:"generated_at_ms"`
	Status              string                           `json:"status"`
	PackageWritten      bool                             `json:"package_written"`
	FlashAllowed        bool                             `json:"flash_allowed"`
	FlashExecuted       bool                             `json:"flash_executed"`
	ProductReady        bool                             `json:"product_ready"`
	FirmwareID          string                           `json:"firmware_id"`
	TargetBoard         string                           `json:"target_board"`
	TargetProfile       string                           `json:"target_profile"`
	Mode                string                           `json:"mode"`
	DesiredPhrase       string                           `json:"desired_phrase"`
	DesiredPinyin       string                           `json:"desired_pinyin"`
	Threshold           int                              `json:"threshold"`
	Commit              string                           `json:"commit"`
	Timestamp           string                           `json:"timestamp"`
	SourcePlanReport    string                           `json:"source_plan_report"`
	BuildReceipt        string                           `json:"build_receipt"`
	BuildDirName        string                           `json:"build_dir_name"`
	ArtifactName        string                           `json:"artifact_name"`
	SHA256Name          string                           `json:"sha256_name"`
	ManifestName        string                           `json:"manifest_name"`
	SHA256              string                           `json:"sha256"`
	OTA                 xiaozhiFirmwareOTAConfig         `json:"ota"`
	Parts               []xiaozhiFirmwareFlashPart       `json:"parts"`
	NextRequiredActions []string                         `json:"next_required_actions,omitempty"`
	Findings            []wakeWordFirmwarePackageFinding `json:"findings,omitempty"`
	ReportPath          string                           `json:"report_path,omitempty"`
}

type wakeWordFirmwarePackageFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func runWakeWordFirmwarePackage(args []string, stdout io.Writer, stderr io.Writer) int {
	options := wakeWordFirmwarePackageOptions{
		OutputDir: filepath.Join("firmware", "artifacts", "wake-word"),
		Timestamp: time.Now().Format("20060102-150405"),
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 wake-word-firmware-package --plan reports/a21-wake-word-firmware-plan-*.json --build-dir /path/to/xiaozhi/build-m5stack-core-s3 --commit <git-sha> [--output-dir firmware/artifacts/wake-word] [--timestamp YYYYMMDD-HHMMSS]")
			return 0
		case "--plan":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--plan requires a value")
				return 2
			}
			i++
			options.PlanPath = args[i]
		case "--build-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--build-dir requires a value")
				return 2
			}
			i++
			options.BuildDir = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
		case "--timestamp":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--timestamp requires a value")
				return 2
			}
			i++
			options.Timestamp = args[i]
		default:
			fmt.Fprintf(stderr, "unknown wake-word-firmware-package option %q\n", args[i])
			return 2
		}
	}
	report, err := packageWakeWordFirmware(options)
	if err != nil {
		fmt.Fprintln(stderr, "wake word firmware package failed: invalid package inputs")
		return 1
	}
	if err := writeJSONWakeWordFirmwarePackage(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode wake word firmware package report: %v\n", err)
		return 1
	}
	return 0
}

func packageWakeWordFirmware(options wakeWordFirmwarePackageOptions) (wakeWordFirmwarePackageReport, error) {
	if err := validateWakeWordFirmwarePackageOptions(options); err != nil {
		return wakeWordFirmwarePackageReport{}, err
	}
	plan, findings := loadProductWakeWordFirmwarePlanEvidence(options.PlanPath)
	if len(findings) != 0 || !plan.Valid || plan.Mode != wakeWordFirmwareCustomMode || !plan.FirmwareBuildRequired {
		return wakeWordFirmwarePackageReport{}, fmt.Errorf("wake word firmware plan is invalid")
	}
	buildDir := filepath.Clean(options.BuildDir)
	receipt, err := loadWakeWordFirmwareBuildReceipt(filepath.Join(buildDir, "a21-wake-word-build.json"))
	if err != nil {
		return wakeWordFirmwarePackageReport{}, err
	}
	if !matchingWakeWordFirmwareBuildReceipt(plan, receipt) {
		return wakeWordFirmwarePackageReport{}, fmt.Errorf("wake word build receipt does not match plan")
	}
	ota, err := inspectXiaozhiFirmwareOTAConfig(buildDir)
	if err != nil {
		return wakeWordFirmwarePackageReport{}, err
	}
	parts, err := collectXiaozhiFirmwareFlashParts(buildDir)
	if err != nil {
		return wakeWordFirmwarePackageReport{}, err
	}
	appPart, ok := findWakeWordAppPart(parts)
	if !ok || appPart.File != receipt.AppBinary {
		return wakeWordFirmwarePackageReport{}, fmt.Errorf("wake word app binary is missing")
	}
	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return wakeWordFirmwarePackageReport{}, err
	}
	artifactName := fmt.Sprintf("%s-%s-%s-%s.bin", wakeWordFirmwareArtifactPrefix, wakeWordFirmwareTargetBoard, options.Commit, options.Timestamp)
	artifactPath := filepath.Join(options.OutputDir, artifactName)
	checksum, err := copyFileWithSHA256(filepath.Join(buildDir, appPart.File), artifactPath)
	if err != nil {
		return wakeWordFirmwarePackageReport{}, err
	}
	shaName := artifactName + ".sha256"
	if err := os.WriteFile(filepath.Join(options.OutputDir, shaName), []byte(checksum+"  "+artifactName+"\n"), 0o644); err != nil {
		return wakeWordFirmwarePackageReport{}, err
	}
	report := wakeWordFirmwarePackageReport{
		SchemaVersion:    wakeWordFirmwarePackageSchema,
		GeneratedAtMS:    time.Now().UnixMilli(),
		Status:           "packaged",
		PackageWritten:   true,
		FlashAllowed:     false,
		FlashExecuted:    false,
		ProductReady:     false,
		FirmwareID:       wakeWordFirmwareID,
		TargetBoard:      wakeWordFirmwareTargetBoard,
		TargetProfile:    wakeWordFirmwareTargetProfile,
		Mode:             wakeWordFirmwareCustomMode,
		DesiredPhrase:    plan.DesiredPhrase,
		DesiredPinyin:    plan.DesiredPinyin,
		Threshold:        plan.Threshold,
		Commit:           strings.ToLower(options.Commit),
		Timestamp:        options.Timestamp,
		SourcePlanReport: filepath.Base(filepath.Clean(options.PlanPath)),
		BuildReceipt:     "a21-wake-word-build.json",
		BuildDirName:     filepath.Base(buildDir),
		ArtifactName:     artifactName,
		SHA256Name:       shaName,
		ManifestName:     artifactName + ".manifest.json",
		SHA256:           checksum,
		OTA:              ota,
		Parts:            parts,
		NextRequiredActions: []string{
			"Run a guarded wake-word firmware flash plan against this package in a foreground hardware window.",
			"Prove the custom wake model is active on the physical StackChan before product readiness can pass.",
		},
		Findings: []wakeWordFirmwarePackageFinding{{
			Code:     "wake_word_firmware_package_below_activation",
			Severity: "info",
			Message:  "Wake word firmware package exists, but flash and physical custom wake evidence are still required.",
		}},
	}
	if err := writeWakeWordFirmwarePackageManifest(filepath.Join(options.OutputDir, report.ManifestName), report); err != nil {
		return wakeWordFirmwarePackageReport{}, err
	}
	reportPath := filepath.Join(options.OutputDir, fmt.Sprintf("a21-wake-word-firmware-package-%s-%d.json", time.Now().Format("20060102-150405"), time.Now().UnixNano()))
	report.ReportPath = filepath.Base(reportPath)
	if err := writeWakeWordFirmwarePackageReport(reportPath, report); err != nil {
		return wakeWordFirmwarePackageReport{}, err
	}
	return report, nil
}

func validateWakeWordFirmwarePackageOptions(options wakeWordFirmwarePackageOptions) error {
	if strings.TrimSpace(options.PlanPath) == "" {
		return fmt.Errorf("plan is required")
	}
	if strings.TrimSpace(options.BuildDir) == "" {
		return fmt.Errorf("build dir is required")
	}
	if strings.TrimSpace(options.OutputDir) == "" {
		return fmt.Errorf("output dir is required")
	}
	for _, path := range []string{options.PlanPath, options.BuildDir, options.OutputDir} {
		if err := validateA21InputPath(path); err != nil {
			return err
		}
	}
	if ok, _ := regexp.MatchString(`^[0-9a-fA-F]{7,40}$`, strings.TrimSpace(options.Commit)); !ok {
		return fmt.Errorf("commit must be a 7-40 character git sha")
	}
	if ok, _ := regexp.MatchString(`^\d{8}-\d{6}$`, strings.TrimSpace(options.Timestamp)); !ok {
		return fmt.Errorf("timestamp must be YYYYMMDD-HHMMSS")
	}
	return nil
}

func loadWakeWordFirmwareBuildReceipt(path string) (wakeWordFirmwareBuildReceipt, error) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 8192 {
		return wakeWordFirmwareBuildReceipt{}, fmt.Errorf("wake word build receipt is required")
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return wakeWordFirmwareBuildReceipt{}, fmt.Errorf("wake word build receipt invalid")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return wakeWordFirmwareBuildReceipt{}, fmt.Errorf("wake word build receipt invalid")
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productWakeWordFirmwarePlanContainsForbiddenValue(raw) {
		return wakeWordFirmwareBuildReceipt{}, fmt.Errorf("wake word build receipt unsafe")
	}
	var receipt wakeWordFirmwareBuildReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return wakeWordFirmwareBuildReceipt{}, fmt.Errorf("wake word build receipt invalid")
	}
	if !validWakeWordFirmwareBuildReceipt(receipt) {
		return wakeWordFirmwareBuildReceipt{}, fmt.Errorf("wake word build receipt invalid")
	}
	return receipt, nil
}

func validWakeWordFirmwareBuildReceipt(receipt wakeWordFirmwareBuildReceipt) bool {
	return receipt.SchemaVersion == wakeWordFirmwareBuildReceiptSchema &&
		strings.TrimSpace(receipt.Status) == "built" &&
		strings.TrimSpace(receipt.FirmwareID) == wakeWordFirmwareID &&
		strings.TrimSpace(receipt.TargetBoard) == wakeWordFirmwareTargetBoard &&
		strings.TrimSpace(receipt.TargetProfile) == wakeWordFirmwareTargetProfile &&
		strings.TrimSpace(receipt.Mode) == wakeWordFirmwareCustomMode &&
		strings.TrimSpace(receipt.DesiredPhrase) != "" &&
		strings.TrimSpace(receipt.DesiredPinyin) != "" &&
		receipt.Threshold >= 1 &&
		receipt.Threshold <= 100 &&
		strings.TrimSpace(receipt.AppBinary) == "xiaozhi.bin"
}

func matchingWakeWordFirmwareBuildReceipt(plan productWakeWordFirmwarePlanEvidence, receipt wakeWordFirmwareBuildReceipt) bool {
	return strings.TrimSpace(receipt.Mode) == plan.Mode &&
		strings.TrimSpace(receipt.DesiredPhrase) == plan.DesiredPhrase &&
		strings.TrimSpace(receipt.DesiredPinyin) == plan.DesiredPinyin &&
		receipt.Threshold == plan.Threshold
}

func findWakeWordAppPart(parts []xiaozhiFirmwareFlashPart) (xiaozhiFirmwareFlashPart, bool) {
	for _, part := range parts {
		if part.Name == "app" {
			return part, true
		}
	}
	return xiaozhiFirmwareFlashPart{}, false
}

func copyFileWithSHA256(src, dst string) (string, error) {
	input, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer input.Close()
	output, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer output.Close()
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(output, hash), input); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeWakeWordFirmwarePackageManifest(path string, report wakeWordFirmwarePackageReport) error {
	return writeWakeWordFirmwarePackageJSONFile(path, report)
}

func writeWakeWordFirmwarePackageReport(path string, report wakeWordFirmwarePackageReport) error {
	return writeWakeWordFirmwarePackageJSONFile(path, report)
}

func writeWakeWordFirmwarePackageJSONFile(path string, report wakeWordFirmwarePackageReport) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeJSONWakeWordFirmwarePackage(file, report)
}

func writeJSONWakeWordFirmwarePackage(writer io.Writer, report wakeWordFirmwarePackageReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
