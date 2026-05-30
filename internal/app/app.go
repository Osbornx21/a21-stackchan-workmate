package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
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
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "provider-smoke":
		return runProviderSmoke(args[1:], stdout, stderr)
	case "provider-realtime-plan":
		return runProviderRealtimePlan(args[1:], stdout, stderr)
	case "provider-realtime-fixture":
		return runProviderRealtimeFixture(args[1:], stdout, stderr)
	case "audio-front-end-plan":
		return runAudioFrontEndPlan(args[1:], stdout, stderr)
	case "audio-front-end-eval":
		return runAudioFrontEndEval(args[1:], stdout, stderr)
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
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 provider-smoke --provider <provider> [--execute]")
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
			fmt.Fprintf(stderr, "unknown provider-smoke option %q\n", args[i])
			return 2
		}
	}
	report := providers.ProviderSmokeFromEnv(context.Background(), os.Environ(), provider, execute, nil)
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
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 audio-front-end-eval --mock")
			return 0
		case "--mock":
			mock = true
		case "--execute":
			fmt.Fprintln(stderr, "audio-front-end-eval only supports --mock until recorded-office and physical-device fixtures exist")
			return 2
		default:
			fmt.Fprintf(stderr, "unknown audio-front-end-eval option %q\n", args[i])
			return 2
		}
	}
	if !mock {
		fmt.Fprintln(stderr, "audio-front-end-eval requires --mock")
		return 2
	}
	if err := writeJSONAudioFrontEndEval(stdout, audio.RunMockFrontEndEval()); err != nil {
		fmt.Fprintf(stderr, "encode audio front-end eval: %v\n", err)
		return 1
	}
	return 0
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
	Mode       string              `json:"mode"`
	Iterations int                 `json:"iterations"`
	Summary    latencyBenchSummary `json:"summary"`
	OK         bool                `json:"ok"`
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
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 latency-bench --mock --iterations 5")
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

func writeJSONProviderSmoke(writer io.Writer, report providers.ProviderSmokeReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONAudioFrontEndPlan(writer io.Writer, report audio.FrontEndPlan) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONAudioFrontEndEval(writer io.Writer, report audio.FrontEndEvalReport) error {
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
			fmt.Fprintln(stdout, "a21 firmware-device-check --artifact firmware/artifacts/<a21-stackchan...bin> --device-report reports/devices.json --device-id stackchan-001 --commit <git-sha>")
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
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 firmware-flash-plan --artifact firmware/artifacts/<a21-stackchan...bin> --port /dev/cu.usbmodemXXXX --device-report reports/devices.json --device-id stackchan-001 --commit <git-sha>")
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
