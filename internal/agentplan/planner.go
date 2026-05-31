package agentplan

import (
	"strings"
	"unicode/utf8"

	"a21.local/a21/internal/protocol"
)

const SchemaVersion = "a21.agent_plan.v1"

type ExecutionPath string

const (
	ExecutionNativeCore      ExecutionPath = "native_core"
	ExecutionAgentIO         ExecutionPath = "agent_io"
	ExecutionV21Professional ExecutionPath = "v21_professional"
)

type MemoryPolicy string

const (
	MemoryPolicyNone                    MemoryPolicy = "none"
	MemoryPolicyEphemeral               MemoryPolicy = "ephemeral"
	MemoryPolicyCandidate               MemoryPolicy = "candidate"
	MemoryPolicyConfirmedSave           MemoryPolicy = "confirmed_save"
	MemoryPolicyProfessionalPointerOnly MemoryPolicy = "professional_pointer_only"
)

type PrivacyState string

const (
	PrivacyDefault          PrivacyState = "default"
	PrivacyPublic           PrivacyState = "public"
	PrivacyPrivate          PrivacyState = "private"
	PrivacyProfessionalOnly PrivacyState = "professional_only"
)

type MemorySource string

const (
	MemorySourceTaskWindow        MemorySource = "task_window"
	MemorySourceUserPreference    MemorySource = "user_preference"
	MemorySourceProjectConvention MemorySource = "project_convention"
	MemorySourceConfiguration     MemorySource = "configuration_choice"
	MemorySourcePrivate           MemorySource = "private"
	MemorySourceCompanion         MemorySource = "companion"
	MemorySourceV21Evidence       MemorySource = "v21_evidence"
	MemorySourceExternalRaw       MemorySource = "external_agent_raw"
)

type MemoryItem struct {
	ID        string       `json:"id,omitempty"`
	Source    MemorySource `json:"source"`
	Text      string       `json:"text"`
	Sensitive bool         `json:"sensitive,omitempty"`
}

type MemorySnippet struct {
	ID     string       `json:"id,omitempty"`
	Source MemorySource `json:"source"`
	Text   string       `json:"text"`
}

type AgentIOBinding struct {
	Enabled            bool   `json:"enabled"`
	Name               string `json:"name,omitempty"`
	EndpointConfigured bool   `json:"endpoint_configured,omitempty"`
}

type PlannerRequest struct {
	TraceID                 string         `json:"trace_id"`
	SessionID               string         `json:"session_id"`
	DeviceID                string         `json:"device_id"`
	Mode                    protocol.Mode  `json:"mode"`
	PrivacyState            PrivacyState   `json:"privacy_state,omitempty"`
	TaskSummary             string         `json:"task_summary,omitempty"`
	CurrentTurnTranscript   string         `json:"current_turn_transcript,omitempty"`
	ExpectedOutput          string         `json:"expected_output,omitempty"`
	Constraints             []string       `json:"constraints,omitempty"`
	TaskWindowMemory        []MemoryItem   `json:"task_window_memory,omitempty"`
	NonSensitivePreferences []MemoryItem   `json:"non_sensitive_preferences,omitempty"`
	AgentBinding            AgentIOBinding `json:"agent_binding,omitempty"`
	V21Requested            bool           `json:"v21_requested,omitempty"`
}

type Plan struct {
	SchemaVersion    string          `json:"schema_version"`
	TraceID          string          `json:"trace_id"`
	SessionID        string          `json:"session_id"`
	DeviceID         string          `json:"device_id"`
	Mode             protocol.Mode   `json:"mode"`
	PrivacyState     PrivacyState    `json:"privacy_state"`
	ExecutionPath    ExecutionPath   `json:"execution_path"`
	PersonaPack      string          `json:"persona_pack"`
	MemoryPolicy     MemoryPolicy    `json:"memory_policy"`
	ScenarioPlaybook string          `json:"scenario_playbook"`
	V21Allowed       bool            `json:"v21_allowed"`
	AgentIOAllowed   bool            `json:"agent_io_allowed"`
	AgentIO          *AgentIOMessage `json:"agent_io,omitempty"`
	Expression       ExpressionPlan  `json:"expression"`
	Markers          []string        `json:"markers"`
	Findings         []Finding       `json:"findings,omitempty"`
}

type AgentIOMessage struct {
	ExternalAgentName string          `json:"external_agent_name"`
	TraceID           string          `json:"trace_id"`
	SessionID         string          `json:"session_id"`
	Mode              protocol.Mode   `json:"mode"`
	TaskSummary       string          `json:"task_summary"`
	ExpectedOutput    string          `json:"expected_output"`
	Constraints       []string        `json:"constraints"`
	TaskWindowMemory  []MemorySnippet `json:"task_window_memory,omitempty"`
	Preferences       []MemorySnippet `json:"preferences,omitempty"`
	InputSource       string          `json:"input_source"`
	Boundary          []string        `json:"boundary"`
}

