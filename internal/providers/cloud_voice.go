package providers

import "strings"

const (
	CloudVoiceCatalogSchemaVersion = "a21.cloud_voice_profiles.v1"
	CloudVoiceDefaultProfile       = "a21_doubao_tts_realtime"

	CloudVoiceStatusCatalogOnly = "catalog_only"
	CloudVoiceStatusStaticReady = "static_ready"
)

type CloudVoiceCatalogReport struct {
	SchemaVersion             string                            `json:"schema_version"`
	SelectedCloudVoiceProfile string                            `json:"selected_cloud_voice_profile"`
	Profiles                  []CloudVoiceProfileReadiness      `json:"profiles"`
	Findings                  []CloudVoiceProfileCatalogFinding `json:"findings,omitempty"`
}

type CloudVoiceProfileReadiness struct {
	ID            string   `json:"id"`
	Label         string   `json:"label"`
	Vendor        string   `json:"vendor"`
	Family        string   `json:"family"`
	Protocol      string   `json:"protocol"`
	Lane          string   `json:"lane"`
	Status        string   `json:"status"`
	Selected      bool     `json:"selected"`
	Default       bool     `json:"default,omitempty"`
	Configured    bool     `json:"configured"`
	Realtime      bool     `json:"realtime"`
	VoiceClone    bool     `json:"voice_clone,omitempty"`
	Dispatchable  bool     `json:"dispatchable"`
	Adapter       string   `json:"adapter"`
	Capabilities  []string `json:"capabilities,omitempty"`
	RequiredEnv   []string `json:"required_env,omitempty"`
	PresentEnv    []string `json:"present_env,omitempty"`
	MissingEnv    []string `json:"missing_env,omitempty"`
	OptionalEnv   []string `json:"optional_env,omitempty"`
	PromotionGate []string `json:"promotion_gate,omitempty"`
}

type CloudVoiceProfileCatalogFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type cloudVoiceProfileSpec struct {
	ID               string
	Label            string
	Vendor           string
	Family           ProviderFamily
	Protocol         string
	Lane             string
	Capabilities     []string
	AuthEnv          []string
	RequiredEnv      []string
	OptionalEnv      []string
	Realtime         bool
	VoiceClone       bool
	AdapterAvailable bool
	Default          bool
	PromotionGate    []string
}

func BuiltinCloudVoiceProfiles() []CloudVoiceProfileReadiness {
	report := CloudVoiceCatalogFromEnv(nil, "")
	return append([]CloudVoiceProfileReadiness(nil), report.Profiles...)
}

func CloudVoiceCatalogFromEnv(env []string, selected string) CloudVoiceCatalogReport {
	selectedID, findings := cloudVoiceSelectedProfile(env, selected)
	envMap := envNameSet(env)
	report := CloudVoiceCatalogReport{
		SchemaVersion:             CloudVoiceCatalogSchemaVersion,
		SelectedCloudVoiceProfile: selectedID,
		Findings:                  findings,
	}
	for _, spec := range cloudVoiceProfileSpecs() {
		requiredEnv, presentEnv, missingEnv := cloudVoiceProfileEnvReadiness(envMap, spec)
		configured := len(missingEnv) == 0
		status := CloudVoiceStatusCatalogOnly
		dispatchable := false
		adapter := "planned"
		if spec.AdapterAvailable {
			adapter = "available"
		}
		if configured && spec.AdapterAvailable {
			status = CloudVoiceStatusStaticReady
			dispatchable = true
		}
		report.Profiles = append(report.Profiles, CloudVoiceProfileReadiness{
			ID:            spec.ID,
			Label:         spec.Label,
			Vendor:        spec.Vendor,
			Family:        string(spec.Family),
			Protocol:      spec.Protocol,
			Lane:          spec.Lane,
			Status:        status,
			Selected:      spec.ID == selectedID,
			Default:       spec.Default,
			Configured:    configured,
			Realtime:      spec.Realtime,
			VoiceClone:    spec.VoiceClone,
			Dispatchable:  dispatchable,
			Adapter:       adapter,
			Capabilities:  append([]string(nil), spec.Capabilities...),
			RequiredEnv:   requiredEnv,
			PresentEnv:    presentEnv,
			MissingEnv:    missingEnv,
			OptionalEnv:   append([]string(nil), spec.OptionalEnv...),
			PromotionGate: append([]string(nil), spec.PromotionGate...),
		})
	}
	return report
}

