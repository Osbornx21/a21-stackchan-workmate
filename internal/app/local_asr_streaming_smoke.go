package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const localASRStreamingSmokeSchemaVersion = "a21.audio.local_asr_streaming_smoke.v1"

type localASRStreamingSmokeOptions struct {
	HelperPath string
	PythonPath string
	ModelDir   string
	Family     string
	OutputDir  string
	Timeout    time.Duration
}

type localASRStreamingSmokeReport struct {
	SchemaVersion           string   `json:"schema_version"`
	GeneratedAtMS           int64    `json:"generated_at_ms"`
	Status                  string   `json:"status"`
	Provider                string   `json:"provider"`
	Family                  string   `json:"family"`
	EvidenceMode            string   `json:"evidence_mode"`
	HelperEnv               string   `json:"helper_env"`
	HelperConfigured        bool     `json:"helper_configured"`
	Helper                  string   `json:"helper,omitempty"`
	PythonEnv               string   `json:"python_env"`
	PythonConfigured        bool     `json:"python_configured"`
	Python                  string   `json:"python,omitempty"`
	ModelDirEnv             string   `json:"model_dir_env"`
	ModelDirConfigured      bool     `json:"model_dir_configured"`
	ModelDir                string   `json:"model_dir,omitempty"`
	ModelFamilyEnv          string   `json:"model_family_env"`
	ModelFilesPresent       bool     `json:"model_files_present"`
	MissingModelFiles       []string `json:"missing_model_files,omitempty"`
	HelperFakeEnvConfigured bool     `json:"helper_fake_env_configured"`
	SampleRateHz            int      `json:"sample_rate_hz"`
	Channels                int      `json:"channels"`
	FrameDurationMS         int      `json:"frame_duration_ms"`
	FramesAppended          int      `json:"frames_appended"`
	ReadyEvents             int      `json:"ready_events"`
	PartialEvents           int      `json:"partial_events"`
	FinalEvents             int      `json:"final_events"`
	FinalTextChars          int      `json:"final_text_chars"`
	TranscriptPolicy        string   `json:"transcript_policy"`
	ProviderOutputPolicy    string   `json:"provider_output_policy"`
	AudioPayloadPolicy      string   `json:"audio_payload_policy"`
	URLPolicy               string   `json:"url_policy"`
	LocalPathPolicy         string   `json:"local_path_policy"`
	Findings                []string `json:"findings,omitempty"`
	ReportPath              string   `json:"report_path,omitempty"`
}

