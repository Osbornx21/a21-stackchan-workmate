package runtimeguard

type Config struct {
	ProjectName         string
	EnvPrefix           string
	LegacyEnvPrefixes   []string
	LegacyEndpointPorts []int
	ReservedPorts       []int
	LegacyPathParts     []string
}

func DefaultConfig() Config {
	return Config{
		ProjectName: "A21",
		EnvPrefix:   "A21_",
		LegacyEnvPrefixes: []string{
			"X21_",
			"V21_",
			"ROLEPLAY_",
			"VOICE_KNOWLEDGE_",
		},
		LegacyEndpointPorts: []int{8000, 8080, 10095, 18080, 4173, 42173, 16686, 16687},
		ReservedPorts:       []int{21080, 21081, 21073, 21086, 21095, 21114, 21434},
		LegacyPathParts: []string{
			"/Users/jiyurun/Documents/小马暴力",
			"/Users/jiyurun/Documents/v21-knowledge-platform",
		},
	}
}

func (c Config) IsReservedPort(port int) bool {
	for _, reserved := range c.ReservedPorts {
		if reserved == port {
			return true
		}
	}
	return false
}
