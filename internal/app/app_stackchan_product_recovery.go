package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/gateway"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type stackChanProductRecoveryOptions struct {
	GatewayURL string
	DeviceID   string
	UploadPort string
	SerialGlob string
	ReportsDir string
	OutputDir  string
}

type stackChanProductRecoveryReport struct {
	SchemaVersion       string                                `json:"schema_version"`
	GeneratedAtMS       int64                                 `json:"generated_at_ms"`
	Status              string                                `json:"status"`
	GatewayURL          string                                `json:"gateway_url"`
	DeviceID            string                                `json:"device_id"`
	DeviceOnline        bool                                  `json:"device_online"`
	DeviceCount         int                                   `json:"device_count"`
	Device              *firmwarecheck.DeviceIdentityRecord   `json:"device,omitempty"`
	OfficialRelay       stackChanProductRecoveryOfficialRelay `json:"official_relay"`
	Serial              stackChanProductRecoverySerial        `json:"serial"`
	LatestProductFlash  *stackChanProductRecoveryFlashReport  `json:"latest_product_flash,omitempty"`
	ROMDownloadRequired bool                                  `json:"rom_download_required"`
	NextActions         []string                              `json:"next_actions"`
	Findings            []stackChanProductRecoveryFinding     `json:"findings,omitempty"`
	ReportPath          string                                `json:"report_path,omitempty"`
}

type stackChanProductRecoveryOfficialRelay struct {
	Checked            bool   `json:"checked"`
	Connected          bool   `json:"connected"`
	DeliveredTransport string `json:"delivered_transport,omitempty"`
	PhysicalAccepted   bool   `json:"physical_accepted"`
	NextAction         string `json:"next_action,omitempty"`
	Error              string `json:"error,omitempty"`
}

type stackChanProductRecoverySerial struct {
	UploadPort        string   `json:"upload_port"`
	UploadPortPresent bool     `json:"upload_port_present"`
	SerialGlob        string   `json:"serial_glob"`
	Candidates        []string `json:"candidates"`
}

type stackChanProductRecoveryFlashReport struct {
	Path                string                                 `json:"path"`
	Status              string                                 `json:"status"`
	FlashExecuted       bool                                   `json:"flash_executed"`
	Port                string                                 `json:"port,omitempty"`
	EsptoolBefore       string                                 `json:"esptool_before,omitempty"`
	WaitROMDownloadMode bool                                   `json:"wait_rom_download_mode"`
	WaitROMTimeoutSec   int                                    `json:"wait_rom_timeout_seconds,omitempty"`
	FlashLogFile        string                                 `json:"flash_log_file,omitempty"`
	Findings            []stackChanProductRecoveryFlashFinding `json:"findings,omitempty"`
}

