package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"a21.local/a21/internal/personality"
	"a21.local/a21/internal/providers"
)

type productReadinessOptions struct {
	GatewayURL                 string
	Addr                       string
	DeviceID                   string
	OutputDir                  string
	ProviderSmokeReport        string
	ProviderRealtimeReport     string
	VoiceChainReadinessReport  string
	RoleplayVoiceReport        string
	XiaozhiReport              string
	V21ProfessionalReport      string
	V21AdapterSmokeReport      string
	PhysicalStackChanReport    string
	WakeWordFirmwarePlan       string
	WakeWordFirmwarePackage    string
	WakeWordPhysicalAcceptance string
	UseLatestReports           bool
	RequireReal                bool
	OpenBrowser                bool
	StatusOnly                 bool
	LatestReportFindings       []productReadinessFinding
}

type productReadinessReport struct {
	SchemaVersion     string                            `json:"schema_version"`
	GeneratedAtMS     int64                             `json:"generated_at_ms"`
	Status            string                            `json:"status"`
	LaunchReady       bool                              `json:"launch_ready"`
	DemoReady         bool                              `json:"demo_ready"`
	SimulatorURL      string                            `json:"simulator_url"`
	Gateway           productGatewayReadiness           `json:"gateway"`
	Provider          productProviderReadiness          `json:"provider"`
	V21               productV21Readiness               `json:"v21"`
	StackChan         productStackChanReadiness         `json:"stackchan"`
	Voice             productVoiceReadiness             `json:"voice"`
	Roleplay          productRoleplayReadiness          `json:"roleplay"`
	Memory            personality.MemoryState           `json:"memory"`
	WakeWord          productWakeWordReadiness          `json:"wake_word"`
	ServerSide        productServerSideReadiness        `json:"server_side"`
	CanonicalDecision productCanonicalReadinessDecision `json:"canonical_decision"`
	NextActions       []string                          `json:"next_actions,omitempty"`
	Findings          []productReadinessFinding         `json:"findings,omitempty"`
	ReportPath        string                            `json:"report_path,omitempty"`
}

type productGatewayReadiness struct {
	URL            string `json:"url"`
	Healthy        bool   `json:"healthy"`
	SimulatorReady bool   `json:"simulator_ready"`
	DeviceRegistry bool   `json:"device_registry_ready"`
	Status         string `json:"status"`
}

type productProviderReadiness struct {
	Primary               string   `json:"primary"`
	Selected              string   `json:"selected"`
	SelectedFamily        string   `json:"selected_family,omitempty"`
	SelectedConfigured    bool     `json:"selected_configured"`
	SelectedRouteEligible bool     `json:"selected_route_eligible"`
	RealProviderReady     bool     `json:"real_provider_ready"`
	TextStreamReady       bool     `json:"text_stream_ready"`
	VoiceRealtimeReady    bool     `json:"voice_realtime_ready"`
	SmokeEvidenceValid    bool     `json:"smoke_evidence_valid"`
	SmokeProvider         string   `json:"smoke_provider,omitempty"`
	SmokeFamily           string   `json:"smoke_family,omitempty"`
	SmokeStatus           string   `json:"smoke_status,omitempty"`
	SmokeExecuted         bool     `json:"smoke_executed,omitempty"`
	SmokeSourceReport     string   `json:"smoke_source_report,omitempty"`
	RealtimeEvidenceValid bool     `json:"realtime_evidence_valid"`
	RealtimeProvider      string   `json:"realtime_provider,omitempty"`
	RealtimeFamily        string   `json:"realtime_family,omitempty"`
	RealtimeProtocol      string   `json:"realtime_protocol,omitempty"`
	RealtimeStatus        string   `json:"realtime_status,omitempty"`
	RealtimeExecuted      bool     `json:"realtime_executed,omitempty"`
	RealtimeRouteEligible bool     `json:"realtime_route_eligible"`
	RealtimeEvidenceMode  string   `json:"realtime_evidence_mode,omitempty"`
	RealtimeSourceReport  string   `json:"realtime_source_report,omitempty"`
	MissingEnv            []string `json:"missing_env,omitempty"`
	PresentEnv            []string `json:"present_env,omitempty"`
}

