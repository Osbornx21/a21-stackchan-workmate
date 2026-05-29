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
