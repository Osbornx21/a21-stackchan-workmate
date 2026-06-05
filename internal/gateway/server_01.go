package gateway

import (
	"sync"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/personality"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
)

const xiaozhiPlaybackInterruptWindowMS int64 = 3000

const xiaozhiTouchBargeInInputCooldownMS int64 = 700

const xiaozhiHostSayInputCooldownMS int64 = 1200

const xiaozhiNoSpeechInputCooldownMS int64 = 1200

const xiaozhiPostTTSInputCooldownMS int64 = 900

const xiaozhiSuppressedListenDrainMS int64 = 1200

const defaultXiaozhiListenMaxDurationMS int64 = 7000

const maxXiaozhiWakePrerollFrames = 5

const maxXiaozhiOpusIngressQueueFrames = 16

const defaultBodySceneStepDelay = 20 * time.Millisecond

const maxBodySceneStepDelay = 1000 * time.Millisecond

const defaultWorkspaceDocumentMaxBytes int64 = 16 << 20

const workspaceDocumentMultipartOverheadBytes int64 = 1 << 20

const defaultOfficialStackChanDeviceID = "stackchan-official"

const localFallbackText = "外部大脑连不上，但我还在。你可以继续说，我先记下来。"

const xiaozhiSTTScreenPolicyRaw = "raw"

const xiaozhiSTTScreenPolicyStatusOnly = "status_only"

const xiaozhiSTTScreenPolicyOff = "off"

const xiaozhiSTTStatusOnlyText = "语音已识别"

type Server struct {
	mu                           sync.Mutex
	next                         uint64
	now                          func() time.Time
	metrics                      *metrics
	voice                        providers.VoiceProvider
	v21                          v21adapter.Client
	v21TTL                       time.Duration
	devices                      map[string]DeviceRecord
	traces                       map[string][]TraceEvent
	audioStreams                 map[string]string
	activeStreams                map[string]string
	realtimeAudio                map[string]providers.RealtimeVoiceSession
	realtimeAudioCommitAt        map[string]time.Time
	realtimeAudioFirstDownlink   map[string]bool
	audioProbeSessions           map[string]bool
	mockPlaybackArmedSessions    map[string]int
	realtimeArmedSessions        map[string]bool
	audioIngress                 *audio.Ingress
	audioSockets                 map[string]*deviceSocket
	xiaozhiSockets               map[string]*xiaozhiDeviceSocket
	officialStackChanSockets     map[string]*deviceSocket
	audioCaptureFrames           []AudioCaptureFrame
	xiaozhiVoicePipelineRunner   func() xiaozhiVoicePipelineRunner
	xiaozhiVoicePipelineMeta     xiaozhiVoicePipelineMeta
	xiaozhiVoicePipelineASR      providers.ASRAdapter
	xiaozhiProfessionalASR       providers.ASRAdapter
	xiaozhiFastAckTTS            providers.TTSAdapter
	xiaozhiFastAckEnabled        bool
	xiaozhiFastAckDelay          time.Duration
	xiaozhiSTTScreenPolicy       string
	xiaozhiStockProfessional     bool
	xiaozhiProductPlaybackEvents bool
	xiaozhiProductTouchEvents    bool
	xiaozhiProductTouchReactions bool
	xiaozhiProductStateReactions bool
	xiaozhiListenMaxDurationMS   int64
	bodySceneStepDelay           time.Duration
	wakeWordConfigPath           string
	voiceModeConfig              string
	roleplayProfileConfig        string
	roleplayScenarioConfig       string
	roleplayMemoryConfigured     bool
	roleplayMemoryHintsConfig    []string
	roleplayMemoryFindings       []personality.MemoryFinding
	professionalUserIDConfig     string
	professionalWorkspaceConfig  string
	professionalQueryScopeConfig string
	professionalReadRecords      map[string]ProfessionalReadRecord
	professionalReadRecordSeq    uint64
	workspaceUploadJobs          map[string]WorkspaceUploadJob
	workspaceUploadJobSeq        uint64
	workspaceIndexJobs           map[string]WorkspaceIndexJob
	workspaceIndexJobSeq         uint64
	workspaceSources             map[string]WorkspaceSource
	workspaceSourceSeq           uint64
	workspaceDocuments           map[string]WorkspaceDocument
	workspaceDocumentSeq         uint64
	workspaceDeviceBindings      map[string]WorkspaceDeviceBinding
	workspaceDeviceBindingSeq    uint64
	workspaceDocumentStoreDir    string
	workspaceDocumentMaxBytes    int64
	voiceChainModeConfig         string
	cascadeASRProfileConfig      string
	cascadeLLMProfileConfig      string
	fixedTTSProfileConfig        string
	realtimeProviderConfig       string
	voiceCloneProfileConfig      string
	gatewayProfileConfig         string
	cloudVoiceProfileConfig      string
	cloudVoiceEnv                []string
	macLocalGatewayURL           string
	publicGatewayURL             string
}

