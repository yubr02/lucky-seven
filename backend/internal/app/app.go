// Package app is the composition root. Domain and storage packages do not depend on Fx.
package app

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"lucky-seven/backend/internal/config"
	"lucky-seven/backend/internal/game"
	httpapi "lucky-seven/backend/internal/http"
	"lucky-seven/backend/internal/storage/postgres"
)

var Module = fx.Module("api",
	fx.Provide(
		newPool,
		fx.Annotate(postgres.New, fx.As(new(game.Repository))),
		game.New,
		httpapi.NewRouter,
		newServer,
	),
	fx.Invoke(func(*http.Server) {}),
)

func New(cfg config.Config) *fx.App {
	return fx.New(fx.Supply(cfg), Module,
		fx.StartTimeout(10*time.Second),
		// Leave time for pool cleanup after HTTP's own shutdown deadline.
		fx.StopTimeout(cfg.ShutdownTimeout+5*time.Second),
	)
}

func newPool(lc fx.Lifecycle) (*pgxpool.Pool, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" && os.Getenv("PGHOST") == "" {
		return nil, errors.New("DATABASE_URL or PGHOST is required")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		// Driver errors can contain connection parameters. Keep secrets out of Fx logs.
		return nil, errors.New("invalid PostgreSQL configuration")
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := pool.Ping(ctx); err != nil {
				// Fx does not call OnStop for the hook whose OnStart failed.
				pool.Close()
				return errors.New("database connection failed; verify PostgreSQL configuration and availability")
			}
			return nil
		},
		OnStop: func(context.Context) error { pool.Close(); return nil },
	})
	return pool, nil
}

func newServer(lc fx.Lifecycle, cfg config.Config, handler http.Handler, shutdown fx.Shutdowner) *http.Server {
	server := &http.Server{
		Addr: ":" + strconv.Itoa(cfg.Port), Handler: handler,
		ReadTimeout: cfg.ReadTimeout, ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout: cfg.WriteTimeout, IdleTimeout: cfg.IdleTimeout,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Bind synchronously so an occupied port fails startup and rolls back the pool.
			listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", server.Addr)
			if err != nil {
				return err
			}
			server.Addr = listener.Addr().String()
			go func() {
				if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Printf("HTTP server failed: %v", err)
					_ = shutdown.Shutdown(fx.ExitCode(1))
				}
			}()
			log.Printf("API listening on %s", server.Addr)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, cfg.ShutdownTimeout)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				// Cancel active handlers before the database pool is closed.
				_ = server.Close()
				return err
			}
			return nil
		},
	})
	return server
}
