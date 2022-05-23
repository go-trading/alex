package tinkoff

import (
	"sync"
	"time"

	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrderTradesChan chan *proto.OrderTrades

type OrderTrades struct {
	client             *Client
	locker             sync.RWMutex
	subscribers        []OrderTradesChan
	accounts           []string
	ordersStreamClient proto.OrdersStreamService_TradesStreamClient
}

func NewOrderTrades(c *Client) *OrderTrades {
	return &OrderTrades{
		client: c,
	}
}

func (ot *OrderTrades) SetAccounts(accounts []string) {
	ot.locker.Lock()
	defer ot.locker.Unlock()

	ot.accounts = accounts
}

func (ot *OrderTrades) open() (err error) {
	ot.locker.Lock()
	defer ot.locker.Unlock()

	l.Debug("openOrderTradesStream")
	ot.ordersStreamClient, err = ot.client.GetOrdersStreamServiceClient().TradesStream(
		ot.client.ctx,
		&proto.TradesStreamRequest{
			Accounts: ot.accounts,
		},
	)
	if err != nil {
		l.DPanic("не удалось подписаться на сделки", zap.Error(err))
		return err
	}
	go ot.streamReader()
	return nil
}

func (ot *OrderTrades) reconnect() {
	sleepTime := time.Second
	for {
		time.Sleep(sleepTime)
		sleepTime = 2 * sleepTime
		err := ot.open()
		if err != nil {
			l.Error("OrderTrades reconnect open", zap.Error(err))
			continue
		}
		//TODO BUG при переподключении надо отправлсять запрос на переподписку
		//но только если были запрошены боевые аккаунты, чтобы не словить лимит на количество потоков подписки на сделки
		break
	}
}

func (ot *OrderTrades) Subscribe() (OrderTradesChan, error) {
	ot.locker.Lock()
	defer ot.locker.Unlock()

	ch := make(OrderTradesChan, 10)
	ot.subscribers = append(ot.subscribers, ch)
	return ch, nil
}

func (ot *OrderTrades) Unsubscribe(ch OrderTradesChan) (err error) {
	ot.locker.Lock()
	defer ot.locker.Unlock()

	unsubscribeWithoutSubscription := true
	//удаляю подписчика
	for i, c := range ot.subscribers {
		if ch == c {
			ot.subscribers = append(ot.subscribers[:i], ot.subscribers[i+1:]...)
			unsubscribeWithoutSubscription = false
			break
		}
	}
	if unsubscribeWithoutSubscription {
		l.DPanic("отписываюсь от сделок, хотя не подписывался на них")
	}
	return nil
}

func (ot *OrderTrades) streamReader() {
	for {
		recv, err := ot.ordersStreamClient.Recv()
		l.Debug("orderTradeStreamClient.Recv()", zap.Any("orderTrades", recv))
		if err != nil {
			if status.Code(err) == codes.Canceled {
				l.Debug("streamReader - закрыто соединения")
			} else if status.Code(err) == codes.ResourceExhausted {
				l.DPanic("Превышены доступные ресурсы подключения.")
				//TODO     ot.client.Stop()
			} else {
				l.Error("streamReader получена ошибка", zap.Error(err))
				//переподключение
				ot.reconnect()
			}
			return
		}
		orderTrades := recv.GetOrderTrades()
		if orderTrades != nil {
			ot.locker.RLock()
			for _, sub := range ot.subscribers {
				if len(sub) == cap(sub) {
					l.Error("переполнен поток обработки сделок, медленная работа робота?")
				} else {
					sub <- orderTrades
				}
			}
			ot.locker.RUnlock()
		}
	}
}
