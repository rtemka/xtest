package memdb

import (
	"context"
	"xtestserver/domain"
	"xtestserver/pkg/storage"
)

var _ storage.Storage = New()

type item = domain.Rate

// MemDB - заглушка настоящей БД
type MemDB struct{}

func New() *MemDB {
	return &MemDB{}
}

// SampleItem можно использовать для тестов
var SampleItem = item{
	Id:       1,
	CharCode: "USD",
	Nominal:  1,
	Time:     1658252361,
	Value:    100,
}

func (db *MemDB) RUBUSDRate(ctx context.Context) (item, error) {
	return SampleItem, nil
}

// AddBtcRate - no-op
func (db *MemDB) AddBtcRate(_ context.Context, _ item) error {
	return nil
}

// AddFiats - no-op
func (db *MemDB) AddFiats(_ context.Context, _ ...item) error {
	return nil
}

// BtcRate - возвращает столько Rate, сколько запрошено
func (db *MemDB) BtcRate(_ context.Context, limit, _ int) ([]item, error) {
	items := make([]item, 0, limit)
	for i := 0; i < limit; i++ {
		items = append(items, SampleItem)
	}
	return items, nil
}

// FiatsCurrent - no-op
func (db *MemDB) FiatsCurrent(_ context.Context) ([]item, error) {
	return nil, nil
}

// Close - no-op
func (db *MemDB) Close() error {
	return nil
}
