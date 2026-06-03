package providers

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
)

func TestDoubaoRealtimeASRSessionUpdateEventUsesRedactedStreamingShape(t *testing.T) {
	event, err := DoubaoRealtimeASRSessionUpdateEvent(DoubaoRealtimeASRProviderConfig{
		Model:       "doubao-asr",
		InputRateHz: 16000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if event["type"] != "transcription_session.update" {
		t.Fatalf("type = %v, want transcription_session.update", event["type"])
	}
	session, ok := event["session"].(map[string]any)
	if !ok {
		t.Fatalf("session = %#v, want object", event["session"])
	}
	if session["input_audio_format"] != "pcm" ||
		session["input_audio_codec"] != "raw" ||
		session["input_audio_sample_rate"] != 16000 ||
		session["input_audio_bits"] != 16 ||
		session["input_audio_channel"] != 1 {
		t.Fatalf("input audio session = %#v", session)
	}
	transcription, ok := session["input_audio_transcription"].(map[string]any)
	if !ok || transcription["model"] != "doubao-asr" {
		t.Fatalf("input_audio_transcription = %#v", session["input_audio_transcription"])
	}
	rendered := realtimeMustJSON(t, event)
	for _, forbidden := range []string{"sk-", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("event leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestDoubaoRealtimeASRAudioAndTranscriptEvents(t *testing.T) {
	frame := VoicePipelinePCMFrame{
		Codec:        "pcm_s16le",
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   60,
		ByteCount:    4,
		PCM16LE:      []byte{1, 2, 3, 4},
	}
	appendEvent, err := DoubaoRealtimeASRInputAudioAppendEvent(frame)
	if err != nil {
		t.Fatal(err)
	}
	if appendEvent["type"] != "input_audio_buffer.append" ||
		appendEvent["audio"] != base64.StdEncoding.EncodeToString(frame.PCM16LE) {
		t.Fatalf("append event = %#v", appendEvent)
	}
	if _, err := DoubaoRealtimeASRInputAudioAppendEvent(VoicePipelinePCMFrame{}); err == nil {
		t.Fatal("expected empty audio frame to be rejected")
	}
	partial, ok, err := DoubaoRealtimeASRServerEventToASREvent(map[string]any{
		"type": "conversation.item.input_audio_transcription.result",
		"text": "你好",
	})
	if err != nil || !ok || partial.Text != "你好" || partial.Final {
		t.Fatalf("partial = %#v ok=%v err=%v", partial, ok, err)
	}
	final, ok, err := DoubaoRealtimeASRServerEventToASREvent(map[string]any{
		"type":       "conversation.item.input_audio_transcription.completed",
		"transcript": "你好 A21",
	})
	if err != nil || !ok || final.Text != "你好 A21" || !final.Final {
		t.Fatalf("final = %#v ok=%v err=%v", final, ok, err)
	}
}

func TestVoicePipelineAdaptersFromEnvSelectsDoubaoCloudStreamingASR(t *testing.T) {
	conn := &fakeRealtimeConn{
		serverMessages: []map[string]any{
			{"type": "conversation.item.input_audio_transcription.delta", "delta": "你好"},
			{"type": "conversation.item.input_audio_transcription.completed", "transcript": "你好 A21"},
		},
	}
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_ASR_PROFILE=cloud",
		"A21_ASR_CLOUD_PROFILE=doubao_asr_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_ASR_MODEL=doubao-asr",
	}, VoicePipelineAdapterOptions{
		TTSRealtimeDialer: fakeRealtimeDialer{conn: conn},
	})
	if adapters.ExecutionMode != "cloud_edge" || adapters.ASR.Name() != "doubao_asr_realtime" {
		t.Fatalf("adapters execution/ASR = %q/%q, want cloud_edge/doubao_asr_realtime", adapters.ExecutionMode, adapters.ASR.Name())
	}
	streaming, ok := adapters.ASR.(StreamingASRAdapter)
	if !ok {
		t.Fatalf("selected ASR %T does not implement StreamingASRAdapter", adapters.ASR)
	}
	session, err := streaming.StartStreamingASR(context.Background(), StreamingASRStartRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.AppendFrame(context.Background(), VoicePipelinePCMFrame{PCM16LE: []byte{1, 2, 3, 4}, ByteCount: 4}); err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	var got []ASRAdapterEvent
	for event := range session.Events() {
		got = append(got, event)
	}
	if len(got) != 2 || got[0].Text != "你好" || got[0].Final || got[1].Text != "你好 A21" || !got[1].Final {
		t.Fatalf("events = %#v", got)
	}
	if types := realtimeEventTypes(conn.messages); strings.Join(types, ",") != "transcription_session.update,input_audio_buffer.append,input_audio_buffer.commit" {
		t.Fatalf("client event types = %#v", types)
	}
}