type stackChanProductRecoveryFlashFinding struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type stackChanProductRecoveryFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func runStackChanProductRecovery(args []string, stdout io.Writer, stderr io.Writer) int {
	options := stackChanProductRecoveryOptions{
		GatewayURL: firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:   firstNonEmpty(strings.TrimSpace(os.Getenv("A21_PRODUCT_DEVICE_ID")), "44:1b:f6:e2:6a:60"),
		UploadPort: firstNonEmpty(strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")), "/dev/cu.usbmodem1101"),
		SerialGlob: "/dev/cu.usbmodem*",
		ReportsDir: "reports",
		OutputDir:  "reports",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-accept --check product-recovery [--gateway-url http://47.103.57.217] [--device-id 44:1b:f6:e2:6a:60] [--upload-port /dev/cu.usbmodem1101] [--serial-glob '/dev/cu.usbmodem*'] [--reports-dir reports] [--output-dir reports]")
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
		case "--upload-port":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--upload-port requires a value")
				return 2
			}
			i++
			options.UploadPort = args[i]
		case "--serial-glob":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--serial-glob requires a value")
				return 2
			}
			i++
			options.SerialGlob = args[i]
		case "--reports-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--reports-dir requires a value")
				return 2
			}
			i++
			options.ReportsDir = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown product-recovery option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(options.ReportsDir); err != nil {
		fmt.Fprintf(stderr, "product recovery reports dir invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "product recovery output dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanProductRecoveryReport(options)
	reportPath, err := writeStackChanProductRecoveryReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write product recovery report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanProductRecovery(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode product recovery report: %v\n", err)
		return 1
	}
	return 0
}

func buildStackChanProductRecoveryReport(options stackChanProductRecoveryOptions) stackChanProductRecoveryReport {
	report := stackChanProductRecoveryReport{
		SchemaVersion: "a21.stackchan_product_recovery.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		DeviceID:      strings.TrimSpace(options.DeviceID),
		Serial:        stackChanProductRecoverySerial{UploadPort: strings.TrimSpace(options.UploadPort), SerialGlob: strings.TrimSpace(options.SerialGlob)},
	}
	_, safeGatewayURL, err := firmwareGatewayEndpoint(options.GatewayURL, "", nil)
	if err != nil {
		report.GatewayURL = ""
		report.Findings = append(report.Findings, stackChanProductRecoveryFinding{Code: "gateway_url_invalid", Message: err.Error()})
	} else {
		report.GatewayURL = safeGatewayURL
	}
	deviceReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.Findings = append(report.Findings, stackChanProductRecoveryFinding{Code: "gateway_devices_unavailable", Message: err.Error()})
	} else {
		report.DeviceCount = len(deviceReport.Devices)
		for _, device := range deviceReport.Devices {
			if strings.EqualFold(device.DeviceID, report.DeviceID) {
				deviceCopy := device
				report.Device = &deviceCopy
				report.DeviceOnline = strings.TrimSpace(device.ConnectionStatus) == "online"
				break
			}
		}
		if report.Device == nil {
			report.Findings = append(report.Findings, stackChanProductRecoveryFinding{Code: "product_device_not_online", Message: "product device is absent from Gateway registry", Detail: report.DeviceID})
		}
	}
	report.OfficialRelay = fetchStackChanProductRecoveryOfficialStatus(options.GatewayURL, report.DeviceID)
	report.Serial = inspectStackChanProductRecoverySerial(options.UploadPort, options.SerialGlob)
	latestFlash, flashFinding := latestStackChanProductRecoveryFlashReport(options.ReportsDir)
	if flashFinding.Code != "" {
		report.Findings = append(report.Findings, flashFinding)
	} else {
		report.LatestProductFlash = latestFlash
	}
	report.Status, report.ROMDownloadRequired, report.NextActions = classifyStackChanProductRecovery(report)
	return report
}

func fetchStackChanProductRecoveryOfficialStatus(gatewayURL string, deviceID string) stackChanProductRecoveryOfficialRelay {
	query := url.Values{"device_id": []string{deviceID}}
	endpoint, _, err := firmwareGatewayEndpoint(gatewayURL, "/v1/stackchan/official/status", query)
	if err != nil {
		return stackChanProductRecoveryOfficialRelay{Error: err.Error()}
	}
	client := *a21DirectHTTPClient(3 * time.Second)
	resp, err := client.Get(endpoint)
	if err != nil {
		return stackChanProductRecoveryOfficialRelay{Error: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return stackChanProductRecoveryOfficialRelay{Error: fmt.Sprintf("official status returned status %d", resp.StatusCode)}
	}
	var payload gateway.OfficialStackChanStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return stackChanProductRecoveryOfficialRelay{Error: err.Error()}
	}
	return stackChanProductRecoveryOfficialRelay{
		Checked:            true,
		Connected:          payload.Connected,
		DeliveredTransport: payload.DeliveredTransport,
		PhysicalAccepted:   payload.PhysicalAccepted,
		NextAction:         payload.NextAction,
	}
}

