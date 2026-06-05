package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
)

func runProviderSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	provider := ""
	execute := false
	stream := false
	repeat := 1
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 provider-smoke --provider <provider> [--execute] [--stream] [--repeat 3] [--output-dir reports]")
			return 0
		case "--provider":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--provider requires a value")
				return 2
			}
			i++
			provider = args[i]
		case "--execute":
			execute = true
		case "--stream":
			stream = true
		case "--repeat":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--repeat requires a value")
				return 2
			}
			i++
			parsed, err := strconv.Atoi(args[i])
			if err != nil || parsed <= 0 || parsed > 10 {
				fmt.Fprintln(stderr, "--repeat requires an integer between 1 and 10")
				return 2
			}
			repeat = parsed
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown provider-smoke option %q\n", args[i])
			return 2
		}
	}
	report := providers.ProviderSmokeFromEnvWithOptions(context.Background(), os.Environ(), providers.ProviderSmokeOptions{
		ProviderName: provider,
		Execute:      execute,
		Stream:       stream,
		Repeat:       repeat,
	})
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "provider smoke report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeProviderSmokeReport(outputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write provider smoke report: %v\n", err)
			return 1
		}
		report.ReportPath = filepath.Base(filepath.Clean(reportPath))
	}
	if err := writeJSONProviderSmoke(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode provider smoke report: %v\n", err)
		return 1
	}
	if report.Status == providers.ProviderSmokeFailed {
		return 1
	}
	if execute && report.Status != providers.ProviderSmokePassed {
		return 1
	}
	return 0
}
func runProviderRealtimePlan(args []string, stdout io.Writer, stderr io.Writer) int {
	provider := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 provider-realtime-plan [--provider <provider>]")
			return 0
		case "--provider":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--provider requires a value")
				return 2
			}
			i++
			provider = args[i]
		case "--execute":
			fmt.Fprintln(stderr, "provider-realtime-plan does not support --execute; use a future explicit realtime smoke command")
			return 2
		default:
			fmt.Fprintf(stderr, "unknown provider-realtime-plan option %q\n", args[i])
			return 2
		}
	}
	report := providers.RealtimeWebSocketPlanFromEnv(os.Environ(), provider)
	if err := writeJSONProviderSmoke(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode provider realtime plan: %v\n", err)
		return 1
	}
	if report.Status == providers.ProviderSmokeFailed {
		return 1
	}
	return 0
}
func runProviderRealtimeFixture(args []string, stdout io.Writer, stderr io.Writer) int {
	provider := ""
	execute := false
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 provider-realtime-fixture [--provider <provider>] [--execute] [--output-dir reports]")
			return 0
		case "--provider":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--provider requires a value")
				return 2
			}
			i++
			provider = args[i]
		case "--execute":
			execute = true
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown provider-realtime-fixture option %q\n", args[i])
			return 2
		}
	}
	report := providers.RealtimeFixtureSmokeFromEnv(context.Background(), os.Environ(), provider, execute)
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "provider realtime fixture report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeProviderRealtimeFixtureReport(outputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write provider realtime fixture report: %v\n", err)
			return 1
		}
		report.ReportPath = filepath.Base(filepath.Clean(reportPath))
	}
	if err := writeJSONProviderSmoke(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode provider realtime fixture report: %v\n", err)
		return 1
	}
	if report.Status == providers.ProviderSmokeFailed {
		return 1
	}
	if execute && report.Status != providers.ProviderSmokePassed {
		return 1
	}
	return 0
}
func runV21AdapterSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	adapterURL := strings.TrimSpace(os.Getenv("A21_V21_ADAPTER_URL"))
	query := ""
	execute := false
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 v21-adapter-smoke [--adapter-url http://127.0.0.1:21121] [--query <professional-query>] [--execute] [--output-dir reports]")
			return 0
		case "--adapter-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--adapter-url requires a value")
				return 2
			}
			i++
			adapterURL = args[i]
		case "--query":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--query requires a value")
				return 2
			}
			i++
			query = args[i]
		case "--execute":
			execute = true
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown v21-adapter-smoke option %q\n", args[i])
			return 2
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	report := v21adapter.Smoke(ctx, adapterURL, query, execute, nil)
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "v21 adapter smoke report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeV21AdapterSmokeReport(outputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write v21 adapter smoke report: %v\n", err)
			return 1
		}
		report.ReportPath = filepath.Base(filepath.Clean(reportPath))
	}
	if err := writeJSONV21AdapterSmoke(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode v21 adapter smoke report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	if execute && report.Status != "passed" {
		return 1
	}
	return 0
}

