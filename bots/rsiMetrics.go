package bots

// Метрики, специфичные для RSI робота

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sdcoffey/big"
)

var (
	rsiMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bot_rsi",
		Help: "рассчитанное значение RSI",
	},
		[]string{"figi", "candles_period", "timeframe"},
	)
	rsi4buyMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bot_rsi4buy",
		Help: "Значение RSI, выше которого робот покупает",
	},
		[]string{"figi", "name"},
	)
	rsi4sellMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bot_rsi4sell",
		Help: "Значение RSI, ниже которого робот продаёт",
	},
		[]string{"figi", "name"},
	)
)

// метод записи метрики значения индекса RSI, рассчитанного роботом
func (b *RSIBot) writeMetricsRSI(rsi big.Decimal) {
	rsiMetric.WithLabelValues(
		b.instrument.GetFigi(),
		b.candles.GetPeriod().String(),
		strconv.Itoa(b.timeframe),
	).Set(rsi.Float())
}

// метод записи метрик, которые меняются только при конфигурации
func (b *RSIBot) writeConfigMetrics() {
	rsi4buyMetric.WithLabelValues(
		b.instrument.GetFigi(),
		b.name,
	).Set(float64(b.rsi4buy))

	rsi4sellMetric.WithLabelValues(
		b.instrument.GetFigi(),
		b.name,
	).Set(float64(b.rsi4sell))
}
