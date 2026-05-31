package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/audio/opuscodec"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestHealthz(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"service":"a21-gateway"`)) {
		t.Fatalf("body missing service: %s", rec.Body.String())
	}
}

func TestVoiceProviderHealthEndpoint(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		VoiceProvider: scriptedVoiceProvider{events: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "ready", Final: true},
		}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/providers/voice/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var health providers.VoiceProviderHealth
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health.Provider != "scripted" || health.Status != providers.VoiceProviderHealthy || !health.Realtime || !health.Configured {
		t.Fatalf("health = %+v", health)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"provider":"scripted"`)) {
		t.Fatalf("body missing provider json field: %s", rec.Body.String())
	}
}

func TestVoiceProviderHealthEndpointReturnsUnavailable(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{VoiceProvider: unavailableVoiceProvider{}})
	req := httptest.NewRequest(http.MethodGet, "/v1/providers/voice/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", rec.Code, rec.Body.String())
	}
	var health providers.VoiceProviderHealth
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health.Provider != "unavailable" || health.Status != providers.VoiceProviderUnavailable {
		t.Fatalf("health = %+v", health)
	}
}

func TestSimulatorPageServed(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/simulator", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content-type = %q, want text/html", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	for _, want := range []string{
		"A21 Device Simulator",
		`data-testid="simulator-root"`,
		"/ws/control",
		"/v1/devices",
		"/v1/traces",
		"Device Registry",
		`id="registryConnection"`,
		`id="registryMode"`,
		`id="registryExpression"`,
		"Waterfall",
		"Latency Summary",
		"Professional Evidence",
		"Audio Link",
		"Office Visibility",
		`id="visibilityBadge"`,
		`id="privacyBadge"`,
		`id="listeningBadge"`,
		"PUBLIC",
		"PRIVATE",
		"PRO",
		"MUTED",
		"updateVisibilityBadges",
		`id="startMic"`,
		`id="stopMic"`,
		`id="mockAudioBurst"`,
		`id="audioFramesSent"`,
		`id="playbackState"`,
		`id="playbackChunksReceived"`,
		`id="playbackBufferedChunks"`,
		`id="playbackStream"`,
		`id="playbackScheduledChunks"`,
		`id="latencyAudioPlayback"`,
		`id="latencyV21"`,
		`id="latencyBargeIn"`,
		`id="latencyProviderFirstAudio"`,
		"renderLatencySummary",
		"startMicrophoneStream",
		"stopMicrophoneStream",
		"sendMockAudioBurst",
		"handleAudioPlaybackChunk",
		"decodePCM16Base64",
		"schedulePCMPlayback",
		"stopScheduledPlayback",
		"audio.playback.chunk",
		`id="professionalEvidence"`,
		"firmware_id",
		"a21-stackchan",
		"m5stack-cores3",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
}

func TestMockTurnReturnsDeterministicStateSequence(t *testing.T) {
	server := NewServer()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"先说，我在","mode":"workmate"}`)
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
	if response.TraceID != "a21-trace-000001" {
		t.Fatalf("TraceID = %q", response.TraceID)
	}
	if len(response.Events) != 3 {
		t.Fatalf("events = %d, want 3", len(response.Events))
	}
	wantStates := []protocol.ExpressionState{
		protocol.ExpressionListening,
		protocol.ExpressionThinking,
		protocol.ExpressionSpeaking,
	}
	for i, want := range wantStates {
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(response.Events[i].Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.State != want {
			t.Fatalf("event %d state = %q, want %q", i, payload.State, want)
		}
		if response.Events[i].TraceID != response.TraceID {
			t.Fatalf("event %d trace = %q, want %q", i, response.Events[i].TraceID, response.TraceID)
		}
	}
}

func TestTraceEndpointRecordsMockTurnWaterfall(t *testing.T) {
	server := NewServer()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"先说，我在","mode":"workmate","trace_id":"a21-trace-test-001","session_id":"a21-session-test-001"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-test-001", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
	}
	var response TraceResponse
	if err := json.Unmarshal(traceRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.TraceID != "a21-trace-test-001" {
		t.Fatalf("trace id = %q", response.TraceID)
	}
	if len(response.Events) < 4 {
		t.Fatalf("events = %d, want at least 4: %+v", len(response.Events), response.Events)
	}
	wantNames := []string{
		"http.mock_turn.received",
		"control.listening.sent",
		"control.thinking.sent",
		"control.speaking.sent",
	}
	for i, want := range wantNames {
		if response.Events[i].Name != want {
			t.Fatalf("event %d name = %q, want %q", i, response.Events[i].Name, want)
		}
		if response.Events[i].OffsetMS < 0 {
			t.Fatalf("event %d offset = %d, want non-negative", i, response.Events[i].OffsetMS)
		}
	}
}

func TestTraceEndpointReturnsLatencySummary(t *testing.T) {
	server := NewServer()
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "audio.frame.received", 1000)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "audio.playback.chunk.sent", 1123)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "v21.query.start", 2000)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "v21.query.first_result", 2456)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "barge_in.detected", 3000)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "playback.stop", 3033)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "provider.audio.commit", 4000)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "provider.audio.first_downlink", 4088)
	req := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-summary-001", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response TraceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Summary.EventCount != 8 || response.Summary.LastOffsetMS != 3088 {
		t.Fatalf("summary shape = %+v", response.Summary)
	}
	if response.Summary.AudioFrameToPlaybackMS == nil || *response.Summary.AudioFrameToPlaybackMS != 123 {
		t.Fatalf("audio frame to playback = %v, want 123", response.Summary.AudioFrameToPlaybackMS)
	}
	if response.Summary.V21QueryFirstResultMS == nil || *response.Summary.V21QueryFirstResultMS != 456 {
		t.Fatalf("v21 first result = %v, want 456", response.Summary.V21QueryFirstResultMS)
	}
	if response.Summary.BargeInStopMS == nil || *response.Summary.BargeInStopMS != 33 {
		t.Fatalf("barge-in stop = %v, want 33", response.Summary.BargeInStopMS)
	}
	if response.Summary.ProviderCommitToFirstAudioMS == nil || *response.Summary.ProviderCommitToFirstAudioMS != 88 {
		t.Fatalf("provider commit to first audio = %v, want 88", response.Summary.ProviderCommitToFirstAudioMS)
	}
}

func TestTraceEndpointReturnsVoicePipelineSplitSummary(t *testing.T) {
	server := NewServer()
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "xiaozhi.listen.start", 1000)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "xiaozhi.opus_frame.received", 1010)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "xiaozhi.opus_frame.decoded", 1024)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "audio.ingress.buffered", 1030)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "asr.first_partial", 1140)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "provider.first_content", 1300)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "tts.first_audio", 1375)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "audio.downlink.first_frame", 1400)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "device.playback.start", 1460)

	req := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pipeline-001", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response TraceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	assertSummaryDelta(t, "xiaozhi_listen_to_audio_ingress_ms", response.Summary.XiaozhiListenToAudioIngressMS, 30)
	assertSummaryDelta(t, "xiaozhi_opus_decode_ms", response.Summary.XiaozhiOpusDecodeMS, 14)
	assertSummaryDelta(t, "asr_first_partial_ms", response.Summary.ASRFirstPartialMS, 110)
	assertSummaryDelta(t, "llm_first_content_ms", response.Summary.LLMFirstContentMS, 160)
	assertSummaryDelta(t, "tts_first_audio_ms", response.Summary.TTSFirstAudioMS, 75)
	assertSummaryDelta(t, "audio_downlink_first_frame_ms", response.Summary.AudioDownlinkFirstFrameMS, 25)
	assertSummaryDelta(t, "device_playback_start_ms", response.Summary.DevicePlaybackStartMS, 60)
	assertSummaryDelta(t, "answer_first_audio_total_ms", response.Summary.AnswerFirstAudioTotalMS, 370)
}

func TestTraceEndpointRequiresTraceID(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/traces", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestMockTurnUsesVoiceProviderEvents(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		VoiceProvider: scriptedVoiceProvider{
			events: []providers.VoiceEvent{
				{Kind: providers.VoiceEventThinking, Text: "provider thinking"},
				{Kind: providers.VoiceEventSpeaking, Text: "provider speaking", Final: true, StreamID: "provider-stream"},
			},
		},
	})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"custom","mode":"workmate"}`)
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
	var speaking protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[2].Payload, &speaking); err != nil {
		t.Fatal(err)
	}
	if speaking.Text != "provider speaking" {
		t.Fatalf("speaking text = %q, want provider speaking", speaking.Text)
	}
	if speaking.StreamID != "provider-stream" {
		t.Fatalf("stream = %q, want provider-stream", speaking.StreamID)
	}
}

func TestFastCompanionHybridRoutesLocalAudioFrontendToTextStreamBoundary(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "selected provider should not run", Final: true},
		},
	}
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider, V21Client: v21})
	handler := server.Handler()
	req := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", bytes.NewBufferString(`{
		"device_id":"stackchan-sim-001",
		"mode":"companion",
		"trace_id":"a21-trace-fast-hybrid-001",
		"session_id":"a21-session-fast-hybrid-001",
		"local_audio":{"asr_provider":"mock_asr","first_partial_ms":42,"final_transcript_chars":11}
	}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		TraceID            string              `json:"trace_id"`
		SessionID          string              `json:"session_id"`
		DeviceID           string              `json:"device_id"`
		Mode               protocol.Mode       `json:"mode"`
		Status             string              `json:"status"`
		Route              string              `json:"route"`
		AudioFrontend      string              `json:"audio_frontend"`
		TextStreamProvider string              `json:"text_stream_provider"`
		ProviderFamily     string              `json:"provider_family"`
		TextStreamExecuted bool                `json:"text_stream_executed"`
		Events             []protocol.Envelope `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.TraceID != "a21-trace-fast-hybrid-001" || response.SessionID != "a21-session-fast-hybrid-001" || response.DeviceID != "stackchan-sim-001" {
		t.Fatalf("response identity = %+v", response)
	}
	if response.Mode != protocol.ModeCompanion || response.Status != "boundary_ready" || response.Route != "fast_companion_hybrid" {
		t.Fatalf("response route = %+v", response)
	}
	if response.AudioFrontend != "local_audio" || response.ProviderFamily != "text_stream" || response.TextStreamProvider != "mock_text_stream" || response.TextStreamExecuted {
		t.Fatalf("provider boundary = %+v", response)
	}
	if len(response.Events) != 3 {
		t.Fatalf("events = %d, want 3", len(response.Events))
	}
	var speaking protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[2].Payload, &speaking); err != nil {
		t.Fatal(err)
	}
	if speaking.State != protocol.ExpressionSpeaking || speaking.Mode != protocol.ModeCompanion || speaking.StreamID != "a21-fast-companion-placeholder-stream" {
		t.Fatalf("speaking payload = %+v", speaking)
	}
	if provider.startCalls != 0 {
		t.Fatalf("voice provider start calls = %d, want 0", provider.startCalls)
	}
	if v21.calls != 0 {
		t.Fatalf("v21 calls = %d, want 0", v21.calls)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-fast-hybrid-001", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
	}
	var traceResponse TraceResponse
	if err := json.Unmarshal(traceRec.Body.Bytes(), &traceResponse); err != nil {
		t.Fatal(err)
	}
	foundASRFirstPartial := false
	for _, event := range traceResponse.Events {
		if event.Name == "asr.first_partial" {
			foundASRFirstPartial = true
			if event.OffsetMS != 42 {
				t.Fatalf("asr first partial offset = %d, want 42", event.OffsetMS)
			}
		}
	}
	if !foundASRFirstPartial {
		t.Fatalf("trace missing asr.first_partial event: %s", traceRec.Body.String())
	}
	for _, want := range []string{
		"fast_companion.local_audio.frontend.accepted",
		"asr.first_partial",
		"provider.text_stream.route.placeholder",
		"provider.first_byte",
		"provider.first_content",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"device.playback.start",
	} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
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
			body: `{"device_id":"stackchan-sim-001","mode":"roleplay","local_audio":{"asr_provider":"mock_asr"}}`,
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
	for _, mode := range []protocol.Mode{protocol.ModePublic, protocol.ModePrivate, protocol.ModeMuted} {
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
}

func TestProfessionalModeV21FailureIsHonestFallback(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{V21Client: failingV21Client{}})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-fail","session_id":"a21-session-pro-fail"}`)
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
	var fallback protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[len(response.Events)-1].Payload, &fallback); err != nil {
		t.Fatal(err)
	}
	if fallback.State != protocol.ExpressionError || fallback.Mode != protocol.ModeProfessional {
		t.Fatalf("fallback payload = %+v", fallback)
	}
	if !strings.Contains(fallback.Text, "V21 现在没接上") {
		t.Fatalf("fallback text = %q", fallback.Text)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-fail", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if !strings.Contains(traceRec.Body.String(), "v21.query.error") {
		t.Fatalf("trace missing v21 error marker: %s", traceRec.Body.String())
	}
}

