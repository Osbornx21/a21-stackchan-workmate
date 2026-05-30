package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/buildinfo"
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/runtimeguard"
	"a21.local/a21/internal/v21adapter"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

var detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
	return firmwarecheck.DetectPortUsage(context.Background(), port)
}

type firmwareSourceState struct {
	Root   string
	Clean  bool
	Detail string
}

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

var detectFirmwareSourceState = detectGitFirmwareSourceState

func detectGitFirmwareSourceState() (firmwareSourceState, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return firmwareSourceState{}, err
	}
	root := findProjectRoot(cwd)
	output, err := exec.Command("git", "-C", root, "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil {
		return firmwareSourceState{}, err
	}
	detail := strings.TrimSpace(string(output))
	return firmwareSourceState{
		Root:   root,
		Clean:  detail == "",
		Detail: detail,
	}, nil
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
	case "namespace-audit":
		return runNamespaceAudit(stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "provider-smoke":
		return runProviderSmoke(args[1:], stdout, stderr)
	case "provider-realtime-plan":
		return runProviderRealtimePlan(args[1:], stdout, stderr)
	case "provider-realtime-fixture":
		return runProviderRealtimeFixture(args[1:], stdout, stderr)
	case "v21-adapter-smoke":
		return runV21AdapterSmoke(args[1:], stdout, stderr)
	case "lan-probe":
		return runLANProbe(args[1:], stdout, stderr)
	case "audio-front-end-plan":
		return runAudioFrontEndPlan(args[1:], stdout, stderr)
	case "audio-front-end-eval":
		return runAudioFrontEndEval(args[1:], stdout, stderr)
	case "firmware-device-report":
		return runFirmwareDeviceReport(args[1:], stdout, stderr)
	case "office-preflight":
		return runOfficePreflight(args[1:], stdout, stderr)
	case "office-handoff":
		return runOfficeHandoff(args[1:], stdout, stderr)
	case "office-acceptance":
		return runOfficeAcceptance(args[1:], stdout, stderr)
	case "stackchan-identity-acceptance":
		return runStackChanIdentityAcceptance(args[1:], stdout, stderr)
	case "stackchan-physical-evidence":
		return runStackChanPhysicalEvidence(args[1:], stdout, stderr)
	case "stackchan-capability-acceptance":
		return runStackChanCapabilityAcceptance(args[1:], stdout, stderr)
	case "latency-bench":
		return runLatencyBench(args[1:], stdout, stderr)
	case "serial-list":
		return runSerialList(args[1:], stdout, stderr)
	case "firmware-check":
		return runFirmwareCheck(args[1:], stdout, stderr)
	case "firmware-package":
		return runFirmwarePackage(args[1:], stdout, stderr)
	case "firmware-artifact-check":
		return runFirmwareArtifactCheck(args[1:], stdout, stderr)
	case "firmware-current-artifact-check":
		return runFirmwareCurrentArtifactCheck(args[1:], stdout, stderr)
	case "firmware-artifact-prune-plan":
		return runFirmwareArtifactPrunePlan(args[1:], stdout, stderr)
	case "firmware-upload-check":
		return runFirmwareUploadCheck(args[1:], stdout, stderr)
	case "firmware-device-check":
		return runFirmwareDeviceCheck(args[1:], stdout, stderr)
	case "firmware-flash-plan":
		return runFirmwareFlashPlan(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func runProviderSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	provider := ""
	execute := false
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 provider-smoke --provider <provider> [--execute] [--output-dir reports]")
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
			fmt.Fprintf(stderr, "unknown provider-smoke option %q\n", args[i])
			return 2
		}
	}
	report := providers.ProviderSmokeFromEnv(context.Background(), os.Environ(), provider, execute, nil)
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
		report.ReportPath = reportPath
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
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 provider-realtime-fixture [--provider <provider>] [--execute]")
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
		default:
			fmt.Fprintf(stderr, "unknown provider-realtime-fixture option %q\n", args[i])
			return 2
		}
	}
	report := providers.RealtimeFixtureSmokeFromEnv(context.Background(), os.Environ(), provider, execute)
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
		report.ReportPath = reportPath
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

type lanProbeReport struct {
	SchemaVersion string                 `json:"schema_version"`
	OK            bool                   `json:"ok"`
	Metadata      latencyBenchMetadata   `json:"metadata"`
	Targets       []lanProbeTargetReport `json:"targets"`
	ReportPath    string                 `json:"report_path,omitempty"`
}

type lanProbeTargetReport struct {
	Name          string  `json:"name"`
	Endpoint      string  `json:"endpoint,omitempty"`
	Host          string  `json:"host,omitempty"`
	Port          string  `json:"port,omitempty"`
	Status        string  `json:"status"`
	Direct        bool    `json:"direct"`
	Samples       int     `json:"samples"`
	PassedSamples int     `json:"passed_samples"`
	FailedSamples int     `json:"failed_samples"`
	DurationMS    float64 `json:"duration_ms,omitempty"`
	P50MS         float64 `json:"p50_ms"`
	P95MS         float64 `json:"p95_ms"`
	JitterMS      float64 `json:"jitter_ms"`
	ErrorCode     string  `json:"error_code,omitempty"`
}

type lanProbeTarget struct {
	name     string
	endpoint string
	host     string
	port     string
}

func runLANProbe(args []string, stdout io.Writer, stderr io.Writer) int {
	targets := make([]lanProbeTarget, 0)
	outputDir := ""
	timeout := time.Second
	samples := 1
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 lan-probe --target a21-gateway=127.0.0.1:21080 [--target a21-v21-adapter=127.0.0.1:21121] [--samples 5] [--timeout-ms 1000] [--output-dir reports]")
			return 0
		case "--target":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--target requires a value")
				return 2
			}
			i++
			target, err := parseLANProbeTarget(args[i])
			if err != nil {
				fmt.Fprintf(stderr, "lan probe target invalid: %v\n", err)
				return 2
			}
			targets = append(targets, target)
		case "--samples":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--samples requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 {
				fmt.Fprintln(stderr, "--samples must be a positive integer")
				return 2
			}
			samples = value
		case "--timeout-ms":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--timeout-ms requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 {
				fmt.Fprintln(stderr, "--timeout-ms must be a positive integer")
				return 2
			}
			timeout = time.Duration(value) * time.Millisecond
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown lan-probe option %q\n", args[i])
			return 2
		}
	}
	if len(targets) == 0 {
		fmt.Fprintln(stderr, "lan-probe requires at least one --target")
		return 2
	}
	report := runLANProbeTargets(targets, timeout, samples)
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "lan probe report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeLANProbeReport(outputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write lan probe report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONLANProbe(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode lan probe report: %v\n", err)
		return 1
	}
	if !report.OK {
		return 1
	}
	return 0
}

func parseLANProbeTarget(raw string) (lanProbeTarget, error) {
	name, endpoint, ok := strings.Cut(strings.TrimSpace(raw), "=")
	if !ok || strings.TrimSpace(name) == "" || strings.TrimSpace(endpoint) == "" {
		return lanProbeTarget{}, fmt.Errorf("expected name=host:port")
	}
	name = strings.TrimSpace(name)
	endpoint = strings.TrimSpace(endpoint)
	if strings.Contains(strings.ToLower(name), "x21") || strings.Contains(strings.ToLower(endpoint), "x21") {
		return lanProbeTarget{}, fmt.Errorf("target contains forbidden legacy identity")
	}
	if strings.Contains(endpoint, "@") {
		return lanProbeTarget{}, fmt.Errorf("target endpoint must not include credentials")
	}
	if strings.Contains(endpoint, "://") {
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return lanProbeTarget{}, fmt.Errorf("target endpoint is invalid")
		}
		if parsed.User != nil {
			return lanProbeTarget{}, fmt.Errorf("target endpoint must not include credentials")
		}
		endpoint = parsed.Host
	}
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil || strings.TrimSpace(host) == "" || strings.TrimSpace(port) == "" {
		return lanProbeTarget{}, fmt.Errorf("expected host:port endpoint")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber <= 0 || portNumber > 65535 {
		return lanProbeTarget{}, fmt.Errorf("target port is invalid")
	}
	if runtimeguard.DefaultConfig().IsLegacyEndpointPort(portNumber) {
		return lanProbeTarget{}, fmt.Errorf("target uses forbidden legacy internal port")
	}
	return lanProbeTarget{
		name:     name,
		endpoint: net.JoinHostPort(host, port),
		host:     host,
		port:     port,
	}, nil
}

func runLANProbeTargets(targets []lanProbeTarget, timeout time.Duration, samples int) lanProbeReport {
	if samples <= 0 {
		samples = 1
	}
	report := lanProbeReport{
		SchemaVersion: "a21.lan_probe.v1",
		OK:            true,
		Metadata:      buildLatencyBenchMetadata(),
		Targets:       make([]lanProbeTargetReport, 0, len(targets)),
	}
	for _, target := range targets {
		targetReport := lanProbeTargetReport{
			Name:     target.name,
			Endpoint: target.endpoint,
			Host:     target.host,
			Port:     target.port,
			Status:   "failed",
			Direct:   true,
			Samples:  samples,
		}
		durations := make([]time.Duration, 0, samples)
		for sample := 0; sample < samples; sample++ {
			started := time.Now()
			conn, err := (&net.Dialer{Timeout: timeout}).Dial("tcp", target.endpoint)
			duration := time.Since(started)
			if err != nil {
				report.OK = false
				targetReport.FailedSamples++
				targetReport.ErrorCode = "tcp_dial_failed"
				continue
			}
			_ = conn.Close()
			targetReport.PassedSamples++
			durations = append(durations, duration)
		}
		if targetReport.FailedSamples == 0 {
			targetReport.Status = "passed"
		}
		if len(durations) > 0 {
			targetReport.DurationMS = percentileMS(durations, 0.50)
			targetReport.P50MS = percentileMS(durations, 0.50)
			targetReport.P95MS = percentileMS(durations, 0.95)
			targetReport.JitterMS = jitterMS(durations)
		}
		report.Targets = append(report.Targets, targetReport)
	}
	return report
}

func runAudioFrontEndPlan(args []string, stdout io.Writer, stderr io.Writer) int {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 audio-front-end-plan")
			return 0
		case "--execute":
			fmt.Fprintln(stderr, "audio-front-end-plan is plan-only and does not execute audio libraries")
			return 2
		default:
			fmt.Fprintf(stderr, "unknown audio-front-end-plan option %q\n", args[i])
			return 2
		}
	}
	if err := writeJSONAudioFrontEndPlan(stdout, audio.BaselineFrontEndPlan()); err != nil {
		fmt.Fprintf(stderr, "encode audio front-end plan: %v\n", err)
		return 1
	}
	return 0
}

