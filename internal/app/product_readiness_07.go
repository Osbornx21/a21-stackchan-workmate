package app

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type productXiaozhiReportFixture struct {
	SchemaVersion    string                        `json:"schema_version"`
	ExecutionMode    string                        `json:"execution_mode"`
	BaselineScope    string                        `json:"baseline_scope"`
	Repeat           *int                          `json:"repeat"`
	AcceptanceStatus string                        `json:"acceptance_status"`
	PRDAccepted      bool                          `json:"prd_accepted"`
	Summary          productXiaozhiReportSummary   `json:"summary"`
	Counts           productXiaozhiReportCounts    `json:"counts"`
	Execution        productXiaozhiReportExecution `json:"execution"`
	Redaction        productXiaozhiReportRedaction `json:"redaction"`
}

type productXiaozhiReportExecution struct {
	ProviderExecuted           bool   `json:"provider_executed"`
	V21Executed                bool   `json:"v21_executed"`
	HardwareExecuted           bool   `json:"hardware_executed"`
	VoicePipelineObserved      bool   `json:"voice_pipeline_observed"`
	VoicePipelineExecutionMode string `json:"voice_pipeline_execution_mode"`
	ASRProfile                 string `json:"asr_profile,omitempty"`
	ASRProfileEnv              string `json:"asr_profile_env,omitempty"`
	LLMProfile                 string `json:"llm_profile,omitempty"`
	LLMProfileEnv              string `json:"llm_profile_env,omitempty"`
	TTSProfile                 string `json:"tts_profile,omitempty"`
	TTSProfileEnv              string `json:"tts_profile_env,omitempty"`
	HostLocalASRExecuted       bool   `json:"host_local_asr_executed"`
	HostLocalTextExecuted      bool   `json:"host_local_text_executed"`
	HostLocalTTSExecuted       bool   `json:"host_local_tts_executed"`
	HostProductChainReady      *bool  `json:"host_product_chain_ready,omitempty"`
}

func (execution productXiaozhiReportExecution) providerLatencyBenchExecution() providerLatencyBenchExecution {
	return providerLatencyBenchExecution{
		ProviderExecuted:           execution.ProviderExecuted,
		V21Executed:                execution.V21Executed,
		HardwareExecuted:           execution.HardwareExecuted,
		VoicePipelineObserved:      execution.VoicePipelineObserved,
		VoicePipelineExecutionMode: execution.VoicePipelineExecutionMode,
		ASRProfile:                 execution.ASRProfile,
		ASRProfileEnv:              execution.ASRProfileEnv,
		LLMProfile:                 execution.LLMProfile,
		LLMProfileEnv:              execution.LLMProfileEnv,
		TTSProfile:                 execution.TTSProfile,
		TTSProfileEnv:              execution.TTSProfileEnv,
		HostLocalASRExecuted:       execution.HostLocalASRExecuted,
		HostLocalTextExecuted:      execution.HostLocalTextExecuted,
		HostLocalTTSExecuted:       execution.HostLocalTTSExecuted,
	}
}

type productXiaozhiReportSummary struct {
	AnswerFirstAudioP95MS *float64 `json:"answer_first_audio_total_p95_ms"`
	BargeInStopP95MS      *float64 `json:"barge_in_stop_p95_ms"`
}

type productXiaozhiReportCounts struct {
	AnswerTurnCount  *int `json:"answer_turn_count"`
	BargeInTurnCount *int `json:"barge_in_turn_count"`
	FailureCount     *int `json:"failure_count"`
}

type productXiaozhiReportRedaction struct {
	PayloadsStored         *bool `json:"payloads_stored"`
	CredentialValuesStored *bool `json:"credential_values_stored"`
	FullURLsStored         *bool `json:"full_urls_stored"`
	LocalPathsStored       *bool `json:"local_paths_stored"`
}

