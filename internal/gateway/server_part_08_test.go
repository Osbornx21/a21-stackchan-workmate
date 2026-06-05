package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestOfficialStackChanControlEndpointFallsBackToXiaozhiMCPWhenAllowed(t *testing.T) {
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
		"device_id":  "44:1b:f6:e2:6a:60",
		"trace_id":   "a21-trace-official-mcp-fallback",
		"session_id": "a21-session-official-mcp-fallback",
		"features": map[string]any{
			"mcp": true,
			"aec": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(
			httpServer.URL+"/v1/stackchan/official/control",
			"application/json",
			bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","event":"face","emotion":"happy","allow_mcp_fallback":true,"trace_id":"a21-trace-official-mcp-fallback","session_id":"a21-session-official-mcp-fallback"}`),
		)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	ledMessage := readXiaozhiJSON(t, ctx, conn)
	assertXiaozhiMCPMessage(t, ledMessage, xiaozhiMCPRobotSetLEDColorToolName, map[string]any{
		"red":   float64(0),
		"green": float64(168),
		"blue":  float64(80),
	})
	headMessage := readXiaozhiJSON(t, ctx, conn)
	assertXiaozhiMCPMessage(t, headMessage, xiaozhiMCPRobotSetHeadAnglesToolName, map[string]any{
		"yaw":   float64(45),
		"pitch": float64(68),
		"speed": float64(680),
	})

	var response XiaozhiDeviceControlResponse
	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d: %s", resp.StatusCode, body)
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if response.Status != "fallback_delivered" ||
		response.DeliveredTransport != "xiaozhi_mcp_sequence" ||
		response.Event != "face" ||
		response.Value != "happy" ||
		response.PacketCount != 0 ||
		response.OfficialActionPhysicalAccepted == nil ||
		*response.OfficialActionPhysicalAccepted ||
		response.OfficialActionFallbackReason != "official_stackchan_ws_disconnected" {
		t.Fatalf("response = %+v, want mcp fallback without physical acceptance", response)
	}
	if response.OfficialActionSurfaces["fallback"] != "xiaozhi_mcp_sequence" ||
		response.OfficialActionSurfaces["fallback_name"] != "body_preset:celebrate" ||
		response.OfficialActionSurfaces["official_relay"] != "disconnected" ||
		response.OfficialActionSurfaces["packet_delivery_status"] != "not_sent_no_official_ws" ||
		response.OfficialActionSurfaces["planned_packet_count"] != "1" ||
		response.OfficialActionSurfaces["planned_avatar"] != "happy" {
		t.Fatalf("surfaces = %#v, want explicit official relay fallback metadata", response.OfficialActionSurfaces)
	}

	for _, want := range []string{
		"xiaozhi.mcp.robot_led_color.sent",
		"stackchan.official_mcp_fallback.face.happy.step1.robot_led_color.sent",
		"stackchan.official_mcp_fallback.face.happy.step2.robot_head_angles_set.sent",
		"stackchan.official_mcp_fallback.delivered",
	} {
		if !traceContains(server.traceEvents("a21-trace-official-mcp-fallback"), want) {
			t.Fatalf("trace missing %q: %+v", want, server.traceEvents("a21-trace-official-mcp-fallback"))
		}
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "stackchan.official_mcp_fallback.face" {
		t.Fatalf("last event = %#v, want official mcp fallback", registry["last_event"])
	}
	runtimeEcho, ok := registry["runtime_echo"].(map[string]any)
	if !ok {
		t.Fatalf("registry = %#v, want runtime_echo", registry)
	}
	for key, want := range map[string]any{
		"official_stackchan_packets":           "0",
		"official_stackchan_planned_packets":   "1",
		"official_stackchan_physical_accepted": "false",
		"official_stackchan_fallback":          "xiaozhi_mcp_sequence",
		"official_stackchan_fallback_name":     "body_preset:celebrate",
		"official_stackchan_fallback_status":   "delivered",
		"official_stackchan_official_relay":    "disconnected",
		"official_stackchan_planned_avatar":    "happy",
	} {
		if runtimeEcho[key] != want {
			t.Fatalf("runtime_echo[%s] = %#v, want %#v in %#v", key, runtimeEcho[key], want, runtimeEcho)
		}
	}
}

func TestOfficialStackChanStatusReportsDisconnectedAndNextAction(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	resp, err := http.Get(httpServer.URL + "/v1/stackchan/official/status?device_id=stackchan-official-001")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200: %s", resp.StatusCode, data)
	}
	var status map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status["schema_version"] != "a21.stackchan.official.status.v1" ||
		status["device_id"] != "stackchan-official-001" ||
		status["connected"] != false ||
		status["fallback_available"] != true ||
		status["delivered_transport"] != "xiaozhi_mcp_fallback_available" ||
		status["next_action"] != "connect_official_stackchan_ws" {
		t.Fatalf("status = %#v, want disconnected official relay with clear next action", status)
	}
}

func TestOfficialStackChanStatusReportsConnectedFallbackSocket(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/stackChan/ws"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	resp, err := http.Get(httpServer.URL + "/v1/stackchan/official/status?device_id=44:1b:f6:e2:6a:60")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200: %s", resp.StatusCode, data)
	}
	var status map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status["connected"] != true ||
		status["official_device_id"] != defaultOfficialStackChanDeviceID ||
		status["delivered_transport"] != "stackchan_official_ws" ||
		status["next_action"] != "send_official_control_and_collect_physical_acceptance" {
		t.Fatalf("status = %#v, want product device routed through default official StackChan socket", status)
	}
	if connectedSince, ok := status["connected_since_ms"].(float64); !ok || connectedSince <= 0 {
		t.Fatalf("connected_since_ms = %#v, want positive timestamp", status["connected_since_ms"])
	}
}

