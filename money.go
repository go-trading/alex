package alex

import (
	mathbig "math/big"
	"time"

	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/sdcoffey/big"
)

type Money struct {
	Currency string      // строковый ISO-код валюты
	Value    big.Decimal // сумма
}

func NewMoney(mv *proto.MoneyValue) *Money {
	if mv == nil {
		return nil
	}
	return &Money{
		Currency: mv.Currency,
		Value:    UnitsNano2Decimal(mv.Units, mv.Nano),
	}
}

func NewDecimal(q *proto.Quotation) big.Decimal {
	if q == nil {
		return big.NaN
	}
	return UnitsNano2Decimal(q.Units, q.Nano)
}

var big10_9 = big.NewFromInt(1000000000)
var int10_9 = mathbig.NewInt(1000000000)

func NewQuotation(d big.Decimal) *proto.Quotation {
	units, _ := new(mathbig.Float).SetFloat64(d.Float()).Int(nil)
	mul10_9, _ := new(mathbig.Float).SetFloat64(d.Mul(big10_9).Float()).Int(nil)

	return &proto.Quotation{
		Units: units.Int64(),
		Nano:  int32(mul10_9.Sub(mul10_9, units.Mul(units, int10_9)).Int64()),
	}
}

func UnitsNano2Decimal(units int64, nano int32) big.Decimal {
	return big.NewFromInt(int(units)).
		Add(
			big.NewFromInt(int(nano)).Div(big10_9),
		)
}

func NewMoneyValue(m *Money) *proto.MoneyValue {
	quotation := NewQuotation(m.Value)
	return &proto.MoneyValue{
		Currency: m.Currency,
		Units:    quotation.Units,
		Nano:     quotation.Nano,
	}
}

//Информация о цене.
type LastPrice struct {
	Figi  string      //Идентификатор инструмента.
	Price big.Decimal //Последняя цена за 1 инструмент. Для получения стоимости лота требуется умножить на лотность инструмента.
	Time  time.Time   //Время получения последней цены в часовом поясе UTC по времени биржи.
}

func NewLastPrice(figi string, price big.Decimal, time time.Time) *LastPrice {
	return &LastPrice{
		Figi:  figi,
		Price: price,
		Time:  time,
	}
}
