package gateway

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"a21.local/a21/internal/providers"
)

func safeOptionalWorkspaceLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	if validProfessionalLabel(label) {
		return label
	}
	return ""
}

func workspaceDocumentStoreDir(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	if envDir := strings.TrimSpace(os.Getenv("A21_WORKSPACE_DOCUMENT_STORE_DIR")); envDir != "" {
		return envDir
	}
	return filepath.Join(".a21-run", "gateway", "workspace-documents")
}

func workspaceUploadJobRedaction() WorkspaceUploadJobRedaction {
	return WorkspaceUploadJobRedaction{
		DocumentTextStored:    false,
		DocumentBytesStored:   false,
		Base64PayloadStored:   false,
		ImportURLStored:       false,
		LocalPathStored:       false,
		CredentialValueStored: false,
		ProviderOutputStored:  false,
	}
}

func workspaceUploadJobStoredDocumentRedaction() WorkspaceUploadJobRedaction {
	redaction := workspaceUploadJobRedaction()
	redaction.DocumentBytesStored = true
	return redaction
}

func workspaceUploadJobRedactionForJobs(jobs []WorkspaceUploadJob) WorkspaceUploadJobRedaction {
	redaction := workspaceUploadJobRedaction()
	for _, job := range jobs {
		redaction = mergeWorkspaceUploadJobRedaction(redaction, job.Redaction)
	}
	return redaction
}

func workspaceUploadJobRedactionForSources(sources []WorkspaceSource) WorkspaceUploadJobRedaction {
	redaction := workspaceUploadJobRedaction()
	for _, source := range sources {
		redaction = mergeWorkspaceUploadJobRedaction(redaction, source.Redaction)
	}
	return redaction
}

func workspaceUploadJobRedactionForIndexJobs(jobs []WorkspaceIndexJob) WorkspaceUploadJobRedaction {
	redaction := workspaceUploadJobRedaction()
	for _, job := range jobs {
		redaction = mergeWorkspaceUploadJobRedaction(redaction, job.Redaction)
	}
	return redaction
}

func mergeWorkspaceUploadJobRedaction(base WorkspaceUploadJobRedaction, next WorkspaceUploadJobRedaction) WorkspaceUploadJobRedaction {
	base.DocumentTextStored = base.DocumentTextStored || next.DocumentTextStored
	base.DocumentBytesStored = base.DocumentBytesStored || next.DocumentBytesStored
	base.Base64PayloadStored = base.Base64PayloadStored || next.Base64PayloadStored
	base.ImportURLStored = base.ImportURLStored || next.ImportURLStored
	base.LocalPathStored = base.LocalPathStored || next.LocalPathStored
	base.CredentialValueStored = base.CredentialValueStored || next.CredentialValueStored
	base.ProviderOutputStored = base.ProviderOutputStored || next.ProviderOutputStored
	return base
}

func workspaceDocumentStoredRedaction() WorkspaceDocumentRedaction {
	return WorkspaceDocumentRedaction{
		DocumentTextStored:     false,
		DocumentBytesStored:    true,
		DocumentBytesReturned:  false,
		OriginalFilenameStored: false,
		Base64PayloadStored:    false,
		ImportURLStored:        false,
		LocalPathStored:        false,
		LocalPathReturned:      false,
		CredentialValueStored:  false,
		ProviderOutputStored:   false,
	}
}

