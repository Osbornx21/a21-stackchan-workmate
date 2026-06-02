package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/runtimeguard"
)

const xiaozhiFirmwareFlashPlanSchema = "a21.xiaozhi_firmware_flash_plan.v1"
const xiaozhiFirmwareFlashExecutionSchema = "a21.xiaozhi_firmware_flash_execution.v1"
const xiaozhiFirmwareFlashConfirm = "WRITE_A21_XIAOZHI_FIRMWARE"

var runXiaozhiFirmwareFlashCommand = runStackChanOfficialSmokeFlashCommandExec

type xiaozhiFirmwareFlashOptions struct {
	BuildDir  string
	IDFExport string
	Port      string
	OutputDir string
	Confirm   string
	Execute   bool
}

type xiaozhiFirmwareFlashReport struct {
	SchemaVersion            string                             `json:"schema_version"`
	GeneratedAtMS            int64                              `json:"generated_at_ms"`
	Status                   string                             `json:"status"`
	DryRun                   bool                               `json:"dry_run"`
	FlashAllowed             bool                               `json:"flash_allowed"`
	FlashExecuted            bool                               `json:"flash_executed"`
	ControlGuard             *runtimeguard.ControlGuardReport   `json:"control_guard,omitempty"`
	Port                     string                             `json:"port"`
	BuildDirName             string                             `json:"build_dir_name"`
	IDFExportConfigured      bool                               `json:"idf_export_configured"`
	OTA                      xiaozhiFirmwareOTAConfig           `json:"ota"`
	FlashLogFile             string                             `json:"flash_log_file,omitempty"`
	NextRequiredConfirmation string                             `json:"next_required_confirmation,omitempty"`
	Parts                    []xiaozhiFirmwareFlashPart         `json:"parts"`
	Findings                 []stackChanOfficialBaselineFinding `json:"findings,omitempty"`
	ReportPath               string                             `json:"report_path,omitempty"`
}

type xiaozhiFirmwareOTAConfig struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	Path   string `json:"path"`
}