type ServerOptions struct {
	VoiceProvider                providers.VoiceProvider
	V21Client                    v21adapter.Client
	V21Timeout                   time.Duration
	XiaozhiVoicePipelineAdapters *providers.VoicePipelineAdapters
	AudioIngressConfig           audio.IngressConfig
	XiaozhiStockProfessional     bool
	XiaozhiProductPlaybackEvents bool
	XiaozhiProductTouchEvents    bool
	XiaozhiProductTouchReactions bool
	XiaozhiProductStateReactions bool
	XiaozhiListenMaxDuration     time.Duration
	XiaozhiSTTScreenPolicy       string
	BodySceneStepDelay           time.Duration
	WakeWordConfigPath           string
	WorkspaceDocumentStoreDir    string
	WorkspaceDocumentMaxBytes    int64
	GatewayProfile               string
	CloudVoiceProfile            string
	CloudVoiceEnv                []string
	MacLocalGatewayURL           string
	PublicGatewayURL             string
}

type xiaozhiVoicePipelineMeta struct {
	Selection     providers.VoicePipelineSelection
	ExecutionMode string
}

type MockTurnRequest struct {
	DeviceID  string        `json:"device_id"`
	Text      string        `json:"text,omitempty"`
	Mode      protocol.Mode `json:"mode,omitempty"`
	TraceID   string        `json:"trace_id,omitempty"`
	SessionID string        `json:"session_id,omitempty"`
}

type MockTurnResponse struct {
	TraceID   string              `json:"trace_id"`
	SessionID string              `json:"session_id"`
	DeviceID  string              `json:"device_id"`
	Events    []protocol.Envelope `json:"events"`
}

type RealtimeSessionRequest struct {
	DeviceID  string        `json:"device_id"`
	Text      string        `json:"text,omitempty"`
	Mode      protocol.Mode `json:"mode,omitempty"`
	TraceID   string        `json:"trace_id,omitempty"`
	SessionID string        `json:"session_id,omitempty"`
}

type RealtimeSessionCancelRequest struct {
	DeviceID  string                      `json:"device_id"`
	Mode      protocol.Mode               `json:"mode,omitempty"`
	TraceID   string                      `json:"trace_id,omitempty"`
	SessionID string                      `json:"session_id,omitempty"`
	StreamID  string                      `json:"stream_id,omitempty"`
	Reason    providers.VoiceCancelReason `json:"reason,omitempty"`
}

type RealtimeSessionResponse struct {
	TraceID   string              `json:"trace_id"`
	SessionID string              `json:"session_id"`
	DeviceID  string              `json:"device_id"`
	Provider  string              `json:"provider"`
	Status    string              `json:"status"`
	Events    []protocol.Envelope `json:"events"`
}

type FastCompanionLocalAudioResult struct {
	ASRProvider          string                         `json:"asr_provider,omitempty"`
	FirstPartialMS       int64                          `json:"first_partial_ms,omitempty"`
	FinalTranscriptChars int                            `json:"final_transcript_chars,omitempty"`
	Frames               []FastCompanionLocalAudioFrame `json:"frames,omitempty"`
}

