package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"xtestserver/domain"
	api "xtestserver/pkg/api/rest"
	wsapi "xtestserver/pkg/api/websocket"
	"xtestserver/pkg/poller"
	"xtestserver/pkg/rates"
	"xtestserver/pkg/storage"
	"xtestserver/pkg/storage/postgres"
)

// переменная окружения
const (
	dbConnStrEnv = "POSTGRES_DB_URL"
	logfileEnv   = "LOG_FILE"
)

const (
	btcURL          = "https://api.kucoin.com/api/v1/market/stats?symbol=BTC-USDT" // ссылка на курс биткоина
	crbURL          = "http://www.cbr.ru/scripts/XML_daily.asp"                    // ссылка на курсы фиатных валют
	btcPollInterval = 10 * time.Second                                             // интервал опроса курса биткоина
	crbPollInterval = 24 * time.Hour                                               // интервал опроса курса фиатных валют
)

const (
	logIndent = 17
)

// имя подсистемы для логирования
var (
	restAPIName   = fmt.Sprintf("%*s", logIndent, "[REST API]: ")
	wsAPIName     = fmt.Sprintf("%*s", logIndent, "[WEBSOCKET API]: ")
	errLoggerName = fmt.Sprintf("%*s", logIndent, "[ERROR]: ")
)

// envs собирает ожидаемые переменные окружения,
// возвращает ошибку, если какая-либо из переменных env не задана.
func envs(envs ...string) (map[string]string, error) {
	em := make(map[string]string, len(envs))
	var ok bool
	for _, env := range envs {
		if em[env], ok = os.LookupEnv(env); !ok {
			log.Println(em[env])
			return nil, fmt.Errorf("environment variable %q must be set", env)
		}
	}
	return em, nil
}

func main() {

	em, err := envs(dbConnStrEnv, logfileEnv)
	if err != nil {
		log.Fatal(err)
	}

	logfile, err := os.OpenFile(em[logfileEnv], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = logfile.Close()
	}()

	db, err := postgres.New(em[dbConnStrEnv])
	if err != nil {
		_ = logfile.Close()
		log.Fatal(err)
	}
	defer db.Close()

	// создаем контекст для регулирования закрытия всех подсистем
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// опрашиваем url ссылки
	btc, btcPollErrs := poller.Poll(ctx, btcURL, btcPollInterval)
	crb, crbPollErrs := poller.Poll(ctx, crbURL, crbPollInterval)
	// десериализуем
	btcRates, btcErrs := domain.DecodeStream(btc, domain.JsonDec)
	crbRates, crbErrs := domain.DecodeStream(crb, domain.XmlDec)
	// обрабатываем десериализованные данные
	repls, procErrs := rates.ProcessStream(ctx, db, btcRates, rates.BtcProcessFunc)
	_, procCrbErrs := rates.ProcessStream(ctx, db, crbRates, rates.BtcProcessFunc)

	var wg sync.WaitGroup
	wg.Add(3)

	logFW := logWriter{w: logfile}              // потокобезопасный writer
	logout := io.MultiWriter(os.Stdout, &logFW) // куда пишем логи

	// читаем каналы с ошибками
	go errsLogger(io.MultiWriter(os.Stderr, &logFW), &wg,
		btcPollErrs, crbPollErrs, btcErrs, crbErrs, procErrs, procCrbErrs)

	servers := []*http.Server{
		startRestServer(db, logout, &wg),
		startWebsoketServer(ctx, logout, repls, &wg),
	}

	cancelation(cancel, logout, servers) // логика закрытия сервера

	wg.Wait() // ждём всех
}

// cancelation мониторит os-сингналы прерывания и в случае получения
// отменяет контекст приложения и "мягко" останавливает серверы
func cancelation(cancel context.CancelFunc, logout io.Writer, servers []*http.Server) {
	// ловим сигналы прерывания типа CTRL-C
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		s := <-stop // получили сигнал прерывания
		fmt.Fprintf(logout, "got os signal %q", s)

		// закрываем серверы
		for i := range servers {
			if err := servers[i].Shutdown(context.Background()); err != nil {
				fmt.Fprintf(logout, "%v", err)
			}
		}

		cancel() // закрываем контекст приложения
	}()
}

// startWebsoketServer запускает websoket сервер
func startWebsoketServer(ctx context.Context, logout io.Writer, upd <-chan []byte, wg *sync.WaitGroup) *http.Server {
	// WEBSOKET API
	logger := log.New(logout, wsAPIName, log.Lmsgprefix|log.LstdFlags)
	api := wsapi.New(ctx, logger, upd)

	// конфигурируем сервер WEBSOKET API
	srv := &http.Server{
		Addr:              ":8090",
		Handler:           api.Router(),
		ReadHeaderTimeout: time.Minute,
	}

	// сервер WEBSOKET API
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Println(err)
		}
		wg.Done()
	}()
	return srv
}

// startRestServer запускает REST API сервер
func startRestServer(db storage.Storage, logout io.Writer, wg *sync.WaitGroup) *http.Server {
	// REST API
	logger := log.New(logout, restAPIName, log.Lmsgprefix|log.LstdFlags)
	api := api.New(db, logger)

	// конфигурируем сервер REST API
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           api.Router(),
		IdleTimeout:       3 * time.Minute,
		ReadHeaderTimeout: time.Minute,
	}

	// сервер WEBSOKET API
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Println(err)
		}
		wg.Done()
	}()
	return srv
}

// errsLogger логирует ошибки из предоставленного списка каналов
func errsLogger(logout io.Writer, wg *sync.WaitGroup, errs ...<-chan error) {

	logger := log.New(logout, errLoggerName, log.Lmsgprefix|log.LstdFlags)

	// читаем все каналы ошибок и логгируем
	for _, ch := range errs {

		go func(c <-chan error) {

			for err := range c {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, context.DeadlineExceeded) {
					logger.Println(err)
				}
			}

		}(ch)
	}

	wg.Done()
}

type logWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (lw *logWriter) Write(b []byte) (n int, err error) {
	lw.mu.Lock()
	n, err = lw.w.Write(b)
	lw.mu.Unlock()
	return n, err
}