type productV21Readiness struct {
	Configured                bool                                     `json:"configured"`
	Healthy                   bool                                     `json:"healthy"`
	Status                    string                                   `json:"status"`
	ProfessionalBridgeReady   bool                                     `json:"professional_bridge_ready"`
	CheckingFeedbackSupported bool                                     `json:"checking_feedback_supported"`
	MaxFirstResponseMS        int                                      `json:"max_first_response_ms"`
	EvidenceContractReady     bool                                     `json:"evidence_contract_ready"`
	QueryExecuted             bool                                     `json:"query_executed"`
	QueryPath                 string                                   `json:"query_path"`
	HealthPath                string                                   `json:"health_path"`
	Professional              productV21ProfessionalReadiness          `json:"professional"`
	ProfessionalExecution     productV21ProfessionalExecutionReadiness `json:"v21_professional_execution"`
}

type productV21ProfessionalReadiness struct {
	Valid                        bool           `json:"valid"`
	CheckingAckWithin1200        bool           `json:"checking_ack_within_1200"`
	EvidenceAvailable            bool           `json:"evidence_available"`
	CardsAvailable               bool           `json:"cards_available"`
	FollowUpsAvailable           bool           `json:"follow_ups_available"`
	QueryScope                   string         `json:"query_scope,omitempty"`
	WorkspaceStatus              string         `json:"workspace_status,omitempty"`
	SourceScopeCounts            map[string]int `json:"source_scope_counts,omitempty"`
	EvidenceCount                int            `json:"evidence_count"`
	CardCount                    int            `json:"card_count"`
	FollowUpCount                int            `json:"follow_up_count"`
	ProfessionalAcceptanceStatus string         `json:"professional_acceptance_status,omitempty"`
	SourceReport                 string         `json:"source_report,omitempty"`
	AdapterExecuted              bool           `json:"adapter_executed"`
	ReadRecordObserved           bool           `json:"read_record_observed"`
	ReadRecordCompleted          bool           `json:"read_record_completed"`
	ReadRecordQueryScope         string         `json:"read_record_query_scope,omitempty"`
	ReadRecordWorkspaceStatus    string         `json:"read_record_workspace_status,omitempty"`
	ReadRecordSourceScopeCounts  map[string]int `json:"read_record_source_scope_counts,omitempty"`
	PRDAccepted                  bool           `json:"prd_accepted"`
}

type productV21ProfessionalExecutionReadiness struct {
	Valid                        bool           `json:"valid"`
	SourceKind                   string         `json:"source_kind,omitempty"`
	SourceReport                 string         `json:"source_report,omitempty"`
	QueryExecuted                bool           `json:"query_executed"`
	AdapterExecuted              bool           `json:"adapter_executed"`
	QueryScope                   string         `json:"query_scope,omitempty"`
	WorkspaceStatus              string         `json:"workspace_status,omitempty"`
	SourceScopeCounts            map[string]int `json:"source_scope_counts,omitempty"`
	CheckingAckWithin1200        bool           `json:"checking_ack_within_1200"`
	EvidenceAvailable            bool           `json:"evidence_available"`
	CardsAvailable               bool           `json:"cards_available"`
	FollowUpsAvailable           bool           `json:"follow_ups_available"`
	EvidenceCount                int            `json:"evidence_count"`
	CardCount                    int            `json:"card_count"`
	FollowUpCount                int            `json:"follow_up_count"`
	ProfessionalAcceptanceStatus string         `json:"professional_acceptance_status,omitempty"`
	RedactionOK                  bool           `json:"redaction_ok"`
	ReadRecordObserved           bool           `json:"read_record_observed"`
	ReadRecordCompleted          bool           `json:"read_record_completed"`
	ReadRecordQueryScope         string         `json:"read_record_query_scope,omitempty"`
	ReadRecordWorkspaceStatus    string         `json:"read_record_workspace_status,omitempty"`
	ReadRecordSourceScopeCounts  map[string]int `json:"read_record_source_scope_counts,omitempty"`
	PRDAccepted                  bool           `json:"prd_accepted"`
}

type productStackChanReadiness struct {
	DeviceID                string                            `json:"device_id"`
	PhysicalDeviceOnline    bool                              `json:"physical_device_online"`
	PhysicalDeviceStale     bool                              `json:"physical_device_stale,omitempty"`
	PhysicalDeviceAgeMS     int64                             `json:"physical_device_age_ms,omitempty"`
	MaxPhysicalDeviceAgeMS  int64                             `json:"max_physical_device_age_ms,omitempty"`
	PhysicalMicrophoneReady bool                              `json:"physical_microphone_ready"`
	MicrophoneStatus        string                            `json:"microphone_status,omitempty"`
	SimulatorDeviceOnline   bool                              `json:"simulator_device_online"`
	USBSerialCandidateCount int                               `json:"usb_serial_candidate_count"`
	Status                  string                            `json:"status"`
	PhysicalEvidence        productPhysicalStackChanReadiness `json:"physical_evidence"`
}

