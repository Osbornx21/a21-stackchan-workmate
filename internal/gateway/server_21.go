package gateway

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	stackchantransport "a21.local/a21/internal/transport/stackchan"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
)

func officialStackChanStatusPacketCount(runtimeEcho map[string]string) int {
	if runtimeEcho == nil {
		return 0
	}
	for _, key := range []string{"official_stackchan_packets", "official_stackchan_auto_packets"} {
		if count, err := strconv.Atoi(strings.TrimSpace(runtimeEcho[key])); err == nil && count > 0 {
			return count
		}
	}
	return 0
}

func officialStackChanStatusPhysicalAccepted(runtimeEcho map[string]string) bool {
	if runtimeEcho == nil {
		return false
	}
	accepted, err := strconv.ParseBool(strings.TrimSpace(runtimeEcho["official_stackchan_physical_accepted"]))
	return err == nil && accepted
}

func officialStackChanStatusSurfaces(runtimeEcho map[string]string) map[string]string {
	if runtimeEcho == nil {
		return nil
	}
	surfaces := map[string]string{}
	for key, value := range runtimeEcho {
		surface, ok := strings.CutPrefix(key, "official_stackchan_")
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		switch surface {
		case "packets", "physical_accepted", "last_motion", "auto_state", "auto_reason", "auto_target", "auto_packets":
			continue
		}
		surfaces[surface] = value
	}
	if len(surfaces) == 0 {
		return nil
	}
	return surfaces
}

func (s *Server) recordOfficialStackChanConnected(deviceID string) {
	key := deviceIDLookupKey(deviceID)
	nowMS := s.now().UnixMilli()
	s.recordTrace("", "", key, "stackchan.official_ws.connected", nowMS)
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[key]
	if record.DeviceID == "" {
		record.DeviceID = key
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.ConnectionStatus = "official_stackchan_ws_connected"
	record.LastSeenMS = nowMS
	record.LastEvent = protocol.DeviceEventKind("stackchan.official_ws.connected")
	s.devices[key] = record
}

func (s *Server) writeXiaozhiOfficialStackChanState(ctx context.Context, session *xiaozhiSession, state string, reason string, force bool) bool {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return false
	}
	event := xiaozhitransport.DeviceExtensionEvent{
		Kind:  xiaozhitransport.DeviceEventKindState,
		Value: state,
	}
	s.recordDeviceDisplayState(id.deviceID, id.traceID, id.sessionID, "xiaozhi", state)
	s.maybeSendXiaozhiStateReaction(ctx, session, state, reason, force)
	packets, err := stackchantransport.BuildOfficialPackets(event)
	if err != nil {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "stackchan.official_auto.unsupported", s.now().UnixMilli())
		return false
	}
	socket, officialDeviceID, ok := s.officialStackChanSocketForXiaozhiDevice(id.deviceID)
	if !ok {
		if reason != "opus_downlink" {
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "stackchan.official_auto.not_connected", s.now().UnixMilli())
		}
		return false
	}
	if !session.claimOfficialStackChanState(state, force) {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	writeCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	socket.writeMu.Lock()
	for _, packet := range packets {
		err = socket.conn.Write(writeCtx, websocket.MessageBinary, packet.Bytes())
		if err != nil {
			break
		}
	}
	socket.writeMu.Unlock()
	if err != nil {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "stackchan.official_auto.delivery_error", s.now().UnixMilli())
		return false
	}
	s.recordOfficialStackChanAutoDelivered(id.deviceID, officialDeviceID, id.traceID, id.sessionID, event, len(packets), reason)
	return true
}

func (s *Server) recordOfficialStackChanAutoDelivered(deviceID string, officialDeviceID string, traceID string, sessionID string, event xiaozhitransport.DeviceExtensionEvent, packetCount int, reason string) {
	nowMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.official_auto."+string(event.Kind), nowMS)
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.official_auto.delivered", nowMS)
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[deviceID]
	if record.DeviceID == "" {
		record.DeviceID = deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	if event.Kind == xiaozhitransport.DeviceEventKindState {
		record.CurrentExpr = protocol.ExpressionState(event.Value)
	}
	record.LastEvent = protocol.DeviceEventKind("stackchan.official_auto." + string(event.Kind))
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = nowMS
	record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, map[string]string{
		"official_stackchan_auto_state":   event.Value,
		"official_stackchan_auto_reason":  safeGatewayFallbackToken(reason, "unknown"),
		"official_stackchan_auto_target":  officialDeviceID,
		"official_stackchan_auto_packets": strconv.Itoa(packetCount),
	})
	s.devices[deviceID] = record
}

