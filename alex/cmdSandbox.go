package main

import (
	"fmt"

	"github.com/go-trading/alex/tinkoff"
	"github.com/sdcoffey/big"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func sandboxOpenAccount(c *cli.Context) error {
	t := tinkoff.NewClient(c.String("api"), c.String("token"), c.String("data"))
	if err := t.Open(c.Context); err != nil {
		l.DPanic("не смог открыть соединение", zap.Error(err))
	}
	defer t.Close()

	account, err := t.OpenSandboxAccount(c.Context)
	if err != nil {
		return err
	}
	fmt.Printf("В песочнице создан счёт %s\n", account)

	return nil
}

func sandboxCloseAccount(c *cli.Context) error {
	t := tinkoff.NewClient(c.String("api"), c.String("token"), c.String("data"))
	if err := t.Open(c.Context); err != nil {
		l.DPanic("не смог открыть соединение", zap.Error(err))
	}
	defer t.Close()

	err := t.CloseSandboxAccount(c.Context, c.String("account"))
	if err != nil {
		return err
	}
	fmt.Println("счёт закрыт")

	return nil
}

func sandboxPayIn(c *cli.Context) error {
	t := tinkoff.NewClient(c.String("api"), c.String("token"), c.String("data"))
	if err := t.Open(c.Context); err != nil {
		l.DPanic("не смог открыть соединение", zap.Error(err))
	}
	defer t.Close()

	res, err := t.SandboxPayIn(c.Context,
		c.String("account"),
		big.NewDecimal(c.Float64("rub")),
	)
	if err != nil {
		return err
	}
	fmt.Printf("Cчёт в песочнице пополнен. Сейчас на нём %s\n", res.Value)

	return nil
}
