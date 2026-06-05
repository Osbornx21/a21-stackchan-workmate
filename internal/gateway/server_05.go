package gateway

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"a21.local/a21/internal/personality"
	stackchantransport "a21.local/a21/internal/transport/stackchan"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"a21.local/a21/internal/v21adapter"
)

func voiceModeAvailableForFastCompanion(mode string) bool {
	return defaultVoiceMode(mode) == VoiceModeRoleplay
}

func plannedVoiceModeError(mode string) string {
	return "voice_mode " + defaultVoiceMode(mode) + " must use the professional path and cannot execute dialogue turns"
}

func (s *Server) roleplayProfileResponse(override RoleplayProfileSelectionRequest) (RoleplayProfileResponse, error) {
	summary, memory, err := s.roleplayRuntimeSummary(override)
	if err != nil {
		return RoleplayProfileResponse{}, err
	}
	expressionPlan := roleplayExpressionPlan(summary)
	return RoleplayProfileResponse{
		SchemaVersion:             RoleplayProfileSchemaVersion,
		Service:                   DeviceRegistryServiceName,
		SelectedRoleplayProfile:   summary.RoleplayProfile,
		SelectedScenario:          summary.Scenario,
		SelectedVoiceCloneProfile: summary.VoiceCloneProfile,
		Memory:                    memory,
		Runtime:                   summary,
		ExpressionPlan:            expressionPlan,
		Profiles:                  roleplayProfileOptions(summary.RoleplayProfile),
		Scenarios:                 roleplayScenarioOptions(summary.Scenario),
	}, nil
}

