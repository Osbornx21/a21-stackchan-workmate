package app

import (
	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/buildinfo"
	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/runtimeguard"
	"a21.local/a21/internal/v21adapter"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

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
	case "demo":
		return runDemo(args[1:], stdout, stderr)
	case "product-readiness":
		return runProductReadiness(args[1:], stdout, stderr)
	case "server-side-readiness-bundle":
		return runServerSideReadinessBundle(args[1:], stdout, stderr)
	case "gate":
		return runGate(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "provider-smoke":
		return runProviderSmoke(args[1:], stdout, stderr)
	case "provider-compat-matrix":
		return runProviderCompatMatrix(args[1:], stdout, stderr)
	case "provider-evidence-package":
		return runProviderEvidencePackage(args[1:], stdout, stderr)
	case "provider-evidence-import":
		return runProviderEvidenceImport(args[1:], stdout, stderr)
	case "provider-latency-bench":
		return runProviderLatencyBench(args[1:], stdout, stderr)
	case "xiaozhi-voice-bench":
		return runXiaozhiVoiceBench(args[1:], stdout, stderr)
	case "xiaozhi-realtime-parity":
		return runXiaozhiRealtimeParity(args[1:], stdout, stderr)
	case "xiaozhi-streaming-provider-readiness":
		return runXiaozhiStreamingProviderReadiness(args[1:], stdout, stderr)
	case "xiaozhi-professional-bench":
		return runXiaozhiProfessionalBench(args[1:], stdout, stderr)
	case "xiaozhi-physical-evidence":
		return runXiaozhiPhysicalEvidence(args[1:], stdout, stderr)
	case "xiaozhi-instrument-observation":
		return runXiaozhiInstrumentObservation(args[1:], stdout, stderr)
	case "physical-stackchan-evidence":
		return runPhysicalStackChanEvidence(args[1:], stdout, stderr)
	case "lan-probe":
		return runLANProbe(args[1:], stdout, stderr)
	case "local-voice-loopback":
		return runLocalVoiceLoopback(args[1:], stdout, stderr)
	case "stackchan-accept":
		return runStackChanAccept(args[1:], stdout, stderr)
	case "stackchan-local-tts-playback":
		return runStackChanLocalTTSPlayback(args[1:], stdout, stderr)
	case "stackchan-fast-companion-turn":
		return runStackChanFastCompanionTurn(args[1:], stdout, stderr)
	case "latency-bench":
		return runLatencyBench(args[1:], stdout, stderr)
	case "firmware-check":
		return runFirmwareCheck(args[1:], stdout, stderr)
	default:
		if code, ok := runDeprecatedGateAlias(args, stdout, stderr); ok {
			return code
		}
		if code, ok := runDeprecatedStackChanAcceptAlias(args, stdout, stderr); ok {
			return code
		}
		if code, ok := runDeprecatedFirmwareCheckAlias(args, stdout, stderr); ok {
			return code
		}
		if code, ok := runDeprecatedPlanExecuteAlias(args, stdout, stderr); ok {
			return code
		}
		if code, ok := runAuxiliaryCommandAlias(args, stdout, stderr); ok {
			return code
		}
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
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
		report.ReportPath = filepath.Base(reportPath)
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

func parseRatioCLIOption(args []string, index *int, stderr io.Writer, option string) (float64, bool) {
	if *index+1 >= len(args) || strings.HasPrefix(args[*index+1], "-") {
		fmt.Fprintf(stderr, "%s requires a value\n", option)
		return 0, false
	}
	*index = *index + 1
	value, err := strconv.ParseFloat(args[*index], 64)
	if err != nil || value < 0 || value > 1 {
		fmt.Fprintf(stderr, "%s must be a number between 0 and 1\n", option)
		return 0, false
	}
	return value, true
}

func parsePositiveIntCLIOption(args []string, index *int, stderr io.Writer, option string) (int, bool) {
	if *index+1 >= len(args) || strings.HasPrefix(args[*index+1], "-") {
		fmt.Fprintf(stderr, "%s requires a value\n", option)
		return 0, false
	}
	*index = *index + 1
	value, err := strconv.Atoi(args[*index])
	if err != nil || value < 0 {
		fmt.Fprintf(stderr, "%s must be a non-negative integer\n", option)
		return 0, false
	}
	return value, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parsePositiveIntOrDefault(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
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

func validateA21ReportDir(outputDir string) error {
	if containsLegacyIdentityPathToken(outputDir) {
		return fmt.Errorf("report directory contains forbidden legacy identity")
	}
	return nil
}

func validateA21InputPath(path string) error {
	if containsLegacyIdentityPathToken(path) {
		return fmt.Errorf("path contains forbidden legacy identity")
	}
	return nil
}

func containsLegacyIdentityPathToken(path string) bool {
	cleaned := strings.ToLower(filepath.ToSlash(filepath.Clean(path)))
	for _, component := range strings.Split(cleaned, "/") {
		if containsLegacyIdentityToken(component) {
			return true
		}
	}
	return false
}

func containsLegacyIdentityToken(component string) bool {
	for _, identity := range []string{"x21", "v21"} {
		for start := 0; start < len(component); {
			idx := strings.Index(component[start:], identity)
			if idx < 0 {
				break
			}
			pos := start + idx
			beforeBoundary := pos == 0 || !isLegacyIdentityTokenChar(component[pos-1])
			after := pos + len(identity)
			afterBoundary := after == len(component) || !isLegacyIdentityTokenChar(component[after])
			if beforeBoundary && afterBoundary {
				return true
			}
			start = pos + 1
		}
	}
	return false
}

func isLegacyIdentityTokenChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
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
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONLANProbe(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
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
	report.ReportPath = filepath.Base(reportPath)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
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

func elapsedReportMS(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000
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

func writeJSONReport(writer io.Writer, report runtimeguard.PreflightReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONLANProbe(writer io.Writer, report lanProbeReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func runGateway(args []string, stdout io.Writer, stderr io.Writer) int {
	options := gatewayCLIOptions{Addr: "127.0.0.1:21080"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 gateway --addr 127.0.0.1:21080 [--product-chain host_local] [--local-ollama-base-url http://127.0.0.1:11434] [--local-ollama-model qwen2.5:0.5b] [--voice-text-max-tokens 32] [--mac-local-gateway-url ws://127.0.0.1:21081/v1/xiaozhi] [--public-gateway-url https://a21.example.com] [--warm-product-chain]")
			return 0
		case "--addr":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--addr requires a value")
				return 2
			}
			i++
			options.Addr = args[i]
		case "--product-chain", "--xiaozhi-product-chain":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintf(stderr, "%s requires a value\n", args[i])
				return 2
			}
			i++
			options.ProductChain = args[i]
		case "--local-ollama-base-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--local-ollama-base-url requires a value")
				return 2
			}
			i++
			options.LocalOllamaBaseURL = args[i]
		case "--local-ollama-model":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--local-ollama-model requires a value")
				return 2
			}
			i++
			options.LocalOllamaModel = args[i]
		case "--voice-text-max-tokens":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--voice-text-max-tokens requires a value")
				return 2
			}
			i++
			options.VoiceTextMaxTokens = args[i]
		case "--public-gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--public-gateway-url requires a value")
				return 2
			}
			i++
			options.PublicGatewayURL = args[i]
		case "--mac-local-gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--mac-local-gateway-url requires a value")
				return 2
			}
			i++
			options.MacLocalGatewayURL = args[i]
		case "--warm-product-chain":
			options.WarmProductChain = true
		default:
			fmt.Fprintf(stderr, "unknown gateway option %q\n", args[i])
			return 2
		}
	}

	env := applyXiaozhiProductChainEnvDefaults(gatewayEnvWithCLIOptions(os.Environ(), options))
	if options.WarmProductChain {
		if err := warmGatewayProductChain(context.Background(), env); err != nil {
			fmt.Fprintf(stderr, "gateway product chain warmup failed: %v\n", err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "a21 gateway listening on %s\n", options.Addr)
	if err := http.ListenAndServe(options.Addr, newGatewayServerFromEnv(env).Handler()); err != nil {
		fmt.Fprintf(stderr, "gateway: %v\n", err)
		return 1
	}
	return 0
}

type gatewayCLIOptions struct {
	Addr               string
	ProductChain       string
	LocalOllamaBaseURL string
	LocalOllamaModel   string
	VoiceTextMaxTokens string
	MacLocalGatewayURL string
	PublicGatewayURL   string
	WarmProductChain   bool
}

func gatewayEnvWithCLIOptions(env []string, options gatewayCLIOptions) []string {
	out := append([]string(nil), env...)
	out = gatewayEnvSetIfNotEmpty(out, "A21_XIAOZHI_PRODUCT_CHAIN", options.ProductChain)
	out = gatewayEnvSetIfNotEmpty(out, "A21_LOCAL_OLLAMA_BASE_URL", options.LocalOllamaBaseURL)
	out = gatewayEnvSetIfNotEmpty(out, "A21_LOCAL_OLLAMA_MODEL", options.LocalOllamaModel)
	out = gatewayEnvSetIfNotEmpty(out, "A21_VOICE_TEXT_MAX_TOKENS", options.VoiceTextMaxTokens)
	out = gatewayEnvSetIfNotEmpty(out, "A21_MAC_LOCAL_GATEWAY_URL", options.MacLocalGatewayURL)
	out = gatewayEnvSetIfNotEmpty(out, "A21_PUBLIC_GATEWAY_URL", options.PublicGatewayURL)
	return out
}

func gatewayEnvSetIfNotEmpty(env []string, key string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return env
	}
	prefix := key + "="
	out := env[:0]
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			out = append(out, entry)
		}
	}
	return append(out, prefix+value)
}