type ExpressionPlan struct {
	State      protocol.ExpressionState `json:"state"`
	Screen     string                   `json:"screen"`
	OutputHint string                   `json:"output_hint"`
}

type Finding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func BuildPlan(request PlannerRequest) Plan {
	mode := request.Mode
	if mode == "" {
		mode = protocol.ModeWorkmate
	}
	privacy := normalizePrivacy(mode, request.PrivacyState)
	plan := Plan{
		SchemaVersion:    SchemaVersion,
		TraceID:          strings.TrimSpace(request.TraceID),
		SessionID:        strings.TrimSpace(request.SessionID),
		DeviceID:         strings.TrimSpace(request.DeviceID),
		Mode:             mode,
		PrivacyState:     privacy,
		ExecutionPath:    ExecutionNativeCore,
		PersonaPack:      personaPack(mode, privacy),
		MemoryPolicy:     memoryPolicy(mode, privacy, request),
		ScenarioPlaybook: scenarioPlaybook(mode),
		Expression:       expressionPlan(mode, ExecutionNativeCore),
		Markers:          []string{"agent.plan.start", "memory.policy.selected", "scenario.selected"},
	}
	if mode == protocol.ModeProfessional {
		plan.ExecutionPath = ExecutionV21Professional
		plan.V21Allowed = true
		plan.MemoryPolicy = MemoryPolicyProfessionalPointerOnly
		plan.Expression = expressionPlan(mode, plan.ExecutionPath)
		plan.Findings = append(plan.Findings, Finding{Code: "agent_io_blocked_professional", Message: "Agent I/O is blocked in professional mode; use the V21 adapter boundary"})
		return finalizePlan(plan)
	}
	if request.V21Requested {
		plan.Findings = append(plan.Findings, Finding{Code: "v21_requires_professional", Message: "V21 access requires explicit professional mode"})
	}
	if canUseAgentIO(mode, privacy) {
		if request.AgentBinding.Enabled && request.AgentBinding.EndpointConfigured && strings.TrimSpace(request.AgentBinding.Name) != "" {
			plan.ExecutionPath = ExecutionAgentIO
			plan.AgentIOAllowed = true
			plan.AgentIO = buildAgentIOMessage(request, mode)
			plan.Expression = expressionPlan(mode, plan.ExecutionPath)
			plan.Markers = append(plan.Markers, "agent_io.message.sent")
		} else {
			plan.Findings = append(plan.Findings, Finding{Code: "agent_io_not_configured", Message: "Agent I/O remains native because no explicit configured binding is enabled"})
		}
	} else if isAgentIOMode(mode) {
		plan.Findings = append(plan.Findings, Finding{Code: "agent_io_blocked_private", Message: "Agent I/O is blocked by private privacy state"})
	} else if request.AgentBinding.Enabled {
		plan.Findings = append(plan.Findings, Finding{Code: "agent_io_blocked_mode", Message: "Agent I/O is only allowed in co_creation or roleplay"})
	}
	return finalizePlan(plan)
}

func finalizePlan(plan Plan) Plan {
	plan.Markers = append(plan.Markers, "agent.plan.selected")
	return plan
}

func normalizePrivacy(mode protocol.Mode, explicit PrivacyState) PrivacyState {
	if explicit != "" {
		return explicit
	}
	switch mode {
	case protocol.ModePrivate:
		return PrivacyPrivate
	case protocol.ModePublic:
		return PrivacyPublic
	case protocol.ModeProfessional:
		return PrivacyProfessionalOnly
	default:
		return PrivacyDefault
	}
}

func canUseAgentIO(mode protocol.Mode, privacy PrivacyState) bool {
	return isAgentIOMode(mode) && privacy != PrivacyPrivate && privacy != PrivacyProfessionalOnly
}

func isAgentIOMode(mode protocol.Mode) bool {
	return mode == protocol.ModeCoCreation || mode == protocol.ModeRoleplay
}

func buildAgentIOMessage(request PlannerRequest, mode protocol.Mode) *AgentIOMessage {
	summary, inputSource := taskSummary(request.TaskSummary, request.CurrentTurnTranscript)
	return &AgentIOMessage{
		ExternalAgentName: strings.TrimSpace(request.AgentBinding.Name),
		TraceID:           strings.TrimSpace(request.TraceID),
		SessionID:         strings.TrimSpace(request.SessionID),
		Mode:              mode,
		TaskSummary:       summary,
		ExpectedOutput:    firstNonEmpty(request.ExpectedOutput, defaultExpectedOutput(mode)),
		Constraints:       append(defaultAgentIOConstraints(), compactStrings(request.Constraints)...),
		TaskWindowMemory:  allowedMemory(request.TaskWindowMemory),
		Preferences:       allowedPreferences(request.NonSensitivePreferences),
		InputSource:       inputSource,
		Boundary: []string{
			"A21 remains the voice, persona, privacy, memory, and expression authority",
			"External agent output is a draft and must be rewritten by A21 before user-facing expression",
			"No V21 evidence body, private history, firmware command, device control, or provider secret may be sent",
		},
	}
}

