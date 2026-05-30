package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type ProviderSmokeStatus string

const (
	ProviderSmokeSkipped     ProviderSmokeStatus = "skipped"
	ProviderSmokeReady       ProviderSmokeStatus = "ready"
	ProviderSmokePassed      ProviderSmokeStatus = "passed"
	ProviderSmokeFailed      ProviderSmokeStatus = "failed"
	ProviderSmokeUnsupported ProviderSmokeStatus = "unsupported"
)

type ProviderSmokeReport struct {
	Provider      string                   `json:"provider"`
	Family        string                   `json:"family,omitempty"`
	Protocol      string                   `json:"protocol"`
	Status        ProviderSmokeStatus      `json:"status"`
	Configured    bool                     `json:"configured"`
	Executed      bool                     `json:"executed"`
	Stream        bool                     `json:"stream,omitempty"`
	Repeat        int                      `json:"repeat,omitempty"`
	HTTPStatus    int                      `json:"http_status,omitempty"`
	DurationMS    float64                  `json:"duration_ms,omitempty"`
	Attempts      []ProviderSmokeAttempt   `json:"attempts,omitempty"`
	TimingSummary *ProviderSmokeTiming     `json:"timing_summary,omitempty"`
	Fallback      *ProviderSmokeFallback   `json:"fallback,omitempty"`
	TraceID       string                   `json:"trace_id,omitempty"`
	TraceMarkers  []ProviderSmokeMarker    `json:"trace_markers,omitempty"`
	Metrics       []ProviderSmokeMetric    `json:"metrics,omitempty"`
	NetworkMode   NetworkMode              `json:"network_mode"`
	EndpointHost  string                   `json:"endpoint_host,omitempty"`
	BaseURLEnv    string                   `json:"base_url_env,omitempty"`
	APIKeyEnv     string                   `json:"api_key_env,omitempty"`
	ModelEnv      string                   `json:"model_env,omitempty"`
	MissingEnv    []string                 `json:"missing_env,omitempty"`
	ReportPath    string                   `json:"report_path,omitempty"`
	Detail        string                   `json:"detail,omitempty"`
	Findings      []ProviderCatalogFinding `json:"findings,omitempty"`
}

type ProviderSmokeAttempt struct {
	Index               int     `json:"index"`
	HTTPStatus          int     `json:"http_status,omitempty"`
	FirstByteMS         float64 `json:"first_byte_ms,omitempty"`
	FirstContentMS      float64 `json:"first_content_ms,omitempty"`
	TotalDurationMS     float64 `json:"total_duration_ms,omitempty"`
	ContentDeltaCount   int     `json:"content_delta_count,omitempty"`
	ReasoningDeltaCount int     `json:"reasoning_delta_count,omitempty"`
	Done                bool    `json:"done,omitempty"`
}

type ProviderSmokeTiming struct {
	Repeat             int     `json:"repeat,omitempty"`
	FirstByteP50MS     float64 `json:"first_byte_p50_ms,omitempty"`
	FirstByteP95MS     float64 `json:"first_byte_p95_ms,omitempty"`
	FirstContentP50MS  float64 `json:"first_content_p50_ms,omitempty"`
	FirstContentP95MS  float64 `json:"first_content_p95_ms,omitempty"`
	TotalDurationP50MS float64 `json:"total_duration_p50_ms,omitempty"`
	TotalDurationP95MS float64 `json:"total_duration_p95_ms,omitempty"`
}

