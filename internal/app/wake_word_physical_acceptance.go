package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	wakeWordPhysicalAcceptanceSchema = "a21.wake_word_physical_acceptance.v1"
	wakeWordPhysicalProofSchema      = "a21.wake_word_physical_proof.v1"
)

type wakeWordPhysicalAcceptanceOptions struct {
	PackageReport string
	ProofReport   string
	OutputDir     string
}

type wakeWordPhysicalProofOptions struct {
	PhysicalDeviceOnline  bool
	FirmwareFlashExecuted bool
	GuardedFlashReport    string
	OperatorObserved      bool
	WakePhraseMatched     bool
	FalseWakeRejected     bool
	StockWakeRejected     bool
	OutputDir             string
}

type wakeWordPhysicalProofReport struct {
	SchemaVersion                        string `json:"schema_version"`
	GeneratedAtMS                        int64  `json:"generated_at_ms"`
	Status                               string `json:"status"`
	PhysicalDeviceOnline                 bool   `json:"physical_device_online"`
	FirmwareFlashExecuted                bool   `json:"firmware_flash_executed"`
	GuardedFlashReportSource             string `json:"guarded_flash_report_source"`
	OperatorCustomWakeObservationPresent bool   `json:"operator_custom_wake_observation_present"`
	WakePhraseMatched                    bool   `json:"wake_phrase_matched"`
	FalseWakeAccepted                    bool   `json:"false_wake_accepted"`
	StockWakeAccepted                    bool   `json:"stock_wake_accepted"`
	RedactionOK                          bool   `json:"redaction_ok"`
	ReportPath                           string `json:"report_path,omitempty"`
}

type wakeWordPhysicalAcceptanceReport struct {
	SchemaVersion                        string `json:"schema_version"`
	GeneratedAtMS                        int64  `json:"generated_at_ms"`
	Status                               string `json:"status"`
	ProductReady                         bool   `json:"product_ready"`
	Mode                                 string `json:"mode"`
	DesiredPhrase                        string `json:"desired_phrase"`
	DesiredPinyin                        string `json:"desired_pinyin"`
	Threshold                            int    `json:"threshold"`
	PackageSourceReport                  string `json:"package_source_report"`
	PackageArtifactName                  string `json:"package_artifact_name"`
	PackageManifestName                  string `json:"package_manifest_name"`
	PhysicalDeviceOnline                 bool   `json:"physical_device_online"`
	FirmwareFlashExecuted                bool   `json:"firmware_flash_executed"`
	GuardedFlashReportSource             string `json:"guarded_flash_report_source"`
	OperatorCustomWakeObservationPresent bool   `json:"operator_custom_wake_observation_present"`
	WakePhraseMatched                    bool   `json:"wake_phrase_matched"`
	FalseWakeAccepted                    bool   `json:"false_wake_accepted"`
	StockWakeAccepted                    bool   `json:"stock_wake_accepted"`
	RedactionOK                          bool   `json:"redaction_ok"`
	ReportPath                           string `json:"report_path,omitempty"`
}

type wakeWordPhysicalProofEvidence struct {
	PhysicalDeviceOnline                 bool
	FirmwareFlashExecuted                bool
	GuardedFlashReportSource             string
	OperatorCustomWakeObservationPresent bool
	WakePhraseMatched                    bool
	FalseWakeAccepted                    bool
	StockWakeAccepted                    bool
	RedactionOK                          bool
}

type wakeWordPhysicalProofFixture struct {
	SchemaVersion                        string `json:"schema_version"`
	GeneratedAtMS                        *int64 `json:"generated_at_ms"`
	Status                               string `json:"status"`
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

func runWakeWordPhysicalProof(args []string, stdout io.Writer, stderr io.Writer) int {
	options := wakeWordPhysicalProofOptions{OutputDir: "reports"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 wake-word-physical-proof --physical-device-online --firmware-flash-executed --guarded-flash-report a21-wake-word-guarded-flash-*.json --operator-observed --wake-phrase-matched --false-wake-rejected --stock-wake-rejected [--output-dir reports]")
			return 0
		case "--physical-device-online":
			options.PhysicalDeviceOnline = true
		case "--firmware-flash-executed":
			options.FirmwareFlashExecuted = true
		case "--guarded-flash-report":
			if !readStringOption(args, &i, stderr, "--guarded-flash-report", &options.GuardedFlashReport) {
				return 2
			}
		case "--operator-observed":
			options.OperatorObserved = true
		case "--wake-phrase-matched":
			options.WakePhraseMatched = true
		case "--false-wake-rejected":
			options.FalseWakeRejected = true
		case "--stock-wake-rejected":
			options.StockWakeRejected = true
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown wake-word-physical-proof option %q\n", args[i])
			return 2
		}
	}
	report, err := buildWakeWordPhysicalProofReport(options)
	if err != nil {
		fmt.Fprintln(stderr, "wake word physical proof rejected: see structured input contract")
		return 1
	}
	if err := writeJSONWakeWordPhysicalProof(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode wake word physical proof report: %v\n", err)
		return 1
	}
	return 0
}

