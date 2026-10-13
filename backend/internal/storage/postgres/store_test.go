package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"lucky-seven/backend/internal/game"
)

// Uses an isolated schema, never truncates existing tables.
func testStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("REQUIRE_DATABASE_TESTS") == "1" {
			t.Fatal("TEST_DATABASE_URL is required when REQUIRE_DATABASE_TESTS=1")
		}
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	schema := fmt.Sprintf("game_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if _, err := admin.Exec(cleanup, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	sql, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "001_game.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(sql)); err != nil {
		t.Fatal(err)
	}
	return New(pool), ctx
}

func TestConcurrentRetries(t *testing.T) {
	s, ctx := testStore(t)
	var draws atomic.Int32
	draw := func() ([3]int, error) { draws.Add(1); return [3]int{1, 2, 3}, nil }
	var wg sync.WaitGroup
	results := make(chan game.Result, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := s.Play(ctx, "demo", "same", 100, draw)
			if err != nil {
				t.Error(err)
				return
			}
			results <- r
		}()
	}
	wg.Wait()
	close(results)
	var first game.Result
	for r := range results {
		if first.ID == 0 {
			first = r
		}
		if r != first {
			t.Errorf("retry differs: %+v vs %+v", r, first)
		}
	}
	b, err := s.Balance(ctx, "demo")
	if err != nil {
		t.Fatal(err)
	}
	h, err := s.History(ctx, "demo", 0)
	if err != nil {
		t.Fatal(err)
	}
	if draws.Load() != 1 || b != 900 || len(h) != 1 {
		t.Fatalf("draws=%d balance=%d history=%d", draws.Load(), b, len(h))
	}
	if _, err = s.Play(ctx, "demo", "same", 1, draw); !errors.Is(err, game.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
}

func TestConcurrentSpending(t *testing.T) {
	s, ctx := testStore(t)
	var successes, denied, draws atomic.Int32
	draw := func() ([3]int, error) { draws.Add(1); return [3]int{1, 2, 3}, nil }
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.Play(ctx, "demo", fmt.Sprint(i), 100, draw)
			switch {
			case err == nil:
				successes.Add(1)
			case errors.Is(err, game.ErrInsufficient):
				denied.Add(1)
			default:
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	b, err := s.Balance(ctx, "demo")
	if err != nil {
		t.Fatal(err)
	}
	h, err := s.History(ctx, "demo", 0)
	if err != nil {
		t.Fatal(err)
	}
	if successes.Load() != 10 || denied.Load() != 10 || draws.Load() != 10 || b != 0 || len(h) != 10 {
		t.Fatalf("success=%d denied=%d draws=%d balance=%d history=%d", successes.Load(), denied.Load(), draws.Load(), b, len(h))
	}
	var key string
	if err = s.pool.QueryRow(ctx, `SELECT idempotency_key FROM plays ORDER BY id LIMIT 1`).Scan(&key); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Play(ctx, "demo", key, 100, draw); err != nil {
		t.Fatalf("replay at zero balance: %v", err)
	}
	if draws.Load() != 10 {
		t.Fatal("replay drew again")
	}
}

func TestRollbackAndPayout(t *testing.T) {
	s, ctx := testStore(t)
	failure := errors.New("entropy unavailable")
	if _, err := s.Play(ctx, "demo", "retry", 100, func() ([3]int, error) { return [3]int{}, failure }); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	// Force an INSERT failure after the wallet UPDATE to prove both roll back.
	if _, err := s.Play(ctx, "demo", "retry", 100, func() ([3]int, error) { return [3]int{10, 2, 3}, nil }); err == nil {
		t.Fatal("expected constraint failure")
	}
	b, err := s.Balance(ctx, "demo")
	if err != nil || b != 1000 {
		t.Fatalf("rollback balance=%d err=%v", b, err)
	}
	h, err := s.History(ctx, "demo", 0)
	if err != nil || len(h) != 0 {
		t.Fatalf("rollback history=%v err=%v", h, err)
	}
	r, err := s.Play(ctx, "demo", "retry", 100, func() ([3]int, error) { return [3]int{7, 7, 7}, nil })
	if err != nil {
		t.Fatal(err)
	}
	if r.Payout != 10000 || r.Balance != 10900 {
		t.Fatalf("wrong payout: %+v", r)
	}
	h, err = s.History(ctx, "demo", r.ID)
	if err != nil || len(h) != 0 {
		t.Fatalf("cursor history=%v err=%v", h, err)
	}
}
