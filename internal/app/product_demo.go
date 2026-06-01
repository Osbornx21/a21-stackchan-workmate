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
	GatewayURL    string
	Addr          string
	DeviceID      string
	OutputDir     string
	XiaozhiReport string
	RequireReal   bool
	OpenBrowser   bool
	StatusOnly    bool
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
	Configured                bool   `json:"configured"`
	Healthy                   bool   `json:"healthy"`
	Status                    string `json:"status"`
	ProfessionalBridgeReady   bool   `json:"professional_bridge_ready"`
	CheckingFeedbackSupported bool   `json:"checking_feedback_supported"`
	MaxFirstResponseMS        int    `json:"max_first_response_ms"`
	EvidenceContractReady     bool   `json:"evidence_contract_ready"`
	QueryExecuted             bool   `json:"query_executed"`
	QueryPath                 string `json:"query_path"`
	HealthPath                string `json:"health_path"`
}

type productStackChanReadiness struct {
	DeviceID                string `json:"device_id"`
	PhysicalDeviceOnline    bool   `json:"physical_device_online"`
	PhysicalMicrophoneReady bool   `json:"physical_microphone_ready"`
	MicrophoneStatus        string `json:"microphone_status,omitempty"`
	SimulatorDeviceOnline   bool   `json:"simulator_device_online"`
	USBSerialCandidateCount int    `json:"usb_serial_candidate_count"`
	Status                  string `json:"status"`
}

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
			fmt.Fprintln(stdout, "a21 product-readiness [--gateway-url http://127.0.0.1:21080] [--device-id stackchan-001] [--xiaozhi-report report.json] [--output-dir reports] [--require-real]")
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
	xiaozhiEvidence, xiaozhiFindings := loadProductXiaozhiReportEvidence(options.XiaozhiReport)
	report.Findings = append(report.Findings, xiaozhiFindings...)
	report.Voice = buildProductVoiceReadiness(env, report.Provider, report.StackChan, xiaozhiEvidence)
	report.LaunchReady = report.Gateway.Healthy &&
		report.Provider.RealProviderReady &&
		report.V21.Healthy &&
		report.StackChan.PhysicalDeviceOnline &&
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
	readiness := productStackChanReadiness{DeviceID: deviceID}
	for _, device := range deviceReport.Devices {
		online := device.ConnectionStatus == "" || device.ConnectionStatus == "online"
		if device.DeviceID == deviceID && online && !strings.Contains(strings.ToLower(device.DeviceID), "sim") {
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
	case readiness.SimulatorDeviceOnline:
		readiness.Status = "simulator_only"
	default:
		readiness.Status = "physical_offline"
	}
	return readiness
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
		voicePipeline.HostLoopbackCandidateReady = xiaozhiEvidence.AcceptanceStatus == "candidate_host_only" &&
			xiaozhiEvidence.AnswerFirstAudioP95MS > 0 &&
			xiaozhiEvidence.AnswerFirstAudioP95MS < 1500 &&
			xiaozhiEvidence.BargeInStopP95MS < 300 &&
			xiaozhiEvidence.FailureCount == 0 &&
			!xiaozhiEvidence.PRDAccepted
	} else {
		voicePipeline.HostLocalASRReady = selection.ASRProfile == "sherpa_onnx" && productSherpaASRReady(env)
		voicePipeline.HostLocalTextReady = selection.LLMProfile != "" && selection.LLMProfile != "mock"
		voicePipeline.HostLocalTTSReady = selection.TTSProfile == "sherpa_onnx_tts" || selection.TTSProfile == "sherpa_onnx"
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
	AnswerFirstAudioP95MS float64 `json:"answer_first_audio_total_p95_ms"`
	BargeInStopP95MS      float64 `json:"barge_in_stop_p95_ms"`
}

type productXiaozhiReportCounts struct {
	FailureCount int `json:"failure_count"`
}

type productXiaozhiReportRedaction struct {
	PayloadsStored         bool `json:"payloads_stored"`
	CredentialValuesStored bool `json:"credential_values_stored"`
	FullURLsStored         bool `json:"full_urls_stored"`
	LocalPathsStored       bool `json:"local_paths_stored"`
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
	var fixture productXiaozhiReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	if fixture.SchemaVersion != "a21.xiaozhi_voice_bench.v1" ||
		fixture.ExecutionMode != "host_loopback" ||
		fixture.BaselineScope != "host_only" ||
		fixture.Execution.ProviderExecuted ||
		fixture.Execution.V21Executed ||
		fixture.Execution.HardwareExecuted ||
		fixture.Redaction.PayloadsStored ||
		fixture.Redaction.CredentialValuesStored ||
		fixture.Redaction.FullURLsStored ||
		fixture.Redaction.LocalPathsStored {
		return productXiaozhiReportEvidence{}, []productReadinessFinding{invalidProductXiaozhiReportFinding()}
	}
	execution := providerLatencyHostLoopbackExecution(fixture.Execution)
	return productXiaozhiReportEvidence{
		Valid:                 true,
		SourceReport:          filepath.Base(filepath.Clean(path)),
		AcceptanceStatus:      firstNonEmpty(strings.TrimSpace(fixture.AcceptanceStatus), "not_accepted"),
		PRDAccepted:           fixture.PRDAccepted,
		AnswerFirstAudioP95MS: fixture.Summary.AnswerFirstAudioP95MS,
		BargeInStopP95MS:      fixture.Summary.BargeInStopP95MS,
		FailureCount:          fixture.Counts.FailureCount,
		Execution:             execution,
	}, nil
}

func invalidProductXiaozhiReportFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "xiaozhi_report_invalid",
		Message: "Xiaozhi host-loopback report is invalid or unsafe",
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
	}
	if !report.StackChan.PhysicalDeviceOnline {
		actions = append(actions, "bring a physical StackChan online against the A21 Gateway")
	}
	if report.StackChan.PhysicalDeviceOnline && !report.StackChan.PhysicalMicrophoneReady {
		status := firstNonEmpty(report.StackChan.MicrophoneStatus, "unknown")
		actions = append(actions, "promote StackChan microphone to a product-ready firmware capability; current status: "+status)
	}
	if !report.Voice.RealASRReady {
		actions = append(actions, "install or configure real local ASR with A21_LOCAL_ASR_PROVIDER=sherpa_onnx and A21_SHERPA_ONNX_ASR_MODEL_DIR")
	}
	return actions
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
