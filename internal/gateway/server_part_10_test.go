package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestXiaozhiSessionTurnCancelInvalidatesCurrentTurnWithoutBlockingPacer(t *testing.T) {
	session := &xiaozhiSession{}
	turn := session.startXiaozhiTurn(context.Background(), protocol.ModeWorkmate)
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
	if turn.pacer.SentFrames() != 1 {
		t.Fatalf("sent frames = %d, want cancelled turn pacer left untouched", turn.pacer.SentFrames())
	}

	next := session.startXiaozhiTurn(context.Background(), protocol.ModeWorkmate)
	if next.id != 2 {
		t.Fatalf("next turn id = %d, want 2", next.id)
	}
	if session.shouldAbortXiaozhiTurn(next) {
		t.Fatal("new current turn should not abort")
	}
}

func TestXiaozhiSessionRecentDownlinkCanBeInterruptedAfterTurnCompletion(t *testing.T) {
	session := &xiaozhiSession{
		traceID:            "a21-trace-xiaozhi-recent-downlink",
		sessionID:          "a21-session-xiaozhi-recent-downlink",
		deviceID:           "stackchan-001",
		lastDownlinkAtMS:   1000,
		lastDownlinkTurnID: "a21-xiaozhi-turn-000007",
	}

	task, ok := session.prepareXiaozhiListenStartBargeIn("barge_in", 2500, xiaozhiPlaybackInterruptWindowMS)
	if !ok {
		t.Fatal("recent downlink should allow a listen/start playback stop")
	}
	if task.turn != nil {
		t.Fatalf("recent playback task turn = %+v, want no active turn", task.turn)
	}
	if task.turnID != "a21-xiaozhi-turn-000007" || task.traceID != session.traceID || task.sessionID != session.sessionID || task.deviceID != session.deviceID {
		t.Fatalf("recent playback task = %+v", task)
	}

	if _, ok := session.prepareXiaozhiListenStartBargeIn("barge_in", 5000, xiaozhiPlaybackInterruptWindowMS); ok {
		t.Fatal("stale downlink should not be interrupted as playback")
	}
}

