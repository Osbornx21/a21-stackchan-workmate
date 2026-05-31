package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/runtimeguard"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var runFirmwareBootstrapFlashCommand = runFirmwareBootstrapFlashCommandExec

type firmwareBootstrapFlashExecutionReport struct {
	SchemaVersion  string                                 `json:"schema_version"`
	GeneratedAtMS  int64                                  `json:"generated_at_ms"`
	DryRun         bool                                   `json:"dry_run"`
	FlashExecuted  bool                                   `json:"flash_executed"`
	ControlGuard   *runtimeguard.ControlGuardReport       `json:"control_guard,omitempty"`
	Port           string                                 `json:"port"`
	ArtifactPath   string                                 `json:"artifact_path"`
	ArtifactSHA256 string                                 `json:"artifact_sha256"`
	Commit         string                                 `json:"commit"`
	Plan           firmwarecheck.BootstrapFlashPlanResult `json:"plan"`
	Command        []string                               `json:"command"`
	ReportPath     string                                 `json:"report_path,omitempty"`
}

func writeFirmwareFlashPlanReport(outputDir string, result firmwarecheck.FlashPlanResult) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-firmware-flash-plan-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	result.ReportPath = reportPath
	if err := writeJSONFirmwareFlashPlan(file, result); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeFirmwareBootstrapFlashPlanReport(outputDir string, result firmwarecheck.BootstrapFlashPlanResult) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-firmware-bootstrap-flash-plan-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	result.ReportPath = reportPath
	if err := writeJSONFirmwareBootstrapFlashPlan(file, result); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeFirmwareBootstrapFlashExecutionReport(outputDir string, report firmwareBootstrapFlashExecutionReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-firmware-bootstrap-flash-execution-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONFirmwareBootstrapFlashExecution(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func runFirmwareFlashPlan(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.FlashPlanOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
	}
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-flash-plan --artifact firmware/artifacts/<a21-stackchan...bin> --port /dev/cu.usbmodemXXXX --device-report reports/devices.json --device-id stackchan-001 --commit <git-sha> --max-device-age-ms 300000 [--output-dir reports]")
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
		case "--device-report":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-report requires a value")
				return 2
			}
			i++
			options.ReportPath = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.ExpectedDeviceID = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.ExpectedGitCommit = args[i]
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
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown firmware-flash-plan option %q\n", args[i])
			return 2
		}
	}
	if options.ArtifactPath == "" {
		fmt.Fprintln(stderr, "--artifact requires a value")
		return 2
	}
	if options.Port == "" {
		fmt.Fprintln(stderr, "--port requires a value")
		return 2
	}
	if options.ReportPath == "" {
		fmt.Fprintln(stderr, "--device-report requires a value")
		return 2
	}
	if options.ExpectedDeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if options.ExpectedGitCommit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	if options.MaxDeviceAgeMS <= 0 {
		fmt.Fprintln(stderr, "--max-device-age-ms requires a value")
		return 2
	}
	for _, path := range []string{options.ManifestPath, options.ArtifactPath, options.ReportPath} {
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "firmware flash plan path invalid: %v\n", err)
			return 1
		}
	}
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "firmware flash plan report dir invalid: %v\n", err)
			return 1
		}
	}
	portUsage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		fmt.Fprintf(stderr, "firmware flash plan failed: upload port ownership check failed: %v\n", err)
		return 1
	}
	options.PortUsage = portUsage
	result, err := firmwarecheck.BuildFlashPlan(options)
	if err != nil {
		fmt.Fprintf(stderr, "firmware flash plan failed: %v\n", err)
		return 1
	}
	if outputDir != "" {
		reportPath, err := writeFirmwareFlashPlanReport(outputDir, result)
		if err != nil {
			fmt.Fprintf(stderr, "write firmware flash plan report: %v\n", err)
			return 1
		}
		result.ReportPath = reportPath
	}
	if err := writeJSONFirmwareFlashPlan(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware flash plan result: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware flash plan guard ok (no flash performed)")
	return 0
}
func runFirmwareBootstrapFlashPlan(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.BootstrapFlashPlanOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		BuildDir:     filepath.Join("firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3"),
		CoreDir:      filepath.Join(".a21-tools", "platformio-core"),
	}
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-bootstrap-flash --artifact firmware/artifacts/<a21-stackchan...bin> --port /dev/cu.usbmodemXXXX --commit <git-sha> [--execute --confirm WRITE_A21_STACKCHAN_FIRMWARE] [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
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
			fmt.Fprintf(stderr, "unknown firmware-bootstrap-flash-plan option %q\n", args[i])
			return 2
		}
	}
	result, code := buildFirmwareBootstrapFlashPlanFromCLI(options, outputDir, stderr)
	if code != 0 {
		return code
	}
	if err := writeJSONFirmwareBootstrapFlashPlan(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware bootstrap flash plan: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware bootstrap flash plan ok (no flash performed)")
	return 0
}
func runFirmwareBootstrapFlashExecute(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.BootstrapFlashPlanOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		BuildDir:     filepath.Join("firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3"),
		CoreDir:      filepath.Join(".a21-tools", "platformio-core"),
	}
	outputDir := ""
	confirm := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-bootstrap-flash --execute --artifact firmware/artifacts/<a21-stackchan...bin> --port /dev/cu.usbmodemXXXX --commit <git-sha> --confirm WRITE_A21_STACKCHAN_FIRMWARE [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
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
				fmt.Fprintln(stderr, "--confirm requires WRITE_A21_STACKCHAN_FIRMWARE")
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
			fmt.Fprintf(stderr, "unknown firmware-bootstrap-flash-execute option %q\n", args[i])
			return 2
		}
	}
	if confirm != "WRITE_A21_STACKCHAN_FIRMWARE" {
		fmt.Fprintln(stderr, "--confirm WRITE_A21_STACKCHAN_FIRMWARE is required")
		return 2
	}
	controlGuard, code := requireA21ControlAllowed("firmware-bootstrap-flash --execute", stderr)
	if code != 0 {
		return code
	}
	plan, code := buildFirmwareBootstrapFlashPlanFromCLI(options, "", stderr)
	if code != 0 {
		return code
	}
	command, err := firmwareBootstrapFlashCommand(plan)
	if err != nil {
		fmt.Fprintf(stderr, "firmware bootstrap flash execute failed: %v\n", err)
		return 1
	}
	if err := runFirmwareBootstrapFlashCommand(context.Background(), command, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "firmware bootstrap flash execute failed: %v\n", err)
		return 1
	}
	report := firmwareBootstrapFlashExecutionReport{
		SchemaVersion:  "a21.firmware.bootstrap_flash_execution.v1",
		GeneratedAtMS:  time.Now().UnixMilli(),
		DryRun:         false,
		FlashExecuted:  true,
		ControlGuard:   &controlGuard,
		Port:           plan.Port,
		ArtifactPath:   plan.ArtifactPath,
		ArtifactSHA256: plan.ArtifactSHA256,
		Commit:         plan.Commit,
		Plan:           plan,
		Command:        redactBootstrapFlashCommand(command),
	}
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "firmware bootstrap flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeFirmwareBootstrapFlashExecutionReport(outputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write firmware bootstrap flash execution report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONFirmwareBootstrapFlashExecution(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode firmware bootstrap flash execution: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware bootstrap flash executed")
	return 0
}
func buildFirmwareBootstrapFlashPlanFromCLI(options firmwarecheck.BootstrapFlashPlanOptions, outputDir string, stderr io.Writer) (firmwarecheck.BootstrapFlashPlanResult, int) {
	if options.ArtifactPath == "" {
		fmt.Fprintln(stderr, "--artifact requires a value")
		return firmwarecheck.BootstrapFlashPlanResult{}, 2
	}
	if options.Port == "" {
		fmt.Fprintln(stderr, "--port requires a value")
		return firmwarecheck.BootstrapFlashPlanResult{}, 2
	}
	if options.ExpectedGitCommit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return firmwarecheck.BootstrapFlashPlanResult{}, 2
	}
	for _, path := range []string{options.ManifestPath, options.ArtifactPath, options.BuildDir, options.CoreDir} {
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "firmware bootstrap flash path invalid: %v\n", err)
			return firmwarecheck.BootstrapFlashPlanResult{}, 1
		}
	}
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "firmware bootstrap flash report dir invalid: %v\n", err)
			return firmwarecheck.BootstrapFlashPlanResult{}, 1
		}
	}
	portUsage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		fmt.Fprintf(stderr, "firmware bootstrap flash failed: upload port ownership check failed: %v\n", err)
		return firmwarecheck.BootstrapFlashPlanResult{}, 1
	}
	options.PortUsage = portUsage
	result, err := firmwarecheck.BuildBootstrapFlashPlan(options)
	if err != nil {
		fmt.Fprintf(stderr, "firmware bootstrap flash failed: %v\n", err)
		return firmwarecheck.BootstrapFlashPlanResult{}, 1
	}
	if outputDir != "" {
		reportPath, err := writeFirmwareBootstrapFlashPlanReport(outputDir, result)
		if err != nil {
			fmt.Fprintf(stderr, "write firmware bootstrap flash plan: %v\n", err)
			return firmwarecheck.BootstrapFlashPlanResult{}, 1
		}
		result.ReportPath = reportPath
	}
	return result, 0
}
func firmwareBootstrapFlashCommand(plan firmwarecheck.BootstrapFlashPlanResult) ([]string, error) {
	if plan.GuardID != "a21.firmware.bootstrap_flash_plan.v1" || !plan.OK {
		return nil, fmt.Errorf("invalid bootstrap flash plan")
	}
	if plan.Port == "" || len(plan.Parts) != 4 {
		return nil, fmt.Errorf("bootstrap flash plan missing port or image parts")
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
			return nil, fmt.Errorf("bootstrap flash plan contains empty image part")
		}
		args = append(args, part.Offset, part.Path)
	}
	return args, nil
}
func runFirmwareBootstrapFlashCommandExec(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("empty bootstrap flash command")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
func redactBootstrapFlashCommand(args []string) []string {
	return append([]string(nil), args...)
}
func writeJSONFirmwareFlashPlan(writer io.Writer, result firmwarecheck.FlashPlanResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
func writeJSONFirmwareBootstrapFlashPlan(writer io.Writer, result firmwarecheck.BootstrapFlashPlanResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
func writeJSONFirmwareBootstrapFlashExecution(writer io.Writer, report firmwareBootstrapFlashExecutionReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