func taskSummary(summary string, transcript string) (string, string) {
	if clean := oneLine(summary); clean != "" {
		return clean, "task_summary"
	}
	if clean := oneLine(transcript); clean != "" {
		return truncateRunes(clean, 240), "fallback_current_turn"
	}
	return "A21 task context is minimal; ask for a concise draft that A21 can rewrite.", "fallback_minimal"
}

func allowedMemory(items []MemoryItem) []MemorySnippet {
	var out []MemorySnippet
	for _, item := range items {
		if item.Sensitive || item.Source != MemorySourceTaskWindow {
			continue
		}
		text := oneLine(item.Text)
		if text == "" {
			continue
		}
		out = append(out, MemorySnippet{ID: item.ID, Source: item.Source, Text: truncateRunes(text, 180)})
	}
	return out
}

func allowedPreferences(items []MemoryItem) []MemorySnippet {
	var out []MemorySnippet
	for _, item := range items {
		if item.Sensitive {
			continue
		}
		switch item.Source {
		case MemorySourceUserPreference, MemorySourceProjectConvention, MemorySourceConfiguration:
			text := oneLine(item.Text)
			if text != "" {
				out = append(out, MemorySnippet{ID: item.ID, Source: item.Source, Text: truncateRunes(text, 180)})
			}
		}
	}
	return out
}

func memoryPolicy(mode protocol.Mode, privacy PrivacyState, request PlannerRequest) MemoryPolicy {
	switch {
	case mode == protocol.ModeProfessional:
		return MemoryPolicyProfessionalPointerOnly
	case privacy == PrivacyPrivate:
		return MemoryPolicyEphemeral
	case len(allowedPreferences(request.NonSensitivePreferences)) > 0:
		return MemoryPolicyCandidate
	case isAgentIOMode(mode):
		return MemoryPolicyEphemeral
	default:
		return MemoryPolicyNone
	}
}

func personaPack(mode protocol.Mode, privacy PrivacyState) string {
	if privacy == PrivacyPrivate {
		return "private_boundary_workmate"
	}
	switch mode {
	case protocol.ModeProfessional:
		return "professional_evidence"
	case protocol.ModeCoCreation:
		return "co_creation_rewriter"
	case protocol.ModeRoleplay:
		return "office_roleplay"
	case protocol.ModeFocus:
		return "quiet_focus"
	default:
		return "workmate_default"
	}
}

func scenarioPlaybook(mode protocol.Mode) string {
	switch mode {
	case protocol.ModeCoCreation:
		return "messy_input_to_usable_language"
	case protocol.ModeRoleplay:
		return "office_scenario_rehearsal"
	case protocol.ModeProfessional:
		return "v21_evidence_handoff"
	case protocol.ModeFocus:
		return "low_interruption_focus"
	default:
		return "default_workmate"
	}
}

func expressionPlan(mode protocol.Mode, path ExecutionPath) ExpressionPlan {
	switch path {
	case ExecutionAgentIO:
		return ExpressionPlan{State: protocol.ExpressionThinking, Screen: "AGENT IO", OutputHint: "draft_pending"}
	case ExecutionV21Professional:
		return ExpressionPlan{State: protocol.ExpressionProfessional, Screen: "PRO / V21", OutputHint: "evidence_pending"}
	default:
		if mode == protocol.ModePrivate {
			return ExpressionPlan{State: protocol.ExpressionListening, Screen: "PRIVATE", OutputHint: "local_only"}
		}
		return ExpressionPlan{State: protocol.ExpressionListening, Screen: strings.ToUpper(string(mode)), OutputHint: "a21_native"}
	}
}

func defaultExpectedOutput(mode protocol.Mode) string {
	if mode == protocol.ModeRoleplay {
		return "roleplay draft for A21 to rewrite"
	}
	return "co-creation draft for A21 to rewrite"
}

func defaultAgentIOConstraints() []string {
	return []string{
		"Return draft text only; A21 decides user-facing wording",
		"Do not call firmware, device-control, provider-secret, or V21-internal tools",
		"Do not store private, V21 evidence, or raw external-agent output as memory",
	}
}

func compactStrings(values []string) []string {
	var out []string
	for _, value := range values {
		if clean := oneLine(value); clean != "" {
			out = append(out, clean)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if clean := oneLine(value); clean != "" {
			return clean
		}
	}
	return ""
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}
