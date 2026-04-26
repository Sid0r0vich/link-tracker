package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
)

func LoggingMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		logger.Info("request", "from", r.Host, "method", r.Method, "URL", r.URL)
		next.ServeHTTP(recorder, r)

		if recorder.statusCode != http.StatusOK {
			logger.Warn(
				"response",
				"from", r.Host,
				"method", r.Method,
				"URL", r.URL,
				"status", recorder.statusCode,
				"body", recorder.body.String(),
			)
		}
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	r.body.Write(data)
	return r.ResponseWriter.Write(data)
}