func (s *Server) currentVoiceProvider() providers.VoiceProvider {
	if s == nil {
		return providers.NewMockVoiceProvider()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.voice == nil {
		return providers.NewMockVoiceProvider()
	}
	return s.voice
}

func (s *Server) currentXiaozhiVoicePipelineASR() providers.ASRAdapter {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.xiaozhiVoicePipelineASR
}

func (s *Server) currentXiaozhiVoicePipelineRunnerFactory() func() xiaozhiVoicePipelineRunner {
	if s == nil {
		return defaultXiaozhiVoicePipelineRunner
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.xiaozhiVoicePipelineRunner == nil {
		return defaultXiaozhiVoicePipelineRunner
	}
	return s.xiaozhiVoicePipelineRunner
}

func (s *Server) currentXiaozhiVoicePipelineMeta() xiaozhiVoicePipelineMeta {
	if s == nil {
		return xiaozhiVoicePipelineMeta{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.xiaozhiVoicePipelineMeta
}

func (s *Server) voiceChainProfileCatalog() VoiceChainProfilesResponse {
	s.mu.Lock()
	mode := defaultVoiceChainMode(s.voiceChainModeConfig)
	asrProfile := defaultCascadeASRProfile(s.cascadeASRProfileConfig)
	llmProfile := defaultCascadeLLMProfile(s.cascadeLLMProfileConfig)
	ttsProfile := defaultFixedTTSProfile(s.fixedTTSProfileConfig)
	realtimeProvider := defaultRealtimeProvider(s.realtimeProviderConfig)
	voiceCloneProfile := defaultVoiceCloneProfile(s.voiceCloneProfileConfig)
	selectedTTSProfile := voiceChainTTSForVoice(ttsProfile, voiceCloneProfile)
	s.mu.Unlock()
	findings := []string{}
	if llmProfile == "deepseek" {
		findings = append(findings, "stepfun_not_selected")
	}
	return VoiceChainProfilesResponse{
		SchemaVersion:             VoiceChainProfileSchemaVersion,
		Service:                   DeviceRegistryServiceName,
		SelectedVoiceChainMode:    mode,
		SelectedASRProfile:        asrProfile,
		SelectedLLMProfile:        llmProfile,
		FixedTTSProfile:           ttsProfile,
		SelectedTTSProfile:        selectedTTSProfile,
		SelectedRealtimeProvider:  realtimeProvider,
		SelectedVoiceCloneProfile: voiceCloneProfile,
		Cascade: VoiceChainCascadeCatalog{
			ASRProfiles: voiceChainASRProfileOptions(asrProfile),
			LLMProfiles: voiceChainLLMProfileOptions(llmProfile),
			TTSProfile: VoiceChainProfileOption{
				ID:      ttsProfile,
				Label:   voiceChainTTSLabel(ttsProfile),
				Status:  "fixed",
				Default: true,
				Reason:  "A21 keeps TTS fixed in cascade mode so ASR/LLM latency tests do not also change voice quality",
			},
		},
		Realtime:  VoiceChainRealtimeCatalog{Providers: voiceChainRealtimeProviderOptions(realtimeProvider)},
		Voices:    voiceChainVoiceOptions(voiceCloneProfile),
		HotSwitch: true,
		Findings:  findings,
	}
}

func (s *Server) setVoiceChainProfile(req VoiceChainProfileSelectionRequest) error {
	s.mu.Lock()
	mode := defaultVoiceChainMode(s.voiceChainModeConfig)
	asrProfile := defaultCascadeASRProfile(s.cascadeASRProfileConfig)
	llmProfile := defaultCascadeLLMProfile(s.cascadeLLMProfileConfig)
	ttsProfile := defaultFixedTTSProfile(s.fixedTTSProfileConfig)
	realtimeProvider := defaultRealtimeProvider(s.realtimeProviderConfig)
	voiceCloneProfile := defaultVoiceCloneProfile(s.voiceCloneProfileConfig)
	env := append([]string(nil), s.cloudVoiceEnv...)
	s.mu.Unlock()

	if strings.TrimSpace(req.VoiceChainMode) != "" {
		mode = strings.ToLower(strings.TrimSpace(req.VoiceChainMode))
		if !validVoiceChainMode(mode) {
			return fmt.Errorf("valid voice_chain_mode is required")
		}
	}
	if strings.TrimSpace(req.ASRProfile) != "" {
		asrProfile = strings.ToLower(strings.TrimSpace(req.ASRProfile))
		if !validCascadeASRProfile(asrProfile) {
			return fmt.Errorf("valid asr_profile is required")
		}
	}
	if strings.TrimSpace(req.LLMProfile) != "" {
		llmProfile = strings.ToLower(strings.TrimSpace(req.LLMProfile))
		if !validCascadeLLMProfile(llmProfile) {
			return fmt.Errorf("valid llm_profile is required")
		}
	}
	if strings.TrimSpace(req.RealtimeProvider) != "" {
		realtimeProvider = strings.ToLower(strings.TrimSpace(req.RealtimeProvider))
		if !validRealtimeProvider(realtimeProvider) {
			return fmt.Errorf("valid realtime_provider is required")
		}
	}
	if strings.TrimSpace(req.VoiceCloneProfile) != "" {
		voiceCloneProfile = strings.ToLower(strings.TrimSpace(req.VoiceCloneProfile))
		if !validVoiceCloneProfile(voiceCloneProfile) {
			return fmt.Errorf("valid voice_clone_profile is required")
		}
	}
	env = voiceChainEnvWithSelection(env, mode, asrProfile, llmProfile, ttsProfile, realtimeProvider, voiceCloneProfile)
	adapters := providers.VoicePipelineAdaptersFromEnv(env)
	voiceProvider := providers.NewGatewayVoiceProviderFromEnv(env)
	runnerFactory := func() xiaozhiVoicePipelineRunner {
		return providers.NewVoicePipelineRunner(adapters)
	}
	meta := xiaozhiVoicePipelineMeta{Selection: adapters.Selection, ExecutionMode: adapters.ExecutionMode}
	if isZeroGatewayVoicePipelineSelection(meta.Selection) {
		meta.Selection = providers.VoicePipelineSelectionFromEnv(env)
	}
	if strings.TrimSpace(meta.ExecutionMode) == "" {
		meta.ExecutionMode = "fixture"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.voiceChainModeConfig = mode
	s.cascadeASRProfileConfig = asrProfile
	s.cascadeLLMProfileConfig = llmProfile
	s.fixedTTSProfileConfig = ttsProfile
	s.realtimeProviderConfig = realtimeProvider
	s.voiceCloneProfileConfig = voiceCloneProfile
	s.cloudVoiceEnv = env
	s.voice = voiceProvider
	s.xiaozhiVoicePipelineRunner = runnerFactory
	s.xiaozhiVoicePipelineMeta = meta
	if adapters.ASR != nil {
		s.xiaozhiVoicePipelineASR = adapters.ASR
		s.xiaozhiProfessionalASR = adapters.ASR
	}
	if adapters.TTS != nil {
		s.xiaozhiFastAckTTS = adapters.TTS
	}
	return nil
}

func defaultVoiceChainMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if validVoiceChainMode(mode) {
		return mode
	}
	return VoiceChainModeCascade
}

func validVoiceChainMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case VoiceChainModeCascade, VoiceChainModeRealtime:
		return true
	default:
		return false
	}
}

func defaultCascadeASRProfile(profile string) string {
	profile = strings.ToLower(strings.TrimSpace(profile))
	if validCascadeASRProfile(profile) {
		return profile
	}
	return DefaultCascadeASRProfile
}

func validCascadeASRProfile(profile string) bool {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "dashscope_qwen_asr_realtime", "doubao_asr_realtime", "sherpa_onnx_streaming":
		return true
	default:
		return false
	}
}

func defaultCascadeLLMProfile(profile string) string {
	profile = strings.ToLower(strings.TrimSpace(profile))
	if validCascadeLLMProfile(profile) {
		return profile
	}
	return DefaultCascadeLLMProfile
}

func validCascadeLLMProfile(profile string) bool {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "stepfun", "bailian_dashscope", "siliconflow", "deepseek", "local_ollama":
		return true
	default:
		return false
	}
}