type productPhysicalStackChanReadiness struct {
	Valid                                  bool            `json:"valid"`
	Status                                 string          `json:"status"`
	SourceReport                           string          `json:"source_report,omitempty"`
	ExecutionMode                          string          `json:"execution_mode,omitempty"`
	PromotionGate                          string          `json:"promotion_gate,omitempty"`
	AcceptanceStatus                       string          `json:"acceptance_status,omitempty"`
	PRDAccepted                            bool            `json:"prd_accepted"`
	RequiredPhysicalMetricsAvailable       bool            `json:"required_physical_metrics_available"`
	MicEvidenceAvailable                   bool            `json:"mic_evidence_available"`
	OperatorInstrumentObservationAvailable bool            `json:"operator_instrument_observation_available"`
	CandidatePhysicalEvidence              bool            `json:"candidate_physical_evidence"`
	CandidatePhysicalVoiceEvidence         bool            `json:"candidate_physical_voice_evidence"`
	GatewayDownlinkPhysicalDeviceEvidence  bool            `json:"gateway_downlink_physical_device_evidence"`
	HostLoopbackOnly                       bool            `json:"host_loopback_only"`
	PRDPhysicalAccepted                    bool            `json:"prd_physical_accepted"`
	CanonicalMetricAvailability            map[string]bool `json:"canonical_metric_availability,omitempty"`
	FindingCodes                           []string        `json:"finding_codes,omitempty"`
}

const productPhysicalDeviceFreshMaxAgeMS int64 = 300000

type productVoiceReadiness struct {
	LocalTTSReady        bool                          `json:"local_tts_ready"`
	LocalTTSEngine       string                        `json:"local_tts_engine"`
	RealASRReady         bool                          `json:"real_asr_ready"`
	ASRProvider          string                        `json:"asr_provider"`
	ContinuousVoiceReady bool                          `json:"continuous_voice_ready"`
	VoicePipeline        productVoicePipelineReadiness `json:"voice_pipeline"`
	VoiceChain           productVoiceChainReadiness    `json:"voice_chain"`
}

type productRoleplayReadiness struct {
	Available                  bool     `json:"available"`
	ImmersionReady             bool     `json:"immersion_ready"`
	Status                     string   `json:"status"`
	SelectedRoleplayProfile    string   `json:"selected_roleplay_profile,omitempty"`
	SelectedScenario           string   `json:"selected_scenario,omitempty"`
	SelectedVoiceCloneProfile  string   `json:"selected_voice_clone_profile,omitempty"`
	SoulPromptInputReady       bool     `json:"soul_prompt_input_ready"`
	PromptComposed             bool     `json:"prompt_composed"`
	PromptPartCount            int      `json:"prompt_part_count"`
	MemoryConfigured           bool     `json:"memory_configured"`
	MemoryPromptInputReady     bool     `json:"memory_prompt_input_ready"`
	MemoryReady                bool     `json:"memory_ready"`
	MemoryHintCount            int      `json:"memory_hint_count"`
	VoiceCloneProfileReady     bool     `json:"voice_clone_profile_ready"`
	ExpressionPlanAvailable    bool     `json:"expression_plan_available"`
	ExpressionDeliveryPolicy   string   `json:"expression_delivery_policy,omitempty"`
	ExpressionActionCount      int      `json:"expression_action_count"`
	ExpressionPacketCount      int      `json:"expression_packet_count"`
	ExpressionPhysicalAccepted bool     `json:"expression_physical_accepted"`
	PhysicalAccepted           bool     `json:"physical_accepted"`
	PromptStored               bool     `json:"prompt_stored"`
	MemoryTextStored           bool     `json:"memory_text_stored"`
	TranscriptStored           bool     `json:"asr_text_stored"`
	ProviderOutputStored       bool     `json:"provider_output_stored"`
	AudioStored                bool     `json:"audio_stored"`
	VoiceCloneSampleStored     bool     `json:"voice_clone_sample_stored"`
	ProfessionalRouteAllowed   bool     `json:"professional_route_allowed"`
	V21Executed                bool     `json:"v21_executed"`
	ExpressionRedactionOK      bool     `json:"expression_redaction_ok"`
	RuntimeRedactionOK         bool     `json:"runtime_redaction_ok"`
	VoiceRuntimeEvidence       bool     `json:"voice_runtime_evidence_available"`
	VoiceRuntimeMatched        bool     `json:"voice_runtime_evidence_matched"`
	VoiceRuntimeReady          bool     `json:"voice_runtime_ready"`
	VoiceRuntimeStatus         string   `json:"voice_runtime_status,omitempty"`
	VoiceRuntimeSourceReport   string   `json:"voice_runtime_source_report,omitempty"`
	VoiceRuntimeRoute          string   `json:"voice_runtime_route,omitempty"`
	VoiceRuntimeExecutionMode  string   `json:"voice_runtime_execution_mode,omitempty"`
	VoiceRuntimeTraceMarkers   int      `json:"voice_runtime_trace_marker_count"`
	VoiceRuntimeTextExecuted   bool     `json:"voice_runtime_text_stream_executed"`
	VoiceRuntimePromptUsed     bool     `json:"voice_runtime_prompt_input_used"`
	VoiceRuntimeVoiceCloneUsed bool     `json:"voice_runtime_voice_clone_used"`
	VoiceRuntimeAudioDownlink  bool     `json:"voice_runtime_audio_downlink_observed"`
	VoiceRuntimePlaybackStart  bool     `json:"voice_runtime_playback_start_observed"`
	Findings                   []string `json:"findings,omitempty"`
}

