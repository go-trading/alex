package tinkoff

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/sdcoffey/big"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ alex.TargetPosition = (*TargetPosition)(nil)

type TargetPosition struct {
	bot               alex.Bot
	quantityLots      int64
	priceIncrement    big.Decimal
	errs              error
	stabilizationTime time.Time
}

func (tpi *TargetPosition) Error() string {
	if tpi.errs == nil {
		return ""
	}
	return tpi.errs.Error()
}

func (tpi *TargetPosition) GetError() error {
	savedErr := tpi.errs
	tpi.errs = nil
	return savedErr
}

func (tpi *TargetPosition) IsLimitError() bool {
	return IsLimitError(tpi)
}

func (tp *TargetPosition) saveError(err error) {
	tp.errs = multierror.Append(tp.errs, err)
	if tp.IsLimitError() {
		l.DPanic("Ошибка превышения лимитов. Останавливаю робота", zap.Error(err))
		_ = tp.bot.Stop()
	}
}

func (tp *TargetPosition) setStabilizationTime() {
	tp.stabilizationTime = time.Now().Add(1 * time.Second)
}

func (tp *TargetPosition) waitStabilization() bool {
	return tp.stabilizationTime.After(time.Now())
}

type TargetPositions struct {
	locker sync.Mutex
	target map[string]*TargetPosition
}

func NewTargetPositions() *TargetPositions {
	return &TargetPositions{
		target: make(map[string]*TargetPosition),
	}
}

// Функция запускает процесс приведения позиции по инструменту к указанной
// ОГРАНИЧЕНИЯ - только один робот на инструмент, иначе они будут
// конфликтовать между собой по позициям
func (a *AccountAbstract) DoPosition(ctx context.Context, bot alex.Bot, instrument alex.Instrument, targetPositionLots int64) alex.TargetPosition {
	return a.DoPositionExtended(ctx, bot, instrument, targetPositionLots, instrument.GetMinPriceIncrement())
}

func (a *AccountAbstract) DoPositionExtended(ctx context.Context, bot alex.Bot, instrument alex.Instrument, targetPositionLots int64, priceIncrement big.Decimal) alex.TargetPosition {
	a.targetPositions.locker.Lock()

	// устанавливаю метрику tinkoff_target_position
	targetPositionMetric.WithLabelValues(instrument.GetFigi(), bot.Name()).Set(float64(targetPositionLots))

	tp, ok := a.targetPositions.target[instrument.GetFigi()]
	if ok {
		if tp.bot != bot && a.targetPositions.target[instrument.GetFigi()].quantityLots != 0 {
			l.DPanic("На одном и том же счёте, с одной бумагой, работают разные роботы",
				zap.String("bot1", tp.bot.Name()),
				zap.String("bot2", bot.Name()),
				zap.String("account", a.id),
			)
			tp.errs = multierror.Append(tp.errs, errors.New("DIFFRENT BOT ON ONE ACCOUNT"))
			a.targetPositions.locker.Unlock()
			return tp
		}
		if tp.quantityLots == targetPositionLots && tp.priceIncrement == priceIncrement {
			a.targetPositions.locker.Unlock()
			return tp
		}
		tp.quantityLots = targetPositionLots
		tp.priceIncrement = priceIncrement
	} else {
		tp = &TargetPosition{
			bot:            bot,
			quantityLots:   targetPositionLots,
			priceIncrement: priceIncrement,
		}
		a.targetPositions.target[instrument.GetFigi()] = tp
	}
	a.targetPositions.locker.Unlock()

	a.doTracking(ctx) // повторные вызовы doTracking происходят при инвалидации кеша балансов/ордеров
	return tp
}

