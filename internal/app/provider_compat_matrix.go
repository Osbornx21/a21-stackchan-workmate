package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/providers"
)

const providerCompatMatrixSchemaVersion = "a21.provider_compat_matrix.v1"
const providerCompatAudioSmokeSchemaVersion = "a21.provider_audio_smoke.v1"

type providerCompatMatrixReport struct {
	SchemaVersion       string                    `json:"schema_version"`
	GeneratedAtMS       int64                     `json:"generated_at_ms"`
	Status              string                    `json:"status"`
	SourceSummaryReport string                    `json:"source_summary_report,omitempty"`
	UseLatestReports    bool                      `json:"use_latest_reports"`
	Rows                []providerCompatMatrixRow `json:"rows"`
	Coverage            providerCompatCoverage    `json:"coverage"`
	MissingCapabilities []string                  `json:"missing_capabilities,omitempty"`
	NextActions         []string                  `json:"next_actions,omitempty"`
	ReportPath          string                    `json:"report_path,omitempty"`
}

type providerCompatMatrixRow struct {
	Stage                string   `json:"stage"`
	Placement            string   `json:"placement"`
	Provider             string   `json:"provider"`
	Family               string   `json:"family,omitempty"`
	Protocol             string   `json:"protocol,omitempty"`
	Model                string   `json:"model,omitempty"`
	Status               string   `json:"status"`
	Executed             bool     `json:"executed"`
	Configured           bool     `json:"configured"`
	RouteEligible        bool     `json:"route_eligible"`
	EvidenceMode         string   `json:"evidence_mode"`
	SourceReport         string   `json:"source_report,omitempty"`
	Attempts             int      `json:"attempts,omitempty"`
	OK                   int      `json:"ok,omitempty"`
	Failures             int      `json:"failures,omitempty"`
	FirstByteP95MS       *float64 `json:"first_byte_p95_ms,omitempty"`
	FirstContentP95MS    *float64 `json:"first_content_p95_ms,omitempty"`
	TotalP95MS           *float64 `json:"total_p95_ms,omitempty"`
	ASRFinalP95MS        *float64 `json:"asr_final_p95_ms,omitempty"`
	DecodeDurationMS     *float64 `json:"decode_duration_ms,omitempty"`
	RealTimeFactor       *float64 `json:"real_time_factor,omitempty"`
	TTSFirstAudioMS      *float64 `json:"tts_first_audio_ms,omitempty"`
	EndpointHost         string   `json:"endpoint_host,omitempty"`
	Detail               string   `json:"detail,omitempty"`
	CompatibilityOnly    bool     `json:"compatibility_only,omitempty"`
	ProductReadyEvidence bool     `json:"product_ready_evidence,omitempty"`
}

type providerCompatCoverage struct {
	LocalASR bool `json:"local_asr"`
	CloudASR bool `json:"cloud_asr"`
	LocalLLM bool `json:"local_llm"`
	CloudLLM bool `json:"cloud_llm"`
	LocalTTS bool `json:"local_tts"`
	CloudTTS bool `json:"cloud_tts"`
}

type providerCompatFullSummary struct {
	SchemaVersion       string                         `json:"schema_version"`
	LabLLMStreamSummary []providerCompatLabLLMEntry    `json:"lab_llm_stream_summary"`
	A21DeepSeekExecute  *providers.ProviderSmokeReport `json:"a21_deepseek_execute"`
	LocalASR            []providerCompatLocalAudioRow  `json:"local_asr"`
	LocalTTS            []providerCompatLocalAudioRow  `json:"local_tts"`
}

type providerCompatLabLLMEntry struct {
	Provider          string   `json:"provider"`
	Attempts          int      `json:"attempts"`
	OK                int      `json:"ok"`
	Failures          int      `json:"failures"`
	FirstContentP50MS *float64 `json:"first_content_p50_ms"`
	FirstContentP95MS *float64 `json:"first_content_p95_ms"`
	TotalP95MS        *float64 `json:"total_p95_ms"`
	LastError         string   `json:"last_error"`
}

