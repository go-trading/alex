package bots

// Метрики, общие для всех роботов.
// Если робот их записывает - то за ним можно будет наблюдать в мониторенге, если нет, то нечего страшного

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	botDurationMetric = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "bot_step_time_seconds",
		Help: "Длительность выполнение роботом одного тика",
	},
		[]string{"name"},
	)
)
