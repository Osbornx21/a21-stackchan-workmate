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

type stackChanIdentityAcceptanceOptions struct {
	OfficeAcceptancePath string
	ManifestPath         string
	ArtifactDir          string
	GatewayURL           string
	DeviceID             string
	Commit               string
	MaxDeviceAgeMS       int64
	OutputDir            string
}
type stackChanIdentityAcceptanceReport struct {
	SchemaVersion              string                              `json:"schema_version"`
	GeneratedAtMS              int64                               `json:"generated_at_ms"`
	Metadata                   latencyBenchMetadata                `json:"metadata"`
	DryRun                     bool                                `json:"dry_run"`
	FlashAllowed               bool                                `json:"flash_allowed"`
	DeleteAllowed              bool                                `json:"delete_allowed"`
	HardwareAcceptanceScope    string                              `json:"hardware_acceptance_scope"`
	IdentityAcceptanceStatus   string                              `json:"identity_acceptance_status"`
	OfficeAcceptanceReportPath string                              `json:"office_acceptance_report_path"`
	GatewayURL                 string                              `json:"gateway_url"`
	DeviceID                   string                              `json:"device_id"`
	Commit                     string                              `json:"commit"`
	MaxDeviceAgeMS             int64                               `json:"max_device_age_ms"`
	ArtifactPath               string                              `json:"artifact_path,omitempty"`
	ArtifactSHA256             string                              `json:"artifact_sha256,omitempty"`
	Artifact                   *firmwarecheck.ArtifactResult       `json:"artifact,omitempty"`
	Gateway                    *firmwareDeviceReport               `json:"gateway,omitempty"`
	DeviceIdentity             *firmwarecheck.DeviceIdentityResult `json:"device_identity,omitempty"`
	SerialDevices              []firmwarecheck.SerialDevice        `json:"serial_devices"`
	USBSerialCandidates        []firmwarecheck.SerialDevice        `json:"usb_serial_candidates"`
	DeviceReportPath           string                              `json:"device_report_path,omitempty"`
	NextRequiredActions        []string                            `json:"next_required_actions"`
	ReportPath                 string                              `json:"report_path,omitempty"`
	Findings                   []officePreflightFinding            `json:"findings,omitempty"`
}

func runStackChanIdentityAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanIdentityAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-accept --check identity --office-acceptance reports/a21-office-acceptance-...json --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> --max-device-age-ms 300000 [--output-dir reports]")
			return 0
		case "--office-acceptance":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--office-acceptance requires a value")
				return 2
			}
			i++
			options.OfficeAcceptancePath = args[i]
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
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			options.GatewayURL = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
		case "--max-device-age-ms":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--max-device-age-ms requires a value")
				return 2
			}
			i++
			value, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil || value <= 0 {
				fmt.Fprintln(stderr, "--max-device-age-ms must be a positive integer")
				return 2
			}
			options.MaxDeviceAgeMS = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-identity-acceptance option %q\n", args[i])
			return 2
		}
	}
	if options.OfficeAcceptancePath == "" {
		fmt.Fprintln(stderr, "--office-acceptance requires a value")
		return 2
	}
	if options.DeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	if options.MaxDeviceAgeMS <= 0 {
		fmt.Fprintln(stderr, "--max-device-age-ms requires a value")
		return 2
	}
	for _, path := range []string{options.OfficeAcceptancePath, options.ManifestPath, options.ArtifactDir} {
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "stackchan identity acceptance path invalid: %v\n", err)
			return 1
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan identity acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanIdentityAcceptanceReport(options)
	reportPath, err := writeStackChanIdentityAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan identity acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanIdentityAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan identity acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan identity acceptance failed (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan identity acceptance ok (no flash performed)")
	return 0
}
func defaultStackChanIdentityAcceptanceOptions() stackChanIdentityAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	maxDeviceAgeMS, _ := strconv.ParseInt(strings.TrimSpace(os.Getenv("A21_DEVICE_MAX_AGE_MS")), 10, 64)
	return stackChanIdentityAcceptanceOptions{
		ManifestPath:   "firmware/stackchan/a21-firmware.json",
		ArtifactDir:    "firmware/artifacts",
		GatewayURL:     firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:       deviceID,
		Commit:         currentGitCommit(projectRoot),
		MaxDeviceAgeMS: maxDeviceAgeMS,
		OutputDir:      "reports",
	}
}
func buildStackChanIdentityAcceptanceReport(options stackChanIdentityAcceptanceOptions) stackChanIdentityAcceptanceReport {
	report := stackChanIdentityAcceptanceReport{
		SchemaVersion:              "a21.stackchan_identity_acceptance.v1",
		GeneratedAtMS:              time.Now().UnixMilli(),
		Metadata:                   buildLatencyBenchMetadata(),
		DryRun:                     true,
		FlashAllowed:               false,
		DeleteAllowed:              false,
		HardwareAcceptanceScope:    "identity_only",
		IdentityAcceptanceStatus:   "identity_confirmed",
		OfficeAcceptanceReportPath: options.OfficeAcceptancePath,
		GatewayURL:                 sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                   options.DeviceID,
		Commit:                     options.Commit,
		MaxDeviceAgeMS:             options.MaxDeviceAgeMS,
		NextRequiredActions: []string{
			"Keep this report with the office acceptance, device report, and firmware artifact receipts.",
			"Identity acceptance does not prove microphone, speaker, screen, servo, RGB, OTA, or latency acceptance.",
			"Real flashing remains locked until a future explicit guarded A21 flash command exists.",
		},
	}

	var officeAcceptance officeAcceptanceReport
	if err := readJSONFile(options.OfficeAcceptancePath, &officeAcceptance); err != nil {
		report.addFinding("office_acceptance_unreadable", err.Error())
	} else {
		validateOfficeAcceptanceForStackChanIdentity(&report, officeAcceptance)
	}

	artifact, err := firmwarecheck.ValidateLatestArtifactForCommit(firmwarecheck.LatestArtifactOptions{
		ManifestPath: options.ManifestPath,
		ArtifactDir:  options.ArtifactDir,
		Commit:       options.Commit,
	})
	if err != nil {
		report.addFinding("current_artifact_invalid", err.Error())
	} else {
		report.Artifact = &artifact
		report.ArtifactPath = artifact.ArtifactPath
		report.ArtifactSHA256 = artifact.SHA256
		if officeAcceptance.ArtifactPath != "" && !sameCleanCLIPath(officeAcceptance.ArtifactPath, artifact.ArtifactPath) {
			report.addFinding("office_acceptance_artifact_mismatch", "office acceptance artifact differs from current artifact")
		}
		if officeAcceptance.ArtifactSHA256 != "" && !strings.EqualFold(officeAcceptance.ArtifactSHA256, artifact.SHA256) {
			report.addFinding("office_acceptance_artifact_sha_mismatch", "office acceptance artifact sha256 differs from current artifact")
		}
	}

	gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_failed", err.Error())
	} else {
		deviceReportPath, writeErr := writeFirmwareDeviceReport(options.OutputDir, gatewayReport)
		if writeErr != nil {
			report.addFinding("device_report_write_failed", writeErr.Error())
		} else {
			gatewayReport.DeviceReportPath = deviceReportPath
			report.DeviceReportPath = deviceReportPath
			report.Gateway = &gatewayReport
			if report.Artifact != nil {
				deviceIdentity, identityErr := firmwarecheck.ValidateDeviceIdentity(firmwarecheck.DeviceIdentityOptions{
					ManifestPath:      options.ManifestPath,
					ArtifactPath:      report.Artifact.ArtifactPath,
					ReportPath:        deviceReportPath,
					ExpectedDeviceID:  options.DeviceID,
					ExpectedGitCommit: options.Commit,
					MaxDeviceAgeMS:    options.MaxDeviceAgeMS,
				})
				if identityErr != nil {
					report.addFinding("device_identity_not_confirmed", identityErr.Error())
				} else {
					report.DeviceIdentity = &deviceIdentity
					if quiescentErr := firmwarecheck.ValidateFlashPlanDeviceQuiescent(deviceIdentity.Device); quiescentErr != nil {
						report.addFinding("device_not_quiescent", quiescentErr.Error())
					}
				}
			}
		}
	}

	serialDevices, err := listFirmwareSerialDevices()
	if err != nil {
		report.addFinding("serial_inventory_failed", err.Error())
	} else {
		report.SerialDevices = serialDevices
		report.USBSerialCandidates = officeUSBSerialCandidates(serialDevices)
		if len(report.USBSerialCandidates) == 0 {
			report.addFinding("usb_serial_candidate_missing", "no available USB serial candidate found for physical identity acceptance")
		}
	}

	if len(report.Findings) > 0 {
		report.IdentityAcceptanceStatus = "blocked"
	}
	return report
}
func validateOfficeAcceptanceForStackChanIdentity(report *stackChanIdentityAcceptanceReport, officeAcceptance officeAcceptanceReport) {
	if officeAcceptance.SchemaVersion != "a21.office_acceptance.v1" {
		report.addFinding("office_acceptance_schema_invalid", "office acceptance report schema is not a21.office_acceptance.v1")
	}
	if officeAcceptance.FlashAllowed || officeAcceptance.DeleteAllowed {
		report.addFinding("unsafe_permission", "office acceptance unexpectedly allowed flash/delete")
	}
	if officeAcceptance.PhysicalAcceptanceStatus != "ready_for_physical_acceptance" || len(officeAcceptance.Findings) > 0 {
		report.addFinding("office_acceptance_not_ready", "office acceptance report is not ready for physical identity confirmation")
	}
	if officeAcceptance.Commit != "" && !sameCLICommit(officeAcceptance.Commit, report.Commit) {
		report.addFinding("office_acceptance_commit_mismatch", "office acceptance commit differs from expected commit")
	}
	if officeAcceptance.DeviceID != "" && officeAcceptance.DeviceID != report.DeviceID {
		report.addFinding("office_acceptance_device_mismatch", "office acceptance device differs from expected device")
	}
}
func (report *stackChanIdentityAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}
func writeStackChanIdentityAcceptanceReport(outputDir string, report stackChanIdentityAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-identity-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanIdentityAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONStackChanIdentityAcceptance(writer io.Writer, report stackChanIdentityAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
