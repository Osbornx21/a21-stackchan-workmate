package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type LocalASROptions struct {
	OutputDir  string
	PythonPath string
	ScriptPath string
	ModelDir   string
	Family     string
	WAVPath    string
}

type LocalASRReport struct {
	SchemaVersion    string   `json:"schema_version"`
	GeneratedAtMS    int64    `json:"generated_at_ms"`
	Status           string   `json:"status"`
	Provider         string   `json:"provider"`
	Engine           string   `json:"engine"`
	ModelDir         string   `json:"model_dir,omitempty"`
	WAVName          string   `json:"wav_name,omitempty"`
	InputDurationMS  float64  `json:"input_duration_ms,omitempty"`
	DecodeDurationMS float64  `json:"decode_duration_ms,omitempty"`
	RealTimeFactor   float64  `json:"real_time_factor,omitempty"`
	TextChars        int      `json:"text_chars,omitempty"`
	TranscriptPolicy string   `json:"transcript_policy"`
	ReportPath       string   `json:"report_path,omitempty"`
	Findings         []string `json:"findings,omitempty"`
}

type sherpaONNXASRScriptReport struct {
	Status           string  `json:"status"`
	InputDurationMS  float64 `json:"input_duration_ms"`
	DecodeDurationMS float64 `json:"decode_duration_ms"`
	RealTimeFactor   float64 `json:"real_time_factor"`
	TextChars        int     `json:"text_chars"`
}

func RunSherpaONNXASRSmoke(ctx context.Context, options LocalASROptions) (LocalASRReport, error) {
	start := time.Now()
	modelDir, modelDirExplicit := firstNonEmptyLocalTTS(options.ModelDir, defaultSherpaONNXASRModelDir()), strings.TrimSpace(options.ModelDir) != ""
	family, err := normalizeSherpaONNXASRFamily(firstNonEmptyLocalTTS(options.Family, inferSherpaONNXASRFamily(modelDir)))
	if err != nil {
		return LocalASRReport{}, err
	}
	wavPath := firstNonEmptyLocalTTS(options.WAVPath, defaultSherpaONNXASRWAVPath(modelDir, family))
	report := LocalASRReport{
		SchemaVersion:    "a21.audio.local_asr.v1",
		GeneratedAtMS:    time.Now().UnixMilli(),
		Status:           "failed",
		Provider:         "sherpa_onnx",
		Engine:           family,
		ModelDir:         filepath.Base(filepath.Clean(modelDir)),
		WAVName:          filepath.Base(filepath.Clean(wavPath)),
		TranscriptPolicy: "transcript_not_recorded",
	}
	outputDir := strings.TrimSpace(options.OutputDir)
	if outputDir == "" {
		outputDir = "reports"
	}
	if err := validateLocalTTSOutputDir(outputDir); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	if err := validateSherpaONNXASRModelDir(modelDir, family); err != nil {
		report.Findings = append(report.Findings, err.Error())
		if !modelDirExplicit {
			report.Status = "skipped"
			return report, nil
		}
		return report, err
	}
	if err := validateSherpaONNXASRWAVPath(wavPath); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	pythonPath, ok := resolveLocalTTSPath(options.PythonPath, defaultSherpaONNXPythonPath())
	if !ok {
		report.Status = "skipped"
		report.Findings = append(report.Findings, "isolated sherpa-onnx python is missing")
		return report, nil
	}
	scriptPath, ok := resolveLocalTTSPath(options.ScriptPath, defaultSherpaONNXASRScriptPath())
	if !ok {
		report.Status = "skipped"
		report.Findings = append(report.Findings, "a21 sherpa-onnx ASR script is missing")
		return report, nil
	}
	output, err := runSherpaONNXASRScript(ctx, pythonPath, scriptPath, family, modelDir, wavPath)
	if err != nil {
		report.Findings = append(report.Findings, "sherpa-onnx ASR command failed")
		return report, err
	}
	var scriptReport sherpaONNXASRScriptReport
	if err := json.Unmarshal(output, &scriptReport); err != nil {
		report.Findings = append(report.Findings, "sherpa-onnx ASR output was not valid JSON")
		return report, err
	}
	if scriptReport.Status != "passed" {
		report.Findings = append(report.Findings, "sherpa-onnx ASR script did not pass")
		return report, nil
	}
	report.Status = "passed"
	report.InputDurationMS = scriptReport.InputDurationMS
	report.DecodeDurationMS = scriptReport.DecodeDurationMS
	report.RealTimeFactor = scriptReport.RealTimeFactor
	report.TextChars = scriptReport.TextChars
	if report.DecodeDurationMS == 0 {
		report.DecodeDurationMS = elapsedLocalTTSMS(start)
	}
	return report, nil
}

