package providers

import (
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
	Provider     string                   `json:"provider"`
	Protocol     string                   `json:"protocol"`
	Status       ProviderSmokeStatus      `json:"status"`
	Configured   bool                     `json:"configured"`
	Executed     bool                     `json:"executed"`
	HTTPStatus   int                      `json:"http_status,omitempty"`
	DurationMS   float64                  `json:"duration_ms,omitempty"`
	NetworkMode  NetworkMode              `json:"network_mode"`
	EndpointHost string                   `json:"endpoint_host,omitempty"`
	BaseURLEnv   string                   `json:"base_url_env,omitempty"`
	APIKeyEnv    string                   `json:"api_key_env,omitempty"`
	ModelEnv     string                   `json:"model_env,omitempty"`
	MissingEnv   []string                 `json:"missing_env,omitempty"`
	ReportPath   string                   `json:"report_path,omitempty"`
	Detail       string                   `json:"detail,omitempty"`
	Findings     []ProviderCatalogFinding `json:"findings,omitempty"`
}

type providerSmokeSpec struct {
	Name           string
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
	{Name: "mock", Protocol: "mock", Executable: false},
	{
		Name:           "deepseek",
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
		Protocol:   "websocket_realtime",
		APIKeyEnv:  "A21_OPENAI_API_KEY",
		ModelEnv:   "A21_OPENAI_REALTIME_MODEL",
		Executable: false,
	},
	{
		Name:       "doubao_realtime",
		Protocol:   "websocket_realtime",
		APIKeyEnv:  "A21_DOUBAO_API_KEY",
		ModelEnv:   "A21_DOUBAO_REALTIME_MODEL",
		Executable: false,
	},
	{
		Name:        "doubao_tts_realtime",
		Protocol:    "websocket_realtime",
		APIKeyEnv:   "A21_DOUBAO_API_KEY",
		ModelEnv:    "A21_DOUBAO_TTS_MODEL",
		RequiredEnv: []string{"A21_DOUBAO_TTS_VOICE"},
		Executable:  false,
	},
}

func ProviderSmokeFromEnv(ctx context.Context, env []string, providerName string, execute bool, client *http.Client) ProviderSmokeReport {
	policy, network := NetworkPolicyFromEnv(env)
	rawProvider := strings.TrimSpace(providerName)
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
		NetworkMode: network.Mode,
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
	if !execute {
		report.Status = ProviderSmokeReady
		report.Detail = "provider smoke is configured; pass --execute to perform a network call"
		return report
	}
	if client == nil {
		var err error
		client, err = NewProviderHTTPClient(policy)
		if err != nil {
			report.Status = ProviderSmokeFailed
			report.Detail = redactProviderSmokeDetail(err.Error())
			return report
		}
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
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	report.HTTPStatus = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		report.Status = ProviderSmokePassed
		report.Detail = "provider smoke request succeeded"
		return report
	}
	report.Status = ProviderSmokeFailed
	report.Detail = fmt.Sprintf("provider smoke HTTP status %d", resp.StatusCode)
	return report
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