type providerCompatLocalAudioRow struct {
	SchemaVersion    string  `json:"schema_version"`
	Status           string  `json:"status"`
	Provider         string  `json:"provider"`
	Engine           string  `json:"engine"`
	Model            string  `json:"model"`
	ModelDir         string  `json:"model_dir"`
	MatrixModel      string  `json:"matrix_model"`
	MatrixFamily     string  `json:"matrix_family"`
	MatrixStage      string  `json:"matrix_stage"`
	ReportPath       string  `json:"report_path"`
	ExitCode         int     `json:"exit_code"`
	DecodeDurationMS float64 `json:"decode_duration_ms"`
	RealTimeFactor   float64 `json:"real_time_factor"`
	DurationMS       float64 `json:"duration_ms"`
	TTSFirstAudioMS  float64 `json:"tts_first_audio_ms"`
}

type providerCompatMatrixOptions struct {
	ProviderFullSummary  string
	ProviderAudioReports []string
	UseLatestReports     bool
	ReportsDir           string
	OutputDir            string
}

type providerCompatAudioSmokeReport struct {
	SchemaVersion      string   `json:"schema_version"`
	GeneratedAtMS      int64    `json:"generated_at_ms"`
	Stage              string   `json:"stage"`
	Placement          string   `json:"placement"`
	Provider           string   `json:"provider"`
	Status             string   `json:"status"`
	Executed           bool     `json:"executed"`
	Configured         bool     `json:"configured"`
	EndpointHost       string   `json:"endpoint_host,omitempty"`
	ASRFinalP95MS      *float64 `json:"asr_final_p95_ms,omitempty"`
	TTSFirstAudioP95MS *float64 `json:"tts_first_audio_p95_ms,omitempty"`
	ReportPath         string   `json:"report_path,omitempty"`
}

func runProviderCompatMatrix(args []string, stdout io.Writer, stderr io.Writer) int {
	options := providerCompatMatrixOptions{ReportsDir: "reports"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 provider-compat-matrix [--provider-full-summary report.json] [--use-latest-reports] [--reports-dir reports] [--output-dir reports]")
			return 0
		case "--provider-full-summary", "--summary":
			if !readStringOption(args, &i, stderr, args[i], &options.ProviderFullSummary) {
				return 2
			}
		case "--provider-audio-smoke-report":
			var path string
			if !readStringOption(args, &i, stderr, "--provider-audio-smoke-report", &path) {
				return 2
			}
			options.ProviderAudioReports = append(options.ProviderAudioReports, path)
		case "--use-latest-reports":
			options.UseLatestReports = true
		case "--reports-dir":
			if !readStringOption(args, &i, stderr, "--reports-dir", &options.ReportsDir) {
				return 2
			}
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown provider-compat-matrix option %q\n", args[i])
			return 2
		}
	}
	report, err := buildProviderCompatMatrix(options)
	if err != nil {
		fmt.Fprintf(stderr, "provider compat matrix: %v\n", err)
		return 1
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "provider compat matrix report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeProviderCompatMatrixReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write provider compat matrix report: %v\n", err)
			return 1
		}
		report.ReportPath = filepath.Base(filepath.Clean(reportPath))
	}
	if err := writeJSONProviderCompatMatrix(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode provider compat matrix report: %v\n", err)
		return 1
	}
	return 0
}

func buildProviderCompatMatrix(options providerCompatMatrixOptions) (providerCompatMatrixReport, error) {
	report := providerCompatMatrixReport{
		SchemaVersion:    providerCompatMatrixSchemaVersion,
		GeneratedAtMS:    time.Now().UnixMilli(),
		Status:           "partial",
		UseLatestReports: options.UseLatestReports,
	}
	summaryPath := strings.TrimSpace(options.ProviderFullSummary)
	if summaryPath == "" && options.UseLatestReports {
		summaryPath = latestProviderFullSummaryPath(options.ReportsDir)
	}
	if summaryPath != "" {
		if err := validateA21InputPath(summaryPath); err != nil {
			return report, err
		}
		rows, err := providerCompatRowsFromFullSummary(summaryPath)
		if err != nil {
			return report, err
		}
		report.SourceSummaryReport = filepath.Base(filepath.Clean(summaryPath))
		report.Rows = append(report.Rows, rows...)
	}
	for _, audioReportPath := range options.ProviderAudioReports {
		rows, err := providerCompatRowsFromAudioSmokeReport(audioReportPath)
		if err != nil {
			return report, err
		}
		report.Rows = append(report.Rows, rows...)
	}
	if options.UseLatestReports {
		report.Rows = append(report.Rows, providerCompatRowsFromLatestReports(options.ReportsDir)...)
	}
	sort.SliceStable(report.Rows, func(i, j int) bool {
		left := report.Rows[i].Stage + ":" + report.Rows[i].Placement + ":" + report.Rows[i].Provider + ":" + report.Rows[i].SourceReport
		right := report.Rows[j].Stage + ":" + report.Rows[j].Placement + ":" + report.Rows[j].Provider + ":" + report.Rows[j].SourceReport
		return left < right
	})
	report.Coverage = providerCompatCoverageFromRows(report.Rows)
	report.MissingCapabilities = providerCompatMissingCapabilities(report.Coverage)
	report.NextActions = providerCompatNextActions(report.MissingCapabilities)
	if len(report.MissingCapabilities) == 0 {
		report.Status = "ready"
	}
	return report, nil
}