type productWakeWordReadiness struct {
	Available                     bool   `json:"available"`
	ProductReady                  bool   `json:"product_ready"`
	SchemaVersion                 string `json:"schema_version,omitempty"`
	Mode                          string `json:"mode"`
	ActivePhrase                  string `json:"active_phrase,omitempty"`
	ActivePinyin                  string `json:"active_pinyin,omitempty"`
	DesiredPhrase                 string `json:"desired_phrase,omitempty"`
	DesiredPinyin                 string `json:"desired_pinyin,omitempty"`
	Threshold                     int    `json:"threshold,omitempty"`
	RuntimeStatus                 string `json:"runtime_status"`
	RuntimeConfigurable           bool   `json:"runtime_configurable"`
	FirmwareBuildRequired         bool   `json:"firmware_build_required"`
	Code                          string `json:"code,omitempty"`
	FirmwarePlanAvailable         bool   `json:"firmware_plan_available"`
	FirmwarePlanStatus            string `json:"firmware_plan_status,omitempty"`
	FirmwarePlanSource            string `json:"firmware_plan_source_report,omitempty"`
	FirmwarePlanDryRun            bool   `json:"firmware_plan_dry_run"`
	FirmwarePlanBuild             bool   `json:"firmware_plan_build_allowed"`
	FirmwarePlanFlash             bool   `json:"firmware_plan_flash_allowed"`
	FirmwarePackageAvailable      bool   `json:"firmware_package_available"`
	FirmwarePackageStatus         string `json:"firmware_package_status,omitempty"`
	FirmwarePackageSource         string `json:"firmware_package_source_report,omitempty"`
	FirmwarePackageArtifact       string `json:"firmware_package_artifact_name,omitempty"`
	FirmwarePackageManifest       string `json:"firmware_package_manifest_name,omitempty"`
	FirmwarePackageWritten        bool   `json:"firmware_package_written"`
	FirmwarePackageFlash          bool   `json:"firmware_package_flash_allowed"`
	FirmwarePackageExecuted       bool   `json:"firmware_package_flash_executed"`
	PhysicalAcceptanceAvailable   bool   `json:"physical_acceptance_available"`
	PhysicalAcceptanceStatus      string `json:"physical_acceptance_status,omitempty"`
	PhysicalAcceptanceSource      string `json:"physical_acceptance_source_report,omitempty"`
	PhysicalAcceptanceFlashReport string `json:"physical_acceptance_flash_report,omitempty"`
	PhysicalDeviceOnline          bool   `json:"physical_device_online"`
	PhysicalFirmwareFlashed       bool   `json:"physical_firmware_flash_executed"`
	PhysicalOperatorObserved      bool   `json:"physical_operator_custom_wake_observation_present"`
	PhysicalWakePhraseMatched     bool   `json:"physical_wake_phrase_matched"`
}

