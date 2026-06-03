package providers

import (
	"a21.local/a21/internal/audio"
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"
)

var ErrVoicePipelineBargeIn = errors.New("a21 voice pipeline barge-in")

type VoicePipelineStatus string

const (
	VoicePipelineStatusCompleted VoicePipelineStatus = "completed"
	VoicePipelineStatusCancelled VoicePipelineStatus = "cancelled"
	VoicePipelineStatusFailed    VoicePipelineStatus = "failed"
)

type VoicePipelinePCMFrame struct {
	Seq          uint64
	Codec        string
	SampleRateHz int
	Channels     int
	DurationMS   int
	ByteCount    int
	RMS          float64
	PCM16LE      []byte `json:"-"`
}

type VoicePipelineRequest struct {
	Session             VoiceSession
	Mode                string
	Frames              []VoicePipelinePCMFrame
	ASRTranscript       string `json:"-"`
	ASRTranscriptSource string `json:"-"`
}

const (
	VoicePipelineASRTranscriptSourceFinal   = "streaming_final"
	VoicePipelineASRTranscriptSourcePartial = "streaming_partial"
)

type VoicePipelineSelection struct {
	ASRMode               string `json:"asr_mode"`
	ASRProfile            string `json:"asr_profile"`
	ASRProfileEnv         string `json:"asr_profile_env"`
	LLMProfile            string `json:"llm_profile"`
	LLMProfileEnv         string `json:"llm_profile_env"`
	LLMFallbackProfile    string `json:"llm_fallback_profile,omitempty"`
	LLMFallbackProfileEnv string `json:"llm_fallback_profile_env,omitempty"`
	TTSMode               string `json:"tts_mode"`
	TTSProfile            string `json:"tts_profile"`
	TTSProfileEnv         string `json:"tts_profile_env"`
}

type VoicePipelineTiming struct {
	ASRFirstPartialMS       int64 `json:"asr_first_partial_ms"`
	ASRFinalMS              int64 `json:"asr_final_ms"`
	LLMFirstContentMS       int64 `json:"llm_first_content_ms"`
	TTSFirstAudioMS         int64 `json:"tts_first_audio_ms"`
	AudioDownlinkFirstMS    int64 `json:"audio_downlink_first_frame_ms"`
	ProviderCancelMS        int64 `json:"provider_cancel_ms,omitempty"`
	BargeInStopMS           int64 `json:"barge_in_stop_ms,omitempty"`
	SpeechEndToFinalASRMS   int64 `json:"speech_end_to_final_asr_ms"`
	SpeechEndToFirstTokenMS int64 `json:"speech_end_to_first_llm_token_ms"`
}

type VoicePipelineResult struct {
	Status       VoicePipelineStatus
	CancelReason VoiceCancelReason
	Timing       VoicePipelineTiming
	AudioChunks  []VoiceAudioChunk
	Report       VoicePipelineReport
}

type VoicePipelineReport struct {
	SchemaVersion string                         `json:"schema_version"`
	Status        string                         `json:"status"`
	TraceID       string                         `json:"trace_id"`
	SessionID     string                         `json:"session_id"`
	DeviceID      string                         `json:"device_id"`
	Mode          string                         `json:"mode"`
	ExecutionMode string                         `json:"execution_mode"`
	Selection     VoicePipelineSelection         `json:"selection"`
	Input         VoicePipelineInputReport       `json:"input"`
	Output        VoicePipelineOutputReport      `json:"output"`
	Fallback      *VoicePipelineFallbackReport   `json:"fallback,omitempty"`
	Timing        VoicePipelineTiming            `json:"timing"`
	Redaction     VoicePipelineRedactionPolicies `json:"redaction"`
	Findings      []string                       `json:"findings,omitempty"`
}

type VoicePipelineFallbackReport struct {
	Activated bool   `json:"activated"`
	Provider  string `json:"provider"`
	Reason    string `json:"reason"`
}

type VoicePipelineInputReport struct {
	FrameCount      int    `json:"frame_count"`
	TotalBytes      int    `json:"total_bytes"`
	Codec           string `json:"codec,omitempty"`
	SampleRateHz    int    `json:"sample_rate_hz,omitempty"`
	Channels        int    `json:"channels,omitempty"`
	FrameDurationMS int    `json:"frame_duration_ms,omitempty"`
}