func (s *Server) setRoleplayProfile(req RoleplayProfileSelectionRequest) error {
	profile, scenario, voiceClone, err := s.resolveRoleplaySelection(req)
	if err != nil {
		return err
	}
	memoryConfigured := false
	memoryHints := []string(nil)
	memoryFindings := []personality.MemoryFinding(nil)
	updateMemory := req.ClearMemory || req.MemoryHints != nil
	if req.MemoryHints != nil {
		memoryHints, memoryFindings, memoryConfigured = sanitizeRoleplayMemoryHints(req.MemoryHints)
	}
	if strings.TrimSpace(req.VoiceCloneProfile) != "" {
		if err := s.setVoiceChainProfile(VoiceChainProfileSelectionRequest{VoiceCloneProfile: voiceClone}); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roleplayProfileConfig = profile
	s.roleplayScenarioConfig = scenario
	if updateMemory {
		s.roleplayMemoryConfigured = memoryConfigured
		s.roleplayMemoryHintsConfig = memoryHints
		s.roleplayMemoryFindings = memoryFindings
	}
	return nil
}

func (s *Server) roleplayRuntimeSummary(override RoleplayProfileSelectionRequest) (RoleplayRuntimeSummary, personality.MemoryState, error) {
	profile, scenario, voiceClone, err := s.resolveRoleplaySelection(override)
	if err != nil {
		return RoleplayRuntimeSummary{}, personality.MemoryState{}, err
	}
	memory, hints := s.roleplayMemoryHints()
	promptComposed := false
	if _, err := personality.Compose(personality.Options{
		Mode:             personality.ModeRoleplay,
		RoleSoul:         roleplayPersonalitySoul(profile),
		Scenario:         roleplayPersonalityScenario(scenario),
		UserText:         "我在。",
		MemoryHints:      hints,
		MaxResponseRunes: 12,
	}); err == nil {
		promptComposed = true
	}
	promptParts := roleplayPromptParts(profile, scenario, memory.PromptInputReady)
	return RoleplayRuntimeSummary{
		SchemaVersion:            "a21.roleplay_runtime.v1",
		Mode:                     VoiceModeRoleplay,
		RoleplayProfile:          profile,
		Scenario:                 scenario,
		VoiceCloneProfile:        voiceClone,
		SoulPromptInputReady:     roleplayPersonalitySoul(profile) != "",
		PromptParts:              promptParts,
		MemoryPolicy:             memory.Policy,
		MemoryConfigured:         memory.Configured,
		MemoryPromptInputReady:   memory.PromptInputReady,
		MemoryHintCount:          memory.UserPreferenceCount + memory.SessionMemoryCount,
		PromptComposed:           promptComposed,
		PromptStored:             false,
		MemoryTextStored:         false,
		TranscriptStored:         false,
		ProviderOutputStored:     false,
		VoiceCloneSampleStored:   false,
		ProfessionalRouteAllowed: false,
		V21Executed:              false,
	}, memory, nil
}

func (s *Server) currentRoleplayDeviceState() roleplayDeviceState {
	profile, scenario, _, err := s.resolveRoleplaySelection(RoleplayProfileSelectionRequest{})
	if err != nil {
		profile = DefaultRoleplayProfile
		scenario = DefaultRoleplayScenario
	}
	memory, _ := s.roleplayMemoryHints()
	return roleplayDeviceState{
		Profile:          defaultRoleplayProfile(profile),
		Scenario:         defaultRoleplayScenario(scenario),
		SoulReady:        roleplayPersonalitySoul(profile) != "",
		MemoryReady:      memory.PromptInputReady,
		MemoryHintCount:  memory.UserPreferenceCount + memory.SessionMemoryCount,
		PhysicalAccepted: false,
	}
}

func applyRoleplayDeviceState(record *DeviceRecord, state roleplayDeviceState) {
	if record == nil {
		return
	}
	record.CurrentRoleplayProfile = defaultRoleplayProfile(state.Profile)
	record.CurrentRoleplayScenario = defaultRoleplayScenario(state.Scenario)
	record.RoleplaySoulReady = state.SoulReady
	record.RoleplayMemoryReady = state.MemoryReady
	record.RoleplayMemoryHintCount = state.MemoryHintCount
	record.RoleplayPhysicalAccepted = state.PhysicalAccepted
}

func roleplayDeviceRuntimeEcho(state roleplayDeviceState) map[string]string {
	return map[string]string{
		"roleplay_profile":             defaultRoleplayProfile(state.Profile),
		"roleplay_scenario":            defaultRoleplayScenario(state.Scenario),
		"roleplay_soul_ready":          strconv.FormatBool(state.SoulReady),
		"roleplay_memory_ready":        strconv.FormatBool(state.MemoryReady),
		"roleplay_memory_hint_count":   strconv.Itoa(state.MemoryHintCount),
		"roleplay_physical_accepted":   strconv.FormatBool(state.PhysicalAccepted),
		"roleplay_prompt_text_stored":  "false",
		"roleplay_memory_text_stored":  "false",
		"roleplay_voice_sample_stored": "false",
	}
}

func roleplayExpressionPlan(summary RoleplayRuntimeSummary) RoleplayExpressionPlan {
	actions := make([]RoleplayExpressionAction, 0, 5)
	for _, candidate := range roleplayExpressionEvents(summary) {
		plan, err := stackchantransport.BuildOfficialActionPlan(candidate.event)
		if err != nil {
			continue
		}
		actions = append(actions, RoleplayExpressionAction{
			Phase:            candidate.phase,
			Event:            string(plan.Event.Kind),
			Value:            plan.Event.Value,
			PacketCount:      plan.Metadata.PacketCount,
			PhysicalAccepted: plan.Metadata.PhysicalAccepted,
			Surfaces:         copyStringMap(plan.Metadata.Surfaces),
		})
	}
	packetCount := 0
	physicalAccepted := false
	surfaces := make(map[string]string)
	for _, action := range actions {
		packetCount += action.PacketCount
		physicalAccepted = physicalAccepted || action.PhysicalAccepted
		for key, value := range action.Surfaces {
			surfaces[action.Phase+"_"+key] = value
		}
	}
	return RoleplayExpressionPlan{
		SchemaVersion:     "a21.roleplay_expression_plan.v1",
		Adapter:           "official_stackchan_action_plan",
		DeliveryPolicy:    "no_send_plan_only",
		RoleplayProfile:   summary.RoleplayProfile,
		Scenario:          summary.Scenario,
		VoiceCloneProfile: summary.VoiceCloneProfile,
		MemoryReady:       summary.MemoryPromptInputReady,
		MemoryHintCount:   summary.MemoryHintCount,
		ActionCount:       len(actions),
		PacketCount:       packetCount,
		PhysicalAccepted:  physicalAccepted,
		Actions:           actions,
		Surfaces:          surfaces,
		Redaction: RoleplayExpressionRedaction{
			PromptTextStored:       false,
			MemoryTextStored:       false,
			TranscriptStored:       false,
			ProviderOutputStored:   false,
			AudioStored:            false,
			VoiceCloneSampleStored: false,
		},
	}
}

type roleplayExpressionEvent struct {
	phase string
	event xiaozhitransport.DeviceExtensionEvent
}

func roleplayExpressionEvents(summary RoleplayRuntimeSummary) []roleplayExpressionEvent {
	events := []roleplayExpressionEvent{
		{
			phase: "baseline_posture",
			event: xiaozhitransport.DeviceExtensionEvent{Kind: xiaozhitransport.DeviceEventKindState, Value: "listening"},
		},
		{
			phase: "role_soul",
			event: roleplaySoulExpressionEvent(summary.RoleplayProfile),
		},
		{
			phase: "scenario_emphasis",
			event: roleplayScenarioExpressionEvent(summary.Scenario),
		},
	}
	if summary.MemoryPromptInputReady {
		events = append(events, roleplayExpressionEvent{
			phase: "memory_cue",
			event: xiaozhitransport.DeviceExtensionEvent{Kind: xiaozhitransport.DeviceEventKindMotion, Value: "nod"},
		})
	}
	return events
}

func roleplaySoulExpressionEvent(profile string) xiaozhitransport.DeviceExtensionEvent {
	switch defaultRoleplayProfile(profile) {
	case RoleplayProfileWryPeer:
		return xiaozhitransport.DeviceExtensionEvent{Kind: xiaozhitransport.DeviceEventKindFace, Value: "happy"}
	case RoleplayProfileCalmAnchor:
		return xiaozhitransport.DeviceExtensionEvent{Kind: xiaozhitransport.DeviceEventKindFace, Value: "attentive"}
	default:
		return xiaozhitransport.DeviceExtensionEvent{Kind: xiaozhitransport.DeviceEventKindFace, Value: "idle"}
	}
}

func roleplayScenarioExpressionEvent(scenario string) xiaozhitransport.DeviceExtensionEvent {
	switch defaultRoleplayScenario(scenario) {
	case "boss_challenge", "engineer_pushback", "user_complaint":
		return xiaozhitransport.DeviceExtensionEvent{Kind: xiaozhitransport.DeviceEventKindState, Value: "thinking"}
	case "late_night_radio":
		return xiaozhitransport.DeviceExtensionEvent{Kind: xiaozhitransport.DeviceEventKindState, Value: "speaking"}
	default:
		return xiaozhitransport.DeviceExtensionEvent{Kind: xiaozhitransport.DeviceEventKindState, Value: "listening"}
	}
}

func copyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func (s *Server) roleplayPromptInput(override RoleplayProfileSelectionRequest, userText string) (string, error) {
	profile, scenario, _, err := s.resolveRoleplaySelection(override)
	if err != nil {
		return "", err
	}
	_, hints := s.roleplayMemoryHints()
	prompt, err := personality.Compose(personality.Options{
		Mode:             personality.ModeRoleplay,
		RoleSoul:         roleplayPersonalitySoul(profile),
		Scenario:         roleplayPersonalityScenario(scenario),
		UserText:         userText,
		MemoryHints:      hints,
		MaxResponseRunes: 12,
	})
	if err != nil {
		return "", err
	}
	return prompt, nil
}

func (s *Server) roleplayMemoryHints() (personality.MemoryState, []personality.MemoryHint) {
	env, runtimeMemoryConfigured, runtimeMemoryFindings := s.roleplayRuntimeEnv()
	memory, hints := personality.MemoryStateFromEnv(env)
	return mergeRoleplayRuntimeMemoryFindings(memory, runtimeMemoryConfigured, runtimeMemoryFindings), hints
}

func (s *Server) resolveRoleplaySelection(override RoleplayProfileSelectionRequest) (string, string, string, error) {
	s.mu.Lock()
	profile := defaultRoleplayProfile(s.roleplayProfileConfig)
	scenario := defaultRoleplayScenario(s.roleplayScenarioConfig)
	voiceClone := defaultVoiceCloneProfile(s.voiceCloneProfileConfig)
	s.mu.Unlock()
	if strings.TrimSpace(override.RoleplayProfile) != "" {
		profile = strings.ToLower(strings.TrimSpace(override.RoleplayProfile))
		if !validRoleplayProfile(profile) {
			return "", "", "", fmt.Errorf("valid roleplay_profile is required")
		}
	}
	if strings.TrimSpace(override.Scenario) != "" {
		scenario = strings.ToLower(strings.TrimSpace(override.Scenario))
		if !validRoleplayScenario(scenario) {
			return "", "", "", fmt.Errorf("valid roleplay scenario is required")
		}
	}
	if strings.TrimSpace(override.VoiceCloneProfile) != "" {
		voiceClone = strings.ToLower(strings.TrimSpace(override.VoiceCloneProfile))
		if !validVoiceCloneProfile(voiceClone) {
			return "", "", "", fmt.Errorf("valid voice_clone_profile is required")
		}
	}
	return profile, scenario, voiceClone, nil
}

func (s *Server) roleplayRuntimeEnv() ([]string, bool, []personality.MemoryFinding) {
	env := os.Environ()
	s.mu.Lock()
	env = append(env, s.cloudVoiceEnv...)
	runtimeMemoryConfigured := s.roleplayMemoryConfigured
	runtimeMemoryHints := append([]string(nil), s.roleplayMemoryHintsConfig...)
	runtimeMemoryFindings := append([]personality.MemoryFinding(nil), s.roleplayMemoryFindings...)
	s.mu.Unlock()
	if len(runtimeMemoryHints) > 0 {
		env = append(env, personality.MemorySessionNotesEnv+"="+strings.Join(runtimeMemoryHints, "\n"))
	}
	return env, runtimeMemoryConfigured, runtimeMemoryFindings
}

func sanitizeRoleplayMemoryHints(raw []string) ([]string, []personality.MemoryFinding, bool) {
	configured := false
	values := make([]string, 0, len(raw))
	for _, value := range raw {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		configured = true
		values = append(values, value)
	}
	if len(values) == 0 {
		return nil, nil, configured
	}
	state, hints := personality.MemoryStateFromEnv([]string{
		personality.MemorySessionNotesEnv + "=" + strings.Join(values, "\n"),
	})
	safe := make([]string, 0, len(hints))
	for _, hint := range hints {
		if hint.Scope == personality.MemoryScopeSessionMemory {
			safe = append(safe, hint.Text)
		}
	}
	return safe, append([]personality.MemoryFinding(nil), state.Findings...), configured
}

func mergeRoleplayRuntimeMemoryFindings(memory personality.MemoryState, runtimeConfigured bool, findings []personality.MemoryFinding) personality.MemoryState {
	if runtimeConfigured {
		memory.Configured = true
		if !memory.PromptInputReady && len(findings) > 0 {
			memory.Status = "blocked"
		}
	}
	if len(findings) > 0 {
		merged := make([]personality.MemoryFinding, 0, len(findings)+len(memory.Findings))
		merged = append(merged, findings...)
		merged = append(merged, memory.Findings...)
		memory.Findings = merged
	}
	return memory
}

func roleplayPromptUserText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "我在。"
	}
	return text
}

