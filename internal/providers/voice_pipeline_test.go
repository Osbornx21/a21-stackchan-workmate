package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestVoicePipelineRunnerProducesDownlinkReadyMockChunksAndRedactedReport(t *testing.T) {
	runner := NewVoicePipelineRunner(VoicePipelineAdapters{
		ASR:        NewMockASRAdapter("mock-local-asr"),
		TextStream: NewMockTextStreamAdapter("mock-text-stream"),
		TTS:        NewMockTTSAdapter("mock-fast-tts"),
		Selection: VoicePipelineSelection{
			ASRMode:       "local",
			ASRProfile:    "mock-local-asr",
			ASRProfileEnv: "A21_ASR_PROFILE",
			LLMProfile:    "mock-text-stream",
			LLMProfileEnv: "A21_PROVIDER_PRIMARY",
			TTSMode:       "fast",
			TTSProfile:    "mock-fast-tts",
			TTSProfileEnv: "A21_TTS_PROFILE",
		},
	})

	result, err := runner.Run(context.Background(), VoicePipelineRequest{
		Session: VoiceSession{
			TraceID:   "a21-trace-pipeline-001",
			SessionID: "a21-session-pipeline-001",
			DeviceID:  "stackchan-sim-001",
		},
		Mode: "workmate",
		Frames: []VoicePipelinePCMFrame{
			{
				Seq:          1,
				Codec:        "pcm_s16le",
				SampleRateHz: 16000,
				Channels:     1,
				DurationMS:   60,
				ByteCount:    1920,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Status != VoicePipelineStatusCompleted {
		t.Fatalf("status = %q, want completed", result.Status)
	}
	if len(result.AudioChunks) == 0 {
		t.Fatal("audio chunks = 0, want at least one downlink-ready chunk")
	}
	for _, chunk := range result.AudioChunks {
		if chunk.Codec != "pcm_s16le" || chunk.SampleRateHz != 24000 || chunk.Channels != 1 || chunk.DurationMS != 60 {
			t.Fatalf("chunk = %+v, want pcm_s16le 24k mono 60ms", chunk)
		}
		if chunk.DataBase64 == "" {
			t.Fatal("chunk data_base64 is empty")
		}
	}
	if result.Timing.ASRFirstPartialMS < 0 || result.Timing.LLMFirstContentMS < 0 || result.Timing.TTSFirstAudioMS < 0 {
		t.Fatalf("timing = %+v, want non-negative stage markers", result.Timing)
	}
	if result.Timing.TTSFirstAudioMS < result.Timing.LLMFirstContentMS {
		t.Fatalf("tts_first_audio_ms=%d before llm_first_content_ms=%d", result.Timing.TTSFirstAudioMS, result.Timing.LLMFirstContentMS)
	}

	reportJSON, err := json.Marshal(result.Report)
	if err != nil {
		t.Fatal(err)
	}
	report := string(reportJSON)
	for _, forbidden := range []string{
		"fixture transcript should never be stored",
		"mock provider output should never be stored",
		"data_base64",
		"raw_audio",
		"http://",
		"https://",
		"socks5://",
		"/Users/",
	} {
		if strings.Contains(report, forbidden) {
			t.Fatalf("redacted report leaked %q: %s", forbidden, report)
		}
	}
	for _, want := range []string{
		"a21.voice_pipeline.fixture.v1",
		"A21_ASR_PROFILE",
		"A21_PROVIDER_PRIMARY",
		"A21_TTS_PROFILE",
		"transcript_not_recorded",
		"provider_output_not_recorded",
		"audio_payload_not_recorded",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("redacted report missing %q: %s", want, report)
		}
	}
}

func TestVoicePipelineRunnerUsesPromptInputWithoutRecordingPromptText(t *testing.T) {
	textStream := &recordingPipelineTextStreamAdapter{
		events: []TextStreamEvent{{Kind: TextStreamDeltaContent, Text: "收到。"}, {Kind: TextStreamDeltaDone}},
	}
	runner := NewVoicePipelineRunner(VoicePipelineAdapters{
		ASR:        scriptedPipelineASRAdapter{text: "raw transcript should stay out"},
		TextStream: textStream,
		TTS:        NewMockTTSAdapter("mock-fast-tts"),
		Selection:  VoicePipelineSelectionFromEnv(nil),
	})
	prompt := "# A21 roleplay prompt\nMemory Hints\nsession_memory:session_memory_1: private hint should stay out"

	result, err := runner.Run(context.Background(), VoicePipelineRequest{
		Session:    VoiceSession{TraceID: "a21-trace-prompt-input", SessionID: "a21-session-prompt-input", DeviceID: "stackchan-sim-001"},
		Mode:       "roleplay",
		TextPrompt: prompt,
		Frames:     []VoicePipelinePCMFrame{{Seq: 1, Codec: "pcm_s16le", SampleRateHz: 16000, Channels: 1, DurationMS: 60, ByteCount: 1920}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != VoicePipelineStatusCompleted {
		t.Fatalf("status = %q, want completed", result.Status)
	}
	if len(textStream.requests) != 1 || textStream.requests[0] != prompt {
		t.Fatalf("text stream did not receive roleplay prompt input")
	}
	if !result.Report.Input.PromptInputReady || result.Report.Redaction.PromptPolicy != "prompt_input_not_recorded" {
		t.Fatalf("prompt redaction/readiness = %+v / %+v", result.Report.Input, result.Report.Redaction)
	}
	rendered := mustProviderJSON(t, result.Report)
	for _, forbidden := range []string{"A21 roleplay prompt", "Memory Hints", "private hint should stay out", "raw transcript should stay out"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("voice pipeline report leaked prompt/transcript text")
		}
	}
}

func TestVoicePipelineRunnerPassesVoiceCloneProfileToTTSAndReport(t *testing.T) {
	textStream := &recordingPipelineTextStreamAdapter{
		events: []TextStreamEvent{{Kind: TextStreamDeltaContent, Text: "收到。"}, {Kind: TextStreamDeltaDone}},
	}
	tts := &recordingPipelineTTSAdapter{}
	runner := NewVoicePipelineRunner(VoicePipelineAdapters{
		ASR:        scriptedPipelineASRAdapter{text: "raw transcript should stay out"},
		TextStream: textStream,
		TTS:        tts,
		Selection: VoicePipelineSelection{
			ASRMode:       "cloud",
			ASRProfile:    "dashscope_qwen_asr_realtime",
			ASRProfileEnv: "A21_ASR_CLOUD_PROFILE",
			LLMProfile:    "stepfun",
			LLMProfileEnv: "A21_TEXT_STREAM_PROFILE",
			TTSMode:       "fast",
			TTSProfile:    "voice_clone_cli",
			TTSProfileEnv: "A21_TTS_FAST_PROFILE",
		},
	})

	result, err := runner.Run(context.Background(), VoicePipelineRequest{
		Session:           VoiceSession{TraceID: "a21-trace-voice-clone", SessionID: "a21-session-voice-clone", DeviceID: "stackchan-sim-001"},
		Mode:              "roleplay",
		TextPrompt:        "# A21 roleplay prompt",
		VoiceCloneProfile: "a21_voice_clone_default",
		Frames:            []VoicePipelinePCMFrame{{Seq: 1, Codec: "pcm_s16le", SampleRateHz: 16000, Channels: 1, DurationMS: 60, ByteCount: 1920}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != VoicePipelineStatusCompleted {
		t.Fatalf("status = %q, want completed", result.Status)
	}
	requests := tts.capturedRequests()
	if len(requests) != 1 || requests[0].VoiceCloneProfile != "a21_voice_clone_default" {
		t.Fatalf("tts requests = %+v, want selected voice clone profile", requests)
	}
	if result.Report.Input.VoiceCloneProfile != "a21_voice_clone_default" ||
		result.Report.Redaction.VoiceCloneSamplePolicy != "voice_clone_sample_not_recorded" {
		t.Fatalf("voice clone report = input %+v redaction %+v", result.Report.Input, result.Report.Redaction)
	}
	rendered := mustProviderJSON(t, result.Report)
	for _, forbidden := range []string{"raw transcript should stay out", "reference.wav", "/Users/", "data_base64"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("voice pipeline report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestMockStreamingASRAdapterEmitsPartialOnFrameAndFinalOnCommit(t *testing.T) {
	adapter := NewMockStreamingASRAdapter("mock-streaming-asr")
	session, err := adapter.StartStreamingASR(context.Background(), StreamingASRStartRequest{
		Session: VoiceSession{
			TraceID:   "a21-trace-streaming-asr",
			SessionID: "a21-session-streaming-asr",
			DeviceID:  "stackchan-001",
		},
		Mode: "workmate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.AppendFrame(context.Background(), VoicePipelinePCMFrame{
		Seq:          1,
		Codec:        "pcm_s16le",
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   60,
		ByteCount:    1920,
		RMS:          0.12,
	}); err != nil {
		t.Fatal(err)
	}
	partial := <-session.Events()
	if partial.Final || strings.TrimSpace(partial.Text) == "" {
		t.Fatalf("partial = %+v, want non-final text", partial)
	}
	if err := session.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	final := <-session.Events()
	if !final.Final || strings.TrimSpace(final.Text) == "" {
		t.Fatalf("final = %+v, want final text", final)
	}
	if _, ok := <-session.Events(); ok {
		t.Fatal("events channel still open after final commit")
	}
}

func TestVoicePipelineReportFlagsTTSClippingWithoutAudioPayload(t *testing.T) {
	runner := NewVoicePipelineRunner(VoicePipelineAdapters{
		ASR:        scriptedPipelineASRAdapter{text: "private asr words never stored"},
		TextStream: scriptedPipelineTextStreamAdapter{events: []TextStreamEvent{{Kind: TextStreamDeltaContent, Text: "收到。"}, {Kind: TextStreamDeltaDone}}},
		TTS:        clippingPipelineTTSAdapter{},
		Selection:  VoicePipelineSelectionFromEnv(nil),
	})

	result, err := runner.Run(context.Background(), VoicePipelineRequest{
		Session: VoiceSession{TraceID: "a21-trace-audio-quality", SessionID: "a21-session-audio-quality", DeviceID: "stackchan-sim-001"},
		Mode:    "workmate",
		Frames:  []VoicePipelinePCMFrame{{Seq: 1, Codec: "pcm_s16le", SampleRateHz: 16000, Channels: 1, DurationMS: 60, ByteCount: 1920}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Report.Output.AudioQuality == nil {
		t.Fatal("audio quality report = nil, want aggregate quality guard")
	}
	if result.Report.Output.AudioQuality.Status != "warning" ||
		result.Report.Output.AudioQuality.ClippedSamples == 0 ||
		!voicePipelineStringSliceHas(result.Report.Output.AudioQuality.Findings, "audio_quality_clipping_detected") ||
		!voicePipelineStringSliceHas(result.Report.Findings, "audio_quality_clipping_detected") {
		t.Fatalf("audio quality/report findings = %+v / %+v", result.Report.Output.AudioQuality, result.Report.Findings)
	}
	rendered := mustProviderJSON(t, result.Report)
	for _, forbidden := range []string{"data_base64", clippingPipelineBase64(), "mock provider output", "private asr words never stored"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("voice pipeline quality report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestVoicePipelineRunnerTreatsTTSChunkErrorAsAdapterFailure(t *testing.T) {
	runner := NewVoicePipelineRunner(VoicePipelineAdapters{
		ASR:        scriptedPipelineASRAdapter{text: "private asr words never stored"},
		TextStream: scriptedPipelineTextStreamAdapter{events: []TextStreamEvent{{Kind: TextStreamDeltaContent, Text: "收到。"}, {Kind: TextStreamDeltaDone}}},
		TTS:        failingChunkPipelineTTSAdapter{},
		Selection:  VoicePipelineSelectionFromEnv(nil),
	})

	result, err := runner.Run(context.Background(), VoicePipelineRequest{
		Session: VoiceSession{TraceID: "a21-trace-tts-chunk-error", SessionID: "a21-session-tts-chunk-error", DeviceID: "stackchan-sim-001"},
		Mode:    "workmate",
		Frames:  []VoicePipelinePCMFrame{{Seq: 1, Codec: "pcm_s16le", SampleRateHz: 16000, Channels: 1, DurationMS: 60, ByteCount: 1920}},
	})
	if err == nil {
		t.Fatal("Run err = nil, want TTS adapter failure")
	}
	if result.Status != VoicePipelineStatusFailed || len(result.AudioChunks) != 0 {
		t.Fatalf("result status/chunks = %s/%d, want failed with no audio", result.Status, len(result.AudioChunks))
	}
	if !voicePipelineStringSliceHas(result.Report.Findings, "tts adapter failed") {
		t.Fatalf("report findings = %#v, want tts adapter failed", result.Report.Findings)
	}
}

func TestVoicePipelineRunnerCancelStopsBeforeTTSChunkOutput(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	runner := NewVoicePipelineRunner(VoicePipelineAdapters{
		ASR: NewMockASRAdapter("mock-local-asr"),
		TextStream: NewMockTextStreamAdapter("mock-text-stream", func() {
			cancel(ErrVoicePipelineBargeIn)
		}),
		TTS: NewMockTTSAdapter("mock-fast-tts"),
		Selection: VoicePipelineSelection{
			ASRMode:       "local",
			ASRProfile:    "mock-local-asr",
			ASRProfileEnv: "A21_ASR_PROFILE",
			LLMProfile:    "mock-text-stream",
			LLMProfileEnv: "A21_PROVIDER_PRIMARY",
			TTSMode:       "fast",
			TTSProfile:    "mock-fast-tts",
			TTSProfileEnv: "A21_TTS_PROFILE",
		},
	})

	result, err := runner.Run(ctx, VoicePipelineRequest{
		Session: VoiceSession{
			TraceID:   "a21-trace-pipeline-cancel",
			SessionID: "a21-session-pipeline-cancel",
			DeviceID:  "stackchan-sim-001",
		},
		Mode: "workmate",
		Frames: []VoicePipelinePCMFrame{
			{Seq: 1, Codec: "pcm_s16le", SampleRateHz: 16000, Channels: 1, DurationMS: 60, ByteCount: 1920},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Status != VoicePipelineStatusCancelled {
		t.Fatalf("status = %q, want cancelled", result.Status)
	}
	if result.CancelReason != CancelBargeIn {
		t.Fatalf("cancel reason = %q, want barge_in", result.CancelReason)
	}
	if len(result.AudioChunks) != 0 {
		t.Fatalf("audio chunks = %d, want 0 after barge-in cancel", len(result.AudioChunks))
	}
	if result.Timing.ProviderCancelMS < 0 || result.Timing.BargeInStopMS < 0 {
		t.Fatalf("cancel timing = %+v, want provider_cancel_ms and barge_in_stop_ms", result.Timing)
	}
	if result.Report.Status != string(VoicePipelineStatusCancelled) {
		t.Fatalf("report status = %q, want cancelled", result.Report.Status)
	}
}

func TestVoicePipelineRunKeepsFullResponseCollector(t *testing.T) {
	tts := &recordingPipelineTTSAdapter{}
	runner := NewVoicePipelineRunner(VoicePipelineAdapters{
		ASR:        scriptedPipelineASRAdapter{text: "transcript"},
		TextStream: scriptedPipelineTextStreamAdapter{events: []TextStreamEvent{{Kind: TextStreamDeltaContent, Text: "第一句。"}, {Kind: TextStreamDeltaContent, Text: "第二句。"}, {Kind: TextStreamDeltaDone}}},
		TTS:        tts,
		Selection:  VoicePipelineSelectionFromEnv(nil),
	})

	result, err := runner.Run(context.Background(), VoicePipelineRequest{
		Session: VoiceSession{TraceID: "a21-trace-run-collector", SessionID: "a21-session-run-collector", DeviceID: "stackchan-sim-001"},
		Mode:    "workmate",
		Frames:  []VoicePipelinePCMFrame{{Seq: 1, Codec: "pcm_s16le", SampleRateHz: 16000, Channels: 1, DurationMS: 60, ByteCount: 1920}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Status != VoicePipelineStatusCompleted {
		t.Fatalf("status = %q, want completed", result.Status)
	}
	if got := tts.texts(); len(got) != 1 || got[0] != "第一句。第二句。" {
		t.Fatalf("tts texts = %#v, want one full collected response", got)
	}
	if result.Report.Output.StreamingAnswer {
		t.Fatalf("streaming_answer = true, want false for Run collector")
	}
	if result.Report.Output.LLMSegmentCount != 0 {
		t.Fatalf("llm_segment_count = %d, want 0 for Run collector", result.Report.Output.LLMSegmentCount)
	}
}

func TestVoicePipelineRunStreamDeliversDoneAfterBufferedChunksDrain(t *testing.T) {
	runner := NewVoicePipelineRunner(VoicePipelineAdapters{
		ASR: scriptedPipelineASRAdapter{text: "transcript"},
		TextStream: scriptedPipelineTextStreamAdapter{events: []TextStreamEvent{
			{Kind: TextStreamDeltaContent, Text: "一。"},
			{Kind: TextStreamDeltaContent, Text: "二。"},
			{Kind: TextStreamDeltaContent, Text: "三。"},
			{Kind: TextStreamDeltaContent, Text: "四。"},
			{Kind: TextStreamDeltaDone},
		}},
		TTS:       &recordingPipelineTTSAdapter{},
		Selection: VoicePipelineSelectionFromEnv(nil),
	})

	events, err := runner.RunStream(context.Background(), VoicePipelineRequest{
		Session: VoiceSession{TraceID: "a21-trace-run-stream-done", SessionID: "a21-session-run-stream-done", DeviceID: "stackchan-sim-001"},
		Mode:    "workmate",
		Frames:  []VoicePipelinePCMFrame{{Seq: 1, Codec: "pcm_s16le", SampleRateHz: 16000, Channels: 1, DurationMS: 60, ByteCount: 1920}},
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)

	var chunkCount int
	var doneResult VoicePipelineResult
	var sawDone bool
	for event := range events {
		switch event.Kind {
		case VoicePipelineStreamAudioChunk:
			chunkCount++
		case VoicePipelineStreamDone:
			sawDone = true
			doneResult = event.Result
			if event.Err != nil {
				t.Fatal(event.Err)
			}
		}
	}
	if chunkCount != 4 {
		t.Fatalf("chunk count = %d, want 4", chunkCount)
	}
	if !sawDone {
		t.Fatal("RunStream closed without VoicePipelineStreamDone")
	}
	if doneResult.Status != VoicePipelineStatusCompleted {
		t.Fatalf("done status = %q, want completed", doneResult.Status)
	}
	if doneResult.Report.Output.LLMSegmentCount != 4 || !doneResult.Report.Output.StreamingAnswer {
		t.Fatalf("done output = %+v, want 4 streaming segments", doneResult.Report.Output)
	}
}

func TestVoicePipelineRunStreamCarriesFallbackReportOnAudioChunks(t *testing.T) {
	runner := NewVoicePipelineRunner(VoicePipelineAdapters{
		ASR: scriptedPipelineASRAdapter{text: "transcript"},
		TextStream: scriptedPipelineTextStreamAdapter{events: []TextStreamEvent{
			{
				Finding: "provider_fallback_used",
				Fallback: &TextStreamFallbackEvent{
					Activated: true,
					Provider:  "a21_voice_fallback",
					Reason:    "primary_failed",
				},
			},
			{Kind: TextStreamDeltaContent, Text: "fallback voice answer。"},
			{Kind: TextStreamDeltaDone},
		}},
		TTS: &recordingPipelineTTSAdapter{},
		Selection: VoicePipelineSelection{
			ASRMode:               "local",
			ASRProfile:            "mock-local-asr",
			ASRProfileEnv:         "A21_ASR_LOCAL_PROFILE",
			LLMProfile:            "deepseek",
			LLMProfileEnv:         "A21_PROVIDER_PRIMARY",
			LLMFallbackProfile:    "a21_voice_fallback",
			LLMFallbackProfileEnv: "A21_TEXT_STREAM_FALLBACK_PROFILE",
			TTSMode:               "fast",
			TTSProfile:            "mock-fast-tts",
			TTSProfileEnv:         "A21_TTS_FAST_PROFILE",
		},
	})

	events, err := runner.RunStream(context.Background(), VoicePipelineRequest{
		Session: VoiceSession{TraceID: "a21-trace-run-stream-fallback", SessionID: "a21-session-run-stream-fallback", DeviceID: "stackchan-sim-001"},
		Mode:    "workmate",
		Frames:  []VoicePipelinePCMFrame{{Seq: 1, Codec: "pcm_s16le", SampleRateHz: 16000, Channels: 1, DurationMS: 60, ByteCount: 1920}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var sawAudio bool
	var sawDone bool
	for event := range events {
		switch event.Kind {
		case VoicePipelineStreamAudioChunk:
			sawAudio = true
			if event.Report.Fallback == nil ||
				!event.Report.Fallback.Activated ||
				event.Report.Fallback.Provider != "a21_voice_fallback" ||
				event.Report.Fallback.Reason != "primary_failed" {
				t.Fatalf("audio event fallback report = %+v", event.Report.Fallback)
			}
			if event.Report.Selection.LLMFallbackProfile != "a21_voice_fallback" {
				t.Fatalf("audio event selection = %+v", event.Report.Selection)
			}
		case VoicePipelineStreamDone:
			sawDone = true
			if event.Err != nil {
				t.Fatal(event.Err)
			}
		}
	}
	if !sawAudio || !sawDone {
		t.Fatalf("saw audio=%v done=%v, want both", sawAudio, sawDone)
	}
}

func TestVoicePipelineRunStreamStartsLLMFromStreamingASRPartialBeforeFinal(t *testing.T) {
	tts := &recordingPipelineTTSAdapter{}
	runner := NewVoicePipelineRunner(VoicePipelineAdapters{
		ASR: scriptedPipelineASRAdapter{text: "batch fallback should not run for partial bridge"},
		TextStream: scriptedPipelineTextStreamAdapter{events: []TextStreamEvent{
			{Kind: TextStreamDeltaContent, Text: "partial-driven answer。"},
			{Kind: TextStreamDeltaDone},
		}},
		TTS:       tts,
		Selection: VoicePipelineSelectionFromEnv(nil),
	})

	events, err := runner.RunStream(context.Background(), VoicePipelineRequest{
		Session:             VoiceSession{TraceID: "a21-trace-partial-bridge", SessionID: "a21-session-partial-bridge", DeviceID: "stackchan-sim-001"},
		Mode:                "workmate",
		Frames:              []VoicePipelinePCMFrame{{Seq: 1, Codec: "pcm_s16le", SampleRateHz: 16000, Channels: 1, DurationMS: 60, ByteCount: 1920}},
		ASRTranscript:       "streaming partial should not be stored",
		ASRTranscriptSource: VoicePipelineASRTranscriptSourcePartial,
	})
	if err != nil {
		t.Fatal(err)
	}

	var firstAudio VoicePipelineStreamEvent
	var done VoicePipelineStreamEvent
	for event := range events {
		switch event.Kind {
		case VoicePipelineStreamAudioChunk:
			if firstAudio.Kind == "" {
				firstAudio = event
			}
		case VoicePipelineStreamDone:
			done = event
		}
	}
	if firstAudio.Kind != VoicePipelineStreamAudioChunk {
		t.Fatal("missing first streaming audio chunk")
	}
	if done.Kind != VoicePipelineStreamDone || done.Err != nil {
		t.Fatalf("done event = %+v", done)
	}
	if firstAudio.Timing.ASRFirstPartialMS != 0 || firstAudio.Timing.ASRFinalMS != -1 {
		t.Fatalf("first audio timing = %+v, want partial available and final not yet available", firstAudio.Timing)
	}
	if firstAudio.Timing.LLMFirstContentMS < 0 || firstAudio.Timing.TTSFirstAudioMS < firstAudio.Timing.LLMFirstContentMS {
		t.Fatalf("first audio timing = %+v, want LLM content before TTS audio", firstAudio.Timing)
	}
	if done.Result.Timing.ASRFinalMS != -1 || done.Result.Timing.LLMFirstContentMS < 0 {
		t.Fatalf("done timing = %+v, want no synthetic ASR final for partial-driven path", done.Result.Timing)
	}
	if !voicePipelineStringSliceHas(done.Result.Report.Findings, "streaming_asr_partial_reused") {
		t.Fatalf("findings = %+v, want streaming_asr_partial_reused", done.Result.Report.Findings)
	}
	rendered := mustProviderJSON(t, done.Result.Report)
	if strings.Contains(rendered, "streaming partial should not be stored") {
		t.Fatalf("partial transcript leaked into report: %s", rendered)
	}
	if got := tts.texts(); len(got) != 1 || got[0] != "partial-driven answer。" {
		t.Fatalf("tts texts = %#v, want one streamed answer segment", got)
	}
}

func TestVoicePipelineSelectionCanChangeWithoutGatewayCore(t *testing.T) {
	selection := VoicePipelineSelectionFromEnv([]string{
		"A21_ASR_PROFILE=cloud",
		"A21_ASR_CLOUD_PROFILE=fixture-cloud-asr",
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_TEXT_STREAM_PROFILE=deepseek",
		"A21_TTS_MODE=quality",
		"A21_TTS_QUALITY_PROFILE=fixture-quality-tts",
	})

	if selection.ASRMode != "cloud" || selection.ASRProfile != "fixture-cloud-asr" {
		t.Fatalf("asr selection = %+v, want cloud fixture-cloud-asr", selection)
	}
	if selection.LLMProfile != "deepseek" {
		t.Fatalf("llm profile = %q, want deepseek", selection.LLMProfile)
	}
	if selection.TTSMode != "quality" || selection.TTSProfile != "fixture-quality-tts" {
		t.Fatalf("tts selection = %+v, want quality fixture-quality-tts", selection)
	}
	if selection.ASRProfileEnv != "A21_ASR_CLOUD_PROFILE" || selection.LLMProfileEnv != "A21_TEXT_STREAM_PROFILE" || selection.TTSProfileEnv != "A21_TTS_QUALITY_PROFILE" {
		t.Fatalf("selection env names = %+v, want profile env names only", selection)
	}
}

type scriptedPipelineASRAdapter struct {
	text string
}

func (a scriptedPipelineASRAdapter) Name() string {
	return "a21-scripted-asr"
}

func (a scriptedPipelineASRAdapter) Transcribe(ctx context.Context, req ASRAdapterRequest) (<-chan ASRAdapterEvent, error) {
	out := make(chan ASRAdapterEvent, 1)
	out <- ASRAdapterEvent{Text: a.text, Final: true}
	close(out)
	return out, nil
}

type scriptedPipelineTextStreamAdapter struct {
	events []TextStreamEvent
}

func (a scriptedPipelineTextStreamAdapter) Name() string {
	return "a21-scripted-text-stream"
}

func (a scriptedPipelineTextStreamAdapter) StreamText(ctx context.Context, req TextStreamAdapterRequest) (<-chan TextStreamEvent, error) {
	out := make(chan TextStreamEvent, len(a.events))
	for _, event := range a.events {
		out <- event
	}
	close(out)
	return out, nil
}

type recordingPipelineTextStreamAdapter struct {
	mu       sync.Mutex
	requests []string
	events   []TextStreamEvent
}

func (a *recordingPipelineTextStreamAdapter) Name() string {
	return "a21-recording-text-stream"
}

func (a *recordingPipelineTextStreamAdapter) StreamText(ctx context.Context, req TextStreamAdapterRequest) (<-chan TextStreamEvent, error) {
	a.mu.Lock()
	a.requests = append(a.requests, req.Text)
	a.mu.Unlock()
	out := make(chan TextStreamEvent, len(a.events))
	for _, event := range a.events {
		out <- event
	}
	close(out)
	return out, nil
}

type recordingPipelineTTSAdapter struct {
	mu       sync.Mutex
	requests []TTSAdapterRequest
}

func (a *recordingPipelineTTSAdapter) Name() string {
	return "a21-recording-tts"
}

func (a *recordingPipelineTTSAdapter) Synthesize(ctx context.Context, req TTSAdapterRequest) (<-chan VoiceAudioChunk, error) {
	a.mu.Lock()
	a.requests = append(a.requests, req)
	a.mu.Unlock()
	out := make(chan VoiceAudioChunk, 1)
	out <- VoiceAudioChunk{
		Codec:        "pcm_s16le",
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   base64.StdEncoding.EncodeToString(make([]byte, 2880)),
	}
	close(out)
	return out, nil
}

func (a *recordingPipelineTTSAdapter) texts() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, 0, len(a.requests))
	for _, req := range a.requests {
		out = append(out, req.Text)
	}
	return out
}

func (a *recordingPipelineTTSAdapter) capturedRequests() []TTSAdapterRequest {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]TTSAdapterRequest(nil), a.requests...)
}

type clippingPipelineTTSAdapter struct{}

func (clippingPipelineTTSAdapter) Name() string {
	return "a21-clipping-tts"
}

func (clippingPipelineTTSAdapter) Synthesize(ctx context.Context, req TTSAdapterRequest) (<-chan VoiceAudioChunk, error) {
	out := make(chan VoiceAudioChunk, 1)
	out <- VoiceAudioChunk{
		Codec:        "pcm_s16le",
		SampleRateHz: 48000,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   clippingPipelineBase64(),
	}
	close(out)
	return out, nil
}

type failingChunkPipelineTTSAdapter struct{}

func (failingChunkPipelineTTSAdapter) Name() string {
	return "a21-failing-chunk-tts"
}

func (failingChunkPipelineTTSAdapter) Synthesize(ctx context.Context, req TTSAdapterRequest) (<-chan VoiceAudioChunk, error) {
	out := make(chan VoiceAudioChunk, 1)
	out <- VoiceAudioChunk{Finding: "tts adapter failed", Err: errors.New("a21 redacted tts chunk failure")}
	close(out)
	return out, nil
}

func clippingPipelineBase64() string {
	data := make([]byte, 5760)
	for i, sample := range []int16{0, 1200, -1200, 32767, -32768, 29491} {
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func mustProviderJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
