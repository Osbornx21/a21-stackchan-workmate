package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type serverSideReadinessBundleOptions struct {
	GatewayURL            string
	DeviceID              string
	OutputDir             string
	ProviderSmokeReport   string
	XiaozhiReport         string
	V21ProfessionalReport string
	V21AdapterSmokeReport string
	WakeWordFirmwarePlan  string
	UseLatestReports      bool
	RequireCandidate      bool
	CollectMissing        bool
	ExecuteProviderSmoke  bool
	ExecuteV21Smoke       bool
	CollectRepeat         int
}

type serverSideReadinessBundleReport struct {
	SchemaVersion              string                             `json:"schema_version"`
	GeneratedAtMS              int64                              `json:"generated_at_ms"`
	Status                     string                             `json:"status"`
	CandidateReady             bool                               `json:"candidate_ready"`
	ProductReadinessStatus     string                             `json:"product_readiness_status"`
	LaunchReady                bool                               `json:"launch_ready"`
	PRDAccepted                bool                               `json:"prd_accepted"`
	RequiresPhysicalAcceptance bool                               `json:"requires_physical_acceptance"`
	Gateway                    serverSideReadinessBundleEvidence  `json:"gateway"`
	Provider                   serverSideReadinessBundleEvidence  `json:"provider"`
	V21                        serverSideReadinessBundleEvidence  `json:"v21"`
	HostVoice                  serverSideReadinessBundleEvidence  `json:"host_voice"`
	WakeWord                   serverSideReadinessBundleEvidence  `json:"wake_word"`
	ServerSide                 productServerSideReadiness         `json:"server_side"`
	Collection                 serverSideReadinessCollection      `json:"collection"`
	MissingEvidence            []string                           `json:"missing_evidence,omitempty"`
	CollectionCommands         []string                           `json:"collection_commands,omitempty"`
	NextActions                []string                           `json:"next_actions,omitempty"`
	Redaction                  serverSideReadinessBundleRedaction `json:"redaction"`
	ReportPath                 string                             `json:"report_path,omitempty"`
}

type serverSideReadinessCollection struct {
	Enabled bool                                `json:"enabled"`
	Status  string                              `json:"status"`
	Steps   []serverSideReadinessCollectionStep `json:"steps,omitempty"`
}

type serverSideReadinessCollectionStep struct {
	Name                string `json:"name"`
	Status              string `json:"status"`
	Reason              string `json:"reason,omitempty"`
	Command             string `json:"command,omitempty"`
	SourceReport        string `json:"source_report,omitempty"`
	ExecutionAuthorized bool   `json:"execution_authorized"`
	ExternalExecution   bool   `json:"external_execution"`
	AbsorbedByReadiness bool   `json:"absorbed_by_readiness"`
}

type serverSideReadinessBundleEvidence struct {
	Ready        bool   `json:"ready"`
	Status       string `json:"status,omitempty"`
	SourceReport string `json:"source_report,omitempty"`
}

type serverSideReadinessBundleRedaction struct {
	PayloadsStored         bool `json:"payloads_stored"`
	PromptTextStored       bool `json:"prompt_text_stored"`
	TranscriptStored       bool `json:"transcript_stored"`
	ProviderOutputStored   bool `json:"provider_output_stored"`
	EvidenceBodyStored     bool `json:"evidence_body_stored"`
	FullURLsStored         bool `json:"full_urls_stored"`
	LocalPathsStored       bool `json:"local_paths_stored"`
	CredentialValuesStored bool `json:"credential_values_stored"`
}

