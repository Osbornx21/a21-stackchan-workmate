package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const openAIRealtimeDefaultURL = "wss://api.openai.com/v1/realtime"
const doubaoRealtimeDefaultURL = "wss://ai-gateway.vei.volces.com/v1/realtime"

type RealtimeWebSocketConfig struct {
	Provider      string
	URL           string
	Headers       map[string]string
	NetworkPolicy NetworkPolicy
}

type RealtimeDialer interface {
	Dial(ctx context.Context, url string, header http.Header, policy NetworkPolicy) (RealtimeConn, error)
}

type RealtimeConn interface {
	WriteJSON(ctx context.Context, value any) error
	ReadJSON(ctx context.Context, value any) error
	Close(ctx context.Context) error
}

type RealtimeWebSocketAdapter struct {
	config RealtimeWebSocketConfig
	dialer RealtimeDialer
}

type RealtimeWebSocketSession struct {
	provider string
	conn     RealtimeConn
}

func RealtimeWebSocketPlanFromEnv(env []string, providerName string) ProviderSmokeReport {
	_, network := NetworkPolicyFromEnv(env)
	rawProvider := strings.TrimSpace(providerName)
	if rawProvider == "" {
		rawProvider = strings.TrimSpace(envValue(env, "A21_PROVIDER_PRIMARY"))
	}
	if rawProvider == "" {
		rawProvider = "mock"
	}
	provider := strings.ToLower(rawProvider)
	report := ProviderSmokeReport{
		Provider:    safeProviderName(provider),
		Protocol:    "websocket_realtime",
		Status:      ProviderSmokeFailed,
		NetworkMode: network.Mode,
	}
	if containsLegacyProviderIdentity(provider) {
		report.Findings = append(report.Findings, ProviderCatalogFinding{
			Code:    "provider_legacy_identity",
			Message: "A21 realtime provider target contains a forbidden legacy identity",
			Detail:  "A21_PROVIDER_PRIMARY",
		})
		report.Detail = "provider target rejected"
		return report
	}
	switch provider {
	case "openai_realtime":
		report.Provider = provider
		report.APIKeyEnv = "A21_OPENAI_API_KEY"
		report.ModelEnv = "A21_OPENAI_REALTIME_MODEL"
		report.Configured, report.MissingEnv = providerSmokeConfigured(env, providerSmokeSpec{
			APIKeyEnv: report.APIKeyEnv,
			ModelEnv:  report.ModelEnv,
		})
		if !report.Configured {
			report.Status = ProviderSmokeSkipped
			report.Detail = "realtime websocket plan skipped because required env is missing"
			return report
		}
		endpoint, err := openAIRealtimeURL(strings.TrimSpace(envValue(env, report.ModelEnv)))
		if err != nil {
			report.Status = ProviderSmokeFailed
			report.Detail = redactProviderSmokeDetail(err.Error())
			return report
		}
		report.EndpointHost = endpointHost(endpoint)
		report.Status = ProviderSmokeReady
		report.Detail = "realtime websocket plan is configured; explicit adapter connect is required"
		return report
	case "doubao_realtime":
		report.Provider = provider
		report.APIKeyEnv = "A21_DOUBAO_API_KEY"
		report.ModelEnv = "A21_DOUBAO_REALTIME_MODEL"
		report.Configured, report.MissingEnv = providerSmokeConfigured(env, providerSmokeSpec{
			APIKeyEnv: report.APIKeyEnv,
			ModelEnv:  report.ModelEnv,
			RequiredEnv: []string{
				"A21_DOUBAO_APP_ID",
				"A21_DOUBAO_RESOURCE_ID",
			},
		})
		if !report.Configured {
			report.Status = ProviderSmokeSkipped
			report.Detail = "doubao realtime speech-to-speech plan skipped because required env is missing"
			return report
		}
		endpoint, err := doubaoRealtimeURL(strings.TrimSpace(envValue(env, report.ModelEnv)), report.ModelEnv)
		if err != nil {
			report.Status = ProviderSmokeFailed
			report.Detail = redactProviderSmokeDetail(err.Error())
			return report
		}
		report.EndpointHost = endpointHost(endpoint)
		report.Status = ProviderSmokeReady
		report.Detail = "doubao realtime speech-to-speech websocket plan is configured; explicit adapter connect is required"
		return report
	case "doubao_tts_realtime":
		report.Provider = provider
		report.APIKeyEnv = "A21_DOUBAO_API_KEY"
		report.ModelEnv = "A21_DOUBAO_TTS_MODEL"
		report.Configured, report.MissingEnv = providerSmokeConfigured(env, providerSmokeSpec{
			APIKeyEnv:   report.APIKeyEnv,
			ModelEnv:    report.ModelEnv,
			RequiredEnv: []string{"A21_DOUBAO_TTS_VOICE"},
		})
		if !report.Configured {
			report.Status = ProviderSmokeSkipped
			report.Detail = "doubao realtime TTS plan skipped because required env is missing"
			return report
		}
		endpoint, err := doubaoRealtimeURL(strings.TrimSpace(envValue(env, report.ModelEnv)), report.ModelEnv)
		if err != nil {
			report.Status = ProviderSmokeFailed
			report.Detail = redactProviderSmokeDetail(err.Error())
			return report
		}
		report.EndpointHost = endpointHost(endpoint)
		report.Status = ProviderSmokeReady
		report.Detail = "doubao realtime TTS websocket plan is configured; explicit adapter connect is required"
		return report
	default:
		report.Status = ProviderSmokeUnsupported
		report.Detail = "provider-specific realtime websocket plan is not implemented yet"
		if provider == "mock" {
			report.Status = ProviderSmokeSkipped
			report.Detail = "mock provider does not require a realtime websocket plan"
		}
		return report
	}
}

