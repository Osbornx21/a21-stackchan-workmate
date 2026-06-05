package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/personality"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
)

type GatewayProfileSelectionRequest struct {
	GatewayProfile string `json:"gateway_profile"`
}

type CloudVoiceProfileCatalogResponse struct {
	SchemaVersion             string                                      `json:"schema_version"`
	Service                   string                                      `json:"service"`
	SelectedCloudVoiceProfile string                                      `json:"selected_cloud_voice_profile"`
	Profiles                  []providers.CloudVoiceProfileReadiness      `json:"profiles"`
	Findings                  []providers.CloudVoiceProfileCatalogFinding `json:"findings,omitempty"`
}

type CloudVoiceProfileSelectionRequest struct {
	CloudVoiceProfile string `json:"cloud_voice_profile"`
}

type XiaozhiOTAResponse struct {
	ServerTime XiaozhiOTAServerTime      `json:"server_time"`
	WebSocket  XiaozhiOTAWebSocketConfig `json:"websocket"`
	Firmware   map[string]string         `json:"firmware,omitempty"`
	Activation map[string]string         `json:"activation,omitempty"`
}

type XiaozhiOTAServerTime struct {
	Timestamp      int64 `json:"timestamp"`
	TimezoneOffset int   `json:"timezone_offset"`
}

type XiaozhiOTAWebSocketConfig struct {
	URL     string `json:"url"`
	Token   string `json:"token"`
	Version int    `json:"version"`
}

const (
	DeviceRegistrySchemaVersion               = "a21.gateway.devices.v1"
	DeviceRegistryServiceName                 = "a21-gateway"
	HardwareAcceptanceSchemaVersion           = "a21.gateway.hardware_acceptance.v1"
	VoiceModeSchemaVersion                    = "a21.gateway.voice_modes.v1"
	VoiceModeDialogue                         = "dialogue"
	VoiceModeRoleplay                         = "roleplay"
	VoiceModeProfessional                     = "professional"
	RoleplayProfileSchemaVersion              = "a21.gateway.roleplay_profile.v1"
	DefaultRoleplayProfile                    = string(personality.RoleSoulA21Default)
	RoleplayProfileWryPeer                    = string(personality.RoleSoulWryPeer)
	RoleplayProfileCalmAnchor                 = string(personality.RoleSoulCalmAnchor)
	DefaultRoleplayScenario                   = "desk_mouthpiece"
	ProfessionalWorkspaceSchemaVersion        = "a21.gateway.professional_workspace.v1"
	ProfessionalWorkspaceRuntimeSchemaVersion = "a21.professional_workspace_runtime.v1"
	ProfessionalAdapterContractVersion        = "a21.v21_adapter_query.v2"
	ProfessionalReadRecordsSchemaVersion      = "a21.gateway.professional_read_records.v1"
	ProfessionalQuerySchemaVersion            = "a21.gateway.professional_query.v1"
	WorkspaceDocumentsSchemaVersion           = "a21.gateway.workspace_documents.v1"
	WorkspaceUploadJobsSchemaVersion          = "a21.gateway.workspace_upload_jobs.v1"
	WorkspaceIndexJobsSchemaVersion           = "a21.gateway.workspace_index_jobs.v1"
	WorkspaceSourcesSchemaVersion             = "a21.gateway.workspace_sources.v1"
	WorkspaceDeviceBindingsSchemaVersion      = "a21.gateway.workspace_device_bindings.v1"
	VoiceChainProfileSchemaVersion            = "a21.gateway.voice_chain_profiles.v1"
	VoiceChainModeCascade                     = "cascade"
	VoiceChainModeRealtime                    = "realtime"
	DefaultCascadeASRProfile                  = "dashscope_qwen_asr_realtime"
	DefaultCascadeLLMProfile                  = "stepfun"
	DefaultFixedTTSProfile                    = "dashscope_qwen_tts_realtime"
	DefaultRealtimeProvider                   = "doubao_realtime"
	DefaultVoiceCloneProfile                  = "a21_voice_default_dashscope"
	GatewayProfileSchemaVersion               = "a21.gateway.profiles.v1"
	GatewayProfileMacLocal                    = "mac_local"
	GatewayProfilePublicWSS                   = "public_wss"
	CloudVoiceProfileSchemaVersion            = "a21.gateway.cloud_voice_profiles.v1"
	AudioRecentSchemaVersion                  = "a21.gateway.audio_recent.v1"
	maxAudioCaptureFrames                     = 512
	xiaozhiOTAWebSocketVersion                = 1
)