func TestOfficialStackChanStatusAndControlNormalizeHardwareMACCase(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	officialID := "44:1B:F6:E2:6A:60"
	productID := "44:1b:f6:e2:6a:60"
	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/stackChan/ws?device_id="+url.QueryEscape(officialID)), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	resp, err := http.Get(httpServer.URL + "/v1/stackchan/official/status?device_id=" + url.QueryEscape(productID))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200: %s", resp.StatusCode, data)
	}
	var status OfficialStackChanStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.Connected ||
		status.DeviceID != productID ||
		status.OfficialDeviceID != productID ||
		status.DeliveredTransport != "stackchan_official_ws" ||
		status.NextAction != "send_official_control_and_collect_physical_acceptance" {
		t.Fatalf("status = %+v, want lowercase product id connected to uppercase official relay", status)
	}

	controlBody := bytes.NewBufferString(`{"device_id":"` + productID + `","event":"motion","name":"look_up","y_angle":120,"trace_id":"a21-trace-mac-case","session_id":"a21-session-mac-case"}`)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(httpServer.URL+"/v1/stackchan/official/control", "application/json", controlBody)
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
	if msgType != websocket.MessageBinary || len(frame) < 5 || frame[0] != 0x04 {
		t.Fatalf("frame type=%v frame=%#v, want official ControlMotion", msgType, frame[:min(len(frame), 5)])
	}
	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			t.Fatalf("control status = %d, want 200: %s", resp.StatusCode, data)
		}
		var control XiaozhiDeviceControlResponse
		if err := json.NewDecoder(resp.Body).Decode(&control); err != nil {
			t.Fatal(err)
		}
		if control.Status != "delivered" || control.DeliveredTransport != "stackchan_official_ws" {
			t.Fatalf("control = %+v, want official delivery through normalized MAC key", control)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	devicesResp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	defer devicesResp.Body.Close()
	var devices struct {
		Devices []map[string]any `json:"devices"`
	}
	if err := json.NewDecoder(devicesResp.Body).Decode(&devices); err != nil {
		t.Fatal(err)
	}
	var matches int
	for _, device := range devices.Devices {
		if strings.EqualFold(fmt.Sprint(device["device_id"]), productID) {
			matches++
			if device["device_id"] != productID {
				t.Fatalf("device_id = %#v, want normalized lowercase MAC", device["device_id"])
			}
		}
	}
	if matches != 1 {
		t.Fatalf("devices = %#v, want one normalized MAC record", devices.Devices)
	}
}