type FastCompanionLocalAudioFrame struct {
	Seq          uint64  `json:"seq,omitempty"`
	Codec        string  `json:"codec,omitempty"`
	SampleRateHz int     `json:"sample_rate_hz,omitempty"`
	Channels     int     `json:"channels,omitempty"`
	DurationMS   int     `json:"duration_ms,omitempty"`
	ByteCount    int     `json:"byte_count,omitempty"`
	RMS          float64 `json:"rms,omitempty"`
	DataBase64   string  `json:"data_base64,omitempty"`
}

type FastCompanionTurnRequest struct {
	DeviceID   string                          `json:"device_id"`
	Mode       protocol.Mode                   `json:"mode,omitempty"`
	TraceID    string                          `json:"trace_id,omitempty"`
	SessionID  string                          `json:"session_id,omitempty"`
	Roleplay   RoleplayProfileSelectionRequest `json:"roleplay,omitempty"`
	LocalAudio FastCompanionLocalAudioResult   `json:"local_audio,omitempty"`
}

type FastCompanionTurnResponse struct {
	TraceID                    string                 `json:"trace_id"`
	SessionID                  string                 `json:"session_id"`
	DeviceID                   string                 `json:"device_id"`
	Mode                       protocol.Mode          `json:"mode"`
	Status                     string                 `json:"status"`
	Route                      string                 `json:"route"`
	AudioFrontend              string                 `json:"audio_frontend"`
	TextStreamProvider         string                 `json:"text_stream_provider"`
	ProviderFamily             string                 `json:"provider_family"`
	TextStreamExecuted         bool                   `json:"text_stream_executed"`
	TextStreamFallbackUsed     bool                   `json:"text_stream_fallback_used,omitempty"`
	TextStreamFallbackProvider string                 `json:"text_stream_fallback_provider,omitempty"`
	TextStreamFallbackReason   string                 `json:"text_stream_fallback_reason,omitempty"`
	Roleplay                   RoleplayRuntimeSummary `json:"roleplay"`
	Events                     []protocol.Envelope    `json:"events"`
}

type RoleplayProfileOption struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	Status         string   `json:"status"`
	Default        bool     `json:"default,omitempty"`
	Description    string   `json:"description,omitempty"`
	VoiceHint      string   `json:"voice_hint,omitempty"`
	ExpressionHint string   `json:"expression_hint,omitempty"`
	PromptParts    []string `json:"prompt_parts,omitempty"`
}

type RoleplayProfileSelectionRequest struct {
	RoleplayProfile   string   `json:"roleplay_profile,omitempty"`
	Scenario          string   `json:"scenario,omitempty"`
	VoiceCloneProfile string   `json:"voice_clone_profile,omitempty"`
	MemoryHints       []string `json:"memory_hints,omitempty"`
	ClearMemory       bool     `json:"clear_memory,omitempty"`
}

type RoleplayProfileResponse struct {
	SchemaVersion             string                  `json:"schema_version"`
	Service                   string                  `json:"service"`
	SelectedRoleplayProfile   string                  `json:"selected_roleplay_profile"`
	SelectedScenario          string                  `json:"selected_scenario"`
	SelectedVoiceCloneProfile string                  `json:"selected_voice_clone_profile"`
	Memory                    personality.MemoryState `json:"memory"`
	Runtime                   RoleplayRuntimeSummary  `json:"runtime"`
	ExpressionPlan            RoleplayExpressionPlan  `json:"expression_plan"`
	Profiles                  []RoleplayProfileOption `json:"profiles"`
	Scenarios                 []RoleplayProfileOption `json:"scenarios"`
}

type RoleplayExpressionPlan struct {
	SchemaVersion     string                      `json:"schema_version"`
	Adapter           string                      `json:"adapter"`
	DeliveryPolicy    string                      `json:"delivery_policy"`
	RoleplayProfile   string                      `json:"roleplay_profile"`
	Scenario          string                      `json:"scenario"`
	VoiceCloneProfile string                      `json:"voice_clone_profile"`
	MemoryReady       bool                        `json:"memory_ready"`
	MemoryHintCount   int                         `json:"memory_hint_count"`
	ActionCount       int                         `json:"action_count"`
	PacketCount       int                         `json:"packet_count"`
	PhysicalAccepted  bool                        `json:"physical_accepted"`
	Actions           []RoleplayExpressionAction  `json:"actions"`
	Surfaces          map[string]string           `json:"surfaces,omitempty"`
	Redaction         RoleplayExpressionRedaction `json:"redaction"`
}

