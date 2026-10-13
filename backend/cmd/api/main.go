package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lucky-seven/backend/internal/config"
	"lucky-seven/backend/internal/game"
	httpapi "lucky-seven/backend/internal/http"
	"lucky-seven/backend/internal/storage/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" && os.Getenv("PGHOST") == "" {
		log.Fatal("DATABASE_URL or PGHOST is required; apply migrations/001_game.sql before starting")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	pool, err := pgxpool.New(ctx, databaseURL)
	if err == nil {
		err = pool.Ping(ctx)
	}
	cancel()
	if err != nil {
		log.Fatal("database connection failed; verify DATABASE_URL and PostgreSQL availability")
	}
	defer pool.Close()

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           httpapi.NewRouter(game.New(postgres.New(pool))),
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("API listening on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	case <-shutdownContext.Done():
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
			if closeErr := server.Close(); closeErr != nil {
				log.Printf("force close failed: %v", closeErr)
			}
		}
		if err := <-serverErrors; !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server stopped with error: %v", err)
		}
	}
}
