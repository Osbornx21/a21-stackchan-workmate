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
	"a21.local/a21/internal/v21adapter"
)

func loadProductWakeWordPhysicalAcceptanceEvidence(path string) (productWakeWordPhysicalAcceptanceEvidence, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productWakeWordPhysicalAcceptanceEvidence{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productWakeWordPhysicalAcceptanceEvidence{}, []productReadinessFinding{invalidProductWakeWordPhysicalAcceptanceFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productWakeWordPhysicalAcceptanceEvidence{}, []productReadinessFinding{invalidProductWakeWordPhysicalAcceptanceFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productWakeWordPhysicalAcceptanceEvidence{}, []productReadinessFinding{invalidProductWakeWordPhysicalAcceptanceFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productWakeWordPhysicalAcceptanceEvidence{}, []productReadinessFinding{invalidProductWakeWordPhysicalAcceptanceFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productWakeWordPhysicalAcceptanceContainsForbiddenValue(raw) {
		return productWakeWordPhysicalAcceptanceEvidence{}, []productReadinessFinding{invalidProductWakeWordPhysicalAcceptanceFinding()}
	}
	var fixture productWakeWordPhysicalAcceptanceFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productWakeWordPhysicalAcceptanceEvidence{}, []productReadinessFinding{invalidProductWakeWordPhysicalAcceptanceFinding()}
	}
	if missingField := missingProductWakeWordPhysicalAcceptanceField(fixture); missingField != "" {
		return productWakeWordPhysicalAcceptanceEvidence{}, []productReadinessFinding{missingProductWakeWordPhysicalAcceptanceFieldFinding(missingField)}
	}
	if !validProductWakeWordPhysicalAcceptanceFixture(fixture) {
		return productWakeWordPhysicalAcceptanceEvidence{}, []productReadinessFinding{invalidProductWakeWordPhysicalAcceptanceFinding()}
	}
	return productWakeWordPhysicalAcceptanceEvidence{
		Valid:                                true,
		Status:                               strings.TrimSpace(fixture.Status),
		ProductReady:                         *fixture.ProductReady,
		SourceReport:                         filepath.Base(filepath.Clean(path)),
		Mode:                                 strings.TrimSpace(fixture.Mode),
		DesiredPhrase:                        strings.TrimSpace(fixture.DesiredPhrase),
		DesiredPinyin:                        strings.TrimSpace(fixture.DesiredPinyin),
		Threshold:                            *fixture.Threshold,
		PackageSourceReport:                  strings.TrimSpace(fixture.PackageSourceReport),
		PackageArtifactName:                  strings.TrimSpace(fixture.PackageArtifactName),
		PackageManifestName:                  strings.TrimSpace(fixture.PackageManifestName),
		PhysicalDeviceOnline:                 *fixture.PhysicalDeviceOnline,
		FirmwareFlashExecuted:                *fixture.FirmwareFlashExecuted,
		GuardedFlashReportSource:             strings.TrimSpace(fixture.GuardedFlashReportSource),
		OperatorCustomWakeObservationPresent: *fixture.OperatorCustomWakeObservationPresent,
		WakePhraseMatched:                    *fixture.WakePhraseMatched,
		FalseWakeAccepted:                    *fixture.FalseWakeAccepted,
		StockWakeAccepted:                    *fixture.StockWakeAccepted,
		RedactionOK:                          *fixture.RedactionOK,
	}, nil
}

func attachProductWakeWordPhysicalAcceptance(readiness *productWakeWordReadiness, packageEvidence productWakeWordFirmwarePackageEvidence, evidence productWakeWordPhysicalAcceptanceEvidence) []productReadinessFinding {
	if !matchingProductWakeWordPhysicalAcceptance(*readiness, packageEvidence, evidence) {
		return []productReadinessFinding{{
			Code:    "wake_word_physical_acceptance_mismatch",
			Message: "Wake word physical acceptance report does not match the current Gateway wake-word intent and firmware package",
			Detail:  evidence.SourceReport,
		}}
	}
	readiness.ProductReady = true
	readiness.PhysicalAcceptanceAvailable = true
	readiness.PhysicalAcceptanceStatus = evidence.Status
	readiness.PhysicalAcceptanceSource = evidence.SourceReport
	readiness.PhysicalAcceptanceFlashReport = evidence.GuardedFlashReportSource
	readiness.PhysicalDeviceOnline = evidence.PhysicalDeviceOnline
	readiness.PhysicalFirmwareFlashed = evidence.FirmwareFlashExecuted
	readiness.PhysicalOperatorObserved = evidence.OperatorCustomWakeObservationPresent
	readiness.PhysicalWakePhraseMatched = evidence.WakePhraseMatched
	return []productReadinessFinding{{
		Code:    "wake_word_physical_acceptance_available",
		Message: "Wake word physical acceptance evidence is available for the current custom MultiNet package",
		Detail:  evidence.SourceReport,
	}}
}

func matchingProductWakeWordPhysicalAcceptance(readiness productWakeWordReadiness, packageEvidence productWakeWordFirmwarePackageEvidence, evidence productWakeWordPhysicalAcceptanceEvidence) bool {
	return readiness.Available &&
		packageEvidence.Valid &&
		matchingProductWakeWordFirmwarePackage(readiness, packageEvidence) &&
		evidence.Valid &&
		evidence.ProductReady &&
		strings.TrimSpace(readiness.Mode) == evidence.Mode &&
		strings.TrimSpace(readiness.DesiredPhrase) == evidence.DesiredPhrase &&
		strings.TrimSpace(readiness.DesiredPinyin) == evidence.DesiredPinyin &&
		readiness.Threshold == evidence.Threshold &&
		evidence.PackageSourceReport == packageEvidence.SourceReport &&
		evidence.PackageArtifactName == packageEvidence.ArtifactName &&
		evidence.PackageManifestName == packageEvidence.ManifestName &&
		evidence.PhysicalDeviceOnline &&
		evidence.FirmwareFlashExecuted &&
		evidence.OperatorCustomWakeObservationPresent &&
		evidence.WakePhraseMatched &&
		!evidence.FalseWakeAccepted &&
		!evidence.StockWakeAccepted &&
		evidence.RedactionOK
}

func validProductWakeWordPhysicalAcceptanceFixture(fixture productWakeWordPhysicalAcceptanceFixture) bool {
	if fixture.SchemaVersion != wakeWordPhysicalAcceptanceSchema ||
		*fixture.GeneratedAtMS <= 0 ||
		strings.TrimSpace(fixture.Status) != "accepted" ||
		!*fixture.ProductReady ||
		strings.TrimSpace(fixture.Mode) != wakeWordFirmwareCustomMode ||
		strings.TrimSpace(fixture.DesiredPhrase) == "" ||
		strings.TrimSpace(fixture.DesiredPinyin) == "" ||
		*fixture.Threshold < 1 ||
		*fixture.Threshold > 100 ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.PackageSourceReport, ".json") ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.PackageArtifactName, ".bin") ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.PackageManifestName, ".json") ||
		!*fixture.PhysicalDeviceOnline ||
		!*fixture.FirmwareFlashExecuted ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.GuardedFlashReportSource, ".json") ||
		!*fixture.OperatorCustomWakeObservationPresent ||
		!*fixture.WakePhraseMatched ||
		*fixture.FalseWakeAccepted ||
		*fixture.StockWakeAccepted ||
		!*fixture.RedactionOK ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.ReportPath, ".json") {
		return false
	}
	return true
}

