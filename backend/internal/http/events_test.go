package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"lucky-seven/backend/internal/game"
	"lucky-seven/backend/internal/realtime"
)

func readEvent(t *testing.T, ctx context.Context, conn *websocket.Conn, want string) {
	t.Helper()
	var event struct {
		Type string `json:"type"`
	}
	if err := wsjson.Read(ctx, conn, &event); err != nil {
		t.Fatal(err)
	}
	if event.Type != want {
		t.Fatalf("got %q, want %q", event.Type, want)
	}
}

func TestEventsNotifyOnlySuccessfulPlays(t *testing.T) {
	hub := realtime.New()
	defer hub.Close()
	repo := &fakeRepo{err: game.ErrInsufficient}
	server := httptest.NewServer(NewRouterWithEvents(game.New(repo), hub))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer first.CloseNow()
	second, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer second.CloseNow()
	readEvent(t, ctx, first, "ready")
	readEvent(t, ctx, second, "ready")
	// Directly invoke the REST handler to avoid a data race when changing the fake.
	router := NewRouterWithEvents(game.New(repo), hub)
	send := func() int {
		r := httptest.NewRequest("POST", "/api/play", strings.NewReader(`{"bet":10}`))
		r.Header.Set("Idempotency-Key", "events-test")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w.Code
	}
	if got := send(); got != 422 {
		t.Fatalf("got %d", got)
	}
	failureDeadline, failureDone := context.WithTimeout(ctx, 80*time.Millisecond)
	_, _, err = first.Read(failureDeadline)
	failureDone()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("failed play emitted an event: %v", err)
	}
	first, _, err = websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer first.CloseNow()
	readEvent(t, ctx, first, "ready")
	repo.err = nil
	if got := send(); got != 200 {
		t.Fatalf("got %d", got)
	}
	readEvent(t, ctx, first, "wallet.changed")
	readEvent(t, ctx, second, "wallet.changed")
	// No second event should be queued for the failed request.
	deadline, done := context.WithTimeout(ctx, 80*time.Millisecond)
	defer done()
	_, _, err = first.Read(deadline)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected extra event: %v", err)
	}
	// New connections always get a refresh instruction, even after missed events.
	third, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer third.CloseNow()
	readEvent(t, ctx, third, "ready")
	hub.Close()
	if _, _, err = second.Read(ctx); err == nil {
		t.Fatal("socket survived hub shutdown")
	}
}

func TestEventsRejectForeignOrigin(t *testing.T) {
	hub := realtime.New()
	defer hub.Close()
	server := httptest.NewServer(hub)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, response, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{"https://unrelated.example"}}})
	if conn != nil {
		conn.CloseNow()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("origin accepted: %v", err)
	}
}