func xiaozhiOfficialStateForTTSStop(reason string) string {
	reason = strings.ToLower(strings.TrimSpace(reason))
	if reason == "" || strings.Contains(reason, "abort") || strings.Contains(reason, "barge") || strings.Contains(reason, "wake") {
		return "listening"
	}
	if strings.Contains(reason, "error") || strings.Contains(reason, "unavailable") {
		return "error"
	}
	return "idle"
}

func (s *Server) recordXiaozhiDeviceControlDelivered(deviceID string, traceID string, sessionID string, event xiaozhitransport.DeviceExtensionEvent) {
	nowMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, deviceID, "xiaozhi.device_command."+string(event.Kind), nowMS)
	s.recordTrace(traceID, sessionID, deviceID, "xiaozhi.device_command.delivered", nowMS)
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[deviceID]
	if record.DeviceID == "" {
		record.DeviceID = deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	switch event.Kind {
	case xiaozhitransport.DeviceEventKindState, xiaozhitransport.DeviceEventKindFace:
		record.CurrentExpr = protocol.ExpressionState(event.Value)
	case xiaozhitransport.DeviceEventKindMotion:
		record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, map[string]string{"last_motion_command": event.Value})
	case xiaozhitransport.DeviceEventKindDisplay:
		record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, map[string]string{"last_display_slot": event.Value})
	}
	record.LastEvent = protocol.DeviceEventKind("xiaozhi.device_command." + string(event.Kind))
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = nowMS
	s.devices[deviceID] = record
}

func (s *Server) recordOfficialStackChanControlDelivered(deviceID string, traceID string, sessionID string, event xiaozhitransport.DeviceExtensionEvent, metadata stackchantransport.OfficialActionMetadata) {
	nowMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.official_control."+string(event.Kind), nowMS)
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.official_control.delivered", nowMS)
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[deviceID]
	if record.DeviceID == "" {
		record.DeviceID = deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.ConnectionStatus = "official_stackchan_ws_connected"
	record.LastEvent = protocol.DeviceEventKind("stackchan.official_control." + string(event.Kind))
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = nowMS
	switch event.Kind {
	case xiaozhitransport.DeviceEventKindState, xiaozhitransport.DeviceEventKindFace:
		record.CurrentExpr = protocol.ExpressionState(event.Value)
	case xiaozhitransport.DeviceEventKindMotion:
		record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, map[string]string{"official_stackchan_last_motion": event.Value})
	}
	echo := map[string]string{
		"official_stackchan_packets":           strconv.Itoa(metadata.PacketCount),
		"official_stackchan_physical_accepted": strconv.FormatBool(metadata.PhysicalAccepted),
	}
	for key, value := range metadata.Surfaces {
		echo["official_stackchan_"+key] = value
	}
	record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, echo)
	s.devices[deviceID] = record
}

