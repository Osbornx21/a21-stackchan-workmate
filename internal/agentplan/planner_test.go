package agentplan

import (
	"encoding/json"
	"strings"
	"testing"

	"a21.local/a21/internal/protocol"
)

func TestCoCreationUsesConfiguredAgentIOWithTaskWindowMemoryOnly(t *testing.T) {
	plan := BuildPlan(PlannerRequest{
		TraceID:        "a21-trace-agent-io-1",
		SessionID:      "a21-session-agent-io-1",
		DeviceID:       "stackchan-sim-001",
		Mode:           protocol.ModeCoCreation,
		TaskSummary:    "turn this rough complaint into review language",
		ExpectedOutput: "meeting-ready wording",
		Constraints:    []string{"keep it sharp but not cruel"},
		TaskWindowMemory: []MemoryItem{
			{ID: "keep", Source: MemorySourceTaskWindow, Text: "current task is PRD review"},
			{ID: "private", Source: MemorySourcePrivate, Text: "private emotional content"},
			{ID: "v21", Source: MemorySourceV21Evidence, Text: "raw V21 citation detail"},
			{ID: "raw", Source: MemorySourceExternalRaw, Text: "external agent raw output"},
			{ID: "sensitive", Source: MemorySourceTaskWindow, Text: "coworker-sensitive comment", Sensitive: true},
		},
		NonSensitivePreferences: []MemoryItem{
			{ID: "tone", Source: MemorySourceUserPreference, Text: "prefer concise product language"},
			{ID: "blocked", Source: MemorySourceCompanion, Text: "companion history"},
		},
		AgentBinding: AgentIOBinding{Enabled: true, Name: "Hermes", EndpointConfigured: true},
	})

	if plan.ExecutionPath != ExecutionAgentIO || !plan.AgentIOAllowed || plan.AgentIO == nil {
		t.Fatalf("plan execution = %q allowed=%v agent=%#v", plan.ExecutionPath, plan.AgentIOAllowed, plan.AgentIO)
	}
	if plan.V21Allowed {
		t.Fatal("V21Allowed = true, want false for co_creation Agent I/O")
	}
	if plan.MemoryPolicy != MemoryPolicyCandidate {
		t.Fatalf("memory policy = %q, want candidate because safe preferences are present", plan.MemoryPolicy)
	}
	if got := len(plan.AgentIO.TaskWindowMemory); got != 1 {
		t.Fatalf("task window memory count = %d, want 1: %#v", got, plan.AgentIO.TaskWindowMemory)
	}
	if plan.AgentIO.TaskWindowMemory[0].Text != "current task is PRD review" {
		t.Fatalf("task memory = %#v", plan.AgentIO.TaskWindowMemory)
	}
	if got := len(plan.AgentIO.Preferences); got != 1 {
		t.Fatalf("preferences count = %d, want 1: %#v", got, plan.AgentIO.Preferences)
	}
	for _, marker := range []string{"agent.plan.start", "memory.policy.selected", "scenario.selected", "agent_io.message.sent", "agent.plan.selected"} {
		if !containsString(plan.Markers, marker) {
			t.Fatalf("markers lack %q: %#v", marker, plan.Markers)
		}
	}
	rendered := mustJSON(t, plan)
	for _, forbidden := range []string{"private emotional content", "raw V21 citation detail", "external agent raw output", "coworker-sensitive comment", "companion history"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("plan leaked forbidden memory %q: %s", forbidden, rendered)
		}
	}
}

func TestProfessionalModeUsesV21AndBlocksAgentIO(t *testing.T) {
	plan := BuildPlan(PlannerRequest{
		TraceID:      "a21-trace-pro-1",
		SessionID:    "a21-session-pro-1",
		DeviceID:     "stackchan-sim-001",
		Mode:         protocol.ModeProfessional,
		TaskSummary:  "查一下历史证据",
		AgentBinding: AgentIOBinding{Enabled: true, Name: "Mimo", EndpointConfigured: true},
		TaskWindowMemory: []MemoryItem{
			{ID: "evidence", Source: MemorySourceV21Evidence, Text: "full evidence body must not persist"},
		},
	})

	if plan.ExecutionPath != ExecutionV21Professional || !plan.V21Allowed || plan.AgentIOAllowed || plan.AgentIO != nil {
		t.Fatalf("professional plan = %#v", plan)
	}
	if plan.MemoryPolicy != MemoryPolicyProfessionalPointerOnly {
		t.Fatalf("memory policy = %q, want professional pointer only", plan.MemoryPolicy)
	}
	if !findingContains(plan.Findings, "agent_io_blocked_professional") {
		t.Fatalf("findings = %#v, want professional block", plan.Findings)
	}
	if strings.Contains(mustJSON(t, plan), "full evidence body") {
		t.Fatalf("professional plan leaked V21 evidence body: %s", mustJSON(t, plan))
	}
}

