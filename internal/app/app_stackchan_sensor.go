package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/firmwarecheck"
)

type stackChanSensorProbeAcceptanceOptions struct {
	GatewayURL    string
	DeviceID      string
	Commit        string
	WindowMS      int
	MinSamples    int
	MinBatteryMV  int
	MaxReadErrors int
	OutputDir     string
}
type stackChanSensorProbeAcceptanceThresholds struct {
	MinSamples    int `json:"min_samples"`
	MinBatteryMV  int `json:"min_battery_mv"`
	MaxReadErrors int `json:"max_read_errors"`
}
type stackChanSensorProbeAcceptanceReport struct {
	SchemaVersion                string                                   `json:"schema_version"`
	GeneratedAtMS                int64                                    `json:"generated_at_ms"`
	Metadata                     latencyBenchMetadata                     `json:"metadata"`
	DryRun                       bool                                     `json:"dry_run"`
	FlashAllowed                 bool                                     `json:"flash_allowed"`
	DeleteAllowed                bool                                     `json:"delete_allowed"`
	HardwareAcceptanceScope      string                                   `json:"hardware_acceptance_scope"`
	SensorProbeAcceptanceStatus  string                                   `json:"sensor_probe_acceptance_status"`
	ProductionCapabilityPromoted bool                                     `json:"production_capability_promoted"`
	GatewayURL                   string                                   `json:"gateway_url"`
	DeviceID                     string                                   `json:"device_id"`
	Commit                       string                                   `json:"commit"`
	WindowMS                     int                                      `json:"window_ms"`
	WindowStartedAtMS            int64                                    `json:"window_started_at_ms,omitempty"`
	WindowEndedAtMS              int64                                    `json:"window_ended_at_ms,omitempty"`
	Firmware                     firmwarecheck.DeviceIdentityFirmware     `json:"firmware"`
	Capabilities                 map[string]string                        `json:"capabilities,omitempty"`
	AmbientLight                 string                                   `json:"ambient_light,omitempty"`
	Proximity                    string                                   `json:"proximity,omitempty"`
	Battery                      string                                   `json:"battery,omitempty"`
	RuntimeEcho                  map[string]string                        `json:"runtime_echo,omitempty"`
	RuntimeEchoBefore            map[string]string                        `json:"runtime_echo_before,omitempty"`
	SensorAvailable              int                                      `json:"sensor_available"`
	SensorSamples                int                                      `json:"sensor_samples"`
	SensorReadErrors             int                                      `json:"sensor_read_errors"`
	AmbientLightRaw              int                                      `json:"ambient_light_raw"`
	ProximityRaw                 int                                      `json:"proximity_raw"`
	BatteryMV                    int                                      `json:"battery_mv"`
	BatteryMA                    int                                      `json:"battery_ma"`
	SensorSamplesDelta           int                                      `json:"sensor_samples_delta"`
	SensorReadErrorsDelta        int                                      `json:"sensor_read_errors_delta"`
	SampleRateHz                 float64                                  `json:"sample_rate_hz,omitempty"`
	Thresholds                   stackChanSensorProbeAcceptanceThresholds `json:"thresholds"`
	NextRequiredActions          []string                                 `json:"next_required_actions"`
	ReportPath                   string                                   `json:"report_path,omitempty"`
	Findings                     []officePreflightFinding                 `json:"findings,omitempty"`
}

func runStackChanSensorProbeAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanSensorProbeAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-accept --check sensor-probe --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> [--window-ms 1500] [--min-samples 10] [--min-battery-mv 3000] [--max-read-errors 0] [--output-dir reports]")
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
		case "--min-battery-mv":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-battery-mv")
			if !ok {
				return 2
			}
			options.MinBatteryMV = value
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
			fmt.Fprintf(stderr, "unknown stackchan-sensor-probe-acceptance option %q\n", args[i])
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
		fmt.Fprintf(stderr, "stackchan sensor probe acceptance gateway URL invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan sensor probe acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanSensorProbeAcceptanceReport(options)
	reportPath, err := writeStackChanSensorProbeAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan sensor probe acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanSensorProbeAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan sensor probe acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan sensor probe acceptance blocked (diagnostic only, no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan sensor probe acceptance ok (diagnostic only, no flash performed)")
	return 0
}
func defaultStackChanSensorProbeAcceptanceOptions() stackChanSensorProbeAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanSensorProbeAcceptanceOptions{
		GatewayURL:    firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:      deviceID,
		Commit:        currentGitCommit(projectRoot),
		WindowMS:      1500,
		MinSamples:    10,
		MinBatteryMV:  3000,
		MaxReadErrors: 0,
		OutputDir:     "reports",
	}
}
func buildStackChanSensorProbeAcceptanceReport(options stackChanSensorProbeAcceptanceOptions) stackChanSensorProbeAcceptanceReport {
	report := stackChanSensorProbeAcceptanceReport{
		SchemaVersion:                "a21.stackchan_sensor_probe_acceptance.v1",
		GeneratedAtMS:                time.Now().UnixMilli(),
		Metadata:                     buildLatencyBenchMetadata(),
		DryRun:                       true,
		FlashAllowed:                 false,
		DeleteAllowed:                false,
		HardwareAcceptanceScope:      "diagnostic_sensor_only",
		SensorProbeAcceptanceStatus:  "confirmed",
		ProductionCapabilityPromoted: false,
		GatewayURL:                   sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                     options.DeviceID,
		Commit:                       options.Commit,
		WindowMS:                     options.WindowMS,
		Thresholds: stackChanSensorProbeAcceptanceThresholds{
			MinSamples:    options.MinSamples,
			MinBatteryMV:  options.MinBatteryMV,
			MaxReadErrors: options.MaxReadErrors,
		},
		NextRequiredActions: []string{
			"Keep this report with the sensor-probe flash receipt and Gateway device report.",
			"Treat ambient/proximity/battery telemetry as read-only diagnostic evidence, not product behavior acceptance.",
			"Promote adaptive brightness, presence behavior, or power-state UI only through a later ADR and release-firmware acceptance gate.",
		},
	}

	beforeGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_before_failed", err.Error())
		report.SensorProbeAcceptanceStatus = "blocked"
		return report
	}
	beforeDevice, ok := findFirmwareDeviceRecord(beforeGatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_before_missing", "expected StackChan device is missing from Gateway before sensor probe")
		report.SensorProbeAcceptanceStatus = "blocked"
		return report
	}
	validateStackChanSensorProbeIdentity(&report, options, beforeDevice)
	report.RuntimeEchoBefore = beforeDevice.RuntimeEcho
	beforeSamples := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "sensor_samples")
	beforeReadErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "sensor_read_errors")
	if len(report.Findings) > 0 {
		report.SensorProbeAcceptanceStatus = "blocked"
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
			report.addFinding("gateway_device_after_missing", "expected StackChan device is missing from Gateway after sensor probe")
		} else {
			validateStackChanSensorProbeDevice(&report, options, afterDevice)
			report.SensorSamplesDelta = report.SensorSamples - beforeSamples
			report.SensorReadErrorsDelta = report.SensorReadErrors - beforeReadErrors
			if report.WindowEndedAtMS > report.WindowStartedAtMS {
				report.SampleRateHz = roundedStackChanDiagnosticValue(float64(report.SensorSamplesDelta) * 1000 / float64(report.WindowEndedAtMS-report.WindowStartedAtMS))
			}
			validateStackChanSensorProbeDeltas(&report, options)
		}
	}
	if len(report.Findings) > 0 {
		report.SensorProbeAcceptanceStatus = "blocked"
	}
	return report
}
func validateStackChanSensorProbeIdentity(report *stackChanSensorProbeAcceptanceReport, options stackChanSensorProbeAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	report.Firmware = device.Firmware
	report.Capabilities = device.Capabilities
	report.AmbientLight = device.Capabilities["ambient_light"]
	report.Proximity = device.Capabilities["proximity"]
	report.Battery = device.Capabilities["battery"]
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
	if report.AmbientLight != firmwarecheck.SensorProbeAmbientLightCapabilityStatus {
		report.addFinding("ambient_light_not_diagnostic_probe", "device ambient light capability is not the diagnostic sensor-probe status")
	}
	if report.Proximity != firmwarecheck.SensorProbeProximityCapabilityStatus {
		report.addFinding("proximity_not_diagnostic_probe", "device proximity capability is not the diagnostic sensor-probe status")
	}
	if report.Battery != firmwarecheck.SensorProbeBatteryCapabilityStatus {
		report.addFinding("battery_not_diagnostic_probe", "device battery capability is not the diagnostic sensor-probe status")
	}
}
func validateStackChanSensorProbeDevice(report *stackChanSensorProbeAcceptanceReport, options stackChanSensorProbeAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	validateStackChanSensorProbeIdentity(report, options, device)
	report.RuntimeEcho = device.RuntimeEcho
	report.SensorAvailable = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "sensor_available")
	report.SensorSamples = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "sensor_samples")
	report.SensorReadErrors = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "sensor_read_errors")
	report.AmbientLightRaw = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "ambient_light_raw")
	report.ProximityRaw = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "proximity_raw")
	report.BatteryMV = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "battery_mv")
	report.BatteryMA = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "battery_ma")
	if report.SensorAvailable != 1 {
		report.addFinding("sensor_not_available", "sensor diagnostic runtime did not report availability")
	}
	if report.SensorReadErrors > options.MaxReadErrors {
		report.addFinding("sensor_read_errors_above_threshold", "sensor read errors exceed threshold")
	}
	if report.BatteryMV < options.MinBatteryMV {
		report.addFinding("battery_mv_below_threshold", "battery voltage evidence is below threshold")
	}
}
func validateStackChanSensorProbeDeltas(report *stackChanSensorProbeAcceptanceReport, options stackChanSensorProbeAcceptanceOptions) {
	if report.SensorSamplesDelta < options.MinSamples {
		report.addFinding("sensor_samples_delta_below_threshold", "sensor sample delta is below threshold")
	}
	if report.SensorReadErrorsDelta > options.MaxReadErrors {
		report.addFinding("sensor_read_error_delta_above_threshold", "sensor read error delta exceeds threshold")
	}
}
func (report *stackChanSensorProbeAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}
func writeStackChanSensorProbeAcceptanceReport(outputDir string, report stackChanSensorProbeAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-sensor-probe-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanSensorProbeAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONStackChanSensorProbeAcceptance(writer io.Writer, report stackChanSensorProbeAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