func TestOfficialStackChanStatusReportsDeliveryMetadata(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/stackChan/ws?device_id=stackchan-official-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-official-001","event":"motion","name":"look_up","y_angle":120,"trace_id":"a21-trace-stackchan-official-status","session_id":"a21-session-stackchan-official-status"}`)
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

	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatal(err)
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
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	statusResp, err := http.Get(httpServer.URL + "/v1/stackchan/official/status?device_id=stackchan-official-001")
	if err != nil {
		t.Fatal(err)
	}
	defer statusResp.Body.Close()
	if statusResp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(statusResp.Body)
		t.Fatalf("status = %d, want 200: %s", statusResp.StatusCode, data)
	}
	var status map[string]any
	if err := json.NewDecoder(statusResp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	surfaces, ok := status["official_action_surfaces"].(map[string]any)
	if !ok {
		t.Fatalf("status = %#v, want official_action_surfaces", status)
	}
	if status["connected"] != true ||
		status["last_trace_id"] != "a21-trace-stackchan-official-status" ||
		status["last_session_id"] != "a21-session-stackchan-official-status" ||
		status["last_event"] != "stackchan.official_control.motion" ||
		status["last_packet_count"] != float64(1) ||
		status["physical_accepted"] != false ||
		status["next_action"] != "send_official_control_and_collect_physical_acceptance" ||
		surfaces["servo_y"] != "pitch_clamped" ||
		surfaces["servo_x"] != "not_used" ||
		surfaces["rgb"] != "unchanged_no_rgb_frame" {
		t.Fatalf("status = %#v, want latest official delivery metadata", status)
	}
}

