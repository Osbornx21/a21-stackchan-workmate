package providers

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"a21.local/a21/internal/protocol"
)

func TestOpenAIRealtimeInputAudioAppendEventMapsA21Chunk(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4})
	event, err := OpenAIRealtimeInputAudioAppendEvent(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if event["type"] != "input_audio_buffer.append" {
		t.Fatalf("type = %v, want input_audio_buffer.append", event["type"])
	}
	if event["audio"] != payload {
		t.Fatalf("audio = %v, want original payload", event["audio"])
	}
}

func TestOpenAIRealtimeInputAudioAppendEventRejectsUnsupportedChunk(t *testing.T) {
	tests := []struct {
		name  string
		chunk protocol.AudioChunk
		want  string
	}{
		{
			name: "opus",
			chunk: protocol.AudioChunk{
				Codec:        protocol.AudioCodecOpus,
				SampleRateHz: 24000,
				Channels:     1,
				DurationMS:   20,
				DataBase64:   base64.StdEncoding.EncodeToString([]byte{1, 2}),
			},
			want: "pcm_s16le",
		},
		{
			name: "stereo",
			chunk: protocol.AudioChunk{
				Codec:        protocol.AudioCodecPCMS16LE,
				SampleRateHz: 24000,
				Channels:     2,
				DurationMS:   20,
				DataBase64:   base64.StdEncoding.EncodeToString([]byte{1, 2}),
			},
			want: "mono",
		},
		{
			name: "invalid sample rate",
			chunk: protocol.AudioChunk{
				Codec:        protocol.AudioCodecPCMS16LE,
				SampleRateHz: 44100,
				Channels:     1,
				DurationMS:   20,
				DataBase64:   base64.StdEncoding.EncodeToString([]byte{1, 2}),
			},
			want: "sample rate",
		},
		{
			name: "invalid base64",
			chunk: protocol.AudioChunk{
				Codec:        protocol.AudioCodecPCMS16LE,
				SampleRateHz: 24000,
				Channels:     1,
				DurationMS:   20,
				DataBase64:   "not-base64!",
			},
			want: "base64",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := OpenAIRealtimeInputAudioAppendEvent(tt.chunk)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestOpenAIRealtimeManualTurnEvents(t *testing.T) {
	events := OpenAIRealtimeManualTurnEvents()
	if len(events) != 2 {
		t.Fatalf("events = %#v, want 2", events)
	}
	if events[0]["type"] != "input_audio_buffer.commit" {
		t.Fatalf("first type = %v, want input_audio_buffer.commit", events[0]["type"])
	}
	if events[1]["type"] != "response.create" {
		t.Fatalf("second type = %v, want response.create", events[1]["type"])
	}
	cancel := OpenAIRealtimeCancelEvent(VoiceCancelRequest{Reason: CancelBargeIn})
	if cancel["type"] != "response.cancel" {
		t.Fatalf("cancel = %#v, want response.cancel", cancel)
	}
	if _, ok := cancel["reason"]; ok {
		t.Fatalf("cancel leaked A21-local reason into provider event: %#v", cancel)
	}
}

func TestOpenAIRealtimeServerAudioDeltaMapsToVoiceEvent(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte{0, 1})
	event, ok, err := OpenAIRealtimeServerEventToVoiceEvent(VoiceSession{
		TraceID:   "a21-trace-openai-001",
		SessionID: "a21-session-openai-001",
		DeviceID:  "stackchan-001",
	}, map[string]any{
		"type":        "response.output_audio.delta",
		"response_id": "resp_123",
		"delta":       payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if event.Kind != VoiceEventSpeaking {
		t.Fatalf("kind = %q, want speaking", event.Kind)
	}
	if event.StreamID != "resp_123" {
		t.Fatalf("stream id = %q, want resp_123", event.StreamID)
	}
	if event.Audio == nil {
		t.Fatal("audio = nil, want audio chunk")
	}
	if event.Audio.DataBase64 != payload {
		t.Fatalf("audio payload = %q, want original delta", event.Audio.DataBase64)
	}
	if event.Audio.Codec != string(protocol.AudioCodecPCMS16LE) || event.Audio.SampleRateHz != 24000 || event.Audio.Channels != 1 {
		t.Fatalf("audio format = %#v, want pcm_s16le 24k mono", event.Audio)
	}
}

func TestOpenAIRealtimeServerEventIgnoresUnrelatedEvent(t *testing.T) {
	event, ok, err := OpenAIRealtimeServerEventToVoiceEvent(VoiceSession{}, map[string]any{
		"type": "rate_limits.updated",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("ok = true with event %#v, want ignored", event)
	}
}

func TestOpenAIRealtimeSessionWritesAudioAndManualTurnEvents(t *testing.T) {
	conn := &fakeRealtimeConn{}
	session := &RealtimeWebSocketSession{
		provider: "openai_realtime",
		conn:     conn,
	}
	payload := base64.StdEncoding.EncodeToString([]byte{1, 2})
	err := OpenAIRealtimeSendAudioChunk(context.Background(), session, protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := OpenAIRealtimeCommitAndCreateResponse(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if len(conn.messages) != 3 {
		t.Fatalf("messages = %#v, want 3", conn.messages)
	}
	if got, want := conn.messages[0]["type"], "input_audio_buffer.append"; got != want {
		t.Fatalf("first type = %v, want %s", got, want)
	}
	if got, want := conn.messages[1]["type"], "input_audio_buffer.commit"; got != want {
		t.Fatalf("second type = %v, want %s", got, want)
	}
	if got, want := conn.messages[2]["type"], "response.create"; got != want {
		t.Fatalf("third type = %v, want %s", got, want)
	}
}
