package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"lucky-seven/backend/internal/game"
	"lucky-seven/backend/internal/realtime"
)

func NewRouter(service *game.Service) http.Handler {
	return NewRouterWithEvents(service, nil)
}

func NewRouterWithEvents(service *game.Service, events *realtime.Hub) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	if events != nil {
		mux.Handle("GET /api/events", events)
	}
	if service != nil {
		mux.HandleFunc("POST /api/play", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Bet int64 `json:"bet"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, 1024)
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(w, game.ErrInvalid)
				return
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				writeError(w, game.ErrInvalid)
				return
			}
			keys := r.Header.Values("Idempotency-Key")
			if len(keys) != 1 {
				writeError(w, game.ErrInvalid)
				return
			}
			result, err := service.Play(r.Context(), "demo", keys[0], body.Bet)
			if err != nil {
				writeError(w, err)
				return
			}
			// Play has committed (or returned a persisted replay). Never notify errors.
			if events != nil {
				events.Publish()
			}
			writeJSON(w, http.StatusOK, result)
		})
		mux.HandleFunc("GET /api/balance", func(w http.ResponseWriter, r *http.Request) {
			balance, err := service.Balance(r.Context(), "demo")
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]int64{"balance": balance})
		})
		mux.HandleFunc("GET /api/history", func(w http.ResponseWriter, r *http.Request) {
			var before int64
			if value := r.URL.Query().Get("before"); value != "" {
				var err error
				before, err = strconv.ParseInt(value, 10, 64)
				if err != nil || before <= 0 {
					writeError(w, game.ErrInvalid)
					return
				}
			}
			results, err := service.History(r.Context(), "demo", before)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, results)
		})
	}
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal_error"
	switch {
	case errors.Is(err, game.ErrInvalid):
		status, code = http.StatusBadRequest, "invalid_request"
	case errors.Is(err, game.ErrInsufficient):
		status, code = http.StatusUnprocessableEntity, "insufficient_balance"
	case errors.Is(err, game.ErrConflict):
		status, code = http.StatusConflict, "idempotency_conflict"
	default:
		log.Printf("game request failed: %v", err)
	}
	writeJSON(w, status, map[string]string{"error": code})
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
