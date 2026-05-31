package providers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentTaskProviderContractConsumesFakeEventStream(t *testing.T) {
	var _ AgentTaskProvider = NewFakeAgentTaskProvider(AgentTaskFakeStyleSSE, []AgentTaskEvent{
		{Kind: AgentTaskEventStarted, TraceID: "a21-trace-agent-1", SessionID: "a21-session-agent-1"},
		{Kind: AgentTaskEventProgress, TraceID: "a21-trace-agent-1", SessionID: "a21-session-agent-1"},
		{Kind: AgentTaskEventTextDelta, TraceID: "a21-trace-agent-1", SessionID: "a21-session-agent-1", Text: "external draft body must not persist"},
		{Kind: AgentTaskEventToolCallRedacted, TraceID: "a21-trace-agent-1", SessionID: "a21-session-agent-1", ToolName: "external_tool", ToolPayload: "credential=do-not-store"},
		{Kind: AgentTaskEventResult, TraceID: "a21-trace-agent-1", SessionID: "a21-session-agent-1", Text: "final external answer"},
		{Kind: AgentTaskEventFinal, TraceID: "a21-trace-agent-1", SessionID: "a21-session-agent-1", Final: true},
	})

	provider := NewFakeAgentTaskProvider(AgentTaskFakeStyleSSE, []AgentTaskEvent{
		{Kind: AgentTaskEventStarted, TraceID: "a21-trace-agent-1", SessionID: "a21-session-agent-1"},
		{Kind: AgentTaskEventTextDelta, TraceID: "a21-trace-agent-1", SessionID: "a21-session-agent-1", Text: "external draft body must not persist"},
		{Kind: AgentTaskEventToolCallRedacted, TraceID: "a21-trace-agent-1", SessionID: "a21-session-agent-1", ToolName: "external_tool", ToolPayload: "credential=do-not-store"},
		{Kind: AgentTaskEventFinal, TraceID: "a21-trace-agent-1", SessionID: "a21-session-agent-1", Final: true},
	})
	stream, err := provider.StartAgentTask(context.Background(), AgentTaskRequest{
		TraceID:   "a21-trace-agent-1",
		SessionID: "a21-session-agent-1",
		Task:      "summarize product review notes",
		Context:   map[string]string{"mode": "co_creation"},
	})
	if err != nil {
		t.Fatalf("StartAgentTask returned error: %v", err)
	}

	report, err := ConsumeAgentTaskEvents(context.Background(), stream)
	if err != nil {
		t.Fatalf("ConsumeAgentTaskEvents returned error: %v", err)
	}
	if report.SchemaVersion != "a21.agent_task.semantic_report.v1" {
		t.Fatalf("schema_version = %q", report.SchemaVersion)
	}
	if report.StreamStyle != string(AgentTaskFakeStyleSSE) {
		t.Fatalf("stream_style = %q, want sse", report.StreamStyle)
	}
	if len(report.Events) != 4 {
		t.Fatalf("events = %d, want 4: %#v", len(report.Events), report.Events)
	}
	if report.Events[1].Kind != string(AgentTaskEventTextDelta) || report.Events[1].TextLength == 0 || !report.Events[1].RedactedText {
		t.Fatalf("text delta semantic event was not redacted: %#v", report.Events[1])
	}
	if !report.Events[2].ToolCallRedacted {
		t.Fatalf("tool event did not record redacted marker: %#v", report.Events[2])
	}
	if !report.Final {
		t.Fatalf("report final = false, want true")
	}

	rendered := agentTaskMustJSON(t, report)
	for _, forbidden := range []string{
		"external draft body",
		"final external answer",
		"credential=do-not-store",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("semantic report leaked external payload %q: %s", forbidden, rendered)
		}
	}
}

func TestAgentTaskFakeProviderConsumesSupportedStreamStyles(t *testing.T) {
	for _, style := range []AgentTaskFakeStyle{AgentTaskFakeStyleSSE, AgentTaskFakeStyleHTTP, AgentTaskFakeStyleStdio} {
		provider := NewFakeAgentTaskProvider(style, []AgentTaskEvent{
			{Kind: AgentTaskEventStarted},
			{Kind: AgentTaskEventFinal, Final: true},
		})
		stream, err := provider.StartAgentTask(context.Background(), AgentTaskRequest{
			TraceID:   "a21-trace-agent-style",
			SessionID: "a21-session-agent-style",
			Task:      "consume fake stream",
		})
		if err != nil {
			t.Fatalf("%s StartAgentTask error: %v", style, err)
		}
		report, err := ConsumeAgentTaskEvents(context.Background(), stream)
		if err != nil {
			t.Fatalf("%s ConsumeAgentTaskEvents error: %v", style, err)
		}
		if report.StreamStyle != string(style) || len(report.Events) != 2 || !report.Final {
			t.Fatalf("%s report = %#v", style, report)
		}
	}
}