func newGatewayServerFromEnv(env []string) *gateway.Server {
	return gateway.NewServerWithOptions(newGatewayServerOptionsFromEnv(env))
}

func newGatewayServerOptionsFromEnv(env []string) gateway.ServerOptions {
	env = applyXiaozhiProductChainEnvDefaults(env)
	xiaozhiVoicePipelineAdapters := providers.VoicePipelineAdaptersFromEnv(env)
	options := gateway.ServerOptions{
		VoiceProvider:                providers.NewGatewayVoiceProviderFromEnv(env),
		XiaozhiVoicePipelineAdapters: &xiaozhiVoicePipelineAdapters,
		AudioIngressConfig:           newAudioIngressConfigFromEnv(env),
		XiaozhiStockProfessional:     appEnvBool(env, "A21_XIAOZHI_STOCK_PROFESSIONAL_ROUTE"),
		MacLocalGatewayURL:           appEnvValue(env, "A21_MAC_LOCAL_GATEWAY_URL"),
		PublicGatewayURL:             appEnvValue(env, "A21_PUBLIC_GATEWAY_URL"),
	}
	if listenMaxMS, err := strconv.Atoi(strings.TrimSpace(appEnvValue(env, "A21_XIAOZHI_LISTEN_MAX_MS"))); err == nil && listenMaxMS > 0 {
		options.XiaozhiListenMaxDuration = time.Duration(listenMaxMS) * time.Millisecond
	}
	if adapterURL := strings.TrimSpace(appEnvValue(env, "A21_V21_ADAPTER_URL")); adapterURL != "" {
		client, err := v21adapter.NewHTTPClient(adapterURL)
		if err != nil {
			options.V21Client = v21ConfigurationErrorClient{err: err}
		} else {
			options.V21Client = client
		}
	}
	return options
}

