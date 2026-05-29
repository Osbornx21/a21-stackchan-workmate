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
