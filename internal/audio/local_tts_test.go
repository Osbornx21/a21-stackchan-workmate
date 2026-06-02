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

type validWAVTTSCommandRunner struct{}

func (validWAVTTSCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	for i, arg := range args {
		if arg == "--output" && i+1 < len(args) {
			return WritePCM16MonoWAV(args[i+1], 16000, pcm16Bytes(0, 900, -900, 1600, -1600))
		}
	}
	if strings.Contains(name, "afconvert") && len(args) >= 1 {
		return WritePCM16MonoWAV(args[len(args)-1], 16000, pcm16Bytes(0, 900, -900, 1600, -1600))
	}
	for i, arg := range args {
		if arg == "-o" && i+1 < len(args) {
			return os.WriteFile(args[i+1], []byte("a21-aiff"), 0o644)
		}
	}
	return nil
}

type recordingVoiceCloneRunner struct {
	name string
	args []string
}

func (r *recordingVoiceCloneRunner) Run(ctx context.Context, name string, args ...string) error {
	r.name = name
	r.args = append([]string(nil), args...)
	for i, arg := range args {
		if arg == "--output" && i+1 < len(args) {
			return WritePCM16MonoWAV(args[i+1], 16000, pcm16Bytes(0, 900, -900, 1600, -1600))
		}
	}
	return nil
}

type inspectingVoiceCloneRunner struct {
	name    string
	args    []string
	text    string
	refText string
}

