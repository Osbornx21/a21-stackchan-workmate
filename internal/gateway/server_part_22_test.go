package gateway

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
)

func readAllFilesUnder(t *testing.T, root string) string {
	t.Helper()
	var out strings.Builder
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out.Write(data)
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func webSocketURL(serverURL string, path string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http") + path
}

type scriptedVoiceProvider struct {
	events []providers.VoiceEvent
}

func (p scriptedVoiceProvider) Name() string {
	return "scripted"
}

func (p scriptedVoiceProvider) StartTurn(ctx context.Context, req providers.VoiceTurnRequest) (<-chan providers.VoiceEvent, error) {
	events := make(chan providers.VoiceEvent, len(p.events))
	defer close(events)
	for _, event := range p.events {
		event.Session = req.Session
		events <- event
	}
	return events, nil
}

func (p scriptedVoiceProvider) Cancel(ctx context.Context, req providers.VoiceCancelRequest) (<-chan providers.VoiceEvent, error) {
	events := make(chan providers.VoiceEvent, 1)
	defer close(events)
	events <- providers.VoiceEvent{Session: req.Session, Kind: providers.VoiceEventCancelled, Text: "cancelled", Final: true}
	return events, nil
}

func (p scriptedVoiceProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{Provider: p.Name(), Status: providers.VoiceProviderHealthy, Configured: true, Realtime: true}, ctx.Err()
}

func (p scriptedVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

type unavailableVoiceProvider struct{}

func (p unavailableVoiceProvider) Name() string {
	return "unavailable"
}

func (p unavailableVoiceProvider) StartTurn(ctx context.Context, req providers.VoiceTurnRequest) (<-chan providers.VoiceEvent, error) {
	return nil, providers.ErrVoiceProviderUnavailable
}

func (p unavailableVoiceProvider) Cancel(ctx context.Context, req providers.VoiceCancelRequest) (<-chan providers.VoiceEvent, error) {
	return nil, providers.ErrVoiceProviderUnavailable
}

func (p unavailableVoiceProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{
		Provider:   p.Name(),
		Status:     providers.VoiceProviderUnavailable,
		Configured: false,
		Realtime:   false,
		Detail:     "test unavailable",
	}, providers.ErrVoiceProviderUnavailable
}

func (p unavailableVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

type capturingVoiceProvider struct {
	startEvents   []providers.VoiceEvent
	startCalls    int
	cancelRequest providers.VoiceCancelRequest
}

func (p *capturingVoiceProvider) Name() string {
	return "capturing"
}

func (p *capturingVoiceProvider) StartTurn(ctx context.Context, req providers.VoiceTurnRequest) (<-chan providers.VoiceEvent, error) {
	p.startCalls++
	events := make(chan providers.VoiceEvent, len(p.startEvents))
	defer close(events)
	for _, event := range p.startEvents {
		event.Session = req.Session
		events <- event
	}
	return events, nil
}

func (p *capturingVoiceProvider) Cancel(ctx context.Context, req providers.VoiceCancelRequest) (<-chan providers.VoiceEvent, error) {
	p.cancelRequest = req
	events := make(chan providers.VoiceEvent, 1)
	defer close(events)
	events <- providers.VoiceEvent{
		Session:  req.Session,
		Kind:     providers.VoiceEventCancelled,
		Text:     "capturing cancelled",
		Final:    true,
		StreamID: req.StreamID,
	}
	return events, nil
}

func (p *capturingVoiceProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{Provider: p.Name(), Status: providers.VoiceProviderHealthy, Configured: true, Realtime: true}, ctx.Err()
}

func (p *capturingVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

type capturingRealtimeAudioProvider struct {
	capturingVoiceProvider
	startCalls    int
	session       providers.VoiceSession
	sessionHandle *capturingRealtimeVoiceSession
	closed        chan struct{}
}

func (p *capturingRealtimeAudioProvider) Name() string {
	return "capturing-realtime-audio"
}

func (p *capturingRealtimeAudioProvider) StartRealtimeSession(ctx context.Context, session providers.VoiceSession) (providers.RealtimeVoiceSession, error) {
	p.startCalls++
	p.session = session
	p.sessionHandle = &capturingRealtimeVoiceSession{closed: p.closed, events: make(chan providers.VoiceEvent, 4)}
	return p.sessionHandle, ctx.Err()
}

func (p *capturingRealtimeAudioProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{Provider: p.Name(), Status: providers.VoiceProviderHealthy, Configured: true, Realtime: true}, ctx.Err()
}

type capturingRealtimeVoiceSession struct {
	audioChunks []protocol.AudioChunk
	commitCalls int
	cancelCalls int
	closed      chan struct{}
	events      chan providers.VoiceEvent
}

func (s *capturingRealtimeVoiceSession) SendAudio(ctx context.Context, chunk protocol.AudioChunk) error {
	s.audioChunks = append(s.audioChunks, chunk)
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) CommitAndCreateResponse(ctx context.Context) error {
	s.commitCalls++
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) Cancel(ctx context.Context, _ providers.VoiceCancelRequest) error {
	s.cancelCalls++
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) Close(ctx context.Context) error {
	if s.closed != nil {
		select {
		case s.closed <- struct{}{}:
		default:
		}
	}
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) Events() <-chan providers.VoiceEvent {
	return s.events
}

type countingV21Client struct {
	calls int
}

func (c *countingV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	c.calls++
	return v21adapter.NewMockClient().Query(ctx, request)
}

type readRecordV21Client struct{}

func (readRecordV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	return v21adapter.QueryResponse{
		TraceID:    request.TraceID,
		FastAnswer: "RAW_SECRET_FAST_ANSWER",
		Confidence: 0.91,
		Evidence: []v21adapter.Evidence{{
			Title:    "RAW_SECRET_TITLE",
			Type:     "v21_retrieval_evidence",
			SourceID: "v21-doc-read-record-001",
			Summary:  "RAW_SECRET_EVIDENCE_BODY",
			Quote:    "RAW_SECRET_QUOTE",
		}},
		SpeechBlocks:      []string{"RAW_SECRET_SPEECH_BLOCK"},
		ScreenCards:       []v21adapter.ScreenCard{{Label: "结论", Text: "RAW_SECRET_CARD_TEXT"}},
		FollowUps:         []string{"RAW_SECRET_FOLLOW_UP"},
		SourceScopeCounts: map[string]int{"public": 2, "personal": 1},
		WorkspaceStatus:   v21adapter.WorkspaceSearchable,
	}, nil
}

type capturingV21Client struct {
	request v21adapter.QueryRequest
}

func (c *capturingV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	c.request = request
	return v21adapter.QueryResponse{
		TraceID:    request.TraceID,
		FastAnswer: "V21 找到一条可引用证据。",
		Confidence: 0.9,
		Evidence: []v21adapter.Evidence{{
			Title:    "adapter source",
			Type:     "v21_retrieval_evidence",
			SourceID: "v21-doc-contract-001",
			Summary:  "sanitized evidence fixture",
		}},
		SpeechBlocks: []string{"V21 找到一条可引用证据。"},
		ScreenCards:  []v21adapter.ScreenCard{{Label: "结论", Text: "1 条证据"}},
		FollowUps:    []string{"要不要展开来源？"},
	}, nil
}

type failingV21Client struct{}

func (failingV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	return v21adapter.QueryResponse{}, errors.New("v21 unavailable")
}

type slowV21Client struct {
	delay time.Duration
}

func (c slowV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	timer := time.NewTimer(c.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return v21adapter.QueryResponse{}, ctx.Err()
	case <-timer.C:
		return v21adapter.NewMockClient().Query(ctx, request)
	}
}

type delayedXiaozhiProfessionalV21Client struct {
	delay    time.Duration
	started  chan struct{}
	released chan struct{}
	canceled chan struct{}
	once     sync.Once
	mu       sync.Mutex
	requests []v21adapter.QueryRequest
}

func newDelayedXiaozhiProfessionalV21Client(delay time.Duration) *delayedXiaozhiProfessionalV21Client {
	return &delayedXiaozhiProfessionalV21Client{
		delay:    delay,
		started:  make(chan struct{}),
		released: make(chan struct{}),
		canceled: make(chan struct{}),
	}
}

func (c *delayedXiaozhiProfessionalV21Client) release() {
	c.once.Do(func() {
		close(c.released)
	})
}

func (c *delayedXiaozhiProfessionalV21Client) lastUtterance() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.requests) == 0 {
		return ""
	}
	return c.requests[len(c.requests)-1].Utterance
}

func (c *delayedXiaozhiProfessionalV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	c.mu.Lock()
	c.requests = append(c.requests, request)
	c.mu.Unlock()
	select {
	case <-c.started:
	default:
		close(c.started)
	}
	if c.delay > 0 {
		timer := time.NewTimer(c.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			close(c.canceled)
			return v21adapter.QueryResponse{}, ctx.Err()
		case <-timer.C:
		}
	} else if c.delay < 0 {
		select {
		case <-ctx.Done():
			close(c.canceled)
			return v21adapter.QueryResponse{}, ctx.Err()
		case <-c.released:
		}
	}
	select {
	case <-ctx.Done():
		close(c.canceled)
		return v21adapter.QueryResponse{}, ctx.Err()
	default:
	}
	return v21adapter.QueryResponse{
		TraceID:    request.TraceID,
		FastAnswer: "结论：需要按可引用证据复核。",
		Confidence: 0.77,
		Evidence: []v21adapter.Evidence{{
			Title:    "raw evidence title",
			Type:     "meeting",
			SourceID: "v21-doc-secret-raw",
			Summary:  "RAW_SECRET_EVIDENCE_BODY",
			Quote:    "RAW_SECRET_QUOTE",
		}},
		SpeechBlocks: []string{"RAW_SECRET_SPEECH_BLOCK"},
		ScreenCards:  []v21adapter.ScreenCard{{Label: "结论", Text: "RAW_SECRET_CARD_TEXT"}},
		FollowUps:    []string{"RAW_SECRET_FOLLOW_UP"},
	}, nil
}

type gatewaySileroVADRunner struct {
	decision audio.VADDecision
	err      error
	calls    int
}

func (r *gatewaySileroVADRunner) Detect(ctx context.Context, frame audio.Frame) (audio.VADDecision, error) {
	r.calls++
	if r.err != nil {
		return audio.VADDecision{}, r.err
	}
	return r.decision, ctx.Err()
}

func traceEventAtMS(events []TraceEvent, name string) (int64, bool) {
	for _, event := range events {
		if event.Name == name {
			return event.AtMS, true
		}
	}
	return 0, false
}

func traceEventCount(events []TraceEvent, name string) int {
	count := 0
	for _, event := range events {
		if event.Name == name {
			count++
		}
	}
	return count
}
