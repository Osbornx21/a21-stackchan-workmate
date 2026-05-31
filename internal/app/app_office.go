package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type officeHandoffOptions struct {
	ManifestPath string
	ArtifactDir  string
	Commit       string
	KeepRecent   int
	OutputDir    string
}
type officeHandoffReport struct {
	SchemaVersion              string                           `json:"schema_version"`
	GeneratedAtMS              int64                            `json:"generated_at_ms"`
	Metadata                   latencyBenchMetadata             `json:"metadata"`
	DryRun                     bool                             `json:"dry_run"`
	FlashAllowed               bool                             `json:"flash_allowed"`
	DeleteAllowed              bool                             `json:"delete_allowed"`
	PhysicalAcceptanceRequired bool                             `json:"physical_acceptance_required"`
	OfficePreflightRequired    bool                             `json:"office_preflight_required"`
	Commit                     string                           `json:"commit"`
	CurrentArtifactPath        string                           `json:"current_artifact_path"`
	Artifact                   firmwarecheck.ArtifactResult     `json:"artifact"`
	ArtifactRetention          firmwareArtifactPrunePlanSummary `json:"artifact_retention"`
	SerialDevices              []firmwarecheck.SerialDevice     `json:"serial_devices"`
	USBSerialCandidateCount    int                              `json:"usb_serial_candidate_count"`
	NextRequiredActions        []string                         `json:"next_required_actions"`
	ReportPath                 string                           `json:"report_path,omitempty"`
	Findings                   []officePreflightFinding         `json:"findings,omitempty"`
}
type officeAcceptanceOptions struct {
	HandoffPath       string
	OfficePreflight   string
	FirmwareFlashPlan string
	OutputDir         string
}
type officeAcceptanceReport struct {
	SchemaVersion             string                   `json:"schema_version"`
	GeneratedAtMS             int64                    `json:"generated_at_ms"`
	Metadata                  latencyBenchMetadata     `json:"metadata"`
	DryRun                    bool                     `json:"dry_run"`
	FlashAllowed              bool                     `json:"flash_allowed"`
	DeleteAllowed             bool                     `json:"delete_allowed"`
	PhysicalAcceptanceStatus  string                   `json:"physical_acceptance_status"`
	HandoffReportPath         string                   `json:"handoff_report_path"`
	OfficePreflightReportPath string                   `json:"office_preflight_report_path"`
	FirmwareFlashPlanReport   string                   `json:"firmware_flash_plan_report_path,omitempty"`
	Commit                    string                   `json:"commit,omitempty"`
	ArtifactPath              string                   `json:"artifact_path,omitempty"`
	ArtifactSHA256            string                   `json:"artifact_sha256,omitempty"`
	DeviceID                  string                   `json:"device_id,omitempty"`
	NextRequiredActions       []string                 `json:"next_required_actions"`
	ReportPath                string                   `json:"report_path,omitempty"`
	Findings                  []officePreflightFinding `json:"findings,omitempty"`
}

