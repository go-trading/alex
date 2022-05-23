package tinkoff

import (
	"context"
	"time"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/sdcoffey/big"
	"go.uber.org/zap"
)

type OrderFromAPI interface {
	GetOrderId() string
	GetExecutionReportStatus() proto.OrderExecutionReportStatus
	GetLotsRequested() int64
	GetLotsExecuted() int64
	GetInitialOrderPrice() *proto.MoneyValue
	GetExecutedOrderPrice() *proto.MoneyValue
	GetTotalOrderAmount() *proto.MoneyValue
	GetInitialCommission() *proto.MoneyValue
	GetExecutedCommission() *proto.MoneyValue
	GetFigi() string
	GetDirection() proto.OrderDirection
	GetInitialSecurityPrice() *proto.MoneyValue
	GetOrderType() proto.OrderType
}

var _ alex.Order = (*BaseOrder)(nil)

type BaseOrder struct {
	account               alex.Account
	instrument            alex.Instrument
	orderId               string                           //Идентификатор заявки.
	executionReportStatus proto.OrderExecutionReportStatus //Текущий статус заявки.
	lotsRequested         int64                            //Запрошено лотов.
	lotsExecuted          int64                            //Исполнено лотов.
	initialOrderPrice     *alex.Money                      //Начальная цена заявки. Произведение количества запрошенных лотов на цену.
	executedOrderPrice    *alex.Money                      //Исполненная цена заявки. Произведение средней цены покупки на количество лотов.
	totalOrderAmount      *alex.Money                      //Итоговая стоимость заявки, включающая все комиссии.
	initialCommission     *alex.Money                      //Начальная комиссия. Комиссия рассчитанная при выставлении заявки.
	executedCommission    *alex.Money                      //Фактическая комиссия по итогам исполнения заявки.
	direction             proto.OrderDirection             //Направление сделки.
	initialSecurityPrice  *alex.Money                      //Начальная цена за 1 инструмент. Для получения стоимости лота требуется умножить на лотность инструмента.
	orderType             proto.OrderType                  //Тип заявки.
	orderDate             time.Time                        //Дата и время выставления заявки в часовом поясе UTC.
}

func (o *BaseOrder) GetFigi() string {
	return o.instrument.GetFigi()
}
func (o *BaseOrder) GetExecutionReportStatus() proto.OrderExecutionReportStatus {
	return o.executionReportStatus
}
func (o *BaseOrder) GetDirection() proto.OrderDirection {
	return o.direction
}
func (o *BaseOrder) GetLotsRequested() int64 {
	return o.lotsRequested
}
func (o *BaseOrder) GetLotsExecuted() int64 {
	return o.lotsExecuted
}
func (o *BaseOrder) GetOrderId() string {
	return o.orderId
}
func (o *BaseOrder) GetOrderDate() time.Time {
	return o.orderDate
}
func (o *BaseOrder) GetInitialOrderPrice() *alex.Money {
	return o.initialOrderPrice
}

func NewBaseOrder(o OrderFromAPI, a Account, orderDate time.Time) BaseOrder {
	return BaseOrder{
		account:               a,
		orderId:               o.GetOrderId(),
		executionReportStatus: o.GetExecutionReportStatus(),
		lotsRequested:         o.GetLotsRequested(),
		lotsExecuted:          o.GetLotsExecuted(),
		initialOrderPrice:     alex.NewMoney(o.GetInitialOrderPrice()),
		executedOrderPrice:    alex.NewMoney(o.GetExecutedOrderPrice()),
		totalOrderAmount:      alex.NewMoney(o.GetTotalOrderAmount()),
		initialCommission:     alex.NewMoney(o.GetInitialCommission()),
		executedCommission:    alex.NewMoney(o.GetExecutedCommission()),
		instrument:            a.GetClient().GetInstrument(o.GetFigi()),
		direction:             o.GetDirection(),
		initialSecurityPrice:  alex.NewMoney(o.GetInitialSecurityPrice()),
		orderType:             o.GetOrderType(),
		orderDate:             orderDate,
	}
}

//Метод отмены торгового поручения в песочнице.
func (o *BaseOrder) Cancel(ctx context.Context) (time.Time, error) {
	o.executionReportStatus = proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED
	return o.account.CancelOrder(ctx, o.GetOrderId())
}

//Определить, является ли ордер активной.
func (o *BaseOrder) IsActive() bool {
	return o.GetExecutionReportStatus() == proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_NEW ||
		o.GetExecutionReportStatus() == proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_PARTIALLYFILL
}

func (o *BaseOrder) IsBestInOrderBook(ctx context.Context) bool {
	//TODO данный метод может вызываться очень часто. Сейчас, ценой замедления спаает ratelimit, но по хорошему надо реализовывать кеширование или подписку на стаканы
	ob, err := o.instrument.GetOrderBook(ctx, 1)
	if err != nil {
		l.DPanic("GetOrderBook", zap.Error(err))
		return false
	}
	return (o.direction == proto.OrderDirection_ORDER_DIRECTION_BUY &&
		len(ob.Bids) > 0 &&
		o.initialSecurityPrice.Value.GTE(ob.Bids[0].Price)) ||
		(o.direction == proto.OrderDirection_ORDER_DIRECTION_SELL &&
			len(ob.Asks) > 0 &&
			o.initialSecurityPrice.Value.LTE(ob.Asks[0].Price))
}

var _ alex.Order = (*FromGetOrder)(nil)

//следующие атрибуты заполняется только в PostOrder
type FromPostOrder struct {
	BaseOrder
	AciValue            *alex.Money //Значение НКД (накопленного купонного дохода) на дату. Подробнее: [НКД при выставлении торговых поручений](https://tinkoff.github.io/proto/head-orders#coupon)
	Message             string      //Дополнительные данные об исполнении заявки.
	InitialOrderPricePt big.Decimal //Начальная цена заявки в пунктах (для фьючерсов).
}

//Структура, получаемая из GetOrderState / GetOrders
type FromGetOrder struct {
	BaseOrder
	AveragePositionPrice *alex.Money  //Средняя цена позиции по сделке.
	Stages               []OrderStage //Стадии выполнения заявки.
	ServiceCommission    *alex.Money  //Сервисная комиссия.
	Currency             string       //Валюта заявки.
}

//Сделки в рамках торгового поручения.
type OrderStage struct {
	BaseOrder
	Price    *alex.Money //Цена за 1 инструмент. Для получения стоимости лота требуется умножить на лотность инструмента..
	Quantity int64       //Количество лотов.
	TradeId  string      //Идентификатор торговой операции.
}

func NewOrderFromPost(a Account, o *proto.PostOrderResponse) *FromPostOrder {
	return &FromPostOrder{
		BaseOrder:           NewBaseOrder(o, a, time.Now().UTC()),
		AciValue:            alex.NewMoney(o.AciValue),
		Message:             o.Message,
		InitialOrderPricePt: alex.NewDecimal(o.InitialOrderPricePt),
	}
}

func NewOrderFromGet(a Account, o *proto.OrderState) *FromGetOrder {
	order := &FromGetOrder{
		BaseOrder:            NewBaseOrder(o, a, o.OrderDate.AsTime()),
		AveragePositionPrice: alex.NewMoney(o.AveragePositionPrice),
		ServiceCommission:    alex.NewMoney(o.ServiceCommission),
		Currency:             o.Currency,
	}
	for _, os := range o.Stages {
		order.Stages = append(order.Stages, OrderStage{
			Price:    alex.NewMoney(os.Price),
			Quantity: os.Quantity,
			TradeId:  os.TradeId,
		})
	}
	return order
}