func defaultRoleplayProfile(profile string) string {
	profile = strings.ToLower(strings.TrimSpace(profile))
	if validRoleplayProfile(profile) {
		return profile
	}
	return DefaultRoleplayProfile
}

func validRoleplayProfile(profile string) bool {
	return roleplayPersonalitySoul(profile) != ""
}

func defaultRoleplayScenario(scenario string) string {
	scenario = strings.ToLower(strings.TrimSpace(scenario))
	if validRoleplayScenario(scenario) {
		return scenario
	}
	return DefaultRoleplayScenario
}

func validRoleplayScenario(scenario string) bool {
	return roleplayPersonalityScenario(scenario) != ""
}

func roleplayPersonalityScenario(scenario string) personality.Scenario {
	switch strings.ToLower(strings.TrimSpace(scenario)) {
	case "pre_meeting":
		return personality.ScenarioPreMeeting
	case "post_meeting":
		return personality.ScenarioPostMeeting
	case "boss_challenge":
		return personality.ScenarioBossChallenge
	case "engineer_pushback":
		return personality.ScenarioEngineerPushback
	case "user_complaint":
		return personality.ScenarioUserComplaint
	case "desk_mouthpiece":
		return personality.ScenarioDeskMouthpiece
	case "late_night_radio":
		return personality.ScenarioLateNightRadio
	default:
		return ""
	}
}

