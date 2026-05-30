package providers

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"a21.local/a21/internal/protocol"
)

func RealtimeFixtureSmokeFromEnv(ctx context.Context, env []string, providerName string, execute bool) ProviderSmokeReport {
	report := RealtimeWebSocketPlanFromEnv(env, providerName)
	report.Protocol = "websocket_realtime_fixture"
	if report.Status == ProviderSmokeFailed || report.Status == ProviderSmokeSkipped || report.Status == ProviderSmokeUnsupported {
		return report
	}
	if !execute {
		report.Status = ProviderSmokeReady
		report.Executed = false
		report.Detail = "offline realtime fixture is configured; pass --execute to run the local fake-connection smoke"
		return report
	}

	started := time.Now()
	if err := runRealtimeFixtureSmoke(ctx, env, report.Provider); err != nil {
		report.Status = ProviderSmokeFailed
		report.Executed = true
		report.DurationMS = float64(time.Since(started)) / float64(time.Millisecond)
		report.Detail = redactProviderSmokeDetail(err.Error())
		return report
	}
	report.Status = ProviderSmokePassed
	report.Executed = true
	report.DurationMS = float64(time.Since(started)) / float64(time.Millisecond)
	report.Detail = "offline realtime fixture passed; no provider network call performed"
	return report
}

func runRealtimeFixtureSmoke(ctx context.Context, env []string, providerName string) error {
	dialer := &realtimeFixtureDialer{}
	voice := VoiceSession{
		TraceID:   "a21-trace-realtime-fixture",
		SessionID: "a21-session-realtime-fixture",
		DeviceID:  "stackchan-fixture-001",
	}
	switch providerName {
	case "openai_realtime":
		provider := NewOpenAIRealtimeVoiceProviderFromEnv(env, dialer)
		session, err := provider.StartRealtimeSession(ctx, voice)
		if err != nil {
			return err
		}
		defer session.Close(ctx)
		if err := session.SendAudio(ctx, protocol.AudioChunk{
			Codec:        protocol.AudioCodecPCMS16LE,
			SampleRateHz: 24000,
			Channels:     1,
			DurationMS:   20,
			DataBase64:   base64.StdEncoding.EncodeToString([]byte{0, 1, 0, 1}),
		}); err != nil {
			return err
		}
		if err := session.CommitAndCreateResponse(ctx); err != nil {
			return err
		}
		if err := session.Cancel(ctx, VoiceCancelRequest{Session: voice, Reason: CancelBargeIn}); err != nil {
			return err
		}
		return nil
	case "doubao_tts_realtime":
		provider := NewDoubaoRealtimeTTSProviderFromEnv(env, dialer)
		session, err := provider.StartRealtimeTTSSession(ctx, voice)
		if err != nil {
			return err
		}
		defer session.Close(ctx)
		if err := session.SendText(ctx, "A21 offline realtime fixture smoke."); err != nil {
			return err
		}
		if err := session.TextDone(ctx); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("realtime fixture is not implemented for provider")
	}
}

type realtimeFixtureDialer struct {
	conn realtimeFixtureConn
}

func (d *realtimeFixtureDialer) Dial(_ context.Context, _ string, _ http.Header, _ NetworkPolicy) (RealtimeConn, error) {
	return &d.conn, nil
}

type realtimeFixtureConn struct {
	closed bool
}

func (c *realtimeFixtureConn) WriteJSON(ctx context.Context, value any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.closed {
		return fmt.Errorf("realtime fixture connection is closed")
	}
	if value == nil {
		return fmt.Errorf("realtime fixture message is nil")
	}
	return nil
}

func (c *realtimeFixtureConn) ReadJSON(ctx context.Context, value any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("realtime fixture connection has no server events")
}

func (c *realtimeFixtureConn) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.closed = true
	return nil
}
