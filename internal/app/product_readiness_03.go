package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
)

func missingProductProviderSmokeReportField(fixture productProviderSmokeReportFixture) string {
	switch {
	case strings.TrimSpace(fixture.SchemaVersion) == "":
		return "schema_version"
	case fixture.GeneratedAtMS == nil:
		return "generated_at_ms"
	case strings.TrimSpace(fixture.Provider) == "":
		return "provider"
	case strings.TrimSpace(fixture.Family) == "":
		return "family"
	case strings.TrimSpace(fixture.Protocol) == "":
		return "protocol"
	case strings.TrimSpace(fixture.Status) == "":
		return "status"
	case fixture.Configured == nil:
		return "configured"
	case fixture.Executed == nil:
		return "executed"
	case fixture.RouteEligible == nil:
		return "route_eligible"
	case fixture.Stream == nil:
		return "stream"
	case fixture.TimingSummary == nil:
		return "timing_summary"
	default:
		return ""
	}
}

func missingProductProviderSmokeReportFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "provider_smoke_report_missing_field",
		Message: "Provider smoke report is missing a required field",
		Detail:  field,
	}
}

func invalidProductProviderSmokeReportFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "provider_smoke_report_invalid",
		Message: "Provider smoke report is invalid, unsafe, not executed, or not a real route-eligible text provider",
	}
}

func validProductProviderSmokeProtocol(protocol string) bool {
	switch protocol {
	case "openai_chat_completions", "ollama_chat":
		return true
	default:
		return false
	}
}

func validProductProviderSmokeStreamingEvidence(fixture productProviderSmokeReportFixture) bool {
	if fixture.Stream == nil || !*fixture.Stream || fixture.Repeat < 3 || len(fixture.Attempts) != fixture.Repeat || fixture.TimingSummary == nil {
		return false
	}
	if fixture.TimingSummary.Repeat != fixture.Repeat ||
		fixture.TimingSummary.FirstByteP95MS <= 0 ||
		fixture.TimingSummary.FirstByteP99MS <= 0 ||
		fixture.TimingSummary.FirstContentP95MS <= 0 ||
		fixture.TimingSummary.FirstContentP99MS <= 0 ||
		fixture.TimingSummary.TotalDurationP95MS <= 0 ||
		fixture.TimingSummary.TotalDurationP99MS <= 0 {
		return false
	}
	for _, attempt := range fixture.Attempts {
		if attempt.Index <= 0 ||
			attempt.HTTPStatus < 200 ||
			attempt.HTTPStatus >= 300 ||
			attempt.FirstByteMS <= 0 ||
			attempt.FirstContentMS <= 0 ||
			attempt.TotalDurationMS <= 0 ||
			attempt.ContentDeltaCount <= 0 ||
			!attempt.Done {
			return false
		}
	}
	return true
}

func productProviderSmokeFallbackObserved(fixture productProviderSmokeReportFixture) bool {
	for _, marker := range fixture.TraceMarkers {
		if strings.TrimSpace(marker.Name) == "provider_fallback_used" {
			return true
		}
	}
	for _, metric := range fixture.Metrics {
		if strings.TrimSpace(metric.Name) == "a21_provider_fallback_total" && metric.Value > 0 {
			return true
		}
	}
	return false
}

func productProviderSmokeEnvNamesSafe(values ...string) bool {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len(value) > 80 || !strings.HasPrefix(value, "A21_") || containsLegacyIdentity(value) {
			return false
		}
		for _, r := range value {
			if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
				continue
			}
			return false
		}
	}
	return true
}

func productProviderSmokeEndpointHostUnsafe(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return true
	}
	for _, forbidden := range []string{"http://", "https://", "@", "/users/", "key", "token", "secret"} {
		if strings.Contains(host, forbidden) {
			return true
		}
	}
	return containsLegacyIdentity(host)
}

func productProviderSmokeReportContainsForbiddenValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			if productProviderSmokeReportContainsForbiddenValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productProviderSmokeReportContainsForbiddenValue(child) {
				return true
			}
		}
	case string:
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
			"raw transcript",
			"transcript text",
			"raw provider output",
			"provider output",
			"raw reasoning",
			"reasoning text",
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

