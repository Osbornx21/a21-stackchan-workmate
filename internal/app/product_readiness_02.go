package app

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"a21.local/a21/internal/personality"
	"a21.local/a21/internal/providers"
)

func productLatestXiaozhiReportAccepted(path string) bool {
	evidence, _ := loadProductXiaozhiReportEvidence(path)
	return evidence.Valid &&
		evidence.HostProductChainReady &&
		evidence.FailureCount == 0 &&
		evidence.AnswerFirstAudioP95MS > 0 &&
		evidence.AnswerFirstAudioP95MS < 1500 &&
		evidence.BargeInStopP95MS >= 0 &&
		evidence.BargeInStopP95MS < 300 &&
		!evidence.PRDAccepted
}

func productLatestV21ProfessionalReportAccepted(path string) bool {
	evidence, _ := loadProductV21ProfessionalReportEvidence(path)
	return evidence.Valid && evidence.AdapterExecuted
}

func productLatestV21AdapterSmokeReportAccepted(path string) bool {
	evidence, _ := loadProductV21AdapterSmokeReportEvidence(path, productV21Readiness{})
	return evidence.Valid && evidence.AdapterExecuted
}

func resolveLatestProductV21Reports(options *productReadinessOptions, v21 productV21Readiness) []productReadinessFinding {
	if options == nil || !options.UseLatestReports ||
		strings.TrimSpace(options.V21ProfessionalReport) != "" ||
		strings.TrimSpace(options.V21AdapterSmokeReport) != "" {
		return nil
	}
	reportDir := firstNonEmpty(strings.TrimSpace(options.OutputDir), "reports")
	selected, findings := latestAcceptedProductReadinessReportPath(reportDir, "v21_professional", []string{
		"a21-xiaozhi-professional-bench-*.json",
		"a21-v21-professional-readiness-*.json",
	}, func(path string) bool {
		evidence, _ := loadProductV21ProfessionalReportEvidence(path)
		return evidence.Valid && evidence.AdapterExecuted
	})
	if selected != "" {
		options.V21ProfessionalReport = selected
		return findings
	}
	adapterSelected, adapterFindings := latestAcceptedProductReadinessReportPath(reportDir, "v21_adapter_smoke", []string{
		"a21-v21-adapter-smoke-*.json",
	}, func(path string) bool {
		evidence, _ := loadProductV21AdapterSmokeReportEvidence(path, v21)
		return evidence.Valid && evidence.AdapterExecuted
	})
	options.V21AdapterSmokeReport = adapterSelected
	return append(findings, adapterFindings...)
}

func resolveLatestProductWakeWordFirmwarePlan(options *productReadinessOptions, wakeWord productWakeWordReadiness) []productReadinessFinding {
	if options == nil || !options.UseLatestReports || strings.TrimSpace(options.WakeWordFirmwarePlan) != "" {
		return nil
	}
	reportDir := firstNonEmpty(strings.TrimSpace(options.OutputDir), "reports")
	selected, findings := latestAcceptedProductReadinessReportPath(reportDir, "wake_word_firmware_plan", []string{
		"a21-wake-word-firmware-plan-*.json",
	}, func(path string) bool {
		evidence, _ := loadProductWakeWordFirmwarePlanEvidence(path)
		return evidence.Valid && matchingProductWakeWordFirmwarePlan(wakeWord, evidence)
	})
	options.WakeWordFirmwarePlan = selected
	return findings
}

func resolveLatestProductWakeWordFirmwarePackage(options *productReadinessOptions, wakeWord productWakeWordReadiness) []productReadinessFinding {
	if options == nil || !options.UseLatestReports || strings.TrimSpace(options.WakeWordFirmwarePackage) != "" {
		return nil
	}
	reportDir := firstNonEmpty(strings.TrimSpace(options.OutputDir), "reports")
	selected, findings := latestAcceptedProductReadinessReportPath(reportDir, "wake_word_firmware_package", []string{
		"a21-wake-word-firmware-package-*.json",
	}, func(path string) bool {
		evidence, _ := loadProductWakeWordFirmwarePackageEvidence(path)
		return evidence.Valid && matchingProductWakeWordFirmwarePackage(wakeWord, evidence)
	})
	options.WakeWordFirmwarePackage = selected
	return findings
}

