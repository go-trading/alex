package main

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/go-trading/alex/tinkoff"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/urfave/cli/v2"
)

var instrumentsFlags = []cli.Flag{
	&cli.IntFlag{
		Name:  "status",
		Usage: "Выводить инструменты только в заданном статусе. Ожидается числовое значение статуса. см. SecurityTradingStatus на https://tinkoff.github.io/proto/operations/",
	},
	&cli.BoolFlag{
		Name:  "all",
		Usage: "Выводить список всех инструментов. По умолчанию выводятся только инструменты доступные для торговли через TINKOFF INVEST API.",
	},
}

func instruments(c *cli.Context) error {
	t := tinkoff.NewClient(c.String("api"), c.String("token"), c.String("data"))

	if err := t.Open(c.Context); err != nil {
		log.Fatalf("не смог открыть соединение  %s", err)
	}
	defer t.Close()

	tbl := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tbl, "Type\tExchange\tClassCode\tFigi\tIsin\tTicker\tCurrency\tName\t")

	list := proto.InstrumentStatus_INSTRUMENT_STATUS_BASE
	if c.Bool("all") {
		list = proto.InstrumentStatus_INSTRUMENT_STATUS_ALL
	}

	shares, err := t.Shares(c.Context, list)
	if err != nil {
		log.Fatalf("не смог получить список ETF  %s", err)
	}
	//TODO добавить сортировку
	for _, s := range shares {
		if !c.IsSet("status") || c.Int("status") == int(s.TradingStatus) {
			fmt.Fprintf(tbl, "share\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n", s.Exchange, s.ClassCode, s.Figi, s.Isin, s.Ticker, s.Currency, s.Name)
		}
	}

	etfs, err := t.Etfs(c.Context, list)
	if err != nil {
		log.Fatalf("не смог получить список ETF  %s", err)
	}
	for _, e := range etfs {
		if !c.IsSet("status") || c.Int("status") == int(e.TradingStatus) {
			fmt.Fprintf(tbl, "etf\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n", e.Exchange, e.ClassCode, e.Figi, e.Isin, e.Ticker, e.Currency, e.Name)
		}
	}
	tbl.Flush()

	return nil

	// TODO реализовать получения списка других инструментов
	// Bonds(ctx context.Context, in *InstrumentsRequest, opts ...grpc.CallOption) (*BondsResponse, error)
	// Currencies(ctx context.Context, in *InstrumentsRequest, opts ...grpc.CallOption) (*CurrenciesResponse, error)
	// Futures(ctx context.Context, in *InstrumentsRequest, opts ...grpc.CallOption) (*FuturesResponse, error)
}
