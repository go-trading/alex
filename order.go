package alex

import (
	"context"
	"time"

	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
)

// информация о торговом поручении
type Order interface {
	GetFigi() string                                            //Идентификатор инструмента.
	GetExecutionReportStatus() proto.OrderExecutionReportStatus //Текущий статус заявки.
	GetDirection() proto.OrderDirection                         //Направление сделки.
	GetLotsRequested() int64                                    //Запрошено лотов.
	GetLotsExecuted() int64                                     //Исполнено лотов.
	GetOrderId() string                                         // Идентификатор заявки
	GetOrderDate() time.Time                                    //Дата и время выставления заявки в часовом поясе UTC.
	GetInitialOrderPrice() *Money                               //Начальная цена заявки. Произведение количества запрошенных лотов на цену.
	Cancel(ctx context.Context) (time.Time, error)              // Отменить заявку
	IsActive() bool                                             // Является ли заявка активной
	IsBestInOrderBook(ctx context.Context) bool                 // Является ли заявка лучшей в стакане
}
