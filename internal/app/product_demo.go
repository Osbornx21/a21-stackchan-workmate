package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
)

type productReadinessOptions struct {
	GatewayURL              string
	Addr                    string
	DeviceID                string
	OutputDir               string
	XiaozhiReport           string
	V21ProfessionalReport   string
	V21AdapterSmokeReport   string
	PhysicalStackChanReport string
	UseLatestReports        bool
	RequireReal             bool
	OpenBrowser             bool
	StatusOnly              bool
}

type productReadinessReport struct {
	SchemaVersion string                    `json:"schema_version"`
	GeneratedAtMS int64                     `json:"generated_at_ms"`
	Status        string                    `json:"status"`
	LaunchReady   bool                      `json:"launch_ready"`
	DemoReady     bool                      `json:"demo_ready"`
	SimulatorURL  string                    `json:"simulator_url"`
	Gateway       productGatewayReadiness   `json:"gateway"`
	Provider      productProviderReadiness  `json:"provider"`
	V21           productV21Readiness       `json:"v21"`
	StackChan     productStackChanReadiness `json:"stackchan"`
	Voice         productVoiceReadiness     `json:"voice"`
	NextActions   []string                  `json:"next_actions,omitempty"`
	Findings      []productReadinessFinding `json:"findings,omitempty"`
	ReportPath    string                    `json:"report_path,omitempty"`
}

type productGatewayReadiness struct {
	URL            string `json:"url"`
	Healthy        bool   `json:"healthy"`
	SimulatorReady bool   `json:"simulator_ready"`
	DeviceRegistry bool   `json:"device_registry_ready"`
	Status         string `json:"status"`
}

type productProviderReadiness struct {
	Primary            string   `json:"primary"`
	Selected           string   `json:"selected"`
	SelectedFamily     string   `json:"selected_family,omitempty"`
	SelectedConfigured bool     `json:"selected_configured"`
	RealProviderReady  bool     `json:"real_provider_ready"`
	TextStreamReady    bool     `json:"text_stream_ready"`
	VoiceRealtimeReady bool     `json:"voice_realtime_ready"`
	MissingEnv         []string `json:"missing_env,omitempty"`
	PresentEnv         []string `json:"present_env,omitempty"`
}

type productV21Readiness struct {
	Configured                bool                            `json:"configured"`
	Healthy                   bool                            `json:"healthy"`
	Status                    string                          `json:"status"`
	ProfessionalBridgeReady   bool                            `json:"professional_bridge_ready"`
	CheckingFeedbackSupported bool                            `json:"checking_feedback_supported"`
	MaxFirstResponseMS        int                             `json:"max_first_response_ms"`
	EvidenceContractReady     bool                            `json:"evidence_contract_ready"`
	QueryExecuted             bool                            `json:"query_executed"`
	QueryPath                 string                          `json:"query_path"`
	HealthPath                string                          `json:"health_path"`
	Professional              productV21ProfessionalReadiness `json:"professional"`
}

type productV21ProfessionalReadiness struct {
	Valid                        bool   `json:"valid"`
	CheckingAckWithin1200        bool   `json:"checking_ack_within_1200"`
	EvidenceAvailable            bool   `json:"evidence_available"`
	CardsAvailable               bool   `json:"cards_available"`
	FollowUpsAvailable           bool   `json:"follow_ups_available"`
	EvidenceCount                int    `json:"evidence_count"`
	CardCount                    int    `json:"card_count"`
	FollowUpCount                int    `json:"follow_up_count"`
	ProfessionalAcceptanceStatus string `json:"professional_acceptance_status,omitempty"`
	SourceReport                 string `json:"source_report,omitempty"`
	AdapterExecuted              bool   `json:"adapter_executed"`
	PRDAccepted                  bool   `json:"prd_accepted"`
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
}

