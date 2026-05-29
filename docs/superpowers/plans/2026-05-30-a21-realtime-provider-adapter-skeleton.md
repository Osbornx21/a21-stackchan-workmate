# A21 Realtime Provider Adapter Skeleton Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a testable realtime WebSocket provider transport skeleton without claiming OpenAI or Doubao production voice readiness.

**Architecture:** A21 keeps Gateway and firmware isolated from provider-specific details. The new provider layer builds redacted connection plans, uses the existing `github.com/coder/websocket` dependency through a narrow dialer interface, and sends only generic JSON realtime events such as session update and cancel. Real audio/event semantics remain behind later provider-specific adapters.

**Tech Stack:** Go, `internal/providers`, `github.com/coder/websocket`, table-driven tests, existing A21 provider network policy.

---

### Task 1: Redacted Realtime Plan

**Files:**
- Create: `internal/providers/realtime_test.go`
- Create: `internal/providers/realtime.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRealtimeWebSocketPlanFromEnvBuildsOpenAIPlanWithoutLeakingSecrets(t *testing.T) {
	report := RealtimeWebSocketPlanFromEnv([]string{
		"A21_PROVIDER_PRIMARY=openai_realtime",
		"A21_OPENAI_API_KEY=sk-a21-secret",
		"A21_OPENAI_REALTIME_MODEL=gpt-realtime-2",
	}, "openai_realtime")

	if report.Status != ProviderSmokeReady {
		t.Fatalf("status = %q, want ready", report.Status)
	}
	if report.EndpointHost != "api.openai.com" {
		t.Fatalf("endpoint host = %q, want api.openai.com", report.EndpointHost)
	}
	rendered := mustJSON(t, report)
	for _, forbidden := range []string{"sk-a21-secret", "gpt-realtime-2", "Authorization"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("plan leaked %q: %s", forbidden, rendered)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers -run TestRealtimeWebSocketPlanFromEnvBuildsOpenAIPlanWithoutLeakingSecrets -count=1`

Expected: FAIL because `RealtimeWebSocketPlanFromEnv` is undefined.

- [ ] **Step 3: Write minimal implementation**

Create `RealtimeWebSocketPlanFromEnv` with explicit support for `openai_realtime`, required env checks, endpoint host redaction, and legacy-provider rejection.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers -run TestRealtimeWebSocketPlanFromEnvBuildsOpenAIPlanWithoutLeakingSecrets -count=1`

Expected: PASS.

### Task 2: WebSocket Event Transport

**Files:**
- Modify: `internal/providers/realtime_test.go`
- Modify: `internal/providers/realtime.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRealtimeWebSocketSessionSendsSessionUpdateAndCancel(t *testing.T) {
	conn := &fakeRealtimeConn{}
	adapter := NewRealtimeWebSocketAdapter(RealtimeWebSocketConfig{
		Provider: "openai_realtime",
		URL:      "wss://api.openai.com/v1/realtime?model=gpt-realtime-2",
		Headers:  map[string]string{"Authorization": "Bearer sk-a21-secret"},
	}, fakeRealtimeDialer{conn: conn})

	session, err := adapter.Connect(context.Background(), map[string]any{
		"type": "realtime",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Cancel(context.Background(), VoiceCancelRequest{Reason: CancelBargeIn}); err != nil {
		t.Fatal(err)
	}

	if got, want := conn.messages[0]["type"], "session.update"; got != want {
		t.Fatalf("first event type = %v, want %s", got, want)
	}
	if got, want := conn.messages[1]["type"], "response.cancel"; got != want {
		t.Fatalf("second event type = %v, want %s", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers -run TestRealtimeWebSocketSessionSendsSessionUpdateAndCancel -count=1`

Expected: FAIL because the adapter/session types are undefined.

- [ ] **Step 3: Write minimal implementation**

Add `RealtimeDialer`, `RealtimeConn`, `RealtimeWebSocketAdapter`, and `RealtimeWebSocketSession`. The adapter must write JSON events via an injected connection and avoid logs/reports that include header values.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers -run TestRealtimeWebSocketSessionSendsSessionUpdateAndCancel -count=1`

Expected: PASS.

### Task 3: Documentation and Verification

**Files:**
- Create: `docs/engineering/PHASE4D_REALTIME_ADAPTER_SKELETON.md`
- Modify: `docs/engineering/PHASE4C_PROVIDER_SMOKE.md`

- [ ] **Step 1: Document boundaries**

Document that Phase 4D adds a transport skeleton only, using official OpenAI Realtime WebSocket guidance for server-to-server sessions and keeping Doubao realtime as a documented candidate pending provider-specific implementation.

- [ ] **Step 2: Run complete verification**

Run:

```bash
make verify
make release-check
```

Expected: both pass.

- [ ] **Step 3: Preserve firmware discipline**

After a clean verified commit, run:

```bash
make firmware-package
go run ./cmd/a21 firmware-artifact-check --artifact <new-artifact>
go run ./cmd/a21 firmware-upload-check --artifact <new-artifact> --port /dev/cu.usbmodem1101 --commit $(git rev-parse --short=12 HEAD)
```

Expected: artifact identity matches A21/version/board/commit and upload remains a dry-run unless explicit flashing is enabled.
