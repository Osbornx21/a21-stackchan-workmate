# A21 Phase 1 Clean Skeleton Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first A21 code skeleton with a Go module, runtime guardrails, environment fingerprinting, protocol/provider contracts, and repeatable local verification.

**Architecture:** A21 starts as a Go modular monolith named `a21-core`, with small internal packages for build metadata, runtime guardrails, protocol types, provider contracts, and the CLI. Phase 1 deliberately avoids real provider calls, firmware changes, Docker services, and any connection to X21/V21 so the new project begins from a clean, defensible baseline.

**Tech Stack:** Go 1.26+, standard library only, `make`, local CLI command `a21`, tests via `go test ./...`.

---

## Scope Check

This plan implements only Phase 1 from the approved architecture spec:

- Go module with `a21-core`.
- Runtime preflight CLI.
- Reserved port checker.
- Environment fingerprint.
- Fake provider interfaces.
- Minimal protocol types.
- Local verification.

It does not implement real ASR/LLM/TTS providers, Qdrant/PostgreSQL, TypeScript console, firmware, Docker Compose, or office deployment. Those are separate future plans.

## File Structure

- Create `go.mod`: local module path `a21.local/a21`, avoiding X21/V21 identity.
- Create `cmd/a21/main.go`: thin binary entrypoint; no business logic.
- Create `internal/app/app.go`: CLI command parsing and output formatting.
- Create `internal/app/app_test.go`: CLI behavior tests without shelling out.
- Create `internal/buildinfo/buildinfo.go`: A21 service identity and version constants.
- Create `internal/buildinfo/buildinfo_test.go`: identity tests.
- Create `internal/runtimeguard/result.go`: shared preflight result types.
- Create `internal/runtimeguard/config.go`: A21 namespace, legacy namespace, and reserved port config.
- Create `internal/runtimeguard/env.go`: legacy environment variable guard.
- Create `internal/runtimeguard/env_test.go`: legacy environment variable tests.
- Create `internal/runtimeguard/ports.go`: reserved port occupancy guard.
- Create `internal/runtimeguard/ports_test.go`: deterministic port tests using ephemeral listeners.
- Create `internal/runtimeguard/paths.go`: legacy working-directory guard.
- Create `internal/runtimeguard/paths_test.go`: path guard tests.
- Create `internal/runtimeguard/fingerprint.go`: environment fingerprint and command runner.
- Create `internal/runtimeguard/fingerprint_test.go`: fake-command tests for proxy/TUN detection.
- Create `internal/runtimeguard/preflight.go`: composes all guard checks.
- Create `internal/runtimeguard/preflight_test.go`: aggregate preflight tests.
- Create `internal/protocol/message.go`: versioned A21 device protocol message types.
- Create `internal/protocol/message_test.go`: JSON and message-family tests.
- Create `internal/providers/contracts.go`: ASR/LLM/TTS/knowledge provider interfaces and event types.
- Create `internal/providers/contracts_test.go`: compile-time fake provider tests.
- Create `Makefile`: local verification commands.
- Modify `docs/a21/00-project-charter-and-home-baseline.md`: add Phase 1 implementation pointer.

## Task 1: Bootstrap Go Module and Build Identity

**Files:**
- Create: `go.mod`
- Create: `internal/buildinfo/buildinfo_test.go`
- Create: `internal/buildinfo/buildinfo.go`
- Create: `cmd/a21/main.go`

- [ ] **Step 1: Create the Go module file**

Create `go.mod`:

```go
module a21.local/a21

go 1.26
```

- [ ] **Step 2: Write the failing build identity test**

Create `internal/buildinfo/buildinfo_test.go`:

```go
package buildinfo

import "testing"

func TestBuildInfoDefaults(t *testing.T) {
	if ServiceName != "a21-core" {
		t.Fatalf("ServiceName = %q, want %q", ServiceName, "a21-core")
	}
	if ProjectName != "A21" {
		t.Fatalf("ProjectName = %q, want %q", ProjectName, "A21")
	}
	if Version != "0.1.0-dev" {
		t.Fatalf("Version = %q, want %q", Version, "0.1.0-dev")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/buildinfo -run TestBuildInfoDefaults -v`

