package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/protocol"
)

type roleplayVoiceProbeOptions struct {
	GatewayURL           string
	DeviceID             string
	TraceID              string
	SessionID            string
	OutputDir            string
	ASRProvider          string
	ProbeFrameDurationMS int
	RequireReady         bool
}

func runRoleplayVoiceProbe(args []string, stdout io.Writer, stderr io.Writer) int {
	options := roleplayVoiceProbeOptions{
		GatewayURL:           firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:             firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-sim-001"),
		OutputDir:            "reports",
		ASRProvider:          firstNonEmpty(strings.TrimSpace(os.Getenv("A21_LOCAL_ASR_PROVIDER")), "roleplay_probe"),
		ProbeFrameDurationMS: 20,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 roleplay-voice-probe [--gateway-url http://127.0.0.1:21080] [--device-id stackchan-sim-001] [--trace-id a21-trace-roleplay-voice-probe-...] [--session-id a21-session-roleplay-voice-probe-...] [--asr-provider roleplay_probe] [--output-dir reports] [--require-ready]")
			return 0
		case "--gateway-url":
			if !readStringOption(args, &i, stderr, "--gateway-url", &options.GatewayURL) {
				return 2
			}
		case "--device-id":
			if !readStringOption(args, &i, stderr, "--device-id", &options.DeviceID) {
				return 2
			}
		case "--trace-id":
			if !readStringOption(args, &i, stderr, "--trace-id", &options.TraceID) {
				return 2
			}
		case "--session-id":
			if !readStringOption(args, &i, stderr, "--session-id", &options.SessionID) {
				return 2
			}
		case "--asr-provider":
			if !readStringOption(args, &i, stderr, "--asr-provider", &options.ASRProvider) {
				return 2
			}
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		case "--require-ready":
			options.RequireReady = true
		default:
			fmt.Fprintf(stderr, "unknown roleplay-voice-probe option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "roleplay voice probe report dir invalid: %v\n", err)
		return 1
	}
	report := buildRoleplayVoiceProbeReport(context.Background(), options)
	reportPath, err := writeRoleplayVoiceProbeReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write roleplay voice probe report: %v\n", err)
		return 1
	}
	if err := writeJSONRoleplayVoiceProbe(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode roleplay voice probe report: %v\n", err)
		return 1
	}
	if options.RequireReady && report.Status != "passed" {
		fmt.Fprintf(stderr, "roleplay voice probe not ready; report=%s\n", filepath.Base(reportPath))
		return 1
	}
	return 0
}

func buildRoleplayVoiceProbeReport(ctx context.Context, options roleplayVoiceProbeOptions) productRoleplayVoiceReportFixture {
	generatedAtMS := time.Now().UnixMilli()
	traceID := strings.TrimSpace(options.TraceID)
	if traceID == "" {
		traceID = fmt.Sprintf("a21-trace-roleplay-voice-probe-%d", generatedAtMS)
	}
	sessionID := strings.TrimSpace(options.SessionID)
	if sessionID == "" {
		sessionID = fmt.Sprintf("a21-session-roleplay-voice-probe-%d", generatedAtMS)
	}
	report := productRoleplayVoiceReportFixture{
		SchemaVersion: productRoleplayVoiceReportSchemaVersion,
		GeneratedAtMS: &generatedAtMS,
		Status:        "blocked",
		Mode:          string(protocol.ModeRoleplay),
		Route:         "fast_companion_hybrid",
		ExecutionMode: "gateway_fast_companion",
		DeviceID:      strings.TrimSpace(options.DeviceID),
		TraceID:       traceID,
		SessionID:     sessionID,
		Runtime:       defaultRoleplayVoiceProbeRuntime(),
		VoicePipeline: productRoleplayVoicePipelineFixture{
			Status:   "blocked",
			Findings: []string{"gateway_turn_not_completed"},
		},
		Redaction: productRoleplayVoiceRedaction{},
	}
	response, err := postRoleplayVoiceProbeTurn(ctx, options, traceID, sessionID)
	if err != nil {
		report.VoicePipeline.Findings = []string{"gateway_turn_failed"}
		return report
	}
	report.TraceID = firstNonEmpty(strings.TrimSpace(response.TraceID), traceID)
	report.SessionID = firstNonEmpty(strings.TrimSpace(response.SessionID), sessionID)
	report.DeviceID = firstNonEmpty(strings.TrimSpace(response.DeviceID), strings.TrimSpace(options.DeviceID))
	report.Route = firstNonEmpty(strings.TrimSpace(response.Route), "fast_companion_hybrid")
	if strings.TrimSpace(response.Roleplay.RoleplayProfile) != "" ||
		strings.TrimSpace(response.Roleplay.Scenario) != "" ||
		strings.TrimSpace(response.Roleplay.VoiceCloneProfile) != "" {
		report.Runtime = response.Roleplay
	}
	report.VoicePipeline = productRoleplayVoicePipelineFixture{
		Observed:                response.Status == "pipeline_completed",
		Status:                  firstNonEmpty(strings.TrimSpace(response.Status), "blocked"),
		TextStreamProvider:      providerLatencySafeIdentifier(response.TextStreamProvider, false),
		ProviderFamily:          providerLatencySafeIdentifier(response.ProviderFamily, false),
		TextStreamExecuted:      response.TextStreamExecuted,
		PromptInputUsed:         false,
		VoiceCloneProfileUsed:   false,
		AudioDownlinkFirstFrame: false,
		DevicePlaybackStart:     false,
		AudioChunkCount:         roleplayVoiceProbeAudioChunkCount(response.Events),
	}
	trace, err := fetchGatewayTrace(options.GatewayURL, report.TraceID)
	if err != nil {
		report.VoicePipeline.Findings = append(report.VoicePipeline.Findings, "gateway_trace_unavailable")
	} else {
		report.TraceMarkers = roleplayVoiceProbeTraceMarkers(trace)
		report.VoicePipeline.Observed = report.VoicePipeline.Observed || productRoleplayVoiceHasMarker(report.TraceMarkers, "fast_companion.voice_pipeline.start")
		report.VoicePipeline.PromptInputUsed = productRoleplayVoiceHasMarker(report.TraceMarkers, "roleplay.prompt_input.used")
		report.VoicePipeline.VoiceCloneProfileUsed = productRoleplayVoiceHasMarker(report.TraceMarkers, "roleplay.voice_clone_profile.used")
		report.VoicePipeline.AudioDownlinkFirstFrame = productRoleplayVoiceHasMarker(report.TraceMarkers, "audio.downlink.first_frame")
		report.VoicePipeline.DevicePlaybackStart = productRoleplayVoiceHasMarker(report.TraceMarkers, "device.playback.start")
		report.VoicePipeline.AudioChunkCount += roleplayVoiceProbeMarkerCount(report.TraceMarkers, "audio.playback.chunk.sent")
	}
	report.VoicePipeline.Findings = roleplayVoiceProbeFindings(report)
	if roleplayVoiceProbeReportReady(report) {
		report.Status = "passed"
		report.VoicePipeline.Findings = nil
	}
	return report
}

func postRoleplayVoiceProbeTurn(ctx context.Context, options roleplayVoiceProbeOptions, traceID string, sessionID string) (gateway.FastCompanionTurnResponse, error) {
	endpoint, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/fast-companion/turn", nil)
	if err != nil {
		return gateway.FastCompanionTurnResponse{}, err
	}
	durationMS := options.ProbeFrameDurationMS
	if durationMS <= 0 {
		durationMS = 20
	}
	payload := latencyBenchPCM16Base64(0)
	request := gateway.FastCompanionTurnRequest{
		DeviceID:  strings.TrimSpace(options.DeviceID),
		Mode:      protocol.ModeRoleplay,
		TraceID:   traceID,
		SessionID: sessionID,
		LocalAudio: gateway.FastCompanionLocalAudioResult{
			ASRProvider:          firstNonEmpty(strings.TrimSpace(options.ASRProvider), "roleplay_probe"),
			FirstPartialMS:       32,
			FinalTranscriptChars: 1,
			Frames: []gateway.FastCompanionLocalAudioFrame{{
				Seq:          1,
				Codec:        string(protocol.AudioCodecPCMS16LE),
				SampleRateHz: 16000,
				Channels:     1,
				DurationMS:   durationMS,
				ByteCount:    640,
				RMS:          0,
				DataBase64:   payload,
			}},
		},
	}
	data, err := json.Marshal(request)
	if err != nil {
		return gateway.FastCompanionTurnResponse{}, err
	}
	client := http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return gateway.FastCompanionTurnResponse{}, err
	}
	req.Header.Set("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return gateway.FastCompanionTurnResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return gateway.FastCompanionTurnResponse{}, fmt.Errorf("gateway turn status %d", resp.StatusCode)
	}
	var response gateway.FastCompanionTurnResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return gateway.FastCompanionTurnResponse{}, err
	}
	return response, nil
}

