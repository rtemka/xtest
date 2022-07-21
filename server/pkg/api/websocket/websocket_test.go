package ws

import (
	"context"
	"io"
	"log"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

const howManyClients = 20
const howManyMessages = 20

func TestAPI_clientHandler(t *testing.T) {

	ch := make(chan []byte)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api := New(ctx, log.New(io.Discard, "", 0), ch)

	ts := httptest.NewServer(api.Router())
	defer ts.Close()

	var clients []*websocket.Conn

	for i := 0; i < howManyClients; i++ {
		u := url.URL{Scheme: "ws", Host: strings.TrimPrefix(ts.URL, "http://"), Path: "/"}

		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			t.Fatalf("clientHandler() = error %v", err)
		}
		clients = append(clients, c)
	}

	defer func() {
		for i := range clients {
			_ = clients[i].Close()
		}
	}()

	sended := "update"
	for i := 0; i < howManyMessages; i++ {
		ch <- []byte(sended)
	}
	close(ch)

	var wg sync.WaitGroup
	wg.Add(len(clients))

	var mu sync.Mutex
	var count int

	for i := range clients {
		go func(idx int) {
			defer wg.Done()
			for i := 0; i < howManyMessages; i++ {
				mt, received, err := clients[idx].ReadMessage()
				if err != nil {
					t.Errorf("clientHandler() = read err: %v", err)
				}
				if mt != websocket.TextMessage {
					t.Errorf("clientHandler() message type = %d, want %d", mt, websocket.TextMessage)
				}
				if string(received) != sended {
					t.Errorf("clientHandler() = %q, want %q", string(received), sended)
				}
				mu.Lock()
				count++
				mu.Unlock()
			}
		}(i)
	}

	cancel()
	wg.Wait()

	if count != howManyClients*howManyMessages {
		t.Fatalf("clientHandler() total messages = %d, want %d", count, howManyClients*howManyMessages)
	}
}
