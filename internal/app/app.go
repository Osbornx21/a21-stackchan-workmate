package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/buildinfo"
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/runtimeguard"
)

var detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
	return firmwarecheck.DetectPortUsage(context.Background(), port)
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		args = []string{"version"}
	}

	switch args[0] {
	case "version":
		fmt.Fprintf(stdout, "%s %s (%s)\n", buildinfo.ProjectName, buildinfo.Version, buildinfo.ServiceName)
		return 0
	case "gateway":
		return runGateway(args[1:], stdout, stderr)
	case "preflight":
		return runPreflight(stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "firmware-check":
		return runFirmwareCheck(args[1:], stdout, stderr)
	case "firmware-package":
		return runFirmwarePackage(args[1:], stdout, stderr)
	case "firmware-artifact-check":
		return runFirmwareArtifactCheck(args[1:], stdout, stderr)
	case "firmware-upload-check":
		return runFirmwareUploadCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
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

func writeJSONReport(writer io.Writer, report runtimeguard.PreflightReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONDoctorReport(writer io.Writer, report doctorReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func runGateway(args []string, stdout io.Writer, stderr io.Writer) int {
	addr := "127.0.0.1:21080"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 gateway --addr 127.0.0.1:21080")
			return 0
		case "--addr":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--addr requires a value")
				return 2
			}
			i++
			addr = args[i]
		default:
			fmt.Fprintf(stderr, "unknown gateway option %q\n", args[i])
			return 2
		}
	}

	fmt.Fprintf(stdout, "a21 gateway listening on %s\n", addr)
	if err := http.ListenAndServe(addr, gateway.NewServer().Handler()); err != nil {
		fmt.Fprintf(stderr, "gateway: %v\n", err)
		return 1
	}
	return 0
}

func runFirmwareCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	manifestPath := "firmware/stackchan/a21-firmware.json"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-check --manifest firmware/stackchan/a21-firmware.json")
			return 0
		case "--manifest":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--manifest requires a value")
				return 2
			}
			i++
			manifestPath = args[i]
		default:
			fmt.Fprintf(stderr, "unknown firmware-check option %q\n", args[i])
			return 2
		}
	}

	result, err := firmwarecheck.LoadAndValidate(manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "firmware manifest check failed: %v\n", err)
		return 1
	}
	if err := writeJSONFirmwareCheck(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware check result: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware manifest ok")
	return 0
}

func runFirmwarePackage(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.PackageOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		InputPath:    "firmware/stackchan/.pio/build/a21_stackchan_cores3/firmware.bin",
		OutputDir:    "firmware/artifacts",
		Commit:       "unknown",
		Timestamp:    time.Now().Format("20060102-150405"),
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-package --commit <git-sha>")
			return 0
		case "--manifest":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--manifest requires a value")
				return 2
			}
			i++
			options.ManifestPath = args[i]
		case "--input":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--input requires a value")
				return 2
			}
			i++
			options.InputPath = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
		case "--timestamp":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--timestamp requires a value")
				return 2
			}
			i++
			options.Timestamp = args[i]
		default:
			fmt.Fprintf(stderr, "unknown firmware-package option %q\n", args[i])
			return 2
		}
	}

	result, err := firmwarecheck.PackageArtifact(options)
	if err != nil {
		fmt.Fprintf(stderr, "firmware package failed: %v\n", err)
		return 1
	}
	if err := writeJSONFirmwarePackage(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware package result: %v\n", err)
		return 1
	}
	return 0
}

func runFirmwareArtifactCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.ArtifactOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-artifact-check --artifact firmware/artifacts/<a21-stackchan...bin>")
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
		default:
			fmt.Fprintf(stderr, "unknown firmware-artifact-check option %q\n", args[i])
			return 2
		}
	}
	result, err := firmwarecheck.ValidateArtifact(options)
	if err != nil {
		fmt.Fprintf(stderr, "firmware artifact check failed: %v\n", err)
		return 1
	}
	if err := writeJSONFirmwareArtifact(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware artifact result: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware artifact ok")
	return 0
}

func runFirmwareUploadCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.UploadCheckOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-upload-check --artifact firmware/artifacts/<a21-stackchan...bin> --port /dev/cu.usbmodemXXXX")
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
			options.Commit = args[i]
		default:
			fmt.Fprintf(stderr, "unknown firmware-upload-check option %q\n", args[i])
			return 2
		}
	}
	if options.Port == "" {
		fmt.Fprintln(stderr, "--port requires a value")
		return 2
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	result, err := firmwarecheck.ValidateUploadCandidate(options)
	if err != nil {
		fmt.Fprintf(stderr, "firmware upload check failed: %v\n", err)
		return 1
	}
	portUsage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		fmt.Fprintf(stderr, "firmware upload check failed: upload port ownership check failed: %v\n", err)
		return 1
	}
	if portUsage.InUse {
		fmt.Fprintf(stderr, "firmware upload check failed: upload port %s is already in use: %s\n", options.Port, portUsage.Detail)
		return 1
	}
	if err := writeJSONFirmwareUploadCheck(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware upload check result: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware upload guard ok")
	return 0
}

func writeJSONFirmwareCheck(writer io.Writer, result firmwarecheck.Result) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func writeJSONFirmwarePackage(writer io.Writer, result firmwarecheck.PackageResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func writeJSONFirmwareArtifact(writer io.Writer, result firmwarecheck.ArtifactResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func writeJSONFirmwareUploadCheck(writer io.Writer, result firmwarecheck.UploadCheckResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