type productServerSideReadiness struct {
	Status                             string                     `json:"status"`
	CandidateReady                     bool                       `json:"candidate_ready"`
	AcceptanceStatus                   string                     `json:"acceptance_status"`
	PRDAccepted                        bool                       `json:"prd_accepted"`
	GatewayReady                       bool                       `json:"gateway_ready"`
	ProviderEvidenceReady              bool                       `json:"provider_evidence_ready"`
	V21ProfessionalEvidenceReady       bool                       `json:"v21_professional_evidence_ready"`
	ProfessionalRitualReady            bool                       `json:"professional_ritual_ready"`
	ProfessionalReadRecordReady        bool                       `json:"professional_read_record_ready"`
	HostVoiceLoopbackReady             bool                       `json:"host_voice_loopback_ready"`
	RoleplayVoiceRuntimeReady          bool                       `json:"roleplay_voice_runtime_ready"`
	WakeWordReady                      bool                       `json:"wake_word_ready"`
	RequiresPhysicalAcceptance         bool                       `json:"requires_physical_acceptance"`
	VoiceChain                         productVoiceChainReadiness `json:"voice_chain"`
	ProviderSmokeSourceReport          string                     `json:"provider_smoke_source_report,omitempty"`
	V21ProfessionalSourceReport        string                     `json:"v21_professional_source_report,omitempty"`
	ProfessionalRitualSourceReport     string                     `json:"professional_ritual_source_report,omitempty"`
	ProfessionalReadRecordSourceReport string                     `json:"professional_read_record_source_report,omitempty"`
	HostVoiceSourceReport              string                     `json:"host_voice_source_report,omitempty"`
	RoleplayVoiceSourceReport          string                     `json:"roleplay_voice_source_report,omitempty"`
	MissingEvidence                    []string                   `json:"missing_evidence,omitempty"`
}

type productCanonicalReadinessDecision struct {
	Authority                  string   `json:"authority"`
	FullPRDStatus              string   `json:"full_prd_status"`
	LaunchReady                bool     `json:"launch_ready"`
	PRDAccepted                bool     `json:"prd_accepted"`
	ServerSideCandidateReady   bool     `json:"server_side_candidate_ready"`
	HostOnlyEvidenceUse        string   `json:"host_only_evidence_use"`
	RequiresPhysicalAcceptance bool     `json:"requires_physical_acceptance"`
	MissingRealEvidence        []string `json:"missing_real_evidence,omitempty"`
	MissingReportFields        []string `json:"missing_report_fields,omitempty"`
}

type productVoicePipelineReadiness struct {
	ASRProfile                   string  `json:"asr_profile"`
	ASRProfileEnv                string  `json:"asr_profile_env"`
	TextStreamProfile            string  `json:"text_stream_profile"`
	TextStreamProfileEnv         string  `json:"text_stream_profile_env"`
	TextStreamFallbackProfile    string  `json:"text_stream_fallback_profile,omitempty"`
	TextStreamFallbackProfileEnv string  `json:"text_stream_fallback_profile_env,omitempty"`
	TextStreamFallbackReady      bool    `json:"text_stream_fallback_ready,omitempty"`
	TTSProfile                   string  `json:"tts_profile"`
	TTSProfileEnv                string  `json:"tts_profile_env"`
	ExecutionMode                string  `json:"execution_mode,omitempty"`
	HostLocalASRReady            bool    `json:"host_local_asr_ready"`
	HostLocalTextReady           bool    `json:"host_local_text_ready"`
	HostLocalTTSReady            bool    `json:"host_local_tts_ready"`
	HostLoopbackCandidateReady   bool    `json:"host_loopback_candidate_ready"`
	HostProductChainReady        bool    `json:"host_product_chain_ready"`
	AcceptanceStatus             string  `json:"acceptance_status,omitempty"`
	AnswerFirstAudioP95MS        float64 `json:"answer_first_audio_p95_ms"`
	BargeInStopP95MS             float64 `json:"barge_in_stop_p95_ms"`
	FailureCount                 int     `json:"failure_count"`
	SourceReport                 string  `json:"source_report,omitempty"`
	PRDAccepted                  bool    `json:"prd_accepted"`
}

