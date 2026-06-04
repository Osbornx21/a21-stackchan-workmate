package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/agentplan"
	"a21.local/a21/internal/protocol"
)

type agentPlanCLIReport struct {
	SchemaVersion string         `json:"schema_version"`
	Plan          agentplan.Plan `json:"plan"`
	ReportPath    string         `json:"report_path,omitempty"`
}

type agentIOSmokeReport struct {
	SchemaVersion         string                  `json:"schema_version"`
	GeneratedAtMS         int64                   `json:"generated_at_ms"`
	AgentProfile          string                  `json:"agent_profile"`
	ExternalAgentName     string                  `json:"external_agent_name,omitempty"`
	Status                string                  `json:"status"`
	Configured            bool                    `json:"configured"`
	Executed              bool                    `json:"executed"`
	Endpoint              string                  `json:"endpoint,omitempty"`
	Method                string                  `json:"method,omitempty"`
	Mode                  protocol.Mode           `json:"mode"`
	PrivacyState          agentplan.PrivacyState  `json:"privacy_state"`
	ExecutionPath         agentplan.ExecutionPath `json:"execution_path"`
	MemoryPolicy          agentplan.MemoryPolicy  `json:"memory_policy"`
	AgentIOAllowed        bool                    `json:"agent_io_allowed"`
	PlannerFindings       []agentplan.Finding     `json:"planner_findings,omitempty"`
	RequestSummaryLength  int                     `json:"request_summary_length,omitempty"`
	TaskWindowMemoryCount int                     `json:"task_window_memory_count,omitempty"`
	PreferenceCount       int                     `json:"preference_count,omitempty"`
	ResponseStatusCode    int                     `json:"response_status_code,omitempty"`
	ResponseContentType   string                  `json:"response_content_type,omitempty"`
	ResponseDurationMS    float64                 `json:"response_duration_ms,omitempty"`
	ResponseTextLength    int                     `json:"response_text_length,omitempty"`
	ResponseEventCount    int                     `json:"response_event_count,omitempty"`
	Redaction             agentIOSmokeRedaction   `json:"redaction"`
	Findings              []agentIOSmokeFinding   `json:"findings,omitempty"`
	PlanMarkers           []string                `json:"plan_markers,omitempty"`
	Boundary              []string                `json:"boundary,omitempty"`
	ReportPath            string                  `json:"report_path,omitempty"`
}

type agentIOSmokeRedaction struct {
	TaskTextStored           bool `json:"task_text_stored"`
	MemoryTextStored         bool `json:"memory_text_stored"`
	ExternalAgentTextStored  bool `json:"external_agent_text_stored"`
	FullEndpointURLStored    bool `json:"full_endpoint_url_stored"`
	CredentialsStored        bool `json:"credentials_stored"`
	ProviderEnvValuesStored  bool `json:"provider_env_values_stored"`
	V21EvidenceBodyStored    bool `json:"v21_evidence_body_stored"`
	FirmwareOrDeviceCommands bool `json:"firmware_or_device_commands"`
}

type agentIOSmokeFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type agentIOOptions struct {
	Mode                  string
	PrivacyState          string
	TaskSummary           string
	CurrentTurnTranscript string
	ExpectedOutput        string
	Constraints           []string
	TaskMemory            []string
	PreferenceMemory      []string
	AgentProfile          string
	AgentName             string
	EndpointURL           string
	AgentKey              string
	AgentEnabled          bool
	EndpointConfigured    bool
	OutputDir             string
	Execute               bool
	TimeoutMS             int
}