func TestXiaozhiSessionRecentDownlinkAfterPlaybackStopDoneIsNotBargeIn(t *testing.T) {
	session := &xiaozhiSession{
		traceID:                  "a21-trace-xiaozhi-playback-drained",
		sessionID:                "a21-session-xiaozhi-playback-drained",
		deviceID:                 "stackchan-001",
		lastDownlinkAtMS:         1000,
		lastDownlinkTurnID:       "a21-xiaozhi-turn-000007",
		lastPlaybackStopDoneAtMS: 1400,
	}

	if _, ok := session.prepareXiaozhiListenStartBargeIn("barge_in", 1500, xiaozhiPlaybackInterruptWindowMS); ok {
		t.Fatal("recent downlink with later playback stop_done should not be mislabeled as barge-in")
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
	startAck := readXiaozhiJSON(t, ctx, conn)
	firstTurnID, ok := startAck["turn_id"].(string)
	if !ok || firstTurnID == "" {
		t.Fatalf("listen/start ack turn_id = %#v, want stable turn id", startAck["turn_id"])
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["turn_id"] != firstTurnID {
		t.Fatalf("abort stop turn_id = %#v, want %q", stop["turn_id"], firstTurnID)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	nextStartAck := readXiaozhiJSON(t, ctx, conn)
	nextTurnID, ok := nextStartAck["turn_id"].(string)
	if !ok || nextTurnID == "" || nextTurnID == firstTurnID {
		t.Fatalf("new listen turn_id = %#v, want new non-empty id different from %q", nextStartAck["turn_id"], firstTurnID)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestOpusPacket(t)); err != nil {
		t.Fatal(err)
	}

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
	for _, want := range []string{"xiaozhi.turn.start", "xiaozhi.turn.cancel", "turn_cancelled", "downlink_queue_cleared", "barge_in_detected"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}

func TestXiaozhiListenReplySuppressedForStockPhysicalMACDevice(t *testing.T) {
	if (&xiaozhiSession{deviceID: "44:1b:f6:e2:6a:60"}).shouldSendXiaozhiListenReply() {
		t.Fatal("stock physical MAC device should not receive server listen replies")
	}
	if !(&xiaozhiSession{deviceID: "44:1b:f6:e2:6a:60", features: xiaozhitransport.HelloFeatures{DebugMetrics: true}}).shouldSendXiaozhiListenReply() {
		t.Fatal("debug profile should keep listen replies for diagnostics")
	}
	if !(&xiaozhiSession{deviceID: "stackchan-001"}).shouldSendXiaozhiListenReply() {
		t.Fatal("virtual/test device should keep listen replies")
	}
}

func TestXiaozhiSpeakerVolumeUsesStockMCPToolCall(t *testing.T) {
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
		"trace_id":   "a21-trace-volume-mcp",
		"session_id": "a21-session-volume-mcp",
	})
	readXiaozhiJSON(t, ctx, conn)

	resp, err := http.Post(httpServer.URL+"/v1/xiaozhi/speaker-volume", "application/json", bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","volume":100,"trace_id":"a21-trace-volume-mcp","session_id":"a21-session-volume-mcp"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("volume status = %d: %s", resp.StatusCode, string(body))
	}

	var response XiaozhiSpeakerVolumeResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.DeliveredTransport != "xiaozhi_mcp" || response.ToolName != xiaozhiSpeakerVolumeToolName || response.Volume != 100 {
		t.Fatalf("volume response = %+v", response)
	}

	message := readXiaozhiJSON(t, ctx, conn)
	if message["type"] != "mcp" {
		t.Fatalf("mcp message type = %#v in %#v", message["type"], message)
	}
	payload, ok := message["payload"].(map[string]any)
	if !ok {
		t.Fatalf("mcp payload = %#v", message["payload"])
	}
	params, ok := payload["params"].(map[string]any)
	if !ok {
		t.Fatalf("mcp params = %#v", payload["params"])
	}
	if params["name"] != xiaozhiSpeakerVolumeToolName {
		t.Fatalf("mcp tool name = %#v", params["name"])
	}
	args, ok := params["arguments"].(map[string]any)
	if !ok {
		t.Fatalf("mcp arguments = %#v", params["arguments"])
	}
	if args["volume"] != float64(100) {
		t.Fatalf("mcp volume = %#v", args["volume"])
	}
}

func TestXiaozhiMCPStatusParityAllowsOnlyScopedTools(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		toolName string
		marker   string
		args     map[string]any
	}{
		{
			name:     "device status",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.get_device_status","trace_id":"a21-trace-mcp-device-status","session_id":"a21-session-mcp-device-status"}`,
			toolName: xiaozhiMCPGetDeviceStatusToolName,
			marker:   "xiaozhi.mcp.device_status.sent",
		},
		{
			name:     "speaker volume",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.audio_speaker.set_volume","volume":88,"trace_id":"a21-trace-mcp-speaker-volume","session_id":"a21-session-mcp-speaker-volume"}`,
			toolName: xiaozhiSpeakerVolumeToolName,
			marker:   "xiaozhi.mcp.speaker_volume.sent",
			args:     map[string]any{"volume": float64(88)},
		},
		{
			name:     "screen brightness",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_brightness","brightness":72,"trace_id":"a21-trace-mcp-brightness","session_id":"a21-session-mcp-brightness"}`,
			toolName: xiaozhiMCPScreenSetBrightnessToolName,
			marker:   "xiaozhi.mcp.screen_brightness.sent",
			args:     map[string]any{"brightness": float64(72)},
		},
		{
			name:     "screen theme",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_theme","theme":"dark","trace_id":"a21-trace-mcp-theme","session_id":"a21-session-mcp-theme"}`,
			toolName: xiaozhiMCPScreenSetThemeToolName,
			marker:   "xiaozhi.mcp.screen_theme.sent",
			args:     map[string]any{"theme": "dark"},
		},
		{
			name:     "screen info",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.get_info","trace_id":"a21-trace-mcp-info","session_id":"a21-session-mcp-info"}`,
			toolName: xiaozhiMCPScreenGetInfoToolName,
			marker:   "xiaozhi.mcp.screen_info.sent",
		},
		{
			name:     "robot head angles",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.get_head_angles","trace_id":"a21-trace-mcp-robot-head-read","session_id":"a21-session-mcp-robot-head-read"}`,
			toolName: xiaozhiMCPRobotGetHeadAnglesToolName,
			marker:   "xiaozhi.mcp.robot_head_angles.sent",
		},
		{
			name:     "robot set head angles",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_head_angles","yaw":15,"pitch":25,"speed":150,"trace_id":"a21-trace-mcp-robot-head-set","session_id":"a21-session-mcp-robot-head-set"}`,
			toolName: xiaozhiMCPRobotSetHeadAnglesToolName,
			marker:   "xiaozhi.mcp.robot_head_angles_set.sent",
			args:     map[string]any{"yaw": float64(15), "pitch": float64(25), "speed": float64(150)},
		},
		{
			name:     "robot set led color",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_led_color","red":0,"green":80,"blue":168,"trace_id":"a21-trace-mcp-robot-led","session_id":"a21-session-mcp-robot-led"}`,
			toolName: xiaozhiMCPRobotSetLEDColorToolName,
			marker:   "xiaozhi.mcp.robot_led_color.sent",
			args:     map[string]any{"red": float64(0), "green": float64(80), "blue": float64(168)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
				"device_id": "44:1b:f6:e2:6a:60",
			})
			readXiaozhiJSON(t, ctx, conn)

			resp, err := http.Post(httpServer.URL+"/v1/xiaozhi/mcp-control", "application/json", bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("mcp control status = %d: %s", resp.StatusCode, string(body))
			}

			var response XiaozhiMCPControlResponse
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}
			if response.DeliveredTransport != "xiaozhi_mcp" || response.ToolName != tc.toolName {
				t.Fatalf("mcp control response = %+v", response)
			}
			if response.TraceID == "" || response.SessionID == "" || response.DeviceID != "44:1b:f6:e2:6a:60" {
				t.Fatalf("mcp control identity = %+v", response)
			}

			message := readXiaozhiJSON(t, ctx, conn)
			if message["type"] != "mcp" || message["trace_id"] == "" || message["session_id"] == "" || message["device_id"] != "44:1b:f6:e2:6a:60" {
				t.Fatalf("mcp wrapper = %#v", message)
			}
			payload, ok := message["payload"].(map[string]any)
			if !ok {
				t.Fatalf("mcp payload = %#v", message["payload"])
			}
			params, ok := payload["params"].(map[string]any)
			if !ok {
				t.Fatalf("mcp params = %#v", payload["params"])
			}
			if params["name"] != tc.toolName {
				t.Fatalf("mcp tool name = %#v", params["name"])
			}
			args, ok := params["arguments"].(map[string]any)
			if len(tc.args) == 0 {
				if ok && len(args) != 0 {
					t.Fatalf("mcp args = %#v, want none", args)
				}
			} else {
				if !ok {
					t.Fatalf("mcp args = %#v, want %#v", params["arguments"], tc.args)
				}
				if len(args) != len(tc.args) {
					t.Fatalf("mcp args = %#v, want %#v", args, tc.args)
				}
				for key, want := range tc.args {
					if args[key] != want {
						t.Fatalf("mcp args = %#v, want %s=%#v", args, key, want)
					}
				}
			}

			traceURL := httpServer.URL + "/v1/traces?trace_id=" + response.TraceID
			traceResp, err := http.Get(traceURL)
			if err != nil {
				t.Fatal(err)
			}
			defer traceResp.Body.Close()
			var traces TraceResponse
			if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
				t.Fatal(err)
			}
			if !traceContains(traces.Events, tc.marker) {
				t.Fatalf("trace missing %q: %+v", tc.marker, traces.Events)
			}
			traceBody, _ := json.Marshal(traces)
			for _, forbidden := range []string{"raw_result", "provider_output", "transcript", "data_base64", "full_url", "local_path"} {
				if strings.Contains(string(traceBody), forbidden) {
					t.Fatalf("trace leaked forbidden token %q: %s", forbidden, string(traceBody))
				}
			}
		})
	}
}