func providerCompatRowsFromFullSummary(path string) ([]providerCompatMatrixRow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var summary providerCompatFullSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, err
	}
	if summary.SchemaVersion != "a21.provider_full_validation.v1" {
		return nil, fmt.Errorf("unsupported provider full summary schema")
	}
	var rows []providerCompatMatrixRow
	for _, entry := range summary.LabLLMStreamSummary {
		provider := providerLatencySafeIdentifier(entry.Provider, false)
		if provider == "" {
			continue
		}
		status := "failed"
		if entry.OK > 0 && entry.Failures == 0 {
			status = "passed"
		} else if entry.OK > 0 {
			status = "mixed"
		}
		detail := "5080lab LLM stream benchmark"
		if strings.TrimSpace(entry.LastError) != "" {
			detail = "5080lab LLM stream benchmark reported failure"
		}
		rows = append(rows, providerCompatMatrixRow{
			Stage:             "llm",
			Placement:         "cloud",
			Provider:          provider,
			Family:            string(providers.ProviderFamilyTextStream),
			Protocol:          "openai_chat_completions",
			Status:            status,
			Executed:          entry.Attempts > 0,
			Configured:        entry.Attempts > 0,
			RouteEligible:     false,
			EvidenceMode:      "5080lab_llm_stream_summary",
			Attempts:          entry.Attempts,
			OK:                entry.OK,
			Failures:          entry.Failures,
			FirstContentP95MS: entry.FirstContentP95MS,
			TotalP95MS:        entry.TotalP95MS,
			Detail:            detail,
			CompatibilityOnly: true,
		})
	}
	if summary.A21DeepSeekExecute != nil && summary.A21DeepSeekExecute.Provider != "" {
		rows = append(rows, providerCompatRowFromProviderSmoke(*summary.A21DeepSeekExecute, "5080lab_a21_provider_smoke", "cloud", summary.A21DeepSeekExecute.ReportPath))
	}
	for _, entry := range summary.LocalASR {
		rows = append(rows, providerCompatRowFromLocalAudioSummary("asr", entry))
	}
	for _, entry := range summary.LocalTTS {
		rows = append(rows, providerCompatRowFromLocalAudioSummary("tts", entry))
	}
	return rows, nil
}

func providerCompatRowsFromLocalAudioReport(path string, stage string) []providerCompatMatrixRow {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	switch stage {
	case "asr":
		var report audio.LocalASRReport
		if err := json.Unmarshal(data, &report); err != nil || report.SchemaVersion != "a21.audio.local_asr.v1" {
			return nil
		}
		return []providerCompatMatrixRow{{
			Stage:            "asr",
			Placement:        "local",
			Provider:         providerLatencySafeIdentifier(report.Provider, false),
			Family:           string(providers.ProviderFamilyLocalAudio),
			Protocol:         "local_asr_smoke",
			Model:            providerCompatBaseName(report.ModelDir),
			Status:           providerCompatSafeStatus(report.Status),
			Executed:         report.Status == "passed",
			Configured:       report.Status == "passed",
			EvidenceMode:     "a21_local_asr_smoke",
			SourceReport:     providerCompatBaseName(providerCompatFirstNonEmpty(report.ReportPath, path)),
			DecodeDurationMS: optionalFloat64(report.DecodeDurationMS),
			RealTimeFactor:   optionalFloat64(report.RealTimeFactor),
			Detail:           "local ASR smoke report",
		}}
	case "tts":
		var report audio.LocalTTSReport
		if err := json.Unmarshal(data, &report); err != nil || report.SchemaVersion != "a21.audio.local_tts.v1" {
			return nil
		}
		return []providerCompatMatrixRow{{
			Stage:           "tts",
			Placement:       "local",
			Provider:        providerLatencySafeIdentifier(report.Provider, false),
			Family:          string(providers.ProviderFamilyLocalAudio),
			Protocol:        "local_tts_smoke",
			Model:           providerCompatBaseName(providerCompatFirstNonEmpty(report.ModelDir, report.Model)),
			Status:          providerCompatSafeStatus(report.Status),
			Executed:        report.Status == "passed",
			Configured:      report.Status == "passed",
			EvidenceMode:    "a21_local_tts_smoke",
			SourceReport:    providerCompatBaseName(providerCompatFirstNonEmpty(report.ReportPath, path)),
			TTSFirstAudioMS: optionalFloat64(report.TTSFirstAudioMS),
			Detail:          "local TTS smoke report",
		}}
	default:
		return nil
	}
}

