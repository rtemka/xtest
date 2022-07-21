package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}
}

func Test_connect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(echo))
	defer ts.Close()
	u := url.URL{Scheme: "ws", Host: strings.TrimPrefix(ts.URL, "http://"), Path: "/"}

	c, err := connect(u, 1, time.Microsecond)
	if err != nil {
		t.Errorf("connect() = err: %v", err)
	}
	c.Close()
	ts.Close()

	_, err = connect(u, 2, time.Nanosecond)
	if err != ErrRetryExceeded {
		t.Errorf("connect() = %v, want %v", err, ErrRetryExceeded)
	}
}
