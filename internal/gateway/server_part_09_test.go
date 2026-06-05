package gateway

import (
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
	stackchantransport "a21.local/a21/internal/transport/stackchan"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestXiaozhiProductKeepaliveEventsAllowanceRecordsHeartbeat(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-product-keepalive",
		"session_id": "a21-session-xiaozhi-product-keepalive",
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
			t.Fatalf("product keepalive hello leaked debug field %q: %s", forbidden, replyJSON)
		}
	}
	a21, ok := reply["a21"].(map[string]any)
	if !ok || a21["profile"] != "product" || a21["keepalive_events"] != true {
		t.Fatalf("a21 hello extension = %#v, want product keepalive allowance", reply["a21"])
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
		t.Fatalf("product keepalive allowance accepted state event: %#v", rejected)
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":                   "device",
		"kind":                   "heartbeat",
		"trace_id":               "a21-trace-xiaozhi-product-keepalive",
		"session_id":             "a21-session-xiaozhi-product-keepalive",
		"device_id":              "44:1b:f6:e2:6a:60",
		"battery_level":          73,
		"battery_charging":       false,
		"battery_discharging":    true,
		"external_power":         false,
		"power_source":           "battery_discharging",
		"pmic_power_key_profile": "a21_stackchan_axp2101_pwrkey_v1",
	}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-product-keepalive")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	body, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "device.heartbeat") {
		t.Fatalf("trace missing product heartbeat: %s", body)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "device.heartbeat" {
		t.Fatalf("registry heartbeat = %#v, want device.heartbeat", registry)
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok ||
		capabilities["xiaozhi_feature_keepalive_events"] != "true" ||
		capabilities["xiaozhi_product_keepalive_events"] != "true" {
		t.Fatalf("registry capabilities = %#v, want product keepalive events", registry["capabilities"])
	}
	if _, ok := capabilities["xiaozhi_debug_extension_isolated"]; ok {
		t.Fatalf("product keepalive allowance marked debug capabilities: %#v", capabilities)
	}
	runtimeEcho, ok := registry["runtime_echo"].(map[string]any)
	if !ok ||
		runtimeEcho["battery_level"] != "73" ||
		runtimeEcho["battery_discharging"] != "true" ||
		runtimeEcho["external_power"] != "false" ||
		runtimeEcho["power_source"] != "battery_discharging" ||
		runtimeEcho["pmic_power_key_profile"] != "a21_stackchan_axp2101_pwrkey_v1" {
		t.Fatalf("registry runtime_echo = %#v, want heartbeat power diagnostics", registry["runtime_echo"])
	}
}

