package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func executeStackChanOfficialPCMBridgeFlash(ctx context.Context, options stackChanOfficialPCMBridgeFlashPlanOptions, report *stackChanOfficialPCMBridgeFlashPlanReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	report.DryRun = false
	report.FlashAllowed = true
	report.NextRequiredConfirmation = ""
	report.FlashLogPath = filepath.Join(options.BuildDir, fmt.Sprintf("a21-official-pcm-bridge-flash-%s.log", time.Now().Format("20060102-150405")))
	script := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("cd %s", shellSingleQuote(options.BuildDir)),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after hard_reset write_flash @flash_args", shellSingleQuote(options.Port)),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeFlashCommand(ctx, report.FlashLogPath, script); err != nil {
		return err
	}
	report.FlashExecuted = true
	report.Status = "passed"
	return nil
}

func executeStackChanOfficialXiaozhiCompatibleFlash(ctx context.Context, options stackChanOfficialXiaozhiCompatibleFlashOptions, report *stackChanOfficialXiaozhiCompatibleFlashReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	report.DryRun = false
	report.FlashAllowed = true
	report.NextRequiredConfirmation = ""
	flashLogFile := fmt.Sprintf("a21-official-xiaozhi-compatible-flash-%s.log", time.Now().Format("20060102-150405"))
	flashLogPath := filepath.Join(options.BuildDir, flashLogFile)
	report.FlashLogFile = flashLogFile
	esptoolBefore := report.EsptoolBefore
	if esptoolBefore == "" {
		var err error
		esptoolBefore, err = validateStackChanOfficialXiaozhiCompatibleEsptoolBefore(options.EsptoolBefore)
		if err != nil {
			return err
		}
	}
	scriptLines := []string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("cd %s", shellSingleQuote(options.BuildDir)),
	}
	flashPortArg := shellSingleQuote(options.Port)
	if report.WaitROM {
		waitTimeout := report.WaitROMTimeoutSeconds
		if waitTimeout <= 0 {
			waitTimeout = stackChanOfficialXiaozhiCompatibleDefaultWaitROMTimeoutSeconds
		}
		flashPortArg = `"${A21_FLASH_PORT}"`
		scriptLines = append(scriptLines,
			fmt.Sprintf("A21_FLASH_PORT=%s", shellSingleQuote(options.Port)),
			`A21_WAIT_ROM_PROBE_LOG="$(mktemp -t a21-wait-rom-probe.XXXXXX)"`,
			`trap 'rm -f "${A21_WAIT_ROM_PROBE_LOG:-}"' EXIT`,
			fmt.Sprintf("A21_WAIT_ROM_DEADLINE=$((SECONDS + %d))", waitTimeout),
			fmt.Sprintf("echo %s", shellSingleQuote("Waiting for ESP32-S3 ROM download mode on "+options.Port)),
			"A21_WAIT_ROM_ATTEMPT=0",
			"while true; do",
			"  A21_WAIT_ROM_ATTEMPT=$((A21_WAIT_ROM_ATTEMPT + 1))",
			`  A21_WAIT_ROM_CANDIDATES=("${A21_FLASH_PORT}")`,
			"  for candidate in /dev/cu.usbmodem*; do",
			`    [ -e "$candidate" ] || continue`,
			`    [ "$candidate" = "$A21_FLASH_PORT" ] || A21_WAIT_ROM_CANDIDATES+=("$candidate")`,
			"  done",
			`  for candidate in "${A21_WAIT_ROM_CANDIDATES[@]}"; do`,
			`    [ -e "$candidate" ] || continue`,
			`    if python -m esptool --chip esp32s3 --port "$candidate" -b 115200 --before no_reset --after no_reset --no-stub chip_id >"${A21_WAIT_ROM_PROBE_LOG}" 2>&1; then`,
			`      A21_FLASH_PORT="$candidate"`,
			`      echo "ESP32-S3 ROM download mode detected on ${A21_FLASH_PORT}; starting guarded product app flash."`,
			"      break 2",
			"    fi",
			"  done",
			"  if (( SECONDS >= A21_WAIT_ROM_DEADLINE )); then",
			fmt.Sprintf("    echo %s >&2", shellSingleQuote("Timed out waiting for ESP32-S3 ROM download mode; hold BOOT, press/release RESET, keep holding BOOT, then retry.")),
			`    echo "ROM candidates checked: ${A21_WAIT_ROM_CANDIDATES[*]}" >&2`,
			`    echo "Last esptool probe output:" >&2`,
			`    tail -n 12 "${A21_WAIT_ROM_PROBE_LOG}" >&2 || true`,
			"    exit 1",
			"  fi",
			`  if (( A21_WAIT_ROM_ATTEMPT % 10 == 0 )); then`,
			`    echo "Still waiting for ESP32-S3 ROM download mode; candidates: ${A21_WAIT_ROM_CANDIDATES[*]}"`,
			"  fi",
			"  sleep 1",
			"done",
		)
	}
	scriptLines = append(scriptLines, fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before %s --after hard_reset write_flash @flash_args", flashPortArg, shellSingleQuote(esptoolBefore)))
	script := strings.Join(scriptLines, "\n")
	if err := runStackChanOfficialXiaozhiCompatibleFlashCommand(ctx, flashLogPath, script); err != nil {
		return err
	}
	report.FlashExecuted = true
	report.Status = "passed"
	return nil
}

