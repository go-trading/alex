package main

import (
	"fmt"
	"os"

	"github.com/go-trading/alex/tinkoff"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func main() {
	app := &cli.App{
		Name:     "alex",
		Usage:    "Пример использования API Тинькофф Инвестиций",
		Version:  "v0.0.1",
		Before:   before,
		After:    after,
		Flags:    globalFlags,
		Commands: сommands,
		Metadata: map[string]interface{}{"monitoring": &tinkoff.PrometheusService{}},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
	}
}

func before(c *cli.Context) error {
	if c.Bool("debug") {
		initDebugLogger()
	}
	monitoring := c.App.Metadata["monitoring"].(*tinkoff.PrometheusService)
	if monitoring != nil {
		if c.IsSet("monitoring") {
			err := monitoring.Start(c.String("monitoring"))
			if err != nil {
				l.DPanic("MonitoringService не остановлен", zap.Error(err))
			}
		}
	} else {
		l.DPanic("MonitoringService не определён")
	}
	return nil
}

func after(c *cli.Context) error {
	monitoring := c.App.Metadata["monitoring"].(*tinkoff.PrometheusService)
	if monitoring != nil {
		err := monitoring.Stop()
		if err != nil {
			l.DPanic("MonitoringService не остановлен", zap.Error(err))
		}
	} else {
		l.DPanic("MonitoringService не определён")
	}
	return nil
}
