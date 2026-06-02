package app

import (
	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/protocol"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	stackChanSpeakerProbeChunkDurationMS = 20
	stackChanSpeakerProbeBatchChunks     = 8
	stackChanSpeakerPrerollBatchChunks   = 8
	stackChanPlaybackPrebufferBatches    = 1
	stackChanSpeakerProbeMaxChunks       = 64
)

type stackChanLocalTTSPlaybackReport struct {
	SchemaVersion           string               `json:"schema_version"`
	GeneratedAtMS           int64                `json:"generated_at_ms"`
	Metadata                latencyBenchMetadata `json:"metadata"`
	Status                  string               `json:"status"`
	GatewayURL              string               `json:"gateway_url"`
	DeviceID                string               `json:"device_id"`
	TraceID                 string               `json:"trace_id"`
	SessionID               string               `json:"session_id"`
	StreamID                string               `json:"stream_id"`
	InputTextBytes          int                  `json:"input_text_bytes"`
	TTSProvider             string               `json:"tts_provider"`
	TTSEngine               string               `json:"tts_engine,omitempty"`
	TTSModel                string               `json:"tts_model,omitempty"`
	TTSVoice                string               `json:"tts_voice"`
	TTSVoicePersona         string               `json:"tts_voice_persona,omitempty"`
	TTSStyleProfile         string               `json:"tts_style_profile,omitempty"`
	TTSReferenceAudio       string               `json:"tts_reference_audio,omitempty"`
	TTSOutputFormat         string               `json:"tts_output_format"`
	TTSAudioPath            string               `json:"tts_audio_path,omitempty"`
	TTSFirstAudioMS         float64              `json:"tts_first_audio_ms"`
	PlaybackChunks          int                  `json:"playback_chunks"`
	PlaybackBatches         int                  `json:"playback_batches"`
	ExpectedAudioDurationMS int                  `json:"expected_audio_duration_ms"`
	PhysicalSoundObserved   bool                 `json:"physical_sound_observed"`
	ReportPath              string               `json:"report_path,omitempty"`
	Findings                []string             `json:"findings,omitempty"`
}