func providerCompatRowsFromAudioSmokeReport(path string) ([]providerCompatMatrixRow, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("provider audio smoke report path is required")
	}
	if err := validateA21InputPath(path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("invalid provider audio smoke report")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("invalid provider audio smoke report")
	}
	if productProviderSmokeReportContainsForbiddenValue(raw) {
		return nil, fmt.Errorf("unsafe provider audio smoke report")
	}
	var report providerCompatAudioSmokeReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("invalid provider audio smoke report")
	}
	stage := strings.ToLower(strings.TrimSpace(report.Stage))
	placement := strings.ToLower(strings.TrimSpace(report.Placement))
	provider := providerLatencySafeIdentifier(report.Provider, false)
	if report.SchemaVersion != providerCompatAudioSmokeSchemaVersion ||
		report.GeneratedAtMS <= 0 ||
		(stage != "asr" && stage != "tts") ||
		(placement != "local" && placement != "cloud") ||
		provider == "" {
		return nil, fmt.Errorf("invalid provider audio smoke report")
	}
	endpointHost := ""
	if strings.TrimSpace(report.EndpointHost) != "" && !productProviderSmokeEndpointHostUnsafe(report.EndpointHost) {
		endpointHost = strings.TrimSpace(report.EndpointHost)
	}
	row := providerCompatMatrixRow{
		Stage:             stage,
		Placement:         placement,
		Provider:          provider,
		Family:            string(providers.ProviderFamilyVoiceHybrid),
		Protocol:          "provider_audio_smoke",
		Status:            providerCompatSafeStatus(report.Status),
		Executed:          report.Executed,
		Configured:        report.Configured,
		EvidenceMode:      "a21_provider_audio_smoke",
		SourceReport:      providerCompatBaseName(providerCompatFirstNonEmpty(report.ReportPath, path)),
		EndpointHost:      endpointHost,
		Detail:            "provider audio smoke report",
		CompatibilityOnly: true,
	}
	if stage == "asr" {
		row.ASRFinalP95MS = report.ASRFinalP95MS
	}
	if stage == "tts" {
		row.TTSFirstAudioMS = report.TTSFirstAudioP95MS
	}
	return []providerCompatMatrixRow{row}, nil
}

func providerCompatRowsFromLatestReports(reportDir string) []providerCompatMatrixRow {
	reportDir = strings.TrimSpace(reportDir)
	if reportDir == "" {
		reportDir = "reports"
	}
	var rows []providerCompatMatrixRow
	for _, path := range latestProductReadinessReportCandidates(reportDir, []string{"a21-provider-smoke-*.json", "a21-provider-realtime-fixture-*.json"}) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var report providers.ProviderSmokeReport
		if err := json.Unmarshal(data, &report); err != nil || report.SchemaVersion != providers.ProviderSmokeSchemaVersion {
			continue
		}
		placement := "cloud"
		if strings.HasPrefix(report.Provider, "local_") {
			placement = "local"
		}
		if report.Family == string(providers.ProviderFamilyVoiceHybrid) {
			rows = append(rows, providerCompatRowFromProviderSmoke(report, "a21_provider_realtime_fixture", "cloud", path))
			continue
		}
		rows = append(rows, providerCompatRowFromProviderSmoke(report, "a21_provider_smoke", placement, path))
	}
	for _, path := range latestProductReadinessReportCandidates(reportDir, []string{"a21-provider-audio-smoke-*.json"}) {
		audioRows, err := providerCompatRowsFromAudioSmokeReport(path)
		if err == nil {
			rows = append(rows, audioRows...)
		}
	}
	for _, path := range latestProductReadinessReportCandidates(reportDir, []string{"a21-local-asr-smoke-*.json"}) {
		rows = append(rows, providerCompatRowsFromLocalAudioReport(path, "asr")...)
	}
	for _, path := range latestProductReadinessReportCandidates(reportDir, []string{"a21-local-tts-smoke-*.json"}) {
		rows = append(rows, providerCompatRowsFromLocalAudioReport(path, "tts")...)
	}
	return rows
}

