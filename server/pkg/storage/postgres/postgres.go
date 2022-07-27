// Пакет postgres представляет релизацию контракта БД.
package postgres

import (
	"context"
	"fmt"
	"xtestserver/domain"
	"xtestserver/pkg/storage"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var ErrNoRows = pgx.ErrNoRows

// Postgres выполняет CRUD операции с БД
type Postgres struct {
	db *pgxpool.Pool
}

// New выполняет подключение
// и возвращает объект для взаимодействия с БД
func New(connString string) (*Postgres, error) {

	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return nil, err
	}

	return &Postgres{db: pool}, pool.Ping(context.Background())
}

// Close выполняет закрытие подключения к БД
func (p *Postgres) Close() error {
	p.db.Close()
	return nil
}

type statement struct {
	sql  string
	args []any
}

// AddBtcRate добавляет в БД текущий курс BTC/USD.
func (p *Postgres) AddBtcRate(ctx context.Context, rate domain.Rate) error {
	sql := `INSERT INTO btc_usdt(time, value) VALUES ($1, $2);`
	return p.exec(ctx, sql, rate.Time, rate.Value)
}

// AddFiats добавляет в БД текущий курс фиатных валют.
func (p *Postgres) AddFiats(ctx context.Context, rates ...domain.Rate) error {

	sql := `
			INSERT INTO fiats(char_code, nominal) 
			VALUES ($1, $2)
			ON CONFLICT(char_code) DO NOTHING;`

	stmts := make([]statement, 0, len(rates)*2)

	for i := range rates {
		stmts = append(stmts, statement{sql: sql,
			args: []any{rates[i].CharCode, rates[i].Nominal}})
	}

	sql = `INSERT INTO rub(char_code, time, value) VALUES ($1, $2, $3);`
	for i := range rates {
		stmts = append(stmts, statement{sql: sql,
			args: []any{rates[i].CharCode, rates[i].Time, rates[i].Value}})
	}

	return p.execBatch(ctx, stmts...)
}

// BtcRate возвращает курс BTC/USDT. Накладывает на результаты
// фильтры по дате и пагинации, если есть.
func (p *Postgres) BtcRate(ctx context.Context, filter storage.Filter) ([]domain.Rate, error) {
	var stmt statement
	var limit string
	if filter.Limit > 0 {
		limit = "LIMIT $1"
		stmt.args = append(stmt.args, filter.Limit)
	}
	if filter.Time > 0 {
		stmt.sql = fmt.Sprintf(`
		SELECT btc.id, btc.time, btc.value
		FROM btc_usdt as btc 
		JOIN (SELECT btc_usdt.id FROM btc_usdt WHERE btc_usdt.time %s $%d %s OFFSET $%d) 
		as btc_offset ON btc.id = btc_offset.id ORDER BY btc.id DESC;`,
			filter.Operator, len(stmt.args)+1, limit, len(stmt.args)+2)

		stmt.args = append(stmt.args, filter.Time, filter.Offset)
	} else {
		stmt.sql = fmt.Sprintf(`
		SELECT btc.id, btc.time, btc.value FROM btc_usdt as btc
		ORDER BY btc.id DESC %s OFFSET $%d;`, limit, len(stmt.args)+1)
		stmt.args = append(stmt.args, filter.Offset)
	}
	return p.btcrate(ctx, stmt.sql, stmt.args...)
}

func (p *Postgres) btcrate(ctx context.Context, sql string, args ...any) ([]domain.Rate, error) {

	var rates []domain.Rate

	rows, err := p.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var rate domain.Rate

		err := rows.Scan(&rate.Id, &rate.Time, &rate.Value)
		if err != nil {
			return nil, err
		}

		rates = append(rates, rate)
	}

	return rates, rows.Err()
}

// FiatsCurrent возвращает текущий курс к рублю
// всех фиатных валют.
func (p *Postgres) FiatsCurrent(ctx context.Context) ([]domain.Rate, error) {
	sql := `
		SELECT rub.id, fiats.char_code, fiats.nominal, rub.time, rub.value	
		FROM rub JOIN (SELECT rub.id FROM rub LIMIT (SELECT COUNT(fiats.char_code) FROM fiats)) 
		as rub_offset ON rub.id = rub_offset.id
		JOIN fiats ON rub.char_code = fiats.char_code;`
	return p.fiats(ctx, sql)
}

