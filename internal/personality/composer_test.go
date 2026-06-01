package personality

import (
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