type RoleplayExpressionAction struct {
	Phase            string            `json:"phase"`
	Event            string            `json:"event"`
	Value            string            `json:"value"`
	PacketCount      int               `json:"packet_count"`
	PhysicalAccepted bool              `json:"physical_accepted"`
	Surfaces         map[string]string `json:"surfaces,omitempty"`
}

type RoleplayExpressionRedaction struct {
	PromptTextStored       bool `json:"prompt_text_stored"`
	MemoryTextStored       bool `json:"memory_text_stored"`
	TranscriptStored       bool `json:"transcript_stored"`
	ProviderOutputStored   bool `json:"provider_output_stored"`
	AudioStored            bool `json:"audio_stored"`
	VoiceCloneSampleStored bool `json:"voice_clone_sample_stored"`
}

type RoleplayRuntimeSummary struct {
	SchemaVersion            string   `json:"schema_version"`
	Mode                     string   `json:"mode"`
	RoleplayProfile          string   `json:"roleplay_profile"`
	Scenario                 string   `json:"scenario"`
	VoiceCloneProfile        string   `json:"voice_clone_profile"`
	SoulPromptInputReady     bool     `json:"soul_prompt_input_ready"`
	PromptParts              []string `json:"prompt_parts,omitempty"`
	MemoryPolicy             string   `json:"memory_policy"`
	MemoryConfigured         bool     `json:"memory_configured"`
	MemoryPromptInputReady   bool     `json:"memory_prompt_input_ready"`
	MemoryHintCount          int      `json:"memory_hint_count"`
	PromptComposed           bool     `json:"prompt_composed"`
	PromptStored             bool     `json:"prompt_stored"`
	MemoryTextStored         bool     `json:"memory_text_stored"`
	TranscriptStored         bool     `json:"transcript_stored"`
	ProviderOutputStored     bool     `json:"provider_output_stored"`
	VoiceCloneSampleStored   bool     `json:"voice_clone_sample_stored"`
	ProfessionalRouteAllowed bool     `json:"professional_route_allowed"`
	V21Executed              bool     `json:"v21_executed"`
}

type roleplayDeviceState struct {
	Profile          string
	Scenario         string
	SoulReady        bool
	MemoryReady      bool
	MemoryHintCount  int
	PhysicalAccepted bool
}

type ProfessionalWorkspaceOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Status      string `json:"status"`
	Default     bool   `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

type ProfessionalWorkspaceSelectionRequest struct {
	UserID      string `json:"user_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	QueryScope  string `json:"query_scope,omitempty"`
}

type ProfessionalWorkspaceResponse struct {
	SchemaVersion          string                         `json:"schema_version"`
	Service                string                         `json:"service"`
	AdapterContractVersion string                         `json:"adapter_contract_version"`
	SelectedUserID         string                         `json:"selected_user_id"`
	SelectedWorkspaceID    string                         `json:"selected_workspace_id"`
	SelectedQueryScope     string                         `json:"selected_query_scope"`
	WorkspaceStatus        string                         `json:"workspace_status"`
	PrivacyScope           string                         `json:"privacy_scope"`
	V21ExecutionAllowed    bool                           `json:"v21_execution_allowed"`
	Runtime                ProfessionalWorkspaceRuntime   `json:"runtime"`
	QueryScopes            []ProfessionalWorkspaceOption  `json:"query_scopes"`
	Redaction              ProfessionalWorkspaceRedaction `json:"redaction"`
}

