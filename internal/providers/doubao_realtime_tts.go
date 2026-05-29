package providers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"a21.local/a21/internal/protocol"
)

const (
	doubaoRealtimeTTSSessionUpdate = "tts_session.update"
	doubaoRealtimeTTSInputAppend   = "input_text.append"
	doubaoRealtimeTTSInputDone     = "input_text.done"
	doubaoRealtimeTTSAudioDelta    = "response.audio.delta"
)

type DoubaoRealtimeTTSConfig struct {
	Model                 string
	Voice                 string
	OutputAudioFormat     string
	OutputAudioSampleRate int
}

func DoubaoRealtimeTTSSessionUpdateEvent(config DoubaoRealtimeTTSConfig) (map[string]any, error) {
	model := strings.TrimSpace(config.Model)
	if model == "" {
		return nil, fmt.Errorf("doubao realtime TTS model is required")
	}
	voice := strings.TrimSpace(config.Voice)
	if voice == "" {
		return nil, fmt.Errorf("doubao realtime TTS voice is required")
	}
	format := strings.TrimSpace(config.OutputAudioFormat)
	if format == "" {
		format = "pcm"
	}
	sampleRate := config.OutputAudioSampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}
	if !doubaoRealtimeTTSSupportedSampleRate(sampleRate) {
		return nil, fmt.Errorf("doubao realtime TTS sample rate %d is unsupported", sampleRate)
	}
	return map[string]any{
		"type": doubaoRealtimeTTSSessionUpdate,
		"session": map[string]any{
			"voice":                    voice,
			"output_audio_format":      format,
			"output_audio_sample_rate": sampleRate,
			"text_to_speech": map[string]any{
				"model": model,
			},
		},
	}, nil
}

func DoubaoRealtimeTTSInputTextAppendEvent(delta string) (map[string]any, error) {
	if strings.TrimSpace(delta) == "" {
		return nil, fmt.Errorf("doubao realtime TTS text delta is required")
	}
	return map[string]any{
		"type":  doubaoRealtimeTTSInputAppend,
		"delta": delta,
	}, nil
}

func DoubaoRealtimeTTSInputTextDoneEvent() map[string]any {
	return map[string]any{"type": doubaoRealtimeTTSInputDone}
}

func DoubaoRealtimeTTSServerEventToVoiceEvent(session VoiceSession, outputSampleRate int, raw map[string]any) (VoiceEvent, bool, error) {
	eventType, _ := raw["type"].(string)
	if eventType != doubaoRealtimeTTSAudioDelta {
		return VoiceEvent{}, false, nil
	}
	delta, _ := raw["delta"].(string)
	if _, err := base64.StdEncoding.DecodeString(delta); err != nil || delta == "" {
		return VoiceEvent{}, false, fmt.Errorf("doubao realtime TTS output audio delta must be valid base64")
	}
	if outputSampleRate == 0 {
		outputSampleRate = 16000
	}
	if !doubaoRealtimeTTSSupportedSampleRate(outputSampleRate) {
		return VoiceEvent{}, false, fmt.Errorf("doubao realtime TTS sample rate %d is unsupported", outputSampleRate)
	}
	streamID, _ := raw["item_id"].(string)
	if streamID == "" {
		streamID = "a21-doubao-tts-output"
	}
	return VoiceEvent{
		Session:  session,
		Kind:     VoiceEventSpeaking,
		StreamID: streamID,
		Audio: &VoiceAudioChunk{
			Codec:        string(protocol.AudioCodecPCMS16LE),
			SampleRateHz: outputSampleRate,
			Channels:     1,
			DataBase64:   delta,
		},
	}, true, nil
}

func DoubaoRealtimeTTSSendSessionUpdate(ctx context.Context, session *RealtimeWebSocketSession, config DoubaoRealtimeTTSConfig) error {
	event, err := DoubaoRealtimeTTSSessionUpdateEvent(config)
	if err != nil {
		return err
	}
	return doubaoRealtimeTTSWriteEvent(ctx, session, event)
}

func DoubaoRealtimeTTSSendText(ctx context.Context, session *RealtimeWebSocketSession, delta string) error {
	event, err := DoubaoRealtimeTTSInputTextAppendEvent(delta)
	if err != nil {
		return err
	}
	return doubaoRealtimeTTSWriteEvent(ctx, session, event)
}

func DoubaoRealtimeTTSSendTextDone(ctx context.Context, session *RealtimeWebSocketSession) error {
	return doubaoRealtimeTTSWriteEvent(ctx, session, DoubaoRealtimeTTSInputTextDoneEvent())
}

func doubaoRealtimeTTSWriteEvent(ctx context.Context, session *RealtimeWebSocketSession, event map[string]any) error {
	if session == nil || session.conn == nil {
		return fmt.Errorf("doubao realtime TTS session is not connected")
	}
	if event["type"] == "" {
		return fmt.Errorf("doubao realtime TTS event type is required")
	}
	return session.conn.WriteJSON(ctx, event)
}

func doubaoRealtimeTTSSupportedSampleRate(sampleRate int) bool {
	switch sampleRate {
	case 16000, 24000, 48000:
		return true
	default:
		return false
	}
}
