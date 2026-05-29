package runtimeguard

import "testing"

func TestCheckEnvBlocksLegacyPrefixes(t *testing.T) {
	findings := CheckEnv(DefaultConfig(), []string{
		"A21_CORE_PORT=21080",
		"X21_BACKEND=http://127.0.0.1:8000",
		"V21_KNOWLEDGE_BASE_URL=http://127.0.0.1:18080",
	})

	if len(findings) != 2 {
		t.Fatalf("findings length = %d, want 2", len(findings))
	}
	for _, finding := range findings {
		if finding.Severity != SeverityBlock {
			t.Fatalf("Severity = %q, want block", finding.Severity)
		}
		if finding.Code != "legacy_env" {
			t.Fatalf("Code = %q, want legacy_env", finding.Code)
		}
	}
}

func TestCheckEnvAllowsA21Prefixes(t *testing.T) {
	findings := CheckEnv(DefaultConfig(), []string{
		"A21_CORE_PORT=21080",
		"PATH=/usr/bin",
	})
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
}

func TestCheckEnvBlocksA21EndpointValuesPointingAtLegacyPorts(t *testing.T) {
	findings := CheckEnv(DefaultConfig(), []string{
		"A21_PROVIDER_ENDPOINT=http://127.0.0.1:8000",
		"A21_KNOWLEDGE_BASE_URL=http://127.0.0.1:18080",
	})

	if len(findings) != 2 {
		t.Fatalf("findings length = %d, want 2", len(findings))
	}
	for _, finding := range findings {
		if finding.Code != "legacy_endpoint" {
			t.Fatalf("Code = %q, want legacy_endpoint", finding.Code)
		}
		if finding.Severity != SeverityBlock {
			t.Fatalf("Severity = %q, want block", finding.Severity)
		}
		if finding.Detail == "http://127.0.0.1:8000" || finding.Detail == "http://127.0.0.1:18080" {
			t.Fatalf("Detail leaked endpoint value: %q", finding.Detail)
		}
	}
}

func TestCheckEnvAllowsA21EndpointValuesOnA21Ports(t *testing.T) {
	findings := CheckEnv(DefaultConfig(), []string{
		"A21_PROVIDER_ENDPOINT=http://127.0.0.1:21080",
	})
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
}