func resolveLatestProductWakeWordPhysicalAcceptance(options *productReadinessOptions, wakeWord productWakeWordReadiness, packageEvidence productWakeWordFirmwarePackageEvidence) []productReadinessFinding {
	if options == nil || !options.UseLatestReports || strings.TrimSpace(options.WakeWordPhysicalAcceptance) != "" {
		return nil
	}
	reportDir := firstNonEmpty(strings.TrimSpace(options.OutputDir), "reports")
	selected, findings := latestAcceptedProductReadinessReportPath(reportDir, "wake_word_physical_acceptance", []string{
		"a21-wake-word-physical-acceptance-*.json",
	}, func(path string) bool {
		evidence, _ := loadProductWakeWordPhysicalAcceptanceEvidence(path)
		return evidence.Valid && matchingProductWakeWordPhysicalAcceptance(wakeWord, packageEvidence, evidence)
	})
	options.WakeWordPhysicalAcceptance = selected
	return findings
}

type productLatestReadinessReportKind struct {
	Kind     string
	Patterns []string
	Accept   func(string) bool
}

type productLatestReadinessReportKindCandidate struct {
	Path    string
	Kind    string
	Accept  func(string) bool
	ModTime time.Time
}

func latestAcceptedProductReadinessReportAcrossKinds(reportDir string, kinds []productLatestReadinessReportKind) (string, string, []productReadinessFinding) {
	var candidates []productLatestReadinessReportKindCandidate
	for _, kind := range kinds {
		for _, path := range latestProductReadinessReportCandidates(reportDir, kind.Patterns) {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			candidates = append(candidates, productLatestReadinessReportKindCandidate{
				Path:    path,
				Kind:    strings.TrimSpace(kind.Kind),
				Accept:  kind.Accept,
				ModTime: info.ModTime(),
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ModTime.Equal(candidates[j].ModTime) {
			return filepath.Base(candidates[i].Path) > filepath.Base(candidates[j].Path)
		}
		return candidates[i].ModTime.After(candidates[j].ModTime)
	})
	var findings []productReadinessFinding
	skipped := 0
	for _, candidate := range candidates {
		if candidate.Accept == nil || candidate.Accept(candidate.Path) {
			findings = append(findings, latestProductReadinessSkippedSummaryFinding("v21_evidence", skipped)...)
			return candidate.Path, candidate.Kind, findings
		}
		skipped++
		if skipped <= latestProductReadinessSkippedDetailLimit {
			findings = append(findings, productReadinessFinding{
				Code:    "latest_report_candidate_skipped",
				Message: "Latest product-readiness candidate report did not satisfy the evidence contract",
				Detail:  candidate.Kind + ":" + filepath.Base(filepath.Clean(candidate.Path)),
			})
		}
	}
	findings = append(findings, latestProductReadinessSkippedSummaryFinding("v21_evidence", skipped)...)
	return "", "", findings
}

func buildProductReadinessReport(ctx context.Context, options productReadinessOptions, env []string) productReadinessReport {
	gatewayURL := firstNonEmpty(strings.TrimSpace(options.GatewayURL), "http://127.0.0.1:21080")
	deviceID := firstNonEmpty(strings.TrimSpace(options.DeviceID), "stackchan-001")
	memoryState, _ := personality.MemoryStateFromEnv(env)
	report := productReadinessReport{
		SchemaVersion: "a21.product_readiness.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "blocked",
		Gateway: productGatewayReadiness{
			URL: productSurfaceLabel(gatewayURL, ""),
		},
		StackChan: productStackChanReadiness{DeviceID: deviceID},
		Memory:    memoryState,
	}
	report.Findings = append(report.Findings, options.LatestReportFindings...)
	report.Gateway.Healthy = gatewayHealthOK(ctx, gatewayURL)
	report.Gateway.DeviceRegistry = false
	if report.Gateway.Healthy {
		report.Gateway.Status = "ready"
	} else {
		report.Gateway.Status = "blocked"
		report.Findings = append(report.Findings, productReadinessFinding{Code: "gateway_not_ready", Message: "A21 Gateway is not reachable", Detail: "run `go run ./cmd/a21 gateway --addr 127.0.0.1:21080`"})
	}
	deviceReport, err := fetchFirmwareDeviceReport(gatewayURL)
	if err == nil {
		report.Gateway.DeviceRegistry = true
		report.StackChan = buildProductStackChanReadiness(deviceReport, deviceID)
	} else {
		report.StackChan.Status = "gateway_device_registry_unavailable"
		report.Findings = append(report.Findings, productReadinessFinding{Code: "device_registry_unavailable", Message: "A21 device registry is not reachable"})
	}
	voiceChain, voiceChainFindings := fetchProductVoiceChainReadiness(ctx, gatewayURL)
	report.Findings = append(report.Findings, voiceChainFindings...)
	report.Provider = buildProductProviderReadiness(env)
	if productReadinessProviderSmokeSelectionEnabled(options) {
		alignProductProviderReadinessWithGatewayVoiceChain(&report.Provider, voiceChain)
	}
	report.Findings = append(report.Findings, resolveLatestProductProviderSmokeReport(&options, report.Provider)...)
	providerSmokeEvidence, providerSmokeFindings := loadProductProviderSmokeReportEvidence(options.ProviderSmokeReport)
	report.Findings = append(report.Findings, providerSmokeFindings...)
	if providerSmokeEvidence.Valid {
		report.Findings = append(report.Findings, attachProductProviderSmokeEvidence(&report.Provider, providerSmokeEvidence)...)
	}
	providerRealtimeEvidence, providerRealtimeFindings := loadProductProviderRealtimeReportEvidence(options.ProviderRealtimeReport)
	report.Findings = append(report.Findings, providerRealtimeFindings...)
	if providerRealtimeEvidence.Valid {
		report.Findings = append(report.Findings, attachProductProviderRealtimeEvidence(&report.Provider, providerRealtimeEvidence)...)
	}
	report.V21 = buildProductV21Readiness(env)
	roleplay, roleplayFindings := fetchProductRoleplayReadiness(ctx, gatewayURL)
	report.Roleplay = roleplay
	report.Findings = append(report.Findings, roleplayFindings...)
	roleplayVoiceEvidence, roleplayVoiceFindings := loadProductRoleplayVoiceReportEvidence(options.RoleplayVoiceReport)
	report.Findings = append(report.Findings, roleplayVoiceFindings...)
	if roleplayVoiceEvidence.Valid {
		report.Findings = append(report.Findings, attachProductRoleplayVoiceEvidence(&report.Roleplay, roleplayVoiceEvidence)...)
	}
	wakeWord, wakeWordFindings := fetchProductWakeWordReadiness(ctx, gatewayURL)
	report.WakeWord = wakeWord
	report.Findings = append(report.Findings, wakeWordFindings...)
	voiceChainCapabilityEvidence, voiceChainCapabilityFindings := loadProductVoiceChainReadinessReportEvidence(options.VoiceChainReadinessReport)
	report.Findings = append(report.Findings, voiceChainCapabilityFindings...)
	if voiceChainCapabilityEvidence.Valid {
		report.Findings = append(report.Findings, attachProductVoiceChainCapabilityEvidence(&voiceChain, voiceChainCapabilityEvidence)...)
	}
	report.Findings = append(report.Findings, resolveLatestProductWakeWordFirmwarePlan(&options, report.WakeWord)...)
	wakeWordPlan, wakeWordPlanFindings := loadProductWakeWordFirmwarePlanEvidence(options.WakeWordFirmwarePlan)
	report.Findings = append(report.Findings, wakeWordPlanFindings...)
	if wakeWordPlan.Valid {
		report.Findings = append(report.Findings, attachProductWakeWordFirmwarePlan(&report.WakeWord, wakeWordPlan)...)
	}
	report.Findings = append(report.Findings, resolveLatestProductWakeWordFirmwarePackage(&options, report.WakeWord)...)
	wakeWordPackage, wakeWordPackageFindings := loadProductWakeWordFirmwarePackageEvidence(options.WakeWordFirmwarePackage)
	report.Findings = append(report.Findings, wakeWordPackageFindings...)
	if wakeWordPackage.Valid {
		report.Findings = append(report.Findings, attachProductWakeWordFirmwarePackage(&report.WakeWord, wakeWordPackage)...)
	}
	report.Findings = append(report.Findings, resolveLatestProductWakeWordPhysicalAcceptance(&options, report.WakeWord, wakeWordPackage)...)
	wakeWordPhysicalAcceptance, wakeWordPhysicalAcceptanceFindings := loadProductWakeWordPhysicalAcceptanceEvidence(options.WakeWordPhysicalAcceptance)
	report.Findings = append(report.Findings, wakeWordPhysicalAcceptanceFindings...)
	if wakeWordPhysicalAcceptance.Valid {
		report.Findings = append(report.Findings, attachProductWakeWordPhysicalAcceptance(&report.WakeWord, wakeWordPackage, wakeWordPhysicalAcceptance)...)
	}
	report.Findings = append(report.Findings, resolveLatestProductV21Reports(&options, report.V21)...)
	adapterSmokeEvidence, adapterSmokeFindings := loadProductV21AdapterSmokeReportEvidence(options.V21AdapterSmokeReport, report.V21)
	report.Findings = append(report.Findings, adapterSmokeFindings...)
	if adapterSmokeEvidence.Valid {
		report.V21.QueryExecuted = true
		report.V21.Professional = adapterSmokeEvidence
		report.V21.ProfessionalExecution = buildProductV21ProfessionalExecutionReadiness(adapterSmokeEvidence)
	}
	professionalEvidence, professionalFindings := loadProductV21ProfessionalReportEvidence(options.V21ProfessionalReport)
	report.Findings = append(report.Findings, professionalFindings...)
	if professionalEvidence.Valid {
		if professionalEvidence.AdapterExecuted {
			report.V21.QueryExecuted = true
			report.V21.Professional = professionalEvidence
			report.V21.ProfessionalExecution = buildProductV21ProfessionalExecutionReadiness(professionalEvidence)
		} else if !report.V21.Professional.Valid {
			report.V21.Professional = professionalEvidence
		}
	}
	xiaozhiEvidence, xiaozhiFindings := loadProductXiaozhiReportEvidence(options.XiaozhiReport)
	report.Findings = append(report.Findings, xiaozhiFindings...)
	physicalEvidence, physicalFindings := loadProductPhysicalStackChanReportEvidence(options.PhysicalStackChanReport)
	report.StackChan.PhysicalEvidence = physicalEvidence
	report.Findings = append(report.Findings, physicalFindings...)
	report.Voice = buildProductVoiceReadiness(env, report.Provider, report.StackChan, xiaozhiEvidence)
	report.Voice.VoiceChain = voiceChain
	report.ServerSide = buildProductServerSideReadiness(report)
	report.LaunchReady = report.Gateway.Healthy &&
		report.Provider.RealProviderReady &&
		report.V21.Healthy &&
		productV21ProfessionalReady(report.V21) &&
		productProfessionalRitualReady(report.V21) &&
		productProfessionalReadRecordReady(report.V21) &&
		report.StackChan.PhysicalDeviceOnline &&
		report.StackChan.PhysicalEvidence.PRDPhysicalAccepted &&
		report.Voice.ContinuousVoiceReady &&
		report.WakeWord.ProductReady &&
		productWakeWordPhysicalAccepted(report.WakeWord) &&
		productVoiceChainLaunchPolicySatisfied(report.Voice.VoiceChain)
	report.DemoReady = false
	report.NextActions = buildProductNextActions(report)
	for _, action := range report.NextActions {
		report.Findings = append(report.Findings, productReadinessFinding{Code: "launch_gap", Message: action})
	}
	report.Status = productReadinessStatus(report)
	report.CanonicalDecision = buildProductCanonicalReadinessDecision(report)
	return report
}

func productReadinessStatus(report productReadinessReport) string {
	switch {
	case report.LaunchReady:
		return "real_launch_ready"
	case report.ServerSide.CandidateReady:
		return "server_side_candidate_ready"
	case productReadinessHasExecutedServerSideEvidence(report):
		return "server_side_blocked"
	default:
		return "blocked"
	}
}

func productReadinessHasExecutedServerSideEvidence(report productReadinessReport) bool {
	return report.Provider.RealProviderReady ||
		productV21ProfessionalReady(report.V21) ||
		report.Voice.VoicePipeline.HostProductChainReady
}

func buildProductProviderReadiness(env []string) productProviderReadiness {
	catalog := providers.ProviderCatalogFromEnv(env)
	readiness := productProviderReadiness{Primary: catalog.Primary}
	for _, provider := range catalog.Providers {
		if provider.Selected {
			readiness.Selected = provider.Name
			readiness.SelectedFamily = provider.Family
			readiness.SelectedConfigured = provider.Configured
			readiness.SelectedRouteEligible = provider.RouteEligible
			readiness.MissingEnv = append([]string(nil), provider.MissingEnv...)
			readiness.PresentEnv = append([]string(nil), provider.PresentEnv...)
		}
	}
	return readiness
}

func productReadinessProviderSmokeSelectionEnabled(options productReadinessOptions) bool {
	return options.UseLatestReports || strings.TrimSpace(options.ProviderSmokeReport) != ""
}

func alignProductProviderReadinessWithGatewayVoiceChain(readiness *productProviderReadiness, voiceChain productVoiceChainReadiness) {
	if readiness == nil || !voiceChain.Available {
		return
	}
	selected := strings.TrimSpace(voiceChain.SelectedLLMProfile)
	if selected == "" ||
		selected == "mock" ||
		providerLatencySafeIdentifier(selected, false) != selected {
		return
	}
	if strings.TrimSpace(readiness.Selected) != "" && strings.TrimSpace(readiness.Selected) != "mock" {
		return
	}
	readiness.Primary = selected
	readiness.Selected = selected
	readiness.SelectedFamily = string(providers.ProviderFamilyTextStream)
	readiness.SelectedConfigured = false
	readiness.SelectedRouteEligible = true
	readiness.RealProviderReady = false
	readiness.TextStreamReady = false
	readiness.VoiceRealtimeReady = false
	readiness.MissingEnv = nil
	readiness.PresentEnv = nil
}

type productProviderSmokeReportEvidence struct {
	Valid         bool
	Provider      string
	Family        string
	Protocol      string
	Status        string
	Configured    bool
	Executed      bool
	RouteEligible bool
	Stream        bool
	SourceReport  string
}

type productProviderSmokeReportFixture struct {
	SchemaVersion string                           `json:"schema_version"`
	GeneratedAtMS *int64                           `json:"generated_at_ms"`
	Provider      string                           `json:"provider"`
	Family        string                           `json:"family"`
	Protocol      string                           `json:"protocol"`
	Status        string                           `json:"status"`
	Configured    *bool                            `json:"configured"`
	Executed      *bool                            `json:"executed"`
	RouteEligible *bool                            `json:"route_eligible"`
	Stream        *bool                            `json:"stream"`
	Repeat        int                              `json:"repeat"`
	HTTPStatus    int                              `json:"http_status"`
	Attempts      []providers.ProviderSmokeAttempt `json:"attempts"`
	TimingSummary *providers.ProviderSmokeTiming   `json:"timing_summary"`
	EndpointHost  string                           `json:"endpoint_host"`
	BaseURLEnv    string                           `json:"base_url_env"`
	APIKeyEnv     string                           `json:"api_key_env"`
	ModelEnv      string                           `json:"model_env"`
	MissingEnv    []string                         `json:"missing_env"`
	Fallback      *providers.ProviderSmokeFallback `json:"fallback"`
	TraceMarkers  []providers.ProviderSmokeMarker  `json:"trace_markers"`
	Metrics       []providers.ProviderSmokeMetric  `json:"metrics"`
}

func loadProductProviderSmokeReportEvidence(path string) (productProviderSmokeReportEvidence, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productProviderSmokeReportEvidence{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productProviderSmokeReportEvidence{}, []productReadinessFinding{invalidProductProviderSmokeReportFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productProviderSmokeReportEvidence{}, []productReadinessFinding{invalidProductProviderSmokeReportFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productProviderSmokeReportEvidence{}, []productReadinessFinding{invalidProductProviderSmokeReportFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productProviderSmokeReportEvidence{}, []productReadinessFinding{invalidProductProviderSmokeReportFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productProviderSmokeReportContainsForbiddenValue(raw) {
		return productProviderSmokeReportEvidence{}, []productReadinessFinding{invalidProductProviderSmokeReportFinding()}
	}
	var fixture productProviderSmokeReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productProviderSmokeReportEvidence{}, []productReadinessFinding{invalidProductProviderSmokeReportFinding()}
	}
	if missingField := missingProductProviderSmokeReportField(fixture); missingField != "" {
		return productProviderSmokeReportEvidence{}, []productReadinessFinding{missingProductProviderSmokeReportFieldFinding(missingField)}
	}
	provider := providerLatencySafeIdentifier(fixture.Provider, false)
	family := providerLatencySafeIdentifier(fixture.Family, false)
	protocol := providerLatencySafeIdentifier(fixture.Protocol, false)
	if fixture.SchemaVersion != providers.ProviderSmokeSchemaVersion ||
		fixture.GeneratedAtMS == nil ||
		*fixture.GeneratedAtMS <= 0 ||
		provider == "" ||
		provider == "mock" ||
		family != string(providers.ProviderFamilyTextStream) ||
		!validProductProviderSmokeProtocol(protocol) ||
		strings.TrimSpace(fixture.Status) != string(providers.ProviderSmokePassed) ||
		!*fixture.Configured ||
		!*fixture.Executed ||
		!*fixture.RouteEligible ||
		len(fixture.MissingEnv) != 0 ||
		(fixture.HTTPStatus != 0 && (fixture.HTTPStatus < 200 || fixture.HTTPStatus >= 300)) ||
		productProviderSmokeEndpointHostUnsafe(fixture.EndpointHost) ||
		!productProviderSmokeEnvNamesSafe(fixture.APIKeyEnv, fixture.ModelEnv, fixture.BaseURLEnv) ||
		(fixture.Fallback != nil && fixture.Fallback.Activated) ||
		productProviderSmokeFallbackObserved(fixture) ||
		!validProductProviderSmokeStreamingEvidence(fixture) {
		return productProviderSmokeReportEvidence{}, []productReadinessFinding{invalidProductProviderSmokeReportFinding()}
	}
	stream := false
	if fixture.Stream != nil {
		stream = *fixture.Stream
	}
	return productProviderSmokeReportEvidence{
		Valid:         true,
		Provider:      provider,
		Family:        family,
		Protocol:      protocol,
		Status:        strings.TrimSpace(fixture.Status),
		Configured:    *fixture.Configured,
		Executed:      *fixture.Executed,
		RouteEligible: *fixture.RouteEligible,
		Stream:        stream,
		SourceReport:  filepath.Base(filepath.Clean(path)),
	}, nil
}

func attachProductProviderSmokeEvidence(readiness *productProviderReadiness, evidence productProviderSmokeReportEvidence) []productReadinessFinding {
	if !productProviderSmokeEvidenceMatchesSelected(*readiness, evidence) {
		return []productReadinessFinding{{
			Code:    "provider_smoke_report_mismatch",
			Message: "Provider smoke report does not match the currently selected configured A21 provider",
			Detail:  evidence.SourceReport,
		}}
	}
	readiness.Selected = evidence.Provider
	readiness.SelectedFamily = evidence.Family
	readiness.SelectedConfigured = evidence.Configured
	readiness.RealProviderReady = true
	readiness.TextStreamReady = evidence.Family == string(providers.ProviderFamilyTextStream)
	readiness.VoiceRealtimeReady = false
	readiness.MissingEnv = nil
	readiness.SmokeEvidenceValid = true
	readiness.SmokeProvider = evidence.Provider
	readiness.SmokeFamily = evidence.Family
	readiness.SmokeStatus = evidence.Status
	readiness.SmokeExecuted = evidence.Executed
	readiness.SmokeSourceReport = evidence.SourceReport
	return nil
}

func productProviderReadinessRequiresSmokeMatch(readiness productProviderReadiness) bool {
	return readiness.Selected != "" &&
		readiness.Selected != "mock" &&
		readiness.SelectedRouteEligible &&
		readiness.SelectedFamily == string(providers.ProviderFamilyTextStream)
}

func productProviderSmokeEvidenceMatchesSelected(readiness productProviderReadiness, evidence productProviderSmokeReportEvidence) bool {
	return productProviderReadinessRequiresSmokeMatch(readiness) &&
		evidence.Valid &&
		evidence.Executed &&
		evidence.RouteEligible &&
		evidence.Stream &&
		evidence.Provider == readiness.Selected &&
		evidence.Family == readiness.SelectedFamily
}

type productProviderRealtimeReportEvidence struct {
	Valid         bool
	Provider      string
	Family        string
	Protocol      string
	Status        string
	Configured    bool
	Executed      bool
	RouteEligible bool
	SourceReport  string
	EvidenceMode  string
}

func loadProductProviderRealtimeReportEvidence(path string) (productProviderRealtimeReportEvidence, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productProviderRealtimeReportEvidence{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productProviderRealtimeReportEvidence{}, []productReadinessFinding{invalidProductProviderRealtimeReportFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productProviderRealtimeReportEvidence{}, []productReadinessFinding{invalidProductProviderRealtimeReportFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productProviderRealtimeReportEvidence{}, []productReadinessFinding{invalidProductProviderRealtimeReportFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productProviderRealtimeReportEvidence{}, []productReadinessFinding{invalidProductProviderRealtimeReportFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productProviderSmokeReportContainsForbiddenValue(raw) {
		return productProviderRealtimeReportEvidence{}, []productReadinessFinding{invalidProductProviderRealtimeReportFinding()}
	}
	var fixture productProviderSmokeReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productProviderRealtimeReportEvidence{}, []productReadinessFinding{invalidProductProviderRealtimeReportFinding()}
	}
	if missingField := missingProductProviderRealtimeReportField(fixture); missingField != "" {
		return productProviderRealtimeReportEvidence{}, []productReadinessFinding{missingProductProviderRealtimeReportFieldFinding(missingField)}
	}
	provider := providerLatencySafeIdentifier(fixture.Provider, false)
	family := providerLatencySafeIdentifier(fixture.Family, false)
	protocol := providerLatencySafeIdentifier(fixture.Protocol, false)
	if fixture.SchemaVersion != providers.ProviderSmokeSchemaVersion ||
		fixture.GeneratedAtMS == nil ||
		*fixture.GeneratedAtMS <= 0 ||
		provider == "" ||
		provider == "mock" ||
		!validProductProviderRealtimeFamily(family) ||
		!validProductProviderRealtimeProtocol(protocol) ||
		strings.TrimSpace(fixture.Status) != string(providers.ProviderSmokePassed) ||
		!*fixture.Configured ||
		!*fixture.Executed ||
		len(fixture.MissingEnv) != 0 ||
		productProviderSmokeEndpointHostUnsafe(fixture.EndpointHost) ||
		!productProviderSmokeEnvNamesSafe(fixture.APIKeyEnv, fixture.ModelEnv, fixture.BaseURLEnv) ||
		(fixture.Fallback != nil && fixture.Fallback.Activated) ||
		productProviderSmokeFallbackObserved(fixture) {
		return productProviderRealtimeReportEvidence{}, []productReadinessFinding{invalidProductProviderRealtimeReportFinding()}
	}
	return productProviderRealtimeReportEvidence{
		Valid:         true,
		Provider:      provider,
		Family:        family,
		Protocol:      protocol,
		Status:        strings.TrimSpace(fixture.Status),
		Configured:    *fixture.Configured,
		Executed:      *fixture.Executed,
		RouteEligible: *fixture.RouteEligible,
		SourceReport:  filepath.Base(filepath.Clean(path)),
		EvidenceMode:  "offline_fixture",
	}, nil
}

func attachProductProviderRealtimeEvidence(readiness *productProviderReadiness, evidence productProviderRealtimeReportEvidence) []productReadinessFinding {
	if readiness.Selected != evidence.Provider || !readiness.SelectedConfigured {
		return []productReadinessFinding{{
			Code:    "provider_realtime_fixture_report_mismatch",
			Message: "Provider realtime fixture report does not match the currently selected configured A21 provider",
			Detail:  evidence.SourceReport,
		}}
	}
	readiness.RealtimeEvidenceValid = true
	readiness.RealtimeProvider = evidence.Provider
	readiness.RealtimeFamily = evidence.Family
	readiness.RealtimeProtocol = evidence.Protocol
	readiness.RealtimeStatus = evidence.Status
	readiness.RealtimeExecuted = evidence.Executed
	readiness.RealtimeRouteEligible = evidence.RouteEligible && readiness.SelectedRouteEligible
	readiness.RealtimeEvidenceMode = evidence.EvidenceMode
	readiness.RealtimeSourceReport = evidence.SourceReport
	readiness.VoiceRealtimeReady = evidence.Configured &&
		evidence.Executed &&
		readiness.RealtimeRouteEligible &&
		validProductProviderRealtimeFamily(evidence.Family)
	return nil
}

func missingProductProviderRealtimeReportField(fixture productProviderSmokeReportFixture) string {
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
	default:
		return ""
	}
}

func missingProductProviderRealtimeReportFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "provider_realtime_fixture_report_missing_field",
		Message: "Provider realtime fixture report is missing a required field",
		Detail:  field,
	}
}

func invalidProductProviderRealtimeReportFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "provider_realtime_fixture_report_invalid",
		Message: "Provider realtime fixture report is invalid or unsafe",
	}
}

func validProductProviderRealtimeFamily(family string) bool {
	switch family {
	case string(providers.ProviderFamilyVoiceRealtime), string(providers.ProviderFamilyVoiceHybrid):
		return true
	default:
		return false
	}
}

func validProductProviderRealtimeProtocol(protocol string) bool {
	return protocol == "websocket_realtime_fixture"
}
