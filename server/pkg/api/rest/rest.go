// пакет api предоставляет маршрутизатор REST API
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"xtestserver/domain"
	"xtestserver/pkg/storage"
	"xtestserver/rates"

	"github.com/gorilla/mux"
)

type stor = storage.Storage
type filter = storage.Filter
type item = domain.Rate

const (
	dateTimeFilter = "date"
	currencyFilter = "currency"
	limit          = "limit"
	offset         = "offset"
)

const (
	layoutDateTime = "2006-01-02T15:04:04" // YYYY-MM-DDTHH:MM:SS
	layoutDate     = "2006-01-02"          // YYYY-MM-DD
)

// REST API приложения.
type API struct {
	r      *mux.Router
	db     stor
	logger *log.Logger
}

// Возвращает новый объект *API.
func New(storage stor, logger *log.Logger) *API {
	api := API{
		r:      mux.NewRouter(),
		db:     storage,
		logger: logger,
	}
	api.endpoints()
	return &api
}

// Router возвращает маршрутизатор запросов.
func (api *API) Router() *mux.Router {
	return api.r
}

func (api *API) endpoints() {
	api.r.Use(api.logRequestMiddleware, api.closerMiddleware, api.headersMiddleware)
	api.r.HandleFunc("/api/btcusdt", api.btcusdtLatestHandler).Methods(http.MethodGet, http.MethodOptions)
	api.r.HandleFunc("/api/btcusdt", api.btcusdtHistoryHandler).Methods(http.MethodPost, http.MethodOptions) // почему POST???
	api.r.HandleFunc("/api/latest", api.fiatsBTCLatestHandler).Methods(http.MethodGet, http.MethodOptions)
	api.r.HandleFunc("/api/currencies", api.fiatsRubLatestHandler).Methods(http.MethodGet, http.MethodOptions)
	api.r.HandleFunc("/api/currencies", api.fiatsRubHistoryHandler).Methods(http.MethodPost, http.MethodOptions) // почему POST???
}

// closerMiddleware читает и закрывает тело запроса
// для переиспользования TCP-соединения.
func (api *API) closerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	})
}

// headersMiddleware устанавливает хедеры всем ответам.
func (api *API) headersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

// logRequestMiddleware логирует request
func (api *API) logRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.logger.Printf("method=%s, path=%s, query=%s host=%s", r.Method, r.URL.Path, r.URL.Query(), r.Host)
		next.ServeHTTP(w, r)
	})
}

// btcusdtLatestHandler возвращает последнее
// (текущее) значение пары BTC/USDT.
func (api *API) btcusdtLatestHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	latest, err := api.db.BtcRate(ctx, filter{Limit: 1})
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	// Не нашли в БД ничего
	if len(latest) == 0 {
		http.Error(w, "latest rate is not found", http.StatusNotFound)
	}
	m := domain.RateMapTimestamp(latest)
	// Отправка данных клиенту в формате JSON.
	json.NewEncoder(w).Encode(&m)
}