type ProfessionalWorkspaceRuntime struct {
	SchemaVersion                      string                        `json:"schema_version"`
	Mode                               string                        `json:"mode"`
	UserID                             string                        `json:"user_id"`
	WorkspaceID                        string                        `json:"workspace_id"`
	QueryScope                         string                        `json:"query_scope"`
	PrivacyScope                       string                        `json:"privacy_scope"`
	AdapterContractVersion             string                        `json:"adapter_contract_version"`
	WorkspaceStatus                    string                        `json:"workspace_status"`
	QueryScopeReadiness                string                        `json:"query_scope_readiness,omitempty"`
	SourceScopeCounts                  map[string]int                `json:"source_scope_counts,omitempty"`
	IndexingRequestedSourceScopeCounts map[string]int                `json:"indexing_requested_source_scope_counts,omitempty"`
	SearchableSourceScopeCounts        map[string]int                `json:"searchable_source_scope_counts,omitempty"`
	UploadAPIReady                     bool                          `json:"upload_api_ready"`
	IndexingAPIReady                   bool                          `json:"indexing_api_ready"`
	QueryScopeReady                    bool                          `json:"query_scope_ready"`
	V21ExecutionAllowed                bool                          `json:"v21_execution_allowed"`
	DeviceBindingPolicy                string                        `json:"device_binding_policy"`
	BoundDeviceCount                   int                           `json:"bound_device_count"`
	ActiveDeviceBindingCount           int                           `json:"active_device_binding_count"`
	RevokedDeviceBindingCount          int                           `json:"revoked_device_binding_count"`
	DeviceBindingSummary               WorkspaceDeviceBindingSummary `json:"device_binding_summary"`
}

type ProfessionalWorkspaceRedaction struct {
	DocumentTextStored    bool `json:"document_text_stored"`
	QueryTextStored       bool `json:"query_text_stored"`
	RetrievedTextStored   bool `json:"retrieved_text_stored"`
	FullURLStored         bool `json:"full_url_stored"`
	LocalPathStored       bool `json:"local_path_stored"`
	CredentialValueStored bool `json:"credential_value_stored"`
	ProviderOutputStored  bool `json:"provider_output_stored"`
	VoiceTranscriptStored bool `json:"voice_transcript_stored"`
}

type ProfessionalReadRecordsResponse struct {
	SchemaVersion string                         `json:"schema_version"`
	Service       string                         `json:"service"`
	Status        string                         `json:"status"`
	Records       []ProfessionalReadRecord       `json:"records"`
	Redaction     ProfessionalWorkspaceRedaction `json:"redaction"`
}

type ProfessionalQueryRequest struct {
	DeviceID    string `json:"device_id,omitempty"`
	Text        string `json:"text,omitempty"`
	Utterance   string `json:"utterance,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	QueryScope  string `json:"query_scope,omitempty"`
	TraceID     string `json:"trace_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
}

type ProfessionalQueryResponse struct {
	SchemaVersion          string                                       `json:"schema_version"`
	Service                string                                       `json:"service"`
	Status                 string                                       `json:"status"`
	FailureCode            string                                       `json:"failure_code,omitempty"`
	TraceID                string                                       `json:"trace_id"`
	SessionID              string                                       `json:"session_id"`
	DeviceID               string                                       `json:"device_id,omitempty"`
	Mode                   string                                       `json:"mode"`
	Route                  string                                       `json:"route"`
	AdapterContractVersion string                                       `json:"adapter_contract_version"`
	Workspace              ProfessionalWorkspaceRuntime                 `json:"workspace"`
	DeviceBinding          ProfessionalQueryDeviceBinding               `json:"device_binding"`
	ReadRecordID           string                                       `json:"read_record_id,omitempty"`
	ReadRecordStatus       string                                       `json:"read_record_status,omitempty"`
	Answer                 *protocol.ControlEventPayload                `json:"answer,omitempty"`
	EvidenceReport         *v21adapter.ProfessionalBridgeEvidenceReport `json:"evidence_report,omitempty"`
	Events                 []protocol.Envelope                          `json:"events"`
	Redaction              ProfessionalWorkspaceRedaction               `json:"redaction"`
}

type ProfessionalQueryDeviceBinding struct {
	Policy      string `json:"policy"`
	Status      string `json:"status"`
	BindingID   string `json:"binding_id,omitempty"`
	FailureCode string `json:"failure_code,omitempty"`
}

