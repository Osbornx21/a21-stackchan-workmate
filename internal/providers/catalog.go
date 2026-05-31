package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ProviderFamily string

const (
	ProviderFamilyMock          ProviderFamily = "mock"
	ProviderFamilyTextStream    ProviderFamily = "text_stream"
	ProviderFamilyVoiceRealtime ProviderFamily = "voice_realtime"
	ProviderFamilyVoiceHybrid   ProviderFamily = "voice_hybrid"
	ProviderFamilyAgentTask     ProviderFamily = "agent_task"
	ProviderFamilyLocalAudio    ProviderFamily = "local_audio"
)

const DeepSeekDefaultModel = "deepseek-chat"

type ProviderProfile struct {
	Name           string         `json:"name"`
	Label          string         `json:"label"`
	Family         ProviderFamily `json:"family"`
	Protocol       string         `json:"protocol"`
	Capabilities   []string       `json:"capabilities,omitempty"`
	RequiredEnv    []string       `json:"required_env,omitempty"`
	APIKeyEnv      string         `json:"api_key_env,omitempty"`
	ModelEnv       string         `json:"model_env,omitempty"`
	DefaultModel   string         `json:"default_model,omitempty"`
	BaseURLEnv     string         `json:"base_url_env,omitempty"`
	DefaultBaseURL string         `json:"default_base_url,omitempty"`
	EndpointPath   string         `json:"endpoint_path,omitempty"`
	RouteEligible  bool           `json:"route_eligible"`
}

type ProviderCatalogReport struct {
	Primary   string                   `json:"primary"`
	Providers []ProviderReadiness      `json:"providers"`
	Findings  []ProviderCatalogFinding `json:"findings,omitempty"`
}

