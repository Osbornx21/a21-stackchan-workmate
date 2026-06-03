package providers

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
)

func TestVoicePipelineAdaptersFromEnvSelectsDashScopeCloudStreamingASR(t *testing.T) {
	conn := &fakeRealtimeConn{
		serverMessages: []map[string]any{
			{"type": "conversation.item.input_audio_transcription.text", "text": "你好"},
			{"type": "conversation.item.input_audio_transcription.completed", "transcript": "你好 A21"},
		},
	}
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_ASR_PROFILE=cloud",
		"A21_ASR_CLOUD_PROFILE=dashscope_qwen_asr_realtime",
		"A21_DASHSCOPE_API_KEY=sk-a21-secret",
		"A21_DASHSCOPE_ASR_MODEL=qwen3-asr-flash-realtime",
	}, VoicePipelineAdapterOptions{
		TTSRealtimeDialer: fakeRealtimeDialer{conn: conn},
	})
	if adapters.ExecutionMode != "cloud_edge" || adapters.ASR.Name() != "dashscope_qwen_asr_realtime" {
		t.Fatalf("adapters execution/ASR = %q/%q, want cloud_edge/dashscope_qwen_asr_realtime", adapters.ExecutionMode, adapters.ASR.Name())
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
	if types := realtimeEventTypes(conn.messages); strings.Join(types, ",") != "session.update,input_audio_buffer.append,input_audio_buffer.commit,session.finish" {
		t.Fatalf("client event types = %#v", types)
	}
	assertDashScopeRealtimeEventIDs(t, conn.messages)
}

func TestVoicePipelineAdaptersFromEnvSelectsDashScopeRealtimeTTS(t *testing.T) {
	conn := &fakeRealtimeConn{
		serverMessages: []map[string]any{
			{"type": "session.created"},
			{"type": "session.updated"},
			{"type": "response.created"},
			{
				"type":  "response.audio.delta",
				"delta": base64.StdEncoding.EncodeToString(make([]byte, 2880)),
			},
			{"type": "response.audio.done"},
			{"type": "response.done"},
			{"type": "session.finished"},
		},
	}
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_TTS_FAST_PROFILE=dashscope_qwen_tts_realtime",
		"A21_DASHSCOPE_API_KEY=sk-a21-secret",
		"A21_DASHSCOPE_TTS_MODEL=qwen3-tts-flash-realtime",
		"A21_DASHSCOPE_TTS_VOICE=Cherry",
	}, VoicePipelineAdapterOptions{
		TTSRealtimeDialer: fakeRealtimeDialer{conn: conn},
	})
	if adapters.ExecutionMode != "cloud_edge" || adapters.TTS.Name() != "dashscope_qwen_tts_realtime" {
		t.Fatalf("adapters execution/TTS = %q/%q, want cloud_edge/dashscope_qwen_tts_realtime", adapters.ExecutionMode, adapters.TTS.Name())
	}
	if _, ok := adapters.TTS.(StreamingTTSAdapter); !ok {
		t.Fatalf("selected TTS %T does not implement StreamingTTSAdapter", adapters.TTS)
	}
	chunks, err := adapters.TTS.Synthesize(context.Background(), TTSAdapterRequest{Text: "文本不进报告"})
	if err != nil {
		t.Fatal(err)
	}
	first := receiveVoiceAudioChunk(t, chunks)
	if first.SampleRateHz != 24000 || first.DurationMS != 60 {
		t.Fatalf("first chunk = %+v, want 24k 60ms", first)
	}
	for range chunks {
	}
	if types := realtimeEventTypes(conn.messages); strings.Join(types, ",") != "session.update,input_text_buffer.append,input_text_buffer.commit,session.finish" {
		t.Fatalf("client event types = %#v", types)
	}
	assertDashScopeRealtimeEventIDs(t, conn.messages)
}

func TestDashScopeRealtimeTTSWaitsForSessionUpdatedBeforeText(t *testing.T) {
	conn := &fakeRealtimeConn{
		serverMessages: []map[string]any{
			{"type": "session.created"},
			{"type": "session.updated"},
			{
				"type":  "response.audio.delta",
				"delta": base64.StdEncoding.EncodeToString(make([]byte, 2880)),
			},
			{"type": "response.audio.done"},
			{"type": "response.done"},
			{"type": "session.finished"},
		},
	}
	adapter := NewDashScopeRealtimeTTSAdapter(DashScopeRealtimeTTSAdapterOptions{
		Env:    []string{"A21_DASHSCOPE_API_KEY=sk-a21-secret"},
		Dialer: fakeRealtimeDialer{conn: conn},
	})
	chunks, err := adapter.Synthesize(context.Background(), TTSAdapterRequest{Text: "文本不进报告"})
	if err != nil {
		t.Fatal(err)
	}
	first := receiveVoiceAudioChunk(t, chunks)
	if first.Err != nil || first.SampleRateHz != 24000 {
		t.Fatalf("first chunk = %+v", first)
	}
	for range chunks {
	}
	if timelineIndex(conn.timeline, "write:input_text_buffer.append") < timelineIndex(conn.timeline, "read:session.updated") {
		t.Fatalf("timeline = %#v, text append must wait for session.updated", conn.timeline)
	}
}