func productTextStreamProviderReady(env []string, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	catalog := providers.ProviderCatalogFromEnv(env)
	for _, provider := range catalog.Providers {
		if provider.Name == name {
			return provider.Configured &&
				provider.RouteEligible &&
				providers.ProviderFamily(provider.Family) == providers.ProviderFamilyTextStream
		}
	}
	return false
}

func buildProductV21Readiness(env []string) productV21Readiness {
	adapterURL := appEnvValue(env, "A21_V21_ADAPTER_URL")
	bridge := v21adapter.NewProfessionalBridgeReadiness(adapterURL)
	v21 := buildV21DoctorReport(adapterURL)
	return productV21Readiness{
		Configured:                v21.Configured,
		Healthy:                   v21.Healthy,
		Status:                    v21.Status,
		ProfessionalBridgeReady:   bridge.ContractReady,
		CheckingFeedbackSupported: bridge.CheckingFeedbackSupported,
		MaxFirstResponseMS:        bridge.MaxFirstResponseMS,
		EvidenceContractReady:     bridge.EvidenceContractReady,
		QueryExecuted:             bridge.QueryExecuted,
		QueryPath:                 bridge.QueryPath,
		HealthPath:                bridge.HealthPath,
	}
}

func fetchProductWakeWordReadiness(ctx context.Context, gatewayURL string) (productWakeWordReadiness, []productReadinessFinding) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayURL, "/v1/wake-word", nil)
	if err != nil {
		return productWakeWordUnavailable(), []productReadinessFinding{{
			Code:    "wake_word_status_unavailable",
			Message: "A21 Gateway wake-word status is unavailable",
			Detail:  "invalid_gateway_url",
		}}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return productWakeWordUnavailable(), []productReadinessFinding{{
			Code:    "wake_word_status_unavailable",
			Message: "A21 Gateway wake-word status is unavailable",
			Detail:  "invalid_request",
		}}
	}
	client := *a21DirectHTTPClient(700 * time.Millisecond)
	response, err := client.Do(request)
	if err != nil {
		return productWakeWordUnavailable(), []productReadinessFinding{{
			Code:    "wake_word_status_unavailable",
			Message: "A21 Gateway wake-word status is unavailable",
			Detail:  "request_failed",
		}}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return productWakeWordUnavailable(), []productReadinessFinding{{
			Code:    "wake_word_status_unavailable",
			Message: "A21 Gateway wake-word status is unavailable",
			Detail:  "status_not_ok",
		}}
	}
	var status gateway.WakeWordConfigResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4096))
	if err := decoder.Decode(&status); err != nil {
		return productWakeWordUnavailable(), []productReadinessFinding{{
			Code:    "wake_word_status_invalid",
			Message: "A21 Gateway wake-word status is invalid",
			Detail:  "decode_failed",
		}}
	}
	if !validProductWakeWordStatus(status) {
		return productWakeWordUnavailable(), []productReadinessFinding{{
			Code:    "wake_word_status_invalid",
			Message: "A21 Gateway wake-word status is invalid",
			Detail:  "invalid_fields",
		}}
	}
	readiness := productWakeWordReadiness{
		Available:             true,
		ProductReady:          productWakeWordStatusReady(status),
		SchemaVersion:         status.SchemaVersion,
		Mode:                  status.Mode,
		ActivePhrase:          status.ActivePhrase,
		ActivePinyin:          status.ActivePinyin,
		DesiredPhrase:         status.DesiredPhrase,
		DesiredPinyin:         status.DesiredPinyin,
		Threshold:             status.Threshold,
		RuntimeStatus:         status.RuntimeStatus,
		RuntimeConfigurable:   status.RuntimeConfigurable,
		FirmwareBuildRequired: status.FirmwareBuildRequired,
		Code:                  status.Code,
	}
	if status.FirmwareBuildRequired {
		return readiness, []productReadinessFinding{{
			Code:    "wake_word_firmware_build_required",
			Message: "Custom wake word config is pending a guarded firmware build and flash",
			Detail:  status.Mode,
		}}
	}
	return readiness, nil
}

func productWakeWordUnavailable() productWakeWordReadiness {
	return productWakeWordReadiness{
		Available:     false,
		ProductReady:  false,
		Mode:          "unknown",
		RuntimeStatus: "unavailable",
	}
}