func defaultFixedTTSProfile(profile string) string {
	profile = strings.ToLower(strings.TrimSpace(profile))
	if profile == "dashscope_qwen_tts_realtime" || profile == "doubao_tts_realtime" || profile == "voice_clone_cli" {
		return profile
	}
	return DefaultFixedTTSProfile
}

func defaultRealtimeProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if validRealtimeProvider(provider) {
		return provider
	}
	return DefaultRealtimeProvider
}

func validRealtimeProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "doubao_realtime", "openai_realtime", "doubao_tts_realtime":
		return true
	default:
		return false
	}
}

func defaultVoiceCloneProfile(profile string) string {
	profile = strings.ToLower(strings.TrimSpace(profile))
	if validVoiceCloneProfile(profile) {
		return profile
	}
	return DefaultVoiceCloneProfile
}

func (s *Server) currentVoiceCloneProfile() string {
	if s == nil {
		return DefaultVoiceCloneProfile
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return defaultVoiceCloneProfile(s.voiceCloneProfileConfig)
}

func voiceCloneProfileUsesClone(profile string) bool {
	return defaultVoiceCloneProfile(profile) != DefaultVoiceCloneProfile
}

func validVoiceCloneProfile(profile string) bool {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "a21_voice_default_dashscope", "a21_voice_clone_default", "a21_voice_clone_cosyvoice", "a21_voice_clone_minimax":
		return true
	default:
		return false
	}
}

