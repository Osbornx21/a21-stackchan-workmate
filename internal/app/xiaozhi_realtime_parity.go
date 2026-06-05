package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/gateway"
)

const xiaozhiRealtimeParitySchemaVersion = "a21.xiaozhi_realtime_parity.v1"

type xiaozhiRealtimeParityOptions struct {
	GatewayURL string
	DeviceID   string
	TraceID    string
	SessionID  string
	OutputDir  string
}

type xiaozhiRealtimeParityReport struct {
	SchemaVersion        string                             `json:"schema_version"`
	GeneratedAtMS        int64                              `json:"generated_at_ms"`
	ExecutionMode        string                             `json:"execution_mode"`
	Gateway              string                             `json:"gateway"`
	DeviceID             string                             `json:"device_id"`
	TraceID              string                             `json:"trace_id"`
	SessionID            string                             `json:"session_id"`
	Profile              string                             `json:"profile"`
	Transport            string                             `json:"transport"`
	PhysicalDeviceOnline bool                               `json:"physical_device_online"`
	Counts               xiaozhiRealtimeParityCounts        `json:"counts"`
	Ordering             xiaozhiRealtimeParityOrdering      `json:"ordering"`
	StageAvailability    map[string]physicalStackChanMetric `json:"stage_availability"`
	Classification       string                             `json:"classification"`
	AcceptanceStatus     string                             `json:"acceptance_status"`
	PRDAccepted          bool                               `json:"prd_accepted"`
	Findings             []physicalStackChanEvidenceFinding `json:"findings"`
	Redaction            xiaozhiVoiceBenchRedaction         `json:"redaction"`
	ReportPath           string                             `json:"report_path,omitempty"`
}

type xiaozhiRealtimeParityCounts struct {
	TraceEvents             int `json:"trace_events"`
	ListenStart             int `json:"listen_start"`
	ListenAutoStop          int `json:"listen_auto_stop"`
	OpusFramesReceived      int `json:"opus_frames_received"`
	OpusFramesDecoded       int `json:"opus_frames_decoded"`
	PCMIngressFrames        int `json:"pcm_ingress_frames"`
	VADSpeechStart          int `json:"vad_speech_start"`
	VADSpeechEnd            int `json:"vad_speech_end"`
	ASRFirstPartial         int `json:"asr_first_partial"`
	ASRFinal                int `json:"asr_final"`
	ASRStreamStart          int `json:"asr_stream_start"`
	ASRAudioAppend          int `json:"asr_audio_append"`
	ASRStreamCommit         int `json:"asr_stream_commit"`
	ProviderFirstContent    int `json:"provider_first_content"`
	TTSFirstAudio           int `json:"tts_first_audio"`
	AudioDownlinkFirstFrame int `json:"audio_downlink_first_frame"`
	OpusDownlinkFrames      int `json:"opus_downlink_frames"`
	AnswerDownlinkFrames    int `json:"answer_downlink_frames"`
	VoicePipelineStart      int `json:"voice_pipeline_start"`
	VoicePipelineCompleted  int `json:"voice_pipeline_completed"`
	DevicePlaybackStart     int `json:"device_playback_start"`
	ASRRealStreamingProfile int `json:"asr_real_streaming_profile"`
	LLMRealStreamingProfile int `json:"llm_real_streaming_profile"`
	TTSRealStreamingProfile int `json:"tts_real_streaming_profile"`
	RealtimeProfileBlocked  int `json:"realtime_profile_blocked"`
	ForbiddenFakePath       int `json:"forbidden_fake_path"`
}

type xiaozhiRealtimeParityOrdering struct {
	StreamingASRBeforeSpeechEnd       bool `json:"streaming_asr_before_speech_end"`
	ProviderBeforeASRFinal            bool `json:"provider_before_asr_final"`
	TTSBeforePipelineCompleted        bool `json:"tts_before_pipeline_completed"`
	DownlinkBeforePipelineCompleted   bool `json:"downlink_before_pipeline_completed"`
	PhysicalOpusIngressBeforePipeline bool `json:"physical_opus_ingress_before_pipeline"`
}