// btcusdtHistoryHandler возвращает
// историю BTC/USDT с фильтрами по дате и времени и пагинацией.
func (api *API) btcusdtHistoryHandler(w http.ResponseWriter, r *http.Request) {

	f, err := api.parseQP(r.URL, layoutDateTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	items, err := api.db.BtcRate(ctx, f)
	if err != nil {
		api.logger.Printf("db err: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type h struct {
		Timestamp int64   `json:"timestamp"`
		Value     float64 `json:"value"`
	}
	history := make([]h, len(items))

	for i := range items {
		history[i].Timestamp = items[i].Time
		history[i].Value = items[i].Value
	}

	box := struct {
		Total   int `json:"total"`
		History []h `json:"history"`
	}{
		Total:   len(items),
		History: history,
	}
	// Отправка данных клиенту в формате JSON.
	json.NewEncoder(w).Encode(&box)
}

// fiatsRubLatestHandler возвращает последние
// (текущие) значения фиатных валют к RUB.
func (api *API) fiatsRubLatestHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	latest, err := api.db.FiatsCurrent(ctx)
	if err != nil {
		api.logger.Printf("db err: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	// Не нашли в БД ничего
	if len(latest) == 0 {
		http.Error(w, "latest RUB rates not found", http.StatusNotFound)
		return
	}
	m := domain.RateMap(latest)
	// Отправка данных клиенту в формате JSON.
	json.NewEncoder(w).Encode(&m)
}

// fiatsRubLatestHandler возвращает историю изменения фиатных валют
// к RUB с фильтрами по дате и валюте и пагинацией.
func (api *API) fiatsRubHistoryHandler(w http.ResponseWriter, r *http.Request) {

	f, err := api.parseQP(r.URL, layoutDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	items, err := api.db.Fiats(ctx, f)
	if err != nil {
		api.logger.Printf("db err: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Не нашли в БД ничего
	if len(items) == 0 {
		http.Error(w, "history RUB rates not found", http.StatusNotFound)
		return
	}

	var m = map[string]any{"date": time.Unix(items[0].Time, 0).Format("2006-01-02")}
	var history = []map[string]any{m}

	for i, j, t := 0, 0, items[0].Time; i < len(items); i, t = i+1, items[i].Time {
		if t != items[i].Time {
			history = append(history, m)
			j++
			m = make(map[string]any)
			history[j]["date"] = time.Unix(items[i].Time, 0).Format("2006-01-02")
		}
		history[j][items[i].CharCode] = items[i].Value
	}

	box := struct {
		Total   int              `json:"total"`
		History []map[string]any `json:"history"`
	}{
		Total:   len(history),
		History: history,
	}

	// Отправка данных клиенту в формате JSON.
	json.NewEncoder(w).Encode(&box)
}

// fiatsBTCLatestHandler возвращает последние
// (текущие) значения фиатных валют к BTC.
func (api *API) fiatsBTCLatestHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// последний курс BTC/USDT
	latest, err := api.db.BtcRate(ctx, filter{Limit: 1})
	if err != nil || len(latest) == 0 {
		api.logger.Printf("db err: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	// считаем курсы фиатных валют к BTC
	rates, err := rates.CalcRates(ctx, api.db, latest[0].Value)
	if err != nil || len(rates) == 0 {
		api.logger.Printf("db err: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Отправка данных клиенту в формате JSON.
	json.NewEncoder(w).Encode(&rates)
}

func op(o string) string {
	if o == "gte" {
		return ">="
	} else if o == "lte" {
		return "<="
	}
	return "="
}

// parseQP - парсит параметеры запроса ?limit=NUM, ?offset=NUM
// и ?date=[gte:lte:]YYYY-MM-DDTHH:MM:SS
func (api *API) parseQP(url *url.URL, dateTimeLayout string) (filter, error) {
	f, err := timeQParser(url, dateTimeFilter, dateTimeLayout)
	if err != nil {
		api.logger.Printf("parse query param %q: %v", url.Query().Get(dateTimeFilter), err)
		return filter{}, errors.New("bad datetime parameter")
	}

	qp := url.Query().Get(limit)
	if qp != "" {
		f.Limit, err = strconv.Atoi(qp)
		if err != nil {
			api.logger.Printf("parse query param %q: %v", qp, err)
			return filter{}, errors.New("bad limit parameter")
		}
	}
	qp = url.Query().Get(offset)
	if qp != "" {
		f.Offset, err = strconv.Atoi(qp)
		if err != nil {
			api.logger.Printf("parse query param %q: %v", qp, err)
			return filter{}, errors.New("bad offset parameter")
		}
	}
	f.Currency = url.Query().Get(currencyFilter)

	return f, nil
}

// timeQParser - парсит параметер запроса ?date=[gte:lte:]YYYY-MM-DDTHH:MM:SS
func timeQParser(url *url.URL, qpname, layout string) (filter, error) {
	qp := url.Query().Get(qpname)
	if qp == "" {
		return filter{}, nil
	}
	if strings.HasPrefix(qp, "gte:") || strings.HasPrefix(qp, "lte:") {

		split := strings.SplitN(qp, ":", 2)
		// ожидаем в виде ?date=[gte:lte:]2012-12-31T22:00:00
		t, err := time.Parse(layout, split[1])
		if err != nil {
			return filter{}, err
		}
		return filter{Time: t.Unix(), Operator: op(split[0])}, nil
	}

	t, err := time.Parse(layout, qp) // ожидаем в виде ?date=2012-12-31T22:00:00
	if err != nil {
		return filter{}, err
	}

	return filter{Time: t.Unix(), Operator: "="}, nil
}
