package app

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/providers"
)

type productRoleplayVoiceReportFixture struct {
	SchemaVersion    string                              `json:"schema_version"`
	GeneratedAtMS    *int64                              `json:"generated_at_ms"`
	Status           string                              `json:"status"`
	Mode             string                              `json:"mode"`
	Route            string                              `json:"route"`
	ExecutionMode    string                              `json:"execution_mode"`
	DeviceID         string                              `json:"device_id"`
	TraceID          string                              `json:"trace_id"`
	SessionID        string                              `json:"session_id"`
	Runtime          gateway.RoleplayRuntimeSummary      `json:"runtime"`
	VoicePipeline    productRoleplayVoicePipelineFixture `json:"voice_pipeline"`
	TraceMarkers     []string                            `json:"trace_markers"`
	Redaction        productRoleplayVoiceRedaction       `json:"redaction"`
	PhysicalAccepted bool                                `json:"physical_accepted"`
	PRDAccepted      bool                                `json:"prd_accepted"`
}

type productRoleplayVoicePipelineFixture struct {
	Observed                bool     `json:"observed"`
	Status                  string   `json:"status"`
	TextStreamProvider      string   `json:"text_stream_provider"`
	ProviderFamily          string   `json:"provider_family"`
	TextStreamExecuted      bool     `json:"text_stream_executed"`
	PromptInputUsed         bool     `json:"prompt_input_used"`
	VoiceCloneProfileUsed   bool     `json:"voice_clone_profile_used"`
	AudioDownlinkFirstFrame bool     `json:"audio_downlink_first_frame_observed"`
	DevicePlaybackStart     bool     `json:"device_playback_start_observed"`
	AudioChunkCount         int      `json:"audio_chunk_count"`
	Findings                []string `json:"findings,omitempty"`
}

type productRoleplayVoiceRedaction struct {
	PromptStored           bool `json:"prompt_stored"`
	MemoryTextStored       bool `json:"memory_text_stored"`
	ASRTextStored          bool `json:"asr_text_stored"`
	ProviderOutputStored   bool `json:"provider_output_stored"`
	AudioStored            bool `json:"audio_stored"`
	VoiceCloneSampleStored bool `json:"voice_clone_sample_stored"`
	FullURLsStored         bool `json:"full_urls_stored"`
	LocalPathsStored       bool `json:"local_paths_stored"`
	CredentialValuesStored bool `json:"credential_values_stored"`
}

