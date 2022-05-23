package bots

// Робот демонстрирующий возможность торговли в стакане
// Не лучший пример, начинай знакомство с файла rsi.go :)
// Не доступен для тестирования на свечах, т.к. торгует в стакане

import (
	"context"
	"time"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sdcoffey/big"
)

type BestInOrderbookBot struct {
	ctx             context.Context
	cancel          context.CancelFunc
	name            string
	account         alex.Account
	instrument      alex.Instrument
	candles         alex.Candles
	candlesChan     alex.CandleChan
	maxPositionLots int64
	sleepTime       time.Time
}

//Создать нового робота
func NewBestInOrderbookBot(ctx context.Context) *BestInOrderbookBot {
	botCtx, cancel := context.WithCancel(ctx)
	return &BestInOrderbookBot{
		ctx:    botCtx,
		cancel: cancel,
	}
}

//Настроить робота. Если робот не готов торговать с такими настройками, то должен вернуть ошибку
func (b *BestInOrderbookBot) Config(configs *alex.BotConfig) error {
	b.name = configs.Name
	b.account = configs.Account
	b.instrument = configs.Instrument
	b.candles = configs.Instrument.GetCandles(time.Minute)
	b.maxPositionLots = int64(configs.GetIntOrDie("max-position"))
	return nil
}

//Начать торговлю
func (b *BestInOrderbookBot) Start() (err error) {
	b.candlesChan, err = b.candles.Subscribe()
	go b.botLoop()
	return err
}

//Остановить торговлю
func (b *BestInOrderbookBot) Stop() error {
	b.cancel()
	return b.account.DoPosition(b.ctx, b, b.instrument, 0)
}

//Основной цикл, в котором получаю информацию о свечах, и передаю в функцию принятия торгового решения
func (b *BestInOrderbookBot) botLoop() {
	for {
		select {
		case <-b.candlesChan:
			if len(b.candlesChan) == 0 && b.instrument.Now().After(b.sleepTime) {
				b.sleepTime = b.instrument.Now().Add(5 * time.Second)
				timer := prometheus.NewTimer(botDurationMetric.WithLabelValues(b.name))
				b.OnCandle()
				timer.ObserveDuration()
			}
		case <-b.ctx.Done():
			b.account.GetClient().Printf("Завершаю обработку свечей роботом.\n")
			return
		}
	}
}

//Обработка пришедших свечей. Принятие решения о покупке/продаже принимается здесь
func (b *BestInOrderbookBot) OnCandle() {
	if !b.instrument.IsStatus(proto.SecurityTradingStatus_SECURITY_TRADING_STATUS_NORMAL_TRADING) {
		return
	}
	balanceLots := b.account.GetBalance(b.ctx, b.instrument) / int64(b.instrument.GetLot())
	blockedLots := b.account.GetBlocked(b.ctx, b.instrument) / int64(b.instrument.GetLot())

	// рассчитываю, какие объёмы всего должны быть в ордерах на покупку и продажу
	needSellTotalLots := balanceLots + blockedLots
	needBuyTotalLots := b.maxPositionLots - needSellTotalLots
	orders, err := b.account.GetOrders(b.ctx)
	if err != nil {
		panic(err)
	}
	//рассчитываю, какое количество уже сейчас выставленно, паралельно отменяю заявки, не по лучшей цене
	inOrderSellLots := int64(0)
	inOrderBuyLots := int64(0)
	for _, o := range orders {
		if o.GetFigi() == b.instrument.GetFigi() && o.IsActive() {
			if !o.IsBestInOrderBook(b.ctx) {
				_, err = o.Cancel(b.ctx)
				if err != nil {
					_ = b.Stop()
				}
				return
			} else {
				if o.GetDirection() == proto.OrderDirection_ORDER_DIRECTION_BUY {
					inOrderBuyLots += o.GetLotsRequested() - o.GetLotsExecuted()
				} else {
					inOrderSellLots += o.GetLotsRequested() - o.GetLotsExecuted()
				}
			}
		}
	}

	//получаю, сколько сколько недостаёт в заявках
	needSellLots := needSellTotalLots - inOrderSellLots
	needBuyLots := needBuyTotalLots - inOrderBuyLots

	if needSellLots <= 0 && needBuyLots <= 0 {
		return
	}
	ob, _ := b.instrument.GetOrderBook(b.ctx, 1)

	// выставляю заявки
	if needSellLots > 0 && len(ob.Asks) > 0 && b.instrument.IsLimitOrderAvailable() {
		_, _ = b.account.PostOrder(b.ctx, b.instrument, needSellLots, ob.Asks[0].Price.Sub(big.NewDecimal(0.01)), proto.OrderDirection_ORDER_DIRECTION_SELL, proto.OrderType_ORDER_TYPE_LIMIT, uuid.NewString())
	}
	// аналогично продаже
	if needBuyLots > 0 && len(ob.Bids) > 0 && b.instrument.IsLimitOrderAvailable() {
		_, _ = b.account.PostOrder(b.ctx, b.instrument, needBuyLots, ob.Bids[0].Price.Add(big.NewDecimal(0.01)), proto.OrderDirection_ORDER_DIRECTION_BUY, proto.OrderType_ORDER_TYPE_LIMIT, uuid.NewString())
	}
}

//реализация интервейса Bot
func (b *BestInOrderbookBot) Name() string             { return b.name }
func (b *BestInOrderbookBot) Context() context.Context { return b.ctx }