func buildWakeWordPhysicalProofReport(options wakeWordPhysicalProofOptions) (wakeWordPhysicalProofReport, error) {
	if strings.TrimSpace(options.OutputDir) == "" {
		return wakeWordPhysicalProofReport{}, fmt.Errorf("output dir is required")
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		return wakeWordPhysicalProofReport{}, err
	}
	guardedFlashReport := strings.TrimSpace(options.GuardedFlashReport)
	if !options.PhysicalDeviceOnline ||
		!options.FirmwareFlashExecuted ||
		!options.OperatorObserved ||
		!options.WakePhraseMatched ||
		!options.FalseWakeRejected ||
		!options.StockWakeRejected ||
		!productWakeWordFirmwarePackageBasenameOK(guardedFlashReport, ".json") {
		return wakeWordPhysicalProofReport{}, fmt.Errorf("wake word physical proof is incomplete")
	}
	report := wakeWordPhysicalProofReport{
		SchemaVersion:                        wakeWordPhysicalProofSchema,
		GeneratedAtMS:                        time.Now().UnixMilli(),
		Status:                               "observed",
		PhysicalDeviceOnline:                 true,
		FirmwareFlashExecuted:                true,
		GuardedFlashReportSource:             guardedFlashReport,
		OperatorCustomWakeObservationPresent: true,
		WakePhraseMatched:                    true,
		FalseWakeAccepted:                    false,
		StockWakeAccepted:                    false,
		RedactionOK:                          true,
	}
	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return wakeWordPhysicalProofReport{}, err
	}
	reportPath := filepath.Join(options.OutputDir, fmt.Sprintf("a21-wake-word-physical-proof-%s-%d.json", time.Now().Format("20060102-150405"), time.Now().UnixNano()))
	report.ReportPath = filepath.Base(reportPath)
	if err := writeWakeWordPhysicalProofReport(reportPath, report); err != nil {
		return wakeWordPhysicalProofReport{}, err
	}
	return report, nil
}

func runWakeWordPhysicalAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := wakeWordPhysicalAcceptanceOptions{OutputDir: "reports"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 wake-word-physical-acceptance --package-report reports/a21-wake-word-firmware-package-*.json --proof-report reports/a21-wake-word-physical-proof-*.json [--output-dir reports]")
			return 0
		case "--package-report":
			if !readStringOption(args, &i, stderr, "--package-report", &options.PackageReport) {
				return 2
			}
		case "--proof-report":
			if !readStringOption(args, &i, stderr, "--proof-report", &options.ProofReport) {
				return 2
			}
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown wake-word-physical-acceptance option %q\n", args[i])
			return 2
		}
	}
	report, err := buildWakeWordPhysicalAcceptanceReport(options)
	if err != nil {
		fmt.Fprintln(stderr, "wake word physical acceptance rejected: see structured input contract")
		return 1
	}
	if err := writeJSONWakeWordPhysicalAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode wake word physical acceptance report: %v\n", err)
		return 1
	}
	return 0
}

func buildWakeWordPhysicalAcceptanceReport(options wakeWordPhysicalAcceptanceOptions) (wakeWordPhysicalAcceptanceReport, error) {
	if strings.TrimSpace(options.PackageReport) == "" {
		return wakeWordPhysicalAcceptanceReport{}, fmt.Errorf("package report is required")
	}
	if strings.TrimSpace(options.ProofReport) == "" {
		return wakeWordPhysicalAcceptanceReport{}, fmt.Errorf("proof report is required")
	}
	if strings.TrimSpace(options.OutputDir) == "" {
		return wakeWordPhysicalAcceptanceReport{}, fmt.Errorf("output dir is required")
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		return wakeWordPhysicalAcceptanceReport{}, err
	}
	packageEvidence, packageFindings := loadProductWakeWordFirmwarePackageEvidence(options.PackageReport)
	if len(packageFindings) != 0 || !packageEvidence.Valid || packageEvidence.ProductReady || packageEvidence.FlashExecuted {
		return wakeWordPhysicalAcceptanceReport{}, fmt.Errorf("wake word firmware package is invalid")
	}
	proof, err := loadWakeWordPhysicalProofEvidence(options.ProofReport)
	if err != nil {
		return wakeWordPhysicalAcceptanceReport{}, err
	}
	report := wakeWordPhysicalAcceptanceReport{
		SchemaVersion:                        wakeWordPhysicalAcceptanceSchema,
		GeneratedAtMS:                        time.Now().UnixMilli(),
		Status:                               "accepted",
		ProductReady:                         true,
		Mode:                                 packageEvidence.Mode,
		DesiredPhrase:                        packageEvidence.DesiredPhrase,
		DesiredPinyin:                        packageEvidence.DesiredPinyin,
		Threshold:                            packageEvidence.Threshold,
		PackageSourceReport:                  packageEvidence.SourceReport,
		PackageArtifactName:                  packageEvidence.ArtifactName,
		PackageManifestName:                  packageEvidence.ManifestName,
		PhysicalDeviceOnline:                 proof.PhysicalDeviceOnline,
		FirmwareFlashExecuted:                proof.FirmwareFlashExecuted,
		GuardedFlashReportSource:             proof.GuardedFlashReportSource,
		OperatorCustomWakeObservationPresent: proof.OperatorCustomWakeObservationPresent,
		WakePhraseMatched:                    proof.WakePhraseMatched,
		FalseWakeAccepted:                    proof.FalseWakeAccepted,
		StockWakeAccepted:                    proof.StockWakeAccepted,
		RedactionOK:                          proof.RedactionOK,
	}
	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return wakeWordPhysicalAcceptanceReport{}, err
	}
	reportPath := filepath.Join(options.OutputDir, "a21-wake-word-physical-acceptance-"+time.Now().Format("20060102-150405")+".json")
	report.ReportPath = filepath.Base(reportPath)
	if err := writeWakeWordPhysicalAcceptanceReport(reportPath, report); err != nil {
		return wakeWordPhysicalAcceptanceReport{}, err
	}
	return report, nil
}

