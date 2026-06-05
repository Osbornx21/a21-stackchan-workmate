package gateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
)

func (s *Server) handleVoiceModeRitualAcceptance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req VoiceModeRitualAcceptanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	mode := canonicalVoiceMode(req.VoiceMode)
	if mode == "" {
		http.Error(w, "voice_mode must be roleplay or professional", http.StatusBadRequest)
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
	if !s.hasMatchingVoiceModeRitualEvidence(req.DeviceID, mode, traceID, sessionID) {
		http.Error(w, "matching voice mode ritual evidence is required before physical acceptance", http.StatusConflict)
		return
	}

	nowMS := s.now().UnixMilli()
	marker := "voice_mode.ritual." + mode + ".physical_acceptance.accepted"
	s.recordTrace(traceID, sessionID, req.DeviceID, marker, nowMS)
	s.recordVoiceModeRitualPhysicalAcceptance(req.DeviceID, mode, traceID, sessionID, observer, nowMS, marker)
	writeJSON(w, http.StatusOK, VoiceModeRitualAcceptanceResponse{
		SchemaVersion:     "a21.gateway.voice_mode_ritual_acceptance.v1",
		TraceID:           traceID,
		SessionID:         sessionID,
		DeviceID:          req.DeviceID,
		Status:            "accepted",
		SelectedVoiceMode: mode,
		Observer:          observer,
		AcceptedSurfaces:  []string{"screen", "rgb", "servo"},
		PhysicalAccepted:  true,
		ResultRedacted:    true,
		AcceptanceEventMS: nowMS,
	})
}