func TestXiaozhiProductTouchEventsAllowanceRecordsTouch(t *testing.T) {
	httpServer := httptest.NewServer(NewServerWithOptions(ServerOptions{XiaozhiProductTouchEvents: true}).Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-product-touch",
		"session_id": "a21-session-xiaozhi-product-touch",
		"device_id":  "44:1b:f6:e2:6a:60",
		"features": map[string]any{
			"mcp":          true,
			"aec":          true,
			"touch_events": true,
		},
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	replyJSON := mustJSON(t, reply)
	for _, forbidden := range []string{"debug_metrics", `"device_events":true`, `"profile":"debug"`} {
		if strings.Contains(strings.ToLower(replyJSON), strings.ToLower(forbidden)) {
			t.Fatalf("product touch hello leaked debug field %q: %s", forbidden, replyJSON)
		}
	}
	a21, ok := reply["a21"].(map[string]any)
	if !ok || a21["profile"] != "product" || a21["touch_events"] != true {
		t.Fatalf("a21 hello extension = %#v, want product touch allowance", reply["a21"])
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":    "device",
		"kind":    "motion",
		"name":    "nod",
		"y_angle": 20,
	}); err != nil {
		t.Fatal(err)
	}
	rejected := readXiaozhiJSON(t, ctx, conn)
	if rejected["type"] != "error" || rejected["code"] != "unsupported_device_event" {
		t.Fatalf("product touch allowance accepted motion event: %#v", rejected)
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "device",
		"kind":       "touch",
		"touch":      "top_swipe_forward",
		"source":     "top_sensor",
		"trace_id":   "a21-trace-xiaozhi-product-touch",
		"session_id": "a21-session-xiaozhi-product-touch",
		"device_id":  "44:1b:f6:e2:6a:60",
	}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-product-touch")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	body, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "device.touch.top.swipe_forward.received") {
		t.Fatalf("trace missing product touch event: %s", body)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "touch.top.swipe_forward" || registry["last_touch_event"] != "touch.top.swipe_forward" || registry["last_touch_source"] != "top_sensor" {
		t.Fatalf("registry touch = %#v, want top swipe forward source", registry)
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok ||
		capabilities["xiaozhi_feature_touch_events"] != "true" ||
		capabilities["xiaozhi_product_touch_events"] != "true" {
		t.Fatalf("registry capabilities = %#v, want product touch events", registry["capabilities"])
	}
	if _, ok := capabilities["xiaozhi_debug_extension_isolated"]; ok {
		t.Fatalf("product touch allowance marked debug capabilities: %#v", capabilities)
	}
}

func TestXiaozhiProductTouchBargeInCancelsActiveTurnAndStopsPlayback(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{XiaozhiProductTouchEvents: true})
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
		"trace_id":   "a21-trace-xiaozhi-touch-barge",
		"session_id": "a21-session-xiaozhi-touch-barge",
		"device_id":  "44:1b:f6:e2:6a:60",
		"features": map[string]any{
			"mcp":          true,
			"aec":          true,
			"touch_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)

	socket, ok := server.xiaozhiSocket("44:1b:f6:e2:6a:60")
	if !ok || socket.session == nil {
		t.Fatal("xiaozhi session missing after hello")
	}
	turn := socket.session.startXiaozhiTurn(ctx, protocol.ModeWorkmate)
	socket.session.resetXiaozhiTTSStop()

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "device",
		"kind":       "touch",
		"touch":      "screen_barge_in",
		"source":     "screen",
		"trace_id":   "a21-trace-xiaozhi-touch-barge",
		"session_id": "a21-session-xiaozhi-touch-barge",
		"device_id":  "44:1b:f6:e2:6a:60",
	}); err != nil {
		t.Fatal(err)
	}

	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "touch_barge_in" || stop["turn_id"] != xiaozhiTurnID(turn) {
		t.Fatalf("touch barge stop = %#v", stop)
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-touch-barge")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"device.touch.barge_in.received",
		"xiaozhi.touch.barge_in",
		"barge_in.detected",
		"turn_cancelled",
		"downlink_queue_cleared",
		"playback.stop",
		"xiaozhi.tts.stop",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	if traces.Summary.BargeInStopMS == nil || *traces.Summary.BargeInStopMS != 0 {
		t.Fatalf("barge summary = %+v, want zero-ms touch stop request", traces.Summary)
	}
	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "touch.barge_in" || registry["last_touch_source"] != "screen" {
		t.Fatalf("registry touch barge = %#v", registry)
	}
}

