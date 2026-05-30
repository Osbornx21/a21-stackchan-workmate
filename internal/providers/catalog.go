package providers

import "strings"

type ProviderFamily string

const (
	ProviderFamilyMock       ProviderFamily = "mock"
	ProviderFamilyTextStream ProviderFamily = "text_stream"
)

const DeepSeekDefaultModel = "deepseek-v4-flash"

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
	}
}

func ProviderProfilesFromEnv(env []string) ([]ProviderProfile, []ProviderCatalogFinding) {
	return BuiltinProviderProfiles(), nil
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
	primaryKnown := knownProviderInProfiles(profiles, primary)
	safePrimary := primary
	if containsLegacyProviderIdentity(primary) {
		safePrimary = "invalid_legacy_provider"
	} else if containsBlockedProviderIdentity(primary) {
		safePrimary = "blocked_provider"
	} else if !primaryKnown {
		safePrimary = "unknown_provider"
	}
	report := ProviderCatalogReport{Primary: safePrimary, Findings: append([]ProviderCatalogFinding(nil), findings...)}
	for _, profile := range profiles {
		readiness := ProviderReadiness{
			Name:          profile.Name,
			Label:         profile.Label,
			Family:        string(profile.Family),
			Protocol:      profile.Protocol,
			Selected:      primaryKnown && profile.Name == primary,
			Realtime:      profile.Family == ProviderFamilyMock,
			RouteEligible: profile.RouteEligible,
			Capabilities:  append([]string(nil), profile.Capabilities...),
			RequiredEnv:   providerProfileRequiredEnv(profile),
		}
		for _, name := range readiness.RequiredEnv {
			if envMap[name] {
				readiness.PresentEnv = append(readiness.PresentEnv, name)
			} else {
				readiness.MissingEnv = append(readiness.MissingEnv, name)
			}
		}
		readiness.Configured = len(readiness.MissingEnv) == 0
		report.Providers = append(report.Providers, readiness)
	}
	if !primaryKnown {
		code := "provider_unknown"
		message := "A21 provider primary is not in the P0 provider registry"
		if containsLegacyProviderIdentity(primary) {
			code = "provider_legacy_identity"
			message = "A21 provider primary contains a forbidden legacy identity"
		} else if containsBlockedProviderIdentity(primary) {
			code = "provider_blocked"
			message = "A21 provider primary is blocked by project policy"
		}
		report.Findings = append(report.Findings, ProviderCatalogFinding{Code: code, Message: message, Detail: "A21_PROVIDER_PRIMARY"})
	}
	return report
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

func knownProvider(name string) bool {
	profiles, _ := ProviderProfilesFromEnv(nil)
	return knownProviderInProfiles(profiles, name)
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
