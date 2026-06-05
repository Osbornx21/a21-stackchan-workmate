package app

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/audio/opuscodec"
)

func xiaozhiVoiceBenchOpusPackets(options xiaozhiVoiceBenchOptions) ([][]byte, xiaozhiVoiceBenchInput, error) {
	if strings.TrimSpace(options.InputWAV) != "" {
		packets, err := xiaozhiVoiceBenchOpusPacketsFromWAV(options.InputWAV)
		if err != nil {
			return nil, xiaozhiVoiceBenchInput{}, err
		}
		return packets, xiaozhiVoiceBenchInput{
			Source:         "wav_fixture",
			WAVName:        filepath.Base(filepath.Clean(options.InputWAV)),
			OpusFrameCount: len(packets),
		}, nil
	}
	packet, err := xiaozhiVoiceBenchSyntheticOpusPacket()
	if err != nil {
		return nil, xiaozhiVoiceBenchInput{}, err
	}
	return [][]byte{packet}, xiaozhiVoiceBenchInput{
		Source:         "synthetic_sine",
		OpusFrameCount: 1,
	}, nil
}

func xiaozhiVoiceBenchOpusPacketsFromWAV(path string) ([][]byte, error) {
	chunks, err := audio.ReadPCM16MonoWAVChunks(path, 60)
	if err != nil {
		return nil, err
	}
	codec, err := opuscodec.New(16000, 1, 60)
	if err != nil {
		return nil, err
	}
	packets := make([][]byte, 0, len(chunks))
	for _, chunk := range chunks {
		data, err := base64.StdEncoding.DecodeString(chunk.DataBase64)
		if err != nil {
			return nil, err
		}
		if len(data)%2 != 0 {
			return nil, fmt.Errorf("wav pcm chunk must contain 16-bit samples")
		}
		pcm := make([]int16, len(data)/2)
		for i := range pcm {
			pcm[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
		}
		packet, err := codec.EncodePCM16(pcm)
		if err != nil {
			return nil, err
		}
		packets = append(packets, packet)
	}
	if len(packets) == 0 {
		return nil, fmt.Errorf("wav fixture produced no Opus frames")
	}
	return packets, nil
}

func xiaozhiVoiceBenchSyntheticOpusPacket() ([]byte, error) {
	codec, err := opuscodec.New(16000, 1, 60)
	if err != nil {
		return nil, err
	}
	pcm := make([]int16, codec.FrameSamples())
	for i := range pcm {
		pcm[i] = int16(math.Sin(2*math.Pi*440*float64(i)/16000) * 12000)
	}
	return codec.EncodePCM16(pcm)
}

func wrapXiaozhiVoiceBenchOpus(packet []byte, protocolVersion int) []byte {
	switch protocolVersion {
	case 2:
		wrapped := make([]byte, 16+len(packet))
		binary.BigEndian.PutUint16(wrapped[0:2], 2)
		binary.BigEndian.PutUint32(wrapped[8:12], uint32(time.Now().UnixMilli()))
		binary.BigEndian.PutUint32(wrapped[12:16], uint32(len(packet)))
		copy(wrapped[16:], packet)
		return wrapped
	case 3:
		wrapped := make([]byte, 4+len(packet))
		binary.BigEndian.PutUint16(wrapped[2:4], uint16(len(packet)))
		copy(wrapped[4:], packet)
		return wrapped
	default:
		return append([]byte(nil), packet...)
	}
}

func summarizeXiaozhiVoiceBench(answerTurns []xiaozhiVoiceBenchTurn, bargeInTurns []xiaozhiVoiceBenchTurn) xiaozhiVoiceBenchSummary {
	answerDurations := make([]time.Duration, 0, len(answerTurns))
	for _, turn := range answerTurns {
		if turn.FirstAudioMS != nil {
			answerDurations = append(answerDurations, time.Duration(*turn.FirstAudioMS)*time.Millisecond)
		}
	}
	bargeDurations := make([]time.Duration, 0, len(bargeInTurns))
	for _, turn := range bargeInTurns {
		if turn.AbortStopMS != nil {
			bargeDurations = append(bargeDurations, time.Duration(*turn.AbortStopMS)*time.Millisecond)
		}
	}
	return xiaozhiVoiceBenchSummary{
		AnswerFirstAudioP50MS: percentileMS(answerDurations, 0.50),
		AnswerFirstAudioP95MS: percentileMS(answerDurations, 0.95),
		BargeInStopP50MS:      percentileMS(bargeDurations, 0.50),
		BargeInStopP95MS:      percentileMS(bargeDurations, 0.95),
	}
}

func xiaozhiVoiceBenchWebSocketFailureFinding(resp *http.Response, err error) string {
	const base = "gateway_websocket_unavailable"
	if resp != nil && resp.StatusCode > 0 {
		return fmt.Sprintf("%s_http_%d", base, resp.StatusCode)
	}
	if err == nil {
		return base
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "proxy") || strings.Contains(message, "tunnel"):
		return base + "_proxy_or_tunnel"
	case strings.Contains(message, "connection refused"):
		return base + "_connection_refused"
	case strings.Contains(message, "operation not permitted"):
		return base + "_operation_not_permitted"
	case strings.Contains(message, "deadline exceeded") || strings.Contains(message, "i/o timeout"):
		return base + "_timeout"
	case strings.Contains(message, "no such host") || strings.Contains(message, "lookup"):
		return base + "_dns"
	case strings.Contains(message, "expected handshake response status code"):
		return base + "_bad_status"
	case strings.Contains(message, "sec-websocket"):
		return base + "_bad_handshake"
	default:
		return base + "_dial_error"
	}
}