const (
	XiaozhiOpusNoFramesState           = "opus_no_frames"
	XiaozhiOpusDecodedPCMState         = "opus_decoded_pcm16"
	XiaozhiOpusDecodeErrorState        = "opus_decode_error"
	XiaozhiOpusPartialDecodeErrorState = "opus_partial_decode_error"
)

type AudioRecentResponse struct {
	SchemaVersion string              `json:"schema_version"`
	DeviceID      string              `json:"device_id,omitempty"`
	TraceID       string              `json:"trace_id,omitempty"`
	SessionID     string              `json:"session_id,omitempty"`
	IncludeAudio  bool                `json:"include_audio"`
	Frames        []AudioCaptureFrame `json:"frames"`
}

type AudioCaptureFrame struct {
	DeviceID            string  `json:"device_id"`
	TraceID             string  `json:"trace_id"`
	SessionID           string  `json:"session_id"`
	Seq                 uint64  `json:"seq"`
	SentAtMS            int64   `json:"sent_at_ms,omitempty"`
	ReceivedAtMS        int64   `json:"received_at_ms"`
	SampleRateHz        int     `json:"sample_rate_hz"`
	Channels            int     `json:"channels"`
	DurationMS          int     `json:"duration_ms"`
	CaptureStartedAtMS  int64   `json:"capture_started_at_ms,omitempty"`
	CaptureEndedAtMS    int64   `json:"capture_ended_at_ms,omitempty"`
	DataBytes           int     `json:"data_bytes"`
	DataBase64          string  `json:"data_base64,omitempty"`
	RMS                 float64 `json:"rms"`
	VADDetector         string  `json:"vad_detector"`
	VADStatus           string  `json:"vad_status,omitempty"`
	VADFinding          string  `json:"vad_finding,omitempty"`
	SpeechDetected      bool    `json:"speech_detected"`
	SpeechActive        bool    `json:"speech_active"`
	DroppedFrames       int     `json:"dropped_frames,omitempty"`
	DroppedFrameDelta   int     `json:"dropped_frame_delta,omitempty"`
	IngressBufferFrames int     `json:"ingress_buffer_frames"`
}