func validProductWakeWordStatus(status gateway.WakeWordConfigResponse) bool {
	if strings.TrimSpace(status.SchemaVersion) != "a21.gateway.wake_word.v1" {
		return false
	}
	for _, value := range []string{
		status.SchemaVersion,
		status.Mode,
		status.ActivePhrase,
		status.ActivePinyin,
		status.DesiredPhrase,
		status.DesiredPinyin,
		status.RuntimeStatus,
		status.Code,
	} {
		lower := strings.ToLower(value)
		if containsLegacyIdentity(value) || strings.Contains(lower, "http://") || strings.Contains(lower, "https://") {
			return false
		}
	}
	return strings.TrimSpace(status.Mode) != "" && strings.TrimSpace(status.RuntimeStatus) != ""
}

func productWakeWordStatusReady(status gateway.WakeWordConfigResponse) bool {
	if status.FirmwareBuildRequired {
		return false
	}
	return strings.TrimSpace(status.RuntimeStatus) != "" && strings.TrimSpace(status.RuntimeStatus) != "unavailable"
}

func productWakeWordPhysicalAccepted(readiness productWakeWordReadiness) bool {
	return readiness.PhysicalAcceptanceAvailable &&
		strings.TrimSpace(readiness.PhysicalAcceptanceStatus) == "accepted" &&
		readiness.PhysicalDeviceOnline &&
		readiness.PhysicalFirmwareFlashed &&
		readiness.PhysicalOperatorObserved &&
		readiness.PhysicalWakePhraseMatched
}

const productLaunchPolicyExpectedLLMProfile = "stepfun"

func fetchProductVoiceChainReadiness(ctx context.Context, gatewayURL string) (productVoiceChainReadiness, []productReadinessFinding) {
	unavailable := productVoiceChainReadiness{
		Available:                      false,
		Status:                         "unavailable",
		LaunchPolicyExpectedLLMProfile: productLaunchPolicyExpectedLLMProfile,
		LaunchPolicySatisfied:          false,
	}
	endpoint, _, err := firmwareGatewayEndpoint(gatewayURL, "/v1/voice-chain-profiles", nil)
	if err != nil {
		return unavailable, []productReadinessFinding{{
			Code:    "voice_chain_status_unavailable",
			Message: "A21 Gateway voice-chain selector status is unavailable",
			Detail:  "invalid_gateway_url",
		}}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return unavailable, []productReadinessFinding{{
			Code:    "voice_chain_status_unavailable",
			Message: "A21 Gateway voice-chain selector status is unavailable",
			Detail:  "invalid_request",
		}}
	}
	client := *a21DirectHTTPClient(700 * time.Millisecond)
	response, err := client.Do(request)
	if err != nil {
		return unavailable, nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return unavailable, nil
	}
	var status gateway.VoiceChainProfilesResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, 8192))
	if err := decoder.Decode(&status); err != nil {
		return unavailable, []productReadinessFinding{{
			Code:    "voice_chain_status_invalid",
			Message: "A21 Gateway voice-chain selector status is invalid",
			Detail:  "decode_failed",
		}}
	}
	readiness, ok := productVoiceChainReadinessFromGateway(status)
	if !ok {
		return unavailable, []productReadinessFinding{{
			Code:    "voice_chain_status_invalid",
			Message: "A21 Gateway voice-chain selector status is invalid",
			Detail:  "invalid_fields",
		}}
	}
	var findings []productReadinessFinding
	if productVoiceChainHasFinding(readiness, "stepfun_not_selected") {
		findings = append(findings, productReadinessFinding{
			Code:    "stepfun_not_selected",
			Message: "Launch policy expects StepFun for the selected cascade LLM profile",
		})
	}
	return readiness, findings
}

