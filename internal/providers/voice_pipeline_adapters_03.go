package providers

import (
	"os"
	"path/filepath"
	"strings"
)

func defaultSherpaStreamingASRPythonPath() string {
	candidate := filepath.Join(".a21-tools", "sherpa-onnx-venv", "bin", "python")
	if pipelinePathExists(candidate) {
		return candidate
	}
	return ""
}

func defaultSherpaStreamingASRModelDir() string {
	candidate := filepath.Join(".a21-tools", "sherpa-onnx-asr-models", "sherpa-onnx-streaming-zipformer-zh-int8-2025-06-30")
	if pipelineDirExists(candidate) {
		return candidate
	}
	return ""
}

func pipelinePathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func pipelineDirExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
