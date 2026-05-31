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

var runFirmwareBootstrapFlashCommand = runFirmwareBootstrapFlashCommandExec
var runA21ControlGuard = runA21ControlGuardExec
var synthesizeMacOSSay = audio.SynthesizeMacOSSay
var synthesizeSherpaONNX = audio.SynthesizeSherpaONNX
var runSherpaONNXASR = audio.RunSherpaONNXASR
var runSherpaONNXASRSmoke = audio.RunSherpaONNXASRSmoke

const (
	stackChanSpeakerProbeChunkDurationMS = 20
	stackChanSpeakerProbeBatchChunks     = 8
	stackChanSpeakerPrerollBatchChunks   = 8
	stackChanPlaybackPrebufferBatches    = 3
	stackChanSpeakerProbeMaxChunks       = 64
)

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

type firmwareSensorProbeFlashExecutionReport struct {
	SchemaVersion  string                                   `json:"schema_version"`
	GeneratedAtMS  int64                                    `json:"generated_at_ms"`
	DryRun         bool                                     `json:"dry_run"`
	FlashExecuted  bool                                     `json:"flash_executed"`
	ControlGuard   *runtimeguard.ControlGuardReport         `json:"control_guard,omitempty"`
	Port           string                                   `json:"port"`
	ArtifactPath   string                                   `json:"artifact_path"`
	ArtifactSHA256 string                                   `json:"artifact_sha256"`
	Commit         string                                   `json:"commit"`
	PlatformIOEnv  string                                   `json:"platformio_env"`
	Plan           firmwarecheck.SensorProbeFlashPlanResult `json:"plan"`
	Command        []string                                 `json:"command"`
	ReportPath     string                                   `json:"report_path,omitempty"`
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
	case "promotion-readiness":
		return runPromotionReadiness(args[1:], stdout, stderr)
	case "control-guard":
		return runControlGuard(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "provider-smoke":
		return runProviderSmoke(args[1:], stdout, stderr)
	case "provider-latency-bench":
		return runProviderLatencyBench(args[1:], stdout, stderr)
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
	case "local-tts-smoke":
		return runLocalTTSSmoke(args[1:], stdout, stderr)
	case "local-asr-smoke":
		return runLocalASRSmoke(args[1:], stdout, stderr)
	case "local-voice-loopback":
		return runLocalVoiceLoopback(args[1:], stdout, stderr)
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
	case "stackchan-mic-probe-acceptance":
		return runStackChanMicProbeAcceptance(args[1:], stdout, stderr)
	case "stackchan-imu-probe-acceptance":
		return runStackChanIMUProbeAcceptance(args[1:], stdout, stderr)
	case "stackchan-sensor-probe-acceptance":
		return runStackChanSensorProbeAcceptance(args[1:], stdout, stderr)
	case "stackchan-half-duplex-acceptance":
		return runStackChanHalfDuplexAcceptance(args[1:], stdout, stderr)
	case "stackchan-speaker-acceptance":
		return runStackChanSpeakerAcceptance(args[1:], stdout, stderr)
	case "stackchan-local-tts-playback":
		return runStackChanLocalTTSPlayback(args[1:], stdout, stderr)
	case "stackchan-fast-companion-turn":
		return runStackChanFastCompanionTurn(args[1:], stdout, stderr)
	case "stackchan-touch-acceptance":
		return runStackChanTouchAcceptance(args[1:], stdout, stderr)
	case "stackchan-hardware-mainline":
		return runStackChanHardwareMainline(args[1:], stdout, stderr)
	case "stackchan-official-baseline":
		return runStackChanOfficialBaseline(args[1:], stdout, stderr)
	case "stackchan-official-audio-smoke-flash-plan":
		return runStackChanOfficialAudioSmokeFlash(args[1:], false, stdout, stderr)
	case "stackchan-official-audio-smoke-flash-execute":
		return runStackChanOfficialAudioSmokeFlash(args[1:], true, stdout, stderr)
	case "stackchan-official-pcm-bridge-flash-plan":
		return runStackChanOfficialPCMBridgeFlashPlan(args[1:], stdout, stderr)
	case "stackchan-official-pcm-bridge-flash-execute":
		report := runA21ControlGuard(context.Background(), runtimeguard.ControlGuardInput{
			Config:  runtimeguard.DefaultConfig(),
			Command: "stackchan-official-pcm-bridge-flash-execute",
			Env:     os.Environ(),
			Runner:  runtimeguard.OSRunner{},
		})
		fmt.Fprintf(stderr, "stackchan official pcm bridge app flash execute is blocked by A21 control guard (%s); write an ADR and add a reviewed execute guard before app partition writes are allowed\n", summarizeControlFindings(report.Result.Findings))
		return 2
	case "stackchan-official-pcm-bridge-nvs-plan":
		return runStackChanOfficialPCMBridgeNVS(args[1:], false, stdout, stderr)
	case "stackchan-official-pcm-bridge-nvs-execute":
		return runStackChanOfficialPCMBridgeNVS(args[1:], true, stdout, stderr)
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
	case "firmware-bootstrap-flash-plan":
		return runFirmwareBootstrapFlashPlan(args[1:], stdout, stderr)
	case "firmware-bootstrap-flash-execute":
		return runFirmwareBootstrapFlashExecute(args[1:], stdout, stderr)
	case "firmware-mic-probe-flash-plan":
		return runFirmwareMicProbeFlashPlan(args[1:], stdout, stderr)
	case "firmware-mic-probe-flash-execute":
		return runFirmwareMicProbeFlashExecute(args[1:], stdout, stderr)
	case "firmware-imu-probe-flash-plan":
		return runFirmwareIMUProbeFlashPlan(args[1:], stdout, stderr)
	case "firmware-imu-probe-flash-execute":
		return runFirmwareIMUProbeFlashExecute(args[1:], stdout, stderr)
	case "firmware-sensor-probe-flash-plan":
		return runFirmwareSensorProbeFlashPlan(args[1:], stdout, stderr)
	case "firmware-sensor-probe-flash-execute":
		return runFirmwareSensorProbeFlashExecute(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

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

func runLocalTTSSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	text := "A21 本地语音链路测试。"
	engine := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_TTS_ENGINE"), "sherpa_onnx"))
	voice := strings.TrimSpace(os.Getenv("A21_LOCAL_TTS_VOICE"))
	modelDir := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_MODEL_DIR"))
	speakerID := parsePositiveIntOrDefault(os.Getenv("A21_SHERPA_ONNX_SPEAKER_ID"), 21)
	outputDir := "reports"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 local-tts-smoke [--engine sherpa_onnx|macos_say] [--text <text>] [--voice Tingting] [--model-dir <dir>] [--speaker-id 21] [--output-dir reports]")
			return 0
		case "--engine":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--engine requires a value")
				return 2
			}
			i++
			engine = args[i]
		case "--text":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--text requires a value")
				return 2
			}
			i++
			text = args[i]
		case "--voice":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--voice requires a value")
				return 2
			}
			i++
			voice = args[i]
		case "--model-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--model-dir requires a value")
				return 2
			}
			i++
			modelDir = args[i]
		case "--speaker-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--speaker-id requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 {
				fmt.Fprintln(stderr, "--speaker-id must be a positive integer")
				return 2
			}
			speakerID = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown local-tts-smoke option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(outputDir); err != nil {
		fmt.Fprintf(stderr, "local TTS report dir invalid: %v\n", err)
		return 1
	}
	report, err := synthesizeLocalTTS(context.Background(), localTTSRuntimeOptions{
		Engine:    engine,
		Text:      text,
		Voice:     voice,
		ModelDir:  modelDir,
		SpeakerID: speakerID,
		OutputDir: outputDir,
	})
	if err != nil {
		fmt.Fprintf(stderr, "local TTS smoke failed: %v\n", err)
		return 1
	}
	reportPath, err := writeLocalTTSSmokeReport(outputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write local TTS smoke report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONLocalTTSSmoke(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode local TTS smoke report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}

type localTTSRuntimeOptions struct {
	Engine    string
	Text      string
	Voice     string
	ModelDir  string
	SpeakerID int
	OutputDir string
}

func synthesizeLocalTTS(ctx context.Context, options localTTSRuntimeOptions) (audio.LocalTTSReport, error) {
	engine, err := normalizeLocalTTSEngine(options.Engine)
	if err != nil {
		return audio.LocalTTSReport{}, err
	}
	ttsOptions := audio.LocalTTSOptions{
		Text:      options.Text,
		Voice:     options.Voice,
		OutputDir: options.OutputDir,
		ModelDir:  options.ModelDir,
		SpeakerID: options.SpeakerID,
	}
	switch engine {
	case "macos_say":
		return synthesizeMacOSSay(ctx, ttsOptions)
	case "sherpa_onnx":
		return synthesizeSherpaONNX(ctx, ttsOptions)
	default:
		return audio.LocalTTSReport{}, fmt.Errorf("unsupported local TTS engine")
	}
}

func normalizeLocalTTSEngine(raw string) (string, error) {
	engine := strings.ToLower(strings.TrimSpace(firstNonEmpty(raw, "sherpa_onnx")))
	engine = strings.ReplaceAll(engine, "-", "_")
	switch engine {
	case "sherpa", "sherpa_onnx":
		return "sherpa_onnx", nil
	case "macos", "macos_say", "say":
		return "macos_say", nil
	default:
		return "", fmt.Errorf("unsupported local TTS engine")
	}
}

func runLocalASRSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	engine := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_ASR_ENGINE"), "sherpa_onnx"))
	family := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_FAMILY"))
	modelDir := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_MODEL_DIR"))
	wavPath := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_WAV"))
	outputDir := "reports"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 local-asr-smoke [--engine sherpa_onnx] [--family paraformer|sense_voice|streaming_zipformer] [--model-dir <dir>] [--wav <path>] [--output-dir reports]")
			return 0
		case "--engine":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--engine requires a value")
				return 2
			}
			i++
			engine = args[i]
		case "--family":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--family requires a value")
				return 2
			}
			i++
			family = args[i]
		case "--model-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--model-dir requires a value")
				return 2
			}
			i++
			modelDir = args[i]
		case "--wav":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--wav requires a value")
				return 2
			}
			i++
			wavPath = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown local-asr-smoke option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(outputDir); err != nil {
		fmt.Fprintf(stderr, "local ASR report dir invalid: %v\n", err)
		return 1
	}
	if engine, err := normalizeLocalASREngine(engine); err != nil {
		fmt.Fprintf(stderr, "local ASR engine invalid: %v\n", err)
		return 1
	} else if engine != "sherpa_onnx" {
		fmt.Fprintln(stderr, "unsupported local ASR engine")
		return 1
	}
	report, err := runSherpaONNXASRSmoke(context.Background(), audio.LocalASROptions{
		OutputDir: outputDir,
		ModelDir:  modelDir,
		Family:    family,
		WAVPath:   wavPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "local ASR smoke failed: %v\n", err)
		return 1
	}
	reportPath, err := writeLocalASRSmokeReport(outputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write local ASR smoke report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONLocalASRSmoke(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode local ASR smoke report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}

func normalizeLocalASREngine(raw string) (string, error) {
	engine := strings.ToLower(strings.TrimSpace(firstNonEmpty(raw, "sherpa_onnx")))
	engine = strings.ReplaceAll(engine, "-", "_")
	switch engine {
	case "sherpa", "sherpa_onnx":
		return "sherpa_onnx", nil
	default:
		return "", fmt.Errorf("unsupported local ASR engine")
	}
}

type localVoiceLoopbackReport struct {
	SchemaVersion             string               `json:"schema_version"`
	GeneratedAtMS             int64                `json:"generated_at_ms"`
	Metadata                  latencyBenchMetadata `json:"metadata"`
	Status                    string               `json:"status"`
	Repeat                    int                  `json:"repeat"`
	InputTextBytes            int                  `json:"input_text_bytes"`
	VADStatus                 string               `json:"vad_status"`
	VADDetector               string               `json:"vad_detector"`
	VADSpeechStartEvents      int                  `json:"vad_speech_start_events"`
	VADSpeechEndEvents        int                  `json:"vad_speech_end_events"`
	ASRProvider               string               `json:"asr_provider"`
	ASREngine                 string               `json:"asr_engine,omitempty"`
	ASRModelDir               string               `json:"asr_model_dir,omitempty"`
	ASRWAVName                string               `json:"asr_wav_name,omitempty"`
	ASRFirstPartialMS         float64              `json:"asr_first_partial_ms"`
	ASRInputDurationMS        float64              `json:"asr_input_duration_ms,omitempty"`
	ASRDecodeDurationMS       float64              `json:"asr_decode_duration_ms,omitempty"`
	ASRRealTimeFactor         float64              `json:"asr_real_time_factor,omitempty"`
	ASRTextChars              int                  `json:"asr_text_chars,omitempty"`
	ASRTranscriptPolicy       string               `json:"asr_transcript_policy,omitempty"`
	TextStreamProvider        string               `json:"text_stream_provider"`
	TextStreamFamily          string               `json:"text_stream_family"`
	TextStreamExecuted        bool                 `json:"text_stream_executed"`
	TextStreamEndpointHost    string               `json:"text_stream_endpoint_host,omitempty"`
	TextStreamFirstContentMS  float64              `json:"text_stream_first_content_ms"`
	TextStreamContentDeltas   int                  `json:"text_stream_content_delta_count"`
	TextStreamReasoningDeltas int                  `json:"text_stream_reasoning_delta_count"`
	TextStreamDone            bool                 `json:"text_stream_done"`
	LocalAckEnabled           bool                 `json:"local_ack_enabled"`
	LocalAckStatus            string               `json:"local_ack_status,omitempty"`
	LocalAckTTSProvider       string               `json:"local_ack_tts_provider,omitempty"`
	LocalAckTTSFirstAudioMS   float64              `json:"local_ack_tts_first_audio_ms,omitempty"`
	LocalAckFirstAudioTotalMS float64              `json:"local_ack_first_audio_total_ms,omitempty"`
	LocalAckAudioPath         string               `json:"local_ack_audio_path,omitempty"`
	TTSProvider               string               `json:"tts_provider"`
	TTSVoice                  string               `json:"tts_voice"`
	TTSOutputFormat           string               `json:"tts_output_format"`
	TTSAudioPath              string               `json:"tts_audio_path,omitempty"`
	TTSFirstAudioMS           float64              `json:"tts_first_audio_ms"`
	TTSFirstAudioP50MS        float64              `json:"tts_first_audio_p50_ms,omitempty"`
	TTSFirstAudioP95MS        float64              `json:"tts_first_audio_p95_ms,omitempty"`
	AnswerFirstAudioP50MS     float64              `json:"answer_first_audio_total_p50_ms,omitempty"`
	AnswerFirstAudioP95MS     float64              `json:"answer_first_audio_total_p95_ms,omitempty"`
	FirstAudioTotalP50MS      float64              `json:"first_audio_total_p50_ms,omitempty"`
	FirstAudioTotalP95MS      float64              `json:"first_audio_total_p95_ms,omitempty"`
	TotalDurationMS           float64              `json:"total_duration_ms"`
	BargeInStatus             string               `json:"barge_in_status"`
	BargeInStopP95MS          float64              `json:"barge_in_stop_p95_ms,omitempty"`
	ReportPath                string               `json:"report_path,omitempty"`
	Findings                  []string             `json:"findings,omitempty"`
}

type stackChanLocalTTSPlaybackReport struct {
	SchemaVersion           string               `json:"schema_version"`
	GeneratedAtMS           int64                `json:"generated_at_ms"`
	Metadata                latencyBenchMetadata `json:"metadata"`
	Status                  string               `json:"status"`
	GatewayURL              string               `json:"gateway_url"`
	DeviceID                string               `json:"device_id"`
	TraceID                 string               `json:"trace_id"`
	SessionID               string               `json:"session_id"`
	StreamID                string               `json:"stream_id"`
	InputTextBytes          int                  `json:"input_text_bytes"`
	TTSProvider             string               `json:"tts_provider"`
	TTSEngine               string               `json:"tts_engine,omitempty"`
	TTSVoice                string               `json:"tts_voice"`
	TTSOutputFormat         string               `json:"tts_output_format"`
	TTSAudioPath            string               `json:"tts_audio_path,omitempty"`
	TTSFirstAudioMS         float64              `json:"tts_first_audio_ms"`
	PlaybackChunks          int                  `json:"playback_chunks"`
	PlaybackBatches         int                  `json:"playback_batches"`
	ExpectedAudioDurationMS int                  `json:"expected_audio_duration_ms"`
	PhysicalSoundObserved   bool                 `json:"physical_sound_observed"`
	ReportPath              string               `json:"report_path,omitempty"`
	Findings                []string             `json:"findings,omitempty"`
}

func runLocalVoiceLoopback(args []string, stdout io.Writer, stderr io.Writer) int {
	inputText := "A21 本地语音 loopback 测试。"
	engine := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_TTS_ENGINE"), "sherpa_onnx"))
	voice := strings.TrimSpace(os.Getenv("A21_LOCAL_TTS_VOICE"))
	modelDir := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_MODEL_DIR"))
	speakerID := parsePositiveIntOrDefault(os.Getenv("A21_SHERPA_ONNX_SPEAKER_ID"), 21)
	asrProvider := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_ASR_PROVIDER"), "mock_asr"))
	asrFamily := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_FAMILY"))
	asrModelDir := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_MODEL_DIR"))
	asrWAVPath := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_WAV"))
	textProvider := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_TEXT_PROVIDER"), "mock_text_stream"))
	executeTextProvider := false
	repeat := parsePositiveIntOrDefault(os.Getenv("A21_LOCAL_VOICE_LOOPBACK_REPEAT"), 1)
	outputDir := "reports"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 local-voice-loopback [--engine sherpa_onnx|macos_say] [--asr-provider mock_asr|sherpa_onnx] [--asr-family paraformer|sense_voice|streaming_zipformer] [--asr-model-dir <dir>] [--asr-wav <path>] [--text-provider mock_text_stream|deepseek] [--execute-text-provider] [--text <text>] [--voice Tingting] [--model-dir <dir>] [--speaker-id 21] [--repeat 3] [--output-dir reports]")
			return 0
		case "--engine":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--engine requires a value")
				return 2
			}
			i++
			engine = args[i]
		case "--text":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--text requires a value")
				return 2
			}
			i++
			inputText = args[i]
		case "--voice":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--voice requires a value")
				return 2
			}
			i++
			voice = args[i]
		case "--model-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--model-dir requires a value")
				return 2
			}
			i++
			modelDir = args[i]
		case "--speaker-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--speaker-id requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 {
				fmt.Fprintln(stderr, "--speaker-id must be a positive integer")
				return 2
			}
			speakerID = value
		case "--text-provider":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--text-provider requires a value")
				return 2
			}
			i++
			textProvider = args[i]
		case "--asr-provider":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--asr-provider requires a value")
				return 2
			}
			i++
			asrProvider = args[i]
		case "--asr-family":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--asr-family requires a value")
				return 2
			}
			i++
			asrFamily = args[i]
		case "--asr-model-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--asr-model-dir requires a value")
				return 2
			}
			i++
			asrModelDir = args[i]
		case "--asr-wav":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--asr-wav requires a value")
				return 2
			}
			i++
			asrWAVPath = args[i]
		case "--execute-text-provider":
			executeTextProvider = true
		case "--repeat":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--repeat requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 {
				fmt.Fprintln(stderr, "--repeat must be a positive integer")
				return 2
			}
			repeat = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown local-voice-loopback option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(outputDir); err != nil {
		fmt.Fprintf(stderr, "local voice loopback report dir invalid: %v\n", err)
		return 1
	}
	report, err := buildLocalVoiceLoopbackReport(context.Background(), localTTSRuntimeOptions{
		Engine:    engine,
		Text:      inputText,
		Voice:     voice,
		ModelDir:  modelDir,
		SpeakerID: speakerID,
		OutputDir: outputDir,
	}, repeat, localVoiceLoopbackTextStreamOptions{
		Provider: textProvider,
		Execute:  executeTextProvider,
		Env:      os.Environ(),
	}, localVoiceLoopbackASROptions{
		Provider:  asrProvider,
		Family:    asrFamily,
		ModelDir:  asrModelDir,
		WAVPath:   asrWAVPath,
		OutputDir: outputDir,
	})
	if err != nil {
		fmt.Fprintf(stderr, "local voice loopback failed: %v\n", err)
		return 1
	}
	reportPath, err := writeLocalVoiceLoopbackReport(outputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write local voice loopback report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONLocalVoiceLoopback(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode local voice loopback report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}

type localVoiceLoopbackTextStreamOptions struct {
	Provider string
	Execute  bool
	Env      []string
	Client   *http.Client
}

type localVoiceLoopbackASROptions struct {
	Provider  string
	Family    string
	ModelDir  string
	WAVPath   string
	OutputDir string
}

func buildLocalVoiceLoopbackReport(ctx context.Context, ttsOptions localTTSRuntimeOptions, repeat int, textOptions localVoiceLoopbackTextStreamOptions, asrOptions localVoiceLoopbackASROptions) (localVoiceLoopbackReport, error) {
	if repeat <= 0 {
		repeat = 1
	}
	start := time.Now()
	frontEnd := audio.RunMockFrontEndEval()
	report := localVoiceLoopbackReport{
		SchemaVersion:        "a21.audio.local_voice_loopback.v1",
		GeneratedAtMS:        time.Now().UnixMilli(),
		Metadata:             buildLatencyBenchMetadata(),
		Status:               "failed",
		Repeat:               repeat,
		InputTextBytes:       len([]byte(ttsOptions.Text)),
		VADStatus:            frontEnd.Status,
		VADDetector:          frontEnd.Detector,
		VADSpeechStartEvents: frontEnd.SpeechStartEvents,
		VADSpeechEndEvents:   frontEnd.SpeechEndEvents,
		ASRProvider:          "mock_asr",
		TextStreamProvider:   "mock_text_stream",
		TextStreamFamily:     string(providers.ProviderFamilyTextStream),
		BargeInStatus:        "not_run",
	}

	transcript, err := runLocalVoiceLoopbackASR(ctx, asrOptions, &report)
	if err != nil {
		return report, err
	}
	if strings.TrimSpace(transcript) == "" {
		report.Findings = append(report.Findings, "ASR did not produce text")
		return report, nil
	}

	textResultCh := make(chan localVoiceLoopbackTextResult, 1)
	go func() {
		textReport := localVoiceLoopbackReport{}
		ttsText, textErr := runLocalVoiceLoopbackTextStream(ctx, transcript, textOptions, &textReport)
		textResultCh <- localVoiceLoopbackTextResult{text: ttsText, report: textReport, err: textErr}
	}()
	if err := runLocalVoiceLoopbackLocalAck(ctx, ttsOptions, &report); err != nil {
		return report, err
	}
	textResult := <-textResultCh
	mergeLocalVoiceLoopbackTextReport(&report, textResult.report)
	if textResult.err != nil {
		return report, textResult.err
	}
	ttsText := textResult.text
	if ttsText == "" {
		report.Findings = append(report.Findings, "text stream produced no content")
		return report, nil
	}

	ttsSamples := make([]time.Duration, 0, repeat)
	firstAudioTotalSamples := make([]time.Duration, 0, repeat)
	for sample := 0; sample < repeat; sample++ {
		ttsOptions.Text = ttsText
		ttsReport, err := synthesizeLocalTTS(ctx, ttsOptions)
		if err != nil {
			report.Findings = append(report.Findings, "local TTS failed")
			return report, err
		}
		report.TTSProvider = ttsReport.Provider
		report.TTSVoice = ttsReport.Voice
		report.TTSOutputFormat = ttsReport.OutputFormat
		report.TTSAudioPath = ttsReport.OutputPath
		report.TTSFirstAudioMS = ttsReport.TTSFirstAudioMS
		if ttsReport.Status != "passed" {
			report.Findings = append(report.Findings, "local TTS did not pass")
			return report, nil
		}
		ttsSamples = append(ttsSamples, time.Duration(ttsReport.TTSFirstAudioMS*1000)*time.Microsecond)
		answerStartGateMS := math.Max(report.TextStreamFirstContentMS, report.LocalAckTTSFirstAudioMS)
		firstAudioTotalSamples = append(firstAudioTotalSamples, time.Duration((report.ASRFirstPartialMS+answerStartGateMS+ttsReport.TTSFirstAudioMS)*1000)*time.Microsecond)
	}
	report.TTSFirstAudioP50MS = percentileMS(ttsSamples, 0.50)
	report.TTSFirstAudioP95MS = percentileMS(ttsSamples, 0.95)
	report.AnswerFirstAudioP50MS = percentileMS(firstAudioTotalSamples, 0.50)
	report.AnswerFirstAudioP95MS = percentileMS(firstAudioTotalSamples, 0.95)
	report.FirstAudioTotalP50MS = report.AnswerFirstAudioP50MS
	report.FirstAudioTotalP95MS = report.AnswerFirstAudioP95MS

	bench, err := runMockLatencyBench(1)
	if err != nil {
		report.Findings = append(report.Findings, "barge-in latency bench failed")
		return report, err
	}
	report.BargeInStatus = "benchmarked"
	report.BargeInStopP95MS = bench.Summary.AudioWSBargeInMS.P95MS
	report.TotalDurationMS = elapsedReportMS(start)
	report.Status = "passed"
	return report, nil
}

type localVoiceLoopbackTextResult struct {
	text   string
	report localVoiceLoopbackReport
	err    error
}

func runLocalVoiceLoopbackLocalAck(ctx context.Context, options localTTSRuntimeOptions, report *localVoiceLoopbackReport) error {
	report.LocalAckEnabled = true
	ackOptions := options
	ackOptions.Text = localCompanionAckText()
	ackReport, err := synthesizeLocalTTS(ctx, ackOptions)
	if err != nil {
		report.LocalAckStatus = "failed"
		report.Findings = append(report.Findings, "local ack TTS failed")
		return err
	}
	report.LocalAckTTSProvider = ackReport.Provider
	report.LocalAckTTSFirstAudioMS = ackReport.TTSFirstAudioMS
	report.LocalAckAudioPath = ackReport.OutputPath
	report.LocalAckFirstAudioTotalMS = report.ASRFirstPartialMS + ackReport.TTSFirstAudioMS
	if ackReport.Status != "passed" {
		report.LocalAckStatus = "failed"
		report.Findings = append(report.Findings, "local ack TTS did not pass")
		return nil
	}
	report.LocalAckStatus = "passed"
	return nil
}

func localCompanionAckText() string {
	return "嗯，我在。"
}

func mergeLocalVoiceLoopbackTextReport(report *localVoiceLoopbackReport, textReport localVoiceLoopbackReport) {
	report.TextStreamProvider = textReport.TextStreamProvider
	report.TextStreamFamily = textReport.TextStreamFamily
	report.TextStreamExecuted = textReport.TextStreamExecuted
	report.TextStreamEndpointHost = textReport.TextStreamEndpointHost
	report.TextStreamFirstContentMS = textReport.TextStreamFirstContentMS
	report.TextStreamContentDeltas = textReport.TextStreamContentDeltas
	report.TextStreamReasoningDeltas = textReport.TextStreamReasoningDeltas
	report.TextStreamDone = textReport.TextStreamDone
	if len(textReport.Findings) > 0 {
		report.Findings = append(report.Findings, textReport.Findings...)
	}
}

func runLocalVoiceLoopbackASR(ctx context.Context, options localVoiceLoopbackASROptions, report *localVoiceLoopbackReport) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(firstNonEmpty(options.Provider, "mock_asr")))
	provider = strings.ReplaceAll(provider, "-", "_")
	switch provider {
	case "", "mock", "mock_asr":
		asrStart := time.Now()
		report.ASRProvider = "mock_asr"
		report.ASRFirstPartialMS = elapsedReportMS(asrStart)
		return "a21 mock transcript", nil
	case "sherpa", "sherpa_onnx":
		result, err := runSherpaONNXASR(ctx, audio.LocalASROptions{
			OutputDir: options.OutputDir,
			ModelDir:  options.ModelDir,
			Family:    options.Family,
			WAVPath:   options.WAVPath,
		})
		if err != nil {
			report.Findings = append(report.Findings, "sherpa-onnx ASR failed")
			return "", err
		}
		report.ASRProvider = result.Report.Provider
		report.ASREngine = result.Report.Engine
		report.ASRModelDir = result.Report.ModelDir
		report.ASRWAVName = result.Report.WAVName
		report.ASRInputDurationMS = result.Report.InputDurationMS
		report.ASRDecodeDurationMS = result.Report.DecodeDurationMS
		report.ASRRealTimeFactor = result.Report.RealTimeFactor
		report.ASRTextChars = result.Report.TextChars
		report.ASRTranscriptPolicy = result.Report.TranscriptPolicy
		report.ASRFirstPartialMS = result.Report.DecodeDurationMS
		if result.Report.Status != "passed" {
			report.Findings = append(report.Findings, "sherpa-onnx ASR did not pass")
			return "", nil
		}
		return result.Transcript, nil
	default:
		report.Findings = append(report.Findings, "unsupported local ASR provider")
		return "", fmt.Errorf("unsupported local ASR provider")
	}
}