func productVoiceChainReadinessFromGateway(status gateway.VoiceChainProfilesResponse) (productVoiceChainReadiness, bool) {
	if strings.TrimSpace(status.SchemaVersion) != gateway.VoiceChainProfileSchemaVersion {
		return productVoiceChainReadiness{}, false
	}
	selectedMode := strings.TrimSpace(status.SelectedVoiceChainMode)
	selectedASR := strings.TrimSpace(status.SelectedASRProfile)
	selectedLLM := strings.TrimSpace(status.SelectedLLMProfile)
	fixedTTS := strings.TrimSpace(status.FixedTTSProfile)
	selectedTTS := strings.TrimSpace(status.SelectedTTSProfile)
	selectedRealtime := strings.TrimSpace(status.SelectedRealtimeProvider)
	selectedVoiceClone := strings.TrimSpace(status.SelectedVoiceCloneProfile)
	if !productVoiceChainSafeID(selectedMode) ||
		!productVoiceChainSafeID(selectedASR) ||
		!productVoiceChainSafeID(selectedLLM) ||
		!productVoiceChainSafeID(fixedTTS) ||
		!productVoiceChainSafeID(selectedTTS) ||
		!productVoiceChainSafeOptionalID(selectedRealtime) ||
		!productVoiceChainSafeOptionalID(selectedVoiceClone) {
		return productVoiceChainReadiness{}, false
	}
	findings := productVoiceChainSafeFindings(status.Findings)
	if selectedLLM != productLaunchPolicyExpectedLLMProfile {
		findings = appendProductFindingCode(findings, "stepfun_not_selected")
	}
	satisfied := !productVoiceChainHasFinding(productVoiceChainReadiness{Findings: findings}, "stepfun_not_selected") &&
		selectedLLM == productLaunchPolicyExpectedLLMProfile
	readinessStatus := "ready"
	if !satisfied {
		readinessStatus = "launch_policy_blocked"
	}
	return productVoiceChainReadiness{
		Available:                      true,
		Status:                         readinessStatus,
		SelectedVoiceChainMode:         selectedMode,
		SelectedASRProfile:             selectedASR,
		SelectedLLMProfile:             selectedLLM,
		FixedTTSProfile:                fixedTTS,
		SelectedTTSProfile:             selectedTTS,
		SelectedRealtimeProvider:       selectedRealtime,
		SelectedVoiceCloneProfile:      selectedVoiceClone,
		HotSwitch:                      status.HotSwitch,
		LaunchPolicyExpectedLLMProfile: productLaunchPolicyExpectedLLMProfile,
		LaunchPolicySatisfied:          satisfied,
		Findings:                       findings,
	}, true
}

func productVoiceChainSafeID(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && providerLatencySafeIdentifier(value, false) == value
}

func productVoiceChainSafeOptionalID(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || productVoiceChainSafeID(value)
}

func productVoiceChainSafeFindings(values []string) []string {
	var findings []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if productVoiceChainSafeID(value) {
			findings = appendProductFindingCode(findings, value)
		}
	}
	return findings
}

func productVoiceChainHasFinding(readiness productVoiceChainReadiness, code string) bool {
	return productStringSliceContains(readiness.Findings, code)
}

func productVoiceChainLaunchPolicySatisfied(readiness productVoiceChainReadiness) bool {
	return readiness.Available && readiness.LaunchPolicySatisfied
}

type productVoiceChainCapabilityEvidence struct {
	Valid         bool
	SourceReport  string
	GateStatus    string
	ExecutionMode string
	ASRProfile    string
	LLMProfile    string
	TTSProfile    string
	ASRReady      bool
	LLMReady      bool
	TTSReady      bool
	Findings      []string
}