func (s *Server) recordOfficialStackChanMCPFallbackDelivered(deviceID string, traceID string, sessionID string, event xiaozhitransport.DeviceExtensionEvent, metadata stackchantransport.OfficialActionMetadata, fallbackName string, stepCount int, surfaces map[string]string) {
	nowMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.official_mcp_fallback."+string(event.Kind), nowMS)
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.official_mcp_fallback.delivered", nowMS)
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[deviceID]
	if record.DeviceID == "" {
		record.DeviceID = deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.LastEvent = protocol.DeviceEventKind("stackchan.official_mcp_fallback." + string(event.Kind))
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = nowMS
	switch event.Kind {
	case xiaozhitransport.DeviceEventKindState, xiaozhitransport.DeviceEventKindFace:
		record.CurrentExpr = protocol.ExpressionState(event.Value)
	case xiaozhitransport.DeviceEventKindMotion:
		record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, map[string]string{"official_stackchan_last_motion": event.Value})
	}
	echo := map[string]string{
		"official_stackchan_packets":           "0",
		"official_stackchan_planned_packets":   strconv.Itoa(metadata.PacketCount),
		"official_stackchan_physical_accepted": "false",
		"official_stackchan_fallback":          "xiaozhi_mcp_sequence",
		"official_stackchan_fallback_name":     fallbackName,
		"official_stackchan_fallback_status":   "delivered",
		"official_stackchan_fallback_steps":    strconv.Itoa(stepCount),
		"official_stackchan_official_relay":    "disconnected",
		"official_stackchan_event":             string(event.Kind),
		"official_stackchan_value":             event.Value,
		"official_stackchan_packet_delivery":   "not_sent_no_official_ws",
		"official_stackchan_fallback_reason":   "official_stackchan_ws_disconnected",
	}
	for key, value := range surfaces {
		echo["official_stackchan_"+key] = value
	}
	record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, echo)
	s.devices[deviceID] = record
}

func boolValue(value bool) *bool {
	return &value
}

func (s *Server) hardwareAcceptance(deviceID string) HardwareAcceptanceResponse {
	record, found := s.deviceRecord(deviceID)
	items := []HardwareAcceptanceItem{
		hardwareModeRitualAcceptanceItem(record, found),
		hardwareFullCheckAcceptanceItem(record, found),
		hardwarePowerLifecycleAcceptanceItem(record, found),
	}
	acceptedCount := 0
	deliveredCount := 0
	for _, item := range items {
		if item.PhysicalAccepted {
			acceptedCount++
		}
		if item.DeliveryStatus == "delivered" {
			deliveredCount++
		}
	}
	overall := "machine_evidence_pending"
	if !found {
		overall = "device_missing"
	} else if acceptedCount == len(items) {
		overall = "accepted"
	} else if deliveredCount > 0 {
		overall = "physical_pending"
	}
	return HardwareAcceptanceResponse{
		SchemaVersion:    HardwareAcceptanceSchemaVersion,
		Service:          DeviceRegistryServiceName,
		DeviceID:         deviceID,
		ConnectionStatus: record.ConnectionStatus,
		OverallStatus:    overall,
		PhysicalAccepted: found && acceptedCount == len(items),
		Items:            items,
	}
}

func (s *Server) powerLifecycle(deviceID string) PowerLifecycleResponse {
	record, found := s.deviceRecord(deviceID)
	xiaozhiOnline := s.xiaozhiSocketOnline(deviceID)
	capabilities := record.Capabilities
	runtimeEcho := record.RuntimeEcho
	batteryTelemetry := "missing"
	if status := strings.TrimSpace(capabilities["battery"]); status != "" {
		batteryTelemetry = status
	}
	hasRuntimeBatteryTelemetry := strings.TrimSpace(runtimeEcho["battery_mv"]) != "" ||
		strings.TrimSpace(runtimeEcho["battery_level"]) != ""
	if hasRuntimeBatteryTelemetry {
		batteryTelemetry = "diagnostic_runtime_echo"
	}
	physicalAccepted := found && capabilities["power_lifecycle_physical_accepted"] == "true"
	pmicProfileStatus := strings.TrimSpace(capabilities["stackchan_pmic_power_key_profile"])
	if pmicProfileStatus == "" {
		pmicProfileStatus = strings.TrimSpace(capabilities["last_power_lifecycle_pmic_profile"])
	}
	pmicProfileAccepted := physicalAccepted && pmicProfileStatus == stackChanPMICPowerKeyProfileAXP2101V1
	items := []PowerLifecycleItem{
		powerLifecycleItem("runtime_online", "Runtime online", found && record.ConnectionStatus == "online", false, record.ConnectionStatus, "gateway_registry", "reconnect_device"),
		powerLifecycleItem("xiaozhi_socket", "Xiaozhi socket", xiaozhiOnline, false, boolStatus(xiaozhiOnline), "gateway_socket_registry", "restore_xiaozhi_socket"),
		powerLifecycleItem("battery_telemetry", "Battery telemetry", hasRuntimeBatteryTelemetry, false, batteryTelemetry, "device_runtime_echo", "run_sensor_battery_diagnostic_probe"),
		powerLifecycleItem("pmic_power_key_profile", "PMIC power-key profile", pmicProfileAccepted, pmicProfileAccepted, pmicProfileStatus, "operator_or_instrument", "accept_pmic_power_key_profile"),
		powerLifecycleItem("no_cable_cold_boot", "No-cable cold boot", physicalAccepted, physicalAccepted, capabilities["last_power_lifecycle_acceptance_status"], "operator_or_instrument", "accept_no_cable_cold_boot"),
		powerLifecycleItem("physical_power_button", "Physical power button", physicalAccepted, physicalAccepted, capabilities["last_power_lifecycle_acceptance_status"], "operator_or_instrument", "accept_power_button_start"),
	}
	overall := "device_missing"
	switch {
	case !found:
		overall = "device_missing"
	case physicalAccepted:
		overall = "accepted"
	case record.ConnectionStatus != "online" || !xiaozhiOnline:
		overall = "blocked"
	default:
		overall = "physical_pending"
	}
	return PowerLifecycleResponse{
		SchemaVersion:    "a21.gateway.power_lifecycle.v1",
		Service:          DeviceRegistryServiceName,
		DeviceID:         deviceID,
		ConnectionStatus: record.ConnectionStatus,
		OverallStatus:    overall,
		PhysicalAccepted: physicalAccepted,
		XiaozhiWSOnline:  xiaozhiOnline,
		BatteryTelemetry: batteryTelemetry,
		Items:            items,
		ResultRedacted:   true,
	}
}

