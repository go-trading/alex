package tinkoff

import "github.com/go-trading/alex"

func (cs *Candles) Subscribe() (candleChan alex.CandleChan, err error) {
	cs.subscribersLock.Lock()
	defer cs.subscribersLock.Unlock()

	candleChan = make(alex.CandleChan, 50)
	cs.subscribers = append(cs.subscribers, candleChan)
	return candleChan, nil
}

func (cs *Candles) Unsubscribe(candleChan alex.CandleChan) (err error) {
	cs.subscribersLock.Lock()
	defer cs.subscribersLock.Unlock()

	//удаляю подписчика
	if cs.RemoveSubscriber(candleChan) {
		cs.l.DPanic("отписываюсь от свеч, хотя не подписывался на них")
	}

	// если подписчики ещё остались, то отписываться на сервере не надо
	if len(cs.subscribers) > 0 {
		return nil
	}
	return nil
	// если подписчиков не осталось, то можно отписаться от потока с сервера, но тогда надо думать, как синхронизировать, если через час снова подпишится
	//return cs.sendSubscribeRequest(proto.SubscriptionAction_SUBSCRIPTION_ACTION_UNSUBSCRIBE)
}

func (cs *Candles) RemoveSubscriber(candleChan alex.CandleChan) bool {
	for i, c := range cs.subscribers {
		if candleChan == c {
			cs.subscribers = append(cs.subscribers[:i], cs.subscribers[i+1:]...)
			close(candleChan)
			return true
		}
	}
	return false
}

func (cs *Candles) incomingCandleRecv() {
	for candle := range cs.incomingChannel {
		cs.Upsert(candle)
		cs.subscribersLock.RLock()
		for _, sub := range cs.subscribers {
			if len(sub) == cap(sub) {
				l.Error("переполнен поток обработки свечей. медленная работа робота? deadlock?")
			} else {
				sub <- candle
			}
		}
		cs.subscribersLock.RUnlock()
	}
}
