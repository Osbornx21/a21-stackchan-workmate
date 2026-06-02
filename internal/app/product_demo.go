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
	"sort"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
)

type productReadinessOptions struct {
	GatewayURL                 string
	Addr                       string
	DeviceID                   string
	OutputDir                  string
	ProviderSmokeReport        string
	ProviderRealtimeReport     string
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

type productV21ProfessionalExecutionReadiness struct {
	Valid                        bool   `json:"valid"`
	SourceKind                   string `json:"source_kind,omitempty"`
	SourceReport                 string `json:"source_report,omitempty"`
	QueryExecuted                bool   `json:"query_executed"`
	AdapterExecuted              bool   `json:"adapter_executed"`
	CheckingAckWithin1200        bool   `json:"checking_ack_within_1200"`
	EvidenceAvailable            bool   `json:"evidence_available"`
	CardsAvailable               bool   `json:"cards_available"`
	FollowUpsAvailable           bool   `json:"follow_ups_available"`
	EvidenceCount                int    `json:"evidence_count"`
	CardCount                    int    `json:"card_count"`
	FollowUpCount                int    `json:"follow_up_count"`
	ProfessionalAcceptanceStatus string `json:"professional_acceptance_status,omitempty"`
	RedactionOK                  bool   `json:"redaction_ok"`
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
	Status                       string   `json:"status"`
	CandidateReady               bool     `json:"candidate_ready"`
	AcceptanceStatus             string   `json:"acceptance_status"`
	PRDAccepted                  bool     `json:"prd_accepted"`
	GatewayReady                 bool     `json:"gateway_ready"`
	ProviderEvidenceReady        bool     `json:"provider_evidence_ready"`
	V21ProfessionalEvidenceReady bool     `json:"v21_professional_evidence_ready"`
	HostVoiceLoopbackReady       bool     `json:"host_voice_loopback_ready"`
	WakeWordReady                bool     `json:"wake_word_ready"`
	RequiresPhysicalAcceptance   bool     `json:"requires_physical_acceptance"`
	ProviderSmokeSourceReport    string   `json:"provider_smoke_source_report,omitempty"`
	V21ProfessionalSourceReport  string   `json:"v21_professional_source_report,omitempty"`
	HostVoiceSourceReport        string   `json:"host_voice_source_report,omitempty"`
	MissingEvidence              []string `json:"missing_evidence,omitempty"`
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
			fmt.Fprintln(stdout, "a21 product-readiness [--gateway-url http://127.0.0.1:21080] [--device-id stackchan-001] [--provider-smoke-report report.json] [--provider-realtime-fixture-report report.json] [--xiaozhi-report report.json] [--v21-professional-report report.json] [--v21-adapter-smoke-report report.json] [--physical-stackchan-report report.json] [--wake-word-firmware-plan report.json] [--wake-word-firmware-package-report report.json] [--wake-word-physical-acceptance-report report.json] [--use-latest-reports] [--output-dir reports] [--require-real]")
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
	if strings.TrimSpace(options.ProviderRealtimeReport) == "" {
		var findings []productReadinessFinding
		options.ProviderRealtimeReport, findings = latestAcceptedProductReadinessReportPath(reportDir, "provider_realtime_fixture", []string{
			"a21-provider-realtime-fixture-*.json",
		}, productLatestProviderRealtimeFixtureReportAccepted)
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
	selected, kind, findings := latestAcceptedProductReadinessReportAcrossKinds(reportDir, []productLatestReadinessReportKind{
		{
			Kind:     "v21_professional",
			Patterns: []string{"a21-xiaozhi-professional-bench-*.json", "a21-v21-professional-readiness-*.json"},
			Accept: func(path string) bool {
				evidence, _ := loadProductV21ProfessionalReportEvidence(path)
				return evidence.Valid && evidence.AdapterExecuted
			},
		},
		{
			Kind:     "v21_adapter_smoke",
			Patterns: []string{"a21-v21-adapter-smoke-*.json"},
			Accept: func(path string) bool {
				evidence, _ := loadProductV21AdapterSmokeReportEvidence(path, v21)
				return evidence.Valid && evidence.AdapterExecuted
			},
		},
	})
	switch kind {
	case "v21_professional":
		options.V21ProfessionalReport = selected
	case "v21_adapter_smoke":
		options.V21AdapterSmokeReport = selected
	}
	return findings
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
	report.Findings = append(report.Findings, options.LatestReportFindings...)
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
	wakeWord, wakeWordFindings := fetchProductWakeWordReadiness(ctx, gatewayURL)
	report.WakeWord = wakeWord
	report.Findings = append(report.Findings, wakeWordFindings...)
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
	professionalEvidence, professionalFindings := loadProductV21ProfessionalReportEvidence(options.V21ProfessionalReport)
	report.Findings = append(report.Findings, professionalFindings...)
	if professionalEvidence.Valid {
		report.V21.Professional = professionalEvidence
		if professionalEvidence.AdapterExecuted {
			report.V21.QueryExecuted = true
		}
		report.V21.ProfessionalExecution = buildProductV21ProfessionalExecutionReadiness(professionalEvidence)
	}
	adapterSmokeEvidence, adapterSmokeFindings := loadProductV21AdapterSmokeReportEvidence(options.V21AdapterSmokeReport, report.V21)
	report.Findings = append(report.Findings, adapterSmokeFindings...)
	if adapterSmokeEvidence.Valid {
		report.V21.QueryExecuted = true
		report.V21.Professional = adapterSmokeEvidence
		report.V21.ProfessionalExecution = buildProductV21ProfessionalExecutionReadiness(adapterSmokeEvidence)
	}
	xiaozhiEvidence, xiaozhiFindings := loadProductXiaozhiReportEvidence(options.XiaozhiReport)
	report.Findings = append(report.Findings, xiaozhiFindings...)
	physicalEvidence, physicalFindings := loadProductPhysicalStackChanReportEvidence(options.PhysicalStackChanReport)
	report.StackChan.PhysicalEvidence = physicalEvidence
	report.Findings = append(report.Findings, physicalFindings...)
	report.Voice = buildProductVoiceReadiness(env, report.Provider, report.StackChan, xiaozhiEvidence)
	report.ServerSide = buildProductServerSideReadiness(report)
	report.LaunchReady = report.Gateway.Healthy &&
		report.Provider.RealProviderReady &&
		report.V21.Healthy &&
		productV21ProfessionalReady(report.V21) &&
		report.StackChan.PhysicalDeviceOnline &&
		report.StackChan.PhysicalEvidence.PRDPhysicalAccepted &&
		report.Voice.ContinuousVoiceReady &&
		report.WakeWord.ProductReady
	report.DemoReady = report.Gateway.Healthy && report.Gateway.SimulatorReady && report.Voice.LocalTTSReady
	report.NextActions = buildProductNextActions(report)
	for _, action := range report.NextActions {
		report.Findings = append(report.Findings, productReadinessFinding{Code: "launch_gap", Message: action})
	}
	if report.LaunchReady {
		report.Status = "real_launch_ready"
	} else if report.ServerSide.CandidateReady {
		report.Status = "server_side_candidate_ready"
	} else if report.DemoReady {
		report.Status = "mock_demo_ready"
	} else {
		report.Status = "blocked"
	}
	report.CanonicalDecision = buildProductCanonicalReadinessDecision(report)
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
			readiness.SelectedRouteEligible = provider.RouteEligible
			readiness.MissingEnv = append([]string(nil), provider.MissingEnv...)
			readiness.PresentEnv = append([]string(nil), provider.PresentEnv...)
		}
	}
	return readiness
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
		readiness.SelectedConfigured &&
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
	client := http.Client{Timeout: 700 * time.Millisecond, Transport: &http.Transport{Proxy: nil}}
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

type productWakeWordFirmwarePlanEvidence struct {
	Valid                 bool
	Status                string
	SourceReport          string
	DryRun                bool
	FirmwareBuildRequired bool
	BuildAllowed          bool
	FlashAllowed          bool
	Mode                  string
	DesiredPhrase         string
	DesiredPinyin         string
	Threshold             int
}

type productWakeWordFirmwarePlanFixture struct {
	SchemaVersion            string `json:"schema_version"`
	Status                   string `json:"status"`
	DryRun                   *bool  `json:"dry_run"`
	FirmwareBuildRequired    *bool  `json:"firmware_build_required"`
	BuildAllowed             *bool  `json:"build_allowed"`
	FlashAllowed             *bool  `json:"flash_allowed"`
	FirmwareID               string `json:"firmware_id"`
	TargetBoard              string `json:"target_board"`
	TargetProfile            string `json:"target_profile"`
	GuardTier                string `json:"guard_tier"`
	Mode                     string `json:"mode"`
	DesiredPhrase            string `json:"desired_phrase"`
	DesiredPinyin            string `json:"desired_pinyin"`
	Threshold                int    `json:"threshold"`
	RuntimeStatus            string `json:"runtime_status"`
	NextRequiredConfirmation string `json:"next_required_confirmation"`
	ReportPath               string `json:"report_path"`
}

func loadProductWakeWordFirmwarePlanEvidence(path string) (productWakeWordFirmwarePlanEvidence, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productWakeWordFirmwarePlanEvidence{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productWakeWordFirmwarePlanContainsForbiddenValue(raw) {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	var fixture productWakeWordFirmwarePlanFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	if missingField := missingProductWakeWordFirmwarePlanField(fixture); missingField != "" {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{missingProductWakeWordFirmwarePlanFieldFinding(missingField)}
	}
	if !validProductWakeWordFirmwarePlanFixture(fixture) {
		return productWakeWordFirmwarePlanEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePlanFinding()}
	}
	return productWakeWordFirmwarePlanEvidence{
		Valid:                 true,
		Status:                strings.TrimSpace(fixture.Status),
		SourceReport:          filepath.Base(filepath.Clean(path)),
		DryRun:                *fixture.DryRun,
		FirmwareBuildRequired: *fixture.FirmwareBuildRequired,
		BuildAllowed:          *fixture.BuildAllowed,
		FlashAllowed:          *fixture.FlashAllowed,
		Mode:                  strings.TrimSpace(fixture.Mode),
		DesiredPhrase:         strings.TrimSpace(fixture.DesiredPhrase),
		DesiredPinyin:         strings.TrimSpace(fixture.DesiredPinyin),
		Threshold:             fixture.Threshold,
	}, nil
}

func attachProductWakeWordFirmwarePlan(readiness *productWakeWordReadiness, plan productWakeWordFirmwarePlanEvidence) []productReadinessFinding {
	if !matchingProductWakeWordFirmwarePlan(*readiness, plan) {
		return []productReadinessFinding{{
			Code:    "wake_word_firmware_plan_mismatch",
			Message: "Wake word firmware plan does not match the current Gateway wake-word intent",
			Detail:  plan.SourceReport,
		}}
	}
	readiness.FirmwarePlanAvailable = true
	readiness.FirmwarePlanStatus = plan.Status
	readiness.FirmwarePlanSource = plan.SourceReport
	readiness.FirmwarePlanDryRun = plan.DryRun
	readiness.FirmwarePlanBuild = plan.BuildAllowed
	readiness.FirmwarePlanFlash = plan.FlashAllowed
	return []productReadinessFinding{{
		Code:    "wake_word_firmware_plan_available",
		Message: "Wake word firmware plan evidence is available for the current stored intent",
		Detail:  plan.SourceReport,
	}}
}

func matchingProductWakeWordFirmwarePlan(readiness productWakeWordReadiness, plan productWakeWordFirmwarePlanEvidence) bool {
	if !readiness.Available || strings.TrimSpace(readiness.Mode) != plan.Mode {
		return false
	}
	if readiness.FirmwareBuildRequired != plan.FirmwareBuildRequired {
		return false
	}
	if readiness.Threshold != plan.Threshold {
		return false
	}
	if plan.Mode == wakeWordFirmwareCustomMode {
		return strings.TrimSpace(readiness.DesiredPhrase) == plan.DesiredPhrase &&
			strings.TrimSpace(readiness.DesiredPinyin) == plan.DesiredPinyin
	}
	return plan.Mode == wakeWordFirmwareBuiltinMode
}

func validProductWakeWordFirmwarePlanFixture(fixture productWakeWordFirmwarePlanFixture) bool {
	if fixture.SchemaVersion != wakeWordFirmwarePlanSchema ||
		!*fixture.DryRun ||
		*fixture.BuildAllowed ||
		*fixture.FlashAllowed ||
		strings.TrimSpace(fixture.FirmwareID) != wakeWordFirmwareID ||
		strings.TrimSpace(fixture.TargetBoard) != wakeWordFirmwareTargetBoard ||
		strings.TrimSpace(fixture.TargetProfile) != wakeWordFirmwareTargetProfile ||
		strings.TrimSpace(fixture.GuardTier) != wakeWordFirmwareGuardTier ||
		fixture.Threshold < 1 ||
		fixture.Threshold > 100 {
		return false
	}
	status := strings.TrimSpace(fixture.Status)
	mode := strings.TrimSpace(fixture.Mode)
	switch status {
	case "pending_firmware_build":
		return mode == wakeWordFirmwareCustomMode &&
			*fixture.FirmwareBuildRequired &&
			strings.TrimSpace(fixture.DesiredPhrase) != "" &&
			strings.TrimSpace(fixture.DesiredPinyin) != "" &&
			strings.TrimSpace(fixture.RuntimeStatus) == "pending_firmware_build" &&
			strings.TrimSpace(fixture.NextRequiredConfirmation) == wakeWordFirmwareBuildConfirm
	case "builtin_noop":
		return mode == wakeWordFirmwareBuiltinMode && !*fixture.FirmwareBuildRequired
	default:
		return false
	}
}

func missingProductWakeWordFirmwarePlanField(fixture productWakeWordFirmwarePlanFixture) string {
	switch {
	case fixture.DryRun == nil:
		return "dry_run"
	case fixture.FirmwareBuildRequired == nil:
		return "firmware_build_required"
	case fixture.BuildAllowed == nil:
		return "build_allowed"
	case fixture.FlashAllowed == nil:
		return "flash_allowed"
	default:
		return ""
	}
}

func missingProductWakeWordFirmwarePlanFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "wake_word_firmware_plan_missing_field",
		Message: "Wake word firmware plan report is missing a required field",
		Detail:  field,
	}
}

func invalidProductWakeWordFirmwarePlanFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "wake_word_firmware_plan_invalid",
		Message: "Wake word firmware plan report is invalid or unsafe",
	}
}

func productWakeWordFirmwarePlanContainsForbiddenValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			if productWakeWordFirmwarePlanContainsForbiddenValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productWakeWordFirmwarePlanContainsForbiddenValue(child) {
				return true
			}
		}
	case string:
		lower := strings.ToLower(typed)
		if containsLegacyIdentity(typed) {
			return true
		}
		for _, forbidden := range []string{"http://", "https://", "/users/", "secret", "token", "proxy", "transcript", "provider output", "raw audio"} {
			if strings.Contains(lower, forbidden) {
				return true
			}
		}
	}
	return false
}

type productWakeWordFirmwarePackageEvidence struct {
	Valid          bool
	Status         string
	SourceReport   string
	PackageWritten bool
	FlashAllowed   bool
	FlashExecuted  bool
	ProductReady   bool
	Mode           string
	DesiredPhrase  string
	DesiredPinyin  string
	Threshold      int
	ArtifactName   string
	ManifestName   string
}

type productWakeWordFirmwarePackageFixture struct {
	SchemaVersion    string                     `json:"schema_version"`
	GeneratedAtMS    *int64                     `json:"generated_at_ms"`
	Status           string                     `json:"status"`
	PackageWritten   *bool                      `json:"package_written"`
	FlashAllowed     *bool                      `json:"flash_allowed"`
	FlashExecuted    *bool                      `json:"flash_executed"`
	ProductReady     *bool                      `json:"product_ready"`
	FirmwareID       string                     `json:"firmware_id"`
	TargetBoard      string                     `json:"target_board"`
	TargetProfile    string                     `json:"target_profile"`
	Mode             string                     `json:"mode"`
	DesiredPhrase    string                     `json:"desired_phrase"`
	DesiredPinyin    string                     `json:"desired_pinyin"`
	Threshold        *int                       `json:"threshold"`
	Commit           string                     `json:"commit"`
	Timestamp        string                     `json:"timestamp"`
	SourcePlanReport string                     `json:"source_plan_report"`
	BuildReceipt     string                     `json:"build_receipt"`
	BuildDirName     string                     `json:"build_dir_name"`
	ArtifactName     string                     `json:"artifact_name"`
	SHA256Name       string                     `json:"sha256_name"`
	ManifestName     string                     `json:"manifest_name"`
	SHA256           string                     `json:"sha256"`
	Parts            []xiaozhiFirmwareFlashPart `json:"parts"`
	ReportPath       string                     `json:"report_path"`
}