type productVoicePipelineReadiness struct {
	ASRProfile                 string  `json:"asr_profile"`
	ASRProfileEnv              string  `json:"asr_profile_env"`
	TextStreamProfile          string  `json:"text_stream_profile"`
	TextStreamProfileEnv       string  `json:"text_stream_profile_env"`
	TTSProfile                 string  `json:"tts_profile"`
	TTSProfileEnv              string  `json:"tts_profile_env"`
	ExecutionMode              string  `json:"execution_mode,omitempty"`
	HostLocalASRReady          bool    `json:"host_local_asr_ready"`
	HostLocalTextReady         bool    `json:"host_local_text_ready"`
	HostLocalTTSReady          bool    `json:"host_local_tts_ready"`
	HostLoopbackCandidateReady bool    `json:"host_loopback_candidate_ready"`
	AcceptanceStatus           string  `json:"acceptance_status,omitempty"`
	AnswerFirstAudioP95MS      float64 `json:"answer_first_audio_p95_ms"`
	BargeInStopP95MS           float64 `json:"barge_in_stop_p95_ms"`
	FailureCount               int     `json:"failure_count"`
	SourceReport               string  `json:"source_report,omitempty"`
	PRDAccepted                bool    `json:"prd_accepted"`
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
			fmt.Fprintln(stdout, "a21 product-readiness [--gateway-url http://127.0.0.1:21080] [--device-id stackchan-001] [--xiaozhi-report report.json] [--v21-professional-report report.json] [--v21-adapter-smoke-report report.json] [--physical-stackchan-report report.json] [--use-latest-reports] [--output-dir reports] [--require-real]")
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
			fmt.Fprintln(stdout, "a21 demo [--addr 127.0.0.1:21080] [--device-id stackchan-001] [--open] [--status-only] [--require-real]")
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
	fmt.Fprintln(stdout, "A21 demo surface")
	fmt.Fprintf(stdout, "Simulator: %s\n", simulatorURL)
	fmt.Fprintln(stdout, "Mode: mock demo is allowed; real provider, real V21, and physical StackChan are reported by `a21 product-readiness`.")
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
	if strings.TrimSpace(options.XiaozhiReport) == "" {
		options.XiaozhiReport = latestProductReadinessReportPath(reportDir, []string{
			"a21-xiaozhi-voice-bench-*.json",
			"a21-local-voice-loopback-*.json",
		})
	}
	if strings.TrimSpace(options.V21ProfessionalReport) == "" {
		options.V21ProfessionalReport = latestProductReadinessReportPath(reportDir, []string{
			"a21-xiaozhi-professional-bench-*.json",
			"a21-v21-professional-readiness-*.json",
		})
	}
	if strings.TrimSpace(options.V21AdapterSmokeReport) == "" && strings.TrimSpace(options.V21ProfessionalReport) == "" {
		options.V21AdapterSmokeReport = latestProductReadinessReportPath(reportDir, []string{
			"a21-v21-adapter-smoke-*.json",
		})
	}
	if strings.TrimSpace(options.PhysicalStackChanReport) == "" {
		options.PhysicalStackChanReport = latestProductReadinessReportPath(reportDir, []string{
			"a21-physical-stackchan-evidence-*.json",
			"a21-xiaozhi-physical-evidence-*.json",
		})
	}
	return options
}

func latestProductReadinessReportPath(reportDir string, patterns []string) string {
	var selected string
	var selectedMod time.Time
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
			modTime := info.ModTime()
			if selected == "" || modTime.After(selectedMod) || (modTime.Equal(selectedMod) && filepath.Base(match) > filepath.Base(selected)) {
				selected = match
				selectedMod = modTime
			}
		}
	}
	return selected
}

func buildProductReadinessReport(ctx context.Context, options productReadinessOptions, env []string) productReadinessReport {
	gatewayURL := firstNonEmpty(strings.TrimSpace(options.GatewayURL), "http://127.0.0.1:21080")
	deviceID := firstNonEmpty(strings.TrimSpace(options.DeviceID), "stackchan-001")
	report := productReadinessReport{
		SchemaVersion: "a21.product_readiness.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "blocked",
		SimulatorURL:  productSurfaceLabel(gatewayURL, "/simulator"),
		Gateway: productGatewayReadiness{
			URL: productSurfaceLabel(gatewayURL, ""),
		},
		StackChan: productStackChanReadiness{DeviceID: deviceID},
	}
	report.Gateway.Healthy = gatewayHealthOK(ctx, gatewayURL)
	report.Gateway.SimulatorReady = gatewaySimulatorOK(ctx, gatewayURL)
	report.Gateway.DeviceRegistry = false
	if report.Gateway.Healthy && report.Gateway.SimulatorReady {
		report.Gateway.Status = "ready"
	} else {
		report.Gateway.Status = "blocked"
		report.Findings = append(report.Findings, productReadinessFinding{Code: "gateway_not_ready", Message: "A21 Gateway or Simulator is not reachable", Detail: "run `make demo`"})
	}
	deviceReport, err := fetchFirmwareDeviceReport(gatewayURL)
	if err == nil {
		report.Gateway.DeviceRegistry = true
		report.StackChan = buildProductStackChanReadiness(deviceReport, deviceID)
	} else {
		report.StackChan.Status = "gateway_device_registry_unavailable"
		report.Findings = append(report.Findings, productReadinessFinding{Code: "device_registry_unavailable", Message: "A21 device registry is not reachable"})
	}
	report.Provider = buildProductProviderReadiness(env)
	report.V21 = buildProductV21Readiness(env)
	professionalEvidence, professionalFindings := loadProductV21ProfessionalReportEvidence(options.V21ProfessionalReport)
	report.Findings = append(report.Findings, professionalFindings...)
	if professionalEvidence.Valid {
		report.V21.Professional = professionalEvidence
		if professionalEvidence.AdapterExecuted {
			report.V21.QueryExecuted = true
		}
	}
	adapterSmokeEvidence, adapterSmokeFindings := loadProductV21AdapterSmokeReportEvidence(options.V21AdapterSmokeReport, report.V21)
	report.Findings = append(report.Findings, adapterSmokeFindings...)
	if adapterSmokeEvidence.Valid {
		report.V21.QueryExecuted = true
		report.V21.Professional = adapterSmokeEvidence
	}
	xiaozhiEvidence, xiaozhiFindings := loadProductXiaozhiReportEvidence(options.XiaozhiReport)
	report.Findings = append(report.Findings, xiaozhiFindings...)
	physicalEvidence, physicalFindings := loadProductPhysicalStackChanReportEvidence(options.PhysicalStackChanReport)
	report.StackChan.PhysicalEvidence = physicalEvidence
	report.Findings = append(report.Findings, physicalFindings...)
	report.Voice = buildProductVoiceReadiness(env, report.Provider, report.StackChan, xiaozhiEvidence)
	report.LaunchReady = report.Gateway.Healthy &&
		report.Provider.RealProviderReady &&
		report.V21.Healthy &&
		productV21ProfessionalReady(report.V21) &&
		report.StackChan.PhysicalDeviceOnline &&
		report.StackChan.PhysicalEvidence.PRDPhysicalAccepted &&
		report.Voice.ContinuousVoiceReady
	report.DemoReady = report.Gateway.Healthy && report.Gateway.SimulatorReady && report.Voice.LocalTTSReady
	report.NextActions = buildProductNextActions(report)
	for _, action := range report.NextActions {
		report.Findings = append(report.Findings, productReadinessFinding{Code: "launch_gap", Message: action})
	}
	if report.LaunchReady {
		report.Status = "real_launch_ready"
	} else if report.DemoReady {
		report.Status = "mock_demo_ready"
	} else {
		report.Status = "blocked"
	}
	return report
}

