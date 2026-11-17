package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"blood/apps/api/internal/contracts"
)

func New(checkReady func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, contracts.Health{Status: "ok"})
	})
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := checkReady(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, contracts.Health{Status: "not_ready"})
			return
		}
		writeJSON(w, http.StatusOK, contracts.Health{Status: "ok"})
	})
	mux.HandleFunc("GET /api/v1/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, contracts.Status{Service: "blood-api", Stage: "bootstrap"})
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