func TestProfessionalModeRecordsV21QueryLatencyMetric(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{V21Client: v21adapter.NewMockClient()})
	handler := server.Handler()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)

	if metricsRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", metricsRec.Code)
	}
	bodyText := metricsRec.Body.String()
	for _, want := range []string{"a21_v21_query_ms_bucket", "a21_v21_query_ms_count 1"} {
		if !strings.Contains(bodyText, want) {
			t.Fatalf("metrics missing %q:\n%s", want, bodyText)
		}
	}
}

func TestProfessionalModeV21TimeoutCancelsQueryAndFallsBack(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		V21Client:  slowV21Client{delay: 200 * time.Millisecond},
		V21Timeout: 10 * time.Millisecond,
	})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-timeout","session_id":"a21-session-pro-timeout"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()
	start := time.Now()

	server.Handler().ServeHTTP(rec, req)

	if elapsed := time.Since(start); elapsed >= 150*time.Millisecond {
		t.Fatalf("request took %s, want timeout before slow V21 delay", elapsed)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var fallback protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[len(response.Events)-1].Payload, &fallback); err != nil {
		t.Fatal(err)
	}
	if fallback.State != protocol.ExpressionError || fallback.Mode != protocol.ModeProfessional {
		t.Fatalf("fallback payload = %+v", fallback)
	}
	if !strings.Contains(fallback.Text, "V21 现在没接上") {
		t.Fatalf("fallback text = %q", fallback.Text)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-timeout", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if !strings.Contains(traceRec.Body.String(), "v21.query.timeout") {
		t.Fatalf("trace missing v21 timeout marker: %s", traceRec.Body.String())
	}
}

func TestMockInterruptReturnsInterruptedThenListening(t *testing.T) {
	server := NewServer()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","trace_id":"a21-trace-000009","session_id":"a21-session-000009"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-interrupt", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(response.Events))
	}
	var first protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[0].Payload, &first); err != nil {
		t.Fatal(err)
	}
	if first.State != protocol.ExpressionInterrupted {
		t.Fatalf("first state = %q, want interrupted", first.State)
	}
}

func TestMetricsEndpointRecordsMockHTTPFlow(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	mockTurn := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", bytes.NewBufferString(`{"device_id":"stackchan-sim-001"}`))
	handler.ServeHTTP(httptest.NewRecorder(), mockTurn)
	interrupt := httptest.NewRequest(http.MethodPost, "/v1/mock-interrupt", bytes.NewBufferString(`{"device_id":"stackchan-sim-001"}`))
	handler.ServeHTTP(httptest.NewRecorder(), interrupt)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"a21_mock_turn_total 1", "a21_barge_in_total 1"} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics missing %q:\n%s", want, body)
		}
	}
}

func TestMetricsEndpointRecordsAudioFrame(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	audio, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-sim-001",
		Kind:     protocol.KindAudioFrame,
		Seq:      1,
		Payload:  audio,
	}); err != nil {
		t.Fatal(err)
	}
	readControlEvents(t, ctx, conn, 2)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("playback kind = %q, want %q", playback.Kind, protocol.KindAudioPlaybackChunk)
	}

	resp, err := http.Get(httpServer.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, want := range []string{"a21_audio_frame_total 1", "a21_audio_playback_chunk_total 1"} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics missing %q:\n%s", want, data)
		}
	}
}

func TestControlWebSocketMockTurn(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-sim-001",
		Kind:     protocol.KindDeviceEvent,
		Seq:      1,
	}, protocol.DeviceEventPayload{
		Event: protocol.DeviceEventMockTurn,
		Mode:  protocol.ModeWorkmate,
		Text:  "先说，我在",
	})

	events := readControlEvents(t, ctx, conn, 3)
	wantStates := []protocol.ExpressionState{
		protocol.ExpressionListening,
		protocol.ExpressionThinking,
		protocol.ExpressionSpeaking,
	}
	for i, want := range wantStates {
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.State != want {
			t.Fatalf("event %d state = %q, want %q", i, payload.State, want)
		}
		if events[i].TraceID != "a21-trace-000001" {
			t.Fatalf("event %d trace = %q, want a21-trace-000001", i, events[i].TraceID)
		}
	}
}

func TestControlWebSocketAcceptsArduinoClientHandshake(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, resp, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), &websocket.DialOptions{
		HTTPHeader:   http.Header{"Origin": []string{"file://"}},
		Subprotocols: []string{"arduino"},
	})
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("arduino websocket dial failed status=%d err=%v", status, err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })
	if conn.Subprotocol() != "arduino" {
		t.Fatalf("subprotocol = %q, want arduino", conn.Subprotocol())
	}

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       11,
		TraceID:   "a21-trace-arduino",
		SessionID: "a21-session-arduino",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventRuntimeEcho,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
		RuntimeEcho: map[string]string{
			"screen":  "local_fallback",
			"servo_y": "45deg",
			"rgb":     "#080808",
		},
	})

	resp, err = http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var registry DeviceRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(registry.Devices))
	}
	if registry.Devices[0].LastEvent != protocol.DeviceEventRuntimeEcho {
		t.Fatalf("last event = %q, want runtime echo", registry.Devices[0].LastEvent)
	}
}

func TestXiaozhiWebSocketHelloAcceptsStockProtocol(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"device_id": "stackchan-001",
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	if reply["type"] != "hello" || reply["transport"] != "websocket" {
		t.Fatalf("hello reply = %#v", reply)
	}
	if reply["device_id"] != "stackchan-001" {
		t.Fatalf("device_id = %#v, want stackchan-001", reply["device_id"])
	}
	sessionID, _ := reply["session_id"].(string)
	if !strings.HasPrefix(sessionID, "a21-session-") {
		t.Fatalf("session_id = %q, want generated A21 session", sessionID)
	}
	audio, ok := reply["audio"].(map[string]any)
	if !ok {
		t.Fatalf("reply audio = %#v", reply["audio"])
	}
	if audio["format"] != "opus" || audio["sample_rate"] != float64(24000) || audio["channels"] != float64(1) || audio["frame_duration"] != float64(60) {
		t.Fatalf("server audio = %#v", audio)
	}
	replyJSON, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"x21", "v21", "debug_metrics", "device_events"} {
		if strings.Contains(strings.ToLower(string(replyJSON)), forbidden) {
			t.Fatalf("hello reply leaked forbidden/debug field %q: %s", forbidden, replyJSON)
		}
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["device_id"] != "stackchan-001" || registry["identity_status"] != "unknown" {
		t.Fatalf("registry = %#v, want xiaozhi device with unknown firmware identity", registry)
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("registry capabilities = %#v, want sanitized xiaozhi feature map", registry["capabilities"])
	}
	for key, want := range map[string]any{
		"xiaozhi_profile":     "stock",
		"xiaozhi_feature_mcp": "true",
		"xiaozhi_feature_aec": "true",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
	for _, forbidden := range []string{"xiaozhi_feature_debug_metrics", "xiaozhi_feature_device_events"} {
		if _, ok := capabilities[forbidden]; ok {
			t.Fatalf("stock registry leaked debug feature %q: %#v", forbidden, capabilities)
		}
	}
}

func TestXiaozhiWebSocketRecordsDebugProfileWithoutLeakingHelloReply(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"device_id": "stackchan-debug-001",
		"features": map[string]any{
			"mcp":           true,
			"aec":           true,
			"device_events": true,
			"debug_metrics": true,
		},
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	replyJSON, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"debug_metrics", "device_events"} {
		if strings.Contains(strings.ToLower(string(replyJSON)), forbidden) {
			t.Fatalf("hello reply leaked debug field %q: %s", forbidden, replyJSON)
		}
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("registry capabilities = %#v, want xiaozhi debug feature map", registry["capabilities"])
	}
	for key, want := range map[string]any{
		"xiaozhi_profile":                  "debug",
		"xiaozhi_feature_mcp":              "true",
		"xiaozhi_feature_aec":              "true",
		"xiaozhi_feature_device_events":    "true",
		"xiaozhi_feature_debug_metrics":    "true",
		"xiaozhi_debug_extension_isolated": "true",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestXiaozhiSessionTurnCancelInvalidatesCurrentTurnAndResetsPacer(t *testing.T) {
	session := &xiaozhiSession{}
	turn := session.startXiaozhiTurn(context.Background())
	if turn.id != 1 {
		t.Fatalf("turn id = %d, want 1", turn.id)
	}
	if session.shouldAbortXiaozhiTurn(turn) {
		t.Fatal("fresh current turn should not abort")
	}

	ok, err := turn.pacer.Send(context.Background(), []byte("a"), func(context.Context, []byte) error {
		return nil
	}, nil)
	if err != nil || !ok {
		t.Fatalf("pacer send = ok:%v err:%v", ok, err)
	}
	if turn.pacer.SentFrames() != 1 {
		t.Fatalf("sent frames = %d, want 1 before cancel", turn.pacer.SentFrames())
	}

	session.cancelCurrentXiaozhiTurn("abort")
	if session.currentTurn != nil {
		t.Fatalf("current turn = %+v, want cleared", session.currentTurn)
	}
	if turn.ctx.Err() == nil {
		t.Fatal("cancelled turn context is still active")
	}
	if !session.shouldAbortXiaozhiTurn(turn) {
		t.Fatal("cancelled turn should abort frame send checks")
	}
	if turn.pacer.SentFrames() != 0 {
		t.Fatalf("sent frames = %d, want pacer reset on cancel", turn.pacer.SentFrames())
	}

	next := session.startXiaozhiTurn(context.Background())
	if next.id != 2 {
		t.Fatalf("next turn id = %d, want 2", next.id)
	}
	if session.shouldAbortXiaozhiTurn(next) {
		t.Fatal("new current turn should not abort")
	}
}

func TestXiaozhiWebSocketAbortCancelsCurrentTurn(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"device_id":  "stackchan-001",
		"trace_id":   "a21-trace-xiaozhi-turn-cancel",
		"session_id": "a21-session-xiaozhi-turn-cancel",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-turn-cancel", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "xiaozhi.turn.start") || !traceContains(traces.Events, "xiaozhi.turn.cancel") {
		t.Fatalf("trace missing turn lifecycle markers: %+v", traces.Events)
	}
}

func TestWriteXiaozhiOpusDownlinkUsesPacerAndCurrentTurn(t *testing.T) {
	server := NewServer()
	session := &xiaozhiSession{
		deviceID:  "stackchan-001",
		traceID:   "a21-trace-xiaozhi-downlink",
		sessionID: "a21-session-xiaozhi-downlink",
	}
	turn := session.startXiaozhiTurn(context.Background())
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, a21WebSocketAcceptOptions())
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "test done")
		ok, err := server.writeXiaozhiOpusDownlink(context.Background(), conn, session, turn, providers.VoiceAudioChunk{
			Codec:        string(protocol.AudioCodecPCMS16LE),
			SampleRateHz: 24000,
			Channels:     1,
			DurationMS:   60,
			DataBase64:   xiaozhiTestPCM16Base64(24000, 60, 6000),
		})
		if err != nil || !ok {
			t.Errorf("downlink = ok:%v err:%v", ok, err)
		}
	}))
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, ""), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })
	messageType, packet, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary {
		t.Fatalf("message type = %v, want binary", messageType)
	}
	codec, err := opuscodec.New(24000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	pcm, err := codec.DecodePCM16(packet)
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) != codec.FrameSamples() {
		t.Fatalf("decoded samples = %d, want %d", len(pcm), codec.FrameSamples())
	}
	if turn.pacer.SentFrames() != 1 {
		t.Fatalf("pacer sent frames = %d, want 1", turn.pacer.SentFrames())
	}
	if !traceContains(server.traceEvents("a21-trace-xiaozhi-downlink"), "xiaozhi.tts.opus_frame.downlink") {
		t.Fatalf("trace missing downlink marker: %+v", server.traceEvents("a21-trace-xiaozhi-downlink"))
	}
}