Expected: FAIL with undefined identifiers `ServiceName`, `ProjectName`, or `Version`.

- [ ] **Step 4: Implement build identity constants**

Create `internal/buildinfo/buildinfo.go`:

```go
package buildinfo

const (
	ProjectName = "A21"
	ServiceName = "a21-core"
	Version     = "0.1.0-dev"
)
```

- [ ] **Step 5: Create the first binary entrypoint**

Create `cmd/a21/main.go`:

```go
package main

import (
	"fmt"

	"a21.local/a21/internal/buildinfo"
)

func main() {
	fmt.Printf("%s %s (%s)\n", buildinfo.ProjectName, buildinfo.Version, buildinfo.ServiceName)
}
```

- [ ] **Step 6: Run tests and binary**

Run: `go test ./internal/buildinfo -v`

Expected: PASS.

Run: `go run ./cmd/a21`

Expected output:

```text
A21 0.1.0-dev (a21-core)
```

- [ ] **Step 7: Commit**

```bash
git add go.mod cmd/a21/main.go internal/buildinfo
git commit -m "chore: bootstrap a21 core module"
```

## Task 2: Add Runtime Guard Result and Config Types

**Files:**
- Create: `internal/runtimeguard/result.go`
- Create: `internal/runtimeguard/config.go`
- Create: `internal/runtimeguard/config_test.go`

- [ ] **Step 1: Write the failing config test**

Create `internal/runtimeguard/config_test.go`:

```go
package runtimeguard

import "testing"

func TestDefaultConfigKeepsA21Isolated(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ProjectName != "A21" {
		t.Fatalf("ProjectName = %q, want A21", cfg.ProjectName)
	}
	if cfg.EnvPrefix != "A21_" {
		t.Fatalf("EnvPrefix = %q, want A21_", cfg.EnvPrefix)
	}
	if len(cfg.ReservedPorts) == 0 {
		t.Fatal("ReservedPorts is empty")
	}
	if len(cfg.LegacyEnvPrefixes) == 0 {
		t.Fatal("expected legacy env prefixes")
	}
	for _, port := range []int{8000, 8080, 10095, 18080} {
		if cfg.IsReservedPort(port) {
			t.Fatalf("legacy port %d must not be reserved for A21", port)
		}
	}
	for _, port := range []int{21080, 21081, 21073, 21086, 21095, 21114, 21434} {
		if !cfg.IsReservedPort(port) {
			t.Fatalf("A21 port %d should be reserved", port)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/runtimeguard -run TestDefaultConfigKeepsA21Isolated -v`

Expected: FAIL with `undefined: DefaultConfig`.

- [ ] **Step 3: Implement result types**

Create `internal/runtimeguard/result.go`:

```go
package runtimeguard

type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityBlock Severity = "block"
)

type Finding struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Detail   string   `json:"detail,omitempty"`
}

type Result struct {
	OK       bool      `json:"ok"`
	Findings []Finding `json:"findings"`
}

func NewResult(findings []Finding) Result {
	ok := true
	for _, finding := range findings {
		if finding.Severity == SeverityBlock {
			ok = false
			break
		}
	}
	return Result{OK: ok, Findings: findings}
}
```

- [ ] **Step 4: Implement config**

Create `internal/runtimeguard/config.go`:

```go
package runtimeguard

type Config struct {
	ProjectName       string
	EnvPrefix         string
	LegacyEnvPrefixes []string
	LegacyEndpointPorts []int
	ReservedPorts     []int
	LegacyPathParts   []string
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
		ReservedPorts: []int{21080, 21081, 21073, 21086, 21095, 21114, 21434},
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
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/runtimeguard -run TestDefaultConfigKeepsA21Isolated -v`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/runtimeguard/result.go internal/runtimeguard/config.go internal/runtimeguard/config_test.go
git commit -m "feat: define a21 runtime guard config"
```

## Task 3: Add Legacy Environment Variable Guard

**Files:**
- Create: `internal/runtimeguard/env.go`
- Create: `internal/runtimeguard/env_test.go`

- [ ] **Step 1: Write failing env guard tests**

Create `internal/runtimeguard/env_test.go`:

```go
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
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/runtimeguard -run TestCheckEnv -v`

Expected: FAIL with `undefined: CheckEnv`.

- [ ] **Step 3: Implement env guard**

Create `internal/runtimeguard/env.go`:

```go
package runtimeguard

