package history

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/google/uuid"
	"github.com/sdcoffey/big"
	"go.uber.org/zap"
)

var _ alex.Account = (*account)(nil)

type account struct {
	client *Client
	name   string
}

func newAccount(client *Client, name string) *account {
	return &account{
		client: client,
		name:   name,
	}
}

// геттеры, реализующие интерфейс alex.Account
func (a *account) GetId() string                  { return a.name }
func (a *account) GetType() proto.AccountType     { return proto.AccountType_ACCOUNT_TYPE_UNSPECIFIED }
func (a *account) GetEngineType() alex.EngineType { return alex.EngineType_HISTORICAL }
func (a *account) GetName() string                { return a.name }
func (a *account) GetStatus() proto.AccountStatus { return proto.AccountStatus_ACCOUNT_STATUS_OPEN }
func (a *account) GetClient() alex.Client         { return a.client }
func (a *account) GetClosedDate() time.Time       { return time.Time{} }
func (a *account) GetOpenedDate() time.Time       { return a.client.from }

func (a *account) GetAccessLevel() proto.AccessLevel {
	return proto.AccessLevel_ACCOUNT_ACCESS_LEVEL_FULL_ACCESS
}
func (a *account) PostOrder(_ context.Context, inst alex.Instrument, quantity int64, price big.Decimal, direction proto.OrderDirection, orderType proto.OrderType, orderId string) (alex.Order, error) {
	i := inst.(*instrument)
	order := newOrder(
		a,
		i,
		quantity,
		price,
		direction,
		orderType,
	)
	i.PostOrder(order)
	return order, nil
}
func (a *account) GetOrders(ctx context.Context) (result []alex.Order, _ error) {
	for _, instrument := range a.client.instruments {
		for _, o := range instrument.orders {
			if o.IsActive() && a == o.account {
				result = append(result, o)
			}
		}
	}
	return result, nil
}
func (a *account) CancelOrder(ctx context.Context, orderId string) (time.Time, error) {
	for _, instrument := range a.client.instruments {
		for _, o := range instrument.orders {
			if o.orderId == orderId {
				return o.Cancel(ctx)
			}
		}
	}
	return time.Time{}, nil
}

func (a *account) DoPosition(ctx context.Context, bot alex.Bot, instrument alex.Instrument, targetPosition int64) alex.TargetPosition {
	return a.DoPositionExtended(ctx, bot, instrument, targetPosition, big.ZERO)
}
func (a *account) DoPositionExtended(ctx context.Context, bot alex.Bot, i alex.Instrument, targetPosition int64, priceIncriment big.Decimal) alex.TargetPosition {
	hi := i.(*instrument)

	for _, o := range hi.orders {
		if o.IsActive() && o.GetOrderDate().Add(time.Minute).Before(hi.Now()) && o.isTargetPosition {
			_, _ = o.Cancel(ctx)
		}
	}

	if targetPosition != hi.getBalance(a)+hi.getBuy(a) { // текущая позиция не соответствует целевой
		if hi.getBuy(a)+hi.getBlocked(a) != 0 { // есть активные заявки
			for _, o := range hi.orders {
				_, _ = o.Cancel(context.TODO())
			}
		}
		o, err := a.PostOrderWithBestPrice(ctx, i, targetPosition-(hi.getBalance(a)+hi.getBuy(a)), priceIncriment)
		if err != nil {
			l.DPanic("WTF на исторических данных ордера должны выставляться без ошибок...")
			return nil
		}
		if o == nil {
			return nil
		}
		historyOrder := o.(*order)
		historyOrder.isTargetPosition = true
		return historyOrder
	}
	return POSITION_NOT_NEED_ORDERS
}

