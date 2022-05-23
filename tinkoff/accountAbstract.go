package tinkoff

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/sdcoffey/big"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Account interface {
	alex.Account
}

type AccountAbstract struct {
	client          *Client
	id              string
	accountType     proto.AccountType
	name            string
	status          proto.AccountStatus
	accessLevel     proto.AccessLevel
	closeDate       time.Time
	openDate        time.Time
	targetPositions *TargetPositions
	engine          alex.AccountEngine
	cache           *accountCache
}

func NewAccountAbstract(client *Client, ad *proto.Account) *AccountAbstract {
	closedDate := ad.GetClosedDate()
	if closedDate == nil {
		closedDate = &timestamppb.Timestamp{} //set zero time
	}

	account := &AccountAbstract{
		client:          client,
		id:              ad.GetId(),
		accountType:     ad.GetType(),
		name:            ad.GetName(),
		status:          ad.GetStatus(),
		accessLevel:     ad.GetAccessLevel(),
		closeDate:       closedDate.AsTime(),
		openDate:        ad.GetOpenedDate().AsTime(),
		targetPositions: NewTargetPositions(),
	}
	account.cache = newAccountCache(account, 10*time.Second, 10*time.Second) // TODO перенести таймауты в настройки
	return account
}

func (a *AccountAbstract) GetId() string                       { return a.id }
func (a *AccountAbstract) GetType() proto.AccountType          { return a.accountType }
func (a *AccountAbstract) GetName() string                     { return a.name }
func (a *AccountAbstract) GetStatus() proto.AccountStatus      { return a.status }
func (a *AccountAbstract) GetAccessLevel() proto.AccessLevel   { return a.accessLevel }
func (a *AccountAbstract) GetClosedDate() time.Time            { return a.closeDate }
func (a *AccountAbstract) GetOpenedDate() time.Time            { return a.openDate }
func (a *AccountAbstract) GetTargetPosition() *TargetPositions { return a.targetPositions }
func (a *AccountAbstract) GetClient() alex.Client              { return a.client }
func (a *AccountAbstract) GetTinkoffClient() *Client           { return a.client }

func (a *AccountAbstract) GetPositions(ctx context.Context) (*alex.Positions, error) {
	return a.cache.GetPositions(ctx)
}

func (a *AccountAbstract) GetOrders(ctx context.Context) ([]alex.Order, error) {
	return a.cache.GetOrders(ctx)
}
func (a *AccountAbstract) GetBalance(ctx context.Context, i alex.Instrument) int64 {
	return a.cache.GetBalance(ctx, i)
}
func (a *AccountAbstract) GetBlocked(ctx context.Context, i alex.Instrument) int64 {
	return a.cache.GetBlocked(ctx, i)
}

func (a *AccountAbstract) PostOrder(ctx context.Context, instrument alex.Instrument, quantity int64, price big.Decimal, direction proto.OrderDirection, orderType proto.OrderType, orderId string) (alex.Order, error) {
	return a.engine.PostOrder(ctx, instrument, quantity, price, direction, orderType, orderId)
}
func (a *AccountAbstract) CancelOrder(ctx context.Context, orderId string) (time.Time, error) {
	return a.engine.CancelOrder(ctx, orderId)
}

func AccountStringTableHead() string {
	return "Id\tType\tName\tStatus\tOpenedDate\tClosedDate\tAccessLevel\t"
}
func (a *AccountAbstract) String() string {
	closedDate := a.GetClosedDate().Format("2006-01-02")
	if closedDate == "1970-01-01" {
		closedDate = ""
	}
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
		a.GetId(),
		strings.Replace(a.GetType().String(), "ACCOUNT_TYPE_", "", 1),
		a.GetName(),
		strings.Replace(a.GetStatus().String(), "ACCOUNT_STATUS_", "", 1),
		a.GetOpenedDate().Format("2006-01-02"),
		closedDate,
		strings.Replace(a.GetAccessLevel().String(), "ACCOUNT_ACCESS_LEVEL_", "", 1),
	)
}
