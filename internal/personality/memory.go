package personality

import "strings"

const (
	MemoryStateSchemaVersion = "a21.personality_memory_state.v1"
	MemoryPolicyBoundedHints = "bounded_prompt_hints"
	MemoryMaxItems           = 6
	MemoryMaxItemRunes       = 80

	MemoryUserPreferencesEnv = "A21_MEMORY_USER_PREFERENCES"
	MemorySessionNotesEnv    = "A21_MEMORY_SESSION_NOTES"
)

type MemoryScope string

const (
	MemoryScopeUserPreference MemoryScope = "user_preference"
	MemoryScopeSessionMemory  MemoryScope = "session_memory"
)

type MemoryHint struct {
	Scope MemoryScope
	ID    string
	Text  string
}

type MemoryState struct {
	SchemaVersion       string          `json:"schema_version"`
	Status              string          `json:"status"`
	ContractReady       bool            `json:"contract_ready"`
	Configured          bool            `json:"configured"`
	PromptInputReady    bool            `json:"prompt_input_ready"`
	Policy              string          `json:"policy"`
	MaxItems            int             `json:"max_items"`
	MaxItemChars        int             `json:"max_item_chars"`
	UserPreferenceCount int             `json:"user_preference_count"`
	SessionMemoryCount  int             `json:"session_memory_count"`
	SourceEnv           []string        `json:"source_env,omitempty"`
	Redaction           MemoryRedaction `json:"redaction"`
	Findings            []MemoryFinding `json:"findings,omitempty"`
}

type MemoryRedaction struct {
	MemoryTextStored      bool `json:"memory_text_stored"`
	PromptTextStored      bool `json:"instruction_text_stored"`
	TranscriptStored      bool `json:"asr_text_stored"`
	ProviderOutputStored  bool `json:"model_text_stored"`
	FullURLsStored        bool `json:"network_locator_stored"`
	LocalPathsStored      bool `json:"filesystem_locator_stored"`
	CredentialValueStored bool `json:"credential_value_stored"`
}

type MemoryFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func MemoryStateFromEnv(env []string) (MemoryState, []MemoryHint) {
	candidates := make([]MemoryHint, 0, MemoryMaxItems)
	sourceEnv := make([]string, 0, 2)
	if values, ok := memoryEnvValues(env, MemoryUserPreferencesEnv); ok {
		sourceEnv = append(sourceEnv, MemoryUserPreferencesEnv)
		candidates = appendMemoryCandidates(candidates, MemoryScopeUserPreference, values)
	}
	if values, ok := memoryEnvValues(env, MemorySessionNotesEnv); ok {
		sourceEnv = append(sourceEnv, MemorySessionNotesEnv)
		candidates = appendMemoryCandidates(candidates, MemoryScopeSessionMemory, values)
	}

	state := MemoryState{
		SchemaVersion: MemoryStateSchemaVersion,
		Status:        "empty",
		ContractReady: true,
		Configured:    len(sourceEnv) > 0,
		Policy:        MemoryPolicyBoundedHints,
		MaxItems:      MemoryMaxItems,
		MaxItemChars:  MemoryMaxItemRunes,
		SourceEnv:     sourceEnv,
	}
	hints := make([]MemoryHint, 0, MemoryMaxItems)
	for _, candidate := range candidates {
		if len(hints) >= MemoryMaxItems {
			state.Findings = append(state.Findings, MemoryFinding{
				Code:    "memory_hint_limit_reached",
				Message: "Additional A21 memory hints were ignored after the bounded prompt limit",
			})
			break
		}
		text := strings.TrimSpace(candidate.Text)
		if text == "" {
			continue
		}
		if unsafeMemoryHintText(text) {
			state.Findings = append(state.Findings, MemoryFinding{
				Code:    "memory_hint_rejected",
				Message: "A21 memory hint was rejected by redaction policy",
			})
			continue
		}
		truncated, didTruncate := truncateMemoryHintText(text)
		if didTruncate {
			state.Findings = append(state.Findings, MemoryFinding{
				Code:    "memory_hint_truncated",
				Message: "A21 memory hint was truncated to the prompt limit",
			})
		}
		hint := MemoryHint{Scope: candidate.Scope, ID: candidate.ID, Text: truncated}
		hints = append(hints, hint)
		switch candidate.Scope {
		case MemoryScopeUserPreference:
			state.UserPreferenceCount++
		case MemoryScopeSessionMemory:
			state.SessionMemoryCount++
		}
	}
	if len(hints) > 0 {
		state.Status = "ready"
		state.PromptInputReady = true
	} else if state.Configured && len(state.Findings) > 0 {
		state.Status = "blocked"
	}
	return state, hints
}

func appendMemoryCandidates(out []MemoryHint, scope MemoryScope, values []string) []MemoryHint {
	count := 0
	for _, value := range values {
		for _, line := range strings.Split(value, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			count++
			out = append(out, MemoryHint{
				Scope: scope,
				ID:    string(scope) + "_" + positiveIntString(count),
				Text:  line,
			})
		}
	}
	return out
}

func memoryEnvValues(env []string, name string) ([]string, bool) {
	prefix := name + "="
	var values []string
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			values = append(values, strings.TrimPrefix(entry, prefix))
		}
	}
	return values, len(values) > 0
}

func safeMemoryHints(hints []MemoryHint) []MemoryHint {
	out := make([]MemoryHint, 0, MemoryMaxItems)
	for _, hint := range hints {
		if len(out) >= MemoryMaxItems {
			break
		}
		text := strings.TrimSpace(hint.Text)
		if text == "" || unsafeMemoryHintText(text) {
			continue
		}
		text, _ = truncateMemoryHintText(text)
		scope := hint.Scope
		if scope == "" {
			scope = MemoryScopeSessionMemory
		}
		id := safeMemoryHintID(hint.ID)
		if id == "" {
			id = string(scope) + "_" + positiveIntString(len(out)+1)
		}
		out = append(out, MemoryHint{Scope: scope, ID: id, Text: text})
	}
	return out
}

func unsafeMemoryHintText(text string) bool {
	lower := strings.ToLower(text)
	for _, marker := range []string{
		"http://",
		"https://",
		"file://",
		"/users/",
		"/tmp/",
		"\\users\\",
		"secret",
		"token",
		"sk-",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func truncateMemoryHintText(text string) (string, bool) {
	runes := []rune(text)
	if len(runes) <= MemoryMaxItemRunes {
		return text, false
	}
	return strings.TrimSpace(string(runes[:MemoryMaxItemRunes])), true
}

func safeMemoryHintID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-':
		default:
			return ""
		}
	}
	return id
}

func positiveIntString(value int) string {
	if value <= 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}
