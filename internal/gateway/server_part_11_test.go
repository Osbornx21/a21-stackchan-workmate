package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"a21.local/a21/internal/providers"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestXiaozhiBodySceneFullCheckRunsOperatorVisibleSequence(t *testing.T) {
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
		"trace_id":   "a21-trace-body-scene-full-check-hello",
		"session_id": "a21-session-body-scene-full-check-hello",
	})
	readXiaozhiJSON(t, ctx, conn)

	resp, err := http.Post(
		httpServer.URL+"/v1/xiaozhi/body-scene",
		"application/json",
		bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","scene":"full_check","trace_id":"a21-trace-body-scene-full-check","session_id":"a21-session-body-scene-full-check"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("body scene full_check status = %d: %s", resp.StatusCode, string(body))
	}
	var response XiaozhiBodySceneResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Scene != "full_check" || response.Status != "delivered" || len(response.Steps) != 16 || response.PhysicalAccepted {
		t.Fatalf("body scene full_check response = %+v", response)
	}

	for i := 0; i < 16; i++ {
		readXiaozhiJSON(t, ctx, conn)
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-body-scene-full-check")
	if err != nil {
		t.Fatal(err)
	}
	defer traceResp.Body.Close()
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"xiaozhi.body_scene.full_check.step1.screen_theme.sent",
		"xiaozhi.body_scene.full_check.step8.robot_led_color.sent",
		"xiaozhi.body_scene.full_check.step16.robot_head_angles_set.sent",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities := registry["capabilities"].(map[string]any)
	for key, want := range map[string]any{
		"last_body_scene":        "full_check",
		"last_body_scene_status": "delivered",
		"last_body_scene_step":   "16",
		"screen_theme":           "auto",
		"screen_brightness":      "55",
		"robot_head_yaw":         "0",
		"robot_head_pitch":       "18",
		"robot_led_blue":         "32",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestXiaozhiBodySceneReportsAndAppliesStepPacing(t *testing.T) {
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
		"trace_id":   "a21-trace-body-scene-paced-hello",
		"session_id": "a21-session-body-scene-paced-hello",
	})
	readXiaozhiJSON(t, ctx, conn)

	resp, err := http.Post(
		httpServer.URL+"/v1/xiaozhi/body-scene",
		"application/json",
		bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","scene":"showtime","trace_id":"a21-trace-body-scene-paced","session_id":"a21-session-body-scene-paced"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	stepDelay, ok := response["step_delay_ms"].(float64)
	if !ok || stepDelay < 10 {
		t.Fatalf("step_delay_ms = %#v, want visible pacing", response["step_delay_ms"])
	}
	totalDelay, ok := response["total_planned_delay_ms"].(float64)
	if !ok || totalDelay < 70 {
		t.Fatalf("total_planned_delay_ms = %#v, want planned scene pacing", response["total_planned_delay_ms"])
	}
	for i := 0; i < 8; i++ {
		readXiaozhiJSON(t, ctx, conn)
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-body-scene-paced")
	if err != nil {
		t.Fatal(err)
	}
	defer traceResp.Body.Close()
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if traces.Summary.EventCount < 16 || traces.Summary.LastOffsetMS < int64(totalDelay)-20 {
		t.Fatalf("trace summary = %+v, want paced offsets near total planned delay %v", traces.Summary, totalDelay)
	}
}

func TestXiaozhiBodyScenePhysicalAcceptanceRecordsOperatorEvidence(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"device_id":  "44:1b:f6:e2:6a:60",
		"trace_id":   "a21-trace-body-scene-acceptance-hello",
		"session_id": "a21-session-body-scene-acceptance-hello",
	})
	readXiaozhiJSON(t, ctx, conn)

	sceneResp, err := http.Post(
		httpServer.URL+"/v1/xiaozhi/body-scene",
		"application/json",
		bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","scene":"full_check","trace_id":"a21-trace-body-scene-acceptance","session_id":"a21-session-body-scene-acceptance"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer sceneResp.Body.Close()
	if sceneResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(sceneResp.Body)
		t.Fatalf("body scene status = %d: %s", sceneResp.StatusCode, string(body))
	}
	for i := 0; i < 16; i++ {
		readXiaozhiJSON(t, ctx, conn)
	}

	acceptResp, err := http.Post(
		httpServer.URL+"/v1/xiaozhi/body-scene-acceptance",
		"application/json",
		bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","scene":"full_check","trace_id":"a21-trace-body-scene-acceptance","session_id":"a21-session-body-scene-acceptance","screen_visible":true,"rgb_visible":true,"servo_visible":true,"observer":"operator"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer acceptResp.Body.Close()
	if acceptResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(acceptResp.Body)
		t.Fatalf("body scene acceptance status = %d: %s", acceptResp.StatusCode, string(body))
	}
	var response map[string]any
	if err := json.NewDecoder(acceptResp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"schema_version":    "a21.gateway.xiaozhi_body_scene_acceptance.v1",
		"status":            "accepted",
		"scene":             "full_check",
		"physical_accepted": true,
		"result_redacted":   true,
	} {
		if response[key] != want {
			t.Fatalf("response[%s] = %#v, want %#v in %#v", key, response[key], want, response)
		}
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-body-scene-acceptance")
	if err != nil {
		t.Fatal(err)
	}
	defer traceResp.Body.Close()
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "xiaozhi.body_scene.full_check.physical_acceptance.accepted") {
		t.Fatalf("trace missing physical acceptance marker: %+v", traces.Events)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities := registry["capabilities"].(map[string]any)
	for key, want := range map[string]any{
		"body_scene_physical_accepted":        "true",
		"last_body_scene_acceptance_status":   "operator_visible_accepted",
		"last_body_scene_acceptance_scene":    "full_check",
		"body_scene_screen_physical_accepted": "true",
		"body_scene_rgb_physical_accepted":    "true",
		"body_scene_servo_physical_accepted":  "true",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestXiaozhiBodyScenePhysicalAcceptanceRequiresMatchingSceneEvidence(t *testing.T) {
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/xiaozhi/body-scene-acceptance", bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","scene":"full_check","trace_id":"a21-trace-missing-scene","session_id":"a21-session-missing-scene","screen_visible":true,"rgb_visible":true,"servo_visible":true,"observer":"operator"}`))
	NewServer().Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("body scene acceptance status = %d, want 409: %s", resp.Code, resp.Body.String())
	}
}

func TestXiaozhiBodySceneRejectsUnknownScene(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/xiaozhi/body-scene", bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","scene":"camera"}`))
	rec := httptest.NewRecorder()
	NewServer().Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("body scene status = %d, want 400", rec.Code)
	}
}

