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
	wakeWordFirmwareBuildReviewSchema  = "a21.wake_word_firmware_build_review.v1"
	wakeWordFirmwareArtifactPrefix     = "a21-wake-word-xiaozhi-esp-sr-multinet"
)

type wakeWordFirmwarePackageOptions struct {
	PlanPath         string
	BuildDir         string
	BuildReceiptPath string
	OutputDir        string
	Commit           string
	Timestamp        string
}

type wakeWordFirmwareBuildReceiptOptions struct {
	PlanPath         string
	BuildDir         string
	ReviewReportPath string
	OutputDir        string
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
	BuildDirName  string `json:"build_dir_name,omitempty"`
	SDKConfig     string `json:"sdkconfig,omitempty"`
	FlasherArgs   string `json:"flasher_args,omitempty"`
	ReviewReport  string `json:"review_report,omitempty"`
}

type wakeWordFirmwareBuildReview struct {
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
	SDKConfig     string `json:"sdkconfig"`
	FlasherArgs   string `json:"flasher_args"`
	Reviewer      string `json:"reviewer,omitempty"`
	BuildTool     string `json:"build_tool,omitempty"`
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
	BuildReview         string                           `json:"build_review,omitempty"`
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

type wakeWordFirmwarePackageInputError struct {
	Code    string
	Message string
}

func (err wakeWordFirmwarePackageInputError) Error() string {
	return err.Message
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
		case "--build-receipt":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--build-receipt requires a value")
				return 2
			}
			i++
			options.BuildReceiptPath = args[i]
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
		report = rejectedWakeWordFirmwarePackageReport(options, err)
		if writeErr := writeWakeWordFirmwarePackageRejectionReport(options.OutputDir, &report); writeErr != nil {
			fmt.Fprintln(stderr, "wake word firmware package rejected: diagnostic report unavailable")
			return 1
		}
		if encodeErr := writeJSONWakeWordFirmwarePackage(stdout, report); encodeErr != nil {
			fmt.Fprintf(stderr, "encode wake word firmware package rejection: %v\n", encodeErr)
			return 1
		}
		fmt.Fprintln(stderr, "wake word firmware package rejected: see structured report")
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
	receiptPath := wakeWordFirmwareBuildReceiptPath(options, buildDir)
	receipt, err := loadWakeWordFirmwareBuildReceipt(receiptPath)
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
		BuildReceipt:     filepath.Base(filepath.Clean(receiptPath)),
		BuildReview:      receipt.ReviewReport,
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

func rejectedWakeWordFirmwarePackageReport(options wakeWordFirmwarePackageOptions, err error) wakeWordFirmwarePackageReport {
	code := "wake_word_firmware_package_invalid"
	message := "Wake word firmware package inputs are invalid."
	status := "rejected"
	var inputErr wakeWordFirmwarePackageInputError
	if ok := errorAsWakeWordFirmwarePackageInput(err, &inputErr); ok {
		code = inputErr.Code
		message = inputErr.Message
	}
	switch code {
	case "wake_word_firmware_build_dir_missing":
		status = "missing_build_dir"
	case "wake_word_firmware_build_receipt_missing":
		status = "missing_build_receipt"
	}
	plan, _ := loadProductWakeWordFirmwarePlanEvidence(options.PlanPath)
	report := wakeWordFirmwarePackageReport{
		SchemaVersion:    wakeWordFirmwarePackageSchema,
		GeneratedAtMS:    time.Now().UnixMilli(),
		Status:           status,
		PackageWritten:   false,
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
		SourcePlanReport: safeWakeWordFirmwarePackageBase(options.PlanPath),
		BuildReceipt:     wakeWordFirmwareBuildReceiptName(options),
		BuildDirName:     safeWakeWordFirmwarePackageBase(options.BuildDir),
		Commit:           safeWakeWordFirmwarePackageCommit(options.Commit),
		Timestamp:        safeWakeWordFirmwarePackageTimestamp(options.Timestamp),
		NextRequiredActions: []string{
			"Prepare a reviewed A21 xiaozhi/ESP-SR MultiNet build directory containing a matching a21-wake-word-build.json receipt.",
			"Re-run wake-word-firmware-package before any guarded flash plan or physical custom wake acceptance.",
		},
		Findings: []wakeWordFirmwarePackageFinding{{
			Code:     code,
			Severity: "error",
			Message:  message,
		}},
	}
	return report
}

func writeWakeWordFirmwarePackageRejectionReport(outputDir string, report *wakeWordFirmwarePackageReport) error {
	if strings.TrimSpace(outputDir) == "" {
		return fmt.Errorf("output dir is required")
	}
	if err := validateA21InputPath(outputDir); err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-wake-word-firmware-package-%s-%d.json", time.Now().Format("20060102-150405"), time.Now().UnixNano()))
	report.ReportPath = filepath.Base(reportPath)
	return writeWakeWordFirmwarePackageReport(reportPath, *report)
}

func errorAsWakeWordFirmwarePackageInput(err error, target *wakeWordFirmwarePackageInputError) bool {
	if err == nil {
		return false
	}
	if typed, ok := err.(wakeWordFirmwarePackageInputError); ok {
		*target = typed
		return true
	}
	return false
}

func safeWakeWordFirmwarePackageBase(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Base(filepath.Clean(path))
}

func safeWakeWordFirmwarePackageCommit(commit string) string {
	commit = strings.ToLower(strings.TrimSpace(commit))
	if ok, _ := regexp.MatchString(`^[0-9a-f]{7,40}$`, commit); ok {
		return commit
	}
	return ""
}

func safeWakeWordFirmwarePackageTimestamp(timestamp string) string {
	timestamp = strings.TrimSpace(timestamp)
	if ok, _ := regexp.MatchString(`^\d{8}-\d{6}$`, timestamp); ok {
		return timestamp
	}
	return ""
}

func validateWakeWordFirmwarePackageOptions(options wakeWordFirmwarePackageOptions) error {
	if strings.TrimSpace(options.PlanPath) == "" {
		return wakeWordFirmwarePackageInputError{
			Code:    "wake_word_firmware_plan_required",
			Message: "Wake word firmware package requires a matching firmware plan report.",
		}
	}
	if strings.TrimSpace(options.BuildDir) == "" {
		return wakeWordFirmwarePackageInputError{
			Code:    "wake_word_firmware_build_dir_missing",
			Message: "Wake word firmware package requires the reviewed xiaozhi/ESP-SR build directory.",
		}
	}
	if strings.TrimSpace(options.OutputDir) == "" {
		return wakeWordFirmwarePackageInputError{
			Code:    "wake_word_firmware_output_dir_required",
			Message: "Wake word firmware package requires an A21 output directory.",
		}
	}
	for _, path := range []string{options.PlanPath, options.BuildDir, options.BuildReceiptPath, options.OutputDir} {
		if strings.TrimSpace(path) == "" {
			continue
		}
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

func wakeWordFirmwareBuildReceiptPath(options wakeWordFirmwarePackageOptions, buildDir string) string {
	if strings.TrimSpace(options.BuildReceiptPath) != "" {
		return filepath.Clean(options.BuildReceiptPath)
	}
	return filepath.Join(buildDir, "a21-wake-word-build.json")
}

func wakeWordFirmwareBuildReceiptName(options wakeWordFirmwarePackageOptions) string {
	if strings.TrimSpace(options.BuildReceiptPath) != "" {
		return safeWakeWordFirmwarePackageBase(options.BuildReceiptPath)
	}
	return "a21-wake-word-build.json"
}

func runWakeWordFirmwareBuildReceipt(args []string, stdout io.Writer, stderr io.Writer) int {
	var options wakeWordFirmwareBuildReceiptOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 wake-word-firmware-build-receipt --plan reports/a21-wake-word-firmware-plan-*.json --build-dir /path/to/xiaozhi/build-m5stack-core-s3 --review-report <review.json> [--output-dir receipts]")
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
		case "--review-report", "--build-review":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--review-report requires a value")
				return 2
			}
			i++
			options.ReviewReportPath = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown wake-word-firmware-build-receipt option %q\n", args[i])
			return 2
		}
	}
	receipt, err := buildWakeWordFirmwareBuildReceipt(options)
	if err != nil {
		fmt.Fprintln(stderr, "wake word firmware build receipt failed")
		return 1
	}
	if strings.TrimSpace(options.OutputDir) != "" {
		if err := writeWakeWordFirmwareBuildReceipt(options.OutputDir, receipt); err != nil {
			fmt.Fprintln(stderr, "write wake word firmware build receipt failed")
			return 1
		}
	}
	if err := writeJSONWakeWordFirmwareBuildReceipt(stdout, receipt); err != nil {
		fmt.Fprintf(stderr, "encode wake word firmware build receipt: %v\n", err)
		return 1
	}
	return 0
}