func loadProductRoleplayVoiceReportEvidence(path string) (productRoleplayVoiceReportEvidence, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productRoleplayVoiceReportEvidence{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productRoleplayVoiceReportEvidence{}, []productReadinessFinding{invalidProductRoleplayVoiceReportFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productRoleplayVoiceReportEvidence{}, []productReadinessFinding{invalidProductRoleplayVoiceReportFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productRoleplayVoiceReportEvidence{}, []productReadinessFinding{invalidProductRoleplayVoiceReportFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productRoleplayVoiceReportEvidence{}, []productReadinessFinding{invalidProductRoleplayVoiceReportFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productRoleplayVoiceReportContainsForbiddenValue(raw) {
		return productRoleplayVoiceReportEvidence{}, []productReadinessFinding{invalidProductRoleplayVoiceReportFinding()}
	}
	var fixture productRoleplayVoiceReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productRoleplayVoiceReportEvidence{}, []productReadinessFinding{invalidProductRoleplayVoiceReportFinding()}
	}
	if missingField := missingProductRoleplayVoiceReportField(fixture); missingField != "" {
		return productRoleplayVoiceReportEvidence{}, []productReadinessFinding{missingProductRoleplayVoiceReportFieldFinding(missingField)}
	}
	evidence, ok := productRoleplayVoiceReportEvidenceFromFixture(path, fixture)
	if !ok {
		return productRoleplayVoiceReportEvidence{}, []productReadinessFinding{invalidProductRoleplayVoiceReportFinding()}
	}
	return evidence, nil
}

func productRoleplayVoiceReportEvidenceFromFixture(path string, fixture productRoleplayVoiceReportFixture) (productRoleplayVoiceReportEvidence, bool) {
	mode := strings.TrimSpace(fixture.Mode)
	route := strings.TrimSpace(fixture.Route)
	executionMode := strings.TrimSpace(fixture.ExecutionMode)
	runtime := fixture.Runtime
	pipeline := fixture.VoicePipeline
	profile := strings.TrimSpace(runtime.RoleplayProfile)
	scenario := strings.TrimSpace(runtime.Scenario)
	voiceClone := strings.TrimSpace(runtime.VoiceCloneProfile)
	if fixture.SchemaVersion != productRoleplayVoiceReportSchemaVersion ||
		fixture.GeneratedAtMS == nil ||
		*fixture.GeneratedAtMS <= 0 ||
		!productRoleplayVoiceStatusSafe(fixture.Status) ||
		mode != "roleplay" ||
		!productRoleplayVoiceRouteSafe(route) ||
		!productRoleplayVoiceExecutionModeSafe(executionMode) ||
		!productRoleplayRuntimeSafe(runtime) ||
		!productRoleplayVoicePipelineSafe(pipeline) ||
		!productRoleplayVoiceTraceMarkersSafe(fixture.TraceMarkers) ||
		!productRoleplayVoiceRedactionOK(fixture.Redaction) ||
		!productVoiceChainSafeID(profile) ||
		!productVoiceChainSafeID(scenario) ||
		!productVoiceChainSafeID(voiceClone) ||
		!productRoleplayVoiceSafeOptionalID(fixture.DeviceID) ||
		!productRoleplayVoiceSafeOptionalID(fixture.TraceID) ||
		!productRoleplayVoiceSafeOptionalID(fixture.SessionID) ||
		fixture.PRDAccepted {
		return productRoleplayVoiceReportEvidence{}, false
	}
	memoryReady := !runtime.MemoryConfigured || runtime.MemoryPromptInputReady
	runtimeRedactionOK := !runtime.PromptStored &&
		!runtime.MemoryTextStored &&
		!runtime.TranscriptStored &&
		!runtime.ProviderOutputStored &&
		!runtime.VoiceCloneSampleStored &&
		!runtime.ProfessionalRouteAllowed &&
		!runtime.V21Executed
	voiceRuntimeReady := strings.TrimSpace(fixture.Status) == "passed" &&
		pipeline.Observed &&
		pipeline.Status == "pipeline_completed" &&
		pipeline.TextStreamExecuted &&
		pipeline.PromptInputUsed &&
		pipeline.VoiceCloneProfileUsed &&
		pipeline.AudioDownlinkFirstFrame &&
		pipeline.DevicePlaybackStart &&
		pipeline.AudioChunkCount > 0 &&
		runtime.SoulPromptInputReady &&
		runtime.PromptComposed &&
		memoryReady &&
		runtimeRedactionOK &&
		productRoleplayVoiceHasMarker(fixture.TraceMarkers, "roleplay.profile.ready") &&
		productRoleplayVoiceHasMarker(fixture.TraceMarkers, "roleplay.prompt_input.used") &&
		productRoleplayVoiceHasMarker(fixture.TraceMarkers, "roleplay.voice_clone_profile.used") &&
		productRoleplayVoiceHasMarker(fixture.TraceMarkers, "audio.downlink.first_frame") &&
		productRoleplayVoiceHasMarker(fixture.TraceMarkers, "device.playback.start") &&
		!fixture.PhysicalAccepted
	return productRoleplayVoiceReportEvidence{
		Valid:                       true,
		SourceReport:                filepath.Base(filepath.Clean(path)),
		Status:                      strings.TrimSpace(fixture.Status),
		Mode:                        mode,
		Route:                       route,
		ExecutionMode:               executionMode,
		RoleplayProfile:             profile,
		Scenario:                    scenario,
		VoiceCloneProfile:           voiceClone,
		VoiceRuntimeReady:           voiceRuntimeReady,
		TraceMarkerCount:            len(fixture.TraceMarkers),
		TextStreamExecuted:          pipeline.TextStreamExecuted,
		PromptInputUsed:             pipeline.PromptInputUsed,
		VoiceCloneProfileUsed:       pipeline.VoiceCloneProfileUsed,
		AudioDownlinkObserved:       pipeline.AudioDownlinkFirstFrame,
		DevicePlaybackStartObserved: pipeline.DevicePlaybackStart,
		PhysicalAccepted:            fixture.PhysicalAccepted,
		PRDAccepted:                 fixture.PRDAccepted,
	}, true
}

func attachProductRoleplayVoiceEvidence(readiness *productRoleplayReadiness, evidence productRoleplayVoiceReportEvidence) []productReadinessFinding {
	if readiness == nil || !evidence.Valid {
		return nil
	}
	readiness.VoiceRuntimeEvidence = true
	readiness.VoiceRuntimeMatched = false
	readiness.VoiceRuntimeReady = false
	readiness.VoiceRuntimeStatus = evidence.Status
	readiness.VoiceRuntimeSourceReport = evidence.SourceReport
	readiness.VoiceRuntimeRoute = evidence.Route
	readiness.VoiceRuntimeExecutionMode = evidence.ExecutionMode
	readiness.VoiceRuntimeTraceMarkers = evidence.TraceMarkerCount
	readiness.VoiceRuntimeTextExecuted = evidence.TextStreamExecuted
	readiness.VoiceRuntimePromptUsed = evidence.PromptInputUsed
	readiness.VoiceRuntimeVoiceCloneUsed = evidence.VoiceCloneProfileUsed
	readiness.VoiceRuntimeAudioDownlink = evidence.AudioDownlinkObserved
	readiness.VoiceRuntimePlaybackStart = evidence.DevicePlaybackStartObserved
	if !readiness.Available {
		readiness.Findings = appendProductFindingCode(readiness.Findings, "roleplay_profile_unavailable")
		return []productReadinessFinding{{
			Code:    "roleplay_voice_report_mismatch",
			Message: "Roleplay voice runtime report cannot be matched because the Gateway roleplay profile is unavailable",
			Detail:  evidence.SourceReport,
		}}
	}
	if readiness.SelectedRoleplayProfile != evidence.RoleplayProfile ||
		readiness.SelectedScenario != evidence.Scenario ||
		readiness.SelectedVoiceCloneProfile != evidence.VoiceCloneProfile {
		readiness.Findings = appendProductFindingCode(readiness.Findings, "roleplay_voice_report_mismatch")
		return []productReadinessFinding{{
			Code:    "roleplay_voice_report_mismatch",
			Message: "Roleplay voice runtime report does not match the current Gateway roleplay profile",
			Detail:  evidence.SourceReport,
		}}
	}
	readiness.VoiceRuntimeMatched = true
	readiness.VoiceRuntimeReady = evidence.VoiceRuntimeReady
	if !evidence.VoiceRuntimeReady {
		readiness.Findings = appendProductFindingCode(readiness.Findings, "roleplay_voice_runtime_not_ready")
		return []productReadinessFinding{{
			Code:    "roleplay_voice_runtime_not_ready",
			Message: "Roleplay voice runtime evidence is available but not ready",
			Detail:  evidence.SourceReport,
		}}
	}
	return nil
}

func productRoleplayVoiceStatusSafe(status string) bool {
	switch strings.TrimSpace(status) {
	case "passed", "blocked":
		return true
	default:
		return false
	}
}

func productRoleplayVoiceRouteSafe(route string) bool {
	switch strings.TrimSpace(route) {
	case "fast_companion_hybrid", "xiaozhi_roleplay_voice_pipeline":
		return true
	default:
		return false
	}
}

func productRoleplayVoiceExecutionModeSafe(mode string) bool {
	switch strings.TrimSpace(mode) {
	case "host_local", "cloud_edge", "gateway_fast_companion":
		return true
	default:
		return false
	}
}

func productRoleplayVoiceSafeOptionalID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || productVoiceChainSafeID(value) {
		return true
	}
	if len(value) > 96 {
		return false
	}
	hasAlphaNum := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			hasAlphaNum = true
		case r >= 'A' && r <= 'Z':
			hasAlphaNum = true
		case r >= '0' && r <= '9':
			hasAlphaNum = true
		case r == '-' || r == '_' || r == '.' || r == ':':
		default:
			return false
		}
	}
	return hasAlphaNum &&
		!strings.Contains(value, "..") &&
		!strings.Contains(strings.ToLower(value), "://")
}

func productRoleplayVoicePipelineSafe(pipeline productRoleplayVoicePipelineFixture) bool {
	if pipeline.AudioChunkCount < 0 ||
		!productVoiceChainSafeOptionalID(pipeline.TextStreamProvider) ||
		!productVoiceChainSafeOptionalID(pipeline.ProviderFamily) ||
		len(pipeline.Findings) > 12 {
		return false
	}
	for _, finding := range pipeline.Findings {
		if !productVoiceChainSafeID(finding) {
			return false
		}
	}
	switch pipeline.Status {
	case "pipeline_completed", "boundary_ready", "blocked":
	default:
		return false
	}
	return true
}

func productRoleplayVoiceTraceMarkersSafe(markers []string) bool {
	if len(markers) > 64 {
		return false
	}
	for _, marker := range markers {
		if !productRoleplayVoiceMarkerAllowed(marker) {
			return false
		}
	}
	return true
}

func productRoleplayVoiceMarkerAllowed(marker string) bool {
	switch strings.TrimSpace(marker) {
	case "roleplay.profile.ready",
		"roleplay.memory.ready",
		"fast_companion.local_audio.frontend.accepted",
		"asr.first_partial",
		"provider.first_byte",
		"provider.first_content",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"device.playback.start",
		"fast_companion.voice_pipeline.start",
		"roleplay.prompt_input.used",
		"roleplay.voice_clone_profile.used",
		"fast_companion.voice_pipeline.completed",
		"control.listening.sent",
		"control.thinking.sent",
		"control.speaking.sent",
		"audio.playback.chunk.sent":
		return true
	default:
		return false
	}
}

func productRoleplayVoiceHasMarker(markers []string, want string) bool {
	for _, marker := range markers {
		if strings.TrimSpace(marker) == want {
			return true
		}
	}
	return false
}

func productRoleplayVoiceRedactionOK(redaction productRoleplayVoiceRedaction) bool {
	return !redaction.PromptStored &&
		!redaction.MemoryTextStored &&
		!redaction.ASRTextStored &&
		!redaction.ProviderOutputStored &&
		!redaction.AudioStored &&
		!redaction.VoiceCloneSampleStored &&
		!redaction.FullURLsStored &&
		!redaction.LocalPathsStored &&
		!redaction.CredentialValuesStored
}

func productRoleplayVoiceReportContainsForbiddenValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			if productRoleplayVoiceReportContainsForbiddenValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productRoleplayVoiceReportContainsForbiddenValue(child) {
				return true
			}
		}
	case string:
		if productRoleplayVoiceMarkerAllowed(typed) {
			return false
		}
		lower := strings.ToLower(typed)
		if containsLegacyIdentity(typed) {
			return true
		}
		for _, forbidden := range []string{
			"http://",
			"https://",
			"/users/",
			"bearer ",
			"sk-",
			"raw prompt",
			"prompt text",
			"memory text",
			"raw transcript",
			"transcript text",
			"raw provider output",
			"provider output",
			"raw reasoning",
			"reasoning text",
			"raw audio",
			"data_base64",
			"audio_base64",
			"secret-value",
		} {
			if strings.Contains(lower, forbidden) {
				return true
			}
		}
	}
	return false
}

