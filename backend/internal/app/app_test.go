package app

import (
	"context"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"lucky-seven/backend/internal/config"
)

func testConfig() config.Config {
	return config.Config{Port: 0, ReadTimeout: time.Second, ReadHeaderTimeout: time.Second,
		WriteTimeout: time.Second, IdleTimeout: time.Second, ShutdownTimeout: time.Second}
}

func TestDependencyGraph(t *testing.T) {
	if err := fx.ValidateApp(fx.Supply(testConfig()), Module); err != nil {
		t.Fatal(err)
	}
}

func database(t *testing.T) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("REQUIRE_DATABASE_TESTS") == "1" {
			t.Fatal("TEST_DATABASE_URL is required")
		}
		t.Skip("set TEST_DATABASE_URL for lifecycle integration tests")
	}
	t.Setenv("DATABASE_URL", url)
}

func TestStartAndStop(t *testing.T) {
	database(t)
	var server *http.Server
	var pool *pgxpool.Pool
	a := fx.New(fx.NopLogger, fx.Supply(testConfig()), Module, fx.Populate(&server, &pool))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Stop(context.Background()) })
	_, port, err := net.SplitHostPort(server.Addr)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if err := a.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(ctx); err == nil {
		t.Fatal("pool still open after stop")
	}
	if conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, time.Second); err == nil {
		conn.Close()
		t.Fatal("listener still open")
	}
}

func TestOccupiedPortRollsBackPool(t *testing.T) {
	database(t)
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	cfg := testConfig()
	cfg.Port, _ = strconv.Atoi(port)
	var pool *pgxpool.Pool
	a := fx.New(fx.NopLogger, fx.Supply(cfg), Module, fx.Populate(&pool))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.Start(ctx); err == nil {
		_ = a.Stop(ctx)
		t.Fatal("expected occupied port failure")
	}
	if err := pool.Ping(ctx); err == nil {
		t.Fatal("pool not closed after startup rollback")
	}
}

func TestDatabaseFailureClosesPool(t *testing.T) {
	// A listener that never speaks PostgreSQL forces the bounded startup deadline.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	t.Setenv("DATABASE_URL", "postgres://test:test@"+listener.Addr().String()+"/test?sslmode=disable")
	var pool *pgxpool.Pool
	a := fx.New(fx.NopLogger, fx.Supply(testConfig()), Module, fx.Populate(&pool))
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := a.Start(ctx); err == nil {
		_ = a.Stop(context.Background())
		t.Fatal("expected database failure")
	}
	check, done := context.WithTimeout(context.Background(), time.Second)
	defer done()
	if err := pool.Ping(check); err == nil {
		t.Fatal("pool still open after failed ping")
	}
}