func TestWriteXiaozhiOpusDownlinkSkipsStaleTurn(t *testing.T) {
	server := NewServer()
	session := &xiaozhiSession{
		deviceID:  "stackchan-001",
		traceID:   "a21-trace-xiaozhi-stale-turn",
		sessionID: "a21-session-xiaozhi-stale-turn",
	}
	turn := session.startXiaozhiTurn(context.Background())
	session.cancelCurrentXiaozhiTurn("abort")

	ok, err := server.writeXiaozhiOpusDownlink(context.Background(), nil, session, turn, providers.VoiceAudioChunk{
		Codec:        string(protocol.AudioCodecPCMS16LE),
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   xiaozhiTestPCM16Base64(24000, 60, 6000),
	})
	if err != nil {
		t.Fatalf("stale turn err = %v", err)
	}
	if ok {
		t.Fatal("stale turn downlink unexpectedly sent")
	}
}

func TestXiaozhiWebSocketUsesStockHandshakeHeaders(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Device-Id":        []string{"stackchan-header-001"},
			"Protocol-Version": []string{"3"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"version": 3,
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	if reply["type"] != "hello" || reply["device_id"] != "stackchan-header-001" || reply["version"] != float64(3) {
		t.Fatalf("hello reply = %#v", reply)
	}
	if _, ok := reply["audio_params"].(map[string]any); !ok {
		t.Fatalf("hello reply missing stock audio_params: %#v", reply)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["device_id"] != "stackchan-header-001" {
		t.Fatalf("registry = %#v, want header device id", registry)
	}
}

func TestXiaozhiWebSocketListenDecodesOpusIngressTelemetry(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-raw-opus",
		"session_id": "a21-session-xiaozhi-raw-opus",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	startAck := readXiaozhiJSON(t, ctx, conn)
	if startAck["type"] != "listen" || startAck["state"] != "start" || startAck["status"] != "accepted" {
		t.Fatalf("listen start ack = %#v", startAck)
	}
	packet := xiaozhiTestOpusPacket(t)
	if err := conn.Write(ctx, websocket.MessageBinary, packet); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, packet); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	summary, ok := ttsStart["audio_ingress"].(map[string]any)
	if !ok {
		t.Fatalf("audio summary = %#v", ttsStart["audio_ingress"])
	}
	if summary["codec"] != "opus" || summary["decode_status"] != XiaozhiOpusDecodedPCMState || summary["frame_count"] != float64(2) || summary["byte_count"] != float64(len(packet)*2) {
		t.Fatalf("audio summary = %#v", summary)
	}
	if summary["decoded_frame_count"] != float64(2) || summary["decoded_sample_count"] != float64(1920) || summary["decoded_duration_ms"] != float64(120) {
		t.Fatalf("audio summary = %#v", summary)
	}
	sentence := readXiaozhiJSON(t, ctx, conn)
	if sentence["type"] != "tts" || sentence["state"] != "sentence_start" {
		t.Fatalf("sentence start = %#v", sentence)
	}
	ttsStop := readXiaozhiJSON(t, ctx, conn)
	if ttsStop["type"] != "tts" || ttsStop["state"] != "stop" {
		t.Fatalf("tts stop = %#v", ttsStop)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-raw-opus")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "xiaozhi.opus_frame.received") || !traceContains(traces.Events, "xiaozhi.opus_frame.decoded") || !traceContains(traces.Events, "xiaozhi."+XiaozhiOpusDecodedPCMState) {
		t.Fatalf("trace missing xiaozhi opus markers: %+v", traces.Events)
	}
}

func TestXiaozhiWebSocketKeepsBadOpusDecodeHonest(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-bad-opus",
		"session_id": "a21-session-xiaozhi-bad-opus",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, bytes.Repeat([]byte{0x7f}, opuscodec.MaxOpusPacketBytes+1)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	summary, ok := ttsStart["audio_ingress"].(map[string]any)
	if !ok {
		t.Fatalf("audio summary = %#v", ttsStart["audio_ingress"])
	}
	if summary["decode_status"] != XiaozhiOpusDecodeErrorState || summary["frame_count"] != float64(1) || summary["decode_error_count"] != float64(1) {
		t.Fatalf("audio summary = %#v", summary)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
}

func TestXiaozhiWebSocketDecodedOpusFeedsAudioIngress(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-audio-ingress",
		"session_id": "a21-session-xiaozhi-audio-ingress",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	readXiaozhiJSON(t, ctx, conn)

	req := httptest.NewRequest(http.MethodGet, "/v1/audio/recent?device_id=stackchan-001&session_id=a21-session-xiaozhi-audio-ingress", nil)
	req.RemoteAddr = "127.0.0.1:45678"
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{
		`"sample_rate_hz":16000`,
		`"duration_ms":60`,
		`"data_bytes":1920`,
		`"speech_detected":true`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("recent response missing %q: %s", want, rec.Body.String())
		}
	}
	if strings.Contains(rec.Body.String(), `"data_base64"`) {
		t.Fatalf("default recent response leaked decoded audio: %s", rec.Body.String())
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-audio-ingress", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "audio.ingress.buffered") || !traceContains(traces.Events, string(audio.EventVADSpeechStart)) {
		t.Fatalf("trace missing decoded ingress markers: %+v", traces.Events)
	}
}

func TestXiaozhiWebSocketListenStopRunsVoicePipelineAndSendsPacedOpus(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-pipeline",
		"session_id": "a21-session-xiaozhi-pipeline",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	audioIngress, ok := ttsStart["audio_ingress"].(map[string]any)
	if !ok {
		t.Fatalf("audio ingress = %#v", ttsStart["audio_ingress"])
	}
	if audioIngress["asr_status"] != "fixture_completed" || audioIngress["tts_status"] != "opus_downlink_fixture" {
		t.Fatalf("audio ingress = %#v, want fixture pipeline status", audioIngress)
	}
	pipeline, ok := ttsStart["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("voice pipeline = %#v", ttsStart["voice_pipeline"])
	}
	if pipeline["schema_version"] != "a21.voice_pipeline.fixture.v1" || pipeline["execution_mode"] != "fixture" || pipeline["audio_chunk_count"] != float64(1) {
		t.Fatalf("voice pipeline = %#v", pipeline)
	}
	startPayload := mustJSON(t, ttsStart)
	for _, forbidden := range []string{"data_base64", "raw_audio", "provider output", "fixture transcript", "http://", "https://", "/Users/"} {
		if strings.Contains(startPayload, forbidden) {
			t.Fatalf("tts start leaked %q: %s", forbidden, startPayload)
		}
	}

	sentence := readXiaozhiJSON(t, ctx, conn)
	if sentence["type"] != "tts" || sentence["state"] != "sentence_start" || sentence["placeholder"] == true {
		t.Fatalf("sentence start = %#v", sentence)
	}
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	ttsStop := readXiaozhiJSON(t, ctx, conn)
	if ttsStop["type"] != "tts" || ttsStop["state"] != "stop" || ttsStop["reason"] != "voice_pipeline_fixture_completed" {
		t.Fatalf("tts stop = %#v", ttsStop)
	}

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-pipeline")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"xiaozhi.voice_pipeline.start",
		"asr.first_partial",
		"provider.first_content",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"xiaozhi.tts.opus_frame.downlink",
		"xiaozhi.voice_pipeline.completed",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}

func TestXiaozhiWebSocketAcceptsProtocolVersion3BinaryFrames(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"version":    3,
		"trace_id":   "a21-trace-xiaozhi-v3",
		"session_id": "a21-session-xiaozhi-v3",
		"device_id":  "stackchan-001",
	})
	hello := readXiaozhiJSON(t, ctx, conn)
	if hello["version"] != float64(3) {
		t.Fatalf("hello version = %#v, want 3", hello["version"])
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)

	payload := xiaozhiTestOpusPacket(t)
	wire := make([]byte, 4+len(payload))
	wire[0] = 0
	binary.BigEndian.PutUint16(wire[2:4], uint16(len(payload)))
	copy(wire[4:], payload)
	if err := conn.Write(ctx, websocket.MessageBinary, wire); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	summary, ok := ttsStart["audio_ingress"].(map[string]any)
	if !ok {
		t.Fatalf("audio summary = %#v", ttsStart["audio_ingress"])
	}
	if summary["profile"] != "xiaozhi_binary_v3" || summary["frame_count"] != float64(1) || summary["byte_count"] != float64(len(payload)) || summary["decode_status"] != XiaozhiOpusDecodedPCMState {
		t.Fatalf("audio summary = %#v", summary)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
}

