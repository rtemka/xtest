package storage

import (
	"context"
	"xtestserver/domain"
)

type Filter struct {
	Operator string
	Currency string
	Limit    int
	Offset   int
	Time     int64
}

// Storage - контракт реализуемый базой данных.
type Storage interface {
	AddBtcRate(context.Context, domain.Rate) error   // Добавляет в БД текущий курс BTC/USD.
	AddFiats(context.Context, ...domain.Rate) error  // Добавляет в БД текущий курс фиатных валют.
	RUBUSDRate(context.Context) (domain.Rate, error) // Возвращает текущий курс доллара к рублю.
	Close() error                                    // закрываем соединение с БД.
	// Возвращает отфильтрованное кол-во из таблицы курса BTC/USD.
	BtcRate(ctx context.Context, filter Filter) ([]domain.Rate, error)
	// Возвращает отфильтрованное кол-во из таблицы курса фиатных валют.
	Fiats(ctx context.Context, filter Filter) ([]domain.Rate, error)
	// Возвращает текущий курс к рублю из таблицы курсов фиатных валют.
	FiatsCurrent(context.Context) ([]domain.Rate, error)
}
