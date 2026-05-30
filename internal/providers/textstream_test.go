package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseOpenAICompatibleTextStreamDeltas(t *testing.T) {
	stream := strings.NewReader(strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"你"}}]}`,
		`data: {"choices":[{"delta":{"content":"好"}}]}`,
		`data: {"choices":[{"delta":{"reasoning":"先判断意图"}}]}`,
		`data: [DONE]`,
		``,
	}, "\n"))

	result, err := ParseOpenAICompatibleTextStream(stream)

	if err != nil {
		t.Fatal(err)
	}
	if got := result.ContentText(); got != "你好" {
		t.Fatalf("content text = %q, want 你好", got)
	}
	if got := result.ReasoningText(); got != "先判断意图" {
		t.Fatalf("reasoning text = %q, want 先判断意图", got)
	}
	if !result.Done {
		t.Fatal("done = false, want true")
	}
	if result.ContentDeltaCount != 2 || result.ReasoningDeltaCount != 1 {
		t.Fatalf("delta counts = content %d reasoning %d", result.ContentDeltaCount, result.ReasoningDeltaCount)
	}
}

func TestParseOpenAICompatibleTextStreamSupportsReasoningContent(t *testing.T) {
	stream := strings.NewReader(strings.Join([]string{
		`event: message`,
		`data: {"choices":[{"delta":{"reasoning_content":"拆成三步"}}]}`,
		`data: {"choices":[{"delta":{"content":"结论"}}]}`,
		`data: [DONE]`,
		``,
	}, "\n"))

	result, err := ParseOpenAICompatibleTextStream(stream)

	if err != nil {
		t.Fatal(err)
	}
	if result.ReasoningText() != "拆成三步" {
		t.Fatalf("reasoning text = %q", result.ReasoningText())
	}
	if result.ContentText() != "结论" {
		t.Fatalf("content text = %q", result.ContentText())
	}
}

func TestRedactedProviderHTTPErrorDoesNotLeakBody(t *testing.T) {
	detail := redactedProviderHTTPError(401, []byte(`{"error":"bad sk-a21-secret for deepseek-v4-flash"}`))

	for _, forbidden := range []string{"sk-a21-secret", "deepseek-v4-flash", "bad"} {
		if strings.Contains(detail, forbidden) {
			t.Fatalf("detail leaked %q: %s", forbidden, detail)
		}
	}
	for _, want := range []string{"HTTP status 401", "body_sha256=", "body_bytes="} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail missing %q: %s", want, detail)
		}
	}
}

func TestRunTextStreamCompletionFromEnvUsesDeepSeekProfile(t *testing.T) {
	var sawAuth bool
	var sawModel bool
	var sawPrompt bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization") == "Bearer sk-a21-secret"
		var body struct {
			Model    string `json:"model"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
			Stream bool `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		sawModel = body.Model == "deepseek-v4-flash"
		sawPrompt = len(body.Messages) == 1 && body.Messages[0].Content == "不要进报告"
		if !body.Stream {
			t.Fatal("stream = false, want true")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"reasoning_content":"先想一下"}}]}`,
			`data: {"choices":[{"delta":{"content":"给 StackChan 一个短回复"}}]}`,
			`data: [DONE]`,
			``,
		}, "\n")))
	}))
	defer server.Close()

	result, err := RunTextStreamCompletionFromEnv(context.Background(), []string{
		"A21_LAB_DEEPSEEK_API_KEY=sk-a21-secret",
		"A21_DEEPSEEK_BASE_URL=" + server.URL,
	}, TextStreamCompletionOptions{
		ProviderName: "deepseek",
		Prompt:       "不要进报告",
		Client:       server.Client(),
	})

	if err != nil {
		t.Fatal(err)
	}
	if !sawAuth || !sawModel || !sawPrompt {
		t.Fatalf("saw auth/model/prompt = %v/%v/%v, want true/true/true", sawAuth, sawModel, sawPrompt)
	}
	if result.Provider != "deepseek" || result.Family != ProviderFamilyTextStream {
		t.Fatalf("provider/family = %q/%q", result.Provider, result.Family)
	}
	if result.ContentText != "给 StackChan 一个短回复" || !result.Done {
		t.Fatalf("content/done = %q/%v", result.ContentText, result.Done)
	}
	if result.FirstByteMS <= 0 || result.FirstContentMS <= 0 || result.TotalDurationMS <= 0 {
		t.Fatalf("timings missing: %+v", result)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-v4-flash", "不要进报告", "给 StackChan 一个短回复", "先想一下"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("text stream result leaked %q: %s", forbidden, data)
		}
	}
}
