package rates

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	duration prometheus.Histogram
}

func newMetrics() *metrics {
	return &metrics{
		duration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: "wallets_service",
				Subsystem: "",
				Name:      "xr_resp_duration",
				Help:      "exchange rates server response duration",
				Buckets:   []float64{0.0001, 0.0005, 0.001, 0.003, 0.005, 0.01, 0.05, 0.1, 1},
			}),
	}
}
