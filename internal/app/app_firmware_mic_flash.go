package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/runtimeguard"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type firmwareMicProbeFlashExecutionReport struct {
	SchemaVersion  string                                `json:"schema_version"`
	GeneratedAtMS  int64                                 `json:"generated_at_ms"`
	DryRun         bool                                  `json:"dry_run"`
	FlashExecuted  bool                                  `json:"flash_executed"`
	ControlGuard   *runtimeguard.ControlGuardReport      `json:"control_guard,omitempty"`
	Port           string                                `json:"port"`
	ArtifactPath   string                                `json:"artifact_path"`
	ArtifactSHA256 string                                `json:"artifact_sha256"`
	Commit         string                                `json:"commit"`
	PlatformIOEnv  string                                `json:"platformio_env"`
	Plan           firmwarecheck.MicProbeFlashPlanResult `json:"plan"`
	Command        []string                              `json:"command"`
	ReportPath     string                                `json:"report_path,omitempty"`
}

func writeFirmwareMicProbeFlashPlanReport(outputDir string, result firmwarecheck.MicProbeFlashPlanResult) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-firmware-mic-probe-flash-plan-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	result.ReportPath = reportPath
	if err := writeJSONFirmwareMicProbeFlashPlan(file, result); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeFirmwareMicProbeFlashExecutionReport(outputDir string, report firmwareMicProbeFlashExecutionReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-firmware-mic-probe-flash-execution-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONFirmwareMicProbeFlashExecution(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func runFirmwareMicProbeFlashPlan(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.MicProbeFlashPlanOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		ArtifactPath: filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanMicProbePlatformIOEnv, "firmware.bin"),
		BuildDir:     filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanMicProbePlatformIOEnv),
		CoreDir:      filepath.Join(".a21-tools", "platformio-core"),
	}
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-mic-probe-flash-plan --port /dev/cu.usbmodemXXXX --commit <git-sha> [--artifact firmware/stackchan/.pio/build/a21_stackchan_cores3_mic_probe/firmware.bin] [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3_mic_probe] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
			return 0
		case "--manifest":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--manifest requires a value")
				return 2
			}
			i++
			options.ManifestPath = args[i]
		case "--artifact":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--artifact requires a value")
				return 2
			}
			i++
			options.ArtifactPath = args[i]
		case "--port":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--port requires a value")
				return 2
			}
			i++
			options.Port = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.ExpectedGitCommit = args[i]
		case "--build-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--build-dir requires a value")
				return 2
			}
			i++
			options.BuildDir = args[i]
		case "--core-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--core-dir requires a value")
				return 2
			}
			i++
			options.CoreDir = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown firmware-mic-probe-flash-plan option %q\n", args[i])
			return 2
		}
	}
	result, code := buildFirmwareMicProbeFlashPlanFromCLI(options, outputDir, stderr)
	if code != 0 {
		return code
	}
	if err := writeJSONFirmwareMicProbeFlashPlan(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware mic probe flash plan: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware mic probe flash plan ok (no flash performed)")
	return 0
}
func runFirmwareMicProbeFlashExecute(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.MicProbeFlashPlanOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		ArtifactPath: filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanMicProbePlatformIOEnv, "firmware.bin"),
		BuildDir:     filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanMicProbePlatformIOEnv),
		CoreDir:      filepath.Join(".a21-tools", "platformio-core"),
	}
	outputDir := ""
	confirm := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-mic-probe-flash-execute --port /dev/cu.usbmodemXXXX --commit <git-sha> --confirm WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE [--artifact firmware/stackchan/.pio/build/a21_stackchan_cores3_mic_probe/firmware.bin] [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3_mic_probe] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
			return 0
		case "--manifest":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--manifest requires a value")
				return 2
			}
			i++
			options.ManifestPath = args[i]
		case "--artifact":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--artifact requires a value")
				return 2
			}
			i++
			options.ArtifactPath = args[i]
		case "--port":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--port requires a value")
				return 2
			}
			i++
			options.Port = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.ExpectedGitCommit = args[i]
		case "--confirm":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--confirm requires WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE")
				return 2
			}
			i++
			confirm = args[i]
		case "--build-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--build-dir requires a value")
				return 2
			}
			i++
			options.BuildDir = args[i]
		case "--core-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--core-dir requires a value")
				return 2
			}
			i++
			options.CoreDir = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown firmware-mic-probe-flash-execute option %q\n", args[i])
			return 2
		}
	}
	if confirm != "WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE" {
		fmt.Fprintln(stderr, "--confirm WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE is required")
		return 2
	}
	controlGuard, code := requireA21ControlAllowed("firmware-mic-probe-flash-execute", stderr)
	if code != 0 {
		return code
	}
	plan, code := buildFirmwareMicProbeFlashPlanFromCLI(options, "", stderr)
	if code != 0 {
		return code
	}
	command, err := firmwareMicProbeFlashCommand(plan)
	if err != nil {
		fmt.Fprintf(stderr, "firmware mic probe flash execute failed: %v\n", err)
		return 1
	}
	if err := runFirmwareBootstrapFlashCommand(context.Background(), command, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "firmware mic probe flash execute failed: %v\n", err)
		return 1
	}
	report := firmwareMicProbeFlashExecutionReport{
		SchemaVersion:  "a21.firmware.mic_probe_flash_execution.v1",
		GeneratedAtMS:  time.Now().UnixMilli(),
		DryRun:         false,
		FlashExecuted:  true,
		ControlGuard:   &controlGuard,
		Port:           plan.Port,
		ArtifactPath:   plan.ArtifactPath,
		ArtifactSHA256: plan.ArtifactSHA256,
		Commit:         plan.Commit,
		PlatformIOEnv:  plan.PlatformIOEnv,
		Plan:           plan,
		Command:        redactBootstrapFlashCommand(command),
	}
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "firmware mic probe flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeFirmwareMicProbeFlashExecutionReport(outputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write firmware mic probe flash execution report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONFirmwareMicProbeFlashExecution(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode firmware mic probe flash execution: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware mic probe flash executed")
	return 0
}
func buildFirmwareMicProbeFlashPlanFromCLI(options firmwarecheck.MicProbeFlashPlanOptions, outputDir string, stderr io.Writer) (firmwarecheck.MicProbeFlashPlanResult, int) {
	if options.Port == "" {
		fmt.Fprintln(stderr, "--port requires a value")
		return firmwarecheck.MicProbeFlashPlanResult{}, 2
	}
	if options.ExpectedGitCommit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return firmwarecheck.MicProbeFlashPlanResult{}, 2
	}
	for _, path := range []string{options.ManifestPath, options.ArtifactPath, options.BuildDir, options.CoreDir} {
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "firmware mic probe flash path invalid: %v\n", err)
			return firmwarecheck.MicProbeFlashPlanResult{}, 1
		}
	}
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "firmware mic probe flash report dir invalid: %v\n", err)
			return firmwarecheck.MicProbeFlashPlanResult{}, 1
		}
	}
	sourceState, err := detectFirmwareSourceState()
	if err != nil {
		fmt.Fprintf(stderr, "firmware mic probe flash failed: source tree check failed: %v\n", err)
		return firmwarecheck.MicProbeFlashPlanResult{}, 1
	}
	if !sourceState.Clean {
		fmt.Fprintf(stderr, "firmware mic probe flash failed: source tree is dirty under %s\n%s\n", sourceState.Root, sourceState.Detail)
		return firmwarecheck.MicProbeFlashPlanResult{}, 1
	}
	portUsage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		fmt.Fprintf(stderr, "firmware mic probe flash failed: upload port ownership check failed: %v\n", err)
		return firmwarecheck.MicProbeFlashPlanResult{}, 1
	}
	options.PortUsage = portUsage
	result, err := firmwarecheck.BuildMicProbeFlashPlan(options)
	if err != nil {
		fmt.Fprintf(stderr, "firmware mic probe flash failed: %v\n", err)
		return firmwarecheck.MicProbeFlashPlanResult{}, 1
	}
	if outputDir != "" {
		reportPath, err := writeFirmwareMicProbeFlashPlanReport(outputDir, result)
		if err != nil {
			fmt.Fprintf(stderr, "write firmware mic probe flash plan: %v\n", err)
			return firmwarecheck.MicProbeFlashPlanResult{}, 1
		}
		result.ReportPath = reportPath
	}
	return result, 0
}
func firmwareMicProbeFlashCommand(plan firmwarecheck.MicProbeFlashPlanResult) ([]string, error) {
	if plan.GuardID != "a21.firmware.mic_probe_flash_plan.v1" || !plan.OK {
		return nil, fmt.Errorf("invalid mic probe flash plan")
	}
	if plan.PlatformIOEnv != firmwarecheck.StackChanMicProbePlatformIOEnv {
		return nil, fmt.Errorf("mic probe flash plan has wrong PlatformIO env")
	}
	if plan.Port == "" || len(plan.Parts) != 4 {
		return nil, fmt.Errorf("mic probe flash plan missing port or image parts")
	}
	pythonPath := filepath.Join(".a21-tools", "platformio-venv", "bin", "python")
	esptoolPath := filepath.Join(".a21-tools", "platformio-core", "packages", "tool-esptoolpy", "esptool.py")
	args := []string{
		pythonPath,
		esptoolPath,
		"--chip", "esp32s3",
		"--port", plan.Port,
		"--baud", "460800",
		"--before", "default_reset",
		"--after", "hard_reset",
		"write_flash",
		"-z",
		"--flash_mode", "dio",
		"--flash_freq", "80m",
		"--flash_size", "16MB",
	}
	for _, part := range plan.Parts {
		if part.Offset == "" || part.Path == "" {
			return nil, fmt.Errorf("mic probe flash plan contains empty image part")
		}
		args = append(args, part.Offset, part.Path)
	}
	return args, nil
}
func writeJSONFirmwareMicProbeFlashPlan(writer io.Writer, result firmwarecheck.MicProbeFlashPlanResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
func writeJSONFirmwareMicProbeFlashExecution(writer io.Writer, report firmwareMicProbeFlashExecutionReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
