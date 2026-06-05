package audio

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
)

type LocalTTSOptions struct {
	Text                         string
	Voice                        string
	OutputDir                    string
	CommandRunner                CommandRunner
	SayPath                      string
	AFConvertPath                string
	PythonPath                   string
	ScriptPath                   string
	ModelDir                     string
	SpeakerID                    int
	OutputSampleRateHz           int
	VoiceCloneCommand            string
	VoiceCloneModel              string
	VoiceCloneReferenceAudioPath string
	VoiceCloneReferenceText      string
	VoiceCloneReferenceTextPath  string
	VoiceClonePersona            string
	VoiceCloneStyle              string
}

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
}

type execCommandRunner struct{}

type LocalTTSReport struct {
	SchemaVersion   string            `json:"schema_version"`
	GeneratedAtMS   int64             `json:"generated_at_ms"`
	Status          string            `json:"status"`
	Provider        string            `json:"provider"`
	Engine          string            `json:"engine,omitempty"`
	EndpointHost    string            `json:"endpoint_host,omitempty"`
	NetworkMode     string            `json:"network_mode,omitempty"`
	Voice           string            `json:"voice"`
	Model           string            `json:"model,omitempty"`
	VoicePersona    string            `json:"voice_persona,omitempty"`
	StyleProfile    string            `json:"style_profile,omitempty"`
	ReferenceAudio  string            `json:"reference_audio,omitempty"`
	ModelDir        string            `json:"model_dir,omitempty"`
	OutputFormat    string            `json:"output_format"`
	OutputPath      string            `json:"output_path,omitempty"`
	ReportPath      string            `json:"report_path,omitempty"`
	OutputBytes     int64             `json:"output_bytes,omitempty"`
	TextBytes       int               `json:"text_bytes,omitempty"`
	DurationMS      float64           `json:"duration_ms,omitempty"`
	TTSFirstAudioMS float64           `json:"tts_first_audio_ms,omitempty"`
	AudioQuality    *PCMQualityReport `json:"audio_quality,omitempty"`
	Findings        []string          `json:"findings,omitempty"`
}

func (report LocalTTSReport) MarshalJSON() ([]byte, error) {
	type localTTSReportJSON LocalTTSReport
	redacted := localTTSReportJSON(report)
	redacted.ReferenceAudio = localTTSBaseName(report.ReferenceAudio)
	redacted.ModelDir = localTTSBaseName(report.ModelDir)
	redacted.OutputPath = localTTSBaseName(report.OutputPath)
	redacted.ReportPath = localTTSBaseName(report.ReportPath)
	redacted.EndpointHost = localTTSEndpointHostLabel(report.EndpointHost)
	return json.Marshal(redacted)
}

func (execCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

const (
	iflytekTTSEngine          = "iflytek_tts"
	iflytekTTSDefaultEndpoint = "wss://tts-api.xfyun.cn/v2/tts"
	iflytekTTSDefaultVoice    = "xiaoyan"
	iflytekTTSSampleRateHz    = 16000
)

type iflytekTTSConfig struct {
	AppID     string
	APIKey    string
	APISecret string
	Endpoint  string
	Voice     string
}

type iflytekTTSWebSocketConn interface {
	Write(ctx context.Context, typ websocket.MessageType, p []byte) error
	Read(ctx context.Context) (websocket.MessageType, []byte, error)
	Close(code websocket.StatusCode, reason string) error
}

var iflytekTTSWebSocketDial = func(ctx context.Context, rawURL string) (iflytekTTSWebSocketConn, int, error) {
	conn, response, err := websocket.Dial(ctx, rawURL, &websocket.DialOptions{HTTPClient: iflytekTTSWebSocketHTTPClient()})
	statusCode := 0
	if response != nil {
		statusCode = response.StatusCode
	}
	return conn, statusCode, err
}

var iflytekTTSWebSocketHTTPClient = func() *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{Timeout: 15 * time.Second}
	}
	direct := transport.Clone()
	direct.Proxy = nil
	return &http.Client{Transport: direct, Timeout: 15 * time.Second}
}

var iflytekTTSEndpoint = func() string {
	return iflytekTTSDefaultEndpoint
}