func (s *Server) xiaozhiSocketOnline(deviceID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.xiaozhiSockets[deviceID] != nil
}

func (s *Server) deviceRecord(deviceID string) (DeviceRecord, bool) {
	for _, record := range s.deviceRecords() {
		if record.DeviceID == deviceID {
			return record, true
		}
	}
	return DeviceRecord{DeviceID: deviceID, ConnectionStatus: "missing"}, false
}

func hardwareModeRitualAcceptanceItem(record DeviceRecord, found bool) HardwareAcceptanceItem {
	capabilities := record.Capabilities
	delivered := found &&
		capabilities["last_voice_mode_ritual_status"] == "delivered" &&
		capabilities["last_voice_mode_ritual_completed"] == "true"
	accepted := found && capabilities["voice_mode_ritual_physical_accepted"] == "true"
	return hardwareAcceptanceItem(
		"mode_ritual",
		"Mode ritual",
		delivered,
		accepted,
		capabilities["last_voice_mode_ritual_trace_id"],
		capabilities["last_voice_mode_ritual_session_id"],
		"/v1/voice-mode-ritual-acceptance",
		"run_mode_ritual",
		"accept_visible_mode_ritual",
	)
}

func hardwareFullCheckAcceptanceItem(record DeviceRecord, found bool) HardwareAcceptanceItem {
	capabilities := record.Capabilities
	delivered := found &&
		capabilities["last_body_scene"] == "full_check" &&
		capabilities["last_body_scene_status"] == "delivered"
	accepted := found && capabilities["body_scene_physical_accepted"] == "true"
	return hardwareAcceptanceItem(
		"full_check",
		"Full check",
		delivered,
		accepted,
		capabilities["last_body_scene_trace_id"],
		capabilities["last_body_scene_session_id"],
		"/v1/xiaozhi/body-scene-acceptance",
		"run_full_check",
		"accept_visible_full_check",
	)
}

func hardwarePowerLifecycleAcceptanceItem(record DeviceRecord, found bool) HardwareAcceptanceItem {
	capabilities := record.Capabilities
	delivered := found && record.ConnectionStatus == "online"
	accepted := found && capabilities["power_lifecycle_physical_accepted"] == "true"
	return hardwareAcceptanceItem(
		"power_lifecycle",
		"Power lifecycle",
		delivered,
		accepted,
		capabilities["last_power_lifecycle_acceptance_trace_id"],
		capabilities["last_power_lifecycle_acceptance_session_id"],
		"/v1/power-lifecycle-acceptance",
		"verify_no_cable_power_button_boot",
		"accept_no_cable_power_button_boot",
	)
}

