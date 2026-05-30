package providers

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type TextStreamDeltaKind string

const (
	TextStreamDeltaContent   TextStreamDeltaKind = "content"
	TextStreamDeltaReasoning TextStreamDeltaKind = "reasoning"
	TextStreamDeltaDone      TextStreamDeltaKind = "done"
)

type TextStreamEvent struct {
	Kind TextStreamDeltaKind
	Text string
}

type TextStreamParseResult struct {
	Events              []TextStreamEvent
	ContentDeltaCount   int
	ReasoningDeltaCount int
	Done                bool
}

func (r TextStreamParseResult) ContentText() string {
	var builder strings.Builder
	for _, event := range r.Events {
		if event.Kind == TextStreamDeltaContent {
			builder.WriteString(event.Text)
		}
	}
	return builder.String()
}

func (r TextStreamParseResult) ReasoningText() string {
	var builder strings.Builder
	for _, event := range r.Events {
		if event.Kind == TextStreamDeltaReasoning {
			builder.WriteString(event.Text)
		}
	}
	return builder.String()
}

func ParseOpenAICompatibleTextStream(reader io.Reader) (TextStreamParseResult, error) {
	var result TextStreamParseResult
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		events, done, err := parseOpenAICompatibleTextStreamLine(line)
		if err != nil {
			return result, err
		}
		if done {
			result.Done = true
			result.Events = append(result.Events, TextStreamEvent{Kind: TextStreamDeltaDone})
			continue
		}
		for _, event := range events {
			if event.Text == "" {
				continue
			}
			switch event.Kind {
			case TextStreamDeltaContent:
				result.ContentDeltaCount++
			case TextStreamDeltaReasoning:
				result.ReasoningDeltaCount++
			}
			result.Events = append(result.Events, event)
		}
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func parseOpenAICompatibleTextStreamLine(line string) ([]TextStreamEvent, bool, error) {
	if line == "" || strings.HasPrefix(line, ":") || strings.HasPrefix(line, "event:") {
		return nil, false, nil
	}
	if !strings.HasPrefix(line, "data:") {
		return nil, false, nil
	}
	data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if data == "[DONE]" {
		return nil, true, nil
	}
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content          string `json:"content"`
				Reasoning        string `json:"reasoning"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, false, fmt.Errorf("parse text stream chunk: %w", err)
	}
	var events []TextStreamEvent
	for _, choice := range chunk.Choices {
		if choice.Delta.Reasoning != "" {
			events = append(events, TextStreamEvent{Kind: TextStreamDeltaReasoning, Text: choice.Delta.Reasoning})
		}
		if choice.Delta.ReasoningContent != "" {
			events = append(events, TextStreamEvent{Kind: TextStreamDeltaReasoning, Text: choice.Delta.ReasoningContent})
		}
		if choice.Delta.Content != "" {
			events = append(events, TextStreamEvent{Kind: TextStreamDeltaContent, Text: choice.Delta.Content})
		}
	}
	return events, false, nil
}

func redactedProviderHTTPError(status int, body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf("provider smoke HTTP status %d body_sha256=%s body_bytes=%d", status, hex.EncodeToString(sum[:]), len(body))
}