func ValidCloudVoiceProfile(id string) bool {
	_, ok := cloudVoiceProfileByID(id)
	return ok
}

func DefaultCloudVoiceProfile(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	if ValidCloudVoiceProfile(id) {
		return id
	}
	return CloudVoiceDefaultProfile
}

func cloudVoiceSelectedProfile(env []string, selected string) (string, []CloudVoiceProfileCatalogFinding) {
	raw := strings.ToLower(strings.TrimSpace(selected))
	if raw == "" {
		raw = strings.ToLower(strings.TrimSpace(envValue(env, "A21_CLOUD_VOICE_PROFILE")))
	}
	if raw == "" {
		return CloudVoiceDefaultProfile, nil
	}
	if ValidCloudVoiceProfile(raw) {
		return raw, nil
	}
	code := "cloud_voice_profile_unknown"
	message := "A21 cloud voice profile is not in the cloud voice registry"
	switch {
	case containsLegacyProviderIdentity(raw):
		code = "cloud_voice_profile_legacy_identity"
		message = "A21 cloud voice profile contains a forbidden legacy identity"
	case containsBlockedProviderIdentity(raw):
		code = "cloud_voice_profile_blocked"
		message = "A21 cloud voice profile is blocked by project policy"
	}
	return CloudVoiceDefaultProfile, []CloudVoiceProfileCatalogFinding{{
		Code:    code,
		Message: message,
		Detail:  "A21_CLOUD_VOICE_PROFILE",
	}}
}

func cloudVoiceProfileByID(id string) (cloudVoiceProfileSpec, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, spec := range cloudVoiceProfileSpecs() {
		if spec.ID == id {
			return spec, true
		}
	}
	return cloudVoiceProfileSpec{}, false
}

func cloudVoiceProfileEnvReadiness(envMap map[string]bool, spec cloudVoiceProfileSpec) ([]string, []string, []string) {
	required := append([]string(nil), spec.AuthEnv...)
	required = append(required, spec.RequiredEnv...)
	var present []string
	var missing []string
	authPresent := len(spec.AuthEnv) == 0
	for _, name := range spec.AuthEnv {
		if envMap[name] {
			authPresent = true
			present = append(present, name)
		}
	}
	if !authPresent {
		missing = append(missing, spec.AuthEnv...)
	}
	for _, name := range spec.RequiredEnv {
		if envMap[name] {
			present = append(present, name)
		} else {
			missing = append(missing, name)
		}
	}
	return required, present, missing
}