func loadProductVoiceChainReadinessReportEvidence(path string) (productVoiceChainCapabilityEvidence, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productVoiceChainCapabilityEvidence{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productVoiceChainCapabilityEvidence{}, []productReadinessFinding{invalidProductVoiceChainReadinessReportFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productVoiceChainCapabilityEvidence{}, []productReadinessFinding{invalidProductVoiceChainReadinessReportFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productVoiceChainCapabilityEvidence{}, []productReadinessFinding{invalidProductVoiceChainReadinessReportFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productVoiceChainCapabilityEvidence{}, []productReadinessFinding{invalidProductVoiceChainReadinessReportFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productVoiceChainReadinessReportContainsForbiddenValue(raw) {
		return productVoiceChainCapabilityEvidence{}, []productReadinessFinding{invalidProductVoiceChainReadinessReportFinding()}
	}
	var report xiaozhiStreamingProviderReadinessReport
	if err := json.Unmarshal(data, &report); err != nil {
		return productVoiceChainCapabilityEvidence{}, []productReadinessFinding{invalidProductVoiceChainReadinessReportFinding()}
	}
	if missingField := missingProductVoiceChainReadinessReportField(report); missingField != "" {
		return productVoiceChainCapabilityEvidence{}, []productReadinessFinding{missingProductVoiceChainReadinessReportFieldFinding(missingField)}
	}
	asrProfile := providerLatencySafeIdentifier(report.ASR.Profile, false)
	llmProfile := providerLatencySafeIdentifier(report.LLM.Profile, false)
	ttsProfile := providerLatencySafeIdentifier(report.TTS.Profile, false)
	if report.SchemaVersion != xiaozhiStreamingProviderReadinessSchemaVersion ||
		report.GeneratedAtMS <= 0 ||
		report.ExecutionMode != "static_no_execute_provider_capability_gate" ||
		report.ProductMode != "dialogue" ||
		report.ChainMode != "dialogue_low_latency" ||
		report.ProfessionalBoundary != "v21_adapter_only" ||
		!validProductVoiceChainReadinessGateStatus(report.GateStatus) ||
		report.PRDAccepted ||
		!productVoiceChainReadinessRedactionOK(report.Redaction) ||
		!productVoiceChainReadinessStageSafe(report.ASR) ||
		!productVoiceChainReadinessStageSafe(report.LLM) ||
		!productVoiceChainReadinessStageSafe(report.TTS) ||
		!productVoiceChainSelectionSafe(report.Selection) ||
		asrProfile == "" ||
		llmProfile == "" ||
		ttsProfile == "" ||
		(report.ReportPath != "" && !productVoiceChainReadinessReportPathSafe(report.ReportPath)) {
		return productVoiceChainCapabilityEvidence{}, []productReadinessFinding{invalidProductVoiceChainReadinessReportFinding()}
	}
	return productVoiceChainCapabilityEvidence{
		Valid:         true,
		SourceReport:  filepath.Base(filepath.Clean(path)),
		GateStatus:    strings.TrimSpace(report.GateStatus),
		ExecutionMode: strings.TrimSpace(report.ExecutionMode),
		ASRProfile:    asrProfile,
		LLMProfile:    llmProfile,
		TTSProfile:    ttsProfile,
		ASRReady:      report.ASR.Ready && report.ASR.RealProvider && report.ASR.Streaming && report.ASR.ImplementedInGateway,
		LLMReady:      report.LLM.Ready && report.LLM.RealProvider && report.LLM.Streaming && report.LLM.ImplementedInGateway,
		TTSReady:      report.TTS.Ready && report.TTS.RealProvider && report.TTS.Streaming && report.TTS.ImplementedInGateway,
		Findings:      productVoiceChainSafeFindings(report.Findings),
	}, nil
}

func attachProductVoiceChainCapabilityEvidence(readiness *productVoiceChainReadiness, evidence productVoiceChainCapabilityEvidence) []productReadinessFinding {
	if readiness == nil || !evidence.Valid {
		return nil
	}
	readiness.CapabilityEvidenceAvailable = true
	readiness.CapabilityEvidenceMatched = false
	readiness.CapabilityEvidenceStatus = evidence.GateStatus
	readiness.CapabilityEvidenceSourceReport = evidence.SourceReport
	readiness.CapabilityExecutionMode = evidence.ExecutionMode
	readiness.CapabilityFindings = append([]string(nil), evidence.Findings...)
	readiness.ASRStreamingReady = evidence.ASRReady
	readiness.LLMStreamingReady = evidence.LLMReady
	readiness.TTSStreamingReady = evidence.TTSReady
	if !readiness.Available {
		readiness.Findings = appendProductFindingCode(readiness.Findings, "voice_chain_selector_unavailable")
		return []productReadinessFinding{{
			Code:    "voice_chain_selector_unavailable",
			Message: "Voice-chain capability report could not be matched because the Gateway selector is unavailable",
			Detail:  evidence.SourceReport,
		}}
	}
	if !productVoiceChainCapabilityEvidenceMatchesSelected(*readiness, evidence) {
		readiness.Findings = appendProductFindingCode(readiness.Findings, "voice_chain_capability_report_mismatch")
		return []productReadinessFinding{{
			Code:    "voice_chain_capability_report_mismatch",
			Message: "Voice-chain capability report does not match the selected A21 Gateway voice-chain profiles",
			Detail:  evidence.SourceReport,
		}}
	}
	readiness.CapabilityEvidenceMatched = true
	readiness.StaticCapabilityReady = evidence.GateStatus == "passed" && evidence.ASRReady && evidence.LLMReady && evidence.TTSReady
	return nil
}

func productVoiceChainCapabilityEvidenceMatchesSelected(readiness productVoiceChainReadiness, evidence productVoiceChainCapabilityEvidence) bool {
	selectedTTS := firstNonEmpty(readiness.SelectedTTSProfile, readiness.FixedTTSProfile)
	return evidence.Valid &&
		evidence.ASRProfile == readiness.SelectedASRProfile &&
		evidence.LLMProfile == readiness.SelectedLLMProfile &&
		(evidence.TTSProfile == selectedTTS || evidence.TTSProfile == readiness.FixedTTSProfile)
}

func missingProductVoiceChainReadinessReportField(report xiaozhiStreamingProviderReadinessReport) string {
	switch {
	case strings.TrimSpace(report.SchemaVersion) == "":
		return "schema_version"
	case report.GeneratedAtMS <= 0:
		return "generated_at_ms"
	case strings.TrimSpace(report.ExecutionMode) == "":
		return "execution_mode"
	case strings.TrimSpace(report.ProductMode) == "":
		return "product_mode"
	case strings.TrimSpace(report.ChainMode) == "":
		return "chain_mode"
	case strings.TrimSpace(report.ProfessionalBoundary) == "":
		return "professional_boundary"
	case strings.TrimSpace(report.ASR.Profile) == "":
		return "asr.profile"
	case strings.TrimSpace(report.LLM.Profile) == "":
		return "llm.profile"
	case strings.TrimSpace(report.TTS.Profile) == "":
		return "tts.profile"
	case strings.TrimSpace(report.GateStatus) == "":
		return "gate_status"
	default:
		return ""
	}
}

func missingProductVoiceChainReadinessReportFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "voice_chain_readiness_report_missing_field",
		Message: "Voice-chain readiness report is missing a required field",
		Detail:  field,
	}
}

func invalidProductVoiceChainReadinessReportFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "voice_chain_readiness_report_invalid",
		Message: "Voice-chain readiness report is invalid or unsafe",
	}
}

func validProductVoiceChainReadinessGateStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "passed", "blocked":
		return true
	default:
		return false
	}
}

