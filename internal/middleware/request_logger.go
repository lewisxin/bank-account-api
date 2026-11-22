package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type requestLogAttrsKey struct{}

// AddRequestLogAttrs adds attributes to the log emitted when the request finishes.
func AddRequestLogAttrs(ctx context.Context, attrs ...slog.Attr) {
	requestAttrs, ok := ctx.Value(requestLogAttrsKey{}).(*[]slog.Attr)
	if ok {
		*requestAttrs = append(*requestAttrs, attrs...)
	}
}

func RequestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		var requestAttrs []slog.Attr
		r = r.WithContext(context.WithValue(r.Context(), requestLogAttrsKey{}, &requestAttrs))
		response := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(response, r)

		status := response.status
		if status == 0 {
			status = http.StatusOK
		}
		attrs := append(requestAttrs,
			slog.String("method", r.Method),
			slog.String("route", r.Pattern),
			slog.Int("status", status),
			slog.Float64("duration_ms", float64(time.Since(startedAt))/float64(time.Millisecond)),
		)
		logger.LogAttrs(r.Context(), slog.LevelInfo, "request completed", attrs...)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 && (status >= 200 || status == http.StatusSwitchingProtocols) {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(p)
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