func runAudioFrontEndEval(args []string, stdout io.Writer, stderr io.Writer) int {
	mock := false
	fixturePath := ""
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 audio-front-end-eval (--mock | --fixture reports/a21-audio-fixture.json) [--output-dir reports]")
			return 0
		case "--mock":
			mock = true
		case "--fixture":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--fixture requires a value")
				return 2
			}
			i++
			fixturePath = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		case "--execute":
			fmt.Fprintln(stderr, "audio-front-end-eval only supports --mock until recorded-office and physical-device fixtures exist")
			return 2
		default:
			fmt.Fprintf(stderr, "unknown audio-front-end-eval option %q\n", args[i])
			return 2
		}
	}
	if mock && fixturePath != "" {
		fmt.Fprintln(stderr, "audio-front-end-eval accepts either --mock or --fixture, not both")
		return 2
	}
	if !mock && fixturePath == "" {
		fmt.Fprintln(stderr, "audio-front-end-eval requires --mock or --fixture")
		return 2
	}
	var report audio.FrontEndEvalReport
	if mock {
		report = audio.RunMockFrontEndEval()
	} else {
		var err error
		report, err = audio.RunFrontEndEvalFromFixture(fixturePath)
		if err != nil {
			fmt.Fprintf(stderr, "audio front-end fixture eval failed: %v\n", err)
			return 1
		}
	}
	cliReport := buildAudioFrontEndEvalCLIReport(report)
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "audio front-end report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeAudioFrontEndEvalReport(outputDir, cliReport)
		if err != nil {
			fmt.Fprintf(stderr, "write audio front-end eval report: %v\n", err)
			return 1
		}
		cliReport.ReportPath = reportPath
	}
	if err := writeJSONAudioFrontEndEval(stdout, cliReport); err != nil {
		fmt.Fprintf(stderr, "encode audio front-end eval: %v\n", err)
		return 1
	}
	return 0
}

type audioFrontEndEvalCLIReport struct {
	audio.FrontEndEvalReport
	Metadata latencyBenchMetadata `json:"metadata"`
}

func buildAudioFrontEndEvalCLIReport(report audio.FrontEndEvalReport) audioFrontEndEvalCLIReport {
	return audioFrontEndEvalCLIReport{
		FrontEndEvalReport: report,
		Metadata:           buildLatencyBenchMetadata(),
	}
}

type firmwareDeviceReport struct {
	SchemaVersion        string                               `json:"schema_version"`
	GatewaySchemaVersion string                               `json:"gateway_schema_version"`
	GatewayService       string                               `json:"gateway_service"`
	CapturedAtMS         int64                                `json:"captured_at_ms"`
	GatewayURL           string                               `json:"gateway_url"`
	DeviceReportPath     string                               `json:"device_report_path,omitempty"`
	Devices              []firmwarecheck.DeviceIdentityRecord `json:"devices"`
}

func runFirmwareDeviceReport(args []string, stdout io.Writer, stderr io.Writer) int {
	gatewayURL := "http://127.0.0.1:21080"
	outputDir := "reports"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-device-report --gateway-url http://127.0.0.1:21080 --output-dir reports")
			return 0
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			gatewayURL = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown firmware-device-report option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(outputDir); err != nil {
		fmt.Fprintf(stderr, "firmware device report dir invalid: %v\n", err)
		return 1
	}
	report, err := fetchFirmwareDeviceReport(gatewayURL)
	if err != nil {
		fmt.Fprintf(stderr, "firmware device report failed: %v\n", err)
		return 1
	}
	reportPath, err := writeFirmwareDeviceReport(outputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write firmware device report: %v\n", err)
		return 1
	}
	report.DeviceReportPath = reportPath
	if err := writeJSONFirmwareDeviceReport(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode firmware device report: %v\n", err)
		return 1
	}
	return 0
}

type officePreflightOptions struct {
	ManifestPath   string
	ArtifactDir    string
	GatewayURL     string
	DeviceID       string
	Commit         string
	MaxDeviceAgeMS int64
	OutputDir      string
}

type officePreflightFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type officePreflightReport struct {
	SchemaVersion            string                              `json:"schema_version"`
	GeneratedAtMS            int64                               `json:"generated_at_ms"`
	Metadata                 latencyBenchMetadata                `json:"metadata"`
	DryRun                   bool                                `json:"dry_run"`
	FlashAllowed             bool                                `json:"flash_allowed"`
	ReadyForFlashPlan        bool                                `json:"ready_for_flash_plan"`
	NextRequiredConfirmation string                              `json:"next_required_confirmation"`
	GatewayURL               string                              `json:"gateway_url"`
	DeviceID                 string                              `json:"device_id"`
	Commit                   string                              `json:"commit"`
	MaxDeviceAgeMS           int64                               `json:"max_device_age_ms"`
	Artifact                 *firmwarecheck.ArtifactResult       `json:"artifact,omitempty"`
	Gateway                  *firmwareDeviceReport               `json:"gateway,omitempty"`
	DeviceIdentity           *firmwarecheck.DeviceIdentityResult `json:"device_identity,omitempty"`
	SerialDevices            []firmwarecheck.SerialDevice        `json:"serial_devices"`
	USBSerialCandidates      []firmwarecheck.SerialDevice        `json:"usb_serial_candidates"`
	DeviceReportPath         string                              `json:"device_report_path,omitempty"`
	ReportPath               string                              `json:"report_path,omitempty"`
	Findings                 []officePreflightFinding            `json:"findings,omitempty"`
}

type officeHandoffOptions struct {
	ManifestPath string
	ArtifactDir  string
	Commit       string
	KeepRecent   int
	OutputDir    string
}

type officeHandoffReport struct {
	SchemaVersion              string                           `json:"schema_version"`
	GeneratedAtMS              int64                            `json:"generated_at_ms"`
	Metadata                   latencyBenchMetadata             `json:"metadata"`
	DryRun                     bool                             `json:"dry_run"`
	FlashAllowed               bool                             `json:"flash_allowed"`
	DeleteAllowed              bool                             `json:"delete_allowed"`
	PhysicalAcceptanceRequired bool                             `json:"physical_acceptance_required"`
	OfficePreflightRequired    bool                             `json:"office_preflight_required"`
	Commit                     string                           `json:"commit"`
	CurrentArtifactPath        string                           `json:"current_artifact_path"`
	Artifact                   firmwarecheck.ArtifactResult     `json:"artifact"`
	ArtifactRetention          firmwareArtifactPrunePlanSummary `json:"artifact_retention"`
	SerialDevices              []firmwarecheck.SerialDevice     `json:"serial_devices"`
	USBSerialCandidateCount    int                              `json:"usb_serial_candidate_count"`
	NextRequiredActions        []string                         `json:"next_required_actions"`
	ReportPath                 string                           `json:"report_path,omitempty"`
	Findings                   []officePreflightFinding         `json:"findings,omitempty"`
}

type officeAcceptanceOptions struct {
	HandoffPath       string
	OfficePreflight   string
	FirmwareFlashPlan string
	OutputDir         string
}

type officeAcceptanceReport struct {
	SchemaVersion             string                   `json:"schema_version"`
	GeneratedAtMS             int64                    `json:"generated_at_ms"`
	Metadata                  latencyBenchMetadata     `json:"metadata"`
	DryRun                    bool                     `json:"dry_run"`
	FlashAllowed              bool                     `json:"flash_allowed"`
	DeleteAllowed             bool                     `json:"delete_allowed"`
	PhysicalAcceptanceStatus  string                   `json:"physical_acceptance_status"`
	HandoffReportPath         string                   `json:"handoff_report_path"`
	OfficePreflightReportPath string                   `json:"office_preflight_report_path"`
	FirmwareFlashPlanReport   string                   `json:"firmware_flash_plan_report_path,omitempty"`
	Commit                    string                   `json:"commit,omitempty"`
	ArtifactPath              string                   `json:"artifact_path,omitempty"`
	ArtifactSHA256            string                   `json:"artifact_sha256,omitempty"`
	DeviceID                  string                   `json:"device_id,omitempty"`
	NextRequiredActions       []string                 `json:"next_required_actions"`
	ReportPath                string                   `json:"report_path,omitempty"`
	Findings                  []officePreflightFinding `json:"findings,omitempty"`
}

type stackChanIdentityAcceptanceOptions struct {
	OfficeAcceptancePath string
	ManifestPath         string
	ArtifactDir          string
	GatewayURL           string
	DeviceID             string
	Commit               string
	MaxDeviceAgeMS       int64
	OutputDir            string
}

type stackChanIdentityAcceptanceReport struct {
	SchemaVersion              string                              `json:"schema_version"`
	GeneratedAtMS              int64                               `json:"generated_at_ms"`
	Metadata                   latencyBenchMetadata                `json:"metadata"`
	DryRun                     bool                                `json:"dry_run"`
	FlashAllowed               bool                                `json:"flash_allowed"`
	DeleteAllowed              bool                                `json:"delete_allowed"`
	HardwareAcceptanceScope    string                              `json:"hardware_acceptance_scope"`
	IdentityAcceptanceStatus   string                              `json:"identity_acceptance_status"`
	OfficeAcceptanceReportPath string                              `json:"office_acceptance_report_path"`
	GatewayURL                 string                              `json:"gateway_url"`
	DeviceID                   string                              `json:"device_id"`
	Commit                     string                              `json:"commit"`
	MaxDeviceAgeMS             int64                               `json:"max_device_age_ms"`
	ArtifactPath               string                              `json:"artifact_path,omitempty"`
	ArtifactSHA256             string                              `json:"artifact_sha256,omitempty"`
	Artifact                   *firmwarecheck.ArtifactResult       `json:"artifact,omitempty"`
	Gateway                    *firmwareDeviceReport               `json:"gateway,omitempty"`
	DeviceIdentity             *firmwarecheck.DeviceIdentityResult `json:"device_identity,omitempty"`
	SerialDevices              []firmwarecheck.SerialDevice        `json:"serial_devices"`
	USBSerialCandidates        []firmwarecheck.SerialDevice        `json:"usb_serial_candidates"`
	DeviceReportPath           string                              `json:"device_report_path,omitempty"`
	NextRequiredActions        []string                            `json:"next_required_actions"`
	ReportPath                 string                              `json:"report_path,omitempty"`
	Findings                   []officePreflightFinding            `json:"findings,omitempty"`
}

type stackChanCapabilityAcceptanceOptions struct {
	IdentityAcceptancePath string
	EvidencePath           string
	DeviceID               string
	Commit                 string
	OutputDir              string
}

type stackChanPhysicalEvidenceOptions struct {
	IdentityAcceptancePath string
	GatewayURL             string
	DeriveGateway          bool
	DeviceID               string
	Commit                 string
	OutputDir              string
	Passed                 map[string]string
}

type stackChanPhysicalEvidenceReport struct {
	SchemaVersion                string                                 `json:"schema_version"`
	GeneratedAtMS                int64                                  `json:"generated_at_ms,omitempty"`
	DryRun                       bool                                   `json:"dry_run,omitempty"`
	FlashAllowed                 bool                                   `json:"flash_allowed"`
	DeleteAllowed                bool                                   `json:"delete_allowed"`
	IdentityAcceptanceReportPath string                                 `json:"identity_acceptance_report_path,omitempty"`
	GatewayURL                   string                                 `json:"gateway_url,omitempty"`
	GatewayTraceID               string                                 `json:"gateway_trace_id,omitempty"`
	DeviceID                     string                                 `json:"device_id"`
	Commit                       string                                 `json:"commit"`
	ArtifactSHA256               string                                 `json:"artifact_sha256,omitempty"`
	RequiredCapabilities         []string                               `json:"required_capabilities,omitempty"`
	Observations                 []stackChanPhysicalEvidenceObservation `json:"observations"`
	ReportPath                   string                                 `json:"report_path,omitempty"`
	Findings                     []officePreflightFinding               `json:"findings,omitempty"`
}

