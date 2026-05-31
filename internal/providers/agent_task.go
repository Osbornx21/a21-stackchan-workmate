package providers

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

type AgentTaskEventKind string

const (
	AgentTaskEventStarted          AgentTaskEventKind = "started"
	AgentTaskEventProgress         AgentTaskEventKind = "progress"
	AgentTaskEventTextDelta        AgentTaskEventKind = "text_delta"
	AgentTaskEventToolCallRedacted AgentTaskEventKind = "tool_call_redacted"
	AgentTaskEventResult           AgentTaskEventKind = "result"
	AgentTaskEventError            AgentTaskEventKind = "error"
	AgentTaskEventFinal            AgentTaskEventKind = "final"
)

type AgentTaskFakeStyle string

const (
	AgentTaskFakeStyleSSE   AgentTaskFakeStyle = "sse"
	AgentTaskFakeStyleHTTP  AgentTaskFakeStyle = "http"
	AgentTaskFakeStyleStdio AgentTaskFakeStyle = "stdio"
)

type AgentTaskRequest struct {
	TraceID   string            `json:"trace_id"`
	SessionID string            `json:"session_id"`
	Task      string            `json:"task"`
	Context   map[string]string `json:"context,omitempty"`
}

type AgentTaskEvent struct {
	Kind        AgentTaskEventKind `json:"kind"`
	TraceID     string             `json:"trace_id"`
	SessionID   string             `json:"session_id"`
	Text        string             `json:"-"`
	ToolName    string             `json:"-"`
	ToolPayload string             `json:"-"`
	ErrorCode   string             `json:"error_code,omitempty"`
	Final       bool               `json:"final,omitempty"`
	StreamStyle string             `json:"stream_style,omitempty"`
}

type AgentTaskProvider interface {
	StartAgentTask(ctx context.Context, request AgentTaskRequest) (<-chan AgentTaskEvent, error)
}

type A21AgentTaskSemanticEvent struct {
	SchemaVersion    string `json:"schema_version,omitempty"`
	Kind             string `json:"kind"`
	TraceID          string `json:"trace_id"`
	SessionID        string `json:"session_id"`
	Final            bool   `json:"final,omitempty"`
	TextLength       int    `json:"text_length,omitempty"`
	RedactedText     bool   `json:"redacted_text,omitempty"`
	ToolCallRedacted bool   `json:"tool_call_redacted,omitempty"`
	ErrorCode        string `json:"error_code,omitempty"`
}

type A21AgentTaskSemanticReport struct {
	SchemaVersion string                      `json:"schema_version"`
	StreamStyle   string                      `json:"stream_style,omitempty"`
	Events        []A21AgentTaskSemanticEvent `json:"events"`
	Final         bool                        `json:"final"`
	Findings      []ProviderCatalogFinding    `json:"findings,omitempty"`
}

type fakeAgentTaskProvider struct {
	style  AgentTaskFakeStyle
	events []AgentTaskEvent
}

func NewFakeAgentTaskProvider(style AgentTaskFakeStyle, events []AgentTaskEvent) AgentTaskProvider {
	return fakeAgentTaskProvider{style: style, events: append([]AgentTaskEvent(nil), events...)}
}

func (p fakeAgentTaskProvider) StartAgentTask(ctx context.Context, request AgentTaskRequest) (<-chan AgentTaskEvent, error) {
	if findings := ValidateAgentTaskRequest(request); len(findings) > 0 {
		return nil, fmt.Errorf("agent task request rejected by A21 safety guard")
	}
	out := make(chan AgentTaskEvent)
	go func() {
		defer close(out)
		for _, event := range p.events {
			if event.TraceID == "" {
				event.TraceID = request.TraceID
			}
			if event.SessionID == "" {
				event.SessionID = request.SessionID
			}
			if event.StreamStyle == "" {
				event.StreamStyle = string(p.style)
			}
			select {
			case <-ctx.Done():
				return
			case out <- event:
			}
		}
	}()
	return out, nil
}

