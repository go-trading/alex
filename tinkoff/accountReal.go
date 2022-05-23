package tinkoff

import (
	"context"
	"time"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/sdcoffey/big"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//статическая проверка, что тип RealAccount реализует интерфейс alex.Account
var _ alex.Account = (*RealAccount)(nil)

type RealAccount struct {
	*AccountAbstract
}

func NewRealAccount(ctx context.Context, client *Client, ad *proto.Account) alex.Account {
	this := &RealAccount{
		AccountAbstract: NewAccountAbstract(client, ad),
	}
	this.engine = this
	l.Debug("подписываюсь на сделки по счёту", zap.String("account", ad.Id))
	ch, err := client.orderTrades.Subscribe()
	if err != nil {
		l.DPanic("не смог подписаться на сделки по счёту", zap.Error(err))
	}
	go this.cache.ReadOrderTrades(ch)
	return this
}

func (ra *RealAccount) PostOrder(ctx context.Context, instrument alex.Instrument, quantity int64, price big.Decimal, direction proto.OrderDirection, orderType proto.OrderType, orderId string) (alex.Order, error) {
	postOrderRequest := &proto.PostOrderRequest{
		Figi:      instrument.GetFigi(),
		Quantity:  quantity,
		Price:     alex.NewQuotation(price),
		Direction: direction,
		AccountId: ra.id,
		OrderType: orderType,
		OrderId:   orderId,
	}
	resp, err := ra.client.GetOrdersServiceClient().PostOrder(ctx, postOrderRequest)
	l.Debug("PostOrder",
		zap.Any("req", postOrderRequest),
		zap.Any("resp", resp),
		zap.Error(err),
	)
	if err != nil {
		l.Error("PostOrder", zap.Error(err))
		return nil, err
	}
	ra.cache.invalidateCache()
	return NewOrderFromPost(ra, resp), nil
}

//Метод отмены торгового поручения.
func (ra *RealAccount) CancelOrder(ctx context.Context, orderId string) (time.Time, error) {
	cancelOrderReq := &proto.CancelOrderRequest{
		AccountId: ra.id,
		OrderId:   orderId,
	}
	resp, err := ra.client.GetOrdersServiceClient().CancelOrder(ctx, cancelOrderReq)
	// ловил здесь, однократно, на работу программы не повлияло. Будет повторятся, присмотреться
	// 70002	INTERNAL	internal network error	Неизвестная сетевая ошибка, попробуйте выполнить запрос позднее.

	ra.cache.invalidateCache()
	if err != nil {
		if status.Code(err) == codes.NotFound {
			l.Debug("CancelOrder NotFound", zap.String("orderId", orderId))
		} else {
			l.Error("CancelOrder", zap.Error(err), zap.Any("req", cancelOrderReq))
		}
		return time.Time{}, err
	}
	return resp.Time.AsTime(), nil
}

//Метод получения списка активных заявок по счёту в песочнице.
func (ra *RealAccount) GetOrders(ctx context.Context) ([]alex.Order, error) {
	resp, err := ra.client.GetOrdersServiceClient().GetOrders(ctx, &proto.GetOrdersRequest{
		AccountId: ra.id,
	})
	if err != nil {
		l.Error("GetOrders", zap.Error(err))
		return nil, err
	}
	orders := make([]alex.Order, len(resp.Orders))
	for i, o := range resp.Orders {
		orders[i] = NewOrderFromGet(ra, o)
	}
	return orders, nil
}

//Метод получения позиций по счёту.
func (ra *RealAccount) GetPositions(ctx context.Context) (*alex.Positions, error) {
	resp, err := ra.client.GetOperationsServiceClient().GetPositions(ctx, &proto.PositionsRequest{
		AccountId: ra.id,
	})
	if err != nil {
		l.Error("GetPositions", zap.Error(err))
		return nil, err
	}
	positions := NewPositions(resp)
	// сохраняю текущие позиции для прометеуса
	if positions != nil {
		for _, m := range positions.Money {
			positionMetric.WithLabelValues(ra.GetId(), m.Currency).Set(m.Value.Float())
		}
		for _, p := range positions.Positions {
			balanceMetric.WithLabelValues(ra.GetId(), p.GetFigi()).Set(
				float64(p.GetBalance()) + float64(p.GetBlocked()),
			)
		}
	}
	return positions, nil
}

func (ra *RealAccount) GetEngineType() alex.EngineType {
	return alex.EngineType_REAL
}
