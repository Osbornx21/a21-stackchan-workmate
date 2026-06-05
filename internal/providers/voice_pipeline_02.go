package providers

import (
	"context"
	"encoding/base64"
	"strings"
	"sync"
	"time"

	"a21.local/a21/internal/audio"
)

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