func runOfficeHandoff(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultOfficeHandoffOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 office-handoff --commit <git-sha> [--manifest firmware/stackchan/a21-firmware.json] [--artifact-dir firmware/artifacts] [--keep-recent 5] [--output-dir reports]")
			return 0
		case "--manifest":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--manifest requires a value")
				return 2
			}
			i++
			options.ManifestPath = args[i]
		case "--artifact-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--artifact-dir requires a value")
				return 2
			}
			i++
			options.ArtifactDir = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
		case "--keep-recent":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--keep-recent requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value < 1 {
				fmt.Fprintln(stderr, "--keep-recent must be a positive integer")
				return 2
			}
			options.KeepRecent = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown office-handoff option %q\n", args[i])
			return 2
		}
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	if err := validateA21InputPath(options.ManifestPath); err != nil {
		fmt.Fprintf(stderr, "office handoff manifest path invalid: %v\n", err)
		return 1
	}
	if err := validateA21InputPath(options.ArtifactDir); err != nil {
		fmt.Fprintf(stderr, "office handoff artifact dir invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "office handoff report dir invalid: %v\n", err)
		return 1
	}
	report, err := buildOfficeHandoffReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "office handoff failed: %v\n", err)
		return 1
	}
	reportPath, err := writeOfficeHandoffReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write office handoff report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONOfficeHandoff(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode office handoff report: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "office handoff manifest ok (no flash, no delete)")
	return 0
}
func defaultOfficeHandoffOptions() officeHandoffOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	return officeHandoffOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		ArtifactDir:  "firmware/artifacts",
		Commit:       currentGitCommit(projectRoot),
		KeepRecent:   5,
		OutputDir:    "reports",
	}
}
func buildOfficeHandoffReport(options officeHandoffOptions) (officeHandoffReport, error) {
	artifact, err := firmwarecheck.ValidateLatestArtifactForCommit(firmwarecheck.LatestArtifactOptions{
		ManifestPath: options.ManifestPath,
		ArtifactDir:  options.ArtifactDir,
		Commit:       options.Commit,
	})
	if err != nil {
		return officeHandoffReport{}, err
	}
	prunePlan, err := firmwarecheck.BuildArtifactPrunePlan(firmwarecheck.ArtifactPrunePlanOptions{
		ManifestPath: options.ManifestPath,
		ArtifactDir:  options.ArtifactDir,
		Commit:       options.Commit,
		KeepRecent:   options.KeepRecent,
	})
	if err != nil {
		return officeHandoffReport{}, err
	}
	serialDevices, serialErr := listFirmwareSerialDevices()
	findings := []officePreflightFinding{}
	if serialErr != nil {
		findings = append(findings, officePreflightFinding{
			Code:    "serial_inventory_failed",
			Message: serialErr.Error(),
		})
	}
	usbCandidates := officeUSBSerialCandidates(serialDevices)
	return officeHandoffReport{
		SchemaVersion:              "a21.office_handoff.v1",
		GeneratedAtMS:              time.Now().UnixMilli(),
		Metadata:                   buildLatencyBenchMetadata(),
		DryRun:                     true,
		FlashAllowed:               false,
		DeleteAllowed:              false,
		PhysicalAcceptanceRequired: true,
		OfficePreflightRequired:    true,
		Commit:                     options.Commit,
		CurrentArtifactPath:        artifact.ArtifactPath,
		Artifact:                   artifact,
		ArtifactRetention:          summarizeFirmwareArtifactPrunePlan(prunePlan),
		SerialDevices:              serialDevices,
		USBSerialCandidateCount:    len(usbCandidates),
		NextRequiredActions: []string{
			"Run make release-check on the travel machine before leaving home.",
			"At the office, connect only the intended StackChan/CoreS3 and run A21_DEVICE_ID=stackchan-001 make office-preflight.",
			"Only after a fresh A21 Gateway device report and explicit USB serial path, run make firmware-flash-plan.",
			"Real flashing remains locked until a future explicit guarded A21 flash command exists.",
		},
		Findings: findings,
	}, nil
}
func runOfficeAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := officeAcceptanceOptions{OutputDir: "reports"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 office-acceptance --handoff reports/a21-office-handoff-...json --office-preflight reports/a21-office-preflight-...json [--firmware-flash-plan reports/a21-firmware-flash-plan-...json] [--output-dir reports]")
			return 0
		case "--handoff":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--handoff requires a value")
				return 2
			}
			i++
			options.HandoffPath = args[i]
		case "--office-preflight":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--office-preflight requires a value")
				return 2
			}
			i++
			options.OfficePreflight = args[i]
		case "--firmware-flash-plan":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--firmware-flash-plan requires a value")
				return 2
			}
			i++
			options.FirmwareFlashPlan = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown office-acceptance option %q\n", args[i])
			return 2
		}
	}
	if options.HandoffPath == "" {
		fmt.Fprintln(stderr, "--handoff requires a value")
		return 2
	}
	if options.OfficePreflight == "" {
		fmt.Fprintln(stderr, "--office-preflight requires a value")
		return 2
	}
	for _, path := range []string{options.HandoffPath, options.OfficePreflight, options.FirmwareFlashPlan} {
		if path == "" {
			continue
		}
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "office acceptance report path invalid: %v\n", err)
			return 1
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "office acceptance report dir invalid: %v\n", err)
		return 1
	}
	report, err := buildOfficeAcceptanceReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "office acceptance failed: %v\n", err)
		return 1
	}
	reportPath, err := writeOfficeAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write office acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONOfficeAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode office acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "office acceptance gate failed (no flash, no delete)")
		return 1
	}
	fmt.Fprintln(stdout, "office acceptance gate ok (no flash, no delete)")
	return 0
}
func buildOfficeAcceptanceReport(options officeAcceptanceOptions) (officeAcceptanceReport, error) {
	var handoff officeHandoffReport
	if err := readJSONFile(options.HandoffPath, &handoff); err != nil {
		return officeAcceptanceReport{}, err
	}
	var preflight officePreflightReport
	if err := readJSONFile(options.OfficePreflight, &preflight); err != nil {
		return officeAcceptanceReport{}, err
	}
	report := officeAcceptanceReport{
		SchemaVersion:             "a21.office_acceptance.v1",
		GeneratedAtMS:             time.Now().UnixMilli(),
		Metadata:                  buildLatencyBenchMetadata(),
		DryRun:                    true,
		FlashAllowed:              false,
		DeleteAllowed:             false,
		PhysicalAcceptanceStatus:  "ready_for_physical_acceptance",
		HandoffReportPath:         options.HandoffPath,
		OfficePreflightReportPath: options.OfficePreflight,
		Commit:                    firstNonEmpty(handoff.Commit, preflight.Commit),
		ArtifactPath:              firstNonEmpty(handoff.CurrentArtifactPath, preflightArtifactPath(preflight)),
		ArtifactSHA256:            firstNonEmpty(handoff.Artifact.SHA256, preflightArtifactSHA256(preflight)),
		DeviceID:                  preflight.DeviceID,
		NextRequiredActions: []string{
			"Keep this report with the handoff and office-preflight receipts.",
			"Run firmware-flash-plan only when an explicit USB serial port and fresh A21 Gateway device report are present.",
			"Real flashing remains locked until a future explicit guarded A21 flash command exists.",
		},
	}
	if handoff.SchemaVersion != "a21.office_handoff.v1" {
		report.addFinding("handoff_schema_invalid", "handoff report schema is not a21.office_handoff.v1")
	}
	if preflight.SchemaVersion != "a21.office_preflight.v1" {
		report.addFinding("office_preflight_schema_invalid", "office preflight report schema is not a21.office_preflight.v1")
	}
	if handoff.FlashAllowed || handoff.DeleteAllowed || preflight.FlashAllowed {
		report.addFinding("unsafe_permission", "handoff or office preflight unexpectedly allowed flash/delete")
	}
	if !preflight.ReadyForFlashPlan {
		report.addFinding("office_preflight_not_ready", "office preflight is not ready for flash-plan evidence")
	}
	if handoff.Commit != "" && preflight.Commit != "" && !sameCLICommit(handoff.Commit, preflight.Commit) {
		report.addFinding("commit_mismatch", "handoff and office preflight commits differ")
	}
	handoffArtifact := handoff.CurrentArtifactPath
	preflightArtifact := preflightArtifactPath(preflight)
	if handoffArtifact != "" && preflightArtifact != "" && !sameCleanCLIPath(handoffArtifact, preflightArtifact) {
		report.addFinding("artifact_mismatch", "handoff and office preflight artifacts differ")
	}
	handoffSHA := handoff.Artifact.SHA256
	preflightSHA := preflightArtifactSHA256(preflight)
	if handoffSHA != "" && preflightSHA != "" && !strings.EqualFold(handoffSHA, preflightSHA) {
		report.addFinding("artifact_sha_mismatch", "handoff and office preflight artifact sha256 values differ")
	}
	if options.FirmwareFlashPlan != "" {
		var flashPlan firmwarecheck.FlashPlanResult
		if err := readJSONFile(options.FirmwareFlashPlan, &flashPlan); err != nil {
			return officeAcceptanceReport{}, err
		}
		report.FirmwareFlashPlanReport = options.FirmwareFlashPlan
		if flashPlan.FlashAllowed {
			report.addFinding("unsafe_flash_plan", "firmware flash plan unexpectedly allowed flashing")
		}
		if flashPlan.Commit != "" && report.Commit != "" && !sameCLICommit(report.Commit, flashPlan.Commit) {
			report.addFinding("flash_plan_commit_mismatch", "firmware flash plan commit differs")
		}
		if flashPlan.ArtifactPath != "" && report.ArtifactPath != "" && !sameCleanCLIPath(report.ArtifactPath, flashPlan.ArtifactPath) {
			report.addFinding("flash_plan_artifact_mismatch", "firmware flash plan artifact differs")
		}
		if flashPlan.DeviceID != "" && report.DeviceID != "" && flashPlan.DeviceID != report.DeviceID {
			report.addFinding("flash_plan_device_mismatch", "firmware flash plan device differs")
		}
	}
	if len(report.Findings) > 0 {
		report.PhysicalAcceptanceStatus = "blocked"
	}
	return report, nil
}
func (report *officeAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}
func writeOfficeHandoffReport(outputDir string, report officeHandoffReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-office-handoff-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONOfficeHandoff(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeOfficeAcceptanceReport(outputDir string, report officeAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-office-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONOfficeAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONOfficeHandoff(writer io.Writer, report officeHandoffReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
func writeJSONOfficeAcceptance(writer io.Writer, report officeAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
