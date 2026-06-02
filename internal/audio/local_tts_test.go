package audio

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
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

func TestIflytekTTSMissingEnvSkipsWithoutSecretLeak(t *testing.T) {
	t.Setenv("A21_IFLYTEK_TTS_APP_ID", "")
	t.Setenv("A21_IFLYTEK_TTS_API_KEY", "")
	t.Setenv("A21_IFLYTEK_TTS_API_SECRET", "")

	report, err := SynthesizeIflytekTTS(context.Background(), LocalTTSOptions{
		Text:      "这段文本不能出现在报告里",
		OutputDir: t.TempDir(),
	})

	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "skipped" || report.Provider != "iflytek_tts" || report.Engine != "iflytek_tts" {
		t.Fatalf("report identity/status = %+v", report)
	}
	for _, want := range []string{"missing A21_IFLYTEK_TTS_APP_ID", "missing A21_IFLYTEK_TTS_API_KEY", "missing A21_IFLYTEK_TTS_API_SECRET"} {
		if !stringSliceContainsLocalTTS(report.Findings, want) {
			t.Fatalf("findings missing %q: %+v", want, report.Findings)
		}
	}
	rendered := mustJSON(t, report)
	for _, forbidden := range []string{"这段文本不能出现在报告里", "authorization", "Authorization", "Bearer", "http://", "wss://", "data_base64", "raw_audio"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("iflytek missing-env report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestIflytekTTSWebSocketClientIgnoresAmbientProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy-secret@127.0.0.1:7891")
	t.Setenv("HTTPS_PROXY", "http://proxy-secret@127.0.0.1:7891")

	client := iflytekTTSWebSocketHTTPClient()

	if client.Timeout != 15*time.Second {
		t.Fatalf("timeout = %s, want 15s", client.Timeout)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		req, _ := http.NewRequest(http.MethodGet, "https://tts-api.xfyun.cn/v2/tts", nil)
		proxyURL, _ := transport.Proxy(req)
		t.Fatalf("iflytek TTS websocket transport inherited proxy %v", proxyURL)
	}
}

func TestIflytekTTSAuthURLShapeWithoutRawSecret(t *testing.T) {
	config := iflytekTTSConfig{
		AppID:     "a21-test-app",
		APIKey:    "a21-test-api-key",
		APISecret: "a21-test-api-secret",
		Endpoint:  "wss://tts-api.xfyun.cn/v2/tts",
		Voice:     "xiaoyan",
	}

	signedURL, endpointHost, err := buildIflytekTTSAuthURL(config, time.Date(2026, 6, 3, 5, 4, 3, 0, time.UTC))

	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(signedURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if parsed.Scheme != "wss" || parsed.Host != "tts-api.xfyun.cn" || parsed.Path != "/v2/tts" {
		t.Fatalf("signed URL shape = %s", signedURL)
	}
	if endpointHost != "tts-api.xfyun.cn" || query.Get("host") != "tts-api.xfyun.cn" || !strings.Contains(query.Get("date"), "GMT") {
		t.Fatalf("host/date = endpoint:%q query:%v", endpointHost, query)
	}
	authorization := query.Get("authorization")
	if authorization == "" {
		t.Fatal("authorization query is empty")
	}
	decodedAuthorization, err := base64.StdEncoding.DecodeString(authorization)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`api_key="a21-test-api-key"`, `algorithm="hmac-sha256"`, `headers="host date request-line"`, `signature="`} {
		if !strings.Contains(string(decodedAuthorization), want) {
			t.Fatalf("authorization missing %q: %s", want, decodedAuthorization)
		}
	}
	for _, forbidden := range []string{config.APIKey, config.APISecret} {
		if strings.Contains(signedURL, forbidden) {
			t.Fatalf("signed URL leaked raw secret %q: %s", forbidden, signedURL)
		}
	}
}

func TestIflytekTTSSynthesizesPCMChunksToRedactedWAVReport(t *testing.T) {
	const (
		appID     = "a21-test-app-id"
		apiKey    = "a21-test-api-key"
		apiSecret = "a21-test-api-secret"
		rawText   = "真实合成文本不能进入报告"
	)
	t.Setenv("A21_IFLYTEK_TTS_APP_ID", appID)
	t.Setenv("A21_IFLYTEK_TTS_API_KEY", apiKey)
	t.Setenv("A21_IFLYTEK_TTS_API_SECRET", apiSecret)
	chunkOne := pcm16Bytes(0, 900, -900, 1200)
	chunkTwo := pcm16Bytes(-1200, 1600, -1600, 0)
	chunkOneBase64 := base64.StdEncoding.EncodeToString(chunkOne)
	chunkTwoBase64 := base64.StdEncoding.EncodeToString(chunkTwo)
	queryCh := make(chan url.Values, 1)
	requestCh := make(chan iflytekTTSRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/tts" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		queryCh <- r.URL.Query()
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close(websocket.StatusNormalClosure, "test done")
		messageType, payload, err := conn.Read(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if messageType != websocket.MessageText {
			t.Fatalf("message type = %v, want text", messageType)
		}
		var request iflytekTTSRequest
		if err := json.Unmarshal(payload, &request); err != nil {
			t.Fatal(err)
		}
		requestCh <- request
		if err := conn.Write(context.Background(), websocket.MessageText, []byte(`{"code":0,"message":"success","sid":"a21-test-sid","data":{"audio":"`+chunkOneBase64+`","status":1}}`)); err != nil {
			t.Fatal(err)
		}
		if err := conn.Write(context.Background(), websocket.MessageText, []byte(`{"code":0,"message":"success","sid":"a21-test-sid","data":{"audio":"`+chunkTwoBase64+`","status":2}}`)); err != nil {
			t.Fatal(err)
		}
	}))
	t.Cleanup(server.Close)
	originalEndpoint := iflytekTTSEndpoint
	t.Cleanup(func() { iflytekTTSEndpoint = originalEndpoint })
	iflytekTTSEndpoint = func() string {
		return "ws" + strings.TrimPrefix(server.URL, "http") + "/v2/tts"
	}
	dir := t.TempDir()

	report, err := SynthesizeIflytekTTS(context.Background(), LocalTTSOptions{
		Text:      rawText,
		Voice:     "x4_xiaoyan",
		OutputDir: dir,
	})

	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || report.Provider != "iflytek_tts" || report.Engine != "iflytek_tts" {
		t.Fatalf("report identity/status = %+v", report)
	}
	if report.EndpointHost == "" || strings.Contains(report.EndpointHost, "/v2/tts") || strings.Contains(report.EndpointHost, "ws://") {
		t.Fatalf("endpoint host = %q, want host label only", report.EndpointHost)
	}
	if report.Voice != "x4_xiaoyan" || report.OutputFormat != "wav_pcm_s16le_16000_mono" {
		t.Fatalf("voice/output format = %q/%q", report.Voice, report.OutputFormat)
	}
	if report.OutputBytes <= 44 || report.TTSFirstAudioMS <= 0 || report.AudioQuality == nil {
		t.Fatalf("output/timing/quality = %+v", report)
	}
	if report.AudioQuality.SampleRateHz != 16000 || report.AudioQuality.Codec != "pcm_s16le" {
		t.Fatalf("audio quality = %+v", report.AudioQuality)
	}
	query := <-queryCh
	if query.Get("authorization") == "" || query.Get("date") == "" || query.Get("host") == "" {
		t.Fatalf("auth query missing fields: %v", query)
	}
	request := <-requestCh
	if request.Common.AppID != appID ||
		request.Business.AUE != "raw" ||
		request.Business.AUF != "audio/L16;rate=16000" ||
		request.Business.VCN != "x4_xiaoyan" ||
		request.Business.TTE != "UTF8" ||
		request.Data.Status != 2 {
		t.Fatalf("iflytek request = %+v", request)
	}
	decodedText, err := base64.StdEncoding.DecodeString(request.Data.Text)
	if err != nil {
		t.Fatal(err)
	}
	if string(decodedText) != rawText {
		t.Fatalf("decoded text = %q", decodedText)
	}
	rendered := mustJSON(t, report)
	for _, forbidden := range []string{
		rawText,
		appID,
		apiKey,
		apiSecret,
		query.Get("authorization"),
		chunkOneBase64,
		chunkTwoBase64,
		server.URL,
		"/v2/tts",
		dir,
		report.OutputPath,
		"http://",
		"wss://",
		"data_base64",
		"raw_audio",
	} {
		if forbidden != "" && strings.Contains(rendered, forbidden) {
			t.Fatalf("iflytek report leaked %q: %s", forbidden, rendered)
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

func stringSliceContainsLocalTTS(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
