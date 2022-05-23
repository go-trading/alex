package tinkoff

import (
	"context"
	"strings"
	"time"

	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

type Limits struct {
	limits map[string]*rate.Limiter
}

func (limits *Limits) Load(ctx context.Context, conn grpc.ClientConnInterface) error {
	usersServiceClient := proto.NewUsersServiceClient(conn)
	userTariff, err := usersServiceClient.GetUserTariff(ctx, &proto.GetUserTariffRequest{})
	if err != nil {
		l.DPanic("GetUserTariff", zap.Error(err))
		return err
	}

	limits.limits = make(map[string]*rate.Limiter)
	for _, limit := range userTariff.UnaryLimits {
		rateLimit := rate.NewLimiter(rate.Every(time.Minute/time.Duration(limit.LimitPerMinute)), int(limit.LimitPerMinute))
		for _, metod := range limit.Methods {
			limits.limits["/"+metod] = rateLimit
		}
	}

	return nil
}

func (limits *Limits) withLimit(ctx context.Context,
	method string,
	req interface{},
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	l.Debug("call api", zap.String("method", strings.Replace(method, "/tinkoff.public.invest.api.contract.v1.", "", -1)))
	limit := limits.limits[method]
	if limit != nil {
		err := limit.Wait(ctx)
		if err != nil {
			l.Debug("Не смог дождаться ratelimit", zap.String("metod", method), zap.Error(err))
		}
	} else {
		if method != "/tinkoff.public.invest.api.contract.v1.UsersService/GetUserTariff" {
			l.DPanic("Лимит для метода не найден", zap.String("metod", method))
		}
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}