func buildProductProviderReadiness(env []string) productProviderReadiness {
	catalog := providers.ProviderCatalogFromEnv(env)
	readiness := productProviderReadiness{Primary: catalog.Primary}
	for _, provider := range catalog.Providers {
		if provider.Selected {
			readiness.Selected = provider.Name
			readiness.SelectedFamily = provider.Family
			readiness.SelectedConfigured = provider.Configured
			readiness.MissingEnv = append([]string(nil), provider.MissingEnv...)
			readiness.PresentEnv = append([]string(nil), provider.PresentEnv...)
		}
		if provider.Configured && provider.Name != "mock" && provider.RouteEligible {
			switch providers.ProviderFamily(provider.Family) {
			case providers.ProviderFamilyTextStream:
				readiness.TextStreamReady = true
			case providers.ProviderFamilyVoiceRealtime, providers.ProviderFamilyVoiceHybrid:
				readiness.VoiceRealtimeReady = true
			}
		}
	}
	readiness.RealProviderReady = readiness.Selected != "" &&
		readiness.Selected != "mock" &&
		readiness.SelectedConfigured &&
		(readiness.TextStreamReady || readiness.VoiceRealtimeReady)
	return readiness
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
	return v21.Healthy &&
		v21.Professional.Valid &&
		v21.Professional.CheckingAckWithin1200 &&
		v21.Professional.EvidenceAvailable &&
		v21.Professional.CardsAvailable &&
		v21.Professional.FollowUpsAvailable &&
		v21.Professional.AdapterExecuted
}

func productMicrophoneReady(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return status == "available" || strings.HasPrefix(status, "available_")
}