func TestDashScopeRealtimeTTSDoesNotFinishBeforeAudioDone(t *testing.T) {
	conn := &fakeRealtimeConn{
		serverMessages: []map[string]any{
			{"type": "session.created"},
			{"type": "session.updated"},
			{"type": "response.created"},
			{
				"type":  "response.audio.delta",
				"delta": base64.StdEncoding.EncodeToString(make([]byte, 2880)),
			},
			{"type": "response.audio.done"},
			{"type": "response.done"},
			{"type": "session.finished"},
		},
	}
	adapter := NewDashScopeRealtimeTTSAdapter(DashScopeRealtimeTTSAdapterOptions{
		Env:    []string{"A21_DASHSCOPE_API_KEY=sk-a21-secret"},
		Dialer: fakeRealtimeDialer{conn: conn},
	})
	chunks, err := adapter.Synthesize(context.Background(), TTSAdapterRequest{Text: "文本不进报告"})
	if err != nil {
		t.Fatal(err)
	}
	first := receiveVoiceAudioChunk(t, chunks)
	if first.Err != nil || first.SampleRateHz != 24000 {
		t.Fatalf("first chunk = %+v", first)
	}
	for range chunks {
	}
	if timelineIndex(conn.timeline, "write:session.finish") < timelineIndex(conn.timeline, "read:response.audio.done") {
		t.Fatalf("timeline = %#v, session.finish must follow audio completion", conn.timeline)
	}
	if timelineIndex(conn.timeline, "write:session.finish") < timelineIndex(conn.timeline, "read:response.audio.delta") {
		t.Fatalf("timeline = %#v, audio delta must arrive before session.finish", conn.timeline)
	}
}

func TestDashScopeRealtimeTTSCancelDoesNotSendSessionFinish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	conn := &fakeRealtimeConn{
		serverMessages: []map[string]any{
			{"type": "session.updated"},
		},
	}
	adapter := NewDashScopeRealtimeTTSAdapter(DashScopeRealtimeTTSAdapterOptions{
		Env:    []string{"A21_DASHSCOPE_API_KEY=sk-a21-secret"},
		Dialer: fakeRealtimeDialer{conn: conn},
	})
	chunks, err := adapter.Synthesize(ctx, TTSAdapterRequest{Text: "文本不进报告"})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	for range chunks {
	}
	if timelineIndex(conn.timeline, "write:session.finish") < len(conn.timeline) {
		t.Fatalf("timeline = %#v, cancelled TTS must not send session.finish", conn.timeline)
	}
}

func TestDashScopeRealtimeTTSReportsDoneWithoutAudioAsAdapterFailure(t *testing.T) {
	conn := &fakeRealtimeConn{
		serverMessages: []map[string]any{
			{"type": "session.updated"},
			{"type": "response.done"},
		},
	}
	adapter := NewDashScopeRealtimeTTSAdapter(DashScopeRealtimeTTSAdapterOptions{
		Env:    []string{"A21_DASHSCOPE_API_KEY=sk-a21-secret"},
		Dialer: fakeRealtimeDialer{conn: conn},
	})
	chunks, err := adapter.Synthesize(context.Background(), TTSAdapterRequest{Text: "文本不进报告"})
	if err != nil {
		t.Fatal(err)
	}
	chunk, ok := <-chunks
	if !ok {
		t.Fatal("chunks closed without adapter failure marker")
	}
	if chunk.Err == nil || chunk.Finding != "tts adapter no audio" {
		t.Fatalf("chunk error = %+v, want redacted no-audio TTS adapter failure", chunk)
	}
}

func timelineIndex(timeline []string, value string) int {
	for i, got := range timeline {
		if got == value {
			return i
		}
	}
	return 1 << 30
}

func assertDashScopeRealtimeEventIDs(t *testing.T, messages []map[string]any) {
	t.Helper()
	for _, message := range messages {
		eventID, ok := message["event_id"].(string)
		if !ok || !strings.HasPrefix(eventID, "a21_") {
			t.Fatalf("dashscope event missing A21 event_id: %#v", message)
		}
	}
}
