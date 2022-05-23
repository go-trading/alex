package tinkoff

import (
	"context"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/sdcoffey/big"
	"go.uber.org/zap"
)

//Портфель по счёту.
type Portfolio struct {
	TotalAmountShares     *alex.Money          //Общая стоимость акций в портфеле в рублях.
	TotalAmountBonds      *alex.Money          //Общая стоимость облигаций в портфеле в рублях.
	TotalAmountEtf        *alex.Money          //Общая стоимость фондов в портфеле в рублях.
	TotalAmountCurrencies *alex.Money          //Общая стоимость валют в портфеле в рублях.
	TotalAmountFutures    *alex.Money          //Общая стоимость фьючерсов в портфеле в рублях.
	ExpectedYield         big.Decimal          //Текущая относительная доходность портфеля, в %.
	Positions             []*PortfolioPosition //Список позиций портфеля.
}

func NewPortfolio(responce *proto.PortfolioResponse) *Portfolio {
	result := &Portfolio{
		TotalAmountShares:     alex.NewMoney(responce.TotalAmountShares),
		TotalAmountBonds:      alex.NewMoney(responce.TotalAmountBonds),
		TotalAmountEtf:        alex.NewMoney(responce.TotalAmountEtf),
		TotalAmountCurrencies: alex.NewMoney(responce.TotalAmountCurrencies),
		TotalAmountFutures:    alex.NewMoney(responce.TotalAmountFutures),
		ExpectedYield:         alex.NewDecimal(responce.ExpectedYield),
	}
	for _, p := range responce.Positions {
		result.Positions = append(result.Positions, NewPortfolioPosition(p))
	}
	return result
}

//Позиции портфеля.
type PortfolioPosition struct {
	Figi                     string      //Figi-идентификатора инструмента.
	InstrumentType           string      //Тип инструмента.
	Quantity                 big.Decimal //Количество инструмента в портфеле в штуках.
	AveragePositionPrice     *alex.Money //Средневзвешенная цена позиции. **Возможна задержка до секунды для пересчёта**.
	ExpectedYield            big.Decimal //Текущая рассчитанная относительная доходность позиции, в %.
	CurrentNkd               *alex.Money // Текущий НКД.
	AveragePositionPricePt   big.Decimal //Средняя цена лота в позиции в пунктах (для фьючерсов). **Возможна задержка до секунды для пересчёта**.
	CurrentPrice             *alex.Money //Текущая цена за 1 инструмент. Для получения стоимости лота требуется умножить на лотность инструмента..
	AveragePositionPriceFifo *alex.Money //Средняя цена лота в позиции по методу FIFO. **Возможна задержка до секунды для пересчёта**.
	QuantityLots             big.Decimal //Количество лотов в портфеле.
}

func NewPortfolioPosition(responce *proto.PortfolioPosition) *PortfolioPosition {
	return &PortfolioPosition{
		Figi:                     responce.Figi,
		InstrumentType:           responce.InstrumentType,
		Quantity:                 alex.NewDecimal(responce.Quantity),
		AveragePositionPrice:     alex.NewMoney(responce.AveragePositionPrice),
		ExpectedYield:            alex.NewDecimal(responce.ExpectedYield),
		CurrentNkd:               alex.NewMoney(responce.CurrentNkd),
		AveragePositionPricePt:   alex.NewDecimal(responce.AveragePositionPricePt),
		CurrentPrice:             alex.NewMoney(responce.CurrentPrice),
		AveragePositionPriceFifo: alex.NewMoney(responce.AveragePositionPriceFifo),
		QuantityLots:             alex.NewDecimal(responce.QuantityLots),
	}
}

func (c *Client) GetPortfolio(ctx context.Context, accountId string) (*Portfolio, error) {
	resp, err := c.GetOperationsServiceClient().GetPortfolio(ctx, &proto.PortfolioRequest{
		AccountId: accountId,
	})
	if err != nil {
		l.DPanic("GetPortfolio", zap.Error(err))
	}
	return NewPortfolio(resp), err
}
