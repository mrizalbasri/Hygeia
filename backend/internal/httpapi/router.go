package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

// NewHandler builds the API routes and request logging middleware.
func NewHandler(logger *slog.Logger) http.Handler {
	return NewHandlerWithDatabase(logger, nil)
}

// NewHandlerWithDatabase adds a readiness check backed by PostgreSQL.
func NewHandlerWithDatabase(logger *slog.Logger, ping func(context.Context) error) http.Handler {
	return newHandler(logger, ping, nil, nil, false)
}

func NewAuthenticatedHandler(logger *slog.Logger, ping func(context.Context) error, h *auth.Handler, key []byte, secure bool, register ...func(*http.ServeMux)) http.Handler {
	return newHandler(logger, ping, h, key, secure, register...)
}

func newHandler(logger *slog.Logger, ping func(context.Context) error, h *auth.Handler, key []byte, secure bool, register ...func(*http.ServeMux)) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "hygeia-api",
		}); err != nil {
			logger.Error("write health response", "error", err)
		}
	})
	mux.HandleFunc("GET /api/v1/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		status := http.StatusOK
		body := map[string]string{"status": "ok", "database": "connected"}
		if ping == nil || ping(ctx) != nil {
			status = http.StatusServiceUnavailable
			body = map[string]string{"status": "unavailable", "database": "unavailable"}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(body); err != nil {
			logger.Error("write readiness response", "error", err)
		}
	})

	var handler http.Handler = mux
	if h != nil {
		h.Register(mux)
		for _, routes := range register {
			routes(mux)
		}
		handler = auth.Protect(mux, key, secure, h.CSRFTrustedOrigins...)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := make([]byte, 16)
		if _, err := rand.Read(id); err != nil {
			http.Error(w, "Service unavailable", 503)
			return
		}
		requestID := hex.EncodeToString(id)
		w.Header().Set("X-Request-ID", requestID)
		start := time.Now()
		handler.ServeHTTP(w, r)
		logger.Info("http request", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