func loadProductWakeWordFirmwarePackageEvidence(path string) (productWakeWordFirmwarePackageEvidence, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productWakeWordFirmwarePackageEvidence{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	if productWakeWordFirmwarePackageContainsForbiddenKey(raw) || productWakeWordFirmwarePackageContainsForbiddenValue(raw) {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	var fixture productWakeWordFirmwarePackageFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	if missingField := missingProductWakeWordFirmwarePackageField(fixture); missingField != "" {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{missingProductWakeWordFirmwarePackageFieldFinding(missingField)}
	}
	if !validProductWakeWordFirmwarePackageFixture(fixture) {
		return productWakeWordFirmwarePackageEvidence{}, []productReadinessFinding{invalidProductWakeWordFirmwarePackageFinding()}
	}
	return productWakeWordFirmwarePackageEvidence{
		Valid:          true,
		Status:         strings.TrimSpace(fixture.Status),
		SourceReport:   filepath.Base(filepath.Clean(path)),
		PackageWritten: *fixture.PackageWritten,
		FlashAllowed:   *fixture.FlashAllowed,
		FlashExecuted:  *fixture.FlashExecuted,
		ProductReady:   *fixture.ProductReady,
		Mode:           strings.TrimSpace(fixture.Mode),
		DesiredPhrase:  strings.TrimSpace(fixture.DesiredPhrase),
		DesiredPinyin:  strings.TrimSpace(fixture.DesiredPinyin),
		Threshold:      *fixture.Threshold,
		ArtifactName:   strings.TrimSpace(fixture.ArtifactName),
		ManifestName:   strings.TrimSpace(fixture.ManifestName),
	}, nil
}

func attachProductWakeWordFirmwarePackage(readiness *productWakeWordReadiness, evidence productWakeWordFirmwarePackageEvidence) []productReadinessFinding {
	if !matchingProductWakeWordFirmwarePackage(*readiness, evidence) {
		return []productReadinessFinding{{
			Code:    "wake_word_firmware_package_mismatch",
			Message: "Wake word firmware package does not match the current Gateway wake-word intent",
			Detail:  evidence.SourceReport,
		}}
	}
	readiness.FirmwarePackageAvailable = true
	readiness.FirmwarePackageStatus = evidence.Status
	readiness.FirmwarePackageSource = evidence.SourceReport
	readiness.FirmwarePackageArtifact = evidence.ArtifactName
	readiness.FirmwarePackageManifest = evidence.ManifestName
	readiness.FirmwarePackageWritten = evidence.PackageWritten
	readiness.FirmwarePackageFlash = evidence.FlashAllowed
	readiness.FirmwarePackageExecuted = evidence.FlashExecuted
	return []productReadinessFinding{{
		Code:    "wake_word_firmware_package_available",
		Message: "Wake word firmware package evidence is available, but guarded flash and physical custom wake proof are still required",
		Detail:  evidence.SourceReport,
	}}
}

func matchingProductWakeWordFirmwarePackage(readiness productWakeWordReadiness, evidence productWakeWordFirmwarePackageEvidence) bool {
	return readiness.Available &&
		readiness.FirmwareBuildRequired &&
		strings.TrimSpace(readiness.Mode) == evidence.Mode &&
		strings.TrimSpace(readiness.DesiredPhrase) == evidence.DesiredPhrase &&
		strings.TrimSpace(readiness.DesiredPinyin) == evidence.DesiredPinyin &&
		readiness.Threshold == evidence.Threshold
}

func validProductWakeWordFirmwarePackageFixture(fixture productWakeWordFirmwarePackageFixture) bool {
	commit := strings.ToLower(strings.TrimSpace(fixture.Commit))
	timestamp := strings.TrimSpace(fixture.Timestamp)
	artifactName := strings.TrimSpace(fixture.ArtifactName)
	expectedArtifactName := fmt.Sprintf("%s-%s-%s-%s.bin", wakeWordFirmwareArtifactPrefix, wakeWordFirmwareTargetBoard, commit, timestamp)
	if fixture.SchemaVersion != wakeWordFirmwarePackageSchema ||
		*fixture.GeneratedAtMS <= 0 ||
		strings.TrimSpace(fixture.Status) != "packaged" ||
		!*fixture.PackageWritten ||
		*fixture.FlashAllowed ||
		*fixture.FlashExecuted ||
		*fixture.ProductReady ||
		strings.TrimSpace(fixture.FirmwareID) != wakeWordFirmwareID ||
		strings.TrimSpace(fixture.TargetBoard) != wakeWordFirmwareTargetBoard ||
		strings.TrimSpace(fixture.TargetProfile) != wakeWordFirmwareTargetProfile ||
		strings.TrimSpace(fixture.Mode) != wakeWordFirmwareCustomMode ||
		strings.TrimSpace(fixture.DesiredPhrase) == "" ||
		strings.TrimSpace(fixture.DesiredPinyin) == "" ||
		*fixture.Threshold < 1 ||
		*fixture.Threshold > 100 ||
		!productWakeWordFirmwarePackageValidHex(commit, 7, 40) ||
		!productWakeWordFirmwarePackageValidTimestamp(timestamp) ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.SourcePlanReport, ".json") ||
		strings.TrimSpace(fixture.BuildReceipt) != "a21-wake-word-build.json" ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.BuildDirName, "") ||
		artifactName != expectedArtifactName ||
		!productWakeWordFirmwarePackageBasenameOK(artifactName, ".bin") ||
		strings.TrimSpace(fixture.SHA256Name) != artifactName+".sha256" ||
		strings.TrimSpace(fixture.ManifestName) != artifactName+".manifest.json" ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.ManifestName, ".json") ||
		!productWakeWordFirmwarePackageValidHex(strings.TrimSpace(fixture.SHA256), 64, 64) ||
		!productWakeWordFirmwarePackageBasenameOK(fixture.ReportPath, ".json") ||
		!validProductWakeWordFirmwarePackageParts(fixture.Parts) {
		return false
	}
	return true
}