type sherpaStreamingSmokeCommand struct {
	Type         string `json:"type"`
	Seq          uint64 `json:"seq,omitempty"`
	SampleRateHz int    `json:"sample_rate_hz,omitempty"`
	Channels     int    `json:"channels,omitempty"`
	DurationMS   int    `json:"duration_ms,omitempty"`
	PCM16LEB64   string `json:"pcm16le_b64,omitempty"`
	Mode         string `json:"mode,omitempty"`
	TraceID      string `json:"trace_id,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	ModelDir     string `json:"model_dir,omitempty"`
	Family       string `json:"family,omitempty"`
}

type sherpaStreamingSmokeEvent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Code string `json:"code,omitempty"`
}

func runLocalASRStreamingSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	options := localASRStreamingSmokeOptions{
		HelperPath: strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_STREAMING_HELPER")),
		PythonPath: strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_STREAMING_PYTHON")),
		ModelDir:   strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_MODEL_DIR")),
		Family:     strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_FAMILY")),
		OutputDir:  "reports",
		Timeout:    30 * time.Second,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 local-asr-streaming-smoke [--helper <path>] [--python <python>] [--model-dir <dir>] [--family streaming_zipformer] [--timeout-ms 30000] [--output-dir reports]")
			return 0
		case "--helper":
			if !readStringOption(args, &i, stderr, "--helper", &options.HelperPath) {
				return 2
			}
		case "--python":
			if !readStringOption(args, &i, stderr, "--python", &options.PythonPath) {
				return 2
			}
		case "--model-dir":
			if !readStringOption(args, &i, stderr, "--model-dir", &options.ModelDir) {
				return 2
			}
		case "--family":
			if !readStringOption(args, &i, stderr, "--family", &options.Family) {
				return 2
			}
		case "--timeout-ms":
			raw := ""
			if !readStringOption(args, &i, stderr, "--timeout-ms", &raw) {
				return 2
			}
			timeoutMS, err := parsePositiveIntOption(raw, "--timeout-ms")
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 2
			}
			options.Timeout = time.Duration(timeoutMS) * time.Millisecond
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown local-asr-streaming-smoke option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "local ASR streaming report dir invalid: %v\n", err)
		return 1
	}
	report := runSherpaONNXStreamingASRSmoke(context.Background(), options)
	reportPath, err := writeLocalASRStreamingSmokeReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write local ASR streaming smoke report: %v\n", err)
		return 1
	}
	report.ReportPath = filepath.Base(filepath.Clean(reportPath))
	if err := writeJSONLocalASRStreamingSmoke(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode local ASR streaming smoke report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}

func runSherpaONNXStreamingASRSmoke(ctx context.Context, options localASRStreamingSmokeOptions) localASRStreamingSmokeReport {
	family := normalizeSherpaStreamingFamily(options.Family)
	report := localASRStreamingSmokeReport{
		SchemaVersion:           localASRStreamingSmokeSchemaVersion,
		GeneratedAtMS:           time.Now().UnixMilli(),
		Status:                  "blocked",
		Provider:                "sherpa_onnx_streaming",
		Family:                  family,
		EvidenceMode:            "real_model_no_audio_streaming_smoke",
		HelperEnv:               "A21_SHERPA_ONNX_STREAMING_HELPER",
		HelperConfigured:        strings.TrimSpace(options.HelperPath) != "",
		Helper:                  safeSmokeBaseName(options.HelperPath),
		PythonEnv:               "A21_SHERPA_ONNX_STREAMING_PYTHON",
		PythonConfigured:        strings.TrimSpace(options.PythonPath) != "",
		Python:                  safeSmokeBaseName(options.PythonPath),
		ModelDirEnv:             "A21_SHERPA_ONNX_ASR_MODEL_DIR",
		ModelDirConfigured:      strings.TrimSpace(options.ModelDir) != "",
		ModelDir:                safeSmokeBaseName(options.ModelDir),
		ModelFamilyEnv:          "A21_SHERPA_ONNX_ASR_FAMILY",
		HelperFakeEnvConfigured: os.Getenv("A21_SHERPA_STREAMING_ASR_FAKE") == "1",
		SampleRateHz:            16000,
		Channels:                1,
		FrameDurationMS:         60,
		TranscriptPolicy:        "transcript_not_recorded",
		ProviderOutputPolicy:    "provider_output_not_recorded",
		AudioPayloadPolicy:      "raw_audio_not_recorded",
		URLPolicy:               "urls_not_recorded",
		LocalPathPolicy:         "basenames_only",
	}
	report.MissingModelFiles = missingSherpaStreamingModelFiles(options.ModelDir)
	report.ModelFilesPresent = report.ModelDirConfigured && len(report.MissingModelFiles) == 0
	if !report.HelperConfigured || !pathExists(options.HelperPath) {
		report.Findings = append(report.Findings, "streaming_helper_missing")
		return report
	}
	if !report.ModelDirConfigured || !dirExists(options.ModelDir) {
		report.Findings = append(report.Findings, "model_dir_missing")
		return report
	}
	if family != "streaming_zipformer" {
		report.Findings = append(report.Findings, "unsupported_streaming_family")
		return report
	}
	if len(report.MissingModelFiles) > 0 {
		report.Findings = append(report.Findings, "model_files_missing")
		return report
	}
	if report.HelperFakeEnvConfigured {
		report.EvidenceMode = "helper_fake_unit_smoke"
		report.Findings = append(report.Findings, "helper_fake_unit_not_real_model")
		return report
	}
	if options.Timeout <= 0 {
		options.Timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	result := runSherpaStreamingHelperSmoke(runCtx, options, report.SampleRateHz, report.Channels, report.FrameDurationMS)
	report.FramesAppended = result.framesAppended
	report.ReadyEvents = result.readyEvents
	report.PartialEvents = result.partialEvents
	report.FinalEvents = result.finalEvents
	report.FinalTextChars = result.finalTextChars
	if result.finding != "" {
		report.Findings = append(report.Findings, result.finding)
		if sherpaStreamingBlockerFinding(result.finding) {
			report.Status = "blocked"
		} else {
			report.Status = "failed"
		}
		return report
	}
	if result.finalEvents == 0 {
		report.Findings = append(report.Findings, "helper_final_event_missing")
		report.Status = "failed"
		return report
	}
	report.Status = "passed"
	return report
}

type sherpaStreamingHelperSmokeResult struct {
	framesAppended int
	readyEvents    int
	partialEvents  int
	finalEvents    int
	finalTextChars int
	finding        string
}

func runSherpaStreamingHelperSmoke(ctx context.Context, options localASRStreamingSmokeOptions, sampleRateHz int, channels int, frameDurationMS int) sherpaStreamingHelperSmokeResult {
	cmdName := strings.TrimSpace(options.HelperPath)
	args := []string{}
	if pythonPath := strings.TrimSpace(options.PythonPath); pythonPath != "" {
		cmdName = pythonPath
		args = append(args, strings.TrimSpace(options.HelperPath))
	}
	cmd := exec.CommandContext(ctx, cmdName, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return sherpaStreamingHelperSmokeResult{finding: "helper_runtime_failed"}
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return sherpaStreamingHelperSmokeResult{finding: "helper_runtime_failed"}
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return sherpaStreamingHelperSmokeResult{finding: "helper_runtime_failed"}
	}
	encoder := json.NewEncoder(stdin)
	decoder := json.NewDecoder(stdout)
	result := sherpaStreamingHelperSmokeResult{}
	if err := encoder.Encode(sherpaStreamingSmokeCommand{
		Type:         "start",
		SampleRateHz: sampleRateHz,
		Channels:     channels,
		Mode:         "workmate",
		TraceID:      "redacted",
		SessionID:    "redacted",
		ModelDir:     strings.TrimSpace(options.ModelDir),
		Family:       normalizeSherpaStreamingFamily(options.Family),
	}); err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		return sherpaStreamingHelperSmokeResult{finding: "helper_runtime_failed"}
	}
	if finding := readSherpaStreamingUntilReady(ctx, decoder, &result); finding != "" {
		_ = stdin.Close()
		_ = cmd.Wait()
		result.finding = finding
		return result
	}
	pcm := make([]byte, sampleRateHz*frameDurationMS/1000*2)
	if err := encoder.Encode(sherpaStreamingSmokeCommand{
		Type:         "append",
		Seq:          1,
		SampleRateHz: sampleRateHz,
		Channels:     channels,
		DurationMS:   frameDurationMS,
		PCM16LEB64:   base64.StdEncoding.EncodeToString(pcm),
	}); err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		return sherpaStreamingHelperSmokeResult{finding: "helper_runtime_failed", readyEvents: result.readyEvents}
	}
	result.framesAppended = 1
	if err := encoder.Encode(sherpaStreamingSmokeCommand{Type: "commit"}); err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		result.finding = "helper_runtime_failed"
		return result
	}
	_ = stdin.Close()
	result.finding = readSherpaStreamingUntilFinal(ctx, decoder, &result)
	if waitErr := cmd.Wait(); waitErr != nil && result.finding == "" {
		result.finding = "helper_runtime_failed"
	}
	if ctx.Err() != nil && result.finding == "" {
		result.finding = "helper_timeout"
	}
	return result
}

func readSherpaStreamingUntilReady(ctx context.Context, decoder *json.Decoder, result *sherpaStreamingHelperSmokeResult) string {
	for {
		if ctx.Err() != nil {
			return "helper_timeout"
		}
		var event sherpaStreamingSmokeEvent
		if err := decoder.Decode(&event); err != nil {
			return "helper_protocol_error"
		}
		finding := applySherpaStreamingSmokeEvent(event, result)
		if finding != "" {
			return finding
		}
		if result.readyEvents > 0 {
			return ""
		}
	}
}

func readSherpaStreamingUntilFinal(ctx context.Context, decoder *json.Decoder, result *sherpaStreamingHelperSmokeResult) string {
	for {
		if ctx.Err() != nil {
			return "helper_timeout"
		}
		var event sherpaStreamingSmokeEvent
		if err := decoder.Decode(&event); err != nil {
			return "helper_protocol_error"
		}
		finding := applySherpaStreamingSmokeEvent(event, result)
		if finding != "" {
			return finding
		}
		if result.finalEvents > 0 {
			return ""
		}
	}
}

func applySherpaStreamingSmokeEvent(event sherpaStreamingSmokeEvent, result *sherpaStreamingHelperSmokeResult) string {
	switch strings.ToLower(strings.TrimSpace(event.Type)) {
	case "ready":
		result.readyEvents++
	case "partial":
		result.partialEvents++
	case "final":
		result.finalEvents++
		result.finalTextChars = len([]rune(strings.TrimSpace(event.Text)))
	case "error":
		code := stableSherpaStreamingFinding(event.Code)
		if code == "" {
			code = "helper_runtime_failed"
		}
		return code
	default:
		return "helper_protocol_error"
	}
	return ""
}

func normalizeSherpaStreamingFamily(raw string) string {
	family := strings.ToLower(strings.TrimSpace(raw))
	family = strings.ReplaceAll(family, "-", "_")
	if family == "" {
		return "streaming_zipformer"
	}
	return family
}

func missingSherpaStreamingModelFiles(modelDir string) []string {
	if strings.TrimSpace(modelDir) == "" || !dirExists(modelDir) {
		return []string{"encoder.int8.onnx", "decoder.onnx", "joiner.int8.onnx", "tokens.txt"}
	}
	required := []string{"encoder.int8.onnx", "decoder.onnx", "joiner.int8.onnx", "tokens.txt"}
	var missing []string
	for _, name := range required {
		if !pathExists(filepath.Join(modelDir, name)) {
			missing = append(missing, name)
		}
	}
	return missing
}

func stableSherpaStreamingFinding(code string) string {
	switch strings.TrimSpace(code) {
	case "invalid_pcm",
		"model_dir_missing",
		"model_files_missing",
		"session_not_started",
		"sherpa_onnx_unavailable",
		"unsupported_streaming_family",
		"unknown_command":
		return strings.TrimSpace(code)
	default:
		return ""
	}
}

func sherpaStreamingBlockerFinding(finding string) bool {
	switch finding {
	case "model_dir_missing", "model_files_missing", "sherpa_onnx_unavailable", "streaming_helper_missing", "unsupported_streaming_family":
		return true
	default:
		return false
	}
}

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func safeSmokeBaseName(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Base(filepath.Clean(path))
}

func parsePositiveIntOption(raw string, name string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s requires a positive integer", name)
	}
	return value, nil
}

func writeLocalASRStreamingSmokeReport(outputDir string, report localASRStreamingSmokeReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-local-asr-streaming-smoke-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.OpenFile(reportPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(filepath.Clean(reportPath))
	if err := writeJSONLocalASRStreamingSmoke(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeJSONLocalASRStreamingSmoke(writer io.Writer, report localASRStreamingSmokeReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