type ProfessionalReadRecord struct {
	RecordID          string                         `json:"record_id"`
	Status            string                         `json:"status"`
	FailureCode       string                         `json:"failure_code,omitempty"`
	TraceID           string                         `json:"trace_id,omitempty"`
	SessionID         string                         `json:"session_id,omitempty"`
	DeviceID          string                         `json:"device_id,omitempty"`
	UserID            string                         `json:"user_id"`
	WorkspaceID       string                         `json:"workspace_id"`
	QueryScope        string                         `json:"query_scope"`
	PrivacyScope      string                         `json:"privacy_scope"`
	LatencyProfile    string                         `json:"latency_profile"`
	AnswerStyle       string                         `json:"answer_style"`
	UtteranceBucket   string                         `json:"utterance_bucket"`
	SourceScopeCounts map[string]int                 `json:"source_scope_counts,omitempty"`
	WorkspaceStatus   string                         `json:"workspace_status,omitempty"`
	StartedAtMS       int64                          `json:"started_at_ms"`
	CompletedAtMS     int64                          `json:"completed_at_ms,omitempty"`
	Redaction         ProfessionalWorkspaceRedaction `json:"redaction"`
}

type WorkspaceUploadJobRequest struct {
	JobID         string `json:"job_id,omitempty"`
	Action        string `json:"action,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	WorkspaceID   string `json:"workspace_id,omitempty"`
	SourceScope   string `json:"source_scope,omitempty"`
	SourceKind    string `json:"source_kind,omitempty"`
	DocumentLabel string `json:"document_label,omitempty"`
	ContentType   string `json:"content_type,omitempty"`
	SizeBytes     int64  `json:"size_bytes,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
	DeviceID      string `json:"device_id,omitempty"`
}

type WorkspaceUploadJobsResponse struct {
	SchemaVersion string                      `json:"schema_version"`
	Service       string                      `json:"service"`
	Status        string                      `json:"status"`
	Jobs          []WorkspaceUploadJob        `json:"jobs"`
	Redaction     WorkspaceUploadJobRedaction `json:"redaction"`
	Findings      []WorkspaceUploadJobFinding `json:"findings,omitempty"`
}

type WorkspaceUploadJob struct {
	JobID            string                      `json:"job_id"`
	SourceID         string                      `json:"source_id,omitempty"`
	DocumentID       string                      `json:"document_id,omitempty"`
	DocumentHash     string                      `json:"document_hash,omitempty"`
	StorageStatus    string                      `json:"storage_status,omitempty"`
	UserID           string                      `json:"user_id"`
	WorkspaceID      string                      `json:"workspace_id"`
	SourceScope      string                      `json:"source_scope"`
	SourceKind       string                      `json:"source_kind"`
	DocumentLabel    string                      `json:"document_label"`
	ContentType      string                      `json:"content_type,omitempty"`
	SizeBytes        int64                       `json:"size_bytes,omitempty"`
	Status           string                      `json:"status"`
	IndexStatus      string                      `json:"index_status"`
	Attempt          int                         `json:"attempt"`
	CreatedAtMS      int64                       `json:"created_at_ms"`
	UpdatedAtMS      int64                       `json:"updated_at_ms"`
	TraceID          string                      `json:"trace_id,omitempty"`
	SessionID        string                      `json:"session_id,omitempty"`
	DeviceID         string                      `json:"device_id,omitempty"`
	UploadAPIReady   bool                        `json:"upload_api_ready"`
	ImportAPIReady   bool                        `json:"import_api_ready"`
	IndexingAPIReady bool                        `json:"indexing_api_ready"`
	ExecutionStarted bool                        `json:"execution_started"`
	RetryAllowed     bool                        `json:"retry_allowed"`
	DeleteAllowed    bool                        `json:"delete_allowed"`
	Redaction        WorkspaceUploadJobRedaction `json:"redaction"`
	Findings         []WorkspaceUploadJobFinding `json:"findings,omitempty"`
}

type WorkspaceIndexJobRequest struct {
	IndexJobID string `json:"index_job_id,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	JobID      string `json:"job_id,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	DeviceID   string `json:"device_id,omitempty"`
}