func xiaozhiVoiceBenchFailureCount(turns []xiaozhiVoiceBenchTurn) int {
	failures := 0
	for _, turn := range turns {
		if turn.Status != "passed" || !turn.HelloAccepted || !turn.ListenAck || turn.BinaryDownlinkFrames <= 0 || !turn.MetricsObserved || len(turn.Findings) > 0 {
			failures++
			continue
		}
		if turn.Kind == "answer" && turn.FirstAudioMS == nil {
			failures++
		}
		if turn.Kind == "barge_in" && turn.AbortStopMS == nil {
			failures++
		}
	}
	return failures
}

func xiaozhiVoiceBenchHasAnswerSamples(turns []xiaozhiVoiceBenchTurn) bool {
	for _, turn := range turns {
		if turn.Kind == "answer" && turn.FirstAudioMS != nil {
			return true
		}
	}
	return false
}

func xiaozhiVoiceBenchHasBargeInSamples(turns []xiaozhiVoiceBenchTurn) bool {
	for _, turn := range turns {
		if turn.Kind == "barge_in" && turn.AbortStopMS != nil {
			return true
		}
	}
	return false
}

func attachXiaozhiVoiceBenchTraceSummary(ctx context.Context, gatewayURL string, receipt *xiaozhiVoiceBenchTurn) {
	if receipt == nil || receipt.TraceID == "" {
		return
	}
	summary, err := fetchXiaozhiVoiceBenchTraceSummary(ctx, gatewayURL, receipt.TraceID)
	if err != nil {
		receipt.Findings = append(receipt.Findings, "trace_summary_unavailable")
		return
	}
	receipt.TraceSummary = &summary
	if receipt.Kind == "barge_in" {
		if summary.BargeInStopMS == nil && receipt.AbortStopMS != nil {
			summary.BargeInStopMS = receipt.AbortStopMS
			receipt.TraceSummary = &summary
		}
		if !xiaozhiVoiceBenchBargeInTraceMetricsPresent(summary) {
			receipt.Findings = append(receipt.Findings, "trace_barge_in_metrics_missing")
		}
		return
	}
	if !xiaozhiVoiceBenchCoreTraceMetricsPresent(summary) {
		receipt.Findings = append(receipt.Findings, "trace_core_stage_metrics_missing")
	}
}