func missingProductWakeWordPhysicalAcceptanceField(fixture productWakeWordPhysicalAcceptanceFixture) string {
	switch {
	case fixture.GeneratedAtMS == nil:
		return "generated_at_ms"
	case fixture.ProductReady == nil:
		return "product_ready"
	case fixture.Threshold == nil:
		return "threshold"
	case fixture.PhysicalDeviceOnline == nil:
		return "physical_device_online"
	case fixture.FirmwareFlashExecuted == nil:
		return "firmware_flash_executed"
	case fixture.OperatorCustomWakeObservationPresent == nil:
		return "operator_custom_wake_observation_present"
	case fixture.WakePhraseMatched == nil:
		return "wake_phrase_matched"
	case fixture.FalseWakeAccepted == nil:
		return "false_wake_accepted"
	case fixture.StockWakeAccepted == nil:
		return "stock_wake_accepted"
	case fixture.RedactionOK == nil:
		return "redaction_ok"
	default:
		return ""
	}
}

func missingProductWakeWordPhysicalAcceptanceFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "wake_word_physical_acceptance_missing_field",
		Message: "Wake word physical acceptance report is missing a required field",
		Detail:  field,
	}
}

func invalidProductWakeWordPhysicalAcceptanceFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "wake_word_physical_acceptance_invalid",
		Message: "Wake word physical acceptance report is invalid or unsafe",
	}
}