type VoicePipelineOutputReport struct {
	AudioChunkCount int                     `json:"audio_chunk_count"`
	AudioCodec      string                  `json:"audio_codec,omitempty"`
	SampleRateHz    int                     `json:"sample_rate_hz,omitempty"`
	Channels        int                     `json:"channels,omitempty"`
	ChunkDurationMS int                     `json:"chunk_duration_ms,omitempty"`
	AudioQuality    *audio.PCMQualityReport `json:"audio_quality,omitempty"`
	ASRTextChars    int                     `json:"asr_text_chars,omitempty"`
	LLMContentChars int                     `json:"llm_content_chars,omitempty"`
	LLMSegmentCount int                     `json:"llm_segment_count,omitempty"`
	StreamingAnswer bool                    `json:"streaming_answer,omitempty"`
}

type VoicePipelineRedactionPolicies struct {
	TranscriptPolicy     string `json:"transcript_policy"`
	ProviderOutputPolicy string `json:"provider_output_policy"`
	AudioPayloadPolicy   string `json:"audio_payload_policy"`
	URLPolicy            string `json:"url_policy"`
	ProxyPolicy          string `json:"proxy_policy"`
	LocalPathPolicy      string `json:"local_path_policy"`
}

type ASRAdapter interface {
	Name() string
	Transcribe(ctx context.Context, req ASRAdapterRequest) (<-chan ASRAdapterEvent, error)
}

type StreamingASRAdapter interface {
	Name() string
	StartStreamingASR(ctx context.Context, req StreamingASRStartRequest) (StreamingASRSession, error)
}

type StreamingASRStartRequest struct {
	Session VoiceSession
	Mode    string
}

type StreamingASRSession interface {
	AppendFrame(ctx context.Context, frame VoicePipelinePCMFrame) error
	Events() <-chan ASRAdapterEvent
	Commit(ctx context.Context) error
	Cancel(error)
}

type ASRAdapterRequest struct {
	Session VoiceSession
	Mode    string
	Frames  []VoicePipelinePCMFrame
}

type ASRAdapterEvent struct {
	Text    string
	Final   bool
	Finding string
	Err     error
}

type TextStreamAdapter interface {
	Name() string
	StreamText(ctx context.Context, req TextStreamAdapterRequest) (<-chan TextStreamEvent, error)
}

type TextStreamAdapterRequest struct {
	Session VoiceSession
	Mode    string
	Text    string
}

type TTSAdapter interface {
	Name() string
	Synthesize(ctx context.Context, req TTSAdapterRequest) (<-chan VoiceAudioChunk, error)
}

type StreamingTTSAdapter interface {
	TTSAdapter
	StreamingTTSAdapter()
}

type TTSAdapterRequest struct {
	Session VoiceSession
	Mode    string
	Text    string
}

type VoicePipelineAdapters struct {
	ASR           ASRAdapter
	TextStream    TextStreamAdapter
	TTS           TTSAdapter
	Selection     VoicePipelineSelection
	ExecutionMode string
}

type VoicePipelineRunner struct {
	adapters VoicePipelineAdapters
}

type VoicePipelineStreamEventKind string

const (
	VoicePipelineStreamAudioChunk VoicePipelineStreamEventKind = "audio_chunk"
	VoicePipelineStreamDone       VoicePipelineStreamEventKind = "done"
)

type VoicePipelineStreamEvent struct {
	Kind       VoicePipelineStreamEventKind
	AudioChunk VoiceAudioChunk
	SegmentSeq int
	Timing     VoicePipelineTiming
	Report     VoicePipelineReport
	Result     VoicePipelineResult
	Err        error
}

func NewVoicePipelineRunner(adapters VoicePipelineAdapters) *VoicePipelineRunner {
	if isZeroVoicePipelineSelection(adapters.Selection) {
		adapters.Selection = VoicePipelineSelectionFromEnv(nil)
	}
	return &VoicePipelineRunner{adapters: adapters}
}

func (r *VoicePipelineRunner) Run(ctx context.Context, req VoicePipelineRequest) (VoicePipelineResult, error) {
	return r.run(ctx, req, nil)
}