func runXiaozhiRealtimeParity(args []string, stdout io.Writer, stderr io.Writer) int {
	options := xiaozhiRealtimeParityOptions{
		GatewayURL: firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		OutputDir:  "reports",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 xiaozhi-realtime-parity --gateway-url http://127.0.0.1:21080 --device-id <device-id> --trace-id <trace-id> --session-id <session-id> [--output-dir reports]")
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
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown xiaozhi-realtime-parity option %q\n", args[i])
			return 2
		}
	}
	if err := validateXiaozhiRealtimeParityOptions(options); err != nil {
		fmt.Fprintf(stderr, "xiaozhi realtime parity option invalid: %v\n", err)
		return 2
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "xiaozhi realtime parity report dir invalid: %v\n", err)
		return 1
	}
	report, err := buildXiaozhiRealtimeParityReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "xiaozhi realtime parity failed: %v\n", err)
		return 1
	}
	if options.OutputDir != "" {
		reportPath, err := writeXiaozhiRealtimeParityReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "xiaozhi realtime parity report write failed: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONXiaozhiRealtimeParity(stdout, report); err != nil {
		fmt.Fprintf(stderr, "xiaozhi realtime parity report encode failed: %v\n", err)
		return 1
	}
	if !xiaozhiRealtimeParityCommandAccepted(report.Classification) {
		return 1
	}
	return 0
}

func validateXiaozhiRealtimeParityOptions(options xiaozhiRealtimeParityOptions) error {
	if _, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/devices", nil); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"device_id":  options.DeviceID,
		"trace_id":   options.TraceID,
		"session_id": options.SessionID,
	} {
		if !xiaozhiPhysicalSafeID(value) {
			return fmt.Errorf("%s is invalid or unsafe", name)
		}
	}
	return nil
}

func buildXiaozhiRealtimeParityReport(options xiaozhiRealtimeParityOptions) (xiaozhiRealtimeParityReport, error) {
	deviceReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		return xiaozhiRealtimeParityReport{}, err
	}
	device, ok := findFirmwareDeviceRecord(deviceReport.Devices, options.DeviceID)
	if !ok {
		return xiaozhiRealtimeParityReport{}, fmt.Errorf("device not found in gateway registry")
	}
	trace, err := fetchGatewayTrace(options.GatewayURL, options.TraceID)
	if err != nil {
		return xiaozhiRealtimeParityReport{}, err
	}
	if err := validateXiaozhiRealtimeParityTrace(options, trace); err != nil {
		return xiaozhiRealtimeParityReport{}, err
	}
	counts := xiaozhiRealtimeParityCountEvents(trace.Events)
	ordering := xiaozhiRealtimeParityOrderingFromTrace(trace.Events)
	report := xiaozhiRealtimeParityReport{
		SchemaVersion:        xiaozhiRealtimeParitySchemaVersion,
		GeneratedAtMS:        time.Now().UnixMilli(),
		ExecutionMode:        "trace_only_physical_realtime_parity",
		Gateway:              productSurfaceLabel(options.GatewayURL, ""),
		DeviceID:             options.DeviceID,
		TraceID:              options.TraceID,
		SessionID:            options.SessionID,
		Profile:              firstNonEmpty(strings.TrimSpace(device.Capabilities["xiaozhi_profile"]), "unknown"),
		Transport:            firstNonEmpty(strings.TrimSpace(device.Capabilities["xiaozhi_transport"]), "unknown"),
		PhysicalDeviceOnline: xiaozhiPhysicalDeviceOnline(device),
		Counts:               counts,
		Ordering:             ordering,
		AcceptanceStatus:     "blocked",
		PRDAccepted:          false,
		Redaction: xiaozhiVoiceBenchRedaction{
			PayloadsStored:         false,
			CredentialValuesStored: false,
			FullURLsStored:         false,
			LocalPathsStored:       false,
		},
	}
	report.StageAvailability = xiaozhiRealtimeParityStageAvailability(report, device, trace)
	report.Classification = xiaozhiRealtimeParityClassification(report)
	report.AcceptanceStatus = report.Classification
	report.Findings = xiaozhiRealtimeParityFindings(report, device)
	if report.Classification == "blocked" {
		report.AcceptanceStatus = "blocked"
	}
	return report, nil
}

