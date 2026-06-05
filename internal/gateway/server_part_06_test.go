package gateway

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
)

func TestFastCompanionVoicePipelineRecordsDefaultVoiceProfileAndPlaybackStart(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "selected provider should not run", Final: true},
		},
	}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider, V21Client: &countingV21Client{}})
	runner := newRecordingXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	handler := server.Handler()
	frameBase64 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1, 0}, 960))
	req := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", bytes.NewBufferString(fmt.Sprintf(`{
		"device_id":"stackchan-sim-001",
		"mode":"roleplay",
		"trace_id":"a21-trace-fast-pipeline-default-voice",
		"session_id":"a21-session-fast-pipeline-default-voice",
		"local_audio":{
			"asr_provider":"mock_asr",
			"first_partial_ms":42,
			"final_transcript_chars":11,
			"frames":[{"seq":1,"codec":"pcm_s16le","sample_rate_hz":16000,"channels":1,"duration_ms":60,"byte_count":1920,"rms":0.04,"data_base64":%q}]
		}
	}`, frameBase64)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var captured providers.VoicePipelineRequest
	select {
	case captured = <-runner.requests:
	case <-time.After(time.Second):
		t.Fatal("voice pipeline runner did not capture request")
	}
	if captured.VoiceCloneProfile != DefaultVoiceCloneProfile {
		t.Fatalf("captured voice profile = %q, want default %q", captured.VoiceCloneProfile, DefaultVoiceCloneProfile)
	}
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-fast-pipeline-default-voice", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
	}
	for _, want := range []string{
		"roleplay.voice_clone_profile.used",
		"audio.downlink.first_frame",
		"device.playback.start",
	} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
}

func TestFastCompanionVoicePipelineRecordsProviderFallbackObservability(t *testing.T) {
	server := NewServer()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return fallbackReportingXiaozhiPipelineRunner{}
	}
	handler := server.Handler()
	frameBase64 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1, 0}, 960))
	req := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", bytes.NewBufferString(fmt.Sprintf(`{
		"device_id":"stackchan-sim-001",
		"mode":"workmate",
		"trace_id":"a21-trace-fast-fallback",
		"session_id":"a21-session-fast-fallback",
		"local_audio":{
			"asr_provider":"mock_asr",
			"first_partial_ms":42,
			"final_transcript_chars":11,
			"frames":[{"seq":1,"codec":"pcm_s16le","sample_rate_hz":16000,"channels":1,"duration_ms":60,"byte_count":1920,"rms":0.04,"data_base64":%q}]
		}
	}`, frameBase64)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Status                     string `json:"status"`
		TextStreamProvider         string `json:"text_stream_provider"`
		TextStreamFallbackUsed     bool   `json:"text_stream_fallback_used"`
		TextStreamFallbackProvider string `json:"text_stream_fallback_provider"`
		TextStreamFallbackReason   string `json:"text_stream_fallback_reason"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "pipeline_completed" ||
		response.TextStreamProvider != "a21_voice_fallback" ||
		!response.TextStreamFallbackUsed ||
		response.TextStreamFallbackProvider != "a21_voice_fallback" ||
		response.TextStreamFallbackReason != "primary_failed" {
		t.Fatalf("fallback response = %+v", response)
	}
	for _, forbidden := range []string{"fallback voice answer", "fallback reasoning", frameBase64, "http://", "https://", "/Users/", "sk-"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("fast companion fallback response leaked %q: %s", forbidden, rec.Body.String())
		}
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-fast-fallback", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"fallback.used", "provider.failover", "fast_companion.voice_pipeline.completed"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	for _, forbidden := range []string{"fallback voice answer", "fallback reasoning", "http://", "https://", "/Users/", "sk-"} {
		if strings.Contains(traceRec.Body.String(), forbidden) {
			t.Fatalf("fallback trace leaked %q: %s", forbidden, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{"a21_provider_failover_total 1", "a21_fallback_total 1"} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestFastCompanionVoicePipelineUnavailableEntersLocalFallbackState(t *testing.T) {
	server := NewServer()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return unavailableXiaozhiPipelineRunner{}
	}
	handler := server.Handler()
	frameBase64 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1, 0}, 960))
	req := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", bytes.NewBufferString(fmt.Sprintf(`{
		"device_id":"stackchan-sim-001",
		"mode":"workmate",
		"trace_id":"a21-trace-fast-local-fallback",
		"session_id":"a21-session-fast-local-fallback",
		"local_audio":{
			"asr_provider":"mock_asr",
			"first_partial_ms":42,
			"final_transcript_chars":11,
			"frames":[{"seq":1,"codec":"pcm_s16le","sample_rate_hz":16000,"channels":1,"duration_ms":60,"byte_count":1920,"rms":0.04,"data_base64":%q}]
		}
	}`, frameBase64)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Status string              `json:"status"`
		Mode   protocol.Mode       `json:"mode"`
		Events []protocol.Envelope `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "local_fallback" || response.Mode != protocol.ModeLocalFallback {
		t.Fatalf("response status/mode = %s/%s, want local_fallback", response.Status, response.Mode)
	}
	var fallback protocol.ControlEventPayload
	for _, event := range response.Events {
		if event.Kind != protocol.KindControlEvent {
			continue
		}
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.State == protocol.ExpressionLocalFallback {
			fallback = payload
		}
	}
	if fallback.State != protocol.ExpressionLocalFallback ||
		fallback.Mode != protocol.ModeLocalFallback ||
		!fallback.Final ||
		!strings.Contains(fallback.Text, "外部大脑连不上") ||
		!strings.Contains(fallback.Text, "我还在") {
		t.Fatalf("fallback payload = %+v", fallback)
	}
	for _, forbidden := range []string{frameBase64, "fixture transcript", "provider output", "http://", "https://", "/Users/", "sk-"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("local fallback response leaked %q: %s", forbidden, rec.Body.String())
		}
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-fast-local-fallback", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"fast_companion.voice_pipeline.unavailable", "local_fallback.entered", "control.local_fallback.sent"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	if !strings.Contains(metricsRec.Body.String(), "a21_fallback_total 1") {
		t.Fatalf("metrics missing local fallback count:\n%s", metricsRec.Body.String())
	}
	if strings.Contains(metricsRec.Body.String(), "a21_provider_failover_total 1") {
		t.Fatalf("local fallback incremented provider failover count:\n%s", metricsRec.Body.String())
	}

	devicesReq := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
	devicesRec := httptest.NewRecorder()
	handler.ServeHTTP(devicesRec, devicesReq)
	for _, want := range []string{`"current_mode":"local_fallback"`, `"current_expression":"local_fallback"`} {
		if !strings.Contains(devicesRec.Body.String(), want) {
			t.Fatalf("devices missing %q: %s", want, devicesRec.Body.String())
		}
	}
}