func (r *VoicePipelineRunner) RunStream(ctx context.Context, req VoicePipelineRequest) (<-chan VoicePipelineStreamEvent, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	events := make(chan VoicePipelineStreamEvent, 4)
	go func() {
		defer close(events)
		result, err := r.run(ctx, req, func(chunk VoiceAudioChunk, segmentSeq int, timing VoicePipelineTiming, report VoicePipelineReport) bool {
			select {
			case <-ctx.Done():
				return false
			case events <- VoicePipelineStreamEvent{Kind: VoicePipelineStreamAudioChunk, AudioChunk: chunk, SegmentSeq: segmentSeq, Timing: timing, Report: report}:
				return true
			}
		})
		emitVoicePipelineStreamDone(ctx, events, VoicePipelineStreamEvent{Kind: VoicePipelineStreamDone, Result: result, Err: err})
	}()
	return events, nil
}

func emitVoicePipelineStreamDone(ctx context.Context, out chan<- VoicePipelineStreamEvent, event VoicePipelineStreamEvent) {
	select {
	case <-ctx.Done():
	case out <- event:
	}
}

func (r *VoicePipelineRunner) run(ctx context.Context, req VoicePipelineRequest, emit func(VoiceAudioChunk, int, VoicePipelineTiming, VoicePipelineReport) bool) (VoicePipelineResult, error) {
	start := time.Now()
	result := VoicePipelineResult{
		Status: VoicePipelineStatusFailed,
		Timing: VoicePipelineTiming{
			ASRFirstPartialMS:       -1,
			ASRFinalMS:              -1,
			LLMFirstContentMS:       -1,
			TTSFirstAudioMS:         -1,
			AudioDownlinkFirstMS:    -1,
			ProviderCancelMS:        -1,
			BargeInStopMS:           -1,
			SpeechEndToFinalASRMS:   -1,
			SpeechEndToFirstTokenMS: -1,
		},
	}
	report := newVoicePipelineReport(req, r.adapters.Selection, r.adapters.ExecutionMode)
	if err := validateVoicePipelineAdapters(r.adapters); err != nil {
		report.Status = string(VoicePipelineStatusFailed)
		report.Findings = append(report.Findings, err.Error())
		result.Report = report
		return result, err
	}
	if cancelled := r.applyCancel(ctx, start, &result); cancelled {
		result.Report = finalizeVoicePipelineReport(report, result)
		return result, nil
	}

	var transcript string
	if strings.TrimSpace(req.ASRTranscript) != "" {
		transcript = req.ASRTranscript
		switch voicePipelineASRTranscriptSource(req.ASRTranscriptSource) {
		case VoicePipelineASRTranscriptSourcePartial:
			result.Timing.ASRFirstPartialMS = 0
			report.Findings = append(report.Findings, "streaming_asr_partial_reused")
		default:
			result.Timing.ASRFirstPartialMS = 0
			result.Timing.ASRFinalMS = 0
			result.Timing.SpeechEndToFinalASRMS = 0
			report.Findings = append(report.Findings, "streaming_asr_final_reused")
		}
	} else {
		asrEvents, err := r.adapters.ASR.Transcribe(ctx, ASRAdapterRequest{Session: req.Session, Mode: req.Mode, Frames: req.Frames})
		if err != nil {
			report.Status = string(VoicePipelineStatusFailed)
			report.Findings = append(report.Findings, "asr adapter failed")
			result.Report = report
			return result, err
		}
		for event := range asrEvents {
			if event.Finding != "" {
				report.Findings = append(report.Findings, event.Finding)
			}
			if event.Err != nil {
				report.Status = string(VoicePipelineStatusFailed)
				if event.Finding == "" {
					report.Findings = append(report.Findings, "asr adapter failed")
				}
				result.Report = report
				return result, event.Err
			}
			if result.Timing.ASRFirstPartialMS < 0 && strings.TrimSpace(event.Text) != "" {
				result.Timing.ASRFirstPartialMS = pipelineElapsedMS(start)
			}
			if event.Final {
				transcript = event.Text
				result.Timing.ASRFinalMS = pipelineElapsedMS(start)
				result.Timing.SpeechEndToFinalASRMS = result.Timing.ASRFinalMS
			}
		}
	}
	if cancelled := r.applyCancel(ctx, start, &result); cancelled {
		result.Report = finalizeVoicePipelineReport(report, result)
		return result, nil
	}

	textEvents, err := r.adapters.TextStream.StreamText(ctx, TextStreamAdapterRequest{Session: req.Session, Mode: req.Mode, Text: transcript})
	if err != nil {
		report.Status = string(VoicePipelineStatusFailed)
		report.Findings = append(report.Findings, "text stream adapter failed")
		result.Report = report
		return result, err
	}
	var response strings.Builder
	var segments voicePipelineTextSegmenter
	segmentSeq := 0
	for event := range textEvents {
		if cancelled := r.applyCancel(ctx, start, &result); cancelled {
			result.Report = finalizeVoicePipelineReport(report, result)
			return result, nil
		}
		if event.Finding != "" {
			report.Findings = append(report.Findings, event.Finding)
		}
		if event.Fallback != nil && event.Fallback.Activated {
			report.Fallback = &VoicePipelineFallbackReport{
				Activated: true,
				Provider:  safeVoicePipelineProfileName(event.Fallback.Provider),
				Reason:    sanitizeVoicePipelineValue(event.Fallback.Reason, "primary_failed"),
			}
			if !voicePipelineStringSliceHas(report.Findings, "provider_fallback_used") {
				report.Findings = append(report.Findings, "provider_fallback_used")
			}
		}
		if event.Err != nil {
			report.Status = string(VoicePipelineStatusFailed)
			if event.Finding == "" {
				report.Findings = append(report.Findings, "text stream adapter failed")
			}
			result.Report = report
			return result, event.Err
		}
		if event.Kind == TextStreamDeltaDone {
			break
		}
		if event.Kind != TextStreamDeltaContent {
			continue
		}
		if result.Timing.LLMFirstContentMS < 0 && strings.TrimSpace(event.Text) != "" {
			result.Timing.LLMFirstContentMS = pipelineElapsedMS(start)
			result.Timing.SpeechEndToFirstTokenMS = result.Timing.LLMFirstContentMS
		}
		response.WriteString(event.Text)
		if emit != nil {
			for _, segment := range segments.Append(event.Text) {
				segmentSeq++
				if err := r.synthesizeVoicePipelineSegment(ctx, start, req, segment, segmentSeq, emit, &result, &report); err != nil {
					return result, err
				}
			}
		}
	}
	if cancelled := r.applyCancel(ctx, start, &result); cancelled {
		result.Report = finalizeVoicePipelineReport(report, result)
		return result, nil
	}
	if emit == nil {
		if err := r.synthesizeVoicePipelineSegment(ctx, start, req, response.String(), 1, nil, &result, &report); err != nil {
			return result, err
		}
	} else {
		for _, segment := range segments.Flush() {
			segmentSeq++
			if err := r.synthesizeVoicePipelineSegment(ctx, start, req, segment, segmentSeq, emit, &result, &report); err != nil {
				return result, err
			}
		}
	}
	if cancelled := r.applyCancel(ctx, start, &result); cancelled {
		result.Report = finalizeVoicePipelineReport(report, result)
		return result, nil
	}

	result.Status = VoicePipelineStatusCompleted
	report.Output.ASRTextChars = len([]rune(transcript))
	report.Output.LLMContentChars = len([]rune(response.String()))
	report.Output.StreamingAnswer = emit != nil
	if emit != nil {
		report.Output.LLMSegmentCount = segmentSeq
	}
	result.Report = finalizeVoicePipelineReport(report, result)
	return result, nil
}

func voicePipelineASRTranscriptSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case VoicePipelineASRTranscriptSourcePartial:
		return VoicePipelineASRTranscriptSourcePartial
	default:
		return VoicePipelineASRTranscriptSourceFinal
	}
}

func (r *VoicePipelineRunner) synthesizeVoicePipelineSegment(ctx context.Context, start time.Time, req VoicePipelineRequest, segment string, segmentSeq int, emit func(VoiceAudioChunk, int, VoicePipelineTiming, VoicePipelineReport) bool, result *VoicePipelineResult, report *VoicePipelineReport) error {
	if strings.TrimSpace(segment) == "" {
		return nil
	}
	ttsChunks, err := r.adapters.TTS.Synthesize(ctx, TTSAdapterRequest{Session: req.Session, Mode: req.Mode, Text: segment})
	if err != nil {
		report.Status = string(VoicePipelineStatusFailed)
		report.Findings = appendUniqueVoicePipelineFindings(report.Findings, "tts adapter failed", voicePipelineTTSAdapterFailureFinding(err))
		result.Report = *report
		return err
	}
	for chunk := range ttsChunks {
		if chunk.Finding != "" {
			report.Findings = appendUniqueVoicePipelineFindings(report.Findings, chunk.Finding)
		}
		if chunk.Err != nil {
			report.Status = string(VoicePipelineStatusFailed)
			if chunk.Finding == "" {
				report.Findings = appendUniqueVoicePipelineFindings(report.Findings, "tts adapter failed")
			}
			result.Report = *report
			return chunk.Err
		}
		if cancelled := r.applyCancel(ctx, start, result); cancelled {
			result.Report = finalizeVoicePipelineReport(*report, *result)
			return nil
		}
		if result.Timing.TTSFirstAudioMS < 0 {
			result.Timing.TTSFirstAudioMS = pipelineElapsedMS(start)
			result.Timing.AudioDownlinkFirstMS = result.Timing.TTSFirstAudioMS
		}
		result.AudioChunks = append(result.AudioChunks, chunk)
		if emit != nil && !emit(chunk, segmentSeq, result.Timing, *report) {
			r.applyCancel(ctx, start, result)
			result.Report = finalizeVoicePipelineReport(*report, *result)
			return nil
		}
	}
	return nil
}

