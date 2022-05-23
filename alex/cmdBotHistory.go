package main

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/go-trading/alex"
	"github.com/go-trading/alex/bots"
	"github.com/go-trading/alex/history"
)

func botHistory(c *cli.Context) error {
	h := history.NewClient(
		c.String("data"),
		*c.Timestamp("from"),
		*c.Timestamp("to"),
	)
	var allBots alex.Bots

	for _, figi := range c.StringSlice("figi") {
		err := h.LoadData(figi)
		if err != nil {
			l.Panic("не смог загрузить данные", zap.Error(err))
			return err
		}

		name := fmt.Sprintf("rsi-%s-%s-%d", figi, time.Minute, c.Int("timeframe"))
		account := h.CreateAccount(name)

		b := bots.NewRSIBot(c.Context)
		err = b.Config(alex.NewConfig(
			name,
			account,
			h.GetInstrument(figi),
			map[string]any{
				"candles-period": time.Minute,
				"timeframe":      c.Int("timeframe"),
				"rsi4buy":        c.Int("rsi4buy"),
				"rsi4sell":       c.Int("rsi4sell"),
				"max-position":   c.Int("max-position"),
			},
		))
		if err != nil {
			l.Panic("Не смог сконфигурировать робота", zap.Error(err))
			return err
		}
		allBots = append(allBots, b)
	}

	if err := allBots.StartAll(); err != nil {
		return err
	}
	if err := h.Run(); err != nil {
		return err
	}
	h.PrintResult()
	return nil
}
