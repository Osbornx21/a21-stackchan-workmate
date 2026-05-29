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
	deviceIdentityInvalidTotal prometheus.Counter
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
		deviceIdentityInvalidTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "a21_device_identity_invalid_total",
			Help: "Total A21 device events rejected because firmware identity was invalid.",
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
	registry.MustRegister(m.mockTurnTotal, m.bargeInTotal, m.audioFrameTotal, m.audioPlaybackChunkTotal, m.deviceIdentityInvalidTotal, m.v21QueryMS, m.wsConnections)
	return m
}

func (m *metrics) handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