func ConsumeAgentTaskEvents(ctx context.Context, events <-chan AgentTaskEvent) (A21AgentTaskSemanticReport, error) {
	if events == nil {
		return A21AgentTaskSemanticReport{}, fmt.Errorf("agent task event stream is nil")
	}
	report := A21AgentTaskSemanticReport{
		SchemaVersion: "a21.agent_task.semantic_report.v1",
	}
	for {
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		case event, ok := <-events:
			if !ok {
				return report, nil
			}
			if report.StreamStyle == "" {
				report.StreamStyle = event.StreamStyle
			}
			semantic := MapAgentTaskEventToA21Semantic(event)
			if semantic.Final {
				report.Final = true
			}
			report.Events = append(report.Events, semantic)
		}
	}
}

func MapAgentTaskEventToA21Semantic(event AgentTaskEvent) A21AgentTaskSemanticEvent {
	semantic := A21AgentTaskSemanticEvent{
		SchemaVersion: "a21.agent_task.semantic_event.v1",
		Kind:          string(event.Kind),
		TraceID:       event.TraceID,
		SessionID:     event.SessionID,
		Final:         event.Final || event.Kind == AgentTaskEventFinal,
		ErrorCode:     safeAgentTaskErrorCode(event.ErrorCode),
	}
	if event.Text != "" {
		semantic.TextLength = utf8.RuneCountInString(event.Text)
		semantic.RedactedText = true
	}
	if event.Kind == AgentTaskEventToolCallRedacted || event.ToolName != "" || event.ToolPayload != "" {
		semantic.ToolCallRedacted = true
	}
	return semantic
}

func ValidateAgentTaskRequest(request AgentTaskRequest) []ProviderCatalogFinding {
	var findings []ProviderCatalogFinding
	scan := strings.ToLower(request.Task)
	for key, value := range request.Context {
		scan += " " + strings.ToLower(key) + " " + strings.ToLower(value)
	}
	add := func(code, message string) {
		if !agentTaskHasFinding(findings, code) {
			findings = append(findings, ProviderCatalogFinding{Code: code, Message: message, Detail: "A21_AGENT_TASK_REQUEST"})
		}
	}
	if strings.Contains(scan, "firmware") || strings.Contains(scan, "nvs") || strings.Contains(scan, "flash") || strings.Contains(scan, "raw upload") || strings.Contains(scan, "write_flash") {
		add("agent_task_forbidden_firmware", "Agent task requests cannot perform firmware, NVS, flash, raw upload, or write commands")
	}
	if strings.Contains(scan, "provider_env") || strings.Contains(scan, "provider env") || strings.Contains(scan, "env mutation") || containsAgentTaskProviderEnvIdentity(scan) {
		add("agent_task_forbidden_provider_env", "Agent task requests cannot mutate or expose provider environment values")
	}
	if strings.Contains(scan, "v21") && (strings.Contains(scan, "internal") || strings.Contains(scan, "mutation") || strings.Contains(scan, "write")) {
		add("agent_task_forbidden_v21_internals", "Agent task requests cannot mutate V21 internals")
	}
	if strings.Contains(scan, "gateway") && (strings.Contains(scan, "runtime") || strings.Contains(scan, "state") || strings.Contains(scan, "write")) {
		add("agent_task_forbidden_gateway", "Agent task requests cannot write Gateway runtime state")
	}
	if strings.Contains(scan, "/v1/devices/control") {
		add("agent_task_forbidden_device_control", "Agent task requests cannot call the physical device control path")
	}
	if strings.Contains(scan, "/dev/cu.") || strings.Contains(scan, "/dev/tty.") || strings.Contains(scan, "physical device path") {
		add("agent_task_forbidden_physical_path", "Agent task requests cannot use physical device paths")
	}
	return findings
}

func agentTaskHasFinding(findings []ProviderCatalogFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func containsAgentTaskProviderEnvIdentity(value string) bool {
	if !strings.Contains(value, "a21_") {
		return false
	}
	for _, marker := range []string{
		"api_key",
		"base_url",
		"model",
		"provider",
		"deepseek",
		"openai",
		"doubao",
		"dashscope",
		"siliconflow",
		"stepfun",
		"moonshot",
		"volcengine",
	} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func safeAgentTaskErrorCode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "agent_task_error"
	}
	return b.String()
}
