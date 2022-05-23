package alex

import (
	"time"

	"go.uber.org/zap"
)

// настройки для робота
type BotConfig struct {
	Name       string         // имя робота
	Account    Account        // счёт, на котором робот будет торгавать
	Instrument Instrument     // инструмент, которым робот будет торгавать
	Values     map[string]any // произвольные параметры робота
}

func NewConfig(name string, account Account, instrument Instrument, values map[string]any) *BotConfig {
	if name == "" || account == nil || instrument == nil || values == nil {
		l.DPanic("Некорректная конфигурация робота")
		return nil
	}
	return &BotConfig{
		Name:       name,
		Account:    account,
		Instrument: instrument,
		Values:     values,
	}
}

func (c *BotConfig) GetAny(key string) any {
	v, ok := c.Values[key]
	if !ok {
		l.Error("config key not found",
			zap.String("name", c.Name),
			zap.String("key", key),
		)
		panic("config key not found")
	}
	return v
}

func (c *BotConfig) GetIntOrDie(key string) int {
	v := c.GetAny(key)

	typedValue, ok := v.(int)
	if !ok {
		l.Error("value not int", zap.String("name", c.Name), zap.String("key", key))
		panic("value not string")
	}
	return typedValue
}

func (c *BotConfig) GetStringOrDie(key string) string {
	v := c.GetAny(key)

	typedValue, ok := v.(string)
	if !ok {
		l.Error("value not string", zap.String("name", c.Name), zap.String("key", key))
		panic("value not string")
	}
	return typedValue
}

func (c *BotConfig) GetDurationOrDie(key string) time.Duration {
	v := c.GetAny(key)

	typedValue, ok := v.(time.Duration)
	if !ok {
		l.Error("value not time.Duration", zap.String("name", c.Name), zap.String("key", key))
		panic("value not time.Duration")
	}
	return typedValue
}
