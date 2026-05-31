package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAgentPlanBuildsAgentIOPlan(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"agent-plan",
		"--mode", "co_creation",
		"--task", "把这段抱怨改成评审语言",
		"--agent", "hermes_agent",
		"--agent-name", "Hermes",
		"--endpoint-url", "http://127.0.0.1:21130/a21/agent-task",
		"--task-memory", "m1:当前任务是座舱 PRD 评审",
		"--preference", "tone:偏好短句",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.agent_plan_cli.v1"`,
		`"execution_path": "agent_io"`,
		`"memory_policy": "candidate"`,
		`"external_agent_name": "Hermes"`,
		`"agent_io.message.sent"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunAgentIOSmokeDryRunRedactsTaskAndEndpoint(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"agent-io-smoke",
		"--endpoint-url", "http://127.0.0.1:21130/a21/agent-task",
		"--task", "真实任务正文不要进入报告",
		"--task-memory", "m1:真实记忆也不要进入报告",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.agent_io_smoke.v1"`,
		`"status": "ready"`,
		`"executed": false`,
		`"endpoint": "loopback:21130"`,
		`"task_text_stored": false`,
		`"memory_text_stored": false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-agent-io-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"真实任务正文", "真实记忆", "http://127.0.0.1:21130"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(data), forbidden) {
			t.Fatalf("agent io smoke leaked %q: stdout=%s report=%s", forbidden, stdout.String(), string(data))
		}
	}
}

func TestRunAgentIOSmokeWithoutEndpointReportsNeedsEndpoint(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"agent-io-smoke"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"status": "needs_endpoint"`,
		`"configured": false`,
		`"executed": false`,
		`"execution_path": "agent_io"`,
		`"agent_io_endpoint_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunAgentIOSmokeExecutePostsToHermesAndRedactsResponse(t *testing.T) {
	var sawRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		var request struct {
			SchemaVersion  string `json:"schema_version"`
			AgentProfile   string `json:"agent_profile"`
			AgentIOMessage struct {
				ExternalAgentName string   `json:"external_agent_name"`
				Mode              string   `json:"mode"`
				TaskSummary       string   `json:"task_summary"`
				Constraints       []string `json:"constraints"`
				TaskWindowMemory  []struct {
					Text string `json:"text"`
				} `json:"task_window_memory"`
			} `json:"agent_io_message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		sawRequest = request.SchemaVersion == "a21.agent_io.task.v1" &&
			request.AgentProfile == "hermes_agent" &&
			request.AgentIOMessage.ExternalAgentName == "Hermes" &&
			request.AgentIOMessage.Mode == "co_creation" &&
			request.AgentIOMessage.TaskSummary == "Hermes should receive this task" &&
			len(request.AgentIOMessage.TaskWindowMemory) == 1 &&
			request.AgentIOMessage.TaskWindowMemory[0].Text == "Hermes should receive this task memory"
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: external draft body must stay out of A21 reports\n\ndata: [DONE]\n\n"))
	}))
	defer server.Close()
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"agent-io-smoke",
		"--endpoint-url", server.URL,
		"--task", "Hermes should receive this task",
		"--task-memory", "m1:Hermes should receive this task memory",
		"--execute",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !sawRequest {
		t.Fatal("Hermes test endpoint did not receive expected A21 Agent I/O payload")
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"executed": true`,
		`"response_status_code": 200`,
		`"response_content_type": "text/event-stream"`,
		`"response_event_count": 1`,
		`"external_agent_text_stored": false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-agent-io-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{server.URL, "Hermes should receive this task", "Hermes should receive this task memory", "external draft body"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(data), forbidden) {
			t.Fatalf("execute report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), string(data))
		}
	}
}

func TestRunAgentIOSmokeExecuteUsesOpenAICompatibleHermesBaseURL(t *testing.T) {
	t.Setenv("A21_HERMES_AGENT_KEY", "test-bearer-token")
	var sawRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s, want /v1/chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-bearer-token" {
			t.Fatalf("authorization = %q, want bearer header", got)
		}
		var request struct {
			Model    string `json:"model"`
			Stream   bool   `json:"stream"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			Metadata map[string]string `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		sawRequest = request.Model == "hermes-agent" &&
			request.Stream &&
			len(request.Messages) == 2 &&
			request.Messages[1].Role == "user" &&
			strings.Contains(request.Messages[1].Content, "Hermes base URL task") &&
			request.Metadata["schema_version"] == "a21.agent_io.task.v1" &&
			request.Metadata["agent_profile"] == "hermes_agent"
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"external draft body must stay out of reports"}}]}`))
	}))
	defer server.Close()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"agent-io-smoke",
		"--endpoint-url", server.URL + "/v1",
		"--task", "Hermes base URL task",
		"--execute",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !sawRequest {
		t.Fatal("Hermes OpenAI-compatible endpoint did not receive expected request")
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"response_status_code": 200`,
		`"response_content_type": "application/json"`,
		`"external_agent_text_stored": false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"test-bearer-token", server.URL, "Hermes base URL task", "external draft body"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("execute report leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunAgentIOSmokePassesWhenSSEStreamStaysOpen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: first event stays out of reports\n\n"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
	}))
	defer server.Close()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"agent-io-smoke",
		"--endpoint-url", server.URL,
		"--task", "SSE task text must not be stored",
		"--execute",
		"--timeout-ms", "1000",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"response_content_type": "text/event-stream"`,
		`"response_event_count": 1`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"SSE task text", "first event"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("SSE report leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunAgentIOSmokeBlocksProfessionalAndPrivateModes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "professional",
			args: []string{"agent-io-smoke", "--mode", "professional", "--endpoint-url", "http://127.0.0.1:21130/a21/agent-task"},
			want: "agent_io_blocked_professional",
		},
		{
			name: "private",
			args: []string{"agent-io-smoke", "--mode", "co_creation", "--privacy-state", "private", "--endpoint-url", "http://127.0.0.1:21130/a21/agent-task"},
			want: "agent_io_blocked_private",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), tc.want) {
				t.Fatalf("stdout missing %q: %s", tc.want, stdout.String())
			}
		})
	}
}

func TestRunAgentIOSmokeRejectsUnsafeEndpointWithoutEchoingURL(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"agent-io-smoke",
		"--endpoint-url", "http://user:secret@127.0.0.1:21130/v1/devices/control",
		"--execute",
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"agent_io_endpoint_invalid"`) {
		t.Fatalf("stdout missing invalid endpoint finding: %s", stdout.String())
	}
	for _, forbidden := range []string{"user:secret", "/v1/devices/control"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("unsafe endpoint leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}
