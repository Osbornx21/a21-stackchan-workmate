package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunLocalASRStreamingSmokeWritesRedactedPassReport(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(t.TempDir(), "a21-streaming-model")
	createStreamingASRModelFiles(t, modelDir)
	logPath := filepath.Join(t.TempDir(), "helper-commands.jsonl")
	helperPath := writeStreamingASRPassHelper(t, logPath)
	t.Setenv("A21_FAKE_STREAMING_HELPER_LOG", logPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"local-asr-streaming-smoke",
		"--helper", helperPath,
		"--model-dir", modelDir,
		"--family", "streaming_zipformer",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.audio.local_asr_streaming_smoke.v1"`,
		`"status": "passed"`,
		`"provider": "sherpa_onnx_streaming"`,
		`"family": "streaming_zipformer"`,
		`"model_dir": "a21-streaming-model"`,
		`"model_files_present": true`,
		`"frames_appended": 1`,
		`"final_events": 1`,
		`"transcript_policy": "transcript_not_recorded"`,
		`"audio_payload_policy": "raw_audio_not_recorded"`,
		`"report_path"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-asr-streaming-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	rendered := stdout.String() + string(reportData)
	for _, forbidden := range []string{
		"private partial",
		"private final",
		modelDir,
		helperPath,
		"pcm16le_b64",
		"audio_base64",
		"data_base64",
		"audio_data",
		".wav",
		"Authorization",
		"Bearer",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("streaming ASR smoke leaked %q: %s", forbidden, rendered)
		}
	}
	commands := eventuallyReadAppTestFile(t, logPath)
	startIndex := strings.Index(commands, `"type":"start"`)
	appendIndex := strings.Index(commands, `"type":"append"`)
	commitIndex := strings.Index(commands, `"type":"commit"`)
	if startIndex < 0 || appendIndex < 0 || commitIndex < 0 || !(startIndex < appendIndex && appendIndex < commitIndex) {
		t.Fatalf("helper commands out of order:\n%s", commands)
	}
	if strings.Contains(commands, ".wav") {
		t.Fatalf("streaming ASR helper command used WAV boundary:\n%s", commands)
	}
}

func TestRunLocalASRStreamingSmokeRecordsMissingModelBlocker(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "helper-commands.jsonl")
	helperPath := writeStreamingASRPassHelper(t, logPath)
	t.Setenv("A21_FAKE_STREAMING_HELPER_LOG", logPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"local-asr-streaming-smoke",
		"--helper", helperPath,
		"--model-dir", filepath.Join(t.TempDir(), "missing-model"),
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"status": "blocked"`,
		`"model_dir": "missing-model"`,
		`"model_files_present": false`,
		`"model_dir_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("helper was executed for missing model blocker, stat err=%v", err)
	}
}

func TestRunLocalASRStreamingSmokeRecordsSherpaPackageBlocker(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(t.TempDir(), "a21-streaming-model")
	createStreamingASRModelFiles(t, modelDir)
	logPath := filepath.Join(t.TempDir(), "helper-commands.jsonl")
	helperPath := writeStreamingASRErrorHelper(t, logPath, "sherpa_onnx_unavailable")
	t.Setenv("A21_FAKE_STREAMING_HELPER_LOG", logPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"local-asr-streaming-smoke",
		"--helper", helperPath,
		"--model-dir", modelDir,
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"status": "blocked"`,
		`"sherpa_onnx_unavailable"`,
		`"model_files_present": true`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{modelDir, helperPath, "private final", "pcm16le_b64"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("streaming ASR blocker leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func createStreamingASRModelFiles(t *testing.T, modelDir string) {
	t.Helper()
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"encoder.int8.onnx", "decoder.onnx", "joiner.int8.onnx", "tokens.txt"} {
		if err := os.WriteFile(filepath.Join(modelDir, name), []byte("a21"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func writeStreamingASRPassHelper(t *testing.T, logPath string) string {
	t.Helper()
	return writeStreamingASRHelper(t, `#!/bin/sh
while IFS= read line; do
  printf '%s\n' "$line" >> "$A21_FAKE_STREAMING_HELPER_LOG"
  case "$line" in
    *'"type":"start"'*) printf '{"type":"ready"}\n' ;;
    *'"type":"append"'*) printf '{"type":"partial","text":"private partial"}\n' ;;
    *'"type":"commit"'*) printf '{"type":"final","text":"private final"}\n'; exit 0 ;;
  esac
done
`, logPath)
}

func writeStreamingASRErrorHelper(t *testing.T, logPath string, code string) string {
	t.Helper()
	return writeStreamingASRHelper(t, `#!/bin/sh
while IFS= read line; do
  printf '%s\n' "$line" >> "$A21_FAKE_STREAMING_HELPER_LOG"
  printf '{"type":"error","code":"`+code+`"}\n'
  exit 1
done
`, logPath)
}

func writeStreamingASRHelper(t *testing.T, script string, logPath string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a21-fake-streaming-helper")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func eventuallyReadAppTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
