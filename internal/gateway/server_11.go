package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/protocol"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
)

func (s *Server) handleXiaozhiBodyScene(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiBodySceneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	scene, plans, err := xiaozhiBodyScenePlans(req.DeviceID, req.Scene)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	steps := make([]XiaozhiBodyPresetStepResponse, 0, len(plans))
	stepDelay := s.bodySceneStepDelay
	for index, plan := range plans {
		if index > 0 && stepDelay > 0 {
			if err := sleepBodySceneStep(r.Context(), stepDelay); err != nil {
				s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.body_scene."+scene+".cancelled", s.now().UnixMilli())
				http.Error(w, "xiaozhi body scene delivery cancelled", http.StatusBadGateway)
				return
			}
		}
		plan.TraceID = traceID
		plan.SessionID = sessionID
		delivery, status, message := s.sendXiaozhiMCPControl(r.Context(), plan)
		if status != 0 {
			s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.body_scene."+scene+".failed", s.now().UnixMilli())
			http.Error(w, message, status)
			return
		}
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		sceneMarker := "xiaozhi.body_scene." + scene + ".step" + strconv.Itoa(index+1) + "." + delivery.Marker + ".sent"
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, s.now().UnixMilli())
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, sceneMarker, s.now().UnixMilli())
		activity := xiaozhiMCPActivity(delivery.Marker, delivery.Args)
		activity["last_body_scene"] = scene
		activity["last_body_scene_status"] = "delivered"
		activity["last_body_scene_step"] = strconv.Itoa(index + 1)
		activity["last_body_scene_tool"] = delivery.Marker
		activity["last_body_scene_trace_id"] = delivery.Response.TraceID
		activity["last_body_scene_session_id"] = delivery.Response.SessionID
		s.recordXiaozhiDeviceActivity(&xiaozhiSession{
			deviceID:  delivery.Response.DeviceID,
			traceID:   delivery.Response.TraceID,
			sessionID: delivery.Response.SessionID,
		}, sceneMarker, activity)
		steps = append(steps, XiaozhiBodyPresetStepResponse{
			ToolName:  delivery.Response.ToolName,
			MCPID:     delivery.Response.MCPID,
			Marker:    delivery.Marker,
			Arguments: delivery.Response.Arguments,
		})
	}
	writeJSON(w, http.StatusOK, XiaozhiBodySceneResponse{
		SchemaVersion:       "a21.gateway.xiaozhi_body_scene.v1",
		TraceID:             traceID,
		SessionID:           sessionID,
		DeviceID:            req.DeviceID,
		Status:              "delivered",
		DeliveredTransport:  "xiaozhi_mcp_sequence",
		Scene:               scene,
		Steps:               steps,
		StepDelayMS:         int64(stepDelay / time.Millisecond),
		TotalPlannedDelayMS: bodySceneTotalPlannedDelayMS(stepDelay, len(steps)),
		ResultRedacted:      true,
		PhysicalAccepted:    false,
	})
}

func normalizedBodySceneStepDelay(delay time.Duration) time.Duration {
	if delay < 0 {
		return 0
	}
	if delay == 0 {
		return defaultBodySceneStepDelay
	}
	if delay > maxBodySceneStepDelay {
		return maxBodySceneStepDelay
	}
	return delay
}

func sleepBodySceneStep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func bodySceneTotalPlannedDelayMS(delay time.Duration, stepCount int) int64 {
	if delay <= 0 || stepCount <= 1 {
		return 0
	}
	return int64(delay/time.Millisecond) * int64(stepCount-1)
}

