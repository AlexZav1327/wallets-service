package walletserver

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func newMetrics() *metrics {
	return &metrics{
		requests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "wallets_service",
				Subsystem: "",
				Name:      "http_req_total",
				Help:      "total quantity of http requests",
			}, []string{"code", "method", "path"}),
		duration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "wallets_service",
				Subsystem: "",
				Name:      "http_req_duration",
				Help:      "http requests duration",
				Buckets:   []float64{0.0001, 0.0005, 0.001, 0.003, 0.005, 0.01, 0.05, 0.1, 1},
			}, []string{"code", "method", "path"}),
	}
}