type ProviderReadiness struct {
	Name          string   `json:"name"`
	Label         string   `json:"label"`
	Family        string   `json:"family,omitempty"`
	Protocol      string   `json:"protocol,omitempty"`
	Selected      bool     `json:"selected"`
	Configured    bool     `json:"configured"`
	Realtime      bool     `json:"realtime"`
	RouteEligible bool     `json:"route_eligible,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
	RequiredEnv   []string `json:"required_env,omitempty"`
	PresentEnv    []string `json:"present_env,omitempty"`
	MissingEnv    []string `json:"missing_env,omitempty"`
}

type ProviderCatalogFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func BuiltinProviderProfiles() []ProviderProfile {
	return []ProviderProfile{
		{
			Name:          "mock",
			Label:         "A21 deterministic mock voice",
			Family:        ProviderFamilyMock,
			Protocol:      "mock",
			Capabilities:  []string{"voice", "mock", "realtime", "barge_in"},
			RouteEligible: true,
		},
		{
			Name:           "deepseek",
			Label:          "DeepSeek text stream",
			Family:         ProviderFamilyTextStream,
			Protocol:       "openai_chat_completions",
			Capabilities:   []string{"llm", "streaming_text", "text_stream", "professional_reasoning"},
			APIKeyEnv:      "A21_LAB_DEEPSEEK_API_KEY",
			ModelEnv:       "A21_DEEPSEEK_MODEL",
			DefaultModel:   DeepSeekDefaultModel,
			BaseURLEnv:     "A21_DEEPSEEK_BASE_URL",
			DefaultBaseURL: "https://api.deepseek.com",
			EndpointPath:   "/chat/completions",
			RouteEligible:  true,
		},
		{
			Name:           "siliconflow",
			Label:          "SiliconFlow text stream",
			Family:         ProviderFamilyTextStream,
			Protocol:       "openai_chat_completions",
			Capabilities:   []string{"llm", "streaming_text", "text_stream", "mainland_latency_candidate"},
			APIKeyEnv:      "A21_LAB_SILICONFLOW_API_KEY",
			ModelEnv:       "A21_SILICONFLOW_MODEL",
			BaseURLEnv:     "A21_SILICONFLOW_BASE_URL",
			DefaultBaseURL: "https://api.siliconflow.cn/v1",
			EndpointPath:   "/chat/completions",
		},
		{
			Name:           "stepfun",
			Label:          "StepFun text stream",
			Family:         ProviderFamilyTextStream,
			Protocol:       "openai_chat_completions",
			Capabilities:   []string{"llm", "streaming_text", "text_stream", "mainland_latency_candidate"},
			APIKeyEnv:      "A21_LAB_STEPFUN_API_KEY",
			ModelEnv:       "A21_STEPFUN_MODEL",
			BaseURLEnv:     "A21_STEPFUN_BASE_URL",
			DefaultBaseURL: "https://api.stepfun.com/v1",
			EndpointPath:   "/chat/completions",
		},
		{
			Name:           "bailian_dashscope",
			Label:          "Alibaba Bailian DashScope text stream",
			Family:         ProviderFamilyTextStream,
			Protocol:       "openai_chat_completions",
			Capabilities:   []string{"llm", "streaming_text", "text_stream", "dashscope_candidate"},
			APIKeyEnv:      "A21_DASHSCOPE_API_KEY",
			ModelEnv:       "A21_DASHSCOPE_MODEL",
			BaseURLEnv:     "A21_DASHSCOPE_BASE_URL",
			DefaultBaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
			EndpointPath:   "/chat/completions",
		},
		{
			Name:           "moonshot",
			Label:          "Moonshot text stream",
			Family:         ProviderFamilyTextStream,
			Protocol:       "openai_chat_completions",
			Capabilities:   []string{"llm", "streaming_text", "text_stream", "long_context_candidate"},
			APIKeyEnv:      "A21_LAB_MOONSHOT_API_KEY",
			ModelEnv:       "A21_MOONSHOT_MODEL",
			BaseURLEnv:     "A21_MOONSHOT_BASE_URL",
			DefaultBaseURL: "https://api.moonshot.cn/v1",
			EndpointPath:   "/chat/completions",
		},
		{
			Name:           "volcengine_ark",
			Label:          "Volcengine Ark text stream",
			Family:         ProviderFamilyTextStream,
			Protocol:       "openai_chat_completions",
			Capabilities:   []string{"llm", "streaming_text", "text_stream", "cold_start_candidate"},
			APIKeyEnv:      "A21_LAB_VOLCENGINE_ARK_API_KEY",
			ModelEnv:       "A21_VOLCENGINE_ARK_MODEL",
			BaseURLEnv:     "A21_VOLCENGINE_ARK_BASE_URL",
			DefaultBaseURL: "https://ark.cn-beijing.volces.com/api/v3",
			EndpointPath:   "/chat/completions",
		},
		{
			Name:          "local_ollama",
			Label:         "Local Ollama text stream",
			Family:        ProviderFamilyTextStream,
			Protocol:      "ollama_chat",
			Capabilities:  []string{"llm", "local", "text_stream", "local_fallback"},
			ModelEnv:      "A21_LOCAL_OLLAMA_MODEL",
			BaseURLEnv:    "A21_LOCAL_OLLAMA_BASE_URL",
			RequiredEnv:   []string{"A21_LOCAL_OLLAMA_BASE_URL", "A21_LOCAL_OLLAMA_MODEL"},
			EndpointPath:  "/api/chat",
			RouteEligible: true,
		},
		{
			Name:         "local_vllm",
			Label:        "Local vLLM text stream",
			Family:       ProviderFamilyTextStream,
			Protocol:     "openai_chat_completions",
			Capabilities: []string{"llm", "local", "text_stream", "local_fallback"},
			ModelEnv:     "A21_LOCAL_VLLM_MODEL",
			BaseURLEnv:   "A21_LOCAL_VLLM_BASE_URL",
			RequiredEnv:  []string{"A21_LOCAL_VLLM_BASE_URL", "A21_LOCAL_VLLM_MODEL"},
			EndpointPath: "/chat/completions",
		},
		{
			Name:           "openai_realtime",
			Label:          "OpenAI realtime voice",
			Family:         ProviderFamilyVoiceRealtime,
			Protocol:       "openai_realtime_websocket",
			Capabilities:   []string{"voice", "realtime", "speech_to_speech", "barge_in"},
			APIKeyEnv:      "A21_OPENAI_API_KEY",
			ModelEnv:       "A21_OPENAI_REALTIME_MODEL",
			BaseURLEnv:     "A21_OPENAI_REALTIME_URL",
			DefaultBaseURL: openAIRealtimeDefaultURL,
		},
		{
			Name:         "doubao_realtime",
			Label:        "Doubao realtime speech-to-speech",
			Family:       ProviderFamilyVoiceRealtime,
			Protocol:     "doubao_realtime_s2s_websocket",
			Capabilities: []string{"voice", "realtime", "speech_to_speech", "barge_in"},
			APIKeyEnv:    "A21_DOUBAO_API_KEY",
			ModelEnv:     "A21_DOUBAO_REALTIME_MODEL",
			RequiredEnv:  []string{"A21_DOUBAO_APP_ID", "A21_DOUBAO_RESOURCE_ID"},
			BaseURLEnv:   "A21_DOUBAO_REALTIME_URL",
		},
		{
			Name:         "doubao_tts_realtime",
			Label:        "Doubao realtime TTS",
			Family:       ProviderFamilyVoiceHybrid,
			Protocol:     "doubao_realtime_tts_websocket",
			Capabilities: []string{"voice", "tts", "realtime", "voice_hybrid"},
			APIKeyEnv:    "A21_DOUBAO_API_KEY",
			ModelEnv:     "A21_DOUBAO_TTS_MODEL",
			RequiredEnv:  []string{"A21_DOUBAO_TTS_VOICE"},
			BaseURLEnv:   "A21_DOUBAO_TTS_REALTIME_URL",
		},
		{
			Name:         "hermes_agent",
			Label:        "Hermes agent task bridge",
			Family:       ProviderFamilyAgentTask,
			Protocol:     "agent_task_bridge",
			Capabilities: []string{"agent_task", "background_task", "tool_use"},
			RequiredEnv:  []string{"A21_AGENT_PROVIDER_PRIMARY"},
		},
		{
			Name:         "mimo_agent",
			Label:        "Mimo agent task bridge",
			Family:       ProviderFamilyAgentTask,
			Protocol:     "agent_task_bridge",
			Capabilities: []string{"agent_task", "background_task", "co_creation"},
			RequiredEnv:  []string{"A21_AGENT_PROVIDER_PRIMARY"},
		},
	}
}

func ProviderProfilesFromEnv(env []string) ([]ProviderProfile, []ProviderCatalogFinding) {
	profiles := BuiltinProviderProfiles()
	loaded, findings := loadProviderProfilesFromEnv(env, profiles)
	profiles = append(profiles, loaded...)
	return profiles, findings
}

func ProviderProfileByNameFromEnv(env []string, name string) (ProviderProfile, []ProviderCatalogFinding, bool) {
	profiles, findings := ProviderProfilesFromEnv(env)
	profile, ok := providerProfileByName(profiles, name)
	return profile, findings, ok
}

func ProviderCatalogFromEnv(env []string) ProviderCatalogReport {
	envMap := envNameSet(env)
	profiles, findings := ProviderProfilesFromEnv(env)
	rawPrimary := strings.TrimSpace(envValue(env, "A21_PROVIDER_PRIMARY"))
	if rawPrimary == "" {
		rawPrimary = "mock"
	}
	primary := strings.ToLower(rawPrimary)
	primaryProfile, primaryKnown := providerProfileByName(profiles, primary)
	primaryIsAgentTask := primaryKnown && primaryProfile.Family == ProviderFamilyAgentTask
	primarySelectable := primaryKnown && !primaryIsAgentTask
	safePrimary := primary
	if containsLegacyProviderIdentity(primary) {
		safePrimary = "invalid_legacy_provider"
	} else if containsBlockedProviderIdentity(primary) {
		safePrimary = "blocked_provider"
	} else if primaryIsAgentTask {
		safePrimary = "invalid_agent_task_primary"
	} else if !primaryKnown {
		safePrimary = "unknown_provider"
	}
	report := ProviderCatalogReport{Primary: safePrimary, Findings: append([]ProviderCatalogFinding(nil), findings...)}
	for _, profile := range profiles {
		requiredEnv, presentEnv, missingEnv := providerProfileReadinessEnv(env, envMap, profile)
		readiness := ProviderReadiness{
			Name:          profile.Name,
			Label:         profile.Label,
			Family:        string(profile.Family),
			Protocol:      profile.Protocol,
			Selected:      primarySelectable && profile.Name == primary,
			Realtime:      providerProfileRealtime(profile),
			RouteEligible: profile.RouteEligible,
			Capabilities:  append([]string(nil), profile.Capabilities...),
			RequiredEnv:   requiredEnv,
			PresentEnv:    presentEnv,
			MissingEnv:    missingEnv,
		}
		readiness.Configured = len(readiness.MissingEnv) == 0
		report.Providers = append(report.Providers, readiness)
	}
	if !primarySelectable {
		code := "provider_unknown"
		message := "A21 provider primary is not in the P0 provider registry"
		if containsLegacyProviderIdentity(primary) {
			code = "provider_legacy_identity"
			message = "A21 provider primary contains a forbidden legacy identity"
		} else if containsBlockedProviderIdentity(primary) {
			code = "provider_blocked"
			message = "A21 provider primary is blocked by project policy"
		} else if primaryIsAgentTask {
			code = "provider_agent_task_primary"
			message = "A21_PROVIDER_PRIMARY cannot select agent-task profiles; use A21_AGENT_PROVIDER_PRIMARY"
		}
		report.Findings = append(report.Findings, ProviderCatalogFinding{Code: code, Message: message, Detail: "A21_PROVIDER_PRIMARY"})
	}
	return report
}

func providerProfileRealtime(profile ProviderProfile) bool {
	switch profile.Family {
	case ProviderFamilyMock, ProviderFamilyVoiceRealtime, ProviderFamilyVoiceHybrid:
		return true
	default:
		return false
	}
}

func providerProfileByName(profiles []ProviderProfile, name string) (ProviderProfile, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, profile := range profiles {
		if profile.Name == name {
			return profile, true
		}
	}
	return ProviderProfile{}, false
}

func loadProviderProfilesFromEnv(env []string, builtins []ProviderProfile) ([]ProviderProfile, []ProviderCatalogFinding) {
	rawPath := strings.TrimSpace(envValue(env, "A21_PROVIDER_PROFILES_PATH"))
	if rawPath == "" {
		return nil, nil
	}
	var loaded []ProviderProfile
	var findings []ProviderCatalogFinding
	for _, path := range providerProfilePaths(rawPath) {
		profiles, profileFindings := loadProviderProfilesFile(path, builtins, append(builtins, loaded...))
		findings = append(findings, profileFindings...)
		loaded = append(loaded, profiles...)
	}
	return loaded, findings
}

func providerProfilePaths(rawPath string) []string {
	var paths []string
	for _, path := range filepath.SplitList(rawPath) {
		path = strings.TrimSpace(path)
		if path != "" {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return []string{rawPath}
	}
	return paths
}

func loadProviderProfilesFile(path string, builtins []ProviderProfile, existing []ProviderProfile) ([]ProviderProfile, []ProviderCatalogFinding) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, []ProviderCatalogFinding{providerProfileFinding("provider_profile_load_failed")}
	}
	profiles, err := decodeProviderProfiles(data)
	if err != nil {
		return nil, []ProviderCatalogFinding{providerProfileFinding("provider_profile_invalid_json")}
	}
	var loaded []ProviderProfile
	var findings []ProviderCatalogFinding
	for _, profile := range profiles {
		normalized, finding := validateLoadedProviderProfile(profile, builtins, append(existing, loaded...))
		if finding != nil {
			findings = append(findings, *finding)
			continue
		}
		loaded = append(loaded, normalized)
	}
	return loaded, findings
}

func decodeProviderProfiles(data []byte) ([]ProviderProfile, error) {
	var profiles []ProviderProfile
	if err := decodeJSONStrict(data, &profiles); err == nil {
		return profiles, nil
	}
	var profile ProviderProfile
	if err := decodeJSONStrict(data, &profile); err != nil {
		return nil, err
	}
	return []ProviderProfile{profile}, nil
}

func decodeJSONStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("extra JSON content")
	}
	return nil
}

var loadedProviderNamePattern = regexp.MustCompile(`^a21_[a-z0-9_]+$`)
var providerEnvNamePattern = regexp.MustCompile(`^A21_[A-Z0-9_]+$`)

func validateLoadedProviderProfile(profile ProviderProfile, builtins []ProviderProfile, existing []ProviderProfile) (ProviderProfile, *ProviderCatalogFinding) {
	profile.Name = strings.ToLower(strings.TrimSpace(profile.Name))
	profile.Label = strings.TrimSpace(profile.Label)
	profile.Protocol = strings.TrimSpace(profile.Protocol)
	profile.APIKeyEnv = strings.TrimSpace(profile.APIKeyEnv)
	profile.ModelEnv = strings.TrimSpace(profile.ModelEnv)
	profile.BaseURLEnv = strings.TrimSpace(profile.BaseURLEnv)
	profile.DefaultBaseURL = strings.TrimSpace(profile.DefaultBaseURL)
	profile.EndpointPath = strings.TrimSpace(profile.EndpointPath)
	profile.Capabilities = normalizeProviderStringSlice(profile.Capabilities)
	profile.RequiredEnv = normalizeProviderStringSlice(profile.RequiredEnv)

	switch {
	case profile.Name == "":
		return ProviderProfile{}, providerProfileFindingPtr("provider_profile_invalid_name")
	case containsBlockedProviderIdentity(profile.Name):
		return ProviderProfile{}, providerProfileFindingPtr("provider_profile_blocked")
	case containsLegacyProviderIdentity(profile.Name):
		return ProviderProfile{}, providerProfileFindingPtr("provider_profile_legacy_identity")
	case !loadedProviderNamePattern.MatchString(profile.Name):
		return ProviderProfile{}, providerProfileFindingPtr("provider_profile_invalid_name")
	case knownProviderInProfiles(existing, profile.Name) || knownProviderInProfiles(builtins, profile.Name):
		return ProviderProfile{}, providerProfileFindingPtr("provider_profile_duplicate")
	case profile.Family != ProviderFamilyTextStream || profile.Protocol != "openai_chat_completions":
		return ProviderProfile{}, providerProfileFindingPtr("provider_profile_unsupported")
	case profile.EndpointPath == "":
		return ProviderProfile{}, providerProfileFindingPtr("provider_profile_invalid_endpoint")
	case strings.Contains(profile.EndpointPath, "://") || !strings.HasPrefix(profile.EndpointPath, "/"):
		return ProviderProfile{}, providerProfileFindingPtr("provider_profile_invalid_endpoint")
	case strings.Contains(profile.EndpointPath, "@"):
		return ProviderProfile{}, providerProfileFindingPtr("provider_profile_endpoint_credentials")
	case containsLegacyProviderIdentity(profile.EndpointPath):
		return ProviderProfile{}, providerProfileFindingPtr("provider_profile_legacy_identity")
	}
	if profile.DefaultBaseURL != "" {
		parsed, err := url.Parse(profile.DefaultBaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return ProviderProfile{}, providerProfileFindingPtr("provider_profile_invalid_endpoint")
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return ProviderProfile{}, providerProfileFindingPtr("provider_profile_invalid_endpoint")
		}
		if parsed.User != nil {
			return ProviderProfile{}, providerProfileFindingPtr("provider_profile_endpoint_credentials")
		}
		if containsLegacyProviderIdentity(parsed.Path) {
			return ProviderProfile{}, providerProfileFindingPtr("provider_profile_legacy_identity")
		}
	}
	if profile.BaseURLEnv == "" && profile.DefaultBaseURL == "" {
		return ProviderProfile{}, providerProfileFindingPtr("provider_profile_missing_endpoint")
	}
	for _, name := range append([]string{profile.APIKeyEnv, profile.ModelEnv, profile.BaseURLEnv}, profile.RequiredEnv...) {
		if name == "" {
			continue
		}
		switch {
		case !providerEnvNamePattern.MatchString(name):
			return ProviderProfile{}, providerProfileFindingPtr("provider_profile_invalid_env")
		case containsLegacyProviderIdentity(name):
			return ProviderProfile{}, providerProfileFindingPtr("provider_profile_legacy_identity")
		case containsBlockedProviderIdentity(name):
			return ProviderProfile{}, providerProfileFindingPtr("provider_profile_blocked")
		}
	}
	if profile.Label == "" {
		profile.Label = profile.Name
	}
	return profile, nil
}

func normalizeProviderStringSlice(values []string) []string {
	var normalized []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !stringSliceHas(normalized, value) {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func providerProfileFindingPtr(code string) *ProviderCatalogFinding {
	finding := providerProfileFinding(code)
	return &finding
}

func providerProfileFinding(code string) ProviderCatalogFinding {
	return ProviderCatalogFinding{
		Code:    code,
		Message: providerProfileFindingMessage(code),
		Detail:  "A21_PROVIDER_PROFILES_PATH",
	}
}

func providerProfileFindingMessage(code string) string {
	switch code {
	case "provider_profile_load_failed":
		return "A21 provider profile file could not be loaded"
	case "provider_profile_invalid_json":
		return "A21 provider profile file is not valid JSON"
	case "provider_profile_invalid_name":
		return "A21 provider profile name is invalid"
	case "provider_profile_blocked":
		return "A21 provider profile is blocked by project policy"
	case "provider_profile_legacy_identity":
		return "A21 provider profile contains a forbidden legacy identity"
	case "provider_profile_duplicate":
		return "A21 provider profile duplicates an existing provider"
	case "provider_profile_unsupported":
		return "A21 provider profile uses an unsupported family or protocol"
	case "provider_profile_invalid_endpoint":
		return "A21 provider profile endpoint is invalid"
	case "provider_profile_endpoint_credentials":
		return "A21 provider profile endpoint must not contain credentials"
	case "provider_profile_missing_endpoint":
		return "A21 provider profile must declare a base URL env or default base URL"
	case "provider_profile_invalid_env":
		return "A21 provider profile env names must use the A21_ namespace"
	default:
		return "A21 provider profile was rejected"
	}
}

func knownProviderInProfiles(profiles []ProviderProfile, name string) bool {
	_, ok := providerProfileByName(profiles, name)
	return ok
}

func providerProfileRequiredEnv(profile ProviderProfile) []string {
	var required []string
	for _, name := range []string{profile.APIKeyEnv} {
		if name != "" {
			required = append(required, name)
		}
	}
	if profile.ModelEnv != "" && profile.DefaultModel == "" {
		required = append(required, profile.ModelEnv)
	}
	for _, name := range profile.RequiredEnv {
		name = strings.TrimSpace(name)
		if name != "" && !stringSliceHas(required, name) {
			required = append(required, name)
		}
	}
	return required
}

func providerProfileReadinessEnv(env []string, envMap map[string]bool, profile ProviderProfile) ([]string, []string, []string) {
	required := providerProfileRequiredEnv(profile)
	if profile.Family == ProviderFamilyAgentTask {
		selected := strings.ToLower(strings.TrimSpace(envValue(env, "A21_AGENT_PROVIDER_PRIMARY")))
		if selected == profile.Name {
			return required, append([]string(nil), required...), nil
		}
		return required, nil, append([]string(nil), required...)
	}
	var present []string
	var missing []string
	for _, name := range required {
		if envMap[name] {
			present = append(present, name)
		} else {
			missing = append(missing, name)
		}
	}
	return required, present, missing
}

func knownProvider(name string) bool {
	profiles, _ := ProviderProfilesFromEnv(nil)
	return knownProviderInProfiles(profiles, name)
}

func providerProfileIsAgentTask(name string) bool {
	profiles, _ := ProviderProfilesFromEnv(nil)
	profile, ok := providerProfileByName(profiles, name)
	return ok && profile.Family == ProviderFamilyAgentTask
}

func containsLegacyProviderIdentity(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "x21") || strings.Contains(lower, "v21")
}

func containsBlockedProviderIdentity(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "baidu") || strings.Contains(lower, "huawei")
}

func safeProviderName(value string) string {
	name := strings.ToLower(strings.TrimSpace(value))
	switch {
	case containsLegacyProviderIdentity(name):
		return "invalid_legacy_provider"
	case containsBlockedProviderIdentity(name):
		return "blocked_provider"
	case knownProvider(name):
		return name
	default:
		return "unknown_provider"
	}
}

func stringSliceHas(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func envNameSet(env []string) map[string]bool {
	set := make(map[string]bool)
	for _, entry := range env {
		name, value, _ := strings.Cut(entry, "=")
		if strings.TrimSpace(value) == "" {
			continue
		}
		set[strings.TrimSpace(name)] = true
	}
	return set
}

func envValue(env []string, want string) string {
	for _, entry := range env {
		name, value, _ := strings.Cut(entry, "=")
		if strings.TrimSpace(name) == want {
			return value
		}
	}
	return ""
}
