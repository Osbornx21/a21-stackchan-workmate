package gateway

import (
	"sync"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
)

type WorkspaceDeviceBindingRequest struct {
	BindingID          string   `json:"binding_id,omitempty"`
	Action             string   `json:"action,omitempty"`
	DeviceID           string   `json:"device_id,omitempty"`
	DeviceLabel        string   `json:"device_label,omitempty"`
	UserID             string   `json:"user_id,omitempty"`
	WorkspaceID        string   `json:"workspace_id,omitempty"`
	AllowedQueryScopes []string `json:"allowed_query_scopes,omitempty"`
	TraceID            string   `json:"trace_id,omitempty"`
	SessionID          string   `json:"session_id,omitempty"`
}

type WorkspaceDeviceBindingsResponse struct {
	SchemaVersion string                          `json:"schema_version"`
	Service       string                          `json:"service"`
	Status        string                          `json:"status"`
	Bindings      []WorkspaceDeviceBinding        `json:"bindings"`
	Summary       WorkspaceDeviceBindingSummary   `json:"summary"`
	Redaction     WorkspaceDeviceBindingRedaction `json:"redaction"`
	Findings      []WorkspaceUploadJobFinding     `json:"findings,omitempty"`
}

type WorkspaceDeviceBinding struct {
	BindingID           string                          `json:"binding_id"`
	DeviceID            string                          `json:"device_id"`
	DeviceLabel         string                          `json:"device_label,omitempty"`
	UserID              string                          `json:"user_id"`
	WorkspaceID         string                          `json:"workspace_id"`
	Status              string                          `json:"status"`
	AccessScope         string                          `json:"access_scope"`
	AllowedQueryScopes  []string                        `json:"allowed_query_scopes"`
	ProfessionalAllowed bool                            `json:"professional_allowed"`
	PhysicalAccepted    bool                            `json:"physical_accepted"`
	CreatedAtMS         int64                           `json:"created_at_ms"`
	UpdatedAtMS         int64                           `json:"updated_at_ms"`
	RevokedAtMS         int64                           `json:"revoked_at_ms,omitempty"`
	TraceID             string                          `json:"trace_id,omitempty"`
	SessionID           string                          `json:"session_id,omitempty"`
	Redaction           WorkspaceDeviceBindingRedaction `json:"redaction"`
	Findings            []WorkspaceUploadJobFinding     `json:"findings,omitempty"`
}

type WorkspaceDeviceBindingSummary struct {
	TotalBindings              int    `json:"total_bindings"`
	ActiveBindings             int    `json:"active_bindings"`
	RevokedBindings            int    `json:"revoked_bindings"`
	DeletedBindings            int    `json:"deleted_bindings"`
	ProfessionalAllowedDevices int    `json:"professional_allowed_devices"`
	DeviceBindingPolicy        string `json:"device_binding_policy"`
	WorkspaceAccessStatus      string `json:"workspace_access_status"`
}

type WorkspaceDeviceBindingRedaction struct {
	PairingSecretStored    bool `json:"pairing_secret_stored"`
	DeviceCredentialStored bool `json:"device_credential_stored"`
	DocumentTextStored     bool `json:"document_text_stored"`
	QueryTextStored        bool `json:"query_text_stored"`
	RetrievedTextStored    bool `json:"retrieved_text_stored"`
	FullURLStored          bool `json:"full_url_stored"`
	LocalPathStored        bool `json:"local_path_stored"`
	CredentialValueStored  bool `json:"credential_value_stored"`
	ProviderOutputStored   bool `json:"provider_output_stored"`
	VoiceTranscriptStored  bool `json:"voice_transcript_stored"`
}

type professionalDeviceBindingDecision struct {
	Allowed     bool
	Policy      string
	BindingID   string
	Status      string
	FailureCode string
	TraceMarker string
}