type stackChanPhysicalEvidenceObservation struct {
	Capability   string `json:"capability"`
	Status       string `json:"status"`
	EvidenceType string `json:"evidence_type"`
	ObservedAtMS int64  `json:"observed_at_ms,omitempty"`
}

type stackChanCapabilityResult struct {
	Capability     string `json:"capability"`
	DeclaredStatus string `json:"declared_status,omitempty"`
	EvidenceStatus string `json:"evidence_status,omitempty"`
	EvidenceType   string `json:"evidence_type,omitempty"`
	ObservedAtMS   int64  `json:"observed_at_ms,omitempty"`
	Accepted       bool   `json:"accepted"`
}

type stackChanCapabilityAcceptanceReport struct {
	SchemaVersion                string                      `json:"schema_version"`
	GeneratedAtMS                int64                       `json:"generated_at_ms"`
	Metadata                     latencyBenchMetadata        `json:"metadata"`
	DryRun                       bool                        `json:"dry_run"`
	FlashAllowed                 bool                        `json:"flash_allowed"`
	DeleteAllowed                bool                        `json:"delete_allowed"`
	HardwareAcceptanceScope      string                      `json:"hardware_acceptance_scope"`
	CapabilityAcceptanceStatus   string                      `json:"capability_acceptance_status"`
	IdentityAcceptanceReportPath string                      `json:"identity_acceptance_report_path"`
	EvidenceReportPath           string                      `json:"evidence_report_path"`
	DeviceID                     string                      `json:"device_id"`
	Commit                       string                      `json:"commit"`
	ArtifactPath                 string                      `json:"artifact_path,omitempty"`
	ArtifactSHA256               string                      `json:"artifact_sha256,omitempty"`
	RequiredCapabilities         []string                    `json:"required_capabilities"`
	CapabilityResults            []stackChanCapabilityResult `json:"capability_results"`
	NextRequiredActions          []string                    `json:"next_required_actions"`
	ReportPath                   string                      `json:"report_path,omitempty"`
	Findings                     []officePreflightFinding    `json:"findings,omitempty"`
}

func runOfficeHandoff(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultOfficeHandoffOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 office-handoff --commit <git-sha> [--manifest firmware/stackchan/a21-firmware.json] [--artifact-dir firmware/artifacts] [--keep-recent 5] [--output-dir reports]")
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
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown office-handoff option %q\n", args[i])
			return 2
		}
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	if err := validateA21InputPath(options.ManifestPath); err != nil {
		fmt.Fprintf(stderr, "office handoff manifest path invalid: %v\n", err)
		return 1
	}
	if err := validateA21InputPath(options.ArtifactDir); err != nil {
		fmt.Fprintf(stderr, "office handoff artifact dir invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "office handoff report dir invalid: %v\n", err)
		return 1
	}
	report, err := buildOfficeHandoffReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "office handoff failed: %v\n", err)
		return 1
	}
	reportPath, err := writeOfficeHandoffReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write office handoff report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONOfficeHandoff(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode office handoff report: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "office handoff manifest ok (no flash, no delete)")
	return 0
}

func defaultOfficeHandoffOptions() officeHandoffOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	return officeHandoffOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		ArtifactDir:  "firmware/artifacts",
		Commit:       currentGitCommit(projectRoot),
		KeepRecent:   5,
		OutputDir:    "reports",
	}
}

func buildOfficeHandoffReport(options officeHandoffOptions) (officeHandoffReport, error) {
	artifact, err := firmwarecheck.ValidateLatestArtifactForCommit(firmwarecheck.LatestArtifactOptions{
		ManifestPath: options.ManifestPath,
		ArtifactDir:  options.ArtifactDir,
		Commit:       options.Commit,
	})
	if err != nil {
		return officeHandoffReport{}, err
	}
	prunePlan, err := firmwarecheck.BuildArtifactPrunePlan(firmwarecheck.ArtifactPrunePlanOptions{
		ManifestPath: options.ManifestPath,
		ArtifactDir:  options.ArtifactDir,
		Commit:       options.Commit,
		KeepRecent:   options.KeepRecent,
	})
	if err != nil {
		return officeHandoffReport{}, err
	}
	serialDevices, serialErr := listFirmwareSerialDevices()
	findings := []officePreflightFinding{}
	if serialErr != nil {
		findings = append(findings, officePreflightFinding{
			Code:    "serial_inventory_failed",
			Message: serialErr.Error(),
		})
	}
	usbCandidates := officeUSBSerialCandidates(serialDevices)
	return officeHandoffReport{
		SchemaVersion:              "a21.office_handoff.v1",
		GeneratedAtMS:              time.Now().UnixMilli(),
		Metadata:                   buildLatencyBenchMetadata(),
		DryRun:                     true,
		FlashAllowed:               false,
		DeleteAllowed:              false,
		PhysicalAcceptanceRequired: true,
		OfficePreflightRequired:    true,
		Commit:                     options.Commit,
		CurrentArtifactPath:        artifact.ArtifactPath,
		Artifact:                   artifact,
		ArtifactRetention:          summarizeFirmwareArtifactPrunePlan(prunePlan),
		SerialDevices:              serialDevices,
		USBSerialCandidateCount:    len(usbCandidates),
		NextRequiredActions: []string{
			"Run make release-check on the travel machine before leaving home.",
			"At the office, connect only the intended StackChan/CoreS3 and run A21_DEVICE_ID=stackchan-001 make office-preflight.",
			"Only after a fresh A21 Gateway device report and explicit USB serial path, run make firmware-flash-plan.",
			"Real flashing remains locked until a future explicit guarded A21 flash command exists.",
		},
		Findings: findings,
	}, nil
}

func runOfficeAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := officeAcceptanceOptions{OutputDir: "reports"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 office-acceptance --handoff reports/a21-office-handoff-...json --office-preflight reports/a21-office-preflight-...json [--firmware-flash-plan reports/a21-firmware-flash-plan-...json] [--output-dir reports]")
			return 0
		case "--handoff":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--handoff requires a value")
				return 2
			}
			i++
			options.HandoffPath = args[i]
		case "--office-preflight":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--office-preflight requires a value")
				return 2
			}
			i++
			options.OfficePreflight = args[i]
		case "--firmware-flash-plan":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--firmware-flash-plan requires a value")
				return 2
			}
			i++
			options.FirmwareFlashPlan = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown office-acceptance option %q\n", args[i])
			return 2
		}
	}
	if options.HandoffPath == "" {
		fmt.Fprintln(stderr, "--handoff requires a value")
		return 2
	}
	if options.OfficePreflight == "" {
		fmt.Fprintln(stderr, "--office-preflight requires a value")
		return 2
	}
	for _, path := range []string{options.HandoffPath, options.OfficePreflight, options.FirmwareFlashPlan} {
		if path == "" {
			continue
		}
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "office acceptance report path invalid: %v\n", err)
			return 1
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "office acceptance report dir invalid: %v\n", err)
		return 1
	}
	report, err := buildOfficeAcceptanceReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "office acceptance failed: %v\n", err)
		return 1
	}
	reportPath, err := writeOfficeAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write office acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONOfficeAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode office acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "office acceptance gate failed (no flash, no delete)")
		return 1
	}
	fmt.Fprintln(stdout, "office acceptance gate ok (no flash, no delete)")
	return 0
}

func buildOfficeAcceptanceReport(options officeAcceptanceOptions) (officeAcceptanceReport, error) {
	var handoff officeHandoffReport
	if err := readJSONFile(options.HandoffPath, &handoff); err != nil {
		return officeAcceptanceReport{}, err
	}
	var preflight officePreflightReport
	if err := readJSONFile(options.OfficePreflight, &preflight); err != nil {
		return officeAcceptanceReport{}, err
	}
	report := officeAcceptanceReport{
		SchemaVersion:             "a21.office_acceptance.v1",
		GeneratedAtMS:             time.Now().UnixMilli(),
		Metadata:                  buildLatencyBenchMetadata(),
		DryRun:                    true,
		FlashAllowed:              false,
		DeleteAllowed:             false,
		PhysicalAcceptanceStatus:  "ready_for_physical_acceptance",
		HandoffReportPath:         options.HandoffPath,
		OfficePreflightReportPath: options.OfficePreflight,
		Commit:                    firstNonEmpty(handoff.Commit, preflight.Commit),
		ArtifactPath:              firstNonEmpty(handoff.CurrentArtifactPath, preflightArtifactPath(preflight)),
		ArtifactSHA256:            firstNonEmpty(handoff.Artifact.SHA256, preflightArtifactSHA256(preflight)),
		DeviceID:                  preflight.DeviceID,
		NextRequiredActions: []string{
			"Keep this report with the handoff and office-preflight receipts.",
			"Run firmware-flash-plan only when an explicit USB serial port and fresh A21 Gateway device report are present.",
			"Real flashing remains locked until a future explicit guarded A21 flash command exists.",
		},
	}
	if handoff.SchemaVersion != "a21.office_handoff.v1" {
		report.addFinding("handoff_schema_invalid", "handoff report schema is not a21.office_handoff.v1")
	}
	if preflight.SchemaVersion != "a21.office_preflight.v1" {
		report.addFinding("office_preflight_schema_invalid", "office preflight report schema is not a21.office_preflight.v1")
	}
	if handoff.FlashAllowed || handoff.DeleteAllowed || preflight.FlashAllowed {
		report.addFinding("unsafe_permission", "handoff or office preflight unexpectedly allowed flash/delete")
	}
	if !preflight.ReadyForFlashPlan {
		report.addFinding("office_preflight_not_ready", "office preflight is not ready for flash-plan evidence")
	}
	if handoff.Commit != "" && preflight.Commit != "" && !sameCLICommit(handoff.Commit, preflight.Commit) {
		report.addFinding("commit_mismatch", "handoff and office preflight commits differ")
	}
	handoffArtifact := handoff.CurrentArtifactPath
	preflightArtifact := preflightArtifactPath(preflight)
	if handoffArtifact != "" && preflightArtifact != "" && !sameCleanCLIPath(handoffArtifact, preflightArtifact) {
		report.addFinding("artifact_mismatch", "handoff and office preflight artifacts differ")
	}
	handoffSHA := handoff.Artifact.SHA256
	preflightSHA := preflightArtifactSHA256(preflight)
	if handoffSHA != "" && preflightSHA != "" && !strings.EqualFold(handoffSHA, preflightSHA) {
		report.addFinding("artifact_sha_mismatch", "handoff and office preflight artifact sha256 values differ")
	}
	if options.FirmwareFlashPlan != "" {
		var flashPlan firmwarecheck.FlashPlanResult
		if err := readJSONFile(options.FirmwareFlashPlan, &flashPlan); err != nil {
			return officeAcceptanceReport{}, err
		}
		report.FirmwareFlashPlanReport = options.FirmwareFlashPlan
		if flashPlan.FlashAllowed {
			report.addFinding("unsafe_flash_plan", "firmware flash plan unexpectedly allowed flashing")
		}
		if flashPlan.Commit != "" && report.Commit != "" && !sameCLICommit(report.Commit, flashPlan.Commit) {
			report.addFinding("flash_plan_commit_mismatch", "firmware flash plan commit differs")
		}
		if flashPlan.ArtifactPath != "" && report.ArtifactPath != "" && !sameCleanCLIPath(report.ArtifactPath, flashPlan.ArtifactPath) {
			report.addFinding("flash_plan_artifact_mismatch", "firmware flash plan artifact differs")
		}
		if flashPlan.DeviceID != "" && report.DeviceID != "" && flashPlan.DeviceID != report.DeviceID {
			report.addFinding("flash_plan_device_mismatch", "firmware flash plan device differs")
		}
	}
	if len(report.Findings) > 0 {
		report.PhysicalAcceptanceStatus = "blocked"
	}
	return report, nil
}

func readJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (report *officeAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}

func runStackChanIdentityAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanIdentityAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-identity-acceptance --office-acceptance reports/a21-office-acceptance-...json --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> --max-device-age-ms 300000 [--output-dir reports]")
			return 0
		case "--office-acceptance":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--office-acceptance requires a value")
				return 2
			}
			i++
			options.OfficeAcceptancePath = args[i]
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
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			options.GatewayURL = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
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
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-identity-acceptance option %q\n", args[i])
			return 2
		}
	}
	if options.OfficeAcceptancePath == "" {
		fmt.Fprintln(stderr, "--office-acceptance requires a value")
		return 2
	}
	if options.DeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	if options.MaxDeviceAgeMS <= 0 {
		fmt.Fprintln(stderr, "--max-device-age-ms requires a value")
		return 2
	}
	for _, path := range []string{options.OfficeAcceptancePath, options.ManifestPath, options.ArtifactDir} {
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "stackchan identity acceptance path invalid: %v\n", err)
			return 1
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan identity acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanIdentityAcceptanceReport(options)
	reportPath, err := writeStackChanIdentityAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan identity acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanIdentityAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan identity acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan identity acceptance failed (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan identity acceptance ok (no flash performed)")
	return 0
}

func defaultStackChanIdentityAcceptanceOptions() stackChanIdentityAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	maxDeviceAgeMS, _ := strconv.ParseInt(strings.TrimSpace(os.Getenv("A21_DEVICE_MAX_AGE_MS")), 10, 64)
	return stackChanIdentityAcceptanceOptions{
		ManifestPath:   "firmware/stackchan/a21-firmware.json",
		ArtifactDir:    "firmware/artifacts",
		GatewayURL:     firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:       deviceID,
		Commit:         currentGitCommit(projectRoot),
		MaxDeviceAgeMS: maxDeviceAgeMS,
		OutputDir:      "reports",
	}
}

func buildStackChanIdentityAcceptanceReport(options stackChanIdentityAcceptanceOptions) stackChanIdentityAcceptanceReport {
	report := stackChanIdentityAcceptanceReport{
		SchemaVersion:              "a21.stackchan_identity_acceptance.v1",
		GeneratedAtMS:              time.Now().UnixMilli(),
		Metadata:                   buildLatencyBenchMetadata(),
		DryRun:                     true,
		FlashAllowed:               false,
		DeleteAllowed:              false,
		HardwareAcceptanceScope:    "identity_only",
		IdentityAcceptanceStatus:   "identity_confirmed",
		OfficeAcceptanceReportPath: options.OfficeAcceptancePath,
		GatewayURL:                 sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                   options.DeviceID,
		Commit:                     options.Commit,
		MaxDeviceAgeMS:             options.MaxDeviceAgeMS,
		NextRequiredActions: []string{
			"Keep this report with the office acceptance, device report, and firmware artifact receipts.",
			"Identity acceptance does not prove microphone, speaker, screen, servo, RGB, OTA, or latency acceptance.",
			"Real flashing remains locked until a future explicit guarded A21 flash command exists.",
		},
	}

	var officeAcceptance officeAcceptanceReport
	if err := readJSONFile(options.OfficeAcceptancePath, &officeAcceptance); err != nil {
		report.addFinding("office_acceptance_unreadable", err.Error())
	} else {
		validateOfficeAcceptanceForStackChanIdentity(&report, officeAcceptance)
	}

	artifact, err := firmwarecheck.ValidateLatestArtifactForCommit(firmwarecheck.LatestArtifactOptions{
		ManifestPath: options.ManifestPath,
		ArtifactDir:  options.ArtifactDir,
		Commit:       options.Commit,
	})
	if err != nil {
		report.addFinding("current_artifact_invalid", err.Error())
	} else {
		report.Artifact = &artifact
		report.ArtifactPath = artifact.ArtifactPath
		report.ArtifactSHA256 = artifact.SHA256
		if officeAcceptance.ArtifactPath != "" && !sameCleanCLIPath(officeAcceptance.ArtifactPath, artifact.ArtifactPath) {
			report.addFinding("office_acceptance_artifact_mismatch", "office acceptance artifact differs from current artifact")
		}
		if officeAcceptance.ArtifactSHA256 != "" && !strings.EqualFold(officeAcceptance.ArtifactSHA256, artifact.SHA256) {
			report.addFinding("office_acceptance_artifact_sha_mismatch", "office acceptance artifact sha256 differs from current artifact")
		}
	}

	gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_failed", err.Error())
	} else {
		deviceReportPath, writeErr := writeFirmwareDeviceReport(options.OutputDir, gatewayReport)
		if writeErr != nil {
			report.addFinding("device_report_write_failed", writeErr.Error())
		} else {
			gatewayReport.DeviceReportPath = deviceReportPath
			report.DeviceReportPath = deviceReportPath
			report.Gateway = &gatewayReport
			if report.Artifact != nil {
				deviceIdentity, identityErr := firmwarecheck.ValidateDeviceIdentity(firmwarecheck.DeviceIdentityOptions{
					ManifestPath:      options.ManifestPath,
					ArtifactPath:      report.Artifact.ArtifactPath,
					ReportPath:        deviceReportPath,
					ExpectedDeviceID:  options.DeviceID,
					ExpectedGitCommit: options.Commit,
					MaxDeviceAgeMS:    options.MaxDeviceAgeMS,
				})
				if identityErr != nil {
					report.addFinding("device_identity_not_confirmed", identityErr.Error())
				} else {
					report.DeviceIdentity = &deviceIdentity
					if quiescentErr := firmwarecheck.ValidateFlashPlanDeviceQuiescent(deviceIdentity.Device); quiescentErr != nil {
						report.addFinding("device_not_quiescent", quiescentErr.Error())
					}
				}
			}
		}
	}

	serialDevices, err := listFirmwareSerialDevices()
	if err != nil {
		report.addFinding("serial_inventory_failed", err.Error())
	} else {
		report.SerialDevices = serialDevices
		report.USBSerialCandidates = officeUSBSerialCandidates(serialDevices)
		if len(report.USBSerialCandidates) == 0 {
			report.addFinding("usb_serial_candidate_missing", "no available USB serial candidate found for physical identity acceptance")
		}
	}

	if len(report.Findings) > 0 {
		report.IdentityAcceptanceStatus = "blocked"
	}
	return report
}

func validateOfficeAcceptanceForStackChanIdentity(report *stackChanIdentityAcceptanceReport, officeAcceptance officeAcceptanceReport) {
	if officeAcceptance.SchemaVersion != "a21.office_acceptance.v1" {
		report.addFinding("office_acceptance_schema_invalid", "office acceptance report schema is not a21.office_acceptance.v1")
	}
	if officeAcceptance.FlashAllowed || officeAcceptance.DeleteAllowed {
		report.addFinding("unsafe_permission", "office acceptance unexpectedly allowed flash/delete")
	}
	if officeAcceptance.PhysicalAcceptanceStatus != "ready_for_physical_acceptance" || len(officeAcceptance.Findings) > 0 {
		report.addFinding("office_acceptance_not_ready", "office acceptance report is not ready for physical identity confirmation")
	}
	if officeAcceptance.Commit != "" && !sameCLICommit(officeAcceptance.Commit, report.Commit) {
		report.addFinding("office_acceptance_commit_mismatch", "office acceptance commit differs from expected commit")
	}
	if officeAcceptance.DeviceID != "" && officeAcceptance.DeviceID != report.DeviceID {
		report.addFinding("office_acceptance_device_mismatch", "office acceptance device differs from expected device")
	}
}

func (report *stackChanIdentityAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}

var requiredStackChanCapabilities = []string{
	"microphone",
	"speaker",
	"screen",
	"screen_touch",
	"top_touch",
	"servo_y",
	"rgb",
}

