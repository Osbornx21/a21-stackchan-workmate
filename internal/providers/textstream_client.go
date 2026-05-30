package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type TextStreamCompletionOptions struct {
	ProviderName string
	Prompt       string
	MaxTokens    int
	Client       *http.Client
}

type TextStreamCompletionResult struct {
	Provider            string         `json:"provider"`
	Family              ProviderFamily `json:"family"`
	Protocol            string         `json:"protocol"`
	Configured          bool           `json:"configured"`
	EndpointHost        string         `json:"endpoint_host,omitempty"`
	FirstByteMS         float64        `json:"first_byte_ms,omitempty"`
	FirstContentMS      float64        `json:"first_content_ms,omitempty"`
	TotalDurationMS     float64        `json:"total_duration_ms,omitempty"`
	ContentDeltaCount   int            `json:"content_delta_count,omitempty"`
	ReasoningDeltaCount int            `json:"reasoning_delta_count,omitempty"`
	Done                bool           `json:"done,omitempty"`
	ContentText         string         `json:"-"`
}

func RunTextStreamCompletionFromEnv(ctx context.Context, env []string, options TextStreamCompletionOptions) (TextStreamCompletionResult, error) {
	rawProvider := strings.TrimSpace(options.ProviderName)
	if rawProvider == "" {
		rawProvider = strings.TrimSpace(envValue(env, "A21_PROVIDER_PRIMARY"))
	}
	if rawProvider == "" {
		rawProvider = "deepseek"
	}
	provider := strings.ToLower(rawProvider)
	result := TextStreamCompletionResult{Provider: safeProviderName(provider)}
	if containsLegacyProviderIdentity(provider) || containsBlockedProviderIdentity(provider) {
		return result, fmt.Errorf("provider target rejected")
	}
	profile, _, ok := ProviderProfileByNameFromEnv(env, provider)
	if !ok {
		return result, fmt.Errorf("provider target is unknown")
	}
	spec := providerSmokeSpecFromProfile(profile)
	result.Provider = spec.Name
	result.Family = spec.Family
	result.Protocol = spec.Protocol
	result.EndpointHost = providerSmokeEndpointHost(env, spec)
	configured, missing := providerSmokeConfigured(env, spec)
	result.Configured = configured
	if !configured {
		return result, fmt.Errorf("provider text stream missing env: %s", strings.Join(missing, ","))
	}
	if !spec.Executable {
		return result, fmt.Errorf("provider text stream is not executable for protocol")
	}
	endpoint, err := providerSmokeEndpoint(env, spec)
	if err != nil {
		return result, err
	}
	prompt := strings.TrimSpace(options.Prompt)
	if prompt == "" {
		prompt = "A21 local voice loopback. Reply briefly."
	}
	maxTokens := options.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 48
	}
	body := map[string]any{
		"model": providerSmokeModel(env, spec),
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": maxTokens,
		"stream":     true,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return result, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return result, err
	}
	if spec.APIKeyEnv != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(envValue(env, spec.APIKeyEnv)))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", "a21-text-stream/0.1")
	client := options.Client
	if client == nil {
		policy, _ := NetworkPolicyFromEnv(env)
		client, err = NewProviderHTTPClient(policy)
		if err != nil {
			return result, err
		}
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		result.TotalDurationMS = elapsedMS(start)
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.TotalDurationMS = elapsedMS(start)
		return result, fmt.Errorf("%s", redactedProviderHTTPError(resp.StatusCode, nil))
	}
	var content strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		if result.FirstByteMS == 0 {
			result.FirstByteMS = elapsedMS(start)
		}
		events, done, err := parseOpenAICompatibleTextStreamLine(strings.TrimSpace(scanner.Text()))
		if err != nil {
			result.TotalDurationMS = elapsedMS(start)
			return result, err
		}
		if done {
			result.Done = true
			continue
		}
		for _, event := range events {
			if event.Text == "" {
				continue
			}
			if result.FirstContentMS == 0 {
				result.FirstContentMS = elapsedMS(start)
			}
			switch event.Kind {
			case TextStreamDeltaContent:
				result.ContentDeltaCount++
				content.WriteString(event.Text)
			case TextStreamDeltaReasoning:
				result.ReasoningDeltaCount++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		result.TotalDurationMS = elapsedMS(start)
		return result, err
	}
	result.TotalDurationMS = elapsedMS(start)
	result.ContentText = content.String()
	return result, nil
}
