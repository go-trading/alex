package alex

import (
	"context"
	"time"

	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/sdcoffey/big"
)

// получение информации по инструменту
type Instrument interface {
	GetCandles(period time.Duration) Candles                           // Получить интерфейс дл работы со свечами указанного периода
	GetFigi() string                                                   // Figi-идентификатор инструмента.
	GetExchange() string                                               // Торговая площадка.
	GetClassCode() string                                              // Класс-код (секция торгов).
	GetIsin() string                                                   // Isin-идентификатор инструмента.
	GetLot() int32                                                     //Количество в лоте
	GetTicker() string                                                 // Тикер  инструмента.
	GetCurrency() string                                               // Валюта расчётов.
	GetName() string                                                   // Название инструмента.
	GetLastPrices(ctx context.Context) ([]*LastPrice, error)           // Получить массив последних цен инструмента
	GetOrderBook(ctx context.Context, depth int32) (*OrderBook, error) // Получить стакан инструмента
	GetMinPriceIncrement() big.Decimal                                 // Шаг цены.
	IsStatus(tradingStatus ...proto.SecurityTradingStatus) bool        // Проверяет, является ли статус инструмента, любым из указанных в аргументах
	IsLimitOrderAvailable() bool                                       // Можно ли выставлять лимитные заявки по данному инструменту
	IsMarketOrderAvailable() bool                                      // Можно ли выставлять рыночные заявки по данному инструменту
	Now() time.Time                                                    // Текущее (для тестирования на истории)
}