func (s *Server) handleXiaozhiBodySceneAcceptance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiBodySceneAcceptanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	scene, _, err := xiaozhiBodyScenePlans(req.DeviceID, req.Scene)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if scene != "full_check" {
		http.Error(w, "body scene physical acceptance requires full_check", http.StatusBadRequest)
		return
	}
	traceID := strings.TrimSpace(req.TraceID)
	sessionID := strings.TrimSpace(req.SessionID)
	if traceID == "" || sessionID == "" {
		http.Error(w, "trace_id and session_id are required", http.StatusBadRequest)
		return
	}
	observer := strings.ToLower(strings.TrimSpace(req.Observer))
	if observer == "" {
		observer = "operator"
	}
	if observer != "operator" && observer != "instrument" {
		http.Error(w, "observer must be operator or instrument", http.StatusBadRequest)
		return
	}
	if !req.ScreenVisible || !req.RGBVisible || !req.ServoVisible {
		http.Error(w, "screen_visible, rgb_visible, and servo_visible must be true", http.StatusBadRequest)
		return
	}
	if !s.hasMatchingBodySceneEvidence(req.DeviceID, scene, traceID, sessionID) {
		http.Error(w, "matching body scene evidence is required before physical acceptance", http.StatusConflict)
		return
	}

	nowMS := s.now().UnixMilli()
	marker := "xiaozhi.body_scene." + scene + ".physical_acceptance.accepted"
	s.recordTrace(traceID, sessionID, req.DeviceID, marker, nowMS)
	s.recordBodyScenePhysicalAcceptance(req.DeviceID, scene, traceID, sessionID, observer, nowMS, marker)
	writeJSON(w, http.StatusOK, XiaozhiBodySceneAcceptanceResponse{
		SchemaVersion:     "a21.gateway.xiaozhi_body_scene_acceptance.v1",
		TraceID:           traceID,
		SessionID:         sessionID,
		DeviceID:          req.DeviceID,
		Status:            "accepted",
		Scene:             scene,
		Observer:          observer,
		AcceptedSurfaces:  []string{"screen", "rgb", "servo"},
		PhysicalAccepted:  true,
		ResultRedacted:    true,
		AcceptanceEventMS: nowMS,
	})
}

func (s *Server) hasMatchingBodySceneEvidence(deviceID string, scene string, traceID string, sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[deviceID]
	if record.DeviceID == "" || record.Capabilities == nil {
		return false
	}
	return record.Capabilities["last_body_scene"] == scene &&
		record.Capabilities["last_body_scene_status"] == "delivered" &&
		record.Capabilities["last_body_scene_trace_id"] == traceID &&
		record.Capabilities["last_body_scene_session_id"] == sessionID
}

