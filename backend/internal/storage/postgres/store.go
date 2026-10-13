package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"lucky-seven/backend/internal/game"
)

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

const columns = `id, number_1, number_2, number_3, bet, payout, balance_after, created_at`

func scan(row pgx.Row) (game.Result, error) {
	var r game.Result
	err := row.Scan(&r.ID, &r.Numbers[0], &r.Numbers[1], &r.Numbers[2], &r.Bet, &r.Payout, &r.Balance, &r.CreatedAt)
	return r, err
}

func (s *Store) Play(ctx context.Context, player, key string, bet int64, draw game.Draw) (game.Result, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return game.Result{}, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()

	// Serialize all plays for this wallet, including concurrent retries.
	var balance int64
	if err = tx.QueryRow(ctx, `SELECT balance FROM wallets WHERE player_id=$1 FOR UPDATE`, player).Scan(&balance); err != nil {
		return game.Result{}, err
	}
	previous, err := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM plays WHERE player_id=$1 AND idempotency_key=$2`, player, key))
	if err == nil {
		if previous.Bet != bet {
			return game.Result{}, game.ErrConflict
		}
		return previous, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return game.Result{}, err
	}
	if balance < bet {
		return game.Result{}, game.ErrInsufficient
	}
	numbers, err := draw()
	if err != nil {
		return game.Result{}, err
	}
	payout := game.Payout(numbers, bet)
	// Arithmetic happens in PostgreSQL so bigint overflow aborts the transaction.
	if err = tx.QueryRow(ctx, `UPDATE wallets SET balance=balance-$2+$3 WHERE player_id=$1 RETURNING balance`, player, bet, payout).Scan(&balance); err != nil {
		return game.Result{}, err
	}
	r, err := scan(tx.QueryRow(ctx, `INSERT INTO plays (player_id,idempotency_key,bet,number_1,number_2,number_3,payout,balance_after) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING `+columns, player, key, bet, numbers[0], numbers[1], numbers[2], payout, balance))
	if err != nil {
		return game.Result{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return game.Result{}, err
	}
	return r, nil
}

func (s *Store) Balance(ctx context.Context, player string) (int64, error) {
	var balance int64
	err := s.pool.QueryRow(ctx, `SELECT balance FROM wallets WHERE player_id=$1`, player).Scan(&balance)
	return balance, err
}

func (s *Store) History(ctx context.Context, player string, before int64) ([]game.Result, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+columns+` FROM plays WHERE player_id=$1 AND ($2::bigint=0 OR id<$2) ORDER BY id DESC LIMIT 50`, player, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make([]game.Result, 0)
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