func runStackChanLocalTTSPlayback(args []string, stdout io.Writer, stderr io.Writer) int {
	inputText := "A21 本地语音实机播放测试。"
	engine := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_TTS_ENGINE"), "sherpa_onnx"))
	voice := strings.TrimSpace(os.Getenv("A21_LOCAL_TTS_VOICE"))
	modelDir := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_MODEL_DIR"))
	speakerID := parsePositiveIntOrDefault(os.Getenv("A21_SHERPA_ONNX_SPEAKER_ID"), 21)
	clone := voiceCloneRuntimeOptionsFromEnv(os.Environ())
	gatewayURL := firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080")
	deviceID := firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-001")
	wavPath := ""
	outputDir := "reports"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-local-tts-playback [--gateway-url http://127.0.0.1:21080] [--device-id stackchan-001] [--engine sherpa_onnx|macos_say|voice_clone_cli] [--text <text>] [--voice Tingting] [--model-dir <dir>] [--speaker-id 21] [--clone-command <path>] [--clone-model index_tts2|cosyvoice3|f5_tts|gpt_sovits] [--clone-ref-audio <wav>] [--clone-ref-text <text>] [--clone-ref-text-file <txt>] [--voice-persona a21_workmate] [--voice-style workmate_warm] [--wav <a21-16k-mono-wav>] [--output-dir reports]")
			return 0
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			gatewayURL = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			deviceID = args[i]
		case "--engine":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--engine requires a value")
				return 2
			}
			i++
			engine = args[i]
		case "--text":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--text requires a value")
				return 2
			}
			i++
			inputText = args[i]
		case "--voice":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--voice requires a value")
				return 2
			}
			i++
			voice = args[i]
		case "--model-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--model-dir requires a value")
				return 2
			}
			i++
			modelDir = args[i]
		case "--speaker-id":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--speaker-id")
			if !ok {
				return 2
			}
			speakerID = value
		case "--clone-command":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--clone-command requires a value")
				return 2
			}
			i++
			clone.Command = args[i]
		case "--clone-model":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--clone-model requires a value")
				return 2
			}
			i++
			clone.Model = args[i]
		case "--clone-ref-audio":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--clone-ref-audio requires a value")
				return 2
			}
			i++
			clone.ReferenceAudioPath = args[i]
		case "--clone-ref-text":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--clone-ref-text requires a value")
				return 2
			}
			i++
			clone.ReferenceText = args[i]
		case "--clone-ref-text-file":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--clone-ref-text-file requires a value")
				return 2
			}
			i++
			clone.ReferenceTextPath = args[i]
		case "--voice-persona":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--voice-persona requires a value")
				return 2
			}
			i++
			clone.Persona = args[i]
		case "--voice-style":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--voice-style requires a value")
				return 2
			}
			i++
			clone.Style = args[i]
		case "--wav":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--wav requires a value")
				return 2
			}
			i++
			wavPath = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-local-tts-playback option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(outputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan local TTS playback report dir invalid: %v\n", err)
		return 1
	}
	report, err := buildStackChanLocalTTSPlaybackReport(context.Background(), localTTSRuntimeOptions{
		Engine:     engine,
		Text:       inputText,
		Voice:      voice,
		ModelDir:   modelDir,
		SpeakerID:  speakerID,
		OutputDir:  outputDir,
		VoiceClone: clone,
	}, gatewayURL, deviceID, wavPath)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan local TTS playback failed: %v\n", err)
		return 1
	}
	reportPath, err := writeStackChanLocalTTSPlaybackReport(outputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan local TTS playback report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanLocalTTSPlayback(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan local TTS playback report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}
func buildStackChanLocalTTSPlaybackReport(ctx context.Context, ttsOptions localTTSRuntimeOptions, gatewayURL string, deviceID string, wavPath string) (stackChanLocalTTSPlaybackReport, error) {
	generatedAtMS := time.Now().UnixMilli()
	traceID := fmt.Sprintf("a21-trace-local-tts-playback-%d", generatedAtMS)
	sessionID := fmt.Sprintf("a21-session-local-tts-playback-%d", generatedAtMS)
	streamID := fmt.Sprintf("a21-local-tts-playback-stream-%d", generatedAtMS)
	report := stackChanLocalTTSPlaybackReport{
		SchemaVersion:         "a21.stackchan_local_tts_playback.v1",
		GeneratedAtMS:         generatedAtMS,
		Metadata:              buildLatencyBenchMetadata(),
		Status:                "failed",
		GatewayURL:            productSurfaceLabel(gatewayURL, "/v1/devices/control"),
		DeviceID:              deviceID,
		TraceID:               traceID,
		SessionID:             sessionID,
		StreamID:              streamID,
		InputTextBytes:        len([]byte(ttsOptions.Text)),
		PhysicalSoundObserved: false,
	}
	if _, _, err := firmwareGatewayEndpoint(gatewayURL, "/v1/devices/control", nil); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, nil
	}
	playbackWAVPath := strings.TrimSpace(wavPath)
	if playbackWAVPath == "" {
		ttsReport, err := synthesizeLocalTTS(ctx, ttsOptions)
		if err != nil {
			report.Findings = append(report.Findings, "local TTS failed")
			return report, err
		}
		report.TTSProvider = ttsReport.Provider
		report.TTSEngine = ttsReport.Engine
		report.TTSModel = ttsReport.Model
		report.TTSVoice = ttsReport.Voice
		report.TTSVoicePersona = ttsReport.VoicePersona
		report.TTSStyleProfile = ttsReport.StyleProfile
		report.TTSReferenceAudio = ttsReport.ReferenceAudio
		report.TTSOutputFormat = ttsReport.OutputFormat
		report.TTSAudioPath = filepath.Base(ttsReport.OutputPath)
		report.TTSFirstAudioMS = ttsReport.TTSFirstAudioMS
		if ttsReport.Status != "passed" {
			report.Findings = append(report.Findings, "local TTS did not pass")
			return report, nil
		}
		playbackWAVPath = ttsReport.OutputPath
	} else {
		if containsLegacyIdentityPathToken(playbackWAVPath) {
			report.Findings = append(report.Findings, "playback WAV path contains forbidden legacy project identity")
			return report, fmt.Errorf("playback WAV path contains forbidden legacy project identity")
		}
		report.TTSProvider = "wav_file"
		report.TTSVoice = "diagnostic_wav"
		report.TTSOutputFormat = "wav_pcm_s16le_16000_mono"
		report.TTSAudioPath = filepath.Base(playbackWAVPath)
	}
	pcmChunks, err := audio.ReadPCM16MonoWAVChunks(playbackWAVPath, 20)
	if err != nil {
		report.Findings = append(report.Findings, "local TTS wav parse failed")
		return report, err
	}
	report.PlaybackChunks = len(pcmChunks)
	report.ExpectedAudioDurationMS = len(pcmChunks) * 20
	for offset := 0; offset < len(pcmChunks); {
		batchChunks := stackChanPlaybackBatchSize(offset, len(pcmChunks))
		end := offset + batchChunks
		if end > len(pcmChunks) {
			end = len(pcmChunks)
		}
		batch := make([]protocol.AudioPlaybackChunk, 0, end-offset)
		for _, chunk := range pcmChunks[offset:end] {
			batch = append(batch, protocol.AudioPlaybackChunk{
				StreamID:     streamID,
				Codec:        protocol.AudioCodecPCMS16LE,
				SampleRateHz: chunk.SampleRateHz,
				Channels:     chunk.Channels,
				DurationMS:   chunk.DurationMS,
				DataBase64:   chunk.DataBase64,
			})
		}
		batch = padStackChanPlaybackBatch(streamID, batch)
		if _, err := postStackChanAudioPlaybackBatch(gatewayURL, deviceID, traceID, sessionID, streamID, batch); err != nil {
			report.Findings = append(report.Findings, "device playback delivery failed")
			return report, err
		}
		report.PlaybackBatches++
		time.Sleep(stackChanPlaybackBatchDelay(offset, end, len(pcmChunks), len(batch)))
		offset = end
	}
	if _, err := postStackChanSpeakerControl(gatewayURL, deviceID, protocol.ExpressionIdle, protocol.ModeWorkmate, "IDLE", traceID, sessionID, streamID, 0); err != nil {
		report.Findings = append(report.Findings, "device playback idle delivery failed")
		return report, err
	}
	report.Status = "passed"
	return report, nil
}
func writeStackChanLocalTTSPlaybackReport(outputDir string, report stackChanLocalTTSPlaybackReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-local-tts-playback-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanLocalTTSPlayback(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONStackChanLocalTTSPlayback(writer io.Writer, report stackChanLocalTTSPlaybackReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