func TestXiaozhiWebSocketAbortStopsPlaceholderTTSAndPreventsStaleBinary(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-abort",
		"session_id": "a21-session-xiaozhi-abort",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, []byte{0x01, 0x02, 0x03}); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "wake_word_detected"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, []byte{0x04, 0x05}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)
}

func TestXiaozhiWebSocketRejectsLegacyIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"device_id": "x21-device",
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	if reply["type"] != "error" || reply["code"] != "invalid_device_id" {
		t.Fatalf("reply = %#v, want invalid_device_id", reply)
	}
}

func TestControlWebSocketRegistersFirmwareIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       7,
		TraceID:   "a21-trace-device-000007",
		SessionID: "a21-session-device",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		Text:            "先说，我在",
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
	})
	readControlEvents(t, ctx, conn, 3)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var registry DeviceRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(registry.Devices))
	}
	device := registry.Devices[0]
	if device.DeviceID != "stackchan-001" {
		t.Fatalf("device id = %q", device.DeviceID)
	}
	if device.IdentityStatus != "ok" {
		t.Fatalf("identity status = %q, want ok", device.IdentityStatus)
	}
	if device.Firmware.ID != "a21-stackchan" || device.Firmware.Version != "0.1.0" || device.Firmware.Board != "m5stack-cores3" || device.Firmware.Commit != "082eb938b713" {
		t.Fatalf("firmware identity = %+v", device.Firmware)
	}
	if device.LastEvent != "mock.turn" {
		t.Fatalf("last event = %q, want mock.turn", device.LastEvent)
	}
	if device.LastTraceID != "a21-trace-device-000007" {
		t.Fatalf("last trace = %q", device.LastTraceID)
	}
}

func TestControlWebSocketRegistersStackChanCapabilities(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       8,
		TraceID:   "a21-trace-device-000008",
		SessionID: "a21-session-device",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		Text:            "能力上报",
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
		Capabilities: map[string]string{
			"microphone":    "available",
			"speaker":       "available",
			"screen":        "available",
			"screen_touch":  "available",
			"top_touch":     "available",
			"servo_y":       "available",
			"servo_x":       "planned_continuous_rotation_axis",
			"rgb":           "available",
			"camera":        "planned_core_s3_camera",
			"imu":           "planned_9_axis_imu",
			"ambient_light": "planned_ambient_light_sensor",
			"proximity":     "planned_proximity_sensor",
			"battery":       "planned_550mah_battery",
			"nfc":           "planned_nfc",
			"infrared":      "planned_infrared_tx_rx",
		},
	})
	readControlEvents(t, ctx, conn, 3)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var registry DeviceRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(registry.Devices))
	}
	capabilities := registry.Devices[0].Capabilities
	wantCapabilities := map[string]string{
		"microphone":    "available",
		"speaker":       "available",
		"screen":        "available",
		"screen_touch":  "available",
		"top_touch":     "available",
		"servo_y":       "available",
		"servo_x":       "planned_continuous_rotation_axis",
		"rgb":           "available",
		"camera":        "planned_core_s3_camera",
		"imu":           "planned_9_axis_imu",
		"ambient_light": "planned_ambient_light_sensor",
		"proximity":     "planned_proximity_sensor",
		"battery":       "planned_550mah_battery",
		"nfc":           "planned_nfc",
		"infrared":      "planned_infrared_tx_rx",
	}
	for key, want := range wantCapabilities {
		if capabilities[key] != want {
			t.Fatalf("capability %s = %q, want %q; all=%#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestControlWebSocketRegistersRuntimeEchoWithoutAssistantReply(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       11,
		TraceID:   "a21-trace-runtime-echo",
		SessionID: "a21-session-runtime-echo",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventRuntimeEcho,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
		RuntimeEcho: map[string]string{
			"screen":  "speaking",
			"servo_y": "48deg",
			"rgb":     "#002430",
		},
	})

	ctxShort, cancelShort := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancelShort()
	var reply protocol.Envelope
	if err := wsjson.Read(ctxShort, conn, &reply); err == nil {
		t.Fatalf("runtime echo produced unexpected assistant reply: %+v", reply)
	}

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var registry DeviceRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(registry.Devices))
	}
	device := registry.Devices[0]
	if device.LastEvent != protocol.DeviceEventRuntimeEcho {
		t.Fatalf("last event = %q, want runtime echo", device.LastEvent)
	}
	for key, want := range map[string]string{
		"screen":  "speaking",
		"servo_y": "48deg",
		"rgb":     "#002430",
	} {
		if device.RuntimeEcho[key] != want {
			t.Fatalf("runtime echo %s = %q, want %q; all=%#v", key, device.RuntimeEcho[key], want, device.RuntimeEcho)
		}
	}
}

func TestControlWebSocketRejectsLegacyRuntimeEchoIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-001",
		Kind:     protocol.KindDeviceEvent,
		Seq:      12,
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventRuntimeEcho,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "abcdef1",
		RuntimeEcho: map[string]string{
			"screen": "x21-render",
		},
	})

	events := readControlEvents(t, ctx, conn, 1)
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != protocol.ExpressionError {
		t.Fatalf("state = %q, want error", payload.State)
	}
	if !strings.Contains(payload.Text, "invalid device firmware identity") {
		t.Fatalf("text = %q", payload.Text)
	}
	if strings.Contains(payload.Text, "x21-render") {
		t.Fatalf("error leaked forbidden echo value: %q", payload.Text)
	}
}

func TestControlWebSocketRejectsLegacyCapabilityIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-001",
		Kind:     protocol.KindDeviceEvent,
		Seq:      1,
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "abcdef1",
		Capabilities: map[string]string{
			"screen": "x21-compatible",
		},
	})

	events := readControlEvents(t, ctx, conn, 1)
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != protocol.ExpressionError {
		t.Fatalf("state = %q, want error", payload.State)
	}
	if !strings.Contains(payload.Text, "invalid device firmware identity") {
		t.Fatalf("text = %q", payload.Text)
	}
}

func TestDevicesEndpointDeclaresA21GatewayIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["schema_version"] != "a21.gateway.devices.v1" {
		t.Fatalf("schema_version = %#v, want a21.gateway.devices.v1", payload["schema_version"])
	}
	if payload["service"] != "a21-gateway" {
		t.Fatalf("service = %#v, want a21-gateway", payload["service"])
	}
}

func TestControlWebSocketRegistryExposesCurrentModeAndExpressionWithoutText(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       9,
		TraceID:   "a21-trace-device-state",
		SessionID: "a21-session-device-state",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModePrivate,
		Text:            "这句私人吐槽不应该进入设备 registry",
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
	})
	readControlEvents(t, ctx, conn, 3)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var registry struct {
		Devices []map[string]any `json:"devices"`
	}
	if err := json.Unmarshal(body, &registry); err != nil {
		t.Fatal(err)
	}
	devices := registry.Devices
	if len(devices) != 1 {
		t.Fatalf("devices = %d, want 1: %s", len(devices), string(body))
	}
	device := devices[0]
	if device["current_mode"] != string(protocol.ModePrivate) {
		t.Fatalf("current_mode = %#v, want private; body=%s", device["current_mode"], string(body))
	}
	if device["current_expression"] != string(protocol.ExpressionSpeaking) {
		t.Fatalf("current_expression = %#v, want speaking; body=%s", device["current_expression"], string(body))
	}
	if strings.Contains(string(body), "私人吐槽") {
		t.Fatalf("registry leaked utterance text: %s", string(body))
	}
}

func TestDeviceRegistryMarksOnlineAndStaleByLastSeenAge(t *testing.T) {
	server := NewServer()
	current := time.UnixMilli(2_000_000)
	server.now = func() time.Time { return current }
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-001",
		Kind:     protocol.KindDeviceEvent,
		Seq:      10,
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		Text:            "heartbeat",
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
	})
	readControlEvents(t, ctx, conn, 3)

	current = time.UnixMilli(2_000_000 + 299_999)
	online := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if online["connection_status"] != "online" || int64(online["device_age_ms"].(float64)) != 299_997 {
		t.Fatalf("online status/age = %#v/%#v, want online/299997", online["connection_status"], online["device_age_ms"])
	}

	current = time.UnixMilli(2_000_000 + 300_003)
	stale := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if stale["connection_status"] != "stale" || int64(stale["device_age_ms"].(float64)) != 300_001 {
		t.Fatalf("stale status/age = %#v/%#v, want stale/300001", stale["connection_status"], stale["device_age_ms"])
	}
}

func TestControlWebSocketRejectsForbiddenFirmwareIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-legacy-001",
		Kind:     protocol.KindDeviceEvent,
		Seq:      1,
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "x21-stackchan",
		FirmwareVersion: "1.0.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "abcdef1",
	})

	events := readControlEvents(t, ctx, conn, 1)
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != protocol.ExpressionError {
		t.Fatalf("state = %q, want error", payload.State)
	}
	if !strings.Contains(payload.Text, "invalid device firmware identity") {
		t.Fatalf("text = %q", payload.Text)
	}

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var registry DeviceRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(registry.Devices))
	}
	if registry.Devices[0].IdentityStatus != "invalid" {
		t.Fatalf("identity status = %q, want invalid", registry.Devices[0].IdentityStatus)
	}
	if !strings.Contains(registry.Devices[0].IdentityError, "forbidden") {
		t.Fatalf("identity error = %q", registry.Devices[0].IdentityError)
	}

	metricsResp, err := http.Get(httpServer.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = metricsResp.Body.Close() })
	metricsBody, err := io.ReadAll(metricsResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(metricsBody), "a21_device_identity_invalid_total 1") {
		t.Fatalf("metrics missing invalid device identity count:\n%s", metricsBody)
	}
}

