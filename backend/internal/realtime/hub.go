// Package realtime delivers best-effort invalidations for the single demo wallet.
// PostgreSQL remains the source of truth; clients refresh on every connection.
package realtime

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type Hub struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	closed  bool
	workers sync.WaitGroup
}

func New() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{clients: make(map[chan struct{}]struct{}), ctx: ctx, cancel: cancel}
}

// Publish never waits for network I/O. One pending invalidation is sufficient.
func (h *Hub) Publish() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (h *Hub) Close() {
	h.mu.Lock()
	h.closed = true
	h.cancel()
	h.mu.Unlock()
	h.workers.Wait()
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The library validates Origin against Host. Never accept arbitrary origins.
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	if h.closed || len(h.clients) >= 64 {
		h.mu.Unlock()
		return
	}
	h.clients[ch] = struct{}{}
	h.workers.Add(1)
	h.mu.Unlock()
	defer func() {
		_ = conn.CloseNow()
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
		h.workers.Done()
	}()
	conn.SetReadLimit(1024)
	ctx := conn.CloseRead(h.ctx)
	send := func(payload string) error {
		writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return conn.Write(writeCtx, websocket.MessageText, []byte(payload))
	}
	// A reconnect can have missed events: always request an authoritative refresh.
	if send(`{"type":"ready"}`) != nil {
		return
	}
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			if send(`{"type":"wallet.changed"}`) != nil {
				return
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
