package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"a21.local/a21/internal/runtimeguard"
)

func runStackChanOfficialAudioSmokeFlash(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialSmokeFlashOptions{
		BuildDir:  firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_BUILD_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-build")),
		IDFExport: firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:      strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		OutputDir: "",
		Confirm:   "",
		Execute:   execute,
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE_FLASH_CONFIRM"))
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-official-audio-smoke-flash --build-dir /tmp/a21-stackchan-official-build --port /dev/cu.usbmodemXXXX [--execute --confirm WRITE_A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE] [--idf-export /path/to/export.sh] [--output-dir reports]")
			return 0
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
		case "--port":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--port requires a value")
				return 2
			}
			i++
			options.Port = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		case "--confirm":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--confirm requires a value")
				return 2
			}
			i++
			options.Confirm = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan official audio smoke flash option %q\n", args[i])
			return 2
		}
	}

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != stackChanOfficialAudioSmokeFlashConfirm {
			fmt.Fprintf(stderr, "stackchan official audio smoke flash requires --confirm %s\n", stackChanOfficialAudioSmokeFlashConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("stackchan-official-audio-smoke-flash --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildStackChanOfficialSmokeFlashReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official audio smoke flash: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
	}
	if execute {
		if err := executeStackChanOfficialSmokeFlash(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "flash_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official audio smoke flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialSmokeFlashReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official audio smoke flash report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialSmokeFlash(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official audio smoke flash report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func runStackChanOfficialPCMBridgeFlash(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialPCMBridgeFlashPlanOptions{
		BuildDir:   firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_BUILD_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-build")),
		IDFExport:  firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:       strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		DeviceID:   firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-001"),
		AudioWSURL: strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL")),
		OutputDir:  "",
		Confirm:    "",
		Execute:    execute,
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP_FLASH_CONFIRM"))
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-official-pcm-bridge-flash --build-dir /tmp/a21-stackchan-official-build --port /dev/cu.usbmodemXXXX --device-id stackchan-001 --audio-ws-url ws://host:21080/ws/audio?device_id=stackchan-001 [--execute --confirm WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP] [--idf-export /path/to/export.sh] [--output-dir reports]")
			return 0
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
		case "--port":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--port requires a value")
				return 2
			}
			i++
			options.Port = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--audio-ws-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--audio-ws-url requires a value")
				return 2
			}
			i++
			options.AudioWSURL = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		case "--confirm":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--confirm requires a value")
				return 2
			}
			i++
			options.Confirm = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan official pcm bridge flash option %q\n", args[i])
			return 2
		}
	}

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != stackChanOfficialPCMBridgeAppFlashConfirm {
			fmt.Fprintf(stderr, "stackchan official pcm bridge app flash requires --confirm %s\n", stackChanOfficialPCMBridgeAppFlashConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("stackchan-official-pcm-bridge-flash --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildStackChanOfficialPCMBridgeFlashPlanReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official pcm bridge flash: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
		if err := executeStackChanOfficialPCMBridgeFlash(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "flash_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official pcm bridge flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialPCMBridgeFlashPlanReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official pcm bridge flash report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialPCMBridgeFlashPlan(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official pcm bridge flash report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func runStackChanOfficialXiaozhiCompatibleFlash(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialXiaozhiCompatibleFlashOptions{
		BuildDir:              firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_BUILD_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-build")),
		IDFExport:             firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:                  strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		EsptoolBefore:         firstNonEmpty(strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_ESPTOOL_BEFORE")), "default_reset"),
		WaitROM:               appEnvBool(os.Environ(), "A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WAIT_ROM"),
		WaitROMTimeoutSeconds: parsePositiveIntOrDefault(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WAIT_ROM_TIMEOUT_SECONDS"), stackChanOfficialXiaozhiCompatibleDefaultWaitROMTimeoutSeconds),
		OutputDir:             "",
		Confirm:               "",
		Execute:               execute,
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP_FLASH_CONFIRM"))
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 a21-stackchan-official-xiaozhi-compatible-flash --build-dir /tmp/a21-stackchan-official-build --port /dev/cu.usbmodemXXXX [--execute --confirm WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP] [--idf-export /path/to/export.sh] [--esptool-before default_reset|usb_reset|no_reset] [--wait-rom --wait-rom-timeout-seconds 60] [--output-dir reports]")
			return 0
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
		case "--port":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--port requires a value")
				return 2
			}
			i++
			options.Port = args[i]
		case "--esptool-before":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--esptool-before requires a value")
				return 2
			}
			i++
			options.EsptoolBefore = args[i]
		case "--wait-rom":
			options.WaitROM = true
		case "--wait-rom-timeout-seconds":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--wait-rom-timeout-seconds")
			if !ok {
				return 2
			}
			options.WaitROMTimeoutSeconds = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		case "--confirm":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--confirm requires a value")
				return 2
			}
			i++
			options.Confirm = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan official xiaozhi compatible flash option %q\n", args[i])
			return 2
		}
	}

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != stackChanOfficialXiaozhiCompatibleAppFlashConfirm {
			fmt.Fprintf(stderr, "stackchan official xiaozhi compatible app flash requires --confirm %s\n", stackChanOfficialXiaozhiCompatibleAppFlashConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("a21-stackchan-official-xiaozhi-compatible-flash --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildStackChanOfficialXiaozhiCompatibleFlashReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official xiaozhi compatible flash: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
		if err := executeStackChanOfficialXiaozhiCompatibleFlash(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "flash_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official xiaozhi compatible flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialXiaozhiCompatibleFlashReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official xiaozhi compatible flash report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialXiaozhiCompatibleFlash(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official xiaozhi compatible flash report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func runStackChanOfficialXiaozhiCompatibleNVS(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialXiaozhiCompatibleNVSOptions{
		IDFExport:        firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		IDFPython:        strings.TrimSpace(os.Getenv("A21_IDF_PYTHON")),
		Port:             strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		RunDir:           firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_NVS_RUN_DIR"), filepath.Join(".a21-run", "firmware", "official-xiaozhi-compatible-nvs")),
		OTAURL:           strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_OTA_URL")),
		WebSocketURL:     strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WEBSOCKET_URL")),
		WiFiSSID:         strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WIFI_SSID")),
		WiFiPassword:     strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WIFI_PASSWORD")),
		WebSocketVersion: 1,
		Execute:          execute,
	}
	if version := strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WEBSOCKET_VERSION")); version != "" {
		parsed, err := strconv.Atoi(version)
		if err == nil {
			options.WebSocketVersion = parsed
		}
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_NVS_CONFIRM"))
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 a21-stackchan-official-xiaozhi-compatible-nvs --port /dev/cu.usbmodemXXXX --ota-url http://LAN:21080/xiaozhi/ota/ --websocket-url ws://LAN:21080/v1/xiaozhi [--websocket-version 1] [--wifi-ssid SSID --wifi-password PASSWORD] [--execute --confirm WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_NVS] [--idf-export /path/to/export.sh] [--idf-python /path/to/python] [--run-dir .a21-run/firmware/official-xiaozhi-compatible-nvs] [--output-dir reports]")
			return 0
		case "--idf-export":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--idf-export requires a value")
				return 2
			}
			i++
			options.IDFExport = args[i]
		case "--idf-python":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--idf-python requires a value")
				return 2
			}
			i++
			options.IDFPython = strings.TrimSpace(args[i])
		case "--port":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--port requires a value")
				return 2
			}
			i++
			options.Port = args[i]
		case "--run-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--run-dir requires a value")
				return 2
			}
			i++
			options.RunDir = args[i]
		case "--ota-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--ota-url requires a value")
				return 2
			}
			i++
			options.OTAURL = args[i]
		case "--websocket-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--websocket-url requires a value")
				return 2
			}
			i++
			options.WebSocketURL = args[i]
		case "--websocket-version":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--websocket-version requires a value")
				return 2
			}
			i++
			version, err := strconv.Atoi(args[i])
			if err != nil {
				fmt.Fprintln(stderr, "--websocket-version must be an integer")
				return 2
			}
			options.WebSocketVersion = version
		case "--wifi-ssid":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--wifi-ssid requires a value")
				return 2
			}
			i++
			options.WiFiSSID = strings.TrimSpace(args[i])
		case "--wifi-password":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--wifi-password requires a value")
				return 2
			}
			i++
			options.WiFiPassword = strings.TrimSpace(args[i])
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		case "--confirm":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--confirm requires a value")
				return 2
			}
			i++
			options.Confirm = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan official xiaozhi compatible nvs option %q\n", args[i])
			return 2
		}
	}

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != stackChanOfficialXiaozhiCompatibleNVSConfirm {
			fmt.Fprintf(stderr, "stackchan official xiaozhi compatible nvs execute requires --confirm %s\n", stackChanOfficialXiaozhiCompatibleNVSConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("a21-stackchan-official-xiaozhi-compatible-nvs --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildStackChanOfficialXiaozhiCompatibleNVSReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official xiaozhi compatible nvs: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
		if err := executeStackChanOfficialXiaozhiCompatibleNVS(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "nvs_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official xiaozhi compatible nvs report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialXiaozhiCompatibleNVSReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official xiaozhi compatible nvs report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialXiaozhiCompatibleNVS(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official xiaozhi compatible nvs report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func runStackChanOfficialPCMBridgeNVS(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialPCMBridgeNVSOptions{
		IDFExport:  firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:       strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		RunDir:     firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS_RUN_DIR"), filepath.Join(".a21-run", "firmware", "official-pcm-bridge-nvs")),
		DeviceID:   firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-001"),
		AudioWSURL: strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL")),
		Execute:    execute,
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS_CONFIRM"))
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-official-pcm-bridge-nvs --port /dev/cu.usbmodemXXXX --device-id stackchan-001 --audio-ws-url ws://host:21080/ws/audio?device_id=stackchan-001 [--execute --confirm WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS] [--idf-export /path/to/export.sh] [--run-dir .a21-run/firmware/official-pcm-bridge-nvs] [--output-dir reports]")
			return 0
		case "--idf-export":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--idf-export requires a value")
				return 2
			}
			i++
			options.IDFExport = args[i]
		case "--port":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--port requires a value")
				return 2
			}
			i++
			options.Port = args[i]
		case "--run-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--run-dir requires a value")
				return 2
			}
			i++
			options.RunDir = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--audio-ws-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--audio-ws-url requires a value")
				return 2
			}
			i++
			options.AudioWSURL = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		case "--confirm":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--confirm requires a value")
				return 2
			}
			i++
			options.Confirm = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan official pcm bridge nvs option %q\n", args[i])
			return 2
		}
	}

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != stackChanOfficialPCMBridgeNVSConfirm {
			fmt.Fprintf(stderr, "stackchan official pcm bridge nvs execute requires --confirm %s\n", stackChanOfficialPCMBridgeNVSConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("stackchan-official-pcm-bridge-nvs --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildStackChanOfficialPCMBridgeNVSReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official pcm bridge nvs: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
	}
	if execute {
		if err := executeStackChanOfficialPCMBridgeNVS(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "nvs_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official pcm bridge nvs report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialPCMBridgeNVSReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official pcm bridge nvs report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialPCMBridgeNVS(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official pcm bridge nvs report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func normalizeOfficialOverlayPaths(options *stackChanOfficialBaselineOptions) error {
	if len(options.Overlays) == 0 {
		return nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	for i, overlay := range options.Overlays {
		cleanOverlay := filepath.Clean(overlay)
		if !filepath.IsAbs(cleanOverlay) {
			cleanOverlay = filepath.Join(cwd, cleanOverlay)
		}
		options.Overlays[i] = filepath.Clean(cleanOverlay)
	}
	return nil
}