import (
	"net/url"
	"strconv"
	"strings"
)

func CheckEnv(cfg Config, env []string) []Finding {
	findings := make([]Finding, 0)
	for _, entry := range env {
		name, value, _ := strings.Cut(entry, "=")
		for _, legacy := range cfg.LegacyEnvPrefixes {
			if strings.HasPrefix(name, legacy) {
				findings = append(findings, Finding{
					Code:     "legacy_env",
					Severity: SeverityBlock,
					Message:  "A21 runtime refuses legacy project environment variables",
					Detail:   name,
				})
			}
		}
		if strings.HasPrefix(name, cfg.EnvPrefix) && isEndpointEnvName(name) {
			if port, ok := legacyEndpointPort(value, cfg.LegacyEndpointPorts); ok {
				findings = append(findings, Finding{
					Code:     "legacy_endpoint",
					Severity: SeverityBlock,
					Message:  "A21 runtime refuses endpoints that point to legacy project ports",
					Detail:   name + " uses legacy port " + strconv.Itoa(port),
				})
			}
		}
	}
	return findings
}

func isEndpointEnvName(name string) bool {
	upper := strings.ToUpper(name)
	return strings.Contains(upper, "URL") ||
		strings.Contains(upper, "ENDPOINT") ||
		strings.HasSuffix(upper, "_ADDR") ||
		strings.HasSuffix(upper, "_HOST")
}

func legacyEndpointPort(value string, legacyPorts []int) (int, bool) {
	for _, candidate := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	}) {
		if !strings.Contains(candidate, "://") && strings.Contains(candidate, ":") {
			candidate = "a21://" + candidate
		}
		parsed, err := url.Parse(candidate)
		if err != nil || parsed.Port() == "" {
			continue
		}
		port, err := strconv.Atoi(parsed.Port())
		if err != nil {
			continue
		}
		for _, legacyPort := range legacyPorts {
			if port == legacyPort {
				return port, true
			}
		}
	}
	return 0, false
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/runtimeguard -run TestCheckEnv -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/runtimeguard/env.go internal/runtimeguard/env_test.go
git commit -m "feat: block legacy environment leakage"
```

## Task 4: Add Reserved Port Guard

**Files:**
- Create: `internal/runtimeguard/ports.go`
- Create: `internal/runtimeguard/ports_test.go`

- [ ] **Step 1: Write failing port guard tests**

Create `internal/runtimeguard/ports_test.go`:

```go
package runtimeguard

import (
	"net"
	"strconv"
	"testing"
)

func TestCheckPortsBlocksOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	findings := CheckPorts([]int{port})
	result := NewResult(findings)

	if result.OK {
		t.Fatal("expected occupied port to block startup")
	}
	if got := result.Findings[0].Detail; got != strconv.Itoa(port) {
		t.Fatalf("detail = %q, want %q", got, strconv.Itoa(port))
	}
}