func runStackChanPhysicalEvidence(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanPhysicalEvidenceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-physical-evidence --identity-acceptance reports/a21-stackchan-identity-acceptance-...json --device-id stackchan-001 --commit <git-sha> [--derive-gateway --gateway-url http://127.0.0.1:21080] [--pass capability=evidence_type] [--output-dir reports]")
			return 0
		case "--identity-acceptance":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--identity-acceptance requires a value")
				return 2
			}
			i++
			options.IdentityAcceptancePath = args[i]
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			options.GatewayURL = args[i]
		case "--derive-gateway":
			options.DeriveGateway = true
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
		case "--pass":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--pass requires capability=evidence_type")
				return 2
			}
			i++
			capability, evidenceType, err := parseStackChanPhysicalEvidencePass(args[i])
			if err != nil {
				fmt.Fprintf(stderr, "stackchan physical evidence pass invalid: %v\n", err)
				return 1
			}
			options.Passed[capability] = evidenceType
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-physical-evidence option %q\n", args[i])
			return 2
		}
	}
	if options.IdentityAcceptancePath == "" {
		fmt.Fprintln(stderr, "--identity-acceptance requires a value")
		return 2
	}
	if options.DeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	if err := validateA21InputPath(options.IdentityAcceptancePath); err != nil {
		fmt.Fprintf(stderr, "stackchan physical evidence path invalid: %v\n", err)
		return 1
	}
	if options.DeriveGateway {
		if _, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/devices", nil); err != nil {
			fmt.Fprintf(stderr, "stackchan physical evidence gateway URL invalid: %v\n", err)
			return 1
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan physical evidence report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanPhysicalEvidenceReport(options)
	reportPath, err := writeStackChanPhysicalEvidenceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan physical evidence report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanPhysicalEvidence(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan physical evidence report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan physical evidence blocked (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan physical evidence template written (no flash performed)")
	return 0
}

func defaultStackChanPhysicalEvidenceOptions() stackChanPhysicalEvidenceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	return stackChanPhysicalEvidenceOptions{
		GatewayURL: firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:   deviceID,
		Commit:     currentGitCommit(projectRoot),
		OutputDir:  "reports",
		Passed:     map[string]string{},
	}
}

func parseStackChanPhysicalEvidencePass(value string) (string, string, error) {
	if containsLegacyIdentity(value) {
		return "", "", fmt.Errorf("contains forbidden legacy identity")
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("must use capability=evidence_type")
	}
	capability := strings.TrimSpace(parts[0])
	evidenceType := strings.TrimSpace(parts[1])
	if capability == "" || evidenceType == "" {
		return "", "", fmt.Errorf("capability and evidence_type are required")
	}
	if !isRequiredStackChanCapability(capability) {
		return "", "", fmt.Errorf("unknown StackChan capability")
	}
	return capability, evidenceType, nil
}

func buildStackChanPhysicalEvidenceReport(options stackChanPhysicalEvidenceOptions) stackChanPhysicalEvidenceReport {
	nowMS := time.Now().UnixMilli()
	report := stackChanPhysicalEvidenceReport{
		SchemaVersion:                "a21.stackchan_physical_evidence.v1",
		GeneratedAtMS:                nowMS,
		DryRun:                       true,
		FlashAllowed:                 false,
		DeleteAllowed:                false,
		IdentityAcceptanceReportPath: options.IdentityAcceptancePath,
		DeviceID:                     options.DeviceID,
		Commit:                       options.Commit,
		RequiredCapabilities:         append([]string(nil), requiredStackChanCapabilities...),
	}
	var identity stackChanIdentityAcceptanceReport
	if err := readJSONFile(options.IdentityAcceptancePath, &identity); err != nil {
		report.addFinding("identity_acceptance_unreadable", err.Error())
	} else {
		validateIdentityAcceptanceForPhysicalEvidence(&report, identity)
	}
	declared := map[string]string{}
	if identity.DeviceIdentity != nil {
		declared = identity.DeviceIdentity.Device.Capabilities
	}
	observed := map[string]stackChanPhysicalEvidenceObservation{}
	if options.DeriveGateway {
		deriveGatewayPhysicalEvidence(&report, options, observed)
	}
	for capability, evidenceType := range options.Passed {
		observed[capability] = stackChanPhysicalEvidenceObservation{
			Capability:   capability,
			Status:       "passed",
			EvidenceType: evidenceType,
			ObservedAtMS: nowMS,
		}
	}
	for _, capability := range requiredStackChanCapabilities {
		observation := stackChanPhysicalEvidenceObservation{
			Capability:   capability,
			Status:       "pending",
			EvidenceType: "operator_observation_required",
		}
		if derived, ok := observed[capability]; ok {
			observation = derived
		}
		if declared[capability] != "available" {
			report.addFinding("capability_not_declared_available", "required StackChan capability is not declared available by the identity acceptance report")
		}
		report.Observations = append(report.Observations, observation)
	}
	return report
}

func deriveGatewayPhysicalEvidence(report *stackChanPhysicalEvidenceReport, options stackChanPhysicalEvidenceOptions, observed map[string]stackChanPhysicalEvidenceObservation) {
	gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_failed", err.Error())
		return
	}
	report.GatewayURL = gatewayReport.GatewayURL
	device, ok := findFirmwareDeviceRecord(gatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_missing", "expected device is missing from Gateway report")
		return
	}
	if device.RuntimeEcho["screen"] != "" {
		observed["screen"] = stackChanPhysicalEvidenceObservation{
			Capability:   "screen",
			Status:       "passed",
			EvidenceType: "device_screen_echo",
			ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
		}
	}
	if device.RuntimeEcho["servo_y"] != "" {
		observed["servo_y"] = stackChanPhysicalEvidenceObservation{
			Capability:   "servo_y",
			Status:       "passed",
			EvidenceType: "device_servo_echo",
			ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
		}
	}
	if device.RuntimeEcho["rgb"] != "" {
		observed["rgb"] = stackChanPhysicalEvidenceObservation{
			Capability:   "rgb",
			Status:       "passed",
			EvidenceType: "device_rgb_echo",
			ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
		}
	}
	if device.CurrentExpression != "" || device.CurrentMode != "" {
		if _, ok := observed["screen"]; !ok {
			observed["screen"] = stackChanPhysicalEvidenceObservation{
				Capability:   "screen",
				Status:       "passed",
				EvidenceType: "gateway_render_state",
				ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
			}
		}
	}
	if strings.HasPrefix(device.LastEvent, "touch.") {
		switch device.LastTouchSource {
		case "screen":
			observed["screen_touch"] = stackChanPhysicalEvidenceObservation{
				Capability:   "screen_touch",
				Status:       "passed",
				EvidenceType: "gateway_touch_event",
				ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
			}
		case "top_sensor":
			observed["top_touch"] = stackChanPhysicalEvidenceObservation{
				Capability:   "top_touch",
				Status:       "passed",
				EvidenceType: "gateway_touch_event",
				ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
			}
		}
	}
	if device.LastTraceID == "" {
		return
	}
	trace, err := fetchGatewayTrace(options.GatewayURL, device.LastTraceID)
	if err != nil {
		report.addFinding("gateway_trace_fetch_failed", err.Error())
		return
	}
	report.GatewayTraceID = trace.TraceID
	for _, event := range trace.Events {
		switch event.Name {
		case "audio.frame.received":
			observed["microphone"] = stackChanPhysicalEvidenceObservation{
				Capability:   "microphone",
				Status:       "passed",
				EvidenceType: "gateway_audio_frame",
				ObservedAtMS: bestObservedAtMS(event.AtMS, report.GeneratedAtMS),
			}
		case "audio.playback.chunk.sent":
			observed["speaker"] = stackChanPhysicalEvidenceObservation{
				Capability:   "speaker",
				Status:       "passed",
				EvidenceType: "gateway_audio_downlink",
				ObservedAtMS: bestObservedAtMS(event.AtMS, report.GeneratedAtMS),
			}
		}
	}
}

func findFirmwareDeviceRecord(devices []firmwarecheck.DeviceIdentityRecord, deviceID string) (firmwarecheck.DeviceIdentityRecord, bool) {
	for _, device := range devices {
		if device.DeviceID == deviceID {
			return device, true
		}
	}
	return firmwarecheck.DeviceIdentityRecord{}, false
}

func bestObservedAtMS(candidate int64, fallback int64) int64 {
	if candidate > 0 {
		return candidate
	}
	return fallback
}

func validateIdentityAcceptanceForPhysicalEvidence(report *stackChanPhysicalEvidenceReport, identity stackChanIdentityAcceptanceReport) {
	if identity.SchemaVersion != "a21.stackchan_identity_acceptance.v1" {
		report.addFinding("identity_acceptance_schema_invalid", "identity acceptance report schema is not a21.stackchan_identity_acceptance.v1")
	}
	if identity.FlashAllowed || identity.DeleteAllowed {
		report.addFinding("unsafe_permission", "identity acceptance unexpectedly allowed flash/delete")
	}
	if identity.IdentityAcceptanceStatus != "identity_confirmed" || len(identity.Findings) > 0 {
		report.addFinding("identity_acceptance_not_confirmed", "identity acceptance report is not confirmed")
	}
	if identity.DeviceID != "" && identity.DeviceID != report.DeviceID {
		report.addFinding("identity_acceptance_device_mismatch", "identity acceptance device differs from expected device")
	}
	if identity.Commit != "" && !sameCLICommit(identity.Commit, report.Commit) {
		report.addFinding("identity_acceptance_commit_mismatch", "identity acceptance commit differs from expected commit")
	}
	if identity.ArtifactSHA256 != "" {
		report.ArtifactSHA256 = identity.ArtifactSHA256
	}
	if identity.DeviceIdentity == nil || !identity.DeviceIdentity.DeviceIdentityConfirmed {
		report.addFinding("device_identity_missing", "identity acceptance report is missing confirmed device identity")
		return
	}
	device := identity.DeviceIdentity.Device
	for _, value := range []string{device.DeviceID, device.Firmware.ID, device.Firmware.Version, device.Firmware.Board, device.Firmware.Commit} {
		if containsLegacyIdentity(value) {
			report.addFinding("identity_acceptance_legacy_identity", "identity acceptance contains forbidden legacy identity")
			return
		}
	}
	for key, value := range device.Capabilities {
		if containsLegacyIdentity(key) || containsLegacyIdentity(value) {
			report.addFinding("identity_acceptance_legacy_identity", "identity acceptance contains forbidden legacy identity")
			return
		}
	}
}

func isRequiredStackChanCapability(capability string) bool {
	for _, required := range requiredStackChanCapabilities {
		if capability == required {
			return true
		}
	}
	return false
}

func (report *stackChanPhysicalEvidenceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}

func runStackChanCapabilityAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanCapabilityAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-capability-acceptance --identity-acceptance reports/a21-stackchan-identity-acceptance-...json --evidence reports/a21-stackchan-physical-evidence.json --device-id stackchan-001 --commit <git-sha> [--output-dir reports]")
			return 0
		case "--identity-acceptance":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--identity-acceptance requires a value")
				return 2
			}
			i++
			options.IdentityAcceptancePath = args[i]
		case "--evidence":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--evidence requires a value")
				return 2
			}
			i++
			options.EvidencePath = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-capability-acceptance option %q\n", args[i])
			return 2
		}
	}
	if options.IdentityAcceptancePath == "" {
		fmt.Fprintln(stderr, "--identity-acceptance requires a value")
		return 2
	}
	if options.EvidencePath == "" {
		fmt.Fprintln(stderr, "--evidence requires a value")
		return 2
	}
	if options.DeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	for _, path := range []string{options.IdentityAcceptancePath, options.EvidencePath} {
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "stackchan capability acceptance path invalid: %v\n", err)
			return 1
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan capability acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanCapabilityAcceptanceReport(options)
	reportPath, err := writeStackChanCapabilityAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan capability acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanCapabilityAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan capability acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan capability acceptance failed (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan capability acceptance ok (no flash performed)")
	return 0
}

func defaultStackChanCapabilityAcceptanceOptions() stackChanCapabilityAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	return stackChanCapabilityAcceptanceOptions{
		DeviceID:  deviceID,
		Commit:    currentGitCommit(projectRoot),
		OutputDir: "reports",
	}
}

func buildStackChanCapabilityAcceptanceReport(options stackChanCapabilityAcceptanceOptions) stackChanCapabilityAcceptanceReport {
	report := stackChanCapabilityAcceptanceReport{
		SchemaVersion:                "a21.stackchan_capability_acceptance.v1",
		GeneratedAtMS:                time.Now().UnixMilli(),
		Metadata:                     buildLatencyBenchMetadata(),
		DryRun:                       true,
		FlashAllowed:                 false,
		DeleteAllowed:                false,
		HardwareAcceptanceScope:      "physical_capability_evidence",
		CapabilityAcceptanceStatus:   "confirmed",
		IdentityAcceptanceReportPath: options.IdentityAcceptancePath,
		EvidenceReportPath:           options.EvidencePath,
		DeviceID:                     options.DeviceID,
		Commit:                       options.Commit,
		RequiredCapabilities:         append([]string(nil), requiredStackChanCapabilities...),
		NextRequiredActions: []string{
			"Keep this report with identity acceptance, physical evidence, firmware artifact, and office acceptance receipts.",
			"Capability acceptance is evidence-gated and still does not enable flashing.",
			"Run latency and full-duplex physical acceptance separately before claiming the complete A21 voice experience.",
		},
	}

	var identity stackChanIdentityAcceptanceReport
	if err := readJSONFile(options.IdentityAcceptancePath, &identity); err != nil {
		report.addFinding("identity_acceptance_unreadable", err.Error())
	} else {
		validateIdentityAcceptanceForStackChanCapability(&report, identity)
	}

	var evidence stackChanPhysicalEvidenceReport
	if err := readJSONFile(options.EvidencePath, &evidence); err != nil {
		report.addFinding("physical_evidence_unreadable", err.Error())
	} else {
		validatePhysicalEvidenceForStackChanCapability(&report, identity, evidence)
	}

	if len(report.Findings) > 0 {
		report.CapabilityAcceptanceStatus = "blocked"
	}
	return report
}