func validateXiaozhiRealtimeParityTrace(options xiaozhiRealtimeParityOptions, trace gateway.TraceResponse) error {
	if trace.TraceID != options.TraceID {
		return fmt.Errorf("trace id mismatch")
	}
	for _, event := range trace.Events {
		if event.TraceID != "" && event.TraceID != options.TraceID {
			return fmt.Errorf("trace event id mismatch")
		}
		if event.SessionID != "" && event.SessionID != options.SessionID {
			return fmt.Errorf("trace session id mismatch")
		}
		if event.DeviceID != "" && event.DeviceID != options.DeviceID {
			return fmt.Errorf("trace device id mismatch")
		}
		if xiaozhiPhysicalUnsafeString(event.Name) {
			return fmt.Errorf("trace contains unsafe event")
		}
	}
	return nil
}

func xiaozhiRealtimeParityCountEvents(events []gateway.TraceEvent) xiaozhiRealtimeParityCounts {
	counts := xiaozhiRealtimeParityCounts{TraceEvents: len(events)}
	for _, event := range events {
		switch event.Name {
		case "xiaozhi.listen.start":
			counts.ListenStart++
		case "xiaozhi.listen.auto_stop":
			counts.ListenAutoStop++
		case "xiaozhi.opus_frame.received":
			counts.OpusFramesReceived++
		case "xiaozhi.opus_frame.decoded":
			counts.OpusFramesDecoded++
		case "audio.ingress.buffered":
			counts.PCMIngressFrames++
		case "vad.speech.start":
			counts.VADSpeechStart++
		case "vad.speech.end":
			counts.VADSpeechEnd++
		case "asr.first_partial":
			counts.ASRFirstPartial++
		case "asr.final":
			counts.ASRFinal++
		case "asr.stream.start":
			counts.ASRStreamStart++
		case "asr.audio.append":
			counts.ASRAudioAppend++
		case "asr.stream.commit":
			counts.ASRStreamCommit++
		case "provider.first_content":
			counts.ProviderFirstContent++
		case "tts.first_audio":
			counts.TTSFirstAudio++
		case "audio.downlink.first_frame":
			counts.AudioDownlinkFirstFrame++
		case "xiaozhi.tts.opus_frame.downlink":
			counts.OpusDownlinkFrames++
		case "xiaozhi.voice_pipeline.answer.downlink":
			counts.AnswerDownlinkFrames++
		case "xiaozhi.voice_pipeline.start":
			counts.VoicePipelineStart++
		case "xiaozhi.voice_pipeline.completed":
			counts.VoicePipelineCompleted++
		case "device.playback.start":
			counts.DevicePlaybackStart++
		case "xiaozhi.voice_pipeline.asr.real_streaming":
			counts.ASRRealStreamingProfile++
		case "xiaozhi.voice_pipeline.llm.real_streaming":
			counts.LLMRealStreamingProfile++
		case "xiaozhi.voice_pipeline.tts.real_streaming":
			counts.TTSRealStreamingProfile++
		}
		if xiaozhiRealtimeParityProfileBlockedEvent(event.Name) {
			counts.RealtimeProfileBlocked++
		}
		if xiaozhiRealtimeParityForbiddenFakeEvent(event.Name) {
			counts.ForbiddenFakePath++
		}
	}
	return counts
}

func xiaozhiRealtimeParityOrderingFromTrace(events []gateway.TraceEvent) xiaozhiRealtimeParityOrdering {
	asrPartialAt, hasASRPartial := firstXiaozhiRealtimeParityEventAt(events, "asr.first_partial")
	asrFinalAt, hasASRFinal := firstXiaozhiRealtimeParityEventAt(events, "asr.final")
	providerAt, hasProvider := firstXiaozhiRealtimeParityEventAt(events, "provider.first_content")
	ttsAt, hasTTS := firstXiaozhiRealtimeParityEventAt(events, "tts.first_audio")
	downlinkAt, hasDownlink := firstXiaozhiRealtimeParityEventAt(events, "audio.downlink.first_frame")
	pipelineStartAt, hasPipelineStart := firstXiaozhiRealtimeParityEventAt(events, "xiaozhi.voice_pipeline.start")
	pipelineDoneAt, hasPipelineDone := firstXiaozhiRealtimeParityEventAt(events, "xiaozhi.voice_pipeline.completed")
	speechEndAt, hasSpeechEnd := firstXiaozhiRealtimeParityEventAt(events, "vad.speech.end")
	ingressAt, hasIngress := firstXiaozhiRealtimeParityEventAt(events, "audio.ingress.buffered")
	return xiaozhiRealtimeParityOrdering{
		StreamingASRBeforeSpeechEnd:       hasASRPartial && hasSpeechEnd && asrPartialAt < speechEndAt,
		ProviderBeforeASRFinal:            hasProvider && hasASRFinal && providerAt < asrFinalAt,
		TTSBeforePipelineCompleted:        hasTTS && hasPipelineDone && ttsAt < pipelineDoneAt,
		DownlinkBeforePipelineCompleted:   hasDownlink && hasPipelineDone && downlinkAt < pipelineDoneAt,
		PhysicalOpusIngressBeforePipeline: hasIngress && hasPipelineStart && ingressAt <= pipelineStartAt,
	}
}

