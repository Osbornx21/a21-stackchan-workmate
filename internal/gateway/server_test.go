package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		"Waterfall",
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
	if len(response.Events) != 3 {
		t.Fatalf("events = %d, want 3", len(response.Events))
	}
	var pro protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[1].Payload, &pro); err != nil {
		t.Fatal(err)
	}
	if pro.State != protocol.ExpressionProfessional || pro.Mode != protocol.ModeProfessional {
		t.Fatalf("professional payload = %+v", pro)
	}
	var answer protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[2].Payload, &answer); err != nil {
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
		DataBase64:   "AAAA",
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
		Seq:       seq,
		TraceID:   traceID,
		SessionID: sessionID,
		Payload:   audio,
	}); err != nil {
		t.Fatal(err)
	}
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
