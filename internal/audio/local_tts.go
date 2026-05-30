package audio

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type LocalTTSOptions struct {
	Text          string
	Voice         string
	OutputDir     string
	CommandRunner CommandRunner
	SayPath       string
	AFConvertPath string
}

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
}

type execCommandRunner struct{}

type LocalTTSReport struct {
	SchemaVersion   string   `json:"schema_version"`
	GeneratedAtMS   int64    `json:"generated_at_ms"`
	Status          string   `json:"status"`
	Provider        string   `json:"provider"`
	Voice           string   `json:"voice"`
	OutputFormat    string   `json:"output_format"`
	OutputPath      string   `json:"output_path,omitempty"`
	ReportPath      string   `json:"report_path,omitempty"`
	OutputBytes     int64    `json:"output_bytes,omitempty"`
	TextBytes       int      `json:"text_bytes,omitempty"`
	DurationMS      float64  `json:"duration_ms,omitempty"`
	TTSFirstAudioMS float64  `json:"tts_first_audio_ms,omitempty"`
	Findings        []string `json:"findings,omitempty"`
}

func (execCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

func SynthesizeMacOSSay(ctx context.Context, options LocalTTSOptions) (LocalTTSReport, error) {
	start := time.Now()
	report := LocalTTSReport{
		SchemaVersion: "a21.audio.local_tts.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "failed",
		Provider:      "macos_say",
		Voice:         firstNonEmptyLocalTTS(options.Voice, "Tingting"),
		OutputFormat:  "wav_pcm_s16le_16000_mono",
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
	wavPath := filepath.Join(outputDir, "a21-local-tts-"+time.Now().Format("20060102-150405")+".wav")
	if err := runner.Run(ctx, sayPath, "-v", report.Voice, "-o", aiffPath, options.Text); err != nil {
		report.Findings = append(report.Findings, "say command failed")
		return report, err
	}
	if err := runner.Run(ctx, afconvertPath, "-f", "WAVE", "-d", "LEI16@16000", aiffPath, wavPath); err != nil {
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
	return report, nil
}

func validateLocalTTSOutputDir(path string) error {
	lower := strings.ToLower(filepath.Clean(path))
	if strings.Contains(lower, "x21") || strings.Contains(lower, "v21") {
		return fmt.Errorf("local TTS output dir contains forbidden legacy project identity")
	}
	return nil
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