func inspectStackChanProductRecoverySerial(uploadPort string, serialGlob string) stackChanProductRecoverySerial {
	report := stackChanProductRecoverySerial{UploadPort: strings.TrimSpace(uploadPort), SerialGlob: strings.TrimSpace(serialGlob)}
	if report.SerialGlob == "" {
		report.SerialGlob = "/dev/cu.usbmodem*"
	}
	matches, err := filepath.Glob(report.SerialGlob)
	if err == nil {
		report.Candidates = matches
	}
	if report.UploadPort != "" {
		if _, err := os.Stat(report.UploadPort); err == nil {
			report.UploadPortPresent = true
		}
	}
	return report
}

func latestStackChanProductRecoveryFlashReport(reportsDir string) (*stackChanProductRecoveryFlashReport, stackChanProductRecoveryFinding) {
	for _, path := range latestProductReadinessReportCandidates(firstNonEmpty(strings.TrimSpace(reportsDir), "reports"), []string{"a21-stackchan-official-xiaozhi-compatible-flash-*.json"}) {
		report, err := readStackChanProductRecoveryFlashReport(path)
		if err != nil {
			return nil, stackChanProductRecoveryFinding{Code: "latest_product_flash_report_invalid", Message: err.Error(), Detail: filepath.Base(path)}
		}
		report.Path = path
		return &report, stackChanProductRecoveryFinding{}
	}
	return nil, stackChanProductRecoveryFinding{Code: "latest_product_flash_report_missing", Message: "no official Xiaozhi-compatible product flash report found"}
}

func readStackChanProductRecoveryFlashReport(path string) (stackChanProductRecoveryFlashReport, error) {
	file, err := os.Open(path)
	if err != nil {
		return stackChanProductRecoveryFlashReport{}, err
	}
	defer file.Close()
	var raw struct {
		Status              string                                 `json:"status"`
		FlashExecuted       bool                                   `json:"flash_executed"`
		Port                string                                 `json:"port"`
		EsptoolBefore       string                                 `json:"esptool_before"`
		WaitROMDownloadMode bool                                   `json:"wait_rom_download_mode"`
		WaitROMTimeoutSec   int                                    `json:"wait_rom_timeout_seconds"`
		FlashLogFile        string                                 `json:"flash_log_file"`
		Findings            []stackChanProductRecoveryFlashFinding `json:"findings"`
	}
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return stackChanProductRecoveryFlashReport{}, err
	}
	return stackChanProductRecoveryFlashReport{
		Status:              raw.Status,
		FlashExecuted:       raw.FlashExecuted,
		Port:                raw.Port,
		EsptoolBefore:       raw.EsptoolBefore,
		WaitROMDownloadMode: raw.WaitROMDownloadMode,
		WaitROMTimeoutSec:   raw.WaitROMTimeoutSec,
		FlashLogFile:        raw.FlashLogFile,
		Findings:            raw.Findings,
	}, nil
}

func classifyStackChanProductRecovery(report stackChanProductRecoveryReport) (string, bool, []string) {
	switch {
	case report.DeviceOnline && report.OfficialRelay.Connected:
		return "product_online_official_relay_ready", false, []string{"collect_physical_power_wake_voice_body_acceptance"}
	case report.DeviceOnline:
		return "product_online_official_relay_disconnected", false, []string{"inspect_product_official_stackchan_ws_relay", "use_status_endpoint_before_official_actions"}
	case !report.Serial.UploadPortPresent && len(report.Serial.Candidates) == 0:
		return "product_offline_serial_missing", false, []string{"connect_product_usb_or_power", "recheck_product_recovery_status"}
	case report.LatestProductFlash != nil && !report.LatestProductFlash.FlashExecuted:
		return "product_offline_rom_download_required", true, []string{"enter_esp32s3_rom_download_mode", "rerun_guarded_wait_rom_product_flash", "verify_public_devices_and_official_status"}
	default:
		return "product_offline_recovery_required", false, []string{"recheck_usb_serial_and_latest_flash_report", "rerun_guarded_wait_rom_product_flash_when_rom_ready"}
	}
}

func writeStackChanProductRecoveryReport(outputDir string, report stackChanProductRecoveryReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-product-recovery-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanProductRecovery(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeJSONStackChanProductRecovery(writer io.Writer, report stackChanProductRecoveryReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