func buildWakeWordFirmwareBuildReceipt(options wakeWordFirmwareBuildReceiptOptions) (wakeWordFirmwareBuildReceipt, error) {
	if err := validateWakeWordFirmwareBuildReceiptOptions(options); err != nil {
		return wakeWordFirmwareBuildReceipt{}, err
	}
	plan, findings := loadProductWakeWordFirmwarePlanEvidence(options.PlanPath)
	if len(findings) != 0 || !plan.Valid || plan.Mode != wakeWordFirmwareCustomMode || !plan.FirmwareBuildRequired {
		return wakeWordFirmwareBuildReceipt{}, fmt.Errorf("wake word firmware plan is invalid")
	}
	buildDir := filepath.Clean(options.BuildDir)
	sdkconfig, err := wakeWordFirmwareBuildSDKConfigName(buildDir)
	if err != nil {
		return wakeWordFirmwareBuildReceipt{}, err
	}
	if _, err := os.Stat(filepath.Join(buildDir, "flasher_args.json")); err != nil {
		return wakeWordFirmwareBuildReceipt{}, fmt.Errorf("flasher_args.json is required")
	}
	if _, err := os.Stat(filepath.Join(buildDir, "xiaozhi.bin")); err != nil {
		return wakeWordFirmwareBuildReceipt{}, fmt.Errorf("xiaozhi.bin is required")
	}
	review, err := loadWakeWordFirmwareBuildReview(options.ReviewReportPath)
	if err != nil {
		return wakeWordFirmwareBuildReceipt{}, err
	}
	if !matchingWakeWordFirmwareBuildReview(plan, review) ||
		review.SDKConfig != filepath.Base(sdkconfig) ||
		review.FlasherArgs != "flasher_args.json" ||
		review.AppBinary != "xiaozhi.bin" {
		return wakeWordFirmwareBuildReceipt{}, fmt.Errorf("wake word build review does not match plan and build")
	}
	return wakeWordFirmwareBuildReceipt{
		SchemaVersion: wakeWordFirmwareBuildReceiptSchema,
		Status:        "built",
		FirmwareID:    wakeWordFirmwareID,
		TargetBoard:   wakeWordFirmwareTargetBoard,
		TargetProfile: wakeWordFirmwareTargetProfile,
		Mode:          wakeWordFirmwareCustomMode,
		DesiredPhrase: plan.DesiredPhrase,
		DesiredPinyin: plan.DesiredPinyin,
		Threshold:     plan.Threshold,
		AppBinary:     "xiaozhi.bin",
		BuildDirName:  filepath.Base(buildDir),
		SDKConfig:     filepath.Base(sdkconfig),
		FlasherArgs:   "flasher_args.json",
		ReviewReport:  filepath.Base(filepath.Clean(options.ReviewReportPath)),
	}, nil
}