func loadWakeWordPhysicalProofEvidence(path string) (wakeWordPhysicalProofEvidence, error) {
	data, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return wakeWordPhysicalProofEvidence{}, fmt.Errorf("wake word physical proof is invalid")
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return wakeWordPhysicalProofEvidence{}, fmt.Errorf("wake word physical proof is invalid")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return wakeWordPhysicalProofEvidence{}, fmt.Errorf("wake word physical proof is invalid")
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productWakeWordPhysicalAcceptanceContainsForbiddenValue(raw) {
		return wakeWordPhysicalProofEvidence{}, fmt.Errorf("wake word physical proof is invalid")
	}
	var fixture wakeWordPhysicalProofFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return wakeWordPhysicalProofEvidence{}, fmt.Errorf("wake word physical proof is invalid")
	}
	if !validWakeWordPhysicalProofFixture(fixture) {
		return wakeWordPhysicalProofEvidence{}, fmt.Errorf("wake word physical proof is invalid")
	}
	return wakeWordPhysicalProofEvidence{
		PhysicalDeviceOnline:                 *fixture.PhysicalDeviceOnline,
		FirmwareFlashExecuted:                *fixture.FirmwareFlashExecuted,
		GuardedFlashReportSource:             strings.TrimSpace(fixture.GuardedFlashReportSource),
		OperatorCustomWakeObservationPresent: *fixture.OperatorCustomWakeObservationPresent,
		WakePhraseMatched:                    *fixture.WakePhraseMatched,
		FalseWakeAccepted:                    *fixture.FalseWakeAccepted,
		StockWakeAccepted:                    *fixture.StockWakeAccepted,
		RedactionOK:                          *fixture.RedactionOK,
	}, nil
}

func validWakeWordPhysicalProofFixture(fixture wakeWordPhysicalProofFixture) bool {
	if fixture.GeneratedAtMS == nil ||
		fixture.PhysicalDeviceOnline == nil ||
		fixture.FirmwareFlashExecuted == nil ||
		fixture.OperatorCustomWakeObservationPresent == nil ||
		fixture.WakePhraseMatched == nil ||
		fixture.FalseWakeAccepted == nil ||
		fixture.StockWakeAccepted == nil ||
		fixture.RedactionOK == nil {
		return false
	}
	if fixture.SchemaVersion != wakeWordPhysicalProofSchema ||
		*fixture.GeneratedAtMS <= 0 ||
		strings.TrimSpace(fixture.Status) != "observed" ||
		!*fixture.PhysicalDeviceOnline ||
		!*fixture.FirmwareFlashExecuted ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.GuardedFlashReportSource, ".json") ||
		!*fixture.OperatorCustomWakeObservationPresent ||
		!*fixture.WakePhraseMatched ||
		*fixture.FalseWakeAccepted ||
		*fixture.StockWakeAccepted ||
		!*fixture.RedactionOK ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.ReportPath, ".json") {
		return false
	}
	return true
}

func writeWakeWordPhysicalAcceptanceReport(path string, report wakeWordPhysicalAcceptanceReport) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeJSONWakeWordPhysicalAcceptance(file, report)
}

func writeWakeWordPhysicalProofReport(path string, report wakeWordPhysicalProofReport) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeJSONWakeWordPhysicalProof(file, report)
}

func writeJSONWakeWordPhysicalAcceptance(writer io.Writer, report wakeWordPhysicalAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONWakeWordPhysicalProof(writer io.Writer, report wakeWordPhysicalProofReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
