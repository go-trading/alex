package alex

import (
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/sdcoffey/big"
)

// информация о заявке
type OrderBookOrder struct {
	Price    big.Decimal //Цена за 1 инструмент. Для получения стоимости лота требуется умножить на лотность инструмента.
	Quantity int64       //Количество в лотах.
}

func NewOrderBookOrders(orders []*proto.Order) []OrderBookOrder {
	result := make([]OrderBookOrder, len(orders))
	for i := range orders {
		result[i] = OrderBookOrder{
			Price:    NewDecimal(orders[i].Price),
			Quantity: orders[i].Quantity,
		}
	}
	return result
}

//Информация о стакане.
type OrderBook struct {
	Figi       string           //Figi-идентификатор инструмента.
	Depth      int32            //Глубина стакана.
	Bids       []OrderBookOrder //Множество пар значений на покупку.
	Asks       []OrderBookOrder //Множество пар значений на продажу.
	LastPrice  big.Decimal      //Цена последней сделки за 1 инструмент. Для получения стоимости лота требуется умножить на лотность инструмента.
	ClosePrice big.Decimal      //Цена закрытия за 1 инструмент. Для получения стоимости лота требуется умножить на лотность инструмента.
	LimitUp    big.Decimal      //Верхний лимит цены за 1 инструмент. Для получения стоимости лота требуется умножить на лотность инструмента.
	LimitDown  big.Decimal      //Нижний лимит цены за 1 инструмент. Для получения стоимости лота требуется умножить на лотность инструмента.
}

func NewOrderBook(ob *proto.GetOrderBookResponse) *OrderBook {
	return &OrderBook{
		Figi:       ob.Figi,
		Depth:      ob.Depth,
		LastPrice:  NewDecimal(ob.LastPrice),
		ClosePrice: NewDecimal(ob.ClosePrice),
		LimitUp:    NewDecimal(ob.LimitUp),
		LimitDown:  NewDecimal(ob.LimitDown),
		Asks:       NewOrderBookOrders(ob.Asks),
		Bids:       NewOrderBookOrders(ob.Bids),
	}
}
