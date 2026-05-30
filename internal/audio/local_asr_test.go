package audio

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSherpaONNXASRSmokeParsesScriptOutputAndRedactsPaths(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(dir, "a21-asr-model")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"model.int8.onnx", "tokens.txt"} {
		if err := os.WriteFile(filepath.Join(modelDir, name), []byte("a21"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	wavDir := filepath.Join(modelDir, "test_wavs")
	if err := os.MkdirAll(wavDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wavPath := filepath.Join(wavDir, "private.wav")
	if err := os.WriteFile(wavPath, []byte("RIFF-a21"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakePython := filepath.Join(dir, "a21-fake-python")
	if err := os.WriteFile(fakePython, []byte("#!/bin/sh\nprintf '%s\\n' '{\"status\":\"passed\",\"input_duration_ms\":1200,\"decode_duration_ms\":90,\"real_time_factor\":0.075,\"text_chars\":8}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := RunSherpaONNXASRSmoke(context.Background(), LocalASROptions{
		OutputDir:  dir,
		PythonPath: fakePython,
		ScriptPath: filepath.Join(dir, "unused-script.py"),
		ModelDir:   modelDir,
		Family:     "paraformer",
		WAVPath:    wavPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	if report.Status != "passed" || report.Provider != "sherpa_onnx" || report.Engine != "paraformer" {
		t.Fatalf("report status/provider/engine = %+v", report)
	}
	if report.ModelDir != "a21-asr-model" || report.WAVName != "private.wav" {
		t.Fatalf("report leaked non-basename path data: %+v", report)
	}
	if report.TextChars != 8 || report.TranscriptPolicy != "transcript_not_recorded" {
		t.Fatalf("report transcript fields = %+v", report)
	}
	if strings.Contains(report.ModelDir, dir) || strings.Contains(report.WAVName, dir) {
		t.Fatalf("report leaked temp path: %+v", report)
	}
}

func TestValidateSherpaONNXASRModelDirRequiresStreamingFiles(t *testing.T) {
	modelDir := t.TempDir()
	for _, name := range []string{"encoder.int8.onnx", "decoder.onnx", "tokens.txt"} {
		if err := os.WriteFile(filepath.Join(modelDir, name), []byte("a21"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := validateSherpaONNXASRModelDir(modelDir, "streaming_zipformer")

	if err == nil || !strings.Contains(err.Error(), "joiner.int8.onnx") {
		t.Fatalf("error = %v, want missing joiner", err)
	}
}
