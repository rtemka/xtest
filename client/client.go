package main

import (
	"errors"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	ws "github.com/gorilla/websocket"
)

const serverAddr = "SERVER_URL"

const (
	retries       = 1000
	retryInterval = 5 * time.Second
)

var ErrRetryExceeded = errors.New("connect: number of retries exceeded")

func connect(url url.URL, retries int, interval time.Duration) (*ws.Conn, error) {

	log.Printf("connecting to %s...", url.String())

	for i := 0; i < retries; i++ {

		c, _, err := ws.DefaultDialer.Dial(url.String(), nil)
		if err != nil {
			log.Println("dial:", err)
			time.Sleep(interval)
			log.Printf("retry connecting to %s...", url.String())
			continue
		}

		log.Printf("connection to %s established", url.String())
		return c, nil
	}

	return nil, ErrRetryExceeded
}

func main() {
	addr := os.Getenv(serverAddr)
	if addr == "" {
		log.Fatalf("environment variable %q must be set", serverAddr)
	}
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Lmsgprefix | log.LstdFlags)
	log.SetPrefix("[ws CLIENT]: ")

	u := url.URL{Scheme: "ws", Host: addr, Path: "/"}

	c, err := connect(u, retries, retryInterval)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if ws.IsCloseError(err, ws.CloseNormalClosure) {
					log.Println("read:", err)
					break
				}
				if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway,
					ws.CloseAbnormalClosure, ws.CloseInternalServerErr,
					ws.CloseServiceRestart, ws.CloseTryAgainLater) {

					log.Printf("error: %v", err)
					log.Println("trying to reconnect")

					c.Close()
					c, err = connect(u, retries, retryInterval)
					if err != nil {
						log.Fatal(err)
					}
					continue
				}
			}

			log.Printf("recv: %s", message)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-done:
	case s := <-stop:
		log.Printf("got os signal %q", s)
		err = c.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.CloseNormalClosure, ""))
		if err != nil {
			log.Println("write close:", err)
		}
	}
}
