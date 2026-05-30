package providers

import (
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
