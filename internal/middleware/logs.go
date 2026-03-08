package middleware

import (
	"log/slog"
	"net/http"
)

func LoggingMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("request", "from", r.Host, "method", r.Method, "URL", r.URL)
		next.ServeHTTP(w, r)
	})
}