type iflytekTTSRequest struct {
	Common struct {
		AppID string `json:"app_id"`
	} `json:"common"`
	Business struct {
		AUE    string `json:"aue"`
		AUF    string `json:"auf"`
		VCN    string `json:"vcn"`
		TTE    string `json:"tte"`
		Speed  int    `json:"speed"`
		Volume int    `json:"volume"`
		Pitch  int    `json:"pitch"`
	} `json:"business"`
	Data struct {
		Status int    `json:"status"`
		Text   string `json:"text"`
	} `json:"data"`
}

type iflytekTTSResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	SID     string `json:"sid"`
	Data    *struct {
		Audio  string `json:"audio"`
		CED    string `json:"ced"`
		Status int    `json:"status"`
	} `json:"data"`
}

func SynthesizeIflytekTTS(ctx context.Context, options LocalTTSOptions) (LocalTTSReport, error) {
	start := time.Now()
	config, missingEnv := loadIflytekTTSConfig(options)
	report := LocalTTSReport{
		SchemaVersion: "a21.audio.local_tts.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "failed",
		Provider:      iflytekTTSEngine,
		Engine:        iflytekTTSEngine,
		EndpointHost:  localTTSEndpointHostLabel(config.Endpoint),
		NetworkMode:   "direct",
		Voice:         safeLocalTTSIdentifier(config.Voice),
		OutputFormat:  fmt.Sprintf("wav_pcm_s16le_%d_mono", iflytekTTSSampleRateHz),
		TextBytes:     len([]byte(options.Text)),
	}
	if strings.TrimSpace(options.Text) == "" {
		report.Findings = append(report.Findings, "text is required")
		return report, fmt.Errorf("local TTS text is required")
	}
	outputDir := strings.TrimSpace(options.OutputDir)
	if outputDir == "" {
		outputDir = "reports"
	}
	if err := validateLocalTTSOutputDir(outputDir); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	if len(missingEnv) > 0 {
		report.Status = "skipped"
		for _, name := range missingEnv {
			report.Findings = append(report.Findings, "missing "+name)
		}
		return report, nil
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		report.Findings = append(report.Findings, "output_dir_unavailable")
		return report, fmt.Errorf("local TTS output dir unavailable")
	}

	authURL, endpointHost, err := buildIflytekTTSAuthURL(config, time.Now())
	if err != nil {
		report.Findings = append(report.Findings, "iflytek_tts_auth_url_invalid")
		return report, fmt.Errorf("iflytek TTS auth URL invalid")
	}
	report.EndpointHost = endpointHost
	conn, statusCode, err := iflytekTTSWebSocketDial(ctx, authURL)
	if err != nil {
		report.Findings = append(report.Findings, iflytekTTSDialFailureFinding(statusCode))
		return report, fmt.Errorf("iflytek TTS websocket dial failed")
	}
	defer conn.Close(websocket.StatusNormalClosure, "a21 iflytek tts complete")

	payload, err := json.Marshal(buildIflytekTTSRequest(config, options.Text))
	if err != nil {
		report.Findings = append(report.Findings, "iflytek_tts_request_encode_failed")
		return report, fmt.Errorf("iflytek TTS request encode failed")
	}
	if err := conn.Write(ctx, websocket.MessageText, payload); err != nil {
		report.Findings = append(report.Findings, "iflytek_tts_request_write_failed")
		return report, fmt.Errorf("iflytek TTS websocket write failed")
	}

	pcm, firstAudioMS, err := readIflytekTTSPCM(ctx, conn, start)
	if err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, fmt.Errorf("iflytek TTS synthesis failed")
	}
	wavPath := filepath.Join(outputDir, uniqueLocalTTSFilename("a21-iflytek-tts"))
	if err := WritePCM16MonoWAV(wavPath, iflytekTTSSampleRateHz, pcm); err != nil {
		report.Findings = append(report.Findings, "iflytek_tts_wav_write_failed")
		return report, fmt.Errorf("iflytek TTS wav write failed")
	}
	info, err := os.Stat(wavPath)
	if err != nil {
		report.Findings = append(report.Findings, "output wav missing")
		return report, fmt.Errorf("iflytek TTS output wav missing")
	}
	report.Status = "passed"
	report.OutputPath = wavPath
	report.OutputBytes = info.Size()
	report.DurationMS = elapsedLocalTTSMS(start)
	report.TTSFirstAudioMS = firstAudioMS
	attachLocalTTSAudioQuality(&report, iflytekTTSSampleRateHz)
	return report, nil
}