func TestXiaozhiTurnLifecycleFansOutToOfficialStackChanState(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	officialConn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/stackChan/ws?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = officialConn.Close(websocket.StatusNormalClosure, "test done") })

	xiaozhiConn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = xiaozhiConn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, xiaozhiConn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-official-state",
		"session_id": "a21-session-xiaozhi-official-state",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, xiaozhiConn)

	if err := wsjson.Write(ctx, xiaozhiConn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, xiaozhiConn)
	assertOfficialStackChanState(t, ctx, officialConn, "listening")

	if err := wsjson.Write(ctx, xiaozhiConn, map[string]any{"type": "abort", "reason": "wake_word_detected"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, xiaozhiConn)
	if stop["type"] != "tts" || stop["state"] != "stop" {
		t.Fatalf("abort stop = %#v", stop)
	}
	assertOfficialStackChanState(t, ctx, officialConn, "listening")

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-official-state")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"stackchan.official_auto.state",
		"stackchan.official_auto.delivered",
		"stackchan.display_state.received",
		"stackchan.display_state.normalized",
		"stackchan.display_state.registry_updated",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["display_state"] != string(protocol.DisplayStateListening) ||
		registry["display_state_source"] != "xiaozhi" ||
		registry["display_state_trace_id"] != "a21-trace-xiaozhi-official-state" ||
		registry["display_state_session_id"] != "a21-session-xiaozhi-official-state" ||
		registry["display_state_physical_accepted"] != false {
		t.Fatalf("display registry = %#v, want listening xiaozhi non-accepted state", registry)
	}
	if updated, ok := registry["display_state_updated_at_ms"].(float64); !ok || updated <= 0 {
		t.Fatalf("display updated_at = %#v, want positive ms", registry["display_state_updated_at_ms"])
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok || capabilities["xiaozhi_feature_mcp"] != "true" {
		t.Fatalf("display update erased xiaozhi capabilities: %#v", registry)
	}
}

func TestDeviceEventDisplayStateRegistryNormalizesAndPreservesState(t *testing.T) {
	server := NewServer()
	event := protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-display-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       1,
		TraceID:   "a21-trace-display-device-event",
		SessionID: "a21-session-display-device-event",
	}
	server.recordDeviceEvent(event, protocol.DeviceEventPayload{
		Event:        protocol.DeviceEventRuntimeEcho,
		DisplayState: protocol.NormalizeOfficialDisplayState("x21-render"),
		Capabilities: map[string]string{
			"screen": "available",
		},
		RuntimeEcho: map[string]string{
			"battery": "planned_diagnostic",
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["display_state"] != string(protocol.DisplayStateError) ||
		registry["display_state_source"] != "device_event" ||
		registry["display_state_trace_id"] != "a21-trace-display-device-event" ||
		registry["display_state_session_id"] != "a21-session-display-device-event" ||
		registry["display_state_physical_accepted"] != false {
		t.Fatalf("display registry = %#v, want normalized error device_event state", registry)
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok || capabilities["screen"] != "available" {
		t.Fatalf("capabilities erased by display state update: %#v", registry)
	}
	runtimeEcho, ok := registry["runtime_echo"].(map[string]any)
	if !ok || runtimeEcho["battery"] != "planned_diagnostic" {
		t.Fatalf("runtime_echo erased by display state update: %#v", registry)
	}
}

func TestXiaozhiOpusDownlinkFansOutOfficialStackChanSpeaking(t *testing.T) {
	server := NewServer()
	session := &xiaozhiSession{
		deviceID:  "stackchan-001",
		traceID:   "a21-trace-xiaozhi-official-speaking",
		sessionID: "a21-session-xiaozhi-official-speaking",
	}
	turn := session.startXiaozhiTurn(context.Background(), protocol.ModeWorkmate)
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	officialConn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/stackChan/ws?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = officialConn.Close(websocket.StatusNormalClosure, "test done") })

	xiaozhiRelay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	t.Cleanup(xiaozhiRelay.Close)

	xiaozhiConn, _, err := websocket.Dial(ctx, webSocketURL(xiaozhiRelay.URL, ""), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = xiaozhiConn.Close(websocket.StatusNormalClosure, "test done") })
	readXiaozhiBinary(t, ctx, xiaozhiConn)
	assertOfficialStackChanState(t, ctx, officialConn, "speaking")

	if !traceContains(server.traceEvents("a21-trace-xiaozhi-official-speaking"), "stackchan.official_auto.delivered") {
		t.Fatalf("trace missing official speaking fanout: %+v", server.traceEvents("a21-trace-xiaozhi-official-speaking"))
	}
}

func TestXiaozhiDebugProfileRecordsPlaybackStartDeviceEvent(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-playback",
		"session_id": "a21-session-xiaozhi-playback",
		"device_id":  "stackchan-debug-001",
		"features": map[string]any{
			"mcp":           true,
			"aec":           true,
			"device_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "device",
		"kind":       "playback",
		"playback":   "start",
		"stream_id":  "a21-xiaozhi-stream-001",
		"trace_id":   "a21-trace-xiaozhi-playback",
		"session_id": "a21-session-xiaozhi-playback",
		"device_id":  "stackchan-debug-001",
	}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-playback")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	body, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "device.playback.start") {
		t.Fatalf("trace missing device playback start: %s", body)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "device.playback.start" {
		t.Fatalf("last event = %#v, want device.playback.start", registry["last_event"])
	}
}

func TestXiaozhiDebugProfileRecordsPlaybackStopDoneDeviceEvent(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-playback-stop-done",
		"session_id": "a21-session-xiaozhi-playback-stop-done",
		"device_id":  "stackchan-debug-001",
		"features": map[string]any{
			"mcp":           true,
			"aec":           true,
			"device_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "device",
		"kind":       "playback",
		"playback":   "stop_done",
		"stream_id":  "a21-xiaozhi-stream-001",
		"trace_id":   "a21-trace-xiaozhi-playback-stop-done",
		"session_id": "a21-session-xiaozhi-playback-stop-done",
		"device_id":  "stackchan-debug-001",
	}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-playback-stop-done")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	body, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "device.playback.stop_done") {
		t.Fatalf("trace missing device playback stop_done: %s", body)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "device.playback.stop_done" {
		t.Fatalf("last event = %#v, want device.playback.stop_done", registry["last_event"])
	}
}

func TestXiaozhiDebugProfileRejectsUnsafePlaybackStreamID(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-unsafe-playback",
		"session_id": "a21-session-xiaozhi-unsafe-playback",
		"device_id":  "stackchan-debug-001",
		"features": map[string]any{
			"mcp":           true,
			"aec":           true,
			"device_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "device",
		"kind":       "playback",
		"playback":   "start",
		"stream_id":  "sk-test-secret",
		"trace_id":   "a21-trace-xiaozhi-unsafe-playback",
		"session_id": "a21-session-xiaozhi-unsafe-playback",
		"device_id":  "stackchan-debug-001",
	}); err != nil {
		t.Fatal(err)
	}
	reply := readXiaozhiJSON(t, ctx, conn)
	replyJSON := mustJSON(t, reply)
	if reply["type"] != "error" || reply["code"] != "unsupported_device_event_value" {
		t.Fatalf("reply = %#v, want unsupported_device_event_value error", reply)
	}
	for _, forbidden := range []string{"sk-test-secret", "secret", "token", "/Users", "http://"} {
		if strings.Contains(strings.ToLower(replyJSON), strings.ToLower(forbidden)) {
			t.Fatalf("unsafe playback error leaked %q: %s", forbidden, replyJSON)
		}
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-unsafe-playback")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	body, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "device.playback.start") {
		t.Fatalf("unsafe stream id recorded playback start: %s", body)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if _, ok := registry["playback_stream_id"]; ok {
		t.Fatalf("unsafe stream id stored in registry: %#v", registry)
	}
}

func TestXiaozhiStockProfileRejectsPlaybackStartDeviceEvent(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-stock-playback",
		"session_id": "a21-session-xiaozhi-stock-playback",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":      "device",
		"kind":      "playback",
		"playback":  "start",
		"stream_id": "a21-xiaozhi-stream-001",
	}); err != nil {
		t.Fatal(err)
	}
	reply := readXiaozhiJSON(t, ctx, conn)
	if reply["type"] != "error" || reply["code"] != "device_events_disabled" {
		t.Fatalf("reply = %#v, want device_events_disabled error", reply)
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-stock-playback")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	body, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "device.playback.start") {
		t.Fatalf("stock profile recorded playback start: %s", body)
	}
}

