// пакет rates реализовывает логику обработки курсов валют
package rates

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"xtestserver/domain"
	"xtestserver/pkg/storage"
)

type stor = storage.Storage
type rate = domain.Rate

// ProcessStream получает канал входящих данных
// и функцию для обработки этих данных. Пропускает данные
// через функцию и возвращает канал выходных данных и канал ошибок.
func ProcessStream(ctx context.Context, db stor, in <-chan []rate, procFunc ProcessFunc) (<-chan []byte, <-chan error) {

	out := make(chan []byte)
	errs := make(chan error, 2)

	go func() {

		defer func() {
			close(out)
			close(errs)
		}()

		f := procFunc(db, out, errs)

		for v := range in {
			f(ctx, v)
		}
	}()

	return out, errs
}

// ProcessFunc возвращает функцию, которая способна
// обработать срез входящих данных и отправить обработанный результат
// в канал выходящих данных.
type ProcessFunc func(db stor, out chan<- []byte, errs chan<- error) processor
type processor func(ctx context.Context, r []rate)

// FiatProcessFunc это ProcessFunc, которая
// возвращает обработчик курсов фиатных валют.
func FiatProcessFunc(db stor, _ chan<- []byte, errs chan<- error) processor {
	return func(ctx context.Context, r []rate) {
		if err := db.AddFiats(ctx, r...); err != nil {
			errs <- fmt.Errorf("process fiats update stream: %w", err)
		}
	}
}

// BtcProcessFunc это ProcessFunc, которая
// возвращает обработчик курса BTC/USD.
func BtcProcessFunc(db stor, out chan<- []byte, errs chan<- error) processor {
	// ship упаковывает данные, сериализует и отправляет
	// в канал, если ошибка сериализации, то ошибка в канал ошибок
	ship := func(label string, m map[string]any) {
		box := struct {
			Label string         `json:"label"`
			Data  map[string]any `json:"data"`
		}{
			Label: label,
			Data:  m,
		}
		b, err := json.Marshal(box)
		if err == nil {
			out <- b
		} else {
			errs <- fmt.Errorf("process btc update stream: %w", err)
		}
	}

	// upd отправляет обновления в таблицу BTC/USD
	upd := func(ctx context.Context, r rate) {
		if err := db.AddBtcRate(ctx, r); err != nil {
			errs <- err
		}
	}

	// isNew сравнивает текущее значение BTC/USD с предыдущим
	isNew := func(ctx context.Context, new float64) bool {
		old, err := db.BtcRate(ctx, storage.Filter{Limit: 1})
		if err != nil {
			errs <- err
			return true
		}
		if len(old) == 0 {
			return true
		}
		return !floatEqual(old[0].Value, new) // если не равны
	}

	calc := func(ctx context.Context, r rate) map[string]any {
		rates, err := CalcRates(ctx, db, r.Value)
		if err != nil {
			errs <- fmt.Errorf("process btc update stream: %w", err)
		}
		return rates
	}

	return func(ctx context.Context, r []rate) {
		// ожидаем только один элемент
		if len(r) == 0 {
			return
		}

		if isNew(ctx, r[0].Value) {
			go upd(ctx, r[0])
			ship("BTC/USDT", domain.RateMapTimestamp(r))
			ship("BTC/*", calc(ctx, r[0]))
		}
	}
}

// CalcRates рассчитывает курс фиатных валют по отношению к BTC
func CalcRates(ctx context.Context, db stor, btsusdt float64) (map[string]any, error) {
	rub, err := db.RUBUSDRate(ctx) // получаем курс рубля к доллару
	if err != nil {
		return nil, err
	}

	rates, err := db.FiatsCurrent(ctx) // получаем курсы валют к рублю
	if err != nil {
		return nil, err
	}

	rcc := btsusdt * rub.Value // кросс-курс BTC/RUB
	m := make(map[string]any, len(rates)+1)
	m["RUB"] = rcc
	for i := range rates {
		if rates[i].Nominal != 0 {
			// считаем кросс-курс валют к биткоину через рубль :)
			m[rates[i].CharCode] = rcc * float64(rates[i].Nominal) / rates[i].Value
		}
	}
	return m, nil
}

// floatEqual - равны ли float64 приблизительно?:)
func floatEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9
}