func validProductWakeWordFirmwarePackageParts(parts []xiaozhiFirmwareFlashPart) bool {
	hasApp := false
	for _, part := range parts {
		if strings.TrimSpace(part.Name) == "" ||
			strings.TrimSpace(part.Offset) == "" ||
			!strings.HasPrefix(strings.TrimSpace(part.Offset), "0x") ||
			!productWakeWordFirmwarePackageBasenameOK(part.File, "") ||
			!productWakeWordFirmwarePackageValidHex(strings.TrimSpace(part.SHA256), 64, 64) ||
			part.SizeBytes <= 0 {
			return false
		}
		if strings.TrimSpace(part.Name) == "app" && strings.TrimSpace(part.File) == "xiaozhi.bin" {
			hasApp = true
		}
	}
	return hasApp
}

func missingProductWakeWordFirmwarePackageField(fixture productWakeWordFirmwarePackageFixture) string {
	switch {
	case fixture.GeneratedAtMS == nil:
		return "generated_at_ms"
	case fixture.PackageWritten == nil:
		return "package_written"
	case fixture.FlashAllowed == nil:
		return "flash_allowed"
	case fixture.FlashExecuted == nil:
		return "flash_executed"
	case fixture.ProductReady == nil:
		return "product_ready"
	case fixture.Threshold == nil:
		return "threshold"
	default:
		return ""
	}
}

func missingProductWakeWordFirmwarePackageFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "wake_word_firmware_package_missing_field",
		Message: "Wake word firmware package report is missing a required field",
		Detail:  field,
	}
}

func invalidProductWakeWordFirmwarePackageFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "wake_word_firmware_package_invalid",
		Message: "Wake word firmware package report is invalid or unsafe",
	}
}

func productWakeWordFirmwarePackageBasenameOK(value string, suffix string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "/") || strings.Contains(value, "\\") || strings.Contains(value, "..") || containsLegacyIdentity(value) {
		return false
	}
	if suffix != "" && !strings.HasSuffix(value, suffix) {
		return false
	}
	return value == filepath.Base(filepath.Clean(value))
}

func productWakeWordFirmwarePackageValidHex(value string, minLen int, maxLen int) bool {
	if len(value) < minLen || len(value) > maxLen {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func productWakeWordFirmwarePackageValidTimestamp(value string) bool {
	if len(value) != len("20260602-030000") || value[8] != '-' {
		return false
	}
	for index, char := range value {
		if index == 8 {
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func productWakeWordFirmwarePackageContainsForbiddenKey(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if forbiddenProductWakeWordFirmwarePackageKey(key) || productWakeWordFirmwarePackageContainsForbiddenKey(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productWakeWordFirmwarePackageContainsForbiddenKey(child) {
				return true
			}
		}
	}
	return false
}

func forbiddenProductWakeWordFirmwarePackageKey(key string) bool {
	normalized := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(strings.TrimSpace(key)))
	switch normalized {
	case "raw_pcm", "raw_audio", "pcm_bytes", "data_base64", "audio_base64", "base64_audio",
		"prompt", "transcript", "provider_output", "reasoning", "credential_values",
		"api_key", "access_token", "token", "full_url", "url", "proxy_url", "local_path":
		return true
	default:
		return false
	}
}

func productWakeWordFirmwarePackageContainsForbiddenValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			if productWakeWordFirmwarePackageContainsForbiddenValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productWakeWordFirmwarePackageContainsForbiddenValue(child) {
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

type productWakeWordPhysicalAcceptanceEvidence struct {
	Valid                                bool
	Status                               string
	ProductReady                         bool
	SourceReport                         string
	Mode                                 string
	DesiredPhrase                        string
	DesiredPinyin                        string
	Threshold                            int
	PackageSourceReport                  string
	PackageArtifactName                  string
	PackageManifestName                  string
	PhysicalDeviceOnline                 bool
	FirmwareFlashExecuted                bool
	GuardedFlashReportSource             string
	OperatorCustomWakeObservationPresent bool
	WakePhraseMatched                    bool
	FalseWakeAccepted                    bool
	StockWakeAccepted                    bool
	RedactionOK                          bool
}

type productWakeWordPhysicalAcceptanceFixture struct {
	SchemaVersion                        string `json:"schema_version"`
	GeneratedAtMS                        *int64 `json:"generated_at_ms"`
	Status                               string `json:"status"`
	ProductReady                         *bool  `json:"product_ready"`
	Mode                                 string `json:"mode"`
	DesiredPhrase                        string `json:"desired_phrase"`
	DesiredPinyin                        string `json:"desired_pinyin"`
	Threshold                            *int   `json:"threshold"`
	PackageSourceReport                  string `json:"package_source_report"`
	PackageArtifactName                  string `json:"package_artifact_name"`
	PackageManifestName                  string `json:"package_manifest_name"`
	PhysicalDeviceOnline                 *bool  `json:"physical_device_online"`
	FirmwareFlashExecuted                *bool  `json:"firmware_flash_executed"`
	GuardedFlashReportSource             string `json:"guarded_flash_report_source"`
	OperatorCustomWakeObservationPresent *bool  `json:"operator_custom_wake_observation_present"`
	WakePhraseMatched                    *bool  `json:"wake_phrase_matched"`
	FalseWakeAccepted                    *bool  `json:"false_wake_accepted"`
	StockWakeAccepted                    *bool  `json:"stock_wake_accepted"`
	RedactionOK                          *bool  `json:"redaction_ok"`
	ReportPath                           string `json:"report_path"`
}

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
		CheckingAckWithin1200:        professional.CheckingAckWithin1200,
		EvidenceAvailable:            professional.EvidenceAvailable,
		CardsAvailable:               professional.CardsAvailable,
		FollowUpsAvailable:           professional.FollowUpsAvailable,
		EvidenceCount:                professional.EvidenceCount,
		CardCount:                    professional.CardCount,
		FollowUpCount:                professional.FollowUpCount,
		ProfessionalAcceptanceStatus: professional.ProfessionalAcceptanceStatus,
		RedactionOK:                  true,
		PRDAccepted:                  false,
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
		Status:                      "blocked",
		AcceptanceStatus:            "server_side_blocked",
		ProviderSmokeSourceReport:   report.Provider.SmokeSourceReport,
		V21ProfessionalSourceReport: report.V21.Professional.SourceReport,
		HostVoiceSourceReport:       report.Voice.VoicePipeline.SourceReport,
		RequiresPhysicalAcceptance:  !report.StackChan.PhysicalEvidence.PRDPhysicalAccepted,
	}
	readiness.GatewayReady = report.Gateway.Healthy && report.Gateway.SimulatorReady
	readiness.ProviderEvidenceReady = report.Provider.RealProviderReady &&
		report.Provider.SmokeEvidenceValid &&
		report.Provider.SmokeExecuted
	readiness.V21ProfessionalEvidenceReady = productV21ProfessionalReady(report.V21)
	readiness.HostVoiceLoopbackReady = report.Voice.VoicePipeline.HostProductChainReady
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
	if !readiness.HostVoiceLoopbackReady {
		readiness.MissingEvidence = append(readiness.MissingEvidence, "host_voice_loopback")
	}
	if !readiness.WakeWordReady {
		readiness.MissingEvidence = append(readiness.MissingEvidence, "wake_word")
	}
	readiness.CandidateReady = len(readiness.MissingEvidence) == 0
	if readiness.CandidateReady {
		readiness.Status = "candidate_ready"
		readiness.AcceptanceStatus = "server_side_candidate_ready"
	}
	return readiness
}

func buildProductCanonicalReadinessDecision(report productReadinessReport) productCanonicalReadinessDecision {
	status := "blocked"
	switch {
	case report.LaunchReady:
		status = "prd_accepted"
	case report.ServerSide.CandidateReady:
		status = "server_side_candidate_only"
	}
	return productCanonicalReadinessDecision{
		Authority:                  "a21.product_readiness.v1",
		FullPRDStatus:              status,
		LaunchReady:                report.LaunchReady,
		PRDAccepted:                report.LaunchReady,
		ServerSideCandidateReady:   report.ServerSide.CandidateReady,
		HostOnlyEvidenceUse:        "gap_reduction_only",
		RequiresPhysicalAcceptance: !report.StackChan.PhysicalEvidence.PRDPhysicalAccepted,
		MissingRealEvidence:        productMissingRealEvidence(report),
		MissingReportFields:        productMissingReportFields(report.Findings),
	}
}

func productMissingRealEvidence(report productReadinessReport) []string {
	var missing []string
	if !report.Gateway.Healthy || !report.Gateway.SimulatorReady {
		missing = append(missing, "gateway")
	}
	if !report.Provider.RealProviderReady || !report.Provider.SmokeEvidenceValid || !report.Provider.SmokeExecuted {
		missing = append(missing, "real_provider_smoke")
	}
	if !productV21ProfessionalReady(report.V21) {
		missing = append(missing, "v21_professional_execution")
	}
	if !report.StackChan.PhysicalDeviceOnline {
		missing = append(missing, "physical_stackchan_online")
	}
	if !report.StackChan.PhysicalEvidence.PRDPhysicalAccepted {
		missing = append(missing, "physical_stackchan_prd_acceptance")
	}
	if !report.Voice.ContinuousVoiceReady {
		missing = append(missing, "continuous_voice_pipeline")
	}
	if !report.WakeWord.ProductReady {
		missing = append(missing, "wake_word_product_ready")
	}
	return missing
}

func productMissingReportFields(findings []productReadinessFinding) []string {
	var fields []string
	for _, finding := range findings {
		prefix := productMissingReportFieldPrefix(finding.Code)
		if prefix == "" || strings.TrimSpace(finding.Detail) == "" {
			continue
		}
		fields = append(fields, prefix+":"+strings.TrimSpace(finding.Detail))
	}
	return fields
}

func productMissingReportFieldPrefix(code string) string {
	switch code {
	case "provider_smoke_report_missing_field":
		return "provider_smoke"
	case "provider_realtime_fixture_report_missing_field":
		return "provider_realtime_fixture"
	case "xiaozhi_report_missing_field":
		return "xiaozhi_report"
	case "wake_word_firmware_plan_missing_field":
		return "wake_word_firmware_plan"
	case "wake_word_firmware_package_missing_field":
		return "wake_word_firmware_package"
	case "wake_word_physical_acceptance_missing_field":
		return "wake_word_physical_acceptance"
	case "v21_professional_report_missing_field":
		return "v21_professional_report"
	case "v21_adapter_smoke_report_missing_field":
		return "v21_adapter_smoke"
	default:
		return ""
	}
}

func productMicrophoneReady(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return status == "available" || strings.HasPrefix(status, "available_")
}

func buildProductVoiceReadiness(env []string, provider productProviderReadiness, stackchan productStackChanReadiness, evidenceList ...productXiaozhiReportEvidence) productVoiceReadiness {
	engine := firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_LOCAL_TTS_ENGINE")), "macos_say")
	asrProvider := firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_LOCAL_ASR_PROVIDER")), defaultProductASRProvider(env))
	selection := providers.VoicePipelineSelectionFromEnv(env)
	if strings.TrimSpace(appEnvValue(env, "A21_LOCAL_TTS_ENGINE")) == "" && selection.TTSProfile == "voice_clone_cli" {
		engine = "voice_clone_cli"
	}
	voicePipeline := productVoicePipelineReadiness{
		ASRProfile:                   selection.ASRProfile,
		ASRProfileEnv:                selection.ASRProfileEnv,
		TextStreamProfile:            selection.LLMProfile,
		TextStreamProfileEnv:         selection.LLMProfileEnv,
		TextStreamFallbackProfile:    selection.LLMFallbackProfile,
		TextStreamFallbackProfileEnv: selection.LLMFallbackProfileEnv,
		TextStreamFallbackReady:      productTextStreamProviderReady(env, selection.LLMFallbackProfile),
		TTSProfile:                   selection.TTSProfile,
		TTSProfileEnv:                selection.TTSProfileEnv,
		ExecutionMode:                "env_configured",
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
		voicePipeline.HostProductChainReady = voicePipeline.HostLoopbackCandidateReady &&
			xiaozhiEvidence.HostProductChainReady
	} else {
		if selection.ASRProfile == "sherpa_onnx" || (selection.ASRProfile == "mock-local-asr" && asrProvider == "sherpa_onnx") {
			voicePipeline.ASRProfile = "sherpa_onnx"
			voicePipeline.ASRProfileEnv = "A21_SHERPA_ONNX_ASR_MODEL_DIR"
			voicePipeline.HostLocalASRReady = productSherpaASRReady(env)
		}
		voicePipeline.HostLocalTextReady = provider.TextStreamReady
		voicePipeline.HostLocalTTSReady =
			((selection.TTSProfile == "sherpa_onnx_tts" || selection.TTSProfile == "sherpa_onnx") && productSherpaTTSReady(env)) ||
				(selection.TTSProfile == "voice_clone_cli" && productVoiceCloneTTSReady(env))
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
	case "voice_clone_cli":
		readiness.LocalTTSReady = productVoiceCloneTTSReady(env)
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
	if voicePipeline.HostProductChainReady {
		readiness.ContinuousVoiceReady = true
	}
	return readiness
}

func defaultProductASRProvider(env []string) string {
	if productSherpaASRReady(env) {
		return "sherpa_onnx"
	}
	return "mock_asr"
}

type productXiaozhiReportEvidence struct {
	Valid                 bool
	SourceReport          string
	AcceptanceStatus      string
	PRDAccepted           bool
	AnswerFirstAudioP95MS float64
	BargeInStopP95MS      float64
	FailureCount          int
	HostProductChainReady bool
	Execution             providerLatencyBenchExecution
}

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
	derived := providerLatencySafeExecutionMode(execution.VoicePipelineExecutionMode) == "host_local" &&
		execution.VoicePipelineObserved &&
		execution.HostLocalASRExecuted &&
		execution.HostLocalTextExecuted &&
		execution.HostLocalTTSExecuted &&
		providerLatencySafeIdentifier(execution.ASRProfile, false) != "" &&
		providerLatencySafeIdentifier(execution.LLMProfile, false) != "" &&
		providerLatencySafeIdentifier(execution.TTSProfile, false) != ""
	if explicit != nil {
		return *explicit && derived
	}
	return derived
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
	SchemaVersion      string   `json:"schema_version"`
	GeneratedAtMS      *int64   `json:"generated_at_ms"`
	Adapter            string   `json:"adapter"`
	Protocol           string   `json:"protocol"`
	Status             string   `json:"status"`
	Configured         *bool    `json:"configured"`
	Executed           *bool    `json:"executed"`
	Mode               string   `json:"mode"`
	LatencyProfile     string   `json:"latency_profile"`
	AnswerStyle        string   `json:"answer_style"`
	PrivacyScope       string   `json:"privacy_scope"`
	MaxFirstResponseMS *int     `json:"max_first_response_ms"`
	QueryPath          string   `json:"query_path"`
	HealthPath         string   `json:"health_path"`
	EvidenceCount      *int     `json:"evidence_count"`
	SpeechBlockCount   *int     `json:"speech_block_count"`
	ScreenCardCount    *int     `json:"screen_card_count"`
	FollowUpCount      *int     `json:"follow_up_count"`
	Confidence         *float64 `json:"confidence"`
	RedactionOK        *bool    `json:"redaction_ok"`
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
	if fixture.SchemaVersion != v21adapter.SmokeSchemaVersion ||
		fixture.GeneratedAtMS == nil ||
		*fixture.GeneratedAtMS <= 0 ||
		fixture.Adapter != "a21-v21-adapter" ||
		fixture.Protocol != "a21_v21_query" ||
		strings.TrimSpace(fixture.Status) != "passed" ||
		!*fixture.Configured ||
		!*fixture.Executed ||
		strings.TrimSpace(fixture.Mode) != "professional" ||
		strings.TrimSpace(fixture.LatencyProfile) != "fast_first" ||
		strings.TrimSpace(fixture.AnswerStyle) != "voice_first_with_citations" ||
		strings.TrimSpace(fixture.PrivacyScope) != "professional_only" ||
		*fixture.MaxFirstResponseMS != v21adapter.ProfessionalMaxFirstResponseMS ||
		fixture.QueryPath != v21adapter.QueryPath ||
		fixture.HealthPath != v21adapter.HealthPath ||
		*fixture.Confidence <= 0 ||
		*fixture.Confidence > 1 ||
		*fixture.EvidenceCount <= 0 ||
		*fixture.SpeechBlockCount <= 0 ||
		*fixture.ScreenCardCount <= 0 ||
		*fixture.FollowUpCount <= 0 ||
		!*fixture.RedactionOK {
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
	case fixture.GeneratedAtMS == nil:
		return "generated_at_ms"
	case fixture.Configured == nil:
		return "configured"
	case fixture.Executed == nil:
		return "executed"
	case strings.TrimSpace(fixture.Mode) == "":
		return "mode"
	case strings.TrimSpace(fixture.LatencyProfile) == "":
		return "latency_profile"
	case strings.TrimSpace(fixture.AnswerStyle) == "":
		return "answer_style"
	case strings.TrimSpace(fixture.PrivacyScope) == "":
		return "privacy_scope"
	case fixture.MaxFirstResponseMS == nil:
		return "max_first_response_ms"
	case fixture.Confidence == nil:
		return "confidence"
	case fixture.EvidenceCount == nil:
		return "evidence_count"
	case fixture.SpeechBlockCount == nil:
		return "speech_block_count"
	case fixture.ScreenCardCount == nil:
		return "screen_card_count"
	case fixture.FollowUpCount == nil:
		return "follow_up_count"
	case fixture.RedactionOK == nil:
		return "redaction_ok"
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

func productVoiceCloneTTSReady(env []string) bool {
	command := strings.TrimSpace(appEnvValue(env, "A21_VOICE_CLONE_COMMAND"))
	refAudio := strings.TrimSpace(appEnvValue(env, "A21_VOICE_CLONE_REF_AUDIO"))
	if command == "" || refAudio == "" {
		return false
	}
	if containsLegacyIdentityPathToken(command) || containsLegacyIdentityPathToken(refAudio) {
		return false
	}
	if _, err := os.Stat(command); err != nil {
		if _, lookErr := exec.LookPath(command); lookErr != nil {
			return false
		}
	}
	info, err := os.Stat(refAudio)
	return err == nil && info.Mode().IsRegular()
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
		if report.Provider.Selected != "" && report.Provider.Selected != "mock" && report.Provider.SelectedConfigured {
			actions = append(actions, fmt.Sprintf("run `go run ./cmd/a21 provider-smoke --provider %s --execute --stream --repeat 3 --output-dir reports` and pass it to product-readiness with --provider-smoke-report", report.Provider.Selected))
		} else {
			actions = append(actions, "configure a real A21 provider with A21_PROVIDER_PRIMARY plus its required env names")
		}
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
	if !report.WakeWord.ProductReady {
		if report.WakeWord.FirmwareBuildRequired {
			if report.WakeWord.FirmwarePackageAvailable {
				actions = append(actions, "use the wake word firmware package in a guarded hardware-window flash plan and collect physical custom wake proof")
			} else if report.WakeWord.FirmwarePlanAvailable {
				actions = append(actions, "use the wake word firmware plan to prepare the guarded build/package and hardware-window flash")
			} else {
				actions = append(actions, "run wake word firmware plan with `go run ./cmd/a21 wake-word-firmware-plan --output-dir reports` for the stored custom MultiNet profile")
			}
		} else {
			actions = append(actions, "restore A21 Gateway wake word readiness before launch")
		}
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