func TestControlWebSocketInterruptKeepsTrace(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-sim-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       1,
		TraceID:   "a21-trace-000009",
		SessionID: "a21-session-000009",
	}, protocol.DeviceEventPayload{
		Event: protocol.DeviceEventInterrupt,
		Mode:  protocol.ModeWorkmate,
	})

	events := readControlEvents(t, ctx, conn, 2)
	var first protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &first); err != nil {
		t.Fatal(err)
	}
	if first.State != protocol.ExpressionInterrupted {
		t.Fatalf("first state = %q, want interrupted", first.State)
	}
	if events[0].TraceID != "a21-trace-000009" {
		t.Fatalf("trace = %q, want a21-trace-000009", events[0].TraceID)
	}
	if events[0].SessionID != "a21-session-000009" {
		t.Fatalf("session = %q, want a21-session-000009", events[0].SessionID)
	}
}

func TestControlWebSocketTouchWakeOrListenMapsToMockTurn(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       3,
		TraceID:   "a21-trace-touch-001",
		SessionID: "a21-session-touch-001",
	}, protocol.DeviceEventPayload{
		Event:       protocol.DeviceEventTouchWakeOrListen,
		Mode:        protocol.ModeWorkmate,
		Text:        "先说，我在",
		TouchSource: protocol.TouchSourceScreen,
	})

	events := readControlEvents(t, ctx, conn, 3)
	wantStates := []protocol.ExpressionState{
		protocol.ExpressionListening,
		protocol.ExpressionThinking,
		protocol.ExpressionSpeaking,
	}
	for i, want := range wantStates {
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.State != want {
			t.Fatalf("event %d state = %q, want %q", i, payload.State, want)
		}
	}

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var registry DeviceRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(registry.Devices))
	}
	if registry.Devices[0].LastEvent != protocol.DeviceEventTouchWakeOrListen {
		t.Fatalf("last event = %q, want touch wake", registry.Devices[0].LastEvent)
	}
	if registry.Devices[0].LastTouchSource != protocol.TouchSourceScreen {
		t.Fatalf("last touch source = %q, want screen", registry.Devices[0].LastTouchSource)
	}
}

func TestControlWebSocketTouchBargeInMapsToInterrupt(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       4,
		TraceID:   "a21-trace-touch-002",
		SessionID: "a21-session-touch-002",
	}, protocol.DeviceEventPayload{
		Event:       protocol.DeviceEventTouchBargeIn,
		Mode:        protocol.ModeWorkmate,
		TouchSource: protocol.TouchSourceTopSensor,
	})

	events := readControlEvents(t, ctx, conn, 2)
	var first protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &first); err != nil {
		t.Fatal(err)
	}
	if first.State != protocol.ExpressionInterrupted {
		t.Fatalf("first state = %q, want interrupted", first.State)
	}
	if events[0].TraceID != "a21-trace-touch-002" {
		t.Fatalf("trace = %q, want a21-trace-touch-002", events[0].TraceID)
	}
}

func TestControlWebSocketTopTouchGesturesStayDistinct(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	gestureCases := []struct {
		name      string
		seq       uint64
		traceID   string
		event     protocol.DeviceEventKind
		wantState protocol.ExpressionState
		wantMode  protocol.Mode
	}{
		{
			name:      "tap",
			seq:       5,
			traceID:   "a21-trace-touch-top-tap",
			event:     protocol.DeviceEventTouchTopTap,
			wantState: protocol.ExpressionListening,
			wantMode:  protocol.ModeWorkmate,
		},
		{
			name:      "swipe forward",
			seq:       6,
			traceID:   "a21-trace-touch-top-forward",
			event:     protocol.DeviceEventTouchTopSwipeForward,
			wantState: protocol.ExpressionThinking,
			wantMode:  protocol.ModeCoCreation,
		},
		{
			name:      "swipe backward",
			seq:       7,
			traceID:   "a21-trace-touch-top-backward",
			event:     protocol.DeviceEventTouchTopSwipeBackward,
			wantState: protocol.ExpressionIdle,
			wantMode:  protocol.ModeFocus,
		},
	}

	for _, tc := range gestureCases {
		writeDeviceEvent(t, ctx, conn, protocol.Envelope{
			Protocol:  protocol.ProtocolVersion,
			DeviceID:  "stackchan-001",
			Kind:      protocol.KindDeviceEvent,
			Seq:       tc.seq,
			TraceID:   tc.traceID,
			SessionID: "a21-session-touch-top",
		}, protocol.DeviceEventPayload{
			Event:       tc.event,
			Mode:        protocol.ModeWorkmate,
			TouchSource: protocol.TouchSourceTopSensor,
		})

		events := readControlEvents(t, ctx, conn, 1)
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.State != tc.wantState || payload.Mode != tc.wantMode {
			t.Fatalf("%s control = %+v, want state=%q mode=%q", tc.name, payload, tc.wantState, tc.wantMode)
		}

		traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id="+tc.traceID, nil)
		traceRec := httptest.NewRecorder()
		httpServer.Config.Handler.ServeHTTP(traceRec, traceReq)
		if !strings.Contains(traceRec.Body.String(), "device."+string(tc.event)+".received") {
			t.Fatalf("%s trace missing device event marker: %s", tc.name, traceRec.Body.String())
		}
	}
}

func TestAudioWebSocketAcceptsAudioFrame(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	audio, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-sim-001",
		Kind:      protocol.KindAudioFrame,
		Seq:       1,
		TraceID:   "a21-trace-audio-001",
		SessionID: "a21-session-audio-001",
		Payload:   audio,
	}); err != nil {
		t.Fatal(err)
	}

	events := readControlEvents(t, ctx, conn, 1)
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != protocol.ExpressionListening {
		t.Fatalf("state = %q, want listening", payload.State)
	}
	if events[0].TraceID != "a21-trace-audio-001" {
		t.Fatalf("trace = %q, want a21-trace-audio-001", events[0].TraceID)
	}
}

func TestAudioRecentEndpointReturnsLoopbackOnlyRedactedFrames(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	payload := pcm16Base64WithSample(12000)
	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 7, "a21-trace-physical-capture", "a21-session-physical-capture", payload)
	readControlEvents(t, ctx, conn, 2)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/audio/recent?device_id=stackchan-001&session_id=a21-session-physical-capture", nil)
	req.RemoteAddr = "127.0.0.1:45678"
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), payload) {
		t.Fatalf("default recent audio response leaked raw audio: %s", rec.Body.String())
	}
	for _, want := range []string{
		`"schema_version":"a21.gateway.audio_recent.v1"`,
		`"device_id":"stackchan-001"`,
		`"session_id":"a21-session-physical-capture"`,
		`"frames"`,
		`"data_bytes":640`,
		`"speech_detected":true`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("recent response missing %q: %s", want, rec.Body.String())
		}
	}

	withAudioReq := httptest.NewRequest(http.MethodGet, "/v1/audio/recent?device_id=stackchan-001&session_id=a21-session-physical-capture&include_audio=1", nil)
	withAudioReq.RemoteAddr = "127.0.0.1:45678"
	withAudioRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(withAudioRec, withAudioReq)
	if withAudioRec.Code != http.StatusOK {
		t.Fatalf("include audio status = %d: %s", withAudioRec.Code, withAudioRec.Body.String())
	}
	if !strings.Contains(withAudioRec.Body.String(), payload) {
		t.Fatalf("include audio response missing payload: %s", withAudioRec.Body.String())
	}

	remoteReq := httptest.NewRequest(http.MethodGet, "/v1/audio/recent?device_id=stackchan-001&session_id=a21-session-physical-capture&include_audio=1", nil)
	remoteReq.RemoteAddr = "192.168.1.42:45678"
	remoteRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(remoteRec, remoteReq)
	if remoteRec.Code != http.StatusForbidden {
		t.Fatalf("remote status = %d, want 403: %s", remoteRec.Code, remoteRec.Body.String())
	}
	if strings.Contains(remoteRec.Body.String(), payload) {
		t.Fatalf("remote rejection leaked raw audio: %s", remoteRec.Body.String())
	}
}

func TestAudioWebSocketReturnsMockPlaybackChunk(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	audio, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-sim-001",
		Kind:      protocol.KindAudioFrame,
		Seq:       1,
		TraceID:   "a21-trace-audio-downlink",
		SessionID: "a21-session-audio-downlink",
		Payload:   audio,
	}); err != nil {
		t.Fatal(err)
	}

	events := readControlEvents(t, ctx, conn, 2)
	var speaking protocol.ControlEventPayload
	if err := json.Unmarshal(events[1].Payload, &speaking); err != nil {
		t.Fatal(err)
	}
	if speaking.State != protocol.ExpressionSpeaking || speaking.StreamID != "a21-audio-stream-000001" {
		t.Fatalf("speaking payload = %+v", speaking)
	}
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("kind = %q, want %q", playback.Kind, protocol.KindAudioPlaybackChunk)
	}
	if playback.TraceID != "a21-trace-audio-downlink" {
		t.Fatalf("trace = %q, want a21-trace-audio-downlink", playback.TraceID)
	}
	var payload protocol.AudioPlaybackChunk
	if err := json.Unmarshal(playback.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.StreamID != "a21-audio-stream-000001" {
		t.Fatalf("stream = %q, want a21-audio-stream-000001", payload.StreamID)
	}
	if payload.Codec != protocol.AudioCodecPCMS16LE || payload.SampleRateHz != 16000 || payload.Channels != 1 || payload.DurationMS != 20 {
		t.Fatalf("playback payload = %+v", payload)
	}
	decoded, err := base64.StdEncoding.DecodeString(payload.DataBase64)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 640 {
		t.Fatalf("decoded payload bytes = %d, want 640", len(decoded))
	}
}

func TestAudioWebSocketReturnsAudibleMockPlaybackForPhysicalStackChan(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	audio, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindAudioFrame,
		Seq:       1,
		TraceID:   "a21-trace-physical-audio",
		SessionID: "a21-session-physical-audio",
		Payload:   audio,
	}); err != nil {
		t.Fatal(err)
	}

	readControlEvents(t, ctx, conn, 2)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	var payload protocol.AudioPlaybackChunk
	if err := json.Unmarshal(playback.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(payload.DataBase64)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 640 || bytes.Equal(decoded, make([]byte, len(decoded))) {
		t.Fatalf("physical StackChan mock playback should be audible non-silent PCM, got %d bytes", len(decoded))
	}
}

func TestAudioWebSocketDoesNotLoopPhysicalMockPlaybackForSameSpeech(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-physical-loop", "a21-session-physical-loop", pcm16Base64WithSample(12000))
	readControlEvents(t, ctx, conn, 2)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("kind = %q, want playback chunk", playback.Kind)
	}

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 2, "a21-trace-physical-loop", "a21-session-physical-loop", pcm16Base64WithSample(12000))
	assertNoEnvelope(t, conn, 100*time.Millisecond)
}

