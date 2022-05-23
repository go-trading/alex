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

//статическая проверка, что тип SandboxAccount реализует интерфейс alex.Account
var _ alex.Account = (*SandboxAccount)(nil)

type SandboxAccount struct {
	*AccountAbstract
}

func NewSandboxAccount(c *Client, ad *proto.Account) alex.Account {
	this := &SandboxAccount{
		AccountAbstract: NewAccountAbstract(c, ad),
	}
	this.engine = this
	return this
}

func (sa *SandboxAccount) PostOrder(ctx context.Context, instrument alex.Instrument, quantity int64, price big.Decimal, direction proto.OrderDirection, orderType proto.OrderType, orderId string) (alex.Order, error) {
	resp, err := sa.client.GetSandboxServiceClient().PostSandboxOrder(ctx, &proto.PostOrderRequest{
		Figi:      instrument.GetFigi(),
		Quantity:  quantity,
		Price:     alex.NewQuotation(price),
		Direction: direction,
		AccountId: sa.id,
		OrderType: orderType,
		OrderId:   orderId,
	})
	sa.cache.invalidateCache()
	if err != nil {
		l.Error("PostSandboxOrder", zap.Error(err))
		return nil, err
	}
	return NewOrderFromPost(sa, resp), nil
}

//Метод отмены торгового поручения в песочнице.
func (sa SandboxAccount) CancelOrder(ctx context.Context, orderId string) (time.Time, error) {
	resp, err := sa.client.GetSandboxServiceClient().CancelSandboxOrder(ctx, &proto.CancelOrderRequest{
		AccountId: sa.id,
		OrderId:   orderId,
	})
	sa.cache.invalidateCache()
	if err != nil {
		if status.Code(err) == codes.NotFound {
			l.Debug("CancelSandboxOrder NotFound", zap.String("orderId", orderId))
		} else {
			l.DPanic("CancelSandboxOrder", zap.Error(err), zap.String("orderId", orderId))
		}
		return time.Time{}, err
	}
	return resp.Time.AsTime(), nil
}

//Метод получения списка активных заявок по счёту в песочнице.
func (sa *SandboxAccount) GetOrders(ctx context.Context) ([]alex.Order, error) {
	resp, err := sa.client.GetSandboxServiceClient().GetSandboxOrders(ctx, &proto.GetOrdersRequest{
		AccountId: sa.id,
	})
	if err != nil {
		l.Error("GetSandboxOrders", zap.Error(err))
		return nil, err
	}
	orders := make([]alex.Order, len(resp.Orders))
	for i, o := range resp.Orders {
		orders[i] = NewOrderFromGet(sa, o)
	}
	return orders, nil
}

//Метод получения позиций по виртуальному счёту песочницы.
func (sa *SandboxAccount) GetPositions(ctx context.Context) (*alex.Positions, error) {
	resp, err := sa.client.sandboxServiceClient.GetSandboxPositions(ctx, &proto.PositionsRequest{
		AccountId: sa.id,
	})
	if err != nil {
		l.Error("GetSandboxPositions", zap.Error(err))
		return nil, err
	}
	return NewPositions(resp), err
}

func (sa *SandboxAccount) GetEngineType() alex.EngineType {
	return alex.EngineType_SANDBOX
}
