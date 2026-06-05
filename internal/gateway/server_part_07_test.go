package gateway

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/v21adapter"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestOrdinaryOfficeModesDoNotExposeProfessionalEvidenceOrV21Trace(t *testing.T) {
	for _, mode := range []protocol.Mode{protocol.ModePublic, protocol.ModePrivate, protocol.ModeFocus} {
		t.Run(string(mode), func(t *testing.T) {
			v21 := &countingV21Client{}
			server := NewServerWithOptions(ServerOptions{V21Client: v21})
			body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"private raw evidence body must stay out","mode":"` + string(mode) + `","trace_id":"a21-trace-ordinary-` + string(mode) + `","session_id":"a21-session-ordinary-` + string(mode) + `"}`)
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
			for _, event := range response.Events {
				var payload protocol.ControlEventPayload
				if err := json.Unmarshal(event.Payload, &payload); err != nil {
					t.Fatal(err)
				}
				if payload.Mode == protocol.ModeProfessional || len(payload.Evidence) > 0 || len(payload.ScreenCards) > 0 || len(payload.SpeechBlocks) > 0 {
					t.Fatalf("ordinary %s mode exposed professional payload: %+v", mode, payload)
				}
				if strings.Contains(payload.Text, "raw evidence body") {
					t.Fatalf("ordinary %s mode echoed sensitive text: %+v", mode, payload)
				}
			}
			if v21.calls != 0 {
				t.Fatalf("v21 calls = %d, want 0", v21.calls)
			}

			traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-ordinary-"+string(mode), nil)
			traceRec := httptest.NewRecorder()
			server.Handler().ServeHTTP(traceRec, traceReq)
			if traceRec.Code != http.StatusOK {
				t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
			}
			for _, forbidden := range []string{"v21.query", "professional.", "raw evidence body"} {
				if strings.Contains(traceRec.Body.String(), forbidden) {
					t.Fatalf("ordinary %s trace leaked %q: %s", mode, forbidden, traceRec.Body.String())
				}
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

func TestProfessionalModeV21StatusFailureRecordsRedactedReasonMarkers(t *testing.T) {
	adapter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "raw query 查一下语音唤醒误触发 http://127.0.0.1:18080/internal", http.StatusBadGateway)
	}))
	defer adapter.Close()
	client, err := v21adapter.NewHTTPClient(adapter.URL)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServerWithOptions(ServerOptions{V21Client: client})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-status","session_id":"a21-session-pro-status"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-status", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	traceBody := traceRec.Body.String()
	for _, want := range []string{"v21.query.error", "v21.query.error.upstream_status", "v21.query.error.status_5xx", "v21.query.utterance.length_"} {
		if !strings.Contains(traceBody, want) {
			t.Fatalf("trace missing %q: %s", want, traceBody)
		}
	}
	for _, forbidden := range []string{"查一下语音唤醒误触发", adapter.URL, "127.0.0.1:18080", "/internal"} {
		if strings.Contains(traceBody, forbidden) {
			t.Fatalf("trace leaked %q: %s", forbidden, traceBody)
		}
	}
}

func TestProfessionalModeV21JSONErrorBodyOverridesStatusMarker(t *testing.T) {
	adapter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{
			"code":"no_evidence",
			"status_class":"status_4xx",
			"detail":"raw query 查一下语音唤醒误触发 http://127.0.0.1:18080/internal"
		}`))
	}))
	defer adapter.Close()
	client, err := v21adapter.NewHTTPClient(adapter.URL)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServerWithOptions(ServerOptions{V21Client: client})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-json-error","session_id":"a21-session-pro-json-error"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-json-error", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	traceBody := traceRec.Body.String()
	for _, want := range []string{"v21.query.error", "v21.query.error.no_evidence", "v21.query.error.status_4xx"} {
		if !strings.Contains(traceBody, want) {
			t.Fatalf("trace missing %q: %s", want, traceBody)
		}
	}
	if strings.Contains(traceBody, "v21.query.error.upstream_status") {
		t.Fatalf("trace kept generic upstream_status instead of body code: %s", traceBody)
	}
	for _, forbidden := range []string{"查一下语音唤醒误触发", adapter.URL, "127.0.0.1:18080", "/internal"} {
		if strings.Contains(traceBody, forbidden) {
			t.Fatalf("trace leaked %q: %s", forbidden, traceBody)
		}
	}
}