func loadIflytekTTSConfig(options LocalTTSOptions) (iflytekTTSConfig, []string) {
	config := iflytekTTSConfig{
		AppID:     strings.TrimSpace(os.Getenv("A21_IFLYTEK_TTS_APP_ID")),
		APIKey:    strings.TrimSpace(os.Getenv("A21_IFLYTEK_TTS_API_KEY")),
		APISecret: strings.TrimSpace(os.Getenv("A21_IFLYTEK_TTS_API_SECRET")),
		Endpoint:  strings.TrimSpace(iflytekTTSEndpoint()),
		Voice:     firstNonEmptyLocalTTS(options.Voice, iflytekTTSDefaultVoice),
	}
	if config.Endpoint == "" {
		config.Endpoint = iflytekTTSDefaultEndpoint
	}
	var missing []string
	if config.AppID == "" {
		missing = append(missing, "A21_IFLYTEK_TTS_APP_ID")
	}
	if config.APIKey == "" {
		missing = append(missing, "A21_IFLYTEK_TTS_API_KEY")
	}
	if config.APISecret == "" {
		missing = append(missing, "A21_IFLYTEK_TTS_API_SECRET")
	}
	return config, missing
}

func buildIflytekTTSAuthURL(config iflytekTTSConfig, now time.Time) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(config.Endpoint))
	if err != nil {
		return "", "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", "", fmt.Errorf("iflytek TTS endpoint requires websocket scheme and host")
	}
	if parsed.Path == "" {
		parsed.Path = "/v2/tts"
	}
	endpointHost := urlHostLabel(parsed)
	date := now.UTC().Format(time.RFC1123)
	date = strings.Replace(date, "UTC", "GMT", 1)
	signatureOrigin := strings.Join([]string{
		"host: " + parsed.Host,
		"date: " + date,
		"GET " + parsed.EscapedPath() + " HTTP/1.1",
	}, "\n")
	mac := hmac.New(sha256.New, []byte(config.APISecret))
	_, _ = mac.Write([]byte(signatureOrigin))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	authorizationOrigin := fmt.Sprintf(`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`, config.APIKey, signature)
	query := parsed.Query()
	query.Set("authorization", base64.StdEncoding.EncodeToString([]byte(authorizationOrigin)))
	query.Set("date", date)
	query.Set("host", parsed.Host)
	parsed.RawQuery = query.Encode()
	return parsed.String(), endpointHost, nil
}

func buildIflytekTTSRequest(config iflytekTTSConfig, text string) iflytekTTSRequest {
	var request iflytekTTSRequest
	request.Common.AppID = config.AppID
	request.Business.AUE = "raw"
	request.Business.AUF = "audio/L16;rate=16000"
	request.Business.VCN = config.Voice
	request.Business.TTE = "UTF8"
	request.Business.Speed = 50
	request.Business.Volume = 50
	request.Business.Pitch = 50
	request.Data.Status = 2
	request.Data.Text = base64.StdEncoding.EncodeToString([]byte(text))
	return request
}

func readIflytekTTSPCM(ctx context.Context, conn iflytekTTSWebSocketConn, start time.Time) ([]byte, float64, error) {
	var pcm bytes.Buffer
	var firstAudioMS float64
	for {
		messageType, payload, err := conn.Read(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("iflytek_tts_response_read_failed")
		}
		if messageType != websocket.MessageText {
			return nil, 0, fmt.Errorf("iflytek_tts_response_not_text")
		}
		var response iflytekTTSResponse
		if err := json.Unmarshal(payload, &response); err != nil {
			return nil, 0, fmt.Errorf("iflytek_tts_response_decode_failed")
		}
		if response.Code != 0 {
			return nil, 0, fmt.Errorf("iflytek_tts_provider_error_code_%d", response.Code)
		}
		if response.Data == nil {
			continue
		}
		if strings.TrimSpace(response.Data.Audio) != "" {
			chunk, err := base64.StdEncoding.DecodeString(response.Data.Audio)
			if err != nil {
				return nil, 0, fmt.Errorf("iflytek_tts_audio_decode_failed")
			}
			if len(chunk) > 0 {
				if firstAudioMS == 0 {
					firstAudioMS = elapsedLocalTTSMS(start)
				}
				_, _ = pcm.Write(chunk)
			}
		}
		if response.Data.Status == 2 {
			break
		}
	}
	if pcm.Len() == 0 {
		return nil, 0, fmt.Errorf("iflytek_tts_audio_empty")
	}
	if firstAudioMS == 0 {
		firstAudioMS = elapsedLocalTTSMS(start)
	}
	return pcm.Bytes(), firstAudioMS, nil
}

