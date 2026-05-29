package providers

import "strings"

func NewGatewayVoiceProviderFromEnv(env []string) VoiceProvider {
	rawMode := strings.TrimSpace(envValue(env, "A21_GATEWAY_VOICE_PROVIDER"))
	if rawMode == "" {
		rawMode = "mock"
	}
	mode := strings.ToLower(rawMode)
	switch mode {
	case "mock":
		return NewMockVoiceProvider()
	case "selected":
		return NewVoiceProviderFromEnv(env)
	default:
		if containsLegacyProviderIdentity(mode) {
			return unavailableVoiceProvider{
				name:   "invalid_gateway_voice_provider",
				detail: "gateway voice provider mode rejected",
			}
		}
		return unavailableVoiceProvider{
			name:   "invalid_gateway_voice_provider",
			detail: "gateway voice provider mode is unknown",
		}
	}
}