func voicePipelineTTSAdapterFailureFinding(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "missing"):
		return "tts adapter missing configuration"
	case strings.Contains(message, "dial") || strings.Contains(message, "connect"):
		return "tts adapter connect failed"
	case strings.Contains(message, "session update"):
		return "tts adapter session update failed"
	case strings.Contains(message, "text append"):
		return "tts adapter text append failed"
	case strings.Contains(message, "text commit"):
		return "tts adapter text commit failed"
	case strings.Contains(message, "session finish"):
		return "tts adapter session finish failed"
	default:
		return "tts adapter failed"
	}
}

func (r *VoicePipelineRunner) applyCancel(ctx context.Context, start time.Time, result *VoicePipelineResult) bool {
	if ctx.Err() == nil {
		return false
	}
	result.Status = VoicePipelineStatusCancelled
	result.Timing.ProviderCancelMS = pipelineElapsedMS(start)
	result.Timing.BargeInStopMS = result.Timing.ProviderCancelMS
	if errors.Is(context.Cause(ctx), ErrVoicePipelineBargeIn) {
		result.CancelReason = CancelBargeIn
	} else {
		result.CancelReason = CancelError
	}
	result.AudioChunks = nil
	return true
}

func validateVoicePipelineAdapters(adapters VoicePipelineAdapters) error {
	switch {
	case adapters.ASR == nil:
		return errors.New("a21 voice pipeline ASR adapter is required")
	case adapters.TextStream == nil:
		return errors.New("a21 voice pipeline text stream adapter is required")
	case adapters.TTS == nil:
		return errors.New("a21 voice pipeline TTS adapter is required")
	default:
		return nil
	}
}

func VoicePipelineSelectionFromEnv(env []string) VoicePipelineSelection {
	asrMode := sanitizeVoicePipelineValue(envValue(env, "A21_ASR_PROFILE"), "local")
	asrEnv := "A21_ASR_LOCAL_PROFILE"
	if asrMode == "cloud" {
		asrEnv = "A21_ASR_CLOUD_PROFILE"
	}
	asrProfile := sanitizeVoicePipelineValue(envValue(env, asrEnv), "mock-local-asr")

	llmEnv := "A21_TEXT_STREAM_PROFILE"
	llmProfile := sanitizeVoicePipelineValue(envValue(env, llmEnv), "")
	if llmProfile == "" {
		llmEnv = "A21_PROVIDER_PRIMARY"
		llmProfile = sanitizeVoicePipelineValue(envValue(env, llmEnv), "mock")
	}
	llmFallbackEnv := ""
	llmFallbackProfile := sanitizeVoicePipelineValue(envValue(env, "A21_TEXT_STREAM_FALLBACK_PROFILE"), "")
	if llmFallbackProfile != "" {
		llmFallbackEnv = "A21_TEXT_STREAM_FALLBACK_PROFILE"
	}

	ttsMode := sanitizeVoicePipelineValue(envValue(env, "A21_TTS_MODE"), "fast")
	ttsEnv := "A21_TTS_FAST_PROFILE"
	switch ttsMode {
	case "quality":
		ttsEnv = "A21_TTS_QUALITY_PROFILE"
	case "balanced":
		ttsEnv = "A21_TTS_BALANCED_PROFILE"
	}
	ttsProfile := sanitizeVoicePipelineValue(envValue(env, ttsEnv), "mock-fast-tts")

	return VoicePipelineSelection{
		ASRMode:               asrMode,
		ASRProfile:            asrProfile,
		ASRProfileEnv:         asrEnv,
		LLMProfile:            llmProfile,
		LLMProfileEnv:         llmEnv,
		LLMFallbackProfile:    llmFallbackProfile,
		LLMFallbackProfileEnv: llmFallbackEnv,
		TTSMode:               ttsMode,
		TTSProfile:            ttsProfile,
		TTSProfileEnv:         ttsEnv,
	}
}

