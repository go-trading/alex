package tinkoff

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
)

var _ alex.Candles = (*Candles)(nil)

type Candles struct {
	client          *Client
	Figi            string
	Period          time.Duration
	Series          *techan.TimeSeries
	subscribersLock sync.RWMutex
	incomingChannel alex.CandleChan
	subscribers     []alex.CandleChan
	l               *zap.Logger
}

func NewCandles(client *Client, figi string, period time.Duration) *Candles {
	cs := &Candles{
		client: client,
		Figi:   figi,
		Period: period,
		l: l.Named("candles").With(
			zap.String("figi", figi),
			zap.Duration("period", period),
		),
		Series: techan.NewTimeSeries(),
	}
	cs.incomingChannel = client.SubscribeCandles(cs)
	go cs.incomingCandleRecv()

	return cs
}

func (cs *Candles) MergeApiCandles(apiCandles []*proto.HistoricCandle) {
	for _, c := range apiCandles {
		cs.Upsert(&techan.Candle{
			Period:     techan.NewTimePeriod(c.Time.AsTime(), cs.Period),
			OpenPrice:  alex.NewDecimal(c.Open),
			ClosePrice: alex.NewDecimal(c.Close),
			MaxPrice:   alex.NewDecimal(c.High),
			MinPrice:   alex.NewDecimal(c.Low),
			Volume:     big.NewFromInt(int(c.Volume)),
		})
	}
}

func (cs *Candles) LoadFromData() error {
	series, err := alex.LoadTimeSeries(cs.client.dataDir, cs.Figi, cs.Period)
	for _, candle := range series.Candles {
		cs.Upsert(candle)
	}
	return err
}

func (cs *Candles) Upsert(newCandle *techan.Candle) {
	if cs.Period != newCandle.Period.Length() {
		l.DPanic("cs.Interval != newCandle.Period.Length()")
		return
	}

	alex.UpsertSeries(cs.Series, newCandle)
}

func (cs *Candles) Save() error {
	return alex.SaveTimeSeries(cs.client.dataDir, cs.Figi, cs.Period, cs.Series)
}

func (cs *Candles) GetPeriod() time.Duration {
	return cs.Period
}
func (cs *Candles) GetSeries() *techan.TimeSeries {
	return cs.Series
}
func (cs *Candles) GetFigi() string {
	return cs.Figi
}
