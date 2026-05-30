package gateway

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	registry                   *prometheus.Registry
	mockTurnTotal              prometheus.Counter
	bargeInTotal               prometheus.Counter
	audioFrameTotal            prometheus.Counter
	audioPlaybackChunkTotal    prometheus.Counter
	audioIngressFramesTotal    prometheus.Counter
	audioIngressDroppedTotal   prometheus.Counter
	audioIngressBufferDepth    prometheus.Gauge
	vadSpeechStartTotal        prometheus.Counter
	vadSpeechEndTotal          prometheus.Counter
	deviceIdentityInvalidTotal prometheus.Counter
	realtimeSessionTotal       prometheus.Counter
	realtimeSessionCancelTotal prometheus.Counter
	voiceProviderStartTurnMS   prometheus.Histogram
	voiceProviderCancelMS      prometheus.Histogram
	v21QueryMS                 prometheus.Histogram
	wsConnections              *prometheus.GaugeVec
}

func newMetrics() *metrics {
	registry := prometheus.NewRegistry()
	m := &metrics{
		registry: registry,
		mockTurnTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_mock_turn_total",
			Help: "Total mock A21 turns handled by the gateway.",
		}),
		bargeInTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_barge_in_total",
			Help: "Total A21 interrupt or barge-in events handled by the gateway.",
		}),
		audioFrameTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_audio_frame_total",
			Help: "Total mock audio frames accepted by the gateway.",
		}),
		audioPlaybackChunkTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_audio_playback_chunk_total",
			Help: "Total mock playback audio chunks sent by the gateway.",
		}),
		audioIngressFramesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_audio_ingress_frames_total",
			Help: "Total A21 audio frames accepted by the Gateway ingress buffer.",
		}),
		audioIngressDroppedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_audio_ingress_dropped_frames_total",
			Help: "Total A21 audio ingress frames dropped by bounded buffering.",
		}),
		audioIngressBufferDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "a21_audio_ingress_buffer_depth",
			Help: "Current A21 audio ingress buffer depth for the most recently handled stream.",
		}),
		vadSpeechStartTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_vad_speech_start_total",
			Help: "Total mock VAD speech-start transitions detected at Gateway ingress.",
		}),
		vadSpeechEndTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_vad_speech_end_total",
			Help: "Total mock VAD speech-end transitions detected at Gateway ingress.",
		}),
		deviceIdentityInvalidTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_device_identity_invalid_total",
			Help: "Total A21 device events rejected because firmware identity was invalid.",
		}),
		realtimeSessionTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_realtime_session_total",
			Help: "Total A21 realtime voice sessions started through the Gateway session boundary.",
		}),
		realtimeSessionCancelTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_realtime_session_cancel_total",
			Help: "Total A21 realtime voice session cancellations requested through the Gateway session boundary.",
		}),
		voiceProviderStartTurnMS: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "a21_voice_provider_start_turn_ms",
			Help:    "A21 voice provider StartTurn latency in milliseconds.",
			Buckets: []float64{25, 50, 100, 250, 500, 750, 1000, 1500, 2500, 5000},
		}),
		voiceProviderCancelMS: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "a21_voice_provider_cancel_ms",
			Help:    "A21 voice provider Cancel latency in milliseconds.",
			Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000},
		}),
		v21QueryMS: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "a21_v21_query_ms",
			Help:    "A21 professional-mode V21 adapter query latency in milliseconds.",
			Buckets: []float64{50, 100, 250, 500, 750, 1000, 1500, 2000, 3000, 5000, 10000},
		}),
		wsConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "a21_ws_connections_active",
			Help: "Active A21 WebSocket connections by channel.",
		}, []string{"channel"}),
	}
	registry.MustRegister(
		m.mockTurnTotal,
		m.bargeInTotal,
		m.audioFrameTotal,
		m.audioPlaybackChunkTotal,
		m.audioIngressFramesTotal,
		m.audioIngressDroppedTotal,
		m.audioIngressBufferDepth,
		m.vadSpeechStartTotal,
		m.vadSpeechEndTotal,
		m.deviceIdentityInvalidTotal,
		m.realtimeSessionTotal,
		m.realtimeSessionCancelTotal,
		m.voiceProviderStartTurnMS,
		m.voiceProviderCancelMS,
		m.v21QueryMS,
		m.wsConnections,
	)
	return m
}

func (m *metrics) handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