func TestXiaozhiNamedMCPStatusEndpointsUseScopedTools(t *testing.T) {
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
		"device_id": "44:1b:f6:e2:6a:60",
	})
	readXiaozhiJSON(t, ctx, conn)

	tests := []struct {
		name     string
		path     string
		body     string
		toolName string
		argKey   string
		argValue any
		marker   string
	}{
		{
			name:     "device status",
			path:     "/v1/xiaozhi/device-status",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","trace_id":"a21-trace-named-device-status","session_id":"a21-session-named-device-status"}`,
			toolName: xiaozhiMCPGetDeviceStatusToolName,
			marker:   "xiaozhi.mcp.device_status.sent",
		},
		{
			name:     "screen brightness",
			path:     "/v1/xiaozhi/screen-brightness",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","brightness":64,"trace_id":"a21-trace-named-brightness","session_id":"a21-session-named-brightness"}`,
			toolName: xiaozhiMCPScreenSetBrightnessToolName,
			argKey:   "brightness",
			argValue: float64(64),
			marker:   "xiaozhi.mcp.screen_brightness.sent",
		},
		{
			name:     "screen theme",
			path:     "/v1/xiaozhi/screen-theme",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","theme":"dark","trace_id":"a21-trace-named-theme","session_id":"a21-session-named-theme"}`,
			toolName: xiaozhiMCPScreenSetThemeToolName,
			argKey:   "theme",
			argValue: "dark",
			marker:   "xiaozhi.mcp.screen_theme.sent",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(httpServer.URL+tc.path, "application/json", bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("%s status = %d: %s", tc.path, resp.StatusCode, string(body))
			}
			var response XiaozhiMCPControlResponse
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}
			if response.ToolName != tc.toolName || !response.ResultRedacted || response.DeliveredTransport != "xiaozhi_mcp" {
				t.Fatalf("named response = %+v", response)
			}

			message := readXiaozhiJSON(t, ctx, conn)
			payload := message["payload"].(map[string]any)
			params := payload["params"].(map[string]any)
			if params["name"] != tc.toolName {
				t.Fatalf("tool name = %#v, want %q", params["name"], tc.toolName)
			}
			args, _ := params["arguments"].(map[string]any)
			if tc.argKey == "" {
				if len(args) != 0 {
					t.Fatalf("args = %#v, want none", args)
				}
			} else if args[tc.argKey] != tc.argValue {
				t.Fatalf("args = %#v, want %s=%#v", args, tc.argKey, tc.argValue)
			}

			traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=" + response.TraceID)
			if err != nil {
				t.Fatal(err)
			}
			defer traceResp.Body.Close()
			var traces TraceResponse
			if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
				t.Fatal(err)
			}
			if !traceContains(traces.Events, tc.marker) {
				t.Fatalf("trace missing %q: %+v", tc.marker, traces.Events)
			}
		})
	}
}