func providerCompatRowFromProviderSmoke(report providers.ProviderSmokeReport, evidenceMode string, placement string, sourcePath string) providerCompatMatrixRow {
	stage := "llm"
	if report.Family == string(providers.ProviderFamilyVoiceHybrid) && strings.Contains(report.Provider, "tts") {
		stage = "tts"
	}
	if report.Family == string(providers.ProviderFamilyVoiceRealtime) {
		stage = "asr_tts"
	}
	row := providerCompatMatrixRow{
		Stage:             stage,
		Placement:         placement,
		Provider:          providerLatencySafeIdentifier(report.Provider, false),
		Family:            report.Family,
		Protocol:          report.Protocol,
		Status:            providerCompatSafeStatus(string(report.Status)),
		Executed:          report.Executed,
		Configured:        report.Configured,
		RouteEligible:     report.RouteEligible,
		EvidenceMode:      evidenceMode,
		SourceReport:      providerCompatBaseName(providerCompatFirstNonEmpty(report.ReportPath, sourcePath)),
		Attempts:          report.Repeat,
		EndpointHost:      report.EndpointHost,
		Detail:            report.Detail,
		CompatibilityOnly: !report.RouteEligible,
	}
	if report.TimingSummary != nil {
		row.FirstByteP95MS = optionalFloat64(report.TimingSummary.FirstByteP95MS)
		row.FirstContentP95MS = optionalFloat64(report.TimingSummary.FirstContentP95MS)
		row.TotalP95MS = optionalFloat64(report.TimingSummary.TotalDurationP95MS)
	}
	row.ProductReadyEvidence = row.Stage == "llm" &&
		row.Placement == "cloud" &&
		row.Status == "passed" &&
		row.Executed &&
		row.RouteEligible &&
		report.Stream
	return row
}

func providerCompatRowFromLocalAudioSummary(stage string, entry providerCompatLocalAudioRow) providerCompatMatrixRow {
	model := providerCompatFirstNonEmpty(entry.MatrixModel, entry.ModelDir, entry.Model)
	provider := providerLatencySafeIdentifier(providerCompatFirstNonEmpty(entry.Provider, "local_audio"), false)
	protocol := "local_" + stage + "_smoke"
	row := providerCompatMatrixRow{
		Stage:        stage,
		Placement:    "local",
		Provider:     provider,
		Family:       string(providers.ProviderFamilyLocalAudio),
		Protocol:     protocol,
		Model:        providerCompatBaseName(model),
		Status:       providerCompatSafeStatus(entry.Status),
		Executed:     entry.Status == "passed",
		Configured:   entry.Status == "passed",
		EvidenceMode: "5080lab_" + protocol,
		SourceReport: providerCompatBaseName(entry.ReportPath),
		Detail:       "5080lab local audio matrix",
	}
	if stage == "asr" {
		row.DecodeDurationMS = optionalFloat64(entry.DecodeDurationMS)
		row.RealTimeFactor = optionalFloat64(entry.RealTimeFactor)
	}
	if stage == "tts" {
		row.TTSFirstAudioMS = optionalFloat64(firstNonZeroFloat64(entry.TTSFirstAudioMS, entry.DurationMS))
	}
	if row.SourceReport == "." {
		row.SourceReport = ""
	}
	return row
}

func providerCompatCoverageFromRows(rows []providerCompatMatrixRow) providerCompatCoverage {
	var coverage providerCompatCoverage
	for _, row := range rows {
		if row.Status != "passed" || !row.Executed {
			continue
		}
		if row.EvidenceMode == "a21_provider_realtime_fixture" {
			continue
		}
		switch row.Stage + ":" + row.Placement {
		case "asr:local":
			coverage.LocalASR = true
		case "asr:cloud":
			coverage.CloudASR = true
		case "llm:local":
			coverage.LocalLLM = true
		case "llm:cloud":
			coverage.CloudLLM = true
		case "tts:local":
			coverage.LocalTTS = true
		case "tts:cloud":
			coverage.CloudTTS = true
		case "asr_tts:cloud":
			coverage.CloudASR = true
			coverage.CloudTTS = true
		}
	}
	return coverage
}

