package audio

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

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
	commandName, commandPrefixArgs, err := splitLocalTTSCommand(commandPath)
	if err != nil {
		report.Findings = append(report.Findings, "voice clone command is invalid")
		return report, err
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
	args := append([]string(nil), commandPrefixArgs...)
	args = append(args,
		"--text-file", textPath,
		"--output", wavPath,
		"--sample-rate", strconv.Itoa(outputSampleRateHz),
		"--ref-audio", referenceAudioPath,
		"--ref-text-file", refTextPath,
		"--model", model,
		"--persona", persona,
		"--style", style,
	)
	if err := runner.Run(ctx, commandName, args...); err != nil {
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

func splitLocalTTSCommand(command string) (string, []string, error) {
	fields, err := localTTSCommandFields(command)
	if err != nil || len(fields) == 0 {
		return "", nil, fmt.Errorf("voice clone command is invalid")
	}
	return fields[0], append([]string(nil), fields[1:]...), nil
}

func localTTSCommandFields(command string) ([]string, error) {
	var fields []string
	var current strings.Builder
	var quote rune
	escaped := false
	sawField := false
	for _, r := range strings.TrimSpace(command) {
		if escaped {
			current.WriteRune(r)
			escaped = false
			sawField = true
			continue
		}
		if quote != '\'' && r == '\\' {
			escaped = true
			sawField = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				sawField = true
				continue
			}
			current.WriteRune(r)
			sawField = true
			continue
		}
		switch {
		case r == '\'' || r == '"':
			quote = r
			sawField = true
		case unicode.IsSpace(r):
			if sawField {
				fields = append(fields, current.String())
				current.Reset()
				sawField = false
			}
		default:
			current.WriteRune(r)
			sawField = true
		}
	}
	if escaped || quote != 0 {
		return nil, fmt.Errorf("voice clone command is invalid")
	}
	if sawField {
		fields = append(fields, current.String())
	}
	return fields, nil
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