func validateWakeWordFirmwareBuildReceiptOptions(options wakeWordFirmwareBuildReceiptOptions) error {
	if strings.TrimSpace(options.PlanPath) == "" {
		return fmt.Errorf("--plan is required")
	}
	if strings.TrimSpace(options.BuildDir) == "" {
		return fmt.Errorf("--build-dir is required")
	}
	if strings.TrimSpace(options.ReviewReportPath) == "" {
		return fmt.Errorf("--review-report is required")
	}
	for _, path := range []string{options.PlanPath, options.BuildDir, options.ReviewReportPath, options.OutputDir} {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if err := validateA21InputPath(path); err != nil {
			return err
		}
	}
	return nil
}

func wakeWordFirmwareBuildSDKConfigName(buildDir string) (string, error) {
	for _, candidate := range []string{
		filepath.Join(buildDir, "config", "sdkconfig.json"),
		filepath.Join(buildDir, "sdkconfig.json"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("sdkconfig.json is required")
}

func loadWakeWordFirmwareBuildReview(path string) (wakeWordFirmwareBuildReview, error) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 8192 {
		return wakeWordFirmwareBuildReview{}, fmt.Errorf("wake word build review is required")
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return wakeWordFirmwareBuildReview{}, fmt.Errorf("wake word build review invalid")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return wakeWordFirmwareBuildReview{}, fmt.Errorf("wake word build review invalid")
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productWakeWordFirmwarePlanContainsForbiddenValue(raw) {
		return wakeWordFirmwareBuildReview{}, fmt.Errorf("wake word build review unsafe")
	}
	var review wakeWordFirmwareBuildReview
	if err := json.Unmarshal(data, &review); err != nil {
		return wakeWordFirmwareBuildReview{}, fmt.Errorf("wake word build review invalid")
	}
	if !validWakeWordFirmwareBuildReview(review) {
		return wakeWordFirmwareBuildReview{}, fmt.Errorf("wake word build review invalid")
	}
	return review, nil
}

func validWakeWordFirmwareBuildReview(review wakeWordFirmwareBuildReview) bool {
	return review.SchemaVersion == wakeWordFirmwareBuildReviewSchema &&
		strings.TrimSpace(review.Status) == "reviewed" &&
		strings.TrimSpace(review.FirmwareID) == wakeWordFirmwareID &&
		strings.TrimSpace(review.TargetBoard) == wakeWordFirmwareTargetBoard &&
		strings.TrimSpace(review.TargetProfile) == wakeWordFirmwareTargetProfile &&
		strings.TrimSpace(review.Mode) == wakeWordFirmwareCustomMode &&
		strings.TrimSpace(review.DesiredPhrase) != "" &&
		strings.TrimSpace(review.DesiredPinyin) != "" &&
		review.Threshold >= 1 &&
		review.Threshold <= 100 &&
		validWakeWordFirmwareBuildBasename(review.AppBinary) &&
		validWakeWordFirmwareBuildBasename(review.SDKConfig) &&
		validWakeWordFirmwareBuildBasename(review.FlasherArgs) &&
		validWakeWordFirmwareReviewLabel(review.Reviewer) &&
		validWakeWordFirmwareReviewLabel(review.BuildTool)
}

func matchingWakeWordFirmwareBuildReview(plan productWakeWordFirmwarePlanEvidence, review wakeWordFirmwareBuildReview) bool {
	return strings.TrimSpace(review.Mode) == plan.Mode &&
		strings.TrimSpace(review.DesiredPhrase) == plan.DesiredPhrase &&
		strings.TrimSpace(review.DesiredPinyin) == plan.DesiredPinyin &&
		review.Threshold == plan.Threshold
}

func validWakeWordFirmwareBuildBasename(name string) bool {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || trimmed != name || strings.Contains(trimmed, "..") || strings.ContainsAny(trimmed, `/\`) {
		return false
	}
	return filepath.Base(filepath.Clean(trimmed)) == trimmed
}

func validWakeWordFirmwareReviewLabel(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	return trimmed == value && !strings.Contains(trimmed, "..") && !strings.ContainsAny(trimmed, `/\`)
}

func writeWakeWordFirmwareBuildReceipt(outputDir string, receipt wakeWordFirmwareBuildReceipt) error {
	if err := validateA21InputPath(outputDir); err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(outputDir, "a21-wake-word-build.json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeJSONWakeWordFirmwareBuildReceipt(file, receipt)
}

func writeJSONWakeWordFirmwareBuildReceipt(writer io.Writer, receipt wakeWordFirmwareBuildReceipt) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(receipt)
}

func loadWakeWordFirmwareBuildReceipt(path string) (wakeWordFirmwareBuildReceipt, error) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 8192 {
		return wakeWordFirmwareBuildReceipt{}, wakeWordFirmwarePackageInputError{
			Code:    "wake_word_firmware_build_receipt_missing",
			Message: "Wake word firmware package requires a matching a21-wake-word-build.json build receipt.",
		}
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
		strings.TrimSpace(receipt.AppBinary) == "xiaozhi.bin" &&
		validWakeWordFirmwareBuildBasename(receipt.ReviewReport)
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
