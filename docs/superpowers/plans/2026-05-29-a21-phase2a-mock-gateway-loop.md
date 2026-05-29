# A21 Phase 2A Mock Gateway Loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first no-hardware A21 interaction loop: a Go standard-library mock gateway that returns deterministic A21 control events for a simulated turn.

**Architecture:** Keep the current Go-first foundation. Extend the protocol without breaking the Phase 1 wire test, add an `internal/gateway` package with pure `net/http` handlers, and expose a CLI `gateway` command for local smoke use. Real WebSocket audio/control transport and browser simulator stay out of this slice.

**Tech Stack:** Go 1.26, standard library only, `net/http`, `httptest`, existing `internal/protocol`, existing `internal/app`.

---

## Scope Check

This is Phase 2A, not full Phase 2. It deliberately implements:

- protocol metadata needed by a gateway loop
- control event payload types
- deterministic mock turn endpoint
- deterministic mock interrupt endpoint
- health endpoint
- CLI gateway command

It does not implement:

- real audio capture/playback
- WebSocket transport
- browser simulator
- real ASR/LLM/TTS providers
- V21 adapter calls
- OpenTelemetry/Prometheus server

## File Structure

- Modify `internal/protocol/message.go`: add trace/session/sent time/payload fields and mode/expression/control payload types.
- Modify `internal/protocol/message_test.go`: preserve Phase 1 JSON contract and add metadata/control-event tests.
- Create `internal/gateway/server.go`: standard-library HTTP gateway handler.
- Create `internal/gateway/server_test.go`: health, mock turn, and interrupt tests.
- Modify `internal/app/app.go`: add `gateway` command with `--addr`.
- Modify `internal/app/app_test.go`: add gateway command argument validation tests.
- Modify `Makefile`: add `gateway` target.
- Create `docs/engineering/PHASE2A_GATEWAY_MOCK.md`: describe current mock loop and next boundary.

## Task 1: Extend Protocol For Gateway Mock Loop

**Files:**
- Modify: `internal/protocol/message.go`
- Modify: `internal/protocol/message_test.go`

- [ ] **Step 1: Add failing protocol tests**

Append to `internal/protocol/message_test.go`:

```go
func TestEnvelopeSupportsTraceSessionAndPayload(t *testing.T) {
	payload := json.RawMessage(`{"state":"listening","mode":"workmate"}`)
	msg := Envelope{
		Protocol: ProtocolVersion,
		DeviceID: "stackchan-sim-001",
		Kind:     KindControlEvent,
		Seq:      1,
		TraceID:  "a21-trace-000001",
		SessionID: "a21-session-000001",
		SentAtMS:  1234,
		Payload:  payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"trace_id":"a21-trace-000001"`) {
		t.Fatalf("json missing trace_id: %s", data)
	}
	if !strings.Contains(string(data), `"payload":{"state":"listening","mode":"workmate"}`) {
		t.Fatalf("json missing payload: %s", data)
	}
}

func TestControlEventPayloadStates(t *testing.T) {
	event := ControlEventPayload{
		State: ExpressionListening,
		Mode:  ModeWorkmate,
		Text:  "我在听",
		Final: false,
	}
	if event.State != "listening" {
		t.Fatalf("State = %q, want listening", event.State)
	}
	if event.Mode != "workmate" {
		t.Fatalf("Mode = %q, want workmate", event.Mode)
	}
}
```

Also add `"strings"` to the import block.

- [ ] **Step 2: Verify red**

Run: `go test ./internal/protocol -run 'TestEnvelopeSupportsTraceSessionAndPayload|TestControlEventPayloadStates' -v`

Expected: FAIL with undefined `KindControlEvent`, `TraceID`, `SessionID`, `SentAtMS`, `Payload`, `ControlEventPayload`, `ExpressionListening`, or `ModeWorkmate`.

- [ ] **Step 3: Implement protocol extension**

Replace `internal/protocol/message.go` with:

```go
package protocol

import "encoding/json"

const ProtocolVersion = "a21.device.v1"

type Kind string

const (
	KindAudioFrame       Kind = "audio.frame"
	KindDeviceEvent      Kind = "device.event"
	KindAssistantState   Kind = "assistant.state"
	KindScreenExpression Kind = "screen.expression"
	KindMotionCommand    Kind = "motion.command"
	KindControlEvent     Kind = "control.event"
)

