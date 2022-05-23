package tinkoff

import (
	"errors"
	"sync"
	"time"

	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
)

type MarketDataStream struct {
	client                        *Client
	marketDataStreamServiceClient proto.MarketDataStreamServiceClient
	marketDataStreamClient        proto.MarketDataStreamService_MarketDataStreamClient
	locker                        sync.RWMutex
	subscribers                   map[alex.Candles]alex.CandleChan
}

func NewMarketDataStream(client *Client) *MarketDataStream {
	return &MarketDataStream{
		client:                        client,
		marketDataStreamServiceClient: proto.NewMarketDataStreamServiceClient(client.conn),
		subscribers:                   make(map[alex.Candles]alex.CandleChan),
	}
}

func (s *MarketDataStream) open() (err error) {
	l.Debug("MarketDataStream.Open")
	s.marketDataStreamClient, err = s.marketDataStreamServiceClient.MarketDataStream(s.client.ctx)
	if err != nil {
		l.Error("MarketDataStream", zap.Error(err))
		return err
	}
	go s.streamReader()
	return nil
}

func (s *MarketDataStream) reconnect() {
	sleepTime := time.Second
	for {
		time.Sleep(sleepTime)
		sleepTime = 2 * sleepTime
		err := s.open()
		if err != nil {
			l.Error("MarketDataStream reconnect open", zap.Error(err))
			continue
		}
		var instruments []*proto.CandleInstrument
		for c := range s.subscribers {
			instruments = append(instruments, &proto.CandleInstrument{
				Figi:     c.GetFigi(),
				Interval: alex.Duration2SubscriptionInterval(c.GetPeriod()),
			})
		}
		err = s.sendSubscribeRequest(proto.SubscriptionAction_SUBSCRIPTION_ACTION_SUBSCRIBE, instruments...)
		if err != nil {
			l.Error("MarketDataStream reconnect sendSubscribeRequest", zap.Error(err))
			continue
		}
		break
	}
}

func (s *MarketDataStream) Subscribe(candles *Candles) alex.CandleChan {
	s.locker.Lock()
	defer s.locker.Unlock()
	ch, ok := s.subscribers[candles]
	if ok {
		return ch
	}
	ch = make(alex.CandleChan, 10)
	s.subscribers[candles] = ch
	err := s.sendSubscribeRequest(proto.SubscriptionAction_SUBSCRIPTION_ACTION_SUBSCRIBE,
		&proto.CandleInstrument{
			Figi:     candles.Figi,
			Interval: alex.Duration2SubscriptionInterval(candles.Period),
		})
	if err != nil {
		l.DPanic("не удалось подписаться на свечи на сервере.", zap.Error(err))
	}
	return ch
}

func (s *MarketDataStream) Unsubscribe(candles *Candles) error {
	s.locker.Lock()
	defer s.locker.Unlock()
	ch, ok := s.subscribers[candles]
	if !ok {
		l.DPanic("отписываюсь не подписавшись")
		return errors.New("NO SUBSCRIPTION")
	}
	close(ch)
	delete(s.subscribers, candles)
	return s.sendSubscribeRequest(proto.SubscriptionAction_SUBSCRIPTION_ACTION_UNSUBSCRIBE,
		&proto.CandleInstrument{
			Figi:     candles.Figi,
			Interval: alex.Duration2SubscriptionInterval(candles.Period),
		})
}

func (s *MarketDataStream) sendSubscribeRequest(subscriptionAction proto.SubscriptionAction, instruments ...*proto.CandleInstrument) error {
	subscribeCandlesRequest := &proto.SubscribeCandlesRequest{
		SubscriptionAction: subscriptionAction,
		Instruments:        instruments,
	}

	if err := s.marketDataStreamClient.Send(&proto.MarketDataRequest{
		Payload: &proto.MarketDataRequest_SubscribeCandlesRequest{
			SubscribeCandlesRequest: subscribeCandlesRequest,
		},
	}); err != nil {
		return err
	}

	//TODO разделить на статусы статус торгов и свечей
	//TODO BUG если будет несколько подписок свечи на разные периоды, здесь будут дубли инструментов
	var ii []*proto.InfoInstrument
	for _, inst := range instruments {
		ii = append(ii, &proto.InfoInstrument{Figi: inst.Figi})
	}

	subscribeInfoRequest := &proto.SubscribeInfoRequest{
		SubscriptionAction: subscriptionAction,
		Instruments:        ii,
	}

	if err := s.marketDataStreamClient.Send(&proto.MarketDataRequest{
		Payload: &proto.MarketDataRequest_SubscribeInfoRequest{
			SubscribeInfoRequest: subscribeInfoRequest,
		},
	}); err != nil {
		return err
	}

	return nil
}

func (s *MarketDataStream) GetCandles(figi string, interval proto.SubscriptionInterval) (alex.Candles, alex.CandleChan) {
	s.locker.RLock()
	defer s.locker.RUnlock()
	duration := alex.SubscriptionInterval2Duration(interval)
	candles := s.client.Instruments.Get(figi).GetCandles(duration)
	return candles, s.subscribers[candles]
}

func (s *MarketDataStream) streamReader() {
	for {
		marketdata, err := s.marketDataStreamClient.Recv()
		if err != nil {
			if status.Code(err) == codes.Canceled {
				l.Debug("marketDataStreamClient - закрыто соединения")
			} else if status.Code(err) == codes.ResourceExhausted {
				l.DPanic("Превышены доступные ресурсы подключения.")

			} else {
				l.Error("marketDataStreamClient получена ошибка", zap.Error(err))
				//переподключение
				s.reconnect()
			}
			return
		}
		apiCandle := marketdata.GetCandle()
		if apiCandle != nil {
			closePrice := alex.NewDecimal(apiCandle.Close)
			candles, ch := s.GetCandles(apiCandle.Figi, apiCandle.Interval)
			candle := &techan.Candle{
				Period:     techan.NewTimePeriod(apiCandle.Time.AsTime(), candles.GetPeriod()),
				OpenPrice:  alex.NewDecimal(apiCandle.Open),
				MaxPrice:   alex.NewDecimal(apiCandle.High),
				MinPrice:   alex.NewDecimal(apiCandle.Low),
				ClosePrice: closePrice,
				Volume:     big.NewFromInt(int(apiCandle.Volume)),
			}

			if len(ch) == cap(ch) {
				l.Error("переполнен поток обработки свечей. медленная работа робота? deadlock?")
			} else {
				ch <- candle
			}
			lastPriceMetric.WithLabelValues(apiCandle.Figi).Set(closePrice.Float())
		}

		tradingStatus := marketdata.GetTradingStatus()
		if tradingStatus != nil {
			if tradingStatus.Time.AsTime().Before(time.Now().Add(-time.Minute)) &&
				tradingStatus.Time.AsTime().After(time.Now().Add(time.Minute)) {
				l.DPanic("оказывается бывает, что приходят не актуальные статусы")
			}
			i := s.client.Instruments.Get(tradingStatus.Figi)
			i.SetStatus(tradingStatus.TradingStatus)
			i.SetOrderAvailable(tradingStatus.LimitOrderAvailableFlag, tradingStatus.MarketOrderAvailableFlag)
		}
	}
}
