package audio

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

var ErrSileroVADRunnerUnavailable = errors.New("a21 silero vad runner unavailable")

type CommandSileroVADRunnerConfig struct {
	CommandPath string
	ModelPath   string
	Args        []string
}

type CommandSileroVADRunner struct {
	config CommandSileroVADRunnerConfig
}

func NewCommandSileroVADRunner(config CommandSileroVADRunnerConfig) CommandSileroVADRunner {
	return CommandSileroVADRunner{config: config}
}

func (r CommandSileroVADRunner) Detect(ctx context.Context, frame Frame) (VADDecision, error) {
	commandPath := strings.TrimSpace(r.config.CommandPath)
	if commandPath == "" {
		return VADDecision{}, ErrSileroVADRunnerUnavailable
	}
	pcm, err := base64.StdEncoding.DecodeString(frame.DataBase64)
	if err != nil {
		return VADDecision{}, ErrSileroVADRunnerUnavailable
	}

	args := append([]string{}, r.config.Args...)
	args = append(args,
		"--sample-rate", strconv.Itoa(frame.SampleRateHz),
		"--channels", strconv.Itoa(frame.Channels),
		"--duration-ms", strconv.Itoa(frame.DurationMS),
	)
	if modelPath := strings.TrimSpace(r.config.ModelPath); modelPath != "" {
		args = append(args, "--model", modelPath)
	}

	cmd := exec.CommandContext(ctx, commandPath, args...)
	cmd.Stdin = bytes.NewReader(pcm)
	stdout, err := cmd.Output()
	if ctx != nil && ctx.Err() != nil {
		return VADDecision{}, ctx.Err()
	}
	if err != nil {
		return VADDecision{}, ErrSileroVADRunnerUnavailable
	}

	var payload struct {
		SpeechDetected bool    `json:"speech_detected"`
		Score          float64 `json:"score"`
	}
	if err := json.Unmarshal(stdout, &payload); err != nil {
		return VADDecision{}, ErrSileroVADRunnerUnavailable
	}
	return VADDecision{
		SpeechDetected: payload.SpeechDetected,
		Score:          payload.Score,
	}, nil
}