type productVoiceChainReadiness struct {
	Available                      bool     `json:"available"`
	Status                         string   `json:"status"`
	SelectedVoiceChainMode         string   `json:"selected_voice_chain_mode,omitempty"`
	SelectedASRProfile             string   `json:"selected_asr_profile,omitempty"`
	SelectedLLMProfile             string   `json:"selected_llm_profile,omitempty"`
	FixedTTSProfile                string   `json:"fixed_tts_profile,omitempty"`
	SelectedTTSProfile             string   `json:"selected_tts_profile,omitempty"`
	SelectedRealtimeProvider       string   `json:"selected_realtime_provider,omitempty"`
	SelectedVoiceCloneProfile      string   `json:"selected_voice_clone_profile,omitempty"`
	HotSwitch                      bool     `json:"hot_switch"`
	LaunchPolicyExpectedLLMProfile string   `json:"launch_policy_expected_llm_profile,omitempty"`
	LaunchPolicySatisfied          bool     `json:"launch_policy_satisfied"`
	CapabilityEvidenceAvailable    bool     `json:"capability_evidence_available"`
	CapabilityEvidenceMatched      bool     `json:"capability_evidence_matched"`
	CapabilityEvidenceStatus       string   `json:"capability_evidence_status,omitempty"`
	CapabilityEvidenceSourceReport string   `json:"capability_evidence_source_report,omitempty"`
	CapabilityExecutionMode        string   `json:"capability_execution_mode,omitempty"`
	StaticCapabilityReady          bool     `json:"static_capability_ready"`
	ASRStreamingReady              bool     `json:"asr_streaming_ready"`
	LLMStreamingReady              bool     `json:"llm_streaming_ready"`
	TTSStreamingReady              bool     `json:"tts_streaming_ready"`
	CapabilityFindings             []string `json:"capability_findings,omitempty"`
	Findings                       []string `json:"findings,omitempty"`
}

type productReadinessFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func runProductReadiness(args []string, stdout io.Writer, stderr io.Writer) int {
	options := productReadinessOptions{
		GatewayURL: firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:   firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-001"),
		OutputDir:  "reports",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 product-readiness [--gateway-url http://127.0.0.1:21080] [--device-id stackchan-001] [--provider-smoke-report report.json] [--provider-realtime-fixture-report report.json] [--voice-chain-readiness-report report.json] [--roleplay-voice-report report.json] [--xiaozhi-report report.json] [--v21-professional-report report.json] [--v21-adapter-smoke-report report.json] [--physical-stackchan-report report.json] [--wake-word-firmware-plan report.json] [--wake-word-firmware-package-report report.json] [--wake-word-physical-acceptance-report report.json] [--use-latest-reports] [--output-dir reports] [--require-real]")
			return 0
		case "--gateway-url":
			if !readStringOption(args, &i, stderr, "--gateway-url", &options.GatewayURL) {
				return 2
			}
		case "--device-id":
			if !readStringOption(args, &i, stderr, "--device-id", &options.DeviceID) {
				return 2
			}
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		case "--provider-smoke-report":
			if !readStringOption(args, &i, stderr, "--provider-smoke-report", &options.ProviderSmokeReport) {
				return 2
			}
		case "--provider-realtime-fixture-report":
			if !readStringOption(args, &i, stderr, "--provider-realtime-fixture-report", &options.ProviderRealtimeReport) {
				return 2
			}
		case "--voice-chain-readiness-report":
			if !readStringOption(args, &i, stderr, "--voice-chain-readiness-report", &options.VoiceChainReadinessReport) {
				return 2
			}
		case "--roleplay-voice-report":
			if !readStringOption(args, &i, stderr, "--roleplay-voice-report", &options.RoleplayVoiceReport) {
				return 2
			}
		case "--xiaozhi-report":
			if !readStringOption(args, &i, stderr, "--xiaozhi-report", &options.XiaozhiReport) {
				return 2
			}
		case "--v21-professional-report":
			if !readStringOption(args, &i, stderr, "--v21-professional-report", &options.V21ProfessionalReport) {
				return 2
			}
		case "--v21-adapter-smoke-report":
			if !readStringOption(args, &i, stderr, "--v21-adapter-smoke-report", &options.V21AdapterSmokeReport) {
				return 2
			}
		case "--physical-stackchan-report":
			if !readStringOption(args, &i, stderr, "--physical-stackchan-report", &options.PhysicalStackChanReport) {
				return 2
			}
		case "--wake-word-firmware-plan":
			if !readStringOption(args, &i, stderr, "--wake-word-firmware-plan", &options.WakeWordFirmwarePlan) {
				return 2
			}
		case "--wake-word-firmware-package-report":
			if !readStringOption(args, &i, stderr, "--wake-word-firmware-package-report", &options.WakeWordFirmwarePackage) {
				return 2
			}
		case "--wake-word-physical-acceptance-report":
			if !readStringOption(args, &i, stderr, "--wake-word-physical-acceptance-report", &options.WakeWordPhysicalAcceptance) {
				return 2
			}
		case "--use-latest-reports":
			options.UseLatestReports = true
		case "--require-real":
			options.RequireReal = true
		default:
			fmt.Fprintf(stderr, "unknown product-readiness option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "product-readiness report dir invalid: %v\n", err)
		return 1
	}
	if options.UseLatestReports {
		options = resolveLatestProductReadinessReports(options)
	}
	report := buildProductReadinessReport(context.Background(), options, os.Environ())
	if options.OutputDir != "" {
		reportPath, err := writeProductReadinessReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write product-readiness report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONProductReadiness(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode product-readiness report: %v\n", err)
		return 1
	}
	if options.RequireReal && !report.LaunchReady {
		return 1
	}
	return 0
}

func runDemo(args []string, stdout io.Writer, stderr io.Writer) int {
	options := productReadinessOptions{
		Addr:      "127.0.0.1:21080",
		DeviceID:  firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-001"),
		OutputDir: "reports",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21-lab demo [--addr 127.0.0.1:21080] [--device-id stackchan-001] [--open] [--status-only] [--require-real]")
			return 0
		case "--addr":
			if !readStringOption(args, &i, stderr, "--addr", &options.Addr) {
				return 2
			}
		case "--device-id":
			if !readStringOption(args, &i, stderr, "--device-id", &options.DeviceID) {
				return 2
			}
		case "--open":
			options.OpenBrowser = true
		case "--status-only":
			options.StatusOnly = true
		case "--require-real":
			options.RequireReal = true
		default:
			fmt.Fprintf(stderr, "unknown demo option %q\n", args[i])
			return 2
		}
	}
	options.GatewayURL = "http://" + options.Addr
	if options.StatusOnly {
		report := buildProductReadinessReport(context.Background(), options, os.Environ())
		if err := writeJSONProductReadiness(stdout, report); err != nil {
			fmt.Fprintf(stderr, "encode demo readiness report: %v\n", err)
			return 1
		}
		if options.RequireReal && !report.LaunchReady {
			return 1
		}
		return 0
	}

	simulatorURL := productSimulatorURL(options.GatewayURL)
	fmt.Fprintln(stdout, "A21 lab demo surface")
	fmt.Fprintf(stdout, "Simulator: %s\n", simulatorURL)
	fmt.Fprintln(stdout, "Mode: lab demo only; product readiness is reported by `a21 product-readiness`.")
	if options.OpenBrowser {
		openBrowserURL(simulatorURL)
	}
	if gatewayHealthOK(context.Background(), options.GatewayURL) {
		fmt.Fprintln(stdout, "A21 Gateway is already running.")
		return 0
	}
	return runGateway([]string{"--addr", options.Addr}, stdout, stderr)
}

func resolveLatestProductReadinessReports(options productReadinessOptions) productReadinessOptions {
	reportDir := firstNonEmpty(strings.TrimSpace(options.OutputDir), "reports")
	if strings.TrimSpace(options.ProviderRealtimeReport) == "" {
		var findings []productReadinessFinding
		options.ProviderRealtimeReport, findings = latestAcceptedProductReadinessReportPath(reportDir, "provider_realtime_fixture", []string{
			"a21-provider-realtime-fixture-*.json",
		}, productLatestProviderRealtimeFixtureReportAccepted)
		options.LatestReportFindings = append(options.LatestReportFindings, findings...)
	}
	if strings.TrimSpace(options.VoiceChainReadinessReport) == "" {
		var findings []productReadinessFinding
		options.VoiceChainReadinessReport, findings = latestAcceptedProductReadinessReportPath(reportDir, "voice_chain_readiness", []string{
			"a21-xiaozhi-streaming-provider-readiness-*.json",
		}, productLatestVoiceChainReadinessReportAccepted)
		options.LatestReportFindings = append(options.LatestReportFindings, findings...)
	}
	if strings.TrimSpace(options.RoleplayVoiceReport) == "" {
		var findings []productReadinessFinding
		options.RoleplayVoiceReport, findings = latestAcceptedProductReadinessReportPath(reportDir, "roleplay_voice_runtime", []string{
			"a21-roleplay-voice-probe-*.json",
		}, productLatestRoleplayVoiceReportAccepted)
		options.LatestReportFindings = append(options.LatestReportFindings, findings...)
	}
	if strings.TrimSpace(options.XiaozhiReport) == "" {
		var findings []productReadinessFinding
		options.XiaozhiReport, findings = latestAcceptedProductReadinessReportPath(reportDir, "xiaozhi_voice", []string{
			"a21-xiaozhi-voice-bench-*.json",
			"a21-local-voice-loopback-*.json",
		}, productLatestXiaozhiReportAccepted)
		options.LatestReportFindings = append(options.LatestReportFindings, findings...)
	}
	if strings.TrimSpace(options.PhysicalStackChanReport) == "" {
		options.PhysicalStackChanReport = latestProductReadinessReportPath(reportDir, []string{
			"a21-physical-stackchan-evidence-*.json",
			"a21-xiaozhi-physical-evidence-*.json",
		})
	}
	return options
}

type productLatestReadinessReportCandidate struct {
	Path    string
	ModTime time.Time
}

func latestProductReadinessReportPath(reportDir string, patterns []string) string {
	candidates := latestProductReadinessReportCandidates(reportDir, patterns)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

func latestAcceptedProductReadinessReportPath(reportDir string, kind string, patterns []string, accept func(string) bool) (string, []productReadinessFinding) {
	var findings []productReadinessFinding
	skipped := 0
	for _, candidate := range latestProductReadinessReportCandidates(reportDir, patterns) {
		if accept == nil || accept(candidate) {
			findings = append(findings, latestProductReadinessSkippedSummaryFinding(kind, skipped)...)
			return candidate, findings
		}
		skipped++
		if skipped <= latestProductReadinessSkippedDetailLimit {
			findings = append(findings, productReadinessFinding{
				Code:    "latest_report_candidate_skipped",
				Message: "Latest product-readiness candidate report did not satisfy the evidence contract",
				Detail:  strings.TrimSpace(kind) + ":" + filepath.Base(filepath.Clean(candidate)),
			})
		}
	}
	findings = append(findings, latestProductReadinessSkippedSummaryFinding(kind, skipped)...)
	return "", findings
}

const latestProductReadinessSkippedDetailLimit = 3

func latestProductReadinessSkippedSummaryFinding(kind string, skipped int) []productReadinessFinding {
	if skipped <= latestProductReadinessSkippedDetailLimit {
		return nil
	}
	return []productReadinessFinding{{
		Code:    "latest_report_candidates_skipped_summary",
		Message: "Older product-readiness candidate reports were also skipped because they did not satisfy the evidence contract",
		Detail:  fmt.Sprintf("%s:%d_total", strings.TrimSpace(kind), skipped),
	}}
}

func latestProductReadinessReportCandidates(reportDir string, patterns []string) []string {
	var candidates []productLatestReadinessReportCandidate
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(reportDir, pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			candidates = append(candidates, productLatestReadinessReportCandidate{Path: match, ModTime: info.ModTime()})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ModTime.Equal(candidates[j].ModTime) {
			return filepath.Base(candidates[i].Path) > filepath.Base(candidates[j].Path)
		}
		return candidates[i].ModTime.After(candidates[j].ModTime)
	})
	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		paths = append(paths, candidate.Path)
	}
	return paths
}

func resolveLatestProductProviderSmokeReport(options *productReadinessOptions, readiness productProviderReadiness) []productReadinessFinding {
	if options == nil || !options.UseLatestReports || strings.TrimSpace(options.ProviderSmokeReport) != "" {
		return nil
	}
	if !productProviderReadinessRequiresSmokeMatch(readiness) {
		return nil
	}
	reportDir := firstNonEmpty(strings.TrimSpace(options.OutputDir), "reports")
	selected, findings := latestAcceptedProductReadinessReportPath(reportDir, "provider_smoke", []string{
		"a21-provider-smoke-*.json",
	}, func(path string) bool {
		evidence, _ := loadProductProviderSmokeReportEvidence(path)
		return productProviderSmokeEvidenceMatchesSelected(readiness, evidence)
	})
	options.ProviderSmokeReport = selected
	return findings
}

func productLatestProviderRealtimeFixtureReportAccepted(path string) bool {
	evidence, _ := loadProductProviderRealtimeReportEvidence(path)
	return evidence.Valid && evidence.Configured && evidence.Executed && evidence.Status == string(providers.ProviderSmokePassed)
}

func productLatestVoiceChainReadinessReportAccepted(path string) bool {
	evidence, _ := loadProductVoiceChainReadinessReportEvidence(path)
	return evidence.Valid
}

func productLatestRoleplayVoiceReportAccepted(path string) bool {
	evidence, _ := loadProductRoleplayVoiceReportEvidence(path)
	return evidence.Valid && evidence.VoiceRuntimeReady
}
