package history

import "github.com/go-trading/alex"

var _ alex.Position = (*position)(nil)

type position struct {
	figi    string //Figi-идентификатор.
	balance int64  //Текущий незаблокированный баланс
	blocked int64  //Заблокировано.(в сделках на продажу)
	buy     int64  //В сделках на покупку
}

//Figi-идентификатор.
func (p *position) GetFigi() string { return p.figi }

//Заблокировано.
func (p *position) GetBlocked() int64 { return p.blocked }

//Текущий незаблокированный баланс.
func (p *position) GetBalance() int64 { return p.balance }
