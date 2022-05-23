package tinkoff

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
)

var _ alex.Client = (*Client)(nil)

type Client struct {
	ctx                       context.Context
	endpoint                  string
	grpcOpts                  []grpc.DialOption
	conn                      *grpc.ClientConn
	dataDir                   string
	marketDataServiceClient   proto.MarketDataServiceClient
	usersServiceClient        proto.UsersServiceClient
	sandboxServiceClient      proto.SandboxServiceClient
	instrumentsServiceClient  proto.InstrumentsServiceClient
	operationsServiceClient   proto.OperationsServiceClient
	ordersStreamServiceClient proto.OrdersStreamServiceClient
	ordersServiceClient       proto.OrdersServiceClient
	//TODO add lock
	dataStreamMarket *MarketDataStream
	Instruments      *Instruments
	Accounts         *Accounts
	limit            *Limits
	orderTrades      *OrderTrades
}

func NewClient(endpoint string, token string, dataDir string) *Client {
	client := &Client{
		endpoint: endpoint,
		dataDir:  dataDir,
		limit:    &Limits{},
	}
	client.grpcOpts = []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithPerRPCCredentials(oauth.NewOauthAccess(&oauth2.Token{
			AccessToken: token,
		})),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			client.withAppName,
			client.limit.withLimit,
			grpc_prometheus.UnaryClientInterceptor,
		)),
	}
	client.Instruments = NewInstruments(client)
	client.Accounts = NewAccounts(client)
	client.orderTrades = NewOrderTrades(client)

	return client
}

func (c *Client) Open(ctx context.Context) (err error) {
	c.ctx = ctx
	c.conn, err = grpc.Dial(c.endpoint, c.grpcOpts...)
	if err != nil {
		return err
	}

	c.marketDataServiceClient = proto.NewMarketDataServiceClient(c.conn)
	c.usersServiceClient = proto.NewUsersServiceClient(c.conn)
	c.sandboxServiceClient = proto.NewSandboxServiceClient(c.conn)
	c.instrumentsServiceClient = proto.NewInstrumentsServiceClient(c.conn)
	c.operationsServiceClient = proto.NewOperationsServiceClient(c.conn)
	c.ordersStreamServiceClient = proto.NewOrdersStreamServiceClient(c.conn)
	c.ordersServiceClient = proto.NewOrdersServiceClient(c.conn)

	c.dataStreamMarket = NewMarketDataStream(c)

	err = c.limit.Load(ctx, c.conn)
	if err != nil {
		l.DPanic("limit.Load", zap.Error(err))
	}
	err = c.Instruments.LoadNew(ctx)
	if err != nil {
		l.DPanic("Instruments.LoadNew", zap.Error(err))
	}
	err = c.dataStreamMarket.open()
	if err != nil {
		l.DPanic("openMarketDataStream", zap.Error(err))
	}
	return err
}

func (c *Client) InitOrderStream(accounts []string) (err error) {
	if c.orderTrades.accounts != nil {
		//TODO сейчас просто подписываюсь на все доступные аккаунты, но надо научиться подписываться только на нужные
		return
	}
	c.orderTrades.SetAccounts(accounts)
	err = c.orderTrades.open()
	if err != nil {
		l.DPanic("c.OrderTrades.openOrderTradesStream", zap.Error(err))
	}
	return nil
}

func (c *Client) Close() error {
	l.Debug("закрываю соединение")
	c.marketDataServiceClient = nil
	c.usersServiceClient = nil
	c.sandboxServiceClient = nil
	c.instrumentsServiceClient = nil
	c.ordersStreamServiceClient = nil
	c.ordersServiceClient = nil
	return c.conn.Close()
}

func (c *Client) Etfs(ctx context.Context, status proto.InstrumentStatus) ([]*proto.Etf, error) {
	l.Debug("запрашиваю все etfs")
	etfsResponse, err := c.instrumentsServiceClient.Etfs(ctx, &proto.InstrumentsRequest{
		InstrumentStatus: status,
	})
	if err != nil {
		return nil, err
	}
	return etfsResponse.Instruments, nil
}

func (c *Client) Shares(ctx context.Context, status proto.InstrumentStatus) ([]*proto.Share, error) {
	l.Debug("запрашиваю все shares")
	sharesResponse, err := c.instrumentsServiceClient.Shares(ctx, &proto.InstrumentsRequest{
		InstrumentStatus: status,
	})
	if err != nil {
		return nil, err
	}
	return sharesResponse.Instruments, nil
}

func (c *Client) withAppName(ctx context.Context,
	method string,
	req interface{},
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	appName := metadata.AppendToOutgoingContext(ctx, "x-app-name", "github.com/go-trading/alex")
	return invoker(appName, method, req, reply, cc, opts...)
}

func (c *Client) GetOrdersServiceClient() proto.OrdersServiceClient {
	return c.ordersServiceClient
}
func (c *Client) GetSandboxServiceClient() proto.SandboxServiceClient {
	return c.sandboxServiceClient
}
func (c *Client) GetOperationsServiceClient() proto.OperationsServiceClient {
	return c.operationsServiceClient
}
func (c *Client) GetUsersServiceClient() proto.UsersServiceClient {
	return c.usersServiceClient
}
func (c *Client) GetInstrument(figi string) alex.Instrument {
	return c.Instruments.Get(figi)
}
func (c *Client) GetDataDir() string {
	return c.dataDir
}
func (c *Client) GetOrdersStreamServiceClient() proto.OrdersStreamServiceClient {
	return c.ordersStreamServiceClient
}
func (c *Client) GetMarketDataServiceClient() proto.MarketDataServiceClient {
	return c.marketDataServiceClient
}
func (c *Client) SubscribeCandles(cs *Candles) alex.CandleChan {
	return c.dataStreamMarket.Subscribe(cs)
}
func (c *Client) Now() time.Time {
	return time.Now()
}
func (c *Client) GetAccounts(ctx context.Context, engine alex.EngineType) (map[string]alex.Account, error) {
	switch engine {
	case alex.EngineType_REAL:
		return c.Accounts.GetRealAccounts(ctx)
	case alex.EngineType_SANDBOX:
		return c.Accounts.GetSandboxAccounts(ctx)
	default:
		return nil, errors.New("UNSUPPORTED ENGINE TYPE")
	}
}

func (c *Client) SaveCandles(candles alex.Candles) error {
	tinkoffCandles, ok := candles.(*Candles)
	if ok {
		return tinkoffCandles.Save()
	}
	return errors.New("UNSAPPORT CANDLES TYPE")
}

//Перенаправляем вывод робота в лог, в режиме Info + в консоль
func (c *Client) Printf(format string, arg ...any) (n int, err error) {
	msg := fmt.Sprintf(format, arg...)
	l.Info(msg)
	return 0, nil
}
