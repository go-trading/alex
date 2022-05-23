package alex

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

// интерфейс, который должны реализовывать роботы
type Bot interface {
	Start() error             // вызывается при старте робота. Отличное место чтобы подписатсья на рыночную информацию
	Stop() error              // вызывается при остановки робота. Отличное место чтобы отписаться от рыночной информации, закрыть позиции.
	Context() context.Context // контекст робота, именно от него происходит взаимодействие с tinkoff api
	Name() string             // имя робота, желательно читаемое и уникальное. Активно используется в качестве меток метрик
}

// тип для роботы с массивами роботов
type Bots []Bot

func (bs Bots) StartAll() error {
	for _, b := range bs {
		if err := b.Start(); err != nil {
			bs.StopAll() //nolint:golint,errcheck
			l.DPanic("не смог стартовать робота", zap.Error(err))
		}
	}
	return nil
}

func (bs Bots) StopAll() (result error) {
	for _, b := range bs {
		if err := b.Stop(); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}