func applyXiaozhiProductChainEnvDefaults(env []string) []string {
	switch gatewayProductChainMode(env) {
	case "host_local", "host-local", "product", "real":
	default:
		return env
	}
	out := append([]string(nil), env...)
	if strings.TrimSpace(appEnvValue(out, "A21_ASR_LOCAL_PROFILE")) == "" {
		out = append(out, "A21_ASR_LOCAL_PROFILE=sherpa_onnx")
	}
	if strings.TrimSpace(appEnvValue(out, "A21_TTS_FAST_PROFILE")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_TTS_BALANCED_PROFILE")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_TTS_QUALITY_PROFILE")) == "" {
		if strings.TrimSpace(appEnvValue(out, "A21_VOICE_CLONE_COMMAND")) != "" &&
			strings.TrimSpace(appEnvValue(out, "A21_VOICE_CLONE_REF_AUDIO")) != "" {
			out = append(out, "A21_TTS_FAST_PROFILE=voice_clone_cli")
		} else {
			out = append(out, "A21_TTS_FAST_PROFILE=sherpa_onnx_tts")
		}
	}
	if strings.TrimSpace(appEnvValue(out, "A21_TEXT_STREAM_PROFILE")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_PROVIDER_PRIMARY")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_LOCAL_OLLAMA_BASE_URL")) != "" &&
		strings.TrimSpace(appEnvValue(out, "A21_LOCAL_OLLAMA_MODEL")) != "" {
		out = append(out, "A21_TEXT_STREAM_PROFILE=local_ollama")
	}
	return out
}