func TestDeviceControlArmsSinglePhysicalMockPlaybackForNextAudioFrame(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening","mode":"workmate","trace_id":"a21-trace-armed-loop","session_id":"a21-session-armed-loop","mock_playback_on_next_audio_frame":true}`)
	resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d: %s", resp.StatusCode, data)
	}
	readControlEvents(t, ctx, conn, 1)

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-armed-loop", "a21-session-armed-loop", pcm16Base64WithSample(0))
	readControlEvents(t, ctx, conn, 2)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("kind = %q, want playback chunk", playback.Kind)
	}

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 2, "a21-trace-armed-loop", "a21-session-armed-loop", pcm16Base64WithSample(0))
	assertNoEnvelope(t, conn, 100*time.Millisecond)
}

func TestPhysicalStackChanDeviceIDExcludesSimulatorAndBenchDevices(t *testing.T) {
	tests := []struct {
		deviceID string
		want     bool
	}{
		{deviceID: "stackchan-001", want: true},
		{deviceID: "stackchan-sim-001", want: false},
		{deviceID: "stackchan-bench-001", want: false},
	}
	for _, tc := range tests {
		if got := physicalStackChanDeviceID(tc.deviceID); got != tc.want {
			t.Fatalf("physicalStackChanDeviceID(%q) = %v, want %v", tc.deviceID, got, tc.want)
		}
	}
}

func TestDeviceControlEndpointDeliversControlAndAudioToRegisteredDevice(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"speaking","mode":"workmate","text":"beep","trace_id":"a21-trace-device-control","session_id":"a21-session-device-control","stream_id":"a21-device-command-stream-001","mock_audio_chunks":2}`)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	events := readControlEvents(t, ctx, conn, 1)
	var control protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &control); err != nil {
		t.Fatal(err)
	}
	if control.State != protocol.ExpressionSpeaking || control.StreamID != "a21-device-command-stream-001" {
		t.Fatalf("control = %+v", control)
	}

	for i := 0; i < 2; i++ {
		var playback protocol.Envelope
		if err := wsjson.Read(ctx, conn, &playback); err != nil {
			t.Fatal(err)
		}
		if playback.Kind != protocol.KindAudioPlaybackChunk {
			t.Fatalf("playback %d kind = %q", i, playback.Kind)
		}
		var chunk protocol.AudioPlaybackChunk
		if err := json.Unmarshal(playback.Payload, &chunk); err != nil {
			t.Fatal(err)
		}
		if chunk.StreamID != "a21-device-command-stream-001" || chunk.DataBase64 == "" {
			t.Fatalf("chunk = %+v", chunk)
		}
		decoded, err := base64.StdEncoding.DecodeString(chunk.DataBase64)
		if err != nil {
			t.Fatal(err)
		}
		if len(decoded) != 640 || bytes.Equal(decoded, make([]byte, len(decoded))) {
			t.Fatalf("expected non-silent 640-byte validation audio, got len=%d", len(decoded))
		}
	}

	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d: %s", resp.StatusCode, data)
		}
		var delivered DeviceControlResponse
		if err := json.NewDecoder(resp.Body).Decode(&delivered); err != nil {
			t.Fatal(err)
		}
		if delivered.Status != "delivered" || delivered.DeliveredTransport != "audio_ws" || len(delivered.Events) != 3 {
			t.Fatalf("delivered = %+v", delivered)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestDeviceControlEndpointDeliversDiagnosticSpeakerTone(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"speaking","mode":"workmate","text":"TONE","trace_id":"a21-trace-speaker-tone","session_id":"a21-session-speaker-tone","diagnostic_tone_hz":1000,"diagnostic_tone_duration_ms":3000,"diagnostic_tone_volume":160}`)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	events := readControlEvents(t, ctx, conn, 1)
	var control protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &control); err != nil {
		t.Fatal(err)
	}
	if control.DiagnosticToneHz != 1000 || control.DiagnosticToneDurationMS != 3000 || control.DiagnosticToneVolume != 160 {
		t.Fatalf("diagnostic tone payload = %+v", control)
	}

	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d: %s", resp.StatusCode, data)
		}
		var delivered DeviceControlResponse
		if err := json.NewDecoder(resp.Body).Decode(&delivered); err != nil {
			t.Fatal(err)
		}
		if delivered.Status != "delivered" || len(delivered.Events) != 1 {
			t.Fatalf("delivered = %+v", delivered)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestDeviceControlEndpointDeliversProvidedAudioChunks(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	pcm := mockPCM16SquareWaveBase64(16000, 20)
	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"speaking","mode":"workmate","trace_id":"a21-trace-provided-audio","session_id":"a21-session-provided-audio","stream_id":"a21-provided-audio-stream","audio_chunks":[{"stream_id":"a21-provided-audio-stream","codec":"pcm_s16le","sample_rate_hz":16000,"channels":1,"duration_ms":20,"data_base64":"` + pcm + `"}]}`)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	readControlEvents(t, ctx, conn, 1)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("kind = %q, want playback chunk", playback.Kind)
	}
	var chunk protocol.AudioPlaybackChunk
	if err := json.Unmarshal(playback.Payload, &chunk); err != nil {
		t.Fatal(err)
	}
	if chunk.StreamID != "a21-provided-audio-stream" || chunk.DataBase64 != pcm {
		t.Fatalf("chunk = %+v", chunk)
	}

	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d: %s", resp.StatusCode, data)
		}
		var delivered DeviceControlResponse
		if err := json.NewDecoder(resp.Body).Decode(&delivered); err != nil {
			t.Fatal(err)
		}
		if len(delivered.Events) != 2 {
			t.Fatalf("events = %d, want control + provided audio", len(delivered.Events))
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestDeviceControlIdleClearsPlaybackStreamFromRegistry(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	speakingBody := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"speaking","mode":"workmate","trace_id":"a21-trace-clear-stream","session_id":"a21-session-clear-stream","stream_id":"a21-clear-stream","mock_audio_chunks":1}`)
	speakingResp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", speakingBody)
	if err != nil {
		t.Fatal(err)
	}
	speakingResp.Body.Close()
	readControlEvents(t, ctx, conn, 1)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}

	idleBody := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"idle","mode":"workmate","trace_id":"a21-trace-clear-stream","session_id":"a21-session-clear-stream","stream_id":"a21-clear-stream"}`)
	idleResp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", idleBody)
	if err != nil {
		t.Fatal(err)
	}
	idleResp.Body.Close()
	readControlEvents(t, ctx, conn, 1)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var registry map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	devices := registry["devices"].([]any)
	device := devices[0].(map[string]any)
	if device["current_expression"] != string(protocol.ExpressionIdle) {
		t.Fatalf("current_expression = %#v, want idle; registry=%v", device["current_expression"], device)
	}
	if stream, ok := device["playback_stream_id"].(string); ok && stream != "" {
		t.Fatalf("playback_stream_id = %q, want empty after idle", stream)
	}
}

func TestAudioProbeOnlyDeviceControlSuppressesMockPlayback(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening","mode":"workmate","text":"probe","trace_id":"a21-trace-probe-control","session_id":"a21-session-probe","audio_probe_only":true}`)
	resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d: %s", resp.StatusCode, data)
	}
	readControlEvents(t, ctx, conn, 1)

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-probe-audio", "a21-session-probe", pcm16Base64WithSample(12000))
	assertNoEnvelope(t, conn, 100*time.Millisecond)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-probe-audio", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"audio.frame.received", "audio.ingress.buffered", "audio.probe.frame.accepted"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
}

func TestAudioWebSocketRegistersRuntimeEchoWithoutAudioPlayback(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       12,
		TraceID:   "a21-trace-audio-runtime-echo",
		SessionID: "a21-session-audio-runtime-echo",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventRuntimeEcho,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
		RuntimeEcho: map[string]string{
			"screen":               "listening",
			"audio_ws_sent_frames": "24",
		},
	})
	assertNoEnvelope(t, conn, 100*time.Millisecond)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var registry DeviceRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(registry.Devices))
	}
	device := registry.Devices[0]
	if device.Firmware.ID != "a21-stackchan" || device.IdentityStatus != "ok" {
		t.Fatalf("device identity = %+v status=%q, want ok a21-stackchan", device.Firmware, device.IdentityStatus)
	}
	if device.RuntimeEcho["screen"] != "listening" {
		t.Fatalf("runtime echo screen = %q, want listening; all=%#v", device.RuntimeEcho["screen"], device.RuntimeEcho)
	}
}

func TestDeviceControlEndpointRequiresConnectedAudioSocket(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 409: %s", resp.StatusCode, data)
	}
}