func runServerSideReadinessBundle(args []string, stdout io.Writer, stderr io.Writer) int {
	options := serverSideReadinessBundleOptions{
		CollectRepeat: 3,
		GatewayURL:    firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:      firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-001"),
		OutputDir:     "reports",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 server-side-readiness-bundle [--gateway-url http://127.0.0.1:21080] [--device-id stackchan-001] [--provider-smoke-report report.json] [--xiaozhi-report report.json] [--v21-professional-report report.json] [--v21-adapter-smoke-report report.json] [--wake-word-firmware-plan report.json] [--use-latest-reports] [--collect-missing] [--execute-provider-smoke] [--execute-v21-smoke] [--collect-repeat 3] [--require-candidate] [--output-dir reports]")
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
		case "--wake-word-firmware-plan":
			if !readStringOption(args, &i, stderr, "--wake-word-firmware-plan", &options.WakeWordFirmwarePlan) {
				return 2
			}
		case "--use-latest-reports":
			options.UseLatestReports = true
		case "--collect-missing":
			options.CollectMissing = true
		case "--execute-provider-smoke":
			options.ExecuteProviderSmoke = true
		case "--execute-v21-smoke":
			options.ExecuteV21Smoke = true
		case "--collect-repeat":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--collect-repeat")
			if !ok {
				return 2
			}
			options.CollectRepeat = value
		case "--require-candidate":
			options.RequireCandidate = true
		default:
			fmt.Fprintf(stderr, "unknown server-side-readiness-bundle option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "server-side readiness bundle report dir invalid: %v\n", err)
		return 1
	}
	productOptions := serverSideProductReadinessOptions(options)
	collection := serverSideReadinessCollection{Enabled: options.CollectMissing, Status: "not_requested"}
	if options.CollectMissing {
		productOptions, collection = collectServerSideReadinessEvidence(context.Background(), options, productOptions, os.Environ())
	}
	report := buildServerSideReadinessBundleReport(context.Background(), productOptions, os.Environ(), collection)
	if options.OutputDir != "" {
		reportPath, err := writeServerSideReadinessBundleReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write server-side readiness bundle report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONServerSideReadinessBundle(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode server-side readiness bundle report: %v\n", err)
		return 1
	}
	if options.RequireCandidate && !report.CandidateReady {
		return 1
	}
	return 0
}

func serverSideProductReadinessOptions(options serverSideReadinessBundleOptions) productReadinessOptions {
	productOptions := productReadinessOptions{
		GatewayURL:            options.GatewayURL,
		DeviceID:              options.DeviceID,
		OutputDir:             options.OutputDir,
		ProviderSmokeReport:   options.ProviderSmokeReport,
		XiaozhiReport:         options.XiaozhiReport,
		V21ProfessionalReport: options.V21ProfessionalReport,
		V21AdapterSmokeReport: options.V21AdapterSmokeReport,
		WakeWordFirmwarePlan:  options.WakeWordFirmwarePlan,
	}
	if options.UseLatestReports {
		productOptions = resolveLatestProductReadinessReports(productOptions)
		productOptions.PhysicalStackChanReport = ""
	}
	return productOptions
}

func buildServerSideReadinessBundleReport(ctx context.Context, options productReadinessOptions, env []string, collection serverSideReadinessCollection) serverSideReadinessBundleReport {
	productReport := buildProductReadinessReport(ctx, options, env)
	collection = markServerSideCollectionAbsorbed(collection, productReport)
	status := "server_side_blocked"
	if productReport.ServerSide.CandidateReady {
		status = "server_side_candidate_ready"
	}
	report := serverSideReadinessBundleReport{
		SchemaVersion:              "a21.server_side_readiness_bundle.v1",
		GeneratedAtMS:              time.Now().UnixMilli(),
		Status:                     status,
		CandidateReady:             productReport.ServerSide.CandidateReady,
		ProductReadinessStatus:     productReport.Status,
		LaunchReady:                productReport.LaunchReady,
		PRDAccepted:                productReport.ServerSide.PRDAccepted,
		RequiresPhysicalAcceptance: productReport.ServerSide.RequiresPhysicalAcceptance,
		Gateway: serverSideReadinessBundleEvidence{
			Ready:  productReport.ServerSide.GatewayReady,
			Status: productReport.Gateway.Status,
		},
		Provider: serverSideReadinessBundleEvidence{
			Ready:        productReport.ServerSide.ProviderEvidenceReady,
			Status:       productReport.Provider.SmokeStatus,
			SourceReport: productReport.Provider.SmokeSourceReport,
		},
		V21: serverSideReadinessBundleEvidence{
			Ready:        productReport.ServerSide.V21ProfessionalEvidenceReady,
			Status:       productReport.V21.Professional.ProfessionalAcceptanceStatus,
			SourceReport: productReport.V21.Professional.SourceReport,
		},
		HostVoice: serverSideReadinessBundleEvidence{
			Ready:        productReport.ServerSide.HostVoiceLoopbackReady,
			Status:       productReport.Voice.VoicePipeline.AcceptanceStatus,
			SourceReport: productReport.Voice.VoicePipeline.SourceReport,
		},
		WakeWord: serverSideReadinessBundleEvidence{
			Ready:  productReport.ServerSide.WakeWordReady,
			Status: productReport.WakeWord.RuntimeStatus,
		},
		ServerSide:         productReport.ServerSide,
		Collection:         collection,
		MissingEvidence:    append([]string(nil), productReport.ServerSide.MissingEvidence...),
		CollectionCommands: buildServerSideReadinessCollectionCommands(productReport),
		NextActions:        buildServerSideReadinessNextActions(productReport),
		Redaction:          serverSideReadinessBundleRedaction{},
	}
	return report
}

func collectServerSideReadinessEvidence(ctx context.Context, options serverSideReadinessBundleOptions, productOptions productReadinessOptions, env []string) (productReadinessOptions, serverSideReadinessCollection) {
	collection := serverSideReadinessCollection{Enabled: true, Status: "passed"}
	initial := buildProductReadinessReport(ctx, productOptions, env)
	for _, missing := range initial.ServerSide.MissingEvidence {
		step := collectServerSideReadinessStep(options, initial, missing)
		collection.Steps = append(collection.Steps, step)
		if step.Status != "passed" && collection.Status == "passed" {
			collection.Status = "partial"
		}
	}
	if len(collection.Steps) == 0 {
		collection.Status = "nothing_missing"
		return productOptions, collection
	}
	refreshed := resolveLatestProductReadinessReports(productOptions)
	refreshed.PhysicalStackChanReport = ""
	return refreshed, collection
}

func collectServerSideReadinessStep(options serverSideReadinessBundleOptions, report productReadinessReport, missing string) serverSideReadinessCollectionStep {
	switch missing {
	case "provider_smoke":
		return collectServerSideProviderSmoke(options, report)
	case "v21_professional_smoke":
		return collectServerSideV21Smoke(options)
	case "host_voice_loopback":
		return collectServerSideHostVoice(options)
	case "gateway":
		return serverSideReadinessCollectionStep{Name: missing, Status: "skipped", Reason: "start gateway separately", Command: "go run ./cmd/a21 gateway --addr 127.0.0.1:21080"}
	case "wake_word":
		return serverSideReadinessCollectionStep{Name: missing, Status: "skipped", Reason: "gateway wake-word status required", Command: "go run ./cmd/a21 product-readiness --use-latest-reports --output-dir reports"}
	default:
		return serverSideReadinessCollectionStep{Name: safeServerSideEvidenceName(missing), Status: "skipped", Reason: "unknown missing evidence"}
	}
}

func collectServerSideProviderSmoke(options serverSideReadinessBundleOptions, report productReadinessReport) serverSideReadinessCollectionStep {
	provider := safeServerSideProviderName(report.Provider.Selected)
	step := serverSideReadinessCollectionStep{
		Name:                "provider_smoke",
		Status:              "skipped",
		Reason:              "requires --execute-provider-smoke",
		Command:             fmt.Sprintf("go run ./cmd/a21 provider-smoke --provider %s --execute --stream --repeat 3 --output-dir reports", provider),
		ExecutionAuthorized: options.ExecuteProviderSmoke,
		ExternalExecution:   true,
	}
	if !options.ExecuteProviderSmoke {
		return step
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-smoke", "--provider", provider, "--execute", "--stream", "--repeat", "3", "--output-dir", options.OutputDir}, &stdout, &stderr)
	return serverSideExecutedCollectionStep(step, code, options.OutputDir, []string{"a21-provider-smoke-*.json"})
}

func collectServerSideV21Smoke(options serverSideReadinessBundleOptions) serverSideReadinessCollectionStep {
	step := serverSideReadinessCollectionStep{
		Name:                "v21_professional_smoke",
		Status:              "skipped",
		Reason:              "requires --execute-v21-smoke",
		Command:             "go run ./cmd/a21 v21-adapter-smoke --execute --output-dir reports",
		ExecutionAuthorized: options.ExecuteV21Smoke,
		ExternalExecution:   true,
	}
	if !options.ExecuteV21Smoke {
		return step
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"v21-adapter-smoke", "--execute", "--output-dir", options.OutputDir}, &stdout, &stderr)
	return serverSideExecutedCollectionStep(step, code, options.OutputDir, []string{"a21-v21-adapter-smoke-*.json"})
}

func collectServerSideHostVoice(options serverSideReadinessBundleOptions) serverSideReadinessCollectionStep {
	repeat := options.CollectRepeat
	if repeat <= 0 {
		repeat = 3
	}
	step := serverSideReadinessCollectionStep{
		Name:                "host_voice_loopback",
		Status:              "failed",
		Command:             fmt.Sprintf("go run ./cmd/a21 xiaozhi-voice-bench --repeat %d --output-dir reports", repeat),
		ExecutionAuthorized: true,
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"xiaozhi-voice-bench", "--gateway-url", options.GatewayURL, "--repeat", fmt.Sprintf("%d", repeat), "--output-dir", options.OutputDir}, &stdout, &stderr)
	return serverSideExecutedCollectionStep(step, code, options.OutputDir, []string{"a21-xiaozhi-voice-bench-*.json"})
}

func markServerSideCollectionAbsorbed(collection serverSideReadinessCollection, report productReadinessReport) serverSideReadinessCollection {
	for index := range collection.Steps {
		step := &collection.Steps[index]
		switch step.Name {
		case "provider_smoke":
			step.AbsorbedByReadiness = step.SourceReport != "" &&
				step.SourceReport == report.Provider.SmokeSourceReport &&
				report.ServerSide.ProviderEvidenceReady
		case "v21_professional_smoke":
			step.AbsorbedByReadiness = step.SourceReport != "" &&
				step.SourceReport == report.V21.Professional.SourceReport &&
				report.ServerSide.V21ProfessionalEvidenceReady
		case "host_voice_loopback":
			step.AbsorbedByReadiness = step.SourceReport != "" &&
				step.SourceReport == report.Voice.VoicePipeline.SourceReport &&
				report.ServerSide.HostVoiceLoopbackReady
		}
	}
	return collection
}

func serverSideExecutedCollectionStep(step serverSideReadinessCollectionStep, code int, outputDir string, patterns []string) serverSideReadinessCollectionStep {
	sourceReport := latestProductReadinessReportPath(outputDir, patterns)
	if sourceReport != "" {
		step.SourceReport = filepath.Base(sourceReport)
	}
	if code == 0 {
		step.Status = "passed"
		step.Reason = ""
		return step
	}
	step.Status = "failed"
	step.Reason = fmt.Sprintf("exit_code_%d", code)
	return step
}

func buildServerSideReadinessCollectionCommands(report productReadinessReport) []string {
	var commands []string
	for _, missing := range report.ServerSide.MissingEvidence {
		switch missing {
		case "gateway":
			commands = append(commands, "go run ./cmd/a21 gateway --addr 127.0.0.1:21080")
		case "provider_smoke":
			provider := safeServerSideProviderName(report.Provider.Selected)
			commands = append(commands, fmt.Sprintf("go run ./cmd/a21 provider-smoke --provider %s --execute --stream --repeat 3 --output-dir reports", provider))
		case "v21_professional_smoke":
			commands = append(commands, "go run ./cmd/a21 v21-adapter-smoke --execute --output-dir reports")
		case "host_voice_loopback":
			commands = append(commands, "go run ./cmd/a21 xiaozhi-voice-bench --repeat 3 --output-dir reports")
		case "wake_word":
			commands = append(commands, "go run ./cmd/a21 product-readiness --use-latest-reports --output-dir reports")
		}
	}
	return commands
}

func buildServerSideReadinessNextActions(report productReadinessReport) []string {
	var actions []string
	for _, action := range report.NextActions {
		if strings.Contains(action, "physical StackChan") ||
			strings.Contains(action, "physical evidence") ||
			strings.Contains(action, "microphone") {
			continue
		}
		actions = append(actions, action)
	}
	return actions
}

func safeServerSideProviderName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 64 || containsLegacyIdentity(value) {
		return "selected"
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "selected"
	}
	return value
}

func safeServerSideEvidenceName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 80 || containsLegacyIdentity(value) {
		return "unknown"
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "unknown"
	}
	return value
}

func writeServerSideReadinessBundleReport(outputDir string, report serverSideReadinessBundleReport) (string, error) {
	if outputDir == "" {
		return "", nil
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-server-side-readiness-bundle-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONServerSideReadinessBundle(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONServerSideReadinessBundle(writer io.Writer, report serverSideReadinessBundleReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