func executeStackChanOfficialXiaozhiCompatibleNVS(ctx context.Context, options stackChanOfficialXiaozhiCompatibleNVSOptions, report *stackChanOfficialXiaozhiCompatibleNVSReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	if options.IDFPython != "" {
		if _, err := os.Stat(options.IDFPython); err != nil {
			return fmt.Errorf("ESP-IDF python override is missing: %w", err)
		}
	}
	if _, err := os.Stat(report.Tools.NVSToolPath); err != nil {
		return fmt.Errorf("ESP-IDF nvs_tool.py is missing: %w", err)
	}
	if _, err := os.Stat(report.Tools.NVSGeneratorPath); err != nil {
		return fmt.Errorf("ESP-IDF nvs_partition_gen.py is missing: %w", err)
	}
	if err := os.MkdirAll(options.RunDir, 0o700); err != nil {
		return fmt.Errorf("create nvs run dir: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-before-"+timestamp+".bin")
	beforeJSONPath := filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-before-"+timestamp+".json")
	provisionCSVPath := filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-provision-"+timestamp+".csv")
	provisionedBinPath := filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-provision-"+timestamp+".bin")
	afterJSONPath := filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-provision-"+timestamp+".json")
	report.ReadLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-read-"+timestamp+".log")
	report.ParseLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-parse-"+timestamp+".log")
	report.GenerateLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-generate-"+timestamp+".log")
	report.VerifyLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-verify-"+timestamp+".log")
	report.WriteLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-write-"+timestamp+".log")

	report.DryRun = false
	report.WriteAllowed = true
	report.NextRequiredConfirmation = ""
	report.BackupPath = backupPath
	report.ProvisionCSVPath = provisionCSVPath
	report.ProvisionedBinPath = provisionedBinPath

	pythonCommand := "python"
	sourceIDF := true
	if options.IDFPython != "" {
		pythonCommand = shellSingleQuote(options.IDFPython)
		sourceIDF = false
	}
	nvsScriptLines := func(lines ...string) string {
		script := []string{"set -euo pipefail"}
		if sourceIDF {
			script = append(script, fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)))
		}
		script = append(script, lines...)
		return strings.Join(script, "\n")
	}

	readScript := nvsScriptLines(
		fmt.Sprintf("%s -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after no_reset read_flash %s %s %s",
			pythonCommand,
			shellSingleQuote(options.Port),
			stackChanOfficialPCMBridgeNVSOffset,
			stackChanOfficialPCMBridgeNVSSizeHex,
			shellSingleQuote(backupPath)),
	)
	if err := runStackChanOfficialXiaozhiCompatibleNVSCommand(ctx, report.ReadLogPath, readScript); err != nil {
		return fmt.Errorf("read current NVS partition: %w", err)
	}
	backupSHA, err := sha256File(backupPath)
	if err != nil {
		return fmt.Errorf("hash NVS backup: %w", err)
	}
	report.BackupSHA256 = backupSHA

	parseScript := nvsScriptLines(
		fmt.Sprintf("%s %s -d minimal -f json %s > %s",
			pythonCommand,
			shellSingleQuote(report.Tools.NVSToolPath),
			shellSingleQuote(backupPath),
			shellSingleQuote(beforeJSONPath)),
	)
	if err := runStackChanOfficialXiaozhiCompatibleNVSCommand(ctx, report.ParseLogPath, parseScript); err != nil {
		return fmt.Errorf("parse current NVS partition: %w", err)
	}
	entries, err := readStackChanNVSMinimalEntries(beforeJSONPath)
	if err != nil {
		return err
	}
	csvFile, err := os.Create(provisionCSVPath)
	if err != nil {
		return fmt.Errorf("create NVS provision CSV: %w", err)
	}
	summary, csvErr := writeOfficialXiaozhiCompatibleNVSCSV(csvFile, entries, options.OTAURL, options.WebSocketURL, options.WebSocketVersion, options.WiFiSSID, options.WiFiPassword)
	closeErr := csvFile.Close()
	if csvErr != nil {
		return csvErr
	}
	if closeErr != nil {
		return fmt.Errorf("close NVS provision CSV: %w", closeErr)
	}
	report.Summary = &summary

	generateScript := nvsScriptLines(
		fmt.Sprintf("%s %s generate %s %s %s",
			pythonCommand,
			shellSingleQuote(report.Tools.NVSGeneratorPath),
			shellSingleQuote(provisionCSVPath),
			shellSingleQuote(provisionedBinPath),
			stackChanOfficialPCMBridgeNVSSizeHex),
	)
	if err := runStackChanOfficialXiaozhiCompatibleNVSCommand(ctx, report.GenerateLogPath, generateScript); err != nil {
		return fmt.Errorf("generate provisioned NVS partition: %w", err)
	}
	provisionedSHA, err := sha256File(provisionedBinPath)
	if err != nil {
		return fmt.Errorf("hash provisioned NVS partition: %w", err)
	}
	report.ProvisionedBinSHA256 = provisionedSHA

	verifyScript := nvsScriptLines(
		fmt.Sprintf("%s %s -d minimal -f json %s > %s",
			pythonCommand,
			shellSingleQuote(report.Tools.NVSToolPath),
			shellSingleQuote(provisionedBinPath),
			shellSingleQuote(afterJSONPath)),
	)
	if err := runStackChanOfficialXiaozhiCompatibleNVSCommand(ctx, report.VerifyLogPath, verifyScript); err != nil {
		return fmt.Errorf("verify provisioned NVS partition: %w", err)
	}
	if err := verifyOfficialXiaozhiCompatibleNVSProvision(afterJSONPath, options.OTAURL, options.WebSocketURL, options.WebSocketVersion, options.WiFiSSID, options.WiFiPassword); err != nil {
		return err
	}

	writeScript := nvsScriptLines(
		fmt.Sprintf("%s -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after hard_reset write_flash %s %s",
			pythonCommand,
			shellSingleQuote(options.Port),
			stackChanOfficialPCMBridgeNVSOffset,
			shellSingleQuote(provisionedBinPath)),
	)
	if err := runStackChanOfficialXiaozhiCompatibleNVSCommand(ctx, report.WriteLogPath, writeScript); err != nil {
		return fmt.Errorf("write provisioned NVS partition: %w", err)
	}
	report.WriteExecuted = true
	report.Status = "passed"
	return nil
}