func TestAudioWebSocketRecordsIngressAndVADTrace(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	_ = writeAudioFrameWithPayload(t, ctx, conn, 1, "a21-trace-vad-ws", "a21-session-vad-ws", pcm16Base64WithSample(12000))

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-vad-ws", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	body := traceRec.Body.String()
	for _, want := range []string{"audio.ingress.buffered", "vad.speech.start"} {
		if !strings.Contains(body, want) {
			t.Fatalf("trace missing %q: %s", want, body)
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{
		"a21_audio_ingress_frames_total 1",
		"a21_audio_ingress_rms",
		"a21_vad_speech_start_total 1",
		`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 1`,
	} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestAudioWebSocketRejectsInvalidPCMFrame(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	audio, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   "AAAA",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-sim-001",
		Kind:      protocol.KindAudioFrame,
		Seq:       1,
		TraceID:   "a21-trace-audio-invalid",
		SessionID: "a21-session-audio-invalid",
		Payload:   audio,
	}); err != nil {
		t.Fatal(err)
	}

	events := readControlEvents(t, ctx, conn, 1)
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != protocol.ExpressionError || !strings.Contains(payload.Text, "invalid audio frame") {
		t.Fatalf("payload = %+v, want invalid audio frame error", payload)
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer readCancel()
	var unexpected protocol.Envelope
	if err := wsjson.Read(readCtx, conn, &unexpected); err == nil {
		t.Fatalf("unexpected envelope after invalid audio: %+v", unexpected)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-audio-invalid", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if !strings.Contains(traceRec.Body.String(), "audio.ingress.invalid") {
		t.Fatalf("trace missing invalid marker: %s", traceRec.Body.String())
	}
	if strings.Contains(traceRec.Body.String(), "audio.playback.chunk.sent") {
		t.Fatalf("invalid audio should not emit playback trace: %s", traceRec.Body.String())
	}
}

func TestAudioWebSocketVADStartDuringPlaybackTriggersBargeIn(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	firstChunk := writeAudioFrameWithPayload(t, ctx, conn, 1, "a21-trace-barge-audio", "a21-session-barge-audio", pcm16Base64WithSample(0))
	if firstChunk.StreamID != "a21-audio-stream-000001" {
		t.Fatalf("first stream = %q, want a21-audio-stream-000001", firstChunk.StreamID)
	}

	writeAudioFrameEnvelope(t, ctx, conn, 2, "a21-trace-barge-audio", "a21-session-barge-audio", pcm16Base64WithSample(12000))

	events := readControlEvents(t, ctx, conn, 2)
	var interrupted protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &interrupted); err != nil {
		t.Fatal(err)
	}
	if interrupted.State != protocol.ExpressionInterrupted || interrupted.StreamID != firstChunk.StreamID {
		t.Fatalf("interrupted payload = %+v, want interrupted stream %q", interrupted, firstChunk.StreamID)
	}
	var listening protocol.ControlEventPayload
	if err := json.Unmarshal(events[1].Payload, &listening); err != nil {
		t.Fatal(err)
	}
	if listening.State != protocol.ExpressionListening {
		t.Fatalf("second state = %q, want listening", listening.State)
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer readCancel()
	var unexpected protocol.Envelope
	if err := wsjson.Read(readCtx, conn, &unexpected); err == nil {
		t.Fatalf("unexpected envelope after barge-in: %+v", unexpected)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-barge-audio", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"barge_in.detected", "provider.cancel", "playback.stop"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	if !strings.Contains(metricsRec.Body.String(), "a21_barge_in_total 1") {
		t.Fatalf("metrics missing barge-in count:\n%s", metricsRec.Body.String())
	}
}

func TestAudioWebSocketKeepsPlaybackStreamStableForTrace(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	first := writeAudioFrame(t, ctx, conn, 11, "a21-trace-audio-stable", "a21-session-audio-stable")
	second := writeAudioFrame(t, ctx, conn, 12, "a21-trace-audio-stable", "a21-session-audio-stable")

	if first.StreamID == "" {
		t.Fatal("first stream id is empty")
	}
	if second.StreamID != first.StreamID {
		t.Fatalf("second stream = %q, want stable stream %q", second.StreamID, first.StreamID)
	}
}

func TestAudioWebSocketForwardsSpeechToRealtimeProviderAndCommitsOnVADEnd(t *testing.T) {
	provider := &capturingRealtimeAudioProvider{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeAudioFrameEnvelope(t, ctx, conn, 1, "a21-trace-realtime-audio", "a21-session-realtime-audio", pcm16Base64WithSample(12000))
	listening := readControlEvents(t, ctx, conn, 1)
	var listeningPayload protocol.ControlEventPayload
	if err := json.Unmarshal(listening[0].Payload, &listeningPayload); err != nil {
		t.Fatal(err)
	}
	if listeningPayload.State != protocol.ExpressionListening {
		t.Fatalf("first state = %q, want listening", listeningPayload.State)
	}

	writeAudioFrameEnvelope(t, ctx, conn, 2, "a21-trace-realtime-audio", "a21-session-realtime-audio", pcm16Base64WithSample(0))
	writeAudioFrameEnvelope(t, ctx, conn, 3, "a21-trace-realtime-audio", "a21-session-realtime-audio", pcm16Base64WithSample(0))
	thinking := readControlEvents(t, ctx, conn, 1)
	var thinkingPayload protocol.ControlEventPayload
	if err := json.Unmarshal(thinking[0].Payload, &thinkingPayload); err != nil {
		t.Fatal(err)
	}
	if thinkingPayload.State != protocol.ExpressionThinking {
		t.Fatalf("commit state = %q, want thinking", thinkingPayload.State)
	}
	assertNoEnvelope(t, conn, 100*time.Millisecond)

	if provider.startCalls != 1 {
		t.Fatalf("startCalls = %d, want 1", provider.startCalls)
	}
	if provider.session.SessionID != "a21-session-realtime-audio" || provider.session.DeviceID != "stackchan-sim-001" {
		t.Fatalf("provider session = %+v", provider.session)
	}
	if got := len(provider.sessionHandle.audioChunks); got != 1 {
		t.Fatalf("audio chunks = %d, want speech frame only", got)
	}
	if provider.sessionHandle.audioChunks[0].SampleRateHz != 16000 || provider.sessionHandle.audioChunks[0].DurationMS != 20 {
		t.Fatalf("audio chunk = %+v", provider.sessionHandle.audioChunks[0])
	}
	if provider.sessionHandle.commitCalls != 1 {
		t.Fatalf("commitCalls = %d, want 1", provider.sessionHandle.commitCalls)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-realtime-audio", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"provider.realtime_session.start", "provider.audio.append", "provider.audio.commit"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{"a21_realtime_audio_uplink_frames_total 1", "a21_realtime_audio_commit_total 1"} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestAudioWebSocketDoesNotStartRealtimeProviderForUnarmedPhysicalStackChan(t *testing.T) {
	provider := &capturingRealtimeAudioProvider{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-physical-realtime-unarmed", "a21-session-physical-realtime-unarmed", pcm16Base64WithSample(12000))
	assertNoEnvelope(t, conn, 100*time.Millisecond)

	if provider.startCalls != 0 {
		t.Fatalf("realtime provider startCalls = %d, want 0 for unarmed physical StackChan audio", provider.startCalls)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-physical-realtime-unarmed", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	body := traceRec.Body.String()
	for _, want := range []string{"audio.ingress.buffered", "vad.speech.start", "provider.realtime_audio.physical_suppressed"} {
		if !strings.Contains(body, want) {
			t.Fatalf("trace missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "provider.realtime_session.start") {
		t.Fatalf("unarmed physical audio should not start realtime provider: %s", body)
	}
}

func TestDeviceControlArmsRealtimeProviderForNextPhysicalSpeech(t *testing.T) {
	provider := &capturingRealtimeAudioProvider{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening","mode":"workmate","trace_id":"a21-trace-physical-realtime-armed","session_id":"a21-session-physical-realtime-armed","realtime_on_next_speech":true}`)
	resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d: %s", resp.StatusCode, data)
	}
	readControlEvents(t, ctx, conn, 1)

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-physical-realtime-armed", "a21-session-physical-realtime-armed", pcm16Base64WithSample(12000))
	events := readControlEvents(t, ctx, conn, 1)
	var listening protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &listening); err != nil {
		t.Fatal(err)
	}
	if listening.State != protocol.ExpressionListening {
		t.Fatalf("state = %q, want listening", listening.State)
	}
	if provider.startCalls != 1 {
		t.Fatalf("realtime provider startCalls = %d, want 1 after explicit arm", provider.startCalls)
	}
	if provider.session.DeviceID != "stackchan-001" || provider.session.SessionID != "a21-session-physical-realtime-armed" {
		t.Fatalf("provider session = %+v", provider.session)
	}
	if got := len(provider.sessionHandle.audioChunks); got != 1 {
		t.Fatalf("provider audio chunks = %d, want 1", got)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-physical-realtime-armed", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"provider.realtime_audio.physical_armed", "provider.realtime_session.start", "provider.audio.append"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
}

func TestAudioWebSocketClosesRealtimeProviderSessionWhenSocketCloses(t *testing.T) {
	provider := &capturingRealtimeAudioProvider{closed: make(chan struct{}, 1)}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}

	writeAudioFrameEnvelope(t, ctx, conn, 1, "a21-trace-realtime-close", "a21-session-realtime-close", pcm16Base64WithSample(12000))
	readControlEvents(t, ctx, conn, 1)
	if err := conn.Close(websocket.StatusNormalClosure, "test done"); err != nil {
		t.Fatal(err)
	}

	select {
	case <-provider.closed:
	case <-time.After(time.Second):
		t.Fatal("realtime provider session was not closed after audio WebSocket closed")
	}
}

func TestAudioWebSocketStreamsRealtimeProviderOutputEventsAfterCommit(t *testing.T) {
	provider := &capturingRealtimeAudioProvider{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeAudioFrameEnvelope(t, ctx, conn, 1, "a21-trace-realtime-downlink", "a21-session-realtime-downlink", pcm16Base64WithSample(12000))
	readControlEvents(t, ctx, conn, 1)
	writeAudioFrameEnvelope(t, ctx, conn, 2, "a21-trace-realtime-downlink", "a21-session-realtime-downlink", pcm16Base64WithSample(0))
	writeAudioFrameEnvelope(t, ctx, conn, 3, "a21-trace-realtime-downlink", "a21-session-realtime-downlink", pcm16Base64WithSample(0))
	readControlEvents(t, ctx, conn, 1)

	provider.sessionHandle.events <- providers.VoiceEvent{
		Session:  provider.session,
		Kind:     providers.VoiceEventSpeaking,
		Final:    false,
		StreamID: "a21-provider-stream-001",
		Audio: &providers.VoiceAudioChunk{
			Codec:        string(protocol.AudioCodecPCMS16LE),
			SampleRateHz: 16000,
			Channels:     1,
			DurationMS:   20,
			DataBase64:   "AAAA",
		},
	}

	events := readControlEvents(t, ctx, conn, 1)
	var speaking protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &speaking); err != nil {
		t.Fatal(err)
	}
	if speaking.State != protocol.ExpressionSpeaking || speaking.StreamID != "a21-provider-stream-001" {
		t.Fatalf("speaking payload = %+v", speaking)
	}
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("playback kind = %q, want audio.playback.chunk", playback.Kind)
	}
	var chunk protocol.AudioPlaybackChunk
	if err := json.Unmarshal(playback.Payload, &chunk); err != nil {
		t.Fatal(err)
	}
	if chunk.StreamID != "a21-provider-stream-001" || chunk.DataBase64 != "AAAA" {
		t.Fatalf("chunk = %+v", chunk)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-realtime-downlink", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"provider.audio.downlink", "provider.audio.first_downlink", "audio.playback.chunk.sent"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{
		"a21_realtime_audio_downlink_events_total 1",
		"a21_realtime_first_audio_ms_count 1",
	} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func writeDeviceEvent(t *testing.T, ctx context.Context, conn *websocket.Conn, envelope protocol.Envelope, payload protocol.DeviceEventPayload) {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	envelope.Payload = data
	if envelope.Protocol == "" {
		envelope.Protocol = protocol.ProtocolVersion
	}
	if err := wsjson.Write(ctx, conn, envelope); err != nil {
		t.Fatal(err)
	}
}

func writeAudioFrame(t *testing.T, ctx context.Context, conn *websocket.Conn, seq uint64, traceID string, sessionID string) protocol.AudioPlaybackChunk {
	t.Helper()
	return writeAudioFrameWithPayload(t, ctx, conn, seq, traceID, sessionID, pcm16Base64WithSample(0))
}

func writeAudioFrameWithPayload(t *testing.T, ctx context.Context, conn *websocket.Conn, seq uint64, traceID string, sessionID string, payloadBase64 string) protocol.AudioPlaybackChunk {
	t.Helper()
	writeAudioFrameEnvelope(t, ctx, conn, seq, traceID, sessionID, payloadBase64)
	readControlEvents(t, ctx, conn, 2)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("kind = %q, want playback chunk", playback.Kind)
	}
	var chunk protocol.AudioPlaybackChunk
	if err := json.Unmarshal(playback.Payload, &chunk); err != nil {
		t.Fatal(err)
	}
	return chunk
}

func writeAudioFrameEnvelope(t *testing.T, ctx context.Context, conn *websocket.Conn, seq uint64, traceID string, sessionID string, payloadBase64 string) {
	t.Helper()
	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-sim-001", seq, traceID, sessionID, payloadBase64)
}

func writeAudioFrameEnvelopeForDevice(t *testing.T, ctx context.Context, conn *websocket.Conn, deviceID string, seq uint64, traceID string, sessionID string, payloadBase64 string) {
	t.Helper()
	audio, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   payloadBase64,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  deviceID,
		Kind:      protocol.KindAudioFrame,
		Seq:       seq,
		TraceID:   traceID,
		SessionID: sessionID,
		Payload:   audio,
	}); err != nil {
		t.Fatal(err)
	}
}

func pcm16Base64WithSample(sample int16) string {
	const samplesPer20MS16K = 320
	data := make([]byte, samplesPer20MS16K*2)
	for i := 0; i < samplesPer20MS16K; i++ {
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func readControlEvents(t *testing.T, ctx context.Context, conn *websocket.Conn, count int) []protocol.Envelope {
	t.Helper()
	events := make([]protocol.Envelope, 0, count)
	for i := 0; i < count; i++ {
		var event protocol.Envelope
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			t.Fatal(err)
		}
		if event.Kind != protocol.KindControlEvent {
			t.Fatalf("event %d kind = %q, want control.event", i, event.Kind)
		}
		events = append(events, event)
	}
	return events
}

func fetchSingleDeviceRegistryItem(t *testing.T, serverURL string) map[string]any {
	t.Helper()
	resp, err := http.Get(serverURL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("device registry status = %d, want 200", resp.StatusCode)
	}
	var registry struct {
		Devices []map[string]any `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	devices := registry.Devices
	if len(devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(devices))
	}
	return devices[0]
}

func traceContains(events []TraceEvent, name string) bool {
	for _, event := range events {
		if event.Name == name {
			return true
		}
	}
	return false
}

func assertSummaryDelta(t *testing.T, name string, got *int64, want int64) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %d", name, got, want)
	}
}

func assertNoEnvelope(t *testing.T, conn *websocket.Conn, timeout time.Duration) {
	t.Helper()
	readCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var unexpected protocol.Envelope
	if err := wsjson.Read(readCtx, conn, &unexpected); err == nil {
		t.Fatalf("unexpected envelope: %+v", unexpected)
	}
}

func assertNoXiaozhiMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) {
	t.Helper()
	readCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var unexpected map[string]any
	if err := wsjson.Read(readCtx, conn, &unexpected); err == nil {
		t.Fatalf("unexpected xiaozhi message: %#v", unexpected)
	}
}

func writeXiaozhiHello(t *testing.T, ctx context.Context, conn *websocket.Conn, overrides map[string]any) {
	t.Helper()
	hello := map[string]any{
		"type":      "hello",
		"version":   1,
		"transport": "websocket",
		"features": map[string]any{
			"mcp": true,
			"aec": true,
		},
		"audio": map[string]any{
			"format":         "opus",
			"sample_rate":    16000,
			"channels":       1,
			"frame_duration": 60,
		},
	}
	for key, value := range overrides {
		hello[key] = value
	}
	if err := wsjson.Write(ctx, conn, hello); err != nil {
		t.Fatal(err)
	}
}

func xiaozhiTestOpusPacket(t *testing.T) []byte {
	t.Helper()
	codec, err := opuscodec.New(16000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := codec.EncodePCM16(make([]int16, codec.FrameSamples()))
	if err != nil {
		t.Fatal(err)
	}
	return packet
}

func xiaozhiTestSpeechOpusPacket(t *testing.T) []byte {
	t.Helper()
	codec, err := opuscodec.New(16000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	pcm := make([]int16, codec.FrameSamples())
	for i := range pcm {
		if (i/10)%2 == 0 {
			pcm[i] = 12000
		} else {
			pcm[i] = -12000
		}
	}
	packet, err := codec.EncodePCM16(pcm)
	if err != nil {
		t.Fatal(err)
	}
	return packet
}

func xiaozhiTestPCM16Base64(sampleRate int, durationMS int, sample int16) string {
	sampleCount := sampleRate * durationMS / 1000
	data := make([]byte, sampleCount*2)
	for i := 0; i < sampleCount; i++ {
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	return base64.StdEncoding.EncodeToString(data)
}

func readXiaozhiJSON(t *testing.T, ctx context.Context, conn *websocket.Conn) map[string]any {
	t.Helper()
	var message map[string]any
	if err := wsjson.Read(ctx, conn, &message); err != nil {
		t.Fatal(err)
	}
	return message
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func webSocketURL(serverURL string, path string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http") + path
}

type scriptedVoiceProvider struct {
	events []providers.VoiceEvent
}

func (p scriptedVoiceProvider) Name() string {
	return "scripted"
}

func (p scriptedVoiceProvider) StartTurn(ctx context.Context, req providers.VoiceTurnRequest) (<-chan providers.VoiceEvent, error) {
	events := make(chan providers.VoiceEvent, len(p.events))
	defer close(events)
	for _, event := range p.events {
		event.Session = req.Session
		events <- event
	}
	return events, nil
}

func (p scriptedVoiceProvider) Cancel(ctx context.Context, req providers.VoiceCancelRequest) (<-chan providers.VoiceEvent, error) {
	events := make(chan providers.VoiceEvent, 1)
	defer close(events)
	events <- providers.VoiceEvent{Session: req.Session, Kind: providers.VoiceEventCancelled, Text: "cancelled", Final: true}
	return events, nil
}

func (p scriptedVoiceProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{Provider: p.Name(), Status: providers.VoiceProviderHealthy, Configured: true, Realtime: true}, ctx.Err()
}

func (p scriptedVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

type unavailableVoiceProvider struct{}

func (p unavailableVoiceProvider) Name() string {
	return "unavailable"
}

func (p unavailableVoiceProvider) StartTurn(ctx context.Context, req providers.VoiceTurnRequest) (<-chan providers.VoiceEvent, error) {
	return nil, providers.ErrVoiceProviderUnavailable
}

func (p unavailableVoiceProvider) Cancel(ctx context.Context, req providers.VoiceCancelRequest) (<-chan providers.VoiceEvent, error) {
	return nil, providers.ErrVoiceProviderUnavailable
}

func (p unavailableVoiceProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{
		Provider:   p.Name(),
		Status:     providers.VoiceProviderUnavailable,
		Configured: false,
		Realtime:   false,
		Detail:     "test unavailable",
	}, providers.ErrVoiceProviderUnavailable
}

func (p unavailableVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

type capturingVoiceProvider struct {
	startEvents   []providers.VoiceEvent
	startCalls    int
	cancelRequest providers.VoiceCancelRequest
}

func (p *capturingVoiceProvider) Name() string {
	return "capturing"
}

func (p *capturingVoiceProvider) StartTurn(ctx context.Context, req providers.VoiceTurnRequest) (<-chan providers.VoiceEvent, error) {
	p.startCalls++
	events := make(chan providers.VoiceEvent, len(p.startEvents))
	defer close(events)
	for _, event := range p.startEvents {
		event.Session = req.Session
		events <- event
	}
	return events, nil
}

func (p *capturingVoiceProvider) Cancel(ctx context.Context, req providers.VoiceCancelRequest) (<-chan providers.VoiceEvent, error) {
	p.cancelRequest = req
	events := make(chan providers.VoiceEvent, 1)
	defer close(events)
	events <- providers.VoiceEvent{
		Session:  req.Session,
		Kind:     providers.VoiceEventCancelled,
		Text:     "capturing cancelled",
		Final:    true,
		StreamID: req.StreamID,
	}
	return events, nil
}

func (p *capturingVoiceProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{Provider: p.Name(), Status: providers.VoiceProviderHealthy, Configured: true, Realtime: true}, ctx.Err()
}

func (p *capturingVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

type capturingRealtimeAudioProvider struct {
	capturingVoiceProvider
	startCalls    int
	session       providers.VoiceSession
	sessionHandle *capturingRealtimeVoiceSession
	closed        chan struct{}
}

func (p *capturingRealtimeAudioProvider) Name() string {
	return "capturing-realtime-audio"
}

func (p *capturingRealtimeAudioProvider) StartRealtimeSession(ctx context.Context, session providers.VoiceSession) (providers.RealtimeVoiceSession, error) {
	p.startCalls++
	p.session = session
	p.sessionHandle = &capturingRealtimeVoiceSession{closed: p.closed, events: make(chan providers.VoiceEvent, 4)}
	return p.sessionHandle, ctx.Err()
}

func (p *capturingRealtimeAudioProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{Provider: p.Name(), Status: providers.VoiceProviderHealthy, Configured: true, Realtime: true}, ctx.Err()
}

type capturingRealtimeVoiceSession struct {
	audioChunks []protocol.AudioChunk
	commitCalls int
	cancelCalls int
	closed      chan struct{}
	events      chan providers.VoiceEvent
}

func (s *capturingRealtimeVoiceSession) SendAudio(ctx context.Context, chunk protocol.AudioChunk) error {
	s.audioChunks = append(s.audioChunks, chunk)
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) CommitAndCreateResponse(ctx context.Context) error {
	s.commitCalls++
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) Cancel(ctx context.Context, _ providers.VoiceCancelRequest) error {
	s.cancelCalls++
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) Close(ctx context.Context) error {
	if s.closed != nil {
		select {
		case s.closed <- struct{}{}:
		default:
		}
	}
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) Events() <-chan providers.VoiceEvent {
	return s.events
}

type countingV21Client struct {
	calls int
}

func (c *countingV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	c.calls++
	return v21adapter.NewMockClient().Query(ctx, request)
}

type failingV21Client struct{}

func (failingV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	return v21adapter.QueryResponse{}, errors.New("v21 unavailable")
}

type slowV21Client struct {
	delay time.Duration
}

func (c slowV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	timer := time.NewTimer(c.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return v21adapter.QueryResponse{}, ctx.Err()
	case <-timer.C:
		return v21adapter.NewMockClient().Query(ctx, request)
	}
}

func traceEventAtMS(events []TraceEvent, name string) (int64, bool) {
	for _, event := range events {
		if event.Name == name {
			return event.AtMS, true
		}
	}
	return 0, false
}