func sanitizeVoicePipelineValue(value string, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	if containsLegacyProviderIdentity(value) || containsBlockedProviderIdentity(value) {
		return fallback
	}
	return value
}

func voicePipelineStringSliceHas(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func safeVoicePipelineProfileName(value string) string {
	name := strings.ToLower(strings.TrimSpace(value))
	if containsLegacyProviderIdentity(name) {
		return "invalid_legacy_provider"
	}
	if containsBlockedProviderIdentity(name) {
		return "blocked_provider"
	}
	if knownProvider(name) || loadedProviderNamePattern.MatchString(name) {
		return name
	}
	return "unknown_provider"
}

func newVoicePipelineReport(req VoicePipelineRequest, selection VoicePipelineSelection, executionMode string) VoicePipelineReport {
	input := VoicePipelineInputReport{FrameCount: len(req.Frames)}
	for i, frame := range req.Frames {
		input.TotalBytes += frame.ByteCount
		if i == 0 {
			input.Codec = frame.Codec
			input.SampleRateHz = frame.SampleRateHz
			input.Channels = frame.Channels
			input.FrameDurationMS = frame.DurationMS
		}
	}
	executionMode = sanitizeVoicePipelineValue(executionMode, "fixture")
	schemaVersion := "a21.voice_pipeline.fixture.v1"
	var findings []string
	switch executionMode {
	case "host_local":
		schemaVersion = "a21.voice_pipeline.host_local.v1"
		findings = append(findings, "host_local_voice_pipeline_candidate_not_prd_accepted")
	case "cloud_edge":
		schemaVersion = "a21.voice_pipeline.cloud_edge.v1"
		findings = append(findings, "cloud_edge_voice_pipeline_candidate_not_prd_accepted")
	default:
		executionMode = "fixture"
	}
	return VoicePipelineReport{
		SchemaVersion: schemaVersion,
		Status:        string(VoicePipelineStatusFailed),
		TraceID:       req.Session.TraceID,
		SessionID:     req.Session.SessionID,
		DeviceID:      req.Session.DeviceID,
		Mode:          req.Mode,
		ExecutionMode: executionMode,
		Selection:     selection,
		Input:         input,
		Findings:      findings,
		Redaction: VoicePipelineRedactionPolicies{
			TranscriptPolicy:     "transcript_not_recorded",
			ProviderOutputPolicy: "provider_output_not_recorded",
			AudioPayloadPolicy:   "audio_payload_not_recorded",
			URLPolicy:            "full_url_not_recorded",
			ProxyPolicy:          "proxy_value_not_recorded",
			LocalPathPolicy:      "local_path_not_recorded",
		},
	}
}

func finalizeVoicePipelineReport(report VoicePipelineReport, result VoicePipelineResult) VoicePipelineReport {
	report.Status = string(result.Status)
	report.Timing = result.Timing
	report.Output.AudioChunkCount = len(result.AudioChunks)
	if len(result.AudioChunks) > 0 {
		first := result.AudioChunks[0]
		report.Output.AudioCodec = first.Codec
		report.Output.SampleRateHz = first.SampleRateHz
		report.Output.Channels = first.Channels
		report.Output.ChunkDurationMS = first.DurationMS
		report.Output.AudioQuality = voicePipelineAudioQuality(result.AudioChunks, &report)
	}
	return report
}

func voicePipelineAudioQuality(chunks []VoiceAudioChunk, report *VoicePipelineReport) *audio.PCMQualityReport {
	if len(chunks) == 0 {
		return nil
	}
	first := chunks[0]
	var pcm []byte
	for _, chunk := range chunks {
		if chunk.Codec != first.Codec || chunk.SampleRateHz != first.SampleRateHz || chunk.Channels != first.Channels || chunk.DurationMS != first.DurationMS {
			if report != nil {
				report.Findings = appendUniqueVoicePipelineFindings(report.Findings, "audio_quality_format_mismatch")
			}
			return &audio.PCMQualityReport{
				Status:       "warning",
				Codec:        first.Codec,
				SampleRateHz: first.SampleRateHz,
				Channels:     first.Channels,
				Findings:     []string{"audio_quality_format_mismatch"},
			}
		}
		data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(chunk.DataBase64))
		if err != nil {
			if report != nil {
				report.Findings = appendUniqueVoicePipelineFindings(report.Findings, "audio_quality_invalid_payload")
			}
			return &audio.PCMQualityReport{
				Status:       "failed",
				Codec:        first.Codec,
				SampleRateHz: first.SampleRateHz,
				Channels:     first.Channels,
				Findings:     []string{"audio_quality_invalid_payload"},
			}
		}
		pcm = append(pcm, data...)
	}
	quality, err := audio.AnalyzePCM16LEQuality(first.Codec, first.SampleRateHz, first.Channels, 0, pcm)
	if err != nil {
		if report != nil {
			report.Findings = appendUniqueVoicePipelineFindings(report.Findings, "audio_quality_unavailable")
		}
		return &audio.PCMQualityReport{
			Status:       "failed",
			Codec:        first.Codec,
			SampleRateHz: first.SampleRateHz,
			Channels:     first.Channels,
			Findings:     []string{"audio_quality_unavailable"},
		}
	}
	if report != nil {
		report.Findings = appendUniqueVoicePipelineFindings(report.Findings, quality.Findings...)
	}
	return &quality
}