func runV21ProfessionalReadiness(args []string, stdout io.Writer, stderr io.Writer) int {
	adapterURL := strings.TrimSpace(os.Getenv("A21_V21_ADAPTER_URL"))
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 v21-professional-readiness [--adapter-url http://127.0.0.1:21121] [--output-dir reports]")
			return 0
		case "--adapter-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--adapter-url requires a value")
				return 2
			}
			i++
			adapterURL = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		case "--execute":
			fmt.Fprintln(stderr, "v21-professional-readiness is host-only and does not support --execute")
			return 2
		default:
			fmt.Fprintf(stderr, "unknown v21-professional-readiness option %q\n", args[i])
			return 2
		}
	}
	report := v21adapter.ProfessionalReadiness(context.Background(), adapterURL, nil)
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "v21 professional readiness report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeV21ProfessionalReadinessReport(outputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write v21 professional readiness report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONV21ProfessionalReadiness(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode v21 professional readiness report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}

func writeProviderSmokeReport(outputDir string, report providers.ProviderSmokeReport) (string, error) {
	return writeProviderSmokeReportWithPrefix(outputDir, "a21-provider-smoke", report)
}

func writeProviderRealtimeFixtureReport(outputDir string, report providers.ProviderSmokeReport) (string, error) {
	return writeProviderSmokeReportWithPrefix(outputDir, "a21-provider-realtime-fixture", report)
}

func writeProviderSmokeReportWithPrefix(outputDir string, prefix string, report providers.ProviderSmokeReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	ensureProviderSmokeReportIdentity(&report)
	var reportPath string
	var file *os.File
	var err error
	for attempt := 0; attempt < 100; attempt++ {
		now := time.Now()
		stamp := fmt.Sprintf("%s-%09d", now.Format("20060102-150405"), now.Nanosecond())
		if attempt > 0 {
			stamp = fmt.Sprintf("%s-%02d", stamp, attempt)
		}
		reportPath = filepath.Join(outputDir, prefix+"-"+stamp+".json")
		file, err = os.OpenFile(reportPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		break
	}
	if file == nil {
		return "", fmt.Errorf("could not allocate unique provider report path")
	}
	defer file.Close()
	report.ReportPath = filepath.Base(filepath.Clean(reportPath))
	if err := writeJSONProviderSmoke(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func ensureProviderSmokeReportIdentity(report *providers.ProviderSmokeReport) {
	if report.SchemaVersion == "" {
		report.SchemaVersion = providers.ProviderSmokeSchemaVersion
	}
	if report.GeneratedAtMS <= 0 {
		report.GeneratedAtMS = time.Now().UnixMilli()
	}
}
func writeV21AdapterSmokeReport(outputDir string, report v21adapter.SmokeReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-v21-adapter-smoke-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	ensureV21AdapterSmokeReportIdentity(&report)
	report.ReportPath = filepath.Base(filepath.Clean(reportPath))
	if err := writeJSONV21AdapterSmoke(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func ensureV21AdapterSmokeReportIdentity(report *v21adapter.SmokeReport) {
	if report.SchemaVersion == "" {
		report.SchemaVersion = v21adapter.SmokeSchemaVersion
	}
	if report.GeneratedAtMS <= 0 {
		report.GeneratedAtMS = time.Now().UnixMilli()
	}
}

func writeV21ProfessionalReadinessReport(outputDir string, report v21adapter.ProfessionalReadinessReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-v21-professional-readiness-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONV21ProfessionalReadiness(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONProviderSmoke(writer io.Writer, report providers.ProviderSmokeReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
func writeJSONV21AdapterSmoke(writer io.Writer, report v21adapter.SmokeReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONV21ProfessionalReadiness(writer io.Writer, report v21adapter.ProfessionalReadinessReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