func runAgentPlan(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultAgentIOOptions(os.Environ())
	options.OutputDir = ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 agent-plan [--mode co_creation] [--privacy-state default|public|private|professional_only] [--task <summary>] [--current-turn <text>] [--expected-output <text>] [--constraint <text>] [--task-memory <id:text>] [--preference <id:text>] [--agent hermes_agent] [--agent-name Hermes] [--endpoint-url <url>] [--output-dir reports]")
			return 0
		case "--mode":
			if !readStringOption(args, &i, stderr, "--mode", &options.Mode) {
				return 2
			}
		case "--privacy-state":
			if !readStringOption(args, &i, stderr, "--privacy-state", &options.PrivacyState) {
				return 2
			}
		case "--task":
			if !readStringOption(args, &i, stderr, "--task", &options.TaskSummary) {
				return 2
			}
		case "--current-turn":
			if !readStringOption(args, &i, stderr, "--current-turn", &options.CurrentTurnTranscript) {
				return 2
			}
		case "--expected-output":
			if !readStringOption(args, &i, stderr, "--expected-output", &options.ExpectedOutput) {
				return 2
			}
		case "--constraint":
			value := ""
			if !readStringOption(args, &i, stderr, "--constraint", &value) {
				return 2
			}
			options.Constraints = append(options.Constraints, value)
		case "--task-memory":
			value := ""
			if !readStringOption(args, &i, stderr, "--task-memory", &value) {
				return 2
			}
			options.TaskMemory = append(options.TaskMemory, value)
		case "--preference":
			value := ""
			if !readStringOption(args, &i, stderr, "--preference", &value) {
				return 2
			}
			options.PreferenceMemory = append(options.PreferenceMemory, value)
		case "--agent":
			if !readStringOption(args, &i, stderr, "--agent", &options.AgentProfile) {
				return 2
			}
		case "--agent-name":
			if !readStringOption(args, &i, stderr, "--agent-name", &options.AgentName) {
				return 2
			}
		case "--endpoint-url":
			if !readStringOption(args, &i, stderr, "--endpoint-url", &options.EndpointURL) {
				return 2
			}
			options.EndpointConfigured = true
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown agent-plan option %q\n", args[i])
			return 2
		}
	}
	report := agentPlanCLIReport{
		SchemaVersion: "a21.agent_plan_cli.v1",
		Plan:          buildAgentPlanFromOptions(options),
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "agent plan report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeAgentPlanReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write agent plan report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONAgentPlan(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode agent plan report: %v\n", err)
		return 1
	}
	return 0
}

func runAgentIOSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultAgentIOOptions(os.Environ())
	options.OutputDir = ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 agent-io-smoke [--agent hermes_agent] [--agent-name Hermes] [--endpoint-url http://127.0.0.1:21130/a21/agent-task] [--mode co_creation] [--task <summary>] [--current-turn <text>] [--execute] [--timeout-ms 5000] [--output-dir reports]")
			return 0
		case "--agent":
			if !readStringOption(args, &i, stderr, "--agent", &options.AgentProfile) {
				return 2
			}
		case "--agent-name":
			if !readStringOption(args, &i, stderr, "--agent-name", &options.AgentName) {
				return 2
			}
		case "--endpoint-url":
			if !readStringOption(args, &i, stderr, "--endpoint-url", &options.EndpointURL) {
				return 2
			}
			options.EndpointConfigured = true
		case "--mode":
			if !readStringOption(args, &i, stderr, "--mode", &options.Mode) {
				return 2
			}
		case "--privacy-state":
			if !readStringOption(args, &i, stderr, "--privacy-state", &options.PrivacyState) {
				return 2
			}
		case "--task":
			if !readStringOption(args, &i, stderr, "--task", &options.TaskSummary) {
				return 2
			}
		case "--current-turn":
			if !readStringOption(args, &i, stderr, "--current-turn", &options.CurrentTurnTranscript) {
				return 2
			}
		case "--expected-output":
			if !readStringOption(args, &i, stderr, "--expected-output", &options.ExpectedOutput) {
				return 2
			}
		case "--constraint":
			value := ""
			if !readStringOption(args, &i, stderr, "--constraint", &value) {
				return 2
			}
			options.Constraints = append(options.Constraints, value)
		case "--task-memory":
			value := ""
			if !readStringOption(args, &i, stderr, "--task-memory", &value) {
				return 2
			}
			options.TaskMemory = append(options.TaskMemory, value)
		case "--preference":
			value := ""
			if !readStringOption(args, &i, stderr, "--preference", &value) {
				return 2
			}
			options.PreferenceMemory = append(options.PreferenceMemory, value)
		case "--execute":
			options.Execute = true
		case "--timeout-ms":
			value := ""
			if !readStringOption(args, &i, stderr, "--timeout-ms", &value) {
				return 2
			}
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 100 || parsed > 60000 {
				fmt.Fprintln(stderr, "--timeout-ms requires an integer between 100 and 60000")
				return 2
			}
			options.TimeoutMS = parsed
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown agent-io-smoke option %q\n", args[i])
			return 2
		}
	}
	report := buildAgentIOSmokeReport(context.Background(), options)
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "agent io smoke report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeAgentIOSmokeReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write agent io smoke report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONAgentIOSmoke(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode agent io smoke report: %v\n", err)
		return 1
	}
	if report.Status == "failed" || (options.Execute && report.Status != "passed") {
		return 1
	}
	return 0
}