func appendUniqueVoicePipelineFindings(findings []string, values ...string) []string {
	for _, value := range values {
		if strings.TrimSpace(value) == "" || voicePipelineStringSliceHas(findings, value) {
			continue
		}
		findings = append(findings, value)
	}
	return findings
}

const voicePipelineSegmentRuneThreshold = 40

type voicePipelineTextSegmenter struct {
	pending      strings.Builder
	pendingRunes int
	flushed      int
}

func (s *voicePipelineTextSegmenter) Append(text string) []string {
	var segments []string
	for _, r := range text {
		s.pending.WriteRune(r)
		s.pendingRunes++
		if isVoicePipelineSentenceBoundary(r) || s.pendingRunes >= voicePipelineSegmentRuneThreshold {
			segments = append(segments, s.flushOne())
		}
	}
	return nonEmptyVoicePipelineSegments(segments)
}

func (s *voicePipelineTextSegmenter) Flush() []string {
	if s.pendingRunes == 0 {
		return nil
	}
	return nonEmptyVoicePipelineSegments([]string{s.flushOne()})
}

func (s *voicePipelineTextSegmenter) FlushedCount() int {
	return s.flushed
}

func (s *voicePipelineTextSegmenter) flushOne() string {
	segment := strings.TrimSpace(s.pending.String())
	s.pending.Reset()
	s.pendingRunes = 0
	if segment != "" {
		s.flushed++
	}
	return segment
}

func nonEmptyVoicePipelineSegments(segments []string) []string {
	out := segments[:0]
	for _, segment := range segments {
		if strings.TrimSpace(segment) != "" {
			out = append(out, segment)
		}
	}
	return out
}

func isVoicePipelineSentenceBoundary(r rune) bool {
	switch r {
	case '。', '！', '？', '；', '.', '!', '?', ';', '\n':
		return true
	default:
		return false
	}
}

func pipelineElapsedMS(start time.Time) int64 {
	ms := time.Since(start).Milliseconds()
	if ms < 0 {
		return 0
	}
	return ms
}

func isZeroVoicePipelineSelection(selection VoicePipelineSelection) bool {
	return selection.ASRMode == "" &&
		selection.ASRProfile == "" &&
		selection.ASRProfileEnv == "" &&
		selection.LLMProfile == "" &&
		selection.LLMProfileEnv == "" &&
		selection.LLMFallbackProfile == "" &&
		selection.LLMFallbackProfileEnv == "" &&
		selection.TTSMode == "" &&
		selection.TTSProfile == "" &&
		selection.TTSProfileEnv == ""
}

