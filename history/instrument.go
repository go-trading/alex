package history

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
	"go.uber.org/zap"
)

var _ alex.Instrument = (*instrument)(nil)

type instrument struct {
	client     *Client
	figi       string
	candles    *Candles
	lastPrices []*alex.LastPrice
	orderBook  *alex.OrderBook
	orders     []*order
	positions  map[*account]*position

	FUTURE *techan.TimeSeries
}

func newInstrument(client *Client, figi string) *instrument {
	i := &instrument{
		client:    client,
		figi:      figi,
		positions: make(map[*account]*position),
	}
	i.candles = NewCandles(figi, client, i)
	return i
}

func (i *instrument) load() (err error) {
	i.FUTURE, err = alex.LoadTimeSeries(i.client.dataDir, i.figi, time.Minute)
	return err
}

func (i *instrument) GetFigi() string                   { return i.figi }
func (i *instrument) GetTicker() string                 { return i.figi }
func (i *instrument) GetName() string                   { return i.figi }
func (i *instrument) GetExchange() string               { return "history" }
func (i *instrument) GetClassCode() string              { return "history" }
func (i *instrument) GetIsin() string                   { return "history" }
func (i *instrument) GetCurrency() string               { return "history" }
func (i *instrument) GetMinPriceIncrement() big.Decimal { return big.NewDecimal(0.01) }
func (i *instrument) IsLimitOrderAvailable() bool       { return true }
func (i *instrument) IsMarketOrderAvailable() bool      { return true }
func (i *instrument) GetLot() int32                     { return 1 }
func (i *instrument) Now() time.Time                    { return i.client.Now() }

func (i *instrument) IsStatus(tradingStatus ...proto.SecurityTradingStatus) bool {
	for _, s := range tradingStatus {
		if s == proto.SecurityTradingStatus_SECURITY_TRADING_STATUS_NORMAL_TRADING {
			return true
		}
	}
	return false
}

func (i *instrument) GetCandles(period time.Duration) alex.Candles {
	if period != time.Minute {
		l.DPanic("торги на истории доступны только на минутных свечках")
	}
	return i.candles
}
func (i *instrument) GetLastPrices(ctx context.Context) ([]*alex.LastPrice, error) {
	return i.lastPrices, nil
}
func (i *instrument) GetOrderBook(ctx context.Context, depth int32) (*alex.OrderBook, error) {
	return i.orderBook, nil
}

//геттеры для позиций
func (i *instrument) getPosition(a *account) *position { return i.positions[a] }
func (i *instrument) getBalance(a *account) int64 {
	position := i.positions[a]
	if position == nil {
		return 0
	}
	return position.balance
}
func (i *instrument) getBlocked(a *account) int64 {
	position := i.positions[a]
	if position == nil {
		return 0
	}
	return position.blocked
}
func (i *instrument) getBuy(a *account) int64 {
	position := i.positions[a]
	if position == nil {
		return 0
	}
	return position.buy
}

func (i *instrument) Tick(lastPrice *alex.LastPrice) {
	//если цена подходит текущим ордерам, то исполнить их
	for _, o := range i.orders {
		//TODO чтобы увеличить производительность, можно выделить активные ардера в отдельный список
		if o.IsActive() {
			if (o.direction == proto.OrderDirection_ORDER_DIRECTION_BUY &&
				o.InitialSecurityPrice.GTE(lastPrice.Price)) ||
				(o.direction == proto.OrderDirection_ORDER_DIRECTION_SELL &&
					o.InitialSecurityPrice.LTE(lastPrice.Price)) {
				l.Debug("Исполняю заявку",
					zap.Time("time", i.Now()),
					zap.Any("direction", o.direction),
					zap.String("order.price", o.InitialSecurityPrice.FormattedString(2)),
					zap.String("lastPrice", lastPrice.Price.FormattedString(2)),
				)
				i.fillOrder(o)
			}
		}
	}

	i.lastPrices = append(i.lastPrices, lastPrice)
	i.orderBook = &alex.OrderBook{
		Figi:  i.figi,
		Depth: 1,
		Bids: []alex.OrderBookOrder{{
			Price:    lastPrice.Price.Sub(i.GetMinPriceIncrement()),
			Quantity: 1,
		}},
		Asks: []alex.OrderBookOrder{{
			Price:    lastPrice.Price.Add(i.GetMinPriceIncrement()),
			Quantity: 1,
		}},
		LastPrice:  lastPrice.Price,
		ClosePrice: lastPrice.Price,
		LimitUp:    big.NaN,
		LimitDown:  big.NaN,
	}
	candle := i.candles.series.LastCandle()
	if candle == nil || candle.Period.End.Before(lastPrice.Time.Add(1)) {
		candle = &techan.Candle{
			Period: techan.NewTimePeriod(lastPrice.Time, time.Minute),
		}
	}
	candle.AddTrade(big.ONE, lastPrice.Price)
	i.candles.OnTick(candle)
}

func (i *instrument) PostOrder(o *order) {
	_, ok := i.positions[o.account]
	if !ok {
		i.positions[o.account] = &position{figi: i.figi}
	}
	if o.direction == proto.OrderDirection_ORDER_DIRECTION_BUY {
		i.positions[o.account].buy += o.quantity
	} else {
		i.positions[o.account].balance -= o.quantity
		i.positions[o.account].blocked += o.quantity
	}
	i.orders = append(i.orders, o)
}

func (i *instrument) fillOrder(o *order) {
	o.filledTime = o.instrument.client.now
	o.status = proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL
	if o.direction == proto.OrderDirection_ORDER_DIRECTION_BUY {
		i.positions[o.account].balance += o.quantity
		i.positions[o.account].buy -= o.quantity
	} else {
		i.positions[o.account].blocked -= o.quantity
	}

	fmt.Println(o.String())
}

func (i *instrument) cancel(o *order) (time.Time, error) {
	if o.status != proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL &&
		o.status != proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED {
		o.status = proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED
		o.cancelTime = o.instrument.client.now
		if o.direction == proto.OrderDirection_ORDER_DIRECTION_BUY {
			i.positions[o.account].buy -= o.quantity
		} else {
			i.positions[o.account].balance += o.quantity
			i.positions[o.account].blocked -= o.quantity
		}
		return o.instrument.client.now, nil
	}
	return time.Time{}, errors.New("ORDER DONT CANCELED")
}