func cloudVoiceProfileSpecs() []cloudVoiceProfileSpec {
	commonPromotion := []string{
		"provider_fixture_smoke",
		"redacted_5080_smoke",
		"physical_stackchan_ab",
		"barge_in_cancel_proof",
	}
	return []cloudVoiceProfileSpec{
		{
			ID:               CloudVoiceDefaultProfile,
			Label:            "Doubao realtime TTS",
			Vendor:           "doubao",
			Family:           ProviderFamilyVoiceHybrid,
			Protocol:         "doubao_realtime_tts_websocket",
			Lane:             "tts_realtime",
			Capabilities:     []string{"tts", "realtime", "streaming_audio", "voice_hybrid"},
			AuthEnv:          []string{"A21_DOUBAO_API_KEY", "A21_DOUBAO_ACCESS_TOKEN"},
			RequiredEnv:      []string{"A21_DOUBAO_TTS_MODEL", "A21_DOUBAO_TTS_VOICE"},
			OptionalEnv:      []string{"A21_DOUBAO_TTS_SAMPLE_RATE_HZ", "A21_DOUBAO_TTS_OUTPUT_FORMAT", "A21_DOUBAO_TTS_ENDPOINT_PROFILE"},
			Realtime:         true,
			AdapterAvailable: true,
			Default:          true,
			PromotionGate:    commonPromotion,
		},
		{
			ID:            "a21_doubao_voice_clone_tts",
			Label:         "Doubao voice clone TTS",
			Vendor:        "doubao",
			Family:        ProviderFamilyVoiceHybrid,
			Protocol:      "doubao_voice_clone_tts",
			Lane:          "voice_clone_tts",
			Capabilities:  []string{"tts", "voice_clone", "custom_voice"},
			AuthEnv:       []string{"A21_DOUBAO_API_KEY", "A21_DOUBAO_ACCESS_TOKEN"},
			RequiredEnv:   []string{"A21_DOUBAO_TTS_MODEL", "A21_DOUBAO_TTS_VOICE"},
			OptionalEnv:   []string{"A21_DOUBAO_VOICE_CLONE_PROFILE"},
			VoiceClone:    true,
			PromotionGate: commonPromotion,
		},
		{
			ID:            "a21_bailian_qwen_tts_realtime",
			Label:         "Bailian Qwen-TTS realtime",
			Vendor:        "bailian",
			Family:        ProviderFamilyVoiceHybrid,
			Protocol:      "dashscope_qwen_tts_realtime_websocket",
			Lane:          "tts_realtime",
			Capabilities:  []string{"tts", "realtime", "streaming_audio", "mainland_latency_candidate"},
			AuthEnv:       []string{"A21_DASHSCOPE_API_KEY"},
			RequiredEnv:   []string{"A21_BAILIAN_QWEN_TTS_MODEL"},
			OptionalEnv:   []string{"A21_BAILIAN_QWEN_TTS_VOICE_ID", "A21_BAILIAN_QWEN_TTS_SAMPLE_RATE_HZ", "A21_BAILIAN_QWEN_TTS_FORMAT"},
			Realtime:      true,
			PromotionGate: commonPromotion,
		},
		{
			ID:            "a21_bailian_qwen3_tts_vc_realtime",
			Label:         "Bailian Qwen3 TTS voice-clone realtime",
			Vendor:        "bailian",
			Family:        ProviderFamilyVoiceHybrid,
			Protocol:      "dashscope_qwen3_tts_vc_realtime_websocket",
			Lane:          "voice_clone_tts_realtime",
			Capabilities:  []string{"tts", "realtime", "voice_clone", "custom_voice"},
			AuthEnv:       []string{"A21_DASHSCOPE_API_KEY"},
			RequiredEnv:   []string{"A21_BAILIAN_QWEN_TTS_MODEL", "A21_BAILIAN_QWEN_TTS_VOICE_ID"},
			OptionalEnv:   []string{"A21_BAILIAN_QWEN_TTS_SAMPLE_RATE_HZ", "A21_BAILIAN_QWEN_TTS_FORMAT"},
			Realtime:      true,
			VoiceClone:    true,
			PromotionGate: commonPromotion,
		},
		{
			ID:            "a21_bailian_cosyvoice_realtime",
			Label:         "Bailian CosyVoice realtime",
			Vendor:        "bailian",
			Family:        ProviderFamilyVoiceHybrid,
			Protocol:      "dashscope_cosyvoice_realtime_websocket",
			Lane:          "tts_realtime",
			Capabilities:  []string{"tts", "realtime", "streaming_audio", "custom_voice"},
			AuthEnv:       []string{"A21_DASHSCOPE_API_KEY"},
			RequiredEnv:   []string{"A21_BAILIAN_COSYVOICE_MODEL"},
			OptionalEnv:   []string{"A21_BAILIAN_COSYVOICE_VOICE_ID", "A21_BAILIAN_COSYVOICE_SAMPLE_RATE_HZ", "A21_BAILIAN_COSYVOICE_FORMAT"},
			Realtime:      true,
			PromotionGate: commonPromotion,
		},
		{
			ID:            "a21_bailian_cosyvoice_clone_tts",
			Label:         "Bailian CosyVoice clone TTS",
			Vendor:        "bailian",
			Family:        ProviderFamilyVoiceHybrid,
			Protocol:      "dashscope_cosyvoice_clone_tts",
			Lane:          "voice_clone_tts",
			Capabilities:  []string{"tts", "voice_clone", "custom_voice"},
			AuthEnv:       []string{"A21_DASHSCOPE_API_KEY"},
			RequiredEnv:   []string{"A21_BAILIAN_COSYVOICE_MODEL", "A21_BAILIAN_COSYVOICE_VOICE_ID"},
			OptionalEnv:   []string{"A21_BAILIAN_COSYVOICE_SAMPLE_RATE_HZ", "A21_BAILIAN_COSYVOICE_FORMAT"},
			VoiceClone:    true,
			PromotionGate: commonPromotion,
		},
		{
			ID:            "a21_bailian_qwen_omni_realtime",
			Label:         "Bailian Qwen Omni realtime",
			Vendor:        "bailian",
			Family:        ProviderFamilyVoiceRealtime,
			Protocol:      "dashscope_qwen_omni_realtime_websocket",
			Lane:          "speech_to_speech_realtime",
			Capabilities:  []string{"speech_to_speech", "realtime", "barge_in", "explicit_arm_required"},
			AuthEnv:       []string{"A21_DASHSCOPE_API_KEY"},
			RequiredEnv:   []string{"A21_BAILIAN_OMNI_REALTIME_MODEL", "A21_BAILIAN_OMNI_REALTIME_VOICE_ID"},
			Realtime:      true,
			PromotionGate: commonPromotion,
		},
		{
			ID:            "a21_minimax_t2a_ws",
			Label:         "MiniMax T2A websocket",
			Vendor:        "minimax",
			Family:        ProviderFamilyVoiceHybrid,
			Protocol:      "minimax_t2a_websocket",
			Lane:          "tts_realtime",
			Capabilities:  []string{"tts", "realtime", "streaming_audio", "quality_candidate"},
			AuthEnv:       []string{"A21_MINIMAX_API_KEY"},
			RequiredEnv:   []string{"A21_MINIMAX_GROUP_ID", "A21_MINIMAX_TTS_MODEL", "A21_MINIMAX_VOICE_ID"},
			OptionalEnv:   []string{"A21_MINIMAX_TTS_SAMPLE_RATE_HZ", "A21_MINIMAX_TTS_FORMAT"},
			Realtime:      true,
			PromotionGate: commonPromotion,
		},
		{
			ID:            "a21_minimax_t2a_http",
			Label:         "MiniMax T2A HTTP",
			Vendor:        "minimax",
			Family:        ProviderFamilyVoiceHybrid,
			Protocol:      "minimax_t2a_http",
			Lane:          "tts_quality_http",
			Capabilities:  []string{"tts", "quality_candidate", "http_fallback"},
			AuthEnv:       []string{"A21_MINIMAX_API_KEY"},
			RequiredEnv:   []string{"A21_MINIMAX_GROUP_ID", "A21_MINIMAX_TTS_MODEL", "A21_MINIMAX_VOICE_ID"},
			OptionalEnv:   []string{"A21_MINIMAX_TTS_SAMPLE_RATE_HZ", "A21_MINIMAX_TTS_FORMAT"},
			PromotionGate: commonPromotion,
		},
		{
			ID:            "a21_minimax_voice_clone_tts",
			Label:         "MiniMax voice clone TTS",
			Vendor:        "minimax",
			Family:        ProviderFamilyVoiceHybrid,
			Protocol:      "minimax_voice_clone_tts",
			Lane:          "voice_clone_tts",
			Capabilities:  []string{"tts", "voice_clone", "custom_voice"},
			AuthEnv:       []string{"A21_MINIMAX_API_KEY"},
			RequiredEnv:   []string{"A21_MINIMAX_GROUP_ID", "A21_MINIMAX_TTS_MODEL", "A21_MINIMAX_VOICE_ID"},
			OptionalEnv:   []string{"A21_MINIMAX_TTS_SAMPLE_RATE_HZ", "A21_MINIMAX_TTS_FORMAT", "A21_MINIMAX_VOICE_CLONE_PROFILE"},
			VoiceClone:    true,
			PromotionGate: commonPromotion,
		},
	}
}
