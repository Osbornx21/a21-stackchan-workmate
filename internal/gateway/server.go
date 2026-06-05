package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/audio/opuscodec"
	"a21.local/a21/internal/buildinfo"
	"a21.local/a21/internal/personality"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	stackchantransport "a21.local/a21/internal/transport/stackchan"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"a21.local/a21/internal/v21adapter"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
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
	DeviceID            string `json:"device_id"`
	TraceID             string `json:"trace_id"`
	SessionID           string `json:"session_id"`
	ColdBootWithoutUSB  bool   `json:"cold_boot_without_usb"`
	PowerButtonStarted  bool   `json:"power_button_started"`
	GatewayConnected    bool   `json:"gateway_connected"`
	XiaozhiSocketOnline bool   `json:"xiaozhi_socket_online"`
	StandaloneRuntimeOK bool   `json:"standalone_runtime_ok"`
	Observer            string `json:"observer"`
}

type PowerLifecycleAcceptanceResponse struct {
	SchemaVersion     string   `json:"schema_version"`
	TraceID           string   `json:"trace_id"`
	SessionID         string   `json:"session_id"`
	DeviceID          string   `json:"device_id"`
	Status            string   `json:"status"`
	Observer          string   `json:"observer"`
	AcceptedSurfaces  []string `json:"accepted_surfaces"`
	PhysicalAccepted  bool     `json:"physical_accepted"`
	ResultRedacted    bool     `json:"result_redacted"`
	AcceptanceEventMS int64    `json:"acceptance_event_ms"`
}

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
	mux.HandleFunc("/simulator", s.handleSimulator)
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
	state := s.powerLifecycle(req.DeviceID)
	if state.ConnectionStatus != "online" || !state.XiaozhiWSOnline {
		http.Error(w, "online device and Xiaozhi websocket evidence are required before power lifecycle acceptance", http.StatusConflict)
		return
	}

	nowMS := s.now().UnixMilli()
	marker := "power_lifecycle.physical_acceptance.accepted"
	s.recordTrace(traceID, sessionID, req.DeviceID, marker, nowMS)
	s.recordPowerLifecyclePhysicalAcceptance(req.DeviceID, traceID, sessionID, observer, nowMS, marker)
	writeJSON(w, http.StatusOK, PowerLifecycleAcceptanceResponse{
		SchemaVersion:     "a21.gateway.power_lifecycle_acceptance.v1",
		TraceID:           traceID,
		SessionID:         sessionID,
		DeviceID:          req.DeviceID,
		Status:            "accepted",
		Observer:          observer,
		AcceptedSurfaces:  []string{"no_cable_cold_boot", "physical_power_button", "gateway_reconnect", "xiaozhi_socket"},
		PhysicalAccepted:  true,
		ResultRedacted:    true,
		AcceptanceEventMS: nowMS,
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
			Label:          "A21 desk workmate",
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

func professionalWorkspaceRedaction() ProfessionalWorkspaceRedaction {
	return ProfessionalWorkspaceRedaction{
		DocumentTextStored:    false,
		QueryTextStored:       false,
		RetrievedTextStored:   false,
		FullURLStored:         false,
		LocalPathStored:       false,
		CredentialValueStored: false,
		ProviderOutputStored:  false,
		VoiceTranscriptStored: false,
	}
}

func decodeProfessionalQueryRequest(r *http.Request) (ProfessionalQueryRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return ProfessionalQueryRequest{}, fmt.Errorf("invalid json")
	}
	for key := range raw {
		if professionalQueryForbiddenKey(key) {
			return ProfessionalQueryRequest{}, fmt.Errorf("professional query request must not include document text, evidence bodies, provider output, URLs, paths, credentials, base64, transcript, or audio payload fields")
		}
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return ProfessionalQueryRequest{}, fmt.Errorf("invalid json")
	}
	var req ProfessionalQueryRequest
	if err := json.Unmarshal(encoded, &req); err != nil {
		return ProfessionalQueryRequest{}, fmt.Errorf("invalid json")
	}
	return req, nil
}

func professionalQueryForbiddenKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if workspaceUploadJobForbiddenKey(key) {
		return true
	}
	for _, forbidden := range []string{
		"answer",
		"fast_answer",
		"screen_card",
		"screen_cards",
		"speech_block",
		"speech_blocks",
		"follow_up",
		"follow_ups",
		"quote",
		"quotes",
		"retrieval",
		"retrieved",
		"source_text",
		"source_body",
	} {
		if key == forbidden {
			return true
		}
	}
	return false
}

func professionalQueryUtterance(req ProfessionalQueryRequest) string {
	if strings.TrimSpace(req.Text) != "" {
		return strings.TrimSpace(req.Text)
	}
	return strings.TrimSpace(req.Utterance)
}

func (s *Server) startProfessionalReadRecord(request v21adapter.QueryRequest, utteranceBucket string) string {
	nowMS := s.now().UnixMilli()
	record := ProfessionalReadRecord{
		Status:          "started",
		TraceID:         safeOptionalWorkspaceLabel(request.TraceID),
		SessionID:       safeOptionalWorkspaceLabel(request.SessionID),
		DeviceID:        safeOptionalWorkspaceLabel(request.DeviceID),
		UserID:          defaultProfessionalLabel(request.UserID, v21adapter.DefaultUserID),
		WorkspaceID:     defaultProfessionalLabel(request.WorkspaceID, v21adapter.DefaultWorkspaceID),
		QueryScope:      defaultProfessionalQueryScope(request.QueryScope),
		PrivacyScope:    defaultProfessionalPrivacyScope(request.PrivacyScope),
		LatencyProfile:  defaultProfessionalLatencyProfile(request.LatencyProfile),
		AnswerStyle:     defaultProfessionalAnswerStyle(request.AnswerStyle),
		UtteranceBucket: defaultProfessionalUtteranceBucket(utteranceBucket),
		StartedAtMS:     nowMS,
		Redaction:       professionalWorkspaceRedaction(),
	}
	s.mu.Lock()
	s.professionalReadRecordSeq++
	record.RecordID = fmt.Sprintf("a21-professional-read-%06d", s.professionalReadRecordSeq)
	s.professionalReadRecords[record.RecordID] = record
	s.mu.Unlock()
	if record.TraceID != "" {
		s.recordTrace(record.TraceID, record.SessionID, record.DeviceID, "professional.read_record.started", nowMS)
	}
	return record.RecordID
}

func (s *Server) completeProfessionalReadRecord(recordID string, response v21adapter.QueryResponse) {
	s.updateProfessionalReadRecord(recordID, "completed", "", safeProfessionalSourceScopeCounts(response.SourceScopeCounts), safeProfessionalWorkspaceStatus(response.WorkspaceStatus))
}

func (s *Server) failProfessionalReadRecord(recordID string, code string) {
	s.updateProfessionalReadRecord(recordID, "failed", safeProfessionalReadFailureCode(code), nil, "")
}

func (s *Server) updateProfessionalReadRecord(recordID string, status string, failureCode string, counts map[string]int, workspaceStatus string) {
	recordID = strings.TrimSpace(recordID)
	if recordID == "" {
		return
	}
	nowMS := s.now().UnixMilli()
	s.mu.Lock()
	record, ok := s.professionalReadRecords[recordID]
	if !ok {
		s.mu.Unlock()
		return
	}
	record.Status = status
	record.FailureCode = failureCode
	record.CompletedAtMS = nowMS
	record.SourceScopeCounts = counts
	record.WorkspaceStatus = workspaceStatus
	s.professionalReadRecords[recordID] = record
	s.mu.Unlock()
	if record.TraceID != "" {
		marker := "professional.read_record." + status
		s.recordTrace(record.TraceID, record.SessionID, record.DeviceID, marker, nowMS)
	}
}

func (s *Server) professionalReadRecordsResponse(recordID string, traceID string, sessionID string) ProfessionalReadRecordsResponse {
	records := s.professionalReadRecordsSnapshot(recordID, traceID, sessionID)
	status := "ok"
	if (strings.TrimSpace(recordID) != "" || strings.TrimSpace(traceID) != "" || strings.TrimSpace(sessionID) != "") && len(records) == 0 {
		status = "not_found"
	}
	return ProfessionalReadRecordsResponse{
		SchemaVersion: ProfessionalReadRecordsSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        status,
		Records:       records,
		Redaction:     professionalWorkspaceRedaction(),
	}
}

func (s *Server) professionalReadRecordsSnapshot(recordID string, traceID string, sessionID string) []ProfessionalReadRecord {
	recordID = strings.TrimSpace(recordID)
	traceID = strings.TrimSpace(traceID)
	sessionID = strings.TrimSpace(sessionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	records := make([]ProfessionalReadRecord, 0, len(s.professionalReadRecords))
	keys := make([]string, 0, len(s.professionalReadRecords))
	if recordID != "" {
		keys = append(keys, recordID)
	} else {
		for key := range s.professionalReadRecords {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}
	for _, key := range keys {
		record, ok := s.professionalReadRecords[key]
		if !ok {
			continue
		}
		if traceID != "" && record.TraceID != traceID {
			continue
		}
		if sessionID != "" && record.SessionID != sessionID {
			continue
		}
		records = append(records, copyProfessionalReadRecord(record))
	}
	return records
}

func copyProfessionalReadRecord(record ProfessionalReadRecord) ProfessionalReadRecord {
	if len(record.SourceScopeCounts) > 0 {
		counts := make(map[string]int, len(record.SourceScopeCounts))
		for scope, count := range record.SourceScopeCounts {
			counts[scope] = count
		}
		record.SourceScopeCounts = counts
	}
	return record
}

func defaultProfessionalPrivacyScope(scope string) string {
	if strings.TrimSpace(scope) == "professional_only" {
		return "professional_only"
	}
	return "professional_only"
}

func defaultProfessionalLatencyProfile(profile string) string {
	switch strings.TrimSpace(profile) {
	case "fast_first":
		return "fast_first"
	default:
		return "fast_first"
	}
}

func defaultProfessionalAnswerStyle(style string) string {
	switch strings.TrimSpace(style) {
	case "voice_first_with_citations":
		return "voice_first_with_citations"
	default:
		return "voice_first_with_citations"
	}
}

func defaultProfessionalUtteranceBucket(bucket string) string {
	switch strings.TrimSpace(bucket) {
	case "length_empty", "length_1_16", "length_17_64", "length_65_160", "length_gt_160":
		return strings.TrimSpace(bucket)
	default:
		return "length_empty"
	}
}

func safeProfessionalSourceScopeCounts(counts map[string]int) map[string]int {
	if len(counts) == 0 {
		return nil
	}
	out := make(map[string]int, len(counts))
	for scope, count := range counts {
		switch scope {
		case "public", "personal":
		default:
			return nil
		}
		if count < 0 {
			return nil
		}
		out[scope] = count
	}
	return out
}

func safeProfessionalWorkspaceStatus(status string) string {
	switch strings.TrimSpace(status) {
	case v21adapter.WorkspaceSearchable, "uploaded", "indexing", "failed", "unavailable":
		return strings.TrimSpace(status)
	default:
		return ""
	}
}

func professionalReadFailureCode(err error, queryCtx context.Context) string {
	if errors.Is(err, context.DeadlineExceeded) || (queryCtx != nil && errors.Is(queryCtx.Err(), context.DeadlineExceeded)) {
		return "timeout"
	}
	if class := v21adapter.QueryFailureClassOf(err); class != "" {
		return string(class)
	}
	if statusClass := v21adapter.QueryFailureStatusClassOf(err); statusClass != "" {
		switch statusClass {
		case "4xx", "5xx":
			return "upstream_" + statusClass
		}
	}
	return "query_error"
}

func safeProfessionalReadFailureCode(code string) string {
	switch strings.TrimSpace(code) {
	case "timeout", "contract_invalid", "upstream_status", "no_evidence", "adapter_status", "transport_error", "query_error", "suppressed", "upstream_4xx", "upstream_5xx", "device_unbound", "device_binding_revoked", "device_scope_denied":
		return strings.TrimSpace(code)
	default:
		return "query_error"
	}
}

var errWorkspaceDocumentTooLarge = errors.New("workspace document exceeds A21 local intake limit")

type WorkspaceDocumentUploadRequest struct {
	UserID        string
	WorkspaceID   string
	SourceScope   string
	DocumentLabel string
	ContentType   string
	TraceID       string
	SessionID     string
	DeviceID      string
}

func workspaceDocumentUploadRequestFromForm(r *http.Request) WorkspaceDocumentUploadRequest {
	return WorkspaceDocumentUploadRequest{
		UserID:        strings.TrimSpace(r.FormValue("user_id")),
		WorkspaceID:   strings.TrimSpace(r.FormValue("workspace_id")),
		SourceScope:   strings.TrimSpace(r.FormValue("source_scope")),
		DocumentLabel: strings.TrimSpace(r.FormValue("document_label")),
		ContentType:   strings.TrimSpace(r.FormValue("content_type")),
		TraceID:       strings.TrimSpace(r.FormValue("trace_id")),
		SessionID:     strings.TrimSpace(r.FormValue("session_id")),
		DeviceID:      strings.TrimSpace(r.FormValue("device_id")),
	}
}

func (s *Server) storeWorkspaceDocumentUpload(req WorkspaceDocumentUploadRequest, file multipart.File, header *multipart.FileHeader) (WorkspaceDocumentsResponse, error) {
	userID, workspaceID, _, err := s.resolveProfessionalWorkspace(ProfessionalWorkspaceSelectionRequest{
		UserID:      req.UserID,
		WorkspaceID: req.WorkspaceID,
	})
	if err != nil {
		return WorkspaceDocumentsResponse{}, err
	}
	sourceScope := defaultWorkspaceSourceScope(req.SourceScope)
	if !validWorkspaceSourceScope(sourceScope) {
		return WorkspaceDocumentsResponse{}, fmt.Errorf("source_scope must be personal or public")
	}
	documentLabel := safeWorkspaceDocumentLabel(req.DocumentLabel)
	if documentLabel == "" {
		return WorkspaceDocumentsResponse{}, fmt.Errorf("valid redacted document_label is required")
	}
	contentType := safeWorkspaceContentType(req.ContentType)
	if contentType == "" && header != nil {
		contentType = safeWorkspaceContentType(header.Header.Get("Content-Type"))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	traceID := safeOptionalWorkspaceLabel(req.TraceID)
	sessionID := safeOptionalWorkspaceLabel(req.SessionID)
	deviceID := safeOptionalWorkspaceLabel(req.DeviceID)
	maxBytes := s.workspaceDocumentMaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultWorkspaceDocumentMaxBytes
	}
	hash, sizeBytes, err := s.storeWorkspaceDocumentFile(userID, workspaceID, sourceScope, file, maxBytes)
	if err != nil {
		if errors.Is(err, errWorkspaceDocumentTooLarge) {
			return WorkspaceDocumentsResponse{}, err
		}
		return WorkspaceDocumentsResponse{}, fmt.Errorf("workspace document local storage unavailable")
	}
	nowMS := s.now().UnixMilli()
	s.mu.Lock()
	s.workspaceDocumentSeq++
	documentID := fmt.Sprintf("a21-workspace-document-%06d", s.workspaceDocumentSeq)
	s.workspaceUploadJobSeq++
	jobID := fmt.Sprintf("a21-workspace-job-%06d", s.workspaceUploadJobSeq)
	s.workspaceSourceSeq++
	sourceID := fmt.Sprintf("a21-workspace-source-%06d", s.workspaceSourceSeq)
	documentHash := "sha256:" + hash
	findings := []WorkspaceUploadJobFinding{{
		Code:    "workspace_document_stored_local_pending_index",
		Message: "A21 stored the upload in the local workspace intake store; indexing and V21 execution remain disabled",
	}}
	document := WorkspaceDocument{
		DocumentID:    documentID,
		SourceID:      sourceID,
		JobID:         jobID,
		UserID:        userID,
		WorkspaceID:   workspaceID,
		SourceScope:   sourceScope,
		SourceKind:    "upload",
		DocumentLabel: documentLabel,
		ContentType:   contentType,
		SizeBytes:     sizeBytes,
		DocumentHash:  documentHash,
		Status:        "stored_local_pending_index",
		StorageStatus: "stored_local",
		IndexStatus:   "not_started_no_execute",
		Readiness:     "stored_local_pending_index",
		CreatedAtMS:   nowMS,
		UpdatedAtMS:   nowMS,
		TraceID:       traceID,
		SessionID:     sessionID,
		DeviceID:      deviceID,
		Redaction:     workspaceDocumentStoredRedaction(),
		Findings:      append([]WorkspaceUploadJobFinding(nil), findings...),
	}
	job := WorkspaceUploadJob{
		JobID:            jobID,
		SourceID:         sourceID,
		DocumentID:       documentID,
		DocumentHash:     documentHash,
		StorageStatus:    "stored_local",
		UserID:           userID,
		WorkspaceID:      workspaceID,
		SourceScope:      sourceScope,
		SourceKind:       "upload",
		DocumentLabel:    documentLabel,
		ContentType:      contentType,
		SizeBytes:        sizeBytes,
		Status:           "stored_local_pending_index",
		IndexStatus:      "not_started_no_execute",
		Attempt:          1,
		CreatedAtMS:      nowMS,
		UpdatedAtMS:      nowMS,
		TraceID:          traceID,
		SessionID:        sessionID,
		DeviceID:         deviceID,
		UploadAPIReady:   true,
		ImportAPIReady:   false,
		IndexingAPIReady: false,
		ExecutionStarted: false,
		RetryAllowed:     true,
		DeleteAllowed:    true,
		Redaction:        workspaceUploadJobStoredDocumentRedaction(),
		Findings:         append([]WorkspaceUploadJobFinding(nil), findings...),
	}
	source := workspaceSourceFromJob(job, "stored_local_pending_index")
	s.workspaceDocuments[documentID] = document
	s.workspaceUploadJobs[jobID] = job
	s.workspaceSources[sourceID] = source
	s.mu.Unlock()
	if traceID != "" {
		s.recordTrace(traceID, sessionID, deviceID, "workspace.document.stored_local", nowMS)
		s.recordTrace(traceID, sessionID, deviceID, "workspace.source.stored_local_pending_index", nowMS)
		s.recordTrace(traceID, sessionID, deviceID, "workspace.index_job.not_started_no_execute", nowMS)
	}
	return WorkspaceDocumentsResponse{
		SchemaVersion: WorkspaceDocumentsSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        "stored_local_pending_index",
		Documents:     []WorkspaceDocument{document},
		Jobs:          []WorkspaceUploadJob{job},
		Sources:       []WorkspaceSource{source},
		Redaction:     workspaceDocumentStoredRedaction(),
	}, nil
}

func (s *Server) storeWorkspaceDocumentFile(userID string, workspaceID string, sourceScope string, file multipart.File, maxBytes int64) (string, int64, error) {
	if maxBytes <= 0 {
		maxBytes = defaultWorkspaceDocumentMaxBytes
	}
	storeDir := strings.TrimSpace(s.workspaceDocumentStoreDir)
	if storeDir == "" {
		storeDir = workspaceDocumentStoreDir("")
	}
	targetDir := filepath.Join(storeDir, userID, workspaceID, sourceScope)
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp(targetDir, "a21-upload-*.tmp")
	if err != nil {
		return "", 0, err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(tmp, hasher), io.LimitReader(file, maxBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil {
		return "", 0, copyErr
	}
	if closeErr != nil {
		return "", 0, closeErr
	}
	if written == 0 {
		return "", 0, fmt.Errorf("workspace document file is empty")
	}
	if written > maxBytes {
		return "", 0, errWorkspaceDocumentTooLarge
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	finalPath := filepath.Join(targetDir, "sha256-"+hash+".bin")
	if err := os.Rename(tmpName, finalPath); err != nil {
		return "", 0, err
	}
	return hash, written, nil
}

func (s *Server) deleteWorkspaceDocumentForJobLocked(job *WorkspaceUploadJob, nowMS int64) {
	if job == nil || strings.TrimSpace(job.DocumentID) == "" {
		return
	}
	document, ok := s.workspaceDocuments[job.DocumentID]
	if ok {
		if path := s.workspaceDocumentPath(document); path != "" {
			_ = os.Remove(path)
		}
		document.Status = "deleted"
		document.StorageStatus = "deleted_local"
		document.IndexStatus = "deleted_no_execute"
		document.Readiness = "deleted_metadata_only"
		document.DocumentLabel = "deleted"
		document.ContentType = ""
		document.SizeBytes = 0
		document.DocumentHash = ""
		document.UpdatedAtMS = nowMS
		document.Redaction = WorkspaceDocumentRedaction{}
		document.Findings = append(document.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_document_deleted",
			Message: "A21 deleted the local intake file and retained only a redacted tombstone",
		})
		s.workspaceDocuments[document.DocumentID] = document
		for indexJobID, indexJob := range s.workspaceIndexJobs {
			if indexJob.DocumentID != document.DocumentID {
				continue
			}
			indexJob.Status = "deleted_metadata_only"
			indexJob.IndexStatus = "deleted_no_execute"
			indexJob.StorageStatus = "deleted_local"
			indexJob.DocumentHash = ""
			indexJob.DocumentLabel = "deleted"
			indexJob.ContentType = ""
			indexJob.SizeBytes = 0
			indexJob.UpdatedAtMS = nowMS
			indexJob.Redaction = workspaceUploadJobRedaction()
			indexJob.Findings = append(indexJob.Findings, WorkspaceUploadJobFinding{
				Code:    "workspace_index_job_deleted",
				Message: "A21 retained only a redacted index request tombstone after local document deletion",
			})
			s.workspaceIndexJobs[indexJobID] = indexJob
		}
	}
	job.DocumentHash = ""
	job.StorageStatus = "deleted_local"
	job.Redaction = workspaceUploadJobRedaction()
}

func (s *Server) workspaceDocumentPath(document WorkspaceDocument) string {
	hash := strings.TrimPrefix(strings.TrimSpace(document.DocumentHash), "sha256:")
	if !validWorkspaceDocumentSHA256(hash) {
		return ""
	}
	storeDir := strings.TrimSpace(s.workspaceDocumentStoreDir)
	if storeDir == "" {
		storeDir = workspaceDocumentStoreDir("")
	}
	return filepath.Join(storeDir, document.UserID, document.WorkspaceID, document.SourceScope, "sha256-"+hash+".bin")
}

func validWorkspaceDocumentSHA256(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, r := range hash {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return false
	}
	return true
}

func decodeWorkspaceUploadJobRequest(r *http.Request) (WorkspaceUploadJobRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return WorkspaceUploadJobRequest{}, fmt.Errorf("invalid json")
	}
	for key := range raw {
		if workspaceUploadJobForbiddenKey(key) {
			return WorkspaceUploadJobRequest{}, fmt.Errorf("workspace upload job request must not include raw document, URL, path, credential, or payload fields")
		}
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return WorkspaceUploadJobRequest{}, fmt.Errorf("invalid json")
	}
	var req WorkspaceUploadJobRequest
	if err := json.Unmarshal(encoded, &req); err != nil {
		return WorkspaceUploadJobRequest{}, fmt.Errorf("invalid json")
	}
	return req, nil
}

func workspaceUploadJobForbiddenKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, forbidden := range []string{
		"content",
		"document_text",
		"raw_text",
		"text_body",
		"content_base64",
		"data_base64",
		"bytes",
		"file_path",
		"local_path",
		"import_url",
		"url",
		"credential",
		"credentials",
		"api_key",
		"token",
		"provider_output",
		"retrieved_text",
		"evidence",
		"transcript",
		"audio",
		"wav_path",
	} {
		if key == forbidden {
			return true
		}
	}
	for _, forbidden := range []string{"base64", "credential", "api_key"} {
		if strings.Contains(key, forbidden) {
			return true
		}
	}
	return false
}

func decodeWorkspaceIndexJobRequest(r *http.Request) (WorkspaceIndexJobRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return WorkspaceIndexJobRequest{}, fmt.Errorf("invalid json")
	}
	for key := range raw {
		if workspaceUploadJobForbiddenKey(key) {
			return WorkspaceIndexJobRequest{}, fmt.Errorf("workspace index job request must not include raw document, URL, path, credential, provider output, or payload fields")
		}
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return WorkspaceIndexJobRequest{}, fmt.Errorf("invalid json")
	}
	var req WorkspaceIndexJobRequest
	if err := json.Unmarshal(encoded, &req); err != nil {
		return WorkspaceIndexJobRequest{}, fmt.Errorf("invalid json")
	}
	return req, nil
}

func (s *Server) createWorkspaceIndexJob(req WorkspaceIndexJobRequest) (WorkspaceIndexJobsResponse, error) {
	nowMS := s.now().UnixMilli()
	traceID := safeOptionalWorkspaceLabel(req.TraceID)
	sessionID := safeOptionalWorkspaceLabel(req.SessionID)
	deviceID := safeOptionalWorkspaceLabel(req.DeviceID)
	s.mu.Lock()
	document, job, _, err := s.resolveWorkspaceIndexTargetLocked(req)
	if err != nil {
		s.mu.Unlock()
		return WorkspaceIndexJobsResponse{}, err
	}
	if document.Status == "deleted" || job.Status == "deleted" || document.StorageStatus != "stored_local" || job.StorageStatus != "stored_local" {
		s.mu.Unlock()
		return WorkspaceIndexJobsResponse{}, fmt.Errorf("workspace document must be stored_local before indexing can be requested")
	}
	if path := s.workspaceDocumentPath(document); path == "" {
		s.mu.Unlock()
		return WorkspaceIndexJobsResponse{}, fmt.Errorf("workspace document local store metadata unavailable")
	} else if _, err := os.Stat(path); err != nil {
		s.mu.Unlock()
		return WorkspaceIndexJobsResponse{}, fmt.Errorf("workspace document local store file unavailable")
	}
	if traceID == "" {
		traceID = document.TraceID
	}
	if sessionID == "" {
		sessionID = document.SessionID
	}
	if deviceID == "" {
		deviceID = document.DeviceID
	}
	s.workspaceIndexJobSeq++
	indexJobID := fmt.Sprintf("a21-workspace-index-job-%06d", s.workspaceIndexJobSeq)
	finding := WorkspaceUploadJobFinding{
		Code:    "workspace_index_requested_no_execute",
		Message: "A21 recorded an indexing request for a stored-local document; parsing, embedding, upload, and V21 execution remain disabled",
	}
	document.Status = "indexing_requested_no_execute"
	document.IndexStatus = "indexing_requested_no_execute"
	document.Readiness = "indexing_requested_no_execute"
	document.UpdatedAtMS = nowMS
	document.TraceID = traceID
	document.SessionID = sessionID
	document.DeviceID = deviceID
	document.Findings = append(document.Findings, finding)
	job.Status = "indexing_requested_no_execute"
	job.IndexStatus = "indexing_requested_no_execute"
	job.IndexingAPIReady = true
	job.ExecutionStarted = false
	job.UpdatedAtMS = nowMS
	job.TraceID = traceID
	job.SessionID = sessionID
	job.DeviceID = deviceID
	job.Findings = append(job.Findings, finding)
	source := workspaceSourceFromJob(job, "indexing_requested_no_execute")
	indexJob := WorkspaceIndexJob{
		IndexJobID:             indexJobID,
		DocumentID:             document.DocumentID,
		SourceID:               document.SourceID,
		JobID:                  document.JobID,
		DocumentHash:           document.DocumentHash,
		StorageStatus:          document.StorageStatus,
		UserID:                 document.UserID,
		WorkspaceID:            document.WorkspaceID,
		SourceScope:            document.SourceScope,
		SourceKind:             document.SourceKind,
		DocumentLabel:          document.DocumentLabel,
		ContentType:            document.ContentType,
		SizeBytes:              document.SizeBytes,
		Status:                 "indexing_requested_no_execute",
		IndexStatus:            "indexing_requested_no_execute",
		AdapterContractVersion: ProfessionalAdapterContractVersion,
		CreatedAtMS:            nowMS,
		UpdatedAtMS:            nowMS,
		TraceID:                traceID,
		SessionID:              sessionID,
		DeviceID:               deviceID,
		V21ExecutionAllowed:    false,
		ExecutionStarted:       false,
		Redaction:              workspaceUploadJobStoredDocumentRedaction(),
		Findings:               []WorkspaceUploadJobFinding{finding},
	}
	s.workspaceDocuments[document.DocumentID] = document
	s.workspaceUploadJobs[job.JobID] = job
	s.workspaceSources[source.SourceID] = source
	s.workspaceIndexJobs[indexJobID] = indexJob
	s.mu.Unlock()
	if traceID != "" {
		s.recordTrace(traceID, sessionID, deviceID, "workspace.index_job.requested_no_execute", nowMS)
		s.recordTrace(traceID, sessionID, deviceID, "workspace.source.indexing_requested_no_execute", nowMS)
	}
	response, err := s.workspaceIndexJobsResponse(indexJobID, "", "", "", nil)
	if err != nil {
		return WorkspaceIndexJobsResponse{}, err
	}
	response.Status = "indexing_requested_no_execute"
	return response, nil
}

func (s *Server) resolveWorkspaceIndexTargetLocked(req WorkspaceIndexJobRequest) (WorkspaceDocument, WorkspaceUploadJob, WorkspaceSource, error) {
	documentID, err := normalizeWorkspaceLedgerID("document_id", req.DocumentID)
	if err != nil {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, err
	}
	jobID, err := normalizeWorkspaceLedgerID("job_id", req.JobID)
	if err != nil {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, err
	}
	sourceID, err := normalizeWorkspaceLedgerID("source_id", req.SourceID)
	if err != nil {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, err
	}
	if documentID == "" && jobID != "" {
		job, ok := s.workspaceUploadJobs[jobID]
		if !ok || strings.TrimSpace(job.DocumentID) == "" {
			return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace stored document not found")
		}
		documentID = job.DocumentID
	}
	if documentID == "" && sourceID != "" {
		source, ok := s.workspaceSources[sourceID]
		if !ok || strings.TrimSpace(source.DocumentID) == "" {
			return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace stored document not found")
		}
		documentID = source.DocumentID
	}
	if documentID == "" {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("document_id, job_id, or source_id is required")
	}
	document, ok := s.workspaceDocuments[documentID]
	if !ok {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace stored document not found")
	}
	if jobID != "" && document.JobID != jobID {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace stored document not found")
	}
	if sourceID != "" && document.SourceID != sourceID {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace stored document not found")
	}
	job, ok := s.workspaceUploadJobs[document.JobID]
	if !ok {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace upload job not found")
	}
	source, ok := s.workspaceSources[document.SourceID]
	if !ok {
		source = workspaceSourceFromJob(job, workspaceSourceReadinessForJob(job))
	}
	return document, job, source, nil
}

func (s *Server) workspaceIndexJobsResponse(indexJobID string, documentID string, jobID string, sourceID string, findings []WorkspaceUploadJobFinding) (WorkspaceIndexJobsResponse, error) {
	jobs, err := s.workspaceIndexJobsSnapshot(indexJobID, documentID, jobID, sourceID)
	if err != nil {
		return WorkspaceIndexJobsResponse{}, err
	}
	status := "ok"
	if (strings.TrimSpace(indexJobID) != "" || strings.TrimSpace(documentID) != "" || strings.TrimSpace(jobID) != "" || strings.TrimSpace(sourceID) != "") && len(jobs) == 0 {
		status = "not_found"
		findings = append(findings, WorkspaceUploadJobFinding{
			Code:    "workspace_index_job_not_found",
			Message: "A21 has no redacted workspace index job matching those filters",
		})
	} else if len(jobs) == 1 && strings.TrimSpace(indexJobID) != "" {
		status = jobs[0].Status
	}
	return WorkspaceIndexJobsResponse{
		SchemaVersion: WorkspaceIndexJobsSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        status,
		Jobs:          jobs,
		Redaction:     workspaceUploadJobRedactionForIndexJobs(jobs),
		Findings:      findings,
	}, nil
}

func (s *Server) workspaceIndexJobsSnapshot(indexJobID string, documentID string, jobID string, sourceID string) ([]WorkspaceIndexJob, error) {
	var err error
	if indexJobID, err = normalizeWorkspaceLedgerID("index_job_id", indexJobID); err != nil {
		return nil, err
	}
	if documentID, err = normalizeWorkspaceLedgerID("document_id", documentID); err != nil {
		return nil, err
	}
	if jobID, err = normalizeWorkspaceLedgerID("job_id", jobID); err != nil {
		return nil, err
	}
	if sourceID, err = normalizeWorkspaceLedgerID("source_id", sourceID); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.workspaceIndexJobs))
	if indexJobID != "" {
		keys = append(keys, indexJobID)
	} else {
		for key := range s.workspaceIndexJobs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}
	jobs := make([]WorkspaceIndexJob, 0, len(keys))
	for _, key := range keys {
		job, ok := s.workspaceIndexJobs[key]
		if !ok {
			continue
		}
		if documentID != "" && job.DocumentID != documentID {
			continue
		}
		if jobID != "" && job.JobID != jobID {
			continue
		}
		if sourceID != "" && job.SourceID != sourceID {
			continue
		}
		jobs = append(jobs, copyWorkspaceIndexJob(job))
	}
	return jobs, nil
}

func (s *Server) createWorkspaceUploadJob(req WorkspaceUploadJobRequest) (WorkspaceUploadJobsResponse, error) {
	userID, workspaceID, _, err := s.resolveProfessionalWorkspace(ProfessionalWorkspaceSelectionRequest{
		UserID:      req.UserID,
		WorkspaceID: req.WorkspaceID,
	})
	if err != nil {
		return WorkspaceUploadJobsResponse{}, err
	}
	sourceScope := defaultWorkspaceSourceScope(req.SourceScope)
	if !validWorkspaceSourceScope(sourceScope) {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("source_scope must be personal or public")
	}
	sourceKind := defaultWorkspaceSourceKind(req.SourceKind)
	if !validWorkspaceSourceKind(sourceKind) {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("source_kind must be upload or import")
	}
	documentLabel := safeWorkspaceDocumentLabel(req.DocumentLabel)
	if documentLabel == "" {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("valid redacted document_label is required")
	}
	contentType := safeWorkspaceContentType(req.ContentType)
	if strings.TrimSpace(req.ContentType) != "" && contentType == "" {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("valid redacted content_type is required")
	}
	if req.SizeBytes < 0 {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("size_bytes must be non-negative")
	}
	nowMS := s.now().UnixMilli()
	traceID := safeOptionalWorkspaceLabel(req.TraceID)
	sessionID := safeOptionalWorkspaceLabel(req.SessionID)
	deviceID := safeOptionalWorkspaceLabel(req.DeviceID)
	s.mu.Lock()
	s.workspaceUploadJobSeq++
	jobID := fmt.Sprintf("a21-workspace-job-%06d", s.workspaceUploadJobSeq)
	s.workspaceSourceSeq++
	sourceID := fmt.Sprintf("a21-workspace-source-%06d", s.workspaceSourceSeq)
	job := WorkspaceUploadJob{
		JobID:            jobID,
		SourceID:         sourceID,
		UserID:           userID,
		WorkspaceID:      workspaceID,
		SourceScope:      sourceScope,
		SourceKind:       sourceKind,
		DocumentLabel:    documentLabel,
		ContentType:      contentType,
		SizeBytes:        req.SizeBytes,
		Status:           "accepted_no_execute",
		IndexStatus:      "not_started_no_execute",
		Attempt:          1,
		CreatedAtMS:      nowMS,
		UpdatedAtMS:      nowMS,
		TraceID:          traceID,
		SessionID:        sessionID,
		DeviceID:         deviceID,
		UploadAPIReady:   true,
		ImportAPIReady:   true,
		IndexingAPIReady: false,
		ExecutionStarted: false,
		RetryAllowed:     true,
		DeleteAllowed:    true,
		Redaction:        workspaceUploadJobRedaction(),
		Findings: []WorkspaceUploadJobFinding{{
			Code:    "workspace_job_no_execute",
			Message: "A21 accepted only redacted workspace job metadata; upload bytes and indexing execution are not implemented in this slice",
		}},
	}
	s.workspaceUploadJobs[jobID] = job
	s.workspaceSources[sourceID] = workspaceSourceFromJob(job, "metadata_only")
	s.mu.Unlock()
	if traceID != "" {
		s.recordTrace(traceID, sessionID, deviceID, "workspace.upload_job.accepted_no_execute", nowMS)
		s.recordTrace(traceID, sessionID, deviceID, "workspace.index_job.not_started_no_execute", nowMS)
		s.recordTrace(traceID, sessionID, deviceID, "workspace.source.created_metadata_only", nowMS)
	}
	return s.workspaceUploadJobsResponse(jobID, nil), nil
}

func (s *Server) applyWorkspaceUploadJobAction(req WorkspaceUploadJobRequest) (WorkspaceUploadJobsResponse, error) {
	jobID := strings.TrimSpace(req.JobID)
	if jobID == "" {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("job_id is required")
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "retry"
	}
	s.mu.Lock()
	job, ok := s.workspaceUploadJobs[jobID]
	if !ok {
		s.mu.Unlock()
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("workspace upload job not found")
	}
	nowMS := s.now().UnixMilli()
	switch action {
	case "mark_failed", "fail":
		job.Status = "failed"
		job.IndexStatus = "failed_no_execute"
		job.UpdatedAtMS = nowMS
		job.RetryAllowed = true
		job.DeleteAllowed = true
		job.Findings = append(job.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_job_marked_failed",
			Message: "A21 marked the redacted workspace job failed without storing document text or bytes",
		})
	case "mark_searchable", "mark_indexed_metadata_only":
		job.Status = "accepted_no_execute"
		job.IndexStatus = "searchable_metadata_only"
		job.UpdatedAtMS = nowMS
		job.RetryAllowed = true
		job.DeleteAllowed = true
		job.Findings = append(job.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_source_searchable_metadata_only",
			Message: "A21 marked this redacted source searchable as metadata-only readiness; no indexing execution occurred",
		})
	case "retry":
		if job.DocumentID != "" && job.StorageStatus == "stored_local" {
			job.Status = "stored_local_pending_index"
		} else {
			job.Status = "accepted_no_execute"
		}
		job.IndexStatus = "not_started_no_execute"
		job.Attempt++
		job.UpdatedAtMS = nowMS
		job.RetryAllowed = true
		job.DeleteAllowed = true
		job.Findings = append(job.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_job_retry_no_execute",
			Message: "A21 accepted a retry request but did not execute upload or indexing",
		})
	case "delete":
		job.Status = "deleted"
		job.IndexStatus = "deleted_no_execute"
		job.DocumentLabel = "deleted"
		job.ContentType = ""
		job.SizeBytes = 0
		job.UpdatedAtMS = nowMS
		job.RetryAllowed = false
		job.DeleteAllowed = false
		if job.DocumentID != "" {
			s.deleteWorkspaceDocumentForJobLocked(&job, nowMS)
		}
		job.Findings = append(job.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_job_deleted",
			Message: "A21 retained only a redacted deletion tombstone",
		})
	default:
		s.mu.Unlock()
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("action must be retry, mark_failed, mark_searchable, or delete")
	}
	s.workspaceUploadJobs[jobID] = job
	if job.SourceID != "" {
		s.workspaceSources[job.SourceID] = workspaceSourceFromJob(job, workspaceSourceReadinessForJob(job))
	}
	s.mu.Unlock()
	if job.TraceID != "" {
		s.recordTrace(job.TraceID, job.SessionID, job.DeviceID, "workspace.upload_job."+job.Status, nowMS)
		if job.IndexStatus == "searchable_metadata_only" {
			s.recordTrace(job.TraceID, job.SessionID, job.DeviceID, "workspace.index_job.searchable_metadata_only", nowMS)
		}
	}
	return s.workspaceUploadJobsResponse(jobID, nil), nil
}

func (s *Server) workspaceUploadJobsResponse(jobID string, findings []WorkspaceUploadJobFinding) WorkspaceUploadJobsResponse {
	jobs := s.workspaceUploadJobsSnapshot(jobID)
	status := "ok"
	if strings.TrimSpace(jobID) != "" && len(jobs) == 0 {
		status = "not_found"
		findings = append(findings, WorkspaceUploadJobFinding{
			Code:    "workspace_job_not_found",
			Message: "A21 has no redacted workspace job with that id",
		})
	}
	return WorkspaceUploadJobsResponse{
		SchemaVersion: WorkspaceUploadJobsSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        status,
		Jobs:          jobs,
		Redaction:     workspaceUploadJobRedactionForJobs(jobs),
		Findings:      findings,
	}
}

func (s *Server) workspaceUploadJobsSnapshot(jobID string) []WorkspaceUploadJob {
	jobID = strings.TrimSpace(jobID)
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs := make([]WorkspaceUploadJob, 0, len(s.workspaceUploadJobs))
	if jobID != "" {
		if job, ok := s.workspaceUploadJobs[jobID]; ok {
			return []WorkspaceUploadJob{job}
		}
		return nil
	}
	keys := make([]string, 0, len(s.workspaceUploadJobs))
	for key := range s.workspaceUploadJobs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		jobs = append(jobs, s.workspaceUploadJobs[key])
	}
	return jobs
}

func (s *Server) workspaceSourcesResponse(sourceID string, userID string, workspaceID string, sourceScope string) (WorkspaceSourcesResponse, error) {
	sources, err := s.workspaceSourcesSnapshot(sourceID, userID, workspaceID, sourceScope)
	if err != nil {
		return WorkspaceSourcesResponse{}, err
	}
	status := "ok"
	findings := []WorkspaceUploadJobFinding(nil)
	if (strings.TrimSpace(sourceID) != "" || strings.TrimSpace(userID) != "" || strings.TrimSpace(workspaceID) != "" || strings.TrimSpace(sourceScope) != "") && len(sources) == 0 {
		status = "not_found"
		findings = append(findings, WorkspaceUploadJobFinding{
			Code:    "workspace_source_not_found",
			Message: "A21 has no redacted workspace source matching those filters",
		})
	}
	return WorkspaceSourcesResponse{
		SchemaVersion: WorkspaceSourcesSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        status,
		Sources:       sources,
		Summary:       workspaceSourceSummary(sources),
		Redaction:     workspaceUploadJobRedactionForSources(sources),
		Findings:      findings,
	}, nil
}

func (s *Server) workspaceSourcesSnapshot(sourceID string, userID string, workspaceID string, sourceScope string) ([]WorkspaceSource, error) {
	sourceID = strings.TrimSpace(sourceID)
	userID = strings.ToLower(strings.TrimSpace(userID))
	workspaceID = strings.ToLower(strings.TrimSpace(workspaceID))
	sourceScope = strings.ToLower(strings.TrimSpace(sourceScope))
	if sourceID != "" && safeOptionalWorkspaceLabel(sourceID) == "" {
		return nil, fmt.Errorf("valid redacted source_id is required")
	}
	if userID != "" && !validProfessionalLabel(userID) {
		return nil, fmt.Errorf("valid redacted user_id is required")
	}
	if workspaceID != "" && !validProfessionalLabel(workspaceID) {
		return nil, fmt.Errorf("valid redacted workspace_id is required")
	}
	if sourceScope != "" && !validWorkspaceSourceScope(sourceScope) {
		return nil, fmt.Errorf("source_scope must be personal or public")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sources := make([]WorkspaceSource, 0, len(s.workspaceSources))
	keys := make([]string, 0, len(s.workspaceSources))
	if sourceID != "" {
		keys = append(keys, sourceID)
	} else {
		for key := range s.workspaceSources {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		source, ok := s.workspaceSources[key]
		if !ok {
			continue
		}
		if userID != "" && source.UserID != userID {
			continue
		}
		if workspaceID != "" && source.WorkspaceID != workspaceID {
			continue
		}
		if sourceScope != "" && source.SourceScope != sourceScope {
			continue
		}
		sources = append(sources, copyWorkspaceSource(source))
	}
	return sources, nil
}

func (s *Server) workspaceSourceSummaryForWorkspace(userID string, workspaceID string) WorkspaceSourceSummary {
	sources, err := s.workspaceSourcesSnapshot("", userID, workspaceID, "")
	if err != nil {
		return emptyWorkspaceSourceSummary()
	}
	return workspaceSourceSummary(sources)
}

func decodeWorkspaceDeviceBindingRequest(r *http.Request) (WorkspaceDeviceBindingRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return WorkspaceDeviceBindingRequest{}, fmt.Errorf("invalid json")
	}
	for key := range raw {
		if workspaceDeviceBindingForbiddenKey(key) {
			return WorkspaceDeviceBindingRequest{}, fmt.Errorf("workspace device binding request must not include pairing secrets, URLs, paths, credentials, document text, provider output, or audio")
		}
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return WorkspaceDeviceBindingRequest{}, fmt.Errorf("invalid json")
	}
	var req WorkspaceDeviceBindingRequest
	if err := json.Unmarshal(encoded, &req); err != nil {
		return WorkspaceDeviceBindingRequest{}, fmt.Errorf("invalid json")
	}
	return req, nil
}

func workspaceDeviceBindingForbiddenKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if workspaceUploadJobForbiddenKey(key) {
		return true
	}
	for _, forbidden := range []string{
		"pairing_secret",
		"pairing_code",
		"device_secret",
		"device_credential",
		"wifi_password",
		"authorization",
		"cookie",
	} {
		if key == forbidden || strings.Contains(key, forbidden) {
			return true
		}
	}
	return false
}

func (s *Server) createWorkspaceDeviceBinding(req WorkspaceDeviceBindingRequest) (WorkspaceDeviceBindingsResponse, error) {
	userID, workspaceID, _, err := s.resolveProfessionalWorkspace(ProfessionalWorkspaceSelectionRequest{
		UserID:      req.UserID,
		WorkspaceID: req.WorkspaceID,
	})
	if err != nil {
		return WorkspaceDeviceBindingsResponse{}, err
	}
	deviceID := safeWorkspaceDeviceID(req.DeviceID)
	if deviceID == "" {
		return WorkspaceDeviceBindingsResponse{}, fmt.Errorf("valid A21 device_id is required")
	}
	deviceLabel := safeWorkspaceDeviceLabel(req.DeviceLabel, deviceID)
	if strings.TrimSpace(req.DeviceLabel) != "" && deviceLabel == "" {
		return WorkspaceDeviceBindingsResponse{}, fmt.Errorf("valid redacted device_label is required")
	}
	allowedScopes, err := normalizeWorkspaceBindingQueryScopes(req.AllowedQueryScopes)
	if err != nil {
		return WorkspaceDeviceBindingsResponse{}, err
	}
	traceID := safeOptionalWorkspaceLabel(req.TraceID)
	sessionID := safeOptionalWorkspaceLabel(req.SessionID)
	nowMS := s.now().UnixMilli()
	s.mu.Lock()
	for existingID, existing := range s.workspaceDeviceBindings {
		if existing.DeviceID == deviceID && existing.UserID == userID && existing.WorkspaceID == workspaceID && existing.Status != "deleted_metadata_only" {
			existing.DeviceLabel = deviceLabel
			existing.Status = "bound"
			existing.AccessScope = "professional_workspace"
			existing.AllowedQueryScopes = allowedScopes
			existing.ProfessionalAllowed = true
			existing.RevokedAtMS = 0
			existing.UpdatedAtMS = nowMS
			if traceID != "" {
				existing.TraceID = traceID
			}
			if sessionID != "" {
				existing.SessionID = sessionID
			}
			existing.Findings = append(existing.Findings, WorkspaceUploadJobFinding{
				Code:    "workspace_device_binding_updated",
				Message: "A21 refreshed an existing metadata-only device binding for this professional workspace",
			})
			s.workspaceDeviceBindings[existingID] = existing
			s.mu.Unlock()
			if traceID != "" {
				s.recordTrace(traceID, sessionID, deviceID, "workspace.device.bound", nowMS)
			}
			return s.workspaceDeviceBindingsResponse(existingID, "", "", "", "", nil)
		}
	}
	s.workspaceDeviceBindingSeq++
	bindingID := fmt.Sprintf("a21-workspace-device-binding-%06d", s.workspaceDeviceBindingSeq)
	binding := WorkspaceDeviceBinding{
		BindingID:           bindingID,
		DeviceID:            deviceID,
		DeviceLabel:         deviceLabel,
		UserID:              userID,
		WorkspaceID:         workspaceID,
		Status:              "bound",
		AccessScope:         "professional_workspace",
		AllowedQueryScopes:  allowedScopes,
		ProfessionalAllowed: true,
		PhysicalAccepted:    false,
		CreatedAtMS:         nowMS,
		UpdatedAtMS:         nowMS,
		TraceID:             traceID,
		SessionID:           sessionID,
		Redaction:           workspaceDeviceBindingRedaction(),
		Findings: []WorkspaceUploadJobFinding{{
			Code:    "workspace_device_bound",
			Message: "A21 bound this device to the selected professional workspace using metadata only; no pairing secret or provider credential is stored",
		}},
	}
	s.workspaceDeviceBindings[bindingID] = binding
	s.mu.Unlock()
	if traceID != "" {
		s.recordTrace(traceID, sessionID, deviceID, "workspace.device.bound", nowMS)
	}
	return s.workspaceDeviceBindingsResponse(bindingID, "", "", "", "", nil)
}

func (s *Server) applyWorkspaceDeviceBindingAction(req WorkspaceDeviceBindingRequest) (WorkspaceDeviceBindingsResponse, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "revoke"
	}
	nowMS := s.now().UnixMilli()
	s.mu.Lock()
	bindingID, err := s.resolveWorkspaceDeviceBindingIDLocked(req)
	if err != nil {
		s.mu.Unlock()
		return WorkspaceDeviceBindingsResponse{}, err
	}
	binding, ok := s.workspaceDeviceBindings[bindingID]
	if !ok {
		s.mu.Unlock()
		return WorkspaceDeviceBindingsResponse{}, fmt.Errorf("workspace device binding not found")
	}
	switch action {
	case "revoke":
		binding.Status = "revoked"
		binding.ProfessionalAllowed = false
		binding.RevokedAtMS = nowMS
		binding.UpdatedAtMS = nowMS
		binding.Findings = append(binding.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_device_binding_revoked",
			Message: "A21 revoked professional workspace access for this device without requiring firmware or NVS changes",
		})
	case "restore":
		binding.Status = "bound"
		binding.ProfessionalAllowed = true
		binding.RevokedAtMS = 0
		binding.UpdatedAtMS = nowMS
		binding.Findings = append(binding.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_device_binding_restored",
			Message: "A21 restored metadata-only professional workspace access for this device",
		})
	case "delete":
		binding.Status = "deleted_metadata_only"
		binding.ProfessionalAllowed = false
		binding.RevokedAtMS = nowMS
		binding.UpdatedAtMS = nowMS
		binding.DeviceLabel = "deleted"
		binding.Findings = append(binding.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_device_binding_deleted",
			Message: "A21 retained only a redacted binding tombstone",
		})
	default:
		s.mu.Unlock()
		return WorkspaceDeviceBindingsResponse{}, fmt.Errorf("action must be revoke, restore, or delete")
	}
	s.workspaceDeviceBindings[bindingID] = binding
	s.mu.Unlock()
	if binding.TraceID != "" {
		s.recordTrace(binding.TraceID, binding.SessionID, binding.DeviceID, "workspace.device."+binding.Status, nowMS)
	}
	return s.workspaceDeviceBindingsResponse(bindingID, "", "", "", "", nil)
}

func (s *Server) resolveWorkspaceDeviceBindingIDLocked(req WorkspaceDeviceBindingRequest) (string, error) {
	if id := strings.ToLower(strings.TrimSpace(req.BindingID)); id != "" {
		if safeOptionalWorkspaceLabel(id) == "" {
			return "", fmt.Errorf("valid redacted binding_id is required")
		}
		return id, nil
	}
	deviceID := safeWorkspaceDeviceID(req.DeviceID)
	if deviceID == "" {
		return "", fmt.Errorf("binding_id or valid A21 device_id is required")
	}
	userID, workspaceID, _, err := s.resolveProfessionalWorkspaceNoLock(req.UserID, req.WorkspaceID, "")
	if err != nil {
		return "", err
	}
	for bindingID, binding := range s.workspaceDeviceBindings {
		if binding.DeviceID == deviceID && binding.UserID == userID && binding.WorkspaceID == workspaceID && binding.Status != "deleted_metadata_only" {
			return bindingID, nil
		}
	}
	return "", fmt.Errorf("workspace device binding not found")
}

func (s *Server) workspaceDeviceBindingsResponse(bindingID string, deviceID string, userID string, workspaceID string, status string, findings []WorkspaceUploadJobFinding) (WorkspaceDeviceBindingsResponse, error) {
	bindings, err := s.workspaceDeviceBindingsSnapshot(bindingID, deviceID, userID, workspaceID, status)
	if err != nil {
		return WorkspaceDeviceBindingsResponse{}, err
	}
	responseStatus := "ok"
	if (strings.TrimSpace(bindingID) != "" || strings.TrimSpace(deviceID) != "" || strings.TrimSpace(userID) != "" || strings.TrimSpace(workspaceID) != "" || strings.TrimSpace(status) != "") && len(bindings) == 0 {
		responseStatus = "not_found"
		findings = append(findings, WorkspaceUploadJobFinding{
			Code:    "workspace_device_binding_not_found",
			Message: "A21 has no redacted workspace device binding matching those filters",
		})
	} else if len(bindings) == 1 && strings.TrimSpace(bindingID) != "" {
		responseStatus = bindings[0].Status
	}
	return WorkspaceDeviceBindingsResponse{
		SchemaVersion: WorkspaceDeviceBindingsSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        responseStatus,
		Bindings:      bindings,
		Summary:       workspaceDeviceBindingSummary(bindings),
		Redaction:     workspaceDeviceBindingRedaction(),
		Findings:      findings,
	}, nil
}

func (s *Server) workspaceDeviceBindingsSnapshot(bindingID string, deviceID string, userID string, workspaceID string, status string) ([]WorkspaceDeviceBinding, error) {
	bindingID = strings.ToLower(strings.TrimSpace(bindingID))
	rawDeviceID := strings.TrimSpace(deviceID)
	deviceID = safeWorkspaceDeviceID(deviceID)
	userID = strings.ToLower(strings.TrimSpace(userID))
	workspaceID = strings.ToLower(strings.TrimSpace(workspaceID))
	status = strings.ToLower(strings.TrimSpace(status))
	if bindingID != "" && safeOptionalWorkspaceLabel(bindingID) == "" {
		return nil, fmt.Errorf("valid redacted binding_id is required")
	}
	if rawDeviceID != "" && deviceID == "" {
		return nil, fmt.Errorf("valid A21 device_id is required")
	}
	if userID != "" && !validProfessionalLabel(userID) {
		return nil, fmt.Errorf("valid redacted user_id is required")
	}
	if workspaceID != "" && !validProfessionalLabel(workspaceID) {
		return nil, fmt.Errorf("valid redacted workspace_id is required")
	}
	if status != "" && !validWorkspaceDeviceBindingStatus(status) {
		return nil, fmt.Errorf("status must be bound, revoked, or deleted_metadata_only")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.workspaceDeviceBindings))
	if bindingID != "" {
		keys = append(keys, bindingID)
	} else {
		for key := range s.workspaceDeviceBindings {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	bindings := make([]WorkspaceDeviceBinding, 0, len(keys))
	for _, key := range keys {
		binding, ok := s.workspaceDeviceBindings[key]
		if !ok {
			continue
		}
		if deviceID != "" && binding.DeviceID != deviceID {
			continue
		}
		if userID != "" && binding.UserID != userID {
			continue
		}
		if workspaceID != "" && binding.WorkspaceID != workspaceID {
			continue
		}
		if status != "" && binding.Status != status {
			continue
		}
		bindings = append(bindings, copyWorkspaceDeviceBinding(binding))
	}
	return bindings, nil
}

func (s *Server) workspaceDeviceBindingSummaryForWorkspace(userID string, workspaceID string) WorkspaceDeviceBindingSummary {
	bindings, err := s.workspaceDeviceBindingsSnapshot("", "", userID, workspaceID, "")
	if err != nil {
		return emptyWorkspaceDeviceBindingSummary()
	}
	return workspaceDeviceBindingSummary(bindings)
}

func workspaceDeviceBindingSummary(bindings []WorkspaceDeviceBinding) WorkspaceDeviceBindingSummary {
	summary := emptyWorkspaceDeviceBindingSummary()
	summary.TotalBindings = 0
	for _, binding := range bindings {
		summary.TotalBindings++
		switch binding.Status {
		case "bound":
			summary.ActiveBindings++
			if binding.ProfessionalAllowed {
				summary.ProfessionalAllowedDevices++
			}
		case "revoked":
			summary.RevokedBindings++
		case "deleted_metadata_only":
			summary.DeletedBindings++
		}
	}
	switch {
	case summary.ActiveBindings > 0:
		summary.DeviceBindingPolicy = "bound_devices_only"
		summary.WorkspaceAccessStatus = "bound_device_ready"
	case summary.TotalBindings > 0:
		summary.DeviceBindingPolicy = "bound_devices_only"
		summary.WorkspaceAccessStatus = "no_active_device_binding"
	default:
		summary.DeviceBindingPolicy = "open_until_binding_configured"
		summary.WorkspaceAccessStatus = "binding_not_configured"
	}
	return summary
}

func emptyWorkspaceDeviceBindingSummary() WorkspaceDeviceBindingSummary {
	return WorkspaceDeviceBindingSummary{
		DeviceBindingPolicy:   "open_until_binding_configured",
		WorkspaceAccessStatus: "binding_not_configured",
	}
}

func copyWorkspaceDeviceBinding(binding WorkspaceDeviceBinding) WorkspaceDeviceBinding {
	binding.AllowedQueryScopes = append([]string(nil), binding.AllowedQueryScopes...)
	binding.Findings = append([]WorkspaceUploadJobFinding(nil), binding.Findings...)
	return binding
}

func workspaceDeviceBindingRedaction() WorkspaceDeviceBindingRedaction {
	return WorkspaceDeviceBindingRedaction{
		PairingSecretStored:    false,
		DeviceCredentialStored: false,
		DocumentTextStored:     false,
		QueryTextStored:        false,
		RetrievedTextStored:    false,
		FullURLStored:          false,
		LocalPathStored:        false,
		CredentialValueStored:  false,
		ProviderOutputStored:   false,
		VoiceTranscriptStored:  false,
	}
}

func (s *Server) professionalDeviceBindingDecision(deviceID string, userID string, workspaceID string, queryScope string) professionalDeviceBindingDecision {
	deviceID = safeWorkspaceDeviceID(deviceID)
	userID = defaultProfessionalLabel(userID, v21adapter.DefaultUserID)
	workspaceID = defaultProfessionalLabel(workspaceID, v21adapter.DefaultWorkspaceID)
	queryScope = defaultProfessionalQueryScope(queryScope)
	s.mu.Lock()
	defer s.mu.Unlock()
	candidates := make([]WorkspaceDeviceBinding, 0)
	for _, binding := range s.workspaceDeviceBindings {
		if binding.UserID == userID && binding.WorkspaceID == workspaceID {
			candidates = append(candidates, binding)
		}
	}
	if len(candidates) == 0 {
		return professionalDeviceBindingDecision{
			Allowed:     true,
			Policy:      "open_until_binding_configured",
			Status:      "not_configured",
			TraceMarker: "professional.device_binding.open_until_configured",
		}
	}
	if deviceID == "" {
		return professionalDeviceBindingDecision{
			Allowed:     false,
			Policy:      "bound_devices_only",
			Status:      "device_id_invalid",
			FailureCode: "device_unbound",
			TraceMarker: "professional.device_binding.blocked.device_unbound",
		}
	}
	for _, binding := range candidates {
		if binding.DeviceID != deviceID {
			continue
		}
		if binding.Status == "deleted_metadata_only" {
			return professionalDeviceBindingDecision{
				Allowed:     false,
				Policy:      "bound_devices_only",
				BindingID:   binding.BindingID,
				Status:      "deleted_metadata_only",
				FailureCode: "device_unbound",
				TraceMarker: "professional.device_binding.blocked.device_unbound",
			}
		}
		if binding.Status == "revoked" || !binding.ProfessionalAllowed {
			return professionalDeviceBindingDecision{
				Allowed:     false,
				Policy:      "bound_devices_only",
				BindingID:   binding.BindingID,
				Status:      "revoked",
				FailureCode: "device_binding_revoked",
				TraceMarker: "professional.device_binding.blocked.revoked",
			}
		}
		if !workspaceBindingAllowsQueryScope(binding, queryScope) {
			return professionalDeviceBindingDecision{
				Allowed:     false,
				Policy:      "bound_devices_only",
				BindingID:   binding.BindingID,
				Status:      "scope_denied",
				FailureCode: "device_scope_denied",
				TraceMarker: "professional.device_binding.blocked.scope_denied",
			}
		}
		return professionalDeviceBindingDecision{
			Allowed:     true,
			Policy:      "bound_devices_only",
			BindingID:   binding.BindingID,
			Status:      "bound",
			TraceMarker: "professional.device_binding.bound",
		}
	}
	return professionalDeviceBindingDecision{
		Allowed:     false,
		Policy:      "bound_devices_only",
		Status:      "unbound",
		FailureCode: "device_unbound",
		TraceMarker: "professional.device_binding.blocked.device_unbound",
	}
}

func workspaceSourceFromJob(job WorkspaceUploadJob, readiness string) WorkspaceSource {
	readiness = defaultWorkspaceSourceReadiness(readiness)
	storedLocal := job.StorageStatus == "stored_local" && readiness != "deleted_metadata_only"
	redaction := job.Redaction
	return WorkspaceSource{
		SourceID:          job.SourceID,
		JobID:             job.JobID,
		DocumentID:        job.DocumentID,
		DocumentHash:      job.DocumentHash,
		StorageStatus:     job.StorageStatus,
		UserID:            job.UserID,
		WorkspaceID:       job.WorkspaceID,
		SourceScope:       job.SourceScope,
		SourceKind:        job.SourceKind,
		DocumentLabel:     job.DocumentLabel,
		ContentType:       job.ContentType,
		SizeBytes:         job.SizeBytes,
		Readiness:         readiness,
		IndexStatus:       job.IndexStatus,
		CreatedAtMS:       job.CreatedAtMS,
		UpdatedAtMS:       job.UpdatedAtMS,
		TraceID:           job.TraceID,
		SessionID:         job.SessionID,
		DeviceID:          job.DeviceID,
		MetadataOnly:      !storedLocal,
		StoredLocal:       storedLocal,
		IndexingRequested: readiness == "indexing_requested_no_execute",
		Searchable:        readiness == "searchable_metadata_only",
		Deleted:           readiness == "deleted_metadata_only",
		Redaction:         redaction,
		Findings:          append([]WorkspaceUploadJobFinding(nil), job.Findings...),
	}
}

func workspaceSourceReadinessForJob(job WorkspaceUploadJob) string {
	switch {
	case job.Status == "deleted":
		return "deleted_metadata_only"
	case job.Status == "failed":
		return "failed_metadata_only"
	case job.Status == "indexing_requested_no_execute" || job.IndexStatus == "indexing_requested_no_execute":
		return "indexing_requested_no_execute"
	case job.Status == "stored_local_pending_index":
		return "stored_local_pending_index"
	case job.IndexStatus == "searchable_metadata_only":
		return "searchable_metadata_only"
	default:
		return "metadata_only"
	}
}

func defaultWorkspaceSourceReadiness(readiness string) string {
	switch strings.TrimSpace(readiness) {
	case "metadata_only", "stored_local_pending_index", "indexing_requested_no_execute", "searchable_metadata_only", "failed_metadata_only", "deleted_metadata_only":
		return strings.TrimSpace(readiness)
	default:
		return "metadata_only"
	}
}

func copyWorkspaceSource(source WorkspaceSource) WorkspaceSource {
	source.Findings = append([]WorkspaceUploadJobFinding(nil), source.Findings...)
	return source
}

func workspaceSourceSummary(sources []WorkspaceSource) WorkspaceSourceSummary {
	summary := emptyWorkspaceSourceSummary()
	for _, source := range sources {
		summary.TotalSources++
		if source.MetadataOnly {
			summary.MetadataOnlySources++
		}
		if source.StoredLocal {
			summary.StoredLocalSources++
		}
		if source.Deleted {
			summary.DeletedSources++
			continue
		}
		if validWorkspaceSourceScope(source.SourceScope) {
			summary.SourceScopeCounts[source.SourceScope]++
		}
		if source.Searchable {
			summary.SearchableSources++
			if validWorkspaceSourceScope(source.SourceScope) {
				summary.SearchableSourceScopeCounts[source.SourceScope]++
			}
		}
		if source.StoredLocal && validWorkspaceSourceScope(source.SourceScope) {
			summary.StoredLocalSourceScopeCounts[source.SourceScope]++
		}
		if source.IndexingRequested {
			summary.IndexingRequestedSources++
			if validWorkspaceSourceScope(source.SourceScope) {
				summary.IndexingRequestedSourceScopeCounts[source.SourceScope]++
			}
		}
	}
	switch {
	case summary.SearchableSources > 0:
		summary.WorkspaceStatus = "searchable_metadata_only"
	case summary.IndexingRequestedSources > 0:
		summary.WorkspaceStatus = "indexing_requested_no_execute"
	case summary.TotalSources == summary.DeletedSources && summary.TotalSources > 0:
		summary.WorkspaceStatus = "deleted_metadata_only"
	case summary.StoredLocalSources > 0:
		summary.WorkspaceStatus = "stored_local_pending_index"
	case summary.TotalSources > 0:
		summary.WorkspaceStatus = "metadata_only"
	default:
		summary.WorkspaceStatus = "no_sources_metadata_only"
	}
	return summary
}

func emptyWorkspaceSourceSummary() WorkspaceSourceSummary {
	return WorkspaceSourceSummary{
		SourceScopeCounts:                  map[string]int{"public": 0, "personal": 0},
		StoredLocalSourceScopeCounts:       map[string]int{"public": 0, "personal": 0},
		IndexingRequestedSourceScopeCounts: map[string]int{"public": 0, "personal": 0},
		SearchableSourceScopeCounts:        map[string]int{"public": 0, "personal": 0},
		WorkspaceStatus:                    "no_sources_metadata_only",
	}
}

func workspaceQueryScopeReadiness(queryScope string, summary WorkspaceSourceSummary) string {
	switch defaultProfessionalQueryScope(queryScope) {
	case v21adapter.QueryScopePublic:
		return workspaceSingleScopeReadiness("public", summary)
	case v21adapter.QueryScopePersonal:
		return workspaceSingleScopeReadiness("personal", summary)
	case v21adapter.QueryScopeCombined:
		publicReady := summary.SearchableSourceScopeCounts["public"] > 0
		personalReady := summary.SearchableSourceScopeCounts["personal"] > 0
		publicIndexing := summary.IndexingRequestedSourceScopeCounts["public"] > 0
		personalIndexing := summary.IndexingRequestedSourceScopeCounts["personal"] > 0
		publicStored := summary.StoredLocalSourceScopeCounts["public"] > 0
		personalStored := summary.StoredLocalSourceScopeCounts["personal"] > 0
		switch {
		case publicReady && personalReady:
			return "combined_searchable_metadata_only"
		case publicReady || personalReady:
			return "partial_searchable_metadata_only"
		case publicIndexing || personalIndexing:
			return "indexing_requested_no_execute"
		case publicStored || personalStored:
			return "stored_local_pending_index"
		case summary.SourceScopeCounts["public"] > 0 || summary.SourceScopeCounts["personal"] > 0:
			return "metadata_only"
		default:
			return "no_sources_metadata_only"
		}
	default:
		return "no_sources_metadata_only"
	}
}

func workspaceSingleScopeReadiness(scope string, summary WorkspaceSourceSummary) string {
	switch {
	case summary.SearchableSourceScopeCounts[scope] > 0:
		return "searchable_metadata_only"
	case summary.IndexingRequestedSourceScopeCounts[scope] > 0:
		return "indexing_requested_no_execute"
	case summary.StoredLocalSourceScopeCounts[scope] > 0:
		return "stored_local_pending_index"
	case summary.SourceScopeCounts[scope] > 0:
		return "metadata_only"
	default:
		return "no_sources_metadata_only"
	}
}

func copyWorkspaceSourceCounts(counts map[string]int) map[string]int {
	if len(counts) == 0 {
		return nil
	}
	out := make(map[string]int, len(counts))
	for scope, count := range counts {
		out[scope] = count
	}
	return out
}

func copyWorkspaceIndexJob(job WorkspaceIndexJob) WorkspaceIndexJob {
	job.Findings = append([]WorkspaceUploadJobFinding(nil), job.Findings...)
	return job
}

func normalizeWorkspaceLedgerID(field string, value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	if len(value) > 96 {
		return "", fmt.Errorf("valid redacted %s is required", field)
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return "", fmt.Errorf("valid redacted %s is required", field)
	}
	return value, nil
}

func defaultWorkspaceSourceScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return "personal"
	}
	return scope
}

func validWorkspaceSourceScope(scope string) bool {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "personal", "public":
		return true
	default:
		return false
	}
}

func defaultWorkspaceSourceKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		return "upload"
	}
	return kind
}

func validWorkspaceSourceKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "upload", "import":
		return true
	default:
		return false
	}
}

func safeWorkspaceDocumentLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" || len([]rune(label)) > 96 {
		return ""
	}
	lower := strings.ToLower(label)
	for _, forbidden := range []string{"http://", "https://", "/", "\\", "api_key", "secret", "token", "bearer "} {
		if strings.Contains(lower, forbidden) {
			return ""
		}
	}
	for _, r := range label {
		if r < 32 || r == 127 {
			return ""
		}
	}
	return label
}

func safeWorkspaceDeviceID(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || len(deviceID) > 96 || !validA21DeviceID(deviceID) {
		return ""
	}
	lower := strings.ToLower(deviceID)
	for _, forbidden := range []string{"http://", "https://", "/", "\\", "secret", "token", "credential", "api_key", "bearer ", "sk-"} {
		if strings.Contains(lower, forbidden) {
			return ""
		}
	}
	for _, r := range deviceID {
		if r < 33 || r == 127 {
			return ""
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == ':' || r == '.' {
			continue
		}
		return ""
	}
	return deviceID
}

func safeWorkspaceDeviceLabel(label string, fallback string) string {
	if strings.TrimSpace(label) == "" {
		return safeWorkspaceDeviceID(fallback)
	}
	return safeWorkspaceDocumentLabel(label)
}

func normalizeWorkspaceBindingQueryScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return []string{v21adapter.QueryScopePublic, v21adapter.QueryScopePersonal, v21adapter.QueryScopeCombined}, nil
	}
	seen := make(map[string]bool, len(scopes))
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.ToLower(strings.TrimSpace(scope))
		if !v21adapter.ValidQueryScope(scope) {
			return nil, fmt.Errorf("allowed_query_scopes must contain only public_only, personal_only, or personal_plus_public")
		}
		if seen[scope] {
			continue
		}
		seen[scope] = true
		out = append(out, scope)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("allowed_query_scopes must contain at least one query scope")
	}
	return out, nil
}

func workspaceBindingAllowsQueryScope(binding WorkspaceDeviceBinding, queryScope string) bool {
	queryScope = defaultProfessionalQueryScope(queryScope)
	for _, scope := range binding.AllowedQueryScopes {
		if scope == queryScope {
			return true
		}
	}
	return false
}

func validWorkspaceDeviceBindingStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "bound", "revoked", "deleted_metadata_only":
		return true
	default:
		return false
	}
}

func safeWorkspaceContentType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if contentType == "" {
		return ""
	}
	if len(contentType) > 80 {
		return ""
	}
	for _, r := range contentType {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '/' || r == '-' || r == '_' || r == '+' || r == '.' {
			continue
		}
		return ""
	}
	return contentType
}

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
		{ID: "a21_voice_default_dashscope", Label: "A21 natural voice", Status: "default", ProviderProfile: "a21_bailian_qwen_tts_realtime", TTSProfile: "dashscope_qwen_tts_realtime", Default: selected == "a21_voice_default_dashscope"},
		{ID: "a21_voice_clone_default", Label: "A21 cloned voice", Status: "available", ProviderProfile: "voice_clone_cli", TTSProfile: "voice_clone_cli", VoiceClone: true, Default: selected == "a21_voice_clone_default"},
		{ID: "a21_voice_clone_cosyvoice", Label: "CosyVoice clone", Status: "planned", ProviderProfile: "a21_bailian_cosyvoice_clone_tts", TTSProfile: "voice_clone_cli", VoiceClone: true, Default: selected == "a21_voice_clone_cosyvoice"},
		{ID: "a21_voice_clone_minimax", Label: "MiniMax clone", Status: "planned", ProviderProfile: "a21_minimax_voice_clone_tts", TTSProfile: "voice_clone_cli", VoiceClone: true, Default: selected == "a21_voice_clone_minimax"},
	}
}

func voiceChainTTSLabel(profile string) string {
	switch profile {
	case "dashscope_qwen_tts_realtime":
		return "A21 natural voice"
	case "doubao_tts_realtime":
		return "Doubao realtime voice"
	case "voice_clone_cli":
		return "A21 cloned voice"
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

func sanitizedXiaozhiOTAHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" ||
		strings.ContainsAny(host, "/\\?#% \t\r\n") ||
		strings.Contains(host, "@") ||
		strings.Contains(strings.ToLower(host), "token") ||
		strings.Contains(strings.ToLower(host), "secret") {
		return ""
	}
	return host
}

func (s *Server) handleDeviceControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req DeviceControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	if req.State == "" {
		req.State = protocol.ExpressionListening
	}
	if req.StreamID == "" && req.State == protocol.ExpressionSpeaking {
		req.StreamID = "a21-device-command-stream-000001"
	}
	if req.MockAudioChunks != nil && (*req.MockAudioChunks < 0 || *req.MockAudioChunks > 8) {
		http.Error(w, "mock_audio_chunks must be between 0 and 8", http.StatusBadRequest)
		return
	}
	if len(req.AudioChunks) > 8 {
		http.Error(w, "audio_chunks must contain at most 8 chunks", http.StatusBadRequest)
		return
	}
	if (req.DiagnosticToneHz != 0 || req.DiagnosticToneDurationMS != 0 || req.DiagnosticToneVolume != 0) &&
		(req.DiagnosticToneHz < 50 || req.DiagnosticToneHz > 8000 ||
			req.DiagnosticToneDurationMS < 1 || req.DiagnosticToneDurationMS > 10000 ||
			req.DiagnosticToneVolume < 1 || req.DiagnosticToneVolume > 255) {
		http.Error(w, "diagnostic tone must set hz 50-8000, duration 1-10000ms, and volume 1-255", http.StatusBadRequest)
		return
	}
	for i := range req.AudioChunks {
		if req.AudioChunks[i].StreamID == "" {
			req.AudioChunks[i].StreamID = req.StreamID
		}
		if req.AudioChunks[i].StreamID != req.StreamID {
			http.Error(w, "audio_chunks stream_id must match request stream_id", http.StatusBadRequest)
			return
		}
		if err := protocol.ValidateAudioPlaybackChunk(req.AudioChunks[i]); err != nil {
			http.Error(w, "invalid audio_chunks payload", http.StatusBadRequest)
			return
		}
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	socket, ok := s.audioSocket(req.DeviceID)
	if !ok {
		http.Error(w, "device audio websocket is not connected", http.StatusConflict)
		return
	}

	events := s.deviceControlEvents(req)
	s.applyDeviceControlStreamState(req)
	for _, event := range events {
		if err := r.Context().Err(); err != nil {
			s.clearDeviceControlStreamState(req)
			http.Error(w, "device command delivery failed", http.StatusBadGateway)
			return
		}
		if err := writeAudioEnvelope(r.Context(), socket.conn, socket.writeMu, event); err != nil {
			s.clearDeviceControlStreamState(req)
			http.Error(w, "device command delivery failed", http.StatusBadGateway)
			return
		}
	}
	s.recordTrace(traceID, sessionID, req.DeviceID, "device.command.delivered", s.now().UnixMilli())
	writeJSON(w, http.StatusOK, DeviceControlResponse{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "audio_ws",
		Events:             events,
	})
}

func (s *Server) handleXiaozhiDeviceControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiDeviceControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	event, err := xiaozhiDeviceControlEventFromRequest(req)
	if err != nil {
		http.Error(w, "invalid xiaozhi device event", http.StatusBadRequest)
		return
	}
	switch event.Kind {
	case xiaozhitransport.DeviceEventKindState, xiaozhitransport.DeviceEventKindFace, xiaozhitransport.DeviceEventKindDisplay, xiaozhitransport.DeviceEventKindMotion:
	default:
		http.Error(w, "unsupported xiaozhi device control event", http.StatusBadRequest)
		return
	}
	socket, ok := s.xiaozhiSocket(req.DeviceID)
	if !ok {
		http.Error(w, "xiaozhi websocket is not connected", http.StatusConflict)
		return
	}
	if !socket.features.DeviceEvents {
		http.Error(w, "xiaozhi device events require debug profile negotiation", http.StatusConflict)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	payload, err := xiaozhitransport.BuildDeviceExtensionEvent(xiaozhitransport.Identity{
		DeviceID:  req.DeviceID,
		TraceID:   traceID,
		SessionID: sessionID,
	}, socket.features, xiaozhitransport.DeviceExtensionProfileDebug, event)
	if err != nil {
		http.Error(w, "invalid xiaozhi device event", http.StatusBadRequest)
		return
	}
	socket.writeMu.Lock()
	err = socket.conn.Write(r.Context(), websocket.MessageText, payload)
	socket.writeMu.Unlock()
	if err != nil {
		http.Error(w, "xiaozhi device command delivery failed", http.StatusBadGateway)
		return
	}
	s.recordXiaozhiDeviceControlDelivered(req.DeviceID, traceID, sessionID, event)
	writeJSON(w, http.StatusOK, XiaozhiDeviceControlResponse{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "xiaozhi_ws",
		Event:              string(event.Kind),
		Value:              event.Value,
	})
}

const (
	xiaozhiSpeakerVolumeToolName          = "self.audio_speaker.set_volume"
	xiaozhiMCPGetDeviceStatusToolName     = "self.get_device_status"
	xiaozhiMCPScreenSetBrightnessToolName = "self.screen.set_brightness"
	xiaozhiMCPScreenSetThemeToolName      = "self.screen.set_theme"
	xiaozhiMCPScreenGetInfoToolName       = "self.screen.get_info"
	xiaozhiMCPRobotGetHeadAnglesToolName  = "self.robot.get_head_angles"
	xiaozhiMCPRobotSetHeadAnglesToolName  = "self.robot.set_head_angles"
	xiaozhiMCPRobotSetLEDColorToolName    = "self.robot.set_led_color"
)

func (s *Server) handleXiaozhiSay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiSayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(req.Text)
	wavPath := strings.TrimSpace(req.WAVPath)
	if text == "" && wavPath == "" {
		http.Error(w, "text or wav_path is required", http.StatusBadRequest)
		return
	}
	if text != "" && wavPath != "" {
		http.Error(w, "provide either text or wav_path, not both", http.StatusBadRequest)
		return
	}
	var wavChunks []providers.VoiceAudioChunk
	audioSource := ""
	audioBasename := ""
	if wavPath != "" {
		chunks, err := audio.ReadPCM16MonoWAVChunksForSampleRate(wavPath, 60, 16000)
		if err != nil || len(chunks) == 0 {
			http.Error(w, "wav_path must reference an A21-compatible 16 kHz mono PCM WAV", http.StatusBadRequest)
			return
		}
		audioSource = "wav_file"
		audioBasename = filepath.Base(wavPath)
		wavChunks = make([]providers.VoiceAudioChunk, 0, len(chunks))
		for _, chunk := range chunks {
			wavChunks = append(wavChunks, providers.VoiceAudioChunk{
				Codec:        string(protocol.AudioCodecPCMS16LE),
				SampleRateHz: chunk.SampleRateHz,
				Channels:     chunk.Channels,
				DurationMS:   chunk.DurationMS,
				DataBase64:   chunk.DataBase64,
			})
		}
	}
	socket, ok := s.xiaozhiSocket(req.DeviceID)
	if !ok || socket.session == nil {
		http.Error(w, "xiaozhi websocket is not connected", http.StatusConflict)
		return
	}
	mode := req.Mode
	if mode == "" {
		mode = protocol.ModeWorkmate
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	session := socket.session
	session.deviceID = req.DeviceID
	session.traceID = traceID
	session.sessionID = sessionID
	turn := session.startXiaozhiTurn(r.Context(), mode)
	session.resetXiaozhiTTSStop()
	task := xiaozhiTurnTask{
		turn:      turn,
		turnID:    xiaozhiTurnID(turn),
		traceID:   traceID,
		sessionID: sessionID,
		deviceID:  req.DeviceID,
		mode:      mode,
	}
	s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.say.start", s.now().UnixMilli())
	if err := session.writeXiaozhiJSON(r.Context(), socket.conn, turn, map[string]any{
		"type":       "tts",
		"state":      "start",
		"phase":      "host_say",
		"turn_id":    task.turnID,
		"trace_id":   traceID,
		"session_id": sessionID,
		"device_id":  req.DeviceID,
	}); err != nil {
		http.Error(w, "xiaozhi say start delivery failed", http.StatusBadGateway)
		return
	}
	if err := session.writeXiaozhiJSON(r.Context(), socket.conn, turn, map[string]any{
		"type":       "tts",
		"state":      "sentence_start",
		"phase":      "host_say",
		"turn_id":    task.turnID,
		"trace_id":   traceID,
		"session_id": sessionID,
		"device_id":  req.DeviceID,
		"text":       "",
	}); err != nil {
		http.Error(w, "xiaozhi say sentence delivery failed", http.StatusBadGateway)
		return
	}
	var audioChunks int
	if wavPath != "" {
		audioChunks, ok = s.writeXiaozhiWAVAudioDownlink(r.Context(), socket.conn, session, task, wavChunks, "xiaozhi.say")
	} else {
		audioChunks, ok = s.writeXiaozhiTextAudioDownlink(r.Context(), socket.conn, session, task, text, mode, "xiaozhi.say")
	}
	if !ok {
		if interruptReason, interrupted := xiaozhiUserInterruptReason(session.xiaozhiTurnCancelReason(turn)); interrupted {
			s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.say.interrupted", s.now().UnixMilli())
			writeJSON(w, http.StatusOK, XiaozhiSayResponse{
				TraceID:            traceID,
				SessionID:          sessionID,
				DeviceID:           req.DeviceID,
				Status:             "interrupted",
				DeliveredTransport: "xiaozhi_ws",
				TextChars:          len([]rune(text)),
				AudioChunks:        audioChunks,
				InterruptReason:    interruptReason,
				AudioSource:        audioSource,
				AudioBasename:      audioBasename,
			})
			return
		}
		s.writeXiaozhiTTSStop(r.Context(), socket.conn, session, turn, task, "host_say_unavailable")
		http.Error(w, "xiaozhi say audio delivery failed", http.StatusBadGateway)
		return
	}
	s.writeXiaozhiTTSStop(r.Context(), socket.conn, session, turn, task, "host_say_complete")
	s.suppressXiaozhiInputAfterHostSay(session, task)
	session.completeXiaozhiTurn(turn)
	s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.say.delivered", s.now().UnixMilli())
	writeJSON(w, http.StatusOK, XiaozhiSayResponse{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "xiaozhi_ws",
		TextChars:          len([]rune(text)),
		AudioChunks:        audioChunks,
		AudioSource:        audioSource,
		AudioBasename:      audioBasename,
	})
}

func (s *Server) handleXiaozhiSpeakerVolume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiSpeakerVolumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	if req.Volume < 0 || req.Volume > 100 {
		http.Error(w, "volume must be 0..100", http.StatusBadRequest)
		return
	}
	socket, ok := s.xiaozhiSocket(req.DeviceID)
	if !ok {
		http.Error(w, "xiaozhi websocket is not connected", http.StatusConflict)
		return
	}
	if !socket.features.MCP {
		http.Error(w, "xiaozhi speaker volume requires stock MCP support", http.StatusConflict)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	mcpID := "a21-mcp-speaker-volume-" + safeGatewayFallbackToken(traceID, "trace")
	payload, err := xiaozhitransport.BuildMCPToolsCallRequest(mcpID, xiaozhiSpeakerVolumeToolName, map[string]any{
		"volume": req.Volume,
	})
	if err != nil {
		http.Error(w, "invalid xiaozhi speaker volume mcp request", http.StatusBadRequest)
		return
	}
	var payloadObject map[string]any
	if err := json.Unmarshal(payload, &payloadObject); err != nil {
		http.Error(w, "invalid xiaozhi speaker volume mcp request", http.StatusBadRequest)
		return
	}
	message, err := json.Marshal(map[string]any{
		"type":       "mcp",
		"payload":    payloadObject,
		"trace_id":   traceID,
		"session_id": sessionID,
		"device_id":  req.DeviceID,
	})
	if err != nil {
		http.Error(w, "invalid xiaozhi speaker volume mcp message", http.StatusBadRequest)
		return
	}
	socket.writeMu.Lock()
	err = socket.conn.Write(r.Context(), websocket.MessageText, message)
	socket.writeMu.Unlock()
	if err != nil {
		http.Error(w, "xiaozhi speaker volume delivery failed", http.StatusBadGateway)
		return
	}
	s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.mcp.speaker_volume.sent", s.now().UnixMilli())
	s.recordXiaozhiDeviceActivity(&xiaozhiSession{deviceID: req.DeviceID, traceID: traceID, sessionID: sessionID}, "xiaozhi.mcp.speaker_volume.sent", map[string]string{
		"speaker_volume": strconv.Itoa(req.Volume),
	})
	writeJSON(w, http.StatusOK, XiaozhiSpeakerVolumeResponse{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "xiaozhi_mcp",
		ToolName:           xiaozhiSpeakerVolumeToolName,
		Volume:             req.Volume,
		MCPID:              mcpID,
	})
}

func (s *Server) handleXiaozhiDeviceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiDeviceStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	response, status, message := s.deliverXiaozhiMCPControl(r.Context(), XiaozhiMCPControlRequest{
		DeviceID:  req.DeviceID,
		ToolName:  xiaozhiMCPGetDeviceStatusToolName,
		TraceID:   req.TraceID,
		SessionID: req.SessionID,
	})
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleXiaozhiScreenBrightness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiScreenBrightnessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	response, status, message := s.deliverXiaozhiMCPControl(r.Context(), XiaozhiMCPControlRequest{
		DeviceID:   req.DeviceID,
		ToolName:   xiaozhiMCPScreenSetBrightnessToolName,
		Brightness: req.Brightness,
		TraceID:    req.TraceID,
		SessionID:  req.SessionID,
	})
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleXiaozhiScreenTheme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiScreenThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validXiaozhiNamedScreenTheme(req.Theme) {
		http.Error(w, "theme must be light, dark, or auto", http.StatusBadRequest)
		return
	}
	response, status, message := s.deliverXiaozhiMCPControl(r.Context(), XiaozhiMCPControlRequest{
		DeviceID:  req.DeviceID,
		ToolName:  xiaozhiMCPScreenSetThemeToolName,
		Theme:     req.Theme,
		TraceID:   req.TraceID,
		SessionID: req.SessionID,
	})
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleXiaozhiBodyPreset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiBodyPresetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	preset, plans, err := xiaozhiBodyPresetPlans(req.DeviceID, req.Preset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	steps := make([]XiaozhiBodyPresetStepResponse, 0, len(plans))
	for _, plan := range plans {
		plan.TraceID = traceID
		plan.SessionID = sessionID
		delivery, status, message := s.sendXiaozhiMCPControl(r.Context(), plan)
		if status != 0 {
			s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.body_preset."+preset+".failed", s.now().UnixMilli())
			http.Error(w, message, status)
			return
		}
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		presetMarker := "xiaozhi.body_preset." + preset + "." + delivery.Marker + ".sent"
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, s.now().UnixMilli())
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, presetMarker, s.now().UnixMilli())
		activity := xiaozhiMCPActivity(delivery.Marker, delivery.Args)
		activity["last_body_preset"] = preset
		activity["last_body_preset_status"] = "delivered"
		activity["last_body_preset_tool"] = delivery.Marker
		s.recordXiaozhiDeviceActivity(&xiaozhiSession{
			deviceID:  delivery.Response.DeviceID,
			traceID:   delivery.Response.TraceID,
			sessionID: delivery.Response.SessionID,
		}, presetMarker, activity)
		steps = append(steps, XiaozhiBodyPresetStepResponse{
			ToolName:  delivery.Response.ToolName,
			MCPID:     delivery.Response.MCPID,
			Marker:    delivery.Marker,
			Arguments: delivery.Response.Arguments,
		})
	}
	writeJSON(w, http.StatusOK, XiaozhiBodyPresetResponse{
		SchemaVersion:      "a21.gateway.xiaozhi_body_preset.v1",
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "xiaozhi_mcp_sequence",
		Preset:             preset,
		Steps:              steps,
		ResultRedacted:     true,
		PhysicalAccepted:   false,
	})
}

func (s *Server) handleXiaozhiBodyMotion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiBodyMotionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	motion, plans, err := xiaozhiBodyMotionPlans(req.DeviceID, req.Motion)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	steps := make([]XiaozhiBodyPresetStepResponse, 0, len(plans))
	for index, plan := range plans {
		plan.TraceID = traceID
		plan.SessionID = sessionID
		delivery, status, message := s.sendXiaozhiMCPControl(r.Context(), plan)
		if status != 0 {
			s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.body_motion."+motion+".failed", s.now().UnixMilli())
			http.Error(w, message, status)
			return
		}
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		motionMarker := "xiaozhi.body_motion." + motion + ".step" + strconv.Itoa(index+1) + "." + delivery.Marker + ".sent"
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, s.now().UnixMilli())
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, motionMarker, s.now().UnixMilli())
		activity := xiaozhiMCPActivity(delivery.Marker, delivery.Args)
		activity["last_body_motion"] = motion
		activity["last_body_motion_status"] = "delivered"
		activity["last_body_motion_step"] = strconv.Itoa(index + 1)
		activity["last_body_motion_tool"] = delivery.Marker
		s.recordXiaozhiDeviceActivity(&xiaozhiSession{
			deviceID:  delivery.Response.DeviceID,
			traceID:   delivery.Response.TraceID,
			sessionID: delivery.Response.SessionID,
		}, motionMarker, activity)
		steps = append(steps, XiaozhiBodyPresetStepResponse{
			ToolName:  delivery.Response.ToolName,
			MCPID:     delivery.Response.MCPID,
			Marker:    delivery.Marker,
			Arguments: delivery.Response.Arguments,
		})
	}
	writeJSON(w, http.StatusOK, XiaozhiBodyMotionResponse{
		SchemaVersion:      "a21.gateway.xiaozhi_body_motion.v1",
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "xiaozhi_mcp_sequence",
		Motion:             motion,
		Steps:              steps,
		ResultRedacted:     true,
		PhysicalAccepted:   false,
	})
}

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

func (s *Server) recordPowerLifecyclePhysicalAcceptance(deviceID string, traceID string, sessionID string, observer string, atMS int64, event string) {
	capabilities := map[string]string{
		"power_lifecycle_physical_accepted":             "true",
		"no_cable_cold_boot_physical_accepted":          "true",
		"physical_power_button_start_physical_accepted": "true",
		"gateway_reconnect_after_power_button_accepted": "true",
		"xiaozhi_socket_after_power_button_accepted":    "true",
		"last_power_lifecycle_acceptance_status":        "operator_power_button_accepted",
		"last_power_lifecycle_acceptance_trace_id":      traceID,
		"last_power_lifecycle_acceptance_session_id":    sessionID,
		"last_power_lifecycle_acceptance_observer":      observer,
		"last_power_lifecycle_acceptance_at_ms":         strconv.FormatInt(atMS, 10),
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
			head(0, 22, 180),
		}, nil
	case "listening":
		return preset, []XiaozhiMCPControlRequest{
			led(0, 120, 88),
			head(0, 35, 220),
		}, nil
	case "thinking":
		return preset, []XiaozhiMCPControlRequest{
			led(120, 72, 0),
			head(-12, 28, 180),
		}, nil
	case "speaking":
		return preset, []XiaozhiMCPControlRequest{
			led(20, 24, 168),
			head(12, 30, 220),
		}, nil
	case "celebrate":
		return preset, []XiaozhiMCPControlRequest{
			led(0, 168, 80),
			head(18, 36, 260),
		}, nil
	case "reset_idle":
		return preset, []XiaozhiMCPControlRequest{
			led(0, 0, 32),
			head(0, 18, 180),
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
			head(0, 42, 220),
		}, nil
	case "nod":
		return motion, []XiaozhiMCPControlRequest{
			led(0, 120, 88),
			head(0, 38, 260),
			head(0, 18, 260),
			head(0, 30, 220),
		}, nil
	case "shake":
		return motion, []XiaozhiMCPControlRequest{
			led(120, 60, 0),
			head(-18, 28, 260),
			head(18, 28, 260),
			head(0, 24, 220),
		}, nil
	case "dance":
		return motion, []XiaozhiMCPControlRequest{
			led(168, 80, 0),
			head(-18, 36, 260),
			led(0, 168, 80),
			head(18, 36, 260),
			head(0, 24, 220),
		}, nil
	case "stop":
		return motion, []XiaozhiMCPControlRequest{
			led(0, 0, 32),
			head(0, 18, 200),
		}, nil
	default:
		return "", nil, errors.New("body_motion must be look_up, nod, shake, dance, or stop")
	}
}

func xiaozhiOfficialMCPFallbackPlans(deviceID string, event xiaozhitransport.DeviceExtensionEvent) (string, []XiaozhiMCPControlRequest, error) {
	switch event.Kind {
	case xiaozhitransport.DeviceEventKindState:
		preset := map[string]string{
			"idle":      "reset_idle",
			"listening": "listening",
			"thinking":  "thinking",
			"speaking":  "speaking",
			"error":     "reset_idle",
		}[event.Value]
		if preset == "" {
			return "", nil, errors.New("official state has no safe xiaozhi mcp fallback")
		}
		name, plans, err := xiaozhiBodyPresetPlans(deviceID, preset)
		return "body_preset:" + name, plans, err
	case xiaozhitransport.DeviceEventKindFace:
		preset := map[string]string{
			"idle":      "reset_idle",
			"attentive": "listening",
			"thinking":  "thinking",
			"speaking":  "speaking",
			"happy":     "celebrate",
			"error":     "reset_idle",
		}[event.Value]
		if preset == "" {
			return "", nil, errors.New("official face has no safe xiaozhi mcp fallback")
		}
		name, plans, err := xiaozhiBodyPresetPlans(deviceID, preset)
		return "body_preset:" + name, plans, err
	case xiaozhitransport.DeviceEventKindMotion:
		name, plans, err := xiaozhiBodyMotionPlans(deviceID, event.Value)
		return "body_motion:" + name, plans, err
	default:
		return "", nil, errors.New("official event has no safe xiaozhi mcp fallback")
	}
}

func (s *Server) deliverOfficialStackChanMCPFallback(ctx context.Context, deviceID string, traceID string, sessionID string, event xiaozhitransport.DeviceExtensionEvent, metadata stackchantransport.OfficialActionMetadata) (XiaozhiDeviceControlResponse, int, string) {
	fallbackName, plans, err := xiaozhiOfficialMCPFallbackPlans(deviceID, event)
	if err != nil {
		return XiaozhiDeviceControlResponse{}, http.StatusBadRequest, err.Error()
	}
	if len(plans) == 0 {
		return XiaozhiDeviceControlResponse{}, http.StatusBadRequest, "official stackchan fallback has no safe xiaozhi mcp steps"
	}
	valueToken := safeGatewayFallbackToken(event.Value, "value")
	for index, plan := range plans {
		plan.TraceID = traceID
		plan.SessionID = sessionID
		delivery, status, message := s.sendXiaozhiMCPControl(ctx, plan)
		if status != 0 {
			s.recordTrace(traceID, sessionID, deviceID, "stackchan.official_mcp_fallback.failed", s.now().UnixMilli())
			return XiaozhiDeviceControlResponse{}, status, message
		}
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		fallbackMarker := "stackchan.official_mcp_fallback." + string(event.Kind) + "." + valueToken + ".step" + strconv.Itoa(index+1) + "." + delivery.Marker + ".sent"
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, s.now().UnixMilli())
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, fallbackMarker, s.now().UnixMilli())
		activity := xiaozhiMCPActivity(delivery.Marker, delivery.Args)
		activity["official_stackchan_fallback"] = "xiaozhi_mcp_sequence"
		activity["official_stackchan_fallback_name"] = fallbackName
		activity["official_stackchan_fallback_status"] = "delivered"
		activity["official_stackchan_fallback_step"] = strconv.Itoa(index + 1)
		activity["official_stackchan_event"] = string(event.Kind)
		activity["official_stackchan_value"] = event.Value
		s.recordXiaozhiDeviceActivity(&xiaozhiSession{
			deviceID:  delivery.Response.DeviceID,
			traceID:   delivery.Response.TraceID,
			sessionID: delivery.Response.SessionID,
		}, fallbackMarker, activity)
	}

	surfaces := officialStackChanMCPFallbackSurfaces(metadata.Surfaces, fallbackName, len(plans), metadata.PacketCount)
	s.recordOfficialStackChanMCPFallbackDelivered(deviceID, traceID, sessionID, event, metadata, fallbackName, len(plans), surfaces)
	accepted := false
	return XiaozhiDeviceControlResponse{
		TraceID:                        traceID,
		SessionID:                      sessionID,
		DeviceID:                       deviceID,
		Status:                         "fallback_delivered",
		DeliveredTransport:             "xiaozhi_mcp_sequence",
		Event:                          string(event.Kind),
		Value:                          event.Value,
		OfficialActionPhysicalAccepted: &accepted,
		OfficialActionSurfaces:         surfaces,
		OfficialActionFallbackReason:   "official_stackchan_ws_disconnected",
	}, 0, ""
}

func officialStackChanMCPFallbackSurfaces(metadata map[string]string, fallbackName string, stepCount int, plannedPackets int) map[string]string {
	surfaces := map[string]string{
		"official_relay":         "disconnected",
		"fallback":               "xiaozhi_mcp_sequence",
		"fallback_name":          fallbackName,
		"fallback_steps":         strconv.Itoa(stepCount),
		"planned_packet_count":   strconv.Itoa(plannedPackets),
		"physical_accepted":      "false",
		"packet_delivery_status": "not_sent_no_official_ws",
	}
	for key, value := range metadata {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		surfaces["planned_"+key] = value
	}
	return surfaces
}

func xiaozhiBodyScenePlans(deviceID string, scene string) (string, []XiaozhiMCPControlRequest, error) {
	scene = strings.ToLower(strings.TrimSpace(scene))
	if scene == "" {
		return "", nil, errors.New("body_scene is required")
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
	switch scene {
	case "showtime":
		return scene, []XiaozhiMCPControlRequest{
			theme("dark"),
			brightness(72),
			led(168, 80, 0),
			head(-18, 36, 260),
			led(0, 168, 80),
			head(18, 36, 260),
			head(0, 24, 220),
			led(0, 36, 96),
		}, nil
	case "focus":
		return scene, []XiaozhiMCPControlRequest{
			theme("dark"),
			brightness(62),
			led(120, 72, 0),
			head(-12, 28, 180),
		}, nil
	case "reset":
		return scene, []XiaozhiMCPControlRequest{
			theme("auto"),
			brightness(55),
			led(0, 0, 32),
			head(0, 18, 200),
		}, nil
	case "full_check":
		return scene, []XiaozhiMCPControlRequest{
			theme("dark"),
			brightness(72),
			led(168, 80, 0),
			head(-18, 36, 260),
			led(0, 168, 80),
			head(18, 36, 260),
			head(0, 24, 220),
			led(0, 36, 96),
			theme("dark"),
			brightness(62),
			led(120, 72, 0),
			head(-12, 28, 180),
			theme("auto"),
			brightness(55),
			led(0, 0, 32),
			head(0, 18, 200),
		}, nil
	default:
		return "", nil, errors.New("body_scene must be showtime, focus, reset, or full_check")
	}
}

func xiaozhiPresetInt(value int) *int {
	return &value
}

func xiaozhiMCPAllowedTools() []string {
	return []string{
		xiaozhiSpeakerVolumeToolName,
		xiaozhiMCPGetDeviceStatusToolName,
		xiaozhiMCPScreenSetBrightnessToolName,
		xiaozhiMCPScreenSetThemeToolName,
		xiaozhiMCPScreenGetInfoToolName,
		xiaozhiMCPRobotGetHeadAnglesToolName,
		xiaozhiMCPRobotSetHeadAnglesToolName,
		xiaozhiMCPRobotSetLEDColorToolName,
	}
}

func xiaozhiMCPBlockedToolClasses() []string {
	return []string{
		"reboot",
		"firmware_upgrade",
		"camera_photo",
		"screen_snapshot",
		"camera_stream_video",
		"nfc",
		"infrared",
		"power_shutdown",
		"power_sleep",
		"app_lifecycle",
	}
}

func xiaozhiMCPHasAnyArgument(req XiaozhiMCPControlRequest) bool {
	return xiaozhiMCPHasAnyArgumentExcept(req)
}

func xiaozhiMCPHasAnyArgumentExcept(req XiaozhiMCPControlRequest, allowed ...string) bool {
	allowedSet := map[string]bool{}
	for _, name := range allowed {
		allowedSet[name] = true
	}
	if req.Brightness != nil && !allowedSet["brightness"] {
		return true
	}
	if req.Volume != nil && !allowedSet["volume"] {
		return true
	}
	if strings.TrimSpace(req.Theme) != "" && !allowedSet["theme"] {
		return true
	}
	if req.Yaw != nil && !allowedSet["yaw"] {
		return true
	}
	if req.Pitch != nil && !allowedSet["pitch"] {
		return true
	}
	if req.Speed != nil && !allowedSet["speed"] {
		return true
	}
	if req.Red != nil && !allowedSet["red"] {
		return true
	}
	if req.Green != nil && !allowedSet["green"] {
		return true
	}
	if req.Blue != nil && !allowedSet["blue"] {
		return true
	}
	return false
}

func validXiaozhiScreenTheme(theme string) bool {
	if theme == "" || len(theme) > 32 {
		return false
	}
	for _, ch := range theme {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			continue
		}
		return false
	}
	return true
}

func validXiaozhiNamedScreenTheme(theme string) bool {
	switch strings.ToLower(strings.TrimSpace(theme)) {
	case "light", "dark", "auto":
		return true
	default:
		return false
	}
}

func (s *Server) suppressXiaozhiInputAfterHostSay(session *xiaozhiSession, task xiaozhiTurnTask) {
	if session == nil || !hardwareMACDeviceID(session.deviceID) || xiaozhiClientProfile(session.features) == "debug" {
		return
	}
	untilMS := s.now().UnixMilli() + xiaozhiHostSayInputCooldownMS
	session.suppressXiaozhiInputUntil(untilMS, "after_host_say")
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.say.input_suppression_armed", s.now().UnixMilli())
}

func (s *Server) suppressXiaozhiInputAfterNoSpeechPlaceholder(session *xiaozhiSession, task xiaozhiTurnTask) {
	if session == nil || !hardwareMACDeviceID(session.deviceID) || xiaozhiClientProfile(session.features) == "debug" {
		return
	}
	untilMS := s.now().UnixMilli() + xiaozhiNoSpeechInputCooldownMS
	session.suppressXiaozhiInputUntil(untilMS, "after_no_speech")
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.no_speech.input_suppression_armed", s.now().UnixMilli())
}

func (s *Server) handleOfficialStackChanControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiDeviceControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	event, err := xiaozhiDeviceControlEventFromRequest(req)
	if err != nil {
		http.Error(w, "invalid official stackchan event", http.StatusBadRequest)
		return
	}
	plan, err := stackchantransport.BuildOfficialActionPlan(event)
	if err != nil {
		http.Error(w, "unsupported official stackchan event", http.StatusBadRequest)
		return
	}
	event = plan.Event
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	socket, ok := s.officialStackChanSocket(req.DeviceID)
	if !ok {
		if req.AllowMCPFallback {
			response, status, message := s.deliverOfficialStackChanMCPFallback(r.Context(), req.DeviceID, traceID, sessionID, event, plan.Metadata)
			if status != 0 {
				http.Error(w, message, status)
				return
			}
			writeJSON(w, http.StatusOK, response)
			return
		}
		http.Error(w, "official stackchan websocket is not connected", http.StatusConflict)
		return
	}
	socket.writeMu.Lock()
	for _, packet := range plan.Packets {
		err = socket.conn.Write(r.Context(), websocket.MessageBinary, packet.Bytes())
		if err != nil {
			break
		}
	}
	socket.writeMu.Unlock()
	if err != nil {
		http.Error(w, "official stackchan command delivery failed", http.StatusBadGateway)
		return
	}
	s.recordOfficialStackChanControlDelivered(req.DeviceID, traceID, sessionID, event, plan.Metadata)
	writeJSON(w, http.StatusOK, XiaozhiDeviceControlResponse{
		TraceID:                        traceID,
		SessionID:                      sessionID,
		DeviceID:                       req.DeviceID,
		Status:                         "delivered",
		DeliveredTransport:             "stackchan_official_ws",
		Event:                          string(event.Kind),
		Value:                          event.Value,
		PacketCount:                    plan.Metadata.PacketCount,
		OfficialActionPhysicalAccepted: boolValue(plan.Metadata.PhysicalAccepted),
		OfficialActionSurfaces:         plan.Metadata.Surfaces,
	})
}

func (s *Server) handleOfficialStackChanStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if deviceID == "" {
		deviceID = defaultOfficialStackChanDeviceID
	}
	if !validA21DeviceID(deviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, s.officialStackChanStatus(deviceID))
}

func xiaozhiDeviceControlEventFromRequest(req XiaozhiDeviceControlRequest) (xiaozhitransport.DeviceExtensionEvent, error) {
	kind := strings.TrimSpace(strings.ToLower(firstNonEmpty(req.Event, req.Kind)))
	if kind == "" {
		switch {
		case strings.TrimSpace(req.State) != "":
			kind = string(xiaozhitransport.DeviceEventKindState)
		case strings.TrimSpace(firstNonEmpty(req.Emotion, req.Face)) != "":
			kind = string(xiaozhitransport.DeviceEventKindFace)
		case strings.TrimSpace(firstNonEmpty(req.Slot, req.Display)) != "":
			kind = string(xiaozhitransport.DeviceEventKindDisplay)
		case strings.TrimSpace(firstNonEmpty(req.Name, req.Motion)) != "":
			kind = string(xiaozhitransport.DeviceEventKindMotion)
		}
	}
	value := ""
	switch xiaozhitransport.DeviceEventKind(kind) {
	case xiaozhitransport.DeviceEventKindState:
		value = req.State
	case xiaozhitransport.DeviceEventKindFace:
		value = firstNonEmpty(req.Emotion, req.Face)
	case xiaozhitransport.DeviceEventKindDisplay:
		value = firstNonEmpty(req.Slot, req.Display)
	case xiaozhitransport.DeviceEventKindMotion:
		value = firstNonEmpty(req.Name, req.Motion)
	}
	return xiaozhitransport.NormalizeDeviceExtensionEvent(xiaozhitransport.DeviceExtensionEvent{
		Kind:   xiaozhitransport.DeviceEventKind(kind),
		Value:  value,
		YAngle: req.YAngle,
		Text:   req.Text,
		Reason: req.Reason,
	})
}

func (s *Server) applyDeviceControlStreamState(req DeviceControlRequest) {
	s.setAudioProbeOnly(req.DeviceID, req.TraceID, req.SessionID, req.AudioProbeOnly)
	s.setMockPlaybackOnNextAudioFrame(req.DeviceID, req.TraceID, req.SessionID, req.MockPlaybackOnNextAudioFrame, req.MockAudioChunks)
	s.setRealtimeOnNextSpeech(req.DeviceID, req.TraceID, req.SessionID, req.RealtimeOnNextSpeech)
}

func (s *Server) clearDeviceControlStreamState(req DeviceControlRequest) {
	s.setAudioProbeOnly(req.DeviceID, req.TraceID, req.SessionID, false)
	s.setMockPlaybackOnNextAudioFrame(req.DeviceID, req.TraceID, req.SessionID, false, nil)
	s.setRealtimeOnNextSpeech(req.DeviceID, req.TraceID, req.SessionID, false)
}

func (s *Server) handleTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	traceID := r.URL.Query().Get("trace_id")
	if traceID == "" {
		http.Error(w, "trace_id is required", http.StatusBadRequest)
		return
	}
	events := s.traceEvents(traceID)
	writeJSON(w, http.StatusOK, TraceResponse{TraceID: traceID, Events: events, Summary: traceLatencySummary(events)})
}

func (s *Server) handleAudioRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !loopbackHTTPRemote(r.RemoteAddr) {
		http.Error(w, "audio capture is available only from loopback", http.StatusForbidden)
		return
	}
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	traceID := strings.TrimSpace(r.URL.Query().Get("trace_id"))
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if deviceID != "" && !validA21DeviceID(deviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	if traceID == "" && sessionID == "" {
		http.Error(w, "trace_id or session_id is required", http.StatusBadRequest)
		return
	}
	limit := 64
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 || parsed > maxAudioCaptureFrames {
			http.Error(w, "limit must be between 1 and 512", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	includeAudio := r.URL.Query().Get("include_audio") == "1"
	writeJSON(w, http.StatusOK, AudioRecentResponse{
		SchemaVersion: AudioRecentSchemaVersion,
		DeviceID:      deviceID,
		TraceID:       traceID,
		SessionID:     sessionID,
		IncludeAudio:  includeAudio,
		Frames:        s.recentAudioFrames(deviceID, traceID, sessionID, limit, includeAudio),
	})
}

func loopbackHTTPRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "a21-gateway",
		"status":  "ok",
		"version": buildinfo.Version,
	})
}

func (s *Server) handleVoiceProviderHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	health, err := s.currentVoiceProvider().Health(ctx)
	status := http.StatusOK
	if err != nil || health.Status == providers.VoiceProviderUnavailable {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, health)
}

func (s *Server) handleRealtimeSessionStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RealtimeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	if req.Mode == protocol.ModeProfessional {
		http.Error(w, "professional mode must use the professional path with V21 evidence", http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	s.metrics.realtimeSessionTotal.Inc()
	s.recordTrace(traceID, sessionID, req.DeviceID, "realtime.session.start.received", s.now().UnixMilli())

	outputs, err := s.startRealtimeVoiceTurn(r.Context(), req)
	status := "completed"
	code := http.StatusOK
	if err != nil {
		status = "error"
		code = http.StatusBadGateway
		outputs = []realtimeVoiceOutput{{Control: protocol.ControlEventPayload{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: err.Error(), Final: true}}}
	}
	events := s.realtimeOutputSequence(req.DeviceID, traceID, sessionID, outputs)
	writeJSON(w, code, RealtimeSessionResponse{
		TraceID:   traceID,
		SessionID: sessionID,
		DeviceID:  req.DeviceID,
		Provider:  s.currentVoiceProvider().Name(),
		Status:    status,
		Events:    events,
	})
}

func (s *Server) startRealtimeVoiceTurn(ctx context.Context, req RealtimeSessionRequest) ([]realtimeVoiceOutput, error) {
	started := time.Now()
	s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.start_turn.start", s.now().UnixMilli())
	providerEvents, err := s.currentVoiceProvider().StartTurn(ctx, providers.VoiceTurnRequest{
		Session: providers.VoiceSession{TraceID: req.TraceID, SessionID: req.SessionID, DeviceID: req.DeviceID},
		Text:    req.Text,
		Mode:    string(req.Mode),
	})
	s.metrics.voiceProviderStartTurnMS.Observe(float64(time.Since(started)) / float64(time.Millisecond))
	if err != nil {
		s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.start_turn.error", s.now().UnixMilli())
		return nil, err
	}
	outputs := make([]realtimeVoiceOutput, 0, 4)
	firstEvent := true
	for event := range providerEvents {
		if firstEvent {
			s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.start_turn.first_event", s.now().UnixMilli())
			firstEvent = false
		}
		payload := voiceEventToControlPayload(event, req.Mode)
		if payload.State == protocol.ExpressionSpeaking && payload.StreamID != "" {
			s.setActiveStream(req.TraceID, req.SessionID, req.DeviceID, payload.StreamID)
		}
		outputs = append(outputs, realtimeVoiceOutput{Control: payload, Audio: event.Audio})
	}
	s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.start_turn.end", s.now().UnixMilli())
	return outputs, nil
}

func (s *Server) handleRealtimeSessionCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RealtimeSessionCancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	if req.Reason == "" {
		req.Reason = providers.CancelBargeIn
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	if req.StreamID == "" {
		req.StreamID = s.clearActiveStream(traceID, sessionID, req.DeviceID)
	} else {
		s.clearActiveStream(traceID, sessionID, req.DeviceID)
	}
	s.metrics.realtimeSessionCancelTotal.Inc()
	s.recordTrace(traceID, sessionID, req.DeviceID, "realtime.session.cancel.received", s.now().UnixMilli())

	outputs, err := s.cancelRealtimeVoiceTurn(r.Context(), req)
	status := "cancelled"
	code := http.StatusOK
	if err != nil {
		status = "error"
		code = http.StatusBadGateway
		outputs = []realtimeVoiceOutput{{Control: protocol.ControlEventPayload{State: protocol.ExpressionInterrupted, Mode: req.Mode, Text: "好，我听新的。", Final: true, StreamID: req.StreamID}}}
	}
	events := s.realtimeOutputSequence(req.DeviceID, traceID, sessionID, outputs)
	writeJSON(w, code, RealtimeSessionResponse{
		TraceID:   traceID,
		SessionID: sessionID,
		DeviceID:  req.DeviceID,
		Provider:  s.currentVoiceProvider().Name(),
		Status:    status,
		Events:    events,
	})
}

func (s *Server) cancelRealtimeVoiceTurn(ctx context.Context, req RealtimeSessionCancelRequest) ([]realtimeVoiceOutput, error) {
	started := time.Now()
	s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.cancel.start", s.now().UnixMilli())
	providerEvents, err := s.currentVoiceProvider().Cancel(ctx, providers.VoiceCancelRequest{
		Session:  providers.VoiceSession{TraceID: req.TraceID, SessionID: req.SessionID, DeviceID: req.DeviceID},
		Reason:   req.Reason,
		StreamID: req.StreamID,
	})
	s.metrics.voiceProviderCancelMS.Observe(float64(time.Since(started)) / float64(time.Millisecond))
	if err != nil {
		s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.cancel.error", s.now().UnixMilli())
		return nil, err
	}
	outputs := make([]realtimeVoiceOutput, 0, 2)
	for event := range providerEvents {
		outputs = append(outputs, realtimeVoiceOutput{Control: voiceEventToControlPayload(event, req.Mode), Audio: event.Audio})
	}
	s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.cancel.end", s.now().UnixMilli())
	return outputs, nil
}

func (s *Server) handleMockTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req MockTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	s.recordTrace(traceID, sessionID, req.DeviceID, "http.mock_turn.received", s.now().UnixMilli())
	writeJSON(w, http.StatusOK, s.mockTurnResponse(req))
}

func (s *Server) handleMockInterrupt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req MockTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	s.recordTrace(traceID, sessionID, req.DeviceID, "http.mock_interrupt.received", s.now().UnixMilli())
	writeJSON(w, http.StatusOK, s.mockInterruptResponse(req))
}

type xiaozhiSession struct {
	mu                             sync.Mutex
	writeMu                        sync.Mutex
	traceID                        string
	sessionID                      string
	deviceID                       string
	features                       xiaozhitransport.HelloFeatures
	currentTurn                    *xiaozhiTurn
	nextTurnID                     uint64
	helloReceived                  bool
	listening                      bool
	listenStartedAtMS              int64
	suppressedListenActive         bool
	binaryProtocolVersion          int
	opusCodec                      *opuscodec.Codec
	opusSampleRateHz               int
	opusChannels                   int
	opusFrameDurationMS            int
	opusFrameCount                 int
	opusByteCount                  int
	opusDecodedFrameCount          int
	opusDecodedSampleCount         int
	opusDecodeErrorCount           int
	opusIngressProcessedFrameCount int
	voicePipelineFrames            []providers.VoicePipelinePCMFrame
	voicePipelineHasSpeech         bool
	ttsStopSent                    bool
	lastDownlinkAtMS               int64
	lastDownlinkTurnID             string
	lastPlaybackStopDoneAtMS       int64
	inputCooldownUntilMS           int64
	inputCooldownReason            string
	officialStackChanState         string
	xiaozhiStateReactionState      string
	streamingASRSession            providers.StreamingASRSession
	streamingASRHasPartial         bool
	streamingASRPartialText        string
	streamingASRHasFinal           bool
	streamingASRFinalText          string
	streamingASRClosed             bool
	streamingASRCommitStarted      bool
	streamingASRAnswerStarted      bool
	wakePrerollFrames              []providers.VoicePipelinePCMFrame
	wakePrerollPayloadBytes        []int
	wakePrerollHasSpeech           bool
	opusIngressQueue               chan xiaozhiOpusIngressFrame
	opusIngressCtx                 context.Context
	opusIngressCancel              context.CancelFunc
}

type xiaozhiTurn struct {
	id           uint64
	ctx          context.Context
	cancel       context.CancelCauseFunc
	pacer        *audio.AudioRateController
	downlink     *xiaozhiTurnDownlinkCodec
	mode         protocol.Mode
	cancelReason string
}

type xiaozhiTurnDownlinkCodec struct {
	sampleRateHz int
	channels     int
	durationMS   int
	codec        *opuscodec.Codec
}

type xiaozhiOpusIngressFrame struct {
	ctx   context.Context
	conn  *websocket.Conn
	frame xiaozhitransport.Frame
	turn  *xiaozhiTurn
	seq   uint64
}

type xiaozhiVoicePipelineRunner interface {
	Run(context.Context, providers.VoicePipelineRequest) (providers.VoicePipelineResult, error)
}

type xiaozhiVoicePipelineStreamer interface {
	RunStream(context.Context, providers.VoicePipelineRequest) (<-chan providers.VoicePipelineStreamEvent, error)
}

type xiaozhiTurnTask struct {
	turn                      *xiaozhiTurn
	turnID                    string
	traceID                   string
	sessionID                 string
	deviceID                  string
	audioIngressBase          map[string]any
	voicePipelineFrames       []providers.VoicePipelinePCMFrame
	voicePipelineHasSpeech    bool
	streamingASRPartialText   string
	streamingASRPartialDriven bool
	streamingASRFinalText     string
	streamingASRUsed          bool
	mode                      protocol.Mode
}

type xiaozhiSessionIdentity struct {
	deviceID  string
	traceID   string
	sessionID string
}

func defaultXiaozhiVoicePipelineRunner() xiaozhiVoicePipelineRunner {
	return providers.NewVoicePipelineRunner(providers.VoicePipelineAdapters{
		ASR:        providers.NewMockASRAdapter("mock-local-asr"),
		TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
		TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
		Selection:  providers.VoicePipelineSelectionFromEnv(nil),
	})
}

func (session *xiaozhiSession) identitySnapshot() xiaozhiSessionIdentity {
	if session == nil {
		return xiaozhiSessionIdentity{}
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.identityLocked()
}

func (session *xiaozhiSession) identityLocked() xiaozhiSessionIdentity {
	return xiaozhiSessionIdentity{
		deviceID:  session.deviceID,
		traceID:   session.traceID,
		sessionID: session.sessionID,
	}
}

func (session *xiaozhiSession) identity() xiaozhitransport.Identity {
	id := session.identitySnapshot()
	return xiaozhitransport.Identity{
		DeviceID:  id.deviceID,
		TraceID:   id.traceID,
		SessionID: id.sessionID,
	}
}

func (session *xiaozhiSession) adoptFrame(frame xiaozhitransport.Frame) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.deviceID = frame.DeviceID
	session.traceID = frame.TraceID
	session.sessionID = frame.SessionID
}

func (session *xiaozhiSession) featuresSnapshot() xiaozhitransport.HelloFeatures {
	if session == nil {
		return xiaozhitransport.HelloFeatures{}
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.features
}

func (session *xiaozhiSession) helloReceivedSnapshot() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.helloReceived
}

func (session *xiaozhiSession) binaryProtocolVersionSnapshot() int {
	if session == nil {
		return 1
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.binaryProtocolVersion
}

func (session *xiaozhiSession) setXiaozhiListening(listening bool, startedAtMS int64) {
	if session == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	session.listening = listening
	if listening {
		session.listenStartedAtMS = startedAtMS
	} else {
		session.listenStartedAtMS = 0
	}
}

func (session *xiaozhiSession) xiaozhiListening() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.listening
}

func (session *xiaozhiSession) startXiaozhiTurn(parent context.Context, mode protocol.Mode) *xiaozhiTurn {
	if parent == nil {
		parent = context.Background()
	}
	if mode == "" {
		mode = protocol.ModeWorkmate
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	session.cancelCurrentXiaozhiTurnLocked("new_turn")
	session.nextTurnID++
	ctx, cancel := context.WithCancelCause(parent)
	turn := &xiaozhiTurn{
		id:     session.nextTurnID,
		ctx:    ctx,
		cancel: cancel,
		mode:   mode,
		pacer: audio.NewAudioRateController(audio.AudioRateControllerConfig{
			FrameDuration:   60 * time.Millisecond,
			PrebufferFrames: 1,
		}),
	}
	session.currentTurn = turn
	return turn
}

func (session *xiaozhiSession) setCurrentXiaozhiTurnMode(mode protocol.Mode) {
	if mode == "" {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.currentTurn != nil {
		session.currentTurn.mode = mode
	}
}

func (session *xiaozhiSession) currentXiaozhiTurn() *xiaozhiTurn {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.currentTurn
}

func (session *xiaozhiSession) currentXiaozhiTurnID() string {
	session.mu.Lock()
	defer session.mu.Unlock()
	return xiaozhiTurnID(session.currentTurn)
}

func (session *xiaozhiSession) cancelCurrentXiaozhiTurn(reason string) *xiaozhiTurn {
	session.mu.Lock()
	turn := session.cancelCurrentXiaozhiTurnLocked(reason)
	session.mu.Unlock()
	return turn
}

func (session *xiaozhiSession) cancelCurrentXiaozhiTurnLocked(reason string) *xiaozhiTurn {
	if session.currentTurn == nil {
		return nil
	}
	turn := session.currentTurn
	turn.cancelReason = strings.TrimSpace(reason)
	turn.cancel(xiaozhiTurnCancelCause(reason))
	session.currentTurn = nil
	return turn
}

func (session *xiaozhiSession) prepareXiaozhiListenStartBargeIn(reason string, nowMS int64, recentWindowMS int64) (xiaozhiTurnTask, bool) {
	session.mu.Lock()
	turn := session.cancelCurrentXiaozhiTurnLocked(reason)
	turnID := xiaozhiTurnID(turn)
	if turnID == "" && session.lastDownlinkTurnID != "" && nowMS-session.lastDownlinkAtMS >= 0 && nowMS-session.lastDownlinkAtMS <= recentWindowMS {
		if session.lastPlaybackStopDoneAtMS >= session.lastDownlinkAtMS && session.lastPlaybackStopDoneAtMS <= nowMS {
			session.mu.Unlock()
			return xiaozhiTurnTask{}, false
		}
		turnID = session.lastDownlinkTurnID
	}
	if turnID == "" {
		session.mu.Unlock()
		return xiaozhiTurnTask{}, false
	}
	task := xiaozhiTurnTask{
		turn:      turn,
		turnID:    turnID,
		traceID:   session.traceID,
		sessionID: session.sessionID,
		deviceID:  session.deviceID,
		mode:      xiaozhiTurnMode(turn),
	}
	session.mu.Unlock()
	return task, true
}

func (session *xiaozhiSession) suppressXiaozhiInputUntil(untilMS int64, reason string) {
	session.mu.Lock()
	defer session.mu.Unlock()
	if untilMS > session.inputCooldownUntilMS {
		session.inputCooldownUntilMS = untilMS
		session.inputCooldownReason = safeGatewayFallbackToken(reason, "cooldown")
	}
}

func (session *xiaozhiSession) xiaozhiInputSuppression(nowMS int64) (bool, string) {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.inputCooldownUntilMS > 0 && nowMS >= 0 && nowMS < session.inputCooldownUntilMS {
		return true, firstNonEmpty(session.inputCooldownReason, "cooldown")
	}
	return false, ""
}

func (session *xiaozhiSession) xiaozhiSuppressedListenActive() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.suppressedListenActive
}

func (session *xiaozhiSession) clearSuppressedXiaozhiListen() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if !session.suppressedListenActive {
		return false
	}
	session.suppressedListenActive = false
	return true
}

func (session *xiaozhiSession) cancelXiaozhiTurnContext(turn *xiaozhiTurn, reason string) {
	if turn == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	turn.cancelReason = strings.TrimSpace(reason)
	turn.cancel(xiaozhiTurnCancelCause(reason))
	if turn.pacer != nil {
		turn.pacer.Reset()
	}
}

func (session *xiaozhiSession) xiaozhiTurnCancelReason(turn *xiaozhiTurn) string {
	if session == nil || turn == nil {
		return ""
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return strings.TrimSpace(turn.cancelReason)
}

func (session *xiaozhiSession) shouldAbortXiaozhiTurn(turn *xiaozhiTurn) bool {
	if turn == nil {
		return true
	}
	session.mu.Lock()
	currentTurn := session.currentTurn
	session.mu.Unlock()
	return currentTurn != turn || turn.ctx.Err() != nil
}

func xiaozhiUserInterruptReason(reason string) (string, bool) {
	reason = safeGatewayFallbackToken(reason, "")
	if reason == "" {
		return "", false
	}
	lower := strings.ToLower(reason)
	if strings.Contains(lower, "error") || strings.Contains(lower, "unavailable") {
		return "", false
	}
	if strings.Contains(lower, "barge") ||
		strings.Contains(lower, "wake") ||
		strings.Contains(lower, "abort") ||
		strings.Contains(lower, "interrupt") {
		return reason, true
	}
	return "", false
}

func (session *xiaozhiSession) completeXiaozhiTurn(turn *xiaozhiTurn) {
	if turn == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.currentTurn == turn {
		session.currentTurn = nil
	}
}

func xiaozhiTurnID(turn *xiaozhiTurn) string {
	if turn == nil || turn.id == 0 {
		return ""
	}
	return fmt.Sprintf("a21-xiaozhi-turn-%06d", turn.id)
}

func xiaozhiTurnMode(turn *xiaozhiTurn) protocol.Mode {
	if turn == nil || turn.mode == "" {
		return protocol.ModeWorkmate
	}
	return turn.mode
}

func (session *xiaozhiSession) resetXiaozhiTTSStop() {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.ttsStopSent = false
}

func (session *xiaozhiSession) claimXiaozhiTTSStop() bool {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.ttsStopSent {
		return false
	}
	session.ttsStopSent = true
	return true
}

func (session *xiaozhiSession) claimOfficialStackChanState(state string, force bool) bool {
	state = strings.TrimSpace(state)
	if state == "" {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if !force && session.officialStackChanState == state {
		return false
	}
	session.officialStackChanState = state
	return true
}

func (session *xiaozhiSession) claimXiaozhiStateReaction(state string, force bool) bool {
	state = strings.TrimSpace(state)
	if state == "" {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if !force && session.xiaozhiStateReactionState == state {
		return false
	}
	session.xiaozhiStateReactionState = state
	return true
}

func (session *xiaozhiSession) markXiaozhiDownlink(turn *xiaozhiTurn, atMS int64) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.lastDownlinkAtMS = atMS
	session.lastDownlinkTurnID = xiaozhiTurnID(turn)
}

func xiaozhiTurnCancelCause(reason string) error {
	if xiaozhiAbortIsBargeIn(reason) {
		return providers.ErrVoicePipelineBargeIn
	}
	return context.Canceled
}

func xiaozhiAbortIsBargeIn(reason string) bool {
	reason = strings.ToLower(strings.TrimSpace(reason))
	return reason == "" || strings.Contains(reason, "barge") || strings.Contains(reason, "wake")
}

func (session *xiaozhiSession) writeXiaozhiJSON(ctx context.Context, conn *websocket.Conn, turn *xiaozhiTurn, value any) error {
	if conn == nil {
		return fmt.Errorf("xiaozhi websocket is nil")
	}
	session.writeMu.Lock()
	defer session.writeMu.Unlock()
	if turn != nil && session.shouldAbortXiaozhiTurn(turn) {
		return context.Canceled
	}
	return wsjson.Write(ctx, conn, value)
}

func (session *xiaozhiSession) writeXiaozhiBinary(ctx context.Context, conn *websocket.Conn, turn *xiaozhiTurn, frame []byte) error {
	if conn == nil {
		return fmt.Errorf("xiaozhi websocket is nil")
	}
	session.writeMu.Lock()
	defer session.writeMu.Unlock()
	if turn != nil && session.shouldAbortXiaozhiTurn(turn) {
		return context.Canceled
	}
	return conn.Write(ctx, websocket.MessageBinary, frame)
}

func (session *xiaozhiSession) configureXiaozhiAudio(params xiaozhitransport.AudioParams) error {
	codec, err := opuscodec.New(params.SampleRate, params.Channels, params.FrameDuration)
	if err != nil {
		return err
	}
	session.mu.Lock()
	session.opusCodec = codec
	session.opusSampleRateHz = params.SampleRate
	session.opusChannels = params.Channels
	session.opusFrameDurationMS = params.FrameDuration
	stream, shouldCancel := session.resetXiaozhiOpusIngressLocked()
	session.mu.Unlock()
	if shouldCancel {
		stream.Cancel(context.Canceled)
	}
	return nil
}

func (session *xiaozhiSession) resetXiaozhiOpusIngress() {
	session.mu.Lock()
	stream, shouldCancel := session.resetXiaozhiOpusIngressLocked()
	session.mu.Unlock()
	if shouldCancel {
		stream.Cancel(context.Canceled)
	}
}

func (session *xiaozhiSession) resetXiaozhiOpusIngressLocked() (providers.StreamingASRSession, bool) {
	session.opusFrameCount = 0
	session.opusByteCount = 0
	session.opusDecodedFrameCount = 0
	session.opusDecodedSampleCount = 0
	session.opusDecodeErrorCount = 0
	session.opusIngressProcessedFrameCount = 0
	session.voicePipelineFrames = nil
	session.voicePipelineHasSpeech = false
	stream := session.streamingASRSession
	shouldCancel := stream != nil && !session.streamingASRClosed
	if session.streamingASRSession != nil && !session.streamingASRClosed {
		session.streamingASRClosed = true
	}
	session.streamingASRSession = nil
	session.streamingASRHasPartial = false
	session.streamingASRPartialText = ""
	session.streamingASRHasFinal = false
	session.streamingASRFinalText = ""
	session.streamingASRClosed = false
	session.streamingASRAnswerStarted = false
	return stream, shouldCancel
}

func (session *xiaozhiSession) xiaozhiOpusIngressQueue() chan xiaozhiOpusIngressFrame {
	if session == nil {
		return nil
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.opusIngressQueue
}

func (session *xiaozhiSession) resetXiaozhiWakePreroll() {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.wakePrerollFrames = nil
	session.wakePrerollPayloadBytes = nil
	session.wakePrerollHasSpeech = false
}

func (session *xiaozhiSession) shouldBufferXiaozhiWakePreroll(nowMS int64) bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.currentTurn != nil {
		return false
	}
	if session.inputCooldownUntilMS > 0 && nowMS >= 0 && nowMS < session.inputCooldownUntilMS {
		return false
	}
	return true
}

func (session *xiaozhiSession) xiaozhiOpusDecodeStatus() string {
	if session == nil {
		return XiaozhiOpusDecodeErrorState
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return xiaozhiOpusDecodeStatusFromCounts(session.opusDecodedFrameCount, session.opusDecodeErrorCount)
}

func xiaozhiOpusDecodeStatusFromCounts(decodedFrameCount int, decodeErrorCount int) string {
	if decodeErrorCount > 0 && decodedFrameCount > 0 {
		return XiaozhiOpusPartialDecodeErrorState
	}
	if decodeErrorCount > 0 {
		return XiaozhiOpusDecodeErrorState
	}
	if decodedFrameCount > 0 {
		return XiaozhiOpusDecodedPCMState
	}
	return XiaozhiOpusNoFramesState
}

func (session *xiaozhiSession) xiaozhiDecodedDurationMS() int {
	if session == nil {
		return 0
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.xiaozhiDecodedDurationMSLocked()
}

func (session *xiaozhiSession) xiaozhiDecodedDurationMSLocked() int {
	if session.opusSampleRateHz <= 0 {
		return 0
	}
	return session.opusDecodedSampleCount * 1000 / session.opusSampleRateHz
}

func copyXiaozhiVoicePipelineFrames(frames []providers.VoicePipelinePCMFrame) []providers.VoicePipelinePCMFrame {
	if len(frames) == 0 {
		return nil
	}
	copied := append([]providers.VoicePipelinePCMFrame(nil), frames...)
	for i := range copied {
		copied[i].PCM16LE = append([]byte(nil), copied[i].PCM16LE...)
	}
	return copied
}

func (s *Server) startXiaozhiStreamingASR(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, mode protocol.Mode) {
	if s == nil || session == nil {
		return
	}
	streamingASR := s.currentXiaozhiVoicePipelineASR()
	if streamingASR == nil {
		return
	}
	streaming, ok := streamingASR.(providers.StreamingASRAdapter)
	if !ok {
		return
	}
	id := session.identitySnapshot()
	stream, err := streaming.StartStreamingASR(ctx, providers.StreamingASRStartRequest{
		Session: providers.VoiceSession{
			TraceID:   id.traceID,
			SessionID: id.sessionID,
			DeviceID:  id.deviceID,
		},
		Mode: string(mode),
	})
	if err != nil {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.unavailable", s.now().UnixMilli())
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.start_failed."+xiaozhiStreamingASRErrorCode(err), s.now().UnixMilli())
		return
	}
	session.mu.Lock()
	session.streamingASRSession = stream
	session.streamingASRHasPartial = false
	session.streamingASRPartialText = ""
	session.streamingASRHasFinal = false
	session.streamingASRFinalText = ""
	session.streamingASRClosed = false
	session.streamingASRCommitStarted = false
	session.streamingASRAnswerStarted = false
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.start", s.now().UnixMilli())
	go s.consumeXiaozhiStreamingASREvents(ctx, conn, session, stream)
}

func xiaozhiStreamingASRErrorCode(err error) string {
	message := strings.ToLower(strings.TrimSpace(errString(err)))
	switch {
	case message == "":
		return "unknown"
	case strings.Contains(message, "missing"):
		return "missing_env"
	case strings.Contains(message, "dial") || strings.Contains(message, "websocket"):
		return "dial_failed"
	case strings.Contains(message, "session") || strings.Contains(message, "update"):
		return "session_update_failed"
	default:
		return "provider_failed"
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *Server) consumeXiaozhiStreamingASREvents(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, stream providers.StreamingASRSession) {
	for event := range stream.Events() {
		if event.Err != nil {
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.error", s.now().UnixMilli())
			continue
		}
		partialText := strings.TrimSpace(event.Text)
		if partialText != "" {
			recordPartial := false
			session.mu.Lock()
			if !session.streamingASRHasPartial {
				session.streamingASRHasPartial = true
				session.streamingASRPartialText = partialText
				recordPartial = true
			}
			session.mu.Unlock()
			if recordPartial {
				id := session.identitySnapshot()
				s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.first_partial", s.now().UnixMilli())
				s.startXiaozhiPartialVoicePipeline(ctx, conn, session, partialText)
			}
		}
		if event.Final {
			session.mu.Lock()
			session.streamingASRHasFinal = true
			session.streamingASRFinalText = event.Text
			if strings.TrimSpace(event.Text) != "" {
				session.voicePipelineHasSpeech = true
			}
			session.mu.Unlock()
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.final", s.now().UnixMilli())
			s.maybeStartXiaozhiStreamingASRFinalAnswer(ctx, conn, session)
		}
	}
	session.mu.Lock()
	if session.streamingASRSession == stream {
		session.streamingASRClosed = true
	}
	session.mu.Unlock()
}

func (s *Server) startXiaozhiPartialVoicePipeline(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, partialText string) {
	if s == nil || session == nil || conn == nil || strings.TrimSpace(partialText) == "" {
		return
	}
	session.mu.Lock()
	if session.streamingASRAnswerStarted || !session.listening || session.currentTurn == nil || len(session.voicePipelineFrames) == 0 {
		session.mu.Unlock()
		return
	}
	session.voicePipelineHasSpeech = true
	session.mu.Unlock()

	id := session.identitySnapshot()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.voice_pipeline.partial_prewarm_deferred", s.now().UnixMilli())
}

func (session *xiaozhiSession) claimXiaozhiStreamingASRAnswer(turn *xiaozhiTurn) bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if turn == nil || session.currentTurn != turn || session.streamingASRAnswerStarted {
		return false
	}
	session.streamingASRAnswerStarted = true
	return true
}

func (s *Server) appendXiaozhiStreamingASRFrame(ctx context.Context, session *xiaozhiSession, frame providers.VoicePipelinePCMFrame) {
	if session == nil {
		return
	}
	session.mu.Lock()
	stream := session.streamingASRSession
	closed := session.streamingASRClosed
	session.mu.Unlock()
	if stream == nil || closed {
		return
	}
	if err := stream.AppendFrame(ctx, frame); err != nil {
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.audio.append_error", s.now().UnixMilli())
		return
	}
	id := session.identitySnapshot()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.audio.append", s.now().UnixMilli())
}

func (s *Server) startXiaozhiStreamingASRCommit(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn) bool {
	if session == nil || conn == nil || turn == nil {
		return false
	}
	session.mu.Lock()
	stream := session.streamingASRSession
	closed := session.streamingASRClosed
	started := session.streamingASRCommitStarted
	if stream != nil && !closed && !started {
		session.streamingASRCommitStarted = true
	}
	session.mu.Unlock()
	if stream == nil || closed {
		return false
	}
	if started {
		return true
	}
	go s.finishXiaozhiStreamingASRCommit(ctx, conn, session, turn, stream)
	return true
}

func (s *Server) finishXiaozhiStreamingASRCommit(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, stream providers.StreamingASRSession) {
	commitCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	id := session.identitySnapshot()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.commit", s.now().UnixMilli())
	if err := stream.Commit(commitCtx); err != nil {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.commit_error", s.now().UnixMilli())
		return
	}
	if !s.waitXiaozhiStreamingASRFinal(session, 200*time.Millisecond) {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.final_timeout", s.now().UnixMilli())
		return
	}
	s.maybeStartXiaozhiStreamingASRFinalAnswer(ctx, conn, session)
}

func (s *Server) maybeStartXiaozhiStreamingASRFinalAnswer(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession) bool {
	if s == nil || session == nil || conn == nil {
		return false
	}
	session.mu.Lock()
	turn := session.currentTurn
	listening := session.listening
	commitStarted := session.streamingASRCommitStarted
	hasFinal := session.streamingASRHasFinal && strings.TrimSpace(session.streamingASRFinalText) != ""
	session.mu.Unlock()
	if listening || !commitStarted || !hasFinal || turn == nil || session.shouldAbortXiaozhiTurn(turn) {
		return false
	}
	if !session.claimXiaozhiStreamingASRAnswer(turn) {
		return false
	}
	task := s.newXiaozhiTurnTask(session, turn)
	if strings.TrimSpace(task.streamingASRFinalText) == "" {
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.final_empty", s.now().UnixMilli())
		return false
	}
	s.startXiaozhiTurnTask(ctx, conn, session, task)
	return true
}

func (s *Server) waitXiaozhiStreamingASRFinal(session *xiaozhiSession, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		session.mu.Lock()
		hasFinal := session.streamingASRHasFinal && strings.TrimSpace(session.streamingASRFinalText) != ""
		closed := session.streamingASRClosed
		session.mu.Unlock()
		if hasFinal {
			return true
		}
		if closed || time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (s *Server) cancelXiaozhiStreamingASR(session *xiaozhiSession, reason string) {
	if session == nil {
		return
	}
	session.mu.Lock()
	stream := session.streamingASRSession
	if stream == nil || session.streamingASRClosed {
		session.mu.Unlock()
		return
	}
	session.streamingASRClosed = true
	session.mu.Unlock()
	stream.Cancel(context.Canceled)
	if strings.TrimSpace(reason) != "" {
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.cancelled", s.now().UnixMilli())
	}
}

func (s *Server) handleXiaozhiWS(w http.ResponseWriter, r *http.Request) {
	queryDeviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	headerDeviceID := strings.TrimSpace(r.Header.Get("Device-Id"))
	deviceID := firstNonEmpty(queryDeviceID, headerDeviceID)
	if deviceID != "" && !validA21DeviceID(deviceID) {
		http.Error(w, "invalid device_id", http.StatusBadRequest)
		return
	}
	protocolVersion, err := parseXiaozhiProtocolVersionHeader(r.Header.Get("Protocol-Version"))
	if err != nil {
		http.Error(w, "invalid Protocol-Version", http.StatusBadRequest)
		return
	}
	conn, err := websocket.Accept(w, r, a21WebSocketAcceptOptions())
	if err != nil {
		return
	}
	s.metrics.wsConnections.WithLabelValues("xiaozhi").Inc()
	defer s.metrics.wsConnections.WithLabelValues("xiaozhi").Dec()
	defer conn.Close(websocket.StatusNormalClosure, "a21 xiaozhi closed")

	ctx := context.Background()
	session := &xiaozhiSession{
		deviceID:              deviceID,
		binaryProtocolVersion: protocolVersion,
	}
	defer func() {
		s.cancelXiaozhiStreamingASR(session, "socket_closed")
		s.cancelXiaozhiOpusIngressQueue(session, "socket_closed")
		id := session.identitySnapshot()
		s.unregisterXiaozhiSocket(id.deviceID, conn)
	}()
	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		switch messageType {
		case websocket.MessageText:
			if !s.handleXiaozhiText(ctx, conn, session, data) {
				return
			}
		case websocket.MessageBinary:
			if !s.handleXiaozhiBinary(ctx, conn, session, data) {
				return
			}
		}
	}
}

func parseXiaozhiProtocolVersionHeader(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 1, nil
	}
	version, err := strconv.Atoi(raw)
	if err != nil || !xiaozhitransport.SupportedBinaryProtocolVersion(version) {
		return 0, fmt.Errorf("unsupported xiaozhi protocol version")
	}
	return version, nil
}

func (s *Server) handleXiaozhiText(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, data []byte) bool {
	if xiaozhiTextMessageType(data) == "device" {
		return s.handleXiaozhiDeviceExtension(ctx, conn, session, data)
	}
	frame, err := xiaozhitransport.ParseTextFrame(data, xiaozhitransport.DirectionDeviceToServer, session.identity())
	if err != nil {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, xiaozhiErrorCode(err), xiaozhiErrorDetail(err)))
		return true
	}
	session.adoptFrame(frame)
	if frame.Control == nil {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_message_type", "unsupported xiaozhi message"))
		return true
	}
	switch frame.Control.Type {
	case xiaozhitransport.MessageTypeHello:
		if err := session.configureXiaozhiAudio(frame.Control.Hello.AudioParams); err != nil {
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_audio_params", err.Error()))
			return true
		}
		session.mu.Lock()
		session.helloReceived = true
		session.listening = false
		session.listenStartedAtMS = 0
		session.suppressedListenActive = false
		session.ttsStopSent = false
		session.wakePrerollFrames = nil
		session.wakePrerollPayloadBytes = nil
		session.wakePrerollHasSpeech = false
		session.binaryProtocolVersion = frame.Control.Hello.AudioParams.BinaryProtocolVersion
		session.features = frame.Control.Hello.Features
		id := session.identityLocked()
		features := session.features
		session.mu.Unlock()
		s.registerXiaozhiSocket(id.deviceID, conn, &session.writeMu, session, features)
		s.recordXiaozhiDeviceSeen(frame)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.hello.received", s.now().UnixMilli())
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiHelloReply(session))
	case xiaozhitransport.MessageTypeListen:
		if !session.helloReceivedSnapshot() {
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "hello_required", "hello is required before listen"))
			return true
		}
		rawListenMode := frame.Control.Listen.Mode
		features := session.featuresSnapshot()
		mode := s.xiaozhiListenMode(rawListenMode, features)
		switch frame.Control.Listen.State {
		case "start":
			nowMS := s.now().UnixMilli()
			if suppressed, reason := session.xiaozhiInputSuppression(nowMS); suppressed {
				session.setXiaozhiListening(false, 0)
				session.mu.Lock()
				session.suppressedListenActive = true
				session.mu.Unlock()
				session.resetXiaozhiOpusIngress()
				session.resetXiaozhiWakePreroll()
				id := session.identitySnapshot()
				s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.start.input_suppressed", nowMS)
				s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.start.suppressed_"+reason, nowMS)
				s.writeXiaozhiListenReply(ctx, conn, session, "start", "ignored", "")
				return true
			}
			bargeTask, shouldStopPlayback := session.prepareXiaozhiListenStartBargeIn("barge_in", nowMS, xiaozhiPlaybackInterruptWindowMS)
			if shouldStopPlayback {
				s.recordXiaozhiListenBargeInMarkers(session, bargeTask.turn != nil)
				s.writeXiaozhiTTSStopForce(ctx, conn, session, bargeTask.turn, bargeTask, "barge_in")
			}
			turn := session.startXiaozhiTurn(ctx, mode)
			session.setXiaozhiListening(true, nowMS)
			session.resetXiaozhiOpusIngress()
			s.startXiaozhiOpusIngressQueue(ctx, session)
			s.startXiaozhiStreamingASR(ctx, conn, session, mode)
			s.attachXiaozhiWakePreroll(ctx, session)
			session.resetXiaozhiTTSStop()
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.turn.start", s.now().UnixMilli())
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.start", s.now().UnixMilli())
			s.writeXiaozhiOfficialStackChanState(ctx, session, "listening", "listen_start", true)
			if s.xiaozhiStockProfessionalRouteSelected(rawListenMode, features) {
				s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.professional_route.stock_override", s.now().UnixMilli())
			}
			s.writeXiaozhiListenReply(ctx, conn, session, "start", "accepted", xiaozhiTurnID(turn))
		case "detect":
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.detect", s.now().UnixMilli())
			s.writeXiaozhiListenReply(ctx, conn, session, "detect", "accepted", session.currentXiaozhiTurnID())
		case "stop":
			if !session.xiaozhiListening() {
				if session.clearSuppressedXiaozhiListen() {
					nowMS := s.now().UnixMilli()
					session.suppressXiaozhiInputUntil(nowMS+xiaozhiSuppressedListenDrainMS, "after_suppressed_listen")
					id := session.identitySnapshot()
					s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.stop.suppressed_session_ended", nowMS)
					s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.stop.suppressed_session_drain_armed", nowMS)
				}
				id := session.identitySnapshot()
				s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.stop.ignored", s.now().UnixMilli())
				s.writeXiaozhiListenReply(ctx, conn, session, "stop", "ignored", "")
				return true
			}
			session.setXiaozhiListening(false, 0)
			if mode == protocol.ModeProfessional {
				session.setCurrentXiaozhiTurnMode(mode)
			}
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.stop", s.now().UnixMilli())
			s.startXiaozhiListenStopAfterIngressDrain(ctx, conn, session, session.currentXiaozhiTurn())
		}
	case xiaozhitransport.MessageTypeAbort:
		if !session.helloReceivedSnapshot() {
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "hello_required", "hello is required before abort"))
			return true
		}
		session.setXiaozhiListening(false, 0)
		abortReason := frame.Control.Abort.Reason
		s.cancelXiaozhiStreamingASR(session, "abort")
		abortTask, shouldStopPlayback := session.prepareXiaozhiListenStartBargeIn(abortReason, s.now().UnixMilli(), xiaozhiPlaybackInterruptWindowMS)
		s.recordXiaozhiAbortMarkers(session, abortReason, shouldStopPlayback)
		if strings.TrimSpace(abortReason) == "" {
			session.suppressXiaozhiInputUntil(s.now().UnixMilli()+xiaozhiTouchBargeInInputCooldownMS, "after_barge")
		}
		if shouldStopPlayback {
			s.writeXiaozhiTTSStopForce(ctx, conn, session, abortTask.turn, abortTask, "abort")
		}
	case xiaozhitransport.MessageTypeMCP:
		if !session.helloReceivedSnapshot() {
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "hello_required", "hello is required before mcp"))
			return true
		}
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.mcp.response.received", s.now().UnixMilli())
		s.recordXiaozhiDeviceActivity(session, "xiaozhi.mcp.response.received", map[string]string{
			"xiaozhi_mcp_response": "received_redacted",
		})
	}
	return true
}

func (s *Server) writeXiaozhiListenReply(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, state string, status string, turnID string) {
	if !session.shouldSendXiaozhiListenReply() {
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen."+state+".reply_suppressed_stock_physical", s.now().UnixMilli())
		return
	}
	_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiBaseReply(session, "listen", state, status, turnID))
}

func (session *xiaozhiSession) shouldSendXiaozhiListenReply() bool {
	if session == nil {
		return false
	}
	features := session.featuresSnapshot()
	id := session.identitySnapshot()
	if xiaozhiClientProfile(features) == "debug" {
		return true
	}
	return !hardwareMACDeviceID(id.deviceID)
}

func (s *Server) handleXiaozhiDeviceExtension(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, data []byte) bool {
	if !session.helloReceivedSnapshot() {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "hello_required", "hello is required before device events"))
		return true
	}
	features := session.featuresSnapshot()
	productPlaybackEvents := s.xiaozhiProductPlaybackEventsAllowed(session)
	productKeepaliveEvents := s.xiaozhiProductKeepaliveEventsAllowed(session)
	productTouchEvents := s.xiaozhiProductTouchEventsAllowed(session)
	if !features.DeviceEvents && !productPlaybackEvents && !productKeepaliveEvents && !productTouchEvents {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "device_events_disabled", "device events require debug profile negotiation or product allowance"))
		return true
	}
	event, err := xiaozhitransport.ParseDeviceExtensionEvent(data)
	if err != nil {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, xiaozhiErrorCode(err), xiaozhiErrorDetail(err)))
		return true
	}
	if !features.DeviceEvents {
		switch event.Kind {
		case xiaozhitransport.DeviceEventKindPlayback:
			if !productPlaybackEvents {
				_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "product keepalive allowance does not accept playback acknowledgements"))
				return true
			}
		case xiaozhitransport.DeviceEventKindHeartbeat:
			if !productKeepaliveEvents {
				_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "product playback allowance does not accept keepalive heartbeats"))
				return true
			}
		case xiaozhitransport.DeviceEventKindTouch:
			if !productTouchEvents {
				_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "product allowance does not accept touch events"))
				return true
			}
		default:
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "product allowance only accepts playback acknowledgements, keepalive heartbeats, and touch events"))
			return true
		}
	}
	switch event.Kind {
	case xiaozhitransport.DeviceEventKindPlayback:
		switch event.Value {
		case "start":
			s.recordXiaozhiPlaybackStart(session, event.StreamID)
			return true
		case "stop_done":
			s.recordXiaozhiPlaybackStopDone(session, event.StreamID)
			return true
		default:
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "unsupported xiaozhi device event"))
			return true
		}
	case xiaozhitransport.DeviceEventKindHeartbeat:
		s.recordXiaozhiHeartbeat(session)
		return true
	case xiaozhitransport.DeviceEventKindTouch:
		if !s.recordXiaozhiTouchEvent(session, event) {
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "unsupported xiaozhi touch event"))
			return true
		}
		if xiaozhiTouchEventIsBargeIn(event) {
			nowMS := s.now().UnixMilli()
			bargeTask, shouldStopPlayback := session.prepareXiaozhiListenStartBargeIn("touch_barge_in", nowMS, xiaozhiPlaybackInterruptWindowMS)
			if shouldStopPlayback {
				s.recordXiaozhiTouchBargeInMarkers(session, bargeTask.turn != nil)
				s.writeXiaozhiTTSStopForce(ctx, conn, session, bargeTask.turn, bargeTask, "touch_barge_in")
			}
		}
		s.maybeSendXiaozhiTouchReaction(ctx, session, event)
		return true
	default:
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "unsupported xiaozhi device event"))
		return true
	}
}

func xiaozhiTextMessageType(data []byte) string {
	var common struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &common); err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(common.Type))
}

func (s *Server) handleXiaozhiBinary(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, data []byte) bool {
	if !session.helloReceivedSnapshot() {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "hello_required", "hello is required before binary audio"))
		return true
	}
	frame, err := xiaozhitransport.ParseBinaryFrameVersion(data, xiaozhitransport.DirectionDeviceToServer, session.identity(), session.binaryProtocolVersionSnapshot())
	if err != nil {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, xiaozhiErrorCode(err), xiaozhiErrorDetail(err)))
		return true
	}
	session.adoptFrame(frame)
	session.mu.Lock()
	listening := session.listening
	suppressedListenActive := session.suppressedListenActive
	id := session.identityLocked()
	session.mu.Unlock()
	if !listening {
		if suppressedListenActive {
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.ignored_suppressed_listen", s.now().UnixMilli())
			return true
		}
		if session.shouldBufferXiaozhiWakePreroll(s.now().UnixMilli()) {
			return s.bufferXiaozhiWakePreroll(session, frame)
		}
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.ignored_not_listening", s.now().UnixMilli())
		return true
	}
	session.mu.Lock()
	session.opusFrameCount++
	session.opusByteCount += frame.Opus.PayloadBytes
	seq := uint64(session.opusFrameCount)
	id = session.identityLocked()
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.received", s.now().UnixMilli())
	turn := session.currentXiaozhiTurn()
	if s.enqueueXiaozhiOpusIngressFrame(ctx, conn, session, frame, turn, seq) {
		return true
	}
	s.processXiaozhiOpusIngressFrame(ctx, conn, session, frame, seq)
	return true
}

func (s *Server) startXiaozhiOpusIngressQueue(ctx context.Context, session *xiaozhiSession) {
	if session == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	queueCtx, cancel := context.WithCancel(ctx)
	queue := make(chan xiaozhiOpusIngressFrame, maxXiaozhiOpusIngressQueueFrames)
	session.mu.Lock()
	if session.opusIngressCancel != nil {
		session.opusIngressCancel()
	}
	session.opusIngressCtx = queueCtx
	session.opusIngressCancel = cancel
	session.opusIngressQueue = queue
	session.mu.Unlock()
	go s.runXiaozhiOpusIngressQueue(queueCtx, session, queue)
}

func (s *Server) cancelXiaozhiOpusIngressQueue(session *xiaozhiSession, reason string) {
	if session == nil {
		return
	}
	session.mu.Lock()
	cancel := session.opusIngressCancel
	session.opusIngressCancel = nil
	session.opusIngressCtx = nil
	session.opusIngressQueue = nil
	session.mu.Unlock()
	if cancel != nil {
		cancel()
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_ingress.queue_cancelled."+safeGatewayFallbackToken(reason, "unknown"), s.now().UnixMilli())
	}
}

func (s *Server) enqueueXiaozhiOpusIngressFrame(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, frame xiaozhitransport.Frame, turn *xiaozhiTurn, seq uint64) bool {
	queue := session.xiaozhiOpusIngressQueue()
	if queue == nil {
		return false
	}
	item := xiaozhiOpusIngressFrame{
		ctx:   firstNonNilContext(xiaozhiTurnContext(turn), ctx),
		conn:  conn,
		frame: frame,
		turn:  turn,
		seq:   seq,
	}
	select {
	case queue <- item:
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_ingress.queued", s.now().UnixMilli())
	default:
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_ingress.queue_dropped", s.now().UnixMilli())
	}
	return true
}

func (s *Server) runXiaozhiOpusIngressQueue(ctx context.Context, session *xiaozhiSession, queue <-chan xiaozhiOpusIngressFrame) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-queue:
			s.processXiaozhiOpusIngressFrame(item.ctx, item.conn, session, item.frame, item.seq)
		}
	}
}

func (s *Server) startXiaozhiListenStopAfterIngressDrain(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn) {
	if session == nil || turn == nil {
		return
	}
	go func() {
		waitCtx, cancel := context.WithTimeout(firstNonNilContext(turn.ctx, ctx), 300*time.Millisecond)
		defer cancel()
		if !s.waitXiaozhiOpusIngressDrained(waitCtx, session) {
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_ingress.drain_timeout", s.now().UnixMilli())
		}
		if session.shouldAbortXiaozhiTurn(turn) {
			return
		}
		if s.startXiaozhiStreamingASRCommit(ctx, conn, session, turn) {
			return
		}
		task := s.newXiaozhiTurnTask(session, turn)
		s.startXiaozhiTurnTask(ctx, conn, session, task)
	}()
}

func (s *Server) waitXiaozhiOpusIngressDrained(ctx context.Context, session *xiaozhiSession) bool {
	for {
		session.mu.Lock()
		received := session.opusFrameCount
		processed := session.opusIngressProcessedFrameCount
		session.mu.Unlock()
		if processed >= received {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func xiaozhiTurnContext(turn *xiaozhiTurn) context.Context {
	if turn == nil {
		return nil
	}
	return turn.ctx
}

func firstNonNilContext(contexts ...context.Context) context.Context {
	for _, ctx := range contexts {
		if ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

func (s *Server) processXiaozhiOpusIngressFrame(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, frame xiaozhitransport.Frame, seq uint64) {
	defer func() {
		session.mu.Lock()
		session.opusIngressProcessedFrameCount++
		session.mu.Unlock()
	}()
	if ctx != nil && ctx.Err() != nil {
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_ingress.stale_frame_suppressed", s.now().UnixMilli())
		return
	}
	session.mu.Lock()
	if session.opusCodec == nil {
		session.opusDecodeErrorCount++
		id := session.identityLocked()
		session.mu.Unlock()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.decode_error", s.now().UnixMilli())
		return
	}
	pcm, err := session.opusCodec.DecodePCM16(frame.Opus.Payload)
	if err != nil {
		session.opusDecodeErrorCount++
		id := session.identityLocked()
		session.mu.Unlock()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.decode_error", s.now().UnixMilli())
		return
	}
	session.opusDecodedFrameCount++
	session.opusDecodedSampleCount += len(pcm)
	id := session.identityLocked()
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.decoded", s.now().UnixMilli())
	s.recordXiaozhiDeviceActivity(session, "xiaozhi.opus_frame.decoded", map[string]string{
		"microphone": "available_xiaozhi_opus_ingress",
	})
	s.observeXiaozhiDecodedIngress(ctx, conn, session, pcm, seq)
}

func (s *Server) bufferXiaozhiWakePreroll(session *xiaozhiSession, frame xiaozhitransport.Frame) bool {
	if session == nil || frame.Opus == nil {
		return true
	}
	id := session.identitySnapshot()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.wake_preroll.opus_frame.received", s.now().UnixMilli())
	session.mu.Lock()
	if session.opusCodec == nil {
		session.mu.Unlock()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.wake_preroll.opus_frame.decode_error", s.now().UnixMilli())
		return true
	}
	sampleRateHz := session.opusSampleRateHz
	channels := session.opusChannels
	pcm, err := session.opusCodec.DecodePCM16(frame.Opus.Payload)
	session.mu.Unlock()
	if err != nil {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.wake_preroll.opus_frame.decode_error", s.now().UnixMilli())
		return true
	}
	if len(pcm) == 0 || sampleRateHz <= 0 || channels <= 0 {
		return true
	}
	durationMS := len(pcm) * 1000 / sampleRateHz / channels
	chunk := protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: sampleRateHz,
		Channels:     channels,
		DurationMS:   durationMS,
		DataBase64:   pcm16Base64(pcm),
	}
	result := s.audioIngress.Push(audio.Frame{
		DeviceID:     id.deviceID,
		TraceID:      id.traceID,
		SessionID:    id.sessionID,
		Seq:          0,
		SampleRateHz: chunk.SampleRateHz,
		Channels:     chunk.Channels,
		DurationMS:   chunk.DurationMS,
		DataBase64:   chunk.DataBase64,
	})
	pipelineFrame := providers.VoicePipelinePCMFrame{
		Codec:        string(chunk.Codec),
		SampleRateHz: chunk.SampleRateHz,
		Channels:     chunk.Channels,
		DurationMS:   chunk.DurationMS,
		ByteCount:    len(pcm) * 2,
		RMS:          result.RMS,
		PCM16LE:      pcm16Bytes(pcm),
	}
	session.mu.Lock()
	seq := uint64(len(session.wakePrerollFrames) + 1)
	pipelineFrame.Seq = seq
	session.wakePrerollFrames = append(session.wakePrerollFrames, pipelineFrame)
	session.wakePrerollPayloadBytes = append(session.wakePrerollPayloadBytes, frame.Opus.PayloadBytes)
	if len(session.wakePrerollFrames) > maxXiaozhiWakePrerollFrames {
		session.wakePrerollFrames = session.wakePrerollFrames[len(session.wakePrerollFrames)-maxXiaozhiWakePrerollFrames:]
		session.wakePrerollPayloadBytes = session.wakePrerollPayloadBytes[len(session.wakePrerollPayloadBytes)-maxXiaozhiWakePrerollFrames:]
	}
	if result.SpeechDetected || result.SpeechActive || containsAudioIngressEvent(result.Events, audio.EventVADSpeechStart) {
		session.wakePrerollHasSpeech = true
	}
	session.mu.Unlock()
	s.recordAudioCaptureFrame(protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  id.deviceID,
		Kind:      protocol.KindAudioFrame,
		Seq:       seq,
		TraceID:   id.traceID,
		SessionID: id.sessionID,
		SentAtMS:  s.now().UnixMilli(),
	}, chunk, result, id.traceID, id.sessionID)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.wake_preroll.opus_frame.buffered", s.now().UnixMilli())
	return true
}

func (s *Server) attachXiaozhiWakePreroll(ctx context.Context, session *xiaozhiSession) {
	if session == nil {
		return
	}
	session.mu.Lock()
	if len(session.wakePrerollFrames) == 0 {
		session.mu.Unlock()
		return
	}
	id := session.identityLocked()
	frames := append([]providers.VoicePipelinePCMFrame(nil), session.wakePrerollFrames...)
	payloadBytes := append([]int(nil), session.wakePrerollPayloadBytes...)
	for i := range frames {
		frames[i].PCM16LE = append([]byte(nil), session.wakePrerollFrames[i].PCM16LE...)
	}
	hasSpeech := session.wakePrerollHasSpeech
	session.wakePrerollFrames = nil
	session.wakePrerollPayloadBytes = nil
	session.wakePrerollHasSpeech = false
	session.voicePipelineFrames = append(session.voicePipelineFrames, frames...)
	for i, frame := range frames {
		session.opusFrameCount++
		if i < len(payloadBytes) {
			session.opusByteCount += payloadBytes[i]
		}
		session.opusDecodedFrameCount++
		session.opusDecodedSampleCount += frame.ByteCount / 2
	}
	if hasSpeech {
		session.voicePipelineHasSpeech = true
	}
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.wake_preroll.attached", s.now().UnixMilli())
	for _, frame := range frames {
		s.appendXiaozhiStreamingASRFrame(ctx, session, frame)
	}
}

func (s *Server) observeXiaozhiDecodedIngress(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, pcm []int16, seq uint64) {
	session.mu.Lock()
	id := session.identityLocked()
	sampleRateHz := session.opusSampleRateHz
	channels := session.opusChannels
	session.mu.Unlock()
	if len(pcm) == 0 || sampleRateHz <= 0 || channels <= 0 {
		return
	}
	durationMS := len(pcm) * 1000 / sampleRateHz / channels
	chunk := protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: sampleRateHz,
		Channels:     channels,
		DurationMS:   durationMS,
		DataBase64:   pcm16Base64(pcm),
	}
	frame := protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  id.deviceID,
		Kind:      protocol.KindAudioFrame,
		Seq:       seq,
		TraceID:   id.traceID,
		SessionID: id.sessionID,
		SentAtMS:  s.now().UnixMilli(),
	}
	result := s.audioIngress.Push(audio.Frame{
		DeviceID:     id.deviceID,
		TraceID:      id.traceID,
		SessionID:    id.sessionID,
		Seq:          frame.Seq,
		SampleRateHz: chunk.SampleRateHz,
		Channels:     chunk.Channels,
		DurationMS:   chunk.DurationMS,
		DataBase64:   chunk.DataBase64,
	})
	pipelineFrame := providers.VoicePipelinePCMFrame{
		Seq:          frame.Seq,
		Codec:        string(chunk.Codec),
		SampleRateHz: chunk.SampleRateHz,
		Channels:     chunk.Channels,
		DurationMS:   chunk.DurationMS,
		ByteCount:    len(pcm) * 2,
		RMS:          result.RMS,
		PCM16LE:      pcm16Bytes(pcm),
	}
	session.mu.Lock()
	session.voicePipelineFrames = append(session.voicePipelineFrames, pipelineFrame)
	if result.SpeechDetected || result.SpeechActive || containsAudioIngressEvent(result.Events, audio.EventVADSpeechStart) {
		session.voicePipelineHasSpeech = true
	}
	session.mu.Unlock()
	s.appendXiaozhiStreamingASRFrame(ctx, session, pipelineFrame)
	s.recordAudioCaptureFrame(frame, chunk, result, id.traceID, id.sessionID)
	s.metrics.audioIngressFramesTotal.Inc()
	if result.DroppedFrameDelta > 0 {
		s.metrics.audioIngressDroppedTotal.Add(float64(result.DroppedFrameDelta))
	}
	s.metrics.audioIngressBufferDepth.Set(float64(result.BufferedFrames))
	s.metrics.audioIngressRMS.Set(result.RMS)
	s.metrics.vadDetectorDecisions.WithLabelValues(vadDetectorLabel(result.VADDetector), vadDecisionLabel(result.SpeechDetected)).Inc()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "audio.ingress.buffered", s.now().UnixMilli())
	for _, event := range result.Events {
		switch event {
		case audio.EventVADSpeechStart:
			s.metrics.vadSpeechStartTotal.Inc()
		case audio.EventVADSpeechEnd:
			s.metrics.vadSpeechEndTotal.Inc()
		}
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, string(event), s.now().UnixMilli())
	}
	s.maybeAutoStopXiaozhiTurnOnIngress(ctx, conn, session, result.Events)
}

func pcm16Base64(pcm []int16) string {
	data := pcm16Bytes(pcm)
	if len(data) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func pcm16Bytes(pcm []int16) []byte {
	if len(pcm) == 0 {
		return nil
	}
	data := make([]byte, len(pcm)*2)
	for i, sample := range pcm {
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	return data
}

func (s *Server) recordXiaozhiDeviceSeen(frame xiaozhitransport.Frame) {
	session := &xiaozhiSession{
		deviceID:  frame.DeviceID,
		traceID:   frame.TraceID,
		sessionID: frame.SessionID,
	}
	capabilities := map[string]string(nil)
	if frame.Control != nil && frame.Control.Hello != nil {
		session.features = frame.Control.Hello.Features
		capabilities = s.xiaozhiFeatureCapabilities(frame.Control.Hello.Features, session)
	}
	s.recordXiaozhiDeviceActivity(session, "xiaozhi.hello", capabilities)
}

func (s *Server) recordXiaozhiDeviceActivity(session *xiaozhiSession, event string, capabilities map[string]string) {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return
	}
	nowMS := s.now().UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[id.deviceID]
	if record.DeviceID == "" {
		record.DeviceID = id.deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	if len(capabilities) > 0 {
		record.Capabilities = mergeDeviceCapabilities(record.Capabilities, capabilities)
	}
	record.ConnectionStatus = "online"
	if cleanEvent := strings.TrimSpace(event); cleanEvent != "" {
		record.LastEvent = protocol.DeviceEventKind(cleanEvent)
	}
	record.LastTraceID = id.traceID
	record.LastSessionID = id.sessionID
	record.LastSeenMS = nowMS
	s.devices[id.deviceID] = record
}

func (s *Server) recordDeviceDisplayState(deviceID string, traceID string, sessionID string, source string, rawState string) bool {
	deviceID = strings.TrimSpace(deviceID)
	rawState = strings.TrimSpace(rawState)
	if deviceID == "" || rawState == "" {
		return false
	}
	source = normalizeDisplayStateSource(source)
	state := protocol.NormalizeOfficialDisplayState(rawState)
	nowMS := s.now().UnixMilli()
	s.mu.Lock()
	record := s.devices[deviceID]
	if record.DeviceID == "" {
		record.DeviceID = deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	changed := record.DisplayState != state ||
		record.DisplayStateSource != source ||
		record.DisplayStateTraceID != traceID ||
		record.DisplayStateSessionID != sessionID
	record.DisplayState = state
	record.DisplayStateSource = source
	record.DisplayStateTraceID = traceID
	record.DisplayStateSessionID = sessionID
	record.DisplayStateUpdatedAtMS = nowMS
	record.DisplayStateAccepted = false
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = nowMS
	s.devices[deviceID] = record
	s.mu.Unlock()
	if !changed {
		return false
	}
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.display_state.received", nowMS)
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.display_state.normalized", nowMS)
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.display_state.registry_updated", nowMS)
	return true
}

func normalizeDisplayStateSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "xiaozhi", "device_event", "official_stackchan":
		return strings.ToLower(strings.TrimSpace(source))
	default:
		return "unknown"
	}
}

func (s *Server) recordXiaozhiPlaybackStart(session *xiaozhiSession, streamID string) {
	s.recordXiaozhiPlaybackEvent(session, "device.playback.start", streamID)
}

func (s *Server) recordXiaozhiPlaybackStopDone(session *xiaozhiSession, streamID string) {
	s.recordXiaozhiPlaybackEvent(session, "device.playback.stop_done", streamID)
}

func (s *Server) recordXiaozhiHeartbeat(session *xiaozhiSession) {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "device.heartbeat", s.now().UnixMilli())
	s.recordXiaozhiDeviceActivity(session, "device.heartbeat", nil)
}

func (s *Server) recordXiaozhiTouchEvent(session *xiaozhiSession, event xiaozhitransport.DeviceExtensionEvent) bool {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return false
	}
	deviceEvent, source, ok := xiaozhiTouchEventKindAndSource(event)
	if !ok {
		return false
	}
	nowMS := s.now().UnixMilli()
	traceName := "device." + string(deviceEvent) + ".received"
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, traceName, nowMS)
	s.mu.Lock()
	record := s.devices[id.deviceID]
	if record.DeviceID == "" {
		record.DeviceID = id.deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.ConnectionStatus = "online"
	record.LastEvent = deviceEvent
	record.LastTouchEvent = deviceEvent
	record.LastTouchSource = source
	record.LastTouchTraceID = id.traceID
	record.LastTouchSessionID = id.sessionID
	record.LastTouchSeenMS = nowMS
	record.LastTraceID = id.traceID
	record.LastSessionID = id.sessionID
	record.LastSeenMS = nowMS
	s.devices[id.deviceID] = record
	s.mu.Unlock()
	return true
}

func xiaozhiTouchEventKindAndSource(event xiaozhitransport.DeviceExtensionEvent) (protocol.DeviceEventKind, protocol.TouchSource, bool) {
	var deviceEvent protocol.DeviceEventKind
	var source protocol.TouchSource
	switch event.Value {
	case "screen_tap":
		deviceEvent = protocol.DeviceEventTouchWakeOrListen
		source = protocol.TouchSourceScreen
	case "screen_barge_in":
		deviceEvent = protocol.DeviceEventTouchBargeIn
		source = protocol.TouchSourceScreen
	case "top_tap":
		deviceEvent = protocol.DeviceEventTouchTopTap
		source = protocol.TouchSourceTopSensor
	case "top_swipe_forward":
		deviceEvent = protocol.DeviceEventTouchTopSwipeForward
		source = protocol.TouchSourceTopSensor
	case "top_swipe_backward":
		deviceEvent = protocol.DeviceEventTouchTopSwipeBackward
		source = protocol.TouchSourceTopSensor
	case "top_barge_in":
		deviceEvent = protocol.DeviceEventTouchBargeIn
		source = protocol.TouchSourceTopSensor
	default:
		return "", "", false
	}
	if event.Source == string(protocol.TouchSourceScreen) {
		source = protocol.TouchSourceScreen
	} else if event.Source == string(protocol.TouchSourceTopSensor) {
		source = protocol.TouchSourceTopSensor
	}
	return deviceEvent, source, true
}

func xiaozhiTouchEventIsBargeIn(event xiaozhitransport.DeviceExtensionEvent) bool {
	switch event.Value {
	case "screen_barge_in", "top_barge_in":
		return true
	default:
		return false
	}
}

func (s *Server) maybeSendXiaozhiTouchReaction(ctx context.Context, session *xiaozhiSession, event xiaozhitransport.DeviceExtensionEvent) {
	if !s.xiaozhiProductTouchReactionsAllowed(session) {
		return
	}
	id := session.identitySnapshot()
	plans := xiaozhiTouchReactionPlans(session, event)
	if len(plans) == 0 {
		return
	}
	for _, req := range plans {
		delivery, status, message := s.sendXiaozhiMCPControl(ctx, req)
		if status != 0 {
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.touch_reaction.failed", s.now().UnixMilli())
			if message != "" {
				s.recordXiaozhiTouchReactionEcho(session, event, map[string]string{
					"last_touch_reaction_status": "failed_" + strconv.Itoa(status),
				})
			}
			continue
		}
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		reactionMarker := "xiaozhi.touch_reaction." + delivery.Marker + ".sent"
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, s.now().UnixMilli())
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, reactionMarker, s.now().UnixMilli())
		echo := xiaozhiMCPActivity(delivery.Marker, delivery.Args)
		echo["last_touch_reaction_status"] = "delivered"
		echo["last_touch_reaction_tool"] = delivery.Marker
		s.recordXiaozhiTouchReactionEcho(session, event, echo)
	}
}

func (s *Server) maybeSendXiaozhiStateReaction(ctx context.Context, session *xiaozhiSession, state string, reason string, force bool) {
	if !s.xiaozhiProductStateReactionsAllowed(session) {
		return
	}
	id := session.identitySnapshot()
	if xiaozhiStateReactionSuppressed(state, reason) {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.state_reaction.listen_start_suppressed", s.now().UnixMilli())
		s.recordXiaozhiStateReactionEcho(session, state, reason, map[string]string{
			"last_state_reaction_status": "suppressed_listen_start",
		})
		return
	}
	plans := xiaozhiStateReactionPlans(session, state)
	if len(plans) == 0 {
		return
	}
	if !session.claimXiaozhiStateReaction(state, force) {
		return
	}
	for _, req := range plans {
		delivery, status, _ := s.sendXiaozhiMCPControl(ctx, req)
		if status != 0 {
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.state_reaction.failed", s.now().UnixMilli())
			s.recordXiaozhiStateReactionEcho(session, state, reason, map[string]string{
				"last_state_reaction_status": "failed_" + strconv.Itoa(status),
			})
			continue
		}
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		reactionMarker := "xiaozhi.state_reaction." + delivery.Marker + ".sent"
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, s.now().UnixMilli())
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, reactionMarker, s.now().UnixMilli())
		echo := xiaozhiMCPActivity(delivery.Marker, delivery.Args)
		echo["last_state_reaction_status"] = "delivered"
		echo["last_state_reaction_tool"] = delivery.Marker
		s.recordXiaozhiStateReactionEcho(session, state, reason, echo)
	}
}

func xiaozhiStateReactionSuppressed(state string, reason string) bool {
	return strings.EqualFold(strings.TrimSpace(state), "listening") &&
		strings.EqualFold(strings.TrimSpace(reason), "listen_start")
}

func xiaozhiStateReactionPlans(session *xiaozhiSession, state string) []XiaozhiMCPControlRequest {
	if session == nil {
		return nil
	}
	id := session.identitySnapshot()
	base := XiaozhiMCPControlRequest{
		DeviceID:  id.deviceID,
		TraceID:   id.traceID,
		SessionID: id.sessionID,
	}
	led := func(red, green, blue int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetLEDColorToolName
		req.Red = xiaozhiReactionInt(red)
		req.Green = xiaozhiReactionInt(green)
		req.Blue = xiaozhiReactionInt(blue)
		return req
	}
	head := func(yaw *int, pitch int, speed int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetHeadAnglesToolName
		req.Yaw = yaw
		req.Pitch = xiaozhiReactionInt(pitch)
		req.Speed = xiaozhiReactionInt(speed)
		return req
	}
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "idle":
		return []XiaozhiMCPControlRequest{
			led(20, 20, 40),
			head(xiaozhiReactionInt(0), 24, 160),
		}
	case "listening":
		return []XiaozhiMCPControlRequest{
			led(0, 72, 168),
			head(xiaozhiReactionInt(0), 30, 180),
		}
	case "thinking":
		return []XiaozhiMCPControlRequest{
			led(90, 0, 168),
			head(xiaozhiReactionInt(0), 36, 160),
		}
	case "speaking":
		return []XiaozhiMCPControlRequest{
			led(0, 120, 90),
			head(xiaozhiReactionInt(8), 28, 180),
		}
	case "error", "fatal_error":
		return []XiaozhiMCPControlRequest{
			led(168, 24, 0),
			head(xiaozhiReactionInt(0), 20, 220),
		}
	default:
		return nil
	}
}

func xiaozhiTouchReactionPlans(session *xiaozhiSession, event xiaozhitransport.DeviceExtensionEvent) []XiaozhiMCPControlRequest {
	if session == nil {
		return nil
	}
	id := session.identitySnapshot()
	base := XiaozhiMCPControlRequest{
		DeviceID:  id.deviceID,
		TraceID:   id.traceID,
		SessionID: id.sessionID,
	}
	led := func(red, green, blue int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetLEDColorToolName
		req.Red = xiaozhiReactionInt(red)
		req.Green = xiaozhiReactionInt(green)
		req.Blue = xiaozhiReactionInt(blue)
		return req
	}
	head := func(yaw *int, pitch int, speed int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetHeadAnglesToolName
		req.Yaw = yaw
		req.Pitch = xiaozhiReactionInt(pitch)
		req.Speed = xiaozhiReactionInt(speed)
		return req
	}
	switch event.Value {
	case "screen_tap":
		return []XiaozhiMCPControlRequest{
			led(0, 72, 168),
			head(nil, 28, 180),
		}
	case "screen_barge_in":
		return []XiaozhiMCPControlRequest{
			led(168, 40, 0),
			head(xiaozhiReactionInt(0), 24, 240),
		}
	case "top_tap":
		return []XiaozhiMCPControlRequest{
			led(60, 0, 168),
			head(nil, 32, 180),
		}
	case "top_swipe_forward":
		return []XiaozhiMCPControlRequest{
			led(0, 120, 90),
			head(xiaozhiReactionInt(18), 24, 200),
		}
	case "top_swipe_backward":
		return []XiaozhiMCPControlRequest{
			led(120, 60, 0),
			head(xiaozhiReactionInt(-18), 24, 200),
		}
	case "top_barge_in":
		return []XiaozhiMCPControlRequest{
			led(168, 24, 0),
			head(xiaozhiReactionInt(0), 20, 240),
		}
	default:
		return nil
	}
}

func xiaozhiReactionInt(value int) *int {
	return &value
}

func (s *Server) recordXiaozhiTouchReactionEcho(session *xiaozhiSession, event xiaozhitransport.DeviceExtensionEvent, echo map[string]string) {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return
	}
	deviceEvent, source, ok := xiaozhiTouchEventKindAndSource(event)
	if !ok {
		return
	}
	nowMS := s.now().UnixMilli()
	reactionEcho := map[string]string{
		"last_touch_reaction_event":  safeGatewayFallbackToken(event.Value, "unknown"),
		"last_touch_reaction_source": string(source),
	}
	for key, value := range echo {
		reactionEcho[key] = value
	}
	s.mu.Lock()
	record := s.devices[id.deviceID]
	if record.DeviceID == "" {
		record.DeviceID = id.deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.ConnectionStatus = "online"
	record.LastEvent = deviceEvent
	record.LastTouchEvent = deviceEvent
	record.LastTouchSource = source
	record.LastTouchTraceID = id.traceID
	record.LastTouchSessionID = id.sessionID
	record.LastTouchSeenMS = nowMS
	record.LastTraceID = id.traceID
	record.LastSessionID = id.sessionID
	record.LastSeenMS = nowMS
	record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, reactionEcho)
	s.devices[id.deviceID] = record
	s.mu.Unlock()
}

func (s *Server) recordXiaozhiStateReactionEcho(session *xiaozhiSession, state string, reason string, echo map[string]string) {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return
	}
	nowMS := s.now().UnixMilli()
	reactionEcho := map[string]string{
		"last_state_reaction_state":  safeGatewayFallbackToken(state, "unknown"),
		"last_state_reaction_reason": safeGatewayFallbackToken(reason, "unknown"),
	}
	for key, value := range echo {
		reactionEcho[key] = value
	}
	s.mu.Lock()
	record := s.devices[id.deviceID]
	if record.DeviceID == "" {
		record.DeviceID = id.deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.ConnectionStatus = "online"
	record.LastTraceID = id.traceID
	record.LastSessionID = id.sessionID
	record.LastSeenMS = nowMS
	record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, reactionEcho)
	s.devices[id.deviceID] = record
	s.mu.Unlock()
}

func (s *Server) recordXiaozhiPlaybackEvent(session *xiaozhiSession, event string, streamID string) {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return
	}
	nowMS := s.now().UnixMilli()
	if event == "device.playback.stop_done" {
		session.mu.Lock()
		session.lastPlaybackStopDoneAtMS = nowMS
		session.mu.Unlock()
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, event, nowMS)
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[id.deviceID]
	if record.DeviceID == "" {
		record.DeviceID = id.deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.ConnectionStatus = "online"
	record.LastEvent = protocol.DeviceEventKind(event)
	record.LastTraceID = id.traceID
	record.LastSessionID = id.sessionID
	record.LastSeenMS = nowMS
	if streamID != "" {
		record.PlaybackStream = streamID
	}
	s.devices[id.deviceID] = record
}

func (s *Server) xiaozhiFeatureCapabilities(features xiaozhitransport.HelloFeatures, session *xiaozhiSession) map[string]string {
	capabilities := map[string]string{
		"xiaozhi_profile":   xiaozhiClientProfile(features),
		"xiaozhi_transport": "websocket",
		"xiaozhi_audio":     "opus_16000hz_mono_60ms",
	}
	if features.MCP {
		capabilities["xiaozhi_feature_mcp"] = "true"
		if s.xiaozhiProductStateReactionsAllowed(session) {
			capabilities["xiaozhi_product_state_reactions"] = "true"
		}
	}
	if features.AEC {
		capabilities["xiaozhi_feature_aec"] = "true"
	}
	if features.DeviceEvents {
		capabilities["xiaozhi_feature_device_events"] = "true"
	}
	if features.PlaybackEvents {
		capabilities["xiaozhi_feature_playback_events"] = "true"
		if s.xiaozhiProductPlaybackEventsAllowed(session) {
			capabilities["xiaozhi_product_playback_events"] = "true"
		}
	}
	if features.KeepaliveEvents {
		capabilities["xiaozhi_feature_keepalive_events"] = "true"
		if s.xiaozhiProductKeepaliveEventsAllowed(session) {
			capabilities["xiaozhi_product_keepalive_events"] = "true"
		}
	}
	if features.TouchEvents {
		capabilities["xiaozhi_feature_touch_events"] = "true"
		if s.xiaozhiProductTouchEventsAllowed(session) {
			capabilities["xiaozhi_product_touch_events"] = "true"
		}
		if s.xiaozhiProductTouchReactionsAllowed(session) {
			capabilities["xiaozhi_product_touch_reactions"] = "true"
		}
	}
	if features.DebugMetrics {
		capabilities["xiaozhi_feature_debug_metrics"] = "true"
	}
	if xiaozhiClientProfile(features) == "debug" {
		capabilities["xiaozhi_debug_extension_isolated"] = "true"
	}
	return capabilities
}

func (s *Server) xiaozhiProductPlaybackEventsAllowed(session *xiaozhiSession) bool {
	features := session.featuresSnapshot()
	id := session.identitySnapshot()
	return s != nil &&
		s.xiaozhiProductPlaybackEvents &&
		features.PlaybackEvents &&
		!features.DeviceEvents &&
		!features.DebugMetrics &&
		hardwareMACDeviceID(id.deviceID)
}

func (s *Server) xiaozhiProductKeepaliveEventsAllowed(session *xiaozhiSession) bool {
	features := session.featuresSnapshot()
	id := session.identitySnapshot()
	return s != nil &&
		s.xiaozhiProductPlaybackEvents &&
		features.KeepaliveEvents &&
		!features.DeviceEvents &&
		!features.DebugMetrics &&
		hardwareMACDeviceID(id.deviceID)
}

func (s *Server) xiaozhiProductTouchEventsAllowed(session *xiaozhiSession) bool {
	features := session.featuresSnapshot()
	id := session.identitySnapshot()
	return s != nil &&
		s.xiaozhiProductTouchEvents &&
		features.TouchEvents &&
		!features.DeviceEvents &&
		!features.DebugMetrics &&
		hardwareMACDeviceID(id.deviceID)
}

func (s *Server) xiaozhiProductTouchReactionsAllowed(session *xiaozhiSession) bool {
	features := session.featuresSnapshot()
	return s != nil &&
		s.xiaozhiProductTouchReactions &&
		s.xiaozhiProductTouchEventsAllowed(session) &&
		features.MCP
}

func (s *Server) xiaozhiProductStateReactionsAllowed(session *xiaozhiSession) bool {
	features := session.featuresSnapshot()
	id := session.identitySnapshot()
	return s != nil &&
		s.xiaozhiProductStateReactions &&
		features.MCP &&
		!features.DeviceEvents &&
		!features.DebugMetrics &&
		hardwareMACDeviceID(id.deviceID)
}

func xiaozhiClientProfile(features xiaozhitransport.HelloFeatures) string {
	if features.DeviceEvents || features.DebugMetrics {
		return "debug"
	}
	return "stock"
}

func (s *Server) xiaozhiListenMode(raw string, features xiaozhitransport.HelloFeatures) protocol.Mode {
	if strings.TrimSpace(strings.ToLower(raw)) == string(protocol.ModeProfessional) {
		return protocol.ModeProfessional
	}
	if s.xiaozhiStockProfessionalRouteSelected(raw, features) {
		return protocol.ModeProfessional
	}
	return protocol.ModeWorkmate
}

func (s *Server) xiaozhiStockProfessionalRouteSelected(raw string, features xiaozhitransport.HelloFeatures) bool {
	return s.xiaozhiStockProfessional &&
		s.selectedVoiceMode() == VoiceModeProfessional &&
		xiaozhiClientProfile(features) == "stock" &&
		strings.TrimSpace(strings.ToLower(raw)) != string(protocol.ModeProfessional)
}

func mergeDeviceCapabilities(existing map[string]string, additions map[string]string) map[string]string {
	if len(existing) == 0 {
		merged := make(map[string]string, len(additions))
		for key, value := range additions {
			merged[key] = value
		}
		return merged
	}
	merged := make(map[string]string, len(existing)+len(additions))
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range additions {
		merged[key] = value
	}
	return merged
}

func (s *Server) newXiaozhiTurnTask(session *xiaozhiSession, turn *xiaozhiTurn) xiaozhiTurnTask {
	session.mu.Lock()
	id := session.identityLocked()
	decodeStatus := xiaozhiOpusDecodeStatusFromCounts(session.opusDecodedFrameCount, session.opusDecodeErrorCount)
	decodedDurationMS := session.xiaozhiDecodedDurationMSLocked()
	streamingASRPartialText := session.streamingASRPartialText
	streamingASRFinalText := session.streamingASRFinalText
	streamingASRUsed := session.streamingASRHasFinal && strings.TrimSpace(session.streamingASRFinalText) != ""
	binaryProtocolVersion := session.binaryProtocolVersion
	opusSampleRateHz := session.opusSampleRateHz
	opusChannels := session.opusChannels
	opusFrameDurationMS := session.opusFrameDurationMS
	opusFrameCount := session.opusFrameCount
	opusByteCount := session.opusByteCount
	opusDecodedFrameCount := session.opusDecodedFrameCount
	opusDecodedSampleCount := session.opusDecodedSampleCount
	opusDecodeErrorCount := session.opusDecodeErrorCount
	voicePipelineFrames := copyXiaozhiVoicePipelineFrames(session.voicePipelineFrames)
	voicePipelineHasSpeech := session.voicePipelineHasSpeech
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi."+decodeStatus, s.now().UnixMilli())
	return xiaozhiTurnTask{
		turn:      turn,
		turnID:    xiaozhiTurnID(turn),
		traceID:   id.traceID,
		sessionID: id.sessionID,
		deviceID:  id.deviceID,
		mode:      xiaozhiTurnMode(turn),
		audioIngressBase: map[string]any{
			"codec":                "opus",
			"profile":              xiaozhiBinaryProfile(binaryProtocolVersion),
			"sample_rate_hz":       opusSampleRateHz,
			"channels":             opusChannels,
			"frame_duration_ms":    opusFrameDurationMS,
			"decode_status":        decodeStatus,
			"frame_count":          opusFrameCount,
			"byte_count":           opusByteCount,
			"decoded_frame_count":  opusDecodedFrameCount,
			"decoded_sample_count": opusDecodedSampleCount,
			"decoded_duration_ms":  decodedDurationMS,
			"decode_error_count":   opusDecodeErrorCount,
		},
		voicePipelineFrames:     voicePipelineFrames,
		voicePipelineHasSpeech:  voicePipelineHasSpeech,
		streamingASRPartialText: streamingASRPartialText,
		streamingASRFinalText:   streamingASRFinalText,
		streamingASRUsed:        streamingASRUsed,
	}
}

func (task xiaozhiTurnTask) audioIngressSummary(asrStatus string, ttsStatus string) map[string]any {
	summary := make(map[string]any, len(task.audioIngressBase)+2)
	for key, value := range task.audioIngressBase {
		summary[key] = value
	}
	summary["asr_status"] = asrStatus
	summary["tts_status"] = ttsStatus
	return summary
}

func (s *Server) startXiaozhiTurnTask(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask) {
	go func() {
		defer session.completeXiaozhiTurn(task.turn)
		s.writeXiaozhiTTS(ctx, conn, session, task)
	}()
}

func (s *Server) maybeAutoStopXiaozhiTurnOnIngress(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, events []audio.Event) {
	autoStopReason := ""
	if containsAudioIngressEvent(events, audio.EventVADSpeechEnd) {
		autoStopReason = "speech_end"
	} else if s.xiaozhiListenMaxDurationReached(session, s.now().UnixMilli()) {
		autoStopReason = "max_duration"
	}
	if autoStopReason == "" {
		return
	}
	session.mu.Lock()
	if !session.voicePipelineHasSpeech || !session.listening {
		session.mu.Unlock()
		return
	}
	stockPhysicalStop := autoStopReason == "speech_end" && hardwareMACDeviceID(session.deviceID) && xiaozhiClientProfile(session.features) == "stock"
	turn := session.currentTurn
	id := session.identityLocked()
	if stockPhysicalStop {
		session.mu.Unlock()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.gateway_vad_stop_deferred_stock_physical", s.now().UnixMilli())
		return
	}
	if turn == nil {
		session.mu.Unlock()
		return
	}
	session.listening = false
	session.listenStartedAtMS = 0
	session.mu.Unlock()
	if autoStopReason == "max_duration" {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.max_duration_auto_stop", s.now().UnixMilli())
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.auto_stop", s.now().UnixMilli())
	if s.startXiaozhiStreamingASRCommit(ctx, conn, session, turn) {
		return
	}
	task := s.newXiaozhiTurnTask(session, turn)
	s.startXiaozhiTurnTask(ctx, conn, session, task)
}

func (s *Server) xiaozhiListenMaxDurationReached(session *xiaozhiSession, nowMS int64) bool {
	if s == nil || session == nil || s.xiaozhiListenMaxDurationMS <= 0 || nowMS <= 0 {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if !session.listening || !session.voicePipelineHasSpeech || session.listenStartedAtMS <= 0 {
		return false
	}
	return nowMS-session.listenStartedAtMS >= s.xiaozhiListenMaxDurationMS
}

func (session *xiaozhiSession) xiaozhiUsesStockPhysicalStop() bool {
	if session == nil {
		return false
	}
	id := session.identitySnapshot()
	features := session.featuresSnapshot()
	return hardwareMACDeviceID(id.deviceID) && xiaozhiClientProfile(features) == "stock"
}

func (s *Server) recordXiaozhiAbortMarkers(session *xiaozhiSession, reason string, hadTurn bool) {
	now := s.now().UnixMilli()
	id := session.identitySnapshot()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.abort.received", now)
	if !hadTurn {
		return
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.start", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.end", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.turn.cancel", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "turn_cancelled", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "downlink_queue_cleared", now)
	if !xiaozhiAbortIsBargeIn(reason) {
		return
	}
	s.metrics.bargeInTotal.Inc()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "barge_in.detected", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "barge_in_detected", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "playback.stop", now)
}

func (s *Server) recordXiaozhiListenBargeInMarkers(session *xiaozhiSession, hadActiveTurn bool) {
	now := s.now().UnixMilli()
	id := session.identitySnapshot()
	s.metrics.bargeInTotal.Inc()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.barge_in", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "barge_in.detected", now)
	if hadActiveTurn {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.start", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.end", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.turn.cancel", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "turn_cancelled", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "downlink_queue_cleared", now)
	} else {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "playback.stop.recent_downlink", now)
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "playback.stop", now)
}

func (s *Server) recordXiaozhiTouchBargeInMarkers(session *xiaozhiSession, hadActiveTurn bool) {
	now := s.now().UnixMilli()
	id := session.identitySnapshot()
	s.metrics.bargeInTotal.Inc()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.touch.barge_in", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "barge_in.detected", now)
	if hadActiveTurn {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.start", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.end", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.turn.cancel", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "turn_cancelled", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "downlink_queue_cleared", now)
	} else {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "playback.stop.recent_downlink", now)
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "playback.stop", now)
}

func (s *Server) writeXiaozhiTTS(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask) {
	if task.mode == protocol.ModeProfessional {
		s.writeXiaozhiProfessionalTTS(ctx, conn, session, task)
		return
	}
	if professionalVoiceTriggerModeAllowed(task.mode) && professionalVoiceTrigger(task.streamingASRFinalText) {
		task.mode = protocol.ModeProfessional
		session.setCurrentXiaozhiTurnMode(protocol.ModeProfessional)
		now := s.now().UnixMilli()
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "professional.voice_trigger.detected", now)
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_route.voice_trigger", now)
		s.writeXiaozhiProfessionalTTS(ctx, conn, session, task)
		return
	}
	if s.writeXiaozhiVoicePipelineTTS(ctx, conn, session, task) {
		return
	}
	if session.shouldAbortXiaozhiTurn(task.turn) {
		return
	}
	s.writeXiaozhiPlaceholderTTS(ctx, conn, session, task)
}

func (s *Server) writeXiaozhiProfessionalTTS(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask) {
	turn := task.turn
	if session.shouldAbortXiaozhiTurn(turn) {
		return
	}
	s.writeXiaozhiOfficialStackChanState(ctx, session, "thinking", "professional_tts_start", false)
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":          "tts",
		"state":         "start",
		"mode":          string(protocol.ModeProfessional),
		"turn_id":       task.turnID,
		"trace_id":      task.traceID,
		"session_id":    task.sessionID,
		"device_id":     task.deviceID,
		"audio_ingress": task.audioIngressSummary("professional_boundary", "checking_then_evidence"),
	}); err != nil {
		return
	}
	receipt := xiaozhiProfessionalCheckingReceipt(task)
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":         "tts",
		"state":        "sentence_start",
		"phase":        "professional_checking",
		"mode":         string(protocol.ModeProfessional),
		"turn_id":      task.turnID,
		"trace_id":     task.traceID,
		"session_id":   task.sessionID,
		"device_id":    task.deviceID,
		"text":         receipt.Text,
		"professional": receipt,
	}); err != nil {
		session.cancelXiaozhiTurnContext(turn, "professional_checking_write_error")
		return
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "professional.checking_feedback.sent", s.now().UnixMilli())
	s.writeXiaozhiProfessionalAudioDownlink(ctx, conn, session, task, receipt.Text, "xiaozhi.professional_checking")
	if session.shouldAbortXiaozhiTurn(turn) {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
		return
	}
	utterance, err := s.xiaozhiProfessionalASRFinal(turn.ctx, task)
	if err != nil {
		if errors.Is(err, context.Canceled) || session.shouldAbortXiaozhiTurn(turn) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
			return
		}
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_asr_unavailable", s.now().UnixMilli())
		s.writeXiaozhiProfessionalFallback(ctx, conn, session, task, "professional_asr_unavailable")
		return
	}
	if strings.TrimSpace(utterance) == "" {
		if session.shouldAbortXiaozhiTurn(turn) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
			return
		}
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_asr_empty", s.now().UnixMilli())
		s.writeXiaozhiProfessionalFallback(ctx, conn, session, task, "professional_asr_empty")
		return
	}
	workspace := s.selectedProfessionalWorkspace()
	request := v21adapter.QueryRequest{
		TraceID:            task.traceID,
		SessionID:          task.sessionID,
		DeviceID:           task.deviceID,
		UserID:             workspace.UserID,
		WorkspaceID:        workspace.WorkspaceID,
		Mode:               "professional",
		QueryScope:         workspace.QueryScope,
		Utterance:          utterance,
		LatencyProfile:     "fast_first",
		AnswerStyle:        "voice_first_with_citations",
		MaxFirstResponseMS: v21adapter.ProfessionalMaxFirstResponseMS,
		PrivacyScope:       "professional_only",
	}
	utteranceBucket := v21UtteranceLengthBucket(utterance)
	readRecordID := s.startProfessionalReadRecord(request, utteranceBucket)
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "professional.workspace.ready", s.now().UnixMilli())
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "professional.query_scope."+workspace.QueryScope, s.now().UnixMilli())
	bindingDecision := s.professionalDeviceBindingDecision(task.deviceID, workspace.UserID, workspace.WorkspaceID, workspace.QueryScope)
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, bindingDecision.TraceMarker, s.now().UnixMilli())
	if !bindingDecision.Allowed {
		s.failProfessionalReadRecord(readRecordID, bindingDecision.FailureCode)
		s.writeXiaozhiProfessionalFallback(ctx, conn, session, task, "professional_device_binding_blocked")
		return
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "v21.query.utterance."+utteranceBucket, s.now().UnixMilli())
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "v21.query.start", s.now().UnixMilli())
	queryCtx, cancel := context.WithTimeout(turn.ctx, s.v21TTL)
	defer cancel()
	started := time.Now()
	response, err := s.v21.Query(queryCtx, request)
	s.metrics.v21QueryMS.Observe(float64(time.Since(started)) / float64(time.Millisecond))
	if err != nil {
		if errors.Is(err, context.Canceled) || session.shouldAbortXiaozhiTurn(turn) {
			s.failProfessionalReadRecord(readRecordID, "suppressed")
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
			return
		}
		s.failProfessionalReadRecord(readRecordID, professionalReadFailureCode(err, queryCtx))
		s.recordV21QueryFailure(task.traceID, task.sessionID, task.deviceID, err, queryCtx)
		s.writeXiaozhiProfessionalFallback(ctx, conn, session, task, "professional_v21_unavailable")
		return
	}
	if session.shouldAbortXiaozhiTurn(turn) {
		s.failProfessionalReadRecord(readRecordID, "suppressed")
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
		return
	}
	report, err := v21adapter.NewProfessionalBridgeEvidenceReport(response)
	if err != nil {
		s.failProfessionalReadRecord(readRecordID, "contract_invalid")
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "v21.query.error", s.now().UnixMilli())
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "v21.query.error.contract_invalid", s.now().UnixMilli())
		s.writeXiaozhiProfessionalFallback(ctx, conn, session, task, "professional_contract_invalid")
		return
	}
	s.completeProfessionalReadRecord(readRecordID, response)
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "v21.query.first_result", s.now().UnixMilli())
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":         "tts",
		"state":        "sentence_start",
		"phase":        "professional_result",
		"mode":         string(protocol.ModeProfessional),
		"turn_id":      task.turnID,
		"trace_id":     task.traceID,
		"session_id":   task.sessionID,
		"device_id":    task.deviceID,
		"text":         response.FastAnswer,
		"confidence":   response.Confidence,
		"professional": report,
	}); err != nil {
		session.cancelXiaozhiTurnContext(turn, "professional_result_write_error")
		return
	}
	s.writeXiaozhiProfessionalAudioDownlink(ctx, conn, session, task, response.FastAnswer, "xiaozhi.professional_result")
	if session.shouldAbortXiaozhiTurn(turn) {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
		return
	}
	s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "professional_result_completed")
}

func xiaozhiProfessionalCheckingReceipt(task xiaozhiTurnTask) v21adapter.ProfessionalBridgeReceipt {
	return v21adapter.ProfessionalBridgeReceipt{
		SchemaVersion:      "a21.v21_professional_bridge_receipt.v1",
		TraceID:            task.traceID,
		SessionID:          task.sessionID,
		Mode:               "professional",
		Status:             "checking",
		Text:               v21adapter.ProfessionalCheckingFeedbackText,
		MaxFirstResponseMS: v21adapter.ProfessionalMaxFirstResponseMS,
		EvidenceCompleted:  false,
	}
}

func (s *Server) xiaozhiProfessionalASRFinal(ctx context.Context, task xiaozhiTurnTask) (string, error) {
	if finalText := strings.TrimSpace(task.streamingASRFinalText); finalText != "" {
		return finalText, nil
	}
	if s.xiaozhiProfessionalASR == nil {
		return "", fmt.Errorf("professional ASR adapter unavailable")
	}
	events, err := s.xiaozhiProfessionalASR.Transcribe(ctx, providers.ASRAdapterRequest{
		Session: providers.VoiceSession{
			TraceID:   task.traceID,
			SessionID: task.sessionID,
			DeviceID:  task.deviceID,
		},
		Mode:   string(protocol.ModeProfessional),
		Frames: task.voicePipelineFrames,
	})
	if err != nil {
		return "", err
	}
	finalText := ""
	for event := range events {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Final {
			finalText = strings.TrimSpace(event.Text)
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return finalText, nil
}

func (s *Server) writeXiaozhiProfessionalFallback(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask, reason string) {
	turn := task.turn
	if session.shouldAbortXiaozhiTurn(turn) {
		return
	}
	text := "V21 现在没接上。我先把这个问题留住，等专业系统回来再查证据。"
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":       "tts",
		"state":      "sentence_start",
		"phase":      "professional_unavailable",
		"mode":       string(protocol.ModeProfessional),
		"turn_id":    task.turnID,
		"trace_id":   task.traceID,
		"session_id": task.sessionID,
		"device_id":  task.deviceID,
		"text":       text,
	}); err != nil {
		session.cancelXiaozhiTurnContext(turn, "professional_fallback_write_error")
		return
	}
	s.writeXiaozhiProfessionalAudioDownlink(ctx, conn, session, task, text, "xiaozhi.professional_fallback")
	if session.shouldAbortXiaozhiTurn(turn) {
		return
	}
	s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, reason)
}

func (s *Server) writeXiaozhiProfessionalAudioDownlink(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask, text string, marker string) bool {
	chunks, ok := s.writeXiaozhiTextAudioDownlink(ctx, conn, session, task, text, protocol.ModeProfessional, marker)
	return ok && chunks > 0
}

func (s *Server) writeXiaozhiTextAudioDownlink(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask, text string, mode protocol.Mode, marker string) (int, bool) {
	turn := task.turn
	if session.shouldAbortXiaozhiTurn(turn) {
		return 0, false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".tts_unavailable", s.now().UnixMilli())
		return 0, false
	}
	if mode == "" {
		mode = protocol.ModeWorkmate
	}
	tts := s.xiaozhiFastAckTTS
	if tts == nil {
		tts = providers.NewMockTTSAdapter("mock-fast-tts")
	}
	chunks, err := tts.Synthesize(turn.ctx, providers.TTSAdapterRequest{
		Session: providers.VoiceSession{
			TraceID:   task.traceID,
			SessionID: task.sessionID,
			DeviceID:  task.deviceID,
		},
		Mode: string(mode),
		Text: text,
	})
	if err != nil {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".tts_unavailable", s.now().UnixMilli())
		return 0, false
	}
	firstAudio := true
	wrote := false
	audioChunks := 0
	for chunk := range chunks {
		if session.shouldAbortXiaozhiTurn(turn) {
			return audioChunks, false
		}
		if firstAudio {
			firstAudio = false
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "tts.first_audio", s.now().UnixMilli())
		}
		ok, err := s.writeXiaozhiOpusDownlink(ctx, conn, session, turn, chunk)
		if err != nil {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink_error", s.now().UnixMilli())
			session.cancelXiaozhiTurnContext(turn, marker+"_downlink_error")
			return audioChunks, false
		}
		if !ok {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink_aborted", s.now().UnixMilli())
			return audioChunks, false
		}
		audioChunks++
		if !wrote {
			wrote = true
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "audio.downlink.first_frame", s.now().UnixMilli())
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink", s.now().UnixMilli())
		}
	}
	if !wrote {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".tts_unavailable", s.now().UnixMilli())
	}
	return audioChunks, wrote
}

func (s *Server) writeXiaozhiWAVAudioDownlink(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask, chunks []providers.VoiceAudioChunk, marker string) (int, bool) {
	turn := task.turn
	if session.shouldAbortXiaozhiTurn(turn) {
		return 0, false
	}
	if len(chunks) == 0 {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".tts_unavailable", s.now().UnixMilli())
		return 0, false
	}
	firstAudio := true
	wrote := false
	audioChunks := 0
	for _, chunk := range chunks {
		if session.shouldAbortXiaozhiTurn(turn) {
			return audioChunks, false
		}
		if firstAudio {
			firstAudio = false
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "tts.first_audio", s.now().UnixMilli())
		}
		ok, err := s.writeXiaozhiOpusDownlink(ctx, conn, session, turn, chunk)
		if err != nil {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink_error", s.now().UnixMilli())
			session.cancelXiaozhiTurnContext(turn, marker+"_downlink_error")
			return audioChunks, false
		}
		if !ok {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink_aborted", s.now().UnixMilli())
			return audioChunks, false
		}
		audioChunks++
		if !wrote {
			wrote = true
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "audio.downlink.first_frame", s.now().UnixMilli())
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink", s.now().UnixMilli())
		}
	}
	return audioChunks, wrote
}

func (s *Server) writeXiaozhiVoicePipelineTTS(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask) bool {
	turn := task.turn
	hasStreamingFinal := strings.TrimSpace(task.streamingASRFinalText) != ""
	if session.shouldAbortXiaozhiTurn(turn) || len(task.voicePipelineFrames) == 0 || (!task.voicePipelineHasSpeech && !hasStreamingFinal) {
		return false
	}
	startAtMS := s.now().UnixMilli()
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.start", startAtMS)
	newRunner := s.currentXiaozhiVoicePipelineRunnerFactory()
	if newRunner == nil {
		newRunner = defaultXiaozhiVoicePipelineRunner
	}
	runner := newRunner()
	asrTranscript := task.streamingASRFinalText
	asrTranscriptSource := ""
	if task.streamingASRPartialDriven && strings.TrimSpace(task.streamingASRPartialText) != "" {
		asrTranscript = task.streamingASRPartialText
		asrTranscriptSource = providers.VoicePipelineASRTranscriptSourcePartial
	} else if strings.TrimSpace(asrTranscript) != "" {
		asrTranscriptSource = providers.VoicePipelineASRTranscriptSourceFinal
	}
	if strings.TrimSpace(asrTranscript) != "" {
		if !s.writeXiaozhiSTT(ctx, conn, session, turn, task, asrTranscript, asrTranscriptSource) {
			return true
		}
	}
	s.writeXiaozhiOfficialStackChanState(ctx, session, "thinking", "voice_pipeline_start", false)
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":           "tts",
		"state":          "start",
		"turn_id":        task.turnID,
		"trace_id":       task.traceID,
		"session_id":     task.sessionID,
		"device_id":      task.deviceID,
		"audio_ingress":  task.audioIngressSummary("pipeline_running", s.xiaozhiVoicePipelineInitialTTSStatus()),
		"voice_pipeline": s.xiaozhiVoicePipelineInitialSummary(),
	}); err != nil {
		return true
	}
	request := providers.VoicePipelineRequest{
		Session: providers.VoiceSession{
			TraceID:   task.traceID,
			SessionID: task.sessionID,
			DeviceID:  task.deviceID,
		},
		Mode:                string(protocol.ModeWorkmate),
		Frames:              append([]providers.VoicePipelinePCMFrame(nil), task.voicePipelineFrames...),
		VoiceCloneProfile:   s.currentVoiceCloneProfile(),
		ASRTranscript:       asrTranscript,
		ASRTranscriptSource: asrTranscriptSource,
	}
	if s.selectedVoiceMode() == VoiceModeRoleplay {
		if prompt, err := s.roleplayPromptInput(RoleplayProfileSelectionRequest{}, roleplayPromptUserText(asrTranscript)); err == nil && strings.TrimSpace(prompt) != "" {
			request.TextPrompt = prompt
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "roleplay.prompt_input.used", s.now().UnixMilli())
		}
		if voiceCloneProfileUsesClone(request.VoiceCloneProfile) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "roleplay.voice_clone_profile.used", s.now().UnixMilli())
		}
	}
	markAnswerReady := s.startXiaozhiFastAckBackchannel(ctx, conn, session, turn, task)
	defer markAnswerReady()
	if streamer, ok := runner.(xiaozhiVoicePipelineStreamer); ok {
		return s.writeXiaozhiStreamingVoicePipelineAnswer(ctx, conn, session, turn, task, streamer, request, startAtMS, markAnswerReady)
	}
	result, err := runner.Run(turn.ctx, request)
	s.recordVoicePipelineFallback(task.traceID, task.sessionID, task.deviceID, result.Report)
	if err != nil || result.Status != providers.VoicePipelineStatusCompleted || len(result.AudioChunks) == 0 {
		if session.shouldAbortXiaozhiTurn(turn) || result.Status == providers.VoicePipelineStatusCancelled {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.cancelled", s.now().UnixMilli())
			return true
		}
		markAnswerReady()
		s.recordXiaozhiVoicePipelineProfileMarkers(task.traceID, task.sessionID, task.deviceID, result.Report)
		s.recordXiaozhiVoicePipelineFailureMarkers(task.traceID, task.sessionID, task.deviceID, result.Report, result, err)
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.unavailable", s.now().UnixMilli())
		s.writeXiaozhiLocalFallback(ctx, conn, session, turn, task, "voice_pipeline_unavailable_after_fast_ack")
		return true
	}
	s.recordXiaozhiVoicePipelineStageMarkers(task.traceID, task.sessionID, task.deviceID, startAtMS, result.Timing)
	markAnswerReady()
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":           "tts",
		"state":          "sentence_start",
		"phase":          "answer",
		"turn_id":        task.turnID,
		"trace_id":       task.traceID,
		"session_id":     task.sessionID,
		"device_id":      task.deviceID,
		"voice_pipeline": xiaozhiVoicePipelineSummary(result.Report),
		"text":           "",
	}); err != nil {
		return true
	}
	firstDownlink := true
	for _, chunk := range result.AudioChunks {
		ok, err := s.writeXiaozhiOpusDownlink(ctx, conn, session, turn, chunk)
		if err != nil {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.downlink_error", s.now().UnixMilli())
			s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_downlink_error")
			return true
		}
		if !ok {
			s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_aborted")
			return true
		}
		if firstDownlink {
			firstDownlink = false
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "audio.downlink.first_frame", s.now().UnixMilli())
		}
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.completed", s.now().UnixMilli())
	s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_answer_completed")
	return true
}

func (s *Server) writeXiaozhiSTT(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, text string, source string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	payload := map[string]any{
		"type":       "stt",
		"session_id": task.sessionID,
		"text":       text,
	}
	if err := session.writeXiaozhiJSON(ctx, conn, turn, payload); err != nil {
		session.cancelXiaozhiTurnContext(turn, "stt_write_error")
		return false
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.stt.sent", s.now().UnixMilli())
	if source != "" {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.stt."+safeGatewayFallbackToken(source, "streaming"), s.now().UnixMilli())
	}
	return true
}

func (s *Server) writeXiaozhiStreamingVoicePipelineAnswer(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, runner xiaozhiVoicePipelineStreamer, req providers.VoicePipelineRequest, startAtMS int64, markAnswerReady func()) bool {
	events, err := runner.RunStream(turn.ctx, req)
	if err != nil {
		markAnswerReady()
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.unavailable", s.now().UnixMilli())
		s.writeXiaozhiLocalFallback(ctx, conn, session, turn, task, "voice_pipeline_unavailable_after_fast_ack")
		return true
	}
	answerStarted := false
	lastSegmentSeq := 0
	firstDownlink := true
	stageMarkersRecorded := false
	profileMarkersRecorded := false
	var finalResult providers.VoicePipelineResult
	for event := range events {
		if session.shouldAbortXiaozhiTurn(turn) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.cancelled", s.now().UnixMilli())
			return true
		}
		switch event.Kind {
		case providers.VoicePipelineStreamAudioChunk:
			if !answerStarted || event.SegmentSeq != lastSegmentSeq {
				markAnswerReady()
				answerStarted = true
				lastSegmentSeq = event.SegmentSeq
				if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
					"type":           "tts",
					"state":          "sentence_start",
					"phase":          "answer",
					"turn_id":        task.turnID,
					"trace_id":       task.traceID,
					"session_id":     task.sessionID,
					"device_id":      task.deviceID,
					"voice_pipeline": s.xiaozhiVoicePipelineStreamingSummary(event.Report),
					"text":           "",
				}); err != nil {
					session.cancelXiaozhiTurnContext(turn, "voice_pipeline_answer_write_error")
					return true
				}
			}
			if !stageMarkersRecorded {
				stageMarkersRecorded = true
				s.recordXiaozhiVoicePipelineStageMarkers(task.traceID, task.sessionID, task.deviceID, startAtMS, event.Timing)
			}
			if !profileMarkersRecorded {
				profileMarkersRecorded = true
				s.recordXiaozhiVoicePipelineProfileMarkers(task.traceID, task.sessionID, task.deviceID, event.Report)
			}
			ok, err := s.writeXiaozhiOpusDownlink(ctx, conn, session, turn, event.AudioChunk)
			if err != nil {
				s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.downlink_error", s.now().UnixMilli())
				s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_downlink_error")
				session.cancelXiaozhiTurnContext(turn, "voice_pipeline_downlink_error")
				return true
			}
			if !ok {
				s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_aborted")
				session.cancelXiaozhiTurnContext(turn, "voice_pipeline_aborted")
				return true
			}
			if firstDownlink {
				firstDownlink = false
				s.recordTrace(task.traceID, task.sessionID, task.deviceID, "audio.downlink.first_frame", s.now().UnixMilli())
				s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.answer.downlink", s.now().UnixMilli())
			}
		case providers.VoicePipelineStreamDone:
			finalResult = event.Result
			err = event.Err
		}
	}
	s.recordVoicePipelineFallback(task.traceID, task.sessionID, task.deviceID, finalResult.Report)
	if err != nil || finalResult.Status != providers.VoicePipelineStatusCompleted || len(finalResult.AudioChunks) == 0 {
		if session.shouldAbortXiaozhiTurn(turn) || finalResult.Status == providers.VoicePipelineStatusCancelled {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.cancelled", s.now().UnixMilli())
			return true
		}
		if answerStarted {
			markAnswerReady()
			s.recordXiaozhiVoicePipelineProfileMarkers(task.traceID, task.sessionID, task.deviceID, finalResult.Report)
			s.recordXiaozhiVoicePipelineFailureMarkers(task.traceID, task.sessionID, task.deviceID, finalResult.Report, finalResult, err)
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.completed_degraded_after_audio", s.now().UnixMilli())
			s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_answer_completed_degraded")
			return true
		}
		markAnswerReady()
		s.recordXiaozhiVoicePipelineProfileMarkers(task.traceID, task.sessionID, task.deviceID, finalResult.Report)
		s.recordXiaozhiVoicePipelineFailureMarkers(task.traceID, task.sessionID, task.deviceID, finalResult.Report, finalResult, err)
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.unavailable", s.now().UnixMilli())
		s.writeXiaozhiLocalFallback(ctx, conn, session, turn, task, "voice_pipeline_unavailable_after_fast_ack")
		return true
	}
	if !stageMarkersRecorded {
		s.recordXiaozhiVoicePipelineStageMarkers(task.traceID, task.sessionID, task.deviceID, startAtMS, finalResult.Timing)
	}
	if !profileMarkersRecorded {
		s.recordXiaozhiVoicePipelineProfileMarkers(task.traceID, task.sessionID, task.deviceID, finalResult.Report)
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.completed", s.now().UnixMilli())
	s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_answer_completed")
	return true
}

func (s *Server) startXiaozhiFastAckBackchannel(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask) func() {
	if !s.xiaozhiFastAckEnabled {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.disabled", s.now().UnixMilli())
		return func() {}
	}
	delay := s.xiaozhiFastAckDelay
	if delay <= 0 {
		s.writeXiaozhiFastAckDownlink(ctx, conn, session, turn, task)
		return func() {}
	}
	answerReady := make(chan struct{})
	var answerReadyOnce sync.Once
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-turn.ctx.Done():
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.cancelled_before_delay", s.now().UnixMilli())
			return
		case <-answerReady:
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.skipped_answer_ready", s.now().UnixMilli())
			return
		case <-timer.C:
		}
		if session.shouldAbortXiaozhiTurn(turn) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.cancelled_after_delay", s.now().UnixMilli())
			return
		}
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.delay_elapsed", s.now().UnixMilli())
		if s.writeXiaozhiFastAckDownlink(ctx, conn, session, turn, task) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.delayed", s.now().UnixMilli())
		}
	}()
	return func() {
		answerReadyOnce.Do(func() {
			close(answerReady)
		})
	}
}

func (s *Server) writeXiaozhiFastAckDownlink(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask) bool {
	if session.shouldAbortXiaozhiTurn(turn) {
		return false
	}
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":       "tts",
		"state":      "sentence_start",
		"phase":      "fast_ack",
		"turn_id":    task.turnID,
		"trace_id":   task.traceID,
		"session_id": task.sessionID,
		"device_id":  task.deviceID,
		"text":       "",
	}); err != nil {
		return false
	}
	tts := s.xiaozhiFastAckTTS
	if tts == nil {
		tts = providers.NewMockTTSAdapter("mock-fast-tts")
	}
	chunks, err := tts.Synthesize(turn.ctx, providers.TTSAdapterRequest{
		Session: providers.VoiceSession{
			TraceID:   task.traceID,
			SessionID: task.sessionID,
			DeviceID:  task.deviceID,
		},
		Mode: string(protocol.ModeWorkmate),
		Text: "我在",
	})
	if err != nil {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.unavailable", s.now().UnixMilli())
		return false
	}
	for chunk := range chunks {
		ok, err := s.writeXiaozhiOpusDownlink(ctx, conn, session, turn, chunk)
		if err != nil {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.downlink_error", s.now().UnixMilli())
			s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "fast_ack_downlink_error")
			return false
		}
		if !ok {
			return false
		}
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "audio.downlink.first_frame", s.now().UnixMilli())
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.downlink", s.now().UnixMilli())
		return true
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.unavailable", s.now().UnixMilli())
	return false
}

func (s *Server) xiaozhiVoicePipelineInitialTTSStatus() string {
	if s.xiaozhiFastAckEnabled && s.xiaozhiFastAckDelay > 0 {
		return "delayed_fast_ack_then_answer"
	}
	if s.xiaozhiFastAckEnabled {
		return "fast_ack_then_answer"
	}
	return "answer_only"
}

func (s *Server) xiaozhiVoicePipelineInitialSummary() map[string]any {
	summary := s.xiaozhiVoicePipelineFastAckSummary()
	summary["fast_ack_enabled"] = s.xiaozhiFastAckEnabled
	if s.xiaozhiFastAckEnabled && s.xiaozhiFastAckDelay > 0 {
		summary["schema_version"] = "a21.voice_pipeline.delayed_fast_ack.v1"
		summary["stage"] = "answer_pending"
		summary["fast_ack_delay_ms"] = s.xiaozhiFastAckDelay.Milliseconds()
	} else if !s.xiaozhiFastAckEnabled {
		summary["schema_version"] = "a21.voice_pipeline.answer_only.v1"
		summary["stage"] = "answer_pending"
	}
	return summary
}

func (s *Server) xiaozhiVoicePipelineFastAckSummary() map[string]any {
	meta := s.currentXiaozhiVoicePipelineMeta()
	if isZeroGatewayVoicePipelineSelection(meta.Selection) {
		meta.Selection = providers.VoicePipelineSelectionFromEnv(nil)
	}
	executionMode := strings.TrimSpace(meta.ExecutionMode)
	if executionMode == "" {
		executionMode = "fixture"
	}
	return map[string]any{
		"schema_version": "a21.voice_pipeline.fast_ack.v1",
		"status":         "running",
		"stage":          "fast_ack",
		"execution_mode": executionMode,
		"selection": map[string]any{
			"asr_mode":        meta.Selection.ASRMode,
			"asr_profile":     meta.Selection.ASRProfile,
			"asr_profile_env": meta.Selection.ASRProfileEnv,
			"llm_profile":     meta.Selection.LLMProfile,
			"llm_profile_env": meta.Selection.LLMProfileEnv,
			"tts_mode":        meta.Selection.TTSMode,
			"tts_profile":     meta.Selection.TTSProfile,
			"tts_profile_env": meta.Selection.TTSProfileEnv,
		},
	}
}

func (s *Server) xiaozhiVoicePipelineStreamingSummary(reports ...providers.VoicePipelineReport) map[string]any {
	summary := s.xiaozhiVoicePipelineFastAckSummary()
	summary["schema_version"] = "a21.voice_pipeline.streaming_answer.v1"
	summary["stage"] = "answer"
	summary["streaming"] = true
	if len(reports) == 0 {
		return summary
	}
	report := reports[0]
	if report.Fallback != nil && report.Fallback.Activated {
		summary["fallback"] = map[string]any{
			"activated": true,
			"provider":  safeGatewayFallbackToken(report.Fallback.Provider, "fallback"),
			"reason":    safeGatewayFallbackToken(report.Fallback.Reason, "fallback_activated"),
		}
		summary["selection"] = map[string]any{
			"asr_mode":                 report.Selection.ASRMode,
			"asr_profile":              report.Selection.ASRProfile,
			"asr_profile_env":          report.Selection.ASRProfileEnv,
			"llm_profile":              report.Selection.LLMProfile,
			"llm_profile_env":          report.Selection.LLMProfileEnv,
			"llm_fallback_profile":     report.Selection.LLMFallbackProfile,
			"llm_fallback_profile_env": report.Selection.LLMFallbackProfileEnv,
			"tts_mode":                 report.Selection.TTSMode,
			"tts_profile":              report.Selection.TTSProfile,
			"tts_profile_env":          report.Selection.TTSProfileEnv,
		}
	}
	return summary
}

func (s *Server) recordXiaozhiVoicePipelineStageMarkers(traceID string, sessionID string, deviceID string, startAtMS int64, timing providers.VoicePipelineTiming) {
	if timing.ASRFirstPartialMS >= 0 {
		s.recordTrace(traceID, sessionID, deviceID, "asr.first_partial", startAtMS+timing.ASRFirstPartialMS)
	}
	if timing.ASRFinalMS >= 0 {
		s.recordTrace(traceID, sessionID, deviceID, "asr.final", startAtMS+timing.ASRFinalMS)
	}
	if timing.LLMFirstContentMS >= 0 {
		s.recordTrace(traceID, sessionID, deviceID, "provider.first_content", startAtMS+timing.LLMFirstContentMS)
	}
	if timing.TTSFirstAudioMS >= 0 {
		s.recordTrace(traceID, sessionID, deviceID, "tts.first_audio", startAtMS+timing.TTSFirstAudioMS)
	}
}

func (s *Server) recordXiaozhiVoicePipelineProfileMarkers(traceID string, sessionID string, deviceID string, report providers.VoicePipelineReport) {
	now := s.now().UnixMilli()
	for _, marker := range xiaozhiVoicePipelineProfileMarkers(report) {
		s.recordTrace(traceID, sessionID, deviceID, marker, now)
	}
}

func (s *Server) recordXiaozhiVoicePipelineFailureMarkers(traceID string, sessionID string, deviceID string, report providers.VoicePipelineReport, result providers.VoicePipelineResult, err error) {
	now := s.now().UnixMilli()
	markers := xiaozhiVoicePipelineFailureMarkers(report, result, err)
	for _, marker := range markers {
		s.recordTrace(traceID, sessionID, deviceID, marker, now)
	}
}

func xiaozhiVoicePipelineFailureMarkers(report providers.VoicePipelineReport, result providers.VoicePipelineResult, err error) []string {
	markers := make([]string, 0, 4)
	if report.Status != "" && report.Status != string(providers.VoicePipelineStatusCompleted) {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.status_"+safeGatewayFallbackToken(report.Status, "failed"))
	}
	if result.Status != "" && result.Status != providers.VoicePipelineStatusCompleted {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.result_"+safeGatewayFallbackToken(string(result.Status), "failed"))
	}
	for _, finding := range report.Findings {
		marker := xiaozhiVoicePipelineFindingMarker(finding)
		if marker != "" && !gatewayStringSliceHas(markers, marker) {
			markers = append(markers, marker)
		}
	}
	if result.Timing.LLMFirstContentMS < 0 {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.no_llm_first_content")
	}
	if result.Timing.TTSFirstAudioMS < 0 {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.no_tts_first_audio")
	}
	if len(result.AudioChunks) == 0 {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.empty_audio")
	}
	if err != nil {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.err")
	}
	return markers
}

func xiaozhiVoicePipelineFindingMarker(finding string) string {
	finding = strings.ToLower(strings.TrimSpace(finding))
	switch finding {
	case "asr adapter failed":
		return "xiaozhi.voice_pipeline.failed.asr_adapter_failed"
	case "text stream adapter failed":
		return "xiaozhi.voice_pipeline.failed.text_stream_adapter_failed"
	case "text stream adapter http status was not successful":
		return "xiaozhi.voice_pipeline.failed.text_stream_http_status"
	case "text stream adapter parse failed":
		return "xiaozhi.voice_pipeline.failed.text_stream_parse_failed"
	case "text stream adapter read failed":
		return "xiaozhi.voice_pipeline.failed.text_stream_read_failed"
	case "tts adapter failed":
		return "xiaozhi.voice_pipeline.failed.tts_adapter_failed"
	case "tts adapter no audio":
		return "xiaozhi.voice_pipeline.failed.tts_no_audio"
	case "tts adapter read failed":
		return "xiaozhi.voice_pipeline.failed.tts_read_failed"
	case "tts adapter provider error":
		return "xiaozhi.voice_pipeline.failed.tts_provider_error"
	case "tts adapter invalid audio delta":
		return "xiaozhi.voice_pipeline.failed.tts_invalid_audio_delta"
	case "tts adapter missing configuration":
		return "xiaozhi.voice_pipeline.failed.tts_missing_configuration"
	case "tts adapter connect failed":
		return "xiaozhi.voice_pipeline.failed.tts_connect_failed"
	case "tts adapter session update failed":
		return "xiaozhi.voice_pipeline.failed.tts_session_update_failed"
	case "tts adapter text append failed":
		return "xiaozhi.voice_pipeline.failed.tts_text_append_failed"
	case "tts adapter text commit failed":
		return "xiaozhi.voice_pipeline.failed.tts_text_commit_failed"
	case "tts adapter session finish failed":
		return "xiaozhi.voice_pipeline.failed.tts_session_finish_failed"
	case "provider_fallback_used":
		return "xiaozhi.voice_pipeline.provider_fallback_used"
	case "streaming_asr_partial_reused":
		return "xiaozhi.voice_pipeline.streaming_asr_partial_reused"
	case "streaming_asr_final_reused":
		return "xiaozhi.voice_pipeline.streaming_asr_final_reused"
	default:
		return ""
	}
}

func xiaozhiVoicePipelineProfileMarkers(report providers.VoicePipelineReport) []string {
	markers := make([]string, 0, 3)
	if xiaozhiVoicePipelineASRRealStreaming(report) {
		markers = append(markers, "xiaozhi.voice_pipeline.asr.real_streaming")
	} else if strings.Contains(report.Selection.ASRProfile, "mock") {
		markers = append(markers, "xiaozhi.voice_pipeline.asr.mock_blocked")
	} else {
		markers = append(markers, "xiaozhi.voice_pipeline.asr.batch_blocked")
	}
	if xiaozhiVoicePipelineLLMRealStreaming(report.Selection.LLMProfile) {
		markers = append(markers, "xiaozhi.voice_pipeline.llm.real_streaming")
	} else {
		markers = append(markers, "xiaozhi.voice_pipeline.llm.mock_blocked")
	}
	if xiaozhiVoicePipelineTTSRealStreaming(report.Selection.TTSProfile) {
		markers = append(markers, "xiaozhi.voice_pipeline.tts.real_streaming")
	} else if strings.Contains(report.Selection.TTSProfile, "mock") {
		markers = append(markers, "xiaozhi.voice_pipeline.tts.mock_blocked")
	} else {
		markers = append(markers, "xiaozhi.voice_pipeline.tts.file_boundary_blocked")
	}
	return markers
}

func xiaozhiVoicePipelineASRRealStreaming(report providers.VoicePipelineReport) bool {
	profile := strings.ToLower(strings.TrimSpace(report.Selection.ASRProfile))
	return (profile == "sherpa_onnx_streaming" ||
		profile == "local_sherpa_onnx_streaming" ||
		profile == "streaming_zipformer" ||
		profile == "dashscope_qwen_asr_realtime" ||
		profile == "doubao_asr_realtime" ||
		profile == "qwen_asr_realtime") &&
		(gatewayStringSliceHas(report.Findings, "streaming_asr_partial_reused") || gatewayStringSliceHas(report.Findings, "streaming_asr_final_reused"))
}

func xiaozhiVoicePipelineLLMRealStreaming(profile string) bool {
	profile = strings.ToLower(strings.TrimSpace(profile))
	return profile != "" && !strings.Contains(profile, "mock")
}

func xiaozhiVoicePipelineTTSRealStreaming(profile string) bool {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "doubao_tts_realtime", "doubao_realtime_tts", "dashscope_qwen_tts_realtime", "dashscope_tts_realtime", "qwen_tts_realtime", "qwen3_tts_realtime":
		return true
	default:
		return false
	}
}

func gatewayStringSliceHas(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (s *Server) xiaozhiAudioIngressSummary(session *xiaozhiSession, asrStatus string, ttsStatus string) map[string]any {
	session.mu.Lock()
	id := session.identityLocked()
	decodeStatus := xiaozhiOpusDecodeStatusFromCounts(session.opusDecodedFrameCount, session.opusDecodeErrorCount)
	decodedDurationMS := session.xiaozhiDecodedDurationMSLocked()
	binaryProtocolVersion := session.binaryProtocolVersion
	opusSampleRateHz := session.opusSampleRateHz
	opusChannels := session.opusChannels
	opusFrameDurationMS := session.opusFrameDurationMS
	opusFrameCount := session.opusFrameCount
	opusByteCount := session.opusByteCount
	opusDecodedFrameCount := session.opusDecodedFrameCount
	opusDecodedSampleCount := session.opusDecodedSampleCount
	opusDecodeErrorCount := session.opusDecodeErrorCount
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi."+decodeStatus, s.now().UnixMilli())
	return map[string]any{
		"codec":                "opus",
		"profile":              xiaozhiBinaryProfile(binaryProtocolVersion),
		"sample_rate_hz":       opusSampleRateHz,
		"channels":             opusChannels,
		"frame_duration_ms":    opusFrameDurationMS,
		"decode_status":        decodeStatus,
		"frame_count":          opusFrameCount,
		"byte_count":           opusByteCount,
		"decoded_frame_count":  opusDecodedFrameCount,
		"decoded_sample_count": opusDecodedSampleCount,
		"decoded_duration_ms":  decodedDurationMS,
		"decode_error_count":   opusDecodeErrorCount,
		"asr_status":           asrStatus,
		"tts_status":           ttsStatus,
	}
}

func xiaozhiVoicePipelineSummary(report providers.VoicePipelineReport) map[string]any {
	summary := map[string]any{
		"schema_version":    report.SchemaVersion,
		"status":            report.Status,
		"stage":             "answer",
		"execution_mode":    report.ExecutionMode,
		"audio_chunk_count": report.Output.AudioChunkCount,
		"llm_segment_count": report.Output.LLMSegmentCount,
		"streaming":         report.Output.StreamingAnswer,
		"selection": map[string]any{
			"asr_mode":                 report.Selection.ASRMode,
			"asr_profile":              report.Selection.ASRProfile,
			"asr_profile_env":          report.Selection.ASRProfileEnv,
			"llm_profile":              report.Selection.LLMProfile,
			"llm_profile_env":          report.Selection.LLMProfileEnv,
			"llm_fallback_profile":     report.Selection.LLMFallbackProfile,
			"llm_fallback_profile_env": report.Selection.LLMFallbackProfileEnv,
			"tts_mode":                 report.Selection.TTSMode,
			"tts_profile":              report.Selection.TTSProfile,
			"tts_profile_env":          report.Selection.TTSProfileEnv,
		},
		"timing": map[string]any{
			"asr_first_partial_ms":             report.Timing.ASRFirstPartialMS,
			"asr_final_ms":                     report.Timing.ASRFinalMS,
			"llm_first_content_ms":             report.Timing.LLMFirstContentMS,
			"tts_first_audio_ms":               report.Timing.TTSFirstAudioMS,
			"audio_downlink_first_frame_ms":    report.Timing.AudioDownlinkFirstMS,
			"speech_end_to_final_asr_ms":       report.Timing.SpeechEndToFinalASRMS,
			"speech_end_to_first_llm_token_ms": report.Timing.SpeechEndToFirstTokenMS,
		},
	}
	if report.Fallback != nil && report.Fallback.Activated {
		summary["fallback"] = map[string]any{
			"activated": true,
			"provider":  safeGatewayFallbackToken(report.Fallback.Provider, "fallback"),
			"reason":    safeGatewayFallbackToken(report.Fallback.Reason, "fallback_activated"),
		}
	}
	return summary
}

func (s *Server) recordVoicePipelineFallback(traceID string, sessionID string, deviceID string, report providers.VoicePipelineReport) {
	if report.Fallback == nil || !report.Fallback.Activated {
		return
	}
	s.metrics.providerFailoverTotal.Inc()
	s.metrics.fallbackTotal.Inc()
	atMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, deviceID, "fallback.used", atMS)
	s.recordTrace(traceID, sessionID, deviceID, "provider.failover", atMS)
}

func (s *Server) localFallbackPayload() protocol.ControlEventPayload {
	return protocol.ControlEventPayload{
		State: protocol.ExpressionLocalFallback,
		Mode:  protocol.ModeLocalFallback,
		Text:  localFallbackText,
		Final: true,
	}
}

func (s *Server) recordLocalFallback(traceID string, sessionID string, deviceID string, reason string) {
	s.metrics.fallbackTotal.Inc()
	atMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, deviceID, "fallback.used", atMS)
	s.recordTrace(traceID, sessionID, deviceID, "local_fallback.entered", atMS)
	if reason != "" {
		s.recordTrace(traceID, sessionID, deviceID, "local_fallback."+safeGatewayFallbackToken(reason, "unavailable"), atMS)
	}
	s.recordDeviceControl(deviceID, traceID, sessionID, s.localFallbackPayload(), atMS)
}

func (s *Server) writeXiaozhiLocalFallback(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, reason string) bool {
	s.recordLocalFallback(task.traceID, task.sessionID, task.deviceID, reason)
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.local_fallback.sent", s.now().UnixMilli())
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":       "tts",
		"state":      "sentence_start",
		"phase":      "local_fallback",
		"mode":       string(protocol.ModeLocalFallback),
		"turn_id":    task.turnID,
		"trace_id":   task.traceID,
		"session_id": task.sessionID,
		"device_id":  task.deviceID,
		"text":       localFallbackText,
	}); err != nil {
		return false
	}
	return s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "local_fallback")
}

func voicePipelineTextStreamProvider(report providers.VoicePipelineReport) string {
	if report.Fallback != nil && report.Fallback.Activated {
		if provider := safeGatewayFallbackToken(report.Fallback.Provider, "fallback"); provider != "" {
			return provider
		}
	}
	return firstNonEmpty(safeGatewayFallbackToken(report.Selection.LLMProfile, ""), "unknown")
}

func voicePipelineFallbackResponse(report providers.VoicePipelineReport) (bool, string, string) {
	if report.Fallback == nil || !report.Fallback.Activated {
		return false, "", ""
	}
	return true,
		safeGatewayFallbackToken(report.Fallback.Provider, "fallback"),
		safeGatewayFallbackToken(report.Fallback.Reason, "fallback_activated")
}

var gatewayFallbackTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,80}$`)

func safeGatewayFallbackToken(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "http") ||
		strings.Contains(lower, "bearer") ||
		strings.Contains(lower, "sk-") ||
		strings.Contains(lower, "x21") ||
		strings.Contains(lower, "v21") ||
		strings.Contains(value, "/") ||
		strings.Contains(value, "\\") ||
		!gatewayFallbackTokenPattern.MatchString(value) {
		return fallback
	}
	return value
}

func (s *Server) writeXiaozhiPlaceholderTTS(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask) {
	s.writeXiaozhiOfficialStackChanState(ctx, session, "thinking", "placeholder_tts_start", false)
	if err := session.writeXiaozhiJSON(ctx, conn, task.turn, map[string]any{
		"type":          "tts",
		"state":         "start",
		"turn_id":       task.turnID,
		"trace_id":      task.traceID,
		"session_id":    task.sessionID,
		"device_id":     task.deviceID,
		"audio_ingress": task.audioIngressSummary("not_connected", "placeholder_only"),
	}); err != nil {
		return
	}
	if err := session.writeXiaozhiJSON(ctx, conn, task.turn, map[string]any{
		"type":        "tts",
		"state":       "sentence_start",
		"turn_id":     task.turnID,
		"trace_id":    task.traceID,
		"session_id":  task.sessionID,
		"device_id":   task.deviceID,
		"placeholder": true,
		"text":        "",
	}); err != nil {
		return
	}
	s.writeXiaozhiTTSStop(ctx, conn, session, task.turn, task, "placeholder_no_asr_tts")
	s.suppressXiaozhiInputAfterNoSpeechPlaceholder(session, task)
}

func (s *Server) writeXiaozhiTTSStop(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, reason string) bool {
	return s.writeXiaozhiTTSStopWithOptions(ctx, conn, session, turn, task, reason, false)
}

func (s *Server) writeXiaozhiTTSStopForce(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, reason string) bool {
	return s.writeXiaozhiTTSStopWithOptions(ctx, conn, session, turn, task, reason, true)
}

func (s *Server) writeXiaozhiTTSStopWithOptions(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, reason string, force bool) bool {
	if conn == nil {
		return false
	}
	session.writeMu.Lock()
	if !force && turn != nil && session.shouldAbortXiaozhiTurn(turn) {
		session.writeMu.Unlock()
		return false
	}
	if !force && !session.claimXiaozhiTTSStop() {
		session.writeMu.Unlock()
		return false
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.tts.stop", s.now().UnixMilli())
	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "tts",
		"state":      "stop",
		"turn_id":    task.turnID,
		"trace_id":   task.traceID,
		"session_id": task.sessionID,
		"device_id":  task.deviceID,
		"reason":     reason,
	}); err != nil {
		session.writeMu.Unlock()
		return false
	}
	session.writeMu.Unlock()
	s.writeXiaozhiOfficialStackChanState(ctx, session, xiaozhiOfficialStateForTTSStop(reason), "tts_stop_"+safeGatewayFallbackToken(reason, "unknown"), true)
	if xiaozhiShouldSuppressInputAfterTTSStop(reason) {
		untilMS := s.now().UnixMilli() + xiaozhiPostTTSInputCooldownMS
		session.suppressXiaozhiInputUntil(untilMS, "post_tts_drain")
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.tts.stop.input_suppression_armed", s.now().UnixMilli())
	}
	return true
}

func xiaozhiShouldSuppressInputAfterTTSStop(reason string) bool {
	reason = strings.ToLower(strings.TrimSpace(reason))
	if reason == "" {
		return false
	}
	if strings.Contains(reason, "abort") || strings.Contains(reason, "barge") || strings.Contains(reason, "wake") {
		return false
	}
	return strings.Contains(reason, "completed") ||
		strings.Contains(reason, "local_fallback") ||
		strings.Contains(reason, "placeholder") ||
		strings.Contains(reason, "host_say_complete")
}

func (s *Server) writeXiaozhiOpusDownlink(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, chunk providers.VoiceAudioChunk) (bool, error) {
	if session == nil || session.shouldAbortXiaozhiTurn(turn) {
		s.recordXiaozhiStaleDownlinkSuppressed(session)
		return false, nil
	}
	if conn == nil {
		return false, fmt.Errorf("xiaozhi downlink websocket is nil")
	}
	if turn.pacer == nil {
		turn.pacer = audio.NewAudioRateController(audio.AudioRateControllerConfig{
			FrameDuration:   60 * time.Millisecond,
			PrebufferFrames: 1,
		})
	}
	pcm, err := xiaozhiDownlinkPCM16(chunk)
	if err != nil {
		return false, err
	}
	codec, err := turn.xiaozhiDownlinkCodec(chunk)
	if err != nil {
		return false, err
	}
	packet, err := codec.EncodePCM16(pcm)
	if err != nil {
		return false, err
	}
	s.writeXiaozhiOfficialStackChanState(ctx, session, "speaking", "opus_downlink", false)
	return turn.pacer.Send(ctx, packet, func(ctx context.Context, frame []byte) error {
		if session.shouldAbortXiaozhiTurn(turn) {
			s.recordXiaozhiStaleDownlinkSuppressed(session)
			return context.Canceled
		}
		if err := session.writeXiaozhiBinary(ctx, conn, turn, frame); err != nil {
			return err
		}
		nowMS := s.now().UnixMilli()
		session.markXiaozhiDownlink(turn, nowMS)
		s.recordTrace(session.traceID, session.sessionID, session.deviceID, "xiaozhi.tts.opus_frame.downlink", nowMS)
		s.recordXiaozhiDeviceActivity(session, "xiaozhi.tts.opus_frame.downlink", map[string]string{
			"speaker": "available_xiaozhi_opus_downlink",
		})
		return nil
	}, func() bool {
		return session.shouldAbortXiaozhiTurn(turn)
	})
}

func (s *Server) recordXiaozhiStaleDownlinkSuppressed(session *xiaozhiSession) {
	if session == nil {
		return
	}
	s.recordTrace(session.traceID, session.sessionID, session.deviceID, "xiaozhi.tts.stale_frame_suppressed", s.now().UnixMilli())
}

func (turn *xiaozhiTurn) xiaozhiDownlinkCodec(chunk providers.VoiceAudioChunk) (*opuscodec.Codec, error) {
	if turn == nil {
		return nil, fmt.Errorf("%w: nil xiaozhi turn", opuscodec.ErrUnsupportedConfig)
	}
	if turn.downlink != nil &&
		turn.downlink.sampleRateHz == chunk.SampleRateHz &&
		turn.downlink.channels == chunk.Channels &&
		turn.downlink.durationMS == chunk.DurationMS &&
		turn.downlink.codec != nil {
		return turn.downlink.codec, nil
	}
	codec, err := opuscodec.New(chunk.SampleRateHz, chunk.Channels, chunk.DurationMS)
	if err != nil {
		return nil, err
	}
	turn.downlink = &xiaozhiTurnDownlinkCodec{
		sampleRateHz: chunk.SampleRateHz,
		channels:     chunk.Channels,
		durationMS:   chunk.DurationMS,
		codec:        codec,
	}
	return codec, nil
}

const xiaozhiDownlinkPCM16HeadroomPeak = 29490
const xiaozhiDownlinkPCM16TargetPeak = 24576
const xiaozhiDownlinkPCM16MaxGainMilli = 3000
const xiaozhiDownlinkPCM16NoiseGatePeak = 512

func xiaozhiDownlinkPCM16(chunk providers.VoiceAudioChunk) ([]int16, error) {
	if chunk.Codec != string(protocol.AudioCodecPCMS16LE) {
		return nil, fmt.Errorf("xiaozhi downlink requires pcm_s16le provider audio")
	}
	if (chunk.SampleRateHz != 16000 && chunk.SampleRateHz != 24000 && chunk.SampleRateHz != 48000) || chunk.Channels != 1 || chunk.DurationMS != 60 {
		return nil, fmt.Errorf("xiaozhi downlink requires 16kHz, 24kHz, or 48kHz mono 60ms audio")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(chunk.DataBase64))
	if err != nil {
		return nil, fmt.Errorf("xiaozhi downlink audio must be valid base64")
	}
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("xiaozhi downlink pcm_s16le byte length must be even")
	}
	pcm := make([]int16, len(data)/2)
	for i := range pcm {
		pcm[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
	}
	applyXiaozhiDownlinkLeveling(pcm)
	return pcm, nil
}

func applyXiaozhiDownlinkLeveling(pcm []int16) {
	maxAbs := 0
	for _, sample := range pcm {
		abs := int(sample)
		if abs < 0 {
			abs = -abs
		}
		if abs > maxAbs {
			maxAbs = abs
		}
	}
	if maxAbs == 0 || maxAbs < xiaozhiDownlinkPCM16NoiseGatePeak {
		return
	}
	targetPeak := maxAbs
	if maxAbs > xiaozhiDownlinkPCM16HeadroomPeak {
		targetPeak = xiaozhiDownlinkPCM16HeadroomPeak
	} else if maxAbs < xiaozhiDownlinkPCM16TargetPeak {
		targetPeak = xiaozhiDownlinkPCM16TargetPeak
		maxBoostedPeak := maxAbs * xiaozhiDownlinkPCM16MaxGainMilli / 1000
		if maxBoostedPeak < targetPeak {
			targetPeak = maxBoostedPeak
		}
		if targetPeak > xiaozhiDownlinkPCM16HeadroomPeak {
			targetPeak = xiaozhiDownlinkPCM16HeadroomPeak
		}
	}
	if targetPeak == maxAbs {
		return
	}
	for i, sample := range pcm {
		pcm[i] = int16(int(sample) * targetPeak / maxAbs)
	}
}

func (s *Server) xiaozhiHelloReply(session *xiaozhiSession) map[string]any {
	audioParams := map[string]any{
		"format":         "opus",
		"sample_rate":    24000,
		"channels":       1,
		"frame_duration": 60,
	}
	reply := map[string]any{
		"type":         "hello",
		"version":      session.binaryProtocolVersion,
		"transport":    "websocket",
		"trace_id":     session.traceID,
		"session_id":   session.sessionID,
		"device_id":    session.deviceID,
		"audio":        audioParams,
		"audio_params": audioParams,
	}
	if session.features.DeviceEvents {
		reply["a21"] = map[string]any{
			"profile":       "debug",
			"device_events": true,
		}
	} else if s.xiaozhiProductPlaybackEventsAllowed(session) || s.xiaozhiProductKeepaliveEventsAllowed(session) || s.xiaozhiProductTouchEventsAllowed(session) || s.xiaozhiProductStateReactionsAllowed(session) {
		a21 := map[string]any{
			"profile": "product",
		}
		if s.xiaozhiProductPlaybackEventsAllowed(session) {
			a21["playback_events"] = true
		}
		if s.xiaozhiProductKeepaliveEventsAllowed(session) {
			a21["keepalive_events"] = true
		}
		if s.xiaozhiProductTouchEventsAllowed(session) {
			a21["touch_events"] = true
		}
		if s.xiaozhiProductTouchReactionsAllowed(session) {
			a21["touch_reactions"] = true
		}
		if s.xiaozhiProductStateReactionsAllowed(session) {
			a21["state_reactions"] = true
		}
		reply["a21"] = a21
	}
	return reply
}

func xiaozhiBinaryProfile(version int) string {
	if version <= 1 {
		return "xiaozhi_binary_v1_raw"
	}
	return fmt.Sprintf("xiaozhi_binary_v%d", version)
}

func (s *Server) xiaozhiBaseReply(session *xiaozhiSession, msgType string, state string, status string, turnID string) map[string]any {
	reply := map[string]any{
		"type":       msgType,
		"state":      state,
		"status":     status,
		"trace_id":   session.traceID,
		"session_id": session.sessionID,
		"device_id":  session.deviceID,
	}
	if turnID != "" {
		reply["turn_id"] = turnID
	}
	return reply
}

func (s *Server) xiaozhiError(session *xiaozhiSession, code string, detail string) map[string]any {
	traceID, sessionID := s.ids(session.traceID, session.sessionID)
	session.traceID = traceID
	session.sessionID = sessionID
	return map[string]any{
		"type":       "error",
		"code":       code,
		"detail":     detail,
		"trace_id":   session.traceID,
		"session_id": session.sessionID,
		"device_id":  session.deviceID,
	}
}

func xiaozhiErrorCode(err error) string {
	switch {
	case errors.Is(err, xiaozhitransport.ErrMalformedJSON):
		return "invalid_json"
	case errors.Is(err, xiaozhitransport.ErrMissingDeviceIdentity), errors.Is(err, xiaozhitransport.ErrLegacyIdentity):
		return "invalid_device_id"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedMessageType):
		return "unsupported_message_type"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedListenState):
		return "unsupported_listen_state"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedHelloVersion):
		return "unsupported_hello_version"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedTransport):
		return "unsupported_transport"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedAudioParams):
		return "unsupported_audio_params"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedBinaryProtocol):
		return "unsupported_binary_protocol_version"
	case errors.Is(err, xiaozhitransport.ErrMalformedBinaryFrame):
		return "malformed_binary_frame"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedBinaryFrameType):
		return "unsupported_binary_frame_type"
	case errors.Is(err, xiaozhitransport.ErrEmptyBinaryPayload):
		return "empty_binary_payload"
	case errors.Is(err, xiaozhitransport.ErrUnexpectedBinaryDirection):
		return "unexpected_binary_frame_direction"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedDeviceEventKind):
		return "unsupported_device_event_kind"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedDeviceEventValue):
		return "unsupported_device_event_value"
	default:
		return "invalid_xiaozhi_message"
	}
}

func xiaozhiErrorDetail(err error) string {
	if err == nil {
		return "invalid xiaozhi message"
	}
	return err.Error()
}

func (s *Server) handleOfficialStackChanWS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	deviceID := officialStackChanDeviceID(r)
	if !validA21DeviceID(deviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	conn, err := websocket.Accept(w, r, a21WebSocketAcceptOptions())
	if err != nil {
		return
	}
	writeMu := &sync.Mutex{}
	s.registerOfficialStackChanSocket(deviceID, conn, writeMu)
	s.recordOfficialStackChanConnected(deviceID)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer s.unregisterOfficialStackChanSocket(deviceID, conn)
	defer conn.Close(websocket.StatusNormalClosure, "official stackchan websocket closed")

	go s.writeOfficialStackChanHeartbeat(ctx, conn, writeMu)

	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if messageType == websocket.MessageBinary && len(data) > 0 && data[0] == stackchantransport.DataTypeHeartbeatPong {
			s.recordTrace("", "", deviceID, "stackchan.official_ws.heartbeat_pong", s.now().UnixMilli())
		}
	}
}

func officialStackChanDeviceID(r *http.Request) string {
	query := r.URL.Query()
	deviceID := firstNonEmpty(query.Get("device_id"), query.Get("deviceId"), query.Get("id"))
	if strings.TrimSpace(deviceID) == "" {
		return defaultOfficialStackChanDeviceID
	}
	return strings.TrimSpace(deviceID)
}

func (s *Server) writeOfficialStackChanHeartbeat(ctx context.Context, conn *websocket.Conn, writeMu *sync.Mutex) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	packet := stackchantransport.OfficialPacket{Type: stackchantransport.DataTypeHeartbeatPing}.Bytes()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			writeMu.Lock()
			err := conn.Write(ctx, websocket.MessageBinary, packet)
			writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (s *Server) handleControlWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, a21WebSocketAcceptOptions())
	if err != nil {
		return
	}
	s.metrics.wsConnections.WithLabelValues("control").Inc()
	defer s.metrics.wsConnections.WithLabelValues("control").Dec()
	defer conn.Close(websocket.StatusNormalClosure, "a21 control closed")

	ctx := context.Background()
	for {
		var event protocol.Envelope
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			return
		}
		events := s.controlEventsForDeviceEvent(event)
		for _, control := range events {
			if err := wsjson.Write(ctx, conn, control); err != nil {
				return
			}
		}
	}
}

func (s *Server) handleAudioWS(w http.ResponseWriter, r *http.Request) {
	queryDeviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if queryDeviceID != "" && !validA21DeviceID(queryDeviceID) {
		http.Error(w, "invalid device_id", http.StatusBadRequest)
		return
	}
	conn, err := websocket.Accept(w, r, a21WebSocketAcceptOptions())
	if err != nil {
		return
	}
	s.metrics.wsConnections.WithLabelValues("audio").Inc()
	defer s.metrics.wsConnections.WithLabelValues("audio").Dec()
	defer conn.Close(websocket.StatusNormalClosure, "a21 audio closed")
	realtimeAudioKeys := make(map[string]struct{})
	defer s.closeRealtimeAudioSessions(context.Background(), realtimeAudioKeys)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writeMu := &sync.Mutex{}
	registeredDeviceID := queryDeviceID
	if registeredDeviceID != "" {
		s.registerAudioSocket(registeredDeviceID, conn, writeMu)
	}
	defer func() {
		if registeredDeviceID != "" {
			s.unregisterAudioSocket(registeredDeviceID, conn)
		}
	}()
	for {
		var frame protocol.Envelope
		if err := wsjson.Read(ctx, conn, &frame); err != nil {
			return
		}
		if registeredDeviceID == "" && validA21DeviceID(frame.DeviceID) {
			registeredDeviceID = frame.DeviceID
			s.registerAudioSocket(registeredDeviceID, conn, writeMu)
		}
		if frame.Kind == protocol.KindDeviceEvent {
			events := s.controlEventsForDeviceEvent(frame)
			for _, event := range events {
				if err := writeAudioEnvelope(ctx, conn, writeMu, event); err != nil {
					return
				}
			}
			continue
		}
		if frame.Kind == protocol.KindAudioFrame {
			s.metrics.audioFrameTotal.Inc()
		}
		traceID, sessionID := s.ids(frame.TraceID, frame.SessionID)
		s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.frame.received", s.now().UnixMilli())
		ingress, validAudio := s.observeAudioIngress(frame, traceID, sessionID)
		if frame.Kind == protocol.KindAudioFrame && !validAudio {
			events := s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
				{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: "invalid audio frame", Final: true},
			})
			for _, event := range events {
				if err := writeAudioEnvelope(ctx, conn, writeMu, event); err != nil {
					return
				}
			}
			continue
		}
		if s.shouldBargeIn(frame, traceID, sessionID, ingress) {
			events := s.audioBargeInEvents(frame, traceID, sessionID)
			for _, event := range events {
				if err := writeAudioEnvelope(ctx, conn, writeMu, event); err != nil {
					return
				}
			}
			continue
		}
		if frame.Kind == protocol.KindAudioFrame && s.audioProbeOnly(frame.DeviceID, traceID, sessionID) {
			s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.probe.frame.accepted", s.now().UnixMilli())
			continue
		}
		if events, handled := s.realtimeAudioEvents(ctx, conn, writeMu, frame, traceID, sessionID, ingress, realtimeAudioKeys); handled {
			for _, event := range events {
				if err := writeAudioEnvelope(ctx, conn, writeMu, event); err != nil {
					return
				}
			}
			continue
		}
		mockPlaybackChunks, shouldEmitMockAudioPlayback := s.mockAudioPlaybackChunkCount(frame, traceID, sessionID, ingress)
		if !shouldEmitMockAudioPlayback {
			continue
		}
		if mockPlaybackChunks <= 0 {
			continue
		}
		streamID := s.mockAudioStreamID(frame, traceID, sessionID)
		events := s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionListening, Mode: protocol.ModeWorkmate, Text: "audio frame accepted"},
			{State: protocol.ExpressionSpeaking, Mode: protocol.ModeWorkmate, Text: "mock playback chunk", Final: true, StreamID: streamID},
		})
		for _, event := range events {
			if err := writeAudioEnvelope(ctx, conn, writeMu, event); err != nil {
				return
			}
		}
		for i := 0; i < mockPlaybackChunks; i++ {
			playback := s.mockAudioPlaybackChunk(frame, traceID, sessionID, streamID, uint64(len(events)+i+1))
			if err := writeAudioEnvelope(ctx, conn, writeMu, playback); err != nil {
				return
			}
		}
		s.setActiveStream(traceID, sessionID, frame.DeviceID, streamID)
	}
}

func (s *Server) mockAudioPlaybackChunkCount(frame protocol.Envelope, traceID string, sessionID string, ingress audio.IngressResult) (int, bool) {
	if frame.Kind != protocol.KindAudioFrame {
		return 1, true
	}
	if !physicalStackChanDeviceID(frame.DeviceID) {
		return 1, true
	}
	if chunks, armed := s.consumeMockPlaybackOnNextAudioFrame(frame.DeviceID, traceID, sessionID); armed {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.mock_physical.armed", s.now().UnixMilli())
		return chunks, true
	}
	if containsAudioIngressEvent(ingress.Events, audio.EventVADSpeechStart) {
		return 1, true
	}
	s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.mock_physical.suppressed", s.now().UnixMilli())
	return 0, false
}

func (s *Server) observeAudioIngress(frame protocol.Envelope, traceID string, sessionID string) (audio.IngressResult, bool) {
	if frame.Kind != protocol.KindAudioFrame {
		return audio.IngressResult{}, true
	}
	var chunk protocol.AudioChunk
	if err := json.Unmarshal(frame.Payload, &chunk); err != nil {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.ingress.invalid", s.now().UnixMilli())
		return audio.IngressResult{}, false
	}
	if err := protocol.ValidateAudioChunk(chunk); err != nil {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.ingress.invalid", s.now().UnixMilli())
		return audio.IngressResult{}, false
	}
	result := s.audioIngress.Push(audio.Frame{
		DeviceID:     frame.DeviceID,
		TraceID:      traceID,
		SessionID:    sessionID,
		Seq:          frame.Seq,
		SampleRateHz: chunk.SampleRateHz,
		Channels:     chunk.Channels,
		DurationMS:   chunk.DurationMS,
		DataBase64:   chunk.DataBase64,
	})
	s.recordAudioCaptureFrame(frame, chunk, result, traceID, sessionID)
	s.metrics.audioIngressFramesTotal.Inc()
	if result.DroppedFrameDelta > 0 {
		s.metrics.audioIngressDroppedTotal.Add(float64(result.DroppedFrameDelta))
	}
	s.metrics.audioIngressBufferDepth.Set(float64(result.BufferedFrames))
	s.metrics.audioIngressRMS.Set(result.RMS)
	s.metrics.vadDetectorDecisions.WithLabelValues(vadDetectorLabel(result.VADDetector), vadDecisionLabel(result.SpeechDetected)).Inc()
	s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.ingress.buffered", s.now().UnixMilli())
	for _, event := range result.Events {
		switch event {
		case audio.EventVADSpeechStart:
			s.metrics.vadSpeechStartTotal.Inc()
		case audio.EventVADSpeechEnd:
			s.metrics.vadSpeechEndTotal.Inc()
		}
		s.recordTrace(traceID, sessionID, frame.DeviceID, string(event), s.now().UnixMilli())
	}
	return result, true
}

func (s *Server) recordAudioCaptureFrame(frame protocol.Envelope, chunk protocol.AudioChunk, ingress audio.IngressResult, traceID string, sessionID string) {
	dataBytes := 0
	if data, err := base64.StdEncoding.DecodeString(chunk.DataBase64); err == nil {
		dataBytes = len(data)
	}
	capture := AudioCaptureFrame{
		DeviceID:            frame.DeviceID,
		TraceID:             traceID,
		SessionID:           sessionID,
		Seq:                 frame.Seq,
		SentAtMS:            frame.SentAtMS,
		ReceivedAtMS:        s.now().UnixMilli(),
		SampleRateHz:        chunk.SampleRateHz,
		Channels:            chunk.Channels,
		DurationMS:          chunk.DurationMS,
		CaptureStartedAtMS:  chunk.CaptureStartedAtMS,
		CaptureEndedAtMS:    chunk.CaptureEndedAtMS,
		DataBytes:           dataBytes,
		DataBase64:          chunk.DataBase64,
		RMS:                 ingress.RMS,
		VADDetector:         ingress.VADDetector,
		VADStatus:           ingress.VADStatus,
		VADFinding:          ingress.VADFinding,
		SpeechDetected:      ingress.SpeechDetected,
		SpeechActive:        ingress.SpeechActive,
		DroppedFrames:       ingress.DroppedFrames,
		DroppedFrameDelta:   ingress.DroppedFrameDelta,
		IngressBufferFrames: ingress.BufferedFrames,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audioCaptureFrames = append(s.audioCaptureFrames, capture)
	if len(s.audioCaptureFrames) > maxAudioCaptureFrames {
		s.audioCaptureFrames = s.audioCaptureFrames[len(s.audioCaptureFrames)-maxAudioCaptureFrames:]
	}
}

func (s *Server) recentAudioFrames(deviceID string, traceID string, sessionID string, limit int, includeAudio bool) []AudioCaptureFrame {
	s.mu.Lock()
	defer s.mu.Unlock()
	matches := make([]AudioCaptureFrame, 0, limit)
	for i := len(s.audioCaptureFrames) - 1; i >= 0 && len(matches) < limit; i-- {
		frame := s.audioCaptureFrames[i]
		if deviceID != "" && frame.DeviceID != deviceID {
			continue
		}
		if traceID != "" && frame.TraceID != traceID {
			continue
		}
		if sessionID != "" && frame.SessionID != sessionID {
			continue
		}
		if !includeAudio {
			frame.DataBase64 = ""
		}
		matches = append(matches, frame)
	}
	for left, right := 0, len(matches)-1; left < right; left, right = left+1, right-1 {
		matches[left], matches[right] = matches[right], matches[left]
	}
	return matches
}

func vadDetectorLabel(detector string) string {
	if detector == "" {
		return "unknown"
	}
	return detector
}

func vadDecisionLabel(speechDetected bool) string {
	if speechDetected {
		return "speech"
	}
	return "silence"
}

func (s *Server) realtimeAudioEvents(ctx context.Context, conn *websocket.Conn, writeMu *sync.Mutex, frame protocol.Envelope, traceID string, sessionID string, ingress audio.IngressResult, connectionSessionKeys map[string]struct{}) ([]protocol.Envelope, bool) {
	provider, ok := s.currentVoiceProvider().(providers.RealtimeVoiceProvider)
	if !ok || frame.Kind != protocol.KindAudioFrame {
		return nil, false
	}
	key := streamStateKey(traceID, sessionID, frame.DeviceID)
	if s.shouldSuppressUnarmedPhysicalRealtimeAudio(frame, traceID, sessionID, ingress) {
		return nil, true
	}
	if containsAudioIngressEvent(ingress.Events, audio.EventVADSpeechEnd) {
		session := s.realtimeAudioSession(traceID, sessionID, frame.DeviceID)
		if session == nil {
			return nil, true
		}
		connectionSessionKeys[key] = struct{}{}
		if err := session.CommitAndCreateResponse(ctx); err != nil {
			s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.audio.commit.error", s.now().UnixMilli())
			return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
				{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: err.Error(), Final: true},
			}), true
		}
		s.metrics.realtimeAudioCommitTotal.Inc()
		s.markRealtimeAudioCommit(traceID, sessionID, frame.DeviceID, s.now())
		s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.audio.commit", s.now().UnixMilli())
		return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionThinking, Mode: protocol.ModeWorkmate, Text: "我在想"},
		}), true
	}
	if !ingress.SpeechActive || !ingress.SpeechDetected {
		return nil, true
	}

	var chunk protocol.AudioChunk
	if err := json.Unmarshal(frame.Payload, &chunk); err != nil {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.audio.append.error", s.now().UnixMilli())
		return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: "invalid audio frame", Final: true},
		}), true
	}
	session := s.realtimeAudioSession(traceID, sessionID, frame.DeviceID)
	started := false
	if session == nil {
		created, err := provider.StartRealtimeSession(ctx, providers.VoiceSession{TraceID: traceID, SessionID: sessionID, DeviceID: frame.DeviceID})
		if err != nil {
			s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.realtime_session.error", s.now().UnixMilli())
			return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
				{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: err.Error(), Final: true},
			}), true
		}
		session = created
		s.setRealtimeAudioSession(traceID, sessionID, frame.DeviceID, session)
		connectionSessionKeys[key] = struct{}{}
		s.startRealtimeAudioDownlinkPump(ctx, conn, writeMu, frame.DeviceID, traceID, sessionID, session)
		started = true
		s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.realtime_session.start", s.now().UnixMilli())
	} else {
		connectionSessionKeys[key] = struct{}{}
	}
	if err := session.SendAudio(ctx, chunk); err != nil {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.audio.append.error", s.now().UnixMilli())
		return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: err.Error(), Final: true},
		}), true
	}
	s.metrics.realtimeAudioUplinkFrames.Inc()
	s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.audio.append", s.now().UnixMilli())
	if started {
		return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionListening, Mode: protocol.ModeWorkmate, Text: "我在听"},
		}), true
	}
	return nil, true
}

func (s *Server) shouldSuppressUnarmedPhysicalRealtimeAudio(frame protocol.Envelope, traceID string, sessionID string, ingress audio.IngressResult) bool {
	if !physicalStackChanDeviceID(frame.DeviceID) {
		return false
	}
	if s.realtimeAudioSession(traceID, sessionID, frame.DeviceID) != nil {
		return false
	}
	if !ingress.SpeechDetected {
		return false
	}
	if s.consumeRealtimeOnNextSpeech(frame.DeviceID, traceID, sessionID) {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.realtime_audio.physical_armed", s.now().UnixMilli())
		return false
	}
	s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.realtime_audio.physical_suppressed", s.now().UnixMilli())
	return true
}

func (s *Server) startRealtimeAudioDownlinkPump(ctx context.Context, conn *websocket.Conn, writeMu *sync.Mutex, deviceID string, traceID string, sessionID string, session providers.RealtimeVoiceSession) {
	events := session.Events()
	if events == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				outputTraceID := firstNonEmpty(event.Session.TraceID, traceID)
				outputSessionID := firstNonEmpty(event.Session.SessionID, sessionID)
				outputDeviceID := firstNonEmpty(event.Session.DeviceID, deviceID)
				payload := voiceEventToControlPayload(event, protocol.ModeWorkmate)
				if payload.State == protocol.ExpressionSpeaking && payload.StreamID != "" {
					s.setActiveStream(outputTraceID, outputSessionID, outputDeviceID, payload.StreamID)
				}
				s.metrics.realtimeAudioDownlinkEvents.Inc()
				eventAt := s.now()
				s.recordTrace(outputTraceID, outputSessionID, outputDeviceID, "provider.audio.downlink", eventAt.UnixMilli())
				if event.Audio != nil {
					s.observeRealtimeFirstAudioDownlink(outputTraceID, outputSessionID, outputDeviceID, eventAt)
				}
				envelopes := s.realtimeOutputSequence(outputDeviceID, outputTraceID, outputSessionID, []realtimeVoiceOutput{
					{Control: payload, Audio: event.Audio},
				})
				for _, envelope := range envelopes {
					if err := writeAudioEnvelope(ctx, conn, writeMu, envelope); err != nil {
						return
					}
				}
			}
		}
	}()
}

func (s *Server) shouldBargeIn(frame protocol.Envelope, traceID string, sessionID string, ingress audio.IngressResult) bool {
	if frame.Kind != protocol.KindAudioFrame {
		return false
	}
	if !containsAudioIngressEvent(ingress.Events, audio.EventVADSpeechStart) {
		return false
	}
	return s.activeStream(traceID, sessionID, frame.DeviceID) != ""
}

func containsAudioIngressEvent(events []audio.Event, want audio.Event) bool {
	for _, event := range events {
		if event == want {
			return true
		}
	}
	return false
}

func (s *Server) audioBargeInEvents(frame protocol.Envelope, traceID string, sessionID string) []protocol.Envelope {
	streamID := s.clearActiveStream(traceID, sessionID, frame.DeviceID)
	s.metrics.bargeInTotal.Inc()
	now := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, frame.DeviceID, "barge_in.detected", now)
	s.recordTrace(traceID, sessionID, frame.DeviceID, "playback.stop", now)
	s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.cancel", now)
	payloads := make([]protocol.ControlEventPayload, 0, 2)
	providerEvents, err := s.currentVoiceProvider().Cancel(context.Background(), providers.VoiceCancelRequest{
		Session:  providers.VoiceSession{TraceID: traceID, SessionID: sessionID, DeviceID: frame.DeviceID},
		Reason:   providers.CancelBargeIn,
		StreamID: streamID,
	})
	if err != nil {
		payloads = append(payloads, protocol.ControlEventPayload{
			State:    protocol.ExpressionInterrupted,
			Mode:     protocol.ModeWorkmate,
			Text:     "好，我听新的。",
			StreamID: streamID,
		})
	} else {
		for event := range providerEvents {
			payloads = append(payloads, voiceEventToControlPayload(event, protocol.ModeWorkmate))
		}
	}
	payloads = append(payloads, protocol.ControlEventPayload{State: protocol.ExpressionListening, Mode: protocol.ModeWorkmate, Text: "你说。"})
	return s.controlSequence(frame.DeviceID, traceID, sessionID, payloads)
}

func a21WebSocketAcceptOptions() *websocket.AcceptOptions {
	return &websocket.AcceptOptions{
		Subprotocols:   []string{"arduino"},
		OriginPatterns: []string{"file://"},
	}
}

func (s *Server) controlEventsForDeviceEvent(event protocol.Envelope) []protocol.Envelope {
	var payload protocol.DeviceEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return s.errorEvents(event, "invalid device event")
	}
	traceID, sessionID := s.ids(event.TraceID, event.SessionID)
	event.TraceID = traceID
	event.SessionID = sessionID
	s.recordTrace(traceID, sessionID, event.DeviceID, "device."+string(payload.Event)+".received", s.now().UnixMilli())
	record := s.recordDeviceEvent(event, payload)
	if record.IdentityStatus == "invalid" {
		s.metrics.deviceIdentityInvalidTotal.Inc()
		return s.errorEvents(event, "invalid device firmware identity: "+record.IdentityError)
	}
	req := MockTurnRequest{
		DeviceID:  event.DeviceID,
		Text:      payload.Text,
		Mode:      payload.Mode,
		TraceID:   event.TraceID,
		SessionID: event.SessionID,
	}
	switch payload.Event {
	case protocol.DeviceEventMockTurn, protocol.DeviceEventTouchWakeOrListen:
		return s.mockTurnResponse(req).Events
	case protocol.DeviceEventInterrupt, protocol.DeviceEventTouchBargeIn:
		return s.mockInterruptResponse(req).Events
	case protocol.DeviceEventTouchTopTap:
		return s.controlSequence(event.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionListening, Mode: protocol.ModeWorkmate, Text: "我在。", Final: true},
		})
	case protocol.DeviceEventTouchTopSwipeForward:
		return s.controlSequence(event.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionThinking, Mode: protocol.ModeCoCreation, Text: "往前推一步。", Final: true},
		})
	case protocol.DeviceEventTouchTopSwipeBackward:
		return s.controlSequence(event.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionIdle, Mode: protocol.ModeFocus, Text: "我先收一收。", Final: true},
		})
	case protocol.DeviceEventRuntimeEcho:
		return nil
	default:
		return s.errorEvents(event, "unsupported device event")
	}
}

func (s *Server) recordTrace(traceID string, sessionID string, deviceID string, name string, atMS int64) {
	if traceID == "" || name == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	events := s.traces[traceID]
	offset := int64(0)
	if len(events) > 0 {
		offset = atMS - events[0].AtMS
		if offset < 0 {
			offset = 0
		}
	}
	event := TraceEvent{
		Name:      name,
		TraceID:   traceID,
		SessionID: sessionID,
		DeviceID:  deviceID,
		AtMS:      atMS,
		OffsetMS:  offset,
	}
	s.traces[traceID] = append(events, event)
}

func (s *Server) traceEvents(traceID string) []TraceEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := append([]TraceEvent(nil), s.traces[traceID]...)
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].OffsetMS < events[j].OffsetMS
	})
	return events
}

func traceLatencySummary(events []TraceEvent) TraceLatencySummary {
	summary := TraceLatencySummary{EventCount: len(events)}
	if len(events) == 0 {
		return summary
	}
	for _, event := range events {
		if event.OffsetMS > summary.LastOffsetMS {
			summary.LastOffsetMS = event.OffsetMS
		}
	}
	summary.AudioFrameToPlaybackMS = traceDeltaMS(events, "audio.frame.received", "audio.playback.chunk.sent")
	summary.V21QueryFirstResultMS = traceDeltaMS(events, "v21.query.start", "v21.query.first_result")
	summary.BargeInStopMS = traceDeltaMS(events, "barge_in.detected", "playback.stop")
	summary.ProviderCommitToFirstAudioMS = traceDeltaMS(events, "provider.audio.commit", "provider.audio.first_downlink")
	summary.XiaozhiListenToAudioIngressMS = traceDeltaMS(events, "xiaozhi.listen.start", "audio.ingress.buffered")
	summary.XiaozhiOpusDecodeMS = traceDeltaMS(events, "xiaozhi.opus_frame.received", "xiaozhi.opus_frame.decoded")
	summary.ASRFirstPartialMS = firstTraceDelta(
		traceDeltaMS(events, "audio.ingress.buffered", "asr.first_partial"),
		traceDeltaMS(events, "audio.ingress.buffered", "asr.final"),
	)
	llmStartName := "asr.first_partial"
	if !traceHasEvent(events, llmStartName) {
		llmStartName = "asr.final"
	}
	summary.LLMFirstContentMS = traceDeltaMS(events, llmStartName, "provider.first_content")
	summary.TTSFirstAudioMS = traceDeltaMS(events, "provider.first_content", "tts.first_audio")
	summary.AudioDownlinkFirstFrameMS = traceDeltaMS(events, "tts.first_audio", "audio.downlink.first_frame")
	summary.DevicePlaybackStartMS = traceDeltaMS(events, "audio.downlink.first_frame", "device.playback.start")
	summary.AnswerFirstAudioTotalMS = traceDeltaMS(events, "audio.ingress.buffered", "audio.downlink.first_frame")
	return summary
}

func traceDeltaMS(events []TraceEvent, startName string, endName string) *int64 {
	var startAtMS int64
	hasStart := false
	var latestDelta *int64
	for _, event := range events {
		switch {
		case event.Name == startName:
			startAtMS = event.AtMS
			hasStart = true
		case event.Name == endName && hasStart:
			delta := event.AtMS - startAtMS
			if delta < 0 {
				delta = 0
			}
			latestDelta = &delta
			hasStart = false
		}
	}
	return latestDelta
}

func firstTraceDelta(values ...*int64) *int64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func traceHasEvent(events []TraceEvent, name string) bool {
	for _, event := range events {
		if event.Name == name {
			return true
		}
	}
	return false
}

func (s *Server) recordDeviceEvent(event protocol.Envelope, payload protocol.DeviceEventPayload) DeviceRecord {
	nowMS := s.now().UnixMilli()
	firmware := DeviceFirmwareIdentity{
		ID:      payload.FirmwareID,
		Version: payload.FirmwareVersion,
		Board:   payload.FirmwareBoard,
		Commit:  payload.FirmwareCommit,
	}
	status, identityError := validateFirmwareIdentity(firmware)
	capabilities, capabilityError := sanitizeDeviceStringMap(payload.Capabilities, "device capabilities")
	if capabilityError != "" {
		status = "invalid"
		if identityError != "" {
			identityError += "; " + capabilityError
		} else {
			identityError = capabilityError
		}
	}
	runtimeEcho, runtimeEchoError := sanitizeDeviceStringMap(payload.RuntimeEcho, "device runtime echo values")
	if runtimeEchoError != "" {
		status = "invalid"
		if identityError != "" {
			identityError += "; " + runtimeEchoError
		} else {
			identityError = runtimeEchoError
		}
	}

	s.mu.Lock()
	record := s.devices[event.DeviceID]
	if record.DeviceID == "" {
		record.DeviceID = event.DeviceID
		record.FirstSeenMS = nowMS
	}
	record.Firmware = firmware
	record.Capabilities = capabilities
	if runtimeEcho != nil {
		record.RuntimeEcho = runtimeEcho
	}
	record.IdentityStatus = status
	record.IdentityError = identityError
	record.LastEvent = payload.Event
	if payload.TouchSource != "" {
		record.LastTouchSource = payload.TouchSource
	}
	record.LastSeq = event.Seq
	record.LastTraceID = event.TraceID
	record.LastSessionID = event.SessionID
	record.LastSeenMS = nowMS
	s.devices[event.DeviceID] = record
	displayState := payload.DisplayState
	s.mu.Unlock()
	if displayState != "" {
		s.recordDeviceDisplayState(event.DeviceID, event.TraceID, event.SessionID, "device_event", string(displayState))
		s.mu.Lock()
		record = s.devices[event.DeviceID]
		s.mu.Unlock()
		return record
	}
	return record
}

func sanitizeDeviceStringMap(input map[string]string, label string) (map[string]string, string) {
	if len(input) == 0 {
		return nil, ""
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		cleanKey := strings.TrimSpace(key)
		cleanValue := strings.TrimSpace(value)
		if cleanKey == "" || cleanValue == "" {
			continue
		}
		if strings.Contains(strings.ToLower(cleanKey), "x21") ||
			strings.Contains(strings.ToLower(cleanKey), "v21") ||
			strings.Contains(strings.ToLower(cleanValue), "x21") ||
			strings.Contains(strings.ToLower(cleanValue), "v21") {
			return nil, label + " contain forbidden legacy identity"
		}
		output[cleanKey] = cleanValue
	}
	return output, ""
}

func (s *Server) recordDeviceControl(deviceID string, traceID string, sessionID string, payload protocol.ControlEventPayload, atMS int64) {
	if deviceID == "" {
		return
	}
	roleplayState := s.currentRoleplayDeviceState()
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[deviceID]
	if record.DeviceID == "" {
		record.DeviceID = deviceID
		record.FirstSeenMS = atMS
		record.IdentityStatus = "unknown"
	}
	if payload.Mode != "" {
		record.CurrentMode = payload.Mode
	}
	record.CurrentVoiceMode = defaultVoiceMode(s.voiceModeConfig)
	record.CurrentVoiceChainMode = defaultVoiceChainMode(s.voiceChainModeConfig)
	record.CurrentASRProfile = defaultCascadeASRProfile(s.cascadeASRProfileConfig)
	record.CurrentLLMProfile = defaultCascadeLLMProfile(s.cascadeLLMProfileConfig)
	record.CurrentTTSProfile = voiceChainTTSForVoice(defaultFixedTTSProfile(s.fixedTTSProfileConfig), defaultVoiceCloneProfile(s.voiceCloneProfileConfig))
	record.CurrentRealtimeProvider = defaultRealtimeProvider(s.realtimeProviderConfig)
	record.CurrentVoiceCloneProfile = defaultVoiceCloneProfile(s.voiceCloneProfileConfig)
	applyRoleplayDeviceState(&record, roleplayState)
	record.CurrentCloudVoiceProfile = providers.DefaultCloudVoiceProfile(s.cloudVoiceProfileConfig)
	record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, roleplayDeviceRuntimeEcho(roleplayState))
	if payload.State != "" {
		record.CurrentExpr = payload.State
	}
	if payload.State != "" {
		streamKey := streamStateKey(traceID, sessionID, deviceID)
		if payload.State == protocol.ExpressionSpeaking {
			if payload.StreamID != "" {
				record.PlaybackStream = payload.StreamID
				s.activeStreams[streamKey] = payload.StreamID
			}
		} else {
			record.PlaybackStream = ""
			delete(s.activeStreams, streamKey)
		}
	}
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = atMS
	s.devices[deviceID] = record
}

func (s *Server) registerAudioSocket(deviceID string, conn *websocket.Conn, writeMu *sync.Mutex) {
	if !validA21DeviceID(deviceID) || conn == nil || writeMu == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audioSockets[deviceID] = &deviceSocket{conn: conn, writeMu: writeMu, connected: s.now().UnixMilli()}
}

func (s *Server) unregisterAudioSocket(deviceID string, conn *websocket.Conn) {
	if deviceID == "" || conn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.audioSockets[deviceID]
	if socket != nil && socket.conn == conn {
		delete(s.audioSockets, deviceID)
	}
}

func (s *Server) audioSocket(deviceID string) (*deviceSocket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.audioSockets[deviceID]
	return socket, socket != nil
}

func (s *Server) registerXiaozhiSocket(deviceID string, conn *websocket.Conn, writeMu *sync.Mutex, session *xiaozhiSession, features xiaozhitransport.HelloFeatures) {
	if !validA21DeviceID(deviceID) || conn == nil || writeMu == nil || session == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.xiaozhiSockets[deviceID] = &xiaozhiDeviceSocket{
		conn:      conn,
		writeMu:   writeMu,
		session:   session,
		features:  features,
		connected: s.now().UnixMilli(),
	}
}

func (s *Server) unregisterXiaozhiSocket(deviceID string, conn *websocket.Conn) {
	if deviceID == "" || conn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.xiaozhiSockets[deviceID]
	if socket != nil && socket.conn == conn {
		delete(s.xiaozhiSockets, deviceID)
		nowMS := s.now().UnixMilli()
		record := s.devices[deviceID]
		if record.DeviceID == "" {
			record.DeviceID = deviceID
			record.FirstSeenMS = nowMS
		}
		if record.IdentityStatus == "" {
			record.IdentityStatus = "unknown"
		}
		record.ConnectionStatus = "xiaozhi_ws_disconnected"
		if socket.session != nil {
			record.LastTraceID = socket.session.traceID
			record.LastSessionID = socket.session.sessionID
		}
		record.LastSeenMS = nowMS
		s.devices[deviceID] = record
	}
}

func (s *Server) xiaozhiSocket(deviceID string) (*xiaozhiDeviceSocket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.xiaozhiSockets[deviceID]
	return socket, socket != nil
}

func (s *Server) registerOfficialStackChanSocket(deviceID string, conn *websocket.Conn, writeMu *sync.Mutex) {
	if !validA21DeviceID(deviceID) || conn == nil || writeMu == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.officialStackChanSockets[deviceID] = &deviceSocket{conn: conn, writeMu: writeMu, connected: s.now().UnixMilli()}
}

func (s *Server) unregisterOfficialStackChanSocket(deviceID string, conn *websocket.Conn) {
	if deviceID == "" || conn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.officialStackChanSockets[deviceID]
	if socket != nil && socket.conn == conn {
		delete(s.officialStackChanSockets, deviceID)
	}
}

func (s *Server) officialStackChanSocket(deviceID string) (*deviceSocket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.officialStackChanSockets[deviceID]
	return socket, socket != nil
}

func (s *Server) officialStackChanSocketForXiaozhiDevice(deviceID string) (*deviceSocket, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if socket := s.officialStackChanSockets[deviceID]; socket != nil {
		return socket, deviceID, true
	}
	if socket := s.officialStackChanSockets[defaultOfficialStackChanDeviceID]; socket != nil {
		return socket, defaultOfficialStackChanDeviceID, true
	}
	return nil, "", false
}

func (s *Server) officialStackChanStatus(deviceID string) OfficialStackChanStatusResponse {
	nowMS := s.now().UnixMilli()
	var record DeviceRecord
	var found bool
	officialDeviceID := ""
	connectedSinceMS := int64(0)
	s.mu.Lock()
	record, found = s.devices[deviceID]
	if socket := s.officialStackChanSockets[deviceID]; socket != nil {
		officialDeviceID = deviceID
		connectedSinceMS = socket.connected
	} else if socket := s.officialStackChanSockets[defaultOfficialStackChanDeviceID]; socket != nil {
		officialDeviceID = defaultOfficialStackChanDeviceID
		connectedSinceMS = socket.connected
	}
	if found && record.RuntimeEcho != nil {
		record.RuntimeEcho = mergeDeviceCapabilities(nil, record.RuntimeEcho)
	}
	s.mu.Unlock()

	connected := connectedSinceMS > 0
	deliveredTransport := "xiaozhi_mcp_fallback_available"
	connectedMS := int64(0)
	if connected {
		deliveredTransport = "stackchan_official_ws"
		connectedMS = nowMS - connectedSinceMS
		if connectedMS < 0 {
			connectedMS = 0
		}
	}
	runtimeEcho := record.RuntimeEcho
	packetCount := officialStackChanStatusPacketCount(runtimeEcho)
	physicalAccepted := officialStackChanStatusPhysicalAccepted(runtimeEcho)
	nextAction := "connect_official_stackchan_ws"
	if connected && physicalAccepted {
		nextAction = "official_relay_ready"
	} else if connected {
		nextAction = "send_official_control_and_collect_physical_acceptance"
	}
	lastEvent := ""
	if found && record.LastEvent != "" {
		lastEvent = string(record.LastEvent)
	}
	return OfficialStackChanStatusResponse{
		SchemaVersion:          "a21.stackchan.official.status.v1",
		DeviceID:               deviceID,
		OfficialDeviceID:       officialDeviceID,
		Connected:              connected,
		ConnectedMS:            connectedMS,
		ConnectedSinceMS:       connectedSinceMS,
		FallbackAvailable:      true,
		DeliveredTransport:     deliveredTransport,
		LastTraceID:            record.LastTraceID,
		LastSessionID:          record.LastSessionID,
		LastEvent:              lastEvent,
		LastAutoState:          runtimeEcho["official_stackchan_auto_state"],
		LastAutoReason:         runtimeEcho["official_stackchan_auto_reason"],
		LastAutoTarget:         runtimeEcho["official_stackchan_auto_target"],
		LastPacketCount:        packetCount,
		PhysicalAccepted:       physicalAccepted,
		OfficialActionSurfaces: officialStackChanStatusSurfaces(runtimeEcho),
		NextAction:             nextAction,
	}
}

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
	nowMS := s.now().UnixMilli()
	s.recordTrace("", "", deviceID, "stackchan.official_ws.connected", nowMS)
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
	record.LastSeenMS = nowMS
	record.LastEvent = protocol.DeviceEventKind("stackchan.official_ws.connected")
	s.devices[deviceID] = record
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
	if strings.TrimSpace(runtimeEcho["battery_mv"]) != "" {
		batteryTelemetry = "diagnostic_runtime_echo"
	}
	physicalAccepted := found && capabilities["power_lifecycle_physical_accepted"] == "true"
	items := []PowerLifecycleItem{
		powerLifecycleItem("runtime_online", "Runtime online", found && record.ConnectionStatus == "online", false, record.ConnectionStatus, "gateway_registry", "reconnect_device"),
		powerLifecycleItem("xiaozhi_socket", "Xiaozhi socket", xiaozhiOnline, false, boolStatus(xiaozhiOnline), "gateway_socket_registry", "restore_xiaozhi_socket"),
		powerLifecycleItem("battery_telemetry", "Battery telemetry", strings.TrimSpace(runtimeEcho["battery_mv"]) != "", false, batteryTelemetry, "device_runtime_echo", "run_sensor_battery_diagnostic_probe"),
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

func (s *Server) handleFastCompanionTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req FastCompanionTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	if req.Mode == protocol.ModeProfessional {
		http.Error(w, "professional mode must use the professional path with V21 evidence", http.StatusBadRequest)
		return
	}
	if req.Mode != protocol.ModeRoleplay && req.Mode != protocol.ModeWorkmate && req.Mode != protocol.ModeCompanion {
		http.Error(w, "fast companion mode must be roleplay, companion, or workmate", http.StatusBadRequest)
		return
	}
	selectedVoiceMode := s.selectedVoiceMode()
	if !voiceModeAvailableForFastCompanion(selectedVoiceMode) {
		http.Error(w, plannedVoiceModeError(selectedVoiceMode), http.StatusConflict)
		return
	}
	if strings.TrimSpace(req.LocalAudio.ASRProvider) == "" {
		http.Error(w, "local_audio.asr_provider is required", http.StatusBadRequest)
		return
	}
	if req.LocalAudio.FirstPartialMS < 0 || req.LocalAudio.FinalTranscriptChars < 0 {
		http.Error(w, "local_audio timing and transcript counts must be non-negative", http.StatusBadRequest)
		return
	}
	frames, err := fastCompanionVoicePipelineFrames(req.LocalAudio.Frames)
	if err != nil {
		http.Error(w, "invalid local_audio.frames", http.StatusBadRequest)
		return
	}
	roleplay, _, err := s.roleplayRuntimeSummary(req.Roleplay)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	receivedAtMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.turn.received", receivedAtMS)
	s.recordTrace(traceID, sessionID, req.DeviceID, "roleplay.profile.ready", receivedAtMS)
	if roleplay.MemoryPromptInputReady {
		s.recordTrace(traceID, sessionID, req.DeviceID, "roleplay.memory.ready", receivedAtMS)
	}
	writeJSON(w, http.StatusOK, s.fastCompanionTurnResponse(r.Context(), req, receivedAtMS, frames, roleplay))
}

func (s *Server) fastCompanionTurnResponse(ctx context.Context, req FastCompanionTurnRequest, receivedAtMS int64, frames []providers.VoicePipelinePCMFrame, roleplay RoleplayRuntimeSummary) FastCompanionTurnResponse {
	if len(frames) > 0 {
		return s.fastCompanionVoicePipelineTurnResponse(ctx, req, receivedAtMS, frames, roleplay)
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	asrFirstPartialAtMS := receivedAtMS + req.LocalAudio.FirstPartialMS
	placeholderStartAtMS := asrFirstPartialAtMS + 1
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.local_audio.frontend.accepted", receivedAtMS)
	s.recordTrace(traceID, sessionID, req.DeviceID, "asr.first_partial", asrFirstPartialAtMS)
	s.recordTrace(traceID, sessionID, req.DeviceID, "provider.text_stream.route.placeholder", placeholderStartAtMS)
	s.recordTrace(traceID, sessionID, req.DeviceID, "provider.first_byte", placeholderStartAtMS+1)
	s.recordTrace(traceID, sessionID, req.DeviceID, "provider.first_content", placeholderStartAtMS+2)
	s.recordTrace(traceID, sessionID, req.DeviceID, "tts.first_audio", placeholderStartAtMS+3)
	s.recordTrace(traceID, sessionID, req.DeviceID, "audio.downlink.first_frame", placeholderStartAtMS+4)
	s.recordTrace(traceID, sessionID, req.DeviceID, "device.playback.start", placeholderStartAtMS+5)
	events := s.controlSequence(req.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
		{State: protocol.ExpressionListening, Mode: req.Mode, Text: "我在听"},
		{State: protocol.ExpressionThinking, Mode: req.Mode, Text: "我把本地语音结果接到文本流边界。"},
		{State: protocol.ExpressionSpeaking, Mode: req.Mode, Text: "先走文本流占位边界，不启动真实 provider。", Final: true, StreamID: "a21-fast-companion-placeholder-stream"},
	})
	return FastCompanionTurnResponse{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Mode:               req.Mode,
		Status:             "boundary_ready",
		Route:              "fast_companion_hybrid",
		AudioFrontend:      "local_audio",
		TextStreamProvider: "mock_text_stream",
		ProviderFamily:     string(providers.ProviderFamilyTextStream),
		TextStreamExecuted: false,
		Roleplay:           roleplay,
		Events:             events,
	}
}

func (s *Server) fastCompanionVoicePipelineTurnResponse(ctx context.Context, req FastCompanionTurnRequest, receivedAtMS int64, frames []providers.VoicePipelinePCMFrame, roleplay RoleplayRuntimeSummary) FastCompanionTurnResponse {
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.local_audio.frontend.accepted", receivedAtMS)
	startAtMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.voice_pipeline.start", startAtMS)
	newRunner := s.currentXiaozhiVoicePipelineRunnerFactory()
	if newRunner == nil {
		newRunner = defaultXiaozhiVoicePipelineRunner
	}
	runner := newRunner()
	request := providers.VoicePipelineRequest{
		Session: providers.VoiceSession{
			TraceID:   traceID,
			SessionID: sessionID,
			DeviceID:  req.DeviceID,
		},
		Mode:              string(req.Mode),
		Frames:            append([]providers.VoicePipelinePCMFrame(nil), frames...),
		VoiceCloneProfile: roleplay.VoiceCloneProfile,
	}
	if prompt, err := s.roleplayPromptInput(req.Roleplay, "我在。"); err == nil && strings.TrimSpace(prompt) != "" {
		request.TextPrompt = prompt
		s.recordTrace(traceID, sessionID, req.DeviceID, "roleplay.prompt_input.used", s.now().UnixMilli())
	}
	if validVoiceCloneProfile(roleplay.VoiceCloneProfile) {
		s.recordTrace(traceID, sessionID, req.DeviceID, "roleplay.voice_clone_profile.used", s.now().UnixMilli())
	}
	result, err := runner.Run(ctx, request)
	s.recordVoicePipelineFallback(traceID, sessionID, req.DeviceID, result.Report)
	fallbackUsed, fallbackProvider, fallbackReason := voicePipelineFallbackResponse(result.Report)
	if err != nil || result.Status != providers.VoicePipelineStatusCompleted || len(result.AudioChunks) == 0 {
		s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.voice_pipeline.unavailable", s.now().UnixMilli())
		s.recordLocalFallback(traceID, sessionID, req.DeviceID, "voice_pipeline_unavailable")
		events := s.controlSequence(req.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionListening, Mode: req.Mode, Text: "我在听"},
			{State: protocol.ExpressionThinking, Mode: req.Mode, Text: "本地语音链路暂时没有给出可播放答案。"},
			s.localFallbackPayload(),
		})
		return FastCompanionTurnResponse{
			TraceID:                    traceID,
			SessionID:                  sessionID,
			DeviceID:                   req.DeviceID,
			Mode:                       protocol.ModeLocalFallback,
			Status:                     "local_fallback",
			Route:                      "fast_companion_hybrid",
			AudioFrontend:              "local_audio",
			TextStreamProvider:         voicePipelineTextStreamProvider(result.Report),
			ProviderFamily:             string(providers.ProviderFamilyTextStream),
			TextStreamExecuted:         result.Report.Output.LLMContentChars > 0,
			TextStreamFallbackUsed:     fallbackUsed,
			TextStreamFallbackProvider: fallbackProvider,
			TextStreamFallbackReason:   fallbackReason,
			Roleplay:                   roleplay,
			Events:                     events,
		}
	}
	s.recordXiaozhiVoicePipelineStageMarkers(traceID, sessionID, req.DeviceID, startAtMS, result.Timing)
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.voice_pipeline.completed", s.now().UnixMilli())
	streamID := "a21-fast-companion-voice-pipeline"
	events := s.controlSequence(req.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
		{State: protocol.ExpressionListening, Mode: req.Mode, Text: "我在听"},
		{State: protocol.ExpressionThinking, Mode: req.Mode, Text: "我把本地语音接到文本流。"},
		{State: protocol.ExpressionSpeaking, Mode: req.Mode, Final: true, StreamID: streamID},
	})
	sentAt := s.now().UnixMilli()
	for _, chunk := range result.AudioChunks {
		firstChunk := len(events) == 3
		chunkAtMS := sentAt + int64(len(events)-3)
		if firstChunk {
			s.recordTrace(traceID, sessionID, req.DeviceID, "audio.downlink.first_frame", sentAt)
			s.recordTrace(traceID, sessionID, req.DeviceID, "device.playback.start", chunkAtMS+1)
		}
		chunk := chunk
		events = append(events, s.voiceAudioPlaybackChunk(req.DeviceID, traceID, sessionID, uint64(len(events)+1), chunkAtMS, streamID, &chunk))
	}
	return FastCompanionTurnResponse{
		TraceID:                    traceID,
		SessionID:                  sessionID,
		DeviceID:                   req.DeviceID,
		Mode:                       req.Mode,
		Status:                     "pipeline_completed",
		Route:                      "fast_companion_hybrid",
		AudioFrontend:              "local_audio",
		TextStreamProvider:         voicePipelineTextStreamProvider(result.Report),
		ProviderFamily:             string(providers.ProviderFamilyTextStream),
		TextStreamExecuted:         result.Report.Output.LLMContentChars > 0,
		TextStreamFallbackUsed:     fallbackUsed,
		TextStreamFallbackProvider: fallbackProvider,
		TextStreamFallbackReason:   fallbackReason,
		Roleplay:                   roleplay,
		Events:                     events,
	}
}

func fastCompanionVoicePipelineFrames(raw []FastCompanionLocalAudioFrame) ([]providers.VoicePipelinePCMFrame, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if len(raw) > 64 {
		return nil, fmt.Errorf("too many local audio frames")
	}
	frames := make([]providers.VoicePipelinePCMFrame, 0, len(raw))
	for i, frame := range raw {
		codec := strings.ToLower(strings.TrimSpace(frame.Codec))
		if codec == "" {
			codec = string(protocol.AudioCodecPCMS16LE)
		}
		if codec != string(protocol.AudioCodecPCMS16LE) {
			return nil, fmt.Errorf("local audio frame codec must be pcm_s16le")
		}
		if frame.SampleRateHz <= 0 || frame.Channels != 1 || frame.DurationMS <= 0 {
			return nil, fmt.Errorf("local audio frame audio shape invalid")
		}
		if frame.RMS < 0 {
			return nil, fmt.Errorf("local audio frame rms invalid")
		}
		pcm, err := base64.StdEncoding.DecodeString(strings.TrimSpace(frame.DataBase64))
		if err != nil || len(pcm) == 0 || len(pcm)%2 != 0 {
			return nil, fmt.Errorf("local audio frame payload invalid")
		}
		if frame.ByteCount > 0 && frame.ByteCount != len(pcm) {
			return nil, fmt.Errorf("local audio frame byte count mismatch")
		}
		seq := frame.Seq
		if seq == 0 {
			seq = uint64(i + 1)
		}
		frames = append(frames, providers.VoicePipelinePCMFrame{
			Seq:          seq,
			Codec:        codec,
			SampleRateHz: frame.SampleRateHz,
			Channels:     frame.Channels,
			DurationMS:   frame.DurationMS,
			ByteCount:    len(pcm),
			RMS:          frame.RMS,
			PCM16LE:      pcm,
		})
	}
	return frames, nil
}

func (s *Server) mockTurnResponse(req MockTurnRequest) MockTurnResponse {
	s.metrics.mockTurnTotal.Inc()
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	if req.Mode == protocol.ModeProfessional {
		return s.professionalTurnResponse(req)
	}
	if professionalVoiceTriggerModeAllowed(req.Mode) && professionalVoiceTrigger(req.Text) {
		traceID, sessionID := s.ids(req.TraceID, req.SessionID)
		req.TraceID = traceID
		req.SessionID = sessionID
		req.Mode = protocol.ModeProfessional
		s.recordTrace(traceID, sessionID, req.DeviceID, "professional.voice_trigger.detected", s.now().UnixMilli())
		return s.professionalTurnResponse(req)
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	payloads := []protocol.ControlEventPayload{
		{State: protocol.ExpressionListening, Mode: req.Mode, Text: "我在听"},
	}
	providerEvents, err := s.currentVoiceProvider().StartTurn(context.Background(), providers.VoiceTurnRequest{
		Session: providers.VoiceSession{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID},
		Text:    req.Text,
		Mode:    string(req.Mode),
	})
	if err != nil {
		payloads = append(payloads, protocol.ControlEventPayload{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: err.Error(), Final: true})
	} else {
		for event := range providerEvents {
			payloads = append(payloads, voiceEventToControlPayload(event, req.Mode))
		}
	}
	events := s.controlSequence(req.DeviceID, traceID, sessionID, payloads)
	return MockTurnResponse{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID, Events: events}
}

func (s *Server) professionalTurnResponse(req MockTurnRequest) MockTurnResponse {
	response, err := s.executeProfessionalQuery(ProfessionalQueryRequest{
		DeviceID:  req.DeviceID,
		Text:      req.Text,
		TraceID:   req.TraceID,
		SessionID: req.SessionID,
	})
	if err != nil {
		traceID, sessionID := s.ids(req.TraceID, req.SessionID)
		events := s.controlSequence(req.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{{
			State: protocol.ExpressionError,
			Mode:  protocol.ModeProfessional,
			Text:  "专业模式参数不完整。我先停在证据边界。",
			Final: true,
		}})
		return MockTurnResponse{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID, Events: events}
	}
	return MockTurnResponse{TraceID: response.TraceID, SessionID: response.SessionID, DeviceID: response.DeviceID, Events: response.Events}
}

func (s *Server) executeProfessionalQuery(req ProfessionalQueryRequest) (ProfessionalQueryResponse, error) {
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	deviceID := strings.TrimSpace(req.DeviceID)
	utterance := professionalQueryUtterance(req)
	workspace, err := s.professionalWorkspaceRuntime(ProfessionalWorkspaceSelectionRequest{
		UserID:      req.UserID,
		WorkspaceID: req.WorkspaceID,
		QueryScope:  req.QueryScope,
	})
	if err != nil {
		return ProfessionalQueryResponse{}, err
	}
	preQueryPayloads := []protocol.ControlEventPayload{
		{State: protocol.ExpressionListening, Mode: protocol.ModeProfessional, Text: "我在听"},
		{State: protocol.ExpressionProfessional, Mode: protocol.ModeProfessional, Text: "进入专业模式。情绪先放旁边，现在只看证据。"},
		{State: protocol.ExpressionThinking, Mode: protocol.ModeProfessional, Text: "我在查，先把证据和置信度拉出来。"},
	}
	events := s.controlSequence(deviceID, traceID, sessionID, preQueryPayloads)
	s.recordTrace(traceID, sessionID, deviceID, "professional.query.started", s.now().UnixMilli())
	s.recordTrace(traceID, sessionID, deviceID, "professional.checking_feedback.sent", s.now().UnixMilli())
	s.recordTrace(traceID, sessionID, deviceID, "professional.workspace.ready", s.now().UnixMilli())
	s.recordTrace(traceID, sessionID, deviceID, "professional.query_scope."+workspace.QueryScope, s.now().UnixMilli())
	utteranceBucket := v21UtteranceLengthBucket(utterance)
	queryRequest := v21adapter.QueryRequest{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           deviceID,
		UserID:             workspace.UserID,
		WorkspaceID:        workspace.WorkspaceID,
		Mode:               "professional",
		QueryScope:         workspace.QueryScope,
		Utterance:          utterance,
		LatencyProfile:     "fast_first",
		AnswerStyle:        "voice_first_with_citations",
		MaxFirstResponseMS: 1200,
		PrivacyScope:       "professional_only",
	}
	readRecordID := s.startProfessionalReadRecord(queryRequest, utteranceBucket)
	bindingDecision := s.professionalDeviceBindingDecision(deviceID, workspace.UserID, workspace.WorkspaceID, workspace.QueryScope)
	deviceBinding := ProfessionalQueryDeviceBinding{
		Policy:      bindingDecision.Policy,
		Status:      bindingDecision.Status,
		BindingID:   bindingDecision.BindingID,
		FailureCode: bindingDecision.FailureCode,
	}
	baseResponse := ProfessionalQueryResponse{
		SchemaVersion:          ProfessionalQuerySchemaVersion,
		Service:                DeviceRegistryServiceName,
		Status:                 "started",
		TraceID:                traceID,
		SessionID:              sessionID,
		DeviceID:               deviceID,
		Mode:                   VoiceModeProfessional,
		Route:                  "professional_query",
		AdapterContractVersion: ProfessionalAdapterContractVersion,
		Workspace:              workspace,
		DeviceBinding:          deviceBinding,
		ReadRecordID:           readRecordID,
		ReadRecordStatus:       "started",
		Events:                 events,
		Redaction:              professionalWorkspaceRedaction(),
	}
	s.recordTrace(traceID, sessionID, deviceID, bindingDecision.TraceMarker, s.now().UnixMilli())
	if !bindingDecision.Allowed {
		s.failProfessionalReadRecord(readRecordID, bindingDecision.FailureCode)
		events = append(events, s.controlSequenceFrom(deviceID, traceID, sessionID, uint64(len(events)+1), []protocol.ControlEventPayload{{
			State: protocol.ExpressionError,
			Mode:  protocol.ModeProfessional,
			Text:  "这台设备还没绑定到当前专业工作区。我先停在证据边界，避免把个人资料查错地方。",
			Final: true,
		}})...)
		baseResponse.Status = "failed"
		baseResponse.FailureCode = bindingDecision.FailureCode
		baseResponse.ReadRecordStatus = "failed"
		baseResponse.Events = events
		baseResponse.DeviceBinding = deviceBinding
		return baseResponse, nil
	}
	s.recordTrace(traceID, sessionID, deviceID, "v21.query.utterance."+utteranceBucket, s.now().UnixMilli())
	s.recordTrace(traceID, sessionID, deviceID, "v21.query.start", s.now().UnixMilli())
	queryCtx, cancel := context.WithTimeout(context.Background(), s.v21TTL)
	defer cancel()
	started := time.Now()
	response, err := s.v21.Query(queryCtx, queryRequest)
	s.metrics.v21QueryMS.Observe(float64(time.Since(started)) / float64(time.Millisecond))
	postQueryPayloads := make([]protocol.ControlEventPayload, 0, 1)
	if err != nil {
		s.failProfessionalReadRecord(readRecordID, professionalReadFailureCode(err, queryCtx))
		s.recordV21QueryFailure(traceID, sessionID, deviceID, err, queryCtx)
		postQueryPayloads = append(postQueryPayloads, protocol.ControlEventPayload{
			State: protocol.ExpressionError,
			Mode:  protocol.ModeProfessional,
			Text:  "V21 现在没接上。我先把这个问题留住，等专业系统回来再查证据。",
			Final: true,
		})
		baseResponse.Status = "failed"
		baseResponse.FailureCode = professionalReadFailureCode(err, queryCtx)
		baseResponse.ReadRecordStatus = "failed"
	} else {
		report, reportErr := v21adapter.NewProfessionalBridgeEvidenceReport(response)
		if reportErr != nil {
			s.failProfessionalReadRecord(readRecordID, "contract_invalid")
			s.recordTrace(traceID, sessionID, deviceID, "v21.query.error", s.now().UnixMilli())
			s.recordTrace(traceID, sessionID, deviceID, "v21.query.error.contract_invalid", s.now().UnixMilli())
			postQueryPayloads = append(postQueryPayloads, protocol.ControlEventPayload{
				State: protocol.ExpressionError,
				Mode:  protocol.ModeProfessional,
				Text:  "V21 现在没接上。我先把这个问题留住，等专业系统回来再查证据。",
				Final: true,
			})
			baseResponse.Status = "failed"
			baseResponse.FailureCode = "contract_invalid"
			baseResponse.ReadRecordStatus = "failed"
		} else {
			s.completeProfessionalReadRecord(readRecordID, response)
			s.recordTrace(traceID, sessionID, deviceID, "v21.query.first_result", s.now().UnixMilli())
			answer := protocol.ControlEventPayload{
				State:        protocol.ExpressionSpeaking,
				Mode:         protocol.ModeProfessional,
				Text:         response.FastAnswer,
				Final:        true,
				Confidence:   response.Confidence,
				Evidence:     v21EvidenceToProtocol(response.Evidence),
				SpeechBlocks: response.SpeechBlocks,
				ScreenCards:  v21ScreenCardsToProtocol(response.ScreenCards),
				FollowUps:    response.FollowUps,
			}
			postQueryPayloads = append(postQueryPayloads, answer)
			baseResponse.Status = "completed"
			baseResponse.ReadRecordStatus = "completed"
			baseResponse.Answer = &answer
			baseResponse.EvidenceReport = &report
		}
	}
	events = append(events, s.controlSequenceFrom(deviceID, traceID, sessionID, uint64(len(events)+1), postQueryPayloads)...)
	baseResponse.Events = events
	return baseResponse, nil
}

func (s *Server) recordV21QueryFailure(traceID string, sessionID string, deviceID string, err error, queryCtx context.Context) {
	for _, marker := range v21QueryFailureMarkers(err, queryCtx) {
		s.recordTrace(traceID, sessionID, deviceID, marker, s.now().UnixMilli())
	}
}

func v21QueryFailureMarkers(err error, queryCtx context.Context) []string {
	if errors.Is(err, context.DeadlineExceeded) || (queryCtx != nil && errors.Is(queryCtx.Err(), context.DeadlineExceeded)) {
		return []string{"v21.query.timeout", "v21.query.error.timeout"}
	}
	markers := []string{"v21.query.error"}
	if class := v21adapter.QueryFailureClassOf(err); class != "" {
		markers = append(markers, "v21.query.error."+string(class))
	}
	if statusClass := v21adapter.QueryFailureStatusClassOf(err); statusClass != "" {
		markers = append(markers, "v21.query.error."+statusClass)
	}
	return markers
}

func v21UtteranceLengthBucket(utterance string) string {
	length := len([]rune(strings.TrimSpace(utterance)))
	switch {
	case length <= 0:
		return "length_empty"
	case length <= 16:
		return "length_1_16"
	case length <= 64:
		return "length_17_64"
	case length <= 160:
		return "length_65_160"
	default:
		return "length_gt_160"
	}
}

func (s *Server) mockInterruptResponse(req MockTurnRequest) MockTurnResponse {
	s.metrics.bargeInTotal.Inc()
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	mode := req.Mode
	if mode == "" {
		mode = protocol.ModeWorkmate
	}
	payloads := make([]protocol.ControlEventPayload, 0, 2)
	providerEvents, err := s.currentVoiceProvider().Cancel(context.Background(), providers.VoiceCancelRequest{
		Session: providers.VoiceSession{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID},
		Reason:  providers.CancelBargeIn,
	})
	if err != nil {
		payloads = append(payloads, protocol.ControlEventPayload{State: protocol.ExpressionInterrupted, Mode: mode, Text: "好，我听新的。"})
	} else {
		for event := range providerEvents {
			payloads = append(payloads, voiceEventToControlPayload(event, mode))
		}
	}
	payloads = append(payloads, protocol.ControlEventPayload{State: protocol.ExpressionListening, Mode: mode, Text: "你说。"})
	events := s.controlSequence(req.DeviceID, traceID, sessionID, payloads)
	return MockTurnResponse{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID, Events: events}
}

func voiceEventToControlPayload(event providers.VoiceEvent, mode protocol.Mode) protocol.ControlEventPayload {
	switch event.Kind {
	case providers.VoiceEventThinking:
		return protocol.ControlEventPayload{State: protocol.ExpressionThinking, Mode: mode, Text: event.Text}
	case providers.VoiceEventSpeaking:
		return protocol.ControlEventPayload{State: protocol.ExpressionSpeaking, Mode: mode, Text: event.Text, Final: event.Final, StreamID: event.StreamID}
	case providers.VoiceEventCancelled:
		return protocol.ControlEventPayload{State: protocol.ExpressionInterrupted, Mode: mode, Text: event.Text, Final: event.Final, StreamID: event.StreamID}
	default:
		return protocol.ControlEventPayload{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: event.Text, Final: true}
	}
}

func v21EvidenceToProtocol(evidence []v21adapter.Evidence) []protocol.EvidenceItem {
	items := make([]protocol.EvidenceItem, 0, len(evidence))
	for _, item := range evidence {
		items = append(items, protocol.EvidenceItem{
			Title:    item.Title,
			Type:     item.Type,
			SourceID: item.SourceID,
			Summary:  item.Summary,
			Quote:    item.Quote,
		})
	}
	return items
}

func v21ScreenCardsToProtocol(cards []v21adapter.ScreenCard) []protocol.ScreenCard {
	result := make([]protocol.ScreenCard, 0, len(cards))
	for _, card := range cards {
		result = append(result, protocol.ScreenCard{Label: card.Label, Text: card.Text})
	}
	return result
}

func (s *Server) errorEvents(event protocol.Envelope, text string) []protocol.Envelope {
	traceID, sessionID := s.ids(event.TraceID, event.SessionID)
	return s.controlSequence(event.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
		{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: text, Final: true},
	})
}

func (s *Server) ids(traceID string, sessionID string) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if traceID == "" || sessionID == "" {
		s.next++
	}
	if traceID == "" {
		traceID = fmt.Sprintf("a21-trace-%06d", s.next)
	}
	if sessionID == "" {
		sessionID = fmt.Sprintf("a21-session-%06d", s.next)
	}
	return traceID, sessionID
}

func (s *Server) controlSequence(deviceID string, traceID string, sessionID string, payloads []protocol.ControlEventPayload) []protocol.Envelope {
	return s.controlSequenceFrom(deviceID, traceID, sessionID, 1, payloads)
}

func (s *Server) controlSequenceFrom(deviceID string, traceID string, sessionID string, startSeq uint64, payloads []protocol.ControlEventPayload) []protocol.Envelope {
	events := make([]protocol.Envelope, 0, len(payloads))
	sentAt := s.now().UnixMilli()
	for i, payload := range payloads {
		data, _ := json.Marshal(payload)
		events = append(events, protocol.Envelope{
			Protocol:  protocol.ProtocolVersion,
			DeviceID:  deviceID,
			Kind:      protocol.KindControlEvent,
			Seq:       startSeq + uint64(i),
			TraceID:   traceID,
			SessionID: sessionID,
			SentAtMS:  sentAt + int64(i),
			Payload:   data,
		})
		s.recordTrace(traceID, sessionID, deviceID, "control."+string(payload.State)+".sent", sentAt+int64(i))
		s.recordDeviceControl(deviceID, traceID, sessionID, payload, sentAt+int64(i))
	}
	return events
}

func (s *Server) realtimeOutputSequence(deviceID string, traceID string, sessionID string, outputs []realtimeVoiceOutput) []protocol.Envelope {
	events := make([]protocol.Envelope, 0, len(outputs)*2)
	sentAt := s.now().UnixMilli()
	var seq uint64 = 1
	for _, output := range outputs {
		if output.Control.State != "" {
			data, _ := json.Marshal(output.Control)
			eventAt := sentAt + int64(seq-1)
			events = append(events, protocol.Envelope{
				Protocol:  protocol.ProtocolVersion,
				DeviceID:  deviceID,
				Kind:      protocol.KindControlEvent,
				Seq:       seq,
				TraceID:   traceID,
				SessionID: sessionID,
				SentAtMS:  eventAt,
				Payload:   data,
			})
			s.recordTrace(traceID, sessionID, deviceID, "control."+string(output.Control.State)+".sent", eventAt)
			s.recordDeviceControl(deviceID, traceID, sessionID, output.Control, eventAt)
			seq++
		}
		if output.Audio != nil {
			streamID := output.Control.StreamID
			if streamID == "" {
				streamID = "a21-provider-audio-stream"
			}
			eventAt := sentAt + int64(seq-1)
			events = append(events, s.voiceAudioPlaybackChunk(deviceID, traceID, sessionID, seq, eventAt, streamID, output.Audio))
			seq++
		}
	}
	return events
}

func (s *Server) voiceAudioPlaybackChunk(deviceID string, traceID string, sessionID string, seq uint64, sentAt int64, streamID string, audio *providers.VoiceAudioChunk) protocol.Envelope {
	codec := protocol.AudioCodec(audio.Codec)
	if codec == "" {
		codec = protocol.AudioCodecPCMS16LE
	}
	sampleRateHz := audio.SampleRateHz
	if sampleRateHz == 0 {
		sampleRateHz = 16000
	}
	channels := audio.Channels
	if channels == 0 {
		channels = 1
	}
	durationMS := audio.DurationMS
	if durationMS == 0 {
		durationMS = 20
	}
	payload := protocol.AudioPlaybackChunk{
		StreamID:     streamID,
		Codec:        codec,
		SampleRateHz: sampleRateHz,
		Channels:     channels,
		DurationMS:   durationMS,
		DataBase64:   audio.DataBase64,
	}
	data, _ := json.Marshal(payload)
	s.metrics.audioPlaybackChunkTotal.Inc()
	s.recordTrace(traceID, sessionID, deviceID, "audio.playback.chunk.sent", sentAt)
	return protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  deviceID,
		Kind:      protocol.KindAudioPlaybackChunk,
		Seq:       seq,
		TraceID:   traceID,
		SessionID: sessionID,
		SentAtMS:  sentAt,
		Payload:   data,
	}
}

func (s *Server) mockAudioPlaybackChunk(frame protocol.Envelope, traceID string, sessionID string, streamID string, seq uint64) protocol.Envelope {
	payload := protocol.AudioPlaybackChunk{
		StreamID:     streamID,
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   mockAudioPlaybackBase64(frame.DeviceID, 16000, 20),
	}
	data, _ := json.Marshal(payload)
	sentAt := s.now().UnixMilli()
	s.metrics.audioPlaybackChunkTotal.Inc()
	s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.playback.chunk.sent", sentAt)
	return protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  frame.DeviceID,
		Kind:      protocol.KindAudioPlaybackChunk,
		Seq:       seq,
		TraceID:   traceID,
		SessionID: sessionID,
		SentAtMS:  sentAt,
		Payload:   data,
	}
}

func mockAudioPlaybackBase64(deviceID string, sampleRateHz int, durationMS int) string {
	if physicalStackChanDeviceID(deviceID) {
		return mockPCM16SquareWaveBase64(sampleRateHz, durationMS)
	}
	return mockPCM16SilenceBase64(sampleRateHz, durationMS)
}

func physicalStackChanDeviceID(deviceID string) bool {
	return strings.HasPrefix(deviceID, "stackchan-") &&
		!strings.HasPrefix(deviceID, "stackchan-sim-") &&
		!strings.HasPrefix(deviceID, "stackchan-bench-")
}

func mockPCM16SilenceBase64(sampleRateHz int, durationMS int) string {
	if sampleRateHz <= 0 || durationMS <= 0 {
		return ""
	}
	byteCount := sampleRateHz * durationMS * 2 / 1000
	return base64.StdEncoding.EncodeToString(make([]byte, byteCount))
}

func mockPCM16SquareWaveBase64(sampleRateHz int, durationMS int) string {
	if sampleRateHz <= 0 || durationMS <= 0 {
		return ""
	}
	samples := sampleRateHz * durationMS / 1000
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		sample := int16(9000)
		if (i/8)%2 == 1 {
			sample = -9000
		}
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func (s *Server) setActiveStream(traceID string, sessionID string, deviceID string, streamID string) {
	if streamID == "" {
		return
	}
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeStreams[key] = streamID
}

func (s *Server) activeStream(traceID string, sessionID string, deviceID string) string {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeStreams[key]
}

func (s *Server) clearActiveStream(traceID string, sessionID string, deviceID string) string {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	streamID := s.activeStreams[key]
	delete(s.activeStreams, key)
	return streamID
}

func (s *Server) setAudioProbeOnly(deviceID string, traceID string, sessionID string, enabled bool) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		s.audioProbeSessions[key] = true
		return
	}
	delete(s.audioProbeSessions, key)
}

func (s *Server) audioProbeOnly(deviceID string, traceID string, sessionID string) bool {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.audioProbeSessions[key]
}

func (s *Server) setMockPlaybackOnNextAudioFrame(deviceID string, traceID string, sessionID string, enabled bool, mockAudioChunks *int) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		chunks := 1
		if mockAudioChunks != nil {
			chunks = *mockAudioChunks
		}
		s.mockPlaybackArmedSessions[key] = chunks
		return
	}
	delete(s.mockPlaybackArmedSessions, key)
}

func (s *Server) consumeMockPlaybackOnNextAudioFrame(deviceID string, traceID string, sessionID string) (int, bool) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	chunks, armed := s.mockPlaybackArmedSessions[key]
	if !armed {
		return 0, false
	}
	delete(s.mockPlaybackArmedSessions, key)
	return chunks, true
}

func (s *Server) setRealtimeOnNextSpeech(deviceID string, traceID string, sessionID string, enabled bool) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		s.realtimeArmedSessions[key] = true
		return
	}
	delete(s.realtimeArmedSessions, key)
}

func (s *Server) consumeRealtimeOnNextSpeech(deviceID string, traceID string, sessionID string) bool {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.realtimeArmedSessions[key] {
		return false
	}
	delete(s.realtimeArmedSessions, key)
	return true
}

func (s *Server) realtimeAudioSession(traceID string, sessionID string, deviceID string) providers.RealtimeVoiceSession {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.realtimeAudio[key]
}

func (s *Server) setRealtimeAudioSession(traceID string, sessionID string, deviceID string, session providers.RealtimeVoiceSession) {
	if session == nil {
		return
	}
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.realtimeAudio[key] = session
}

func (s *Server) markRealtimeAudioCommit(traceID string, sessionID string, deviceID string, at time.Time) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.realtimeAudioCommitAt[key] = at
	delete(s.realtimeAudioFirstDownlink, key)
}

func (s *Server) observeRealtimeFirstAudioDownlink(traceID string, sessionID string, deviceID string, at time.Time) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	commitAt, hasCommit := s.realtimeAudioCommitAt[key]
	alreadyObserved := s.realtimeAudioFirstDownlink[key]
	if hasCommit && !alreadyObserved {
		s.realtimeAudioFirstDownlink[key] = true
	}
	s.mu.Unlock()
	if !hasCommit || alreadyObserved {
		return
	}
	durationMS := float64(at.Sub(commitAt)) / float64(time.Millisecond)
	if durationMS < 0 {
		durationMS = 0
	}
	s.metrics.realtimeFirstAudioMS.Observe(durationMS)
	s.recordTrace(traceID, sessionID, deviceID, "provider.audio.first_downlink", at.UnixMilli())
}

func (s *Server) closeRealtimeAudioSessions(ctx context.Context, keys map[string]struct{}) {
	for key := range keys {
		session := s.clearRealtimeAudioSessionByKey(key)
		if session != nil {
			_ = session.Close(ctx)
		}
	}
}

func (s *Server) clearRealtimeAudioSessionByKey(key string) providers.RealtimeVoiceSession {
	if key == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.realtimeAudio[key]
	delete(s.realtimeAudio, key)
	delete(s.realtimeAudioCommitAt, key)
	delete(s.realtimeAudioFirstDownlink, key)
	return session
}

func (s *Server) mockAudioStreamID(frame protocol.Envelope, traceID string, sessionID string) string {
	streamSeq := frame.Seq
	if streamSeq == 0 {
		streamSeq = 1
	}
	key := traceID
	if key == "" {
		key = sessionID
	}
	if key == "" {
		return fmt.Sprintf("a21-audio-stream-%06d", streamSeq)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.audioStreams[key]; existing != "" {
		return existing
	}
	streamID := fmt.Sprintf("a21-audio-stream-%06d", streamSeq)
	s.audioStreams[key] = streamID
	return streamID
}

func streamStateKey(traceID string, sessionID string, deviceID string) string {
	switch {
	case sessionID != "":
		return sessionID
	case traceID != "":
		return traceID
	case deviceID != "":
		return deviceID
	default:
		return "a21-audio-stream-default"
	}
}

func writeAudioEnvelope(ctx context.Context, conn *websocket.Conn, mu *sync.Mutex, event protocol.Envelope) error {
	mu.Lock()
	defer mu.Unlock()
	return wsjson.Write(ctx, conn, event)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(value)
}