type WorkspaceIndexJobsResponse struct {
	SchemaVersion string                      `json:"schema_version"`
	Service       string                      `json:"service"`
	Status        string                      `json:"status"`
	Jobs          []WorkspaceIndexJob         `json:"jobs"`
	Redaction     WorkspaceUploadJobRedaction `json:"redaction"`
	Findings      []WorkspaceUploadJobFinding `json:"findings,omitempty"`
}

type WorkspaceIndexJob struct {
	IndexJobID             string                      `json:"index_job_id"`
	DocumentID             string                      `json:"document_id"`
	SourceID               string                      `json:"source_id"`
	JobID                  string                      `json:"job_id"`
	DocumentHash           string                      `json:"document_hash"`
	StorageStatus          string                      `json:"storage_status"`
	UserID                 string                      `json:"user_id"`
	WorkspaceID            string                      `json:"workspace_id"`
	SourceScope            string                      `json:"source_scope"`
	SourceKind             string                      `json:"source_kind"`
	DocumentLabel          string                      `json:"document_label"`
	ContentType            string                      `json:"content_type,omitempty"`
	SizeBytes              int64                       `json:"size_bytes"`
	Status                 string                      `json:"status"`
	IndexStatus            string                      `json:"index_status"`
	AdapterContractVersion string                      `json:"adapter_contract_version"`
	CreatedAtMS            int64                       `json:"created_at_ms"`
	UpdatedAtMS            int64                       `json:"updated_at_ms"`
	TraceID                string                      `json:"trace_id,omitempty"`
	SessionID              string                      `json:"session_id,omitempty"`
	DeviceID               string                      `json:"device_id,omitempty"`
	V21ExecutionAllowed    bool                        `json:"v21_execution_allowed"`
	ExecutionStarted       bool                        `json:"execution_started"`
	Redaction              WorkspaceUploadJobRedaction `json:"redaction"`
	Findings               []WorkspaceUploadJobFinding `json:"findings,omitempty"`
}

type WorkspaceSourcesResponse struct {
	SchemaVersion string                      `json:"schema_version"`
	Service       string                      `json:"service"`
	Status        string                      `json:"status"`
	Sources       []WorkspaceSource           `json:"sources"`
	Summary       WorkspaceSourceSummary      `json:"summary"`
	Redaction     WorkspaceUploadJobRedaction `json:"redaction"`
	Findings      []WorkspaceUploadJobFinding `json:"findings,omitempty"`
}

type WorkspaceSource struct {
	SourceID          string                      `json:"source_id"`
	JobID             string                      `json:"job_id"`
	DocumentID        string                      `json:"document_id,omitempty"`
	DocumentHash      string                      `json:"document_hash,omitempty"`
	StorageStatus     string                      `json:"storage_status,omitempty"`
	UserID            string                      `json:"user_id"`
	WorkspaceID       string                      `json:"workspace_id"`
	SourceScope       string                      `json:"source_scope"`
	SourceKind        string                      `json:"source_kind"`
	DocumentLabel     string                      `json:"document_label"`
	ContentType       string                      `json:"content_type,omitempty"`
	SizeBytes         int64                       `json:"size_bytes,omitempty"`
	Readiness         string                      `json:"readiness"`
	IndexStatus       string                      `json:"index_status"`
	CreatedAtMS       int64                       `json:"created_at_ms"`
	UpdatedAtMS       int64                       `json:"updated_at_ms"`
	TraceID           string                      `json:"trace_id,omitempty"`
	SessionID         string                      `json:"session_id,omitempty"`
	DeviceID          string                      `json:"device_id,omitempty"`
	MetadataOnly      bool                        `json:"metadata_only"`
	StoredLocal       bool                        `json:"stored_local"`
	IndexingRequested bool                        `json:"indexing_requested"`
	Searchable        bool                        `json:"searchable"`
	Deleted           bool                        `json:"deleted"`
	Redaction         WorkspaceUploadJobRedaction `json:"redaction"`
	Findings          []WorkspaceUploadJobFinding `json:"findings,omitempty"`
}

