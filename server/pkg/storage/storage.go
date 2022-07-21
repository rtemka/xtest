package storage

import (
	"context"
	"xtestserver/domain"
)

// Storage - контракт реализуемый базой данных.
type Storage interface {
	AddBtcRate(context.Context, domain.Rate) error   // Добавляет в БД текущий курс BTC/USD.
	AddFiats(context.Context, ...domain.Rate) error  // Добавляет в БД текущий курс фиатных валют.
	RUBUSDRate(context.Context) (domain.Rate, error) // Возвращает текущий курс доллара к рублю.
	// Возвращает требуемое кол-во из таблицы курса BTC/USD.
	BtcRate(ctx context.Context, limit, offset int) ([]domain.Rate, error)
	// Возвращает текущий курс к рублю из таблицы курсов фиатных валют.
	FiatsCurrent(context.Context) ([]domain.Rate, error)
}
