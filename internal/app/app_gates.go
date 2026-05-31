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
	"strings"
	"time"
)

var runA21ControlGuard = runA21ControlGuardExec

func runA21ControlGuardExec(ctx context.Context, input runtimeguard.ControlGuardInput) runtimeguard.ControlGuardReport {
	if input.Config.ProjectName == "" {
		input.Config = runtimeguard.DefaultConfig()
	}
	if strings.TrimSpace(input.CWD) == "" {
		if cwd, err := os.Getwd(); err == nil {
			input.CWD = cwd
		}
	}
	if input.Runner == nil {
		input.Runner = runtimeguard.OSRunner{}
	}
	return runtimeguard.EvaluateControlGuard(ctx, input)
}
func runControlGuard(args []string, stdout io.Writer, stderr io.Writer) int {
	input := runtimeguard.ControlGuardInput{
		Config: runtimeguard.DefaultConfig(),
		Env:    os.Environ(),
		Runner: runtimeguard.OSRunner{},
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 control-guard --command <a21-command-or-command-with-flags> [--tier T0|T1|T2|T3|T4|T5|T6|T7|T8] [--cwd <path>]")
			return 0
		case "--command":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--command requires a value")
				return 2
			}
			i++
			input.Command = args[i]
		case "--tier":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--tier requires a value")
				return 2
			}
			i++
			input.Tier = args[i]
		case "--cwd":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--cwd requires a value")
				return 2
			}
			i++
			input.CWD = args[i]
		default:
			fmt.Fprintf(stderr, "unknown control-guard option %q\n", args[i])
			return 2
		}
	}
	if strings.TrimSpace(input.Command) == "" {
		fmt.Fprintln(stderr, "--command requires a value")
		return 2
	}
	report := runA21ControlGuard(context.Background(), input)
	if err := writeJSONControlGuard(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode control guard report: %v\n", err)
		return 1
	}
	if !report.Result.OK {
		return 1
	}
	return 0
}
func ensureA21ControlAllowed(command string, stderr io.Writer) int {
	_, code := requireA21ControlAllowed(command, stderr)
	return code
}
func requireA21ControlAllowed(command string, stderr io.Writer) (runtimeguard.ControlGuardReport, int) {
	report := runA21ControlGuard(context.Background(), runtimeguard.ControlGuardInput{
		Config:  runtimeguard.DefaultConfig(),
		Command: command,
		Env:     os.Environ(),
		Runner:  runtimeguard.OSRunner{},
	})
	if report.Result.OK {
		return report, 0
	}
	fmt.Fprintf(stderr, "a21 control guard blocked %s at %s: %s\n", report.Command, report.Tier, summarizeControlFindings(report.Result.Findings))
	return report, 1
}
func summarizeControlFindings(findings []runtimeguard.Finding) string {
	parts := make([]string, 0, len(findings))
	for _, finding := range findings {
		if finding.Severity != runtimeguard.SeverityBlock {
			continue
		}
		if finding.Detail != "" {
			parts = append(parts, finding.Code+"="+finding.Detail)
		} else {
			parts = append(parts, finding.Code)
		}
	}
	if len(parts) == 0 {
		return "unknown control guard block"
	}
	return strings.Join(parts, "; ")
}
func writeJSONControlGuard(writer io.Writer, report runtimeguard.ControlGuardReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
func runPreflight(stdout io.Writer, stderr io.Writer) int {
	report, code := buildPreflightReport(stderr)
	if code != 0 {
		return code
	}
	if err := writeJSONReport(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode preflight report: %v\n", err)
		return 1
	}
	if !report.Result.OK {
		return 1
	}
	return 0
}
func runNamespaceAudit(stdout io.Writer, stderr io.Writer) int {
	paths, err := gitTrackedFiles()
	if err != nil {
		fmt.Fprintf(stderr, "namespace audit failed: %v\n", err)
		return 1
	}
	report := runtimeguard.AuditNamespacePaths(paths)
	if err := writeJSONNamespaceAudit(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode namespace audit report: %v\n", err)
		return 1
	}
	if !report.Result.OK {
		return 1
	}
	return 0
}
func gitTrackedFiles() ([]string, error) {
	output, err := exec.Command("git", "ls-files").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(output), "\n")
	paths := make([]string, 0, len(lines))
	for _, line := range lines {
		path := strings.TrimSpace(line)
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths, nil
}
func runDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	outputDir := "reports"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 doctor --output-dir reports")
			return 0
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown doctor option %q\n", args[i])
			return 2
		}
	}

	preflight, code := buildPreflightReport(stderr)
	if code != 0 {
		return code
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "get working directory: %v\n", err)
		return 1
	}
	projectRoot := findProjectRoot(cwd)
	report := buildDoctorReport(preflight, projectRoot, currentGitCommit(projectRoot))
	if err := writeJSONDoctorReport(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode doctor report: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "create doctor report directory: %v\n", err)
		return 1
	}
	reportPath := filepath.Join(outputDir, "a21-doctor-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "create doctor report: %v\n", err)
		return 1
	}
	defer file.Close()
	if err := writeJSONDoctorReport(file, report); err != nil {
		fmt.Fprintf(stderr, "write doctor report: %v\n", err)
		return 1
	}
	if !report.Result.OK {
		return 1
	}
	return 0
}
func runSerialList(args []string, stdout io.Writer, stderr io.Writer) int {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 serial-list")
			return 0
		default:
			fmt.Fprintf(stderr, "unknown serial-list option %q\n", args[i])
			return 2
		}
	}
	devices, err := listFirmwareSerialDevices()
	if err != nil {
		fmt.Fprintf(stderr, "serial-list failed: %v\n", err)
		return 1
	}
	payload := struct {
		SerialDevices []firmwarecheck.SerialDevice `json:"serial_devices"`
	}{SerialDevices: devices}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		fmt.Fprintf(stderr, "encode serial-list result: %v\n", err)
		return 1
	}
	return 0
}
func buildPreflightReport(stderr io.Writer) (runtimeguard.PreflightReport, int) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "get working directory: %v\n", err)
		return runtimeguard.PreflightReport{}, 1
	}
	return runtimeguard.RunPreflightReport(context.Background(), runtimeguard.PreflightInput{
		Config: runtimeguard.DefaultConfig(),
		Env:    os.Environ(),
		CWD:    cwd,
		Runner: runtimeguard.OSRunner{},
	}), 0
}
func writeJSONNamespaceAudit(writer io.Writer, report runtimeguard.NamespaceAuditReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
func writeJSONDoctorReport(writer io.Writer, report doctorReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