func TestCheckPortsAllowsFreePort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	result := NewResult(CheckPorts([]int{port}))
	if !result.OK {
		t.Fatalf("expected free port, got findings: %#v", result.Findings)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/runtimeguard -run TestCheckPorts -v`

Expected: FAIL with `undefined: CheckPorts`.

- [ ] **Step 3: Implement port guard**

Create `internal/runtimeguard/ports.go`:

```go
package runtimeguard

import (
	"fmt"
	"net"
)

func CheckPorts(ports []int) []Finding {
	findings := make([]Finding, 0)
	for _, port := range ports {
		address := fmt.Sprintf("127.0.0.1:%d", port)
		listener, err := net.Listen("tcp", address)
		if err != nil {
			findings = append(findings, Finding{
				Code:     "port_occupied",
				Severity: SeverityBlock,
				Message:  "A21 reserved port is occupied",
				Detail:   fmt.Sprintf("%d", port),
			})
			continue
		}
		_ = listener.Close()
	}
	return findings
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/runtimeguard -run TestCheckPorts -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/runtimeguard/ports.go internal/runtimeguard/ports_test.go
git commit -m "feat: guard a21 reserved ports"
```

## Task 5: Add Legacy Path Guard

**Files:**
- Create: `internal/runtimeguard/paths.go`
- Create: `internal/runtimeguard/paths_test.go`

- [ ] **Step 1: Write failing path guard tests**

Create `internal/runtimeguard/paths_test.go`:

```go
package runtimeguard

import "testing"

func TestCheckWorkingDirectoryBlocksLegacyProjectPath(t *testing.T) {
	cfg := DefaultConfig()
	findings := CheckWorkingDirectory(cfg, "/Users/jiyurun/Documents/小马暴力")
	result := NewResult(findings)

	if result.OK {
		t.Fatal("expected legacy working directory to block startup")
	}
	if result.Findings[0].Code != "legacy_cwd" {
		t.Fatalf("code = %q, want legacy_cwd", result.Findings[0].Code)
	}
}

func TestCheckWorkingDirectoryAllowsA21Workspace(t *testing.T) {
	cfg := DefaultConfig()
	result := NewResult(CheckWorkingDirectory(cfg, "/Users/jiyurun/Documents/New project"))
	if !result.OK {
		t.Fatalf("expected A21 workspace to pass, got %#v", result.Findings)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/runtimeguard -run TestCheckWorkingDirectory -v`

Expected: FAIL with `undefined: CheckWorkingDirectory`.

- [ ] **Step 3: Implement path guard**

Create `internal/runtimeguard/paths.go`:

```go
package runtimeguard

import "strings"

func CheckWorkingDirectory(cfg Config, cwd string) []Finding {
	for _, legacyPath := range cfg.LegacyPathParts {
		if strings.HasPrefix(cwd, legacyPath) {
			return []Finding{{
				Code:     "legacy_cwd",
				Severity: SeverityBlock,
				Message:  "A21 runtime refuses to start from a legacy project working directory",
				Detail:   cwd,
			}}
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/runtimeguard -run TestCheckWorkingDirectory -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/runtimeguard/paths.go internal/runtimeguard/paths_test.go
git commit -m "feat: block legacy project working directories"
```

## Task 6: Add Network and Environment Fingerprint

**Files:**
- Create: `internal/runtimeguard/fingerprint.go`
- Create: `internal/runtimeguard/fingerprint_test.go`

- [ ] **Step 1: Write failing fingerprint tests**

Create `internal/runtimeguard/fingerprint_test.go`:

```go
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/runtimeguard -run TestFingerprint -v`

Expected: FAIL with undefined `DetectFingerprint`.

- [ ] **Step 3: Implement fingerprinting**

Create `internal/runtimeguard/fingerprint.go`:

```go
package runtimeguard

import (
	"context"
	"os/exec"
	"strings"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

type Fingerprint struct {
	NetworkInterface    string   `json:"network_interface,omitempty"`
	ExternalDNSMappedIP string   `json:"external_dns_mapped_ip,omitempty"`
	ProxyEnv            []string `json:"proxy_env,omitempty"`
}

const DefaultDNSProbeHost = "example.com"

func (f Fingerprint) UsesTun() bool {
	return strings.HasPrefix(f.NetworkInterface, "utun")
}

func CheckFingerprint(fp Fingerprint) []Finding {
	findings := make([]Finding, 0)
	if fp.NetworkInterface == "" {
		findings = append(findings, Finding{
			Code:     "fingerprint_missing",
			Severity: SeverityBlock,
			Message:  "A21 runtime requires a network interface fingerprint",
			Detail:   "network_interface",
		})
	}
	if fp.ExternalDNSMappedIP == "" {
		findings = append(findings, Finding{
			Code:     "fingerprint_missing",
			Severity: SeverityBlock,
			Message:  "A21 runtime requires an external DNS fingerprint",
			Detail:   "external_dns_mapped_ip",
		})
	}
	return findings
}

func DetectFingerprint(ctx context.Context, runner CommandRunner, env []string) Fingerprint {
	fp := Fingerprint{ProxyEnv: proxyEnv(env)}
	if runner == nil {
		return fp
	}

	if out, err := runner.Run(ctx, "route", "-n", "get", "default"); err == nil {
		fp.NetworkInterface = parseRouteInterface(out)
	}
	if out, err := runner.Run(ctx, "dig", "+short", DefaultDNSProbeHost); err == nil {
		fp.ExternalDNSMappedIP = firstNonEmptyLine(out)
	}
	return fp
}

func proxyEnv(env []string) []string {
	keys := make([]string, 0)
	for _, entry := range env {
		name, _, _ := strings.Cut(entry, "=")
		upper := strings.ToUpper(name)
		if upper == "HTTP_PROXY" || upper == "HTTPS_PROXY" || upper == "ALL_PROXY" || upper == "NO_PROXY" {
			keys = append(keys, name)
		}
	}
	return keys
}

func parseRouteInterface(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "interface:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
		}
	}
	return ""
}

func firstNonEmptyLine(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/runtimeguard -run TestFingerprint -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/runtimeguard/fingerprint.go internal/runtimeguard/fingerprint_test.go
git commit -m "feat: record a21 environment fingerprint"
```

## Task 7: Compose Runtime Preflight

**Files:**
- Create: `internal/runtimeguard/preflight.go`
- Create: `internal/runtimeguard/preflight_test.go`

- [ ] **Step 1: Write failing preflight tests**

Create `internal/runtimeguard/preflight_test.go`:

```go
package runtimeguard

import (
	"context"
	"testing"
)

func TestRunPreflightBlocksLegacyEnv(t *testing.T) {
	cfg := DefaultConfig()
	input := PreflightInput{
		Config: cfg,
		Env:    []string{"X21_BACKEND=http://127.0.0.1:8000"},
		CWD:    "/Users/jiyurun/Documents/New project",
	}

	result := RunPreflight(context.Background(), input)
	if result.OK {
		t.Fatal("expected preflight to block legacy env")
	}
	if result.Findings[0].Code != "legacy_env" {
		t.Fatalf("first code = %q, want legacy_env", result.Findings[0].Code)
	}
}

func TestRunPreflightPassesCleanConfigWithNoPortChecks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ReservedPorts = nil
	input := PreflightInput{
		Config: cfg,
		Env:    []string{"A21_MODE=dev"},
		CWD:    "/Users/jiyurun/Documents/New project",
	}

	result := RunPreflight(context.Background(), input)
	if !result.OK {
		t.Fatalf("expected clean preflight, got %#v", result.Findings)
	}
}

func TestRunPreflightReportBlocksMissingFingerprint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ReservedPorts = nil
	report := RunPreflightReport(context.Background(), PreflightInput{
		Config: cfg,
		Env:    []string{"A21_MODE=dev"},
		CWD:    "/Users/jiyurun/Documents/New project",
	})

	if report.Result.OK {
		t.Fatal("expected missing fingerprint to block preflight report")
	}
	if report.Result.Findings[0].Code != "fingerprint_missing" {
		t.Fatalf("first code = %q, want fingerprint_missing", report.Result.Findings[0].Code)
	}
}

func TestRunPreflightReportAllowsCompleteFingerprint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ReservedPorts = nil
	report := RunPreflightReport(context.Background(), PreflightInput{
		Config: cfg,
		Env:    []string{"A21_MODE=dev"},
		CWD:    "/Users/jiyurun/Documents/New project",
		Runner: fakeRunner{
			"route -n get default":              "interface: en0\n",
			"dig +short " + DefaultDNSProbeHost: "198.18.0.145\n",
		},
	})

	if !report.Result.OK {
		t.Fatalf("expected complete fingerprint to pass, got %#v", report.Result.Findings)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/runtimeguard -run TestRunPreflight -v`

Expected: FAIL with undefined `PreflightInput` or `RunPreflight`.

- [ ] **Step 3: Implement preflight**

Create `internal/runtimeguard/preflight.go`:

```go
package runtimeguard

import "context"

type PreflightInput struct {
	Config Config
	Env    []string
	CWD    string
	Runner CommandRunner
}

type PreflightReport struct {
	Result      Result      `json:"result"`
	Fingerprint Fingerprint `json:"fingerprint"`
}

func RunPreflight(ctx context.Context, input PreflightInput) Result {
	cfg := input.Config
	if cfg.ProjectName == "" {
		cfg = DefaultConfig()
	}

	findings := make([]Finding, 0)
	findings = append(findings, CheckEnv(cfg, input.Env)...)
	findings = append(findings, CheckWorkingDirectory(cfg, input.CWD)...)
	findings = append(findings, CheckPorts(cfg.ReservedPorts)...)

	return NewResult(findings)
}

func RunPreflightReport(ctx context.Context, input PreflightInput) PreflightReport {
	fingerprint := DetectFingerprint(ctx, input.Runner, input.Env)
	result := RunPreflight(ctx, input)
	findings := append([]Finding{}, result.Findings...)
	findings = append(findings, CheckFingerprint(fingerprint)...)
	return PreflightReport{
		Result:      NewResult(findings),
		Fingerprint: fingerprint,
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/runtimeguard -run TestRunPreflight -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/runtimeguard/preflight.go internal/runtimeguard/preflight_test.go
git commit -m "feat: compose a21 runtime preflight"
```

## Task 8: Replace Thin CLI with Version and Preflight Commands

**Files:**
- Create: `internal/app/app.go`
- Create: `internal/app/app_test.go`
- Modify: `cmd/a21/main.go`

- [ ] **Step 1: Write failing CLI tests**

Create `internal/app/app_test.go`:

```go
package app

import (
	"bytes"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := stdout.String(); got != "A21 0.1.0-dev (a21-core)\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"nope"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/app -run TestRun -v`

Expected: FAIL because package `internal/app` has no implementation.

- [ ] **Step 3: Implement CLI package**

Create `internal/app/app.go`:

```go
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"a21.local/a21/internal/buildinfo"
	"a21.local/a21/internal/runtimeguard"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		args = []string{"version"}
	}

	switch args[0] {
	case "version":
		fmt.Fprintf(stdout, "%s %s (%s)\n", buildinfo.ProjectName, buildinfo.Version, buildinfo.ServiceName)
		return 0
	case "preflight":
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get working directory: %v\n", err)
			return 1
		}
		report := runtimeguard.RunPreflightReport(context.Background(), runtimeguard.PreflightInput{
			Config: runtimeguard.DefaultConfig(),
			Env:    os.Environ(),
			CWD:    cwd,
			Runner: runtimeguard.OSRunner{},
		})
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "encode preflight report: %v\n", err)
			return 1
		}
		if !report.Result.OK {
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}
```

- [ ] **Step 4: Update main**

Replace `cmd/a21/main.go` with:

```go
package main

import (
	"os"

	"a21.local/a21/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
```

- [ ] **Step 5: Run CLI tests**

Run: `go test ./internal/app -run TestRun -v`

Expected: PASS.

- [ ] **Step 6: Run CLI manually**

Run: `go run ./cmd/a21 version`

Expected output:

```text
A21 0.1.0-dev (a21-core)
```

Run: `go run ./cmd/a21 preflight`

Expected: JSON report. In the current workspace it should not report X21/V21 env leakage unless those env vars are set in this shell. It may report occupied reserved ports only if a future local process binds an A21 port.

- [ ] **Step 7: Commit**

```bash
git add cmd/a21/main.go internal/app
git commit -m "feat: add a21 version and preflight cli"
```

## Task 9: Add Versioned A21 Device Protocol Types

**Files:**
- Create: `internal/protocol/message.go`
- Create: `internal/protocol/message_test.go`

- [ ] **Step 1: Write failing protocol tests**

Create `internal/protocol/message_test.go`:

```go
package protocol

import (
	"encoding/json"
	"testing"
)

func TestEnvelopeJSONUsesA21Protocol(t *testing.T) {
	msg := Envelope{
		Protocol: ProtocolVersion,
		DeviceID: "stackchan-home-01",
		Kind:     KindAudioFrame,
		Seq:      42,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"protocol":"a21.device.v1","device_id":"stackchan-home-01","kind":"audio.frame","seq":42}` {
		t.Fatalf("json = %s", data)
	}
}

func TestMessageKindsCoverCoreDeviceLoop(t *testing.T) {
	tests := map[string]Kind{
		"audio.frame":       KindAudioFrame,
		"device.event":      KindDeviceEvent,
		"assistant.state":   KindAssistantState,
		"screen.expression": KindScreenExpression,
		"motion.command":    KindMotionCommand,
	}

	for want, got := range tests {
		if string(got) != want {
			t.Fatalf("kind = %q, want %q", got, want)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/protocol -v`

Expected: FAIL because package `internal/protocol` has no implementation.

- [ ] **Step 3: Implement message types**

Create `internal/protocol/message.go`:

```go
package protocol

const ProtocolVersion = "a21.device.v1"

type Kind string

const (
	KindAudioFrame       Kind = "audio.frame"
	KindDeviceEvent      Kind = "device.event"
	KindAssistantState   Kind = "assistant.state"
	KindScreenExpression Kind = "screen.expression"
	KindMotionCommand    Kind = "motion.command"
)

type Envelope struct {
	Protocol string `json:"protocol"`
	DeviceID string `json:"device_id"`
	Kind     Kind   `json:"kind"`
	Seq      uint64 `json:"seq"`
)
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/protocol -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/protocol
git commit -m "feat: define a21 device protocol envelope"
```

## Task 10: Add Provider Contracts

**Files:**
- Create: `internal/providers/contracts.go`
- Create: `internal/providers/contracts_test.go`

- [ ] **Step 1: Write failing provider contract tests**

Create `internal/providers/contracts_test.go`:

```go
package providers

import (
	"context"
	"testing"
)

type fakeASR struct{}

func (fakeASR) Transcribe(ctx context.Context, in AudioChunk) (TranscriptEvent, error) {
	return TranscriptEvent{Text: "hello", Final: true}, nil
}

type fakeLLM struct{}

func (fakeLLM) Respond(ctx context.Context, in DialogueTurn) (<-chan TextDelta, error) {
	ch := make(chan TextDelta, 1)
	ch <- TextDelta{Text: "hi", Final: true}
	close(ch)
	return ch, nil
}

type fakeTTS struct{}

func (fakeTTS) Synthesize(ctx context.Context, in TTSRequest) (<-chan AudioChunk, error) {
	ch := make(chan AudioChunk, 1)
	ch <- AudioChunk{Codec: "opus", Payload: []byte{1}}
	close(ch)
	return ch, nil
}

type fakeKnowledge struct{}

func (fakeKnowledge) Retrieve(ctx context.Context, query KnowledgeQuery) ([]KnowledgeHit, error) {
	return []KnowledgeHit{{ID: "doc-1", Text: "fact", Score: 0.9}}, nil
}

func TestProviderContractsCompile(t *testing.T) {
	var _ ASRProvider = fakeASR{}
	var _ LLMProvider = fakeLLM{}
	var _ TTSProvider = fakeTTS{}
	var _ KnowledgeProvider = fakeKnowledge{}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/providers -v`

Expected: FAIL because package `internal/providers` has no implementation.

- [ ] **Step 3: Implement provider contracts**

Create `internal/providers/contracts.go`:

```go
package providers

import "context"

type AudioChunk struct {
	Codec      string
	SampleRate int
	Payload    []byte
}

type TranscriptEvent struct {
	Text  string
	Final bool
}

type DialogueTurn struct {
	SessionID string
	UserText  string
	Context   []KnowledgeHit
}

type TextDelta struct {
	Text  string
	Final bool
}

type TTSRequest struct {
	SessionID string
	Text      string
	VoiceID   string
}

type KnowledgeQuery struct {
	SessionID string
	Text      string
	Limit     int
}

type KnowledgeHit struct {
	ID    string
	Text  string
	Score float64
}

type ASRProvider interface {
	Transcribe(ctx context.Context, in AudioChunk) (TranscriptEvent, error)
}

type LLMProvider interface {
	Respond(ctx context.Context, in DialogueTurn) (<-chan TextDelta, error)
}

type TTSProvider interface {
	Synthesize(ctx context.Context, in TTSRequest) (<-chan AudioChunk, error)
}

type KnowledgeProvider interface {
	Retrieve(ctx context.Context, query KnowledgeQuery) ([]KnowledgeHit, error)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/providers -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/providers
git commit -m "feat: define a21 provider contracts"
```

## Task 11: Add Makefile and Verification Target

**Files:**
- Create: `Makefile`
- Modify: `docs/a21/00-project-charter-and-home-baseline.md`

- [ ] **Step 1: Write the Makefile**

Create `Makefile`:

```makefile
.PHONY: test verify preflight

test:
	go test ./...

preflight:
	go run ./cmd/a21 preflight

verify:
	go test ./...
	git diff --check
```

- [ ] **Step 2: Update the baseline document**

Append this section to `docs/a21/00-project-charter-and-home-baseline.md`:

````markdown
## Phase 1 Implementation Pointer

Phase 1 starts the clean A21 skeleton from `docs/superpowers/plans/2026-05-29-a21-phase1-clean-skeleton.md`.

The first verification target is:

```bash
make verify
```
````

- [ ] **Step 3: Run verification**

Run: `make verify`

Expected:

```text
go test ./...
...
git diff --check
```

The command exits with code 0.

- [ ] **Step 4: Commit**

```bash
git add Makefile docs/a21/00-project-charter-and-home-baseline.md
git commit -m "chore: add a21 phase1 verification target"
```

## Task 12: Final Phase 1 Skeleton Review

**Files:**
- Inspect: `go.mod`
- Inspect: `cmd/a21/main.go`
- Inspect: `internal/runtimeguard`
- Inspect: `internal/protocol`
- Inspect: `internal/providers`
- Inspect: `Makefile`

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`

Expected: PASS for every package.

- [ ] **Step 2: Run the CLI version command**

Run: `go run ./cmd/a21 version`

Expected:

```text
A21 0.1.0-dev (a21-core)
```

- [ ] **Step 3: Run preflight**

Run: `go run ./cmd/a21 preflight`

Expected: JSON output containing `result` and `fingerprint`. If the current shell contains legacy `X21_*` or `V21_*` variables, A21 endpoint env vars pointing at legacy ports, occupied A21 reserved ports, or missing fingerprint prerequisites such as `route`/`dig`, the command exits with code 1 and lists the blocking findings. A clean shell with available fingerprint commands and free A21 ports exits with code 0.

- [ ] **Step 4: Check formatting and diffs**

Run:

```bash
gofmt -w cmd internal
git diff --check
git status --short
```

Expected: no whitespace errors. `git status --short` shows only intentional files before the final commit.

- [ ] **Step 5: Final commit if any files remain staged or unstaged**

```bash
git add go.mod cmd internal Makefile docs/a21/00-project-charter-and-home-baseline.md
git commit -m "feat: establish a21 clean skeleton"
```

## Self-Review

Spec coverage:

- A21 namespace: Task 2.
- Runtime preflight: Tasks 3-8.
- Reserved port checker: Task 4.
- Environment fingerprint: Task 6.
- Provider contracts: Task 10.
- Minimal protocol types: Task 9.
- Local verification: Task 11 and Task 12.
- No X21/V21 accidental connection: Tasks 3, 5, and 7.

Execution guardrails:

- This plan does not stop, mutate, or depend on X21/V21 processes.
- This plan does not read secrets.
- This plan does not call DeepSeek, Bailian, DashScope, OpenAI, Qdrant, PostgreSQL, or Ollama.
- This plan uses only Go standard library in Phase 1.