func voiceChainASRProfileOptions(selected string) []VoiceChainProfileOption {
	return []VoiceChainProfileOption{
		{ID: "dashscope_qwen_asr_realtime", Label: "Qwen ASR realtime", Status: "available", Default: selected == "dashscope_qwen_asr_realtime", Recommended: true, Reason: "public cloud edge default"},
		{ID: "doubao_asr_realtime", Label: "Doubao ASR realtime", Status: "available", Default: selected == "doubao_asr_realtime"},
		{ID: "sherpa_onnx_streaming", Label: "Sherpa streaming local", Status: "mac_local", Default: selected == "sherpa_onnx_streaming"},
	}
}

func voiceChainLLMProfileOptions(selected string) []VoiceChainProfileOption {
	return []VoiceChainProfileOption{
		{ID: "stepfun", Label: "StepFun 8k fast", Status: "recommended", Default: selected == "stepfun", Recommended: true, Reason: "fastest validated short-answer LLM when credentials are present"},
		{ID: "bailian_dashscope", Label: "DashScope Qwen flash", Status: "available", Default: selected == "bailian_dashscope"},
		{ID: "siliconflow", Label: "SiliconFlow Qwen", Status: "available", Default: selected == "siliconflow"},
		{ID: "deepseek", Label: "DeepSeek fallback", Status: "fallback", Default: selected == "deepseek"},
		{ID: "local_ollama", Label: "Local Ollama", Status: "mac_local", Default: selected == "local_ollama"},
	}
}

func voiceChainRealtimeProviderOptions(selected string) []VoiceChainProfileOption {
	return []VoiceChainProfileOption{
		{ID: "doubao_realtime", Label: "Doubao speech-to-speech realtime", Status: "available", Default: selected == "doubao_realtime"},
		{ID: "openai_realtime", Label: "OpenAI realtime", Status: "available", Default: selected == "openai_realtime"},
		{ID: "doubao_tts_realtime", Label: "Doubao realtime TTS bridge", Status: "available", Default: selected == "doubao_tts_realtime"},
		{ID: "dashscope_qwen_omni_realtime", Label: "Qwen Omni realtime", Status: "planned", Default: selected == "dashscope_qwen_omni_realtime"},
	}
}

func voiceChainVoiceOptions(selected string) []VoiceChainVoiceOption {
	return []VoiceChainVoiceOption{
		{ID: "a21_voice_default_dashscope", Label: "紫悦自然声音", Status: "default", ProviderProfile: "a21_bailian_qwen_tts_realtime", TTSProfile: "dashscope_qwen_tts_realtime", Default: selected == "a21_voice_default_dashscope"},
		{ID: "a21_voice_clone_default", Label: "紫悦克隆声音", Status: "available", ProviderProfile: "voice_clone_cli", TTSProfile: "voice_clone_cli", VoiceClone: true, Default: selected == "a21_voice_clone_default"},
		{ID: "a21_voice_clone_cosyvoice", Label: "CosyVoice clone", Status: "planned", ProviderProfile: "a21_bailian_cosyvoice_clone_tts", TTSProfile: "voice_clone_cli", VoiceClone: true, Default: selected == "a21_voice_clone_cosyvoice"},
		{ID: "a21_voice_clone_minimax", Label: "MiniMax clone", Status: "planned", ProviderProfile: "a21_minimax_voice_clone_tts", TTSProfile: "voice_clone_cli", VoiceClone: true, Default: selected == "a21_voice_clone_minimax"},
	}
}

