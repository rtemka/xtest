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

	var clients = make([]*websocket.Conn, 0, howManyClients)

	for i := 0; i < howManyClients; i++ {
		u := url.URL{Scheme: "ws", Host: strings.TrimPrefix(ts.URL, "http://"), Path: "/"}

		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			t.Fatalf("clientHandler() = error %v", err)
		}
		clients = append(clients, c)
	}

	sended := "update"

	var wg sync.WaitGroup
	wg.Add(len(clients))
	var wg2 sync.WaitGroup
	wg2.Add(len(clients))

	var count int
	var mu sync.Mutex

	for i := range clients {
		go func(idx int) {
			wg.Done()
			defer wg2.Done()
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

	wg.Wait() // все клиенты готовы читать
	for i := 0; i < howManyMessages; i++ {
		ch <- []byte(sended)
	}
	close(ch)
	wg2.Wait() // все клиенты прочитали

	for i := range clients {
		_ = clients[i].Close()
	}

	if count != howManyClients*howManyMessages {
		t.Fatalf("clientHandler() total messages = %d, want %d", count, howManyClients*howManyMessages)
	}
}
