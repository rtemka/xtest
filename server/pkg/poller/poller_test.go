package poller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

const jblob = `
		{
			"code": "200000",
			"data": {
			"time": 1658237004004,
			"symbol": "BTC-USDT",
			"buy": "22278.80"
			}
		}`

func TestPoll(t *testing.T) {
	var m sync.Mutex
	var want int

	ctx, cancel := context.WithCancel(context.Background())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.Lock()
		defer m.Unlock()
		if want == 20 {
			cancel()
			return
		}
		want++
		fmt.Fprintln(w, jblob)
	}))
	defer ts.Close()

	t.Run("count_polls", func(t *testing.T) {

		values, errs := Poll(ctx, ts.URL, time.Millisecond)

		var wg sync.WaitGroup
		wg.Add(2)

		got := 0

		go func() {
			for range values {
				got++
			}
			wg.Done()
		}()

		go func() {
			for range errs {
			}
			wg.Done()
		}()

		wg.Wait()

		if got != want {
			t.Fatalf("Poll() got values = %d, want values = %d", got, want)
		}
	})
}
