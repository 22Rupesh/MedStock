package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"medstock/pkg/logger"
)

type contextKey string

const (
	ClientKey   contextKey = "client"
	RequestIDKey contextKey = "request_id"
)

type APIKeyAuth struct {
	authenticate func(ctx context.Context, key string) (interface{}, error)
}

func NewAPIKeyAuth(auth func(ctx context.Context, key string) (interface{}, error)) *APIKeyAuth {
	return &APIKeyAuth{authenticate: auth}
}

func (a *APIKeyAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error": "invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		client, err := a.authenticate(r.Context(), parts[1])
		if err != nil {
			http.Error(w, `{"error": "invalid api key"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClientKey, client)
		ctx = context.WithValue(ctx, RequestIDKey, uuid.New().String())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestLogger(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", rw.statusCode).
				Int64("duration_ms", time.Since(start).Milliseconds()).
				Str("user_agent", r.UserAgent()).
				Msg("http_request")
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func GetClientFromContext(ctx context.Context) interface{} {
	return ctx.Value(ClientKey)
}

func GetRequestIDFromContext(ctx context.Context) string {
	if id := ctx.Value(RequestIDKey); id != nil {
		return id.(string)
	}
	return ""
}

type RateLimiter struct {
	limit int
}

func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{limit: limit}
}

func (r *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ip := req.RemoteAddr
		_ = ip
		next.ServeHTTP(w, req)
	})
}