func gatewayProductChainMode(env []string) string {
	return strings.ToLower(strings.TrimSpace(firstNonEmpty(appEnvValue(env, "A21_XIAOZHI_PRODUCT_CHAIN"), appEnvValue(env, "A21_PRODUCT_CHAIN"))))
}

func warmGatewayProductChain(ctx context.Context, env []string) error {
	switch gatewayProductChainMode(env) {
	case "host_local", "host-local", "product", "real":
	default:
		return nil
	}
	warmCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if strings.Contains(strings.ToLower(appEnvValue(env, "A21_ASR_LOCAL_PROFILE")), "sherpa") {
		result, err := audio.RunSherpaONNXASR(warmCtx, audio.LocalASROptions{})
		if err != nil {
			return fmt.Errorf("ASR warmup failed")
		}
		if result.Report.Status != "passed" {
			return fmt.Errorf("ASR warmup did not pass")
		}
	}
	textProfile := firstNonEmpty(appEnvValue(env, "A21_TEXT_STREAM_PROFILE"), appEnvValue(env, "A21_PROVIDER_PRIMARY"))
	if xiaozhiVoiceBenchNonMockStageProfile(textProfile) {
		if _, err := providers.RunTextStreamCompletionFromEnv(warmCtx, env, providers.TextStreamCompletionOptions{
			ProviderName: textProfile,
			Prompt:       "A21 warmup. Reply briefly.",
			MaxTokens:    8,
		}); err != nil {
			return fmt.Errorf("text stream warmup failed")
		}
	}
	ttsProfile := firstNonEmpty(appEnvValue(env, "A21_TTS_FAST_PROFILE"), appEnvValue(env, "A21_TTS_BALANCED_PROFILE"), appEnvValue(env, "A21_TTS_QUALITY_PROFILE"))
	normalizedTTSProfile := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(ttsProfile)), "-", "_")
	if strings.Contains(normalizedTTSProfile, "sherpa") {
		outputDir := filepath.Join(os.TempDir(), "a21-product-chain-warmup")
		_ = os.RemoveAll(outputDir)
		defer os.RemoveAll(outputDir)
		report, err := audio.SynthesizeSherpaONNX(warmCtx, audio.LocalTTSOptions{
			Text:      "A21 warmup",
			OutputDir: outputDir,
			SpeakerID: 21,
		})
		if err != nil {
			return fmt.Errorf("TTS warmup failed")
		}
		if report.Status != "passed" {
			return fmt.Errorf("TTS warmup did not pass")
		}
	} else if normalizedTTSProfile == "voice_clone_cli" {
		outputDir := filepath.Join(os.TempDir(), "a21-product-chain-warmup")
		_ = os.RemoveAll(outputDir)
		defer os.RemoveAll(outputDir)
		clone := voiceCloneRuntimeOptionsFromEnv(env)
		report, err := audio.SynthesizeVoiceCloneCLI(warmCtx, audio.LocalTTSOptions{
			Text:                         "A21 warmup",
			OutputDir:                    outputDir,
			VoiceCloneCommand:            clone.Command,
			VoiceCloneModel:              clone.Model,
			VoiceCloneReferenceAudioPath: clone.ReferenceAudioPath,
			VoiceCloneReferenceText:      clone.ReferenceText,
			VoiceCloneReferenceTextPath:  clone.ReferenceTextPath,
			VoiceClonePersona:            clone.Persona,
			VoiceCloneStyle:              clone.Style,
		})
		if err != nil {
			return fmt.Errorf("voice clone TTS warmup failed")
		}
		if report.Status != "passed" {
			return fmt.Errorf("voice clone TTS warmup did not pass")
		}
	} else if normalizedTTSProfile == "iflytek_tts" || normalizedTTSProfile == "iflytek" || normalizedTTSProfile == "xfyun" || normalizedTTSProfile == "xfyun_tts" {
		outputDir := filepath.Join(os.TempDir(), "a21-product-chain-warmup")
		_ = os.RemoveAll(outputDir)
		defer os.RemoveAll(outputDir)
		report, err := audio.SynthesizeIflytekTTS(warmCtx, audio.LocalTTSOptions{
			Text:      "A21 warmup",
			OutputDir: outputDir,
		})
		if err != nil {
			return fmt.Errorf("Iflytek TTS warmup failed")
		}
		if report.Status != "passed" {
			return fmt.Errorf("Iflytek TTS warmup did not pass")
		}
	}
	return nil
}

