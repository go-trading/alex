package tinkoff

import (
	"context"
	"fmt"
	"time"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (cs *Candles) Load(ctx context.Context, from time.Time, to time.Time) error {
	maxGetCandles := time.Hour * 24
	switch {
	case cs.Period >= time.Hour*24:
		maxGetCandles = time.Hour * 24 * 365
	case cs.Period >= time.Hour:
		maxGetCandles = time.Hour * 24 * 7
	}

	sleepDuration := time.Duration(0)
	sleepTime := time.Now()
	for from.Before(to.Add(-time.Minute)) {
		requestTo := from.Add(maxGetCandles)
		if requestTo.After(to) {
			requestTo = to
		}

		now := time.Now() // не использую Until, т.к. может потребоваться брать фейковое время
		time.Sleep(sleepTime.Sub(now))

		l.Info("скачиваю",
			zap.String("figi", cs.Figi),
			zap.Duration("period", cs.Period),
			zap.Time("from", from), zap.Time("to", requestTo),
		)
		candles, err := cs.client.GetMarketDataServiceClient().GetCandles(
			ctx,
			&proto.GetCandlesRequest{
				Figi:     cs.Figi,
				From:     timestamppb.New(from),
				To:       timestamppb.New(requestTo),
				Interval: alex.Duration2CandleInterval(cs.Period),
			})
		if err != nil {
			l.Debug("ошибка скачивания исторических свеч GetCandlesRequest ", zap.Error(err))
		}
		if candles == nil {
			if sleepDuration == 0 {
				sleepDuration = time.Second
			} else {
				sleepDuration *= 2
				if sleepDuration > 300*time.Second {
					return errors.Wrap(err, fmt.Sprintf("не смог скачать %s %s %v - %v", cs.Figi, cs.Period, from, requestTo))
				}
			}
			l.Debug("свечи не скачаны((( candles == nil, повторю попытку",
				zap.Duration("sleepDuration", sleepDuration))
			sleepTime = time.Now().Add(sleepDuration)
			continue
		} else {
			sleepDuration = 0
		}
		cs.MergeApiCandles(candles.Candles)
		from = requestTo
	}

	return nil
}

//Метод запроса последних цен по инструментам.
func (i *Instrument) GetLastPrices(ctx context.Context) ([]*alex.LastPrice, error) {
	resp, err := i.client.marketDataServiceClient.GetLastPrices(ctx, &proto.GetLastPricesRequest{
		Figi: []string{i.GetFigi()},
	})
	if err != nil {
		l.DPanic("GetLastPrices", zap.Error(err))
		return nil, err
	}
	result := make([]*alex.LastPrice, len(resp.LastPrices))
	for i := range resp.LastPrices {
		result[i] = alex.NewLastPrice(
			resp.LastPrices[i].Figi,
			alex.NewDecimal(resp.LastPrices[i].Price),
			resp.LastPrices[i].Time.AsTime(),
		)
	}
	return result, nil
}