// Fiats возвращает отфильтрованное по дате, валюте
// кол-во из таблицы курса фиатных валют.
func (p *Postgres) Fiats(ctx context.Context, filter storage.Filter) ([]domain.Rate, error) {
	var stmt statement
	stmt.sql = `
		SELECT rub.id, fiats.char_code, fiats.nominal, rub.time, rub.value	
		FROM rub JOIN fiats ON rub.char_code = fiats.char_code
	`

	if filter.Time > 0 {
		stmt.args = append(stmt.args, filter.Time)
		stmt.sql = fmt.Sprintf(`
		SELECT rub.id, fiats.char_code, fiats.nominal, rub.time, rub.value	
		FROM rub JOIN (SELECT rub.id FROM rub WHERE rub.time %s $1) 
		as rub_offset ON rub.id = rub_offset.id
		JOIN fiats ON rub.char_code = fiats.char_code`, filter.Operator)
		if filter.Currency != "" {
			stmt.sql += " WHERE fiats.char_code = $2"
			stmt.args = append(stmt.args, filter.Currency)
		}
	}
	if filter.Currency != "" && filter.Time == 0 {
		stmt.args = append(stmt.args, filter.Currency)
		stmt.sql = `
			SELECT rub.id, fiats.char_code, fiats.nominal, rub.time, rub.value	
			FROM (SELECT fiats.char_code, fiats.nominal FROM fiats WHERE fiats.char_code = $1)
			as fiats JOIN rub ON rub.char_code = fiats.char_code`
	}
	if filter.Limit > 0 {
		stmt.sql = fmt.Sprintf("%s LIMIT $%d", stmt.sql, len(stmt.args)+1)
		stmt.args = append(stmt.args, filter.Limit)
	}
	if filter.Offset > 0 {
		stmt.sql = fmt.Sprintf("%s OFFSET $%d", stmt.sql, len(stmt.args)+1)
		stmt.args = append(stmt.args, filter.Offset)
	}

	return p.fiats(ctx, stmt.sql, stmt.args...)
}

func (p *Postgres) fiats(ctx context.Context, sql string, args ...any) ([]domain.Rate, error) {

	rows, err := p.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rates []domain.Rate
	for rows.Next() {
		var rate domain.Rate
		err := rows.Scan(&rate.Id, &rate.CharCode, &rate.Nominal, &rate.Time, &rate.Value)
		if err != nil {
			return nil, err
		}
		rates = append(rates, rate)
	}

	return rates, rows.Err()
}

// RUBUSDRate возвращает текущий курск доллара
func (p *Postgres) RUBUSDRate(ctx context.Context) (domain.Rate, error) {
	sql := `
		SELECT rub.id, fiats.char_code, fiats.nominal, rub.time, rub.value
		FROM rub JOIN fiats ON rub.char_code = fiats.char_code 
		WHERE rub.char_code = 'USD' ORDER BY rub.time DESC LIMIT 1;`

	var rate domain.Rate
	return rate, p.db.QueryRow(ctx, sql).
		Scan(&rate.Id, &rate.CharCode, &rate.Nominal, &rate.Time, &rate.Value)
}

// exec вспомогательная функция, выполняет
// *pgx.conn.Exec() в транзакции.
func (p *Postgres) exec(ctx context.Context, sql string, args ...any) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = p.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// execBatch вспомогательная функция, выполняет
// множественные запросы *pgx.conn.Exec()
// в одной транзакции. Эти запросы не возвращают
// результатов (INSERT, UPDATE, DELETE).
func (p *Postgres) execBatch(ctx context.Context, stmts ...statement) error {

	b := new(pgx.Batch)
	for i := range stmts {
		b.Queue(stmts[i].sql, stmts[i].args...)
	}

	return p.db.BeginFunc(ctx, func(tx pgx.Tx) error {
		return tx.SendBatch(ctx, b).Close()
	})
}