func defaultRoleplayVoiceProbeRuntime() gateway.RoleplayRuntimeSummary {
	return gateway.RoleplayRuntimeSummary{
		SchemaVersion:        "a21.roleplay_runtime.v1",
		Mode:                 string(protocol.ModeRoleplay),
		RoleplayProfile:      "unavailable",
		Scenario:             "unavailable",
		VoiceCloneProfile:    "unavailable",
		MemoryPolicy:         "unavailable",
		MemoryConfigured:     true,
		MemoryHintCount:      0,
		SoulPromptInputReady: false,
		PromptComposed:       false,
	}
}

func roleplayVoiceProbeReportReady(report productRoleplayVoiceReportFixture) bool {
	candidate := report
	candidate.Status = "passed"
	evidence, ok := productRoleplayVoiceReportEvidenceFromFixture("a21-roleplay-voice-probe-generated.json", candidate)
	return ok && evidence.VoiceRuntimeReady
}

func roleplayVoiceProbeAudioChunkCount(events []protocol.Envelope) int {
	count := 0
	for _, event := range events {
		if event.Kind == protocol.KindAudioPlaybackChunk {
			count++
		}
	}
	return count
}

func roleplayVoiceProbeTraceMarkers(trace gateway.TraceResponse) []string {
	markers := make([]string, 0, len(trace.Events))
	seen := map[string]bool{}
	for _, event := range trace.Events {
		name := strings.TrimSpace(event.Name)
		if name == "" || !productRoleplayVoiceMarkerAllowed(name) || seen[name] {
			continue
		}
		seen[name] = true
		markers = append(markers, name)
	}
	return markers
}

