package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/runtimeguard"
)

type firmwareIMUProbeFlashExecutionReport struct {
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
	Plan           firmwarecheck.IMUProbeFlashPlanResult `json:"plan"`
	Command        []string                              `json:"command"`
	ReportPath     string                                `json:"report_path,omitempty"`
}

func writeFirmwareIMUProbeFlashPlanReport(outputDir string, result firmwarecheck.IMUProbeFlashPlanResult) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-firmware-imu-probe-flash-plan-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	result.ReportPath = reportPath
	if err := writeJSONFirmwareIMUProbeFlashPlan(file, result); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeFirmwareIMUProbeFlashExecutionReport(outputDir string, report firmwareIMUProbeFlashExecutionReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-firmware-imu-probe-flash-execution-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONFirmwareIMUProbeFlashExecution(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func runFirmwareIMUProbeFlashPlan(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.IMUProbeFlashPlanOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		ArtifactPath: filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanIMUProbePlatformIOEnv, "firmware.bin"),
		BuildDir:     filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanIMUProbePlatformIOEnv),
		CoreDir:      filepath.Join(".a21-tools", "platformio-core"),
	}
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-imu-probe-flash --port /dev/cu.usbmodemXXXX --commit <git-sha> [--execute --confirm WRITE_A21_STACKCHAN_IMU_PROBE_FIRMWARE] [--artifact firmware/stackchan/.pio/build/a21_stackchan_cores3_imu_probe/firmware.bin] [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3_imu_probe] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
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
			fmt.Fprintf(stderr, "unknown firmware-imu-probe-flash-plan option %q\n", args[i])
			return 2
		}
	}
	result, code := buildFirmwareIMUProbeFlashPlanFromCLI(options, outputDir, stderr)
	if code != 0 {
		return code
	}
	if err := writeJSONFirmwareIMUProbeFlashPlan(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware IMU probe flash plan: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware IMU probe flash plan ok (no flash performed)")
	return 0
}
func runFirmwareIMUProbeFlashExecute(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.IMUProbeFlashPlanOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		ArtifactPath: filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanIMUProbePlatformIOEnv, "firmware.bin"),
		BuildDir:     filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanIMUProbePlatformIOEnv),
		CoreDir:      filepath.Join(".a21-tools", "platformio-core"),
	}
	outputDir := ""
	confirm := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-imu-probe-flash --execute --port /dev/cu.usbmodemXXXX --commit <git-sha> --confirm WRITE_A21_STACKCHAN_IMU_PROBE_FIRMWARE [--artifact firmware/stackchan/.pio/build/a21_stackchan_cores3_imu_probe/firmware.bin] [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3_imu_probe] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
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
				fmt.Fprintln(stderr, "--confirm requires WRITE_A21_STACKCHAN_IMU_PROBE_FIRMWARE")
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
			fmt.Fprintf(stderr, "unknown firmware-imu-probe-flash-execute option %q\n", args[i])
			return 2
		}
	}
	if confirm != "WRITE_A21_STACKCHAN_IMU_PROBE_FIRMWARE" {
		fmt.Fprintln(stderr, "--confirm WRITE_A21_STACKCHAN_IMU_PROBE_FIRMWARE is required")
		return 2
	}
	controlGuard, code := requireA21ControlAllowed("firmware-imu-probe-flash --execute", stderr)
	if code != 0 {
		return code
	}
	plan, code := buildFirmwareIMUProbeFlashPlanFromCLI(options, "", stderr)
	if code != 0 {
		return code
	}
	command, err := firmwareIMUProbeFlashCommand(plan)
	if err != nil {
		fmt.Fprintf(stderr, "firmware IMU probe flash execute failed: %v\n", err)
		return 1
	}
	if err := runFirmwareBootstrapFlashCommand(context.Background(), command, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "firmware IMU probe flash execute failed: %v\n", err)
		return 1
	}
	report := firmwareIMUProbeFlashExecutionReport{
		SchemaVersion:  "a21.firmware.imu_probe_flash_execution.v1",
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
			fmt.Fprintf(stderr, "firmware IMU probe flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeFirmwareIMUProbeFlashExecutionReport(outputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write firmware IMU probe flash execution report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONFirmwareIMUProbeFlashExecution(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode firmware IMU probe flash execution: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware IMU probe flash executed")
	return 0
}
func buildFirmwareIMUProbeFlashPlanFromCLI(options firmwarecheck.IMUProbeFlashPlanOptions, outputDir string, stderr io.Writer) (firmwarecheck.IMUProbeFlashPlanResult, int) {
	if options.Port == "" {
		fmt.Fprintln(stderr, "--port requires a value")
		return firmwarecheck.IMUProbeFlashPlanResult{}, 2
	}
	if options.ExpectedGitCommit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return firmwarecheck.IMUProbeFlashPlanResult{}, 2
	}
	for _, path := range []string{options.ManifestPath, options.ArtifactPath, options.BuildDir, options.CoreDir} {
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "firmware IMU probe flash path invalid: %v\n", err)
			return firmwarecheck.IMUProbeFlashPlanResult{}, 1
		}
	}
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "firmware IMU probe flash report dir invalid: %v\n", err)
			return firmwarecheck.IMUProbeFlashPlanResult{}, 1
		}
	}
	sourceState, err := detectFirmwareSourceState()
	if err != nil {
		fmt.Fprintf(stderr, "firmware IMU probe flash failed: source tree check failed: %v\n", err)
		return firmwarecheck.IMUProbeFlashPlanResult{}, 1
	}
	if !sourceState.Clean {
		fmt.Fprintf(stderr, "firmware IMU probe flash failed: source tree is dirty under %s\n%s\n", sourceState.Root, sourceState.Detail)
		return firmwarecheck.IMUProbeFlashPlanResult{}, 1
	}
	portUsage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		fmt.Fprintf(stderr, "firmware IMU probe flash failed: upload port ownership check failed: %v\n", err)
		return firmwarecheck.IMUProbeFlashPlanResult{}, 1
	}
	options.PortUsage = portUsage
	result, err := firmwarecheck.BuildIMUProbeFlashPlan(options)
	if err != nil {
		fmt.Fprintf(stderr, "firmware IMU probe flash failed: %v\n", err)
		return firmwarecheck.IMUProbeFlashPlanResult{}, 1
	}
	if outputDir != "" {
		reportPath, err := writeFirmwareIMUProbeFlashPlanReport(outputDir, result)
		if err != nil {
			fmt.Fprintf(stderr, "write firmware IMU probe flash plan: %v\n", err)
			return firmwarecheck.IMUProbeFlashPlanResult{}, 1
		}
		result.ReportPath = reportPath
	}
	return result, 0
}
func firmwareIMUProbeFlashCommand(plan firmwarecheck.IMUProbeFlashPlanResult) ([]string, error) {
	if plan.GuardID != "a21.firmware.imu_probe_flash_plan.v1" || !plan.OK {
		return nil, fmt.Errorf("invalid IMU probe flash plan")
	}
	if plan.PlatformIOEnv != firmwarecheck.StackChanIMUProbePlatformIOEnv {
		return nil, fmt.Errorf("IMU probe flash plan has wrong PlatformIO env")
	}
	if plan.Port == "" || len(plan.Parts) != 4 {
		return nil, fmt.Errorf("IMU probe flash plan missing port or image parts")
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
			return nil, fmt.Errorf("IMU probe flash plan contains empty image part")
		}
		args = append(args, part.Offset, part.Path)
	}
	return args, nil
}
func writeJSONFirmwareIMUProbeFlashPlan(writer io.Writer, result firmwarecheck.IMUProbeFlashPlanResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
func writeJSONFirmwareIMUProbeFlashExecution(writer io.Writer, report firmwareIMUProbeFlashExecutionReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
