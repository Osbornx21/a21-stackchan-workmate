package providers

import "strings"

type ProviderCatalogReport struct {
	Primary   string                   `json:"primary"`
	Providers []ProviderReadiness      `json:"providers"`
	Findings  []ProviderCatalogFinding `json:"findings,omitempty"`
}

type ProviderReadiness struct {
	Name         string   `json:"name"`
	Label        string   `json:"label"`
	Selected     bool     `json:"selected"`
	Configured   bool     `json:"configured"`
	Realtime     bool     `json:"realtime"`
	Capabilities []string `json:"capabilities,omitempty"`
	RequiredEnv  []string `json:"required_env,omitempty"`
	PresentEnv   []string `json:"present_env,omitempty"`
	MissingEnv   []string `json:"missing_env,omitempty"`
}

type ProviderCatalogFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type providerSpec struct {
	Name         string
	Label        string
	Realtime     bool
	Capabilities []string
	RequiredEnv  []string
}

var providerSpecs = []providerSpec{
	{
		Name:         "mock",
		Label:        "A21 deterministic mock voice",
		Realtime:     true,
		Capabilities: []string{"voice", "mock", "realtime", "barge_in"},
	},
	{
		Name:     "doubao_realtime",
		Label:    "Doubao realtime speech-to-speech",
		Realtime: true,
		Capabilities: []string{
			"voice",
			"speech_to_speech",
			"realtime",
			"barge_in",
			"voice_clone",
			"character",
		},
		RequiredEnv: []string{
			"A21_DOUBAO_API_KEY",
			"A21_DOUBAO_APP_ID",
			"A21_DOUBAO_RESOURCE_ID",
			"A21_DOUBAO_REALTIME_MODEL",
		},
	},
	{
		Name:         "openai_realtime",
		Label:        "OpenAI realtime voice",
		Realtime:     true,
		Capabilities: []string{"voice", "speech_to_speech", "realtime", "barge_in", "tool_calling"},
		RequiredEnv:  []string{"A21_OPENAI_API_KEY", "A21_OPENAI_REALTIME_MODEL"},
	},
	{
		Name:         "bailian_dashscope",
		Label:        "Alibaba Bailian/DashScope model studio",
		Realtime:     false,
		Capabilities: []string{"tts", "asr", "llm", "openai_compatible"},
		RequiredEnv:  []string{"A21_DASHSCOPE_API_KEY"},
	},
	{
		Name:         "deepseek",
		Label:        "DeepSeek text reasoning",
		Realtime:     false,
		Capabilities: []string{"llm", "streaming_text", "professional_reasoning"},
		RequiredEnv:  []string{"A21_DEEPSEEK_API_KEY", "A21_DEEPSEEK_MODEL"},
	},
}

func ProviderCatalogFromEnv(env []string) ProviderCatalogReport {
	envMap := envNameSet(env)
	rawPrimary := strings.TrimSpace(envValue(env, "A21_PROVIDER_PRIMARY"))
	if rawPrimary == "" {
		rawPrimary = "mock"
	}
	primary := strings.ToLower(rawPrimary)
	primaryKnown := knownProvider(primary)
	safePrimary := primary
	if containsLegacyProviderIdentity(primary) {
		safePrimary = "invalid_legacy_provider"
	} else if !primaryKnown {
		safePrimary = "unknown_provider"
	}
	report := ProviderCatalogReport{Primary: safePrimary}
	for _, spec := range providerSpecs {
		readiness := ProviderReadiness{
			Name:         spec.Name,
			Label:        spec.Label,
			Selected:     primaryKnown && spec.Name == primary,
			Realtime:     spec.Realtime,
			Capabilities: append([]string(nil), spec.Capabilities...),
			RequiredEnv:  append([]string(nil), spec.RequiredEnv...),
		}
		for _, name := range spec.RequiredEnv {
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
		message := "A21 provider primary is not in the provider registry"
		if containsLegacyProviderIdentity(primary) {
			code = "provider_legacy_identity"
			message = "A21 provider primary contains a forbidden legacy identity"
		}
		report.Findings = append(report.Findings, ProviderCatalogFinding{Code: code, Message: message, Detail: "A21_PROVIDER_PRIMARY"})
	}
	return report
}

func knownProvider(name string) bool {
	for _, spec := range providerSpecs {
		if spec.Name == name {
			return true
		}
	}
	return false
}

func containsLegacyProviderIdentity(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "x21") || strings.Contains(lower, "v21")
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