func roleplayPersonalitySoul(profile string) personality.RoleSoul {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case DefaultRoleplayProfile:
		return personality.RoleSoulA21Default
	case RoleplayProfileWryPeer:
		return personality.RoleSoulWryPeer
	case RoleplayProfileCalmAnchor:
		return personality.RoleSoulCalmAnchor
	default:
		return ""
	}
}

func roleplayPromptParts(profile string, scenario string, memoryReady bool) []string {
	parts := []string{
		"core_identity",
		"tone_rules",
		"role_soul:" + defaultRoleplayProfile(profile),
		"mode:roleplay",
		"scenario:" + defaultRoleplayScenario(scenario),
	}
	if memoryReady {
		parts = append(parts, "memory_hints")
	}
	return parts
}

func roleplayProfileOptions(selected string) []RoleplayProfileOption {
	selected = defaultRoleplayProfile(selected)
	options := []RoleplayProfileOption{
		{
			ID:             DefaultRoleplayProfile,
			Label:          "紫悦桌面伙伴",
			Status:         "available",
			Description:    "close desk workmate that listens first and turns pressure into usable words",
			VoiceHint:      "natural_short_warm",
			ExpressionHint: "idle_listening_thinking_speaking",
			PromptParts:    roleplayPromptParts(DefaultRoleplayProfile, DefaultRoleplayScenario, false),
		},
		{
			ID:             RoleplayProfileWryPeer,
			Label:          "Wry peer",
			Status:         "available",
			Description:    "sharp loyal peer with dry humor for boundary pressure and review defense",
			VoiceHint:      "short_wry_clear",
			ExpressionHint: "thinking_speaking_interrupted",
			PromptParts:    roleplayPromptParts(RoleplayProfileWryPeer, DefaultRoleplayScenario, false),
		},
		{
			ID:             RoleplayProfileCalmAnchor,
			Label:          "Calm anchor",
			Status:         "available",
			Description:    "steady late-night co-thinker for overloaded work without therapy drift",
			VoiceHint:      "calm_low_noise",
			ExpressionHint: "idle_listening_local_fallback",
			PromptParts:    roleplayPromptParts(RoleplayProfileCalmAnchor, DefaultRoleplayScenario, false),
		},
	}
	for i := range options {
		options[i].Default = options[i].ID == selected
	}
	return options
}