func missingProductRoleplayVoiceReportField(fixture productRoleplayVoiceReportFixture) string {
	switch {
	case strings.TrimSpace(fixture.SchemaVersion) == "":
		return "schema_version"
	case fixture.GeneratedAtMS == nil:
		return "generated_at_ms"
	case strings.TrimSpace(fixture.Status) == "":
		return "status"
	case strings.TrimSpace(fixture.Mode) == "":
		return "mode"
	case strings.TrimSpace(fixture.Route) == "":
		return "route"
	case strings.TrimSpace(fixture.ExecutionMode) == "":
		return "execution_mode"
	case strings.TrimSpace(fixture.Runtime.RoleplayProfile) == "":
		return "runtime.roleplay_profile"
	case strings.TrimSpace(fixture.Runtime.Scenario) == "":
		return "runtime.scenario"
	case strings.TrimSpace(fixture.Runtime.VoiceCloneProfile) == "":
		return "runtime.voice_clone_profile"
	default:
		return ""
	}
}

func missingProductRoleplayVoiceReportFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "roleplay_voice_report_missing_field",
		Message: "Roleplay voice runtime report is missing a required field",
		Detail:  field,
	}
}

func invalidProductRoleplayVoiceReportFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "roleplay_voice_report_invalid",
		Message: "Roleplay voice runtime report is invalid or unsafe",
	}
}

func invalidProductRoleplayProfileFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "roleplay_profile_invalid",
		Message: "A21 Gateway roleplay profile readiness is invalid or unsafe",
	}
}

func buildProductCanonicalReadinessDecision(report productReadinessReport) productCanonicalReadinessDecision {
	status := "blocked"
	switch {
	case report.LaunchReady:
		status = "prd_accepted"
	case report.ServerSide.CandidateReady:
		status = "server_side_candidate_only"
	}
	return productCanonicalReadinessDecision{
		Authority:                  "a21.product_readiness.v1",
		FullPRDStatus:              status,
		LaunchReady:                report.LaunchReady,
		PRDAccepted:                report.LaunchReady,
		ServerSideCandidateReady:   report.ServerSide.CandidateReady,
		HostOnlyEvidenceUse:        "gap_reduction_only",
		RequiresPhysicalAcceptance: !report.StackChan.PhysicalEvidence.PRDPhysicalAccepted,
		MissingRealEvidence:        productMissingRealEvidence(report),
		MissingReportFields:        productMissingReportFields(report.Findings),
	}
}

func productMissingRealEvidence(report productReadinessReport) []string {
	var missing []string
	if !report.Gateway.Healthy {
		missing = append(missing, "gateway")
	}
	if !report.Provider.RealProviderReady || !report.Provider.SmokeEvidenceValid || !report.Provider.SmokeExecuted {
		missing = append(missing, "real_provider_smoke")
	}
	if !productV21ProfessionalReady(report.V21) {
		missing = append(missing, "v21_professional_execution")
	}
	if !productProfessionalRitualReady(report.V21) {
		missing = append(missing, "professional_ritual_execution")
	} else if !productProfessionalReadRecordReady(report.V21) {
		missing = append(missing, "professional_read_record")
	}
	if !report.StackChan.PhysicalDeviceOnline {
		missing = append(missing, "physical_stackchan_online")
	}
	if !report.StackChan.PhysicalEvidence.PRDPhysicalAccepted {
		missing = append(missing, "physical_stackchan_prd_acceptance")
	}
	if !report.Voice.ContinuousVoiceReady {
		missing = append(missing, "continuous_voice_pipeline")
	}
	if !report.Roleplay.VoiceRuntimeReady {
		missing = append(missing, "roleplay_voice_runtime")
	}
	if !report.WakeWord.ProductReady {
		missing = append(missing, "wake_word_product_ready")
	}
	if !productWakeWordPhysicalAccepted(report.WakeWord) {
		missing = append(missing, "wake_word_physical_acceptance")
	}
	if !productVoiceChainLaunchPolicySatisfied(report.Voice.VoiceChain) {
		if productVoiceChainHasFinding(report.Voice.VoiceChain, "stepfun_not_selected") {
			missing = append(missing, "stepfun_not_selected")
		} else if report.Voice.VoiceChain.Available {
			missing = append(missing, "voice_chain_launch_policy")
		} else {
			missing = append(missing, "voice_chain_selector")
		}
	}
	return missing
}