func TestFastCompanionHybridRejectsUnsupportedModesAndMissingLocalAudio(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "should not run", Final: true},
		},
	}
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{
		VoiceProvider: provider,
		V21Client:     v21,
	})
	handler := server.Handler()
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			name: "unsupported mode",
			body: `{"device_id":"stackchan-sim-001","mode":"public","local_audio":{"asr_provider":"mock_asr"}}`,
		},
		{
			name: "missing local audio provider",
			body: `{"device_id":"stackchan-sim-001","mode":"workmate","local_audio":{"first_partial_ms":42}}`,
		},
		{
			name: "negative local audio counters",
			body: `{"device_id":"stackchan-sim-001","mode":"companion","local_audio":{"asr_provider":"mock_asr","first_partial_ms":-1}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
			}
			if provider.startCalls != 0 {
				t.Fatalf("provider start calls = %d, want 0", provider.startCalls)
			}
			if v21.calls != 0 {
				t.Fatalf("v21 calls = %d, want 0", v21.calls)
			}
		})
	}
}

func TestFastCompanionHybridKeepsProfessionalRealtimeOutAndV21EvidenceIn(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "should not run", Final: true, StreamID: "rt-stream-pro"},
		},
	}
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider, V21Client: v21})
	handler := server.Handler()

	realtimeReq := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", bytes.NewBufferString(`{"device_id":"stackchan-001","text":"查一下证据","mode":"professional","trace_id":"a21-trace-pro-boundary","session_id":"a21-session-pro-boundary"}`))
	realtimeRec := httptest.NewRecorder()
	handler.ServeHTTP(realtimeRec, realtimeReq)

	if realtimeRec.Code != http.StatusBadRequest {
		t.Fatalf("realtime status = %d, want 400: %s", realtimeRec.Code, realtimeRec.Body.String())
	}
	if provider.startCalls != 0 {
		t.Fatalf("provider start calls = %d, want 0", provider.startCalls)
	}

	proReq := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下证据","mode":"professional","trace_id":"a21-trace-pro-boundary","session_id":"a21-session-pro-boundary"}`))
	proRec := httptest.NewRecorder()
	handler.ServeHTTP(proRec, proReq)

	if proRec.Code != http.StatusOK {
		t.Fatalf("professional status = %d, want 200: %s", proRec.Code, proRec.Body.String())
	}
	if v21.calls != 1 {
		t.Fatalf("v21 calls = %d, want 1", v21.calls)
	}
	var response MockTurnResponse
	if err := json.Unmarshal(proRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var answer protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[len(response.Events)-1].Payload, &answer); err != nil {
		t.Fatal(err)
	}
	if answer.Mode != protocol.ModeProfessional || answer.State != protocol.ExpressionSpeaking || len(answer.Evidence) == 0 {
		t.Fatalf("professional answer = %+v", answer)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-boundary", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"v21.query.start", "v21.query.first_result"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
}