func roleplayScenarioOptions(selected string) []RoleplayProfileOption {
	selected = defaultRoleplayScenario(selected)
	options := []RoleplayProfileOption{
		{ID: "desk_mouthpiece", Label: "Desk mouthpiece", Status: "available", Description: "Turn messy workplace emotion into usable language"},
		{ID: "boss_challenge", Label: "Boss challenge", Status: "available", Description: "Rehearse pressure and sharpen the answer"},
		{ID: "engineer_pushback", Label: "Engineer pushback", Status: "available", Description: "Prepare for implementation-boundary questions"},
		{ID: "user_complaint", Label: "User complaint", Status: "available", Description: "Practice user-facing response under pressure"},
		{ID: "pre_meeting", Label: "Pre-meeting", Status: "available", Description: "Get the framing ready before a meeting"},
		{ID: "post_meeting", Label: "Post-meeting", Status: "available", Description: "Clean up a meeting aftermath into next actions"},
		{ID: "late_night_radio", Label: "Late-night radio", Status: "available", Description: "Calmer reflective companionship without losing boundaries"},
	}
	for i := range options {
		options[i].Default = options[i].ID == selected
	}
	return options
}

func (s *Server) professionalWorkspaceResponse(override ProfessionalWorkspaceSelectionRequest) (ProfessionalWorkspaceResponse, error) {
	runtime, err := s.professionalWorkspaceRuntime(override)
	if err != nil {
		return ProfessionalWorkspaceResponse{}, err
	}
	return ProfessionalWorkspaceResponse{
		SchemaVersion:          ProfessionalWorkspaceSchemaVersion,
		Service:                DeviceRegistryServiceName,
		AdapterContractVersion: ProfessionalAdapterContractVersion,
		SelectedUserID:         runtime.UserID,
		SelectedWorkspaceID:    runtime.WorkspaceID,
		SelectedQueryScope:     runtime.QueryScope,
		WorkspaceStatus:        runtime.WorkspaceStatus,
		PrivacyScope:           runtime.PrivacyScope,
		V21ExecutionAllowed:    runtime.V21ExecutionAllowed,
		Runtime:                runtime,
		QueryScopes:            professionalQueryScopeOptions(runtime.QueryScope),
		Redaction:              professionalWorkspaceRedaction(),
	}, nil
}