func roleplayVoiceProbeMarkerCount(markers []string, want string) int {
	count := 0
	for _, marker := range markers {
		if strings.TrimSpace(marker) == want {
			count++
		}
	}
	return count
}

func roleplayVoiceProbeFindings(report productRoleplayVoiceReportFixture) []string {
	findings := make([]string, 0, 8)
	pipeline := report.VoicePipeline
	if pipeline.Status != "pipeline_completed" {
		findings = append(findings, "voice_pipeline_not_completed")
	}
	if !pipeline.TextStreamExecuted {
		findings = append(findings, "text_stream_not_executed")
	}
	if !pipeline.PromptInputUsed {
		findings = append(findings, "prompt_input_not_observed")
	}
	if !pipeline.VoiceCloneProfileUsed {
		findings = append(findings, "voice_clone_profile_not_observed")
	}
	if !pipeline.AudioDownlinkFirstFrame {
		findings = append(findings, "audio_downlink_not_observed")
	}
	if !pipeline.DevicePlaybackStart {
		findings = append(findings, "device_playback_not_observed")
	}
	if pipeline.AudioChunkCount <= 0 {
		findings = append(findings, "audio_chunk_not_observed")
	}
	return findings
}

func writeRoleplayVoiceProbeReport(outputDir string, report productRoleplayVoiceReportFixture) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-roleplay-voice-probe-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if err := writeJSONRoleplayVoiceProbe(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeJSONRoleplayVoiceProbe(writer io.Writer, report productRoleplayVoiceReportFixture) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