func TestAgentTaskMapperRedactsExternalPayloads(t *testing.T) {
	semantic := MapAgentTaskEventToA21Semantic(AgentTaskEvent{
		Kind:        AgentTaskEventResult,
		TraceID:     "a21-trace-agent-2",
		SessionID:   "a21-session-agent-2",
		Text:        "raw agent result should not be saved",
		ToolName:    "shell",
		ToolPayload: "A21_OPENAI_API_KEY=value",
		Final:       true,
	})

	if semantic.Kind != string(AgentTaskEventResult) {
		t.Fatalf("kind = %q", semantic.Kind)
	}
	if semantic.TraceID != "a21-trace-agent-2" || semantic.SessionID != "a21-session-agent-2" {
		t.Fatalf("trace/session not preserved: %#v", semantic)
	}
	if semantic.TextLength == 0 || !semantic.RedactedText || !semantic.ToolCallRedacted || !semantic.Final {
		t.Fatalf("semantic event did not preserve redacted markers: %#v", semantic)
	}
	rendered := agentTaskMustJSON(t, semantic)
	for _, forbidden := range []string{"raw agent result", "A21_OPENAI_API_KEY", "shell"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("semantic event leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestAgentTaskRequestSafetyRejectsForbiddenIntentContextAndKeys(t *testing.T) {
	request := AgentTaskRequest{
		TraceID:   "a21-trace-agent-3",
		SessionID: "a21-session-agent-3",
		Task:      "firmware command write NVS flash raw upload",
		Context: map[string]string{
			"provider_env_mutation": "set A21_DEEPSEEK_MODEL",
			"target_url":            "http://127.0.0.1:21080/v1/devices/control",
			"physical_path":         "/dev/cu.usbmodem1101",
			"gateway_state":         "write runtime state",
			"v21_internals":         "mutation write",
		},
	}

	findings := ValidateAgentTaskRequest(request)
	if len(findings) == 0 {
		t.Fatal("findings = 0, want forbidden agent task findings")
	}
	for _, want := range []string{
		"agent_task_forbidden_firmware",
		"agent_task_forbidden_provider_env",
		"agent_task_forbidden_v21_internals",
		"agent_task_forbidden_gateway",
		"agent_task_forbidden_device_control",
		"agent_task_forbidden_physical_path",
	} {
		if !agentTaskFindingContains(findings, want) {
			t.Fatalf("findings lack %q: %#v", want, findings)
		}
	}
	rendered := agentTaskMustJSON(t, findings)
	for _, forbidden := range []string{"/v1/devices/control", "/dev/cu.usbmodem1101", "A21_DEEPSEEK_MODEL"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("finding leaked forbidden context %q: %s", forbidden, rendered)
		}
	}
}

func TestAgentTaskRequestSafetyAllowsA21SemanticContextWithoutProviderEnv(t *testing.T) {
	findings := ValidateAgentTaskRequest(AgentTaskRequest{
		TraceID:   "a21-trace-agent-4",
		SessionID: "a21-session-agent-4",
		Task:      "summarize A21 product review notes",
		Context: map[string]string{
			"a21_mode": "co_creation",
		},
	})

	if len(findings) != 0 {
		t.Fatalf("findings = %#v, want no findings for ordinary A21 semantic context", findings)
	}
}

func TestAgentTaskEventKindsCoverPhase5Contract(t *testing.T) {
	for _, kind := range []AgentTaskEventKind{
		AgentTaskEventStarted,
		AgentTaskEventProgress,
		AgentTaskEventTextDelta,
		AgentTaskEventToolCallRedacted,
		AgentTaskEventResult,
		AgentTaskEventError,
		AgentTaskEventFinal,
	} {
		if strings.TrimSpace(string(kind)) == "" {
			t.Fatalf("empty agent task event kind: %#v", kind)
		}
	}
}

func agentTaskFindingContains(findings []ProviderCatalogFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func agentTaskMustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