func providerCompatMissingCapabilities(coverage providerCompatCoverage) []string {
	var missing []string
	if !coverage.LocalASR {
		missing = append(missing, "local_asr")
	}
	if !coverage.CloudASR {
		missing = append(missing, "cloud_asr")
	}
	if !coverage.LocalLLM {
		missing = append(missing, "local_llm")
	}
	if !coverage.CloudLLM {
		missing = append(missing, "cloud_llm")
	}
	if !coverage.LocalTTS {
		missing = append(missing, "local_tts")
	}
	if !coverage.CloudTTS {
		missing = append(missing, "cloud_tts")
	}
	return missing
}

func providerCompatNextActions(missing []string) []string {
	if len(missing) == 0 {
		return nil
	}
	var actions []string
	for _, item := range missing {
		switch item {
		case "local_asr":
			actions = append(actions, "run `go run ./cmd/a21 local-asr-smoke --engine sherpa_onnx --output-dir reports` on an A21 host with the local ASR model cache")
		case "cloud_asr":
			actions = append(actions, "import or run a redacted 5080lab cloud ASR smoke report, then rerun `go run ./cmd/a21 provider-compat-matrix --use-latest-reports --output-dir reports`")
		case "local_llm":
			actions = append(actions, "run `go run ./cmd/a21 provider-smoke --provider local_ollama --execute --stream --repeat 3 --output-dir reports` against the local fallback LLM")
		case "cloud_llm":
			actions = append(actions, "run `go run ./cmd/a21 provider-smoke --provider siliconflow --execute --stream --repeat 5 --output-dir reports` on 5080lab, or another configured cloud LLM provider")
		case "local_tts":
			actions = append(actions, "run `go run ./cmd/a21 local-tts-smoke --engine sherpa_onnx --output-dir reports` on an A21 host with the local TTS model cache")
		case "cloud_tts":
			actions = append(actions, "import or run a redacted 5080lab cloud TTS smoke report; `provider-realtime-fixture` is only a contract fixture and does not count as real cloud TTS")
		}
	}
	return actions
}

func latestProviderFullSummaryPath(reportDir string) string {
	reportDir = strings.TrimSpace(reportDir)
	if reportDir == "" {
		reportDir = "reports"
	}
	matches, err := filepath.Glob(filepath.Join(reportDir, "a21-provider-full-*", "a21-provider-full-summary.json"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	sort.Slice(matches, func(i, j int) bool {
		left, lerr := os.Stat(matches[i])
		right, rerr := os.Stat(matches[j])
		if lerr == nil && rerr == nil && !left.ModTime().Equal(right.ModTime()) {
			return left.ModTime().After(right.ModTime())
		}
		return filepath.Base(filepath.Dir(matches[i])) > filepath.Base(filepath.Dir(matches[j]))
	})
	return matches[0]
}

func writeProviderCompatMatrixReport(outputDir string, report providerCompatMatrixReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	var reportPath string
	var file *os.File
	var err error
	for attempt := 0; attempt < 100; attempt++ {
		now := time.Now()
		stamp := fmt.Sprintf("%s-%09d", now.Format("20060102-150405"), now.Nanosecond())
		if attempt > 0 {
			stamp = fmt.Sprintf("%s-%02d", stamp, attempt)
		}
		reportPath = filepath.Join(outputDir, "a21-provider-compat-matrix-"+stamp+".json")
		file, err = os.OpenFile(reportPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		break
	}
	if file == nil {
		return "", fmt.Errorf("could not allocate unique provider compat matrix report path")
	}
	defer file.Close()
	report.ReportPath = filepath.Base(filepath.Clean(reportPath))
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeJSONProviderCompatMatrix(writer io.Writer, report providerCompatMatrixReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func providerCompatSafeStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "passed", "ready", "failed", "skipped", "unsupported", "mixed", "command_failed":
		return status
	default:
		if status == "" {
			return "unknown"
		}
		return "unknown"
	}
}

func providerCompatFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func providerCompatBaseName(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	base := filepath.Base(filepath.Clean(value))
	if base == "." {
		return ""
	}
	return base
}

func optionalFloat64(value float64) *float64 {
	if value <= 0 {
		return nil
	}
	return &value
}

func firstNonZeroFloat64(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