func productWakeWordPhysicalAcceptanceContainsForbiddenValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			if productWakeWordPhysicalAcceptanceContainsForbiddenValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productWakeWordPhysicalAcceptanceContainsForbiddenValue(child) {
				return true
			}
		}
	case string:
		lower := strings.ToLower(typed)
		if containsLegacyIdentity(typed) {
			return true
		}
		for _, forbidden := range []string{"http://", "https://", "/users/", "secret", "token", "proxy", "transcript", "provider output", "raw audio", "base64"} {
			if strings.Contains(lower, forbidden) {
				return true
			}
		}
	}
	return false
}

func buildProductStackChanReadiness(deviceReport firmwareDeviceReport, deviceID string) productStackChanReadiness {
	readiness := productStackChanReadiness{
		DeviceID:               deviceID,
		MaxPhysicalDeviceAgeMS: productPhysicalDeviceFreshMaxAgeMS,
	}
	for _, device := range deviceReport.Devices {
		online := device.ConnectionStatus == "" || device.ConnectionStatus == "online"
		isPhysicalTarget := device.DeviceID == deviceID && !strings.Contains(strings.ToLower(device.DeviceID), "sim")
		if isPhysicalTarget && device.DeviceAgeMS > 0 {
			readiness.PhysicalDeviceAgeMS = device.DeviceAgeMS
		}
		fresh := device.DeviceAgeMS <= 0 || device.DeviceAgeMS <= productPhysicalDeviceFreshMaxAgeMS
		if isPhysicalTarget && online && !fresh {
			readiness.PhysicalDeviceStale = true
			readiness.MicrophoneStatus = strings.TrimSpace(device.Capabilities["microphone"])
			continue
		}
		if isPhysicalTarget && online && fresh {
			readiness.PhysicalDeviceOnline = true
			readiness.MicrophoneStatus = strings.TrimSpace(device.Capabilities["microphone"])
			readiness.PhysicalMicrophoneReady = productMicrophoneReady(readiness.MicrophoneStatus)
		}
		if online && strings.Contains(strings.ToLower(device.DeviceID), "sim") {
			readiness.SimulatorDeviceOnline = true
		}
	}
	if serialDevices, err := listFirmwareSerialDevices(); err == nil {
		readiness.USBSerialCandidateCount = len(officeUSBSerialCandidates(serialDevices))
	}
	switch {
	case readiness.PhysicalDeviceOnline:
		readiness.Status = "physical_online"
	case readiness.PhysicalDeviceStale:
		readiness.Status = "physical_stale"
	case readiness.SimulatorDeviceOnline:
		readiness.Status = "simulator_only"
	default:
		readiness.Status = "physical_offline"
	}
	return readiness
}

func productV21ProfessionalReady(v21 productV21Readiness) bool {
	if productV21ProfessionalExecutionReady(v21.ProfessionalExecution) {
		return true
	}
	return v21.Healthy &&
		v21.Professional.Valid &&
		v21.Professional.CheckingAckWithin1200 &&
		v21.Professional.EvidenceAvailable &&
		v21.Professional.CardsAvailable &&
		v21.Professional.FollowUpsAvailable &&
		v21.Professional.AdapterExecuted
}

func productV21ProfessionalExecutionReady(execution productV21ProfessionalExecutionReadiness) bool {
	return execution.Valid &&
		execution.QueryExecuted &&
		execution.AdapterExecuted &&
		execution.CheckingAckWithin1200 &&
		execution.EvidenceAvailable &&
		execution.CardsAvailable &&
		execution.FollowUpsAvailable &&
		execution.RedactionOK
}

func productProfessionalRitualReady(v21 productV21Readiness) bool {
	return productV21ProfessionalExecutionReady(v21.ProfessionalExecution) &&
		v21.ProfessionalExecution.SourceKind == "xiaozhi_professional_bench_report" &&
		v21.ProfessionalExecution.ProfessionalAcceptanceStatus == "external_gateway_ready"
}