func executeStackChanOfficialPCMBridgeNVS(ctx context.Context, options stackChanOfficialPCMBridgeNVSOptions, report *stackChanOfficialPCMBridgeNVSReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	if _, err := os.Stat(report.Tools.NVSToolPath); err != nil {
		return fmt.Errorf("ESP-IDF nvs_tool.py is missing: %w", err)
	}
	if _, err := os.Stat(report.Tools.NVSGeneratorPath); err != nil {
		return fmt.Errorf("ESP-IDF nvs_partition_gen.py is missing: %w", err)
	}
	if err := os.MkdirAll(options.RunDir, 0o700); err != nil {
		return fmt.Errorf("create nvs run dir: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-before-"+timestamp+".bin")
	beforeJSONPath := filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-before-"+timestamp+".json")
	provisionCSVPath := filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-provision-"+timestamp+".csv")
	provisionedBinPath := filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-provision-"+timestamp+".bin")
	afterJSONPath := filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-provision-"+timestamp+".json")
	report.ReadLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-read-"+timestamp+".log")
	report.ParseLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-parse-"+timestamp+".log")
	report.GenerateLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-generate-"+timestamp+".log")
	report.VerifyLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-verify-"+timestamp+".log")
	report.WriteLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-write-"+timestamp+".log")

	report.DryRun = false
	report.WriteAllowed = true
	report.NextRequiredConfirmation = ""
	report.BackupPath = backupPath
	report.ProvisionCSVPath = provisionCSVPath
	report.ProvisionedBinPath = provisionedBinPath

	readScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after no_reset read_flash %s %s %s",
			shellSingleQuote(options.Port),
			stackChanOfficialPCMBridgeNVSOffset,
			stackChanOfficialPCMBridgeNVSSizeHex,
			shellSingleQuote(backupPath)),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeNVSCommand(ctx, report.ReadLogPath, readScript); err != nil {
		return fmt.Errorf("read current NVS partition: %w", err)
	}
	backupSHA, err := sha256File(backupPath)
	if err != nil {
		return fmt.Errorf("hash NVS backup: %w", err)
	}
	report.BackupSHA256 = backupSHA

	parseScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python %s -d minimal -f json %s > %s",
			shellSingleQuote(report.Tools.NVSToolPath),
			shellSingleQuote(backupPath),
			shellSingleQuote(beforeJSONPath)),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeNVSCommand(ctx, report.ParseLogPath, parseScript); err != nil {
		return fmt.Errorf("parse current NVS partition: %w", err)
	}
	entries, err := readStackChanNVSMinimalEntries(beforeJSONPath)
	if err != nil {
		return err
	}
	csvFile, err := os.Create(provisionCSVPath)
	if err != nil {
		return fmt.Errorf("create NVS provision CSV: %w", err)
	}
	summary, csvErr := writeOfficialPCMBridgeNVSCSV(csvFile, entries, report.DeviceID, options.AudioWSURL)
	closeErr := csvFile.Close()
	if csvErr != nil {
		return csvErr
	}
	if closeErr != nil {
		return fmt.Errorf("close NVS provision CSV: %w", closeErr)
	}
	report.Summary = &summary

	generateScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python %s generate %s %s %s",
			shellSingleQuote(report.Tools.NVSGeneratorPath),
			shellSingleQuote(provisionCSVPath),
			shellSingleQuote(provisionedBinPath),
			stackChanOfficialPCMBridgeNVSSizeHex),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeNVSCommand(ctx, report.GenerateLogPath, generateScript); err != nil {
		return fmt.Errorf("generate provisioned NVS partition: %w", err)
	}
	provisionedSHA, err := sha256File(provisionedBinPath)
	if err != nil {
		return fmt.Errorf("hash provisioned NVS partition: %w", err)
	}
	report.ProvisionedBinSHA256 = provisionedSHA

	verifyScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python %s -d minimal -f json %s > %s",
			shellSingleQuote(report.Tools.NVSToolPath),
			shellSingleQuote(provisionedBinPath),
			shellSingleQuote(afterJSONPath)),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeNVSCommand(ctx, report.VerifyLogPath, verifyScript); err != nil {
		return fmt.Errorf("verify provisioned NVS partition: %w", err)
	}
	if err := verifyOfficialPCMBridgeNVSProvision(afterJSONPath, report.DeviceID, options.AudioWSURL); err != nil {
		return err
	}

	writeScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after hard_reset write_flash %s %s",
			shellSingleQuote(options.Port),
			stackChanOfficialPCMBridgeNVSOffset,
			shellSingleQuote(provisionedBinPath)),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeNVSCommand(ctx, report.WriteLogPath, writeScript); err != nil {
		return fmt.Errorf("write provisioned NVS partition: %w", err)
	}
	report.WriteExecuted = true
	report.Status = "passed"
	return nil
}