func defaultAgentIOOptions(env []string) agentIOOptions {
	profile := firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_AGENT_PROVIDER_PRIMARY")), "hermes_agent")
	endpoint := strings.TrimSpace(appEnvValue(env, "A21_AGENT_IO_ENDPOINT_URL"))
	agentKey := strings.TrimSpace(appEnvValue(env, "A21_AGENT_IO_API_KEY"))
	if profile == "hermes_agent" {
		endpoint = firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_HERMES_AGENT_URL")), endpoint)
		agentKey = firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_HERMES_AGENT_KEY")), agentKey)
	}
	return agentIOOptions{
		Mode:               "co_creation",
		PrivacyState:       string(agentplan.PrivacyDefault),
		TaskSummary:        "turn rough office input into usable product language",
		ExpectedOutput:     "A21-rewritable draft",
		AgentProfile:       profile,
		AgentName:          defaultAgentDisplayName(profile),
		EndpointURL:        endpoint,
		AgentKey:           agentKey,
		AgentEnabled:       strings.TrimSpace(profile) != "",
		EndpointConfigured: strings.TrimSpace(endpoint) != "",
		TimeoutMS:          5000,
	}
}

func buildAgentPlanFromOptions(options agentIOOptions) agentplan.Plan {
	return agentplan.BuildPlan(agentplan.PlannerRequest{
		TraceID:                 fmt.Sprintf("a21-trace-agent-plan-%d", time.Now().UnixNano()),
		SessionID:               fmt.Sprintf("a21-session-agent-plan-%d", time.Now().UnixNano()),
		DeviceID:                "none_host_fixture",
		Mode:                    protocol.Mode(strings.TrimSpace(options.Mode)),
		PrivacyState:            agentplan.PrivacyState(strings.TrimSpace(options.PrivacyState)),
		TaskSummary:             options.TaskSummary,
		CurrentTurnTranscript:   options.CurrentTurnTranscript,
		ExpectedOutput:          options.ExpectedOutput,
		Constraints:             options.Constraints,
		TaskWindowMemory:        parseAgentMemoryItems(options.TaskMemory, agentplan.MemorySourceTaskWindow),
		NonSensitivePreferences: parseAgentMemoryItems(options.PreferenceMemory, agentplan.MemorySourceUserPreference),
		AgentBinding: agentplan.AgentIOBinding{
			Enabled:            options.AgentEnabled,
			Name:               options.AgentName,
			EndpointConfigured: options.EndpointConfigured,
		},
	})
}