func validateIdentityAcceptanceForStackChanCapability(report *stackChanCapabilityAcceptanceReport, identity stackChanIdentityAcceptanceReport) {
	if identity.SchemaVersion != "a21.stackchan_identity_acceptance.v1" {
		report.addFinding("identity_acceptance_schema_invalid", "identity acceptance report schema is not a21.stackchan_identity_acceptance.v1")
	}
	if identity.FlashAllowed || identity.DeleteAllowed {
		report.addFinding("unsafe_permission", "identity acceptance unexpectedly allowed flash/delete")
	}
	if identity.IdentityAcceptanceStatus != "identity_confirmed" || len(identity.Findings) > 0 {
		report.addFinding("identity_acceptance_not_confirmed", "identity acceptance report is not confirmed")
	}
	if identity.DeviceID != "" && identity.DeviceID != report.DeviceID {
		report.addFinding("identity_acceptance_device_mismatch", "identity acceptance device differs from expected device")
	}
	if identity.Commit != "" && !sameCLICommit(identity.Commit, report.Commit) {
		report.addFinding("identity_acceptance_commit_mismatch", "identity acceptance commit differs from expected commit")
	}
	if identity.ArtifactPath != "" {
		report.ArtifactPath = identity.ArtifactPath
	}
	if identity.ArtifactSHA256 != "" {
		report.ArtifactSHA256 = identity.ArtifactSHA256
	}
	if identity.DeviceIdentity == nil || !identity.DeviceIdentity.DeviceIdentityConfirmed {
		report.addFinding("device_identity_missing", "identity acceptance report is missing confirmed device identity")
		return
	}
	device := identity.DeviceIdentity.Device
	for _, value := range []string{device.DeviceID, device.Firmware.ID, device.Firmware.Version, device.Firmware.Board, device.Firmware.Commit} {
		if containsLegacyIdentity(value) {
			report.addFinding("identity_acceptance_legacy_identity", "identity acceptance contains forbidden legacy identity")
			return
		}
	}
	for key, value := range device.Capabilities {
		if containsLegacyIdentity(key) || containsLegacyIdentity(value) {
			report.addFinding("identity_acceptance_legacy_identity", "identity acceptance contains forbidden legacy identity")
			return
		}
	}
}

func validatePhysicalEvidenceForStackChanCapability(report *stackChanCapabilityAcceptanceReport, identity stackChanIdentityAcceptanceReport, evidence stackChanPhysicalEvidenceReport) {
	if evidence.SchemaVersion != "a21.stackchan_physical_evidence.v1" {
		report.addFinding("physical_evidence_schema_invalid", "physical evidence report schema is not a21.stackchan_physical_evidence.v1")
	}
	if evidence.FlashAllowed || evidence.DeleteAllowed {
		report.addFinding("unsafe_permission", "physical evidence unexpectedly allowed flash/delete")
	}
	if evidence.DeviceID != "" && evidence.DeviceID != report.DeviceID {
		report.addFinding("physical_evidence_device_mismatch", "physical evidence device differs from expected device")
	}
	if evidence.Commit != "" && !sameCLICommit(evidence.Commit, report.Commit) {
		report.addFinding("physical_evidence_commit_mismatch", "physical evidence commit differs from expected commit")
	}
	if evidence.ArtifactSHA256 != "" && report.ArtifactSHA256 != "" && !strings.EqualFold(evidence.ArtifactSHA256, report.ArtifactSHA256) {
		report.addFinding("physical_evidence_artifact_sha_mismatch", "physical evidence artifact sha256 differs from identity acceptance")
	}

	observations := map[string]stackChanPhysicalEvidenceObservation{}
	for _, observation := range evidence.Observations {
		if containsLegacyIdentity(observation.Capability) || containsLegacyIdentity(observation.Status) || containsLegacyIdentity(observation.EvidenceType) {
			report.addFinding("physical_evidence_legacy_identity", "physical evidence contains forbidden legacy identity")
			continue
		}
		capability := strings.TrimSpace(observation.Capability)
		if capability == "" {
			report.addFinding("capability_evidence_invalid", "physical evidence contains an observation without a capability")
			continue
		}
		if _, exists := observations[capability]; !exists || observation.Status == "passed" {
			observations[capability] = observation
		}
	}

	declared := map[string]string{}
	if identity.DeviceIdentity != nil {
		declared = identity.DeviceIdentity.Device.Capabilities
	}
	for _, capability := range requiredStackChanCapabilities {
		result := stackChanCapabilityResult{
			Capability:     capability,
			DeclaredStatus: declared[capability],
		}
		if result.DeclaredStatus != "available" {
			report.addFinding("capability_not_declared_available", "required StackChan capability is not declared available by the fresh device identity")
		}
		observation, ok := observations[capability]
		if !ok {
			report.addFinding("capability_evidence_missing", "physical evidence missing for required StackChan capability")
			report.CapabilityResults = append(report.CapabilityResults, result)
			continue
		}
		result.EvidenceStatus = observation.Status
		result.EvidenceType = observation.EvidenceType
		result.ObservedAtMS = observation.ObservedAtMS
		if observation.Status != "passed" {
			report.addFinding("capability_evidence_not_passed", "physical evidence for required StackChan capability is not passed")
		}
		if strings.TrimSpace(observation.EvidenceType) == "" {
			report.addFinding("capability_evidence_type_missing", "physical evidence for required StackChan capability is missing evidence_type")
		}
		if observation.ObservedAtMS <= 0 {
			report.addFinding("capability_evidence_time_missing", "physical evidence for required StackChan capability is missing observed_at_ms")
		}
		result.Accepted = result.DeclaredStatus == "available" &&
			result.EvidenceStatus == "passed" &&
			strings.TrimSpace(result.EvidenceType) != "" &&
			result.ObservedAtMS > 0
		report.CapabilityResults = append(report.CapabilityResults, result)
	}
}

func (report *stackChanCapabilityAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}

func preflightArtifactPath(report officePreflightReport) string {
	if report.Artifact == nil {
		return ""
	}
	return report.Artifact.ArtifactPath
}

func preflightArtifactSHA256(report officePreflightReport) string {
	if report.Artifact == nil {
		return ""
	}
	return report.Artifact.SHA256
}

func sameCLICommit(left string, right string) bool {
	left = strings.ToLower(left)
	right = strings.ToLower(right)
	return left == right || strings.HasPrefix(left, right) || strings.HasPrefix(right, left)
}

func sameCleanCLIPath(left string, right string) bool {
	leftClean, leftErr := filepath.Abs(filepath.Clean(left))
	rightClean, rightErr := filepath.Abs(filepath.Clean(right))
	if leftErr != nil || rightErr != nil {
		return filepath.Clean(left) == filepath.Clean(right)
	}
	return leftClean == rightClean
}

func runOfficePreflight(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultOfficePreflightOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 office-preflight --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> --max-device-age-ms 300000 [--manifest firmware/stackchan/a21-firmware.json] [--artifact-dir firmware/artifacts] [--output-dir reports]")
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
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			options.GatewayURL = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
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
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown office-preflight option %q\n", args[i])
			return 2
		}
	}
	if options.DeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	if options.MaxDeviceAgeMS <= 0 {
		fmt.Fprintln(stderr, "--max-device-age-ms requires a value")
		return 2
	}
	if err := validateA21InputPath(options.ManifestPath); err != nil {
		fmt.Fprintf(stderr, "office preflight manifest path invalid: %v\n", err)
		return 1
	}
	if err := validateA21InputPath(options.ArtifactDir); err != nil {
		fmt.Fprintf(stderr, "office preflight artifact dir invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "office preflight report dir invalid: %v\n", err)
		return 1
	}
	report := buildOfficePreflightReport(options)
	reportPath, err := writeOfficePreflightReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write office preflight report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONOfficePreflight(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode office preflight report: %v\n", err)
		return 1
	}
	if !report.ReadyForFlashPlan {
		fmt.Fprintln(stdout, "office preflight failed (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "office preflight ok (no flash performed)")
	return 0
}

func defaultOfficePreflightOptions() officePreflightOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	maxDeviceAgeMS, _ := strconv.ParseInt(strings.TrimSpace(os.Getenv("A21_DEVICE_MAX_AGE_MS")), 10, 64)
	return officePreflightOptions{
		ManifestPath:   "firmware/stackchan/a21-firmware.json",
		ArtifactDir:    "firmware/artifacts",
		GatewayURL:     firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:       deviceID,
		Commit:         currentGitCommit(projectRoot),
		MaxDeviceAgeMS: maxDeviceAgeMS,
		OutputDir:      "reports",
	}
}

func buildOfficePreflightReport(options officePreflightOptions) officePreflightReport {
	report := officePreflightReport{
		SchemaVersion:            "a21.office_preflight.v1",
		GeneratedAtMS:            time.Now().UnixMilli(),
		Metadata:                 buildLatencyBenchMetadata(),
		DryRun:                   true,
		FlashAllowed:             false,
		NextRequiredConfirmation: "firmware-flash-plan_with_explicit_usb_port",
		GatewayURL:               sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                 options.DeviceID,
		Commit:                   options.Commit,
		MaxDeviceAgeMS:           options.MaxDeviceAgeMS,
	}

	artifact, err := firmwarecheck.ValidateLatestArtifactForCommit(firmwarecheck.LatestArtifactOptions{
		ManifestPath: options.ManifestPath,
		ArtifactDir:  options.ArtifactDir,
		Commit:       options.Commit,
	})
	if err != nil {
		report.addFinding("current_artifact_invalid", err.Error())
	} else {
		report.Artifact = &artifact
	}

	gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_failed", err.Error())
	} else {
		deviceReportPath, writeErr := writeFirmwareDeviceReport(options.OutputDir, gatewayReport)
		if writeErr != nil {
			report.addFinding("device_report_write_failed", writeErr.Error())
		} else {
			gatewayReport.DeviceReportPath = deviceReportPath
			report.DeviceReportPath = deviceReportPath
			report.Gateway = &gatewayReport
			if report.Artifact != nil {
				deviceIdentity, identityErr := firmwarecheck.ValidateDeviceIdentity(firmwarecheck.DeviceIdentityOptions{
					ManifestPath:      options.ManifestPath,
					ArtifactPath:      report.Artifact.ArtifactPath,
					ReportPath:        deviceReportPath,
					ExpectedDeviceID:  options.DeviceID,
					ExpectedGitCommit: options.Commit,
					MaxDeviceAgeMS:    options.MaxDeviceAgeMS,
				})
				if identityErr != nil {
					report.addFinding("device_identity_not_ready", identityErr.Error())
				} else {
					report.DeviceIdentity = &deviceIdentity
					if quiescentErr := firmwarecheck.ValidateFlashPlanDeviceQuiescent(deviceIdentity.Device); quiescentErr != nil {
						report.addFinding("device_not_quiescent", quiescentErr.Error())
					}
				}
			}
		}
	}

	serialDevices, err := listFirmwareSerialDevices()
	if err != nil {
		report.addFinding("serial_inventory_failed", err.Error())
	} else {
		report.SerialDevices = serialDevices
		report.USBSerialCandidates = officeUSBSerialCandidates(serialDevices)
		if len(report.USBSerialCandidates) == 0 {
			report.addFinding("usb_serial_candidate_missing", "no available USB serial candidate found for a future explicit firmware flash plan")
		}
	}

	report.ReadyForFlashPlan = report.Artifact != nil &&
		report.Gateway != nil &&
		report.DeviceIdentity != nil &&
		len(report.USBSerialCandidates) > 0 &&
		len(report.Findings) == 0
	return report
}