func TestXiaozhiBodyPresetSendsBoundedMCPSequence(t *testing.T) {
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
		"trace_id":   "a21-trace-body-preset-hello",
		"session_id": "a21-session-body-preset-hello",
	})
	readXiaozhiJSON(t, ctx, conn)

	resp, err := http.Post(
		httpServer.URL+"/v1/xiaozhi/body-preset",
		"application/json",
		bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","preset":"celebrate","trace_id":"a21-trace-body-preset","session_id":"a21-session-body-preset"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("body preset status = %d: %s", resp.StatusCode, string(body))
	}
	var response XiaozhiBodyPresetResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != "a21.gateway.xiaozhi_body_preset.v1" ||
		response.Status != "delivered" ||
		response.DeliveredTransport != "xiaozhi_mcp_sequence" ||
		response.Preset != "celebrate" ||
		!response.ResultRedacted ||
		response.PhysicalAccepted ||
		len(response.Steps) != 2 {
		t.Fatalf("body preset response = %+v", response)
	}
	if response.Steps[0].ToolName != xiaozhiMCPRobotSetLEDColorToolName || response.Steps[0].Marker != "robot_led_color" {
		t.Fatalf("first step = %+v", response.Steps[0])
	}
	if response.Steps[1].ToolName != xiaozhiMCPRobotSetHeadAnglesToolName || response.Steps[1].Marker != "robot_head_angles_set" {
		t.Fatalf("second step = %+v", response.Steps[1])
	}

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

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-body-preset")
	if err != nil {
		t.Fatal(err)
	}
	defer traceResp.Body.Close()
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"xiaozhi.body_preset.celebrate.robot_led_color.sent",
		"xiaozhi.body_preset.celebrate.robot_head_angles_set.sent",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities = %#v", registry["capabilities"])
	}
	for key, want := range map[string]any{
		"last_body_preset":        "celebrate",
		"last_body_preset_status": "delivered",
		"robot_led_green":         "168",
		"robot_head_yaw":          "45",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestXiaozhiBodyPresetRejectsUnknownPreset(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/xiaozhi/body-preset", bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","preset":"camera"}`))
	rec := httptest.NewRecorder()
	NewServer().Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("body preset status = %d, want 400", rec.Code)
	}
}

func TestXiaozhiBodyMotionSendsBoundedMCPSequence(t *testing.T) {
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
		"trace_id":   "a21-trace-body-motion-hello",
		"session_id": "a21-session-body-motion-hello",
	})
	readXiaozhiJSON(t, ctx, conn)

	resp, err := http.Post(
		httpServer.URL+"/v1/xiaozhi/body-motion",
		"application/json",
		bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","motion":"dance","trace_id":"a21-trace-body-motion","session_id":"a21-session-body-motion"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("body motion status = %d: %s", resp.StatusCode, string(body))
	}
	var response XiaozhiBodyMotionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != "a21.gateway.xiaozhi_body_motion.v1" ||
		response.Status != "delivered" ||
		response.DeliveredTransport != "xiaozhi_mcp_sequence" ||
		response.Motion != "dance" ||
		!response.ResultRedacted ||
		response.PhysicalAccepted ||
		len(response.Steps) != 5 {
		t.Fatalf("body motion response = %+v", response)
	}

	expected := []struct {
		tool string
		args map[string]any
	}{
		{tool: xiaozhiMCPRobotSetLEDColorToolName, args: map[string]any{"red": float64(168), "green": float64(80), "blue": float64(0)}},
		{tool: xiaozhiMCPRobotSetHeadAnglesToolName, args: map[string]any{"yaw": float64(-55), "pitch": float64(38), "speed": float64(820)}},
		{tool: xiaozhiMCPRobotSetLEDColorToolName, args: map[string]any{"red": float64(0), "green": float64(168), "blue": float64(80)}},
		{tool: xiaozhiMCPRobotSetHeadAnglesToolName, args: map[string]any{"yaw": float64(55), "pitch": float64(70), "speed": float64(820)}},
		{tool: xiaozhiMCPRobotSetHeadAnglesToolName, args: map[string]any{"yaw": float64(0), "pitch": float64(45), "speed": float64(620)}},
	}
	for _, want := range expected {
		message := readXiaozhiJSON(t, ctx, conn)
		assertXiaozhiMCPMessage(t, message, want.tool, want.args)
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-body-motion")
	if err != nil {
		t.Fatal(err)
	}
	defer traceResp.Body.Close()
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"xiaozhi.body_motion.dance.step1.robot_led_color.sent",
		"xiaozhi.body_motion.dance.step2.robot_head_angles_set.sent",
		"xiaozhi.body_motion.dance.step5.robot_head_angles_set.sent",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities = %#v", registry["capabilities"])
	}
	for key, want := range map[string]any{
		"last_body_motion":        "dance",
		"last_body_motion_status": "delivered",
		"last_body_motion_step":   "5",
		"robot_head_yaw":          "0",
		"robot_head_pitch":        "45",
		"robot_led_green":         "168",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestXiaozhiBodyMotionRejectsUnknownMotion(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/xiaozhi/body-motion", bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","motion":"camera"}`))
	rec := httptest.NewRecorder()
	NewServer().Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("body motion status = %d, want 400", rec.Code)
	}
}