func (a *AccountAbstract) doTracking(ctx context.Context) {
	//скорее всего пришёл из invalidateCache, которую сам же и вызвал. Нечего страшного если пропущу один вызов
	if !a.targetPositions.locker.TryLock() {
		return
	}
	defer a.targetPositions.locker.Unlock()

	positions, err := a.GetPositions(ctx)
	if err != nil {
		l.DPanic("не удалось получить текущие позиции", zap.Error(err))
	}

	orders, err := a.GetOrders(ctx)
	if err != nil {
		l.DPanic("не удалось получить текущие позиции", zap.Error(err))
	}
	lotsInOrders := CalcLotsInOrders(orders)
	l.Debug("AccountAbstractImpl.doTracking",
		zap.Any("positions", positions.Positions),
		zap.Any("lotsInOrders", lotsInOrders))

	for figi, targetPosition := range a.targetPositions.target {
		if targetPosition.waitStabilization() {
			continue
		}

		positionLots := int64(0)
		positionFromAPI, ok := positions.Positions[figi]
		instrument := a.GetClient().GetInstrument(figi)

		if ok && positionFromAPI != nil {
			positionLots = (positionFromAPI.GetBalance() + positionFromAPI.GetBlocked()) / int64(instrument.GetLot())
		}
		if lotsInOrders[figi] == 0 &&
			positionLots == targetPosition.quantityLots {
			//с данным инструментом всё нормально, переходим к следующему
			//TODO если со всеми инструментами всё нормально, то можно реже заходить в doTracking
			continue
		}

		if positionLots+lotsInOrders[figi] != targetPosition.quantityLots {
			//отменить активные заявки
			for _, o := range orders {
				targetPosition.setStabilizationTime()
				_, err = o.Cancel(targetPosition.bot.Context())
				if err != nil {
					targetPosition.errs = multierror.Append(targetPosition.errs, err)
					l.Debug("не смог отменить заявку", zap.Error(err))
				}
			}
			//выставить новую заявку на недостающее количество
			targetPosition.setStabilizationTime()

			if !targetPosition.IsLimitError() && instrument.IsLimitOrderAvailable() {
				_, err = a.PostOrderWithBestPrice(
					targetPosition.bot.Context(),
					instrument,
					targetPosition.quantityLots-positionLots,
					targetPosition.priceIncrement,
				)
				if err != nil {
					targetPosition.saveError(err)
					l.Error("не смог отправить заявку", zap.String("figi", figi), zap.Error(err))
				}
			}
		} else {
			//если с количеством всё впорядке, то надо проверить, а не застоялась ли заявка
			for _, o := range orders {
				if o.IsActive() {
					if o.GetOrderDate().Add(time.Minute).Before(time.Now()) {
						if !o.IsBestInOrderBook(targetPosition.bot.Context()) {
							targetPosition.setStabilizationTime()
							_, err = o.Cancel(targetPosition.bot.Context())
							if err != nil {
								targetPosition.errs = multierror.Append(targetPosition.errs, err)
								l.Debug("не смог отменить долго висящую заявку", zap.Error(err))
							}
						}
					}
				}
			}
		}
	}
}

func (a *AccountAbstract) PostOrderWithBestPrice(ctx context.Context, instrument alex.Instrument, quantity int64, priceIncrement big.Decimal) (alex.Order, error) {
	if quantity == 0 {
		l.Debug("PostOrderWithBestPrice quantity == 0")
		return nil, nil
	}

	ob, err := instrument.GetOrderBook(ctx, 1)
	if err != nil {
		l.DPanic("GetOrderBook", zap.Error(err))
		return nil, nil
	}

	bestBid := big.NaN
	if len(ob.Bids) > 0 {
		bestBid = ob.Bids[0].Price
	}
	bestAsk := big.NaN
	if len(ob.Asks) > 0 {
		bestAsk = ob.Asks[0].Price
	}
	//if quantity > 0, для <0 будет переопределено ниже
	direction := proto.OrderDirection_ORDER_DIRECTION_BUY
	price := bestBid.Add(priceIncrement)
	if quantity < 0 {
		quantity = -quantity
		direction = proto.OrderDirection_ORDER_DIRECTION_SELL
		price = bestAsk.Sub(priceIncrement)
	}

	if price == big.NaN {
		l.Warn("лучшая цена не определена (стакан пустой?)")
		return nil, errors.New("лучшая цена не определена (стакан пустой?)")
	}

	return a.engine.PostOrder(ctx,
		instrument,
		quantity,
		price,
		direction,
		proto.OrderType_ORDER_TYPE_LIMIT,
		uuid.New().String(),
	)
}

func CalcLotsInOrders(orders []alex.Order) map[string]int64 {
	lotsInOrders := make(map[string]int64)
	for _, o := range orders {
		if o.IsActive() {
			if o.GetDirection() == proto.OrderDirection_ORDER_DIRECTION_BUY {
				lotsInOrders[o.GetFigi()] += o.GetLotsRequested() - o.GetLotsExecuted()
			} else {
				lotsInOrders[o.GetFigi()] -= o.GetLotsRequested() - o.GetLotsExecuted()
			}
		}
	}
	return lotsInOrders
}

//TODO надо вводить свой класс ошибок
func IsLimitError(e error) bool {
	switch err := e.(type) {
	case *TargetPosition:
		return IsLimitError(err.errs)
	case *multierror.Error:
		for _, subError := range err.Errors {
			if IsLimitError(subError) {
				return true
			}
		}
		return false
	default:
		if se, ok := e.(interface {
			GRPCStatus() *status.Status
		}); ok {
			l.Error("se.GRPCStatus()",
				zap.Any("GRPCStatus", se.GRPCStatus()),
				zap.String("Code", se.GRPCStatus().Code().String()),
				zap.Any("Message", se.GRPCStatus().Message()),
			)
			return se.GRPCStatus().Code() == codes.InvalidArgument &&
				se.GRPCStatus().Message() == "30042"
		}
		return false
	}
}
