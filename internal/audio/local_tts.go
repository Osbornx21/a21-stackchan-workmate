package audio

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LocalTTSOptions struct {
	Text                         string
	Voice                        string
	OutputDir                    string
	CommandRunner                CommandRunner
	SayPath                      string
	AFConvertPath                string
	PythonPath                   string
	ScriptPath                   string
	ModelDir                     string
	SpeakerID                    int
	OutputSampleRateHz           int
	VoiceCloneCommand            string
	VoiceCloneModel              string
	VoiceCloneReferenceAudioPath string
	VoiceCloneReferenceText      string
	VoiceCloneReferenceTextPath  string
	VoiceClonePersona            string
	VoiceCloneStyle              string
}

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
}

type execCommandRunner struct{}

type LocalTTSReport struct {
	SchemaVersion   string            `json:"schema_version"`
	GeneratedAtMS   int64             `json:"generated_at_ms"`
	Status          string            `json:"status"`
	Provider        string            `json:"provider"`
	Engine          string            `json:"engine,omitempty"`
	Voice           string            `json:"voice"`
	Model           string            `json:"model,omitempty"`
	VoicePersona    string            `json:"voice_persona,omitempty"`
	StyleProfile    string            `json:"style_profile,omitempty"`
	ReferenceAudio  string            `json:"reference_audio,omitempty"`
	ModelDir        string            `json:"model_dir,omitempty"`
	OutputFormat    string            `json:"output_format"`
	OutputPath      string            `json:"output_path,omitempty"`
	ReportPath      string            `json:"report_path,omitempty"`
	OutputBytes     int64             `json:"output_bytes,omitempty"`
	TextBytes       int               `json:"text_bytes,omitempty"`
	DurationMS      float64           `json:"duration_ms,omitempty"`
	TTSFirstAudioMS float64           `json:"tts_first_audio_ms,omitempty"`
	AudioQuality    *PCMQualityReport `json:"audio_quality,omitempty"`
	Findings        []string          `json:"findings,omitempty"`
}

