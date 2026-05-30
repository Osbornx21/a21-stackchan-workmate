package audio

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeTTSCommandRunner struct {
	commands []string
}

func (r *fakeTTSCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	r.commands = append(r.commands, name+" "+strings.Join(args, " "))
	for i, arg := range args {
		if arg == "-o" && i+1 < len(args) {
			return os.WriteFile(args[i+1], []byte("a21-aiff"), 0o644)
		}
		if arg == "--output" && i+1 < len(args) {
			return os.WriteFile(args[i+1], []byte("RIFF-a21-sherpa-wav"), 0o644)
		}
	}
	if strings.Contains(name, "afconvert") && len(args) >= 2 {
		return os.WriteFile(args[len(args)-1], []byte("RIFF-a21-wav"), 0o644)
	}
	return nil
}

func TestMacOSSayLocalTTSSynthesizesRedactedWAVReport(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeTTSCommandRunner{}

	report, err := SynthesizeMacOSSay(context.Background(), LocalTTSOptions{
		Text:          "这句话不能出现在报告里",
		Voice:         "Tingting",
		OutputDir:     dir,
		CommandRunner: runner,
		SayPath:       "/usr/bin/say",
		AFConvertPath: "/usr/bin/afconvert",
	})

	if err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != "a21.audio.local_tts.v1" || report.Status != "passed" {
		t.Fatalf("report status = %+v", report)
	}
	if report.Provider != "macos_say" || report.OutputFormat != "wav_pcm_s16le_16000_mono" {
		t.Fatalf("provider/output = %q/%q", report.Provider, report.OutputFormat)
	}
	if report.OutputBytes <= 0 || report.DurationMS <= 0 || report.TTSFirstAudioMS <= 0 {
		t.Fatalf("timings/bytes not populated: %+v", report)
	}
	if !strings.HasPrefix(report.OutputPath, filepath.Clean(dir)+string(os.PathSeparator)) {
		t.Fatalf("output path %q not under %q", report.OutputPath, dir)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("commands = %#v, want say and afconvert", runner.commands)
	}
	rendered := mustJSON(t, report)
	for _, forbidden := range []string{"这句话不能出现在报告里", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestMacOSSayLocalTTSRejectsLegacyOutputDir(t *testing.T) {
	_, err := SynthesizeMacOSSay(context.Background(), LocalTTSOptions{
		Text:          "A21",
		OutputDir:     filepath.Join(t.TempDir(), "x21-reports"),
		CommandRunner: &fakeTTSCommandRunner{},
		SayPath:       "/usr/bin/say",
		AFConvertPath: "/usr/bin/afconvert",
	})

	if err == nil {
		t.Fatal("expected legacy output dir rejection")
	}
}

func TestSherpaONNXLocalTTSSynthesizesRedactedWAVReport(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(t.TempDir(), "vits-icefall-zh-aishell3")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"model.onnx", "lexicon.txt", "tokens.txt", "phone.fst", "date.fst", "number.fst"} {
		if err := os.WriteFile(filepath.Join(modelDir, name), []byte("fixture"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runner := &fakeTTSCommandRunner{}

	report, err := SynthesizeSherpaONNX(context.Background(), LocalTTSOptions{
		Text:          "这句话也不能出现在报告里",
		OutputDir:     dir,
		CommandRunner: runner,
		PythonPath:    "/a21/python",
		ScriptPath:    "/a21/scripts/a21_sherpa_onnx_tts.py",
		AFConvertPath: "/usr/bin/afconvert",
		ModelDir:      modelDir,
		SpeakerID:     21,
	})

	if err != nil {
		t.Fatal(err)
	}
	if report.Provider != "sherpa_onnx" || report.Engine != "vits_icefall_zh_aishell3" {
		t.Fatalf("provider/engine = %q/%q", report.Provider, report.Engine)
	}
	if report.ModelDir == "" || strings.Contains(report.ModelDir, modelDir) {
		t.Fatalf("model dir should be labeled, not full path: %q", report.ModelDir)
	}
	if report.OutputFormat != "wav_pcm_s16le_16000_mono" || report.OutputBytes <= 0 {
		t.Fatalf("output = %q bytes=%d", report.OutputFormat, report.OutputBytes)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("commands = %#v, want python and afconvert", runner.commands)
	}
	pythonCommand := runner.commands[0]
	for _, want := range []string{"/a21/python", "--model-dir", modelDir, "--speaker-id", "21"} {
		if !strings.Contains(pythonCommand, want) {
			t.Fatalf("python command missing %q: %s", want, pythonCommand)
		}
	}
	rendered := mustJSON(t, report)
	for _, forbidden := range []string{"这句话也不能出现在报告里", modelDir, "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestLocalTTSOutputPathsAreUniqueAcrossFastConsecutiveCalls(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeTTSCommandRunner{}

	first, err := SynthesizeMacOSSay(context.Background(), LocalTTSOptions{
		Text:          "A21 ack",
		OutputDir:     dir,
		CommandRunner: runner,
		SayPath:       "/usr/bin/say",
		AFConvertPath: "/usr/bin/afconvert",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := SynthesizeMacOSSay(context.Background(), LocalTTSOptions{
		Text:          "A21 answer",
		OutputDir:     dir,
		CommandRunner: runner,
		SayPath:       "/usr/bin/say",
		AFConvertPath: "/usr/bin/afconvert",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.OutputPath == second.OutputPath {
		t.Fatalf("output paths collided: %q", first.OutputPath)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