func TestXiaozhiNamedMCPEndpointsValidateAndRequireMCP(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		body   string
		status int
	}{
		{name: "device status missing device", path: "/v1/xiaozhi/device-status", body: `{}`, status: http.StatusBadRequest},
		{name: "brightness out of range", path: "/v1/xiaozhi/screen-brightness", body: `{"device_id":"44:1b:f6:e2:6a:60","brightness":101}`, status: http.StatusBadRequest},
		{name: "brightness missing", path: "/v1/xiaozhi/screen-brightness", body: `{"device_id":"44:1b:f6:e2:6a:60"}`, status: http.StatusBadRequest},
		{name: "theme invalid", path: "/v1/xiaozhi/screen-theme", body: `{"device_id":"44:1b:f6:e2:6a:60","theme":"../../theme"}`, status: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()
			NewServer().Handler().ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("%s status = %d, want %d: %s", tc.path, rec.Code, tc.status, rec.Body.String())
			}
		})
	}

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
		"features":  map[string]any{"aec": true},
	})
	readXiaozhiJSON(t, ctx, conn)
	resp, err := http.Post(httpServer.URL+"/v1/xiaozhi/device-status", "application/json", bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("non-mcp status = %d, want 409: %s", resp.StatusCode, string(body))
	}
	assertNoXiaozhiWebSocketMessage(t, conn, 120*time.Millisecond)
}

func TestXiaozhiMCPCapabilitiesEndpointReportsAllowedAndBlockedTools(t *testing.T) {
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

	resp, err := http.Get(httpServer.URL + "/v1/xiaozhi/mcp-capabilities?device_id=44:1b:f6:e2:6a:60")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("capabilities status = %d: %s", resp.StatusCode, string(body))
	}
	var response XiaozhiMCPCapabilitiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != "a21.gateway.xiaozhi_mcp_capabilities.v1" || !response.MCPAdvertised || !response.ResultRedacted || response.PhysicalAccepted {
		t.Fatalf("capabilities response = %+v", response)
	}
	if !containsString(response.AllowedTools, xiaozhiMCPScreenSetBrightnessToolName) || !containsString(response.AllowedTools, xiaozhiMCPRobotSetLEDColorToolName) {
		t.Fatalf("allowed tools = %#v", response.AllowedTools)
	}
	if !containsString(response.BlockedToolClasses, "nfc") ||
		!containsString(response.BlockedToolClasses, "firmware_upgrade") ||
		!containsString(response.BlockedToolClasses, "power_shutdown") {
		t.Fatalf("blocked tool classes = %#v", response.BlockedToolClasses)
	}
}