func runSherpaONNXASRScript(ctx context.Context, pythonPath string, scriptPath string, family string, modelDir string, wavPath string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, pythonPath, scriptPath, "--family", family, "--model-dir", modelDir, "--wav", wavPath)
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return output, nil
}

func normalizeSherpaONNXASRFamily(raw string) (string, error) {
	family := strings.ToLower(strings.TrimSpace(raw))
	family = strings.ReplaceAll(family, "-", "_")
	switch family {
	case "", "paraformer":
		return "paraformer", nil
	case "sensevoice", "sense_voice":
		return "sense_voice", nil
	case "zipformer", "streaming_zipformer", "streaming_transducer", "transducer":
		return "streaming_zipformer", nil
	default:
		return "", fmt.Errorf("unsupported local ASR family")
	}
}

func inferSherpaONNXASRFamily(modelDir string) string {
	lower := strings.ToLower(filepath.Base(filepath.Clean(modelDir)))
	switch {
	case strings.Contains(lower, "sense"):
		return "sense_voice"
	case strings.Contains(lower, "zipformer") || strings.Contains(lower, "streaming"):
		return "streaming_zipformer"
	default:
		return "paraformer"
	}
}

func validateSherpaONNXASRModelDir(path string, family string) error {
	cleaned := filepath.Clean(path)
	lower := strings.ToLower(cleaned)
	if strings.Contains(lower, "x21") || strings.Contains(lower, "v21") {
		return fmt.Errorf("sherpa-onnx ASR model dir contains forbidden legacy project identity")
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		return fmt.Errorf("sherpa-onnx ASR model dir is unavailable")
	}
	if !info.IsDir() {
		return fmt.Errorf("sherpa-onnx ASR model dir is not a directory")
	}
	required := []string{"model.int8.onnx", "tokens.txt"}
	if family == "streaming_zipformer" {
		required = []string{"encoder.int8.onnx", "decoder.onnx", "joiner.int8.onnx", "tokens.txt"}
	}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(cleaned, name)); err != nil {
			return fmt.Errorf("sherpa-onnx ASR model file %s is missing", name)
		}
	}
	return nil
}

func validateSherpaONNXASRWAVPath(path string) error {
	cleaned := filepath.Clean(path)
	lower := strings.ToLower(cleaned)
	if strings.Contains(lower, "x21") || strings.Contains(lower, "v21") {
		return fmt.Errorf("sherpa-onnx ASR wav path contains forbidden legacy project identity")
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		return fmt.Errorf("sherpa-onnx ASR wav is unavailable")
	}
	if info.IsDir() {
		return fmt.Errorf("sherpa-onnx ASR wav path is a directory")
	}
	if strings.ToLower(filepath.Ext(cleaned)) != ".wav" {
		return fmt.Errorf("sherpa-onnx ASR input must be a wav file")
	}
	return nil
}

func defaultSherpaONNXASRModelDir() string {
	return filepath.Join(".a21-tools", "sherpa-onnx-asr-models", "sherpa-onnx-paraformer-zh-small-2024-03-09")
}

func defaultSherpaONNXASRWAVPath(modelDir string, family string) string {
	name := "0.wav"
	if family == "sense_voice" {
		name = "zh.wav"
	}
	return filepath.Join(modelDir, "test_wavs", name)
}

func defaultSherpaONNXASRScriptPath() string {
	return filepath.Join("scripts", "a21_sherpa_onnx_asr_smoke.py")
}
