package history

import (
	"time"

	"github.com/go-trading/alex"
	"go.uber.org/zap"
)

var _ alex.Client = (*Client)(nil)

type Client struct {
	dataDir     string
	from        time.Time
	to          time.Time
	now         time.Time
	instruments map[string]*instrument
	accounts    map[string]*account
}

func NewClient(dataDir string, from time.Time, to time.Time) *Client {
	return &Client{
		dataDir:     dataDir,
		from:        from,
		to:          to,
		now:         from,
		instruments: make(map[string]*instrument),
		accounts:    make(map[string]*account),
	}
}

func (c *Client) Now() time.Time { return c.now }

func (c *Client) GetInstrument(figi string) alex.Instrument {
	i, ok := c.instruments[figi]
	if !ok {
		l.DPanic("запрошен инструмент, данные по которому не скачивались")
		return nil
	}
	return i
}

func (c *Client) LoadData(figi string) (err error) {
	i := newInstrument(c, figi)
	err = i.load()
	if err != nil {
		return err
	}
	c.instruments[figi] = i
	return nil
}

func (c *Client) CreateAccount(name string) alex.Account {
	_, ok := c.accounts[name]
	if ok {
		l.DPanic("счёт с таким именем уже существует")
		return nil
	}
	c.accounts[name] = newAccount(c, name)
	return c.accounts[name]
}

func (c *Client) Printf(format string, arg ...any) (n int, err error) {
	//return fmt.Printf(format, arg...)
	return 0, nil
}

//Пробегаюсь по истории, для каждой свечи передаю в instrument.Tick 4 события:
//с ценой открытия, hi и low (меняя каждый шаг порядой, в котором передаю hi и low), и close
func (c *Client) Run() error {
	for c.now = c.from.Add(time.Second); c.now.Before(c.to); {
		// OPEN
		for figi, instrument := range c.instruments {
			idx := alex.FindSeries(instrument.FUTURE, c.now)
			if idx == -1 {
				continue
			}
			candle := instrument.FUTURE.Candles[idx]
			l.Debug("отправляю цену открытия свечи",
				zap.Time("c.now", c.now),
				zap.Time("candle.Period.Start", candle.Period.Start),
				zap.String("Price", candle.OpenPrice.FormattedString(2)),
			)
			instrument.Tick(&alex.LastPrice{
				Figi:  figi,
				Price: candle.OpenPrice,
				Time:  c.now,
			})
		}
		// HI
		c.now = c.now.Add(time.Second)
		for figi, instrument := range c.instruments {
			idx := alex.FindSeries(instrument.FUTURE, c.now)
			if idx == -1 {
				continue
			}
			candle := instrument.FUTURE.Candles[idx]
			if candle.OpenPrice.LT(candle.MaxPrice) {
				l.Debug("отправляю макс цену свечи",
					zap.Time("c.now", c.now),
					zap.Time("candle.Period.Start", candle.Period.Start),
					zap.String("Price", candle.MaxPrice.FormattedString(2)),
				)
				instrument.Tick(&alex.LastPrice{
					Figi:  figi,
					Price: candle.MaxPrice,
					Time:  c.now.Add(2 * time.Microsecond),
				})
			}
		}
		//LOW
		c.now = c.now.Add(time.Second)
		for figi, instrument := range c.instruments {
			idx := alex.FindSeries(instrument.FUTURE, c.now)
			if idx == -1 {
				continue
			}
			candle := instrument.FUTURE.Candles[idx]
			if candle.OpenPrice.GT(candle.MinPrice) {
				l.Debug("отправляю мин цену свечи",
					zap.Time("c.now", c.now),
					zap.Time("candle.Period.Start", candle.Period.Start),
					zap.String("Price", candle.MinPrice.FormattedString(2)),
				)
				instrument.Tick(&alex.LastPrice{
					Figi:  figi,
					Price: candle.MinPrice,
					Time:  c.now.Add(3 * time.Microsecond),
				})
			}
		}
		//CLOSE
		c.now = c.now.Add(58 * time.Second)
		for figi, instrument := range c.instruments {
			idx := alex.FindSeries(instrument.FUTURE, c.now.Add(-time.Minute))
			if idx == -1 {
				continue
			}
			candle := instrument.FUTURE.Candles[idx]
			if candle.ClosePrice.GT(candle.MinPrice) && candle.ClosePrice.LT(candle.MaxPrice) {
				l.Debug("отправляю цену закрытия свечи",
					zap.Time("c.now", c.now),
					zap.Time("candle.Period.Start", candle.Period.Start),
					zap.String("Price", candle.ClosePrice.FormattedString(2)),
				)
				instrument.Tick(&alex.LastPrice{
					Figi:  figi,
					Price: candle.ClosePrice,
					Time:  c.now,
				})
			}
		}
	}
	return nil
}

func (c *Client) PrintResult() {
	for _, a := range c.accounts {
		a.PrintResult()
	}
}