func (execCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

func SynthesizeMacOSSay(ctx context.Context, options LocalTTSOptions) (LocalTTSReport, error) {
	start := time.Now()
	outputSampleRateHz := localTTSOutputSampleRateHz(options)
	report := LocalTTSReport{
		SchemaVersion: "a21.audio.local_tts.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "failed",
		Provider:      "macos_say",
		Voice:         firstNonEmptyLocalTTS(options.Voice, "Tingting"),
		OutputFormat:  fmt.Sprintf("wav_pcm_s16le_%d_mono", outputSampleRateHz),
		TextBytes:     len([]byte(options.Text)),
	}
	if strings.TrimSpace(options.Text) == "" {
		report.Findings = append(report.Findings, "text is required")
		return report, fmt.Errorf("local TTS text is required")
	}
	outputDir := strings.TrimSpace(options.OutputDir)
	if outputDir == "" {
		outputDir = "reports"
	}
	if err := validateLocalTTSOutputDir(outputDir); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	sayPath := options.SayPath
	if sayPath == "" {
		found, err := exec.LookPath("say")
		if err != nil {
			report.Status = "skipped"
			report.Findings = append(report.Findings, "macOS say command is missing")
			return report, nil
		}
		sayPath = found
	}
	afconvertPath := options.AFConvertPath
	if afconvertPath == "" {
		found, err := exec.LookPath("afconvert")
		if err != nil {
			report.Status = "skipped"
			report.Findings = append(report.Findings, "macOS afconvert command is missing")
			return report, nil
		}
		afconvertPath = found
	}
	runner := options.CommandRunner
	if runner == nil {
		runner = execCommandRunner{}
	}
	tempDir, err := os.MkdirTemp("", "a21-local-tts-*")
	if err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	defer os.RemoveAll(tempDir)

	aiffPath := filepath.Join(tempDir, "a21-local-tts.aiff")
	wavPath := filepath.Join(outputDir, uniqueLocalTTSFilename("a21-local-tts"))
	if err := runner.Run(ctx, sayPath, "-v", report.Voice, "-o", aiffPath, options.Text); err != nil {
		report.Findings = append(report.Findings, "say command failed")
		return report, err
	}
	if err := runner.Run(ctx, afconvertPath, "-f", "WAVE", "-d", fmt.Sprintf("LEI16@%d", outputSampleRateHz), aiffPath, wavPath); err != nil {
		report.Findings = append(report.Findings, "afconvert command failed")
		return report, err
	}
	info, err := os.Stat(wavPath)
	if err != nil {
		report.Findings = append(report.Findings, "output wav missing")
		return report, err
	}
	report.Status = "passed"
	report.OutputPath = wavPath
	report.OutputBytes = info.Size()
	report.DurationMS = elapsedLocalTTSMS(start)
	report.TTSFirstAudioMS = report.DurationMS
	attachLocalTTSAudioQuality(&report, outputSampleRateHz)
	return report, nil
}

func SynthesizeSherpaONNX(ctx context.Context, options LocalTTSOptions) (LocalTTSReport, error) {
	start := time.Now()
	outputSampleRateHz := localTTSOutputSampleRateHz(options)
	speakerID := options.SpeakerID
	if speakerID == 0 {
		speakerID = 21
	}
	report := LocalTTSReport{
		SchemaVersion: "a21.audio.local_tts.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "failed",
		Provider:      "sherpa_onnx",
		Engine:        "vits_icefall_zh_aishell3",
		Voice:         "sid_" + strconv.Itoa(speakerID),
		OutputFormat:  fmt.Sprintf("wav_pcm_s16le_%d_mono", outputSampleRateHz),
		TextBytes:     len([]byte(options.Text)),
	}
	if strings.TrimSpace(options.Text) == "" {
		report.Findings = append(report.Findings, "text is required")
		return report, fmt.Errorf("local TTS text is required")
	}
	outputDir := strings.TrimSpace(options.OutputDir)
	if outputDir == "" {
		outputDir = "reports"
	}
	if err := validateLocalTTSOutputDir(outputDir); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	modelDir, modelDirExplicit := firstNonEmptyLocalTTS(options.ModelDir, defaultSherpaONNXModelDir()), strings.TrimSpace(options.ModelDir) != ""
	report.ModelDir = filepath.Base(filepath.Clean(modelDir))
	if err := validateSherpaONNXModelDir(modelDir); err != nil {
		report.Findings = append(report.Findings, err.Error())
		if !modelDirExplicit {
			report.Status = "skipped"
			return report, nil
		}
		return report, err
	}
	pythonPath, ok := resolveLocalTTSPath(options.PythonPath, defaultSherpaONNXPythonPath())
	if !ok {
		report.Status = "skipped"
		report.Findings = append(report.Findings, "isolated sherpa-onnx python is missing")
		return report, nil
	}
	scriptPath, ok := resolveLocalTTSPath(options.ScriptPath, defaultSherpaONNXScriptPath())
	if !ok {
		report.Status = "skipped"
		report.Findings = append(report.Findings, "a21 sherpa-onnx script is missing")
		return report, nil
	}
	afconvertPath := options.AFConvertPath
	if afconvertPath == "" {
		found, err := exec.LookPath("afconvert")
		if err != nil {
			report.Status = "skipped"
			report.Findings = append(report.Findings, "macOS afconvert command is missing")
			return report, nil
		}
		afconvertPath = found
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	runner := options.CommandRunner
	if runner == nil {
		runner = execCommandRunner{}
	}
	tempDir, err := os.MkdirTemp("", "a21-sherpa-onnx-tts-*")
	if err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	defer os.RemoveAll(tempDir)

	textPath := filepath.Join(tempDir, "a21-tts-input.txt")
	rawWAVPath := filepath.Join(tempDir, "a21-sherpa-onnx-tts.raw.wav")
	wavPath := filepath.Join(outputDir, uniqueLocalTTSFilename("a21-sherpa-onnx-tts"))
	if err := os.WriteFile(textPath, []byte(options.Text), 0o600); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	if err := runner.Run(ctx, pythonPath, scriptPath, "--model-dir", modelDir, "--text-file", textPath, "--output", rawWAVPath, "--speaker-id", strconv.Itoa(speakerID)); err != nil {
		report.Findings = append(report.Findings, "sherpa-onnx synthesis command failed")
		return report, err
	}
	if err := runner.Run(ctx, afconvertPath, "-f", "WAVE", "-d", fmt.Sprintf("LEI16@%d", outputSampleRateHz), rawWAVPath, wavPath); err != nil {
		report.Findings = append(report.Findings, "afconvert command failed")
		return report, err
	}
	info, err := os.Stat(wavPath)
	if err != nil {
		report.Findings = append(report.Findings, "output wav missing")
		return report, err
	}
	report.Status = "passed"
	report.OutputPath = wavPath
	report.OutputBytes = info.Size()
	report.DurationMS = elapsedLocalTTSMS(start)
	report.TTSFirstAudioMS = report.DurationMS
	attachLocalTTSAudioQuality(&report, outputSampleRateHz)
	return report, nil
}

func SynthesizeVoiceCloneCLI(ctx context.Context, options LocalTTSOptions) (LocalTTSReport, error) {
	start := time.Now()
	outputSampleRateHz := localTTSOutputSampleRateHz(options)
	model := safeLocalTTSIdentifier(firstNonEmptyLocalTTS(options.VoiceCloneModel, "index_tts2"))
	persona := safeLocalTTSIdentifier(firstNonEmptyLocalTTS(options.VoiceClonePersona, options.Voice, "a21_workmate"))
	style := safeLocalTTSIdentifier(firstNonEmptyLocalTTS(options.VoiceCloneStyle, "workmate_warm"))
	report := LocalTTSReport{
		SchemaVersion:  "a21.audio.local_tts.v1",
		GeneratedAtMS:  time.Now().UnixMilli(),
		Status:         "failed",
		Provider:       "voice_clone_cli",
		Engine:         "voice_clone_cli",
		Voice:          persona,
		Model:          model,
		VoicePersona:   persona,
		StyleProfile:   style,
		OutputFormat:   fmt.Sprintf("wav_pcm_s16le_%d_mono", outputSampleRateHz),
		TextBytes:      len([]byte(options.Text)),
		ReferenceAudio: localTTSBaseName(options.VoiceCloneReferenceAudioPath),
	}
	if strings.TrimSpace(options.Text) == "" {
		report.Findings = append(report.Findings, "text is required")
		return report, fmt.Errorf("local TTS text is required")
	}
	outputDir := strings.TrimSpace(options.OutputDir)
	if outputDir == "" {
		outputDir = "reports"
	}
	if err := validateLocalTTSOutputDir(outputDir); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	commandPath := strings.TrimSpace(options.VoiceCloneCommand)
	if commandPath == "" {
		report.Status = "skipped"
		report.Findings = append(report.Findings, "voice clone command is missing")
		return report, nil
	}
	referenceAudioPath := strings.TrimSpace(options.VoiceCloneReferenceAudioPath)
	if err := validateVoiceCloneReferenceAudioPath(referenceAudioPath); err != nil {
		report.Status = "skipped"
		report.Findings = append(report.Findings, err.Error())
		return report, nil
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	runner := options.CommandRunner
	if runner == nil {
		runner = execCommandRunner{}
	}
	tempDir, err := os.MkdirTemp("", "a21-voice-clone-tts-*")
	if err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	defer os.RemoveAll(tempDir)

	textPath := filepath.Join(tempDir, "a21-voice-clone-input.txt")
	refTextPath := filepath.Join(tempDir, "a21-voice-clone-reference.txt")
	wavPath := filepath.Join(outputDir, uniqueLocalTTSFilename("a21-voice-clone-tts"))
	if err := os.WriteFile(textPath, []byte(options.Text), 0o600); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	refText := strings.TrimSpace(options.VoiceCloneReferenceText)
	if strings.TrimSpace(options.VoiceCloneReferenceTextPath) != "" {
		data, err := os.ReadFile(options.VoiceCloneReferenceTextPath)
		if err != nil {
			report.Findings = append(report.Findings, "voice clone reference text is unavailable")
			return report, err
		}
		refText = strings.TrimSpace(string(data))
	}
	if err := os.WriteFile(refTextPath, []byte(refText), 0o600); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	args := []string{
		"--text-file", textPath,
		"--output", wavPath,
		"--sample-rate", strconv.Itoa(outputSampleRateHz),
		"--ref-audio", referenceAudioPath,
		"--ref-text-file", refTextPath,
		"--model", model,
		"--persona", persona,
		"--style", style,
	}
	if err := runner.Run(ctx, commandPath, args...); err != nil {
		report.Findings = append(report.Findings, "voice clone command failed")
		return report, err
	}
	info, err := os.Stat(wavPath)
	if err != nil {
		report.Findings = append(report.Findings, "output wav missing")
		return report, err
	}
	report.Status = "passed"
	report.OutputPath = wavPath
	report.OutputBytes = info.Size()
	report.DurationMS = elapsedLocalTTSMS(start)
	report.TTSFirstAudioMS = report.DurationMS
	attachLocalTTSAudioQuality(&report, outputSampleRateHz)
	return report, nil
}

func attachLocalTTSAudioQuality(report *LocalTTSReport, expectedSampleRateHz int) {
	if report == nil || strings.TrimSpace(report.OutputPath) == "" {
		return
	}
	quality, err := AnalyzePCM16MonoWAVQuality(report.OutputPath, expectedSampleRateHz)
	if err != nil {
		report.Findings = append(report.Findings, "audio_quality_unavailable")
		return
	}
	report.AudioQuality = &quality
	report.Findings = appendUniqueLocalTTSFindings(report.Findings, quality.Findings...)
}

func appendUniqueLocalTTSFindings(findings []string, values ...string) []string {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		seen := false
		for _, existing := range findings {
			if existing == value {
				seen = true
				break
			}
		}
		if !seen {
			findings = append(findings, value)
		}
	}
	return findings
}

func validateLocalTTSOutputDir(path string) error {
	lower := strings.ToLower(filepath.Clean(path))
	if strings.Contains(lower, "x21") || strings.Contains(lower, "v21") {
		return fmt.Errorf("local TTS output dir contains forbidden legacy project identity")
	}
	return nil
}

func validateVoiceCloneReferenceAudioPath(path string) error {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	lower := strings.ToLower(cleaned)
	if cleaned == "" || cleaned == "." {
		return fmt.Errorf("voice clone reference audio is required")
	}
	if strings.Contains(lower, "x21") || strings.Contains(lower, "v21") {
		return fmt.Errorf("voice clone reference audio contains forbidden legacy project identity")
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		return fmt.Errorf("voice clone reference audio is unavailable")
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("voice clone reference audio is not a file")
	}
	return nil
}

func safeLocalTTSIdentifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "\t", "_")
	value = strings.ReplaceAll(value, ".", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, ":", "_")
	if value == "" {
		return "a21_voice"
	}
	var out strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			if r == '_' {
				if lastUnderscore {
					continue
				}
				lastUnderscore = true
			} else {
				lastUnderscore = false
			}
			out.WriteRune(r)
		}
	}
	cleaned := strings.Trim(out.String(), "_")
	if cleaned == "" {
		return "a21_voice"
	}
	if len(cleaned) > 64 {
		return cleaned[:64]
	}
	return cleaned
}