type xiaozhiFirmwareFlashPart struct {
	Name      string `json:"name"`
	Offset    string `json:"offset"`
	File      string `json:"file"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

func runXiaozhiFirmwareFlash(args []string, stdout io.Writer, stderr io.Writer) int {
	clean, execute := splitExecuteFlag(args)
	options := xiaozhiFirmwareFlashOptions{
		BuildDir:  strings.TrimSpace(os.Getenv("A21_XIAOZHI_FIRMWARE_BUILD_DIR")),
		IDFExport: firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:      strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		OutputDir: "",
		Execute:   execute,
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_XIAOZHI_FIRMWARE_FLASH_CONFIRM"))
	}
	for i := 0; i < len(clean); i++ {
		switch clean[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 xiaozhi-firmware-flash --build-dir /path/to/xiaozhi/build-m5stack-core-s3 --port /dev/cu.usbmodemXXXX [--execute --confirm WRITE_A21_XIAOZHI_FIRMWARE] [--idf-export /path/to/export.sh] [--output-dir reports]")
			return 0
		case "--build-dir":
			if i+1 >= len(clean) || strings.HasPrefix(clean[i+1], "-") {
				fmt.Fprintln(stderr, "--build-dir requires a value")
				return 2
			}
			i++
			options.BuildDir = clean[i]
		case "--idf-export":
			if i+1 >= len(clean) || strings.HasPrefix(clean[i+1], "-") {
				fmt.Fprintln(stderr, "--idf-export requires a value")
				return 2
			}
			i++
			options.IDFExport = clean[i]
		case "--port":
			if i+1 >= len(clean) || strings.HasPrefix(clean[i+1], "-") {
				fmt.Fprintln(stderr, "--port requires a value")
				return 2
			}
			i++
			options.Port = clean[i]
		case "--output-dir":
			if i+1 >= len(clean) || strings.HasPrefix(clean[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = clean[i]
		case "--confirm":
			if i+1 >= len(clean) || strings.HasPrefix(clean[i+1], "-") {
				fmt.Fprintln(stderr, "--confirm requires a value")
				return 2
			}
			i++
			options.Confirm = clean[i]
		default:
			fmt.Fprintf(stderr, "unknown xiaozhi firmware flash option %q\n", clean[i])
			return 2
		}
	}

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != xiaozhiFirmwareFlashConfirm {
			fmt.Fprintf(stderr, "xiaozhi firmware flash execute requires --confirm %s\n", xiaozhiFirmwareFlashConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("xiaozhi-firmware-flash --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildXiaozhiFirmwareFlashReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "xiaozhi firmware flash: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
		if err := executeXiaozhiFirmwareFlash(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "flash_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "xiaozhi firmware flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeXiaozhiFirmwareFlashReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write xiaozhi firmware flash report: %v\n", err)
			return 1
		}
		report.ReportPath = filepath.Base(reportPath)
	}
	if err := writeJSONXiaozhiFirmwareFlash(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode xiaozhi firmware flash report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func buildXiaozhiFirmwareFlashReport(options xiaozhiFirmwareFlashOptions) (xiaozhiFirmwareFlashReport, error) {
	buildDir := filepath.Clean(strings.TrimSpace(options.BuildDir))
	if buildDir == "." || buildDir == "" {
		return xiaozhiFirmwareFlashReport{}, fmt.Errorf("--build-dir is required")
	}
	if err := validateNotFrozenExternalXiaozhiFirmwareBuildDir(buildDir); err != nil {
		return xiaozhiFirmwareFlashReport{}, err
	}
	if containsLegacyIdentityPathToken(buildDir) {
		return xiaozhiFirmwareFlashReport{}, fmt.Errorf("build dir contains forbidden legacy identity")
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return xiaozhiFirmwareFlashReport{}, err
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return xiaozhiFirmwareFlashReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return xiaozhiFirmwareFlashReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return xiaozhiFirmwareFlashReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}
	ota, err := inspectXiaozhiFirmwareOTAConfig(buildDir)
	if err != nil {
		return xiaozhiFirmwareFlashReport{}, err
	}
	parts, err := collectXiaozhiFirmwareFlashParts(buildDir)
	if err != nil {
		return xiaozhiFirmwareFlashReport{}, err
	}
	schema := xiaozhiFirmwareFlashPlanSchema
	if options.Execute {
		schema = xiaozhiFirmwareFlashExecutionSchema
	}
	return xiaozhiFirmwareFlashReport{
		SchemaVersion:            schema,
		GeneratedAtMS:            time.Now().UnixMilli(),
		Status:                   "ready",
		DryRun:                   !options.Execute,
		FlashAllowed:             false,
		FlashExecuted:            false,
		Port:                     options.Port,
		BuildDirName:             filepath.Base(buildDir),
		IDFExportConfigured:      strings.TrimSpace(options.IDFExport) != "",
		OTA:                      ota,
		NextRequiredConfirmation: "xiaozhi-firmware-flash-execute_with_confirmation_token",
		Parts:                    parts,
	}, nil
}

func inspectXiaozhiFirmwareOTAConfig(buildDir string) (xiaozhiFirmwareOTAConfig, error) {
	data, err := os.ReadFile(filepath.Join(buildDir, "config", "sdkconfig.json"))
	if err != nil {
		return xiaozhiFirmwareOTAConfig{}, fmt.Errorf("sdkconfig.json is required for board and OTA verification: %w", err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return xiaozhiFirmwareOTAConfig{}, fmt.Errorf("parse sdkconfig.json: %w", err)
	}
	if !configBool(config, "BOARD_TYPE_M5STACK_CORE_S3") {
		return xiaozhiFirmwareOTAConfig{}, fmt.Errorf("xiaozhi firmware must be built for M5Stack CoreS3")
	}
	for key, value := range config {
		if containsLegacyIdentity(key) && configValueEnabled(value) {
			return xiaozhiFirmwareOTAConfig{}, fmt.Errorf("xiaozhi firmware config contains enabled forbidden legacy identity")
		}
	}
	rawOTA := strings.TrimSpace(configString(config, "OTA_URL"))
	if rawOTA == "" {
		return xiaozhiFirmwareOTAConfig{}, fmt.Errorf("CONFIG_OTA_URL is required")
	}
	parsed, err := url.Parse(rawOTA)
	if err != nil {
		return xiaozhiFirmwareOTAConfig{}, fmt.Errorf("parse CONFIG_OTA_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return xiaozhiFirmwareOTAConfig{}, fmt.Errorf("CONFIG_OTA_URL must use http or https")
	}
	if parsed.User != nil || parsed.Host == "" {
		return xiaozhiFirmwareOTAConfig{}, fmt.Errorf("CONFIG_OTA_URL must not contain credentials and must include host")
	}
	if parsed.Path != "/xiaozhi/ota/" && parsed.Path != "/xiaozhi/ota" {
		return xiaozhiFirmwareOTAConfig{}, fmt.Errorf("CONFIG_OTA_URL path must be /xiaozhi/ota/")
	}
	if isLoopbackOrUnspecifiedHost(parsed.Hostname()) {
		return xiaozhiFirmwareOTAConfig{}, fmt.Errorf("CONFIG_OTA_URL host must be reachable by the physical device")
	}
	return xiaozhiFirmwareOTAConfig{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
		Path:   parsed.Path,
	}, nil
}

func collectXiaozhiFirmwareFlashParts(buildDir string) ([]xiaozhiFirmwareFlashPart, error) {
	entries := readOfficialFlashArgsEntries(buildDir)
	if len(entries) == 0 {
		return nil, fmt.Errorf("xiaozhi flash_args missing or empty")
	}
	required := map[string]string{
		"0x0":      "bootloader",
		"0x8000":   "partition_table",
		"0xd000":   "ota_data_initial",
		"0x20000":  "app",
		"0x800000": "assets",
	}
	seenOffsets := make(map[string]bool)
	parts := make([]xiaozhiFirmwareFlashPart, 0, len(entries))
	for _, entry := range entries {
		name, ok := required[entry.offset]
		if !ok {
			continue
		}
		fullPath := filepath.Join(buildDir, filepath.FromSlash(entry.path))
		if containsLegacyIdentityPathToken(fullPath) {
			return nil, fmt.Errorf("flash part path contains forbidden legacy identity")
		}
		if name == "app" && filepath.Base(fullPath) != "xiaozhi.bin" {
			return nil, fmt.Errorf("xiaozhi app must be xiaozhi.bin")
		}
		stat, err := os.Stat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("flash part %s missing: %w", name, err)
		}
		sum, err := sha256File(fullPath)
		if err != nil {
			return nil, fmt.Errorf("hash flash part %s: %w", name, err)
		}
		seenOffsets[entry.offset] = true
		parts = append(parts, xiaozhiFirmwareFlashPart{
			Name:      name,
			Offset:    entry.offset,
			File:      filepath.Base(fullPath),
			SHA256:    sum,
			SizeBytes: stat.Size(),
		})
	}
	for offset, name := range required {
		if !seenOffsets[offset] {
			return nil, fmt.Errorf("flash_args missing required %s at %s", name, offset)
		}
	}
	return parts, nil
}

func executeXiaozhiFirmwareFlash(ctx context.Context, options xiaozhiFirmwareFlashOptions, report *xiaozhiFirmwareFlashReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	report.DryRun = false
	report.FlashAllowed = true
	report.NextRequiredConfirmation = ""
	logPath := filepath.Join(filepath.Clean(options.BuildDir), fmt.Sprintf("a21-xiaozhi-firmware-flash-%s.log", time.Now().Format("20060102-150405")))
	report.FlashLogFile = filepath.Base(logPath)
	script := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("cd %s", shellSingleQuote(filepath.Clean(options.BuildDir))),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after hard_reset write_flash @flash_args", shellSingleQuote(options.Port)),
	}, "\n")
	if err := runXiaozhiFirmwareFlashCommand(ctx, logPath, script); err != nil {
		return err
	}
	report.FlashExecuted = true
	report.Status = "passed"
	return nil
}

func configBool(config map[string]any, key string) bool {
	value, ok := config[key]
	if !ok {
		return false
	}
	typed, ok := value.(bool)
	return ok && typed
}

func configString(config map[string]any, key string) string {
	value, ok := config[key]
	if !ok {
		return ""
	}
	typed, ok := value.(string)
	if !ok {
		return ""
	}
	return typed
}

func configValueEnabled(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.TrimSpace(typed) != ""
	case float64:
		return typed != 0
	default:
		return false
	}
}

func isLoopbackOrUnspecifiedHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsUnspecified())
}

func writeXiaozhiFirmwareFlashReport(outputDir string, report xiaozhiFirmwareFlashReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-xiaozhi-firmware-flash-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONXiaozhiFirmwareFlash(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeJSONXiaozhiFirmwareFlash(writer io.Writer, report xiaozhiFirmwareFlashReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
