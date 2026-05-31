package app

import (
	"a21.local/a21/internal/audio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var synthesizeMacOSSay = audio.SynthesizeMacOSSay
var synthesizeSherpaONNX = audio.SynthesizeSherpaONNX
var runSherpaONNXASR = audio.RunSherpaONNXASR
var runSherpaONNXASRSmoke = audio.RunSherpaONNXASRSmoke

func runAudioFrontEndPlan(args []string, stdout io.Writer, stderr io.Writer) int {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 audio-front-end-plan")
			return 0
		case "--execute":
			fmt.Fprintln(stderr, "audio-front-end-plan is plan-only and does not execute audio libraries")
			return 2
		default:
			fmt.Fprintf(stderr, "unknown audio-front-end-plan option %q\n", args[i])
			return 2
		}
	}
	if err := writeJSONAudioFrontEndPlan(stdout, audio.BaselineFrontEndPlan()); err != nil {
		fmt.Fprintf(stderr, "encode audio front-end plan: %v\n", err)
		return 1
	}
	return 0
}
func runAudioFrontEndEval(args []string, stdout io.Writer, stderr io.Writer) int {
	mock := false
	fixturePath := ""
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 audio-front-end-eval (--mock | --fixture reports/a21-audio-fixture.json) [--output-dir reports]")
			return 0
		case "--mock":
			mock = true
		case "--fixture":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--fixture requires a value")
				return 2
			}
			i++
			fixturePath = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		case "--execute":
			fmt.Fprintln(stderr, "audio-front-end-eval only supports --mock until recorded-office and physical-device fixtures exist")
			return 2
		default:
			fmt.Fprintf(stderr, "unknown audio-front-end-eval option %q\n", args[i])
			return 2
		}
	}
	if mock && fixturePath != "" {
		fmt.Fprintln(stderr, "audio-front-end-eval accepts either --mock or --fixture, not both")
		return 2
	}
	if !mock && fixturePath == "" {
		fmt.Fprintln(stderr, "audio-front-end-eval requires --mock or --fixture")
		return 2
	}
	var report audio.FrontEndEvalReport
	if mock {
		report = audio.RunMockFrontEndEval()
	} else {
		var err error
		report, err = audio.RunFrontEndEvalFromFixture(fixturePath)
		if err != nil {
			fmt.Fprintf(stderr, "audio front-end fixture eval failed for %s: %s\n", filepath.Base(fixturePath), redactAudioFrontEndFixtureEvalError(fixturePath, err))
			return 1
		}
	}
	cliReport := buildAudioFrontEndEvalCLIReport(report)
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "audio front-end report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeAudioFrontEndEvalReport(outputDir, cliReport)
		if err != nil {
			fmt.Fprintf(stderr, "write audio front-end eval report: %v\n", err)
			return 1
		}
		cliReport.ReportPath = reportPath
	}
	if err := writeJSONAudioFrontEndEval(stdout, cliReport); err != nil {
		fmt.Fprintf(stderr, "encode audio front-end eval: %v\n", err)
		return 1
	}
	return 0
}
func redactAudioFrontEndFixtureEvalError(path string, err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if path == "" {
		return message
	}
	message = strings.ReplaceAll(message, path, filepath.Base(path))
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		message = strings.ReplaceAll(message, dir, "[redacted-dir]")
	}
	return message
}
func runLocalTTSSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	text := "A21 本地语音链路测试。"
	engine := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_TTS_ENGINE"), "sherpa_onnx"))
	voice := strings.TrimSpace(os.Getenv("A21_LOCAL_TTS_VOICE"))
	modelDir := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_MODEL_DIR"))
	speakerID := parsePositiveIntOrDefault(os.Getenv("A21_SHERPA_ONNX_SPEAKER_ID"), 21)
	outputDir := "reports"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 local-tts-smoke [--engine sherpa_onnx|macos_say] [--text <text>] [--voice Tingting] [--model-dir <dir>] [--speaker-id 21] [--output-dir reports]")
			return 0
		case "--engine":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--engine requires a value")
				return 2
			}
			i++
			engine = args[i]
		case "--text":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--text requires a value")
				return 2
			}
			i++
			text = args[i]
		case "--voice":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--voice requires a value")
				return 2
			}
			i++
			voice = args[i]
		case "--model-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--model-dir requires a value")
				return 2
			}
			i++
			modelDir = args[i]
		case "--speaker-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--speaker-id requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 {
				fmt.Fprintln(stderr, "--speaker-id must be a positive integer")
				return 2
			}
			speakerID = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown local-tts-smoke option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(outputDir); err != nil {
		fmt.Fprintf(stderr, "local TTS report dir invalid: %v\n", err)
		return 1
	}
	report, err := synthesizeLocalTTS(context.Background(), localTTSRuntimeOptions{
		Engine:    engine,
		Text:      text,
		Voice:     voice,
		ModelDir:  modelDir,
		SpeakerID: speakerID,
		OutputDir: outputDir,
	})
	if err != nil {
		fmt.Fprintf(stderr, "local TTS smoke failed: %v\n", err)
		return 1
	}
	reportPath, err := writeLocalTTSSmokeReport(outputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write local TTS smoke report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONLocalTTSSmoke(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode local TTS smoke report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}

type localTTSRuntimeOptions struct {
	Engine    string
	Text      string
	Voice     string
	ModelDir  string
	SpeakerID int
	OutputDir string
}

func synthesizeLocalTTS(ctx context.Context, options localTTSRuntimeOptions) (audio.LocalTTSReport, error) {
	engine, err := normalizeLocalTTSEngine(options.Engine)
	if err != nil {
		return audio.LocalTTSReport{}, err
	}
	ttsOptions := audio.LocalTTSOptions{
		Text:      options.Text,
		Voice:     options.Voice,
		OutputDir: options.OutputDir,
		ModelDir:  options.ModelDir,
		SpeakerID: options.SpeakerID,
	}
	switch engine {
	case "macos_say":
		return synthesizeMacOSSay(ctx, ttsOptions)
	case "sherpa_onnx":
		return synthesizeSherpaONNX(ctx, ttsOptions)
	default:
		return audio.LocalTTSReport{}, fmt.Errorf("unsupported local TTS engine")
	}
}
func normalizeLocalTTSEngine(raw string) (string, error) {
	engine := strings.ToLower(strings.TrimSpace(firstNonEmpty(raw, "sherpa_onnx")))
	engine = strings.ReplaceAll(engine, "-", "_")
	switch engine {
	case "sherpa", "sherpa_onnx":
		return "sherpa_onnx", nil
	case "macos", "macos_say", "say":
		return "macos_say", nil
	default:
		return "", fmt.Errorf("unsupported local TTS engine")
	}
}
func runLocalASRSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	engine := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_ASR_ENGINE"), "sherpa_onnx"))
	family := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_FAMILY"))
	modelDir := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_MODEL_DIR"))
	wavPath := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_WAV"))
	outputDir := "reports"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 local-asr-smoke [--engine sherpa_onnx] [--family paraformer|sense_voice|streaming_zipformer] [--model-dir <dir>] [--wav <path>] [--output-dir reports]")
			return 0
		case "--engine":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--engine requires a value")
				return 2
			}
			i++
			engine = args[i]
		case "--family":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--family requires a value")
				return 2
			}
			i++
			family = args[i]
		case "--model-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--model-dir requires a value")
				return 2
			}
			i++
			modelDir = args[i]
		case "--wav":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--wav requires a value")
				return 2
			}
			i++
			wavPath = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown local-asr-smoke option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(outputDir); err != nil {
		fmt.Fprintf(stderr, "local ASR report dir invalid: %v\n", err)
		return 1
	}
	if engine, err := normalizeLocalASREngine(engine); err != nil {
		fmt.Fprintf(stderr, "local ASR engine invalid: %v\n", err)
		return 1
	} else if engine != "sherpa_onnx" {
		fmt.Fprintln(stderr, "unsupported local ASR engine")
		return 1
	}
	report, err := runSherpaONNXASRSmoke(context.Background(), audio.LocalASROptions{
		OutputDir: outputDir,
		ModelDir:  modelDir,
		Family:    family,
		WAVPath:   wavPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "local ASR smoke failed: %v\n", err)
		return 1
	}
	reportPath, err := writeLocalASRSmokeReport(outputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write local ASR smoke report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONLocalASRSmoke(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode local ASR smoke report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}
func normalizeLocalASREngine(raw string) (string, error) {
	engine := strings.ToLower(strings.TrimSpace(firstNonEmpty(raw, "sherpa_onnx")))
	engine = strings.ReplaceAll(engine, "-", "_")
	switch engine {
	case "sherpa", "sherpa_onnx":
		return "sherpa_onnx", nil
	default:
		return "", fmt.Errorf("unsupported local ASR engine")
	}
}

type audioFrontEndEvalCLIReport struct {
	audio.FrontEndEvalReport
	Metadata latencyBenchMetadata `json:"metadata"`
}

func buildAudioFrontEndEvalCLIReport(report audio.FrontEndEvalReport) audioFrontEndEvalCLIReport {
	return audioFrontEndEvalCLIReport{
		FrontEndEvalReport: report,
		Metadata:           buildLatencyBenchMetadata(),
	}
}
func writeAudioFrontEndEvalReport(outputDir string, report audioFrontEndEvalCLIReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-audio-front-end-eval-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONAudioFrontEndEval(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}
func writeLocalTTSSmokeReport(outputDir string, report audio.LocalTTSReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-local-tts-smoke-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONLocalTTSSmoke(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeLocalASRSmokeReport(outputDir string, report audio.LocalASRReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-local-asr-smoke-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONLocalASRSmoke(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONAudioFrontEndPlan(writer io.Writer, report audio.FrontEndPlan) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
func writeJSONAudioFrontEndEval(writer io.Writer, report audioFrontEndEvalCLIReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
func writeJSONLocalTTSSmoke(writer io.Writer, report audio.LocalTTSReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
func writeJSONLocalASRSmoke(writer io.Writer, report audio.LocalASRReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