func buildProductVoiceReadiness(env []string, provider productProviderReadiness, stackchan productStackChanReadiness, evidenceList ...productXiaozhiReportEvidence) productVoiceReadiness {
	engine := firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_LOCAL_TTS_ENGINE")), "macos_say")
	asrProvider := firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_LOCAL_ASR_PROVIDER")), "mock_asr")
	selection := providers.VoicePipelineSelectionFromEnv(env)
	voicePipeline := productVoicePipelineReadiness{
		ASRProfile:           selection.ASRProfile,
		ASRProfileEnv:        selection.ASRProfileEnv,
		TextStreamProfile:    selection.LLMProfile,
		TextStreamProfileEnv: selection.LLMProfileEnv,
		TTSProfile:           selection.TTSProfile,
		TTSProfileEnv:        selection.TTSProfileEnv,
		ExecutionMode:        "env_configured",
	}
	var xiaozhiEvidence productXiaozhiReportEvidence
	if len(evidenceList) > 0 {
		xiaozhiEvidence = evidenceList[0]
	}
	if xiaozhiEvidence.Valid {
		voicePipeline.ExecutionMode = firstNonEmpty(xiaozhiEvidence.Execution.VoicePipelineExecutionMode, voicePipeline.ExecutionMode)
		voicePipeline.AcceptanceStatus = xiaozhiEvidence.AcceptanceStatus
		voicePipeline.AnswerFirstAudioP95MS = xiaozhiEvidence.AnswerFirstAudioP95MS
		voicePipeline.BargeInStopP95MS = xiaozhiEvidence.BargeInStopP95MS
		voicePipeline.FailureCount = xiaozhiEvidence.FailureCount
		voicePipeline.SourceReport = xiaozhiEvidence.SourceReport
		voicePipeline.PRDAccepted = xiaozhiEvidence.PRDAccepted
		if xiaozhiEvidence.Execution.ASRProfile != "" {
			voicePipeline.ASRProfile = xiaozhiEvidence.Execution.ASRProfile
		}
		if xiaozhiEvidence.Execution.ASRProfileEnv != "" {
			voicePipeline.ASRProfileEnv = xiaozhiEvidence.Execution.ASRProfileEnv
		}
		if xiaozhiEvidence.Execution.LLMProfile != "" {
			voicePipeline.TextStreamProfile = xiaozhiEvidence.Execution.LLMProfile
		}
		if xiaozhiEvidence.Execution.LLMProfileEnv != "" {
			voicePipeline.TextStreamProfileEnv = xiaozhiEvidence.Execution.LLMProfileEnv
		}
		if xiaozhiEvidence.Execution.TTSProfile != "" {
			voicePipeline.TTSProfile = xiaozhiEvidence.Execution.TTSProfile
		}
		if xiaozhiEvidence.Execution.TTSProfileEnv != "" {
			voicePipeline.TTSProfileEnv = xiaozhiEvidence.Execution.TTSProfileEnv
		}
		voicePipeline.HostLocalASRReady = xiaozhiEvidence.Execution.HostLocalASRExecuted
		voicePipeline.HostLocalTextReady = xiaozhiEvidence.Execution.HostLocalTextExecuted
		voicePipeline.HostLocalTTSReady = xiaozhiEvidence.Execution.HostLocalTTSExecuted
		hostLoopbackEvidenceAccepted := xiaozhiEvidence.AcceptanceStatus == "candidate_host_only" ||
			xiaozhiEvidence.AcceptanceStatus == "host_local_loopback_passed"
		voicePipeline.HostLoopbackCandidateReady = hostLoopbackEvidenceAccepted &&
			xiaozhiEvidence.AnswerFirstAudioP95MS > 0 &&
			xiaozhiEvidence.AnswerFirstAudioP95MS < 1500 &&
			xiaozhiEvidence.BargeInStopP95MS < 300 &&
			xiaozhiEvidence.FailureCount == 0 &&
			!xiaozhiEvidence.PRDAccepted
	} else {
		voicePipeline.HostLocalASRReady = selection.ASRProfile == "sherpa_onnx" && productSherpaASRReady(env)
		voicePipeline.HostLocalTextReady = provider.TextStreamReady
		voicePipeline.HostLocalTTSReady = (selection.TTSProfile == "sherpa_onnx_tts" || selection.TTSProfile == "sherpa_onnx") && productSherpaTTSReady(env)
	}
	readiness := productVoiceReadiness{
		LocalTTSEngine: engine,
		ASRProvider:    asrProvider,
		VoicePipeline:  voicePipeline,
	}
	switch engine {
	case "macos_say":
		_, err := exec.LookPath("say")
		readiness.LocalTTSReady = err == nil
	case "sherpa_onnx":
		readiness.LocalTTSReady = productSherpaTTSReady(env)
	default:
		readiness.LocalTTSReady = false
	}
	readiness.RealASRReady = asrProvider == "sherpa_onnx" && productSherpaASRReady(env)
	if voicePipeline.HostLocalASRReady && voicePipeline.ASRProfile != "" && voicePipeline.ASRProfile != "mock-local-asr" {
		readiness.RealASRReady = true
		readiness.ASRProvider = voicePipeline.ASRProfile
	}
	if voicePipeline.HostLocalTTSReady && !readiness.LocalTTSReady {
		readiness.LocalTTSReady = true
		readiness.LocalTTSEngine = voicePipeline.TTSProfile
	}
	readiness.ContinuousVoiceReady = readiness.LocalTTSReady &&
		readiness.RealASRReady &&
		provider.RealProviderReady &&
		stackchan.PhysicalDeviceOnline &&
		stackchan.PhysicalMicrophoneReady
	return readiness
}

type productXiaozhiReportEvidence struct {
	Valid                 bool
	SourceReport          string
	AcceptanceStatus      string
	PRDAccepted           bool
	AnswerFirstAudioP95MS float64
	BargeInStopP95MS      float64
	FailureCount          int
	Execution             providerLatencyBenchExecution
}

type productXiaozhiReportFixture struct {
	SchemaVersion    string                        `json:"schema_version"`
	ExecutionMode    string                        `json:"execution_mode"`
	BaselineScope    string                        `json:"baseline_scope"`
	AcceptanceStatus string                        `json:"acceptance_status"`
	PRDAccepted      bool                          `json:"prd_accepted"`
	Summary          productXiaozhiReportSummary   `json:"summary"`
	Counts           productXiaozhiReportCounts    `json:"counts"`
	Execution        providerLatencyBenchExecution `json:"execution"`
	Redaction        productXiaozhiReportRedaction `json:"redaction"`
}

type productXiaozhiReportSummary struct {
	AnswerFirstAudioP95MS *float64 `json:"answer_first_audio_total_p95_ms"`
	BargeInStopP95MS      *float64 `json:"barge_in_stop_p95_ms"`
}

type productXiaozhiReportCounts struct {
	FailureCount *int `json:"failure_count"`
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
	if fixture.SchemaVersion != "a21.xiaozhi_voice_bench.v1" ||
		fixture.ExecutionMode != "host_loopback" ||
		fixture.BaselineScope != "host_only" ||
		fixture.Execution.ProviderExecuted ||
		fixture.Execution.V21Executed ||
		fixture.Execution.HardwareExecuted ||
		*fixture.Redaction.PayloadsStored ||
		*fixture.Redaction.CredentialValuesStored ||
		*fixture.Redaction.FullURLsStored ||
		*fixture.Redaction.LocalPathsStored {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	execution := providerLatencyHostLoopbackExecution(fixture.Execution)
	return productXiaozhiReportEvidence{
		Valid:                 true,
		SourceReport:          filepath.Base(filepath.Clean(path)),
		AcceptanceStatus:      firstNonEmpty(strings.TrimSpace(fixture.AcceptanceStatus), "not_accepted"),
		PRDAccepted:           fixture.PRDAccepted,
		AnswerFirstAudioP95MS: *fixture.Summary.AnswerFirstAudioP95MS,
		BargeInStopP95MS:      *fixture.Summary.BargeInStopP95MS,
		FailureCount:          *fixture.Counts.FailureCount,
		Execution:             execution,
	}, nil
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
		Execution:             execution,
	}, nil
}

