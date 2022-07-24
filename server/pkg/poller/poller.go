// Пакет poller предоставляет функционал интервального опроса
// ресурса по сети.
package poller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Poll - опрашивает переданную url-ссылку
// с заданным интервалом и отдает байты тела ответа в канал.
func Poll(ctx context.Context, url string, interval time.Duration) (<-chan []byte, <-chan error) {

	out := make(chan []byte)
	errs := make(chan error)

	go func() {
		defer func() {
			close(out)
			close(errs)
		}()

		poll := func() {
			b, err := request(ctx, url) // выполняем опрос
			if err == nil {
				out <- b
			} else {
				errs <- fmt.Errorf("poll %s: %w", url, err)
			}
		}

		poll() // первый опрос сразу

		for {
			select {
			case <-time.After(interval):
				poll()
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			}
		}

	}()

	return out, errs
}

// request - вспомогательная функция, выполняющая обращение
// к ресурсу с таймаутом.
func request(ctx context.Context, url string) ([]byte, error) {
	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(c, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// чтобы сервера не посылали нам ошибку 403
	// ставим заголовок User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return io.ReadAll(resp.Body)
}
