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

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