func productProfessionalReadRecordReady(v21 productV21Readiness) bool {
	return productProfessionalRitualReady(v21) &&
		v21.ProfessionalExecution.ReadRecordObserved &&
		v21.ProfessionalExecution.ReadRecordCompleted &&
		v21adapter.ValidQueryScope(v21.ProfessionalExecution.ReadRecordQueryScope) &&
		validProductV21WorkspaceStatus(v21.ProfessionalExecution.ReadRecordWorkspaceStatus) &&
		validProductV21SourceScopeCounts(v21.ProfessionalExecution.ReadRecordSourceScopeCounts)
}

func productProfessionalRitualSourceReport(v21 productV21Readiness) string {
	if v21.ProfessionalExecution.SourceKind != "xiaozhi_professional_bench_report" {
		return ""
	}
	return v21.ProfessionalExecution.SourceReport
}

func productProfessionalReadRecordSourceReport(v21 productV21Readiness) string {
	if !productProfessionalReadRecordReady(v21) {
		return ""
	}
	return v21.ProfessionalExecution.SourceReport
}

func buildProductV21ProfessionalExecutionReadiness(professional productV21ProfessionalReadiness) productV21ProfessionalExecutionReadiness {
	if !professional.Valid || !professional.AdapterExecuted {
		return productV21ProfessionalExecutionReadiness{}
	}
	return productV21ProfessionalExecutionReadiness{
		Valid:                        true,
		SourceKind:                   productV21ProfessionalExecutionSourceKind(professional.ProfessionalAcceptanceStatus),
		SourceReport:                 professional.SourceReport,
		QueryExecuted:                true,
		AdapterExecuted:              true,
		QueryScope:                   professional.QueryScope,
		WorkspaceStatus:              professional.WorkspaceStatus,
		SourceScopeCounts:            copyStringIntMap(professional.SourceScopeCounts),
		CheckingAckWithin1200:        professional.CheckingAckWithin1200,
		EvidenceAvailable:            professional.EvidenceAvailable,
		CardsAvailable:               professional.CardsAvailable,
		FollowUpsAvailable:           professional.FollowUpsAvailable,
		EvidenceCount:                professional.EvidenceCount,
		CardCount:                    professional.CardCount,
		FollowUpCount:                professional.FollowUpCount,
		ProfessionalAcceptanceStatus: professional.ProfessionalAcceptanceStatus,
		RedactionOK:                  true,
		ReadRecordObserved:           professional.ReadRecordObserved,
		ReadRecordCompleted:          professional.ReadRecordCompleted,
		ReadRecordQueryScope:         professional.ReadRecordQueryScope,
		ReadRecordWorkspaceStatus:    professional.ReadRecordWorkspaceStatus,
		ReadRecordSourceScopeCounts:  copyStringIntMap(professional.ReadRecordSourceScopeCounts),
		PRDAccepted:                  false,
	}
}

func copyStringIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func validProductV21SourceScopeCounts(counts map[string]int) bool {
	for scope, count := range counts {
		switch scope {
		case "public", "personal":
		default:
			return false
		}
		if count < 0 {
			return false
		}
	}
	return true
}

func validProductV21WorkspaceStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "searchable", "uploaded", "indexing", "failed", "unavailable":
		return true
	default:
		return false
	}
}

func productV21ProfessionalExecutionSourceKind(status string) string {
	switch strings.TrimSpace(status) {
	case "adapter_smoke_passed":
		return "v21_adapter_smoke_report"
	case "external_gateway_ready":
		return "xiaozhi_professional_bench_report"
	default:
		return "v21_professional_report"
	}
}

