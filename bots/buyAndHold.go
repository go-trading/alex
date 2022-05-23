package bots

// Робот, реализующий стратегию BuyAndHold
// Не лучший пример, начинай знакомство с файла rsi.go :)

import (
	"context"

	"github.com/go-trading/alex"
)

type BuyAndHoldBot struct {
	ctx         context.Context
	cancel      context.CancelFunc
	name        string
	account     alex.Account
	instrument  alex.Instrument
	maxPosition int64
}

//Создать нового робота
func NewBuyAndHoldBot(ctx context.Context) *BuyAndHoldBot {
	botCtx, cancel := context.WithCancel(ctx)
	return &BuyAndHoldBot{
		ctx:    botCtx,
		cancel: cancel,
	}
}

//Настроить робота. Если робот не готов торговать с такими настройками, то должен вернуть ошибку
func (b *BuyAndHoldBot) Config(configs *alex.BotConfig) error {
	b.name = configs.Name
	b.account = configs.Account
	b.instrument = configs.Instrument
	b.maxPosition = int64(configs.GetIntOrDie("max-position"))
	return nil
}

//старт робота, сразу покупаю на всю котлету
func (b *BuyAndHoldBot) Start() error {
	return b.account.DoPosition(b.ctx, b, b.instrument, b.maxPosition)
}

//остановка робота, продаю всё что было куплено
func (b *BuyAndHoldBot) Stop() error {
	err := b.account.DoPosition(b.ctx, b, b.instrument, 0)
	b.cancel()
	return err
}

//реализация интервейса Bot
func (b *BuyAndHoldBot) Name() string             { return b.name }
func (b *BuyAndHoldBot) Context() context.Context { return b.ctx }