func hardwareAcceptanceItem(id string, label string, delivered bool, accepted bool, traceID string, sessionID string, endpoint string, runAction string, acceptAction string) HardwareAcceptanceItem {
	deliveryStatus := "not_delivered"
	nextAction := runAction
	if delivered {
		deliveryStatus = "delivered"
		nextAction = acceptAction
	}
	if accepted {
		nextAction = "accepted"
	}
	return HardwareAcceptanceItem{
		ID:                 id,
		Label:              label,
		DeliveryStatus:     deliveryStatus,
		PhysicalAccepted:   accepted,
		TraceID:            strings.TrimSpace(traceID),
		SessionID:          strings.TrimSpace(sessionID),
		AcceptanceEndpoint: endpoint,
		NextAction:         nextAction,
	}
}

func powerLifecycleItem(id string, label string, ready bool, accepted bool, status string, source string, nextAction string) PowerLifecycleItem {
	status = strings.TrimSpace(status)
	if status == "" {
		status = "missing"
	}
	if ready && !accepted {
		status = "ready"
	}
	if accepted {
		status = "accepted"
		nextAction = "accepted"
	}
	return PowerLifecycleItem{
		ID:               id,
		Label:            label,
		Status:           status,
		PhysicalAccepted: accepted,
		EvidenceSource:   source,
		NextAction:       nextAction,
	}
}

func boolStatus(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func (s *Server) deviceRecords() []DeviceRecord {
	roleplayState := s.currentRoleplayDeviceState()
	s.mu.Lock()
	defer s.mu.Unlock()
	nowMS := s.now().UnixMilli()
	voiceMode := defaultVoiceMode(s.voiceModeConfig)
	voiceChainMode := defaultVoiceChainMode(s.voiceChainModeConfig)
	asrProfile := defaultCascadeASRProfile(s.cascadeASRProfileConfig)
	llmProfile := defaultCascadeLLMProfile(s.cascadeLLMProfileConfig)
	ttsProfile := voiceChainTTSForVoice(defaultFixedTTSProfile(s.fixedTTSProfileConfig), defaultVoiceCloneProfile(s.voiceCloneProfileConfig))
	realtimeProvider := defaultRealtimeProvider(s.realtimeProviderConfig)
	voiceCloneProfile := defaultVoiceCloneProfile(s.voiceCloneProfileConfig)
	cloudVoiceProfile := providers.DefaultCloudVoiceProfile(s.cloudVoiceProfileConfig)
	records := make([]DeviceRecord, 0, len(s.devices))
	for _, record := range s.devices {
		record.CurrentVoiceMode = voiceMode
		record.CurrentVoiceChainMode = voiceChainMode
		record.CurrentASRProfile = asrProfile
		record.CurrentLLMProfile = llmProfile
		record.CurrentTTSProfile = ttsProfile
		record.CurrentRealtimeProvider = realtimeProvider
		record.CurrentVoiceCloneProfile = voiceCloneProfile
		applyRoleplayDeviceState(&record, roleplayState)
		record.CurrentCloudVoiceProfile = cloudVoiceProfile
		record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, roleplayDeviceRuntimeEcho(roleplayState))
		if socket := s.xiaozhiSockets[record.DeviceID]; socket != nil {
			record.ConnectionStatus = "online"
		}
		records = append(records, withDeviceFreshness(record, nowMS))
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].DeviceID < records[j].DeviceID
	})
	return records
}

const deviceOnlineWindowMS int64 = 300_000

func withDeviceFreshness(record DeviceRecord, nowMS int64) DeviceRecord {
	if record.LastSeenMS <= 0 {
		record.ConnectionStatus = "unknown"
		return record
	}
	ageMS := nowMS - record.LastSeenMS
	if ageMS < 0 {
		ageMS = 0
	}
	record.DeviceAgeMS = ageMS
	if record.ConnectionStatus == "xiaozhi_ws_disconnected" {
		return record
	}
	if ageMS <= deviceOnlineWindowMS {
		record.ConnectionStatus = "online"
	} else {
		record.ConnectionStatus = "stale"
	}
	return record
}

var a21FirmwareCommitPattern = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