func firstXiaozhiRealtimeParityEventAt(events []gateway.TraceEvent, name string) (int64, bool) {
	for _, event := range events {
		if event.Name == name {
			return event.AtMS, true
		}
	}
	return 0, false
}

func xiaozhiRealtimeParityForbiddenFakeEvent(name string) bool {
	switch name {
	case "xiaozhi.say.start",
		"xiaozhi.say.wav_loaded",
		"xiaozhi.say.wav_playback",
		"xiaozhi.voice_bench.host_loopback",
		"xiaozhi.voice_bench.start",
		"local_voice_loopback.start",
		"host_loopback.voice_pipeline.start",
		"answer.first_audio.host_loopback",
		"fast_companion.voice_pipeline.start",
		"fast_companion.voice_pipeline.completed",
		"control.local_fallback.sent",
		"xiaozhi.local_fallback.sent":
		return true
	default:
		return false
	}
}

func xiaozhiRealtimeParityProfileBlockedEvent(name string) bool {
	switch name {
	case "xiaozhi.voice_pipeline.asr.mock_blocked",
		"xiaozhi.voice_pipeline.asr.batch_blocked",
		"xiaozhi.voice_pipeline.llm.mock_blocked",
		"xiaozhi.voice_pipeline.tts.mock_blocked",
		"xiaozhi.voice_pipeline.tts.file_boundary_blocked":
		return true
	default:
		return false
	}
}

func xiaozhiRealtimeParityStageAvailability(report xiaozhiRealtimeParityReport, device firmwarecheck.DeviceIdentityRecord, trace gateway.TraceResponse) map[string]physicalStackChanMetric {
	return map[string]physicalStackChanMetric{
		"physical_device.online":             xiaozhiPhysicalBoolMetric(report.PhysicalDeviceOnline, "gateway_device_registry"),
		"device_id.physical":                 xiaozhiPhysicalBoolMetric(xiaozhiRealtimeParityPhysicalDeviceID(device.DeviceID), "gateway_device_registry"),
		"xiaozhi.transport.websocket":        xiaozhiPhysicalBoolMetric(report.Transport == "websocket", "gateway_device_registry"),
		"xiaozhi.profile.stock_or_debug":     xiaozhiPhysicalBoolMetric(report.Profile == "stock" || report.Profile == "debug", "gateway_device_registry"),
		"xiaozhi.listen.start":               xiaozhiPhysicalBoolMetric(report.Counts.ListenStart > 0, "gateway_trace"),
		"xiaozhi.opus.ingress":               xiaozhiPhysicalBoolMetric(report.Counts.OpusFramesDecoded > 0 && report.Counts.PCMIngressFrames > 0, "gateway_trace"),
		"vad.speech.end":                     xiaozhiPhysicalBoolMetric(report.Counts.VADSpeechEnd > 0, "gateway_trace"),
		"asr.partial_before_speech_end":      xiaozhiPhysicalBoolMetric(report.Ordering.StreamingASRBeforeSpeechEnd, "gateway_trace"),
		"asr.stream.start":                   xiaozhiPhysicalBoolMetric(report.Counts.ASRStreamStart > 0, "gateway_trace"),
		"asr.audio.append":                   xiaozhiPhysicalBoolMetric(report.Counts.ASRAudioAppend > 0, "gateway_trace"),
		"asr.stream.commit":                  xiaozhiPhysicalBoolMetric(report.Counts.ASRStreamCommit > 0, "gateway_trace"),
		"asr.final":                          xiaozhiPhysicalBoolMetric(report.Counts.ASRFinal > 0, "gateway_trace"),
		"llm.provider.first_content":         xiaozhiPhysicalBoolMetric(report.Counts.ProviderFirstContent > 0, "gateway_trace"),
		"llm.provider_before_asr_final":      xiaozhiPhysicalBoolMetric(report.Ordering.ProviderBeforeASRFinal, "gateway_trace"),
		"tts.first_audio":                    xiaozhiPhysicalBoolMetric(report.Counts.TTSFirstAudio > 0, "gateway_trace"),
		"tts.before_pipeline_completed":      xiaozhiPhysicalBoolMetric(report.Ordering.TTSBeforePipelineCompleted, "gateway_trace"),
		"opus.downlink.first_frame":          xiaozhiPhysicalBoolMetric(report.Counts.AudioDownlinkFirstFrame > 0 && report.Counts.OpusDownlinkFrames > 0, "gateway_trace"),
		"voice_pipeline.answer.downlink":     xiaozhiPhysicalBoolMetric(report.Counts.AnswerDownlinkFrames > 0, "gateway_trace"),
		"downlink.before_pipeline_completed": xiaozhiPhysicalBoolMetric(report.Ordering.DownlinkBeforePipelineCompleted, "gateway_trace"),
		"voice_pipeline.completed":           xiaozhiPhysicalBoolMetric(report.Counts.VoicePipelineCompleted > 0, "gateway_trace"),
		"asr.real_streaming_profile":         xiaozhiPhysicalBoolMetric(report.Counts.ASRRealStreamingProfile > 0, "gateway_trace"),
		"llm.real_streaming_profile":         xiaozhiPhysicalBoolMetric(report.Counts.LLMRealStreamingProfile > 0, "gateway_trace"),
		"tts.real_streaming_profile":         xiaozhiPhysicalBoolMetric(report.Counts.TTSRealStreamingProfile > 0, "gateway_trace"),
		"realtime_profile.blocker_absent":    xiaozhiPhysicalBoolMetric(report.Counts.RealtimeProfileBlocked == 0, "gateway_trace"),
		"fake_path.absent":                   xiaozhiPhysicalBoolMetric(report.Counts.ForbiddenFakePath == 0, "gateway_trace"),
		"device.playback.start":              xiaozhiPhysicalBoolMetric(trace.Summary.DevicePlaybackStartMS != nil || report.Counts.DevicePlaybackStart > 0, "gateway_trace"),
	}
}