func TestXiaozhiProductPlaybackEventsAllowanceRecordsPlaybackStart(t *testing.T) {
	httpServer := httptest.NewServer(NewServerWithOptions(ServerOptions{XiaozhiProductPlaybackEvents: true}).Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-product-playback",
		"session_id": "a21-session-xiaozhi-product-playback",
		"device_id":  "44:1b:f6:e2:6a:60",
		"features": map[string]any{
			"mcp":              true,
			"aec":              true,
			"playback_events":  true,
			"keepalive_events": true,
		},
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	replyJSON := mustJSON(t, reply)
	for _, forbidden := range []string{"debug_metrics", `"device_events":true`, `"profile":"debug"`} {
		if strings.Contains(strings.ToLower(replyJSON), strings.ToLower(forbidden)) {
			t.Fatalf("product playback hello leaked debug field %q: %s", forbidden, replyJSON)
		}
	}
	a21, ok := reply["a21"].(map[string]any)
	if !ok || a21["profile"] != "product" || a21["playback_events"] != true || a21["keepalive_events"] != true {
		t.Fatalf("a21 hello extension = %#v, want product playback allowance", reply["a21"])
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":  "device",
		"kind":  "state",
		"state": "speaking",
	}); err != nil {
		t.Fatal(err)
	}
	rejected := readXiaozhiJSON(t, ctx, conn)
	if rejected["type"] != "error" || rejected["code"] != "unsupported_device_event" {
		t.Fatalf("product playback allowance accepted non-playback event: %#v", rejected)
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":      "device",
		"kind":      "playback",
		"playback":  "start",
		"stream_id": "a21-xiaozhi-stream-001",
	}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-product-playback")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	body, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "device.playback.start") {
		t.Fatalf("trace missing product playback start: %s", body)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "device.playback.start" || registry["playback_stream_id"] != "a21-xiaozhi-stream-001" {
		t.Fatalf("registry playback = %#v, want playback start and stream", registry)
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok ||
		capabilities["xiaozhi_profile"] != "stock" ||
		capabilities["xiaozhi_feature_playback_events"] != "true" ||
		capabilities["xiaozhi_product_playback_events"] != "true" {
		t.Fatalf("registry capabilities = %#v, want stock product playback events", registry["capabilities"])
	}
	if _, ok := capabilities["xiaozhi_debug_extension_isolated"]; ok {
		t.Fatalf("product playback allowance marked debug capabilities: %#v", capabilities)
	}
}
