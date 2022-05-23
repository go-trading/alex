package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-trading/alex"
	"github.com/go-trading/alex/tinkoff"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func ReadCandles(figi string, ch alex.CandleChan) {
	for c := range ch {
		fmt.Printf("%v %v\n\n", figi, c)
	}
}

func onlineCandle(c *cli.Context) error {
	t := tinkoff.NewClient(c.String("api"), c.String("token"), c.String("data"))
	if err := t.Open(c.Context); err != nil {
		panic(fmt.Sprintf("не смог открыть соединение  %s", err))
	}
	defer t.Close()

	chs := make(map[string]alex.CandleChan)
	for _, figi := range c.StringSlice("figi") {
		ch, err := t.Instruments.Get(figi).GetCandles(c.Duration("interval")).Subscribe()
		if err != nil {
			l.DPanic("Не смог подписаться на свечи", zap.Error(err))
		}
		go ReadCandles(figi, ch)
		chs[figi] = ch
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc

	for figi, ch := range chs {
		err := t.Instruments.Get(figi).GetCandles(c.Duration("interval")).Unsubscribe(ch)
		if err != nil {
			l.DPanic("Не смог отписаться", zap.Error(err))
		}
	}

	return nil
}