func xiaozhiRealtimeParityClassification(report xiaozhiRealtimeParityReport) string {
	if !report.StageAvailability["fake_path.absent"].Available ||
		!report.StageAvailability["physical_device.online"].Available ||
		!report.StageAvailability["device_id.physical"].Available ||
		!report.StageAvailability["xiaozhi.transport.websocket"].Available ||
		!report.StageAvailability["xiaozhi.opus.ingress"].Available {
		return "blocked"
	}
	hasTurnCandidate := report.StageAvailability["vad.speech.end"].Available &&
		report.StageAvailability["asr.final"].Available &&
		report.StageAvailability["llm.provider.first_content"].Available &&
		report.StageAvailability["tts.first_audio"].Available &&
		report.StageAvailability["opus.downlink.first_frame"].Available &&
		report.StageAvailability["voice_pipeline.completed"].Available
	if !hasTurnCandidate {
		return "stock_opus_transport_only"
	}
	hasRealtime := report.StageAvailability["asr.partial_before_speech_end"].Available &&
		report.StageAvailability["asr.stream.start"].Available &&
		report.StageAvailability["asr.audio.append"].Available &&
		report.StageAvailability["asr.stream.commit"].Available &&
		report.StageAvailability["llm.provider_before_asr_final"].Available &&
		report.StageAvailability["tts.before_pipeline_completed"].Available &&
		report.StageAvailability["voice_pipeline.answer.downlink"].Available &&
		report.StageAvailability["downlink.before_pipeline_completed"].Available &&
		report.StageAvailability["asr.real_streaming_profile"].Available &&
		report.StageAvailability["llm.real_streaming_profile"].Available &&
		report.StageAvailability["tts.real_streaming_profile"].Available &&
		report.StageAvailability["realtime_profile.blocker_absent"].Available
	if hasRealtime {
		return "xiaozhi_realtime_candidate"
	}
	return "turn_buffered_xiaozhi_candidate"
}

func xiaozhiRealtimeParityCommandAccepted(classification string) bool {
	switch classification {
	case "turn_buffered_xiaozhi_candidate", "xiaozhi_realtime_candidate":
		return true
	default:
		return false
	}
}

