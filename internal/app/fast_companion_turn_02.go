package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/protocol"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func deliverStackChanAudioFile(ctx context.Context, gatewayURL string, deviceID string, traceID string, sessionID string, streamID string, wavPath string) (stackChanAudioDelivery, error) {
	if strings.TrimSpace(wavPath) == "" {
		return stackChanAudioDelivery{}, fmt.Errorf("audio path is required")
	}
	pcmChunks, err := audio.ReadPCM16MonoWAVChunks(wavPath, 20)
	if err != nil {
		return stackChanAudioDelivery{}, err
	}
	delivery := stackChanAudioDelivery{Chunks: len(pcmChunks)}
	for offset := 0; offset < len(pcmChunks); {
		batchChunks := stackChanPlaybackBatchSize(offset, len(pcmChunks))
		end := int(math.Min(float64(offset+batchChunks), float64(len(pcmChunks))))
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
		response, err := postStackChanAudioPlaybackBatch(gatewayURL, deviceID, traceID, sessionID, streamID, batch)
		if err != nil {
			return delivery, err
		}
		delivery.Batches++
		delivery.DeliveredTransport = firstNonEmpty(response.DeliveredTransport, delivery.DeliveredTransport)
		select {
		case <-ctx.Done():
			return delivery, ctx.Err()
		case <-time.After(stackChanPlaybackBatchDelay(offset, end, len(pcmChunks), len(batch))):
		}
		offset = end
	}
	return delivery, nil
}

func stackChanPlaybackBatchSize(offset int, totalChunks int) int {
	if offset == 0 && totalChunks > stackChanSpeakerProbeBatchChunks {
		if totalChunks < stackChanSpeakerPrerollBatchChunks {
			return totalChunks
		}
		return stackChanSpeakerPrerollBatchChunks
	}
	remaining := totalChunks - offset
	if remaining < stackChanSpeakerProbeBatchChunks {
		return remaining
	}
	return stackChanSpeakerProbeBatchChunks
}

func stackChanPlaybackBatchDelay(offset int, nextOffset int, totalChunks int, transportBatchChunks int) time.Duration {
	if transportBatchChunks <= 0 {
		return 0
	}
	if nextOffset < totalChunks {
		batchIndex := offset / stackChanSpeakerProbeBatchChunks
		if batchIndex < stackChanPlaybackPrebufferBatches-1 {
			return 0
		}
	}
	return time.Duration(transportBatchChunks*stackChanSpeakerProbeChunkDurationMS) * time.Millisecond
}

func padStackChanPlaybackBatch(streamID string, batch []protocol.AudioPlaybackChunk) []protocol.AudioPlaybackChunk {
	if len(batch) == 0 || len(batch) >= stackChanSpeakerProbeBatchChunks {
		return batch
	}
	sampleRateHz := batch[len(batch)-1].SampleRateHz
	channels := batch[len(batch)-1].Channels
	durationMS := batch[len(batch)-1].DurationMS
	silence := stackChanPCM16SilenceBase64(sampleRateHz, durationMS)
	for len(batch) < stackChanSpeakerProbeBatchChunks {
		batch = append(batch, protocol.AudioPlaybackChunk{
			StreamID:     streamID,
			Codec:        protocol.AudioCodecPCMS16LE,
			SampleRateHz: sampleRateHz,
			Channels:     channels,
			DurationMS:   durationMS,
			DataBase64:   silence,
		})
	}
	return batch
}

func stackChanPCM16SilenceBase64(sampleRateHz int, durationMS int) string {
	if sampleRateHz <= 0 || durationMS <= 0 {
		return ""
	}
	byteCount := sampleRateHz * durationMS * 2 / 1000
	return base64.StdEncoding.EncodeToString(make([]byte, byteCount))
}

func measureStackChanFastCompanionBargeIn(ctx context.Context, gatewayURL string, deviceID string, traceID string, sessionID string, seq uint64) (float64, error) {
	bargeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(bargeCtx, latencyBenchWebSocketURL(gatewayURL, "/ws/audio"), nil)
	if err != nil {
		return stackChanFastCompanionTraceBargeInStopMS(gatewayURL, traceID, err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "a21 fast companion barge-in done")

	started := time.Now()
	if err := writeStackChanFastCompanionAudioFrame(bargeCtx, conn, deviceID, seq, traceID, sessionID, latencyBenchPCM16Base64(12000)); err != nil {
		return 0, err
	}
	var interrupted protocol.Envelope
	if err := wsjson.Read(bargeCtx, conn, &interrupted); err != nil {
		return 0, err
	}
	if interrupted.Kind != protocol.KindControlEvent {
		return 0, fmt.Errorf("fast companion barge-in event kind %q, want %q", interrupted.Kind, protocol.KindControlEvent)
	}
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(interrupted.Payload, &payload); err != nil {
		return 0, err
	}
	if payload.State != protocol.ExpressionInterrupted {
		return 0, fmt.Errorf("fast companion barge-in state %q, want %q", payload.State, protocol.ExpressionInterrupted)
	}
	trace, err := fetchGatewayTrace(gatewayURL, traceID)
	if err == nil && trace.Summary.BargeInStopMS != nil {
		return float64(*trace.Summary.BargeInStopMS), nil
	}
	return elapsedReportMS(started), nil
}

func stackChanFastCompanionTraceBargeInStopMS(gatewayURL string, traceID string, cause error) (float64, error) {
	trace, err := fetchGatewayTrace(gatewayURL, traceID)
	if err != nil {
		return 0, cause
	}
	if trace.Summary.BargeInStopMS == nil {
		return 0, fmt.Errorf("fast companion trace missing barge-in stop summary")
	}
	return float64(*trace.Summary.BargeInStopMS), nil
}

func writeStackChanFastCompanionAudioFrame(ctx context.Context, conn *websocket.Conn, deviceID string, seq uint64, traceID string, sessionID string, dataBase64 string) error {
	payload, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   dataBase64,
	})
	if err != nil {
		return err
	}
	return wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  deviceID,
		Kind:      protocol.KindAudioFrame,
		Seq:       seq,
		TraceID:   traceID,
		SessionID: sessionID,
		Payload:   payload,
	})
}

func normalizeFastCompanionListenSource(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "stackchan_mic":
		return "stackchan_mic"
	default:
		return "host_fixture"
	}
}

func writeStackChanFastCompanionTurnReport(outputDir string, report stackChanFastCompanionTurnReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-fast-companion-turn-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanFastCompanionTurn(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeJSONStackChanFastCompanionTurn(writer io.Writer, report stackChanFastCompanionTurnReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
