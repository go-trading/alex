package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/go-trading/alex"
	"github.com/go-trading/alex/tinkoff"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func accounts(c *cli.Context) error {
	t := tinkoff.NewClient(c.String("api"), c.String("token"), c.String("data"))

	if err := t.Open(c.Context); err != nil {
		l.Fatal("не смог открыть соединение", zap.Error(err))
	}
	defer t.Close()

	tbl := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tbl, tinkoff.AccountStringTableHead())

	accounts, err := t.Accounts.GetRealAccounts(c.Context)
	if err != nil {
		l.Fatal("не смог получить список счетов", zap.Error(err))
	}
	for _, a := range accounts {
		fmt.Fprintf(tbl, "%s\n", a)
	}

	accounts, err = t.GetAccounts(c.Context, alex.EngineType_SANDBOX)
	if err != nil {
		l.Fatal("не смог получить список счетов", zap.Error(err))
	}
	for _, a := range accounts {
		fmt.Fprintf(tbl, "%s\n", a)
	}

	tbl.Flush()

	return nil
}
