package rates

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
	"xtestserver/pkg/storage/memdb"
)

var btcusd = rate{Time: 1658423781, Value: 22_918.90}
var testRate1 = rate{Id: 0, CharCode: "USD", Nominal: 1, Time: 1658252361, Value: 22_918.90}
var testRate2 = rate{Id: 0, CharCode: "HUF", Nominal: 100, Time: 1658252361, Value: (22_918.90 * 56.4783) * 100 / 14.3324}
var testRate3 = rate{Id: 0, CharCode: "GBP", Nominal: 1, Time: 1658252361, Value: (22_918.90 * 56.4783) * 1 / 67.7627}

var testBox1 = struct {
	Label string         `json:"label"`
	Data  map[string]any `json:"data"`
}{
	Label: "BTC/USDT",
	Data:  map[string]any{"timestamp": 1658423781, "value": 22_918.90},
}

var testBox2 = struct {
	Label string `json:"label"`
	Data  []rate `json:"data"`
}{
	Label: "BTC/*",
	Data:  []rate{testRate1, testRate2, testRate3},
}

func TestProcessStream(t *testing.T) {
	out := make(chan []rate)
	vals, errs := ProcessStream(context.Background(), memdb.New(), out, BtcProcessFunc)

	go func() {
		out <- []rate{btcusd}
		close(out)
	}()

	done := make(chan struct{})
	go func() {
		for err := range errs {
			t.Errorf("ProcessStream() = err: %v", err)
		}
		close(done)
	}()

	select {
	case v1 := <-vals:
		// tm := domain.RateMapTimestamp([]rate{btcusd})
		var c = struct {
			Label string         `json:"label"`
			Data  map[string]any `json:"data"`
		}{}
		err := json.NewDecoder(bytes.NewReader(v1)).Decode(&c)
		if err != nil {
			t.Errorf("ProcessStream() = err: %v", err)
		}

		if reflect.DeepEqual(c, testBox1) {
			t.Errorf("ProcessStream() = %v, want %v", c, testBox1)
		}
	case <-time.After(time.Second):
		t.Fatal("ProcessStream() = err: didn't receive value from channel")
	}

	select {
	case v2 := <-vals:
		var c = struct {
			Label string         `json:"label"`
			Data  map[string]any `json:"data"`
		}{}
		err := json.Unmarshal(v2, &c)
		if err != nil {
			t.Errorf("ProcessStream() = err: %v", err)
		}
		if c.Label != testBox2.Label {
			t.Errorf("ProcessStream() = %v, want %v", c.Label, testBox2.Label)
		}
		if _, ok := c.Data[testRate1.CharCode]; !ok {
			t.Errorf("ProcessStream() = nothing in %v, want to find key %q", c.Data, testRate1.CharCode)
		}
	case <-time.After(time.Second):
		t.Fatal("ProcessStream() = err: didn't receive value from channel")
	}

	<-done
}
