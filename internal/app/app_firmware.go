package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/gateway"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
	return firmwarecheck.DetectPortUsage(context.Background(), port)
}

type firmwareSourceState struct {
	Root   string
	Clean  bool
	Detail string
}

var detectFirmwareSourceState = detectGitFirmwareSourceState

func detectGitFirmwareSourceState() (firmwareSourceState, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return firmwareSourceState{}, err
	}
	root := findProjectRoot(cwd)
	output, err := exec.Command("git", "-C", root, "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil {
		return firmwareSourceState{}, err
	}
	detail := strings.TrimSpace(string(output))
	return firmwareSourceState{
		Root:   root,
		Clean:  detail == "",
		Detail: detail,
	}, nil
}

type firmwareDeviceReport struct {
	SchemaVersion        string                               `json:"schema_version"`
	GatewaySchemaVersion string                               `json:"gateway_schema_version"`
	GatewayService       string                               `json:"gateway_service"`
	CapturedAtMS         int64                                `json:"captured_at_ms"`
	GatewayURL           string                               `json:"gateway_url"`
	DeviceReportPath     string                               `json:"device_report_path,omitempty"`
	Devices              []firmwarecheck.DeviceIdentityRecord `json:"devices"`
}

func runFirmwareDeviceReport(args []string, stdout io.Writer, stderr io.Writer) int {
	gatewayURL := "http://127.0.0.1:21080"
	outputDir := "reports"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-device-report --gateway-url http://127.0.0.1:21080 --output-dir reports")
			return 0
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			gatewayURL = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown firmware-device-report option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(outputDir); err != nil {
		fmt.Fprintf(stderr, "firmware device report dir invalid: %v\n", err)
		return 1
	}
	report, err := fetchFirmwareDeviceReport(gatewayURL)
	if err != nil {
		fmt.Fprintf(stderr, "firmware device report failed: %v\n", err)
		return 1
	}
	reportPath, err := writeFirmwareDeviceReport(outputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write firmware device report: %v\n", err)
		return 1
	}
	report.DeviceReportPath = reportPath
	if err := writeJSONFirmwareDeviceReport(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode firmware device report: %v\n", err)
		return 1
	}
	return 0
}
func fetchFirmwareDeviceReport(gatewayBaseURL string) (firmwareDeviceReport, error) {
	endpoint, safeGatewayURL, err := firmwareDeviceReportEndpoint(gatewayBaseURL)
	if err != nil {
		return firmwareDeviceReport{}, err
	}
	client := *a21DirectHTTPClient(3 * time.Second)
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
func validateFirmwareDeviceReportGatewayIdentity(schemaVersion string, service string) error {
	if containsLegacyIdentity(schemaVersion) || containsLegacyIdentity(service) {
		return fmt.Errorf("gateway device report contains forbidden legacy identity")
	}
	if strings.TrimSpace(schemaVersion) != gateway.DeviceRegistrySchemaVersion || strings.TrimSpace(service) != gateway.DeviceRegistryServiceName {
		return fmt.Errorf("gateway device report is missing required A21 Gateway identity")
	}
	return nil
}
func validateFirmwareDeviceReportDevices(devices []firmwarecheck.DeviceIdentityRecord) error {
	for _, device := range devices {
		values := []string{
			device.DeviceID,
			device.Firmware.ID,
			device.Firmware.Version,
			device.Firmware.Board,
			device.Firmware.Commit,
			device.LastEvent,
			device.LastTouchEvent,
			device.LastTouchSource,
			device.LastTouchTraceID,
			device.LastTouchSessionID,
			device.LastTraceID,
			device.LastSessionID,
		}
		for _, value := range values {
			if containsLegacyIdentity(value) {
				return fmt.Errorf("gateway device report contains forbidden legacy identity")
			}
		}
		for key, value := range device.Capabilities {
			if containsLegacyIdentity(key) || containsLegacyIdentity(value) {
				return fmt.Errorf("gateway device report contains forbidden legacy identity")
			}
		}
		for key, value := range device.RuntimeEcho {
			if containsLegacyIdentity(key) || containsLegacyIdentity(value) {
				return fmt.Errorf("gateway device report contains forbidden legacy identity")
			}
		}
	}
	return nil
}
func writeFirmwareDeviceReport(outputDir string, report firmwareDeviceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-devices-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.DeviceReportPath = reportPath
	if err := writeJSONFirmwareDeviceReport(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONFirmwareDeviceReport(writer io.Writer, report firmwareDeviceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