func (report *officePreflightReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{
		Code:    code,
		Message: message,
	})
}

func officeUSBSerialCandidates(devices []firmwarecheck.SerialDevice) []firmwarecheck.SerialDevice {
	candidates := make([]firmwarecheck.SerialDevice, 0, len(devices))
	for _, device := range devices {
		if device.USBModem && device.Usage.Exists && !device.Usage.InUse {
			candidates = append(candidates, device)
		}
	}
	return candidates
}

func sanitizedOfficeGatewayURL(raw string) string {
	_, safe, err := firmwareDeviceReportEndpoint(raw)
	if err != nil {
		return ""
	}
	return safe
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func fetchFirmwareDeviceReport(gatewayBaseURL string) (firmwareDeviceReport, error) {
	endpoint, safeGatewayURL, err := firmwareDeviceReportEndpoint(gatewayBaseURL)
	if err != nil {
		return firmwareDeviceReport{}, err
	}
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return firmwareDeviceReport{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return firmwareDeviceReport{}, fmt.Errorf("gateway device report returned status %d", resp.StatusCode)
	}
	var payload struct {
		SchemaVersion string                               `json:"schema_version"`
		Service       string                               `json:"service"`
		Devices       []firmwarecheck.DeviceIdentityRecord `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return firmwareDeviceReport{}, err
	}
	if err := validateFirmwareDeviceReportGatewayIdentity(payload.SchemaVersion, payload.Service); err != nil {
		return firmwareDeviceReport{}, err
	}
	if err := validateFirmwareDeviceReportDevices(payload.Devices); err != nil {
		return firmwareDeviceReport{}, err
	}
	return firmwareDeviceReport{
		SchemaVersion:        "a21.firmware.device_report.v1",
		GatewaySchemaVersion: payload.SchemaVersion,
		GatewayService:       payload.Service,
		CapturedAtMS:         time.Now().UnixMilli(),
		GatewayURL:           safeGatewayURL,
		Devices:              payload.Devices,
	}, nil
}

func fetchGatewayTrace(gatewayBaseURL string, traceID string) (gateway.TraceResponse, error) {
	if strings.TrimSpace(traceID) == "" {
		return gateway.TraceResponse{}, fmt.Errorf("trace_id is required")
	}
	if containsLegacyIdentity(traceID) {
		return gateway.TraceResponse{}, fmt.Errorf("trace_id contains forbidden legacy identity")
	}
	query := url.Values{}
	query.Set("trace_id", traceID)
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/v1/traces", query)
	if err != nil {
		return gateway.TraceResponse{}, err
	}
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return gateway.TraceResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return gateway.TraceResponse{}, fmt.Errorf("gateway trace returned status %d", resp.StatusCode)
	}
	var response gateway.TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return gateway.TraceResponse{}, err
	}
	if containsLegacyIdentity(response.TraceID) {
		return gateway.TraceResponse{}, fmt.Errorf("gateway trace contains forbidden legacy identity")
	}
	for _, event := range response.Events {
		if containsLegacyIdentity(event.TraceID) || containsLegacyIdentity(event.SessionID) || containsLegacyIdentity(event.DeviceID) {
			return gateway.TraceResponse{}, fmt.Errorf("gateway trace contains forbidden legacy identity")
		}
	}
	return response, nil
}

func validateFirmwareDeviceReportGatewayIdentity(schemaVersion string, service string) error {
	if containsLegacyIdentity(schemaVersion) || containsLegacyIdentity(service) {
		return fmt.Errorf("gateway device report contains forbidden legacy identity")
	}
	if strings.TrimSpace(schemaVersion) != gateway.DeviceRegistrySchemaVersion || strings.TrimSpace(service) != gateway.DeviceRegistryServiceName {
		return fmt.Errorf("gateway device report is missing required A21 Gateway identity")
	}
	return nil
}

func validateFirmwareDeviceReportDevices(devices []firmwarecheck.DeviceIdentityRecord) error {
	for _, device := range devices {
		values := []string{
			device.DeviceID,
			device.Firmware.ID,
			device.Firmware.Version,
			device.Firmware.Board,
			device.Firmware.Commit,
			device.LastEvent,
			device.LastTouchSource,
			device.LastTraceID,
			device.LastSessionID,
		}
		for _, value := range values {
			if containsLegacyIdentity(value) {
				return fmt.Errorf("gateway device report contains forbidden legacy identity")
			}
		}
		for key, value := range device.Capabilities {
			if containsLegacyIdentity(key) || containsLegacyIdentity(value) {
				return fmt.Errorf("gateway device report contains forbidden legacy identity")
			}
		}
		for key, value := range device.RuntimeEcho {
			if containsLegacyIdentity(key) || containsLegacyIdentity(value) {
				return fmt.Errorf("gateway device report contains forbidden legacy identity")
			}
		}
	}
	return nil
}

func containsLegacyIdentity(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "x21") || strings.Contains(lower, "v21")
}

func firmwareDeviceReportEndpoint(gatewayBaseURL string) (string, string, error) {
	return firmwareGatewayEndpoint(gatewayBaseURL, "/v1/devices", nil)
}

func firmwareGatewayEndpoint(gatewayBaseURL string, endpointPath string, query url.Values) (string, string, error) {
	parsed, err := url.Parse(gatewayBaseURL)
	if err != nil {
		return "", "", fmt.Errorf("gateway URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", fmt.Errorf("gateway URL must use http or https")
	}
	if parsed.Host == "" {
		return "", "", fmt.Errorf("gateway URL must include a host")
	}
	if parsed.User != nil {
		return "", "", fmt.Errorf("gateway URL must not include credentials")
	}
	if strings.Contains(strings.ToLower(parsed.String()), "x21") || strings.Contains(strings.ToLower(parsed.String()), "v21") {
		return "", "", fmt.Errorf("gateway URL contains forbidden legacy identity")
	}
	if port := parsed.Port(); port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber <= 0 || portNumber > 65535 {
			return "", "", fmt.Errorf("gateway URL port is invalid")
		}
		if runtimeguard.DefaultConfig().IsLegacyEndpointPort(portNumber) {
			return "", "", fmt.Errorf("gateway URL uses forbidden legacy internal port")
		}
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	basePath := strings.TrimRight(parsed.Path, "/")
	parsed.Path = basePath
	safeGatewayURL := parsed.String()
	parsed.Path = basePath + endpointPath
	if query != nil {
		parsed.RawQuery = query.Encode()
	}
	return parsed.String(), safeGatewayURL, nil
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

func validateA21ReportDir(outputDir string) error {
	lowerPath := strings.ToLower(filepath.Clean(outputDir))
	if strings.Contains(lowerPath, "x21") || strings.Contains(lowerPath, "v21") {
		return fmt.Errorf("report directory contains forbidden legacy identity")
	}
	return nil
}

func validateA21InputPath(path string) error {
	lowerPath := strings.ToLower(filepath.Clean(path))
	if strings.Contains(lowerPath, "x21") || strings.Contains(lowerPath, "v21") {
		return fmt.Errorf("path contains forbidden legacy identity")
	}
	return nil
}

func writeAudioFrontEndEvalReport(outputDir string, report audioFrontEndEvalCLIReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-audio-front-end-eval-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONAudioFrontEndEval(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeFirmwareDeviceReport(outputDir string, report firmwareDeviceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-devices-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.DeviceReportPath = reportPath
	if err := writeJSONFirmwareDeviceReport(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeOfficePreflightReport(outputDir string, report officePreflightReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-office-preflight-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONOfficePreflight(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeOfficeHandoffReport(outputDir string, report officeHandoffReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-office-handoff-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONOfficeHandoff(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeOfficeAcceptanceReport(outputDir string, report officeAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-office-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONOfficeAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanIdentityAcceptanceReport(outputDir string, report stackChanIdentityAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-identity-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanIdentityAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanPhysicalEvidenceReport(outputDir string, report stackChanPhysicalEvidenceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-physical-evidence-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanPhysicalEvidence(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanCapabilityAcceptanceReport(outputDir string, report stackChanCapabilityAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-capability-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanCapabilityAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
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

func writeProviderSmokeReport(outputDir string, report providers.ProviderSmokeReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-provider-smoke-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONProviderSmoke(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
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
	report.ReportPath = reportPath
	if err := writeJSONV21AdapterSmoke(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeLANProbeReport(outputDir string, report lanProbeReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-lan-probe-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONLANProbe(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
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

type latencyBenchReport struct {
	Mode       string               `json:"mode"`
	Iterations int                  `json:"iterations"`
	Metadata   latencyBenchMetadata `json:"metadata"`
	Summary    latencyBenchSummary  `json:"summary"`
	OK         bool                 `json:"ok"`
	ReportPath string               `json:"report_path,omitempty"`
}

type latencyBenchMetadata struct {
	GeneratedAt   string                         `json:"generated_at"`
	CurrentCommit string                         `json:"current_commit,omitempty"`
	Fingerprint   runtimeguard.Fingerprint       `json:"fingerprint"`
	Proxy         runtimeguard.ProxyPolicyReport `json:"proxy"`
}

type latencyBenchSummary struct {
	MockTurnMS        latencyBenchSeries `json:"mock_turn_ms"`
	ProfessionalMS    latencyBenchSeries `json:"professional_turn_ms"`
	BargeInStopMS     latencyBenchSeries `json:"barge_in_stop_ms"`
	AudioWSDownlinkMS latencyBenchSeries `json:"audio_ws_downlink_ms"`
	AudioWSBargeInMS  latencyBenchSeries `json:"audio_ws_barge_in_stop_ms"`
}

type latencyBenchSeries struct {
	Samples int     `json:"samples"`
	P50MS   float64 `json:"p50_ms"`
	P95MS   float64 `json:"p95_ms"`
}

func runLatencyBench(args []string, stdout io.Writer, stderr io.Writer) int {
	mock := false
	iterations := 5
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 latency-bench --mock --iterations 5 [--output-dir reports]")
			return 0
		case "--mock":
			mock = true
		case "--iterations":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--iterations requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 {
				fmt.Fprintln(stderr, "--iterations must be a positive integer")
				return 2
			}
			iterations = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown latency-bench option %q\n", args[i])
			return 2
		}
	}
	if !mock {
		fmt.Fprintln(stderr, "--mock is required until real provider/device benchmarks are implemented")
		return 2
	}
	report, err := runMockLatencyBench(iterations)
	if err != nil {
		fmt.Fprintf(stderr, "latency bench failed: %v\n", err)
		return 1
	}
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "latency bench report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeLatencyBenchReport(outputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write latency bench report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(stderr, "encode latency bench report: %v\n", err)
		return 1
	}
	return 0
}

func runMockLatencyBench(iterations int) (latencyBenchReport, error) {
	handler := gateway.NewServer().Handler()
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()
	mockTurn := make([]time.Duration, 0, iterations)
	professional := make([]time.Duration, 0, iterations)
	bargeIn := make([]time.Duration, 0, iterations)
	audioDownlink := make([]time.Duration, 0, iterations)
	audioBargeIn := make([]time.Duration, 0, iterations)
	for i := 0; i < iterations; i++ {
		duration, err := measureGatewayRequest(handler, http.MethodPost, "/v1/mock-turn", `{"device_id":"stackchan-bench-001","text":"先说，我在","mode":"workmate"}`)
		if err != nil {
			return latencyBenchReport{}, err
		}
		mockTurn = append(mockTurn, duration)

		duration, err = measureGatewayRequest(handler, http.MethodPost, "/v1/mock-turn", `{"device_id":"stackchan-bench-001","text":"查一下语音唤醒误触发","mode":"professional"}`)
		if err != nil {
			return latencyBenchReport{}, err
		}
		professional = append(professional, duration)

		duration, err = measureGatewayRequest(handler, http.MethodPost, "/v1/mock-interrupt", `{"device_id":"stackchan-bench-001","mode":"workmate"}`)
		if err != nil {
			return latencyBenchReport{}, err
		}
		bargeIn = append(bargeIn, duration)

		duration, err = measureGatewayAudioDownlink(httpServer.URL, uint64(i+1))
		if err != nil {
			return latencyBenchReport{}, err
		}
		audioDownlink = append(audioDownlink, duration)

		duration, err = measureGatewayAudioBargeIn(httpServer.URL, uint64(i+1))
		if err != nil {
			return latencyBenchReport{}, err
		}
		audioBargeIn = append(audioBargeIn, duration)
	}
	return latencyBenchReport{
		Mode:       "mock",
		Iterations: iterations,
		Metadata:   buildLatencyBenchMetadata(),
		Summary: latencyBenchSummary{
			MockTurnMS:        latencySeries(mockTurn),
			ProfessionalMS:    latencySeries(professional),
			BargeInStopMS:     latencySeries(bargeIn),
			AudioWSDownlinkMS: latencySeries(audioDownlink),
			AudioWSBargeInMS:  latencySeries(audioBargeIn),
		},
		OK: true,
	}, nil
}

func buildLatencyBenchMetadata() latencyBenchMetadata {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	return latencyBenchMetadata{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		CurrentCommit: currentGitCommit(projectRoot),
		Fingerprint:   runtimeguard.DetectFingerprint(context.Background(), runtimeguard.OSRunner{}, os.Environ()),
		Proxy:         runtimeguard.EvaluateProxyPolicy(os.Environ()),
	}
}

func writeLatencyBenchReport(outputDir string, report latencyBenchReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-latency-bench-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func measureGatewayAudioDownlink(serverURL string, seq uint64) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	started := time.Now()
	conn, _, err := websocket.Dial(ctx, latencyBenchWebSocketURL(serverURL, "/ws/audio"), nil)
	if err != nil {
		return 0, err
	}
	defer conn.Close(websocket.StatusNormalClosure, "a21 latency bench done")

	traceID := fmt.Sprintf("a21-trace-audio-bench-%06d", seq)
	sessionID := fmt.Sprintf("a21-session-audio-bench-%06d", seq)
	if err := writeLatencyBenchAudioFrame(ctx, conn, seq, traceID, sessionID, latencyBenchPCM16Base64(0)); err != nil {
		return 0, err
	}

	for i := 0; i < 2; i++ {
		var event protocol.Envelope
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			return 0, err
		}
		if event.Kind != protocol.KindControlEvent {
			return 0, fmt.Errorf("audio bench event %d kind %q, want %q", i, event.Kind, protocol.KindControlEvent)
		}
	}
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		return 0, err
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		return 0, fmt.Errorf("audio bench playback kind %q, want %q", playback.Kind, protocol.KindAudioPlaybackChunk)
	}
	if playback.TraceID != traceID || playback.SessionID != sessionID {
		return 0, fmt.Errorf("audio bench playback trace/session mismatch")
	}
	return time.Since(started), nil
}

func measureGatewayAudioBargeIn(serverURL string, seq uint64) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, latencyBenchWebSocketURL(serverURL, "/ws/audio"), nil)
	if err != nil {
		return 0, err
	}
	defer conn.Close(websocket.StatusNormalClosure, "a21 latency bench done")

	traceID := fmt.Sprintf("a21-trace-audio-barge-bench-%06d", seq)
	sessionID := fmt.Sprintf("a21-session-audio-barge-bench-%06d", seq)
	if err := writeLatencyBenchAudioFrame(ctx, conn, seq*2, traceID, sessionID, latencyBenchPCM16Base64(0)); err != nil {
		return 0, err
	}
	for i := 0; i < 2; i++ {
		var event protocol.Envelope
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			return 0, err
		}
		if event.Kind != protocol.KindControlEvent {
			return 0, fmt.Errorf("audio barge-in setup event %d kind %q, want %q", i, event.Kind, protocol.KindControlEvent)
		}
	}
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		return 0, err
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		return 0, fmt.Errorf("audio barge-in setup playback kind %q, want %q", playback.Kind, protocol.KindAudioPlaybackChunk)
	}

	started := time.Now()
	if err := writeLatencyBenchAudioFrame(ctx, conn, seq*2+1, traceID, sessionID, latencyBenchPCM16Base64(12000)); err != nil {
		return 0, err
	}
	var interrupted protocol.Envelope
	if err := wsjson.Read(ctx, conn, &interrupted); err != nil {
		return 0, err
	}
	duration := time.Since(started)
	if interrupted.Kind != protocol.KindControlEvent {
		return 0, fmt.Errorf("audio barge-in event kind %q, want %q", interrupted.Kind, protocol.KindControlEvent)
	}
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(interrupted.Payload, &payload); err != nil {
		return 0, err
	}
	if payload.State != protocol.ExpressionInterrupted {
		return 0, fmt.Errorf("audio barge-in state %q, want %q", payload.State, protocol.ExpressionInterrupted)
	}
	if payload.StreamID == "" {
		return 0, fmt.Errorf("audio barge-in missing stream_id")
	}
	return duration, nil
}

func writeLatencyBenchAudioFrame(ctx context.Context, conn *websocket.Conn, seq uint64, traceID string, sessionID string, dataBase64 string) error {
	payload, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   dataBase64,
	})
	if err != nil {
		return err
	}
	return wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-bench-001",
		Kind:      protocol.KindAudioFrame,
		Seq:       seq,
		TraceID:   traceID,
		SessionID: sessionID,
		Payload:   payload,
	})
}

func latencyBenchPCM16Base64(sample int16) string {
	sampleCount := 16000 * 20 / 1000
	data := make([]byte, sampleCount*2)
	for i := 0; i < sampleCount; i++ {
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func latencyBenchWebSocketURL(serverURL string, path string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http") + path
}

func measureGatewayRequest(handler http.Handler, method string, path string, body string) (time.Duration, error) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	started := time.Now()
	handler.ServeHTTP(rec, req)
	duration := time.Since(started)
	if rec.Code != http.StatusOK {
		return 0, fmt.Errorf("%s %s returned status %d: %s", method, path, rec.Code, rec.Body.String())
	}
	return duration, nil
}

func latencySeries(samples []time.Duration) latencyBenchSeries {
	return latencyBenchSeries{
		Samples: len(samples),
		P50MS:   percentileMS(samples, 0.50),
		P95MS:   percentileMS(samples, 0.95),
	}
}

func percentileMS(samples []time.Duration, quantile float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	rank := int(math.Ceil(quantile*float64(len(sorted)))) - 1
	if rank < 0 {
		rank = 0
	}
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return float64(sorted[rank].Microseconds()) / 1000
}

func jitterMS(samples []time.Duration) float64 {
	if len(samples) == 0 {
		return 0
	}
	minimum := samples[0]
	maximum := samples[0]
	for _, sample := range samples[1:] {
		if sample < minimum {
			minimum = sample
		}
		if sample > maximum {
			maximum = sample
		}
	}
	return float64((maximum - minimum).Microseconds()) / 1000
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

func writeJSONLANProbe(writer io.Writer, report lanProbeReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONAudioFrontEndPlan(writer io.Writer, report audio.FrontEndPlan) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONAudioFrontEndEval(writer io.Writer, report audioFrontEndEvalCLIReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONFirmwareDeviceReport(writer io.Writer, report firmwareDeviceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONOfficePreflight(writer io.Writer, report officePreflightReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONOfficeHandoff(writer io.Writer, report officeHandoffReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONOfficeAcceptance(writer io.Writer, report officeAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanIdentityAcceptance(writer io.Writer, report stackChanIdentityAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanPhysicalEvidence(writer io.Writer, report stackChanPhysicalEvidenceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanCapabilityAcceptance(writer io.Writer, report stackChanCapabilityAcceptanceReport) error {
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
	if err := http.ListenAndServe(addr, newGatewayServerFromEnv(os.Environ()).Handler()); err != nil {
		fmt.Fprintf(stderr, "gateway: %v\n", err)
		return 1
	}
	return 0
}

func newGatewayServerFromEnv(env []string) *gateway.Server {
	return gateway.NewServerWithOptions(gateway.ServerOptions{
		VoiceProvider: providers.NewGatewayVoiceProviderFromEnv(env),
	})
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

func runFirmwareCurrentArtifactCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.LatestArtifactOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		ArtifactDir:  "firmware/artifacts",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-current-artifact-check --commit <expected-git-commit> [--artifact-dir firmware/artifacts]")
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
			fmt.Fprintln(stdout, "a21 firmware-upload-check --artifact firmware/artifacts/<a21-stackchan...bin> --commit <expected-git-commit> --port /dev/cu.usbmodemXXXX")
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
			fmt.Fprintln(stdout, "a21 firmware-device-check --artifact firmware/artifacts/<a21-stackchan...bin> --device-report reports/devices.json --device-id stackchan-001 --commit <git-sha> --max-device-age-ms 300000")
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

func writeJSONFirmwareFlashPlan(writer io.Writer, result firmwarecheck.FlashPlanResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