func runLocalVoiceLoopbackTextStream(ctx context.Context, prompt string, options localVoiceLoopbackTextStreamOptions, report *localVoiceLoopbackReport) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(options.Provider))
	switch provider {
	case "", "mock", "mock_text_stream":
		return runMockLocalVoiceLoopbackTextStream(report)
	case "deepseek":
		if !options.Execute {
			report.Findings = append(report.Findings, "deepseek text stream not executed; mock text stream used")
			return runMockLocalVoiceLoopbackTextStream(report)
		}
		result, err := providers.RunTextStreamCompletionFromEnv(ctx, options.Env, providers.TextStreamCompletionOptions{
			ProviderName: "deepseek",
			Prompt:       fastCompanionTextStreamPrompt(prompt),
			MaxTokens:    20,
			Client:       options.Client,
		})
		if err != nil {
			report.Findings = append(report.Findings, "deepseek text stream failed")
			return "", err
		}
		report.TextStreamProvider = result.Provider
		report.TextStreamFamily = string(result.Family)
		report.TextStreamExecuted = true
		report.TextStreamEndpointHost = result.EndpointHost
		report.TextStreamFirstContentMS = result.FirstContentMS
		report.TextStreamContentDeltas = result.ContentDeltaCount
		report.TextStreamReasoningDeltas = result.ReasoningDeltaCount
		report.TextStreamDone = result.Done
		return result.ContentText, nil
	default:
		report.Findings = append(report.Findings, "unsupported local text provider")
		return "", fmt.Errorf("unsupported local text provider")
	}
}

func fastCompanionTextStreamPrompt(transcript string) string {
	cleaned := strings.TrimSpace(transcript)
	if cleaned == "" {
		cleaned = "我在。"
	}
	return "你是 A21 桌面伙伴。用中文不超过12个字自然回应，不要解释，不要列点。用户说：" + cleaned
}

func runMockLocalVoiceLoopbackTextStream(report *localVoiceLoopbackReport) (string, error) {
	providerStart := time.Now()
	streamResult, err := providers.ParseOpenAICompatibleTextStream(strings.NewReader(strings.Join([]string{
		`data: {"choices":[{"delta":{"reasoning":"classify loopback"}}]}`,
		`data: {"choices":[{"delta":{"content":"A21 loopback response"}}]}`,
		`data: [DONE]`,
		``,
	}, "\n")))
	if err != nil {
		report.Findings = append(report.Findings, "mock text stream parse failed")
		return "", err
	}
	report.TextStreamProvider = "mock_text_stream"
	report.TextStreamFamily = string(providers.ProviderFamilyTextStream)
	report.TextStreamExecuted = false
	report.TextStreamFirstContentMS = elapsedReportMS(providerStart)
	report.TextStreamContentDeltas = streamResult.ContentDeltaCount
	report.TextStreamReasoningDeltas = streamResult.ReasoningDeltaCount
	report.TextStreamDone = streamResult.Done
	return streamResult.ContentText(), nil
}