func buildProductServerSideReadiness(report productReadinessReport) productServerSideReadiness {
	readiness := productServerSideReadiness{
		Status:                             "blocked",
		AcceptanceStatus:                   "server_side_blocked",
		ProviderSmokeSourceReport:          report.Provider.SmokeSourceReport,
		V21ProfessionalSourceReport:        report.V21.Professional.SourceReport,
		ProfessionalRitualSourceReport:     productProfessionalRitualSourceReport(report.V21),
		ProfessionalReadRecordSourceReport: productProfessionalReadRecordSourceReport(report.V21),
		HostVoiceSourceReport:              report.Voice.VoicePipeline.SourceReport,
		RoleplayVoiceSourceReport:          report.Roleplay.VoiceRuntimeSourceReport,
		RequiresPhysicalAcceptance:         !report.StackChan.PhysicalEvidence.PRDPhysicalAccepted,
		VoiceChain:                         report.Voice.VoiceChain,
	}
	readiness.GatewayReady = report.Gateway.Healthy
	readiness.ProviderEvidenceReady = report.Provider.RealProviderReady &&
		report.Provider.SmokeEvidenceValid &&
		report.Provider.SmokeExecuted
	readiness.V21ProfessionalEvidenceReady = productV21ProfessionalReady(report.V21)
	readiness.ProfessionalRitualReady = productProfessionalRitualReady(report.V21)
	readiness.ProfessionalReadRecordReady = productProfessionalReadRecordReady(report.V21)
	readiness.HostVoiceLoopbackReady = report.Voice.VoicePipeline.HostProductChainReady
	readiness.RoleplayVoiceRuntimeReady = report.Roleplay.VoiceRuntimeReady
	readiness.WakeWordReady = report.WakeWord.ProductReady
	if !readiness.GatewayReady {
		readiness.MissingEvidence = append(readiness.MissingEvidence, "gateway")
	}
	if !readiness.ProviderEvidenceReady {
		readiness.MissingEvidence = append(readiness.MissingEvidence, "provider_smoke")
	}
	if !readiness.V21ProfessionalEvidenceReady {
		readiness.MissingEvidence = append(readiness.MissingEvidence, "v21_professional_smoke")
	}
	if !readiness.ProfessionalRitualReady {
		readiness.MissingEvidence = append(readiness.MissingEvidence, "professional_ritual_execution")
	} else if !readiness.ProfessionalReadRecordReady {
		readiness.MissingEvidence = append(readiness.MissingEvidence, "professional_read_record")
	}
	if !readiness.HostVoiceLoopbackReady {
		readiness.MissingEvidence = append(readiness.MissingEvidence, "host_voice_loopback")
	}
	if !readiness.RoleplayVoiceRuntimeReady {
		readiness.MissingEvidence = append(readiness.MissingEvidence, "roleplay_voice_runtime")
	}
	if !readiness.WakeWordReady {
		readiness.MissingEvidence = append(readiness.MissingEvidence, "wake_word")
	}
	if !productVoiceChainLaunchPolicySatisfied(report.Voice.VoiceChain) {
		if productVoiceChainHasFinding(report.Voice.VoiceChain, "stepfun_not_selected") {
			readiness.MissingEvidence = append(readiness.MissingEvidence, "stepfun_not_selected")
		} else if report.Voice.VoiceChain.Available {
			readiness.MissingEvidence = append(readiness.MissingEvidence, "voice_chain_launch_policy")
		} else {
			readiness.MissingEvidence = append(readiness.MissingEvidence, "voice_chain_selector")
		}
	}
	readiness.CandidateReady = len(readiness.MissingEvidence) == 0
	if readiness.CandidateReady {
		readiness.Status = "candidate_ready"
		readiness.AcceptanceStatus = "server_side_candidate_ready"
	}
	return readiness
}

