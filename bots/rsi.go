package bots

// Робот, реализующий стратегию на основе индекса относительной силы (RSI)
// Покупает, если RSI ниже заданного уровня (рынок перепродан),
// и продаёт если RSI выше заданного уровня (рынок перекуплен)
// описание RSI: https://ru.wikipedia.org/wiki/%D0%98%D0%BD%D0%B4%D0%B5%D0%BA%D1%81_%D0%BE%D1%82%D0%BD%D0%BE%D1%81%D0%B8%D1%82%D0%B5%D0%BB%D1%8C%D0%BD%D0%BE%D0%B9_%D1%81%D0%B8%D0%BB%D1%8B

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
	"go.uber.org/multierr"
)

type RSIBot struct {
	name            string             // имя робота
	account         alex.Account       // счёт, на котором осуществляется торговля
	instrument      alex.Instrument    // инструмент, которым осуществляется торговля
	timeframe       int                // сколько последних свечей участвует в рассчёте RSI
	rsi4buy         int                // уровень, ниже которого покупаем
	rsi4sell        int                // уровень, выше которого продаём
	maxPosition     int64              // максимальное количество (в лотах), которым может торгавать робот
	candles         alex.Candles       // свечи на основании которых рассчитывается RSI
	bestPriceInc    big.Decimal        // насколько лучше лучшей цены, надо выставлять заявку. Для BBG000000001 выставляется в 0, для остальных инструментов равно минимальному шагу. см. метод Config
	ctx             context.Context    // контекст робота
	cancel          context.CancelFunc // функция отмены контекста робота
	candlesChan     alex.CandleChan    // канал, в который SDK присылает обновление по свечам
	lastCloseCandle int                // индекс последней свечи, которую использовал для торговли
}

// Конструктор нового робота
func NewRSIBot(ctx context.Context) *RSIBot {
	botCtx, cancel := context.WithCancel(ctx)
	return &RSIBot{
		ctx:    botCtx,
		cancel: cancel,
	}
}

// Конфигурация нового робота - вынесено из конструктора, т.к. можно изменять в конфигурацию в процессе
// потребуется в будущем, при разработке инструментов оптимизации параметров робота
func (b *RSIBot) Config(configs *alex.BotConfig) error {
	period := configs.GetDurationOrDie("candles-period")
	if alex.Duration2CandleInterval(period) == proto.CandleInterval_CANDLE_INTERVAL_UNSPECIFIED {
		return errors.New("INCORRECT CANDLES PERIOD")
	}
	b.name = configs.Name
	b.account = configs.Account
	b.instrument = configs.Instrument
	b.candles = configs.Instrument.GetCandles(period)
	//TODO при неправельной конфигурации возвращать ошибку, а не кидать исключение
	b.timeframe = configs.GetIntOrDie("timeframe")
	b.rsi4buy = configs.GetIntOrDie("rsi4buy")
	b.rsi4sell = configs.GetIntOrDie("rsi4sell")
	b.maxPosition = int64(configs.GetIntOrDie("max-position"))
	// для всех инструментов выставляю заявки по цене немного лучше чем лучшая в стакане
	// для BBG000000001 разница между покупкой и продажей всегда равна minIncriment, и заявка сразу исполняется
	//    и чтобы продемонстрировать работу с долгоживущими заявками на BBG000000001 буду ставить заявки по цене равной лучшей
	if b.instrument.GetFigi() != "BBG000000001" {
		b.bestPriceInc = b.instrument.GetMinPriceIncrement()
	}
	b.writeConfigMetrics()
	return nil
}

//Начать торговлю
func (b *RSIBot) Start() error {
	//если запустить робота в середине дня, он сразу скачает нужные ему свечи
	//в начале дня свечей ещё не будет, можно подумать над более хитрым механизмом инициализации, который бы позволил скачать свечи из предыдущего торгового дня
	// TODO реализовать _, err := b.candles.GetLast(b.ctx, b.timeframe)
	err := b.candles.Load(b.ctx,
		b.instrument.Now().Add(time.Duration(-int(b.candles.GetPeriod())*(b.timeframe+1))),
		b.instrument.Now())
	if err != nil {
		return err
	}
	b.candlesChan, err = b.candles.Subscribe()
	go b.botLoop()
	return err
}

