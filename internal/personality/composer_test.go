package personality

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPersonalityComposeWorkmateUsesRuntimeAssetsAndOneMode(t *testing.T) {
	prompt, err := Compose(Options{
		Mode:     ModeWorkmate,
		UserText: "今天评审被打断了",
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Core Identity",
		"Tone Rules",
		"Workmate Mode",
		"不超过12个字",
		"今天评审被打断了",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
	for _, forbidden := range []string{
		"Professional Mode",
		"Companion Mode",
		"Post-Meeting Playbook",
		"Failure Overlay",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt included unselected asset %q:\n%s", forbidden, prompt)
		}
	}
}

func TestPersonalityComposeAddsOneScenarioAndFailureOnlyWhenRequested(t *testing.T) {
	prompt, err := Compose(Options{
		Mode:     ModeWorkmate,
		Scenario: ScenarioPostMeeting,
		UserText: "刚开完会，脑子乱",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Workmate Mode", "Post-Meeting Playbook"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
	for _, forbidden := range []string{"Boss Challenge Playbook", "Failure Overlay"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt included unrequested overlay %q:\n%s", forbidden, prompt)
		}
	}

	failedPrompt, err := Compose(Options{
		Mode:           ModeWorkmate,
		UserText:       "本地播放失败",
		FailureOverlay: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(failedPrompt, "Failure Overlay") {
		t.Fatalf("failure overlay missing:\n%s", failedPrompt)
	}
	if strings.Contains(failedPrompt, "Post-Meeting Playbook") {
		t.Fatalf("failure-only prompt included scenario:\n%s", failedPrompt)
	}
}

func TestPersonalityComposeRoleSoulAddsOnlySelectedSoul(t *testing.T) {
	prompt, err := Compose(Options{
		Mode:     ModeRoleplay,
		RoleSoul: RoleSoulWryPeer,
		Scenario: ScenarioDeskMouthpiece,
		UserText: "这个需求边界又变了",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Role Soul: Wry Peer",
		"Roleplay Mode",
		"Desk Mouthpiece Playbook",
		"这个需求边界又变了",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
	for _, forbidden := range []string{
		"Role Soul: Calm Anchor",
		"Role Soul: A21 Desk Workmate",
		"Boss Challenge Playbook",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt included unselected role soul/scenario %q:\n%s", forbidden, prompt)
		}
	}
}

func TestPersonalityComposeProfessionalIncludesEvidenceBoundaryAndPublicPrivateSafety(t *testing.T) {
	prompt, err := Compose(Options{
		Mode:     ModeProfessional,
		UserText: "专业模式，给我证据",
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Professional Mode",
		"V21 adapter path only",
		"professional mode must not run through opaque realtime chat",
		"public/private state",
		"专业模式，给我证据",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("professional prompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "Workmate Mode") {
		t.Fatalf("professional prompt included workmate mode:\n%s", prompt)
	}
}

func TestPersonalityMemoryStateFromEnvRedactsReportAndBuildsPromptHints(t *testing.T) {
	state, hints := MemoryStateFromEnv([]string{
		"A21_MEMORY_USER_PREFERENCES=偏好短句\n不要鸡汤",
		"A21_MEMORY_SESSION_NOTES=当前任务是座舱 PRD 评审\nhttp://secret.example/leak\n/Users/me/a21-secret.txt",
	})

	if !state.ContractReady || !state.Configured || !state.PromptInputReady || state.Status != "ready" {
		t.Fatalf("memory state = %+v, want configured prompt-ready contract", state)
	}
	if state.UserPreferenceCount != 2 || state.SessionMemoryCount != 1 {
		t.Fatalf("memory counts = preferences:%d session:%d, want 2/1", state.UserPreferenceCount, state.SessionMemoryCount)
	}
	if len(hints) != 3 {
		t.Fatalf("hints = %#v, want three safe prompt hints", hints)
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"偏好短句", "不要鸡汤", "当前任务是座舱 PRD 评审", "secret.example", "/Users/me"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("memory report leaked %q: %s", forbidden, string(data))
		}
	}

	prompt, err := Compose(Options{
		Mode:        ModeWorkmate,
		UserText:    "先接住这一轮",
		MemoryHints: hints,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Memory Hints",
		"user_preference:user_preference_1",
		"偏好短句",
		"session_memory:session_memory_1",
		"当前任务是座舱 PRD 评审",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing memory hint %q:\n%s", want, prompt)
		}
	}
	for _, forbidden := range []string{"secret.example", "/Users/me"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt included unsafe memory hint %q:\n%s", forbidden, prompt)
		}
	}
}