type WorkspaceSourceSummary struct {
	TotalSources                       int            `json:"total_sources"`
	MetadataOnlySources                int            `json:"metadata_only_sources"`
	StoredLocalSources                 int            `json:"stored_local_sources"`
	IndexingRequestedSources           int            `json:"indexing_requested_sources"`
	SearchableSources                  int            `json:"searchable_sources"`
	DeletedSources                     int            `json:"deleted_sources"`
	SourceScopeCounts                  map[string]int `json:"source_scope_counts"`
	StoredLocalSourceScopeCounts       map[string]int `json:"stored_local_source_scope_counts"`
	IndexingRequestedSourceScopeCounts map[string]int `json:"indexing_requested_source_scope_counts"`
	SearchableSourceScopeCounts        map[string]int `json:"searchable_source_scope_counts"`
	WorkspaceStatus                    string         `json:"workspace_status"`
}

type WorkspaceUploadJobRedaction struct {
	DocumentTextStored    bool `json:"document_text_stored"`
	DocumentBytesStored   bool `json:"document_bytes_stored"`
	Base64PayloadStored   bool `json:"base64_payload_stored"`
	ImportURLStored       bool `json:"import_url_stored"`
	LocalPathStored       bool `json:"local_path_stored"`
	CredentialValueStored bool `json:"credential_value_stored"`
	ProviderOutputStored  bool `json:"provider_output_stored"`
}

type WorkspaceUploadJobFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type WorkspaceDocumentsResponse struct {
	SchemaVersion string                      `json:"schema_version"`
	Service       string                      `json:"service"`
	Status        string                      `json:"status"`
	Documents     []WorkspaceDocument         `json:"documents"`
	Jobs          []WorkspaceUploadJob        `json:"jobs,omitempty"`
	Sources       []WorkspaceSource           `json:"sources,omitempty"`
	Redaction     WorkspaceDocumentRedaction  `json:"redaction"`
	Findings      []WorkspaceUploadJobFinding `json:"findings,omitempty"`
}

type WorkspaceDocument struct {
	DocumentID    string                      `json:"document_id"`
	SourceID      string                      `json:"source_id"`
	JobID         string                      `json:"job_id"`
	UserID        string                      `json:"user_id"`
	WorkspaceID   string                      `json:"workspace_id"`
	SourceScope   string                      `json:"source_scope"`
	SourceKind    string                      `json:"source_kind"`
	DocumentLabel string                      `json:"document_label"`
	ContentType   string                      `json:"content_type,omitempty"`
	SizeBytes     int64                       `json:"size_bytes"`
	DocumentHash  string                      `json:"document_hash"`
	Status        string                      `json:"status"`
	StorageStatus string                      `json:"storage_status"`
	IndexStatus   string                      `json:"index_status"`
	Readiness     string                      `json:"readiness"`
	CreatedAtMS   int64                       `json:"created_at_ms"`
	UpdatedAtMS   int64                       `json:"updated_at_ms"`
	TraceID       string                      `json:"trace_id,omitempty"`
	SessionID     string                      `json:"session_id,omitempty"`
	DeviceID      string                      `json:"device_id,omitempty"`
	Redaction     WorkspaceDocumentRedaction  `json:"redaction"`
	Findings      []WorkspaceUploadJobFinding `json:"findings,omitempty"`
}

type WorkspaceDocumentRedaction struct {
	DocumentTextStored     bool `json:"document_text_stored"`
	DocumentBytesStored    bool `json:"document_bytes_stored"`
	DocumentBytesReturned  bool `json:"document_bytes_returned"`
	OriginalFilenameStored bool `json:"original_filename_stored"`
	Base64PayloadStored    bool `json:"base64_payload_stored"`
	ImportURLStored        bool `json:"import_url_stored"`
	LocalPathStored        bool `json:"local_path_stored"`
	LocalPathReturned      bool `json:"local_path_returned"`
	CredentialValueStored  bool `json:"credential_value_stored"`
	ProviderOutputStored   bool `json:"provider_output_stored"`
}