func xiaozhiRealtimeParityFindings(report xiaozhiRealtimeParityReport, device firmwarecheck.DeviceIdentityRecord) []physicalStackChanEvidenceFinding {
	var findings []physicalStackChanEvidenceFinding
	if report.Counts.ForbiddenFakePath > 0 {
		findings = append(findings, physicalStackChanFinding("xiaozhi_realtime_fake_path_detected", "error", "trace contains /say, fast-companion, or local-fallback markers and cannot prove realtime Xiaozhi parity"))
	}
	if !xiaozhiRealtimeParityPhysicalDeviceID(device.DeviceID) {
		findings = append(findings, physicalStackChanFinding("xiaozhi_realtime_not_physical_device", "error", "device id looks virtual, simulator, or host-loopback"))
	}
	for _, stage := range []struct {
		key  string
		code string
		msg  string
	}{
		{"physical_device.online", "xiaozhi_realtime_device_offline", "physical device is not online"},
		{"xiaozhi.transport.websocket", "xiaozhi_realtime_transport_missing", "stock Xiaozhi WebSocket transport evidence is missing"},
		{"xiaozhi.opus.ingress", "xiaozhi_realtime_opus_ingress_missing", "real Opus ingress and PCM decode evidence are missing"},
		{"vad.speech.end", "xiaozhi_realtime_vad_speech_end_missing", "speech-end evidence is missing"},
		{"asr.stream.commit", "xiaozhi_realtime_stream_commit_missing", "ASR stream commit evidence is missing"},
		{"asr.final", "xiaozhi_realtime_asr_final_missing", "ASR final evidence is missing"},
		{"llm.provider.first_content", "xiaozhi_realtime_provider_first_content_missing", "LLM/text-stream first content evidence is missing"},
		{"tts.first_audio", "xiaozhi_realtime_tts_first_audio_missing", "TTS first audio evidence is missing"},
		{"opus.downlink.first_frame", "xiaozhi_realtime_downlink_missing", "Opus downlink first-frame evidence is missing"},
		{"voice_pipeline.completed", "xiaozhi_realtime_pipeline_completed_missing", "voice pipeline completion evidence is missing"},
	} {
		if !report.StageAvailability[stage.key].Available {
			findings = append(findings, physicalStackChanFinding(stage.code, "error", stage.msg))
		}
	}
	if report.Classification == "turn_buffered_xiaozhi_candidate" {
		findings = append(findings, physicalStackChanFinding("xiaozhi_realtime_turn_buffered", "warning", "physical stock Opus path reached downlink, but ASR/LLM/TTS ordering still looks turn-buffered rather than fully Xiaozhi-style streaming"))
	}
	if report.Counts.RealtimeProfileBlocked > 0 ||
		!report.StageAvailability["asr.real_streaming_profile"].Available ||
		!report.StageAvailability["llm.real_streaming_profile"].Available ||
		!report.StageAvailability["tts.real_streaming_profile"].Available {
		findings = append(findings, physicalStackChanFinding("xiaozhi_realtime_real_profile_evidence_missing", "error", "trace is missing real streaming ASR/LLM/TTS profile evidence or contains mock/file-boundary profile blockers"))
	}
	if report.Classification == "xiaozhi_realtime_candidate" {
		findings = append(findings, physicalStackChanFinding("xiaozhi_realtime_candidate_not_product_accepted", "info", "trace ordering matches realtime candidate criteria, but wake, audible playback, interruption, and setup-free physical acceptance are still separate gates"))
	}
	if !report.StageAvailability["device.playback.start"].Available {
		findings = append(findings, physicalStackChanFinding("xiaozhi_realtime_device_playback_unproven", "warning", "device playback start was not observed in Gateway/runtime trace"))
	}
	return findings
}

func xiaozhiRealtimeParityPhysicalDeviceID(deviceID string) bool {
	lower := strings.ToLower(strings.TrimSpace(deviceID))
	if lower == "" {
		return false
	}
	for _, marker := range []string{"virtual", "sim", "bench", "loopback", "fixture", "mock"} {
		if strings.Contains(lower, marker) {
			return false
		}
	}
	return true
}

func writeXiaozhiRealtimeParityReport(outputDir string, report xiaozhiRealtimeParityReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-xiaozhi-realtime-parity-"+time.Now().Format("20060102-150405.000000000")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONXiaozhiRealtimeParity(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONXiaozhiRealtimeParity(writer io.Writer, report xiaozhiRealtimeParityReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
