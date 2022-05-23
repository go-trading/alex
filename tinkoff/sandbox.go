package tinkoff

import (
	"context"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"github.com/sdcoffey/big"
	"go.uber.org/zap"
)

func (c *Client) OpenSandboxAccount(ctx context.Context) (string, error) {
	sandboxAccount, err := c.GetSandboxServiceClient().OpenSandboxAccount(ctx,
		&proto.OpenSandboxAccountRequest{})
	if err != nil {
		l.DPanic("OpenSandboxAccount", zap.Error(err))
		return "", err
	}

	return sandboxAccount.AccountId, nil
}

func (c *Client) CloseSandboxAccount(ctx context.Context, accountId string) error {
	_, err := c.GetSandboxServiceClient().CloseSandboxAccount(ctx,
		&proto.CloseSandboxAccountRequest{AccountId: accountId})
	if err != nil {
		l.DPanic("CloseSandboxAccount", zap.Error(err))
		return err
	}
	return nil
}

func (c *Client) SandboxPayIn(ctx context.Context, accountId string, rub big.Decimal) (*alex.Money, error) {
	res, err := c.GetSandboxServiceClient().SandboxPayIn(ctx,
		&proto.SandboxPayInRequest{
			AccountId: accountId,
			Amount: alex.NewMoneyValue(&alex.Money{
				Currency: "RUB",
				Value:    rub,
			}),
		})
	if err != nil {
		l.DPanic("SandboxPayIn", zap.Error(err))
		return nil, err
	}
	return alex.NewMoney(res.Balance), nil
}
