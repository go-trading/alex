package tinkoff

import (
	"context"
	"sync"
	"time"

	"github.com/go-trading/alex"
	"go.uber.org/zap"
)

type accountCache struct {
	account                  *AccountAbstract
	locker                   sync.RWMutex
	positionsInvalidateTimer *time.Timer
	ordersInvalidateTimer    *time.Timer
	orders                   []alex.Order
	ordersLiveTime           time.Duration
	ordersRequestTime        time.Time
	positions                *alex.Positions
	positionsLiveTime        time.Duration
	positionsRequestTime     time.Time
}

func newAccountCache(account *AccountAbstract, ordersLiveTime time.Duration, positionsLiveTime time.Duration) *accountCache {
	return &accountCache{
		account:           account,
		ordersLiveTime:    ordersLiveTime,
		positionsLiveTime: positionsLiveTime,
	}
}

func (ac *accountCache) invalidateCache() {
	l.Debug("accountCache.invalidateCache")
	ac.locker.Lock()
	ac.ordersRequestTime = time.Time{}
	ac.positionsRequestTime = time.Time{}
	ac.locker.Unlock()
	ac.account.doTracking(context.TODO())
}

func (ac *accountCache) ReadOrderTrades(ch OrderTradesChan) {
	for range ch {
		// TODO проверять, что сделка пришла по текущему счёту (кэш дольше проживёт)
		ac.invalidateCache()
	}
}

func (ac *accountCache) GetPositions(ctx context.Context) (pp *alex.Positions, err error) {
	ac.locker.RLock()
	positions := ac.positions
	valid := ac.positionsRequestTime.Add(ac.positionsLiveTime).After(time.Now())
	ac.locker.RUnlock()

	if !valid {
		ac.locker.Lock()
		defer ac.locker.Unlock()

		if ac.positionsInvalidateTimer != nil {
			ac.positionsInvalidateTimer.Stop()
		}

		positions, err = ac.account.engine.GetPositions(ctx)
		if err != nil {
			l.DPanic("accountCache.GetPositions", zap.Error(err))
			return nil, err
		}
		ac.positions = positions
		ac.positionsRequestTime = time.Now()
		ac.positionsInvalidateTimer = time.AfterFunc(ac.positionsLiveTime, ac.invalidateCache)
	}
	return positions, nil
}

func (ac *accountCache) GetOrders(ctx context.Context) (oo []alex.Order, err error) {
	ac.locker.RLock()
	orders := ac.orders
	valid := ac.ordersRequestTime.Add(ac.ordersLiveTime).After(time.Now())
	ac.locker.RUnlock()

	if !valid {
		ac.locker.Lock()
		defer ac.locker.Unlock()

		if ac.ordersInvalidateTimer != nil {
			ac.ordersInvalidateTimer.Stop()
		}

		orders, err = ac.account.engine.GetOrders(ctx)
		if err != nil {
			l.DPanic("accountCache.GetOrders", zap.Error(err))
			return nil, err
		}
		ac.orders = orders
		ac.ordersRequestTime = time.Now()
		ac.ordersInvalidateTimer = time.AfterFunc(ac.ordersLiveTime, ac.invalidateCache)
	}
	return orders, nil
}

func (ac *accountCache) GetBalance(ctx context.Context, i alex.Instrument) int64 {
	positions, err := ac.GetPositions(ctx)
	if err != nil {
		l.DPanic("accountCache.Get", zap.Error(err))
		return 0
	}
	position, ok := positions.Positions[i.GetFigi()]
	if !ok || position == nil {
		return 0
	}
	return position.GetBalance()
}

func (ac *accountCache) GetBlocked(ctx context.Context, i alex.Instrument) int64 {
	positions, err := ac.GetPositions(ctx)
	if err != nil {
		l.DPanic("accountCache.Get", zap.Error(err))
		return 0
	}
	position, ok := positions.Positions[i.GetFigi()]
	if !ok || position == nil {
		return 0
	}
	return position.GetBlocked()
}
