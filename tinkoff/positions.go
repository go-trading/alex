package tinkoff

import (
	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
)

func NewPositions(responce *proto.PositionsResponse) *alex.Positions {
	result := &alex.Positions{
		LimitsLoadingInProgress: responce.LimitsLoadingInProgress,
		Positions:               make(map[string]alex.Position),
	}
	for _, m := range responce.Money {
		result.Money = append(result.Money, alex.NewMoney(m))
	}
	for _, b := range responce.Blocked {
		result.Blocked = append(result.Blocked, alex.NewMoney(b))
	}
	for _, p := range responce.Futures {
		result.Positions[p.Figi] = p
	}
	for _, p := range responce.Securities {
		result.Positions[p.Figi] = p
	}
	return result
}