func voiceChainTTSLabel(profile string) string {
	switch profile {
	case "dashscope_qwen_tts_realtime":
		return "紫悦自然声音"
	case "doubao_tts_realtime":
		return "Doubao realtime voice"
	case "voice_clone_cli":
		return "紫悦克隆声音"
	default:
		return profile
	}
}

func voiceChainEnvWithSelection(env []string, mode string, asrProfile string, llmProfile string, ttsProfile string, realtimeProvider string, voiceCloneProfile string) []string {
	out := append([]string(nil), env...)
	out = upsertEnv(out, "A21_VOICE_CHAIN_MODE", mode)
	out = upsertEnv(out, "A21_GATEWAY_VOICE_PROVIDER", "selected")
	out = upsertEnv(out, "A21_ASR_PROFILE", voiceChainASRMode(asrProfile))
	out = upsertEnv(out, "A21_ASR_CLOUD_PROFILE", asrProfile)
	if asrProfile == "sherpa_onnx_streaming" {
		out = upsertEnv(out, "A21_ASR_LOCAL_PROFILE", asrProfile)
	}
	out = upsertEnv(out, "A21_TEXT_STREAM_PROFILE", llmProfile)
	out = upsertEnv(out, "A21_TTS_FAST_PROFILE", voiceChainTTSForVoice(ttsProfile, voiceCloneProfile))
	out = upsertEnv(out, "A21_PROVIDER_PRIMARY", realtimeProvider)
	out = upsertEnv(out, "A21_VOICE_CLONE_PROFILE", voiceCloneProfile)
	return out
}

func voiceChainASRMode(profile string) string {
	if profile == "sherpa_onnx_streaming" {
		return "local"
	}
	return "cloud"
}

func voiceChainTTSForVoice(ttsProfile string, voiceCloneProfile string) string {
	if voiceCloneProfile != "" && voiceCloneProfile != "a21_voice_default_dashscope" {
		return "voice_clone_cli"
	}
	return defaultFixedTTSProfile(ttsProfile)
}

