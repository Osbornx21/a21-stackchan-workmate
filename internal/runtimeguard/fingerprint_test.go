package runtimeguard

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

type fakeRunner map[string]string

func (f fakeRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	return f[name+" "+strings.Join(args, " ")], nil
}

func TestFingerprintDetectsTunAndMappedDNS(t *testing.T) {
	runner := fakeRunner{
		"route -n get default":              "interface: utun8\n",
		"dig +short " + DefaultDNSProbeHost: "198.18.0.44\n",
	}

	fp := DetectFingerprint(context.Background(), runner, []string{"PATH=/usr/bin"})

	if fp.NetworkInterface != "utun8" {
		t.Fatalf("NetworkInterface = %q, want utun8", fp.NetworkInterface)
	}
	if fp.ExternalDNSMappedIP != "198.18.0.44" {
		t.Fatalf("ExternalDNSMappedIP = %q, want 198.18.0.44", fp.ExternalDNSMappedIP)
	}
	if !fp.UsesTun() {
		t.Fatal("expected UsesTun to be true")
	}
}

func TestFingerprintFallsBackToLinuxDefaultRoute(t *testing.T) {
	runner := fakeRunner{
		"route -n get default":              "",
		"ip route show default":             "default via 172.24.30.253 dev eth0 proto dhcp src 172.24.30.54 metric 100\n",
		"dig +short " + DefaultDNSProbeHost: "104.20.23.154\n",
	}

	fp := DetectFingerprint(context.Background(), runner, []string{"PATH=/usr/bin"})

	if fp.NetworkInterface != "eth0" {
		t.Fatalf("NetworkInterface = %q, want eth0", fp.NetworkInterface)
	}
	if fp.ExternalDNSMappedIP != "104.20.23.154" {
		t.Fatalf("ExternalDNSMappedIP = %q, want 104.20.23.154", fp.ExternalDNSMappedIP)
	}
}

func TestFingerprintRecordsProxyEnv(t *testing.T) {
	fp := DetectFingerprint(context.Background(), fakeRunner{}, []string{
		"HTTPS_PROXY=http://127.0.0.1:7892",
		"NO_PROXY=127.0.0.1,localhost",
	})

	want := []string{"HTTPS_PROXY", "NO_PROXY"}
	if !reflect.DeepEqual(fp.ProxyEnv, want) {
		t.Fatalf("ProxyEnv = %#v, want %#v", fp.ProxyEnv, want)
	}
}

func TestCheckFingerprintBlocksMissingFields(t *testing.T) {
	findings := CheckFingerprint(Fingerprint{})
	if len(findings) != 2 {
		t.Fatalf("findings length = %d, want 2", len(findings))
	}
	for _, finding := range findings {
		if finding.Code != "fingerprint_missing" {
			t.Fatalf("Code = %q, want fingerprint_missing", finding.Code)
		}
		if finding.Severity != SeverityBlock {
			t.Fatalf("Severity = %q, want block", finding.Severity)
		}
	}
}

func TestCheckFingerprintAllowsCompleteFingerprint(t *testing.T) {
	findings := CheckFingerprint(Fingerprint{
		NetworkInterface:    "en0",
		ExternalDNSMappedIP: "198.18.0.145",
	})
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
}
