package main

import (
	"github.com/go-trading/alex/tinkoff"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func load(c *cli.Context) error {
	t := tinkoff.NewClient(c.String("api"), c.String("token"), c.String("data"))

	if err := t.Open(c.Context); err != nil {
		l.Fatal("не смог открыть соединение", zap.Error(err))
	}
	defer t.Close()

	for _, figi := range c.StringSlice("figi") {
		candles := t.GetInstrument(figi).GetCandles(c.Duration("candles-period"))
		err := candles.Load(c.Context, *c.Timestamp("from"), *c.Timestamp("to"))
		if err != nil {
			l.Fatal("не смог скачать", zap.String("figi", figi), zap.Error(err))
		}
		err = t.SaveCandles(candles)
		if err != nil {
			l.DPanic("Не смог сохранить сделки", zap.Error(err))
		}
	}
	return nil
}