func upsertEnv(env []string, key string, value string) []string {
	key = strings.TrimSpace(key)
	if key == "" {
		return env
	}
	prefix := key + "="
	for i, item := range env {
		if strings.HasPrefix(item, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func (s *Server) gatewayProfileCatalog(r *http.Request) GatewayProfileCatalogResponse {
	publicURL := s.publicGatewayWebSocketURL()
	publicStatus := "needs_config"
	if publicURL != "" {
		publicStatus = "available"
	}
	selected := s.selectedGatewayProfile()
	return GatewayProfileCatalogResponse{
		SchemaVersion:          GatewayProfileSchemaVersion,
		Service:                DeviceRegistryServiceName,
		SelectedGatewayProfile: selected,
		Profiles: []GatewayProfileOption{
			{
				ID:           GatewayProfileMacLocal,
				Label:        "Mac Local Gateway",
				Status:       "available",
				Default:      selected == GatewayProfileMacLocal,
				WebSocketURL: s.localGatewayWebSocketURL(r),
				Description:  "selectable low-latency Mac Gateway for local models and local processing",
			},
			{
				ID:           GatewayProfilePublicWSS,
				Label:        "Public WSS Gateway",
				Status:       publicStatus,
				Default:      selected == GatewayProfilePublicWSS,
				WebSocketURL: publicURL,
				Description:  "main product Gateway for public 443/wss StackChan relay",
			},
		},
	}
}

func validGatewayProfile(profile string) bool {
	switch profile {
	case GatewayProfileMacLocal, GatewayProfilePublicWSS:
		return true
	default:
		return false
	}
}

func (s *Server) selectedGatewayProfile() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return defaultGatewayProfile(s.gatewayProfileConfig, s.publicGatewayURL)
}

func (s *Server) setGatewayProfile(profile string) {
	if !validGatewayProfile(profile) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gatewayProfileConfig = profile
}

func (s *Server) cloudVoiceProfileCatalog() CloudVoiceProfileCatalogResponse {
	s.mu.Lock()
	selected := s.cloudVoiceProfileConfig
	env := append([]string(nil), s.cloudVoiceEnv...)
	s.mu.Unlock()
	catalog := providers.CloudVoiceCatalogFromEnv(env, selected)
	return CloudVoiceProfileCatalogResponse{
		SchemaVersion:             CloudVoiceProfileSchemaVersion,
		Service:                   DeviceRegistryServiceName,
		SelectedCloudVoiceProfile: catalog.SelectedCloudVoiceProfile,
		Profiles:                  catalog.Profiles,
		Findings:                  catalog.Findings,
	}
}

func (s *Server) selectedCloudVoiceProfile() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return providers.DefaultCloudVoiceProfile(s.cloudVoiceProfileConfig)
}

func (s *Server) setCloudVoiceProfile(profile string) {
	if !providers.ValidCloudVoiceProfile(profile) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cloudVoiceProfileConfig = strings.ToLower(strings.TrimSpace(profile))
}

func defaultGatewayProfile(profile string, publicGatewayURL string) string {
	if profile == GatewayProfileMacLocal {
		return GatewayProfileMacLocal
	}
	if profile == GatewayProfilePublicWSS && gatewayPublicWebSocketURL(publicGatewayURL) != "" {
		return GatewayProfilePublicWSS
	}
	if gatewayPublicWebSocketURL(publicGatewayURL) != "" {
		return GatewayProfilePublicWSS
	}
	return GatewayProfileMacLocal
}

func (s *Server) localGatewayWebSocketURL(r *http.Request) string {
	s.mu.Lock()
	configured := s.macLocalGatewayURL
	s.mu.Unlock()
	if configuredURL := gatewayConfiguredWebSocketURL(configured); configuredURL != "" {
		return configuredURL
	}
	host := sanitizedXiaozhiOTAHost(r.Host)
	if host == "" {
		return ""
	}
	return xiaozhiOTAWebSocketScheme(r) + "://" + host + "/v1/xiaozhi"
}

func (s *Server) selectedGatewayWebSocketURL(r *http.Request) string {
	if s.selectedGatewayProfile() == GatewayProfilePublicWSS {
		if publicURL := s.publicGatewayWebSocketURL(); publicURL != "" {
			return publicURL
		}
	}
	return s.localGatewayWebSocketURL(r)
}

func (s *Server) publicGatewayWebSocketURL() string {
	s.mu.Lock()
	raw := s.publicGatewayURL
	s.mu.Unlock()
	return gatewayConfiguredWebSocketURL(raw)
}

func gatewayPublicWebSocketURL(raw string) string {
	return gatewayConfiguredWebSocketURL(raw)
}

func gatewayConfiguredWebSocketURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" ||
		strings.ContainsAny(raw, " \t\r\n") ||
		strings.Contains(strings.ToLower(raw), "token") ||
		strings.Contains(strings.ToLower(raw), "secret") {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil || parsed.Host == "" || parsed.User != nil {
		return ""
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return ""
	}
	if sanitizedXiaozhiOTAHost(parsed.Host) == "" {
		return ""
	}
	scheme := ""
	switch strings.ToLower(parsed.Scheme) {
	case "http", "ws":
		scheme = "ws"
	case "https", "wss":
		scheme = "wss"
	default:
		return ""
	}
	endpointPath := strings.TrimRight(parsed.EscapedPath(), "/")
	switch endpointPath {
	case "", "/v1/xiaozhi":
		endpointPath = "/v1/xiaozhi"
	default:
		return ""
	}
	return scheme + "://" + parsed.Host + endpointPath
}

func (s *Server) handleXiaozhiOTA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	webSocketURL := s.selectedGatewayWebSocketURL(r)
	if webSocketURL == "" {
		http.Error(w, "invalid host", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, XiaozhiOTAResponse{
		ServerTime: XiaozhiOTAServerTime{
			Timestamp:      s.now().UnixMilli(),
			TimezoneOffset: 8 * 60,
		},
		WebSocket: XiaozhiOTAWebSocketConfig{
			URL:     webSocketURL,
			Token:   "",
			Version: xiaozhiOTAWebSocketVersion,
		},
	})
}

func xiaozhiOTAWebSocketScheme(r *http.Request) string {
	if r.TLS != nil {
		return "wss"
	}
	forwardedProto := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]))
	if forwardedProto == "https" {
		return "wss"
	}
	return "ws"
}
