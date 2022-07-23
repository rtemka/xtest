package rates

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"xtestserver/pkg/storage/memdb"
)

var btcusd = rate{Time: 1658423781, Value: 22_918.90}
var testRate1 = rate{Id: 0, CharCode: "USD", Nominal: 1, Time: 1658252361, Value: 22_918.90}
var testRate2 = rate{Id: 0, CharCode: "HUF", Nominal: 100, Time: 1658252361, Value: (22_918.90 * 56.4783) * 100 / 14.3324}
var testRate3 = rate{Id: 0, CharCode: "GBP", Nominal: 1, Time: 1658252361, Value: (22_918.90 * 56.4783) * 1 / 67.7627}

var testBox1 = struct {
	Label string
	Data  rate `json:"data"`
}{
	Label: "BTC/USDT",
	Data:  btcusd,
}

var testBox2 = struct {
	Label string
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

		var c = struct {
			Label string
			Data  rate `json:"data"`
		}{}
		err := json.Unmarshal(v1, &c)
		if err != nil {
			t.Errorf("ProcessStream() = err: %v", err)
		}
		if c != testBox1 {
			t.Errorf("ProcessStream() = %v, want %v", c, testBox1)
		}
	case <-time.After(time.Second):
		t.Fatal("ProcessStream() = err: didn't receive value from channel")
	}

	select {
	case v2 := <-vals:

		var c = struct {
			Label string
			Data  []rate `json:"data"`
		}{}
		err := json.Unmarshal(v2, &c)
		if err != nil {
			t.Errorf("ProcessStream() = err: %v", err)
		}
		if c.Label != testBox2.Label {
			t.Errorf("ProcessStream() = %v, want %v", c.Label, testBox2.Label)
		}
		for i := range c.Data {
			if c.Data[i].CharCode != testBox2.Data[i].CharCode &&
				c.Data[i].Nominal != testBox2.Data[i].Nominal &&
				c.Data[i].Time != testBox2.Data[i].Time &&
				!floatEqual(c.Data[i].Value, testBox2.Data[i].Value) {
				t.Errorf("ProcessStream() = %v, want %v", c.Data[i], testBox2.Data[i])
			}
		}
	case <-time.After(time.Second):
		t.Fatal("ProcessStream() = err: didn't receive value from channel")
	}

	<-done
}