type DeviceControlRequest struct {
	DeviceID                     string                        `json:"device_id"`
	State                        protocol.ExpressionState      `json:"state,omitempty"`
	Mode                         protocol.Mode                 `json:"mode,omitempty"`
	Text                         string                        `json:"text,omitempty"`
	TraceID                      string                        `json:"trace_id,omitempty"`
	SessionID                    string                        `json:"session_id,omitempty"`
	StreamID                     string                        `json:"stream_id,omitempty"`
	DiagnosticToneHz             int                           `json:"diagnostic_tone_hz,omitempty"`
	DiagnosticToneDurationMS     int                           `json:"diagnostic_tone_duration_ms,omitempty"`
	DiagnosticToneVolume         int                           `json:"diagnostic_tone_volume,omitempty"`
	MockAudioChunks              *int                          `json:"mock_audio_chunks,omitempty"`
	AudioChunks                  []protocol.AudioPlaybackChunk `json:"audio_chunks,omitempty"`
	AudioProbeOnly               bool                          `json:"audio_probe_only,omitempty"`
	MockPlaybackOnNextAudioFrame bool                          `json:"mock_playback_on_next_audio_frame,omitempty"`
	RealtimeOnNextSpeech         bool                          `json:"realtime_on_next_speech,omitempty"`
}

type DeviceControlResponse struct {
	TraceID            string              `json:"trace_id"`
	SessionID          string              `json:"session_id"`
	DeviceID           string              `json:"device_id"`
	Status             string              `json:"status"`
	DeliveredTransport string              `json:"delivered_transport"`
	Events             []protocol.Envelope `json:"events"`
}

type deviceSocket struct {
	conn      *websocket.Conn
	writeMu   *sync.Mutex
	connected int64
}

type XiaozhiDeviceControlRequest struct {
	DeviceID         string `json:"device_id"`
	Kind             string `json:"kind,omitempty"`
	Event            string `json:"event,omitempty"`
	State            string `json:"state,omitempty"`
	Face             string `json:"face,omitempty"`
	Emotion          string `json:"emotion,omitempty"`
	Display          string `json:"display,omitempty"`
	Slot             string `json:"slot,omitempty"`
	Motion           string `json:"motion,omitempty"`
	Name             string `json:"name,omitempty"`
	Text             string `json:"text,omitempty"`
	Reason           string `json:"reason,omitempty"`
	YAngle           int    `json:"y_angle,omitempty"`
	TraceID          string `json:"trace_id,omitempty"`
	SessionID        string `json:"session_id,omitempty"`
	AllowMCPFallback bool   `json:"allow_mcp_fallback,omitempty"`
}

type XiaozhiDeviceControlResponse struct {
	TraceID                        string            `json:"trace_id"`
	SessionID                      string            `json:"session_id"`
	DeviceID                       string            `json:"device_id"`
	Status                         string            `json:"status"`
	DeliveredTransport             string            `json:"delivered_transport"`
	Event                          string            `json:"event"`
	Value                          string            `json:"value"`
	PacketCount                    int               `json:"packet_count,omitempty"`
	OfficialActionPhysicalAccepted *bool             `json:"official_action_physical_accepted,omitempty"`
	OfficialActionSurfaces         map[string]string `json:"official_action_surfaces,omitempty"`
	OfficialActionFallbackReason   string            `json:"official_action_fallback_reason,omitempty"`
}

type OfficialStackChanStatusResponse struct {
	SchemaVersion          string            `json:"schema_version"`
	DeviceID               string            `json:"device_id"`
	OfficialDeviceID       string            `json:"official_device_id,omitempty"`
	Connected              bool              `json:"connected"`
	ConnectedMS            int64             `json:"connected_ms,omitempty"`
	ConnectedSinceMS       int64             `json:"connected_since_ms,omitempty"`
	FallbackAvailable      bool              `json:"fallback_available"`
	DeliveredTransport     string            `json:"delivered_transport"`
	LastTraceID            string            `json:"last_trace_id,omitempty"`
	LastSessionID          string            `json:"last_session_id,omitempty"`
	LastEvent              string            `json:"last_event,omitempty"`
	LastAutoState          string            `json:"last_auto_state,omitempty"`
	LastAutoReason         string            `json:"last_auto_reason,omitempty"`
	LastAutoTarget         string            `json:"last_auto_target,omitempty"`
	LastPacketCount        int               `json:"last_packet_count,omitempty"`
	PhysicalAccepted       bool              `json:"physical_accepted"`
	OfficialActionSurfaces map[string]string `json:"official_action_surfaces,omitempty"`
	NextAction             string            `json:"next_action"`
}