func localTTSBaseName(path string) string {
	path = strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
	if path == "" {
		return ""
	}
	base := filepath.Base(filepath.Clean(path))
	if base == "." {
		return ""
	}
	return base
}

func uniqueLocalTTSFilename(prefix string) string {
	now := time.Now()
	return fmt.Sprintf("%s-%s-%09d.wav", prefix, now.Format("20060102-150405"), now.Nanosecond())
}

func validateSherpaONNXModelDir(path string) error {
	cleaned := filepath.Clean(path)
	lower := strings.ToLower(cleaned)
	if strings.Contains(lower, "x21") || strings.Contains(lower, "v21") {
		return fmt.Errorf("sherpa-onnx model dir contains forbidden legacy project identity")
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		return fmt.Errorf("sherpa-onnx model dir is unavailable")
	}
	if !info.IsDir() {
		return fmt.Errorf("sherpa-onnx model dir is not a directory")
	}
	for _, name := range []string{"model.onnx", "lexicon.txt", "tokens.txt", "phone.fst", "date.fst", "number.fst"} {
		if _, err := os.Stat(filepath.Join(cleaned, name)); err != nil {
			return fmt.Errorf("sherpa-onnx model file %s is missing", name)
		}
	}
	return nil
}

func SherpaONNXTTSModelDirReady(path string) bool {
	return validateSherpaONNXModelDir(path) == nil
}

func resolveLocalTTSPath(explicitPath string, defaultPath string) (string, bool) {
	if strings.TrimSpace(explicitPath) != "" {
		return strings.TrimSpace(explicitPath), true
	}
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, true
	}
	return "", false
}

func defaultSherpaONNXModelDir() string {
	return filepath.Join(".a21-tools", "sherpa-onnx-models", "vits-icefall-zh-aishell3")
}

func DefaultSherpaONNXTTSModelDir() string {
	return defaultSherpaONNXModelDir()
}

func defaultSherpaONNXPythonPath() string {
	return filepath.Join(".a21-tools", "sherpa-onnx-venv", "bin", "python")
}

func defaultSherpaONNXScriptPath() string {
	return filepath.Join("scripts", "a21_sherpa_onnx_tts.py")
}

func firstNonEmptyLocalTTS(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func elapsedLocalTTSMS(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000
}

func localTTSOutputSampleRateHz(options LocalTTSOptions) int {
	if options.OutputSampleRateHz > 0 {
		return options.OutputSampleRateHz
	}
	return 16000
}
