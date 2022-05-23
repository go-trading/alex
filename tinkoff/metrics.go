package tinkoff

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	balanceMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tinkoff_balance",
	},
		[]string{"account", "figi"},
	)
	positionMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tinkoff_position",
	},
		[]string{"account", "currency"},
	)
	lastPriceMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tinkoff_last_price",
	},
		[]string{"figi"},
	)
	targetPositionMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tinkoff_target_position",
		Help: "Какаое значение позиции по инструменту пытается достич робот",
	},
		[]string{"figi", "name"},
	)
)
