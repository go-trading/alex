package history

import (
	"context"
	"time"

	"github.com/go-trading/alex"
	"github.com/sdcoffey/techan"
)

var _ alex.Candles = (*Candles)(nil)

type Candles struct {
	series      *techan.TimeSeries
	subscribers []alex.CandleChan
	figi        string
	client      *Client
	instrument  *instrument
}

func NewCandles(figi string, client *Client, instrument *instrument) *Candles {
	return &Candles{
		figi:       figi,
		client:     client,
		instrument: instrument,
		series:     techan.NewTimeSeries(),
	}
}

func (c *Candles) GetFigi() string {
	return c.figi
}
func (c *Candles) GetPeriod() time.Duration {
	return time.Minute
}
func (c *Candles) GetSeries() *techan.TimeSeries {
	return c.series
}
func (c *Candles) Load(ctx context.Context, from time.Time, to time.Time) error {
	now := c.client.now
	for _, historyCandle := range c.instrument.FUTURE.Candles {
		if historyCandle.Period.End.Before(now) {
			if from.Before(historyCandle.Period.Start) && to.After(historyCandle.Period.Start) {
				alex.UpsertSeries(c.series, historyCandle)
			}
		} else {
			break
		}
	}
	return nil
}
func (c *Candles) Subscribe() (candleChan alex.CandleChan, err error) {
	ch := make(alex.CandleChan)
	c.subscribers = append(c.subscribers, ch)
	return ch, nil
}
func (c *Candles) Unsubscribe(candleChan alex.CandleChan) error {
	if !c.RemoveSubscriber(candleChan) {
		l.DPanic("отписываюсь не подписываясь")
	}
	return nil
}
func (c *Candles) OnTick(cc *techan.Candle) {
	alex.UpsertSeries(c.series, cc)

	for _, ch := range c.subscribers {
		ch <- cc
	}
}

func (c *Candles) RemoveSubscriber(candleChan alex.CandleChan) bool {
	for i, subscriber := range c.subscribers {
		if candleChan == subscriber {
			c.subscribers = append(c.subscribers[:i], c.subscribers[i+1:]...)
			close(candleChan)
			return true
		}
	}
	return false
}