func TestXiaozhiMCPResponseFromDeviceIsAcceptedWithoutErrorReply(t *testing.T) {
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
		"trace_id":   "a21-trace-mcp-device-response",
		"session_id": "a21-session-mcp-device-response",
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type": "mcp",
		"payload": map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]any{
				"content": []any{},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiWebSocketMessage(t, conn, 120*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-mcp-device-response")
	if err != nil {
		t.Fatal(err)
	}
	defer traceResp.Body.Close()
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "xiaozhi.mcp.response.received") {
		t.Fatalf("trace missing mcp response marker: %+v", traces.Events)
	}
	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "xiaozhi.mcp.response.received" {
		t.Fatalf("last event = %#v, want mcp response received", registry["last_event"])
	}
}

func TestXiaozhiMCPStatusParityBlocksHighRiskTools(t *testing.T) {
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

	blockedTools := []string{
		"self.reboot",
		"self.upgrade_firmware",
		"self.camera.take_photo",
		"self.screen.snapshot",
		"self.camera.start_stream",
		"self.nfc.read",
		"self.infrared.send",
		"self.nfc.write",
		"self.infrared.receive",
		"self.app.launch",
	}
	for _, tool := range blockedTools {
		body := fmt.Sprintf(`{"device_id":"44:1b:f6:e2:6a:60","tool_name":%q,"trace_id":"a21-trace-blocked-mcp","session_id":"a21-session-blocked-mcp"}`, tool)
		resp, err := http.Post(httpServer.URL+"/v1/xiaozhi/mcp-control", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			t.Fatalf("%s status = %d: %s", tool, resp.StatusCode, string(body))
		}
		_ = resp.Body.Close()
	}
	assertNoXiaozhiWebSocketMessage(t, conn, 120*time.Millisecond)
}

func TestXiaozhiMCPStatusParityRequiresSafeArguments(t *testing.T) {
	tests := []string{
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.audio_speaker.set_volume","volume":101}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.audio_speaker.set_volume","volume":-1}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.audio_speaker.set_volume"}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.audio_speaker.set_volume","volume":88,"theme":"dark"}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.audio_speaker.set_volume","volume":88,"brightness":20}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_brightness","brightness":101}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_brightness"}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_brightness","brightness":72,"volume":88}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_theme","theme":"http://example.test/theme"}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_theme","theme":"../../theme"}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_theme","theme":"dark","volume":88}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.get_device_status","theme":"dark"}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.get_device_status","volume":88}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.get_device_status","yaw":10}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.get_info","brightness":10}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.get_info","volume":88}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.get_info","red":10,"green":10,"blue":10}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.get_head_angles","pitch":20}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_head_angles"}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_head_angles","yaw":-129}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_head_angles","yaw":129}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_head_angles","pitch":-1}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_head_angles","pitch":91}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_head_angles","yaw":10,"speed":99}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_head_angles","yaw":10,"speed":1001}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_head_angles","yaw":10,"red":10}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_led_color","red":1,"green":2}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_led_color","red":-1,"green":2,"blue":3}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_led_color","red":1,"green":169,"blue":3}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_led_color","red":1,"green":2,"blue":169}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.robot.set_led_color","red":1,"green":2,"blue":3,"pitch":20}`,
	}
	for _, body := range tests {
		req := httptest.NewRequest(http.MethodPost, "/v1/xiaozhi/mcp-control", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		NewServer().Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s status = %d, want 400", body, rec.Code)
		}
	}
}

