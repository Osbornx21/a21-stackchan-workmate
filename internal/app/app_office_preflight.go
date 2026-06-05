package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/firmwarecheck"
)

type officePreflightOptions struct {
	ManifestPath   string
	ArtifactDir    string
	GatewayURL     string
	DeviceID       string
	Commit         string
	MaxDeviceAgeMS int64
	OutputDir      string
}
type officePreflightFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type officePreflightReport struct {
	SchemaVersion               string                                        `json:"schema_version"`
	GeneratedAtMS               int64                                         `json:"generated_at_ms"`
	Metadata                    latencyBenchMetadata                          `json:"metadata"`
	DryRun                      bool                                          `json:"dry_run"`
	FlashAllowed                bool                                          `json:"flash_allowed"`
	ReadyForFlashPlan           bool                                          `json:"ready_for_flash_plan"`
	NextRequiredConfirmation    string                                        `json:"next_required_confirmation"`
	GatewayURL                  string                                        `json:"gateway_url"`
	DeviceID                    string                                        `json:"device_id"`
	Commit                      string                                        `json:"commit"`
	MaxDeviceAgeMS              int64                                         `json:"max_device_age_ms"`
	Artifact                    *firmwarecheck.ArtifactResult                 `json:"artifact,omitempty"`
	ProductLaneArtifactEvidence *officialStackChanProductLaneArtifactEvidence `json:"product_lane_artifact_evidence,omitempty"`
	Gateway                     *firmwareDeviceReport                         `json:"gateway,omitempty"`
	DeviceIdentity              *firmwarecheck.DeviceIdentityResult           `json:"device_identity,omitempty"`
	SerialDevices               []firmwarecheck.SerialDevice                  `json:"serial_devices"`
	USBSerialCandidates         []firmwarecheck.SerialDevice                  `json:"usb_serial_candidates"`
	DeviceReportPath            string                                        `json:"device_report_path,omitempty"`
	ReportPath                  string                                        `json:"report_path,omitempty"`
	Findings                    []officePreflightFinding                      `json:"findings,omitempty"`
}

func readJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
func preflightArtifactPath(report officePreflightReport) string {
	if report.Artifact == nil {
		return ""
	}
	return report.Artifact.ArtifactPath
}
func preflightArtifactSHA256(report officePreflightReport) string {
	if report.Artifact == nil {
		return ""
	}
	return report.Artifact.SHA256
}
func sameCLICommit(left string, right string) bool {
	left = strings.ToLower(left)
	right = strings.ToLower(right)
	return left == right || strings.HasPrefix(left, right) || strings.HasPrefix(right, left)
}
func sameCleanCLIPath(left string, right string) bool {
	leftClean, leftErr := filepath.Abs(filepath.Clean(left))
	rightClean, rightErr := filepath.Abs(filepath.Clean(right))
	if leftErr != nil || rightErr != nil {
		return filepath.Clean(left) == filepath.Clean(right)
	}
	return leftClean == rightClean
}
func runOfficePreflight(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultOfficePreflightOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 office-preflight --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> --max-device-age-ms 300000 [--manifest firmware/stackchan/a21-firmware.json] [--artifact-dir firmware/artifacts] [--output-dir reports]")
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
			fmt.Fprintf(stderr, "unknown office-preflight option %q\n", args[i])
			return 2
		}
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
	if err := validateA21InputPath(options.ManifestPath); err != nil {
		fmt.Fprintf(stderr, "office preflight manifest path invalid: %v\n", err)
		return 1
	}
	if err := validateA21InputPath(options.ArtifactDir); err != nil {
		fmt.Fprintf(stderr, "office preflight artifact dir invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "office preflight report dir invalid: %v\n", err)
		return 1
	}
	report := buildOfficePreflightReport(options)
	reportPath, err := writeOfficePreflightReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write office preflight report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONOfficePreflight(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode office preflight report: %v\n", err)
		return 1
	}
	if !report.ReadyForFlashPlan {
		fmt.Fprintln(stdout, "office preflight failed (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "office preflight ok (no flash performed)")
	return 0
}
func defaultOfficePreflightOptions() officePreflightOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	maxDeviceAgeMS, _ := strconv.ParseInt(strings.TrimSpace(os.Getenv("A21_DEVICE_MAX_AGE_MS")), 10, 64)
	return officePreflightOptions{
		ManifestPath:   "firmware/stackchan/a21-firmware.json",
		ArtifactDir:    "firmware/artifacts",
		GatewayURL:     firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:       deviceID,
		Commit:         currentGitCommit(projectRoot),
		MaxDeviceAgeMS: maxDeviceAgeMS,
		OutputDir:      "reports",
	}
}
func buildOfficePreflightReport(options officePreflightOptions) officePreflightReport {
	report := officePreflightReport{
		SchemaVersion:            "a21.office_preflight.v1",
		GeneratedAtMS:            time.Now().UnixMilli(),
		Metadata:                 buildLatencyBenchMetadata(),
		DryRun:                   true,
		FlashAllowed:             false,
		NextRequiredConfirmation: "firmware-flash-plan_with_explicit_usb_port",
		GatewayURL:               sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                 options.DeviceID,
		Commit:                   options.Commit,
		MaxDeviceAgeMS:           options.MaxDeviceAgeMS,
	}
	report.ProductLaneArtifactEvidence = discoverOfficialStackChanProductLaneArtifactEvidence(options.OutputDir)

	artifact, err := firmwarecheck.ValidateLatestArtifactForCommit(firmwarecheck.LatestArtifactOptions{
		ManifestPath: options.ManifestPath,
		ArtifactDir:  options.ArtifactDir,
		Commit:       options.Commit,
	})
	if err != nil {
		if officialProductLaneEvidenceSatisfiesArtifact(report.ProductLaneArtifactEvidence) {
			report.addFinding("official_product_lane_artifact_evidence_present", "official-compatible product-lane artifact evidence is present; release-ledger artifact is still required before a new flash plan")
		} else {
			if report.ProductLaneArtifactEvidence != nil {
				report.addFinding("official_product_lane_artifact_evidence_planned", "official-compatible product-lane evidence is dry-run or planned only; release-ledger artifact is still required before a new flash plan")
			}
			report.addFinding("current_artifact_invalid", err.Error())
		}
	} else {
		report.Artifact = &artifact
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
					report.addFinding("device_identity_not_ready", identityErr.Error())
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
			report.addFinding("usb_serial_candidate_missing", "no available USB serial candidate found for a future explicit firmware flash plan")
		}
	}

	report.ReadyForFlashPlan = report.Artifact != nil &&
		report.Gateway != nil &&
		report.DeviceIdentity != nil &&
		len(report.USBSerialCandidates) > 0 &&
		len(report.Findings) == 0
	return report
}
func (report *officePreflightReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{
		Code:    code,
		Message: message,
	})
}
func officeUSBSerialCandidates(devices []firmwarecheck.SerialDevice) []firmwarecheck.SerialDevice {
	candidates := make([]firmwarecheck.SerialDevice, 0, len(devices))
	for _, device := range devices {
		if device.USBModem && device.Usage.Exists && !device.Usage.InUse {
			candidates = append(candidates, device)
		}
	}
	return candidates
}
func sanitizedOfficeGatewayURL(raw string) string {
	_, safe, err := firmwareDeviceReportEndpoint(raw)
	if err != nil {
		return ""
	}
	return safe
}
func writeOfficePreflightReport(outputDir string, report officePreflightReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-office-preflight-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONOfficePreflight(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONOfficePreflight(writer io.Writer, report officePreflightReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