func (s *Server) setProfessionalWorkspace(req ProfessionalWorkspaceSelectionRequest) error {
	userID, workspaceID, queryScope, err := s.resolveProfessionalWorkspace(req)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.professionalUserIDConfig = userID
	s.professionalWorkspaceConfig = workspaceID
	s.professionalQueryScopeConfig = queryScope
	return nil
}

func (s *Server) professionalWorkspaceRuntime(override ProfessionalWorkspaceSelectionRequest) (ProfessionalWorkspaceRuntime, error) {
	userID, workspaceID, queryScope, err := s.resolveProfessionalWorkspace(override)
	if err != nil {
		return ProfessionalWorkspaceRuntime{}, err
	}
	sourceSummary := s.workspaceSourceSummaryForWorkspace(userID, workspaceID)
	deviceBindingSummary := s.workspaceDeviceBindingSummaryForWorkspace(userID, workspaceID)
	queryScopeReadiness := workspaceQueryScopeReadiness(queryScope, sourceSummary)
	workspaceStatus := "contract_ready"
	if sourceSummary.TotalSources > 0 {
		workspaceStatus = queryScopeReadiness
	}
	return ProfessionalWorkspaceRuntime{
		SchemaVersion:                      ProfessionalWorkspaceRuntimeSchemaVersion,
		Mode:                               VoiceModeProfessional,
		UserID:                             userID,
		WorkspaceID:                        workspaceID,
		QueryScope:                         queryScope,
		PrivacyScope:                       "professional_only",
		AdapterContractVersion:             ProfessionalAdapterContractVersion,
		WorkspaceStatus:                    workspaceStatus,
		QueryScopeReadiness:                queryScopeReadiness,
		SourceScopeCounts:                  copyWorkspaceSourceCounts(sourceSummary.SourceScopeCounts),
		IndexingRequestedSourceScopeCounts: copyWorkspaceSourceCounts(sourceSummary.IndexingRequestedSourceScopeCounts),
		SearchableSourceScopeCounts:        copyWorkspaceSourceCounts(sourceSummary.SearchableSourceScopeCounts),
		UploadAPIReady:                     sourceSummary.TotalSources > 0,
		IndexingAPIReady:                   sourceSummary.IndexingRequestedSources > 0,
		QueryScopeReady:                    true,
		V21ExecutionAllowed:                false,
		DeviceBindingPolicy:                deviceBindingSummary.DeviceBindingPolicy,
		BoundDeviceCount:                   deviceBindingSummary.TotalBindings,
		ActiveDeviceBindingCount:           deviceBindingSummary.ActiveBindings,
		RevokedDeviceBindingCount:          deviceBindingSummary.RevokedBindings,
		DeviceBindingSummary:               deviceBindingSummary,
	}, nil
}

