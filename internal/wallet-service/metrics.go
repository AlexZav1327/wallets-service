package walletservice

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	wallets        prometheus.Counter
	deletedWallets prometheus.Counter
	funds          *prometheus.GaugeVec
	duration       *prometheus.HistogramVec
}

func newMetrics() *metrics {
	return &metrics{
		wallets: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: "wallets_service",
				Subsystem: "",
				Name:      "wallets_total",
				Help:      "total quantity of wallets have been created",
			}),
		deletedWallets: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: "wallets_service",
				Subsystem: "",
				Name:      "deleted_wallets_total",
				Help:      "total quantity of wallets have been deleted",
			}),
		funds: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "wallets_service",
				Subsystem: "",
				Name:      "total_funds_amount",
				Help:      "total amount of funds in wallets",
			}, []string{"currency"}),
		duration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "wallets_service",
				Subsystem: "",
				Name:      "db_resp_duration",
				Help:      "database response duration",
				Buckets:   []float64{0.0001, 0.0005, 0.001, 0.003, 0.005, 0.01, 0.05, 0.1, 1},
			}, []string{"operation_type"}),
	}
}