type XiaozhiSpeakerVolumeRequest struct {
	DeviceID  string `json:"device_id"`
	Volume    int    `json:"volume"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type XiaozhiSpeakerVolumeResponse struct {
	TraceID            string `json:"trace_id"`
	SessionID          string `json:"session_id"`
	DeviceID           string `json:"device_id"`
	Status             string `json:"status"`
	DeliveredTransport string `json:"delivered_transport"`
	ToolName           string `json:"tool_name"`
	Volume             int    `json:"volume"`
	MCPID              string `json:"mcp_id"`
}

type XiaozhiMCPControlRequest struct {
	DeviceID   string `json:"device_id"`
	ToolName   string `json:"tool_name"`
	Brightness *int   `json:"brightness,omitempty"`
	Volume     *int   `json:"volume,omitempty"`
	Theme      string `json:"theme,omitempty"`
	Yaw        *int   `json:"yaw,omitempty"`
	Pitch      *int   `json:"pitch,omitempty"`
	Speed      *int   `json:"speed,omitempty"`
	Red        *int   `json:"red,omitempty"`
	Green      *int   `json:"green,omitempty"`
	Blue       *int   `json:"blue,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
}

type XiaozhiDeviceStatusRequest struct {
	DeviceID  string `json:"device_id"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type XiaozhiScreenBrightnessRequest struct {
	DeviceID   string `json:"device_id"`
	Brightness *int   `json:"brightness"`
	TraceID    string `json:"trace_id,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
}