func productMissingReportFields(findings []productReadinessFinding) []string {
	var fields []string
	for _, finding := range findings {
		prefix := productMissingReportFieldPrefix(finding.Code)
		if prefix == "" || strings.TrimSpace(finding.Detail) == "" {
			continue
		}
		fields = append(fields, prefix+":"+strings.TrimSpace(finding.Detail))
	}
	return fields
}

func productMissingReportFieldPrefix(code string) string {
	switch code {
	case "provider_smoke_report_missing_field":
		return "provider_smoke"
	case "provider_realtime_fixture_report_missing_field":
		return "provider_realtime_fixture"
	case "xiaozhi_report_missing_field":
		return "xiaozhi_report"
	case "roleplay_voice_report_missing_field":
		return "roleplay_voice_report"
	case "wake_word_firmware_plan_missing_field":
		return "wake_word_firmware_plan"
	case "wake_word_firmware_package_missing_field":
		return "wake_word_firmware_package"
	case "wake_word_physical_acceptance_missing_field":
		return "wake_word_physical_acceptance"
	case "v21_professional_report_missing_field":
		return "v21_professional_report"
	case "v21_adapter_smoke_report_missing_field":
		return "v21_adapter_smoke"
	default:
		return ""
	}
}

func productMicrophoneReady(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return status == "available" || strings.HasPrefix(status, "available_")
}