func missingProductLocalVoiceLoopbackReportField(fixture productLocalVoiceLoopbackReportFixture) string {
	switch {
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
	case fixture.Summary.AnswerFirstAudioP95MS == nil:
		return "summary.answer_first_audio_total_p95_ms"
	case fixture.Summary.BargeInStopP95MS == nil:
		return "summary.barge_in_stop_p95_ms"
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

func loadProductV21ProfessionalReportEvidence(path string) (productV21ProfessionalReadiness, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productV21ProfessionalReadiness{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productV21ProfessionalReportContainsForbiddenValue(raw) {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	var header struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	if strings.TrimSpace(header.SchemaVersion) == "a21.xiaozhi_professional_bench.v1" {
		return productXiaozhiProfessionalBenchReportEvidence(path, data)
	}
	var fixture productV21ProfessionalReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	if missingField := missingProductV21ProfessionalReportField(fixture); missingField != "" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{missingProductV21ProfessionalReportFieldFinding(missingField)}
	}
	if fixture.SchemaVersion != v21adapter.ProfessionalReadinessSchemaVersion ||
		strings.TrimSpace(fixture.Status) != "passed" ||
		!*fixture.CheckingAckWithin1200 ||
		!*fixture.EvidenceAvailable ||
		!*fixture.CardsAvailable ||
		!*fixture.FollowUpsAvailable ||
		*fixture.EvidenceCount <= 0 ||
		*fixture.CardCount <= 0 ||
		*fixture.FollowUpCount <= 0 ||
		*fixture.AdapterExecuted ||
		!*fixture.RedactionOK ||
		strings.TrimSpace(*fixture.ProfessionalAcceptanceStatus) == "" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	return productV21ProfessionalReadiness{
		Valid:                        true,
		CheckingAckWithin1200:        *fixture.CheckingAckWithin1200,
		EvidenceAvailable:            *fixture.EvidenceAvailable,
		CardsAvailable:               *fixture.CardsAvailable,
		FollowUpsAvailable:           *fixture.FollowUpsAvailable,
		EvidenceCount:                *fixture.EvidenceCount,
		CardCount:                    *fixture.CardCount,
		FollowUpCount:                *fixture.FollowUpCount,
		ProfessionalAcceptanceStatus: strings.TrimSpace(*fixture.ProfessionalAcceptanceStatus),
		SourceReport:                 filepath.Base(filepath.Clean(path)),
		AdapterExecuted:              *fixture.AdapterExecuted,
		PRDAccepted:                  false,
	}, nil
}

type productXiaozhiProfessionalBenchReportFixture struct {
	SchemaVersion                   string                                          `json:"schema_version"`
	SourceProfile                   string                                          `json:"source_profile"`
	AcceptanceStatus                string                                          `json:"acceptance_status"`
	PRDAccepted                     bool                                            `json:"prd_accepted"`
	CheckingFeedbackWithin1200      *bool                                           `json:"checking_feedback_within_1200"`
	ProfessionalResultObserved      *bool                                           `json:"professional_result_observed"`
	ProfessionalResultAfterChecking *bool                                           `json:"professional_result_after_checking"`
	EvidenceCount                   *int                                            `json:"evidence_count"`
	ScreenCardCount                 *int                                            `json:"screen_card_count"`
	FollowUpCount                   *int                                            `json:"follow_up_count"`
	ConfidencePresent               *bool                                           `json:"confidence_present"`
	NoPlaceholderUtterance          *bool                                           `json:"no_placeholder_utterance"`
	NoASRTextLeak                   *bool                                           `json:"no_asr_text_leak"`
	FailureCount                    *int                                            `json:"failure_count"`
	Execution                       productXiaozhiProfessionalBenchExecutionFixture `json:"execution"`
	Redaction                       productXiaozhiProfessionalBenchRedactionFixture `json:"redaction"`
}

type productXiaozhiProfessionalBenchExecutionFixture struct {
	ProviderExecuted *bool  `json:"provider_executed"`
	V21Executed      *bool  `json:"v21_executed"`
	HardwareExecuted *bool  `json:"hardware_executed"`
	GatewayRuntime   string `json:"gateway_runtime"`
}

type productXiaozhiProfessionalBenchRedactionFixture struct {
	PayloadsStored       *bool `json:"payloads_stored"`
	ASRTextStored        *bool `json:"asr_text_stored"`
	EvidenceBodyStored   *bool `json:"evidence_body_stored"`
	FullURLStored        *bool `json:"full_url_stored"`
	LocalPathStored      *bool `json:"local_path_stored"`
	PromptStored         *bool `json:"prompt_stored"`
	ProviderOutputStored *bool `json:"provider_output_stored"`
}

func productXiaozhiProfessionalBenchReportEvidence(path string, data []byte) (productV21ProfessionalReadiness, []productReadinessFinding) {
	var fixture productXiaozhiProfessionalBenchReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	if missingField := missingProductXiaozhiProfessionalBenchReportField(fixture); missingField != "" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{missingProductV21ProfessionalReportFieldFinding(missingField)}
	}
	if fixture.SchemaVersion != "a21.xiaozhi_professional_bench.v1" ||
		strings.TrimSpace(fixture.SourceProfile) != "external_gateway" ||
		strings.TrimSpace(fixture.AcceptanceStatus) != "external_gateway_ready" ||
		fixture.PRDAccepted ||
		!*fixture.CheckingFeedbackWithin1200 ||
		!*fixture.ProfessionalResultObserved ||
		!*fixture.ProfessionalResultAfterChecking ||
		*fixture.EvidenceCount <= 0 ||
		*fixture.ScreenCardCount <= 0 ||
		*fixture.FollowUpCount <= 0 ||
		!*fixture.ConfidencePresent ||
		!*fixture.NoPlaceholderUtterance ||
		!*fixture.NoASRTextLeak ||
		*fixture.FailureCount != 0 ||
		*fixture.Execution.ProviderExecuted ||
		!*fixture.Execution.V21Executed ||
		*fixture.Execution.HardwareExecuted ||
		strings.TrimSpace(fixture.Execution.GatewayRuntime) != "external_gateway" ||
		*fixture.Redaction.PayloadsStored ||
		*fixture.Redaction.ASRTextStored ||
		*fixture.Redaction.EvidenceBodyStored ||
		*fixture.Redaction.FullURLStored ||
		*fixture.Redaction.LocalPathStored ||
		*fixture.Redaction.PromptStored ||
		*fixture.Redaction.ProviderOutputStored {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	return productV21ProfessionalReadiness{
		Valid:                        true,
		CheckingAckWithin1200:        *fixture.CheckingFeedbackWithin1200,
		EvidenceAvailable:            *fixture.EvidenceCount > 0,
		CardsAvailable:               *fixture.ScreenCardCount > 0,
		FollowUpsAvailable:           *fixture.FollowUpCount > 0,
		EvidenceCount:                *fixture.EvidenceCount,
		CardCount:                    *fixture.ScreenCardCount,
		FollowUpCount:                *fixture.FollowUpCount,
		ProfessionalAcceptanceStatus: strings.TrimSpace(fixture.AcceptanceStatus),
		SourceReport:                 filepath.Base(filepath.Clean(path)),
		AdapterExecuted:              *fixture.Execution.V21Executed,
		PRDAccepted:                  false,
	}, nil
}

func missingProductXiaozhiProfessionalBenchReportField(fixture productXiaozhiProfessionalBenchReportFixture) string {
	switch {
	case fixture.CheckingFeedbackWithin1200 == nil:
		return "checking_feedback_within_1200"
	case fixture.ProfessionalResultObserved == nil:
		return "professional_result_observed"
	case fixture.ProfessionalResultAfterChecking == nil:
		return "professional_result_after_checking"
	case fixture.EvidenceCount == nil:
		return "evidence_count"
	case fixture.ScreenCardCount == nil:
		return "screen_card_count"
	case fixture.FollowUpCount == nil:
		return "follow_up_count"
	case fixture.ConfidencePresent == nil:
		return "confidence_present"
	case fixture.NoPlaceholderUtterance == nil:
		return "no_placeholder_utterance"
	case fixture.NoASRTextLeak == nil:
		return "no_asr_text_leak"
	case fixture.FailureCount == nil:
		return "failure_count"
	case fixture.Execution.ProviderExecuted == nil:
		return "execution.provider_executed"
	case fixture.Execution.V21Executed == nil:
		return "execution.v21_executed"
	case fixture.Execution.HardwareExecuted == nil:
		return "execution.hardware_executed"
	case fixture.Redaction.PayloadsStored == nil:
		return "redaction.payloads_stored"
	case fixture.Redaction.ASRTextStored == nil:
		return "redaction.asr_text_stored"
	case fixture.Redaction.EvidenceBodyStored == nil:
		return "redaction.evidence_body_stored"
	case fixture.Redaction.FullURLStored == nil:
		return "redaction.full_url_stored"
	case fixture.Redaction.LocalPathStored == nil:
		return "redaction.local_path_stored"
	case fixture.Redaction.PromptStored == nil:
		return "redaction.prompt_stored"
	case fixture.Redaction.ProviderOutputStored == nil:
		return "redaction.provider_output_stored"
	default:
		return ""
	}
}

func missingProductV21ProfessionalReportField(fixture productV21ProfessionalReportFixture) string {
	switch {
	case fixture.CheckingAckWithin1200 == nil:
		return "checking_ack_within_1200"
	case fixture.EvidenceAvailable == nil:
		return "evidence_available"
	case fixture.CardsAvailable == nil:
		return "cards_available"
	case fixture.FollowUpsAvailable == nil:
		return "follow_ups_available"
	case fixture.EvidenceCount == nil:
		return "evidence_count"
	case fixture.CardCount == nil:
		return "card_count"
	case fixture.FollowUpCount == nil:
		return "follow_up_count"
	case fixture.AdapterExecuted == nil:
		return "adapter_executed"
	case fixture.RedactionOK == nil:
		return "redaction_ok"
	case fixture.ProfessionalAcceptanceStatus == nil:
		return "professional_acceptance_status"
	default:
		return ""
	}
}

func missingProductV21ProfessionalReportFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "v21_professional_report_missing_field",
		Message: "V21 professional readiness report is missing a required field",
		Detail:  field,
	}
}

func invalidProductV21ProfessionalReportFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "v21_professional_report_invalid",
		Message: "V21 professional readiness report is invalid or unsafe",
	}
}

func productV21ProfessionalReportContainsForbiddenValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			if productV21ProfessionalReportContainsForbiddenValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productV21ProfessionalReportContainsForbiddenValue(child) {
				return true
			}
		}
	case string:
		lower := strings.ToLower(typed)
		for _, forbidden := range []string{
			"professional readiness fixture query",
			"raw evidence text",
			"raw retrieved evidence",
			"raw prompt text",
			"raw transcript text",
			"raw provider output",
			"provider output",
			"raw reasoning text",
			"http://",
			"https://",
			"/users/",
			"api_key",
			"secret-token",
		} {
			if strings.Contains(lower, forbidden) {
				return true
			}
		}
	}
	return false
}

type productV21AdapterSmokeReportFixture struct {
	SchemaVersion   string   `json:"schema_version"`
	Adapter         string   `json:"adapter"`
	Protocol        string   `json:"protocol"`
	Status          string   `json:"status"`
	Configured      *bool    `json:"configured"`
	Executed        *bool    `json:"executed"`
	QueryPath       string   `json:"query_path"`
	HealthPath      string   `json:"health_path"`
	EvidenceCount   *int     `json:"evidence_count"`
	ScreenCardCount *int     `json:"screen_card_count"`
	FollowUpCount   *int     `json:"follow_up_count"`
	Confidence      *float64 `json:"confidence"`
}

