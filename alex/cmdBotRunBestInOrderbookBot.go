package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/go-trading/alex"
	"github.com/go-trading/alex/bots"
	"github.com/go-trading/alex/tinkoff"
)

func botRunBestInOrderbookBot(c *cli.Context) error {
	t := tinkoff.NewClient(c.String("api"), c.String("token"), c.String("data"))
	if err := t.Open(c.Context); err != nil {
		l.Fatal("не смог открыть соединение", zap.Error(err))
	}
	defer t.Close()

	var allBots alex.Bots

	account := t.Accounts.GetOrDie(c.Context, c.String("account"))

	for _, figi := range c.StringSlice("figi") {
		b := bots.NewBestInOrderbookBot(c.Context)
		err := b.Config(alex.NewConfig(
			fmt.Sprintf("rsi-%s-%s-%d", figi, c.Duration("candles-period"), c.Int("timeframe")),
			account,
			t.GetInstrument(figi),
			map[string]any{
				"max-position": c.Int("max-position"),
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

	// весь код ниже нужен чтобы дождаться ctrl-c, и корректно остановить роботов
	allBotsStopSignal := make(chan struct{})

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		l.Info("Got interrupt, shutting down...")
		go func() {
			err := allBots.StopAll()
			if err != nil {
				l.DPanic("Не смог корректно остановить роботов", zap.Error(err))
			}
			close(allBotsStopSignal)
		}()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				l.Info("Already shutting down, interrupt more to panic", zap.Int("times", i-1))
			}
		}
		panic("Недождался остановки роботов")
	}()
	// Ожидаю, когда роботы будут остановленны
	<-allBotsStopSignal

	return nil
}