func buildAgentIOSmokeReport(ctx context.Context, options agentIOOptions) agentIOSmokeReport {
	configured := options.EndpointConfigured && strings.TrimSpace(options.EndpointURL) != ""
	planOptions := options
	if !configured {
		planOptions.EndpointConfigured = true
	}
	plan := buildAgentPlanFromOptions(planOptions)
	report := agentIOSmokeReport{
		SchemaVersion:         "a21.agent_io_smoke.v1",
		GeneratedAtMS:         time.Now().UnixMilli(),
		AgentProfile:          safeAgentProfile(options.AgentProfile),
		ExternalAgentName:     planAgentName(plan),
		Status:                "ready",
		Configured:            configured,
		Executed:              false,
		Endpoint:              safeEndpointLabel(options.EndpointURL),
		Method:                "POST",
		Mode:                  plan.Mode,
		PrivacyState:          plan.PrivacyState,
		ExecutionPath:         plan.ExecutionPath,
		MemoryPolicy:          plan.MemoryPolicy,
		AgentIOAllowed:        plan.AgentIOAllowed,
		PlannerFindings:       append([]agentplan.Finding(nil), plan.Findings...),
		PlanMarkers:           append([]string(nil), plan.Markers...),
		Redaction:             defaultAgentIOSmokeRedaction(),
		RequestSummaryLength:  safePlanTaskSummaryLength(plan),
		TaskWindowMemoryCount: safePlanMemoryCount(plan),
		PreferenceCount:       safePlanPreferenceCount(plan),
		Boundary:              safePlanBoundary(plan),
	}
	if !plan.AgentIOAllowed || plan.AgentIO == nil {
		report.Status = "failed"
		report.Findings = append(report.Findings, agentIOSmokeFinding{Code: "agent_io_blocked", Message: "Planner did not allow Agent I/O for this mode/privacy/binding"})
		return report
	}
	if !report.Configured {
		report.Status = "needs_endpoint"
		report.Findings = append(report.Findings, agentIOSmokeFinding{Code: "agent_io_endpoint_missing", Message: "Set A21_HERMES_AGENT_URL or pass --endpoint-url before --execute"})
		if options.Execute {
			report.Status = "failed"
		}
		return report
	}
	if err := validateAgentIOEndpoint(options.EndpointURL); err != nil {
		report.Status = "failed"
		report.Findings = append(report.Findings, agentIOSmokeFinding{Code: "agent_io_endpoint_invalid", Message: err.Error()})
		return report
	}
	if !options.Execute {
		return report
	}
	report.Executed = true
	execution := executeAgentIO(ctx, options, plan)
	report.Status = execution.Status
	report.ResponseStatusCode = execution.StatusCode
	report.ResponseContentType = execution.ContentType
	report.ResponseDurationMS = execution.DurationMS
	report.ResponseTextLength = execution.TextLength
	report.ResponseEventCount = execution.EventCount
	report.Findings = append(report.Findings, execution.Findings...)
	return report
}

type agentIOExecutionResult struct {
	Status      string
	StatusCode  int
	ContentType string
	DurationMS  float64
	TextLength  int
	EventCount  int
	Findings    []agentIOSmokeFinding
}

func executeAgentIO(ctx context.Context, options agentIOOptions, plan agentplan.Plan) agentIOExecutionResult {
	timeout := time.Duration(options.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	requestURL := agentIOExecutionURL(options.EndpointURL)
	body := agentIOExecutionPayload(requestURL, options, plan)
	data, err := json.Marshal(body)
	if err != nil {
		return agentIOExecutionResult{Status: "failed", Findings: []agentIOSmokeFinding{{Code: "agent_io_request_encode_failed", Message: "Agent I/O request could not be encoded"}}}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(data))
	if err != nil {
		return agentIOExecutionResult{Status: "failed", Findings: []agentIOSmokeFinding{{Code: "agent_io_request_invalid", Message: "Agent I/O request could not be built"}}}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream, text/plain")
	request.Header.Set("User-Agent", "a21-agent-io-smoke/0.1")
	if key := strings.TrimSpace(options.AgentKey); key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	client := *a21DirectHTTPClient(timeout)
	start := time.Now()
	response, err := client.Do(request)
	duration := time.Since(start)
	if err != nil {
		return agentIOExecutionResult{
			Status:     "failed",
			DurationMS: float64(duration.Microseconds()) / 1000,
			Findings:   []agentIOSmokeFinding{{Code: "agent_io_execute_failed", Message: "Agent I/O endpoint did not return a successful response"}},
		}
	}
	defer response.Body.Close()
	contentType := response.Header.Get("Content-Type")
	textLength, eventCount, readErr := readAgentIOResponseEvidence(response.Body, contentType)
	result := agentIOExecutionResult{
		StatusCode:  response.StatusCode,
		ContentType: safeContentType(contentType),
		DurationMS:  float64(duration.Microseconds()) / 1000,
		TextLength:  textLength,
		EventCount:  eventCount,
	}
	if readErr != nil {
		result.Status = "failed"
		result.Findings = append(result.Findings, agentIOSmokeFinding{Code: "agent_io_response_read_failed", Message: "Agent I/O response could not be read"})
		return result
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		result.Status = "failed"
		result.Findings = append(result.Findings, agentIOSmokeFinding{Code: "agent_io_status_failed", Message: "Agent I/O endpoint returned a non-2xx status"})
		return result
	}
	result.Status = "passed"
	return result
}

func readAgentIOResponseEvidence(body io.Reader, contentType string) (int, int, error) {
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") {
		return readAgentIOSSEEvidence(body)
	}
	limited, err := io.ReadAll(io.LimitReader(body, 256*1024))
	return len([]rune(string(limited))), 0, err
}

func readAgentIOSSEEvidence(body io.Reader) (int, int, error) {
	reader := bufio.NewReader(io.LimitReader(body, 256*1024))
	textLength := 0
	eventCount := 0
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			textLength += len([]rune(line))
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "data:") {
				eventCount++
			}
			if eventCount > 0 && (trimmed == "" || strings.Contains(trimmed, "[DONE]")) {
				return textLength, eventCount, nil
			}
		}
		if err != nil {
			if err == io.EOF && eventCount > 0 {
				return textLength, eventCount, nil
			}
			return textLength, eventCount, err
		}
	}
}

func agentIOExecutionURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return rawURL
	}
	cleanPath := strings.TrimRight(parsed.EscapedPath(), "/")
	if cleanPath == "/v1" {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/chat/completions"
		parsed.RawPath = ""
	}
	return parsed.String()
}

func agentIOExecutionPayload(requestURL string, options agentIOOptions, plan agentplan.Plan) map[string]any {
	if isOpenAICompatibleAgentIOURL(requestURL) {
		return map[string]any{
			"model":  "hermes-agent",
			"stream": true,
			"messages": []map[string]string{
				{"role": "system", "content": "You are an external draft agent behind A21 Agent I/O. Return concise draft text only. A21 owns final persona, memory, privacy, and user-facing wording."},
				{"role": "user", "content": agentIOPrompt(plan)},
			},
			"metadata": map[string]string{
				"schema_version": "a21.agent_io.task.v1",
				"agent_profile":  safeAgentProfile(options.AgentProfile),
				"trace_id":       plan.TraceID,
				"session_id":     plan.SessionID,
			},
		}
	}
	return map[string]any{
		"schema_version":   "a21.agent_io.task.v1",
		"agent_profile":    safeAgentProfile(options.AgentProfile),
		"agent_io_message": plan.AgentIO,
	}
}

func isOpenAICompatibleAgentIOURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	path := strings.TrimRight(strings.ToLower(parsed.EscapedPath()), "/")
	return strings.HasSuffix(path, "/v1/chat/completions")
}