func TestProfessionalModeV21ContractFailureRecordsReasonMarker(t *testing.T) {
	adapter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"trace_id":"a21-trace-pro-contract-invalid",
			"fast_answer":"raw answer that must not leak",
			"confidence":0.8,
			"speech_blocks":["raw speech block"]
		}`))
	}))
	defer adapter.Close()
	client, err := v21adapter.NewHTTPClient(adapter.URL)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServerWithOptions(ServerOptions{V21Client: client})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下合同证据","mode":"professional","trace_id":"a21-trace-pro-contract-invalid","session_id":"a21-session-pro-contract-invalid"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-contract-invalid", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	traceBody := traceRec.Body.String()
	if !strings.Contains(traceBody, "v21.query.error.contract_invalid") {
		t.Fatalf("trace missing contract_invalid marker: %s", traceBody)
	}
	for _, forbidden := range []string{"查一下合同证据", "raw answer", "raw speech"} {
		if strings.Contains(traceBody, forbidden) {
			t.Fatalf("trace leaked %q: %s", forbidden, traceBody)
		}
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
	if _, ok := reply["a21"]; ok {
		t.Fatalf("stock hello reply leaked a21 debug allowance: %#v", reply)
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

func TestXiaozhiWebSocketDebugProfileHelloReplyIncludesA21DeviceEventsAllowance(t *testing.T) {
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
	for _, forbidden := range []string{"debug_metrics", `"features"`} {
		if strings.Contains(strings.ToLower(string(replyJSON)), forbidden) {
			t.Fatalf("hello reply leaked debug field %q: %s", forbidden, replyJSON)
		}
	}
	a21, ok := reply["a21"].(map[string]any)
	if !ok {
		t.Fatalf("hello reply missing a21 debug allowance: %#v", reply)
	}
	if a21["profile"] != "debug" {
		t.Fatalf("a21 profile = %#v, want debug in %#v", a21["profile"], a21)
	}
	deviceEvents, ok := a21["device_events"].(bool)
	if !ok || !deviceEvents {
		t.Fatalf("a21 device_events = %#v, want true in %#v", a21["device_events"], a21)
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

func TestXiaozhiControlEndpointDeliversFirmwareCompatibleDeviceEvent(t *testing.T) {
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
		},
	})
	readXiaozhiJSON(t, ctx, conn)

	body := bytes.NewBufferString(`{"device_id":"stackchan-debug-001","event":"motion","name":"nod","reason":"acceptance","y_angle":120,"trace_id":"a21-trace-xiaozhi-control","session_id":"a21-session-xiaozhi-control"}`)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(httpServer.URL+"/v1/xiaozhi/control", "application/json", body)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	var delivered map[string]any
	if err := wsjson.Read(ctx, conn, &delivered); err != nil {
		t.Fatal(err)
	}
	if delivered["type"] != "device" || delivered["event"] != "motion" || delivered["name"] != "nod" {
		t.Fatalf("delivered = %#v, want firmware-compatible motion command", delivered)
	}
	if delivered["kind"] != nil || delivered["motion"] != nil {
		t.Fatalf("delivered leaked legacy server-only device fields: %#v", delivered)
	}
	if delivered["y_angle"] != float64(85) {
		t.Fatalf("y_angle = %#v, want clamped 85", delivered["y_angle"])
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
		var response XiaozhiDeviceControlResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.Status != "delivered" || response.DeliveredTransport != "xiaozhi_ws" || response.Event != "motion" || response.Value != "nod" {
			t.Fatalf("response = %+v, want delivered motion nod", response)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-control")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	traceBody, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"xiaozhi.device_command.motion", "xiaozhi.device_command.delivered"} {
		if !strings.Contains(string(traceBody), want) {
			t.Fatalf("trace missing %q: %s", want, traceBody)
		}
	}
}

func TestXiaozhiControlEndpointRejectsStockProfile(t *testing.T) {
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
		"device_id": "stackchan-stock-001",
	})
	readXiaozhiJSON(t, ctx, conn)

	resp, err := http.Post(httpServer.URL+"/v1/xiaozhi/control", "application/json", bytes.NewBufferString(`{"device_id":"stackchan-stock-001","event":"face","emotion":"happy"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 409: %s", resp.StatusCode, data)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)
}

func TestOfficialStackChanControlEndpointDeliversOfficialMotionFrame(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/stackChan/ws?deviceType=StackChan&device_id=stackchan-official-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-official-001","event":"motion","name":"look_up","y_angle":120,"trace_id":"a21-trace-stackchan-official-control","session_id":"a21-session-stackchan-official-control"}`)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(httpServer.URL+"/v1/stackchan/official/control", "application/json", body)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	msgType, frame, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if msgType != websocket.MessageBinary {
		t.Fatalf("message type = %v, want binary", msgType)
	}
	if len(frame) < 5 || frame[0] != 0x04 {
		t.Fatalf("frame header = %#v, want official ControlMotion", frame[:min(len(frame), 5)])
	}
	if got := binary.BigEndian.Uint32(frame[1:5]); got != uint32(len(frame)-5) {
		t.Fatalf("frame length = %d, want %d", got, len(frame)-5)
	}
	var payload map[string]map[string]int
	if err := json.Unmarshal(frame[5:], &payload); err != nil {
		t.Fatal(err)
	}
	if payload["pitchServo"]["angle"] != 850 {
		t.Fatalf("payload = %#v, want y_angle 120 clamped to official 850 units", payload)
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
		var response XiaozhiDeviceControlResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.Status != "delivered" || response.DeliveredTransport != "stackchan_official_ws" {
			t.Fatalf("response = %+v, want official delivery", response)
		}
		if response.PacketCount != 1 ||
			response.OfficialActionPhysicalAccepted == nil ||
			*response.OfficialActionPhysicalAccepted ||
			response.OfficialActionSurfaces["servo_y"] != "pitch_clamped" ||
			response.OfficialActionSurfaces["servo_x"] != "not_used" ||
			response.OfficialActionSurfaces["rgb"] != "unchanged_no_rgb_frame" {
			t.Fatalf("response = %+v, want official action metadata without physical acceptance", response)
		}
		registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
		runtimeEcho, ok := registry["runtime_echo"].(map[string]any)
		if !ok {
			t.Fatalf("registry = %#v, want runtime_echo", registry)
		}
		if runtimeEcho["official_stackchan_packets"] != "1" ||
			runtimeEcho["official_stackchan_physical_accepted"] != "false" ||
			runtimeEcho["official_stackchan_servo_y"] != "pitch_clamped" ||
			runtimeEcho["official_stackchan_servo_x"] != "not_used" ||
			runtimeEcho["official_stackchan_rgb"] != "unchanged_no_rgb_frame" {
			t.Fatalf("runtime_echo = %#v, want official action metadata without physical acceptance", runtimeEcho)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestOfficialStackChanControlEndpointRequiresConnectedOfficialSocket(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	resp, err := http.Post(httpServer.URL+"/v1/stackchan/official/control", "application/json", bytes.NewBufferString(`{"device_id":"stackchan-official-001","event":"face","emotion":"happy"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 409: %s", resp.StatusCode, data)
	}
}