func productVoiceChainReadinessRedactionOK(redaction xiaozhiVoiceBenchRedaction) bool {
	return !redaction.PayloadsStored &&
		!redaction.CredentialValuesStored &&
		!redaction.FullURLsStored &&
		!redaction.LocalPathsStored
}

func productVoiceChainReadinessStageSafe(stage xiaozhiStreamingProviderReadinessStage) bool {
	return productVoiceChainSafeID(stage.Profile) &&
		productVoiceChainSafeOptionalID(stage.ProfileEnv) &&
		productVoiceChainSafeOptionalID(stage.SelectionRole) &&
		productVoiceChainSafeID(stage.Adapter) &&
		productVoiceChainSafeID(stage.Capability) &&
		productProviderSmokeEnvNamesSafe(stage.RequiredEnv...) &&
		productProviderSmokeEnvNamesSafe(stage.PresentEnv...) &&
		productProviderSmokeEnvNamesSafe(stage.MissingEnv...)
}

func productVoiceChainSelectionSafe(selection providers.VoicePipelineSelection) bool {
	return productVoiceChainSafeOptionalID(selection.ASRMode) &&
		productVoiceChainSafeID(selection.ASRProfile) &&
		productVoiceChainSafeOptionalID(selection.ASRProfileEnv) &&
		productVoiceChainSafeID(selection.LLMProfile) &&
		productVoiceChainSafeOptionalID(selection.LLMProfileEnv) &&
		productVoiceChainSafeOptionalID(selection.LLMFallbackProfile) &&
		productVoiceChainSafeOptionalID(selection.LLMFallbackProfileEnv) &&
		productVoiceChainSafeOptionalID(selection.TTSMode) &&
		productVoiceChainSafeID(selection.TTSProfile) &&
		productVoiceChainSafeOptionalID(selection.TTSProfileEnv)
}
