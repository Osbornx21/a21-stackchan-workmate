package providers

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
)

func TestDoubaoRealtimeTTSSessionUpdateEventUsesDocumentedShape(t *testing.T) {
	event, err := DoubaoRealtimeTTSSessionUpdateEvent(DoubaoRealtimeTTSConfig{
		Model:                 "doubao-tts",
		Voice:                 "zh_female_kailangjiejie_moon_bigtts",
		OutputAudioFormat:     "pcm",
		OutputAudioSampleRate: 16000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if event["type"] != "tts_session.update" {
		t.Fatalf("type = %v, want tts_session.update", event["type"])
	}
	session, ok := event["session"].(map[string]any)
	if !ok {
		t.Fatalf("session = %#v, want object", event["session"])
	}
	if session["voice"] != "zh_female_kailangjiejie_moon_bigtts" {
		t.Fatalf("voice = %v", session["voice"])
	}
	if session["output_audio_format"] != "pcm" || session["output_audio_sample_rate"] != 16000 {
		t.Fatalf("audio format/rate = %v/%v", session["output_audio_format"], session["output_audio_sample_rate"])
	}
	tts, ok := session["text_to_speech"].(map[string]any)
	if !ok || tts["model"] != "doubao-tts" {
		t.Fatalf("text_to_speech = %#v", session["text_to_speech"])
	}
	rendered := realtimeMustJSON(t, event)
	for _, forbidden := range []string{"sk-", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("event leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestDoubaoRealtimeTTSTextEvents(t *testing.T) {
	appendEvent, err := DoubaoRealtimeTTSInputTextAppendEvent("你好")
	if err != nil {
		t.Fatal(err)
	}
	if appendEvent["type"] != "input_text.append" || appendEvent["delta"] != "你好" {
		t.Fatalf("append event = %#v", appendEvent)
	}
	doneEvent := DoubaoRealtimeTTSInputTextDoneEvent()
	if doneEvent["type"] != "input_text.done" {
		t.Fatalf("done event = %#v", doneEvent)
	}
	if _, err := DoubaoRealtimeTTSInputTextAppendEvent("   "); err == nil {
		t.Fatal("expected blank text delta to be rejected")
	}
}

func TestDoubaoRealtimeTTSServerAudioDeltaMapsToVoiceEvent(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4})
	event, ok, err := DoubaoRealtimeTTSServerEventToVoiceEvent(VoiceSession{
		TraceID:   "a21-trace-doubao-tts-001",
		SessionID: "a21-session-doubao-tts-001",
		DeviceID:  "stackchan-001",
	}, 16000, map[string]any{
		"type":    "response.audio.delta",
		"item_id": "item-001",
		"delta":   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if event.Kind != VoiceEventSpeaking || event.StreamID != "item-001" {
		t.Fatalf("event kind/stream = %q/%q", event.Kind, event.StreamID)
	}
	if event.Audio == nil || event.Audio.Codec != "pcm_s16le" || event.Audio.SampleRateHz != 16000 || event.Audio.DataBase64 != payload {
		t.Fatalf("audio = %#v", event.Audio)
	}
}

func TestDoubaoRealtimeTTSSessionWritesUpdateTextAndDone(t *testing.T) {
	conn := &fakeRealtimeConn{}
	session := &RealtimeWebSocketSession{provider: "doubao_tts_realtime", conn: conn}

	err := DoubaoRealtimeTTSSendSessionUpdate(context.Background(), session, DoubaoRealtimeTTSConfig{
		Model:                 "doubao-tts",
		Voice:                 "zh_female_kailangjiejie_moon_bigtts",
		OutputAudioFormat:     "pcm",
		OutputAudioSampleRate: 16000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := DoubaoRealtimeTTSSendText(context.Background(), session, "你好"); err != nil {
		t.Fatal(err)
	}
	if err := DoubaoRealtimeTTSSendTextDone(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	got := realtimeEventTypes(conn.messages)
	want := []string{"tts_session.update", "input_text.append", "input_text.done"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("event types = %#v, want %#v", got, want)
	}
}
