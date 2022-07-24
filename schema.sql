DROP TABLE IF EXISTS fiats, btc_usdt, rub;

CREATE TABLE IF NOT EXISTS fiats (
    char_code VARCHAR(3),
    nominal INT NOT NULL,
    PRIMARY KEY(char_code)
);

CREATE TABLE IF NOT EXISTS btc_usdt (
    id BIGSERIAL PRIMARY KEY,
    time BIGINT CHECK(time > 0) DEFAULT extract(epoch from now()),
    value NUMERIC(20, 4) NOT NULL,
    UNIQUE(time)
);

CREATE TABLE IF NOT EXISTS rub (
    id BIGSERIAL PRIMARY KEY,
    char_code VARCHAR(3),
    time BIGINT CHECK(time > 0),
    value NUMERIC(20, 4) NOT NULL,
    UNIQUE(char_code, time),
    FOREIGN KEY (char_code) REFERENCES fiats(char_code)
);

CREATE INDEX IF NOT EXISTS btc_time_idx ON btc_usdt(time DESC);
CREATE INDEX IF NOT EXISTS rub_time_idx ON rub(time DESC);