func NewRealtimeWebSocketAdapter(config RealtimeWebSocketConfig, dialer RealtimeDialer) *RealtimeWebSocketAdapter {
	if dialer == nil {
		dialer = coderRealtimeDialer{}
	}
	return &RealtimeWebSocketAdapter{config: config, dialer: dialer}
}

func (a *RealtimeWebSocketAdapter) Connect(ctx context.Context, session map[string]any) (*RealtimeWebSocketSession, error) {
	if a == nil {
		return nil, fmt.Errorf("realtime websocket adapter is nil")
	}
	if strings.TrimSpace(a.config.Provider) == "" {
		return nil, fmt.Errorf("realtime provider is required")
	}
	if strings.TrimSpace(a.config.URL) == "" {
		return nil, fmt.Errorf("realtime websocket URL is required")
	}
	if session == nil {
		session = map[string]any{}
	}
	conn, err := a.dialer.Dial(ctx, a.config.URL, realtimeHeaders(a.config.Headers), a.config.NetworkPolicy)
	if err != nil {
		return nil, err
	}
	if err := conn.WriteJSON(ctx, map[string]any{
		"type":    "session.update",
		"session": session,
	}); err != nil {
		_ = conn.Close(ctx)
		return nil, err
	}
	return &RealtimeWebSocketSession{
		provider: a.config.Provider,
		conn:     conn,
	}, nil
}

func (a *RealtimeWebSocketAdapter) Report() ProviderSmokeReport {
	report := ProviderSmokeReport{
		Provider:    safeProviderName(strings.ToLower(strings.TrimSpace(a.config.Provider))),
		Protocol:    "websocket_realtime",
		Status:      ProviderSmokeReady,
		Configured:  true,
		NetworkMode: a.config.NetworkPolicy.Mode,
		Detail:      "realtime websocket adapter is configured",
	}
	if report.NetworkMode == "" {
		report.NetworkMode = NetworkModeDirect
	}
	if containsLegacyProviderIdentity(a.config.Provider) {
		report.Status = ProviderSmokeFailed
		report.Configured = false
		report.Detail = "provider target rejected"
		return report
	}
	if strings.TrimSpace(a.config.Provider) == "" || strings.TrimSpace(a.config.URL) == "" {
		report.Status = ProviderSmokeSkipped
		report.Configured = false
		report.Detail = "realtime websocket adapter is missing provider or URL"
		return report
	}
	report.EndpointHost = endpointHost(a.config.URL)
	return report
}

func (s *RealtimeWebSocketSession) Cancel(ctx context.Context, _ VoiceCancelRequest) error {
	if s == nil || s.conn == nil {
		return fmt.Errorf("realtime websocket session is not connected")
	}
	return s.conn.WriteJSON(ctx, map[string]any{
		"type": "response.cancel",
	})
}

func (s *RealtimeWebSocketSession) ReadEvent(ctx context.Context) (map[string]any, error) {
	if s == nil || s.conn == nil {
		return nil, fmt.Errorf("realtime websocket session is not connected")
	}
	var event map[string]any
	if err := s.conn.ReadJSON(ctx, &event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *RealtimeWebSocketSession) Close(ctx context.Context) error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close(ctx)
}

type coderRealtimeDialer struct{}

func (coderRealtimeDialer) Dial(ctx context.Context, rawURL string, header http.Header, policy NetworkPolicy) (RealtimeConn, error) {
	client, err := NewProviderHTTPClient(policy)
	if err != nil {
		return nil, err
	}
	conn, _, err := websocket.Dial(ctx, rawURL, &websocket.DialOptions{
		HTTPClient: client,
		HTTPHeader: header,
	})
	if err != nil {
		return nil, fmt.Errorf("%s", redactProviderSmokeDetail(err.Error()))
	}
	return coderRealtimeConn{conn: conn}, nil
}

type coderRealtimeConn struct {
	conn *websocket.Conn
}

func (c coderRealtimeConn) WriteJSON(ctx context.Context, value any) error {
	return wsjson.Write(ctx, c.conn, value)
}

func (c coderRealtimeConn) ReadJSON(ctx context.Context, value any) error {
	return wsjson.Read(ctx, c.conn, value)
}

func (c coderRealtimeConn) Close(_ context.Context) error {
	return c.conn.Close(websocket.StatusNormalClosure, "a21 realtime session closed")
}

func realtimeHeaders(values map[string]string) http.Header {
	header := http.Header{}
	for name, value := range values {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		header.Set(name, value)
	}
	return header
}

func openAIRealtimeURL(model string) (string, error) {
	if model == "" {
		return "", fmt.Errorf("A21_OPENAI_REALTIME_MODEL is required")
	}
	parsed, err := url.Parse(openAIRealtimeDefaultURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("model", model)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func doubaoRealtimeURL(model string, modelEnv string) (string, error) {
	if model == "" {
		if modelEnv == "" {
			modelEnv = "A21_DOUBAO_REALTIME_MODEL"
		}
		return "", fmt.Errorf("%s is required", modelEnv)
	}
	parsed, err := url.Parse(doubaoRealtimeDefaultURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("model", model)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func endpointHost(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "invalid_base_url"
	}
	return parsed.Host
}