func TestXiaozhiSayDeliversTextAsStockTTSDownlink(t *testing.T) {
	adapters := providers.VoicePipelineAdapters{
		ASR:        providers.NewMockASRAdapter("mock-local-asr"),
		TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
		TTS:        segmentChunkTTSAdapter{},
		Selection:  providers.VoicePipelineSelectionFromEnv(nil),
	}
	server := NewServerWithOptions(ServerOptions{XiaozhiVoicePipelineAdapters: &adapters})
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
		"trace_id":   "a21-trace-say-physical",
		"session_id": "a21-session-say-physical",
	})
	readXiaozhiJSON(t, ctx, conn)

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := (&http.Client{Timeout: 2 * time.Second}).Post(
			httpServer.URL+"/v1/xiaozhi/say",
			"application/json",
			bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","text":"这是一段实体小智长文本播放测试。","trace_id":"a21-trace-say-physical","session_id":"a21-session-say-physical"}`),
		)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	start := readXiaozhiJSON(t, ctx, conn)
	if start["type"] != "tts" || start["state"] != "start" || start["phase"] != "host_say" {
		t.Fatalf("say start = %#v", start)
	}
	sentence := readXiaozhiJSON(t, ctx, conn)
	if sentence["type"] != "tts" || sentence["state"] != "sentence_start" || sentence["phase"] != "host_say" {
		t.Fatalf("say sentence = %#v", sentence)
	}
	if packet := readXiaozhiBinary(t, ctx, conn); len(packet) == 0 {
		t.Fatal("say binary packet is empty")
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "host_say_complete" {
		t.Fatalf("say stop = %#v", stop)
	}

	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("say status = %d: %s", resp.StatusCode, string(body))
		}
		var response XiaozhiSayResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.DeliveredTransport != "xiaozhi_ws" || response.AudioChunks != 1 || response.TextChars == 0 {
			t.Fatalf("say response = %+v", response)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestXiaozhiSayReportsInterruptedWhenProductTouchBargeInCancelsDownlink(t *testing.T) {
	releaseSecondChunk := make(chan struct{})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() {
			close(releaseSecondChunk)
		})
	}
	t.Cleanup(release)

	adapters := providers.VoicePipelineAdapters{
		ASR:        providers.NewMockASRAdapter("mock-local-asr"),
		TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
		TTS:        gatedSecondChunkTTSAdapter{release: releaseSecondChunk},
		Selection:  providers.VoicePipelineSelectionFromEnv(nil),
	}
	server := NewServerWithOptions(ServerOptions{
		XiaozhiProductTouchEvents:    true,
		XiaozhiVoicePipelineAdapters: &adapters,
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
		"device_id":  "44:1b:f6:e2:6a:60",
		"trace_id":   "a21-trace-say-touch-interrupt",
		"session_id": "a21-session-say-touch-interrupt",
		"features": map[string]any{
			"mcp":          true,
			"aec":          true,
			"touch_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := (&http.Client{Timeout: 2 * time.Second}).Post(
			httpServer.URL+"/v1/xiaozhi/say",
			"application/json",
			bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","text":"这是一段会被用户触摸打断的长播放测试。","trace_id":"a21-trace-say-touch-interrupt","session_id":"a21-session-say-touch-interrupt"}`),
		)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	start := readXiaozhiJSON(t, ctx, conn)
	if start["type"] != "tts" || start["state"] != "start" || start["phase"] != "host_say" {
		t.Fatalf("say start = %#v", start)
	}
	sentence := readXiaozhiJSON(t, ctx, conn)
	if sentence["type"] != "tts" || sentence["state"] != "sentence_start" || sentence["phase"] != "host_say" {
		t.Fatalf("say sentence = %#v", sentence)
	}
	if packet := readXiaozhiBinary(t, ctx, conn); len(packet) == 0 {
		t.Fatal("say binary packet is empty")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "device",
		"kind":       "touch",
		"touch":      "top_barge_in",
		"source":     "top_sensor",
		"trace_id":   "a21-trace-say-touch-interrupt",
		"session_id": "a21-session-say-touch-interrupt",
		"device_id":  "44:1b:f6:e2:6a:60",
	}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "touch_barge_in" {
		t.Fatalf("touch barge stop = %#v", stop)
	}
	release()

	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("say interrupted status = %d: %s", resp.StatusCode, string(body))
		}
		var response XiaozhiSayResponse
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatal(err)
		}
		if response.Status != "interrupted" || response.DeliveredTransport != "xiaozhi_ws" || response.AudioChunks != 1 || response.InterruptReason != "touch_barge_in" {
			t.Fatalf("say interrupted response = %+v", response)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-say-touch-interrupt", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"device.touch.barge_in.received", "barge_in.detected", "playback.stop", "xiaozhi.say.interrupted"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	if traceContains(traces.Events, "xiaozhi.say.delivered") {
		t.Fatalf("trace marked interrupted say as delivered: %+v", traces.Events)
	}
}

func TestXiaozhiSayKeepsBadGatewayForActualDownlinkError(t *testing.T) {
	adapters := providers.VoicePipelineAdapters{
		ASR:        providers.NewMockASRAdapter("mock-local-asr"),
		TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
		TTS:        chunkThenErrorTTSAdapter{},
		Selection:  providers.VoicePipelineSelectionFromEnv(nil),
	}
	server := NewServerWithOptions(ServerOptions{XiaozhiVoicePipelineAdapters: &adapters})
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
		"trace_id":   "a21-trace-say-downlink-error",
		"session_id": "a21-session-say-downlink-error",
	})
	readXiaozhiJSON(t, ctx, conn)

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := (&http.Client{Timeout: 2 * time.Second}).Post(
			httpServer.URL+"/v1/xiaozhi/say",
			"application/json",
			bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","text":"这是一段真实下行错误测试。","trace_id":"a21-trace-say-downlink-error","session_id":"a21-session-say-downlink-error"}`),
		)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	if packet := readXiaozhiBinary(t, ctx, conn); len(packet) == 0 {
		t.Fatal("say binary packet is empty")
	}

	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadGateway {
			t.Fatalf("say downlink error status = %d, body %s", resp.StatusCode, string(body))
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-say-downlink-error", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "xiaozhi.say.downlink_error") {
		t.Fatalf("trace missing downlink error: %+v", traces.Events)
	}
	if traceContains(traces.Events, "xiaozhi.say.interrupted") {
		t.Fatalf("trace misclassified downlink error as interrupted: %+v", traces.Events)
	}
}