func TestRealtimeSessionStartUsesVoiceProviderAndRecordsMetrics(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		VoiceProvider: scriptedVoiceProvider{
			events: []providers.VoiceEvent{
				{Kind: providers.VoiceEventThinking, Text: "provider realtime thinking"},
				{Kind: providers.VoiceEventSpeaking, Text: "provider realtime speaking", Final: true, StreamID: "rt-stream-001"},
			},
		},
	})
	handler := server.Handler()
	body := bytes.NewBufferString(`{"device_id":"stackchan-001","text":"先说，我在","mode":"workmate","trace_id":"a21-trace-rt-001","session_id":"a21-session-rt-001"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response RealtimeSessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Provider != "scripted" || response.Status != "completed" {
		t.Fatalf("response provider/status = %q/%q", response.Provider, response.Status)
	}
	if response.TraceID != "a21-trace-rt-001" || response.SessionID != "a21-session-rt-001" || response.DeviceID != "stackchan-001" {
		t.Fatalf("response identity = %+v", response)
	}
	if len(response.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(response.Events))
	}
	var speaking protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[1].Payload, &speaking); err != nil {
		t.Fatal(err)
	}
	if speaking.State != protocol.ExpressionSpeaking || speaking.StreamID != "rt-stream-001" {
		t.Fatalf("speaking payload = %+v", speaking)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-rt-001", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"realtime.session.start.received", "provider.start_turn.start", "provider.start_turn.first_event", "provider.start_turn.end"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{"a21_realtime_session_total 1", "a21_voice_provider_start_turn_ms_count 1"} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestRealtimeSessionStartMapsProviderAudioToPlaybackChunk(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		VoiceProvider: scriptedVoiceProvider{
			events: []providers.VoiceEvent{
				{
					Kind:     providers.VoiceEventSpeaking,
					Final:    true,
					StreamID: "rt-audio-stream-001",
					Audio: &providers.VoiceAudioChunk{
						Codec:        string(protocol.AudioCodecPCMS16LE),
						SampleRateHz: 16000,
						Channels:     1,
						DurationMS:   20,
						DataBase64:   "AAAA",
					},
				},
			},
		},
	})
	handler := server.Handler()
	req := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", bytes.NewBufferString(`{"device_id":"stackchan-001","text":"播一小段","mode":"workmate","trace_id":"a21-trace-rt-audio","session_id":"a21-session-rt-audio"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response RealtimeSessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Events) != 2 {
		t.Fatalf("events = %d, want control plus playback: %+v", len(response.Events), response.Events)
	}
	if response.Events[0].Kind != protocol.KindControlEvent {
		t.Fatalf("event 0 kind = %q, want control", response.Events[0].Kind)
	}
	if response.Events[1].Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("event 1 kind = %q, want audio playback", response.Events[1].Kind)
	}
	var playback protocol.AudioPlaybackChunk
	if err := json.Unmarshal(response.Events[1].Payload, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.StreamID != "rt-audio-stream-001" || playback.Codec != protocol.AudioCodecPCMS16LE || playback.SampleRateHz != 16000 || playback.Channels != 1 || playback.DurationMS != 20 || playback.DataBase64 != "AAAA" {
		t.Fatalf("playback = %+v", playback)
	}
	if response.Events[1].TraceID != "a21-trace-rt-audio" || response.Events[1].SessionID != "a21-session-rt-audio" {
		t.Fatalf("playback trace/session = %q/%q", response.Events[1].TraceID, response.Events[1].SessionID)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-rt-audio", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if !strings.Contains(traceRec.Body.String(), "audio.playback.chunk.sent") {
		t.Fatalf("trace missing audio playback marker: %s", traceRec.Body.String())
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	if !strings.Contains(metricsRec.Body.String(), "a21_audio_playback_chunk_total 1") {
		t.Fatalf("metrics missing playback count:\n%s", metricsRec.Body.String())
	}
}

func TestRealtimeSessionCancelUsesActiveStreamAndRecordsMetrics(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "provider realtime speaking", Final: true, StreamID: "rt-stream-002"},
		},
	}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider})
	handler := server.Handler()
	startReq := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", bytes.NewBufferString(`{"device_id":"stackchan-001","text":"先说，我在","mode":"workmate","trace_id":"a21-trace-rt-002","session_id":"a21-session-rt-002"}`))
	handler.ServeHTTP(httptest.NewRecorder(), startReq)

	cancelReq := httptest.NewRequest(http.MethodPost, "/v1/realtime/session/cancel", bytes.NewBufferString(`{"device_id":"stackchan-001","mode":"workmate","trace_id":"a21-trace-rt-002","session_id":"a21-session-rt-002","reason":"barge_in"}`))
	cancelRec := httptest.NewRecorder()
	handler.ServeHTTP(cancelRec, cancelReq)

	if cancelRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", cancelRec.Code, cancelRec.Body.String())
	}
	if provider.cancelRequest.StreamID != "rt-stream-002" {
		t.Fatalf("cancel stream = %q, want active stream", provider.cancelRequest.StreamID)
	}
	if provider.cancelRequest.Reason != providers.CancelBargeIn {
		t.Fatalf("cancel reason = %q", provider.cancelRequest.Reason)
	}
	var response RealtimeSessionResponse
	if err := json.Unmarshal(cancelRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "cancelled" || len(response.Events) != 1 {
		t.Fatalf("response = %+v", response)
	}
	var interrupted protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[0].Payload, &interrupted); err != nil {
		t.Fatal(err)
	}
	if interrupted.State != protocol.ExpressionInterrupted || interrupted.StreamID != "rt-stream-002" {
		t.Fatalf("interrupted payload = %+v", interrupted)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-rt-002", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"realtime.session.cancel.received", "provider.cancel.start", "provider.cancel.end"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{"a21_realtime_session_cancel_total 1", "a21_voice_provider_cancel_ms_count 1"} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestRealtimeSessionStartRejectsProfessionalModeWithoutCallingProviders(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "should not run", Final: true, StreamID: "rt-stream-pro"},
		},
	}
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider, V21Client: v21})
	req := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", bytes.NewBufferString(`{"device_id":"stackchan-001","text":"查一下证据","mode":"professional"}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	if provider.startCalls != 0 {
		t.Fatalf("provider start calls = %d, want 0", provider.startCalls)
	}
	if v21.calls != 0 {
		t.Fatalf("v21 calls = %d, want 0", v21.calls)
	}
	if !strings.Contains(rec.Body.String(), "professional path") {
		t.Fatalf("body = %q, want professional path guidance", rec.Body.String())
	}
}

