package alex

import (
	"context"
	"time"

	"github.com/sdcoffey/techan"
)

// интерфейс работы со свечами
type Candles interface {
	GetFigi() string
	GetPeriod() time.Duration
	GetSeries() *techan.TimeSeries
	Load(ctx context.Context, from time.Time, to time.Time) error
	Subscribe() (candleChan CandleChan, err error)
	Unsubscribe(candleChan CandleChan) error
}

// канал получения информации о изменении свечей
type CandleChan chan *techan.Candle