func loadProductV21AdapterSmokeReportEvidence(path string, v21 productV21Readiness) (productV21ProfessionalReadiness, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productV21ProfessionalReadiness{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productV21ProfessionalReportContainsForbiddenValue(raw) {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	var fixture productV21AdapterSmokeReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	if missingField := missingProductV21AdapterSmokeReportField(fixture); missingField != "" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{missingProductV21AdapterSmokeReportFieldFinding(missingField)}
	}
	if fixture.SchemaVersion != "a21.v21_adapter_smoke.v1" ||
		fixture.Adapter != "a21-v21-adapter" ||
		fixture.Protocol != "a21_v21_query" ||
		strings.TrimSpace(fixture.Status) != "passed" ||
		!*fixture.Configured ||
		!*fixture.Executed ||
		fixture.QueryPath != v21adapter.QueryPath ||
		fixture.HealthPath != v21adapter.HealthPath ||
		*fixture.EvidenceCount <= 0 ||
		*fixture.ScreenCardCount <= 0 ||
		*fixture.FollowUpCount <= 0 {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	return productV21ProfessionalReadiness{
		Valid:                        true,
		CheckingAckWithin1200:        v21.CheckingFeedbackSupported && v21.MaxFirstResponseMS <= v21adapter.ProfessionalMaxFirstResponseMS,
		EvidenceAvailable:            *fixture.EvidenceCount > 0,
		CardsAvailable:               *fixture.ScreenCardCount > 0,
		FollowUpsAvailable:           *fixture.FollowUpCount > 0,
		EvidenceCount:                *fixture.EvidenceCount,
		CardCount:                    *fixture.ScreenCardCount,
		FollowUpCount:                *fixture.FollowUpCount,
		ProfessionalAcceptanceStatus: "adapter_smoke_passed",
		SourceReport:                 filepath.Base(filepath.Clean(path)),
		AdapterExecuted:              true,
		PRDAccepted:                  false,
	}, nil
}

func missingProductV21AdapterSmokeReportField(fixture productV21AdapterSmokeReportFixture) string {
	switch {
	case fixture.Configured == nil:
		return "configured"
	case fixture.Executed == nil:
		return "executed"
	case fixture.EvidenceCount == nil:
		return "evidence_count"
	case fixture.ScreenCardCount == nil:
		return "screen_card_count"
	case fixture.FollowUpCount == nil:
		return "follow_up_count"
	default:
		return ""
	}
}

func missingProductV21AdapterSmokeReportFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "v21_adapter_smoke_report_missing_field",
		Message: "V21 adapter smoke report is missing a required field",
		Detail:  field,
	}
}

func invalidProductV21AdapterSmokeReportFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "v21_adapter_smoke_report_invalid",
		Message: "V21 adapter smoke report is invalid or unsafe",
	}
}

func productSherpaTTSReady(env []string) bool {
	modelDir := firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_SHERPA_ONNX_MODEL_DIR")), audio.DefaultSherpaONNXTTSModelDir())
	return audio.SherpaONNXTTSModelDirReady(modelDir)
}

func productSherpaASRReady(env []string) bool {
	modelDir := firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_SHERPA_ONNX_ASR_MODEL_DIR")), audio.DefaultSherpaONNXASRModelDir())
	family := strings.TrimSpace(appEnvValue(env, "A21_SHERPA_ONNX_ASR_FAMILY"))
	return audio.SherpaONNXASRModelDirReady(modelDir, family)
}