func TestXiaozhiBodySceneSendsScreenAndBodyMCPSequence(t *testing.T) {
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
		"trace_id":   "a21-trace-body-scene-hello",
		"session_id": "a21-session-body-scene-hello",
	})
	readXiaozhiJSON(t, ctx, conn)

	resp, err := http.Post(
		httpServer.URL+"/v1/xiaozhi/body-scene",
		"application/json",
		bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","scene":"showtime","trace_id":"a21-trace-body-scene","session_id":"a21-session-body-scene"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("body scene status = %d: %s", resp.StatusCode, string(body))
	}
	var response XiaozhiBodySceneResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != "a21.gateway.xiaozhi_body_scene.v1" ||
		response.Status != "delivered" ||
		response.DeliveredTransport != "xiaozhi_mcp_sequence" ||
		response.Scene != "showtime" ||
		!response.ResultRedacted ||
		response.PhysicalAccepted ||
		len(response.Steps) != 8 {
		t.Fatalf("body scene response = %+v", response)
	}

	expected := []struct {
		tool string
		args map[string]any
	}{
		{tool: xiaozhiMCPScreenSetThemeToolName, args: map[string]any{"theme": "dark"}},
		{tool: xiaozhiMCPScreenSetBrightnessToolName, args: map[string]any{"brightness": float64(72)}},
		{tool: xiaozhiMCPRobotSetLEDColorToolName, args: map[string]any{"red": float64(168), "green": float64(80), "blue": float64(0)}},
		{tool: xiaozhiMCPRobotSetHeadAnglesToolName, args: map[string]any{"yaw": float64(-18), "pitch": float64(36), "speed": float64(260)}},
		{tool: xiaozhiMCPRobotSetLEDColorToolName, args: map[string]any{"red": float64(0), "green": float64(168), "blue": float64(80)}},
		{tool: xiaozhiMCPRobotSetHeadAnglesToolName, args: map[string]any{"yaw": float64(18), "pitch": float64(36), "speed": float64(260)}},
		{tool: xiaozhiMCPRobotSetHeadAnglesToolName, args: map[string]any{"yaw": float64(0), "pitch": float64(24), "speed": float64(220)}},
		{tool: xiaozhiMCPRobotSetLEDColorToolName, args: map[string]any{"red": float64(0), "green": float64(36), "blue": float64(96)}},
	}
	for _, want := range expected {
		message := readXiaozhiJSON(t, ctx, conn)
		assertXiaozhiMCPMessage(t, message, want.tool, want.args)
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-body-scene")
	if err != nil {
		t.Fatal(err)
	}
	defer traceResp.Body.Close()
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"xiaozhi.body_scene.showtime.step1.screen_theme.sent",
		"xiaozhi.body_scene.showtime.step2.screen_brightness.sent",
		"xiaozhi.body_scene.showtime.step8.robot_led_color.sent",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities = %#v", registry["capabilities"])
	}
	for key, want := range map[string]any{
		"last_body_scene":        "showtime",
		"last_body_scene_status": "delivered",
		"last_body_scene_step":   "8",
		"screen_theme":           "dark",
		"screen_brightness":      "72",
		"robot_head_yaw":         "0",
		"robot_head_pitch":       "24",
		"robot_led_blue":         "96",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
}
