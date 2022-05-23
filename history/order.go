package history

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/google/uuid"
	"github.com/sdcoffey/big"
	"go.uber.org/zap"
)

var _ alex.Order = (*order)(nil)
var _ alex.TargetPosition = (*order)(nil)
var POSITION_NOT_NEED_ORDERS alex.TargetPosition = (*order)(&order{})

type order struct {
	account              *account
	instrument           *instrument
	quantity             int64
	InitialSecurityPrice big.Decimal
	direction            proto.OrderDirection
	orderType            proto.OrderType
	status               proto.OrderExecutionReportStatus
	orderId              string
	orderDate            time.Time
	cancelTime           time.Time
	filledTime           time.Time
	isTargetPosition     bool
}

func newOrder(account *account, instrument *instrument, quantity int64, price big.Decimal, direction proto.OrderDirection, orderType proto.OrderType) *order {
	return &order{
		account:              account,
		instrument:           instrument,
		quantity:             quantity,
		InitialSecurityPrice: price,
		direction:            direction,
		orderType:            orderType,
		status:               proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_NEW,
		orderId:              uuid.NewString(),
		orderDate:            instrument.client.now,
	}
}

func (o *order) GetFigi() string                                            { return o.instrument.GetFigi() }
func (o *order) GetExecutionReportStatus() proto.OrderExecutionReportStatus { return o.status }
func (o *order) GetDirection() proto.OrderDirection                         { return o.direction }
func (o *order) GetLotsRequested() int64                                    { return o.quantity }
func (o *order) GetOrderId() string                                         { return o.orderId }
func (o *order) GetOrderDate() time.Time                                    { return o.orderDate }

func (o *order) GetLotsExecuted() int64 {
	if o.status == proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL {
		return o.quantity
	}
	return 0
}

//Начальная цена заявки. Произведение количества запрошенных лотов на цену.
func (o *order) GetInitialOrderPrice() *alex.Money {
	return &alex.Money{
		Currency: o.instrument.GetCurrency(),
		Value:    o.InitialSecurityPrice.Mul(big.NewFromInt(int(o.quantity))),
	}
}
func (o *order) Cancel(ctx context.Context) (time.Time, error) {
	return o.instrument.cancel(o)
}
func (o *order) IsActive() bool {
	return o.GetExecutionReportStatus() == proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_NEW ||
		o.GetExecutionReportStatus() == proto.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_PARTIALLYFILL
}
func (o *order) IsBestInOrderBook(ctx context.Context) bool {
	ob, err := o.instrument.GetOrderBook(ctx, 1)
	if err != nil {
		l.DPanic("GetOrderBook", zap.Error(err))
		return false
	}
	return (o.direction == proto.OrderDirection_ORDER_DIRECTION_BUY &&
		len(ob.Bids) > 0 &&
		o.InitialSecurityPrice.GTE(ob.Bids[0].Price)) ||
		(o.direction == proto.OrderDirection_ORDER_DIRECTION_SELL &&
			len(ob.Asks) > 0 &&
			o.InitialSecurityPrice.LTE(ob.Asks[0].Price))
}

//TargetPosition interface
func (o *order) Error() string      { return "" }
func (o *order) GetError() error    { return nil }
func (o *order) IsLimitError() bool { return false }

//Stringer interface
func (o *order) String() string {
	return o.filledTime.Format("2006-01-02 15:04") + "\t" +
		o.instrument.figi + "\t" +
		strings.ReplaceAll(o.direction.String(), "ORDER_DIRECTION_", "") + "\t" +
		strconv.Itoa(int(o.quantity)) + "\t" +
		o.InitialSecurityPrice.FormattedString(2)
}
