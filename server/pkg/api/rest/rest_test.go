// пакет api предоставляет маршрутизатор REST API

package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"xtestserver/pkg/storage/memdb"

	"github.com/gorilla/mux"
)

var testServerUrl = "localhost:5080"
var api *API

type stamp struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

type hist struct {
	Total   int     `json:"total"`
	History []stamp `json:"history"`
}

func TestMain(m *testing.M) {
	api = New(memdb.New(), log.New(os.Stdout, "", 0))
	ts := httptest.NewServer(api.Router())
	defer ts.Close()

	testServerUrl = ts.URL

	os.Exit(m.Run())
}

func Test_endpoints(t *testing.T) {
	err := api.r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.URLPath()
		methods, _ := route.GetMethods()
		url := fmt.Sprintf("%s%s", testServerUrl, path)

		for i := range methods {

			if methods[i] == http.MethodGet {
				res, err := http.Get(url)
				if err != nil {
					t.Errorf("endpoints() = err: %v", err)
				}
				if res.StatusCode != http.StatusOK {
					t.Errorf("endpoints() %s %q status code = %d, want %d",
						http.MethodPost, url, res.StatusCode, http.StatusOK)
				}
			}
			if methods[i] == http.MethodPost {
				res, err := http.Post(url, "", nil)
				if err != nil {
					t.Errorf("endpoints() %s %q = err: %v", http.MethodPost, url, err)
				}
				if res.StatusCode != http.StatusOK {
					t.Errorf("endpoints() %s %q status code = %d, want %d",
						http.MethodPost, url, res.StatusCode, http.StatusOK)
				}
			}
		}

		return nil
	})
	if err != nil {
		t.Errorf("endpoints() = err: %v", err)
	}
}

func TestAPI_btcusdtLatestHandler(t *testing.T) {

	url := fmt.Sprintf("%s/api/btcusdt", testServerUrl)

	res, err := http.Get(url)
	if err != nil {
		t.Errorf("btcusdtLatestHandler() = err: %v", err)
	}

	got := struct {
		Timestamp int64   `json:"timestamp"`
		Value     float64 `json:"value"`
	}{}

	want := struct {
		Timestamp int64   `json:"timestamp"`
		Value     float64 `json:"value"`
	}{1658252361, 56.4783}

	err = json.NewDecoder(res.Body).Decode(&got)
	if err != nil {
		t.Errorf("btcusdtLatestHandler() = err: %v", err)
	}

	if got != want {
		t.Errorf("btcusdtLatestHandler() = %v, want %v", got, want)
	}
}

func TestAPI_btcusdtHistoryHandler(t *testing.T) {

	limit, offset := 10, 5
	url := fmt.Sprintf("%s/api/btcusdt?limit=%d&offset=%d", testServerUrl, limit, offset)

	res, err := http.Post(url, "", nil)
	if err != nil {
		t.Errorf("btcusdtHistoryHandler() = err: %v", err)
	}

	var got hist

	err = json.NewDecoder(res.Body).Decode(&got)
	if err != nil {
		t.Errorf("btcusdtHistoryHandler() = err: %v", err)
	}

	if got.Total != limit+offset {
		t.Errorf("btcusdtHistoryHandler() total = %v, want %v", got.Total, limit+offset)
	}
}

func TestAPI_fiatsRubLatestHandler(t *testing.T) {
	url := fmt.Sprintf("%s/api/currencies", testServerUrl)

	res, err := http.Get(url)
	if err != nil {
		t.Errorf("fiatsRubLatestHandler() = err: %v", err)
	}

	var got map[string]any

	err = json.NewDecoder(res.Body).Decode(&got)
	if err != nil {
		t.Errorf("fiatsRubLatestHandler() = err: %v", err)
	}

	if _, ok := got["USD"]; !ok {
		t.Error("fiatsRubLatestHandler() = nothing, want USD")
	}
}

func TestAPI_fiatsRubHistoryHandler(t *testing.T) {
	// использование здесь ?date=gte:2022-05-23 не влияет ни на что
	url := fmt.Sprintf("%s/api/currencies?date=gte:2022-05-23&limit=2&offset=1", testServerUrl)

	res, err := http.Post(url, "", nil)
	if err != nil {
		t.Errorf("fiatsRubHistoryHandler() = err: %v", err)
	}

	got := struct {
		Total   int              `json:"total"`
		History []map[string]any `json:"history"`
	}{}

	want := struct {
		Total   int              `json:"total"`
		History []map[string]any `json:"history"`
	}{
		Total:   1,
		History: []map[string]any{{"USD": 56.4783, "date": "2022-07-19"}},
	}

	err = json.NewDecoder(res.Body).Decode(&got)
	if err != nil {
		t.Errorf("fiatsRubHistoryHandler() = err: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("fiatsRubHistoryHandler() = %v, want %v", got, want)
	}
}
