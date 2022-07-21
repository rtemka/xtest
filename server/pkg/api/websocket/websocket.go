package ws

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// API - websocket сервер
type API struct {
	r       *mux.Router
	logger  *log.Logger
	clients clients
	upd     <-chan []byte
	done    chan struct{}
}

// clients - потокобезопасное отображение.
type clients struct {
	mu    sync.RWMutex
	conns map[*websocket.Conn]bool
}

// put блокирует семафор и добавляет в отображение.
func (cm *clients) put(c *websocket.Conn) {
	cm.mu.Lock()
	cm.conns[c] = true
	cm.mu.Unlock()
}

// delete блокирует семафор и удаляет из отображения.
func (cm *clients) delete(c *websocket.Conn) {
	cm.mu.Lock()
	delete(cm.conns, c)
	cm.mu.Unlock()
}

// len возвращает текущее число
// элементов в отображении.
func (cm *clients) len() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.conns)
}

// Возвращает новый объект *API
func New(ctx context.Context, logger *log.Logger, upd <-chan []byte) *API {
	api := API{
		r:      mux.NewRouter(),
		logger: logger,
		upd:    upd,
		clients: clients{
			conns: make(map[*websocket.Conn]bool),
		},
		done: make(chan struct{}),
	}
	api.endpoints()
	go api.updater(ctx)
	return &api
}

// endpoints - регистрирует обработчики запросов.
func (api *API) endpoints() {
	api.r.HandleFunc("/", api.clientHandler)
}

// Router возвращает маршрутизатор.
func (api *API) Router() *mux.Router {
	return api.r
}

// updater следит за каналом обновлений и
// вызывает рассылающую функцию в случае нового обновления.
// В случае закрытия контекста передает сигнал обработчикам
// о том что, пора тушить соединения.
func (api *API) updater(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(api.done)
			return
		case u := <-api.upd:
			if api.clients.len() != 0 {
				api.broadcast(u)
			}
		}
	}
}

var upgrader = websocket.Upgrader{} // дефолтные опции.

// clientHandler регистрирует нового клиента
// во внутренней мапе API, ждет сигнала закрытия.
// Получив сигнал, удаляет клиента и закрывает соединение.
func (api *API) clientHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		api.logger.Printf("upgrade: %v", err)
		return
	}

	api.clients.put(c) // сохраняем соединение

	done := make(chan struct{})
	// слушаем, если клиент сам закроет соединение
	go func() {
		defer close(done)
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				break
			}
		}
	}()

	select {
	case <-done: // клиент сам закрыл соединение
	case <-api.done: // сигнал закрытия сервера
		err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			api.logger.Printf("close: %v", err)
		}
	}
	_ = c.Close()
	api.clients.delete(c) // удаляем соединение
}

func (api *API) broadcast(message []byte) {
	api.clients.mu.RLock()
	defer api.clients.mu.RUnlock()

	for c := range api.clients.conns {
		err := c.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			api.logger.Printf("conn %q: write: %v", c.RemoteAddr(), err)
		}
	}
}
