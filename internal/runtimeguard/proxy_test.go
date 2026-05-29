package runtimeguard

import "testing"

func TestProxyPolicyBlocksGlobalProxyWithoutDirectBypass(t *testing.T) {
	report := EvaluateProxyPolicy([]string{
		"HTTPS_PROXY=http://127.0.0.1:7890",
		"NO_PROXY=localhost,127.0.0.1",
	})

	findings := CheckProxyPolicy(report)
	if len(findings) != 1 {
		t.Fatalf("findings length = %d, want 1", len(findings))
	}
	if findings[0].Code != "proxy_direct_bypass_missing" {
		t.Fatalf("code = %q, want proxy_direct_bypass_missing", findings[0].Code)
	}
	if findings[0].Severity != SeverityBlock {
		t.Fatalf("severity = %q, want block", findings[0].Severity)
	}
	if report.DirectConnectOK {
		t.Fatal("direct-connect coverage should not be ok")
	}
	if len(report.DirectConnectMissing) == 0 {
		t.Fatal("expected missing direct-connect targets")
	}
}

func TestProxyPolicyAcceptsA21NoProxyCoverage(t *testing.T) {
	report := EvaluateProxyPolicy([]string{
		"ALL_PROXY=socks5://127.0.0.1:7890",
		"A21_NO_PROXY=localhost,127.0.0.1,::1,.local,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16",
	})

	if findings := CheckProxyPolicy(report); len(findings) != 0 {
		t.Fatalf("expected no proxy findings, got %#v", findings)
	}
	if !report.GlobalProxyConfigured {
		t.Fatal("expected global proxy to be detected")
	}
	if !report.DirectConnectOK {
		t.Fatalf("direct-connect coverage should be ok, missing %#v", report.DirectConnectMissing)
	}
	if len(report.GlobalProxyEnv) != 1 || report.GlobalProxyEnv[0] != "ALL_PROXY" {
		t.Fatalf("global proxy env = %#v", report.GlobalProxyEnv)
	}
	if len(report.NoProxyEnv) != 1 || report.NoProxyEnv[0] != "A21_NO_PROXY" {
		t.Fatalf("no proxy env = %#v", report.NoProxyEnv)
	}
}

func TestProxyPolicyNormalizesNoProxyHostPortsAndWildcardLocal(t *testing.T) {
	report := EvaluateProxyPolicy([]string{
		"HTTP_PROXY=http://127.0.0.1:7890",
		"NO_PROXY=localhost:21080,127.0.0.1:21080,[::1]:21080,*.local,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16",
	})

	if findings := CheckProxyPolicy(report); len(findings) != 0 {
		t.Fatalf("expected no proxy findings, got %#v with report %#v", findings, report)
	}
}

func TestProxyPolicyAllowsMissingNoProxyWhenNoGlobalProxyExists(t *testing.T) {
	report := EvaluateProxyPolicy([]string{"A21_ENV=development"})

	if findings := CheckProxyPolicy(report); len(findings) != 0 {
		t.Fatalf("expected no proxy findings, got %#v", findings)
	}
	if report.GlobalProxyConfigured {
		t.Fatal("did not expect global proxy")
	}
	if !report.DirectConnectOK {
		t.Fatalf("direct-connect should be ok without global proxy, missing %#v", report.DirectConnectMissing)
	}
}

func TestProxyPolicyReportDoesNotLeakProxyValues(t *testing.T) {
	report := EvaluateProxyPolicy([]string{
		"HTTPS_PROXY=http://user:secret@127.0.0.1:7890",
		"A21_PROVIDER_PROXY_URL=http://provider-secret@127.0.0.1:7891",
		"A21_NO_PROXY=localhost,127.0.0.1,::1,.local,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16",
	})

	if len(report.GlobalProxyEnv) != 1 || report.GlobalProxyEnv[0] != "HTTPS_PROXY" {
		t.Fatalf("global proxy env = %#v", report.GlobalProxyEnv)
	}
	if len(report.ProviderProxyEnv) != 1 || report.ProviderProxyEnv[0] != "A21_PROVIDER_PROXY_URL" {
		t.Fatalf("provider proxy env = %#v", report.ProviderProxyEnv)
	}
	for _, leaked := range []string{"secret", "7890", "7891"} {
		if containsProxyReportValue(report, leaked) {
			t.Fatalf("proxy report leaked %q: %#v", leaked, report)
		}
	}
}

func containsProxyReportValue(report ProxyPolicyReport, value string) bool {
	for _, values := range [][]string{
		report.GlobalProxyEnv,
		report.ProviderProxyEnv,
		report.NoProxyEnv,
		report.DirectConnectMissing,
		report.DirectConnectCovered,
	} {
		for _, item := range values {
			if item == value {
				return true
			}
		}
	}
	return false
}
