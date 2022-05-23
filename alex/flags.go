package main

// описание аргументов командной строки

import (
	"time"

	"github.com/urfave/cli/v2"
)

var (
	accountFlag = &cli.StringFlag{
		Name:     "account",
		Usage:    "Номер счёта",
		Required: true,
		EnvVars:  []string{"ALEX_ACCOUNT"},
	}
	rubFlag = &cli.Float64Flag{
		Name:  "rub",
		Usage: "Cумма в рублях",
		Value: 200000,
	}
	figisFlag = &cli.StringSliceFlag{
		Name:     "figi",
		Usage:    "Идентификатор инструмента",
		Required: true,
		EnvVars:  []string{"ALEX_FIGI"},
	}
	candlesPeriodFlag = &cli.DurationFlag{
		Name:    "candles-period",
		Value:   time.Minute,
		Usage:   "Размер свечи",
		EnvVars: []string{"ALEX_CANDLES_PERIOD"},
	}
	timeframe = &cli.IntFlag{
		Name:     "timeframe",
		Usage:    "Количество свечей, по которым надо рассчитывать RSI",
		Required: true,
		EnvVars:  []string{"ALEX_RSI_TIMEFRAME"},
	}
	rsi4buy = &cli.IntFlag{
		Name:     "rsi4buy",
		Usage:    "На каком уровне RSI покупать",
		Required: true,
		EnvVars:  []string{"ALEX_RSI_BUY"},
	}
	rsi4sell = &cli.IntFlag{
		Name:     "rsi4sell",
		Usage:    "language for the greeting",
		Required: true,
		EnvVars:  []string{"ALEX_RSI_SELL"},
	}
	maxPosition = &cli.IntFlag{
		Name:    "max-position",
		Usage:   "Максимальная позиция, доступная роботу для открытия",
		Value:   1,
		EnvVars: []string{"ALEX_MAX_POSITION"},
	}
	dataFlag = &cli.PathFlag{
		Name:    "data",
		Value:   "./data/",
		Usage:   "Каталог в котором хранятся скаченные свечи",
		EnvVars: []string{"DATA"},
	}
	fromFlag = &cli.TimestampFlag{
		Name:    "from",
		Value:   cli.NewTimestamp(time.Now().AddDate(0, 0, -7)),
		Usage:   "Время c которого нужно производить действие (В зависимости от команды: скачивать историю, или тестировать робота)",
		Layout:  "2006-01-02T15:04",
		EnvVars: []string{"ALEX_FROM"},
	}

	toFlag = &cli.TimestampFlag{
		Name:    "to",
		Value:   cli.NewTimestamp(time.Now()),
		Usage:   "Время по которое нужно производить действие (В зависимости от команды: скачивать историю, или тестировать робота)",
		Layout:  "2006-01-02T15:04",
		EnvVars: []string{"ALEX_TO"},
	}

	connectionFlags = []cli.Flag{
		&cli.StringFlag{
			Name:    "api",
			Value:   "invest-public-api.tinkoff.ru:443",
			Usage:   "host:port api tinkoff к которому требуется подключиться",
			Aliases: []string{"a"},
			EnvVars: []string{"ALEX_TINKOFF_API"},
		},
		&cli.StringFlag{
			Name:     "token",
			Usage:    "Токен, для доступа к api Tinkoff",
			Required: true,
			Aliases:  []string{"t"},
			EnvVars:  []string{"ALEX_TINKOFF_TOKEN"},
		},
	}
	globalFlags = []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			Value:   false,
			Usage:   "Устанавливает уровень логирования в debug уровень",
			Aliases: []string{"d"},
			EnvVars: []string{"ALEX_DEBUG"},
		},
		&cli.StringFlag{
			Name:    "monitoring",
			Usage:   "Адрес, по которому включить метрики prometeus. Например :8080",
			Aliases: []string{"m"},
			EnvVars: []string{"ALEX_MONITORING"},
		},
	}
)
