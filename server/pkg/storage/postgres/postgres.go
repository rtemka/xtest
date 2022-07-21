package postgres

import (
	"context"
	"xtestserver/domain"

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

// AddBtcRate
func (p *Postgres) AddBtcRate(ctx context.Context, rate domain.Rate) error {
	sql := `INSERT INTO btc_usdt(time, value) VALUES ($1, $2);`
	return p.exec(ctx, sql, rate.Time, rate.Value)
}

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

func (p *Postgres) BtcRate(ctx context.Context, limit, offset int) ([]domain.Rate, error) {
	sql := `
		SELECT btc.id, btc.time, btc.value
		FROM btc_usdt as btc 
		JOIN (SELECT btc_usdt.time FROM btc_usdt LIMIT $1 OFFSET $2) as btc_offset 
		ON btc.time = btc_offset.time;`

	var rates []domain.Rate

	rows, err := p.db.Query(ctx, sql, limit, offset)
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

func (p *Postgres) FiatsCurrent(ctx context.Context) ([]domain.Rate, error) {
	sql := `
		SELECT
			rub.id,
			fiats.char_code, 
			fiats.nominal, 
			rub.time, 
			rub.value	
		FROM rub JOIN (SELECT rub.id FROM rub LIMIT (SELECT COUNT(fiats.char_code) FROM fiats)) 
		as rub_offset ON rub.id = rub_offset.id
		JOIN fiats ON rub.char_code = fiats.char_code;`

	var rates []domain.Rate

	rows, err := p.db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (p *Postgres) RUBUSDRate(ctx context.Context) (domain.Rate, error) {
	sql := `
		SELECT
			rub.id,
			fiats.char_code, 
			fiats.nominal, 
			rub.time, 
			rub.value
		FROM rub JOIN fiats ON rub.char_code = fiats.char_code 
		WHERE rub.char_code = $1 ORDER BY rub.time DESC LIMIT 1;`

	var rate domain.Rate

	return rate, p.db.QueryRow(ctx, sql, "USD").
		Scan(&rate.Id, &rate.CharCode, &rate.Nominal, &rate.Time, &rate.Value)
}

// exec вспомогательная функция, выполняет
// *pgx.conn.Exec() в транзакции.
func (p *Postgres) exec(ctx context.Context, sql string, args ...any) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = p.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// execBatch is helper function, runs
// multiple batch queries *pgx.conn.Exec()
// in one transaction. This queries must be ones
// that don't return results (INSERT, UPDATE, DELETE).
func (p *Postgres) execBatch(ctx context.Context, stmts ...statement) error {

	b := new(pgx.Batch)

	for i := range stmts {
		b.Queue(stmts[i].sql, stmts[i].args...)
	}

	return p.db.BeginFunc(ctx, func(tx pgx.Tx) error {
		return tx.SendBatch(ctx, b).Close()
	})
}
