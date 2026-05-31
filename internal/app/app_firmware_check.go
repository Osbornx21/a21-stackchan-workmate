package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type firmwareArtifactPrunePlanSummary struct {
	SchemaVersion       string `json:"schema_version"`
	DryRun              bool   `json:"dry_run"`
	DeleteAllowed       bool   `json:"delete_allowed"`
	ArtifactDir         string `json:"artifact_dir"`
	Commit              string `json:"commit"`
	KeepRecent          int    `json:"keep_recent"`
	CurrentArtifactPath string `json:"current_artifact_path"`
	TotalArtifacts      int    `json:"total_artifacts"`
	ReleaseValidCount   int    `json:"release_valid_count"`
	KeepCount           int    `json:"keep_count"`
	PruneCandidateCount int    `json:"prune_candidate_count"`
	ManualReviewCount   int    `json:"manual_review_count"`
	ReportPath          string `json:"report_path,omitempty"`
}

func runFirmwareCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	kind := "manifest"
	passThrough := make([]string, 0, len(args))
	showHelp := len(args) == 0
	explicitKind := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			if explicitKind {
				passThrough = append(passThrough, args[i])
			} else {
				showHelp = true
			}
		case "--kind":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--kind requires a value")
				return 2
			}
			i++
			kind = args[i]
			explicitKind = true
		default:
			passThrough = append(passThrough, args[i])
		}
	}
	if showHelp && !explicitKind && len(passThrough) == 0 {
		fmt.Fprintln(stdout, "a21 firmware-check --kind manifest|artifact|current-artifact|upload|device [kind options]")
		return 0
	}
	return dispatchFirmwareCheck(kind, passThrough, stdout, stderr)
}

func runDeprecatedFirmwareCheckAlias(args []string, stdout io.Writer, stderr io.Writer) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	kind, ok := firmwareCheckAliasKind(args[0])
	if !ok {
		return 0, false
	}
	return dispatchFirmwareCheck(kind, args[1:], stdout, stderr), true
}

func firmwareCheckAliasKind(command string) (string, bool) {
	switch command {
	case "firmware-artifact-check":
		return "artifact", true
	case "firmware-current-artifact-check":
		return "current-artifact", true
	case "firmware-upload-check":
		return "upload", true
	case "firmware-device-check":
		return "device", true
	default:
		return "", false
	}
}

