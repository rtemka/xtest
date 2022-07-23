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
func (db *MemDB) BtcRate(_ context.Context, filter storage.Filter) ([]item, error) {
	items := make([]item, 0, filter.Limit+filter.Offset)
	for i := 0; i < filter.Limit+filter.Offset; i++ {
		items = append(items, SampleItem)
	}
	return items, nil
}

// FiatsCurrent - return exactly three items
func (db *MemDB) FiatsCurrent(_ context.Context) ([]item, error) {
	return []item{SampleItem, SampleItem2, SampleItem3}, nil
}

// Fiats - возвращает столько Rate, сколько запрошено,
// если нет фильтров, то ровно три
func (db *MemDB) Fiats(_ context.Context, filter storage.Filter) ([]item, error) {
	if filter.Limit+filter.Offset == 0 {
		return []item{SampleItem, SampleItem2, SampleItem3}, nil
	}
	items := make([]item, 0, filter.Limit+filter.Offset)
	for i := 0; i < filter.Limit+filter.Offset; i++ {
		items = append(items, SampleItem)
	}
	return items, nil
}

// Close - no-op
func (db *MemDB) Close() error {
	return nil
}

// SampleItem можно использовать для тестов
var SampleItem = item{
	Id:       1,
	CharCode: "USD",
	Nominal:  1,
	Time:     1658252361,
	Value:    56.4783,
}

var SampleItem2 = item{
	Id:       2,
	CharCode: "HUF",
	Nominal:  100,
	Time:     1658252361,
	Value:    14.3324,
}
var SampleItem3 = item{
	Id:       3,
	CharCode: "GBP",
	Nominal:  1,
	Time:     1658252361,
	Value:    67.7627,
}
