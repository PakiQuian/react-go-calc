package api

import (
	"log/slog"
	"net/http"
	"time"
)

type middleware func(http.Handler) http.Handler

// statusRecorder captures the status code for logging. net/http offers no way
// to read back what a handler wrote.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Write records an implicit 200 for handlers that never call WriteHeader.
func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func logRequests(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w}

			next.ServeHTTP(recorder, r)

			logger.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", recorder.status),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

// recoverPanics keeps one bad request from taking down the process.
//
// This is a backstop, not a strategy: the decimal library panics on division by
// zero, so the calculator guards every such case explicitly. If this middleware
// ever fires, a guard is missing.
func recoverPanics(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered",
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.Any("panic", recovered),
					)
					writeError(w, http.StatusInternalServerError, errorBody{
						Code:    codeInternalError,
						Message: "the request could not be completed",
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