func (a *account) PostOrderWithBestPrice(ctx context.Context, instrument alex.Instrument, quantity int64, priceIncriment big.Decimal) (alex.Order, error) {
	if quantity == 0 {
		l.Debug("PostOrderWithBestPrice quantity == 0")
		return nil, nil
	}

	ob, err := instrument.GetOrderBook(ctx, 1)
	if err != nil {
		l.DPanic("GetOrderBook", zap.Error(err))
		return nil, nil
	}

	bestBid := big.NaN
	if len(ob.Bids) > 0 {
		bestBid = ob.Bids[0].Price
	}
	bestAsk := big.NaN
	if len(ob.Asks) > 0 {
		bestAsk = ob.Asks[0].Price
	}
	//if quantity > 0, для <0 будет переопределено ниже
	direction := proto.OrderDirection_ORDER_DIRECTION_BUY
	price := bestBid.Add(instrument.GetMinPriceIncrement())
	if quantity < 0 {
		quantity = -quantity
		direction = proto.OrderDirection_ORDER_DIRECTION_SELL
		price = bestAsk.Sub(instrument.GetMinPriceIncrement())
	}
	//на самом деле, на исторических данных, всегда выставляется по lastPrice,
	//т.к. сначала, определения стакана, к последней цене прибавляется минимальный инкримент, а потом
	//для определения цены заявки данный инкримент вычитается. Но оставляю так, на случай, если логика
	//определения цены в стакане изменится

	if price == big.NaN {
		l.Warn("лучшая цена не определена (стакан пустой?)")
		return nil, errors.New("лучшая цена не определена (стакан пустой?)")
	}

	l.Debug("выставляю заявку",
		zap.Time("time", a.client.now),
		zap.Any("direction", direction),
		zap.String("bestAsk", bestAsk.FormattedString(2)),
		zap.String("bestBid", bestBid.FormattedString(2)),
		zap.String("price", price.FormattedString(2)),
	)

	return a.PostOrder(ctx,
		instrument,
		quantity,
		price,
		direction,
		proto.OrderType_ORDER_TYPE_LIMIT,
		uuid.New().String(),
	)
}

//GetBalance реализация интерфейса Account
//Позиции храняться на инструментах, функция переадресуют запрос в инструменты
func (a *account) GetBalance(_ context.Context, i alex.Instrument) int64 {
	historyInstrument := i.(*instrument)
	return historyInstrument.getBalance(a)
}

//GetBlocked реализация интерфейса Account
//Позиции храняться на инструментах, функция переадресуют запрос в инструменты
func (a *account) GetBlocked(_ context.Context, i alex.Instrument) int64 {
	historyInstrument := i.(*instrument)
	return historyInstrument.getBlocked(a)
}

//GetPositions реализация интерфейса Account
//Позиции храняться на инструментах, функция переадресуют запрос в инструменты
func (a *account) GetPositions(_ context.Context) (*alex.Positions, error) {
	result := &alex.Positions{Positions: make(map[string]alex.Position)}
	for figi, instrument := range a.client.instruments {
		position := instrument.getPosition(a)
		if position != nil {
			result.Positions[figi] = instrument.getPosition(a)
		}
	}
	return result, nil
}

func (a *account) PrintResult() {
	for _, instrument := range a.client.instruments {
		fmt.Println("Результат по бумаге", instrument.GetFigi())
		orderFilled := 0
		orderTotal := 0
		total := big.NewDecimal(0)
		//inOrder := 0
		//Max Drawdown
		//Sharpe Ratio
		for _, o := range instrument.orders {
			if o.account == a {
				orderTotal++
				if o.status == proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL {
					orderFilled++
					if o.direction == proto.OrderDirection_ORDER_DIRECTION_BUY {
						total = total.Sub(o.InitialSecurityPrice.Mul(big.NewFromInt(int(o.quantity))))
					} else {
						total = total.Add(o.InitialSecurityPrice.Mul(big.NewFromInt(int(o.quantity))))
					}
				}
			}
		}
		openPosition := instrument.getBalance(a) + instrument.getBlocked(a)
		total = total.Add(instrument.orderBook.LastPrice.Mul(big.NewFromInt(int(openPosition))))
		fmt.Println("Открытые позиции (закрыл по последней свече)", openPosition)
		fmt.Println("Имя счёта", a.name)
		fmt.Println("Количество сделок", orderFilled)
		fmt.Println("Количество заявок", orderTotal)
		fmt.Println("Результат работы стратегии", total.FormattedString(2))
	}
}