type mockASRAdapter struct {
	name string
}

type mockStreamingASRAdapter struct {
	name string
}

type mockStreamingASRSession struct {
	mu          sync.Mutex
	events      chan ASRAdapterEvent
	partialSent bool
	finalSent   bool
	closed      bool
}

func NewMockASRAdapter(name string) ASRAdapter {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "mock-local-asr"
	}
	return mockASRAdapter{name: name}
}

func NewMockStreamingASRAdapter(name string) StreamingASRAdapter {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "mock-streaming-asr"
	}
	return mockStreamingASRAdapter{name: name}
}

func (a mockASRAdapter) Name() string {
	return a.name
}

func (a mockStreamingASRAdapter) Name() string {
	return a.name
}

func (a mockStreamingASRAdapter) Transcribe(ctx context.Context, req ASRAdapterRequest) (<-chan ASRAdapterEvent, error) {
	return mockASRAdapter{name: a.name}.Transcribe(ctx, req)
}

func (a mockASRAdapter) Transcribe(ctx context.Context, req ASRAdapterRequest) (<-chan ASRAdapterEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	events := make(chan ASRAdapterEvent, 2)
	events <- ASRAdapterEvent{Text: "fixture transcript should never be stored", Final: false}
	events <- ASRAdapterEvent{Text: "fixture transcript should never be stored", Final: true}
	close(events)
	return events, nil
}

func (a mockStreamingASRAdapter) StartStreamingASR(ctx context.Context, req StreamingASRStartRequest) (StreamingASRSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &mockStreamingASRSession{events: make(chan ASRAdapterEvent, 4)}, nil
}

func (s *mockStreamingASRSession) AppendFrame(ctx context.Context, frame VoicePipelinePCMFrame) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.partialSent || (frame.RMS <= 0 && frame.ByteCount <= 0) {
		return nil
	}
	s.partialSent = true
	return s.sendLocked(ctx, ASRAdapterEvent{Text: "fixture transcript should never be stored"})
}

func (s *mockStreamingASRSession) Events() <-chan ASRAdapterEvent {
	return s.events
}

func (s *mockStreamingASRSession) Commit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.finalSent {
		return nil
	}
	s.finalSent = true
	if err := s.sendLocked(ctx, ASRAdapterEvent{Text: "fixture transcript should never be stored", Final: true}); err != nil {
		return err
	}
	close(s.events)
	s.closed = true
	return nil
}

func (s *mockStreamingASRSession) Cancel(error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	close(s.events)
	s.closed = true
}

func (s *mockStreamingASRSession) sendLocked(ctx context.Context, event ASRAdapterEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.events <- event:
		return nil
	}
}

type mockTextStreamAdapter struct {
	name           string
	onFirstContent func()
}

func NewMockTextStreamAdapter(name string, onFirstContent ...func()) TextStreamAdapter {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "mock-text-stream"
	}
	var hook func()
	if len(onFirstContent) > 0 {
		hook = onFirstContent[0]
	}
	return mockTextStreamAdapter{name: name, onFirstContent: hook}
}

func (a mockTextStreamAdapter) Name() string {
	return a.name
}

func (a mockTextStreamAdapter) StreamText(ctx context.Context, req TextStreamAdapterRequest) (<-chan TextStreamEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	events := make(chan TextStreamEvent, 2)
	if a.onFirstContent != nil {
		a.onFirstContent()
	}
	events <- TextStreamEvent{Kind: TextStreamDeltaContent, Text: "mock provider output should never be stored"}
	events <- TextStreamEvent{Kind: TextStreamDeltaDone}
	close(events)
	return events, nil
}

type mockTTSAdapter struct {
	name string
}

func NewMockTTSAdapter(name string) TTSAdapter {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "mock-fast-tts"
	}
	return mockTTSAdapter{name: name}
}

func (a mockTTSAdapter) Name() string {
	return a.name
}

func (a mockTTSAdapter) Synthesize(ctx context.Context, req TTSAdapterRequest) (<-chan VoiceAudioChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	events := make(chan VoiceAudioChunk, 1)
	events <- VoiceAudioChunk{
		Codec:        "pcm_s16le",
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   base64.StdEncoding.EncodeToString(make([]byte, 2880)),
	}
	close(events)
	return events, nil
}
