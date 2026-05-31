package providers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
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
