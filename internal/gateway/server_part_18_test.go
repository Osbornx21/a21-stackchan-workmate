package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

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

func TestGatewayDeviceRegistryExposesExplicitModeStatesWithoutProfessionalBodies(t *testing.T) {
	tests := []struct {
		mode protocol.Mode
		expr protocol.ExpressionState
	}{
		{mode: protocol.ModePublic, expr: protocol.ExpressionListening},
		{mode: protocol.ModePrivate, expr: protocol.ExpressionListening},
		{mode: protocol.ModeFocus, expr: protocol.ExpressionIdle},
		{mode: protocol.ModeProfessional, expr: protocol.ExpressionProfessional},
		{mode: protocol.ModeMuted, expr: protocol.ExpressionIdle},
		{mode: protocol.ModeWorkmate, expr: protocol.ExpressionListening},
	}
	for _, tc := range tests {
		t.Run(string(tc.mode), func(t *testing.T) {
			server := NewServer()
			traceID := "a21-trace-registry-" + string(tc.mode)
			sessionID := "a21-session-registry-" + string(tc.mode)
			server.controlSequence("stackchan-sim-001", traceID, sessionID, []protocol.ControlEventPayload{{
				State:        tc.expr,
				Mode:         tc.mode,
				Text:         "professional answer body must not persist",
				Evidence:     []protocol.EvidenceItem{{Title: "private evidence title", Summary: "private evidence summary"}},
				ScreenCards:  []protocol.ScreenCard{{Label: "card", Text: "private card body"}},
				SpeechBlocks: []string{"private speech block"},
			}})

			req := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
			}
			var registry DeviceRegistryResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &registry); err != nil {
				t.Fatal(err)
			}
			if len(registry.Devices) != 1 {
				t.Fatalf("devices = %d, want 1: %s", len(registry.Devices), rec.Body.String())
			}
			device := registry.Devices[0]
			if device.CurrentMode != tc.mode || device.CurrentExpr != tc.expr {
				t.Fatalf("device state = %+v, want mode=%s expr=%s", device, tc.mode, tc.expr)
			}
			for _, forbidden := range []string{"professional answer body", "private evidence title", "private evidence summary", "private card body", "private speech block"} {
				if strings.Contains(rec.Body.String(), forbidden) {
					t.Fatalf("registry leaked %q: %s", forbidden, rec.Body.String())
				}
			}
		})
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

func TestDeviceControlArmsMultiplePhysicalMockPlaybackChunksForNextAudioFrame(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening","mode":"workmate","trace_id":"a21-trace-armed-multi","session_id":"a21-session-armed-multi","mock_playback_on_next_audio_frame":true,"mock_audio_chunks":4}`)
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

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-armed-multi", "a21-session-armed-multi", pcm16Base64WithSample(0))
	readControlEvents(t, ctx, conn, 2)
	for i := 0; i < 4; i++ {
		var playback protocol.Envelope
		if err := wsjson.Read(ctx, conn, &playback); err != nil {
			t.Fatal(err)
		}
		if playback.Kind != protocol.KindAudioPlaybackChunk {
			t.Fatalf("envelope %d kind = %q, want playback chunk", i, playback.Kind)
		}
	}

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 2, "a21-trace-armed-multi", "a21-session-armed-multi", pcm16Base64WithSample(0))
	assertNoEnvelope(t, conn, 100*time.Millisecond)
}