type Envelope struct {
	Protocol  string          `json:"protocol"`
	DeviceID  string          `json:"device_id"`
	Kind      Kind            `json:"kind"`
	Seq       uint64          `json:"seq"`
	TraceID   string          `json:"trace_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	SentAtMS  int64           `json:"sent_at_ms,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type Mode string

const (
	ModeWorkmate      Mode = "workmate"
	ModeProfessional  Mode = "professional"
	ModeLocalFallback Mode = "local_fallback"
	ModeError         Mode = "error"
)

type ExpressionState string

const (
	ExpressionIdle         ExpressionState = "idle"
	ExpressionListening    ExpressionState = "listening"
	ExpressionThinking     ExpressionState = "thinking"
	ExpressionSpeaking     ExpressionState = "speaking"
	ExpressionInterrupted ExpressionState = "interrupted"
	ExpressionProfessional ExpressionState = "professional"
	ExpressionError        ExpressionState = "error"
)

type ControlEventPayload struct {
	State    ExpressionState `json:"state"`
	Mode     Mode            `json:"mode"`
	Text     string          `json:"text,omitempty"`
	Final    bool            `json:"final,omitempty"`
	StreamID string          `json:"stream_id,omitempty"`
}
```

- [ ] **Step 4: Verify green**

Run: `go test ./internal/protocol -v`

Expected: PASS.

## Task 2: Add Mock Gateway HTTP Handler

**Files:**
- Create: `internal/gateway/server.go`
- Create: `internal/gateway/server_test.go`

- [ ] **Step 1: Add failing gateway tests**

Create `internal/gateway/server_test.go`:

```go
package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"a21.local/a21/internal/protocol"
)