func (s *Server) resolveProfessionalWorkspace(override ProfessionalWorkspaceSelectionRequest) (string, string, string, error) {
	s.mu.Lock()
	userID, workspaceID, queryScope, err := s.resolveProfessionalWorkspaceNoLock(override.UserID, override.WorkspaceID, override.QueryScope)
	s.mu.Unlock()
	return userID, workspaceID, queryScope, err
}

func (s *Server) resolveProfessionalWorkspaceNoLock(userOverride string, workspaceOverride string, queryScopeOverride string) (string, string, string, error) {
	userID := defaultProfessionalLabel(s.professionalUserIDConfig, v21adapter.DefaultUserID)
	workspaceID := defaultProfessionalLabel(s.professionalWorkspaceConfig, v21adapter.DefaultWorkspaceID)
	queryScope := defaultProfessionalQueryScope(s.professionalQueryScopeConfig)
	if strings.TrimSpace(userOverride) != "" {
		userID = strings.ToLower(strings.TrimSpace(userOverride))
		if !validProfessionalLabel(userID) {
			return "", "", "", fmt.Errorf("valid redacted user_id is required")
		}
	}
	if strings.TrimSpace(workspaceOverride) != "" {
		workspaceID = strings.ToLower(strings.TrimSpace(workspaceOverride))
		if !validProfessionalLabel(workspaceID) {
			return "", "", "", fmt.Errorf("valid redacted workspace_id is required")
		}
	}
	if strings.TrimSpace(queryScopeOverride) != "" {
		queryScope = strings.ToLower(strings.TrimSpace(queryScopeOverride))
		if !v21adapter.ValidQueryScope(queryScope) {
			return "", "", "", fmt.Errorf("query_scope must be public_only, personal_only, or personal_plus_public")
		}
	}
	return userID, workspaceID, queryScope, nil
}

func (s *Server) selectedProfessionalWorkspace() ProfessionalWorkspaceRuntime {
	runtime, err := s.professionalWorkspaceRuntime(ProfessionalWorkspaceSelectionRequest{})
	if err != nil {
		return ProfessionalWorkspaceRuntime{
			SchemaVersion:          ProfessionalWorkspaceRuntimeSchemaVersion,
			Mode:                   VoiceModeProfessional,
			UserID:                 v21adapter.DefaultUserID,
			WorkspaceID:            v21adapter.DefaultWorkspaceID,
			QueryScope:             v21adapter.QueryScopePublic,
			PrivacyScope:           "professional_only",
			AdapterContractVersion: ProfessionalAdapterContractVersion,
			WorkspaceStatus:        "contract_ready",
			QueryScopeReady:        true,
		}
	}
	return runtime
}

func defaultProfessionalLabel(value string, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if validProfessionalLabel(value) {
		return value
	}
	return fallback
}

func validProfessionalLabel(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	return v21adapter.ValidateProfessionalQueryRequest(v21adapter.QueryRequest{
		Utterance:    "a21 validation",
		Mode:         VoiceModeProfessional,
		PrivacyScope: "professional_only",
		UserID:       value,
		WorkspaceID:  value,
	}) == nil
}

func defaultProfessionalQueryScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if v21adapter.ValidQueryScope(scope) {
		return scope
	}
	return v21adapter.QueryScopePublic
}

func professionalQueryScopeOptions(selected string) []ProfessionalWorkspaceOption {
	selected = defaultProfessionalQueryScope(selected)
	options := []ProfessionalWorkspaceOption{
		{ID: v21adapter.QueryScopePublic, Label: "Public only", Status: "available", Description: "Use only public A21/V21 resources"},
		{ID: v21adapter.QueryScopePersonal, Label: "Personal only", Status: "available", Description: "Contract scope for the signed-in user's uploaded workspace; upload/index readiness remains separate"},
		{ID: v21adapter.QueryScopeCombined, Label: "Personal plus public", Status: "available", Description: "Contract scope combining personal workspace and public resources; upload/index readiness remains separate"},
	}
	for i := range options {
		options[i].Default = options[i].ID == selected
	}
	return options
}