//Остановить торговлю
func (b *RSIBot) Stop() error {
	b.cancel()
	err := b.account.DoPosition(b.ctx, b, b.instrument, 0)
	return multierr.Append(err, b.candles.Unsubscribe(b.candlesChan))
}

//Основной цикл, в котором получаю информацию о свечах, и передаю в функцию принятия торгового решения
func (b *RSIBot) botLoop() {
	for {
		select {
		case <-b.candlesChan:
			// интересует только последняя информация по свечам. Если в очереде есть необработанные свечи, перехожу срузу к ним
			if len(b.candlesChan) == 0 {
				timer := prometheus.NewTimer(botDurationMetric.WithLabelValues(b.name))
				b.OnCandle() // Именно в этом методе происходит вся магия
				// сохраняю метрику скорости обработки информации роботом
				timer.ObserveDuration()
			}
		case <-b.ctx.Done():
			b.account.GetClient().Printf("Завершаю обработку свечей роботом.\n")
			return
		}
	}
}

// метод обработки рыночной информации роботом
func (b *RSIBot) OnCandle() {
	// получаю свечи, и проверяю, что полученное валидно и появилась новая закрытая свеча
	series := b.candles.GetSeries()
	lastCloseCandle := series.LastIndex()
	if series.Candles[lastCloseCandle].Period.End.After(b.instrument.Now()) {
		lastCloseCandle--
	}
	if lastCloseCandle < 0 || lastCloseCandle == b.lastCloseCandle {
		return
	}
	b.lastCloseCandle = lastCloseCandle

	// рассчитываю значение индекса RSI
	rsi := techan.NewRelativeStrengthIndexIndicator(
		techan.NewClosePriceIndicator(series),
		b.timeframe,
	).Calculate(lastCloseCandle)
	// сохраняю значение индекса, для просмотра в grafana
	b.writeMetricsRSI(rsi)

	if rsi == big.ZERO {
		return
	}

	// сравниваю RSI с конфигурационными параметрами, и принимаю решение о желаемой позиции
	targetPosition := int64(math.MinInt64)
	hasMadeADecision := false
	if rsi.LTE(big.NewFromInt(b.rsi4buy)) {
		targetPosition = b.maxPosition
		hasMadeADecision = true
	} else if rsi.GTE(big.NewFromInt(b.rsi4sell)) {
		targetPosition = 0
		hasMadeADecision = true
	}

	// вывожу статус / отладочную информацию.
	// делаю это через b.account.Printf, т.к. куда пишиет робот, зависит от режима (на истории / на реальном счёте), и за это отвечает account
	b.account.GetClient().Printf("%s %s rsi=%s\ttargetPosition=%d\n",
		b.instrument.Now().Format(time.StampMilli),
		b.name,
		rsi.FormattedString(2),
		targetPosition)

	// если решение о позиции было принято, то сообщаю желаемую позицию в SDK, дальще SDK само выставит нужные заявки, и отследит их исполнение
	if hasMadeADecision {
		pos := b.account.DoPositionExtended(b.ctx, b, b.instrument, targetPosition, b.bestPriceInc)
		// если в процессе достежения заявки произошли какие нибудь ошибки, то обрабатываю их
		if pos != nil && pos.Error() != "" {
			// в случае превышение лимитов, останавливаю робота
			if pos.IsLimitError() {
				b.account.GetClient().Printf("Превышение по лимитам. Останавливаю робота. %v", pos.Error())
				_ = pos.GetError() //clear error
				_ = b.Stop()
			} else {
				// если ошибка не связана с лимитами (например, потерено соединеие), то SDK само попытает исправить ошибку и в роботе её можно игнорировать.
				b.account.GetClient().Printf("Ошибка при формировании позиции. %v", pos.Error())
				_ = pos.GetError() //clear error
			}
		}
	}
}

// гетеры, реализующие интерфейс alex.Bot
func (b *RSIBot) Name() string             { return b.name }
func (b *RSIBot) Context() context.Context { return b.ctx }
