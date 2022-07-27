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
	done    chan struct{}
}

// clients - потокобезопасное отображение.
type clients struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]struct{}
}

// put блокирует семафор и добавляет в отображение.
func (cm *clients) put(c *websocket.Conn) {
	cm.mu.Lock()
	cm.conns[c] = struct{}{}
	cm.mu.Unlock()
}

// delete блокирует семафор и удаляет из отображения.
func (cm *clients) delete(c *websocket.Conn) {
	cm.mu.Lock()
	delete(cm.conns, c)
	cm.mu.Unlock()
}

// clean блокирует семафор и удаляет
// из отображения все соединения.
func (cm *clients) clean() {
	cm.mu.Lock()
	for c := range cm.conns {
		delete(cm.conns, c)
	}
	cm.mu.Unlock()
}

// Возвращает новый объект *API
func New(ctx context.Context, logger *log.Logger, upd <-chan []byte) *API {
	api := API{
		r:      mux.NewRouter(),
		logger: logger,
		clients: clients{
			conns: make(map[*websocket.Conn]struct{}),
		},
		done: make(chan struct{}),
	}
	api.endpoints()
	go api.update(upd)
	go api.closer(ctx)
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

// closer слушает контектс и закрывает сервер
// в случае отмены контекста.
func (api *API) closer(ctx context.Context) {
	<-ctx.Done()
	close(api.done)
	api.broadcast(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server closed"))
	api.clients.clean()
}

// updater следит за каналом обновлений и
// вызывает рассылающую функцию в случае нового обновления.
// В случае закрытия контекста передает сигнал обработчикам
// о том что, пора тушить соединения.
func (api *API) update(upd <-chan []byte) {
	for u := range upd {
		api.broadcast(websocket.TextMessage, u)
	}
}

var upgrader = websocket.Upgrader{} // дефолтные опции.

// clientHandler апгрейдит нового клиента до WebSocket,
// и отправляет его в пул соединений.
// Если сервер закрыт к этому моменту, то сразу закрывает соединение.
func (api *API) clientHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		api.logger.Printf("upgrade: %v", err)
		return
	}
	select {
	case <-api.done:
		err := c.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server closed"))
		if err != nil {
			api.logger.Printf("close: %v", err)
		}
		_ = c.Close()
	default:
		api.logger.Printf("client connected: %s", c.RemoteAddr())
		api.clients.put(c) // сохраняем соединение
		go func() {
			for {
				_, _, err := c.ReadMessage()
				if err != nil {
					break
				}
			}
			api.logger.Printf("client disconnected: %s", c.RemoteAddr())
			api.clients.delete(c) // удаляем соединение
		}()
	}
}

// broadcast рассылает сообщение всем
// подключенным клиентам.
func (api *API) broadcast(messageType int, message []byte) {
	api.clients.mu.Lock()
	for c := range api.clients.conns {
		err := c.WriteMessage(messageType, message)
		if err != nil {
			api.logger.Printf("conn %q: write: %v", c.RemoteAddr(), err)
		}
	}
	api.clients.mu.Unlock()
}