func buildProductNextActions(report productReadinessReport) []string {
	var actions []string
	if !report.Gateway.Healthy || !report.Gateway.SimulatorReady {
		actions = append(actions, "start A21 Gateway with `make demo`")
	}
	if !report.Provider.RealProviderReady {
		actions = append(actions, "configure a real A21 provider with A21_PROVIDER_PRIMARY plus its required env names")
	}
	if !report.V21.Healthy {
		actions = append(actions, "start/configure the A21 V21 adapter boundary with A21_V21_ADAPTER_URL")
	} else if !productV21ProfessionalReady(report.V21) {
		actions = append(actions, "run `go run ./cmd/a21 v21-adapter-smoke --execute --output-dir reports` and pass it to product-readiness with --v21-adapter-smoke-report")
	}
	if !report.StackChan.PhysicalDeviceOnline {
		actions = append(actions, "bring a physical StackChan online against the A21 Gateway")
	}
	if report.StackChan.PhysicalDeviceOnline && !report.StackChan.PhysicalMicrophoneReady {
		status := firstNonEmpty(report.StackChan.MicrophoneStatus, "unknown")
		actions = append(actions, "promote StackChan microphone to a product-ready firmware capability; current status: "+status)
	}
	if (report.StackChan.PhysicalDeviceOnline || report.StackChan.PhysicalEvidence.Valid) && !report.StackChan.PhysicalEvidence.PRDPhysicalAccepted {
		switch {
		case report.StackChan.PhysicalEvidence.CandidatePhysicalVoiceEvidence:
			actions = append(actions, productXiaozhiPhysicalEvidenceNextAction(report.StackChan.PhysicalEvidence))
		case report.StackChan.PhysicalEvidence.CandidatePhysicalEvidence:
			actions = append(actions, "complete human physical StackChan review for the candidate evidence report")
		case report.StackChan.PhysicalEvidence.HostLoopbackOnly:
			actions = append(actions, "collect physical StackChan evidence; host-loopback evidence is not production acceptance")
		default:
			actions = append(actions, "attach a PRD-accepted physical StackChan evidence report")
		}
	}
	if !report.Voice.RealASRReady {
		actions = append(actions, "install or configure real local ASR with A21_LOCAL_ASR_PROVIDER=sherpa_onnx and A21_SHERPA_ONNX_ASR_MODEL_DIR")
	}
	return actions
}

func productXiaozhiPhysicalEvidenceNextAction(physical productPhysicalStackChanReadiness) string {
	availability := physical.CanonicalMetricAvailability
	var missing []string
	if !availability["device_playback_start_ms"] {
		missing = append(missing, "device playback ack")
	}
	if !physical.OperatorInstrumentObservationAvailable {
		missing = append(missing, "operator audible observation or instrumented playback observation")
	}
	if !availability["device_downlink_first_frame_ms"] {
		missing = append(missing, "device downlink first-frame timing")
	}
	if !availability["speech_end_to_first_audible_response_ms"] {
		missing = append(missing, "speech-end to first audible response timing")
	}
	if !availability["barge_in_stop_ms"] {
		missing = append(missing, "barge-in stop timing")
	}
	if !availability["barge_in_playback_stop_done_ms"] {
		missing = append(missing, "barge-in playback stop_done")
	}
	if len(missing) == 0 {
		return "run three consecutive physical Xiaozhi PRD acceptance rounds and attach the evidence report"
	}
	return "collect missing " + strings.Join(missing, ", ") + " for the Xiaozhi physical evidence report"
}

func productSimulatorURL(gatewayURL string) string {
	base := strings.TrimRight(gatewayURL, "/")
	return base + "/simulator"
}

func productSurfaceLabel(gatewayURL string, suffix string) string {
	parsed, err := url.Parse(gatewayURL)
	if err != nil || parsed.Host == "" {
		return "invalid_gateway"
	}
	host := parsed.Hostname()
	port := parsed.Port()
	scope := "remote"
	switch host {
	case "127.0.0.1", "localhost", "::1":
		scope = "loopback"
	}
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	suffix = strings.TrimSpace(suffix)
	if suffix == "" || suffix == "/" {
		return scope + ":" + port
	}
	return scope + ":" + port + "/" + strings.TrimLeft(suffix, "/")
}

func gatewayHealthOK(ctx context.Context, gatewayURL string) bool {
	endpoint := strings.TrimRight(gatewayURL, "/") + "/healthz"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false
	}
	client := http.Client{Timeout: 700 * time.Millisecond, Transport: &http.Transport{Proxy: nil}}
	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK
}

func gatewaySimulatorOK(ctx context.Context, gatewayURL string) bool {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, productSimulatorURL(gatewayURL), nil)
	if err != nil {
		return false
	}
	client := http.Client{Timeout: 700 * time.Millisecond, Transport: &http.Transport{Proxy: nil}}
	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK
}

func openBrowserURL(rawURL string) {
	if strings.TrimSpace(rawURL) == "" {
		return
	}
	_ = exec.Command("open", rawURL).Start()
}

func writeProductReadinessReport(outputDir string, report productReadinessReport) (string, error) {
	if outputDir == "" {
		return "", nil
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-product-readiness-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONProductReadiness(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONProductReadiness(writer io.Writer, report productReadinessReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