func TestHealthz(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"service":"a21-gateway"`)) {
		t.Fatalf("body missing service: %s", rec.Body.String())
	}
}

func TestMockTurnReturnsDeterministicStateSequence(t *testing.T) {
	server := NewServer()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"先说，我在","mode":"workmate"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.TraceID != "a21-trace-000001" {
		t.Fatalf("TraceID = %q", response.TraceID)
	}
	if len(response.Events) != 3 {
		t.Fatalf("events = %d, want 3", len(response.Events))
	}
	wantStates := []protocol.ExpressionState{
		protocol.ExpressionListening,
		protocol.ExpressionThinking,
		protocol.ExpressionSpeaking,
	}
	for i, want := range wantStates {
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(response.Events[i].Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.State != want {
			t.Fatalf("event %d state = %q, want %q", i, payload.State, want)
		}
		if response.Events[i].TraceID != response.TraceID {
			t.Fatalf("event %d trace = %q, want %q", i, response.Events[i].TraceID, response.TraceID)
		}
	}
}

func TestMockInterruptReturnsInterruptedThenListening(t *testing.T) {
	server := NewServer()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","trace_id":"a21-trace-000009","session_id":"a21-session-000009"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-interrupt", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(response.Events))
	}
	var first protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[0].Payload, &first); err != nil {
		t.Fatal(err)
	}
	if first.State != protocol.ExpressionInterrupted {
		t.Fatalf("first state = %q, want interrupted", first.State)
	}
}
```

- [ ] **Step 2: Verify red**

Run: `go test ./internal/gateway -v`

Expected: FAIL because package or symbols are missing.

- [ ] **Step 3: Implement gateway server**

Create `internal/gateway/server.go`:

```go
package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"a21.local/a21/internal/buildinfo"
	"a21.local/a21/internal/protocol"
)

type Server struct {
	mu   sync.Mutex
	next uint64
	now  func() time.Time
}

type MockTurnRequest struct {
	DeviceID  string        `json:"device_id"`
	Text      string        `json:"text,omitempty"`
	Mode      protocol.Mode `json:"mode,omitempty"`
	TraceID   string        `json:"trace_id,omitempty"`
	SessionID string        `json:"session_id,omitempty"`
}

type MockTurnResponse struct {
	TraceID   string              `json:"trace_id"`
	SessionID string              `json:"session_id"`
	DeviceID  string              `json:"device_id"`
	Events    []protocol.Envelope `json:"events"`
}

func NewServer() *Server {
	return &Server{now: time.Now}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/v1/mock-turn", s.handleMockTurn)
	mux.HandleFunc("/v1/mock-interrupt", s.handleMockInterrupt)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "a21-gateway",
		"status":  "ok",
		"version": buildinfo.Version,
	})
}

func (s *Server) handleMockTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req MockTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	events := s.controlSequence(req.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
		{State: protocol.ExpressionListening, Mode: req.Mode, Text: "我在听"},
		{State: protocol.ExpressionThinking, Mode: req.Mode, Text: "我想一下"},
		{State: protocol.ExpressionSpeaking, Mode: req.Mode, Text: "先说，我在。", Final: true, StreamID: "a21-mock-stream-000001"},
	})
	writeJSON(w, http.StatusOK, MockTurnResponse{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID, Events: events})
}

func (s *Server) handleMockInterrupt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req MockTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	mode := req.Mode
	if mode == "" {
		mode = protocol.ModeWorkmate
	}
	events := s.controlSequence(req.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
		{State: protocol.ExpressionInterrupted, Mode: mode, Text: "好，我听新的。"},
		{State: protocol.ExpressionListening, Mode: mode, Text: "你说。"},
	})
	writeJSON(w, http.StatusOK, MockTurnResponse{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID, Events: events})
}

func (s *Server) ids(traceID string, sessionID string) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if traceID == "" || sessionID == "" {
		s.next++
	}
	if traceID == "" {
		traceID = fmt.Sprintf("a21-trace-%06d", s.next)
	}
	if sessionID == "" {
		sessionID = fmt.Sprintf("a21-session-%06d", s.next)
	}
	return traceID, sessionID
}

func (s *Server) controlSequence(deviceID string, traceID string, sessionID string, payloads []protocol.ControlEventPayload) []protocol.Envelope {
	events := make([]protocol.Envelope, 0, len(payloads))
	sentAt := s.now().UnixMilli()
	for i, payload := range payloads {
		data, _ := json.Marshal(payload)
		events = append(events, protocol.Envelope{
			Protocol:  protocol.ProtocolVersion,
			DeviceID:  deviceID,
			Kind:      protocol.KindControlEvent,
			Seq:       uint64(i + 1),
			TraceID:   traceID,
			SessionID: sessionID,
			SentAtMS:  sentAt + int64(i),
			Payload:   data,
		})
	}
	return events
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}
```

- [ ] **Step 4: Verify green**

Run: `go test ./internal/gateway -v`

Expected: PASS.

## Task 3: Add Gateway CLI Command

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `Makefile`

- [ ] **Step 1: Add failing CLI tests**

Append to `internal/app/app_test.go`:

```go
func TestRunGatewayHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"gateway", "--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "a21 gateway") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunGatewayRejectsUnknownFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"gateway", "--wat"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}
```

- [ ] **Step 2: Verify red**

Run: `go test ./internal/app -run TestRunGateway -v`

Expected: FAIL because `gateway` is unknown.

- [ ] **Step 3: Implement CLI command**

Modify `internal/app/app.go`:

Add imports:

```go
	"net/http"
	"strings"

	"a21.local/a21/internal/gateway"
```

Add switch case before `preflight`:

```go
	case "gateway":
		return runGateway(args[1:], stdout, stderr)
```

Append:

```go
func runGateway(args []string, stdout io.Writer, stderr io.Writer) int {
	addr := "127.0.0.1:21080"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 gateway --addr 127.0.0.1:21080")
			return 0
		case "--addr":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--addr requires a value")
				return 2
			}
			i++
			addr = args[i]
		default:
			fmt.Fprintf(stderr, "unknown gateway option %q\n", args[i])
			return 2
		}
	}

	fmt.Fprintf(stdout, "a21 gateway listening on %s\n", addr)
	if err := http.ListenAndServe(addr, gateway.NewServer().Handler()); err != nil {
		fmt.Fprintf(stderr, "gateway: %v\n", err)
		return 1
	}
	return 0
}
```

- [ ] **Step 4: Add Makefile target**

Modify `Makefile`:

```makefile
.PHONY: test verify preflight doctor gateway

gateway:
	go run ./cmd/a21 gateway --addr 127.0.0.1:21080
```

- [ ] **Step 5: Verify green**

Run: `go test ./internal/app -run TestRunGateway -v`

Expected: PASS.

## Task 4: Document Phase 2A

**Files:**
- Create: `docs/engineering/PHASE2A_GATEWAY_MOCK.md`
- Modify: `docs/engineering/A21_CODEX_AUDIT.md`

- [ ] **Step 1: Create Phase 2A doc**

Create `docs/engineering/PHASE2A_GATEWAY_MOCK.md`:

```markdown
# A21 Phase 2A Gateway Mock

## Purpose

Phase 2A proves a no-hardware interaction loop without introducing WebSocket, browser UI, real audio, or real providers.

## Current Endpoints

- `GET /healthz`
- `POST /v1/mock-turn`
- `POST /v1/mock-interrupt`

## Run

```bash
make gateway
```

## Smoke

```bash
curl -s http://127.0.0.1:21080/healthz
curl -s -X POST http://127.0.0.1:21080/v1/mock-turn \
  -d '{"device_id":"stackchan-sim-001","text":"先说，我在","mode":"workmate"}'
```

## Boundaries

This phase does not claim real-time audio. It creates deterministic state/control events so Phase 2B can add WebSocket transport and a simulator against stable contracts.
```

- [ ] **Step 2: Update audit doc gap**

In `docs/engineering/A21_CODEX_AUDIT.md`, change `No gateway HTTP server yet.` to `Phase 2A adds a mock gateway HTTP handler; real audio/control WebSocket remains future work.`

- [ ] **Step 3: Verify docs**

Run: `git diff --check`

Expected: PASS.

## Task 5: Final Verification

**Files:**
- All files touched in Tasks 1-4.

- [ ] **Step 1: Run full tests**

Run: `go test -count=1 ./...`

Expected: PASS.

- [ ] **Step 2: Run make verify**

Run: `make verify`

Expected: PASS.

- [ ] **Step 3: Smoke gateway help**

Run: `go run ./cmd/a21 gateway --help`

Expected:

```text
a21 gateway --addr 127.0.0.1:21080
```

- [ ] **Step 4: Optional local server smoke**

Run in one terminal:

```bash
go run ./cmd/a21 gateway --addr 127.0.0.1:21080
```

Run in another terminal:

```bash
curl -s http://127.0.0.1:21080/healthz
```

Expected JSON contains `"service":"a21-gateway"`.

- [ ] **Step 5: Commit**

```bash
git add Makefile cmd internal docs/engineering docs/superpowers/plans/2026-05-29-a21-phase2a-mock-gateway-loop.md
git commit -m "feat: add a21 mock gateway loop"
```