type ProviderSmokeFallback struct {
	Activated bool   `json:"activated,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type ProviderSmokeMarker struct {
	Name    string  `json:"name"`
	ValueMS float64 `json:"value_ms,omitempty"`
}

type ProviderSmokeMetric struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type ProviderSmokeOptions struct {
	ProviderName string
	Execute      bool
	Stream       bool
	Repeat       int
	Client       *http.Client
}

type providerSmokeSpec struct {
	Name           string
	Family         ProviderFamily
	Protocol       string
	APIKeyEnv      string
	ModelEnv       string
	BaseURLEnv     string
	RequiredEnv    []string
	DefaultBaseURL string
	EndpointPath   string
	Executable     bool
}

var providerSmokeSpecs = []providerSmokeSpec{
	{Name: "mock", Family: ProviderFamilyMock, Protocol: "mock", Executable: false},
	{
		Name:           "deepseek",
		Family:         ProviderFamilyTextStream,
		Protocol:       "openai_chat_completions",
		APIKeyEnv:      "A21_DEEPSEEK_API_KEY",
		ModelEnv:       "A21_DEEPSEEK_MODEL",
		BaseURLEnv:     "A21_DEEPSEEK_BASE_URL",
		DefaultBaseURL: "https://api.deepseek.com",
		EndpointPath:   "/chat/completions",
		Executable:     true,
	},
	{
		Name:           "bailian_dashscope",
		Family:         ProviderFamilyTextStream,
		Protocol:       "openai_chat_completions",
		APIKeyEnv:      "A21_DASHSCOPE_API_KEY",
		ModelEnv:       "A21_DASHSCOPE_MODEL",
		BaseURLEnv:     "A21_DASHSCOPE_BASE_URL",
		DefaultBaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		EndpointPath:   "/chat/completions",
		Executable:     true,
	},
	{
		Name:       "openai_realtime",
		Family:     ProviderFamilyVoiceRealtime,
		Protocol:   "websocket_realtime",
		APIKeyEnv:  "A21_OPENAI_API_KEY",
		ModelEnv:   "A21_OPENAI_REALTIME_MODEL",
		Executable: false,
	},
	{
		Name:       "doubao_realtime",
		Family:     ProviderFamilyVoiceRealtime,
		Protocol:   "websocket_realtime",
		APIKeyEnv:  "A21_DOUBAO_API_KEY",
		ModelEnv:   "A21_DOUBAO_REALTIME_MODEL",
		Executable: false,
	},
	{
		Name:        "doubao_tts_realtime",
		Family:      ProviderFamilyVoiceRealtime,
		Protocol:    "websocket_realtime",
		APIKeyEnv:   "A21_DOUBAO_API_KEY",
		ModelEnv:    "A21_DOUBAO_TTS_MODEL",
		RequiredEnv: []string{"A21_DOUBAO_TTS_VOICE"},
		Executable:  false,
	},
}

func ProviderSmokeFromEnv(ctx context.Context, env []string, providerName string, execute bool, client *http.Client) ProviderSmokeReport {
	return ProviderSmokeFromEnvWithOptions(ctx, env, ProviderSmokeOptions{
		ProviderName: providerName,
		Execute:      execute,
		Client:       client,
	})
}

func ProviderSmokeFromEnvWithOptions(ctx context.Context, env []string, options ProviderSmokeOptions) ProviderSmokeReport {
	policy, network := NetworkPolicyFromEnv(env)
	rawProvider := strings.TrimSpace(options.ProviderName)
	if rawProvider == "" {
		rawProvider = strings.TrimSpace(envValue(env, "A21_PROVIDER_PRIMARY"))
	}
	if rawProvider == "" {
		rawProvider = "mock"
	}
	provider := strings.ToLower(rawProvider)
	report := ProviderSmokeReport{
		Provider:    safeProviderName(provider),
		Status:      ProviderSmokeFailed,
		Stream:      options.Stream,
		NetworkMode: network.Mode,
	}
	if options.Stream {
		report.Repeat = normalizedProviderSmokeRepeat(options.Repeat)
	}
	if containsLegacyProviderIdentity(provider) {
		report.Findings = append(report.Findings, ProviderCatalogFinding{
			Code:    "provider_legacy_identity",
			Message: "A21 provider smoke target contains a forbidden legacy identity",
			Detail:  "A21_PROVIDER_PRIMARY",
		})
		report.Detail = "provider target rejected"
		return report
	}
	spec, ok := providerSmokeSpecByName(provider)
	if !ok {
		report.Findings = append(report.Findings, ProviderCatalogFinding{
			Code:    "provider_unknown",
			Message: "A21 provider smoke target is not in the provider registry",
			Detail:  "A21_PROVIDER_PRIMARY",
		})
		report.Detail = "provider target is unknown"
		return report
	}
	report.Provider = spec.Name
	report.Family = string(spec.Family)
	report.Protocol = spec.Protocol
	report.APIKeyEnv = spec.APIKeyEnv
	report.ModelEnv = spec.ModelEnv
	report.BaseURLEnv = spec.BaseURLEnv
	report.Configured, report.MissingEnv = providerSmokeConfigured(env, spec)
	if host := providerSmokeEndpointHost(env, spec); host != "" {
		report.EndpointHost = host
	}
	if spec.Name == "mock" {
		report.Status = ProviderSmokePassed
		report.Configured = true
		report.Detail = "deterministic mock provider is available"
		return report
	}
	if !report.Configured {
		report.Status = ProviderSmokeSkipped
		report.Detail = "provider smoke skipped because required env is missing"
		return report
	}
	if !spec.Executable {
		report.Status = ProviderSmokeUnsupported
		report.Detail = "provider smoke execution is not implemented for this realtime protocol yet"
		return report
	}
	if !options.Execute {
		report.Status = ProviderSmokeReady
		report.Detail = "provider smoke is configured; pass --execute to perform a network call"
		return report
	}
	client := options.Client
	if client == nil {
		var err error
		client, err = NewProviderHTTPClient(policy)
		if err != nil {
			report.Status = ProviderSmokeFailed
			report.Detail = redactProviderSmokeDetail(err.Error())
			return report
		}
	}
	if options.Stream {
		return executeOpenAICompatibleStreamingSmoke(ctx, env, spec, report, client)
	}
	return executeOpenAICompatibleSmoke(ctx, env, spec, report, client)
}

func providerSmokeSpecByName(name string) (providerSmokeSpec, bool) {
	for _, spec := range providerSmokeSpecs {
		if spec.Name == name {
			return spec, true
		}
	}
	return providerSmokeSpec{}, false
}

func providerSmokeConfigured(env []string, spec providerSmokeSpec) (bool, []string) {
	var missing []string
	for _, name := range []string{spec.APIKeyEnv, spec.ModelEnv} {
		if name == "" {
			continue
		}
		if strings.TrimSpace(envValue(env, name)) == "" {
			missing = append(missing, name)
		}
	}
	for _, name := range spec.RequiredEnv {
		if strings.TrimSpace(envValue(env, name)) == "" {
			missing = append(missing, name)
		}
	}
	return len(missing) == 0, missing
}

func providerSmokeEndpointHost(env []string, spec providerSmokeSpec) string {
	if spec.DefaultBaseURL == "" {
		return ""
	}
	baseURL := strings.TrimSpace(envValue(env, spec.BaseURLEnv))
	if baseURL == "" {
		baseURL = spec.DefaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "invalid_base_url"
	}
	if parsed.Host == "" {
		return "invalid_base_url"
	}
	return parsed.Host
}

func executeOpenAICompatibleSmoke(ctx context.Context, env []string, spec providerSmokeSpec, report ProviderSmokeReport, client *http.Client) ProviderSmokeReport {
	endpoint, err := providerSmokeEndpoint(env, spec)
	if err != nil {
		report.Status = ProviderSmokeFailed
		report.Detail = redactProviderSmokeDetail(err.Error())
		return report
	}
	body := map[string]any{
		"model": strings.TrimSpace(envValue(env, spec.ModelEnv)),
		"messages": []map[string]string{
			{"role": "user", "content": "A21 provider smoke check. Reply OK."},
		},
		"max_tokens": 8,
		"stream":     false,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		report.Status = ProviderSmokeFailed
		report.Detail = redactProviderSmokeDetail(err.Error())
		return report
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		report.Status = ProviderSmokeFailed
		report.Detail = redactProviderSmokeDetail(err.Error())
		return report
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(envValue(env, spec.APIKeyEnv)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "a21-provider-smoke/0.1")
	report.Executed = true
	start := time.Now()
	resp, err := client.Do(req)
	report.DurationMS = float64(time.Since(start).Microseconds()) / 1000
	if err != nil {
		report.Status = ProviderSmokeFailed
		report.Detail = redactProviderSmokeDetail(err.Error())
		return report
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	report.HTTPStatus = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		report.Status = ProviderSmokePassed
		report.Detail = "provider smoke request succeeded"
		return report
	}
	report.Status = ProviderSmokeFailed
	report.Detail = redactedProviderHTTPError(resp.StatusCode, bodyBytes)
	return report
}

func executeOpenAICompatibleStreamingSmoke(ctx context.Context, env []string, spec providerSmokeSpec, report ProviderSmokeReport, client *http.Client) ProviderSmokeReport {
	repeat := normalizedProviderSmokeRepeat(report.Repeat)
	report.Repeat = repeat
	report.Executed = true
	report.TraceID = fmt.Sprintf("a21-trace-provider-smoke-%d", time.Now().UnixNano())
	for i := 1; i <= repeat; i++ {
		attempt, err := executeOpenAICompatibleStreamingSmokeAttempt(ctx, env, spec, client, i)
		report.Attempts = append(report.Attempts, attempt)
		report.HTTPStatus = attempt.HTTPStatus
		if attempt.TotalDurationMS > 0 {
			report.DurationMS = attempt.TotalDurationMS
		}
		if attempt.FirstByteMS > 0 {
			report.TraceMarkers = append(report.TraceMarkers, ProviderSmokeMarker{Name: "provider_first_byte", ValueMS: attempt.FirstByteMS})
			report.Metrics = append(report.Metrics, ProviderSmokeMetric{Name: "a21_provider_first_byte_ms", Value: attempt.FirstByteMS})
		}
		if attempt.FirstContentMS > 0 {
			report.TraceMarkers = append(report.TraceMarkers, ProviderSmokeMarker{Name: "provider_first_content", ValueMS: attempt.FirstContentMS})
			report.Metrics = append(report.Metrics, ProviderSmokeMetric{Name: "a21_provider_first_content_ms", Value: attempt.FirstContentMS})
		}
		if err != nil {
			report.Status = ProviderSmokeFailed
			report.Detail = redactProviderSmokeDetail(err.Error())
			report.Fallback = &ProviderSmokeFallback{Activated: true, Provider: "mock", Reason: "primary_failed"}
			report.TraceMarkers = append(report.TraceMarkers, ProviderSmokeMarker{Name: "provider_fallback_used"})
			report.Metrics = append(report.Metrics, ProviderSmokeMetric{Name: "a21_provider_fallback_total", Value: 1})
			report.TimingSummary = summarizeProviderSmokeTimings(report.Attempts)
			return report
		}
	}
	report.Executed = true
	report.Status = ProviderSmokePassed
	report.Detail = "provider streaming smoke request succeeded"
	report.TimingSummary = summarizeProviderSmokeTimings(report.Attempts)
	return report
}

func executeOpenAICompatibleStreamingSmokeAttempt(ctx context.Context, env []string, spec providerSmokeSpec, client *http.Client, index int) (ProviderSmokeAttempt, error) {
	attempt := ProviderSmokeAttempt{Index: index}
	endpoint, err := providerSmokeEndpoint(env, spec)
	if err != nil {
		return attempt, err
	}
	body := map[string]any{
		"model": strings.TrimSpace(envValue(env, spec.ModelEnv)),
		"messages": []map[string]string{
			{"role": "user", "content": "A21 provider smoke check. Reply OK."},
		},
		"max_tokens": 8,
		"stream":     true,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return attempt, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return attempt, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(envValue(env, spec.APIKeyEnv)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", "a21-provider-smoke/0.1")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		attempt.TotalDurationMS = elapsedMS(start)
		return attempt, err
	}
	defer resp.Body.Close()
	attempt.HTTPStatus = resp.StatusCode
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		attempt.TotalDurationMS = elapsedMS(start)
		return attempt, fmt.Errorf("%s", redactedProviderHTTPError(resp.StatusCode, bodyBytes))
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		if attempt.FirstByteMS == 0 {
			attempt.FirstByteMS = elapsedMS(start)
		}
		line := scanner.Text()
		events, done, err := parseOpenAICompatibleTextStreamLine(strings.TrimSpace(line))
		if err != nil {
			attempt.TotalDurationMS = elapsedMS(start)
			return attempt, err
		}
		if done {
			attempt.Done = true
			continue
		}
		for _, event := range events {
			if event.Text == "" {
				continue
			}
			if attempt.FirstContentMS == 0 {
				attempt.FirstContentMS = elapsedMS(start)
			}
			switch event.Kind {
			case TextStreamDeltaContent:
				attempt.ContentDeltaCount++
			case TextStreamDeltaReasoning:
				attempt.ReasoningDeltaCount++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		attempt.TotalDurationMS = elapsedMS(start)
		return attempt, err
	}
	attempt.TotalDurationMS = elapsedMS(start)
	return attempt, nil
}

func normalizedProviderSmokeRepeat(repeat int) int {
	if repeat <= 0 {
		return 1
	}
	if repeat > 10 {
		return 10
	}
	return repeat
}

func summarizeProviderSmokeTimings(attempts []ProviderSmokeAttempt) *ProviderSmokeTiming {
	summary := ProviderSmokeTiming{Repeat: len(attempts)}
	var firstByte []float64
	var firstContent []float64
	var total []float64
	for _, attempt := range attempts {
		if attempt.FirstByteMS > 0 {
			firstByte = append(firstByte, attempt.FirstByteMS)
		}
		if attempt.FirstContentMS > 0 {
			firstContent = append(firstContent, attempt.FirstContentMS)
		}
		if attempt.TotalDurationMS > 0 {
			total = append(total, attempt.TotalDurationMS)
		}
	}
	summary.FirstByteP50MS = percentileNearestRank(firstByte, 0.50)
	summary.FirstByteP95MS = percentileNearestRank(firstByte, 0.95)
	summary.FirstContentP50MS = percentileNearestRank(firstContent, 0.50)
	summary.FirstContentP95MS = percentileNearestRank(firstContent, 0.95)
	summary.TotalDurationP50MS = percentileNearestRank(total, 0.50)
	summary.TotalDurationP95MS = percentileNearestRank(total, 0.95)
	return &summary
}

func percentileNearestRank(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	index := int(float64(len(sorted))*percentile+0.999999) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func elapsedMS(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000
}

func providerSmokeEndpoint(env []string, spec providerSmokeSpec) (string, error) {
	baseURL := strings.TrimSpace(envValue(env, spec.BaseURLEnv))
	if baseURL == "" {
		baseURL = spec.DefaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("provider smoke base URL must use http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("provider smoke base URL host is required")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + spec.EndpointPath
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func safeProviderName(name string) string {
	if containsLegacyProviderIdentity(name) {
		return "invalid_legacy_provider"
	}
	if knownProvider(name) {
		return name
	}
	return "unknown_provider"
}

var providerSmokeURLCredentialPattern = regexp.MustCompile(`(https?://)[^/\s"']+@`)

func redactProviderSmokeDetail(text string) string {
	if text == "" {
		return ""
	}
	return providerSmokeURLCredentialPattern.ReplaceAllString(text, "${1}<redacted>@")
}
