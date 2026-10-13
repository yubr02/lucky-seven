BEGIN;
CREATE TABLE wallets (
    player_id text PRIMARY KEY,
    balance bigint NOT NULL DEFAULT 1000 CHECK (balance >= 0)
);
CREATE TABLE plays (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    player_id text NOT NULL REFERENCES wallets(player_id),
    idempotency_key varchar(128) NOT NULL,
    bet bigint NOT NULL CHECK (bet BETWEEN 1 AND 100),
    number_1 smallint NOT NULL CHECK (number_1 BETWEEN 0 AND 9),
    number_2 smallint NOT NULL CHECK (number_2 BETWEEN 0 AND 9),
    number_3 smallint NOT NULL CHECK (number_3 BETWEEN 0 AND 9),
    payout bigint NOT NULL CHECK (payout >= 0),
    balance_after bigint NOT NULL CHECK (balance_after >= 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (player_id, idempotency_key)
);
CREATE INDEX plays_history ON plays (player_id, id DESC);
-- Single local demonstration player. No public player identity is accepted by the API.
INSERT INTO wallets (player_id) VALUES ('demo');
COMMIT;
