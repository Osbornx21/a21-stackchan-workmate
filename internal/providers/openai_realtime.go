package providers

import (
	"context"
	"encoding/base64"
	"fmt"

	"a21.local/a21/internal/protocol"
)

const (
	openAIRealtimeInputAudioAppend = "input_audio_buffer.append"
	openAIRealtimeInputAudioCommit = "input_audio_buffer.commit"
	openAIRealtimeResponseCreate   = "response.create"
	openAIRealtimeResponseCancel   = "response.cancel"
	openAIRealtimeAudioDelta       = "response.output_audio.delta"
)

func OpenAIRealtimeInputAudioAppendEvent(chunk protocol.AudioChunk) (map[string]any, error) {
	if chunk.Codec != protocol.AudioCodecPCMS16LE {
		return nil, fmt.Errorf("openai realtime audio requires pcm_s16le codec")
	}
	if chunk.Channels != 1 {
		return nil, fmt.Errorf("openai realtime audio requires mono input")
	}
	if !openAIRealtimeSupportedSampleRate(chunk.SampleRateHz) {
		return nil, fmt.Errorf("openai realtime audio sample rate %d is unsupported", chunk.SampleRateHz)
	}
	if chunk.DurationMS <= 0 || chunk.DurationMS > 200 {
		return nil, fmt.Errorf("openai realtime audio duration %dms is unsupported", chunk.DurationMS)
	}
	if _, err := base64.StdEncoding.DecodeString(chunk.DataBase64); err != nil || chunk.DataBase64 == "" {
		return nil, fmt.Errorf("openai realtime audio payload must be valid base64")
	}
	return map[string]any{
		"type":  openAIRealtimeInputAudioAppend,
		"audio": chunk.DataBase64,
	}, nil
}

func OpenAIRealtimeManualTurnEvents() []map[string]any {
	return []map[string]any{
		{"type": openAIRealtimeInputAudioCommit},
		{"type": openAIRealtimeResponseCreate},
	}
}

func OpenAIRealtimeCancelEvent(req VoiceCancelRequest) map[string]any {
	return map[string]any{
		"type": openAIRealtimeResponseCancel,
	}
}

func OpenAIRealtimeSendAudioChunk(ctx context.Context, session *RealtimeWebSocketSession, chunk protocol.AudioChunk) error {
	event, err := OpenAIRealtimeInputAudioAppendEvent(chunk)
	if err != nil {
		return err
	}
	return openAIRealtimeWriteEvent(ctx, session, event)
}

func OpenAIRealtimeCommitAndCreateResponse(ctx context.Context, session *RealtimeWebSocketSession) error {
	for _, event := range OpenAIRealtimeManualTurnEvents() {
		if err := openAIRealtimeWriteEvent(ctx, session, event); err != nil {
			return err
		}
	}
	return nil
}

func OpenAIRealtimeServerEventToVoiceEvent(session VoiceSession, raw map[string]any) (VoiceEvent, bool, error) {
	eventType, _ := raw["type"].(string)
	if eventType != openAIRealtimeAudioDelta {
		return VoiceEvent{}, false, nil
	}
	delta, _ := raw["delta"].(string)
	if _, err := base64.StdEncoding.DecodeString(delta); err != nil || delta == "" {
		return VoiceEvent{}, false, fmt.Errorf("openai realtime output audio delta must be valid base64")
	}
	streamID, _ := raw["response_id"].(string)
	if streamID == "" {
		streamID, _ = raw["item_id"].(string)
	}
	if streamID == "" {
		streamID = "a21-openai-realtime-output"
	}
	return VoiceEvent{
		Session:  session,
		Kind:     VoiceEventSpeaking,
		StreamID: streamID,
		Audio: &VoiceAudioChunk{
			Codec:        string(protocol.AudioCodecPCMS16LE),
			SampleRateHz: 24000,
			Channels:     1,
			DataBase64:   delta,
		},
	}, true, nil
}

func openAIRealtimeWriteEvent(ctx context.Context, session *RealtimeWebSocketSession, event map[string]any) error {
	if session == nil || session.conn == nil {
		return fmt.Errorf("openai realtime session is not connected")
	}
	if event["type"] == "" {
		return fmt.Errorf("openai realtime event type is required")
	}
	return session.conn.WriteJSON(ctx, event)
}

func openAIRealtimeSupportedSampleRate(sampleRate int) bool {
	switch sampleRate {
	case 16000, 24000, 48000:
		return true
	default:
		return false
	}
}