func fetchProductRoleplayReadiness(ctx context.Context, gatewayURL string) (productRoleplayReadiness, []productReadinessFinding) {
	unavailable := productRoleplayReadiness{
		Available:      false,
		ImmersionReady: false,
		Status:         "unavailable",
	}
	endpoint, _, err := firmwareGatewayEndpoint(gatewayURL, "/v1/roleplay-profile", nil)
	if err != nil {
		return unavailable, []productReadinessFinding{{
			Code:    "roleplay_profile_unavailable",
			Message: "A21 Gateway roleplay profile readiness is unavailable",
			Detail:  "invalid_gateway_url",
		}}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return unavailable, []productReadinessFinding{{
			Code:    "roleplay_profile_unavailable",
			Message: "A21 Gateway roleplay profile readiness is unavailable",
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
	var status gateway.RoleplayProfileResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, 16384))
	if err := decoder.Decode(&status); err != nil {
		return unavailable, []productReadinessFinding{invalidProductRoleplayProfileFinding()}
	}
	readiness, ok := productRoleplayReadinessFromGateway(status)
	if !ok {
		return unavailable, []productReadinessFinding{invalidProductRoleplayProfileFinding()}
	}
	return readiness, nil
}

func productRoleplayReadinessFromGateway(status gateway.RoleplayProfileResponse) (productRoleplayReadiness, bool) {
	if strings.TrimSpace(status.SchemaVersion) != gateway.RoleplayProfileSchemaVersion {
		return productRoleplayReadiness{}, false
	}
	runtime := status.Runtime
	plan := status.ExpressionPlan
	profile := strings.TrimSpace(status.SelectedRoleplayProfile)
	scenario := strings.TrimSpace(status.SelectedScenario)
	voiceClone := strings.TrimSpace(status.SelectedVoiceCloneProfile)
	if !productVoiceChainSafeID(profile) ||
		!productVoiceChainSafeID(scenario) ||
		!productVoiceChainSafeID(voiceClone) ||
		!productRoleplayRuntimeSafe(runtime) ||
		!productRoleplayExpressionPlanSafe(plan) {
		return productRoleplayReadiness{}, false
	}
	memoryReady := !runtime.MemoryConfigured || runtime.MemoryPromptInputReady
	runtimeRedactionOK := !runtime.PromptStored &&
		!runtime.MemoryTextStored &&
		!runtime.TranscriptStored &&
		!runtime.ProviderOutputStored &&
		!runtime.VoiceCloneSampleStored &&
		!runtime.ProfessionalRouteAllowed &&
		!runtime.V21Executed
	expressionRedactionOK := !plan.Redaction.PromptTextStored &&
		!plan.Redaction.MemoryTextStored &&
		!plan.Redaction.TranscriptStored &&
		!plan.Redaction.ProviderOutputStored &&
		!plan.Redaction.AudioStored &&
		!plan.Redaction.VoiceCloneSampleStored
	expressionAvailable := plan.SchemaVersion == "a21.roleplay_expression_plan.v1" &&
		plan.Adapter == "official_stackchan_action_plan" &&
		plan.DeliveryPolicy == "no_send_plan_only" &&
		plan.ActionCount > 0 &&
		plan.PacketCount > 0
	voiceCloneReady := voiceClone != "" && !runtime.VoiceCloneSampleStored
	immersionReady := runtime.Mode == "roleplay" &&
		runtime.SoulPromptInputReady &&
		runtime.PromptComposed &&
		memoryReady &&
		voiceCloneReady &&
		expressionAvailable &&
		runtimeRedactionOK &&
		expressionRedactionOK
	readinessStatus := "ready"
	var findings []string
	if !immersionReady {
		readinessStatus = "blocked"
		findings = productRoleplayReadinessFindings(runtime, plan, memoryReady, voiceCloneReady, expressionAvailable, runtimeRedactionOK, expressionRedactionOK)
	}
	return productRoleplayReadiness{
		Available:                  true,
		ImmersionReady:             immersionReady,
		Status:                     readinessStatus,
		SelectedRoleplayProfile:    profile,
		SelectedScenario:           scenario,
		SelectedVoiceCloneProfile:  voiceClone,
		SoulPromptInputReady:       runtime.SoulPromptInputReady,
		PromptComposed:             runtime.PromptComposed,
		PromptPartCount:            len(runtime.PromptParts),
		MemoryConfigured:           runtime.MemoryConfigured,
		MemoryPromptInputReady:     runtime.MemoryPromptInputReady,
		MemoryReady:                memoryReady,
		MemoryHintCount:            runtime.MemoryHintCount,
		VoiceCloneProfileReady:     voiceCloneReady,
		ExpressionPlanAvailable:    expressionAvailable,
		ExpressionDeliveryPolicy:   plan.DeliveryPolicy,
		ExpressionActionCount:      plan.ActionCount,
		ExpressionPacketCount:      plan.PacketCount,
		ExpressionPhysicalAccepted: plan.PhysicalAccepted,
		PhysicalAccepted:           false,
		PromptStored:               runtime.PromptStored || plan.Redaction.PromptTextStored,
		MemoryTextStored:           runtime.MemoryTextStored || plan.Redaction.MemoryTextStored,
		TranscriptStored:           runtime.TranscriptStored || plan.Redaction.TranscriptStored,
		ProviderOutputStored:       runtime.ProviderOutputStored || plan.Redaction.ProviderOutputStored,
		AudioStored:                plan.Redaction.AudioStored,
		VoiceCloneSampleStored:     runtime.VoiceCloneSampleStored || plan.Redaction.VoiceCloneSampleStored,
		ProfessionalRouteAllowed:   runtime.ProfessionalRouteAllowed,
		V21Executed:                runtime.V21Executed,
		ExpressionRedactionOK:      expressionRedactionOK,
		RuntimeRedactionOK:         runtimeRedactionOK,
		Findings:                   findings,
	}, true
}

func productRoleplayRuntimeSafe(runtime gateway.RoleplayRuntimeSummary) bool {
	if runtime.SchemaVersion != "a21.roleplay_runtime.v1" ||
		runtime.Mode != "roleplay" ||
		!productVoiceChainSafeID(runtime.RoleplayProfile) ||
		!productVoiceChainSafeID(runtime.Scenario) ||
		!productVoiceChainSafeID(runtime.VoiceCloneProfile) ||
		runtime.MemoryHintCount < 0 ||
		len(runtime.PromptParts) > 12 {
		return false
	}
	for _, part := range runtime.PromptParts {
		if !productRoleplayPromptPartSafe(part) {
			return false
		}
	}
	return true
}

func productRoleplayExpressionPlanSafe(plan gateway.RoleplayExpressionPlan) bool {
	if plan.SchemaVersion != "a21.roleplay_expression_plan.v1" ||
		plan.Adapter != "official_stackchan_action_plan" ||
		plan.DeliveryPolicy != "no_send_plan_only" ||
		!productVoiceChainSafeID(plan.RoleplayProfile) ||
		!productVoiceChainSafeID(plan.Scenario) ||
		!productVoiceChainSafeID(plan.VoiceCloneProfile) ||
		plan.MemoryHintCount < 0 ||
		plan.ActionCount < 0 ||
		plan.PacketCount < 0 ||
		len(plan.Actions) > 12 {
		return false
	}
	for _, action := range plan.Actions {
		if !productVoiceChainSafeID(action.Phase) ||
			!productVoiceChainSafeID(action.Event) ||
			!productVoiceChainSafeID(action.Value) ||
			action.PacketCount < 0 {
			return false
		}
		for key, value := range action.Surfaces {
			if !productVoiceChainSafeID(key) || !productVoiceChainSafeID(value) {
				return false
			}
		}
	}
	for key, value := range plan.Surfaces {
		if !productVoiceChainSafeID(key) || !productVoiceChainSafeID(value) {
			return false
		}
	}
	return true
}

func productRoleplayPromptPartSafe(part string) bool {
	part = strings.TrimSpace(part)
	if productVoiceChainSafeID(part) {
		return true
	}
	left, right, ok := strings.Cut(part, ":")
	return ok && productVoiceChainSafeID(left) && productVoiceChainSafeID(right)
}

func productRoleplayReadinessFindings(runtime gateway.RoleplayRuntimeSummary, plan gateway.RoleplayExpressionPlan, memoryReady bool, voiceCloneReady bool, expressionAvailable bool, runtimeRedactionOK bool, expressionRedactionOK bool) []string {
	var findings []string
	if !runtime.SoulPromptInputReady {
		findings = appendProductFindingCode(findings, "roleplay_soul_prompt_not_ready")
	}
	if !runtime.PromptComposed {
		findings = appendProductFindingCode(findings, "roleplay_prompt_not_composed")
	}
	if !memoryReady {
		findings = appendProductFindingCode(findings, "roleplay_memory_prompt_not_ready")
	}
	if !voiceCloneReady {
		findings = appendProductFindingCode(findings, "roleplay_voice_clone_not_ready")
	}
	if !expressionAvailable {
		findings = appendProductFindingCode(findings, "roleplay_expression_plan_not_ready")
	}
	if plan.PhysicalAccepted {
		findings = appendProductFindingCode(findings, "roleplay_expression_physical_claim_ignored")
	}
	if !runtimeRedactionOK || !expressionRedactionOK {
		findings = appendProductFindingCode(findings, "roleplay_redaction_not_ok")
	}
	if runtime.ProfessionalRouteAllowed || runtime.V21Executed {
		findings = appendProductFindingCode(findings, "roleplay_professional_boundary_not_ok")
	}
	return findings
}

const productRoleplayVoiceReportSchemaVersion = "a21.roleplay_voice_probe.v1"

type productRoleplayVoiceReportEvidence struct {
	Valid                       bool
	SourceReport                string
	Status                      string
	Mode                        string
	Route                       string
	ExecutionMode               string
	RoleplayProfile             string
	Scenario                    string
	VoiceCloneProfile           string
	VoiceRuntimeReady           bool
	TraceMarkerCount            int
	TextStreamExecuted          bool
	PromptInputUsed             bool
	VoiceCloneProfileUsed       bool
	AudioDownlinkObserved       bool
	DevicePlaybackStartObserved bool
	PhysicalAccepted            bool
	PRDAccepted                 bool
}