func TestProfessionalModeUsesV21AdapterEvidence(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		V21Client: v21adapter.NewMockClient(),
	})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-001","session_id":"a21-session-pro-001"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Events) != 4 {
		t.Fatalf("events = %d, want 4", len(response.Events))
	}
	var pro protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[1].Payload, &pro); err != nil {
		t.Fatal(err)
	}
	if pro.State != protocol.ExpressionProfessional || pro.Mode != protocol.ModeProfessional {
		t.Fatalf("professional payload = %+v", pro)
	}
	var checking protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[2].Payload, &checking); err != nil {
		t.Fatal(err)
	}
	if checking.State != protocol.ExpressionThinking || checking.Mode != protocol.ModeProfessional || !strings.Contains(checking.Text, "我在查") {
		t.Fatalf("checking payload = %+v", checking)
	}
	var answer protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[3].Payload, &answer); err != nil {
		t.Fatal(err)
	}
	if answer.State != protocol.ExpressionSpeaking || answer.Mode != protocol.ModeProfessional {
		t.Fatalf("answer payload = %+v", answer)
	}
	if answer.Text == "" || len(answer.Evidence) == 0 || len(answer.ScreenCards) == 0 {
		t.Fatalf("professional answer missing evidence fields: %+v", answer)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-001", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
	}
	if !strings.Contains(traceRec.Body.String(), "v21.query.start") || !strings.Contains(traceRec.Body.String(), "v21.query.first_result") {
		t.Fatalf("trace missing v21 markers: %s", traceRec.Body.String())
	}
}