func TestXiaozhiProductTouchReactionsUseOfficialRelayNotXiaozhiMCP(t *testing.T) {
	httpServer := httptest.NewServer(NewServerWithOptions(ServerOptions{
		XiaozhiProductTouchEvents:    true,
		XiaozhiProductTouchReactions: true,
	}).Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	officialConn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/stackChan/ws?device_id="+url.QueryEscape("44:1b:f6:e2:6a:60")), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = officialConn.Close(websocket.StatusNormalClosure, "test done") })

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-product-touch-reaction",
		"session_id": "a21-session-xiaozhi-product-touch-reaction",
		"device_id":  "44:1b:f6:e2:6a:60",
		"features": map[string]any{
			"mcp":          true,
			"aec":          true,
			"touch_events": true,
		},
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	a21, ok := reply["a21"].(map[string]any)
	if !ok || a21["profile"] != "product" || a21["touch_events"] != true || a21["touch_reactions"] != true {
		t.Fatalf("a21 hello extension = %#v, want product touch reactions", reply["a21"])
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "device",
		"kind":       "touch",
		"touch":      "top_swipe_forward",
		"source":     "top_sensor",
		"trace_id":   "a21-trace-xiaozhi-product-touch-reaction",
		"session_id": "a21-session-xiaozhi-product-touch-reaction",
		"device_id":  "44:1b:f6:e2:6a:60",
	}); err != nil {
		t.Fatal(err)
	}

	msgType, frame, err := officialConn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if msgType != websocket.MessageBinary || len(frame) < 5 || frame[0] != stackchantransport.DataTypeControlMotion {
		t.Fatalf("official reaction frame type=%v frame=%#v, want ControlMotion", msgType, frame[:min(len(frame), 5)])
	}
	var motion map[string]map[string]int
	if err := json.Unmarshal(frame[5:], &motion); err != nil {
		t.Fatal(err)
	}
	if motion["pitchServo"]["angle"] != 780 {
		t.Fatalf("official touch reaction motion = %#v, want top swipe forward pitch 780", motion)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-product-touch-reaction")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"device.touch.top.swipe_forward.received",
		"xiaozhi.touch_reaction.official_ws.sent",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	for _, forbidden := range []string{
		"xiaozhi.touch_reaction.robot_led_color.sent",
		"xiaozhi.touch_reaction.robot_head_angles_set.sent",
	} {
		if traceContains(traces.Events, forbidden) {
			t.Fatalf("trace contains forbidden Xiaozhi MCP touch reaction %q: %+v", forbidden, traces.Events)
		}
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "touch.top.swipe_forward" ||
		registry["last_touch_event"] != "touch.top.swipe_forward" ||
		registry["last_touch_source"] != "top_sensor" ||
		registry["last_touch_trace_id"] != "a21-trace-xiaozhi-product-touch-reaction" ||
		registry["last_touch_session_id"] != "a21-session-xiaozhi-product-touch-reaction" {
		t.Fatalf("registry touch = %#v, want touch last_event preserved", registry)
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok || capabilities["xiaozhi_product_touch_reactions"] != "true" {
		t.Fatalf("registry capabilities = %#v, want product touch reactions", registry["capabilities"])
	}
	runtimeEcho, ok := registry["runtime_echo"].(map[string]any)
	if !ok ||
		runtimeEcho["last_touch_reaction_status"] != "delivered" ||
		runtimeEcho["last_touch_reaction_event"] != "top_swipe_forward" ||
		runtimeEcho["last_touch_reaction_transport"] != "stackchan_official_ws" ||
		runtimeEcho["last_touch_reaction_packets"] != "1" ||
		runtimeEcho["last_touch_reaction_servo_y"] != "pitch_clamped" {
		t.Fatalf("registry runtime_echo = %#v, want official touch reaction echo", registry["runtime_echo"])
	}
}

func TestXiaozhiProductTouchReactionsFallbackToXiaozhiMCPWithoutOfficialRelay(t *testing.T) {
	httpServer := httptest.NewServer(NewServerWithOptions(ServerOptions{
		XiaozhiProductTouchEvents:    true,
		XiaozhiProductTouchReactions: true,
	}).Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-product-touch-no-official-relay",
		"session_id": "a21-session-xiaozhi-product-touch-no-official-relay",
		"device_id":  "44:1b:f6:e2:6a:60",
		"features": map[string]any{
			"mcp":          true,
			"aec":          true,
			"touch_events": true,
		},
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	a21, ok := reply["a21"].(map[string]any)
	if !ok || a21["profile"] != "product" || a21["touch_events"] != true {
		t.Fatalf("a21 hello extension = %#v, want product touch allowance", reply["a21"])
	}
	if a21["touch_reactions"] != true {
		t.Fatalf("touch reactions = %#v, want product official relay reaction advertised", a21)
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "device",
		"kind":       "touch",
		"touch":      "top_tap",
		"source":     "top_sensor",
		"trace_id":   "a21-trace-xiaozhi-product-touch-no-official-relay",
		"session_id": "a21-session-xiaozhi-product-touch-no-official-relay",
		"device_id":  "44:1b:f6:e2:6a:60",
	}); err != nil {
		t.Fatal(err)
	}
	ledMessage := readXiaozhiJSON(t, ctx, conn)
	assertXiaozhiMCPMessage(t, ledMessage, xiaozhiMCPRobotSetLEDColorToolName, map[string]any{
		"red":   float64(60),
		"green": float64(0),
		"blue":  float64(168),
	})
	headMessage := readXiaozhiJSON(t, ctx, conn)
	assertXiaozhiMCPMessage(t, headMessage, xiaozhiMCPRobotSetHeadAnglesToolName, map[string]any{
		"pitch": float64(68),
		"speed": float64(680),
	})
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-product-touch-no-official-relay")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"device.touch.top.tap.received",
		"xiaozhi.touch_reaction.official_ws_not_connected",
		"xiaozhi.touch_reaction.xiaozhi_mcp_fallback.step1.robot_led_color.sent",
		"xiaozhi.touch_reaction.xiaozhi_mcp_fallback.step2.robot_head_angles_set.sent",
		"xiaozhi.touch_reaction.xiaozhi_mcp_fallback.sent",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "touch.top.tap" || registry["last_touch_event"] != "touch.top.tap" {
		t.Fatalf("registry touch without mcp = %#v", registry)
	}
	runtimeEcho, ok := registry["runtime_echo"].(map[string]any)
	if !ok ||
		runtimeEcho["last_touch_reaction_status"] != "delivered" ||
		runtimeEcho["last_touch_reaction_transport"] != "xiaozhi_mcp_sequence" ||
		runtimeEcho["last_touch_reaction_fallback_reason"] != "official_ws_disconnected" ||
		runtimeEcho["last_touch_reaction_steps"] != "2" {
		t.Fatalf("registry runtime_echo without official relay = %#v, want Xiaozhi MCP fallback delivered", registry["runtime_echo"])
	}
}

func TestXiaozhiProductStateReactionsSendBoundedBodyMCP(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		XiaozhiProductStateReactions: true,
	})
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
		"trace_id":   "a21-trace-xiaozhi-product-state-reaction",
		"session_id": "a21-session-xiaozhi-product-state-reaction",
		"device_id":  "44:1b:f6:e2:6a:60",
		"features": map[string]any{
			"mcp": true,
			"aec": true,
		},
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	a21, ok := reply["a21"].(map[string]any)
	if !ok || a21["profile"] != "product" || a21["state_reactions"] != true {
		t.Fatalf("a21 hello extension = %#v, want product state reactions", reply["a21"])
	}

	server.writeXiaozhiOfficialStackChanState(ctx, &xiaozhiSession{
		deviceID:  "44:1b:f6:e2:6a:60",
		traceID:   "a21-trace-xiaozhi-product-state-reaction",
		sessionID: "a21-session-xiaozhi-product-state-reaction",
		features:  xiaozhitransport.HelloFeatures{MCP: true, AEC: true},
	}, "thinking", "voice_pipeline_start", false)

	readMCP := func() (string, map[string]any) {
		message := readXiaozhiJSON(t, ctx, conn)
		if message["type"] != "mcp" || message["trace_id"] != "a21-trace-xiaozhi-product-state-reaction" || message["device_id"] != "44:1b:f6:e2:6a:60" {
			t.Fatalf("state reaction mcp wrapper = %#v", message)
		}
		payload, ok := message["payload"].(map[string]any)
		if !ok {
			t.Fatalf("state reaction payload = %#v", message["payload"])
		}
		params, ok := payload["params"].(map[string]any)
		if !ok {
			t.Fatalf("state reaction params = %#v", payload["params"])
		}
		args, ok := params["arguments"].(map[string]any)
		if !ok {
			t.Fatalf("state reaction args = %#v", params["arguments"])
		}
		return fmt.Sprint(params["name"]), args
	}

	tool, args := readMCP()
	if tool != xiaozhiMCPRobotSetLEDColorToolName ||
		args["red"] != float64(90) ||
		args["green"] != float64(0) ||
		args["blue"] != float64(168) {
		t.Fatalf("led state reaction = tool:%s args:%#v", tool, args)
	}
	tool, args = readMCP()
	if tool != xiaozhiMCPRobotSetHeadAnglesToolName ||
		args["yaw"] != float64(0) ||
		args["pitch"] != float64(36) ||
		args["speed"] != float64(160) {
		t.Fatalf("head state reaction = tool:%s args:%#v", tool, args)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-product-state-reaction")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"stackchan.display_state.registry_updated",
		"xiaozhi.state_reaction.robot_led_color.sent",
		"xiaozhi.state_reaction.robot_head_angles_set.sent",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["display_state"] != "thinking" || registry["display_state_source"] != "xiaozhi" {
		t.Fatalf("registry display state = %#v, want xiaozhi/thinking", registry)
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok || capabilities["xiaozhi_product_state_reactions"] != "true" {
		t.Fatalf("registry capabilities = %#v, want product state reactions", registry["capabilities"])
	}
	runtimeEcho, ok := registry["runtime_echo"].(map[string]any)
	if !ok ||
		runtimeEcho["last_state_reaction_status"] != "delivered" ||
		runtimeEcho["last_state_reaction_state"] != "thinking" ||
		runtimeEcho["last_state_reaction_reason"] != "voice_pipeline_start" ||
		runtimeEcho["robot_head_pitch"] != "36" ||
		runtimeEcho["robot_led_blue"] != "168" {
		t.Fatalf("registry runtime_echo = %#v, want state reaction echo", registry["runtime_echo"])
	}
}

func TestXiaozhiProductStateReactionsSuppressListenStartMCP(t *testing.T) {
	httpServer := httptest.NewServer(NewServerWithOptions(ServerOptions{
		XiaozhiProductStateReactions: true,
	}).Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-product-state-listen-start",
		"session_id": "a21-session-xiaozhi-product-state-listen-start",
		"device_id":  "44:1b:f6:e2:6a:60",
		"features": map[string]any{
			"mcp": true,
			"aec": true,
		},
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	a21, ok := reply["a21"].(map[string]any)
	if !ok || a21["profile"] != "product" || a21["state_reactions"] != true {
		t.Fatalf("a21 hello extension = %#v, want product state reactions", reply["a21"])
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":  "listen",
		"state": "start",
	}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-product-state-listen-start")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "xiaozhi.state_reaction.listen_start_suppressed") {
		t.Fatalf("trace missing listen-start suppression: %+v", traces.Events)
	}
	if traceContains(traces.Events, "xiaozhi.state_reaction.robot_led_color.sent") {
		t.Fatalf("listen-start state reaction sent MCP unexpectedly: %+v", traces.Events)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	runtimeEcho, ok := registry["runtime_echo"].(map[string]any)
	if !ok ||
		runtimeEcho["last_state_reaction_status"] != "suppressed_listen_start" ||
		runtimeEcho["last_state_reaction_state"] != "listening" ||
		runtimeEcho["last_state_reaction_reason"] != "listen_start" {
		t.Fatalf("registry runtime_echo = %#v, want listen-start suppression", registry["runtime_echo"])
	}
}

func TestXiaozhiProductStateReactionsRequireMCP(t *testing.T) {
	httpServer := httptest.NewServer(NewServerWithOptions(ServerOptions{
		XiaozhiProductStateReactions: true,
	}).Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-product-state-no-mcp",
		"session_id": "a21-session-xiaozhi-product-state-no-mcp",
		"device_id":  "44:1b:f6:e2:6a:60",
		"features": map[string]any{
			"aec": true,
		},
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	if _, ok := reply["a21"]; ok {
		t.Fatalf("a21 hello extension without mcp = %#v, want omitted", reply["a21"])
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":  "listen",
		"state": "start",
	}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)
	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["display_state"] != "listening" {
		t.Fatalf("registry display state without mcp = %#v, want listening", registry)
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if ok {
		if _, ok := capabilities["xiaozhi_product_state_reactions"]; ok {
			t.Fatalf("state reactions capability present without mcp: %#v", capabilities)
		}
	}
	if runtimeEcho, ok := registry["runtime_echo"].(map[string]any); ok {
		if _, ok := runtimeEcho["last_state_reaction_status"]; ok {
			t.Fatalf("state reaction echo present without mcp: %#v", runtimeEcho)
		}
	}
}

func TestXiaozhiDeviceRegistryMarksSocketDisconnectedOnClose(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-disconnect",
		"session_id": "a21-session-xiaozhi-disconnect",
		"device_id":  "44:1b:f6:e2:6a:60",
		"features": map[string]any{
			"mcp": true,
			"aec": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Close(websocket.StatusNormalClosure, "test disconnect"); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(time.Second)
	var registry map[string]any
	for time.Now().Before(deadline) {
		registry = fetchSingleDeviceRegistryItem(t, httpServer.URL)
		if registry["connection_status"] == "xiaozhi_ws_disconnected" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if registry["connection_status"] != "xiaozhi_ws_disconnected" ||
		registry["last_trace_id"] != "a21-trace-xiaozhi-disconnect" ||
		registry["last_session_id"] != "a21-session-xiaozhi-disconnect" {
		t.Fatalf("registry after disconnect = %#v, want xiaozhi_ws_disconnected", registry)
	}
}

func TestXiaozhiDeviceRegistryRestoresOnlineAfterReconnect(t *testing.T) {
	httpServer := httptest.NewServer(NewServerWithOptions(ServerOptions{XiaozhiProductPlaybackEvents: true}).Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-reconnect-1",
		"session_id": "a21-session-xiaozhi-reconnect-1",
		"device_id":  "44:1b:f6:e2:6a:60",
		"features": map[string]any{
			"mcp":              true,
			"aec":              true,
			"keepalive_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Close(websocket.StatusNormalClosure, "test reconnect"); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
		if registry["connection_status"] == "xiaozhi_ws_disconnected" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	reconn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reconn.Close(websocket.StatusNormalClosure, "test done") })
	writeXiaozhiHello(t, ctx, reconn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-reconnect-2",
		"session_id": "a21-session-xiaozhi-reconnect-2",
		"device_id":  "44:1b:f6:e2:6a:60",
		"features": map[string]any{
			"mcp":              true,
			"aec":              true,
			"keepalive_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, reconn)
	if err := wsjson.Write(ctx, reconn, map[string]any{
		"type":       "device",
		"kind":       "heartbeat",
		"trace_id":   "a21-trace-xiaozhi-reconnect-2",
		"session_id": "a21-session-xiaozhi-reconnect-2",
		"device_id":  "44:1b:f6:e2:6a:60",
	}); err != nil {
		t.Fatal(err)
	}

	var registry map[string]any
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		registry = fetchSingleDeviceRegistryItem(t, httpServer.URL)
		if registry["last_event"] == "device.heartbeat" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if registry["connection_status"] != "online" ||
		registry["last_event"] != "device.heartbeat" ||
		registry["last_trace_id"] != "a21-trace-xiaozhi-reconnect-2" ||
		registry["last_session_id"] != "a21-session-xiaozhi-reconnect-2" {
		t.Fatalf("registry after reconnect = %#v, want online heartbeat", registry)
	}
}