func runStackChanLocalTTSPlayback(args []string, stdout io.Writer, stderr io.Writer) int {
	inputText := "A21 本地语音实机播放测试。"
	engine := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_TTS_ENGINE"), "sherpa_onnx"))
	voice := strings.TrimSpace(os.Getenv("A21_LOCAL_TTS_VOICE"))
	modelDir := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_MODEL_DIR"))
	speakerID := parsePositiveIntOrDefault(os.Getenv("A21_SHERPA_ONNX_SPEAKER_ID"), 21)
	gatewayURL := firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080")
	deviceID := firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-001")
	wavPath := ""
	outputDir := "reports"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-local-tts-playback [--gateway-url http://127.0.0.1:21080] [--device-id stackchan-001] [--engine sherpa_onnx|macos_say] [--text <text>] [--voice Tingting] [--model-dir <dir>] [--speaker-id 21] [--wav <a21-16k-mono-wav>] [--output-dir reports]")
			return 0
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			gatewayURL = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			deviceID = args[i]
		case "--engine":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--engine requires a value")
				return 2
			}
			i++
			engine = args[i]
		case "--text":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--text requires a value")
				return 2
			}
			i++
			inputText = args[i]
		case "--voice":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--voice requires a value")
				return 2
			}
			i++
			voice = args[i]
		case "--model-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--model-dir requires a value")
				return 2
			}
			i++
			modelDir = args[i]
		case "--speaker-id":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--speaker-id")
			if !ok {
				return 2
			}
			speakerID = value
		case "--wav":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--wav requires a value")
				return 2
			}
			i++
			wavPath = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-local-tts-playback option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(outputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan local TTS playback report dir invalid: %v\n", err)
		return 1
	}
	report, err := buildStackChanLocalTTSPlaybackReport(context.Background(), localTTSRuntimeOptions{
		Engine:    engine,
		Text:      inputText,
		Voice:     voice,
		ModelDir:  modelDir,
		SpeakerID: speakerID,
		OutputDir: outputDir,
	}, gatewayURL, deviceID, wavPath)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan local TTS playback failed: %v\n", err)
		return 1
	}
	reportPath, err := writeStackChanLocalTTSPlaybackReport(outputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan local TTS playback report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanLocalTTSPlayback(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan local TTS playback report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}

func buildStackChanLocalTTSPlaybackReport(ctx context.Context, ttsOptions localTTSRuntimeOptions, gatewayURL string, deviceID string, wavPath string) (stackChanLocalTTSPlaybackReport, error) {
	generatedAtMS := time.Now().UnixMilli()
	traceID := fmt.Sprintf("a21-trace-local-tts-playback-%d", generatedAtMS)
	sessionID := fmt.Sprintf("a21-session-local-tts-playback-%d", generatedAtMS)
	streamID := fmt.Sprintf("a21-local-tts-playback-stream-%d", generatedAtMS)
	report := stackChanLocalTTSPlaybackReport{
		SchemaVersion:         "a21.stackchan_local_tts_playback.v1",
		GeneratedAtMS:         generatedAtMS,
		Metadata:              buildLatencyBenchMetadata(),
		Status:                "failed",
		GatewayURL:            sanitizedOfficeGatewayURL(gatewayURL),
		DeviceID:              deviceID,
		TraceID:               traceID,
		SessionID:             sessionID,
		StreamID:              streamID,
		InputTextBytes:        len([]byte(ttsOptions.Text)),
		PhysicalSoundObserved: false,
	}
	if _, _, err := firmwareGatewayEndpoint(gatewayURL, "/v1/devices/control", nil); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, nil
	}
	playbackWAVPath := strings.TrimSpace(wavPath)
	if playbackWAVPath == "" {
		ttsReport, err := synthesizeLocalTTS(ctx, ttsOptions)
		if err != nil {
			report.Findings = append(report.Findings, "local TTS failed")
			return report, err
		}
		report.TTSProvider = ttsReport.Provider
		report.TTSEngine = ttsReport.Engine
		report.TTSVoice = ttsReport.Voice
		report.TTSOutputFormat = ttsReport.OutputFormat
		report.TTSAudioPath = ttsReport.OutputPath
		report.TTSFirstAudioMS = ttsReport.TTSFirstAudioMS
		if ttsReport.Status != "passed" {
			report.Findings = append(report.Findings, "local TTS did not pass")
			return report, nil
		}
		playbackWAVPath = ttsReport.OutputPath
	} else {
		if containsLegacyIdentity(playbackWAVPath) {
			report.Findings = append(report.Findings, "playback WAV path contains forbidden legacy project identity")
			return report, fmt.Errorf("playback WAV path contains forbidden legacy project identity")
		}
		report.TTSProvider = "wav_file"
		report.TTSVoice = "diagnostic_wav"
		report.TTSOutputFormat = "wav_pcm_s16le_16000_mono"
		report.TTSAudioPath = filepath.Base(playbackWAVPath)
	}
	pcmChunks, err := audio.ReadPCM16MonoWAVChunks(playbackWAVPath, 20)
	if err != nil {
		report.Findings = append(report.Findings, "local TTS wav parse failed")
		return report, err
	}
	report.PlaybackChunks = len(pcmChunks)
	report.ExpectedAudioDurationMS = len(pcmChunks) * 20
	for offset := 0; offset < len(pcmChunks); {
		batchChunks := stackChanPlaybackBatchSize(offset, len(pcmChunks))
		end := offset + batchChunks
		if end > len(pcmChunks) {
			end = len(pcmChunks)
		}
		batch := make([]protocol.AudioPlaybackChunk, 0, end-offset)
		for _, chunk := range pcmChunks[offset:end] {
			batch = append(batch, protocol.AudioPlaybackChunk{
				StreamID:     streamID,
				Codec:        protocol.AudioCodecPCMS16LE,
				SampleRateHz: chunk.SampleRateHz,
				Channels:     chunk.Channels,
				DurationMS:   chunk.DurationMS,
				DataBase64:   chunk.DataBase64,
			})
		}
		batch = padStackChanPlaybackBatch(streamID, batch)
		if _, err := postStackChanAudioPlaybackBatch(gatewayURL, deviceID, traceID, sessionID, streamID, batch); err != nil {
			report.Findings = append(report.Findings, "device playback delivery failed")
			return report, err
		}
		report.PlaybackBatches++
		time.Sleep(stackChanPlaybackBatchDelay(offset, end, len(pcmChunks), len(batch)))
		offset = end
	}
	if _, err := postStackChanSpeakerControl(gatewayURL, deviceID, protocol.ExpressionIdle, protocol.ModeWorkmate, "IDLE", traceID, sessionID, streamID, 0); err != nil {
		report.Findings = append(report.Findings, "device playback idle delivery failed")
		return report, err
	}
	report.Status = "passed"
	return report, nil
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

type stackChanMicProbeAcceptanceOptions struct {
	GatewayURL          string
	DeviceID            string
	Commit              string
	WindowMS            int
	MinFrames           int
	MinAbsPeak          int
	MinNonzeroSamples   int
	MinGatewayRMS       float64
	MinGatewayVADSpeech int
	MinDeliveryRatio    float64
	OutputDir           string
}

type stackChanMicProbeGatewayMetrics struct {
	AudioFrameTotal         int     `json:"gateway_audio_frame_total"`
	AudioIngressFramesTotal int     `json:"gateway_audio_ingress_frames_total"`
	AudioIngressRMS         float64 `json:"gateway_audio_ingress_rms"`
	AudioPlaybackChunkTotal int     `json:"gateway_audio_playback_chunk_total"`
	VADSpeechTotal          int     `json:"gateway_vad_speech_total"`
}

type stackChanMicProbeAcceptanceReport struct {
	SchemaVersion                  string                                `json:"schema_version"`
	GeneratedAtMS                  int64                                 `json:"generated_at_ms"`
	Metadata                       latencyBenchMetadata                  `json:"metadata"`
	DryRun                         bool                                  `json:"dry_run"`
	FlashAllowed                   bool                                  `json:"flash_allowed"`
	DeleteAllowed                  bool                                  `json:"delete_allowed"`
	HardwareAcceptanceScope        string                                `json:"hardware_acceptance_scope"`
	MicProbeAcceptanceStatus       string                                `json:"mic_probe_acceptance_status"`
	ProductionCapabilityPromoted   bool                                  `json:"production_capability_promoted"`
	GatewayURL                     string                                `json:"gateway_url"`
	DeviceID                       string                                `json:"device_id"`
	Commit                         string                                `json:"commit"`
	WindowMS                       int                                   `json:"window_ms,omitempty"`
	ControlTraceID                 string                                `json:"control_trace_id,omitempty"`
	ControlSessionID               string                                `json:"control_session_id,omitempty"`
	WindowStartedAtMS              int64                                 `json:"window_started_at_ms,omitempty"`
	WindowEndedAtMS                int64                                 `json:"window_ended_at_ms,omitempty"`
	Firmware                       firmwarecheck.DeviceIdentityFirmware  `json:"firmware"`
	Capabilities                   map[string]string                     `json:"capabilities,omitempty"`
	Microphone                     string                                `json:"microphone,omitempty"`
	RuntimeEcho                    map[string]string                     `json:"runtime_echo,omitempty"`
	RuntimeEchoBefore              map[string]string                     `json:"runtime_echo_before,omitempty"`
	MicFramesCaptured              int                                   `json:"mic_frames_captured"`
	AudioWSSentAudioFrames         int                                   `json:"audio_ws_sent_audio_frames"`
	MicDriverErrors                int                                   `json:"mic_driver_errors"`
	MicQueueDepth                  int                                   `json:"mic_queue_depth"`
	MicQueueDroppedFrames          int                                   `json:"mic_queue_dropped_frames"`
	MicLastAbsPeak                 int                                   `json:"mic_last_abs_peak"`
	MicLastNonzeroSamples          int                                   `json:"mic_last_nonzero_samples"`
	MicCaptureRateHz               float64                               `json:"mic_capture_rate_hz,omitempty"`
	AudioWSSentRateHz              float64                               `json:"audio_ws_sent_rate_hz,omitempty"`
	GatewayAudioIngressRateHz      float64                               `json:"gateway_audio_ingress_rate_hz,omitempty"`
	AudioWSDeliveryRatio           float64                               `json:"audio_ws_delivery_ratio,omitempty"`
	GatewayIngressDeliveryRatio    float64                               `json:"gateway_ingress_delivery_ratio,omitempty"`
	GatewayAudioFrameTotal         int                                   `json:"gateway_audio_frame_total"`
	GatewayAudioIngressFramesTotal int                                   `json:"gateway_audio_ingress_frames_total"`
	GatewayAudioIngressRMS         float64                               `json:"gateway_audio_ingress_rms"`
	GatewayAudioPlaybackChunkTotal int                                   `json:"gateway_audio_playback_chunk_total"`
	GatewayVADSpeechTotal          int                                   `json:"gateway_vad_speech_total"`
	MicFramesCapturedDelta         int                                   `json:"mic_frames_captured_delta"`
	AudioWSSentAudioFramesDelta    int                                   `json:"audio_ws_sent_audio_frames_delta"`
	MicDriverErrorsDelta           int                                   `json:"mic_driver_errors_delta"`
	MicQueueDroppedFramesDelta     int                                   `json:"mic_queue_dropped_frames_delta"`
	GatewayAudioFrameDelta         int                                   `json:"gateway_audio_frame_delta"`
	GatewayAudioIngressFramesDelta int                                   `json:"gateway_audio_ingress_frames_delta"`
	GatewayAudioPlaybackChunkDelta int                                   `json:"gateway_audio_playback_chunk_delta"`
	GatewayVADSpeechDelta          int                                   `json:"gateway_vad_speech_delta"`
	GatewayMetricsBefore           *stackChanMicProbeGatewayMetrics      `json:"gateway_metrics_before,omitempty"`
	GatewayMetricsAfter            *stackChanMicProbeGatewayMetrics      `json:"gateway_metrics_after,omitempty"`
	Thresholds                     stackChanMicProbeAcceptanceThresholds `json:"thresholds"`
	NextRequiredActions            []string                              `json:"next_required_actions"`
	ReportPath                     string                                `json:"report_path,omitempty"`
	Findings                       []officePreflightFinding              `json:"findings,omitempty"`
}

type stackChanMicProbeAcceptanceThresholds struct {
	MinFrames           int     `json:"min_frames"`
	MinAbsPeak          int     `json:"min_abs_peak"`
	MinNonzeroSamples   int     `json:"min_nonzero_samples"`
	MinGatewayRMS       float64 `json:"min_gateway_rms"`
	MinGatewayVADSpeech int     `json:"min_gateway_vad_speech"`
	MinDeliveryRatio    float64 `json:"min_delivery_ratio"`
}

type stackChanIMUProbeAcceptanceOptions struct {
	GatewayURL      string
	DeviceID        string
	Commit          string
	WindowMS        int
	MinSamples      int
	MinAccelTotalMG int
	MaxReadErrors   int
	OutputDir       string
}

type stackChanIMUProbeAcceptanceThresholds struct {
	MinSamples      int `json:"min_samples"`
	MinAccelTotalMG int `json:"min_accel_total_mg"`
	MaxReadErrors   int `json:"max_read_errors"`
}

type stackChanIMUProbeAcceptanceReport struct {
	SchemaVersion                string                                `json:"schema_version"`
	GeneratedAtMS                int64                                 `json:"generated_at_ms"`
	Metadata                     latencyBenchMetadata                  `json:"metadata"`
	DryRun                       bool                                  `json:"dry_run"`
	FlashAllowed                 bool                                  `json:"flash_allowed"`
	DeleteAllowed                bool                                  `json:"delete_allowed"`
	HardwareAcceptanceScope      string                                `json:"hardware_acceptance_scope"`
	IMUProbeAcceptanceStatus     string                                `json:"imu_probe_acceptance_status"`
	ProductionCapabilityPromoted bool                                  `json:"production_capability_promoted"`
	GatewayURL                   string                                `json:"gateway_url"`
	DeviceID                     string                                `json:"device_id"`
	Commit                       string                                `json:"commit"`
	WindowMS                     int                                   `json:"window_ms"`
	WindowStartedAtMS            int64                                 `json:"window_started_at_ms,omitempty"`
	WindowEndedAtMS              int64                                 `json:"window_ended_at_ms,omitempty"`
	Firmware                     firmwarecheck.DeviceIdentityFirmware  `json:"firmware"`
	Capabilities                 map[string]string                     `json:"capabilities,omitempty"`
	IMU                          string                                `json:"imu,omitempty"`
	RuntimeEcho                  map[string]string                     `json:"runtime_echo,omitempty"`
	RuntimeEchoBefore            map[string]string                     `json:"runtime_echo_before,omitempty"`
	IMUAvailable                 int                                   `json:"imu_available"`
	IMUSamples                   int                                   `json:"imu_samples"`
	IMUReadErrors                int                                   `json:"imu_read_errors"`
	IMUAccelMGX                  int                                   `json:"imu_accel_mg_x"`
	IMUAccelMGY                  int                                   `json:"imu_accel_mg_y"`
	IMUAccelMGZ                  int                                   `json:"imu_accel_mg_z"`
	IMUGyroMDPSX                 int                                   `json:"imu_gyro_mdps_x"`
	IMUGyroMDPSY                 int                                   `json:"imu_gyro_mdps_y"`
	IMUGyroMDPSZ                 int                                   `json:"imu_gyro_mdps_z"`
	IMUPosture                   string                                `json:"imu_posture"`
	IMUAccelTotalMG              int                                   `json:"imu_accel_total_mg"`
	IMUSamplesDelta              int                                   `json:"imu_samples_delta"`
	IMUReadErrorsDelta           int                                   `json:"imu_read_errors_delta"`
	SampleRateHz                 float64                               `json:"sample_rate_hz,omitempty"`
	Thresholds                   stackChanIMUProbeAcceptanceThresholds `json:"thresholds"`
	NextRequiredActions          []string                              `json:"next_required_actions"`
	ReportPath                   string                                `json:"report_path,omitempty"`
	Findings                     []officePreflightFinding              `json:"findings,omitempty"`
}

type stackChanSensorProbeAcceptanceOptions struct {
	GatewayURL    string
	DeviceID      string
	Commit        string
	WindowMS      int
	MinSamples    int
	MinBatteryMV  int
	MaxReadErrors int
	OutputDir     string
}

type stackChanSensorProbeAcceptanceThresholds struct {
	MinSamples    int `json:"min_samples"`
	MinBatteryMV  int `json:"min_battery_mv"`
	MaxReadErrors int `json:"max_read_errors"`
}

type stackChanSensorProbeAcceptanceReport struct {
	SchemaVersion                string                                   `json:"schema_version"`
	GeneratedAtMS                int64                                    `json:"generated_at_ms"`
	Metadata                     latencyBenchMetadata                     `json:"metadata"`
	DryRun                       bool                                     `json:"dry_run"`
	FlashAllowed                 bool                                     `json:"flash_allowed"`
	DeleteAllowed                bool                                     `json:"delete_allowed"`
	HardwareAcceptanceScope      string                                   `json:"hardware_acceptance_scope"`
	SensorProbeAcceptanceStatus  string                                   `json:"sensor_probe_acceptance_status"`
	ProductionCapabilityPromoted bool                                     `json:"production_capability_promoted"`
	GatewayURL                   string                                   `json:"gateway_url"`
	DeviceID                     string                                   `json:"device_id"`
	Commit                       string                                   `json:"commit"`
	WindowMS                     int                                      `json:"window_ms"`
	WindowStartedAtMS            int64                                    `json:"window_started_at_ms,omitempty"`
	WindowEndedAtMS              int64                                    `json:"window_ended_at_ms,omitempty"`
	Firmware                     firmwarecheck.DeviceIdentityFirmware     `json:"firmware"`
	Capabilities                 map[string]string                        `json:"capabilities,omitempty"`
	AmbientLight                 string                                   `json:"ambient_light,omitempty"`
	Proximity                    string                                   `json:"proximity,omitempty"`
	Battery                      string                                   `json:"battery,omitempty"`
	RuntimeEcho                  map[string]string                        `json:"runtime_echo,omitempty"`
	RuntimeEchoBefore            map[string]string                        `json:"runtime_echo_before,omitempty"`
	SensorAvailable              int                                      `json:"sensor_available"`
	SensorSamples                int                                      `json:"sensor_samples"`
	SensorReadErrors             int                                      `json:"sensor_read_errors"`
	AmbientLightRaw              int                                      `json:"ambient_light_raw"`
	ProximityRaw                 int                                      `json:"proximity_raw"`
	BatteryMV                    int                                      `json:"battery_mv"`
	BatteryMA                    int                                      `json:"battery_ma"`
	SensorSamplesDelta           int                                      `json:"sensor_samples_delta"`
	SensorReadErrorsDelta        int                                      `json:"sensor_read_errors_delta"`
	SampleRateHz                 float64                                  `json:"sample_rate_hz,omitempty"`
	Thresholds                   stackChanSensorProbeAcceptanceThresholds `json:"thresholds"`
	NextRequiredActions          []string                                 `json:"next_required_actions"`
	ReportPath                   string                                   `json:"report_path,omitempty"`
	Findings                     []officePreflightFinding                 `json:"findings,omitempty"`
}

type stackChanHalfDuplexAcceptanceOptions struct {
	GatewayURL        string
	DeviceID          string
	Commit            string
	WindowMS          int
	MinMicFrames      int
	MinPlaybackChunks int
	MinDeliveryRatio  float64
	OutputDir         string
}

type stackChanHalfDuplexAcceptanceThresholds struct {
	MinMicFrames      int     `json:"min_mic_frames"`
	MinPlaybackChunks int     `json:"min_playback_chunks"`
	MinDeliveryRatio  float64 `json:"min_delivery_ratio"`
}

type stackChanHalfDuplexAcceptanceReport struct {
	SchemaVersion                    string                                  `json:"schema_version"`
	GeneratedAtMS                    int64                                   `json:"generated_at_ms"`
	Metadata                         latencyBenchMetadata                    `json:"metadata"`
	DryRun                           bool                                    `json:"dry_run"`
	FlashAllowed                     bool                                    `json:"flash_allowed"`
	DeleteAllowed                    bool                                    `json:"delete_allowed"`
	HardwareAcceptanceScope          string                                  `json:"hardware_acceptance_scope"`
	HalfDuplexAcceptanceStatus       string                                  `json:"half_duplex_acceptance_status"`
	PhysicalSoundObserved            bool                                    `json:"physical_sound_observed"`
	GatewayURL                       string                                  `json:"gateway_url"`
	DeviceID                         string                                  `json:"device_id"`
	Commit                           string                                  `json:"commit"`
	WindowMS                         int                                     `json:"window_ms"`
	ControlTraceID                   string                                  `json:"control_trace_id,omitempty"`
	ControlSessionID                 string                                  `json:"control_session_id,omitempty"`
	WindowStartedAtMS                int64                                   `json:"window_started_at_ms,omitempty"`
	WindowEndedAtMS                  int64                                   `json:"window_ended_at_ms,omitempty"`
	Firmware                         firmwarecheck.DeviceIdentityFirmware    `json:"firmware"`
	Capabilities                     map[string]string                       `json:"capabilities,omitempty"`
	Microphone                       string                                  `json:"microphone,omitempty"`
	Speaker                          string                                  `json:"speaker,omitempty"`
	RuntimeEcho                      map[string]string                       `json:"runtime_echo,omitempty"`
	RuntimeEchoBefore                map[string]string                       `json:"runtime_echo_before,omitempty"`
	MicFramesCapturedDelta           int                                     `json:"mic_frames_captured_delta"`
	AudioWSSentAudioFramesDelta      int                                     `json:"audio_ws_sent_audio_frames_delta"`
	MicDriverErrorsDelta             int                                     `json:"mic_driver_errors_delta"`
	MicQueueDroppedFramesDelta       int                                     `json:"mic_queue_dropped_frames_delta"`
	PlaybackBufferTotalChunksDelta   int                                     `json:"playback_buffer_total_chunks_delta"`
	PlaybackBufferDroppedChunksDelta int                                     `json:"playback_buffer_dropped_chunks_delta"`
	SpeakerFramesPlayedDelta         int                                     `json:"speaker_frames_played_delta"`
	SpeakerDriverErrorsDelta         int                                     `json:"speaker_driver_errors_delta"`
	AudioWSDeliveryRatio             float64                                 `json:"audio_ws_delivery_ratio,omitempty"`
	GatewayIngressDeliveryRatio      float64                                 `json:"gateway_ingress_delivery_ratio,omitempty"`
	GatewayAudioFrameDelta           int                                     `json:"gateway_audio_frame_delta"`
	GatewayAudioIngressFramesDelta   int                                     `json:"gateway_audio_ingress_frames_delta"`
	GatewayAudioPlaybackChunkDelta   int                                     `json:"gateway_audio_playback_chunk_delta"`
	GatewayVADSpeechDelta            int                                     `json:"gateway_vad_speech_delta"`
	GatewayMetricsBefore             *stackChanMicProbeGatewayMetrics        `json:"gateway_metrics_before,omitempty"`
	GatewayMetricsAfter              *stackChanMicProbeGatewayMetrics        `json:"gateway_metrics_after,omitempty"`
	Thresholds                       stackChanHalfDuplexAcceptanceThresholds `json:"thresholds"`
	NextRequiredActions              []string                                `json:"next_required_actions"`
	ReportPath                       string                                  `json:"report_path,omitempty"`
	Findings                         []officePreflightFinding                `json:"findings,omitempty"`
}

type stackChanSpeakerAcceptanceOptions struct {
	GatewayURL      string
	DeviceID        string
	Commit          string
	WindowMS        int
	MockAudioChunks int
	MinPlayedFrames int
	OutputDir       string
}

type stackChanSpeakerGatewayMetrics struct {
	AudioPlaybackChunkTotal int `json:"gateway_audio_playback_chunk_total"`
}

type stackChanSpeakerAcceptanceThresholds struct {
	MinPlayedFrames  int `json:"min_played_frames"`
	MinGatewayChunks int `json:"min_gateway_chunks"`
}

type stackChanSpeakerAcceptanceReport struct {
	SchemaVersion                    string                               `json:"schema_version"`
	GeneratedAtMS                    int64                                `json:"generated_at_ms"`
	Metadata                         latencyBenchMetadata                 `json:"metadata"`
	DryRun                           bool                                 `json:"dry_run"`
	FlashAllowed                     bool                                 `json:"flash_allowed"`
	DeleteAllowed                    bool                                 `json:"delete_allowed"`
	HardwareAcceptanceScope          string                               `json:"hardware_acceptance_scope"`
	SpeakerAcceptanceStatus          string                               `json:"speaker_acceptance_status"`
	PhysicalSoundObserved            bool                                 `json:"physical_sound_observed"`
	GatewayURL                       string                               `json:"gateway_url"`
	DeviceID                         string                               `json:"device_id"`
	Commit                           string                               `json:"commit"`
	WindowMS                         int                                  `json:"window_ms"`
	MockAudioChunks                  int                                  `json:"mock_audio_chunks"`
	ExpectedAudioDurationMS          int                                  `json:"expected_audio_duration_ms"`
	StreamID                         string                               `json:"stream_id"`
	ControlTraceID                   string                               `json:"control_trace_id,omitempty"`
	ControlSessionID                 string                               `json:"control_session_id,omitempty"`
	WindowStartedAtMS                int64                                `json:"window_started_at_ms,omitempty"`
	WindowEndedAtMS                  int64                                `json:"window_ended_at_ms,omitempty"`
	Firmware                         firmwarecheck.DeviceIdentityFirmware `json:"firmware"`
	Capabilities                     map[string]string                    `json:"capabilities,omitempty"`
	Speaker                          string                               `json:"speaker,omitempty"`
	RuntimeEcho                      map[string]string                    `json:"runtime_echo,omitempty"`
	RuntimeEchoBefore                map[string]string                    `json:"runtime_echo_before,omitempty"`
	PlaybackBufferQueuedChunks       int                                  `json:"playback_buffer_queued_chunks"`
	PlaybackBufferTotalChunks        int                                  `json:"playback_buffer_total_chunks"`
	PlaybackBufferDroppedChunks      int                                  `json:"playback_buffer_dropped_chunks"`
	PlaybackBufferClearCount         int                                  `json:"playback_buffer_clear_count"`
	SpeakerFramesPlayed              int                                  `json:"speaker_frames_played"`
	SpeakerBusyTicks                 int                                  `json:"speaker_busy_ticks"`
	SpeakerDriverErrors              int                                  `json:"speaker_driver_errors"`
	SpeakerLastStreamID              string                               `json:"speaker_last_stream_id"`
	PlaybackBufferTotalChunksDelta   int                                  `json:"playback_buffer_total_chunks_delta"`
	PlaybackBufferDroppedChunksDelta int                                  `json:"playback_buffer_dropped_chunks_delta"`
	PlaybackBufferClearCountDelta    int                                  `json:"playback_buffer_clear_count_delta"`
	SpeakerFramesPlayedDelta         int                                  `json:"speaker_frames_played_delta"`
	SpeakerBusyTicksDelta            int                                  `json:"speaker_busy_ticks_delta"`
	SpeakerDriverErrorsDelta         int                                  `json:"speaker_driver_errors_delta"`
	GatewayAudioPlaybackChunkTotal   int                                  `json:"gateway_audio_playback_chunk_total"`
	GatewayAudioPlaybackChunkDelta   int                                  `json:"gateway_audio_playback_chunk_delta"`
	GatewayMetricsBefore             *stackChanSpeakerGatewayMetrics      `json:"gateway_metrics_before,omitempty"`
	GatewayMetricsAfter              *stackChanSpeakerGatewayMetrics      `json:"gateway_metrics_after,omitempty"`
	Thresholds                       stackChanSpeakerAcceptanceThresholds `json:"thresholds"`
	NextRequiredActions              []string                             `json:"next_required_actions"`
	ReportPath                       string                               `json:"report_path,omitempty"`
	Findings                         []officePreflightFinding             `json:"findings,omitempty"`
}

type stackChanTouchCaseSpec struct {
	Name            string
	AcceptanceScope string
	CoreUX          bool
	ExpectedEvent   string
	ExpectedTrace   string
	ExpectedSource  string
	ControlState    protocol.ExpressionState
	ControlMode     protocol.Mode
	DevicePrompt    string
	HumanGuidance   string
	MockAudioChunks int
	StreamID        string
	NeedsAffordance bool
}

type stackChanTouchAcceptanceOptions struct {
	GatewayURL string
	DeviceID   string
	Case       string
	WindowMS   int
	OutputDir  string
}

type stackChanTouchAcceptanceObservation struct {
	Event     string `json:"event"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	AtMS      int64  `json:"at_ms,omitempty"`
	OffsetMS  int64  `json:"offset_ms,omitempty"`
	Source    string `json:"source,omitempty"`
}

type stackChanTouchAcceptanceReport struct {
	SchemaVersion      string                                `json:"schema_version"`
	GeneratedAtMS      int64                                 `json:"generated_at_ms"`
	DryRun             bool                                  `json:"dry_run"`
	FlashAllowed       bool                                  `json:"flash_allowed"`
	DeleteAllowed      bool                                  `json:"delete_allowed"`
	AcceptanceStatus   string                                `json:"touch_acceptance_status"`
	AcceptanceScope    string                                `json:"acceptance_scope"`
	CoreUX             bool                                  `json:"core_ux"`
	NeedsAffordance    bool                                  `json:"needs_affordance"`
	GatewayURL         string                                `json:"gateway_url"`
	DeviceID           string                                `json:"device_id"`
	Case               string                                `json:"case"`
	WindowMS           int                                   `json:"window_ms"`
	ExpectedEvent      string                                `json:"expected_event"`
	ExpectedTraceEvent string                                `json:"expected_trace_event"`
	ExpectedSource     string                                `json:"expected_source"`
	DevicePrompt       string                                `json:"device_prompt"`
	HumanGuidance      string                                `json:"human_guidance"`
	ControlTraceID     string                                `json:"control_trace_id,omitempty"`
	ControlSessionID   string                                `json:"control_session_id,omitempty"`
	ObservedEvents     []stackChanTouchAcceptanceObservation `json:"observed_events,omitempty"`
	ReportPath         string                                `json:"report_path,omitempty"`
	Findings           []officePreflightFinding              `json:"findings,omitempty"`
}

type stackChanHardwareMainlineOptions struct {
	GatewayURL string
	DeviceID   string
	OutputDir  string
}

type stackChanHardwareTrack struct {
	Capability       string `json:"capability"`
	DeclaredStatus   string `json:"declared_status,omitempty"`
	TargetStatus     string `json:"target_status"`
	Track            string `json:"track"`
	PromotionAllowed bool   `json:"promotion_allowed"`
	NextAction       string `json:"next_action"`
}

type stackChanHardwareCapabilityInvariant struct {
	Capability     string `json:"capability"`
	DeclaredStatus string `json:"declared_status,omitempty"`
	Requirement    string `json:"requirement"`
	Accepted       bool   `json:"accepted"`
}

type stackChanHardwareCapabilityInvariantSpec struct {
	Capability      string
	Requirement     string
	AllowedExact    []string
	AllowedPrefixes []string
}

type stackChanHardwareMainlineReport struct {
	SchemaVersion          string                                 `json:"schema_version"`
	GeneratedAtMS          int64                                  `json:"generated_at_ms"`
	Metadata               latencyBenchMetadata                   `json:"metadata"`
	DryRun                 bool                                   `json:"dry_run"`
	FlashAllowed           bool                                   `json:"flash_allowed"`
	DeleteAllowed          bool                                   `json:"delete_allowed"`
	HardwareMainlineStatus string                                 `json:"hardware_mainline_status"`
	GatewayURL             string                                 `json:"gateway_url"`
	DeviceID               string                                 `json:"device_id"`
	Firmware               firmwarecheck.DeviceIdentityFirmware   `json:"firmware"`
	Capabilities           map[string]string                      `json:"capabilities,omitempty"`
	RuntimeEcho            map[string]string                      `json:"runtime_echo,omitempty"`
	CapabilityInvariants   []stackChanHardwareCapabilityInvariant `json:"capability_invariants"`
	OrderedTracks          []stackChanHardwareTrack               `json:"ordered_tracks"`
	NextRequiredActions    []string                               `json:"next_required_actions"`
	ReportPath             string                                 `json:"report_path,omitempty"`
	Findings               []officePreflightFinding               `json:"findings,omitempty"`
}

func (report *stackChanTouchAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
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
			report.addFinding("capability_not_declared_available", fmt.Sprintf("required StackChan capability %q is not declared available by the identity acceptance report", capability))
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
			report.addFinding("capability_not_declared_available", fmt.Sprintf("required StackChan capability %q is not declared available by the fresh device identity", capability))
		}
		observation, ok := observations[capability]
		if !ok {
			report.addFinding("capability_evidence_missing", fmt.Sprintf("physical evidence missing for required StackChan capability %q", capability))
			report.CapabilityResults = append(report.CapabilityResults, result)
			continue
		}
		result.EvidenceStatus = observation.Status
		result.EvidenceType = observation.EvidenceType
		result.ObservedAtMS = observation.ObservedAtMS
		if observation.Status != "passed" {
			report.addFinding("capability_evidence_not_passed", fmt.Sprintf("physical evidence for required StackChan capability %q is not passed", capability))
		}
		if strings.TrimSpace(observation.EvidenceType) == "" {
			report.addFinding("capability_evidence_type_missing", fmt.Sprintf("physical evidence for required StackChan capability %q is missing evidence_type", capability))
		}
		if observation.ObservedAtMS <= 0 {
			report.addFinding("capability_evidence_time_missing", fmt.Sprintf("physical evidence for required StackChan capability %q is missing observed_at_ms", capability))
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

func runStackChanMicProbeAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanMicProbeAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-mic-probe-acceptance --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> [--window-ms 0] [--min-frames 90] [--min-abs-peak 1] [--min-nonzero-samples 1] [--min-gateway-rms 0] [--min-vad-speech 0] [--min-delivery-ratio 0.95] [--output-dir reports]")
			return 0
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
		case "--window-ms":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--window-ms")
			if !ok {
				return 2
			}
			if value > 120000 {
				fmt.Fprintln(stderr, "--window-ms must be at most 120000")
				return 2
			}
			options.WindowMS = value
		case "--min-frames":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-frames")
			if !ok {
				return 2
			}
			options.MinFrames = value
		case "--min-abs-peak":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-abs-peak")
			if !ok {
				return 2
			}
			options.MinAbsPeak = value
		case "--min-nonzero-samples":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-nonzero-samples")
			if !ok {
				return 2
			}
			options.MinNonzeroSamples = value
		case "--min-gateway-rms":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--min-gateway-rms requires a value")
				return 2
			}
			i++
			value, err := strconv.ParseFloat(args[i], 64)
			if err != nil || value < 0 {
				fmt.Fprintln(stderr, "--min-gateway-rms must be a non-negative number")
				return 2
			}
			options.MinGatewayRMS = value
		case "--min-delivery-ratio":
			value, ok := parseRatioCLIOption(args, &i, stderr, "--min-delivery-ratio")
			if !ok {
				return 2
			}
			options.MinDeliveryRatio = value
		case "--min-vad-speech":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-vad-speech")
			if !ok {
				return 2
			}
			options.MinGatewayVADSpeech = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-mic-probe-acceptance option %q\n", args[i])
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
	if _, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/devices", nil); err != nil {
		fmt.Fprintf(stderr, "stackchan mic probe acceptance gateway URL invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan mic probe acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanMicProbeAcceptanceReport(options)
	reportPath, err := writeStackChanMicProbeAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan mic probe acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanMicProbeAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan mic probe acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan mic probe acceptance blocked (diagnostic only, no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan mic probe acceptance ok (diagnostic only, no flash performed)")
	return 0
}

func defaultStackChanMicProbeAcceptanceOptions() stackChanMicProbeAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanMicProbeAcceptanceOptions{
		GatewayURL:          firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:            deviceID,
		Commit:              currentGitCommit(projectRoot),
		MinFrames:           90,
		MinAbsPeak:          1,
		MinNonzeroSamples:   1,
		MinGatewayRMS:       0,
		MinGatewayVADSpeech: 0,
		MinDeliveryRatio:    0.95,
		OutputDir:           "reports",
	}
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

func buildStackChanMicProbeAcceptanceReport(options stackChanMicProbeAcceptanceOptions) stackChanMicProbeAcceptanceReport {
	report := stackChanMicProbeAcceptanceReport{
		SchemaVersion:                "a21.stackchan_mic_probe_acceptance.v1",
		GeneratedAtMS:                time.Now().UnixMilli(),
		Metadata:                     buildLatencyBenchMetadata(),
		DryRun:                       true,
		FlashAllowed:                 false,
		DeleteAllowed:                false,
		HardwareAcceptanceScope:      "diagnostic_microphone_only",
		MicProbeAcceptanceStatus:     "confirmed",
		ProductionCapabilityPromoted: false,
		GatewayURL:                   sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                     options.DeviceID,
		Commit:                       options.Commit,
		WindowMS:                     options.WindowMS,
		Thresholds: stackChanMicProbeAcceptanceThresholds{
			MinFrames:           options.MinFrames,
			MinAbsPeak:          options.MinAbsPeak,
			MinNonzeroSamples:   options.MinNonzeroSamples,
			MinGatewayRMS:       options.MinGatewayRMS,
			MinGatewayVADSpeech: options.MinGatewayVADSpeech,
			MinDeliveryRatio:    options.MinDeliveryRatio,
		},
		NextRequiredActions: []string{
			"Keep this report with the mic-probe flash receipt and Gateway device report.",
			"Do not treat this as production microphone, speaker, AEC, full-duplex, provider, or latency acceptance.",
			"Promote microphone capability only through a later ADR and release-firmware acceptance gate.",
		},
	}

	if options.WindowMS > 0 {
		runStackChanMicProbeAcceptanceWindow(&report, options)
		if len(report.Findings) > 0 {
			report.MicProbeAcceptanceStatus = "blocked"
		}
		return report
	}

	gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_failed", err.Error())
	} else {
		report.GatewayURL = gatewayReport.GatewayURL
		device, ok := findFirmwareDeviceRecord(gatewayReport.Devices, options.DeviceID)
		if !ok {
			report.addFinding("gateway_device_missing", "expected StackChan device is missing from Gateway")
		} else {
			validateStackChanMicProbeDevice(&report, options, device)
		}
	}

	metrics, err := fetchStackChanMicProbeGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_failed", err.Error())
	} else {
		report.GatewayAudioFrameTotal = metrics.AudioFrameTotal
		report.GatewayAudioIngressFramesTotal = metrics.AudioIngressFramesTotal
		report.GatewayAudioIngressRMS = metrics.AudioIngressRMS
		report.GatewayAudioPlaybackChunkTotal = metrics.AudioPlaybackChunkTotal
		report.GatewayVADSpeechTotal = metrics.VADSpeechTotal
		validateStackChanMicProbeMetrics(&report, options)
	}

	if len(report.Findings) > 0 {
		report.MicProbeAcceptanceStatus = "blocked"
	}
	return report
}

func runStackChanMicProbeAcceptanceWindow(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions) {
	beforeGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_before_failed", err.Error())
		return
	}
	beforeDevice, ok := findFirmwareDeviceRecord(beforeGatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_before_missing", "expected StackChan device is missing from Gateway before probe window")
		return
	}
	report.RuntimeEchoBefore = beforeDevice.RuntimeEcho
	beforeMicFramesCaptured := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_frames_captured")
	beforeAudioWSSentAudioFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "audio_ws_sent_audio_frames")
	beforeMicDriverErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_driver_errors")
	beforeMicQueueDroppedFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_queue_dropped_frames")

	beforeMetrics, err := fetchStackChanMicProbeGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_before_failed", err.Error())
		return
	}
	report.GatewayMetricsBefore = &beforeMetrics

	traceID := fmt.Sprintf("a21-trace-mic-probe-%d", report.GeneratedAtMS)
	sessionID := fmt.Sprintf("a21-session-mic-probe-%d", report.GeneratedAtMS)
	control, err := postStackChanMicProbeControl(options.GatewayURL, options.DeviceID, protocol.ExpressionListening, protocol.ModeWorkmate, "MIC PROBE", traceID, sessionID, true, false)
	if err != nil {
		report.addFinding("mic_probe_control_failed", err.Error())
		return
	}
	report.ControlTraceID = control.TraceID
	report.ControlSessionID = control.SessionID
	report.WindowStartedAtMS = time.Now().UnixMilli()
	time.Sleep(time.Duration(options.WindowMS) * time.Millisecond)
	report.WindowEndedAtMS = time.Now().UnixMilli()

	afterGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_after_failed", err.Error())
	} else {
		report.GatewayURL = afterGatewayReport.GatewayURL
		afterDevice, ok := findFirmwareDeviceRecord(afterGatewayReport.Devices, options.DeviceID)
		if !ok {
			report.addFinding("gateway_device_after_missing", "expected StackChan device is missing from Gateway after probe window")
		} else {
			validateStackChanMicProbeDevice(report, options, afterDevice)
			report.MicFramesCapturedDelta = report.MicFramesCaptured - beforeMicFramesCaptured
			report.AudioWSSentAudioFramesDelta = report.AudioWSSentAudioFrames - beforeAudioWSSentAudioFrames
			report.MicDriverErrorsDelta = report.MicDriverErrors - beforeMicDriverErrors
			report.MicQueueDroppedFramesDelta = report.MicQueueDroppedFrames - beforeMicQueueDroppedFrames
			validateStackChanMicProbeRuntimeDeltas(report, options)
		}
	}

	afterMetrics, err := fetchStackChanMicProbeGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_after_failed", err.Error())
	} else {
		report.GatewayMetricsAfter = &afterMetrics
		report.GatewayAudioFrameTotal = afterMetrics.AudioFrameTotal
		report.GatewayAudioIngressFramesTotal = afterMetrics.AudioIngressFramesTotal
		report.GatewayAudioIngressRMS = afterMetrics.AudioIngressRMS
		report.GatewayAudioPlaybackChunkTotal = afterMetrics.AudioPlaybackChunkTotal
		report.GatewayVADSpeechTotal = afterMetrics.VADSpeechTotal
		report.GatewayAudioFrameDelta = afterMetrics.AudioFrameTotal - beforeMetrics.AudioFrameTotal
		report.GatewayAudioIngressFramesDelta = afterMetrics.AudioIngressFramesTotal - beforeMetrics.AudioIngressFramesTotal
		report.GatewayAudioPlaybackChunkDelta = afterMetrics.AudioPlaybackChunkTotal - beforeMetrics.AudioPlaybackChunkTotal
		report.GatewayVADSpeechDelta = afterMetrics.VADSpeechTotal - beforeMetrics.VADSpeechTotal
		validateStackChanMicProbeMetricDeltas(report, options)
	}
	populateStackChanMicProbeWindowQuality(report)
	validateStackChanMicProbeDeliveryRatios(report, options)

	if _, err := postStackChanMicProbeControl(options.GatewayURL, options.DeviceID, protocol.ExpressionIdle, protocol.ModeWorkmate, "IDLE", report.ControlTraceID, report.ControlSessionID, false, false); err != nil {
		report.addFinding("mic_probe_idle_control_failed", err.Error())
	}
}

func validateStackChanMicProbeDevice(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	report.Firmware = device.Firmware
	report.Capabilities = device.Capabilities
	report.RuntimeEcho = device.RuntimeEcho
	report.Microphone = device.Capabilities["microphone"]
	if device.DeviceID != options.DeviceID {
		report.addFinding("device_id_mismatch", "Gateway device id differs from expected StackChan device")
	}
	if device.IdentityStatus != "ok" {
		report.addFinding("device_identity_not_ok", "Gateway device identity is not ok")
	}
	if device.ConnectionStatus != "" && device.ConnectionStatus != "online" {
		report.addFinding("device_not_online", "Gateway device is not online")
	}
	if device.Firmware.ID != "a21-stackchan" {
		report.addFinding("firmware_id_mismatch", "device firmware id is not a21-stackchan")
	}
	if device.Firmware.Board != "m5stack-cores3" {
		report.addFinding("firmware_board_mismatch", "device firmware board is not m5stack-cores3")
	}
	if device.Firmware.Commit == "" || !sameCLICommit(device.Firmware.Commit, options.Commit) {
		report.addFinding("firmware_commit_mismatch", "device firmware commit differs from expected commit")
	}
	if report.Microphone != firmwarecheck.MicProbeCapabilityStatus {
		report.addFinding("microphone_not_diagnostic_probe", "device microphone capability is not the diagnostic mic-probe status")
	}
	report.MicFramesCaptured = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_frames_captured")
	report.AudioWSSentAudioFrames = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "audio_ws_sent_audio_frames")
	report.MicDriverErrors = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_driver_errors")
	report.MicQueueDepth = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_queue_depth")
	report.MicQueueDroppedFrames = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_queue_dropped_frames")
	report.MicLastAbsPeak = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_last_abs_peak")
	report.MicLastNonzeroSamples = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_last_nonzero_samples")

	if report.MicFramesCaptured < options.MinFrames {
		report.addFinding("mic_frames_below_threshold", "captured microphone frame count is below threshold")
	}
	if report.AudioWSSentAudioFrames < options.MinFrames {
		report.addFinding("audio_ws_frames_below_threshold", "sent audio WebSocket frame count is below threshold")
	}
	if report.MicDriverErrors != 0 {
		report.addFinding("mic_driver_errors_present", "microphone driver reported errors")
	}
	if report.MicQueueDroppedFrames != 0 {
		report.addFinding("mic_queue_drops_present", "microphone queue dropped frames")
	}
	if report.MicLastAbsPeak < options.MinAbsPeak {
		report.addFinding("mic_peak_below_threshold", "latest microphone peak is below threshold")
	}
	if report.MicLastNonzeroSamples < options.MinNonzeroSamples {
		report.addFinding("mic_nonzero_samples_below_threshold", "latest microphone frame has too few non-zero samples")
	}
}

func stackChanRuntimeEchoInt(findings *[]officePreflightFinding, runtimeEcho map[string]string, key string) int {
	value := strings.TrimSpace(runtimeEcho[key])
	if value == "" {
		*findings = append(*findings, officePreflightFinding{Code: "runtime_echo_missing", Message: "runtime echo missing " + key})
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		*findings = append(*findings, officePreflightFinding{Code: "runtime_echo_invalid", Message: "runtime echo value is not an integer for " + key})
		return 0
	}
	return parsed
}

func fetchStackChanMicProbeGatewayMetrics(gatewayBaseURL string) (stackChanMicProbeGatewayMetrics, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/metrics", nil)
	if err != nil {
		return stackChanMicProbeGatewayMetrics{}, err
	}
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return stackChanMicProbeGatewayMetrics{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return stackChanMicProbeGatewayMetrics{}, fmt.Errorf("gateway metrics returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return stackChanMicProbeGatewayMetrics{}, err
	}
	return parseStackChanMicProbeGatewayMetrics(string(data))
}

func parseStackChanMicProbeGatewayMetrics(data string) (stackChanMicProbeGatewayMetrics, error) {
	var metrics stackChanMicProbeGatewayMetrics
	seen := map[string]bool{}
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			continue
		}
		name := fields[0]
		metricName := name
		if labelsAt := strings.Index(metricName, "{"); labelsAt >= 0 {
			metricName = metricName[:labelsAt]
		}
		switch metricName {
		case "a21_audio_frame_total":
			metrics.AudioFrameTotal = int(value)
			seen["a21_audio_frame_total"] = true
		case "a21_audio_ingress_frames_total":
			metrics.AudioIngressFramesTotal = int(value)
			seen["a21_audio_ingress_frames_total"] = true
		case "a21_audio_ingress_rms":
			metrics.AudioIngressRMS = value
			seen["a21_audio_ingress_rms"] = true
		case "a21_audio_playback_chunk_total":
			metrics.AudioPlaybackChunkTotal = int(value)
			seen["a21_audio_playback_chunk_total"] = true
		case "a21_vad_detector_decisions_total":
			if strings.Contains(name, `result="speech"`) {
				metrics.VADSpeechTotal += int(value)
				seen["a21_vad_detector_decisions_total_speech"] = true
			}
		}
	}
	for _, required := range []string{
		"a21_audio_frame_total",
		"a21_audio_ingress_frames_total",
		"a21_audio_ingress_rms",
		"a21_audio_playback_chunk_total",
	} {
		if !seen[required] {
			return stackChanMicProbeGatewayMetrics{}, fmt.Errorf("gateway metrics missing %s", required)
		}
	}
	return metrics, nil
}

func validateStackChanMicProbeMetrics(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions) {
	if report.GatewayAudioFrameTotal < options.MinFrames {
		report.addFinding("gateway_audio_frames_below_threshold", "Gateway audio frame metric is below threshold")
	}
	if report.GatewayAudioIngressFramesTotal < options.MinFrames {
		report.addFinding("gateway_audio_ingress_frames_below_threshold", "Gateway audio ingress metric is below threshold")
	}
	if report.GatewayAudioIngressRMS < options.MinGatewayRMS {
		report.addFinding("gateway_audio_rms_below_threshold", "Gateway audio ingress RMS is below threshold")
	}
	if report.GatewayAudioPlaybackChunkTotal != 0 {
		report.addFinding("gateway_playback_chunks_present", "Gateway emitted playback chunks during audio-probe-only acceptance")
	}
	if report.GatewayVADSpeechTotal < options.MinGatewayVADSpeech {
		report.addFinding("gateway_vad_speech_below_threshold", "Gateway VAD speech decisions are below threshold")
	}
}

func validateStackChanMicProbeRuntimeDeltas(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions) {
	if report.MicFramesCapturedDelta < options.MinFrames {
		report.addFinding("mic_frames_delta_below_threshold", "captured microphone frame delta is below threshold")
	}
	if report.AudioWSSentAudioFramesDelta < options.MinFrames {
		report.addFinding("audio_ws_frames_delta_below_threshold", "sent audio WebSocket frame delta is below threshold")
	}
	if report.MicDriverErrorsDelta != 0 {
		report.addFinding("mic_driver_error_delta_present", "microphone driver error counter changed during probe window")
	}
	if report.MicQueueDroppedFramesDelta != 0 {
		report.addFinding("mic_queue_drop_delta_present", "microphone queue drop counter changed during probe window")
	}
}

func validateStackChanMicProbeMetricDeltas(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions) {
	if report.GatewayAudioFrameDelta < options.MinFrames {
		report.addFinding("gateway_audio_frame_delta_below_threshold", "Gateway audio frame delta is below threshold")
	}
	if report.GatewayAudioIngressFramesDelta < options.MinFrames {
		report.addFinding("gateway_audio_ingress_delta_below_threshold", "Gateway audio ingress delta is below threshold")
	}
	if report.GatewayAudioIngressRMS < options.MinGatewayRMS {
		report.addFinding("gateway_audio_rms_below_threshold", "Gateway audio ingress RMS is below threshold")
	}
	if report.GatewayAudioPlaybackChunkDelta != 0 {
		report.addFinding("gateway_playback_chunk_delta_present", "Gateway playback chunk counter changed during audio-probe-only window")
	}
	if report.GatewayVADSpeechDelta < options.MinGatewayVADSpeech {
		report.addFinding("gateway_vad_speech_delta_below_threshold", "Gateway VAD speech decision delta is below threshold")
	}
}

func populateStackChanMicProbeWindowQuality(report *stackChanMicProbeAcceptanceReport) {
	elapsedMS := report.WindowEndedAtMS - report.WindowStartedAtMS
	if elapsedMS > 0 {
		report.MicCaptureRateHz = roundedStackChanDiagnosticValue(float64(report.MicFramesCapturedDelta) * 1000 / float64(elapsedMS))
		report.AudioWSSentRateHz = roundedStackChanDiagnosticValue(float64(report.AudioWSSentAudioFramesDelta) * 1000 / float64(elapsedMS))
		report.GatewayAudioIngressRateHz = roundedStackChanDiagnosticValue(float64(report.GatewayAudioIngressFramesDelta) * 1000 / float64(elapsedMS))
	}
	report.AudioWSDeliveryRatio = roundedStackChanDiagnosticRatio(report.AudioWSSentAudioFramesDelta, report.MicFramesCapturedDelta)
	report.GatewayIngressDeliveryRatio = roundedStackChanDiagnosticRatio(report.GatewayAudioIngressFramesDelta, report.AudioWSSentAudioFramesDelta)
}

func validateStackChanMicProbeDeliveryRatios(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions) {
	if report.MicFramesCapturedDelta > 0 && report.AudioWSDeliveryRatio < options.MinDeliveryRatio {
		report.addFinding("audio_ws_delivery_ratio_below_threshold", "audio WebSocket sent frame ratio is below captured microphone frames")
	}
	if report.AudioWSSentAudioFramesDelta > 0 && report.GatewayIngressDeliveryRatio < options.MinDeliveryRatio {
		report.addFinding("gateway_ingress_delivery_ratio_below_threshold", "Gateway ingress frame ratio is below audio WebSocket sent frames")
	}
}

func roundedStackChanDiagnosticRatio(numerator int, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return roundedStackChanDiagnosticValue(float64(numerator) / float64(denominator))
}

func roundedStackChanDiagnosticValue(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func (report *stackChanMicProbeAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}

func runStackChanIMUProbeAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanIMUProbeAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-imu-probe-acceptance --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> [--window-ms 1500] [--min-samples 10] [--min-accel-total-mg 500] [--max-read-errors 0] [--output-dir reports]")
			return 0
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
		case "--window-ms":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--window-ms")
			if !ok {
				return 2
			}
			if value > 120000 {
				fmt.Fprintln(stderr, "--window-ms must be at most 120000")
				return 2
			}
			options.WindowMS = value
		case "--min-samples":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-samples")
			if !ok {
				return 2
			}
			options.MinSamples = value
		case "--min-accel-total-mg":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-accel-total-mg")
			if !ok {
				return 2
			}
			options.MinAccelTotalMG = value
		case "--max-read-errors":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--max-read-errors")
			if !ok {
				return 2
			}
			options.MaxReadErrors = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-imu-probe-acceptance option %q\n", args[i])
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
	if _, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/devices", nil); err != nil {
		fmt.Fprintf(stderr, "stackchan IMU probe acceptance gateway URL invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan IMU probe acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanIMUProbeAcceptanceReport(options)
	reportPath, err := writeStackChanIMUProbeAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan IMU probe acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanIMUProbeAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan IMU probe acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan IMU probe acceptance blocked (diagnostic only, no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan IMU probe acceptance ok (diagnostic only, no flash performed)")
	return 0
}

func defaultStackChanIMUProbeAcceptanceOptions() stackChanIMUProbeAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanIMUProbeAcceptanceOptions{
		GatewayURL:      firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:        deviceID,
		Commit:          currentGitCommit(projectRoot),
		WindowMS:        1500,
		MinSamples:      10,
		MinAccelTotalMG: 500,
		MaxReadErrors:   0,
		OutputDir:       "reports",
	}
}

func buildStackChanIMUProbeAcceptanceReport(options stackChanIMUProbeAcceptanceOptions) stackChanIMUProbeAcceptanceReport {
	report := stackChanIMUProbeAcceptanceReport{
		SchemaVersion:                "a21.stackchan_imu_probe_acceptance.v1",
		GeneratedAtMS:                time.Now().UnixMilli(),
		Metadata:                     buildLatencyBenchMetadata(),
		DryRun:                       true,
		FlashAllowed:                 false,
		DeleteAllowed:                false,
		HardwareAcceptanceScope:      "diagnostic_imu_only",
		IMUProbeAcceptanceStatus:     "confirmed",
		ProductionCapabilityPromoted: false,
		GatewayURL:                   sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                     options.DeviceID,
		Commit:                       options.Commit,
		WindowMS:                     options.WindowMS,
		Thresholds: stackChanIMUProbeAcceptanceThresholds{
			MinSamples:      options.MinSamples,
			MinAccelTotalMG: options.MinAccelTotalMG,
			MaxReadErrors:   options.MaxReadErrors,
		},
		NextRequiredActions: []string{
			"Keep this report with the IMU-probe flash receipt and Gateway device report.",
			"Treat IMU telemetry as read-only diagnostic evidence, not product gesture or posture acceptance.",
			"Promote IMU behavior only through a later ADR and release-firmware acceptance gate.",
		},
	}

	beforeGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_before_failed", err.Error())
		report.IMUProbeAcceptanceStatus = "blocked"
		return report
	}
	beforeDevice, ok := findFirmwareDeviceRecord(beforeGatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_before_missing", "expected StackChan device is missing from Gateway before IMU probe")
		report.IMUProbeAcceptanceStatus = "blocked"
		return report
	}
	validateStackChanIMUProbeIdentity(&report, options, beforeDevice)
	report.RuntimeEchoBefore = beforeDevice.RuntimeEcho
	beforeSamples := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "imu_samples")
	beforeReadErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "imu_read_errors")
	if len(report.Findings) > 0 {
		report.IMUProbeAcceptanceStatus = "blocked"
		return report
	}

	report.WindowStartedAtMS = time.Now().UnixMilli()
	if options.WindowMS > 0 {
		time.Sleep(time.Duration(options.WindowMS) * time.Millisecond)
	}
	report.WindowEndedAtMS = time.Now().UnixMilli()

	afterGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_after_failed", err.Error())
	} else {
		report.GatewayURL = afterGatewayReport.GatewayURL
		afterDevice, ok := findFirmwareDeviceRecord(afterGatewayReport.Devices, options.DeviceID)
		if !ok {
			report.addFinding("gateway_device_after_missing", "expected StackChan device is missing from Gateway after IMU probe")
		} else {
			validateStackChanIMUProbeDevice(&report, options, afterDevice)
			report.IMUSamplesDelta = report.IMUSamples - beforeSamples
			report.IMUReadErrorsDelta = report.IMUReadErrors - beforeReadErrors
			if report.WindowEndedAtMS > report.WindowStartedAtMS {
				report.SampleRateHz = roundedStackChanDiagnosticValue(float64(report.IMUSamplesDelta) * 1000 / float64(report.WindowEndedAtMS-report.WindowStartedAtMS))
			}
			validateStackChanIMUProbeDeltas(&report, options)
		}
	}
	if len(report.Findings) > 0 {
		report.IMUProbeAcceptanceStatus = "blocked"
	}
	return report
}

func validateStackChanIMUProbeIdentity(report *stackChanIMUProbeAcceptanceReport, options stackChanIMUProbeAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	report.Firmware = device.Firmware
	report.Capabilities = device.Capabilities
	report.IMU = device.Capabilities["imu"]
	if device.DeviceID != options.DeviceID {
		report.addFinding("device_id_mismatch", "Gateway device id differs from expected StackChan device")
	}
	if device.IdentityStatus != "ok" {
		report.addFinding("device_identity_not_ok", "Gateway device identity is not ok")
	}
	if device.ConnectionStatus != "" && device.ConnectionStatus != "online" {
		report.addFinding("device_not_online", "Gateway device is not online")
	}
	if device.Firmware.ID != "a21-stackchan" {
		report.addFinding("firmware_id_mismatch", "device firmware id is not a21-stackchan")
	}
	if device.Firmware.Board != "m5stack-cores3" {
		report.addFinding("firmware_board_mismatch", "device firmware board is not m5stack-cores3")
	}
	if device.Firmware.Commit == "" || !sameCLICommit(device.Firmware.Commit, options.Commit) {
		report.addFinding("firmware_commit_mismatch", "device firmware commit differs from expected commit")
	}
	if report.IMU != firmwarecheck.IMUProbeCapabilityStatus {
		report.addFinding("imu_not_diagnostic_probe", "device IMU capability is not the diagnostic IMU-probe status")
	}
}

func validateStackChanIMUProbeDevice(report *stackChanIMUProbeAcceptanceReport, options stackChanIMUProbeAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	validateStackChanIMUProbeIdentity(report, options, device)
	report.RuntimeEcho = device.RuntimeEcho
	report.IMUAvailable = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_available")
	report.IMUSamples = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_samples")
	report.IMUReadErrors = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_read_errors")
	report.IMUAccelMGX = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_accel_mg_x")
	report.IMUAccelMGY = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_accel_mg_y")
	report.IMUAccelMGZ = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_accel_mg_z")
	report.IMUGyroMDPSX = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_gyro_mdps_x")
	report.IMUGyroMDPSY = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_gyro_mdps_y")
	report.IMUGyroMDPSZ = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "imu_gyro_mdps_z")
	report.IMUPosture = strings.TrimSpace(device.RuntimeEcho["imu_posture"])
	report.IMUAccelTotalMG = stackChanAbsInt(report.IMUAccelMGX) + stackChanAbsInt(report.IMUAccelMGY) + stackChanAbsInt(report.IMUAccelMGZ)
	if report.IMUAvailable != 1 {
		report.addFinding("imu_not_available", "IMU diagnostic runtime did not report availability")
	}
	if report.IMUReadErrors > options.MaxReadErrors {
		report.addFinding("imu_read_errors_above_threshold", "IMU read errors exceed threshold")
	}
	if report.IMUAccelTotalMG < options.MinAccelTotalMG {
		report.addFinding("imu_accel_total_below_threshold", "IMU acceleration evidence is below threshold")
	}
	if report.IMUPosture == "" || report.IMUPosture == "unknown" {
		report.addFinding("imu_posture_unknown", "IMU posture is missing or unknown")
	}
}

func validateStackChanIMUProbeDeltas(report *stackChanIMUProbeAcceptanceReport, options stackChanIMUProbeAcceptanceOptions) {
	if report.IMUSamplesDelta < options.MinSamples {
		report.addFinding("imu_samples_delta_below_threshold", "IMU sample delta is below threshold")
	}
	if report.IMUReadErrorsDelta > options.MaxReadErrors {
		report.addFinding("imu_read_error_delta_above_threshold", "IMU read error delta exceeds threshold")
	}
}

func stackChanAbsInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (report *stackChanIMUProbeAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}

func runStackChanSensorProbeAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanSensorProbeAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-sensor-probe-acceptance --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> [--window-ms 1500] [--min-samples 10] [--min-battery-mv 3000] [--max-read-errors 0] [--output-dir reports]")
			return 0
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
		case "--window-ms":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--window-ms")
			if !ok {
				return 2
			}
			if value > 120000 {
				fmt.Fprintln(stderr, "--window-ms must be at most 120000")
				return 2
			}
			options.WindowMS = value
		case "--min-samples":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-samples")
			if !ok {
				return 2
			}
			options.MinSamples = value
		case "--min-battery-mv":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-battery-mv")
			if !ok {
				return 2
			}
			options.MinBatteryMV = value
		case "--max-read-errors":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--max-read-errors")
			if !ok {
				return 2
			}
			options.MaxReadErrors = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-sensor-probe-acceptance option %q\n", args[i])
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
	if _, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/devices", nil); err != nil {
		fmt.Fprintf(stderr, "stackchan sensor probe acceptance gateway URL invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan sensor probe acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanSensorProbeAcceptanceReport(options)
	reportPath, err := writeStackChanSensorProbeAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan sensor probe acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanSensorProbeAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan sensor probe acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan sensor probe acceptance blocked (diagnostic only, no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan sensor probe acceptance ok (diagnostic only, no flash performed)")
	return 0
}

func defaultStackChanSensorProbeAcceptanceOptions() stackChanSensorProbeAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanSensorProbeAcceptanceOptions{
		GatewayURL:    firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:      deviceID,
		Commit:        currentGitCommit(projectRoot),
		WindowMS:      1500,
		MinSamples:    10,
		MinBatteryMV:  3000,
		MaxReadErrors: 0,
		OutputDir:     "reports",
	}
}

func buildStackChanSensorProbeAcceptanceReport(options stackChanSensorProbeAcceptanceOptions) stackChanSensorProbeAcceptanceReport {
	report := stackChanSensorProbeAcceptanceReport{
		SchemaVersion:                "a21.stackchan_sensor_probe_acceptance.v1",
		GeneratedAtMS:                time.Now().UnixMilli(),
		Metadata:                     buildLatencyBenchMetadata(),
		DryRun:                       true,
		FlashAllowed:                 false,
		DeleteAllowed:                false,
		HardwareAcceptanceScope:      "diagnostic_sensor_only",
		SensorProbeAcceptanceStatus:  "confirmed",
		ProductionCapabilityPromoted: false,
		GatewayURL:                   sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                     options.DeviceID,
		Commit:                       options.Commit,
		WindowMS:                     options.WindowMS,
		Thresholds: stackChanSensorProbeAcceptanceThresholds{
			MinSamples:    options.MinSamples,
			MinBatteryMV:  options.MinBatteryMV,
			MaxReadErrors: options.MaxReadErrors,
		},
		NextRequiredActions: []string{
			"Keep this report with the sensor-probe flash receipt and Gateway device report.",
			"Treat ambient/proximity/battery telemetry as read-only diagnostic evidence, not product behavior acceptance.",
			"Promote adaptive brightness, presence behavior, or power-state UI only through a later ADR and release-firmware acceptance gate.",
		},
	}

	beforeGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_before_failed", err.Error())
		report.SensorProbeAcceptanceStatus = "blocked"
		return report
	}
	beforeDevice, ok := findFirmwareDeviceRecord(beforeGatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_before_missing", "expected StackChan device is missing from Gateway before sensor probe")
		report.SensorProbeAcceptanceStatus = "blocked"
		return report
	}
	validateStackChanSensorProbeIdentity(&report, options, beforeDevice)
	report.RuntimeEchoBefore = beforeDevice.RuntimeEcho
	beforeSamples := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "sensor_samples")
	beforeReadErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "sensor_read_errors")
	if len(report.Findings) > 0 {
		report.SensorProbeAcceptanceStatus = "blocked"
		return report
	}

	report.WindowStartedAtMS = time.Now().UnixMilli()
	if options.WindowMS > 0 {
		time.Sleep(time.Duration(options.WindowMS) * time.Millisecond)
	}
	report.WindowEndedAtMS = time.Now().UnixMilli()

	afterGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_after_failed", err.Error())
	} else {
		report.GatewayURL = afterGatewayReport.GatewayURL
		afterDevice, ok := findFirmwareDeviceRecord(afterGatewayReport.Devices, options.DeviceID)
		if !ok {
			report.addFinding("gateway_device_after_missing", "expected StackChan device is missing from Gateway after sensor probe")
		} else {
			validateStackChanSensorProbeDevice(&report, options, afterDevice)
			report.SensorSamplesDelta = report.SensorSamples - beforeSamples
			report.SensorReadErrorsDelta = report.SensorReadErrors - beforeReadErrors
			if report.WindowEndedAtMS > report.WindowStartedAtMS {
				report.SampleRateHz = roundedStackChanDiagnosticValue(float64(report.SensorSamplesDelta) * 1000 / float64(report.WindowEndedAtMS-report.WindowStartedAtMS))
			}
			validateStackChanSensorProbeDeltas(&report, options)
		}
	}
	if len(report.Findings) > 0 {
		report.SensorProbeAcceptanceStatus = "blocked"
	}
	return report
}

func validateStackChanSensorProbeIdentity(report *stackChanSensorProbeAcceptanceReport, options stackChanSensorProbeAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	report.Firmware = device.Firmware
	report.Capabilities = device.Capabilities
	report.AmbientLight = device.Capabilities["ambient_light"]
	report.Proximity = device.Capabilities["proximity"]
	report.Battery = device.Capabilities["battery"]
	if device.DeviceID != options.DeviceID {
		report.addFinding("device_id_mismatch", "Gateway device id differs from expected StackChan device")
	}
	if device.IdentityStatus != "ok" {
		report.addFinding("device_identity_not_ok", "Gateway device identity is not ok")
	}
	if device.ConnectionStatus != "" && device.ConnectionStatus != "online" {
		report.addFinding("device_not_online", "Gateway device is not online")
	}
	if device.Firmware.ID != "a21-stackchan" {
		report.addFinding("firmware_id_mismatch", "device firmware id is not a21-stackchan")
	}
	if device.Firmware.Board != "m5stack-cores3" {
		report.addFinding("firmware_board_mismatch", "device firmware board is not m5stack-cores3")
	}
	if device.Firmware.Commit == "" || !sameCLICommit(device.Firmware.Commit, options.Commit) {
		report.addFinding("firmware_commit_mismatch", "device firmware commit differs from expected commit")
	}
	if report.AmbientLight != firmwarecheck.SensorProbeAmbientLightCapabilityStatus {
		report.addFinding("ambient_light_not_diagnostic_probe", "device ambient light capability is not the diagnostic sensor-probe status")
	}
	if report.Proximity != firmwarecheck.SensorProbeProximityCapabilityStatus {
		report.addFinding("proximity_not_diagnostic_probe", "device proximity capability is not the diagnostic sensor-probe status")
	}
	if report.Battery != firmwarecheck.SensorProbeBatteryCapabilityStatus {
		report.addFinding("battery_not_diagnostic_probe", "device battery capability is not the diagnostic sensor-probe status")
	}
}

func validateStackChanSensorProbeDevice(report *stackChanSensorProbeAcceptanceReport, options stackChanSensorProbeAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	validateStackChanSensorProbeIdentity(report, options, device)
	report.RuntimeEcho = device.RuntimeEcho
	report.SensorAvailable = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "sensor_available")
	report.SensorSamples = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "sensor_samples")
	report.SensorReadErrors = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "sensor_read_errors")
	report.AmbientLightRaw = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "ambient_light_raw")
	report.ProximityRaw = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "proximity_raw")
	report.BatteryMV = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "battery_mv")
	report.BatteryMA = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "battery_ma")
	if report.SensorAvailable != 1 {
		report.addFinding("sensor_not_available", "sensor diagnostic runtime did not report availability")
	}
	if report.SensorReadErrors > options.MaxReadErrors {
		report.addFinding("sensor_read_errors_above_threshold", "sensor read errors exceed threshold")
	}
	if report.BatteryMV < options.MinBatteryMV {
		report.addFinding("battery_mv_below_threshold", "battery voltage evidence is below threshold")
	}
}

func validateStackChanSensorProbeDeltas(report *stackChanSensorProbeAcceptanceReport, options stackChanSensorProbeAcceptanceOptions) {
	if report.SensorSamplesDelta < options.MinSamples {
		report.addFinding("sensor_samples_delta_below_threshold", "sensor sample delta is below threshold")
	}
	if report.SensorReadErrorsDelta > options.MaxReadErrors {
		report.addFinding("sensor_read_error_delta_above_threshold", "sensor read error delta exceeds threshold")
	}
}

func (report *stackChanSensorProbeAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}

func runStackChanHalfDuplexAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanHalfDuplexAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-half-duplex-acceptance --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> [--window-ms 1500] [--min-mic-frames 1] [--min-playback-chunks 1] [--min-delivery-ratio 0.95] [--output-dir reports]")
			return 0
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
		case "--window-ms":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--window-ms")
			if !ok {
				return 2
			}
			if value > 120000 {
				fmt.Fprintln(stderr, "--window-ms must be at most 120000")
				return 2
			}
			options.WindowMS = value
		case "--min-mic-frames":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-mic-frames")
			if !ok {
				return 2
			}
			options.MinMicFrames = value
		case "--min-playback-chunks":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-playback-chunks")
			if !ok {
				return 2
			}
			options.MinPlaybackChunks = value
		case "--min-delivery-ratio":
			value, ok := parseRatioCLIOption(args, &i, stderr, "--min-delivery-ratio")
			if !ok {
				return 2
			}
			options.MinDeliveryRatio = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-half-duplex-acceptance option %q\n", args[i])
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
	if _, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/devices", nil); err != nil {
		fmt.Fprintf(stderr, "stackchan half-duplex acceptance gateway URL invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan half-duplex acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanHalfDuplexAcceptanceReport(options)
	reportPath, err := writeStackChanHalfDuplexAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan half-duplex acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanHalfDuplexAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan half-duplex acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan half-duplex acceptance blocked (instrumented only, no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan half-duplex acceptance ok (instrumented only, no flash performed)")
	return 0
}

func defaultStackChanHalfDuplexAcceptanceOptions() stackChanHalfDuplexAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanHalfDuplexAcceptanceOptions{
		GatewayURL:        firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:          deviceID,
		Commit:            currentGitCommit(projectRoot),
		WindowMS:          1500,
		MinMicFrames:      1,
		MinPlaybackChunks: 1,
		MinDeliveryRatio:  0.95,
		OutputDir:         "reports",
	}
}

func buildStackChanHalfDuplexAcceptanceReport(options stackChanHalfDuplexAcceptanceOptions) stackChanHalfDuplexAcceptanceReport {
	report := stackChanHalfDuplexAcceptanceReport{
		SchemaVersion:              "a21.stackchan_half_duplex_acceptance.v1",
		GeneratedAtMS:              time.Now().UnixMilli(),
		Metadata:                   buildLatencyBenchMetadata(),
		DryRun:                     true,
		FlashAllowed:               false,
		DeleteAllowed:              false,
		HardwareAcceptanceScope:    "mic_to_mock_playback",
		HalfDuplexAcceptanceStatus: "confirmed",
		PhysicalSoundObserved:      false,
		GatewayURL:                 sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                   options.DeviceID,
		Commit:                     options.Commit,
		WindowMS:                   options.WindowMS,
		Thresholds: stackChanHalfDuplexAcceptanceThresholds{
			MinMicFrames:      options.MinMicFrames,
			MinPlaybackChunks: options.MinPlaybackChunks,
			MinDeliveryRatio:  options.MinDeliveryRatio,
		},
		NextRequiredActions: []string{
			"Keep this report with the mic-probe flash receipt and Gateway device report.",
			"Treat this as an instrumented half-duplex mock loop, not real ASR/LLM/TTS or full-duplex acceptance.",
			"Run a later operator-observed acceptance before claiming human-audible conversational quality.",
		},
	}

	beforeGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_before_failed", err.Error())
		report.HalfDuplexAcceptanceStatus = "blocked"
		return report
	}
	beforeDevice, ok := findFirmwareDeviceRecord(beforeGatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_before_missing", "expected StackChan device is missing from Gateway before half-duplex probe")
		report.HalfDuplexAcceptanceStatus = "blocked"
		return report
	}
	validateStackChanHalfDuplexDeviceIdentity(&report, options, beforeDevice)
	report.RuntimeEchoBefore = beforeDevice.RuntimeEcho
	beforeMicFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_frames_captured")
	beforeAudioWSFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "audio_ws_sent_audio_frames")
	beforeMicErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_driver_errors")
	beforeMicDrops := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_queue_dropped_frames")
	beforePlaybackTotal := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "playback_buffer_total_chunks")
	beforePlaybackDrops := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "playback_buffer_dropped_chunks")
	beforeSpeakerFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "speaker_frames_played")
	beforeSpeakerErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "speaker_driver_errors")
	if len(report.Findings) > 0 {
		report.HalfDuplexAcceptanceStatus = "blocked"
		return report
	}

	beforeMetrics, err := fetchStackChanMicProbeGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_before_failed", err.Error())
		report.HalfDuplexAcceptanceStatus = "blocked"
		return report
	}
	report.GatewayMetricsBefore = &beforeMetrics

	traceID := fmt.Sprintf("a21-trace-half-duplex-%d", report.GeneratedAtMS)
	sessionID := fmt.Sprintf("a21-session-half-duplex-%d", report.GeneratedAtMS)
	control, err := postStackChanMicProbeControl(options.GatewayURL, options.DeviceID, protocol.ExpressionListening, protocol.ModeWorkmate, "HALF DUPLEX", traceID, sessionID, false, true)
	if err != nil {
		report.addFinding("half_duplex_control_failed", err.Error())
		report.HalfDuplexAcceptanceStatus = "blocked"
		return report
	}
	report.ControlTraceID = control.TraceID
	report.ControlSessionID = control.SessionID
	report.WindowStartedAtMS = time.Now().UnixMilli()
	time.Sleep(time.Duration(options.WindowMS) * time.Millisecond)
	report.WindowEndedAtMS = time.Now().UnixMilli()

	afterGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_after_failed", err.Error())
	} else {
		report.GatewayURL = afterGatewayReport.GatewayURL
		afterDevice, ok := findFirmwareDeviceRecord(afterGatewayReport.Devices, options.DeviceID)
		if !ok {
			report.addFinding("gateway_device_after_missing", "expected StackChan device is missing from Gateway after half-duplex probe")
		} else {
			validateStackChanHalfDuplexDeviceIdentity(&report, options, afterDevice)
			report.RuntimeEcho = afterDevice.RuntimeEcho
			report.MicFramesCapturedDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "mic_frames_captured") - beforeMicFrames
			report.AudioWSSentAudioFramesDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "audio_ws_sent_audio_frames") - beforeAudioWSFrames
			report.MicDriverErrorsDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "mic_driver_errors") - beforeMicErrors
			report.MicQueueDroppedFramesDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "mic_queue_dropped_frames") - beforeMicDrops
			report.PlaybackBufferTotalChunksDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "playback_buffer_total_chunks") - beforePlaybackTotal
			report.PlaybackBufferDroppedChunksDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "playback_buffer_dropped_chunks") - beforePlaybackDrops
			report.SpeakerFramesPlayedDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "speaker_frames_played") - beforeSpeakerFrames
			report.SpeakerDriverErrorsDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "speaker_driver_errors") - beforeSpeakerErrors
		}
	}

	afterMetrics, err := fetchStackChanMicProbeGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_after_failed", err.Error())
	} else {
		report.GatewayMetricsAfter = &afterMetrics
		report.GatewayAudioFrameDelta = afterMetrics.AudioFrameTotal - beforeMetrics.AudioFrameTotal
		report.GatewayAudioIngressFramesDelta = afterMetrics.AudioIngressFramesTotal - beforeMetrics.AudioIngressFramesTotal
		report.GatewayAudioPlaybackChunkDelta = afterMetrics.AudioPlaybackChunkTotal - beforeMetrics.AudioPlaybackChunkTotal
		report.GatewayVADSpeechDelta = afterMetrics.VADSpeechTotal - beforeMetrics.VADSpeechTotal
	}
	report.AudioWSDeliveryRatio = roundedStackChanDiagnosticRatio(report.AudioWSSentAudioFramesDelta, report.MicFramesCapturedDelta)
	report.GatewayIngressDeliveryRatio = roundedStackChanDiagnosticRatio(report.GatewayAudioIngressFramesDelta, report.AudioWSSentAudioFramesDelta)
	validateStackChanHalfDuplexDeltas(&report, options)

	if _, err := postStackChanMicProbeControl(options.GatewayURL, options.DeviceID, protocol.ExpressionIdle, protocol.ModeWorkmate, "IDLE", report.ControlTraceID, report.ControlSessionID, false, false); err != nil {
		report.addFinding("half_duplex_idle_control_failed", err.Error())
	}
	if len(report.Findings) > 0 {
		report.HalfDuplexAcceptanceStatus = "blocked"
	}
	return report
}

func validateStackChanHalfDuplexDeviceIdentity(report *stackChanHalfDuplexAcceptanceReport, options stackChanHalfDuplexAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	report.Firmware = device.Firmware
	report.Capabilities = device.Capabilities
	report.Microphone = device.Capabilities["microphone"]
	report.Speaker = device.Capabilities["speaker"]
	if device.DeviceID != options.DeviceID {
		report.addFinding("device_id_mismatch", "Gateway device id differs from expected StackChan device")
	}
	if device.IdentityStatus != "ok" {
		report.addFinding("device_identity_not_ok", "Gateway device identity is not ok")
	}
	if device.ConnectionStatus != "" && device.ConnectionStatus != "online" {
		report.addFinding("device_not_online", "Gateway device is not online")
	}
	if device.Firmware.ID != "a21-stackchan" {
		report.addFinding("firmware_id_mismatch", "device firmware id is not a21-stackchan")
	}
	if device.Firmware.Board != "m5stack-cores3" {
		report.addFinding("firmware_board_mismatch", "device firmware board is not m5stack-cores3")
	}
	if device.Firmware.Commit == "" || !sameCLICommit(device.Firmware.Commit, options.Commit) {
		report.addFinding("firmware_commit_mismatch", "device firmware commit differs from expected commit")
	}
	if report.Microphone != firmwarecheck.MicProbeCapabilityStatus {
		report.addFinding("microphone_not_diagnostic_probe", "device microphone capability is not the diagnostic mic-probe status")
	}
	if report.Speaker != "available" {
		report.addFinding("speaker_not_available", "device speaker capability is not available")
	}
}

func validateStackChanHalfDuplexDeltas(report *stackChanHalfDuplexAcceptanceReport, options stackChanHalfDuplexAcceptanceOptions) {
	if report.MicFramesCapturedDelta < options.MinMicFrames {
		report.addFinding("mic_frames_delta_below_threshold", "captured microphone frame delta is below threshold")
	}
	if report.AudioWSSentAudioFramesDelta < options.MinMicFrames {
		report.addFinding("audio_ws_frames_delta_below_threshold", "sent audio WebSocket frame delta is below threshold")
	}
	if report.GatewayAudioIngressFramesDelta < options.MinMicFrames {
		report.addFinding("gateway_audio_ingress_delta_below_threshold", "Gateway audio ingress delta is below threshold")
	}
	if report.GatewayAudioPlaybackChunkDelta < options.MinPlaybackChunks {
		report.addFinding("gateway_playback_chunk_delta_below_threshold", "Gateway playback chunk delta is below threshold")
	}
	if report.PlaybackBufferTotalChunksDelta < options.MinPlaybackChunks {
		report.addFinding("playback_buffer_delta_below_threshold", "playback buffer accepted chunk delta is below threshold")
	}
	if report.SpeakerFramesPlayedDelta < options.MinPlaybackChunks {
		report.addFinding("speaker_frames_delta_below_threshold", "speaker played frame delta is below threshold")
	}
	if report.MicDriverErrorsDelta != 0 {
		report.addFinding("mic_driver_error_delta_present", "microphone driver error counter changed during half-duplex probe")
	}
	if report.MicQueueDroppedFramesDelta != 0 {
		report.addFinding("mic_queue_drop_delta_present", "microphone queue drop counter changed during half-duplex probe")
	}
	if report.PlaybackBufferDroppedChunksDelta != 0 {
		report.addFinding("playback_buffer_drop_delta_present", "playback buffer dropped chunks during half-duplex probe")
	}
	if report.SpeakerDriverErrorsDelta != 0 {
		report.addFinding("speaker_driver_error_delta_present", "speaker driver error counter changed during half-duplex probe")
	}
	if report.MicFramesCapturedDelta > 0 && report.AudioWSDeliveryRatio < options.MinDeliveryRatio {
		report.addFinding("audio_ws_delivery_ratio_below_threshold", "audio WebSocket sent frame ratio is below captured microphone frames")
	}
	if report.AudioWSSentAudioFramesDelta > 0 && report.GatewayIngressDeliveryRatio < options.MinDeliveryRatio {
		report.addFinding("gateway_ingress_delivery_ratio_below_threshold", "Gateway ingress frame ratio is below audio WebSocket sent frames")
	}
}

func (report *stackChanHalfDuplexAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}

func runStackChanSpeakerAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanSpeakerAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-speaker-acceptance --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> [--window-ms 1000] [--mock-audio-chunks 50] [--min-played-frames 50] [--output-dir reports]")
			return 0
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
		case "--window-ms":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--window-ms")
			if !ok {
				return 2
			}
			if value > 120000 {
				fmt.Fprintln(stderr, "--window-ms must be at most 120000")
				return 2
			}
			options.WindowMS = value
		case "--mock-audio-chunks":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--mock-audio-chunks")
			if !ok {
				return 2
			}
			if value < 1 || value > stackChanSpeakerProbeMaxChunks {
				fmt.Fprintf(stderr, "--mock-audio-chunks must be between 1 and %d\n", stackChanSpeakerProbeMaxChunks)
				return 2
			}
			options.MockAudioChunks = value
		case "--min-played-frames":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-played-frames")
			if !ok {
				return 2
			}
			options.MinPlayedFrames = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-speaker-acceptance option %q\n", args[i])
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
	if _, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/devices", nil); err != nil {
		fmt.Fprintf(stderr, "stackchan speaker acceptance gateway URL invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan speaker acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanSpeakerAcceptanceReport(options)
	reportPath, err := writeStackChanSpeakerAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan speaker acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanSpeakerAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan speaker acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan speaker acceptance blocked (instrumented only, no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan speaker acceptance ok (instrumented only, no flash performed)")
	return 0
}

func defaultStackChanSpeakerAcceptanceOptions() stackChanSpeakerAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanSpeakerAcceptanceOptions{
		GatewayURL:      firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:        deviceID,
		Commit:          currentGitCommit(projectRoot),
		WindowMS:        1500,
		MockAudioChunks: 50,
		MinPlayedFrames: 50,
		OutputDir:       "reports",
	}
}

func buildStackChanSpeakerAcceptanceReport(options stackChanSpeakerAcceptanceOptions) stackChanSpeakerAcceptanceReport {
	streamID := "a21-speaker-acceptance-stream"
	report := stackChanSpeakerAcceptanceReport{
		SchemaVersion:           "a21.stackchan_speaker_acceptance.v1",
		GeneratedAtMS:           time.Now().UnixMilli(),
		Metadata:                buildLatencyBenchMetadata(),
		DryRun:                  true,
		FlashAllowed:            false,
		DeleteAllowed:           false,
		HardwareAcceptanceScope: "instrumented_speaker_downlink",
		SpeakerAcceptanceStatus: "confirmed",
		PhysicalSoundObserved:   false,
		GatewayURL:              sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                options.DeviceID,
		Commit:                  options.Commit,
		WindowMS:                options.WindowMS,
		MockAudioChunks:         options.MockAudioChunks,
		ExpectedAudioDurationMS: options.MockAudioChunks * stackChanSpeakerProbeChunkDurationMS,
		StreamID:                streamID,
		Thresholds: stackChanSpeakerAcceptanceThresholds{
			MinPlayedFrames:  options.MinPlayedFrames,
			MinGatewayChunks: options.MockAudioChunks,
		},
		NextRequiredActions: []string{
			"Keep this report with the Gateway device report before any physical speaker promotion.",
			"Treat this as instrumented downlink and speaker-pump evidence, not human-audible sound acceptance.",
			"Run a later operator-observed speaker acceptance before claiming physical audibility or end-to-end TTS quality.",
		},
	}

	beforeGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_before_failed", err.Error())
		report.SpeakerAcceptanceStatus = "blocked"
		return report
	}
	beforeDevice, ok := findFirmwareDeviceRecord(beforeGatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_before_missing", "expected StackChan device is missing from Gateway before speaker probe")
		report.SpeakerAcceptanceStatus = "blocked"
		return report
	}
	validateStackChanSpeakerDeviceIdentity(&report, options, beforeDevice)
	report.RuntimeEchoBefore = beforeDevice.RuntimeEcho
	beforePlaybackTotal := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "playback_buffer_total_chunks")
	beforePlaybackDropped := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "playback_buffer_dropped_chunks")
	beforePlaybackClearCount := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "playback_buffer_clear_count")
	beforeSpeakerFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "speaker_frames_played")
	beforeSpeakerBusy := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "speaker_busy_ticks")
	beforeSpeakerErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "speaker_driver_errors")
	if len(report.Findings) > 0 {
		report.SpeakerAcceptanceStatus = "blocked"
		return report
	}

	beforeMetrics, err := fetchStackChanSpeakerGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_before_failed", err.Error())
		report.SpeakerAcceptanceStatus = "blocked"
		return report
	}
	report.GatewayMetricsBefore = &beforeMetrics

	traceID := fmt.Sprintf("a21-trace-speaker-acceptance-%d", report.GeneratedAtMS)
	sessionID := fmt.Sprintf("a21-session-speaker-acceptance-%d", report.GeneratedAtMS)
	control, err := postStackChanSpeakerProbeBatches(options.GatewayURL, options.DeviceID, traceID, sessionID, streamID, options.MockAudioChunks)
	if err != nil {
		report.addFinding("speaker_control_failed", err.Error())
		report.SpeakerAcceptanceStatus = "blocked"
		_, _ = postStackChanSpeakerControl(options.GatewayURL, options.DeviceID, protocol.ExpressionIdle, protocol.ModeWorkmate, "IDLE", traceID, sessionID, streamID, 0)
		return report
	}
	report.ControlTraceID = control.TraceID
	report.ControlSessionID = control.SessionID
	report.WindowStartedAtMS = time.Now().UnixMilli()
	time.Sleep(time.Duration(options.WindowMS) * time.Millisecond)
	report.WindowEndedAtMS = time.Now().UnixMilli()

	afterGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_after_failed", err.Error())
	} else {
		report.GatewayURL = afterGatewayReport.GatewayURL
		afterDevice, ok := findFirmwareDeviceRecord(afterGatewayReport.Devices, options.DeviceID)
		if !ok {
			report.addFinding("gateway_device_after_missing", "expected StackChan device is missing from Gateway after speaker probe")
		} else {
			validateStackChanSpeakerDevice(&report, options, afterDevice)
			report.PlaybackBufferTotalChunksDelta = report.PlaybackBufferTotalChunks - beforePlaybackTotal
			report.PlaybackBufferDroppedChunksDelta = report.PlaybackBufferDroppedChunks - beforePlaybackDropped
			report.PlaybackBufferClearCountDelta = report.PlaybackBufferClearCount - beforePlaybackClearCount
			report.SpeakerFramesPlayedDelta = report.SpeakerFramesPlayed - beforeSpeakerFrames
			report.SpeakerBusyTicksDelta = report.SpeakerBusyTicks - beforeSpeakerBusy
			report.SpeakerDriverErrorsDelta = report.SpeakerDriverErrors - beforeSpeakerErrors
			validateStackChanSpeakerRuntimeDeltas(&report, options)
		}
	}

	afterMetrics, err := fetchStackChanSpeakerGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_after_failed", err.Error())
	} else {
		report.GatewayMetricsAfter = &afterMetrics
		report.GatewayAudioPlaybackChunkTotal = afterMetrics.AudioPlaybackChunkTotal
		report.GatewayAudioPlaybackChunkDelta = afterMetrics.AudioPlaybackChunkTotal - beforeMetrics.AudioPlaybackChunkTotal
		validateStackChanSpeakerMetricDeltas(&report, options)
	}

	if _, err := postStackChanSpeakerControl(options.GatewayURL, options.DeviceID, protocol.ExpressionIdle, protocol.ModeWorkmate, "IDLE", report.ControlTraceID, report.ControlSessionID, streamID, 0); err != nil {
		report.addFinding("speaker_idle_control_failed", err.Error())
	}
	if len(report.Findings) > 0 {
		report.SpeakerAcceptanceStatus = "blocked"
	}
	return report
}

func validateStackChanSpeakerDevice(report *stackChanSpeakerAcceptanceReport, options stackChanSpeakerAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	validateStackChanSpeakerDeviceIdentity(report, options, device)
	report.RuntimeEcho = device.RuntimeEcho
	report.PlaybackBufferQueuedChunks = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "playback_buffer_queued_chunks")
	report.PlaybackBufferTotalChunks = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "playback_buffer_total_chunks")
	report.PlaybackBufferDroppedChunks = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "playback_buffer_dropped_chunks")
	report.PlaybackBufferClearCount = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "playback_buffer_clear_count")
	report.SpeakerFramesPlayed = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "speaker_frames_played")
	report.SpeakerBusyTicks = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "speaker_busy_ticks")
	report.SpeakerDriverErrors = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "speaker_driver_errors")
	report.SpeakerLastStreamID = stackChanRuntimeEchoString(&report.Findings, device.RuntimeEcho, "speaker_last_stream_id")
	if report.SpeakerLastStreamID != report.StreamID {
		report.addFinding("speaker_stream_id_mismatch", "speaker runtime echo does not match the commanded stream id")
	}
}

func validateStackChanSpeakerDeviceIdentity(report *stackChanSpeakerAcceptanceReport, options stackChanSpeakerAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	report.Firmware = device.Firmware
	report.Capabilities = device.Capabilities
	report.Speaker = device.Capabilities["speaker"]
	if device.DeviceID != options.DeviceID {
		report.addFinding("device_id_mismatch", "Gateway device id differs from expected StackChan device")
	}
	if device.IdentityStatus != "ok" {
		report.addFinding("device_identity_not_ok", "Gateway device identity is not ok")
	}
	if device.ConnectionStatus != "" && device.ConnectionStatus != "online" {
		report.addFinding("device_not_online", "Gateway device is not online")
	}
	if device.Firmware.ID != "a21-stackchan" {
		report.addFinding("firmware_id_mismatch", "device firmware id is not a21-stackchan")
	}
	if device.Firmware.Board != "m5stack-cores3" {
		report.addFinding("firmware_board_mismatch", "device firmware board is not m5stack-cores3")
	}
	if device.Firmware.Commit == "" || !sameCLICommit(device.Firmware.Commit, options.Commit) {
		report.addFinding("firmware_commit_mismatch", "device firmware commit differs from expected commit")
	}
	if report.Speaker != "available" {
		report.addFinding("speaker_not_available", "device speaker capability is not available")
	}
}

func validateStackChanSpeakerRuntimeDeltas(report *stackChanSpeakerAcceptanceReport, options stackChanSpeakerAcceptanceOptions) {
	if report.PlaybackBufferTotalChunksDelta < options.MockAudioChunks {
		report.addFinding("playback_buffer_delta_below_threshold", "playback buffer accepted chunk delta is below commanded mock audio chunks")
	}
	if report.PlaybackBufferDroppedChunksDelta != 0 {
		report.addFinding("playback_buffer_drop_delta_present", "playback buffer dropped chunks during speaker probe")
	}
	if report.SpeakerFramesPlayedDelta < options.MinPlayedFrames {
		report.addFinding("speaker_frames_delta_below_threshold", "speaker played frame delta is below threshold")
	}
	if report.SpeakerDriverErrorsDelta != 0 {
		report.addFinding("speaker_driver_error_delta_present", "speaker driver error counter changed during speaker probe")
	}
}

func validateStackChanSpeakerMetricDeltas(report *stackChanSpeakerAcceptanceReport, options stackChanSpeakerAcceptanceOptions) {
	if report.GatewayAudioPlaybackChunkDelta < options.MockAudioChunks {
		report.addFinding("gateway_playback_chunk_delta_below_threshold", "Gateway playback chunk delta is below commanded mock audio chunks")
	}
}

func stackChanRuntimeEchoString(findings *[]officePreflightFinding, runtimeEcho map[string]string, key string) string {
	value := strings.TrimSpace(runtimeEcho[key])
	if value == "" {
		*findings = append(*findings, officePreflightFinding{Code: "runtime_echo_missing", Message: "runtime echo missing " + key})
	}
	return value
}

func fetchStackChanSpeakerGatewayMetrics(gatewayBaseURL string) (stackChanSpeakerGatewayMetrics, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/metrics", nil)
	if err != nil {
		return stackChanSpeakerGatewayMetrics{}, err
	}
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return stackChanSpeakerGatewayMetrics{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return stackChanSpeakerGatewayMetrics{}, fmt.Errorf("gateway metrics returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return stackChanSpeakerGatewayMetrics{}, err
	}
	return parseStackChanSpeakerGatewayMetrics(string(data))
}

func parseStackChanSpeakerGatewayMetrics(data string) (stackChanSpeakerGatewayMetrics, error) {
	var metrics stackChanSpeakerGatewayMetrics
	seen := false
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		metricName := fields[0]
		if labelsAt := strings.Index(metricName, "{"); labelsAt >= 0 {
			metricName = metricName[:labelsAt]
		}
		if metricName != "a21_audio_playback_chunk_total" {
			continue
		}
		value, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			continue
		}
		metrics.AudioPlaybackChunkTotal = int(value)
		seen = true
	}
	if !seen {
		return stackChanSpeakerGatewayMetrics{}, fmt.Errorf("gateway metrics missing a21_audio_playback_chunk_total")
	}
	return metrics, nil
}

func (report *stackChanSpeakerAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}

func runStackChanTouchAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanTouchAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-touch-acceptance --case screen_touch|top_tap|top_swipe_forward|top_swipe_backward|top_barge_in --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 [--window-ms 15000] [--output-dir reports]")
			return 0
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
		case "--case":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--case requires a value")
				return 2
			}
			i++
			options.Case = args[i]
		case "--window-ms":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--window-ms requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 || value > 120000 {
				fmt.Fprintln(stderr, "--window-ms must be between 1 and 120000")
				return 2
			}
			options.WindowMS = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-touch-acceptance option %q\n", args[i])
			return 2
		}
	}
	if options.Case == "" {
		fmt.Fprintln(stderr, "--case requires a value")
		return 2
	}
	if options.DeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan touch acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := runStackChanTouchAcceptanceCase(options)
	if options.OutputDir != "" {
		reportPath, err := writeStackChanTouchAcceptanceReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write stackchan touch acceptance report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanTouchAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan touch acceptance report: %v\n", err)
		return 1
	}
	if report.AcceptanceStatus != "passed" {
		fmt.Fprintln(stdout, "stackchan touch acceptance blocked (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan touch acceptance ok (no flash performed)")
	return 0
}

func defaultStackChanTouchAcceptanceOptions() stackChanTouchAcceptanceOptions {
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanTouchAcceptanceOptions{
		GatewayURL: "http://127.0.0.1:21080",
		DeviceID:   deviceID,
		WindowMS:   15000,
		OutputDir:  "reports",
	}
}

func runStackChanTouchAcceptanceCase(options stackChanTouchAcceptanceOptions) stackChanTouchAcceptanceReport {
	spec, ok := stackChanTouchCase(options.Case)
	report := stackChanTouchAcceptanceReport{
		SchemaVersion:    "a21.stackchan_touch_acceptance.v1",
		GeneratedAtMS:    time.Now().UnixMilli(),
		DryRun:           true,
		FlashAllowed:     false,
		DeleteAllowed:    false,
		AcceptanceStatus: "blocked",
		GatewayURL:       options.GatewayURL,
		DeviceID:         options.DeviceID,
		Case:             options.Case,
		WindowMS:         options.WindowMS,
	}
	if !ok {
		report.addFinding("unknown_touch_case", "touch acceptance case is not supported")
		return report
	}
	report.AcceptanceScope = spec.AcceptanceScope
	report.CoreUX = spec.CoreUX
	report.NeedsAffordance = spec.NeedsAffordance
	report.ExpectedEvent = spec.ExpectedEvent
	report.ExpectedTraceEvent = spec.ExpectedTrace
	report.ExpectedSource = spec.ExpectedSource
	report.DevicePrompt = spec.DevicePrompt
	report.HumanGuidance = spec.HumanGuidance

	before, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_failed", err.Error())
		return report
	}
	if _, ok := findFirmwareDeviceRecord(before.Devices, options.DeviceID); !ok {
		report.addFinding("device_missing", "expected StackChan device is missing from Gateway")
		return report
	}

	control, err := postStackChanTouchControl(options.GatewayURL, options.DeviceID, spec)
	if err != nil {
		report.addFinding("device_control_failed", err.Error())
		return report
	}
	report.ControlTraceID = control.TraceID
	report.ControlSessionID = control.SessionID

	controlStartedAtMS := time.Now().UnixMilli()
	deadline := time.Now().Add(time.Duration(options.WindowMS) * time.Millisecond)
	seen := map[string]bool{}
	for time.Now().Before(deadline) {
		gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
		if err != nil {
			report.addFinding("gateway_poll_failed", err.Error())
			return report
		}
		device, ok := findFirmwareDeviceRecord(gatewayReport.Devices, options.DeviceID)
		if !ok {
			report.addFinding("device_disconnected", "expected StackChan device disappeared during touch acceptance")
			return report
		}
		for _, traceID := range candidateStackChanTouchTraceIDs(device.LastTraceID) {
			trace, err := fetchGatewayTrace(options.GatewayURL, traceID)
			if err == nil {
				for _, event := range trace.Events {
					if event.AtMS > 0 && event.AtMS+1000 < controlStartedAtMS {
						continue
					}
					if !strings.HasPrefix(event.Name, "device.touch.") {
						continue
					}
					key := event.TraceID + "|" + event.Name + "|" + strconv.FormatInt(event.AtMS, 10)
					if !seen[key] {
						seen[key] = true
						report.ObservedEvents = append(report.ObservedEvents, stackChanTouchAcceptanceObservation{
							Event:     event.Name,
							TraceID:   event.TraceID,
							SessionID: event.SessionID,
							DeviceID:  event.DeviceID,
							AtMS:      event.AtMS,
							OffsetMS:  event.OffsetMS,
							Source:    string(device.LastTouchSource),
						})
					}
					if event.Name == spec.ExpectedTrace && (spec.ExpectedSource == "" || device.LastTouchSource == spec.ExpectedSource) {
						report.AcceptanceStatus = "passed"
						return report
					}
				}
			}
		}
		if string(device.LastEvent) == spec.ExpectedEvent && (spec.ExpectedSource == "" || device.LastTouchSource == spec.ExpectedSource) {
			report.ObservedEvents = append(report.ObservedEvents, stackChanTouchAcceptanceObservation{
				Event:     "device." + string(device.LastEvent) + ".received",
				TraceID:   device.LastTraceID,
				SessionID: device.LastSessionID,
				DeviceID:  device.DeviceID,
				AtMS:      device.LastSeenMS,
				Source:    string(device.LastTouchSource),
			})
			report.AcceptanceStatus = "passed"
			return report
		}
		time.Sleep(250 * time.Millisecond)
	}
	if len(report.ObservedEvents) == 0 {
		report.addFinding("touch_event_not_observed", "no touch event was observed inside the acceptance window")
	} else {
		report.addFinding("unexpected_touch_event", "touch events were observed, but not the expected event/source for this case")
	}
	if spec.NeedsAffordance {
		report.addFinding("physical_affordance_required", "directional top gestures need screen guidance, operator training, or physical labeling before they can be treated as core UX")
	}
	return report
}

func stackChanTouchCase(name string) (stackChanTouchCaseSpec, bool) {
	switch strings.TrimSpace(name) {
	case "screen_touch":
		return stackChanTouchCaseSpec{
			Name:            "screen_touch",
			AcceptanceScope: "core_touch",
			CoreUX:          true,
			ExpectedEvent:   "touch.wake_or_listen",
			ExpectedTrace:   "device.touch.wake_or_listen.received",
			ExpectedSource:  "screen",
			ControlState:    protocol.ExpressionIdle,
			ControlMode:     protocol.ModeWorkmate,
			DevicePrompt:    "Tap screen once",
			HumanGuidance:   "Tap the screen once and release. Do not touch the top strip.",
		}, true
	case "top_tap":
		return stackChanTouchCaseSpec{
			Name:            "top_tap",
			AcceptanceScope: "core_touch",
			CoreUX:          true,
			ExpectedEvent:   "touch.top.tap",
			ExpectedTrace:   "device.touch.top.tap.received",
			ExpectedSource:  "top_sensor",
			ControlState:    protocol.ExpressionIdle,
			ControlMode:     protocol.ModeWorkmate,
			DevicePrompt:    "Top: tap once",
			HumanGuidance:   "Tap the top touch strip once and release. Do not slide.",
		}, true
	case "top_swipe_forward":
		return stackChanTouchCaseSpec{
			Name:            "top_swipe_forward",
			AcceptanceScope: "guided_directional_touch",
			CoreUX:          false,
			NeedsAffordance: true,
			ExpectedEvent:   "touch.top.swipe_forward",
			ExpectedTrace:   "device.touch.top.swipe_forward.received",
			ExpectedSource:  "top_sensor",
			ControlState:    protocol.ExpressionIdle,
			ControlMode:     protocol.ModeWorkmate,
			DevicePrompt:    "Top: FACE -> USB",
			HumanGuidance:   "Slide once along the top strip from the screen/face side toward the USB/back side, then fully release.",
		}, true
	case "top_swipe_backward":
		return stackChanTouchCaseSpec{
			Name:            "top_swipe_backward",
			AcceptanceScope: "guided_directional_touch",
			CoreUX:          false,
			NeedsAffordance: true,
			ExpectedEvent:   "touch.top.swipe_backward",
			ExpectedTrace:   "device.touch.top.swipe_backward.received",
			ExpectedSource:  "top_sensor",
			ControlState:    protocol.ExpressionIdle,
			ControlMode:     protocol.ModeWorkmate,
			DevicePrompt:    "Top: USB -> FACE",
			HumanGuidance:   "Slide once along the top strip from the USB/back side toward the screen/face side, then fully release.",
		}, true
	case "top_barge_in":
		return stackChanTouchCaseSpec{
			Name:            "top_barge_in",
			AcceptanceScope: "core_touch",
			CoreUX:          true,
			ExpectedEvent:   "touch.barge_in",
			ExpectedTrace:   "device.touch.barge_in.received",
			ExpectedSource:  "top_sensor",
			ControlState:    protocol.ExpressionSpeaking,
			ControlMode:     protocol.ModeWorkmate,
			DevicePrompt:    "Top: interrupt",
			HumanGuidance:   "While A21 is in SPEAKING state, tap any part of the top touch strip once and release.",
			StreamID:        "a21-touch-acceptance-stream",
			MockAudioChunks: 2,
		}, true
	default:
		return stackChanTouchCaseSpec{}, false
	}
}

var stackChanHardwareMainlineTracks = []stackChanHardwareTrack{
	{
		Capability:       "imu",
		TargetStatus:     "planned_9_axis_imu",
		Track:            "read_only_diagnostic_probe",
		PromotionAllowed: false,
		NextAction:       "Add posture, bump, pickup, and safe expression transition telemetry before any product promotion.",
	},
	{
		Capability:       "ambient_light",
		TargetStatus:     "planned_ambient_light_sensor",
		Track:            "read_only_diagnostic_probe",
		PromotionAllowed: false,
		NextAction:       "Add adaptive brightness telemetry and acceptance evidence without changing expression behavior first.",
	},
	{
		Capability:       "proximity",
		TargetStatus:     "planned_proximity_sensor",
		Track:            "read_only_diagnostic_probe",
		PromotionAllowed: false,
		NextAction:       "Add wake, sleep, and interaction-distance telemetry with false-positive protection.",
	},
	{
		Capability:       "battery",
		TargetStatus:     "planned_550mah_battery",
		Track:            "read_only_diagnostic_probe",
		PromotionAllowed: false,
		NextAction:       "Add honest power-state reporting and low-power local fallback evidence.",
	},
	{
		Capability:       "servo_x",
		TargetStatus:     "planned_continuous_rotation_axis",
		Track:            "motion_safety_spike",
		PromotionAllowed: false,
		NextAction:       "Prove second-axis range, clamps, and coordinated head pose before enabling motion.",
	},
	{
		Capability:       "camera",
		TargetStatus:     "planned_core_s3_camera",
		Track:            "privacy_safe_vision_spike",
		PromotionAllowed: false,
		NextAction:       "Design explicit opt-in local presence or gesture context with no silent surveillance.",
	},
	{
		Capability:       "nfc",
		TargetStatus:     "planned_nfc",
		Track:            "explicit_opt_in_interaction_spike",
		PromotionAllowed: false,
		NextAction:       "Define visible consent and desk interaction semantics before runtime use.",
	},
	{
		Capability:       "infrared",
		TargetStatus:     "planned_infrared_tx_rx",
		Track:            "explicit_opt_in_interaction_spike",
		PromotionAllowed: false,
		NextAction:       "Define explicit opt-in remote interaction semantics before runtime use.",
	},
}

var stackChanHardwareCapabilityInvariantSpecs = []stackChanHardwareCapabilityInvariantSpec{
	{
		Capability:      "microphone",
		Requirement:     "release microphone remains disabled or diagnostic-only until separate production evidence exists",
		AllowedPrefixes: []string{"disabled_", "diagnostic_"},
	},
	{Capability: "speaker", Requirement: "stable expression/playback surface must be available", AllowedExact: []string{"available"}},
	{Capability: "screen", Requirement: "stable expression/playback surface must be available", AllowedExact: []string{"available"}},
	{Capability: "screen_touch", Requirement: "stable touch surface must be available", AllowedExact: []string{"available"}},
	{Capability: "top_touch", Requirement: "stable touch surface must be available", AllowedExact: []string{"available"}},
	{Capability: "servo_y", Requirement: "stable Y-axis expression surface must be available", AllowedExact: []string{"available"}},
	{Capability: "rgb", Requirement: "stable RGB expression surface must be available", AllowedExact: []string{"available"}},
	{
		Capability:      "imu",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "ambient_light",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "proximity",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "battery",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "servo_x",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "camera",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "nfc",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "infrared",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
}

func runStackChanHardwareMainline(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanHardwareMainlineOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-hardware-mainline --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 [--output-dir reports]")
			return 0
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
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-hardware-mainline option %q\n", args[i])
			return 2
		}
	}
	if options.DeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if containsLegacyIdentity(options.DeviceID) {
		fmt.Fprintln(stderr, "device id contains forbidden legacy identity")
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan hardware mainline report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanHardwareMainlineReport(options)
	if options.OutputDir != "" {
		reportPath, err := writeStackChanHardwareMainlineReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write stackchan hardware mainline report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanHardwareMainline(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan hardware mainline report: %v\n", err)
		return 1
	}
	if report.HardwareMainlineStatus != "ready_for_diagnostic_spikes" {
		fmt.Fprintln(stdout, "stackchan hardware mainline blocked (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan hardware mainline ready (no flash performed)")
	return 0
}

func defaultStackChanHardwareMainlineOptions() stackChanHardwareMainlineOptions {
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanHardwareMainlineOptions{
		GatewayURL: firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:   deviceID,
		OutputDir:  "reports",
	}
}

func buildStackChanHardwareMainlineReport(options stackChanHardwareMainlineOptions) stackChanHardwareMainlineReport {
	report := stackChanHardwareMainlineReport{
		SchemaVersion:          "a21.stackchan_hardware_mainline.v1",
		GeneratedAtMS:          time.Now().UnixMilli(),
		Metadata:               buildLatencyBenchMetadata(),
		DryRun:                 true,
		FlashAllowed:           false,
		DeleteAllowed:          false,
		HardwareMainlineStatus: "blocked",
		GatewayURL:             sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:               options.DeviceID,
	}
	gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_failed", err.Error())
		report.NextRequiredActions = []string{"Start the A21 Gateway, keep LAN traffic direct, and rerun stackchan-hardware-mainline before touching firmware."}
		return report
	}
	device, ok := findFirmwareDeviceRecord(gatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("device_missing", "expected StackChan device is missing from Gateway")
		report.NextRequiredActions = []string{"Connect the intended A21 StackChan device to Gateway and rerun stackchan-hardware-mainline."}
		return report
	}
	report.Firmware = device.Firmware
	report.Capabilities = cloneStringMap(device.Capabilities)
	report.RuntimeEcho = cloneStringMap(device.RuntimeEcho)
	for _, spec := range stackChanHardwareCapabilityInvariantSpecs {
		declaredStatus := strings.TrimSpace(report.Capabilities[spec.Capability])
		invariant := stackChanHardwareCapabilityInvariant{
			Capability:     spec.Capability,
			DeclaredStatus: declaredStatus,
			Requirement:    spec.Requirement,
			Accepted:       stackChanHardwareCapabilityStatusAccepted(declaredStatus, spec),
		}
		if declaredStatus == "" {
			report.addFinding("capability_missing", fmt.Sprintf("StackChan capability %q must be declared before the hardware mainline can proceed", spec.Capability))
		} else if !invariant.Accepted {
			report.addFinding("capability_status_not_honest", fmt.Sprintf("StackChan capability %q declared status %q violates hardware mainline invariant", spec.Capability, declaredStatus))
		}
		report.CapabilityInvariants = append(report.CapabilityInvariants, invariant)
	}
	for _, spec := range stackChanHardwareMainlineTracks {
		track := spec
		declaredStatus, ok := report.Capabilities[track.Capability]
		if !ok || strings.TrimSpace(declaredStatus) == "" {
			report.addFinding("capability_missing", fmt.Sprintf("StackChan capability %q must be declared before this hardware track can proceed", track.Capability))
			track.DeclaredStatus = ""
		} else {
			track.DeclaredStatus = declaredStatus
		}
		report.OrderedTracks = append(report.OrderedTracks, track)
	}
	report.NextRequiredActions = []string{
		"Run hardware tracks in ordered_tracks order and keep unavailable capabilities planned until instrumented evidence exists.",
		"Add firmware diagnostics as separate guarded builds before promoting any planned capability to product firmware.",
		"Preserve A21 package identity, artifact checks, and no raw upload discipline for every hardware slice.",
	}
	if len(report.Findings) == 0 {
		report.HardwareMainlineStatus = "ready_for_diagnostic_spikes"
	}
	return report
}

func stackChanHardwareCapabilityStatusAccepted(status string, spec stackChanHardwareCapabilityInvariantSpec) bool {
	status = strings.TrimSpace(status)
	if status == "" {
		return false
	}
	for _, exact := range spec.AllowedExact {
		if status == exact {
			return true
		}
	}
	for _, prefix := range spec.AllowedPrefixes {
		if strings.HasPrefix(status, prefix) {
			return true
		}
	}
	return false
}

func (report *stackChanHardwareMainlineReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	copy := make(map[string]string, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}

func postStackChanMicProbeControl(gatewayBaseURL string, deviceID string, state protocol.ExpressionState, mode protocol.Mode, text string, traceID string, sessionID string, audioProbeOnly bool, mockPlaybackOnNextAudioFrame bool) (gateway.DeviceControlResponse, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/v1/devices/control", nil)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	request := gateway.DeviceControlRequest{
		DeviceID:                     deviceID,
		State:                        state,
		Mode:                         mode,
		Text:                         text,
		TraceID:                      traceID,
		SessionID:                    sessionID,
		AudioProbeOnly:               audioProbeOnly,
		MockPlaybackOnNextAudioFrame: mockPlaybackOnNextAudioFrame,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var response gateway.DeviceControlResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	if containsLegacyIdentity(response.TraceID) || containsLegacyIdentity(response.SessionID) || containsLegacyIdentity(response.DeviceID) {
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control contains forbidden legacy identity")
	}
	return response, nil
}

func postStackChanSpeakerControl(gatewayBaseURL string, deviceID string, state protocol.ExpressionState, mode protocol.Mode, text string, traceID string, sessionID string, streamID string, mockAudioChunks int) (gateway.DeviceControlResponse, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/v1/devices/control", nil)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	request := gateway.DeviceControlRequest{
		DeviceID:        deviceID,
		State:           state,
		Mode:            mode,
		Text:            text,
		TraceID:         traceID,
		SessionID:       sessionID,
		StreamID:        streamID,
		MockAudioChunks: mockAudioChunks,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var response gateway.DeviceControlResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	if containsLegacyIdentity(response.TraceID) || containsLegacyIdentity(response.SessionID) || containsLegacyIdentity(response.DeviceID) {
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control contains forbidden legacy identity")
	}
	return response, nil
}

func postStackChanAudioPlaybackBatch(gatewayBaseURL string, deviceID string, traceID string, sessionID string, streamID string, chunks []protocol.AudioPlaybackChunk) (gateway.DeviceControlResponse, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/v1/devices/control", nil)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	request := gateway.DeviceControlRequest{
		DeviceID:    deviceID,
		State:       protocol.ExpressionSpeaking,
		Mode:        protocol.ModeWorkmate,
		Text:        "A21 LOCAL TTS",
		TraceID:     traceID,
		SessionID:   sessionID,
		StreamID:    streamID,
		AudioChunks: chunks,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	client := http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var response gateway.DeviceControlResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	if containsLegacyIdentity(response.TraceID) || containsLegacyIdentity(response.SessionID) || containsLegacyIdentity(response.DeviceID) {
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control contains forbidden legacy identity")
	}
	return response, nil
}

func postStackChanSpeakerProbeBatches(gatewayBaseURL string, deviceID string, traceID string, sessionID string, streamID string, totalChunks int) (gateway.DeviceControlResponse, error) {
	remaining := totalChunks
	var firstControl gateway.DeviceControlResponse
	for remaining > 0 {
		consumedChunks := stackChanSpeakerProbeBatchChunks
		if remaining < consumedChunks {
			consumedChunks = remaining
		}
		control, err := postStackChanSpeakerControl(gatewayBaseURL, deviceID, protocol.ExpressionSpeaking, protocol.ModeWorkmate, "SPEAKER PROBE", traceID, sessionID, streamID, stackChanSpeakerProbeBatchChunks)
		if err != nil {
			return firstControl, err
		}
		if firstControl.TraceID == "" {
			firstControl = control
		}
		remaining -= consumedChunks
		if remaining > 0 {
			time.Sleep(time.Duration(stackChanSpeakerProbeBatchChunks*stackChanSpeakerProbeChunkDurationMS) * time.Millisecond)
		}
	}
	return firstControl, nil
}

func postStackChanTouchControl(gatewayBaseURL string, deviceID string, spec stackChanTouchCaseSpec) (gateway.DeviceControlResponse, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/v1/devices/control", nil)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	request := gateway.DeviceControlRequest{
		DeviceID:        deviceID,
		State:           spec.ControlState,
		Mode:            spec.ControlMode,
		Text:            spec.DevicePrompt,
		TraceID:         "a21-trace-touch-acceptance-" + spec.Name,
		SessionID:       "a21-session-touch-acceptance",
		StreamID:        spec.StreamID,
		MockAudioChunks: spec.MockAudioChunks,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var response gateway.DeviceControlResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	if containsLegacyIdentity(response.TraceID) || containsLegacyIdentity(response.SessionID) || containsLegacyIdentity(response.DeviceID) {
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control contains forbidden legacy identity")
	}
	return response, nil
}

func candidateStackChanTouchTraceIDs(lastTraceID string) []string {
	lastTraceID = strings.TrimSpace(lastTraceID)
	if lastTraceID == "" {
		return nil
	}
	ids := []string{lastTraceID}
	const prefix = "a21-trace-device-"
	if !strings.HasPrefix(lastTraceID, prefix) {
		return ids
	}
	suffix := strings.TrimPrefix(lastTraceID, prefix)
	value, err := strconv.Atoi(suffix)
	if err != nil {
		return ids
	}
	for candidate := value - 1; candidate >= 0 && candidate >= value-6; candidate-- {
		ids = append(ids, fmt.Sprintf("%s%0*d", prefix, len(suffix), candidate))
	}
	return ids
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

func parsePositiveIntOrDefault(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
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

func writeLocalTTSSmokeReport(outputDir string, report audio.LocalTTSReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-local-tts-smoke-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONLocalTTSSmoke(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeLocalASRSmokeReport(outputDir string, report audio.LocalASRReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-local-asr-smoke-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONLocalASRSmoke(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeLocalVoiceLoopbackReport(outputDir string, report localVoiceLoopbackReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-local-voice-loopback-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONLocalVoiceLoopback(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanLocalTTSPlaybackReport(outputDir string, report stackChanLocalTTSPlaybackReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-local-tts-playback-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanLocalTTSPlayback(file, report); err != nil {
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

func writeStackChanMicProbeAcceptanceReport(outputDir string, report stackChanMicProbeAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-mic-probe-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanMicProbeAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanIMUProbeAcceptanceReport(outputDir string, report stackChanIMUProbeAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-imu-probe-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanIMUProbeAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanSensorProbeAcceptanceReport(outputDir string, report stackChanSensorProbeAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-sensor-probe-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanSensorProbeAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanHalfDuplexAcceptanceReport(outputDir string, report stackChanHalfDuplexAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-half-duplex-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanHalfDuplexAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanSpeakerAcceptanceReport(outputDir string, report stackChanSpeakerAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-speaker-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanSpeakerAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanTouchAcceptanceReport(outputDir string, report stackChanTouchAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-touch-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanTouchAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanHardwareMainlineReport(outputDir string, report stackChanHardwareMainlineReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-hardware-mainline-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanHardwareMainline(file, report); err != nil {
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

func writeFirmwareSensorProbeFlashPlanReport(outputDir string, result firmwarecheck.SensorProbeFlashPlanResult) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-firmware-sensor-probe-flash-plan-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	result.ReportPath = reportPath
	if err := writeJSONFirmwareSensorProbeFlashPlan(file, result); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeFirmwareSensorProbeFlashExecutionReport(outputDir string, report firmwareSensorProbeFlashExecutionReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-firmware-sensor-probe-flash-execution-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONFirmwareSensorProbeFlashExecution(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeProviderSmokeReport(outputDir string, report providers.ProviderSmokeReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	var reportPath string
	var file *os.File
	var err error
	for attempt := 0; attempt < 100; attempt++ {
		now := time.Now()
		stamp := fmt.Sprintf("%s-%09d", now.Format("20060102-150405"), now.Nanosecond())
		if attempt > 0 {
			stamp = fmt.Sprintf("%s-%02d", stamp, attempt)
		}
		reportPath = filepath.Join(outputDir, "a21-provider-smoke-"+stamp+".json")
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
		return "", fmt.Errorf("could not allocate unique provider smoke report path")
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

func writeJSONLocalTTSSmoke(writer io.Writer, report audio.LocalTTSReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONLocalASRSmoke(writer io.Writer, report audio.LocalASRReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONLocalVoiceLoopback(writer io.Writer, report localVoiceLoopbackReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanLocalTTSPlayback(writer io.Writer, report stackChanLocalTTSPlaybackReport) error {
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

func writeJSONStackChanMicProbeAcceptance(writer io.Writer, report stackChanMicProbeAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanIMUProbeAcceptance(writer io.Writer, report stackChanIMUProbeAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanSensorProbeAcceptance(writer io.Writer, report stackChanSensorProbeAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanHalfDuplexAcceptance(writer io.Writer, report stackChanHalfDuplexAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanSpeakerAcceptance(writer io.Writer, report stackChanSpeakerAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanTouchAcceptance(writer io.Writer, report stackChanTouchAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanHardwareMainline(writer io.Writer, report stackChanHardwareMainlineReport) error {
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
	options := gateway.ServerOptions{
		VoiceProvider: providers.NewGatewayVoiceProviderFromEnv(env),
	}
	if adapterURL := strings.TrimSpace(appEnvValue(env, "A21_V21_ADAPTER_URL")); adapterURL != "" {
		client, err := v21adapter.NewHTTPClient(adapterURL)
		if err != nil {
			options.V21Client = v21ConfigurationErrorClient{err: err}
		} else {
			options.V21Client = client
		}
	}
	return gateway.NewServerWithOptions(options)
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
			fmt.Fprintln(stdout, "a21 firmware-bootstrap-flash-plan --artifact firmware/artifacts/<a21-stackchan...bin> --port /dev/cu.usbmodemXXXX --commit <git-sha> [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
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
			fmt.Fprintln(stdout, "a21 firmware-bootstrap-flash-execute --artifact firmware/artifacts/<a21-stackchan...bin> --port /dev/cu.usbmodemXXXX --commit <git-sha> --confirm WRITE_A21_STACKCHAN_FIRMWARE [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
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
	controlGuard, code := requireA21ControlAllowed("firmware-bootstrap-flash-execute", stderr)
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
			fmt.Fprintln(stdout, "a21 firmware-imu-probe-flash-plan --port /dev/cu.usbmodemXXXX --commit <git-sha> [--artifact firmware/stackchan/.pio/build/a21_stackchan_cores3_imu_probe/firmware.bin] [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3_imu_probe] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
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
			fmt.Fprintln(stdout, "a21 firmware-imu-probe-flash-execute --port /dev/cu.usbmodemXXXX --commit <git-sha> --confirm WRITE_A21_STACKCHAN_IMU_PROBE_FIRMWARE [--artifact firmware/stackchan/.pio/build/a21_stackchan_cores3_imu_probe/firmware.bin] [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3_imu_probe] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
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
	controlGuard, code := requireA21ControlAllowed("firmware-imu-probe-flash-execute", stderr)
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

func runFirmwareSensorProbeFlashPlan(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.SensorProbeFlashPlanOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		ArtifactPath: filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanSensorProbePlatformIOEnv, "firmware.bin"),
		BuildDir:     filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanSensorProbePlatformIOEnv),
		CoreDir:      filepath.Join(".a21-tools", "platformio-core"),
	}
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-sensor-probe-flash-plan --port /dev/cu.usbmodemXXXX --commit <git-sha> [--artifact firmware/stackchan/.pio/build/a21_stackchan_cores3_sensor_probe/firmware.bin] [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3_sensor_probe] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
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
			fmt.Fprintf(stderr, "unknown firmware-sensor-probe-flash-plan option %q\n", args[i])
			return 2
		}
	}
	result, code := buildFirmwareSensorProbeFlashPlanFromCLI(options, outputDir, stderr)
	if code != 0 {
		return code
	}
	if err := writeJSONFirmwareSensorProbeFlashPlan(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode firmware sensor probe flash plan: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware sensor probe flash plan ok (no flash performed)")
	return 0
}

func runFirmwareSensorProbeFlashExecute(args []string, stdout io.Writer, stderr io.Writer) int {
	options := firmwarecheck.SensorProbeFlashPlanOptions{
		ManifestPath: "firmware/stackchan/a21-firmware.json",
		ArtifactPath: filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanSensorProbePlatformIOEnv, "firmware.bin"),
		BuildDir:     filepath.Join("firmware", "stackchan", ".pio", "build", firmwarecheck.StackChanSensorProbePlatformIOEnv),
		CoreDir:      filepath.Join(".a21-tools", "platformio-core"),
	}
	outputDir := ""
	confirm := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-sensor-probe-flash-execute --port /dev/cu.usbmodemXXXX --commit <git-sha> --confirm WRITE_A21_STACKCHAN_SENSOR_PROBE_FIRMWARE [--artifact firmware/stackchan/.pio/build/a21_stackchan_cores3_sensor_probe/firmware.bin] [--build-dir firmware/stackchan/.pio/build/a21_stackchan_cores3_sensor_probe] [--core-dir .a21-tools/platformio-core] [--output-dir reports]")
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
				fmt.Fprintln(stderr, "--confirm requires WRITE_A21_STACKCHAN_SENSOR_PROBE_FIRMWARE")
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
			fmt.Fprintf(stderr, "unknown firmware-sensor-probe-flash-execute option %q\n", args[i])
			return 2
		}
	}
	if confirm != "WRITE_A21_STACKCHAN_SENSOR_PROBE_FIRMWARE" {
		fmt.Fprintln(stderr, "--confirm WRITE_A21_STACKCHAN_SENSOR_PROBE_FIRMWARE is required")
		return 2
	}
	controlGuard, code := requireA21ControlAllowed("firmware-sensor-probe-flash-execute", stderr)
	if code != 0 {
		return code
	}
	plan, code := buildFirmwareSensorProbeFlashPlanFromCLI(options, "", stderr)
	if code != 0 {
		return code
	}
	command, err := firmwareSensorProbeFlashCommand(plan)
	if err != nil {
		fmt.Fprintf(stderr, "firmware sensor probe flash execute failed: %v\n", err)
		return 1
	}
	if err := runFirmwareBootstrapFlashCommand(context.Background(), command, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "firmware sensor probe flash execute failed: %v\n", err)
		return 1
	}
	report := firmwareSensorProbeFlashExecutionReport{
		SchemaVersion:  "a21.firmware.sensor_probe_flash_execution.v1",
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
			fmt.Fprintf(stderr, "firmware sensor probe flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeFirmwareSensorProbeFlashExecutionReport(outputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write firmware sensor probe flash execution report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONFirmwareSensorProbeFlashExecution(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode firmware sensor probe flash execution: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "firmware sensor probe flash executed")
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

func buildFirmwareSensorProbeFlashPlanFromCLI(options firmwarecheck.SensorProbeFlashPlanOptions, outputDir string, stderr io.Writer) (firmwarecheck.SensorProbeFlashPlanResult, int) {
	if options.Port == "" {
		fmt.Fprintln(stderr, "--port requires a value")
		return firmwarecheck.SensorProbeFlashPlanResult{}, 2
	}
	if options.ExpectedGitCommit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return firmwarecheck.SensorProbeFlashPlanResult{}, 2
	}
	for _, path := range []string{options.ManifestPath, options.ArtifactPath, options.BuildDir, options.CoreDir} {
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "firmware sensor probe flash path invalid: %v\n", err)
			return firmwarecheck.SensorProbeFlashPlanResult{}, 1
		}
	}
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "firmware sensor probe flash report dir invalid: %v\n", err)
			return firmwarecheck.SensorProbeFlashPlanResult{}, 1
		}
	}
	sourceState, err := detectFirmwareSourceState()
	if err != nil {
		fmt.Fprintf(stderr, "firmware sensor probe flash failed: source tree check failed: %v\n", err)
		return firmwarecheck.SensorProbeFlashPlanResult{}, 1
	}
	if !sourceState.Clean {
		fmt.Fprintf(stderr, "firmware sensor probe flash failed: source tree is dirty under %s\n%s\n", sourceState.Root, sourceState.Detail)
		return firmwarecheck.SensorProbeFlashPlanResult{}, 1
	}
	portUsage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		fmt.Fprintf(stderr, "firmware sensor probe flash failed: upload port ownership check failed: %v\n", err)
		return firmwarecheck.SensorProbeFlashPlanResult{}, 1
	}
	options.PortUsage = portUsage
	result, err := firmwarecheck.BuildSensorProbeFlashPlan(options)
	if err != nil {
		fmt.Fprintf(stderr, "firmware sensor probe flash failed: %v\n", err)
		return firmwarecheck.SensorProbeFlashPlanResult{}, 1
	}
	if outputDir != "" {
		reportPath, err := writeFirmwareSensorProbeFlashPlanReport(outputDir, result)
		if err != nil {
			fmt.Fprintf(stderr, "write firmware sensor probe flash plan: %v\n", err)
			return firmwarecheck.SensorProbeFlashPlanResult{}, 1
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

func firmwareSensorProbeFlashCommand(plan firmwarecheck.SensorProbeFlashPlanResult) ([]string, error) {
	if plan.GuardID != "a21.firmware.sensor_probe_flash_plan.v1" || !plan.OK {
		return nil, fmt.Errorf("invalid sensor probe flash plan")
	}
	if plan.PlatformIOEnv != firmwarecheck.StackChanSensorProbePlatformIOEnv {
		return nil, fmt.Errorf("sensor probe flash plan has wrong PlatformIO env")
	}
	if plan.Port == "" || len(plan.Parts) != 4 {
		return nil, fmt.Errorf("sensor probe flash plan missing port or image parts")
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
			return nil, fmt.Errorf("sensor probe flash plan contains empty image part")
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

func writeJSONFirmwareSensorProbeFlashPlan(writer io.Writer, result firmwarecheck.SensorProbeFlashPlanResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func writeJSONFirmwareSensorProbeFlashExecution(writer io.Writer, report firmwareSensorProbeFlashExecutionReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