var a21FirmwareVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[A-Za-z0-9.-]+)?$`)

func validateFirmwareIdentity(identity DeviceFirmwareIdentity) (string, string) {
	if identity.ID == "" && identity.Version == "" && identity.Board == "" && identity.Commit == "" {
		return "unknown", ""
	}
	values := []string{identity.ID, identity.Version, identity.Board, identity.Commit}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), "x21") || strings.Contains(strings.ToLower(value), "v21") {
			return "invalid", "firmware identity contains forbidden legacy identity"
		}
	}
	if identity.ID != "a21-stackchan" {
		return "invalid", "firmware_id must be a21-stackchan"
	}
	if !a21FirmwareVersionPattern.MatchString(identity.Version) {
		return "invalid", "firmware_version must be semver-like"
	}
	if identity.Board != "m5stack-cores3" {
		return "invalid", "firmware_board must be m5stack-cores3"
	}
	if !a21FirmwareCommitPattern.MatchString(identity.Commit) {
		return "invalid", "firmware_commit must be a git sha"
	}
	return "ok", ""
}

func validA21DeviceID(deviceID string) bool {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false
	}
	lower := strings.ToLower(deviceID)
	return !strings.Contains(lower, "x21") && !strings.Contains(lower, "v21")
}

func deviceIDLookupKey(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if hardwareMACDeviceID(deviceID) {
		return strings.ToLower(deviceID)
	}
	return deviceID
}

func hardwareMACDeviceID(deviceID string) bool {
	parts := strings.Split(strings.TrimSpace(deviceID), ":")
	if len(parts) != 6 {
		return false
	}
	for _, part := range parts {
		if len(part) != 2 {
			return false
		}
		for _, ch := range part {
			if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
				return false
			}
		}
	}
	return true
}

func (s *Server) deviceControlEvents(req DeviceControlRequest) []protocol.Envelope {
	payload := protocol.ControlEventPayload{
		State:                    req.State,
		Mode:                     req.Mode,
		Text:                     req.Text,
		Final:                    req.State != protocol.ExpressionListening && req.State != protocol.ExpressionThinking,
		StreamID:                 req.StreamID,
		DiagnosticToneHz:         req.DiagnosticToneHz,
		DiagnosticToneDurationMS: req.DiagnosticToneDurationMS,
		DiagnosticToneVolume:     req.DiagnosticToneVolume,
	}
	events := s.controlSequence(req.DeviceID, req.TraceID, req.SessionID, []protocol.ControlEventPayload{payload})
	if len(req.AudioChunks) > 0 {
		s.setActiveStream(req.TraceID, req.SessionID, req.DeviceID, req.StreamID)
		sentAt := s.now().UnixMilli()
		for i, chunk := range req.AudioChunks {
			events = append(events, s.voiceAudioPlaybackChunk(
				req.DeviceID,
				req.TraceID,
				req.SessionID,
				uint64(len(events)+1),
				sentAt+int64(i+1),
				chunk.StreamID,
				&providers.VoiceAudioChunk{
					Codec:        string(chunk.Codec),
					SampleRateHz: chunk.SampleRateHz,
					Channels:     chunk.Channels,
					DurationMS:   chunk.DurationMS,
					DataBase64:   chunk.DataBase64,
				},
			))
		}
		return events
	}
	mockAudioChunks := 0
	if req.MockAudioChunks != nil {
		mockAudioChunks = *req.MockAudioChunks
	}
	if mockAudioChunks <= 0 || req.StreamID == "" {
		return events
	}
	s.setActiveStream(req.TraceID, req.SessionID, req.DeviceID, req.StreamID)
	sentAt := s.now().UnixMilli()
	for i := 0; i < mockAudioChunks; i++ {
		events = append(events, s.voiceAudioPlaybackChunk(
			req.DeviceID,
			req.TraceID,
			req.SessionID,
			uint64(len(events)+1),
			sentAt+int64(i+1),
			req.StreamID,
			&providers.VoiceAudioChunk{
				Codec:        string(protocol.AudioCodecPCMS16LE),
				SampleRateHz: 16000,
				Channels:     1,
				DurationMS:   20,
				DataBase64:   mockPCM16SquareWaveBase64(16000, 20),
			},
		))
	}
	return events
}
