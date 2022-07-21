DROP TABLE IF EXISTS fiats, btc_usdt, rub;

CREATE TABLE IF NOT EXISTS fiats (
    char_code VARCHAR(3),
    nominal INT NOT NULL,
    PRIMARY KEY(char_code)
);

CREATE TABLE IF NOT EXISTS btc_usdt (
    id BIGSERIAL PRIMARY KEY,
    time BIGINT CHECK(time > 0) DEFAULT extract(epoch from now()),
    value NUMERIC(20, 4) NOT NULL
);

CREATE TABLE IF NOT EXISTS rub (
    id BIGSERIAL PRIMARY KEY,
    char_code VARCHAR(3),
    time BIGINT CHECK(time > 0),
    value NUMERIC(20, 4) NOT NULL,
    UNIQUE(char_code, time),
    FOREIGN KEY (char_code) REFERENCES fiats (char_code)
);

CREATE INDEX IF NOT EXISTS btc_rate_time_idx ON btc_usdt(time DESC);
CREATE INDEX IF NOT EXISTS rub_rate_time_idx ON rub(time DESC);

INSERT INTO btc_usdt(time, value) VALUES (1658252361, 22278.20);
INSERT INTO btc_usdt(time, value) VALUES (1658252362, 22378.20);
INSERT INTO fiats(char_code, nominal) VALUES ('USD', 1);
INSERT INTO fiats(char_code, nominal) VALUES ('HUF', 100);
INSERT INTO rub(char_code, time, value) VALUES ('USD', 1658252361, 22278.20);
INSERT INTO rub(char_code, time, value) VALUES ('HUF', 1658252361, 32378.20);