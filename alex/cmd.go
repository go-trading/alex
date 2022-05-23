package main

// В файле описаны все команды, доступные в командной строке

import (
	"github.com/urfave/cli/v2"
)

var сommands = []*cli.Command{
	{
		Name:   "load",
		Usage:  "Загрузка исторических свечей  (Скачать данные в csv)",
		Action: load,
		Flags:  append(connectionFlags, dataFlag, fromFlag, toFlag, figisFlag, candlesPeriodFlag),
	}, {
		Name:  "online",
		Usage: "Отслеживать данные по торгам в режиме реального времени",
		Subcommands: []*cli.Command{{
			Name:   "candle",
			Usage:  "Отслеживать данные по свечам",
			Action: onlineCandle,
			Flags:  connectionFlags,
		}},
	}, {
		Name:   "instruments",
		Usage:  "Вывести список инструментов и их коды",
		Action: instruments,
		Flags:  append(connectionFlags, instrumentsFlags...),
	}, {
		Name:   "accounts",
		Usage:  "Вывести список аккаунтов",
		Action: accounts,
		Flags:  connectionFlags,
	}, {
		Name:  "bot",
		Usage: "Группа команд для работы с роботом",
		Subcommands: []*cli.Command{
			{
				Name:   "rsi",
				Usage:  "Запустить RSI робота на выбранном счёте (если будет указан боевой счёт, то робот начнёт торговать на нём)",
				Action: botRun,
				Flags:  append(connectionFlags, accountFlag, figisFlag, candlesPeriodFlag, timeframe, maxPosition, rsi4buy, rsi4sell),
			},
			{
				Name:   "BestInOrderbook",
				Usage:  "Запустить BestInOrderbook робота на выбранном счёте (если будет указан боевой счёт, то робот начнёт торговать на нём)",
				Action: botRunBestInOrderbookBot,
				Flags:  append(connectionFlags, accountFlag, figisFlag, maxPosition),
			},
			{
				Name:   "history",
				Usage:  "Протестировать робота RSI на истории. История должна быть заранее скачана командой load.",
				Action: botHistory,
				Flags:  []cli.Flag{dataFlag, fromFlag, toFlag, figisFlag, timeframe, maxPosition, rsi4buy, rsi4sell},
			}},
	}, {
		Name:  "sandbox",
		Usage: "Группа команд по работа со счетами песочницы",
		Subcommands: []*cli.Command{{
			Name:   "open",
			Usage:  "Регистрации счёта в песочнице",
			Action: sandboxOpenAccount,
			Flags:  connectionFlags,
		}, {
			Name:   "close",
			Usage:  "Закрытие счёта в песочнице",
			Action: sandboxCloseAccount,
			Flags:  append(connectionFlags, accountFlag),
		}, {
			Name:   "pay-in",
			Usage:  "Пополнить счёт",
			Action: sandboxPayIn,
			Flags:  append(connectionFlags, accountFlag, rubFlag),
		}},
	},
}
