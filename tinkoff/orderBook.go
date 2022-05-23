package tinkoff

import (
	"context"
	"sync"
	"time"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"go.uber.org/zap"
)

//Метод получения стакана по инструменту.
func (i *Instrument) getOrderBookWithoutCache(ctx context.Context, depth int32) (*alex.OrderBook, error) {
	resp, err := i.client.GetMarketDataServiceClient().GetOrderBook(ctx, &proto.GetOrderBookRequest{
		Figi:  i.GetFigi(),
		Depth: depth,
	})
	if err != nil {
		l.DPanic("GetOrderBook", zap.Error(err))
		return nil, err
	}
	return alex.NewOrderBook(resp), nil
}

type OrderBookCacheItem struct {
	Time      time.Time
	OrderBook *alex.OrderBook
}

type OrderBookCache struct {
	LiveTime   time.Duration
	locker     sync.RWMutex
	OrderBooks map[int32]OrderBookCacheItem
}

//Метод получения стакана по инструменту.
func (i *Instrument) GetOrderBook(ctx context.Context, depth int32) (*alex.OrderBook, error) {
	if i.OrderBookCache.LiveTime == 0 {
		return i.getOrderBookWithoutCache(ctx, depth)
	}
	i.OrderBookCache.locker.RLock()
	cob, ok := i.OrderBookCache.OrderBooks[depth]
	i.OrderBookCache.locker.RUnlock()

	if !ok || cob.Time.Add(i.OrderBookCache.LiveTime).Before(time.Now()) {
		i.OrderBookCache.locker.Lock()
		ob, err := i.getOrderBookWithoutCache(ctx, depth)
		if err == nil {
			cob = OrderBookCacheItem{
				Time:      time.Now(),
				OrderBook: ob,
			}
			i.OrderBookCache.OrderBooks[depth] = cob
		}
		i.OrderBookCache.locker.Unlock()
		if err != nil {
			return nil, err
		}
	}
	return cob.OrderBook, nil
}