func TestPrivateModeBlocksAgentIOEvenWhenBound(t *testing.T) {
	plan := BuildPlan(PlannerRequest{
		TraceID:      "a21-trace-private-1",
		SessionID:    "a21-session-private-1",
		Mode:         protocol.ModeCoCreation,
		PrivacyState: PrivacyPrivate,
		TaskSummary:  "help rewrite this",
		AgentBinding: AgentIOBinding{Enabled: true, Name: "Hermes", EndpointConfigured: true},
	})

	if plan.ExecutionPath != ExecutionNativeCore || plan.AgentIOAllowed || plan.AgentIO != nil {
		t.Fatalf("private plan = %#v", plan)
	}
	if plan.MemoryPolicy != MemoryPolicyEphemeral {
		t.Fatalf("memory policy = %q, want ephemeral", plan.MemoryPolicy)
	}
	if !findingContains(plan.Findings, "agent_io_blocked_private") {
		t.Fatalf("findings = %#v, want private block", plan.Findings)
	}
}

func TestAgentIOFallsBackToCurrentTurnWhenSummaryUnavailable(t *testing.T) {
	plan := BuildPlan(PlannerRequest{
		TraceID:               "a21-trace-fallback-1",
		SessionID:             "a21-session-fallback-1",
		Mode:                  protocol.ModeRoleplay,
		CurrentTurnTranscript: "老板追问的时候我想先说明风险和边界",
		AgentBinding:          AgentIOBinding{Enabled: true, Name: "OpenClaw", EndpointConfigured: true},
	})

	if plan.ExecutionPath != ExecutionAgentIO || plan.AgentIO == nil {
		t.Fatalf("plan = %#v, want Agent I/O", plan)
	}
	if plan.AgentIO.InputSource != "fallback_current_turn" {
		t.Fatalf("input source = %q, want fallback_current_turn", plan.AgentIO.InputSource)
	}
	if !strings.Contains(plan.AgentIO.TaskSummary, "老板追问") {
		t.Fatalf("task summary = %q, want current-turn fallback", plan.AgentIO.TaskSummary)
	}
	if len(plan.AgentIO.Constraints) < 3 {
		t.Fatalf("constraints = %#v, want default safety constraints", plan.AgentIO.Constraints)
	}
}

func TestAgentIORequiresAllowedModeAndConfiguredBinding(t *testing.T) {
	workmate := BuildPlan(PlannerRequest{
		Mode:         protocol.ModeWorkmate,
		TaskSummary:  "summarize",
		AgentBinding: AgentIOBinding{Enabled: true, Name: "Hermes", EndpointConfigured: true},
	})
	if workmate.AgentIOAllowed || workmate.ExecutionPath != ExecutionNativeCore {
		t.Fatalf("workmate plan = %#v, want native core", workmate)
	}
	if !findingContains(workmate.Findings, "agent_io_blocked_mode") {
		t.Fatalf("workmate findings = %#v", workmate.Findings)
	}

	unbound := BuildPlan(PlannerRequest{
		Mode:        protocol.ModeRoleplay,
		TaskSummary: "rehearse",
	})
	if unbound.AgentIOAllowed || unbound.ExecutionPath != ExecutionNativeCore {
		t.Fatalf("unbound plan = %#v, want native core", unbound)
	}
	if !findingContains(unbound.Findings, "agent_io_not_configured") {
		t.Fatalf("unbound findings = %#v", unbound.Findings)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func findingContains(findings []Finding, want string) bool {
	for _, finding := range findings {
		if finding.Code == want {
			return true
		}
	}
	return false
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