func (s *Server) handleRoleplayProfile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		response, err := s.roleplayProfileResponse(RoleplayProfileSelectionRequest{})
		if err != nil {
			http.Error(w, "roleplay profile unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, response)
	case http.MethodPost, http.MethodPut:
		var req RoleplayProfileSelectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := s.setRoleplayProfile(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		response, err := s.roleplayProfileResponse(RoleplayProfileSelectionRequest{})
		if err != nil {
			http.Error(w, "roleplay profile unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, response)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProfessionalWorkspace(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		response, err := s.professionalWorkspaceResponse(ProfessionalWorkspaceSelectionRequest{})
		if err != nil {
			http.Error(w, "professional workspace unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, response)
	case http.MethodPost, http.MethodPut:
		var req ProfessionalWorkspaceSelectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := s.setProfessionalWorkspace(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		response, err := s.professionalWorkspaceResponse(ProfessionalWorkspaceSelectionRequest{})
		if err != nil {
			http.Error(w, "professional workspace unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, response)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProfessionalReadRecords(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		response := s.professionalReadRecordsResponse(
			strings.TrimSpace(r.URL.Query().Get("record_id")),
			strings.TrimSpace(r.URL.Query().Get("trace_id")),
			strings.TrimSpace(r.URL.Query().Get("session_id")),
		)
		writeJSON(w, http.StatusOK, response)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProfessionalQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	req, err := decodeProfessionalQueryRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Text) == "" && strings.TrimSpace(req.Utterance) == "" {
		http.Error(w, "text or utterance is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.DeviceID) != "" && safeWorkspaceDeviceID(req.DeviceID) == "" {
		http.Error(w, "valid A21 device_id is required", http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	s.recordTrace(traceID, sessionID, req.DeviceID, "http.professional_query.received", s.now().UnixMilli())
	response, err := s.executeProfessionalQuery(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleWorkspaceDocuments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "multipart/form-data") {
			http.Error(w, "workspace document upload requires multipart/form-data", http.StatusUnsupportedMediaType)
			return
		}
		maxBytes := s.workspaceDocumentMaxBytes
		if maxBytes <= 0 {
			maxBytes = defaultWorkspaceDocumentMaxBytes
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes+workspaceDocumentMultipartOverheadBytes)
		if err := r.ParseMultipartForm(workspaceDocumentMultipartOverheadBytes); err != nil {
			http.Error(w, "workspace document upload invalid", http.StatusBadRequest)
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "workspace document file is required", http.StatusBadRequest)
			return
		}
		defer file.Close()
		response, err := s.storeWorkspaceDocumentUpload(workspaceDocumentUploadRequestFromForm(r), file, header)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, errWorkspaceDocumentTooLarge) {
				status = http.StatusRequestEntityTooLarge
			}
			http.Error(w, err.Error(), status)
			return
		}
		writeJSON(w, http.StatusOK, response)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWorkspaceUploadJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.workspaceUploadJobsResponse(strings.TrimSpace(r.URL.Query().Get("job_id")), nil))
	case http.MethodPost, http.MethodPut:
		req, err := decodeWorkspaceUploadJobRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var response WorkspaceUploadJobsResponse
		if r.Method == http.MethodPost && strings.TrimSpace(req.Action) == "" {
			response, err = s.createWorkspaceUploadJob(req)
		} else {
			response, err = s.applyWorkspaceUploadJobAction(req)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, response)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWorkspaceIndexJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		response, err := s.workspaceIndexJobsResponse(
			strings.TrimSpace(r.URL.Query().Get("index_job_id")),
			strings.TrimSpace(r.URL.Query().Get("document_id")),
			strings.TrimSpace(r.URL.Query().Get("job_id")),
			strings.TrimSpace(r.URL.Query().Get("source_id")),
			nil,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, response)
	case http.MethodPost:
		req, err := decodeWorkspaceIndexJobRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		response, err := s.createWorkspaceIndexJob(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, response)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWorkspaceSources(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		response, err := s.workspaceSourcesResponse(
			strings.TrimSpace(r.URL.Query().Get("source_id")),
			strings.TrimSpace(r.URL.Query().Get("user_id")),
			strings.TrimSpace(r.URL.Query().Get("workspace_id")),
			strings.TrimSpace(r.URL.Query().Get("source_scope")),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, response)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWorkspaceDeviceBindings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		response, err := s.workspaceDeviceBindingsResponse(
			strings.TrimSpace(r.URL.Query().Get("binding_id")),
			strings.TrimSpace(r.URL.Query().Get("device_id")),
			strings.TrimSpace(r.URL.Query().Get("user_id")),
			strings.TrimSpace(r.URL.Query().Get("workspace_id")),
			strings.TrimSpace(r.URL.Query().Get("status")),
			nil,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, response)
	case http.MethodPost, http.MethodPut:
		req, err := decodeWorkspaceDeviceBindingRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var response WorkspaceDeviceBindingsResponse
		if r.Method == http.MethodPost && strings.TrimSpace(req.Action) == "" {
			response, err = s.createWorkspaceDeviceBinding(req)
		} else {
			response, err = s.applyWorkspaceDeviceBindingAction(req)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, response)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleVoiceChainProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.voiceChainProfileCatalog())
	case http.MethodPost, http.MethodPut:
		var req VoiceChainProfileSelectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := s.setVoiceChainProfile(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, s.voiceChainProfileCatalog())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGatewayProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.gatewayProfileCatalog(r))
	case http.MethodPost, http.MethodPut:
		var req GatewayProfileSelectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if !validGatewayProfile(req.GatewayProfile) {
			http.Error(w, "gateway_profile must be mac_local or public_wss", http.StatusBadRequest)
			return
		}
		if req.GatewayProfile == GatewayProfilePublicWSS && s.publicGatewayWebSocketURL() == "" {
			http.Error(w, "gateway_profile public_wss requires A21_PUBLIC_GATEWAY_URL", http.StatusConflict)
			return
		}
		s.setGatewayProfile(req.GatewayProfile)
		writeJSON(w, http.StatusOK, s.gatewayProfileCatalog(r))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCloudVoiceProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.cloudVoiceProfileCatalog())
	case http.MethodPost, http.MethodPut:
		var req CloudVoiceProfileSelectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if !providers.ValidCloudVoiceProfile(req.CloudVoiceProfile) {
			http.Error(w, "cloud_voice_profile must be a known A21 cloud voice profile", http.StatusBadRequest)
			return
		}
		s.setCloudVoiceProfile(req.CloudVoiceProfile)
		writeJSON(w, http.StatusOK, s.cloudVoiceProfileCatalog())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) voiceModeCatalog() VoiceModeCatalogResponse {
	selected := s.selectedVoiceMode()
	return VoiceModeCatalogResponse{
		SchemaVersion:     VoiceModeSchemaVersion,
		Service:           DeviceRegistryServiceName,
		SelectedVoiceMode: selected,
		SelectedRitual:    voiceModeRitual(selected),
		Modes: []VoiceModeOption{
			{
				ID:          VoiceModeRoleplay,
				Label:       "Roleplay",
				Status:      "available",
				Default:     true,
				Description: "role personality, memory hints, voice clone, low-latency speech, and stock Xiaozhi playback",
				Ritual:      voiceModeRitual(VoiceModeRoleplay),
			},
			{
				ID:          VoiceModeProfessional,
				Label:       "Professional",
				Status:      "available",
				Description: "explicit evidence-first V21 adapter path; not routed through dialogue endpoints",
				Ritual:      voiceModeRitual(VoiceModeProfessional),
			},
		},
	}
}

func voiceModeRitual(mode string) VoiceModeRitual {
	switch defaultVoiceMode(mode) {
	case VoiceModeProfessional:
		return VoiceModeRitual{
			Mode:             VoiceModeProfessional,
			ScreenLabel:      "PRO",
			CueText:          v21adapter.ProfessionalCheckingFeedbackText,
			Expression:       protocol.ExpressionProfessional,
			TraceMarker:      "professional.checking_feedback.sent",
			WorkspacePolicy:  "professional_only",
			V21Allowed:       true,
			PhysicalAccepted: false,
		}
	default:
		return VoiceModeRitual{
			Mode:             VoiceModeRoleplay,
			ScreenLabel:      "A21",
			CueText:          "我在。你说，我先接住。",
			Expression:       protocol.ExpressionListening,
			TraceMarker:      "roleplay.profile.ready",
			WorkspacePolicy:  "roleplay_private_no_v21",
			V21Allowed:       false,
			PhysicalAccepted: false,
		}
	}
}

func voiceModeRitualPlans(deviceID string, mode string) (string, VoiceModeRitual, []XiaozhiMCPControlRequest, error) {
	canonical := canonicalVoiceMode(mode)
	if canonical == "" {
		return "", VoiceModeRitual{}, nil, errors.New("voice_mode must be roleplay or professional")
	}
	base := XiaozhiMCPControlRequest{DeviceID: deviceID}
	brightness := func(value int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPScreenSetBrightnessToolName
		req.Brightness = xiaozhiPresetInt(value)
		return req
	}
	theme := func(value string) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPScreenSetThemeToolName
		req.Theme = value
		return req
	}
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
	switch canonical {
	case VoiceModeProfessional:
		return canonical, voiceModeRitual(canonical), []XiaozhiMCPControlRequest{
			theme("dark"),
			brightness(78),
			led(0, 84, 168),
			head(0, 32, 220),
		}, nil
	default:
		return canonical, voiceModeRitual(canonical), []XiaozhiMCPControlRequest{
			theme("auto"),
			brightness(58),
			led(120, 48, 96),
			head(0, 24, 180),
		}, nil
	}
}

func (s *Server) recordVoiceModeRitualCompleted(deviceID string, mode string, traceID string, sessionID string, atMS int64) {
	capabilities := map[string]string{
		"last_voice_mode_ritual":              mode,
		"last_voice_mode_ritual_status":       "delivered",
		"last_voice_mode_ritual_completed":    "true",
		"last_voice_mode_ritual_trace_id":     traceID,
		"last_voice_mode_ritual_session_id":   sessionID,
		"voice_mode_ritual_physical_accepted": "false",
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
	record.CurrentVoiceMode = mode
	record.LastEvent = protocol.DeviceEventKind("voice_mode.ritual." + mode + ".completed")
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = atMS
	s.devices[deviceID] = record
}

func (s *Server) hasMatchingVoiceModeRitualEvidence(deviceID string, mode string, traceID string, sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[deviceID]
	if record.DeviceID == "" || record.Capabilities == nil {
		return false
	}
	return record.Capabilities["last_voice_mode_ritual"] == mode &&
		record.Capabilities["last_voice_mode_ritual_status"] == "delivered" &&
		record.Capabilities["last_voice_mode_ritual_completed"] == "true" &&
		record.Capabilities["last_voice_mode_ritual_trace_id"] == traceID &&
		record.Capabilities["last_voice_mode_ritual_session_id"] == sessionID
}

func (s *Server) recordVoiceModeRitualPhysicalAcceptance(deviceID string, mode string, traceID string, sessionID string, observer string, atMS int64, event string) {
	capabilities := map[string]string{
		"voice_mode_ritual_physical_accepted":          "true",
		"voice_mode_ritual_screen_physical_accepted":   "true",
		"voice_mode_ritual_rgb_physical_accepted":      "true",
		"voice_mode_ritual_servo_physical_accepted":    "true",
		"last_voice_mode_ritual_acceptance_status":     "operator_visible_accepted",
		"last_voice_mode_ritual_acceptance_mode":       mode,
		"last_voice_mode_ritual_acceptance_trace_id":   traceID,
		"last_voice_mode_ritual_acceptance_session_id": sessionID,
		"last_voice_mode_ritual_acceptance_observer":   observer,
		"last_voice_mode_ritual_acceptance_at_ms":      strconv.FormatInt(atMS, 10),
	}
	if observer == "instrument" {
		capabilities["last_voice_mode_ritual_acceptance_status"] = "instrument_visible_accepted"
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
	record.CurrentVoiceMode = mode
	if cleanEvent := strings.TrimSpace(event); cleanEvent != "" {
		record.LastEvent = protocol.DeviceEventKind(cleanEvent)
	}
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = atMS
	s.devices[deviceID] = record
}

func professionalVoiceTriggerModeAllowed(mode protocol.Mode) bool {
	switch mode {
	case "", protocol.ModeWorkmate, protocol.ModeRoleplay, protocol.ModeCompanion:
		return true
	default:
		return false
	}
}

func professionalVoiceTrigger(text string) bool {
	normalized := normalizeProfessionalVoiceTriggerText(text)
	if normalized == "" {
		return false
	}
	for _, phrase := range []string{
		"不要进专业",
		"不用专业模式",
		"不要专业模式",
		"别查v21",
		"不要查v21",
		"不用查v21",
	} {
		if strings.Contains(normalized, phrase) {
			return false
		}
	}
	for _, phrase := range []string{
		"专业模式",
		"进入专业模式",
		"认真查",
		"认真查一下",
		"帮我查v21",
		"查v21",
		"给我证据",
	} {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func normalizeProfessionalVoiceTriggerText(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	replacer := strings.NewReplacer(
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
		"　", "",
	)
	return replacer.Replace(text)
}

func validVoiceMode(mode string) bool {
	return canonicalVoiceMode(mode) != ""
}

func canonicalVoiceMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case VoiceModeDialogue, VoiceModeRoleplay:
		return VoiceModeRoleplay
	case VoiceModeProfessional:
		return VoiceModeProfessional
	default:
		return ""
	}
}

func (s *Server) selectedVoiceMode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return defaultVoiceMode(s.voiceModeConfig)
}

func (s *Server) setVoiceMode(mode string) {
	canonical := canonicalVoiceMode(mode)
	if canonical == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.voiceModeConfig = canonical
}

func defaultVoiceMode(mode string) string {
	if canonical := canonicalVoiceMode(mode); canonical != "" {
		return canonical
	}
	return VoiceModeRoleplay
}