func agentIOPrompt(plan agentplan.Plan) string {
	if plan.AgentIO == nil {
		return "Draft an A21-safe response for the current task."
	}
	var builder strings.Builder
	builder.WriteString("Task:\n")
	builder.WriteString(plan.AgentIO.TaskSummary)
	builder.WriteString("\n\nExpected output:\n")
	builder.WriteString(plan.AgentIO.ExpectedOutput)
	if len(plan.AgentIO.TaskWindowMemory) > 0 {
		builder.WriteString("\n\nTask-window memory:\n")
		for _, item := range plan.AgentIO.TaskWindowMemory {
			builder.WriteString("- ")
			builder.WriteString(item.Text)
			builder.WriteString("\n")
		}
	}
	if len(plan.AgentIO.Preferences) > 0 {
		builder.WriteString("\nPreferences:\n")
		for _, item := range plan.AgentIO.Preferences {
			builder.WriteString("- ")
			builder.WriteString(item.Text)
			builder.WriteString("\n")
		}
	}
	if len(plan.AgentIO.Constraints) > 0 {
		builder.WriteString("\nConstraints:\n")
		for _, item := range plan.AgentIO.Constraints {
			builder.WriteString("- ")
			builder.WriteString(item)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func parseAgentMemoryItems(values []string, source agentplan.MemorySource) []agentplan.MemoryItem {
	var out []agentplan.MemoryItem
	for idx, raw := range values {
		clean := strings.TrimSpace(raw)
		if clean == "" {
			continue
		}
		id := fmt.Sprintf("%s-%d", source, idx+1)
		text := clean
		if before, after, ok := strings.Cut(clean, ":"); ok {
			if strings.TrimSpace(before) != "" {
				id = strings.TrimSpace(before)
			}
			text = strings.TrimSpace(after)
		}
		out = append(out, agentplan.MemoryItem{ID: id, Source: source, Text: text})
	}
	return out
}

func validateAgentIOEndpoint(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("endpoint must be an absolute http or https URL")
	}
	if parsed.User != nil {
		return fmt.Errorf("endpoint URL credentials are not allowed")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("endpoint scheme must be http or https")
	}
	path := strings.ToLower(parsed.EscapedPath())
	if strings.Contains(path, "/v1/devices/control") {
		return fmt.Errorf("endpoint cannot target device control paths")
	}
	return nil
}

func safeEndpointLabel(rawURL string) string {
	if strings.TrimSpace(rawURL) == "" {
		return ""
	}
	return productSurfaceLabel(rawURL, "")
}

func safeAgentProfile(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "hermes_agent", "mimo_agent":
		return value
	case "":
		return "unknown_agent"
	default:
		if strings.Contains(value, "x21") || strings.Contains(value, "v21") {
			return "invalid_legacy_agent"
		}
		return "custom_agent"
	}
}

func defaultAgentDisplayName(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "mimo_agent":
		return "Mimo"
	case "hermes_agent":
		return "Hermes"
	default:
		return "External Agent"
	}
}

func defaultAgentIOSmokeRedaction() agentIOSmokeRedaction {
	return agentIOSmokeRedaction{
		TaskTextStored:           false,
		MemoryTextStored:         false,
		ExternalAgentTextStored:  false,
		FullEndpointURLStored:    false,
		CredentialsStored:        false,
		ProviderEnvValuesStored:  false,
		V21EvidenceBodyStored:    false,
		FirmwareOrDeviceCommands: false,
	}
}

func safeContentType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	if strings.Contains(value, "text/event-stream") {
		return "text/event-stream"
	}
	if strings.Contains(value, "application/json") {
		return "application/json"
	}
	if strings.Contains(value, "text/plain") {
		return "text/plain"
	}
	return "other"
}

func planAgentName(plan agentplan.Plan) string {
	if plan.AgentIO == nil {
		return ""
	}
	return plan.AgentIO.ExternalAgentName
}

func safePlanTaskSummaryLength(plan agentplan.Plan) int {
	if plan.AgentIO == nil {
		return 0
	}
	return len([]rune(plan.AgentIO.TaskSummary))
}

func safePlanMemoryCount(plan agentplan.Plan) int {
	if plan.AgentIO == nil {
		return 0
	}
	return len(plan.AgentIO.TaskWindowMemory)
}

func safePlanPreferenceCount(plan agentplan.Plan) int {
	if plan.AgentIO == nil {
		return 0
	}
	return len(plan.AgentIO.Preferences)
}

func safePlanBoundary(plan agentplan.Plan) []string {
	if plan.AgentIO == nil {
		return nil
	}
	return append([]string(nil), plan.AgentIO.Boundary...)
}

func writeAgentPlanReport(outputDir string, report agentPlanCLIReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-agent-plan-"+agentIOReportStamp()+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONAgentPlan(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeAgentIOSmokeReport(outputDir string, report agentIOSmokeReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-agent-io-smoke-"+agentIOReportStamp()+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONAgentIOSmoke(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONAgentPlan(writer io.Writer, report agentPlanCLIReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONAgentIOSmoke(writer io.Writer, report agentIOSmokeReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func agentIOReportStamp() string {
	now := time.Now()
	return fmt.Sprintf("%s-%09d", now.Format("20060102-150405"), now.Nanosecond())
}
