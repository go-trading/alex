package alex

//Баланс позиции
type Position interface {
	GetFigi() string   //Figi-идентификатор.
	GetBlocked() int64 //Заблокировано.
	GetBalance() int64 //Текущий незаблокированный баланс.
}

//Портфель по счёту.
type Positions struct {
	Money                   []*Money            //Массив валютных позиций портфеля.
	Blocked                 []*Money            //Массив заблокированных валютных позиций портфеля.
	Positions               map[string]Position //Список позиций портфеля (обединяет ценно-бумажные и фьючерсы). Ключ figi.
	LimitsLoadingInProgress bool                //Признак идущей в данный момент выгрузки лимитов.
}