func iflytekTTSDialFailureFinding(statusCode int) string {
	if statusCode > 0 {
		return fmt.Sprintf("iflytek_tts_websocket_dial_failed_http_%d", statusCode)
	}
	return "iflytek_tts_websocket_dial_failed"
}

func localTTSEndpointHostLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		return urlHostLabel(parsed)
	}
	value = strings.TrimPrefix(value, "ws://")
	value = strings.TrimPrefix(value, "wss://")
	if index := strings.IndexAny(value, "/?"); index >= 0 {
		value = value[:index]
	}
	if host, port, err := net.SplitHostPort(value); err == nil {
		if port != "" {
			return net.JoinHostPort(host, port)
		}
		return host
	}
	return value
}

func urlHostLabel(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	host := parsed.Hostname()
	if host == "" {
		return ""
	}
	if port := parsed.Port(); port != "" {
		return net.JoinHostPort(host, port)
	}
	return host
}

func SynthesizeMacOSSay(ctx context.Context, options LocalTTSOptions) (LocalTTSReport, error) {
	start := time.Now()
	outputSampleRateHz := localTTSOutputSampleRateHz(options)
	report := LocalTTSReport{
		SchemaVersion: "a21.audio.local_tts.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "failed",
		Provider:      "macos_say",
		Voice:         firstNonEmptyLocalTTS(options.Voice, "Tingting"),
		OutputFormat:  fmt.Sprintf("wav_pcm_s16le_%d_mono", outputSampleRateHz),
		TextBytes:     len([]byte(options.Text)),
	}
	if strings.TrimSpace(options.Text) == "" {
		report.Findings = append(report.Findings, "text is required")
		return report, fmt.Errorf("local TTS text is required")
	}
	outputDir := strings.TrimSpace(options.OutputDir)
	if outputDir == "" {
		outputDir = "reports"
	}
	if err := validateLocalTTSOutputDir(outputDir); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	sayPath := options.SayPath
	if sayPath == "" {
		found, err := exec.LookPath("say")
		if err != nil {
			report.Status = "skipped"
			report.Findings = append(report.Findings, "macOS say command is missing")
			return report, nil
		}
		sayPath = found
	}
	afconvertPath := options.AFConvertPath
	if afconvertPath == "" {
		found, err := exec.LookPath("afconvert")
		if err != nil {
			report.Status = "skipped"
			report.Findings = append(report.Findings, "macOS afconvert command is missing")
			return report, nil
		}
		afconvertPath = found
	}
	runner := options.CommandRunner
	if runner == nil {
		runner = execCommandRunner{}
	}
	tempDir, err := os.MkdirTemp("", "a21-local-tts-*")
	if err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	defer os.RemoveAll(tempDir)

	aiffPath := filepath.Join(tempDir, "a21-local-tts.aiff")
	wavPath := filepath.Join(outputDir, uniqueLocalTTSFilename("a21-local-tts"))
	if err := runner.Run(ctx, sayPath, "-v", report.Voice, "-o", aiffPath, options.Text); err != nil {
		report.Findings = append(report.Findings, "say command failed")
		return report, err
	}
	if err := runner.Run(ctx, afconvertPath, "-f", "WAVE", "-d", fmt.Sprintf("LEI16@%d", outputSampleRateHz), aiffPath, wavPath); err != nil {
		report.Findings = append(report.Findings, "afconvert command failed")
		return report, err
	}
	info, err := os.Stat(wavPath)
	if err != nil {
		report.Findings = append(report.Findings, "output wav missing")
		return report, err
	}
	report.Status = "passed"
	report.OutputPath = wavPath
	report.OutputBytes = info.Size()
	report.DurationMS = elapsedLocalTTSMS(start)
	report.TTSFirstAudioMS = report.DurationMS
	attachLocalTTSAudioQuality(&report, outputSampleRateHz)
	return report, nil
}