type TraceEvent struct {
	Name      string `json:"name"`
	TraceID   string `json:"trace_id"`
	SessionID string `json:"session_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	AtMS      int64  `json:"at_ms"`
	OffsetMS  int64  `json:"offset_ms"`
}

type TraceLatencySummary struct {
	EventCount                    int    `json:"event_count"`
	LastOffsetMS                  int64  `json:"last_offset_ms"`
	AudioFrameToPlaybackMS        *int64 `json:"audio_frame_to_playback_ms,omitempty"`
	V21QueryFirstResultMS         *int64 `json:"v21_query_first_result_ms,omitempty"`
	BargeInStopMS                 *int64 `json:"barge_in_stop_ms,omitempty"`
	ProviderCommitToFirstAudioMS  *int64 `json:"provider_commit_to_first_audio_ms,omitempty"`
	XiaozhiListenToAudioIngressMS *int64 `json:"xiaozhi_listen_to_audio_ingress_ms,omitempty"`
	XiaozhiOpusDecodeMS           *int64 `json:"xiaozhi_opus_decode_ms,omitempty"`
	ASRFirstPartialMS             *int64 `json:"asr_first_partial_ms,omitempty"`
	LLMFirstContentMS             *int64 `json:"llm_first_content_ms,omitempty"`
	TTSFirstAudioMS               *int64 `json:"tts_first_audio_ms,omitempty"`
	AudioDownlinkFirstFrameMS     *int64 `json:"audio_downlink_first_frame_ms,omitempty"`
	DevicePlaybackStartMS         *int64 `json:"device_playback_start_ms,omitempty"`
	AnswerFirstAudioTotalMS       *int64 `json:"answer_first_audio_total_ms,omitempty"`
}

type TraceResponse struct {
	TraceID string              `json:"trace_id"`
	Events  []TraceEvent        `json:"events"`
	Summary TraceLatencySummary `json:"summary"`
}

func NewServer() *Server {
	return NewServerWithOptions(ServerOptions{})
}

func NewServerWithOptions(options ServerOptions) *Server {
	voiceProvider := options.VoiceProvider
	if voiceProvider == nil {
		voiceProvider = providers.NewMockVoiceProvider()
	}
	v21Client := options.V21Client
	if v21Client == nil {
		v21Client = v21adapter.NewMockClient()
	}
	v21TTL := options.V21Timeout
	if v21TTL <= 0 {
		v21TTL = 3 * time.Second
	}
	xiaozhiListenMaxDurationMS := defaultXiaozhiListenMaxDurationMS
	if options.XiaozhiListenMaxDuration > 0 {
		xiaozhiListenMaxDurationMS = int64(options.XiaozhiListenMaxDuration / time.Millisecond)
	}
	bodySceneStepDelay := normalizedBodySceneStepDelay(options.BodySceneStepDelay)
	xiaozhiRunnerFactory := defaultXiaozhiVoicePipelineRunner
	xiaozhiPipelineMeta := xiaozhiVoicePipelineMeta{
		Selection:     providers.VoicePipelineSelectionFromEnv(nil),
		ExecutionMode: "fixture",
	}
	xiaozhiVoicePipelineASR := providers.NewMockASRAdapter("mock-local-asr")
	xiaozhiProfessionalASR := providers.NewMockASRAdapter("mock-local-asr")
	xiaozhiFastAckTTS := providers.NewMockTTSAdapter("mock-fast-tts")
	xiaozhiFastAckEnabled := gatewayEnvBoolDefault(options.CloudVoiceEnv, "A21_XIAOZHI_FAST_ACK_ENABLED", true)
	xiaozhiFastAckDelay := gatewayEnvDurationMSDefault(options.CloudVoiceEnv, "A21_XIAOZHI_FAST_ACK_DELAY_MS", 0)
	xiaozhiSTTScreenPolicy := normalizeXiaozhiSTTScreenPolicy(firstNonEmpty(options.XiaozhiSTTScreenPolicy, gatewayEnvValue(options.CloudVoiceEnv, "A21_XIAOZHI_STT_SCREEN_POLICY")))
	if options.XiaozhiVoicePipelineAdapters != nil {
		adapters := *options.XiaozhiVoicePipelineAdapters
		xiaozhiRunnerFactory = func() xiaozhiVoicePipelineRunner {
			return providers.NewVoicePipelineRunner(adapters)
		}
		xiaozhiPipelineMeta = xiaozhiVoicePipelineMeta{
			Selection:     adapters.Selection,
			ExecutionMode: adapters.ExecutionMode,
		}
		if isZeroGatewayVoicePipelineSelection(xiaozhiPipelineMeta.Selection) {
			xiaozhiPipelineMeta.Selection = providers.VoicePipelineSelectionFromEnv(nil)
		}
		if strings.TrimSpace(xiaozhiPipelineMeta.ExecutionMode) == "" {
			xiaozhiPipelineMeta.ExecutionMode = "fixture"
		}
		if adapters.TTS != nil {
			xiaozhiFastAckTTS = adapters.TTS
		}
		if adapters.ASR != nil {
			xiaozhiVoicePipelineASR = adapters.ASR
			xiaozhiProfessionalASR = adapters.ASR
		}
	}
	initialASRProfile := defaultCascadeASRProfile(xiaozhiPipelineMeta.Selection.ASRProfile)
	initialLLMProfile := defaultCascadeLLMProfile(xiaozhiPipelineMeta.Selection.LLMProfile)
	initialTTSProfile := defaultFixedTTSProfile(xiaozhiPipelineMeta.Selection.TTSProfile)
	initialRealtimeProvider := defaultRealtimeProvider(gatewayEnvValue(options.CloudVoiceEnv, "A21_PROVIDER_PRIMARY"))
	initialVoiceCloneProfile := defaultVoiceCloneProfile(gatewayEnvValue(options.CloudVoiceEnv, "A21_VOICE_CLONE_PROFILE"))
	workspaceDocumentMaxBytes := options.WorkspaceDocumentMaxBytes
	if workspaceDocumentMaxBytes <= 0 {
		workspaceDocumentMaxBytes = defaultWorkspaceDocumentMaxBytes
	}
	return &Server{
		now:                          time.Now,
		metrics:                      newMetrics(),
		voice:                        voiceProvider,
		v21:                          v21Client,
		v21TTL:                       v21TTL,
		devices:                      make(map[string]DeviceRecord),
		traces:                       make(map[string][]TraceEvent),
		audioStreams:                 make(map[string]string),
		activeStreams:                make(map[string]string),
		realtimeAudio:                make(map[string]providers.RealtimeVoiceSession),
		realtimeAudioCommitAt:        make(map[string]time.Time),
		realtimeAudioFirstDownlink:   make(map[string]bool),
		audioProbeSessions:           make(map[string]bool),
		mockPlaybackArmedSessions:    make(map[string]int),
		realtimeArmedSessions:        make(map[string]bool),
		audioIngress:                 audio.NewIngress(options.AudioIngressConfig),
		audioSockets:                 make(map[string]*deviceSocket),
		xiaozhiSockets:               make(map[string]*xiaozhiDeviceSocket),
		officialStackChanSockets:     make(map[string]*deviceSocket),
		audioCaptureFrames:           make([]AudioCaptureFrame, 0, maxAudioCaptureFrames),
		xiaozhiVoicePipelineRunner:   xiaozhiRunnerFactory,
		xiaozhiVoicePipelineMeta:     xiaozhiPipelineMeta,
		xiaozhiVoicePipelineASR:      xiaozhiVoicePipelineASR,
		xiaozhiProfessionalASR:       xiaozhiProfessionalASR,
		xiaozhiFastAckTTS:            xiaozhiFastAckTTS,
		xiaozhiFastAckEnabled:        xiaozhiFastAckEnabled,
		xiaozhiFastAckDelay:          xiaozhiFastAckDelay,
		xiaozhiSTTScreenPolicy:       xiaozhiSTTScreenPolicy,
		xiaozhiStockProfessional:     options.XiaozhiStockProfessional,
		xiaozhiProductPlaybackEvents: options.XiaozhiProductPlaybackEvents || gatewayEnvBool(options.CloudVoiceEnv, "A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS"),
		xiaozhiProductTouchEvents:    options.XiaozhiProductTouchEvents || gatewayEnvBool(options.CloudVoiceEnv, "A21_XIAOZHI_PRODUCT_TOUCH_EVENTS"),
		xiaozhiProductTouchReactions: options.XiaozhiProductTouchReactions || gatewayEnvBool(options.CloudVoiceEnv, "A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS"),
		xiaozhiProductStateReactions: options.XiaozhiProductStateReactions || gatewayEnvBool(options.CloudVoiceEnv, "A21_XIAOZHI_PRODUCT_STATE_REACTIONS"),
		xiaozhiListenMaxDurationMS:   xiaozhiListenMaxDurationMS,
		bodySceneStepDelay:           bodySceneStepDelay,
		wakeWordConfigPath:           wakeWordConfigPath(options.WakeWordConfigPath),
		roleplayProfileConfig:        DefaultRoleplayProfile,
		roleplayScenarioConfig:       DefaultRoleplayScenario,
		professionalUserIDConfig:     v21adapter.DefaultUserID,
		professionalWorkspaceConfig:  v21adapter.DefaultWorkspaceID,
		professionalQueryScopeConfig: v21adapter.QueryScopePublic,
		professionalReadRecords:      make(map[string]ProfessionalReadRecord),
		workspaceUploadJobs:          make(map[string]WorkspaceUploadJob),
		workspaceIndexJobs:           make(map[string]WorkspaceIndexJob),
		workspaceSources:             make(map[string]WorkspaceSource),
		workspaceDocuments:           make(map[string]WorkspaceDocument),
		workspaceDeviceBindings:      make(map[string]WorkspaceDeviceBinding),
		workspaceDocumentStoreDir:    workspaceDocumentStoreDir(options.WorkspaceDocumentStoreDir),
		workspaceDocumentMaxBytes:    workspaceDocumentMaxBytes,
		voiceChainModeConfig:         VoiceChainModeCascade,
		cascadeASRProfileConfig:      initialASRProfile,
		cascadeLLMProfileConfig:      initialLLMProfile,
		fixedTTSProfileConfig:        initialTTSProfile,
		realtimeProviderConfig:       initialRealtimeProvider,
		voiceCloneProfileConfig:      initialVoiceCloneProfile,
		gatewayProfileConfig:         defaultGatewayProfile(options.GatewayProfile, options.PublicGatewayURL),
		cloudVoiceProfileConfig:      strings.TrimSpace(options.CloudVoiceProfile),
		cloudVoiceEnv:                append([]string(nil), options.CloudVoiceEnv...),
		macLocalGatewayURL:           strings.TrimSpace(options.MacLocalGatewayURL),
		publicGatewayURL:             strings.TrimSpace(options.PublicGatewayURL),
	}
}

func gatewayEnvValue(env []string, key string) string {
	prefix := strings.TrimSpace(key) + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(item, prefix))
		}
	}
	return ""
}

func gatewayEnvBool(env []string, key string) bool {
	switch strings.ToLower(strings.TrimSpace(gatewayEnvValue(env, key))) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

func gatewayEnvBoolDefault(env []string, key string, defaultValue bool) bool {
	value := strings.ToLower(strings.TrimSpace(gatewayEnvValue(env, key)))
	if value == "" {
		return defaultValue
	}
	switch value {
	case "1", "true", "yes", "on", "enabled":
		return true
	case "0", "false", "no", "off", "disabled":
		return false
	default:
		return defaultValue
	}
}

func gatewayEnvDurationMSDefault(env []string, key string, defaultValue time.Duration) time.Duration {
	value := strings.TrimSpace(gatewayEnvValue(env, key))
	if value == "" {
		return defaultValue
	}
	ms, err := strconv.Atoi(value)
	if err != nil || ms < 0 {
		return defaultValue
	}
	return time.Duration(ms) * time.Millisecond
}

func normalizeXiaozhiSTTScreenPolicy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", xiaozhiSTTScreenPolicyRaw:
		return xiaozhiSTTScreenPolicyRaw
	case "status", "status-only", "status_only", "private", "redacted":
		return xiaozhiSTTScreenPolicyStatusOnly
	case "false", "no", "none", "disabled", "suppress", "suppressed", xiaozhiSTTScreenPolicyOff:
		return xiaozhiSTTScreenPolicyOff
	default:
		return xiaozhiSTTScreenPolicyRaw
	}
}

func isZeroGatewayVoicePipelineSelection(selection providers.VoicePipelineSelection) bool {
	return selection.ASRMode == "" &&
		selection.ASRProfile == "" &&
		selection.ASRProfileEnv == "" &&
		selection.LLMProfile == "" &&
		selection.LLMProfileEnv == "" &&
		selection.TTSMode == "" &&
		selection.TTSProfile == "" &&
		selection.TTSProfileEnv == ""
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/workspace", s.handleWorkspaceConsole)
	mux.Handle("/metrics", s.metrics.handler())
	mux.HandleFunc("/v1/devices", s.handleDevices)
	mux.HandleFunc("/v1/hardware-acceptance", s.handleHardwareAcceptance)
	mux.HandleFunc("/v1/power-lifecycle", s.handlePowerLifecycle)
	mux.HandleFunc("/v1/power-lifecycle-acceptance", s.handlePowerLifecycleAcceptance)
	mux.HandleFunc("/v1/devices/control", s.handleDeviceControl)
	mux.HandleFunc("/v1/voice-modes", s.handleVoiceModes)
	mux.HandleFunc("/v1/voice-mode-ritual", s.handleVoiceModeRitual)
	mux.HandleFunc("/v1/voice-mode-ritual-acceptance", s.handleVoiceModeRitualAcceptance)
	mux.HandleFunc("/v1/roleplay-profile", s.handleRoleplayProfile)
	mux.HandleFunc("/v1/professional-workspace", s.handleProfessionalWorkspace)
	mux.HandleFunc("/v1/professional-read-records", s.handleProfessionalReadRecords)
	mux.HandleFunc("/v1/professional-query", s.handleProfessionalQuery)
	mux.HandleFunc("/v1/workspace-documents", s.handleWorkspaceDocuments)
	mux.HandleFunc("/v1/workspace-upload-jobs", s.handleWorkspaceUploadJobs)
	mux.HandleFunc("/v1/workspace-index-jobs", s.handleWorkspaceIndexJobs)
	mux.HandleFunc("/v1/workspace-sources", s.handleWorkspaceSources)
	mux.HandleFunc("/v1/workspace-device-bindings", s.handleWorkspaceDeviceBindings)
	mux.HandleFunc("/v1/voice-chain-profiles", s.handleVoiceChainProfiles)
	mux.HandleFunc("/v1/gateway-profiles", s.handleGatewayProfiles)
	mux.HandleFunc("/v1/cloud-voice-profiles", s.handleCloudVoiceProfiles)
	mux.HandleFunc("/v1/audio/recent", s.handleAudioRecent)
	mux.HandleFunc("/v1/traces", s.handleTraces)
	mux.HandleFunc("/v1/providers/voice/health", s.handleVoiceProviderHealth)
	mux.HandleFunc("/v1/realtime/session", s.handleRealtimeSessionStart)
	mux.HandleFunc("/v1/realtime/session/cancel", s.handleRealtimeSessionCancel)
	mux.HandleFunc("/v1/fast-companion/turn", s.handleFastCompanionTurn)
	mux.HandleFunc("/v1/mock-turn", s.handleMockTurn)
	mux.HandleFunc("/v1/mock-interrupt", s.handleMockInterrupt)
	mux.HandleFunc("/v1/wake-word", s.handleWakeWordConfig)
	mux.HandleFunc("/v1/xiaozhi/control", s.handleXiaozhiDeviceControl)
	mux.HandleFunc("/v1/xiaozhi/say", s.handleXiaozhiSay)
	mux.HandleFunc("/v1/xiaozhi/speaker-volume", s.handleXiaozhiSpeakerVolume)
	mux.HandleFunc("/v1/xiaozhi/device-status", s.handleXiaozhiDeviceStatus)
	mux.HandleFunc("/v1/xiaozhi/screen-brightness", s.handleXiaozhiScreenBrightness)
	mux.HandleFunc("/v1/xiaozhi/screen-theme", s.handleXiaozhiScreenTheme)
	mux.HandleFunc("/v1/xiaozhi/body-preset", s.handleXiaozhiBodyPreset)
	mux.HandleFunc("/v1/xiaozhi/body-motion", s.handleXiaozhiBodyMotion)
	mux.HandleFunc("/v1/xiaozhi/body-scene", s.handleXiaozhiBodyScene)
	mux.HandleFunc("/v1/xiaozhi/body-scene-acceptance", s.handleXiaozhiBodySceneAcceptance)
	mux.HandleFunc("/v1/xiaozhi/mcp-capabilities", s.handleXiaozhiMCPCapabilities)
	mux.HandleFunc("/v1/xiaozhi/mcp-control", s.handleXiaozhiMCPControl)
	mux.HandleFunc("/v1/xiaozhi", s.handleXiaozhiWS)
	mux.HandleFunc("/stackChan/ws", s.handleOfficialStackChanWS)
	mux.HandleFunc("/v1/stackchan/official/control", s.handleOfficialStackChanControl)
	mux.HandleFunc("/v1/stackchan/official/status", s.handleOfficialStackChanStatus)
	mux.HandleFunc("/xiaozhi/ota/", s.handleXiaozhiOTA)
	mux.HandleFunc("/xiaozhi/ota", s.handleXiaozhiOTA)
	mux.HandleFunc("/ws/control", s.handleControlWS)
	mux.HandleFunc("/ws/audio", s.handleAudioWS)
	return mux
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, DeviceRegistryResponse{
		SchemaVersion: DeviceRegistrySchemaVersion,
		Service:       DeviceRegistryServiceName,
		Devices:       s.deviceRecords(),
	})
}

func (s *Server) handleHardwareAcceptance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if !validA21DeviceID(deviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, s.hardwareAcceptance(deviceID))
}

func (s *Server) handlePowerLifecycle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if !validA21DeviceID(deviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, s.powerLifecycle(deviceID))
}

func (s *Server) handlePowerLifecycleAcceptance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req PowerLifecycleAcceptanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
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
	if !req.ColdBootWithoutUSB || !req.PowerButtonStarted || !req.GatewayConnected || !req.XiaozhiSocketOnline || !req.StandaloneRuntimeOK {
		http.Error(w, "cold_boot_without_usb, power_button_started, gateway_connected, xiaozhi_socket_online, and standalone_runtime_ok must be true", http.StatusBadRequest)
		return
	}
	bootSource := strings.ToLower(strings.TrimSpace(req.BootSource))
	if bootSource != stackChanPowerLifecycleBootSourceBatteryKey {
		http.Error(w, "boot_source must be battery_power_key_cold_boot", http.StatusBadRequest)
		return
	}
	if req.USBConnectedDuringBoot == nil || *req.USBConnectedDuringBoot {
		http.Error(w, "usb_connected_during_boot must be explicitly false", http.StatusBadRequest)
		return
	}
	if req.PowerKeyHoldMS < 250 || req.PowerKeyHoldMS > 12000 {
		http.Error(w, "power_key_hold_ms must be between 250 and 12000", http.StatusBadRequest)
		return
	}
	pmicProfile := strings.TrimSpace(req.PMICPowerKeyProfile)
	if pmicProfile != stackChanPMICPowerKeyProfileAXP2101V1 {
		http.Error(w, "pmic_power_key_profile must be a21_stackchan_axp2101_pwrkey_v1", http.StatusBadRequest)
		return
	}
	if req.BootObservedAtMS <= 0 {
		http.Error(w, "boot_observed_at_ms is required", http.StatusBadRequest)
		return
	}
	state := s.powerLifecycle(req.DeviceID)
	if state.ConnectionStatus != "online" || !state.XiaozhiWSOnline {
		http.Error(w, "online device and Xiaozhi websocket evidence are required before power lifecycle acceptance", http.StatusConflict)
		return
	}

	nowMS := s.now().UnixMilli()
	marker := "power_lifecycle.physical_acceptance.accepted"
	s.recordTrace(traceID, sessionID, req.DeviceID, marker, nowMS)
	s.recordPowerLifecyclePhysicalAcceptance(req.DeviceID, traceID, sessionID, observer, bootSource, pmicProfile, req.PowerKeyHoldMS, req.BootObservedAtMS, nowMS, marker)
	writeJSON(w, http.StatusOK, PowerLifecycleAcceptanceResponse{
		SchemaVersion:          "a21.gateway.power_lifecycle_acceptance.v1",
		TraceID:                traceID,
		SessionID:              sessionID,
		DeviceID:               req.DeviceID,
		Status:                 "accepted",
		Observer:               observer,
		BootSource:             bootSource,
		USBConnectedDuringBoot: false,
		PowerKeyHoldMS:         req.PowerKeyHoldMS,
		PMICPowerKeyProfile:    pmicProfile,
		BootObservedAtMS:       req.BootObservedAtMS,
		AcceptedSurfaces:       []string{"no_cable_cold_boot", "physical_power_button", "gateway_reconnect", "xiaozhi_socket", "pmic_power_key_profile"},
		PhysicalAccepted:       true,
		ResultRedacted:         true,
		AcceptanceEventMS:      nowMS,
	})
}

func (s *Server) handleVoiceModes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.voiceModeCatalog())
	case http.MethodPost, http.MethodPut:
		var req VoiceModeSelectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if !validVoiceMode(req.VoiceMode) {
			http.Error(w, "voice_mode must be roleplay or professional", http.StatusBadRequest)
			return
		}
		s.setVoiceMode(req.VoiceMode)
		writeJSON(w, http.StatusOK, s.voiceModeCatalog())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleVoiceModeRitual(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req VoiceModeRitualRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	mode, ritual, plans, err := voiceModeRitualPlans(req.DeviceID, req.VoiceMode)
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
				s.recordTrace(traceID, sessionID, req.DeviceID, "voice_mode.ritual."+mode+".cancelled", s.now().UnixMilli())
				http.Error(w, "voice mode ritual delivery cancelled", http.StatusBadGateway)
				return
			}
		}
		plan.TraceID = traceID
		plan.SessionID = sessionID
		delivery, status, message := s.sendXiaozhiMCPControl(r.Context(), plan)
		if status != 0 {
			s.recordTrace(traceID, sessionID, req.DeviceID, "voice_mode.ritual."+mode+".failed", s.now().UnixMilli())
			http.Error(w, message, status)
			return
		}
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		modeMarker := "voice_mode.ritual." + mode + ".step" + strconv.Itoa(index+1) + "." + delivery.Marker + ".sent"
		nowMS := s.now().UnixMilli()
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, nowMS)
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, modeMarker, nowMS)
		activity := xiaozhiMCPActivity(delivery.Marker, delivery.Args)
		activity["last_voice_mode_ritual"] = mode
		activity["last_voice_mode_ritual_status"] = "delivered"
		activity["last_voice_mode_ritual_step"] = strconv.Itoa(index + 1)
		activity["last_voice_mode_ritual_tool"] = delivery.Marker
		activity["last_voice_mode_ritual_trace_id"] = delivery.Response.TraceID
		activity["last_voice_mode_ritual_session_id"] = delivery.Response.SessionID
		s.recordXiaozhiDeviceActivity(&xiaozhiSession{
			deviceID:  delivery.Response.DeviceID,
			traceID:   delivery.Response.TraceID,
			sessionID: delivery.Response.SessionID,
		}, modeMarker, activity)
		steps = append(steps, XiaozhiBodyPresetStepResponse{
			ToolName:  delivery.Response.ToolName,
			MCPID:     delivery.Response.MCPID,
			Marker:    delivery.Marker,
			Arguments: delivery.Response.Arguments,
		})
	}
	s.setVoiceMode(mode)
	completedAt := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, req.DeviceID, "voice_mode.ritual."+mode+".completed", completedAt)
	s.recordVoiceModeRitualCompleted(req.DeviceID, mode, traceID, sessionID, completedAt)
	writeJSON(w, http.StatusOK, VoiceModeRitualResponse{
		SchemaVersion:        "a21.gateway.voice_mode_ritual.v1",
		TraceID:              traceID,
		SessionID:            sessionID,
		DeviceID:             req.DeviceID,
		Status:               "delivered",
		SelectedVoiceMode:    mode,
		DeliveredTransport:   "xiaozhi_mcp_sequence",
		ScreenLabel:          ritual.ScreenLabel,
		WorkspacePolicy:      ritual.WorkspacePolicy,
		Steps:                steps,
		StepDelayMS:          int64(stepDelay / time.Millisecond),
		TotalPlannedDelayMS:  bodySceneTotalPlannedDelayMS(stepDelay, len(steps)),
		ModeSwitchVisible:    true,
		ProviderExecuted:     false,
		V21Executed:          false,
		OfficialRelayClaimed: false,
		ResultRedacted:       true,
		PhysicalAccepted:     false,
	})
}