func TestProfessionalModeEmitsCheckingFeedbackBeforeV21Query(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		V21Client: v21adapter.NewMockClient(),
	})
	handler := server.Handler()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-checking","session_id":"a21-session-pro-checking"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	checkingEventIndex := -1
	for i, event := range response.Events {
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Mode == protocol.ModeProfessional && payload.State == protocol.ExpressionThinking && strings.Contains(payload.Text, "我在查") {
			checkingEventIndex = i
			break
		}
	}
	if checkingEventIndex < 0 {
		t.Fatalf("professional response missing checking feedback event: %+v", response.Events)
	}
	if checkingEventIndex >= len(response.Events)-1 {
		t.Fatalf("checking feedback must be before final evidence answer: index=%d events=%d", checkingEventIndex, len(response.Events))
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-checking", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.Unmarshal(traceRec.Body.Bytes(), &traces); err != nil {
		t.Fatal(err)
	}
	checkingAt, ok := traceEventAtMS(traces.Events, "professional.checking_feedback.sent")
	if !ok {
		t.Fatalf("trace missing professional.checking_feedback.sent: %s", traceRec.Body.String())
	}
	v21StartAt, ok := traceEventAtMS(traces.Events, "v21.query.start")
	if !ok {
		t.Fatalf("trace missing v21.query.start: %s", traceRec.Body.String())
	}
	if checkingAt > v21StartAt {
		t.Fatalf("checking feedback at %d must be before v21 query start at %d", checkingAt, v21StartAt)
	}
	if v21StartAt-checkingAt > 1200 {
		t.Fatalf("checking feedback to v21 start = %dms, want <=1200ms", v21StartAt-checkingAt)
	}
}