func xiaozhiVoiceBenchBargeInTraceMetricsPresent(summary xiaozhiVoiceBenchTraceSummary) bool {
	return summary.BargeInStopMS != nil &&
		summary.AnswerFirstAudioTotalMS != nil
}

func xiaozhiVoiceBenchCoreTraceMetricsPresent(summary xiaozhiVoiceBenchTraceSummary) bool {
	return summary.ASRFirstPartialMS != nil &&
		summary.ASRFinalMS != nil &&
		summary.LLMFirstContentMS != nil &&
		summary.TTSFirstAudioMS != nil &&
		summary.AudioDownlinkFirstFrameMS != nil &&
		summary.AnswerFirstAudioTotalMS != nil
}

func fetchXiaozhiVoiceBenchTraceSummary(ctx context.Context, gatewayURL string, traceID string) (xiaozhiVoiceBenchTraceSummary, error) {
	endpoint, err := xiaozhiVoiceBenchTraceURL(gatewayURL, traceID)
	if err != nil {
		return xiaozhiVoiceBenchTraceSummary{}, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return xiaozhiVoiceBenchTraceSummary{}, err
	}
	client := *a21DirectHTTPClient(2 * time.Second)
	response, err := client.Do(request)
	if err != nil {
		return xiaozhiVoiceBenchTraceSummary{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return xiaozhiVoiceBenchTraceSummary{}, fmt.Errorf("trace summary unavailable")
	}
	var decoded struct {
		Events  []xiaozhiVoiceBenchTraceEvent `json:"events"`
		Summary xiaozhiVoiceBenchTraceSummary `json:"summary"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return xiaozhiVoiceBenchTraceSummary{}, err
	}
	if decoded.Summary.EventCount <= 0 {
		return xiaozhiVoiceBenchTraceSummary{}, fmt.Errorf("trace summary empty")
	}
	if decoded.Summary.ASRFinalMS == nil {
		decoded.Summary.ASRFinalMS = xiaozhiVoiceBenchTraceDeltaMS(decoded.Events, "audio.ingress.buffered", "asr.final")
	}
	return decoded.Summary, nil
}

func xiaozhiVoiceBenchTraceDeltaMS(events []xiaozhiVoiceBenchTraceEvent, startName string, endName string) *int64 {
	var startAtMS int64
	hasStart := false
	for _, event := range events {
		switch {
		case event.Name == startName && !hasStart:
			startAtMS = event.AtMS
			hasStart = true
		case event.Name == endName && hasStart:
			delta := event.AtMS - startAtMS
			if delta < 0 {
				delta = 0
			}
			return &delta
		}
	}
	return nil
}

func xiaozhiVoiceBenchTraceURL(gatewayURL string, traceID string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(gatewayURL))
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid gateway URL")
	}
	parsed.Path = "/v1/traces"
	query := url.Values{}
	query.Set("trace_id", traceID)
	parsed.RawQuery = query.Encode()
	parsed.Fragment = ""
	return parsed.String(), nil
}

func xiaozhiVoiceBenchWebSocketURL(gatewayURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(gatewayURL))
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid gateway URL")
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported gateway URL scheme")
	}
	parsed.Path = "/v1/xiaozhi"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func xiaozhiVoiceBenchGatewayLabel(gatewayURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(gatewayURL))
	if err != nil || parsed.Host == "" {
		return "invalid_gateway"
	}
	scope := "remote"
	switch parsed.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		scope = "loopback"
	}
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return scope + ":" + port + "/v1/xiaozhi"
}

func xiaozhiVoiceBenchContainsLegacy(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(lower, "x21") || strings.Contains(lower, "v21")
}

func writeXiaozhiVoiceBenchReport(outputDir string, report xiaozhiVoiceBenchReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-xiaozhi-voice-bench-"+time.Now().Format("20060102-150405.000000000")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONXiaozhiVoiceBench(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONXiaozhiVoiceBench(writer io.Writer, report xiaozhiVoiceBenchReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
