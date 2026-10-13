package httpapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"lucky-seven/backend/internal/game"
)

type fakeRepo struct {
	calls int
	err   error
}

func (f *fakeRepo) Play(_ context.Context, player, key string, bet int64, _ game.Draw) (game.Result, error) {
	f.calls++
	return game.Result{ID: 1, Bet: bet}, f.err
}
func (f *fakeRepo) Balance(context.Context, string) (int64, error) { return 1000, nil }
func (f *fakeRepo) History(context.Context, string, int64) ([]game.Result, error) {
	return []game.Result{}, nil
}

func TestPlayValidationAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body, key string
		err             error
		status, calls   int
	}{
		{"valid", `{"bet":10}`, "key-1", nil, 200, 1},
		{"minimum", `{"bet":1}`, "key-1", nil, 200, 1},
		{"maximum", `{"bet":100}`, "key-1", nil, 200, 1},
		{"zero", `{"bet":0}`, "key-1", nil, 400, 0},
		{"negative", `{"bet":-1}`, "key-1", nil, 400, 0},
		{"too large", `{"bet":101}`, "key-1", nil, 400, 0},
		{"fraction", `{"bet":1.5}`, "key-1", nil, 400, 0},
		{"missing key", `{"bet":10}`, "", nil, 400, 0},
		{"long key", `{"bet":10}`, strings.Repeat("x", 129), nil, 400, 0},
		{"extra field", `{"bet":10,"player":"someone"}`, "key", nil, 400, 0},
		{"trailing json", `{"bet":10} {}`, "key", nil, 400, 0},
		{"insufficient", `{"bet":10}`, "key", game.ErrInsufficient, 422, 1},
		{"conflict", `{"bet":10}`, "key", game.ErrConflict, 409, 1},
		{"internal", `{"bet":10}`, "key", errors.New("secret database detail"), 500, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeRepo{err: tc.err}
			r := httptest.NewRequest("POST", "/api/play", strings.NewReader(tc.body))
			if tc.key != "" {
				r.Header.Set("Idempotency-Key", tc.key)
			}
			w := httptest.NewRecorder()
			NewRouter(game.New(f)).ServeHTTP(w, r)
			if w.Code != tc.status || f.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, f.calls, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "secret") {
				t.Fatal("leaked internal error")
			}
		})
	}
}