func (s *Server) recordBodyScenePhysicalAcceptance(deviceID string, scene string, traceID string, sessionID string, observer string, atMS int64, event string) {
	capabilities := map[string]string{
		"body_scene_physical_accepted":          "true",
		"body_scene_screen_physical_accepted":   "true",
		"body_scene_rgb_physical_accepted":      "true",
		"body_scene_servo_physical_accepted":    "true",
		"last_body_scene_acceptance_status":     "operator_visible_accepted",
		"last_body_scene_acceptance_scene":      scene,
		"last_body_scene_acceptance_trace_id":   traceID,
		"last_body_scene_acceptance_session_id": sessionID,
		"last_body_scene_acceptance_observer":   observer,
		"last_body_scene_acceptance_at_ms":      strconv.FormatInt(atMS, 10),
	}
	if observer == "instrument" {
		capabilities["last_body_scene_acceptance_status"] = "instrument_visible_accepted"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[deviceID]
	if record.DeviceID == "" {
		record.DeviceID = deviceID
		record.FirstSeenMS = atMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.Capabilities = mergeDeviceCapabilities(record.Capabilities, capabilities)
	if cleanEvent := strings.TrimSpace(event); cleanEvent != "" {
		record.LastEvent = protocol.DeviceEventKind(cleanEvent)
	}
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = atMS
	s.devices[deviceID] = record
}

func (s *Server) recordPowerLifecyclePhysicalAcceptance(deviceID string, traceID string, sessionID string, observer string, bootSource string, pmicProfile string, powerKeyHoldMS int, bootObservedAtMS int64, atMS int64, event string) {
	capabilities := map[string]string{
		"power_lifecycle_physical_accepted":              "true",
		"no_cable_cold_boot_physical_accepted":           "true",
		"physical_power_button_start_physical_accepted":  "true",
		"gateway_reconnect_after_power_button_accepted":  "true",
		"xiaozhi_socket_after_power_button_accepted":     "true",
		"stackchan_pmic_power_key_profile":               pmicProfile,
		"last_power_lifecycle_boot_source":               bootSource,
		"last_power_lifecycle_usb_connected_during_boot": "false",
		"last_power_lifecycle_power_key_hold_ms":         strconv.Itoa(powerKeyHoldMS),
		"last_power_lifecycle_boot_observed_at_ms":       strconv.FormatInt(bootObservedAtMS, 10),
		"last_power_lifecycle_pmic_profile":              pmicProfile,
		"last_power_lifecycle_acceptance_status":         "operator_power_button_accepted",
		"last_power_lifecycle_acceptance_trace_id":       traceID,
		"last_power_lifecycle_acceptance_session_id":     sessionID,
		"last_power_lifecycle_acceptance_observer":       observer,
		"last_power_lifecycle_acceptance_at_ms":          strconv.FormatInt(atMS, 10),
	}
	if observer == "instrument" {
		capabilities["last_power_lifecycle_acceptance_status"] = "instrument_power_button_accepted"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[deviceID]
	if record.DeviceID == "" {
		record.DeviceID = deviceID
		record.FirstSeenMS = atMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.Capabilities = mergeDeviceCapabilities(record.Capabilities, capabilities)
	if cleanEvent := strings.TrimSpace(event); cleanEvent != "" {
		record.LastEvent = protocol.DeviceEventKind(cleanEvent)
	}
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = atMS
	s.devices[deviceID] = record
}

func (s *Server) handleXiaozhiMCPCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if !validA21DeviceID(deviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	socket, ok := s.xiaozhiSocket(deviceID)
	if !ok {
		http.Error(w, "xiaozhi websocket is not connected", http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, XiaozhiMCPCapabilitiesResponse{
		SchemaVersion:      "a21.gateway.xiaozhi_mcp_capabilities.v1",
		Service:            "a21-gateway",
		DeviceID:           deviceID,
		ConnectionStatus:   "online",
		MCPAdvertised:      socket.features.MCP,
		AllowedTools:       xiaozhiMCPAllowedTools(),
		BlockedToolClasses: xiaozhiMCPBlockedToolClasses(),
		ResultRedacted:     true,
		PhysicalAccepted:   false,
	})
}

func (s *Server) handleXiaozhiMCPControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiMCPControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	response, status, message := s.deliverXiaozhiMCPControl(r.Context(), req)
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) deliverXiaozhiMCPControl(ctx context.Context, req XiaozhiMCPControlRequest) (XiaozhiMCPControlResponse, int, string) {
	delivery, status, message := s.sendXiaozhiMCPControl(ctx, req)
	if status != 0 {
		return XiaozhiMCPControlResponse{}, status, message
	}
	traceMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
	s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, traceMarker, s.now().UnixMilli())
	s.recordXiaozhiDeviceActivity(&xiaozhiSession{
		deviceID:  delivery.Response.DeviceID,
		traceID:   delivery.Response.TraceID,
		sessionID: delivery.Response.SessionID,
	}, traceMarker, xiaozhiMCPActivity(delivery.Marker, delivery.Args))
	return delivery.Response, 0, ""
}

type xiaozhiMCPDelivery struct {
	Response XiaozhiMCPControlResponse
	Marker   string
	Args     map[string]any
}

func (s *Server) sendXiaozhiMCPControl(ctx context.Context, req XiaozhiMCPControlRequest) (xiaozhiMCPDelivery, int, string) {
	if !validA21DeviceID(req.DeviceID) {
		return xiaozhiMCPDelivery{}, http.StatusBadRequest, "valid device_id is required"
	}
	toolName, args, marker, err := xiaozhiMCPStatusParityCall(req)
	if err != nil {
		return xiaozhiMCPDelivery{}, http.StatusBadRequest, err.Error()
	}
	socket, ok := s.xiaozhiSocket(req.DeviceID)
	if !ok {
		return xiaozhiMCPDelivery{}, http.StatusConflict, "xiaozhi websocket is not connected"
	}
	if !socket.features.MCP {
		return xiaozhiMCPDelivery{}, http.StatusConflict, "xiaozhi mcp control requires stock MCP support"
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	mcpID := "a21-mcp-status-parity-" + safeGatewayFallbackToken(marker, "tool") + "-" + safeGatewayFallbackToken(traceID, "trace")
	payload, err := xiaozhitransport.BuildMCPToolsCallRequest(mcpID, toolName, args)
	if err != nil {
		return xiaozhiMCPDelivery{}, http.StatusBadRequest, "invalid xiaozhi mcp control request"
	}
	var payloadObject map[string]any
	if err := json.Unmarshal(payload, &payloadObject); err != nil {
		return xiaozhiMCPDelivery{}, http.StatusBadRequest, "invalid xiaozhi mcp control request"
	}
	message, err := json.Marshal(map[string]any{
		"type":       "mcp",
		"payload":    payloadObject,
		"trace_id":   traceID,
		"session_id": sessionID,
		"device_id":  req.DeviceID,
	})
	if err != nil {
		return xiaozhiMCPDelivery{}, http.StatusBadRequest, "invalid xiaozhi mcp control message"
	}
	socket.writeMu.Lock()
	err = socket.conn.Write(ctx, websocket.MessageText, message)
	socket.writeMu.Unlock()
	if err != nil {
		return xiaozhiMCPDelivery{}, http.StatusBadGateway, "xiaozhi mcp control delivery failed"
	}

	response := XiaozhiMCPControlResponse{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "xiaozhi_mcp",
		ToolName:           toolName,
		MCPID:              mcpID,
		Arguments:          args,
		ResultRedacted:     true,
	}
	return xiaozhiMCPDelivery{Response: response, Marker: marker, Args: args}, 0, ""
}

func xiaozhiMCPActivity(marker string, args map[string]any) map[string]string {
	activity := map[string]string{
		"xiaozhi_mcp_tool": marker,
	}
	if value, ok := args["brightness"]; ok {
		activity["screen_brightness"] = fmt.Sprint(value)
	}
	if value, ok := args["volume"]; ok {
		activity["speaker_volume"] = fmt.Sprint(value)
	}
	if value, ok := args["theme"]; ok {
		activity["screen_theme"] = fmt.Sprint(value)
	}
	if value, ok := args["yaw"]; ok {
		activity["robot_head_yaw"] = fmt.Sprint(value)
	}
	if value, ok := args["pitch"]; ok {
		activity["robot_head_pitch"] = fmt.Sprint(value)
	}
	if value, ok := args["speed"]; ok {
		activity["robot_head_speed"] = fmt.Sprint(value)
	}
	if value, ok := args["red"]; ok {
		activity["robot_led_red"] = fmt.Sprint(value)
	}
	if value, ok := args["green"]; ok {
		activity["robot_led_green"] = fmt.Sprint(value)
	}
	if value, ok := args["blue"]; ok {
		activity["robot_led_blue"] = fmt.Sprint(value)
	}
	return activity
}

func xiaozhiMCPStatusParityCall(req XiaozhiMCPControlRequest) (string, map[string]any, string, error) {
	toolName := strings.TrimSpace(req.ToolName)
	switch toolName {
	case xiaozhiSpeakerVolumeToolName:
		if req.Volume == nil {
			return "", nil, "", errors.New("volume is required")
		}
		if *req.Volume < 0 || *req.Volume > 100 {
			return "", nil, "", errors.New("volume must be 0..100")
		}
		if xiaozhiMCPHasAnyArgumentExcept(req, "volume") {
			return "", nil, "", errors.New("speaker volume accepts only volume")
		}
		return toolName, map[string]any{"volume": *req.Volume}, "speaker_volume", nil
	case xiaozhiMCPGetDeviceStatusToolName:
		if xiaozhiMCPHasAnyArgument(req) {
			return "", nil, "", errors.New("device status does not accept arguments")
		}
		return toolName, nil, "device_status", nil
	case xiaozhiMCPScreenSetBrightnessToolName:
		if req.Brightness == nil {
			return "", nil, "", errors.New("brightness is required")
		}
		if *req.Brightness < 0 || *req.Brightness > 100 {
			return "", nil, "", errors.New("brightness must be 0..100")
		}
		if xiaozhiMCPHasAnyArgumentExcept(req, "brightness") {
			return "", nil, "", errors.New("brightness control accepts only brightness")
		}
		return toolName, map[string]any{"brightness": *req.Brightness}, "screen_brightness", nil
	case xiaozhiMCPScreenSetThemeToolName:
		theme := strings.TrimSpace(req.Theme)
		if !validXiaozhiScreenTheme(theme) {
			return "", nil, "", errors.New("theme must be a safe 1..32 character token")
		}
		if xiaozhiMCPHasAnyArgumentExcept(req, "theme") {
			return "", nil, "", errors.New("theme control accepts only theme")
		}
		return toolName, map[string]any{"theme": theme}, "screen_theme", nil
	case xiaozhiMCPScreenGetInfoToolName:
		if xiaozhiMCPHasAnyArgument(req) {
			return "", nil, "", errors.New("screen info does not accept arguments")
		}
		return toolName, nil, "screen_info", nil
	case xiaozhiMCPRobotGetHeadAnglesToolName:
		if xiaozhiMCPHasAnyArgument(req) {
			return "", nil, "", errors.New("robot head angle read does not accept arguments")
		}
		return toolName, nil, "robot_head_angles", nil
	case xiaozhiMCPRobotSetHeadAnglesToolName:
		if req.Yaw == nil && req.Pitch == nil {
			return "", nil, "", errors.New("yaw or pitch is required")
		}
		if xiaozhiMCPHasAnyArgumentExcept(req, "yaw", "pitch", "speed") {
			return "", nil, "", errors.New("robot head control accepts only yaw, pitch, and speed")
		}
		args := map[string]any{}
		if req.Yaw != nil {
			if *req.Yaw < -128 || *req.Yaw > 128 {
				return "", nil, "", errors.New("yaw must be -128..128")
			}
			args["yaw"] = *req.Yaw
		}
		if req.Pitch != nil {
			if *req.Pitch < 0 || *req.Pitch > 90 {
				return "", nil, "", errors.New("pitch must be 0..90")
			}
			args["pitch"] = *req.Pitch
		}
		speed := 150
		if req.Speed != nil {
			if *req.Speed < 100 || *req.Speed > 1000 {
				return "", nil, "", errors.New("speed must be 100..1000")
			}
			speed = *req.Speed
		}
		args["speed"] = speed
		return toolName, args, "robot_head_angles_set", nil
	case xiaozhiMCPRobotSetLEDColorToolName:
		if req.Red == nil || req.Green == nil || req.Blue == nil {
			return "", nil, "", errors.New("red, green, and blue are required")
		}
		if xiaozhiMCPHasAnyArgumentExcept(req, "red", "green", "blue") {
			return "", nil, "", errors.New("robot led control accepts only red, green, and blue")
		}
		if *req.Red < 0 || *req.Red > 168 {
			return "", nil, "", errors.New("red must be 0..168")
		}
		if *req.Green < 0 || *req.Green > 168 {
			return "", nil, "", errors.New("green must be 0..168")
		}
		if *req.Blue < 0 || *req.Blue > 168 {
			return "", nil, "", errors.New("blue must be 0..168")
		}
		return toolName, map[string]any{"red": *req.Red, "green": *req.Green, "blue": *req.Blue}, "robot_led_color", nil
	default:
		return "", nil, "", errors.New("xiaozhi mcp tool is not allowed by A21 status parity")
	}
}

func xiaozhiBodyPresetPlans(deviceID string, preset string) (string, []XiaozhiMCPControlRequest, error) {
	preset = strings.ToLower(strings.TrimSpace(preset))
	if preset == "" {
		return "", nil, errors.New("body_preset is required")
	}
	base := XiaozhiMCPControlRequest{DeviceID: deviceID}
	led := func(red, green, blue int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetLEDColorToolName
		req.Red = xiaozhiPresetInt(red)
		req.Green = xiaozhiPresetInt(green)
		req.Blue = xiaozhiPresetInt(blue)
		return req
	}
	head := func(yaw, pitch, speed int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetHeadAnglesToolName
		req.Yaw = xiaozhiPresetInt(yaw)
		req.Pitch = xiaozhiPresetInt(pitch)
		req.Speed = xiaozhiPresetInt(speed)
		return req
	}
	switch preset {
	case "ready":
		return preset, []XiaozhiMCPControlRequest{
			led(0, 36, 96),
			head(0, 45, 420),
		}, nil
	case "listening":
		return preset, []XiaozhiMCPControlRequest{
			led(0, 120, 88),
			head(0, 62, 520),
		}, nil
	case "thinking":
		return preset, []XiaozhiMCPControlRequest{
			led(120, 72, 0),
			head(-28, 54, 480),
		}, nil
	case "speaking":
		return preset, []XiaozhiMCPControlRequest{
			led(20, 24, 168),
			head(24, 56, 520),
		}, nil
	case "celebrate":
		return preset, []XiaozhiMCPControlRequest{
			led(0, 168, 80),
			head(45, 68, 680),
		}, nil
	case "reset_idle":
		return preset, []XiaozhiMCPControlRequest{
			led(0, 0, 32),
			head(0, 45, 420),
		}, nil
	default:
		return "", nil, errors.New("body_preset must be ready, listening, thinking, speaking, celebrate, or reset_idle")
	}
}

func xiaozhiBodyMotionPlans(deviceID string, motion string) (string, []XiaozhiMCPControlRequest, error) {
	motion = strings.ToLower(strings.TrimSpace(motion))
	if motion == "" {
		return "", nil, errors.New("body_motion is required")
	}
	base := XiaozhiMCPControlRequest{DeviceID: deviceID}
	led := func(red, green, blue int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetLEDColorToolName
		req.Red = xiaozhiPresetInt(red)
		req.Green = xiaozhiPresetInt(green)
		req.Blue = xiaozhiPresetInt(blue)
		return req
	}
	head := func(yaw, pitch, speed int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetHeadAnglesToolName
		req.Yaw = xiaozhiPresetInt(yaw)
		req.Pitch = xiaozhiPresetInt(pitch)
		req.Speed = xiaozhiPresetInt(speed)
		return req
	}
	switch motion {
	case "look_up":
		return motion, []XiaozhiMCPControlRequest{
			led(0, 80, 168),
			head(0, 68, 620),
		}, nil
	case "nod":
		return motion, []XiaozhiMCPControlRequest{
			led(0, 120, 88),
			head(0, 32, 700),
			head(0, 72, 820),
			head(0, 45, 620),
		}, nil
	case "shake":
		return motion, []XiaozhiMCPControlRequest{
			led(120, 60, 0),
			head(-45, 45, 780),
			head(45, 45, 780),
			head(0, 45, 620),
		}, nil
	case "dance":
		return motion, []XiaozhiMCPControlRequest{
			led(168, 80, 0),
			head(-55, 38, 820),
			led(0, 168, 80),
			head(55, 70, 820),
			head(0, 45, 620),
		}, nil
	case "stop":
		return motion, []XiaozhiMCPControlRequest{
			led(0, 0, 32),
			head(0, 45, 620),
		}, nil
	default:
		return "", nil, errors.New("body_motion must be look_up, nod, shake, dance, or stop")
	}
}
