package alex

// Помошники по трансформации интервалов в разные форматы

import (
	"time"

	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
)

var Interval2string = map[proto.CandleInterval]string{
	proto.CandleInterval_CANDLE_INTERVAL_1_MIN:  "1min",
	proto.CandleInterval_CANDLE_INTERVAL_5_MIN:  "5min",
	proto.CandleInterval_CANDLE_INTERVAL_15_MIN: "15min",
	proto.CandleInterval_CANDLE_INTERVAL_HOUR:   "hour",
	proto.CandleInterval_CANDLE_INTERVAL_DAY:    "day",
}

func Duration2CandleInterval(d time.Duration) proto.CandleInterval {
	switch d {
	case time.Minute:
		return proto.CandleInterval_CANDLE_INTERVAL_1_MIN
	case 5 * time.Minute:
		return proto.CandleInterval_CANDLE_INTERVAL_5_MIN
	case 15 * time.Minute:
		return proto.CandleInterval_CANDLE_INTERVAL_15_MIN
	case time.Hour:
		return proto.CandleInterval_CANDLE_INTERVAL_HOUR
	case 24 * time.Hour:
		return proto.CandleInterval_CANDLE_INTERVAL_DAY
	default:
		return proto.CandleInterval_CANDLE_INTERVAL_UNSPECIFIED
	}
}

func Duration2SubscriptionInterval(period time.Duration) proto.SubscriptionInterval {
	switch period {
	case time.Minute:
		return proto.SubscriptionInterval_SUBSCRIPTION_INTERVAL_ONE_MINUTE
	case time.Minute * 5:
		return proto.SubscriptionInterval_SUBSCRIPTION_INTERVAL_FIVE_MINUTES
	default:
		return proto.SubscriptionInterval_SUBSCRIPTION_INTERVAL_UNSPECIFIED
	}
}

func SubscriptionInterval2Duration(subscriptionInterval proto.SubscriptionInterval) time.Duration {
	switch subscriptionInterval {
	case proto.SubscriptionInterval_SUBSCRIPTION_INTERVAL_ONE_MINUTE:
		return time.Minute
	case proto.SubscriptionInterval_SUBSCRIPTION_INTERVAL_FIVE_MINUTES:
		return 5 * time.Minute
	default:
		return time.Duration(0)
	}
}