type productLocalVoiceLoopbackReportFixture struct {
	SchemaVersion          string   `json:"schema_version"`
	Status                 string   `json:"status"`
	Repeat                 *int     `json:"repeat"`
	AnswerFirstAudioP95MS  *float64 `json:"answer_first_audio_total_p95_ms"`
	BargeInStopP95MS       *float64 `json:"barge_in_stop_p95_ms"`
	TextStreamExecuted     *bool    `json:"text_stream_executed"`
	ASRProvider            string   `json:"asr_provider"`
	TextStreamProvider     string   `json:"text_stream_provider"`
	TTSProvider            string   `json:"tts_provider"`
	ASRTranscriptPolicy    string   `json:"asr_transcript_policy"`
	TextStreamEndpointHost string   `json:"text_stream_endpoint_host"`
}

func loadProductXiaozhiReportEvidence(path string) (productXiaozhiReportEvidence, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productXiaozhiReportEvidence{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	var header struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	if strings.TrimSpace(header.SchemaVersion) == "a21.audio.local_voice_loopback.v1" {
		return productLocalVoiceLoopbackReportEvidence(path, data)
	}
	var fixture productXiaozhiReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	if missingField := missingProductXiaozhiReportField(fixture); missingField != "" {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{missingProductXiaozhiReportFieldFinding(missingField)}
	}
	rawExecution := fixture.Execution.providerLatencyBenchExecution()
	if fixture.SchemaVersion != "a21.xiaozhi_voice_bench.v1" ||
		fixture.ExecutionMode != "host_loopback" ||
		fixture.BaselineScope != "host_only" ||
		!productXiaozhiReportHasSufficientRounds(fixture) ||
		!productXiaozhiProviderExecutionAllowed(fixture.Execution) ||
		fixture.Execution.V21Executed ||
		fixture.Execution.HardwareExecuted ||
		*fixture.Redaction.PayloadsStored ||
		*fixture.Redaction.CredentialValuesStored ||
		*fixture.Redaction.FullURLsStored ||
		*fixture.Redaction.LocalPathsStored {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	execution := providerLatencyHostLoopbackExecution(rawExecution)
	return productXiaozhiReportEvidence{
		Valid:                 true,
		SourceReport:          filepath.Base(filepath.Clean(path)),
		AcceptanceStatus:      firstNonEmpty(strings.TrimSpace(fixture.AcceptanceStatus), "not_accepted"),
		PRDAccepted:           fixture.PRDAccepted,
		AnswerFirstAudioP95MS: *fixture.Summary.AnswerFirstAudioP95MS,
		BargeInStopP95MS:      *fixture.Summary.BargeInStopP95MS,
		FailureCount:          *fixture.Counts.FailureCount,
		HostProductChainReady: productXiaozhiHostProductChainReady(execution, fixture.Execution.HostProductChainReady),
		Execution:             execution,
	}, nil
}

func productXiaozhiProviderExecutionAllowed(execution productXiaozhiReportExecution) bool {
	if !execution.ProviderExecuted {
		return true
	}
	return productXiaozhiHostProductChainReady(
		providerLatencyHostLoopbackExecution(execution.providerLatencyBenchExecution()),
		execution.HostProductChainReady,
	)
}

func productXiaozhiHostProductChainReady(execution providerLatencyBenchExecution, explicit *bool) bool {
	mode := providerLatencySafeExecutionMode(execution.VoicePipelineExecutionMode)
	hostLocal := mode == "host_local" &&
		execution.VoicePipelineObserved &&
		execution.HostLocalASRExecuted &&
		execution.HostLocalTextExecuted &&
		execution.HostLocalTTSExecuted &&
		providerLatencySafeIdentifier(execution.ASRProfile, false) != "" &&
		providerLatencySafeIdentifier(execution.LLMProfile, false) != "" &&
		providerLatencySafeIdentifier(execution.TTSProfile, false) != ""
	cloudEdge := mode == "cloud_edge" &&
		execution.ProviderExecuted &&
		execution.VoicePipelineObserved &&
		!execution.V21Executed &&
		!execution.HardwareExecuted &&
		productXiaozhiNonMockStageProfile(execution.ASRProfile) &&
		productXiaozhiNonMockStageProfile(execution.LLMProfile) &&
		productXiaozhiNonMockStageProfile(execution.TTSProfile)
	derived := hostLocal || cloudEdge
	if explicit != nil {
		return *explicit && derived
	}
	return derived
}

func productXiaozhiNonMockStageProfile(profile string) bool {
	profile = strings.ToLower(strings.TrimSpace(providerLatencySafeIdentifier(profile, false)))
	return profile != "" &&
		profile != "mock" &&
		!strings.HasPrefix(profile, "mock-") &&
		!strings.HasPrefix(profile, "mock_") &&
		!strings.Contains(profile, "fixture")
}

func productXiaozhiReportHasSufficientRounds(fixture productXiaozhiReportFixture) bool {
	return fixture.Repeat != nil &&
		fixture.Counts.AnswerTurnCount != nil &&
		fixture.Counts.BargeInTurnCount != nil &&
		*fixture.Repeat >= 3 &&
		*fixture.Counts.AnswerTurnCount >= *fixture.Repeat &&
		*fixture.Counts.BargeInTurnCount >= *fixture.Repeat
}

func productLocalVoiceLoopbackReportEvidence(path string, data []byte) (productXiaozhiReportEvidence, []productReadinessFinding) {
	var fixture productLocalVoiceLoopbackReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	if missingField := missingProductLocalVoiceLoopbackReportField(fixture); missingField != "" {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{missingProductXiaozhiReportFieldFinding(missingField)}
	}
	asrProfile := providerLatencySafeIdentifier(fixture.ASRProvider, false)
	textProfile := providerLatencySafeIdentifier(fixture.TextStreamProvider, false)
	ttsProfile := providerLatencySafeIdentifier(fixture.TTSProvider, false)
	if ttsProfile == "sherpa_onnx" {
		ttsProfile = "sherpa_onnx_tts"
	}
	if fixture.SchemaVersion != "a21.audio.local_voice_loopback.v1" ||
		strings.TrimSpace(fixture.Status) != "passed" ||
		*fixture.Repeat < 3 ||
		asrProfile == "" ||
		textProfile == "" ||
		ttsProfile == "" ||
		*fixture.AnswerFirstAudioP95MS <= 0 ||
		*fixture.BargeInStopP95MS < 0 ||
		strings.TrimSpace(fixture.ASRTranscriptPolicy) != "transcript_not_recorded" ||
		productLocalVoiceLoopbackUnsafeEndpointHost(fixture.TextStreamEndpointHost) {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	execution := providerLatencyBenchExecution{
		ProviderExecuted:           false,
		V21Executed:                false,
		HardwareExecuted:           false,
		VoicePipelineObserved:      true,
		VoicePipelineExecutionMode: "host_local",
		ASRProfile:                 asrProfile,
		ASRProfileEnv:              "A21_ASR_LOCAL_PROFILE",
		LLMProfile:                 textProfile,
		LLMProfileEnv:              "A21_TEXT_STREAM_PROFILE",
		TTSProfile:                 ttsProfile,
		TTSProfileEnv:              "A21_TTS_FAST_PROFILE",
		HostLocalASRExecuted:       asrProfile != "" && asrProfile != "mock_asr",
		HostLocalTextExecuted:      *fixture.TextStreamExecuted && textProfile != "" && textProfile != "mock_text_stream",
		HostLocalTTSExecuted:       ttsProfile != "" && ttsProfile != "mock-fast-tts",
	}
	return productXiaozhiReportEvidence{
		Valid:                 true,
		SourceReport:          filepath.Base(filepath.Clean(path)),
		AcceptanceStatus:      "host_local_loopback_passed",
		PRDAccepted:           false,
		AnswerFirstAudioP95MS: *fixture.AnswerFirstAudioP95MS,
		BargeInStopP95MS:      *fixture.BargeInStopP95MS,
		FailureCount:          0,
		HostProductChainReady: productXiaozhiHostProductChainReady(execution, nil),
		Execution:             execution,
	}, nil
}

func missingProductLocalVoiceLoopbackReportField(fixture productLocalVoiceLoopbackReportFixture) string {
	switch {
	case fixture.Repeat == nil:
		return "repeat"
	case fixture.AnswerFirstAudioP95MS == nil:
		return "answer_first_audio_total_p95_ms"
	case fixture.BargeInStopP95MS == nil:
		return "barge_in_stop_p95_ms"
	case fixture.TextStreamExecuted == nil:
		return "text_stream_executed"
	default:
		return ""
	}
}

func productLocalVoiceLoopbackUnsafeEndpointHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	return strings.Contains(host, "http://") ||
		strings.Contains(host, "https://") ||
		strings.Contains(host, "@") ||
		strings.Contains(host, "key") ||
		strings.Contains(host, "token") ||
		strings.Contains(host, "secret") ||
		strings.Contains(host, "proxy")
}

func missingProductXiaozhiReportField(fixture productXiaozhiReportFixture) string {
	switch {
	case fixture.Repeat == nil:
		return "repeat"
	case fixture.Summary.AnswerFirstAudioP95MS == nil:
		return "summary.answer_first_audio_total_p95_ms"
	case fixture.Summary.BargeInStopP95MS == nil:
		return "summary.barge_in_stop_p95_ms"
	case fixture.Counts.AnswerTurnCount == nil:
		return "counts.answer_turn_count"
	case fixture.Counts.BargeInTurnCount == nil:
		return "counts.barge_in_turn_count"
	case fixture.Counts.FailureCount == nil:
		return "counts.failure_count"
	case fixture.Redaction.PayloadsStored == nil:
		return "redaction.payloads_stored"
	case fixture.Redaction.CredentialValuesStored == nil:
		return "redaction.credential_values_stored"
	case fixture.Redaction.FullURLsStored == nil:
		return "redaction.full_urls_stored"
	case fixture.Redaction.LocalPathsStored == nil:
		return "redaction.local_paths_stored"
	default:
		return ""
	}
}

func missingProductXiaozhiReportFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "xiaozhi_report_missing_field",
		Message: "Xiaozhi host-loopback report is missing a required field",
		Detail:  field,
	}
}

func invalidProductXiaozhiReportFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "xiaozhi_report_invalid",
		Message: "Xiaozhi host-loopback report is invalid or unsafe",
	}
}

func loadProductPhysicalStackChanReportEvidence(path string) (productPhysicalStackChanReadiness, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productPhysicalStackChanReadiness{Status: "no_report", AcceptanceStatus: "not_provided"}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productPhysicalStackChanReadiness{}, []productReadinessFinding{invalidProductPhysicalStackChanReportFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productPhysicalStackChanReadiness{}, []productReadinessFinding{invalidProductPhysicalStackChanReportFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productPhysicalStackChanReadiness{}, []productReadinessFinding{invalidProductPhysicalStackChanReportFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productPhysicalStackChanReadiness{}, []productReadinessFinding{invalidProductPhysicalStackChanReportFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || physicalStackChanValueUnsafe(raw) {
		return productPhysicalStackChanReadiness{}, []productReadinessFinding{invalidProductPhysicalStackChanReportFinding()}
	}
	var header struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return productPhysicalStackChanReadiness{}, []productReadinessFinding{invalidProductPhysicalStackChanReportFinding()}
	}
	if strings.TrimSpace(header.SchemaVersion) == xiaozhiPhysicalEvidenceSchemaVersion {
		var report xiaozhiPhysicalEvidenceReport
		if err := json.Unmarshal(data, &report); err != nil {
			return productPhysicalStackChanReadiness{}, []productReadinessFinding{invalidProductPhysicalStackChanReportFinding()}
		}
		if report.SchemaVersion != xiaozhiPhysicalEvidenceSchemaVersion || !productPhysicalStackChanRedactionOK(report.Redaction) {
			return productPhysicalStackChanReadiness{}, []productReadinessFinding{invalidProductPhysicalStackChanReportFinding()}
		}
		readiness := buildProductXiaozhiPhysicalReadiness(filepath.Base(filepath.Clean(path)), report)
		findings := []productReadinessFinding{
			{Code: "xiaozhi_physical_gateway_downlink_candidate", Message: "Xiaozhi physical evidence reached Gateway downlink but is not PRD audible playback acceptance"},
		}
		if !readiness.PRDPhysicalAccepted {
			if !readiness.CanonicalMetricAvailability["device_playback_start_ms"] {
				findings = append(findings, productReadinessFinding{Code: "xiaozhi_physical_device_playback_ack_missing", Message: "Missing device playback ack such as device.playback.start or trusted runtime echo"})
			}
			if !readiness.OperatorInstrumentObservationAvailable {
				findings = append(findings, productReadinessFinding{Code: "xiaozhi_physical_operator_observation_missing", Message: "Missing operator audible observation or instrumented first audible playback evidence"})
			}
		}
		return readiness, findings
	}
	var report physicalStackChanEvidenceReport
	if err := json.Unmarshal(data, &report); err != nil {
		return productPhysicalStackChanReadiness{}, []productReadinessFinding{invalidProductPhysicalStackChanReportFinding()}
	}
	if report.SchemaVersion != physicalStackChanEvidenceSchemaVersion || !productPhysicalStackChanRedactionOK(report.Redaction) {
		return productPhysicalStackChanReadiness{}, []productReadinessFinding{invalidProductPhysicalStackChanReportFinding()}
	}
	readiness := buildProductPhysicalStackChanReadiness(filepath.Base(filepath.Clean(path)), report)
	var findings []productReadinessFinding
	switch {
	case readiness.PRDPhysicalAccepted:
	case readiness.HostLoopbackOnly:
		findings = append(findings, productReadinessFinding{Code: "physical_stackchan_host_loopback_only", Message: "Physical StackChan evidence report is host-loopback only"})
	case readiness.CandidatePhysicalEvidence:
		findings = append(findings, productReadinessFinding{Code: "physical_stackchan_review_required", Message: "Physical StackChan evidence is candidate quality and requires human physical review"})
	case report.PRDAccepted && !readiness.PRDPhysicalAccepted:
		findings = append(findings, productReadinessFinding{Code: "physical_stackchan_acceptance_incomplete", Message: "Physical StackChan report claims PRD acceptance but required physical evidence is incomplete"})
	default:
		findings = append(findings, productReadinessFinding{Code: "physical_stackchan_report_blocked", Message: "Physical StackChan evidence report does not satisfy physical acceptance"})
	}
	return readiness, findings
}

func buildProductXiaozhiPhysicalReadiness(sourceReport string, report xiaozhiPhysicalEvidenceReport) productPhysicalStackChanReadiness {
	availability := productPhysicalStackChanMetricAvailability(report.CanonicalMetrics)
	micAvailable := report.Mic.Available && physicalStackChanMicAvailable(report.Mic)
	observationAvailable := xiaozhiPhysicalObservationAvailable(report.Observation)
	mode := strings.TrimSpace(report.ExecutionMode)
	gate := strings.TrimSpace(report.PromotionGate)
	status := strings.TrimSpace(report.AcceptanceStatus)
	gatewayDownlink := mode == "physical_xiaozhi_gateway" &&
		report.PhysicalDeviceOnline &&
		xiaozhiProductPhysicalProfileAccepted(report.Profile) &&
		report.GatewayMetrics.GatewayAnswerFirstDownlinkMS.Available &&
		xiaozhiPhysicalStageAvailable(report.StageAvailability, "xiaozhi.opus.decode") &&
		xiaozhiPhysicalStageAvailable(report.StageAvailability, "audio.ingress.pcm") &&
		xiaozhiPhysicalStageAvailable(report.StageAvailability, "vad.speech.end") &&
		xiaozhiPhysicalStageAvailable(report.StageAvailability, "xiaozhi.listen.auto_stop") &&
		xiaozhiPhysicalStageAvailable(report.StageAvailability, "xiaozhi.tts.downlink")
	playbackObserved := availability["device_playback_start_ms"] && availability["speech_end_to_first_audible_response_ms"] && observationAvailable
	requiredMetrics := productRequiredPhysicalStackChanMetricsAvailable(availability)
	prdAccepted := gatewayDownlink &&
		gate == "accepted" &&
		(status == "prd_accepted" || status == "accepted") &&
		report.PRDAccepted &&
		requiredMetrics &&
		micAvailable &&
		observationAvailable
	candidateVoice := gatewayDownlink &&
		((gate == "not_production" && status == "candidate_gateway_downlink") ||
			(gate == "candidate" && status == "physical_review_required" && playbackObserved)) &&
		!report.PRDAccepted
	findingCodes := productPhysicalStackChanFindingCodes(report.Findings)
	if prdAccepted {
		findingCodes = appendProductFindingCode(findingCodes, "xiaozhi_physical_prd_accepted")
	} else if candidateVoice {
		findingCodes = appendProductFindingCode(findingCodes, "xiaozhi_physical_gateway_downlink_candidate")
		if !availability["device_playback_start_ms"] {
			findingCodes = appendProductFindingCode(findingCodes, "xiaozhi_physical_device_playback_ack_missing")
		}
		if !observationAvailable {
			findingCodes = appendProductFindingCode(findingCodes, "xiaozhi_physical_operator_observation_missing")
		}
	} else {
		findingCodes = appendProductFindingCode(findingCodes, "physical_stackchan_report_blocked")
	}
	return productPhysicalStackChanReadiness{
		Valid:                                  true,
		Status:                                 firstNonEmpty(status, "blocked"),
		SourceReport:                           sourceReport,
		ExecutionMode:                          mode,
		PromotionGate:                          gate,
		AcceptanceStatus:                       status,
		PRDAccepted:                            report.PRDAccepted,
		RequiredPhysicalMetricsAvailable:       requiredMetrics,
		MicEvidenceAvailable:                   micAvailable,
		OperatorInstrumentObservationAvailable: observationAvailable,
		CandidatePhysicalEvidence:              false,
		CandidatePhysicalVoiceEvidence:         candidateVoice,
		GatewayDownlinkPhysicalDeviceEvidence:  gatewayDownlink,
		HostLoopbackOnly:                       false,
		PRDPhysicalAccepted:                    prdAccepted,
		CanonicalMetricAvailability:            availability,
		FindingCodes:                           findingCodes,
	}
}

func xiaozhiProductPhysicalProfileAccepted(profile string) bool {
	switch strings.TrimSpace(profile) {
	case "stock", "debug":
		return true
	default:
		return false
	}
}

func xiaozhiPhysicalStageAvailable(stages map[string]physicalStackChanMetric, key string) bool {
	stage, ok := stages[key]
	return ok && stage.Available
}

func buildProductPhysicalStackChanReadiness(sourceReport string, report physicalStackChanEvidenceReport) productPhysicalStackChanReadiness {
	availability := productPhysicalStackChanMetricAvailability(report.CanonicalMetrics)
	requiredMetrics := productRequiredPhysicalStackChanMetricsAvailable(availability)
	micAvailable := report.Mic.Available && physicalStackChanMicAvailable(report.Mic)
	observationAvailable := report.Observation.Available && physicalStackChanObservationAvailable(report.Observation)
	mode := strings.TrimSpace(report.ExecutionMode)
	gate := strings.TrimSpace(report.PromotionGate)
	status := strings.TrimSpace(report.AcceptanceStatus)
	hostOnly := mode != "physical_stackchan" || status == "candidate_host_only"
	candidate := mode == "physical_stackchan" &&
		gate == "candidate" &&
		status == "physical_review_required" &&
		!report.PRDAccepted &&
		requiredMetrics &&
		micAvailable &&
		observationAvailable &&
		report.Execution.HardwareExecuted
	prdAccepted := mode == "physical_stackchan" &&
		gate == "accepted" &&
		(status == "prd_accepted" || status == "accepted") &&
		report.PRDAccepted &&
		requiredMetrics &&
		micAvailable &&
		observationAvailable &&
		report.Execution.HardwareExecuted
	findingCodes := productPhysicalStackChanFindingCodes(report.Findings)
	switch {
	case prdAccepted:
		findingCodes = appendProductFindingCode(findingCodes, "physical_stackchan_prd_accepted")
	case candidate:
		findingCodes = appendProductFindingCode(findingCodes, "physical_stackchan_review_required")
	case hostOnly:
		findingCodes = appendProductFindingCode(findingCodes, "physical_stackchan_host_loopback_only")
	default:
		findingCodes = appendProductFindingCode(findingCodes, "physical_stackchan_report_blocked")
	}
	return productPhysicalStackChanReadiness{
		Valid:                                  true,
		Status:                                 firstNonEmpty(status, "blocked"),
		SourceReport:                           sourceReport,
		ExecutionMode:                          mode,
		PromotionGate:                          gate,
		AcceptanceStatus:                       status,
		PRDAccepted:                            report.PRDAccepted,
		RequiredPhysicalMetricsAvailable:       requiredMetrics,
		MicEvidenceAvailable:                   micAvailable,
		OperatorInstrumentObservationAvailable: observationAvailable,
		CandidatePhysicalEvidence:              candidate,
		HostLoopbackOnly:                       hostOnly,
		PRDPhysicalAccepted:                    prdAccepted,
		CanonicalMetricAvailability:            availability,
		FindingCodes:                           findingCodes,
	}
}

func productPhysicalStackChanMetricAvailability(metrics physicalStackChanCanonicalMetrics) map[string]bool {
	return map[string]bool{
		"device_downlink_first_frame_ms":          metrics.DeviceDownlinkFirstFrameMS.Available,
		"device_playback_start_ms":                metrics.DevicePlaybackStartMS.Available,
		"speech_end_to_first_audible_response_ms": metrics.SpeechEndToFirstAudibleResponseMS.Available,
		"barge_in_detected_ms":                    metrics.BargeInDetectedMS.Available,
		"barge_in_stop_ms":                        metrics.BargeInStopMS.Available,
		"barge_in_playback_stop_requested_ms":     metrics.BargeInPlaybackStopRequestedMS.Available,
		"barge_in_playback_stop_done_ms":          metrics.BargeInPlaybackStopDoneMS.Available,
	}
}

func productRequiredPhysicalStackChanMetricsAvailable(availability map[string]bool) bool {
	for _, key := range []string{
		"device_downlink_first_frame_ms",
		"device_playback_start_ms",
		"speech_end_to_first_audible_response_ms",
		"barge_in_detected_ms",
		"barge_in_stop_ms",
		"barge_in_playback_stop_requested_ms",
		"barge_in_playback_stop_done_ms",
	} {
		if !availability[key] {
			return false
		}
	}
	return true
}

func productPhysicalStackChanRedactionOK(redaction physicalStackChanEvidenceRedaction) bool {
	return !redaction.UserTextStored &&
		!redaction.InstructionTextStored &&
		!redaction.ModelTextStored &&
		!redaction.AudioPayloadStored &&
		!redaction.EncodedAudioPayloadStored &&
		!redaction.NetworkLocatorStored &&
		!redaction.NetworkRouteStored &&
		!redaction.FilesystemLocatorStored &&
		!redaction.SecretMaterialStored &&
		!redaction.InternalThoughtStored
}

func productPhysicalStackChanFindingCodes(findings []physicalStackChanEvidenceFinding) []string {
	var codes []string
	for _, finding := range findings {
		code := strings.TrimSpace(finding.Code)
		if code != "" {
			codes = appendProductFindingCode(codes, code)
		}
	}
	return codes
}

func appendProductFindingCode(codes []string, code string) []string {
	for _, existing := range codes {
		if existing == code {
			return codes
		}
	}
	return append(codes, code)
}

func invalidProductPhysicalStackChanReportFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "physical_stackchan_report_invalid",
		Message: "Physical StackChan evidence report is invalid or unsafe",
	}
}

type productV21ProfessionalReportFixture struct {
	SchemaVersion                string  `json:"schema_version"`
	Status                       string  `json:"status"`
	CheckingAckWithin1200        *bool   `json:"checking_ack_within_1200"`
	EvidenceAvailable            *bool   `json:"evidence_available"`
	CardsAvailable               *bool   `json:"cards_available"`
	FollowUpsAvailable           *bool   `json:"follow_ups_available"`
	EvidenceCount                *int    `json:"evidence_count"`
	CardCount                    *int    `json:"card_count"`
	FollowUpCount                *int    `json:"follow_up_count"`
	AdapterExecuted              *bool   `json:"adapter_executed"`
	RedactionOK                  *bool   `json:"redaction_ok"`
	ProfessionalAcceptanceStatus *string `json:"professional_acceptance_status"`
}