func TestProfessionalModeSendsExplicitV21PlaceholderContract(t *testing.T) {
	v21 := &capturingV21Client{}
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
	handler := server.Handler()
	workspaceReq := httptest.NewRequest(http.MethodPost, "/v1/professional-workspace", bytes.NewBufferString(`{"user_id":"a21_user_contract","workspace_id":"a21_workspace_contract","query_scope":"personal_plus_public"}`))
	workspaceRec := httptest.NewRecorder()
	handler.ServeHTTP(workspaceRec, workspaceReq)
	if workspaceRec.Code != http.StatusOK {
		t.Fatalf("workspace status = %d: %s", workspaceRec.Code, workspaceRec.Body.String())
	}
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下证据","mode":"professional","trace_id":"a21-trace-pro-contract","session_id":"a21-session-pro-contract"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if v21.request.Mode != "professional" ||
		v21.request.PrivacyScope != "professional_only" ||
		v21.request.DeviceID != "stackchan-sim-001" ||
		v21.request.UserID != "a21_user_contract" ||
		v21.request.WorkspaceID != "a21_workspace_contract" ||
		v21.request.QueryScope != "personal_plus_public" ||
		v21.request.LatencyProfile != "fast_first" ||
		v21.request.AnswerStyle != "voice_first_with_citations" ||
		v21.request.MaxFirstResponseMS != 1200 {
		t.Fatalf("gateway sent incomplete V21 professional placeholder contract: %+v", v21.request)
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var answer protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[len(response.Events)-1].Payload, &answer); err != nil {
		t.Fatal(err)
	}
	if len(answer.Evidence) != 1 || len(answer.ScreenCards) != 1 || len(answer.FollowUps) != 1 {
		t.Fatalf("professional response missing evidence/cards/follow-ups: %+v", answer)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-contract", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	var traces TraceResponse
	if err := json.Unmarshal(traceRec.Body.Bytes(), &traces); err != nil {
		t.Fatal(err)
	}
	checkingAt, ok := traceEventAtMS(traces.Events, "professional.checking_feedback.sent")
	if !ok {
		t.Fatalf("trace missing checking feedback marker: %s", traceRec.Body.String())
	}
	v21StartAt, ok := traceEventAtMS(traces.Events, "v21.query.start")
	if !ok {
		t.Fatalf("trace missing V21 start marker: %s", traceRec.Body.String())
	}
	for _, want := range []string{"professional.workspace.ready", "professional.query_scope.personal_plus_public"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	for _, forbidden := range []string{"a21_user_contract", "a21_workspace_contract", "查一下证据"} {
		if strings.Contains(traceRec.Body.String(), forbidden) {
			t.Fatalf("trace leaked %q: %s", forbidden, traceRec.Body.String())
		}
	}
	if v21StartAt-checkingAt > 1200 {
		t.Fatalf("placeholder boundary = %dms, want <=1200ms", v21StartAt-checkingAt)
	}
}

func TestProfessionalVoiceTriggerRoutesMockTurnToProfessionalPath(t *testing.T) {
	v21 := &capturingV21Client{}
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"认真查一下座舱报警证据","mode":"workmate","trace_id":"a21-trace-pro-trigger","session_id":"a21-session-pro-trigger"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if v21.request.Mode != "professional" || v21.request.Utterance != "认真查一下座舱报警证据" {
		t.Fatalf("v21 request = %+v, want professional trigger utterance", v21.request)
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var answer protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[len(response.Events)-1].Payload, &answer); err != nil {
		t.Fatal(err)
	}
	if answer.Mode != protocol.ModeProfessional || len(answer.Evidence) == 0 || len(answer.ScreenCards) == 0 {
		t.Fatalf("trigger answer = %+v, want professional evidence response", answer)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-trigger", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"professional.voice_trigger.detected", "professional.checking_feedback.sent", "v21.query.start"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	if strings.Contains(traceRec.Body.String(), "认真查一下座舱报警证据") {
		t.Fatalf("trace leaked trigger utterance: %s", traceRec.Body.String())
	}

	recordsReq := httptest.NewRequest(http.MethodGet, "/v1/professional-read-records?trace_id=a21-trace-pro-trigger", nil)
	recordsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(recordsRec, recordsReq)
	var records ProfessionalReadRecordsResponse
	if err := json.Unmarshal(recordsRec.Body.Bytes(), &records); err != nil {
		t.Fatal(err)
	}
	if len(records.Records) != 1 || records.Records[0].Status != "completed" {
		t.Fatalf("read records = %+v, want one completed trigger record", records)
	}
}

func TestProfessionalVoiceTriggerClassifierKeepsNegatedTextOutOfV21(t *testing.T) {
	for _, text := range []string{"不要进专业检索", "不用专业模式", "别查 V21", "普通聊一下座舱报警"} {
		if professionalVoiceTrigger(text) {
			t.Fatalf("professionalVoiceTrigger(%q) = true, want false", text)
		}
	}
	for _, text := range []string{"专业模式", "认真查一下座舱报警", "帮我查 V21 座舱反馈", "给我证据"} {
		if !professionalVoiceTrigger(text) {
			t.Fatalf("professionalVoiceTrigger(%q) = false, want true", text)
		}
	}
}

func TestWorkmateModeDoesNotCallV21Adapter(t *testing.T) {
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"先说，我在","mode":"workmate"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if v21.calls != 0 {
		t.Fatalf("v21 calls = %d, want 0", v21.calls)
	}
}

func TestOfficePrivacyModesDoNotCallV21Adapter(t *testing.T) {
	for _, mode := range []protocol.Mode{protocol.ModeWorkmate, protocol.ModePublic, protocol.ModePrivate, protocol.ModeFocus, protocol.ModeMuted} {
		t.Run(string(mode), func(t *testing.T) {
			v21 := &countingV21Client{}
			server := NewServerWithOptions(ServerOptions{V21Client: v21})
			body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"不要进专业检索","mode":"` + string(mode) + `"}`)
			req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
			rec := httptest.NewRecorder()

			server.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
			}
			if v21.calls != 0 {
				t.Fatalf("v21 calls = %d, want 0", v21.calls)
			}
		})
	}
	for _, mode := range []protocol.Mode{protocol.ModePublic, protocol.ModePrivate, protocol.ModeFocus, protocol.ModeMuted} {
		t.Run(string(mode)+"_positive_trigger_blocked", func(t *testing.T) {
			v21 := &countingV21Client{}
			server := NewServerWithOptions(ServerOptions{V21Client: v21})
			body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"给我证据","mode":"` + string(mode) + `"}`)
			req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
			rec := httptest.NewRecorder()

			server.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
			}
			if v21.calls != 0 {
				t.Fatalf("v21 calls = %d, want 0", v21.calls)
			}
		})
	}
}