func buildProductVoiceReadiness(env []string, provider productProviderReadiness, stackchan productStackChanReadiness, evidenceList ...productXiaozhiReportEvidence) productVoiceReadiness {
	engine := firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_LOCAL_TTS_ENGINE")), "macos_say")
	asrProvider := firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_LOCAL_ASR_PROVIDER")), defaultProductASRProvider(env))
	selection := providers.VoicePipelineSelectionFromEnv(env)
	if strings.TrimSpace(appEnvValue(env, "A21_LOCAL_TTS_ENGINE")) == "" && selection.TTSProfile == "voice_clone_cli" {
		engine = "voice_clone_cli"
	}
	voicePipeline := productVoicePipelineReadiness{
		ASRProfile:                   selection.ASRProfile,
		ASRProfileEnv:                selection.ASRProfileEnv,
		TextStreamProfile:            selection.LLMProfile,
		TextStreamProfileEnv:         selection.LLMProfileEnv,
		TextStreamFallbackProfile:    selection.LLMFallbackProfile,
		TextStreamFallbackProfileEnv: selection.LLMFallbackProfileEnv,
		TextStreamFallbackReady:      productTextStreamProviderReady(env, selection.LLMFallbackProfile),
		TTSProfile:                   selection.TTSProfile,
		TTSProfileEnv:                selection.TTSProfileEnv,
		ExecutionMode:                "env_configured",
	}
	var xiaozhiEvidence productXiaozhiReportEvidence
	if len(evidenceList) > 0 {
		xiaozhiEvidence = evidenceList[0]
	}
	if xiaozhiEvidence.Valid {
		voicePipeline.ExecutionMode = firstNonEmpty(xiaozhiEvidence.Execution.VoicePipelineExecutionMode, voicePipeline.ExecutionMode)
		voicePipeline.AcceptanceStatus = xiaozhiEvidence.AcceptanceStatus
		voicePipeline.AnswerFirstAudioP95MS = xiaozhiEvidence.AnswerFirstAudioP95MS
		voicePipeline.BargeInStopP95MS = xiaozhiEvidence.BargeInStopP95MS
		voicePipeline.FailureCount = xiaozhiEvidence.FailureCount
		voicePipeline.SourceReport = xiaozhiEvidence.SourceReport
		voicePipeline.PRDAccepted = xiaozhiEvidence.PRDAccepted
		if xiaozhiEvidence.Execution.ASRProfile != "" {
			voicePipeline.ASRProfile = xiaozhiEvidence.Execution.ASRProfile
		}
		if xiaozhiEvidence.Execution.ASRProfileEnv != "" {
			voicePipeline.ASRProfileEnv = xiaozhiEvidence.Execution.ASRProfileEnv
		}
		if xiaozhiEvidence.Execution.LLMProfile != "" {
			voicePipeline.TextStreamProfile = xiaozhiEvidence.Execution.LLMProfile
		}
		if xiaozhiEvidence.Execution.LLMProfileEnv != "" {
			voicePipeline.TextStreamProfileEnv = xiaozhiEvidence.Execution.LLMProfileEnv
		}
		if xiaozhiEvidence.Execution.TTSProfile != "" {
			voicePipeline.TTSProfile = xiaozhiEvidence.Execution.TTSProfile
		}
		if xiaozhiEvidence.Execution.TTSProfileEnv != "" {
			voicePipeline.TTSProfileEnv = xiaozhiEvidence.Execution.TTSProfileEnv
		}
		voicePipeline.HostLocalASRReady = xiaozhiEvidence.Execution.HostLocalASRExecuted
		voicePipeline.HostLocalTextReady = xiaozhiEvidence.Execution.HostLocalTextExecuted
		voicePipeline.HostLocalTTSReady = xiaozhiEvidence.Execution.HostLocalTTSExecuted
		hostLoopbackEvidenceAccepted := xiaozhiEvidence.AcceptanceStatus == "candidate_host_only" ||
			xiaozhiEvidence.AcceptanceStatus == "host_local_loopback_passed"
		voicePipeline.HostLoopbackCandidateReady = hostLoopbackEvidenceAccepted &&
			xiaozhiEvidence.AnswerFirstAudioP95MS > 0 &&
			xiaozhiEvidence.AnswerFirstAudioP95MS < 1500 &&
			xiaozhiEvidence.BargeInStopP95MS < 300 &&
			xiaozhiEvidence.FailureCount == 0 &&
			!xiaozhiEvidence.PRDAccepted
		voicePipeline.HostProductChainReady = voicePipeline.HostLoopbackCandidateReady &&
			xiaozhiEvidence.HostProductChainReady
	} else {
		if selection.ASRProfile == "sherpa_onnx" || (selection.ASRProfile == "mock-local-asr" && asrProvider == "sherpa_onnx") {
			voicePipeline.ASRProfile = "sherpa_onnx"
			voicePipeline.ASRProfileEnv = "A21_SHERPA_ONNX_ASR_MODEL_DIR"
			voicePipeline.HostLocalASRReady = productSherpaASRReady(env)
		}
		voicePipeline.HostLocalTextReady = provider.TextStreamReady
		voicePipeline.HostLocalTTSReady =
			((selection.TTSProfile == "sherpa_onnx_tts" || selection.TTSProfile == "sherpa_onnx") && productSherpaTTSReady(env)) ||
				(selection.TTSProfile == "voice_clone_cli" && productVoiceCloneTTSReady(env))
	}
	readiness := productVoiceReadiness{
		LocalTTSEngine: engine,
		ASRProvider:    asrProvider,
		VoicePipeline:  voicePipeline,
	}
	switch engine {
	case "macos_say":
		_, err := exec.LookPath("say")
		readiness.LocalTTSReady = err == nil
	case "sherpa_onnx":
		readiness.LocalTTSReady = productSherpaTTSReady(env)
	case "voice_clone_cli":
		readiness.LocalTTSReady = productVoiceCloneTTSReady(env)
	default:
		readiness.LocalTTSReady = false
	}
	readiness.RealASRReady = asrProvider == "sherpa_onnx" && productSherpaASRReady(env)
	if voicePipeline.HostLocalASRReady && voicePipeline.ASRProfile != "" && voicePipeline.ASRProfile != "mock-local-asr" {
		readiness.RealASRReady = true
		readiness.ASRProvider = voicePipeline.ASRProfile
	}
	if voicePipeline.HostLocalTTSReady && !readiness.LocalTTSReady {
		readiness.LocalTTSReady = true
		readiness.LocalTTSEngine = voicePipeline.TTSProfile
	}
	readiness.ContinuousVoiceReady = readiness.LocalTTSReady &&
		readiness.RealASRReady &&
		provider.RealProviderReady &&
		stackchan.PhysicalDeviceOnline &&
		stackchan.PhysicalMicrophoneReady
	if voicePipeline.HostProductChainReady {
		readiness.ContinuousVoiceReady = true
	}
	return readiness
}

func defaultProductASRProvider(env []string) string {
	if productSherpaASRReady(env) {
		return "sherpa_onnx"
	}
	return "mock_asr"
}

type productXiaozhiReportEvidence struct {
	Valid                 bool
	SourceReport          string
	AcceptanceStatus      string
	PRDAccepted           bool
	AnswerFirstAudioP95MS float64
	BargeInStopP95MS      float64
	FailureCount          int
	HostProductChainReady bool
	Execution             providerLatencyBenchExecution
}