type XiaozhiScreenThemeRequest struct {
	DeviceID  string `json:"device_id"`
	Theme     string `json:"theme"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type XiaozhiBodyPresetRequest struct {
	DeviceID  string `json:"device_id"`
	Preset    string `json:"preset"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type XiaozhiBodyMotionRequest struct {
	DeviceID  string `json:"device_id"`
	Motion    string `json:"motion"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type XiaozhiBodySceneRequest struct {
	DeviceID  string `json:"device_id"`
	Scene     string `json:"scene"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type XiaozhiBodySceneAcceptanceRequest struct {
	DeviceID      string `json:"device_id"`
	Scene         string `json:"scene"`
	TraceID       string `json:"trace_id"`
	SessionID     string `json:"session_id"`
	ScreenVisible bool   `json:"screen_visible"`
	RGBVisible    bool   `json:"rgb_visible"`
	ServoVisible  bool   `json:"servo_visible"`
	Observer      string `json:"observer"`
}

type PowerLifecycleAcceptanceRequest struct {
	DeviceID               string `json:"device_id"`
	TraceID                string `json:"trace_id"`
	SessionID              string `json:"session_id"`
	ColdBootWithoutUSB     bool   `json:"cold_boot_without_usb"`
	PowerButtonStarted     bool   `json:"power_button_started"`
	GatewayConnected       bool   `json:"gateway_connected"`
	XiaozhiSocketOnline    bool   `json:"xiaozhi_socket_online"`
	StandaloneRuntimeOK    bool   `json:"standalone_runtime_ok"`
	BootSource             string `json:"boot_source"`
	USBConnectedDuringBoot *bool  `json:"usb_connected_during_boot"`
	PowerKeyHoldMS         int    `json:"power_key_hold_ms"`
	PMICPowerKeyProfile    string `json:"pmic_power_key_profile"`
	BootObservedAtMS       int64  `json:"boot_observed_at_ms"`
	Observer               string `json:"observer"`
}

type PowerLifecycleAcceptanceResponse struct {
	SchemaVersion          string   `json:"schema_version"`
	TraceID                string   `json:"trace_id"`
	SessionID              string   `json:"session_id"`
	DeviceID               string   `json:"device_id"`
	Status                 string   `json:"status"`
	Observer               string   `json:"observer"`
	BootSource             string   `json:"boot_source"`
	USBConnectedDuringBoot bool     `json:"usb_connected_during_boot"`
	PowerKeyHoldMS         int      `json:"power_key_hold_ms"`
	PMICPowerKeyProfile    string   `json:"pmic_power_key_profile"`
	BootObservedAtMS       int64    `json:"boot_observed_at_ms"`
	AcceptedSurfaces       []string `json:"accepted_surfaces"`
	PhysicalAccepted       bool     `json:"physical_accepted"`
	ResultRedacted         bool     `json:"result_redacted"`
	AcceptanceEventMS      int64    `json:"acceptance_event_ms"`
}

const (
	stackChanPowerLifecycleBootSourceBatteryKey = "battery_power_key_cold_boot"
	stackChanPMICPowerKeyProfileAXP2101V1       = "a21_stackchan_axp2101_pwrkey_v1"
)

type XiaozhiMCPControlResponse struct {
	TraceID            string         `json:"trace_id"`
	SessionID          string         `json:"session_id"`
	DeviceID           string         `json:"device_id"`
	Status             string         `json:"status"`
	DeliveredTransport string         `json:"delivered_transport"`
	ToolName           string         `json:"tool_name"`
	MCPID              string         `json:"mcp_id"`
	Arguments          map[string]any `json:"arguments,omitempty"`
	ResultRedacted     bool           `json:"result_redacted"`
}

type XiaozhiBodyPresetStepResponse struct {
	ToolName  string         `json:"tool_name"`
	MCPID     string         `json:"mcp_id"`
	Marker    string         `json:"marker"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type XiaozhiBodyPresetResponse struct {
	SchemaVersion      string                          `json:"schema_version"`
	TraceID            string                          `json:"trace_id"`
	SessionID          string                          `json:"session_id"`
	DeviceID           string                          `json:"device_id"`
	Status             string                          `json:"status"`
	DeliveredTransport string                          `json:"delivered_transport"`
	Preset             string                          `json:"preset"`
	Steps              []XiaozhiBodyPresetStepResponse `json:"steps"`
	ResultRedacted     bool                            `json:"result_redacted"`
	PhysicalAccepted   bool                            `json:"physical_accepted"`
}

type XiaozhiBodyMotionResponse struct {
	SchemaVersion      string                          `json:"schema_version"`
	TraceID            string                          `json:"trace_id"`
	SessionID          string                          `json:"session_id"`
	DeviceID           string                          `json:"device_id"`
	Status             string                          `json:"status"`
	DeliveredTransport string                          `json:"delivered_transport"`
	Motion             string                          `json:"motion"`
	Steps              []XiaozhiBodyPresetStepResponse `json:"steps"`
	ResultRedacted     bool                            `json:"result_redacted"`
	PhysicalAccepted   bool                            `json:"physical_accepted"`
}

type XiaozhiBodySceneResponse struct {
	SchemaVersion       string                          `json:"schema_version"`
	TraceID             string                          `json:"trace_id"`
	SessionID           string                          `json:"session_id"`
	DeviceID            string                          `json:"device_id"`
	Status              string                          `json:"status"`
	DeliveredTransport  string                          `json:"delivered_transport"`
	Scene               string                          `json:"scene"`
	Steps               []XiaozhiBodyPresetStepResponse `json:"steps"`
	StepDelayMS         int64                           `json:"step_delay_ms"`
	TotalPlannedDelayMS int64                           `json:"total_planned_delay_ms"`
	ResultRedacted      bool                            `json:"result_redacted"`
	PhysicalAccepted    bool                            `json:"physical_accepted"`
}

type XiaozhiBodySceneAcceptanceResponse struct {
	SchemaVersion     string   `json:"schema_version"`
	TraceID           string   `json:"trace_id"`
	SessionID         string   `json:"session_id"`
	DeviceID          string   `json:"device_id"`
	Status            string   `json:"status"`
	Scene             string   `json:"scene"`
	Observer          string   `json:"observer"`
	AcceptedSurfaces  []string `json:"accepted_surfaces"`
	PhysicalAccepted  bool     `json:"physical_accepted"`
	ResultRedacted    bool     `json:"result_redacted"`
	AcceptanceEventMS int64    `json:"acceptance_event_ms"`
}

type XiaozhiMCPCapabilitiesResponse struct {
	SchemaVersion      string   `json:"schema_version"`
	Service            string   `json:"service"`
	DeviceID           string   `json:"device_id"`
	ConnectionStatus   string   `json:"connection_status"`
	MCPAdvertised      bool     `json:"mcp_advertised"`
	AllowedTools       []string `json:"allowed_tools"`
	BlockedToolClasses []string `json:"blocked_tool_classes"`
	ResultRedacted     bool     `json:"result_redacted"`
	PhysicalAccepted   bool     `json:"physical_accepted"`
}

type XiaozhiSayRequest struct {
	DeviceID  string        `json:"device_id"`
	Text      string        `json:"text"`
	WAVPath   string        `json:"wav_path,omitempty"`
	Mode      protocol.Mode `json:"mode,omitempty"`
	TraceID   string        `json:"trace_id,omitempty"`
	SessionID string        `json:"session_id,omitempty"`
}

type XiaozhiSayResponse struct {
	TraceID            string `json:"trace_id"`
	SessionID          string `json:"session_id"`
	DeviceID           string `json:"device_id"`
	Status             string `json:"status"`
	DeliveredTransport string `json:"delivered_transport"`
	TextChars          int    `json:"text_chars"`
	AudioChunks        int    `json:"audio_chunks"`
	InterruptReason    string `json:"interrupt_reason,omitempty"`
	AudioSource        string `json:"audio_source,omitempty"`
	AudioBasename      string `json:"audio_basename,omitempty"`
}

type xiaozhiDeviceSocket struct {
	conn      *websocket.Conn
	writeMu   *sync.Mutex
	session   *xiaozhiSession
	features  xiaozhitransport.HelloFeatures
	connected int64
}

type realtimeVoiceOutput struct {
	Control protocol.ControlEventPayload
	Audio   *providers.VoiceAudioChunk
}

type DeviceFirmwareIdentity struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
	Board   string `json:"board,omitempty"`
	Commit  string `json:"commit,omitempty"`
}

type DeviceRecord struct {
	DeviceID                 string                   `json:"device_id"`
	Firmware                 DeviceFirmwareIdentity   `json:"firmware,omitempty"`
	Capabilities             map[string]string        `json:"capabilities,omitempty"`
	RuntimeEcho              map[string]string        `json:"runtime_echo,omitempty"`
	IdentityStatus           string                   `json:"identity_status"`
	IdentityError            string                   `json:"identity_error,omitempty"`
	ConnectionStatus         string                   `json:"connection_status,omitempty"`
	DeviceAgeMS              int64                    `json:"device_age_ms,omitempty"`
	CurrentMode              protocol.Mode            `json:"current_mode,omitempty"`
	CurrentVoiceMode         string                   `json:"current_voice_mode,omitempty"`
	CurrentVoiceChainMode    string                   `json:"current_voice_chain_mode,omitempty"`
	CurrentASRProfile        string                   `json:"current_asr_profile,omitempty"`
	CurrentLLMProfile        string                   `json:"current_llm_profile,omitempty"`
	CurrentTTSProfile        string                   `json:"current_tts_profile,omitempty"`
	CurrentRealtimeProvider  string                   `json:"current_realtime_provider,omitempty"`
	CurrentVoiceCloneProfile string                   `json:"current_voice_clone_profile,omitempty"`
	CurrentRoleplayProfile   string                   `json:"current_roleplay_profile,omitempty"`
	CurrentRoleplayScenario  string                   `json:"current_roleplay_scenario,omitempty"`
	RoleplaySoulReady        bool                     `json:"roleplay_soul_ready"`
	RoleplayMemoryReady      bool                     `json:"roleplay_memory_ready"`
	RoleplayMemoryHintCount  int                      `json:"roleplay_memory_hint_count"`
	RoleplayPhysicalAccepted bool                     `json:"roleplay_physical_accepted"`
	CurrentCloudVoiceProfile string                   `json:"current_cloud_voice_profile,omitempty"`
	CurrentExpr              protocol.ExpressionState `json:"current_expression,omitempty"`
	DisplayState             protocol.DisplayState    `json:"display_state,omitempty"`
	DisplayStateSource       string                   `json:"display_state_source,omitempty"`
	DisplayStateTraceID      string                   `json:"display_state_trace_id,omitempty"`
	DisplayStateSessionID    string                   `json:"display_state_session_id,omitempty"`
	DisplayStateUpdatedAtMS  int64                    `json:"display_state_updated_at_ms,omitempty"`
	DisplayStateAccepted     bool                     `json:"display_state_physical_accepted"`
	PlaybackStream           string                   `json:"playback_stream_id,omitempty"`
	LastEvent                protocol.DeviceEventKind `json:"last_event,omitempty"`
	LastTouchEvent           protocol.DeviceEventKind `json:"last_touch_event,omitempty"`
	LastTouchSource          protocol.TouchSource     `json:"last_touch_source,omitempty"`
	LastTouchTraceID         string                   `json:"last_touch_trace_id,omitempty"`
	LastTouchSessionID       string                   `json:"last_touch_session_id,omitempty"`
	LastTouchSeenMS          int64                    `json:"last_touch_seen_ms,omitempty"`
	LastSeq                  uint64                   `json:"last_seq,omitempty"`
	LastTraceID              string                   `json:"last_trace_id,omitempty"`
	LastSessionID            string                   `json:"last_session_id,omitempty"`
	FirstSeenMS              int64                    `json:"first_seen_ms"`
	LastSeenMS               int64                    `json:"last_seen_ms"`
}

type DeviceRegistryResponse struct {
	SchemaVersion string         `json:"schema_version"`
	Service       string         `json:"service"`
	Devices       []DeviceRecord `json:"devices"`
}

type HardwareAcceptanceResponse struct {
	SchemaVersion    string                   `json:"schema_version"`
	Service          string                   `json:"service"`
	DeviceID         string                   `json:"device_id"`
	ConnectionStatus string                   `json:"connection_status"`
	OverallStatus    string                   `json:"overall_status"`
	PhysicalAccepted bool                     `json:"physical_accepted"`
	Items            []HardwareAcceptanceItem `json:"items"`
}

type HardwareAcceptanceItem struct {
	ID                 string `json:"id"`
	Label              string `json:"label"`
	DeliveryStatus     string `json:"delivery_status"`
	PhysicalAccepted   bool   `json:"physical_accepted"`
	TraceID            string `json:"trace_id,omitempty"`
	SessionID          string `json:"session_id,omitempty"`
	AcceptanceEndpoint string `json:"acceptance_endpoint,omitempty"`
	NextAction         string `json:"next_action"`
}

type PowerLifecycleResponse struct {
	SchemaVersion    string               `json:"schema_version"`
	Service          string               `json:"service"`
	DeviceID         string               `json:"device_id"`
	ConnectionStatus string               `json:"connection_status"`
	OverallStatus    string               `json:"overall_status"`
	PhysicalAccepted bool                 `json:"physical_accepted"`
	XiaozhiWSOnline  bool                 `json:"xiaozhi_ws_online"`
	BatteryTelemetry string               `json:"battery_telemetry"`
	Items            []PowerLifecycleItem `json:"items"`
	ResultRedacted   bool                 `json:"result_redacted"`
}

type PowerLifecycleItem struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Status           string `json:"status"`
	PhysicalAccepted bool   `json:"physical_accepted"`
	EvidenceSource   string `json:"evidence_source"`
	NextAction       string `json:"next_action"`
}

type VoiceModeOption struct {
	ID          string          `json:"id"`
	Label       string          `json:"label"`
	Status      string          `json:"status"`
	Default     bool            `json:"default,omitempty"`
	Description string          `json:"description,omitempty"`
	Ritual      VoiceModeRitual `json:"ritual"`
}

type VoiceModeRitual struct {
	Mode             string                   `json:"mode"`
	ScreenLabel      string                   `json:"screen_label"`
	CueText          string                   `json:"cue_text"`
	Expression       protocol.ExpressionState `json:"expression"`
	TraceMarker      string                   `json:"trace_marker"`
	WorkspacePolicy  string                   `json:"workspace_policy"`
	V21Allowed       bool                     `json:"v21_allowed"`
	PhysicalAccepted bool                     `json:"physical_accepted"`
}

type VoiceModeCatalogResponse struct {
	SchemaVersion     string            `json:"schema_version"`
	Service           string            `json:"service"`
	SelectedVoiceMode string            `json:"selected_voice_mode"`
	SelectedRitual    VoiceModeRitual   `json:"selected_ritual"`
	Modes             []VoiceModeOption `json:"modes"`
}

type VoiceModeSelectionRequest struct {
	VoiceMode string `json:"voice_mode"`
}

type VoiceModeRitualRequest struct {
	DeviceID  string `json:"device_id"`
	VoiceMode string `json:"voice_mode"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type VoiceModeRitualAcceptanceRequest struct {
	DeviceID      string `json:"device_id"`
	VoiceMode     string `json:"voice_mode"`
	TraceID       string `json:"trace_id"`
	SessionID     string `json:"session_id"`
	ScreenVisible bool   `json:"screen_visible"`
	RGBVisible    bool   `json:"rgb_visible"`
	ServoVisible  bool   `json:"servo_visible"`
	Observer      string `json:"observer"`
}

type VoiceModeRitualResponse struct {
	SchemaVersion        string                          `json:"schema_version"`
	TraceID              string                          `json:"trace_id"`
	SessionID            string                          `json:"session_id"`
	DeviceID             string                          `json:"device_id"`
	Status               string                          `json:"status"`
	SelectedVoiceMode    string                          `json:"selected_voice_mode"`
	DeliveredTransport   string                          `json:"delivered_transport"`
	ScreenLabel          string                          `json:"screen_label"`
	WorkspacePolicy      string                          `json:"workspace_policy"`
	Steps                []XiaozhiBodyPresetStepResponse `json:"steps"`
	StepDelayMS          int64                           `json:"step_delay_ms"`
	TotalPlannedDelayMS  int64                           `json:"total_planned_delay_ms"`
	ModeSwitchVisible    bool                            `json:"mode_switch_visible"`
	ProviderExecuted     bool                            `json:"provider_executed"`
	V21Executed          bool                            `json:"v21_executed"`
	OfficialRelayClaimed bool                            `json:"official_relay_claimed"`
	ResultRedacted       bool                            `json:"result_redacted"`
	PhysicalAccepted     bool                            `json:"physical_accepted"`
}

type VoiceModeRitualAcceptanceResponse struct {
	SchemaVersion     string   `json:"schema_version"`
	TraceID           string   `json:"trace_id"`
	SessionID         string   `json:"session_id"`
	DeviceID          string   `json:"device_id"`
	Status            string   `json:"status"`
	SelectedVoiceMode string   `json:"selected_voice_mode"`
	Observer          string   `json:"observer"`
	AcceptedSurfaces  []string `json:"accepted_surfaces"`
	PhysicalAccepted  bool     `json:"physical_accepted"`
	ResultRedacted    bool     `json:"result_redacted"`
	AcceptanceEventMS int64    `json:"acceptance_event_ms"`
}

type VoiceChainProfileOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Status      string `json:"status"`
	Default     bool   `json:"default,omitempty"`
	Recommended bool   `json:"recommended,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type VoiceChainVoiceOption struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	Status          string `json:"status"`
	ProviderProfile string `json:"provider_profile"`
	TTSProfile      string `json:"tts_profile"`
	VoiceClone      bool   `json:"voice_clone,omitempty"`
	Default         bool   `json:"default,omitempty"`
}

type VoiceChainProfilesResponse struct {
	SchemaVersion             string                    `json:"schema_version"`
	Service                   string                    `json:"service"`
	SelectedVoiceChainMode    string                    `json:"selected_voice_chain_mode"`
	SelectedASRProfile        string                    `json:"selected_asr_profile"`
	SelectedLLMProfile        string                    `json:"selected_llm_profile"`
	FixedTTSProfile           string                    `json:"fixed_tts_profile"`
	SelectedTTSProfile        string                    `json:"selected_tts_profile"`
	SelectedRealtimeProvider  string                    `json:"selected_realtime_provider"`
	SelectedVoiceCloneProfile string                    `json:"selected_voice_clone_profile"`
	Cascade                   VoiceChainCascadeCatalog  `json:"cascade"`
	Realtime                  VoiceChainRealtimeCatalog `json:"realtime"`
	Voices                    []VoiceChainVoiceOption   `json:"voices"`
	HotSwitch                 bool                      `json:"hot_switch"`
	Findings                  []string                  `json:"findings,omitempty"`
}

type VoiceChainCascadeCatalog struct {
	ASRProfiles []VoiceChainProfileOption `json:"asr_profiles"`
	LLMProfiles []VoiceChainProfileOption `json:"llm_profiles"`
	TTSProfile  VoiceChainProfileOption   `json:"tts_profile"`
}

type VoiceChainRealtimeCatalog struct {
	Providers []VoiceChainProfileOption `json:"providers"`
}

type VoiceChainProfileSelectionRequest struct {
	VoiceChainMode    string `json:"voice_chain_mode"`
	ASRProfile        string `json:"asr_profile,omitempty"`
	LLMProfile        string `json:"llm_profile,omitempty"`
	RealtimeProvider  string `json:"realtime_provider,omitempty"`
	VoiceCloneProfile string `json:"voice_clone_profile,omitempty"`
}

type GatewayProfileOption struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Status       string `json:"status"`
	Default      bool   `json:"default,omitempty"`
	WebSocketURL string `json:"websocket_url,omitempty"`
	Description  string `json:"description,omitempty"`
}

type GatewayProfileCatalogResponse struct {
	SchemaVersion          string                 `json:"schema_version"`
	Service                string                 `json:"service"`
	SelectedGatewayProfile string                 `json:"selected_gateway_profile"`
	Profiles               []GatewayProfileOption `json:"profiles"`
}
