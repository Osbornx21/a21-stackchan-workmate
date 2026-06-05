package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/gateway"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type stackChanProductRecoveryOptions struct {
	GatewayURL            string
	DeviceID              string
	UploadPort            string
	SerialGlob            string
	ReportsDir            string
	OutputDir             string
	DirectSourceIP        string
	ExecuteFlash          bool
	Confirm               string
	BuildDir              string
	IDFExport             string
	EsptoolBefore         string
	WaitROMTimeoutSeconds int
	PostCheckDelay        time.Duration
}

type stackChanProductRecoveryReport struct {
	SchemaVersion       string                                `json:"schema_version"`
	GeneratedAtMS       int64                                 `json:"generated_at_ms"`
	Status              string                                `json:"status"`
	GatewayURL          string                                `json:"gateway_url"`
	DirectSourceIP      string                                `json:"direct_source_ip,omitempty"`
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
	BuildDirName        string                                 `json:"build_dir_name,omitempty"`
	FlashLogFile        string                                 `json:"flash_log_file,omitempty"`
	FlashLogEvidence    stackChanProductRecoveryFlashLog       `json:"flash_log_evidence,omitempty"`
	Findings            []stackChanProductRecoveryFlashFinding `json:"findings,omitempty"`
}

type stackChanProductRecoveryFlashLog struct {
	Checked          bool   `json:"checked"`
	Found            bool   `json:"found"`
	Path             string `json:"path,omitempty"`
	ROMProbeTimedOut bool   `json:"rom_probe_timed_out,omitempty"`
	ROMNoSerialData  bool   `json:"rom_no_serial_data,omitempty"`
	ESP32S3Detected  bool   `json:"esp32s3_detected,omitempty"`
	LastError        string `json:"last_error,omitempty"`
	RecoveryHint     string `json:"recovery_hint,omitempty"`
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

type stackChanProductRecoveryExecutionReport struct {
	SchemaVersion string                                         `json:"schema_version"`
	GeneratedAtMS int64                                          `json:"generated_at_ms"`
	Status        string                                         `json:"status"`
	GatewayURL    string                                         `json:"gateway_url"`
	DeviceID      string                                         `json:"device_id"`
	Precheck      stackChanProductRecoveryReport                 `json:"precheck"`
	Flash         *stackChanOfficialXiaozhiCompatibleFlashReport `json:"flash,omitempty"`
	PostCheck     *stackChanProductRecoveryReport                `json:"postcheck,omitempty"`
	Findings      []stackChanProductRecoveryFinding              `json:"findings,omitempty"`
	ReportPath    string                                         `json:"report_path,omitempty"`
}

func runStackChanProductRecovery(args []string, stdout io.Writer, stderr io.Writer) int {
	options := stackChanProductRecoveryOptions{
		GatewayURL:            firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:              firstNonEmpty(strings.TrimSpace(os.Getenv("A21_PRODUCT_DEVICE_ID")), "44:1b:f6:e2:6a:60"),
		UploadPort:            firstNonEmpty(strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")), "/dev/cu.usbmodem1101"),
		SerialGlob:            "/dev/cu.usbmodem*",
		ReportsDir:            "reports",
		OutputDir:             "reports",
		DirectSourceIP:        strings.TrimSpace(os.Getenv(a21DirectSourceIPEnv)),
		BuildDir:              firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_BUILD_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-build")),
		IDFExport:             firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		EsptoolBefore:         "no_reset",
		WaitROMTimeoutSeconds: parsePositiveIntOrDefault(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WAIT_ROM_TIMEOUT_SECONDS"), stackChanOfficialXiaozhiCompatibleDefaultWaitROMTimeoutSeconds),
		PostCheckDelay:        8 * time.Second,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-accept --check product-recovery [--gateway-url http://47.103.57.217] [--device-id 44:1b:f6:e2:6a:60] [--direct-source-ip 192.168.1.20] [--upload-port /dev/cu.usbmodem1101] [--serial-glob '/dev/cu.usbmodem*'] [--reports-dir reports] [--output-dir reports] [--execute-flash --confirm WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP --build-dir /tmp/a21-stackchan-official-build --idf-export /path/to/export.sh --wait-rom-timeout-seconds 60]")
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
		case "--direct-source-ip":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--direct-source-ip requires a value")
				return 2
			}
			i++
			options.DirectSourceIP = args[i]
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
		case "--execute-flash":
			options.ExecuteFlash = true
		case "--confirm":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--confirm requires a value")
				return 2
			}
			i++
			options.Confirm = args[i]
		case "--build-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--build-dir requires a value")
				return 2
			}
			i++
			options.BuildDir = args[i]
		case "--idf-export":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--idf-export requires a value")
				return 2
			}
			i++
			options.IDFExport = args[i]
		case "--esptool-before":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--esptool-before requires a value")
				return 2
			}
			i++
			options.EsptoolBefore = args[i]
		case "--wait-rom-timeout-seconds":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--wait-rom-timeout-seconds")
			if !ok {
				return 2
			}
			options.WaitROMTimeoutSeconds = value
		case "--post-check-delay-ms":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--post-check-delay-ms")
			if !ok {
				return 2
			}
			if value > 60000 {
				fmt.Fprintln(stderr, "--post-check-delay-ms must be between 0 and 60000")
				return 2
			}
			options.PostCheckDelay = time.Duration(value) * time.Millisecond
		default:
			fmt.Fprintf(stderr, "unknown product-recovery option %q\n", args[i])
			return 2
		}
	}
	options.DirectSourceIP = strings.TrimSpace(options.DirectSourceIP)
	if options.DirectSourceIP != "" && net.ParseIP(options.DirectSourceIP) == nil {
		fmt.Fprintln(stderr, "--direct-source-ip must be a valid IP address")
		return 2
	}
	if err := validateA21ReportDir(options.ReportsDir); err != nil {
		fmt.Fprintf(stderr, "product recovery reports dir invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "product recovery output dir invalid: %v\n", err)
		return 1
	}
	if options.ExecuteFlash {
		return runStackChanProductRecoveryExecuteFlash(options, stdout, stderr)
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

func runStackChanProductRecoveryExecuteFlash(options stackChanProductRecoveryOptions, stdout io.Writer, stderr io.Writer) int {
	if options.Confirm != stackChanOfficialXiaozhiCompatibleAppFlashConfirm {
		fmt.Fprintf(stderr, "product recovery flash requires --confirm %s\n", stackChanOfficialXiaozhiCompatibleAppFlashConfirm)
		return 2
	}
	precheck := buildStackChanProductRecoveryReport(options)
	report := stackChanProductRecoveryExecutionReport{
		SchemaVersion: "a21.stackchan_product_recovery_execution.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		GatewayURL:    precheck.GatewayURL,
		DeviceID:      precheck.DeviceID,
		Precheck:      precheck,
		Status:        "precheck_completed",
	}
	if precheck.DeviceOnline {
		report.Status = "product_online_flash_skipped"
		report.Findings = append(report.Findings, stackChanProductRecoveryFinding{Code: "product_already_online", Message: "product is already online; guarded recovery flash skipped", Detail: precheck.DeviceID})
		return writeStackChanProductRecoveryExecutionAndExit(options.OutputDir, report, stdout, stderr, 0)
	}
	controlGuard, code := requireA21ControlAllowed("stackchan product recovery official-compatible flash --execute", stderr)
	if code != 0 {
		return code
	}
	flashOptions := stackChanOfficialXiaozhiCompatibleFlashOptions{
		BuildDir:              options.BuildDir,
		IDFExport:             options.IDFExport,
		Port:                  options.UploadPort,
		EsptoolBefore:         options.EsptoolBefore,
		WaitROM:               true,
		WaitROMTimeoutSeconds: options.WaitROMTimeoutSeconds,
		OutputDir:             options.OutputDir,
		Confirm:               options.Confirm,
		Execute:               true,
	}
	flashReport, err := buildStackChanOfficialXiaozhiCompatibleFlashReport(flashOptions)
	if err != nil {
		report.Status = "flash_plan_failed"
		report.Findings = append(report.Findings, stackChanProductRecoveryFinding{Code: "flash_plan_failed", Message: err.Error()})
		return writeStackChanProductRecoveryExecutionAndExit(options.OutputDir, report, stdout, stderr, 1)
	}
	flashReport.ControlGuard = &controlGuard
	if err := executeStackChanOfficialXiaozhiCompatibleFlash(context.Background(), flashOptions, &flashReport); err != nil {
		flashReport.Status = "failed"
		flashReport.Findings = append(flashReport.Findings, stackChanOfficialBaselineFinding{
			Code:    "flash_execute_failed",
			Message: err.Error(),
		})
	}
	flashReportPath, err := writeStackChanOfficialXiaozhiCompatibleFlashReport(options.OutputDir, flashReport)
	if err != nil {
		report.Status = "flash_report_write_failed"
		report.Findings = append(report.Findings, stackChanProductRecoveryFinding{Code: "flash_report_write_failed", Message: err.Error()})
		report.Flash = &flashReport
		return writeStackChanProductRecoveryExecutionAndExit(options.OutputDir, report, stdout, stderr, 1)
	}
	flashReport.ReportPath = flashReportPath
	report.Flash = &flashReport
	if flashReport.Status == "failed" || !flashReport.FlashExecuted {
		report.Status = "flash_failed_rom_download_required"
		report.Findings = append(report.Findings, stackChanProductRecoveryFinding{Code: "product_flash_failed", Message: "guarded product flash did not execute successfully", Detail: filepath.Base(flashReport.ReportPath)})
		return writeStackChanProductRecoveryExecutionAndExit(options.OutputDir, report, stdout, stderr, 1)
	}
	if options.PostCheckDelay > 0 {
		time.Sleep(options.PostCheckDelay)
	}
	postcheck := buildStackChanProductRecoveryReport(options)
	report.PostCheck = &postcheck
	report.Status = classifyStackChanProductRecoveryExecutionPostcheck(postcheck)
	return writeStackChanProductRecoveryExecutionAndExit(options.OutputDir, report, stdout, stderr, 0)
}

func classifyStackChanProductRecoveryExecutionPostcheck(postcheck stackChanProductRecoveryReport) string {
	switch {
	case postcheck.DeviceOnline && postcheck.OfficialRelay.Connected:
		return "flash_passed_product_online_official_relay_ready"
	case postcheck.DeviceOnline:
		return "flash_passed_product_online_official_relay_disconnected"
	default:
		return "flash_passed_reconnect_pending"
	}
}

func writeStackChanProductRecoveryExecutionAndExit(outputDir string, report stackChanProductRecoveryExecutionReport, stdout io.Writer, stderr io.Writer, code int) int {
	reportPath, err := writeStackChanProductRecoveryExecutionReport(outputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write product recovery execution report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanProductRecoveryExecution(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode product recovery execution report: %v\n", err)
		return 1
	}
	return code
}

func buildStackChanProductRecoveryReport(options stackChanProductRecoveryOptions) stackChanProductRecoveryReport {
	report := stackChanProductRecoveryReport{
		SchemaVersion:  "a21.stackchan_product_recovery.v1",
		GeneratedAtMS:  time.Now().UnixMilli(),
		DeviceID:       strings.TrimSpace(options.DeviceID),
		DirectSourceIP: strings.TrimSpace(options.DirectSourceIP),
		Serial:         stackChanProductRecoverySerial{UploadPort: strings.TrimSpace(options.UploadPort), SerialGlob: strings.TrimSpace(options.SerialGlob)},
	}
	_, safeGatewayURL, err := firmwareGatewayEndpoint(options.GatewayURL, "", nil)
	if err != nil {
		report.GatewayURL = ""
		report.Findings = append(report.Findings, stackChanProductRecoveryFinding{Code: "gateway_url_invalid", Message: err.Error()})
	} else {
		report.GatewayURL = safeGatewayURL
	}
	deviceReport, err := fetchStackChanProductRecoveryDeviceReport(options)
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
	report.OfficialRelay = fetchStackChanProductRecoveryOfficialStatusWithSource(options.GatewayURL, report.DeviceID, options.DirectSourceIP)
	report.Serial = inspectStackChanProductRecoverySerial(options.UploadPort, options.SerialGlob)
	latestFlash, flashFinding := latestStackChanProductRecoveryFlashReport(options.ReportsDir)
	if flashFinding.Code != "" {
		report.Findings = append(report.Findings, flashFinding)
	} else {
		report.LatestProductFlash = latestFlash
		if latestFlash.FlashLogEvidence.ROMNoSerialData {
			report.Findings = append(report.Findings, stackChanProductRecoveryFinding{Code: "rom_probe_no_serial_data", Message: "latest guarded flash log reports no ESP32-S3 serial data", Detail: latestFlash.FlashLogEvidence.LastError})
		}
	}
	report.Status, report.ROMDownloadRequired, report.NextActions = classifyStackChanProductRecovery(report)
	return report
}

func fetchStackChanProductRecoveryDeviceReport(options stackChanProductRecoveryOptions) (firmwareDeviceReport, error) {
	endpoint, safeGatewayURL, err := firmwareDeviceReportEndpoint(options.GatewayURL)
	if err != nil {
		return firmwareDeviceReport{}, err
	}
	client := *a21DirectHTTPClientWithSourceIP(3*time.Second, options.DirectSourceIP)
	resp, err := client.Get(endpoint)
	if err != nil {
		return firmwareDeviceReport{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return firmwareDeviceReport{}, fmt.Errorf("gateway device report returned status %d", resp.StatusCode)
	}
	var payload struct {
		SchemaVersion string                               `json:"schema_version"`
		Service       string                               `json:"service"`
		Devices       []firmwarecheck.DeviceIdentityRecord `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return firmwareDeviceReport{}, err
	}
	if err := validateFirmwareDeviceReportGatewayIdentity(payload.SchemaVersion, payload.Service); err != nil {
		return firmwareDeviceReport{}, err
	}
	if err := validateFirmwareDeviceReportDevices(payload.Devices); err != nil {
		return firmwareDeviceReport{}, err
	}
	return firmwareDeviceReport{
		SchemaVersion:        "a21.firmware.device_report.v1",
		GatewaySchemaVersion: payload.SchemaVersion,
		GatewayService:       payload.Service,
		CapturedAtMS:         time.Now().UnixMilli(),
		GatewayURL:           safeGatewayURL,
		Devices:              payload.Devices,
	}, nil
}

func fetchStackChanProductRecoveryOfficialStatus(gatewayURL string, deviceID string) stackChanProductRecoveryOfficialRelay {
	return fetchStackChanProductRecoveryOfficialStatusWithSource(gatewayURL, deviceID, "")
}

func fetchStackChanProductRecoveryOfficialStatusWithSource(gatewayURL string, deviceID string, directSourceIP string) stackChanProductRecoveryOfficialRelay {
	query := url.Values{"device_id": []string{deviceID}}
	endpoint, _, err := firmwareGatewayEndpoint(gatewayURL, "/v1/stackchan/official/status", query)
	if err != nil {
		return stackChanProductRecoveryOfficialRelay{Error: err.Error()}
	}
	client := *a21DirectHTTPClientWithSourceIP(3*time.Second, directSourceIP)
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
		BuildDirName        string                                 `json:"build_dir_name"`
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
		BuildDirName:        raw.BuildDirName,
		FlashLogFile:        raw.FlashLogFile,
		FlashLogEvidence:    readStackChanProductRecoveryFlashLogEvidence(path, raw.BuildDirName, raw.FlashLogFile),
		Findings:            raw.Findings,
	}, nil
}

func readStackChanProductRecoveryFlashLogEvidence(reportPath string, buildDirName string, flashLogFile string) stackChanProductRecoveryFlashLog {
	logName := strings.TrimSpace(flashLogFile)
	evidence := stackChanProductRecoveryFlashLog{Checked: logName != ""}
	if logName == "" {
		return evidence
	}
	for _, candidate := range stackChanProductRecoveryFlashLogCandidates(reportPath, buildDirName, logName) {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		evidence.Found = true
		evidence.Path = candidate
		stackChanProductRecoveryExtractFlashLogEvidence(string(data), &evidence)
		return evidence
	}
	return evidence
}

func stackChanProductRecoveryFlashLogCandidates(reportPath string, buildDirName string, flashLogFile string) []string {
	if filepath.IsAbs(flashLogFile) {
		return []string{flashLogFile}
	}
	buildDir := firstNonEmpty(strings.TrimSpace(buildDirName), "a21-stackchan-official-build")
	candidates := []string{
		filepath.Join("/tmp", buildDir, flashLogFile),
		filepath.Join(filepath.Dir(reportPath), flashLogFile),
	}
	if buildDir != "a21-stackchan-official-build" {
		candidates = append(candidates, filepath.Join("/tmp/a21-stackchan-official-build", flashLogFile))
	}
	return candidates
}

func stackChanProductRecoveryExtractFlashLogEvidence(logText string, evidence *stackChanProductRecoveryFlashLog) {
	for _, line := range strings.Split(logText, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.Contains(trimmed, "Chip is ESP32-S3") {
			evidence.ESP32S3Detected = true
		}
		if strings.Contains(trimmed, "Timed out waiting for ESP32-S3 ROM download mode") {
			evidence.ROMProbeTimedOut = true
			evidence.RecoveryHint = "hold BOOT/download, press and release RESET, keep holding BOOT until ESP32-S3 chip_id is detected"
		}
		if strings.Contains(trimmed, "Failed to connect to ESP32-S3") {
			evidence.LastError = trimmed
		}
		if strings.Contains(lower, "no serial data received") {
			evidence.ROMNoSerialData = true
			if evidence.RecoveryHint == "" {
				evidence.RecoveryHint = "enter true ESP32-S3 ROM/download mode before retrying the guarded product flash"
			}
		}
	}
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

func writeStackChanProductRecoveryExecutionReport(outputDir string, report stackChanProductRecoveryExecutionReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-product-recovery-execution-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanProductRecoveryExecution(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeJSONStackChanProductRecovery(writer io.Writer, report stackChanProductRecoveryReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanProductRecoveryExecution(writer io.Writer, report stackChanProductRecoveryExecutionReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