func SynthesizeSherpaONNX(ctx context.Context, options LocalTTSOptions) (LocalTTSReport, error) {
	start := time.Now()
	outputSampleRateHz := localTTSOutputSampleRateHz(options)
	speakerID := options.SpeakerID
	if speakerID == 0 {
		speakerID = 21
	}
	report := LocalTTSReport{
		SchemaVersion: "a21.audio.local_tts.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "failed",
		Provider:      "sherpa_onnx",
		Engine:        "vits_icefall_zh_aishell3",
		Voice:         "sid_" + strconv.Itoa(speakerID),
		OutputFormat:  fmt.Sprintf("wav_pcm_s16le_%d_mono", outputSampleRateHz),
		TextBytes:     len([]byte(options.Text)),
	}
	if strings.TrimSpace(options.Text) == "" {
		report.Findings = append(report.Findings, "text is required")
		return report, fmt.Errorf("local TTS text is required")
	}
	outputDir := strings.TrimSpace(options.OutputDir)
	if outputDir == "" {
		outputDir = "reports"
	}
	if err := validateLocalTTSOutputDir(outputDir); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	modelDir, modelDirExplicit := firstNonEmptyLocalTTS(options.ModelDir, defaultSherpaONNXModelDir()), strings.TrimSpace(options.ModelDir) != ""
	report.ModelDir = filepath.Base(filepath.Clean(modelDir))
	if err := validateSherpaONNXModelDir(modelDir); err != nil {
		report.Findings = append(report.Findings, err.Error())
		if !modelDirExplicit {
			report.Status = "skipped"
			return report, nil
		}
		return report, err
	}
	pythonPath, ok := resolveLocalTTSPath(options.PythonPath, defaultSherpaONNXPythonPath())
	if !ok {
		report.Status = "skipped"
		report.Findings = append(report.Findings, "isolated sherpa-onnx python is missing")
		return report, nil
	}
	scriptPath, ok := resolveLocalTTSPath(options.ScriptPath, defaultSherpaONNXScriptPath())
	if !ok {
		report.Status = "skipped"
		report.Findings = append(report.Findings, "a21 sherpa-onnx script is missing")
		return report, nil
	}
	afconvertPath := options.AFConvertPath
	if afconvertPath == "" {
		found, err := exec.LookPath("afconvert")
		if err != nil {
			report.Status = "skipped"
			report.Findings = append(report.Findings, "macOS afconvert command is missing")
			return report, nil
		}
		afconvertPath = found
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	runner := options.CommandRunner
	if runner == nil {
		runner = execCommandRunner{}
	}
	tempDir, err := os.MkdirTemp("", "a21-sherpa-onnx-tts-*")
	if err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	defer os.RemoveAll(tempDir)

	textPath := filepath.Join(tempDir, "a21-tts-input.txt")
	rawWAVPath := filepath.Join(tempDir, "a21-sherpa-onnx-tts.raw.wav")
	wavPath := filepath.Join(outputDir, uniqueLocalTTSFilename("a21-sherpa-onnx-tts"))
	if err := os.WriteFile(textPath, []byte(options.Text), 0o600); err != nil {
		report.Findings = append(report.Findings, err.Error())
		return report, err
	}
	if err := runner.Run(ctx, pythonPath, scriptPath, "--model-dir", modelDir, "--text-file", textPath, "--output", rawWAVPath, "--speaker-id", strconv.Itoa(speakerID)); err != nil {
		report.Findings = append(report.Findings, "sherpa-onnx synthesis command failed")
		return report, err
	}
	if err := runner.Run(ctx, afconvertPath, "-f", "WAVE", "-d", fmt.Sprintf("LEI16@%d", outputSampleRateHz), rawWAVPath, wavPath); err != nil {
		report.Findings = append(report.Findings, "afconvert command failed")
		return report, err
	}
	info, err := os.Stat(wavPath)
	if err != nil {
		report.Findings = append(report.Findings, "output wav missing")
		return report, err
	}
	report.Status = "passed"
	report.OutputPath = wavPath
	report.OutputBytes = info.Size()
	report.DurationMS = elapsedLocalTTSMS(start)
	report.TTSFirstAudioMS = report.DurationMS
	attachLocalTTSAudioQuality(&report, outputSampleRateHz)
	return report, nil
}
