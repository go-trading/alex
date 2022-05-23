package tinkoff

import (
	"context"
	"sync"
	"time"

	"github.com/sdcoffey/big"
	"go.uber.org/zap"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
)

var _ alex.Instrument = (*Instrument)(nil)

type InstrumentGeneralDescription interface {
	GetFigi() string
	GetExchange() string
	GetClassCode() string
	GetIsin() string
	GetTicker() string
	GetCurrency() string
	GetName() string
	GetLot() int32
}

type InstrumentAdditionDescriptionInAPI interface {
	InstrumentGeneralDescription
	GetMinPriceIncrement() *proto.Quotation
}

type Instrument struct {
	client                     *Client
	allCandlesOfInstrument     map[time.Duration]*Candles
	allCandlesOfInstrumentLock sync.RWMutex
	InstrumentDescriptionLink  InstrumentAdditionDescriptionInAPI
	OrderBookCache             OrderBookCache
	tradingStatus              proto.SecurityTradingStatus
	limitOrderAvailable        bool
	marketOrderAvailable       bool
}

func NewInstrument(client *Client, instDesc InstrumentAdditionDescriptionInAPI) *Instrument {
	return &Instrument{
		client:                 client,
		allCandlesOfInstrument: make(map[time.Duration]*Candles),
		OrderBookCache: OrderBookCache{
			LiveTime:   5 * time.Second,
			OrderBooks: make(map[int32]OrderBookCacheItem),
		},
		InstrumentDescriptionLink: instDesc,
	}
}

func (i *Instrument) GetClient() *Client   { return i.client }
func (i *Instrument) GetFigi() string      { return i.InstrumentDescriptionLink.GetFigi() }
func (i *Instrument) GetExchange() string  { return i.InstrumentDescriptionLink.GetExchange() }
func (i *Instrument) GetClassCode() string { return i.InstrumentDescriptionLink.GetClassCode() }
func (i *Instrument) GetIsin() string      { return i.InstrumentDescriptionLink.GetIsin() }
func (i *Instrument) GetTicker() string    { return i.InstrumentDescriptionLink.GetTicker() }
func (i *Instrument) GetCurrency() string  { return i.InstrumentDescriptionLink.GetCurrency() }
func (i *Instrument) GetName() string      { return i.InstrumentDescriptionLink.GetName() }
func (i *Instrument) GetLot() int32        { return i.InstrumentDescriptionLink.GetLot() }

func (i *Instrument) GetMinPriceIncrement() big.Decimal {
	return alex.NewDecimal(i.InstrumentDescriptionLink.GetMinPriceIncrement())
}
func (i *Instrument) Now() time.Time { return time.Now() }

func (i *Instrument) SetStatus(tradingStatus proto.SecurityTradingStatus) {
	i.allCandlesOfInstrumentLock.Lock()
	defer i.allCandlesOfInstrumentLock.Unlock()
	i.tradingStatus = tradingStatus
}

func (i *Instrument) IsStatus(tradingStatus ...proto.SecurityTradingStatus) bool {
	i.allCandlesOfInstrumentLock.RLock()
	defer i.allCandlesOfInstrumentLock.RUnlock()
	for _, s := range tradingStatus {
		if i.tradingStatus == s {
			return true
		}
	}
	return false
}

func (i *Instrument) SetOrderAvailable(limitOrderAvailable bool, marketOrderAvailable bool) {
	i.allCandlesOfInstrumentLock.Lock()
	defer i.allCandlesOfInstrumentLock.Unlock()
	i.limitOrderAvailable = limitOrderAvailable
	i.marketOrderAvailable = marketOrderAvailable
}
func (i *Instrument) IsLimitOrderAvailable() bool {
	i.allCandlesOfInstrumentLock.RLock()
	defer i.allCandlesOfInstrumentLock.RUnlock()
	return i.limitOrderAvailable
}
func (i *Instrument) IsMarketOrderAvailable() bool {
	i.allCandlesOfInstrumentLock.RLock()
	defer i.allCandlesOfInstrumentLock.RUnlock()
	return i.marketOrderAvailable
}

// Получить объект свечей уникальный в рамках соединения к АПИ
func (i *Instrument) GetCandles(period time.Duration) alex.Candles {
	if period == 0 {
		l.DPanic("period не определён")
		return nil
	}
	i.allCandlesOfInstrumentLock.RLock()
	candles, ok := i.allCandlesOfInstrument[period]
	i.allCandlesOfInstrumentLock.RUnlock()
	if ok {
		return candles
	}

	l.Debug("свечи по инструменту не найдены, создаю новые")
	i.allCandlesOfInstrumentLock.Lock()
	defer i.allCandlesOfInstrumentLock.Unlock()

	candles, ok = i.allCandlesOfInstrument[period] // другой поток мог успеть записать
	if ok {
		return candles
	}
	candles = NewCandles(i.client, i.GetFigi(), period)
	i.allCandlesOfInstrument[period] = candles
	return candles
}

type Instruments struct {
	client      *Client
	locker      sync.RWMutex
	instruments map[string]*Instrument
}

func NewInstruments(client *Client) *Instruments {
	return &Instruments{
		instruments: make(map[string]*Instrument),
		client:      client,
	}
}

func (ii *Instruments) LoadNew(ctx context.Context) error {
	ii.locker.Lock()
	defer ii.locker.Unlock()

	etfs, err := ii.client.Etfs(ctx, proto.InstrumentStatus_INSTRUMENT_STATUS_ALL)
	if err != nil {
		l.DPanic("InitLimits", zap.Error(err))
	}
	for _, etf := range etfs {
		_, ok := ii.instruments[etf.Figi]
		if !ok {
			ii.instruments[etf.Figi] = NewInstrument(ii.client, etf)
		}
	}

	shares, err := ii.client.Shares(ctx, proto.InstrumentStatus_INSTRUMENT_STATUS_ALL)
	if err != nil {
		l.DPanic("InitLimits", zap.Error(err))
	}
	for _, s := range shares {
		_, ok := ii.instruments[s.Figi]
		if !ok {
			ii.instruments[s.Figi] = NewInstrument(ii.client, s)
		}
	}
	return nil
}

func (ii *Instruments) Get(figi string) *Instrument {
	ii.locker.RLock()
	defer ii.locker.RUnlock()
	i, ok := ii.instruments[figi]
	if !ok {
		l.Warn("Не найден запрошенный инструмент", zap.String("figi", figi))
	}
	return i
}