func (r *inspectingVoiceCloneRunner) Run(ctx context.Context, name string, args ...string) error {
	r.name = name
	r.args = append([]string(nil), args...)
	textPath := argValue(args, "--text-file")
	refTextPath := argValue(args, "--ref-text-file")
	outputPath := argValue(args, "--output")
	textData, err := os.ReadFile(textPath)
	if err != nil {
		return err
	}
	refTextData, err := os.ReadFile(refTextPath)
	if err != nil {
		return err
	}
	r.text = string(textData)
	r.refText = string(refTextData)
	return WritePCM16MonoWAV(outputPath, 16000, pcm16Bytes(0, 900, -900, 1600, -1600))
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

func TestMacOSSayLocalTTSReportIncludesAudioQuality(t *testing.T) {
	dir := t.TempDir()

	report, err := SynthesizeMacOSSay(context.Background(), LocalTTSOptions{
		Text:          "质量分析不要记录原文",
		Voice:         "Tingting",
		OutputDir:     dir,
		CommandRunner: validWAVTTSCommandRunner{},
		SayPath:       "/usr/bin/say",
		AFConvertPath: "/usr/bin/afconvert",
	})

	if err != nil {
		t.Fatal(err)
	}
	if report.AudioQuality == nil {
		t.Fatal("audio quality = nil, want WAV aggregate report")
	}
	if report.AudioQuality.Status != "passed" ||
		report.AudioQuality.Codec != "pcm_s16le" ||
		report.AudioQuality.SampleRateHz != 16000 {
		t.Fatalf("audio quality = %+v", report.AudioQuality)
	}
	rendered := mustJSON(t, report.AudioQuality)
	for _, forbidden := range []string{"质量分析不要记录原文", report.OutputPath, dir, "data_base64", "raw_audio"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("local TTS report leaked %q: %s", forbidden, rendered)
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

func TestVoiceCloneCLILocalTTSSynthesizesRedactedPersonaReport(t *testing.T) {
	dir := t.TempDir()
	refAudio := filepath.Join(t.TempDir(), "a21-reference.wav")
	if err := WritePCM16MonoWAV(refAudio, 16000, pcm16Bytes(0, 300, -300)); err != nil {
		t.Fatal(err)
	}
	runner := &recordingVoiceCloneRunner{}

	report, err := SynthesizeVoiceCloneCLI(context.Background(), LocalTTSOptions{
		Text:                         "用户说的原文不许进报告",
		OutputDir:                    dir,
		CommandRunner:                runner,
		VoiceCloneCommand:            "/a21/bin/a21-index-tts2-wrapper",
		VoiceCloneModel:              "Index-TTS2",
		VoiceCloneReferenceAudioPath: refAudio,
		VoiceCloneReferenceText:      "参考音频文本不许进报告",
		VoiceClonePersona:            "A21 Workmate",
		VoiceCloneStyle:              "Warm-Pro",
	})

	if err != nil {
		t.Fatal(err)
	}
	if report.Provider != "voice_clone_cli" || report.Engine != "voice_clone_cli" || report.Model != "index_tts2" {
		t.Fatalf("voice clone identity = %+v", report)
	}
	if report.VoicePersona != "a21_workmate" || report.StyleProfile != "warm_pro" || report.ReferenceAudio != "a21-reference.wav" {
		t.Fatalf("persona/style/reference = %+v", report)
	}
	if report.Status != "passed" || report.AudioQuality == nil || report.AudioQuality.Status != "passed" {
		t.Fatalf("status/quality = %+v", report)
	}
	if runner.name != "/a21/bin/a21-index-tts2-wrapper" {
		t.Fatalf("runner name = %q", runner.name)
	}
	commandLine := strings.Join(runner.args, " ")
	for _, want := range []string{"--text-file", "--output", "--sample-rate 16000", "--ref-audio " + refAudio, "--ref-text-file", "--model index_tts2", "--persona a21_workmate", "--style warm_pro"} {
		if !strings.Contains(commandLine, want) {
			t.Fatalf("clone command missing %q: %s", want, commandLine)
		}
	}
	rendered := mustJSON(t, report)
	for _, forbidden := range []string{"用户说的原文", "参考音频文本", refAudio, "Authorization", "Bearer", "raw_audio", "data_base64"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("voice clone report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestVoiceCloneCLILocalTTSAllowsRemoteWrapperCommandAndUTF8TempFiles(t *testing.T) {
	dir := t.TempDir()
	refAudio := filepath.Join(t.TempDir(), "a21-reference.wav")
	if err := WritePCM16MonoWAV(refAudio, 16000, pcm16Bytes(0, 300, -300)); err != nil {
		t.Fatal(err)
	}
	refTextPath := filepath.Join(t.TempDir(), "a21-reference.txt")
	refText := "参考文本：你好，A21。"
	if err := os.WriteFile(refTextPath, []byte(refText), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &inspectingVoiceCloneRunner{}

	report, err := SynthesizeVoiceCloneCLI(context.Background(), LocalTTSOptions{
		Text:                         "用户输入：请用轻松语气说这句。",
		OutputDir:                    dir,
		CommandRunner:                runner,
		VoiceCloneCommand:            "/usr/bin/ssh -i /a21-lab/secrets/a21_5080_fixture_ed25519 21@192.168.1.6 powershell -NoProfile -File D:/a21-mainland-latency-lab/outbox/a21-index-tts2-wrapper.ps1",
		VoiceCloneModel:              "Index-TTS2",
		VoiceCloneReferenceAudioPath: refAudio,
		VoiceCloneReferenceTextPath:  refTextPath,
		VoiceClonePersona:            "A21 Workmate",
		VoiceCloneStyle:              "Warm-Pro",
	})

	if err != nil {
		t.Fatal(err)
	}
	if runner.name != "/usr/bin/ssh" {
		t.Fatalf("runner name = %q, want /usr/bin/ssh", runner.name)
	}
	wantPrefix := []string{
		"-i",
		"/a21-lab/secrets/a21_5080_fixture_ed25519",
		"21@192.168.1.6",
		"powershell",
		"-NoProfile",
		"-File",
		"D:/a21-mainland-latency-lab/outbox/a21-index-tts2-wrapper.ps1",
	}
	if len(runner.args) < len(wantPrefix) {
		t.Fatalf("runner args = %#v, want prefix %#v", runner.args, wantPrefix)
	}
	for i, want := range wantPrefix {
		if runner.args[i] != want {
			t.Fatalf("runner arg %d = %q, want %q in %#v", i, runner.args[i], want, runner.args)
		}
	}
	if runner.text != "用户输入：请用轻松语气说这句。" || runner.refText != refText {
		t.Fatalf("wrapper temp files text=%q ref=%q", runner.text, runner.refText)
	}
	if report.Status != "passed" || report.ReferenceAudio != "a21-reference.wav" {
		t.Fatalf("report = %+v", report)
	}
	rendered := mustJSON(t, report)
	for _, forbidden := range []string{
		"用户输入",
		refText,
		refTextPath,
		refAudio,
		dir,
		report.OutputPath,
		"a21_5080_fixture_ed25519",
		"192.168.1.6",
		"Authorization",
		"Bearer",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("voice clone report leaked %q: %s", forbidden, rendered)
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

func argValue(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