func dispatchFirmwareCheck(kind string, args []string, stdout io.Writer, stderr io.Writer) int {
	switch normalizeFirmwareCheckKind(kind) {
	case "manifest":
		return runFirmwareManifestCheck(args, stdout, stderr)
	case "artifact":
		return runFirmwareArtifactCheck(args, stdout, stderr)
	case "current-artifact", "current":
		return runFirmwareCurrentArtifactCheck(args, stdout, stderr)
	case "upload":
		return runFirmwareUploadCheck(args, stdout, stderr)
	case "device":
		return runFirmwareDeviceCheck(args, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown firmware check kind %q\n", kind)
		return 2
	}
}

func normalizeFirmwareCheckKind(kind string) string {
	normalized := strings.ToLower(strings.TrimSpace(kind))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.TrimPrefix(normalized, "firmware-")
	normalized = strings.TrimSuffix(normalized, "-check")
	return normalized
}

func runFirmwareManifestCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	manifestPath := "firmware/stackchan/a21-firmware.json"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-check --kind manifest --manifest firmware/stackchan/a21-firmware.json")
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
		ManifestPath:    "firmware/stackchan/a21-firmware.json",
		InputPath:       "firmware/stackchan/.pio/build/a21_stackchan_cores3/firmware.bin",
		OutputDir:       "firmware/artifacts",
		Commit:          "unknown",
		Timestamp:       time.Now().Format("20060102-150405"),
		PlatformIOEnv:   firmwarecheck.StackChanPlatformIOEnv,
		PlatformIOBoard: "m5stack-cores3",
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
		case "--platformio-env":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--platformio-env requires a value")
				return 2
			}
			i++
			options.PlatformIOEnv = args[i]
		case "--platformio-board":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--platformio-board requires a value")
				return 2
			}
			i++
			options.PlatformIOBoard = args[i]
		default:
			fmt.Fprintf(stderr, "unknown firmware-package option %q\n", args[i])
			return 2
		}
	}

	sourceState, err := detectFirmwareSourceState()
	if err != nil {
		fmt.Fprintf(stderr, "firmware package failed: source tree check failed: %v\n", err)
		return 1
	}
	if !sourceState.Clean {
		fmt.Fprintf(stderr, "firmware package failed: source tree is dirty under %s\n%s\n", sourceState.Root, sourceState.Detail)
		return 1
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
			fmt.Fprintln(stdout, "a21 firmware-check --kind artifact --artifact firmware/artifacts/<a21-stackchan...bin>")
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
func runFirmwareCurrentArtifactCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.LatestArtifactOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		ArtifactDir:  "firmware/artifacts",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-check --kind current-artifact --commit <expected-git-commit> [--artifact-dir firmware/artifacts]")
			return 0
		case "--manifest":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--manifest requires a value")
				return 2
			}
			i++
			options.ManifestPath = args[i]
		case "--artifact-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--artifact-dir requires a value")
				return 2
			}
			i++
			options.ArtifactDir = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
		default:
			fmt.Fprintf(stderr, "unknown firmware-current-artifact-check option %q\n", args[i])
			return 2
		}
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	result, err := firmwarecheck.ValidateLatestArtifactForCommit(options)
	if err != nil {
		fmt.Fprintf(stderr, "firmware current artifact check failed: %v\n", err)
		return 1
	}
	if err := writeJSONFirmwareArtifact(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware current artifact result: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware current artifact ok")
	return 0
}
func runFirmwareArtifactPrunePlan(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.ArtifactPrunePlanOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		ArtifactDir:  "firmware/artifacts",
		KeepRecent:   5,
	}
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-artifact-prune-plan --commit <expected-git-commit> [--artifact-dir firmware/artifacts] [--keep-recent 5] [--output-dir reports]")
			return 0
		case "--manifest":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--manifest requires a value")
				return 2
			}
			i++
			options.ManifestPath = args[i]
		case "--artifact-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--artifact-dir requires a value")
				return 2
			}
			i++
			options.ArtifactDir = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
		case "--keep-recent":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--keep-recent requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value < 1 {
				fmt.Fprintln(stderr, "--keep-recent must be a positive integer")
				return 2
			}
			options.KeepRecent = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown firmware-artifact-prune-plan option %q\n", args[i])
			return 2
		}
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	result, err := firmwarecheck.BuildArtifactPrunePlan(options)
	if err != nil {
		fmt.Fprintf(stderr, "firmware artifact prune plan failed: %v\n", err)
		return 1
	}
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			fmt.Fprintf(stderr, "create firmware artifact prune plan report dir: %v\n", err)
			return 1
		}
		reportPath := filepath.Join(outputDir, "a21-firmware-artifact-prune-plan-"+time.Now().Format("20060102-150405")+".json")
		file, err := os.Create(reportPath)
		if err != nil {
			fmt.Fprintf(stderr, "create firmware artifact prune plan report: %v\n", err)
			return 1
		}
		result.ReportPath = reportPath
		if err := writeJSONFirmwareArtifactPrunePlan(file, result); err != nil {
			_ = file.Close()
			fmt.Fprintf(stderr, "write firmware artifact prune plan report: %v\n", err)
			return 1
		}
		if err := file.Close(); err != nil {
			fmt.Fprintf(stderr, "close firmware artifact prune plan report: %v\n", err)
			return 1
		}
	}
	if outputDir != "" {
		if err := writeJSONFirmwareArtifactPrunePlanSummary(stdout, summarizeFirmwareArtifactPrunePlan(result)); err != nil {
			fmt.Fprintf(stderr, "encode firmware artifact prune plan summary: %v\n", err)
			return 1
		}
	} else {
		if err := writeJSONFirmwareArtifactPrunePlan(stdout, result); err != nil {
			fmt.Fprintf(stderr, "encode firmware artifact prune plan result: %v\n", err)
			return 1
		}
	}
	fmt.Fprintln(stdout, "firmware artifact prune plan ok (no files deleted)")
	return 0
}
func summarizeFirmwareArtifactPrunePlan(plan firmwarecheck.ArtifactPrunePlan) firmwareArtifactPrunePlanSummary {
	return firmwareArtifactPrunePlanSummary{
		SchemaVersion:       "a21.firmware.artifact_prune_plan.summary.v1",
		DryRun:              plan.DryRun,
		DeleteAllowed:       plan.DeleteAllowed,
		ArtifactDir:         plan.ArtifactDir,
		Commit:              plan.Commit,
		KeepRecent:          plan.KeepRecent,
		CurrentArtifactPath: plan.CurrentArtifactPath,
		TotalArtifacts:      plan.TotalArtifacts,
		ReleaseValidCount:   plan.ReleaseValidCount,
		KeepCount:           len(plan.Keep),
		PruneCandidateCount: len(plan.PruneCandidates),
		ManualReviewCount:   len(plan.ManualReview),
		ReportPath:          plan.ReportPath,
	}
}
func runFirmwareUploadCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.UploadCheckOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-check --kind upload --artifact firmware/artifacts/<a21-stackchan...bin> --commit <expected-git-commit> --port /dev/cu.usbmodemXXXX")
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
	for _, path := range []string{options.ManifestPath, options.ArtifactPath} {
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "firmware upload check path invalid: %v\n", err)
			return 1
		}
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
	if !portUsage.Exists {
		fmt.Fprintf(stderr, "firmware upload check failed: upload port %s does not exist\n", options.Port)
		return 1
	}
	if portUsage.InUse {
		fmt.Fprintf(stderr, "firmware upload check failed: upload port %s is already in use: %s\n", options.Port, portUsage.Detail)
		return 1
	}
	result.PortUsage = portUsage
	if err := writeJSONFirmwareUploadCheck(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware upload check result: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware upload dry-run guard ok (no flash performed)")
	return 0
}
func runFirmwareDeviceCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.DeviceIdentityOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-check --kind device --artifact firmware/artifacts/<a21-stackchan...bin> --device-report reports/devices.json --device-id stackchan-001 --commit <git-sha> --max-device-age-ms 300000")
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
		default:
			fmt.Fprintf(stderr, "unknown firmware-device-check option %q\n", args[i])
			return 2
		}
	}
	if options.ArtifactPath == "" {
		fmt.Fprintln(stderr, "--artifact requires a value")
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
			fmt.Fprintf(stderr, "firmware device check path invalid: %v\n", err)
			return 1
		}
	}
	result, err := firmwarecheck.ValidateDeviceIdentity(options)
	if err != nil {
		fmt.Fprintf(stderr, "firmware device check failed: %v\n", err)
		return 1
	}
	if err := writeJSONFirmwareDeviceCheck(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware device check result: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware device identity guard ok (no flash performed)")
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
func writeJSONFirmwareArtifactPrunePlan(writer io.Writer, result firmwarecheck.ArtifactPrunePlan) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
func writeJSONFirmwareArtifactPrunePlanSummary(writer io.Writer, result firmwareArtifactPrunePlanSummary) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
func writeJSONFirmwareUploadCheck(writer io.Writer, result firmwarecheck.UploadCheckResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
func writeJSONFirmwareDeviceCheck(writer io.Writer, result firmwarecheck.DeviceIdentityResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
