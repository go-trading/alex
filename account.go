package alex

import (
	"context"
	"time"

	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/sdcoffey/big"
)

//Тип счёта.
type EngineType int32

const (
	EngineType_UNSPECIFIED EngineType = iota // значение не определено
	EngineType_HISTORICAL  EngineType = iota // счёт для тестирования на истории
	EngineType_SANDBOX     EngineType = iota // счёт для песочницы
	EngineType_REAL        EngineType = iota // боевой счёт
)

// описание счёта (совместим с proto интерфейсами tinkoff инвестиции)
type AccountDescription interface {
	GetId() string
	GetType() proto.AccountType
	GetEngineType() EngineType
	GetName() string
	GetStatus() proto.AccountStatus
	GetAccessLevel() proto.AccessLevel
	GetClosedDate() time.Time
	GetOpenedDate() time.Time
}

// реализация торговых поручений
type AccountEngine interface {
	GetPositions(ctx context.Context) (*Positions, error)
	GetOrders(ctx context.Context) ([]Order, error)
	GetBalance(ctx context.Context, i Instrument) int64
	GetBlocked(ctx context.Context, i Instrument) int64
	PostOrder(ctx context.Context, instrument Instrument, quantity int64, price big.Decimal, direction proto.OrderDirection, orderType proto.OrderType, orderId string) (Order, error)
	CancelOrder(ctx context.Context, orderId string) (time.Time, error)
	GetClient() Client // возвращает интерфейс Client, в рамках которого создан данный счёт
}

// функции, позволяющие писать робота в парадигме "сейчас нужна позиция XXX", вместо нужно выставить сделку
type AccountSmart interface {
	// стремиться достичь требуемую позицию используя лимитированные заявки с ценой на минимальный шаг лучше текущей лучшей цены
	DoPosition(ctx context.Context, bot Bot, instrument Instrument, targetPosition int64) TargetPosition
	// стремиться достичь требуемую позицию используя лимитированные заявки с ценой на priceIncriment лучше текущей лучшей цены
	DoPositionExtended(ctx context.Context, bot Bot, instrument Instrument, targetPosition int64, priceIncriment big.Decimal) TargetPosition
	// отправляет лимитированную заявку с ценой на priceIncriment лучше текущей лучшей цены
	PostOrderWithBestPrice(ctx context.Context, instrument Instrument, quantity int64, priceIncriment big.Decimal) (Order, error)
}

// Интерфейс торгового счёта
type Account interface {
	AccountDescription
	AccountEngine
	AccountSmart
}

// объект возвращаемый DoPosition, который сообщает о статусе достежения требуемой позиции, и об ошибках, если они были
type TargetPosition interface {
	Error() string
	GetError() error
	IsLimitError() bool
}