func runStackChanOfficialSmokeFlashCommandExec(ctx context.Context, logPath string, script string) error {
	return runLoggedCommand(ctx, "", logPath, "bash", "-lc", script)
}

func validateOfficialSmokeUploadPort(port string) error {
	port = strings.TrimSpace(port)
	if port == "" {
		return fmt.Errorf("upload port is required")
	}
	base := filepath.Base(port)
	lowerBase := strings.ToLower(base)
	if !(strings.HasPrefix(base, "cu.") || strings.HasPrefix(base, "tty.")) {
		return fmt.Errorf("upload port must be an explicit serial device")
	}
	if !strings.Contains(lowerBase, "usbmodem") && !strings.Contains(lowerBase, "usbserial") {
		return fmt.Errorf("upload port must be an explicit USB serial device")
	}
	return nil
}

func validateStackChanOfficialXiaozhiCompatibleEsptoolBefore(mode string) (string, error) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return "default_reset", nil
	}
	switch mode {
	case "default_reset", "usb_reset", "no_reset":
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported esptool before mode %q; want default_reset, usb_reset, or no_reset", mode)
	}
}

func stackChanOfficialXiaozhiCompatibleWaitROMTimeoutSeconds(options stackChanOfficialXiaozhiCompatibleFlashOptions) int {
	if !options.WaitROM {
		return 0
	}
	if options.WaitROMTimeoutSeconds > 0 {
		return options.WaitROMTimeoutSeconds
	}
	return stackChanOfficialXiaozhiCompatibleDefaultWaitROMTimeoutSeconds
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func inspectStackChanOfficialEvidence(sourceRoot string) stackChanOfficialBaselineEvidence {
	sdkConfig := readGitTrackedOrFile(sourceRoot, "firmware/sdkconfig.defaults")
	halAudio := readGitTrackedOrFile(sourceRoot, "firmware/main/hal/audio.cpp")
	coreS3Codec := readGitTrackedOrFile(sourceRoot, "firmware/main/hal/board/cores3_audio_codec.cc")
	repos := readGitTrackedOrFile(sourceRoot, "firmware/repos.json")
	xiaozhiService := readGitTrackedOrFile(sourceRoot, "firmware/xiaozhi-esp32/main/audio/audio_service.cc")
	xiaozhiHeader := readGitTrackedOrFile(sourceRoot, "firmware/xiaozhi-esp32/main/audio/audio_service.h")

	evidence := stackChanOfficialBaselineEvidence{
		TrackedSDKConfigHasNoLegacyIdentity:  !containsLegacyIdentity(sdkConfig),
		HalMicTestUsesOutputData:             strings.Contains(halAudio, "audio_codec->OutputData"),
		CoreS3UsesESPCodecDev:                strings.Contains(coreS3Codec, "esp_codec_dev_write") && strings.Contains(coreS3Codec, "esp_codec_dev_open"),
		CoreS3CreatesDuplexChannels:          strings.Contains(coreS3Codec, "CreateDuplexChannels"),
		ReposDeclareXiaoZhiAudioService:      strings.Contains(repos, "xiaozhi-esp32") && strings.Contains(repos, "v2.2.4"),
		XiaoZhiOutputTaskUsesCodecOutputData: strings.Contains(xiaozhiService, "AudioOutputTask") && strings.Contains(xiaozhiService, "codec_->OutputData"),
		XiaoZhiOpusFrameDurationMS:           parseOpusFrameDurationMS(xiaozhiHeader),
		MatureComponents: []string{
			"ESP-IDF",
			"esp_codec_dev",
			"esp_audio_codec",
			"esp-sr",
			"xiaozhi AudioService pattern",
			"Opus frame pipeline",
		},
		ReferenceFiles: []string{
			"firmware/main/hal/audio.cpp",
			"firmware/main/hal/board/cores3_audio_codec.cc",
			"firmware/main/hal/board/config.h",
			"firmware/repos.json",
			"firmware/xiaozhi-esp32/main/audio/audio_service.cc",
			"firmware/xiaozhi-esp32/main/audio/audio_service.h",
		},
	}
	return evidence
}

func applyStackChanOfficialCandidateContract(report *stackChanOfficialBaselineReport, sourceRoot string, overlays []string) {
	if report == nil {
		return
	}
	mainCPP := readGitTrackedOrFile(sourceRoot, "firmware/main/main.cpp")
	overlayText := readOfficialOverlayText(overlays)
	candidate, role := classifyStackChanOfficialCandidate(overlays, mainCPP)

	avatarPreserved := strings.Contains(mainCPP, "AppAvatar") &&
		strings.Contains(mainCPP, "AppAiAgent") &&
		strings.Contains(mainCPP, "GetMooncake().installApp")
	xiaozhiStartPreserved := strings.Contains(mainCPP, "GetHAL().startXiaozhi()") &&
		strings.Contains(mainCPP, "isXiaozhiStartRequested")
	minimalBridgeScreen := strings.Contains(mainCPP, "A21 BRIDGE") ||
		strings.Contains(mainCPP, "renderStatus()") ||
		strings.Contains(overlayText, "A21 BRIDGE") ||
		strings.Contains(overlayText, "renderStatus()")

	if strings.Contains(overlayText, "-    GetMooncake().installApp(std::make_unique<AppAvatar>())") ||
		strings.Contains(overlayText, "-    GetMooncake().installApp(std::make_unique<AppAiAgent>())") {
		avatarPreserved = false
	}
	if strings.Contains(overlayText, "-    GetHAL().startXiaozhi()") {
		xiaozhiStartPreserved = false
	}
	if strings.Contains(candidate, "pcm-bridge") {
		role = "diagnostic_m3_prep"
		minimalBridgeScreen = true
		avatarPreserved = false
		xiaozhiStartPreserved = false
	}
	if strings.Contains(candidate, "audio-smoke") {
		role = "diagnostic_audio_smoke"
		avatarPreserved = false
		xiaozhiStartPreserved = false
	}

	report.FirmwareCandidate = candidate
	report.BuildLaneRole = role
	report.OfficialAvatarActionPreserved = avatarPreserved
	report.OfficialXiaozhiStartPreserved = xiaozhiStartPreserved
	report.MinimalBridgeScreen = minimalBridgeScreen
}

func classifyStackChanOfficialCandidate(overlays []string, mainCPP string) (string, string) {
	for _, overlay := range overlays {
		base := filepath.Base(overlay)
		switch {
		case strings.Contains(base, "xiaozhi-compatible"):
			return stackChanOfficialXiaozhiCompatibleFirmwareCandidate, "product_candidate"
		case strings.Contains(base, "pcm-bridge"):
			return "a21-stackchan-official-pcm-bridge", "diagnostic_m3_prep"
		case strings.Contains(base, "audio-smoke"):
			return "a21-stackchan-official-audio-smoke", "diagnostic_audio_smoke"
		}
	}
	switch {
	case strings.Contains(mainCPP, "A21 BRIDGE"):
		return "a21-stackchan-official-pcm-bridge", "diagnostic_m3_prep"
	case strings.Contains(mainCPP, "a21-stackchan-official-xiaozhi-compatible"):
		return stackChanOfficialXiaozhiCompatibleFirmwareCandidate, "product_candidate"
	case strings.Contains(mainCPP, "a21-stackchan-official-audio-smoke"):
		return "a21-stackchan-official-audio-smoke", "diagnostic_audio_smoke"
	default:
		return "stack-chan-official-baseline", "official_reference"
	}
}

func readOfficialOverlayText(overlays []string) string {
	var builder strings.Builder
	for _, overlay := range overlays {
		if containsLegacyIdentityPathToken(overlay) {
			continue
		}
		data, err := os.ReadFile(overlay)
		if err != nil {
			continue
		}
		builder.Write(data)
		builder.WriteByte('\n')
	}
	return builder.String()
}

func executeStackChanOfficialBaseline(ctx context.Context, options stackChanOfficialBaselineOptions, report *stackChanOfficialBaselineReport) {
	report.Status = "passed"
	if err := os.RemoveAll(options.WorkDir); err != nil {
		report.fail("clean_work_dir_failed", "failed to clean official baseline work dir")
		return
	}
	if err := os.RemoveAll(options.BuildDir); err != nil {
		report.fail("clean_build_dir_failed", "failed to clean official baseline build dir")
		return
	}
	if err := os.MkdirAll(options.WorkDir, 0o755); err != nil {
		report.fail("create_work_dir_failed", "failed to create official baseline work dir")
		return
	}
	if err := exportGitHEAD(ctx, report.SourceRoot, options.WorkDir); err != nil {
		report.fail("export_head_failed", "failed to export official source HEAD")
		return
	}

	buildLog := filepath.Join(options.BuildDir, "a21-official-build.log")
	_ = os.MkdirAll(options.BuildDir, 0o755)

	if options.DepCache != "" {
		report.Build.DependencyCacheUsed = true
		report.Build.DependencyCacheRoot = options.DepCache
		if err := hydrateStackChanOfficialDependenciesFromCache(options.WorkDir, options.DepCache); err != nil {
			report.fail("dependency_cache_failed", err.Error())
			return
		}
	} else {
		fetchLog := filepath.Join(options.BuildDir, "a21-official-fetch.log")
		report.Build.FetchExecuted = true
		report.Build.FetchLogPath = fetchLog
		if err := runLoggedCommand(ctx, filepath.Join(options.WorkDir, "firmware"), fetchLog, "python3", "./fetch_repos.py"); err != nil {
			report.fail("fetch_repos_failed", "official fetch_repos.py failed")
			return
		}
	}
	for index, overlay := range options.Overlays {
		cleanOverlay := filepath.Clean(overlay)
		if containsLegacyIdentityPathToken(cleanOverlay) {
			report.fail("overlay_path_legacy_identity", "overlay path contains forbidden legacy identity")
			return
		}
		if err := runLoggedCommand(ctx, options.WorkDir, filepath.Join(options.BuildDir, "a21-official-overlay.log"), "git", "apply", "--recount", cleanOverlay); err != nil {
			report.fail("overlay_apply_failed", "failed to apply A21 official baseline overlay")
			return
		}
		if index < len(report.Overlays) {
			report.Overlays[index].Applied = true
		}
	}

	report.Evidence = inspectStackChanOfficialEvidence(options.WorkDir)
	applyStackChanOfficialCandidateContract(report, options.WorkDir, options.Overlays)
	report.Build.BuildExecuted = true
	report.Build.BuildLogPath = buildLog
	if _, err := os.Stat(options.IDFExport); err != nil {
		report.fail("idf_export_missing", "ESP-IDF export.sh is missing")
		return
	}
	buildScript := fmt.Sprintf("set -euo pipefail\nsource %q >/dev/null\nidf.py -C %q -B %q build", options.IDFExport, filepath.Join(options.WorkDir, "firmware"), options.BuildDir)
	if err := runLoggedCommand(ctx, "", buildLog, "bash", "-lc", buildScript); err != nil {
		report.fail("idf_build_failed", "official ESP-IDF build failed")
		return
	}
	report.Build.Artifacts = collectOfficialStackChanBuildArtifacts(options.BuildDir)
	if len(report.Build.Artifacts) == 0 {
		report.fail("build_artifacts_missing", "official build completed without expected artifacts")
		return
	}
	if err := validateOfficialAssetsPartitionCapacity(options.BuildDir); err != nil {
		report.fail("assets_partition_capacity_failed", err.Error())
		return
	}
}

func exportGitHEAD(ctx context.Context, sourceRoot string, workDir string) error {
	archive := exec.CommandContext(ctx, "git", "-C", sourceRoot, "archive", "HEAD")
	tar := exec.CommandContext(ctx, "tar", "-x", "-C", workDir)
	reader, err := archive.StdoutPipe()
	if err != nil {
		return err
	}
	tar.Stdin = reader
	if err := tar.Start(); err != nil {
		return err
	}
	if err := archive.Start(); err != nil {
		return err
	}
	archiveErr := archive.Wait()
	tarErr := tar.Wait()
	if archiveErr != nil {
		return archiveErr
	}
	return tarErr
}

type stackChanOfficialRepoConfig struct {
	URL            string `json:"url"`
	Path           string `json:"path"`
	Branch         string `json:"branch"`
	WithSubmodules bool   `json:"with_submodules"`
	Patch          string `json:"patch"`
}
