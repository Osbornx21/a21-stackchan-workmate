package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type stackChanIMUProbeAcceptanceOptions struct {
	GatewayURL      string
	DeviceID        string
	Commit          string
	WindowMS        int
	MinSamples      int
	MinAccelTotalMG int
	MaxReadErrors   int
	OutputDir       string
}
type stackChanIMUProbeAcceptanceThresholds struct {
	MinSamples      int `json:"min_samples"`
	MinAccelTotalMG int `json:"min_accel_total_mg"`
	MaxReadErrors   int `json:"max_read_errors"`
}
type stackChanIMUProbeAcceptanceReport struct {
	SchemaVersion                string                                `json:"schema_version"`
	GeneratedAtMS                int64                                 `json:"generated_at_ms"`
	Metadata                     latencyBenchMetadata                  `json:"metadata"`
	DryRun                       bool                                  `json:"dry_run"`
	FlashAllowed                 bool                                  `json:"flash_allowed"`
	DeleteAllowed                bool                                  `json:"delete_allowed"`
	HardwareAcceptanceScope      string                                `json:"hardware_acceptance_scope"`
	IMUProbeAcceptanceStatus     string                                `json:"imu_probe_acceptance_status"`
	ProductionCapabilityPromoted bool                                  `json:"production_capability_promoted"`
	GatewayURL                   string                                `json:"gateway_url"`
	DeviceID                     string                                `json:"device_id"`
	Commit                       string                                `json:"commit"`
	WindowMS                     int                                   `json:"window_ms"`
	WindowStartedAtMS            int64                                 `json:"window_started_at_ms,omitempty"`
	WindowEndedAtMS              int64                                 `json:"window_ended_at_ms,omitempty"`
	Firmware                     firmwarecheck.DeviceIdentityFirmware  `json:"firmware"`
	Capabilities                 map[string]string                     `json:"capabilities,omitempty"`
	IMU                          string                                `json:"imu,omitempty"`
	RuntimeEcho                  map[string]string                     `json:"runtime_echo,omitempty"`
	RuntimeEchoBefore            map[string]string                     `json:"runtime_echo_before,omitempty"`
	IMUAvailable                 int                                   `json:"imu_available"`
	IMUSamples                   int                                   `json:"imu_samples"`
	IMUReadErrors                int                                   `json:"imu_read_errors"`
	IMUAccelMGX                  int                                   `json:"imu_accel_mg_x"`
	IMUAccelMGY                  int                                   `json:"imu_accel_mg_y"`
	IMUAccelMGZ                  int                                   `json:"imu_accel_mg_z"`
	IMUGyroMDPSX                 int                                   `json:"imu_gyro_mdps_x"`
	IMUGyroMDPSY                 int                                   `json:"imu_gyro_mdps_y"`
	IMUGyroMDPSZ                 int                                   `json:"imu_gyro_mdps_z"`
	IMUPosture                   string                                `json:"imu_posture"`
	IMUAccelTotalMG              int                                   `json:"imu_accel_total_mg"`
	IMUSamplesDelta              int                                   `json:"imu_samples_delta"`
	IMUReadErrorsDelta           int                                   `json:"imu_read_errors_delta"`
	SampleRateHz                 float64                               `json:"sample_rate_hz,omitempty"`
	Thresholds                   stackChanIMUProbeAcceptanceThresholds `json:"thresholds"`
	NextRequiredActions          []string                              `json:"next_required_actions"`
	ReportPath                   string                                `json:"report_path,omitempty"`
	Findings                     []officePreflightFinding              `json:"findings,omitempty"`
}

func runStackChanIMUProbeAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanIMUProbeAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-accept --check imu-probe --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> [--window-ms 1500] [--min-samples 10] [--min-accel-total-mg 500] [--max-read-errors 0] [--output-dir reports]")
			return 0
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
		case "--window-ms":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--window-ms")
			if !ok {
				return 2
			}
			if value > 120000 {
				fmt.Fprintln(stderr, "--window-ms must be at most 120000")
				return 2
			}
			options.WindowMS = value
		case "--min-samples":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-samples")
			if !ok {
				return 2
			}
			options.MinSamples = value
		case "--min-accel-total-mg":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-accel-total-mg")
			if !ok {
				return 2
			}
			options.MinAccelTotalMG = value
		case "--max-read-errors":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--max-read-errors")
			if !ok {
				return 2
			}
			options.MaxReadErrors = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-imu-probe-acceptance option %q\n", args[i])
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
	if _, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/devices", nil); err != nil {
		fmt.Fprintf(stderr, "stackchan IMU probe acceptance gateway URL invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan IMU probe acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanIMUProbeAcceptanceReport(options)
	reportPath, err := writeStackChanIMUProbeAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan IMU probe acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanIMUProbeAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan IMU probe acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan IMU probe acceptance blocked (diagnostic only, no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan IMU probe acceptance ok (diagnostic only, no flash performed)")
	return 0
}
func defaultStackChanIMUProbeAcceptanceOptions() stackChanIMUProbeAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanIMUProbeAcceptanceOptions{
		GatewayURL:      firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:        deviceID,
		Commit:          currentGitCommit(projectRoot),
		WindowMS:        1500,
		MinSamples:      10,
		MinAccelTotalMG: 500,
		MaxReadErrors:   0,
		OutputDir:       "reports",
	}
}
func buildStackChanIMUProbeAcceptanceReport(options stackChanIMUProbeAcceptanceOptions) stackChanIMUProbeAcceptanceReport {
	report := stackChanIMUProbeAcceptanceReport{
		SchemaVersion:                "a21.stackchan_imu_probe_acceptance.v1",
		GeneratedAtMS:                time.Now().UnixMilli(),
		Metadata:                     buildLatencyBenchMetadata(),
		DryRun:                       true,
		FlashAllowed:                 false,
		DeleteAllowed:                false,
		HardwareAcceptanceScope:      "diagnostic_imu_only",
		IMUProbeAcceptanceStatus:     "confirmed",
		ProductionCapabilityPromoted: false,
		GatewayURL:                   sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                     options.DeviceID,
		Commit:                       options.Commit,
		WindowMS:                     options.WindowMS,
		Thresholds: stackChanIMUProbeAcceptanceThresholds{
			MinSamples:      options.MinSamples,
			MinAccelTotalMG: options.MinAccelTotalMG,
			MaxReadErrors:   options.MaxReadErrors,
		},
		NextRequiredActions: []string{
			"Keep this report with the IMU-probe flash receipt and Gateway device report.",
			"Treat IMU telemetry as read-only diagnostic evidence, not product gesture or posture acceptance.",
			"Promote IMU behavior only through a later ADR and release-firmware acceptance gate.",
		},
	}

	beforeGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_before_failed", err.Error())
		report.IMUProbeAcceptanceStatus = "blocked"
		return report
	}
	beforeDevice, ok := findFirmwareDeviceRecord(beforeGatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_before_missing", "expected StackChan device is missing from Gateway before IMU probe")
		report.IMUProbeAcceptanceStatus = "blocked"
		return report
	}
	validateStackChanIMUProbeIdentity(&report, options, beforeDevice)
	report.RuntimeEchoBefore = beforeDevice.RuntimeEcho
	beforeSamples := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "imu_samples")
	beforeReadErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "imu_read_errors")
	if len(report.Findings) > 0 {
		report.IMUProbeAcceptanceStatus = "blocked"
		return report
	}

	report.WindowStartedAtMS = time.Now().UnixMilli()
	if options.WindowMS > 0 {
		time.Sleep(time.Duration(options.WindowMS) * time.Millisecond)
	}
	report.WindowEndedAtMS = time.Now().UnixMilli()

	afterGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_after_failed", err.Error())
	} else {
		report.GatewayURL = afterGatewayReport.GatewayURL
		afterDevice, ok := findFirmwareDeviceRecord(afterGatewayReport.Devices, options.DeviceID)
		if !ok {
			report.addFinding("gateway_device_after_missing", "expected StackChan device is missing from Gateway after IMU probe")
		} else {
			validateStackChanIMUProbeDevice(&report, options, afterDevice)
			report.IMUSamplesDelta = report.IMUSamples - beforeSamples
			report.IMUReadErrorsDelta = report.IMUReadErrors - beforeReadErrors
			if report.WindowEndedAtMS > report.WindowStartedAtMS {
				report.SampleRateHz = roundedStackChanDiagnosticValue(float64(report.IMUSamplesDelta) * 1000 / float64(report.WindowEndedAtMS-report.WindowStartedAtMS))
			}
			validateStackChanIMUProbeDeltas(&report, options)
		}
	}
	if len(report.Findings) > 0 {
		report.IMUProbeAcceptanceStatus = "blocked"
	}
	return report
}
func validateStackChanIMUProbeIdentity(report *stackChanIMUProbeAcceptanceReport, options stackChanIMUProbeAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	report.Firmware = device.Firmware
	report.Capabilities = device.Capabilities
	report.IMU = device.Capabilities["imu"]
	if device.DeviceID != options.DeviceID {
		report.addFinding("device_id_mismatch", "Gateway device id differs from expected StackChan device")
	}
	if device.IdentityStatus != "ok" {
		report.addFinding("device_identity_not_ok", "Gateway device identity is not ok")
	}
	if device.ConnectionStatus != "" && device.ConnectionStatus != "online" {
		report.addFinding("device_not_online", "Gateway device is not online")
	}
	if device.Firmware.ID != "a21-stackchan" {
		report.addFinding("firmware_id_mismatch", "device firmware id is not a21-stackchan")
	}
	if device.Firmware.Board != "m5stack-cores3" {
		report.addFinding("firmware_board_mismatch", "device firmware board is not m5stack-cores3")
	}
	if device.Firmware.Commit == "" || !sameCLICommit(device.Firmware.Commit, options.Commit) {
		report.addFinding("firmware_commit_mismatch", "device firmware commit differs from expected commit")
	}
	if report.IMU != firmwarecheck.IMUProbeCapabilityStatus {
		report.addFinding("imu_not_diagnostic_probe", "device IMU capability is not the diagnostic IMU-probe status")
	}
}
func validateStackChanIMUProbeDevice(report *stackChanIMUProbeAcceptanceReport, options stackChanIMUProbeAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	validateStackChanIMUProbeIdentity(report, options, device)
	report.RuntimeEcho = device.RuntimeEcho
	report.IMUAvailable = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_available")
	report.IMUSamples = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_samples")
	report.IMUReadErrors = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_read_errors")
	report.IMUAccelMGX = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_accel_mg_x")
	report.IMUAccelMGY = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_accel_mg_y")
	report.IMUAccelMGZ = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_accel_mg_z")
	report.IMUGyroMDPSX = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_gyro_mdps_x")
	report.IMUGyroMDPSY = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_gyro_mdps_y")
	report.IMUGyroMDPSZ = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_gyro_mdps_z")
	report.IMUPosture = strings.TrimSpace(device.RuntimeEcho["imu_posture"])
	report.IMUAccelTotalMG = stackChanAbsInt(report.IMUAccelMGX) + stackChanAbsInt(report.IMUAccelMGY) + stackChanAbsInt(report.IMUAccelMGZ)
	if report.IMUAvailable != 1 {
		report.addFinding("imu_not_available", "IMU diagnostic runtime did not report availability")
	}
	if report.IMUReadErrors > options.MaxReadErrors {
		report.addFinding("imu_read_errors_above_threshold", "IMU read errors exceed threshold")
	}
	if report.IMUAccelTotalMG < options.MinAccelTotalMG {
		report.addFinding("imu_accel_total_below_threshold", "IMU acceleration evidence is below threshold")
	}
	if report.IMUPosture == "" || report.IMUPosture == "unknown" {
		report.addFinding("imu_posture_unknown", "IMU posture is missing or unknown")
	}
}
func validateStackChanIMUProbeDeltas(report *stackChanIMUProbeAcceptanceReport, options stackChanIMUProbeAcceptanceOptions) {
	if report.IMUSamplesDelta < options.MinSamples {
		report.addFinding("imu_samples_delta_below_threshold", "IMU sample delta is below threshold")
	}
	if report.IMUReadErrorsDelta > options.MaxReadErrors {
		report.addFinding("imu_read_error_delta_above_threshold", "IMU read error delta exceeds threshold")
	}
}
func (report *stackChanIMUProbeAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}
func writeStackChanIMUProbeAcceptanceReport(outputDir string, report stackChanIMUProbeAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-imu-probe-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanIMUProbeAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONStackChanIMUProbeAcceptance(writer io.Writer, report stackChanIMUProbeAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
