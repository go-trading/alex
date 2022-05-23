package tinkoff

import (
	"context"
	"sync"

	"github.com/go-trading/alex"
	proto "github.com/go-trading/alex/tinkoff/proto/1.0.7"
	"go.uber.org/zap"
)

type Accounts struct {
	client          *Client
	locker          sync.RWMutex
	realAccounts    map[string]alex.Account
	sandboxAccounts map[string]alex.Account
}

func NewAccounts(client *Client) *Accounts {
	return &Accounts{
		realAccounts:    make(map[string]alex.Account),
		sandboxAccounts: make(map[string]alex.Account),
		client:          client,
	}
}

func (aa *Accounts) GetRealAccounts(ctx context.Context) (map[string]alex.Account, error) {
	aa.locker.Lock()
	defer aa.locker.Unlock()

	accountsResponceReal, err := aa.client.GetUsersServiceClient().GetAccounts(ctx, &proto.GetAccountsRequest{})
	if err != nil {
		l.DPanic("UsersService/GetAccounts", zap.Error(err))
		return nil, err

	}

	for _, respA := range accountsResponceReal.Accounts {
		_, ok := aa.realAccounts[respA.GetId()]
		if !ok {
			//TODO надо обновлять атрибуты существующих аккаунтов, т.к. сеййчас просто добавляю новые
			aa.realAccounts[respA.GetId()] = NewRealAccount(ctx, aa.client, respA)
		}
	}

	err = aa.client.InitOrderStream(aa.GetRealAccountsStrings())
	if err != nil {
		l.DPanic("initOrderStream", zap.Error(err))
		return nil, err
	}

	return aa.realAccounts, nil
}

func (aa *Accounts) GetRealAccountsStrings() (result []string) {
	for id := range aa.realAccounts {
		result = append(result, id)
	}
	return result
}

func (aa *Accounts) GetSandboxAccounts(ctx context.Context) (map[string]alex.Account, error) {
	aa.locker.Lock()
	defer aa.locker.Unlock()

	accountsResponceSandbox, err := aa.client.GetSandboxServiceClient().GetSandboxAccounts(ctx, &proto.GetAccountsRequest{})
	if err != nil {
		l.DPanic("SandboxService/GetSandboxAccounts", zap.Error(err))
		return nil, err
	}

	for _, respA := range accountsResponceSandbox.Accounts {
		_, ok := aa.sandboxAccounts[respA.GetId()]
		if !ok {
			//TODO надо обновлять атрибуты существующих аккаунтов, т.к. сеййчас просто добавляю новые
			aa.sandboxAccounts[respA.GetId()] = NewSandboxAccount(aa.client, respA)
		}
	}
	return aa.sandboxAccounts, nil
}

func (aa *Accounts) tryGet(accountId string) (alex.Account, bool) {
	a, ok := aa.realAccounts[accountId]
	if ok {
		return a, ok
	}
	a, ok = aa.sandboxAccounts[accountId]
	if ok {
		return a, ok
	}
	return nil, false
}

func (aa *Accounts) GetOrDie(ctx context.Context, accountId string) alex.Account {
	aa.locker.RLock()
	a, ok := aa.tryGet(accountId)
	aa.locker.RUnlock()
	if ok {
		return a
	}

	_, err := aa.GetSandboxAccounts(ctx)
	if err != nil {
		l.DPanic("не смог получить счета песочницы", zap.Error(err))
	}

	aa.locker.RLock()
	a, ok = aa.tryGet(accountId)
	aa.locker.RUnlock()
	if ok {
		return a
	}

	_, err = aa.GetRealAccounts(ctx)
	if err != nil {
		l.DPanic("не смог получить боевые счета", zap.Error(err))
	}

	aa.locker.RLock()
	a, ok = aa.tryGet(accountId)
	aa.locker.RUnlock()
	if ok {
		return a
	}
	l.Fatal("Account не найден", zap.String("accountId", accountId))
	return nil
}