func newAudioIngressConfigFromEnv(env []string) audio.IngressConfig {
	config := audio.DefaultIngressConfig()
	switch strings.ToLower(strings.TrimSpace(appEnvValue(env, "A21_VAD_PREFERENCE"))) {
	case "silero":
		config.VADPreference = audio.VADDetectorPreferenceSilero
		config.SileroRunner = audio.NewCommandSileroVADRunner(audio.CommandSileroVADRunnerConfig{
			CommandPath: appEnvValue(env, "A21_SILERO_VAD_COMMAND"),
			ModelPath:   appEnvValue(env, "A21_SILERO_VAD_MODEL"),
		})
	case "rms", "":
		config.VADPreference = audio.VADDetectorPreferenceRMS
	default:
		config.VADPreference = audio.VADDetectorPreferenceRMS
	}
	if timeoutMS, err := strconv.Atoi(strings.TrimSpace(appEnvValue(env, "A21_SILERO_VAD_TIMEOUT_MS"))); err == nil && timeoutMS > 0 {
		config.VADTimeout = time.Duration(timeoutMS) * time.Millisecond
	}
	return config
}

type v21ConfigurationErrorClient struct {
	err error
}

func (c v21ConfigurationErrorClient) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	if c.err == nil {
		return v21adapter.QueryResponse{}, fmt.Errorf("v21 adapter configuration invalid")
	}
	return v21adapter.QueryResponse{}, fmt.Errorf("v21 adapter configuration invalid: %w", c.err)
}

func appEnvValue(env []string, want string) string {
	prefix := want + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

func appEnvBool(env []string, want string) bool {
	switch strings.ToLower(strings.TrimSpace(appEnvValue(env, want))) {
	case "1", "true", "yes", "on", "professional":
		return true
	default:
		return false
	}
}
